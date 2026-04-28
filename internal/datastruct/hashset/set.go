// Package hashset implementasi generic Set[T] yang di-back oleh hash table.
//
// Use case di app: lookup membership untuk roles, permissions, visited set
// di traversal graph (graph package pakai map[T]struct{} secara langsung,
// hashset di sini sebagai reference + reusable wrapper).
package hashset

// concept:hash-table-map:start
// Set[T] = unordered collection unique elements, di-back oleh hash table.
//
// Go built-in `map[K]struct{}` adalah hash table dengan separate chaining +
// dynamic resize. Pakai `struct{}` (zero-byte) sebagai value lebih hemat memory
// dibanding `bool` (1 byte) — meaningful saat set berisi jutaan elemen.
//
// Operasi O(1) average untuk Add/Contains/Remove. Worst case O(n) saat
// banyak collision (probability rendah dengan good hash function — Go pakai
// random seed per process untuk mitigate hash flooding attack).
type Set[T comparable] struct {
	items map[T]struct{}
}

// New membuat empty set.
func New[T comparable]() *Set[T] {
	return &Set[T]{items: map[T]struct{}{}}
}

// FromSlice membuat set dengan elemen dari slice (dedup otomatis).
func FromSlice[T comparable](xs []T) *Set[T] {
	s := New[T]()
	for _, x := range xs {
		s.Add(x)
	}
	return s
}

// Add insert v. Return true kalau v belum ada (benar-benar baru ditambahkan).
func (s *Set[T]) Add(v T) bool {
	if _, exists := s.items[v]; exists {
		return false
	}
	s.items[v] = struct{}{}
	return true
}

// Contains membership check.
func (s *Set[T]) Contains(v T) bool {
	_, ok := s.items[v]
	return ok
}

// Remove delete v. Return true kalau v ada sebelum delete.
func (s *Set[T]) Remove(v T) bool {
	if _, ok := s.items[v]; !ok {
		return false
	}
	delete(s.items, v)
	return true
}

// Len jumlah elemen.
func (s *Set[T]) Len() int { return len(s.items) }

// ToSlice return elemen sebagai slice. Order tidak deterministik
// (map iteration order Go diacak per process).
func (s *Set[T]) ToSlice() []T {
	out := make([]T, 0, len(s.items))
	for v := range s.items {
		out = append(out, v)
	}
	return out
}

// concept:hash-table-map:end

// Union return set baru = elemen yang ada di s atau other (atau keduanya).
// O(|s| + |other|).
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	out := New[T]()
	for v := range s.items {
		out.items[v] = struct{}{}
	}
	for v := range other.items {
		out.items[v] = struct{}{}
	}
	return out
}

// Intersect return set = elemen di s DAN other.
// O(min(|s|, |other|)).
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	smaller, larger := s, other
	if other.Len() < s.Len() {
		smaller, larger = other, s
	}
	out := New[T]()
	for v := range smaller.items {
		if _, ok := larger.items[v]; ok {
			out.items[v] = struct{}{}
		}
	}
	return out
}

// Difference return s \ other (elemen di s tapi tidak di other).
// O(|s|).
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	out := New[T]()
	for v := range s.items {
		if _, ok := other.items[v]; !ok {
			out.items[v] = struct{}{}
		}
	}
	return out
}
