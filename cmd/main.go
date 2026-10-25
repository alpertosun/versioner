package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alpertosun/versioner/core"
	"github.com/alpertosun/versioner/git"
)

func main() {
	out, err := run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}

func run() (string, error) {
	cfgPath := os.Getenv("VERSIONER_CONFIG")
	if cfgPath == "" {
		cfgPath = "versioner.config.json"
	}
	cfg, err := core.LoadConfig(cfgPath)
	if err != nil {
		return "", err
	}
	rules, err := cfg.Compile()
	if err != nil {
		return "", err
	}
	branch := os.Getenv("VERSIONER_BRANCH")
	if branch == "" {
		branch, err = git.CurrentBranch()
		if err != nil {
			return "", fmt.Errorf("cannot get branch: %w", err)
		}
	}
	if branch == "HEAD" {
		tag, err := git.TagOnHEAD(rules)
		if err != nil {
			return "", err
		}
		return tag.Name, nil
	}
	rule, rules, err := cfg.ResolveRule(branch)
	if err != nil {
		return "", err
	}
	tags, err := git.ReadTags(rules)
	if err != nil {
		return "", err
	}
	reference, err := git.NearestTag(tags, false)
	if err != nil {
		return "", err
	}
	stable, err := git.NearestTag(tags, true)
	if err != nil {
		return "", err
	}
	ctx := core.GitContext{Branch: branch, VersionerMerge: os.Getenv("VERSIONER_MERGE") == "true"}
	if reference != nil {
		ctx.TagName = reference.Name
		ctx.CommitDistance = reference.Distance
	} else {
		ctx.CommitDistance, err = git.CommitDistanceSinceTagName("")
		if err != nil {
			return "", err
		}
	}
	base := core.Version{Major: 0, Minor: 1, Patch: 0}
	var stableDistance int
	if stable != nil {
		base, stableDistance = stable.Version, stable.Distance
	} else {
		stableDistance, err = git.CommitDistanceSinceTagName("")
		if err != nil {
			return "", err
		}
	}
	ctx.StableCommitDistance = &stableDistance
	ctx.ShortSHA, err = git.ShortSHA()
	if err != nil {
		return "", err
	}
	ctx.ReleaseMajor, ctx.ReleaseMinor, ctx.HasRelease, err = git.HighestReleaseAB()
	if err != nil {
		return "", err
	}
	strategy, err := core.StrategyByName(rule.Strategy)
	if err != nil {
		return "", err
	}
	engine := core.NewEngine()
	engine.Register(ctx.Branch, strategy)
	next, err := engine.Resolve(ctx, base)
	if err != nil {
		return "", err
	}
	return core.RenderVersion(next, ctx, rule, base), nil
}
