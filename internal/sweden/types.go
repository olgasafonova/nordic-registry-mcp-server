// Package sweden provides a client for the Swedish business registry (Bolagsverket).
// It uses the VärdefullaDatamängder (High-Value Datasets) API which requires OAuth2
// authentication.
package sweden

import "strings"

// Dataproducent indicates the source of information.
type Dataproducent string

const (
	DataproducentBolagsverket Dataproducent = "Bolagsverket"
	DataproducentSCB          Dataproducent = "SCB"
)

// FelTyp categorizes error types from the API.
type FelTyp string

const (
	FelTypOrganisationFinnsEj       FelTyp = "ORGANISATION_FINNS_EJ" //nolint:misspell // Swedish API value
	FelTypOgiltigBegaran            FelTyp = "OGILTIG_BEGARAN"
	FelTypOtillgangligUppgiftskalla FelTyp = "OTILLGANGLIG_UPPGIFTSKALLA"
	FelTypTimeout                   FelTyp = "TIMEOUT"
)

// JaNej represents a yes/no value in Swedish.
type JaNej string

const (
	JaNejJA  JaNej = "JA"
	JaNejNEJ JaNej = "NEJ"
)

// Fel represents an error from a data source.
type Fel struct {
	Typ            FelTyp `json:"typ"`
	FelBeskrivning string `json:"felBeskrivning,omitempty"`
}

// KodKlartext represents a code with its human-readable description.
type KodKlartext struct {
	Kod      string `json:"kod"`
	Klartext string `json:"klartext"`
}

// Identitetsbeteckning is the unique identification of an organization.
// Can be organisationsnummer (10 digits), personnummer (12 digits),
// samordningsnummer (12 digits), or GD-nummer (10 digits starting with 302).
type Identitetsbeteckning struct {
	Identitetsbeteckning string       `json:"identitetsbeteckning"`
	Typ                  *KodKlartext `json:"typ,omitempty"`
}

// OrganisationsnamnObjekt represents a single business name.
type OrganisationsnamnObjekt struct {
	Namn                                       string       `json:"namn,omitempty"`
	Organisationsnamntyp                       *KodKlartext `json:"organisationsnamntyp,omitempty"`
	Registreringsdatum                         string       `json:"registreringsdatum,omitempty"`
	VerksamhetsbeskrivningSarskiltForetagsnamn string       `json:"verksamhetsbeskrivningSarskiltForetagsnamn,omitempty"`
}

// Organisationsnamn contains the business names associated with the organization.
type Organisationsnamn struct {
	OrganisationsnamnLista []OrganisationsnamnObjekt `json:"organisationsnamnLista,omitempty"`
	Dataproducent          Dataproducent             `json:"dataproducent,omitempty"`
	Fel                    *Fel                      `json:"fel,omitempty"`
}

// Reklamsparr indicates if the organization is blocked from receiving advertisements.
type Reklamsparr struct {
	Kod           JaNej         `json:"kod,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// Organisationsform indicates the administrative form (e.g., AB, E, HB).
type Organisationsform struct {
	Kod           string        `json:"kod,omitempty"`
	Klartext      string        `json:"klartext,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// AvregistreradOrganisation contains the date an organization was removed from the register.
type AvregistreradOrganisation struct {
	Avregistreringsdatum string        `json:"avregistreringsdatum,omitempty"`
	Dataproducent        Dataproducent `json:"dataproducent,omitempty"`
	Fel                  *Fel          `json:"fel,omitempty"`
}

// Avregistreringsorsak contains the reason an organization was removed from the register.
type Avregistreringsorsak struct {
	Kod           string        `json:"kod,omitempty"`
	Klartext      string        `json:"klartext,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// PagaendeAvvecklingsEllerOmstruktureringsforfarandeObjekt represents an ongoing
// liquidation or restructuring procedure.
type PagaendeAvvecklingsEllerOmstruktureringsforfarandeObjekt struct {
	Kod       string `json:"kod,omitempty"`
	Klartext  string `json:"klartext,omitempty"`
	FromDatum string `json:"fromDatum,omitempty"`
}

// PagaendeAvvecklingsEllerOmstruktureringsforfarande indicates ongoing removal or
// restructuring procedures.
type PagaendeAvvecklingsEllerOmstruktureringsforfarande struct {
	PagaendeAvvecklingsEllerOmstruktureringsforfarandeLista []PagaendeAvvecklingsEllerOmstruktureringsforfarandeObjekt `json:"pagaendeAvvecklingsEllerOmstruktureringsforfarandeLista,omitempty"`
	Dataproducent                                           Dataproducent                                              `json:"dataproducent,omitempty"`
	Fel                                                     *Fel                                                       `json:"fel,omitempty"`
}

// JuridiskForm is the legal form registered at the Swedish Tax Agency.
type JuridiskForm struct {
	Kod           string        `json:"kod,omitempty"`
	Klartext      string        `json:"klartext,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// VerksamOrganisation indicates if the organization is active.
type VerksamOrganisation struct {
	Kod           JaNej         `json:"kod,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// Organisationsdatum contains the date an organization was registered.
type Organisationsdatum struct {
	Registreringsdatum string        `json:"registreringsdatum,omitempty"`
	InfortHosScb       string        `json:"infortHosScb,omitempty"`
	Dataproducent      Dataproducent `json:"dataproducent,omitempty"`
	Fel                *Fel          `json:"fel,omitempty"`
}

// Verksamhetsbeskrivning contains the description of business activities.
type Verksamhetsbeskrivning struct {
	Beskrivning   string        `json:"beskrivning,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// NaringsgrenOrganisation contains the SNI codes (industry classification).
type NaringsgrenOrganisation struct {
	SNI           []KodKlartext `json:"sni,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// Postadress contains postal address details.
type Postadress struct {
	Postnummer       string `json:"postnummer,omitempty"`
	Utdelningsadress string `json:"utdelningsadress,omitempty"`
	Land             string `json:"land,omitempty"`
	CoAdress         string `json:"coAdress,omitempty"`
	Postort          string `json:"postort,omitempty"`
}

// PostadressOrganisation contains the registered postal address.
type PostadressOrganisation struct {
	Postadress    *Postadress   `json:"postadress,omitempty"`
	Dataproducent Dataproducent `json:"dataproducent,omitempty"`
	Fel           *Fel          `json:"fel,omitempty"`
}

// Organisation contains all company information from the API.
//
//nolint:misspell // Swedish API uses "organisation"
type Organisation struct {
	Organisationsidentitet                             *Identitetsbeteckning                               `json:"organisationsidentitet,omitempty"`
	Namnskyddslopnummer                                *int                                                `json:"namnskyddslopnummer,omitempty"`
	Organisationsnamn                                  *Organisationsnamn                                  `json:"organisationsnamn,omitempty"`
	Registreringsland                                  *KodKlartext                                        `json:"registreringsland,omitempty"`
	Reklamsparr                                        *Reklamsparr                                        `json:"reklamsparr,omitempty"`
	Organisationsform                                  *Organisationsform                                  `json:"organisationsform,omitempty"`
	AvregistreradOrganisation                          *AvregistreradOrganisation                          `json:"avregistreradOrganisation,omitempty"`
	Avregistreringsorsak                               *Avregistreringsorsak                               `json:"avregistreringsorsak,omitempty"`
	PagaendeAvvecklingsEllerOmstruktureringsforfarande *PagaendeAvvecklingsEllerOmstruktureringsforfarande `json:"pagaendeAvvecklingsEllerOmstruktureringsforfarande,omitempty"`
	JuridiskForm                                       *JuridiskForm                                       `json:"juridiskForm,omitempty"`
	VerksamOrganisation                                *VerksamOrganisation                                `json:"verksamOrganisation,omitempty"`
	Organisationsdatum                                 *Organisationsdatum                                 `json:"organisationsdatum,omitempty"`
	Verksamhetsbeskrivning                             *Verksamhetsbeskrivning                             `json:"verksamhetsbeskrivning,omitempty"`
	NaringsgrenOrganisation                            *NaringsgrenOrganisation                            `json:"naringsgrenOrganisation,omitempty"`
	PostadressOrganisation                             *PostadressOrganisation                             `json:"postadressOrganisation,omitempty"`
}

// OrganisationerBegaran is the request body for the /organisationer endpoint.
type OrganisationerBegaran struct {
	Identitetsbeteckning string `json:"identitetsbeteckning"`
}

// OrganisationerSvar is the response from the /organisationer endpoint.
//
//nolint:misspell // Swedish API uses "organisationer"
type OrganisationerSvar struct {
	Organisationer []Organisation `json:"organisationer,omitempty"` //nolint:misspell
}

// DokumentlistaBegaran is the request body for the /dokumentlista endpoint.
type DokumentlistaBegaran struct {
	Identitetsbeteckning string `json:"identitetsbeteckning"`
}

// Dokument represents an annual report in the document list.
type Dokument struct {
	DokumentID             string `json:"dokumentId,omitempty"`
	Filformat              string `json:"filformat,omitempty"`
	RapporteringsperiodTom string `json:"rapporteringsperiodTom,omitempty"`
	Registreringstidpunkt  string `json:"registreringstidpunkt,omitempty"`
}

// DokumentlistaSvar is the response from the /dokumentlista endpoint.
type DokumentlistaSvar struct {
	Dokument []Dokument `json:"dokument,omitempty"`
}

// APIError represents an error response from the API (RFC 7807 format).
type APIError struct {
	Type      string `json:"type"`
	Instance  string `json:"instance"`
	Status    int    `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Title     string `json:"title"`
	Detail    string `json:"detail,omitempty"`
}

// TokenResponse represents the OAuth2 token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// GetName returns the primary business name for the organization.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetName() string {
	if o.Organisationsnamn == nil || len(o.Organisationsnamn.OrganisationsnamnLista) == 0 {
		return ""
	}
	// Return the first name (typically the main business name)
	return o.Organisationsnamn.OrganisationsnamnLista[0].Namn
}

// GetOrgNumber returns the organization number.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetOrgNumber() string {
	if o.Organisationsidentitet == nil {
		return ""
	}
	return o.Organisationsidentitet.Identitetsbeteckning
}

// GetFormCode returns the organization form code (e.g., AB, E, HB).
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetFormCode() string {
	if o.Organisationsform == nil {
		return ""
	}
	return o.Organisationsform.Kod
}

// GetFormDescription returns the organization form description.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetFormDescription() string {
	if o.Organisationsform == nil {
		return ""
	}
	return o.Organisationsform.Klartext
}

// IsActive returns true if the organization is currently active.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) IsActive() bool {
	if o.VerksamOrganisation != nil && o.VerksamOrganisation.Kod == JaNejJA {
		return true
	}
	// Also check if deregistered
	if o.AvregistreradOrganisation != nil && o.AvregistreradOrganisation.Avregistreringsdatum != "" {
		return false
	}
	return true // Default to active if no explicit status
}

// GetRegistrationDate returns the registration date.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetRegistrationDate() string {
	if o.Organisationsdatum == nil {
		return ""
	}
	return o.Organisationsdatum.Registreringsdatum
}

// GetAddress returns a formatted address string.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetAddress() string {
	if o.PostadressOrganisation == nil || o.PostadressOrganisation.Postadress == nil {
		return ""
	}
	addr := o.PostadressOrganisation.Postadress
	var parts []string
	if addr.CoAdress != "" {
		parts = append(parts, addr.CoAdress)
	}
	if addr.Utdelningsadress != "" {
		parts = append(parts, addr.Utdelningsadress)
	}
	if addr.Postnummer != "" || addr.Postort != "" {
		parts = append(parts, addr.Postnummer+" "+addr.Postort)
	}
	if addr.Land != "" && addr.Land != "Sverige" {
		parts = append(parts, addr.Land)
	}
	return strings.Join(parts, ", ")
}

// GetBusinessDescription returns the business activity description.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetBusinessDescription() string {
	if o.Verksamhetsbeskrivning == nil {
		return ""
	}
	return o.Verksamhetsbeskrivning.Beskrivning
}

// GetSNICodes returns the SNI industry codes.
//
//nolint:misspell // Organisation is the Swedish API type name
func (o *Organisation) GetSNICodes() []KodKlartext {
	if o.NaringsgrenOrganisation == nil {
		return nil
	}
	return o.NaringsgrenOrganisation.SNI
}
