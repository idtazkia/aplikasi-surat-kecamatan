package tree

import "testing"

// Build sample tree:
//
//        A
//      / | \
//     B  C  D
//    /|     |
//   E F     G
func buildSample() *Node[string] {
	a := NewNode("A")
	b := NewNode("B")
	c := NewNode("C")
	d := NewNode("D")
	e := NewNode("E")
	f := NewNode("F")
	g := NewNode("G")
	a.AddChild(b)
	a.AddChild(c)
	a.AddChild(d)
	b.AddChild(e)
	b.AddChild(f)
	d.AddChild(g)
	return a
}

func TestSize(t *testing.T) {
	if got := Size[string](nil); got != 0 {
		t.Errorf("Size(nil) = %d, want 0", got)
	}
	if got := Size(NewNode("solo")); got != 1 {
		t.Errorf("Size(single) = %d, want 1", got)
	}
	if got := Size(buildSample()); got != 7 {
		t.Errorf("Size(sample) = %d, want 7", got)
	}
}

func TestHeight(t *testing.T) {
	if got := Height[string](nil); got != -1 {
		t.Errorf("Height(nil) = %d, want -1", got)
	}
	if got := Height(NewNode("solo")); got != 0 {
		t.Errorf("Height(single) = %d, want 0", got)
	}
	if got := Height(buildSample()); got != 2 {
		t.Errorf("Height(sample) = %d, want 2", got)
	}
}

func TestDepth(t *testing.T) {
	root := buildSample()
	if root.Depth() != 0 {
		t.Errorf("root depth = %d, want 0", root.Depth())
	}
	// Cari E lewat traversal
	var e *Node[string]
	DFS(root, func(n *Node[string]) bool {
		if n.Value == "E" {
			e = n
			return false
		}
		return true
	})
	if e == nil || e.Depth() != 2 {
		t.Errorf("E.Depth() = %d, want 2", e.Depth())
	}
}

func TestDFS_PreOrder(t *testing.T) {
	root := buildSample()
	var visited []string
	DFS(root, func(n *Node[string]) bool {
		visited = append(visited, n.Value)
		return true
	})
	want := []string{"A", "B", "E", "F", "C", "D", "G"}
	if len(visited) != len(want) {
		t.Fatalf("visited = %v, want %v", visited, want)
	}
	for i := range want {
		if visited[i] != want[i] {
			t.Errorf("visited[%d] = %s, want %s", i, visited[i], want[i])
		}
	}
}

func TestBFS_LevelOrder(t *testing.T) {
	root := buildSample()
	var visited []string
	BFS(root, func(n *Node[string]) bool {
		visited = append(visited, n.Value)
		return true
	})
	want := []string{"A", "B", "C", "D", "E", "F", "G"}
	for i := range want {
		if visited[i] != want[i] {
			t.Errorf("BFS[%d] = %s, want %s", i, visited[i], want[i])
		}
	}
}

func TestDFS_EarlyStop(t *testing.T) {
	root := buildSample()
	var visited []string
	DFS(root, func(n *Node[string]) bool {
		visited = append(visited, n.Value)
		return n.Value != "B" // stop saat ketemu B
	})
	// Setelah B, traversal stop. C, D, E, F, G tidak di-visit.
	if len(visited) != 2 || visited[0] != "A" || visited[1] != "B" {
		t.Errorf("early-stop visited = %v, want [A, B]", visited)
	}
}

func TestIsLeaf(t *testing.T) {
	root := buildSample()
	if root.IsLeaf() {
		t.Error("root not leaf")
	}
	// Cari G (leaf)
	var g *Node[string]
	DFS(root, func(n *Node[string]) bool {
		if n.Value == "G" {
			g = n
			return false
		}
		return true
	})
	if !g.IsLeaf() {
		t.Error("G should be leaf")
	}
}
