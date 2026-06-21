package sweden

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
)

// MCP Tool wrapper methods
// These methods wrap the client methods with Args/Result types for MCP integration.

// Validation patterns for Swedish organization numbers
var (
	// Swedish org number: 10 digits (NNNNNN-NNNN or NNNNNNNNNN)
	// Personal number: 12 digits (YYYYMMDD-NNNN or YYYYMMDDNNNN)
	// GD-nummer: 10 digits starting with 302
	orgNumberPattern = regexp.MustCompile(`^\d{10,12}$`)
)

// ValidateOrgNumber validates a Swedish organization number.
func ValidateOrgNumber(orgNumber string) error {
	normalized := NormalizeOrgNumber(orgNumber)
	if normalized == "" {
		return errors.New("organization number is required")
	}
	if !orgNumberPattern.MatchString(normalized) {
		return fmt.Errorf("invalid organization number format: expected 10-12 digits, got %q", orgNumber)
	}
	return nil
}

// GetCompanyMCP is the MCP wrapper for GetCompany.
func (c *Client) GetCompanyMCP(ctx context.Context, args GetCompanyArgs) (GetCompanyResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetCompanyResult{}, err
	}

	resp, err := c.GetCompany(ctx, args.OrgNumber)
	if err != nil {
		return GetCompanyResult{}, err
	}

	if resp == nil || len(resp.Organisationer) == 0 {
		return GetCompanyResult{}, fmt.Errorf("no company found for organization number %s", args.OrgNumber)
	}

	// Take the first organization (there can be multiple for sole proprietors)
	org := resp.Organisationer[0]

	summary := buildCompanySummary(&org)
	return GetCompanyResult{Company: summary}, nil
}

// buildCompanySummary projects a Bolagsverket Organisation into the flat
// CompanySummary the MCP caller consumes. The Bolagsverket schema is heavily
// optional, so the helpers below isolate each optional field group.
//
//nolint:misspell // Swedish API uses "Organisation"
func buildCompanySummary(org *Organisation) *CompanySummary {
	summary := &CompanySummary{
		OrganizationNumber:  org.GetOrgNumber(),
		Name:                org.GetName(),
		IsActive:            org.IsActive(),
		RegistrationDate:    org.GetRegistrationDate(),
		PostalAddress:       org.GetAddress(),
		BusinessDescription: org.GetBusinessDescription(),
	}
	if org.Organisationsform != nil {
		summary.OrganizationForm = formatCodeKlartext(org.Organisationsform.Kod, org.Organisationsform.Klartext)
	}
	if org.JuridiskForm != nil {
		summary.LegalForm = formatCodeKlartext(org.JuridiskForm.Kod, org.JuridiskForm.Klartext)
	}

	if org.Registreringsland != nil {
		summary.RegistrationCountry = org.Registreringsland.Klartext
	}

	applyDeregistrationInfo(summary, org)
	summary.OngoingProceedings = collectOngoingProceedings(org)
	summary.IndustryCodes = collectSNICodes(org)

	if org.Reklamsparr != nil && org.Reklamsparr.Kod == JaNejJA {
		summary.AdBlockEnabled = true
	}
	return summary
}

// formatCodeKlartext formats a Bolagsverket Kod/Klartext pair as "Kod - Klartext"
// when both are present, otherwise the bare code.
func formatCodeKlartext(kod, klartext string) string {
	if klartext != "" {
		return kod + " - " + klartext
	}
	return kod
}

// applyDeregistrationInfo copies deregistration date/reason fields from the
// upstream response into the summary, marking the company inactive if a
// deregistration date is present.
//
//nolint:misspell // Swedish API uses "Organisation"
func applyDeregistrationInfo(summary *CompanySummary, org *Organisation) {
	if org.AvregistreradOrganisation != nil && org.AvregistreradOrganisation.Avregistreringsdatum != "" {
		summary.DeregisteredDate = org.AvregistreradOrganisation.Avregistreringsdatum
		summary.IsActive = false
	}
	if org.Avregistreringsorsak != nil && org.Avregistreringsorsak.Klartext != "" {
		summary.DeregisteredReason = org.Avregistreringsorsak.Klartext
	}
}

// collectOngoingProceedings flattens the Bolagsverket bankruptcy / liquidation
// list into human-readable strings.
//
//nolint:misspell // Swedish API uses "Organisation"
func collectOngoingProceedings(org *Organisation) []string {
	if org.PagaendeAvvecklingsEllerOmstruktureringsforfarande == nil {
		return nil
	}
	var out []string
	for _, p := range org.PagaendeAvvecklingsEllerOmstruktureringsforfarande.PagaendeAvvecklingsEllerOmstruktureringsforfarandeLista {
		desc := p.Klartext
		if desc == "" {
			desc = p.Kod
		}
		if p.FromDatum != "" {
			desc += " (from " + p.FromDatum + ")"
		}
		out = append(out, desc)
	}
	return out
}

// collectSNICodes flattens the SNI (Swedish industry classification) codes
// into "Kod - Klartext" form for the MCP caller.
//
//nolint:misspell // Swedish API uses "Organisation"
func collectSNICodes(org *Organisation) []string {
	var out []string
	for _, sni := range org.GetSNICodes() {
		code := sni.Kod
		if sni.Klartext != "" {
			code += " - " + sni.Klartext
		}
		out = append(out, code)
	}
	return out
}

// GetDocumentListMCP is the MCP wrapper for GetDocumentList.
func (c *Client) GetDocumentListMCP(ctx context.Context, args GetDocumentListArgs) (GetDocumentListResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetDocumentListResult{}, err
	}

	resp, err := c.GetDocumentList(ctx, args.OrgNumber)
	if err != nil {
		return GetDocumentListResult{}, err
	}

	result := GetDocumentListResult{
		OrganizationNumber: NormalizeOrgNumber(args.OrgNumber),
		Documents:          make([]DocumentSummary, 0, len(resp.Dokument)),
	}

	for _, doc := range resp.Dokument {
		result.Documents = append(result.Documents, DocumentSummary{
			DocumentID:            doc.DokumentID,
			FileFormat:            doc.Filformat,
			ReportingPeriodEnd:    doc.RapporteringsperiodTom,
			RegistrationTimestamp: doc.Registreringstidpunkt,
		})
	}

	result.Count = len(result.Documents)

	return result, nil
}

// CheckStatusMCP is the MCP wrapper for checking API status.
func (c *Client) CheckStatusMCP(ctx context.Context, args CheckStatusArgs) (CheckStatusResult, error) {
	result := CheckStatusResult{
		CircuitBreakerStatus: c.CircuitBreakerStatus(),
		CacheEntries:         c.CacheSize(),
	}

	available, err := c.IsAlive(ctx)
	if err != nil {
		result.Available = false
		return result, nil // Return result even on error
	}

	result.Available = available
	return result, nil
}

// DownloadDocumentMCP is the MCP wrapper for downloading an annual report.
func (c *Client) DownloadDocumentMCP(ctx context.Context, args DownloadDocumentArgs) (DownloadDocumentResult, error) {
	if args.DocumentID == "" {
		return DownloadDocumentResult{}, errors.New("document_id is required")
	}

	data, err := c.DownloadDocument(ctx, args.DocumentID)
	if err != nil {
		return DownloadDocumentResult{}, err
	}

	path, err := writeDocumentToTempFile(data)
	if err != nil {
		return DownloadDocumentResult{}, err
	}

	return DownloadDocumentResult{
		DocumentID: args.DocumentID,
		FileFormat: "application/zip",
		SizeBytes:  len(data),
		Path:       path,
		Description: fmt.Sprintf(
			"Annual report (årsredovisning) ZIP (%d bytes) written to %s. Read or unzip the file at that path; the bytes are not inlined to keep the response small.",
			len(data), path,
		),
	}, nil
}

// writeDocumentToTempFile writes the downloaded ZIP bytes to a server-created
// temp file and returns its path. The path is chosen by os.CreateTemp, never
// from agent input, so there is no path-traversal surface. Returning a handle
// instead of inline base64 keeps the tool response small (HG-2 cost-lens): a
// 1-10 MB annual report base64-encodes to ~33% more than its raw size, far
// past what is safe to carry into the caller's next-turn context.
func writeDocumentToTempFile(data []byte) (string, error) {
	f, err := os.CreateTemp("", "sweden-arsredovisning-*.zip")
	if err != nil {
		return "", fmt.Errorf("creating temp file for downloaded document: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return "", fmt.Errorf("writing downloaded document to temp file: %w", err)
	}
	return f.Name(), nil
}
