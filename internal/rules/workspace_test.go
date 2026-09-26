package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceRuleFilesUseExplicitGlobalAndProjectRoots(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".ainovel", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".ainovel", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ainovel", "rules", "global.md"), []byte("global"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".ainovel", "rules", "project.md"), []byte("project"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := LoadOptions{HomeRulesDir: filepath.Join(home, ".ainovel", "rules"), ProjectRulesDir: DefaultProjectRulesDir(project)}
	files, err := ListRuleFiles(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Scope != SourceGlobal || files[1].Scope != SourceProject {
		t.Fatalf("unexpected rule files: %#v", files)
	}
	if err := SaveRuleFile(opts, SourceProject, "edited.md", "edited"); err != nil {
		t.Fatal(err)
	}
	content, err := ReadRuleFile(opts, SourceProject, "edited.md")
	if err != nil || content != "edited" {
		t.Fatalf("read edited rule: %q %v", content, err)
	}
	if err := RenameRuleFile(opts, SourceProject, "edited.md", "renamed.md"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteRuleFile(opts, SourceProject, "renamed.md"); err != nil {
		t.Fatal(err)
	}
}
