package customers

import (
	"reflect"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

// ContactPersonInput is what staff fill in for a contact person.
type ContactPersonInput struct {
	FirstName string
	LastName  string
	Role      string
	Email     string
	Phone     string
}

func (in ContactPersonInput) applyTo(p *customersDomain.ContactPerson) {
	p.FirstName, p.LastName, p.Role, p.Email, p.Phone = in.FirstName, in.LastName, in.Role, in.Email, in.Phone
}

// changedContactFields names the contact person fields that differ --
// names only, the values are personal data.
func changedContactFields(before, after customersDomain.ContactPerson) []string {
	fields := []struct {
		name          string
		before, after any
	}{
		{"email", before.Email, after.Email},
		{"first_name", before.FirstName, after.FirstName},
		{"last_name", before.LastName, after.LastName},
		{"phone", before.Phone, after.Phone},
		{"role", before.Role, after.Role},
	}
	var out []string
	for _, f := range fields {
		if !reflect.DeepEqual(f.before, f.after) {
			out = append(out, f.name)
		}
	}
	return out
}
