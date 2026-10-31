package models

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a dotted Omada software version, compared numerically per segment.
type Version struct {
	segments []int
}

// ParseVersion parses a dotted version string like "6.2.10.18".
// Each segment contributes its leading run of digits; a segment with no
// leading digits contributes 0. Empty input is an error.
func ParseVersion(s string) (Version, error) {
	if s == "" {
		return Version{}, fmt.Errorf("empty version string")
	}
	parts := strings.Split(s, ".")
	segments := make([]int, len(parts))
	for i, part := range parts {
		n := 0
		digits := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
			digits++
		}
		if i == 0 && digits == 0 {
			return Version{}, fmt.Errorf("invalid version %q", s)
		}
		segments[i] = n
	}
	return Version{segments: segments}, nil
}

// ParseVersionMust is ParseVersion for tests and constants; it panics on bad input.
func ParseVersionMust(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare returns -1, 0 or 1 when v is less than, equal to or greater than o.
// Missing segments are treated as 0.
func (v Version) Compare(o Version) int {
	n := len(v.segments)
	if len(o.segments) > n {
		n = len(o.segments)
	}
	for i := 0; i < n; i++ {
		a, b := 0, 0
		if i < len(v.segments) {
			a = v.segments[i]
		}
		if i < len(o.segments) {
			b = o.segments[i]
		}
		if a != b {
			if a < b {
				return -1
			}
			return 1
		}
	}
	return 0
}

// String renders the version in dotted form.
func (v Version) String() string {
	parts := make([]string, len(v.segments))
	for i, s := range v.segments {
		parts[i] = strconv.Itoa(s)
	}
	return strings.Join(parts, ".")
}
