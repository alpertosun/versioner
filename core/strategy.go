package core

import (
	"fmt"
	"regexp"
	"strconv"
)

type Strategy interface {
	NextVersion(base Version, ctx GitContext) (Version, error)
}

type DevelopStrategy struct{}

func (d DevelopStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	var major, minor int
	if ctx.HasRelease {
		major = ctx.ReleaseMajor
		minor = ctx.ReleaseMinor + 1
	} else {
		major = base.Major
		minor = base.Minor + 1
	}
	v := Version{
		Major:     major,
		Minor:     minor,
		Patch:     0,
		BuildMeta: ctx.ShortSHA,
	}
	v.PreRelease = fmt.Sprintf("beta%d", ctx.CommitDistance)
	return v, nil
}

type FeatureStrategy struct{}

func (f FeatureStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	var major, minor int
	if ctx.HasRelease {
		major = ctx.ReleaseMajor
		minor = ctx.ReleaseMinor + 1
	} else {
		major = base.Major
		minor = base.Minor + 1
	}
	v := Version{
		Major:     major,
		Minor:     minor,
		Patch:     0,
		BuildMeta: ctx.ShortSHA,
	}
	v.PreRelease = fmt.Sprintf("alpha%d", ctx.CommitDistance)
	return v, nil
}

type ReleaseStrategy struct{}

func (r ReleaseStrategy) NextVersion(_ Version, ctx GitContext) (Version, error) {
	re := regexp.MustCompile(`release/(\d+)\.(\d+)\.0`)
	matches := re.FindStringSubmatch(ctx.Branch)
	if len(matches) != 3 {
		return Version{}, fmt.Errorf("invalid release branch name: %s", ctx.Branch)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	v := Version{
		Major:     major,
		Minor:     minor,
		Patch:     0,
		BuildMeta: "g" + ctx.ShortSHA,
	}
	if ctx.CommitDistance > 0 {
		v.PreRelease = fmt.Sprintf("rc%d", ctx.CommitDistance)
	}
	return v, nil
}

type HotfixStrategy struct{}

func (h HotfixStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	base.Patch++
	base.PreRelease = ""
	base.BuildMeta = "g" + ctx.ShortSHA
	return base, nil
}

type MasterStrategy struct{}

func (m MasterStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	base.Patch++
	base.PreRelease = ""
	base.BuildMeta = ""
	return base, nil
}
