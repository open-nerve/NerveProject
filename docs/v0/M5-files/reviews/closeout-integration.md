# M5 收尾：完成标准证据与集成记录

> 收尾合同：[closeout spec](../specs/closeout.md)、[plan](../plans/closeout.md)。本页先列[设计 §14](../M5-design.md#14-完成标准) 十八条的证据，首次 main CI 成功后才在设计中勾选；第二次 main CI、归档与清理的结果写在仓库外的集成收据（§3 给出位置），不由被它验证的提交预报。

## 1. 各阶段的合入与证据

| 阶段 | main 上的 noff 合并 | 阶段记录 | 整体审核 | 归档根（仓库外） |
|---|---|---|---|---|
| 设计与 P1 预检 | `fb3e788b` | — | [re-31](../../../reviews/re-31-M5设计与P1预检.md) | `M5-design-integration-20261102` |
| P1 本地存储 | `85dc4087` | [P1](P1-local-storage-review.md) | [re-33](../../../reviews/re-33-M5P1整体审核.md) | `M5-P1-integration-20261102` |
| P2 资产与清理 | `2d1bcf0c` | [P2](P2-assets-review.md) | [re-35](../../../reviews/re-35-M5P2整体审核.md) | `M5-P2-integration-20261103` |
| P3 S3 | `5807a0d6` | [P3](P3-s3-storage-review.md) | [re-37](../../../reviews/re-37-M5P3整体审核.md) | `M5-P3-integration-20261103` |
| P4 资料图片与引用 | `0f93100a` | [P4](P4-profile-images-review.md) | [re-39](../../../reviews/re-39-M5P4整体审核.md) | `M5-P4-integration-20261103` |
| P5 工作项内容资产 | `c59cedf3`、`ce29aa00`、`f5f4ce94` | [P5](P5-issue-assets-review.md) | [re-41](../../../reviews/re-41-M5P5整体审核.md)、[re-43](../../../reviews/re-43-M5P5集成修复审核.md)、[re-44](../../../reviews/re-44-M5P5作业预算审核.md) | `M5-P5-integration-20261104`、`M5-P5-ci-purge-20261104`、`M5-P5-e2e-budget-20261104` |
| P6 资料图片上传 | `96363675` | [P6](P6-profile-upload-review.md) | [re-45](../../../reviews/re-45-M5P6整体审核.md) | `M5-P6-integration-20261104` |
| P7a 草稿准备件 | `cf07c9fb` | [P7a](P7a-draft-preparations-review.md) | [re-48](../../../reviews/re-48-M5P7a整体审核.md) | `M5-P7a-integration-20261105` |
| P7b 附件与编辑器图片 | `2b2c6f33` | [P7b](P7b-content-assets-review.md) | [re-50](../../../reviews/re-50-M5P7b整体审核.md) | `M5-P7b-integration-20261106` |
| P8 最终验收 | `d8ec8457` | [P8](P8-final-acceptance-review.md) | [re-52](../../../reviews/re-52-M5P8整体审核.md) | `M5-P8-integration-20261106` |

归档根都在主目录 `.superpowers/sdd/` 下（git 排除），各阶段记录写有其 manifest SHA-256 与清理收据。

## 2. 设计 §14 十八条的证据

| # | 条款（简） | 证据 |
|---|---|---|
| 1 | F1–F18 各有与失败路径相区分的 API/字节/页面证据，未把 mock 当真实存储 | P8 阶段记录 T12 的 F1–F18 矩阵（每条指向本次运行的测试名与原件）；演练与 S3 故事用真实 local 与 SeaweedFS；re-52 重点 5 PASS |
| 2 | local 默认可运行；HMAC/路径/期限/超限/并发/重启 | P1 阶段记录与 re-33；P8 T11 默认生产部署演示（F18，三浏览器，重启后字节仍在） |
| 3 | s3 真实兼容服务与 local 共用合同，配置/CORS/CSP 与支持边界 | P3 阶段记录与 re-37；`deploy/s3/README.md`；P8 T12 的 `m5-s3` 三浏览器；[M8 交接 M5-M8-01](../../M8-open-release/handoffs/M5-closeout.md#M5-M8-01)（只实测 SeaweedFS 4.52） |
| 4 | 已确认对象不可被旧上传 URL 覆盖；并发、崩溃、未知外部结果与孤立对象清理 | P2、P3 阶段记录（封存复制、条件写、对账）；P8 T10 全链路演练（失败重试与孤立对象回收） |
| 5 | 全部资源用途、宿主/角色矩阵、复制源与目标、准备件绑定授权 | P2、P5、P7a 阶段记录；P8 矩阵 F13 行 |
| 6 | users/workspaces/projects 引用、成员头像、稳定封面一次写入、organization_size | P4、P6 阶段记录；P8 矩阵 F5–F7 行 |
| 7 | 工作项/评论/草稿附件、真实 attachment_count、排序与显示 | P5、P7b 阶段记录；P8 矩阵 F8、F9 行 |
| 8 | 草稿发布的创建、资产迁移、软删与审计同事务；唯一发布与交错失败 | P5、P7a 阶段记录；P8 矩阵 F10 行 |
| 9 | 图片清洗、有序 src 内容标识、skip_activity、历史/撤销与持久身份 | P5、P7b 阶段记录；P8 T2、T3；P8 矩阵 F11、F12 行 |
| 10 | 删除入口、表目录/外键图、有限批次、原 At/By、60 天 purge 与存储重试 | P2 阶段记录与 [purge 合同决定](P2-purge-contract-decision.md)、[P5 清理续跑](P5-ci-purge-resume.md)；P8 T10 演练；P8 T12 的 `TestPurgeScale`；P8 矩阵 F14、F15 行 |
| 11 | 传输/渲染/store/弹窗在换身份、路由、撤权和迟到回包时失效 | P6、P7b 阶段记录；P8 矩阵 F16 行 |
| 12 | 中英文错误、键盘/读屏/焦点与各种交互的三浏览器核验 | P6、P7b 阶段记录（T13 真实页面）；P8 T12 的 `m5-profile`、`m5-content` 三浏览器；P8 矩阵 F17 行 |
| 13 | 无旧文件 API、死 services 包、到期关键词例外；死成员逐条核实；改动文件 oxlint 为零，新规则全仓清零 | P7b（删除旧文件链与 `@nerve/services`）；P8 T7（死成员逐行闭环）、T8（两条规则清零并设为 error）、T12 Step 2 扫描 |
| 14 | Plane 差异、前端改动、配置/部署说明、素材 SOURCES 与原始证据同步 | P8 T13（README、plane-diff、frontend-changes、交接）；素材 `SOURCES.md`（brand、cover-images、onboarding、project-covers）；§1 各阶段归档 |
| 15 | 输入交接 closed 并有落点；M6/M7/M8 收到 M5-closeout；首个 CLI 与 copy 风险未丢 | 收尾 T2（七份输入关闭）；P8 T13 的三份 `M5-closeout`，M5-04 原文三份各一；旧 copy 方法 P8 T7 删除，复制为 Markdown 省略图片写入 M7/M8 |
| 16 | 最终本地门禁、deep-review 与修复复审、精确分支/main CI 通过；历史失败诚实保留 | P8 T14、T15（re-52）；P8 准确候选分支 CI 42188195625（attempt 3）与 main CI 42212370678（attempt 2）；P8 阶段记录 T14、T16 保留全部失败与重跑 |
| 17 | 全阶段与收尾 noff 合入，完成标记第二次 noff 后 main 全绿，总体 §9.4 完成 | §1 各阶段合并；收尾首次与第二次 noff（§3）；[总体设计 §9.4](../../v0-design.md#94-里程碑进度表) |
| 18 | 原始证据完整归档并验证 hash/bytes/mode/membership，分支与 worktree 正常清理、未知文件为零 | §1 各阶段归档根与清理收据；收尾 T6 |

## 3. 收尾集成

### 3.1 预检

一次独立 builder 只读核对收尾草案、P8 归档与 GitHub 运行记录，结论“补实 12 项后可实施”，全部为执行者可定的局部问题，没有需负责人决定的项（P8 候选的第二次重跑已由负责人在 2026-11-06 决定）。12 项的落点见 [re-53](../../../reviews/re-53-M5收尾预检.md)；原文保存在收尾工作区 `.superpowers/sdd/M5-closeout/preflight-raw.md`，随 T6 归档。

### 3.2 首次集成

准确候选 `9d19478e`（9d19478e927a9adbcba71c0c598922e81f075920，T1–T3 三个提交）在本地通过 `doc-links.py`、关键词守卫与 `make lint-web`，其[分支 CI 42224350248](https://github.com/open-nerve/NerveProject/actions/runs/42224350248)（attempt 1）三项全绿：web 127913955833、server 127913956427、e2e 127922168723（E2E 417/417，作业 16.6 分钟）。根 main 原为 `d8ec8457`，与 origin 一致且干净，worktree 干净，本地与远端分支都是该候选；`--no-ff` 合并生成 `1831dabc`（1831dabcd0687c60c2102572c2af85f205c3cb52），双父依次为 `d8ec8457` 与候选，完整 tree 与候选相同，首次推送因网络连接被关闭失败，同一推送重试成功。准确 main CI 的 attempt 1 中 web 与 server 成功，e2e 416/417：M4 的“I4 (page): table label…”故事在选“Z A”后 5 秒内找不到“Labels”表头——trace 显示换排序时整个表格（含表头）换成骨架屏，数据 20 毫秒返回后骨架仍显示约 5.3 秒才重新渲染（同一故事的“A Z”一步约 3.5 秒），与只改文档的本次合并无关；按负责人规则重跑一次失败作业，[main CI 42229175694](https://github.com/open-nerve/NerveProject/actions/runs/42229175694)（attempt 2）三项全绿：e2e 127952547976、web 127952627789、server 127952549059（E2E 417/417，测试 21.7 分钟、作业 23.9 分钟）。这一机理写入 [M8 交接 M5-M8-06](../../M8-open-release/handoffs/M5-closeout.md#M5-M8-06)。此后 §2 各条在设计 §14 中勾选、总体 §9.4 标为已完成；收尾分支先快进到 `1831dabc` 再写完成标记，第二次合入因此仍是以 main 为祖先的 `--no-ff`。

### 3.3 第二次集成、归档与清理

第二次 main CI、收尾的全量归档与正常清理发生在被它们验证的提交之后，结果写在仓库外的集成收据：主目录 `.superpowers/sdd/M5-closeout-integration-20261106/`（git 排除），与归档 payload 并列而不在其中。设计 §14 第 17、18 条与[收尾合同 §4](../specs/closeout.md#4-验收标准) 的最后两项由这份收据签结。
