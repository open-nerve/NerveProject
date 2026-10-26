# M4/P7 链接、出站与抓取预检评审记录

日期：2026-10-26。状态：**独立预检及修订复审通过，5 项 Medium 全部闭环；T1–T16 未实施**。分支 `codex/m4-p7`；基线为 P6 的 `--no-ff` 合并 `5005be985e7ec78727edc6f8fc5c05999b45cc65`。配套：[spec](../specs/P7-issue-links.md)、[plan](../plans/P7-issue-links.md)。P6 main CI 首次浏览器安装超时取消，重跑后三任务全绿，证据归档和分支/worktree 清理均完成。

## 1. 评审方式

按 [02-method §3](../../claude_to_gpt/02-method.md#3-哪些是要求哪些是做法) 采用紧凑 spec/plan，在已批准 M4 架构下进行独立文档预检。作者先盘点范围和真实接缝，再完成 writing-plans 自查；主控与独立安全、集成审者分别核对逻辑、跨层接线和测试性质。文档逐 Task 的真实红绿、生产变异、恢复、独评属于后续实施，不把本次预检写成完整原型或逐字重放。

- 安全审查全文覆盖 spec/plan，并追踪出站策略、配置与 CLI 隔离、crawler 限额、任务结果映射、事件投递；依据现存配置/架构守卫、批准设计与官方 IANA 快照。
- 集成审查全文覆盖 spec/plan，并追踪两路删除、Locks/授权、Store/审计、HTTP/错误文案、bootstrap/module/worker/订阅者、registrationRows、runtime grants 与 I10/I17 fixtures；核对现存调用方和对应测试。发现长 URL 物理约束冲突后另做最窄可行性实验。
- 主控核对范围、前序集成、OwnOnly 条件、签名、任务依赖、文档事实和静态门禁，合并发现并裁定局部修复。

原始报告、冻结稿 SHA 与实验输出保存在本地 ignored `.superpowers/sdd/P7-issue-links/`，不是已随 Git 发布的正文附件。整合报告见 [P7 预检审核](../../../reviews/re-04-M4P7预检审核.md)。

## 2. 验收标准逐条核对

P7 spec §4 的全部产品/集成验收全部未勾选。准备期只核可实施性，不声称链接 CRUD、SSRF、worker、事务、交错或页面故事已经通过。

| 本次实际检查 | 结论与范围 |
|---|---|
| `node tools/keywords.mjs` | 72 规则、23 例外、零命中；phase 改为 M4/P7，无该阶段到期例外，不延长现有 until。 |
| `GOFLAGS=-p=1 go test -count=1 ./internal/archtest` | exit 0；只证明当前准备分支未破坏架构守卫，不是未来产品代码通过。 |
| 最窄 PG18.6 实验 | 单个 `TestP7ExactURLIndexProbe` 真实执行通过，原 B-tree 拒绝合法长 URL，替代约束满足本表下列结果。 |
| 离线 sqlc v1.31.1 生成 | 自带 schema/query 的生成 exit 0；不替代正式 12 列迁移、外键、grants 与产品查询生成。 |
| 文档与差异检查 | 11 份 Markdown、116 条现存本地链接、16 Task、21 项验收全未勾、表格列数及 `git diff --check` 均通过；4 条基线已存在的 M5–M8 最终 closeout 链接仍按设计留待 M4 收尾，不提前造文件。 |

### 2.1 长 URL 约束实验

原设计同时允许 2048 Unicode 字符、要求 `(issue_id,url)` 原文 B-tree 部分唯一。在本实验 PostgreSQL 18.6 / 8192 字节页上，2048 字符、8135 UTF-8 字节的不可压缩合法 URL 被拒绝：SQLSTATE `54000`，索引行 8168 字节超过 2704 字节上限。这个结果与 [PostgreSQL 的 B-tree 容量约束](https://www.postgresql.org/docs/18/btree.html) 一致。

替代方案是单列表达式 hash exclusion：`EXCLUDE USING hash ((issue_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)`。UUID 规范文本长度固定，组合表达式无歧义；排斥比较仍对原文做等号重检，哈希相同的不同 URL 可以共存。它不是“摘要唯一”，也不是 PostgreSQL 不支持的 hash unique index。官方依据：[hash 索引](https://www.postgresql.org/docs/18/hash-index.html)、[EXCLUDE 约束](https://www.postgresql.org/docs/18/sql-createtable.html)。

真实数据库结果：同一长 URL 成功保存且原文不变；同 issue 的重复 INSERT/UPDATE 报指定约束 `23P01`；不同 issue、不同 URL、软删后重建均成功；2049 字符由 CHECK 拒绝；真实 hash 碰撞的两个不同原文同时存在。完整代码、命令、原始输出与 SHA 在 `index-probe/`；固定 issue UUID `0199a2b4-0000-7000-8000-000000000001` 的可复验碰撞对为 `http://collision.example/271150` 与 `http://collision.example/83166`，完整表达式 hash 均为 `-1669361772`，补验证据单列 `index-probe-v2/`，未覆盖前轮。实验自建的 PostgreSQL/Ryuk 容器均已自动清理，没有操作既有容器。

实验不覆盖正式 API、权限/grants、业务事务或生产并发。T1/T3/T4/T5/T13 必须将长 URL、精确错误映射、碰撞对照和锁串行验证变为正式产品测试。

## 3. 准备中的局部裁定

| 裁定 | 理由 | 错了的代价与证明 |
|---|---|---|
| 原设计 12 粗项展开 16 Task | 将地址策略与 HTTP 执行分开评审，补完整授权/回滚与终态门禁 | 规模估计不是原型实测；实施前继续核大小，不增加空占位。 |
| 标题修改在 SQL 内保留当前 metadata，并返回实际行 | crawler 只锁 link，app 的旧快照不能覆盖它刚写的 metadata | 若错会丢预览；T4 存储与 T13 真实交错证明。 |
| 以 hash exclusion 替代长原文 B-tree | 保留 2048 字符、精确唯一、12 列和现有项目锁，修复物理存储不能兑现契约的问题 | 只改变 issue 存储约束及其精确 SQLSTATE 映射，不改共享字段/端口、事务协议或依赖。实际可行性见 §2.1，正式测试仍待实施。 |
| 不增加 URL generation | 继续已批准的 id+url+未删除写回守卫 | A→B→A 时旧 A 结果可能晚写，作为既定预览限制，不声称严格版本新鲜。 |

请求写资源顺序仍是预检→全部闸门→项目 N→锁下重读/授权→issue→link→activity→事件→同事务 job。metadata worker 只写带条件的 link，不持项目锁、不再索取其他业务资源。排斥约束可能等待冲突事务；同 issue 的生产 URL 写已被项目 N 串行，原文不相等的 hash 碰撞在等待前被排除。绕过既定锁协议的直接 SQL 并发写不在“无死锁”承诺内；静态推导不替代 T5/T13 的真实验证。

## 4. 预检发现与处理

首审累计 High 0、Medium 5、Low 0，全部是既定范围内的局部修订，未要求放宽守卫或改变模块协议。

| 编号 | 问题、修订及验证归属 | 当前结论 |
|---|---|---|
| C1 / Medium | 访客可见性必须依 Grant.OwnOnly；补 guest_view_all_features=true/false、管理员兼访客与普通访客的差别，T3/T5 按真实判定覆盖，并修正交接 03 的条件省略。 | v2 修订，主控核对闭合。 |
| S1 / Medium | title-only updated 无 url Change 是合法不投递，不能按“缺 url”返回错误；created 缺 url、重复 url、nil New 仍是错误，T11 补真实提交/审计/无 job。 | v2 独立复审闭合。 |
| S2 / Medium | 仅检查公开 metadata 无法证明日志脱敏；T9 捕获完整 slog 输出，用不同 query/userinfo/body 哨兵、阶段/分类非空及原始 URL/error 日志生产变异证明。 | v3 独立复审闭合。 |
| I1 / Medium | 2048 字符与原文 B-tree 容量冲突；采纳 §2.1/§3 的精确 hash exclusion，T3 只映射约束名+23P01，补长 URL、软删、碰撞和 API 边界。 | v4 独立定向复审闭合。 |
| I2 / Medium | 既有 TestIssueEventDirectory 固定 12 类、把 link 视 future、向生产订阅者发布空 Event；T3 改独立 15 类目录，T11 分离真实 payload/事务订阅测试。 | v4 独立定向复审闭合。 |

安全分区 v3、集成分区 v4 均已 Approved；未关闭 High/Medium/Low 均为 0，预检通过。所有文件清单/冻结稿哈希与定向结论均保留，不将作者自查代替独评。

## 5. 当前实现与 plan 的边界

尚无 P7 产品实现。新文件名、签名和测试名是约定，须逐 Task 用实际证据更新。准备分支只改正式文档、M4 设计中已证明的物理约束与进度、关键词 phase；实验与原始记录均 ignored。没有新增依赖、迁移、生产抓取或接口，没有为预检提前造完整产品副本。

## 6. 移交事项

P6 已授权 `--no-ff` 合并为 `5005be985e7ec78727edc6f8fc5c05999b45cc65`，双亲为 `ee26db29…` / `17908e03…`，tree 与已审 P6 分支逐字相同。P6 的最终分支 [CI 40303510060](https://github.com/open-nerve/NerveProject/actions/runs/40303510060) 全绿；main [CI 40305247546](https://github.com/open-nerve/NerveProject/actions/runs/40305247546) 首次浏览器安装超时取消，重跑后同一 SHA 三任务全绿。921 个文件、13,550,049 字节证据归档核 hash 后，P6 本地/远端分支和 worktree 已清理；精确双亲/tree、归档根与清单见 [P6 review §2.2](P6-issue-events-review.md#22-授权合并main-ci-与归档清理)。

P7 接收 P6 首个生产订阅者/请求投递、M2 链接审计和 worker/grants、M3 registrationRows。P8 link_count、P9 异步删除、P11/P14 页面及 M4 最终 closeout 仍开放，不提前关闭输入 handoffs。

## 7. 已知限制与下一步

本次只准备 spec/plan 和独立预检，T1 尚未开始。预检与 P6 main CI/归档清理均已完成，等待负责人安排启动。将来的阶段完成后执行 deep-review、局部修复复审，再按授权集成；预检通过本身不授权实现或合并。

最终文档事实复核另发现 1 项 Low：P6 plan 末段仍将已完成的授权合并/main CI/归档清理写成待办。已改为明确历史状态并引用准确合并提交及阶段 review §2.2；独立定向复核已通过，开放 High/Medium/Low 均为 0，报告在 ignored `final-doc-review.md`；不改写原先产品审查范围。
