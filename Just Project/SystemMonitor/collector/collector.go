// Package collector отвечает за периодический сбор метрик runtime
// в фоновой горутине и потокобезопасное хранение последнего снимка.
//
// Синхронизация:
//
//	Фоновая горутина (Ticker)       HTTP-хендлер GET /metrics
//	────────────────────────        ──────────────────────────
//	     mu.Lock()                       mu.RLock()
//	     обновляет snapshot              читает snapshot (копию)
//	     mu.Unlock()                     mu.RUnlock()
//
// RWMutex позволяет множеству HTTP-читателей работать параллельно,
// блокируя только на время короткой записи (~µs).
package collector

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"
)

// ---------- Модели ----------

// Metrics — снимок метрик, отдаваемый по HTTP.
type Metrics struct {
	// Память
	AllocBytes      uint64 `json:"alloc_bytes"`       // байты, выделенные и ещё не освобождённые
	TotalAllocBytes uint64 `json:"total_alloc_bytes"` // суммарно выделено за всё время
	SysBytes        uint64 `json:"sys_bytes"`         // байты, полученные от ОС
	HeapAllocBytes  uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes    uint64 `json:"heap_sys_bytes"`
	HeapObjects     uint64 `json:"heap_objects"` // количество живых объектов в куче

	// GC
	NumGC        uint32  `json:"num_gc"`         // количество завершённых циклов GC
	GCPauseNs    uint64  `json:"gc_pause_ns"`    // длительность последней паузы GC (нс)
	GCCPUPercent float64 `json:"gc_cpu_percent"` // доля CPU, потраченная на GC

	// Горутины
	NumGoroutines int `json:"num_goroutines"`

	// Мета
	GoVersion string    `json:"go_version"`
	GOOS      string    `json:"goos"`
	GOARCH    string    `json:"goarch"`
	NumCPU    int       `json:"num_cpu"`
	Uptime    string    `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
}

// ---------- Collector ----------

// Collector периодически собирает метрики и хранит последний снимок.
type Collector struct {
	mu        sync.RWMutex // защищает snapshot
	snapshot  Metrics
	interval  time.Duration
	startTime time.Time
}

// New создаёт Collector с заданным интервалом опроса.
func New(interval time.Duration) *Collector {
	c := &Collector{
		interval:  interval,
		startTime: time.Now(),
	}
	// Собираем первый снимок сразу, чтобы GET /metrics не возвращал пустоту.
	c.collect()
	return c
}

// Snapshot возвращает копию последнего снимка (потокобезопасно).
func (c *Collector) Snapshot() Metrics {
	c.mu.RLock() // разделяемая блокировка — читатели не блокируют друг друга
	defer c.mu.RUnlock()
	return c.snapshot // копия структуры (value type)
}

// Run запускает фоновый сбор метрик. Блокируется до отмены контекста.
//
// Типичное использование:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	go collector.Run(ctx)   // фоновая горутина
//	...
//	cancel()                // остановка при shutdown
//
// При отмене контекста тикер останавливается и горутина завершается.
func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop() // освобождаем ресурсы тикера

	log.Printf("[collector] started (interval=%s)", c.interval)

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-ctx.Done():
			// Контекст отменён — graceful shutdown.
			log.Println("[collector] stopped")
			return
		}
	}
}

// collect читает метрики runtime и обновляет снимок под Lock.
func (c *Collector) collect() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // ~STW, но очень быстро

	snapshot := Metrics{
		AllocBytes:      m.Alloc,
		TotalAllocBytes: m.TotalAlloc,
		SysBytes:        m.Sys,
		HeapAllocBytes:  m.HeapAlloc,
		HeapSysBytes:    m.HeapSys,
		HeapObjects:     m.HeapObjects,

		NumGC:        m.NumGC,
		GCCPUPercent: m.GCCPUFraction * 100,

		NumGoroutines: runtime.NumGoroutine(),

		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		NumCPU:    runtime.NumCPU(),
		Uptime:    time.Since(c.startTime).Round(time.Second).String(),
		Timestamp: time.Now(),
	}

	// Последняя пауза GC (кольцевой буфер из 256 элементов).
	if m.NumGC > 0 {
		snapshot.GCPauseNs = m.PauseNs[(m.NumGC+255)%256]
	}

	c.mu.Lock() // эксклюзивная блокировка — обновляем данные
	c.snapshot = snapshot
	c.mu.Unlock()
}
