package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"os"
)

func main() {
	devices, err := loadDevices()
	if err != nil {
		log.Fatalf("failed to load devices: %v", err)
	}

	var srv *server
	srv = &server{
		store:   NewStore(),
		devices: devices,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", srv.makeDeviceHandler(srv.heartbeatHandler))
	mux.HandleFunc("POST /api/v1/devices/{device_id}/stats", srv.makeDeviceHandler(srv.postStatsHandler))
	mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", srv.makeDeviceHandler(srv.getStatsHandler))

	addr := ":80"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("stats-server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func loadDevices() ([]string, error) {
	f, err := os.Open("devices.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	devices := make([]string, 0, len(records)-1)
	for _, r := range records[1:] {
		devices = append(devices, r[0])
	}
	return devices, nil
}
