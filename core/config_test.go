package core

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func str(s string) *string { return &s }

func compileRules(t *testing.T, rules ...Rule) []CompiledRule {
	t.Helper()
	compiled, err := (&Config{Rules: rules}).Compile()
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func TestDefaultConfig(t *testing.T) {
	cfg, err := LoadConfig("../versioner.config.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.Compile(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatal("config file differs from built-in defaults")
	}
	missing, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || !reflect.DeepEqual(missing, cfg) {
		t.Fatalf("missing config: %v, %v", missing, err)
	}
	*missing.Rules[0].Suffix = "+{sha}"
	if *DefaultConfig().Rules[0].Suffix != "" {
		t.Fatal("defaults share mutable fields")
	}
}

func TestConfigValidation(t *testing.T) {
	for _, cfg := range []*Config{
		nil,
		{Matching: "unknown"},
		{Rules: []Rule{{Strategy: "master"}}},
		{Rules: []Rule{{Pattern: ".*", Strategy: "unknown"}}},
		{Rules: []Rule{{Pattern: "[", Strategy: "master"}}},
	} {
		if _, err := cfg.Compile(); err == nil {
			t.Fatalf("accepted invalid config: %#v", cfg)
		}
	}
}

func TestRuleMatching(t *testing.T) {
	cfg := &Config{Rules: []Rule{
		{Pattern: "^feature", Strategy: "master", Suffix: str("")},
		{Pattern: "^feature/.*$", Strategy: "feature", Suffix: str("")},
	}}
	rule, _, err := cfg.ResolveRule("feature/test")
	if err != nil || rule.Strategy != "master" {
		t.Fatalf("first match: %v, %v", rule, err)
	}
	cfg.Matching = "longest"
	rule, _, err = cfg.ResolveRule("feature/test")
	if err != nil || rule.Strategy != "feature" {
		t.Fatalf("longest match: %v, %v", rule, err)
	}
	rule, rules, err := cfg.ResolveRule("develop")
	if err != nil || rule.Strategy != "develop" {
		t.Fatalf("fallback: %v, %v", rule, err)
	}
	if _, ok, err := ParseTag("1.3.0-beta5", rules); err != nil || !ok {
		t.Fatalf("fallback tag: %v, %v", ok, err)
	}
	if _, _, err := cfg.ResolveRule("unknown"); err == nil {
		t.Fatal("unknown branch accepted")
	}
}

func TestEmptyConfigUsesAllDefaults(t *testing.T) {
	cfg := &Config{}
	rule, rules, err := cfg.ResolveRule("feature/test")
	if err != nil || rule.Strategy != "feature" || len(rules) != 6 {
		t.Fatalf("empty config fallback: %+v, %d rules, %v", rule, len(rules), err)
	}
	if _, ok, err := ParseTag("1.2.3", rules); err != nil || !ok {
		t.Fatalf("empty config lost stable tags: %v, %v", ok, err)
	}
}

func TestSuffixPresence(t *testing.T) {
	tests := []struct{ setting, want string }{
		{"", "1.2.0-rc0+gabc1234"},
		{`,"suffix":null`, "1.2.0-rc0+gabc1234"},
		{`,"suffix":""`, "1.2.0-rc0"},
		{`,"suffix":"+{sha}"`, "1.2.0-rc0+abc1234"},
	}
	for _, tt := range tests {
		t.Run(tt.setting, func(t *testing.T) {
			var cfg Config
			if err := json.Unmarshal([]byte(`{"rules":[{"pattern":".*","strategy":"release"`+tt.setting+`}]}`), &cfg); err != nil {
				t.Fatal(err)
			}
			rules, err := cfg.Compile()
			if err != nil {
				t.Fatal(err)
			}
			ctx := GitContext{Branch: "release/1.2.0", ShortSHA: "abc1234"}
			v, _ := (ReleaseStrategy{}).NextVersion(Version{}, ctx)
			out := RenderVersion(v, ctx, &rules[0], Version{})
			if out != tt.want {
				t.Fatalf("got %q, want %q", out, tt.want)
			}
			if _, ok, err := ParseTag(out, rules); err != nil || !ok {
				t.Fatalf("cannot read output: %v, %v", ok, err)
			}
		})
	}
}

func TestCompiledRuleSnapshot(t *testing.T) {
	prefix, suffix := "old-", ""
	cfg := &Config{Rules: []Rule{{Pattern: ".*", Strategy: "release", Prefix: &prefix, Suffix: &suffix}}}
	before, err := cfg.Compile()
	if err != nil {
		t.Fatal(err)
	}
	prefix, suffix = "new-", "+{sha}"
	after, err := cfg.Compile()
	if err != nil {
		t.Fatal(err)
	}
	ctx := GitContext{Branch: "release/1.2.0", CommitDistance: 10, ShortSHA: "abc1234"}
	v := Version{Major: 1, Minor: 2, PreRelease: "rc10", BuildMeta: "gabc1234"}
	for _, tt := range []struct {
		rules []CompiledRule
		want  string
	}{
		{before, "old-1.2.0-rc10"},
		{after, "new-1.2.0-rc10+abc1234"},
	} {
		if out := RenderVersion(v, ctx, &tt.rules[0], Version{}); out != tt.want {
			t.Errorf("render %q, want %q", out, tt.want)
		}
		if _, ok, err := ParseTag(tt.want, tt.rules); err != nil || !ok {
			t.Errorf("parse %q: %v, %v", tt.want, ok, err)
		}
	}
}
