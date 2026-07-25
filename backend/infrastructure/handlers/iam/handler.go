package iam

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
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

// ── Request / Response contracts ─────────────────────────────────────────────

type createGroupRequest struct {
	Name        string `json:"name"        binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}

type updateGroupRequest struct {
	Name        string `json:"name"        binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}

type addMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type assignPermissionRequest struct {
	Rule string `json:"rule" binding:"required"`
}

type revokePermissionRequest struct {
	Rule string `json:"rule" binding:"required"`
}

type groupResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type membersResponse struct {
	Members []string `json:"members"`
}

type permissionsResponse struct {
	Permissions []string `json:"permissions"`
}

func toGroupResponse(g *iamDomain.Group) groupResponse {
	return groupResponse{
		ID:          g.ID,
		TenantID:    g.TenantID,
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
	}
}

// RegisterHandler mounts routes for an already-constructed Handler.
func RegisterHandler(rg *gin.RouterGroup, h *Handler) {
	adminOnly := middleware.RequireRole(
		middleware.RoleCustomerOwner,
		middleware.RoleCustomerAdmin,
		middleware.RoleSuperAdmin,
	)

	iam := rg.Group("/iam")
	{
		groups := iam.Group("/groups", adminOnly)
		{
			groups.POST("", h.createGroupHandler)
			groups.GET("", h.listGroupsHandler)
			groups.GET("/:id", h.getGroupHandler)
			groups.PATCH("/:id", h.updateGroupHandler)
			groups.DELETE("/:id", h.deleteGroupHandler)

			groups.POST("/:id/members", h.addMemberHandler)
			groups.DELETE("/:id/members/:user_id", h.removeMemberHandler)
			groups.GET("/:id/members", h.listMembersHandler)

			groups.POST("/:id/permissions", h.assignPermissionHandler)
			groups.DELETE("/:id/permissions/:rule", h.revokePermissionHandler)
			groups.GET("/:id/permissions", h.listPermissionsHandler)
		}

		iam.GET("/users/:user_id/permissions", adminOnly, h.userPermissionsHandler)
	}
}

// Register is the production wiring: creates repos + use cases, then calls RegisterHandler.
func Register(rg *gin.RouterGroup, db *gorm.DB) {
	groupRepo := repositories.NewGroupRepo(db)
	userGroupRepo := repositories.NewUserGroupRepo(db)
	gpRepo := repositories.NewGroupPermissionRepo(db)

	h := NewHandler(
		appiam.NewCreateGroupUseCase(db, groupRepo),
		appiam.NewUpdateGroupUseCase(db, groupRepo),
		appiam.NewDeleteGroupUseCase(db, groupRepo),
		appiam.NewGetGroupUseCase(db, groupRepo),
		appiam.NewListGroupsUseCase(db, groupRepo),
		appiam.NewAddUserToGroupUseCase(db, groupRepo, userGroupRepo),
		appiam.NewRemoveUserFromGroupUseCase(db, groupRepo, userGroupRepo),
		appiam.NewListGroupMembersUseCase(db, groupRepo, userGroupRepo),
		appiam.NewAssignPermissionToGroupUseCase(db, groupRepo, gpRepo),
		appiam.NewRevokePermissionFromGroupUseCase(db, groupRepo, gpRepo),
		appiam.NewListGroupPermissionsUseCase(db, groupRepo, gpRepo),
		appiam.NewGetUserEffectivePermissionsUseCase(db, gpRepo),
	)
	RegisterHandler(rg, h)
}

// ── Route handlers ────────────────────────────────────────────────────────────

func (h *Handler) createGroupHandler(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	group, err := h.createGroup.Execute(c.Request.Context(), appiam.CreateGroupInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toGroupResponse(group), "error": nil})
}

func (h *Handler) updateGroupHandler(c *gin.Context) {
	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	group, err := h.updateGroup.Execute(c.Request.Context(), appiam.UpdateGroupInput{
		ID:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGroupResponse(group))
}

func (h *Handler) deleteGroupHandler(c *gin.Context) {
	err := h.deleteGroup.Execute(c.Request.Context(), c.Param("id"))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) getGroupHandler(c *gin.Context) {
	group, err := h.getGroup.Execute(c.Request.Context(), c.Param("id"))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGroupResponse(group))
}

func (h *Handler) listGroupsHandler(c *gin.Context) {
	groups, err := h.listGroups.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]groupResponse, len(groups))
	for i, g := range groups {
		g := g
		resp[i] = toGroupResponse(&g)
	}
	middleware.OK(c, resp)
}

func (h *Handler) addMemberHandler(c *gin.Context) {
	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	err := h.addMember.Execute(c.Request.Context(), appiam.AddUserToGroupInput{
		GroupID: c.Param("id"),
		UserID:  req.UserID,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) removeMemberHandler(c *gin.Context) {
	err := h.removeMember.Execute(c.Request.Context(), appiam.RemoveUserFromGroupInput{
		GroupID: c.Param("id"),
		UserID:  c.Param("user_id"),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) listMembersHandler(c *gin.Context) {
	members, err := h.listMembers.Execute(c.Request.Context(), c.Param("id"))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, membersResponse{Members: members})
}

func (h *Handler) assignPermissionHandler(c *gin.Context) {
	var req assignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	err := h.assignPermission.Execute(c.Request.Context(), appiam.AssignPermissionInput{
		GroupID: c.Param("id"),
		Rule:    req.Rule,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) revokePermissionHandler(c *gin.Context) {
	err := h.revokePermission.Execute(c.Request.Context(), appiam.RevokePermissionInput{
		GroupID: c.Param("id"),
		Rule:    c.Param("rule"),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) listPermissionsHandler(c *gin.Context) {
	rules, err := h.listPermissions.Execute(c.Request.Context(), c.Param("id"))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, permissionsResponse{Permissions: rulesToStrings(rules)})
}

func (h *Handler) userPermissionsHandler(c *gin.Context) {
	rules, err := h.userPermissions.Execute(c.Request.Context(), c.Param("user_id"))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, permissionsResponse{Permissions: rulesToStrings(rules)})
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
