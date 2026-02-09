package report

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/alesr/lacune/internal/coverage"
)

func Print(w io.Writer, totals coverage.Totals, files []coverage.FileModel, topN int) {
	writer := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer writer.Flush()

	fmt.Fprintf(writer, "Total Coverage: %.2f%% (%d/%d statements)\n", totals.Percent, totals.CoveredStmt, totals.TotalStmt)
	fmt.Fprintf(writer, "\n")

	sort.Slice(files, func(i, j int) bool {
		return files[i].Percent < files[j].Percent
	})

	fmt.Fprintf(writer, "Top %d Lowest-Coverage Files:\n", topN)
	fmt.Fprintf(writer, "File\tCoverage\tCovered/Total\n")

	for i := 0; i < topN && i < len(files); i++ {
		file := files[i]
		fmt.Fprintf(writer, "%s\t%.2f%%\t%d/%d\n", file.FilePath, file.Percent, file.CoveredStmt, file.TotalStmt)
	}

	var zeroCoverageFiles []string
	for _, file := range files {
		if file.Percent == 0 {
			zeroCoverageFiles = append(zeroCoverageFiles, file.FilePath)
		}
	}
	if len(zeroCoverageFiles) > 0 {
		fmt.Fprintf(writer, "\nFiles with 0%% Coverage:\n")
		for _, file := range zeroCoverageFiles {
			fmt.Fprintf(writer, "- %s\n", file)
		}
	}
}
