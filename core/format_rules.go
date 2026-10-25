package core

import (
	"fmt"
	"regexp"
	"strconv"
)

const (
	minSHALength  = 4
	maxSHALength  = 64
	maxSlugLength = 16
)

var (
	maxNumberLength = len(strconv.Itoa(int(^uint(0) >> 1)))
	shaPattern      = fmt.Sprintf(`[0-9a-fA-F]{%d,%d}`, minSHALength, maxSHALength)
)

type versionFormat struct {
	prereleasePrefix string
	metadataEnabled  bool
	metadataPrefix   string
	zeroPatch        bool
}

func (f versionFormat) apply(v Version, ctx GitContext) Version {
	if f.zeroPatch {
		v.Patch = 0
	}
	v.PreRelease = ""
	if f.prereleasePrefix != "" {
		v.PreRelease = f.prereleasePrefix + strconv.Itoa(ctx.CommitDistance)
	}
	v.BuildMeta = ""
	if f.metadataEnabled {
		v.BuildMeta = f.metadataPrefix + ctx.ShortSHA
	}
	return v
}

type templatePart struct {
	literal string
	field   string
}

type fieldDefinition struct {
	value       func(Version, GitContext, *ruleFormat) string
	pattern     func(*ruleFormat) string
	maxLength   func(*ruleFormat) int
	fromVersion bool
	numeric     bool
}

type compiledField struct {
	definition fieldDefinition
	re         *regexp.Regexp
	limit      int
}

type ruleFormat struct {
	parts        []templatePart
	fields       map[string]compiledField
	version      versionFormat
	keepMetadata bool
}

func numberField(value func(Version, GitContext) int, fromVersion bool) fieldDefinition {
	return fieldDefinition{
		value:       func(v Version, ctx GitContext, _ *ruleFormat) string { return strconv.Itoa(value(v, ctx)) },
		pattern:     func(*ruleFormat) string { return numberPattern },
		maxLength:   func(*ruleFormat) int { return maxNumberLength },
		fromVersion: fromVersion,
		numeric:     true,
	}
}

var formatFields = map[string]fieldDefinition{
	"version": {
		value: func(v Version, _ GitContext, f *ruleFormat) string {
			if !f.keepMetadata {
				v.BuildMeta = ""
			}
			return v.String()
		},
		pattern: (*ruleFormat).versionPattern, maxLength: (*ruleFormat).versionLength,
	},
	"major": numberField(func(v Version, _ GitContext) int { return v.Major }, true),
	"minor": numberField(func(v Version, _ GitContext) int { return v.Minor }, true),
	"patch": numberField(func(v Version, _ GitContext) int { return v.Patch }, true),
	"N":     numberField(func(_ Version, ctx GitContext) int { return ctx.CommitDistance }, false),
	"sha": {
		value:     func(_ Version, ctx GitContext, _ *ruleFormat) string { return ctx.ShortSHA },
		pattern:   func(*ruleFormat) string { return shaPattern },
		maxLength: func(*ruleFormat) int { return maxSHALength },
	},
	"pre": {
		value:   func(v Version, _ GitContext, _ *ruleFormat) string { return v.PreRelease },
		pattern: (*ruleFormat).prePattern, maxLength: (*ruleFormat).preLength,
		fromVersion: true,
	},
	"buildMeta": {
		value:   func(v Version, _ GitContext, _ *ruleFormat) string { return v.BuildMeta },
		pattern: (*ruleFormat).metadataPattern, maxLength: (*ruleFormat).metadataLength,
	},
	"branch": {
		value:   func(_ Version, ctx GitContext, _ *ruleFormat) string { return ctx.Branch },
		pattern: func(*ruleFormat) string { return `[^\r\n]+` },
	},
	"slug": {
		value:     func(_ Version, ctx GitContext, _ *ruleFormat) string { return sanitizeSlug(ctx.Branch) },
		pattern:   func(*ruleFormat) string { return fmt.Sprintf(`[a-z0-9-]{0,%d}`, maxSlugLength) },
		maxLength: func(*ruleFormat) int { return maxSlugLength },
	},
	"tag": {
		value:   func(_ Version, ctx GitContext, _ *ruleFormat) string { return ctx.TagName },
		pattern: func(*ruleFormat) string { return `[^\r\n]*` },
	},
}

func compileRuleFormat(rule CompiledRule, format versionFormat) *ruleFormat {
	f := &ruleFormat{
		version: format, keepMetadata: rule.Suffix == nil,
		fields: make(map[string]compiledField, len(formatFields)),
	}
	for key, definition := range formatFields {
		field := compiledField{definition: definition, re: regexp.MustCompile("^(?:" + definition.pattern(f) + ")$")}
		if definition.maxLength != nil {
			field.limit = definition.maxLength(f)
		}
		f.fields[key] = field
	}
	f.parts = append(f.parts, f.tokenize(safeStr(rule.Prefix))...)
	f.parts = append(f.parts, templatePart{field: "version"})
	f.parts = append(f.parts, f.tokenize(safeStr(rule.Suffix))...)
	return f
}

func (f *ruleFormat) tokenize(template string) []templatePart {
	var parts []templatePart
	pos := 0
	for _, loc := range reUnknownPlaceholder.FindAllStringIndex(template, -1) {
		if loc[0] > pos {
			parts = append(parts, templatePart{literal: template[pos:loc[0]]})
		}
		field := template[loc[0]+1 : loc[1]-1]
		if _, ok := f.fields[field]; ok {
			parts = append(parts, templatePart{field: field})
		}
		pos = loc[1]
	}
	if pos < len(template) {
		parts = append(parts, templatePart{literal: template[pos:]})
	}
	return parts
}

func (f *ruleFormat) prePattern() string {
	if f.version.prereleasePrefix == "" {
		return ""
	}
	return regexp.QuoteMeta(f.version.prereleasePrefix) + numberPattern
}

func (f *ruleFormat) preLength() int {
	if f.version.prereleasePrefix == "" {
		return 0
	}
	return len(f.version.prereleasePrefix) + maxNumberLength
}

func (f *ruleFormat) metadataPattern() string {
	if !f.version.metadataEnabled {
		return ""
	}
	return regexp.QuoteMeta(f.version.metadataPrefix) + shaPattern
}

func (f *ruleFormat) metadataLength() int {
	if !f.version.metadataEnabled {
		return 0
	}
	return len(f.version.metadataPrefix) + maxSHALength
}

func (f *ruleFormat) versionPattern() string {
	patch := numberPattern
	if f.version.zeroPatch {
		patch = "0"
	}
	pattern := numberPattern + `\.` + numberPattern + `\.` + patch
	if pre := f.prePattern(); pre != "" {
		pattern += "-" + pre
	}
	if f.keepMetadata && f.version.metadataEnabled {
		pattern += `\+` + f.metadataPattern()
	}
	return pattern
}

func (f *ruleFormat) versionLength() int {
	patch := maxNumberLength
	if f.version.zeroPatch {
		patch = 1
	}
	length := 2*maxNumberLength + patch + 2
	if pre := f.preLength(); pre != 0 {
		length += 1 + pre
	}
	if f.keepMetadata && f.version.metadataEnabled {
		length += 1 + f.metadataLength()
	}
	return length
}
