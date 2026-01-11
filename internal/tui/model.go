// Package tui provides a terminal user interface for displaying sync progress.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ItemStatus represents the sync status of an item.
type ItemStatus int

const (
	StatusPending ItemStatus = iota
	StatusSyncing
	StatusDone
	StatusError
)

// ItemType distinguishes between pages and databases.
type ItemType int

const (
	TypePage ItemType = iota
	TypeDatabase
)

// SyncItem represents a page or database being synced.
type SyncItem struct {
	ID       string
	Title    string
	Type     ItemType
	Status   ItemStatus
	Error    string
	Children []*SyncItem
	Depth    int
}

// Model is the Bubble Tea model for the sync TUI.
type Model struct {
	items    []*SyncItem
	spinner  spinner.Model
	done     bool
	err      error
	quitting bool

	// Styles
	titleStyle   lipgloss.Style
	idStyle      lipgloss.Style
	doneStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	pendingStyle lipgloss.Style
}

// Messages for updating the TUI from sync operations.
type (
	// AddItemMsg adds a new item to the tree.
	AddItemMsg struct {
		Item     *SyncItem
		ParentID string // Empty for root items
	}

	// UpdateStatusMsg updates the status of an item.
	UpdateStatusMsg struct {
		ID     string
		Status ItemStatus
		Error  string
	}

	// DoneMsg signals that sync is complete.
	DoneMsg struct {
		Err error
	}
)

// New creates a new TUI model.
func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		items:   make([]*SyncItem, 0),
		spinner: s,

		titleStyle:   lipgloss.NewStyle().Bold(true),
		idStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		doneStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		pendingStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case AddItemMsg:
		if msg.ParentID == "" {
			m.items = append(m.items, msg.Item)
		} else {
			addChildItem(m.items, msg.ParentID, msg.Item)
		}
		return m, nil

	case UpdateStatusMsg:
		updateItemStatus(m.items, msg.ID, msg.Status, msg.Error)
		return m, nil

	case DoneMsg:
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

// View renders the UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n")
	for _, item := range m.items {
		m.renderItem(&b, item)
	}

	if m.done {
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(m.errorStyle.Render("✗ Sync failed: " + m.err.Error()))
		} else {
			b.WriteString(m.doneStyle.Render("✓ Sync complete"))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderItem(b *strings.Builder, item *SyncItem) {
	indent := strings.Repeat("  ", item.Depth)
	prefix := ""

	// Tree connectors for children
	if item.Depth > 0 {
		prefix = "├── "
	}

	// Icon based on type
	icon := "📄"
	if item.Type == TypeDatabase {
		icon = "📚"
	}

	// Status indicator
	status := ""
	switch item.Status {
	case StatusPending:
		status = m.pendingStyle.Render("pending")
	case StatusSyncing:
		status = m.spinner.View() + " syncing..."
	case StatusDone:
		status = m.doneStyle.Render("✓")
	case StatusError:
		status = m.errorStyle.Render("✗ " + item.Error)
	}

	// Truncate ID for display
	shortID := item.ID
	if len(shortID) > 12 {
		shortID = shortID[:8] + "..."
	}

	// Render the line
	fmt.Fprintf(b, "%s%s%s %s %s  %s\n",
		indent,
		prefix,
		icon,
		m.titleStyle.Render(item.Title),
		m.idStyle.Render("("+shortID+")"),
		status,
	)

	// Render children
	for i, child := range item.Children {
		// Update prefix for last child
		if i == len(item.Children)-1 {
			child.Depth = item.Depth + 1
		}
		m.renderItem(b, child)
	}
}

// Helper functions

func addChildItem(items []*SyncItem, parentID string, child *SyncItem) {
	for _, item := range items {
		if item.ID == parentID {
			child.Depth = item.Depth + 1
			item.Children = append(item.Children, child)
			return
		}
		addChildItem(item.Children, parentID, child)
	}
}

func updateItemStatus(items []*SyncItem, id string, status ItemStatus, errMsg string) {
	for _, item := range items {
		if item.ID == id {
			item.Status = status
			item.Error = errMsg
			return
		}
		updateItemStatus(item.Children, id, status, errMsg)
	}
}

// Items returns all root items.
func (m *Model) Items() []*SyncItem {
	return m.items
}
