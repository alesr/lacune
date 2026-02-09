package coverage

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/cover"
)

const (
	None LineStatus = iota
	Covered
	Uncovered
	Partial
)

type (
	SearchIndex struct {
		Filename  string
		Functions []string
		Variables []string
		Content   string
	}

	Totals struct {
		CoveredStmt int
		TotalStmt   int
		Percent     float64
	}

	LineStatus int

	LineInfo struct {
		LineNo int
		Text   string
		Status LineStatus
	}

	FileModel struct {
		FilePath       string
		LineInfo       []LineInfo
		UncoveredLines []int
		CoveredStmt    int
		TotalStmt      int
		Percent        float64
		SearchIndex    SearchIndex
	}
)

func Load(profilePath string) ([]*cover.Profile, error) {
	profiles, err := cover.ParseProfiles(profilePath)
	if err != nil {
		return nil, fmt.Errorf("could not parse coverage profile: %w", err)
	}
	return profiles, nil
}

func ComputeTotals(profiles []*cover.Profile) Totals {
	var totals Totals
	for _, profile := range profiles {
		for _, block := range profile.Blocks {
			totals.TotalStmt += block.NumStmt
			if block.Count > 0 {
				totals.CoveredStmt += block.NumStmt
			}
		}
	}
	if totals.TotalStmt > 0 {
		totals.Percent = float64(totals.CoveredStmt) / float64(totals.TotalStmt) * 100
	}
	return totals
}

// parseGoFile parses a Go file and extracts functions and variables
func parseGoFile(filePath string) (SearchIndex, error) {
	var index SearchIndex
	index.Filename = filepath.Base(filePath)

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return index, fmt.Errorf("could parse Go file %s: %w", filePath, err)
	}

	ast.Inspect(file, func(node ast.Node) bool {
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			index.Functions = append(index.Functions, funcDecl.Name.Name)
		}
		if genDecl, ok := node.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						index.Variables = append(index.Variables, name.Name)
					}
				}
			}
		}
		return true
	})

	content, err := os.ReadFile(filePath)
	if err != nil {
		return index, fmt.Errorf("could not read file %s: %w", filePath, err)
	}
	index.Content = string(content)
	return index, nil
}

func findModuleRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("could not resolve absolute path for %s: %w", startDir, err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found walking up from %s", startDir)
		}
		dir = parent
	}
}

func readModulePath(goModPath string) (string, error) {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("could not read go.mod %s: %w", goModPath, err)
	}

	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if mod, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(mod), nil
		}
	}
	return "", fmt.Errorf("module path not found in %s", goModPath)
}

func resolveProfileFile(moduleRoot, modulePath, profileFile, fallbackDir string) string {
	if filepath.IsAbs(profileFile) {
		return profileFile
	}

	if modulePath != "" {
		if rel, ok := strings.CutPrefix(profileFile, modulePath+"/"); ok {
			return filepath.Join(moduleRoot, filepath.FromSlash(rel))
		}
	}

	// relative to module root
	candidate := filepath.Join(moduleRoot, filepath.FromSlash(profileFile))
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	// fallback: relative to the provided dir
	return filepath.Join(fallbackDir, filepath.FromSlash(profileFile))
}

func BuildFileModels(profiles []*cover.Profile, dir string) ([]FileModel, error) {
	moduleRoot, err := findModuleRoot(dir)
	if err != nil {
		return nil, err
	}
	modulePath, err := readModulePath(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		return nil, err
	}

	fileMap := make(map[string]*FileModel)
	for _, profile := range profiles {
		if _, exists := fileMap[profile.FileName]; !exists {
			fileMap[profile.FileName] = &FileModel{
				FilePath: profile.FileName,
			}
		}
	}

	for _, profile := range profiles {
		fileModel := fileMap[profile.FileName]
		for _, block := range profile.Blocks {
			fileModel.TotalStmt += block.NumStmt
			if block.Count > 0 {
				fileModel.CoveredStmt += block.NumStmt
			}

			for lineNo := block.StartLine; lineNo <= block.EndLine; lineNo++ {
				status := Uncovered
				if block.Count > 0 {
					status = Covered
				}

				// ensure slice is large enough
				for len(fileModel.LineInfo) < lineNo {
					fileModel.LineInfo = append(fileModel.LineInfo, LineInfo{
						LineNo: len(fileModel.LineInfo) + 1,
						Status: None,
					})
				}

				cur := fileModel.LineInfo[lineNo-1].Status
				if cur == None {
					fileModel.LineInfo[lineNo-1].Status = status
				} else if cur != status {
					fileModel.LineInfo[lineNo-1].Status = Partial
				}
			}
		}
	}

	var fileModels []FileModel
	for _, fileModel := range fileMap {
		resolvedPath := resolveProfileFile(moduleRoot, modulePath, fileModel.FilePath, dir)

		searchIndex, err := parseGoFile(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("could not parse Go file %s: %w", resolvedPath, err)
		}
		fileModel.SearchIndex = searchIndex

		f, err := os.Open(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", resolvedPath, err)
		}

		scanner := bufio.NewScanner(f)
		var lineNo int
		for scanner.Scan() {
			lineNo++
			if lineNo <= len(fileModel.LineInfo) {
				fileModel.LineInfo[lineNo-1].Text = scanner.Text()
			}
		}

		scanErr := scanner.Err()
		closeErr := f.Close()

		if scanErr != nil {
			return nil, fmt.Errorf("could not read file %s: %w", resolvedPath, scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("could not close file %s: %w", resolvedPath, closeErr)
		}

		if fileModel.TotalStmt > 0 {
			fileModel.Percent = float64(fileModel.CoveredStmt) / float64(fileModel.TotalStmt) * 100
		}
		for _, line := range fileModel.LineInfo {
			if line.Status == Uncovered {
				fileModel.UncoveredLines = append(fileModel.UncoveredLines, line.LineNo)
			}
		}
		fileModels = append(fileModels, *fileModel)
	}
	return fileModels, nil
}
