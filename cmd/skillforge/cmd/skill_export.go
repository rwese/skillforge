package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/internal/repo"
	"github.com/spf13/cobra"
)

// skillExportCmd copies a cached skill into a fresh directory the
// caller specifies. The destination MUST NOT already exist (file,
// directory, or symlink); if it does, the command refuses to
// proceed. The skill's files land directly under the destination
// (no skill-name wrapper subdirectory).
var skillExportCmd = &cobra.Command{
	Use:   "export <name> <destination>",
	Short: "Export a cached skill to a new directory",
	Long: `Copy a cached skill into a NEW directory at <destination>.

The destination must not already exist (as a file, directory, or
symlink). If it does, the command refuses to overwrite or merge.
Missing parent directories along the destination path are created.

The skill's files land directly under <destination>; no
skill-name subdirectory is created. Nested skill names like
'architecture/event-sourced-commands' resolve to the same on-disk
skill; the slash-joined name is the internal skill identifier only.

Examples:
  skillforge skill export docker /tmp/docker-skill
  skillforge skill export architecture/event-sourced-commands ./es-skill`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeAvailableSkills,
	RunE:              runSkillExport,
}

func runSkillExport(cmd *cobra.Command, args []string) error {
	skillName := args[0]
	rawDestination := args[1]

	// Resolve the skill using the same local-first/global-fallback
	// resolver used by `skill install` and `skill load`, so a name
	// resolves to the same on-disk skill tree across commands.
	skill, err := resolveCachedSkill(skillName)
	if err != nil {
		PrintHint(HintRepoNotCached)
		return err
	}

	// Resolve the destination: expand ~, then absolute-path it so the
	// user sees a normalized path in both success and error messages.
	destination, err := resolveExportDestination(rawDestination)
	if err != nil {
		return err
	}

	// The destination MUST NOT EXIST in any form. Use Lstat so a
	// symlink (valid or broken) at the destination is treated as
	// "exists" — `os.Stat` would follow a valid symlink and miss a
	// symlink-to-directory collision.
	if err := assertExportDestinationMissing(destination); err != nil {
		return err
	}

	// Create the destination (and any missing parents).
	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("creating destination %q: %w", destination, err)
	}

	// Copy the skill's source tree into the destination. CopyDir is
	// recursive and preserves nested files. The skill's cache
	// directory should not contain a `.grimoire` marker (those are
	// written only into installed-skill directories), so the next
	// block is a defensive filter for an unexpected source that
	// would otherwise leak install metadata into the export.
	if err := repo.CopyDir(skill.Path, destination); err != nil {
		return fmt.Errorf("copying skill %q into %q: %w", skillName, destination, err)
	}

	// Strip any `.grimoire` markers anywhere under the destination
	// so the exported tree never carries install metadata — even if
	// the source contained a nested `.grimoire` for any reason.
	if err := removeGrimoireRecursive(destination); err != nil {
		return fmt.Errorf("stripping .grimoire from exported skill at %q: %w", destination, err)
	}

	fmt.Printf("Exported %s to %s\n", skillName, destination)
	return nil
}

// resolveExportDestination normalizes a user-supplied destination
// path: expand a leading `~` against the user's home directory, then
// resolve to an absolute path. The result is what the user expects to
// see in success and error messages.
func resolveExportDestination(raw string) (string, error) {
	expanded := config.ExpandPath(raw)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolving destination %q: %w", raw, err)
	}
	return abs, nil
}

// assertExportDestinationMissing refuses to proceed when the
// destination path is already occupied in any form (file, directory,
// symlink, broken symlink) or when the path is obstructed (a parent
// component is a non-directory). The latter surfaces a clear
// "destination not creatable" message instead of the underlying
// ENOTDIR from the filesystem.
func assertExportDestinationMissing(destination string) error {
	_, err := os.Lstat(destination)
	if err == nil {
		return fmt.Errorf("destination %q already exists; refusing to overwrite or merge into an existing path", destination)
	}
	if errors.Is(err, syscall.ENOTDIR) {
		return fmt.Errorf("destination %q is not creatable: a parent path is not a directory", destination)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("checking destination %q: %w", destination, err)
	}
	return nil
}

// removeGrimoireRecursive walks root and removes every entry named
// `.grimoire` it finds, at any depth. The exported tree must never
// carry a `.grimoire` marker regardless of where in the source tree
// one might have been sitting. Both file and directory `.grimoire`
// entries are removed; a directory one is removed along with all its
// contents (RemoveAll).
func removeGrimoireRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Base(path) != ".grimoire" {
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}
		// Skip descending into the (now removed) subtree.
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
}

func init() {
	skillCmd.AddCommand(skillExportCmd)
}
