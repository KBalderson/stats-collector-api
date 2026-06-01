# Stat Collector API

A GO API for collecting and reporting stats from IOT Devices.

## Running the Server

To run the server, use the following command:

```bash
go run .
```

Alternatively, you can run the server using Docker:

```bash
docker compose up --build stats-server
```

## Testing against the SafelyYou Device Simulator

To test against the SafelyYou device simulator, point it to the running docker container's port (currently set to :8888) or run the device simulator docker container:

```bash
docker compose up --build device-simulator
```

**Note:** The simulator writes its output to `results.txt` in its working directory, which is bind-mounted to `./results.txt` on the host so you can inspect the results. An initial `results.txt` is included in the repository because Docker requires the source of a single-file bind mount to already exist (otherwise it creates a *directory* by that name). As the simulator runs, it will populate this file, so it will show as changed in git.

## API Endpoints

- `POST /api/v1/devices/{device_id}/heartbeat`: Receives a heartbeat from a device, indicating it's alive.
- `POST /api/v1/devices/{device_id}/stats`: Receives stats from a device, including sentAt and uploadTime.
- `GET /api/v1/devices/{device_id}/stats`: Returns the uptime percentage and average upload time for a device.

## Data Storage

The server uses an in-memory store to keep track of heartbeats and stats for each device. The store is thread-safe, allowing concurrent access from multiple requests.

## Uptime Calculation

Uptime is calculated based on the timestamps of the first and last heartbeats received for a device. The formula is:

```
Uptime (%) = (Number of Heartbeats / Number of Minutes Between First and Last Heartbeat) * 100
```

## Average Upload Time Calculation

The average upload time is calculated by summing the upload times from all stats received for a device and dividing by the number of stats.

## Error Handling

The API returns appropriate HTTP status codes and error messages for various error conditions, such as invalid request bodies, missing heartbeats, or no stats available for a device.

