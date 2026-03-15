package server_test

import (
	"net/http"
	"testing"
)

func TestBuildProfiles(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/simulation/profiles/build", nil)
	assertStatus(t, rec, http.StatusAccepted)
	assertBodyContains(t, rec, "task_id")
}

func TestListProfilesEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/simulation/profiles", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestRunSimulationMissingDescription(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/simulation/run",
		map[string]any{"fork_point": map[string]string{}})
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "fork_point.description")
}

func TestRunSimulationInvalidJSON(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodPost, "/api/v1/simulation/run", "{bad")
	assertStatus(t, rec, http.StatusBadRequest)
	assertBodyContains(t, rec, "invalid")
}

func TestListSimulationsEmpty(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/simulation/sessions", nil)
	assertStatus(t, rec, http.StatusOK)
}

func TestGetSimulationNotFound(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	rec := doJSON(t, app, http.MethodGet, "/api/v1/simulation/sessions/nonexistent", nil)
	assertStatus(t, rec, http.StatusNotFound)
	assertBodyContains(t, rec, "not found")
}
