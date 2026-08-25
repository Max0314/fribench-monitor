package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Max0314/fribench-monitor/internal/monitor"
)

type Auth struct {
	Username string
	Password string
}

type API struct {
	state *monitor.State
}

func NewHandler(state *monitor.State, assets fs.FS, auth Auth) (http.Handler, error) {
	dist, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, err
	}

	api := &API{state: state}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /api/snapshot", api.snapshot)
	mux.HandleFunc("GET /api/history", api.history)
	mux.Handle("/", static(dist))

	var handler http.Handler = securityHeaders(mux)
	if auth.Username != "" || auth.Password != "" {
		if auth.Username == "" || auth.Password == "" {
			return nil, errors.New("both auth username and password are required")
		}
		handler = basicAuth(handler, auth)
		handler = healthWithoutAuth(handler, mux)
	}
	return requestLimits(handler), nil
}

func health(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write([]byte("ok\n"))
}

func (api *API) snapshot(response http.ResponseWriter, _ *http.Request) {
	snapshot, ok := api.state.Snapshot()
	if !ok {
		writeJSON(response, http.StatusServiceUnavailable, map[string]string{
			"error": "metrics are not ready",
		})
		return
	}

	writeJSON(response, http.StatusOK, struct {
		Snapshot              any   `json:"snapshot"`
		SampleIntervalSeconds int64 `json:"sampleIntervalSeconds"`
		RetentionSeconds      int64 `json:"retentionSeconds"`
	}{
		Snapshot:              snapshot,
		SampleIntervalSeconds: int64(api.state.SampleInterval().Seconds()),
		RetentionSeconds:      int64(api.state.Retention().Seconds()),
	})
}

func (api *API) history(response http.ResponseWriter, request *http.Request) {
	rangeSeconds := int64(3600)
	if value := request.URL.Query().Get("range"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			writeJSON(response, http.StatusBadRequest, map[string]string{"error": "invalid range"})
			return
		}
		rangeSeconds = parsed
	}

	maxRange := int64(api.state.Retention().Seconds())
	if rangeSeconds < 60 {
		rangeSeconds = 60
	}
	if rangeSeconds > maxRange {
		rangeSeconds = maxRange
	}

	points := api.state.History(time.Now().Add(-time.Duration(rangeSeconds)*time.Second), 720)
	writeJSON(response, http.StatusOK, struct {
		RangeSeconds int64 `json:"rangeSeconds"`
		Points       any   `json:"points"`
	}{
		RangeSeconds: rangeSeconds,
		Points:       points,
	})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}

func static(dist fs.FS) http.Handler {
	files := http.FileServer(http.FS(dist))
	indexHTML, indexError := fs.ReadFile(dist, "index.html")

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/"))
		if requestPath == "." {
			requestPath = "index.html"
		}

		if _, err := fs.Stat(dist, requestPath); err != nil {
			requestPath = "index.html"
		}

		if extension := path.Ext(requestPath); extension != "" {
			if contentType := mime.TypeByExtension(extension); contentType != "" {
				response.Header().Set("Content-Type", contentType)
			}
		}
		if requestPath == "index.html" {
			if indexError != nil {
				http.Error(response, "frontend is not built", http.StatusServiceUnavailable)
				return
			}
			response.Header().Set("Content-Type", "text/html; charset=utf-8")
			response.Header().Set("Cache-Control", "no-cache")
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write(indexHTML)
			return
		}
		if strings.Contains(path.Base(requestPath), "-") {
			response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		cloned := request.Clone(request.Context())
		cloned.URL.Path = "/" + requestPath
		files.ServeHTTP(response, cloned)
	})
}

func basicAuth(next http.Handler, auth Auth) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(auth.Username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(auth.Password)) == 1
		if !ok || !usernameMatch || !passwordMatch {
			response.Header().Set("WWW-Authenticate", `Basic realm="Fribench Monitor", charset="UTF-8"`)
			http.Error(response, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func healthWithoutAuth(authenticated, original http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/healthz" {
			original.ServeHTTP(response, request)
			return
		}
		authenticated.ServeHTTP(response, request)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https://beian.mps.gov.cn; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		response.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(response, request)
	})
}

func requestLimits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(response, request.Body, 1024)
		next.ServeHTTP(response, request)
	})
}
