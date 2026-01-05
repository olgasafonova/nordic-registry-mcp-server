package finland

// CompanySearchResponse is the top-level response from PRH API
type CompanySearchResponse struct {
	TotalResults int       `json:"totalResults"`
	Companies    []Company `json:"companies"`
}

// Company represents a Finnish company from PRH/YTJ
type Company struct {
	BusinessID          BusinessID         `json:"businessId"`
	EUID                *EUID              `json:"euId,omitempty"`
	Names               []CompanyName      `json:"names,omitempty"`
	MainBusinessLine    *BusinessLine      `json:"mainBusinessLine,omitempty"`
	Website             *Website           `json:"website,omitempty"`
	CompanyForms        []CompanyForm      `json:"companyForms,omitempty"`
	CompanySituations   []CompanySituation `json:"companySituations,omitempty"`
	RegisteredEntries   []RegisteredEntry  `json:"registeredEntries,omitempty"`
	Addresses           []Address          `json:"addresses,omitempty"`
	TradeRegisterStatus string             `json:"tradeRegisterStatus,omitempty"`
	Status              string             `json:"status,omitempty"`
	RegistrationDate    string             `json:"registrationDate,omitempty"`
	EndDate             string             `json:"endDate,omitempty"`
	LastModified        string             `json:"lastModified,omitempty"`
}

// BusinessID contains the Finnish business identifier (Y-tunnus)
type BusinessID struct {
	Value            string `json:"value"`
	RegistrationDate string `json:"registrationDate,omitempty"`
	Source           string `json:"source,omitempty"`
}

// EUID is the European unique identifier
type EUID struct {
	Value  string `json:"value"`
	Source string `json:"source,omitempty"`
}

// CompanyName represents a company name with type and validity period
type CompanyName struct {
	Name             string `json:"name"`
	Type             string `json:"type"` // 1=current, 2=previous, 3=auxiliary
	RegistrationDate string `json:"registrationDate,omitempty"`
	EndDate          string `json:"endDate,omitempty"`
	Version          int    `json:"version,omitempty"`
	Source           string `json:"source,omitempty"`
}

// BusinessLine represents the main business activity (TOL 2008)
type BusinessLine struct {
	Type             string        `json:"type"` // TOL 2008 code
	Descriptions     []Description `json:"descriptions,omitempty"`
	TypeCodeSet      string        `json:"typeCodeSet,omitempty"`
	RegistrationDate string        `json:"registrationDate,omitempty"`
	Source           string        `json:"source,omitempty"`
}

// Description is a multilingual description
type Description struct {
	LanguageCode string `json:"languageCode"` // 1=fi, 2=sv, 3=en
	Description  string `json:"description"`
}

// Website contains company website info
type Website struct {
	URL              string `json:"url"`
	RegistrationDate string `json:"registrationDate,omitempty"`
	Source           string `json:"source,omitempty"`
}

// CompanyForm represents the legal form of the company
type CompanyForm struct {
	Type             string        `json:"type"` // Form code
	Descriptions     []Description `json:"descriptions,omitempty"`
	RegistrationDate string        `json:"registrationDate,omitempty"`
	EndDate          string        `json:"endDate,omitempty"`
	Version          int           `json:"version,omitempty"`
	Source           string        `json:"source,omitempty"`
}

// CompanySituation represents company status (active, liquidation, bankruptcy)
type CompanySituation struct {
	Type             string `json:"type"` // SANE, SELTILA, KONK
	RegistrationDate string `json:"registrationDate,omitempty"`
	EndDate          string `json:"endDate,omitempty"`
	Source           string `json:"source,omitempty"`
}

// RegisteredEntry represents a registration status entry
type RegisteredEntry struct {
	RegistrationStatus    string        `json:"registrationStatus,omitempty"`
	Type                  string        `json:"type,omitempty"`
	TypeDescriptions      []Description `json:"typeDescriptions,omitempty"`
	Register              string        `json:"register,omitempty"`
	RegisterDescriptions  []Description `json:"registerDescriptions,omitempty"`
	Authority             string        `json:"authority,omitempty"`
	AuthorityDescriptions []Description `json:"authorityDescriptions,omitempty"`
	RegistrationDate      string        `json:"registrationDate,omitempty"`
	EndDate               string        `json:"endDate,omitempty"`
	Source                string        `json:"source,omitempty"`
}

// Address represents a company address
type Address struct {
	Type              int          `json:"type"` // 1=street, 2=postal
	Street            string       `json:"street,omitempty"`
	PostCode          string       `json:"postCode,omitempty"`
	PostOffices       []PostOffice `json:"postOffices,omitempty"`
	PostOfficeBox     string       `json:"postOfficeBox,omitempty"`
	BuildingNumber    string       `json:"buildingNumber,omitempty"`
	Entrance          string       `json:"entrance,omitempty"`
	ApartmentNumber   string       `json:"apartmentNumber,omitempty"`
	ApartmentIDSuffix string       `json:"apartmentIdSuffix,omitempty"`
	CO                string       `json:"co,omitempty"`
	Country           string       `json:"country,omitempty"`
	FreeAddressLine   string       `json:"freeAddressLine,omitempty"`
	RegistrationDate  string       `json:"registrationDate,omitempty"`
	Source            string       `json:"source,omitempty"`
}

// PostOffice contains city info with language variants
type PostOffice struct {
	City             string `json:"city"`
	LanguageCode     string `json:"languageCode"` // 1=fi, 2=sv
	MunicipalityCode string `json:"municipalityCode,omitempty"`
}
