package harness

// newCodex builds the codex channel. There is none yet, so every codex
// session attends as a terminal one.
func newCodex(_ func(string) string, _ Session) Harness { return nil }
