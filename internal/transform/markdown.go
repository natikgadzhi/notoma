package transform

import (
	"context"
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// BlockFetcher is an interface for fetching child blocks.
// This allows us to recursively fetch nested blocks during transformation.
type BlockFetcher interface {
	GetBlockChildren(ctx context.Context, blockID string) ([]notionapi.Block, error)
}

// Transformer converts Notion blocks to Obsidian-flavored markdown.
type Transformer struct {
	fetcher BlockFetcher
	ctx     context.Context
}

// NewTransformer creates a new block transformer.
func NewTransformer(ctx context.Context, fetcher BlockFetcher) *Transformer {
	return &Transformer{
		fetcher: fetcher,
		ctx:     ctx,
	}
}

// BlocksToMarkdown converts a slice of Notion blocks to markdown.
func (t *Transformer) BlocksToMarkdown(blocks []notionapi.Block) (string, error) {
	return t.blocksToMarkdownWithIndent(blocks, 0)
}

// blocksToMarkdownWithIndent converts blocks with indentation for nested content.
func (t *Transformer) blocksToMarkdownWithIndent(blocks []notionapi.Block, indent int) (string, error) {
	var sb strings.Builder
	var prevType notionapi.BlockType

	for _, block := range blocks {
		md, err := t.blockToMarkdown(block, indent)
		if err != nil {
			return "", fmt.Errorf("converting block %s: %w", block.GetID(), err)
		}

		// Add extra newline between different block types for readability
		if prevType != "" && prevType != block.GetType() {
			sb.WriteString("\n")
		}

		sb.WriteString(md)
		prevType = block.GetType()
	}

	return sb.String(), nil
}

// blockToMarkdown converts a single block to markdown.
func (t *Transformer) blockToMarkdown(block notionapi.Block, indent int) (string, error) {
	indentStr := strings.Repeat("\t", indent)

	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return t.paragraphToMarkdown(b, indentStr)

	case *notionapi.Heading1Block:
		return t.headingToMarkdown(b.Heading1.RichText, 1, b.Heading1.IsToggleable, indentStr)

	case *notionapi.Heading2Block:
		return t.headingToMarkdown(b.Heading2.RichText, 2, b.Heading2.IsToggleable, indentStr)

	case *notionapi.Heading3Block:
		return t.headingToMarkdown(b.Heading3.RichText, 3, b.Heading3.IsToggleable, indentStr)

	case *notionapi.BulletedListItemBlock:
		return t.bulletedListItemToMarkdown(b, indentStr)

	case *notionapi.NumberedListItemBlock:
		return t.numberedListItemToMarkdown(b, indentStr)

	case *notionapi.ToDoBlock:
		return t.todoToMarkdown(b, indentStr)

	case *notionapi.ToggleBlock:
		return t.toggleToMarkdown(b, indentStr)

	case *notionapi.CodeBlock:
		return t.codeToMarkdown(b, indentStr)

	case *notionapi.QuoteBlock:
		return t.quoteToMarkdown(b, indentStr)

	case *notionapi.CalloutBlock:
		return t.calloutToMarkdown(b, indentStr)

	case *notionapi.DividerBlock:
		return indentStr + "---\n\n", nil

	case *notionapi.TableBlock:
		return t.tableToMarkdown(b, indentStr)

	case *notionapi.ImageBlock:
		return t.imageToMarkdown(b, indentStr)

	case *notionapi.VideoBlock:
		return t.videoToMarkdown(b, indentStr)

	case *notionapi.FileBlock:
		return t.fileToMarkdown(b, indentStr)

	case *notionapi.PdfBlock:
		return t.pdfToMarkdown(b, indentStr)

	case *notionapi.BookmarkBlock:
		return t.bookmarkToMarkdown(b, indentStr)

	case *notionapi.EmbedBlock:
		return t.embedToMarkdown(b, indentStr)

	case *notionapi.EquationBlock:
		return t.equationToMarkdown(b, indentStr)

	case *notionapi.ChildPageBlock:
		return indentStr + "[[" + b.ChildPage.Title + "]]\n\n", nil

	case *notionapi.ChildDatabaseBlock:
		return indentStr + "[[" + b.ChildDatabase.Title + "]]\n\n", nil

	case *notionapi.LinkToPageBlock:
		return t.linkToPageToMarkdown(b, indentStr)

	case *notionapi.SyncedBlock:
		return t.syncedBlockToMarkdown(b, indentStr)

	case *notionapi.ColumnListBlock:
		return t.columnListToMarkdown(b, indentStr)

	case *notionapi.ColumnBlock:
		return t.columnToMarkdown(b, indentStr)

	case *notionapi.AudioBlock:
		return t.audioToMarkdown(b, indentStr)

	case *notionapi.LinkPreviewBlock:
		return indentStr + "[" + b.LinkPreview.URL + "](" + b.LinkPreview.URL + ")\n\n", nil

	case *notionapi.TemplateBlock:
		// Skip template blocks - they're Notion-specific
		return "", nil

	case *notionapi.BreadcrumbBlock:
		// Skip breadcrumbs - not meaningful in Obsidian
		return "", nil

	case *notionapi.TableOfContentsBlock:
		// Skip TOC - Obsidian generates automatically
		return "", nil

	case *notionapi.UnsupportedBlock:
		// Skip unsupported blocks with a comment
		return indentStr + "<!-- Unsupported Notion block -->\n", nil

	default:
		// Unknown block type
		return indentStr + fmt.Sprintf("<!-- Unknown block type: %T -->\n", block), nil
	}
}

// paragraphToMarkdown converts a paragraph block.
func (t *Transformer) paragraphToMarkdown(b *notionapi.ParagraphBlock, indent string) (string, error) {
	text := RichTextToMarkdown(b.Paragraph.RichText)

	var sb strings.Builder
	sb.WriteString(indent + text + "\n\n")

	// Handle children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		childMd, err := t.blocksToMarkdownWithIndent(children, 1)
		if err != nil {
			return "", err
		}
		sb.WriteString(childMd)
	}

	return sb.String(), nil
}

// headingToMarkdown converts heading blocks.
func (t *Transformer) headingToMarkdown(richText []notionapi.RichText, level int, isToggleable bool, indent string) (string, error) {
	text := RichTextToMarkdown(richText)
	prefix := strings.Repeat("#", level)

	if isToggleable {
		// Convert toggleable heading to foldable callout
		return fmt.Sprintf("%s> [!faq]- %s\n\n", indent, text), nil
	}

	return fmt.Sprintf("%s%s %s\n\n", indent, prefix, text), nil
}

// bulletedListItemToMarkdown converts bulleted list items.
func (t *Transformer) bulletedListItemToMarkdown(b *notionapi.BulletedListItemBlock, indent string) (string, error) {
	text := RichTextToMarkdown(b.BulletedListItem.RichText)

	var sb strings.Builder
	sb.WriteString(indent + "- " + text + "\n")

	// Handle children (nested items)
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		childMd, err := t.blocksToMarkdownWithIndent(children, len(indent)/4+1)
		if err != nil {
			return "", err
		}
		sb.WriteString(childMd)
	}

	return sb.String(), nil
}

// numberedListItemToMarkdown converts numbered list items.
func (t *Transformer) numberedListItemToMarkdown(b *notionapi.NumberedListItemBlock, indent string) (string, error) {
	text := RichTextToMarkdown(b.NumberedListItem.RichText)

	var sb strings.Builder
	// Use 1. for all items - markdown renderers handle numbering
	sb.WriteString(indent + "1. " + text + "\n")

	// Handle children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		childMd, err := t.blocksToMarkdownWithIndent(children, len(indent)/4+1)
		if err != nil {
			return "", err
		}
		sb.WriteString(childMd)
	}

	return sb.String(), nil
}

// todoToMarkdown converts todo/checkbox blocks.
func (t *Transformer) todoToMarkdown(b *notionapi.ToDoBlock, indent string) (string, error) {
	text := RichTextToMarkdown(b.ToDo.RichText)
	checkbox := "[ ]"
	if b.ToDo.Checked {
		checkbox = "[x]"
	}

	var sb strings.Builder
	sb.WriteString(indent + "- " + checkbox + " " + text + "\n")

	// Handle children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		childMd, err := t.blocksToMarkdownWithIndent(children, len(indent)/4+1)
		if err != nil {
			return "", err
		}
		sb.WriteString(childMd)
	}

	return sb.String(), nil
}

// toggleToMarkdown converts toggle blocks to foldable callouts.
func (t *Transformer) toggleToMarkdown(b *notionapi.ToggleBlock, indent string) (string, error) {
	title := RichTextToMarkdown(b.Toggle.RichText)

	var sb strings.Builder
	sb.WriteString(indent + "> [!faq]- " + title + "\n")

	// Handle children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		for _, child := range children {
			childMd, err := t.blockToMarkdown(child, 0)
			if err != nil {
				return "", err
			}
			// Prefix each line with > for callout
			lines := strings.Split(childMd, "\n")
			for _, line := range lines {
				if line != "" {
					sb.WriteString(indent + "> " + line + "\n")
				}
			}
		}
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

// codeToMarkdown converts code blocks.
func (t *Transformer) codeToMarkdown(b *notionapi.CodeBlock, indent string) (string, error) {
	code := RichTextToPlain(b.Code.RichText)
	lang := strings.ToLower(string(b.Code.Language))

	// Normalize language
	if lang == "plain text" || lang == "plain_text" {
		lang = ""
	}

	var sb strings.Builder
	sb.WriteString(indent + "```" + lang + "\n")
	sb.WriteString(code)
	if !strings.HasSuffix(code, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString(indent + "```\n\n")

	// Add caption if present
	if len(b.Code.Caption) > 0 {
		caption := RichTextToMarkdown(b.Code.Caption)
		sb.WriteString(indent + "*" + caption + "*\n\n")
	}

	return sb.String(), nil
}

// quoteToMarkdown converts quote blocks.
func (t *Transformer) quoteToMarkdown(b *notionapi.QuoteBlock, indent string) (string, error) {
	text := RichTextToMarkdown(b.Quote.RichText)

	var sb strings.Builder
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		sb.WriteString(indent + "> " + line + "\n")
	}

	// Handle children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		for _, child := range children {
			childMd, err := t.blockToMarkdown(child, 0)
			if err != nil {
				return "", err
			}
			// Prefix each line with > for quote
			childLines := strings.Split(childMd, "\n")
			for _, line := range childLines {
				if line != "" {
					sb.WriteString(indent + "> " + line + "\n")
				}
			}
		}
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

// calloutToMarkdown converts callout blocks to Obsidian callouts.
func (t *Transformer) calloutToMarkdown(b *notionapi.CalloutBlock, indent string) (string, error) {
	text := RichTextToMarkdown(b.Callout.RichText)
	calloutType := "note"

	// Map icon to callout type
	if b.Callout.Icon != nil && b.Callout.Icon.Emoji != nil {
		calloutType = emojiToCalloutType(string(*b.Callout.Icon.Emoji))
	}

	var sb strings.Builder
	sb.WriteString(indent + "> [!" + calloutType + "]\n")
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		sb.WriteString(indent + "> " + line + "\n")
	}

	// Handle children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		for _, child := range children {
			childMd, err := t.blockToMarkdown(child, 0)
			if err != nil {
				return "", err
			}
			// Prefix each line with > for callout
			childLines := strings.Split(childMd, "\n")
			for _, line := range childLines {
				if line != "" {
					sb.WriteString(indent + "> " + line + "\n")
				}
			}
		}
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

// emojiToCalloutType maps Notion callout emojis to Obsidian callout types.
func emojiToCalloutType(emoji string) string {
	mapping := map[string]string{
		"\U0001F4A8": "note",    // 💨
		"\U0001F4D8": "note",    // 📘
		"\U0001F4D7": "tip",     // 📗
		"\U0001F4D9": "example", // 📙
		"\U0001F4D5": "warning", // 📕
		"⚠️":         "warning",
		"\U0001F6A8": "danger", // 🚨
		"ℹ️":         "info",
		"✅":          "success",
		"❌":          "failure",
		"\U0001F4A1": "tip", // 💡
		"❗":          "important",
		"❓":          "question",
		"⚙️":         "abstract",
		"\U0001F3AF": "important", // 🎯
		"\U0001F525": "danger",    // 🔥
		"⭐":          "tip",
		"\U0001F516": "quote", // 🔖
	}

	if calloutType, ok := mapping[emoji]; ok {
		return calloutType
	}
	return "note"
}

// tableToMarkdown converts table blocks.
func (t *Transformer) tableToMarkdown(b *notionapi.TableBlock, indent string) (string, error) {
	if !b.HasChildren {
		return "", nil
	}

	children, err := t.fetchChildren(string(b.ID))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, child := range children {
		row, ok := child.(*notionapi.TableRowBlock)
		if !ok {
			continue
		}

		sb.WriteString(indent + "|")
		for _, cell := range row.TableRow.Cells {
			cellText := RichTextToMarkdown(cell)
			sb.WriteString(" " + cellText + " |")
		}
		sb.WriteString("\n")

		// Add header separator after first row if table has column headers
		if i == 0 && b.Table.HasColumnHeader {
			sb.WriteString(indent + "|")
			for range row.TableRow.Cells {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	return sb.String(), nil
}

// imageToMarkdown converts image blocks.
func (t *Transformer) imageToMarkdown(b *notionapi.ImageBlock, indent string) (string, error) {
	url := getMediaURL(b.Image.File, b.Image.External)
	caption := ""
	if len(b.Image.Caption) > 0 {
		caption = RichTextToPlain(b.Image.Caption)
	}

	// TODO: During actual sync, download image and update URL to local path
	return indent + "![" + caption + "](" + url + ")\n\n", nil
}

// videoToMarkdown converts video blocks.
func (t *Transformer) videoToMarkdown(b *notionapi.VideoBlock, indent string) (string, error) {
	url := getMediaURL(b.Video.File, b.Video.External)
	caption := ""
	if len(b.Video.Caption) > 0 {
		caption = RichTextToPlain(b.Video.Caption)
	}

	return indent + "![" + caption + "](" + url + ")\n\n", nil
}

// fileToMarkdown converts file blocks.
func (t *Transformer) fileToMarkdown(b *notionapi.FileBlock, indent string) (string, error) {
	url := getMediaURL(b.File.File, b.File.External)
	caption := "file"
	if len(b.File.Caption) > 0 {
		caption = RichTextToPlain(b.File.Caption)
	}

	// TODO: During actual sync, download file and update URL to local path
	return indent + "[" + caption + "](" + url + ")\n\n", nil
}

// pdfToMarkdown converts PDF blocks.
func (t *Transformer) pdfToMarkdown(b *notionapi.PdfBlock, indent string) (string, error) {
	url := getMediaURL(b.Pdf.File, b.Pdf.External)
	caption := "PDF"
	if len(b.Pdf.Caption) > 0 {
		caption = RichTextToPlain(b.Pdf.Caption)
	}

	// TODO: During actual sync, download PDF and update URL to local path
	return indent + "![" + caption + "](" + url + ")\n\n", nil
}

// audioToMarkdown converts audio blocks.
func (t *Transformer) audioToMarkdown(b *notionapi.AudioBlock, indent string) (string, error) {
	url := getMediaURL(b.Audio.File, b.Audio.External)
	caption := "audio"
	if len(b.Audio.Caption) > 0 {
		caption = RichTextToPlain(b.Audio.Caption)
	}

	return indent + "![" + caption + "](" + url + ")\n\n", nil
}

// bookmarkToMarkdown converts bookmark blocks.
func (t *Transformer) bookmarkToMarkdown(b *notionapi.BookmarkBlock, indent string) (string, error) {
	url := b.Bookmark.URL
	title := url
	if len(b.Bookmark.Caption) > 0 {
		title = RichTextToPlain(b.Bookmark.Caption)
	}

	return indent + "[" + title + "](" + url + ")\n\n", nil
}

// embedToMarkdown converts embed blocks.
func (t *Transformer) embedToMarkdown(b *notionapi.EmbedBlock, indent string) (string, error) {
	url := b.Embed.URL
	caption := url
	if len(b.Embed.Caption) > 0 {
		caption = RichTextToPlain(b.Embed.Caption)
	}

	// Check for known embed types and handle specially
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return indent + "![" + caption + "](" + url + ")\n\n", nil
	}

	return indent + "[" + caption + "](" + url + ")\n\n", nil
}

// equationToMarkdown converts equation blocks (display math).
func (t *Transformer) equationToMarkdown(b *notionapi.EquationBlock, indent string) (string, error) {
	return indent + "$$\n" + b.Equation.Expression + "\n$$\n\n", nil
}

// linkToPageToMarkdown converts link_to_page blocks.
func (t *Transformer) linkToPageToMarkdown(b *notionapi.LinkToPageBlock, indent string) (string, error) {
	switch b.LinkToPage.Type {
	case "page_id":
		// TODO: Resolve page title during sync
		return indent + "[[" + string(b.LinkToPage.PageID) + "]]\n\n", nil
	case "database_id":
		// TODO: Resolve database title during sync
		return indent + "[[" + string(b.LinkToPage.DatabaseID) + "]]\n\n", nil
	}
	return "", nil
}

// syncedBlockToMarkdown converts synced blocks.
func (t *Transformer) syncedBlockToMarkdown(b *notionapi.SyncedBlock, indent string) (string, error) {
	// If synced_from is set, this is a reference - fetch the original block's children
	if b.SyncedBlock.SyncedFrom != nil && b.SyncedBlock.SyncedFrom.BlockID != "" {
		children, err := t.fetchChildren(string(b.SyncedBlock.SyncedFrom.BlockID))
		if err != nil {
			return "", err
		}
		return t.blocksToMarkdownWithIndent(children, 0)
	}

	// This is the original synced block - convert children
	if b.HasChildren {
		children, err := t.fetchChildren(string(b.ID))
		if err != nil {
			return "", err
		}
		return t.blocksToMarkdownWithIndent(children, 0)
	}

	return "", nil
}

// columnListToMarkdown flattens column lists (Obsidian doesn't support columns).
func (t *Transformer) columnListToMarkdown(b *notionapi.ColumnListBlock, indent string) (string, error) {
	if !b.HasChildren {
		return "", nil
	}

	children, err := t.fetchChildren(string(b.ID))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, child := range children {
		md, err := t.blockToMarkdown(child, 0)
		if err != nil {
			return "", err
		}
		sb.WriteString(md)
	}

	return sb.String(), nil
}

// columnToMarkdown flattens column content.
func (t *Transformer) columnToMarkdown(b *notionapi.ColumnBlock, indent string) (string, error) {
	if !b.HasChildren {
		return "", nil
	}

	children, err := t.fetchChildren(string(b.ID))
	if err != nil {
		return "", err
	}

	return t.blocksToMarkdownWithIndent(children, 0)
}

// fetchChildren fetches child blocks if a fetcher is available.
func (t *Transformer) fetchChildren(blockID string) ([]notionapi.Block, error) {
	if t.fetcher == nil {
		return nil, nil
	}
	return t.fetcher.GetBlockChildren(t.ctx, blockID)
}

// getMediaURL extracts URL from Notion file objects (internal or external).
func getMediaURL(file, external *notionapi.FileObject) string {
	if file != nil {
		return file.URL
	}
	if external != nil {
		return external.URL
	}
	return ""
}

// BlocksToMarkdownSimple converts blocks without fetching children.
// Useful for testing or when children are already fetched.
func BlocksToMarkdownSimple(blocks []notionapi.Block) (string, error) {
	t := &Transformer{}
	return t.BlocksToMarkdown(blocks)
}
