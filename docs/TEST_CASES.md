# Nordic Registry MCP Server - Test Cases

Comprehensive test cases for all 22 tools across Norway, Denmark, Finland, and Sweden.

## Test Data Reference

| Country | ID Type | Valid Examples | Invalid Examples |
|---------|---------|----------------|------------------|
| Norway | Org Number (9 digits) | `923609016` (Equinor), `985399077` (DNB) | `123456789`, `000000000` |
| Denmark | CVR (8 digits) | `10150817` (Novo Nordisk), `25063783` (Mærsk) | `12345678`, `00000000` |
| Finland | Y-tunnus (7+1 digits) | `0112038-9` (Nokia), `2331972-7` | `1234567-8`, `0000000-0` |
| Sweden | Org Number (10 digits) | `5560125790` (Volvo), `5560003468` | `1234567890`, `0000000000` |

---

## Norway (12 tools)

### norway_search_companies

Search companies by name with optional filters.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-S-01 | "Search for Norwegian companies named Equinor" | Returns list with Equinor ASA |
| NO-S-02 | "Find AS companies in Oslo" | Returns aksjeselskap in municipality 0301 |
| NO-S-03 | "Search Telenor in Norway" | Returns Telenor-related companies |
| NO-S-04 | "Find bankrupt companies named XYZ" | Uses bankrupt=true filter |
| NO-S-05 | "Search for ENK companies named Hansen" | Filters by org_form=ENK |
| NO-S-06 | "Find VAT-registered companies named Bygg" | Uses registered_in_vat=true |
| NO-S-07 | "Search voluntary organizations named Røde Kors" | Uses registered_in_voluntary=true |
| NO-S-08 | "Find NUF companies" | Returns Norwegian branches of foreign companies |
| NO-S-09 | "Search companies in Bergen municipality" | Uses municipality filter (4601) |
| NO-S-10 | "Find 5 companies named Consulting" | Uses size parameter |

**Edge cases:**
- Empty search term → Should return error or guidance
- Very long company name → Should handle gracefully
- Special characters (ÆØÅ) → "Search for Røros Flyservice"

---

### norway_get_company

Get company details by org number.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-G-01 | "Get details for org number 923609016" | Returns Equinor ASA details |
| NO-G-02 | "Look up Norwegian company 985399077" | Returns DNB details |
| NO-G-03 | "Get company 923 609 016" | Handles spaces in org number |
| NO-G-04 | "Get full details for 923609016" | Uses full=true, returns all fields |
| NO-G-05 | "What is org number 914778271?" | Natural language lookup |
| NO-G-06 | "Company info for 923-609-016" | Handles dashes in org number |

**Edge cases:**
- Invalid org number (8 digits) → "Get company 12345678"
- Non-existent org number → "Get company 999999999"
- Letters in org number → "Get company ABC123456"

---

### norway_get_roles

Get board members, CEO, and other roles.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-R-01 | "Who is on the board of 923609016?" | Returns board members |
| NO-R-02 | "Find the CEO of Equinor (923609016)" | Returns daglig leder |
| NO-R-03 | "List all directors for org 914778271" | Returns full role list |
| NO-R-04 | "Who are the auditors for 985399077?" | Returns revisor entries |
| NO-R-05 | "Get board composition of company 923609016" | Returns styremedlemmer |
| NO-R-06 | "Show resigned board members for 923609016" | Includes resignation status |

**Edge cases:**
- Company with no roles registered
- Sole proprietorship (ENK) → Different role structure

---

### norway_get_signature_rights

Get signature authority (signaturrett) and prokura.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-SR-01 | "Who can sign for company 923609016?" | Returns signature rights |
| NO-SR-02 | "Get signature rights for Equinor" | Returns signaturrett |
| NO-SR-03 | "Who has prokura for 914778271?" | Returns prokura holders |
| NO-SR-04 | "Check signing authority for 985399077" | Returns combined signature info |
| NO-SR-05 | "List authorized signatories for 923609016" | Natural language variant |

**Edge cases:**
- Company with joint signature requirement
- Company with no prokura

---

### norway_batch_get_companies

Look up multiple companies at once.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-B-01 | "Look up these companies: 923609016, 914778271, 985399077" | Returns 3 company summaries |
| NO-B-02 | "Validate these org numbers: 923609016, 999999999" | Returns found + not_found |
| NO-B-03 | "Get details for multiple Norwegian companies at once" | Batch operation |
| NO-B-04 | "Batch lookup: 923609016, 914778271" | Alternative phrasing |

**Edge cases:**
- Mix of valid and invalid org numbers
- Empty list
- Duplicate org numbers in list
- Large batch (100+ companies)

---

### norway_get_subunits

Get branch offices for a parent company.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-SU-01 | "What branches does 923609016 have?" | Returns sub-units list |
| NO-SU-02 | "List branch offices for Equinor" | Returns sub-units |
| NO-SU-03 | "Get all locations for company 985399077" | Returns sub-units |
| NO-SU-04 | "Show subsidiaries of 914778271" | Returns sub-units (note: not legal subsidiaries) |

**Edge cases:**
- Company with no sub-units
- Company with many sub-units (pagination)

---

### norway_get_subunit

Get details for a specific sub-unit.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-SUD-01 | "Get details for sub-unit 912345678" | Returns sub-unit info |
| NO-SUD-02 | "Look up branch office 912345678" | Returns sub-unit details |
| NO-SUD-03 | "What is sub-unit 912345678?" | Natural language |

**Edge cases:**
- Sub-unit org number that doesn't exist
- Using parent company org number (should redirect to get_company)

---

### norway_search_subunits

Search branch offices by name.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-SSU-01 | "Search for branch offices named Equinor" | Returns matching sub-units |
| NO-SSU-02 | "Find sub-units in Oslo" | Uses municipality filter |
| NO-SSU-03 | "Search branches named Statoil" | Historical name search |

**Edge cases:**
- No matching sub-units
- Very common name with many results

---

### norway_get_updates

Get recent registry changes.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-U-01 | "What companies changed since yesterday?" | Uses recent timestamp |
| NO-U-02 | "Get registry updates from last week" | Returns update feed |
| NO-U-03 | "Show recent company registrations" | Recent changes |
| NO-U-04 | "What changed in the registry since 2024-01-01?" | Specific date |

**Edge cases:**
- Very old timestamp (large result set)
- Future timestamp
- No changes in period

---

### norway_get_subunit_updates

Get recent branch office changes.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-SUU-01 | "What branch offices changed recently?" | Returns sub-unit updates |
| NO-SUU-02 | "Get sub-unit updates since yesterday" | Recent timestamp |
| NO-SUU-03 | "Show recent branch registrations" | Recent changes |

---

### norway_list_municipalities

List municipality codes for filtering.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-M-01 | "List all Norwegian municipalities" | Returns full list |
| NO-M-02 | "What is Oslo's municipality code?" | Can search/filter result |
| NO-M-03 | "Show kommune codes" | Norwegian term |
| NO-M-04 | "Get municipality reference data" | Reference lookup |

---

### norway_list_org_forms

List organization form codes.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| NO-OF-01 | "What does AS mean?" | Returns org form descriptions |
| NO-OF-02 | "List organization form codes" | Returns full list |
| NO-OF-03 | "What is ENK?" | Sole proprietorship |
| NO-OF-04 | "Explain Norwegian company types" | Reference data |
| NO-OF-05 | "What is a NUF?" | Norwegian branch of foreign company |

---

## Denmark (5 tools)

### denmark_search_companies

Search Danish companies by name.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| DK-S-01 | "Find Danish company Novo Nordisk" | Returns Novo Nordisk A/S |
| DK-S-02 | "Search for Carlsberg in Denmark" | Returns Carlsberg |
| DK-S-03 | "Look up Danish company Mærsk" | Handles special characters |
| DK-S-04 | "Find ApS companies named Jensen" | Search with type hint |
| DK-S-05 | "Search Danish companies named LEGO" | Returns LEGO companies |

**Edge cases:**
- CVR API returns single best match (not list)
- Company name with ÆØÅ characters

---

### denmark_get_company

Get company by CVR number.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| DK-G-01 | "Get details for CVR 10150817" | Returns Novo Nordisk |
| DK-G-02 | "Look up CVR DK-10150817" | Handles DK prefix |
| DK-G-03 | "Get Danish company 25063783" | Returns Mærsk |
| DK-G-04 | "Full details for CVR 10150817" | Uses full=true |
| DK-G-05 | "What is CVR 37553472?" | Natural language |
| DK-G-06 | "Company info for DK10150817" | No dash in prefix |

**Edge cases:**
- Invalid CVR (9 digits)
- Non-existent CVR
- Bankrupt company

---

### denmark_get_production_units

Get production units (P-numbers) for a company.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| DK-PU-01 | "What production units does CVR 10150817 have?" | Returns P-numbers |
| DK-PU-02 | "List locations for Novo Nordisk" | Returns production units |
| DK-PU-03 | "Get P-numbers for CVR 25063783" | Returns production units |
| DK-PU-04 | "Show all sites for Danish company 10150817" | Alternative phrasing |

**Edge cases:**
- Company with no production units
- Company with many production units

---

### denmark_search_by_phone

Find company by phone number.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| DK-PH-01 | "Find Danish company with phone 33121212" | Returns matching company |
| DK-PH-02 | "Search CVR by phone +45 33 12 12 12" | Handles +45 prefix |
| DK-PH-03 | "What company has phone number 44440000?" | Natural language |
| DK-PH-04 | "Look up Danish company by phone 4533121212" | Handles country code |

**Edge cases:**
- Phone number not registered
- Multiple formats (+45, 0045, spaces, dashes)

---

### denmark_get_by_pnumber

Get parent company by P-number.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| DK-PN-01 | "Look up production unit P-number 1234567890" | Returns parent company |
| DK-PN-02 | "Get company for P-number 1234567890" | Returns owning company |
| DK-PN-03 | "What company owns P-number 1234567890?" | Natural language |

**Edge cases:**
- Invalid P-number
- Non-existent P-number

---

## Finland (2 tools)

### finland_search_companies

Search Finnish companies by name.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| FI-S-01 | "Find Finnish company Nokia" | Returns Nokia Oyj |
| FI-S-02 | "Search for Kone in Finland" | Returns Kone companies |
| FI-S-03 | "Look up Finnish company Fazer" | Returns Fazer |
| FI-S-04 | "Find OY companies named Consulting" | Filter by company_form |
| FI-S-05 | "Search Finnish companies in Helsinki" | Filter by location |
| FI-S-06 | "Find OYJ companies" | Public limited companies |

**Edge cases:**
- Company name with Finnish characters (ä, ö)
- Very common name

---

### finland_get_company

Get company by Y-tunnus (business ID).

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| FI-G-01 | "Get Finnish company 0112038-9" | Returns Nokia |
| FI-G-02 | "Look up Y-tunnus 0112038-9" | Finnish term |
| FI-G-03 | "Get company FI0112038-9" | Handles FI prefix |
| FI-G-04 | "Full details for Finnish company 0112038-9" | Uses full=true |
| FI-G-05 | "What is business ID 2331972-7?" | Alternative phrasing |

**Edge cases:**
- Missing check digit → "Get company 0112038"
- Invalid check digit → "Get company 0112038-0"
- Non-existent business ID

---

## Sweden (4 tools)

**Note:** Sweden tools require OAuth2 credentials (BOLAGSVERKET_CLIENT_ID/SECRET).

### sweden_get_company

Get company by organization number.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| SE-G-01 | "Get Swedish company 5560125790" | Returns Volvo |
| SE-G-02 | "Look up Swedish org number 5560003468" | Returns company details |
| SE-G-03 | "Get company 556012-5790" | Handles dash format |
| SE-G-04 | "What is Swedish company 5560125790?" | Natural language |
| SE-G-05 | "Look up 5560-1257-90" | Multiple dashes |

**Edge cases:**
- No search by name available (must have org number)
- Invalid org number format
- OAuth2 credentials not configured

---

### sweden_get_document_list

List available annual reports.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| SE-DL-01 | "What annual reports exist for 5560125790?" | Returns document list |
| SE-DL-02 | "List årsredovisningar for Swedish company X" | Swedish term |
| SE-DL-03 | "Show financial reports for 5560125790" | Alternative phrasing |
| SE-DL-04 | "Get available documents for Swedish org 5560003468" | Returns metadata |

**Edge cases:**
- Company with no filed reports
- Very old company with many reports

---

### sweden_download_document

Download annual report by document ID.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| SE-DD-01 | "Download the 2023 annual report for company X" | Downloads specific report |
| SE-DD-02 | "Get the årsredovisning with document ID abc123" | Uses document_id |
| SE-DD-03 | "Download annual report abc123" | Direct ID reference |
| SE-DD-04 | "Fetch Swedish company report with ID xyz789" | Alternative phrasing |

**Edge cases:**
- Invalid document ID
- Document ID from different company
- Large file size

---

### sweden_check_status

Check API availability and OAuth2 status.

| Test ID | Query Variation | Expected Behavior |
|---------|-----------------|-------------------|
| SE-ST-01 | "Is the Swedish API working?" | Returns status |
| SE-ST-02 | "Check connection to Bolagsverket" | Returns availability |
| SE-ST-03 | "Status of Sweden registry" | Returns circuit breaker status |
| SE-ST-04 | "Are Swedish credentials configured?" | OAuth2 check |

**Edge cases:**
- Credentials not configured
- API temporarily unavailable
- Circuit breaker open

---

## HTTP Endpoints (if running in HTTP mode)

| Endpoint | Test Case | Expected |
|----------|-----------|----------|
| `/health` | GET request | `200 OK` |
| `/ready` | GET request | `200 OK` if APIs reachable |
| `/tools` | GET request | JSON list of all tools by country |
| `/status` | GET request | Circuit breaker stats |
| `/metrics` | GET request | Prometheus metrics |
| `/` | MCP protocol request | Streamable HTTP response |

**With authentication (-token flag):**

| Test Case | Expected |
|-----------|----------|
| Request without token | `401 Unauthorized` |
| Request with invalid token | `401 Unauthorized` |
| Request with valid token | `200 OK` |

---

## Cross-Cutting Test Scenarios

### Error Handling

| Scenario | Expected Behavior |
|----------|-------------------|
| Network timeout | Graceful error message |
| API rate limit | Retry with backoff |
| Invalid JSON response | Error handling |
| Circuit breaker open | Fast fail with explanation |

### Caching

| Scenario | Expected Behavior |
|----------|-------------------|
| Repeated identical request | Returns cached result |
| Request after TTL expiry | Fresh API call |
| Cache miss | API call, result cached |

### Input Normalization

| Input | Normalized |
|-------|------------|
| `923 609 016` | `923609016` |
| `DK-10150817` | `10150817` |
| `FI0112038-9` | `0112038-9` |
| `556012-5790` | `5560125790` |

---

## Test Execution Checklist

### Per-Country Smoke Tests

- [ ] **Norway**: `norway_get_company("923609016")` returns Equinor
- [ ] **Denmark**: `denmark_get_company("10150817")` returns Novo Nordisk
- [ ] **Finland**: `finland_get_company("0112038-9")` returns Nokia
- [ ] **Sweden**: `sweden_check_status()` returns availability (requires credentials)

### Integration Tests

- [ ] Search → Get flow works for each country
- [ ] Batch operations handle mixed valid/invalid inputs
- [ ] Reference data endpoints return cached results
- [ ] Update feeds return real-time data

### Resilience Tests

- [ ] Circuit breaker trips after consecutive failures
- [ ] Request deduplication prevents duplicate API calls
- [ ] Cache reduces API load for repeated queries
