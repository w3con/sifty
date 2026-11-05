package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// updateSearchExpressionsInYAML reads the search.yaml at path, replaces the search.expressions
// key with newExprs (and clears titles/keywords when expressions are set), and returns the
// modified YAML string ready to pass to config.Save.
//
// newExprs is a JSON-encoded []{"or":[...],"and":[...]} array. When newExprs is nil or empty,
// the expressions key is removed and titles/keywords are left untouched.
// Comments and unrelated fields are preserved.
func updateSearchExpressionsInYAML(path string, newExprs []map[string][]string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read search.yaml: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse search.yaml: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return "", fmt.Errorf("search.yaml: unexpected document structure")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("search.yaml: root is not a mapping")
	}

	// Find the "search" value node.
	searchNode := yamlMappingLookup(root, "search")
	if searchNode == nil {
		return "", fmt.Errorf("search.yaml: missing search key")
	}
	if searchNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("search.yaml: search is not a mapping")
	}

	if len(newExprs) == 0 {
		// Remove the expressions key so backward-compat titles/keywords remain in control.
		removeYAMLMappingKey(searchNode, "expressions")
	} else {
		// Build a yaml.Node sequence for the expressions array.
		exprSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, e := range newExprs {
			exprMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			for _, key := range []string{"or", "and"} {
				terms, ok := e[key]
				if !ok || len(terms) == 0 {
					continue
				}
				keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
				valSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
				for _, t := range terms {
					valSeq.Content = append(valSeq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: t, Tag: "!!str"})
				}
				exprMap.Content = append(exprMap.Content, keyNode, valSeq)
			}
			exprSeq.Content = append(exprSeq.Content, exprMap)
		}
		// Set or append the expressions key.
		if !yamlSetMappingNode(searchNode, "expressions", exprSeq) {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "expressions", Tag: "!!str"}
			searchNode.Content = append(searchNode.Content, keyNode, exprSeq)
		}
		// When expressions are set explicitly, clear titles and keywords so the effective
		// path uses expressions only (keeps the YAML clean).
		clearYAMLSequence(searchNode, "titles")
		clearYAMLSequence(searchNode, "keywords")
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", fmt.Errorf("encode search.yaml: %w", err)
	}
	_ = enc.Close()
	return buf.String(), nil
}

// updateSearchSiteInYAML reads the search.yaml at path and applies the requested site operation.
// op is one of "add", "remove", "enable", "disable".
// For "add", site must be {name, url, enabled}. For the others, name identifies the site.
func updateSearchSiteInYAML(path, op string, site map[string]any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read search.yaml: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse search.yaml: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return "", fmt.Errorf("search.yaml: unexpected document structure")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("search.yaml: root is not a mapping")
	}

	sitesNode := yamlMappingLookup(root, "sites")
	if sitesNode == nil {
		return "", fmt.Errorf("search.yaml: missing sites key")
	}
	if sitesNode.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("search.yaml: sites is not a sequence")
	}

	switch op {
	case "add":
		name, _ := site["name"].(string)
		siteURL, _ := site["url"].(string)
		enabled, _ := site["enabled"].(bool)
		if name == "" || siteURL == "" {
			return "", fmt.Errorf("add site: name and url are required")
		}
		// Check for duplicate name.
		for _, item := range sitesNode.Content {
			if item.Kind != yaml.MappingNode {
				continue
			}
			if v := yamlMappingLookup(item, "name"); v != nil && v.Value == name {
				return "", fmt.Errorf("site %q already exists", name)
			}
		}
		newSite := buildSiteNode(name, siteURL, enabled)
		sitesNode.Content = append(sitesNode.Content, newSite)

	case "remove":
		name, _ := site["name"].(string)
		if name == "" {
			return "", fmt.Errorf("remove site: name is required")
		}
		filtered := sitesNode.Content[:0]
		found := false
		for _, item := range sitesNode.Content {
			if item.Kind == yaml.MappingNode {
				if v := yamlMappingLookup(item, "name"); v != nil && v.Value == name {
					found = true
					continue
				}
			}
			filtered = append(filtered, item)
		}
		if !found {
			return "", fmt.Errorf("site %q not found", name)
		}
		sitesNode.Content = filtered

	case "enable", "disable":
		name, _ := site["name"].(string)
		if name == "" {
			return "", fmt.Errorf("%s site: name is required", op)
		}
		found := false
		for _, item := range sitesNode.Content {
			if item.Kind != yaml.MappingNode {
				continue
			}
			if v := yamlMappingLookup(item, "name"); v == nil || v.Value != name {
				continue
			}
			found = true
			val := "true"
			if op == "disable" {
				val = "false"
			}
			if !yamlSetMappingScalar(item, "enabled", val) {
				keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "enabled", Tag: "!!str"}
				valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!bool"}
				item.Content = append(item.Content, keyNode, valNode)
			} else {
				// Ensure the value node has the boolean tag.
				for i := 0; i+1 < len(item.Content); i += 2 {
					if item.Content[i].Value == "enabled" {
						item.Content[i+1].Tag = "!!bool"
					}
				}
			}
			break
		}
		if !found {
			return "", fmt.Errorf("site %q not found", name)
		}

	default:
		return "", fmt.Errorf("unknown site operation %q", op)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", fmt.Errorf("encode search.yaml: %w", err)
	}
	_ = enc.Close()
	return buf.String(), nil
}

// buildSiteNode constructs a yaml.Node representing one site entry.
func buildSiteNode(name, siteURL string, enabled bool) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	addScalar := func(key, val, tag string) {
		m.Content = append(m.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: tag},
		)
	}
	addScalar("name", name, "!!str")
	addScalar("url", siteURL, "!!str")
	enabledVal := "true"
	if !enabled {
		enabledVal = "false"
	}
	addScalar("enabled", enabledVal, "!!bool")
	return m
}

// removeYAMLMappingKey removes a key-value pair from a yaml MappingNode.
func removeYAMLMappingKey(m *yaml.Node, key string) {
	out := m.Content[:0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			continue
		}
		out = append(out, m.Content[i], m.Content[i+1])
	}
	m.Content = out
}

// clearYAMLSequence sets the named key's value to an empty sequence, or is a no-op if absent.
func clearYAMLSequence(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			return
		}
	}
}

// yamlSetMappingNode sets the value node of an existing key in a yaml MappingNode.
// Returns true when the key was found and updated.
func yamlSetMappingNode(m *yaml.Node, key string, val *yaml.Node) bool {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = val
			return true
		}
	}
	return false
}

// parseExpressionsBody decodes a JSON body field "expressions" into a []map[string][]string.
func parseExpressionsBody(raw json.RawMessage) ([]map[string][]string, error) {
	if raw == nil {
		return nil, nil
	}
	var out []map[string][]string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse expressions: %w", err)
	}
	return out, nil
}
