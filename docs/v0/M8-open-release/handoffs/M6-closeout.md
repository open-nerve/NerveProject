---
status: open
from: M6/closeout
to: M8
created: 2026-11-07
---

# M6 交给 M8：CI 的 e2e 作业期限与间歇失败的故事、清理分页器的副本、e2e 数据库的连接余量

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页问题已经解决。M6 进行中发现的条目随发现写入；M6 收尾时补齐其余交接。延续 [M5 → M8 交接 M5-M8-06](M5-closeout.md#M5-M8-06) 中仍开放的观察。

<a id="M6-M8-01"></a>
## M6-M8-01 e2e 作业期限由 25 改为 35 分钟，C7 与 C12 再次间歇失败

- M6 设计合入的 main CI 42281169459（main `d1ec8c69`，只改文档；同一内容的分支 CI 全绿）连续两次未通过：
  - attempt 1：e2e 417 条中 1 条失败，M4 的 C7（`e2e/stories/issue/collaboration-interactions.spec.ts:422`，键盘选主题）。选 Dark 收到 503 后，第一次 `reopenDuringExit("Control+k")` 期待弹窗面板停在 30 秒的慢退场（`data-leave`），5 秒内找不到面板元素。
  - attempt 2（重跑失败的作业）：安装 Playwright 浏览器一步用了 7 分 19 秒，e2e 作业在 25 分钟期限处被取消，当时故事跑到约第 314 条；取消前 M4 的 C12（`e2e/stories/issue/review-interactions.spec.ts:83` 的 kanban 布局，见 M5-M8-06）又失败一次，错误详情因取消未打印。
- 负责人 2026-11-07 批准把 e2e 作业的 `timeout-minutes` 由 25 改为 35（`.github/workflows/ci.yml`），server 作业仍为 25。依据：绿色 e2e 作业已用 23–24 分钟、故事本身 21–22 分钟（M5-M8-06），浏览器安装偶发停滞，M6 还会加入迭代与模块的故事。
- C7 与 C12 的原因仍未查实，均为界面时序类的间歇失败；配置中没有重试，失败即算数。
- M6/P1 检查点的分支 CI 42334013393（`c528d682`，已含 35 分钟期限）：attempt 1 的 e2e 417 条中 416 过，C12 在同一处（`review-interactions.spec.ts:83`）的 list 布局失败——新建工作项弹窗 5 秒内未关闭（`toHaveCount(0)` 收到 1，`:117`）；只重跑失败的作业，attempt 2 通过（e2e 作业 16 分钟）。

**何时验收：** M8 的 CI 工作复核 35 分钟的余量（或缓存浏览器的系统依赖、拆分作业，任一调整须负责人同意），并查实或修好 C7 与 C12；在此之前，这两条再失败时按“重跑一次，再不过就停下报告”处理。

<a id="M6-M8-02"></a>
## M6-M8-02 三个模块逐字相同的清理分页器

`planning/app/purge.go` 是 `issue`、`project` 两个模块的同名文件的第三份逐字副本（约 100 行，M6/P1 台账的裁定：模块互不 import，沿用先例）。M6/P1 的整分支审查（[re-55](../../../reviews/re-55-M6P1整体审核.md)）指出，有三份副本时，抽成一个平台包（例如 `platform/purge`）已比重复便宜。P1 不改：它要同时改三个模块与 bootstrap 的清理接线，不属本阶段。M6 其余阶段的模块表仍归 `planning`，不再新增副本。

**何时验收：** M8 的清理工作抽出共用的分页器，三个模块改用它，`TestPurgeScale` 与清理守卫照常通过；或负责人决定保留副本，并在此写明理由。

<a id="M6-M8-03"></a>
## M6-M8-03 e2e 共用的 Postgres 连接余量

M6/P2 最终候选 `254d3255` 的本地 e2e（`make build && make e2e`）第一次运行 426 条中 425 条通过，W6（`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts:74`）的两次注册回答 500：服务器日志是 Postgres 的 `FATAL: sorry, too many clients already`（SQLSTATE 53300），注册请求拿不到连接。同一输入只重跑这一项即 426 条全过。当时本机 Docker 同时运行二十余个其他项目的容器，同一轮的 `make test` 也有两个包因测试容器起不来而失败（环境失败，重跑通过）。P2 没有新增连接池或任务队列，连接数的理论上限没有变；此前两轮同样含模块代码的候选 e2e 都是 426 条全过，分支 CI 也没有出现过。

原因推断（未查实）：全部 Playwright 工作进程共用一个 `postgres:18.6` 测试容器（`e2e/fixtures/db.ts`），它用默认的 `max_connections`（100）；每个工作进程有自己的 Nerve 服务器连接池、River 的连接与测试一侧的连接池，平时用不满，机器变慢时各池同时涨到上限便可能超过 100。证据在 M6/P2 归档的台账（`t15-gate4-build-e2e`）。

**何时验收：** M8 的 CI 工作核算 e2e 的连接预算（工作进程数 × 每进程的连接上限 + 余量），按结果给测试容器设 `max_connections` 或收紧 e2e 服务器的 `database.max_conns`，并在 CI 与本地都留有余量；或负责人决定维持现状，并在此写明理由。
