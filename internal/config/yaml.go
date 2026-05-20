package config

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

func ReadYAMLString(content, key string) (string, bool, error) {
	if strings.TrimSpace(content) == "" {
		return "", false, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return "", false, err
	}
	mapping := yamlMappingRoot(&root)
	if mapping == nil {
		return "", false, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1].Value, true, nil
		}
	}
	return "", false, nil
}

func SetYAMLString(content, key, value string) (string, error) {
	var doc yaml.Node
	if strings.TrimSpace(content) == "" {
		doc = yaml.Node{Kind: yaml.DocumentNode}
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	} else if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", err
	}

	mapping := yamlMappingRoot(&doc)
	if mapping == nil {
		doc = yaml.Node{Kind: yaml.DocumentNode}
		mapping = &yaml.Node{Kind: yaml.MappingNode}
		doc.Content = []*yaml.Node{mapping}
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].Kind = yaml.ScalarNode
			mapping.Content[i+1].Tag = "!!str"
			mapping.Content[i+1].Value = value
			return encodeYAML(&doc)
		}
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
	return encodeYAML(&doc)
}

func RemoveYAMLTopKey(content, key string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return content, nil
	}
	if _, ok, err := ReadYAMLString(content, key); err != nil || !ok {
		return content, err
	}

	lines, hadTrailingNewline := splitLines(content)
	filtered := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if isTopLevelYAMLKey(line, key) {
			removed = true
			continue
		}
		filtered = append(filtered, line)
	}
	if removed {
		return joinRemovedLines(filtered, hadTrailingNewline), nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", err
	}
	mapping := yamlMappingRoot(&doc)
	if mapping == nil {
		return content, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return encodeYAML(&doc)
		}
	}
	return content, nil
}

func isTopLevelYAMLKey(line, key string) bool {
	if strings.TrimSpace(line) == "" || strings.TrimLeft(line, " \t") != line {
		return false
	}
	lineKey, _, ok := parseKeyValueLine(line)
	return ok && lineKey == key
}

func yamlMappingRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		if doc.Content[0].Kind == yaml.MappingNode {
			return doc.Content[0]
		}
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

func encodeYAML(doc *yaml.Node) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
