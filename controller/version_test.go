package controller

import (
	"testing"
)

func TestVersionCompare(t *testing.T) {
	cases := []struct{ a, b string; want int }{
		{"5.1.0", "5.1.0", 0},
		{"5", "5.0.1", -1},
		{"6", "5.20", 1},
		{"5.9", "5.10", -1},
		{"6.2.0.0", "6.3", -1},
		{"5.15.8.12", "5.15.8.9", 1},
		{"6", "6.0.0.0", 0},
		{"5.1.0-beta", "5.1.0", 0},
	}
	for _, c := range cases {
		a, errA := ParseVersion(c.a)
		b, errB := ParseVersion(c.b)
		if errA != nil || errB != nil {
			t.Fatalf("parse %q / %q: %v, %v", c.a, c.b, errA, errB)
		}
		if got := a.Compare(b); got != c.want {
			t.Errorf("%s vs %s = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseVersionError(t *testing.T) {
	if _, err := ParseVersion(""); err == nil {
		t.Error("empty input should fail")
	}
	if _, err := ParseVersion("x"); err == nil {
		t.Error("non-numeric input should fail")
	}
}

func TestVersionString(t *testing.T) {
	v, err := ParseVersion("6.2.10.18")
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "6.2.10.18" {
		t.Errorf("got %q", v.String())
	}
}
