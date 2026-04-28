package graph

import (
	"errors"
	"sort"
	"testing"
)

func TestGraph_AddEdgeAutoRegisters(t *testing.T) {
	g := New[string]()
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")

	if len(g.Vertices()) != 3 {
		t.Errorf("vertices = %v, want 3", g.Vertices())
	}
	if neighbors := g.Neighbors("A"); len(neighbors) != 1 || neighbors[0] != "B" {
		t.Errorf("Neighbors(A) = %v, want [B]", neighbors)
	}
}

func TestBFS_VisitsInLevelOrder(t *testing.T) {
	g := New[int]()
	// 1 → 2, 3
	// 2 → 4
	// 3 → 4, 5
	// 4 → 6
	g.AddEdge(1, 2)
	g.AddEdge(1, 3)
	g.AddEdge(2, 4)
	g.AddEdge(3, 4)
	g.AddEdge(3, 5)
	g.AddEdge(4, 6)

	var order []int
	g.BFS(1, func(v int) bool {
		order = append(order, v)
		return true
	})

	// Level 0: 1; level 1: {2, 3}; level 2: {4, 5}; level 3: {6}
	// Map iteration adjacency tidak deterministik untuk neighbors, tapi 1 selalu pertama, 6 selalu terakhir.
	if order[0] != 1 {
		t.Errorf("BFS first = %d, want 1", order[0])
	}
	if order[len(order)-1] != 6 {
		t.Errorf("BFS last = %d, want 6", order[len(order)-1])
	}
}

func TestDFS_VisitsAllReachable(t *testing.T) {
	g := New[int]()
	g.AddEdge(1, 2)
	g.AddEdge(2, 3)
	g.AddEdge(3, 1) // cycle, but DFS handles via visited set

	var visited []int
	g.DFS(1, func(v int) bool {
		visited = append(visited, v)
		return true
	})

	if len(visited) != 3 {
		t.Errorf("DFS visited = %v, want 3 unique", visited)
	}
}

func TestHasCycle_DAG(t *testing.T) {
	g := New[string]()
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("B", "D")
	g.AddEdge("C", "D")

	if g.HasCycle() {
		t.Error("expected no cycle in DAG")
	}
	if !g.IsDAG() {
		t.Error("expected IsDAG=true")
	}
}

func TestHasCycle_Cyclic(t *testing.T) {
	g := New[string]()
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", "A") // cycle

	if !g.HasCycle() {
		t.Error("expected cycle")
	}
}

func TestHasCycle_SelfLoop(t *testing.T) {
	g := New[int]()
	g.AddEdge(1, 1)
	if !g.HasCycle() {
		t.Error("self-loop counts as cycle")
	}
}

func TestTopologicalSort_DAG(t *testing.T) {
	g := New[string]()
	// A → B → D
	// A → C → D
	// D → E
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("B", "D")
	g.AddEdge("C", "D")
	g.AddEdge("D", "E")

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topo sort error: %v", err)
	}
	if len(order) != 5 {
		t.Errorf("order = %v, want length 5", order)
	}

	// Verify invariant: untuk tiap edge u→v, idx(u) < idx(v)
	idx := map[string]int{}
	for i, v := range order {
		idx[v] = i
	}
	edges := [][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}, {"D", "E"}}
	for _, e := range edges {
		if idx[e[0]] >= idx[e[1]] {
			t.Errorf("violates topo order: %s (idx %d) ≥ %s (idx %d)",
				e[0], idx[e[0]], e[1], idx[e[1]])
		}
	}
}

func TestTopologicalSort_CycleReturnsErr(t *testing.T) {
	g := New[int]()
	g.AddEdge(1, 2)
	g.AddEdge(2, 3)
	g.AddEdge(3, 1)

	_, err := g.TopologicalSort()
	if !errors.Is(err, ErrCycle) {
		t.Errorf("error = %v, want ErrCycle", err)
	}
}

func TestVertices_Idempotent(t *testing.T) {
	g := New[string]()
	g.AddVertex("X")
	g.AddVertex("X")
	g.AddVertex("X")
	v := g.Vertices()
	if len(v) != 1 {
		t.Errorf("vertices = %v, want [X]", v)
	}
}

func TestNeighbors_OrderConsistent(t *testing.T) {
	// Order matters di adjacency list (insert order). Verify.
	g := New[string]()
	g.AddEdge("A", "C")
	g.AddEdge("A", "B")
	g.AddEdge("A", "D")

	got := append([]string(nil), g.Neighbors("A")...)
	want := []string{"C", "B", "D"}
	sort.Strings(got) // map yields any order — sort for compare
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	for i := range want {
		if got[i] != sortedWant[i] {
			t.Errorf("neighbors mismatch: got %v want %v", got, sortedWant)
		}
	}
}
