package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alesr/lacune/internal/coverage"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type (
	packageCoverage struct {
		name    string
		covered int
		total   int
		percent float64
	}

	moduleCoverage struct {
		name     string
		covered  int
		total    int
		percent  float64
		packages []packageCoverage
	}

	packageViewModel struct {
		viewport viewport.Model
		open     bool
		width    int
		height   int
	}
)

func newPackageViewModel() packageViewModel {
	return packageViewModel{viewport: viewport.Model{Width: 0, Height: 0}}
}

func (model packageViewModel) IsOpen() bool { return model.open }

func (model packageViewModel) Open(content string) packageViewModel {
	newModel := model
	newModel.open = true
	newModel.viewport.SetContent(content)
	newModel.viewport.GotoTop()
	return newModel
}

func (model packageViewModel) Close() packageViewModel {
	newModel := model
	newModel.open = false
	return newModel
}

func (model packageViewModel) SetContent(content string) packageViewModel {
	newModel := model
	newModel.viewport.SetContent(content)
	return newModel
}

func (model packageViewModel) SetSize(width, height int) packageViewModel {
	newModel := model
	newModel.viewport.Width = width
	newModel.viewport.Height = height
	newModel.width = width
	newModel.height = height
	return newModel
}

func (model packageViewModel) Update(msg tea.Msg) (packageViewModel, tea.Cmd) {
	newModel := model
	var cmd tea.Cmd
	newModel.viewport, cmd = newModel.viewport.Update(msg)
	return newModel, cmd
}

func (model packageViewModel) ScrollUp(lines int) packageViewModel {
	newModel := model
	newModel.viewport.ScrollUp(lines)
	return newModel
}

func (model packageViewModel) ScrollDown(lines int) packageViewModel {
	newModel := model
	newModel.viewport.ScrollDown(lines)
	return newModel
}

func (model packageViewModel) View() string {
	styles := defaultStyles()
	title := styles.packageHeader.Render("Package Coverage")
	return lipgloss.JoinVertical(lipgloss.Left, title, model.viewport.View())
}

func buildPackageCoverageContent(files []coverage.FileModel, moduleName string, width int) string {
	modules := buildModuleCoverage(files, moduleName)
	if len(modules) == 0 {
		return statusMsgNone
	}

	var content strings.Builder
	for i, module := range modules {
		if i > 0 {
			content.WriteString("\n")
		}
		moduleLine := fmt.Sprintf("%s  %.2f%% (%d/%d)", module.name, module.percent, module.covered, module.total)
		content.WriteString(truncateLine(moduleLine, width))
		content.WriteString("\n")
		for _, pkg := range module.packages {
			pkgLine := fmt.Sprintf("  %s  %.2f%% (%d/%d)", pkg.name, pkg.percent, pkg.covered, pkg.total)
			content.WriteString(truncateLine(pkgLine, width))
			content.WriteString("\n")
		}
	}
	return content.String()
}

func buildModuleCoverage(files []coverage.FileModel, moduleName string) []moduleCoverage {
	modulePackages := make(map[string]map[string]*packageCoverage)
	moduleTotals := make(map[string]*moduleCoverage)

	for _, file := range files {
		module := moduleName
		if module == "" ||
			module == moduleNameUnknown ||
			!strings.HasPrefix(file.FilePath, module+"/") {
			module = moduleFromPath(file.FilePath)
		}

		pkg := packageFromPath(file.FilePath)

		if _, ok := modulePackages[module]; !ok {
			modulePackages[module] = make(map[string]*packageCoverage)
			moduleTotals[module] = &moduleCoverage{name: module}
		}

		if _, ok := modulePackages[module][pkg]; !ok {
			modulePackages[module][pkg] = &packageCoverage{name: pkg}
		}

		moduleTotals[module].covered += file.CoveredStmt
		moduleTotals[module].total += file.TotalStmt
		modulePackages[module][pkg].covered += file.CoveredStmt
		modulePackages[module][pkg].total += file.TotalStmt
	}

	modules := make([]moduleCoverage, 0, len(modulePackages))

	for module, packages := range modulePackages {
		moduleTotals[module].packages = make([]packageCoverage, 0, len(packages))

		for _, pkg := range packages {
			if pkg.total > 0 {
				pkg.percent = float64(pkg.covered) / float64(pkg.total) * 100
			}
			moduleTotals[module].packages = append(moduleTotals[module].packages, *pkg)
		}

		sort.Slice(moduleTotals[module].packages, func(i, j int) bool {
			return moduleTotals[module].packages[i].name < moduleTotals[module].packages[j].name
		})

		if moduleTotals[module].total > 0 {
			moduleTotals[module].percent = float64(moduleTotals[module].covered) / float64(moduleTotals[module].total) * 100
		}
		modules = append(modules, *moduleTotals[module])
	}

	sort.Slice(modules, func(i, j int) bool {
		if moduleName != "" && modules[i].name == moduleName {
			return true
		}
		if moduleName != "" && modules[j].name == moduleName {
			return false
		}
		return modules[i].name < modules[j].name
	})
	return modules
}

func moduleFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		return strings.Join(parts[:3], "/")
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func packageFromPath(path string) string {
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[:idx]
	}
	return path
}

func truncateLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	return ansi.TruncateWc(line, width, "")
}
