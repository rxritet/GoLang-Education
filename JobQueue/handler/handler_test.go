package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jobqueue/store"
	"jobqueue/worker"
)

const errDecodeFmt = "decode error: %v"

// newTestHandler создаёт Handler с быстрым пулом для тестов.
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	s := store.New()
	p := worker.NewPool(s, worker.Config{
		NumWorkers: 1,
		QueueSize:  10,
		JobTimeout: 5 * time.Second,
	})
	t.Cleanup(p.Stop)
	return New(s, p)
}

func TestCreateJob(t *testing.T) {
	h := newTestHandler(t)

	body := bytes.NewBufferString(`{"task":"send_email"}`)
	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateJob(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var resp CreateJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf(errDecodeFmt, err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty job ID")
	}
	if resp.Status != store.StatusQueued {
		t.Errorf("expected status %q, got %q", store.StatusQueued, resp.Status)
	}
}

func TestCreateJobEmptyTask(t *testing.T) {
	h := newTestHandler(t)

	body := bytes.NewBufferString(`{"task":""}`)
	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	rec := httptest.NewRecorder()

	h.CreateJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateJobInvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	rec := httptest.NewRecorder()

	h.CreateJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetJob(t *testing.T) {
	h := newTestHandler(t)

	// Создаём задачу напрямую в store.
	h.Store.Save(&store.Job{
		ID:        "test-id",
		Task:      "resize_image",
		Status:    store.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/jobs/test-id", nil)
	rec := httptest.NewRecorder()

	h.GetJob(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var job store.Job
	if err := json.NewDecoder(rec.Body).Decode(&job); err != nil {
		t.Fatalf(errDecodeFmt, err)
	}
	if job.ID != "test-id" || job.Task != "resize_image" {
		t.Errorf("unexpected job: %+v", job)
	}
}

func TestGetJobNotFound(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/jobs/nonexistent", nil)
	rec := httptest.NewRecorder()

	h.GetJob(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestListJobs(t *testing.T) {
	h := newTestHandler(t)

	h.Store.Save(&store.Job{ID: "1", Task: "a", Status: store.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	h.Store.Save(&store.Job{ID: "2", Task: "b", Status: store.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	rec := httptest.NewRecorder()

	h.ListJobs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var jobs []store.Job
	if err := json.NewDecoder(rec.Body).Decode(&jobs); err != nil {
		t.Fatalf(errDecodeFmt, err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}
