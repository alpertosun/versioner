package core

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

type Rule struct {
	Pattern  string  `json:"pattern"`
	Strategy string  `json:"strategy"`
	Prefix   *string `json:"prefix,omitempty"`
	Suffix   *string `json:"suffix,omitempty"`
}

type Config struct {
	ConfigVersion int    `json:"configVersion"`
	Matching      string `json:"matching,omitempty"` // "first" | "longest"
	Rules         []Rule `json:"rules"`
}

type CompiledRule struct {
	Re       *regexp.Regexp
	Strategy string
	Prefix   *string
	Suffix   *string
	format   *ruleFormat
}

func DefaultConfig() *Config {
	patterns := []string{
		`^main$`, `^master$`, `^release\/(\d+)\.(\d+)\.0$`,
		`^develop$`, `^feature\/.*$`, `^hotfix\/.*$`,
	}
	strategies := []string{"master", "master", "release", "develop", "feature", "hotfix"}
	cfg := &Config{Matching: "first"}
	for i, pattern := range patterns {
		prefix, suffix := "", ""
		cfg.Rules = append(cfg.Rules, Rule{
			Pattern: pattern, Strategy: strategies[i], Prefix: &prefix, Suffix: &suffix,
		})
	}
	return cfg
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	for i, r := range c.Rules {
		if r.Pattern == "" {
			return fmt.Errorf("rule[%d]: empty pattern", i)
		}
		if _, err := lookupStrategy(r.Strategy); err != nil {
			return fmt.Errorf("rule[%d]: unknown strategy %q", i, r.Strategy)
		}
	}
	if c.Matching == "" {
		c.Matching = "first"
	}
	if c.Matching != "first" && c.Matching != "longest" {
		return fmt.Errorf("invalid Matching: %s", c.Matching)
	}
	return nil
}

func (c *Config) Compile() ([]CompiledRule, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if len(c.Rules) == 0 {
		return DefaultConfig().Compile()
	}
	crs := make([]CompiledRule, 0, len(c.Rules))
	for i, r := range c.Rules {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compile rule[%d] pattern:%q: %w", i, r.Pattern, err)
		}
		cr := CompiledRule{
			Re:       re,
			Strategy: r.Strategy,
			Prefix:   r.Prefix,
			Suffix:   r.Suffix,
		}
		strategy, _ := lookupStrategy(r.Strategy)
		cr.format = compileRuleFormat(cr, strategy.format())
		crs = append(crs, cr)
	}
	return crs, nil
}

// ResolveRule includes the selected fallback in the rules used to read tags.
func (c *Config) ResolveRule(branch string) (*CompiledRule, []CompiledRule, error) {
	rules, err := c.Compile()
	if err != nil {
		return nil, nil, err
	}
	if rule, ok := MatchRule(rules, branch, c.Matching); ok {
		return rule, rules, nil
	}
	defaults, err := DefaultConfig().Compile()
	if err != nil {
		return nil, nil, err
	}
	if rule, ok := MatchRule(defaults, branch, "first"); ok {
		rules = append(rules, *rule)
		return &rules[len(rules)-1], rules, nil
	}
	return nil, nil, fmt.Errorf("no strategy found for branch: %s", branch)
}

// MatchRule supports matching modes:
//   - "first"   : ilk eşleşen kural
//   - "longest" : branch içinde en uzun eşleşmeyi yapan kural
func MatchRule(rules []CompiledRule, branch string, matching string) (*CompiledRule, bool) {
	if matching == "longest" {
		var best *CompiledRule
		bestLen := -1
		for i := range rules {
			loc := rules[i].Re.FindStringIndex(branch)
			if loc == nil {
				continue
			}
			l := loc[1] - loc[0]
			if l > bestLen {
				best = &rules[i]
				bestLen = l
			}
		}
		if best != nil {
			return best, true
		}
		return nil, false
	}
	// default: first
	for i := range rules {
		if rules[i].Re.MatchString(branch) {
			return &rules[i], true
		}
	}
	return nil, false
}
