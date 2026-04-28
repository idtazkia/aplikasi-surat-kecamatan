// Package tree implementasi generic n-ary tree dengan traversal BFS dan DFS.
//
// Tree = rooted tree dengan node yang punya value dan list children.
// Setiap node punya tepat satu parent (kecuali root yang punya nil parent).
// Tidak ada cycle — itu yang membedakan tree dari graph umum.
//
// Use case di app: forward reference untuk Fase 2 — visualisasi disposisi
// internal di sisi Vue, kalau perlu structured tree (selain SQL recursive CTE).
package tree

// concept:tree-traversal:start
// Node[T] adalah simpul tree dengan value type T.
// Children disimpan sebagai slice untuk allow N-ary tree (bukan binary saja).
// Parent pointer optional — disimpan untuk traversal balik (mis. cari root).
type Node[T any] struct {
	Value    T
	Children []*Node[T]
	Parent   *Node[T]
}

// NewNode membuat node tanpa parent.
func NewNode[T any](v T) *Node[T] {
	return &Node[T]{Value: v}
}

// AddChild append child ke node receiver, set parent pointer.
func (n *Node[T]) AddChild(child *Node[T]) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// IsLeaf true kalau tidak ada children.
func (n *Node[T]) IsLeaf() bool { return len(n.Children) == 0 }

// Depth dari root: 0 untuk root, 1 untuk children root, dst.
// O(depth) — walk Parent pointer.
func (n *Node[T]) Depth() int {
	d := 0
	for cur := n.Parent; cur != nil; cur = cur.Parent {
		d++
	}
	return d
}

// DFS pre-order traversal: visit node, lalu rekursif ke setiap child.
// visit return false untuk early-stop traversal (propagate ke seluruh stack).
// O(n) — visit setiap node sekali.
func DFS[T any](root *Node[T], visit func(*Node[T]) bool) {
	dfsRecursive(root, visit)
}

// dfsRecursive return false untuk signal stop ke caller.
func dfsRecursive[T any](n *Node[T], visit func(*Node[T]) bool) bool {
	if n == nil {
		return true
	}
	if !visit(n) {
		return false
	}
	for _, child := range n.Children {
		if !dfsRecursive(child, visit) {
			return false
		}
	}
	return true
}

// BFS level-order traversal pakai queue.
// O(n + e) di mana e = total children edges (untuk tree, e = n - 1).
func BFS[T any](root *Node[T], visit func(*Node[T]) bool) {
	if root == nil {
		return
	}
	queue := []*Node[T]{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if !visit(node) {
			return
		}
		queue = append(queue, node.Children...)
	}
}

// concept:tree-traversal:end

// Height dari tree (panjang path dari root ke leaf terjauh).
// 0 untuk single-node tree, -1 untuk nil.
// O(n) — visit setiap node sekali.
func Height[T any](root *Node[T]) int {
	if root == nil {
		return -1
	}
	max := 0
	for _, c := range root.Children {
		h := Height(c) + 1
		if h > max {
			max = h
		}
	}
	return max
}

// Size hitung total node di tree. O(n).
func Size[T any](root *Node[T]) int {
	if root == nil {
		return 0
	}
	count := 1
	for _, c := range root.Children {
		count += Size(c)
	}
	return count
}
