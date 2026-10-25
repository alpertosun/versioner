package core

import (
	"fmt"
	"regexp"
	"strconv"
)

type Strategy interface {
	NextVersion(base Version, ctx GitContext) (Version, error)
}

type formattedStrategy interface {
	Strategy
	format() versionFormat
}

func lookupStrategy(name string) (formattedStrategy, error) {
	switch name {
	case "master", "main":
		return MasterStrategy{}, nil
	case "release":
		return ReleaseStrategy{}, nil
	case "develop":
		return DevelopStrategy{}, nil
	case "feature":
		return FeatureStrategy{}, nil
	case "hotfix":
		return HotfixStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", name)
	}
}

func StrategyByName(name string) (Strategy, error) {
	return lookupStrategy(name)
}

type DevelopStrategy struct{}

func (d DevelopStrategy) format() versionFormat {
	return versionFormat{prereleasePrefix: "beta", metadataEnabled: true, zeroPatch: true}
}

func (d DevelopStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	var major, minor int
	if ctx.HasRelease {
		major = ctx.ReleaseMajor
		minor = ctx.ReleaseMinor
	} else {
		major = base.Major
		minor = base.Minor
	}
	minor, err := increment(minor)
	if err != nil {
		return Version{}, err
	}
	return d.format().apply(Version{Major: major, Minor: minor}, ctx), nil
}

type FeatureStrategy struct{}

func (f FeatureStrategy) format() versionFormat {
	return versionFormat{prereleasePrefix: "alpha", metadataEnabled: true, zeroPatch: true}
}

func (f FeatureStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	var major, minor int
	if ctx.HasRelease {
		major = ctx.ReleaseMajor
		minor = ctx.ReleaseMinor
	} else {
		major = base.Major
		minor = base.Minor
	}
	minor, err := increment(minor)
	if err != nil {
		return Version{}, err
	}
	return f.format().apply(Version{Major: major, Minor: minor}, ctx), nil
}

type ReleaseStrategy struct{}

func (r ReleaseStrategy) format() versionFormat {
	return versionFormat{prereleasePrefix: "rc", metadataEnabled: true, metadataPrefix: "g", zeroPatch: true}
}

func (r ReleaseStrategy) NextVersion(_ Version, ctx GitContext) (Version, error) {
	major, minor, err := parseReleaseBranch(ctx.Branch)
	if err != nil {
		return Version{}, err
	}
	return r.format().apply(Version{Major: major, Minor: minor}, ctx), nil
}

func parseReleaseBranch(branch string) (major, minor int, err error) {
	re := regexp.MustCompile(`^release/(\d+)\.(\d+)\.0$`)
	matches := re.FindStringSubmatch(branch)
	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("invalid release branch name: %s", branch)
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid release major: %w", err)
	}
	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid release minor: %w", err)
	}
	return major, minor, nil
}

type HotfixStrategy struct{}

func (h HotfixStrategy) format() versionFormat {
	return versionFormat{metadataEnabled: true, metadataPrefix: "g"}
}

func (h HotfixStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	patch, err := increment(base.Patch)
	if err != nil {
		return Version{}, err
	}
	base.Patch = patch
	return h.format().apply(base, ctx), nil
}

type MasterStrategy struct{}

func (m MasterStrategy) format() versionFormat {
	return versionFormat{}
}

func (m MasterStrategy) NextVersion(base Version, ctx GitContext) (Version, error) {
	if ctx.VersionerMerge && ctx.HasRelease {
		return m.format().apply(Version{
			Major: ctx.ReleaseMajor,
			Minor: ctx.ReleaseMinor,
			Patch: 0,
		}, ctx), nil
	}

	distance := ctx.CommitDistance
	if ctx.StableCommitDistance != nil {
		distance = *ctx.StableCommitDistance
	}
	if distance > 0 {
		patch, err := increment(base.Patch)
		if err != nil {
			return Version{}, err
		}
		base.Patch = patch
	}

	return m.format().apply(base, ctx), nil
}
