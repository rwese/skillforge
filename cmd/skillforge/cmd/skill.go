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

Skills are linked from cached repositories to agent skill directories.`,
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillRemoveCmd)
	skillCmd.AddCommand(skillSearchCmd)
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
	_ = skillInstallCmd.RegisterFlagCompletionFunc("target", completeTargets)
	_ = skillListCmd.RegisterFlagCompletionFunc("target", completeTargets)
	_ = skillListCmd.RegisterFlagCompletionFunc("format", completeFormats)
	_ = skillRemoveCmd.RegisterFlagCompletionFunc("target", completeTargets)
}

var skillInstallCmd = &cobra.Command{
	Use:               "install [name]...",
	Short:             "Install one or more skills",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeAvailableSkills,
	RunE:              runSkillInstall,
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	// Determine installation scope
	isGlobalScope := scopeFlag == "global"

	// Determine paths to install to (default is local scope)
	installPaths, err := getInstallPaths(targetFlag, scopeFlag)
	if err != nil {
		return err
	}

	if len(installPaths) == 0 {
		PrintHint(HintNoTargets)
		return fmt.Errorf("no install paths found")
	}

	// Load local repos first, then global repos
	localLoader := config.NewLoader(config.ScopeLocal)
	localCfg, _ := localLoader.Load()
	localRepoNames := make(map[string]bool)
	if localCfg != nil {
		for repoName := range localCfg.Repos {
			localRepoNames[repoName] = true
		}
	}

	// Load global repos
	globalLoader := config.NewLoader(config.ScopeGlobal)
	globalCfg, err := globalLoader.Load()
	if err != nil {
		return err
	}

	// Merge: local repos first, then global (local takes precedence)
	allRepos := make(map[string]config.RepoInfo)
	for repoName, info := range globalCfg.Repos {
		allRepos[repoName] = info
	}
	if localCfg != nil {
		for repoName, info := range localCfg.Repos {
			allRepos[repoName] = info
		}
	}

	cache := repo.NewCache(config.ExpandPath(globalCfg.Cache.Path))

	// Process each skill
	var errors []error
	for _, skillName := range args {
		fmt.Printf("\n=== Installing %s ===\n", skillName)

		// Find the skill across all repos (local first, then global)
		var targetSkill *grimoire.Skill
		var skillRepoName string

		// Search local repos first
		if localCfg != nil {
			for repoName, info := range localCfg.Repos {
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
						skillRepoName = repoName
						break
					}
				}

				if targetSkill != nil {
					break
				}
			}
		}

		// If not found locally, search global repos
		if targetSkill == nil {
			for repoName, info := range globalCfg.Repos {
				// Skip if already in local repos
				if localRepoNames[repoName] {
					continue
				}
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
						skillRepoName = repoName
						break
					}
				}

				if targetSkill != nil {
					break
				}
			}
		}

		// Check if skill is from local repo and trying to install to global scope
		if targetSkill != nil && isGlobalScope && localRepoNames[skillRepoName] {
			err := fmt.Errorf("skill %q is from local repository %q and cannot be installed to global scope", skillName, skillRepoName)
			errors = append(errors, err)
			continue
		}

		if targetSkill == nil {
			err := fmt.Errorf("skill %q not found in any cached repository", skillName)
			PrintHint(HintRepoNotCached)
			errors = append(errors, err)
			continue
		}

		// Dry-run mode
		if dryRunFlag {
			fmt.Printf("[DRY-RUN] Would link %q to %d path(s): %v\n", skillName, len(installPaths), installPaths)
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

			fmt.Printf("Linking %s to %s...\n", skillName, ip.Label)

			if err := repo.LinkSkill(*targetSkill, targetPath); err != nil {
				errors = append(errors, fmt.Errorf("linking to %s: %w", ip.Label, err))
				continue
			}
			fmt.Printf("  ✓ %s linked to %s\n", skillName, ip.Label)
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
// Default is local scope. Use -s global for global targets, -s local for local targets.
func getInstallPaths(targetName, scope string) ([]InstallPath, error) {
	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return nil, err
	}

	localCfg, err := loadConfigScope(config.ScopeLocal)
	localConfigExists := config.DetectLocalPath() != ""

	var paths []InstallPath

	// Determine which scopes to include (default is local)
	includeGlobal := scope == "global"
	includeLocal := scope == "local" || scope == ""

	if targetName != "" {
		// Specific target - check if it matches the requested scope
		found := false

		// Check global targets
		if includeGlobal {
			if target, ok := globalCfg.Targets[targetName]; ok && target.Enabled {
				paths = append(paths, InstallPath{
					Path:  config.ExpandPath(target.GlobalPath),
					Label: fmt.Sprintf("%s (global)", targetName),
				})
				found = true
			}
		}

		// Check local targets
		if includeLocal && localConfigExists && localCfg != nil {
			if target, ok := localCfg.Targets[targetName]; ok && target.Enabled {
				paths = append(paths, InstallPath{
					Path:  config.ExpandPath(target.LocalPath),
					Label: fmt.Sprintf("%s (local)", targetName),
				})
				found = true
			}
		}

		if !found {
			return nil, fmt.Errorf("target %q not found or not enabled for scope %q", targetName, scope)
		}
	} else {
		// All enabled targets filtered by scope
		if includeGlobal {
			for name, target := range globalCfg.Targets {
				if !target.Enabled {
					continue
				}
				paths = append(paths, InstallPath{
					Path:  config.ExpandPath(target.GlobalPath),
					Label: fmt.Sprintf("%s (global)", name),
				})
			}
		}

		if includeLocal {
			if localConfigExists && localCfg != nil && len(localCfg.Targets) > 0 {
				// Use targets from local config
				for name, target := range localCfg.Targets {
					if !target.Enabled || target.LocalPath == "" {
						continue
					}
					paths = append(paths, InstallPath{
						Path:  config.ExpandPath(target.LocalPath),
						Label: fmt.Sprintf("%s (local)", name),
					})
				}
			} else {
				// Fall back to global targets that have LocalPath defined
				for name, target := range globalCfg.Targets {
					if !target.Enabled || target.LocalPath == "" {
						continue
					}
					paths = append(paths, InstallPath{
						Path:  config.ExpandPath(target.LocalPath),
						Label: fmt.Sprintf("%s (local)", name),
					})
				}
			}
		}
	}

	return paths, nil
}

// shouldUseScope returns true if the given scope should be used.
// Empty scopeFlag means all scopes.
func shouldUseScope(scopeFlag, scopeValue string) bool {
	switch scopeFlag {
	case "global":
		return scopeValue == "global"
	case "local":
		return scopeValue == "local"
	case "":
		return true // All scopes when no flag specified
	default:
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
	resolverLocalCfg := localCfg
	if !localConfigExists {
		resolverLocalCfg = nil
	}
	repoResolver := newSkillListRepoResolver(globalCfg, resolverLocalCfg)

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
			path := config.ExpandPath(target.GlobalPath)
			skills, err := repo.ListInstalledSkills(path)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("listing skills in %s (global): %w", targetName, err)
				}
			}
			for _, skill := range skills {
				globalSkills = append(globalSkills, SkillOutput{
					Name:   skill.Name,
					Target: fmt.Sprintf("%s/global", targetName),
					Repo:   repoResolver.Resolve(skill),
					Source: skill.Source,
				})
			}
		}
	}

	// Collect skills from local targets
	if shouldUseScope(scopeFlag, "local") {
		if localConfigExists && localCfg != nil && len(localCfg.Targets) > 0 {
			// Use targets from local config
			for targetName, target := range localCfg.Targets {
				if targetFlag != "" && targetFlag != targetName {
					continue
				}
				if !target.Enabled || target.LocalPath == "" {
					continue
				}
				path := config.ExpandPath(target.LocalPath)
				skills, err := repo.ListInstalledSkills(path)
				if err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("listing skills in %s (local): %w", targetName, err)
					}
				}
				for _, skill := range skills {
					localSkills = append(localSkills, SkillOutput{
						Name:   skill.Name,
						Target: fmt.Sprintf("%s/local", targetName),
						Repo:   repoResolver.Resolve(skill),
						Source: skill.Source,
					})
				}
			}
		} else {
			// Fall back to global targets that have LocalPath defined
			for targetName, target := range globalCfg.Targets {
				if targetFlag != "" && targetFlag != targetName {
					continue
				}
				if !target.Enabled || target.LocalPath == "" {
					continue
				}
				path := config.ExpandPath(target.LocalPath)
				skills, err := repo.ListInstalledSkills(path)
				if err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("listing skills in %s (local): %w", targetName, err)
					}
				}
				for _, skill := range skills {
					localSkills = append(localSkills, SkillOutput{
						Name:   skill.Name,
						Target: fmt.Sprintf("%s/local", targetName),
						Repo:   repoResolver.Resolve(skill),
						Source: skill.Source,
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

type skillListRepoResolver struct {
	cache      *repo.Cache
	repos      map[string]config.RepoInfo
	sourceName map[string]string
}

func newSkillListRepoResolver(globalCfg, localCfg *config.Config) skillListRepoResolver {
	repos := make(map[string]config.RepoInfo)
	sourceName := make(map[string]string)

	cachePath := ""
	if globalCfg != nil {
		cachePath = globalCfg.Cache.Path
		for name, info := range globalCfg.Repos {
			repos[name] = info
		}
	}
	if localCfg != nil {
		if localCfg.Cache.Path != "" {
			cachePath = localCfg.Cache.Path
		}
		for name, info := range localCfg.Repos {
			repos[name] = info
		}
	}
	for name, info := range repos {
		sourceName[info.URL] = name
	}

	return skillListRepoResolver{
		cache:      repo.NewCache(config.ExpandPath(cachePath)),
		repos:      repos,
		sourceName: sourceName,
	}
}

func (r skillListRepoResolver) Resolve(skill grimoire.Skill) string {
	if name := r.sourceName[skill.Source]; name != "" {
		return name
	}

	if skill.Path == "" {
		return ""
	}

	skillPath, err := filepath.Abs(skill.Path)
	if err != nil {
		return ""
	}
	skillPath = filepath.Clean(skillPath)

	for repoName := range r.repos {
		repoPath, err := filepath.Abs(r.cache.PathFor(repoName))
		if err != nil {
			continue
		}
		repoPath = filepath.Clean(repoPath)
		rel, err := filepath.Rel(repoPath, skillPath)
		if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return repoName
		}
	}

	return ""
}

var skillRemoveCmd = &cobra.Command{
	Use:               "remove [name]",
	Short:             "Remove a skill",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeInstalledSkills,
	RunE:              runSkillRemove,
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

	// Determine which scopes to include
	includeGlobal := scope == "global" || scope == ""
	includeLocal := scope == "local" || scope == ""

	if targetName != "" {
		// Specific target
		if includeGlobal {
			if target, ok := globalCfg.Targets[targetName]; ok && target.Enabled {
				paths = append(paths, RemovePath{
					Path:  config.ExpandPath(target.GlobalPath),
					Label: fmt.Sprintf("%s (global)", targetName),
				})
			}
		}

		if includeLocal && localConfigExists && localCfg != nil {
			if target, ok := localCfg.Targets[targetName]; ok && target.Enabled {
				paths = append(paths, RemovePath{
					Path:  config.ExpandPath(target.LocalPath),
					Label: fmt.Sprintf("%s (local)", targetName),
				})
			}
		}
	} else {
		// All enabled targets filtered by scope
		if includeGlobal {
			for name, target := range globalCfg.Targets {
				if !target.Enabled {
					continue
				}
				paths = append(paths, RemovePath{
					Path:  config.ExpandPath(target.GlobalPath),
					Label: fmt.Sprintf("%s (global)", name),
				})
			}
		}

		if includeLocal {
			if localConfigExists && localCfg != nil && len(localCfg.Targets) > 0 {
				// Use targets from local config
				for name, target := range localCfg.Targets {
					if !target.Enabled || target.LocalPath == "" {
						continue
					}
					paths = append(paths, RemovePath{
						Path:  config.ExpandPath(target.LocalPath),
						Label: fmt.Sprintf("%s (local)", name),
					})
				}
			} else {
				// Fall back to global targets that have LocalPath defined
				for name, target := range globalCfg.Targets {
					if !target.Enabled || target.LocalPath == "" {
						continue
					}
					paths = append(paths, RemovePath{
						Path:  config.ExpandPath(target.LocalPath),
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
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: noFileCompletion,
	RunE:              runSkillSearch,
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

	// Track local repos for display
	localRepoNames := make(map[string]bool)
	for repoName := range cfg.Repos {
		localRepoNames[repoName] = true
	}

	// Collect all skills from all repos (local first, then global)
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

	// Build repo name to URL mapping for display
	repoDisplayNames := make(map[string]string)
	for repoName, info := range cfg.Repos {
		repoDisplayNames[info.URL] = repoName
	}

	if parseFormat(formatFlag) == formatJSON {
		skillOutputs := make([]SkillOutput, len(results))
		for i, s := range results {
			// Format source as <repo>/<skill>
			repoName := repoDisplayNames[s.Source]
			if repoName == "" {
				repoName = s.Source
			}
			source := fmt.Sprintf("%s/%s", repoName, s.Name)
			skillOutputs[i] = SkillOutput{
				Name:        s.Name,
				Description: s.Description,
				Source:      source,
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
		// Format source as <repo>/<skill>
		repoName := repoDisplayNames[skill.Source]
		if repoName == "" {
			repoName = skill.Source
		}
		source := fmt.Sprintf("%s/%s", repoName, skill.Name)
		fmt.Printf("  %d. %s  %s  %s\n", i+1, skill.Name, desc, source)
	}

	return nil
}
