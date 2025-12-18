package core

import (
	"regexp"
	"strconv"
	"strings"
)

// RenderVersion builds the final output string by applying rule prefix/suffix
// with placeholders around the canonical SemVer produced by strategy.
// To avoid double metadata, if a rule suffix is set (even empty), we render canonical version WITHOUT build metadata.
// Placeholders still see the original v.BuildMeta value.
func RenderVersion(v Version, ctx GitContext, rule *CompiledRule, latestTag Version) string {
	canonical := v.String()
	if rule != nil && rule.Suffix != nil {
		// drop metadata from canonical if suffix will likely add it again (or effectively remove it)
		v2 := v
		v2.BuildMeta = ""
		canonical = v2.String()
	}

	// Prepare map for placeholders.
	values := map[string]string{
		"sha":       ctx.ShortSHA,
		"branch":    ctx.Branch,
		"slug":      sanitizeSlug(ctx.Branch),
		"N":         strconv.Itoa(ctx.CommitDistance),
		"major":     strconv.Itoa(v.Major),
		"minor":     strconv.Itoa(v.Minor),
		"patch":     strconv.Itoa(v.Patch),
		"tag":       latestTag.String(),
		"pre":       v.PreRelease,
		"buildMeta": v.BuildMeta,
		"version":   canonical,
	}

	// Apply prefix/suffix templates.
	prefix := applyTemplate(safeStr(rule.Prefix), values)
	suffix := applyTemplate(safeStr(rule.Suffix), values)

	return prefix + canonical + suffix
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func applyTemplate(t string, values map[string]string) string {
	if t == "" {
		return ""
	}
	// Replace known {key}
	out := t
	for k, v := range values {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	// Remove unknown {something}
	reUnknown := regexp.MustCompile(`\{[a-zA-Z]+\}`)
	out = reUnknown.ReplaceAllString(out, "")
	return out
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
	if len(s) > 16 {
		s = s[:16]
	}
	return s
}
