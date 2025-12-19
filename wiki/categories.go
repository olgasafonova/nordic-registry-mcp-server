package wiki

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ListCategories lists all categories in the wiki
func (c *Client) ListCategories(ctx context.Context, args ListCategoriesArgs) (ListCategoriesResult, error) {
	// Ensure logged in for wikis requiring auth for read
	if err := c.EnsureLoggedIn(ctx); err != nil {
		return ListCategoriesResult{}, err
	}

	limit := normalizeLimit(args.Limit, DefaultLimit, MaxLimit)

	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "allcategories")
	params.Set("aclimit", strconv.Itoa(limit))
	params.Set("acprop", "size")

	if args.Prefix != "" {
		params.Set("acprefix", args.Prefix)
	}

	if args.ContinueFrom != "" {
		params.Set("accontinue", args.ContinueFrom)
	}

	resp, err := c.apiRequest(ctx, params)
	if err != nil {
		return ListCategoriesResult{}, err
	}

	query := getMap(resp["query"])
	if query == nil {
		return ListCategoriesResult{}, fmt.Errorf("unexpected response format: missing query")
	}

	allcats := getSlice(query["allcategories"])
	categories := make([]CategoryInfo, 0, len(allcats))
	for _, cat := range allcats {
		catMap := getMap(cat)
		if catMap == nil {
			continue
		}
		categories = append(categories, CategoryInfo{
			Title:   getString(catMap["*"]),
			Members: getInt(catMap["size"]),
		})
	}

	result := ListCategoriesResult{
		Categories: categories,
	}

	// Check for continuation
	if cont := getMap(resp["continue"]); cont != nil {
		if accontinue := getString(cont["accontinue"]); accontinue != "" {
			result.HasMore = true
			result.ContinueFrom = accontinue
		}
	}

	return result, nil
}

// GetCategoryMembers gets pages in a category
func (c *Client) GetCategoryMembers(ctx context.Context, args CategoryMembersArgs) (CategoryMembersResult, error) {
	if args.Category == "" {
		return CategoryMembersResult{}, fmt.Errorf("category is required")
	}

	// Ensure logged in for wikis requiring auth for read
	if err := c.EnsureLoggedIn(ctx); err != nil {
		return CategoryMembersResult{}, err
	}

	category := normalizeCategoryName(args.Category)
	limit := normalizeLimit(args.Limit, DefaultLimit, MaxLimit)

	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "categorymembers")
	params.Set("cmtitle", category)
	params.Set("cmlimit", strconv.Itoa(limit))

	if args.Type != "" {
		params.Set("cmtype", args.Type)
	}

	if args.ContinueFrom != "" {
		params.Set("cmcontinue", args.ContinueFrom)
	}

	resp, err := c.apiRequest(ctx, params)
	if err != nil {
		return CategoryMembersResult{}, err
	}

	query := getMap(resp["query"])
	if query == nil {
		return CategoryMembersResult{}, fmt.Errorf("unexpected response format: missing query")
	}

	members := getSlice(query["categorymembers"])
	pages := make([]PageSummary, 0, len(members))
	for _, m := range members {
		member := getMap(m)
		if member == nil {
			continue
		}
		pages = append(pages, PageSummary{
			PageID: getInt(member["pageid"]),
			Title:  getString(member["title"]),
		})
	}

	result := CategoryMembersResult{
		Category: category,
		Members:  pages,
	}

	// Check for continuation
	if cont := getMap(resp["continue"]); cont != nil {
		if cmcontinue := getString(cont["cmcontinue"]); cmcontinue != "" {
			result.HasMore = true
			result.ContinueFrom = cmcontinue
		}
	}

	return result, nil
}
