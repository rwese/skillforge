package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/rwese/skillforge/pkg/grimoire"
	"github.com/spf13/cobra"
)

var skillLoadCmd = &cobra.Command{
	Use:               "load <name>",
	Short:             "Copy a cached skill into a temp directory and print its SKILL.md",
	Long: `Copy a cached skill into a fresh temp directory and print its SKILL.md.

The temp directory is created under the system temp directory with a
layout of:

  /tmp/skillforge-<hash>/<skill-name>/

where <hash> is derived from the SHA256 of the current working
directory (formatted as 3-3 hex characters). Nested skill names such
as 'architecture/event-sourced-commands' keep their category path on
disk, so the loaded skill lands at:

  /tmp/skillforge-<hash>/architecture/event-sourced-commands/

The directory is left in place; it is the caller's responsibility to
remove it when finished.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAvailableSkills,
	RunE:              runSkillLoad,
}

func runSkillLoad(cmd *cobra.Command, args []string) error {
	skillName := args[0]

	target, err := resolveCachedSkill(skillName)
	if err != nil {
		PrintHint(HintRepoNotCached)
		return err
	}

	hash, err := cwdHashSegment()
	if err != nil {
		return err
	}
	parent := filepath.Join(os.TempDir(), "skillforge-"+hash)
	dst := filepath.Join(parent, filepath.FromSlash(target.Name))
	if err := repo.CopyDir(target.Path, dst); err != nil {
		return fmt.Errorf("copying skill into %s: %w", dst, err)
	}

	body, err := os.ReadFile(filepath.Join(target.Path, "SKILL.md"))
	if err != nil {
		return fmt.Errorf("reading SKILL.md from %s: %w", target.Path, err)
	}

	fmt.Printf("Skill loaded at: %s\n\nSKILL.md:\n\n%s", dst, string(body))
	return nil
}

// resolveCachedSkill looks up a skill across cached repos using the
// same local-first/global-fallback resolution as runSkillInstall.
// Returns the discovered grimoire.Skill (whose Path is the absolute
// on-disk path inside the cache) or an error if the skill is not in
// any cached repo.
func resolveCachedSkill(skillName string) (*grimoire.Skill, error) {
	globalCfg, err := config.NewLoader(config.ScopeGlobal).Load()
	if err != nil {
		return nil, err
	}

	localCfg, _ := config.NewLoader(config.ScopeLocal).Load()

	cachePath, err := config.NewLoader(config.ScopeGlobal).EffectiveCachePath()
	if err != nil {
		return nil, err
	}
	cache := repo.NewCache(config.ExpandPath(cachePath))

	// Preferred scope first (local wins, then global), then the
	// other scope as fallback. Mirror runSkillInstall so `skill load`
	// and `skill install` agree on which skill is being acted on.
	preferred := make(map[string]config.RepoInfo)
	fallback := make(map[string]config.RepoInfo)
	if localCfg != nil {
		for name, info := range localCfg.Repos {
			preferred[name] = info
		}
	}
	for name, info := range globalCfg.Repos {
		if _, ok := preferred[name]; !ok {
			fallback[name] = info
		}
	}

	for _, set := range []map[string]config.RepoInfo{preferred, fallback} {
		for repoName, info := range set {
			if !cache.Exists(repoName) {
				continue
			}
			skills, err := repo.DiscoverSkills(cache.PathFor(repoName), info.URL)
			if err != nil {
				continue
			}
			for i := range skills {
				if skills[i].Name == skillName {
					return &skills[i], nil
				}
			}
		}
	}

	return nil, fmt.Errorf("skill %q not found in any cached repository", skillName)
}

// cwdHashSegment returns a 6-hex-char hash of the current working
// directory formatted as "xxx-xxx" (3+3 chars). Two characters of
// dash separator improve readability when scanning /tmp; the
// underlying entropy is unchanged.
func cwdHashSegment() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting cwd: %w", err)
	}
	sum := sha256.Sum256([]byte(cwd))
	hex6 := hex.EncodeToString(sum[:3])
	return hex6[:3] + "-" + hex6[3:], nil
}

// init wires the load subcommand into the skill command group.
func init() {
	skillCmd.AddCommand(skillLoadCmd)
}