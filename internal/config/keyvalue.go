package config

import (
	"fmt"
	"strconv"
	"strings"
)

// UpsertKeyValue updates an rc-style key while preserving unrelated lines.
// Keys in removeAliases are commented out because they conflict with the
// managed setting.
func UpsertKeyValue(content, key, value string, aliases, removeAliases []string) string {
	aliases = append([]string{key}, aliases...)
	lines, hadTrailingNewline := splitLines(content)
	found := false

	for i, line := range lines {
		lineKey, _, ok := parseKeyValueLine(line)
		if !ok {
			continue
		}
		if stringIn(lineKey, aliases) {
			lines[i] = key + "=" + value
			found = true
			continue
		}
		if stringIn(lineKey, removeAliases) {
			lines[i] = "# smsc disabled conflicting setting: " + line
		}
	}

	if !found {
		lines = append(lines, key+"="+value)
		hadTrailingNewline = true
	}

	return joinLines(lines, hadTrailingNewline)
}

func ReadKeyValue(content string, aliases []string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := parseKeyValueLine(line)
		if ok && stringIn(key, aliases) {
			return value, true
		}
	}
	return "", false
}

func ReadKeyValueInt(content string, aliases []string) (int, string, bool) {
	raw, ok := ReadKeyValue(content, aliases)
	if !ok {
		return 0, "", false
	}
	value, err := strconv.Atoi(strings.Trim(raw, `"' `))
	if err != nil {
		return 0, raw, false
	}
	return value, raw, true
}

func FormatDays(days int) string {
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func splitLines(content string) ([]string, bool) {
	if content == "" {
		return nil, false
	}
	hadTrailingNewline := strings.HasSuffix(content, "\n")
	trimmed := strings.TrimSuffix(content, "\n")
	if trimmed == "" {
		return []string{""}, hadTrailingNewline
	}
	return strings.Split(trimmed, "\n"), hadTrailingNewline
}

func joinLines(lines []string, trailingNewline bool) string {
	out := strings.Join(lines, "\n")
	if trailingNewline {
		out += "\n"
	}
	return out
}

func parseKeyValueLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
		return "", "", false
	}

	separator := strings.IndexAny(trimmed, "=:")
	if separator < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(trimmed[:separator])
	value := strings.TrimSpace(trimmed[separator+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func stringIn(needle string, haystack []string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
