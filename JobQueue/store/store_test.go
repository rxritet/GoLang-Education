package store

import (
	"testing"
	"time"
)

func TestSaveAndGet(t *testing.T) {
	s := New()

	job := &Job{
		ID:        "job-1",
		Task:      "send_email",
		Status:    StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.Save(job)

	got, err := s.Get("job-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "job-1" || got.Task != "send_email" || got.Status != StatusQueued {
		t.Errorf("unexpected job: %+v", got)
	}
}

func TestGetNotFound(t *testing.T) {
	s := New()

	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestUpdateStatus(t *testing.T) {
	s := New()
	s.Save(&Job{ID: "job-2", Task: "resize_image", Status: StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	err := s.UpdateStatus("job-2", StatusRunning, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := s.Get("job-2")
	if got.Status != StatusRunning {
		t.Errorf("expected status %q, got %q", StatusRunning, got.Status)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	s := New()

	err := s.UpdateStatus("nope", StatusRunning, "")
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestList(t *testing.T) {
	s := New()
	s.Save(&Job{ID: "a", Task: "t1", Status: StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	s.Save(&Job{ID: "b", Task: "t2", Status: StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	jobs := s.List()
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestGetReturnsCopy(t *testing.T) {
	s := New()
	s.Save(&Job{ID: "c", Task: "t", Status: StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	got, _ := s.Get("c")
	got.Status = StatusCompleted // мутируем копию

	original, _ := s.Get("c")
	if original.Status != StatusQueued {
		t.Error("Get should return a copy; original was mutated")
	}
}
