package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alpertosun/versioner/core"
	"github.com/alpertosun/versioner/git"
)

func main() {
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatalf("cannot get branch: %v", err)
	}

	shortSHA, err := git.ShortSHA()
	if err != nil {
		log.Fatalf("cannot get short sha: %v", err)
	}

	tagName, latest, _ := git.LatestTagNameAndVersion()
	var base core.Version
	if tagName == "" {
		base = core.Version{Major: 0, Minor: 1, Patch: 0}
	} else {
		base = latest
	}

	commitDistance, err := git.CommitDistanceSinceTagName(tagName)
	if err != nil {
		log.Printf("warn: cannot compute commit distance: %v", err)
		commitDistance = 0
	}

	relMaj, relMin, hasRel, relErr := git.HighestReleaseAB()
	if relErr != nil {
		log.Printf("warn: cannot scan release branches: %v", relErr)
	}

	versionerMerge := os.Getenv("VERSIONER_MERGE") == "true"

	ctx := core.GitContext{
		Branch:         branch,
		Target:         "",
		ShortSHA:       shortSHA,
		CommitDistance: commitDistance,
		HasRelease:     hasRel,
		ReleaseMajor:   relMaj,
		ReleaseMinor:   relMin,
		VersionerMerge: versionerMerge,
	}

	cfgPath := os.Getenv("VERSIONER_CONFIG")
	if cfgPath == "" {
		cfgPath = "versioner.config.json"
	}
	cfg, cfgErr := core.LoadConfig(cfgPath)

	if cfgErr == nil && cfg != nil && len(cfg.Rules) > 0 {
		if err := cfg.Validate(); err != nil {
			log.Fatalf("config validation error: %v", err)
		}
		compiled, err := cfg.Compile()
		if err != nil {
			log.Fatalf("config compile error: %v", err)
		}
		if rule, ok := core.MatchRule(compiled, ctx.Branch, cfg.Matching); ok {
			strategies := map[string]core.Strategy{
				"master":  core.MasterStrategy{},
				"main":    core.MasterStrategy{},
				"release": core.ReleaseStrategy{},
				"develop": core.DevelopStrategy{},
				"feature": core.FeatureStrategy{},
				"hotfix":  core.HotfixStrategy{},
			}
			strat, exists := strategies[rule.Strategy]
			if !exists {
				log.Fatalf("unknown strategy in config: %s", rule.Strategy)
			}
			next, err := strat.NextVersion(base, ctx)
			if err != nil {
				log.Fatalf("version generation failed: %v", err)
			}
			out := core.RenderVersion(next, ctx, rule, base)
			fmt.Println(out)
			return
		}
	}

	// v1 fallback
	engine := core.NewEngine()
	engine.Register("feature/", core.FeatureStrategy{})
	engine.Register("develop", core.DevelopStrategy{})
	engine.Register("release/", core.ReleaseStrategy{})
	engine.Register("hotfix/", core.HotfixStrategy{})
	engine.Register("master", core.MasterStrategy{})
	engine.Register("main", core.MasterStrategy{})

	next, err := engine.Resolve(ctx, base)
	if err != nil {
		log.Fatalf("version generation failed: %v", err)
	}
	fmt.Println(next.String())
}
