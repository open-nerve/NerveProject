# M5/P1：文件路由与工作区命名空间冲突（待裁定）

发现日期：2026-11-02。分支 `codex/m5-p1`，复现产品提交 `7cc4731b228d07872bd093e9e060168db32fac94`。T1–T8 和 T9 默认慢流已提交；完整七门禁只完成生成检查，阶段 deep-review 尚未开始。

## 1. 根因与影响

M3 允许工作区使用 `storage` 作为 slug，页面路径为 `/storage/projects`、`/storage/settings` 等。P1 设计新增 `/storage/` 的服务端 fallback，这些请求先被文件模块回答为 404，无法取得前端页面。这会影响合法的既有同名工作区；数据库和工作区 API 仍可用。

根因链：P1 选择新顶层 `/storage` → M3 已发布的保留名单没有该名字 → 新路由占用合法工作区页面。P1 预检漏掉了命名空间核对。修复需要调整已预检的传输路径，或改变既有工作区命名/迁移规则，按负责人纪律先提交再上报。

相关代码：`asset/adapter/http/transfer.go` 注册三种方法及两个 fallback；`asset/adapter/local/token.go` 固定签发 `/storage/local/`；`workspace/domain/reserved_slugs.txt` 的 server 段只有 api/assets/healthz/readyz；`web/apps/web/app/routes/core.ts` 定义动态工作区页面。既有 `bootstrap/reserved_test.go` 守卫能捕获该回归，不应放宽。

## 2. 真实复现

探针复用 bootstrap、apitest、pgtest；账户与工作区均经真实 API 创建。本次只操作自己的测试数据库，没有查询或修改实际部署。

| 请求 | 当前 P1 | 仅移除 asset 路由注册的对照 |
|---|---|---|
| POST /api/v0/workspaces，slug=storage | 201 | 201 |
| GET /api/v0/workspaces/storage | 200 | 200 |
| GET /acme/projects | 200 HTML | 200 HTML |
| GET /storage/projects | **404 Problem** | 200 HTML |
| GET /storage/settings | **404 Problem** | 200 HTML |

对照仅用于隔离根因，不是修复，也不是另一份 main checkout。临时探针已移回原件区，注册代码按原字节恢复。现有守卫 `TestTheReservedServerSlugsAreTheServersTopLevelPaths` 失败：实际根路径 `[api assets healthz readyz storage]`，保留名单 `[api assets healthz readyz]`。

原件位于本 worktree 的 `.superpowers/sdd/P1-local-storage/`：

| 文件 | bytes | SHA-256 |
|---|---:|---|
| task-9-namespace-probe.source.go | 1542 | 7bdd19ddd75dfe9e73470092565ac979762dcfdb6963ebd1f534ef1681f35331 |
| task-9-namespace-repro.log | 2623 | 814fb6bc5c218242f8275498fa6dd0616e69479c5e92780f349f20efca2db7ad |
| task-9-namespace-control-without-registration.log | 2304 | a9239df6406bcd60ed89abbecb0881d864227ac0271dfba69f3f41f7bd72e907 |
| task-9-namespace-reserved-guard.log | 339 | efd45d061856105bfd59811dd86111400cf05adf145b1bfef37b189ca1f08ac5 |

## 3. 方案 A（推荐）：使用已保留的服务端命名空间

把 local 传输移到 **`/assets/storage/local/{key}`**；fallback 和禁止重定向策略仅覆盖 `/assets/storage` 子树。`assets` 已被 M3 保留，现有工作区地址和数据保持原样。

批准后的实施顺序：

1. 先同步 M5 design、P1 spec/plan、raw 目录和运行说明。
2. 更新签发路径、handler 的精确路径校验/注册和相应测试。P1 尚未推送/合并且没有业务上传入口，直接修改约定，无需保留有冲突的旧入口。
3. 为开发 Vite 增加准确的 `/assets/storage` 转发；它目前只转发 `/api`。验证普通静态资产的路径、长期缓存和缺失 chunk 的 404 语义。
4. 将探针转为永久回归，覆盖 storage 工作区页面、raw 子树、普通静态文件；原 M3 顶层路径守卫原样通过。
5. 重新完成 T9 七项门禁，再推进分支 CI、deep-review 与正常集成。

代价：调整尚未发布的传输 URL，并补静态资产/开发代理联合回归；文件物理布局、MAC 用途、工作区数据和 JSON API 权限不变。

## 4. 方案 B：保留顶层 /storage，规划工作区迁移

把 storage 加入保留名单只能阻止新创建，既有同名工作区还需盘点、迁移 slug 和处理旧链接。该方案扩大到 M3 的命名/迁移合同，需要单独设计和数据操作授权，成本及风险更高。

## 5. 停点

等待负责人裁定后再修改相关产品行为。默认 5 MiB 慢流 31.9s 及 race、封存/读取一致性已通过；`make gen && make gen-check` 通过。其余六项完整门禁、P1 整分支 deep-review、产品推送、合并和清理均未完成，M5 仍在 P1。
