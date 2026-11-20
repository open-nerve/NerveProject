# M7 插入段：M4 泳道查询的计划翻转（spec 与 plan）

> 状态：spec 与 plan 经预检 [re-89](../../../reviews/re-89-M4泳道计划预检.md) 修正后通过。基线 main `7dfaa8a1`。worktree `.codex/worktress/codex/m7-swimlanes`，分支 `codex/m7-swimlanes`。台账 `.superpowers/sdd/Swimlanes-plan/progress.md`（ignored），根因调研 `research/swimlanes-facts.md`（下文 F§n 指它的节）。负责人 2026-11-20 裁定：“M7 里单独修……按根因改泳道查询的写法让计划稳定（不加索引），带规模证据，合入后再继续 P3a/P3b。”

## 1. 问题与根因

M7/P3a 的规模测试（T7）把 10 个项目的子种子放进 M4 规模种子的同一个库时，1 万个工作项的项目的泳道从 SQL 中位 60.687 ms 变成 2349 ms，空页过 3 秒的请求期限（F§1.6）。

根因（F§1.2–1.5）：`SubgroupIssues`（`server/internal/modules/issue/adapter/postgres/queries/list_issue_subgroups.sql:90-129`）把同一批工作项展开两遍（按组键、按子组键），各自算窗口总数，再在 `cells` 里按工作项 `id` 把两条派生流连起来（`:129`）。两边都是查询里算出来的结果，没有统计也没有索引，连接方法全凭行数估计；而估计在健康库里已经低估约 150 倍（项目行数按 `n_distinct(project_id)` 平均、项目 id 经 `unnest` 连入用不上 MCV、states 连两次、空日期数组的反连接再减半）。库里项目一多，估计再降，规划器就把哈希连接换成嵌套循环：12700 × 12750 个行对、连接过滤去掉 1.619 亿行、EXPLAIN 7.6 s。M4 收尾以 `cells AS MATERIALIZED` 挡过同一连接的另一种触发（`48e73489`），没有去掉连接本身。

波及面（F§2，re-89 I3）：这是唯一**实测到的**平方悬崖。平铺、分组、成员列表、工作区列表、收集箱、搜索都没有两条派生流的连接。另有：

- **B3（同一形状类、未测）**：按 `attachment_count` 排序时，附件数组装着范围内所有带附件的工作项，三条列表 SQL 都把它与估计不准的连接结果相连、没有索引（`app/listing.go:93-95`、`app/list_attachment_counts.go:24`、`queries/list_issue_subgroups.sql:21`）；坏计划里已是带连接过滤的嵌套循环，只因规模种子没有附件才无害。不在负责人裁定的范围内，本段不改；交 M8，并建议 M8 的规模种子加附件。收尾报告里向负责人提一句。
- 线性的风险：B1 第二次连 states、B2 多项目列表连 projects、B4 `label_name` 排序（已交 M8）。交 M8。

## 2. 修法（F§3.1 方案 A）

单流：每个工作项按（组键 × 子组键）只展开一次（两个 `LEFT JOIN LATERAL unnest(...) WITH ORDINALITY`），三级总数都是这一条流上的窗口——组总数数“子组方向上的第 1 行”（`first_lane`），子组总数数“组方向上的第 1 行”（`first_group`），格总数按（组, 子组）分区。没有任何两条流之间的连接，计划与估计无关，工作量 O(行 × |组键| × |子组键|)。

- 只改 `list_issue_subgroups.sql` 的 `expanded` 到 `cells`（`:90-129`）与 `:124-126` 的注释；删去 `subgroup_expanded`、`group_counts`、`subgroup_counts`。其余一字不动：输出列、参数、排序、游标、`numeric_keys` 的原文、`total`。`make gen-go` 后生成的 Go 只有 SQL 常量变。
- 语义逐项相等（F§3.1；re-89 逐项核过：空数组、含一个 NULL 元素的数组、CASE 不匹配得到的 NULL 数组、多值键、只看自己、数值键、游标、空页回退、`total`、LATERAL 的次序）。重复键时两式同样重复——`issue_labels`、`issue_assignees` 的部分唯一索引让活跃行不会重复，这一格靠论证，测试造不出来。
- `cells` 的 `MATERIALIZED`（re-89 I4）：它挡的是组键下推进两条分别分区的窗口流；单流里组键、子组键、游标列都不在全部三个计数窗口的分区里，PG 18 下推不了，没有可挡的了。**去掉**。核实的标准：两个空页 EXPLAIN（首查与回退）里，组键与游标的过滤在 `cells` 子查询扫描上、位于三个计数 WindowAgg 之上，且 oracle 全绿。只在 20k 空页超过健康基线的 1.10 倍（193.5 ms）时以性能理由保留，并写明数字。
- 不加索引、不加迁移、不用 `SET`、不用 `CREATE STATISTICS`、不改接口与游标（F§3.3–3.6 的理由）。

## 3. 证据与通过线（F§5）

- **拥挤库场景**：新文件 `server/internal/bootstrap/issue_list_crowded_scale_test.go`；`TestIssueListScale` 里加一行子测试的挂钩（`issue_list_scale_test.go:157-158` 之间，`NERVE_LIST_SCALE=1` 才跑，CI 上仍是一次跳过，不新增顶层名）。
  - 种子（re-89 I1）：M4 种子之后同库再加 10 个项目、各 n/10 个工作项，用与 `seedProject` 相同的状态分布（5 个状态加 Triage）；walt、bob 不是这些项目的成员（oracle 不变）。拥挤前 `checkDistribution`；拥挤后对与 M4 种子相同的表清单 `ANALYZE`，记下 `pg_stats`。目标统计：10k 时 `n_distinct(project_id)=15`、`n_distinct(state_id)≈90`。
  - 每个场景各用自己的路由（避开测试路由的限流）；证据写在自己的目录下（`t7-medians.py` 那样按 `int(目录名)` 读的工具不与它混放）。
  - 10k、20k 跑全部 13 个场景；100k 只在修后跑一次记录（修前的拥挤 100k 会撞 EXPLAIN 重放的 2 分钟期限，`issue_list_explain_test.go:49`）。
- **红（T1，进 T2 的前提）**：在 main 的 SQL 上，拥挤库的泳道空页过 3 秒期限（或中位远超通过线），证据里含坏计划的估计（每条流约 10 行）与 `pg_stats`。红没复现就停下，调种子的统计到目标再试，不进 T2。
- **健康基线（T1，同一台机器）**：在本台账重跑一次 M4 的 13 个场景（约 65 秒）作为修前基线；这一次的健康泳道若与 60.687 / 116.589 ms 相差超过 10%，以本次为准并记下。
- **通过线**：
  - 拥挤库的泳道中位 ≤ 健康修前基线的 1.10 倍（以 60.687 / 116.589 计为 10k ≤ 66.8 ms、20k ≤ 128.2 ms）；拥挤库每个请求 < 1 s。
  - 逐个读 13 个拥挤计划：没有两条派生结果的连接；`Rows Removed` 不超过“行数 × 目录大小”（10k 约 4.08 万），没有接近 10^7 的。
  - 健康库里泳道不慢于修前基线的 1.10 倍；其余 12 个场景在 10k、20k 不慢于修前 10% 以上（超过即停下查明）；100k 只记录。
- 度量照 M4 P5：1 次预热 + 5 次实测、列表 SQL 中位、留 EXPLAIN (ANALYZE, BUFFERS)。

## 4. Tasks

### T1 拥挤库场景、健康基线与红的证据
新文件与一行挂钩；健康基线；在 main 的 SQL 上跑出红（数字、坏计划的估计、`pg_stats`）。提交测试。红的窗口：`NERVE_LIST_SCALE=1` 下 `TestIssueListScale/crowded` 从 T1 红到 T2（CI 上是跳过，不受影响）。

### T2 单流改写
改 SQL 与注释、`make gen-go`；去掉 `MATERIALIZED`（按 §2 的标准）。

守卫（全跑）：`go test ./internal/modules/issue/adapter/postgres/... -count=1`、bootstrap 两路（`make test-shard SHARD=0 SHARDS=2`、`SHARD=1`）、`make gen-check`、`make lint-go`。

变异——改**生成的** `gen/list_issue_subgroups.sql.go` 里的 SQL 常量（测试跑的是它；锚点取生成后的原文，sqlc 会展开 `e.*`），还原后 `make gen-check` 无差异：

| 变异 | 应让它失败的测试 |
|---|---|
| 组总数去掉 `FILTER (WHERE first_lane)` | `TestIssueSubgroups` |
| 子组总数去掉 `FILTER (WHERE first_group)` | `TestIssueSubgroups` |
| 两个 FILTER 对调 | `TestIssueSubgroups` |
| 格总数的分区少一列 | `TestIssueSubgroups` |
| `coalesce(n,1)=1` 去掉 `coalesce`（空键的工作项不再计入） | `TestIssueSubgroups` 或 `TestIssueListAllGroupingFields`（以实测为准，记下是哪一个） |

规模证据按 §3 的通过线判定。文档：`docs/v0/M4-issue-core/M4-design.md:635`、`:637`，`docs/v0/M4-issue-core/specs/P5-issue-list.md:119` 的实现描述；`docs/v0/M8-open-release/handoffs/M4-closeout.md:85-86`（“100k 物化的代价交 M8”一项随之作废，写处理结果）。M4 的历史记录（阶段记录、收尾性能记录）不改。

### T3 文档与记录（主控）
阶段记录 `reviews/M4fix-swimlanes-plan-review.md`（根因、红与绿的证据、MATERIALIZED 的处理）；交 M8 的 B1–B3：写进 M7 的待交清单（M7 收尾时转入 M8 的 `handoffs/M7-closeout.md`），B3 按“同形状、未测、建议规模种子加附件”写；交 P3a：合并 main 时 T7 的两处调用点各加一个参数、并只把 M4 的 13 个场景交给拥挤库（P3a 有 21 个），T7 的泳道基线数字随之改变；M7 设计 §14 一行。

之后：整分支审查（fable，只读）→ 修复 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理。

## 5. 约束

- 证据经 `run-check.py`、变异经 `mutate.py`；容器的测试一次一套。
- 允许改的已有测试：只有 `issue_list_scale_test.go` 的一行挂钩。新文件不算改。其余测试（adapter、bootstrap、oracle、`list_precision_test.go`）一律不改。
