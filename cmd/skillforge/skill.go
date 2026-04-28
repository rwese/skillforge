package main

import (
	"fmt"
	"os"
	"path/filepath"

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

var targetFlag string

func init() {
	skillInstallCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Target agent (default: all enabled)")
	skillListCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Filter by target")
	skillListCmd.Flags().StringVarP(&formatFlag, "format", "f", "text", "Output format: text, json")
	skillRemoveCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Target agent (default: all)")
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
		return fmt.Errorf("skill %q not found in any cached repository", skillName)
	}

	// Determine targets to install to
	targets := getTargetsToInstall(cfg, targetFlag)

	if len(targets) == 0 {
		return fmt.Errorf("no enabled targets found")
	}

	// Dry-run mode
	if dryRunFlag {
		fmt.Printf("[DRY-RUN] Would install %q to %d target(s): %v\n", skillName, len(targets), targets)
		return nil
	}

	// Install to each target
	for _, targetName := range targets {
		targetBasePath := config.ExpandPath(cfg.Targets[targetName].Path)
		if err := ensureTargetDir(targetBasePath); err != nil {
			return fmt.Errorf("creating target directory: %w", err)
		}
		targetPath := filepath.Join(targetBasePath, skillName)

		// Check for conflicts
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("  ! %s already exists in %s, skipping\n", skillName, targetName)
			continue
		}

		fmt.Printf("Installing %s to %s...\n", skillName, targetName)

		// Use progress callback if verbose
		installErr := error(nil)
		if verboseFlag {
			installErr = repo.InstallSkillWithProgress(*targetSkill, targetPath, commit, func(src, dst string) {
				verbose("Copied %s -> %s", src, dst)
			})
		} else {
			installErr = repo.InstallSkill(*targetSkill, targetPath, commit)
		}

		if installErr != nil {
			return fmt.Errorf("installing to %s: %w", targetName, installErr)
		}
		fmt.Printf("  ✓ %s installed to %s\n", skillName, targetName)
	}

	return nil
}

func getTargetsToInstall(cfg *config.Config, name string) []string {
	var targets []string

	if name != "" {
		// Specific target
		if target, exists := cfg.Targets[name]; exists {
			if target.Enabled {
				targets = append(targets, name)
			}
		}
		return targets
	}

	// All enabled targets
	for tname, target := range cfg.Targets {
		if target.Enabled {
			targets = append(targets, tname)
		}
	}

	return targets
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	RunE:  runSkillList,
}

func runSkillList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	var allSkills []SkillOutput

	for name, target := range cfg.Targets {
		if targetFlag != "" && targetFlag != name {
			continue
		}

		skills, err := repo.ListInstalledSkills(config.ExpandPath(target.Path))
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("listing skills in %s: %w", name, err)
			}
			continue
		}

		for _, skill := range skills {
			commit := skill.Grimoire.Commit
			if len(commit) > 7 {
				commit = commit[:7]
			}
			allSkills = append(allSkills, SkillOutput{
				Name:   skill.Name,
				Commit: commit,
				Target: name,
			})
		}
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(allSkills)
	}

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	fmt.Println("Installed Skills:")
	// Group by target
	targetSkills := make(map[string][]SkillOutput)
	for _, s := range allSkills {
		targetSkills[s.Target] = append(targetSkills[s.Target], s)
	}
	for target, skills := range targetSkills {
		fmt.Printf("  %s/\n", target)
		for i, skill := range skills {
			prefix := "    ├─ "
			if i == len(skills)-1 {
				prefix = "    └─ "
			}
			fmt.Printf("%s%s  @ %s\n", prefix, skill.Name, skill.Commit)
		}
	}
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
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Determine targets to remove from
	targets := getTargetsToRemove(cfg, targetFlag)

	// Confirm if not using --yes
	if !yesFlag && len(targets) > 0 {
		fmt.Printf("Remove skill %q from %d target(s)? ", skillName, len(targets))
		if !confirm("") {
			return fmt.Errorf("cancelled")
		}
	}

	removed := false
	for _, targetName := range targets {
		targetPath := filepath.Join(config.ExpandPath(cfg.Targets[targetName].Path), skillName)

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("Removing %s from %s...\n", skillName, targetName)
		if err := repo.RemoveSkill(targetPath); err != nil {
			return fmt.Errorf("removing from %s: %w", targetName, err)
		}
		fmt.Printf("  ✓ %s removed from %s\n", skillName, targetName)
		removed = true
	}

	if !removed {
		return fmt.Errorf("skill %q not found in any target", skillName)
	}

	return nil
}

func getTargetsToRemove(cfg *config.Config, name string) []string {
	var targets []string

	if name != "" {
		if _, exists := cfg.Targets[name]; exists {
			targets = append(targets, name)
		}
		return targets
	}

	// All targets
	for tname := range cfg.Targets {
		targets = append(targets, tname)
	}

	return targets
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

func init() {
	skillUpdateCmd.Flags().BoolVarP(&checkFlag, "check", "c", false, "Check for updates without applying")
	skillUpdateCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Filter by target")
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

	for targetName, target := range cfg.Targets {
		if targetFlag != "" && targetFlag != targetName {
			continue
		}

		skills, err := repo.ListInstalledSkills(config.ExpandPath(target.Path))
		if err != nil {
			continue
		}

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
				continue
			}

			if currentCommit != newCommit && newCommit != "" {
				hasUpdates = true
				if checkFlag {
					fmt.Printf("  ↻ %s in %s has updates\n", skill.Name, targetName)
				} else {
					// Re-install skill with new commit
					fmt.Printf("Updating %s in %s...\n", skill.Name, targetName)

					// Find skill in repo
					skills, _ := repo.DiscoverSkills(cache.PathFor(repoName), cfg.Repos[repoName].URL)
					for _, s := range skills {
						if s.Name == skill.Name {
							targetPath := filepath.Join(config.ExpandPath(target.Path), skill.Name)
							if err := repo.InstallSkill(s, targetPath, newCommit); err != nil {
								fmt.Printf("  ! Failed to update %s: %v\n", skill.Name, err)
							} else {
								fmt.Printf("  ✓ %s updated\n", skill.Name)
							}
							break
						}
					}
				}
			} else if checkFlag {
				fmt.Printf("  ✓ %s in %s up to date\n", skill.Name, targetName)
			}
		}
	}

	if !hasUpdates && checkFlag {
		fmt.Println("  All skills up to date.")
	}

	return nil
}
