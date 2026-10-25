package core

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestFormatCompatibility(t *testing.T) {
	strategies := []struct {
		name, branch, version, kind, meta string
	}{
		{"master", "master", "1.2.4", "stable", ""},
		{"main", "main", "1.2.4", "stable", ""},
		{"release", "release/1.3.0", "1.3.0-rc", "rc", "gabc1234"},
		{"develop", "develop", "1.3.0-beta", "beta", "abc1234"},
		{"feature", "feature/test", "1.3.0-alpha", "alpha", "abc1234"},
		{"hotfix", "hotfix/test", "1.2.4", "stable", "gabc1234"},
	}
	for _, tt := range strategies {
		for _, distance := range []int{0, 9, 10, 11} {
			n := strconv.Itoa(distance)
			for _, suffix := range []*string{nil, str(""), str("+{sha}"), str("|{pre}|{buildMeta}"), str("-from-{tag}")} {
				t.Run(tt.name+"/"+n+"/"+safeStr(suffix), func(t *testing.T) {
					ctx := GitContext{Branch: tt.branch, CommitDistance: distance, ShortSHA: "abc1234", TagName: "app-1.0.0"}
					rules := compileRules(t, Rule{Pattern: ".*", Strategy: tt.name, Prefix: str("app-{N}-"), Suffix: suffix})
					strategy, err := StrategyByName(tt.name)
					if err != nil {
						t.Fatal(err)
					}
					v, err := strategy.NextVersion(Version{Major: 1, Minor: 2, Patch: 3}, ctx)
					if err != nil {
						t.Fatal(err)
					}
					canonical, pre := tt.version, ""
					if tt.kind != "stable" {
						canonical += n
						pre = tt.kind + n
					} else if (tt.name == "main" || tt.name == "master") && distance == 0 {
						canonical = "1.2.3"
					}
					want := "app-" + n + "-" + canonical
					fields := map[string]string{"N": n, "pre": pre}
					meta := ""
					switch {
					case suffix == nil:
						if tt.meta != "" {
							want += "+" + tt.meta
							meta = tt.meta
							fields["buildMeta"], fields["sha"] = tt.meta, "abc1234"
						}
					case *suffix == "+{sha}":
						want += "+abc1234"
						fields["buildMeta"], fields["sha"] = tt.meta, "abc1234"
					case *suffix == "|{pre}|{buildMeta}":
						want += "|" + pre + "|" + tt.meta
						fields["buildMeta"] = tt.meta
					case *suffix == "-from-{tag}":
						want += "-from-app-1.0.0"
						fields["tag"] = "app-1.0.0"
					}
					out := RenderVersion(v, ctx, &rules[0], Version{})
					if out != want {
						t.Fatalf("output %q, want %q", out, want)
					}
					tag, ok, err := ParseTag(out, rules)
					if err != nil || !ok {
						t.Fatalf("parse %q: %v, %v", out, ok, err)
					}
					minor, patch := 2, 4
					if tt.kind != "stable" {
						minor, patch = 3, 0
					} else if canonical == "1.2.3" {
						patch = 3
					}
					wantVersion := Version{Major: 1, Minor: minor, Patch: patch, PreRelease: pre, BuildMeta: meta}
					if tag.Name != want || tag.Version != wantVersion || tag.Kind != tt.kind {
						t.Fatalf("tag %+v, want version %+v, kind %s", tag, wantVersion, tt.kind)
					}
					for key, value := range fields {
						if actual, present := tag.Fields[key]; !present || actual != value {
							t.Fatalf("field %s = %q (%v), want %q", key, actual, present, value)
						}
					}
				})
			}
		}
	}
}

func checkSharedFormat(t *testing.T, format versionFormat, prefix string, ctx GitContext, want string, wantVersion Version) {
	t.Helper()
	rule := CompiledRule{Re: regexp.MustCompile(".*"), Strategy: "release", Prefix: &prefix}
	rule.format = compileRuleFormat(rule, format)
	v := format.apply(Version{Major: 1, Minor: 2, Patch: 3}, ctx)
	out := RenderVersion(v, ctx, &rule, Version{})
	if out != want {
		t.Fatalf("output %q, want %q", out, want)
	}
	tag, ok, err := ParseTag(out, []CompiledRule{rule})
	if err != nil || !ok {
		t.Fatalf("parse %q: %v, %v", out, ok, err)
	}
	if tag.Name != want || tag.Version != wantVersion || tag.Kind != format.prereleasePrefix {
		t.Fatalf("tag %+v, want version %+v, kind %s", tag, wantVersion, format.prereleasePrefix)
	}
	for key, value := range map[string]string{
		"pre": wantVersion.PreRelease, "buildMeta": wantVersion.BuildMeta,
		"N": strconv.Itoa(ctx.CommitDistance), "sha": ctx.ShortSHA,
	} {
		if actual, present := tag.Fields[key]; !present || actual != value {
			t.Fatalf("field %s = %q (%v), want %q", key, actual, present, value)
		}
	}
}

func TestSharedFormatDefinition(t *testing.T) {
	format := versionFormat{prereleasePrefix: "candidate", metadataEnabled: true, metadataPrefix: "git-"}
	checkSharedFormat(t, format, "", GitContext{CommitDistance: 10, ShortSHA: "abc1234"},
		"1.2.3-candidate10+git-abc1234",
		Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "candidate10", BuildMeta: "git-abc1234"})
}

func TestSharedFormatFieldLengths(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("requires a 64-bit counter")
	}
	format := versionFormat{prereleasePrefix: "candidate", metadataEnabled: true, metadataPrefix: "git-"}
	sha := strings.Repeat("a", 64)
	pre, meta := "candidate9223372036854775807", "git-"+sha
	ctx := GitContext{CommitDistance: int(^uint(0) >> 1), ShortSHA: sha}
	// Keep each field before version so it must be matched, not derived from version.
	for _, tt := range []struct{ prefix, decoration string }{
		{"{pre}|", pre + "|"},
		{"{buildMeta}|", meta + "|"},
		{"{pre}|{buildMeta}|", pre + "|" + meta + "|"},
	} {
		t.Run(tt.prefix, func(t *testing.T) {
			checkSharedFormat(t, format, tt.prefix, ctx, tt.decoration+"1.2.3-"+pre+"+"+meta,
				Version{Major: 1, Minor: 2, Patch: 3, PreRelease: pre, BuildMeta: meta})
		})
	}
}

func TestSharedFormatVersionLength(t *testing.T) {
	phase, sha := strings.Repeat("c", 100), strings.Repeat("a", 64)
	format := versionFormat{prereleasePrefix: phase, metadataEnabled: true, metadataPrefix: "git-"}
	want := "1.2.3-" + phase + "10+git-" + sha
	if len(want) != 177 {
		t.Fatalf("fixture length %d, want 177", len(want))
	}
	checkSharedFormat(t, format, "", GitContext{CommitDistance: 10, ShortSHA: sha}, want,
		Version{Major: 1, Minor: 2, Patch: 3, PreRelease: phase + "10", BuildMeta: "git-" + sha})
}

func TestTemplateContextFields(t *testing.T) {
	for _, tt := range []struct{ branch, slug, tag string }{
		{"feature/{sha}", "feature-sha", "old-{N}-1.2.3"},
		{"feature/örnek", "feature-rnek", ""},
		{"feature/abcdefghijklmnopq", "feature-abcdefgh", "app-1.2.3"},
	} {
		t.Run(tt.branch, func(t *testing.T) {
			rules := compileRules(t, Rule{Pattern: "^feature/.*$", Strategy: "feature", Prefix: str("{branch}|{slug}|{unknown}"), Suffix: str("-from-{tag}")})
			ctx := GitContext{Branch: tt.branch, ShortSHA: "abc1234", CommitDistance: 10, TagName: tt.tag}
			v, err := (FeatureStrategy{}).NextVersion(Version{Major: 1, Minor: 2}, ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := tt.branch + "|" + tt.slug + "|1.3.0-alpha10-from-" + tt.tag
			out := RenderVersion(v, ctx, &rules[0], Version{})
			if out != want {
				t.Fatalf("output %q, want %q", out, want)
			}
			tag, ok, err := ParseTag(out, rules)
			if err != nil || !ok || tag.Kind != "alpha" || tag.Fields["branch"] != tt.branch || tag.Fields["slug"] != tt.slug || tag.Fields["tag"] != tt.tag {
				t.Fatalf("parse %+v, %v, %v", tag, ok, err)
			}
		})
	}
}

func TestUncompiledRuleAndNilRule(t *testing.T) {
	ctx := GitContext{Branch: "release/1.2.0", ShortSHA: "abc1234"}
	v := Version{Major: 1, Minor: 2, PreRelease: "rc0", BuildMeta: "gabc1234"}
	rule := CompiledRule{Strategy: "release", Prefix: str("app-"), Suffix: str("+{sha}")}
	if out := RenderVersion(v, ctx, &rule, Version{}); out != "app-1.2.0-rc0+abc1234" {
		t.Fatalf("uncompiled rule: %q", out)
	}
	if _, ok, err := ParseTag("app-1.2.0-rc0+abc1234", []CompiledRule{rule}); err != nil || !ok {
		t.Fatalf("uncompiled parse: %v, %v", ok, err)
	}
	if out := RenderVersion(v, ctx, nil, Version{}); out != "1.2.0-rc0+gabc1234" {
		t.Fatalf("nil rule: %q", out)
	}
	// A direct renderer call only needs a version and decorations.
	rule = CompiledRule{Prefix: str("v"), Suffix: str("")}
	if out := RenderVersion(v, ctx, &rule, Version{}); out != "v1.2.0-rc0" {
		t.Fatalf("rule without strategy: %q", out)
	}
}
