---
status: open
from: M1/P2
to: M5
created: 2026-09-23
---

# 文件资源：文档页的资源类型不再使用

M1/P2 删掉了文档页（Task 7）。M5 设计文件资源的接口和表时，按下面两点处理：

- 前端的资源类型不再有 `PAGE_DESCRIPTION`（`web/apps/web/core/services/file.service.ts`、`EFileAssetType`），新接口不要提供这个类型；
- Plane 的 `file_assets.page_id` 不再使用。

`EFileAssetType`（`web/packages/types/src/enums.ts`）里保留的类型仍按原样使用：工作项描述、草稿描述、评论描述、工作项附件、项目封面、用户头像与封面、工作区 logo 等。其中团队空间、Initiative 这类企业版类型由 M1/P3 删除。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
