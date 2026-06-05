package norway

import (
	"context"
	"strings"
)

// GetSignatureRightsMCP is the MCP wrapper for getting signature rights
func (c *Client) GetSignatureRightsMCP(ctx context.Context, args GetSignatureRightsArgs) (GetSignatureRightsResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetSignatureRightsResult{}, err
	}

	// Get roles data - signature rights are part of roles
	resp, err := c.GetRoles(ctx, args.OrgNumber)
	if err != nil {
		return GetSignatureRightsResult{}, err
	}

	result := GetSignatureRightsResult{
		OrganizationNumber: args.OrgNumber,
		SignatureRights:    []SignatureRight{},
		Prokura:            []SignatureRight{},
	}
	result.SignatureRights, result.Prokura = collectSignatureRights(resp.RoleGroups)
	result.Summary = formatSignatureSummary(result.SignatureRights, result.Prokura)

	return result, nil
}

// collectSignatureRights walks all active (non-resigned) roles and bins them
// into signaturrett (SIGN) and prokura (PROK) groups per the Brønnøysund role
// taxonomy. Resigned roles and other role codes are ignored.
func collectSignatureRights(roleGroups []RoleGroup) (signatureRights, prokura []SignatureRight) {
	signatureRights = []SignatureRight{}
	prokura = []SignatureRight{}
	for _, rg := range roleGroups {
		for _, r := range rg.Roles {
			if r.Resigned {
				continue
			}
			sr := signatureRightFromRole(r)
			switch r.Type.Code {
			case "SIGN":
				signatureRights = append(signatureRights, sr)
			case "PROK":
				prokura = append(prokura, sr)
			}
		}
	}
	return signatureRights, prokura
}

// signatureRightFromRole projects a Role into a SignatureRight, applying the
// same Person/Entity mutual-exclusivity rule as buildRoleSummary.
func signatureRightFromRole(r Role) SignatureRight {
	sr := SignatureRight{
		Type:        r.Type.Code,
		Description: r.Type.Description,
	}
	if r.Person != nil {
		sr.Name = r.Person.Name.FullName()
		sr.BirthDate = r.Person.BirthDate
	}
	if r.Entity != nil {
		sr.EntityOrgNr = r.Entity.OrganizationNumber
		if len(r.Entity.Name) > 0 {
			sr.Name = strings.Join(r.Entity.Name, " ")
		}
	}
	return sr
}

// formatSignatureSummary builds the human-readable summary string for the
// signature-rights endpoint. Empty input on both slices yields the "none found"
// fallback the upstream callers expect.
func formatSignatureSummary(signatureRights, prokura []SignatureRight) string {
	var summary strings.Builder
	appendSignatureSection(&summary, "Signature rights: ", signatureRights)
	if len(prokura) > 0 && summary.Len() > 0 {
		summary.WriteString(". ")
	}
	appendSignatureSection(&summary, "Prokura: ", prokura)
	if summary.Len() == 0 {
		return "No signature rights or prokura found"
	}
	return summary.String()
}

// appendSignatureSection writes "label: name1, name2, ..." to summary if rights
// is non-empty. No-op for empty input so callers can chain unconditionally.
func appendSignatureSection(summary *strings.Builder, label string, rights []SignatureRight) {
	if len(rights) == 0 {
		return
	}
	summary.WriteString(label)
	names := make([]string, len(rights))
	for i, sr := range rights {
		names[i] = sr.Name
	}
	summary.WriteString(strings.Join(names, ", "))
}
