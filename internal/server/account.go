package server

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type AccountConfig struct{ Origin, Endpoint, Secret string }
type accountSession struct {
	Authenticated bool   `json:"authenticated"`
	PrivateRole   string `json:"privateRole"`
}
type accountCache struct {
	value   accountSession
	expires time.Time
}
type accountGateway struct {
	config AccountConfig
	client *http.Client
	mutex  sync.Mutex
	cache  map[string]accountCache
}

func newAccountGateway(config AccountConfig) (*accountGateway, error) {
	origin, err := url.Parse(config.Origin)
	if err != nil || origin.Scheme == "" || origin.Host == "" || config.Secret == "" {
		return nil, errors.New("invalid protected account configuration")
	}
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || !strings.HasSuffix(endpoint.Path, "/internal/identity") {
		return nil, errors.New("invalid account gateway")
	}
	return &accountGateway{config: config, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, cache: make(map[string]accountCache)}, nil
}

func matchingCookie(request *http.Request) string {
	cookie, err := request.Cookie("__Host-fribench_fribench_monitor")
	if err != nil || len(cookie.Value) != 43 {
		return ""
	}
	for _, char := range cookie.Value {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_' || char == '-') {
			return ""
		}
	}
	return cookie.Name + "=" + cookie.Value
}

func (gateway *accountGateway) call(request *http.Request, claim bool) (accountSession, error) {
	cookie := matchingCookie(request)
	if cookie == "" {
		return accountSession{}, nil
	}
	// Serialize cache misses, including the first concurrent snapshot/history request.
	gateway.mutex.Lock()
	defer gateway.mutex.Unlock()
	if !claim {
		if cached, ok := gateway.cache[cookie]; ok && time.Now().Before(cached.expires) {
			return cached.value, nil
		}
	}
	endpoint := gateway.config.Endpoint
	if claim {
		endpoint = strings.TrimSuffix(endpoint, "/identity") + "/owner-claim"
	}
	upstream, err := http.NewRequestWithContext(request.Context(), http.MethodPost, endpoint, bytes.NewReader(nil))
	if err != nil {
		return accountSession{}, errors.New("account verification unavailable")
	}
	origin, _ := url.Parse(gateway.config.Origin)
	upstream.Host = origin.Host
	upstream.Header.Set("X-Fribench-App", "fribench-monitor")
	upstream.Header.Set("Cookie", cookie)
	upstream.SetBasicAuth("fribench-fribench-monitor", gateway.config.Secret)
	response, err := gateway.client.Do(upstream)
	if err != nil {
		return accountSession{}, errors.New("account verification unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return accountSession{}, errors.New("account permission not granted")
	}
	if claim {
		delete(gateway.cache, cookie)
		return accountSession{Authenticated: true, PrivateRole: "owner"}, nil
	}
	var value accountSession
	if json.NewDecoder(http.MaxBytesReader(nil, response.Body, 4096)).Decode(&value) != nil {
		return accountSession{}, errors.New("invalid account response")
	}
	if len(gateway.cache) >= 256 {
		clear(gateway.cache)
	}
	gateway.cache[cookie] = accountCache{value: value, expires: time.Now().Add(5 * time.Second)}
	return value, nil
}

func validBasic(request *http.Request, auth Auth) bool {
	user, password, ok := request.BasicAuth()
	return ok && auth.Password != "" && subtle.ConstantTimeCompare([]byte(user), []byte(auth.Username)) == 1 && subtle.ConstantTimeCompare([]byte(password), []byte(auth.Password)) == 1
}

func accountAuth(next http.Handler, auth Auth) (http.Handler, error) {
	gateway, err := newAccountGateway(*auth.Account)
	if err != nil {
		return nil, err
	}
	type bucket struct {
		count int
		start time.Time
	}
	attempts := make(map[string]bucket)
	var rateMutex sync.Mutex
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/healthz" {
			next.ServeHTTP(response, request)
			return
		}
		api := strings.HasPrefix(request.URL.Path, "/api/")
		if !api {
			next.ServeHTTP(response, request)
			return
		}
		legacy := validBasic(request, auth)
		session, lookupError := gateway.call(request, false)
		if request.URL.Path == "/api/account" && request.Method == http.MethodGet {
			if lookupError != nil && !legacy {
				writeJSON(response, 503, map[string]string{"error": "account_service_unavailable"})
				return
			}
			writeJSON(response, 200, map[string]bool{"signedIn": session.Authenticated, "authorized": legacy || session.Authenticated && session.PrivateRole == "owner"})
			return
		}
		if request.URL.Path == "/api/account/link" {
			if request.Method != http.MethodPost || request.Header.Get("Origin") != auth.Account.Origin || !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
				writeJSON(response, 403, map[string]string{"error": "origin_rejected"})
				return
			}
			if lookupError != nil || !session.Authenticated {
				writeJSON(response, 401, map[string]string{"error": "login_required"})
				return
			}
			ip := request.Header.Get("X-Real-IP")
			if ip == "" {
				ip = request.RemoteAddr
			}
			rateMutex.Lock()
			entry := attempts[ip]
			if time.Since(entry.start) > 10*time.Minute {
				entry = bucket{start: time.Now()}
			}
			entry.count++
			attempts[ip] = entry
			if len(attempts) > 2048 {
				for key, value := range attempts {
					if time.Since(value.start) > 10*time.Minute {
						delete(attempts, key)
					}
				}
			}
			allowed := entry.count <= 6
			rateMutex.Unlock()
			if !allowed {
				writeJSON(response, 429, map[string]string{"error": "rate_limited"})
				return
			}
			var input struct {
				Password string `json:"password"`
			}
			if json.NewDecoder(request.Body).Decode(&input) != nil || subtle.ConstantTimeCompare([]byte(input.Password), []byte(auth.Password)) != 1 || auth.Password == "" {
				writeJSON(response, 403, map[string]string{"error": "operator_credential_required"})
				return
			}
			if _, err := gateway.call(request, true); err != nil {
				writeJSON(response, 409, map[string]string{"error": "owner_already_assigned_or_unavailable"})
				return
			}
			writeJSON(response, 200, map[string]bool{"authorized": true})
			return
		}
		if legacy || lookupError == nil && session.Authenticated && session.PrivateRole == "owner" {
			next.ServeHTTP(response, request)
			return
		}
		writeJSON(response, 401, map[string]string{"error": "monitor_owner_required"})
	}), nil
}
