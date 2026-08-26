package template

import (
	"testing"
	"testing/fstest"
)

func validConfig() string {
	return `
name: "Test Template"
keywords: [foo, bar]
background:
  color: "#000000"
  width: 100
  height: 100
text_boxes:
  - name: top
    x: 0
    y: 0
    width: 1
    height: 0.5
`
}

func TestLoad(t *testing.T) {
	fsys := fstest.MapFS{
		"fry/config.yaml": {Data: []byte(validConfig())},
	}

	tmpl, err := Load(fsys, "fry")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if tmpl.ID != "fry" {
		t.Errorf("ID = %q, want %q", tmpl.ID, "fry")
	}
	if tmpl.Name != "Test Template" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "Test Template")
	}
	if len(tmpl.TextBoxes) != 1 {
		t.Fatalf("len(TextBoxes) = %d, want 1", len(tmpl.TextBoxes))
	}
}

func TestLoad_MissingConfig(t *testing.T) {
	fsys := fstest.MapFS{}
	if _, err := Load(fsys, "missing"); err == nil {
		t.Fatal("Load() with no config.yaml: want error, got nil")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	cases := map[string]string{
		"missing name": `
background: {color: "#000", width: 10, height: 10}
text_boxes: [{name: a, x: 0, y: 0, width: 1, height: 1}]
`,
		"no text boxes": `
name: "X"
background: {color: "#000", width: 10, height: 10}
`,
		"missing background": `
name: "X"
text_boxes: [{name: a, x: 0, y: 0, width: 1, height: 1}]
`,
		"background with no image or size": `
name: "X"
background: {}
text_boxes: [{name: a, x: 0, y: 0, width: 1, height: 1}]
`,
		"zero-size text box": `
name: "X"
background: {color: "#000", width: 10, height: 10}
text_boxes: [{name: a, x: 0, y: 0, width: 0, height: 1}]
`,
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{"t/config.yaml": {Data: []byte(cfg)}}
			if _, err := Load(fsys, "t"); err == nil {
				t.Errorf("Load(%q): want error, got nil", name)
			}
		})
	}
}

func TestLoadAll(t *testing.T) {
	fsys := fstest.MapFS{
		"fry/config.yaml":      {Data: []byte(validConfig())},
		"drake/config.yaml":    {Data: []byte(validConfig())},
		"not-a-template/notes": {Data: []byte("hello")},
	}

	tmpls, err := LoadAll(fsys)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(tmpls) != 2 {
		t.Fatalf("len(tmpls) = %d, want 2 (got %v)", len(tmpls), SortedIDs(tmpls))
	}
	ids := SortedIDs(tmpls)
	if ids[0] != "drake" || ids[1] != "fry" {
		t.Errorf("SortedIDs() = %v, want [drake fry]", ids)
	}
}

func TestMatches(t *testing.T) {
	tmpl := &Template{ID: "the-scream", Name: "The Scream", Keywords: []string{"panic", "deadline"}}

	for _, q := range []string{"scream", "SCREAM", "the scream", "panic", "dead"} {
		if !tmpl.Matches(q) {
			t.Errorf("Matches(%q) = false, want true", q)
		}
	}
	if tmpl.Matches("drake") {
		t.Error(`Matches("drake") = true, want false`)
	}
}
