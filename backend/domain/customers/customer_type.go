package customers

import "strings"

// Type is whether a customer is a company (B2B: named by its company, with
// contact persons) or a private individual (named by the person).
type Type string

const (
	TypePrivate Type = "private"
	TypeCompany Type = "company"
)

// Valid reports whether t is a known type.
func (t Type) Valid() bool { return t == TypePrivate || t == TypeCompany }

// TypeOrPrivate is the customer's type; one never set is private.
func (c Customer) TypeOrPrivate() Type {
	if c.Type == "" {
		return TypePrivate
	}
	return c.Type
}

// Validate checks what every customer must satisfy before it is saved: a
// known type, a company name for a company, and something to tell it apart
// by (Identifiable). Email is never required.
func (c Customer) Validate() error {
	t := c.TypeOrPrivate()
	if !t.Valid() {
		return ErrInvalidCustomerType
	}
	if t == TypeCompany && strings.TrimSpace(c.CompanyName) == "" {
		return ErrCompanyNameRequired
	}
	if !c.Identifiable() {
		return ErrCustomerUnidentified
	}
	return nil
}
