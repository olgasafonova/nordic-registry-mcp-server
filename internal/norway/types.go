// Package norway provides a client for the Norwegian Brønnøysundregistrene API.
// It enables lookups of companies, roles, and sub-units in the Norwegian business registry.
package norway

import "time"

// Company represents a Norwegian business entity (Enhet)
type Company struct {
	OrganizationNumber string            `json:"organisasjonsnummer"`
	Name               string            `json:"navn"`
	OrganizationForm   *OrganizationForm `json:"organisasjonsform,omitempty"`
	RegistrationDate   string            `json:"registreringsdatoEnhetsregisteret,omitempty"`
	FoundedDate        string            `json:"stiftelsedato,omitempty"`

	// Addresses
	PostalAddress   *Address `json:"postadresse,omitempty"`
	BusinessAddress *Address `json:"forretningsadresse,omitempty"`

	// Registration status
	RegisteredInVAT        bool `json:"registrertIMvaregisteret"`
	RegisteredInBusiness   bool `json:"registrertIForetaksregisteret"`
	RegisteredInFoundation bool `json:"registrertIStiftelsesregisteret"`
	RegisteredInVoluntary  bool `json:"registrertIFrivillighetsregisteret"`

	// Status flags
	Bankrupt               bool `json:"konkurs"`
	UnderLiquidation       bool `json:"underAvvikling"`
	UnderForcedLiquidation bool `json:"underTvangsavviklingEllerTvangsopplosning"`
	Deleted                bool `json:"slettedato,omitempty"`

	// Industry codes
	IndustryCode1 *IndustryCode `json:"naeringskode1,omitempty"`
	IndustryCode2 *IndustryCode `json:"naeringskode2,omitempty"`
	IndustryCode3 *IndustryCode `json:"naeringskode3,omitempty"`

	// Sector
	InstitutionalSector *SectorCode `json:"institusjonellSektorkode,omitempty"`

	// Other
	EmployeeCount          int    `json:"antallAnsatte"`
	HasRegisteredEmployees bool   `json:"harRegistrertAntallAnsatte"`
	Website                string `json:"hjemmeside,omitempty"`
	LanguageForm           string `json:"maalform,omitempty"`

	// Capital information (for AS, ASA)
	Capital *Capital `json:"kapital,omitempty"`

	// Links for HAL
	Links *Links `json:"_links,omitempty"`
}

// OrganizationForm represents the type of organization (AS, ENK, etc.)
type OrganizationForm struct {
	Code        string `json:"kode"`
	Description string `json:"beskrivelse"`
}

// Address represents a Norwegian address
type Address struct {
	Country            string   `json:"land,omitempty"`
	CountryCode        string   `json:"landkode,omitempty"`
	PostalCode         string   `json:"postnummer,omitempty"`
	PostalPlace        string   `json:"poststed,omitempty"`
	Municipality       string   `json:"kommune,omitempty"`
	MunicipalityNumber string   `json:"kommunenummer,omitempty"`
	AddressLines       []string `json:"adresse,omitempty"`
}

// IndustryCode represents a NACE industry classification code
type IndustryCode struct {
	Code        string `json:"kode"`
	Description string `json:"beskrivelse"`
}

// SectorCode represents an institutional sector code
type SectorCode struct {
	Code        string `json:"kode"`
	Description string `json:"beskrivelse"`
}

// Capital represents share capital information
type Capital struct {
	Amount     float64 `json:"belop,omitempty"`
	Currency   string  `json:"valuta,omitempty"`
	ShareCount int     `json:"antallAksjer,omitempty"`
	Type       string  `json:"type,omitempty"`
	Bound      float64 `json:"bundet,omitempty"`
	PaidIn     float64 `json:"innbetalt,omitempty"`
	FullyPaid  bool    `json:"fulltInnbetalt,omitempty"`
}

// Links contains HAL links
type Links struct {
	Self Link `json:"self,omitempty"`
}

// Link is a HAL link
type Link struct {
	Href string `json:"href"`
}

// SubUnit represents a branch office or production unit (Underenhet)
type SubUnit struct {
	OrganizationNumber       string            `json:"organisasjonsnummer"`
	Name                     string            `json:"navn"`
	ParentOrganizationNumber string            `json:"overordnetEnhet,omitempty"`
	OrganizationForm         *OrganizationForm `json:"organisasjonsform,omitempty"`
	RegistrationDate         string            `json:"registreringsdatoEnhetsregisteret,omitempty"`

	// Addresses
	PostalAddress   *Address `json:"postadresse,omitempty"`
	BusinessAddress *Address `json:"beliggenhetsadresse,omitempty"`

	// Industry codes
	IndustryCode1 *IndustryCode `json:"naeringskode1,omitempty"`
	IndustryCode2 *IndustryCode `json:"naeringskode2,omitempty"`
	IndustryCode3 *IndustryCode `json:"naeringskode3,omitempty"`

	// Other
	EmployeeCount int  `json:"antallAnsatte"`
	Deleted       bool `json:"nedleggelsesdato,omitempty"`

	Links *Links `json:"_links,omitempty"`
}

// RolesResponse represents the response from the roles endpoint
type RolesResponse struct {
	RoleGroups []RoleGroup `json:"rollegrupper"`
}

// RoleGroup represents a group of related roles
type RoleGroup struct {
	Type         RoleType `json:"type"`
	LastModified string   `json:"sistEndret,omitempty"`
	Roles        []Role   `json:"roller"`
}

// RoleType represents the type of role
type RoleType struct {
	Code        string `json:"kode"`
	Description string `json:"beskrivelse"`
}

// Role represents an individual role (board member, CEO, etc.)
type Role struct {
	Type                RoleType    `json:"type"`
	Person              *Person     `json:"person,omitempty"`
	Entity              *RoleEntity `json:"enhet,omitempty"`
	Resigned            bool        `json:"fratraadt"`
	Deregistered        bool        `json:"avregistrert,omitempty"`
	ResponsibilityShare string      `json:"ansvarsandel,omitempty"`
	ElectedBy           *ElectedBy  `json:"valgtAv,omitempty"`
}

// Person represents a natural person in a role
type Person struct {
	Name      PersonName `json:"navn"`
	BirthDate string     `json:"fodselsdato,omitempty"`
	Deceased  bool       `json:"erDod"`
}

// PersonName represents a person's name
type PersonName struct {
	FirstName  string `json:"fornavn,omitempty"`
	MiddleName string `json:"mellomnavn,omitempty"`
	LastName   string `json:"etternavn,omitempty"`
}

// FullName returns the complete name as a single string
func (n PersonName) FullName() string {
	name := n.FirstName
	if n.MiddleName != "" {
		if name != "" {
			name += " "
		}
		name += n.MiddleName
	}
	if n.LastName != "" {
		if name != "" {
			name += " "
		}
		name += n.LastName
	}
	return name
}

// ElectedBy represents who elected/appointed a role holder
type ElectedBy struct {
	Code        string `json:"kode"`
	Description string `json:"beskrivelse"`
	Links       *Links `json:"_links,omitempty"`
}

// RoleEntity represents an organization in a role (for corporate roles)
type RoleEntity struct {
	OrganizationNumber string            `json:"organisasjonsnummer"`
	OrganizationForm   *OrganizationForm `json:"organisasjonsform,omitempty"`
	Name               []string          `json:"navn,omitempty"`
	Deleted            bool              `json:"slettedato,omitempty"`
}

// SearchResponse represents the paginated search response
type SearchResponse struct {
	Embedded struct {
		Companies []Company `json:"enheter"`
	} `json:"_embedded"`
	Links struct {
		Self  Link `json:"self"`
		First Link `json:"first,omitempty"`
		Prev  Link `json:"prev,omitempty"`
		Next  Link `json:"next,omitempty"`
		Last  Link `json:"last,omitempty"`
	} `json:"_links"`
	Page PageInfo `json:"page"`
}

// SubUnitSearchResponse represents the paginated sub-unit search response
type SubUnitSearchResponse struct {
	Embedded struct {
		SubUnits []SubUnit `json:"underenheter"`
	} `json:"_embedded"`
	Links struct {
		Self  Link `json:"self"`
		First Link `json:"first,omitempty"`
		Next  Link `json:"next,omitempty"`
		Last  Link `json:"last,omitempty"`
	} `json:"_links"`
	Page PageInfo `json:"page"`
}

// PageInfo contains pagination metadata
type PageInfo struct {
	Size          int `json:"size"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
	Number        int `json:"number"`
}

// UpdatesResponse represents updates feed response
type UpdatesResponse struct {
	Embedded struct {
		Updates []UpdateEntry `json:"oppdaterteEnheter"`
	} `json:"_embedded"`
	Links struct {
		Self Link `json:"self"`
		Next Link `json:"next,omitempty"`
	} `json:"_links"`
}

// UpdateEntry represents a single update in the feed
type UpdateEntry struct {
	UpdateID           int       `json:"oppdateringsid"`
	OrganizationNumber string    `json:"organisasjonsnummer"`
	UpdatedAt          time.Time `json:"dato"`
	ChangeType         string    `json:"endringstype,omitempty"`
}

// APIError represents an error from the Brønnøysund API
type APIError struct {
	Timestamp string `json:"timestamp"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

func (e APIError) String() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Error
}
