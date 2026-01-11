package transform

import (
	"testing"
	"time"

	"github.com/jomei/notionapi"
	"gopkg.in/yaml.v3"
)

func TestParseDatabaseSchema(t *testing.T) {
	tests := []struct {
		name    string
		db      *notionapi.Database
		wantErr bool
		check   func(*testing.T, *DatabaseSchema)
	}{
		{
			name:    "nil database",
			db:      nil,
			wantErr: true,
		},
		{
			name: "empty database",
			db: &notionapi.Database{
				Title:      []notionapi.RichText{{PlainText: "Test DB"}},
				Properties: make(notionapi.PropertyConfigs),
			},
			check: func(t *testing.T, schema *DatabaseSchema) {
				if schema.Title != "Test DB" {
					t.Errorf("got title %q, want %q", schema.Title, "Test DB")
				}
				if len(schema.Properties) != 0 {
					t.Errorf("got %d properties, want 0", len(schema.Properties))
				}
			},
		},
		{
			name: "database with various property types",
			db: &notionapi.Database{
				Title: []notionapi.RichText{{PlainText: "Tasks"}},
				Properties: notionapi.PropertyConfigs{
					"Name": &notionapi.TitlePropertyConfig{
						Type: notionapi.PropertyConfigTypeTitle,
					},
					"Status": &notionapi.SelectPropertyConfig{
						Type: notionapi.PropertyConfigTypeSelect,
					},
					"Tags": &notionapi.MultiSelectPropertyConfig{
						Type: notionapi.PropertyConfigTypeMultiSelect,
					},
					"Due Date": &notionapi.DatePropertyConfig{
						Type: notionapi.PropertyConfigTypeDate,
					},
					"Priority": &notionapi.NumberPropertyConfig{
						Type: notionapi.PropertyConfigTypeNumber,
					},
					"Done": &notionapi.CheckboxPropertyConfig{
						Type: notionapi.PropertyConfigTypeCheckbox,
					},
					"URL": &notionapi.URLPropertyConfig{
						Type: notionapi.PropertyConfigTypeURL,
					},
				},
			},
			check: func(t *testing.T, schema *DatabaseSchema) {
				if schema.Title != "Tasks" {
					t.Errorf("got title %q, want %q", schema.Title, "Tasks")
				}
				if len(schema.Properties) != 7 {
					t.Errorf("got %d properties, want 7", len(schema.Properties))
				}

				// Check title property
				if schema.TitleProperty != "Name" {
					t.Errorf("got title property %q, want %q", schema.TitleProperty, "Name")
				}

				// Check property types
				expected := map[string]string{
					"Name":     "text",
					"Status":   "text",
					"Tags":     "list",
					"Due Date": "date",
					"Priority": "number",
					"Done":     "checkbox",
					"URL":      "text",
				}
				for name, wantType := range expected {
					if prop, ok := schema.Properties[name]; ok {
						if prop.Type != wantType {
							t.Errorf("property %q: got type %q, want %q", name, prop.Type, wantType)
						}
					} else {
						t.Errorf("missing property %q", name)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := ParseDatabaseSchema(tt.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDatabaseSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && schema != nil {
				tt.check(t, schema)
			}
		})
	}
}

func TestMapNotionPropertyType(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		prop     notionapi.PropertyConfig
		wantType string
		isTitle  bool
	}{
		{
			name:     "title property",
			propName: "Name",
			prop:     &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
			wantType: "text",
			isTitle:  true,
		},
		{
			name:     "rich text property",
			propName: "Description",
			prop:     &notionapi.RichTextPropertyConfig{Type: notionapi.PropertyConfigTypeRichText},
			wantType: "text",
		},
		{
			name:     "number property",
			propName: "Count",
			prop:     &notionapi.NumberPropertyConfig{Type: notionapi.PropertyConfigTypeNumber},
			wantType: "number",
		},
		{
			name:     "select property",
			propName: "Status",
			prop:     &notionapi.SelectPropertyConfig{Type: notionapi.PropertyConfigTypeSelect},
			wantType: "text",
		},
		{
			name:     "multi-select property",
			propName: "Tags",
			prop:     &notionapi.MultiSelectPropertyConfig{Type: notionapi.PropertyConfigTypeMultiSelect},
			wantType: "list",
		},
		{
			name:     "date property",
			propName: "Due",
			prop:     &notionapi.DatePropertyConfig{Type: notionapi.PropertyConfigTypeDate},
			wantType: "date",
		},
		{
			name:     "checkbox property",
			propName: "Done",
			prop:     &notionapi.CheckboxPropertyConfig{Type: notionapi.PropertyConfigTypeCheckbox},
			wantType: "checkbox",
		},
		{
			name:     "relation property",
			propName: "Related",
			prop:     &notionapi.RelationPropertyConfig{Type: notionapi.PropertyConfigTypeRelation},
			wantType: "list",
		},
		{
			name:     "formula property",
			propName: "Computed",
			prop:     &notionapi.FormulaPropertyConfig{Type: notionapi.PropertyConfigTypeFormula},
			wantType: "text",
		},
		{
			name:     "rollup property",
			propName: "Sum",
			prop:     &notionapi.RollupPropertyConfig{Type: notionapi.PropertyConfigTypeRollup},
			wantType: "text",
		},
		{
			name:     "people property",
			propName: "Assignee",
			prop:     &notionapi.PeoplePropertyConfig{Type: notionapi.PropertyConfigTypePeople},
			wantType: "list",
		},
		{
			name:     "files property",
			propName: "Attachments",
			prop:     &notionapi.FilesPropertyConfig{Type: notionapi.PropertyConfigTypeFiles},
			wantType: "list",
		},
		{
			name:     "created time property",
			propName: "Created",
			prop:     &notionapi.CreatedTimePropertyConfig{Type: notionapi.PropertyConfigCreatedTime},
			wantType: "datetime",
		},
		{
			name:     "status property",
			propName: "Status",
			prop:     &notionapi.StatusPropertyConfig{Type: notionapi.PropertyConfigStatus},
			wantType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := mapNotionPropertyType(tt.propName, tt.prop)
			if mapping.Type != tt.wantType {
				t.Errorf("got type %q, want %q", mapping.Type, tt.wantType)
			}
			if mapping.IsTitle != tt.isTitle {
				t.Errorf("got isTitle %v, want %v", mapping.IsTitle, tt.isTitle)
			}
			if mapping.Name != tt.propName {
				t.Errorf("got name %q, want %q", mapping.Name, tt.propName)
			}
		})
	}
}

func TestGenerateBaseFile(t *testing.T) {
	schema := &DatabaseSchema{
		Title: "Test Database",
		Properties: map[string]PropertyMapping{
			"Name":   {Name: "Name", Type: "text", IsTitle: true},
			"Status": {Name: "Status", Type: "text"},
			"Tags":   {Name: "Tags", Type: "list"},
		},
		TitleProperty: "Name",
	}

	base, err := GenerateBaseFile(schema, "Databases/Test Database")
	if err != nil {
		t.Fatalf("GenerateBaseFile() error = %v", err)
	}

	// Check filters
	if base.Filters == nil {
		t.Fatal("filters is nil")
	}
	if len(base.Filters.And) != 2 {
		t.Errorf("got %d filters, want 2", len(base.Filters.And))
	}

	// Check views
	if len(base.Views) != 1 {
		t.Errorf("got %d views, want 1", len(base.Views))
	}
	if base.Views[0].Type != "table" {
		t.Errorf("got view type %q, want %q", base.Views[0].Type, "table")
	}
	if base.Views[0].Name != "Table" {
		t.Errorf("got view name %q, want %q", base.Views[0].Name, "Table")
	}

	// Check columns - should have file.name + non-title properties
	if len(base.Views[0].Columns) != 3 { // file.name, status, tags
		t.Errorf("got %d columns, want 3", len(base.Views[0].Columns))
	}
	if base.Views[0].Columns[0].Property != "file.name" {
		t.Errorf("first column should be file.name, got %q", base.Views[0].Columns[0].Property)
	}

	// Check display names
	if len(base.Display) != 3 {
		t.Errorf("got %d display names, want 3", len(base.Display))
	}
}

func TestGenerateBaseFile_NilSchema(t *testing.T) {
	_, err := GenerateBaseFile(nil, "test")
	if err == nil {
		t.Error("expected error for nil schema")
	}
}

func TestMarshalBaseFile(t *testing.T) {
	base := &BaseFile{
		Filters: &FilterGroup{
			And: []string{
				`file.inFolder("Test")`,
				`file.ext == "md"`,
			},
		},
		Display: map[string]string{
			"status": "Status",
		},
		Views: []View{
			{
				Type: "table",
				Name: "Table",
				Columns: []ViewColumn{
					{Property: "file.name"},
					{Property: "status"},
				},
			},
		},
	}

	data, err := MarshalBaseFile(base)
	if err != nil {
		t.Fatalf("MarshalBaseFile() error = %v", err)
	}

	// Verify it's valid YAML
	var parsed BaseFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML output: %v", err)
	}

	// Verify content
	if len(parsed.Filters.And) != 2 {
		t.Errorf("got %d filters, want 2", len(parsed.Filters.And))
	}
	if len(parsed.Views) != 1 {
		t.Errorf("got %d views, want 1", len(parsed.Views))
	}
}

func TestExtractEntryData(t *testing.T) {
	schema := &DatabaseSchema{
		Title: "Test",
		Properties: map[string]PropertyMapping{
			"Name":   {Name: "Name", Type: "text", IsTitle: true},
			"Status": {Name: "Status", Type: "text"},
			"Count":  {Name: "Count", Type: "number"},
		},
		TitleProperty: "Name",
	}

	page := &notionapi.Page{
		ID: "test-page-id",
		Properties: notionapi.Properties{
			"Name": &notionapi.TitleProperty{
				Title: []notionapi.RichText{{PlainText: "Test Entry"}},
			},
			"Status": &notionapi.SelectProperty{
				Select: notionapi.Option{Name: "Active"},
			},
			"Count": &notionapi.NumberProperty{
				Number: 42,
			},
		},
	}

	entry, err := ExtractEntryData(page, schema)
	if err != nil {
		t.Fatalf("ExtractEntryData() error = %v", err)
	}

	if entry.Title != "Test Entry" {
		t.Errorf("got title %q, want %q", entry.Title, "Test Entry")
	}
	if entry.PageID != "test-page-id" {
		t.Errorf("got pageID %q, want %q", entry.PageID, "test-page-id")
	}

	// Check status property
	if status, ok := entry.Properties["status"].(string); !ok || status != "Active" {
		t.Errorf("got status %v, want %q", entry.Properties["status"], "Active")
	}

	// Check count property
	if c, ok := entry.Properties["count"].(float64); !ok || c != 42 {
		t.Errorf("got count %v, want 42", entry.Properties["count"])
	}
}

func TestExtractPropertyValue_Relation(t *testing.T) {
	prop := &notionapi.RelationProperty{
		Relation: []notionapi.Relation{
			{ID: "page-1"},
			{ID: "page-2"},
		},
	}

	value := extractPropertyValue(prop)
	links, ok := value.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", value)
	}

	if len(links) != 2 {
		t.Errorf("got %d links, want 2", len(links))
	}
	if links[0] != "[[page-1]]" {
		t.Errorf("got link %q, want %q", links[0], "[[page-1]]")
	}
	if links[1] != "[[page-2]]" {
		t.Errorf("got link %q, want %q", links[1], "[[page-2]]")
	}
}

func TestExtractPropertyValue_MultiSelect(t *testing.T) {
	prop := &notionapi.MultiSelectProperty{
		MultiSelect: []notionapi.Option{
			{Name: "Tag1"},
			{Name: "Tag2"},
			{Name: "Tag3"},
		},
	}

	value := extractPropertyValue(prop)
	tags, ok := value.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", value)
	}

	if len(tags) != 3 {
		t.Errorf("got %d tags, want 3", len(tags))
	}
	expected := []string{"Tag1", "Tag2", "Tag3"}
	for i, tag := range tags {
		if tag != expected[i] {
			t.Errorf("tag %d: got %q, want %q", i, tag, expected[i])
		}
	}
}

func TestExtractPropertyValue_Date(t *testing.T) {
	startTime := notionapi.Date(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	endTime := notionapi.Date(time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name string
		prop *notionapi.DateProperty
		want string
	}{
		{
			name: "date only",
			prop: &notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &startTime,
				},
			},
			want: "2024-01-15T00:00:00Z",
		},
		{
			name: "date range",
			prop: &notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &startTime,
					End:   &endTime,
				},
			},
			want: "2024-01-15T00:00:00Z/2024-01-20T00:00:00Z",
		},
		{
			name: "nil date",
			prop: &notionapi.DateProperty{
				Date: nil,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := extractPropertyValue(tt.prop)
			if tt.want == "" {
				if value != nil {
					t.Errorf("got %v, want nil", value)
				}
				return
			}
			str, ok := value.(string)
			if !ok {
				t.Fatalf("expected string, got %T", value)
			}
			if str != tt.want {
				t.Errorf("got %q, want %q", str, tt.want)
			}
		})
	}
}

func TestExtractPropertyValue_Formula(t *testing.T) {
	tests := []struct {
		name string
		prop *notionapi.FormulaProperty
		want any
	}{
		{
			name: "string formula",
			prop: &notionapi.FormulaProperty{
				Formula: notionapi.Formula{
					Type:   notionapi.FormulaTypeString,
					String: "computed",
				},
			},
			want: "computed",
		},
		{
			name: "number formula",
			prop: &notionapi.FormulaProperty{
				Formula: notionapi.Formula{
					Type:   notionapi.FormulaTypeNumber,
					Number: 100,
				},
			},
			want: float64(100),
		},
		{
			name: "boolean formula",
			prop: &notionapi.FormulaProperty{
				Formula: notionapi.Formula{
					Type:    notionapi.FormulaTypeBoolean,
					Boolean: true,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := extractPropertyValue(tt.prop)
			if value != tt.want {
				t.Errorf("got %v, want %v", value, tt.want)
			}
		})
	}
}

func TestExtractPropertyValue_People(t *testing.T) {
	prop := &notionapi.PeopleProperty{
		People: []notionapi.User{
			{Name: "Alice"},
			{Name: "Bob"},
		},
	}

	value := extractPropertyValue(prop)
	names, ok := value.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", value)
	}

	if len(names) != 2 {
		t.Errorf("got %d names, want 2", len(names))
	}
	if names[0] != "Alice" || names[1] != "Bob" {
		t.Errorf("got names %v, want [Alice, Bob]", names)
	}
}

func TestExtractPropertyValue_Checkbox(t *testing.T) {
	trueVal := &notionapi.CheckboxProperty{Checkbox: true}
	falseVal := &notionapi.CheckboxProperty{Checkbox: false}

	if v := extractPropertyValue(trueVal); v != true {
		t.Errorf("got %v, want true", v)
	}
	if v := extractPropertyValue(falseVal); v != false {
		t.Errorf("got %v, want false", v)
	}
}

func TestGenerateFrontmatter(t *testing.T) {
	entry := &EntryData{
		Title:  "Test Entry",
		PageID: "abc123",
		Properties: map[string]any{
			"status": "Active",
			"tags":   []string{"tag1", "tag2"},
			"count":  42,
		},
	}

	frontmatter, err := GenerateFrontmatter(entry)
	if err != nil {
		t.Fatalf("GenerateFrontmatter() error = %v", err)
	}

	// Check format
	if !hasPrefix(frontmatter, "---\n") {
		t.Error("frontmatter should start with ---")
	}
	if !hasSuffix(frontmatter, "---\n") {
		t.Error("frontmatter should end with ---")
	}

	// Check it contains notion_id
	if !contains(frontmatter, "notion_id: abc123") {
		t.Error("frontmatter should contain notion_id")
	}

	// Check it's valid YAML
	content := frontmatter[4 : len(frontmatter)-4] // Remove --- delimiters
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}
}

func TestGenerateFrontmatter_Empty(t *testing.T) {
	entry := &EntryData{
		Title:      "Test",
		PageID:     "abc",
		Properties: map[string]any{},
	}

	frontmatter, err := GenerateFrontmatter(entry)
	if err != nil {
		t.Fatalf("GenerateFrontmatter() error = %v", err)
	}

	if frontmatter != "" {
		t.Errorf("expected empty frontmatter for empty properties, got %q", frontmatter)
	}
}

func TestGenerateFrontmatter_Nil(t *testing.T) {
	_, err := GenerateFrontmatter(nil)
	if err == nil {
		t.Error("expected error for nil entry")
	}
}

func TestBuildDatabaseEntry(t *testing.T) {
	entry := &EntryData{
		Title:  "My Task",
		PageID: "page-123",
		Properties: map[string]any{
			"status": "Done",
		},
	}

	dbEntry, err := BuildDatabaseEntry(entry, "# Content\n\nSome text here.")
	if err != nil {
		t.Fatalf("BuildDatabaseEntry() error = %v", err)
	}

	if dbEntry.Filename != "My Task.md" {
		t.Errorf("got filename %q, want %q", dbEntry.Filename, "My Task.md")
	}

	if !contains(dbEntry.Frontmatter, "status: Done") {
		t.Error("frontmatter should contain status")
	}

	if dbEntry.Content != "# Content\n\nSome text here." {
		t.Errorf("got content %q, want %q", dbEntry.Content, "# Content\n\nSome text here.")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Name", "Simple Name"},
		{"With/Slash", "With-Slash"},
		{"With\\Backslash", "With-Backslash"},
		{"With:Colon", "With-Colon"},
		{"With*Star", "WithStar"},
		{"With?Question", "WithQuestion"},
		{"With\"Quote", "WithQuote"},
		{"With<Less", "WithLess"},
		{"With>Greater", "WithGreater"},
		{"With|Pipe", "WithPipe"},
		{"With\nNewline", "With Newline"},
		{"  Spaces  ", "Spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizePropertyName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Status", "status"},
		{"Due Date", "due_date"},
		{"My Property", "my_property"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizePropertyName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePropertyName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Helper functions for tests
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
