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
			wantType: "datetime", // CreatedTime always includes timestamp
		},
		{
			name:     "last edited time property",
			propName: "Updated",
			prop:     &notionapi.LastEditedTimePropertyConfig{Type: notionapi.PropertyConfigLastEditedTime},
			wantType: "datetime", // LastEditedTime always includes timestamp
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

	// Check filters - should only have inFolder, not file.ext
	if base.Filters == nil {
		t.Fatal("filters is nil")
	}
	if len(base.Filters.And) != 1 {
		t.Errorf("got %d filters, want 1", len(base.Filters.And))
	}
	expectedFilter := `file.inFolder("Databases/Test Database")`
	if base.Filters.And[0] != expectedFilter {
		t.Errorf("got filter %q, want %q", base.Filters.And[0], expectedFilter)
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

	// Check column order - should have file.name + non-title properties
	if len(base.Views[0].Order) != 3 { // file.name, status, tags
		t.Errorf("got %d columns in order, want 3", len(base.Views[0].Order))
	}
	if base.Views[0].Order[0] != "file.name" {
		t.Errorf("first column should be file.name, got %q", base.Views[0].Order[0])
	}

	// Check sort configuration
	if len(base.Views[0].Sort) != 1 {
		t.Errorf("got %d sort entries, want 1", len(base.Views[0].Sort))
	}
	if base.Views[0].Sort[0].Property != "file.name" {
		t.Errorf("got sort property %q, want %q", base.Views[0].Sort[0].Property, "file.name")
	}
	if base.Views[0].Sort[0].Direction != "ASC" {
		t.Errorf("got sort direction %q, want %q", base.Views[0].Sort[0].Direction, "ASC")
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
			},
		},
		Display: map[string]string{
			"status": "Status",
		},
		Views: []View{
			{
				Type:  "table",
				Name:  "Table",
				Order: []string{"file.name", "status"},
				Sort: []ViewSort{
					{Property: "file.name", Direction: "ASC"},
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
	if len(parsed.Filters.And) != 1 {
		t.Errorf("got %d filters, want 1", len(parsed.Filters.And))
	}
	if len(parsed.Views) != 1 {
		t.Errorf("got %d views, want 1", len(parsed.Views))
	}
	if len(parsed.Views[0].Order) != 2 {
		t.Errorf("got %d columns in order, want 2", len(parsed.Views[0].Order))
	}
	if len(parsed.Views[0].Sort) != 1 {
		t.Errorf("got %d sort entries, want 1", len(parsed.Views[0].Sort))
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

func TestExtractEntryData_DateFields(t *testing.T) {
	schema := &DatabaseSchema{
		Title: "Test",
		Properties: map[string]PropertyMapping{
			"Name":             {Name: "Name", Type: "text", IsTitle: true},
			"Created time":     {Name: "Created time", Type: "date"},
			"Last edited time": {Name: "Last edited time", Type: "date"},
		},
		TitleProperty: "Name",
	}

	testTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	page := &notionapi.Page{
		ID: "test-page-id",
		Properties: notionapi.Properties{
			"Name": &notionapi.TitleProperty{
				Title: []notionapi.RichText{{PlainText: "Test Entry"}},
			},
			"Created time": &notionapi.CreatedTimeProperty{
				CreatedTime: testTime,
			},
			"Last edited time": &notionapi.LastEditedTimeProperty{
				LastEditedTime: testTime,
			},
		},
	}

	entry, err := ExtractEntryData(page, schema)
	if err != nil {
		t.Fatalf("ExtractEntryData() error = %v", err)
	}

	// Verify created_time is renamed to created_at
	if _, ok := entry.Properties["created_time"]; ok {
		t.Error("should not have created_time, should be renamed to created_at")
	}
	if _, ok := entry.Properties["created_at"]; !ok {
		t.Error("should have created_at property")
	}

	// Verify last_edited_time is renamed to updated_at
	if _, ok := entry.Properties["last_edited_time"]; ok {
		t.Error("should not have last_edited_time, should be renamed to updated_at")
	}
	if _, ok := entry.Properties["updated_at"]; !ok {
		t.Error("should have updated_at property")
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
	// Date-only (midnight) - should return YYYY-MM-DD format
	startDate := notionapi.Date(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	endDate := notionapi.Date(time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC))
	// DateTime (non-midnight) - should return full ISO format
	startDateTime := notionapi.Date(time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC))
	endDateTime := notionapi.Date(time.Date(2024, 1, 20, 18, 45, 0, 0, time.UTC))

	tests := []struct {
		name string
		prop *notionapi.DateProperty
		want string
	}{
		{
			name: "date only (midnight)",
			prop: &notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &startDate,
				},
			},
			want: "2024-01-15", // Date-only format for Obsidian date type
		},
		{
			name: "date range (midnight)",
			prop: &notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &startDate,
					End:   &endDate,
				},
			},
			want: "2024-01-15/2024-01-20", // Date-only format for both
		},
		{
			name: "datetime (non-midnight)",
			prop: &notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &startDateTime,
				},
			},
			want: "2024-01-15T14:30:00Z", // Full datetime for Obsidian datetime type
		},
		{
			name: "datetime range (non-midnight)",
			prop: &notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &startDateTime,
					End:   &endDateTime,
				},
			},
			want: "2024-01-15T14:30:00Z/2024-01-20T18:45:00Z", // Full datetime for both
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

	// Should still contain notion_id even if properties are empty
	if !contains(frontmatter, "notion_id: abc") {
		t.Errorf("frontmatter should contain notion_id, got %q", frontmatter)
	}
}

func TestGenerateFrontmatter_WithIcon(t *testing.T) {
	entry := &EntryData{
		Title:  "Test Entry",
		PageID: "abc123",
		Icon:   "🚀",
		Properties: map[string]any{
			"status": "Active",
		},
	}

	frontmatter, err := GenerateFrontmatter(entry)
	if err != nil {
		t.Fatalf("GenerateFrontmatter() error = %v", err)
	}

	// Check it contains icon (YAML may format it differently)
	if !contains(frontmatter, "icon:") {
		t.Errorf("frontmatter should contain icon, got: %q", frontmatter)
	}

	// Check it's valid YAML
	content := frontmatter[4 : len(frontmatter)-4] // Remove --- delimiters
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	// Verify icon is in parsed YAML
	if parsed["icon"] != "🚀" {
		t.Errorf("icon mismatch: got %v, want 🚀", parsed["icon"])
	}
}

func TestGenerateFrontmatter_NoIcon(t *testing.T) {
	entry := &EntryData{
		Title:  "Test Entry",
		PageID: "abc123",
		Icon:   "", // Empty icon
		Properties: map[string]any{
			"status": "Active",
		},
	}

	frontmatter, err := GenerateFrontmatter(entry)
	if err != nil {
		t.Fatalf("GenerateFrontmatter() error = %v", err)
	}

	// Should NOT contain icon field when icon is empty
	if contains(frontmatter, "icon:") {
		t.Error("frontmatter should not contain icon when empty")
	}
}

func TestExtractEntryData_WithIcon(t *testing.T) {
	rocketEmoji := notionapi.Emoji("🚀")

	schema := &DatabaseSchema{
		Title: "Test",
		Properties: map[string]PropertyMapping{
			"Name": {Name: "Name", Type: "text", IsTitle: true},
		},
		TitleProperty: "Name",
	}

	page := &notionapi.Page{
		ID: "test-page-id",
		Icon: &notionapi.Icon{
			Emoji: &rocketEmoji,
		},
		Properties: notionapi.Properties{
			"Name": &notionapi.TitleProperty{
				Title: []notionapi.RichText{{PlainText: "Test Entry"}},
			},
		},
	}

	entry, err := ExtractEntryData(page, schema)
	if err != nil {
		t.Fatalf("ExtractEntryData() error = %v", err)
	}

	if entry.Icon != "🚀" {
		t.Errorf("got icon %q, want %q", entry.Icon, "🚀")
	}
}

func TestExtractEntryData_NoIcon(t *testing.T) {
	schema := &DatabaseSchema{
		Title: "Test",
		Properties: map[string]PropertyMapping{
			"Name": {Name: "Name", Type: "text", IsTitle: true},
		},
		TitleProperty: "Name",
	}

	page := &notionapi.Page{
		ID: "test-page-id",
		// No icon set
		Properties: notionapi.Properties{
			"Name": &notionapi.TitleProperty{
				Title: []notionapi.RichText{{PlainText: "Test Entry"}},
			},
		},
	}

	entry, err := ExtractEntryData(page, schema)
	if err != nil {
		t.Fatalf("ExtractEntryData() error = %v", err)
	}

	if entry.Icon != "" {
		t.Errorf("got icon %q, want empty string", entry.Icon)
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
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
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

func TestMapPropertyName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"created_time", "created_at"},
		{"last_edited_time", "updated_at"},
		{"status", "status"},     // unchanged
		{"due_date", "due_date"}, // unchanged
		{"my_field", "my_field"}, // unchanged
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapPropertyName(tt.input)
			if got != tt.want {
				t.Errorf("mapPropertyName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBaseFileYAMLFormat(t *testing.T) {
	// Test that the YAML output matches the expected Obsidian Bases format
	base := &BaseFile{
		Filters: &FilterGroup{
			And: []string{
				`type == "book"`,
			},
		},
		Formulas: map[string]string{
			"cover": "link(file.embeds[0])",
		},
		Views: []View{
			{
				Type:  "table",
				Name:  "Table",
				Order: []string{"file.name", "author", "rating", "tags"},
				Sort: []ViewSort{
					{Property: "rating", Direction: "DESC"},
					{Property: "file.name", Direction: "ASC"},
				},
				ColumnSize: map[string]int{
					"author": 224,
					"rating": 92,
				},
			},
		},
	}

	data, err := MarshalBaseFile(base)
	if err != nil {
		t.Fatalf("MarshalBaseFile() error = %v", err)
	}

	yamlStr := string(data)

	// Verify key structural elements
	if !contains(yamlStr, "filters:") {
		t.Error("YAML should contain 'filters:'")
	}
	if !contains(yamlStr, "formulas:") {
		t.Error("YAML should contain 'formulas:'")
	}
	if !contains(yamlStr, "views:") {
		t.Error("YAML should contain 'views:'")
	}
	if !contains(yamlStr, "order:") {
		t.Error("YAML should contain 'order:'")
	}
	if !contains(yamlStr, "sort:") {
		t.Error("YAML should contain 'sort:'")
	}
	if !contains(yamlStr, "columnSize:") {
		t.Error("YAML should contain 'columnSize:'")
	}
	if !contains(yamlStr, "direction:") {
		t.Error("YAML should contain 'direction:'")
	}
	if !contains(yamlStr, "DESC") {
		t.Error("YAML should contain 'DESC'")
	}

	// Verify roundtrip
	var parsed BaseFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("roundtrip unmarshal failed: %v", err)
	}
	if parsed.Formulas["cover"] != base.Formulas["cover"] {
		t.Errorf("formula mismatch: got %q, want %q", parsed.Formulas["cover"], base.Formulas["cover"])
	}
	if parsed.Views[0].ColumnSize["author"] != 224 {
		t.Errorf("columnSize mismatch: got %d, want 224", parsed.Views[0].ColumnSize["author"])
	}
}

func TestBaseFileWithMultipleViews(t *testing.T) {
	base := &BaseFile{
		Filters: &FilterGroup{
			And: []string{
				`file.inFolder("Books")`,
			},
		},
		Views: []View{
			{
				Type:  "table",
				Name:  "Table",
				Order: []string{"file.name", "author"},
				Sort: []ViewSort{
					{Property: "file.name", Direction: "ASC"},
				},
			},
			{
				Type:  "cards",
				Name:  "Cards",
				Order: []string{"file.name", "author", "rating"},
				Sort: []ViewSort{
					{Property: "rating", Direction: "DESC"},
				},
			},
		},
	}

	data, err := MarshalBaseFile(base)
	if err != nil {
		t.Fatalf("MarshalBaseFile() error = %v", err)
	}

	var parsed BaseFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("roundtrip unmarshal failed: %v", err)
	}

	if len(parsed.Views) != 2 {
		t.Fatalf("got %d views, want 2", len(parsed.Views))
	}
	if parsed.Views[0].Type != "table" {
		t.Errorf("first view type: got %q, want %q", parsed.Views[0].Type, "table")
	}
	if parsed.Views[1].Type != "cards" {
		t.Errorf("second view type: got %q, want %q", parsed.Views[1].Type, "cards")
	}
}

func TestBaseFileWithViewFilters(t *testing.T) {
	base := &BaseFile{
		Filters: &FilterGroup{
			And: []string{
				`file.inFolder("Tasks")`,
			},
		},
		Views: []View{
			{
				Type:  "table",
				Name:  "Active Tasks",
				Order: []string{"file.name", "status"},
				Filters: &FilterGroup{
					And: []string{
						`status != "done"`,
					},
				},
			},
			{
				Type:  "table",
				Name:  "Completed",
				Order: []string{"file.name", "status"},
				Filters: &FilterGroup{
					And: []string{
						`status == "done"`,
					},
				},
			},
		},
	}

	data, err := MarshalBaseFile(base)
	if err != nil {
		t.Fatalf("MarshalBaseFile() error = %v", err)
	}

	var parsed BaseFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("roundtrip unmarshal failed: %v", err)
	}

	if parsed.Views[0].Filters == nil {
		t.Fatal("first view filters should not be nil")
	}
	if len(parsed.Views[0].Filters.And) != 1 {
		t.Errorf("first view should have 1 filter, got %d", len(parsed.Views[0].Filters.And))
	}
	if parsed.Views[1].Filters == nil {
		t.Fatal("second view filters should not be nil")
	}
}

func TestBaseFileWithOrFilters(t *testing.T) {
	base := &BaseFile{
		Filters: &FilterGroup{
			Or: []string{
				`file.inFolder("Books")`,
				`file.inFolder("Articles")`,
			},
		},
		Views: []View{
			{
				Type:  "table",
				Name:  "Reading List",
				Order: []string{"file.name"},
			},
		},
	}

	data, err := MarshalBaseFile(base)
	if err != nil {
		t.Fatalf("MarshalBaseFile() error = %v", err)
	}

	yamlStr := string(data)
	if !contains(yamlStr, "or:") {
		t.Error("YAML should contain 'or:'")
	}

	var parsed BaseFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("roundtrip unmarshal failed: %v", err)
	}

	if len(parsed.Filters.Or) != 2 {
		t.Errorf("got %d OR filters, want 2", len(parsed.Filters.Or))
	}
}

func TestBuildColumnOrder(t *testing.T) {
	schema := &DatabaseSchema{
		Title: "Test",
		Properties: map[string]PropertyMapping{
			"Name":     {Name: "Name", Type: "text", IsTitle: true},
			"Status":   {Name: "Status", Type: "text"},
			"Priority": {Name: "Priority", Type: "number"},
			"Due Date": {Name: "Due Date", Type: "date"},
		},
		TitleProperty: "Name",
	}

	order := buildColumnOrder(schema)

	// First should always be file.name
	if order[0] != "file.name" {
		t.Errorf("first column should be file.name, got %q", order[0])
	}

	// Should have 4 columns: file.name + 3 non-title properties
	if len(order) != 4 {
		t.Errorf("got %d columns, want 4", len(order))
	}

	// Title property should not be included (it becomes file.name)
	for _, col := range order {
		if col == "name" {
			t.Error("title property 'name' should not be in column order")
		}
	}

	// Properties should be sanitized
	found := false
	for _, col := range order {
		if col == "due_date" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'Due Date' should be sanitized to 'due_date' in column order")
	}
}

func TestGenerateBaseFile_EmptySchema(t *testing.T) {
	schema := &DatabaseSchema{
		Title:      "Empty DB",
		Properties: map[string]PropertyMapping{},
	}

	base, err := GenerateBaseFile(schema, "Databases/Empty")
	if err != nil {
		t.Fatalf("GenerateBaseFile() error = %v", err)
	}

	// Should still have file.name column
	if len(base.Views[0].Order) != 1 {
		t.Errorf("got %d columns, want 1 (file.name)", len(base.Views[0].Order))
	}
	if base.Views[0].Order[0] != "file.name" {
		t.Errorf("only column should be file.name, got %q", base.Views[0].Order[0])
	}
}

func TestGenerateBaseFile_SpecialCharactersInPath(t *testing.T) {
	schema := &DatabaseSchema{
		Title: "Test",
		Properties: map[string]PropertyMapping{
			"Name": {Name: "Name", Type: "text", IsTitle: true},
		},
		TitleProperty: "Name",
	}

	base, err := GenerateBaseFile(schema, "My Notes/Sub Folder/Database")
	if err != nil {
		t.Fatalf("GenerateBaseFile() error = %v", err)
	}

	expectedFilter := `file.inFolder("My Notes/Sub Folder/Database")`
	if base.Filters.And[0] != expectedFilter {
		t.Errorf("got filter %q, want %q", base.Filters.And[0], expectedFilter)
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
