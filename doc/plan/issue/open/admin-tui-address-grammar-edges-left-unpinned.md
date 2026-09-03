---
tags: chat admin tui addressing
---

# Admin TUI `@` grammar edges left unpinned (2026-09-03)

`parseAddress` in `crabswarm/chat/cli/tui/address.go` takes the first
bare `@token` (word-start, outside backticks, not `\@`) as the target and
sends the text whole. Four edges follow from the rule and are neither
tested nor decided:

- A `@token` on any line of a multi-line draft addresses, since the
  token ends at whitespace and a newline is whitespace; a quoted
  `@alpha/bob` on line three of a paragraph becomes the target.
- Punctuation-terminated mentions such as `@admin,` or `(@admin)` are
  not highlighted, because `mentionsAdmin` compares the whole
  whitespace-delimited token.
- An unbalanced backtick turns the rest of the text into a code span,
  so every later `@` is plain text.
- The TUI's `@alpha/ana` spelling pasted into `chat admin send` argv
  parses as team `@alpha`, name `ana`, and the daemon answers NotFound.

Follow-up: decide which of these deserve a rule (first line only,
trim punctuation, refuse or close an unbalanced span, strip a leading
`@` on argv) and pin the choice in `address_test.go` and
`cli/target_test.go`.
