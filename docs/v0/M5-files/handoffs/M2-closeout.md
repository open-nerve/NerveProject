---
status: open
from: M2/closeout
to: M5
created: 2026-09-28
---

# M2 交给 M5：头像和封面、上传与按路由的中间件、删除 `@nerve/services`

M2（账户认证）的接口里，头像、封面是可为 `null` 的地址，M2 没有文件存储（[M2 设计](../../M2-auth/M2-design.md) 3.2）。下面这些由 M5 接上。第 1、2 节来自 M2 设计 13.2 给 M5 的两行，第 3 节来自 M2/P5 spec 第 5 节"不在 P5 范围内"，第 4 节是 13.2"M3 及以后有 stores 的 M"一行中 M5 的部分，第 5 节是 M0-P6 交接剩下的一项。

## 1. 头像和封面

- **做什么**：加入 `users.avatar_asset_id`、`cover_image_asset_id`；`User.avatar_url`、`cover_image_url` 开始返回签名地址；按新的上传协议加回 general 页和新手引导资料步骤的上传控件（M2 设计 3.2）。
- **迁移的范例**（M2 设计 3.14，总体设计 5.6）：先在 M5 的模块里建 `file_assets`（`<v>_<模块>_file_assets.sql`），再写 `<v+1>_identity_users_avatar_asset.sql` 给 `users` 加列和外键；后者归 `identity` 的 sqlc 条目（`server/sqlc.yaml`）。`TestSQLCSchemaScope` 核对 `ALTER TABLE` 的所有者，它的用例里就有这个文件名（`server/internal/archtest/sqlc_cases_test.go:30-32`）。
- **M2 做了什么**：建 `users` 时不建这两列（差异清单二·按表），接口的两个地址一直是 `null`。M2/P4 删掉了头像、封面的上传控件和 Plane 的上传路径（`cb2487c9`），`EFileAssetType`（`web/packages/types/src/enums.ts:22-29`）的 `USER_AVATAR`、`USER_COVER` 随之删除，现在是 6 种。关键词规则 `plane-user-assets`（`tools/keywords.json`）挡住 Plane 的 `/api/assets/v2/user-assets/`、`/api/users/file-assets/`、`UserImageUploadModal` 和 `USER_AVATAR`、`USER_COVER`（M2 设计 7.9）。
- **加回时**：资源类型按 M5 的协议定。沿用 `USER_AVATAR`、`USER_COVER` 这两个名字时，从规则的 `content` 中去掉 `\bUSER_(?:AVATAR|COVER)\b` 一支，`samples` 随之改；Plane 的两个地址和弹窗名仍然挡住。
- **关闭条件**：两列和外键照上面的两个迁移加入，差异清单二·按表更新；两个地址返回签名地址；两处上传控件用新协议；`node tools/keywords.mjs` 通过，`plane-user-assets` 仍挡住 Plane 的地址。

## 2. 上传与按路由的中间件

- **做什么**：`/api/v0` 的每个操作都经过模块级的 `Middlewares`：请求体上限 `server.max_body_bytes`（默认 1 MiB）和请求期限 `server.request_timeout`（默认 15 秒；默认值在 `server/configs/config.yaml:15,17`，交给平台的字段在 `server/internal/platform/httpserver/api.go:42-43`），它们会让 `/api/v0` 下的上传失败。M5 在平台加按操作的放宽设置，或者把上传放在 `/api/v0` 之外（例如直传对象存储），并按 M2 设计 3.6 的整程序测试处理：写进接口描述，或在设计中说明（M2 设计控制者复核 m7）。
- **`file_size_limit` 的执行**：名称 M2 已定（`api/modules/instance.yaml:106`，前端 `web/apps/web/core/hooks/use-file-size.ts` 读它），值来自配置；上传时由 M5 执行。
- **CSP**：`img-src`、`connect-src`（`server/internal/platform/webui/csp.go:40,42`）加上存储的来源。只有对象存储这种本来就在外部的来源才加（M2 设计 8.3）。
- **与 [M0 加固的交接](M0-hardening-http-timeouts.md)的关系**：那一项管连接上的读写时限（`http.ResponseController`），这一项管按路由的请求体上限和请求期限，两者都要放宽。
- **关闭条件**：大于 1 MiB、耗时超过 15 秒的上传都成功（测试）；超过 `file_size_limit` 的被拒绝；整程序测试（`server/internal/bootstrap/contract_test.go`）仍通过；CSP 只为存储放开来源，有测试。

## 3. 删除 `@nerve/services`

- **做什么**：M5 重写文件的接口调用之后，删除整个 `web/packages/services`（总体设计 7.1，[前端改动清单](../../frontend-changes.md) 3.1 的最后一行）。
- **现在还剩**：上传文件的元数据工具（`web/packages/services/src/file/helper.ts` 的 `getFileMetaDataForUpload`、`generateFileUploadPayload`，调用方是 `web/apps/web/core/services/file.service.ts` 和 `web/apps/web/core/services/issue/issue_attachment.service.ts`），以及地址规范化（`web/packages/services/src/helpers/url.ts` 的 `normalizeAPIRequestURL`，带单元测试）。
- **注意**：地址规范化的调用方是 web 的 axios 基类 `web/apps/web/core/services/api.service.ts`，还没对接新接口的领域经它调 Plane 的旧地址（M5 之后是 M6–M8 的领域）。删包时它还有调用方，就把规范化连同测试挪进 web（例如 `web/apps/web/core/services/` 下），或者证明那时已没有旧地址要补结尾的 `/`。
- **守卫**：规则 `services-files`（`tools/keywords.json`，只允许包里现在的几个文件）随包删除；包不存在之后，这条规则删掉，或改为挡住包名重新出现。
- **关闭条件**：`web/packages/services` 不存在，任何 `package.json` 都不依赖 `@nerve/services`；`make knip`、`make lint-web`、`make test-web` 通过；前端改动清单 3.1 的那一行改为已完成。

## 4. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M5 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M5 的范围（`d97c513` 上数出）：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `FileService` 8 处。它们现在经 web 的 axios 基类调 Plane 的旧地址。
- **关闭条件**：这 8 处都已消失；M5 新加的 SWR 键带 `loginId`。

## 5. e2e 的 `storage.ts`（M0-P6 交接剩下的一项）

- **做什么**：检查文件是否真的写入本地存储，按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/storage.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。
- **关闭条件**：`storage.ts` 照这个写法加入，上传的故事用它核对存储里的文件。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M5 的各行；[M2/P5 spec](../../M2-auth/specs/P5-web-account.md) 第 5 节；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
