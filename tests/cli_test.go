//go:build integration

// Package tests drives the built meme-cli binary through declarative
// act.sh/assert.txt cases using github.com/ofthemachine/clitest — the same
// harness fraglet and operon use, so meme-cli's integration tests look and
// behave the same way across the ofthemachine projects.
package tests

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ofthemachine/clitest"
)

func TestMemeCLI(t *testing.T) {
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	rootDir := filepath.Clean(filepath.Join(testDir, ".."))

	clitest.RunSuite(t, clitest.Options{
		RootDir:           rootDir,
		BaseDirs:          []string{testDir},
		EnvOverrideVar:    "CLI_TEST_SUITE_DIR",
		BinaryName:        "meme-cli",
		BuildCommand:      []string{"go", "build", "-o", "meme-cli", "."},
		ProjectRootMarker: "go.mod",
	})
}
