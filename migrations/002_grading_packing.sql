BEGIN;

-- 包装规格字典（一箱装多少公斤）
CREATE TABLE IF NOT EXISTS pack_spec (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    capacity_kg DECIMAL(10,2) NOT NULL CHECK (capacity_kg > 0),
    material VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pack_spec_name ON pack_spec(name);

-- 分级装箱方案（一个批次一份）
CREATE TABLE IF NOT EXISTS grading_plan (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES crop_batch(id),
    total_yield_kg DECIMAL(12,2) NOT NULL CHECK (total_yield_kg > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','confirmed','completed')),
    remark VARCHAR(255) NOT NULL DEFAULT '',
    created_by VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 一个批次只有一份方案
CREATE UNIQUE INDEX IF NOT EXISTS idx_grading_plan_batch ON grading_plan(batch_id);

-- 级别明细（按大小/品相分级，每个级别指定一种包装规格）
CREATE TABLE IF NOT EXISTS grading_plan_grade (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES grading_plan(id) ON DELETE CASCADE,
    grade_name VARCHAR(64) NOT NULL,
    size_spec VARCHAR(64) NOT NULL DEFAULT '',
    appearance VARCHAR(128) NOT NULL DEFAULT '',
    yield_kg DECIMAL(12,2) NOT NULL CHECK (yield_kg >= 0),
    pack_spec_id BIGINT NOT NULL REFERENCES pack_spec(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grading_grade_plan ON grading_plan_grade(plan_id);

-- 余量处理（装不满一箱的部分单独列出，处理人与时间留痕）
CREATE TABLE IF NOT EXISTS grading_leftover (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES grading_plan(id) ON DELETE CASCADE,
    grade_id BIGINT NOT NULL REFERENCES grading_plan_grade(id) ON DELETE CASCADE,
    leftover_kg DECIMAL(12,2) NOT NULL CHECK (leftover_kg > 0),
    action VARCHAR(16) CHECK (action IN ('discount_sale','process','gift','waste','other')),
    handler VARCHAR(128),
    handled_at TIMESTAMPTZ,
    note VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grading_leftover_plan ON grading_leftover(plan_id);

COMMIT;
