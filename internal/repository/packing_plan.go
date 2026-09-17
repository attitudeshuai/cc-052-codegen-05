package repository

import (
	"cc-052/internal/model"

	"github.com/jmoiron/sqlx"
)

type PackingPlanRepo struct {
	db *sqlx.DB
}

func NewPackingPlanRepo(db *sqlx.DB) *PackingPlanRepo {
	return &PackingPlanRepo{db: db}
}

// Create 事务写入方案头与分级明细
func (r *PackingPlanRepo) Create(plan *model.PackingPlan, grades []model.PackingPlanGrade) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO packing_plan (batch_id, total_yield_kg, status, note, created_by)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	if err := tx.QueryRow(query, plan.BatchID, plan.TotalYieldKg, plan.Status, plan.Note, plan.CreatedBy).
		Scan(&plan.ID, &plan.CreatedAt); err != nil {
		return err
	}

	if err := insertGrades(tx, plan.ID, grades); err != nil {
		return err
	}
	return tx.Commit()
}

// Replace 整体替换草稿方案:更新方案头、清空旧明细、写入新明细
func (r *PackingPlanRepo) Replace(plan *model.PackingPlan, grades []model.PackingPlanGrade) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE packing_plan SET total_yield_kg = $1, note = $2, created_by = $3 WHERE id = $4`
	if _, err := tx.Exec(query, plan.TotalYieldKg, plan.Note, plan.CreatedBy, plan.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM packing_plan_grade WHERE plan_id = $1`, plan.ID); err != nil {
		return err
	}

	if err := insertGrades(tx, plan.ID, grades); err != nil {
		return err
	}
	return tx.Commit()
}

func insertGrades(tx *sqlx.Tx, planID int64, grades []model.PackingPlanGrade) error {
	query := `INSERT INTO packing_plan_grade
	          (plan_id, grade_name, size_spec, quality_desc, weight_kg, box_spec, kg_per_box)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for i := range grades {
		g := &grades[i]
		if _, err := tx.Exec(query, planID, g.GradeName, g.SizeSpec, g.QualityDesc, g.WeightKg, g.BoxSpec, g.KgPerBox); err != nil {
			return err
		}
	}
	return nil
}

func (r *PackingPlanRepo) GetByBatch(batchID int64) (*model.PackingPlan, error) {
	var p model.PackingPlan
	query := `SELECT id, batch_id, total_yield_kg, status, note, created_by, created_at
	          FROM packing_plan WHERE batch_id = $1`
	if err := r.db.Get(&p, query, batchID); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PackingPlanRepo) ListGrades(planID int64) ([]model.PackingPlanGrade, error) {
	grades := make([]model.PackingPlanGrade, 0)
	query := `SELECT id, plan_id, grade_name, size_spec, quality_desc, weight_kg, box_spec, kg_per_box,
	          remainder_handling, remainder_handler, remainder_handled_at, created_at
	          FROM packing_plan_grade WHERE plan_id = $1 ORDER BY id`
	if err := r.db.Select(&grades, query, planID); err != nil {
		return nil, err
	}
	return grades, nil
}

func (r *PackingPlanRepo) GetGradeByID(gradeID int64) (*model.PackingPlanGrade, error) {
	var g model.PackingPlanGrade
	query := `SELECT id, plan_id, grade_name, size_spec, quality_desc, weight_kg, box_spec, kg_per_box,
	          remainder_handling, remainder_handler, remainder_handled_at, created_at
	          FROM packing_plan_grade WHERE id = $1`
	if err := r.db.Get(&g, query, gradeID); err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *PackingPlanRepo) UpdateStatus(id int64, status model.PackingPlanStatus) error {
	query := `UPDATE packing_plan SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

// RecordRemainderHandling 登记余量处理:处理办法 + 处理人,处理时间取服务端当前时间
func (r *PackingPlanRepo) RecordRemainderHandling(gradeID int64, handling, handler string) error {
	query := `UPDATE packing_plan_grade
	          SET remainder_handling = $1, remainder_handler = $2, remainder_handled_at = NOW()
	          WHERE id = $3`
	_, err := r.db.Exec(query, handling, handler, gradeID)
	return err
}
