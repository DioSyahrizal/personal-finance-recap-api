package recap

import "context"

type ImportCreator interface {
	CreateImport(
		ctx context.Context,
		input CreateInput,
		filePath string,
	) (Recap, error)
}
