package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// baselineEnv is a simple test environment.
type baselineEnv struct {
	tmpDir  string
	homeDir string
	binPath string
}

func newBaselineEnv(t *testing.T) *baselineEnv {
	t.Helper()

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create .config/skillforge directory
	configDir := filepath.Join(homeDir, ".config", "skillforge")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	binPath := filepath.Join(tmpDir, "skillforge")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/skillforge/")
	cmd.Dir = findProjectRoot()

	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping integration test - could not build binary: %v", err)
	}

	return &baselineEnv{
		tmpDir:  tmpDir,
		homeDir: homeDir,
		binPath: binPath,
	}
}

func findProjectRoot() string {
	// Simple heuristic: go up until we find go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func (e *baselineEnv) run(args ...string) (string, string, int) {
	cmd := exec.Command(e.binPath, args...)
	cmd.Env = []string{
		"HOME=" + e.homeDir,
		"PATH=" + os.Getenv("PATH"),
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

func (e *baselineEnv) chdir() {
	os.Chdir(e.tmpDir)
}

// ============================================================
// TARGET TESTS - Local Scope
// ============================================================

func TestTarget_AddLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	_, stderr, code := env.run("target", "add", "local-pi", "/tmp/global", "/tmp/skills", "-e")
	if code != 0 {
		t.Fatalf("target add failed: %s", stderr)
	}

	// Verify it was created
	stdout, _, _ := env.run("target", "list", "-f", "json")
	if !strings.Contains(stdout, "local-pi") {
		t.Errorf("target not found in list: %s", stdout)
	}
}

func TestTarget_AddRequiresAllArgs(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	_, stderr, code := env.run("target", "add", "test", "/tmp")
	if code == 0 {
		t.Fatal("expected error for missing argument")
	}
	if !strings.Contains(stderr, "expected 3 args") {
		t.Errorf("expected argument count error, got: %s", stderr)
	}
}

func TestTarget_ListLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	// Add a target
	env.run("target", "add", "pi", "/tmp/global", "/tmp/pi", "-e")

	// List should show it
	stdout, _, _ := env.run("target", "list", "-f", "json")

	var targets []TargetOutput
	if err := json.Unmarshal([]byte(stdout), &targets); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Name != "pi" {
		t.Errorf("expected target 'pi', got %s", targets[0].Name)
	}
	if targets[0].GlobalPath != "" || targets[0].LocalPath == "" {
		t.Errorf("expected LocalPath to be set, got GlobalPath=%s LocalPath=%s", targets[0].GlobalPath, targets[0].LocalPath)
	}
}

func TestTarget_EnableLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("target", "add", "pi", "/tmp/global", "/tmp/pi")
	env.run("target", "enable", "pi")

	stdout, _, _ := env.run("target", "list", "-f", "json")
	if !strings.Contains(stdout, `"enabled":true`) {
		t.Errorf("expected enabled=true: %s", stdout)
	}
}

func TestTarget_DisableLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("target", "add", "pi", "/tmp/global", "/tmp/pi", "-e")
	env.run("target", "disable", "pi")

	stdout, _, _ := env.run("target", "list", "-f", "json")
	if !strings.Contains(stdout, `"enabled":false`) {
		t.Errorf("expected enabled=false: %s", stdout)
	}
}

func TestTarget_RemoveLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("target", "add", "pi", "/tmp/global", "/tmp/pi", "-e")
	env.run("target", "remove", "pi", "--yes")

	stdout, _, _ := env.run("target", "list", "-f", "json")
	if stdout != "[]\n" {
		t.Errorf("expected empty list, got: %s", stdout)
	}
}

func TestTarget_RemoveNotFound(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	_, stderr, code := env.run("target", "remove", "nonexistent", "--yes")
	if code == 0 {
		t.Fatal("expected error for non-existent target")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("expected 'not found' error: %s", stderr)
	}
}

// ============================================================
// TARGET TESTS - Global Scope
// ============================================================

func TestTarget_AddGlobal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	_, stderr, code := env.run("target", "add", "global-pi", "/tmp/global", "/tmp/local", "-e")
	if code != 0 {
		t.Fatalf("target add failed: %s", stderr)
	}

	// Verify it was saved globally
	globalCfg := filepath.Join(env.homeDir, ".config", "skillforge", "config.toml")
	data, err := os.ReadFile(globalCfg)
	if err != nil {
		t.Fatalf("could not read global config: %v", err)
	}
	if !strings.Contains(string(data), "global-pi") {
		t.Errorf("target not found in global config")
	}
}

func TestTarget_AddDuplicate(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("target", "add", "pi", "/tmp/global", "/tmp/pi", "-e")
	_, stderr, code := env.run("target", "add", "pi", "/tmp/other/global", "/tmp/other")

	if code == 0 {
		t.Fatal("expected error for duplicate")
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("expected 'already exists' error: %s", stderr)
	}
}

func TestSkillInstallLocalDoesNotFallbackWithoutGitRoot(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	globalCfg := filepath.Join(env.homeDir, ".config", "skillforge", "config.toml")
	cfg := `[targets.pi]
globalPath = "/tmp/global"
localPath = ".pi/skills"
enabled = true
`
	if err := os.WriteFile(globalCfg, []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, stderr, code := env.run("skill", "install", "docker", "--dry-run")
	if code == 0 {
		t.Fatal("expected local install to fail outside git root")
	}
	if !strings.Contains(stderr, "no install paths found") {
		t.Fatalf("expected no install paths error, got: %s", stderr)
	}
}

func TestSkillInstallLocalConfigRequiresGitRoot(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	localDir := filepath.Join(env.tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	cfg := `[targets.pi]
globalPath = "/tmp/global"
localPath = ".pi/skills"
enabled = true
`
	if err := os.WriteFile(filepath.Join(localDir, "config.toml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, stderr, code := env.run("skill", "install", "docker", "--dry-run")
	if code == 0 {
		t.Fatal("expected local install to fail outside git root")
	}
	if !strings.Contains(stderr, "no install paths found") {
		t.Fatalf("expected no install paths error, got: %s", stderr)
	}
}

func TestTarget_GlobalAddRemove(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("target", "add", "codex", "/tmp/global", "/tmp/local", "-e", "-s", "global")

	_, stderr, code := env.run("target", "global", "add", "codex", "shared", "/tmp/shared", "-s", "global")
	if code != 0 {
		t.Fatalf("target global add failed: %s", stderr)
	}

	stdout, _, _ := env.run("target", "list", "-s", "global", "-f", "json")
	var targets []TargetOutput
	if err := json.Unmarshal([]byte(stdout), &targets); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d: %s", len(targets), stdout)
	}
	if targets[0].GlobalPaths["shared"] != "/tmp/shared" {
		t.Fatalf("expected shared global path, got %#v", targets[0].GlobalPaths)
	}

	_, stderr, code = env.run("target", "global", "remove", "codex", "shared", "-s", "global")
	if code != 0 {
		t.Fatalf("target global remove failed: %s", stderr)
	}

	stdout, _, _ = env.run("target", "list", "-s", "global", "-f", "json")
	targets = nil
	if err := json.Unmarshal([]byte(stdout), &targets); err != nil {
		t.Fatalf("invalid JSON after remove: %v", err)
	}
	if len(targets[0].GlobalPaths) != 0 {
		t.Fatalf("expected global paths removed, got %#v", targets[0].GlobalPaths)
	}
}

// ============================================================
// REPO TESTS - Local Scope
// ============================================================

func TestRepo_AddLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	// Add a repo
	_, stderr, code := env.run("repo", "add", "https://github.com/test/repo.git", "--alias", "test", "-s", "local")
	if code != 0 {
		t.Fatalf("repo add failed: %s", stderr)
	}

	// Verify it was created
	stdout, _, _ := env.run("repo", "list", "-f", "json")
	if !strings.Contains(stdout, "test") {
		t.Errorf("repo not found in list: %s", stdout)
	}
}

func TestRepo_ListLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("repo", "add", "https://github.com/test/repo.git", "--alias", "test", "-s", "local")

	stdout, _, _ := env.run("repo", "list", "-f", "json")

	var repos []RepoOutput
	if err := json.Unmarshal([]byte(stdout), &repos); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(repos))
	}
}

func TestRepo_RemoveLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.run("repo", "add", "https://github.com/test/repo.git", "--alias", "test", "-s", "local")
	env.run("repo", "remove", "test", "--yes")

	stdout, _, _ := env.run("repo", "list", "-f", "json")
	if stdout != "[]\n" {
		t.Errorf("expected empty list, got: %s", stdout)
	}
}

// ============================================================
// REPO TESTS - Global Scope
// ============================================================

func TestRepo_AddGlobal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	_, stderr, code := env.run("repo", "add", "https://github.com/test/repo.git", "--alias", "global-test", "-s", "global")
	if code != 0 {
		t.Fatalf("repo add failed: %s", stderr)
	}

	// Verify it was saved globally
	globalCfg := filepath.Join(env.homeDir, ".config", "skillforge", "config.toml")
	data, err := os.ReadFile(globalCfg)
	if err != nil {
		t.Fatalf("could not read global config: %v", err)
	}
	if !strings.Contains(string(data), "global-test") {
		t.Errorf("repo not found in global config")
	}
}

// ============================================================
// SCOPE ISOLATION TESTS
// ============================================================

func TestScope_LocalDoesNotSeeGlobal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	// Add global target
	env.run("target", "add", "global-pi", "/tmp/global", "/tmp/local", "-e", "-s", "global")

	// Add local target with different name
	env.run("target", "add", "local-pi", "/tmp/global/local", "/tmp/local", "-e", "-s", "local")

	// List should only show local
	stdout, _, _ := env.run("target", "list", "-f", "json")

	var targets []TargetOutput
	json.Unmarshal([]byte(stdout), &targets)

	// Should have exactly 1 target (local only)
	if len(targets) != 1 {
		t.Errorf("expected 1 local target, got %d: %s", len(targets), stdout)
	}
	if targets[0].Name != "local-pi" {
		t.Errorf("expected 'local-pi', got %s", targets[0].Name)
	}
}

func TestScope_GlobalDoesNotSeeLocal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	// Add local target
	env.run("target", "add", "local-pi", "/tmp/global", "/tmp/local", "-e", "-s", "local")

	// Add global target with different name
	env.run("target", "add", "global-pi", "/tmp/global", "/tmp/local", "-e", "-s", "global")

	// List with global scope
	stdout, _, _ := env.run("target", "list", "-s", "global", "-f", "json")

	var targets []TargetOutput
	json.Unmarshal([]byte(stdout), &targets)

	// Should have exactly 1 target (global only)
	if len(targets) != 1 {
		t.Errorf("expected 1 global target, got %d: %s", len(targets), stdout)
	}
	if targets[0].Name != "global-pi" {
		t.Errorf("expected 'global-pi', got %s", targets[0].Name)
	}
}
