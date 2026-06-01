package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testDevice = "device-1"

// build a new *server with a fresh store and a single known device.
func newTestServer() (*server, *http.ServeMux) {
	srv := &server{
		store:   NewStore(),
		devices: []string{testDevice},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", srv.makeDeviceHandler(srv.heartbeatHandler))
	mux.HandleFunc("POST /api/v1/devices/{device_id}/stats", srv.makeDeviceHandler(srv.postStatsHandler))
	mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", srv.makeDeviceHandler(srv.getStatsHandler))

	return srv, mux
}

// serve a single request through the mux and returns the recorder.
func do(mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestUnknownDeviceReturns404(t *testing.T) {
	_, mux := newTestServer()

	// for each endpoint, test that an unknown device ID returns 404
	cases := []struct {
		name   string
		method string
		target string
	}{
		{"heartbeat", http.MethodPost, "/api/v1/devices/ghost/heartbeat"},
		{"post stats", http.MethodPost, "/api/v1/devices/ghost/stats"},
		{"get stats", http.MethodGet, "/api/v1/devices/ghost/stats"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(mux, tc.method, tc.target, "")
			if rec.Code != http.StatusNotFound {
				t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHeartbeatHandler(t *testing.T) {
	t.Run("valid body returns 204 and stores heartbeat", func(t *testing.T) {
		srv, mux := newTestServer()

		rec := do(mux, http.MethodPost, "/api/v1/devices/"+testDevice+"/heartbeat",
			`{"sent_at":"2024-01-01T00:00:00Z"}`)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusNoContent)
		}
		if got := len(srv.store.HeartbeatsFor(testDevice)); got != 1 {
			t.Fatalf("stored %d heartbeats, want 1", got)
		}
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		srv, mux := newTestServer()

		rec := do(mux, http.MethodPost, "/api/v1/devices/"+testDevice+"/heartbeat", `{`)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
		if got := len(srv.store.HeartbeatsFor(testDevice)); got != 0 {
			t.Fatalf("stored %d heartbeats, want 0", got)
		}
	})
}

func TestPostStatsHandler(t *testing.T) {
	t.Run("valid body returns 204 and stores stats", func(t *testing.T) {
		srv, mux := newTestServer()

		rec := do(mux, http.MethodPost, "/api/v1/devices/"+testDevice+"/stats",
			`{"sent_at":"2024-01-01T00:00:00Z","upload_time":1000000000}`)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusNoContent)
		}
		if got := len(srv.store.StatsFor(testDevice)); got != 1 {
			t.Fatalf("stored %d stats, want 1", got)
		}
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		srv, mux := newTestServer()

		rec := do(mux, http.MethodPost, "/api/v1/devices/"+testDevice+"/stats", `not json`)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
		if got := len(srv.store.StatsFor(testDevice)); got != 0 {
			t.Fatalf("stored %d stats, want 0", got)
		}
	})
}

func TestGetStatsErrorPaths(t *testing.T) {
	t.Run("no heartbeats returns 404", func(t *testing.T) {
		_, mux := newTestServer()

		rec := do(mux, http.MethodGet, "/api/v1/devices/"+testDevice+"/stats", "")

		if rec.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
		}
		if !strings.Contains(rec.Body.String(), "no heartbeats for device") {
			t.Fatalf("body = %q, want it to mention missing heartbeats", rec.Body.String())
		}
	})
}

func TestGetStatsHappyPath(t *testing.T) {
	// test the uptime calculation and the average upload time calculation together
	t.Run("computes uptime and average upload time", func(t *testing.T) {
		srv, mux := newTestServer()
		base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		// 3 heartbeats spanning 2 minutes -> 3/2*100 = 150.
		srv.store.AddHeartbeat(Heartbeat{DeviceId: testDevice, SentAt: base})
		srv.store.AddHeartbeat(Heartbeat{DeviceId: testDevice, SentAt: base.Add(1 * time.Minute)})
		srv.store.AddHeartbeat(Heartbeat{DeviceId: testDevice, SentAt: base.Add(2 * time.Minute)})
		// avg of 1s and 3s -> 2s.
		srv.store.AddStats(Stats{DeviceId: testDevice, SentAt: base, UploadTime: 1_000_000_000})
		srv.store.AddStats(Stats{DeviceId: testDevice, SentAt: base, UploadTime: 3_000_000_000})

		rec := do(mux, http.MethodGet, "/api/v1/devices/"+testDevice+"/stats", "")

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}

		var body StatsResponseBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if body.Uptime != 150 {
			t.Errorf("uptime = %v, want 150", body.Uptime)
		}
		if body.AvgUploadTime != "2s" {
			t.Errorf("avg_upload_time = %q, want %q", body.AvgUploadTime, "2s")
		}
	})

	t.Run("zero stats yields 0s average without dividing by zero", func(t *testing.T) {
		srv, mux := newTestServer()
		base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		srv.store.AddHeartbeat(Heartbeat{DeviceId: testDevice, SentAt: base})
		srv.store.AddHeartbeat(Heartbeat{DeviceId: testDevice, SentAt: base.Add(1 * time.Minute)})

		rec := do(mux, http.MethodGet, "/api/v1/devices/"+testDevice+"/stats", "")

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}

		var body StatsResponseBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if body.AvgUploadTime != "0s" {
			t.Errorf("avg_upload_time = %q, want %q", body.AvgUploadTime, "0s")
		}
	})
}
