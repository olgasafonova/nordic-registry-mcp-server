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
		Description: `Search Norwegian companies by name. If you have a 9-digit org number, use norway_get_company instead. Filters: org_form (AS/ENK/NUF), municipality, registered_in_vat, bankrupt, registered_in_voluntary.`,
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
		Description: `Get company details by 9-digit org number. Returns compact summary by default; set full=true for complete data including all addresses, industry codes, and capital info.`,
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
		Description: `Get board members, CEO, auditors, and other roles for a Norwegian company. Returns person names, birth dates, resignation status. For signature authority only, use norway_get_signature_rights.`,
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
		Description: `Get who can sign for a company (signaturrett) and prokura holders. For full board/role list, use norway_get_roles instead.`,
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
		Description: `Look up multiple companies at once (max 2000 org numbers). Returns company summaries and list of not_found entries. More efficient than individual lookups.`,
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
		Description: `Get all sub-units (branches) for a parent company. For a specific sub-unit's details, use norway_get_subunit. For searching branches by name, use norway_search_subunits.`,
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
		Description: `Get details for a specific sub-unit (branch office) by its org number. For listing all branches of a parent, use norway_get_subunits.`,
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
		Description: `Get registry changes since a timestamp (ISO 8601). Not cached - real-time feed. For sub-unit changes, use norway_get_subunit_updates.`,
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
		Description: `Search sub-units by name. Filter by municipality. For listing all branches of a known parent, use norway_get_subunits instead.`,
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
		Description: `Get Norwegian municipality codes for filtering searches. Cached 24h. Example: Oslo = 0301.`,
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
		Description: `Get organization form codes (AS=limited, ENK=sole prop, NUF=foreign branch, etc.) with descriptions. Cached 24h.`,
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
		Description: `Get sub-unit (branch) registry changes since a timestamp. Not cached. For main company updates, use norway_get_updates.`,
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
		Description: `Search Danish companies by name. WARNING: Returns only ONE result (CVR API limitation). Large companies often have multiple legal entities. Try variations: "[Company] Denmark", "[Company] A/S", "[Company] DK", "[Company] Holding", or pre-merger names. If you have an 8-digit CVR number, use denmark_get_company instead.`,
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
		Description: `Get company by 8-digit CVR number. DK prefix auto-removed. Returns summary by default; full=true for complete data with production units and owners.`,
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
		Description: `Get production units (P-numbers) for a Danish company by CVR. Paginated: 20 results per page by default (max 100). Use page parameter for more results.`,
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
		Description: `Find Danish company by phone number. +45 prefix auto-removed. Not all companies have registered phones.`,
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
		Description: `Get parent company for a production unit P-number. Returns the owning company's details.`,
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
		Description: `Search Finnish companies by name. Returns max 20 results per page. Common names return 900+ results. To narrow: use company_form=OY/OYJ for main companies, add location for city, or search exact name "Nokia Oyj" instead of "Nokia". If you have a Y-tunnus, use finland_get_company instead.`,
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
		Description: `Get company by Y-tunnus (e.g., 0112038-9). FI prefix auto-removed. Returns summary by default; full=true for complete data with previous names and registry entries.`,
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
		Description: `Get company by 10-digit org number. No name search available in this API - ask user for org number if not provided.`,
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
		Description: `List available årsredovisningar (annual reports) for a Swedish company. Returns metadata only (document IDs, dates); not the actual reports.`,
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
		Description: `Check if Bolagsverket API is available and OAuth2 credentials are valid. Returns circuit breaker status and cache stats.`,
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
		Description: `Download an annual report (årsredovisning) by document ID. Get IDs from sweden_get_document_list. Returns ZIP containing XBRL/iXBRL files, base64-encoded.`,
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
