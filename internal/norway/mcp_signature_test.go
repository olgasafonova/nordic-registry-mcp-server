package norway

import (
	"testing"
)

// =============================================================================
// collectSignatureRights Tests
// =============================================================================

func TestCollectSignatureRights(t *testing.T) {
	tests := []struct {
		name       string
		roleGroups []RoleGroup
		wantSign   []string
		wantProk   []string
	}{
		{
			name:       "empty input returns empty slices",
			roleGroups: []RoleGroup{},
			wantSign:   []string{},
			wantProk:   []string{},
		},
		{
			name: "SIGN roles collected",
			roleGroups: []RoleGroup{
				{
					Roles: []Role{
						{
							Type:   RoleType{Code: "SIGN", Description: "Signatur"},
							Person: &Person{Name: PersonName{FirstName: "Ola", LastName: "Nordmann"}},
						},
					},
				},
			},
			wantSign: []string{"Ola Nordmann"},
			wantProk: []string{},
		},
		{
			name: "PROK roles collected",
			roleGroups: []RoleGroup{
				{
					Roles: []Role{
						{
							Type:   RoleType{Code: "PROK", Description: "Prokura"},
							Person: &Person{Name: PersonName{FirstName: "Kari", LastName: "Nordmann"}},
						},
					},
				},
			},
			wantSign: []string{},
			wantProk: []string{"Kari Nordmann"},
		},
		{
			name: "resigned roles skipped",
			roleGroups: []RoleGroup{
				{
					Roles: []Role{
						{
							Type:     RoleType{Code: "SIGN"},
							Person:   &Person{Name: PersonName{FirstName: "Gone", LastName: "Person"}},
							Resigned: true,
						},
						{
							Type:   RoleType{Code: "SIGN"},
							Person: &Person{Name: PersonName{FirstName: "Active", LastName: "Person"}},
						},
					},
				},
			},
			wantSign: []string{"Active Person"},
			wantProk: []string{},
		},
		{
			name: "other role codes ignored",
			roleGroups: []RoleGroup{
				{
					Roles: []Role{
						{
							Type:   RoleType{Code: "DAGL", Description: "Daglig leder"},
							Person: &Person{Name: PersonName{FirstName: "The", LastName: "CEO"}},
						},
						{
							Type:   RoleType{Code: "LEDE", Description: "Styreleder"},
							Person: &Person{Name: PersonName{FirstName: "The", LastName: "Chair"}},
						},
					},
				},
			},
			wantSign: []string{},
			wantProk: []string{},
		},
		{
			name: "SIGN and PROK across multiple groups",
			roleGroups: []RoleGroup{
				{
					Roles: []Role{
						{
							Type:   RoleType{Code: "SIGN"},
							Person: &Person{Name: PersonName{FirstName: "First", LastName: "Signer"}},
						},
					},
				},
				{
					Roles: []Role{
						{
							Type:   RoleType{Code: "PROK"},
							Person: &Person{Name: PersonName{FirstName: "The", LastName: "Prokurist"}},
						},
						{
							Type:   RoleType{Code: "SIGN"},
							Person: &Person{Name: PersonName{FirstName: "Second", LastName: "Signer"}},
						},
					},
				},
			},
			wantSign: []string{"First Signer", "Second Signer"},
			wantProk: []string{"The Prokurist"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sign, prok := collectSignatureRights(tt.roleGroups)

			if sign == nil {
				t.Fatal("signatureRights should never be nil")
			}
			if prok == nil {
				t.Fatal("prokura should never be nil")
			}

			if len(sign) != len(tt.wantSign) {
				t.Fatalf("signatureRights count = %d, want %d", len(sign), len(tt.wantSign))
			}
			for i, want := range tt.wantSign {
				if sign[i].Name != want {
					t.Errorf("signatureRights[%d].Name = %q, want %q", i, sign[i].Name, want)
				}
			}

			if len(prok) != len(tt.wantProk) {
				t.Fatalf("prokura count = %d, want %d", len(prok), len(tt.wantProk))
			}
			for i, want := range tt.wantProk {
				if prok[i].Name != want {
					t.Errorf("prokura[%d].Name = %q, want %q", i, prok[i].Name, want)
				}
			}
		})
	}
}

// =============================================================================
// signatureRightFromRole Tests
// =============================================================================

func TestSignatureRightFromRole_Person(t *testing.T) {
	role := Role{
		Type: RoleType{Code: "SIGN", Description: "Signatur i fellesskap"},
		Person: &Person{
			Name:      PersonName{FirstName: "Ola", MiddleName: "Mellom", LastName: "Nordmann"},
			BirthDate: "1970-01-15",
		},
	}

	sr := signatureRightFromRole(role)

	if sr.Type != "SIGN" {
		t.Errorf("Type = %q, want %q", sr.Type, "SIGN")
	}
	if sr.Description != "Signatur i fellesskap" {
		t.Errorf("Description = %q, want %q", sr.Description, "Signatur i fellesskap")
	}
	if sr.Name != "Ola Mellom Nordmann" {
		t.Errorf("Name = %q, want %q", sr.Name, "Ola Mellom Nordmann")
	}
	if sr.BirthDate != "1970-01-15" {
		t.Errorf("BirthDate = %q, want %q", sr.BirthDate, "1970-01-15")
	}
	if sr.EntityOrgNr != "" {
		t.Errorf("EntityOrgNr = %q, want empty for person role", sr.EntityOrgNr)
	}
}

func TestSignatureRightFromRole_Entity(t *testing.T) {
	role := Role{
		Type: RoleType{Code: "PROK", Description: "Prokura"},
		Entity: &RoleEntity{
			OrganizationNumber: "923609016",
			Name:               []string{"REVISJON", "PARTNER AS"},
		},
	}

	sr := signatureRightFromRole(role)

	if sr.EntityOrgNr != "923609016" {
		t.Errorf("EntityOrgNr = %q, want %q", sr.EntityOrgNr, "923609016")
	}
	if sr.Name != "REVISJON PARTNER AS" {
		t.Errorf("Name = %q, want %q", sr.Name, "REVISJON PARTNER AS")
	}
	if sr.BirthDate != "" {
		t.Errorf("BirthDate = %q, want empty for entity role", sr.BirthDate)
	}
}

func TestSignatureRightFromRole_EntityWithEmptyName(t *testing.T) {
	role := Role{
		Type: RoleType{Code: "SIGN"},
		Entity: &RoleEntity{
			OrganizationNumber: "923609016",
			Name:               []string{},
		},
	}

	sr := signatureRightFromRole(role)

	if sr.EntityOrgNr != "923609016" {
		t.Errorf("EntityOrgNr = %q, want %q", sr.EntityOrgNr, "923609016")
	}
	if sr.Name != "" {
		t.Errorf("Name = %q, want empty when entity name list is empty", sr.Name)
	}
}

// =============================================================================
// formatSignatureSummary Tests
// =============================================================================

func TestFormatSignatureSummary(t *testing.T) {
	tests := []struct {
		name            string
		signatureRights []SignatureRight
		prokura         []SignatureRight
		expected        string
	}{
		{
			name:            "both empty",
			signatureRights: []SignatureRight{},
			prokura:         []SignatureRight{},
			expected:        "No signature rights or prokura found",
		},
		{
			name: "signature rights only",
			signatureRights: []SignatureRight{
				{Name: "Ola Nordmann"},
				{Name: "Kari Nordmann"},
			},
			prokura:  []SignatureRight{},
			expected: "Signature rights: Ola Nordmann, Kari Nordmann",
		},
		{
			name:            "prokura only",
			signatureRights: []SignatureRight{},
			prokura: []SignatureRight{
				{Name: "The Prokurist"},
			},
			expected: "Prokura: The Prokurist",
		},
		{
			name: "both present joined with separator",
			signatureRights: []SignatureRight{
				{Name: "Ola Nordmann"},
			},
			prokura: []SignatureRight{
				{Name: "Kari Nordmann"},
			},
			expected: "Signature rights: Ola Nordmann. Prokura: Kari Nordmann",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSignatureSummary(tt.signatureRights, tt.prokura)
			if result != tt.expected {
				t.Errorf("formatSignatureSummary() = %q, want %q", result, tt.expected)
			}
		})
	}
}
