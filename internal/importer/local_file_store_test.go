package importer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFileStoreSave(t *testing.T) {
	tempDirectory := t.TempDir()
	uploadDirectory := filepath.Join(
		tempDirectory,
		"uploads",
	)

	store := NewLocalFileStore(uploadDirectory)

	content := []byte(
		"%PDF-1.7\nfake statement contents",
	)

	path, err := store.Save(
		context.Background(),
		bytes.NewReader(content),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if filepath.Dir(path) != uploadDirectory {
		t.Errorf(
			"expected file inside %q, got %q",
			uploadDirectory,
			path,
		)
	}

	filename := filepath.Base(path)

	if !strings.HasPrefix(filename, "statement-") {
		t.Errorf(
			"expected generated statement filename, got %q",
			filename,
		)
	}

	if filepath.Ext(filename) != ".pdf" {
		t.Errorf(
			"expected .pdf extension, got %q",
			filepath.Ext(filename),
		)
	}

	savedContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if !bytes.Equal(savedContent, content) {
		t.Errorf(
			"expected content %q, got %q",
			content,
			savedContent,
		)
	}
}
