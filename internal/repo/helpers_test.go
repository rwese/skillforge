package repo

import "os"

// writeFile writes body to path. Kept in a separate file because the
// regression test reuses it from multiple helpers.
func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}