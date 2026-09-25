# M1/收尾 `closeout`：M1 收尾 Spec

- 上级设计：[M1 设计](../M1-design.md)（6、7.3–7.6、9 节"收尾"、11、12 节）、[总体设计](../../v0-design.md)（9.4）
- 前一 Phase：[P5 spec](P5-brand.md)、[P5 plan](../plans/P5-brand.md)、[P5 review](../reviews/P5-brand-review.md)（第 4 节裁定、第 5 节计划缺陷、第 7 节交给收尾的事项）
- 实施计划：[收尾 plan](../plans/closeout.md)
- 分支：`worktree-m1-closeout`，基线 `c493255`（P5 合并后的 `main`）

---

## 1. 目标

让 M1 设计第 11 节的完成标准成立，并清掉 P1–P5 留给收尾的全部事项。收尾不删产品功能，删的是从不渲染、没有读取方的代码和资源：

- **安全**：每个 `window.open` 都带 `noopener,noreferrer`，守卫看住；propel 的菜单项不再调用全局的 `window.close()`。
- **死代码**：从不渲染的应用栏；没有其他文件读取的工作区包导出（三轮，512 个导出：274 个连同声明删除，238 个只去掉导出）；各评审点名的死代码；基线的退化结构（`={"…"}` 等 80 处、别名、模板字面量、204 行悬空的导入分组注释）。
- **死资源**：没有代码引用的文案键（每种语言 1599 → 1077 个；T19 为活动迭代卡片加 2 个，终态 1079）、没有被导入的图片（133 张）和导入了却从不显示的图片（8 张）。
- **图片的来源和肖像**（第 9 节第 1 条的裁定）：来源和许可查不到的 29 张预设封面换成 Nerve 自己画的图；保留的图片里的真人照片头像换成字母头像，真实的人名换成中性的名字。
- **测试夹具**（第 9 节第 2 条的裁定）：`handler_test.go` 的夹具用构建真实产出的文件名。
- **守卫**：顶层 `phase` 为 `M1/closeout`，工具据此报出过期的例外，只剩跨 M 的 3 条；每条规则的每个分支都有命中样本，不命中样本都引用保留的代码。
- **工具链与依赖**：删掉 `sanitize-html`（开发环境的外部化警告的根源），给 `prosemirror-codemark` 打补丁（测试输出的 5 行噪声），格式检查覆盖两个配置包和根目录的工具链配置；编辑器标注块的属性按字符串读取。
- **报告与文档**：oxlint 按包、按规则的表和清零计划，构建体积与 P1 基线的对比，最后一次锁文件核对；前端改动清单；P1–P5 交给后续 M 的交接都写上关闭条件，收尾自己的交接（死成员和死 prop、oxlint 的清理）按领域交给 M2–M8。
- **修复轮**（3.16）：各 Task 的评审和控制者的浏览器核对发现的已知问题，T16 之后由控制者派发，10 个提交；整分支评审之后的第二个修复轮：zh-CN 里 5 个还是英文的设置分组标题、`services` 公开的 `ensureAPITrailingSlash`、一行错标的导入注释、写了两遍的根目录格式文件列表，4 个提交。
- **Codex 对抗评审的发现**（2.9，负责人批准的分诊）：粘贴编辑器自己的剪贴板类型时在惰性文档里解析，不再执行里面的脚本（T17）；`make build-web` 从空的产物目录开始（T18）；活动迭代卡片的进度用共享的口径（T19）；保留的截图里不再显示已删的功能（T20）。

死成员和死 prop（类型检查器找得到、knip 和 tsc 都不报的 1350 个）不在收尾删：它们几乎都在 M2 起逐个领域重写的代码里（第 4 节第 4 条），按领域交给后续 M。

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
| B2 | 没人用的包导出 | P3 评审第 7 节（347 个，`symref.mjs`）；P3 spec 7.4 | 类型检查器（`deadsym.mjs`）：426 个导出没有别的文件读取 | **T4**：三轮删完（512 个导出：274 个连同声明删除，238 个只去掉导出，3.8），剩 2 个（`api-client` 类型测试文件里的两个导出，文件本身就是检查） |
| B3 | `convertHexEmojiToDecimal`、`emojiCodeToUnicode`、`TrailingNode` 的导出 | P3、P4 评审第 7 节 | 没有调用方 | **T4**（属于 B2） |
| B4 | `useProjectIssueProperties` 的 5 个 fetcher 只有 `fetchCycles` 有调用方；P3 说的 `fetchStates` | P3、P4 评审第 7 节 | 同左 | **T5**：hook 删除，工作项表单直接调用迭代 store 的 `fetchAllCycles` |
| B5 | `IssueFormRoot` 挂载时的重置（可能是空操作） | P3 评审第 7 节 | 把表单重置为 `useForm` 刚用过的初始值，是空操作 | **T5** |
| B6 | `ui/empty-space.tsx` 没人传的属性和只有一个子元素的片段 | P5 评审第 7 节 | `EmptySpace` 的 `Icon`、`EmptySpaceItem` 的 `description`；2 个片段 | **T5** |
| B7 | knip 看不见的死成员和死 prop | P3 评审第 7 节（启发式脚本：182 个成员、205 个 prop） | `deadsym.mjs`：成员 1150、prop 638 | 收尾删掉自己的 Task 造成的和评审点名的（→ 成员 898、prop 452）；其余**按领域交 M2–M8**，共享部分交 M8（第 4 节第 4 条，第 8 节） |
| B8 | `={"…"}` 这类写法 | P4 评审第 7 节 | `react/jsx-curly-brace-presence`：80 处，58 个文件 | **T6**：oxlint 修掉，规则设为错误 |
| B9 | 只给导入常量起别名的 3 个 `const`；`no-projects.tsx` 没有插值的模板字面量 | P4 评审第 7 节 | 另有一处同类别名（打盹弹窗） | **T6** |
| B10 | `` t(`${getDescriptionPlaceholderI18n(…)}`) `` 外层多余的模板 | P3 评审第 7 节 | 1 处 | **T6** |
| B11 | `use-issues-actions.tsx` 的 `viewId as TProfileViews` | P4 评审第 7 节 | 1 处 | **交 M4**：根治要改工作项 store 共用的 hook 接口，M4 重写工作项 store 时一起做。关闭条件：`git grep -n "as TProfileViews" -- web` 没有输出 |
| B12 | `helpers/index.ts` 仍导出 `ensureAPITrailingSlash` | P1 评审第 6 节 (d)；Codex 对抗评审 Minor 1（2.9） | 仍然如此（这一格原来写"桶文件已不存在"，不对）：`services/src/helpers/index.ts` 是 `export * from "./url"`，包入口 `export * from "./helpers"`；应用只导入 `normalizeAPIRequestURL`（`core/services/api.service.ts`）。函数本身被 `normalizeAPIRequestURL` 调用，按名字导入它的只有 `url.test.ts`，所以 T4 的 `deadsym.mjs` 不报它（测试文件也是"别的文件"） | **整分支评审之后的修复轮**（第 9 节第 7 条，控制者确认；FW15 `df4cbf7`，3.16）：`helpers/index.ts` 只按名字转出 `normalizeAPIRequestURL`；`ensureAPITrailingSlash` 留在 `url.ts`，由 `normalizeAPIRequestURL` 和它的测试使用 |

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
| D2 | 29 张封面照片的来源和许可；导览截图里的真人头像和一位 Plane 联合创始人的名字 | P5 spec 7.3、P5 评审第 7 节 | 见第 9 节第 1 条。照片头像共 79 个：导览的 `cycles.webp`、`views.webp`、`issues.webp`（`modules.webp` 没有头像，也没有名字），"功能未开启"的 8 张，活动迭代的 2 张；人名有 "vamsi"（Plane 的联合创始人，导览）、"Aaryan"、"Bhavesh Raja"（视图的两张） | **T14**：29 张换成 Nerve 自己画的抽象图（SVG 源文件，应用导入 WebP）；**T15**：导览 `cycles.webp`、`views.webp`、`issues.webp` 和"功能未开启"8 张里的照片头像换成字母头像，人名换成中性的名字；`modules.webp` 不改（第 4 节第 19–25 条） |
| D3 | 没有显示的 8 张图 | 收尾原型（T15 复看时发现） | `assetpaths.mjs`（按路径）：142 张中 6 张没有被引用（`intake/intake-*.webp` 是 `disabled-feature/` 的逐字节副本，`search/views-*.webp`，`project/name-filter.svg`，`label.svg`）；活动迭代的 `cycle/active-*.webp` 被导入，却只传给一个被重命名为 `_activeCycleResolvedPath`、没人读的属性 | **T15**：删除，连同那个属性和只为它存在的主题 hook |
| D4 | 附件图标里的第三方标志 | 收尾原型（T15 复看时发现） | `attachment/figma-icon.png`（Figma 的标志）、`pdf-icon.png`（Adobe Acrobat 的标志）、`csv-icon.png`（Excel 的 X 标志）、`excel-icon.png`（Google Sheets 的图标） | **交 M5**（第 9 节第 6 条的裁定）。关闭条件：`app/assets/attachment/` 里没有第三方的标志 |

### 2.5 守卫

| # | 事项 | 来源 | 基线实测 | 去向 |
|---|---|---|---|---|
| E1 | 只剩明确跨 M 的例外；`until` 超出 M8 的逐条重新核对 | M1 设计 7.4、11 节；P2、P3 评审第 7 节（`tlds.ts` 的两条 `M9`）；P5 spec 7.3（`brand` 的 `M9`） | 5 条：`analytics` 到 M6、`project-invitations` 到 M3、`brand`（`pnpm-workspace.yaml`，3 处）到 M9、`tlds.ts` 的 `analytics`、`wiki` 到 M9 | **T1**：顶层 `phase` 改为 `M1/closeout`，工具据此判断到期（M1 内到期的例外都会报错）；**T4** 删掉 `tlds.ts`（它唯一的读取方是没人调用的地址解析）和它的 2 条例外；`brand` 的例外重新核对：3 行注释记录配置来自 Plane 的哪个提交，理由成立，保留到 M9。结束时 3 条 |
| E2 | 规则作用范围的边界（`storybook` 的 `files` 不含 `Makefile`、`.github/**`；`deploy-files` 锚定 `^web/`；`storybook-files` 的一个分支没有样本） | P1 评审第 6 节 (a) | `alts.mjs`：42 个分支没有命中样本 | **T1**：每个分支（含 `files` 选择器）都有命中样本；**不扩大范围**：这些词在 `Makefile`、`.github/`、`deploy/`、`e2e/`、`tools/` 里都没有命中 |
| E3 | 从未存在过的样本路径（约 20 条规则写着 `web/apps/web/app/assets/logo.svg`） | P5 评审第 7 节 | `missquote.mjs`：50 个不命中样本引用的代码或路径不存在（26 个是 `logo.svg`） | **T1**：替换 60 个、删除 5 个（本分支：`aea47e4` 替换 58、删除 7，跟进提交 `483ca9d` 又换掉 3 个、恢复 2 个），结束时 0；之后删掉或改名被引用代码的 Task 同一提交改样本（T11 一处） |
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
| G1 | 导入分组的错标：`app/root.tsx` 悬空的 `// types`，4 对相邻同名分组 | P5 评审第 7 节 | `labels-all.mjs`：悬空 208 行、重复 4 行 | **T6**：同类的全部删掉，不移动导入（另 2 行悬空的随 T4 删掉的文件消失）。本分支（`20ac5b6`）删悬空 204 行、重复 4 行，另删 3 行裸的 `//` 和一行 `//hooks`，共 212 行（3.9） |
| G2 | `server/internal/platform/webui/handler_test.go` 编造的夹具文件名 | P5 评审第 7 节 | `manifest.json`、`favicon/android-192.png`；`handler.go` 不依赖具体文件名 | **T16**（第 9 节第 2 条的裁定）：夹具改为 `site.webmanifest.json`、`icons/icon-192x192.png`，回退用例的目录改为 `/icons`；只改测试 |
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
| H5 | 前端改动清单（第二、四节和第 3 节带来的增减）同步 | M1 设计 9、11 节 | **T13**（1.6 节 13 行、第二节两行）；T14、T15 各在 1.6 节末尾加一行；修复轮加 7 行（`4be6cba`，3.16）；T17–T20 各一行，由控制者在 T20 之后的文档提交里加（3.17）；整分支评审之后的修复轮 FW14–FW17 各一行，在它的文档提交里加（3.16）；第四节在 P5 已全部"已完成" |
| H6 | 本 M 的交接全部关闭；交给后续 M 的交接都有接收的 M 和关闭条件 | M1 设计 9、11 节 | `M0-P5-frontend-trim-notes`、`M0-P6-knip-notes` 在基线已是 `closed`；P2–P5 写给 M2–M8 的 17 份交接各加一节"关闭条件"，收尾自己给 M2–M8 各写一份（**T13**） |
| H7 | M1 设计 12 节收尾一行、11 节打勾、总体设计 9.4 和 M1 的状态改为"已完成" | M1 设计 9、11 节 | 控制者在收尾 review 的提交里改（与 P5 相同）；时机见第 9 节第 3 条 |

### 2.9 Codex 对抗评审的分诊

Codex 对整个 M1 的对抗评审（[报告](../reviews/M1-codex-adversarial-review.md)，`main` 上的提交 `d2bd4fc`，本分支合并时带进来；评审对象 `c493255`）第 4 节有 7 条新发现。按第 9 节第 3 条的裁定分诊进收尾，负责人 2026-09-25 批准按建议处理（plan 的"Codex 对抗评审之后的裁定"）。"核对"一栏是架构师在本分支修复轮之后（`9a98148`）和基线上的复核。报告第 6 节的已知事项和第 7 节的前瞻风险是已有交接的复述，不另分诊；第 7 节里 jsDelivr 的请求随 Important 3 交 M8。每条的处理结果由控制者写进报告的"处理结果"一节（收尾 review 的提交）。

| # | 发现 | 核对 | 去向 |
|---|---|---|---|
| Critical 1 | 编辑器的粘贴处理把剪贴板里自定义类型 `text/nerve-editor-html` 的 HTML 在清洗之前赋给一个离屏 `div` 的 `innerHTML`，带 `onerror` 的 `<img>` 会执行；任何网页都能在自己的复制事件里写这个类型 | 属实：`helpers/paste-asset.ts` 的 `processAssetDuplication`（`tempDiv.innerHTML = htmlContent`，处理之后再赋一次）；`div` 不挂到页面上，但它属于活动文档，图片照样加载。`t17/probe/copy-probe.mjs` 的 C3 在 `c493255` 的构建上 `window.__xss` 变为 `true` | **T17**：在 `DOMParser` 的惰性文档里解析（3.17，第 4 节第 27 条）。复制资源的接口要核对请求者能否读取源资源，**交 M5**（8.1） |
| Important 1 | 命中 Turbo 缓存的 `make build-web` 不删产物目录里原有的文件，`make build` 把陈旧的资源嵌进 Go 程序 | 属实：`t18/prove.sh` 在旧的 `Makefile` 上，事先放进 `build/client/assets/` 的文件在热构建（11 个任务都命中缓存）之后还在；冷构建时 Vite 自己清空了目录 | **T18**：`build-web` 在 Turbo 之前 `rm -rf web/apps/web/build/client`（第 4 节第 28 条） |
| Important 2 | 活动迭代卡片的进度是 (完成 + 取消) / (总数 − 取消)，文字写 "x/y closed"；共享的 `calculateCycleProgress` 是 完成 / (总数 − 取消)。取消 2、完成 8、共 10 时卡片写 "10/8 closed" | 属实：`cycles/active-cycle/progress.tsx`；卡片的公式在 Plane 就有，M1 设计 3.4 改写了共享的口径 | **T19**：卡片经 `calculateCycleProgress` 计算，文字写完成 / (总数 − 取消)，两句文案进 en、zh-CN（第 4 节第 29 条）。迭代侧边栏和模块的进度另有口径，**交 M6**（8.1） |
| Important 3 | 公开部署的 AGPL 验收没有覆盖第 5(a) 条（修改说明和日期）、5(d) 条（交互界面的法律声明）、13 条（网络用户取得对应源码）；另外表情数据（`emojibase-data@latest`）和标注块的默认表情图片从 jsDelivr 取，会向第三方发请求 | 属实：M8 的交接写了 propel 的对应源码、版本号和 OG 地址，没有这三条的验收；jsDelivr 的两类请求在 M1 之前就有。条款怎样适用（例如 5(d) 对原版界面的例外）是法律解读 | **交 M8**：控制者在收尾 review 的提交里给 M8 的收尾交接加关闭条件：第 5(a)、5(d)、13 条逐项验收，页面上的源码入口指向运行的提交和完整的对应源码；jsDelivr 的请求改为自托管固定版本，或明确告知并接受。时点是"任何对外的网络部署之前"，最迟 M8 发布；条款的解读由负责人在 M8 定（8.1） |
| Important 4 | 保留页面的图片还显示已删的功能：收集箱关闭时的两张图印着 "Estimate 2 Months"，导览的 `cycles.webp` 有数据分析按钮 | 属实；T15 复看截图时已记下同类的另外 7 处（T20 的原型又按原尺寸看了一遍，没有更多）：甘特图、时间线布局图标（导览 `cycles.webp`、`views.webp`，"功能未开启"的模块、迭代各两张），导览 `issues.webp` 的 "Public" 发布标签 | **T20**：9 张图里涂掉 10 个控件（3.17，第 4 节第 30 条） |
| Minor 1 | P1 交给 P3 的孤儿导出 `ensureAPITrailingSlash` 没有收口，两个桶文件仍公开它 | 属实，本 spec 的 B12 原来写错了（2.2）：`services/src/helpers/index.ts` 和包入口都是 `export *`，应用只用 `normalizeAPIRequestURL`；T4 没有删它，因为 `url.test.ts` 按名字导入它 | **整分支评审之后的修复轮**（第 9 节第 7 条，控制者确认）：`helpers/index.ts` 只按名字转出 `normalizeAPIRequestURL`（B12；FW15 `df4cbf7`） |
| Minor 2 | M1 设计 3.1 写"保留图片和附件的节点"，社区版的编辑器却没有附件节点（M5 的交接写明了） | 属实 | 控制者在收尾 review 的提交里把 M1 设计 3.1 改为"图片节点保留；附件由工作项的附件区承担，编辑器里是否支持由 M5 决定" |

### 2.10 浏览器核对顺带发现的两件事

控制者第 3 轮浏览器核对（`9a98148` 对基线）看到的，基线上相同。架构师查了来源：

| # | 事项 | 核对 | 去向 |
|---|---|---|---|
| I1 | 个人设置（zh-CN）里主题的下拉框画在页面左上角 | 继承自 Plane：主题选择器是 ui 的 `CustomSelect`（`web/packages/ui/src/dropdowns/custom-select.tsx`，react-popper 定位，`createPortal` 到 `document.body`）。整个 M1（导入的 `48e1a63` 到 `9a98148`）里它只改过包名，去掉改名的行之后与 Plane 的 `02c19e1` 相同；`theme-switch.tsx` 只改了标签的键 | **交 M2**（个人设置）：控制者在收尾 review 的提交里写进 M2 的收尾交接，关闭条件是本 M 的浏览器核对写明个人设置的下拉框在它的按钮旁展开（8.1） |
| I2 | zh-CN 界面上有英文 | 两种来源，都是 Plane 原有的：(1) 没有经过 `t()` 的英文字面量，`t20/investb-hardcoded.mjs` 在 `web/apps/web` 的 978 个 `.tsx` 里粗略找到约 457 处（估计的量级：会算进少数不是界面的字符串，漏掉模板字面量），最多的是工作项 83、项目 49、模块 36、导览 36、收集箱 35、迭代 24，例如 "First day of the week"、"Search commands..."；(2) zh-CN 的值与英文相同的键 11 个（`t20/investb-untranslated.mjs`），其中 5 个是没翻译的词：`common` 的 `developer`、`your_profile`、`work_structure`、`execution`、`administration`，其余是不用翻译的 URL、ID、Webhooks、`name@company.com`。两种语言的键一一对应，没有缺键 | (1) **交 M8**：控制者在收尾 review 的提交里写进 M8 的收尾交接："每个 M 把改到的界面文字接入 `t()`；M8 发布前中文覆盖"（8.1）；(2) 5 个值在整分支评审之后的修复轮里翻译（FW14 `e171d64`，3.16） |

**规模**：20 个 Task 和两个修复轮（第二个在整分支评审之后，只有 4 个小提交，3.16）。T1–T16 的原型改 875 个文件（+6091 / −12814 行；T1–T13 是 763 个文件，+1377 / −12685），修复轮 27 个文件（+212 / −157），T17–T20 的原型 19 个文件（其中 9 张 WebP；+92 / −65）；本分支的实际数字见 3.1。其中大部分是机械的：T4 的三轮删除、T6 的 oxlint 修复和注释、T11 的键、T12 的图片、T14 的封面、T15 的头像和人名、T20 的涂改都由脚本完成，脚本的输出逐项可核对（T14 的 +4658 行几乎都是 29 张 SVG 源文件）。要逐处判断的是 T2、T3、T5、T6 的手写部分、T7、T8、T17、T19。死成员和死 prop 若在收尾删，要在 348 个文件里逐个判断 1350 处（很多是 Plane 接口类型的字段，M2 起按 OpenAPI 生成的类型整体替换），不适合放在一个 Phase 里，交给重写这些代码的 M。

---

## 3. 交付物

### 3.1 文件总览

20 个 Task，每个一个提交（T1、T18、T19 各另有一个跟进提交）；T16 之后另有修复轮的 10 个提交，整分支评审之后又有一个修复轮（4 个代码提交和一个文档提交），都在 3.16。原型（在 `$COTMP/proto`）实测：T1–T13（`c493255..328843f`）`763 files changed, 1377 insertions(+), 12685 deletions(-)`；T1–T16（`c493255..3c64669`）`875 files changed, 6091 insertions(+), 12814 deletions(-)`；T17–T20（分支 `proto-codex`，`9a98148..c9b24e1`）`19 files changed, 92 insertions(+), 65 deletions(-)`（9 张 WebP 不计行数）。

本分支的实际提交是准绳，与下表的原型不同之处：T1 另有跟进提交 `483ca9d`（`tools/keywords.json`，+6 / −5，3.6）；T4 `7f189da` 是 325 个文件，+166 / −9183；T6 `20ac5b6` 是 +101 / −343（3.9）；T9 `98ecc86` 是 +125 / −3；T11 `f9fb015` 的行数相同，`i18n-applications` 的样本不同（3.6）。本分支 T1–T13 的净改动（`c493255..300eff0`，不计本 spec 和 plan）是 `764 files changed, 1379 insertions(+), 12684 deletions(-)`。T15 `e349924` 按 plan 比原型多写 M5 的交接（8.1），是 24 个文件，+64 / −96，其中 M5 的交接 +13 / −2；在 `300eff0` 上重放 plan 的步骤得到的是 +63 / −95（交接 +12 / −1），差的一行是提交另把交接的标题改为写上这两项。T17 `7836c7a`、T18 `fc3069a` 与原型相同；T18 另有跟进提交 `e070065`（`Makefile`，+1 / −1：`make build` 复制时也读 `$(WEB_CLIENT)`）。T19 `c84f26d` 的行数相同，比原型多去掉取消说明外面一层只包着文字的 `<span>`；另有跟进提交 `f9d1ab2`（2 个文件，+24 / −26：分组行是带 key 的列表项，web 的 oxlint 上限 566 → 565）。T20 `ff555bf` 是 +22 / −0：`intake-light.webp` 那一框的填法不同，`disabled-feature/SOURCES.md` 的说明随之多两行（3.17）。其余 Task 与原型相同。本分支到修复轮结束（`c493255..9a98148`，不计本 spec 和 plan）是 `883 files changed, 6280 insertions(+), 12934 deletions(-)`；在它上面加 T17–T20 的原型（`c493255..c9b24e1`）是 `888 files changed, 6367 insertions(+), 12994 deletions(-)`；本分支到 T20（`c493255..ff555bf`）是 `889 files changed, 6394 insertions(+), 13021 deletions(-)`。

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
| 13 | 前端改动清单、README、交接 | — | 7 | 19 | +413 / −2 | `328843f` |
| 14 | 预设封面换成 Nerve 自己画的图 | 29 | 59 | 2 | +4658 / −30 | `131e1de` |
| 15 | 保留的图片里没有真人；没有显示的图片删除 | 8 | 2 | 13 | +51 / −94 | `f997e8a` |
| 16 | `handler_test.go` 的夹具用构建真实产出的文件名 | — | — | 1 | +5 / −5 | `3c64669` |
| 17 | 粘贴编辑器自己的剪贴板类型时，在惰性文档里解析 | — | — | 4 | +56 / −53 | `bad3483` |
| 18 | `make build-web` 从空的产物目录开始 | — | — | 1 | +2 / −0 | `b85f861` |
| 19 | 活动迭代卡片的进度用共享的口径 | — | — | 3 | +14 / −12 | `a994925` |
| 20 | 保留的截图里不再显示已删的功能 | — | — | 11 | +20 / −0（9 张 WebP） | `c9b24e1` |

### 3.2 原型验证：结论与证据

在 `$COTMP/proto`（从本分支的基线克隆）中逐个 Task 做过一遍，每个 Task 结束时 `pnpm exec turbo run check:types`、`make lint-web`、`make test-web`、`make build-web`、`make knip` 都通过。之后在一个从基线新建的克隆（`$COTMP/replay`）上按 plan 的步骤重放了全部 13 个 Task（`replay.mjs`），每个 Task 得到的树与原型提交完全相同；T4–T13 的五个门禁和测试输出的核对又在重放的提交上重跑了一遍（`gates-range.mjs`，T1–T3 的原型提交没有变），全部通过。T14–T16 在 `328843f` 的干净克隆上按 plan 的步骤重放，三棵树与原型提交相同（T14、T15 的 WebP 逐字节相同）；又在本分支 T13（`300eff0`）的克隆上重放，每个 Task 改到的图片、代码和 `server/` 路径与原型提交相同，五个门禁通过，构建体积与下表相同。T15 写 M5 交接的一步（`t15/handoff.mjs`）是原型之后按裁定加的，不在原型提交 `f997e8a` 里；它在 `300eff0` 的重放上跑过（M5 交接 +12 / −1）；本分支的 T15 `e349924` 另把交接的标题改为写上这两项（+13 / −2，3.1）。

T17–T20 在 `$COTMP/proto` 的 `proto-codex` 分支上做，基点是本分支修复轮的最后一个提交 `9a98148`，每个提交结束时五个门禁通过；又在 `9a98148` 的干净检出上按 plan 当时的步骤重放，四棵树与原型提交相同（T20 的 WebP 逐字节相同）。每个提交的孤儿核对都是 0（`infile-orphans.mjs` 没有行），`deadsym.mjs` 都是 `prop 452, member 898, export 2`，`deadorph.mjs` 对 `9a98148` 为 0。本分支的 T17–T20（`7836c7a`–`ff555bf`）每个 Task 的提交都通过五个门禁（T18、T19 的跟进提交之后的树由下一个 Task 的门禁覆盖），`deadsym.mjs` 都是 2 / 898 / 452，`deadorph.mjs` 从修复轮到 T20 为 0；与原型不同的三处见 3.1。

下表的"收尾结束"是本分支 T20（`ff555bf`）的实测；括号里是更早的值或原型的值：T16 的原型 `3c64669`、本分支修复轮之后的 `9a98148`、T20 的原型 `c9b24e1`，或 T13（`328843f`）。T17–T20 只改变 vitest、lint 上限合计（T19 的跟进）、键、构建体积四行，新加粘贴、构建目录、活动迭代卡片三行；修复轮改变 vitest、lint 上限合计和 `deadsym.mjs` 三行。

| 项 | 基线（`c493255`） | 收尾结束（`ff555bf`） |
|---|---|---|
| `make lint-web` | `keywords: 48 rules, 5 exceptions, no hits.` + 52 个 turbo 任务 | `keywords: 51 rules, 3 exceptions, no hits.` + 54 |
| `check:types` / `make test-web` / `make build-web` | 23 / 16 / 11 | 不变 |
| vitest | 77 个测试（web 5、constants 11、editor 16、i18n 10、services 11、utils 24），stderr 有 editor 的 5 行 `prosemirror-codemark` | 101 个（editor 35、utils 29），没有 stderr（`3c64669`：86 个，editor 20；`9a98148`：100 个，editor 34） |
| `make knip` | 零 | 零（每个 Task） |
| lint 上限合计 | 711 | 694（web 566→565、hooks 4→3、propel 21→16、ui 28→25、utils 25→18）；`lintdiff.sh` 没有新增的警告（`3c64669`：696，utils 19；`9a98148`、`c9b24e1`：695，web 566） |
| 守卫：没有命中样本的分支（`alts.mjs`） / 引用不存在代码的不命中样本（`missquote.mjs` / `missreal.mjs`） | 42 / 50 / — | 0 / 0 / 0（本分支；原型 T11 起 `missreal` 为 1，3.6） |
| 例外 | 5（`analytics` M6、`project-invitations` M3、`brand` M9、`tlds.ts` 的 `analytics`、`wiki` M9） | 3（前三条） |
| 不带 `noopener` 的 `window.open` | 16 处中 12 处 | 0（规则 `window-open`） |
| 包导出 / 死成员 / 死 prop（`deadsym.mjs`） | 426 / 1150 / 638 | 2 / 898 / 452（`3c64669`：2 / 894 / 452；多出的四个见结论 3） |
| 每种语言的键 / 无引用的键（`keyref.mjs`） / 只被无关字面量命中的键（`keyfalse.mjs` 的 `?`） | 1599 / 482 / 34 | 1079 / 0 / 12（都到得了 `t()`；T11–T18 是 1077，T19 加 2 个） |
| 图片 / 没有引用的（`assets.mjs`） | 246 / 133 | 134 / 0（T13：113 / 0；T14 加 29 张 SVG 源文件，T15 删 8 张） |
| 按路径没有被引用的图片（`t15/assetpaths.mjs`） | — | 0（T14 之后 6 张，T15 删除） |
| 导入分组注释：悬空 / 重复（`labels-all.mjs`） | 208 / 4 | 0 / 0 |
| `react/jsx-curly-brace-presence` | 80 处，58 个文件（规则关闭） | 0（错误级别） |
| 只给导入常量起别名的 `const` | 4 | 0 |
| 开发服务器的外部化警告（`devwarn.mjs`，打开 `/`，页面加载两次） | 浏览器 44 条、服务端 44 行（每次加载各 22） | 0 / 0 |
| 锁文件 | — | 删 17 个包，不新增、不升级（3.5） |
| 构建体积合计 | 534 个文件，17,306,349 字节 | 531 个，14,763,098 字节（`c9b24e1`：14,763,234；`3c64669`：14,799,463；T13：533 个，17,071,770；整分支评审之后的修复轮 FW14 起 14,763,091；3.4） |
| Go：`make test` / `make lint-go` | 通过 / `0 issues.` | 不变（T16 只改测试数据） |
| 粘贴带 `onerror` 的 `text/nerve-editor-html`（`t17/probe/copy-probe.mjs` 的 C3） | 脚本执行（`window.__xss` 为 `true`） | 不执行；同编辑器之间复制图片仍复制资源（C1） |
| `make build-web` 之前放进产物目录的文件（`t18/prove.sh`） | 热构建（命中缓存）之后还在，`make build` 把它嵌进 Go 程序 | 冷、热构建之后都不在；两次的 531 个文件逐字节相同 |
| 活动迭代卡片（共 10、完成 3、取消 2） | 进度 62.5%，"5/8 Work items closed"；分组行触发 React 的 key 警告（开发模式） | 38%（`calculateCycleProgress`，`progress.test.ts` 有这一例），"3/8 work items completed"、zh-CN "3/8 个工作项已完成"（T19）；分组行的每一项有自己的 key（T19 的跟进）。控制者第 4 轮浏览器核对的 `probe/active-cycle.mjs`（plan 浏览器核对第 15 项，3.14）在 `ff555bf` 的构建上 23 项全部通过，基线上 16 通过、7 失败 |

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
| T14 | 2 / 894 / 452 | 696 | 1077 | 0 / 142 | 51 / 3 | 0 / 0 |
| T15、T16 | 2 / 894 / 452 | 696 | 1077 | 0 / 134 | 51 / 3 | 0 / 0 |
| 修复轮 | 2 / 898 / 452 | 695 | 1077 | 0 / 134 | 51 / 3 | 0 / 0 |
| T17、T18 | 2 / 898 / 452 | 695 | 1077 | 0 / 134 | 51 / 3 | 0 / 0 |
| T19 | 2 / 898 / 452 | 695 | 1079 | 0 / 134 | 51 / 3 | 0 / 0 |
| T19 的跟进、T20 | 2 / 898 / 452 | 694 | 1079 | 0 / 134 | 51 / 3 | 0 / 0 |
| 整分支评审之后的修复轮 | 2 / 898 / 452 | 694 | 1079 | 0 / 134 | 51 / 3 | 0 / 0 |

每个 Task 的孤儿核对（基点是上一个 Task 的提交）：`symref.mjs orphaned`、`dangling.mjs`、`keyref.mjs orphaned` 都是 0，`headers.sh` 没有输出；`deadorph.mjs`（新出现的死成员、死 prop、死导出）除 T7、修复轮的 FW4 和 `9a98148` 外都是 0（结论 3）；`infile-orphans.mjs` 在 T3、T4、T5 列出 12、166、4 行，都以 `defined now: 0)` 结尾（被删掉的符号），其余 Task 为 0。T4 的 166 行是本分支 `7f189da` 上的实测（控制者逐个提交重跑）；重放得到 165 行，少的一行是 `THEMES`：它在基线上的两处使用都是注释（`types/src/users.ts`、`utils/src/theme.ts`），本分支的 T4 把两处注释改为写 `THEME_OPTIONS`，重放保留了它们。T17–T20 的数字是本分支提交上的实测（T18 的跟进 `e070065` 只改 `Makefile`，数字与 T18 相同；T19 的跟进 `f9d1ab2` 把 web 的上限 566 → 565，另占一行）。

结论：

1. **原型的步骤可以照做。** 重放发现 T3、T4、T6、T8 要在脚本之后跑一次 oxfmt，plan 已写进步骤；重放之后又发现 T4 让 propel `EmptyState` 的 `asset` 属性成了孤儿（`deadorph.mjs` 报 1），改在 T4 删掉，T11 补了主题选项（C3），重放和门禁都重跑过。
2. **包导出要用类型检查器找。** knip 把每个包的入口都当作已使用；`symref.mjs` 按名字数引用，同名的局部变量、属性也算。`deadsym.mjs` 用 TypeScript 的程序模型：一个导出只有在它自己以外的文件读取它时才算使用（经过任何桶文件；重新导出不算读取）。删一轮会让上一层的导出变成没人用，所以要删到某一轮为零为止（没有别的文件读取的导出，不计 `api-client` 类型测试文件的 2 个：第 1 轮 424 个、第 2 轮 26 个、第 3 轮 62 个，共 512 个；第 4 轮 0）。
3. **T7、修复轮 FW4 和 `9a98148` 的 `deadorph` 报出的是已知的例外。** 新加的 `parseHTML` 写在属性对象的类型里，由 TipTap 在解析 HTML 时读取，脚本只看项目自己的代码；T7 的测试证明它被读取（去掉它，第一个用例以 `128161` 失败）。修复轮的 FW4 同样报 1，是同一类：`TLogoProps.emoji.url`（`types/src/common.ts`）。它唯一有类型的读取方是标注块 `getStoredLogo` 里的 `as TLogoProps`；FW4 改为在运行时检查本地存储里的 JSON（类型是 `unknown`），仍然读取 `url`，脚本只看有类型的读取。它喂给的 `data-emoji-url` 只有标注块的 Markdown 序列化在用（交 M4，8.1）。FW10（`6048ad3`）删掉 `isCommentEmpty` 走不到的 JSON 分支之后，`JSONContent`（`types/src/editor/editor-content.ts`）的 `type`、`content`、`text` 没有了读取方，FW10 把它们一起删掉（`deadorph.mjs` 为 0）；控制者复核之后的 `9a98148` 恢复了它们，`deadsym.mjs` 于是多报 3 个成员（895 → 898），`deadorph.mjs` 按文件和名字比较，列出 `content`、`text` 两行（`type` 与 T16 时已报的 `marks` 里的 `type` 同名）。它们保留：`JSONContent` 描述的是 TipTap 的 JSON 格式，只写一半不如不写；它唯一的使用者 `TIssueComment.comment_json` 本身没有读写方，属于 M4 的领域，两者由 M4 一起处理（M4 的交接）。
4. **外部化警告的根源在依赖，不在 Vite 配置。** `devwarn.mjs` 打开 `/`，页面加载两次，浏览器 44 条、服务端 44 行（每次加载各 22 条），都来自 postcss 读取 `path`、`fs`、`url`、`source-map-js`；web 里只有 `sanitize-html` 把 postcss 带进浏览器端，它的全部用处是 `@nerve/utils` 的三个 HTML 工具。应用是纯客户端的构建（`ssr: false`），浏览器自带的 `DOMParser` 就够用；删掉之后警告为 0，还修掉了通知预览显示 `&amp;` 的问题（3.10）。
5. **测试输出的噪声也是依赖的问题。** `prosemirror-codemark` 0.4.2 是最新版本，发布的每个构建文件末尾都指向一个列出未发布源文件的 source map；补丁只删这 12 行注释（`strip.mjs`），与仓库里已有的 `react-color` 补丁同一种做法。
6. **没有新的共享可变状态。** 全 Phase 新增的顶层声明都是函数、只读数据或测试辅助；唯一的模块级可变值是 T7 测试文件里等待 `afterEach` 销毁的编辑器列表。新增的行里没有 `={"`、`${"`，也没有不带插值的模板字面量。
7. **T1–T13 的构建产物几乎不变。** 删掉的图片本来就不在构建里（没有被导入）；JS 少 231,564 字节（propel 和 ui 的死组件、`sanitize-html`），CSS 少 3,015 字节（3.4）。T14、T15 让"其他"少 2,272,137 字节：29 张封面从 JPEG 换成 WebP，活动迭代的两张图删除（3.4、3.15）。修复轮和 T17–T20 只差几百字节的 JS；T20 让"其他"再少 35,994 字节（3.4）。
8. **粘贴的缺口只在 ProseMirror 之前的一步（T17）。** ProseMirror 的 `pasteHTML` 在它自己新建的文档（`document.implementation.createHTMLDocument`）里按编辑器的 schema 解析，只留 schema 声明的节点和属性，本来就不执行脚本；执行发生在更早的 `processAssetDuplication`：它在活动文档里建 `div` 解析同一段 HTML，好标记要复制的图片。`div` 没有挂到页面上，但它的元素属于页面，图片照样加载，`onerror` 照样执行。`DOMParser` 的文档没有浏览上下文，里面什么都不加载、不执行。编辑器的新测试在旧代码上失败（带 `onerror` 的标记两次写进活动文档的元素），浏览器里的 C3 在基线上执行、在 T17 之后不执行。
9. **Turbo 的缓存恢复只写不删（T18）。** 冷构建时 Vite 在写之前清空输出目录，所以 Codex 的实验要命中缓存才出现：Turbo 把记录下来的输出文件写回原处，不管目录里还有什么。清空这一步要在 Turbo 之前，由 `make build-web` 做；清空之后，冷、热两次构建的 531 个文件逐字节相同。持续集成里两种情况都有：web 任务和 e2e 任务的 Build 一步从干净的检出开始，`rm -rf` 什么都不删；e2e 任务接着跑 `make e2e`，它依赖 `build`，再构建一次，这时 `rm -rf` 删掉第一次构建的 `client`，Turbo 从本任务自己的缓存恢复它，就是 `prove.sh` 证明的热构建。跟进提交 `e070065` 让 `make build` 复制的也是 `$(WEB_CLIENT)`，清空的目录和嵌入的目录只有一个来源。

### 3.3 oxlint：按包、按规则（M1 设计 7.3）

`linttable.mjs` 与 `tools/lint-cap.mjs` 同样的跑法（在每个包目录里 `oxlint --format=json .`），每个包的合计等于上限。基线 711 个：web 566、editor 65、ui 28、utils 25、propel 21、hooks 4、constants 1、i18n 1。收尾结束时 694 个，39 条规则（修复轮删掉 utils 的 1 个 `no-useless-length-check`，它在删掉的 JSON 分支里；T19 的跟进 `f9d1ab2` 删掉 web 的 1 个 `no-array-index-key`：活动迭代卡片的分组行改用分组名作 key；T17–T20 的其余提交不增不减）：

| 规则 | web | constants | editor | hooks | i18n | propel | ui | utils | 合计 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `eslint(no-shadow)` | 136 | 1 | 22 |  |  | 8 | 3 | 1 | 171 |
| `eslint-plugin-jsx-a11y(no-static-element-interactions)` | 65 |  | 9 |  |  | 1 | 4 |  | 79 |
| `eslint-plugin-promise(always-return)` | 75 |  | 1 |  |  |  |  |  | 76 |
| `eslint-plugin-jsx-a11y(click-events-have-key-events)` | 61 |  | 5 |  |  | 1 | 4 |  | 71 |
| `eslint(no-unneeded-ternary)` | 55 |  |  |  |  |  | 1 |  | 56 |
| `eslint-plugin-react-hooks(exhaustive-deps)` | 35 |  | 1 |  |  |  | 7 |  | 43 |
| `eslint-plugin-react(no-array-index-key)` | 19 |  |  |  |  | 3 | 1 |  | 23 |
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
| `eslint-plugin-unicorn(prefer-array-find)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint(no-new)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint(no-unsafe-optional-chaining)` | 1 |  |  |  |  |  |  |  | 1 |
| **合计** | 565 | 1 | 65 | 3 | 1 | 16 | 25 | 18 | **694** |

`api-client`、`services`、`shared-state`、`types`、`e2e` 没有警告。

**清零计划**（M1 设计 7.3，写进 M2–M8 各自的 `handoffs/M1-closeout.md`，由 T13 生成）：
1. **谁改谁清**：每个 M 改到的文件，在该 M 结束时没有 oxlint 警告；
2. **按规则清一类**：每个 M 另外按规则集中清掉至少一类，优先能机械修复的 `no-shadow`、`promise/always-return`、`no-unneeded-ternary`，写进该 M 的 review；
3. **M8 发布之前清零**，之后 `.oxlintrc.json` 把警告改为错误、删掉各包的上限（总体设计 7.6 的原始要求）。
每一步上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。

### 3.4 构建体积（M1 设计 7.6）

P1 的方法（P1 spec 2.14）：先 `rm -rf web/apps/web/build`，再 `make build-web`，然后 `node web-size.mjs`（`size.sh`）。

| 项 | P1 之前（`0f2b3e0`） | 收尾之前（`c493255`） | T13 之后（`328843f`） | T16 之后（`3c64669`） | 收尾之后（`73f8c36`） | 对 P1 之前 |
|---|---|---|---|---|---|---|
| JS | 1055 个，15,151,327 字节 | 397 个，6,818,887 | 396 个，6,587,323 | 396 个，6,587,153 | 396 个，6,586,775 | −56.5% |
| CSS | 3 个，326,068 | 3 个，296,816 | 3 个，293,801 | 3 个，293,801 | 3 个，293,801 | −9.9% |
| 字体 | 34 个，6,849,620 | 25 个，3,755,608 | 25 个，3,755,608 | 25 个，3,755,608 | 25 个，3,755,608 | −45.2% |
| 其他 | 147 个，9,905,117 | 109 个，6,435,038 | 109 个，6,435,038 | 107 个，4,162,901 | 107 个，4,126,907 | −58.3% |
| 最大的 chunk | `toolbar-*.js` 1,821,937 | `use-parse-editor-content-*.js` 1,379,320 | 同名 1,378,717 | 同名 1,378,717 | 同名 1,378,580 | |
| 语言 chunk | 588 | 34 | 34 | 34 | 34 | |
| **合计** | **1239 个，32,232,132 字节（30.7 MiB）** | **534 个，17,306,349（16.5 MiB）** | **533 个，17,071,770（16.3 MiB）** | **531 个，14,799,463（14.1 MiB）** | **531 个，14,763,091（14.1 MiB）** | **−54.2%** |

文件数只作记录，不作为门禁。本分支 T13（`300eff0`）的构建与 `328843f` 的数字相同；T14–T16 在它上面重放，得到的也是 `3c64669` 一列。T14 把 29 张封面从 3,512,619 字节的 JPEG 换成 1,379,140 字节的 WebP（"其他" −2,133,479，JS +30：29 个导入和兜底文件名各多一个字符）；T15 删掉活动迭代的两张图（−140,562），改过的 11 张共 +1,904 字节，JS −200（删掉的属性和导入）。之后只有 JS 和"其他"变了：修复轮 JS −250（本分支 `9a98148`，合计 14,799,213）；T17 JS −264（粘贴处理变短）；T18 不变；T19 JS +233（两个文案键和对共享函数的调用；原型 `a994925` 的实测）；本分支的 T19 和它的跟进又比原型少 90（`c84f26d` 去掉取消说明外面的 `<span>`，`f9d1ab2` 去掉分组行外面的片段和 `div`；跟进之后 `f9d1ab2` 的 JS 是 6,586,782，"其他"仍是 4,162,901）；T20 "其他" −35,994（`issues.webp` −34,594，其余 8 张合计 −1,400），本分支 T20（`ff555bf`）的 JS 是 6,586,782，合计 14,763,098；T20 的原型 `c9b24e1` 是 JS 6,586,872、"其他" 4,126,953，合计 14,763,234（`intake-light.webp` 大 46 字节）。整分支评审之后的修复轮（3.16）只有 FW14 `e171d64` 改变构建：zh-CN `common` 的语言 chunk 少 7 字节（5 个英文值共 58 字节，换成的 17 个汉字按 UTF-8 是 51 字节）；FW15–FW17 不变。最后一列是 FW17 `73f8c36` 的实测（之后只有文档提交）。

### 3.5 锁文件（M1 设计 9、9.7）

`lock-diff.mjs`（P1 的方法）比较基线和终态的 `pnpm-lock.yaml`，除删除以外的变化只有：
- `patchedDependencies` 加 `prosemirror-codemark@0.4.2`，editor 的这条依赖带上补丁的哈希（T9）；
- `web/packages/utils` 的开发依赖加 `jsdom`（`catalog:`，30.1.1，锁文件里已有这个版本，editor、web 在用）（T8）。

删除的 17 个包是 `sanitize-html` 2.17.7、`@types/sanitize-html` 和只有它们用的 15 个（`htmlparser2`、`domhandler`、`domutils`、`dom-serializer`、`domelementtype` 各两个版本，`deepmerge`、`is-plain-object`、`launder`、`dayjs`、`parse-srcset`），以及 `catalogs` 里 `sanitize-html`、`@types/sanitize-html` 两条（T8）；propel 的 `cmdk` 依赖（T4，web 应用自己仍依赖它）。没有新增或升级的包。`stale-config.mjs` 在基线和终态都只报 `postcss` 的 catalog 条目：它经 `overrides` 固定版本、被 `tailwind-config` 使用，是脚本的误报。

### 3.6 守卫（T1、T2、T3、T4、T11）

- **收尾的 Phase**（T1）：`tools/keywords.mjs` 接受 `M<n>/closeout`，排在该 M 的所有编号 Phase 之后、下一个 M 之前。顶层 `phase` 为 `M1/closeout` 时，`until` 是 M1 任何 Phase 的例外都已到期，工具报错，只剩跨 M 的例外，这是 M1 设计 7.4 对收尾的要求，从此由工具检查。`t1/expiry-probe.mjs` 证明：把一条例外的 `until` 改为 `M1/P5`、`M1/closeout` 时以 1 退出（`1 expired exception`），改为 `M2`、`M6` 时通过，写成 `M1/close` 时以 2 退出。
- **样本**（T1）：每条规则的每个顶层分支、每个 `(?:a|b)` 的每一支都有命中样本，`files` 选择器也一样（新增 41 个）。T1 新增的这 41 个命中样本都取自本仓库的真实代码（大多是基线里、被后来的 Phase 删掉的行），只有三个是示意的写法，因为守卫看住的写法在仓库里从来没有过被跟踪的实例：`next-script-image` 的 `import Image from "next/image";`、`storybook-files` 的 `storybook-static/index.html`、`stickies` 的 `web/apps/web/public/stickies.yaml`。P1–P5 写的命中样本里有 39 个在本仓库从导入 web（`376f662`）起的历史中找不到原样的一行（整分支评审的 `hitreal.mjs`）：其中 23 个把 `@nerve/` 换回 `@plane/` 就能找到，是 P5 改名之前的行（例如 `import { GOD_MODE_URL } from "@nerve/constants";`），其余 16 个（15 行代码和路径 `.prettierignore`）本仓库的历史里没有，例如 `import Script from "next/script";`、`import posthog from "posthog-js";`、`"VITE_SENTRY_DSN",`。
- **不命中样本**（T1）：都引用保留的代码或路径。本分支的结果是替换 60 个、删除 5 个：T1 的提交 `aea47e4` 替换 58 个（其中 26 个是从未存在的 `app/assets/logo.svg`），删除 7 个（在同一边界已有其他不命中样本）；跟进提交 `483ca9d` 改了 5 个，让每个样本都在规则停止命中的地方检验它。3 个已替换的换成更好的样本：`deploy-files` 改为 `web/packages/constants/src/swr.ts`（以 "sw" 开头却不是 `sw.js`），`next-script-image` 改为 `import { useTheme } from "next-themes";`（以 "next" 开头却不是 "next/"），`i18n-key-generator` 改为 `generateQueryParams`（有 "generate"，没有 ":types"）；2 个恢复被删掉的：`storybook-files` 的 `e2e/stories/…`（有 "stories"，没有 `.stories.` 的点），`profile-stats` 的 `./user-user-profile`（有 "user-profile"，没有 "/"）。
- **样本的核对**：`missquote.mjs` 用子串在任何被跟踪的文件里找每个不命中样本（样本自己在 `keywords.json` 里的那一行也算）；`missreal.mjs` 只在规则读取的文件里找，更严。T11 删掉 `i18n-applications` 一个不命中样本引用的文案：原型改为同一段文字仍在的 `settings.json` 那一行，但那不是这条规则读取的文件，`missreal.mjs` 报 1；本分支（`f9fb015`）改为规则读取的 `workspace-settings.json` 第 103 行（`"description": "Any application using this token will no longer have the access to Nerve data. …"`），两个脚本都是 0。
- **新规则**：`window-open`（T2：`web/` 的源文件里，`window.open(` 所在的行没有 `noopener`）；`app-rail`（T3：`app[-_ ]?rail|use-?workspace-?paths`，不区分大小写，`web/` 的源文件和 JSON）；`app-rail-files`（T3：路径）。
- **例外**：T4 随 `tlds.ts` 删掉 `analytics`、`wiki` 两条（`until: M9`）；`brand` 在 `pnpm-workspace.yaml` 的 3 处（`until: M9`）重新核对，理由成立（记录配置来源的注释，在 v0 内不会消失），保留。

### 3.7 安全（T2、T5）

- **`window.open`**（T2）：编辑器链接的点击处理（`clickHandler.ts`）、附件列表、图片工具栏的下载、全屏弹窗的下载和打开原图，以及项目卡片、迭代、模块、视图、工作区视图（两处）、工作项的"在新标签页打开"，共 12 处，都传 `"noopener,noreferrer"`。这些调用都不使用 `window.open` 的返回值（带 `noopener` 时返回 `null`）。
- **`window.close()`**（T5）：propel 的 `MenuItem` 从 ui 的 `CustomMenu` 抄来 `close()`，但 propel 这里没有同名的局部绑定，调用的是全局的 `window.close()`。Chrome 会关掉由脚本打开、或只有一条历史记录的标签页，所以在项目标签页的溢出菜单（`TabNavigationOverflowMenu`）里选一项可能关掉整个标签页。base-ui 自己会关菜单，这次调用删除。`.oxlintrc.json` 把 `no-restricted-globals` 设为错误，列表取 ESLint 的 confusing-browser-globals（`close`、`name`、`event`、`status`、`open` 等读起来像局部变量的全局名）；树里唯一有意的用法 `use-reload-confirmation` 的 `confirm()` 改为 `window.confirm()`。

### 3.8 死代码（T3、T4、T5）

- **应用栏**（T3）：删掉 `core/lib/app-rail/`（4 个文件）、`navigation/app-rail-hoc.tsx`、`app-rail-root.tsx`、`items-root.tsx`、`hooks/use-workspace-paths.ts`；`use-navigation-preferences.ts` 里的 `useAppRailPreferences` 和它的本地存储键；`@nerve/types` 的 `TAppRailDisplayMode`、`TAppRailPreferences`、`DEFAULT_APP_RAIL_PREFERENCES`。只有应用栏喂的东西一起走：顶部栏读应用栏的显示模式，那只有应用栏能改，所以它的 `px-2` 分支永远走不到；`AppSidebarItem` 删掉只有应用栏传的 `label`、`showLabel` 和没人读的复合静态成员（`Label`、`Icon`、`Link`、`Button`）；propel 的右键菜单删掉只有应用栏用的 `Separator` 和 `Trigger`、`Content` 的 `className`。
- **包导出**（T4，3.2 结论 2）：每轮由 `t4/round.mjs` 完成：`deadsym.mjs` 列出没人读的导出 → 没有别的读取方、自己文件里也不用的声明删除，自己文件里还用的去掉 `export` → 什么都不再导出的文件连同桶文件里的那一行和包的子路径一起删除 → 这些删除在同一文件里留下的没用的声明和导入删除（按 `lintdiff.sh` 报出的新 `no-unused-vars`）。三轮共 512 个导出（没有别的文件读取的导出名，每个在声明它的文件里计一次）：274 个连同声明删除（263 个导出的声明；11 个 `export { … }` 里的名字，说明符去掉之后文件里没有别处用它，声明也删掉），238 个当时在自己的文件里还有人用，只去掉导出（220 个 `export` 关键字、18 个 `export { … }` 里的说明符；其中 64 个所在的文件随后什么都不再导出，整个删除）；另外去掉桶文件里按名字转出它们的 34 个说明符。`round.mjs` 每轮打印的 `delete`、`unexport` 数的是 `unexport.mjs` 的操作行，不是导出：三轮合计 274 行 `delete`（上面删掉的声明）和 283 行 `unexport`（220 个关键字、29 个声明所在文件里的说明符、34 个桶文件里的说明符）；那 11 个名字各占一行 `unexport` 和一行 `delete`，所以 274 + 283 − 11 − 34 = 512。166 个文件删除，其中 propel 的 accordion、avatar、badge、banner、collapsible、combobox、command、dialog、input、skeleton、switch、tabs、toolbar 组件和图标注册表连同只有它提到的图标（111 个文件；应用用的是 `@makeplane/propel` 的图标），ui 的 avatar、collapsible、input、tag、textarea。之后：没有导入方的 16 个包入口（`subpaths.mjs`：propel 的 `animated-counter`、`spinners`，两个样式路径的别名，一个指向不存在目录的 tsdown 入口等）；propel 的 `cmdk` 依赖；`tlds.ts`；propel `EmptyState` 的 `asset` 属性（传它的只有被删的 `EmptyState` 包装组件，`assetKey` 每个调用方都传，改为必填，选择二者的分支删除）；只被删掉的代码提到的 22 个键（用户角色、优先级筛选、两个迭代图标标签和 6 个只在 `tlds.ts` 里出现的顶级域名）。
- **点名的死代码**（T5，B4–B6）和复合组件没人读的部分：propel `ContextMenu.Submenu`、`SubmenuTrigger`；propel `Menu.SubMenu` 和只为它存在的子菜单上下文；ui `CustomMenu` 的 `Portal`、`SubMenuTrigger`、`SubMenuContent` 这三个静态成员（`CustomMenu.SubMenu` 保留；`Portal` 组件本身也保留，`CustomMenu` 的子菜单在用）。

### 3.9 退化结构和小项（T6）

- `rulecount.mjs . react/jsx-curly-brace-presence --fix`：oxlint 只用这一条规则修掉基线的 80 处（58 个文件），`.oxlintrc.json` 把规则设为错误。
- 4 个只给导入常量起别名的 `const`（`lite-text/toolbar.tsx`、`issue-layouts/utils.tsx` 两处、打盹弹窗）删除，代码直接读常量；`no-projects.tsx` 没有插值的模板字面量、收集箱描述外层多余的 `` `${…}` `` 删除。
- `labels-all.mjs --fix`：悬空的导入分组注释（标签下面没有导入：叠在另一个标签上，或留在最后一个导入之后）和 4 对相邻同名分组的第二个标签，158 个文件；只删注释行，不移动导入。脚本报 206 行悬空；本分支（`20ac5b6`，以它为准）发现脚本的写法在两个文件里留下一行裸的 `//`，却把它上面真正的分组标签当作悬空删掉，于是恢复这两个标签、改删裸的 `//`，另删一行 `//hooks`。实际共删 212 行注释：悬空的标签 204 行、重复的 4 行、裸的 `//` 3 行、`//hooks` 1 行。
- 错误页、维护页、无权限页三张装饰插图的 `alt="ProjectSettingImg"` 改为 `alt=""`（旁边的标题说明页面是什么）。
- `list-view-types.d.ts` 的 `TPlacement`（G4）。

### 3.10 编辑器与 HTML 工具（T7、T8）

- **标注块**（T7）：每个属性用 `parseHTML: (element) => element.getAttribute(name)` 读取；`logo-selector.tsx` 的 `.toString()` 删除。`extension-config.test.ts` 4 个用例：从 HTML 读出的标注块保留 `"128161"` 和其他属性，缺少的属性取默认值，写回 HTML 时码点不变；去掉 `parseHTML` 时第一个用例以 `128161` 失败。
- **HTML 工具**（T8）：`@nerve/utils` 的 `stripAndTruncateHTML` 用 `DOMParser` 取文本，实体会被解码（通知预览显示 "Tom & Jerry"，基线是 "Tom &amp; Jerry"）；只给它用的 `sanitizeHTML` 删除；`isEmptyHtmlString`（`isCommentEmpty` 和草稿弹窗在用）对编辑器的 HTML 给出同样的答案：不间断空格和 `<br>` 算空，允许的标签（图片、提及等）算内容；正文里 `<script>`、`<style>`、`<textarea>`、`<option>` 中的文字现在算作文字（`sanitize-html` 原来丢掉它们，编辑器不产生这些节点，第 4 节第 11 条）。标注块读自己的本地存储时不再过一遍 `sanitizeHTML`（它会转义 emoji 地址里的 `&`）。`string.test.ts` 5 个用例；utils 的测试在 jsdom 里跑（开发依赖）。

### 3.11 依赖与格式检查（T9、T10）

- **补丁**（T9）：`pnpm patch prosemirror-codemark@0.4.2` → `t9/strip.mjs`（删掉 `dist/esm`、`dist/cjs` 下 12 个文件的 `sourceMappingURL` 注释）→ `pnpm patch-commit`，生成 `patches/prosemirror-codemark@0.4.2.patch`；`pnpm-workspace.yaml` 在这条 `patchedDependencies` 上方用注释写明原因，editor 的 `vitest.config.ts` 的注释写明它现在怎样加载这个包。
- **格式检查**（T10）：`tailwind-config`、`typescript-config` 加 `check:format`、`fix:format`，不加 `check:lint`（它们里面是 CSS 和 JSON；唯一的脚本 `postcss.config.js` 匹配 `.oxlintrc.json` 全局忽略的 `*.config.{js,mjs,cjs,ts}`，oxlint 没有可读的文件）；`typescript-config` 删掉只在打包发布时起作用、又只列了 4 个配置中 3 个的 `files`（包是 `private`）。根目录的 `check:format` 加上 `package.json`、`pnpm-workspace.yaml`、`turbo.json`、`knip.jsonc`、`.oxlintrc.json`、`.oxfmtrc.json`；其中 `package.json` 的键按 oxfmt 的顺序重排、`knip.jsonc` 补上结尾逗号。`t10/negative.mjs` 证明每个新检查对它覆盖的文件改坏时失败、恢复后通过。

### 3.12 文案与图片（T11、T12）

- **文案**（T11）：`t11/theme.mjs` 先让主题选项的标签只剩一个来源（C3）；`keyref.mjs unused` 列出 482 个，`t11/traced.txt` 是追查过的 18 个（C2）；`delkeys.mjs` 从 en、zh-CN 一起删掉 500 个键和它们留下的空对象（每种语言 149 个）。中英文键一致性检查照常通过。
- **图片**（T12）：`assets.mjs` 列出名字没有被 `web/` 下任何文件提到的 133 张图片（应用显示的每张图都按文件名导入），`t12/rm.mjs` 删除它们（6225 KiB）和因此变空的 11 个目录：126 张空状态插图（画的是 Plane 自己的界面，布局、迭代、模块、收集箱、归档、草稿、个人主页、搜索、设置、筛选，深浅和 `-resp` 各一套，以及较早的 `empty_*.webp`、SVG），认证页的无权限插图和两张背景纹理，一个项目 emoji，命令键图形，两张图库人像。删除前拼成 300×200 的联系表逐张看过（`t12/sheet.mjs`）。封面照片和导览截图都被导入，保留（第 9 节第 1 条，后由 T14、T15 替换和修改）。按文件名查有一个漏洞：与在用的图同名、或名字是另一张图结尾的图会被当作在用。T15 复看时按路径查（`t15/assetpaths.mjs`），又找到 6 张，由 T15 删除（2.4 D3、3.15）。

### 3.13 文档（T13）

- 前端改动清单：新增 1.6 节（收尾的改动，13 行；T14、T15 各在末尾再加一行，共 15 行）；第二节加两行（从不渲染的应用栏；Plane 自身的死代码续：包导出、文案键、图片），状态"已完成 / M1/收尾"。
- README：格式检查也覆盖根目录的工具链配置（两处）。
- P2–P5 写给 M2–M8 的 17 份交接，在"来源"之前加一节"关闭条件"：该 M 合并时必须成立的事，最后一句"逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`"。
- 收尾给 M2–M8 各写一份 `handoffs/M1-closeout.md`：本 M 领域里的死成员和死 prop 的数目、列出和处理的方法、关闭条件；oxlint 的清理规则；M4 另有 `viewId as TProfileViews`；M8 另有共享部分的死成员和死 prop、包的 `license` 字段。它们引用本 spec 的 3.3、第 4、8 节和 plan 的附录 A（三个脚本的全文），以及收尾 review（`reviews/closeout-review.md`，控制者写）。

### 3.14 保留行为的核对（M1 设计 7.5）

| 手段 | 核对 | 落点 |
|---|---|---|
| 构建与静态检查 | 五个门禁；守卫；每个 Task 的孤儿核对；`lintdiff.sh`；锁文件 | 每个 Task |
| 进仓库的测试 | 77 → 101 个（T7 的 4 个、T8 的 5 个、修复轮的 14 个、T17 的 1 个）；测试输出没有 stderr（T9 起） | 每个 Task |
| 一次性核对 | `devwarn.mjs` 的外部化警告（T8）；`t1/expiry-probe.mjs`（T1）；`t10/negative.mjs`（T10）；封面和改过的图按能看清 20 px 的尺寸看（T14、T15、T20）；`t16/names.mjs` 对照夹具名和构建产物（T16）；`t18/prove.sh` 的冷、热构建，旧的 `Makefile` 上反向对照（T18） | 对应 Task |
| Go 的测试 | `make test`、`make lint-go`（T16） | T16 |
| 临时核对脚本（控制者写和跑，plan 最后一节） | 重跑 P2–P5 的探测；`window.open` 的参数和 `opener`；菜单项不调用 `window.close()`；通知预览的实体；评论和草稿的"是否为空"；标注块的属性和本地存储；Power K 的主题菜单；页面上没有原样显示的键、没有加载失败的图片；工作项表单的迭代下拉；预设封面的显示、选择和上传的副本（`t14/probe/covers-probe.mjs`）；改过的图在页面上的字节和活动迭代的空状态（`t15/probe/pictures-probe.mjs`，T20 之后预期不变）；编辑器之间的复制粘贴和跨源的剪贴板载荷（`t17/probe/copy-probe.mjs`）；活动迭代卡片的进度、中英文的文字和分组行的 key（`probe/active-cycle.mjs`，T19）；在基线的构建上做反向对照 | 收尾 review 附录 |
| S1–S4 | 照常通过 | 合并前 |

控制者的第 4 轮浏览器核对（T20 之后，`probe/run4.sh`）在 `ff555bf` 的构建上 1147 项全部通过：A 组 768（P4 的 9 段 403、P5 的 `brand.mjs` 177、原样的键和加载失败的图片 188），个人主页的标签 18（`a3-profile.mjs`），B 组第 4–11 项 203，第 12 项 32，第 13 项 81，第 14 项 22，第 15 项（T19，`active-cycle.mjs`）23。同样的脚本在基线 `c493255` 的构建上 1067 项通过、80 项失败，失败的正好是收尾改变的行为：A 组的原样的键 9（个人主页顶部栏窄屏时才显示的菜单按钮里是 `profile.tabs.*`，FW6）、`a3-profile.mjs` 4（同一处，480 px）、B 组 46（第 4 项 36、第 5 项 1、第 6 项 2、第 7 项 2、第 8 项 3、第 9 项 2）、第 12 项 2、第 13 项 11、第 14 项 1（C3）、第 15 项 7。第 15 项在基线上失败的 7 项：en 的标题是 "5/8 Work items closed"（B 场景 "10/8 Work items closed"）、进度 62.5（en、zh-CN 各一），zh-CN 的标题和说明是英文，分组行是不带 key 的片段；key 从 React 自己的记录读，因为生产构建不打印 key 的警告。

### 3.15 图片与测试夹具（T14、T15、T16）

- **封面**（T14）：
  - **封面值的来龙去脉**：预设只有一个来源，即 `helpers/cover-image.helper.ts` 的 29 个导入。应用选用预设时上传一份副本，项目和用户保存的是副本的地址。Go 服务、接口契约和 e2e 都不存、也不校验封面。
  - **画法**：`t14/covers.mjs` 用七种画法（色块、波纹、同心圆、斜条带、低多边形、半色调点阵、等高线）和 15 组配色，按种子画出 29 张，中到深色，画面中部有细节。
  - **格式**：SVG 进仓库作为可修改的形式。应用导入由它渲染的 WebP（1920 × 1080，质量 0.9），因为上传按文件头判断类型，只收 JPEG、PNG、WebP。
  - **格式的实测**（29 张合计，字节）：

    | 格式 | 宽 | 质量 | 合计 |
    |---|---|---|---|
    | 旧 JPEG | 1280 | — | 3,512,619 |
    | WebP | 1280 | 0.8 | 671,148 |
    | WebP | 1280 | 0.9 | 927,196 |
    | WebP | 1920 | 0.8 | 978,296 |
    | WebP | 1920 | 0.9 | 1,387,390（定稿 1,379,140） |
    | JPEG | 1920 | 0.85 | 2,037,864 |
    | PNG | 1280 | — | 14,126,738 |
    | SVG 源文件 | — | — | 523,700 |

  - **取值的理由**：封面都用 `<img class="object-cover">` 显示。1440 宽的窗口里最宽的槽是项目设置，900 × 176 CSS px，二倍屏上是 1800 设备像素，所以取 1920 宽；深色渐变在 0.8 有色块，所以取 0.9。
  - **看图**：每张按一半大小看过；应用的两种裁切看过；白字对比度最低 3.3。
  - **清单**：`SOURCES.md` 带 Nerve 的版权声明，写明做法和每张画的是什么。
- **头像和人名**（T15）：
  - 11 张图里 77 个照片头像换成字母头像，做法见第 4 节第 21 条；
  - "vamsi" → "vera"，"Aaryan" → "Robin"，"Bhavesh Raja" → "Robin Park"；
  - 改过的地方都按 2 倍看过；两个目录的 `SOURCES.md` 保留上游的声明，写明每张改了什么；
  - 另删没有显示的 8 张图和活动迭代的死属性（2.4 D3）；
  - 交给 M5 的两项（8.1）写进 `docs/v0/M5-files/handoffs/M1-closeout.md`。
- **保留图片的复看**（T15；T12 之后的 113 张减去 29 张封面，84 张，按能看清 20 px 的尺寸看）：

  | 图片 | 画的是什么 | 结论 |
  |---|---|---|
  | `404.svg` | 灰色大字 "404" 和 "ERROR" | 保留 |
  | `auth/project-not-authorized.svg`、`workspace-not-authorized.svg`、`unauthorized.svg` | 粉色锁的卡片，分别写 "Project Settings"、"Workspace Settings"、"Error unauthorized" | 保留 |
  | `instance/maintenance-mode-light.svg`、`-dark.svg` | 等轴测的窗口和服务器（P5 已删掉屏幕上的 Plane 标志） | 保留 |
  | `brand/` 9 张、`public/icons/` 2 张 | Nerve 的标志、横版标志、网站和应用图标；分享图写 "nerve / Open-source project management." | Nerve 自己的，保留 |
  | `attachment/` 17 张（512 px 的文件类型图标） | audio 紫色音符；css；csv 带 Excel 的 X 标志；default 黄色文件夹；doc 蓝底 "DOC"；excel-icon 是 Google Sheets 的图标；figma 是 Figma 的标志；html；jpg；js；pdf 是 Adobe Acrobat 的标志；png；rar；svg；txt；video 红底白色播放键；zip | 没有人和人名；4 张带第三方的标志，交 M5（2.4 D4） |
  | `empty-state/active-cycle/` 10 张 | 圆里的灰色线条图标（人形、折线图、标签、信号格、上升箭头），深浅各一 | 保留 |
  | `empty-state/cycle.svg`、`module.svg`、`view.svg`、`issue.svg`、`invitation.svg` | 灰白的清单、文档和信封线稿，带蓝色的勾和线 | 保留 |
  | `empty-state/cycle/all-filters.svg`、`name-filter.svg`、`module/all-filters.svg`、`name-filter.svg` | 迭代或模块的图标，下面一个蓝色的筛选或放大镜圆钮 | 保留 |
  | `empty-state/empty_label.svg`、`empty_members.svg` | 灰色标签；人形剪影 | 保留 |
  | `empty-state/search/issues-*.webp`、`search-*.webp` | 圆里叠放的三张卡片；放大镜 | 保留 |
  | `empty-state/project-settings/no-projects-*.png` | 项目卡片 "Exciting project ahead!"、"MVP-42"（P5 已去掉 Plane 标志） | 保留 |
  | `workspace/workspace-creation-disabled.png`、`workspace-not-available.png` | 灰色线稿，底部圆徽是 Nerve 的标志（P5） | 保留 |
  | `user.png` | 黑色的通用人形图标 | 保留 |
  | `onboarding/` 4 张 | 界面截图，项目和工作项的名字是虚构的 | `cycles`、`views`、`issues` 由 T15 改；`modules` 没有头像和人名，不改 |
  | `empty-state/disabled-feature/` 8 张 | 迭代、模块、视图、收集箱关闭时的界面截图（工作区名 "Acme Design"，P5 改的） | T15 改头像和人名 |
  | `empty-state/cycle/active-*.webp` 2 张 | 活动迭代的界面截图，负责人的头像是照片 | 从不显示，T15 删除 |
  | `empty-state/intake/intake-*.webp` 2 张 | `disabled-feature/intake-*.webp` 的逐字节副本 | 没有引用，T15 删除 |
  | `empty-state/search/views-*.webp` 2 张 | 圆里的图层图标加蓝色放大镜 | 没有引用，T15 删除 |
  | `empty-state/project/name-filter.svg` | 项目图标加蓝色放大镜 | 没有引用，T15 删除 |
  | `empty-state/label.svg` | 标签和文档围成的圆 | 没有引用，T15 删除 |

  没有找到别的真人照片、真实人名或 Plane 的标志。propel 的空状态插图是矢量组件；仓库里的代码和 SVG 都没有内嵌的位图（`data:image` 只出现在两个测试里）。
- **测试夹具**（T16）：`handler_test.go` 的 `built` 用构建真实产出的 `site.webmanifest.json`、`icons/icon-192x192.png`，回退用例的目录用 `/icons`，注释"mirrors the layout of web/apps/web/build/client"因此成立。`t16/names.mjs` 对照构建为 0（基线为 3）；`make test` 通过，`make lint-go` 为 `0 issues.`。

### 3.16 修复轮（T16 之后）

各 Task 的评审和控制者的浏览器核对发现的小缺陷，没有原型，T16 之后由控制者派发（`.superpowers/sdd/closeout/fixbatch-brief.md`），实现者自己设计，每项一个提交；编号是控制者记录里的编号。前端改动清单 1.6 节有它们的 7 行（`4be6cba`）。

| 编号 | 提交 | 改动 | 来源 | 文件，行（+/−） |
|---|---|---|---|---|
| FW1 | `1896bca` | propel 右键菜单的 `Trigger`、`Content` 的属性类型去掉 `className`（`Omit`）：T3 删掉了它们自己声明的 `className`，继承的 base-ui 属性里还有，传进来时 `Trigger` 会用它换掉 `outline-none`，`Content` 会不声不响地丢掉它；现在传它是类型错误 | T3 的评审 | 1，+5 / −2 |
| FW3 | `c167005` | 4 行只有 `//` 的导入分组标签删除或改为 `// local imports`；模块链接列表项的两个错标改名（`// nerve imports`、`// nerve utils`）；只改注释行 | T6 的评审 | 5，+4 / −6 |
| FW4 | `28bedba` | 标注块从本地存储读上次用的图标时先检查形状，不符合的值写日志、删掉、改用默认值；原来 `as TLogoProps` 直接信任，T7 删掉 `.toString()` 之后，不是字符串的 emoji 值会让插入抛错；editor 加 14 个测试 | T7 的评审 | 2，+119 / −27 |
| FW5 | `773b3d8` | 算作内容的标签只写在 `isEmptyHtmlString` 里（`img`、`image-component`、`mention-component`），草稿弹窗和 `isCommentEmpty` 都调用它：只有图片或只有提及的草稿，关闭时不再不问就丢掉；带测试 | 控制者的浏览器核对（基线和本分支都能重现） | 3，+24 / −19 |
| FW6 | `6f2b484` | 个人主页的顶部栏在窄于 768 px 时，菜单按钮显示标签的名字（`t(currentTab.i18n_label)`），不再显示 `profile.tabs.assigned` 这样的键 | 控制者的浏览器核对（`a3-profile.mjs`，480 px） | 2，+7 / −5 |
| FW7 | `4ea6397` | 仓库根目录加 `fix:format`（文件列表与 `check:format` 相同），`turbo.json` 登记 `//#fix:format`（不缓存），`turbo run fix:format` 也格式化根目录的文件 | T10 的评审（P1 spec 的保证） | 2，+5 / −1 |
| FW8、FW9、FW13 | `827823c` | 文档：M4 的收尾交接加两节（工作项弹窗把描述的 ID 迁移当作改动；标注块的 Markdown 序列化不转义，8.1）；512 个导出的单位（3.8）；8.1 M5 一行的请求顺序；前端改动清单 T15 一行的措辞 | T8、T13、T15 的评审 | 4，+27 / −10 |
| FW10 | `6048ad3` | `isCommentEmpty` 只接受评论的 HTML，走不到的 JSON 分支和只有它用的 `Content`、`HTMLContent` 删除；utils 的 oxlint 上限 19 → 18 | 修 FW5 时发现 | 6，+8 / −82 |
| — | `4be6cba` | 文档：前端改动清单加修复轮的 7 行；本 spec、交接里随修复轮变的数字 | 修复轮 | 9，+43 / −36 |
| — | `9a98148` | `JSONContent` 的 `type`、`content`、`text` 恢复（它描述 TipTap 的 JSON，只写一半不如不写），与 `TIssueComment.comment_json` 一起交 M4（3.2 结论 3） | 控制者复核 FW10 | 10，+22 / −18 |

修复轮之后（`9a98148`）：五个门禁通过；vitest 100 个；`deadsym.mjs` 2 / 898 / 452；oxlint 上限合计 695。控制者的第 3 轮浏览器核对（`9a98148` 对基线）：A、B 两组全部通过；在基线上失败的都是收尾修掉的，其中修复轮的两项是草稿（plan 浏览器核对第 7 项，FW5）和个人主页窄屏的顶部栏（FW6）。

**整分支评审之后的修复轮**（T20 之后，`.superpowers/sdd/closeout/fixwave-brief.md`）：整分支评审（opus，`c493255..26cdb33`）的结论是改完即可合并，0 个 Critical、0 个 Important、7 个 Minor（R1–R7）。编号接着上面的修复轮，每项一个提交，基点是 `26cdb33`：

| 编号 | 提交 | 改动 | 来源 | 文件，行（+/−） |
|---|---|---|---|---|
| FW14 | `e171d64` | zh-CN `common` 的 5 个设置侧边栏分组标题翻成中文（"您的个人资料""开发者""工作结构""执行""管理"）；zh-CN 与英文相同的值只剩不用翻译的 6 个（URL、两个 ID、Webhooks、两个 `name@company.com`，`t20/investb-untranslated.mjs`）；构建的 JS 少 7 字节（3.4） | 2.10 的 I2 | 1，+5 / −5 |
| FW15 | `df4cbf7` | `services` 的 `helpers/index.ts` 只按名字转出 `normalizeAPIRequestURL`，包的公开导出 5 → 4 个（`ensureAPITrailingSlash` 不再公开） | 2.9 的 Minor 1（B12，第 9 节第 7 条） | 1，+1 / −1 |
| FW16 | `8a6bea5` | `workspace/content-wrapper.tsx` 的 `// nerve imports` 改为 `// components`（T3 删掉 `@nerve` 的导入之后留下的错标） | 整分支评审 R2 | 1，+1 / −1 |
| FW17 | `73f8c36` | 根目录格式检查的文件列表只写在 `fix:format` 里，`check:format` 是 `pnpm run fix:format --check`；README 指向这个脚本，不再抄一遍列表 | 整分支评审 R3（FW7 留下的两份列表） | 2，+3 / −3 |

其余 5 个 Minor（R1、R4–R7）和控制者逐个提交重跑（`branchrun`）发现的两处不准（T4 的 `infile-orphans` 在本分支是 166 行；`deadorph` 列出 `content`、`text` 的是 `9a98148`，不是 FW10）在随后的文档提交里改，前端改动清单 1.6 节加上面 4 行。之后：五个门禁通过；vitest 101 个；`deadsym.mjs` 2 / 898 / 452，`deadorph.mjs` 对 `26cdb33` 为 0；oxlint 上限合计 694；每种语言 1079 个键；构建合计 14,763,091 字节（3.4）。

### 3.17 Codex 的发现（T17–T20）

- **粘贴**（T17，2.9 Critical 1）：
  - `processAssetDuplication` 用 `DOMParser` 解析剪贴板里 `text/nerve-editor-html` 的 HTML，每类节点的处理函数原地标记元素（已上传的图片设 `status` 为 `duplicating`、换新的 `id`），最后序列化一次 `body.innerHTML`；原来的字符串替换和第二次解析删除。
  - 结果照旧交给 ProseMirror 的 `pasteHTML`，它按 schema 再解析一次（3.2 结论 8）。同编辑器之间复制已上传的图片，仍然请求复制资源（`copy-probe.mjs` 的 C1）。
  - `editor-interaction.test.ts` 加 1 个用例（旧代码上失败）；`vitest.setup.ts` 声明 jsdom 没有的 `ClipboardEvent`。
- **构建目录**（T18，2.9 Important 1）：`Makefile` 加 `WEB_CLIENT := web/apps/web/build/client`，`build-web` 在 Turbo 之前 `rm -rf $(WEB_CLIENT)`。`t18/prove.sh` 在产物目录里放一个文件再构建：旧的 `Makefile` 上热构建之后它还在，新的上冷、热构建之后都不在，两次的 531 个文件逐字节相同（3.2 结论 9）。跟进提交 `e070065`：`make build` 的 `cp` 也读 `$(WEB_CLIENT)`，这个路径只有一个来源；`build-web` 的 `##` 说明仍写字面的路径，因为 `make help` 不展开变量。
- **活动迭代的进度**（T19，2.9 Important 2）：`active-cycle/progress.tsx` 的进度条用 `calculateCycleProgress`，文字写完成 / (总数 − 取消)。文字和取消的说明改为两个文案键 `project_cycles.active_cycle.work_items_completed`、`cancelled_excluded`：en 用 ICU 复数，zh-CN 是"{completed}/{total} 个工作项已完成""报告中已排除 {count} 个已取消的工作项。"；取消的说明外面一层只包着文字的 `<span>` 删除（本分支 `c84f26d`，原型保留了它）。跟进提交 `f9d1ab2`：分组行的 `map` 原来返回一个不带 key、只有一个子元素的片段，key 写在里面的 `div` 上，React 报 key 的警告；现在返回可点击的那一行，`key={group}`，外面那层没有属性的 `div` 删除（它是列的弹性子项，宽度一样，布局不变）。数组下标不再作 key，web 的 oxlint 上限 566 → 565（`react/no-array-index-key`）。
  - 卡片和迭代列表现在是同一个口径；迭代侧边栏的数字仍是完成 / 总数（取消的算在总数里），模块的进度有 (完成 + 取消) / 总数 和 完成 / 总数 两种写法。收尾不改，交 M6（8.1）。
- **截图**（T20，2.9 Important 4）：
  - 10 个控件用框外紧挨着的背景色填平（`t20/spec.json` 的框和取色点），只有 `intake-light.webp` 的一框例外，见下；WebP 按"大小最接近原图"的质量重新编码（T15 的做法）：

    | 图 | 涂掉的 | 字节（前 → 后） |
    |---|---|---|
    | 导览 `issues.webp` | 面包屑旁的 "Public" 可见性标签（项目发布） | 229,264 → 194,670 |
    | 导览 `cycles.webp` | Add Issue 左边的数据分析按钮；工作项栏的甘特图布局图标 | 154,282 → 153,810 |
    | 导览 `views.webp` | 甘特图布局图标 | 135,462 → 134,650 |
    | "功能未开启"的 `modules-light`、`modules-dark` | 标题栏的甘特图/时间线布局图标 | 67,542 → 67,726；64,154 → 64,440 |
    | "功能未开启"的 `cycles-light`、`cycles-dark` | 同上 | 67,230 → 67,294；72,372 → 72,190 |
    | "功能未开启"的 `intake-light`、`intake-dark` | 工作项属性里 "Estimate: 2 Months" 一行 | 86,990 → 87,076；83,694 → 83,140 |

  - **`intake-light.webp` 的一框**（`[1700,1090,650,44]`，控制者的裁定）：那一行被卡片的下边缘截断，框的下部压着卡片的底边和卡片下面透明的边距。原型按别的框的做法填纯白（87,122 字节，SHA-256 前 16 位 `71eb35783ace0bad`），抹掉了 650 px 长的一段底边，还把白色涂进透明的边距，在应用里看得见；实现者看图时发现。本分支改为重复框左边一列（x=1699）的像素（`patch.mjs` 的 `"fill": "left"`），这一列从上到下带着卡片的白、底边和透明的边距：87,076 字节，`cebac06cb5461d08`。其余 8 张与原型逐字节相同（`git diff --stat c9b24e1 ff555bf -- web/apps/web/app/assets` 只列这张图和 `disabled-feature/SOURCES.md`）。
  - 每个框的前后对照放大 4 倍看过；12 张截图（导览 4 张、"功能未开启"8 张）按原尺寸重新看过一遍，没有找到别的已删功能。
  - 两个 `SOURCES.md` 各加一段做法说明和一个 "Removed" 表，`disabled-feature/` 的一段写明 `intake-light.webp` 的填法；不写 "Analytics"，因为守卫的 `analytics` 规则读 `web/` 下的所有文件。
  - `pictures-probe.mjs` 在运行时读仓库里的文件，T20 之后仍是 81 / 0（基线 70 / 11）。

---

## 4. 与上级设计的差异和补充

每一条都已按"能自己定的就自己定"的原则决定，列出决定和理由，供控制者复核。

1. **守卫认识 `M<n>/closeout`（T1）。** 设计 7.4 说收尾时"只允许剩下明确跨 M 的例外"，原来靠人核对。工具加一种 Phase 写法，排在该 M 的编号 Phase 之后，于是 M1 内到期的例外由工具报错。它改动的是 `tools/keywords.mjs` 的 Phase 解析（一个正则、一个常量），`t1/expiry-probe.mjs` 证明到期、不到期和写错三种情况。
2. **每个 `window.open` 都带 `noopener,noreferrer`，包括同源的"在新标签页打开"（T2）。** P5 评审点名的是打开外部内容的 5 处。同源的 7 处不会泄露给别的源，但一条对所有调用都成立的规则才能由守卫看住；这些调用都不读返回值，加上之后行为不变。`target="_blank"` 的链接不改（A2）。
3. **`window.close()` 的缺陷和 `no-restricted-globals`（T5，补充）。** 在清点菜单的复合组件时发现。只删这一行会留下同类缺陷的入口；ESLint 的 confusing-browser-globals 列表正是为这类"缺了局部绑定、落到全局"的情况设计的，设为错误后树里只有一处有意的用法，写成 `window.confirm`。
4. **死成员和死 prop 按领域交给后续 M，不在收尾删（B7）。** 收尾结束时还有 1350 个（成员 898、prop 452）。按总体设计 9.2 的领域划分（`domains.mjs`，按路径取第一个匹配）：M2 66（59 / 7）、M3 210（142 / 68）、M4 561（438 / 123）、M5 48（40 / 8）、M6 84（46 / 38）、M7 88（63 / 25）、M8 6（6 / 0），共享 287（104 / 183）。理由：
   - M2 起每个 M 按总体设计 7.2 重写本领域的 types、services、stores 和组件，前端的数据结构改用 OpenAPI 生成的类型；成员里的大多数是 Plane 接口类型的字段和 store 的方法，会随重写消失，收尾先删一遍等于做两遍；
   - 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员脚本也会列出（例如 ui 表格列对象的 `thRender`、`tdRender` 由 `useProjectColumns` 写入）；删一个 prop 还要收掉读它的分支。这是逐处的工作，不能写成脚本；
   - 收尾的每个 Task 仍然删掉它自己造成的孤儿（`deadorph.mjs` 每个 Task 为 0，T7 的 `parseHTML`，修复轮 FW4 的 `TLogoProps.emoji.url` 和 `9a98148` 恢复的 `JSONContent` 的 `type`、`content`、`text` 除外，3.2 结论 3），评审点名的都已删除。
   交接里写明列出的命令、处理方法和关闭条件；共享部分（propel、ui、types、utils、constants、hooks、i18n 和 web 的通用组件）交 M8，M2–M7 改到时照做。
5. **包导出用类型检查器找，删到零为止（T4，3.2 结论 2）。** P3 交接用的是 `symref.mjs`（347 个）；它按名字数，漏掉同名局部变量遮住的死导出，也数不到"只被重新导出"的情况。`deadsym.mjs` 的第一轮就有 424 个。删除后 `infile-orphans` 的 166 行都是被删的符号（本分支；原型 165 行，差的 `THEMES` 见 3.2）。
6. **`tlds.ts` 整个删除（T4）。** 两条 `M9` 例外原本要"重新核对理由"。核对发现文件唯一的读取方是 T4 删掉的地址解析链（没有调用方），文件和例外一起删。
7. **propel `EmptyState` 的 `asset` 属性（T4，补充）。** 重放之后的逐 Task 核对（`deadorph.mjs`）发现它在 T4 之后只有读取、没有传入；按"孤儿由造成它的 Task 删"在 T4 删掉，`assetKey` 改为必填。
8. **`react/jsx-curly-brace-presence` 设为错误（T6）。** P4 评审说"收尾统一收掉"。只修不设规则，下一个 M 会再写出来；这条规则 oxlint 能自动修，设为错误不会给后续 M 带来负担。
9. **导入分组的错标全部收掉（T6）。** 评审点名的是 `root.tsx` 的一行和 4 对；`labels-all.mjs` 找到同类的共 206 行（悬空）和 4 行（重复），与 P5 的 `labels.mjs` 同一判断（形如 `// hooks` 的标签，下面第一行不是导入，或与上一组同名）。只删注释行，不移动导入。本分支实际删了 212 行注释：悬空的标签 204 行（脚本在两个文件里错把真正的标签当作悬空，本分支恢复了它们）、重复的 4 行、裸的 `//` 3 行、`//hooks` 1 行（3.9）。
10. **`TPlacement` 的类型漏洞（T6，补充）。** 在核对 T4 的导出时发现。`skipLibCheck` 让 `.d.ts` 里失败的导入静默成为 `any`；改为它实际传给的 `CustomMenu` 的 `placement` 类型之后，类型检查照常通过（快捷操作传的值都在 `CustomMenu` 支持的范围内）。
11. **删掉 `sanitize-html`，而不是"记录去处"（T8）。** P4 评审把警告"归到依赖复核"。根因是本地的、小的：三个工具函数，都能用浏览器自带的 `DOMParser` 写，应用没有服务端渲染。行为上的变化有两处：一是修正，通知预览不再把 `&` 显示成 `&amp;`；二是正文里 `<script>`、`<style>`、`<textarea>`、`<option>` 中的文字现在算作文字（预览会显示它，"是否为空"为否），`sanitize-html` 原来把它们丢掉——编辑器产生的 HTML 从来没有这些节点，所以对编辑器的 HTML，"是否为空"的答案不变（5 个测试，控制者的探测再核对）。依赖少了 17 个包。
12. **追查名字与无关字面量相同的键（T11）。** 设计 6 节说"不能证明是死键的，先查清所有变量来源，不直接删"。`keyfalse.mjs` 列出 34 个只被"不像翻译调用"的字面量命中的键，逐个追到字面量的用处：18 个到不了 `t()`，是死键，删除；16 个经常量、配置或组件属性到得了 `t()`，保留（T11 之后脚本只再列出其中 12 个：主题选项的 4 个改由 `i18n_label` 引用，脚本认得这种写法）。
13. **主题选项的标签只剩一个来源（T11）。** P2 评审留下的二选一。`THEME_OPTIONS` 同一个标签有三份：`key`（设置页当键用，`common` 里有翻译）、`i18n_label`（英文原文，Power K 的主题菜单当键用，找不到键，中文界面下显示英文）、`themes.theme_options.*.label`（没人读）。按全仓库其他选项列表的写法，`i18n_label` 就是键：改为 `common` 里的键，设置页和 Power K 都读它，`key` 删除，第三份随死键删除。用户看到的变化：Power K 的主题菜单在中文界面下显示中文，英文界面下 "System preference" 变为与设置页相同的 "System Preference"。
14. **删除的图片按 300×200 的联系表看（T12）。** P5 的裁定"按能看清 20 px 细节的尺寸看"针对的是要保留、会显示给用户的图片（找其中的 Plane 标识）。这 133 张不随应用发布，看它们是为了确认删的是什么（都是 Plane 的界面截图、纹理和图库人像，没有被别名或动态路径引用：`assets.mjs` 按文件名查整个 `web/`）。
15. **格式检查的范围（T10）。** P1 评审说"仓库根目录自身的文件"。收尾加的是前端工具链自己的配置（6 个）。`.github/workflows/*.yml`、`deploy/*.yaml` 今天也能通过 oxfmt，但它们属于 M0 的构建与部署，不在 `make lint-web` 的范围里；`README.md`、`docs/` 是文档；`pnpm-lock.yaml` 是生成的。
16. **路由测试的宽松之处不收紧（F5）。** 见第 2.6 节。
17. **17 份交接补"关闭条件"（T13）。** 设计 11 节要求"交给后续 M 的事项已放进对应 M 的 `handoffs/`"，任务书要求每份都有接收的 M 和关闭条件。P2–P5 的交接有接收的 M（目录），没有写什么时候算完成；每份补一节，写该 M 合并时必须成立的事。
18. **M1 的状态由控制者在收尾 review 的提交里改（H7）。** 设计 9 节说收尾时改为"已完成"。实现者的 Task 在评审、修复和合并之前，那时写"已完成"不真实；与 P5 相同，由控制者在 review 的提交里改 M1 设计 12 节、11 节的勾和总体设计 9.4（时机见第 9 节第 3 条）。

第 19–26 条是 T14–T16 的原型带来的决定（第 9 节第 1、2 条的裁定之后），控制者已采纳（plan 的"T14–T16 原型之后的裁定"）。

19. **封面的源文件是 SVG，应用导入 WebP（T14，实测）。** 第 9 节第 1 条原先建议"文件名不变，代码不改"。实测之后：上传按文件头判断类型，SVG 判断不出，Plane 的资源接口也不收 SVG，所以不能直接发布 SVG；JPEG 在同样的尺寸下大 48%，PNG 大十倍。于是 SVG 作为可修改的形式进仓库，WebP 由 `t14/render.mjs` 生成，做法写在 `SOURCES.md`。扩展名随之从 `.jpg` 变为 `.webp`，辅助函数的 29 个导入和上传的兜底文件名一起改。
20. **不留兼容层（T14）。** 预设的文件名只有辅助函数的导入表读取。已保存的封面值是上传副本的地址，不指向预设文件。Go 服务还没有项目和资源接口。万一有保存了构建路径的值，那个值在任何一次重新构建后（哈希变了）就已失效，与本次改动无关（见 8.1 交 M5 的一条）。
21. **字母头像的做法（T15）。** 平的圆，按字母取六种颜色之一，白色 Inter 500 的字母；叠在上面的圆按原位置剪开，缝隙用背景色补回。"功能未开启"几张里的计数徽章（"+2"、"+210"）底下也是照片，一并换成深蓝底的计数；导览里浅色的 "+3"、"+12" 徽章不是照片，保留。WebP 重新编码的质量取"大小最接近原图"的一档。
22. **人名换成中性的名字（T15）。** "vamsi" → "vera"；"Aaryan" → "Robin"（视图名 "Robin’s Issues" 和说明里的 "Robin."）；"Bhavesh Raja" → "Robin Park"。字重、字号、颜色和基线按旧字拟合。
23. **没有显示的 8 张图在 T15 删除（补充）。** T12 的 `assets.mjs` 按文件名查，同名和"名字是另一张结尾"的图互相遮住；活动迭代的两张被导入，却只传给一个没人读的属性。它们在 T15 复看时发现，按"孤儿由发现它的 Task 删"放在 T15；不移进 T12（T12 的树要与原型相同，而且它已执行）。之后按路径查用 `t15/assetpaths.mjs`，查"导入了、却传给没人读的值"用 `t15/uses.mjs`。
24. **`onboarding/modules.webp` 不改（T15）。** 2.4 D2 原先说它有真人头像，复看发现没有头像，也没有名字；有头像的是 `issues.webp`。
25. **`onboarding/views.webp` 的 5 px 残边保留（T15）。** 一个深色头像只露出视图面板边缘下的一条 5 px 的边，看不出是照片；为它改动面板边缘得不偿失，写在 `SOURCES.md` 里。
26. **T16 只改测试数据。** `handler.go` 只读 `index.html` 和 `assets/` 前缀，与文件名无关；改的是 Go 服务的测试，不算跨模块的协同。

第 27–31 条是 Codex 对抗评审带来的决定（2.9），负责人已批准分诊，T17–T20 的原型已采纳（plan 的"Codex 对抗评审之后的裁定"）。

27. **T17 只把解析换到惰性文档里，不另写白名单。** Codex 推荐的做法是先按编辑器的节点做白名单清洗、再交给 `pasteHTML`。ProseMirror 按 schema 解析就是白名单：只留 schema 声明的节点和属性，而且在它自己新建的、不执行脚本的文档里做（3.2 结论 8）；缺口只在它之前、在活动文档里的那次解析。另写一份白名单就是同一件事的第二个来源，还要随 schema 同步。自定义类型保留，不采用 Codex 的做法 2（删掉这一支）：删掉之后，两个编辑器之间复制的已上传图片会指向同一个资源（`copy-probe.mjs` 的 C2 实测：不请求复制，图片仍是原资源），在一处删掉或换掉它，另一处跟着变。
28. **T18 只清 `build/client`，而且放在 `make build-web` 里。** `make build` 复制的就是这个目录；清空要在 Turbo 恢复缓存之前，Turbo 自己没有这一步。Codex 的做法 2（`make build` 复制时按清单筛选）要维护清单，漏列就漏打包，不采用。持续集成的 web 任务和 e2e 任务的 Build 一步从干净的检出开始，`rm -rf` 在那里什么都不删；e2e 任务接着跑的 `make e2e` 依赖 `build`，再构建一次，这时 `rm -rf` 删掉第一次构建的 `client`，Turbo 从本任务自己的缓存恢复它（3.2 结论 9）。跟进提交 `e070065` 让 `make build` 复制时也读 `$(WEB_CLIENT)`：两处各写一遍路径，改了一处，清空的和嵌入的就不是同一个目录。
29. **T19 的文字与进度条同一口径，并进 i18n。** "完成 / (总数 − 取消)"是 M1 设计 3.4 的取消项排除口径，进度条经 `calculateCycleProgress` 计算，只有一份定义。原来的两句英文由模板字面量拼成、没有翻译，改为两个键（en 用 ICU 复数，zh-CN 不分单复数），每种语言 1077 → 1079 个键；"closed" 改为 "completed"，因为取消的不再算进分子。Codex 要的取消 > 0 和全部完成两个例子已在 `progress.test.ts` 里（38、100），卡片经它计算，不另加测试；web 应用的测试只在 Node 里核对路由表，没有渲染这张卡片的环境，文案的复数形式用 intl-messageformat 核对过，页面上的显示由控制者的浏览器核对（plan 第 15 项，3.14）。迭代侧边栏的数字和模块的进度另有口径（3.17）；收尾只改 Codex 点名的卡片，统一口径交 M6（8.1）。
30. **T20 用旁边的背景色把控件填平，不重画截图。** 这些控件都在平的背景上（白、浅灰、深色面板），填平之后看不出来，其余像素只经历一次有损的重新编码，与 T15 相同。`intake-light.webp` 的一框例外：它压着卡片的底边和卡片外透明的边距，纯色会抹掉底边，所以重复框左边一列的像素（3.17，控制者的裁定）。Codex 的做法 2（换成自己的插画）要重新设计四个空状态和导览，超出收尾。`issues.webp` 小了 34,594 字节：去掉蓝色的字之后，最高的质量 0.95 也到不了原来的大小；其余 8 张相差不到 1,000 字节（最多是 `views.webp` 的 812）。
31. **B12 改正（Codex Minor 1）。** 本 spec 原来写"桶文件已不存在，已在 P3、P4 解决"，没有核对。`deadsym.mjs` 把测试文件的读取也算作使用，所以 T4 的三轮都没有报它。只收窄 `services` 的 `helpers/index.ts`，函数和它的测试不动，放在整分支评审之后的修复轮（Codex 推荐的做法 1）。控制者分诊时依据的是 B12 原来的写法（"T4 已解决"），这个去向由控制者确认（第 9 节第 7 条），做在 FW15 `df4cbf7`（3.16）。

---

## 5. 验收标准

### 5.1 M1 设计 11 节逐项

| # | 完成标准 | 本 Phase 的证据 |
|---|---|---|
| 1 | P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review | P1–P5 已有；收尾的 spec、plan（本提交）和 review（控制者） |
| 2 | 类型检查通过；knip 为零且是门禁；oxlint 每个包等于新上限，上限自动核对；格式检查通过；前端单元测试在持续集成中运行并通过 | 每个 Task 的五个门禁；上限 694；格式检查覆盖到两个配置包和根目录（T10）；vitest 101 个 |
| 3 | 守卫在持续集成中运行，没有未登记的命中，只剩明确跨 M 的例外；`until` 超出 M8 的逐条核对过；没有无引用的键 | `keywords: 51 rules, 3 exceptions, no hits.`，`phase` 为 `M1/closeout`（工具检查到期）；`brand` 的 `M9` 例外核对过（3.6），`tlds.ts` 的两条随文件删除；`keyref.mjs unused` 为 0 |
| 4 | 7.5 的保留行为矩阵逐行核对，临时脚本在各 Phase review 的附录 | 控制者的探测（plan 最后一节）重跑 P2–P5 的场景，写进收尾 review 附录 |
| 5 | `make build` 能构建；S1–S4 通过 | 门禁；合并前 S1–S4 和持续集成 |
| 6 | 只剩 `zh-CN` 和 `en`；界面上没有 Plane 的名称和 Logo；包名都是 `@nerve/*` | P1、P5 已完成；守卫的 `plane-package`、`brand`、`brand-files` 照常通过；删掉的 133 + 8 张图里的 Plane 界面截图不再在仓库里；保留的图片复看过，没有 Plane 的标志和真人（3.15） |
| 7 | oxlint 清零计划、构建体积对比写进收尾 review | 本 spec 3.3、3.4，review 复述 |
| 8 | `handoffs/` 中没有 `open`；交给后续 M 的事项在对应 M 的 `handoffs/` | M1 的两份在基线已 `closed`；17 份补关闭条件，7 份新交接（T13）；M5 的交接另加两项（T15），M4 的另加两项（修复轮），M5、M6 的各再加一项（T20 之后的文档提交），M2、M8 的另加几项（收尾 review 的提交，2.9、2.10，8.1） |
| 9 | 前端改动清单同步；总体设计中 M1 的状态改为"已完成" | T13；状态由控制者改（第 4 节第 18 条） |

### 5.2 本 Phase

- [ ] 20 个 Task 各一个提交（T1、T18、T19 另有跟进提交 `483ca9d`、`e070065`、`f9d1ab2`），T16 之后有修复轮的 10 个提交，整分支评审之后有修复轮的 4 个代码提交和一个文档提交（3.16）；每个提交结束时 `check:types` 23、`make lint-web` 52（T10 起 54）、`make test-web` 16、`make build-web` 11 个任务通过，`make knip` 为零。
- [ ] `make lint-web`：`keywords: 51 rules, 3 exceptions, no hits.`；上限合计 694（T19 的跟进之前 695）；`alts.mjs` 为 0；`missquote.mjs`、`missreal.mjs` 为 0。
- [ ] `make test`、`make lint-go` 通过（T16）；`t16/names.mjs` 为 0。
- [ ] vitest 101 个测试（editor 35），没有 stderr；T17 的新用例在旧代码上失败。
- [ ] 每个 Task 的孤儿核对（基点是上一个 Task 的提交）：`symref`、`dangling`、`keyref orphaned` 为 0，`headers.sh` 没有输出，`deadorph` 为 0（T7 为 `parseHTML` 一行；修复轮的 FW4 为 `url` 一行，`9a98148` 为 `content`、`text` 两行，见 3.2 结论 3），`infile-orphans` 的每一行都以 `defined now: 0)` 结尾（T3、T4、T5 为 12、166、4 行，其余为 0）。
- [ ] `deadsym.mjs`：导出 2、成员 898、prop 452；`keyref.mjs unused` 0，每种语言 1079 个键（T11–T18 为 1077）；`assets.mjs` `0 of 134`（T12、T13 为 `0 of 113`，T14 为 `0 of 142`）；`t15/assetpaths.mjs` `0 of 134`；`labels-all.mjs` `dangling 0, repeat 0`；`rulecount.mjs . react/jsx-curly-brace-presence` 0。
- [ ] 锁文件与 3.5 相同；`devwarn.mjs` 为 0；构建体积与 3.4 的最后一列相同（内容哈希除外）。
- [ ] `t18/prove.sh`：冷、热构建之后都没有事先放进去的文件，两次的 531 个文件逐字节相同；在旧的 `Makefile` 上热构建留下它（T18）。
- [ ] 控制者的浏览器核对全部通过（含 T14、T15 的第 12、13 项，T17 的第 14 项和 T19 的第 15 项），反向对照在基线上失败的正好是收尾修掉的几项（第 14 项只有 C3；第 4 轮的数字见 3.14）；S1–S4、持续集成通过。
- [ ] 前端改动清单 1.6 节 30 行（T13 的 13 行，T14、T15 各一行，修复轮 7 行，T17–T20 各一行，由 T20 之后的文档提交加，整分支评审之后的修复轮 4 行，由它的文档提交加）；README 与 T13 相同，只有写根目录格式检查的两句由 FW17 改为指向根目录 `package.json` 的 `fix:format`；24 份交接与 T13 相同，M5 的收尾交接另有 T15 加的两节，M4 的另有修复轮加的两节和 `comment_json` 一条，M5、M6 的各有 T20 之后的文档提交加的一节，M2–M8 的数目随修复轮和 T19 的跟进更新；P2 写给 M2 的交接（`M1-P2-trim-content.md`）引用的主题注释和取值随 T4 改为 `THEME_OPTIONS`（整分支评审之后的文档提交）。
- [ ] 新画的和改过的图片按能看清 20 px 细节的尺寸看过（T14、T15、T20），哈希与 plan 相同。

---

## 6. 不在收尾范围内

- 死成员和死 prop（第 4 节第 4 条，交 M2–M8）；`viewId as TProfileViews`（交 M4）。
- 评审没有点名、收尾的 Task 也没有改到的两类退化结构（`degen-all.mjs`，收尾结束时）：197 个没有插值的模板字面量（组件 127 个、`helpers/` 32 个、`constants` 18 个，例如 `` `/api/users/api-tokens/` ``）和 159 个只有一个子元素的片段（T19 的跟进 `f9d1ab2` 收掉了活动迭代卡片分组行的一个，之前是 160）。裁定是"在改到的行里收掉"；oxlint 没有能看住它们的规则（`jsx-curly-brace-presence` 只管 JSX 属性和子元素里的，那 80 处 T6 已收掉），一次性扫掉也会再长出来，所以按裁定由改到它们的 M 收掉。
- oxlint 的 694 个警告（3.3 的计划）。
- 附件图标里的第三方标志（2.4 D4，第 9 节第 6 条：交 M5）；新建项目时写进封面值的构建路径（交 M5，8.1）。
- 包的版本号和 `license` 字段（交 M8）；`@makeplane/propel` 的对应源码（P5 交 M8）。
- AGPL 第 5(a)、5(d)、13 条的验收和 jsDelivr 的第三方请求（交 M8，2.9 的 Important 3）；没有经过 `t()` 的英文界面文字（交 M8，2.10 的 I2）；个人设置里主题下拉框的位置（交 M2，2.10 的 I1）。
- 复制资源的接口核对请求者能否读取源资源（交 M5，2.9 的 Critical 1）；迭代侧边栏和模块的进度口径（交 M6，3.17）。

---

## 7. 风险

| 风险 | 应对 |
|---|---|
| 批量删除（T4 的 166 个文件、T11 的 500 个键、T12 的 133 张图、T15 的 8 张图）删掉了保留功能还在用的东西，或者留下了没人显示的东西 | 类型检查、构建、knip 发现缺失的导入；`keyref.mjs orphaned`、`symref.mjs orphaned`、`deadorph.mjs`、`infile-orphans.mjs` 每个 Task 核对；图片按文件名查整个 `web/`，T15 起再按路径查（`assetpaths.mjs`，同名的图不再互相遮住），导入了却传给没人读的值的由 `uses.mjs` 查；控制者的探测检查页面上原样显示的键和加载失败的图片，并重跑 P2–P5 的场景 |
| 类型检查器的"没人读"有误报（经框架、第三方库或展开读取） | 导出只算"别的文件读取"，经桶文件追到真正的读取方；成员和 prop 不在收尾删；T7 的 `parseHTML` 登记为已知的例外，由测试证明被读取 |
| `DOMParser` 取代 `sanitize-html` 后"是否为空"的判断或预览的文字变了，评论或草稿的提交行为改变 | 5 个单元测试覆盖不间断空格、`<br>`、允许的标签和实体；控制者的探测在评论和草稿弹窗里核对，与基线对照。已知的行为变化有两处（第 4 节第 11 条）：一是实体被解码，预览不再显示 `&amp;`（修正）；二是正文里 `<script>`、`<style>`、`<textarea>`、`<option>` 中的文字现在算作文字（预览显示它，"是否为空"为否），`sanitize-html` 原来丢掉它们。编辑器的 schema 没有这些节点，所以对编辑器的 HTML 答案不变。`DOMParser` 解析出的文档是惰性的，不执行脚本、不加载图片 |
| `window.open` 带 `noopener` 后调用方拿不到新窗口 | 12 处都不读返回值（逐处看过）；探测核对新页面打开、`window.opener` 为 `null` |
| 删掉 `window.close()` 后菜单不再关闭 | base-ui 在选中菜单项时自己关闭菜单；探测在溢出菜单里选一项，核对菜单关闭、标签页没有被关 |
| 补丁在 `pnpm install` 时没有生效或随升级失效 | 锁文件带补丁的哈希；T9 之后测试输出没有警告即证明生效；升级到新版本时 pnpm 会因补丁对不上而报错 |
| 守卫为了通过而放宽 | 新规则都有命中样本证明有效；例外只减不增；不命中样本引用的代码由 `missquote.mjs` 核对存在，由 `missreal.mjs` 核对它在规则读取的文件里（样本自己所在的 JSON 行不算） |
| 一个值有两个来源（主题标签）改成一个时，某处读的仍是被删的字段 | `key` 删除后类型检查找到了设置页的第二处读取（`value.key`），两处一起改；探测核对设置页和 Power K 的主题名 |
| 封面的扩展名改为 `.webp` 后，已保存的封面值或上传失效（T14） | 预设文件名只有辅助函数的导入表读取；已保存的值是上传副本的地址；探测核对选择器、新建项目的上传，以及项目卡片、项目设置、个人资料显示的预设（plan 浏览器核对第 12 项，32 项）。新建项目时先写构建路径的行为是 Plane 原有的，交 M5（8.1） |
| 修图改到了头像和名字以外的像素，或者留下了照片的碎片（T15） | 修图只在列出的圆和名字的框里画；`look.mjs` 按 20 px 细节的尺寸出对照图，每张看过；哈希写在 plan；5 px 的残边写明保留的理由（第 4 节第 25 条） |
| 删掉活动迭代的 `activeCycleResolvedPath` 属性后空状态变了（T15） | 内层组件只把它解构成 `_activeCycleResolvedPath`，从来不读；类型检查证明没有别的传入方；探测在没有活动迭代的项目里核对 "No active cycle" 和 propel 的插图，且不请求活动迭代的图片（plan 浏览器核对第 13 项） |
| 惰性解析之后，编辑器之间复制图片不再复制资源，或者粘贴丢了格式（T17） | 解析之后的 HTML 与原来一样交给 `pasteHTML`，只是不再在活动文档里解析；编辑器测试核对已上传的图片被标为复制；探测核对同编辑器复制粘贴时请求复制资源一次、图片指向新资源（plan 浏览器核对第 14 项的 C1） |
| `rm -rf` 删掉了构建之外的东西，或者 Turbo 的缓存恢复不完整（T18） | 只删 `web/apps/web/build/client`，那是构建的输出目录；`prove.sh` 核对冷、热两次构建的文件清单逐字节相同（531 个）；门禁照常构建 |
| 修图盖住了控件旁边的内容，或者留下看得出的方块（T20） | 每个框的前后对照放大 4 倍看过；12 张截图按原尺寸重新看过；哈希写在 plan；`pictures-probe.mjs` 核对页面上的字节与文件相同 |

---

## 8. 移交事项

### 8.1 交给后续 M（T13 写进对应 M 的 `handoffs/M1-closeout.md`；M5 的前两条由 T15 写进，M4 的后两条由修复轮写进，M5 的第三条和 M6 的一条由 T20 之后的文档提交写进，M2 的一条和 M8 的后两条由控制者在收尾 review 的提交里写进）

| M | 事项 | 关闭条件 |
|---|---|---|
| M2–M7 | 本 M 领域里的死成员和死 prop（第 4 节第 4 条的数目）；oxlint 的清理（3.3 的计划） | 本 M 合并时 `domains.mjs … --rows M<n>` 列出的每一行都已消失或写进本 M 的 review；review 写明改到的文件的警告数为 0、清掉的规则和上限的变化 |
| M2 | 另有个人设置里主题的下拉框画在页面左上角：ui 的 `CustomSelect` 的定位，Plane 原有（2.10 的 I1） | 本 M 的浏览器核对写明：个人设置的主题下拉框在它的按钮旁展开（zh-CN 和 en） |
| M4 | 另有 `viewId as TProfileViews` | `git grep -n "as TProfileViews" -- web` 没有输出 |
| M4 | 另有工作项弹窗的描述不读编辑器的 `isMigrationUpdate`（`issue-modal/components/description-editor.tsx`；`description-input/root.tsx` 读它），旧描述补块 ID 时表单被标为已改（修复轮写进交接） | `git grep -n isMigrationUpdate -- web/apps/web/core/components/issues/issue-modal` 有输出，测试或浏览器核对写明不改动时表单没有被标为已改 |
| M4 | 另有标注块的 tiptap-markdown 序列化把 `data-emoji-url` 等属性不转义地拼进 HTML（`callout/extension-config.ts`）；`getMarkdown()` 没有调用方、`transformCopiedText` 为 `false`，各扩展的节点序列化看起来都是死代码，删还是转义由 M4 决定（修复轮写进交接） | `git grep -n "markdown: {" -- web/packages/editor/src` 没有输出，或者测试证明写出的属性经过转义 |
| M5 | 另有新建项目时的封面值：`projects/create/root.tsx` 的 `onSubmit` 先上传预设封面的副本，再创建项目（`POST …/projects/` 的 `cover_image_url` 仍是构建里预设封面的地址 `/assets/image_<n>-<hash>.webp`，每次构建都变），然后登记副本（`POST …/bulk/`）、用副本的地址覆盖（`PATCH …/projects/<id>/`）。上传失败时不创建项目；项目建好之后的两步有一步失败，项目保存的就是下次构建就失效的地址。这是 Plane 原有的行为，T14 的浏览器核对记下了这几个请求的顺序（3.15） | 本 M 的封面接口只保存上传后的资源（或预设的编号），不保存构建路径；新建项目只写一次封面值 |
| M5 | 另有附件图标里的第三方标志：`app/assets/attachment/` 的 `figma-icon.png`、`pdf-icon.png`、`csv-icon.png`、`excel-icon.png`（2.4 D4，第 9 节第 6 条） | `app/assets/attachment/` 里没有第三方的标志 |
| M5 | 另有复制资源的权限：粘贴编辑器自己的剪贴板类型时，`src` 是资源 id 的每个 `image-component` 都让应用请求 `POST …/duplicate-assets/{asset_id}/`（`asset-duplication.ts`、`file.service.ts` 的 `duplicateAsset`）；任何网页都能在这个类型里写别人的资源 id。Plane 原有，T17 的评审确认不是收尾引入的（2.9 的 Critical 1） | 本 M 的复制接口先核对请求者能读取源资源（它的工作区和项目），读不到时返回 404（或本项目对无权资源的约定），有测试 |
| M6 | 另有进度的口径：迭代侧边栏的数字是完成 / 总数（取消的算在总数里），卡片和迭代列表（`calculateCycleProgress`）不算取消的；模块的进度在列表项和排序里是 (完成 + 取消) / 总数，卡片里是完成 / 总数；`calculateCycleProgress` 只要快照不为空就读快照，不看迭代的状态（3.17，2.9 的 Important 2） | 每种进度（迭代、模块）由一个函数计算，每处显示都用它；口径（取消的算不算）由本 M 定，写进 review |
| M8 | 本 M 领域（Webhook）的 6 个死成员；共享部分的 287 个；oxlint 在发布前清零、警告改为错误、删除上限；13 个 `package.json` 的 `"license": "AGPL-3.0"` 改为 `AGPL-3.0-only` | 发布之前 `--rows shared` 为空或写进 review；oxlint 没有警告；`git grep -n '"license": "AGPL-3.0"' -- '*package.json'` 没有输出 |
| M8 | 另有 AGPL 的三项义务：第 5(a) 条（修改说明和日期）、5(d) 条（交互界面的法律声明，含原版界面的例外）、13 条（网络用户取得运行版本的对应源码）；jsDelivr 的两类请求（表情数据 `emojibase-data@latest`、标注块的默认表情图片）（2.9 的 Important 3） | 任何对外的网络部署之前（最迟 M8 发布）：三条逐项验收写进 review，条款的解读由负责人定；页面上的源码入口指向运行的提交和完整的对应源码；jsDelivr 的请求改为自托管固定版本，或明确告知并接受，写进 review |
| M8 | 另有界面文字的中文覆盖：没有经过 `t()` 的英文字面量约 457 处（2.10 的 I2） | 每个 M 把改到的界面文字接入 `t()`；M8 发布之前 zh-CN 界面上没有未翻译的界面文字，核对的方法写进 review |

P2–P5 写给 M2–M8 的 17 份交接各补一节"关闭条件"（T13）。

### 8.2 交给控制者

- 收尾 review（`reviews/closeout-review.md`）：复述 3.3、3.4，写进浏览器核对的脚本和输出；M1 设计 12 节收尾一行、11 节的勾、总体设计 9.4 和 M1 的状态（第 9 节第 3 条）。
- 收尾 review 的提交另外改：8.1 里 M2 的一条和 M8 的后两条写进对应的收尾交接；M1 设计 3.1 的写法（2.9 的 Minor 2）；Codex 评审报告的"处理结果"一节（2.9）。
- T20 之后的文档提交：前端改动清单 1.6 节加 T17–T20 各一行；本 spec、plan 和 M2–M8 的收尾交接里随 T17–T20 实际提交变的数字；8.1 里 M5 的第三条和 M6 的一条写进对应的收尾交接。
- 整分支评审之后的修复轮（3.16）：zh-CN 与英文相同的 5 个值翻译（2.10 的 I2，FW14）；`services` 的 `helpers/index.ts`（2.9 的 Minor 1，第 9 节第 7 条确认，FW15）；整分支评审的 Minor（R2、R3 是 FW16、FW17，其余是文档）。
- 第 9 节的裁定。

---

## 9. 待裁定

1. **封面照片和导览截图的来源（升级）。** 这是许可和来源的问题，收尾没有删也没有换任何一张（任务书的要求），查到的事实和建议如下。
   - **封面**：`app/assets/cover-images/image_1.jpg`–`image_29.jpg`，由 `helpers/cover-image.helper.ts` 导入，是项目封面的预设图片（Plane 的提交说明写着新建项目时随机选一张）。它们由 Plane 的提交 `36d4285`（2025-12-03，PR #8184 "[WEB-5493] feat: implement static cover image handling and selection"）加入；提交说明和仓库里都没有写来源、作者或许可。文件宽 1280 px，只带 ICC 色彩配置，没有 EXIF、XMP、IPTC（`jpgmeta.mjs`）。在那之前 Plane 用 16 张 Unsplash 照片的地址作预设封面（v1.1.0 的 `PROJECT_UNSPLASH_COVERS`）；把 29 张与这 16 张逐张做感知哈希比较（`provenance/phash.mjs`），没有一张相同：距离最近的三对（`image_5`、`image_6`、`image_21`，距离 6–9）打开看是完全不同的照片（摩天楼仰拍对红色日落，绿色多边形对红色日落，模糊的晚霞对黑底的岩石）。结论：来源和许可查不到；看起来是图库照片，Plane 的 AGPL 许可证管不到第三方照片的权利。
   - **导览截图**：`onboarding/cycles.webp`、`modules.webp`、`views.webp`（`onboarding/tour/root.tsx`）是 Plane 的界面截图，界面属于 Plane 的代码（AGPL），但里面有真人的头像照片和一位 Plane 联合创始人的名字，是第三方的肖像和个人信息。
   - **建议**：封面换成 Nerve 自己生成的 29 张抽象图（渐变或几何纹理，由脚本从 SVG 渲染，登记在同目录的 `SOURCES.md`，文件名不变，代码不改），不删功能；导览截图里的头像换成字母头像、人名换成中性的名字（P5 修复轮改 "Plane Design" 的同一种做法）。裁定之后作为收尾的追加 Task（在 T12 之后），我可以先做原型。另一种做法是删掉预设封面（只剩上传，封面上传归 M5），不推荐：它删掉了一个保留的功能。
   - **原型之后的更正**：封面按实测改为 SVG 源文件加 WebP，扩展名和导入随之改（第 4 节第 19、20 条），不再是"文件名不变，代码不改"；导览截图里有头像的是 `issues.webp`，`modules.webp` 没有（2.4 D2、第 4 节第 24 条）。
2. **`handler_test.go` 的夹具名（跨模块）。** `server/internal/platform/webui/handler_test.go` 的夹具叫 `manifest.json`、`favicon/android-192.png`，注释说"对应 `web/apps/web/build/client` 的布局"，名字却是编的（P5 评审第 7 节）；`handler.go` 不依赖具体文件名，所以测试照常通过。改的是 Go 服务的测试，不在 web 之内。建议：收尾追加一个只改测试的提交，夹具改为构建真实产出的 `site.webmanifest.json`、`icons/icon-192x192.png`（注释因此成立）；或者交给下一个改 `webui` 的 M（M2 的认证会动静态资源的服务）。
3. **M1 的状态何时改为"已完成"。** 设计 9 节说收尾时改；任务书说 Codex 对整个 M1 的对抗评审在收尾合并之后进行。建议按设计在收尾 review 的提交里改（总体设计 9.4、M1 设计 12 节、11 节的勾），Codex 评审的发现按 M1 的修复轮处理并记在 M1 设计 12 节；如果控制者希望以 Codex 评审为 M1 的终点，就把状态留到那时改，11 节第 9 项在收尾 review 里写"待 Codex 评审"。
4. **死成员和死 prop 的分法（请确认第 4 节第 4 条）。** 按领域交 M2–M8、共享部分交 M8，每份交接带列出的命令和关闭条件。如果控制者要在 M1 内删，建议另开一个 Phase，按包分 Task（propel、ui、types、utils 等共享包先做），每处写明读取方的核对。
5. **删除的图片按 300×200 看是否足够（请确认第 4 节第 14 条）。**
6. **附件图标里的第三方标志（T15 复看时发现）。** `app/assets/attachment/` 的 17 个文件类型图标里，4 个用了第三方的标志（2.4 D4）。附件归 M5；收尾换掉它们要为 17 个图标定一套新的样子，超出"去掉真人和人名"的范围。已裁定：交 M5，关闭条件是 `app/assets/attachment/` 里没有第三方的标志（8.1），由 T15 写进 M5 的交接。
7. **Codex Minor 1 的去向（已确认，见下面的裁定）。** 批准的分诊写的是"T4 已解决（桶文件已不存在）"，依据是本 spec B12 原来的一格；核对之后不成立：`ensureAPITrailingSlash` 仍经 `services` 的两个 `export *` 桶文件公开（2.2 B12、2.9）。建议放进整分支评审之后的修复轮：`helpers/index.ts` 改为只按名字转出 `normalizeAPIRequestURL`，一行（第 4 节第 31 条）。另一种做法是交给重写 `services` 的 M（M2 或 M5），Codex 认为那样 M2 可能把它当作稳定的接口。

**控制者的裁定**（详见 plan 的"控制者评审补充"和"T14–T16 原型之后的裁定"）：第 1 条替换，封面为 T14、导览截图为 T15，保留的其余图片全部按能看清 20 px 细节的尺寸复看一遍；第 2 条收尾做，T16；第 3 条以收尾合并为 M1 的终点，已经启动的 Codex 评审的发现分诊进收尾，状态在收尾 review 的提交里改；第 4、5 条采纳；第 6 条交 M5（2.4 D4、8.1）。T14–T16 的原型被接受；8.1 交 M5 的两条（新建项目时的封面值、附件图标的标志）已确认；T15 的两处补充采纳（删掉活动迭代没人读的属性和它的两张图；删掉 T12 按文件名查时漏掉的 6 张同名图，T12 不改）。Codex 对抗评审的分诊由负责人批准（2.9）：Critical 1、Important 1、2、4 成为 T17–T20，原型采纳，只有 T20 的 `intake-light.webp` 一框在实现时改为重复左边一列（原型的纯色抹掉了卡片的底边，3.17）；Important 3 交 M8、Minor 2 改 M1 设计 3.1，都在收尾 review 的提交里；第 7 条（Minor 1）按建议放进整分支评审之后的修复轮（FW15 `df4cbf7`）。浏览器核对顺带发现的两件事交 M2、M8，5 个未翻译的值在整分支评审之后的修复轮里翻译（FW14 `e171d64`，2.10）。
