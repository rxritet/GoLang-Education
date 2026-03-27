package worker

import (
	"context"
	"testing"
	"time"

	"jobqueue/store"
)

// ---------- Хелперы ----------

// withFastExecutor подменяет executeTask на быстрый вариант и восстанавливает
// оригинал после теста.
func withFastExecutor(t *testing.T) {
	t.Helper()
	original := executeTask
	executeTask = func(_ context.Context, _ string) error {
		return nil // мгновенное «выполнение»
	}
	t.Cleanup(func() { executeTask = original })
}

// ---------- Тесты ----------

func TestPoolProcessesJob(t *testing.T) {
	withFastExecutor(t)

	s := store.New()
	p := NewPool(s, Config{NumWorkers: 1, QueueSize: 10, JobTimeout: 5 * time.Second})
	defer p.Stop()

	s.Save(&store.Job{
		ID: "j1", Task: "test", Status: store.StatusQueued,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	if !p.Submit("j1") {
		t.Fatal("submit should succeed")
	}

	// Ждём обработки.
	time.Sleep(200 * time.Millisecond)

	job, err := s.Get("j1")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != store.StatusCompleted {
		t.Errorf("expected status %q, got %q", store.StatusCompleted, job.Status)
	}
}

func TestPoolMultipleJobs(t *testing.T) {
	withFastExecutor(t)

	s := store.New()
	p := NewPool(s, Config{NumWorkers: 3, QueueSize: 20, JobTimeout: 5 * time.Second})
	defer p.Stop()

	ids := []string{"a", "b", "c", "d", "e"}
	for _, id := range ids {
		s.Save(&store.Job{
			ID: id, Task: "work", Status: store.StatusQueued,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
		p.Submit(id)
	}

	time.Sleep(500 * time.Millisecond)

	for _, id := range ids {
		job, _ := s.Get(id)
		if job.Status != store.StatusCompleted {
			t.Errorf("job %s: expected %q, got %q", id, store.StatusCompleted, job.Status)
		}
	}
}

func TestPoolQueueFull(t *testing.T) {
	withFastExecutor(t)

	s := store.New()
	// Буфер = 1, воркер = 0 (не запускаем воркеров, чтобы канал оставался полным).
	p := &Pool{
		jobs:  make(chan string, 1),
		store: s,
		cfg:   Config{},
	}

	// Первый submit занимает единственный слот.
	if !p.Submit("x") {
		t.Fatal("first submit should succeed")
	}
	// Второй должен вернуть false — буфер полон.
	if p.Submit("y") {
		t.Fatal("second submit should fail (queue full)")
	}
}

func TestPoolJobTimeout(t *testing.T) {
	// Подменяем executor на «медленный» — 5 секунд.
	original := executeTask
	executeTask = func(ctx context.Context, _ string) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	t.Cleanup(func() { executeTask = original })

	s := store.New()
	// Таймаут 300ms — задача не успеет.
	p := NewPool(s, Config{NumWorkers: 1, QueueSize: 5, JobTimeout: 300 * time.Millisecond})
	defer p.Stop()

	s.Save(&store.Job{
		ID: "slow", Task: "heavy", Status: store.StatusQueued,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	p.Submit("slow")

	time.Sleep(600 * time.Millisecond)

	job, _ := s.Get("slow")
	if job.Status != store.StatusCancelled {
		t.Errorf("expected %q, got %q", store.StatusCancelled, job.Status)
	}
}
