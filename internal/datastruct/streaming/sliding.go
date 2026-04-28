// Package streaming implementasi sliding window counter dan probabilistic
// data structures untuk streaming queries.
//
// Use case di app (Fase 6+ dashboard real-time): hitung event dalam jendela
// waktu tertentu (1 jam, 24 jam) tanpa scan ulang seluruh history.
package streaming

import (
	"errors"
	"sync"
	"time"
)

// concept:streaming-sliding-window:start
// SlidingWindow counter dengan bucketed approach: bagi window jadi N bucket
// berdurasi sama. Tiap bucket simpan count saja, bukan event individual.
// Memory O(N) konstan, granularity = window/N.
//
// Thread-safe via mutex. Setiap operasi O(N) untuk count (jumlah semua bucket),
// O(1) untuk increment. Untuk N kecil (mis. 60), praktis konstan.
type SlidingWindow struct {
	mu          sync.Mutex
	bucketSize  time.Duration // durasi 1 bucket
	bucketCount int           // jumlah bucket di window
	buckets     []int         // count per bucket
	timestamps  []time.Time   // start time tiap bucket
	now         func() time.Time // injectable untuk testing
}

// NewSlidingWindow buat counter dengan window total = bucketSize × bucketCount.
// Contoh: NewSlidingWindow(1*time.Minute, 60) = window 1 jam dengan granularity 1 menit.
func NewSlidingWindow(bucketSize time.Duration, bucketCount int) (*SlidingWindow, error) {
	if bucketSize <= 0 {
		return nil, errors.New("streaming: bucketSize must be positive")
	}
	if bucketCount <= 0 {
		return nil, errors.New("streaming: bucketCount must be positive")
	}
	w := &SlidingWindow{
		bucketSize:  bucketSize,
		bucketCount: bucketCount,
		buckets:     make([]int, bucketCount),
		timestamps:  make([]time.Time, bucketCount),
		now:         time.Now,
	}
	now := w.now()
	for i := range w.timestamps {
		w.timestamps[i] = now
	}
	return w, nil
}

// Increment tambah 1 ke bucket aktif (current time).
// Auto-evict bucket yang sudah expire (di luar window).
func (w *SlidingWindow) Increment() {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now()
	idx := w.bucketIndex(now)
	w.evictExpired(now)
	w.buckets[idx]++
	w.timestamps[idx] = now
}

// Count return total count dalam window aktif.
// Bucket yang sudah expire di-skip.
func (w *SlidingWindow) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now()
	threshold := now.Add(-time.Duration(w.bucketCount) * w.bucketSize)
	total := 0
	for i, t := range w.timestamps {
		if !t.Before(threshold) {
			total += w.buckets[i]
		}
	}
	return total
}

// bucketIndex map current time ke bucket index (0..N-1).
// Pakai modulo nanoseconds — distribusi jadi tidak strict chronological,
// tapi cukup untuk approximate sliding window.
func (w *SlidingWindow) bucketIndex(now time.Time) int {
	return int(now.UnixNano()/int64(w.bucketSize)) % w.bucketCount
}

// evictExpired reset bucket yang timestamp-nya sudah lewat window.
func (w *SlidingWindow) evictExpired(now time.Time) {
	threshold := now.Add(-time.Duration(w.bucketCount) * w.bucketSize)
	for i, t := range w.timestamps {
		if t.Before(threshold) {
			w.buckets[i] = 0
		}
	}
}

// concept:streaming-sliding-window:end

// SetClock injectable untuk testing — replace time.Now dengan fixed clock.
func (w *SlidingWindow) SetClock(now func() time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.now = now
}
