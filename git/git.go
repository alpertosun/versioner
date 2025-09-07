package git

import (
	"bytes"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/alpertosun/versioner/core"
)

func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func CurrentBranch() (string, error) { return Run("rev-parse", "--abbrev-ref", "HEAD") }
func ShortSHA() (string, error)      { return Run("rev-parse", "--short", "HEAD") }

// LatestTagNameAndVersion tries git describe first (closest tag), then falls back to scanning tags.
func LatestTagNameAndVersion() (string, core.Version, error) {
	if name, err := Run("describe", "--tags", "--abbrev=0"); err == nil && name != "" {
		if v, err2 := core.ParseVersion(name); err2 == nil {
			return name, v, nil
		}
	}
	// Fallback: list tags and pick highest by SemVer compare
	rawTags, err := Run("tag", "--list")
	if err != nil {
		return "", core.Version{}, err
	}
	var pairs []struct {
		name string
		ver  core.Version
	}
	for _, t := range strings.Split(rawTags, "\n") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if v, err := core.ParseVersion(t); err == nil {
			pairs = append(pairs, struct {
				name string
				ver  core.Version
			}{t, v})
		}
	}
	if len(pairs) == 0 {
		return "", core.Version{}, nil
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].ver.Compare(pairs[j].ver) > 0 })
	return pairs[0].name, pairs[0].ver, nil
}

func CommitDistanceSinceTagName(tagName string) (int, error) {
	if strings.TrimSpace(tagName) == "" {
		out, err := Run("rev-list", "HEAD", "--count")
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(out)
	}
	out, err := Run("rev-list", tagName+"..HEAD", "--count")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

func HighestReleaseAB() (a int, b int, found bool, err error) {
	var lines []string
	if out, e := Run("branch", "-r", "--list", "origin/release/*"); e == nil && out != "" {
		for _, ln := range strings.Split(out, "\n") {
			ln = strings.TrimSpace(ln)
			if strings.HasPrefix(ln, "origin/") {
				ln = strings.TrimPrefix(ln, "origin/")
			}
			if ln != "" {
				lines = append(lines, ln)
			}
		}
	}
	re := regexp.MustCompile(`^release/(\d+)\.(\d+)\.0$`)
	maxA, maxB := -1, -1
	for _, br := range lines {
		m := re.FindStringSubmatch(br)
		if len(m) != 3 {
			continue
		}
		maj, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		if maj > maxA || (maj == maxA && min > maxB) {
			maxA, maxB = maj, min
		}
	}
	if maxA >= 0 {
		return maxA, maxB, true, nil
	}
	return 0, 0, false, nil
}
