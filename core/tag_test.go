package core

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestDefaultVersionsRoundTrip(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"main", "1.2.4"}, {"master", "1.2.4"},
		{"release/1.3.0", "1.3.0-rc10"}, {"develop", "1.3.0-beta10"},
		{"feature/test", "1.3.0-alpha10"}, {"hotfix/test", "1.2.4"},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			rule, rules, err := DefaultConfig().ResolveRule(tt.branch)
			if err != nil {
				t.Fatal(err)
			}
			strategy, _ := StrategyByName(rule.Strategy)
			ctx := GitContext{Branch: tt.branch, CommitDistance: 10, ShortSHA: "abc1234"}
			v, err := strategy.NextVersion(Version{Major: 1, Minor: 2, Patch: 3}, ctx)
			if err != nil {
				t.Fatal(err)
			}
			out := RenderVersion(v, ctx, rule, Version{})
			if out != tt.want {
				t.Fatalf("got %q, want %q", out, tt.want)
			}
			tag, ok, err := ParseTag(out, rules)
			if err != nil || !ok || tag.Version.String() != out {
				t.Fatalf("round trip: %+v, %v, %v", tag, ok, err)
			}
		})
	}
}

func TestCustomTagRoundTrip(t *testing.T) {
	templates := []struct {
		name   string
		prefix *string
		suffix *string
	}{
		{"empty", str(""), str("")},
		{"omitted", nil, nil},
		{"sha", str("v"), str("+{sha}")},
		{"dash sha", str("app-"), str("-{sha}")},
		{"counter", str("build-{N}-"), str("")},
		{"version twice", str("{version}_"), str("")},
		{"core fields", str("{major}{minor}{patch}_"), str("-{pre}-{buildMeta}")},
		{"adjacent major", str("{major}"), str("")},
		{"context", str("{branch}-{slug}-"), str("-from-{tag}")},
		{"unknown", str("{unknown}"), str("")},
	}
	for _, strategyName := range []string{"master", "release", "develop", "feature", "hotfix"} {
		for _, template := range templates {
			t.Run(strategyName+"/"+template.name, func(t *testing.T) {
				rules := compileRules(t, Rule{Pattern: ".*", Strategy: strategyName, Prefix: template.prefix, Suffix: template.suffix})
				ctx := GitContext{Branch: "release/11.2.0", CommitDistance: 10, ShortSHA: "abc1234", TagName: "old-app-1.0.0"}
				strategy, _ := StrategyByName(strategyName)
				v, err := strategy.NextVersion(Version{Major: 11, Minor: 2, Patch: 3}, ctx)
				if err != nil {
					t.Fatal(err)
				}
				out := RenderVersion(v, ctx, &rules[0], Version{})
				tag, ok, err := ParseTag(out, rules)
				if err != nil || !ok {
					t.Fatalf("cannot read %q: %v, %v", out, ok, err)
				}
				if tag.Version.Major != v.Major || tag.Version.Minor != v.Minor || tag.Version.Patch != v.Patch || tag.Version.PreRelease != v.PreRelease {
					t.Fatalf("version changed: %+v -> %+v", v, tag)
				}
				if tag.Name != out || tag.Stable() != (v.PreRelease == "") {
					t.Fatalf("incorrect tag meaning: %+v", tag)
				}
			})
		}
	}
}

func TestTagMustMatchConfig(t *testing.T) {
	rules, _ := DefaultConfig().Compile()
	for _, name := range []string{
		"snapshot", "v1.2.3", "1.2.3+abc1234", "1.3.0-rc10+abc1234",
		"1.3.0-random", "1.3.1-rc10", "1.999999999999999999999999.0",
	} {
		if tag, ok, err := ParseTag(name, rules); err != nil || ok {
			t.Fatalf("unexpected match for %q: %+v, %v", name, tag, err)
		}
	}
	rules = compileRules(t, Rule{Pattern: ".*", Strategy: "master", Prefix: str("app-"), Suffix: str("-{sha}")})
	tag, ok, err := ParseTag("app-1.2.3-0123456", rules)
	if err != nil || !ok || tag.Fields["sha"] != "0123456" || !tag.Stable() {
		t.Fatalf("SHA suffix: %+v, %v, %v", tag, ok, err)
	}
	if _, ok, _ := ParseTag("1.2.3-0123456", rules); ok {
		t.Fatal("tag without configured prefix accepted")
	}
}

func TestAmbiguousTag(t *testing.T) {
	tests := []struct {
		name  string
		rules []Rule
	}{
		{"111.2.3", []Rule{{Pattern: ".*", Strategy: "master", Prefix: str("{N}"), Suffix: str("")}}},
		{"1.2.3.4", []Rule{
			{Pattern: ".*", Strategy: "master", Prefix: str("1."), Suffix: str("")},
			{Pattern: ".*", Strategy: "master", Suffix: str(".4")},
		}},
	}
	for _, tt := range tests {
		if _, _, err := ParseTag(tt.name, compileRules(t, tt.rules...)); err == nil {
			t.Fatalf("ambiguous tag accepted: %s", tt.name)
		}
	}
}

func TestTemplateValuesAreNotExpanded(t *testing.T) {
	ctx := GitContext{Branch: "feature/{sha}", ShortSHA: "abc1234", CommitDistance: 10}
	rules := compileRules(t, Rule{Pattern: "^feature/.*$", Strategy: "feature", Suffix: str("+{branch}")})
	v, _ := (FeatureStrategy{}).NextVersion(Version{Major: 1}, ctx)
	for i := 0; i < 50; i++ {
		out := RenderVersion(v, ctx, &rules[0], Version{})
		if out != "1.1.0-alpha10+feature/{sha}" {
			t.Fatalf("expanded replacement value: %q", out)
		}
		tag, ok, err := ParseTag(out, rules)
		if err != nil || !ok || tag.Fields["branch"] != ctx.Branch {
			t.Fatalf("round trip: %+v, %v", tag, err)
		}
	}
}

func TestRepeatedFieldsMustAgree(t *testing.T) {
	rules := compileRules(t, Rule{Pattern: ".*", Strategy: "release", Prefix: str("{N}-"), Suffix: str("+{sha}-{buildMeta}")})
	for _, name := range []string{"9-1.2.0-rc10+abc1234-gabc1234", "10-1.2.0-rc10+abc1234-gdef5678"} {
		if _, ok, _ := ParseTag(name, rules); ok {
			t.Fatalf("conflicting fields accepted: %s", name)
		}
	}
}

func TestLargeIdentifiersAndOverflow(t *testing.T) {
	a, _ := ParseVersion("1.2.3-99999999999999999999999999998")
	b, _ := ParseVersion("1.2.3-99999999999999999999999999999")
	if a.Compare(b) >= 0 || b.Compare(a) <= 0 {
		t.Fatal("large numeric identifiers compared incorrectly")
	}
	if _, err := ParseVersion("1.2.3-01"); err != nil {
		t.Fatal("unexpected strict prerelease validation")
	}
	if _, err := ParseVersion("1.999999999999999999999999999.0"); err == nil {
		t.Fatal("overflow accepted")
	}
	maxInt := int(^uint(0) >> 1)
	for _, name := range []string{"master", "hotfix", "develop", "feature", "release"} {
		strategy, _ := StrategyByName(name)
		base := Version{Major: 1, Minor: maxInt, Patch: maxInt}
		ctx := GitContext{Branch: "release/1." + strings.Repeat("9", 30) + ".0", CommitDistance: 1}
		if _, err := strategy.NextVersion(base, ctx); err == nil {
			t.Fatalf("%s ignored overflow", name)
		}
	}
	for _, name := range []string{"develop", "feature"} {
		strategy, _ := StrategyByName(name)
		if _, err := strategy.NextVersion(Version{}, GitContext{HasRelease: true, ReleaseMajor: 1, ReleaseMinor: maxInt}); err == nil {
			t.Fatalf("%s ignored release minor overflow", name)
		}
	}
	for _, n := range []int{0, 9, 10, 11} {
		v, err := (ReleaseStrategy{}).NextVersion(Version{}, GitContext{Branch: "release/1.2.0", CommitDistance: n})
		if err != nil || v.PreRelease != "rc"+strconv.Itoa(n) {
			t.Fatalf("counter %d: %+v, %v", n, v, err)
		}
	}
}

func FuzzDefaultTagRoundTrip(f *testing.F) {
	f.Add(uint32(1), uint32(2), uint32(10))
	f.Add(uint32(0), uint32(0), uint32(0))
	f.Fuzz(func(t *testing.T, major, minor, distance uint32) {
		ctx := GitContext{Branch: fmt.Sprintf("release/%d.%d.0", major, minor), CommitDistance: int(distance), ShortSHA: "abc1234"}
		v, err := (ReleaseStrategy{}).NextVersion(Version{}, ctx)
		if err != nil {
			t.Fatal(err)
		}
		rule, rules, err := DefaultConfig().ResolveRule(ctx.Branch)
		if err != nil {
			t.Fatal(err)
		}
		out := RenderVersion(v, ctx, rule, Version{})
		tag, ok, err := ParseTag(out, rules)
		if err != nil || !ok || tag.Version.String() != out {
			t.Fatalf("round trip %q: %+v, %v", out, tag, err)
		}
		ctx.TagName = "previous"
		for _, template := range []struct{ prefix, suffix string }{
			{"app-", "+{sha}"},
			{"build-{N}-", "-{sha}"},
			{"{branch}|{slug}|", "-from-{tag}"},
			{"{pre}|{buildMeta}|", ""},
		} {
			rules := compileRules(t, Rule{Pattern: "^release/.*$", Strategy: "release", Prefix: str(template.prefix), Suffix: str(template.suffix)})
			out := RenderVersion(v, ctx, &rules[0], Version{})
			tag, ok, err := ParseTag(out, rules)
			want := Version{Major: int(major), Minor: int(minor), PreRelease: "rc" + strconv.FormatUint(uint64(distance), 10)}
			if err != nil || !ok || tag.Name != out || tag.Version != want || tag.Kind != "rc" {
				t.Fatalf("custom round trip %q: %+v, %v, %v", out, tag, ok, err)
			}
		}
	})
}
