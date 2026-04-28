package trie

import (
	"sort"
	"testing"
)

func TestTrie_InsertContains(t *testing.T) {
	tr := New()
	tr.Insert("Kemendagri")
	tr.Insert("Kemensos")
	tr.Insert("Kemenkes")

	if !tr.Contains("Kemendagri") {
		t.Error("Contains(Kemendagri) = false, want true")
	}
	if tr.Contains("Kemen") {
		t.Error("Contains(Kemen) = true, want false (prefix bukan word lengkap)")
	}
	if tr.Contains("Kemenkominfo") {
		t.Error("Contains(Kemenkominfo) = true, want false")
	}
}

func TestTrie_SearchPrefix(t *testing.T) {
	tr := New()
	for _, w := range []string{"Kemendagri", "Kemensos", "Kemenkes", "Kemenag", "Polri"} {
		tr.Insert(w)
	}

	results := tr.SearchPrefix("Kemen")
	sort.Strings(results)
	want := []string{"Kemenag", "Kemendagri", "Kemenkes", "Kemensos"}
	if len(results) != len(want) {
		t.Fatalf("prefix search results = %v, want %v", results, want)
	}
	for i := range want {
		if results[i] != want[i] {
			t.Errorf("results[%d] = %s, want %s", i, results[i], want[i])
		}
	}

	// Prefix yang tidak match: empty
	if r := tr.SearchPrefix("xyz"); len(r) != 0 {
		t.Errorf("non-match prefix = %v, want empty", r)
	}

	// More specific prefix
	results = tr.SearchPrefix("Kemenk")
	sort.Strings(results)
	if len(results) != 1 || results[0] != "Kemenkes" {
		t.Errorf("Kemenk results = %v, want [Kemenkes]", results)
	}
}

func TestTrie_InsertDuplicateNoDoubleCount(t *testing.T) {
	tr := New()
	tr.Insert("foo")
	tr.Insert("foo")
	if tr.Size() != 1 {
		t.Errorf("size after dup insert = %d, want 1", tr.Size())
	}
}

func TestTrie_Delete(t *testing.T) {
	tr := New()
	tr.Insert("apple")
	tr.Insert("application")

	if !tr.Delete("apple") {
		t.Error("delete returned false")
	}
	if tr.Contains("apple") {
		t.Error("Contains(apple) after delete = true")
	}
	// Application masih ada — apple share prefix tapi delete tidak boleh hapus jalur application
	if !tr.Contains("application") {
		t.Error("application terhapus padahal hanya apple yang dihapus")
	}

	if tr.Delete("nonexistent") {
		t.Error("delete non-existent returned true")
	}
}

func TestTrie_EmptyString(t *testing.T) {
	tr := New()
	tr.Insert("")
	if !tr.Contains("") {
		t.Error("Contains('') = false after insert empty")
	}
	if tr.Size() != 1 {
		t.Errorf("size = %d, want 1", tr.Size())
	}
}

func TestTrie_UnicodeChars(t *testing.T) {
	tr := New()
	tr.Insert("café")
	tr.Insert("naïve")
	if !tr.Contains("café") {
		t.Error("unicode contains failed")
	}
	results := tr.SearchPrefix("café")
	if len(results) != 1 {
		t.Errorf("unicode prefix results = %v", results)
	}
}
