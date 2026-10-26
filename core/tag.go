package core

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
)

type ParsedTag struct {
	Name    string
	Version Version
	Kind    string
	Fields  map[string]string
}

func (t ParsedTag) Stable() bool { return t.Kind == "stable" }

func (t ParsedTag) SameMeaning(other ParsedTag) bool {
	a, b := t.Version, other.Version
	a.BuildMeta, b.BuildMeta = "", ""
	return a == b && t.Kind == other.Kind && maps.Equal(tagIdentity(t.Fields), tagIdentity(other.Fields))
}

func tagIdentity(fields map[string]string) map[string]string {
	out := maps.Clone(fields)
	for _, key := range []string{"version", "major", "minor", "patch", "pre", "buildMeta"} {
		delete(out, key)
	}
	return out
}

const maxTagMatchSteps = 100000

// ParseTag only accepts formats described by the supplied rules.
func ParseTag(name string, rules []CompiledRule) (ParsedTag, bool, error) {
	var result ParsedTag
	found := false
	steps := 0
	for _, rule := range rules {
		p := rule.format
		if p == nil {
			strategy, err := lookupStrategy(rule.Strategy)
			if err != nil {
				return ParsedTag{}, false, err
			}
			p = compileRuleFormat(rule, strategy.format())
		}
		err := p.match(name, 0, 0, map[string]string{}, &steps, func(fields map[string]string) error {
			if branch, ok := fields["branch"]; ok {
				if rule.Re != nil && !rule.Re.MatchString(branch) {
					return nil
				}
				if rule.Strategy == "release" {
					major, minor, err := parseReleaseBranch(branch)
					if err != nil || strconv.Itoa(major) != fields["major"] || strconv.Itoa(minor) != fields["minor"] {
						return nil
					}
				}
			}
			v, err := ParseVersion(fields["version"])
			if err != nil {
				return nil
			}
			kind := p.version.prereleasePrefix
			if kind == "" {
				kind = "stable"
			}
			parsed := ParsedTag{Name: name, Version: v, Kind: kind, Fields: maps.Clone(fields)}
			if found && !result.SameMeaning(parsed) {
				return fmt.Errorf("ambiguous tag %q: config rules give different meanings", name)
			}
			if !found {
				result, found = parsed, true
			}
			return nil
		})
		if err != nil {
			return ParsedTag{}, false, err
		}
	}
	return result, found, nil
}

// A bounded search handles repeated and adjacent placeholders without choosing
// an arbitrary capture when more than one interpretation is possible.
func (p *ruleFormat) match(input string, part, pos int, fields map[string]string, steps *int, accept func(map[string]string) error) error {
	*steps++
	if *steps > maxTagMatchSteps {
		return fmt.Errorf("tag matching exceeded its limit; use distinct separators between placeholders")
	}
	if part == len(p.parts) {
		if pos == len(input) {
			return accept(fields)
		}
		return nil
	}
	token := p.parts[part]
	if token.field == "" {
		if strings.HasPrefix(input[pos:], token.literal) {
			return p.match(input, part+1, pos+len(token.literal), fields, steps, accept)
		}
		return nil
	}
	if value, ok := fields[token.field]; ok {
		if strings.HasPrefix(input[pos:], value) {
			return p.match(input, part+1, pos+len(value), fields, steps, accept)
		}
		return nil
	}
	limit := len(input)
	field := p.fields[token.field]
	if field.limit > 0 {
		limit = min(limit, pos+field.limit)
	}
	for end := pos; end <= limit; end++ {
		*steps++
		if *steps > maxTagMatchSteps {
			return fmt.Errorf("tag matching exceeded its limit; use distinct separators between placeholders")
		}
		value := input[pos:end]
		if !field.re.MatchString(value) {
			continue
		}
		next := maps.Clone(fields)
		next[token.field] = value
		if !p.derive(next) {
			continue
		}
		if err := p.match(input, part+1, end, next, steps, accept); err != nil {
			return err
		}
	}
	return nil
}

func (p *ruleFormat) derive(fields map[string]string) bool {
	bind := func(key, value string) bool {
		if old, ok := fields[key]; ok && old != value {
			return false
		}
		fields[key] = value
		return true
	}
	if canonical, ok := fields["version"]; ok {
		v, err := ParseVersion(canonical)
		if err != nil {
			return false
		}
		for key, field := range p.fields {
			if field.definition.fromVersion && !bind(key, field.definition.value(v, GitContext{}, p)) {
				return false
			}
		}
		if v.PreRelease != "" && !bind("N", strings.TrimPrefix(v.PreRelease, p.version.prereleasePrefix)) {
			return false
		}
		if v.BuildMeta != "" && !bind("buildMeta", v.BuildMeta) {
			return false
		}
	}
	for key, value := range fields {
		if p.fields[key].definition.numeric {
			if _, err := strconv.Atoi(value); err != nil {
				return false
			}
		}
	}
	if meta, ok := fields["buildMeta"]; ok && meta != "" {
		if !bind("sha", strings.TrimPrefix(meta, p.version.metadataPrefix)) {
			return false
		}
	}
	if sha, ok := fields["sha"]; ok {
		meta := p.version.apply(Version{}, GitContext{ShortSHA: sha}).BuildMeta
		if !bind("buildMeta", meta) {
			return false
		}
	}
	if branch, ok := fields["branch"]; ok {
		if !bind("slug", p.fields["slug"].definition.value(Version{}, GitContext{Branch: branch}, p)) {
			return false
		}
	}
	return true
}
