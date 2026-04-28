package hashset

import (
	"sort"
	"testing"
)

func TestSet_AddContainsRemove(t *testing.T) {
	s := New[string]()

	if s.Add("a") != true {
		t.Error("first Add should return true")
	}
	if s.Add("a") != false {
		t.Error("dup Add should return false")
	}

	if !s.Contains("a") {
		t.Error("Contains(a) = false")
	}
	if s.Contains("b") {
		t.Error("Contains(b) should be false")
	}

	if !s.Remove("a") {
		t.Error("Remove existing returned false")
	}
	if s.Remove("a") {
		t.Error("Remove already-removed returned true")
	}
	if s.Len() != 0 {
		t.Errorf("len after remove = %d, want 0", s.Len())
	}
}

func TestSet_FromSlice_Dedup(t *testing.T) {
	s := FromSlice([]int{1, 2, 2, 3, 3, 3, 4})
	if s.Len() != 4 {
		t.Errorf("len = %d, want 4 (dedup)", s.Len())
	}
}

func TestSet_ToSlice(t *testing.T) {
	s := FromSlice([]int{1, 2, 3})
	out := s.ToSlice()
	sort.Ints(out)
	want := []int{1, 2, 3}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("ToSlice[%d] = %d, want %d", i, out[i], want[i])
		}
	}
}

func TestSet_Union(t *testing.T) {
	a := FromSlice([]string{"x", "y"})
	b := FromSlice([]string{"y", "z"})
	u := a.Union(b)
	if u.Len() != 3 {
		t.Errorf("union len = %d, want 3", u.Len())
	}
	if !u.Contains("x") || !u.Contains("y") || !u.Contains("z") {
		t.Error("union missing elements")
	}
}

func TestSet_Intersect(t *testing.T) {
	a := FromSlice([]int{1, 2, 3, 4})
	b := FromSlice([]int{3, 4, 5, 6})
	i := a.Intersect(b)
	if i.Len() != 2 {
		t.Errorf("intersect len = %d, want 2", i.Len())
	}
	if !i.Contains(3) || !i.Contains(4) {
		t.Error("intersect missing elements")
	}
}

func TestSet_Difference(t *testing.T) {
	a := FromSlice([]int{1, 2, 3, 4})
	b := FromSlice([]int{3, 4})
	d := a.Difference(b)
	if d.Len() != 2 {
		t.Errorf("diff len = %d, want 2", d.Len())
	}
	if !d.Contains(1) || !d.Contains(2) {
		t.Error("diff should contain 1, 2")
	}
}

func TestSet_DifferentTypes(t *testing.T) {
	type pos struct{ x, y int }
	s := New[pos]()
	s.Add(pos{1, 2})
	s.Add(pos{3, 4})
	if !s.Contains(pos{1, 2}) {
		t.Error("struct contains failed")
	}
	if s.Contains(pos{1, 3}) {
		t.Error("similar but different struct should not match")
	}
}
