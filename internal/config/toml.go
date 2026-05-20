package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func ReadBunMinimumReleaseAge(content string) (int64, string, bool, error) {
	if strings.TrimSpace(content) == "" {
		return 0, "", false, nil
	}
	var data map[string]any
	if err := toml.Unmarshal([]byte(content), &data); err != nil {
		return 0, "", false, err
	}
	install, ok := data["install"].(map[string]any)
	if !ok {
		return 0, "", false, nil
	}
	raw, ok := install["minimumReleaseAge"]
	if !ok {
		return 0, "", false, nil
	}
	value, ok := numericToInt64(raw)
	if !ok {
		return 0, fmt.Sprint(raw), false, nil
	}
	return value, fmt.Sprint(value), true, nil
}

func SetBunMinimumReleaseAge(content string, seconds int64) (string, error) {
	out := upsertTOMLSectionKey(content, "install", "minimumReleaseAge", strconv.FormatInt(seconds, 10))
	var data map[string]any
	if err := toml.Unmarshal([]byte(out), &data); err != nil {
		return "", err
	}
	return out, nil
}

func ReadTOMLTopString(content, key string) (string, bool, error) {
	if strings.TrimSpace(content) == "" {
		return "", false, nil
	}
	var data map[string]any
	if err := toml.Unmarshal([]byte(content), &data); err != nil {
		return "", false, err
	}
	raw, ok := data[key]
	if !ok {
		return "", false, nil
	}
	value, ok := raw.(string)
	if !ok {
		return fmt.Sprint(raw), false, nil
	}
	return value, true, nil
}

func SetTOMLTopString(content, key, value string) (string, error) {
	out := upsertTOMLTopKey(content, key, strconv.Quote(value))
	var data map[string]any
	if err := toml.Unmarshal([]byte(out), &data); err != nil {
		return "", err
	}
	return out, nil
}

func upsertTOMLTopKey(content, key, value string) string {
	lines, hadTrailing := splitLines(content)
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		lineKey, _, ok := parseKeyValueLine(line)
		if ok && lineKey == key {
			lines[i] = key + " = " + value
			found = true
			break
		}
	}
	if !found {
		insertAt := len(lines)
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "[") {
				insertAt = i
				break
			}
		}
		lines = append(lines[:insertAt], append([]string{key + " = " + value}, lines[insertAt:]...)...)
		hadTrailing = true
	}
	return joinLines(lines, hadTrailing)
}

func upsertTOMLSectionKey(content, section, key, value string) string {
	lines, hadTrailing := splitLines(content)
	inSection := false
	sectionStart := -1
	sectionEnd := len(lines)
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inSection {
				sectionEnd = i
				break
			}
			if trimmed == "["+section+"]" {
				inSection = true
				sectionStart = i
			}
			continue
		}
		if inSection {
			lineKey, _, ok := parseKeyValueLine(line)
			if ok && lineKey == key {
				lines[i] = key + " = " + value
				found = true
				break
			}
		}
	}

	if found {
		return joinLines(lines, hadTrailing)
	}

	if sectionStart < 0 {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", key+" = "+value)
		return joinLines(lines, true)
	}

	insertAt := sectionEnd
	lines = append(lines[:insertAt], append([]string{key + " = " + value}, lines[insertAt:]...)...)
	return joinLines(lines, true)
}

func numericToInt64(raw any) (int64, bool) {
	switch value := raw.(type) {
	case int64:
		return value, true
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case float64:
		return int64(value), true
	default:
		return 0, false
	}
}
