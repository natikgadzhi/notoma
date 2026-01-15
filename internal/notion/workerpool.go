package notion

import (
	"context"
	"sync"

	"github.com/jomei/notionapi"
)

// OnStartFunc is called when a worker starts processing an item.
// The pageID parameter identifies which item is now being processed.
type OnStartFunc func(pageID string)

// WorkerPool manages a pool of workers for parallel API fetching.
// It uses a semaphore pattern to limit concurrency while sharing
// a rate-limited client across all workers.
type WorkerPool struct {
	client      *Client
	concurrency int
	semaphore   chan struct{}
	onStart     OnStartFunc
}

// SetOnStart sets a callback that fires when a worker begins processing an item.
// This is called after the semaphore is acquired, just before the API call.
func (p *WorkerPool) SetOnStart(fn OnStartFunc) {
	p.onStart = fn
}

// NewWorkerPool creates a worker pool with the specified concurrency limit.
// Recommended concurrency is 5-10 for Notion API (balancing speed vs rate limits).
func NewWorkerPool(client *Client, concurrency int) *WorkerPool {
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 20 {
		concurrency = 20 // Cap to prevent excessive parallelism
	}
	return &WorkerPool{
		client:      client,
		concurrency: concurrency,
		semaphore:   make(chan struct{}, concurrency),
	}
}

// DefaultWorkerPool creates a worker pool with default concurrency (5).
func DefaultWorkerPool(client *Client) *WorkerPool {
	return NewWorkerPool(client, 5)
}

// BlockFetchResult contains the result of fetching blocks for a page.
type BlockFetchResult struct {
	PageID string
	Blocks []notionapi.Block
	Err    error
}

// FetchBlocksParallel fetches blocks for multiple pages in parallel.
// Results are returned via the results channel in the order they complete.
// The results channel is closed when all fetches complete or context is canceled.
func (p *WorkerPool) FetchBlocksParallel(ctx context.Context, pageIDs []string) <-chan BlockFetchResult {
	results := make(chan BlockFetchResult, len(pageIDs))

	go func() {
		defer close(results)

		var wg sync.WaitGroup
		for _, pageID := range pageIDs {
			// Check context before starting new work
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Acquire semaphore slot
			select {
			case p.semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				defer func() { <-p.semaphore }() // Release semaphore slot

				// Notify that work is starting for this item
				if p.onStart != nil {
					p.onStart(id)
				}

				blocks, err := p.client.GetBlockChildren(ctx, id)
				select {
				case results <- BlockFetchResult{PageID: id, Blocks: blocks, Err: err}:
				case <-ctx.Done():
				}
			}(pageID)
		}

		wg.Wait()
	}()

	return results
}

// PageFetchResult contains the result of fetching a page's metadata.
type PageFetchResult struct {
	PageID string
	Page   *notionapi.Page
	Err    error
}

// FetchPagesParallel fetches page metadata for multiple pages in parallel.
// Results are returned via the results channel in the order they complete.
func (p *WorkerPool) FetchPagesParallel(ctx context.Context, pageIDs []string) <-chan PageFetchResult {
	results := make(chan PageFetchResult, len(pageIDs))

	go func() {
		defer close(results)

		var wg sync.WaitGroup
		for _, pageID := range pageIDs {
			// Check context before starting new work
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Acquire semaphore slot
			select {
			case p.semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				defer func() { <-p.semaphore }() // Release semaphore slot

				// Notify that work is starting for this item
				if p.onStart != nil {
					p.onStart(id)
				}

				page, err := p.client.GetPage(ctx, id)
				select {
				case results <- PageFetchResult{PageID: id, Page: page, Err: err}:
				case <-ctx.Done():
				}
			}(pageID)
		}

		wg.Wait()
	}()

	return results
}

// PageWithBlocksResult contains both page metadata and its blocks.
type PageWithBlocksResult struct {
	PageID string
	Page   *notionapi.Page
	Blocks []notionapi.Block
	Err    error
}

// FetchPagesWithBlocksParallel fetches both page metadata and blocks in parallel.
// This is more efficient than separate calls when both are needed.
func (p *WorkerPool) FetchPagesWithBlocksParallel(ctx context.Context, pageIDs []string) <-chan PageWithBlocksResult {
	results := make(chan PageWithBlocksResult, len(pageIDs))

	go func() {
		defer close(results)

		var wg sync.WaitGroup
		for _, pageID := range pageIDs {
			// Check context before starting new work
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Acquire semaphore slot
			select {
			case p.semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				defer func() { <-p.semaphore }() // Release semaphore slot

				// Notify that work is starting for this item
				if p.onStart != nil {
					p.onStart(id)
				}

				result := PageWithBlocksResult{PageID: id}

				// Fetch page metadata
				page, err := p.client.GetPage(ctx, id)
				if err != nil {
					result.Err = err
					select {
					case results <- result:
					case <-ctx.Done():
					}
					return
				}
				result.Page = page

				// Fetch blocks
				blocks, err := p.client.GetBlockChildren(ctx, id)
				if err != nil {
					result.Err = err
					select {
					case results <- result:
					case <-ctx.Done():
					}
					return
				}
				result.Blocks = blocks

				select {
				case results <- result:
				case <-ctx.Done():
				}
			}(pageID)
		}

		wg.Wait()
	}()

	return results
}
