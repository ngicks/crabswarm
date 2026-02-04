#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY_NAME="crabhook"
SETTINGS_FILE="$PROJECT_ROOT/.claude/settings.local.json"

cd "$PROJECT_ROOT"

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    echo "Install it with: apt install jq, brew install jq, etc."
    exit 1
fi

# Build crabhook
echo "Building $BINARY_NAME..."
mkdir -p bin
go build -o "bin/$BINARY_NAME" ./hook/cmd/crabhook
chmod +x "bin/$BINARY_NAME"
echo "Built bin/$BINARY_NAME"

# Create .claude directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/.claude"

# Hook configuration to add
HOOK_CONFIG=$(cat <<EOF
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "$PROJECT_ROOT/bin/crabhook",
            "timeout": 300,
            "statusMessage": "Waiting for permission..."
          }
        ]
      }
    ]
  }
}
EOF
)

# Create or update settings file
if [[ -f "$SETTINGS_FILE" ]]; then
    echo "Updating $SETTINGS_FILE..."
    # Merge hook configuration into existing settings, preserving other fields
    # Use jq to deep merge, replacing hooks.PreToolUse if it exists
    UPDATED=$(jq --argjson hook_config "$HOOK_CONFIG" '
        . * $hook_config
    ' "$SETTINGS_FILE")
    echo "$UPDATED" > "$SETTINGS_FILE"
else
    echo "Creating $SETTINGS_FILE..."
    echo "$HOOK_CONFIG" | jq '.' > "$SETTINGS_FILE"
fi

echo ""
echo "Done! Hook configuration added to $SETTINGS_FILE"
echo ""
echo "The hook will be active in new Claude Code sessions."
echo "Hook binary: $PROJECT_ROOT/bin/$BINARY_NAME"
