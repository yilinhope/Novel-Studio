package sim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSourcesReturnsSupportedFilesAndFingerprint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("你好"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "b.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skip.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := ListSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].RelativePath != "a.txt" || items[1].RelativePath != "nested/b.md" {
		t.Fatalf("unexpected sources: %#v", items)
	}
	if items[0].Fingerprint == "" || items[0].SHA256 == "" || items[0].SizeBytes == 0 {
		t.Fatalf("source metadata incomplete: %#v", items[0])
	}
}
