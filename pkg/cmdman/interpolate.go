package cmdman

import (
	"iter"
	"slices"
	"strings"
)

// InterpolateKeys returns an iterator that replaces placeholders in each key string.
func InterpolateKeys(keys []string, commandID, commandName string) iter.Seq[string] {
	if len(keys) == 0 {
		return slices.Values(keys)
	}
	return func(yield func(string) bool) {
		for _, k := range keys {
			if !yield(InterpolateKey(k, commandID, commandName)) {
				return
			}
		}
	}
}

var interpolationTargets = [...]string{
	"COMMAND_ID",
	"COMMAND_NAME",
	"INJECT_META",
}

// InterpolateKey replaces placeholders in a single key string.
//
// Supported placeholders:
//   - #{COMMAND_ID}   — replaced with the command UUID
//   - #{COMMAND_NAME} — replaced with the human-readable name
//   - #{INJECT_META}  — replaced with an export command setting CRAB_COMMAND_ID and CRAB_COMMAND_NAME
//
// Escaping: ##{...} produces the literal #{...}.
func InterpolateKey(key, commandID, commandName string) string {
	if !strings.Contains(key, "#{") {
		return key
	}

	replacements := map[string]string{
		"COMMAND_ID":   commandID,
		"COMMAND_NAME": commandName,
		"INJECT_META":  "export CRAB_COMMAND_ID='" + commandID + "' CRAB_COMMAND_NAME='" + commandName + "'",
	}

	var b strings.Builder
	b.Grow(len(key))

	i := 0
LOOP:
	for i < len(key) {
		if key[i] != '#' {
			j := i + 1
			for ; j < len(key) && key[j] != '#'; j++ {
			}
			b.WriteString(key[i:j])
			i = j
			continue
		}

		for _, k := range interpolationTargets {
			if strings.HasPrefix(key[i:], "##{"+k+"}") {
				b.WriteString("#{" + k + "}")
				i += len("##{" + k + "}")
				continue LOOP
			}
			if strings.HasPrefix(key[i:], "#{"+k+"}") {
				b.WriteString(replacements[k])
				i += len("#{" + k + "}")
				continue LOOP
			}
		}

		j := i + 1
		for ; j < len(key) && key[j] != '#'; j++ {
		}
		b.WriteString(key[i:j])
		i = j
	}

	return b.String()
}
