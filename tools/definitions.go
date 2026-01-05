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
- org_form: Filter by organization form code, e.g., "AS", "ENK", "NUF" (optional)
- municipality: Filter by municipality number (optional)
- registered_in_vat: Filter by VAT registration status (optional)
- bankrupt: Filter by bankruptcy status (optional)

RETURNS: List of companies with organization numbers, names, addresses, and basic info.

EXAMPLES:
- "Find companies named Equinor" → query: "Equinor"
- "Search for AS companies in Oslo" → query: "*", org_form: "AS", municipality: "0301"`,
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

RETURNS: Complete company information including:
- Organization number and name
- Organization form (AS, ENK, etc.)
- Addresses (postal and business)
- Registration status (VAT, business register, etc.)
- Industry codes (NACE)
- Employee count
- Share capital (for AS/ASA)
- Status flags (bankrupt, under liquidation)

EXAMPLES:
- "Get company 923609016" → org_number: "923609016"
- "Look up 923 609 016" → org_number: "923609016" (spaces removed)`,
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

RETURNS: Complete company information including:
- CVR number and name
- Company type (ApS, A/S, I/S, K/S, etc.)
- Address (street, postal code, city)
- Phone and email
- Industry code and description
- Employee count
- Registration date
- Production units (P-numbers) if any
- Owners if available

EXAMPLES:
- "Get Danish company 10150817" → cvr: "10150817"
- "Look up CVR DK-10150817" → cvr: "10150817" (DK prefix removed)`,
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
