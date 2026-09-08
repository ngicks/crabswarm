package loggerfactory

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestReadEnv(t *testing.T) {
	t.Run("hyphenated app name maps to underscored prefix", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		err := ReadEnv(c, "my-tool", []string{
			"MY_TOOL_LOG_FORMAT=text",
			"MY_TOOL_LOG_LEVEL=debug",
		})
		if err != nil {
			t.Fatalf("ReadEnv: %v", err)
		}
		if !c.Enabled {
			t.Fatalf("Enabled = false, want true")
		}
		if c.Format != "text" {
			t.Fatalf("Format = %q, want %q", c.Format, "text")
		}
		if c.Level != slog.LevelDebug {
			t.Fatalf("Level = %v, want %v", c.Level, slog.LevelDebug)
		}
	})

	t.Run("missing vars leave config untouched", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		if err := ReadEnv(c, "mytool", []string{"OTHER=1"}); err != nil {
			t.Fatalf("ReadEnv: %v", err)
		}
		if c.Enabled {
			t.Fatalf("Enabled = true, want false")
		}
		if c.Format != "json" {
			t.Fatalf("Format = %q, want %q", c.Format, "json")
		}
	})

	t.Run("empty value leaves field untouched", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		if err := ReadEnv(c, "mytool", []string{"MYTOOL_LOG_FORMAT="}); err != nil {
			t.Fatalf("ReadEnv: %v", err)
		}
		if c.Enabled {
			t.Fatalf("Enabled = true, want false (empty value should not enable)")
		}
		if c.Format != "json" {
			t.Fatalf("Format = %q, want %q", c.Format, "json")
		}
	})

	t.Run("only level set still enables", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		if err := ReadEnv(c, "mytool", []string{"MYTOOL_LOG_LEVEL=warn"}); err != nil {
			t.Fatalf("ReadEnv: %v", err)
		}
		if !c.Enabled {
			t.Fatalf("Enabled = false, want true")
		}
		if c.Level != slog.LevelWarn {
			t.Fatalf("Level = %v, want %v", c.Level, slog.LevelWarn)
		}
	})

	t.Run("invalid value returns error", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		err := ReadEnv(c, "mytool", []string{"MYTOOL_LOG_FORMAT=xml"})
		if err == nil {
			t.Fatalf("expected error for invalid format")
		}
	})

	t.Run("case-insensitive values", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		if err := ReadEnv(c, "mytool", []string{
			"MYTOOL_LOG_FORMAT=TEXT",
			"MYTOOL_LOG_LEVEL=Trace",
		}); err != nil {
			t.Fatalf("ReadEnv: %v", err)
		}
		if c.Format != "text" {
			t.Fatalf("Format = %q, want %q", c.Format, "text")
		}
		if c.Level != LevelTrace {
			t.Fatalf("Level = %v, want %v", c.Level, LevelTrace)
		}
	})

	t.Run("malformed entries are skipped", func(t *testing.T) {
		c := &Config{Format: "json", Level: slog.LevelInfo}
		if err := ReadEnv(c, "mytool", []string{
			"NOTANENV", // no '='
			"MYTOOL_LOG_LEVEL=info",
		}); err != nil {
			t.Fatalf("ReadEnv: %v", err)
		}
		if !c.Enabled {
			t.Fatalf("Enabled = false, want true")
		}
	})
}

// A command nobody configured logging for still reports what went wrong: warn
// and above land on the writer as text, carrying their attributes and no source
// position, and anything below warn is left out.
func TestBuildLoggerTo_Unconfigured(t *testing.T) {
	var buf strings.Builder
	logger := BuildLoggerTo(&Config{}, &buf)

	logger.Info("routine chatter")
	if got := buf.String(); got != "" {
		t.Fatalf("output after an info record = %q, want nothing below warn", got)
	}

	logger.Warn("mirroring member state failed", "member", "alpha/ana")
	got := buf.String()
	for _, want := range []string{
		"level=WARN",
		`msg="mirroring member state failed"`,
		"member=alpha/ana",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("warn record = %q, want it to carry %q", got, want)
		}
	}
	if strings.Contains(got, "source=") {
		t.Errorf("warn record = %q, want no source position", got)
	}
}

// A configured logger replaces the default outright rather than being floored
// by it: the format is the one asked for, and a level below warn is honored.
func TestBuildLoggerTo_ConfiguredReplacesTheDefault(t *testing.T) {
	var buf strings.Builder
	logger := BuildLoggerTo(&Config{Enabled: true, Format: "json", Level: LevelTrace}, &buf)

	logger.Log(t.Context(), LevelTrace, "dialing", "sock", "/run/crabswarm.sock")

	var rec map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &rec); err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}
	if got := rec["msg"]; got != "dialing" {
		t.Errorf("msg = %v, want %q", got, "dialing")
	}
	if got := rec["sock"]; got != "/run/crabswarm.sock" {
		t.Errorf("sock = %v, want %q", got, "/run/crabswarm.sock")
	}
	if _, ok := rec["source"]; !ok {
		t.Errorf("record = %v, want the source position a configured logger adds", rec)
	}
}
