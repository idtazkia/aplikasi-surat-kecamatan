// Package trie implementasi prefix trie untuk autocomplete dan prefix matching.
//
// Use case di app: autocomplete pengirim instansi (Skenario 6 Bagian A) —
// dataset ~500 entries, lookup prefix harus < 50ms. PostgreSQL ILIKE prefix
// dengan B-Tree text index sudah cukup untuk Fase 1, tapi in-memory trie
// menjadi alternatif kalau dataset tumbuh atau butuh fuzzy match.
package trie

// concept:trie-prefix-tree:start
// Trie node: setiap node punya children map[rune]*node + flag isEnd.
// Tidak generic — hanya support string. Implementasi map vs array slot:
// map fleksibel untuk Unicode rune, array 26-slot lebih cepat tapi terbatas
// alphabet ASCII.
type node struct {
	children map[rune]*node
	isEnd    bool
	word     string // disimpan di leaf untuk efficient collect tanpa rebuild path
}

// Trie data structure.
type Trie struct {
	root *node
	size int
}

// New membuat empty trie.
func New() *Trie {
	return &Trie{root: &node{children: map[rune]*node{}}}
}

// Insert kata ke trie. Idempotent — duplicate insert tidak menambah size.
// O(k) di mana k = panjang kata.
func (t *Trie) Insert(word string) {
	cur := t.root
	for _, c := range word {
		next, ok := cur.children[c]
		if !ok {
			next = &node{children: map[rune]*node{}}
			cur.children[c] = next
		}
		cur = next
	}
	if !cur.isEnd {
		cur.isEnd = true
		cur.word = word
		t.size++
	}
}

// Contains check exact match. O(k).
func (t *Trie) Contains(word string) bool {
	cur := t.root
	for _, c := range word {
		next, ok := cur.children[c]
		if !ok {
			return false
		}
		cur = next
	}
	return cur.isEnd
}

// SearchPrefix return semua kata yang dimulai dengan prefix.
// O(k + p) di mana k = panjang prefix, p = total karakter di hasil.
// Hasil tidak deterministik order karena map iteration.
func (t *Trie) SearchPrefix(prefix string) []string {
	cur := t.root
	for _, c := range prefix {
		next, ok := cur.children[c]
		if !ok {
			return nil
		}
		cur = next
	}
	var out []string
	collect(cur, &out)
	return out
}

// collect DFS dari node, append semua kata isEnd ke acc.
func collect(n *node, acc *[]string) {
	if n.isEnd {
		*acc = append(*acc, n.word)
	}
	for _, child := range n.children {
		collect(child, acc)
	}
}

// concept:trie-prefix-tree:end

// Size jumlah unique kata di trie.
func (t *Trie) Size() int { return t.size }

// Delete remove kata kalau ada. Return true kalau ter-delete.
// Cleanup empty branches untuk memory hygiene.
// O(k).
func (t *Trie) Delete(word string) bool {
	sizeBefore := t.size
	deleteHelper(t.root, word, 0, t)
	return t.size < sizeBefore
}

func deleteHelper(n *node, word string, depth int, t *Trie) bool {
	if depth == len(word) {
		if !n.isEnd {
			return false
		}
		n.isEnd = false
		n.word = ""
		t.size--
		return len(n.children) == 0
	}
	r := []rune(word)[depth]
	child, ok := n.children[r]
	if !ok {
		return false
	}
	shouldRemove := deleteHelper(child, word, depth+1, t)
	if shouldRemove {
		delete(n.children, r)
		return !n.isEnd && len(n.children) == 0
	}
	return false
}
