package sweden

import (
	"context"
	"errors"
	"fmt"
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

	// Handle case where no organization was found
	if resp == nil || len(resp.Organisationer) == 0 {
		return GetCompanyResult{}, fmt.Errorf("no company found for organization number %s", args.OrgNumber)
	}

	// Take the first organization (there can be multiple for sole proprietors)
	org := resp.Organisationer[0]

	summary := &CompanySummary{
		OrganizationNumber:  org.GetOrgNumber(),
		Name:                org.GetName(),
		IsActive:            org.IsActive(),
		RegistrationDate:    org.GetRegistrationDate(),
		PostalAddress:       org.GetAddress(),
		BusinessDescription: org.GetBusinessDescription(),
	}

	// Organization form (e.g., "AB - Aktiebolag")
	if org.Organisationsform != nil {
		if org.Organisationsform.Klartext != "" {
			summary.OrganizationForm = org.Organisationsform.Kod + " - " + org.Organisationsform.Klartext
		} else {
			summary.OrganizationForm = org.Organisationsform.Kod
		}
	}

	// Legal form (e.g., "49 - Övriga aktiebolag")
	if org.JuridiskForm != nil {
		if org.JuridiskForm.Klartext != "" {
			summary.LegalForm = org.JuridiskForm.Kod + " - " + org.JuridiskForm.Klartext
		} else {
			summary.LegalForm = org.JuridiskForm.Kod
		}
	}

	// Registration country
	if org.Registreringsland != nil {
		summary.RegistrationCountry = org.Registreringsland.Klartext
	}

	// Deregistration info
	if org.AvregistreradOrganisation != nil && org.AvregistreradOrganisation.Avregistreringsdatum != "" {
		summary.DeregisteredDate = org.AvregistreradOrganisation.Avregistreringsdatum
		summary.IsActive = false
	}
	if org.Avregistreringsorsak != nil && org.Avregistreringsorsak.Klartext != "" {
		summary.DeregisteredReason = org.Avregistreringsorsak.Klartext
	}

	// Ongoing proceedings (bankruptcy, liquidation, etc.)
	if org.PagaendeAvvecklingsEllerOmstruktureringsforfarande != nil {
		for _, p := range org.PagaendeAvvecklingsEllerOmstruktureringsforfarande.PagaendeAvvecklingsEllerOmstruktureringsforfarandeLista {
			desc := p.Klartext
			if desc == "" {
				desc = p.Kod
			}
			if p.FromDatum != "" {
				desc += " (from " + p.FromDatum + ")"
			}
			summary.OngoingProceedings = append(summary.OngoingProceedings, desc)
		}
	}

	// Industry codes (SNI)
	for _, sni := range org.GetSNICodes() {
		code := sni.Kod
		if sni.Klartext != "" {
			code += " - " + sni.Klartext
		}
		summary.IndustryCodes = append(summary.IndustryCodes, code)
	}

	// Ad block (reklamsparr)
	if org.Reklamsparr != nil && org.Reklamsparr.Kod == JaNejJA {
		summary.AdBlockEnabled = true
	}

	return GetCompanyResult{Company: summary}, nil
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
