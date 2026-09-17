package model

import "time"

type PackingPlanStatus string

const (
	PackingPlanDraft     PackingPlanStatus = "draft"
	PackingPlanConfirmed PackingPlanStatus = "confirmed"
)

// PackingPlan 一批货的分级与装箱方案(一个批次一份)
type PackingPlan struct {
	ID           int64             `db:"id" json:"id"`
	BatchID      int64             `db:"batch_id" json:"batch_id"`
	TotalYieldKg float64           `db:"total_yield_kg" json:"total_yield_kg"`
	Status       PackingPlanStatus `db:"status" json:"status"`
	Note         string            `db:"note" json:"note"`
	CreatedBy    string            `db:"created_by" json:"created_by"`
	CreatedAt    time.Time         `db:"created_at" json:"created_at"`
}

// PackingPlanGrade 分级明细:按大小与品相分级,并定该级用什么包装、一箱装多少
type PackingPlanGrade struct {
	ID                 int64      `db:"id" json:"id"`
	PlanID             int64      `db:"plan_id" json:"plan_id"`
	GradeName          string     `db:"grade_name" json:"grade_name"`
	SizeSpec           string     `db:"size_spec" json:"size_spec"`
	QualityDesc        string     `db:"quality_desc" json:"quality_desc"`
	WeightKg           float64    `db:"weight_kg" json:"weight_kg"`
	BoxSpec            string     `db:"box_spec" json:"box_spec"`
	KgPerBox           float64    `db:"kg_per_box" json:"kg_per_box"`
	RemainderHandling  *string    `db:"remainder_handling" json:"remainder_handling,omitempty"`
	RemainderHandler   *string    `db:"remainder_handler" json:"remainder_handler,omitempty"`
	RemainderHandledAt *time.Time `db:"remainder_handled_at" json:"remainder_handled_at,omitempty"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
}

type PackingPlanGradeInput struct {
	GradeName   string  `json:"grade_name" binding:"required"`
	SizeSpec    string  `json:"size_spec"`
	QualityDesc string  `json:"quality_desc"`
	WeightKg    float64 `json:"weight_kg" binding:"required"`
	BoxSpec     string  `json:"box_spec" binding:"required"`
	KgPerBox    float64 `json:"kg_per_box" binding:"required"`
}

// UpsertPackingPlanRequest 创建/整体替换方案;total_yield_kg 缺省取批次预计产量
type UpsertPackingPlanRequest struct {
	TotalYieldKg float64                 `json:"total_yield_kg"`
	Note         string                  `json:"note"`
	CreatedBy    string                  `json:"created_by" binding:"required"`
	Grades       []PackingPlanGradeInput `json:"grades" binding:"required"`
}

// PackingPlanGradeView 分级明细 + 计算结果:整箱数与装不满一箱的余量
type PackingPlanGradeView struct {
	PackingPlanGrade
	BoxesNeeded int     `json:"boxes_needed"`
	RemainderKg float64 `json:"remainder_kg"`
}

// PackingPlanView 方案详情:含合计对账,unallocated_kg 非 0 即对不上的部分
type PackingPlanView struct {
	PackingPlan
	AllocatedKg      float64                `json:"allocated_kg"`
	UnallocatedKg    float64                `json:"unallocated_kg"`
	TotalBoxes       int                    `json:"total_boxes"`
	TotalRemainderKg float64                `json:"total_remainder_kg"`
	Grades           []PackingPlanGradeView `json:"grades"`
}

// RemainderHandlingRequest 登记余量处理:处理办法 + 处理人,处理时间由服务端记录
type RemainderHandlingRequest struct {
	Handling string `json:"handling" binding:"required"`
	Handler  string `json:"handler" binding:"required"`
}
