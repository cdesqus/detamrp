package production

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"order-stock/backend/internal/rbac"
)

type DashboardRepository interface {
	Dashboard(context.Context, Actor, DashboardFilter) (ProductionDashboard, error)
	OrderReport(context.Context, Actor, uuid.UUID) (OrderReport, error)
}

func RegisterDashboardRoutes(router *gin.Engine, s DashboardRepository, authenticator Authenticator) {
	dashboard := func(c *gin.Context) (ProductionDashboard, bool) {
		filter, e := ParseDashboardFilter(c.Query("from"), c.Query("to"), time.Now())
		if e != nil {
			respondError(c, e)
			return ProductionDashboard{}, false
		}
		v, e := s.Dashboard(c, executionActor(c), filter)
		if e != nil {
			respondError(c, e)
			return ProductionDashboard{}, false
		}
		return v, true
	}
	router.GET("/production-dashboard", executionAuth(authenticator), rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		v, ok := dashboard(c)
		if !ok {
			return
		}
		// Quantities are for the floor; money needs the reporting permission.
		permissions, _ := c.Get(rbac.ContextPermissionsKey)
		granted, _ := permissions.([]string)
		if !rbac.Allows(granted, "production.report") {
			v = v.withoutCosts()
		}
		c.JSON(200, v)
	})

	g := router.Group("/production-reports", executionAuth(authenticator), rbac.RequirePermissions("production.report"))
	g.GET("/dashboard.xlsx", func(c *gin.Context) {
		v, ok := dashboard(c)
		if !ok {
			return
		}
		document, e := RenderDashboardXLSX(v)
		if e != nil {
			respondError(c, e)
			return
		}
		attach(c, "production-dashboard-"+v.Filter.From+"-"+v.Filter.To+".xlsx",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", document)
	})
	g.GET("/dashboard.pdf", func(c *gin.Context) {
		v, ok := dashboard(c)
		if !ok {
			return
		}
		document, e := RenderDashboardPDF(v)
		if e != nil {
			respondError(c, e)
			return
		}
		attach(c, "production-dashboard-"+v.Filter.From+"-"+v.Filter.To+".pdf", "application/pdf", document)
	})
	order := func(c *gin.Context) (OrderReport, bool) {
		id, ok := planID(c)
		if !ok {
			return OrderReport{}, false
		}
		r, e := s.OrderReport(c, executionActor(c), id)
		if e != nil {
			respondError(c, e)
			return OrderReport{}, false
		}
		return r, true
	}
	g.GET("/orders/:id/execution.xlsx", func(c *gin.Context) {
		r, ok := order(c)
		if !ok {
			return
		}
		document, e := RenderOrderReportXLSX(r)
		if e != nil {
			respondError(c, e)
			return
		}
		attach(c, r.Order.OrderNumber+"-execution.xlsx",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", document)
	})
	g.GET("/orders/:id/execution.pdf", func(c *gin.Context) {
		r, ok := order(c)
		if !ok {
			return
		}
		document, e := RenderOrderReportPDF(r)
		if e != nil {
			respondError(c, e)
			return
		}
		attach(c, r.Order.OrderNumber+"-execution.pdf", "application/pdf", document)
	})
}

func attach(c *gin.Context, filename, contentType string, document []byte) {
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(200, contentType, document)
}
