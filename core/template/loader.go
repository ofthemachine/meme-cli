package template

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"gopkg.in/yaml.v3"
)

const configFileName = "config.yaml"

// Load reads a single template by id (its directory name) from fsys, which
// is rooted at a MEME_DIR (or the bundled default library).
func Load(fsys fs.FS, id string) (*Template, error) {
	cfgPath := path.Join(id, configFileName)
	data, err := fs.ReadFile(fsys, cfgPath)
	if err != nil {
		return nil, err
	}

	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", cfgPath, err)
	}
	t.ID = id
	t.Dir = id

	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("validating %s: %w", cfgPath, err)
	}
	return &t, nil
}

// LoadAll loads every template directory (one containing a config.yaml)
// directly under fsys's root.
func LoadAll(fsys fs.FS) (map[string]*Template, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("reading meme dir: %w", err)
	}

	out := make(map[string]*Template)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := Load(fsys, e.Name())
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // directory with no config.yaml isn't a template
			}
			return nil, fmt.Errorf("loading template %q: %w", e.Name(), err)
		}
		out[e.Name()] = t
	}
	return out, nil
}

// SortedIDs returns the map's keys in alphabetical order, for stable CLI
// output.
func SortedIDs(m map[string]*Template) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (t *Template) validate() error {
	if t.Name == "" {
		return errors.New("missing name")
	}
	if len(t.TextBoxes) == 0 {
		return errors.New("no text_boxes defined")
	}
	if t.Background == nil {
		return errors.New("missing background")
	}
	if t.Background.Image == "" && (t.Background.Width <= 0 || t.Background.Height <= 0) {
		return errors.New("background needs either an image or width/height")
	}
	for i, tb := range t.TextBoxes {
		n := len(tb.CornerPoints())
		if n == 3 || n == 4 {
			continue
		}
		if n != 0 {
			return fmt.Errorf("text_boxes[%d] (%s): quad/parallelogram needs 3 points (tl,tr,bl) or 4 (tl,tr,br,bl), got %d", i, tb.Name, n)
		}
		if tb.Width <= 0 || tb.Height <= 0 {
			return fmt.Errorf("text_boxes[%d] (%s): width/height must be > 0 (or set quad)", i, tb.Name)
		}
	}
	return nil
}
