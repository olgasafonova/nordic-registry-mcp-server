package wiki

// This file previously contained all Client methods (~4,200 lines).
// The code has been split into logical modules:
//
// - read.go:       Page reading operations (GetPage, GetPageInfo, ListPages, GetSections, etc.)
// - write.go:      Page editing operations (EditPage, FindReplace, UploadFile, etc.)
// - search.go:     Search operations (Search, SearchInPage, FindSimilarPages, etc.)
// - history.go:    Revision history operations (GetRevisions, CompareRevisions, etc.)
// - links.go:      Link operations (GetBacklinks, CheckLinks, FindBrokenInternalLinks, etc.)
// - categories.go: Category operations (ListCategories, GetCategoryMembers)
// - quality.go:    Content quality checks (CheckTerminology, CheckTranslations)
// - users.go:      User operations (ListUsers)
// - security.go:   SSRF protection (isPrivateIP, isPrivateHost, safeDialer)
//
// Helper functions remain in client.go.
