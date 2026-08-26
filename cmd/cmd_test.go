package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run executes meme-cli's command tree in-process with the given args and
// returns combined stdout/stderr output and any error. Building a fresh
// tree per call (via NewRootCmd) means no flag state leaks between tests.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestList_ShowsBundledTemplates(t *testing.T) {
	out, err := run(t, "list")
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if !strings.Contains(out, "the-scream") {
		t.Errorf("list output missing the-scream template:\n%s", out)
	}
}

func TestList_JSON(t *testing.T) {
	out, err := run(t, "list", "--json")
	if err != nil {
		t.Fatalf("list --json: unexpected error: %v", err)
	}

	var got struct {
		MemeDir   string                     `json:"meme_dir"`
		Templates map[string]json.RawMessage `json:"templates"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("list --json: invalid JSON: %v\n%s", err, out)
	}
	if got.MemeDir == "" {
		t.Error("list --json: meme_dir is empty")
	}
	if _, ok := got.Templates["the-scream"]; !ok {
		t.Errorf("list --json: templates missing the-scream:\n%s", out)
	}
}

func TestShow_JSON(t *testing.T) {
	out, err := run(t, "show", "the-scream", "--json")
	if err != nil {
		t.Fatalf("show --json: unexpected error: %v", err)
	}

	var got struct {
		MemeDir  string `json:"meme_dir"`
		Template struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			TextBoxes []struct {
				Name string `json:"name"`
			} `json:"text_boxes"`
		} `json:"template"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("show --json: invalid JSON: %v\n%s", err, out)
	}
	if got.MemeDir == "" {
		t.Error("show --json: meme_dir is empty")
	}
	if got.Template.ID != "the-scream" {
		t.Errorf("show --json: template.id = %q, want %q", got.Template.ID, "the-scream")
	}
	if len(got.Template.TextBoxes) == 0 {
		t.Error("show --json: template.text_boxes is empty")
	}
}

func TestRender_UnknownTemplateErrors(t *testing.T) {
	_, err := run(t, "render", "not-a-real-template")
	if err == nil {
		t.Fatal("render with unknown template id: want error, got none")
	}
}

func TestRender_WritesFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.png")

	out, err := run(t, "render", "quote-card", "unit test", "-o", outFile)
	if err != nil {
		t.Fatalf("render: unexpected error: %v\n%s", err, out)
	}

	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestRender_MemeDirOverride(t *testing.T) {
	memeDir := t.TempDir()
	tmplDir := filepath.Join(memeDir, "custom")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `
name: "Custom"
background: {color: "#222222", width: 100, height: 100}
text_boxes: [{name: a, x: 0, y: 0, width: 1, height: 1}]
`
	if err := os.WriteFile(filepath.Join(tmplDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := run(t, "show", "custom"); err == nil {
		t.Error("show custom without --meme-dir: want error, got none")
	}

	outFile := filepath.Join(memeDir, "out.png")
	if _, err := run(t, "--meme-dir", memeDir, "render", "custom", "hi", "-o", outFile); err != nil {
		t.Fatalf("render with --meme-dir: unexpected error: %v", err)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	out, err := run(t, "search", "definitely-not-a-keyword-xyz")
	if err != nil {
		t.Fatalf("search: unexpected error: %v", err)
	}
	if !strings.Contains(out, "no templates matched") {
		t.Errorf("search output = %q, want a no-match message", out)
	}
}

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatalf("version: unexpected error: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("version printed nothing")
	}
}

func TestLLMS(t *testing.T) {
	out, err := run(t, "llms")
	if err != nil {
		t.Fatalf("llms: unexpected error: %v", err)
	}
	for _, want := range []string{"search", "show", "render", "--meme-dir"} {
		if !strings.Contains(out, want) {
			t.Errorf("llms reference missing expected term %q:\n%s", want, out)
		}
	}
}
