package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndDeleteOverrideUsesBookScope(t *testing.T) {
	book := t.TempDir()
	opts := LoadOptions{BookStyleDir: book}
	if err := SaveOverride(opts, OverrideProject, "voice.md", "book voice"); err != nil {
		t.Fatal(err)
	}
	raw, err := ReadOverride(opts, OverrideProject, "voice.md")
	if err != nil || raw != "book voice" {
		t.Fatalf("raw override not readable: %q %v", raw, err)
	}
	data, err := os.ReadFile(filepath.Join(book, "voice.md"))
	if err != nil || string(data) != "book voice" {
		t.Fatalf("override not saved: %q %v", string(data), err)
	}
	if err := DeleteOverride(opts, OverrideProject, "voice.md"); err != nil {
		t.Fatal(err)
	}
	if err := SaveOverride(opts, OverrideProject, "styles/custom.md", "custom"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteOverride(opts, OverrideProject, "styles/custom.md"); err != nil {
		t.Fatal(err)
	}
}
