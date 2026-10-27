# 03 规矩和坑

## 1. 负责人的纪律
原话：

> 局部问题调整，优化，加固自行决策，不用停下来，只需要符合 SOLID ，不要打补丁 乱接线，根因解决问题 就 OK。 但是如果遇见 架构级问题，比如架构回归 ，或者跨模块协同调整 ，或者 意料之外的 高风险发现，那么就停下来 ，用大白话汇总上报问题 和 你的建议方案 ， 等我决策

**怎么做**：
- 局部的决定写进台账和评审记录第 3 节：结论、理由、错了的代价。
- 要上报时：先提交已完成的工作；用大白话一条条说问题、可选的方案和推荐；然后停下，不再往下做。
- 负责人的决策点和已批准的架构问题（设计第 10、11 节，17.4）不重开。
- 负责人习惯用中文交流。

## 2. 代码标准和产品原则
**后端（Go）**：
- 严格 SOLID。每个包职责清楚，依赖只在一个组合根（`server/internal/bootstrap`）里接线。
- 端口由使用它的一方声明，小接口。
- 模块之间不互相 import，不跨模块 JOIN；模块的 SQL 只经 sqlc，模块的 SQL 里没有 `SET`。
- 没有上帝文件（文件约 400 行以内，plan 的 Global Constraints 写明例外），没有循环依赖，没有全局可变状态，没有 `init()`，没有 `utils`、`common`、`helpers` 一类的包。
- 不过度设计：不做 CQRS、事件溯源，不用依赖注入框架。
- 端口的错误原样返回，不吞掉、不当成空。
- 角色按集合判定，不比较级别的大小。

**前端**：
- 实体类型直接用 OpenAPI 生成的类型，store 直接消费我们的接口的形状，不加把我们的接口翻译成 Plane 旧类型的层，不留兼容代码。

**删干净**：
- 砍掉的功能连同路由、组件、store、service、类型、常量、翻译键、不再用的依赖一起删除。
- 不用开关、桩或返回空的 service 藏起来。
- 没有使用者的代码不留。

**产品**：
- 不区分人和代理，不要提"代理账户"或人机标志。
- 保留的功能照 Plane 一比一（含表结构），先读 `plane/` 的实现再写。
- 和 Plane 不同的地方登记进 [plane-diff.md](../plane-diff.md)。

## 3. 硬规则
- **只读**：`plane/` 和 `refer/` 不改，也不在里面跑 git。
- **容器**：
  - `make test`、`make e2e` 用自己的 testcontainers；一次只跑一个套件；
  - 不停、不重建、不重启已有的容器，也不动 Docker Desktop；
  - 开发库 `nerve-dev-db-1` 是 Exited，别动它，不跑 `make dev-db-down`、`make dev-db-reset`；
  - 别的项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`、`e2e-postgres`、`e2e-redis-app` 等）、别的会话的 testcontainers、遗留的 `Created` 状态的 testcontainers 都不动；
  - 自己起的临时容器用可识别的名字，用完删掉。
- **安装**：
  - 不做全局安装，不 `corepack enable`；
  - 不加设计没有点名的 Go 模块或 npm 包，需要就上报；
  - 不给副本的 `node_modules` 建符号链接，用仓库自己的安装命令。
- **git**：
  - 主项目目录 HEAD 始终保持 `main`；开发分支用 worktree，Codex 路径为 `.codex/worktress/<分支>/`，不在主目录切开发分支；
  - 提交说明按负责人最新 AGENTS 使用详细中文；推送优先用系统 `/usr/bin/git`，不从钥匙串或其他位置提取凭据；
  - 不用 `stash`、`clean`、`reset --hard`（工作树之间共享 stash，别的会话可能在用）；
  - 需要完整 SHA 时用 `git rev-parse` 取，不手工补全；
  - 合并见 [02](02-method.md) 第 8 步。
- **持续集成和凭据**：
  - 没有 `gh`；
  - 作业日志和产物（Playwright 报告）要管理员权限，公开的 API 只给作业和步骤的结论；要日志或报告就找负责人要；
  - 不去钥匙串或别处取凭据。
- **用户的邮箱**只用来识别身份，不发到任何服务。

## 4. M4 的技术不变式（每个 Phase 都要守住）
**加锁的路**（设计 3.5，D1 = B）。普通工作项的每个写都经 `issue/app/lock.go` 的 `Locks.run`，按这个次序：
1. 不加锁读工作项（写指名它时）；
2. 预检：经 `Locks.decide` 回答判定会给的每个 404/403，不取锁、不开事务、不进闸门；访客规则的 404 在任何 403 之前；
3. 这个写到达的每个项目的写闸门：按 `id` 升序、一次全部取得（`Keyed.Enter(ctx, keys...)`）；等闸门时不持有连接和数据库锁；失败时放回取得的每个位置；事务结束后离开；
4. 开事务，项目行 `FOR NO KEY UPDATE`：一个项目用 `Lock`，多个用 `LockMany`（一条语句，`id` 次序）；从不锁工作区行；
5. 在锁下重读；
6. 在锁下再判定（以这次为准）；
7. 按裁定的次序拒绝（已归档的项目 → 已归档的工作项 → 无改动），然后是相关的检查，全部问题在一个 422 里；
8. 在最后一把锁之后读时钟；
9. 写：这个请求写的每一行同一个时刻、同一个写者。

P3 加了这条路的两种形式，不是第二条路：
- `lock_across.go`（加关联）：调用者看不到的另一端不到达，它的项目不进闸门、不加锁，拒绝与不存在的相同；
- `lock_either.go`（删除关联）：先判定 `issue_id` 一端，被拒绝时再判定另一端，预检通过之后两个项目都进闸门、都加锁。

P8 草稿已按设计3.5的既定形式补齐：`DraftLocks`组合上述`Locks`并复用授权、闸门与写事务，不在各用例另接一套。保存/修改/删除为工作区S → 旧新项目S（去重升序）→ 草稿U；发布先在无连接、无锁时进入项目闸门，再工作区S → 项目N → 草稿U。等待后重读、权限重判、全部锁后读时钟照旧，普通工作项写仍不取工作区锁。具体端口见[P8 spec](../M4-issue-core/specs/P8-drafts-search.md)，T7真实PID/交错16/17、T15 race五遍及首次整审证据见[P8 review](../M4-issue-core/reviews/P8-drafts-search-review.md)；文档a65e8605已独评Approved/D1 ADDRESSED（task-16-review.md），最终准确HEAD CI及集成按后续收据追踪，尚未合并。

另外：
- 503 `server_busy`（`Retry-After: 1`）只在 `bootstrap/gate.go` 映射，每个工作项的写都声明它。
- 新的写使用相应的集中编排，不在用例里写近似副本，不放宽架构守卫。

**访客规则**（设计 3.4、6.7）：
- 访客看不到的工作项，在每个指名它的操作上回答 404，从不 403；与不存在的**逐值**相同（码、体、头、数目、次序、422 的位置都不透露）。
- 判定只有一个谓词 `domain.Issue.HiddenFrom`，不复制"创建者 ≠ 调用者"。
- 列表和计数只算调用者在各项目里能看到的：`sub_issues_count`、`listSubIssues`、`listIssueRelations`、内嵌的 `Issue`。
- 回答里不指名看不到的工作项：`seen` 把看不到的父工作项的 `parent_id` 回答为 `null`。
- 负责人裁定 A（2026-10-24）：修改时 patch 里的父工作项与"调用者看到的父工作项"比较，看不到的当前父工作项算作没有。
- 管理员兼访客：在项目里是访客的工作区管理员通过项目的规则；`guest_view_all_features=false` 时 `OwnOnly` 仍为真，设为 true 时 `OwnOnly=false`。是否隐藏统一依返回的 Grant 和 `Issue.HiddenFrom`，不能从访客角色直接断言只看自己。"访客不能 X"的句子对他往往不成立，要写清。
- P4 起每个读动态、评论、表情、描述版本的操作都要套用它；动态里按标识指名别的工作项的行（第 5、20–23 行）也要过滤。

**M3 的约定**（M3 设计 3.6）：
- 约定五：批量写的行按 `id` 先锁；关联行的例外见 P3 spec §3 第 4 条。
- 约定七：几个项目在一条语句里、按 `id` 一次锁住。
- 全局的表序。
- 一次改动一个时刻。
- 按目录驱动的删除测试，没有豁免。

**设计第 12 节的约束**（每个 Phase 都适用）：
1. 故事的前置数据经接口准备，不写 SQL。
2. 每段约 16 个 Task、每个约 1,500 行 plan 以内。
3. 评审敏感的内容不放在同一段。
4. 新的错误码在同一个 Task 里加文案：`web/apps/web/core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`，以及 `en`、`zh-CN` 两份 `errors.json`，然后跑 `make test-web`。
5. 字段随它的数据进入接口。
6. 每段开始的提交改关键词守卫的 phase。
7. 故事随 Phase 长大。
8. 给工作项加表的段，同时延伸三处：`deleteIssue` 的连带、删除的语句（`issue/adapter/postgres/delete_rows.go`）、e2e 的 `furnishIssueRows`。

## 5. 反复出现的坑（P1a–P3 的评审几乎每个 Task 都抓到）
**测试数据分不出性质**。数据要包含变异会错放进来的那一行、会错排出来的那个次序：
- id 不总是升序，调用者不总是最小的；
- 次序是性质时不能只比相等；
- 创建者不是删除者；
- 要证明"只有这一个"时准备两个；
- 父子要两层以上；
- 关联要两个方向、每种存储的类型；
- 项目对要两种 `id` 次序；
- 拒绝的次序要在两个交叉方向上都钉住；
- 访客与成员、自己的与别人的、管理员兼访客、已归档与否、已删除与否都要有。

**其他测试的坑**：
- **只在单元一层的理由常常是假的**：组合测试的每个请求都经 `call`，它的 `CheckResponse` 按契约的 `x-problem-codes` 跑 `checkProblemCode`，契约的变异在组合一层就会失败。说"只能在单元一层"之前，先在更高的层跑一遍。
- **时间**：
  - 等待按条件、有会失败的期限，不用固定的 sleep 同步任何东西；
  - 时间的断言要在慢的、负载高的、`-race` 下的机器上也成立：下界来自不会提前触发的期限，上界放在对与错的行为之间。
- **锁等待**：新测试用 `postgres.LockNotAvailable` 和 `pgtest` 的帮手（含 `pgtest.RowWaits`），不手写 `55P03` 的判断。
- **复制的测试帮手**：扩展已有的，不重写。每种动态行的形状只有一个构造者。已有的帮手：
  - `bootstrap`（分散在几个 `_test.go` 文件里，用 grep 找定义）：`issueWorld` 和它的 `newWorkItem`、`newWorkItemIn`、`makeMember`，`workItemStamps`、`activityBy`、`idArray`、`names`、`createdProject`/`createdProjectFrom`、`keyOfRow`、`relating`、`subIssueIDs`、`listIn`；交错的 `itemWrite`、`wroteNothing`、`serializeOnTable`、`serializeAt`、`inBothOrdersAt`、`heldTx`、`holding`；矩阵的 `matrixRow.prepare`。
  - `issue/app`：`wantStopped*`、`wantRefusal`/`wantRefused`、`wantActivityRows`、`wantNothingWritten`，假实现的 `lost`/`lose`，`ascending`，`held.own()`，`deleteOver`/`archiveOver`。
  - e2e：`fixtures/assert/issue.ts` 的 `byId`、`activityRow`（`changed`、`parentRow`、`relatedRow`、`unrelatedRow`、`createdFor` 都经它）、`madeBy`、`expectIssueArchival`、`expectRelations`；`assert/catalog.ts` 的 `tablesUnder`、`keysBelow`。

**文档的坑**：
- 句子比代码宽：Go 文档（尤其是哨兵错误的文档）、契约描述、测试的文档、交接的处理结果、plane-diff 的行、设计 5.1 的码列，都要与代码逐句相符。
- 不写面向未来的注释，不写会过时的数目、行号。
- 文档不写代码没有提供的路由。
- 错误里不列出一长串 id：错误覆盖一个列表时写数目。

**plan 的大小**：修复轮约增长 5–10%，切 Task 时留余量。

## 6. 环境的坑
- **Firefox**：早期执行环境没有Firefox，这是历史事实而非永久豁免。P7使用项目内Playwright 1.63缓存，T15准确tested `6e28cc16`完整E2E实测153=151Chromium+2Firefox通过；之后按当次缓存/运行结果记录，不全局安装、不把缺浏览器当性质红。C12/WebKit仍由M4收尾核验。
- **负载**：别的会话常在跑容器。第一次运行时容器起不来、套接字超时、Ryuk 不见了、页面故事 5 秒期望超时，都出现过。Docker 空闲时重跑一次再下结论，结果连同数字记下来。
- **持续集成比本机慢**：e2e 那一步约 5 分钟，本机约 2 分钟。持续集成在 Linux 上，`ControlOrMeta` 是 Ctrl。本机 macOS 上通过，不等于持续集成通过。
- **副本**：在 `git archive` 出来的副本里跑时：
  - S3 会失败（构建没有提交号）；
  - `server/internal/platform/webui/dist/.gitkeep` 和 `i18n/locales` 的链接可能要补回；
  - `gen-check` 跑不了（没有 git），要手工比较生成物。

## 7. 怎么看持续集成

有产品变化的开发检查点先完整本地CI，再推分支；逐Task聚焦/受影响回归继续。现行核验先按Git准确40位SHA查Actions run的head_sha、branch与三个job/步骤终态，保存公开JSON/HTML原件及摘要。分支CI不替代mainCI；main合并须另获授权。T12因浏览器安装耗时后整job取消保留为历史失败，T15准确d2的CI40387396589全绿也不能替代T16最终新HEAD。下列旧脚本只作已有工具导航，使用前核可用性；读公开元数据不需要提取凭据，受限raw日志/产物仍按实际权限处理。
- 公开的 Actions 页面（`https://github.com/open-nerve/NerveProject/actions?query=branch%3Amain`）列出每次运行的标签，形如"failed: Run 340 of CI. Merge M4/P3 …"。
- 本机有现成的脚本（[05](05-local-materials.md)）：
  - `wait-page.sh <分支> '<标题开头>'`：每分钟看一次，直到结束，在后台跑；
  - `ci-labels.sh <分支>`：最近几次的标签；
  - `ci-run-jobs.sh <run id>`、`ci-job.sh <run id> <job id>`：作业和步骤的结论；
  - `ci-annotations.sh <job id>`：注解。
- 标签在约 90 个字符处截断，标题开头要短；开头里的 `(`、`)` 要转义，`'` 写成 `&#x27;`。
- 公开的 API `https://api.github.com/repos/open-nerve/NerveProject/actions/runs/<id>/jobs` 给出步骤的时间和结论，不用凭据。
- 日志和 Playwright 报告要找负责人要。

## 8. 只属于 Claude 执行环境的规则（Codex 不必照搬）
Claude 的会话有这些额外的限制，都来自它的权限设置，不是项目的要求：
- 每次只跑一条 git 命令、不串联；
- 不用 `cd` 和 `git -C`；
- 路径里不用 shell 变量；
- 循环和 Python 写成脚本文件再跑；
- 用 `git commit-tree` 做合并提交。

它们留下的痕迹是 plan 里的命令都写绝对路径。
