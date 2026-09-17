BEGIN;

-- 分级与装箱方案(一个批次一份方案)
CREATE TABLE IF NOT EXISTS packing_plan (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES crop_batch(id),
    total_yield_kg DECIMAL(10,2) NOT NULL CHECK (total_yield_kg > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','confirmed')),
    note VARCHAR(512),
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 一个批次只允许一份方案
CREATE UNIQUE INDEX IF NOT EXISTS idx_packing_plan_batch ON packing_plan(batch_id);

-- 分级明细:按大小(size_spec)与品相(quality_desc)分级,并定包装规格
CREATE TABLE IF NOT EXISTS packing_plan_grade (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES packing_plan(id) ON DELETE CASCADE,
    grade_name VARCHAR(64) NOT NULL,
    size_spec VARCHAR(128),
    quality_desc VARCHAR(255),
    weight_kg DECIMAL(10,2) NOT NULL CHECK (weight_kg > 0),
    box_spec VARCHAR(64) NOT NULL,
    kg_per_box DECIMAL(10,2) NOT NULL CHECK (kg_per_box > 0),
    remainder_handling VARCHAR(255),
    remainder_handler VARCHAR(128),
    remainder_handled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_packing_plan_grade_plan ON packing_plan_grade(plan_id);

COMMIT;
