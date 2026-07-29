package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Max0314/fribench-monitor/internal/metrics"
	"github.com/Max0314/fribench-monitor/internal/monitor"
)

func TestSnapshotAndHealthEndpoints(t *testing.T) {
	state := monitor.NewState(5*time.Second, 24*time.Hour)
	state.Update(metrics.Snapshot{
		Timestamp: time.Now(),
		Source:    "live",
		Hostname:  "fribench",
	})

	handler, err := NewHandler(state, testAssets(), Auth{})
	if err != nil {
		t.Fatal(err)
	}

	healthRecorder := httptest.NewRecorder()
	handler.ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRecorder.Code != http.StatusOK || healthRecorder.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q", healthRecorder.Code, healthRecorder.Body.String())
	}

	snapshotRecorder := httptest.NewRecorder()
	handler.ServeHTTP(snapshotRecorder, httptest.NewRequest(http.MethodGet, "/api/snapshot", nil))
	if snapshotRecorder.Code != http.StatusOK {
		t.Fatalf("snapshot response = %d", snapshotRecorder.Code)
	}
	var payload struct {
		Snapshot metrics.Snapshot `json:"snapshot"`
	}
	if err := json.Unmarshal(snapshotRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Snapshot.Hostname != "fribench" {
		t.Fatalf("snapshot hostname = %q", payload.Snapshot.Hostname)
	}
}

func TestAuthProtectsDashboardButNotHealth(t *testing.T) {
	state := monitor.NewState(5*time.Second, time.Hour)
	handler, err := NewHandler(state, testAssets(), Auth{
		Username: "monitor",
		Password: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("dashboard response = %d, want 401", unauthorized.Code)
	}

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health response = %d, want 200", health.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetBasicAuth("monitor", "secret")
	authorized := httptest.NewRecorder()
	handler.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized dashboard response = %d", authorized.Code)
	}
}

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><title>Fribench Monitor</title>"),
		},
	}
}
