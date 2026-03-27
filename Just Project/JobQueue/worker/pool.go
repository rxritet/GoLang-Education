// Package worker реализует Worker Pool — пул горутин-воркеров,
// читающих задачи из буферизованного канала и обрабатывающих их.
//
// Архитектура синхронизации:
//
//	        POST /jobs
//	            │
//	            ▼
//	   ┌─────────────────┐
//	   │  buffered chan   │  ← буфер = QueueSize (не блокирует HTTP-хендлер)
//	   └────────┬────────┘
//	            │  fan-out
//	   ┌────────┼────────┐
//	   ▼        ▼        ▼
//	worker1  worker2  worker3   ← горутины, читающие из общего канала
//	   │        │        │
//	   └────────┼────────┘
//	            ▼
//	      store.UpdateStatus    ← потокобезопасное обновление
//
// Каждый воркер:
//  1. Блокируется на чтении из канала (ожидает задачу).
//  2. Ставит статус «running».
//  3. Выполняет задачу в рамках context.WithTimeout (жёсткий дедлайн).
//  4. Ставит «completed», «failed» или «cancelled» в зависимости от исхода.
//
// Graceful shutdown: при вызове Pool.Stop() закрывается канал задач,
// воркеры дочитывают оставшиеся элементы и завершаются; main ждёт
// через sync.WaitGroup.
package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"jobqueue/store"
)

// ---------- Конфигурация ----------

// Config задаёт параметры пула.
type Config struct {
	NumWorkers int           // количество горутин-воркеров
	QueueSize  int           // размер буфера канала задач
	JobTimeout time.Duration // максимальное время выполнения одной задачи
}

// DefaultConfig возвращает разумные значения по умолчанию.
func DefaultConfig() Config {
	return Config{
		NumWorkers: 3,
		QueueSize:  100,
		JobTimeout: 30 * time.Second,
	}
}

// ---------- Pool ----------

// Pool управляет буферизованным каналом задач и набором воркеров.
type Pool struct {
	jobs  chan string // ID задач; буферизованный, чтобы POST не блокировался
	store *store.MemoryStore
	cfg   Config
	wg    sync.WaitGroup // ожидание завершения всех воркеров при shutdown
}

// NewPool создаёт пул и запускает воркеры.
func NewPool(s *store.MemoryStore, cfg Config) *Pool {
	p := &Pool{
		jobs:  make(chan string, cfg.QueueSize), // буферизованный канал
		store: s,
		cfg:   cfg,
	}

	// Запускаем N воркеров. Каждый — отдельная горутина.
	for i := 1; i <= cfg.NumWorkers; i++ {
		p.wg.Add(1)
		go p.runWorker(i)
	}

	log.Printf("[pool] started %d workers (queue buffer=%d, job timeout=%s)",
		cfg.NumWorkers, cfg.QueueSize, cfg.JobTimeout)

	return p
}

// Submit помещает ID задачи в канал. Возвращает false, если очередь переполнена.
func (p *Pool) Submit(jobID string) bool {
	select {
	case p.jobs <- jobID:
		return true
	default:
		// Буфер полон — задача отклоняется.
		return false
	}
}

// Stop закрывает канал задач и ожидает завершения всех воркеров (graceful shutdown).
func (p *Pool) Stop() {
	log.Println("[pool] shutting down…")
	close(p.jobs) // после этого range в воркерах завершится
	p.wg.Wait()   // блокируемся, пока все воркеры не вызовут wg.Done()
	log.Println("[pool] all workers stopped")
}

// ---------- Внутренняя логика воркера ----------

// runWorker — главный цикл одного воркера. Читает ID из канала,
// извлекает задачу из Store, обрабатывает и обновляет статус.
func (p *Pool) runWorker(id int) {
	defer p.wg.Done() // сигнализируем о завершении

	// range по каналу: цикл продолжается, пока канал открыт.
	// После close(p.jobs) цикл дочитает оставшиеся элементы и завершится.
	for jobID := range p.jobs {
		p.processJob(id, jobID)
	}

	log.Printf("[worker %d] stopped", id)
}

// processJob обрабатывает одну задачу с контролем таймаута через context.
func (p *Pool) processJob(workerID int, jobID string) {
	// Создаём контекст с дедлайном. Если задача не уложится в JobTimeout,
	// ctx.Done() будет закрыт, и мы пометим задачу как «cancelled».
	ctx, cancel := context.WithTimeout(context.Background(), p.cfg.JobTimeout)
	defer cancel() // освобождаем ресурсы контекста

	// Переводим статус в «running».
	_ = p.store.UpdateStatus(jobID, store.StatusRunning, "")
	log.Printf("[worker %d] processing job %s", workerID, jobID)

	// Имитация выполнения задачи в отдельной горутине,
	// чтобы select мог отслеживать таймаут/отмену контекста.
	done := make(chan error, 1)
	go func() {
		done <- executeTask(ctx, jobID)
	}()

	select {
	case err := <-done:
		// Задача завершилась (успех или ошибка).
		if err != nil {
			_ = p.store.UpdateStatus(jobID, store.StatusFailed, err.Error())
			log.Printf("[worker %d] job %s failed: %v", workerID, jobID, err)
		} else {
			_ = p.store.UpdateStatus(jobID, store.StatusCompleted, "")
			log.Printf("[worker %d] job %s completed", workerID, jobID)
		}

	case <-ctx.Done():
		// Контекст отменён (timeout или явная отмена).
		_ = p.store.UpdateStatus(jobID, store.StatusCancelled, ctx.Err().Error())
		log.Printf("[worker %d] job %s cancelled: %v", workerID, jobID, ctx.Err())
	}
}

// executeTask имитирует полезную работу. В реальном сервисе здесь
// была бы отправка email, ресайз картинки и т.д.
// Функция вынесена, чтобы в тестах можно было подменить логику.
var executeTask = defaultExecuteTask

func defaultExecuteTask(ctx context.Context, jobID string) error {
	// Имитируем работу 2–4 секунды.
	sleepDuration := 2*time.Second + time.Duration(len(jobID)%3)*time.Second

	select {
	case <-time.After(sleepDuration):
		return nil // «работа» завершена успешно
	case <-ctx.Done():
		return fmt.Errorf("cancelled: %w", ctx.Err())
	}
}
