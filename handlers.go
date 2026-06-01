package main

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"time"
)

// server holds the dependencies shared by the HTTP handlers. Handlers are
// methods on *server, so they reach the store and device list via the receiver
// instead of package-level globals.
type server struct {
	store   *Store
	devices []string
}

// makeDeviceHandler wraps a device-scoped handler, rejecting requests for
// unknown device IDs with 404 before the inner handler runs.
func (s *server) makeDeviceHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceId := r.PathValue("device_id")
		if !slices.Contains(s.devices, deviceId) {
			http.NotFound(w, r)
			return
		}

		fn(w, r, deviceId)
	}
}

type HeartBeatRequestBody struct {
	SentAt time.Time `json:"sent_at"`
}

type StatsRequestBody struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int       `json:"upload_time"` // in nanoseconds
}
type StatsResponseBody struct {
	Uptime        float64 `json:"uptime"`
	AvgUploadTime string  `json:"avg_upload_time"`
}

func (s *server) heartbeatHandler(w http.ResponseWriter, r *http.Request, deviceId string) {
	var body HeartBeatRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.store.AddHeartbeat(Heartbeat{
		DeviceId: deviceId,
		SentAt:   body.SentAt,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) postStatsHandler(w http.ResponseWriter, r *http.Request, deviceId string) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var body StatsRequestBody
	if err := json.Unmarshal(raw, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.store.AddStats(Stats{
		DeviceId:   deviceId,
		SentAt:     body.SentAt,
		UploadTime: body.UploadTime,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getStatsHandler(w http.ResponseWriter, r *http.Request, deviceId string) {
	// Snapshot this device's heartbeats from the store (read-locked internally).
	deviceHeartBeats := s.store.HeartbeatsFor(deviceId)
	if len(deviceHeartBeats) == 0 {
		http.Error(w, "no heartbeats for device", http.StatusNotFound)
		return
	}

	firstHeartBeat := deviceHeartBeats[0]
	lastHeartBeat := deviceHeartBeats[len(deviceHeartBeats)-1]
	numMinutesBetweenFirstAndLastHeartbeat := int(lastHeartBeat.SentAt.Sub(firstHeartBeat.SentAt).Minutes())

	if numMinutesBetweenFirstAndLastHeartbeat == 0 {
		http.Error(w, "not enough heartbeats to calculate uptime", http.StatusBadRequest)
		return
	}

	uptime := (float64(len(deviceHeartBeats)) / float64(numMinutesBetweenFirstAndLastHeartbeat)) * 100

	deviceStats := s.store.StatsFor(deviceId)

	// Calculate average upload time
	var totalUploadTime int
	for _, s := range deviceStats {
		totalUploadTime += s.UploadTime
	}
	avgUploadTime := 0
	if len(deviceStats) > 0 {
		avgUploadTime = totalUploadTime / len(deviceStats)
	}

	response := StatsResponseBody{
		Uptime:        uptime,
		AvgUploadTime: time.Duration(avgUploadTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
