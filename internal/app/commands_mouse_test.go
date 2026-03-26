package app

import (
	"strings"
	"testing"

	"github.com/vigo999/ms-cli/ui/model"
)

func TestCmdMouseOffEmitsToggleAndDisablesMirrorState(t *testing.T) {
	app := &Application{
		EventCh:      make(chan model.Event, 8),
		mouseEnabled: true,
	}

	app.handleCommand("/mouse off")

	ev := drainUntilEventType(t, app, model.MouseModeToggle)
	if got, want := ev.Message, "off"; got != want {
		t.Fatalf("toggle message = %q, want %q", got, want)
	}
	reply := drainUntilEventType(t, app, model.AgentReply)
	if !strings.Contains(reply.Message, "Mouse scrolling disabled") {
		t.Fatalf("unexpected reply: %q", reply.Message)
	}
	if app.mouseEnabled {
		t.Fatal("expected mouseEnabled mirror state to be false")
	}
}

func TestCmdMouseStatusReflectsCurrentState(t *testing.T) {
	app := &Application{
		EventCh:      make(chan model.Event, 8),
		mouseEnabled: true,
	}

	app.handleCommand("/mouse status")

	reply := drainUntilEventType(t, app, model.AgentReply)
	if !strings.Contains(reply.Message, "Mouse mode is on") || !strings.Contains(reply.Message, "/mouse off") {
		t.Fatalf("unexpected on-status reply: %q", reply.Message)
	}

	app.mouseEnabled = false
	app.handleCommand("/mouse status")

	reply = drainUntilEventType(t, app, model.AgentReply)
	if !strings.Contains(reply.Message, "Mouse mode is off") || !strings.Contains(reply.Message, "selection/copy") {
		t.Fatalf("unexpected off-status reply: %q", reply.Message)
	}
}

func TestCmdHelpIncludesMouseCommandAndCopyTip(t *testing.T) {
	app := &Application{EventCh: make(chan model.Event, 4)}

	app.cmdHelp()

	reply := drainUntilEventType(t, app, model.AgentReply)
	for _, want := range []string{
		"/mouse [on|off|toggle|status]",
		"/mouse off            Restore terminal-native selection/copy",
	} {
		if !strings.Contains(reply.Message, want) {
			t.Fatalf("expected help output to contain %q, got:\n%s", want, reply.Message)
		}
	}
}
