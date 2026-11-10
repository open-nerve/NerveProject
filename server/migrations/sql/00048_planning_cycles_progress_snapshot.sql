-- cycles.progress_snapshot：迭代结束时的进度（M6 设计 3.5、P4 spec 2.4、R5），即接口的 FrozenProgress（字段名同接口），
-- 算的是全部工作项。由周期任务 planning.freeze_cycles 或转移写入，只在为空时写（先写者为准），之后不覆盖也不清除。
-- NULL 即未冻结（Plane 用 {}）；存的只能是对象。冻结不改 updated_at、updated_by_id（spec R6）。
-- 部分索引供周期任务不加锁地找候选：未删除、未冻结、有结束日的迭代按结束日（spec 2.5）。
-- 运行角色已有 cycles 的 DML，不改授权。

-- +goose Up
ALTER TABLE cycles
    ADD COLUMN progress_snapshot jsonb,
    ADD CONSTRAINT cycles_progress_snapshot_check CHECK (
        progress_snapshot IS NULL OR jsonb_typeof(progress_snapshot) = 'object'
    );
CREATE INDEX cycles_freeze_due_idx ON cycles (end_date)
    WHERE deleted_at IS NULL AND progress_snapshot IS NULL AND end_date IS NOT NULL;

-- +goose Down
DROP INDEX cycles_freeze_due_idx;
ALTER TABLE cycles DROP COLUMN progress_snapshot;
