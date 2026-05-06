package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/rwese/skillforge/pkg/grimoire"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories and install skills",
	Long: `Sync repositories and install missing skills.

This command performs up to two operations:
1. Updates all cached repositories (repo update)
2. Syncs skills across agents (agent sync) - installs missing skills as symlinks

Skills are always linked to the latest version from cached repositories.
Use --check to see what would be done without making changes.
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
	syncCmd.Flags().StringVarP(&syncScopeFlag, "scope", "s", "local", "Scope: global or local")
	syncCmd.Flags().BoolVarP(&skipAgentSyncFlag, "skip-agent-sync", "", false, "Skip agent synchronization step")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Step 1: Sync repositories
	fmt.Println("=== Syncing repositories ===")
	if err := runRepoUpdate(cmd, nil); err != nil {
		fmt.Printf("  ! Warning: repo sync failed: %v\n", err)
	}

	// Step 2: Sync across agents (install missing skills as symlinks)
	if !skipAgentSyncFlag {
		fmt.Println()
		if err := runAgentSync(cmd, args); err != nil {
			return fmt.Errorf("agent sync failed: %w", err)
		}
	}

	return nil
}

// runAgentSync syncs skills across targets by installing missing skills.
func runAgentSync(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Agent skill synchronization ===")

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

	// Collect installed skills per target
	installedGlobal := make(map[string]map[string]bool) // key: "target"
	installedLocal := make(map[string]map[string]bool)

	// Process global targets
	for targetName, target := range globalCfg.Targets {
		if syncAgentFlag != "" && syncAgentFlag != targetName {
			continue
		}
		if !target.Enabled {
			continue
		}
		if syncScopeFlag == "local" {
			continue
		}

		path := config.ExpandPath(target.GlobalPath)
		if skills := collectSkillNames(path); skills != nil {
			installedGlobal[targetName] = skills
		}
	}

	// Process local targets (only if local config exists)
	if localConfigExists && localCfg != nil {
		for targetName, target := range localCfg.Targets {
			if syncAgentFlag != "" && syncAgentFlag != targetName {
				continue
			}
			if !target.Enabled {
				continue
			}
			if syncScopeFlag == "global" {
				continue
			}

			path := config.ExpandPath(target.LocalPath)
			if skills := collectSkillNames(path); skills != nil {
				installedLocal[targetName] = skills
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
		fmt.Printf("\n  [DRY-RUN] Would link %d global, %d local skill(s)\n", totalGlobalMissing, totalLocalMissing)
		return nil
	}

	// Apply: install missing skills (as symlinks)
	installed := 0
	failed := 0

	// Install global missing skills to global targets
	for targetName, skills := range missingGlobal {
		if len(skills) == 0 {
			continue
		}

		target, exists := globalCfg.Targets[targetName]
		if !exists {
			continue
		}

		installPath := config.ExpandPath(target.GlobalPath)
		linkMissingSkills(targetName, "global", skills, globalCatalog, installPath, &installed, &failed)
	}

	// Install local missing skills to local targets (only if local config exists)
	if localConfigExists && localCfg != nil {
		for targetName, skills := range missingLocal {
			if len(skills) == 0 {
				continue
			}

			target, exists := localCfg.Targets[targetName]
			if !exists {
				continue
			}

			installPath := config.ExpandPath(target.LocalPath)
			linkMissingSkills(targetName, "local", skills, localCatalog, installPath, &installed, &failed)
		}
	}

	fmt.Printf("\n  ✓ Agent sync complete: %d linked, %d failed\n", installed, failed)

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

// linkMissingSkills creates symlinks for missing skills in an agent's scope.
func linkMissingSkills(agentName, scope string, skillNames []string, catalog map[string]SkillInfo, installPath string, installed, failed *int) {
	for _, skillName := range skillNames {
		skillInfo, exists := catalog[skillName]
		if !exists {
			continue
		}

		targetPath := filepath.Join(installPath, skillName)

		// Check if already exists
		if _, err := os.Lstat(targetPath); err == nil {
			if verboseFlag {
				fmt.Printf("    ! %s already exists in %s/%s, skipping\n", skillName, agentName, scope)
			}
			continue
		}

		fmt.Printf("    Linking %s to %s/%s...\n", skillName, agentName, scope)

		if err := repo.LinkSkill(skillInfo.Skill, targetPath); err != nil {
			fmt.Printf("      ! Failed to link %s: %v\n", skillName, err)
			*failed++
		} else {
			fmt.Printf("      ✓ %s linked\n", skillName)
			*installed++
		}
	}
}

// SkillInfo holds information about a skill from the catalog.
type SkillInfo struct {
	grimoire.Skill
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

		for _, skill := range skills {
			// Prefer first encountered skill (repos are ordered by priority)
			if _, exists := catalog[skill.Name]; exists {
				continue
			}

			catalog[skill.Name] = SkillInfo{
				Skill:    skill,
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
		// Count both directories and symlinks as skills
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
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
