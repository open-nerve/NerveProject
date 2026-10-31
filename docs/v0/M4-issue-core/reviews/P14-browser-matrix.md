# P14 三浏览器实际执行矩阵

日期：2026-10-31。固定产品提交 `e0ee0ff7e75ae4c2234f9c214fdbc91ed7a12de4`。命令均在实施 worktree 根，先完成 `make e2e` 的真实 build，再按 Chromium → Firefox → WebKit 串行执行；无并行数据库世界。

`pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-detail.config.ts --project=<browser> --reporter=list,html,json --output=<绝对证据路径>`；HTML/JSON 输出同样是绝对路径。每个浏览器 44 项（40 个业务页面故事、4 个已有 C2 兼容夹具），均一次通过、零 skipped/unexpected/flaky。不得把默认全量 E2E 的两个 Firefox A4 当作本矩阵。

| 引擎 | 实际版本 | passed | skipped / unexpected / flaky | Playwright 报告耗时 |
|---|---|---:|---|---:|
| chromium | 153.0.8010.12 | 44 | 0 / 0 / 0 | 28.676 秒 |
| firefox | 155.0 | 44 | 0 / 0 / 0 | 33.880 秒 |
| webkit | 26.6 | 44 | 0 / 0 / 0 | 32.534 秒 |

Playwright 锁定 1.63.0；版本探针见 `t16-engine-probe.log`，每轮输入、原始输出、完整 JSON、HTML、产物 hash 见 `.superpowers/sdd/P14-web-issue-detail/t16-final-matrix-<browser>*`。三次报告列出的 test title 完全一致，以下每行均为三个引擎分别通过。

| 序号 | A 条款 | 实际文件/行与 test title |
|---:|---|---|
| 1 | A14 | `e2e/stories/issue/detail-interactions.spec.ts:12` — C4 (page): identifier copy is keyboard reachable and reports a refused clipboard write without success or navigation |
| 2 | A14 | `e2e/stories/issue/detail-interactions.spec.ts:12` — C4 (page): detail copy is keyboard reachable and reports a refused clipboard write without success or navigation |
| 3 | A8/A14 | `e2e/stories/issue/detail-interactions.spec.ts:12` — C4 (page): link copy is keyboard reachable and reports a refused clipboard write without success or navigation |
| 4 | A7/A14 | `e2e/stories/issue/detail-interactions.spec.ts:12` — C4 (page): relation copy is keyboard reachable and reports a refused clipboard write without success or navigation |
| 5 | A6/A14 | `e2e/stories/issue/detail-interactions.spec.ts:12` — C4 (page): sub-item copy is keyboard reachable and reports a refused clipboard write without success or navigation |
| 6 | A14 | `e2e/stories/issue/detail-interactions.spec.ts:12` — C4 (page): peek copy is keyboard reachable and reports a refused clipboard write without success or navigation |
| 7 | A4/A5 | `e2e/stories/issue/detail-interactions.spec.ts:84` — C4: creation owns dismissal, nested draft confirmation owns its Escape and Tab loop, and date/label portals retain the outer form and its focus |
| 8 | A5/A6 | `e2e/stories/issue/detail-interactions.spec.ts:175` — C4: native parent search owns only its inner Escape and restores a clean creation form |
| 9 | A5/A6 | `e2e/stories/issue/detail-interactions.spec.ts:175` — C4: native parent search owns only its inner Escape and restores a dirty creation form |
| 10 | A10/A15 | `e2e/stories/issue/editor-loading.spec.ts:54` — C10 (page): an open peek display panel follows its button when the editor finishes loading |
| 11 | A5/A10 | `e2e/stories/issue/editor-loading.spec.ts:99` — C10 (page): Escape can leave a clean creation while the real editor chunk is still pending |
| 12 | A5/A10 | `e2e/stories/issue/editor-loading.spec.ts:99` — C10 (page): Discard can leave a clean creation while the real editor chunk is still pending |
| 13 | A5/A10 | `e2e/stories/issue/editor-loading.spec.ts:99` — C10 (page): Escape can leave a dirty creation while the real editor chunk is still pending |
| 14 | A5/A10 | `e2e/stories/issue/editor-loading.spec.ts:99` — C10 (page): Discard can leave a dirty creation while the real editor chunk is still pending |
| 15 | A1/A10 | `e2e/stories/issue/editor-loading.spec.ts:151` — C10 (page): anonymous number entry loads no editor implementation; signing in loads it and saves real text without warnings |
| 16 | A1/A10 | `e2e/stories/issue/editor-loading.spec.ts:151` — C10 (page): anonymous id entry loads no editor implementation; signing in loads it and saves real text without warnings |
| 17 | A10 | `e2e/stories/issue/editor-loading.spec.ts:206` — C10 (page): personal settings keeps the command palette without loading a work-item editor |
| 18 | A11 | `e2e/stories/issue/editor-loading.spec.ts:231` — C10 (page): a native callout saves and reopens without external images or workers |
| 19 | A12 | `e2e/stories/issue/editor-loading.spec.ts:268` — C12 (page): detail has no image, attachment or subscription action; image paste and drop make no upload and text still saves |
| 20 | A12 | `e2e/stories/issue/editor-loading.spec.ts:268` — C12 (page): creation has no image, attachment or subscription action; image paste and drop make no upload and text still saves |
| 21 | A15（C2 回归） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:34` — C2: actual add, multi-value and operator selectors support keyboard search, clearing and focus; date drafts expose their error |
| 22 | A15（C2 回归） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:86` — C2: display panels anchor and return focus on desktop and mobile; mobile layout has one named clickable trigger |
| 23 | A15（C2 回归） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:173` — C2: a parent's real custom display and filter buttons anchor their panels and return focus inside the peek |
| 24 | A15（兼容夹具） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:209` — C2: browser compatibility fixture single/en opens during loading, searches, clears, selects and returns keyboard focus |
| 25 | A15（兼容夹具） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:209` — C2: browser compatibility fixture single/zh-CN opens during loading, searches, clears, selects and returns keyboard focus |
| 26 | A15（兼容夹具） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:209` — C2: browser compatibility fixture multi/en opens during loading, searches, clears, selects and returns keyboard focus |
| 27 | A15（兼容夹具） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:209` — C2: browser compatibility fixture multi/zh-CN opens during loading, searches, clears, selects and returns keyboard focus |
| 28 | A15（C2 回归） | `e2e/stories/issue/filter-archive-profile-interactions.spec.ts:258` — C2: Chinese project filters translate condition names, built-in choices and operators |
| 29 | A4 | `e2e/stories/issue/i1-create-issue.spec.ts:23` — I1 (page): c opens a named form; field refusals retain it, native controls create one numbered work item with a mention, and a held confirmation cannot be submitted twice |
| 30 | A8 | `e2e/stories/issue/i10-links.spec.ts:37` — I10 (page): URL refusals are described before focus; a confirmed link first shows its address, then real worker metadata without external image requests; title and URL edits, a duplicate refusal and deletion retain exact activity |
| 31 | A9 | `e2e/stories/issue/i17-delete-issue.spec.ts:32` — I17 (page): the confirmation names the whole subtree including archived descendants; one held deletion stays open until confirmed, deletes every catalog row by the current admin, and preserves the old branch and unrelated item |
| 32 | A3 | `e2e/stories/issue/i2-description.spec.ts:34` — I2 (page): real description editing and undo-redo keep the visible value, a refusal retains it for explicit retry, and pure block-id migration preserves audit stamps |
| 33 | A2 | `e2e/stories/issue/i2-update-issue.spec.ts:34` — I2 (page): the detail announces title refusals before focus, saves each property through its real control and clears selected collections and dates |
| 34 | A13 | `e2e/stories/issue/i20-guest-boundary.spec.ts:73` — I20 (page): project guest with workspace admin=false follows own-only and the returned grant, with hidden and absent details indistinguishable |
| 35 | A13 | `e2e/stories/issue/i20-guest-boundary.spec.ts:73` — I20 (page): project guest with workspace admin=true follows own-only and the returned grant, with hidden and absent details indistinguishable |
| 36 | A1/A13 | `e2e/stories/issue/i22-session-switch.spec.ts:19` — I22 (page): a real creation is committed before another tab signs out and signs a different account in; the old reply cannot toast, navigate, project its hidden row or close the new account's form |
| 37 | A1/A13 | `e2e/stories/issue/i22-session-switch.spec.ts:112` — I22 (page): Back across workspace, project and issue ends creation, deletion, link and peek owners; Forward does not revive them |
| 38 | A12 | `e2e/stories/issue/i8-sub-issues.spec.ts:42` — I8 (page): list child count opens the exact native parent detail by click and keyboard without a fragment |
| 39 | A12 | `e2e/stories/issue/i8-sub-issues.spec.ts:42` — I8 (page): spreadsheet child count opens the exact native parent detail by click and keyboard without a fragment |
| 40 | A6 | `e2e/stories/issue/i8-sub-issues.spec.ts:91` — I8 (page): native searches add two existing children and create one; progress is confirmed, descendants cannot be parents, and moving and removing a parent update counts and activity |
| 41 | A7 | `e2e/stories/issue/i9-relations.spec.ts:26` — I9 (page): native candidates create Blocking and cross-project Relates to; the reverse end shows Blocked by, and a member authorized only at the other end deletes the cross-project relation with both activities |
| 42 | A13 | `e2e/stories/smoke/s2-web-app.spec.ts:458` — S2 (detail): public-project member receives exactly the permitted detail or its 404/403 refusal |
| 43 | A13 | `e2e/stories/smoke/s2-web-app.spec.ts:458` — S2 (detail): public-project guest receives exactly the permitted detail or its 404/403 refusal |
| 44 | A13 | `e2e/stories/smoke/s2-web-app.spec.ts:458` — S2 (detail): public-project project non-member receives exactly the permitted detail or its 404/403 refusal |

A3 的 issue/draft 迁移 dirty、A4 的异步字段反馈、A6/A7 的部分确认、A8 的有界读取/晚答、A9 的拒绝与会话 ABA、A14 的复制成功与等待等补充消费者单测见阶段 review §8；本表不冒称每一条边界均由页面覆盖。C3 由 I2 属性与 I8 父项、C5 由 I8/I9/I10/I17、C11 由 I20/I22/S2 的实际页面承接。

首轮 44/37/34 及后续诊断的失败原件保留；最终通过不能抹去历史失败。非法控制符使用 DOM input 注入、剪贴板使用合成事件；它们验证实际表单/editor/HTTP/数据库，不能当作原生系统剪贴板权限或键盘输入 NUL 的验证。Mac WebKit 走 Option-Tab；HTTP 拒绝按各引擎的精确系统错误匹配，warning 无豁免。
