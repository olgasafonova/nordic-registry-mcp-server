package sweden

// GetCompanyArgs contains parameters for getting a Swedish company.
type GetCompanyArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"required" jsonschema_description:"Swedish organization number (10 digits) or personal number (12 digits)"`
}

// GetCompanyResult is the MCP response for getting a company.
type GetCompanyResult struct {
	Company *CompanySummary `json:"company,omitempty"`
}

// CompanySummary is a simplified company representation for MCP responses.
type CompanySummary struct {
	OrganizationNumber  string   `json:"organization_number"`
	Name                string   `json:"name"`
	OrganizationForm    string   `json:"organization_form,omitempty"`    // e.g., "AB - Aktiebolag"
	LegalForm           string   `json:"legal_form,omitempty"`           // e.g., "49 - Övriga aktiebolag"
	BusinessDescription string   `json:"business_description,omitempty"` // Free text description
	RegistrationDate    string   `json:"registration_date,omitempty"`
	RegistrationCountry string   `json:"registration_country,omitempty"`
	PostalAddress       string   `json:"postal_address,omitempty"`
	IsActive            bool     `json:"is_active"`
	DeregisteredDate    string   `json:"deregistered_date,omitempty"` // If company is deregistered
	DeregisteredReason  string   `json:"deregistered_reason,omitempty"`
	OngoingProceedings  []string `json:"ongoing_proceedings,omitempty"` // Bankruptcy, liquidation, etc.
	IndustryCodes       []string `json:"industry_codes,omitempty"`      // SNI codes
	AdBlockEnabled      bool     `json:"ad_block_enabled,omitempty"`    // "Reklamsparr" - opted out of marketing
}

// GetDocumentListArgs contains parameters for getting annual reports list.
type GetDocumentListArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"required" jsonschema_description:"Swedish organization number (10 digits)"`
}

// GetDocumentListResult is the MCP response for getting document list.
type GetDocumentListResult struct {
	OrganizationNumber string            `json:"organization_number"`
	Documents          []DocumentSummary `json:"documents"`
	Count              int               `json:"count"`
}

// DocumentSummary is a simplified annual report representation.
type DocumentSummary struct {
	DocumentID            string `json:"document_id"`
	FileFormat            string `json:"file_format,omitempty"`
	ReportingPeriodEnd    string `json:"reporting_period_end,omitempty"` // End of financial year
	RegistrationTimestamp string `json:"registration_timestamp,omitempty"`
}

// CheckStatusArgs contains parameters for checking API status.
type CheckStatusArgs struct {
	// No parameters needed
}

// CheckStatusResult is the MCP response for status check.
type CheckStatusResult struct {
	Available            bool   `json:"available"`
	CircuitBreakerStatus string `json:"circuit_breaker_status"`
	CacheEntries         int64  `json:"cache_entries"`
}

// DownloadDocumentArgs contains parameters for downloading an annual report.
type DownloadDocumentArgs struct {
	DocumentID string `json:"document_id" jsonschema:"required" jsonschema_description:"Document ID from sweden_get_document_list"`
}

// DownloadDocumentResult is the MCP response for downloading a document. The
// ZIP bytes are written to a local file (Path) rather than inlined as base64:
// a 1-10 MB annual report base64-encodes to far more than is safe to carry in
// the caller's context (HG-2 cost-lens).
type DownloadDocumentResult struct {
	DocumentID  string `json:"document_id"`
	FileFormat  string `json:"file_format"`
	SizeBytes   int    `json:"size_bytes"`
	Path        string `json:"path"` // local filesystem path to the downloaded ZIP
	Description string `json:"description"`
}

// LogAttrs implementations expose each tool's structured-log attributes so
// the handler layer can log requests and results without per-type dispatch.

// LogAttrs returns structured-log attributes for the company lookup.
func (a GetCompanyArgs) LogAttrs() []any { return []any{"org_number", a.OrgNumber} }

// LogAttrs returns structured-log attributes for the document listing.
func (a GetDocumentListArgs) LogAttrs() []any { return []any{"org_number", a.OrgNumber} }

// LogAttrs marks the type as recognized; there are no arguments to log.
func (CheckStatusArgs) LogAttrs() []any { return []any{} }

// LogAttrs returns structured-log attributes for the document download.
func (a DownloadDocumentArgs) LogAttrs() []any { return []any{"document_id", a.DocumentID} }

// LogAttrs returns structured-log attributes for the company lookup result.
func (r GetCompanyResult) LogAttrs() []any {
	if r.Company != nil {
		return []any{"found", true, "name", r.Company.Name}
	}
	return []any{"found", false}
}

// LogAttrs returns structured-log attributes for the document listing result.
func (r GetDocumentListResult) LogAttrs() []any { return []any{"documents", r.Count} }

// LogAttrs returns structured-log attributes for the availability check.
func (r CheckStatusResult) LogAttrs() []any { return []any{"available", r.Available} }

// LogAttrs returns structured-log attributes for the download result.
func (r DownloadDocumentResult) LogAttrs() []any {
	return []any{"document_id", r.DocumentID, "size_bytes", r.SizeBytes}
}
