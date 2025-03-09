package circuitbreaker

import (
	"sync"
	"testing"
	"time"
)

func TestBreaker_ClosedToOpen(t *testing.T) {
	cfg := Config{FailureThreshold: 3, OpenDuration: time.Minute, HalfOpenProbes: 1}
	b := New(cfg)

	for i := 0; i < 2; i++ {
		if err := b.Allow(); err != nil {
			t.Fatalf("iteration %d: expected nil, got %v", i, err)
		}
		b.RecordFailure()
	}
	if b.State() != StateClosed {
		t.Fatalf("expected Closed at 2 failures, got %s", b.State())
	}

	if err := b.Allow(); err != nil {
		t.Fatal(err)
	}
	b.RecordFailure() // 3rd failure — triggers Open

	if b.State() != StateOpen {
		t.Fatalf("expected Open after %d failures, got %s", cfg.FailureThreshold, b.State())
	}
}

func TestBreaker_OpenReturnsFast(t *testing.T) {
	cfg := Config{FailureThreshold: 1, OpenDuration: time.Hour, HalfOpenProbes: 1}
	b := New(cfg)
	b.Allow()
	b.RecordFailure()

	if err := b.Allow(); err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreaker_OpenToHalfOpenAfterTimeout(t *testing.T) {
	cfg := Config{FailureThreshold: 1, OpenDuration: 10 * time.Millisecond, HalfOpenProbes: 1}
	b := New(cfg)
	b.Allow()
	b.RecordFailure()

	time.Sleep(20 * time.Millisecond)

	if err := b.Allow(); err != nil {
		t.Fatalf("expected nil (half-open probe), got %v", err)
	}
	if b.State() != StateHalfOpen {
		t.Errorf("expected HalfOpen, got %s", b.State())
	}
}

func TestBreaker_HalfOpenSinglePermit(t *testing.T) {
	cfg := Config{FailureThreshold: 1, OpenDuration: 10 * time.Millisecond, HalfOpenProbes: 1}
	b := New(cfg)
	b.Allow()
	b.RecordFailure()
	time.Sleep(20 * time.Millisecond)

	var wg sync.WaitGroup
	errs := make([]error, 5)
	for i := 0; i < 5; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = b.Allow()
		}()
	}
	wg.Wait()

	nilCount := 0
	for _, err := range errs {
		if err == nil {
			nilCount++
		}
	}
	// Exactly 1 probe must be allowed through.
	if nilCount != 1 {
		t.Errorf("expected exactly 1 probe through half-open, got %d", nilCount)
	}
}

func TestBreaker_HalfOpenSuccessCloses(t *testing.T) {
	cfg := Config{FailureThreshold: 1, OpenDuration: 10 * time.Millisecond, HalfOpenProbes: 1}
	b := New(cfg)
	b.Allow()
	b.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	b.Allow() // enters half-open
	b.RecordSuccess()

	if b.State() != StateClosed {
		t.Errorf("expected Closed after probe success, got %s", b.State())
	}
}
