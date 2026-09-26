package taskpicker

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	New      key.Binding
	Edit     key.Binding
	Complete key.Binding
	Untrack  key.Binding
	Close    key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Select,
		k.New,
		k.Edit,
		k.Complete,
		k.Untrack,
		k.Close,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.New, k.Edit, k.Complete, k.Close},
	}
}

var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	New: key.NewBinding(
		key.WithKeys("n", "a"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Complete: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "done"),
	),
	Untrack: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "untrack"),
	),
	Close: key.NewBinding(
		key.WithKeys("esc", "q", "t"),
		key.WithHelp("esc", "back"),
	),
}
