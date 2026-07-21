package purchaseorder

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

// Authenticator supplies the authenticated tenant user for purchase-order routes.
type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorContextKey = "purchase_order_actor"

// RegisterRoutes registers tenant-scoped purchase-order and approval endpoints.
func RegisterRoutes(router *gin.Engine, service *Service, authenticator Authenticator) {
	orders := router.Group("/purchase-orders", authenticate(authenticator))
	orders.GET("", rbac.RequirePermissions("po.view"), func(c *gin.Context) {
		query, ok := listQuery(c)
		if !ok {
			return
		}
		items, total, err := service.List(c.Request.Context(), actorFrom(c), query)
		if writeHTTPError(c, err) {
			return
		}
		if items == nil {
			items = []Order{}
		}
		projected := make([]gin.H, 0, len(items))
		for _, item := range items {
			projected = append(projected, projectOrder(item, hasPermission(c, "po.price.view")))
		}
		c.JSON(http.StatusOK, gin.H{"items": projected, "total": total})
	})
	orders.GET("/:id", rbac.RequirePermissions("po.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		order, err := service.Get(c.Request.Context(), actorFrom(c), id)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusOK, projectOrder(order, hasPermission(c, "po.price.view")))
	})
	orders.POST("", rbac.RequirePermissions("po.create"), func(c *gin.Context) {
		input, ok := orderInput(c)
		if !ok {
			return
		}
		order, err := service.Create(c.Request.Context(), actorFrom(c), input)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusCreated, projectOrder(order, hasPermission(c, "po.price.view")))
	})
	orders.PUT("/:id", rbac.RequirePermissions("po.edit_draft"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		input, ok := orderInput(c)
		if !ok {
			return
		}
		order, err := service.Update(c.Request.Context(), actorFrom(c), id, input)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusOK, projectOrder(order, hasPermission(c, "po.price.view")))
	})
	orders.POST("/:id/submit", rbac.RequirePermissions("po.submit"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		order, err := service.Submit(c.Request.Context(), actorFrom(c), id)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusOK, projectOrder(order, hasPermission(c, "po.price.view")))
	})
	orders.POST("/:id/cancel", rbac.RequirePermissions("po.edit_draft"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		order, err := service.Cancel(c.Request.Context(), actorFrom(c), id)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusOK, projectOrder(order, hasPermission(c, "po.price.view")))
	})

	approvals := router.Group("/purchase-order-approvals", authenticate(authenticator))
	approvals.GET("", rbac.RequirePermissions("po.approve"), func(c *gin.Context) {
		query, ok := listQuery(c)
		if !ok {
			return
		}
		items, total, err := service.ListApprovals(c.Request.Context(), actorFrom(c), query)
		if writeHTTPError(c, err) {
			return
		}
		if items == nil {
			items = []Approval{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
	approvals.POST("/:id/approve", rbac.RequirePermissions("po.approve"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		input, ok := decisionInput(c)
		if !ok {
			return
		}
		approval, err := service.Approve(c.Request.Context(), actorFrom(c), id, input)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusOK, approval)
	})
	approvals.POST("/:id/reject", rbac.RequirePermissions("po.reject"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		input, ok := decisionInput(c)
		if !ok {
			return
		}
		approval, err := service.Reject(c.Request.Context(), actorFrom(c), id, input)
		if writeHTTPError(c, err) {
			return
		}
		c.JSON(http.StatusOK, approval)
	})
}

func authenticate(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		user, err := authenticator.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Set(actorContextKey, Actor{TenantID: user.TenantID, UserID: user.ID, DisplayName: user.DisplayName})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
}

func actorFrom(c *gin.Context) Actor {
	actor, _ := c.Get(actorContextKey)
	return actor.(Actor)
}

func hasPermission(c *gin.Context, permission string) bool {
	value, exists := c.Get(rbac.ContextPermissionsKey)
	permissions, valid := value.([]string)
	return exists && valid && rbac.Allows(permissions, permission)
}

// projectOrder is the public read model. Commercial fields are added only
// after explicit authorization so omitted prices cannot be mistaken for zero.
func projectOrder(order Order, includePrices bool) gin.H {
	lines := make([]gin.H, 0, len(order.Lines))
	for _, line := range order.Lines {
		projected := gin.H{
			"id": line.ID, "tenantId": line.TenantID, "purchaseOrderId": line.PurchaseOrderID,
			"rawMaterialId": line.RawMaterialID, "rawMaterialCode": line.RawMaterialCode, "rawMaterialName": line.RawMaterialName,
			"baseUnitId": line.BaseUnitID, "baseUnitCode": line.BaseUnitCode, "qtyPerKanbanSnapshot": line.QtyPerKanbanSnapshot,
			"totalKanban": line.TotalKanban, "orderedBaseQty": line.OrderedBaseQty, "sortPosition": line.SortPosition,
			"createdBy": line.CreatedBy, "createdAt": line.CreatedAt, "updatedBy": line.UpdatedBy, "updatedAt": line.UpdatedAt,
		}
		if includePrices {
			projected["unitPriceSnapshot"] = line.UnitPriceSnapshot
			projected["lineTotal"] = line.LineTotal
		}
		lines = append(lines, projected)
	}
	projected := gin.H{
		"id": order.ID, "tenantId": order.TenantID, "poNumber": order.PONumber,
		"supplierId": order.SupplierID, "supplierName": order.SupplierName,
		"orderDate": order.OrderDate, "expectedDeliveryDate": order.ExpectedDeliveryDate,
		"currency": order.Currency, "notes": order.Notes, "status": order.Status, "version": order.Version,
		"sagePurchaseOrderNumber":      order.SagePurchaseOrderNumber,
		"submittedApproverUserId":      order.SubmittedApproverUserID,
		"submittedApproverDisplayName": order.SubmittedApproverDisplayName,
		"submittedApproverEmail":       order.SubmittedApproverEmail,
		"createdBy":                    order.CreatedBy, "createdAt": order.CreatedAt, "updatedBy": order.UpdatedBy, "updatedAt": order.UpdatedAt,
		"lines": lines,
	}
	if includePrices {
		projected["totalAmount"] = order.TotalAmount
	}
	return projected
}

func routeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return uuid.Nil, false
	}
	return id, true
}

func listQuery(c *gin.Context) (ListQuery, bool) {
	query := ListQuery{Search: c.Query("search")}
	if raw := c.Query("supplierId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			invalidFilter(c, "supplierId", "Invalid supplier")
			return query, false
		}
		query.SupplierID = id
	}
	if raw := c.Query("status"); raw != "" {
		query.Status = Status(raw)
		if !isSupportedStatus(query.Status) {
			invalidFilter(c, "status", "Unsupported purchase order status")
			return query, false
		}
	}
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			invalidFilter(c, "limit", "Must be an integer")
			return query, false
		}
		query.Limit = value
	}
	if raw := c.Query("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			invalidFilter(c, "offset", "Must be an integer")
			return query, false
		}
		query.Offset = value
	}
	return query, true
}

func orderInput(c *gin.Context) (OrderInput, bool) {
	var request orderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		invalidJSON(c)
		return OrderInput{}, false
	}
	input := OrderInput{
		SupplierID: request.SupplierID,
		Currency:   request.Currency,
		Notes:      request.Notes,
		Lines:      request.Lines,
	}
	fields := FieldErrors{}
	var err error
	if input.OrderDate, err = parseOrderDate(request.OrderDate); err != nil {
		fields["orderDate"] = "Order Date must be a valid date"
	}
	if input.ExpectedDeliveryDate, err = parseOrderDate(request.ExpectedDeliveryDate); err != nil {
		fields["expectedDeliveryDate"] = "Expected Delivery Date must be a valid date"
	}
	if len(fields) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "Please correct the highlighted fields", "fields": fields})
		return OrderInput{}, false
	}
	return input, true
}

type orderRequest struct {
	SupplierID           uuid.UUID   `json:"supplierId"`
	OrderDate            string      `json:"orderDate"`
	ExpectedDeliveryDate string      `json:"expectedDeliveryDate"`
	Currency             string      `json:"currency"`
	Notes                string      `json:"notes"`
	Lines                []LineInput `json:"lines"`
}

func parseOrderDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
}

func isSupportedStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusPendingApproval, StatusApproved, StatusRejected, StatusCancelled:
		return true
	default:
		return false
	}
}

func decisionInput(c *gin.Context) (DecisionInput, bool) {
	var input DecisionInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			invalidJSON(c)
			return DecisionInput{}, false
		}
	}
	// An empty body is valid for approval. Rejection validation remains in the
	// service so a missing reason has the standard field-error response.
	return input, true
}

func invalidJSON(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
}

func invalidFilter(c *gin.Context, field, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter", "fields": FieldErrors{field: message}})
}

func writeHTTPError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var validation ValidationError
	var conflict ConflictError
	var missing NotFoundError
	switch {
	case errors.As(err, &validation):
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "Please correct the highlighted fields", "fields": validation.Fields})
	case errors.As(err, &conflict):
		c.JSON(http.StatusConflict, gin.H{"error": "conflict", "message": "Purchase order conflicts with its current state", "fields": conflict.Fields})
	case errors.As(err, &missing):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": missing.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Request could not be completed"})
	}
	return true
}
