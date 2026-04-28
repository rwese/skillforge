package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rwese/skillforge-ng/internal/agents"
	"github.com/rwese/skillforge-ng/internal/config"
	"github.com/rwese/skillforge-ng/internal/repo"
	"github.com/rwese/skillforge-ng/internal/search"
	"github.com/rwese/skillforge-ng/pkg/grimoire"
	"github.com/spf13/cobra"
)

// skillCmd represents the skill command group.
var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage installed skills",
	Long:  `Manage skills installed to targets.

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
	scopeFlag  string // "global", "local", or "auto"
)

func init() {
	skillInstallCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Install to specific agent (pi, codex, claude)")
	skillInstallCmd.Flags().StringVarP(&scopeFlag, "scope", "s", "auto", "Scope: global, local, or auto")
	skillListCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Filter by agent")
	skillListCmd.Flags().StringVarP(&scopeFlag, "scope", "s", "auto", "Scope: global, local, or auto")
	skillListCmd.Flags().StringVarP(&formatFlag, "format", "f", "text", "Output format: text, json")
	skillRemoveCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Remove from specific agent")
	skillRemoveCmd.Flags().StringVarP(&scopeFlag, "scope", "s", "auto", "Scope: global, local, or auto")
	skillUpdateCmd.Flags().BoolVarP(&checkFlag, "check", "c", false, "Check for updates without applying")
	skillUpdateCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Update skills for specific agent")
	skillUpdateCmd.Flags().StringVarP(&scopeFlag, "scope", "s", "auto", "Scope: global, local, or auto")
}

var skillInstallCmd = &cobra.Command{
	Use:   "install [name]",
	Short: "Install a skill",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillInstall,
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	skillName := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))

	// Find the skill across all repos
	var targetSkill *grimoire.Skill
	var commit string

	for repoName, info := range cfg.Repos {
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
		PrintHint(HintRepoNotCached)
		return fmt.Errorf("skill %q not found in any cached repository", skillName)
	}

	// Determine paths to install to
	installPaths, err := getInstallPaths(agentFlag, scopeFlag)
	if err != nil {
		return err
	}

	if len(installPaths) == 0 {
		err := fmt.Errorf("no install paths found")
		PrintHint(HintNoTargets)
		return err
	}

	// Dry-run mode
	if dryRunFlag {
		fmt.Printf("[DRY-RUN] Would install %q to %d path(s): %v\n", skillName, len(installPaths), installPaths)
		return nil
	}

	// Install to each path
	for _, ip := range installPaths {
		if err := ensureTargetDir(ip.Path); err != nil {
			return fmt.Errorf("creating target directory: %w", err)
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
			return fmt.Errorf("installing to %s: %w", ip.Label, installErr)
		}
		fmt.Printf("  ✓ %s installed to %s\n", skillName, ip.Label)
	}

	return nil
}

// InstallPath represents a path to install a skill to.
type InstallPath struct {
	Path  string
	Label string // e.g., "pi (global)", "codex (local)"
}

// getInstallPaths returns paths to install skills to based on agent and scope flags.
func getInstallPaths(agentName, scope string) ([]InstallPath, error) {
	agentsCfg, err := agents.LoadAgents()
	if err != nil {
		return nil, fmt.Errorf("loading agents config: %w", err)
	}

	var paths []InstallPath

	if agentName != "" {
		// Specific agent
		agent, exists := agentsCfg.Agents[agentName]
		if !exists {
			return nil, fmt.Errorf("agent %q not configured. Run 'skillforge setup list' to see configured agents", agentName)
		}
		paths = appendAgentPaths(paths, agentName, agent, scope)
	} else {
		// All agents
		for name, agent := range agentsCfg.Agents {
			paths = appendAgentPaths(paths, name, agent, scope)
		}
	}

	return paths, nil
}

// appendAgentPaths adds paths from an agent based on scope.
func appendAgentPaths(paths []InstallPath, agentName string, agent agents.Agent, scope string) []InstallPath {
	switch scope {
	case "global":
		if agent.Global != nil {
			paths = append(paths, InstallPath{
				Path:  agents.ExpandPath(agent.Global.Value),
				Label: fmt.Sprintf("%s (global)", agentName),
			})
		}
	case "local":
		if agent.Local != nil {
			paths = append(paths, InstallPath{
				Path:  agents.ExpandPath(agent.Local.Value),
				Label: fmt.Sprintf("%s (local)", agentName),
			})
		}
	case "auto":
		fallthrough
	default:
		// In project = local first, then global; outside = global only
		if isInProject() {
			if agent.Local != nil {
				paths = append(paths, InstallPath{
					Path:  agents.ExpandPath(agent.Local.Value),
					Label: fmt.Sprintf("%s (local)", agentName),
				})
			}
			if agent.Global != nil {
				paths = append(paths, InstallPath{
					Path:  agents.ExpandPath(agent.Global.Value),
					Label: fmt.Sprintf("%s (global)", agentName),
				})
			}
		} else {
			if agent.Global != nil {
				paths = append(paths, InstallPath{
					Path:  agents.ExpandPath(agent.Global.Value),
					Label: fmt.Sprintf("%s (global)", agentName),
				})
			}
		}
	}
	return paths
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	RunE:  runSkillList,
}

func runSkillList(cmd *cobra.Command, args []string) error {
	agentsCfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents config: %w", err)
	}

	var localSkills []SkillOutput
	var globalSkills []SkillOutput

	// Collect skills grouped by scope: local first, then global
	for agentName, agent := range agentsCfg.Agents {
		if agentFlag != "" && agentFlag != agentName {
			continue
		}

		// List local skills
		if shouldListScope(agent.Local, scopeFlag, "local") {
			if agent.Local != nil {
				path := agents.ExpandPath(agent.Local.Value)
				skills, err := repo.ListInstalledSkills(path)
				if err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("listing skills in %s (local): %w", agentName, err)
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
						Target: fmt.Sprintf("%s/local", agentName),
						Source: skill.Grimoire.Source,
					})
				}
			}
		}

		// List global skills
		if shouldListScope(agent.Global, scopeFlag, "global") {
			if agent.Global != nil {
				path := agents.ExpandPath(agent.Global.Value)
				skills, err := repo.ListInstalledSkills(path)
				if err != nil {
					if !os.IsNotExist(err) {
						return fmt.Errorf("listing skills in %s (global): %w", agentName, err)
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
						Target: fmt.Sprintf("%s/global", agentName),
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
		if len(agentsCfg.Agents) == 0 {
			fmt.Println("Run 'skillforge setup detect' to configure agents.")
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

// shouldListScope determines if a scope should be listed based on scope flag.
func shouldListScope(path *agents.Path, scopeFlag, scopeValue string) bool {
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

var skillRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a skill",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillRemove,
}

func runSkillRemove(cmd *cobra.Command, args []string) error {
	skillName := args[0]

	agentsCfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents config: %w", err)
	}

	// Determine paths to remove from
	allPaths := getRemovePaths(agentsCfg, agentFlag, scopeFlag)

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

func getRemovePaths(cfg *agents.AgentsConfig, agentName, scope string) []RemovePath {
	var paths []RemovePath

	if agentName != "" {
		agent, exists := cfg.Agents[agentName]
		if !exists {
			return paths
		}
		paths = appendRemovePaths(paths, agentName, agent, scope)
	} else {
		for name, agent := range cfg.Agents {
			paths = appendRemovePaths(paths, name, agent, scope)
		}
	}

	return paths
}

func appendRemovePaths(paths []RemovePath, agentName string, agent agents.Agent, scope string) []RemovePath {
	switch scope {
	case "global":
		if agent.Global != nil {
			paths = append(paths, RemovePath{
				Path:  agents.ExpandPath(agent.Global.Value),
				Label: fmt.Sprintf("%s (global)", agentName),
			})
		}
	case "local":
		if agent.Local != nil {
			paths = append(paths, RemovePath{
				Path:  agents.ExpandPath(agent.Local.Value),
				Label: fmt.Sprintf("%s (local)", agentName),
			})
		}
	case "auto":
		fallthrough
	default:
		// Remove from wherever it exists
		if agent.Global != nil {
			paths = append(paths, RemovePath{
				Path:  agents.ExpandPath(agent.Global.Value),
				Label: fmt.Sprintf("%s (global)", agentName),
			})
		}
		if agent.Local != nil {
			paths = append(paths, RemovePath{
				Path:  agents.ExpandPath(agent.Local.Value),
				Label: fmt.Sprintf("%s (local)", agentName),
			})
		}
	}
	return paths
}

var skillSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for skills",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillSearch,
}

func runSkillSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
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

	agentsCfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents config: %w", err)
	}

	for agentName, agent := range agentsCfg.Agents {
		if agentFlag != "" && agentFlag != agentName {
			continue
		}

		// Check global
		if shouldListScope(agent.Global, scopeFlag, "global") {
			if agent.Global != nil {
				path := agents.ExpandPath(agent.Global.Value)
				updates, err := checkSkillUpdates(agentName, "global", path, cfg, cache)
				if err != nil {
					fmt.Printf("  ! Error checking %s (global): %v\n", agentName, err)
				}
				hasUpdates = hasUpdates || updates
			}
		}

		// Check local
		if shouldListScope(agent.Local, scopeFlag, "local") {
			if agent.Local != nil {
				path := agents.ExpandPath(agent.Local.Value)
				updates, err := checkSkillUpdates(agentName, "local", path, cfg, cache)
				if err != nil {
					fmt.Printf("  ! Error checking %s (local): %v\n", agentName, err)
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

func checkSkillUpdates(agentName, scope string, path string, cfg *config.Config, cache *repo.Cache) (bool, error) {
	skills, err := repo.ListInstalledSkills(path)
	if err != nil {
		return false, nil
	}

	if len(skills) == 0 {
		if verboseFlag {
			fmt.Printf("Checking %s (%s): no skills installed\n", agentName, scope)
		}
		return false, nil
	}

	hasUpdates := false
	label := fmt.Sprintf("%s (%s)", agentName, scope)

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
