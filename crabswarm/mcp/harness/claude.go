package harness

// newClaudeCode builds the claude channel. There is none yet, so every claude
// session attends as a terminal one.
func newClaudeCode(_ func(string) string, _ Session) Harness { return nil }
