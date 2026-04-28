package priorityq

import (
	"errors"
	"math/rand"
	"sort"
	"testing"
)

func intCompare(a, b int) int { return a - b }

func TestHeap_PushPopMinOrder(t *testing.T) {
	h := New(intCompare)
	for _, v := range []int{5, 3, 8, 1, 9, 2, 7} {
		h.Push(v)
	}

	var popped []int
	for h.Len() > 0 {
		v, _ := h.Pop()
		popped = append(popped, v)
	}

	want := []int{1, 2, 3, 5, 7, 8, 9}
	for i := range want {
		if popped[i] != want[i] {
			t.Errorf("pop[%d] = %d, want %d", i, popped[i], want[i])
		}
	}
}

func TestHeap_PeekDoesNotRemove(t *testing.T) {
	h := New(intCompare)
	h.Push(10)
	h.Push(5)
	h.Push(15)

	v, _ := h.Peek()
	if v != 5 {
		t.Errorf("peek = %d, want 5", v)
	}
	if h.Len() != 3 {
		t.Errorf("len after peek = %d, want 3", h.Len())
	}
}

func TestHeap_PopEmptyReturnsErr(t *testing.T) {
	h := New(intCompare)
	_, err := h.Pop()
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("pop empty = %v, want ErrEmpty", err)
	}
	_, err = h.Peek()
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("peek empty = %v, want ErrEmpty", err)
	}
}

func TestHeap_StressTest1000Items(t *testing.T) {
	h := New(intCompare)
	src := rand.New(rand.NewSource(42))
	expected := make([]int, 1000)
	for i := range expected {
		expected[i] = src.Intn(10000)
		h.Push(expected[i])
	}
	sort.Ints(expected)

	for i := range expected {
		v, _ := h.Pop()
		if v != expected[i] {
			t.Fatalf("at idx %d: got %d, want %d", i, v, expected[i])
		}
	}
}

func TestHeap_MaxHeapViaInverseCompare(t *testing.T) {
	h := New(func(a, b int) int { return b - a })
	for _, v := range []int{3, 1, 5, 2, 4} {
		h.Push(v)
	}
	v, _ := h.Pop()
	if v != 5 {
		t.Errorf("max-heap top = %d, want 5", v)
	}
}

type task struct {
	priority int
	desc     string
}

func TestHeap_StructValueByPriority(t *testing.T) {
	h := New(func(a, b task) int { return a.priority - b.priority })
	h.Push(task{priority: 5, desc: "biasa"})
	h.Push(task{priority: 1, desc: "segera"})
	h.Push(task{priority: 3, desc: "penting"})

	v, _ := h.Pop()
	if v.desc != "segera" {
		t.Errorf("highest priority = %s, want segera", v.desc)
	}
}
