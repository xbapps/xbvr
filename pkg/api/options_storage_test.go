package api

import (
	"os"
	"path/filepath"
	"testing"
)

// A stat failure that is not "not exist" (e.g. permission denied, which
// panicked addStorage on a service-user install) must be an error, never
// a nil dereference.
func TestResolveVolumePath(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, err := resolveVolumePath(dir); err != nil || got == "" {
		t.Errorf("resolveVolumePath(dir) = %q, %v; want abs path, nil", got, err)
	}
	if _, err := resolveVolumePath(filepath.Join(dir, "missing")); err == nil {
		t.Error("resolveVolumePath(missing) = nil error; want error")
	}
	if _, err := resolveVolumePath(file); err == nil {
		t.Error("resolveVolumePath(file) = nil error; want error")
	}

	// Permission-denied stat: stat needs no perms on the target itself,
	// so lock the parent (root bypasses perms, so skip there).
	if os.Geteuid() == 0 {
		t.Skip("running as root, permission case does not apply")
	}
	parent := filepath.Join(dir, "parent")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(parent, 0o755)
	if _, err := resolveVolumePath(filepath.Join(parent, "child")); err == nil {
		t.Error("resolveVolumePath(unreadable) = nil error; want error, not a panic")
	}
}
