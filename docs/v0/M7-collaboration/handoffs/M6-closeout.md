---
status: open
from: M6/closeout
to: M7
created: 2026-11-10
---

# M6 交给 M7：收集箱新建弹窗在编辑器建成之前拒绝关闭

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页问题已经解决。M6 进行中发现的条目随发现写入；M6 收尾时补齐其余交接。

<a id="M6-M7-01"></a>
## M6-M7-01 收集箱新建弹窗在编辑器建成之前拒绝关闭

- M6/P4 查实 M4 新建工作项弹窗的 C12 间歇失败：弹窗刚打开时描述编辑器尚未建成，编辑器的引用却已挂上一套操作空编辑器的 API，`isEditorReadyToDiscard()` 答 false，弹窗把 Escape 当成“编辑器忙”而拒绝关闭。修复分支 `codex/m6-p4-ci` 改为编辑器建成之前引用为 null（`web/packages/editor/src/hooks/use-editor.ts`），“引用存在”即“编辑器存在”；经过见 [P4 阶段记录](../../M6-cycles-modules/reviews/P4-progress-transfer-review.md) §5 与 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)。
- 收集箱的新建弹窗（`web/apps/web/core/components/inbox/modals/create-modal/create-root.tsx`，沿用 Plane）有同类问题，这次修正前后行为相同：Escape（`handleEscKeyDown`）与 Cancel 按钮只在 `descriptionEditorRef.current?.isEditorReadyToDiscard()` 为真时关闭，否则提示 “Editor is still processing changes” 并留在原处。编辑器还在加载或尚未建成时引用为 null，空表单也关不掉。
- 处理方向：与 `issue-modal/draft-issue-layout.tsx` 的 `requestClose` 同一条规则——引用为 null（编辑器未建成）时可以关闭，只有已建成且有未完成的图片时才拒绝；提交仍要求编辑器就绪。M6 不改这个页面，因为收集箱的接通属于 M7。

**何时验收：** 收集箱页面首次接通的阶段。弹窗一打开就按 Escape 或 Cancel 能关闭空表单（可仿 M4/P14 的 C10 编辑器加载故事扣住编辑器代码块），有未完成图片时仍拒绝。
