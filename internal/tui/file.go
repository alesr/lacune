package tui

import (
	"fmt"
	"strings"

	"github.com/alesr/lacune/internal/coverage"
)

type fileItem struct{ file coverage.FileModel }

func (f fileItem) Title() string { return formatTruncatedPath(f.file.FilePath) }

func (f fileItem) Description() string {
	return fmt.Sprintf(
		"%.2f%% - %d/%d statements",
		f.file.Percent,
		f.file.CoveredStmt,
		f.file.TotalStmt,
	)
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
