// Package transform provides conversion from Notion to Obsidian formats.
// base.go handles Notion database to Obsidian .base file conversion.
package transform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jomei/notionapi"
	"gopkg.in/yaml.v3"
)

// BaseFile represents an Obsidian .base file structure.
// See: https://help.obsidian.md/bases/syntax
type BaseFile struct {
	Filters  *FilterGroup      `yaml:"filters,omitempty"`
	Display  map[string]string `yaml:"display,omitempty"`
	Views    []View            `yaml:"views"`
	Formulas map[string]string `yaml:"formula,omitempty"`
}

// FilterGroup represents a group of filters with AND/OR logic.
type FilterGroup struct {
	And []string `yaml:"and,omitempty"`
	Or  []string `yaml:"or,omitempty"`
}

// View represents a single view configuration in a base file.
type View struct {
	Type    string       `yaml:"type"`
	Name    string       `yaml:"name"`
	Columns []ViewColumn `yaml:"columns,omitempty"`
	Order   []ViewOrder  `yaml:"order,omitempty"`
	Filters *FilterGroup `yaml:"filters,omitempty"`
}

// ViewColumn represents a column in a table view.
type ViewColumn struct {
	Property string `yaml:"property"`
	Width    int    `yaml:"width,omitempty"`
}

// ViewOrder represents sorting order for a view.
type ViewOrder struct {
	Property string `yaml:"property"`
	Order    string `yaml:"order"` // "asc" or "desc"
}

// PropertyMapping maps a Notion property to Obsidian frontmatter.
type PropertyMapping struct {
	Name       string // Display name for the property
	Type       string // Obsidian property type (text, number, list, date, checkbox)
	NotionType string // Original Notion type
	IsTitle    bool   // Whether this is the title property
}

// DatabaseSchema holds the parsed schema from a Notion database.
type DatabaseSchema struct {
	Title         string
	Properties    map[string]PropertyMapping
	TitleProperty string // Name of the title property
}

// EntryData represents the data for a single database entry.
type EntryData struct {
	Title      string
	Properties map[string]any
	PageID     string
}

// ParseDatabaseSchema extracts the schema from a Notion database.
func ParseDatabaseSchema(db *notionapi.Database) (*DatabaseSchema, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	schema := &DatabaseSchema{
		Title:      extractRichText(db.Title),
		Properties: make(map[string]PropertyMapping),
	}

	for name, prop := range db.Properties {
		mapping := mapNotionPropertyType(name, prop)
		schema.Properties[name] = mapping
		if mapping.IsTitle {
			schema.TitleProperty = name
		}
	}

	return schema, nil
}

// mapNotionPropertyType converts a Notion property definition to our mapping.
func mapNotionPropertyType(name string, prop notionapi.PropertyConfig) PropertyMapping {
	mapping := PropertyMapping{
		Name:       name,
		NotionType: string(prop.GetType()),
	}

	switch prop.GetType() {
	case notionapi.PropertyConfigTypeTitle:
		mapping.Type = "text"
		mapping.IsTitle = true

	case notionapi.PropertyConfigTypeRichText:
		mapping.Type = "text"

	case notionapi.PropertyConfigTypeNumber:
		mapping.Type = "number"

	case notionapi.PropertyConfigTypeSelect:
		mapping.Type = "text"

	case notionapi.PropertyConfigTypeMultiSelect:
		mapping.Type = "list"

	case notionapi.PropertyConfigTypeDate:
		mapping.Type = "date"

	case notionapi.PropertyConfigTypeCheckbox:
		mapping.Type = "checkbox"

	case notionapi.PropertyConfigTypeURL:
		mapping.Type = "text"

	case notionapi.PropertyConfigTypeEmail:
		mapping.Type = "text"

	case notionapi.PropertyConfigTypePhoneNumber:
		mapping.Type = "text"

	case notionapi.PropertyConfigTypeRelation:
		mapping.Type = "list" // Will contain wiki-links

	case notionapi.PropertyConfigTypeFormula:
		mapping.Type = "text" // Store computed value

	case notionapi.PropertyConfigTypeRollup:
		mapping.Type = "text" // Store computed value

	case notionapi.PropertyConfigTypePeople:
		mapping.Type = "list"

	case notionapi.PropertyConfigTypeFiles:
		mapping.Type = "list" // Will contain file paths

	case notionapi.PropertyConfigCreatedTime:
		mapping.Type = "datetime"

	case notionapi.PropertyConfigCreatedBy:
		mapping.Type = "text"

	case notionapi.PropertyConfigLastEditedTime:
		mapping.Type = "datetime"

	case notionapi.PropertyConfigLastEditedBy:
		mapping.Type = "text"

	case notionapi.PropertyConfigStatus:
		mapping.Type = "text"

	default:
		mapping.Type = "text"
	}

	return mapping
}

// GenerateBaseFile creates an Obsidian .base file YAML for a database.
func GenerateBaseFile(schema *DatabaseSchema, folderPath string) (*BaseFile, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	base := &BaseFile{
		Filters: &FilterGroup{
			And: []string{
				fmt.Sprintf("file.inFolder(\"%s\")", folderPath),
				"file.ext == \"md\"",
			},
		},
		Display: make(map[string]string),
		Views: []View{
			{
				Type:    "table",
				Name:    "Table",
				Columns: buildViewColumns(schema),
				Order: []ViewOrder{
					{Property: "file.name", Order: "asc"},
				},
			},
		},
	}

	// Add display names (property name -> display name mapping)
	for name := range schema.Properties {
		// Use the original Notion property name as display name
		base.Display[sanitizePropertyName(name)] = name
	}

	return base, nil
}

// buildViewColumns creates the columns for the default table view.
func buildViewColumns(schema *DatabaseSchema) []ViewColumn {
	var columns []ViewColumn

	// Always start with file name column
	columns = append(columns, ViewColumn{Property: "file.name"})

	// Get property names and sort them for consistent output
	var propNames []string
	for name, prop := range schema.Properties {
		// Skip title property (it's the file name)
		if prop.IsTitle {
			continue
		}
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	// Add other properties as columns
	for _, name := range propNames {
		columns = append(columns, ViewColumn{
			Property: sanitizePropertyName(name),
		})
	}

	return columns
}

// sanitizePropertyName converts a property name to be valid in YAML/bases.
func sanitizePropertyName(name string) string {
	// Replace characters that might cause issues
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToLower(name)
	return name
}

// MarshalBaseFile converts a BaseFile to YAML bytes.
func MarshalBaseFile(base *BaseFile) ([]byte, error) {
	return yaml.Marshal(base)
}

// ExtractEntryData extracts frontmatter data from a Notion database page.
func ExtractEntryData(page *notionapi.Page, schema *DatabaseSchema) (*EntryData, error) {
	if page == nil {
		return nil, fmt.Errorf("page is nil")
	}

	entry := &EntryData{
		Properties: make(map[string]any),
		PageID:     string(page.ID),
	}

	for name, propValue := range page.Properties {
		value := extractPropertyValue(propValue)

		// Check if this is the title property
		if mapping, ok := schema.Properties[name]; ok && mapping.IsTitle {
			if strVal, ok := value.(string); ok {
				entry.Title = strVal
			}
		} else if value != nil {
			// Use sanitized property name for frontmatter
			entry.Properties[sanitizePropertyName(name)] = value
		}
	}

	return entry, nil
}

// extractPropertyValue extracts the value from a Notion property.
func extractPropertyValue(prop notionapi.Property) any {
	if prop == nil {
		return nil
	}

	switch p := prop.(type) {
	case *notionapi.TitleProperty:
		return extractRichText(p.Title)

	case *notionapi.RichTextProperty:
		return extractRichText(p.RichText)

	case *notionapi.NumberProperty:
		// NumberProperty.Number is float64, not a pointer
		// Return the value - zero is a valid value
		return p.Number

	case *notionapi.SelectProperty:
		// SelectProperty.Select is Option, not a pointer
		// Check if it's empty by checking Name
		if p.Select.Name == "" {
			return nil
		}
		return p.Select.Name

	case *notionapi.MultiSelectProperty:
		var values []string
		for _, opt := range p.MultiSelect {
			values = append(values, opt.Name)
		}
		if len(values) == 0 {
			return nil
		}
		return values

	case *notionapi.DateProperty:
		if p.Date == nil || p.Date.Start == nil {
			return nil
		}
		start := p.Date.Start.String()
		if p.Date.End != nil {
			return fmt.Sprintf("%s/%s", start, p.Date.End.String())
		}
		return start

	case *notionapi.CheckboxProperty:
		return p.Checkbox

	case *notionapi.URLProperty:
		if p.URL == "" {
			return nil
		}
		return p.URL

	case *notionapi.EmailProperty:
		if p.Email == "" {
			return nil
		}
		return p.Email

	case *notionapi.PhoneNumberProperty:
		if p.PhoneNumber == "" {
			return nil
		}
		return p.PhoneNumber

	case *notionapi.RelationProperty:
		var links []string
		for _, rel := range p.Relation {
			// Convert to wiki-link format
			links = append(links, fmt.Sprintf("[[%s]]", string(rel.ID)))
		}
		if len(links) == 0 {
			return nil
		}
		return links

	case *notionapi.FormulaProperty:
		return extractFormulaValue(p.Formula)

	case *notionapi.RollupProperty:
		return extractRollupValue(p.Rollup)

	case *notionapi.PeopleProperty:
		var names []string
		for _, user := range p.People {
			if user.Name != "" {
				names = append(names, user.Name)
			}
		}
		if len(names) == 0 {
			return nil
		}
		return names

	case *notionapi.FilesProperty:
		var files []string
		for _, f := range p.Files {
			url := ""
			if f.File != nil {
				url = f.File.URL
			} else if f.External != nil {
				url = f.External.URL
			}
			if url != "" {
				files = append(files, url)
			}
		}
		if len(files) == 0 {
			return nil
		}
		return files

	case *notionapi.CreatedTimeProperty:
		return p.CreatedTime.String()

	case *notionapi.CreatedByProperty:
		return p.CreatedBy.Name

	case *notionapi.LastEditedTimeProperty:
		return p.LastEditedTime.String()

	case *notionapi.LastEditedByProperty:
		return p.LastEditedBy.Name

	case *notionapi.StatusProperty:
		// StatusProperty.Status is Status (Option), not a pointer
		if p.Status.Name == "" {
			return nil
		}
		return p.Status.Name

	default:
		return nil
	}
}

// extractFormulaValue extracts the computed value from a formula property.
// Formula is notionapi.Formula (not a pointer).
func extractFormulaValue(formula notionapi.Formula) any {
	switch formula.Type {
	case notionapi.FormulaTypeString:
		if formula.String != "" {
			return formula.String
		}
	case notionapi.FormulaTypeNumber:
		return formula.Number
	case notionapi.FormulaTypeBoolean:
		return formula.Boolean
	case notionapi.FormulaTypeDate:
		if formula.Date != nil && formula.Date.Start != nil {
			return formula.Date.Start.String()
		}
	}
	return nil
}

// extractRollupValue extracts the computed value from a rollup property.
// Rollup is notionapi.Rollup (not a pointer).
func extractRollupValue(rollup notionapi.Rollup) any {
	switch rollup.Type {
	case notionapi.RollupTypeNumber:
		return rollup.Number
	case notionapi.RollupTypeDate:
		if rollup.Date != nil && rollup.Date.Start != nil {
			return rollup.Date.Start.String()
		}
	case notionapi.RollupTypeArray:
		// For array rollups, recursively extract values
		if len(rollup.Array) > 0 {
			var values []any
			for _, item := range rollup.Array {
				val := extractPropertyValue(item)
				if val != nil {
					values = append(values, val)
				}
			}
			return values
		}
	}
	return nil
}

// extractRichText extracts plain text from rich text array.
func extractRichText(richText []notionapi.RichText) string {
	var sb strings.Builder
	for _, rt := range richText {
		sb.WriteString(rt.PlainText)
	}
	return sb.String()
}

// GenerateFrontmatter creates YAML frontmatter for a database entry.
func GenerateFrontmatter(entry *EntryData) (string, error) {
	if entry == nil {
		return "", fmt.Errorf("entry is nil")
	}

	if len(entry.Properties) == 0 {
		return "", nil
	}

	// Add notion_id for tracking
	props := make(map[string]any)
	for k, v := range entry.Properties {
		props[k] = v
	}
	props["notion_id"] = entry.PageID

	data, err := yaml.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter: %w", err)
	}

	return fmt.Sprintf("---\n%s---\n", string(data)), nil
}

// DatabaseEntry represents a complete entry ready to be written.
type DatabaseEntry struct {
	Filename    string
	Frontmatter string
	Content     string // Markdown content from page blocks
}

// BuildDatabaseEntry creates a complete entry from a page and its content.
func BuildDatabaseEntry(entry *EntryData, markdownContent string) (*DatabaseEntry, error) {
	if entry == nil {
		return nil, fmt.Errorf("entry is nil")
	}

	frontmatter, err := GenerateFrontmatter(entry)
	if err != nil {
		return nil, err
	}

	// Sanitize filename
	filename := sanitizeFilename(entry.Title)
	if filename == "" {
		filename = entry.PageID
	}

	return &DatabaseEntry{
		Filename:    filename + ".md",
		Frontmatter: frontmatter,
		Content:     markdownContent,
	}, nil
}

// sanitizeFilename makes a string safe for use as a filename.
func sanitizeFilename(name string) string {
	// Replace characters that are problematic in filenames
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
		"\n", " ",
		"\r", "",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)

	// Limit length
	if len(name) > 200 {
		name = name[:200]
	}

	return name
}
