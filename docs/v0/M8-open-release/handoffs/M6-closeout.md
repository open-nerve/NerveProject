---
status: open
from: M6/closeout
to: M8
created: 2026-11-07
---

# M6 交给 M8：CI 的 e2e 作业期限与间歇失败的故事、清理分页器的副本、e2e 数据库的连接余量、server 作业的分路与 lint 工具的在线下载、本机测试容器的端口冲突

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页问题已经解决。M6 进行中发现的条目随发现写入；M6 收尾时补齐其余交接。延续 [M5 → M8 交接 M5-M8-06](M5-closeout.md#M5-M8-06) 中仍开放的观察。

<a id="M6-M8-01"></a>
## M6-M8-01 e2e 作业期限由 25 改为 35 分钟，C7 与 C12 再次间歇失败

- M6 设计合入的 main CI 42281169459（main `d1ec8c69`，只改文档；同一内容的分支 CI 全绿）连续两次未通过：
  - attempt 1：e2e 417 条中 1 条失败，M4 的 C7（`e2e/stories/issue/collaboration-interactions.spec.ts:422`，键盘选主题）。选 Dark 收到 503 后，第一次 `reopenDuringExit("Control+k")` 期待弹窗面板停在 30 秒的慢退场（`data-leave`），5 秒内找不到面板元素。
  - attempt 2（重跑失败的作业）：安装 Playwright 浏览器一步用了 7 分 19 秒，e2e 作业在 25 分钟期限处被取消，当时故事跑到约第 314 条；取消前 M4 的 C12（`e2e/stories/issue/review-interactions.spec.ts:83` 的 kanban 布局，见 M5-M8-06）又失败一次，错误详情因取消未打印。
- 负责人 2026-11-07 批准把 e2e 作业的 `timeout-minutes` 由 25 改为 35（`.github/workflows/ci.yml`），server 作业仍为 25。依据：绿色 e2e 作业已用 23–24 分钟、故事本身 21–22 分钟（M5-M8-06），浏览器安装偶发停滞，M6 还会加入迭代与模块的故事。
- C7 与 C12 的原因当时均未查实，均为界面时序类的间歇失败；配置中没有重试，失败即算数。
- M6/P1 检查点的分支 CI 42334013393（`c528d682`，已含 35 分钟期限）：attempt 1 的 e2e 417 条中 416 过，C12 在同一处（`review-interactions.spec.ts:83`）的 list 布局失败——新建工作项弹窗 5 秒内未关闭（`toHaveCount(0)` 收到 1，`:117`）；只重跑失败的作业，attempt 2 通过（e2e 作业 16 分钟）。
- M6/P2 候选 `784d8e0a` 的分支 CI 42380907752 attempt 1：e2e 426 条中 425 过，M5 的 F16 (page)（`e2e/stories/files/f16-content-lifetime-page.spec.ts:672`，角色改变又改回的故事）失败——展开评论的“Attachments”折叠区后，5 秒内找不到 `Download owner-note.txt`（`:698`）。这与 [M5-closeout](M5-closeout.md) 所记 F16 重连故事的“点开折叠区与晚到的版面位移竞争”是同一类（该条已建议同类辅助函数一并检查）；P2 没有改动文件与评论的前端。只重跑失败的作业，attempt 2 的 F16 通过，但 I21 因 P2 自己的测试竞争失败（P2 在 `78c7f7d3` 修复，见 [P2 阶段记录](../../M6-cycles-modules/reviews/P2-modules-backend-review.md) §5），不属本条。
- M6/P2 合入的 main CI 42399566014 两次未过（自动归档的锁探测被启动时的修剪任务骗过；S3 测试容器发布端口撞上本机连接的临时端口），按负责人指示由修复分支 `codex/m6-p2-ci` 修根因而非重跑：锁探测改为只看其行为者自己的锁与等待，测试容器在端口冲突时重建（至多三次），另修本地暴露的 S7 (page) 菜单按键竞争。M5-M8 交接所记“启动时运行的周期任务插到锁测试”一条，凡全局探测项目行锁的 bootstrap 测试已按此改写；其余仍开放的观察照旧。经过见 [P2 阶段记录](../../M6-cycles-modules/reviews/P2-modules-backend-review.md) §5。
- **C12 的根因已查实并修复**（M6/P4）：P4 合入的 main CI 42809569341 的 e2e 又在 C12 的 list 布局同一处（`:117`）失败。这次从 trace 查实：弹窗刚打开时描述编辑器尚未建成，编辑器的引用却已挂上一套操作空编辑器的 API，`isEditorReadyToDiscard()` 答 false，弹窗把 Escape 当成“编辑器忙”而拒绝关闭（错误提示 “Editor is not ready to discard changes”）。修复分支 `codex/m6-p4-ci` 改为编辑器建成之前引用为 null，经过见 [P4 阶段记录](../../M6-cycles-modules/reviews/P4-progress-transfer-review.md) §5。此前留有详情的各次 C12 失败（M5-M8-06 与本条上文）都是“弹窗打开后不久按 Escape、5 秒内未关”，与此一致。收集箱新建弹窗的同类问题交 M7（[M6-M7-01](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-01)）。

**何时验收：** M8 的 CI 工作复核 35 分钟的余量（或缓存浏览器的系统依赖、拆分作业，任一调整须负责人同意），并查实或修好 C7；在此之前，C7 再失败时按“重跑一次，再不过就停下报告”处理。C12 若再失败，按新的问题定位，不按本条重跑。

<a id="M6-M8-02"></a>
## M6-M8-02 三个模块逐字相同的清理分页器

`planning/app/purge.go` 是 `issue`、`project` 两个模块的同名文件的第三份逐字副本（约 100 行，M6/P1 台账的裁定：模块互不 import，沿用先例）。M6/P1 的整分支审查（[re-55](../../../reviews/re-55-M6P1整体审核.md)）指出，有三份副本时，抽成一个平台包（例如 `platform/purge`）已比重复便宜。P1 不改：它要同时改三个模块与 bootstrap 的清理接线，不属本阶段。M6 其余阶段的模块表仍归 `planning`，不再新增副本。

**何时验收：** M8 的清理工作抽出共用的分页器，三个模块改用它，`TestPurgeScale` 与清理守卫照常通过；或负责人决定保留副本，并在此写明理由。

<a id="M6-M8-03"></a>
## M6-M8-03 e2e 共用的 Postgres 连接余量

M6/P2 的本地 e2e（`make build && make e2e`）两次各有一条因 Postgres 的 `FATAL: sorry, too many clients already`（SQLSTATE 53300）失败，同一输入只重跑这一项即 426 条全过：候选 `254d3255` 上 W6（`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts:74`）的两次注册拿不到连接而回答 500；候选 `78c7f7d3` 上 W12（`e2e/stories/workspace/w12-reactivate-member.spec.ts:143`）的 `nerve workspaces reactivate-member` 命令连不上数据库。W6 那次本机 Docker 同时运行二十余个其他项目的容器，同一候选的前一轮 `make test` 也有两个包因测试容器起不来而失败（环境失败，重跑通过）。P2 没有新增连接池或任务队列，每个进程的连接上限没有变。

连接预算（按配置核算，未逐个计数）：全部 Playwright 工作进程共用一个 `postgres:18.6` 测试容器（`e2e/fixtures/db.ts`），用默认的 `max_connections`（100）；工作进程数取 Playwright 的默认值（CPU 数的一半），本机 18 核为 9 个，CI 运行器 4 核为 2 个；每个工作进程的 Nerve 服务器连接池上限 10（`server/configs/config.yaml` 的 `database.max_conns`），测试一侧的 pg 连接池默认上限 10，另有 River 的连接。本机 9 ×（10 + 10 + River）远超 100，平时用不满，机器繁忙、各池同时涨到上限时就会用尽；CI 约 2 ×（10 + 10 + River），有余量，分支与 main CI 都没有出现过。M6 加入的模块数据、链接抓取任务与故事里的数据库轮询让本机更容易撞线。证据在 M6/P2 归档的台账（`t15-gate4-build-e2e`、`t15-gate5-build-e2e`）。

M6/P3a 的最终候选 `070762f5` 的本地门禁（`t13-gate1-build-e2e`）又一次撞上：W6（API）“a deleted invitation”一步回答 500，该测试 nerve 日志为 `FATAL: sorry, too many clients already`（53300）；只重跑这一项即 429 条全过（`t13-gate1-build-e2e-2`）。同一候选的分支 CI 不受影响。

M6/P5a 的最终候选 `5b5da6ea` 的本地门禁（`t16-gate3-build-e2e`）再一次：C12 的 LAN HTTP 一例（`e2e/stories/issue/collaboration-interactions.spec.ts:556`）页面上 `GET /api/v0/workspaces/<slug>/states` 回答 500，该测试 nerve 日志为 `FATAL: sorry, too many clients already`（53300）；只重跑这一项即 29 条全过（`t16-gate3-e2e-c12-rerun`）。按 M6-M8-01 对 C12 的要求按新问题定位，查实是本条，不是 M6-M8-01 的间歇失败。

**何时验收：** M8 的 CI 工作核算 e2e 的连接预算（工作进程数 × 每进程的连接上限 + 余量），按结果给测试容器设 `max_connections` 或收紧 e2e 服务器的 `database.max_conns`，并在 CI 与本地都留有余量；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-04"></a>
## M6-M8-04 server 作业：Go 测试分两路并行，lint 工具每次在线下载

- `internal/bootstrap` 包把数百个组合测试串行跑在一个测试二进制里，受 `go test -timeout=20m` 的整包期限约束（`Makefile` 的 `GO_TEST_TIMEOUT`）。[M5-closeout](M5-closeout.md) 记它在 CI 上为 756–978 秒；到 M6/P2，最近 12 次 CI 为 919–1185 秒（M6/P1、P2 各加了测试；本地约 840–860 秒，CI 运行器慢约 15–40%）。修复分支 `codex/m6-p2-ci` 的分支 CI 42413098044 中它超过 1200 秒被终止（`panic: test timed out after 20m0s`），修复分支本身没有让它在本地变慢（修复前 840–855 秒，修复后 833–848 秒）。
- 负责人 2026-11-08 批准把 CI 的 `server` 任务按矩阵分两路（`.github/workflows/ci.yml`，`make test-shard`，`tools/go-test-shard.sh`）：bootstrap 包的测试按名字排序后轮流分到两路，其他包和 `server/tools` 只在第 0 路；生成物检查与 lint 只在第 0 路。两路合起来与 `make test` 相同，脚本核对每路的 `-run` 模式恰好选中分到的测试。整包期限 20 分钟和 `server` 任务期限 25 分钟都没有改。按本地逐个测试的计时（644 个测试共 831 秒），两半分别约 450 与 382 秒。
- 本地 `make test` 照旧整包串行运行。
- 分路后 CI 报告的检查名是 `server (0)`、`server (1)`，不再有 `server`；按 [M0-P6 交接](M0-P6-release-notes.md) 设分支保护的必须检查时以此为准（那里建议的汇总任务也要依赖两路）。
- **lint 工具每次从 GitHub 下载**（M6/P3a 的观察）：`server (0)` 的 Lint 一步是 `make lint-go`，它先跑 `make tools`；CI 的运行器上没有 `./bin`，`tools` 目标每次都经 `raw.githubusercontent.com` 上的 `install.sh` 从 GitHub 下载锁定版本的 golangci-lint（`Makefile` 的 `GOLANGCI_LINT_VERSION`，现为 2.13.2；工作流只缓存 pnpm 的存储）。M6/P3a 检查点 `a37bc6370ae651118bd9cb6204279679fdd5031c` 的分支 CI 42478883539 attempt 1 中，`server (0)`（作业 128726250098）在这次下载时收到 GitHub 的 HTTP 504 而失败，lint 没有开始，与代码无关；`server (1)` 与 `web` 通过，`e2e` 随之跳过。根因是外部下载故障，只重跑失败的作业一次，attempt 2 的 `server (0)` 通过（Lint 一步 84 秒）。即 server 作业的每次运行都依赖一次在线的 GitHub 下载；按版本缓存锁定的二进制（例如以 `GOLANGCI_LINT_VERSION` 为键的 Actions 缓存）可去掉这一外部依赖。

**何时验收：** M8 的 CI 工作复核每路的余量（bootstrap 每半与 20 分钟整包期限、各路与 25 分钟任务期限）；需要时增加路数，或把 bootstrap 的测试改为可并行运行。lint 工具不再每次在线下载（例如缓存锁定版本的二进制），或负责人决定维持现状并在此写明理由。改路数、期限、作业结构或缓存，任何 CI 的改动都须负责人同意。

<a id="M6-M8-05"></a>
## M6-M8-05 本机测试容器的主机端口与 macOS 的临时端口冲突

M6/P5a 的最终候选 `5b5da6ea` 上，本机 `make test` 连续两次失败，各有包在 testcontainers 启动容器时等映射端口约 60 秒到期限（`wait until ready: mapped port … get state: GET /containers/<id>/json: context deadline exceeded`），容器内的服务已就绪：`t16-gate2` 是 `asset/adapter/s3`，`t16-gate3` 是 `platform/postgres`（16 例）与 `migrations`。Docker Desktop 的 `~/Library/Containers/com.docker.docker/Data/log/host/com.docker.backend.log` 在这些时刻记有该容器开端口转发失败：“listen tcp 0.0.0.0:56932（56934）: bind: address already in use”。

根因：Docker Desktop 的 VM 从自己的 `ip_local_port_range`（55000–65535）给容器分配随机主机端口，再在 macOS 上监听同一个端口；macOS 自己的临时端口（`net.inet.ip.portrange.first`/`last`）是 49152–65535，两者重叠，分到的端口可能已被本机的出站连接或监听占用。`make test` 并行启动大量容器、开大量连接，撞上的机会随之增加。同一警告此前只在 2026-11-10 的两个时刻出现；M6/P2（`254d3255`）与 M6/P3b（t14-gate1）的“Docker API 超时、容器拿不到映射端口”是同一签名。Linux 的 CI 没有这一层端口转发，不受影响。

负责人 2026-11-11 决定：P5a（没有 Go 改动）的服务端测试以分支 CI 为准，不改本机设置，不改测试代码，交 M8。

**何时验收：** M8 在本机门禁的说明里给出不重叠的做法并验证（例如测试容器助手自己在本机选空闲端口并显式绑定、在端口转发失败时换端口重启容器，或在 README 写明把 macOS 的临时端口范围收到 55000 以下），使本机 `make test` 不再因此失败；或负责人决定维持现状，并在此写明理由。
