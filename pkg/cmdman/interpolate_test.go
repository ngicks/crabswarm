package cmdman

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestInterpolateKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		id       string
		cmdName  string
		expected string
	}{
		{
			name:     "no placeholders",
			key:      "hello world",
			id:       "abc",
			cmdName:  "test",
			expected: "hello world",
		},
		{
			name:     "command id",
			key:      "id=#{COMMAND_ID}",
			id:       "abc-123",
			cmdName:  "test",
			expected: "id=abc-123",
		},
		{
			name:     "command name",
			key:      "name=#{COMMAND_NAME}",
			id:       "abc",
			cmdName:  "mytest",
			expected: "name=mytest",
		},
		{
			name:     "inject meta",
			key:      "#{INJECT_META}",
			id:       "abc-123",
			cmdName:  "mytest",
			expected: "export CRAB_COMMAND_ID='abc-123' CRAB_COMMAND_NAME='mytest'",
		},
		{
			name:     "escaped",
			key:      "##{COMMAND_ID}",
			id:       "abc",
			cmdName:  "test",
			expected: "#{COMMAND_ID}",
		},
		{
			name:     "mixed",
			key:      "#{COMMAND_ID} and ##{COMMAND_NAME}",
			id:       "abc",
			cmdName:  "test",
			expected: "abc and #{COMMAND_NAME}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InterpolateKey(tt.key, tt.id, tt.cmdName)
			assert.Equal(t, result, tt.expected)
		})
	}
}
