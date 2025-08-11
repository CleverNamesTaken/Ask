package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeSearch mode = iota
	modeSelect
)

type model struct {
	table     table.Model
	textInput textinput.Model
	allRows   []table.Row
	mode      mode
	selected  *table.Row
}

func initialModel() model {
	columns := []table.Column{
		{Title: "Country", Width: 20},
		{Title: "Capital", Width: 20},
		{Title: "Population", Width: 10},
	}
	rows := []table.Row{
		{"France", "Paris", "67M"},
		{"Germany", "Berlin", "83M"},
		{"Italy", "Rome", "60M"},
		{"Spain", "Madrid", "47M"},
		{"Portugal", "Lisbon", "10M"},
	}

	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Focus()

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
	)

	return model{
		table:     tbl,
		textInput: ti,
		allRows:   rows,
		mode:      modeSearch,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
	case modeSearch:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q", "esc":
				return m, tea.Quit
			case "tab", "enter", "down":
				// Switch to select mode
				m.mode = modeSelect
				m.textInput.Blur()
				return m, nil
			}
			// Update text input and filter table rows
			m.textInput, cmd = m.textInput.Update(msg)
			query := strings.ToLower(m.textInput.Value())
			var filtered []table.Row
			for _, row := range m.allRows {
				for _, cell := range row {
					if strings.Contains(strings.ToLower(cell), query) {
						filtered = append(filtered, row)
						break
					}
				}
			}
			m.table.SetRows(filtered)
		}
	case modeSelect:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			fmt.Printf("key: %q\n", msg.String())
			switch msg.String() {
			case "j":
				// Simulate down arrow
				m.table, _ = m.table.Update(tea.KeyMsg{Type: tea.KeyDown})
				return m, nil
			case "k":
				// Simulate up arrow
				m.table, _ = m.table.Update(tea.KeyMsg{Type: tea.KeyUp})
				return m, nil
			case "ctrl+c", "q", "esc":
				return m, tea.Quit
			case "/":
				// Back to search mode
				m.mode = modeSearch
				m.textInput.Focus()
				return m, nil
			case "enter":
				// Select current row
				row := m.table.SelectedRow()
				m.selected = &row
				return m, tea.Quit
			}
		}
		// Let the table handle navigation keys in select mode
		m.table, _ = m.table.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	s := ""
	if m.mode == modeSearch {
		s += "Search: " + m.textInput.View() + "\n\n"
	} else {
		s += "Press / to search again. Use arrows/Enter to select.\n\n"
	}
	s += m.table.View()
	if m.selected != nil {
		s += fmt.Sprintf("\n\nSelected: %v", *m.selected)
	}
	s += "\n\nPress q or esc to quit."
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
	// Type assert to your model type
	if m, ok := finalModel.(model); ok && m.selected != nil {
		fmt.Printf("\nYou selected: %v\n", *m.selected)
	}
}
