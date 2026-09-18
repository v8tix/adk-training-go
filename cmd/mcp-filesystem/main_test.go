package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnsureSandboxDir_CreatesDirectoryAndSampleFile proves the
// doesn't-exist-yet branch: a fresh temp path gets both the directory and a
// real, readable hello.txt inside it.
func TestEnsureSandboxDir_CreatesDirectoryAndSampleFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test_files")

	if err := ensureSandboxDir(dir); err != nil {
		t.Fatalf("ensureSandboxDir() error = %v", err)
	}

	sample := filepath.Join(dir, "hello.txt")
	got, err := os.ReadFile(sample)
	if err != nil {
		t.Fatalf("reading sample file: %v", err)
	}
	if string(got) != "Hello from the MCP world!" {
		t.Errorf("sample file content = %q, want %q", got, "Hello from the MCP world!")
	}
}

// TestEnsureSandboxDir_LeavesExistingDirectoryUntouched proves the
// already-exists branch never overwrites a file a learner may have
// modified inside the sandbox.
func TestEnsureSandboxDir_LeavesExistingDirectoryUntouched(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(custom, []byte("my own notes"), 0o644); err != nil {
		t.Fatalf("seeding directory: %v", err)
	}

	if err := ensureSandboxDir(dir); err != nil {
		t.Fatalf("ensureSandboxDir() error = %v", err)
	}

	got, err := os.ReadFile(custom)
	if err != nil {
		t.Fatalf("reading seeded file: %v", err)
	}
	if string(got) != "my own notes" {
		t.Errorf("seeded file content changed: got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "hello.txt")); !os.IsNotExist(err) {
		t.Error("ensureSandboxDir() added hello.txt to an already-existing directory, want it left alone")
	}
}

// TestSandboxDir_ResolvesToAbsolutePath proves sandboxDir() itself resolves
// to an absolute path ending in the expected relative location.
func TestSandboxDir_ResolvesToAbsolutePath(t *testing.T) {
	dir, err := sandboxDir()
	if err != nil {
		t.Fatalf("sandboxDir() error = %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("sandboxDir() = %q, want an absolute path", dir)
	}
	if filepath.Base(dir) != "test_files" {
		t.Errorf("sandboxDir() = %q, want it to end in test_files", dir)
	}
}
