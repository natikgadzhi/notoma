package zipimport

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// NotionExport represents a parsed Notion export.
type NotionExport struct {
	RootDir   string
	Pages     []*ExportedPage
	Databases []*ExportedDatabase
	// Attachments maps relative paths to absolute paths in the extracted directory
	Attachments map[string]string
}

// ExportedPage represents a page from the Notion export.
type ExportedPage struct {
	ID          string
	Title       string
	FilePath    string // Absolute path to the markdown file
	RelPath     string // Relative path from export root
	Content     string // Markdown content
	ParentPath  string // Parent folder path (for hierarchy)
	Attachments []string
}

// ExportedDatabase represents a database from the Notion export.
type ExportedDatabase struct {
	ID         string
	Title      string
	CSVPath    string // Absolute path to CSV file
	RelPath    string // Relative path from export root
	FolderPath string // Folder containing entries (if entries are separate files)
	Headers    []string
	Rows       []map[string]string
	Entries    []*ExportedPage // Database entries as pages
}

// notionIDRegex matches the Notion ID suffix pattern in filenames.
// Notion exports files like "Page Title abc123def456.md" where the ID is a 32-char hex string.
var notionIDRegex = regexp.MustCompile(`\s+([a-f0-9]{32})$`)

// shortIDRegex matches shorter ID patterns (some exports use shorter IDs).
var shortIDRegex = regexp.MustCompile(`\s+([a-f0-9]{8,32})$`)

// Parse parses an extracted Notion export directory.
func Parse(rootDir string) (*NotionExport, error) {
	export := &NotionExport{
		RootDir:     rootDir,
		Attachments: make(map[string]string),
	}

	// Walk the directory tree
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		switch {
		case info.IsDir():
			// Directories might contain database entries
			return nil

		case ext == ".md":
			page, err := parseMarkdownFile(path, relPath)
			if err != nil {
				return fmt.Errorf("parsing markdown %s: %w", relPath, err)
			}
			export.Pages = append(export.Pages, page)

		case ext == ".csv":
			db, err := parseCSVFile(path, relPath)
			if err != nil {
				return fmt.Errorf("parsing CSV %s: %w", relPath, err)
			}
			export.Databases = append(export.Databases, db)

		case isAttachment(ext):
			// Track attachment
			export.Attachments[relPath] = path
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking export directory: %w", err)
	}

	// Link pages to databases (database entries are often in subfolders)
	linkPagesToDatabases(export)

	return export, nil
}

// parseMarkdownFile reads and parses a markdown file from the export.
func parseMarkdownFile(absPath, relPath string) (*ExportedPage, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	// Extract title and ID from filename
	filename := filepath.Base(relPath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	title, id := extractTitleAndID(filename)

	// Get parent path for hierarchy
	parentPath := filepath.Dir(relPath)
	if parentPath == "." {
		parentPath = ""
	}

	return &ExportedPage{
		ID:         id,
		Title:      title,
		FilePath:   absPath,
		RelPath:    relPath,
		Content:    string(content),
		ParentPath: parentPath,
	}, nil
}

// parseCSVFile reads and parses a CSV file (database export).
func parseCSVFile(absPath, relPath string) (*ExportedDatabase, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// First row is headers
	headers := records[0]

	// Parse rows into maps
	var rows []map[string]string
	for _, record := range records[1:] {
		row := make(map[string]string)
		for i, value := range record {
			if i < len(headers) {
				row[headers[i]] = value
			}
		}
		rows = append(rows, row)
	}

	// Extract title and ID from filename
	filename := filepath.Base(relPath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	title, id := extractTitleAndID(filename)

	// Check for matching folder (database entries as separate files)
	folderPath := strings.TrimSuffix(absPath, ".csv")

	return &ExportedDatabase{
		ID:         id,
		Title:      title,
		CSVPath:    absPath,
		RelPath:    relPath,
		FolderPath: folderPath,
		Headers:    headers,
		Rows:       rows,
	}, nil
}

// extractTitleAndID extracts the title and Notion ID from a filename.
// Notion exports use format: "Title abc123def456" or "Title abc12345"
func extractTitleAndID(filename string) (title, id string) {
	// Try full 32-char ID first
	if matches := notionIDRegex.FindStringSubmatch(filename); len(matches) == 2 {
		id = matches[1]
		title = strings.TrimSpace(notionIDRegex.ReplaceAllString(filename, ""))
		return title, id
	}

	// Try shorter ID
	if matches := shortIDRegex.FindStringSubmatch(filename); len(matches) == 2 {
		id = matches[1]
		title = strings.TrimSpace(shortIDRegex.ReplaceAllString(filename, ""))
		return title, id
	}

	// No ID found, use filename as title
	return filename, ""
}

// isAttachment checks if a file extension indicates an attachment.
func isAttachment(ext string) bool {
	attachmentExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
		".svg": true, ".bmp": true, ".ico": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".txt": true,
		".mp3": true, ".wav": true, ".ogg": true, ".m4a": true,
		".mp4": true, ".mov": true, ".avi": true, ".webm": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true,
	}
	return attachmentExts[ext]
}

// linkPagesToDatabases associates pages with databases based on folder structure.
func linkPagesToDatabases(export *NotionExport) {
	for _, db := range export.Databases {
		// Look for a folder with the same name as the CSV (without extension)
		dbFolder := strings.TrimSuffix(db.RelPath, ".csv")

		for _, page := range export.Pages {
			// Check if page is in the database folder
			if strings.HasPrefix(page.RelPath, dbFolder+"/") ||
				strings.HasPrefix(page.RelPath, dbFolder+string(os.PathSeparator)) {
				db.Entries = append(db.Entries, page)
			}
		}
	}
}

// StripNotionID removes the Notion ID suffix from a string.
func StripNotionID(s string) string {
	s = notionIDRegex.ReplaceAllString(s, "")
	s = shortIDRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
