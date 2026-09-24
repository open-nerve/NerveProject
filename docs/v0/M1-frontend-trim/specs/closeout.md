# M1/收尾 `closeout`：M1 收尾 Spec

- 上级设计：[M1 设计](../M1-design.md)（6、7.3–7.6、9 节"收尾"、11、12 节）、[总体设计](../../v0-design.md)（9.4）
- 前一 Phase：[P5 spec](P5-brand.md)、[P5 plan](../plans/P5-brand.md)、[P5 review](../reviews/P5-brand-review.md)（第 4 节裁定、第 5 节计划缺陷、第 7 节交给收尾的事项）
- 实施计划：[收尾 plan](../plans/closeout.md)
- 分支：`worktree-m1-closeout`，基线 `c493255`（P5 合并后的 `main`）

---

## 1. 目标

让 M1 设计第 11 节的完成标准成立，并清掉 P1–P5 留给收尾的全部事项。收尾不删产品功能，删的是从不渲染、没有读取方的代码和资源：

- **安全**：每个 `window.open` 都带 `noopener,noreferrer`，守卫看住；propel 的菜单项不再调用全局的 `window.close()`。
- **死代码**：从不渲染的应用栏；没有其他文件读取的工作区包导出（三轮，512 个）；各评审点名的死代码；基线的退化结构（`={"…"}` 等 80 处、别名、模板字面量、206 行悬空的导入分组注释）。
- **死资源**：没有代码引用的文案键（每种语言 1599 → 1077 个）、没有被导入的图片（133 张）。
- **守卫**：顶层 `phase` 为 `M1/closeout`，工具据此报出过期的例外，只剩跨 M 的 3 条；每条规则的每个分支都有命中样本，不命中样本都引用保留的代码。
- **工具链与依赖**：删掉 `sanitize-html`（开发环境的外部化警告的根源），给 `prosemirror-codemark` 打补丁（测试输出的 5 行噪声），格式检查覆盖两个配置包和根目录的工具链配置；编辑器标注块的属性按字符串读取。
- **报告与文档**：oxlint 按包、按规则的表和清零计划，构建体积与 P1 基线的对比，最后一次锁文件核对；前端改动清单；P1–P5 交给后续 M 的交接都写上关闭条件，收尾自己的交接（死成员和死 prop、oxlint 的清理）按领域交给 M2–M8。

死成员和死 prop（类型检查器找得到、knip 和 tsc 都不报的 1346 个）不在收尾删：它们几乎都在 M2 起逐个领域重写的代码里（第 4 节第 4 条），按领域交给后续 M。

---

## 2. 分诊：交给收尾的全部事项

来源：M1 设计 9 节"收尾"、11 节；P1 评审第 6 节；P2–P5 评审第 7 节；P2–P5 spec 的"交给 M1 收尾"；M1 的交接。"基线实测"是在 `c493255` 上量的。去向：**T*n*** 是收尾的 Task（第 3 节）；**交 M*n*** 写进该 M 的 `handoffs/`，带关闭条件；**不做**写明理由。

### 2.1 安全

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| A1 | `window.open` 没有 `noopener`：编辑器链接的点击处理（别的成员写的链接）、附件列表、图片下载、全屏弹窗 | P5 评审第 7 节（整分支评审 M6） | web 里 16 处 `window.open(`，12 处（11 个文件）不带 `noopener`；其余 4 处（帮助菜单、命令面板的两个帮助命令、兄弟工作项）已带 | **T2**：12 处都传 `"noopener,noreferrer"`，同源的"在新标签页打开"也一样；守卫规则 `window-open` |
| A2 | 其他 `target="_blank"` 没有 `noopener`/`noreferrer` | 收尾任务书 | JSX 里 `target` 可能为 `"_blank"` 的元素 18 个，都带 `rel="noopener noreferrer"`（`t2/blank.mjs`）；ui 的 `ControlLink` 默认 `target="_blank"`、不带 `rel`，只用于站内地址 | **不做**：浏览器对 `<a target="_blank">` 默认按 `noopener` 处理，`ControlLink` 的地址都是同源的 |
| A3 | propel 的 `MenuItem` 每次点击调用全局的 `close()`，即 `window.close()` | 收尾原型中发现 | `propel/src/menu/menu.tsx` 一处；oxlint 没有禁用这类全局名，另一处 `use-reload-confirmation` 的 `confirm()` | **T5**：删掉这次调用；oxlint 的 `no-restricted-globals` 设为错误 |

### 2.2 死代码

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| B1 | 从不渲染的应用栏，包括 `use-workspace-paths.ts` | P4 评审第 4 节裁定 9、第 7 节 | `AppRailVisibilityProvider` 默认关闭，唯一的使用处（工作区布局）不打开它；8 个文件、`@nerve/types` 的 3 个类型、propel 右键菜单里只有它用的部分 | **T3**；守卫 `app-rail`、`app-rail-files` |
| B2 | 没人用的包导出 | P3 评审第 7 节（347 个，`symref.mjs`）；P3 spec 7.4 | 类型检查器（`deadsym.mjs`）：426 个导出没有别的文件读取 | **T4**：三轮删完（512 个），剩 2 个（`api-client` 类型测试文件里的两个导出，文件本身就是检查） |
| B3 | `convertHexEmojiToDecimal`、`emojiCodeToUnicode`、`TrailingNode` 的导出 | P3、P4 评审第 7 节 | 没有调用方 | **T4**（属于 B2） |
| B4 | `useProjectIssueProperties` 的 5 个 fetcher 只有 `fetchCycles` 有调用方；P3 说的 `fetchStates` | P3、P4 评审第 7 节 | 同左 | **T5**：hook 删除，工作项表单直接调用迭代 store 的 `fetchAllCycles` |
| B5 | `IssueFormRoot` 挂载时的重置（可能是空操作） | P3 评审第 7 节 | 把表单重置为 `useForm` 刚用过的初始值，是空操作 | **T5** |
| B6 | `ui/empty-space.tsx` 没人传的属性和只有一个子元素的片段 | P5 评审第 7 节 | `EmptySpace` 的 `Icon`、`EmptySpaceItem` 的 `description`；2 个片段 | **T5** |
| B7 | knip 看不见的死成员和死 prop | P3 评审第 7 节（启发式脚本：182 个成员、205 个 prop） | `deadsym.mjs`：成员 1150、prop 638 | 收尾删掉自己的 Task 造成的和评审点名的（→ 成员 894、prop 452）；其余**按领域交 M2–M8**，共享部分交 M8（第 4 节第 4 条，第 8 节） |
| B8 | `={"…"}` 这类写法 | P4 评审第 7 节 | `react/jsx-curly-brace-presence`：80 处，58 个文件 | **T6**：oxlint 修掉，规则设为错误 |
| B9 | 只给导入常量起别名的 3 个 `const`；`no-projects.tsx` 没有插值的模板字面量 | P4 评审第 7 节 | 另有一处同类别名（打盹弹窗） | **T6** |
| B10 | `` t(`${getDescriptionPlaceholderI18n(…)}`) `` 外层多余的模板 | P3 评审第 7 节 | 1 处 | **T6** |
| B11 | `use-issues-actions.tsx` 的 `viewId as TProfileViews` | P4 评审第 7 节 | 1 处 | **交 M4**：根治要改工作项 store 共用的 hook 接口，M4 重写工作项 store 时一起做。关闭条件：`git grep -n "as TProfileViews" -- web` 没有输出 |
| B12 | `helpers/index.ts` 仍导出 `ensureAPITrailingSlash` | P1 评审第 6 节 (d) | 桶文件已不存在；这个函数在 `services/src/helpers/url.ts` 里被 `normalizeAPIRequestURL` 使用，并有测试 | **不做**：已在 P3、P4 解决 |

### 2.3 死文案

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| C1 | 没有代码引用的键（严格方法："整键字面量 + 模板前缀"，用变量调用的键追到常量） | M1 设计 6、9 节；P2、P3、P5 评审第 7 节 | 每种语言 1599 个键，`keyref.mjs unused` 482 个 | **T11** |
| C2 | 名字与无关的字面量相同、因而"有引用"的键 | P3 评审第 7 节 | `keyfalse.mjs`：34 个键只被"不像翻译调用"的字面量命中 | **T11**：逐个追查，18 个的字面量都到不了 `t()`（表单字段、枚举值、路由段、`AlertModalCore` 未翻译的默认值等），删除；16 个到得了，保留（第 4 节第 12 条） |
| C3 | 5 个 `themes.theme_options.*.label` 键：删掉，或者让 `THEME_OPTIONS` 用上它们 | P2 评审第 7 节（整分支评审 M6） | 没有代码读取；`THEME_OPTIONS` 另有 `key`（设置页用来翻译）和 `i18n_label`（英文原文，Power K 的主题菜单拿它当键，中文界面下显示英文） | **T11**：`i18n_label` 改为 `common` 里已有的键，两处都读它，`key` 删除；这 5 个键是第三份，删除（第 4 节第 13 条） |
| C4 | 只被 T4 删掉的代码提到的键 | 收尾（孤儿由造成它的 Task 删） | T4 之后 `keyref.mjs orphaned` 22 个 | **T4** |

### 2.4 死图片与来源

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| D1 | 没有引用的图片（有的写着 Plane 的名字） | P2、P3、P5 评审第 7 节 | `assets.mjs`：246 张中 133 张 | **T12**：逐张看过后删除（第 4 节第 14 条） |
| D2 | 29 张封面照片的来源和许可；导览截图 `onboarding/cycles.webp`、`modules.webp`、`views.webp` 里的真人头像和一位 Plane 联合创始人的名字 | P5 spec 7.3、P5 评审第 7 节 | 见第 9 节第 1 条 | **待裁定**（第 9 节第 1 条）；收尾不删也不换 |

### 2.5 守卫

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| E1 | 只剩明确跨 M 的例外；`until` 超出 M8 的逐条重新核对 | M1 设计 7.4、11 节；P2、P3 评审第 7 节（`tlds.ts` 的两条 `M9`）；P5 spec 7.3（`brand` 的 `M9`） | 5 条：`analytics` 到 M6、`project-invitations` 到 M3、`brand`（`pnpm-workspace.yaml`，3 处）到 M9、`tlds.ts` 的 `analytics`、`wiki` 到 M9 | **T1**：顶层 `phase` 改为 `M1/closeout`，工具据此判断到期（M1 内到期的例外都会报错）；**T4** 删掉 `tlds.ts`（它唯一的读取方是没人调用的地址解析）和它的 2 条例外；`brand` 的例外重新核对：3 行注释记录配置来自 Plane 的哪个提交，理由成立，保留到 M9。结束时 3 条 |
| E2 | 规则作用范围的边界（`storybook` 的 `files` 不含 `Makefile`、`.github/**`；`deploy-files` 锚定 `^web/`；`storybook-files` 的一个分支没有样本） | P1 评审第 6 节 (a) | `alts.mjs`：42 个分支没有命中样本 | **T1**：每个分支（含 `files` 选择器）都有命中样本；**不扩大范围**：这些词在 `Makefile`、`.github/`、`deploy/`、`e2e/`、`tools/` 里都没有命中 |
| E3 | 从未存在过的样本路径（约 20 条规则写着 `web/apps/web/app/assets/logo.svg`） | P5 评审第 7 节 | `missquote.mjs`：50 个不命中样本引用的代码或路径不存在（26 个是 `logo.svg`） | **T1**：替换 58 个、删除 7 个，结束时 0；之后删掉或改名被引用代码的 Task 同一提交改样本（T11 一处） |
| E4 | 本 Phase 的 `phase` | 收尾任务书 | `M1/P5` | **T1** |

### 2.6 工具链与依赖

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| F1 | 最后一次锁文件核对 | M1 设计 9 节、9.7 | — | 第 3.5 节 |
| F2 | 开发服务器的 "externalized for browser compatibility" 警告 | P4 评审第 7 节（66 条） | `devwarn.mjs` 打开 `/`：页面加载两次，共 44 条（每次 22 条：`path` 11、`source-map-js` 6、`url` 3、`fs` 2），服务端同样 44 行；根源是 `sanitize-html` 把 postcss 带进浏览器端 | **T8**：删掉 `sanitize-html`，三个 HTML 工具改用 `DOMParser`；结束时 0 |
| F3 | 编辑器测试输出里 `prosemirror-codemark` 的 5 行 sourcemap 警告 | P3 评审第 4 节裁定 6、第 7 节 | 5 行；0.4.2 是最新版本（2022 年 11 月），发布的 source map 指向没有发布的 `src/*.ts` | **T9**：pnpm 补丁删掉 12 个构建文件末尾的 `sourceMappingURL` 注释；测试输出干净 |
| F4 | 格式检查不覆盖 `tailwind-config`、`typescript-config` 和根目录的文件 | P1 评审第 6 节 (b) | 两个包没有 `check:format`；根目录只检查 `tools/` | **T10**：两个包加 `check:format`；根目录加 6 个工具链配置文件；`make lint-web` 52 → 54 个任务 |
| F5 | 路由匹配测试的宽松之处 | P4 spec 第 3 节第 8 条、P4 评审第 7 节 | 测试只核对能静态解析的跳转目标 | **不做**：收紧要给跳转目标的表达式写取值推断，是给临时的静态扫描再造一个小解释器；P4 已有两条反向断言和"改坏再恢复"的核对，路由的行为另由控制者的探测每个 Phase 重跑 |
| F6 | 编辑器标注块的 `data-emoji-unicode` 声明为 `string`，运行时是数字 | P4 评审第 7 节 | TipTap 默认的属性解析把 `"128161"` 转成数字，`logo-selector.tsx` 因此 `.toString()` | **T7**：属性用 `element.getAttribute` 读取，加测试 |
| F7 | 两张加载动画 GIF | P1 评审第 6 节 (c) | 已不存在 | **不做**：P5 Task 3 已替换 |

### 2.7 小项

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| G1 | 导入分组的错标：`app/root.tsx` 悬空的 `// types`，4 对相邻同名分组 | P5 评审第 7 节 | `labels-all.mjs`：悬空 208 行、重复 4 行 | **T6**：同类的全部删掉（悬空 206 行，另 2 行随 T4 删掉的文件消失；重复 4 行），不移动导入 |
| G2 | `server/internal/platform/webui/handler_test.go` 编造的夹具文件名 | P5 评审第 7 节 | `manifest.json`、`favicon/android-192.png`；`handler.go` 不依赖具体文件名 | **待裁定**（第 9 节第 2 条）：改的是 `server/`，不在 web 之内 |
| G3 | 错误页插图的 `alt="ProjectSettingImg"` | P5 spec 第 5 节 | 3 张装饰插图（错误页、维护页、无权限页） | **T6**：`alt=""` |
| G4 | `list-view-types.d.ts` 从 propel 不导出的子路径导入 `TPlacement` | 收尾原型中发现 | 在 `skipLibCheck` 下导入静默失败，类型是 `any`，掩盖了快捷操作把它传给 ui `CustomMenu`（后者的 `placement` 没有 `"auto"`） | **T6**：改为 `CustomMenu` 的 `placement` 类型 |
| G5 | 13 个来自 Plane 的 `package.json` 写 `"license": "AGPL-3.0"`（SPDX 已废弃的标识） | 收尾原型中发现 | web 应用和 12 个 `@nerve/*` 包；Nerve 自己的根目录、`api-client`、`e2e` 写 `AGPL-3.0-only` | **交 M8**：与版本号（P5 交 M8）是同一批发布元数据。关闭条件：`git grep -n '"license": "AGPL-3.0"' -- '*package.json'` 没有输出 |

### 2.8 收尾要交的报告和文档

| # | 事项 | 来源 | 去向 |
|---|---|---|---|
| H1 | 按包、按规则的 oxlint 警告数和清零计划 | M1 设计 7.3、11 节 | 第 3.3 节；计划写进 M2–M8 的交接（**T13**）；收尾 review 复述 |
| H2 | 构建体积与 P1 基线的对比（同一条命令） | M1 设计 7.6；P1–P5 spec | 第 3.4 节 |
| H3 | 锁文件核对 | M1 设计 9、9.7 节 | 第 3.5 节 |
| H4 | 第 11 节的完成标准逐项核对 | M1 设计 11 节 | 第 5 节 |
| H5 | 前端改动清单（第二、四节和第 3 节带来的增减）同步 | M1 设计 9、11 节 | **T13**（1.6 节、第二节两行）；第四节在 P5 已全部"已完成" |
| H6 | 本 M 的交接全部关闭；交给后续 M 的交接都有接收的 M 和关闭条件 | M1 设计 9、11 节 | `M0-P5-frontend-trim-notes`、`M0-P6-knip-notes` 在基线已是 `closed`；P2–P5 写给 M2–M8 的 17 份交接各加一节"关闭条件"，收尾自己给 M2–M8 各写一份（**T13**） |
| H7 | M1 设计 12 节收尾一行、11 节打勾、总体设计 9.4 和 M1 的状态改为"已完成" | M1 设计 9、11 节 | 控制者在收尾 review 的提交里改（与 P5 相同）；时机见第 9 节第 3 条 |

**规模**：13 个 Task，改 763 个文件（+1371 / −12685 行），其中大部分是机械的：T4 的三轮删除、T6 的 oxlint 修复和注释、T11 的键、T12 的图片都由脚本完成，脚本的输出逐项可核对。要逐处判断的是 T2、T3、T5、T6 的手写部分、T7、T8。死成员和死 prop 若在收尾删，要在 348 个文件里逐个判断 1346 处（很多是 Plane 接口类型的字段，M2 起按 OpenAPI 生成的类型整体替换），不适合放在一个 Phase 里，交给重写这些代码的 M。

---

## 3. 交付物

### 3.1 文件总览

13 个 Task，每个一个提交。原型（`c493255..328843f`，在 `$COTMP/proto`）实测 `763 files changed, 1371 insertions(+), 12685 deletions(-)`。

| Task | 标题 | 删 | 增 | 改 | 行（+/−） | 原型提交 |
|---|---|---:|---:|---:|---|---|
| 1 | 守卫认识收尾；每个分支都有命中样本，不命中样本都引用保留的代码 | — | — | 2 | +160 / −94 | `c9eeb28` |
| 2 | 每个 `window.open` 都带 `noopener,noreferrer`；规则 `window-open` | — | — | 12 | +42 / −12 | `5eb9494` |
| 3 | 从不渲染的应用栏；规则 `app-rail`、`app-rail-files` | 8 | — | 9 | +73 / −453 | `7b802a8` |
| 4 | 没人用的包导出（三轮）、`tlds.ts`、它们留下的 22 个键 | 166 | — | 158 | +165 / −9186 | `cccf8be` |
| 5 | 评审点名的死代码；菜单项不再调用 `window.close()`；`no-restricted-globals` | 1 | — | 10 | +95 / −274 | `30c798d` |
| 6 | 退化结构、错标的导入分组注释、装饰插图的 `alt`、`TPlacement` | — | — | 216 | +101 / −341 | `17becf9` |
| 7 | 标注块的属性按字符串读取 | — | 1 | 2 | +72 / −2 | `344ab76` |
| 8 | 删掉 `sanitize-html`，HTML 工具改用 `DOMParser` | — | 1 | 5 | +60 / −173 | `ec6c3e5` |
| 9 | `prosemirror-codemark` 的补丁 | — | 1 | 3 | +126 / −4 | `2d51151` |
| 10 | 格式检查覆盖两个配置包和根目录的配置 | — | — | 4 | +18 / −15 | `3cb038f` |
| 11 | 没有代码引用的文案；主题选项的标签只剩一个来源 | — | — | 33 | +54 / −1656 | `b13b3d8` |
| 12 | 没有被导入的图片 | 133 | — | — | +0 / −475 | `3ebfe70` |
| 13 | 前端改动清单、README、交接 | — | 7 | 19 | +407 / −2 | `328843f` |

### 3.2 原型验证：结论与证据

在 `$COTMP/proto`（从本分支的基线克隆）中逐个 Task 做过一遍，每个 Task 结束时 `pnpm exec turbo run check:types`、`make lint-web`、`make test-web`、`make build-web`、`make knip` 都通过。之后在一个从基线新建的克隆（`$COTMP/replay`）上按 plan 的步骤重放了全部 13 个 Task（`replay.mjs`），每个 Task 得到的树与原型提交完全相同；T4–T13 的五个门禁和测试输出的核对又在重放的提交上重跑了一遍（`gates-range.mjs`，T1–T3 的原型提交没有变），全部通过。

| 项 | 基线（`c493255`） | 收尾结束（`328843f`） |
|---|---|---|
| `make lint-web` | `keywords: 48 rules, 5 exceptions, no hits.` + 52 个 turbo 任务 | `keywords: 51 rules, 3 exceptions, no hits.` + 54 |
| `check:types` / `make test-web` / `make build-web` | 23 / 16 / 11 | 不变 |
| vitest | 77 个测试（web 5、constants 11、editor 16、i18n 10、services 11、utils 24），stderr 有 editor 的 5 行 `prosemirror-codemark` | 86 个（editor 20、utils 29），没有 stderr |
| `make knip` | 零 | 零（每个 Task） |
| lint 上限合计 | 711 | 696（hooks 4→3、propel 21→16、ui 28→25、utils 25→19）；`lintdiff.sh` 没有新增的警告 |
| 守卫：没有命中样本的分支（`alts.mjs`） / 引用不存在代码的不命中样本（`missquote.mjs`） | 42 / 50 | 0 / 0 |
| 例外 | 5（`analytics` M6、`project-invitations` M3、`brand` M9、`tlds.ts` 的 `analytics`、`wiki` M9） | 3（前三条） |
| 不带 `noopener` 的 `window.open` | 16 处中 12 处 | 0（规则 `window-open`） |
| 包导出 / 死成员 / 死 prop（`deadsym.mjs`） | 426 / 1150 / 638 | 2 / 894 / 452 |
| 每种语言的键 / 无引用的键（`keyref.mjs`） / 只被无关字面量命中的键（`keyfalse.mjs` 的 `?`） | 1599 / 482 / 34 | 1077 / 0 / 12（都到得了 `t()`） |
| 图片 / 没有引用的（`assets.mjs`） | 246 / 133 | 113 / 0 |
| 导入分组注释：悬空 / 重复（`labels-all.mjs`） | 208 / 4 | 0 / 0 |
| `react/jsx-curly-brace-presence` | 80 处，58 个文件（规则关闭） | 0（错误级别） |
| 只给导入常量起别名的 `const` | 4 | 0 |
| 开发服务器的外部化警告（`devwarn.mjs`，打开 `/`） | 浏览器 44 条、服务端 44 行 | 0 / 0 |
| 锁文件 | — | 删 17 个包，不新增、不升级（3.5） |
| 构建体积合计 | 534 个文件，17,306,349 字节 | 533 个，17,071,770 字节（3.4） |

每个 Task 结束时的中间值（`pertask.mjs` 在重放的提交上重算）：

| Task | 导出 / 成员 / prop | 上限合计 | 每种语言的键 | 没有引用的图片 | 规则 / 例外 | 注释：悬空 / 重复 |
|---|---|---:|---:|---|---|---|
| 基线 | 426 / 1150 / 638 | 711 | 1599 | 133 / 246 | 48 / 5 | 208 / 4 |
| T1 | 426 / 1150 / 638 | 711 | 1599 | 133 / 246 | 48 / 5 | 208 / 4 |
| T2 | 同上 | 711 | 1599 | 133 / 246 | 49 / 5 | 208 / 4 |
| T3 | 426 / 1146 / 635 | 711 | 1599 | 133 / 246 | 51 / 5 | 208 / 4 |
| T4 | 2 / 895 / 468 | 697 | 1577 | 133 / 246 | 51 / 3 | 206 / 4 |
| T5 | 2 / 893 / 452 | 696 | 1577 | 133 / 246 | 51 / 3 | 206 / 4 |
| T6 | 2 / 893 / 452 | 696 | 1577 | 133 / 246 | 51 / 3 | 0 / 0 |
| T7–T10 | 2 / 894 / 452 | 696 | 1577 | 133 / 246 | 51 / 3 | 0 / 0 |
| T11 | 2 / 894 / 452 | 696 | 1077 | 133 / 246 | 51 / 3 | 0 / 0 |
| T12、T13 | 2 / 894 / 452 | 696 | 1077 | 0 / 113 | 51 / 3 | 0 / 0 |

每个 Task 的孤儿核对（基点是上一个 Task 的提交）：`symref.mjs orphaned`、`dangling.mjs`、`keyref.mjs orphaned` 都是 0，`headers.sh` 没有输出；`deadorph.mjs`（新出现的死成员、死 prop、死导出）除 T7 外都是 0；`infile-orphans.mjs` 在 T3、T4、T5 列出 12、165、4 行，都以 `defined now: 0)` 结尾（被删掉的符号），其余 Task 为 0。

结论：

1. **原型的步骤可以照做。** 重放发现 T3、T4、T6、T8 要在脚本之后跑一次 oxfmt，plan 已写进步骤；重放之后又发现 T4 让 propel `EmptyState` 的 `asset` 属性成了孤儿（`deadorph.mjs` 报 1），改在 T4 删掉，T11 补了主题选项（C3），重放和门禁都重跑过。
2. **包导出要用类型检查器找。** knip 把每个包的入口都当作已使用；`symref.mjs` 按名字数引用，同名的局部变量、属性也算。`deadsym.mjs` 用 TypeScript 的程序模型：一个导出只有在它自己以外的文件读取它时才算使用（经过任何桶文件；重新导出不算读取）。删一轮会让上一层的导出变成没人用，所以要删到某一轮为零为止（第 1 轮 424 个、第 2 轮 26 个、第 3 轮 62 个、第 4 轮 0）。
3. **T7 的 `deadorph` 报 1 是已知的例外。** 新加的 `parseHTML` 写在属性对象的类型里，由 TipTap 在解析 HTML 时读取，脚本只看项目自己的代码；T7 的测试证明它被读取（去掉它，第一个用例以 `128161` 失败）。
4. **外部化警告的根源在依赖，不在 Vite 配置。** 44 条来自 postcss 读取 `path`、`fs`、`url`、`source-map-js`；web 里只有 `sanitize-html` 把 postcss 带进浏览器端，它的全部用处是 `@nerve/utils` 的三个 HTML 工具。应用是纯客户端的构建（`ssr: false`），浏览器自带的 `DOMParser` 就够用；删掉之后警告为 0，还修掉了通知预览显示 `&amp;` 的问题（3.10）。
5. **测试输出的噪声也是依赖的问题。** `prosemirror-codemark` 0.4.2 是最新版本，发布的每个构建文件末尾都指向一个列出未发布源文件的 source map；补丁只删这 12 行注释（`strip.mjs`），与仓库里已有的 `react-color` 补丁同一种做法。
6. **没有新的共享可变状态。** 全 Phase 新增的顶层声明都是函数、只读数据或测试辅助；唯一的模块级可变值是 T7 测试文件里等待 `afterEach` 销毁的编辑器列表。新增的行里没有 `={"`、`${"`，也没有不带插值的模板字面量。
7. **构建产物几乎不变。** 删掉的图片本来就不在构建里（没有被导入）；JS 少 231,564 字节（propel 和 ui 的死组件、`sanitize-html`），CSS 少 3,015 字节（3.4）。

### 3.3 oxlint：按包、按规则（M1 设计 7.3）

`linttable.mjs` 与 `tools/lint-cap.mjs` 同样的跑法（在每个包目录里 `oxlint --format=json .`），每个包的合计等于上限。基线 711 个：web 566、editor 65、ui 28、utils 25、propel 21、hooks 4、constants 1、i18n 1。收尾结束时 696 个，40 条规则：

| 规则 | web | constants | editor | hooks | i18n | propel | ui | utils | 合计 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `eslint(no-shadow)` | 136 | 1 | 22 |  |  | 8 | 3 | 1 | 171 |
| `eslint-plugin-jsx-a11y(no-static-element-interactions)` | 65 |  | 9 |  |  | 1 | 4 |  | 79 |
| `eslint-plugin-promise(always-return)` | 75 |  | 1 |  |  |  |  |  | 76 |
| `eslint-plugin-jsx-a11y(click-events-have-key-events)` | 61 |  | 5 |  |  | 1 | 4 |  | 71 |
| `eslint(no-unneeded-ternary)` | 55 |  |  |  |  |  | 1 |  | 56 |
| `eslint-plugin-react-hooks(exhaustive-deps)` | 35 |  | 1 |  |  |  | 7 |  | 43 |
| `eslint-plugin-react(no-array-index-key)` | 20 |  |  |  |  | 3 | 1 |  | 24 |
| `eslint-plugin-unicorn(consistent-function-scoping)` | 15 |  | 4 | 2 |  | 1 |  |  | 22 |
| `eslint-plugin-jsx-a11y(no-autofocus)` | 17 |  | 3 |  |  | 1 |  |  | 21 |
| `eslint(no-unused-expressions)` | 13 |  |  |  |  |  | 2 | 1 | 16 |
| `eslint-plugin-unicorn(prefer-set-has)` | 9 |  |  |  |  | 1 |  | 2 | 12 |
| `eslint-plugin-jsx-a11y(tabindex-no-positive)` | 9 |  |  |  |  |  | 1 |  | 10 |
| `eslint-plugin-import(no-named-as-default)` |  |  | 8 |  |  |  |  |  | 8 |
| `eslint-plugin-unicorn(no-array-sort)` | 4 |  |  |  |  |  |  | 4 | 8 |
| `eslint(no-constant-binary-expression)` | 8 |  |  |  |  |  |  |  | 8 |
| `oxc(no-map-spread)` | 4 |  | 2 |  |  |  |  | 1 | 7 |
| `eslint(no-useless-escape)` |  |  |  |  |  |  |  | 6 | 6 |
| `oxc(const-comparisons)` | 5 |  |  | 1 |  |  |  |  | 6 |
| `eslint(preserve-caught-error)` | 5 |  |  |  |  |  |  |  | 5 |
| `eslint-plugin-import(no-unassigned-import)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint-plugin-jsx-a11y(img-redundant-alt)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint-plugin-jsx-a11y(prefer-tag-over-role)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint(no-useless-catch)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint(valid-typeof)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint-plugin-import(no-named-as-default-member)` | 2 |  |  |  | 1 |  |  |  | 3 |
| `eslint-plugin-unicorn(no-array-reverse)` |  |  | 1 |  |  |  |  | 2 | 3 |
| `oxc(no-accumulating-spread)` | 2 |  | 1 |  |  |  |  |  | 3 |
| `eslint-plugin-unicorn(no-single-promise-in-promise-methods)` | 2 |  |  |  |  |  |  |  | 2 |
| `eslint-plugin-unicorn(no-useless-spread)` | 1 |  | 1 |  |  |  |  |  | 2 |
| `eslint-plugin-unicorn(prefer-add-event-listener)` |  |  | 2 |  |  |  |  |  | 2 |
| `eslint-plugin-unicorn(prefer-array-flat-map)` |  |  | 2 |  |  |  |  |  | 2 |
| `eslint(no-empty-pattern)` | 2 |  |  |  |  |  |  |  | 2 |
| `eslint-plugin-jsx-a11y(alt-text)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint-plugin-jsx-a11y(anchor-is-valid)` |  |  |  |  |  |  | 1 |  | 1 |
| `eslint-plugin-react(jsx-no-constructed-context-values)` |  |  |  |  |  |  | 1 |  | 1 |
| `eslint-plugin-unicorn(no-instanceof-builtins)` |  |  |  |  |  |  |  | 1 | 1 |
| `eslint-plugin-unicorn(no-useless-length-check)` |  |  |  |  |  |  |  | 1 | 1 |
| `eslint-plugin-unicorn(prefer-array-find)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint(no-new)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint(no-unsafe-optional-chaining)` | 1 |  |  |  |  |  |  |  | 1 |
| **合计** | 566 | 1 | 65 | 3 | 1 | 16 | 25 | 19 | **696** |

`api-client`、`services`、`shared-state`、`types`、`e2e` 没有警告。

**清零计划**（M1 设计 7.3，写进 M2–M8 各自的 `handoffs/M1-closeout.md`，由 T13 生成）：
1. **谁改谁清**：每个 M 改到的文件，在该 M 结束时没有 oxlint 警告；
2. **按规则清一类**：每个 M 另外按规则集中清掉至少一类，优先能机械修复的 `no-shadow`、`promise/always-return`、`no-unneeded-ternary`，写进该 M 的 review；
3. **M8 发布之前清零**，之后 `.oxlintrc.json` 把警告改为错误、删掉各包的上限（总体设计 7.6 的原始要求）。
每一步上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。

### 3.4 构建体积（M1 设计 7.6）

P1 的方法（P1 spec 2.14）：先 `rm -rf web/apps/web/build`，再 `make build-web`，然后 `node web-size.mjs`（`size.sh`）。

| 项 | P1 之前（`0f2b3e0`） | 收尾之前（`c493255`） | 收尾之后（`328843f`） | 对 P1 之前 |
|---|---|---|---|---|
| JS | 1055 个，15,151,327 字节 | 397 个，6,818,887 | 396 个，6,587,323 | −56.5% |
| CSS | 3 个，326,068 | 3 个，296,816 | 3 个，293,801 | −9.9% |
| 字体 | 34 个，6,849,620 | 25 个，3,755,608 | 25 个，3,755,608 | −45.2% |
| 其他 | 147 个，9,905,117 | 109 个，6,435,038 | 109 个，6,435,038 | −35.0% |
| 最大的 chunk | `toolbar-*.js` 1,821,937 | `use-parse-editor-content-*.js` 1,379,320 | 同名 1,378,717 | |
| 语言 chunk | 588 | 34 | 34 | |
| **合计** | **1239 个，32,232,132 字节（30.7 MiB）** | **534 个，17,306,349（16.5 MiB）** | **533 个，17,071,770（16.3 MiB）** | **−47.0%** |

文件数只作记录，不作为门禁。

### 3.5 锁文件（M1 设计 9、9.7）

`lock-diff.mjs`（P1 的方法）比较基线和终态的 `pnpm-lock.yaml`，除删除以外的变化只有：
- `patchedDependencies` 加 `prosemirror-codemark@0.4.2`，editor 的这条依赖带上补丁的哈希（T9）；
- `web/packages/utils` 的开发依赖加 `jsdom`（`catalog:`，30.1.1，锁文件里已有这个版本，editor、web 在用）（T8）。

删除的 17 个包是 `sanitize-html` 2.17.7、`@types/sanitize-html` 和只有它们用的 15 个（`htmlparser2`、`domhandler`、`domutils`、`dom-serializer`、`domelementtype` 各两个版本，`deepmerge`、`is-plain-object`、`launder`、`dayjs`、`parse-srcset`），以及 `catalogs` 里 `sanitize-html`、`@types/sanitize-html` 两条（T8）；propel 的 `cmdk` 依赖（T4，web 应用自己仍依赖它）。没有新增或升级的包。`stale-config.mjs` 在基线和终态都只报 `postcss` 的 catalog 条目：它经 `overrides` 固定版本、被 `tailwind-config` 使用，是脚本的误报。

### 3.6 守卫（T1、T2、T3、T4、T11）

- **收尾的 Phase**（T1）：`tools/keywords.mjs` 接受 `M<n>/closeout`，排在该 M 的所有编号 Phase 之后、下一个 M 之前。顶层 `phase` 为 `M1/closeout` 时，`until` 是 M1 任何 Phase 的例外都已到期，工具报错，只剩跨 M 的例外，这是 M1 设计 7.4 对收尾的要求，从此由工具检查。`t1/expiry-probe.mjs` 证明：把一条例外的 `until` 改为 `M1/P5`、`M1/closeout` 时以 1 退出（`1 expired exception`），改为 `M2`、`M6` 时通过，写成 `M1/close` 时以 2 退出。
- **样本**（T1）：每条规则的每个顶层分支、每个 `(?:a|b)` 的每一支都有命中样本，`files` 选择器也一样（新增 41 个）；不命中样本都引用保留的代码或路径（替换 58 个，其中 26 个是从未存在的 `app/assets/logo.svg`；在同一边界已有其他不命中样本的删除 7 个）。`missquote.mjs` 用子串在当前的树里找每个不命中样本；T11 删掉 `i18n-applications` 一个样本引用的文案，同一提交改为同一段文字仍在的位置。
- **新规则**：`window-open`（T2：`web/` 的源文件里，`window.open(` 所在的行没有 `noopener`）；`app-rail`（T3：`app[-_ ]?rail|use-?workspace-?paths`，不区分大小写，`web/` 的源文件和 JSON）；`app-rail-files`（T3：路径）。
- **例外**：T4 随 `tlds.ts` 删掉 `analytics`、`wiki` 两条（`until: M9`）；`brand` 在 `pnpm-workspace.yaml` 的 3 处（`until: M9`）重新核对，理由成立（记录配置来源的注释，在 v0 内不会消失），保留。

### 3.7 安全（T2、T5）

- **`window.open`**（T2）：编辑器链接的点击处理（`clickHandler.ts`）、附件列表、图片工具栏的下载、全屏弹窗的下载和打开原图，以及项目卡片、迭代、模块、视图、工作区视图（两处）、工作项的"在新标签页打开"，共 12 处，都传 `"noopener,noreferrer"`。这些调用都不使用 `window.open` 的返回值（带 `noopener` 时返回 `null`）。
- **`window.close()`**（T5）：propel 的 `MenuItem` 从 ui 的 `CustomMenu` 抄来 `close()`，但 propel 这里没有同名的局部绑定，调用的是全局的 `window.close()`。Chrome 会关掉由脚本打开、或只有一条历史记录的标签页，所以在项目标签页的溢出菜单（`TabNavigationOverflowMenu`）里选一项可能关掉整个标签页。base-ui 自己会关菜单，这次调用删除。`.oxlintrc.json` 把 `no-restricted-globals` 设为错误，列表取 ESLint 的 confusing-browser-globals（`close`、`name`、`event`、`status`、`open` 等读起来像局部变量的全局名）；树里唯一有意的用法 `use-reload-confirmation` 的 `confirm()` 改为 `window.confirm()`。

### 3.8 死代码（T3、T4、T5）

- **应用栏**（T3）：删掉 `core/lib/app-rail/`（4 个文件）、`navigation/app-rail-hoc.tsx`、`app-rail-root.tsx`、`items-root.tsx`、`hooks/use-workspace-paths.ts`；`use-navigation-preferences.ts` 里的 `useAppRailPreferences` 和它的本地存储键；`@nerve/types` 的 `TAppRailDisplayMode`、`TAppRailPreferences`、`DEFAULT_APP_RAIL_PREFERENCES`。只有应用栏喂的东西一起走：顶部栏读应用栏的显示模式，那只有应用栏能改，所以它的 `px-2` 分支永远走不到；`AppSidebarItem` 删掉只有应用栏传的 `label`、`showLabel` 和没人读的复合静态成员（`Label`、`Icon`、`Link`、`Button`）；propel 的右键菜单删掉只有应用栏用的 `Separator` 和 `Trigger`、`Content` 的 `className`。
- **包导出**（T4，3.2 结论 2）：每轮由 `t4/round.mjs` 完成：`deadsym.mjs` 列出没人读的导出 → 没有别的读取方、自己文件里也不用的声明删除，自己文件里还用的去掉 `export` → 什么都不再导出的文件连同桶文件里的那一行和包的子路径一起删除 → 这些删除在同一文件里留下的没用的声明和导入删除（按 `lintdiff.sh` 报出的新 `no-unused-vars`）。三轮共 512 个导出：274 个声明删除，283 个去掉 `export`（220 个关键字、63 个说明符），166 个文件删除，其中 propel 的 accordion、avatar、badge、banner、collapsible、combobox、command、dialog、input、skeleton、switch、tabs、toolbar 组件和只有图标注册表提到的 111 个图标文件（应用用的是 `@makeplane/propel` 的图标），ui 的 avatar、collapsible、input、tag、textarea。之后：没有导入方的 16 个包入口（`subpaths.mjs`：propel 的 `animated-counter`、`spinners`，两个样式路径的别名，一个指向不存在目录的 tsdown 入口等）；propel 的 `cmdk` 依赖；`tlds.ts`；propel `EmptyState` 的 `asset` 属性（传它的只有被删的 `EmptyState` 包装组件，`assetKey` 每个调用方都传，改为必填，选择二者的分支删除）；只被删掉的代码提到的 22 个键（用户角色、优先级筛选、两个迭代图标标签和 6 个只在 `tlds.ts` 里出现的顶级域名）。
- **点名的死代码**（T5，B4–B6）和复合组件没人读的部分：propel `ContextMenu.Submenu`、`SubmenuTrigger`；propel `Menu.SubMenu` 和只为它存在的子菜单上下文；ui `CustomMenu` 的 `Portal`、`SubMenuTrigger`、`SubMenuContent` 这三个静态成员（`Portal` 组件本身保留，`CustomMenu` 的子菜单在用）。

### 3.9 退化结构和小项（T6）

- `rulecount.mjs . react/jsx-curly-brace-presence --fix`：oxlint 只用这一条规则修掉基线的 80 处（58 个文件），`.oxlintrc.json` 把规则设为错误。
- 4 个只给导入常量起别名的 `const`（`lite-text/toolbar.tsx`、`issue-layouts/utils.tsx` 两处、打盹弹窗）删除，代码直接读常量；`no-projects.tsx` 没有插值的模板字面量、收集箱描述外层多余的 `` `${…}` `` 删除。
- `labels-all.mjs --fix`：206 行悬空的导入分组注释（标签下面没有导入：叠在另一个标签上，或留在最后一个导入之后）和 4 对相邻同名分组的第二个标签，158 个文件；只删注释行，不移动导入。
- 错误页、维护页、无权限页三张装饰插图的 `alt="ProjectSettingImg"` 改为 `alt=""`（旁边的标题说明页面是什么）。
- `list-view-types.d.ts` 的 `TPlacement`（G4）。

### 3.10 编辑器与 HTML 工具（T7、T8）

- **标注块**（T7）：每个属性用 `parseHTML: (element) => element.getAttribute(name)` 读取；`logo-selector.tsx` 的 `.toString()` 删除。`extension-config.test.ts` 4 个用例：从 HTML 读出的标注块保留 `"128161"` 和其他属性，缺少的属性取默认值，写回 HTML 时码点不变；去掉 `parseHTML` 时第一个用例以 `128161` 失败。
- **HTML 工具**（T8）：`@nerve/utils` 的 `stripAndTruncateHTML` 用 `DOMParser` 取文本，实体会被解码（通知预览显示 "Tom & Jerry"，基线是 "Tom &amp; Jerry"）；只给它用的 `sanitizeHTML` 删除；`isEmptyHtmlString`（`isCommentEmpty` 和草稿弹窗在用）对编辑器的 HTML 给出同样的答案：不间断空格和 `<br>` 算空，允许的标签（图片、提及等）算内容。标注块读自己的本地存储时不再过一遍 `sanitizeHTML`（它会转义 emoji 地址里的 `&`）。`string.test.ts` 5 个用例；utils 的测试在 jsdom 里跑（开发依赖）。

### 3.11 依赖与格式检查（T9、T10）

- **补丁**（T9）：`pnpm patch prosemirror-codemark@0.4.2` → `t9/strip.mjs`（删掉 `dist/esm`、`dist/cjs` 下 12 个文件的 `sourceMappingURL` 注释）→ `pnpm patch-commit`，生成 `patches/prosemirror-codemark@0.4.2.patch`；`pnpm-workspace.yaml` 在这条 `patchedDependencies` 上方用注释写明原因，editor 的 `vitest.config.ts` 的注释写明它现在怎样加载这个包。
- **格式检查**（T10）：`tailwind-config`、`typescript-config` 加 `check:format`、`fix:format`，不加 `check:lint`（它们里面是 CSS 和 JSON；唯一的脚本 `postcss.config.js` 匹配 `.oxlintrc.json` 全局忽略的 `*.config.{js,mjs,cjs,ts}`，oxlint 没有可读的文件）；`typescript-config` 删掉只在打包发布时起作用、又只列了 4 个配置中 3 个的 `files`（包是 `private`）。根目录的 `check:format` 加上 `package.json`、`pnpm-workspace.yaml`、`turbo.json`、`knip.jsonc`、`.oxlintrc.json`、`.oxfmtrc.json`；其中 `package.json` 的键按 oxfmt 的顺序重排、`knip.jsonc` 补上结尾逗号。`t10/negative.mjs` 证明每个新检查对它覆盖的文件改坏时失败、恢复后通过。

### 3.12 文案与图片（T11、T12）

- **文案**（T11）：`t11/theme.mjs` 先让主题选项的标签只剩一个来源（C3）；`keyref.mjs unused` 列出 482 个，`t11/traced.txt` 是追查过的 18 个（C2）；`delkeys.mjs` 从 en、zh-CN 一起删掉 500 个键和它们留下的空对象（每种语言 149 个）。中英文键一致性检查照常通过。
- **图片**（T12）：`assets.mjs` 列出名字没有被 `web/` 下任何文件提到的 133 张图片（应用显示的每张图都按文件名导入），`t12/rm.mjs` 删除它们（6225 KiB）和因此变空的 11 个目录：126 张空状态插图（画的是 Plane 自己的界面，布局、迭代、模块、收集箱、归档、草稿、个人主页、搜索、设置、筛选，深浅和 `-resp` 各一套，以及较早的 `empty_*.webp`、SVG），认证页的无权限插图和两张背景纹理，一个项目 emoji，命令键图形，两张图库人像。删除前拼成 300×200 的联系表逐张看过（`t12/sheet.mjs`）。封面照片和导览截图都被导入，保留（第 9 节第 1 条）。

### 3.13 文档（T13）

- 前端改动清单：新增 1.6 节（收尾的改动，13 行）；第二节加两行（从不渲染的应用栏；Plane 自身的死代码续：包导出、文案键、图片），状态"已完成 / M1/收尾"。
- README：格式检查也覆盖根目录的工具链配置（两处）。
- P2–P5 写给 M2–M8 的 17 份交接，在"来源"之前加一节"关闭条件"：该 M 合并时必须成立的事，最后一句"逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`"。
- 收尾给 M2–M8 各写一份 `handoffs/M1-closeout.md`：本 M 领域里的死成员和死 prop 的数目、列出和处理的方法、关闭条件；oxlint 的清理规则；M4 另有 `viewId as TProfileViews`；M8 另有共享部分的死成员和死 prop、包的 `license` 字段。它们引用本 spec 的 3.3、第 4、8 节和 plan 的附录 A（三个脚本的全文），以及收尾 review（`reviews/closeout-review.md`，控制者写）。

### 3.14 保留行为的核对（M1 设计 7.5）

| 手段 | 核对 | 落点 |
|---|---|---|
| 构建与静态检查 | 五个门禁；守卫；每个 Task 的孤儿核对；`lintdiff.sh`；锁文件 | 每个 Task |
| 进仓库的测试 | 77 → 86 个（T7 的 4 个、T8 的 5 个）；测试输出没有 stderr（T9 起） | 每个 Task |
| 一次性核对 | `devwarn.mjs` 的外部化警告（T8）；`t1/expiry-probe.mjs`（T1）；`t10/negative.mjs`（T10） | 对应 Task |
| 临时核对脚本（控制者写和跑，plan 最后一节） | 重跑 P2–P5 的探测；`window.open` 的参数和 `opener`；菜单项不调用 `window.close()`；通知预览的实体；评论和草稿的"是否为空"；标注块的属性和本地存储；Power K 的主题菜单；页面上没有原样显示的键、没有加载失败的图片；工作项表单的迭代下拉；在基线的构建上做反向对照 | 收尾 review 附录 |
| S1–S4 | 照常通过 | 合并前 |

---

## 4. 与上级设计的差异和补充

每一条都已按"能自己定的就自己定"的原则决定，列出决定和理由，供控制者复核。

1. **守卫认识 `M<n>/closeout`（T1）。** 设计 7.4 说收尾时"只允许剩下明确跨 M 的例外"，原来靠人核对。工具加一种 Phase 写法，排在该 M 的编号 Phase 之后，于是 M1 内到期的例外由工具报错。它改动的是 `tools/keywords.mjs` 的 Phase 解析（一个正则、一个常量），`t1/expiry-probe.mjs` 证明到期、不到期和写错三种情况。
2. **每个 `window.open` 都带 `noopener,noreferrer`，包括同源的"在新标签页打开"（T2）。** P5 评审点名的是打开外部内容的 5 处。同源的 7 处不会泄露给别的源，但一条对所有调用都成立的规则才能由守卫看住；这些调用都不读返回值，加上之后行为不变。`target="_blank"` 的链接不改（A2）。
3. **`window.close()` 的缺陷和 `no-restricted-globals`（T5，补充）。** 在清点菜单的复合组件时发现。只删这一行会留下同类缺陷的入口；ESLint 的 confusing-browser-globals 列表正是为这类"缺了局部绑定、落到全局"的情况设计的，设为错误后树里只有一处有意的用法，写成 `window.confirm`。
4. **死成员和死 prop 按领域交给后续 M，不在收尾删（B7）。** 收尾结束时还有 1346 个（成员 894、prop 452）。按总体设计 9.2 的领域划分（`domains.mjs`，按路径取第一个匹配）：M2 66（59 / 7）、M3 210（142 / 68）、M4 558（435 / 123）、M5 48（40 / 8）、M6 84（46 / 38）、M7 88（63 / 25）、M8 6（6 / 0），共享 286（103 / 183）。理由：
   - M2 起每个 M 按总体设计 7.2 重写本领域的 types、services、stores 和组件，前端的数据结构改用 OpenAPI 生成的类型；成员里的大多数是 Plane 接口类型的字段和 store 的方法，会随重写消失，收尾先删一遍等于做两遍；
   - 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员脚本也会列出（例如 ui 表格列对象的 `thRender`、`tdRender` 由 `useProjectColumns` 写入）；删一个 prop 还要收掉读它的分支。这是逐处的工作，不能写成脚本；
   - 收尾的每个 Task 仍然删掉它自己造成的孤儿（`deadorph.mjs` 每个 Task 为 0，T7 的 `parseHTML` 除外，3.2 结论 3），评审点名的都已删除。
   交接里写明列出的命令、处理方法和关闭条件；共享部分（propel、ui、types、utils、constants、hooks、i18n 和 web 的通用组件）交 M8，M2–M7 改到时照做。
5. **包导出用类型检查器找，删到零为止（T4，3.2 结论 2）。** P3 交接用的是 `symref.mjs`（347 个）；它按名字数，漏掉同名局部变量遮住的死导出，也数不到"只被重新导出"的情况。`deadsym.mjs` 的第一轮就有 424 个。删除后 `infile-orphans` 的 165 行都是被删的符号。
6. **`tlds.ts` 整个删除（T4）。** 两条 `M9` 例外原本要"重新核对理由"。核对发现文件唯一的读取方是 T4 删掉的地址解析链（没有调用方），文件和例外一起删。
7. **propel `EmptyState` 的 `asset` 属性（T4，补充）。** 重放之后的逐 Task 核对（`deadorph.mjs`）发现它在 T4 之后只有读取、没有传入；按"孤儿由造成它的 Task 删"在 T4 删掉，`assetKey` 改为必填。
8. **`react/jsx-curly-brace-presence` 设为错误（T6）。** P4 评审说"收尾统一收掉"。只修不设规则，下一个 M 会再写出来；这条规则 oxlint 能自动修，设为错误不会给后续 M 带来负担。
9. **导入分组的错标全部收掉（T6）。** 评审点名的是 `root.tsx` 的一行和 4 对；`labels-all.mjs` 找到同类的共 206 行（悬空）和 4 行（重复），与 P5 的 `labels.mjs` 同一判断（形如 `// hooks` 的标签，下面第一行不是导入，或与上一组同名）。只删注释行，不移动导入。
10. **`TPlacement` 的类型漏洞（T6，补充）。** 在核对 T4 的导出时发现。`skipLibCheck` 让 `.d.ts` 里失败的导入静默成为 `any`；改为它实际传给的 `CustomMenu` 的 `placement` 类型之后，类型检查照常通过（快捷操作传的值都在 `CustomMenu` 支持的范围内）。
11. **删掉 `sanitize-html`，而不是"记录去处"（T8）。** P4 评审把警告"归到依赖复核"。根因是本地的、小的：三个工具函数，都能用浏览器自带的 `DOMParser` 写，应用没有服务端渲染。行为上的变化只有一处，是修正：通知预览不再把 `&` 显示成 `&amp;`；"是否为空"对编辑器产生的 HTML 答案不变（5 个测试，控制者的探测再核对）。依赖少了 17 个包。
12. **追查名字与无关字面量相同的键（T11）。** 设计 6 节说"不能证明是死键的，先查清所有变量来源，不直接删"。`keyfalse.mjs` 列出 34 个只被"不像翻译调用"的字面量命中的键，逐个追到字面量的用处：18 个到不了 `t()`，是死键，删除；16 个经常量、配置或组件属性到得了 `t()`，保留（T11 之后脚本只再列出其中 12 个：主题选项的 4 个改由 `i18n_label` 引用，脚本认得这种写法）。
13. **主题选项的标签只剩一个来源（T11）。** P2 评审留下的二选一。`THEME_OPTIONS` 同一个标签有三份：`key`（设置页当键用，`common` 里有翻译）、`i18n_label`（英文原文，Power K 的主题菜单当键用，找不到键，中文界面下显示英文）、`themes.theme_options.*.label`（没人读）。按全仓库其他选项列表的写法，`i18n_label` 就是键：改为 `common` 里的键，设置页和 Power K 都读它，`key` 删除，第三份随死键删除。用户看到的变化：Power K 的主题菜单在中文界面下显示中文，英文界面下 "System preference" 变为与设置页相同的 "System Preference"。
14. **删除的图片按 300×200 的联系表看（T12）。** P5 的裁定"按能看清 20 px 细节的尺寸看"针对的是要保留、会显示给用户的图片（找其中的 Plane 标识）。这 133 张不随应用发布，看它们是为了确认删的是什么（都是 Plane 的界面截图、纹理和图库人像，没有被别名或动态路径引用：`assets.mjs` 按文件名查整个 `web/`）。
15. **格式检查的范围（T10）。** P1 评审说"仓库根目录自身的文件"。收尾加的是前端工具链自己的配置（6 个）。`.github/workflows/*.yml`、`deploy/*.yaml` 今天也能通过 oxfmt，但它们属于 M0 的构建与部署，不在 `make lint-web` 的范围里；`README.md`、`docs/` 是文档；`pnpm-lock.yaml` 是生成的。
16. **路由测试的宽松之处不收紧（F5）。** 见第 2.6 节。
17. **17 份交接补"关闭条件"（T13）。** 设计 11 节要求"交给后续 M 的事项已放进对应 M 的 `handoffs/`"，任务书要求每份都有接收的 M 和关闭条件。P2–P5 的交接有接收的 M（目录），没有写什么时候算完成；每份补一节，写该 M 合并时必须成立的事。
18. **M1 的状态由控制者在收尾 review 的提交里改（H7）。** 设计 9 节说收尾时改为"已完成"。实现者的 Task 在评审、修复和合并之前，那时写"已完成"不真实；与 P5 相同，由控制者在 review 的提交里改 M1 设计 12 节、11 节的勾和总体设计 9.4（时机见第 9 节第 3 条）。

---

## 5. 验收标准

### 5.1 M1 设计 11 节逐项

| # | 完成标准 | 本 Phase 的证据 |
|---|---|---|
| 1 | P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review | P1–P5 已有；收尾的 spec、plan（本提交）和 review（控制者） |
| 2 | 类型检查通过；knip 为零且是门禁；oxlint 每个包等于新上限，上限自动核对；格式检查通过；前端单元测试在持续集成中运行并通过 | 每个 Task 的五个门禁；上限 696；格式检查覆盖到两个配置包和根目录（T10）；vitest 86 个 |
| 3 | 守卫在持续集成中运行，没有未登记的命中，只剩明确跨 M 的例外；`until` 超出 M8 的逐条核对过；没有无引用的键 | `keywords: 51 rules, 3 exceptions, no hits.`，`phase` 为 `M1/closeout`（工具检查到期）；`brand` 的 `M9` 例外核对过（3.6），`tlds.ts` 的两条随文件删除；`keyref.mjs unused` 为 0 |
| 4 | 7.5 的保留行为矩阵逐行核对，临时脚本在各 Phase review 的附录 | 控制者的探测（plan 最后一节）重跑 P2–P5 的场景，写进收尾 review 附录 |
| 5 | `make build` 能构建；S1–S4 通过 | 门禁；合并前 S1–S4 和持续集成 |
| 6 | 只剩 `zh-CN` 和 `en`；界面上没有 Plane 的名称和 Logo；包名都是 `@nerve/*` | P1、P5 已完成；守卫的 `plane-package`、`brand`、`brand-files` 照常通过；删掉的 133 张图里的 Plane 界面截图不再在仓库里 |
| 7 | oxlint 清零计划、构建体积对比写进收尾 review | 本 spec 3.3、3.4，review 复述 |
| 8 | `handoffs/` 中没有 `open`；交给后续 M 的事项在对应 M 的 `handoffs/` | M1 的两份在基线已 `closed`；17 份补关闭条件，7 份新交接（T13） |
| 9 | 前端改动清单同步；总体设计中 M1 的状态改为"已完成" | T13；状态由控制者改（第 4 节第 18 条） |

### 5.2 本 Phase

- [ ] 13 个 Task 各一个提交，每个提交结束时 `check:types` 23、`make lint-web` 52（T10 起 54）、`make test-web` 16、`make build-web` 11 个任务通过，`make knip` 为零。
- [ ] `make lint-web`：`keywords: 51 rules, 3 exceptions, no hits.`；上限合计 696；`alts.mjs` 为 0；`missquote.mjs` 为 0。
- [ ] vitest 86 个测试，没有 stderr。
- [ ] 每个 Task 的孤儿核对（基点是上一个 Task 的提交）：`symref`、`dangling`、`keyref orphaned` 为 0，`headers.sh` 没有输出，`deadorph` 为 0（T7 为 `parseHTML` 一行），`infile-orphans` 的每一行都以 `defined now: 0)` 结尾。
- [ ] `deadsym.mjs`：导出 2、成员 894、prop 452；`keyref.mjs unused` 0，每种语言 1077 个键；`assets.mjs` `0 of 113`；`labels-all.mjs` `dangling 0, repeat 0`；`rulecount.mjs . react/jsx-curly-brace-presence` 0。
- [ ] 锁文件与 3.5 相同；`devwarn.mjs` 为 0；构建体积与 3.4 相同（内容哈希除外）。
- [ ] 控制者的浏览器核对全部通过，反向对照在基线上失败的正好是收尾修掉的几项；S1–S4、持续集成通过。
- [ ] 前端改动清单、README、24 份交接与 T13 相同。

---

## 6. 不在收尾范围内

- 死成员和死 prop（第 4 节第 4 条，交 M2–M8）；`viewId as TProfileViews`（交 M4）。
- 评审没有点名、收尾的 Task 也没有改到的两类退化结构（`degen-all.mjs`，收尾结束时）：197 个没有插值的模板字面量（组件 127 个、`helpers/` 32 个、`constants` 18 个，例如 `` `/api/users/api-tokens/` ``）和 160 个只有一个子元素的片段。裁定是"在改到的行里收掉"；oxlint 没有能看住它们的规则（`jsx-curly-brace-presence` 只管 JSX 属性和子元素里的，那 80 处 T6 已收掉），一次性扫掉也会再长出来，所以按裁定由改到它们的 M 收掉。
- oxlint 的 696 个警告（3.3 的计划）。
- 封面照片和导览截图（第 9 节第 1 条，待裁定）；`handler_test.go`（第 9 节第 2 条，待裁定）。
- 包的版本号和 `license` 字段（交 M8）；`@makeplane/propel` 的对应源码（P5 交 M8）。

---

## 7. 风险

| 风险 | 应对 |
|---|---|
| 批量删除（T4 的 166 个文件、T11 的 500 个键、T12 的 133 张图）删掉了保留功能还在用的东西 | 类型检查、构建、knip 发现缺失的导入；`keyref.mjs orphaned`、`symref.mjs orphaned`、`deadorph.mjs`、`infile-orphans.mjs` 每个 Task 核对；图片按文件名查整个 `web/`；控制者的探测检查页面上原样显示的键和加载失败的图片，并重跑 P2–P5 的场景 |
| 类型检查器的"没人读"有误报（经框架、第三方库或展开读取） | 导出只算"别的文件读取"，经桶文件追到真正的读取方；成员和 prop 不在收尾删；T7 的 `parseHTML` 登记为已知的例外，由测试证明被读取 |
| `DOMParser` 取代 `sanitize-html` 后"是否为空"的判断或预览的文字变了，评论或草稿的提交行为改变 | 5 个单元测试覆盖不间断空格、`<br>`、允许的标签和实体；控制者的探测在评论和草稿弹窗里核对，与基线对照。已知的差别只在编辑器不会产生的元素上：`sanitize-html` 丢掉 `<script>`、`<style>`、`<textarea>`、`<option>` 里的文字，`textContent` 保留（编辑器的 schema 没有这些节点）。`DOMParser` 解析出的文档是惰性的，不执行脚本、不加载图片 |
| `window.open` 带 `noopener` 后调用方拿不到新窗口 | 12 处都不读返回值（逐处看过）；探测核对新页面打开、`window.opener` 为 `null` |
| 删掉 `window.close()` 后菜单不再关闭 | base-ui 在选中菜单项时自己关闭菜单；探测在溢出菜单里选一项，核对菜单关闭、标签页没有被关 |
| 补丁在 `pnpm install` 时没有生效或随升级失效 | 锁文件带补丁的哈希；T9 之后测试输出没有警告即证明生效；升级到新版本时 pnpm 会因补丁对不上而报错 |
| 守卫为了通过而放宽 | 新规则都有命中样本证明有效；例外只减不增；样本引用的代码由 `missquote.mjs` 核对存在 |
| 一个值有两个来源（主题标签）改成一个时，某处读的仍是被删的字段 | `key` 删除后类型检查找到了设置页的第二处读取（`value.key`），两处一起改；探测核对设置页和 Power K 的主题名 |

---

## 8. 移交事项

### 8.1 交给后续 M（T13 写进对应 M 的 `handoffs/M1-closeout.md`）

| M | 事项 | 关闭条件 |
|---|---|---|
| M2–M7 | 本 M 领域里的死成员和死 prop（第 4 节第 4 条的数目）；oxlint 的清理（3.3 的计划） | 本 M 合并时 `domains.mjs … --rows M<n>` 列出的每一行都已消失或写进本 M 的 review；review 写明改到的文件的警告数为 0、清掉的规则和上限的变化 |
| M4 | 另有 `viewId as TProfileViews` | `git grep -n "as TProfileViews" -- web` 没有输出 |
| M8 | 本 M 领域（Webhook）的 6 个死成员；共享部分的 286 个；oxlint 在发布前清零、警告改为错误、删除上限；13 个 `package.json` 的 `"license": "AGPL-3.0"` 改为 `AGPL-3.0-only` | 发布之前 `--rows shared` 为空或写进 review；oxlint 没有警告；`git grep -n '"license": "AGPL-3.0"' -- '*package.json'` 没有输出 |

P2–P5 写给 M2–M8 的 17 份交接各补一节"关闭条件"（T13）。

### 8.2 交给控制者

- 收尾 review（`reviews/closeout-review.md`）：复述 3.3、3.4，写进浏览器核对的脚本和输出；M1 设计 12 节收尾一行、11 节的勾、总体设计 9.4 和 M1 的状态（第 9 节第 3 条）。
- 第 9 节的裁定。

---

## 9. 待裁定

1. **封面照片和导览截图的来源（升级）。** 这是许可和来源的问题，收尾没有删也没有换任何一张（任务书的要求），查到的事实和建议如下。
   - **封面**：`app/assets/cover-images/image_1.jpg`–`image_29.jpg`，由 `helpers/cover-image.helper.ts` 导入，是项目封面的预设图片（Plane 的提交说明写着新建项目时随机选一张）。它们由 Plane 的提交 `36d4285`（2025-12-03，PR #8184 "[WEB-5493] feat: implement static cover image handling and selection"）加入；提交说明和仓库里都没有写来源、作者或许可。文件宽 1280 px，只带 ICC 色彩配置，没有 EXIF、XMP、IPTC（`jpgmeta.mjs`）。在那之前 Plane 用 16 张 Unsplash 照片的地址作预设封面（v1.1.0 的 `PROJECT_UNSPLASH_COVERS`）；把 29 张与这 16 张逐张做感知哈希比较（`provenance/phash.mjs`），没有一张相同：距离最近的三对（`image_5`、`image_6`、`image_21`，距离 6–9）打开看是完全不同的照片（摩天楼仰拍对红色日落，绿色多边形对红色日落，模糊的晚霞对黑底的岩石）。结论：来源和许可查不到；看起来是图库照片，Plane 的 AGPL 许可证管不到第三方照片的权利。
   - **导览截图**：`onboarding/cycles.webp`、`modules.webp`、`views.webp`（`onboarding/tour/root.tsx`）是 Plane 的界面截图，界面属于 Plane 的代码（AGPL），但里面有真人的头像照片和一位 Plane 联合创始人的名字，是第三方的肖像和个人信息。
   - **建议**：封面换成 Nerve 自己生成的 29 张抽象图（渐变或几何纹理，由脚本从 SVG 渲染，登记在同目录的 `SOURCES.md`，文件名不变，代码不改），不删功能；导览截图里的头像换成字母头像、人名换成中性的名字（P5 修复轮改 "Plane Design" 的同一种做法）。裁定之后作为收尾的追加 Task（在 T12 之后），我可以先做原型。另一种做法是删掉预设封面（只剩上传，封面上传归 M5），不推荐：它删掉了一个保留的功能。
2. **`handler_test.go` 的夹具名（跨模块）。** `server/internal/platform/webui/handler_test.go` 的夹具叫 `manifest.json`、`favicon/android-192.png`，注释说"对应 `web/apps/web/build/client` 的布局"，名字却是编的（P5 评审第 7 节）；`handler.go` 不依赖具体文件名，所以测试照常通过。改的是 Go 服务的测试，不在 web 之内。建议：收尾追加一个只改测试的提交，夹具改为构建真实产出的 `site.webmanifest.json`、`icons/icon-192x192.png`（注释因此成立）；或者交给下一个改 `webui` 的 M（M2 的认证会动静态资源的服务）。
3. **M1 的状态何时改为"已完成"。** 设计 9 节说收尾时改；任务书说 Codex 对整个 M1 的对抗评审在收尾合并之后进行。建议按设计在收尾 review 的提交里改（总体设计 9.4、M1 设计 12 节、11 节的勾），Codex 评审的发现按 M1 的修复轮处理并记在 M1 设计 12 节；如果控制者希望以 Codex 评审为 M1 的终点，就把状态留到那时改，11 节第 9 项在收尾 review 里写"待 Codex 评审"。
4. **死成员和死 prop 的分法（请确认第 4 节第 4 条）。** 按领域交 M2–M8、共享部分交 M8，每份交接带列出的命令和关闭条件。如果控制者要在 M1 内删，建议另开一个 Phase，按包分 Task（propel、ui、types、utils 等共享包先做），每处写明读取方的核对。
5. **删除的图片按 300×200 看是否足够（请确认第 4 节第 14 条）。**

**控制者的裁定**（详见 plan 的"控制者评审补充"）：第 1 条替换，封面为 T14、导览截图为 T15，保留的其余图片全部按能看清 20 px 细节的尺寸复看一遍；第 2 条收尾做，T16；第 3 条以收尾合并为 M1 的终点，已经启动的 Codex 评审的发现分诊进收尾，状态在收尾 review 的提交里改；第 4、5 条采纳。
