package iam

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genia "github.com/adirkuhn/commerceos/api/iam"
	appiam "github.com/adirkuhn/commerceos/application/iam"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type createGroupUC interface {
	Execute(ctx context.Context, in appiam.CreateGroupInput) (*iamDomain.Group, error)
}
type updateGroupUC interface {
	Execute(ctx context.Context, in appiam.UpdateGroupInput) (*iamDomain.Group, error)
}
type deleteGroupUC interface {
	Execute(ctx context.Context, id string) error
}
type getGroupUC interface {
	Execute(ctx context.Context, id string) (*iamDomain.Group, error)
}
type listGroupsUC interface {
	Execute(ctx context.Context) ([]iamDomain.Group, error)
}
type addUserToGroupUC interface {
	Execute(ctx context.Context, in appiam.AddUserToGroupInput) error
}
type removeUserFromGroupUC interface {
	Execute(ctx context.Context, in appiam.RemoveUserFromGroupInput) error
}
type listGroupMembersUC interface {
	Execute(ctx context.Context, groupID string) ([]string, error)
}
type assignPermissionUC interface {
	Execute(ctx context.Context, in appiam.AssignPermissionInput) error
}
type revokePermissionUC interface {
	Execute(ctx context.Context, in appiam.RevokePermissionInput) error
}
type listGroupPermissionsUC interface {
	Execute(ctx context.Context, groupID string) ([]iamDomain.Rule, error)
}
type userEffectivePermissionsUC interface {
	Execute(ctx context.Context, userID string) ([]iamDomain.Rule, error)
}
type listUserActivitiesUC interface {
	Execute(ctx context.Context, userID string) ([]omsDomain.SalesOrderActivity, error)
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func toGenGroup(g *iamDomain.Group) genia.Group {
	return genia.Group{
		Id:          g.ID,
		TenantId:    g.TenantID,
		Name:        g.Name,
		Description: g.Description,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func rulesToStrings(rules []iamDomain.Rule) []string {
	ss := make([]string, len(rules))
	for i, r := range rules {
		ss[i] = r.String()
	}
	return ss
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	createGroup      createGroupUC
	updateGroup      updateGroupUC
	deleteGroup      deleteGroupUC
	getGroup         getGroupUC
	listGroups       listGroupsUC
	addMember        addUserToGroupUC
	removeMember     removeUserFromGroupUC
	listMembers      listGroupMembersUC
	assignPermission assignPermissionUC
	revokePermission revokePermissionUC
	listPermissions  listGroupPermissionsUC
	userPermissions  userEffectivePermissionsUC
	userActivities   listUserActivitiesUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	createGroup createGroupUC,
	updateGroup updateGroupUC,
	deleteGroup deleteGroupUC,
	getGroup getGroupUC,
	listGroups listGroupsUC,
	addMember addUserToGroupUC,
	removeMember removeUserFromGroupUC,
	listMembers listGroupMembersUC,
	assignPermission assignPermissionUC,
	revokePermission revokePermissionUC,
	listPermissions listGroupPermissionsUC,
	userPermissions userEffectivePermissionsUC,
	userActivities listUserActivitiesUC,
) *Handler {
	return &Handler{
		createGroup:      createGroup,
		updateGroup:      updateGroup,
		deleteGroup:      deleteGroup,
		getGroup:         getGroup,
		listGroups:       listGroups,
		addMember:        addMember,
		removeMember:     removeMember,
		listMembers:      listMembers,
		assignPermission: assignPermission,
		revokePermission: revokePermission,
		listPermissions:  listPermissions,
		userPermissions:  userPermissions,
		userActivities:   userActivities,
	}
}

var _ genia.ServerInterface = (*Handler)(nil)

// RegisterHandler mounts routes for an already-constructed Handler, guarded
// by role middleware. Spec paths already include the /iam prefix (matching
// the api/pms and api/integrations convention of paths relative to the
// /api/v1 mount point), so this registers directly on rg rather than a
// nested /iam subgroup — nesting would double the prefix. The role guard is
// applied per-route via GinServerOptions.Middlewares instead of Group(),
// since RegisterHandlers owns route registration.
func RegisterHandler(rg gin.IRouter, h *Handler) {
	adminOnly := middleware.RequireRole(
		middleware.RoleCustomerOwner,
		middleware.RoleCustomerAdmin,
		middleware.RoleSuperAdmin,
	)
	genia.RegisterHandlersWithOptions(rg, h, genia.GinServerOptions{
		Middlewares: []genia.MiddlewareFunc{genia.MiddlewareFunc(adminOnly)},
	})
}

// Register is the production wiring: creates repos + use cases, then calls RegisterHandler.
func Register(rg gin.IRouter, db *gorm.DB) {
	groupRepo := repositories.NewGroupRepo(db)
	userGroupRepo := repositories.NewUserGroupRepo(db)
	gpRepo := repositories.NewGroupPermissionRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)

	h := NewHandler(
		appiam.NewCreateGroupUseCase(db, groupRepo, outboxRepo),
		appiam.NewUpdateGroupUseCase(db, groupRepo, outboxRepo),
		appiam.NewDeleteGroupUseCase(db, groupRepo, outboxRepo),
		appiam.NewGetGroupUseCase(db, groupRepo),
		appiam.NewListGroupsUseCase(db, groupRepo),
		appiam.NewAddUserToGroupUseCase(db, groupRepo, userGroupRepo, outboxRepo),
		appiam.NewRemoveUserFromGroupUseCase(db, groupRepo, userGroupRepo, outboxRepo),
		appiam.NewListGroupMembersUseCase(db, groupRepo, userGroupRepo),
		appiam.NewAssignPermissionToGroupUseCase(db, groupRepo, gpRepo, outboxRepo),
		appiam.NewRevokePermissionFromGroupUseCase(db, groupRepo, gpRepo, outboxRepo),
		appiam.NewListGroupPermissionsUseCase(db, groupRepo, gpRepo),
		appiam.NewGetUserEffectivePermissionsUseCase(db, gpRepo),
		appoms.NewListUserActivitiesUseCase(db, repositories.NewSalesOrderActivityRepo(db)),
	)
	RegisterHandler(rg, h)
}

// ── Route handlers ────────────────────────────────────────────────────────────

func (h *Handler) CreateGroup(c *gin.Context) {
	var req genia.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	var description string
	if req.Description != nil {
		description = *req.Description
	}
	group, err := h.createGroup.Execute(c.Request.Context(), appiam.CreateGroupInput{
		Name:        req.Name,
		Description: description,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toGenGroup(group), "error": nil})
}

func (h *Handler) UpdateGroup(c *gin.Context, id genia.Id) {
	var req genia.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	var description string
	if req.Description != nil {
		description = *req.Description
	}
	group, err := h.updateGroup.Execute(c.Request.Context(), appiam.UpdateGroupInput{
		ID:          id,
		Name:        req.Name,
		Description: description,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenGroup(group))
}

func (h *Handler) DeleteGroup(c *gin.Context, id genia.Id) {
	err := h.deleteGroup.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) GetGroup(c *gin.Context, id genia.Id) {
	group, err := h.getGroup.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenGroup(group))
}

func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.listGroups.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genia.Group, len(groups))
	for i, g := range groups {
		g := g
		resp[i] = toGenGroup(&g)
	}
	middleware.OK(c, resp)
}

func (h *Handler) AddGroupMember(c *gin.Context, id genia.Id) {
	var req genia.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.UserId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "user_id is required")
		return
	}
	err := h.addMember.Execute(c.Request.Context(), appiam.AddUserToGroupInput{
		GroupID: id,
		UserID:  req.UserId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) RemoveGroupMember(c *gin.Context, id genia.Id, userId genia.UserId) {
	err := h.removeMember.Execute(c.Request.Context(), appiam.RemoveUserFromGroupInput{
		GroupID: id,
		UserID:  userId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListGroupMembers(c *gin.Context, id genia.Id) {
	members, err := h.listMembers.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"members": members})
}

func (h *Handler) AssignGroupPermission(c *gin.Context, id genia.Id) {
	var req genia.AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Rule == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "rule is required")
		return
	}
	err := h.assignPermission.Execute(c.Request.Context(), appiam.AssignPermissionInput{
		GroupID: id,
		Rule:    req.Rule,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) RevokeGroupPermission(c *gin.Context, id genia.Id, rule genia.Rule) {
	err := h.revokePermission.Execute(c.Request.Context(), appiam.RevokePermissionInput{
		GroupID: id,
		Rule:    rule,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListGroupPermissions(c *gin.Context, id genia.Id) {
	rules, err := h.listPermissions.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"permissions": rulesToStrings(rules)})
}

func (h *Handler) ListUserPermissions(c *gin.Context, userId genia.UserId) {
	rules, err := h.userPermissions.Execute(c.Request.Context(), userId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"permissions": rulesToStrings(rules)})
}

func (h *Handler) ListUserActivities(c *gin.Context, userId genia.UserId) {
	activities, err := h.userActivities.Execute(c.Request.Context(), userId)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genia.UserActivity, len(activities))
	for i, a := range activities {
		var metadata map[string]interface{}
		if err := json.Unmarshal(a.Metadata, &metadata); err != nil && len(a.Metadata) > 0 {
			log.Printf("iamHandler: activity %s: corrupt metadata, omitting: %v", a.ID, err)
		}
		out[i] = genia.UserActivity{
			Id: a.ID, SalesOrderId: a.SalesOrderID, UserId: a.UserID,
			ActivityType: a.ActivityType, Summary: a.Summary, Metadata: metadata, CreatedAt: a.CreatedAt,
		}
	}
	middleware.OK(c, gin.H{"activities": out})
}

// mapError translates domain errors to HTTP responses. Returns non-nil if it wrote a response.
func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, iamDomain.ErrGroupNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, iamDomain.ErrGroupNameTaken):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, iamDomain.ErrAlreadyMember):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, iamDomain.ErrNotMember):
		middleware.Fail(c, http.StatusUnprocessableEntity, "UNPROCESSABLE", err.Error())
	case errors.Is(err, iamDomain.ErrPermissionInvalid):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, iamDomain.ErrPermissionNotAssigned):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
