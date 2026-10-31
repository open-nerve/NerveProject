# P15 三浏览器页面矩阵

状态：T16 三引擎实际执行通过；逐轮输入与影响复验见文末。

配置为 `e2e/playwright.m4-collaboration.config.ts`，八个文件，每引擎 43 个具名页面用例；Chromium、Firefox、WebKit 各显式运行，默认 Firefox 的 A4 限制不适用。原 API 故事及原 I20 由完整门禁覆盖。

逐例的精确英文题名保留以便与原始报告一一对应；以下各行都必须实际执行。C7 的旧会话成功由真实第二标签页登录触发；C9 的权限 ABA 在实际重新加载后观察，不能冒充未重载的权限推送；同 DOM 的失效由既有真实 store 单测负责。I18 的 A→B→Back→A 持有真实旧 200，重新查询后不得复活旧项。

C8 使用真实 LAN HTTP、无异步剪贴板、无临时用户激活的条件，由浏览器拒绝实际复制；键盘到达和无手势激活分别记录，不声称键盘复制必失败。watchPage、CSP、旧 API 和控制台门槛均保持。

| 编号 | 文件与实际题名 | Chromium | Firefox | WebKit |
|---|---|---|---|---|
| 01 | `issue/collaboration-interactions.spec.ts`<br>C6: activity filters keep one choice, support Enter and Space, persist sorting and return focus on Escape | 通过 | 通过 | 通过 |
| 02 | `issue/collaboration-interactions.spec.ts`<br>C6: reaction picker has one named button, keyboard focus returns on Escape, and combined emoji data stays on this origin | 通过 | 通过 | 通过 |
| 03 | `issue/collaboration-interactions.spec.ts`<br>C9: creation saves an empty-title real editor draft, retains field refusals, and its nested prompt owns Escape and confirmed close | 通过 | 通过 | 通过 |
| 04 | `issue/collaboration-interactions.spec.ts`<br>C9: Back retires a pending draft save and its prompt before the confirmed old reply reaches another workspace form | 通过 | 通过 | 通过 |
| 05 | `issue/collaboration-interactions.spec.ts`<br>C9: a draft editor stays retired across member to guest to member permissions after real reloads | 通过 | 通过 | 通过 |
| 06 | `issue/collaboration-interactions.spec.ts`<br>C9: real editor block-id migration in a copied draft is not an unsaved user edit | 通过 | 通过 | 通过 |
| 07 | `issue/collaboration-interactions.spec.ts`<br>C7: empty-query commands use the current issue, Chinese state selection respects IME, and deletion keeps its confirmation and focus | 通过 | 通过 | 通过 |
| 08 | `issue/collaboration-interactions.spec.ts`<br>C7: keyboard theme selection waits for confirmation, reports native refusal, then reloads only after success | 通过 | 通过 | 通过 |
| 09 | `issue/collaboration-interactions.spec.ts`<br>C7: a real theme success from the previous login cannot recolor or reload the new session | 通过 | 通过 | 通过 |
| 10 | `issue/collaboration-interactions.spec.ts`<br>C8: identifier copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 11 | `issue/collaboration-interactions.spec.ts`<br>C8: detail copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 12 | `issue/collaboration-interactions.spec.ts`<br>C8: link copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 13 | `issue/collaboration-interactions.spec.ts`<br>C8: relation copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 14 | `issue/collaboration-interactions.spec.ts`<br>C8: sub-item copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 15 | `issue/collaboration-interactions.spec.ts`<br>C8: peek copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 16 | `issue/collaboration-interactions.spec.ts`<br>C8: comment copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 17 | `issue/collaboration-interactions.spec.ts`<br>C8: layout copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 18 | `issue/collaboration-interactions.spec.ts`<br>C8: creation copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 19 | `issue/collaboration-interactions.spec.ts`<br>C8: command-id copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 20 | `issue/collaboration-interactions.spec.ts`<br>C8: command-title copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 21 | `issue/collaboration-interactions.spec.ts`<br>C8: command-url copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation | 通过 | 通过 | 通过 |
| 22 | `issue/i11-comments-reactions.spec.ts`<br>I11 (page): real comment input ignores IME and empty submission, retains a field refusal, and confirms one create and author edit | 通过 | 通过 | 通过 |
| 23 | `issue/i11-comments-reactions.spec.ts`<br>I11 (page): a named comment menu lets a project administrator delete another author but never edit | 通过 | 通过 | 通过 |
| 24 | `issue/i11-comments-reactions.spec.ts`<br>I11 (page): both reactions remove only the current actor after confirmation and remain visible without writes when archived | 通过 | 通过 | 通过 |
| 25 | `issue/i12-activity.spec.ts`<br>I12 (page): three real pages merge into one timeline, a confirmed description save reorders its existing activity, and reopening replaces remote comment edits and deletions | 通过 | 通过 | 通过 |
| 26 | `issue/i12-activity.spec.ts`<br>I12 (page): hidden historical relation references stay absent while the neighboring visible activity renders | 通过 | 通过 | 通过 |
| 27 | `issue/i13-description-history.spec.ts`<br>I13 (page): history loads a second page, copies actual Markdown after settlement, and restores only after a confirmed ordinary PATCH | 通过 | 通过 | 通过 |
| 28 | `issue/i13-description-history.spec.ts`<br>I13 (page): archived history keeps list and detail refusal retries separate and has no restore action | 通过 | 通过 | 通过 |
| 29 | `issue/i13-description-history.spec.ts`<br>I13 (page): a project guest reads his own visible item without any history request or control | 通过 | 通过 | 通过 |
| 30 | `issue/i14-drafts.spec.ts`<br>I14 (page): personal unassigned drafts paginate with server totals, preserve unresolved fields through real editing, and delete once from a keyboard or pointer menu | 通过 | 通过 | 通过 |
| 31 | `issue/i14-drafts.spec.ts`<br>I14 (page): publish saves latest form first, retains it on refusal, and confirmed publication retries only its failed list read | 通过 | 通过 | 通过 |
| 32 | `issue/i14-drafts.spec.ts`<br>I14 (page): a workspace guest opens the draft route without personal reads or creation controls | 通过 | 通过 | 通过 |
| 33 | `issue/i14-drafts.spec.ts`<br>I14 (page): making a copy preserves unresolved draft fields through the actual editor and leaves the original unchanged | 通过 | 通过 | 通过 |
| 34 | `issue/i16-auto-archive.spec.ts`<br>I16 (page): fresh clock process archives terminal issues and shows the system actor | 通过 | 通过 | 通过 |
| 35 | `issue/i18-search.spec.ts`<br>I18 (page): native command search keeps relevance order, exact Unicode and literal queries, workspace scope and local projects | 通过 | 通过 | 通过 |
| 36 | `issue/i18-search.spec.ts`<br>I18 (page): search errors offer GET retry and a delayed previous query cannot replace the current results | 通过 | 通过 | 通过 |
| 37 | `issue/i18-search.spec.ts`<br>I18 (page): OwnOnly search uses the returned visibility for a workspace role 5 and project guest | 通过 | 通过 | 通过 |
| 38 | `issue/i18-search.spec.ts`<br>I18 (page): OwnOnly search uses the returned visibility for a workspace role 20 and project guest | 通过 | 通过 | 通过 |
| 39 | `issue/i18-search.spec.ts`<br>I18 (page): Back and reopening the same query retire the previous project search answer | 通过 | 通过 | 通过 |
| 40 | `issue/i20-guest-boundary.spec.ts`<br>I20 (page): project guest with workspace admin=false comments on his visible issue following own-only and the returned grant, with hidden and absent details indistinguishable | 通过 | 通过 | 通过 |
| 41 | `issue/i20-guest-boundary.spec.ts`<br>I20 (page): project guest with workspace admin=true comments on his visible issue following own-only and the returned grant, with hidden and absent details indistinguishable | 通过 | 通过 | 通过 |
| 42 | `issue/collaboration-interactions.spec.ts`<br>C7: top navigation keeps a multi-select command open for confirmed label changes and respects IME | 通过 | 通过 | 通过 |
| 43 | `issue/i18-search.spec.ts`<br>I18 (page): top navigation owns native search, IME and late answers across Back while the modal stays closed | 通过 | 通过 | 通过 |

## T14 执行快照

T14 时前 41 行在每个引擎都执行通过，0 skip。证据根为本 worktree 的 `.superpowers/sdd/P15-web-collaboration-drafts-search/`；每个同名前缀的 .json/.log 保留命令、输入/二进制摘要、退出码和完整结果，-report/-results 保留报告及产物。

| 引擎 | 完整运行 | 最终受影响补跑 | 结论 |
|---|---|---|---|
| Chromium | t14-matrix-chromium-final，41/41 | 无 | 41 行通过 |
| Firefox | t14-matrix-firefox-fixed，41/41 | t14-history-firefox-final，I13 的 3/3 | 38 项复用不变行为，3 项新前置复验 |
| WebKit | t14-matrix-webkit-fixed，41/41 | 无 | 41 行通过 |

Chromium/WebKit 完整运行及 Firefox 补跑输入摘要 `85b650eaa2d3796749c5d8d6a4529e9ecf9506a19b26eaf4412fb6f8c6217a39`；Firefox 完整运行摘要 `a2e4887206b6969284ff1e96f7624bbc9d20dd6ad3aae28bec82a59ba211ad33`，此后只改 I13 前置、WebKit 专属键盘/精确错误期望和单测 mock 的 role。二进制均 `4e7df898b364b478f07ab515ab075861e389a874e521ef544c46c8df19fa3bfe`。逐文件等价差异与 F1–3 输入复用见 t14-input-equivalence.json；最后追加执行脚本只登记已有矩阵为真实入口。

Firefox 的 LAN 表单/execCommand 两条安全 warning 必须按次数、精确文本及同源实际 JS 请求断言；WebKit/Chromium 的拒绝资源 console error 按预期状态和数量核对。其它 console、CSP、旧 API、站外请求门槛不变。macOS WebKit 用其原生 Option+Tab 到按钮；I13 先完成真实列表加载再安装 route，避免初始页面禁缓存造成预加载诊断，所有业务应答仍经原服务。


## T16 最终复验

T16 增加顶部宿主的 C7/I18 两页，矩阵现为 43 行。先完整执行 `t16-matrix-chromium/firefox/webkit`，各 43/43、0 skip，输入摘要 da5df5317790e97092ba9e7af4bca95f50cc62e3bea2393cd651594c5f8aa20a，程序 6f2f06670ccc90dbbe631d021e7b976b03e80aced5017d287074bfccf0b4a7e1。

| 引擎 | T16 完整矩阵 | 退场快捷键修复后的相关项 | 退场选择动作保护后的主题页 |
|---|---|---|---|
| Chromium | t16-matrix-chromium，43/43 | t16-final-commands-chromium，13/13 | t16-exit-selection-chromium，1/1 |
| Firefox | t16-matrix-firefox，43/43 | t16-final-commands-firefox，13/13 | t16-exit-selection-firefox，1/1 |
| WebKit | t16-matrix-webkit，43/43 | t16-final-commands-webkit，13/13 | t16-exit-selection-webkit，1/1 |

中间 13 项为四个 C7、六个 I18 和三个命令复制 C8，输入 0fda47026d3aff027f750a6c92dbb664abdcfc153e42d5cff22851c8c587f313，程序 5a124a96ca93084a9291694bd9f163253337250a8321924c4389ace0bc1b3182。最后主题页增加逻辑关闭期间不得进入子页的断言，真实 CSS 退场中验证动作停用、Cmd/Ctrl+K 重开与确认主题后的 reload；输入 1310214ac0aee3ccdb66ba5f0a6793c4b71b952a01d8018e68866ece4f41d5de，程序 9be9d742306019dd0a990422cce39bdd72495a32ac75c1c3c68d2ef22fa06c6d。

输入差异、hash 与复用范围保留在 `t16-final-matrix-equivalence.json`：第一轮之后只改 wrapper 的关闭边界、删除无人调用的 setEnabled、相应测试与文稿，另外 30 行行为不变；第二轮之后只增加已关闭宿主的选择保护和对应单元/主题页，打开时的其它 12 项合同不变。表中明确分别复用和补跑，不将最后输入冒称完整执行 129 项。默认完整 E2E 与最终 CI 的身份/结果见 [阶段记录 §10](P15-web-collaboration-drafts-search-review.md#10-最终验收与交付)。

先前实际失败继续保留：T14 CI 的 C7 退场重开失败、T16 顶部入口/关闭重开断言以及两个新夹具调用错误均在原始目录。后者不是产品缺陷；前者有根因修复与有效红绿，不归类为 W11 或用重跑代替修复。

最终 F4 发现测试样式清理的 Node.remove 类型不匹配，改用 parentNode.removeChild；没有改产品代码或断言。最终输入 3531565df6210e93b8b8ec3e940cb7a411af9c6c48e0f40883aa84c8472ca5f5 下，完整默认 E2E 279/279 已包含 Chromium 主题页，随后 t16-fixture-types-browsers 用单 worker 顺序补验 Firefox、WebKit 的同一主题页，2/2、0 skip，程序摘要仍为 9be9d742306019dd0a990422cce39bdd72495a32ac75c1c3c68d2ef22fa06c6d。此前单独 Firefox 运行在 globalSetup 的临时数据库端口绑定等待超时，未进入页面测试；保留原失败收据，不算产品失败或有效反证。
