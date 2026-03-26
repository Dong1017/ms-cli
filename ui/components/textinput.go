package components

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	uirender "github.com/vigo999/ms-cli/ui/render"
	"github.com/vigo999/ms-cli/ui/slash"
)

var (
	sugCmdStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sugDescStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sugSelCmdStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	sugSelDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	separatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	composerStyle   = lipgloss.NewStyle().PaddingLeft(2)
)

const (
	maxVisibleSuggestions = 8
	minComposerRows       = 1
	composerPrompt        = "❯ "
	composerContinue      = "  "
)

type suggestionKind int

const (
	suggestionKindNone suggestionKind = iota
	suggestionKindSlash
	suggestionKindFile
)

type suggestionItem struct {
	Value       string
	Display     string
	Description string
	Kind        suggestionKind
}

type tokenRange struct {
	start int
	end   int
}

// TextInput wraps a multiline textarea for the chat composer.
type TextInput struct {
	Model            textarea.Model
	slashRegistry    *slash.Registry
	fileSuggestion   *fileSuggestionProvider
	suggestionKind   suggestionKind
	suggestionItems  []suggestionItem
	selectedIdx      int
	suggestionOffset int
	activeToken      tokenRange
	history          []string
	historyIndex     int
	historyDraft     string
	maskedPasteRaw   string
	maskedPasteLabel string
	width            int
}

// NewTextInput creates a focused multiline composer with a prompt.
func NewTextInput() TextInput {
	ti := textarea.New()
	ti.ShowLineNumbers = false
	ti.CharLimit = 0
	ti.MaxHeight = 0
	ti.Prompt = composerPrompt
	ti.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("shift+enter", "ctrl+j"),
		key.WithHelp("shift+enter", "insert newline"),
	)
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ti.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ti.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	ti.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ti.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ti.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ti.SetPromptFunc(lipgloss.Width(composerPrompt), func(line int) string {
		if line == 0 {
			return composerPrompt
		}
		return composerContinue
	})
	ti.Focus()

	input := TextInput{
		Model:         ti,
		slashRegistry: slash.DefaultRegistry,
		historyIndex:  -1,
	}
	input.syncHeight()
	return input
}

// WithFileSuggestions enables @file suggestions from the given workspace root.
func (t TextInput) WithFileSuggestions(workDir string) TextInput {
	t.fileSuggestion = newFileSuggestionProvider(workDir)
	return t
}

// Value returns the current input text.
func (t TextInput) Value() string {
	value := t.Model.Value()
	if t.maskedPasteLabel == "" {
		return value
	}
	return strings.Replace(value, t.maskedPasteLabel, t.maskedPasteRaw, 1)
}

// Reset clears the input.
func (t TextInput) Reset() TextInput {
	t.Model.Reset()
	t = t.clearTransientState().clearSuggestions()
	t.historyIndex = -1
	t.historyDraft = ""
	return t
}

// Focus gives the input focus.
func (t TextInput) Focus() (TextInput, tea.Cmd) {
	cmd := t.Model.Focus()
	return t, cmd
}

// Blur removes focus from the input.
func (t TextInput) Blur() TextInput {
	t.Model.Blur()
	return t
}

// SetWidth updates the rendered input width.
func (t TextInput) SetWidth(width int) TextInput {
	if width < 1 {
		width = 1
	}
	t.Model.SetWidth(width)
	t.width = width
	t.syncHeight()
	return t
}

// Update handles key events.
func (t TextInput) Update(msg tea.Msg) (TextInput, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Paste {
			if summary, ok := uirender.SummarizeLargePaste(string(msg.Runes)); ok {
				t.Model.InsertString(summary)
				t.maskedPasteRaw = string(msg.Runes)
				t.maskedPasteLabel = summary
				t.syncHeight()
				t.updateSuggestions()
				return t, nil
			}
		}
		if isExplicitNewlineKey(msg) {
			t.Model.SetHeight(t.editorHeight() + 1)
		}

		if t.HasSuggestions() {
			switch msg.String() {
			case "up":
				if t.selectedIdx > 0 {
					t.selectedIdx--
				} else {
					t.selectedIdx = len(t.suggestionItems) - 1
				}
				t.syncSuggestionWindow()
				return t, nil
			case "down":
				if t.selectedIdx < len(t.suggestionItems)-1 {
					t.selectedIdx++
				} else {
					t.selectedIdx = 0
				}
				t.syncSuggestionWindow()
				return t, nil
			case "tab", "enter":
				if t.selectedIdx < len(t.suggestionItems) {
					t = t.applySuggestion(t.suggestionItems[t.selectedIdx])
				}
				return t, nil
			case "esc":
				t = t.clearSuggestions()
				return t, nil
			}
		}
	}

	m, cmd := t.Model.Update(msg)
	t.Model = m
	t.syncHeight()
	if t.maskedPasteLabel != "" && !strings.Contains(t.Model.Value(), t.maskedPasteLabel) {
		t = t.clearMaskedPaste()
	}

	// Update suggestions based on current input
	t.updateSuggestions()

	return t, cmd
}

// PushHistory stores a submitted input line for later up/down recall.
func (t TextInput) PushHistory(value string) TextInput {
	value = strings.TrimSpace(value)
	if value == "" {
		return t
	}
	if n := len(t.history); n > 0 && t.history[n-1] == value {
		t.historyIndex = -1
		t.historyDraft = ""
		return t
	}
	t.history = append(t.history, value)
	t.historyIndex = -1
	t.historyDraft = ""
	return t
}

// PrevHistory recalls the previous submitted line.
func (t TextInput) PrevHistory() TextInput {
	if len(t.history) == 0 {
		return t
	}
	if t.historyIndex == -1 {
		t.historyDraft = t.Model.Value()
		t.historyIndex = len(t.history) - 1
	} else if t.historyIndex > 0 {
		t.historyIndex--
	}
	t.Model.SetValue(t.history[t.historyIndex])
	t = t.clearTransientState().clearSuggestions()
	return t
}

// NextHistory moves forward in submitted-line history, restoring the draft at the end.
func (t TextInput) NextHistory() TextInput {
	if len(t.history) == 0 || t.historyIndex == -1 {
		return t
	}
	if t.historyIndex < len(t.history)-1 {
		t.historyIndex++
		t.Model.SetValue(t.history[t.historyIndex])
		t = t.clearTransientState().clearSuggestions()
		return t
	}
	t.historyIndex = -1
	t.Model.SetValue(t.historyDraft)
	t.historyDraft = ""
	t = t.clearTransientState().clearSuggestions()
	return t
}

// updateSuggestions chooses between slash and @file suggestions based on the current token.
func (t *TextInput) updateSuggestions() {
	token, span, ok := t.currentToken()
	if !ok {
		*t = t.clearSuggestions()
		return
	}

	if items, ok := t.slashSuggestionItems(token, span); ok {
		t.setSuggestions(suggestionKindSlash, span, items)
		return
	}
	if items, ok := t.fileSuggestionItems(token, span); ok {
		t.setSuggestions(suggestionKindFile, span, items)
		return
	}

	*t = t.clearSuggestions()
}

func (t TextInput) separator() string {
	width := t.width
	if width < 1 {
		width = 1
	}
	return separatorStyle.Render(strings.Repeat("─", width+4))
}

// View renders the input with optional suggestions.
func (t TextInput) View() string {
	sep := t.separator()
	inputView := composerStyle.Render(t.Model.View())

	if !t.HasSuggestions() {
		if t.suggestionKind != suggestionKindNone {
			return sep + "\n" + inputView + strings.Repeat("\n", maxVisibleSuggestions) + "\n" + sep
		}
		return sep + "\n" + inputView + "\n" + sep
	}

	// Render suggestions below input
	var sb strings.Builder
	sb.WriteString(sep)
	sb.WriteString("\n")
	sb.WriteString(inputView)
	sb.WriteString("\n")

	start := t.suggestionOffset
	if start < 0 {
		start = 0
	}
	end := start + maxVisibleSuggestions
	if end > len(t.suggestionItems) {
		end = len(t.suggestionItems)
	}

	for i := start; i < end; i++ {
		item := t.suggestionItems[i]

		if i == t.selectedIdx {
			sb.WriteString("    ")
			sb.WriteString(sugSelCmdStyle.Render(item.Display))
			sb.WriteString("  ")
			sb.WriteString(sugSelDescStyle.Render(item.Description))
		} else {
			sb.WriteString("    ")
			sb.WriteString(sugCmdStyle.Render(item.Display))
			sb.WriteString("  ")
			sb.WriteString(sugDescStyle.Render(item.Description))
		}

		sb.WriteString("\n")
	}
	// Pad remaining rows to fill the fixed slash suggestion area.
	rendered := end - start
	for i := rendered; i < maxVisibleSuggestions; i++ {
		sb.WriteString("\n")
	}
	sb.WriteString(sep)

	return sb.String()
}

// Height returns the total height including suggestions area.
func (t TextInput) Height() int {
	height := t.editorHeight() + 2
	if t.suggestionKind != suggestionKindNone {
		return height + maxVisibleSuggestions
	}
	return height
}

// IsSlashMode returns true if showing slash suggestions.
func (t TextInput) IsSlashMode() bool {
	return t.HasSuggestions() && t.suggestionKind == suggestionKindSlash
}

// ClearSlashMode exits the slash suggestion reserved area.
func (t TextInput) ClearSlashMode() TextInput {
	if t.suggestionKind == suggestionKindSlash {
		return t.clearSuggestions()
	}
	return t
}

// HasSuggestions returns true if there are visible suggestion candidates.
func (t TextInput) HasSuggestions() bool {
	return t.suggestionKind != suggestionKindNone && len(t.suggestionItems) > 0
}

// HasPasteSummary returns true when the composer is showing a collapsed paste preview.
func (t TextInput) HasPasteSummary() bool {
	return t.maskedPasteLabel != ""
}

func isExplicitNewlineKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "ctrl+j", "shift+enter":
		return true
	default:
		return false
	}
}

// CanNavigateHistory reports whether up/down should switch prompt history
// instead of moving inside the editor.
func (t TextInput) CanNavigateHistory(direction string) bool {
	row, _, lines := t.cursorPosition()
	switch direction {
	case "up":
		return row == 0
	case "down":
		return len(lines) == 0 || row == len(lines)-1
	default:
		return false
	}
}

func (t *TextInput) syncSuggestionWindow() {
	if len(t.suggestionItems) == 0 {
		t.suggestionOffset = 0
		return
	}

	if t.selectedIdx < 0 {
		t.selectedIdx = 0
	}
	if t.selectedIdx >= len(t.suggestionItems) {
		t.selectedIdx = len(t.suggestionItems) - 1
	}

	if t.selectedIdx < t.suggestionOffset {
		t.suggestionOffset = t.selectedIdx
	}
	if t.selectedIdx >= t.suggestionOffset+maxVisibleSuggestions {
		t.suggestionOffset = t.selectedIdx - maxVisibleSuggestions + 1
	}

	maxOffset := len(t.suggestionItems) - maxVisibleSuggestions
	if maxOffset < 0 {
		maxOffset = 0
	}
	if t.suggestionOffset > maxOffset {
		t.suggestionOffset = maxOffset
	}
	if t.suggestionOffset < 0 {
		t.suggestionOffset = 0
	}
}

func (t TextInput) clearSuggestions() TextInput {
	t.suggestionKind = suggestionKindNone
	t.suggestionItems = nil
	t.selectedIdx = 0
	t.suggestionOffset = 0
	t.activeToken = tokenRange{}
	return t
}

func (t *TextInput) setSuggestions(kind suggestionKind, span tokenRange, items []suggestionItem) {
	if len(items) == 0 {
		*t = t.clearSuggestions()
		return
	}
	t.suggestionKind = kind
	t.suggestionItems = items
	t.activeToken = span
	if t.selectedIdx >= len(items) {
		t.selectedIdx = 0
	}
	t.syncSuggestionWindow()
}

func (t TextInput) applySuggestion(item suggestionItem) TextInput {
	replacement := item.Value + " "
	switch item.Kind {
	case suggestionKindFile:
		replacement = "@" + item.Value + " "
	}

	t.Model.SetValue(replaceRunesInRange(t.Model.Value(), t.activeToken, replacement))
	t.Model.SetCursor(t.activeToken.start + len([]rune(replacement)))
	return t.clearTransientState().clearSuggestions()
}

func (t TextInput) slashSuggestionItems(token string, span tokenRange) ([]suggestionItem, bool) {
	if span.start != 0 || !strings.HasPrefix(token, "/") {
		return nil, false
	}

	names := t.slashRegistry.Suggestions(token)
	items := make([]suggestionItem, 0, len(names))
	for _, name := range names {
		cmd, ok := t.slashRegistry.Get(name)
		if !ok {
			continue
		}
		items = append(items, suggestionItem{
			Value:       name,
			Display:     name,
			Description: cmd.Description,
			Kind:        suggestionKindSlash,
		})
	}
	return items, true
}

func (t TextInput) fileSuggestionItems(token string, span tokenRange) ([]suggestionItem, bool) {
	if !IsFileSuggestionToken(token) {
		return nil, false
	}
	return t.fileSuggestion.suggestions(token[1:]), true
}

func (t TextInput) currentToken() (string, tokenRange, bool) {
	value := t.Model.Value()
	runes := []rune(value)
	cursor := t.absoluteCursorPosition()
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	start := cursor
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	end := cursor
	for end < len(runes) && !unicode.IsSpace(runes[end]) {
		end++
	}
	if start == end {
		return "", tokenRange{}, false
	}
	return string(runes[start:end]), tokenRange{start: start, end: end}, true
}

func (t TextInput) absoluteCursorPosition() int {
	row, col, lines := t.cursorPosition()
	offset := 0
	for i := 0; i < row && i < len(lines); i++ {
		offset += len([]rune(lines[i])) + 1
	}
	return offset + col
}

func replaceRunesInRange(value string, span tokenRange, replacement string) string {
	runes := []rune(value)
	prefix := string(runes[:span.start])
	suffix := string(runes[span.end:])
	if strings.HasSuffix(replacement, " ") {
		suffix = strings.TrimLeftFunc(suffix, unicode.IsSpace)
	}
	return prefix + replacement + suffix
}

func IsFileSuggestionToken(token string) bool {
	switch {
	case token == "":
		return false
	case strings.HasPrefix(token, "@@"):
		return false
	case !strings.HasPrefix(token, "@") || len(token) == 1:
		return false
	}
	for _, r := range token[1:] {
		if !isAllowedFileSuggestionRune(r) {
			return false
		}
	}
	return true
}

func (t TextInput) clearMaskedPaste() TextInput {
	t.maskedPasteRaw = ""
	t.maskedPasteLabel = ""
	return t
}

func (t TextInput) clearTransientState() TextInput {
	t.syncHeight()
	return t.clearMaskedPaste()
}

func isAllowedFileSuggestionRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	switch r {
	case '.', '_', '/', '\\', '-':
		return true
	default:
		return false
	}
}

func (t *TextInput) syncHeight() {
	t.Model.SetHeight(t.editorHeight())
}

func (t TextInput) editorHeight() int {
	lines := t.Model.LineCount()
	if lines < minComposerRows {
		return minComposerRows
	}
	return lines
}

func (t TextInput) atInputStart() bool {
	row, col, lines := t.cursorPosition()
	return row == 0 && col == 0 && len(lines) > 0
}

func (t TextInput) atInputEnd() bool {
	row, col, lines := t.cursorPosition()
	if len(lines) == 0 {
		return true
	}
	return row == len(lines)-1 && col == len([]rune(lines[row]))
}

func (t TextInput) cursorPosition() (int, int, []string) {
	lines := t.lines()
	row := t.Model.Line()
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}

	info := t.Model.LineInfo()
	col := info.StartColumn + info.ColumnOffset
	if col < 0 {
		col = 0
	}
	maxCol := len([]rune(lines[row]))
	if col > maxCol {
		col = maxCol
	}
	return row, col, lines
}

func (t TextInput) lines() []string {
	value := t.Model.Value()
	if value == "" {
		return []string{""}
	}
	return strings.Split(value, "\n")
}
