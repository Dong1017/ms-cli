package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vigo999/ms-cli/ui/slash"
)

const largePastedBlock = "line 01\nline 02\nline 03\nline 04\nline 05\nline 06\nline 07\nline 08\n"

func TestTextInputHistoryRecall(t *testing.T) {
	input := NewTextInput()
	input = input.PushHistory("first prompt")
	input = input.PushHistory("second prompt")
	input.Model.SetValue("draft")
	input.Model.SetCursor(len("draft"))

	input = input.PrevHistory()
	if got := input.Value(); got != "second prompt" {
		t.Fatalf("expected latest history entry, got %q", got)
	}

	input = input.PrevHistory()
	if got := input.Value(); got != "first prompt" {
		t.Fatalf("expected previous history entry, got %q", got)
	}

	input = input.NextHistory()
	if got := input.Value(); got != "second prompt" {
		t.Fatalf("expected forward history entry, got %q", got)
	}

	input = input.NextHistory()
	if got := input.Value(); got != "draft" {
		t.Fatalf("expected draft restoration after leaving history, got %q", got)
	}
}

func TestTextInputHistoryDoesNotBreakSlashSuggestions(t *testing.T) {
	input := NewTextInput()
	input = input.PushHistory("/project")
	var cmd tea.Cmd
	input, cmd = input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	_ = cmd
	if !input.IsSlashMode() {
		t.Fatal("expected slash suggestions after typing slash")
	}
}

func TestTextInputCtrlJInsertsNewline(t *testing.T) {
	input := NewTextInput()
	if got := input.Height(); got != 3 {
		t.Fatalf("expected single-row composer block height 3, got %d", got)
	}

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if got := input.Value(); got != "\n" {
		t.Fatalf("expected ctrl+j to insert newline, got %q", got)
	}
	if got := input.Height(); got != 4 {
		t.Fatalf("expected two-row composer block height 4, got %d", got)
	}

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})

	if got := input.Value(); got != "\n\n" {
		t.Fatalf("expected second ctrl+j to insert another newline, got %q", got)
	}
	if got := input.Height(); got != 5 {
		t.Fatalf("expected three-row composer block height 5, got %d", got)
	}
}

func TestTextInputUsesSinglePromptWithContinuationLines(t *testing.T) {
	input := NewTextInput()
	input = input.SetWidth(40)
	input.Model.SetValue("one\ntwo\nthree")
	input.syncHeight()

	view := input.View()
	if got := strings.Count(view, composerPrompt); got != 1 {
		t.Fatalf("expected one primary prompt, got %d in view %q", got, view)
	}
	if got := strings.Count(view, composerContinue+"two"); got != 1 {
		t.Fatalf("expected continuation prompt for second line, got view %q", view)
	}
	if got := strings.Count(view, composerContinue+"three"); got != 1 {
		t.Fatalf("expected continuation prompt for third line, got view %q", view)
	}
}

func TestTextInputKeepsFirstLineVisibleAfterExplicitNewlineGrowth(t *testing.T) {
	input := NewTextInput()
	input = input.SetWidth(40)
	input.Model.SetValue("alpha")

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("beta")})

	view := input.View()
	if !strings.Contains(view, composerPrompt+"alpha") {
		t.Fatalf("expected first line to remain visible after newline growth, got %q", view)
	}
	if !strings.Contains(view, composerContinue+"beta") {
		t.Fatalf("expected second line to remain visible after newline growth, got %q", view)
	}
}

func TestTextInputLargePasteShowsCollapsedSummary(t *testing.T) {
	input := NewTextInput()
	input = input.SetWidth(60)

	input, _ = input.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(largePastedBlock),
		Paste: true,
	})

	if !input.HasPasteSummary() {
		t.Fatal("expected large paste to enter collapsed summary mode")
	}
	if got := input.Height(); got != 3 {
		t.Fatalf("expected collapsed paste preview height 3, got %d", got)
	}

	view := input.View()
	if !strings.Contains(view, "[pasted content:") {
		t.Fatalf("expected collapsed paste summary in view, got %q", view)
	}
	if strings.Contains(view, "line 07") {
		t.Fatalf("expected full pasted content to stay hidden in collapsed view, got %q", view)
	}
}

func TestTextInputLargePasteStaysCollapsedAfterTyping(t *testing.T) {
	input := NewTextInput()
	input = input.SetWidth(60)

	input, _ = input.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(largePastedBlock),
		Paste: true,
	})
	input, _ = input.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{' '},
	})

	view := input.View()
	if !strings.Contains(view, "[pasted content:") {
		t.Fatalf("expected collapsed paste summary to remain after typing, got %q", view)
	}
	if strings.Contains(view, "line 07") {
		t.Fatalf("expected full pasted content to stay hidden after typing, got %q", view)
	}
	if got := input.Value(); got != largePastedBlock+" " {
		t.Fatalf("expected reconstructed value to preserve raw paste and typed suffix, got %q", got)
	}
}

func TestTextInputHistoryRecallOfSlashCommandDoesNotReopenSuggestions(t *testing.T) {
	input := NewTextInput()
	input = input.PushHistory("/project")
	input = input.PushHistory("hello")

	input = input.PrevHistory()
	if got := input.Value(); got != "hello" {
		t.Fatalf("expected latest history entry, got %q", got)
	}

	input = input.PrevHistory()
	if got := input.Value(); got != "/project" {
		t.Fatalf("expected slash command from history, got %q", got)
	}
	if input.IsSlashMode() {
		t.Fatal("expected slash suggestions to stay closed while browsing history")
	}

	input = input.NextHistory()
	if got := input.Value(); got != "hello" {
		t.Fatalf("expected down to continue history recall even in slash mode, got %q", got)
	}
}

func TestTextInputHistoryOnlyTriggersAtEditorBoundaries(t *testing.T) {
	input := NewTextInput()
	input.Model.SetValue("first line\nsecond line")
	input.syncHeight()

	if input.CanNavigateHistory("up") {
		t.Fatal("expected up to stay inside multiline editor while on the last line")
	}

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyUp})

	if !input.CanNavigateHistory("up") {
		t.Fatal("expected up history at the top boundary of the editor")
	}
	if input.CanNavigateHistory("down") {
		t.Fatal("expected down history to stay disabled away from the bottom boundary")
	}

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyDown})

	if !input.CanNavigateHistory("down") {
		t.Fatal("expected down history at the bottom boundary of multiline input")
	}
}

func TestTextInputSuggestionsScrollDownToKeepSelectionVisible(t *testing.T) {
	input := newSlashSuggestionInput(10)
	input.selectedIdx = 7
	input.suggestionOffset = 0

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyDown})

	if input.selectedIdx != 8 {
		t.Fatalf("expected selected index 8, got %d", input.selectedIdx)
	}
	if input.suggestionOffset != 1 {
		t.Fatalf("expected suggestion offset 1, got %d", input.suggestionOffset)
	}

	view := input.View()
	if !strings.Contains(view, "/cmd08") {
		t.Fatalf("expected view to include newly selected command, got %q", view)
	}
	if strings.Contains(view, "/cmd00") {
		t.Fatalf("expected first command to scroll out of view, got %q", view)
	}
}

func TestTextInputSuggestionsScrollUpToKeepSelectionVisible(t *testing.T) {
	input := newSlashSuggestionInput(10)
	input.selectedIdx = 2
	input.suggestionOffset = 2

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyUp})

	if input.selectedIdx != 1 {
		t.Fatalf("expected selected index 1, got %d", input.selectedIdx)
	}
	if input.suggestionOffset != 1 {
		t.Fatalf("expected suggestion offset 1, got %d", input.suggestionOffset)
	}

	view := input.View()
	if !strings.Contains(view, "/cmd01") {
		t.Fatalf("expected view to include newly selected command, got %q", view)
	}
	if strings.Contains(view, "/cmd09") {
		t.Fatalf("expected last command to scroll out of view, got %q", view)
	}
}

func TestTextInputSuggestionsWrapDownToTopPage(t *testing.T) {
	input := newSlashSuggestionInput(10)
	input.selectedIdx = 9
	input.suggestionOffset = 2

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyDown})

	if input.selectedIdx != 0 {
		t.Fatalf("expected selected index 0 after wrap, got %d", input.selectedIdx)
	}
	if input.suggestionOffset != 0 {
		t.Fatalf("expected suggestion offset 0 after wrap, got %d", input.suggestionOffset)
	}

	view := input.View()
	if !strings.Contains(view, "/cmd00") {
		t.Fatalf("expected wrapped view to include first command, got %q", view)
	}
	if strings.Contains(view, "/cmd09") {
		t.Fatalf("expected last command to leave view after wrapping to top, got %q", view)
	}
}

func TestTextInputSuggestionsWrapUpToLastPage(t *testing.T) {
	input := newSlashSuggestionInput(10)
	input.selectedIdx = 0
	input.suggestionOffset = 0

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyUp})

	if input.selectedIdx != 9 {
		t.Fatalf("expected selected index 9 after wrap, got %d", input.selectedIdx)
	}
	if input.suggestionOffset != 2 {
		t.Fatalf("expected suggestion offset 2 after wrap, got %d", input.suggestionOffset)
	}

	view := input.View()
	if !strings.Contains(view, "/cmd09") {
		t.Fatalf("expected wrapped view to include last command, got %q", view)
	}
	if strings.Contains(view, "/cmd00") {
		t.Fatalf("expected first command to leave view after wrapping to bottom, got %q", view)
	}
}

func TestTextInputShowsFileSuggestionsForAtToken(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFile(t, root, "ctx.txt")
	writeSuggestionFile(t, root, "sub/first.md")
	writeSuggestionFile(t, root, "sub/final.md")

	input := NewTextInput().WithFileSuggestions(root)
	input.Model.SetValue("read @sub/f")
	input.Model.SetCursor(len("read @sub/f"))
	input.updateSuggestions()

	if !input.HasSuggestions() {
		t.Fatal("expected file suggestions for @ token")
	}
	if input.IsSlashMode() {
		t.Fatal("expected @file suggestions not to report slash mode")
	}
	if len(input.suggestionItems) != 2 {
		t.Fatalf("expected 2 matching file suggestions, got %d", len(input.suggestionItems))
	}
	if got := input.suggestionItems[0].Value; got != "sub/final.md" {
		t.Fatalf("expected lexicographic file suggestion, got %q", got)
	}
}

func TestTextInputEnterAcceptsFileSuggestionWithoutSubmitting(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFile(t, root, "ctx.txt")

	input := NewTextInput().WithFileSuggestions(root)
	input.Model.SetValue("read @ct")
	input.Model.SetCursor(len("read @ct"))
	input.updateSuggestions()

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := input.Value(); got != "read @ctx.txt " {
		t.Fatalf("expected enter to accept file suggestion, got %q", got)
	}
	if input.HasSuggestions() {
		t.Fatal("expected suggestions cleared after accepting file suggestion")
	}
}

func TestTextInputReplacesOnlyCurrentTokenWhenApplyingFileSuggestion(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFile(t, root, "ctx.txt")

	input := NewTextInput().WithFileSuggestions(root)
	input.Model.SetValue("before @ct after")
	input.Model.SetCursor(len("before @ct"))
	input.updateSuggestions()

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := input.Value(); got != "before @ctx.txt after" {
		t.Fatalf("expected current token replacement only, got %q", got)
	}
}

func TestTextInputLeavesInvalidAtFormsWithoutSuggestions(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFile(t, root, "ctx.txt")

	cases := []string{
		"read @@ctx",
		"read @ctx.txt,",
		"read (@ctx.txt)",
	}

	for _, tc := range cases {
		input := NewTextInput().WithFileSuggestions(root)
		input.Model.SetValue(tc)
		input.Model.SetCursor(len(tc))
		input.updateSuggestions()
		if input.HasSuggestions() {
			t.Fatalf("expected no suggestions for %q", tc)
		}
	}
}

func TestTextInputReplacesCurrentTokenOnSecondLine(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFile(t, root, "ctx.txt")

	input := NewTextInput().WithFileSuggestions(root)
	input.Model.SetValue("line one\nread @ct")
	input.Model.SetCursor(len("line one\nread @ct"))
	input.updateSuggestions()

	input, _ = input.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := input.Value(); got != "line one\nread @ctx.txt " {
		t.Fatalf("expected second-line token replacement, got %q", got)
	}
}

func TestTextInputPrefersFileSuggestionsInSlashCommandArguments(t *testing.T) {
	root := t.TempDir()
	writeSuggestionFile(t, root, "ctx.txt")

	input := NewTextInput().WithFileSuggestions(root)
	input.Model.SetValue("/report accuracy @ct")
	input.Model.SetCursor(len("/report accuracy @ct"))
	input.updateSuggestions()

	if !input.HasSuggestions() {
		t.Fatal("expected file suggestions in slash command arguments")
	}
	if input.IsSlashMode() {
		t.Fatal("expected file suggestions to override slash suggestions outside the command token")
	}
}

func newSlashSuggestionInput(count int) TextInput {
	input := NewTextInput()
	registry := slash.NewRegistry()
	input.slashRegistry = registry
	input.suggestionKind = suggestionKindSlash
	input.suggestionItems = make([]suggestionItem, 0, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("/cmd%02d", i)
		registry.Register(slash.Command{
			Name:        name,
			Description: fmt.Sprintf("Command %02d", i),
			Usage:       name,
		})
		input.suggestionItems = append(input.suggestionItems, suggestionItem{
			Value:       name,
			Display:     name,
			Description: fmt.Sprintf("Command %02d", i),
			Kind:        suggestionKindSlash,
		})
	}

	return input
}

func writeSuggestionFile(t *testing.T, root, relative string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(relative), 0o644); err != nil {
		t.Fatal(err)
	}
}
