package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	semverRe = regexp.MustCompile(`^(?:v)?` + // optional v
		`(?P<maj>0|[1-9]\d*)\.(?P<min>0|[1-9]\d*)\.(?P<pat>0|[1-9]\d*)` +
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
	maj, _ := strconv.Atoi(m[idx("maj")])
	min, _ := strconv.Atoi(m[idx("min")])
	pat, _ := strconv.Atoi(m[idx("pat")])
	pre := m[idx("pre")]
	build := m[idx("build")]
	return Version{Major: maj, Minor: min, Patch: pat, PreRelease: pre, BuildMeta: build}, nil
}

// Compare implements SemVer ordering.
// Returns <0 if v<o, >0 if v>o, 0 if equal (ignoring build meta).
func (v Version) Compare(o Version) int {
	// core
	if v.Major != o.Major {
		return v.Major - o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor - o.Minor
	}
	if v.Patch != o.Patch {
		return v.Patch - o.Patch
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
	ai, aNum := toInt(a)
	bi, bNum := toInt(b)
	if aNum && bNum {
		if ai != bi {
			return ai - bi
		}
		return 0
	}
	if aNum && !bNum {
		return -1 // numerics < non-numerics
	}
	if !aNum && bNum {
		return 1
	}
	return strings.Compare(a, b)
}

func toInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	v, _ := strconv.Atoi(s)
	return v, true
}
