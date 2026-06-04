package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/spf13/cobra"
)

func noFileCompletion(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeFormats(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"text\tHuman-readable table output",
		"table\tHuman-readable table output",
		"compact\tCompact line output",
		"json\tJSON output",
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeScopes(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"local\tUse local project config",
		"global\tUse global user config",
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeShells(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"bash\tGenerate Bash completions",
		"zsh\tGenerate Zsh completions",
		"fish\tGenerate Fish completions",
		"powershell\tGenerate PowerShell completions",
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeRepos(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for name, info := range cfg.Repos {
		label := info.URL
		if info.Alias != "" {
			label = fmt.Sprintf("%s (%s)", info.Alias, info.URL)
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", name, label))
	}
	sort.Strings(completions)
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeTargets(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	targets := allTargets()
	completions := make([]string, 0, len(targets))
	for name, target := range targets {
		status := "disabled"
		if target.Enabled {
			status = "enabled"
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", name, status))
	}
	sort.Strings(completions)
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeAvailableSkills(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	used := usedArgs(args)
	skills, err := availableSkillCompletions()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return excludeUsed(skills, used), cobra.ShellCompDirectiveNoFileComp
}

func completeInstalledSkills(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	skills := installedSkillCompletions()
	return skills, cobra.ShellCompDirectiveNoFileComp
}

func usedArgs(args []string) map[string]bool {
	used := make(map[string]bool, len(args))
	for _, arg := range args {
		used[arg] = true
	}
	return used
}

func excludeUsed(values []string, used map[string]bool) []string {
	if len(used) == 0 {
		return values
	}

	filtered := values[:0]
	for _, value := range values {
		name := value
		if tab := indexTab(value); tab >= 0 {
			name = value[:tab]
		}
		if !used[name] {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func indexTab(s string) int {
	for i, r := range s {
		if r == '\t' {
			return i
		}
	}
	return -1
}

func allTargets() map[string]config.Target {
	targets := make(map[string]config.Target)

	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err == nil {
		for name, target := range globalCfg.Targets {
			targets[name] = target
		}
	}

	localCfg, err := loadConfigScope(config.ScopeLocal)
	if err == nil {
		for name, target := range localCfg.Targets {
			targets[name] = target
		}
	}

	return targets
}

func availableSkillCompletions() ([]string, error) {
	repos, cachePath, err := completionReposAndCache()
	if err != nil {
		return nil, err
	}

	cache := repo.NewCache(config.ExpandPath(cachePath))
	seen := make(map[string]string)
	for repoName, info := range repos {
		if !cache.Exists(repoName) {
			continue
		}
		skills, err := repo.DiscoverSkills(cache.PathFor(repoName), info.URL)
		if err != nil {
			continue
		}
		for _, skill := range skills {
			if _, exists := seen[skill.Name]; !exists {
				seen[skill.Name] = skill.Description
			}
		}
	}

	return completionMap(seen), nil
}

func completionReposAndCache() (map[string]config.RepoInfo, string, error) {
	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return nil, "", err
	}

	repos := make(map[string]config.RepoInfo, len(globalCfg.Repos))
	for name, info := range globalCfg.Repos {
		repos[name] = info
	}

	if scopeFlag != "global" {
		localCfg, err := loadConfigScope(config.ScopeLocal)
		if err == nil {
			for name, info := range localCfg.Repos {
				repos[name] = info
			}
		}
	}

	// Cache is a single shared directory; resolve the effective path so
	// completions that look up skills against the on-disk cache work
	// regardless of which scope owns the config entry.
	cachePath, err := config.NewLoader(config.ScopeGlobal).EffectiveCachePath()
	if err != nil {
		cachePath = globalCfg.Cache.Path
	}

	return repos, cachePath, nil
}

func installedSkillCompletions() []string {
	paths := getRemovePaths(targetFlag, scopeFlag)
	seen := make(map[string]string)

	for _, rp := range paths {
		skills, err := repo.ListInstalledSkills(rp.Path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}
		for _, skill := range skills {
			if _, exists := seen[skill.Name]; !exists {
				seen[skill.Name] = rp.Label
			}
		}
	}

	return completionMap(seen)
}

func completionMap(values map[string]string) []string {
	completions := make([]string, 0, len(values))
	for value, description := range values {
		if description == "" {
			completions = append(completions, value)
			continue
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", value, description))
	}
	sort.Strings(completions)
	return completions
}
