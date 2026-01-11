package tui

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// Runner manages a TUI program and provides methods to update it from sync operations.
type Runner struct {
	program *tea.Program
	model   Model
	mu      sync.Mutex
	started bool
}

// NewRunner creates a new TUI runner.
func NewRunner() *Runner {
	return &Runner{
		model: New(),
	}
}

// Start starts the TUI program in a goroutine and returns immediately.
// The program runs until Done() is called.
func (r *Runner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil
	}

	r.program = tea.NewProgram(r.model)
	r.started = true

	go func() {
		_, _ = r.program.Run()
	}()

	return nil
}

// Wait blocks until the TUI program exits.
func (r *Runner) Wait() {
	if r.program != nil {
		r.program.Wait()
	}
}

// AddItem adds a new item to the tree.
func (r *Runner) AddItem(item *SyncItem, parentID string) {
	if r.program != nil {
		r.program.Send(AddItemMsg{Item: item, ParentID: parentID})
	}
}

// AddRoot adds a root-level item (no parent).
func (r *Runner) AddRoot(id, title string, itemType ItemType) {
	r.AddItem(&SyncItem{
		ID:     id,
		Title:  title,
		Type:   itemType,
		Status: StatusPending,
	}, "")
}

// AddChild adds a child item under a parent.
func (r *Runner) AddChild(parentID, id, title string, itemType ItemType) {
	r.AddItem(&SyncItem{
		ID:     id,
		Title:  title,
		Type:   itemType,
		Status: StatusPending,
	}, parentID)
}

// SetSyncing marks an item as currently syncing.
func (r *Runner) SetSyncing(id string) {
	if r.program != nil {
		r.program.Send(UpdateStatusMsg{ID: id, Status: StatusSyncing})
	}
}

// SetDone marks an item as successfully synced.
func (r *Runner) SetDone(id string) {
	if r.program != nil {
		r.program.Send(UpdateStatusMsg{ID: id, Status: StatusDone})
	}
}

// SetError marks an item as failed with an error message.
func (r *Runner) SetError(id, errMsg string) {
	if r.program != nil {
		r.program.Send(UpdateStatusMsg{ID: id, Status: StatusError, Error: errMsg})
	}
}

// Done signals that the sync is complete.
func (r *Runner) Done(err error) {
	if r.program != nil {
		r.program.Send(DoneMsg{Err: err})
	}
}
