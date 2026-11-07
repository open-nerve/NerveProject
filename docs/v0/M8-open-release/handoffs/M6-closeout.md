---
status: open
from: M6/closeout
to: M8
created: 2026-11-07
---

# M6 交给 M8：CI 的 e2e 作业期限与间歇失败的故事

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页问题已经解决。M6 进行中发现的条目随发现写入；M6 收尾时补齐其余交接。延续 [M5 → M8 交接 M5-M8-06](M5-closeout.md#M5-M8-06) 中仍开放的观察。

<a id="M6-M8-01"></a>
## M6-M8-01 e2e 作业期限由 25 改为 35 分钟，C7 与 C12 再次间歇失败

- M6 设计合入的 main CI 42281169459（main `d1ec8c69`，只改文档；同一内容的分支 CI 全绿）连续两次未通过：
  - attempt 1：e2e 417 条中 1 条失败，M4 的 C7（`e2e/stories/issue/collaboration-interactions.spec.ts:422`，键盘选主题）。选 Dark 收到 503 后，第一次 `reopenDuringExit("Control+k")` 期待弹窗面板停在 30 秒的慢退场（`data-leave`），5 秒内找不到面板元素。
  - attempt 2（重跑失败的作业）：安装 Playwright 浏览器一步用了 7 分 19 秒，e2e 作业在 25 分钟期限处被取消，当时故事跑到约第 314 条；取消前 M4 的 C12（`e2e/stories/issue/review-interactions.spec.ts:83` 的 kanban 布局，见 M5-M8-06）又失败一次，错误详情因取消未打印。
- 负责人 2026-11-07 批准把 e2e 作业的 `timeout-minutes` 由 25 改为 35（`.github/workflows/ci.yml`），server 作业仍为 25。依据：绿色 e2e 作业已用 23–24 分钟、故事本身 21–22 分钟（M5-M8-06），浏览器安装偶发停滞，M6 还会加入迭代与模块的故事。
- C7 与 C12 的原因仍未查实，均为界面时序类的间歇失败；配置中没有重试，失败即算数。

**何时验收：** M8 的 CI 工作复核 35 分钟的余量（或缓存浏览器的系统依赖、拆分作业，任一调整须负责人同意），并查实或修好 C7 与 C12；在此之前，这两条再失败时按“重跑一次，再不过就停下报告”处理。
