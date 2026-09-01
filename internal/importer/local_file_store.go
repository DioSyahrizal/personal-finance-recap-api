package importer

import (
	"context"
	"fmt"
	"io"
	"os"
)

type LocalFileStore struct {
	directory string
}

var _ FileStore = (*LocalFileStore)(nil)

func NewLocalFileStore(directory string) *LocalFileStore {
	return &LocalFileStore{
		directory: directory,
	}
}

func (store *LocalFileStore) Save(
	ctx context.Context,
	source io.Reader,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	err := os.MkdirAll(
		store.directory,
		0o750,
	)

	if err != nil {
		return "", fmt.Errorf(
			"create upload directory: %w",
			err,
		)
	}

	file, err := os.CreateTemp(
		store.directory,
		"statement-*.pdf",
	)

	if err != nil {
		return "", fmt.Errorf(
			"create upload file: %w",
			err,
		)
	}
	defer file.Close()

	path := file.Name()

	_, err = io.Copy(file, source)
	if err != nil {
		_ = os.Remove(path)

		return "", fmt.Errorf(
			"copy upload file: %w",
			err,
		)
	}

	if err := ctx.Err(); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	return path, nil
}

func (store *LocalFileStore) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete uploaded file: %w", err)
	}

	return nil
}
