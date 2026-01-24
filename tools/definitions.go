package tools

// AllTools contains all tool specifications for the Nordic Registry MCP server.
// Tools are organized by country and category for easier maintenance.
// Tool descriptions follow a structured format for optimal LLM tool selection:
// - USE WHEN: Natural language triggers
// - NOT FOR: Disambiguation from similar tools
// - PARAMETERS: Key arguments with defaults
// - RETURNS: What the tool returns
var AllTools = []ToolSpec{
	// ==========================================================================
	// NORWAY - Brønnøysundregistrene (data.brreg.no)
	// ==========================================================================

	// --------------------------------------------------------------------------
	// SEARCH TOOLS
	// --------------------------------------------------------------------------
	{
		Name:     "norway_search_companies",
		Method:   "SearchCompanies",
		Title:    "Search Norwegian Companies",
		Category: "search",
		Country:  "norway",
		Description: `Search for Norwegian companies by name in Brønnøysundregistrene.

USE WHEN: User asks "find Norwegian companies named X", "search for X in Norway", "look up Norwegian business X".

NOT FOR: Looking up a specific company by organization number (use norway_get_company instead).

PARAMETERS:
- query: Company name to search for (required)
- page: Page number for pagination (optional, default 0)
- size: Results per page (optional, default 20, max 100)
- org_form: Filter by organization form code, e.g., "AS", "ENK", "NUF", "FLI" (optional)
- municipality: Filter by municipality number (optional)
- registered_in_vat: Filter by VAT registration status (optional)
- bankrupt: Filter by bankruptcy status (optional)
- registered_in_voluntary: Filter for voluntary/non-profit orgs in Frivillighetsregisteret (optional)

RETURNS: List of companies with organization numbers, names, addresses, and basic info.

EXAMPLES:
- "Find companies named Equinor" → query: "Equinor"
- "Search for AS companies in Oslo" → query: "*", org_form: "AS", municipality: "0301"
- "Find voluntary organizations named Røde Kors" → query: "Røde Kors", registered_in_voluntary: true`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// COMPANY DETAILS
	// --------------------------------------------------------------------------
	{
		Name:     "norway_get_company",
		Method:   "GetCompany",
		Title:    "Get Norwegian Company Details",
		Category: "read",
		Country:  "norway",
		Description: `Get detailed information about a Norwegian company by organization number.

USE WHEN: User provides a 9-digit Norwegian org number, asks "get details for 923609016", "look up org nr 923 609 016".

NOT FOR: Searching by company name (use norway_search_companies instead).

PARAMETERS:
- org_number: 9-digit Norwegian organization number (required)
  Spaces and dashes are automatically removed.
- full: Return complete data instead of summary (optional, default false)

RETURNS: By default returns a compact summary with:
- Organization number, name, organization form
- Business and postal addresses (formatted)
- Employee count, industry, website
- Status flags (VAT registered, bankrupt, under liquidation)
- Voluntary organization info (if registered in Frivillighetsregisteret):
  - registered_in_voluntary: Whether registered
  - voluntary_registration_date: Date registered
  - activity: Activity description

With full=true, returns complete data including all addresses, industry codes, capital info, and HAL links.

EXAMPLES:
- "Get company 923609016" → org_number: "923609016"
- "Look up 923 609 016" → org_number: "923609016" (spaces removed)
- "Get full details for 923609016" → org_number: "923609016", full: true`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// ROLES (Board Members, etc.)
	// --------------------------------------------------------------------------
	{
		Name:     "norway_get_roles",
		Method:   "GetRoles",
		Title:    "Get Company Board & Roles",
		Category: "roles",
		Country:  "norway",
		Description: `Get board members, CEO, and other roles for a Norwegian company.

USE WHEN: User asks "who is on the board of X", "who runs company 923609016", "get directors for X", "find CEO of X".

NOT FOR: Getting general company info (use norway_get_company). Not for finding branch offices (use norway_get_subunits).

PARAMETERS:
- org_number: 9-digit Norwegian organization number (required)

RETURNS: Role groups containing:
- Role type (Board member, CEO, Auditor, etc.)
- Person name and birth date (or company for corporate roles)
- Whether the person has resigned
- Responsibility share (for certain roles)

EXAMPLES:
- "Who is on the board of 923609016?" → org_number: "923609016"
- "Find the CEO of Equinor" → First search for Equinor, then get roles with the org number`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},
	{
		Name:     "norway_get_signature_rights",
		Method:   "GetSignatureRights",
		Title:    "Get Company Signature Rights",
		Category: "roles",
		Country:  "norway",
		Description: `Get signature rights (signaturrett) and prokura for a Norwegian company.

USE WHEN: User asks "who can sign for company X", "signature rights for X", "who has prokura", "signing authority".

NOT FOR: Getting all board members (use norway_get_roles). Not for company details (use norway_get_company).

PARAMETERS:
- org_number: 9-digit Norwegian organization number (required)

RETURNS: Signature-related roles including:
- SignatureRights: People/entities with signaturrett (SIGN role code)
- Prokura: People/entities with prokura authority (PROK role code)
- Summary: Human-readable summary of who can sign

NOTE: Data is extracted from the roles API, filtering for SIGN and PROK role codes only.

EXAMPLES:
- "Who can sign for 923609016?" → org_number: "923609016"
- "Get signature rights for Equinor" → First search for Equinor, then use org number`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// BATCH OPERATIONS
	// --------------------------------------------------------------------------
	{
		Name:     "norway_batch_get_companies",
		Method:   "BatchGetCompanies",
		Title:    "Batch Get Companies",
		Category: "batch",
		Country:  "norway",
		Description: `Look up multiple Norwegian companies in a single request.

USE WHEN: User has a list of organization numbers to look up, needs to validate multiple companies,
or wants to enrich a dataset with company information.

NOT FOR: Searching by name (use norway_search_companies). Not for single lookups (use norway_get_company).

PARAMETERS:
- org_numbers: Array of 9-digit Norwegian organization numbers (required, max 2000)

RETURNS:
- companies: List of company summaries with name, status, address, org form
- total_results: Number of companies found
- not_found: List of org numbers that were not found in the registry

NOTE: Uses native BRREG batch API. Much more efficient than individual lookups.

EXAMPLES:
- "Look up these companies: 923609016, 914778271, 985399077"
- "Validate these org numbers: [list]"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// SUB-UNITS (Branch Offices)
	// --------------------------------------------------------------------------
	{
		Name:     "norway_get_subunits",
		Method:   "GetSubUnits",
		Title:    "Get Company Branch Offices",
		Category: "subunits",
		Country:  "norway",
		Description: `Get all sub-units (branch offices, production units) for a parent company.

USE WHEN: User asks "what branches does X have", "list locations for company X", "find sub-units of X".

NOT FOR: Getting parent company info (use norway_get_company). Not for looking up a specific sub-unit (use norway_get_subunit).

PARAMETERS:
- parent_org_number: 9-digit organization number of the parent company (required)

RETURNS: List of sub-units including:
- Sub-unit organization number
- Name
- Parent company reference
- Business address
- Industry codes
- Employee count

EXAMPLES:
- "What branches does 923609016 have?" → parent_org_number: "923609016"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},
	{
		Name:     "norway_get_subunit",
		Method:   "GetSubUnit",
		Title:    "Get Sub-Unit Details",
		Category: "subunits",
		Country:  "norway",
		Description: `Get detailed information about a specific sub-unit (branch office).

USE WHEN: User has a sub-unit organization number and wants details about that specific branch.

NOT FOR: Listing all sub-units of a parent (use norway_get_subunits). Not for parent company info (use norway_get_company).

PARAMETERS:
- org_number: 9-digit sub-unit organization number (required)

RETURNS: Sub-unit details including:
- Organization number and name
- Parent company reference
- Business address
- Industry codes
- Employee count

EXAMPLES:
- "Get details for sub-unit 912345678" → org_number: "912345678"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// UPDATES FEED
	// --------------------------------------------------------------------------
	{
		Name:     "norway_get_updates",
		Method:   "GetUpdates",
		Title:    "Get Registry Updates",
		Category: "updates",
		Country:  "norway",
		Description: `Get recent updates from the Norwegian business registry.

USE WHEN: User asks "what companies changed recently", "new registrations today", "recent updates in Norway registry".

NOT FOR: Getting specific company info (use norway_get_company).

PARAMETERS:
- since: ISO 8601 timestamp to get updates from (required, e.g., "2024-01-15T00:00:00Z")
- size: Maximum number of updates to return (optional, default 20)

RETURNS: List of updates including:
- Update ID
- Organization number
- Update timestamp
- Change type (new registration, update, deletion)

NOTE: This is a real-time feed and is NOT cached. Useful for monitoring changes.

EXAMPLES:
- "What changed since yesterday?" → since: "2024-01-14T00:00:00Z"
- "Get last 50 updates" → since: <recent timestamp>, size: 50`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// SUB-UNIT SEARCH
	// --------------------------------------------------------------------------
	{
		Name:     "norway_search_subunits",
		Method:   "SearchSubUnits",
		Title:    "Search Norwegian Branch Offices",
		Category: "search",
		Country:  "norway",
		Description: `Search for Norwegian sub-units (branch offices) by name.

USE WHEN: User asks "find branches named X", "search for branch offices", "look up production units by name".

NOT FOR: Listing all branches of a known parent company (use norway_get_subunits instead).

PARAMETERS:
- query: Sub-unit name to search for (required)
- page: Page number for pagination (optional, default 0)
- size: Results per page (optional, default 20, max 100)
- municipality: Filter by municipality number (optional)

RETURNS: List of sub-units with organization numbers, names, parent company, addresses, and employee counts.

EXAMPLES:
- "Find branches named Equinor" → query: "Equinor"
- "Search for sub-units in Oslo" → query: "*", municipality: "0301"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// REFERENCE DATA
	// --------------------------------------------------------------------------
	{
		Name:     "norway_list_municipalities",
		Method:   "ListMunicipalities",
		Title:    "List Norwegian Municipalities",
		Category: "reference",
		Country:  "norway",
		Description: `Get the list of Norwegian municipalities (kommuner) with their codes.

USE WHEN: User asks "what are the municipality codes", "list Norwegian municipalities", "what is the code for Oslo".

NOT FOR: Searching companies (use search tools with municipality filter).

PARAMETERS: None required.

RETURNS: List of municipalities with:
- Municipality number (4-digit code)
- Municipality name

NOTE: This data is cached for 24 hours as it rarely changes. Useful for filtering company searches.

EXAMPLES:
- "List all Norwegian municipalities" → no parameters needed
- "What is Oslo's municipality code?" → search result for "OSLO" with code "0301"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},
	{
		Name:     "norway_list_org_forms",
		Method:   "ListOrgForms",
		Title:    "List Norwegian Organization Forms",
		Category: "reference",
		Country:  "norway",
		Description: `Get the list of Norwegian organization forms (AS, ENK, NUF, etc.) with descriptions.

USE WHEN: User asks "what organization types exist", "what does AS mean", "list company types in Norway".

NOT FOR: Searching companies (use search tools with org_form filter).

PARAMETERS: None required.

RETURNS: List of organization forms with:
- Code (AS, ENK, NUF, etc.)
- Description in Norwegian

NOTE: This data is cached for 24 hours as it rarely changes. Common codes:
- AS: Aksjeselskap (Limited company)
- ASA: Allmennaksjeselskap (Public limited company)
- ENK: Enkeltpersonforetak (Sole proprietorship)
- NUF: Norsk avdeling av utenlandsk foretak (Norwegian branch of foreign company)

EXAMPLES:
- "List all organization forms" → no parameters needed
- "What does ENK mean?" → search for code "ENK" in results`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},
	{
		Name:     "norway_get_subunit_updates",
		Method:   "GetSubUnitUpdates",
		Title:    "Get Sub-Unit Registry Updates",
		Category: "updates",
		Country:  "norway",
		Description: `Get recent updates to sub-units (branch offices) from the Norwegian business registry.

USE WHEN: User asks "what branch offices changed recently", "new sub-unit registrations", "sub-unit updates since yesterday".

NOT FOR: Getting company updates (use norway_get_updates). Not for getting sub-unit details (use norway_get_subunit).

PARAMETERS:
- since: ISO 8601 timestamp to get updates from (required, e.g., "2024-01-15T00:00:00Z")
- size: Maximum number of updates to return (optional, default 20)

RETURNS: List of updates including:
- Update ID
- Organization number
- Update timestamp
- Change type (new registration, update, deletion)

NOTE: This is a real-time feed and is NOT cached. Useful for monitoring branch office changes.

EXAMPLES:
- "What sub-units changed since yesterday?" → since: "2024-01-14T00:00:00Z"
- "Get last 50 sub-unit updates" → since: <recent timestamp>, size: 50`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// ==========================================================================
	// DENMARK - CVR (Central Business Register / cvrapi.dk)
	// ==========================================================================

	// --------------------------------------------------------------------------
	// SEARCH TOOLS
	// --------------------------------------------------------------------------
	{
		Name:     "denmark_search_companies",
		Method:   "DKSearchCompanies",
		Title:    "Search Danish Companies",
		Category: "search",
		Country:  "denmark",
		Description: `Search for Danish companies by name in CVR (Central Business Register).

USE WHEN: User asks "find Danish companies named X", "search for X in Denmark", "look up Danish business X".

NOT FOR: Looking up a specific company by CVR number (use denmark_get_company instead).

PARAMETERS:
- query: Company name to search for (required)

RETURNS: Company information if found, including:
- CVR number (8-digit identifier)
- Company name
- Address, city, postal code
- Company type (ApS, A/S, I/S, etc.)
- Industry description
- Employee count
- Registration date
- Status (active, dissolved, bankrupt)

NOTE: The CVR API returns a single best match, not a list. For multiple results, try more specific queries.

EXAMPLES:
- "Find Danish company Novo Nordisk" → query: "Novo Nordisk"
- "Search for Maersk in Denmark" → query: "Maersk"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// COMPANY DETAILS
	// --------------------------------------------------------------------------
	{
		Name:     "denmark_get_company",
		Method:   "DKGetCompany",
		Title:    "Get Danish Company Details",
		Category: "read",
		Country:  "denmark",
		Description: `Get detailed information about a Danish company by CVR number.

USE WHEN: User provides an 8-digit Danish CVR number, asks "get details for CVR 12345678", "look up Danish company 12345678".

NOT FOR: Searching by company name (use denmark_search_companies instead).

PARAMETERS:
- cvr: 8-digit Danish CVR number (required)
  Spaces, dashes, and "DK" prefix are automatically removed.
- full: Return complete data instead of summary (optional, default false)

RETURNS: By default returns a compact summary with:
- CVR number, name, company type
- Address, city, postal code
- Industry, employee count
- Phone, email, status
- Production units count

With full=true, returns complete data including all production units, owners, and raw API fields.

EXAMPLES:
- "Get Danish company 10150817" → cvr: "10150817"
- "Look up CVR DK-10150817" → cvr: "10150817" (DK prefix removed)
- "Get full details for CVR 10150817" → cvr: "10150817", full: true`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// PRODUCTION UNITS (P-numbers)
	// --------------------------------------------------------------------------
	{
		Name:     "denmark_get_production_units",
		Method:   "DKGetProductionUnits",
		Title:    "Get Danish Production Units",
		Category: "subunits",
		Country:  "denmark",
		Description: `Get production units (P-numbers) for a Danish company.

USE WHEN: User asks "what locations does Danish company X have", "list production units for CVR Y", "find P-numbers for company".

NOT FOR: Getting parent company info (use denmark_get_company). Not for Norwegian sub-units (use norway_get_subunits).

PARAMETERS:
- cvr: 8-digit CVR number of the parent company (required)

RETURNS: List of production units including:
- P-number (production unit identifier)
- Name
- Address
- Whether it's the main production unit
- Employee count
- Industry description

EXAMPLES:
- "What production units does CVR 10150817 have?" → cvr: "10150817"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// PHONE SEARCH
	// --------------------------------------------------------------------------
	{
		Name:     "denmark_search_by_phone",
		Method:   "DKSearchByPhone",
		Title:    "Search Danish Company by Phone",
		Category: "search",
		Country:  "denmark",
		Description: `Search for a Danish company by phone number.

USE WHEN: User asks "find company with phone X", "who has phone number X", "look up Danish business by phone".

NOT FOR: Searching by company name (use denmark_search_companies) or CVR number (use denmark_get_company).

PARAMETERS:
- phone: Phone number to search for (required)
  Spaces, dashes, and +45 prefix are automatically removed.

RETURNS: Company information if found, including:
- CVR number
- Company name
- Address, city, postal code
- Company type, industry
- Contact info (phone, email)

NOTE: Not all companies have phone numbers registered. If a company is not found, it may have no registered phone.

EXAMPLES:
- "Find company with phone 33121212" → phone: "33121212"
- "Who has phone +45 33 12 12 12?" → phone: "33121212" (normalized)`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	{
		Name:     "denmark_get_by_pnumber",
		Method:   "DKGetByPNumber",
		Title:    "Get Danish Company by P-Number",
		Category: "read",
		Country:  "denmark",
		Description: `Get a Danish company by production unit P-number.

USE WHEN: User provides a P-number (production unit number), asks "find company with P-number X", "look up production unit X".

NOT FOR: Searching by company name (use denmark_search_companies) or CVR number (use denmark_get_company).

PARAMETERS:
- p_number: Production unit P-number (required)
  Spaces and dashes are automatically removed.

RETURNS: Parent company information including:
- CVR number
- Company name
- Address, city, postal code
- Company type, industry
- Contact info (phone, email)

NOTE: P-numbers identify specific production units/locations. This returns the parent company that owns the production unit.

EXAMPLES:
- "Find company with P-number 1234567890" → p_number: "1234567890"
- "Look up production unit 1234567890" → p_number: "1234567890"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// ==========================================================================
	// FINLAND - PRH (Patent and Registration Office / avoindata.prh.fi)
	// ==========================================================================

	// --------------------------------------------------------------------------
	// SEARCH TOOLS
	// --------------------------------------------------------------------------
	{
		Name:     "finland_search_companies",
		Method:   "FISearchCompanies",
		Title:    "Search Finnish Companies",
		Category: "search",
		Country:  "finland",
		Description: `Search for Finnish companies by name in PRH (Patent and Registration Office).

USE WHEN: User asks "find Finnish companies named X", "search for X in Finland", "look up Finnish business X".

NOT FOR: Looking up a specific company by business ID (use finland_get_company instead).

PARAMETERS:
- query: Company name to search for (required)
- location: Town or city to filter by (optional)
- company_form: Company form code like OY, OYJ, Ky (optional)
- page: Page number for pagination (optional)

RETURNS: List of companies with:
- Business ID (Y-tunnus, format: 1234567-8)
- Company name
- Company form (OY, OYJ, Ky, etc.)
- City and address
- Industry description
- Website
- Registration date
- Status

EXAMPLES:
- "Find Finnish company Nokia" → query: "Nokia"
- "Search for companies in Helsinki" → query: "*", location: "Helsinki"`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// COMPANY DETAILS
	// --------------------------------------------------------------------------
	{
		Name:     "finland_get_company",
		Method:   "FIGetCompany",
		Title:    "Get Finnish Company Details",
		Category: "read",
		Country:  "finland",
		Description: `Get detailed information about a Finnish company by business ID (Y-tunnus).

USE WHEN: User provides a Finnish business ID, asks "get details for 0112038-9", "look up Finnish company 0112038-9".

NOT FOR: Searching by company name (use finland_search_companies instead).

PARAMETERS:
- business_id: Finnish business ID / Y-tunnus (required, format: 1234567-8)
  FI prefix is automatically removed.
- full: Return complete data instead of summary (optional, default false)

RETURNS: By default returns a compact summary with:
- Business ID, name, company form
- Industry description
- City and street address
- Website
- Registration date and status

With full=true, returns complete data including EU ID, previous names, auxiliary names, postal address, company situations, and registry entries.

EXAMPLES:
- "Get Finnish company 0112038-9" → business_id: "0112038-9"
- "Look up Nokia's details" → First search, then use business_id: "0112038-9"
- "Get full details for 0112038-9" → business_id: "0112038-9", full: true`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// ==========================================================================
	// SWEDEN - Bolagsverket (High-Value Datasets API)
	// ==========================================================================
	// NOTE: Sweden tools require OAuth2 credentials. Set BOLAGSVERKET_CLIENT_ID
	// and BOLAGSVERKET_CLIENT_SECRET environment variables. Register for free
	// at https://bolagsverket.se/apierochoppnadata/vardefulladatamangder/kundanmalantillapiforvardefulladatamangder.5528.html

	// --------------------------------------------------------------------------
	// COMPANY DETAILS
	// --------------------------------------------------------------------------
	{
		Name:     "sweden_get_company",
		Method:   "SEGetCompany",
		Title:    "Get Swedish Company Details",
		Category: "read",
		Country:  "sweden",
		Description: `Get detailed information about a Swedish company by organization number.

USE WHEN: User provides a Swedish organization number, asks "get details for 5560125790", "look up Swedish company X".

NOT FOR: Searching by company name (Bolagsverket API doesn't support name search).

PARAMETERS:
- org_number: Swedish organization number (required)
  - Organization number: 10 digits (e.g., 5560125790)
  - Personal number: 12 digits (e.g., 194009272719)
  - Spaces and dashes are automatically removed.

RETURNS: Company summary including:
- Organization number, name, organization form (AB, E, HB, etc.)
- Legal form (juridisk form from Skatteverket)
- Business description
- Registration date and country
- Postal address
- Status flags (active, deregistered, bankruptcy/liquidation)
- Industry codes (SNI)
- Ad block status (reklamsparr)

NOTE: Unlike Norway/Denmark/Finland, the Swedish API requires OAuth2 credentials.
If credentials are not configured, the tool will return an error.

EXAMPLES:
- "Get Swedish company 5560125790" → org_number: "5560125790"
- "Look up 5560-1257-90" → org_number: "5560125790" (dashes removed)`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// ANNUAL REPORTS
	// --------------------------------------------------------------------------
	{
		Name:     "sweden_get_document_list",
		Method:   "SEGetDocumentList",
		Title:    "List Swedish Annual Reports",
		Category: "documents",
		Country:  "sweden",
		Description: `Get a list of available annual reports (årsredovisningar) for a Swedish company.

USE WHEN: User asks "what annual reports are available for company X", "list årsredovisningar for X", "find financial reports for Swedish company X".

NOT FOR: Getting company details (use sweden_get_company). Not for downloading reports (that requires additional processing).

PARAMETERS:
- org_number: Swedish organization number (required, 10 digits)

RETURNS: List of available annual reports including:
- Document ID (used for downloading)
- File format
- Reporting period end date (end of financial year)
- Registration timestamp (when filed with Bolagsverket)

NOTE: This returns metadata about available reports, not the reports themselves.
The actual reports are ZIP files containing XBRL or PDF data.

EXAMPLES:
- "What annual reports exist for 5560125790?" → org_number: "5560125790"
- "List financial reports for Swedish company X" → First get org number, then use this tool`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
	},

	// --------------------------------------------------------------------------
	// STATUS CHECK
	// --------------------------------------------------------------------------
	{
		Name:     "sweden_check_status",
		Method:   "SECheckStatus",
		Title:    "Check Swedish API Status",
		Category: "status",
		Country:  "sweden",
		Description: `Check the status of the Swedish Bolagsverket API and connection.

USE WHEN: User asks "is the Swedish API working", "check Sweden connection", "status of Bolagsverket".

NOT FOR: Getting company data. Use sweden_get_company for that.

PARAMETERS: None required.

RETURNS:
- available: Whether the API is responding
- circuit_breaker_status: Current circuit breaker state (closed/open/half-open)
- cache_entries: Number of cached responses

NOTE: This tool verifies that OAuth2 credentials are valid and the API is reachable.
Useful for debugging connection issues.

EXAMPLES:
- "Is the Swedish API working?" → no parameters needed
- "Check connection to Bolagsverket" → no parameters needed`,
		ReadOnly:   true,
		Idempotent: true,
		OpenWorld:  true,
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
