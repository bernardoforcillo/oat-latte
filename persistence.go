package oat

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// persistScrollable is a local detection interface for components that expose
// scroll-offset state. Note that the package-level Scrollable interface also
// requires ContentHeight(); this narrower interface is intentionally minimal so
// that any future widget with just ScrollOffset/ScrollTo qualifies.
type persistScrollable interface {
	ScrollOffset() int
	ScrollTo(int)
}

// persistTabbed is a local detection interface for components that expose a
// selectable active-index (e.g. Tabs).
type persistTabbed interface {
	ActiveIndex() int
	SetActive(int)
}

// idGetter is satisfied by any component embedding BaseComponent (which
// provides GetID() via the IDer interface defined in component.go).
type idGetter interface {
	GetID() string
}

// UIState holds the serialisable state of all stateful components, keyed by
// component ID. Only components with non-empty IDs are persisted. Maps are
// omitted from the JSON output when empty to keep the file tidy.
type UIState struct {
	ScrollOffsets map[string]int `json:"scroll_offsets,omitempty"`
	ActiveTabs    map[string]int `json:"active_tabs,omitempty"`
}

// getComponentID returns the component's ID string, or "" if the component
// does not implement the idGetter (GetID) interface.
func getComponentID(c Component) string {
	if ig, ok := c.(idGetter); ok {
		return ig.GetID()
	}
	return ""
}

// collectState walks the component tree rooted at root depth-first and
// populates state with the current UI values of every identified component.
func collectState(root Component, state *UIState) {
	if root == nil {
		return
	}

	id := getComponentID(root)
	if id != "" {
		if ps, ok := root.(persistScrollable); ok {
			state.ScrollOffsets[id] = ps.ScrollOffset()
		}
		if pt, ok := root.(persistTabbed); ok {
			state.ActiveTabs[id] = pt.ActiveIndex()
		}
	}

	// Recurse into children if this component is a Layout.
	if l, ok := root.(Layout); ok {
		for _, child := range l.Children() {
			collectState(child, state)
		}
	}
}

// applyState walks the component tree rooted at root depth-first and restores
// UI state from the provided UIState snapshot.
func applyState(root Component, state *UIState) {
	if root == nil {
		return
	}

	id := getComponentID(root)
	if id != "" {
		if ps, ok := root.(persistScrollable); ok {
			if offset, exists := state.ScrollOffsets[id]; exists {
				ps.ScrollTo(offset)
			}
		}
		if pt, ok := root.(persistTabbed); ok {
			if active, exists := state.ActiveTabs[id]; exists {
				pt.SetActive(active)
			}
		}
	}

	// Recurse into children if this component is a Layout.
	if l, ok := root.(Layout); ok {
		for _, child := range l.Children() {
			applyState(child, state)
		}
	}
}

// writeState serialises state to path, creating any missing parent directories.
// Returns nil without writing if there is nothing to persist.
func writeState(state *UIState, path string) error {
	if len(state.ScrollOffsets) == 0 && len(state.ActiveTabs) == 0 {
		return nil // nothing to save
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// readState deserialises a UIState from path.
// Returns (nil, nil) when the file does not exist so callers can treat a
// missing state file as a normal first-run condition.
func readState(path string) (*UIState, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state UIState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveState walks the component tree rooted at root, collects state from all
// components with non-empty IDs, and writes the result as JSON to path.
// Parent directories are created automatically if they do not exist.
// Returns nil when there is nothing to persist (empty state).
func SaveState(root Component, path string) error {
	state := &UIState{
		ScrollOffsets: make(map[string]int),
		ActiveTabs:    make(map[string]int),
	}
	collectState(root, state)
	return writeState(state, path)
}

// LoadState reads a JSON state file from path and applies it to the component
// tree rooted at root.
// A missing or unreadable file is silently ignored (returns nil).
func LoadState(root Component, path string) error {
	state, err := readState(path)
	if err != nil || state == nil {
		return err
	}
	applyState(root, state)
	return nil
}

// SaveStateMulti is like SaveState but accepts multiple root components
// (e.g. a canvas header, body, and footer). All roots are walked and their
// state is merged into a single file.
func SaveStateMulti(path string, roots ...Component) error {
	state := &UIState{
		ScrollOffsets: make(map[string]int),
		ActiveTabs:    make(map[string]int),
	}
	for _, root := range roots {
		collectState(root, state)
	}
	return writeState(state, path)
}

// LoadStateMulti is like LoadState but accepts multiple root components.
// State is read once from path and applied to every root.
func LoadStateMulti(path string, roots ...Component) error {
	state, err := readState(path)
	if err != nil || state == nil {
		return err
	}
	for _, root := range roots {
		applyState(root, state)
	}
	return nil
}
