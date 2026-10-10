package messagecenter

import mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"

// ListTriggers is every event a rule can fire on, with the variables its
// templates can use -- the catalog the template editor shows. Static, no
// tenant data.
func ListTriggers() []mcDomain.Trigger {
	return mcDomain.Triggers
}
