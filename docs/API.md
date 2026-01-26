# API Reference

Complete reference for all MCP tools provided by the Nordic Registry MCP Server.

---

## Norway (Brønnøysundregistrene)

### norway_search_companies

Search for Norwegian companies by name.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Company name to search for |
| `page` | int | No | Page number (0-indexed, default: 0) |
| `size` | int | No | Results per page (max 100, default: 20) |
| `org_form` | string | No | Filter by org form (AS, ENK, NUF, etc.) |
| `municipality` | string | No | Filter by municipality code |
| `registered_in_vat` | bool | No | Filter by VAT registration status |
| `bankrupt` | bool | No | Filter by bankruptcy status |
| `registered_in_voluntary` | bool | No | Filter for voluntary/non-profit organizations (Frivillighetsregisteret) |

**Returns:**

```json
{
  "companies": [
    {
      "organization_number": "923609016",
      "name": "EQUINOR ASA",
      "organization_form": "ASA",
      "postal_address": "Postboks 8500, 4035 STAVANGER",
      "business_address": "Forusbeen 50, 4035 STAVANGER",
      "status": "ACTIVE"
    }
  ],
  "total_results": 1,
  "page": 0,
  "total_pages": 1
}
```

**Example prompts:**
- "Find Norwegian companies named Equinor"
- "Search for AS companies in Oslo"
- "Find bankrupt companies named Restaurant"
- "Find voluntary organizations named Røde Kors"
- "Search for non-profit organizations in Frivillighetsregisteret"

---

### norway_get_company

Get detailed information about a Norwegian company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_number` | string | Yes | 9-digit organization number |
| `full` | bool | No | Return full details (default: false returns summary) |

**Org Number Formats (all accepted):**
- `923609016` (digits only)
- `923 609 016` (with spaces)
- `923-609-016` (with dashes)

**Returns (summary):**

```json
{
  "summary": {
    "organization_number": "923609016",
    "name": "EQUINOR ASA",
    "organization_form": "ASA",
    "business_address": "Forusbeen 50, 4035 STAVANGER",
    "postal_address": "Postboks 8500, 4035 STAVANGER",
    "registration_date": "2007-10-01",
    "employee_count": 21000,
    "industry": "Utvinning av råolje",
    "website": "https://www.equinor.com",
    "vat_registered": true,
    "status": "ACTIVE",
    "registered_in_voluntary": false,
    "voluntary_registration_date": null,
    "activity": null
  }
}
```

For voluntary organizations:

```json
{
  "summary": {
    "organization_number": "916627939",
    "name": "- P A L M E R A -",
    "organization_form": "FLI - Forening/lag/innretning",
    "status": "ACTIVE",
    "registered_in_voluntary": true,
    "voluntary_registration_date": "2022-06-18",
    "activity": [
      "Er et non-profit kunstnerstyrt visningsrom og kunstgalleri i",
      "Bergen sentrum. Stiller ut samtidskunst fra lokale og",
      "internasjonale kunstnere."
    ]
  }
}
```

**Example prompts:**
- "Get details for org number 923609016"
- "Look up company 914778271"

---

### norway_get_roles

Get board members, CEO, and other roles for a company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_number` | string | Yes | 9-digit organization number |

**Returns:**

```json
{
  "role_groups": [
    {
      "type": "STYR",
      "description": "Styre",
      "last_modified": "2024-01-15",
      "roles": [
        {
          "type": "LEDE",
          "description": "Styrets leder",
          "name": "Ola Nordmann",
          "birth_date": "1965-03-15",
          "resigned": false
        },
        {
          "type": "MEDL",
          "description": "Styremedlem",
          "name": "Kari Nordmann",
          "birth_date": "1970-07-22",
          "resigned": false
        }
      ]
    },
    {
      "type": "DAGL",
      "description": "Daglig leder",
      "roles": [
        {
          "type": "DAGL",
          "description": "Daglig leder",
          "name": "Per Hansen"
        }
      ]
    }
  ]
}
```

**Role Types:**

| Code | Norwegian | English |
|------|-----------|---------|
| STYR | Styre | Board |
| LEDE | Styrets leder | Board chair |
| MEDL | Styremedlem | Board member |
| VARA | Varamedlem | Deputy member |
| DAGL | Daglig leder | CEO |
| NEST | Nestleder | Deputy chair |
| REVI | Revisor | Auditor |

**Example prompts:**
- "Who is on the board of 923609016?"
- "Find the CEO of Equinor"
- "List all directors for org 914778271"

---

### norway_get_signature_rights

Get signature rights and prokura for a company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_number` | string | Yes | 9-digit organization number |

**Returns:**

```json
{
  "organization_number": "923609016",
  "company_name": "EQUINOR ASA",
  "signature_rights": [
    {
      "type": "SIGN",
      "description": "Signatur",
      "name": "Ola Nordmann",
      "birth_date": "1965-03-15"
    }
  ],
  "prokura": [
    {
      "type": "PROK",
      "description": "Prokura",
      "name": "Kari Hansen",
      "birth_date": "1972-04-10"
    }
  ],
  "summary": "2 persons with signature rights, 1 person with prokura"
}
```

**Example prompts:**
- "Who can sign for company 923609016?"
- "Get signature rights for Equinor"
- "Who has prokura for 914778271?"

---

### norway_batch_get_companies

Look up multiple companies in a single request.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_numbers` | string[] | Yes | List of 9-digit org numbers (max 2000) |

**Returns:**

```json
{
  "companies": [
    {
      "organization_number": "923609016",
      "name": "EQUINOR ASA",
      "organization_form": "ASA",
      "status": "ACTIVE"
    },
    {
      "organization_number": "914778271",
      "name": "TELENOR ASA",
      "organization_form": "ASA",
      "status": "ACTIVE"
    }
  ],
  "total_results": 2,
  "not_found": ["999999999"]
}
```

**Example prompts:**
- "Look up these companies: 923609016, 914778271"
- "Validate these org numbers from my spreadsheet"

---

### norway_get_subunits

Get branch offices (sub-units) for a parent company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `parent_org_number` | string | Yes | Parent company's 9-digit org number |

**Returns:**

```json
{
  "sub_units": [
    {
      "organization_number": "974760673",
      "name": "EQUINOR ASA AVD FORNEBU",
      "parent_org_number": "923609016",
      "business_address": "Martin Linges vei 33, 1364 FORNEBU",
      "employee_count": 500
    }
  ],
  "total_results": 1
}
```

**Example prompts:**
- "What branches does 923609016 have?"
- "List sub-units for Equinor"

---

### norway_get_subunit

Get details for a specific branch office.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_number` | string | Yes | Sub-unit's 9-digit org number |

**Example prompts:**
- "Get details for sub-unit 974760673"

---

### norway_search_subunits

Search for branch offices by name.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Sub-unit name to search for |
| `page` | int | No | Page number (0-indexed) |
| `size` | int | No | Results per page (max 100) |
| `municipality` | string | No | Filter by municipality code |

**Example prompts:**
- "Search for branch offices named Equinor"
- "Find sub-units in Oslo"

---

### norway_get_updates

Get recent changes from the business registry.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `since` | string | Yes | ISO 8601 timestamp (e.g., "2024-01-15T00:00:00Z") |
| `size` | int | No | Maximum updates to return (default: 20) |

**Returns:**

```json
{
  "updates": [
    {
      "update_id": 12345,
      "organization_number": "923609016",
      "updated_at": "2024-01-15T14:30:00Z",
      "change_type": "Endring"
    }
  ]
}
```

**Note:** This endpoint is NOT cached (real-time data).

**Example prompts:**
- "What companies changed since yesterday?"
- "Get recent registry updates"

---

### norway_get_subunit_updates

Get recent changes to branch offices.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `since` | string | Yes | ISO 8601 timestamp |
| `size` | int | No | Maximum updates to return |

**Example prompts:**
- "What branch offices changed recently?"

---

### norway_list_municipalities

List all Norwegian municipality codes.

**Parameters:** None

**Returns:**

```json
{
  "municipalities": [
    {"number": "0301", "name": "Oslo"},
    {"number": "1103", "name": "Stavanger"},
    {"number": "4601", "name": "Bergen"}
  ],
  "count": 356
}
```

**Note:** Cached for 24 hours.

**Example prompts:**
- "List all Norwegian municipalities"
- "What is Oslo's municipality code?"

---

### norway_list_org_forms

List all organization form codes.

**Parameters:** None

**Returns:**

```json
{
  "org_forms": [
    {"code": "AS", "description": "Aksjeselskap"},
    {"code": "ASA", "description": "Allmennaksjeselskap"},
    {"code": "ENK", "description": "Enkeltpersonforetak"}
  ],
  "count": 45
}
```

**Note:** Cached for 24 hours.

**Example prompts:**
- "What does AS mean?"
- "List organization form codes"

---

## Denmark (CVR)

### denmark_search_companies

Search for Danish companies by name.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Company name to search for |

**Note:** The CVR API returns a single best match, not a paginated list.

**Returns:**

```json
{
  "found": true,
  "company": {
    "cvr": "10150817",
    "name": "NOVO NORDISK A/S",
    "company_type": "A/S",
    "address": "Novo Allé, 2880 Bagsværd",
    "industry": "Fremstilling af farmaceutiske råvarer",
    "employees": 20000,
    "founded": "1925-01-01",
    "status": "NORMAL"
  }
}
```

**Example prompts:**
- "Find Danish company Novo Nordisk"
- "Search for Carlsberg in Denmark"

---

### denmark_get_company

Get detailed information about a Danish company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `cvr` | string | Yes | 8-digit CVR number |

**CVR Number Formats (all accepted):**
- `10150817` (digits only)
- `DK10150817` (with DK prefix)
- `DK-10150817` (with DK prefix and dash)

**Example prompts:**
- "Get details for CVR 10150817"
- "Look up Danish company DK-10150817"

---

### denmark_get_production_units

Get production units (P-numbers) for a Danish company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `cvr` | string | Yes | 8-digit CVR number |

**Returns:**

```json
{
  "production_units": [
    {
      "p_number": "1000067892",
      "name": "NOVO NORDISK A/S - KALUNDBORG",
      "address": "Hallas Allé 1, 4400 Kalundborg",
      "is_main": false,
      "employee_count": 5000,
      "industry": "Fremstilling af farmaceutiske præparater"
    }
  ],
  "total": 1
}
```

**Example prompts:**
- "What production units does CVR 10150817 have?"
- "List P-numbers for Novo Nordisk"

---

### denmark_search_by_phone

Find a Danish company by phone number.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `phone` | string | Yes | Phone number (any format) |

**Example prompts:**
- "Find company with phone 33121212"

---

### denmark_get_by_pnumber

Get company details by P-number (production unit identifier).

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `p_number` | string | Yes | 10-digit P-number |

**Example prompts:**
- "Look up production unit P-number 1000067892"

---

## Finland (PRH)

### finland_search_companies

Search for Finnish companies by name.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Company name to search for |
| `location` | string | No | Filter by city/town |
| `company_form` | string | No | Filter by form (OY, OYJ, Ky, etc.) |
| `page` | int | No | Page number for pagination |

**Returns:**

```json
{
  "companies": [
    {
      "business_id": "0112038-9",
      "name": "Nokia Oyj",
      "company_form": "OYJ",
      "city": "Espoo",
      "industry": "Tietoliikennelaitteiden valmistus",
      "registration_date": "1871-05-12",
      "status": "ACTIVE"
    }
  ],
  "total_results": 1,
  "page": 1,
  "total_pages": 1
}
```

**Example prompts:**
- "Find Finnish company Nokia"
- "Search for companies in Helsinki"

---

### finland_get_company

Get detailed information about a Finnish company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `business_id` | string | Yes | Finnish business ID (Y-tunnus) |

**Business ID Formats (all accepted):**
- `0112038-9` (standard format)
- `FI0112038-9` (with FI prefix)
- `01120389` (without hyphen - hyphen will be added)

**Returns:**

```json
{
  "company": {
    "business_id": "0112038-9",
    "eu_id": "FI01120389",
    "name": "Nokia Oyj",
    "auxiliary_names": ["Nokia Corporation"],
    "company_form": "OYJ",
    "street_address": "Karakaari 7, 02610 ESPOO",
    "postal_address": "PL 226, 00045 NOKIA GROUP",
    "industry_code": "26300",
    "industry_description": "Tietoliikennelaitteiden valmistus",
    "website": "https://www.nokia.com",
    "registration_date": "1871-05-12",
    "status": "ACTIVE"
  }
}
```

**Example prompts:**
- "Get Finnish company 0112038-9"
- "Look up business ID FI0112038-9"

---

## Sweden (Bolagsverket)

Uses the free **värdefulla datamängder** (High Value Datasets) API, mandated by EU Open Data Directive.

**Registration:** [Kundanmälan till API för värdefulla datamängder](https://bolagsverket.se/apierochoppnadata/vardefulladatamangder/kundanmalantillapiforvardefulladatamangder.5528.html) (free, requires OAuth2 credentials)

Set environment variables:
- `BOLAGSVERKET_CLIENT_ID`
- `BOLAGSVERKET_CLIENT_SECRET`

### sweden_get_company

Get detailed information about a Swedish company by organization number.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_number` | string | Yes | Swedish organization number (10 digits) or personal number (12 digits) |

**Org Number Formats (all accepted):**
- `5560125790` (digits only)
- `5560-1257-90` (with dashes)
- `556012-5790` (common Swedish format)

**Returns:**

```json
{
  "company": {
    "organization_number": "5560125790",
    "name": "Volvo Personvagnar AB",
    "organization_form": "AB",
    "organization_form_description": "Aktiebolag",
    "legal_form": "Privat aktiebolag",
    "business_description": "Tillverkning av personbilar",
    "registration_date": "1927-04-14",
    "registration_country": "Sverige",
    "address": "Assar Gabrielssons Väg, 405 31 GÖTEBORG",
    "is_active": true,
    "ad_block": false,
    "sni_codes": [
      {"code": "29102", "description": "Tillverkning av personbilar"}
    ]
  }
}
```

**Example prompts:**
- "Get Swedish company 5560125790"
- "Look up 5560-1257-90"
- "Get details for Volvo's org number"

---

### sweden_get_document_list

List available annual reports (årsredovisningar) for a Swedish company.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `org_number` | string | Yes | Swedish organization number (10 digits) |

**Returns:**

```json
{
  "documents": [
    {
      "document_id": "abc123",
      "format": "XBRL",
      "period_end": "2023-12-31",
      "registration_date": "2024-06-15"
    }
  ],
  "count": 5
}
```

**Note:** Returns metadata about available reports, not the reports themselves. Reports are ZIP files containing XBRL or PDF data.

**Example prompts:**
- "What annual reports exist for 5560125790?"
- "List financial reports for Swedish company X"
- "Show årsredovisningar for Volvo"

---

### sweden_download_document

Download an annual report (årsredovisning) by document ID.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `document_id` | string | Yes | Document ID from `sweden_get_document_list` |

**Returns:**

```json
{
  "document_id": "abc123",
  "filename": "arsredovisning_5560125790_2023.zip",
  "content_base64": "UEsDBBQAAAAI...",
  "size_bytes": 45678,
  "format": "XBRL"
}
```

**Note:** Returns a base64-encoded ZIP file containing XBRL/iXBRL files. Use `sweden_get_document_list` first to get available document IDs.

**Example prompts:**
- "Download the 2023 annual report for company X"
- "Get the årsredovisning with document ID abc123"

---

### sweden_check_status

Check the status of the Swedish Bolagsverket API and OAuth2 connection.

**Parameters:** None

**Returns:**

```json
{
  "available": true,
  "circuit_breaker_status": "closed",
  "cache_entries": 42
}
```

**Example prompts:**
- "Is the Swedish API working?"
- "Check connection to Bolagsverket"
- "Status of Sweden registry"

---

## Error Responses

All tools return consistent error messages:

### Not Found

```json
{
  "error": "company not found in norway registry: 999999999"
}
```

### Validation Error

```json
{
  "error": "validation failed for org_number=12345: organization number must be 9 digits, got 5"
}
```

### Circuit Breaker Open

```json
{
  "error": "circuit breaker is open: API is experiencing issues, retry after 2024-01-15T14:35:00Z"
}
```

### Rate Limited

```json
{
  "error": "rate limited (429)"
}
```

---

## Organization Form Reference

### Norway

| Code | Norwegian | English |
|------|-----------|---------|
| AS | Aksjeselskap | Private limited company |
| ASA | Allmennaksjeselskap | Public limited company |
| ENK | Enkeltpersonforetak | Sole proprietorship |
| NUF | Norsk avdeling av utenlandsk foretak | Norwegian branch of foreign company |
| ANS | Ansvarlig selskap | General partnership |
| DA | Selskap med delt ansvar | Limited partnership |
| SA | Samvirkeforetak | Cooperative |
| STI | Stiftelse | Foundation |

### Denmark

| Code | Danish | English |
|------|--------|---------|
| A/S | Aktieselskab | Public limited company |
| ApS | Anpartsselskab | Private limited company |
| I/S | Interessentskab | General partnership |
| K/S | Kommanditselskab | Limited partnership |
| P/S | Partnerselskab | Partnership company |
| IVS | Iværksætterselskab | Entrepreneurial company |

### Finland

| Code | Finnish | English |
|------|---------|---------|
| OY | Osakeyhtiö | Private limited company |
| OYJ | Julkinen osakeyhtiö | Public limited company |
| Ky | Kommandiittiyhtiö | Limited partnership |
| Ay | Avoin yhtiö | General partnership |
| Tmi | Toiminimi | Sole proprietorship |
| Osk | Osuuskunta | Cooperative |

### Sweden

| Code | Swedish | English |
|------|---------|---------|
| AB | Aktiebolag | Private limited company |
| Publikt AB | Publikt aktiebolag | Public limited company |
| E | Enskild firma | Sole proprietorship |
| HB | Handelsbolag | General partnership |
| KB | Kommanditbolag | Limited partnership |
| EK | Ekonomisk förening | Cooperative |
| BRF | Bostadsrättsförening | Housing cooperative |
| FL | Filial | Branch of foreign company |
