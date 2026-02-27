package collector

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestNewCollectsImmediately(t *testing.T) {
	c := New(1 * time.Hour) // большой интервал — тикер не сработает

	snap := c.Snapshot()

	// Первый снимок должен быть заполнен сразу при New.
	if snap.AllocBytes == 0 {
		t.Error("expected non-zero AllocBytes after New")
	}
	if snap.SysBytes == 0 {
		t.Error("expected non-zero SysBytes")
	}
	if snap.NumGoroutines == 0 {
		t.Error("expected non-zero NumGoroutines")
	}
	if snap.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
	if snap.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

func TestSnapshotReturnsCopy(t *testing.T) {
	c := New(1 * time.Hour)

	s1 := c.Snapshot()
	s1.NumGoroutines = -999 // мутируем копию

	s2 := c.Snapshot()
	if s2.NumGoroutines == -999 {
		t.Error("Snapshot should return a copy; original was mutated")
	}
}

func TestRunUpdatesMetrics(t *testing.T) {
	c := New(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx)

	// Ждём несколько тиков.
	time.Sleep(350 * time.Millisecond)
	cancel()

	snap := c.Snapshot()
	if snap.NumGoroutines == 0 {
		t.Error("metrics should be updated after ticks")
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	c := New(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		c.Run(ctx) // должен завершиться при cancel
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// горутина завершилась — OK
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after context cancel")
	}
}

func TestMetricsFieldsAreReasonable(t *testing.T) {
	c := New(1 * time.Hour)
	snap := c.Snapshot()

	if snap.GOOS != runtime.GOOS {
		t.Errorf("GOOS = %q, want %q", snap.GOOS, runtime.GOOS)
	}
	if snap.GOARCH != runtime.GOARCH {
		t.Errorf("GOARCH = %q, want %q", snap.GOARCH, runtime.GOARCH)
	}
	if snap.NumCPU != runtime.NumCPU() {
		t.Errorf("NumCPU = %d, want %d", snap.NumCPU, runtime.NumCPU())
	}
	if snap.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", snap.GoVersion, runtime.Version())
	}
	if snap.HeapAllocBytes == 0 {
		t.Error("expected non-zero HeapAllocBytes")
	}
}

func TestUptimeIncreases(t *testing.T) {
	c := New(500 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx)

	// Sleep > 1s so that uptime (rounded to seconds) is non-zero.
	time.Sleep(1100 * time.Millisecond)
	cancel()

	snap := c.Snapshot()
	if snap.Uptime == "" || snap.Uptime == "0s" {
		t.Errorf("uptime should be > 0, got %q", snap.Uptime)
	}
}
