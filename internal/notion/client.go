package notion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jomei/notionapi"
)

// ResourceType indicates whether a Notion ID refers to a page or database.
type ResourceType string

const (
	ResourceTypePage     ResourceType = "page"
	ResourceTypeDatabase ResourceType = "database"
	ResourceTypeUnknown  ResourceType = "unknown"
)

// Resource represents a Notion page or database with its metadata.
type Resource struct {
	ID    string
	Type  ResourceType
	Title string
}

// Client wraps the Notion API client with rate limiting and convenience methods.
type Client struct {
	api     *notionapi.Client
	limiter *RateLimiter
	logger  *slog.Logger
}

// NewClient creates a new Notion client with rate limiting.
func NewClient(token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		api:     notionapi.NewClient(notionapi.Token(token)),
		limiter: DefaultRateLimiter(),
		logger:  logger,
	}
}

// DetectResourceType tries to determine if an ID refers to a page or database.
// It tries the page endpoint first, then falls back to database.
func (c *Client) DetectResourceType(ctx context.Context, id string) (*Resource, error) {
	// Try as page first
	page, err := c.GetPage(ctx, id)
	if err == nil {
		return &Resource{
			ID:    id,
			Type:  ResourceTypePage,
			Title: extractPageTitle(page),
		}, nil
	}

	// Check if it's a "not found" or "wrong type" error that means we should try database
	if !isNotFoundOrWrongTypeError(err) {
		return nil, fmt.Errorf("checking page %s: %w", id, err)
	}

	// Try as database
	db, err := c.GetDatabase(ctx, id)
	if err == nil {
		return &Resource{
			ID:    id,
			Type:  ResourceTypeDatabase,
			Title: extractDatabaseTitle(db),
		}, nil
	}

	if isNotFoundOrWrongTypeError(err) {
		return nil, fmt.Errorf("resource %s not found or not shared with integration", id)
	}

	return nil, fmt.Errorf("checking database %s: %w", id, err)
}

// GetPage retrieves a page by ID with rate limiting.
func (c *Client) GetPage(ctx context.Context, id string) (*notionapi.Page, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	c.logger.Debug("fetching page", "id", id)
	page, err := c.api.Page.Get(ctx, notionapi.PageID(id))
	if err != nil {
		return nil, c.handleError(err)
	}
	return page, nil
}

// GetDatabase retrieves a database by ID with rate limiting.
func (c *Client) GetDatabase(ctx context.Context, id string) (*notionapi.Database, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	c.logger.Debug("fetching database", "id", id)
	db, err := c.api.Database.Get(ctx, notionapi.DatabaseID(id))
	if err != nil {
		return nil, c.handleError(err)
	}
	return db, nil
}

// GetBlockChildren retrieves all child blocks of a block with pagination.
func (c *Client) GetBlockChildren(ctx context.Context, blockID string) ([]notionapi.Block, error) {
	var allBlocks []notionapi.Block
	var cursor notionapi.Cursor

	for {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		c.logger.Debug("fetching block children", "block_id", blockID, "cursor", cursor)
		pagination := &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    100,
		}

		resp, err := c.api.Block.GetChildren(ctx, notionapi.BlockID(blockID), pagination)
		if err != nil {
			return nil, c.handleError(err)
		}

		allBlocks = append(allBlocks, resp.Results...)

		if !resp.HasMore {
			break
		}
		cursor = notionapi.Cursor(resp.NextCursor)
	}

	return allBlocks, nil
}

// QueryDatabase retrieves all pages from a database with pagination.
func (c *Client) QueryDatabase(ctx context.Context, databaseID string) ([]notionapi.Page, error) {
	var allPages []notionapi.Page
	var cursor notionapi.Cursor

	for {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		c.logger.Debug("querying database", "database_id", databaseID, "cursor", cursor)
		req := &notionapi.DatabaseQueryRequest{
			StartCursor: cursor,
			PageSize:    100,
		}

		resp, err := c.api.Database.Query(ctx, notionapi.DatabaseID(databaseID), req)
		if err != nil {
			return nil, c.handleError(err)
		}

		allPages = append(allPages, resp.Results...)

		if !resp.HasMore {
			break
		}
		cursor = notionapi.Cursor(resp.NextCursor)
	}

	return allPages, nil
}

// GetCurrentUser retrieves the current user (bot) information.
// Useful for validating the token.
func (c *Client) GetCurrentUser(ctx context.Context) (*notionapi.User, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	c.logger.Debug("fetching current user")
	user, err := c.api.User.Me(ctx)
	if err != nil {
		return nil, c.handleError(err)
	}
	return user, nil
}

// handleError processes API errors and handles rate limiting.
func (c *Client) handleError(err error) error {
	var apiErr *notionapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Status == http.StatusTooManyRequests {
			// Handle rate limiting - use default 1 second if no Retry-After
			c.limiter.SetRetryAfter(ParseRetryAfter(""))
			c.logger.Warn("rate limited by Notion API")
		}
	}
	return err
}

// isNotFoundOrWrongTypeError checks if the error indicates a resource was not found
// or is the wrong type (e.g., trying to access a database as a page).
func isNotFoundOrWrongTypeError(err error) bool {
	var apiErr *notionapi.Error
	if errors.As(err, &apiErr) {
		// object_not_found: resource doesn't exist
		// validation_error: wrong type (e.g., "this is a database, not a page")
		return apiErr.Status == http.StatusNotFound ||
			apiErr.Code == "object_not_found" ||
			apiErr.Code == "validation_error"
	}
	return false
}

// extractPageTitle extracts the title from a page's properties.
func extractPageTitle(page *notionapi.Page) string {
	if page == nil || page.Properties == nil {
		return ""
	}

	// Look for title property
	for _, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok {
			return extractRichTextPlain(titleProp.Title)
		}
	}
	return ""
}

// extractDatabaseTitle extracts the title from a database.
func extractDatabaseTitle(db *notionapi.Database) string {
	if db == nil {
		return ""
	}
	return extractRichTextPlain(db.Title)
}

// extractRichTextPlain extracts plain text from rich text array.
func extractRichTextPlain(richText []notionapi.RichText) string {
	var result string
	for _, rt := range richText {
		result += rt.PlainText
	}
	return result
}
