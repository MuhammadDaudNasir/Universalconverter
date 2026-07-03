package data

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Format represents structured data formats.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatTOML Format = "toml"
	FormatXML  Format = "xml"
	FormatCSV  Format = "csv"
)

// Convert translates data from one format to another.
func Convert(input string, from, to Format) (string, error) {
	if from == to {
		return input, nil
	}

	// 1. Parse input to generic Go structure (map or slice)
	var rawData interface{}
	var err error

	switch from {
	case FormatJSON:
		decoder := json.NewDecoder(strings.NewReader(input))
		decoder.UseNumber()
		err = decoder.Decode(&rawData)
	case FormatYAML:
		err = yaml.Unmarshal([]byte(input), &rawData)
	case FormatTOML:
		err = toml.Unmarshal([]byte(input), &rawData)
	case FormatXML:
		rawData, err = parseXML(input)
	case FormatCSV:
		rawData, err = parseCSV(input)
	default:
		return "", fmt.Errorf("unsupported source format: %s", from)
	}

	if err != nil {
		return "", fmt.Errorf("failed to parse %s input: %w", from, err)
	}

	// 2. Convert standard structure to target format
	switch to {
	case FormatJSON:
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(rawData); err != nil {
			return "", err
		}
		return buf.String(), nil

	case FormatYAML:
		out, err := yaml.Marshal(rawData)
		if err != nil {
			return "", err
		}
		return string(out), nil

	case FormatTOML:
		out, err := toml.Marshal(rawData)
		if err != nil {
			// TOML only supports tables at the root level (maps). If rawData is a slice, wrap it.
			if _, ok := rawData.(map[string]interface{}); !ok {
				wrapped := map[string]interface{}{"data": rawData}
				out, err = toml.Marshal(wrapped)
				if err != nil {
					return "", err
				}
				return string(out), nil
			}
			return "", err
		}
		return string(out), nil

	case FormatXML:
		return renderXML(rawData)

	case FormatCSV:
		return renderCSV(rawData)

	default:
		return "", fmt.Errorf("unsupported destination format: %s", to)
	}
}

// --- XML Parsing and Rendering ---

type xmlNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Content  string     `xml:",chardata"`
	Children []xmlNode  `xml:",any"`
}

func parseXML(input string) (interface{}, error) {
	var root xmlNode
	decoder := xml.NewDecoder(strings.NewReader(input))
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	return xmlNodeToMap(root), nil
}

func xmlNodeToMap(node xmlNode) interface{} {
	if len(node.Children) == 0 {
		// Just simple content
		val := strings.TrimSpace(node.Content)
		if val == "" && len(node.Attrs) > 0 {
			// Attributes only
			m := make(map[string]interface{})
			for _, attr := range node.Attrs {
				m["@"+attr.Name.Local] = attr.Value
			}
			return m
		}
		return val
	}

	m := make(map[string]interface{})
	for _, attr := range node.Attrs {
		m["@"+attr.Name.Local] = attr.Value
	}

	// Group children by name to handle lists
	childGroups := make(map[string][]xmlNode)
	for _, child := range node.Children {
		name := child.XMLName.Local
		childGroups[name] = append(childGroups[name], child)
	}

	for name, group := range childGroups {
		if len(group) == 1 {
			m[name] = xmlNodeToMap(group[0])
		} else {
			slice := make([]interface{}, len(group))
			for i, child := range group {
				slice[i] = xmlNodeToMap(child)
			}
			m[name] = slice
		}
	}

	if node.Content != "" {
		trimmed := strings.TrimSpace(node.Content)
		if trimmed != "" {
			m["#text"] = trimmed
		}
	}

	return m
}

// renderXML serializes data into XML.
func renderXML(data interface{}) (string, error) {
	type xmlMapEntry struct {
		XMLName xml.Name
		Value   interface{}
	}

	var renderVal func(interface{}) interface{}
	renderVal = func(v interface{}) interface{} {
		switch val := v.(type) {
		case map[string]interface{}:
			var children []interface{}
			for k, vSub := range val {
				if strings.HasPrefix(k, "@") {
					// Attribute representation not supported directly in this simple generic encoder
					continue
				}
				children = append(children, xmlMapEntry{
					XMLName: xml.Name{Local: k},
					Value:   renderVal(vSub),
				})
			}
			return children
		case []interface{}:
			var children []interface{}
			for _, vSub := range val {
				children = append(children, xmlMapEntry{
					XMLName: xml.Name{Local: "item"},
					Value:   renderVal(vSub),
				})
			}
			return children
		default:
			return fmt.Sprintf("%v", val)
		}
	}

	wrapped := xmlMapEntry{
		XMLName: xml.Name{Local: "root"},
		Value:   renderVal(data),
	}

	out, err := xml.MarshalIndent(wrapped, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(out), nil
}

// --- CSV Parsing and Rendering ---

func parseCSV(input string) (interface{}, error) {
	r := csv.NewReader(strings.NewReader(input))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return []interface{}{}, nil
	}

	headers := records[0]
	var list []interface{}

	for _, row := range records[1:] {
		m := make(map[string]interface{})
		for i, val := range row {
			if i < len(headers) {
				m[headers[i]] = val
			} else {
				m[fmt.Sprintf("column_%d", i)] = val
			}
		}
		list = append(list, m)
	}

	return list, nil
}

func renderCSV(data interface{}) (string, error) {
	var records []map[string]interface{}

	// Normalize data to a slice of maps
	switch val := data.(type) {
	case []interface{}:
		for _, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				records = append(records, m)
			}
		}
	case map[string]interface{}:
		records = append(records, val)
	default:
		return "", fmt.Errorf("CSV target format requires array of objects or single object")
	}

	if len(records) == 0 {
		return "", nil
	}

	// 1. Gather all unique headers
	headerMap := make(map[string]bool)
	var headers []string
	for _, rec := range records {
		for k := range rec {
			if !headerMap[k] {
				headerMap[k] = true
				headers = append(headers, k)
			}
		}
	}

	// 2. Write CSV
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(headers); err != nil {
		return "", err
	}

	for _, rec := range records {
		row := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := rec[h]; ok {
				row[i] = fmt.Sprintf("%v", v)
			} else {
				row[i] = ""
			}
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
