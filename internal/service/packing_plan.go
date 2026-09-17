package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
)

type PackingPlanService struct {
	repo      *repository.PackingPlanRepo
	batchRepo *repository.BatchRepo
}

func NewPackingPlanService(repo *repository.PackingPlanRepo, batchRepo *repository.BatchRepo) *PackingPlanService {
	return &PackingPlanService{repo: repo, batchRepo: batchRepo}
}

func (s *PackingPlanService) Create(batchID int64, req *model.UpsertPackingPlanRequest) (*model.PackingPlanView, error) {
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, newBizError(http.StatusNotFound, "batch not found")
	}
	if _, err := s.repo.GetByBatch(batchID); err == nil {
		return nil, newBizError(http.StatusConflict, "该批次已有分级与装箱方案,如需修改请整体替换")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	total, grades, err := s.buildGrades(batch, req)
	if err != nil {
		return nil, err
	}
	plan := &model.PackingPlan{
		BatchID:      batchID,
		TotalYieldKg: total,
		Status:       model.PackingPlanDraft,
		Note:         req.Note,
		CreatedBy:    req.CreatedBy,
	}
	if err := s.repo.Create(plan, grades); err != nil {
		return nil, err
	}
	return s.GetByBatch(batchID)
}

// Replace 整体替换草稿方案的分级明细与装箱规格
func (s *PackingPlanService) Replace(batchID int64, req *model.UpsertPackingPlanRequest) (*model.PackingPlanView, error) {
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, newBizError(http.StatusNotFound, "batch not found")
	}
	plan, err := s.repo.GetByBatch(batchID)
	if err != nil {
		return nil, newBizError(http.StatusNotFound, "该批次还没有分级与装箱方案")
	}
	if plan.Status == model.PackingPlanConfirmed {
		return nil, newBizError(http.StatusConflict, "方案已确认,不可修改")
	}

	total, grades, err := s.buildGrades(batch, req)
	if err != nil {
		return nil, err
	}
	plan.TotalYieldKg = total
	plan.Note = req.Note
	plan.CreatedBy = req.CreatedBy
	if err := s.repo.Replace(plan, grades); err != nil {
		return nil, err
	}
	return s.GetByBatch(batchID)
}

func (s *PackingPlanService) GetByBatch(batchID int64) (*model.PackingPlanView, error) {
	plan, err := s.repo.GetByBatch(batchID)
	if err != nil {
		return nil, newBizError(http.StatusNotFound, "该批次还没有分级与装箱方案")
	}
	grades, err := s.repo.ListGrades(plan.ID)
	if err != nil {
		return nil, err
	}
	return buildView(plan, grades), nil
}

// Confirm 确认方案:各级合计必须等于总产量,有余量的级别必须已登记处理办法与处理人
func (s *PackingPlanService) Confirm(batchID int64) (*model.PackingPlanView, error) {
	plan, err := s.repo.GetByBatch(batchID)
	if err != nil {
		return nil, newBizError(http.StatusNotFound, "该批次还没有分级与装箱方案")
	}
	if plan.Status == model.PackingPlanConfirmed {
		return nil, newBizError(http.StatusConflict, "方案已确认,请勿重复操作")
	}
	grades, err := s.repo.ListGrades(plan.ID)
	if err != nil {
		return nil, err
	}
	view := buildView(plan, grades)

	// 对账:各级加起来要等于总产量,对不上的差额点名
	if math.Abs(view.UnallocatedKg) >= 0.005 {
		if view.UnallocatedKg > 0 {
			return nil, newBizError(http.StatusBadRequest, fmt.Sprintf(
				"各级合计 %.2fkg,比总产量 %.2fkg 少 %.2fkg,尚有产量未分级",
				view.AllocatedKg, plan.TotalYieldKg, view.UnallocatedKg))
		}
		return nil, newBizError(http.StatusBadRequest, fmt.Sprintf(
			"各级合计 %.2fkg,超出总产量 %.2fkg 共 %.2fkg",
			view.AllocatedKg, plan.TotalYieldKg, -view.UnallocatedKg))
	}

	// 装不满一箱的余量必须已有处理办法与处理人
	for _, gv := range view.Grades {
		if gv.RemainderKg > 0 && (gv.RemainderHandling == nil || gv.RemainderHandler == nil) {
			return nil, newBizError(http.StatusBadRequest, fmt.Sprintf(
				"级别「%s」余量 %.2fkg 尚未登记处理办法与处理人", gv.GradeName, gv.RemainderKg))
		}
	}

	if err := s.repo.UpdateStatus(plan.ID, model.PackingPlanConfirmed); err != nil {
		return nil, err
	}
	return s.GetByBatch(batchID)
}

// RecordRemainder 登记某级别的余量处理:处理办法 + 处理人,处理时间由服务端记录
func (s *PackingPlanService) RecordRemainder(batchID, gradeID int64, req *model.RemainderHandlingRequest) (*model.PackingPlanGradeView, error) {
	plan, err := s.repo.GetByBatch(batchID)
	if err != nil {
		return nil, newBizError(http.StatusNotFound, "该批次还没有分级与装箱方案")
	}
	grade, err := s.repo.GetGradeByID(gradeID)
	if err != nil || grade.PlanID != plan.ID {
		return nil, newBizError(http.StatusNotFound, "grade not found")
	}
	if _, remainder := splitBoxes(grade.WeightKg, grade.KgPerBox); remainder <= 0 {
		return nil, newBizError(http.StatusBadRequest, fmt.Sprintf("级别「%s」没有余量,无需登记处理", grade.GradeName))
	}
	if err := s.repo.RecordRemainderHandling(gradeID, req.Handling, req.Handler); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetGradeByID(gradeID)
	if err != nil {
		return nil, err
	}
	boxes, remainder := splitBoxes(updated.WeightKg, updated.KgPerBox)
	return &model.PackingPlanGradeView{
		PackingPlanGrade: *updated,
		BoxesNeeded:      boxes,
		RemainderKg:      remainder,
	}, nil
}

// buildGrades 校验请求并构造分级明细;总产量缺省取批次预计产量
func (s *PackingPlanService) buildGrades(batch *model.CropBatch, req *model.UpsertPackingPlanRequest) (float64, []model.PackingPlanGrade, error) {
	total := req.TotalYieldKg
	if total == 0 {
		total = batch.ExpectedYieldKg
	}
	if total <= 0 {
		return 0, nil, newBizError(http.StatusBadRequest, "批次预计产量为 0,请显式指定 total_yield_kg")
	}
	if len(req.Grades) == 0 {
		return 0, nil, newBizError(http.StatusBadRequest, "至少需要一个分级")
	}

	grades := make([]model.PackingPlanGrade, 0, len(req.Grades))
	for _, g := range req.Grades {
		if g.WeightKg <= 0 {
			return 0, nil, newBizError(http.StatusBadRequest, fmt.Sprintf("级别「%s」产量必须大于 0", g.GradeName))
		}
		if g.KgPerBox <= 0 {
			return 0, nil, newBizError(http.StatusBadRequest, fmt.Sprintf("级别「%s」每箱装量必须大于 0", g.GradeName))
		}
		grades = append(grades, model.PackingPlanGrade{
			GradeName:   g.GradeName,
			SizeSpec:    g.SizeSpec,
			QualityDesc: g.QualityDesc,
			WeightKg:    g.WeightKg,
			BoxSpec:     g.BoxSpec,
			KgPerBox:    g.KgPerBox,
		})
	}
	return total, grades, nil
}

// buildView 计算每级整箱数与余量,并做合计对账
func buildView(plan *model.PackingPlan, grades []model.PackingPlanGrade) *model.PackingPlanView {
	view := &model.PackingPlanView{
		PackingPlan: *plan,
		Grades:      make([]model.PackingPlanGradeView, 0, len(grades)),
	}
	for _, g := range grades {
		boxes, remainder := splitBoxes(g.WeightKg, g.KgPerBox)
		view.Grades = append(view.Grades, model.PackingPlanGradeView{
			PackingPlanGrade: g,
			BoxesNeeded:      boxes,
			RemainderKg:      remainder,
		})
		view.AllocatedKg += g.WeightKg
		view.TotalBoxes += boxes
		view.TotalRemainderKg += remainder
	}
	view.AllocatedKg = round2(view.AllocatedKg)
	view.TotalRemainderKg = round2(view.TotalRemainderKg)
	view.UnallocatedKg = round2(plan.TotalYieldKg - view.AllocatedKg)
	return view
}

// splitBoxes 按每箱装量算出整箱数与装不满一箱的余量
func splitBoxes(weightKg, kgPerBox float64) (int, float64) {
	boxes := int(math.Floor(weightKg/kgPerBox + 1e-9))
	remainder := round2(weightKg - float64(boxes)*kgPerBox)
	if remainder <= 0 {
		remainder = 0
	}
	return boxes, remainder
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
