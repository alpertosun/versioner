package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const numberPattern = `(?:0|[1-9]\d*)`

var (
	semverRe = regexp.MustCompile(`^(?:v)?` + // optional v
		`(?P<maj>` + numberPattern + `)\.(?P<min>` + numberPattern + `)\.(?P<pat>` + numberPattern + `)` +
		`(?:-(?P<pre>[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?` +
		`(?:\+(?P<build>[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string // e.g. rc.1
	BuildMeta  string // e.g. gabc123
}

func (v Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		base += "-" + v.PreRelease
	}
	if v.BuildMeta != "" {
		base += "+" + v.BuildMeta
	}
	return base
}

func ParseVersion(tag string) (Version, error) {
	m := semverRe.FindStringSubmatch(tag)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version: %s", tag)
	}
	idx := func(name string) int { return semverRe.SubexpIndex(name) }
	maj, err := strconv.Atoi(m[idx("maj")])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %w", err)
	}
	min, err := strconv.Atoi(m[idx("min")])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %w", err)
	}
	pat, err := strconv.Atoi(m[idx("pat")])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %w", err)
	}
	pre := m[idx("pre")]
	build := m[idx("build")]
	return Version{Major: maj, Minor: min, Patch: pat, PreRelease: pre, BuildMeta: build}, nil
}

// Compare implements SemVer ordering.
// Returns <0 if v<o, >0 if v>o, 0 if equal (ignoring build meta).
func (v Version) Compare(o Version) int {
	// core
	if v.Major != o.Major {
		return compareInt(v.Major, o.Major)
	}
	if v.Minor != o.Minor {
		return compareInt(v.Minor, o.Minor)
	}
	if v.Patch != o.Patch {
		return compareInt(v.Patch, o.Patch)
	}
	if v.PreRelease == "" && o.PreRelease == "" {
		return 0
	}
	if v.PreRelease == "" {
		return 1
	}
	if o.PreRelease == "" {
		return -1
	}
	va := strings.Split(v.PreRelease, ".")
	ob := strings.Split(o.PreRelease, ".")
	n := len(va)
	if len(ob) < n {
		n = len(ob)
	}
	for i := 0; i < n; i++ {
		c := comparePreID(va[i], ob[i])
		if c != 0 {
			return c
		}
	}
	return len(va) - len(ob)
}

func comparePreID(a, b string) int {
	aNum := isNumeric(a)
	bNum := isNumeric(b)
	if aNum && bNum {
		a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
		if len(a) != len(b) {
			return compareInt(len(a), len(b))
		}
		return strings.Compare(a, b)
	}
	if aNum && !bNum {
		return -1 // numerics < non-numerics
	}
	if !aNum && bNum {
		return 1
	}
	return strings.Compare(a, b)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func increment(n int) (int, error) {
	if n < 0 || n == int(^uint(0)>>1) {
		return 0, fmt.Errorf("version component cannot be incremented: %d", n)
	}
	return n + 1, nil
}
