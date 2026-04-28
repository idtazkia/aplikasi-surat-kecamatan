package uuid7

import (
	"testing"
	"time"
)

func TestNew_VersionAndVariant(t *testing.T) {
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Version field: high 4 bits of byte 6 must be 0111 (= 7)
	version := u[6] >> 4
	if version != 7 {
		t.Errorf("version = %d, want 7", version)
	}

	// Variant field: high 2 bits of byte 8 must be 10
	variant := u[8] >> 6
	if variant != 0b10 {
		t.Errorf("variant = %b, want 10", variant)
	}
}

func TestNew_TimeOrdering(t *testing.T) {
	a, _ := New()
	time.Sleep(2 * time.Millisecond)
	b, _ := New()

	if a.Time().After(b.Time()) {
		t.Errorf("expected a.Time() <= b.Time(), got a=%v b=%v", a.Time(), b.Time())
	}

	// String comparison harus juga preserve ordering (karena timestamp = prefix big-endian)
	if a.String() >= b.String() {
		t.Errorf("expected a.String() < b.String(), got a=%s b=%s", a, b)
	}
}

func TestStringAndParse_Roundtrip(t *testing.T) {
	u, _ := New()
	s := u.String()
	parsed, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", s, err)
	}
	if parsed != u {
		t.Errorf("roundtrip mismatch: %s != %s", parsed, u)
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := []string{
		"",
		"not-a-uuid",
		"00000000-0000-0000-0000-00000000000",  // 35 chars
		"00000000-0000-0000-0000-0000000000000", // 37 chars
		"00000000_0000_0000_0000_000000000000",  // wrong separator
		"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",  // non-hex
	}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", c)
		}
	}
}

func TestNil_IsZero(t *testing.T) {
	if !Nil.IsZero() {
		t.Error("Nil.IsZero() should be true")
	}
	u, _ := New()
	if u.IsZero() {
		t.Error("New().IsZero() should be false")
	}
}

func TestMarshalText(t *testing.T) {
	u := MustParse("01234567-89ab-7cde-8123-456789abcdef")
	b, err := u.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error: %v", err)
	}
	if string(b) != "01234567-89ab-7cde-8123-456789abcdef" {
		t.Errorf("MarshalText got %q", b)
	}

	var u2 UUID
	if err := u2.UnmarshalText(b); err != nil {
		t.Fatalf("UnmarshalText error: %v", err)
	}
	if u2 != u {
		t.Error("UnmarshalText roundtrip mismatch")
	}
}

func TestNew_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[UUID]struct{}, n)
	for i := 0; i < n; i++ {
		u, _ := New()
		if _, dup := seen[u]; dup {
			t.Fatalf("duplicate UUID at iter %d: %s", i, u)
		}
		seen[u] = struct{}{}
	}
}
