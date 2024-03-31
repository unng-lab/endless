package ui

func (ui *UIEngine) Update() error {
	ui.clicked = false
	ui.UI.Update()
	return nil
}
