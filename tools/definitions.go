package tools

// AllTools contains tool specifications for the Nordic Registry MCP server.
// Descriptions are concise for token efficiency. See docs/API.md for full documentation.
var AllTools = []ToolSpec{
	// ==========================================================================
	// NORWAY - Brønnøysundregistrene (data.brreg.no)
	// ==========================================================================
	{
		Name:        "norway_search_companies",
		Method:      "SearchCompanies",
		Title:       "Search Norwegian Companies",
		Category:    "search",
		Country:     "norway",
		Description: `Search Norwegian companies by name. USE WHEN: "find company named X", "search for companies in Oslo". Partial matches and case-insensitive. Returns a paginated list of matching companies with org number, name, status, and org form. If you have a 9-digit org number, use norway_get_company instead. Filters: org_form (AS/ENK/NUF), municipality (4-digit code, use norway_list_municipalities to look up codes), registered_in_vat, bankrupt, registered_in_voluntary.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},

	{
		Name:        "norway_get_company",
		Method:      "GetCompany",
		Title:       "Get Norwegian Company Details",
		Category:    "read",
		Country:     "norway",
		Description: `Get company details by 9-digit org number. USE WHEN: you have an org number and need company details. Spaces and dashes in org number are auto-stripped (e.g., "923 609 016" → "923609016"). Returns compact summary by default; set full=true for complete data including all addresses, industry codes, and capital info. FAILS WHEN: org number is not exactly 9 digits after stripping, or company not found.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_get_roles",
		Method:      "GetRoles",
		Title:       "Get Norwegian Company Board & Roles",
		Category:    "roles",
		Country:     "norway",
		Description: `Get board members, CEO, auditors, and other roles for a Norwegian company. USE WHEN: "who is on the board?", "list directors", "find CEO". Returns person names, birth dates, role types, and resignation status. For signature authority only, use norway_get_signature_rights.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_get_signature_rights",
		Method:      "GetSignatureRights",
		Title:       "Get Norwegian Company Signature Rights",
		Category:    "roles",
		Country:     "norway",
		Description: `Get who can sign for a company (signaturrett) and prokura holders. USE WHEN: "who can sign?", "signature rights", "prokura". Returns authorized signatories with signing rules (alone or jointly). For full board/role list, use norway_get_roles instead.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_batch_get_companies",
		Method:      "BatchGetCompanies",
		Title:       "Batch Get Norwegian Companies",
		Category:    "batch",
		Country:     "norway",
		Description: `Look up multiple companies at once (max 2000 org numbers). USE WHEN: you have a list of org numbers to validate or look up. More efficient than individual lookups. Returns company summaries and list of not_found entries. FAILS WHEN: any org number is not 9 digits (invalid entries are skipped, not failed).`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_get_subunits",
		Method:      "GetSubUnits",
		Title:       "Get Norwegian Company Branch Offices",
		Category:    "subunits",
		Country:     "norway",
		Description: `List all branches of a parent company by its org number. USE WHEN: "what branches does X have?", "list sub-units". For one specific branch by its own org number, use norway_get_subunit. To search branches by name, use norway_search_subunits.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_get_subunit",
		Method:      "GetSubUnit",
		Title:       "Get Norwegian Sub-Unit Details",
		Category:    "subunits",
		Country:     "norway",
		Description: `Get details for a specific sub-unit (branch office) by its org number. USE WHEN: you have a branch org number and want its details. Returns name, address, parent org number, and status. For listing all branches of a parent, use norway_get_subunits.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_get_updates",
		Method:      "GetUpdates",
		Title:       "Get Norwegian Registry Updates",
		Category:    "updates",
		Country:     "norway",
		Description: `Monitor main company registry changes since a timestamp. USE WHEN: "what companies changed recently?", "registry updates". Timestamp format: ISO 8601 datetime with timezone, e.g., "2024-01-08T00:00:00Z". Not cached. Returns list of org numbers with update timestamps. FAILS WHEN: timestamp is missing or malformed. For branch office changes only, use norway_get_subunit_updates.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_search_subunits",
		Method:      "SearchSubUnits",
		Title:       "Search Norwegian Branch Offices",
		Category:    "search",
		Country:     "norway",
		Description: `Search Norwegian branch offices by name. USE WHEN: "find branches named X" and you don't have the parent org number. Partial matches and case-insensitive. Filter by municipality (4-digit code, use norway_list_municipalities to look up codes). For listing all branches of a known parent, use norway_get_subunits.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_list_municipalities",
		Method:      "ListMunicipalities",
		Title:       "List Norwegian Municipalities",
		Category:    "reference",
		Country:     "norway",
		Description: `Get Norwegian municipality codes for filtering searches. Returns code-name pairs (e.g., 0301 = Oslo). Cached 24h. Use before norway_search_companies if you need a municipality code for filtering.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_list_org_forms",
		Method:      "ListOrgForms",
		Title:       "List Norwegian Organization Forms",
		Category:    "reference",
		Country:     "norway",
		Description: `Get organization form codes with descriptions (AS=limited, ENK=sole prop, NUF=foreign branch, etc.). Returns code-description pairs. Cached 24h. Use before norway_search_companies if you need an org_form code for filtering.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "norway_get_subunit_updates",
		Method:      "GetSubUnitUpdates",
		Title:       "Get Norwegian Sub-Unit Registry Updates",
		Category:    "updates",
		Country:     "norway",
		Description: `Get sub-unit (branch) registry changes since a timestamp. USE WHEN: "what branches changed recently?", monitoring branch office updates. Timestamp format: ISO 8601 datetime with timezone, e.g., "2024-01-08T00:00:00Z". Returns list of changed branch org numbers with update timestamps. Not cached. FAILS WHEN: timestamp is missing or malformed. For main company updates, use norway_get_updates.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},

	// ==========================================================================
	// DENMARK - CVR (cvrapi.dk)
	// ==========================================================================
	{
		Name:        "denmark_search_companies",
		Method:      "DKSearchCompanies",
		Title:       "Search Danish Companies",
		Category:    "search",
		Country:     "denmark",
		Description: `Search Danish companies by name. USE WHEN: "find Danish company X" and you don't have a CVR number. Partial matches and case-insensitive. WARNING: Returns only ONE result (CVR API limitation). Large companies often have multiple legal entities. Try variations: "[Company] Denmark", "[Company] A/S", "[Company] DK", "[Company] Holding", or pre-merger names. Returns company name, CVR number, address, status, and employee count. If you have an 8-digit CVR number, use denmark_get_company instead.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "denmark_get_company",
		Method:      "DKGetCompany",
		Title:       "Get Danish Company Details",
		Category:    "read",
		Country:     "denmark",
		Description: `Get company by 8-digit CVR number. USE WHEN: you have a CVR number; use denmark_search_companies to find one by name. DK prefix auto-removed. Returns summary by default; full=true for complete data with production units and owners. FAILS WHEN: CVR number is not exactly 8 digits.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "denmark_get_production_units",
		Method:      "DKGetProductionUnits",
		Title:       "Get Danish Production Units",
		Category:    "subunits",
		Country:     "denmark",
		Description: `Get production units (P-numbers) for a Danish company by CVR. USE WHEN: "list branches", "production units for CVR X". Returns P-number, name, address, and industry code per unit. Paginated: 20 results per page by default, max 100. Use page parameter for more results. FAILS WHEN: CVR number is not exactly 8 digits.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "denmark_search_by_phone",
		Method:      "DKSearchByPhone",
		Title:       "Search Danish Company by Phone",
		Category:    "search",
		Country:     "denmark",
		Description: `Find Danish company by phone number. USE WHEN: you only have a phone number, not a company name or CVR. Returns company name, CVR number, and address if found; empty result if no match. +45 prefix auto-removed. Not all companies have registered phones.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "denmark_get_by_pnumber",
		Method:      "DKGetByPNumber",
		Title:       "Get Danish Company by P-Number",
		Category:    "read",
		Country:     "denmark",
		Description: `Get parent company for a production unit P-number. USE WHEN: you have a P-number and need the owning company. P-number is 10 digits (e.g., 1012345678), obtainable from denmark_get_production_units. Returns the parent company's CVR number, name, address, and status. FAILS WHEN: P-number not found.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},

	// ==========================================================================
	// FINLAND - PRH (avoindata.prh.fi)
	// ==========================================================================
	{
		Name:        "finland_search_companies",
		Method:      "FISearchCompanies",
		Title:       "Search Finnish Companies",
		Category:    "search",
		Country:     "finland",
		Description: `Search Finnish companies by name. USE WHEN: "find Finnish company X" and you don't have a Y-tunnus. Partial matches and case-insensitive. Returns company name, business ID, form, and status. Paginated: 20 results per page by default, max 100. Common names return 900+ results. To narrow: use company_form=OY/OYJ for main companies, add location for city, or search exact name "Nokia Oyj" instead of "Nokia". FAILS WHEN: API is unreachable. If you have a Y-tunnus, use finland_get_company instead.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "finland_get_company",
		Method:      "FIGetCompany",
		Title:       "Get Finnish Company Details",
		Category:    "read",
		Country:     "finland",
		Description: `Get company by Y-tunnus (e.g., 0112038-9). USE WHEN: you have a Finnish business ID; use finland_search_companies to find one by name. FI prefix auto-removed. Returns summary by default; full=true for complete data with previous names and registry entries. FAILS WHEN: Y-tunnus format is invalid (must be 7 digits, hyphen, check digit).`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},

	// ==========================================================================
	// SWEDEN - Bolagsverket (requires OAuth2 credentials)
	// ==========================================================================
	{
		Name:        "sweden_get_company",
		Method:      "SEGetCompany",
		Title:       "Get Swedish Company Details",
		Category:    "read",
		Country:     "sweden",
		Description: `Get company by 10-digit org number (or 12-digit personal/coordination number). USE WHEN: you have a Swedish org number and need company details. Returns company name, organization form, legal form, business description, registration date, postal address, active status, deregistration info, ongoing proceedings, and industry codes. No name search available in this API; ask user for org number if not provided. FAILS WHEN: org number is not 10 or 12 digits, or company not found in Bolagsverket.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "sweden_get_document_list",
		Method:      "SEGetDocumentList",
		Title:       "List Swedish Annual Reports",
		Category:    "documents",
		Country:     "sweden",
		Description: `List available årsredovisningar (annual reports) for a Swedish company. USE WHEN: "what reports are available?", "list annual reports". Returns document IDs, financial year dates, and filing dates. Metadata only; use sweden_download_document with a document ID to get the actual report. FAILS WHEN: org number is not 10 digits, or company has no filed reports.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "sweden_check_status",
		Method:      "SECheckStatus",
		Title:       "Check Swedish API Status",
		Category:    "status",
		Country:     "sweden",
		Description: `Check if Bolagsverket API is available. USE WHEN: Sweden tools return errors, or you want to verify connectivity before a batch of Swedish lookups. Returns availability status, circuit breaker state ("closed", "open", or "half-open"), and number of cached entries.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
	{
		Name:        "sweden_download_document",
		Method:      "SEDownloadDocument",
		Title:       "Download Swedish Annual Report",
		Category:    "documents",
		Country:     "sweden",
		Description: `Download an annual report (årsredovisning) by document ID. USE WHEN: you have a document ID from sweden_get_document_list. Returns ZIP containing XBRL/iXBRL files, base64-encoded. WARNING: base64-encoded response is 1-10 MB. Do not use inside a context-sensitive agent pipeline; save to file instead. FAILS WHEN: document ID is not found or the file has been removed from Bolagsverket.`,
		ReadOnly:    true,
		Idempotent:  true,
		OpenWorld:   true,
	},
}

// ToolsByCountry returns tools filtered by country.
func ToolsByCountry(country string) []ToolSpec {
	var result []ToolSpec
	for _, tool := range AllTools {
		if tool.Country == country {
			result = append(result, tool)
		}
	}
	return result
}

// ToolsByCategory returns tools filtered by category.
func ToolsByCategory(category string) []ToolSpec {
	var result []ToolSpec
	for _, tool := range AllTools {
		if tool.Category == category {
			result = append(result, tool)
		}
	}
	return result
}
