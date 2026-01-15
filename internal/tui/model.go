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
	ID     string
	Title  string
	Icon   string // Emoji icon from Notion (empty if not set)
	Type   ItemType
	Status ItemStatus
	Error  string
}

// maxRecentItems is the number of recent completed/error items to show.
const maxRecentItems = 5

// Model is the Bubble Tea model for the sync TUI.
type Model struct {
	// All items indexed by ID for quick lookup
	items map[string]*SyncItem

	// Counts for progress display
	pendingCount int
	syncingCount int
	doneCount    int
	errorCount   int
	totalCount   int

	// Currently syncing items (limited by worker pool size)
	syncingItems []*SyncItem

	// Recent completed/error items (scrolling buffer)
	recentItems []*SyncItem

	spinner  spinner.Model
	done     bool
	err      error
	quitting bool

	// Styles
	titleStyle    lipgloss.Style
	headerStyle   lipgloss.Style
	countStyle    lipgloss.Style
	doneStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	pendingStyle  lipgloss.Style
	syncingStyle  lipgloss.Style
	progressStyle lipgloss.Style
	dimStyle      lipgloss.Style
}

// Messages for updating the TUI from sync operations.
type (
	// AddItemMsg adds a new item (starts as pending).
	AddItemMsg struct {
		Item *SyncItem
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
		items:        make(map[string]*SyncItem),
		syncingItems: make([]*SyncItem, 0),
		recentItems:  make([]*SyncItem, 0),
		spinner:      s,

		titleStyle:    lipgloss.NewStyle().Bold(true),
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")),
		countStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		doneStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		pendingStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		syncingStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		progressStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		dimStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
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
		item := msg.Item
		item.Status = StatusPending
		m.items[item.ID] = item
		m.pendingCount++
		m.totalCount++
		return m, nil

	case UpdateStatusMsg:
		item, ok := m.items[msg.ID]
		if !ok {
			return m, nil
		}

		oldStatus := item.Status
		newStatus := msg.Status
		item.Status = newStatus
		item.Error = msg.Error

		// Update counts
		switch oldStatus {
		case StatusPending:
			m.pendingCount--
		case StatusSyncing:
			m.syncingCount--
		case StatusDone:
			m.doneCount--
		case StatusError:
			m.errorCount--
		}

		switch newStatus {
		case StatusPending:
			m.pendingCount++
		case StatusSyncing:
			m.syncingCount++
		case StatusDone:
			m.doneCount++
		case StatusError:
			m.errorCount++
		}

		// Update syncing items list
		if oldStatus == StatusSyncing && newStatus != StatusSyncing {
			// Remove from syncing list
			m.syncingItems = removeFromSlice(m.syncingItems, item)
		}
		if newStatus == StatusSyncing && oldStatus != StatusSyncing {
			// Add to syncing list
			m.syncingItems = append(m.syncingItems, item)
		}

		// Add to recent items if completed or errored
		if newStatus == StatusDone || newStatus == StatusError {
			m.recentItems = append(m.recentItems, item)
			// Keep only the last N items
			if len(m.recentItems) > maxRecentItems {
				m.recentItems = m.recentItems[len(m.recentItems)-maxRecentItems:]
			}
		}

		return m, nil

	case DoneMsg:
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

// removeFromSlice removes an item from a slice by pointer.
func removeFromSlice(slice []*SyncItem, item *SyncItem) []*SyncItem {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// View renders the UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString("\n")
	b.WriteString(m.headerStyle.Render("Syncing Notion to Obsidian"))
	b.WriteString("\n")

	// Progress bar
	completed := m.doneCount + m.errorCount
	if m.totalCount > 0 {
		percent := float64(completed) / float64(m.totalCount) * 100
		barWidth := 40
		filledWidth := int(float64(barWidth) * float64(completed) / float64(m.totalCount))
		if filledWidth > barWidth {
			filledWidth = barWidth
		}

		bar := strings.Repeat("━", filledWidth) + strings.Repeat("─", barWidth-filledWidth)
		b.WriteString(m.progressStyle.Render(bar))
		b.WriteString(fmt.Sprintf(" %.0f%% (%d/%d)\n", percent, completed, m.totalCount))
	}

	// Status counts
	counts := fmt.Sprintf("Pending: %d  Syncing: %d  Done: %d  Errors: %d",
		m.pendingCount, m.syncingCount, m.doneCount, m.errorCount)
	b.WriteString(m.countStyle.Render(counts))
	b.WriteString("\n\n")

	// Currently syncing section
	if len(m.syncingItems) > 0 {
		b.WriteString(m.dimStyle.Render("Currently syncing:"))
		b.WriteString("\n")
		for _, item := range m.syncingItems {
			b.WriteString(m.renderSyncingItem(item))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Recent items section
	if len(m.recentItems) > 0 {
		b.WriteString(m.dimStyle.Render("Recent:"))
		b.WriteString("\n")
		for _, item := range m.recentItems {
			b.WriteString(m.renderRecentItem(item))
			b.WriteString("\n")
		}
	}

	// Completion message
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

// renderSyncingItem renders a currently syncing item.
func (m Model) renderSyncingItem(item *SyncItem) string {
	icon := item.Icon
	if icon == "" {
		icon = "📄"
		if item.Type == TypeDatabase {
			icon = "📚"
		}
	}

	// Truncate title if too long
	title := item.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	// Truncate ID for display
	shortID := item.ID
	if len(shortID) > 12 {
		shortID = shortID[:8] + "..."
	}

	return fmt.Sprintf("  %s %s %s %s",
		m.spinner.View(),
		icon,
		m.titleStyle.Render(title),
		m.dimStyle.Render("("+shortID+")"),
	)
}

// renderRecentItem renders a completed or errored item.
func (m Model) renderRecentItem(item *SyncItem) string {
	icon := item.Icon
	if icon == "" {
		icon = "📄"
		if item.Type == TypeDatabase {
			icon = "📚"
		}
	}

	// Truncate title if too long
	title := item.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	var status string
	switch item.Status {
	case StatusDone:
		status = m.doneStyle.Render("✓")
	case StatusError:
		errMsg := item.Error
		if len(errMsg) > 30 {
			errMsg = errMsg[:27] + "..."
		}
		status = m.errorStyle.Render("✗ " + errMsg)
	}

	return fmt.Sprintf("  %s %s %s %s",
		status,
		icon,
		m.dimStyle.Render(title),
		"",
	)
}

// Items returns all items.
func (m *Model) Items() map[string]*SyncItem {
	return m.items
}
