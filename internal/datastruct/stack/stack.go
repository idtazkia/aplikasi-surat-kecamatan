// Package stack implementasi generic LIFO stack — push / pop / peek dalam O(1).
//
// Reference implementation untuk concept catalog matkul Struktur Data.
// Dipakai sebagai building block di internal/datastruct/graph (DFS traversal)
// dan kandidat untuk undo/redo client-side di Fase 2.
package stack

import "errors"

// concept:stack-lifo:start
// Stack[T] = LIFO container generic. Backing array auto-grow saat push.
// Setiap operasi O(1) amortized — append slice di Go = O(1) amortized
// (occasionally O(n) saat resize, tapi total cost N push = O(N)).
type Stack[T any] struct {
	items []T
}

// New membuat empty stack.
func New[T any]() *Stack[T] { return &Stack[T]{} }

// Push tambah elemen ke top stack.
func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

// Pop ambil elemen dari top, removed.
// Return ErrEmpty kalau stack kosong.
func (s *Stack[T]) Pop() (T, error) {
	var zero T
	n := len(s.items)
	if n == 0 {
		return zero, ErrEmpty
	}
	v := s.items[n-1]
	s.items = s.items[:n-1]
	return v, nil
}

// Peek lihat top tanpa remove.
func (s *Stack[T]) Peek() (T, error) {
	var zero T
	n := len(s.items)
	if n == 0 {
		return zero, ErrEmpty
	}
	return s.items[n-1], nil
}

// Len jumlah elemen di stack.
func (s *Stack[T]) Len() int { return len(s.items) }

// IsEmpty true kalau Len == 0.
func (s *Stack[T]) IsEmpty() bool { return len(s.items) == 0 }

// concept:stack-lifo:end

// ErrEmpty dikembalikan saat Pop atau Peek dari stack kosong.
var ErrEmpty = errors.New("stack: empty")
