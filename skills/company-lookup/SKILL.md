---
name: company-lookup
description: Search and verify companies across Nordic business registries (Norway, Denmark, Finland, Sweden). Use for due diligence, vendor verification, board member lookups, and signing authority checks.
---

# Nordic Company Lookup

## Trigger

Verifying company information, checking business legitimacy, looking up board members, or validating organization numbers across Nordic countries.

## Workflow

1. **Identify the country and ID format**:
   - Norway: 9 digits (e.g., `923609016` or `923 609 016`)
   - Denmark: 8 digits (e.g., `10150817` or `DK-10150817`)
   - Finland: 7+1 format (e.g., `0112038-9`)
   - Sweden: 10 digits (e.g., `5560125790` or `556012-5790`)

2. **Search by name** — Use the country-specific search tool:
   - `norway_search_companies` — supports filters: org_form, municipality
   - `denmark_search_companies` — returns single best match; try name variations
   - `finland_search_companies` — use exact legal name or filter by company_form/location
   - Sweden has no name search; you need the org number

3. **Get company details** — Use the country-specific detail tool:
   - `norway_get_company`, `denmark_get_company`, `finland_get_company`, `sweden_get_company`

4. **Check governance (Norway)** — Use `norway_get_roles` for board members, CEO, and auditors. Use `norway_get_signature_rights` to check who can legally sign.

5. **Batch validation** — Use `norway_batch_get_companies` to verify multiple org numbers at once.

6. **Swedish documents** — Use `sweden_get_document_list` and `sweden_download_document` for annual reports (requires OAuth2 credentials).

## Common Scenarios

### Vendor verification
Look up the company, check it's active (not bankrupt), verify the signer has authority.

### Cross-border presence
Search the same company name across multiple country registries to find subsidiaries.

### Data quality
Batch-validate a list of org numbers to flag invalid, bankrupt, or missing entries.

## Guardrails

- All APIs are free and read-only. No authentication needed for Norway, Denmark, and Finland.
- Sweden requires `BOLAGSVERKET_CLIENT_ID` and `BOLAGSVERKET_CLIENT_SECRET` environment variables.
- Danish search returns only one result. Try variations for large companies ("Novo Nordisk A/S", "Novo Nordisk Denmark").
- Finnish common names return many results. Use exact legal names or filters.
- Spaces and dashes in org numbers are stripped automatically.
