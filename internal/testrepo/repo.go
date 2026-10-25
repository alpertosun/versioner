package testrepo

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

type Repo struct {
	T   *testing.T
	Dir string
}

func New(t *testing.T, branch string) *Repo {
	t.Helper()
	r := &Repo{T: t, Dir: t.TempDir()}
	r.Git("init", "-q", "--template="+t.TempDir(), "-b", branch)
	r.Git("config", "user.name", "Versioner")
	r.Git("config", "user.email", "versioner@example.invalid")
	r.Git("config", "commit.gpgsign", "false")
	r.Git("config", "tag.gpgsign", "false")
	r.Commit("initial")
	return r
}

func (r *Repo) Git(args ...string) string {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	for _, value := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(value), "GIT_") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.T.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *Repo) Commit(message string) {
	r.T.Helper()
	r.Git("-c", "core.hooksPath=", "commit", "-q", "--allow-empty", "-m", message)
}

func (r *Repo) Use() {
	r.T.Helper()
	r.T.Chdir(r.Dir)
	r.T.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	r.T.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	r.T.Setenv("VERSIONER_CONFIG", "")
	r.T.Setenv("VERSIONER_BRANCH", "")
	r.T.Setenv("VERSIONER_MERGE", "")
}
