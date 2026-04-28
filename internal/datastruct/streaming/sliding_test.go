package streaming

import (
	"sync"
	"testing"
	"time"
)

func TestSlidingWindow_BasicCount(t *testing.T) {
	w, err := NewSlidingWindow(1*time.Second, 60)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		w.Increment()
	}
	if got := w.Count(); got != 5 {
		t.Errorf("count = %d, want 5", got)
	}
}

func TestSlidingWindow_InvalidParams(t *testing.T) {
	if _, err := NewSlidingWindow(0, 10); err == nil {
		t.Error("expected error for bucketSize=0")
	}
	if _, err := NewSlidingWindow(1*time.Second, 0); err == nil {
		t.Error("expected error for bucketCount=0")
	}
}

func TestSlidingWindow_EvictsAfterWindow(t *testing.T) {
	w, _ := NewSlidingWindow(100*time.Millisecond, 10) // window 1 sec total

	clock := time.Now()
	w.SetClock(func() time.Time { return clock })

	// 5 events at t=0
	for i := 0; i < 5; i++ {
		w.Increment()
	}
	if got := w.Count(); got != 5 {
		t.Errorf("immediate count = %d, want 5", got)
	}

	// Advance clock 2 seconds — semua bucket expire
	clock = clock.Add(2 * time.Second)
	if got := w.Count(); got != 0 {
		t.Errorf("after window count = %d, want 0", got)
	}
}

func TestSlidingWindow_PartialWindow(t *testing.T) {
	w, _ := NewSlidingWindow(100*time.Millisecond, 10) // 1 sec window

	clock := time.Now()
	w.SetClock(func() time.Time { return clock })

	// 3 events at t=0
	for i := 0; i < 3; i++ {
		w.Increment()
	}

	// Advance 500ms — masih dalam window
	clock = clock.Add(500 * time.Millisecond)
	for i := 0; i < 2; i++ {
		w.Increment()
	}
	if got := w.Count(); got != 5 {
		t.Errorf("partial window count = %d, want 5", got)
	}

	// Advance lagi 600ms (total 1.1s) — bucket pertama expire
	clock = clock.Add(600 * time.Millisecond)
	if got := w.Count(); got > 2 || got < 0 {
		t.Errorf("count after partial expire = %d, want close to 2", got)
	}
}

func TestSlidingWindow_ConcurrentSafe(t *testing.T) {
	w, _ := NewSlidingWindow(1*time.Second, 60)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				w.Increment()
			}
		}()
	}
	wg.Wait()
	if got := w.Count(); got != 1000 {
		t.Errorf("concurrent count = %d, want 1000", got)
	}
}
