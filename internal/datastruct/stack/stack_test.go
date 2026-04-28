package stack

import (
	"errors"
	"testing"
)

func TestStack_PushPopPeek(t *testing.T) {
	s := New[int]()

	if !s.IsEmpty() {
		t.Error("new stack should be empty")
	}

	s.Push(1)
	s.Push(2)
	s.Push(3)

	if s.Len() != 3 {
		t.Errorf("len = %d, want 3", s.Len())
	}

	top, err := s.Peek()
	if err != nil || top != 3 {
		t.Errorf("peek = %d %v, want 3 nil", top, err)
	}

	v, _ := s.Pop()
	if v != 3 {
		t.Errorf("pop = %d, want 3 (LIFO)", v)
	}
	v, _ = s.Pop()
	if v != 2 {
		t.Errorf("pop = %d, want 2", v)
	}
	v, _ = s.Pop()
	if v != 1 {
		t.Errorf("pop = %d, want 1", v)
	}
}

func TestStack_PopEmptyReturnsErr(t *testing.T) {
	s := New[string]()
	_, err := s.Pop()
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("pop empty = %v, want ErrEmpty", err)
	}
	_, err = s.Peek()
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("peek empty = %v, want ErrEmpty", err)
	}
}

func TestStack_GenericTypes(t *testing.T) {
	type point struct{ x, y int }
	s := New[point]()
	s.Push(point{1, 2})
	s.Push(point{3, 4})
	v, _ := s.Pop()
	if v != (point{3, 4}) {
		t.Errorf("got %v", v)
	}
}
