package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrBatchNotFound           = errors.New("批次不存在")
	ErrGradingPlanExists       = errors.New("该批次已存在分级装箱方案")
	ErrGradingPlanNotFound     = errors.New("分级装箱方案不存在")
	ErrGradingPlanNotDraft     = errors.New("方案已确认，不能再修改")
	ErrGradingPlanNotConfirmed = errors.New("方案未确认，不能登记余量处理")
	ErrLeftoverNotFound        = errors.New("余量记录不存在")
	ErrLeftoverAlreadyHandled  = errors.New("该余量已处理过")
	ErrPackSpecNotFound        = errors.New("包装规格不存在")
	ErrPackSpecExists          = errors.New("同名包装规格已存在")
	ErrInvalidInput            = errors.New("参数不合法")
)

// BalanceError 分级合计与总产量对不上时返回，点名差多少
type BalanceError struct {
	TotalKg float64
	SumKg   float64
	DiffKg  float64
}

func (e *BalanceError) Error() string {
	if e.DiffKg > 0 {
		return fmt.Sprintf("各级别合计 %.2f kg，比总产量 %.2f kg 少 %.2f kg，有产量未分级", e.SumKg, e.TotalKg, e.DiffKg)
	}
	return fmt.Sprintf("各级别合计 %.2f kg，超出总产量 %.2f kg 共 %.2f kg", e.SumKg, e.TotalKg, -e.DiffKg)
}

type GradingService struct {
	repo         *repository.GradingRepo
	packSpecRepo *repository.PackSpecRepo
	batchRepo    *repository.BatchRepo
}

func NewGradingService(repo *repository.GradingRepo, packSpecRepo *repository.PackSpecRepo, batchRepo *repository.BatchRepo) *GradingService {
	return &GradingService{repo: repo, packSpecRepo: packSpecRepo, batchRepo: batchRepo}
}

// CreatePlan 为一批货创建分级装箱方案，自动算箱数并生成余量记录。
func (s *GradingService) CreatePlan(batchID int64, req *model.CreateGradingPlanRequest) (*model.GradingPlanDetail, error) {
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, ErrBatchNotFound
	}

	if _, err := s.repo.GetPlanByBatch(batchID); err == nil {
		return nil, ErrGradingPlanExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	total := req.TotalYieldKg
	if total <= 0 {
		total = batch.ExpectedYieldKg // 缺省按这一批的预计产量
	}

	grades, leftoverKgs, err := s.buildGrades(total, req.Grades)
	if err != nil {
		return nil, err
	}

	plan := &model.GradingPlan{
		BatchID:      batchID,
		TotalYieldKg: total,
		Status:       model.GradingPlanDraft,
		Remark:       req.Remark,
		CreatedBy:    req.CreatedBy,
	}
	if err := s.repo.CreatePlan(plan, grades, leftoverKgs); err != nil {
		return nil, err
	}
	return s.GetPlan(plan.ID)
}

// UpdatePlan 整体替换级别明细并重新计算余量（仅草稿状态可改）。
func (s *GradingService) UpdatePlan(batchID int64, req *model.UpdateGradingPlanRequest) (*model.GradingPlanDetail, error) {
	plan, err := s.repo.GetPlanByBatch(batchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGradingPlanNotFound
		}
		return nil, err
	}
	if plan.Status != model.GradingPlanDraft {
		return nil, ErrGradingPlanNotDraft
	}

	total := req.TotalYieldKg
	if total <= 0 {
		total = plan.TotalYieldKg
	}

	grades, leftoverKgs, err := s.buildGrades(total, req.Grades)
	if err != nil {
		return nil, err
	}

	plan.TotalYieldKg = total
	plan.Remark = req.Remark
	if err := s.repo.ReplaceGrades(plan, grades, leftoverKgs); err != nil {
		return nil, err
	}
	return s.GetPlan(plan.ID)
}

// ConfirmPlan 确认方案：各级别加起来必须等于总产量，对不上则点名差异并拒绝。
func (s *GradingService) ConfirmPlan(batchID int64) (*model.GradingPlanDetail, error) {
	plan, err := s.repo.GetPlanByBatch(batchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGradingPlanNotFound
		}
		return nil, err
	}
	if plan.Status != model.GradingPlanDraft {
		return nil, ErrGradingPlanNotDraft
	}

	grades, err := s.repo.ListGrades(plan.ID)
	if err != nil {
		return nil, err
	}
	yields := make([]float64, len(grades))
	for i, g := range grades {
		yields[i] = g.YieldKg
	}
	if diff := BalanceDiffKg(plan.TotalYieldKg, yields); diff != 0 {
		sum := plan.TotalYieldKg - diff
		return nil, &BalanceError{TotalKg: plan.TotalYieldKg, SumKg: sum, DiffKg: diff}
	}

	if err := s.repo.UpdatePlanStatus(plan.ID, model.GradingPlanConfirmed); err != nil {
		return nil, err
	}
	return s.GetPlan(plan.ID)
}

// GetPlanByBatch 查询某批次的方案（含箱数、余量与平衡校验结果）。
func (s *GradingService) GetPlanByBatch(batchID int64) (*model.GradingPlanDetail, error) {
	plan, err := s.repo.GetPlanByBatch(batchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGradingPlanNotFound
		}
		return nil, err
	}
	return s.GetPlan(plan.ID)
}

// GetPlan 组装方案详情：逐级别算箱数与余量，并校验总量平衡。
func (s *GradingService) GetPlan(planID int64) (*model.GradingPlanDetail, error) {
	plan, err := s.repo.GetPlanByID(planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGradingPlanNotFound
		}
		return nil, err
	}

	grades, err := s.repo.ListGrades(plan.ID)
	if err != nil {
		return nil, err
	}
	leftovers, err := s.repo.ListLeftovers(plan.ID)
	if err != nil {
		return nil, err
	}
	specs, err := s.packSpecRepo.List()
	if err != nil {
		return nil, err
	}
	specByID := make(map[int64]model.PackSpec, len(specs))
	for _, sp := range specs {
		specByID[sp.ID] = sp
	}

	detail := &model.GradingPlanDetail{
		ID:            plan.ID,
		BatchID:       plan.BatchID,
		TotalYieldKg:  plan.TotalYieldKg,
		Status:        plan.Status,
		Remark:        plan.Remark,
		CreatedBy:     plan.CreatedBy,
		CreatedAt:     plan.CreatedAt,
		UpdatedAt:     plan.UpdatedAt,
		Grades:        make([]model.GradeResult, 0, len(grades)),
		Leftovers:     make([]model.LeftoverView, 0, len(leftovers)),
	}
	yields := make([]float64, len(grades))
	gradeNameByID := make(map[int64]string, len(grades))
	for i, g := range grades {
		yields[i] = g.YieldKg
		gradeNameByID[g.ID] = g.GradeName
		spec := specByID[g.PackSpecID]
		boxes, leftoverKg := PackCalc(g.YieldKg, spec.CapacityKg)
		detail.TotalBoxes += boxes
		detail.Grades = append(detail.Grades, model.GradeResult{
			ID:           g.ID,
			GradeName:    g.GradeName,
			SizeSpec:     g.SizeSpec,
			Appearance:   g.Appearance,
			YieldKg:      g.YieldKg,
			PackSpecID:   g.PackSpecID,
			PackSpecName: spec.Name,
			CapacityKg:   spec.CapacityKg,
			FullBoxes:    boxes,
			LeftoverKg:   leftoverKg,
		})
	}
	for _, l := range leftovers {
		detail.Leftovers = append(detail.Leftovers, model.LeftoverView{
			ID:         l.ID,
			GradeID:    l.GradeID,
			GradeName:  gradeNameByID[l.GradeID],
			LeftoverKg: l.LeftoverKg,
			Action:     l.Action,
			Handler:    l.Handler,
			HandledAt:  l.HandledAt,
			Note:       l.Note,
		})
	}

	diff := BalanceDiffKg(plan.TotalYieldKg, yields)
	detail.BalanceDiffKg = diff
	detail.Balanced = diff == 0
	return detail, nil
}

// HandleLeftover 登记余量处理办法，处理人与时间留痕；全部处理完方案自动完结。
func (s *GradingService) HandleLeftover(planID, leftoverID int64, req *model.HandleLeftoverRequest) (*model.LeftoverView, error) {
	plan, err := s.repo.GetPlanByID(planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGradingPlanNotFound
		}
		return nil, err
	}
	if plan.Status == model.GradingPlanDraft {
		return nil, ErrGradingPlanNotConfirmed
	}

	leftover, err := s.repo.GetLeftover(leftoverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeftoverNotFound
		}
		return nil, err
	}
	if leftover.PlanID != planID {
		return nil, ErrLeftoverNotFound
	}

	if !model.ValidLeftoverAction(req.Action) {
		return nil, fmt.Errorf("%w: action 必须是 discount_sale/process/gift/waste/other 之一", ErrInvalidInput)
	}

	handledAt := time.Now()
	if req.HandledAt != "" {
		handledAt, err = parseTime(req.HandledAt)
		if err != nil {
			return nil, fmt.Errorf("%w: handled_at 时间格式无法解析", ErrInvalidInput)
		}
	}

	ok, err := s.repo.MarkLeftoverHandled(leftoverID, req.Action, req.Handler, handledAt, req.Note)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLeftoverAlreadyHandled
	}

	// 全部余量处理完毕 → 方案完结
	unhandled, err := s.repo.CountUnhandledLeftovers(planID)
	if err != nil {
		return nil, err
	}
	if unhandled == 0 {
		if err := s.repo.UpdatePlanStatus(planID, model.GradingPlanCompleted); err != nil {
			return nil, err
		}
	}

	gradeName := ""
	grades, err := s.repo.ListGrades(planID)
	if err != nil {
		return nil, err
	}
	for _, g := range grades {
		if g.ID == leftover.GradeID {
			gradeName = g.GradeName
			break
		}
	}

	return &model.LeftoverView{
		ID:         leftover.ID,
		GradeID:    leftover.GradeID,
		GradeName:  gradeName,
		LeftoverKg: leftover.LeftoverKg,
		Action:     &req.Action,
		Handler:    &req.Handler,
		HandledAt:  &handledAt,
		Note:       req.Note,
	}, nil
}

// buildGrades 校验级别明细，算出每个级别装不满一箱的余量。
func (s *GradingService) buildGrades(totalKg float64, inputs []model.GradeInput) ([]model.GradingPlanGrade, []float64, error) {
	if totalKg <= 0 {
		return nil, nil, fmt.Errorf("%w: 总产量必须大于 0", ErrInvalidInput)
	}

	specs, err := s.packSpecRepo.List()
	if err != nil {
		return nil, nil, err
	}
	specByID := make(map[int64]model.PackSpec, len(specs))
	for _, sp := range specs {
		specByID[sp.ID] = sp
	}

	grades := make([]model.GradingPlanGrade, len(inputs))
	leftoverKgs := make([]float64, len(inputs))
	for i, in := range inputs {
		if in.YieldKg <= 0 {
			return nil, nil, fmt.Errorf("%w: 级别「%s」产量必须大于 0", ErrInvalidInput, in.GradeName)
		}
		spec, ok := specByID[in.PackSpecID]
		if !ok {
			return nil, nil, fmt.Errorf("%w: 级别「%s」指定的包装规格 %d", ErrPackSpecNotFound, in.GradeName, in.PackSpecID)
		}
		grades[i] = model.GradingPlanGrade{
			GradeName:  in.GradeName,
			SizeSpec:   in.SizeSpec,
			Appearance: in.Appearance,
			YieldKg:    in.YieldKg,
			PackSpecID: in.PackSpecID,
		}
		_, leftoverKgs[i] = PackCalc(in.YieldKg, spec.CapacityKg)
	}
	return grades, leftoverKgs, nil
}

func (s *GradingService) CreatePackSpec(req *model.CreatePackSpecRequest) (*model.PackSpec, error) {
	if req.CapacityKg <= 0 {
		return nil, fmt.Errorf("%w: 每箱容量必须大于 0", ErrInvalidInput)
	}
	spec := &model.PackSpec{
		Name:       req.Name,
		CapacityKg: req.CapacityKg,
		Material:   req.Material,
	}
	if err := s.packSpecRepo.Create(spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func (s *GradingService) ListPackSpecs() ([]model.PackSpec, error) {
	return s.packSpecRepo.List()
}

// parseTime 依次尝试常见时间格式
func parseTime(s string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}
