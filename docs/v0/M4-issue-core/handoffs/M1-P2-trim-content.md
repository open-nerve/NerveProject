---
status: open
from: M1/P2
to: M4
created: 2026-09-23
---

# 工作项：前端不再使用的字段和接口，以及一个继承的描述保存问题

## 不再使用的字段和接口

- `estimate_point`：估算已删除（M1/P2 Task 4）。
- `POST …/issue-dates/`：只有拖动甘特图的条会调用，甘特图已删除（Task 5）。
- 搜索结果里的 `page` 类型：文档页已删除（Task 7）。
- `description_binary`：协作编辑的 Yjs 二进制，前端不再读取也不再写入（Task 7）。工作项描述只用 `description_html`。

## 只补 id 的迁移更新

编辑器给每个块节点一个 `data-id`（`UniqueID`）。打开一个缺 id 的旧描述时，编辑器会补上 id 并发出

```
PATCH …/issues/<id>/  {"description_html": "…", "skip_activity": "true"}
```

M4 的描述更新接口要照此处理：`skip_activity` 为真时不记动态。M1/P2 修复了一处回归（`79ef2b5`）：一旦出现只读编辑器，之后的编辑器都不再补 id。编辑器的交互测试（`web/packages/editor/src/editor-interaction.test.ts`）和 P2 评审附录 6.3 的浏览器探测都覆盖这一点。

## 继承自 Plane 的描述保存问题（不是 P2 引入的）

`web/apps/web/core/components/editor/rich-text/description-input/root.tsx`（自导入 `48e1a63` 以来未改）：

- **现象**：如果在 1.5 秒的保存延迟内先撤销、再重做，最终发出的保存带的是撤销后的内容，而编辑器里显示的是重做后的内容。服务端的这份描述要到下一次编辑才会被纠正。
- **原因**：撤销时，安排了一次带撤销后 HTML 的保存；重做时，HTML 等于上次保存的内容，于是提前返回，待发的那次保存没有被替换。

M4 重写描述保存时一并修复。复现步骤和输出见 [M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)附录 6.3 第 7 节。

## 关闭条件

M4 合并时：

- 工作项接口没有 `estimate_point`、`description_binary`，没有 `/issue-dates/`，搜索结果没有 `page` 类型；
- 描述更新接口在 `skip_activity` 为真时不记动态，有测试；
- 描述保存的"撤销再重做"问题已修复，有覆盖它的测试或写进 review 附录的探测。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

## 处理结果（M4/P1a）

- **接口的字段**（工作项的第一个操作一侧完成）：`createIssue`（`POST /api/v0/projects/{project_id}/issues`，`api/modules/issue.yaml`）的 `IssueCreate`、`IssueDetail` 没有 `estimate_point`、`description_binary`；没有 `/issue-dates/` 的接口，`issues` 表也没有 `estimate_point_id`、`description_binary` 两列（M4 设计 4.2）。以后的工作项操作照此。

仍未处理，状态保持 `open`：搜索结果没有 `page` 类型（M4 的 P8）；`skip_activity`（M4 的 P4，随描述）；描述保存的"撤销再重做"（M4 的 P14）。

来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

## 处理结果（M4/P4）

- 服务端描述已清洗并以 boolean `skip_activity` 表达字段存在性；仅 true 且纯文本、按顺序提及、href、checked 内容标识相同才抑制主审计/动态/版本，仍保存 HTML/stripped 并同步提及资格。旧字符串 `"true"` 不兼容，不把所有 skip 一律免审计。T1/T7/T9及I2普通保存、块id、href/checked独立变异有证据。
- 保留余项：P8 搜索结果无 page；P14 修描述保存撤销再重做、创建/编辑弹窗 `isMigrationUpdate`（本Phase未改页面）。M5 图片开启时将按序 src 纳入标识并扩S3，经M4真正收尾转交。

状态保持 `open`；本次仅完成P4接收部分，待M4真正收尾逐项核对关闭条件。证据：[P4 spec §4/§7](../specs/P4-issue-collaboration.md#4-验收标准)、[P4 review §2/§6](../reviews/P4-issue-collaboration-review.md#2-验收标准核对)。

## 处理结果（M4/P8，2026-10-27）

T12/T14已以真实HTTP/生成客户端验证issue-search返回完整Issue的{data:[]}且不含page/total/next_cursor；I18覆盖空词/数字/Unicode/中文/选择器/OwnOnly与日期，搜索只查标题和编号，不搜索description_stripped。保存/发布结构与HTML清洗沿唯一创建核心，草稿保存不作创建语义。此回执只承接搜索无page子项；P14撤销再重做/isMigrationUpdate、P15页面及原文其他输入不关闭，顶部status保持open。

准确产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6`，首次整审 `166b9efd..10f44790`；[P8 review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)列A条款/真实RUN/变异恢复/四次七门禁及准确产品checkpoint CI，[整体审核](../../../reviews/re-07-M4P8整体审核.md)列三镜头/186文件覆盖与历史D1。文档增量独评及最终准确HEAD CI由后续提交和ignored收据追踪，未合并、不清理；不以本期子项已处理改写历史关闭条件或M4整体状态。
