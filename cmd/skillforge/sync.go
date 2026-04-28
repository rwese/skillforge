package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rwese/skillforge-ng/internal/agents"
	"github.com/rwese/skillforge-ng/internal/config"
	"github.com/rwese/skillforge-ng/internal/repo"
	"github.com/rwese/skillforge-ng/pkg/grimoire"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories and update installed skills",
	Long: `Sync repositories and update installed skills.

This command performs up to three operations:
1. Updates all cached repositories (repo update)
2. Updates all installed skills to latest versions (skill update)
3. Syncs skills across agents (agent sync) - installs missing skills

Use --check to see what would be updated without making changes.
Use --skip-agent-sync to skip the agent synchronization step.`,
	RunE: runSync,
}

var (
	syncAgentFlag      string
	syncScopeFlag      string
	skipAgentSyncFlag  bool
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&checkFlag, "check", "c", false, "Check for updates without applying")
	syncCmd.Flags().StringVarP(&syncAgentFlag, "agent", "a", "", "Sync only specific agent (pi, codex, claude)")
	syncCmd.Flags().StringVarP(&syncScopeFlag, "scope", "s", "auto", "Scope: global, local, or auto")
	syncCmd.Flags().BoolVarP(&skipAgentSyncFlag, "skip-agent-sync", "", false, "Skip agent synchronization step")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Step 1: Sync repositories
	fmt.Println("=== Syncing repositories ===")
	if err := runRepoUpdate(cmd, nil); err != nil {
		fmt.Printf("  ! Warning: repo sync failed: %v\n", err)
	}

	// Step 2: Sync installed skills (update to latest)
	fmt.Println()
	fmt.Println("=== Syncing installed skills ===")
	if err := runSkillUpdate(cmd, nil); err != nil {
		fmt.Printf("  ! Warning: skill sync failed: %v\n", err)
	}

	// Step 3: Sync across agents (install missing skills)
	if !skipAgentSyncFlag {
		fmt.Println()
		if err := runAgentSync(cmd, args); err != nil {
			return fmt.Errorf("agent sync failed: %w", err)
		}
	}

	return nil
}

// runAgentSync syncs skills across agents by installing missing skills.
func runAgentSync(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Agent skill synchronization ===")

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	agentsCfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents config: %w", err)
	}

	if len(agentsCfg.Agents) == 0 {
		fmt.Println("  No agents configured. Run 'skillforge setup detect' first.")
		return nil
	}

	// Build skill catalog from cached repos
	skillCatalog, err := buildSkillCatalog(cfg)
	if err != nil {
		return fmt.Errorf("building skill catalog: %w", err)
	}

	if len(skillCatalog) == 0 {
		fmt.Println("  No skills found in cached repositories.")
		return nil
	}

	fmt.Printf("  Found %d unique skills across repositories\n", len(skillCatalog))

	// Collect installed skills per agent+scope
	installedSkills := make(map[string]map[string]bool) // key: "agent/scope"
	for agentName, agent := range agentsCfg.Agents {
		if syncAgentFlag != "" && syncAgentFlag != agentName {
			continue
		}

		// Global scope
		if shouldSyncScope(agent.Global, syncScopeFlag, "global") && agent.Global != nil {
			key := fmt.Sprintf("%s/global", agentName)
			installedSkills[key] = collectSkillNames(agents.ExpandPath(agent.Global.Value))
		}

		// Local scope
		if shouldSyncScope(agent.Local, syncScopeFlag, "local") && agent.Local != nil {
			key := fmt.Sprintf("%s/local", agentName)
			installedSkills[key] = collectSkillNames(agents.ExpandPath(agent.Local.Value))
		}
	}

	if len(installedSkills) == 0 {
		fmt.Println("  No matching agent scopes to sync.")
		return nil
	}

	// Find and report/apply missing skills
	missingByTarget := findMissingSkills(installedSkills, skillCatalog)

	if len(missingByTarget) == 0 {
		fmt.Println("  All agents have the same skills installed.")
		return nil
	}

	// Report missing skills
	totalMissing := 0
	for target, skills := range missingByTarget {
		if len(skills) > 0 {
			fmt.Printf("\n  %s is missing %d skill(s):\n", target, len(skills))
			for _, skillName := range skills {
				fmt.Printf("    - %s\n", skillName)
				totalMissing++
			}
		}
	}

	// Dry-run mode
	if checkFlag || dryRunFlag {
		fmt.Printf("\n  [DRY-RUN] Would install %d missing skill(s)\n", totalMissing)
		return nil
	}

	// Apply: install missing skills
	fmt.Printf("\n  Installing %d missing skill(s)...\n", totalMissing)
	installed := 0
	failed := 0

	for target, skillNames := range missingByTarget {
		if len(skillNames) == 0 {
			continue
		}

		// Parse target to get agent and scope
		agentName, scope := parseTarget(target)

		// Get agent config
		agent, exists := agentsCfg.Agents[agentName]
		if !exists {
			continue
		}

		// Get install path
		var installPath string
		switch scope {
		case "global":
			if agent.Global != nil {
				installPath = agents.ExpandPath(agent.Global.Value)
			}
		case "local":
			if agent.Local != nil {
				installPath = agents.ExpandPath(agent.Local.Value)
			}
		}

		if installPath == "" {
			continue
		}

		// Install each missing skill
		for _, skillName := range skillNames {
			skillInfo, exists := skillCatalog[skillName]
			if !exists {
				continue
			}

			targetPath := filepath.Join(installPath, skillName)

			// Check if already exists
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("    ! %s already exists in %s, skipping\n", skillName, target)
				continue
			}

			fmt.Printf("    Installing %s to %s...\n", skillName, target)

			if err := installSkillToPath(skillName, skillInfo, targetPath, cfg); err != nil {
				fmt.Printf("      ! Failed to install %s: %v\n", skillName, err)
				failed++
			} else {
				fmt.Printf("      ✓ %s installed\n", skillName)
				installed++
			}
		}
	}

	fmt.Printf("\n  ✓ Agent sync complete: %d installed, %d failed\n", installed, failed)

	return nil
}

// SkillInfo holds information about a skill from the catalog.
type SkillInfo struct {
	Source    string
	Path      string
	Commit    string
	RepoName  string
}

// buildSkillCatalog builds a map of all available skills from cached repos.
func buildSkillCatalog(cfg *config.Config) (map[string]SkillInfo, error) {
	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))
	catalog := make(map[string]SkillInfo)

	for repoName, info := range cfg.Repos {
		if !cache.Exists(repoName) {
			continue
		}

		skills, err := repo.DiscoverSkills(cache.PathFor(repoName), info.URL)
		if err != nil {
			continue
		}

		commit, _ := cache.GetCommit(repoName)

		for _, skill := range skills {
			// Prefer skills from repos that have been synced (have commits)
			if _, exists := catalog[skill.Name]; exists && commit == "" {
				continue
			}

			catalog[skill.Name] = SkillInfo{
				Source:   info.URL,
				Path:     skill.Path,
				Commit:   commit,
				RepoName: repoName,
			}
		}
	}

	return catalog, nil
}

// collectSkillNames returns a set of skill names installed at a path.
func collectSkillNames(path string) map[string]bool {
	skills := make(map[string]bool)

	entries, err := os.ReadDir(path)
	if err != nil {
		if !os.IsNotExist(err) && verboseFlag {
			fmt.Printf("[DEBUG] Error reading directory %s: %v\n", path, err)
		}
		return skills
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skills[entry.Name()] = true
		}
	}

	return skills
}

// findMissingSkills finds skills missing from each agent/scope target.
func findMissingSkills(installedSkills map[string]map[string]bool, catalog map[string]SkillInfo) map[string][]string {
	missingByTarget := make(map[string][]string)

	// Get union of all skill names
	allSkills := make(map[string]bool)
	for _, skills := range installedSkills {
		for name := range skills {
			allSkills[name] = true
		}
	}

	// For each target, find missing skills
	for target, installed := range installedSkills {
		var missing []string
		for skillName := range allSkills {
			if !installed[skillName] {
				// Only include if skill exists in catalog
				if _, exists := catalog[skillName]; exists {
					missing = append(missing, skillName)
				}
			}
		}
		missingByTarget[target] = missing
	}

	return missingByTarget
}

// shouldSyncScope determines if a scope should be synced.
func shouldSyncScope(path *agents.Path, scopeFlag, scopeValue string) bool {
	if path == nil {
		return false
	}
	switch scopeFlag {
	case "global":
		return scopeValue == "global"
	case "local":
		return scopeValue == "local"
	default: // auto
		return true
	}
}

// parseTarget parses "agent/scope" string.
func parseTarget(target string) (agent, scope string) {
	// Split by last "/"
	for i := len(target) - 1; i >= 0; i-- {
		if target[i] == '/' {
			return target[:i], target[i+1:]
		}
	}
	return target, ""
}

// installSkillToPath installs a skill to a specific path.
func installSkillToPath(skillName string, info SkillInfo, targetPath string, cfg *config.Config) error {
	// Re-discover the skill to get the full Skill object
	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))
	skills, err := repo.DiscoverSkills(cache.PathFor(info.RepoName), info.Source)
	if err != nil {
		return err
	}

	var targetSkill *grimoire.Skill
	for i := range skills {
		if skills[i].Name == skillName {
			targetSkill = &skills[i]
			break
		}
	}

	if targetSkill == nil {
		return fmt.Errorf("skill %q not found in catalog", skillName)
	}

	return repo.InstallSkill(*targetSkill, targetPath, info.Commit)
}
