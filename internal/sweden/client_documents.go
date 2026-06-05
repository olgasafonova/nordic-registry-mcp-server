package sweden

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GetDocumentList retrieves the list of annual reports for a company.
func (c *Client) GetDocumentList(ctx context.Context, orgNumber string) (*DokumentlistaSvar, error) {
	orgNumber = NormalizeOrgNumber(orgNumber)
	if orgNumber == "" {
		return nil, errors.New("sweden: organization number is required")
	}

	// Check cache
	cacheKey := "doclist:" + orgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		if result, ok := cached.(*DokumentlistaSvar); ok {
			return result, nil
		}
	}

	// Deduplicate concurrent requests
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (any, error) {
		reqBody := DokumentlistaBegaran{
			Identitetsbeteckning: orgNumber,
		}

		respBody, err := c.doRequest(ctx, http.MethodPost, "/dokumentlista", reqBody)
		if err != nil {
			return nil, err
		}

		var result DokumentlistaSvar
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("sweden: decoding document list response: %w", err)
		}

		// Cache the result
		c.cache.Set(cacheKey, &result, documentCacheTTL)

		return &result, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*DokumentlistaSvar), nil
}

// DownloadDocument downloads an annual report by document ID.
// Returns the raw ZIP file contents.
func (c *Client) DownloadDocument(ctx context.Context, documentID string) ([]byte, error) {
	if documentID == "" {
		return nil, errors.New("sweden: document ID is required")
	}

	if !c.circuitBreaker.Allow() {
		return nil, errors.New("sweden: circuit breaker open")
	}

	token, err := c.getToken(ctx)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, err
	}

	req, err := c.buildDocumentRequest(ctx, documentID, token)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req) // #nosec G704 -- URL constructed from hardcoded base + validated input
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.circuitBreaker.RecordFailure()
		body, _ := io.ReadAll(resp.Body)
		return nil, formatBolagsverketError(resp.StatusCode, body)
	}

	return c.readBoundedDocument(resp.Body)
}

// buildDocumentRequest constructs the GET /dokument/{id} request with the
// ZIP-accept header and Bearer auth.
func (c *Client) buildDocumentRequest(ctx context.Context, documentID, token string) (*http.Request, error) {
	reqURL := c.baseURL + "/dokument/" + url.PathEscape(documentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("sweden: creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/zip")
	return req, nil
}

// readBoundedDocument reads the response body up to maxDocumentSize, recording
// circuit-breaker state for read errors and oversized payloads.
func (c *Client) readBoundedDocument(body io.Reader) ([]byte, error) {
	limitedReader := io.LimitReader(body, maxDocumentSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: reading document: %w", err)
	}
	if len(data) > maxDocumentSize {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: document exceeds maximum size of %d bytes", maxDocumentSize)
	}

	c.circuitBreaker.RecordSuccess()
	return data, nil
}
