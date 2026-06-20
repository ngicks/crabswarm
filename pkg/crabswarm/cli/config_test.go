package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/ngicks/crabswarm/pkg/crabswarm/cli"
)

func sampleConfig() crabswarm.Config {
	return crabswarm.Config{
		Sock:           "/run/crabswarm.sock",
		ProjectDir:     "/work/proj",
		GitRepoBaseDir: "/home/me/gitrepo",
	}
}

// The default (empty format) renders indented JSON, newline-terminated, that
// round-trips back to the same Config.
func TestRenderConfig_JSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := sampleConfig()
	assert.NilError(t, cli.RenderConfig(&buf, cfg, ""))

	out := buf.String()
	assert.Assert(t, strings.HasSuffix(out, "\n"))
	assert.Assert(t, strings.Contains(out, `  "sock": "/run/crabswarm.sock"`))

	var got crabswarm.Config
	assert.NilError(t, json.Unmarshal([]byte(out), &got))
	assert.DeepEqual(t, got, cfg)
}

// A non-empty format renders as a Go text/template against the Config value and
// is newline-terminated.
func TestRenderConfig_Template(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, cli.RenderConfig(&buf, sampleConfig(), "{{.Sock}} {{.ProjectDir}}"))
	assert.Equal(t, buf.String(), "/run/crabswarm.sock /work/proj\n")
}

// A malformed template is a hard error attributed to --format.
func TestRenderConfig_InvalidTemplate(t *testing.T) {
	var buf bytes.Buffer
	err := cli.RenderConfig(&buf, sampleConfig(), "{{.Nope")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "--format"))
}

// A template referencing a non-existent field fails at execution time and is
// also attributed to --format.
func TestRenderConfig_TemplateExecError(t *testing.T) {
	var buf bytes.Buffer
	err := cli.RenderConfig(&buf, sampleConfig(), "{{.DoesNotExist}}")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "--format"))
}
