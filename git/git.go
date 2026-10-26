package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/alpertosun/versioner/core"
)

func Run(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("git command is required")
	}
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", err
		}
		return "", fmt.Errorf("git %s: %s: %w", args[0], msg, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func CurrentBranch() (string, error) { return Run("rev-parse", "--abbrev-ref", "HEAD") }
func ShortSHA() (string, error)      { return Run("rev-parse", "--short", "HEAD") }

type Tag struct {
	core.ParsedTag
	Distance int
}

func ReadTags(rules []core.CompiledRule) ([]Tag, error) {
	return readTags(rules, "--merged=HEAD")
}

func TagOnHEAD(rules []core.CompiledRule) (core.ParsedTag, error) {
	tags, err := readTags(rules, "--points-at=HEAD")
	if err != nil {
		return core.ParsedTag{}, err
	}
	if len(tags) != 1 {
		return core.ParsedTag{}, fmt.Errorf("detached HEAD requires exactly one config-matching tag (found %d); set VERSIONER_BRANCH to select a branch", len(tags))
	}
	return tags[0].ParsedTag, nil
}

func readTags(rules []core.CompiledRule, filter string) ([]Tag, error) {
	rawTags, err := Run("for-each-ref", filter, "--sort=refname", "--format=%(refname:strip=2)", "refs/tags/")
	if err != nil {
		return nil, err
	}
	var tags []Tag
	for _, t := range strings.Split(rawTags, "\n") {
		if t == "" {
			continue
		}
		parsed, ok, err := core.ParseTag(t, rules)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		distance, err := CommitDistanceSinceTagName(t)
		if err != nil {
			return nil, err
		}
		tags = append(tags, Tag{ParsedTag: parsed, Distance: distance})
	}
	return tags, nil
}

func NearestTag(tags []Tag, stableOnly bool) (*Tag, error) {
	var best *Tag
	for i := range tags {
		tag := &tags[i]
		if stableOnly && !tag.Stable() {
			continue
		}
		if best == nil || tag.Distance < best.Distance {
			best = tag
		}
	}
	if best == nil {
		return nil, nil
	}
	for i := range tags {
		tag := &tags[i]
		if (stableOnly && !tag.Stable()) || tag.Distance != best.Distance {
			continue
		}
		if !best.SameMeaning(tag.ParsedTag) {
			return nil, fmt.Errorf("ambiguous nearest tags %q and %q at distance %d", best.Name, tag.Name, best.Distance)
		}
		if tag.Name < best.Name {
			best = tag
		}
	}
	return best, nil
}

func LatestTagNameAndVersion() (string, core.Version, error) {
	rules, err := core.DefaultConfig().Compile()
	if err != nil {
		return "", core.Version{}, err
	}
	tags, err := ReadTags(rules)
	if err != nil {
		return "", core.Version{}, err
	}
	tag, err := NearestTag(tags, false)
	if err != nil || tag == nil {
		return "", core.Version{}, err
	}
	return tag.Name, tag.Version, nil
}

func CommitDistanceSinceTagName(tagName string) (int, error) {
	if strings.TrimSpace(tagName) == "" {
		out, err := Run("rev-list", "HEAD", "--count")
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(out)
	}
	out, err := Run("rev-list", "refs/tags/"+tagName+"..HEAD", "--count", "--")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

func HighestReleaseAB() (a int, b int, found bool, err error) {
	out, err := Run("for-each-ref", "--format=%(refname:strip=3)", "refs/remotes/origin/release/")
	if err != nil {
		return 0, 0, false, err
	}
	re := regexp.MustCompile(`^release/(\d+)\.(\d+)\.0$`)
	maxA, maxB := -1, -1
	for _, br := range strings.Split(out, "\n") {
		m := re.FindStringSubmatch(br)
		if len(m) != 3 {
			continue
		}
		maj, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0, false, fmt.Errorf("invalid release major in %q: %w", br, err)
		}
		min, err := strconv.Atoi(m[2])
		if err != nil {
			return 0, 0, false, fmt.Errorf("invalid release minor in %q: %w", br, err)
		}
		if maj > maxA || (maj == maxA && min > maxB) {
			maxA, maxB = maj, min
		}
	}
	if maxA >= 0 {
		return maxA, maxB, true, nil
	}
	return 0, 0, false, nil
}
