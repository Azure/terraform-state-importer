package filepathparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePath_AbsolutePath(t *testing.T) {
	absPath, _ := os.Getwd()
	result, err := ParsePath(absPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != absPath {
		t.Errorf("Expected %s, got %s", absPath, result)
	}
}

func TestParsePath_HomeDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	testPath := "~/testdir/file.txt"
	expected := filepath.Join(home, "testdir", "file.txt")
	result, err := ParsePath(testPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	absExpected, _ := filepath.Abs(expected)
	if result != absExpected {
		t.Errorf("Expected %s, got %s", absExpected, result)
	}
}

func TestParsePath_RelativePath(t *testing.T) {
	relPath := "some/relative/path.txt"
	result, err := ParsePath(relPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	absExpected, _ := filepath.Abs(relPath)
	if result != absExpected {
		t.Errorf("Expected %s, got %s", absExpected, result)
	}
}

func TestParsePath_EmptyPath(t *testing.T) {
	result, err := ParsePath("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	wd, _ := os.Getwd()
	if result != wd {
		t.Errorf("Expected %s, got %s", wd, result)
	}
}
