package customers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ContactPersonUseCases are the contact person use cases' Execute methods.
type ContactPersonUseCases struct {
	List   func(ctx context.Context, customerID string) ([]customersDomain.ContactPerson, error)
	Add    func(ctx context.Context, customerID string, in appcustomers.ContactPersonInput) (*customersDomain.ContactPerson, error)
	Update func(ctx context.Context, customerID, id string, in appcustomers.ContactPersonInput) (*customersDomain.ContactPerson, error)
	Remove func(ctx context.Context, customerID, id string) error
}

// WithContactPersons mounts the contact person routes' use cases.
func (h *Handler) WithContactPersons(uc ContactPersonUseCases) *Handler {
	h.contactPersons = uc
	return h
}

func (h *Handler) ListContactPersons(c *gin.Context, id gencustomers.Id) {
	persons, err := h.contactPersons.List(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]gencustomers.ContactPerson, len(persons))
	for i, p := range persons {
		out[i] = toGenContactPerson(p)
	}
	c.JSON(http.StatusOK, gencustomers.ContactPersonListResponse{Data: &out})
}

func (h *Handler) AddContactPerson(c *gin.Context, id gencustomers.Id) {
	var req gencustomers.ContactPersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	p, err := h.contactPersons.Add(c.Request.Context(), id, toContactPersonInput(req))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenContactPerson(*p)
	c.JSON(http.StatusCreated, gencustomers.ContactPersonResponse{Data: &resp})
}

func (h *Handler) UpdateContactPerson(c *gin.Context, id gencustomers.Id, personID gencustomers.PersonId) {
	var req gencustomers.ContactPersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	p, err := h.contactPersons.Update(c.Request.Context(), id, personID, toContactPersonInput(req))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenContactPerson(*p)
	c.JSON(http.StatusOK, gencustomers.ContactPersonResponse{Data: &resp})
}

func (h *Handler) RemoveContactPerson(c *gin.Context, id gencustomers.Id, personID gencustomers.PersonId) {
	if err := mapError(c, h.contactPersons.Remove(c.Request.Context(), id, personID)); err != nil {
		return
	}
	c.JSON(http.StatusOK, gencustomers.EmptyResponse{})
}

func toContactPersonInput(req gencustomers.ContactPersonRequest) appcustomers.ContactPersonInput {
	return appcustomers.ContactPersonInput{
		FirstName: strPtrOr(req.FirstName), LastName: strPtrOr(req.LastName), Role: strPtrOr(req.Role),
		Email: strPtrOr(req.Email), Phone: strPtrOr(req.Phone),
	}
}

func toContactPersonInputs(reqs *[]gencustomers.ContactPersonRequest) []appcustomers.ContactPersonInput {
	if reqs == nil {
		return nil
	}
	out := make([]appcustomers.ContactPersonInput, len(*reqs))
	for i, r := range *reqs {
		out[i] = toContactPersonInput(r)
	}
	return out
}

func toGenContactPerson(p customersDomain.ContactPerson) gencustomers.ContactPerson {
	return gencustomers.ContactPerson{
		Id: p.ID, CustomerId: p.CustomerID, FirstName: p.FirstName, LastName: p.LastName,
		Role: p.Role, Email: p.Email, Phone: p.Phone, CreatedAt: p.CreatedAt,
	}
}
