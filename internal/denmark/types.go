// Package denmark provides a client for the Danish CVR (Central Business Register) API.
// It enables lookups of companies in the Danish business registry via cvrapi.dk.
package denmark

// Company represents a Danish business entity from CVR
type Company struct {
	CVR          int    `json:"vat"`             // 8-digit CVR number
	Name         string `json:"name"`            // Company name
	Address      string `json:"address"`         // Street address
	Zipcode      string `json:"zipcode"`         // Postal code
	City         string `json:"city"`            // City name
	Country      string `json:"country"`         // Country (usually "DK")
	Phone        string `json:"phone"`           // Phone number
	Email        string `json:"email"`           // Email address
	StartDate    string `json:"startdate"`       // Registration date (YYYY-MM-DD)
	EndDate      string `json:"enddate"`         // Dissolution date if applicable
	Employees    int    `json:"employees"`       // Employee count
	CompanyType  string `json:"companydesc"`     // Company type description (ApS, A/S, etc.)
	IndustryCode int    `json:"industrycode"`    // NACE industry code
	IndustryDesc string `json:"industrydesc"`    // Industry description
	CreditStart  string `json:"creditstartdate"` // Credit registration date
	CreditEnd    bool   `json:"creditbankrupt"`  // Credit bankruptcy status
	Status       string `json:"status"`          // Company status
	// Production units (P-numbers)
	ProductionUnits []ProductionUnit `json:"productionunits,omitempty"`
	// Owners (if available)
	Owners []Owner `json:"owners,omitempty"`
}

// ProductionUnit represents a Danish P-number (production unit)
type ProductionUnit struct {
	PNumber      int64  `json:"pno"`          // P-number
	Main         bool   `json:"main"`         // Is main production unit
	Name         string `json:"name"`         // Unit name
	Address      string `json:"address"`      // Street address
	Zipcode      string `json:"zipcode"`      // Postal code
	City         string `json:"city"`         // City name
	Phone        string `json:"phone"`        // Phone number
	Email        string `json:"email"`        // Email address
	StartDate    string `json:"startdate"`    // Start date
	EndDate      string `json:"enddate"`      // End date if closed
	Employees    any    `json:"employees"`    // Employee count (can be int, string, or null)
	IndustryCode int    `json:"industrycode"` // Industry code
	IndustryDesc string `json:"industrydesc"` // Industry description
}

// Owner represents a company owner
type Owner struct {
	Name string `json:"name"` // Owner name
}

// APIError represents an error response from the CVR API
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	T       int    `json:"t"` // Error code
}

func (e APIError) String() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Error
}
