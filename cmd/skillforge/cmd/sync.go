package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/rwese/skillforge/pkg/grimoire"
	"github.com/spf13/cobra"
)

type syncRepoCache interface {
	Exists(name string) bool
	Clone(url, branch string) error
	Fetch(name string) error
	Pull(name string) error
	GetCommit(name string) (string, error)
	GetRemoteCommit(name, branch string) (string, error)
	GetIncomingLog(name, branch string) (string, error)
	GetIncomingNameStatus(name, branch string) (string, error)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories and install skills",
	Long: `Sync repositories and install missing skills.

This command performs up to two operations:
1. Checks global targets for missing skills
2. With fix flags, updates cached repositories and installs missing skills as symlinks

Skills are always linked to the latest version from cached repositories.
By default, sync is read-only. Use --fix-sync-repos to update cached repositories,
--fix-outofsync-agents to link missing skills, or --fix-all for both.`,
	RunE: runSync,
}

var (
	syncAgentFlag              string
	syncFixOutofsyncAgentsFlag bool
	syncFixReposFlag           bool
	syncFixAllFlag             bool
	syncDiffFlag               bool
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&syncFixOutofsyncAgentsFlag, "fix-outofsync-agents", false, "Install missing skill links across configured agents")
	syncCmd.Flags().BoolVar(&syncFixReposFlag, "fix-sync-repos", false, "Update cached repositories after checking remote changes")
	syncCmd.Flags().BoolVar(&syncFixAllFlag, "fix-all", false, "Apply repository updates and missing agent skill links")
	syncCmd.Flags().BoolVar(&syncDiffFlag, "diff", false, "Show incoming remote commit and file changes")
	syncCmd.Flags().StringVarP(&syncAgentFlag, "agent", "a", "", "Sync only specific agent (pi, codex, claude)")
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

	cache := repo.NewCache(config.ExpandPath(cacheConfig.Path))
	fixRepos := syncFixReposFlag || syncFixAllFlag
	fixAgents := syncFixOutofsyncAgentsFlag || syncFixAllFlag
	if fixRepos {
		fmt.Println("=== Syncing repositories ===")
	} else {
		fmt.Println("=== Checking repositories ===")
	}
	syncRepositories(cache, allRepos, fixRepos, syncDiffFlag, os.Stdout)

	// Step 2: Sync across agents (install missing skills as symlinks when requested).
	fmt.Println()
	if err := runAgentSync(cmd, args, fixAgents); err != nil {
		return fmt.Errorf("agent sync failed: %w", err)
	}

	return nil
}

func syncRepositories(cache syncRepoCache, repos map[string]config.RepoInfo, fix, showDiff bool, out io.Writer) {
	if len(repos) == 0 {
		fmt.Fprintln(out, "  No repositories configured. Run 'skillforge repo add <url>' to add one.")
		return
	}

	names := make([]string, 0, len(repos))
	for name := range repos {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		info := repos[name]
		branch := info.Branch
		if branch == "" {
			branch = "main"
		}

		if !cache.Exists(name) {
			if !fix {
				fmt.Fprintf(out, "  ! %s is not cached. Use --fix-sync-repos to clone it.\n", name)
				continue
			}
			if err := cache.Clone(info.URL, branch); err != nil {
				fmt.Fprintf(out, "  ! Failed to clone %s: %v\n", name, err)
				continue
			}
			fmt.Fprintf(out, "  ✓ Cloned %s\n", name)
			continue
		}

		localCommit, localErr := cache.GetCommit(name)
		if err := cache.Fetch(name); err != nil {
			fmt.Fprintf(out, "  ! Failed to fetch %s: %v\n", name, err)
			continue
		}
		remoteCommit, remoteErr := cache.GetRemoteCommit(name, branch)
		if localErr != nil {
			fmt.Fprintf(out, "  ! Failed to read local commit for %s: %v\n", name, localErr)
			continue
		}
		if remoteErr != nil {
			fmt.Fprintf(out, "  ! Failed to read remote commit for %s: %v\n", name, remoteErr)
			continue
		}

		if localCommit == remoteCommit {
			fmt.Fprintf(out, "  ✓ %s up to date\n", name)
			continue
		}

		fmt.Fprintf(out, "  ↻ %s has remote changes: %s -> %s\n", name, shortCommit(localCommit), shortCommit(remoteCommit))
		if showDiff {
			printIncomingDiff(cache, name, branch, out)
		}
		if !fix {
			continue
		}

		if err := cache.Pull(name); err != nil {
			fmt.Fprintf(out, "  ! Failed to update %s: %v\n", name, err)
			continue
		}
		newCommit, err := cache.GetCommit(name)
		if err != nil {
			fmt.Fprintf(out, "  ✓ Updated %s\n", name)
			continue
		}
		fmt.Fprintf(out, "  ✓ Updated %s to %s\n", name, shortCommit(newCommit))
	}
}

func printIncomingDiff(cache syncRepoCache, name, branch string, out io.Writer) {
	logOutput, logErr := cache.GetIncomingLog(name, branch)
	nameStatusOutput, statusErr := cache.GetIncomingNameStatus(name, branch)

	if logErr != nil {
		fmt.Fprintf(out, "    ! Failed to read incoming commits: %v\n", logErr)
	} else if logOutput != "" {
		fmt.Fprintln(out, "    Commits:")
		for _, line := range splitNonEmptyLines(logOutput) {
			fmt.Fprintf(out, "      %s\n", line)
		}
	} else {
		fmt.Fprintln(out, "    Commits: none")
	}

	if statusErr != nil {
		fmt.Fprintf(out, "    ! Failed to read incoming file changes: %v\n", statusErr)
	} else if nameStatusOutput != "" {
		fmt.Fprintln(out, "    Files:")
		for _, line := range splitNonEmptyLines(nameStatusOutput) {
			fmt.Fprintf(out, "      %s\n", line)
		}
	} else {
		fmt.Fprintln(out, "    Files: none")
	}
}

func splitNonEmptyLines(text string) []string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

func rejectSyncScopeFlag(cmd *cobra.Command) error {
	if scopeFlag != "" && scopeFlag != "local" {
		return fmt.Errorf("sync does not support --scope")
	}
	return nil
}

// runAgentSync syncs skills across targets by installing missing skills.
func runAgentSync(cmd *cobra.Command, args []string, fixAgents bool) error {
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
	if !fixAgents || dryRunFlag {
		fmt.Printf("\n  [CHECK] Would link %d global skill(s). Use --fix-outofsync-agents to apply.\n", totalGlobalMissing)
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
