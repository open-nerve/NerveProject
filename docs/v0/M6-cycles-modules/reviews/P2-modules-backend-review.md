# M6/P2 阶段评审记录

> 2026-11-08：最终候选（修复轮、编辑器计时器与 I21 等待的两处修复与本记录，门禁于 `78c7f7d3` 全过）待推送、核分支 CI 与合入；分支 CI、合入、main CI 与归档清理在合入后由 P3 的 T1 回填到 §7。基线 main `0a601f212e32057c4e48fe43e8c4f19691254573`（P1 noff）。

## 1. 评审方式与范围

[spec](../specs/P2-modules-backend.md)与[plan](../plans/P2-modules-backend.md)界定 15 个 Task。实施前一次 fresh builder 对照 main 的实际代码核查 P2 的 spec 与 plan，报告见[re-56](../../../reviews/re-56-M6P2预检.md)，两处 High（e2e 删除故事要求链接行、矩阵的 `{link_id}` 已被占用）在实施前改入计划。实施逐 Task 进行（RED → GREEN → 变异，每次运行有收据），由实现子代理执行、主控逐 Task 复核收据与关键改动；台账在本阶段 ignored 目录 `.superpowers/sdd/P2-modules-backend/progress.md`，随阶段归档。T8 做第一个完整检查点并推送；门禁之后一次整分支 deep-review，报告见[re-57](../../../reviews/re-57-M6P2整体审核.md)，处理见 §6。

## 2. 验收状态

A1–A8 满足（[spec §4](../specs/P2-modules-backend.md#4-验收标准)已勾选），证据见 §4：A1（G9 建立）→ T5、T7、T13；A2（G9 读与改）→ T5、T9、T10、T13；A3（G13）→ T11、T13；A4（G10 链接）→ T8、T12、T13；A5（开关与权限）→ T6–T12 的矩阵与 apitest、T13；A6（删除与清理）→ T3、T4、T8 起的 e2e 删除故事；A7（共用链接）→ T2；A8 → T1、T7–T11 的文案、T14 的回执。A9 的本地七项门禁、整审与修复闭环见 §5、§6；准确分支 CI、noff 后 main CI 与归档清理见 §7。

## 3. 局部裁定

台账中约 70 条 `Ruling:`，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题：

- **共用链接的落点**：设计决定 17 写“移到 `platform`”，架构测试不允许 `domain`/`app` 导入 `platform`、`platform` 导入 `shared`，所以检查与元数据类型进 `internal/shared/link.go`，抓取进 `platform/outbound/linkpreview`，`bootstrap` 一个适配器交给两个模块（spec R12，预检确认属局部裁定）；`apigen` 的生成配置加 `nullable-type: true`，否则公共组件里可为空又可省略的字段会丢掉显式的 `null`。
- **表与删除**：三张表各一个迁移；删除任务改为 `cycles → modules → module_members → module_links` 四个命名步骤，预算边界从当前步骤探查；清理子先父；P1 只有一处断言随之改变（清理目录由一张表变四张）。
- **加锁与判定**：`see` 认三种对象，链接分两步读（链接、它的模块）；开关按动作二分（`module.*` 看模块开关，其余看迭代开关；P3 加 `module.preferences` 时须同步归类）；AllowCreator 的目标对链接取模块的建立者；已归档的模块拒绝任何修改，空改动也是。
- **规则与写入**：合并日期与资格在锁下一次报齐；只为新负责人与新加入的成员读角色；移出成员一条语句（这些行只在项目行锁下被写，同工作项的移除负责人）；名字与链接地址的唯一由约束保证、按约束名映射；改地址在更新语句里重置元数据；投递失败整笔回滚。
- **测试**：模块与链接的矩阵表复用迭代的列；`planningOn` 一次打开两个开关；e2e 的链接地址都是出站策略拦截的 `169.254.169.254`，抓取不出网。
- **修复轮**：`planning_module_reads_test.go` 用 SQL 归档两个模块（已完成、已取消，不违 R3）以造出先后不同的归档时刻，固定业务时钟下经接口做不到，保留。
- **CI 暴露的编辑器缺陷**：在 P2 内修复而不交 M8（§5）——根因明确、改动 8 行，不修则分支与 main 的 CI 间歇变红。
- **I21 的测试竞争**：CI 重跑也未过时按负责人规则停下报告，负责人同意后只改 I21 的等待（§5）；本机的连接预算不在 P2 改，交 M8（M6-M8-03）。
- 其余见台账。

## 4. 实施记录

- **T1 回填与阶段**（`7cce5ade`）：P1 阶段记录 §7、spec A8、设计 §14 按 P1 的归档台账回填；phase 改为 `M6/P2`。
- **T2 共用链接**（`11f4d5fb`）：纯移动，工作项链接的 23 个组合测试与移来的测试全过；变异（放过带凭据的地址、放宽页面上限、issue 不再调用标题检查、适配器丢弃结果）各自失败。
- **T3 建表**（`fb78913c`）：三个迁移、运行角色授权、三个动作与规则、表级守卫；变异（删状态 CHECK、名字索引去掉 `lower`、成员外键改 `SET NULL`、去掉链接的排除约束）各自失败。T3 之后的 bootstrap 基线有 8 个删除/清理守卫失败，与 P1 的基线同一组，T4 之后全绿。
- **T4 删除与清理**（`de0495af`、`da318eab`）：命名步骤、清理步骤、守卫与夹具；`TestPurgeScale`（`NERVE_PURGE_SCALE=1`）31 秒通过，追踪器认得三条新语句；bootstrap 整包一次通过；变异 5 个。
- **T5 纯规则**（`ac6fd7ff`）、**T6 加锁之路与角色端口**（`c5fa19bb`）：变异 15 个全部失败，P1 的 `lock_test` 只加假方法。
- **T7 建立模块**（`6052d71d`）、**T8 模块链接**（`d784d160`）：第一个模块接口、链接的增改删与列表、矩阵、e2e 夹具建模块与链接；同名与同地址的交错测试只成一次；e2e 的项目/工作区删除故事随三张表转绿；T8 做第一个完整检查点（§5）。
- **T9 读取**（`b83dfd25`）、**T10 修改**（`2f686238`）：两种列表次序、已归档的详情可读、成员整体替换；变异 10 个。
- **T11 归档与删除**（`e8bbc875`）、**T12 抓取**（`18992101`）：只归档已关闭的模块、删除连带成员与链接同一时刻；事务内投递、地址仍匹配才写；变异 8 个。
- **T13 接口故事**（`8a968ca3`）：G9、G10、G13 共四条，管理员用 PAT、成员用登录令牌；先跑三处故意的 Plane 式期望，各在预期行失败。
- **T14 文档**（`64d43fee`）：plane-diff、五份输入交接的“处理结果（M6/P2）”（均保持 open）、设计 §14；文档链接 0 失效。

## 5. 门禁与 CI

- **第一个检查点**（`d784d160`，T8 之后）：七项门禁串行全过，输入前后一致：`make gen-check`、`make lint-go`、`make test`（66 个包，829 秒）、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`（422 条全过）。分支 CI 42367215686 attempt 1 三个作业成功（e2e 16.5 分钟）。
- **整审前**（`64d43fee`，T14 之后）：七项门禁串行全过，输入前后一致：`make test`（66 个包，848 秒）、`make build && make e2e`（426 条全过，即 422 条加 G9 两条、G10、G13）。
- **修复轮之后**：在 `62bc0de4`（修复轮 `18c06607` 加审查报告与本记录）上串行重跑七项门禁，全部 exit 0、输入前后一致：`make gen-check`、`make lint-go`、`make test`（66 个包，849 秒）、`make lint-web`（50 项任务中 35 项由 turbo 缓存回放）、`make knip`、`make test-web`（16 项任务全部由缓存回放：`web/` 自 `64d43fee` 以来未变）、`make build && make e2e`（426 条全过）。
- **候选 `e7adbbca` 的分支 CI 失败**（42376662419 attempt 1）：server 成功；web 的编辑器单元测试 145 条全过，却因一个未处理错误失败——`ReferenceError: window is not defined`，出自 `code-block-node-view.tsx` 的复制计时器（点“复制代码”后 1 秒恢复图标），由 `editor-block-menu.test.tsx` 的复制用例触发；e2e 因依赖 web 被跳过。这不是已登记的 C7/C12，P2 也没有改编辑器包；本地两轮 `make test-web` 都由 turbo 缓存回放，没有重跑编辑器包。根因是计时器从不清除，节点视图卸载后仍对它设状态，测试环境先拆除时便落在没有 `window` 的环境里（M4 起就有）。修复 `254d3255`：计时器存入 ref、卸载时清除；新测试“代码块移除时复制标记的计时器随之结束”修复前失败（卸载后仍有 1 个计时器）、修复后通过，去掉卸载清除的变异使之失败。旧 CI 不重跑，新候选另起 CI。
- **修复编辑器之后**（`254d3255`）：第一次串行门禁 `make test` 有 2 个包因测试容器起不来而失败（S3 容器等 Docker 应答超时、Postgres 容器拿不到映射端口；62 个包通过；本机 Docker 当时运行二十余个其他项目的容器），属环境失败，重跑完整七项：`make gen-check`、`make lint-go`、`make test`（66 个包）、`make lint-web`、`make knip`、`make test-web`（编辑器包实跑通过）全部 exit 0；`make build && make e2e` 第一次 425 条通过、W6 的两次注册因 e2e 共用的 Postgres 连接数用尽（`too many clients already`）回答 500，同一输入只重跑这一项即 426 条全过。连接余量的观察交 [M8 交接 M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03)。
- **候选 `784d8e0a` 的分支 CI 失败、重跑也未过**（42380907752）：attempt 1 的 server、web（编辑器包实跑）成功，e2e 426 条中 425 过，M5 已记的 F16 (page) 同类间歇失败（`f16-content-lifetime-page.spec.ts:672`，展开评论的“Attachments”后找不到下载按钮；补记入 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)）；只重跑失败的作业，attempt 2 的 F16 通过，I21 (API) 失败——“module_links: retained IDs and every column”：furnish 建的模块链接在全行快照时 `metadata` 为 `{}`，比对时已被抓取写成 `{"error":"blocked",…}`。根因是 P2 自己的测试竞争：T12 起建模块链接即投递抓取，I21 的 `finishLinks` 只在快照前等 `issue_links` 的抓取，没有等 `module_links`；本地各轮与 attempt 1 抓取恰好先于快照。按负责人规则停下报告，负责人 2026-11-08 同意按方案修：`78c7f7d3` 让 `finishLinks` 对模块链接也等到恰好一次抓取完成；e2e 的类型、lint、格式检查与 I21 聚焦通过，把期望改为“没有抓取”的变异使 I21 失败（证明等待作用在模块链接上）。
- **修复 I21 之后**（`78c7f7d3`）：七项串行，前六项全部 exit 0（`make test` 66 个包）；`make build && make e2e` 第一次 425 条通过、W12 的命令因共用 Postgres 连接数用尽连不上数据库（第二次本地出现，按配置核算的连接预算写入 M6-M8-03：本机 9 个工作进程远超 100 个连接，CI 2 个有余量），同一输入只重跑这一项即 426 条全过。

## 6. 整分支审查

冻结候选 `64d43fee`，一次 fresh 只读审查（`fable`），报告见 [re-57](../../../reviews/re-57-M6P2整体审核.md)：Ready to merge: Yes，没有 Critical 或 Important，Minor 4 条。修复轮 `18c06607` 修 1、2：排除约束冲突的判断移入 `platform/postgres` 由两个模块共用、组合测试改经接口归档与删除模块。3（`write` 与 `named` 的字段重复）留到 P3 加新对象时改为嵌入；4（删模块先读时钟、后锁子行）与 M4 先例相同，记录不改。

## 7. 集成、归档与接续

最终候选推送后核准确分支 CI，`--no-ff` 合入 main 并核 main CI，再归档本阶段 ignored 证据（台账、收据、审者原文、CI 原件）并正常清理 worktree 与分支。结果由 P3 的 T1 按台账回填到这里。
