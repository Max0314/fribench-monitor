package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Max0314/fribench-monitor/internal/metrics"
	"github.com/Max0314/fribench-monitor/internal/monitor"
)

func TestUnifiedAccountRequiresExistingOperatorProofAndOneOwner(t *testing.T) {
	owner := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		user, password, ok := request.BasicAuth()
		if !ok || user != "fribench-fribench-monitor" || password != "fixture-service-secret" {
			t.Error("invalid service credential")
		}
		if request.Header.Get("X-Fribench-App") != "fribench-monitor" || strings.Contains(request.Header.Get("Cookie"), "unrelated") {
			t.Error("invalid app isolation")
		}
		token := request.Header.Get("Cookie")
		if strings.HasSuffix(request.URL.Path, "/owner-claim") {
			if owner != "" && owner != token {
				response.WriteHeader(409)
				return
			}
			owner = token
			writeJSON(response, 200, map[string]string{"role": "owner"})
			return
		}
		role := ""
		if owner == token {
			role = "owner"
		}
		writeJSON(response, 200, accountSession{Authenticated: true, PrivateRole: role})
	}))
	defer upstream.Close()
	state := monitor.NewState(time.Second, time.Hour)
	state.Update(metrics.Snapshot{Hostname: "private-fixture-host"})
	handler, err := NewHandler(state, testAssets(), Auth{Username: "monitor", Password: "fixture-old-password", Account: &AccountConfig{Origin: "https://monitor.fribench.cn", Endpoint: upstream.URL + "/internal/identity", Secret: "fixture-service-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, cookie, body, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Cookie", cookie)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		record := httptest.NewRecorder()
		handler.ServeHTTP(record, req)
		return record
	}
	cookie := "__Host-fribench_fribench_monitor=" + strings.Repeat("a", 43) + "; unrelated=never-forward"
	other := "__Host-fribench_fribench_monitor=" + strings.Repeat("b", 43)
	for _, path := range []string{"/api/snapshot", "/api/history"} {
		if record := request("GET", path, "", "", ""); record.Code != 401 || strings.Contains(record.Body.String(), "private-fixture") {
			t.Fatal("anonymous metric exposure")
		}
	}
	if request("GET", "/healthz", "", "", "").Code != 200 || request("GET", "/", "", "", "").Code != 200 {
		t.Fatal("public sign-in shell or health missing")
	}
	if request("GET", "/api/snapshot", cookie, "", "").Code != 401 {
		t.Fatal("normal login granted monitoring")
	}
	if request("POST", "/api/account/link", cookie, `{"password":"fixture-old-password"}`, "https://evil.invalid").Code != 403 {
		t.Fatal("cross origin grant accepted")
	}
	if request("POST", "/api/account/link", cookie, `{"password":"wrong"}`, "https://monitor.fribench.cn").Code != 403 {
		t.Fatal("invalid operator proof accepted")
	}
	if request("POST", "/api/account/link", cookie, `{"password":"fixture-old-password"}`, "https://monitor.fribench.cn").Code != 200 {
		t.Fatal("operator grant failed")
	}
	if request("GET", "/api/snapshot", cookie, "", "").Code != 200 {
		t.Fatal("owner cannot access")
	}
	if request("GET", "/api/snapshot", other, "", "").Code != 401 {
		t.Fatal("another account inherited owner")
	}
	if request("POST", "/api/account/link", other, `{"password":"fixture-old-password"}`, "https://monitor.fribench.cn").Code != 409 {
		t.Fatal("owner changed unexpectedly")
	}
	var access map[string]bool
	if json.Unmarshal(request("GET", "/api/account", cookie, "", "").Body.Bytes(), &access) != nil || !access["authorized"] {
		t.Fatal("account status incorrect")
	}
	legacy := httptest.NewRequest("GET", "/api/snapshot", nil)
	legacy.SetBasicAuth("monitor", "fixture-old-password")
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, legacy)
	if record.Code != 200 {
		t.Fatal("existing operator access changed")
	}
}
