package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	projectBrowserWidth  = 80
	projectBrowserHeight = 20
)

type projectBrowserItem struct {
	title  string
	filter string
	action projectBrowserAction
}

func (item projectBrowserItem) Title() string       { return item.title }
func (projectBrowserItem) Description() string      { return "" }
func (item projectBrowserItem) FilterValue() string { return item.filter }

type projectBrowserModel struct {
	list   list.Model
	action projectBrowserAction
}

func newProjectBrowserModel(view projectBrowserView) projectBrowserModel {
	items := make([]list.Item, 0, len(view.Entries)+1)
	if view.Current != view.Root {
		items = append(items, projectBrowserItem{
			title: "← ..", filter: "..",
			action: projectBrowserAction{Kind: projectBrowserUp},
		})
	}
	for _, entry := range view.Entries {
		title := "→ " + entry.Name + "/"
		action := projectBrowserAction{Kind: projectBrowserEnter, Path: entry.Path}
		if entry.Repository {
			marker := "[ ]"
			if entry.Selected {
				marker = "[x]"
			}
			title = marker + " " + entry.Name
			action.Kind = projectBrowserToggle
		}
		items = append(items, projectBrowserItem{
			title: title, filter: entry.Name, action: action,
		})
	}

	selectKey := key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select/open"),
	)
	confirmKey := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	)
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{selectKey, confirmKey}
	}

	menu := list.New(items, delegate, projectBrowserWidth, projectBrowserHeight)
	menu.Title = projectBrowserTitle(view)
	menu.SetStatusBarItemName("entry", "entries")
	menu.InfiniteScrolling = true
	menu.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j", "tab"),
		key.WithHelp("↓/tab", "next"),
	)
	menu.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "up"),
	)
	menu.KeyMap.Quit = key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("esc", "cancel"),
	)
	for index, item := range items {
		browserItem := item.(projectBrowserItem)
		if browserItem.action.Path == view.FocusPath {
			menu.Select(index)
			break
		}
	}
	return projectBrowserModel{list: menu}
}

func projectBrowserTitle(view projectBrowserView) string {
	title := fmt.Sprintf("%s · %d selected", view.Current, view.Selected)
	if view.SelectedOutside > 0 {
		title += fmt.Sprintf(" · %d outside this root", view.SelectedOutside)
	}
	return title
}

func (model projectBrowserModel) Init() tea.Cmd {
	return nil
}

func (model projectBrowserModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.list.SetSize(message.Width, message.Height)
		return model, nil
	case tea.KeyMsg:
		switch message.String() {
		case "enter":
			if model.list.SettingFilter() {
				break
			}
			model.action = projectBrowserAction{Kind: projectBrowserSave}
			return model, tea.Quit
		case " ":
			if model.list.SettingFilter() {
				break
			}
			item, ok := model.list.SelectedItem().(projectBrowserItem)
			if !ok {
				return model, nil
			}
			model.action = item.action
			return model, tea.Quit
		case "tab":
			return model.moveCursor(true)
		case "shift+tab":
			return model.moveCursor(false)
		case "ctrl+c":
			model.action = projectBrowserAction{Kind: projectBrowserCancel}
			return model, tea.Quit
		case "esc":
			if model.list.FilterState() == list.Unfiltered {
				model.action = projectBrowserAction{Kind: projectBrowserCancel}
				return model, tea.Quit
			}
		case "q":
			if !model.list.SettingFilter() {
				model.action = projectBrowserAction{Kind: projectBrowserCancel}
				return model, tea.Quit
			}
		}
	}

	var command tea.Cmd
	model.list, command = model.list.Update(message)
	return model, command
}

func (model projectBrowserModel) moveCursor(forward bool) (tea.Model, tea.Cmd) {
	var command tea.Cmd
	if model.list.SettingFilter() {
		model.list, command = model.list.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	if forward {
		model.list.CursorDown()
	} else {
		model.list.CursorUp()
	}
	return model, command
}

func (model projectBrowserModel) View() string {
	return model.list.View()
}

func (p huhPrompter) ChooseConfiguredProject(
	ctx context.Context,
	view projectBrowserView,
) (projectBrowserAction, error) {
	program := tea.NewProgram(
		newProjectBrowserModel(view),
		tea.WithContext(ctx),
		tea.WithInput(p.input),
		tea.WithOutput(p.output),
	)
	result, err := program.Run()
	if err != nil {
		return projectBrowserAction{}, err
	}
	model, ok := result.(projectBrowserModel)
	if !ok {
		return projectBrowserAction{}, fmt.Errorf("project browser returned %T", result)
	}
	if model.action.Kind == "" {
		return projectBrowserAction{Kind: projectBrowserCancel}, nil
	}
	return model.action, nil
}
