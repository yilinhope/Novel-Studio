package bootstrap

import "testing"

func TestSaveStyleConfigPatchesOnlyStyle(t *testing.T) {
	path := t.TempDir() + "/.ainovel/config.json"
	if err := SaveConfig(path, Config{Provider: "p", ModelName: "m", Style: "old"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveStyleConfig(path, "new"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Style != "new" || cfg.Provider != "p" || cfg.ModelName != "m" {
		t.Fatalf("style patch lost config: %#v", cfg)
	}
}
