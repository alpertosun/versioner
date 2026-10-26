package core

import (
	"regexp"
	"strings"
)

var reUnknownPlaceholder = regexp.MustCompile(`\{[a-zA-Z0-9_-]+\}`)

// RenderVersion builds the final output string by applying rule prefix/suffix
// with placeholders around the canonical SemVer produced by strategy.
// To avoid double metadata, if a rule suffix is set (even empty), we render canonical version WITHOUT build metadata.
// Placeholders still see the original v.BuildMeta value.
func RenderVersion(v Version, ctx GitContext, rule *CompiledRule, _ Version) string {
	if rule == nil {
		return v.String()
	}

	f := rule.format
	if f == nil {
		var format versionFormat
		if strategy, err := lookupStrategy(rule.Strategy); err == nil {
			format = strategy.format()
		}
		f = compileRuleFormat(*rule, format)
	}
	return f.render(v, ctx)
}

func (f *ruleFormat) render(v Version, ctx GitContext) string {
	var out strings.Builder
	for _, part := range f.parts {
		if part.field == "" {
			out.WriteString(part.literal)
		} else {
			out.WriteString(f.fields[part.field].definition.value(v, ctx, f))
		}
	}
	return out.String()
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "/", "-")
	// replace invalid chars with '-'
	re := regexp.MustCompile(`[^a-z0-9\-]+`)
	s = re.ReplaceAllString(s, "-")
	// collapse multiple '-'
	re2 := regexp.MustCompile(`\-{2,}`)
	s = re2.ReplaceAllString(s, "-")
	// trim '-'
	s = strings.Trim(s, "-")
	// limit length
	if len(s) > maxSlugLength {
		s = s[:maxSlugLength]
	}
	return s
}
