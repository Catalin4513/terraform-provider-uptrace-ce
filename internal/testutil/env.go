// Package testutil provides shared helpers for tests across the provider.
package testutil

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv reads a .env file from the repo root and sets any KEY=VALUE
// entries in the process environment. Shell exports always win.
//
// The repo root is located by walking up from the current working directory
// until a .git directory is found — the same pattern pkg/testbed/fx.go uses
// in the parent app to resolve config files during tests.
func LoadEnv() {
	root, ok := findRepoRoot()
	if !ok {
		return
	}
	_ = godotenv.Load(filepath.Join(root, ".env"))
}

func findRepoRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
