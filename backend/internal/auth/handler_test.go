package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAvatarFileNameFromPath(t *testing.T) {
	valid := "/auth/avatar/demo.webp"
	got, ok := avatarFileNameFromPath(&valid)
	if !ok {
		t.Fatalf("expected valid avatar path to be accepted")
	}
	if got != "demo.webp" {
		t.Fatalf("expected demo.webp, got %q", got)
	}

	invalid := "/auth/avatar/../../evil.webp"
	if _, ok := avatarFileNameFromPath(&invalid); ok {
		t.Fatalf("expected invalid avatar path to be rejected")
	}
}

func TestRemoveAvatarFileIgnoresMissingFileAndDeletesExisting(t *testing.T) {
	dir := t.TempDir()
	h := &Handler{avatarDir: dir}

	fileName := "avatar.png"
	target := filepath.Join(dir, fileName)
	if err := os.WriteFile(target, []byte("avatar"), 0o644); err != nil {
		t.Fatalf("failed to prepare avatar file: %v", err)
	}

	path := "/auth/avatar/" + fileName
	if err := h.removeAvatarFile(&path); err != nil {
		t.Fatalf("removeAvatarFile returned error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected avatar file to be removed, stat err=%v", err)
	}

	if err := h.removeAvatarFile(&path); err != nil {
		t.Fatalf("expected missing avatar removal to be ignored, got %v", err)
	}
}
