package core_test

import (
	"errors"
	"testing"

	"github.com/alpertosun/versioner/core"
)

type testStrategy struct {
	patch int
	err   error
}

func (s testStrategy) NextVersion(base core.Version, ctx core.GitContext) (core.Version, error) {
	base.Patch = s.patch + ctx.CommitDistance
	return base, s.err
}

func TestEngineRegisteredStrategies(t *testing.T) {
	engine := core.NewEngine()
	engine.Register("feature/", testStrategy{patch: 5})
	engine.Register("feature/special", testStrategy{patch: 10})
	engine.Register("integration", core.DevelopStrategy{})
	base := core.Version{Major: 1, Minor: 2, Patch: 3}
	for _, tt := range []struct{ branch, want string }{
		{"feature/test", "1.2.7"},
		{"feature/special", "1.2.12"},
		{"integration", "1.3.0-beta2+abc1234"},
	} {
		t.Run(tt.branch, func(t *testing.T) {
			ctx := core.GitContext{Branch: tt.branch, CommitDistance: 2, ShortSHA: "abc1234"}
			v, err := engine.Resolve(ctx, base)
			if err != nil || v.String() != tt.want {
				t.Fatalf("got %s, %v; want %s", v, err, tt.want)
			}
		})
	}
	if _, err := engine.Resolve(core.GitContext{Branch: "unknown"}, base); err == nil {
		t.Fatal("unknown branch accepted")
	}
	wantErr := errors.New("strategy failed")
	engine.Register("broken", testStrategy{err: wantErr})
	if _, err := engine.Resolve(core.GitContext{Branch: "broken"}, base); !errors.Is(err, wantErr) {
		t.Fatalf("strategy error lost: %v", err)
	}
}
