# `@file` Function Status Plan

## Summary

This document records the current implementation status of `@file` in `ms-cli`,
the updates already landed, and the remaining follow-up work.

`@file` is no longer submit-time only. The current implementation now covers:

- backend-side prompt expansion for supported surfaces
- input-time file suggestions in the composer
- keyboard acceptance for `@file` candidates
- backend-confirmed user echo after expansion

The current design remains intentionally conservative. It optimizes for
predictable behavior rather than editor-like flexibility.

## Current Status

### Implemented

1. Backend expansion

- Plain chat input expands standalone `@relative/path`
- `/report`, `/diagnose`, `/fix`, `/skill <name> ...`, and direct skill aliases
  support the same expansion model
- Expansion still happens only after normal command routing is preserved
- Final validation still belongs to backend logic

2. Input-time suggestions

- Typing `@` or `@path` in the composer can open file suggestions
- Suggestions are sourced from the current workspace
- Matching is prefix-only against normalized relative paths
- Suggestions are sorted lexicographically

3. Suggestion interaction

- `Up` / `Down` moves within candidates
- `Tab` accepts the selected candidate
- `Enter` accepts the selected candidate before submit
- `Esc` closes suggestions without clearing input

4. Surface coverage

- Main chat composer supports `@file` suggestions
- Slash command argument positions can switch from slash suggestions to `@file`
  suggestions
- Issue-detail composer paths now reuse the same suggestion handling

5. User echo behavior

- Raw `@file` input is no longer eagerly shown in chat when the token should be
  expanded later
- The chat stream shows the backend-confirmed `UserInput` event after expansion

### Intentionally Unchanged

- Only standalone whitespace-delimited `@relative/path` tokens are valid
- `@@name` still means a literal `@name`
- `@file,` and `(@file)` still do not participate
- Paths with spaces remain unsupported
- Final file safety checks still happen at submit time, not during typing

## Update Accounting

### Workstream Breakdown

1. Backend expansion work: complete

- Shared file validation and text reading are already in place
- Supported command surfaces are wired
- Regression coverage exists for supported command behavior

2. Composer suggestion work: complete for `v1`

- Suggestion state was generalized from slash-only to multi-kind suggestions
- `@file` token detection and token replacement are implemented
- Workspace file suggestion provider is implemented

3. UI integration work: complete for `v1`

- Main chat composer is wired
- Slash-argument `@file` suggestions are wired
- Issue-detail composer suggestion handling is aligned

4. Documentation and messaging: mostly complete

- Help text mentions input-time `@file` completion
- Status/plan documentation exists
- README alignment can still be improved if a broader user-facing update is wanted

### Progress Snapshot

- Planned `v1` capability set: 4 major areas
- Completed: 4 / 4
- Remaining for current branch: polish and follow-up refinements only

## Tests and Validation Status

### Covered

- `ui/components`
  - `@file` suggestions open on valid tokens
  - prefix filtering works
  - `Tab` and `Enter` accept candidates
  - current-token replacement works
  - multiline replacement works
  - invalid `@` forms stay inactive
  - slash and `@file` suggestion priority works

- `ui`
  - first `Enter` accepts a suggestion instead of submitting
  - second `Enter` submits the completed prompt
  - backend-confirmed expanded user echo appears in chat
  - history navigation behavior is preserved
  - large-paste summary behavior is preserved

- `internal/app`
  - submit-time expansion still works
  - supported command surfaces still behave correctly
  - expanded `UserInput` event behavior is covered

### Validation Commands

The targeted commands used during implementation were:

```powershell
$env:GOCACHE='d:\codex_workspace\ms-cli\.gocache'; go test ./ui/components -run '^(TestTextInput.*)$'
$env:GOCACHE='d:\codex_workspace\ms-cli\.gocache'; go test ./ui -run '^(TestAtFileInputWaitsForBackendEchoBeforeShowingUserMessage|TestEnterAcceptsAtFileSuggestionBeforeSubmittingToBackend|TestLargePastedUserMessageRendersAsSummary|TestUpDownRecallInputHistoryInsteadOfScrollingViewport|TestUpDownContinueHistoryAcrossSlashCommandsWithoutEnteringSuggestionNavigation)$'
$env:GOCACHE='d:\codex_workspace\ms-cli\.gocache'; go test ./internal/app -run '^(TestProcessInputExpandsPlainChatBeforeRunTask|TestProcessInputEmitsExpandedUserInputEvent|TestHandleCommand.*|TestExpandInputText.*)$'
```

## Remaining Gaps

These are known limitations rather than regressions:

- no fuzzy matching
- no directory-first ranking
- no path preview or metadata preview
- no gitignore-aware filtering
- no support for paths with spaces
- no input-time “missing / directory / oversized / binary” validation feedback
- no background refresh or watcher for very large workspaces

## Follow-up Update Plan

### Priority 1: UX polish

- add clearer visual affordance for active `@file` tokens
- add lightweight hint text such as “Enter to accept file suggestion”
- improve ranking beyond plain lexicographic ordering

### Priority 2: File-navigation improvements

- directory-first ordering
- optional directory completion as `@dir/`
- better path bias for shorter or more local matches

### Priority 3: Scale and performance

- cache refresh strategy
- larger workspace scanning limits or debounce
- optional background indexing if current approach becomes too slow

### Priority 4: Syntax expansion

- optional fuzzy matching
- optional support for spaces in paths
- optional support for approved non-workspace file roots

## Recommendation

Treat the current branch state as `v1 complete`.

Further changes should be framed as polish or `v1.1` improvements, not as
unfinished core functionality. The basic `@file` workflow is already present:

- discover a file while typing
- accept it in the composer
- submit the completed prompt
- let backend expansion remain authoritative
