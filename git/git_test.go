package git

import (
	"strings"
	"testing"

	"github.com/alpertosun/versioner/core"
	"github.com/alpertosun/versioner/internal/testrepo"
)

func defaultRules(t *testing.T) []core.CompiledRule {
	t.Helper()
	rules, err := core.DefaultConfig().Compile()
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func TestNearestTagUsesHistory(t *testing.T) {
	r := testrepo.New(t, "release/1.3.0")
	r.Git("tag", "1.3.0-rc9")
	r.Commit("candidate ten")
	r.Git("tag", "1.3.0-rc10")
	r.Commit("next")
	r.Use()
	rules := defaultRules(t)
	for _, snapshot := range []bool{false, true} {
		if snapshot {
			r.Git("tag", "snapshot")
		}
		tags, err := ReadTags(rules)
		if err != nil {
			t.Fatal(err)
		}
		tag, err := NearestTag(tags, false)
		if err != nil || tag == nil || tag.Name != "1.3.0-rc10" || tag.Distance != 1 {
			t.Fatalf("nearest tag: %+v, %v", tag, err)
		}
	}
}

func TestTagsOutsideHistoryAreIgnored(t *testing.T) {
	r := testrepo.New(t, "main")
	r.Git("tag", "1.2.3")
	r.Git("checkout", "-q", "-b", "feature/future")
	r.Commit("future")
	r.Git("tag", "9.0.0")
	r.Git("checkout", "-q", "main")
	r.Commit("main work")
	r.Git("tag", "snapshot")
	r.Use()
	tags, err := ReadTags(defaultRules(t))
	if err != nil || len(tags) != 1 || tags[0].Name != "1.2.3" {
		t.Fatalf("reachable tags: %+v, %v", tags, err)
	}
}

func TestStableTagIsSeparateFromReference(t *testing.T) {
	r := testrepo.New(t, "hotfix/test")
	r.Git("tag", "1.2.3")
	r.Commit("candidate")
	r.Git("tag", "2.0.0-rc1")
	r.Use()
	tags, err := ReadTags(defaultRules(t))
	if err != nil {
		t.Fatal(err)
	}
	reference, err := NearestTag(tags, false)
	if err != nil || reference.Name != "2.0.0-rc1" || reference.Distance != 0 {
		t.Fatalf("reference: %+v, %v", reference, err)
	}
	stable, err := NearestTag(tags, true)
	if err != nil || stable.Name != "1.2.3" || stable.Distance != 1 {
		t.Fatalf("stable base: %+v, %v", stable, err)
	}
}

func TestNearestTagTies(t *testing.T) {
	rules := defaultRules(t)
	a, _, _ := core.ParseTag("1.2.3", rules)
	b, _, _ := core.ParseTag("1.2.4", rules)
	for _, tags := range [][]Tag{
		{{ParsedTag: a, Distance: 1}, {ParsedTag: b, Distance: 1}},
		{{ParsedTag: b, Distance: 1}, {ParsedTag: a, Distance: 1}},
	} {
		if _, err := NearestTag(tags, false); err == nil {
			t.Fatal("different versions at the same distance accepted")
		}
	}
	alias := a
	alias.Name = "alias"
	tags := []Tag{{ParsedTag: alias}, {ParsedTag: a}}
	tag, err := NearestTag(tags, false)
	if err != nil || tag.Name != "1.2.3" {
		t.Fatalf("equivalent tie: %+v, %v", tag, err)
	}
	tags = []Tag{{ParsedTag: a, Distance: 3}, {ParsedTag: b, Distance: 3}, {ParsedTag: a, Distance: 0}}
	tag, err = NearestTag(tags, false)
	if err != nil || tag.Distance != 0 {
		t.Fatalf("distant tie should not hide nearer tag: %+v, %v", tag, err)
	}
}

func TestAnnotatedTagAndRefNameCollision(t *testing.T) {
	r := testrepo.New(t, "main")
	r.Git("tag", "-a", "1.2.3", "-m", "release")
	r.Commit("next")
	r.Git("branch", "1.2.3")
	r.Use()
	distance, err := CommitDistanceSinceTagName("1.2.3")
	if err != nil || distance != 1 {
		t.Fatalf("distance: %d, %v", distance, err)
	}
	r.Git("checkout", "-q", "--detach", "refs/tags/1.2.3")
	tag, err := TagOnHEAD(defaultRules(t))
	if err != nil || tag.Name != "1.2.3" {
		t.Fatalf("annotated tag at HEAD: %+v, %v", tag, err)
	}
}

func TestReleaseRefsIgnoreColor(t *testing.T) {
	r := testrepo.New(t, "develop")
	r.Git("update-ref", "refs/remotes/origin/release/2.5.0", "HEAD")
	r.Git("update-ref", "refs/remotes/origin/release/2.4.0", "HEAD")
	r.Git("update-ref", "refs/remotes/other/release/99.0.0", "HEAD")
	r.Git("config", "color.ui", "always")
	r.Use()
	major, minor, found, err := HighestReleaseAB()
	if err != nil || !found || major != 2 || minor != 5 {
		t.Fatalf("release refs: %d.%d, %v, %v", major, minor, found, err)
	}
	r.Git("update-ref", "refs/remotes/origin/release/2."+strings.Repeat("9", 30)+".0", "HEAD")
	if _, _, _, err := HighestReleaseAB(); err == nil {
		t.Fatal("release ref overflow ignored")
	}
}

func TestGitErrorsPropagate(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := ReadTags(defaultRules(t)); err == nil {
		t.Fatal("tag read error ignored")
	}
	if _, _, _, err := HighestReleaseAB(); err == nil {
		t.Fatal("release read error ignored")
	}
	if _, err := CommitDistanceSinceTagName(""); err == nil {
		t.Fatal("commit count error ignored")
	}
	if _, err := Run(); err == nil {
		t.Fatal("empty command accepted")
	}
}
