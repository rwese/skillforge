package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwese/skillforge-ng/internal/config"
	"github.com/rwese/skillforge-ng/internal/repo"
	"github.com/rwese/skillforge-ng/pkg/grimoire"
	"github.com/spf13/cobra"
)

// repoCmd represents the repo command group.
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage skill repositories",
	Long:  `Manage cached skill repositories.

Repositories are cloned to a local cache and kept up-to-date.`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	repoCmd.AddCommand(repoUpdateCmd)
}

var branchFlag string

func init() {
	repoAddCmd.Flags().StringVarP(&branchFlag, "branch", "b", "main", "Branch to track")
}

var repoAddCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Add a repository to cache",
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoAdd,
}

func runRepoAdd(cmd *cobra.Command, args []string) error {
	url := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))
	if err := cache.Ensure(); err != nil {
		return err
	}

	// Determine repo name
	name := repoName(url)

	// Check if already cached
	if cache.Exists(name) {
		return fmt.Errorf("repository %q already cached (use 'repo update' to refresh)", name)
	}

	// Clone
	fmt.Printf("Cloning %s (branch: %s)...\n", url, branchFlag)
	if err := cache.Clone(url, branchFlag); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Get commit
	commit, err := cache.GetCommit(name)
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	// Update config
	cfg.Repos[name] = config.RepoInfo{
		URL:     url,
		Branch:  branchFlag,
		Updated: time.Now().Format(time.RFC3339),
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	// Discover skills
	skills, err := repo.DiscoverSkills(cache.PathFor(name), url)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	_ = commit // silence unused warning
	fmt.Printf("✓ Cached %s (%d skills)\n", name, len(skills))
	return nil
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached repositories",
	RunE:  runRepoList,
}

func runRepoList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))

	if len(cfg.Repos) == 0 {
		fmt.Println("No repositories cached.")
		return nil
	}

	fmt.Println("Cached Repositories:")
	for name, info := range cfg.Repos {
		skills, _ := repo.DiscoverSkills(cache.PathFor(name), info.URL)
		updated := "unknown"
		if t, err := cache.GetUpdated(name); err == nil {
			updated = formatDuration(time.Since(t))
		}
		fmt.Printf("  ✓ %s  (%d skills, updated %s)\n", name, len(skills), updated)
	}
	return nil
}

var repoRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a cached repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoRemove,
}

func runRepoRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	_, exists := cfg.Repos[name]
	if !exists {
		return fmt.Errorf("repository %q not found", name)
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))

	// Remove from cache
	if err := cache.Remove(name); err != nil {
		return fmt.Errorf("failed to remove cache: %w", err)
	}

	// Remove from config
	delete(cfg.Repos, name)
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("✓ Removed %s\n", name)
	return nil
}

var checkFlag bool

func init() {
	repoUpdateCmd.Flags().BoolVarP(&checkFlag, "check", "c", false, "Check for updates without applying")
}

var repoUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update cached repositories",
	RunE:  runRepoUpdate,
}

func runRepoUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cache := repo.NewCache(config.ExpandPath(cfg.Cache.Path))

	reposToUpdate := cfg.Repos
	if len(args) > 0 {
		name := args[0]
		if _, exists := cfg.Repos[name]; !exists {
			return fmt.Errorf("repository %q not found", name)
		}
		reposToUpdate = map[string]config.RepoInfo{name: cfg.Repos[name]}
	}

	for name, info := range reposToUpdate {
		if !cache.Exists(name) {
			fmt.Printf("  ! %s not in cache, skipping\n", name)
			continue
		}

		oldCommit, _ := cache.GetCommit(name)

		fmt.Printf("Updating %s...\n", name)
		if err := cache.Pull(name); err != nil {
			// Try fetch + reset if pull fails
			cache.Fetch(name)
		}

		newCommit, _ := cache.GetCommit(name)

		if oldCommit == newCommit {
			if checkFlag {
				fmt.Printf("  ✓ %s up to date\n", name)
			}
		} else {
			if checkFlag {
				fmt.Printf("  ↻ %s has updates\n", name)
			} else {
				info.Updated = time.Now().Format(time.RFC3339)
				cfg.Repos[name] = info
				fmt.Printf("  ✓ %s updated\n", name)
			}
		}

		_ = info.URL // silence unused warning
	}

	if !checkFlag {
		return saveConfig(cfg)
	}
	return nil
}

// repoName extracts the repository name from a URL.
func repoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	// Get last path component
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			return url[i+1:]
		}
	}
	return url
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// skillCachePath returns the path to install a skill in a target.
func skillCachePath(skill grimoire.Skill) string {
	return skill.Path
}

// skillTargetPath returns the target installation path for a skill.
func skillTargetPath(target config.Target, skillName string) string {
	return filepath.Join(config.ExpandPath(target.Path), skillName)
}
