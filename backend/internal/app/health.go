package app

import "context"

// HealthStatus is the overall state reported by the health check.
type HealthStatus string

// HealthOK means the backend is up and able to serve requests.
const HealthOK HealthStatus = "ok"

// HealthReport is the result of a health check.
type HealthReport struct {
	Status  HealthStatus
	Version string
}

// Health reports whether the backend is able to serve requests.
type Health struct {
	version string
}

// NewHealth returns a health check that reports the given build version.
func NewHealth(version string) *Health {
	return &Health{version: version}
}

// Check reports the current health.
func (h *Health) Check(_ context.Context) HealthReport {
	return HealthReport{Status: HealthOK, Version: h.version}
}
