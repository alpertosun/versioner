package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/alpertosun/versioner/core"
	"github.com/alpertosun/versioner/internal/testrepo"
)

func writeConfig(t *testing.T, cfg *core.Config) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("versioner.config.json", data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConfigAndFileProduceSameVersion(t *testing.T) {
	tests := []struct{ branch, want string }{
		{"main", "1.2.4"}, {"master", "1.2.4"},
		{"release/1.3.0", "1.3.0-rc1"}, {"develop", "1.3.0-beta1"},
		{"feature/test", "1.3.0-alpha1"}, {"hotfix/test", "1.2.4"},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			r := testrepo.New(t, tt.branch)
			r.Git("tag", "1.2.3")
			r.Commit("next")
			r.Use()
			before, err := run()
			if err != nil || before != tt.want {
				t.Fatalf("without config: %q, %v", before, err)
			}
			writeConfig(t, core.DefaultConfig())
			after, err := run()
			if err != nil || after != before {
				t.Fatalf("with config: %q, %v", after, err)
			}
		})
	}
}

func TestConfiguredBranchStrategy(t *testing.T) {
	r := testrepo.New(t, "integration")
	r.Git("tag", "1.2.3")
	r.Commit("next")
	r.Use()
	cfg := core.DefaultConfig()
	suffix := ""
	cfg.Rules = append(cfg.Rules, core.Rule{Pattern: "^integration$", Strategy: "develop", Suffix: &suffix})
	writeConfig(t, cfg)
	if out, err := run(); err != nil || out != "1.3.0-beta1" {
		t.Fatalf("configured strategy: %q, %v", out, err)
	}
}

func TestInvalidConfigStopsBeforeGit(t *testing.T) {
	for _, data := range []string{
		"{bad json",
		`{"rules":[{"pattern":"[","strategy":"master"}]}`,
		`{"rules":[{"pattern":".*","strategy":"unknown"}]}`,
		`{"matching":"unknown","rules":[]}`,
	} {
		t.Run(data, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("VERSIONER_CONFIG", "versioner.config.json")
			if err := os.WriteFile("versioner.config.json", []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := run(); err == nil || strings.Contains(err.Error(), "cannot get branch") {
				t.Fatalf("config error not reported before Git: %v", err)
			}
		})
	}
	t.Run("unreadable", func(t *testing.T) {
		t.Setenv("VERSIONER_CONFIG", t.TempDir())
		if _, err := run(); err == nil || !strings.Contains(err.Error(), "read config") {
			t.Fatalf("read error ignored: %v", err)
		}
	})
}

func TestCustomConfigReadsItsOwnTags(t *testing.T) {
	r := testrepo.New(t, "main")
	r.Git("tag", "app-1.2.3")
	r.Git("tag", "99.0.0")
	r.Commit("next")
	r.Use()
	cfg := core.DefaultConfig()
	for i := range cfg.Rules {
		prefix := "app-"
		cfg.Rules[i].Prefix = &prefix
	}
	writeConfig(t, cfg)
	t.Setenv("VERSIONER_CONFIG", filepath.Join(r.Dir, "versioner.config.json"))
	out, err := run()
	if err != nil || out != "app-1.2.4" {
		t.Fatalf("custom tag base: %q, %v", out, err)
	}
	r.Git("tag", out)
	r.Commit("another")
	out, err = run()
	if err != nil || out != "app-1.2.5" {
		t.Fatalf("generated tag could not be read: %q, %v", out, err)
	}
}

func TestSHASuffixUsesCommitHistory(t *testing.T) {
	r := testrepo.New(t, "main")
	r.Git("tag", "1.2.3-ffffff0")
	r.Commit("newer hash")
	r.Git("tag", "1.2.4-0000001")
	r.Commit("next")
	r.Use()
	cfg := core.DefaultConfig()
	for i := range cfg.Rules {
		suffix := "-{sha}"
		cfg.Rules[i].Suffix = &suffix
	}
	writeConfig(t, cfg)
	out, err := run()
	if err != nil || !strings.HasPrefix(out, "1.2.5-") {
		t.Fatalf("SHA was treated as version precedence: %q, %v", out, err)
	}
}

func TestSnapshotDoesNotChangeVersion(t *testing.T) {
	r := testrepo.New(t, "release/1.3.0")
	r.Git("tag", "1.3.0-rc9")
	r.Commit("candidate ten")
	r.Git("tag", "1.3.0-rc10")
	r.Commit("next")
	r.Use()
	before, err := run()
	if err != nil || before != "1.3.0-rc1" {
		t.Fatalf("before snapshot: %q, %v", before, err)
	}
	r.Git("tag", "snapshot")
	after, err := run()
	if err != nil || after != before {
		t.Fatalf("snapshot changed version: %q -> %q (%v)", before, after, err)
	}
}

func TestPrereleaseCommitDistances(t *testing.T) {
	for _, tt := range []struct{ branch, prefix string }{
		{"release/1.3.0", "1.3.0-rc"},
		{"develop", "1.3.0-beta"},
		{"feature/test", "1.3.0-alpha"},
	} {
		t.Run(tt.branch, func(t *testing.T) {
			r := testrepo.New(t, tt.branch)
			r.Git("tag", "1.2.3")
			for i := 0; i < 9; i++ {
				r.Commit("next")
			}
			r.Use()
			for distance := 9; distance <= 11; distance++ {
				if distance > 9 {
					r.Commit("next")
				}
				want := tt.prefix + strconv.Itoa(distance)
				out, err := run()
				if err != nil || out != want {
					t.Fatalf("distance %d: %q, %v, want %q", distance, out, err, want)
				}
			}
		})
	}
}

func TestStableBaseWithPrereleaseReference(t *testing.T) {
	for _, branch := range []string{"main", "hotfix/test"} {
		t.Run(branch, func(t *testing.T) {
			r := testrepo.New(t, branch)
			r.Git("tag", "1.2.3")
			r.Commit("candidate")
			r.Git("tag", "2.0.0-rc1")
			r.Use()
			out, err := run()
			if err != nil || out != "1.2.4" {
				t.Fatalf("stable base: %q, %v", out, err)
			}
		})
	}
}

func TestDetachedHEAD(t *testing.T) {
	for _, name := range []string{"single", "none", "multiple", "override", "custom"} {
		t.Run(name, func(t *testing.T) {
			r := testrepo.New(t, "main")
			if name != "none" && name != "custom" {
				r.Git("tag", "-a", "1.2.3", "-m", "release")
			}
			if name == "multiple" {
				r.Git("tag", "1.2.4")
			}
			r.Git("tag", "snapshot")
			r.Git("checkout", "-q", "--detach")
			r.Use()
			want := "1.2.3"
			switch name {
			case "override":
				t.Setenv("VERSIONER_BRANCH", "develop")
				want = "1.3.0-beta0"
			case "custom":
				cfg := core.DefaultConfig()
				for i := range cfg.Rules {
					prefix := "app-"
					cfg.Rules[i].Prefix = &prefix
				}
				writeConfig(t, cfg)
				r.Git("tag", "app-1.2.3")
				want = "app-1.2.3"
			}
			out, err := run()
			if name == "none" || name == "multiple" {
				if err == nil || !strings.Contains(err.Error(), "VERSIONER_BRANCH") {
					t.Fatalf("expected actionable detached HEAD error: %q, %v", out, err)
				}
			} else if err != nil || out != want {
				t.Fatalf("detached HEAD: %q, %v, want %q", out, err, want)
			}
		})
	}
}

func TestReleaseAtTagAndInitialVersion(t *testing.T) {
	t.Run("rc0", func(t *testing.T) {
		r := testrepo.New(t, "release/1.3.0")
		r.Git("tag", "1.3.0-rc9")
		r.Use()
		out, err := run()
		if err != nil || out != "1.3.0-rc0" {
			t.Fatalf("release at tag: %q, %v", out, err)
		}
	})
	t.Run("no tags", func(t *testing.T) {
		r := testrepo.New(t, "develop")
		r.Use()
		out, err := run()
		if err != nil || out != "0.2.0-beta1" {
			t.Fatalf("initial version: %q, %v", out, err)
		}
	})
}

func TestReleaseBranchAndMerge(t *testing.T) {
	r := testrepo.New(t, "develop")
	r.Git("tag", "1.2.3")
	r.Git("update-ref", "refs/remotes/origin/release/2.5.0", "HEAD")
	r.Git("config", "color.ui", "always")
	r.Use()
	out, err := run()
	if err != nil || out != "2.6.0-beta0" {
		t.Fatalf("release base: %q, %v", out, err)
	}
	t.Setenv("VERSIONER_BRANCH", "main")
	t.Setenv("VERSIONER_MERGE", "true")
	out, err = run()
	if err != nil || out != "2.5.0" {
		t.Fatalf("merge version: %q, %v", out, err)
	}
}

func TestCommandOutput(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "versioner.exe")
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	r := testrepo.New(t, "main")
	r.Git("tag", "1.2.3")
	r.Use()
	cmd := exec.Command(binary)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil || strings.TrimSpace(stdout.String()) != "1.2.3" || stderr.Len() != 0 {
		t.Fatalf("output: stdout=%q stderr=%q error=%v", &stdout, &stderr, err)
	}
	if err := os.WriteFile("versioner.config.json", []byte("{bad"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command(binary)
	stdout.Reset()
	stderr.Reset()
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "parse config") {
		t.Fatalf("invalid config output: stdout=%q stderr=%q error=%v", &stdout, &stderr, err)
	}
}
