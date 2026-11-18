-- 分诊的动态（M7 设计 4.2，P2a spec 2.2）：field = 'intake' 的一行以新的状态名为 verb（pending、snoozed、accepted、
-- declined、duplicate），其余的行仍只有 created、updated、deleted。约束名不变。
-- field 可空（创建动态的 field 为空）：写 IS NOT DISTINCT FROM 而不是 =，否则 field 为空时括号里为 NULL，CHECK 对 NULL
-- 放行，空 field 的行就能带上新的 verb（预检 re-85 I1）。
-- Down 先删 field = 'intake' 的动态，00019 的 CHECK 才对表里的每一行成立（预检 re-85 m5）。

-- +goose Up
ALTER TABLE issue_activities
    DROP CONSTRAINT issue_activities_verb_check,
    ADD CONSTRAINT issue_activities_verb_check CHECK (
        verb IN ('created', 'updated', 'deleted')
        OR (field IS NOT DISTINCT FROM 'intake' AND verb IN ('pending', 'snoozed', 'accepted', 'declined', 'duplicate'))
    );

-- +goose Down
DELETE FROM issue_activities WHERE field = 'intake';
ALTER TABLE issue_activities
    DROP CONSTRAINT issue_activities_verb_check,
    ADD CONSTRAINT issue_activities_verb_check CHECK (verb IN ('created', 'updated', 'deleted'));
