package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name       string
		override   string
		env        string
		wantSource string
		wantErr    bool
	}{
		{
			name:       "no override or env falls back to bundled library",
			wantSource: "bundled seed library",
		},
		{
			name:       "env var wins over bundled default",
			env:        dir,
			wantSource: dir,
		},
		{
			name:       "flag override wins over env var",
			override:   dir,
			env:        filepath.Join(dir, "ignored"),
			wantSource: dir,
		},
		{
			name:     "nonexistent override directory errors",
			override: filepath.Join(dir, "does-not-exist"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv(EnvMemeDir, tt.env)
			}

			_, source, err := Resolve(tt.override)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Resolve(%q) with env %q: want error, got none", tt.override, tt.env)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(%q) with env %q: unexpected error: %v", tt.override, tt.env, err)
			}
			if source != tt.wantSource {
				t.Errorf("Resolve(%q) with env %q: source = %q, want %q", tt.override, tt.env, source, tt.wantSource)
			}
		})
	}
}

func TestResolve_RejectsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Resolve(file); err == nil {
		t.Error("Resolve on a regular file: want error, got none")
	}
}
