package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/rwese/skillforge/pkg/grimoire"
	"github.com/spf13/cobra"
)

// repoCmd represents the repo command group.
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage skill repositories",
	Long: `Manage cached skill repositories.

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
var aliasFlag string

func init() {
	repoAddCmd.Flags().StringVarP(&branchFlag, "branch", "b", "main", "Branch to track")
	repoAddCmd.Flags().StringVarP(&aliasFlag, "alias", "a", "", "Alias for this repository")
	repoListCmd.Flags().StringVarP(&formatFlag, "format", "f", "text", "Output format: text, json")
	_ = repoListCmd.RegisterFlagCompletionFunc("format", completeFormats)
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

	scope := parseScope(scopeFlag)
	name := repoName(url)

	// Scopes have independent repo lists: adding a repository to the
	// local scope that is already registered in the global scope is
	// valid (and vice versa). Only a duplicate within the same scope
	// is an error.
	if _, exists := cfg.Repos[name]; exists {
		PrintHint(HintRepoExists)
		return fmt.Errorf("repository %q already exists in the %s scope (use 'repo update' to refresh)", name, scope)
	}

	cachePath, err := config.NewLoader(scope).EffectiveCachePath()
	if err != nil {
		return err
	}
	cache := repo.NewCache(config.ExpandPath(cachePath))
	if err := cache.Ensure(); err != nil {
		return err
	}

	// The on-disk cache is shared across scopes, so the repository may
	// already be cached (e.g. from the global scope). Reuse it instead
	// of cloning again; the config entry is still added to this scope.
	// A same-name cache entry cloned from a DIFFERENT URL is never
	// reused: silently linking the wrong content would be worse than
	// asking the user to resolve the name clash.
	if cache.Exists(name) {
		if origin, err := cache.Origin(name); err == nil && origin != url {
			// Point the hint at the scope that owns the conflicting
			// entry, so `repo remove` actually finds it.
			removeHint := fmt.Sprintf("repo remove %s", name)
			otherScope := config.ScopeGlobal
			if scope == config.ScopeGlobal {
				otherScope = config.ScopeLocal
			}
			if otherCfg, err := config.NewLoader(otherScope).Load(); err == nil {
				if _, ok := otherCfg.Repos[name]; ok {
					removeHint = fmt.Sprintf("repo remove %s -s %s", name, otherScope)
				}
			}
			return fmt.Errorf("repository %q is already cached from a different URL (%s); remove it first: %s", name, origin, removeHint)
		}
		fmt.Printf("✓ %s already cached\n", name)
	} else {
		// Clone with spinner
		spinner := NewSpinner(fmt.Sprintf("Cloning %s (branch: %s)...", url, branchFlag))
		spinner.Start()
		if err := cache.Clone(url, branchFlag); err != nil {
			spinner.Stop()
			return fmt.Errorf("failed to clone: %w", err)
		}
		spinner.Stop()
		fmt.Printf("✓ Cloned %s\n", url)
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
		Alias:   aliasFlag,
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	// Discover skills
	skills, err := repo.DiscoverSkills(cache.PathFor(name), url)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(commit) > 7 {
		commit = commit[:7]
	}
	fmt.Printf("✓ Cached %s at %s (%d skills)\n", name, commit, len(skills))
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

	cachePath, err := config.NewLoader(parseScope(scopeFlag)).EffectiveCachePath()
	if err != nil {
		return err
	}
	cache := repo.NewCache(config.ExpandPath(cachePath))

	// Collect repos
	var repos []RepoOutput
	for name, info := range cfg.Repos {
		skills, _ := repo.DiscoverSkills(cache.PathFor(name), info.URL)
		repos = append(repos, RepoOutput{
			Name:       name,
			Alias:      info.Alias,
			URL:        info.URL,
			Branch:     info.Branch,
			SkillCount: len(skills),
			Updated:    info.Updated,
		})
	}

	if len(repos) == 0 {
		if parseFormat(formatFlag) == formatJSON {
			return printJSON([]RepoOutput{})
		}
		fmt.Println("No repositories cached.")
		PrintHint(HintNoRepos)
		return nil
	}

	fmtmt := parseFormat(formatFlag)

	if fmtmt == formatJSON {
		return printJSON(repos)
	}

	if fmtmt == formatCompact {
		fmt.Println(formatRepoCompact(repos))
		return nil
	}

	// Default: table format
	fmt.Println(formatRepoTable(repos))
	return nil
}

var repoRemoveCmd = &cobra.Command{
	Use:               "remove [name]",
	Short:             "Remove a cached repository",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRepos,
	RunE:              runRepoRemove,
}

func runRepoRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	info, exists := cfg.Repos[name]
	if !exists {
		PrintHint(HintRepoNotFound)
		return fmt.Errorf("repository %q not found", name)
	}

	// Confirm if not using --yes
	if !yesFlag {
		fmt.Printf("Remove cached repository %q (%s)? ", name, info.URL)
		if !confirm("") {
			return fmt.Errorf("cancelled")
		}
	}

	cachePath, err := config.NewLoader(parseScope(scopeFlag)).EffectiveCachePath()
	if err != nil {
		return err
	}
	cache := repo.NewCache(config.ExpandPath(cachePath))

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
	Use:               "update [name]",
	Short:             "Update cached repositories",
	ValidArgsFunction: completeRepos,
	RunE:              runRepoUpdate,
}

func runRepoUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cachePath, err := config.NewLoader(parseScope(scopeFlag)).EffectiveCachePath()
	if err != nil {
		return err
	}
	cache := repo.NewCache(config.ExpandPath(cachePath))

	reposToUpdate := cfg.Repos
	if len(args) > 0 {
		name := args[0]
		if _, exists := cfg.Repos[name]; !exists {
			PrintHint(HintRepoNotFound)
			return fmt.Errorf("repository %q not found", name)
		}
		reposToUpdate = map[string]config.RepoInfo{name: cfg.Repos[name]}
	}

	for name, info := range reposToUpdate {
		if !cache.Exists(name) {
			// Clone if not in cache (sync mode)
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

		oldCommit, _ := cache.GetCommit(name)

		spinner := NewSpinner(fmt.Sprintf("Updating %s ", name))
		spinner.Start()
		if err := cache.Pull(name); err != nil {
			// Try fetch + reset if pull fails
			cache.Fetch(name)
		}
		spinner.Stop()

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
// scope determines whether to use GlobalPath or LocalPath.
func skillTargetPath(target config.Target, skillName string, scope string) string {
	path := target.GlobalPath
	if scope == "local" {
		path = target.LocalPath
	}
	return filepath.Join(config.ExpandPath(path), skillName)
}
