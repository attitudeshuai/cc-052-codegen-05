package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/response"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PackingPlanHandler struct {
	svc *service.PackingPlanService
}

func NewPackingPlanHandler(svc *service.PackingPlanService) *PackingPlanHandler {
	return &PackingPlanHandler{svc: svc}
}

func (h *PackingPlanHandler) respondErr(c *gin.Context, err error) {
	var be *service.BizError
	if errors.As(err, &be) {
		response.Error(c, be.Status, be.Message)
		return
	}
	response.InternalError(c, err.Error())
}

func (h *PackingPlanHandler) Create(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	var req model.UpsertPackingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	view, err := h.svc.Create(batchID, &req)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Created(c, view)
}

func (h *PackingPlanHandler) Replace(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	var req model.UpsertPackingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	view, err := h.svc.Replace(batchID, &req)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Success(c, view)
}

func (h *PackingPlanHandler) GetByBatch(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	view, err := h.svc.GetByBatch(batchID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Success(c, view)
}

func (h *PackingPlanHandler) Confirm(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	view, err := h.svc.Confirm(batchID)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Success(c, view)
}

func (h *PackingPlanHandler) RecordRemainder(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}
	gradeID, err := strconv.ParseInt(c.Param("gradeId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid grade id")
		return
	}

	var req model.RemainderHandlingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	view, err := h.svc.RecordRemainder(batchID, gradeID, &req)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	response.Success(c, view)
}
