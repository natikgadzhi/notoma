package notion

import (
	"testing"

	"github.com/jomei/notionapi"
)

func TestFilterWorkspaceRoots(t *testing.T) {
	// Test the filtering logic used by DiscoverWorkspaceRoots
	// We construct mock search results and verify filtering behavior

	tests := []struct {
		name     string
		objects  []notionapi.Object
		wantLen  int
		wantIDs  []string
		wantType []ResourceType
	}{
		{
			name:    "empty results",
			objects: []notionapi.Object{},
			wantLen: 0,
		},
		{
			name: "single workspace-level page",
			objects: []notionapi.Object{
				&notionapi.Page{
					ID: "page-123",
					Parent: notionapi.Parent{
						Type: notionapi.ParentTypeWorkspace,
					},
				},
			},
			wantLen:  1,
			wantIDs:  []string{"page-123"},
			wantType: []ResourceType{ResourceTypePage},
		},
		{
			name: "single workspace-level database",
			objects: []notionapi.Object{
				&notionapi.Database{
					ID: "db-456",
					Parent: notionapi.Parent{
						Type: notionapi.ParentTypeWorkspace,
					},
				},
			},
			wantLen:  1,
			wantIDs:  []string{"db-456"},
			wantType: []ResourceType{ResourceTypeDatabase},
		},
		{
			name: "mixed workspace and nested items",
			objects: []notionapi.Object{
				&notionapi.Page{
					ID: "page-root",
					Parent: notionapi.Parent{
						Type: notionapi.ParentTypeWorkspace,
					},
				},
				&notionapi.Page{
					ID: "page-nested",
					Parent: notionapi.Parent{
						Type:   notionapi.ParentTypePageID,
						PageID: "page-root",
					},
				},
				&notionapi.Database{
					ID: "db-root",
					Parent: notionapi.Parent{
						Type: notionapi.ParentTypeWorkspace,
					},
				},
				&notionapi.Database{
					ID: "db-nested",
					Parent: notionapi.Parent{
						Type:   notionapi.ParentTypePageID,
						PageID: "page-root",
					},
				},
			},
			wantLen:  2,
			wantIDs:  []string{"page-root", "db-root"},
			wantType: []ResourceType{ResourceTypePage, ResourceTypeDatabase},
		},
		{
			name: "database inside page is filtered out",
			objects: []notionapi.Object{
				&notionapi.Database{
					ID: "db-in-page",
					Parent: notionapi.Parent{
						Type:   notionapi.ParentTypePageID,
						PageID: "some-page",
					},
				},
			},
			wantLen: 0,
		},
		{
			name: "page inside database is filtered out",
			objects: []notionapi.Object{
				&notionapi.Page{
					ID: "page-in-db",
					Parent: notionapi.Parent{
						Type:       notionapi.ParentTypeDatabaseID,
						DatabaseID: "some-db",
					},
				},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same filtering logic as DiscoverWorkspaceRoots
			var roots []Resource
			for _, obj := range tt.objects {
				switch item := obj.(type) {
				case *notionapi.Page:
					if item.Parent.Type == notionapi.ParentTypeWorkspace {
						roots = append(roots, Resource{
							ID:    string(item.ID),
							Type:  ResourceTypePage,
							Title: extractPageTitle(item),
						})
					}
				case *notionapi.Database:
					if item.Parent.Type == notionapi.ParentTypeWorkspace {
						roots = append(roots, Resource{
							ID:    string(item.ID),
							Type:  ResourceTypeDatabase,
							Title: extractDatabaseTitle(item),
						})
					}
				}
			}

			if len(roots) != tt.wantLen {
				t.Errorf("got %d roots, want %d", len(roots), tt.wantLen)
				return
			}

			for i, id := range tt.wantIDs {
				if roots[i].ID != id {
					t.Errorf("root[%d].ID = %q, want %q", i, roots[i].ID, id)
				}
			}

			for i, typ := range tt.wantType {
				if roots[i].Type != typ {
					t.Errorf("root[%d].Type = %q, want %q", i, roots[i].Type, typ)
				}
			}
		})
	}
}

func TestExtractPageTitle(t *testing.T) {
	tests := []struct {
		name string
		page *notionapi.Page
		want string
	}{
		{
			name: "nil page",
			page: nil,
			want: "",
		},
		{
			name: "page with no properties",
			page: &notionapi.Page{},
			want: "",
		},
		{
			name: "page with title property",
			page: &notionapi.Page{
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "My Page Title"},
						},
					},
				},
			},
			want: "My Page Title",
		},
		{
			name: "page with multi-part title",
			page: &notionapi.Page{
				Properties: notionapi.Properties{
					"Title": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "Hello "},
							{PlainText: "World"},
						},
					},
				},
			},
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPageTitle(tt.page)
			if got != tt.want {
				t.Errorf("extractPageTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractDatabaseTitle(t *testing.T) {
	tests := []struct {
		name string
		db   *notionapi.Database
		want string
	}{
		{
			name: "nil database",
			db:   nil,
			want: "",
		},
		{
			name: "database with empty title",
			db:   &notionapi.Database{},
			want: "",
		},
		{
			name: "database with title",
			db: &notionapi.Database{
				Title: []notionapi.RichText{
					{PlainText: "My Database"},
				},
			},
			want: "My Database",
		},
		{
			name: "database with multi-part title",
			db: &notionapi.Database{
				Title: []notionapi.RichText{
					{PlainText: "Task "},
					{PlainText: "Tracker"},
				},
			},
			want: "Task Tracker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDatabaseTitle(tt.db)
			if got != tt.want {
				t.Errorf("extractDatabaseTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRichTextPlain(t *testing.T) {
	tests := []struct {
		name     string
		richText []notionapi.RichText
		want     string
	}{
		{
			name:     "nil slice",
			richText: nil,
			want:     "",
		},
		{
			name:     "empty slice",
			richText: []notionapi.RichText{},
			want:     "",
		},
		{
			name: "single text",
			richText: []notionapi.RichText{
				{PlainText: "Hello"},
			},
			want: "Hello",
		},
		{
			name: "multiple text parts",
			richText: []notionapi.RichText{
				{PlainText: "Hello"},
				{PlainText: " "},
				{PlainText: "World"},
			},
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRichTextPlain(tt.richText)
			if got != tt.want {
				t.Errorf("extractRichTextPlain() = %q, want %q", got, tt.want)
			}
		})
	}
}
