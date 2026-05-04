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
	syncAgentFlag     string
	syncScopeFlag     string
	skipAgentSyncFlag bool
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

	agentsCfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents config: %w", err)
	}

	if len(agentsCfg.Agents) == 0 {
		fmt.Println("  No agents configured. Run 'skillforge setup detect' first.")
		return nil
	}

	// Load global config and repos
	globalLoader := config.NewLoader(config.ScopeGlobal)
	globalCfg, err := globalLoader.Load()
	if err != nil {
		return err
	}

	// Load local config and repos (only if local config exists)
	localLoader := config.NewLoader(config.ScopeLocal)
	localCfg, err := localLoader.Load()
	localConfigExists := config.DetectLocalPath() != ""

	// Build catalogs: global repos → global catalog, local repos → local catalog
	globalCatalog, err := buildSkillCatalog(globalCfg)
	if err != nil {
		return fmt.Errorf("building global skill catalog: %w", err)
	}

	var localCatalog map[string]SkillInfo
	if localConfigExists {
		localCatalog, err = buildSkillCatalog(localCfg)
		if err != nil {
			return fmt.Errorf("building local skill catalog: %w", err)
		}
	}

	if len(globalCatalog) == 0 && len(localCatalog) == 0 {
		fmt.Println("  No skills found in cached repositories.")
		return nil
	}

	// Report found skills
	if len(globalCatalog) > 0 {
		fmt.Printf("  Found %d global skills\n", len(globalCatalog))
	}
	if localCatalog != nil && len(localCatalog) > 0 {
		fmt.Printf("  Found %d local skills\n", len(localCatalog))
	}

	// Collect installed skills per agent+scope
	installedGlobal := make(map[string]map[string]bool) // key: "agent"
	installedLocal := make(map[string]map[string]bool)

	for agentName, agent := range agentsCfg.Agents {
		if syncAgentFlag != "" && syncAgentFlag != agentName {
			continue
		}

		// Global scope - only from global repos
		if shouldSyncScope(agent.Global, syncScopeFlag, "global") && agent.Global != nil {
			path := agents.ExpandPath(agent.Global.Value)
			if skills := collectSkillNames(path); skills != nil {
				installedGlobal[agentName] = skills
			}
		}

		// Local scope - only from local repos (only if local config exists)
		if localConfigExists && shouldSyncScope(agent.Local, syncScopeFlag, "local") && agent.Local != nil {
			path := agents.ExpandPath(agent.Local.Value)
			if skills := collectSkillNames(path); skills != nil {
				installedLocal[agentName] = skills
			}
		}
	}

	if len(installedGlobal) == 0 && len(installedLocal) == 0 {
		fmt.Println("  No matching agent scopes to sync.")
		return nil
	}

	// Find missing skills: global repos → global scope, local repos → local scope
	missingGlobal := findMissingSkills(installedGlobal, globalCatalog)
	missingLocal := findMissingSkills(installedLocal, localCatalog)

	// Check if there's anything to sync
	hasGlobalMissing := false
	for _, skills := range missingGlobal {
		if len(skills) > 0 {
			hasGlobalMissing = true
			break
		}
	}
	hasLocalMissing := false
	for _, skills := range missingLocal {
		if len(skills) > 0 {
			hasLocalMissing = true
			break
		}
	}

	if !hasGlobalMissing && !hasLocalMissing {
		fmt.Println("  All agents have the same skills installed.")
		return nil
	}

	// Report missing skills
	if hasGlobalMissing {
		fmt.Println("  --- Global scope ---")
		for target, skills := range missingGlobal {
			if len(skills) > 0 {
				fmt.Printf("    %s is missing %d skill(s):\n", target, len(skills))
				for _, skillName := range skills {
					fmt.Printf("      - %s\n", skillName)
				}
			}
		}
	}
	if hasLocalMissing {
		fmt.Println("  --- Local scope ---")
		for target, skills := range missingLocal {
			if len(skills) > 0 {
				fmt.Printf("    %s is missing %d skill(s):\n", target, len(skills))
				for _, skillName := range skills {
					fmt.Printf("      - %s\n", skillName)
				}
			}
		}
	}

	// Dry-run mode
	totalGlobalMissing := countMissing(missingGlobal)
	totalLocalMissing := countMissing(missingLocal)
	if checkFlag || dryRunFlag {
		fmt.Printf("\n  [DRY-RUN] Would install %d global, %d local skill(s)\n", totalGlobalMissing, totalLocalMissing)
		return nil
	}

	// Apply: install missing skills
	installed := 0
	failed := 0

	// Install global missing skills to global scope
	for agentName, skills := range missingGlobal {
		if len(skills) == 0 {
			continue
		}

		agent, exists := agentsCfg.Agents[agentName]
		if !exists || agent.Global == nil {
			continue
		}

		installPath := agents.ExpandPath(agent.Global.Value)
		installMissingSkills(agentName, "global", skills, globalCatalog, installPath, &installed, &failed)
	}

	// Install local missing skills to local scope (only if local config exists)
	if localConfigExists {
		for agentName, skills := range missingLocal {
			if len(skills) == 0 {
				continue
			}

			agent, exists := agentsCfg.Agents[agentName]
			if !exists || agent.Local == nil {
				continue
			}

			installPath := agents.ExpandPath(agent.Local.Value)
			installMissingSkills(agentName, "local", skills, localCatalog, installPath, &installed, &failed)
		}
	}

	fmt.Printf("\n  ✓ Agent sync complete: %d installed, %d failed\n", installed, failed)

	return nil
}

// countMissing counts total missing skills across all targets.
func countMissing(missingByTarget map[string][]string) int {
	total := 0
	for _, skills := range missingByTarget {
		total += len(skills)
	}
	return total
}

// installMissingSkills installs missing skills to an agent's scope.
func installMissingSkills(agentName, scope string, skillNames []string, catalog map[string]SkillInfo, installPath string, installed, failed *int) {
	for _, skillName := range skillNames {
		skillInfo, exists := catalog[skillName]
		if !exists {
			continue
		}

		targetPath := filepath.Join(installPath, skillName)

		// Check if already exists
		if _, err := os.Stat(targetPath); err == nil {
			if verboseFlag {
				fmt.Printf("    ! %s already exists in %s/%s, skipping\n", skillName, agentName, scope)
			}
			continue
		}

		fmt.Printf("    Installing %s to %s/%s...\n", skillName, agentName, scope)

		// Build a minimal config for the skill installer
		cfg := &config.Config{
			Cache: config.CacheConfig{Path: skillInfo.Source},
		}

		if err := installSkillToPath(skillName, skillInfo, targetPath, cfg); err != nil {
			fmt.Printf("      ! Failed to install %s: %v\n", skillName, err)
			*failed++
		} else {
			fmt.Printf("      ✓ %s installed\n", skillName)
			*installed++
		}
	}
}

// SkillInfo holds information about a skill from the catalog.
type SkillInfo struct {
	Source   string
	Path     string
	Commit   string
	RepoName string
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
// Returns nil if the directory doesn't exist.
func collectSkillNames(path string) map[string]bool {
	skills := make(map[string]bool)

	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist - return nil to indicate invalid target
		}
		if verboseFlag {
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
