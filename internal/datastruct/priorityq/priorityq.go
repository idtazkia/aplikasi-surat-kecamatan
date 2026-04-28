// Package priorityq implementasi binary min-heap untuk priority queue.
//
// Use case di app: sync queue retry priority (Fase 4) — operasi yang lebih
// lama menunggu naik prioritas; reminder/deadline scheduling (Fase 6).
//
// Implementasi pakai array-based complete binary tree:
//
//	parent(i) = (i - 1) / 2
//	left(i)   = 2*i + 1
//	right(i)  = 2*i + 2
package priorityq

import "errors"

// concept:heap-priority-queue:start
// Heap[T] = binary min-heap. Compare function menentukan ordering — return
// negatif kalau a < b, 0 kalau equal, positif kalau a > b. Element dengan
// nilai compare terkecil = root, accessible via Peek/Pop dalam O(1)/O(log n).
//
// Untuk max-heap, balikan compare semantic (return positif untuk a < b).
type Heap[T any] struct {
	items   []T
	compare func(a, b T) int
}

// New membuat empty heap dengan compare function.
func New[T any](compare func(a, b T) int) *Heap[T] {
	return &Heap[T]{compare: compare}
}

// Push insert item ke heap. O(log n) — sift-up dari leaf.
func (h *Heap[T]) Push(v T) {
	h.items = append(h.items, v)
	h.siftUp(len(h.items) - 1)
}

// Pop ambil min item, removed. Return ErrEmpty kalau kosong. O(log n).
func (h *Heap[T]) Pop() (T, error) {
	var zero T
	n := len(h.items)
	if n == 0 {
		return zero, ErrEmpty
	}
	min := h.items[0]
	last := h.items[n-1]
	h.items = h.items[:n-1]
	if len(h.items) > 0 {
		h.items[0] = last
		h.siftDown(0)
	}
	return min, nil
}

// Peek lihat min tanpa remove. O(1).
func (h *Heap[T]) Peek() (T, error) {
	var zero T
	if len(h.items) == 0 {
		return zero, ErrEmpty
	}
	return h.items[0], nil
}

// Len jumlah item di heap.
func (h *Heap[T]) Len() int { return len(h.items) }

// siftUp restorasi heap-property dari index i ke atas.
// Swap dengan parent kalau item lebih kecil dari parent.
func (h *Heap[T]) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h.compare(h.items[i], h.items[parent]) >= 0 {
			break
		}
		h.items[i], h.items[parent] = h.items[parent], h.items[i]
		i = parent
	}
}

// siftDown restorasi heap-property dari index i ke bawah.
// Swap dengan child terkecil kalau item lebih besar dari child terkecil.
func (h *Heap[T]) siftDown(i int) {
	n := len(h.items)
	for {
		l := 2*i + 1
		r := 2*i + 2
		smallest := i
		if l < n && h.compare(h.items[l], h.items[smallest]) < 0 {
			smallest = l
		}
		if r < n && h.compare(h.items[r], h.items[smallest]) < 0 {
			smallest = r
		}
		if smallest == i {
			return
		}
		h.items[i], h.items[smallest] = h.items[smallest], h.items[i]
		i = smallest
	}
}

// concept:heap-priority-queue:end

// ErrEmpty dikembalikan saat Pop atau Peek dari heap kosong.
var ErrEmpty = errors.New("priorityq: empty")
