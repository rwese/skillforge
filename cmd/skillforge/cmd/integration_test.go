package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// IntegrationTestEnv holds the environment for integration tests.
type IntegrationTestEnv struct {
	tmpDir    string
	homeDir   string
	globalCfg string
	binPath   string
}

// setupIntegrationTest creates a temporary test environment.
func setupIntegrationTest(t *testing.T) *IntegrationTestEnv {
	t.Helper()

	tmpDir := t.TempDir()

	// Create home directory structure
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create .config/skillforge directory
	configDir := filepath.Join(homeDir, ".config", "skillforge")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Build the binary from the project root. The root is resolved from
	// this test file's own location, not from the process cwd: `go test`
	// runs each package with cwd = package directory, and other tests may
	// chdir into temp dirs.
	binPath := filepath.Join(tmpDir, "skillforge")

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller() failed")
	}
	root := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("could not find project root from %s", file)
		}
		root = parent
	}

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/skillforge/")
	cmd.Dir = root

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			t.Skipf("Skipping integration test - could not build binary: %s", stderr.String())
		}
		t.Skipf("Skipping integration test - could not build binary: %v", err)
	}

	return &IntegrationTestEnv{
		tmpDir:    tmpDir,
		homeDir:   homeDir,
		globalCfg: filepath.Join(configDir, "config.toml"),
		binPath:   binPath,
	}
}

// runSkillforge runs the skillforge binary with the given args.
func (e *IntegrationTestEnv) runSkillforge(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(e.binPath, args...)
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", e.homeDir),
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

// writeGlobalConfig writes a global config file.
func (e *IntegrationTestEnv) writeGlobalConfig(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(e.globalCfg, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

// createLocalConfig creates a .skillforge directory and config in tmpDir.
func (e *IntegrationTestEnv) createLocalConfig(t *testing.T) string {
	t.Helper()
	localDir := filepath.Join(e.tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(localDir, "config.toml")
	return configPath
}

// writeLocalConfig writes a local config file.
func (e *IntegrationTestEnv) writeLocalConfig(t *testing.T, content string) string {
	t.Helper()
	configPath := e.createLocalConfig(t)
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return configPath
}

// chdir changes to the tmpDir for local config detection and restores
// the previous cwd after the test so later tests can still resolve the
// project root when building the binary.
func (e *IntegrationTestEnv) chdir(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(e.tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

// --- Target Command Tests ---

func TestTargetRemoveNotFound(t *testing.T) {
	env := setupIntegrationTest(t)

	// Try to remove non-existent target
	_, stderr, exitCode := env.runSkillforge(t, "target", "remove", "nonexistent", "--yes")
	if exitCode == 0 {
		t.Fatal("expected error for non-existent target, got success")
	}

	if !bytes.Contains([]byte(stderr), []byte("not found")) {
		t.Errorf("expected 'not found' error, got: %s", stderr)
	}
}

// --- Output Format Tests ---

func TestTargetListCompactFormat(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
[targets.pi]
globalPath = "/tmp/pi"
enabled = true

[targets.local]
globalPath = "/local/path"
enabled = true
`)

	stdout, _, exitCode := env.runSkillforge(t, "target", "list", "-f", "compact")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	// Compact format should just show names, one per line
	lines := bytes.Split([]byte(stdout), []byte("\n"))
	if len(lines) < 2 {
		t.Errorf("expected multiple lines in compact format, got: %s", stdout)
	}

	// Should contain both targets
	if !bytes.Contains([]byte(stdout), []byte("pi")) {
		t.Errorf("expected 'pi' in compact format, got: %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("local")) {
		t.Errorf("expected 'local' in compact format, got: %s", stdout)
	}
}

// --- Scope Merging Tests ---

func TestScopeMergeGlobalAndLocal(t *testing.T) {
	env := setupIntegrationTest(t)
	env.chdir(t)

	// Write global config
	env.writeGlobalConfig(t, `
[targets.global-target]
globalPath = "/global/path"
enabled = true
`)

	// Write local config
	env.writeLocalConfig(t, `
[targets.local-target]
globalPath = "/local/path"
enabled = true
`)

	// List should show both targets
	stdout, _, exitCode := env.runSkillforge(t, "target", "list")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	if !bytes.Contains([]byte(stdout), []byte("global-target")) {
		t.Errorf("expected 'global-target' in output, got: %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("local-target")) {
		t.Errorf("expected 'local-target' in output, got: %s", stdout)
	}
}

// --- NO_COLOR Tests ---

func TestNoColorDisablesColors(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
[targets.pi]
globalPath = "/tmp/pi"
enabled = true
`)

	// Run with NO_COLOR
	cmd := exec.Command(env.binPath, "target", "list", "-f", "table")
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", env.homeDir),
		"PATH=" + os.Getenv("PATH"),
		"NO_COLOR=1",
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("target list failed: %s", stderr.String())
	}

	// Output should not contain ANSI escape codes
	output := stdout.String()
	if bytes.Contains([]byte(output), []byte("\x1b[")) {
		t.Errorf("expected no ANSI codes with NO_COLOR=1, got: %s", output)
	}
}
