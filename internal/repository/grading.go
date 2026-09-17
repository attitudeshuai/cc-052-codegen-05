package repository

import (
	"cc-052/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type PackSpecRepo struct {
	db *sqlx.DB
}

func NewPackSpecRepo(db *sqlx.DB) *PackSpecRepo {
	return &PackSpecRepo{db: db}
}

func (r *PackSpecRepo) Create(s *model.PackSpec) error {
	query := `INSERT INTO pack_spec (name, capacity_kg, material) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.QueryRow(query, s.Name, s.CapacityKg, s.Material).Scan(&s.ID, &s.CreatedAt)
}

func (r *PackSpecRepo) List() ([]model.PackSpec, error) {
	var list []model.PackSpec
	query := `SELECT id, name, capacity_kg, material, created_at FROM pack_spec ORDER BY capacity_kg`
	if err := r.db.Select(&list, query); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PackSpecRepo) GetByID(id int64) (*model.PackSpec, error) {
	var s model.PackSpec
	query := `SELECT id, name, capacity_kg, material, created_at FROM pack_spec WHERE id = $1`
	if err := r.db.Get(&s, query, id); err != nil {
		return nil, err
	}
	return &s, nil
}

type GradingRepo struct {
	db *sqlx.DB
}

func NewGradingRepo(db *sqlx.DB) *GradingRepo {
	return &GradingRepo{db: db}
}

// CreatePlan 在一个事务里写入方案、级别明细与余量记录。
// leftoverKgs 与 grades 按下标一一对应，>0 才生成余量记录。
func (r *GradingRepo) CreatePlan(plan *model.GradingPlan, grades []model.GradingPlanGrade, leftoverKgs []float64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	planQuery := `INSERT INTO grading_plan (batch_id, total_yield_kg, status, remark, created_by)
	              VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`
	if err := tx.QueryRow(planQuery, plan.BatchID, plan.TotalYieldKg, plan.Status, plan.Remark, plan.CreatedBy).
		Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return err
	}

	if err := insertGradesTx(tx, plan.ID, grades, leftoverKgs); err != nil {
		return err
	}

	return tx.Commit()
}

// ReplaceGrades 整体替换级别明细（仅草稿状态调用），并同步方案总量与备注。
func (r *GradingRepo) ReplaceGrades(plan *model.GradingPlan, grades []model.GradingPlanGrade, leftoverKgs []float64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	planQuery := `UPDATE grading_plan SET total_yield_kg = $1, remark = $2, updated_at = NOW()
	              WHERE id = $3 RETURNING updated_at`
	if err := tx.QueryRow(planQuery, plan.TotalYieldKg, plan.Remark, plan.ID).Scan(&plan.UpdatedAt); err != nil {
		return err
	}

	// 明细与余量（ON DELETE CASCADE）一并重建
	if _, err := tx.Exec(`DELETE FROM grading_plan_grade WHERE plan_id = $1`, plan.ID); err != nil {
		return err
	}

	if err := insertGradesTx(tx, plan.ID, grades, leftoverKgs); err != nil {
		return err
	}

	return tx.Commit()
}

// insertGradesTx 插入级别明细，并按 leftoverKgs 为有余量的级别生成余量记录。
func insertGradesTx(tx *sqlx.Tx, planID int64, grades []model.GradingPlanGrade, leftoverKgs []float64) error {
	gradeQuery := `INSERT INTO grading_plan_grade (plan_id, grade_name, size_spec, appearance, yield_kg, pack_spec_id)
	               VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	leftoverQuery := `INSERT INTO grading_leftover (plan_id, grade_id, leftover_kg) VALUES ($1, $2, $3)`
	for i := range grades {
		grades[i].PlanID = planID
		if err := tx.QueryRow(gradeQuery, planID, grades[i].GradeName, grades[i].SizeSpec,
			grades[i].Appearance, grades[i].YieldKg, grades[i].PackSpecID).
			Scan(&grades[i].ID, &grades[i].CreatedAt); err != nil {
			return err
		}
		if leftoverKgs[i] > 0 {
			if _, err := tx.Exec(leftoverQuery, planID, grades[i].ID, leftoverKgs[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *GradingRepo) GetPlanByBatch(batchID int64) (*model.GradingPlan, error) {
	var p model.GradingPlan
	query := `SELECT id, batch_id, total_yield_kg, status, remark, created_by, created_at, updated_at
	          FROM grading_plan WHERE batch_id = $1`
	if err := r.db.Get(&p, query, batchID); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *GradingRepo) GetPlanByID(id int64) (*model.GradingPlan, error) {
	var p model.GradingPlan
	query := `SELECT id, batch_id, total_yield_kg, status, remark, created_by, created_at, updated_at
	          FROM grading_plan WHERE id = $1`
	if err := r.db.Get(&p, query, id); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *GradingRepo) ListGrades(planID int64) ([]model.GradingPlanGrade, error) {
	var list []model.GradingPlanGrade
	query := `SELECT id, plan_id, grade_name, size_spec, appearance, yield_kg, pack_spec_id, created_at
	          FROM grading_plan_grade WHERE plan_id = $1 ORDER BY id`
	if err := r.db.Select(&list, query, planID); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *GradingRepo) UpdatePlanStatus(id int64, status model.GradingPlanStatus) error {
	query := `UPDATE grading_plan SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *GradingRepo) ListLeftovers(planID int64) ([]model.GradingLeftover, error) {
	var list []model.GradingLeftover
	query := `SELECT id, plan_id, grade_id, leftover_kg, action, handler, handled_at, note, created_at
	          FROM grading_leftover WHERE plan_id = $1 ORDER BY id`
	if err := r.db.Select(&list, query, planID); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *GradingRepo) GetLeftover(id int64) (*model.GradingLeftover, error) {
	var l model.GradingLeftover
	query := `SELECT id, plan_id, grade_id, leftover_kg, action, handler, handled_at, note, created_at
	          FROM grading_leftover WHERE id = $1`
	if err := r.db.Get(&l, query, id); err != nil {
		return nil, err
	}
	return &l, nil
}

// MarkLeftoverHandled 登记余量处理；已处理过的记录不会被覆盖，返回 false。
func (r *GradingRepo) MarkLeftoverHandled(id int64, action, handler string, handledAt time.Time, note string) (bool, error) {
	query := `UPDATE grading_leftover SET action = $1, handler = $2, handled_at = $3, note = $4
	          WHERE id = $5 AND handled_at IS NULL`
	res, err := r.db.Exec(query, action, handler, handledAt, note, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *GradingRepo) CountUnhandledLeftovers(planID int64) (int, error) {
	var n int
	query := `SELECT COUNT(*) FROM grading_leftover WHERE plan_id = $1 AND handled_at IS NULL`
	if err := r.db.Get(&n, query, planID); err != nil {
		return 0, err
	}
	return n, nil
}
