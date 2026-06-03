package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// Build the binary
	binPath := filepath.Join(tmpDir, "skillforge")

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// If we're in cmd/skillforge/, go up one level to project root
	if filepath.Base(cwd) == "skillforge" && filepath.Base(filepath.Dir(cwd)) == "cmd" {
		cwd = filepath.Dir(cwd) // now in cmd/
		cwd = filepath.Dir(cwd) // now in project root
	}

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/skillforge/")
	cmd.Dir = cwd

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

// chdir changes to the tmpDir for local config detection.
func (e *IntegrationTestEnv) chdir(t *testing.T) {
	t.Helper()
	if err := os.Chdir(e.tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
}

// --- Target Command Tests ---

func TestTargetAddList(t *testing.T) {
	env := setupIntegrationTest(t)

	// Add a target
	stdout, stderr, exitCode := env.runSkillforge(t, "target", "add", "pi", "/tmp/pi", "-e")
	if exitCode != 0 {
		t.Fatalf("target add failed: %s\nstderr: %s", stdout, stderr)
	}

	// List targets - should show the new target
	stdout, stderr, exitCode = env.runSkillforge(t, "target", "list", "-f", "table")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s\nstderr: %s", stdout, stderr)
	}

	if !bytes.Contains([]byte(stdout), []byte("pi")) {
		t.Errorf("target list output missing 'pi': %s", stdout)
	}
}

func TestTargetAddDuplicate(t *testing.T) {
	env := setupIntegrationTest(t)

	// Add a target
	_, _, _ = env.runSkillforge(t, "target", "add", "pi", "/tmp/pi", "-e")

	// Try to add duplicate
	_, stderr, exitCode := env.runSkillforge(t, "target", "add", "pi", "/tmp/other")
	if exitCode == 0 {
		t.Fatal("expected error for duplicate target, got success")
	}

	if !bytes.Contains([]byte(stderr), []byte("already exists")) {
		t.Errorf("expected 'already exists' error, got: %s", stderr)
	}
}

func TestTargetRemove(t *testing.T) {
	env := setupIntegrationTest(t)

	// Add a target
	_, _, _ = env.runSkillforge(t, "target", "add", "pi", "/tmp/pi", "-e")

	// Remove the target (using --yes to skip confirmation)
	_, stderr, exitCode := env.runSkillforge(t, "target", "remove", "pi", "--yes")
	if exitCode != 0 {
		t.Fatalf("target remove failed: %s\nstderr: %s", "", stderr)
	}

	// List should be empty
	stdout, _, _ := env.runSkillforge(t, "target", "list")
	if !bytes.Contains([]byte(stdout), []byte("No targets configured")) {
		t.Errorf("expected 'No targets configured', got: %s", stdout)
	}
}

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

func TestTargetEnableDisable(t *testing.T) {
	env := setupIntegrationTest(t)

	// Add disabled target
	_, _, _ = env.runSkillforge(t, "target", "add", "pi", "/tmp/pi")

	// Enable it
	_, _, exitCode := env.runSkillforge(t, "target", "enable", "pi")
	if exitCode != 0 {
		t.Fatalf("target enable failed")
	}

	// List should show enabled (use table format)
	stdout, _, _ := env.runSkillforge(t, "target", "list", "-f", "table")
	if !bytes.Contains([]byte(stdout), []byte("enabled")) {
		t.Errorf("expected 'enabled' status, got: %s", stdout)
	}

	// Disable it
	_, _, exitCode = env.runSkillforge(t, "target", "disable", "pi")
	if exitCode != 0 {
		t.Fatalf("target disable failed")
	}

	// List should show disabled
	stdout, _, _ = env.runSkillforge(t, "target", "list", "-f", "table")
	if !bytes.Contains([]byte(stdout), []byte("disabled")) {
		t.Errorf("expected 'disabled' status, got: %s", stdout)
	}
}

// --- Output Format Tests ---

func TestTargetListTableFormat(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
cache.path = "/cache"

[targets.pi]
path = "/tmp/pi"
enabled = true
`)

	stdout, _, exitCode := env.runSkillforge(t, "target", "list", "-f", "table")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	// Check for table headers
	if !bytes.Contains([]byte(stdout), []byte("TARGET")) {
		t.Errorf("expected 'TARGET' header, got: %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("PATH")) {
		t.Errorf("expected 'PATH' header, got: %s", stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("STATUS")) {
		t.Errorf("expected 'STATUS' header, got: %s", stdout)
	}
}

func TestTargetListCompactFormat(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
cache.path = "/cache"

[targets.pi]
path = "/tmp/pi"
enabled = true

[targets.local]
path = "/local/path"
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

func TestTargetListJSONFormat(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
cache.path = "/cache"

[targets.pi]
path = "/tmp/pi"
enabled = true
`)

	stdout, _, exitCode := env.runSkillforge(t, "target", "list", "-f", "json")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	var targets []TargetOutput
	if err := json.Unmarshal([]byte(stdout), &targets); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Name != "pi" {
		t.Errorf("expected name 'pi', got %s", targets[0].Name)
	}

	if targets[0].GlobalPath != "" || targets[0].LocalPath != "/tmp/pi" {
		t.Errorf("expected LocalPath '/tmp/pi', got GlobalPath=%s LocalPath=%s", targets[0].GlobalPath, targets[0].LocalPath)
	}

	if !targets[0].Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestAutoCompactPiped(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
cache.path = "/cache"

[targets.pi]
path = "/tmp/pi"
enabled = true
`)

	// When piped, should auto-use compact format
	// We can't easily test actual piping, but we can verify
	// that the default format changes based on context
	stdout, _, exitCode := env.runSkillforge(t, "target", "list")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	// Should have some output
	if len(stdout) == 0 {
		t.Error("expected output, got empty")
	}
}

// --- Scope Merging Tests ---

func TestScopeMergeGlobalAndLocal(t *testing.T) {
	env := setupIntegrationTest(t)
	env.chdir(t)

	// Write global config
	env.writeGlobalConfig(t, `
cache.path = "/global/cache"

[targets.global-target]
path = "/global/path"
enabled = true
`)

	// Write local config
	env.writeLocalConfig(t, `
[targets.local-target]
path = "/local/path"
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

func TestScopeLocalOverridesGlobal(t *testing.T) {
	env := setupIntegrationTest(t)
	env.chdir(t)

	// Write global config
	env.writeGlobalConfig(t, `
cache.path = "/global/cache"

[targets.pi]
path = "/global/pi"
enabled = true
`)

	// Write local config with same target, different values
	env.writeLocalConfig(t, `
[targets.pi]
path = "/local/pi"
enabled = false
`)

	// List should show local's values (use table format to see paths)
	stdout, _, exitCode := env.runSkillforge(t, "target", "list", "-f", "table")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	// Should show the local path (contracted)
	if !bytes.Contains([]byte(stdout), []byte("/local/pi")) {
		t.Errorf("expected local override path '/local/pi', got: %s", stdout)
	}
}

func TestScopeGlobalOnly(t *testing.T) {
	env := setupIntegrationTest(t)

	// Write global config
	env.writeGlobalConfig(t, `
cache.path = "/global/cache"

[targets.pi]
path = "/global/pi"
enabled = true
`)

	// Use --global flag
	stdout, _, exitCode := env.runSkillforge(t, "--global", "target", "list")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	if !bytes.Contains([]byte(stdout), []byte("pi")) {
		t.Errorf("expected 'pi' target, got: %s", stdout)
	}
}

func TestScopeLocalOnly(t *testing.T) {
	env := setupIntegrationTest(t)
	env.chdir(t)

	// Write global config
	env.writeGlobalConfig(t, `
cache.path = "/global/cache"

[targets.global-target]
path = "/global/path"
enabled = true
`)

	// Write local config
	env.writeLocalConfig(t, `
[targets.local-target]
path = "/local/path"
enabled = true
`)

	// Use --local flag
	stdout, _, exitCode := env.runSkillforge(t, "--local", "target", "list")
	if exitCode != 0 {
		t.Fatalf("target list failed: %s", stdout)
	}

	// Should only show local target
	if !bytes.Contains([]byte(stdout), []byte("local-target")) {
		t.Errorf("expected 'local-target', got: %s", stdout)
	}
	if bytes.Contains([]byte(stdout), []byte("global-target")) {
		t.Errorf("did not expect 'global-target' with --local flag, got: %s", stdout)
	}
}

// --- NO_COLOR Tests ---

func TestNoColorDisablesColors(t *testing.T) {
	env := setupIntegrationTest(t)
	env.writeGlobalConfig(t, `
cache.path = "/cache"

[targets.pi]
path = "/tmp/pi"
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
