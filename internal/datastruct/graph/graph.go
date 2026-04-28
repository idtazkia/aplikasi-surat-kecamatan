// Package graph implementasi directed graph dengan adjacency list.
// Mendukung BFS, DFS, dan cycle detection (untuk validasi DAG).
//
// Use case di app: validasi `surat_references` sebelum insert (cycle detect),
// shortest path antar surat (Skenario 4), topological order untuk dependent
// operations (sync queue Fase 4).
package graph

// concept:graph-adjacency-list:start
// Graph[T] = directed graph generic dengan adjacency list + BFS/DFS traversal.
// Memory O(V + E). Cocok untuk sparse graph (kebanyakan kasus real-world).
// Tidak generic-hashable di Go 1.22 — vertex harus comparable type
// (string, int, struct tanpa slice/map/func).
//
// Region marker ini cover: struktur (adjacency list), AddVertex/AddEdge,
// BFS, DFS. Cycle detection + topological sort di marker dag-cycle-detection
// terpisah karena fokus algoritma berbeda.
type Graph[T comparable] struct {
	adjacency map[T][]T
}

// New membuat empty graph.
func New[T comparable]() *Graph[T] {
	return &Graph[T]{adjacency: map[T][]T{}}
}

// AddVertex daftarkan vertex tanpa edge. Idempotent.
// O(1).
func (g *Graph[T]) AddVertex(v T) {
	if _, ok := g.adjacency[v]; !ok {
		g.adjacency[v] = nil
	}
}

// AddEdge tambahkan directed edge from -> to. Auto-register vertex.
// O(1) amortized.
func (g *Graph[T]) AddEdge(from, to T) {
	g.AddVertex(from)
	g.AddVertex(to)
	g.adjacency[from] = append(g.adjacency[from], to)
}

// Neighbors return list vertex yang reachable dari v dengan satu edge.
// O(1).
func (g *Graph[T]) Neighbors(v T) []T {
	return g.adjacency[v]
}

// Vertices return semua vertex (urutan tidak deterministik karena map iteration).
// O(V).
func (g *Graph[T]) Vertices() []T {
	out := make([]T, 0, len(g.adjacency))
	for v := range g.adjacency {
		out = append(out, v)
	}
	return out
}

// BFS traversal dari start. visit return false untuk stop.
// Pakai queue (slice append + slice tail). O(V + E).
func (g *Graph[T]) BFS(start T, visit func(T) bool) {
	visited := map[T]struct{}{start: {}}
	queue := []T{start}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		if !visit(v) {
			return
		}
		for _, next := range g.adjacency[v] {
			if _, seen := visited[next]; seen {
				continue
			}
			visited[next] = struct{}{}
			queue = append(queue, next)
		}
	}
}

// DFS traversal dari start. Pakai stack eksplisit (iterative — bukan recursion)
// untuk hindari stack overflow di graph dalam. O(V + E).
func (g *Graph[T]) DFS(start T, visit func(T) bool) {
	visited := map[T]struct{}{}
	stack := []T{start}
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := visited[v]; seen {
			continue
		}
		visited[v] = struct{}{}
		if !visit(v) {
			return
		}
		for _, next := range g.adjacency[v] {
			if _, seen := visited[next]; !seen {
				stack = append(stack, next)
			}
		}
	}
}

// concept:graph-adjacency-list:end

// concept:dag-cycle-detection:start
// HasCycle deteksi cycle pakai DFS dengan 3-color marking:
//   white = belum di-visit
//   gray  = sedang di-visit (di DFS path saat ini)
//   black = sudah selesai
// Kalau encounter gray node saat DFS → cycle. O(V + E).
func (g *Graph[T]) HasCycle() bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[T]int{}
	for v := range g.adjacency {
		color[v] = white
	}

	var dfs func(T) bool
	dfs = func(v T) bool {
		color[v] = gray
		for _, next := range g.adjacency[v] {
			switch color[next] {
			case gray:
				return true // back edge — cycle ditemukan
			case white:
				if dfs(next) {
					return true
				}
			}
		}
		color[v] = black
		return false
	}

	for v := range g.adjacency {
		if color[v] == white {
			if dfs(v) {
				return true
			}
		}
	}
	return false
}

// IsDAG true kalau graph tidak mengandung cycle.
func (g *Graph[T]) IsDAG() bool { return !g.HasCycle() }

// TopologicalSort return urutan vertex sehingga setiap edge u→v, u sebelum v.
// Pakai Kahn's algorithm (BFS-based). O(V + E).
// Return error kalau graph mengandung cycle.
func (g *Graph[T]) TopologicalSort() ([]T, error) {
	inDegree := map[T]int{}
	for v := range g.adjacency {
		inDegree[v] = 0
	}
	for _, neighbors := range g.adjacency {
		for _, n := range neighbors {
			inDegree[n]++
		}
	}

	// Init queue dengan vertex in-degree 0
	queue := make([]T, 0)
	for v, d := range inDegree {
		if d == 0 {
			queue = append(queue, v)
		}
	}

	result := make([]T, 0, len(g.adjacency))
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		result = append(result, v)
		for _, n := range g.adjacency[v] {
			inDegree[n]--
			if inDegree[n] == 0 {
				queue = append(queue, n)
			}
		}
	}

	if len(result) != len(g.adjacency) {
		return nil, ErrCycle
	}
	return result, nil
}

// concept:dag-cycle-detection:end

// ErrCycle dikembalikan TopologicalSort kalau graph cyclic.
var ErrCycle = newErr("graph: cycle detected, cannot topologically sort")

type graphError string

func (e graphError) Error() string { return string(e) }
func newErr(s string) error        { return graphError(s) }
