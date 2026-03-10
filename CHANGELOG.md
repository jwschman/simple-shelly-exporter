# Changelog

## [0.1.0] - 2026-03-10

### Fixed
- Removed hardcoded IP address; `SHELLY_HOST` env var is now used correctly
- Fixed `shelly_voltage_volts` metric not appearing in output due to missing gauge registration

### Added
- Versioned Docker Hub image tags via `VERSION` file
- `/health` endpoint for Kubernetes liveness/readiness probes
- Startup validation for required `SHELLY_HOST` env var

### Fixed
- HTTP client now has a 10s timeout to prevent hung scrape goroutine if device is unreachable
- Removed noisy debug logging of full Shelly payload
