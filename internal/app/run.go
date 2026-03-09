package app

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vigo999/ms-cli/agent/loop"
	"github.com/vigo999/ms-cli/ui"
	"github.com/vigo999/ms-cli/ui/model"
)

const provideAPIKeyFirstMsg = "provide api key first"

// Run parses CLI args, wires dependencies, and starts the application.
func Run(args []string) error {
	fs := flag.NewFlagSet("ms-cli", flag.ContinueOnError)
	demo := fs.Bool("demo", false, "Run in demo mode")
	configPath := fs.String("config", "", "Path to config file")
	url := fs.String("url", "", "OpenAI-compatible base URL")
	modelFlag := fs.String("model", "", "Model name")
	apiKey := fs.String("api-key", "", "API key")

	if err := fs.Parse(args); err != nil {
		return err
	}

	app, err := Wire(BootstrapConfig{
		Demo:       *demo,
		ConfigPath: *configPath,
		URL:        *url,
		Model:      *modelFlag,
		Key:        *apiKey,
	})
	if err != nil {
		return err
	}

	return app.run()
}

// run starts the TUI.
func (a *Application) run() error {
	if closer, ok := a.traceWriter.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	if a.Demo {
		return a.runDemo()
	}
	return a.runReal()
}

func (a *Application) runReal() error {
	userCh := make(chan string, 8)
	tui := ui.New(a.EventCh, userCh, Version, a.WorkDir, a.RepoURL, a.Config.Model.Model, a.Config.Context.MaxTokens)
	p := tea.NewProgram(tui, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go a.inputLoop(userCh)

	_, err := p.Run()
	close(userCh)
	return err
}

func (a *Application) inputLoop(userCh <-chan string) {
	for input := range userCh {
		a.processInput(input)
	}
}

func (a *Application) processInput(input string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return
	}

	if strings.HasPrefix(trimmed, "/") {
		a.handleCommand(trimmed)
		return
	}

	go a.runTask(trimmed)
}

func (a *Application) runTask(description string) {
	if !a.llmReady {
		a.EventCh <- model.Event{
			Type:    model.AgentReply,
			Message: provideAPIKeyFirstMsg,
		}
		return
	}

	a.EventCh <- model.Event{Type: model.AgentThinking}

	task := loop.Task{
		ID:          generateTaskID(),
		Description: description,
	}

	events, err := a.Orchestrator.Run(context.Background(), task)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
			errMsg = fmt.Sprintf("%s\n\nTip: The request timed out. Try:\n  1. Run /compact to reduce context size\n  2. Start a new conversation with /clear\n  3. Increase timeout in config (model.timeout_sec)", errMsg)
		}
		a.EventCh <- model.Event{
			Type:     model.ToolError,
			ToolName: "Engine",
			Message:  errMsg,
		}
		return
	}

	for _, ev := range events {
		uiEvent := a.convertEvent(ev)
		if uiEvent != nil {
			a.EventCh <- *uiEvent
		}
	}
}

func (a *Application) convertEvent(ev loop.Event) *model.Event {
	switch ev.Type {
	case loop.EventAgentReply:
		return &model.Event{
			Type: model.AgentReply, Message: ev.Message,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventAgentThinking:
		return &model.Event{
			Type: model.AgentThinking,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventToolRead:
		return &model.Event{
			Type: model.ToolRead, Message: ev.Message, ToolName: ev.ToolName, Summary: ev.Summary,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventToolGrep:
		return &model.Event{
			Type: model.ToolGrep, Message: ev.Message, ToolName: ev.ToolName, Summary: ev.Summary,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventToolGlob:
		return &model.Event{
			Type: model.ToolGlob, Message: ev.Message, ToolName: ev.ToolName, Summary: ev.Summary,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventToolEdit:
		return &model.Event{
			Type: model.ToolEdit, Message: ev.Message, ToolName: ev.ToolName,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventToolWrite:
		return &model.Event{
			Type: model.ToolWrite, Message: ev.Message, ToolName: ev.ToolName,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventToolError:
		return &model.Event{
			Type: model.ToolError, Message: ev.Message, ToolName: ev.ToolName,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventCmdStarted:
		return &model.Event{
			Type: model.CmdStarted, Message: ev.Message,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventAnalysisReady:
		return &model.Event{
			Type: model.AnalysisReady, Message: ev.Message,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventTokenUpdate:
		return &model.Event{
			Type: model.TokenUpdate,
			CtxUsed: ev.CtxUsed, CtxMax: ev.CtxMax, TokensUsed: ev.TokensUsed,
		}
	case loop.EventTaskCompleted:
		return nil
	case loop.EventTaskFailed:
		return &model.Event{
			Type: model.ToolError, ToolName: "Task", Message: ev.Message,
		}
	default:
		if ev.Message != "" {
			return &model.Event{Type: model.AgentReply, Message: ev.Message}
		}
		return nil
	}
}

func generateTaskID() string {
	return time.Now().Format("20060102-150405-000")
}

func (a *Application) runDemo() error {
	go a.fakeAgentLoop()
	tui := ui.New(a.EventCh, nil, Version, a.WorkDir, a.RepoURL, "demo-model", a.Config.Context.MaxTokens)
	p := tea.NewProgram(tui, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
