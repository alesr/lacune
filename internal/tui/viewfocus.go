package tui

const (
	focusFileList focusArea = iota
	focusViewport
)

type focusArea int

type FocusModel struct {
	focus focusArea
}

func newFocusModel() FocusModel {
	return FocusModel{focus: focusFileList}
}

func (focusModel FocusModel) setFocus(focus focusArea) FocusModel {
	newModel := focusModel
	newModel.focus = focus
	return newModel
}

func (focusModel FocusModel) handleFocusToggle() FocusModel {
	newModel := focusModel
	if newModel.focus == focusFileList {
		newModel.focus = focusViewport
	} else {
		newModel.focus = focusFileList
	}
	return newModel
}

func (focusModel FocusModel) getFocus() focusArea {
	return focusModel.focus
}
