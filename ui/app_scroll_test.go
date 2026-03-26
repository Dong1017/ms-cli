package ui

import (
	"reflect"
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vigo999/ms-cli/ui/model"
)

func TestMainChatMouseWheelScrollKeepsManualPositionOnNewReply(t *testing.T) {
	app := New(nil, nil, "test", ".", "", "demo-model", 4096)
	app.bootActive = false

	next, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app = next.(App)

	for i := 0; i < 12; i++ {
		next, _ = app.handleEvent(model.Event{
			Type:    model.AgentReply,
			Message: "reply " + strconv.Itoa(i),
		})
		app = next.(App)
	}

	if !app.viewport.AtBottom() {
		t.Fatal("expected viewport to start at bottom after initial replies")
	}

	bottomOffset := app.viewport.YOffset()

	next, _ = app.Update(tea.MouseMsg{
		Type:   tea.MouseWheelUp,
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	app = next.(App)

	if app.viewport.AtBottom() {
		t.Fatal("expected mouse wheel up to leave the bottom of chat history")
	}
	if app.viewport.YOffset() >= bottomOffset {
		t.Fatalf("expected wheel up to move viewport upward, bottom offset=%d current=%d", bottomOffset, app.viewport.YOffset())
	}

	manualOffset := app.viewport.YOffset()

	next, _ = app.handleEvent(model.Event{
		Type:    model.AgentReply,
		Message: "new agent result",
	})
	app = next.(App)

	if app.viewport.AtBottom() {
		t.Fatal("expected new agent reply not to force viewport back to bottom while user is reading history")
	}
	if app.viewport.YOffset() != manualOffset {
		t.Fatalf("expected manual scroll position to be preserved, want %d got %d", manualOffset, app.viewport.YOffset())
	}

	for i := 0; i < 20 && !app.viewport.AtBottom(); i++ {
		next, _ = app.Update(tea.MouseMsg{
			Type:   tea.MouseWheelDown,
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		})
		app = next.(App)
	}

	if !app.viewport.AtBottom() {
		t.Fatal("expected mouse wheel down to return viewport to bottom")
	}
}

func TestMainChatKeyboardScrollStillUsesViewport(t *testing.T) {
	app := New(nil, nil, "test", ".", "", "demo-model", 4096)
	app.bootActive = false

	next, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app = next.(App)

	for i := 0; i < 12; i++ {
		next, _ = app.handleEvent(model.Event{
			Type:    model.AgentReply,
			Message: "keyboard scroll regression " + strconv.Itoa(i),
		})
		app = next.(App)
	}

	if !app.viewport.AtBottom() {
		t.Fatal("expected viewport to start at bottom")
	}

	next, _ = app.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	app = next.(App)
	if app.viewport.AtBottom() {
		t.Fatal("expected pgup to scroll away from bottom")
	}

	next, _ = app.handleKey(tea.KeyMsg{Type: tea.KeyEnd})
	app = next.(App)
	if !app.viewport.AtBottom() {
		t.Fatal("expected end to jump back to bottom")
	}
}

func TestMainChatWheelOverInputAreaStillScrollsChat(t *testing.T) {
	app := New(nil, nil, "test", ".", "", "demo-model", 4096)
	app.bootActive = false

	next, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app = next.(App)

	for i := 0; i < 12; i++ {
		next, _ = app.handleEvent(model.Event{
			Type:    model.AgentReply,
			Message: "input area wheel " + strconv.Itoa(i),
		})
		app = next.(App)
	}

	if !app.viewport.AtBottom() {
		t.Fatal("expected viewport to start at bottom")
	}

	next, _ = app.Update(tea.MouseMsg{
		X:      5,
		Y:      app.height - 2,
		Type:   tea.MouseWheelUp,
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	app = next.(App)

	if app.viewport.AtBottom() {
		t.Fatal("expected wheel event over input area to scroll chat history")
	}
}

func TestMouseModeToggleSwitchesRuntimeMouseCommand(t *testing.T) {
	app := New(nil, nil, "test", ".", "", "demo-model", 4096)
	app.bootActive = false

	next, cmd := app.handleEvent(model.Event{Type: model.MouseModeToggle, Message: "off"})
	app = next.(App)

	if app.state.MouseEnabled {
		t.Fatal("expected mouse mode to be disabled")
	}
	if cmd == nil {
		t.Fatal("expected disable mouse command")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("expected batched mouse command, got %T", cmd())
	}
	if got, want := reflect.TypeOf(batch[0]()), reflect.TypeOf(tea.DisableMouse()); got != want {
		t.Fatalf("disable command type = %v, want %v", got, want)
	}

	next, cmd = app.handleEvent(model.Event{Type: model.MouseModeToggle, Message: "on"})
	app = next.(App)

	if !app.state.MouseEnabled {
		t.Fatal("expected mouse mode to be enabled")
	}
	if cmd == nil {
		t.Fatal("expected enable mouse command")
	}
	batch, ok = cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("expected batched mouse command, got %T", cmd())
	}
	if got, want := reflect.TypeOf(batch[0]()), reflect.TypeOf(tea.EnableMouseCellMotion()); got != want {
		t.Fatalf("enable command type = %v, want %v", got, want)
	}
}

func TestMainChatWheelIgnoredWhenMouseModeDisabled(t *testing.T) {
	app := New(nil, nil, "test", ".", "", "demo-model", 4096)
	app.bootActive = false

	next, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	app = next.(App)

	for i := 0; i < 12; i++ {
		next, _ = app.handleEvent(model.Event{
			Type:    model.AgentReply,
			Message: "disabled wheel " + strconv.Itoa(i),
		})
		app = next.(App)
	}

	next, _ = app.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	app = next.(App)
	if app.viewport.AtBottom() {
		t.Fatal("expected pgup to move viewport off bottom before disabling mouse mode")
	}

	manualOffset := app.viewport.YOffset()
	app.state = app.state.WithMouseEnabled(false)

	next, cmd := app.Update(tea.MouseMsg{
		Type:   tea.MouseWheelDown,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	app = next.(App)

	if cmd != nil {
		t.Fatal("expected no mouse command when mouse mode is disabled")
	}
	if got := app.viewport.YOffset(); got != manualOffset {
		t.Fatalf("expected disabled mouse mode to ignore wheel input, want offset %d got %d", manualOffset, got)
	}
}
