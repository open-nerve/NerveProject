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

## 关闭条件

M5 合并时：文件资源的接口没有 `PAGE_DESCRIPTION` 类型，表里没有 `page_id` 列（或在 review 里写明保留它的理由）。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

## 处理结果（M5/P2）

P2首张file_assets表没有page_id或页面资源用途；十种用途直接来自OpenAPI生成，F1逐一经真实PUT/complete/GET。旧前端资源类型的最终删除仍归P6/P7，本交接保持open。证据见[P2阶段评审](../reviews/P2-assets-review.md)，整审/集成结果仍单独核验。

## 处理结果（M5/P7b T12核对）

`EFileAssetType`与旧文件服务已整体删除，前端改用OpenAPI生成的AssetPurpose和内容宿主；新接口从未提供`PAGE_DESCRIPTION`。逐项结论由M5收尾写入review后再关闭，本页保持open。

## 处理结果（M5/P8）

两条关闭条件都已满足：资产接口的用途由OpenAPI生成的`AssetPurpose`给出，从未有`PAGE_DESCRIPTION`；`file_assets`没有`page_id`列（P2表结构，P8未改）。旧`EFileAssetType`与旧文件服务已在P7b整体删除。T12最终矩阵在最终产品候选上重跑F1–F18，见[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)。逐项结论如上，closeout改为closed。
