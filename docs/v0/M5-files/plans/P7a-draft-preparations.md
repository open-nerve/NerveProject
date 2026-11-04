# M5/P7a 草稿准备件恢复协议 Implementation Plan

> **For agentic workers:** 已按superpowers:executing-plans逐Task内联实施、一次fresh whole-branch deep-review及唯一修复轮完成T1–T7。开发基线`963636757fe5c5a8c10d5781b43d1481ba6f32a2`，noff main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`；准确分支/main CI、归档清理和P7b正式预检均已闭环，见[阶段记录§13](../reviews/P7a-draft-preparations-review.md#13-t7-集成归档与接续)。

**Goal:** 作者能经有界私有查询找回草稿未绑定准备件 ID，沿既有 DELETE 明确丢弃后恢复发布。
**Architecture:** asset app 声明窄读取端口，现有 owner Host.Inspect 和 file_assets 查询共用 ReadTx；API 复用 Asset DTO，生成客户端；不改草稿完整性/清理/写锁协议。
**Tech Stack:** 既有 Go、PostgreSQL/sqlc、OpenAPI/oapi-codegen、Playwright 和现有测试存储适配器；无新增依赖。
**Spec:** [P7a 合同](../specs/P7a-draft-preparations.md)、[已批准 A](../reviews/P7-draft-preparation-recovery-decision.md)、[P5 绑定](../specs/P5-content-binding.md)。

## Global Constraints

- 根目录永远 main；仅本 worktree 实施。plane/refer 只读，不在其中跑 git；详细中文提交，系统 /usr/bin/git push，合并 --no-ff。禁止 stash/clean/reset-hard/force、凭据提取、全局安装和未批准依赖。
- 新 API 仅本私有查询；无表、写端点、自动丢弃或平行权限。不得改旧 draft API 的 403 顺序、全局 assetHostError、M3/M4 守卫、P5 发布与清理语义。
- 新文件约400行以内，生成物例外；现有大文件只拆本次职责。七个可独立验收 Task；跨模块新风险先提交再上报。
- Task 用真实行为断言记录 RED/GREEN；缺模块、编译失败、环境失败不算 RED。生成物先改源再 make gen，代码稳定后串行门禁；保留完整 raw/输入hash/cache/skip/变异恢复SHA。
- 数据库套件一次一个，跨包 -p 1；只清理本阶段明确拥有的临时容器，禁止操作既有容器/Docker或 prune。
- 预检阶段未实现、安装依赖或改 phase；现已获准实施，首个产品提交phase=M5/P7，P7a/P7b共用这个既有数字阶段值，不改PHASE_RE。P7b才迁移旧链及到期例外，不把P7a当最后旧消费者迁完；既有until仍按实际期限处理，不延期掩盖。

## Review Focus

1. 未知 ID 准备件必须从新 API 发现，不能在测试中从 SQL 偷出 ID；失败复制的 pending 也能恢复（T3）。
2. 新查询私有404归一仅限 Host.Inspect 的 forbidden，401/意外错误不吞；他人/guest/父已删与随机不存在同形且无文件名（T2/T4）。
3. 授权与列表同一只读快照，每页重新授权，分页 cursor 不扩大范围；旧列表不是删除/发布许可（T1/T4）。
4. DELETE/complete/publish/移动交错继续遵守 P5；迟到 PUT 不复活，已绑定历史图不能因恢复入口被删（T3/T4）。
5. API/生成物/组合根只有一个 owner 读取入口，不建存储I/O、跨模块 JOIN、写锁或返回签名能力；P7a 不冒领页面完成（T1/T2/T5）。

## Task 1（T1）：纯游标、有界查询与只读应用用例

**Files:** 新 `server/internal/modules/asset/domain/draft_preparations.go`、`app/draft_preparations.go`、`adapter/postgres/draft_preparations.go`、`queries/draft_preparations.sql` 及对应测试；sqlc生成物；tools/keywords.json；阶段review。
**Interfaces:** spec§3/4 的 DraftPreparationQuery/Reader/Page、PreparationHostReader、ListDraftPreparations.Execute；既有 shared cursor/PageSize、Store.queries、decodeAsset。

- [x] Step 1：表驱动纯规则与实际 ReadTx/fake host/rows 覆盖规范游标/跨draft/空值、limit、RequireActor、Host拒绝不查rows、错误不吞、同ctx与不依赖写接口。实际仓储覆盖51/101+行、非时间顺序UUID、pending/uploaded、bound/deleted/other actor/other draft/null项目过滤，使用现有fixture和带注释的隔离仓储数据。
- [x] Step 2：`go test -p 1 -count=1 ./internal/modules/asset/domain ./internal/modules/asset/app ./internal/modules/asset/adapter/postgres`（cwd server）。先保留有区分度断言失败，再实现；SQL仅file_assets，最多limit+1，模块依赖守卫保持。generated SQL只由make gen生成。
- [x] Step 3：同命令GREEN，去掉actor过滤和同快照各一个有效变异/恢复SHA；核scope/用途/全父链、损坏记录错误；详细中文提交 `feat(M5/P7a): 增加草稿准备件有界读取用例`。

## Task 2（T2）：契约、HTTP 与现有组合根接线

**Files:** api/modules/asset.yaml；新asset/adapter/http/draft_preparations.go；handler.go、module.go；相应HTTP/contract测试；api/dist、Go/TS生成物（web/packages/api-client/src/schema.gen.ts）；bootstrap新asset_draft_preparations_test.go。现有api-client export type *自动暴露类型，不改index.ts。
**Interfaces:** GET /api/v0/drafts/{draft_id}/asset-preparations、operationId listDraftAssetPreparations、DraftAssetPreparationPage；生成严格handler消费新用例，AssetDTO复用。

- [x] Step 1：先写YAML再make gen，同Task实现handler/UseCases/module注入，不能提交临时空列表桩。真实HTTP覆盖200空/非空、401、非法参数400/422、owner/member/nonauthor-admin/guest、无项目、not_found同码体头（剔除每请求本来不同的request-id），无key/URL/token。
- [x] Step 2：`go test -p 1 -count=1 ./internal/modules/asset/adapter/http`；`go test -p 1 -count=1 -run 'Test(DraftAssetPreparations|AssetHost|IssueDraft)' ./internal/bootstrap`。RED须在可运行HTTP/行为断言上，记录before/after；R1只映新用例Host forbidden，存储/DB故障仍500，旧API403保持。
- [x] Step 3：make gen-check、受影响API契约和架构测试、Go lint；确认web仅生成类型，无未用手写service。提交 `feat(M5/P7a): 开放作者私有的准备件查询接口`。

## Task 3（T3）：证明未知 ID 的真实恢复链

**Files:** bootstrap/asset_draft_preparations_test.go及按职责拆分的新恢复测试；e2e/stories/files/f10-draft-preparations.spec.ts；复用现有资产/草稿HTTP fixture，必要受控存储故障helper。
**Interfaces:** 真实 prepare/duplicate/new list/既有DELETE/draft publish；客户端不知道ID，SQL只做结果断言；无浏览器页面实现。

- [x] Step 1：HTTP prepare已提交但客户端丢弃答案，publish原asset.not_uploaded，重新读取列表找到且只选该pending，DELETE204后publish成功；两同名文件以ID区分，不误删邻域/另一标签页准备件。重复确认/删除无重复事件、计数或任务。
- [x] Step 2：duplicate创建target后让已有封存适配器明确失败，客户端只得错误；从目标draft列表发现pending，检查来源文件仍可读，丢弃后publish成功。完成但未绑定描述图也可列，当前/历史已绑定图不可列、其DELETE仍asset.bound；null项目草稿读取恢复后选项目发布。不能仅用已知ID重复旧P5测试。
- [x] Step 3：`go test -p 1 -count=1 -run '^TestDraftAssetPreparations' ./internal/bootstrap`；`make build` 后 `pnpm --dir e2e exec playwright test stories/files/f10-draft-preparations.spec.ts --project=chromium --workers=1`。实际API恢复、无页面完成声称；记录HTTP/DB/字节证据，提交 `test(M5/P7a): 证明未知准备件可发现并恢复发布`。

## Task 4（T4）：快照、权限与并发退路

**Files:** bootstrap/asset_draft_preparations_snapshot_test.go、asset_draft_preparations_concurrency_test.go；现有 draft_assets/asset_duplicate/asset_http_security 回归按实际影响；纯断言/暂停helper。
**Interfaces:** 同ReadTx/真实PID事务暂停点，已有owner/asset写API；不添加测试专用生产路由。

- [x] Step 1：在host读取后暂停，用第二连接完成撤权/删除/换项目/发布；当前页只能呈现一致的旧快照，下一页/重读按新状态拒绝或新范围。用户退出/停用、草稿作者降guest/恢复、其他工作区/草稿无扰；新API404不影响旧API403。取消暂停读后连接可复用，不固定sleep假定锁已到。
- [x] Step 2：列表读旧unbound后，描述图绑定再DELETE仍asset.bound；普通附件先complete绑定后按原授权DELETE仍可204，两种用途分列，不改旧写合同。DELETE先成功后complete/迟到PUT不复活；pending使publish拒绝、显式丢弃后publish与并发新prepare仍以D(U)次序裁定；分页中新增/删除/过界cursor不会跨作者或无限重复。确认查询无新写锁、存储I/O、事件/计数变化；P5双发布/迁移/清理原合同回归。
- [x] Step 3：`go test -p 1 -race -count=5 -run '^TestDraftAssetPreparations' ./internal/bootstrap` 串行；独立变异漏作者/漏unbound、授权和rows分事务、R1吞全部错误、cursor不绑draft各有有效断言失败并恢复SHA。环境失败单列。提交 `test(M5/P7a): 约束准备件权限快照和交错行为`。

## Task 5（T5）：七门禁、证据与检查点 CI

- [x] Step 1：A1–A4逐项证据，生成物/错误code/唯一读取/无存储依赖/无写副作用核对，P7b接收合同更新；所有raw、失败、输入清单和容器前后快照保留。
- [x] Step 2：串行 `make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e`；完整exit/cache/skip和前后输入hash。无新页面，不为这个纯协议新增三浏览器页面矩阵；既有完整E2E仍为门禁。
- [x] Step 3：详细中文检查点提交/system push，准确HEAD server/web/e2e全绿；用已连接GitHub日志查失败。W11保持已知一次重跑纪律，不自改CI预算掩盖故障。

## Task 6（T6）：一次整分支 deep-review 与修复

- [x] Step 1：冻结正式main基线..候选HEAD，fresh reviewer读完整手写diff和调用链、五Review Focus/所有Ruling、真实证据；三视角报告保存docs/reviews下一编号，生成物核源。T1–T4产品冻结后，该只读审查可与T5门禁/CI并行；待门禁完成才进入修复及最终判定，不以审查替代测试。
- [x] Step 2：主控逐条裁定，局部问题根因修复、必要RED/GREEN及受影响回归；新架构问题先提交上报。不得重复同范围再派审者替代处理，DeclinedToJudge具名闭环。
- [x] Step 3：最终本地门禁覆盖当前输入；准确最终分支CI全绿后关闭A5，明确P7b页面尚未实现，handoffs保持open。

## Task 7（T7）：集成归档与 P7b 正式预检

- [x] Step 1：持续授权下核main/worktree干净、祖先和准确green HEAD，--no-ff后核双父/产品树，system push；准确main CI三job全绿。
- [x] Step 2：归档全部ignored原件、原P7准备记录与本阶段证据；hash/bytes/mode/membership/unknown0，origin/main..branch=0后正常清远端/本地/worktree，归档再次核。不得漏掉改名之前的准备台账。
- [x] Step 3：独立codex/m5-p7b worktree回填A6；将[P7b正式合同](../specs/P7b-content-assets.md)与真实新API/最终main重新核对，完成正式预检。P7b正式预检通过但暂不实施；M5最终F1–F18/P8责任保持。

## 预检自查

- [x] A1→T1/T2，A2→T2/T4，A3→T3，A4→T1/T4，A5→T5/T6，A6→T7；七Tasks，无实现越界。
- [x] 真实API/目录/消费接口/ReadTx/权限顺序/无项目草稿/DTO和游标机制均与源码相符。
- [x] R1为新私有端点局部处理，无全局错误变更；P7b恢复UI与普通副本最终Issue/Draft区别已接收。
- [x] 新P7a范围一次fresh builder完成，问题按已有事实/局部决定/负责人决定闭环；P6准确main CI/归档前置核验。
