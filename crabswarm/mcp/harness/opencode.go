package harness

// newOpenCode builds the opencode channel. There is none yet, so every opencode
// session attends as a terminal one.
func newOpenCode(_ func(string) string, _ Session) Harness { return nil }
