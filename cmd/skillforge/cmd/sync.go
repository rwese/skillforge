package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
1. Checks global targets for missing skills
2. With --fix, updates cached repositories and installs missing skills as symlinks

Skills are always linked to the latest version from cached repositories.
By default, sync is read-only. Use --fix to apply changes.
Use --skip-agent-sync to skip the agent synchronization step.`,
	RunE: runSync,
}

var (
	syncAgentFlag     string
	skipAgentSyncFlag bool
	syncFixFlag       bool
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&syncFixFlag, "fix", false, "Apply repository updates and missing skill links")
	syncCmd.Flags().StringVarP(&syncAgentFlag, "agent", "a", "", "Sync only specific agent (pi, codex, claude)")
	syncCmd.Flags().BoolVarP(&skipAgentSyncFlag, "skip-agent-sync", "", false, "Skip agent synchronization step")
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := rejectSyncScopeFlag(cmd); err != nil {
		return err
	}

	// Load repos from all configs for sync
	allRepos, cacheConfig, err := config.NewLoader(config.ScopeLocal).LoadAllRepos()
	if err != nil {
		return fmt.Errorf("loading repos: %w", err)
	}

	if syncFixFlag {
		fmt.Println("=== Syncing repositories ===")
		cache := repo.NewCache(config.ExpandPath(cacheConfig.Path))

		// Sync each repository
		for name, info := range allRepos {
			if !cache.Exists(name) {
				// Clone if not in cache
				spinner := NewSpinner(fmt.Sprintf("Cloning %s (branch: %s)...", info.URL, info.Branch))
				spinner.Start()
				if err := cache.Clone(info.URL, info.Branch); err != nil {
					spinner.Stop()
					fmt.Printf("  ! Failed to clone %s: %v\n", name, err)
					continue
				}
				spinner.Stop()
				fmt.Printf("  ✓ Cloned %s\n", name)
				continue
			}

			// Pull updates for existing repos
			spinner := NewSpinner(fmt.Sprintf("Updating %s ", name))
			spinner.Start()
			_ = cache.Pull(name) // Ignore pull errors
			spinner.Stop()
			fmt.Printf("  ✓ Updated %s\n", name)
		}

		if len(allRepos) == 0 {
			fmt.Println("  No repositories configured. Run 'skillforge repo add <url>' to add one.")
		}
	} else {
		fmt.Println("=== Checking repositories ===")
		fmt.Printf("  %d repositories configured. Use --fix to update cached repositories.\n", len(allRepos))
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

func rejectSyncScopeFlag(cmd *cobra.Command) error {
	if scopeFlag != "" && scopeFlag != "local" {
		return fmt.Errorf("sync does not support --scope")
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

	// Build catalog from global repos.
	globalCatalog, err := buildSkillCatalog(globalCfg)
	if err != nil {
		return fmt.Errorf("building global skill catalog: %w", err)
	}

	if len(globalCatalog) == 0 {
		fmt.Println("  No skills found in cached repositories.")
		return nil
	}

	// Report found skills
	if len(globalCatalog) > 0 {
		fmt.Printf("  Found %d global skills\n", len(globalCatalog))
	}

	// Collect installed skills per target
	installedGlobal := make(map[string]map[string]bool) // key: "target/global:name"
	globalInstallPaths := make(map[string]string)

	// Process global targets
	for targetName, target := range globalCfg.Targets {
		if syncAgentFlag != "" && syncAgentFlag != targetName {
			continue
		}
		if !target.Enabled {
			continue
		}
		globalPaths := resolvedGlobalPaths(target)
		globalNames := make([]string, 0, len(globalPaths))
		for globalName := range globalPaths {
			globalNames = append(globalNames, globalName)
		}
		sort.Strings(globalNames)
		for _, globalName := range globalNames {
			label := fmt.Sprintf("%s/global:%s", targetName, globalName)
			path := config.ExpandPath(globalPaths[globalName])
			if skills := collectSkillNames(path); skills != nil {
				installedGlobal[label] = skills
				globalInstallPaths[label] = path
			}
		}
	}

	if len(installedGlobal) == 0 {
		fmt.Println("  No matching global targets to sync.")
		return nil
	}

	// Find missing skills across global targets.
	missingGlobal := findMissingSkills(installedGlobal, globalCatalog)

	// Check if there's anything to sync
	hasGlobalMissing := false
	for _, skills := range missingGlobal {
		if len(skills) > 0 {
			hasGlobalMissing = true
			break
		}
	}
	if !hasGlobalMissing {
		fmt.Println("  All global targets have the same skills installed.")
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
	// Dry-run mode
	totalGlobalMissing := countMissing(missingGlobal)
	if !syncFixFlag || dryRunFlag {
		fmt.Printf("\n  [CHECK] Would link %d global skill(s). Use --fix to apply.\n", totalGlobalMissing)
		return nil
	}

	// Apply: install missing skills (as symlinks)
	installed := 0
	failed := 0

	// Install global missing skills to global targets
	for targetLabel, skills := range missingGlobal {
		if len(skills) == 0 {
			continue
		}

		installPath, exists := globalInstallPaths[targetLabel]
		if !exists {
			continue
		}

		linkMissingSkills(targetLabel, "", skills, globalCatalog, installPath, &installed, &failed)
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
		targetLabel := agentName
		if scope != "" {
			targetLabel = fmt.Sprintf("%s/%s", agentName, scope)
		}

		if _, err := os.Lstat(targetPath); err == nil {
			if verboseFlag {
				fmt.Printf("    ! %s already exists in %s, skipping\n", skillName, targetLabel)
			}
			continue
		}

		fmt.Printf("    Linking %s to %s...\n", skillName, targetLabel)

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
