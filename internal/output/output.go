package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

// Format is the global output format, set once from the --output flag.
var Format = "table" // "table" | "json" | "yaml"

// Table prints rows in the active format.
// In json/yaml mode it renders the rows as an array of objects keyed by the headers.
func Table(headers []string, rows [][]string) {
	switch strings.ToLower(Format) {
	case "json":
		objects := rowsToObjects(headers, rows)
		JSON(objects)
	case "yaml", "yml":
		objects := rowsToObjects(headers, rows)
		YAML(objects)
	default:
		printTable(headers, rows)
	}
}

// JSON pretty-prints any value as JSON regardless of global format.
func JSON(v interface{}) {
	switch strings.ToLower(Format) {
	case "yaml", "yml":
		YAML(v)
	default:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			return
		}
		fmt.Println(string(data))
	}
}

// YAML prints any value as YAML.
func YAML(v interface{}) {
	// Round-trip through JSON first so map keys are consistent strings
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding: %v\n", err)
		return
	}
	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		fmt.Fprintf(os.Stderr, "error normalizing: %v\n", err)
		return
	}
	data, err := yaml.Marshal(normalized)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding YAML: %v\n", err)
		return
	}
	fmt.Print(string(data))
}

// Success prints a success message (always plain text regardless of format).
func Success(msg string) {
	fmt.Println(msg)
}

func Error(msg string, v ...any) {
	fmt.Printf("! "+msg+"\n", v...)
}

// Fatal prints an error to stderr and exits.
func Fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func printTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
}

// rowsToObjects converts table rows into a slice of maps for JSON/YAML output.
func rowsToObjects(headers []string, rows [][]string) []map[string]string {
	objects := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		obj := make(map[string]string, len(headers))
		for i, h := range headers {
			key := strings.ToLower(strings.ReplaceAll(h, " ", "_"))
			if i < len(row) {
				obj[key] = row[i]
			}
		}
		objects = append(objects, obj)
	}
	return objects
}

// Str safely dereferences a *string for display.
func Str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IntStr converts an int to string.
func IntStr(i int) string {
	return fmt.Sprintf("%d", i)
}

// BoolStr converts a bool to a display string.
func BoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
