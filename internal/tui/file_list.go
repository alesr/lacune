package tui

import (
	"github.com/alesr/lacune/internal/coverage"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type fileListModel struct {
	list          list.Model
	files         []coverage.FileModel
	keys          keyMap
	width         int
	height        int
	settingFilter bool
}

func newFileListModel(files []coverage.FileModel, keys keyMap) *fileListModel {
	styles := defaultStyles()
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = styles.normalTitle
	delegate.Styles.NormalDesc = styles.normalDesc
	delegate.Styles.SelectedTitle = styles.selectedTitle
	delegate.Styles.SelectedDesc = styles.selectedDesc

	fileList := list.New(convertToFileItems(files), delegate, 0, 0)
	fileList.Title = "Files"
	fileList.Styles.Title = styles.packageHeader
	fileList.SetFilteringEnabled(true)
	fileList.SetShowHelp(false)

	return &fileListModel{
		list:          fileList,
		files:         files,
		keys:          keys,
		settingFilter: false,
	}
}

func (fileList *fileListModel) Init() tea.Cmd { return nil }

func (fileList *fileListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	fileList.list, cmd = fileList.list.Update(msg)
	fileList.settingFilter = fileList.list.SettingFilter()
	return fileList, cmd
}

func (fileList *fileListModel) View() string { return fileList.list.View() }

func (fileList *fileListModel) SetSize(width, height int) *fileListModel {
	fileList.list.SetSize(width, height)
	fileList.width = width
	fileList.height = height
	return fileList
}

func (fileList *fileListModel) SelectedFile() (coverage.FileModel, bool) {
	if selected, ok := fileList.list.SelectedItem().(fileItem); ok {
		return selected.file, true
	}
	return coverage.FileModel{}, false
}

func (fileList *fileListModel) SetItems(files []coverage.FileModel) *fileListModel {
	fileList.list.SetItems(convertToFileItems(files))
	fileList.files = files
	return fileList
}

func (fileList *fileListModel) Select(index int) *fileListModel {
	fileList.list.Select(index)
	return fileList
}

func (fileList *fileListModel) FilterValue() string {
	return fileList.list.FilterValue()
}

func (fileList *fileListModel) SettingFilter() bool {
	return fileList.settingFilter
}

func convertToFileItems(files []coverage.FileModel) []list.Item {
	fileItems := make([]list.Item, len(files))
	for i, file := range files {
		fileItems[i] = fileItem{file: file}
	}
	return fileItems
}
