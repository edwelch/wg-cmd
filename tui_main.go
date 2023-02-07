package main

import (
	"github.com/76creates/stickers"
	"github.com/andrianbdn/wg-cmd/app"
	"github.com/andrianbdn/wg-cmd/tutils"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"log"
)

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	NewPeer    key.Binding
	DeletePeer key.Binding
	Quit       key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NewPeer, k.DeletePeer, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{}, // first column
		{}, // second column
	}
}

type mainScreenTable struct {
	app   *app.App
	sSize tea.WindowSizeMsg

	dialog           tea.Model
	dialogFullScreen bool

	keys  keyMap
	table *stickers.TableSingleType[string]
	help  help.Model
}

func newMainScreenTable(app *app.App, sSize tea.WindowSizeMsg) mainScreenTable {

	keys := keyMap{
		NewPeer: key.NewBinding(
			key.WithKeys("F7"),
			key.WithHelp("F7", "NewPeer"),
		),
		DeletePeer: key.NewBinding(
			key.WithKeys("F8"),
			key.WithHelp("F8", "DelPeer"),
		),
		Quit: key.NewBinding(
			key.WithKeys("F10"),
			key.WithHelp("F10", "Quit"),
		),
	}

	helpModel := help.New()
	helpModel.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	helpModel.Styles.ShortDesc = lipgloss.NewStyle().Background(lipgloss.Color("14")).Foreground(lipgloss.Color("0")).Width(12)
	helpModel.ShortSeparator = " "

	return mainScreenTable{
		app:   app,
		sSize: sSize,
		keys:  keys,
		help:  helpModel,
		table: newTable(app, sSize),
	}
}

func (m mainScreenTable) Init() tea.Cmd {
	return nil
}

func (m mainScreenTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.sSize = msg
		m.help.Width = msg.Width
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 1)
	}

	switch msg := msg.(type) {

	case TuiDialogMsgResult:
		m.dialogFullScreen = false
		m.dialog = nil

	case TuiDialogYes:
		return m.ReallyDeletePeer(), nil

	case TuiDialogNo:
		m.dialog = nil

	case TuiDialogValue:
		peer := string(msg)
		return m.ReallyAddPeer(peer), nil

	case TuiDialogCancel:
		_ = msg
		m.dialog = nil
		return m, nil
	}

	if m.dialog != nil {
		w, c := m.dialog.Update(msg)
		m.dialog = w
		return m, c
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {

		case tea.KeyF10:
			return m, tea.Quit

		case tea.KeyF7:
			return m.CreatePeer()

		case tea.KeyF8:
			return m.DeletePeer()

		case tea.KeyEnter:
			_, row := m.table.GetCursorLocation()
			if row == 0 {
				m.dialog = NewTuiDialogMsg("Error", "Could not view server")
				return m, m.dialog.Init()
			}

			p := peerRow(m.app, m.table)
			if p == nil {
				return m, nil
			}

			m.dialogFullScreen = true
			m.dialog = NewViewPeer(m.sSize, m.app.State.Server, p)
			return m, m.dialog.Init()

		case tea.KeyDown:
			m.table.CursorDown()

		case tea.KeyUp:
			m.table.CursorUp()

		}
	}
	//m.table, cmd = m.table.
	return m, cmd
}

func (m mainScreenTable) View() string {
	helpView := m.help.View(m.keys)
	mainScreen := lipgloss.JoinVertical(lipgloss.Left, m.table.Render(), helpView)

	if m.dialog != nil {
		if m.dialogFullScreen {
			return m.dialog.View()
		}

		bgs := lipgloss.NewStyle().Background(lipgloss.Color(0)).Foreground(lipgloss.Color("237"))
		return tutils.PlaceDialog(m.dialog.View(), mainScreen, bgs)
	}

	return mainScreen

}

func (m mainScreenTable) CreatePeer() (tea.Model, tea.Cmd) {
	d := NewTuiDialogName()
	d.Title = "Create a new Peer"
	d.Question = "Enter new peer name"
	d.ValidationFunc = func(s string) string {
		if s == "" {
			return "cannot be empty"
		}
		_, err := m.app.State.CanAddPeer(s)
		if err != nil {
			return err.Error()
		}
		return ""
	}
	m.dialog = d
	return m, d.Init()
}

func (m mainScreenTable) DeletePeer() (tea.Model, tea.Cmd) {
	_, row := m.table.GetCursorLocation()
	if row == 0 {
		m.dialog = NewTuiDialogMsg("Error", "Could not delete server")
		return m, m.dialog.Init()
	}

	peer := peerRow(m.app, m.table)
	if peer == nil {
		return m, nil
	}

	m.dialog = NewTuiDialogYesNo("Delete", "Delete peer \""+peer.GetName()+"\"?")
	return m, m.dialog.Init()
}

func (m mainScreenTable) ReallyAddPeer(name string) mainScreenTable {
	err := m.app.State.AddPeer(name)
	if err != nil {
		log.Println("Error adding peer", err)
	}
	if err == nil {
		_ = m.app.GenerateWireguardConfig()
	}
	m.dialog = nil
	m.table = newTable(m.app, m.sSize)
	return m
}

func (m mainScreenTable) ReallyDeletePeer() mainScreenTable {
	_, row := m.table.GetCursorLocation()
	if row == 0 {
		panic("we don't delete server")
	}

	peer := peerRow(m.app, m.table)
	if peer != nil {
		err := m.app.State.DeletePeer(peer.GetIPNumber())
		if err != nil {
			log.Println("Error deleting peer", err)
		}
		if err == nil {
			_ = m.app.GenerateWireguardConfig()
		}

		m.table = newTable(m.app, m.sSize)
	}
	m.dialog = nil
	return m
}
