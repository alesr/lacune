package tui

import (
	"fmt"
	"strings"

	"github.com/alesr/lacune/internal/coverage"
)

type fileItem struct {
	file coverage.FileModel
}

func (f fileItem) Title() string {
	return fmt.Sprintf("%s (%.2f%%)", f.file.FilePath, f.file.Percent)
}

func (f fileItem) Description() string {
	return fmt.Sprintf("%d/%d statements", f.file.CoveredStmt, f.file.TotalStmt)
}

func (f fileItem) FilterValue() string {
	return fmt.Sprintf(
		"%s %s %s %s",
		f.file.FilePath,
		strings.Join(f.file.SearchIndex.Functions, " "),
		strings.Join(f.file.SearchIndex.Variables, " "),
		f.file.SearchIndex.Content,
	)
}
