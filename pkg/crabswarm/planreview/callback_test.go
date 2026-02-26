package planreview

import (
	"context"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestRunCallback_Success(t *testing.T) {
	stdout, stderr, err := RunCallback(context.Background(), CallbackConfig{
		Command:         "echo",
		Args:            []string{"hello review"},
		PlanWorkingFile: "/tmp/PLAN.md",
		PlanSourceFile:  "/tmp/source.md",
		PlanDir:         "/tmp/plandir",
		PlanName:        "test-plan",
		Iteration:       1,
		IntermediateDir: "/tmp/intermediate",
	})

	assert.NilError(t, err)
	assert.Equal(t, strings.TrimSpace(stdout), "hello review")
	assert.Equal(t, stderr, "")
}

func TestRunCallback_EnvVars(t *testing.T) {
	// Use env to print all environment variables, then check the ones we set.
	stdout, _, err := RunCallback(context.Background(), CallbackConfig{
		Command:         "env",
		PlanWorkingFile: "/work/PLAN.md",
		PlanSourceFile:  "/src/plan.md",
		PlanDir:         "/work",
		PlanName:        "my-plan",
		Iteration:       3,
		IntermediateDir: "/work/_intermediate",
	})

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(stdout, "PLAN_WORKING_FILE=/work/PLAN.md"))
	assert.Assert(t, strings.Contains(stdout, "PLAN_SOURCE_FILE=/src/plan.md"))
	assert.Assert(t, strings.Contains(stdout, "PLAN_DIR=/work"))
	assert.Assert(t, strings.Contains(stdout, "PLAN_NAME=my-plan"))
	assert.Assert(t, strings.Contains(stdout, "ITERATION=3"))
	assert.Assert(t, strings.Contains(stdout, "INTERMEDIATE_DIR=/work/_intermediate"))
}

func TestRunCallback_EmptyCommand(t *testing.T) {
	_, _, err := RunCallback(context.Background(), CallbackConfig{
		Command: "",
	})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "callback command is empty"))
}

func TestRunCallback_CommandFailure(t *testing.T) {
	_, stderr, err := RunCallback(context.Background(), CallbackConfig{
		Command: "sh",
		Args:    []string{"-c", "echo fail-output >&2; exit 1"},
	})

	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(stderr, "fail-output"))
}

func TestRunCallback_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := RunCallback(ctx, CallbackConfig{
		Command: "sleep",
		Args:    []string{"10"},
	})

	assert.Assert(t, err != nil)
}
