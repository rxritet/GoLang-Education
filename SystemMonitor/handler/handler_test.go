package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sysmonitor/collector"
)

const expectedStatusOK = "expected 200, got %d"

func newTestHandler() *Handler {
	c := collector.New(1 * time.Hour)
	return New(c)
}

func TestGetMetrics(t *testing.T) {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	h.GetMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(expectedStatusOK, rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var m collector.Metrics
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if m.AllocBytes == 0 {
		t.Error("expected non-zero AllocBytes")
	}
	if m.NumGoroutines == 0 {
		t.Error("expected non-zero NumGoroutines")
	}
}

func TestHealth(t *testing.T) {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(expectedStatusOK, rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", resp["status"])
	}
}

func TestDashboard(t *testing.T) {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Dashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(expectedStatusOK, rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}

	body := rec.Body.String()
	if len(body) < 100 {
		t.Error("expected HTML body to be non-trivial")
	}
}
