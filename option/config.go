package option

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"

	"gopkg.in/yaml.v3"
)

// ConfigFormat identifies how a configuration document is serialized.
type ConfigFormat uint8

const (
	// ConfigFormatJSON is the default sing-box configuration document format.
	ConfigFormatJSON ConfigFormat = iota
	// ConfigFormatYAML enables loading configurations authored in YAML.
	// Only full configuration documents are supported; other resources,
	// such as standalone rule-set files, remain JSON or binary format.
	ConfigFormatYAML
)

// String returns a human-readable name for the configuration format.
func (f ConfigFormat) String() string {
	switch f {
	case ConfigFormatYAML:
		return "yaml"
	default:
		return "json"
	}
}

// IsConfigFile reports whether path has an extension recognized as a
// sing-box configuration file (.json, .yaml, or .yml).
func IsConfigFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

// DetectConfigFormat returns the configuration format of path.
// The file extension takes precedence. When it is not recognized —
// such as for "stdin", empty paths, or files with other extensions —
// the leading non-whitespace bytes of content are inspected instead.
func DetectConfigFormat(path string, content []byte) ConfigFormat {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return ConfigFormatJSON
	case ".yaml", ".yml":
		return ConfigFormatYAML
	}
	return detectContentFormat(content)
}

func detectContentFormat(content []byte) ConfigFormat {
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	content = bytes.TrimLeft(content, " \t\r\n")
	if len(content) == 0 {
		return ConfigFormatJSON
	}
	switch content[0] {
	case '{', '[', '/':
		return ConfigFormatJSON
	default:
		return ConfigFormatYAML
	}
}

// UnmarshalConfig decodes a sing-box configuration document in format.
// For YAML documents, the content is first normalized into standard JSON
// so that the decoded Options.RawMessage is always JSON-compatible,
// preserving identical merge and sub-decoder behavior across formats.
func UnmarshalConfig(ctx context.Context, content []byte, format ConfigFormat) (Options, error) {
	if format == ConfigFormatYAML {
		jsonContent, err := yamlToJSON(content)
		if err != nil {
			return Options{}, err
		}
		content = jsonContent
	}
	return json.UnmarshalExtendedContext[Options](ctx, content)
}

// JSONToYAML converts a JSON configuration document to its YAML representation.
// Numbers are converted to their native numeric types to avoid loss of precision
// and map keys are sorted for deterministic output.
func JSONToYAML(content []byte) ([]byte, error) {
	decoder := stdjson.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	var document any
	err := decoder.Decode(&document)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return []byte("{}\n"), nil
		}
		return nil, E.Cause(err, "decode JSON document")
	}
	var next any
	if err = decoder.Decode(&next); !errors.Is(err, io.EOF) {
		return nil, E.New("multiple JSON values are not supported")
	}
	document, err = normalizeJSONNumbers(document)
	if err != nil {
		return nil, err
	}
	output, err := yaml.Marshal(document)
	if err != nil {
		return nil, E.Cause(err, "encode YAML document")
	}
	return output, nil
}

func normalizeJSONNumbers(value any) (any, error) {
	switch v := value.(type) {
	case stdjson.Number:
		if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return i, nil
		}
		if u, err := strconv.ParseUint(string(v), 10, 64); err == nil {
			return u, nil
		}
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return nil, E.Cause(err, "parse JSON number ", string(v))
		}
		return f, nil
	case map[string]any:
		for k, child := range v {
			normalized, err := normalizeJSONNumbers(child)
			if err != nil {
				return nil, err
			}
			v[k] = normalized
		}
		return v, nil
	case []any:
		for i, child := range v {
			normalized, err := normalizeJSONNumbers(child)
			if err != nil {
				return nil, err
			}
			v[i] = normalized
		}
		return v, nil
	default:
		return value, nil
	}
}

func yamlToJSON(content []byte) ([]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var document yaml.Node
	err := decoder.Decode(&document)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return []byte("{}"), nil
		}
		return nil, E.Cause(err, "decode YAML document")
	}
	var next yaml.Node
	if err = decoder.Decode(&next); !errors.Is(err, io.EOF) {
		return nil, E.New("multiple YAML documents are not supported")
	}
	var buffer bytes.Buffer
	err = writeYAMLNode(&buffer, &document)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeYAMLNode(buffer *bytes.Buffer, node *yaml.Node) error {
	node = resolveAlias(node)
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			buffer.WriteString("{}")
			return nil
		}
		return writeYAMLNode(buffer, node.Content[0])
	case yaml.ScalarNode:
		return writeYAMLScalar(buffer, node)
	case yaml.SequenceNode:
		buffer.WriteByte('[')
		for i, child := range node.Content {
			if i > 0 {
				buffer.WriteByte(',')
			}
			if err := writeYAMLNode(buffer, child); err != nil {
				return err
			}
		}
		buffer.WriteByte(']')
		return nil
	case yaml.MappingNode:
		return writeYAMLMapping(buffer, node)
	default:
		return E.New("unsupported YAML node at line ", node.Line)
	}
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

func writeYAMLScalar(buffer *bytes.Buffer, node *yaml.Node) error {
	tag := node.Tag
	if tag == "" && node.Style != 0 {
		tag = "!!str"
	}
	switch tag {
	case "!!null":
		buffer.WriteString("null")
		return nil
	case "!!bool":
		var b bool
		if err := node.Decode(&b); err != nil {
			return E.Cause(err, "decode bool at line ", node.Line)
		}
		if b {
			buffer.WriteString("true")
		} else {
			buffer.WriteString("false")
		}
		return nil
	case "!!int", "!!float":
		var v any
		if err := node.Decode(&v); err != nil {
			return E.Cause(err, "decode number at line ", node.Line)
		}
		encoded, err := stdjson.Marshal(v)
		if err != nil {
			return E.Cause(err, "encode number at line ", node.Line)
		}
		buffer.Write(encoded)
		return nil
	case "!!binary":
		// Emit the base64 literal as a plain JSON string.
		encoded, _ := stdjson.Marshal(node.Value)
		buffer.Write(encoded)
		return nil
	case "!!timestamp":
		// Sing-box options have no timestamp fields; retain the literal
		// text so date-like values survive as strings.
		encoded, _ := stdjson.Marshal(node.Value)
		buffer.Write(encoded)
		return nil
	default:
		// Generic or untyped scalar: unmarshal via Decode to let the
		// resolver handle any edge type, falling back to literal string.
		var v any
		if err := node.Decode(&v); err == nil {
			switch val := v.(type) {
			case nil:
				buffer.WriteString("null")
				return nil
			case bool:
				if val {
					buffer.WriteString("true")
				} else {
					buffer.WriteString("false")
				}
				return nil
			case int, int64, uint64, float64:
				encoded, mErr := stdjson.Marshal(val)
				if mErr == nil {
					buffer.Write(encoded)
					return nil
				}
			case time.Time:
				encoded, _ := stdjson.Marshal(node.Value)
				buffer.Write(encoded)
				return nil
			}
		}
		encoded, err := stdjson.Marshal(node.Value)
		if err != nil {
			return E.Cause(err, "encode string at line ", node.Line)
		}
		buffer.Write(encoded)
		return nil
	}
}

type yamlEntry struct {
	key   string
	value *yaml.Node
}

func isMergeKey(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode &&
		node.Value == "<<" &&
		(node.Tag == "" || node.Tag == "!" || node.Tag == "!!merge")
}

func writeYAMLMapping(buffer *bytes.Buffer, node *yaml.Node) error {
	var (
		entries []yamlEntry
		seen    = make(map[string]bool)
	)

	// Pass 1: explicit keys in document order.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := resolveAlias(node.Content[i])
		valueNode := node.Content[i+1]
		if isMergeKey(keyNode) {
			continue
		}
		key, err := yamlKeyString(keyNode)
		if err != nil {
			return err
		}
		if seen[key] {
			return E.New("duplicate mapping key ", key, " at line ", keyNode.Line)
		}
		seen[key] = true
		entries = append(entries, yamlEntry{key: key, value: valueNode})
	}

	// Pass 2: merged keys (<<).
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := resolveAlias(node.Content[i])
		if !isMergeKey(keyNode) {
			continue
		}
		valueNode := node.Content[i+1]
		err := appendMergedEntries(&entries, seen, valueNode)
		if err != nil {
			return err
		}
	}

	buffer.WriteByte('{')
	for i, entry := range entries {
		if i > 0 {
			buffer.WriteByte(',')
		}
		keyJSON, _ := stdjson.Marshal(entry.key)
		buffer.Write(keyJSON)
		buffer.WriteByte(':')
		if err := writeYAMLNode(buffer, entry.value); err != nil {
			return err
		}
	}
	buffer.WriteByte('}')
	return nil
}

func appendMergedEntries(entries *[]yamlEntry, seen map[string]bool, source *yaml.Node) error {
	source = resolveAlias(source)
	switch source.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(source.Content); i += 2 {
			keyNode := resolveAlias(source.Content[i])
			valueNode := source.Content[i+1]
			if isMergeKey(keyNode) {
				if err := appendMergedEntries(entries, seen, valueNode); err != nil {
					return err
				}
				continue
			}
			key, err := yamlKeyString(keyNode)
			if err != nil {
				return err
			}
			if !seen[key] {
				seen[key] = true
				*entries = append(*entries, yamlEntry{key: key, value: valueNode})
			}
		}
		return nil
	case yaml.SequenceNode:
		for _, child := range source.Content {
			if err := appendMergedEntries(entries, seen, child); err != nil {
				return err
			}
		}
		return nil
	default:
		return E.New("unsupported merge source at line ", source.Line)
	}
}

func yamlKeyString(node *yaml.Node) (string, error) {
	node = resolveAlias(node)
	if node.Kind != yaml.ScalarNode {
		return "", E.New("mapping key at line ", node.Line, " must be a scalar")
	}
	return node.Value, nil
}
