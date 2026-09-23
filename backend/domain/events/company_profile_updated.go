package events

// CompanyProfileUpdated is raised when a tenant saves its company profile
// (legal name, VAT, address... -- the invoice seller block).
type CompanyProfileUpdated struct {
	Base
	TenantID string `json:"tenant_id"`
}

func (CompanyProfileUpdated) EventName() string { return "settings.company_profile.updated" }
