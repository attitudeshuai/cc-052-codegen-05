package model

import "time"

type GradingPlanStatus string

const (
	GradingPlanDraft     GradingPlanStatus = "draft"     // 草稿，可修改
	GradingPlanConfirmed GradingPlanStatus = "confirmed" // 已确认（总量平衡），锁定明细
	GradingPlanCompleted GradingPlanStatus = "completed" // 全部余量已处理
)

// 余量处理方式
type LeftoverAction string

const (
	LeftoverDiscountSale LeftoverAction = "discount_sale" // 折价销售
	LeftoverProcess      LeftoverAction = "process"       // 加工
	LeftoverGift         LeftoverAction = "gift"          // 赠送
	LeftoverWaste        LeftoverAction = "waste"         // 报废
	LeftoverOther        LeftoverAction = "other"         // 其他
)

func ValidLeftoverAction(a string) bool {
	switch LeftoverAction(a) {
	case LeftoverDiscountSale, LeftoverProcess, LeftoverGift, LeftoverWaste, LeftoverOther:
		return true
	}
	return false
}

// PackSpec 包装规格（一箱装多少公斤）
type PackSpec struct {
	ID         int64     `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	CapacityKg float64   `db:"capacity_kg" json:"capacity_kg"`
	Material   string    `db:"material" json:"material"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// GradingPlan 一批货的分级装箱方案
type GradingPlan struct {
	ID           int64             `db:"id" json:"id"`
	BatchID      int64             `db:"batch_id" json:"batch_id"`
	TotalYieldKg float64           `db:"total_yield_kg" json:"total_yield_kg"`
	Status       GradingPlanStatus `db:"status" json:"status"`
	Remark       string            `db:"remark" json:"remark"`
	CreatedBy    string            `db:"created_by" json:"created_by"`
	CreatedAt    time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at" json:"updated_at"`
}

// GradingPlanGrade 级别明细（按大小/品相分级，指定包装规格）
type GradingPlanGrade struct {
	ID         int64     `db:"id" json:"id"`
	PlanID     int64     `db:"plan_id" json:"plan_id"`
	GradeName  string    `db:"grade_name" json:"grade_name"`
	SizeSpec   string    `db:"size_spec" json:"size_spec"`
	Appearance string    `db:"appearance" json:"appearance"`
	YieldKg    float64   `db:"yield_kg" json:"yield_kg"`
	PackSpecID int64     `db:"pack_spec_id" json:"pack_spec_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// GradingLeftover 装不满一箱的余量及处理记录
type GradingLeftover struct {
	ID         int64      `db:"id" json:"id"`
	PlanID     int64      `db:"plan_id" json:"plan_id"`
	GradeID    int64      `db:"grade_id" json:"grade_id"`
	LeftoverKg float64    `db:"leftover_kg" json:"leftover_kg"`
	Action     *string    `db:"action" json:"action,omitempty"`
	Handler    *string    `db:"handler" json:"handler,omitempty"`
	HandledAt  *time.Time `db:"handled_at" json:"handled_at,omitempty"`
	Note       string     `db:"note" json:"note"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// ---- 请求 ----

type GradeInput struct {
	GradeName  string  `json:"grade_name" binding:"required"`
	SizeSpec   string  `json:"size_spec"`
	Appearance string  `json:"appearance"`
	YieldKg    float64 `json:"yield_kg" binding:"required"`
	PackSpecID int64   `json:"pack_spec_id" binding:"required"`
}

type CreateGradingPlanRequest struct {
	TotalYieldKg float64      `json:"total_yield_kg"` // 缺省取批次预计产量
	Remark       string       `json:"remark"`
	CreatedBy    string       `json:"created_by" binding:"required"`
	Grades       []GradeInput `json:"grades" binding:"required,min=1,dive"`
}

type UpdateGradingPlanRequest struct {
	TotalYieldKg float64      `json:"total_yield_kg"` // 缺省沿用方案原值
	Remark       string       `json:"remark"`
	Grades       []GradeInput `json:"grades" binding:"required,min=1,dive"`
}

type HandleLeftoverRequest struct {
	Action    string `json:"action" binding:"required"`  // 处理方式
	Handler   string `json:"handler" binding:"required"` // 处理人
	HandledAt string `json:"handled_at"`                 // 处理时间，缺省为当前时间
	Note      string `json:"note"`
}

type CreatePackSpecRequest struct {
	Name       string  `json:"name" binding:"required"`
	CapacityKg float64 `json:"capacity_kg" binding:"required"`
	Material   string  `json:"material"`
}

// ---- 响应（含计算结果） ----

// GradeResult 级别明细 + 装箱计算结果
type GradeResult struct {
	ID           int64   `json:"id"`
	GradeName    string  `json:"grade_name"`
	SizeSpec     string  `json:"size_spec"`
	Appearance   string  `json:"appearance"`
	YieldKg      float64 `json:"yield_kg"`
	PackSpecID   int64   `json:"pack_spec_id"`
	PackSpecName string  `json:"pack_spec_name"`
	CapacityKg   float64 `json:"capacity_kg"`
	FullBoxes    int64   `json:"full_boxes"`  // 整箱数
	LeftoverKg   float64 `json:"leftover_kg"` // 装不满一箱的余量
}

// LeftoverView 余量记录（带级别名）
type LeftoverView struct {
	ID         int64      `json:"id"`
	GradeID    int64      `json:"grade_id"`
	GradeName  string     `json:"grade_name"`
	LeftoverKg float64    `json:"leftover_kg"`
	Action     *string    `json:"action,omitempty"`
	Handler    *string    `json:"handler,omitempty"`
	HandledAt  *time.Time `json:"handled_at,omitempty"`
	Note       string     `json:"note"`
}

// GradingPlanDetail 方案详情：级别合计必须与总产量对得上，差额单独点名
type GradingPlanDetail struct {
	ID            int64             `json:"id"`
	BatchID       int64             `json:"batch_id"`
	TotalYieldKg  float64           `json:"total_yield_kg"`
	Status        GradingPlanStatus `json:"status"`
	Remark        string            `json:"remark"`
	CreatedBy     string            `json:"created_by"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Balanced      bool              `json:"balanced"`
	BalanceDiffKg float64           `json:"balance_diff_kg"` // 总产量 - 级别合计；>0 有产量未分级，<0 超出总产量
	TotalBoxes    int64             `json:"total_boxes"`
	Grades        []GradeResult     `json:"grades"`
	Leftovers     []LeftoverView    `json:"leftovers"`
}
