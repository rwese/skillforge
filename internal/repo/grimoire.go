package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rwese/skillforge/pkg/grimoire"
)

// ProgressCallback is called during file copy operations.
type ProgressCallback func(src, dst string)

// ReadGrimoire reads a .grimoire file from a skill directory.
// It handles both the legacy [metadata] section format and the newer root-level format.
func ReadGrimoire(skillPath string) (*grimoire.Grimoire, error) {
	grimoirePath := filepath.Join(skillPath, ".grimoire")
	data, err := os.ReadFile(grimoirePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// First try parsing with [metadata] section (legacy format)
	// Then fall back to root-level fields (new format)
	
	// Try legacy [metadata] section format
	var legacy struct {
		Metadata struct {
			Version     int    `toml:"version"`
			Source      string `toml:"source"`
			Commit      string `toml:"commit"`
			InstalledAt string `toml:"installed_at"`
		} `toml:"metadata"`
	}
	
	if _, err := toml.Decode(string(data), &legacy); err == nil && legacy.Metadata.Commit != "" {
		t, _ := time.Parse(time.RFC3339, legacy.Metadata.InstalledAt)
		return &grimoire.Grimoire{
			Version:     legacy.Metadata.Version,
			Source:      legacy.Metadata.Source,
			Commit:      legacy.Metadata.Commit,
			InstalledAt: t,
		}, nil
	}
	
	// Try new root-level format
	var g grimoire.Grimoire
	if _, err := toml.Decode(string(data), &g); err != nil {
		return nil, err
	}

	return &g, nil
}

// WriteGrimoire writes a .grimoire file to a skill directory.
func WriteGrimoire(skillPath string, g *grimoire.Grimoire) error {
	grimoirePath := filepath.Join(skillPath, ".grimoire")
	data, err := toml.Marshal(g)
	if err != nil {
		return fmt.Errorf("marshaling grimoire: %w", err)
	}

	return os.WriteFile(grimoirePath, data, 0644)
}

// InstallSkill copies a skill to a target directory with grimoire metadata.
func InstallSkill(skill grimoire.Skill, targetPath string, commit string) error {
	return InstallSkillWithProgress(skill, targetPath, commit, nil)
}

// InstallSkillWithProgress copies a skill with progress reporting.
func InstallSkillWithProgress(skill grimoire.Skill, targetPath string, commit string, progress ProgressCallback) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Copy skill contents
	if err := copyDirWithProgress(skill.Path, targetPath, progress); err != nil {
		return fmt.Errorf("copying skill: %w", err)
	}

	// Write grimoire
	g := &grimoire.Grimoire{
		Version:     1,
		Source:      skill.Source,
		Commit:      commit,
		InstalledAt: time.Now(),
	}

	if err := WriteGrimoire(targetPath, g); err != nil {
		return fmt.Errorf("writing grimoire: %w", err)
	}

	return nil
}

// copyDir copies a directory recursively.
func copyDir(src, dst string) error {
	return copyDirWithProgress(src, dst, nil)
}

// copyDirWithProgress copies a directory with optional progress reporting.
func copyDirWithProgress(src, dst string, progress ProgressCallback) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if err := copyFile(path, dstPath); err != nil {
			return err
		}

		if progress != nil {
			progress(path, dstPath)
		}

		return nil
	})
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// RemoveSkill removes a skill from a target directory.
func RemoveSkill(targetPath string) error {
	return os.RemoveAll(targetPath)
}

// LinkSkill creates a symlink from targetPath to the skill's actual path.
// targetPath is the full path where the symlink should be created (including skill name).
// skill.Path is the source directory to link to. The symlink always uses an
// absolute path so it remains valid regardless of how the target directory
// is reached (e.g. via a different symlink in the parent chain).
func LinkSkill(skill grimoire.Skill, targetPath string) error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Check if target already exists
	if _, err := os.Lstat(targetPath); err == nil {
		return fmt.Errorf("target path already exists: %s", targetPath)
	}

	// Resolve the skill source to an absolute path. This matters when
	// skill.Path itself is a relative path or contains symlinks we want
	// to normalize before storing it in the new symlink target.
	absSkillPath, err := filepath.Abs(skill.Path)
	if err != nil {
		return fmt.Errorf("resolving absolute path for skill %q: %w", skill.Path, err)
	}

	if err := os.Symlink(absSkillPath, targetPath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}

	return nil
}

// ListInstalledSkillsSymlinks lists all skills in a target directory as symlinks.
// Returns skills with resolved symlink targets.
func ListInstalledSkillsSymlinks(targetPath string) ([]grimoire.Skill, error) {
	var skills []grimoire.Skill

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return skills, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}

		skillPath := filepath.Join(targetPath, entry.Name())

		// Resolve symlink to get actual source
		linkTarget, err := os.Readlink(skillPath)
		if err != nil {
			// Not a symlink, skip
			continue
		}

		// If relative symlink, resolve it
		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(skillPath), linkTarget)
		}

		skills = append(skills, grimoire.Skill{
			Name: entry.Name(),
			Path: linkTarget,
		})
	}

	return skills, nil
}

// IsBrokenSymlink reports whether path is a symlink whose target does not
// resolve to an existing path. Non-symlinks and missing paths return false.
func IsBrokenSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	// os.Stat follows the symlink. If the target is missing, it returns
	// an error (typically ENOENT or ENOTDIR). os.Readlink cannot detect
	// a broken target because it only returns the link string.
	_, err = os.Stat(path)
	return err != nil
}

// BrokenSymlink describes a symlink in a target directory whose target
// does not resolve to an existing path.
type BrokenSymlink struct {
	// Name is the entry name in the target directory (typically the skill
	// name the agent uses to address the link).
	Name string
	// Path is the absolute path of the symlink on disk.
	Path string
	// LinkTarget is the raw target string the symlink was created with
	// (unresolved, exactly as stored in the filesystem).
	LinkTarget string
}

// FindBrokenSymlinks walks targetPath and returns every entry that is a
// symlink whose target does not resolve. Regular files, real directories
// (with or without a .grimoire), and valid symlinks are ignored. A
// nonexistent targetPath yields a nil result with no error so callers
// can treat "no target dir" the same as "no broken symlinks".
func FindBrokenSymlinks(targetPath string) ([]BrokenSymlink, error) {
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var broken []BrokenSymlink
	for _, entry := range entries {
		skillPath := filepath.Join(targetPath, entry.Name())
		if !IsBrokenSymlink(skillPath) {
			continue
		}
		linkTarget, err := os.Readlink(skillPath)
		if err != nil {
			// Lstat said it's a symlink; if Readlink fails something is
			// seriously wrong. Skip rather than abort the whole scan.
			continue
		}
		broken = append(broken, BrokenSymlink{
			Name:       entry.Name(),
			Path:       skillPath,
			LinkTarget: linkTarget,
		})
	}
	return broken, nil
}

// RellinkSkill removes any existing entry at targetPath and creates a new
// symlink to skill.Path using an absolute path. This is the repair
// primitive used when an existing skill symlink is broken (typically
// because the project containing the link was moved while the link was
// still a relative path).
//
// RellinkSkill deliberately does NOT recurse: it only removes targetPath
// itself, so it is safe to call on a single skill link. To remove a
// real directory, the caller should use RemoveSkill.
func RellinkSkill(skill grimoire.Skill, targetPath string) error {
	// Remove the existing entry. os.Remove works for symlinks, regular
	// files, and empty directories. A non-existent path is fine (we are
	// about to create the link either way).
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing existing entry at %s: %w", targetPath, err)
	}
	return LinkSkill(skill, targetPath)
}

// ListInstalledSkills lists all installed skills in a target.
// It includes both directories with .grimoire files and symlinks.
func ListInstalledSkills(targetPath string) ([]grimoire.Skill, error) {
	var skills []grimoire.Skill

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return skills, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		skillPath := filepath.Join(targetPath, entry.Name())
		info, err := os.Lstat(skillPath)
		if err != nil {
			continue
		}

		// Handle symlinks
		isSymlink := info.Mode()&os.ModeSymlink != 0
		if isSymlink {
			// For symlinks, resolve the actual path
			linkTarget, err := os.Readlink(skillPath)
			if err != nil {
				continue
			}
			if !filepath.IsAbs(linkTarget) {
				linkTarget = filepath.Join(filepath.Dir(skillPath), linkTarget)
			}

			skills = append(skills, grimoire.Skill{
				Name: entry.Name(),
				Path: linkTarget,
			})
			continue
		}

		// For directories, only include if they have a .grimoire file
		if !entry.IsDir() {
			continue
		}

		// Try to read grimoire metadata
		g, err := ReadGrimoire(skillPath)
		if err != nil || g == nil {
			continue
		}

		skills = append(skills, grimoire.Skill{
			Name:   entry.Name(),
			Path:   skillPath,
			Source: g.Source,
			Commit: g.Commit,
		})
	}

	return skills, nil
}
