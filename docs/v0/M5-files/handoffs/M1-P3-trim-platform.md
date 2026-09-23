---
status: open
from: M1/P3
to: M5
created: 2026-09-24
---

# 文件：前端剩下的资源类型、封面图和上传地址

M1/P3 删掉了团队等企业版功能（Task 3）、Unsplash（Task 4），以及 Plane 不路由的工作区级批量上传（Task 7）。M5 设计文件接口时，与下面这些保持一致。

- **资源类型。** `EFileAssetType`（`web/packages/types/src/enums.ts`）只剩 `COMMENT_DESCRIPTION`、`ISSUE_ATTACHMENT`、`ISSUE_DESCRIPTION`、`DRAFT_ISSUE_DESCRIPTION`、`PROJECT_COVER`、`USER_AVATAR`、`USER_COVER`、`WORKSPACE_LOGO`。`TEAM_SPACE_DESCRIPTION`、`TEAM_SPACE_COMMENT_DESCRIPTION`、`INITIATIVE_DESCRIPTION`、`PROJECT_DESCRIPTION` 已删除，新接口不需要接受它们。
- **评论的附件总是项目级上传。** 评论框总在某个项目里，`CommentCreate` 的 `projectId` 已改为必填。工作区级的上传分支，以及它调用的 `POST /api/assets/v2/workspaces/<slug>/<entityId>/bulk/`（Plane 不路由这个地址）已删除。
- **封面图没有 Unsplash。** 封面类型只剩 `local_static`（内置图片）和 `uploaded_asset`（上传的文件），图片选择器只有"图片""上传"两个标签页，打开时不再请求 `/api/unsplash/`。实例配置的 `has_unsplash_configured` 已删除。
- **编辑器里拖入或粘贴非图片文件。** 社区版的编辑器没有附件节点（企业版才有），P3 删掉了为它准备的 `"attachment"` 命令类型和上传参数。但编辑器仍按 `ACCEPTED_ATTACHMENT_MIME_TYPES`（`web/packages/editor/src/constants/config.ts`）把 PDF、文档等文件的拖入和粘贴算作"已处理"：什么也不插入，浏览器的默认行为也被阻止。这与基点相同。M5 设计文件接口时决定：编辑器是否支持附件；不支持的话，删掉这份 MIME 列表，让这些文件不被接收。工作项的附件（`ISSUE_ATTACHMENT`，在附件区上传）不受影响。
- **`file_size_limit` 的名称和策略**（M1 设计 3.7）：实例配置里的这个字段仍由 `use-file-size` 读取，没有值时用前端常量 `MAX_FILE_SIZE`。名称和上限策略由 M2 或 M5 定（M2 的交接里也有这一条）。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
