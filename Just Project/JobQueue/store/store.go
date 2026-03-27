// Package store реализует слой хранения задач в памяти.
//
// Потокобезопасность обеспечивается через sync.RWMutex:
//   - RLock / RUnlock — для операций чтения (Get, List), позволяя
//     множеству горутин читать одновременно.
//   - Lock / Unlock — для операций записи (Save, UpdateStatus),
//     блокируя все остальные горутины на время изменения.
package store

import (
	"errors"
	"sync"
	"time"
)

// ErrNotFound возвращается при обращении к несуществующей задаче.
var ErrNotFound = errors.New("job not found")

// ---------- Модели ----------

// Status описывает текущее состояние задачи.
type Status string

const (
	StatusQueued    Status = "queued"    // задача в очереди, ждёт воркера
	StatusRunning   Status = "running"   // воркер выполняет задачу
	StatusCompleted Status = "completed" // задача успешно завершена
	StatusFailed    Status = "failed"    // задача завершилась с ошибкой
	StatusCancelled Status = "cancelled" // задача отменена через context
)

// Job содержит полное описание задачи и её текущее состояние.
type Job struct {
	ID        string    `json:"id"`
	Task      string    `json:"task"`
	Status    Status    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------- In-memory хранилище ----------

// MemoryStore — потокобезопасное хранилище задач в памяти.
type MemoryStore struct {
	mu   sync.RWMutex    // защищает jobs
	jobs map[string]*Job // id → Job
}

// New создаёт пустое хранилище.
func New() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*Job),
	}
}

// Save добавляет новую задачу. Вызывается один раз при создании.
func (s *MemoryStore) Save(job *Job) {
	s.mu.Lock() // эксклюзивная блокировка — никто не читает и не пишет
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Get возвращает копию задачи по ID (или ошибку, если не найдена).
// Возвращаем копию, чтобы вызывающий код не мог изменить оригинал без блокировки.
func (s *MemoryStore) Get(id string) (Job, error) {
	s.mu.RLock() // разделяемая блокировка — можно читать параллельно
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	return *job, nil // копия
}

// UpdateStatus атомарно обновляет статус и (опционально) текст ошибки.
func (s *MemoryStore) UpdateStatus(id string, status Status, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return ErrNotFound
	}
	job.Status = status
	job.Error = errMsg
	job.UpdatedAt = time.Now()
	return nil
}

// List возвращает снимок всех задач (копии).
func (s *MemoryStore) List() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		result = append(result, *j)
	}
	return result
}
