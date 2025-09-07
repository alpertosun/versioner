package core

import (
	"fmt"
	"strings"
)

type Engine struct {
	exact  map[string]Strategy
	prefix map[string]Strategy
}

func NewEngine() *Engine {
	return &Engine{
		exact:  make(map[string]Strategy),
		prefix: make(map[string]Strategy),
	}
}

func (e *Engine) Register(key string, strategy Strategy) {
	if strings.HasSuffix(key, "/") {
		e.prefix[key] = strategy
	} else {
		e.exact[key] = strategy
	}
}

func (e *Engine) Resolve(ctx GitContext, base Version) (Version, error) {
	if strat, ok := e.exact[ctx.Branch]; ok {
		return strat.NextVersion(base, ctx)
	}
	for p, strat := range e.prefix {
		if strings.HasPrefix(ctx.Branch, p) {
			return strat.NextVersion(base, ctx)
		}
	}
	return Version{}, fmt.Errorf("no strategy found for branch: %s", ctx.Branch)
}
