package server

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// updateStageInProvidersYAML reads the providers.yaml at path, sets stages.<stage> to provider
// using yaml.Node surgery (preserving comments, field order, and absent optional fields), and
// returns the modified YAML string ready to pass to config.Save.
//
// If the stages mapping lacks the stage key it is appended. If the file is missing, unreadable,
// or structurally invalid an error is returned and nothing is written.
func updateStageInProvidersYAML(path, stage, provider string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read providers.yaml: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse providers.yaml: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return "", fmt.Errorf("providers.yaml: unexpected document structure")
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("providers.yaml: root is not a mapping")
	}

	// Find the "stages" value node in the root mapping.
	stagesNode := yamlMappingLookup(root, "stages")
	if stagesNode == nil {
		return "", fmt.Errorf("providers.yaml: missing stages key")
	}
	if stagesNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("providers.yaml: stages is not a mapping")
	}

	// Update the existing key, or append a new key-value pair.
	if !yamlSetMappingScalar(stagesNode, stage, provider) {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: stage, Tag: "!!str"}
		valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: provider, Tag: "!!str"}
		stagesNode.Content = append(stagesNode.Content, keyNode, valNode)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", fmt.Errorf("encode providers.yaml: %w", err)
	}
	_ = enc.Close()
	return buf.String(), nil
}

// yamlMappingLookup returns the value node for key in a yaml MappingNode, or nil if absent.
func yamlMappingLookup(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// yamlSetMappingScalar sets the scalar value of an existing key in a yaml MappingNode.
// Returns true when the key was found and updated, false when the key is absent.
func yamlSetMappingScalar(m *yaml.Node, key, value string) bool {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1].Value = value
			return true
		}
	}
	return false
}
