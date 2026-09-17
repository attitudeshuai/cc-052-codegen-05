package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type GradingHandler struct {
	svc *service.GradingService
}

func NewGradingHandler(svc *service.GradingService) *GradingHandler {
	return &GradingHandler{svc: svc}
}

// Create POST /api/v1/batches/:id/grading-plan
func (h *GradingHandler) Create(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}
	var req model.CreateGradingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	detail, err := h.svc.CreatePlan(batchID, &req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Created(c, detail)
}

// GetByBatch GET /api/v1/batches/:id/grading-plan
func (h *GradingHandler) GetByBatch(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}
	detail, err := h.svc.GetPlanByBatch(batchID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, detail)
}

// Update PUT /api/v1/batches/:id/grading-plan
func (h *GradingHandler) Update(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}
	var req model.UpdateGradingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	detail, err := h.svc.UpdatePlan(batchID, &req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, detail)
}

// Confirm POST /api/v1/batches/:id/grading-plan/confirm
func (h *GradingHandler) Confirm(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}
	detail, err := h.svc.ConfirmPlan(batchID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, detail)
}

// HandleLeftover POST /api/v1/grading-plans/:id/leftovers/:leftoverId/handle
func (h *GradingHandler) HandleLeftover(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}
	leftoverID, err := strconv.ParseInt(c.Param("leftoverId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid leftover id")
		return
	}
	var req model.HandleLeftoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	view, err := h.svc.HandleLeftover(planID, leftoverID, &req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Success(c, view)
}

// CreatePackSpec POST /api/v1/pack-specs
func (h *GradingHandler) CreatePackSpec(c *gin.Context) {
	var req model.CreatePackSpecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	spec, err := h.svc.CreatePackSpec(&req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	response.Created(c, spec)
}

// ListPackSpecs GET /api/v1/pack-specs
func (h *GradingHandler) ListPackSpecs(c *gin.Context) {
	list, err := h.svc.ListPackSpecs()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, list)
}

// handleErr 把业务错误映射成对应的 HTTP 状态码
func (h *GradingHandler) handleErr(c *gin.Context, err error) {
	var balErr *service.BalanceError
	switch {
	case errors.As(err, &balErr):
		// 总量对不上：点名差异
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": balErr.Error(),
			"data": gin.H{
				"total_yield_kg":  balErr.TotalKg,
				"graded_sum_kg":   balErr.SumKg,
				"balance_diff_kg": balErr.DiffKg,
			},
		})
	case errors.Is(err, service.ErrBatchNotFound),
		errors.Is(err, service.ErrGradingPlanNotFound),
		errors.Is(err, service.ErrLeftoverNotFound),
		errors.Is(err, service.ErrPackSpecNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, service.ErrGradingPlanExists),
		errors.Is(err, service.ErrGradingPlanNotDraft),
		errors.Is(err, service.ErrGradingPlanNotConfirmed),
		errors.Is(err, service.ErrLeftoverAlreadyHandled),
		errors.Is(err, service.ErrPackSpecExists):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(c, err.Error())
	default:
		response.InternalError(c, err.Error())
	}
}
