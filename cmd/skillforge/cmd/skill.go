package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/rwese/skillforge/internal/search"
	"github.com/rwese/skillforge/pkg/grimoire"
	"github.com/spf13/cobra"
)

// skillCmd represents the skill command group.
var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage installed skills",
	Long: `Manage skills installed to targets.

Skills are copied from cached repositories to agent skill directories.`,
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillRemoveCmd)
	skillCmd.AddCommand(skillSearchCmd)
	skillCmd.AddCommand(skillUpdateCmd)
}

var (
	targetFlag string
	agentFlag  string
)

func init() {
	skillInstallCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Install to specific target")
	skillListCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Filter by target")
	skillListCmd.Flags().StringVarP(&formatFlag, "format", "f", "text", "Output format: text, json")
	skillRemoveCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Remove from specific target")
	skillUpdateCmd.Flags().BoolVarP(&checkFlag, "check", "c", false, "Check for updates without applying")
	skillUpdateCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Update skills for specific target")
}

var skillInstallCmd = &cobra.Command{
	Use:   "install [name]...",
	Short: "Install one or more skills",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSkillInstall,
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	// Load scoped config for installation paths (ignored but validates scope is valid)
	if _, err := loadConfig(); err != nil {
		return err
	}

	// Always load all repos (global + local) for skill discovery since cache is global
	allRepos, allCache, err := config.NewLoader(config.ScopeAuto).LoadAllRepos()
	if err != nil {
		return fmt.Errorf("loading repos: %w", err)
	}

	cache := repo.NewCache(config.ExpandPath(allCache.Path))

	// Determine paths to install to (uses scoped config)
	installPaths, err := getInstallPaths(targetFlag, scopeFlag)
	if err != nil {
		return err
	}

	if len(installPaths) == 0 {
		PrintHint(HintNoTargets)
		return fmt.Errorf("no install paths found")
	}

	// Process each skill
	var errors []error
	for _, skillName := range args {
		fmt.Printf("\n=== Installing %s ===\n", skillName)

		// Find the skill across all repos (use merged repos, not scoped)
		var targetSkill *grimoire.Skill
		var commit string

		for repoName, info := range allRepos {
			if !cache.Exists(repoName) {
				continue
			}

			skills, err := repo.DiscoverSkills(cache.PathFor(repoName), info.URL)
			if err != nil {
				continue
			}

			for i := range skills {
				if skills[i].Name == skillName {
					targetSkill = &skills[i]
					commit, _ = cache.GetCommit(repoName)
					break
				}
			}

			if targetSkill != nil {
				break
			}
		}

		if targetSkill == nil {
			err := fmt.Errorf("skill %q not found in any cached repository", skillName)
			PrintHint(HintRepoNotCached)
			errors = append(errors, err)
			continue
		}

		// Dry-run mode
		if dryRunFlag {
			fmt.Printf("[DRY-RUN] Would install %q to %d path(s): %v\n", skillName, len(installPaths), installPaths)
			continue
		}

		// Install to each path
		for _, ip := range installPaths {
			if err := ensureTargetDir(ip.Path); err != nil {
				errors = append(errors, fmt.Errorf("creating target directory for %s: %w", ip.Label, err))
				continue
			}
			targetPath := filepath.Join(ip.Path, skillName)

			// Check for conflicts
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("  ! %s already exists in %s, skipping\n", skillName, ip.Label)
				continue
			}

			fmt.Printf("Installing %s to %s...\n", skillName, ip.Label)

			// Use progress callback if verbose
			var installErr error
			if verboseFlag {
				installErr = repo.InstallSkillWithProgress(*targetSkill, targetPath, commit, func(src, dst string) {
					verbose("Copied %s -> %s", src, dst)
				})
			} else {
				installErr = repo.InstallSkill(*targetSkill, targetPath, commit)
			}

			if installErr != nil {
				errors = append(errors, fmt.Errorf("installing to %s: %w", ip.Label, installErr))
				continue
			}
			fmt.Printf("  ✓ %s installed to %s\n", skillName, ip.Label)
		}
	}

	if len(errors) > 0 {
		fmt.Println("\nErrors encountered:")
		for _, e := range errors {
			fmt.Printf("  - %v\n", e)
		}
		return fmt.Errorf("%d skill(s) failed to install", len(errors))
	}

	return nil
}

// InstallPath represents a path to install a skill to.
type InstallPath struct {
	Path  string
	Label string // e.g., "pi (global)", "pi (local)"
}

// getInstallPaths returns paths to install skills to based on target and scope flags.
func getInstallPaths(targetName, scope string) ([]InstallPath, error) {
	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return nil, err
	}

	localCfg, err := loadConfigScope(config.ScopeLocal)
	localConfigExists := config.DetectLocalPath() != ""

	var paths []InstallPath

	if targetName != "" {
		// Specific target - check both global and local
		found := false

		// Check global targets
		if target, ok := globalCfg.Targets[targetName]; ok && target.Enabled {
			if shouldUseScope(scope, "global") {
				paths = append(paths, InstallPath{
					Path:  config.ExpandPath(target.Path),
					Label: fmt.Sprintf("%s (global)", targetName),
				})
				found = true
			}
		}

		// Check local targets
		if localConfigExists && localCfg != nil {
			if target, ok := localCfg.Targets[targetName]; ok && target.Enabled {
				if shouldUseScope(scope, "local") {
					paths = append(paths, InstallPath{
						Path:  config.ExpandPath(target.Path),
						Label: fmt.Sprintf("%s (local)", targetName),
					})
					found = true
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("target %q not found or not enabled", targetName)
		}
	} else {
		// All enabled targets
		for name, target := range globalCfg.Targets {
			if !target.Enabled {
				continue
			}
			if shouldUseScope(scope, "global") {
				paths = append(paths, InstallPath{
					Path:  config.ExpandPath(target.Path),
					Label: fmt.Sprintf("%s (global)", name),
				})
			}
		}

		// Local targets
		if localConfigExists && localCfg != nil {
			for name, target := range localCfg.Targets {
				if !target.Enabled {
					continue
				}
				if shouldUseScope(scope, "local") {
					paths = append(paths, InstallPath{
						Path:  config.ExpandPath(target.Path),
						Label: fmt.Sprintf("%s (local)", name),
					})
				}
			}
		}
	}

	return paths, nil
}

// shouldUseScope returns true if the given scope should be used.
func shouldUseScope(scopeFlag, scopeValue string) bool {
	switch scopeFlag {
	case "global":
		return scopeValue == "global"
	case "local":
		return scopeValue == "local"
	default: // auto
		return true
	}
}

// loadConfigScope loads config for a specific scope.
func loadConfigScope(scope config.Scope) (*config.Config, error) {
	loader := config.NewLoader(scope)
	return loader.Load()
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	RunE:  runSkillList,
}

func runSkillList(cmd *cobra.Command, args []string) error {
	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return err
	}

	localCfg, err := loadConfigScope(config.ScopeLocal)
	localConfigExists := config.DetectLocalPath() != ""

	var localSkills []SkillOutput
	var globalSkills []SkillOutput

	// Collect skills from global targets
	for targetName, target := range globalCfg.Targets {
		if targetFlag != "" && targetFlag != targetName {
			continue
		}
		if !target.Enabled {
			continue
		}

		// List global skills
		if shouldUseScope(scopeFlag, "global") {
			path := config.ExpandPath(target.Path)
			skills, err := repo.ListInstalledSkills(path)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("listing skills in %s (global): %w", targetName, err)
				}
			}
			for _, skill := range skills {
				commit := skill.Grimoire.Commit
				if len(commit) > 7 {
					commit = commit[:7]
				}
				globalSkills = append(globalSkills, SkillOutput{
					Name:   skill.Name,
					Commit: commit,
					Target: fmt.Sprintf("%s/global", targetName),
					Source: skill.Grimoire.Source,
				})
			}
		}
	}

	// Collect skills from local targets
	if localConfigExists && localCfg != nil {
		for targetName, target := range localCfg.Targets {
			if targetFlag != "" && targetFlag != targetName {
				continue
			}
			if !target.Enabled {
				continue
			}

			// List local skills
			if shouldUseScope(scopeFlag, "local") {
				path := config.ExpandPath(target.Path)
				skills, err := repo.ListInstalledSkills(path)
				if err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("listing skills in %s (local): %w", targetName, err)
					}
				}
				for _, skill := range skills {
					commit := skill.Grimoire.Commit
					if len(commit) > 7 {
						commit = commit[:7]
					}
					localSkills = append(localSkills, SkillOutput{
						Name:   skill.Name,
						Commit: commit,
						Target: fmt.Sprintf("%s/local", targetName),
						Source: skill.Grimoire.Source,
					})
				}
			}
		}
	}

	// Combine: local first, then global
	allSkills := append(localSkills, globalSkills...)

	fmtmt := parseFormat(formatFlag)

	if fmtmt == formatJSON {
		return printJSON(allSkills)
	}

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		if len(globalCfg.Targets) == 0 {
			fmt.Println("Run 'skillforge target list' to see configured targets.")
		}
		return nil
	}

	if fmtmt == formatCompact {
		fmt.Println(formatSkillCompact(allSkills))
		return nil
	}

	// Default: table format
	fmt.Println(formatSkillTable(allSkills))
	return nil
}

var skillRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a skill",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillRemove,
}

func runSkillRemove(cmd *cobra.Command, args []string) error {
	skillName := args[0]

	// Get paths to remove from
	allPaths := getRemovePaths(targetFlag, scopeFlag)

	// Filter to only paths where skill actually exists
	var removePaths []RemovePath
	for _, rp := range allPaths {
		targetPath := filepath.Join(rp.Path, skillName)
		if _, err := os.Stat(targetPath); err == nil {
			removePaths = append(removePaths, rp)
		}
	}

	if len(removePaths) == 0 {
		PrintHint(HintSkillNotInstalled)
		return fmt.Errorf("skill %q not found in any location", skillName)
	}

	// Confirm if not using --yes
	if !yesFlag {
		fmt.Printf("Remove skill %q from %d location(s)? ", skillName, len(removePaths))
		if !confirm("") {
			return fmt.Errorf("cancelled")
		}
	}

	for _, rp := range removePaths {
		targetPath := filepath.Join(rp.Path, skillName)

		fmt.Printf("Removing %s from %s...\n", skillName, rp.Label)
		if err := repo.RemoveSkill(targetPath); err != nil {
			return fmt.Errorf("removing from %s: %w", rp.Label, err)
		}
		fmt.Printf("  ✓ %s removed from %s\n", skillName, rp.Label)
	}

	return nil
}

// RemovePath represents a path to remove a skill from.
type RemovePath struct {
	Path  string
	Label string
}

func getRemovePaths(targetName, scope string) []RemovePath {
	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return nil
	}

	localCfg, err := loadConfigScope(config.ScopeLocal)
	localConfigExists := config.DetectLocalPath() != ""

	var paths []RemovePath

	if targetName != "" {
		// Specific target
		if target, ok := globalCfg.Targets[targetName]; ok && target.Enabled {
			if shouldUseScope(scope, "global") {
				paths = append(paths, RemovePath{
					Path:  config.ExpandPath(target.Path),
					Label: fmt.Sprintf("%s (global)", targetName),
				})
			}
		}

		if localConfigExists && localCfg != nil {
			if target, ok := localCfg.Targets[targetName]; ok && target.Enabled {
				if shouldUseScope(scope, "local") {
					paths = append(paths, RemovePath{
						Path:  config.ExpandPath(target.Path),
						Label: fmt.Sprintf("%s (local)", targetName),
					})
				}
			}
		}
	} else {
		// All enabled targets
		for name, target := range globalCfg.Targets {
			if !target.Enabled {
				continue
			}
			if shouldUseScope(scope, "global") {
				paths = append(paths, RemovePath{
					Path:  config.ExpandPath(target.Path),
					Label: fmt.Sprintf("%s (global)", name),
				})
			}
		}

		if localConfigExists && localCfg != nil {
			for name, target := range localCfg.Targets {
				if !target.Enabled {
					continue
				}
				if shouldUseScope(scope, "local") {
					paths = append(paths, RemovePath{
						Path:  config.ExpandPath(target.Path),
						Label: fmt.Sprintf("%s (local)", name),
					})
				}
			}
		}
	}

	return paths
}

var skillSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for skills",
	Long: `Search for skills by name or description.

Supports multi-word queries. No flags required - just type your search terms.

Examples:
  skillforge skill search docker
  skillforge skill search "docker container"
  skillforge skill search postgres database`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSkillSearch,
}

func runSkillSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Verbose output: show config loading details
	verbose("Loading config with scope: %s", getScope())

	// Always load global config first to ensure global repos are always included
	globalLoader := config.NewLoader(config.ScopeGlobal)
	verbose("Global config path: %s", globalLoader.GlobalPath())
	cfg, err := globalLoader.Load()
	if err != nil {
		return err
	}

	// If not using global-only mode, merge local repos (local takes precedence)
	if scopeFlag != "global" {
		localLoader := config.NewLoader(config.ScopeLocal)
		localPath := config.DetectLocalPath()
		if localPath != "" {
			verbose("Local config path: %s", localPath)
		} else {
			verbose("No local config found")
		}
		localCfg, err := localLoader.Load()
		if err == nil {
			for k, v := range localCfg.Repos {
				cfg.Repos[k] = v
			}
		}
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))

	// Collect all skills from all repos
	skillSets := make(map[string][]grimoire.Skill)
	for repoName, info := range cfg.Repos {
		if !cache.Exists(repoName) {
			continue
		}

		skills, err := repo.DiscoverSkills(cache.PathFor(repoName), info.URL)
		if err != nil {
			continue
		}
		skillSets[repoName] = skills
	}

	results := search.SearchAll(skillSets, query)

	if parseFormat(formatFlag) == formatJSON {
		skillOutputs := make([]SkillOutput, len(results))
		for i, s := range results {
			skillOutputs[i] = SkillOutput{
				Name:        s.Name,
				Description: s.Description,
				Source:      s.Source,
			}
		}
		return printJSON(SearchResultOutput{
			Query:   query,
			Results: skillOutputs,
			Count:   len(results),
		})
	}

	if len(results) == 0 {
		fmt.Printf("No skills found matching %q\n", query)
		PrintHint(HintSearchNoResults)
		return nil
	}

	fmt.Printf("Search results for %q:\n", query)
	for i, skill := range results {
		desc := skill.Description
		if desc == "" {
			desc = "(no description)"
		}
		source := skill.Source
		fmt.Printf("  %d. %s  %s  %s\n", i+1, skill.Name, desc, source)
	}

	return nil
}

var skillUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for skill updates",
	RunE:  runSkillUpdate,
}

func runSkillUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))

	// First, update repos if not in check mode
	if !checkFlag {
		fmt.Println("Checking repository updates...")
		if err := runRepoUpdate(cmd, []string{}); err != nil {
			fmt.Printf("  ! Warning: repo update failed: %v\n", err)
		}
	}

	fmt.Println("Checking skill updates...")
	hasUpdates := false

	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return err
	}

	localCfg, err := loadConfigScope(config.ScopeLocal)
	localConfigExists := config.DetectLocalPath() != ""

	// Check global targets
	for targetName, target := range globalCfg.Targets {
		if targetFlag != "" && targetFlag != targetName {
			continue
		}
		if !target.Enabled {
			continue
		}

		if shouldUseScope(scopeFlag, "global") {
			path := config.ExpandPath(target.Path)
			updates, err := checkSkillUpdates(targetName, "global", path, cfg, cache)
			if err != nil {
				fmt.Printf("  ! Error checking %s (global): %v\n", targetName, err)
			}
			hasUpdates = hasUpdates || updates
		}
	}

	// Check local targets
	if localConfigExists && localCfg != nil {
		for targetName, target := range localCfg.Targets {
			if targetFlag != "" && targetFlag != targetName {
				continue
			}
			if !target.Enabled {
				continue
			}

			if shouldUseScope(scopeFlag, "local") {
				path := config.ExpandPath(target.Path)
				updates, err := checkSkillUpdates(targetName, "local", path, cfg, cache)
				if err != nil {
					fmt.Printf("  ! Error checking %s (local): %v\n", targetName, err)
				}
				hasUpdates = hasUpdates || updates
			}
		}
	}

	if !hasUpdates && checkFlag {
		fmt.Println("  All skills up to date.")
	}

	return nil
}

func checkSkillUpdates(targetName, scope string, path string, cfg *config.Config, cache *repo.Cache) (bool, error) {
	skills, err := repo.ListInstalledSkills(path)
	if err != nil {
		return false, nil
	}

	if len(skills) == 0 {
		if verboseFlag {
			fmt.Printf("Checking %s (%s): no skills installed\n", targetName, scope)
		}
		return false, nil
	}

	hasUpdates := false
	label := fmt.Sprintf("%s (%s)", targetName, scope)

	if verboseFlag {
		fmt.Printf("Checking %d skills in %s...\n", len(skills), label)
	}

	checkedCount := 0

	for _, skill := range skills {
		// Find the source repo
		var currentCommit string
		var newCommit string
		var repoName string

		for rn, info := range cfg.Repos {
			if info.URL == skill.Grimoire.Source {
				repoName = rn
				currentCommit = skill.Grimoire.Commit
				if cache.Exists(rn) {
					newCommit, _ = cache.GetCommit(rn)
				}
				break
			}
		}

		if repoName == "" {
			if verboseFlag {
				source := skill.Grimoire.Source
				if source == "" {
					source = "<unknown>"
				}
				fmt.Printf("  - %s: no cached repo for %s\n", skill.Name, source)
			}
			continue
		}

		checkedCount++

		if currentCommit != newCommit && newCommit != "" {
			hasUpdates = true
			if checkFlag {
				fmt.Printf("  ↻ %s has updates\n", skill.Name)
				if verboseFlag {
					fmt.Printf("     %s → %s\n", shortenCommit(currentCommit), shortenCommit(newCommit))
				}
			} else {
				// Re-install skill with new commit
				fmt.Printf("Updating %s in %s...\n", skill.Name, label)
				if verboseFlag {
					fmt.Printf("     %s → %s\n", shortenCommit(currentCommit), shortenCommit(newCommit))
				}

				skills, _ := repo.DiscoverSkills(cache.PathFor(repoName), cfg.Repos[repoName].URL)
				for _, s := range skills {
					if s.Name == skill.Name {
						targetPath := filepath.Join(path, skill.Name)
						if err := repo.InstallSkill(s, targetPath, newCommit); err != nil {
							fmt.Printf("  ! Failed to update %s: %v\n", skill.Name, err)
						} else {
							fmt.Printf("  ✓ %s updated\n", skill.Name)
						}
						break
					}
				}
			}
		} else {
			if checkFlag || verboseFlag {
				fmt.Printf("  ✓ %s is up to date\n", skill.Name)
			}
		}
	}

	if verboseFlag && !hasUpdates && !checkFlag {
		fmt.Printf("Checked %d skills: all up to date\n", checkedCount)
	}

	return hasUpdates, nil
}

// shortenCommit returns first 7 chars of commit or empty string.
func shortenCommit(commit string) string {
	if len(commit) >= 7 {
		return commit[:7]
	}
	return commit
}
