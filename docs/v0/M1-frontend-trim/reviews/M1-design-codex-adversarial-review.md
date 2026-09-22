# M1 设计方案对抗性评审

## 1. 评审信息与验证边界

- **对象**：`worktree-m1-design`，HEAD `02fd956e865378a637b5c1bca5fb08844e47a298`；主要设计提交 `0e6671c`，同步修改提交 `02fd956`。日期：2026-09-22。
- **路径基准**：下文代码及文档的完整相对路径以 `/Users/xiaoruan/project/nerve-project/.claude/worktrees/m1-design` 为根，简称 W；局部表格另注明省略的共同前缀，同段连续引用省略共同目录。D 表示 `docs/v0/M1-frontend-trim/M1-design.md`，V 表示 `docs/v0/v0-design.md`，F 表示 `docs/v0/frontend-changes.md`。`plane/` 证据来自只读 `/Users/xiaoruan/project/nerve-project/plane`，HEAD `02c19e1341d93141e8ad7b3278298adce208bafc`。
- **材料**：完整 M1 设计及同步提交、总体设计、前端改动清单、差异清单、M0 设计 5.2/6、两份 M1 handoff、实际 web 源码和相关 Plane 后端路由。三份清点只作为查找线索；未读取 M1 `specs/`、`plans/` 草稿。
- **环境**：macOS 26.6.2 / Darwin 25.6.0 arm64；Node 24.15.0；Corepack 调用 pnpm 11.10.0；Apple Git 2.50.1；锁定的 React Router 8.3.0、TypeScript 5.8.3、Turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0。
- **术语**：“实测”包括实际执行搜索、类型检查和路由匹配原型；不把它们称为浏览器端到端验证。“读代码推断”说明调用关系及可能后果，不表示已经实施 M1 后观察到了缺陷。

### 1.1 实际执行及结果

所有安装、构建、类型生成、试删都在仓库外。主实验副本为 `/var/folders/9y/fk6s7m2j2rd9yzr_zb1388l40000gn/T/nerve-m1-design-review-uz6gk19n/copy`，下称 R；对应日志在 R 的父目录。副本用 `rsync -a` 排除 `.git`、`node_modules`、缓存、构建物及 M1 草稿目录，再补回被排除规则覆盖的跟踪文件。原工作区 4264 个跟踪文件已做 SHA-256 快照。

| 命令 / 实验 | 关键结果与边界 |
|---|---|
| W：`git status --porcelain=v1`、`git branch --show-current`、`git rev-parse HEAD`、`git show 02fd956` | 起始干净；分支/提交符合任务。同步提交改了 5 份文档，没有实现改动 |
| R：`corepack pnpm install --frozen-lockfile` | exit 0；1266 个包，6.8 秒；未全局安装 |
| R：`TURBO_FORCE=true make lint-web` | exit 0；49/49 任务成功、0 cached，31.256 秒，包含 13 个实际类型检查任务 |
| R：各有 lint 脚本的工作区运行 `corepack pnpm exec oxlint --format json .` | 995 warnings / 0 errors；逐包与设计上限完全相同，api-client/e2e 均为 0 |
| R：`corepack pnpm exec knip --no-exit-code`，再以 `--reporter json` 汇总 | files 131、dependencies 19、devDependencies 4、unlisted 2、exports 135、types 83、enumMembers 4、duplicates 1；此副本的生成状态下有 2 条配置提示。并未声称这是干净克隆提示数 |
| R：`TURBO_FORCE=true make build-web` | exit 0；11/11 任务成功、0 cached，14.676 秒；client 1239 文件 / 30.739 MiB，保留设计所列两种构建警告 |
| R：`corepack pnpm --filter web exec react-router typegen`；仅移除 knip 的 web `ignoreUnresolved` 后再检查 | 未产生 unresolved；剩余两条提示是 i18n 生成键和 tailwind 错误入口。支持 7.2 的 typegen 方案，不等于已证明未来 P3 knip 全零 |
| R：按实际外部调用保留 services 的 13 个文件，删另外 49 个、修三个 barrel；运行 services build、services/web check:types | 均 exit 0；62−49=13 复现。保留 APIService、APITokenService、URL helper、文件 metadata/payload helper 及所需配置/测试 |
| R：上述修剪后 `corepack pnpm --filter @plane/services test` | 已有 URL 工具测试 13/13 通过，76 ms；不是新写的镜像测试 |
| `/tmp/m1-router-review-typecheck`：原声明 tsc；仅改成真实 RR `useParams` 签名后同命令 tsc | 基线 0；替换后 exit 2，**152 errors / 68 files**（TS2345 29、TS2322 54、TS18048 69）。命令 `node /Users/xiaoruan/project/nerve-project/.claude/worktrees/m1-design/web/apps/web/node_modules/typescript/bin/tsc -p /tmp/m1-router-review-typecheck/tsconfig.json --noEmit --pretty false`；日志 `/tmp/m1-router-review-baseline.log`、`/tmp/m1-router-review-real-params.log` |
| `node /tmp/m1-router-review-typecheck/route-match.cjs` | 真实 RR `matchRoutes` + 真实 coreRoutes：删 legacy 路由后，两个仍有入口的 URL 落到 not-found。见 C1；不是浏览器点击测试 |
| `/tmp/m1-router-review-i18n`：将 NODE_ENV 整个判断换成 `import.meta.env.DEV` 后 tsc | 命令 `node /Users/xiaoruan/project/nerve-project/.claude/worktrees/m1-design/web/packages/i18n/node_modules/typescript/bin/tsc -p /tmp/m1-router-review-i18n/tsconfig.json --noEmit --pretty false`；TS2339：当前包没有 ImportMeta.env 类型；日志 `/tmp/m1-router-review-i18n.log`。这是 P4 类型门禁可发现的小项 |
| `python3 /tmp/m1-keyword-audit.py` | 从 7.4 提取 55 个内容正则；Apple Git `-P` 全部可编译；同模式交给文件名 `grep -iE`，两个 lookbehind 正则 exit 2。完整输出 `/tmp/m1-keyword-audit.json` |
| 跟踪源码计数、路由/导航/翻译消费链扫描；npm 0.3.0 元数据只读核查 | 数字和功能边界见下表及第 4–6 节。未仅依据清点中的“无引用”作结论 |

补充证据：`/tmp/m1-router-review-route-match.log`、`/tmp/m1-coupling-inventory.json`；R 父目录中的 `install.log`、`lint.log`、`build.log`、`oxlint-counts.json`、`knip.json`、`knip-with-typegen.log`、`services-delete49.json`、`services-prune-types.log`、`services-prune-web-types.log`、`services-existing-tests.log`。这些是本地复核材料，关键结果已写入本报告，不把临时文件当永久验收依据。类型原型先在/tmp执行typegen，使用既有安装的锁定依赖只读链接，并在副本共同关闭composite/incremental，避免写共享tsbuildinfo；对比只改useParams声明。

### 1.2 关键事实核查

| 设计论据 | 核查结论 |
|---|---|
| 4065 个 web 跟踪文件、2336 个 app 文件；21×28 语言 JSON；2926 版权文件；1575 文件含 `@plane/` | **实测成立**。19×28=532 个语言 JSON 可删除。4128 次文本出现不用于估算修改工作量 |
| 保留功能不消费协作编辑、Yjs、`description_binary` | **读代码成立，需限定到功能消费**。描述历史类型仍声明 binary，但历史 UI 用 HTML；Pages 是实时事件/binary 的消费者。不能据此删公共 editor/ref 的其他方法或 UniqueID 本地 ID 功能 |
| 工作项嵌入、页面提及可删 | WorkItemEmbed 在文档编辑器扩展中；CE additional mention hook 为空。用户 @提及仍有完整消费者，必须保留 |
| AuthenticationWrapper 8 次渲染期跳转 | **源码核实成立**：`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx:95` 起的 8 个调用（95、98、106、111、119、124、134、138）；SET_PASSWORD 分支到 P3 会减少两个，不应把 8 当 P4 固定数量 |
| 真实参数类型产生 152 个错误 / 68 文件 | **精确复现**；删除功能后应重计。路由模块用生成 params、共享组件守卫的方向合理 |
| 两处强制尾斜杠、约 12 处比较 | 两机制成立；“约 12”只能算清点估计，不能当完整接受清单。另有 includes/RegExp，见 I6 |
| services 49 个无业务调用方文件 | **试删并通过类型检查**。这里不是删整个包，也不是删剩余 APIService/URL/file helpers；详见上表 |
| 七个企业 i18n 命名空间和 applications 无调用 | 在实际键内容、整键/前缀及动态调用位置中未找到这些企业键的消费者；方向成立。`tour.json` 是企业产品 tour 文案，不能由此删账户 onboarding 或首页现有 TourRoot。applications 数字应是 **161 叶键**，不是 176 |
| applications 删除不会伤 PAT/Webhook/API 日志 | **成立**。PAT 用 `account_settings.api_tokens.*` / `workspace_settings.settings.api_tokens.*`；Webhook 用 `workspace_settings.settings.webhooks.*`；当前没有接口调用日志页面，它属于 M8 新交付。不能扩大删除到 developer 分类或整个 workspace-settings JSON |
| isbot、@react-router/node 保留 | **安装源码核实成立**：RR 8.3 默认 `entry.server.node.tsx` 导入两者；SPA 构建仍使用它做预渲染；typegen 实现会在缺 isbot 时改 package.json 加回它。不能把“应用源码未直接 import”当无用依赖 |
| @makeplane/propel 0.3.0 为 AGPL-3.0-only | 安装包 package.json 与 [npm 0.3.0 元数据](https://registry.npmjs.org/@makeplane%2fpropel/0.3.0)一致；六个工作区依赖它。源码并入退路还需要确定与该包对应的源码版本，npm 元数据没有 gitHead，不能把“随时复制 main”当已验证替代方案 |
| oxlint 995、knip 八类基线 | **实测成立**。1005→995 的三份 M0 文档算术更正正确。提示条数受生成状态影响，不应当固定业务基线 |
| 旧路由 13、构建 1226 文件/30.5 MiB | 当前 HEAD 注册的是 **11 条 legacy redirect**；本次构建 1239 文件/30.739 MiB。计数应写口径，不影响删减方向 |

### 1.3 未验证部分

没有运行 Docker、`make test`、`make e2e` 或 S1–S4，也没有启动/停止任何容器。没有实施 P1–P5 全部方案，没有伪称将来的每个 Phase 已通过；没有用浏览器跑完整业务页面或认证流程。没有 Linux 主机实测正则，也没有重新跑远程 CI、全量依赖漏洞审计。59 个“上游不存在接口”未逐个重做路由解析，只核对本报告涉及的认证、工作项历史、个人主页、评论和邀请端点；剩余数量应继续作为待核清点，不作为批量删除授权。缓存新脚本尚不存在，本次只评估依赖/输入设计，未声称完成其缓存正确性实测。许可证元数据核查不等于完成发布所需的完整源码/告知义务核查。

## 2. 总体结论

**修改后开工。** 五阶段方向可用，多数基础清点经代码或原型复核成立，不需要推倒重做。确定的阻断是 P1 删除旧重定向会损坏保留的个人设置入口；此外，首页最近访问、多选禁用链、路由尾斜杠和实例字段的保留边界还不够明确，静态门禁与当前一次性核对清单无法覆盖共享编辑器、进度和认证提交的退化。建议保留 P1–P5，将这些边界和可执行验收先补进设计，随后按功能依赖链实施。下列为 **1 Critical、6 Important、2 Minor、1 项建议**；未将每个实现细节或已有遗留问题都升级为设计缺陷。

## 3. 新发现

### C1 — Critical：P1 删除旧地址时漏修现用入口，个人设置会落入 404

**现象 / 证据（已复现）**：D:160 只列两处 `/sign-in` 内链修正，D:307 把全部 legacy redirect 放在 P1。实际 PowerK 注册的 `nav_account_settings`（`web/apps/web/core/components/power-k/config/navigation/root.ts:49`、`commands.ts:145`）生成 `/:workspaceSlug/settings/account`，靠 `web/apps/web/app/routes/core.ts:404` 跳到 `/settings/profile/general`。实际侧栏还从 `web/packages/constants/src/workspace.ts:212` 生成 `/analytics/`，经 `sidebar-menu-items.tsx:156` 或现用 extended-sidebar 渲染，靠 core.ts:379 跳到 overview；该功能到 P2 才删除。

真实配置的 RR 匹配实验输出：

```text
/acme/settings/account   before workspace-account-settings.tsx   after ./not-found.tsx
/acme/settings/account/  before workspace-account-settings.tsx   after ./not-found.tsx
/acme/analytics/         before analytics.tsx                    after ./not-found.tsx
```

命令与日志见 1.1。这里没有拿未被挂载的旧 `workspace-menu.tsx` 当证据。现用链可由 `app/(all)/[workspaceSlug]/(projects)/layout.tsx:21` → `_sidebar.tsx:69` → `sidebar.tsx:36` → `sidebar-menu-items.tsx:156` → `sidebar-item.tsx:66` 复核。

**影响**：保留的个人资料/PAT 等设置访问入口损坏，P1 中仍存在的数据分析入口也损坏；类型、knip、S2 的 HTML/资产检查都不会发现字符串目标不存在。

**方案**：① **推荐**把旧路由清理归 P4，与全部内部目标及导航一起改；提前随 P2 删除已下线功能的 redirect。② 维持 P1，但先完整列出 11 条旧路由的活跃调用方，将账户命令改正式终点、分析菜单改 overview，然后同一提交删 redirect。后者可行，但 P1 增加跨功能改动。**开工前必须改责任及验收清单；不得只改“13”为“11”。**

### I1 — Important：运行时验收漏掉最易退化的保留行为，且 `/api/**` 无法覆盖认证提交

**现象 / 证据（读代码推断）**：D:358 的一次性脚本只覆盖 P2 个人主页/侧栏、P3 登录注册/启动、P4 路由。以下关键改动没有明确断言：

- 公共编辑器 ref、扩展、保存 HTML/JSON：`web/packages/editor/src/helpers/editor-ref.ts:70`、`hooks/use-editor.ts:69`；保留的历史复制 Markdown 调用在 `web/apps/web/core/components/core/description-versions/modal.tsx:67`。
- 数量进度与点数在同一工具中：`web/packages/utils/src/distribution-update.ts:108`、`:206`；cycle/module store 捕获失败后只 warn（`web/apps/web/core/store/issue/cycle/issue.store.ts:159`、`module/issue.store.ts:107`），类型通过不代表进度更新正确。
- P3 删 Epic/TEAM 后，`web/apps/web/core/hooks/store/use-issues.ts:90`、`use-issue-layout-store.ts:21` 仍需为项目、迭代、模块、个人主页、归档选择正确 store。
- CSRF 在 `web/apps/web/core/services/auth.service.ts:19` 请求 `/auth/get-csrf-token/`；`core/components/account/auth-forms/password.tsx:138` 原生 POST `/auth/sign-in/` 或 `/auth/sign-up/`。它们不在 `/api/**` 中，显示表单不能验证单步邮箱密码改写后的 payload。
- S2 (`e2e/stories/smoke/s2-web-app.spec.ts:54`) 只检查文档/静态资产/同源/深链；其他故事检查 Go/DB/instance/problem。四个故事共五个测试，不经过这些保留业务路径。

另一个具体遗漏：D:290 把无人调用的 `test` 脚本整体清掉，但 `web/packages/services/src/helpers/url.test.ts:10` 是保留 URL helper 的已有测试，本次 **13/13 实测通过**。留测试文件却删掉运行入口，knip 仍可能因 Vitest 插件将它视为 entry，不等于“已清掉无用任务”。

**影响**：会出现静态验收全绿、登录无法提交、最近访问/描述编辑/历史恢复/进度不正确的情况；把错误拖到 M2–M7 才发现，难以定位是哪次删除造成。

**方案**：① **推荐**补第 7.3 节的最小行为矩阵；提交少量不依赖旧 API 的纯工具/编辑器/路由测试和运行入口，现有 URL 测试保留到 helper 被真正替换。对只能通过旧 API 进入的页面，用同一份临时 harness 同时拦截 `**/api/**` 与 `**/auth/**`，按 URL.pathname 精确分派认证端点，把脚本、最小 fixture、运行命令及断言放进 Phase review 可复现附录；不做完整 Plane 假后端。② 全部用一次性脚本也可以，但必须保存可重跑内容并跨后续 Phase 重跑，不能只留“已核对”的结果。前者维护成本更低、能长期守住稳定行为；后者免新增测试入口，但复核较依赖人工。**开工前补验收责任；P1 加入口，P2/P3/P4 随改动执行。**

### I2 — Important：首页“改好”没有锁定最近访问的保留入口

**现象 / 证据（读代码推断）**：D:68、:316 只规定 Pages/便签删除后改首页。V:58 明确保留最近访问；但 `web/apps/web/core/components/home/root.tsx:27` 通过待删 HomeStore 取 widgets，`:60` 渲染 DashboardWidgets。后者 `home-dashboard-widgets.tsx:39` 是 `RecentActivityWidget` 唯一实际渲染入口，`:69` 依赖待删的 widgetsMap/orderedWidgets。`widgets/recents/index.tsx:48` 的最近访问请求本身可以独立保留。

**影响**：若顺着删除 widgets 的编译/knip 报错清理，最近访问可能一起变成死代码，被合法“清零”。设计没有明确要求误删，但唯一入口不能靠实施者猜测。

**方案**：① **推荐**明确首页固定保留问候、无项目空态、项目/工作项最近访问及需要的 peek，直接挂载普通最近访问组件；删 HomeStore、管理面板和个性化请求。② 先为最近访问指定新入口，再删首页原入口；可减少首页内容，但增加产品变化。**开工前定最终首页内容；P2 实施并测有数据/空态，M7 换新 API。**

### I3 — Important：多选是否保留不能交给 knip；当前 CE 是有引用的整条禁用链

**现象 / 证据（读代码推断）**：D:143 用“还有保留功能在用，由类型检查和 knip 核实”决定多选去留。实际 `web/apps/web/core/hooks/use-bulk-operation-status.ts:7` 恒返回 false；列表 `core/components/issues/issue-layouts/list/default.tsx:134` 和表格 `spreadsheet/spreadsheet-view.tsx:82` 将其变成 disabled，却继续向各层传 selectionHelpers。批量操作 root (`core/components/issues/bulk-operations/root.tsx:30`) 只是升级横幅。删横幅后，链的类型和引用仍然成立，不会被 knip 判为无用。

`use-multiple-select.ts:61` 的离开警告、`:324` 的 active 行键盘导航也在这条受禁用条件控制的链中；不能把它们当已经对 CE 用户提供的保留能力。通知预览/全部已读使用独立 `WorkspaceNotificationStore`，其他表单多选下拉也不属于此链。

**影响**：可能留下 v0 不使用的 store、复选框透传和禁用分支，违背彻底删除目标；也可能误伤名字相似但真实使用的通知/表单交互。

**方案**：① **推荐**3.9 直接裁定删除工作项 MultipleSelectStore、useMultipleSelect、组头/行选择及 selectionHelpers 透传，和批量操作同任务；保留通知及普通多选下拉。② 若确实需要独立键盘行导航，先定义保留交互，再只抽出必要逻辑；会新增行为范围，不宜由“存在引用”自动决定。**开工前明确裁定，P3 完成整链并清 root 构造/退出重置。**

### I4 — Important：关键词验收混用正则引擎，现有模式有保留命中和漏查范围

**现象 / 证据（已复现 + 代码核查）**：D:252 规定内容用 Git PCRE、文件名用 `grep -iE '<同一模式>'`。实测 55 个内容模式在本机可编译，但文件名的 `(?i)(?<!view_)time_?line` 和 `(?<![dD])epic|Epic|EPIC` 都返回 `grep: repetition-operator operand invalid`、exit 2。不是“零命中”。Linux 未实测；ERE 本身不提供 lookbehind，不能用操作系统名称保证支持。[Git 官方说明](https://git-scm.com/docs/git-grep)也明确 `-P` 是可选编译能力。

明确的保留命中：`TExtended` 命中设计自己要求保留的 `web/apps/web/core/components/workspace/sidebar/extended-sidebar-item.tsx:32`；`TAdditional` 命中保留筛选的 `core/components/work-item-filters/filters-hoc/base.tsx:22`。只说“extended-sidebar 不在模式中”并不能排除其类型名。设计允许 spec 收窄，故这不是要求删这些保留代码，但不能把现表直接当可运行门禁。

漏查/模糊点：只搜 web 会漏根 `pnpm-lock.yaml`、catalog/overrides/globalEnv 中的旧依赖/变量；内容关键词不能识别纯二进制 Logo；`TEAM_*` 不覆盖裸 TEAM 分支；“按文件允许后续 Phase 残留”不能精确处理 root store、共享 types、翻译 JSON 这类跨 Phase 共用文件。阶段清单的一行还混了 P2 更新日志和 P3 Unsplash/遥测。

**影响**：误把正则执行失败当清零，或不断增加人工排除、为了零命中无意义改名；还可能放过实际未删依赖及共享字段。

**方案**：① **推荐**在 P1 固定一个小检查器/清单格式：内容和文件名各自定义合法模式；检测 exit 0/1/>1；精确例外记录规则 ID、路径+符号/匹配文本、理由和到期 Phase/M，根清单与二进制品牌另查。共用文件允许的是特定残留符号，不能整文件豁免。② 保持人工命令，但每个 Phase 交完整命令、原始输出及逐命中责任表；成本高，容易漏掉失败码。**开工前改接受规则，P1 做原型；具体删除由所属 Phase 完成。**

### I5 — Important：实例保留字段和跨里程碑字段交接仍有缺口

**现象 / 证据（读代码推断）**：D:137 只点名 `enable_signup`；实际 `web/packages/types/src/instance/base.ts:47` 还含有保留功能消费的 `is_workspace_creation_disabled` 和 `file_size_limit`。前者用于 `app/(all)/create-workspace/page.tsx:41`、onboarding、工作区菜单和 PowerK；后者用于 `core/hooks/use-file-size.ts:21`。D:152 的 M2 handoff 未给这两项明确去向。不能把“删对应功能开关”误实现为 config 只剩 enable_signup。

3.11 也没有贯穿差异清单：`docs/v0/plane-diff.md:66`、`:69` 已要求去掉 `is_bot/bot_type/user_type`；前端仍有成员过滤、评论/通知/链接署名及 PAT 类型消费者。M5 的 `file_assets.page_id`（差异清单:86）、M3 的项目 estimate/type/time-tracking 字段、M6 各种 `*_estimate_points`/快照分布也不能仅凭几个示例词完成闭环。

**影响**：实例删多会影响创建工作区/上传限制；删少则后续生成新类型时反复返工，甚至重新引入产品明确不区分真人和 Agent 的旧约定。这里是遗漏/歧义，不是已发生误删。

**方案**：① **推荐**将 3.7 写成删除/保留/后续重定义三列表；3.11 声明当前是种子清单，并要求从本 Phase 实际 diff 导出完整字段、枚举、调用地址、默认行为清单，逐项对应 M2–M8 和差异清单。② 本 M 立即统一重定义所有实例及账户/文件字段；会越过不做后端/新接口对接的边界，不推荐。**开工前定实例边界与交接格式；P2/P3 产出，具体新契约仍由对应 M 决策。**

### I6 — Important：删掉强制追加斜杠不等于地址统一无斜杠，匹配清点还漏了非等式判断

**现象 / 证据（部分表达式已复现）**：D:172 说删两机制后地址统一不带 `/`，D:186 主要列直接比较。实际内部 Link/push 目标仍带 `/`，例如 `web/apps/web/core/components/navigation/top-navigation-root.tsx:67`；机械 href→to 不会删它。`core/hooks/use-workspace-paths.ts:20` 用 `includes('/'+workspaceSlug+'/')`，`top-navigation-root.tsx:76` 用 `includes('/notifications/')`；项目/工作区设置的 `item-categories.tsx:60` / `:54` 还有要求尾 `/` 的正则。

在路由原型中，`/acme` 与 `/acme/`、`/acme/notifications` 与带斜杠形式得到不同旧匹配结果；RR `matchPath` 可以同时识别两种形式。D 已要求两种形式侧栏/设置都正确，方向对，但 app rail、顶部通知及内部 URL 生成未入清单。

**影响**：页面可以打开但当前导航项不亮；“统一无斜杠”的验收结论无法由既定机械迁移推出。

**方案**：① **推荐**明确内部生成的 UI URL 用无尾斜杠格式，匹配同时接受有/无斜杠；按语义枚举 equality/includes/startsWith/RegExp，补 rail/顶部通知/历史 back/query/hash 验收，继续用 RR 原生匹配。② 只取消自动追加，允许内部目标混用两种形式；改文档承诺并同样修匹配，工作量略低但 URL 风格不一致。不要混同 Django API 的 URL helper，后者到领域对接时才退出。**开工前选定语义，P4 实施。**

### M1 — Minor：同步文档仍有互相冲突的完成条件和几个计数口径

**证据**：F:83 仍写“TypeScript、oxlint、knip 全部为零”，但 V:562/:644、D:232/:249 明确 M1 用警告基线、M8 前清零。同步提交修了 CSRF 和 995，却漏了这处接受条件。D:160 的 13 legacy routes、:223 的 176 applications 键与实测不符；D:213 的 `pnpm -r ls --depth -1` 实际还列出根包 `nerve`，应允许这个预期名字。构建文件数见 1.2，宜记口径而非硬门禁。

**影响**：不同审查人可能使用不同完成条件；不构成需要重做设计的技术问题。

**方案**：① **推荐**统一以上数字/例外和 oxlint 完成条件。② 删除没有后续用途的精确计数，保留可重算命令和量级，维护更轻。**随设计修订；最晚 P1 前统一验收条件。**

### M2 — Minor：“编辑器包只改一次”与 P2/P3 边界矛盾

**证据（读代码推断）**：D:83 将 AI 留 P3 却称 editor 只改一次；D:315 的 P2 已改公共内核。`web/packages/editor/src/types/editor.ts:112`、`:194` 分别含协作和 AI 参数，rich-text wrapper、extensions 也跨两次修改。不是两个 Phase 一定无法编译，而是低估审查次数。

**方案**：① **推荐**把 editor 相关 AI/EE 和协作收敛放同一 P2 末尾任务，应用级 AI 实例字段可 P3 处理，逐符号登记临时残留；若为避免半删功能，也可把整条 AI 功能前移 P2。② 维持分期，删除“只改一次”承诺，两次使用同一保留编辑器验收。**开工前决定任务边界即可，不必合并整个 P2/P3。**

### S1 — 建议：为包来源、版权和体积留下小而可核验的记录

**证据 / 影响**：D:148 的“随时并入源码”没有锁对应 source ref；npm 指定版本记录没有 gitHead。D:198 只写保留旧版权，没约定新增 Nerve 文件/矢量图形的出处。当前构建 30.739 MiB 是总磁盘字节，不代表首次加载成本。它们不阻止本地瘦身，但到发布时再补来源和体积比较容易漏项。

**方案**：① **推荐**P5/收尾记录 propel 0.3.0 的 tarball integrity、许可证和对应源码定位，保留实际可用下载或仓库引用；改写文件保留上游告知，新文件/新 SVG 按实际来源加 Nerve 的版权/SPDX，不能只是给 Plane Logo 改名；无合适头部的资源在同目录来源说明登记。收尾记录 JS/CSS/字体/其他资源字节、最大 chunk、语言 chunk 数，与相同命令的基线比较，不设拍脑袋的体积 KPI。② 直接 vendor 整个设计系统；离线上游依赖更少，但大幅扩大范围，M1 不推荐。M8 再完成发布的完整源码/网络使用告知核查；本报告不作“元数据相同即全部合规”的结论。[GNU 文件告知建议](https://www.gnu.org/licenses/gpl-howto.en.html)可作为记录方式参考。

## 4. 闭环核对表

“成立”表示有明确落点且边界合理，**不表示尚未实施的代码已经验收通过**。“有问题”对应第 3 节。先按上级要求逐项对照，不能用一个总括“2.3 全链删除”替代具体 Phase。

### 4.1 总体设计 1.2

依据 `docs/v0/v0-design.md:63-72`；删除深度统一由 M1:91-96 覆盖。

| 1.2项 | M1处置与证据 | Phase | 闭环 |
|---|---|---|---|
| 文档页 Pages | 2.1:61；3.1:104-106；搜索/收藏/资源字段3.11:154-157 | P2 | 成立 |
| 实时协作 apps/live | live未迁入(v0:516)，editor Yjs/Hocuspocus清除M1:61,105-106 | P2 | 成立 |
| 估算 | M1:62,121；进度按数量 | P2 | 成立 |
| 甘特图 | M1:63；保留关联常量先搬出 | P2 | 成立 |
| 时间线（含模块） | M1:63,258；保留relation hook改名 | P2 | 成立 |
| 自动关闭 | M1:64,153；自动归档保留 | P2 | 成立 |
| 数据分析 | M1:65,122；迭代/模块progress保留 | P2 | 有问题：P1仍用旧重定向（C1） |
| 导出 | M1:66；连同未创建集成服务 | P2 | 成立 |
| 便签 | M1:67 | P2 | 成立 |
| 首页快捷链接 | M1:68 | P2 | 成立 |
| 首页个性化 | M1:68；无入口Dashboard一起删 | P2 | 有问题：须锁定最近访问入口（I2） |
| 侧栏自定义导航 | M1:69,116-118 | P2首项 | 成立；保留项目/本地偏好 |
| 自定义主题 | M1:68；亮暗/系统/高对比保留 | P2 | 成立 |
| 公开发布 apps/space | 未迁入app；前端弹窗链接/anchor删M1:78,139-140 | P3 | 成立 |
| 实例管理 apps/admin | 未迁入app；入口/not-ready/is_setup_done删M1:79,135-137 | P3 | 有问题：实例保留字段需补（I5） |
| OAuth | M1:80,128-133 | P3 | 成立 |
| 验证码登录 | 同上 | P3 | 成立 |
| 找回密码 | 同上，另覆盖重置/设置密码/邮箱检查 | P3 | 成立 |
| 所有邮件发送 | M1:81-82；营销同意、改邮箱；认证验证码 | P3 | 成立；邀请链接见5.1/7.5 M3 |
| 邮件通知偏好 | M1:81；站内通知保留 | P3 | 成立 |
| AI | M1:83；编辑器菜单 | P3 | 成立 |
| Unsplash | M1:84；上传保留 | P3 | 成立 |
| 遥测 | M1:85；配置/文案残留 | P3 | 成立 |
| 更新日志 | M1:72；借用TPage | P2 | 成立；7.4将其与P3关键词放同一行，应按实际P2标记 |
| 项目邀请 | M1:86；改为直接加工作区成员 | P3 | 成立；joinProject旧地址例外到M3(M1:273) |
| Epic | M1:87；issue服务分支 | P3 | 成立 |
| 团队 | M1:87 | P3 | 成立 |
| 工作项类型 | M1:87 | P3 | 成立 |
| 升级计费提示 | M1:87；最后删；保留地址名单208 | P3 | 成立 |
| 企业extended空壳 | M1:87,274；extended-sidebar界面排除 | P3 | 成立；不能按extended字面量整树删 |
| Plane死代码 | M1:88,286-298 | P1无调用方services/工具；P2集成；P3其他 | 成立 |

### 4.2 前端改动清单第二节

| 原清单行 | 项 | Phase / M1证据 | 评估 |
|---|---|---|---|
| 54 | Pages/协作 | P2，61,104-106 | 成立 |
| 55 | Estimates | P2，62,120-122 | 成立 |
| 56 | Gantt/timeline | P2，63 | 成立 |
| 57 | 自动关闭 | P2，64 | 成立；自动归档保留 |
| 58 | analytics | P2，65,122 | 成立 |
| 59 | 导出 | P2，66 | 成立 |
| 60 | 便签 | P2，67 | 成立 |
| 61 | 首页/主题 | P2，68 | 有问题：首页保留入口（I2）；基础主题边界成立 |
| 62 | 自定义导航 | P2，69,116-118 | 成立 |
| 63 | AI | P3，83 | 成立 |
| 64 | Unsplash | P3，84 | 成立 |
| 65 | 公开发布 | P3，78,139-140 | 成立 |
| 66 | god-mode | P3，79,135-137 | 成立 |
| 67 | 个人统计/动态 | P2，70,108-114 | 成立；三工作项标签保留 |
| 68 | 项目邀请 | P3，86 | 成立；改名、直接添加 |
| 69 | OAuth/magic/找回/重置/设置/检查邮箱 | P3，80,128-133 | 成立；CSRF移M2与上级一致 |
| 70 | 修改登录邮箱 | P3，82,162-164 | 成立；旧邮件流程删，替代由M2决定 |
| 71 | 旧路由重定向 | P1，159-160,307 | 有问题（C1）；数量应11 |
| 72 | 邮件通知偏好 | P3，81 | 成立 |
| 73 | 企业残留/活跃迭代 | P2活跃迭代71；P3其他87 | 成立；跨Phase明确 |
| 74 | IndexedDB/集成/无效service | P1 services49个126；P2集成66；P3余项88 | 成立 |
| 75 | 两语言 | P1，218-223 | 成立 |
| 76 | Next垫片 | P1死image/script171；P4其余168-190 | 有问题：I6路由语义；参数原型成立 |
| 77 | 部署遗留 | P1，288 | 成立 |
| 78 | serve/start/preview | P1，288 | 成立 |
| 79 | sw/workbox | P1，288 | 成立 |
| 80 | .env.example | P4，173 | 成立 |

验收文字一致性：frontend-changes:83仍写 oxlint全部为零，M1:31,244-249是M8前清零、M1重测基线；上级v0:562支持后者。应同步改清单验收文字，不能只修改功能状态。

### 4.3 前端改动清单第四节

| 原行 | 项 | Phase / M1证据 | 结论 |
|---|---|---|---|
| 128 | Logo/favicon | P5，199,210 | 成立；简单SVG决策可在P5具体化 |
| 129 | Plane标题/文案 | P5，200-205 | 成立；外链语义替换 |
| 130 | @plane→@nerve | P5，211-213 | 覆盖；@makeplane例外见3.10 |
| 131 | 版权保留 | 持续约束，198 | 成立；不能计入品牌残留 |

### 4.4 两份 M0 交接

| 来源行 | 事项 | Phase / M1证据 | 结论 |
|---|---|---|---|
| P5:12 | Dockerfiles/caddy/.dockerignore | P1，288 | 成立 |
| P5:12 | serve/start/preview崩溃 | P1，288 | 成立 |
| P5:12 | SW/workbox/maps | P1，288 | 成立 |
| P5:13 | .env.example整删 | P4，173 | 成立 |
| P5:13 | dotenv/define审视 | P4，173-175 | 明确删除及顺序 |
| P5:14 | Express覆盖项 | P1，289 | 指定保留RR间接有效项 |
| P5:14 | turbo admin/space/live环境 | P4，173 | 9.7以.env一行概括，不是P1漏项 |
| P5:15 | workspace无效注释 | P1，289 | 成立 |
| P5:16 | catalog/overrides/allowBuilds锁文件 | P1/P2/P3+收尾，321,344 | 成立 |
| P5:17 | turbo无调用任务 | P1，290 | 成立 |
| P5:21 | 警告重测、自动检查 | P1 7.1；收尾7.3；346 | 成立 |
| P5:25 | React#418 | P1，297,311,359 | 成立 |
| P5:26 | 结尾斜线 | P4，172,186,348 | 成立 |
| P5:27 | tailwind module | P1，295,311 | 成立 |
| P5:27 | tsconfigPaths | P1，295,311 | 成立 |
| P5:28 | web-dev包监视/并发 | P1，298 | 成立 |
| P6:12 | knip门禁/去no-exit-code | P3，325,327 | 成立 |
| P6:13 | config hints错误/生成状态 | P3 7.2；P1入口修复295；349 | 成立；typegen 原型成立 |
| P6:14 | typegen/i18n ignoreUnresolved | i18n P1:221；web P3:242 | 成立 |
| P6:15 | 不要删ignoreUnresolved注释 | P3:242先确定生成状态再删 | 有合理替代，非盲删 |
| P6:19 | api-client/e2e上限0 | P1，350、7.1 | 成立 |
| P6:23 | 锁文件变化仅必要项 | P1/2/3，312,321,344 | 成立 |
| P6:27 | S2不锁文字 | P5，214,351 | 成立 |
| P6:28 | knip workspace路径不随包名 | P5，213,351 | 成立 |
| P6:32 | .env删除沿用P5 | P4，173,345 | 成立 |
| P6:33 | S2同源断言保留 | 第10节357保留S2；P4说明174 | 成立 |
| P6:33 | README两处.env说明 | P4，330,345 | 成立 |


上述 P5/P6 行号分别指 `docs/v0/M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md`、`M0-P6-knip-notes.md`。C1 所涉重定向行判为**有问题**；首页入口（I2）、实例保留字段（I5）和现用关键词接受办法（I4）也有问题。其他“覆盖/成立”只评价已有责任映射，不抵消这些专项发现。

### 4.5 总体设计 7.3–7.6、8、9 的 M1 要求

| 上级要求 | 设计落点 | 判断 |
|---|---|---|
| 7.3 保留邮箱密码注册/登录/退出/改密；删多余认证界面 | P3 3.6；M2 替换传输 | 成立；P3 要覆盖 onboarding 设置密码分支，不能只删 URL；认证提交核对缺口见 I1 |
| 7.4 1.2 功能、EE 空壳、死代码按各层删除 | P1/P2/P3，2.3 | 成立；模板/工时等“有引用的空壳”补明列，见第 5 节 |
| 7.4 Next 垫片 | P1 删死文件，P4 原生路由 | 有问题：C1、I6；迁移主方向成立 |
| 7.4 自动化仅自动归档 | P2 | 成立；保留 archive_in/archived_at 与归档入口 |
| 7.4 类型/knip/关键词/构建 | P3 knip 门禁，收尾全清 | 有问题：I4；各 Phase 可用增量报告，不能 P1 就要求全部 knip 零 |
| 7.5 品牌（包括内部包名） | P5 | 成立；第三方名/版权精确例外，根包 nerve 也是预期名字 |
| 7.5 只保留中英文 | P1 两语言+回退；P2/P3删功能文案；收尾死键 | 成立；双语言同删错误不能靠一致性检查发现 |
| 7.6 不留死实现/兼容层、门禁持续生效 | P1 lint 精确值；P3 knip；P4 删垫片 | 有问题：I3 禁用链需人工辨认；自动检查可行性见 7.4 |
| 7.6 新代码职责与 MobX 模式 | 0.2 不做新接口转换；后续 M 按领域改 | 成立；不建议 M1 重写状态架构 |
| 8 单元测试/故事/CI，9.3旧故事仍通过 | 每个 Phase 全门禁及 S1–S4 | 责任成立；本次未运行 Docker 故未重验，新增行为补证见 I1 |
| 9.2 M1 静态完成条件、警告重测及清零计划 | 第7/10/11节 | 成立；F:83需要同步，M1不要求995立刻归零 |

## 5. 灰色地带功能清单与入口全集

先给容易误判的对象结论，再列完整路由与导航。归类中的“1.1”表示该入口承载明确保留的业务能力；首页、app rail、PowerK交互容器本身列为灰色，不声称上级逐字定义了这些产品。判断以现用挂载/消费者为准，未挂载旧菜单不作为保留依据。

### 5.1 灰色对象的明确处置

| 灰色对象 | 来源/为什么灰色 | 设计实际处置 | 是否还需裁定 |
|---|---|---|---|
| 工作区首页容器 | 1.2删首页个性化，不是整条首页URL | P2重写正文，68 | 有问题：I2，先定最近访问的保留入口 |
| 个人主页根地址 | 1.2个人统计删；1.1跨项目工作项保留 | 3.2转assigned+三标签 | 已定；空/无权限状态需验收 |
| 个人主页右侧卡片 | 旧数据来自统计接口，资料本身保留 | 3.2成员store，头像/名字/joining_date，删时区/统计 | 已定；保留信息并非原样全部保留 |
| 访客个人主页 | 原只有统计标签，统计删后何处落脚 | 卡片+无权提示，最终权限M3 | 已分配负责人 |
| 侧栏自定义导航 | 未原列清单，但对应表已砍 | 3.3新增明确删除 | 已定且上级已同步 |
| 项目导航偏好 | 名称也带navigation preferences，但表不同 | 3.3保留workspace_user_properties | 已定 |
| app rail本地显示偏好 | 与自定义导航同文件但本地数据 | 3.3明确不受影响 | 已定 |
| sidebar折叠/宽度/分组展开 | 个性化广义词可误伤，实际布局交互 | 3.3仅删远端导航固定/排序/隐藏；本地布局未要求删 | 按保留交互处理；不扩张“首页个性化” |
| extended-sidebar组件 | extended命名并不都是EE空壳 | 7.4:274明确保留真实界面 | 已定 |
| 迭代/模块analytics | 1.1进度/燃尽，1.2分析 | 3.4改progress保数量，service旧URL暂例外 | 已定 |
| 项目当前迭代 | 名称active cycle像企业推广 | 2.1:71明确只删工作区推广页 | 已定 |
| 自动化设置 | 同页自动关闭与自动归档 | 2.1:64只留归档 | 已定 |
| 基础主题/高对比 | 自定义主题删除可误伤基础theme | 2.1:68保留亮暗/系统/高对比 | 已定 |
| 邮件偏好vs站内通知 | 都叫notifications | 2.2:81只删邮件偏好；独立路由站内留 | 已定 |
| 注册/登录模式与检查邮箱 | 旧模式由email-check后响应决定 | 3.6改按URL模式，邮箱密码直接表单 | 已定 |
| CSRF/session | 1.2认证删减但属于保留认证传输 | 3.6延后M2 | 已定且上级同步 |
| 新手引导中的设置密码/营销同意 | 新手引导1.1保留但子步骤属删除项 | 2.2:80-81,3.6包含设置密码/营销 | 已定；P3不能只删独立URL |
| 修改密码 | 名称包含password，容易与reset/set一起误删 | 1.1保留；3.6保留传输到M2 | 已定 |
| 修改登录邮箱 | 1.1个人资料范围内，旧流程依赖已删邮件 | 3.13旧流程删，M2决定无邮件替代 | 已定且责任明确 |
| 实例未设置页 | 页面错误态，但来自已删后台配置状态 | 3.7删is_setup_done与NotReady | 已定 |
| 实例请求失败Maintenance | 与not-ready相邻，但属于启动失败展示 | 没有要求删；第10节仍认启动错误页存在 | 应保留失败状态，不把两者一并删 |
| is_workspace_creation_disabled | 保留创建工作区有消费者，非OAuth邮件开关 | 3.7未单列 | I5需明确保留/替换/删除决策 |
| file_size_limit | 保留上传有消费者 | 3.7未单列 | I5需明确保留到M2/M5 |
| 评论access | 一般权限词，但仅内外公开范围 | 3.8删UI/type，M4决定列 | 已定 |
| 多选 | 批量操作删，但其他选择逻辑可能保留 | 3.9按剩余消费者决定 | 有问题：I3，CE整链禁用但有引用，不能由knip裁判 |
| PAT及developer分类 | 可能与企业applications一起误删 | 3.5保留APITokenService，上级1.1明确PAT | 已定；翻译专项证明分离 |
| Webhook及developer分类 | 可能与integration一起误删 | 7.4:275明确保留 | 已定；M8对接 |
| API调用日志 | v0保留，但迁入前端只有付费宣传语 | M8新增；M1删billing宣传 | 无须M1造UI；建议设计写明事实 |
| applications.*文案 | 文案涉及OAuth应用、webhook secret，貌似开放能力 | 第6节按死文案P1删 | 已定；161叶键实测无业务引用 |
| 工作区邀请列表 | “所有邮件发送”可被误读为所有邀请删 | 1.1系统内接受明确留 | 已定，M3对接 |
| token工作区邀请链接/复制invite_link | 有活入口，系统内接受和链接接受不是同一流程 | M1未单列 | 见7.5，需明确交M3 |
| 项目“邀请”弹窗 | 实际从工作区成员直接添加 | 2.2:86只改名字文案 | 已定 |
| joinProject /projects/invitations/旧接口 | 字面含invitation但作用是加入项目 | 7.4:273至M3例外 | 已定 |
| Plane旧URL重定向 | 历史兼容可删，但仍有现用内链 | 3.12 P1全删+两sign-in改写 | C1不足，先改全部现用入口 |
| 工作区保留slug名单 | 不是可见导航，但需匹配一级路由 | 3.11分Phase删对应条目，M3前后端一致 | 已定；别过度释放仍用词 |
| @makeplane/propel | 名字Plane但为现用第三方设计系统 | 3.10保留，品牌验收例外 | 已定 |
| Plane帮助/论坛/源码外链 | 不是业务功能，但关系品牌替换 | 第5节删Plane帮助链接，需要外链改Nerve仓库 | 已定 |
| is_bot/user_type | 整体产品不区分人/Agent，但保留页面读旧字段 | 上级plane-diff已定删除，3.11未列具体责任 | 建议后续M显式分派，不自动扩大M1 |


指定非 route 对象的补充核对：

| 对象 | 1.1/1.2归类与代码现状 | 设计当前默认处置 | 具体证据/需补点 |
|---|---|---|---|
| 工作项模板 | 1.1未列；没有模板管理/创建入口和可用实现，存在CE空壳与表单分支 | template命名空间P1明确删；剩余模板代码按2.2企业空壳/死代码在P3顺带删，但文档未点名 | `web/apps/web/core/components/issues/issue-modal/provider.tsx:17,37-40,52`固定null/false和no-op；`web/apps/web/core/components/issues/issue-modal/form.tsx:164-168,201-209,248-253`实际引用模板状态。`web/packages/i18n/src/locales/en/template.json:2-17`为templates.*；TS/TSX无templates.*消费者。应在P3清点明列 workItemTemplateId/isApplyingTemplate/handleTemplateChange/context字段，这些有引用，不能期待knip自动查出无效行为 |
| 定时/重复工作项 | 1.1未列；未找到route、设置/PowerK项或服务/类型实现；所谓recurring的TS命中是计费周期 | 无实现可保留；文案按无引用键收尾删除（D:338）；没有明确独立Phase | `web/packages/i18n/src/locales/en/work-item.json:337-414` recurring_work_items.*、empty-state.json:243；TS/TSX无该前缀消费者。`web/packages/utils/src/subscription.ts:80-102`/payment类型的recurring是月年计费，P3随billing删。应把recurring_work_items明确加入死文案清单，避免“无实现”被当作“不需要清理” |
| 工时记录/Worklogs | 1.1未列；没有记录工时UI/路由/API调用，但有共享类型、动态渲染和资源注册 | 企业计费宣传P3顺带删；类型/动态/插图残留设计只有笼统死代码覆盖，没有逐项命名 | `web/packages/types/src/issues/activity/base.ts:80-84` WORKLOG联合成员；`web/apps/web/core/components/common/activity/helper.tsx:70,278-280` is_time_tracking_enabled分支；`web/packages/propel/src/empty-state/assets/asset-registry.tsx:35,82`和asset-types.ts:28注册worklog。`docs/v0/plane-diff.md:73`明确删除项目is_time_tracking_enabled。建议P3显式删联合分支、渲染分支、无用插图/键；7.4现无worklog/time_tracking模式，且这类残留不一定触发knip |
| app rail整体 | 灰色：1.1未列独立app rail产品；是项目/设置导航容器，有现用入口 | 3.3明确保留其本地偏好；默认保留rail本身，P5换Plane图标 | `web/apps/web/core/components/workspace/content-wrapper.tsx:28`真实挂载；`web/apps/web/core/components/navigation/app-rail-root.tsx:50-60`项目和设置入口，66-80图标显示及Dock切换；`web/apps/web/core/components/navigation/app-rail-hoc.tsx:24-31`仅Projects项指向工作区首页。不要随侧栏远端偏好删除整个rail |
| Tour引导代码 | 1.1“新手引导”可覆盖保留基础引导；当前TourRoot是活代码，和tour.json不是同一使用链 | 默认保留基础Tour；P2随Pages删pages步骤/图片/前后跳转，P5替换硬编码品牌。第6节仅删无引用tour翻译命名空间 | `web/apps/web/core/components/home/root.tsx:16,50-52`挂TourRoot；`web/apps/web/core/components/onboarding/tour/root.tsx:29,39-79`步骤含pages；71由views转pages，需重接结束；99-106硬编码品牌。`web/packages/i18n/src/locales/en/tour.json:2`是product_tour.*，全仓代码搜索该前缀无消费者。**删tour.json不等于删TourRoot，也不会自动清掉活Tour里的Pages介绍** |
| PowerK整体 | 搜索在1.1明确保留（v0:60）；创建/导航/偏好/快捷键是保留业务的交互容器，未在1.1逐条列出 | 默认保留PowerK与有用命令，P2/P3删除被砍功能命令，P4改导航接口 | `web/apps/web/app/(all)/[workspaceSlug]/(projects)/layout.tsx:17`挂ProjectsAppPowerKProvider；`web/apps/web/core/components/power-k/projects-app-provider.tsx:48`聚合命令；`web/apps/web/core/components/power-k/config/commands.ts:17-33`聚合七组；5.4 的72个静态命令ID。不能以整组PowerK“企业版”删除；nav_home归灰色首页，已改表 |
| 首页/nav_home | 1.1未明列首页，recent等保留信息可置于首页；1.2删快捷链接与个性化 | M1:68明确重写首页正文，默认保留首页入口；最终保留块未在设计细化 | `web/apps/web/app/routes/core.ts:63`、`web/apps/web/core/components/power-k/config/navigation/commands.ts:109-119`；`home/root.tsx:56-60`现peek/greeting/widgets容器。列为灰色已定保留容器，不声称1.1明列 |
| workspace token invitation | 1.1明列系统内工作区邀请；token链接接受方式未明列。现有真实页面及复制链接入口 | 设计未点名删除，按剩余代码会默认保留到M3；应显式裁定/交接是否与系统内列表并存 | `web/apps/web/app/routes/core.ts:43`；`web/apps/web/app/(all)/workspace-invitations/page.tsx:40-59,71-77` query token接受/拒绝；`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx:78-98`复制invite_link。Plane serializer:122与邮件任务:32都产链接，说明它不单纯是无入口死页面，也不只是发邮件组件 |


### 5.2 core.ts 全部 68 个 URL 入口

证据统一为 `web/apps/web/app/routes/core.ts:<行>`，行号指 route/index声明。layout本身不增加URL；按其子路由一起保留/删空layout。真实参数类型原型见 1.1。

| 行 | URL | 归类 | 设计处置 |
|---|---|---|---|
| 16 | `/` | 1.1账户认证 | 保留；P3清邮件/设置密码/OAuth等；M2对接 |
| 19 | `sign-up` | 1.1账户认证 | 保留；P3清邮件/设置密码/OAuth等；M2对接 |
| 23 | `accounts/forgot-password` | 1.2认证删 | P3删除，3.6 |
| 26 | `accounts/reset-password` | 1.2认证删 | P3删除，3.6 |
| 29 | `accounts/set-password` | 1.2认证删 | P3删除，3.6 |
| 33 | `create-workspace` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 36 | `onboarding` | 1.1账户认证 | 保留；P3清邮件/设置密码/OAuth等；M2对接 |
| 39 | `invitations` | 1.1工作区邀请 | 保留系统内接受，M3对接 |
| 43 | `workspace-invitations` | 灰色：token邀请链接 | 未独立裁定；系统内邀请保留，M3需明确链接接受方式 |
| 63 | `:workspaceSlug` | 灰色：首页容器 | P2重写正文；1.2只砍快捷链接/个性化/便签，不删首页 |
| 67 | `:workspaceSlug/active-cycles` | 1.2企业推广删 | P2删；项目当前迭代保留，2.1 |
| 72 | `:workspaceSlug/analytics/:tabId` | 1.2删除 | P2删除，2.1及2.3 |
| 77 | `:workspaceSlug/browse/:workItem` | 1.1工作项/归档/搜索 | 保留；M4对接 |
| 82 | `:workspaceSlug/drafts` | 1.1草稿 | 保留；M4对接 |
| 87 | `:workspaceSlug/notifications` | 1.1站内通知 | 保留；M7对接 |
| 92 | `:workspaceSlug/profile/:userId` | 灰色→保留入口改写 | P2重定向assigned，3.2；访客权限M3 |
| 93 | `:workspaceSlug/profile/:userId/:profileViewId` | 1.1跨项目工作项 | 保留assigned/created/subscribed，P2去统计卡片 |
| 97 | `:workspaceSlug/profile/:userId/activity` | 灰色→已定删 | P2个人动态删，2.1/3.2 |
| 105 | `:workspaceSlug/stickies` | 1.2删除 | P2删除，2.1及2.3 |
| 110 | `:workspaceSlug/workspace-views` | 1.1保存视图 | 保留；M7对接 |
| 111 | `:workspaceSlug/workspace-views/:globalViewId` | 1.1保存视图 | 保留；M7对接 |
| 119 | `:workspaceSlug/projects/archives` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 131 | `:workspaceSlug/projects` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 138 | `:workspaceSlug/projects/:projectId/issues` | 1.1工作项/归档/搜索 | 保留；M4对接 |
| 144 | `:workspaceSlug/projects/:projectId/issues/:issueId` | 1.1工作项/归档/搜索 | 保留；M4对接 |
| 151 | `:workspaceSlug/projects/:projectId/cycles/:cycleId` | 1.1迭代/模块/归档 | 保留，P2进度去点数、模块去timeline；M6对接 |
| 159 | `:workspaceSlug/projects/:projectId/cycles` | 1.1迭代/模块/归档 | 保留，P2进度去点数、模块去timeline；M6对接 |
| 167 | `:workspaceSlug/projects/:projectId/modules/:moduleId` | 1.1迭代/模块/归档 | 保留，P2进度去点数、模块去timeline；M6对接 |
| 175 | `:workspaceSlug/projects/:projectId/modules` | 1.1迭代/模块/归档 | 保留，P2进度去点数、模块去timeline；M6对接 |
| 183 | `:workspaceSlug/projects/:projectId/views/:viewId` | 1.1保存视图 | 保留；M7对接 |
| 191 | `:workspaceSlug/projects/:projectId/views` | 1.1保存视图 | 保留；M7对接 |
| 199 | `:workspaceSlug/projects/:projectId/pages/:pageId` | 1.2删除 | P2删除，2.1及2.3 |
| 207 | `:workspaceSlug/projects/:projectId/pages` | 1.2删除 | P2删除，2.1及2.3 |
| 214 | `:workspaceSlug/projects/:projectId/intake` | 1.1需求收集箱 | 保留；M7对接 |
| 224 | `:workspaceSlug/projects/:projectId/archives/issues` | 1.1工作项/归档/搜索 | 保留；M4对接 |
| 232 | `:workspaceSlug/projects/:projectId/archives/issues/:archivedIssueId` | 1.1工作项/归档/搜索 | 保留；M4对接 |
| 240 | `:workspaceSlug/projects/:projectId/archives/cycles` | 1.1迭代/模块/归档 | 保留，P2进度去点数、模块去timeline；M6对接 |
| 248 | `:workspaceSlug/projects/:projectId/archives/modules` | 1.1迭代/模块/归档 | 保留，P2进度去点数、模块去timeline；M6对接 |
| 264 | `:workspaceSlug/settings` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 265 | `:workspaceSlug/settings/members` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 269 | `:workspaceSlug/settings/billing` | 1.2企业计费删 | P3删除，2.2 |
| 273 | `:workspaceSlug/settings/exports` | 1.2删除 | P2删除，2.1及2.3 |
| 277 | `:workspaceSlug/settings/webhooks` | 1.1开放能力 | M1保留，M8新API；勿误删developer分类 |
| 281 | `:workspaceSlug/settings/webhooks/:webhookId` | 1.1开放能力 | M1保留，M8新API；勿误删developer分类 |
| 293 | `:workspaceSlug/settings/projects` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 296 | `:workspaceSlug/settings/projects/:projectId` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 301 | `:workspaceSlug/settings/projects/:projectId/members` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 306 | `:workspaceSlug/settings/projects/:projectId/features/cycles` | 1.1迭代/模块/归档 | 保留项目功能开关，由M3对接；领域业务另属M6/M7 |
| 310 | `:workspaceSlug/settings/projects/:projectId/features/modules` | 1.1迭代/模块/归档 | 保留项目功能开关，由M3对接；领域业务另属M6/M7 |
| 314 | `:workspaceSlug/settings/projects/:projectId/features/views` | 1.1保存视图 | 保留项目功能开关，由M3对接；领域业务另属M6/M7 |
| 318 | `:workspaceSlug/settings/projects/:projectId/features/pages` | 1.2删除 | P2删除，2.1及2.3 |
| 322 | `:workspaceSlug/settings/projects/:projectId/features/intake` | 1.1需求收集箱 | 保留项目功能开关，由M3对接；领域业务另属M6/M7 |
| 327 | `:workspaceSlug/settings/projects/:projectId/states` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 332 | `:workspaceSlug/settings/projects/:projectId/labels` | 1.1工作区/项目/状态标签设置 | 保留；M3对接 |
| 337 | `:workspaceSlug/settings/projects/:projectId/estimates` | 1.2删除 | P2删除，2.1及2.3 |
| 343 | `:workspaceSlug/settings/projects/:projectId/automations` | 混合：1.1归档+1.2关闭 | P2只留自动归档，2.1/3.11 |
| 361 | `settings/profile/:profileTabId` | 混合：1.1+1.2 | 保留general/preferences/security/PAT；P3删notifications邮件页、改邮箱等 |
| 376 | `:workspaceSlug/projects/:projectId/settings/*` | 灰色：历史兼容 | P1删除，3.12 |
| 379 | `:workspaceSlug/analytics` | 灰色：历史兼容 | P1删除，3.12；C1须P1改仍存analytics菜单 |
| 383 | `:workspaceSlug/settings/api-tokens` | 灰色：历史兼容 | P1删除，3.12 |
| 387 | `:workspaceSlug/projects/:projectId/inbox` | 灰色：历史兼容 | P1删除，3.12 |
| 390 | `accounts/sign-up` | 灰色：历史兼容 | P1删除，3.12 |
| 393 | `sign-in` | 灰色：历史兼容 | P1删除，3.12 |
| 394 | `signin` | 灰色：历史兼容 | P1删除，3.12 |
| 395 | `login` | 灰色：历史兼容 | P1删除，3.12 |
| 398 | `register` | 灰色：历史兼容 | P1删除，3.12 |
| 401 | `profile/*` | 灰色：历史兼容 | P1删除，3.12 |
| 404 | `:workspaceSlug/settings/account/*` | 灰色：历史兼容 | P1删除，3.12；C1须先修PowerK目标 |

### 5.3 侧栏、设置导航

现用项目侧栏挂载链：`app/(all)/[workspaceSlug]/(projects)/layout.tsx:21` → `_sidebar.tsx:69` → `sidebar.tsx:36` → `core/components/workspace/sidebar/sidebar-menu-items.tsx:156-158` → `sidebar-item.tsx:66-71`。analytics可被固定偏好隐藏，但它是可用入口。旧`workspace-menu.tsx`/`user-menu.tsx`未发现被其他文件导入，不能作为现用入口证明。

| 入口 | 代码证据（web/相对） | 归类 | 设计处置 |
|---|---|---|---|
| home | packages/constants/src/workspace.ts:232-238 | 灰色→保留容器 | P2正文移除widgets/便签/pages；3.3固定列表 |
| your-work | packages/constants/src/workspace.ts:246-252 | 1.1跨项目工作项 | P2改目标页为assigned；既有链接保留 |
| drafts | packages/constants/src/workspace.ts:260-266 | 1.1草稿 | 保留 |
| notifications/inbox | packages/constants/src/workspace.ts:239-245；apps/web/core/components/navigation/top-navigation-root.tsx:67 | 1.1站内通知 | 保留，不等于邮件偏好 |
| stickies | packages/constants/src/workspace.ts:253-259 | 1.2便签 | P2删 |
| projects | packages/constants/src/workspace.ts:267-273 | 1.1项目 | 保留 |
| workspace views | packages/constants/src/workspace.ts:202-208 | 1.1视图 | 保留 |
| analytics | packages/constants/src/workspace.ts:209-215 | 1.2分析 | P2删；C1修P1仍存入口 |
| archives | packages/constants/src/workspace.ts:216-222 | 1.1归档 | 保留 |
| 自定义导航 | apps/web/core/components/sidebar/sidebar-wrapper.tsx:52,60-66（Preferences按钮及CustomizeNavigationDialog） | 灰色→1.2首页个性化 | P2先删；3.3固定默认顺序 |
| 收藏/项目列表 | apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx:13-15,38-40 | 1.1收藏访问 | 保留；P2删page类型，M7 handoff |
| 新工作项/草稿弹窗 | apps/web/core/components/workspace/sidebar/quick-actions.tsx:72-89 | 1.1工作项草稿 | 保留 |
| 项目工作项 | apps/web/core/components/workspace/sidebar/project-navigation.tsx:80-89 | 1.1工作项 | 保留 |
| 项目迭代 | apps/web/core/components/workspace/sidebar/project-navigation.tsx:90-99 | 1.1迭代 | 保留，M6 |
| 项目模块 | apps/web/core/components/workspace/sidebar/project-navigation.tsx:100-109 | 1.1模块 | 保留，P2删timeline/估算 |
| 项目视图 | apps/web/core/components/workspace/sidebar/project-navigation.tsx:110-119 | 1.1视图 | 保留，M7 |
| 项目Pages | apps/web/core/components/workspace/sidebar/project-navigation.tsx:120-129 | 1.2文档页 | P2删；顶部tab的use-navigation-items.ts:84-93也删 |
| 项目收集箱 | apps/web/core/components/workspace/sidebar/project-navigation.tsx:130-139 | 1.1收集箱 | 保留，M7 |
| 切换/创建工作区 | apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx:160-198 | 1.1工作区 | 保留；创建限制字段见I5 |
| 工作区邀请列表 | apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx:201 | 1.1邀请 | 保留；邮件发信由新后端省去 |
| 用户资料/偏好弹窗 | apps/web/core/components/workspace/sidebar/user-menu-root.tsx:111-134 | 1.1资料偏好 | 保留，P2主题修剪，P3删改邮箱 |
| 退出登录 | apps/web/core/components/workspace/sidebar/user-menu-root.tsx:136-139 | 1.1认证 | 保留，CSRF到M2 |
| god-mode | apps/web/core/components/workspace/sidebar/user-menu-root.tsx:140-147 | 1.2后台 | P3删 |
| 帮助文档/论坛 | apps/web/core/components/workspace/sidebar/help-section/root.tsx:49-54,80-84 | 灰色品牌外链 | P5删Plane外链或改Nerve仓库 |
| 联系销售 | apps/web/core/components/workspace/sidebar/help-section/root.tsx:55-60 | 1.2企业提示 | P3删 |
| 快捷键帮助 | apps/web/core/components/workspace/sidebar/help-section/root.tsx:62-70 | 1.1搜索/交互辅助 | 保留；删功能对应快捷键 |
| whats new/Plane版本 | apps/web/core/components/workspace/sidebar/help-section/root.tsx:71-86 | 1.2更新日志/品牌 | P2删日志；P5处理版本标识 |
| 旧dashboards/pi-chat菜单 | apps/web/core/components/workspace/sidebar/user-menu.tsx:35-58 | 死入口，不列现用灰色功能 | P2 Dashboard/ P3 AI死代码清理 |

设置入口完整清单，常量同时被设置侧栏与 PowerK 设置菜单消费（`core/components/power-k/menus/settings.tsx`、`ui/pages/open-entity/*-settings-menu.tsx`）；动态个人页不能只核core路由。

| 层次/键 | 常量证据 | 归类 | 设计处置 |
|---|---|---|---|
| profile/general | `web/packages/constants/src/settings/profile.ts:32` | 1.1 | 保留资料；P3删除改邮箱 |
| profile/security | `web/packages/constants/src/settings/profile.ts:36` | 1.1 | 保留修改密码；P3去autoset分支，CSRF到M2 |
| profile/preferences | `web/packages/constants/src/settings/profile.ts:40` | 1.1 | 保留语言/时区等；P1两语言，P2去自定义主题 |
| profile/notifications | `web/packages/constants/src/settings/profile.ts:44` | 1.2 | P3删邮件通知偏好 |
| profile/api-tokens | `web/packages/constants/src/settings/profile.ts:48` | 1.1 | 保留PAT及developer分类；M2对接 |
| workspace/general | `web/packages/constants/src/settings/workspace.ts:30` | 1.1 | 保留 |
| workspace/members | `web/packages/constants/src/settings/workspace.ts:37` | 1.1 | 保留系统内邀请/角色 |
| workspace/billing-and-plans | `web/packages/constants/src/settings/workspace.ts:44` | 1.2 | P3删 |
| workspace/export | `web/packages/constants/src/settings/workspace.ts:51` | 1.2 | P2删 |
| workspace/webhooks | `web/packages/constants/src/settings/workspace.ts:58` | 1.1 | 保留及developer分类；M8对接 |
| project/general | `web/packages/constants/src/settings/project.ts:33` | 1.1 | 保留；删page_view/estimate等属性 |
| project/members | `web/packages/constants/src/settings/project.ts:40` | 1.1 | 保留直接添加，P3改邀请文案 |
| project/features_cycles | `web/packages/constants/src/settings/project.ts:47` | 1.1 | 保留 |
| project/features_modules | `web/packages/constants/src/settings/project.ts:54` | 1.1 | 保留 |
| project/features_views | `web/packages/constants/src/settings/project.ts:61` | 1.1 | 保留 |
| project/features_pages | `web/packages/constants/src/settings/project.ts:68` | 1.2 | P2删 |
| project/features_intake | `web/packages/constants/src/settings/project.ts:75` | 1.1 | 保留 |
| project/states | `web/packages/constants/src/settings/project.ts:82` | 1.1 | 保留 |
| project/labels | `web/packages/constants/src/settings/project.ts:89` | 1.1 | 保留层级标签 |
| project/estimates | `web/packages/constants/src/settings/project.ts:96` | 1.2 | P2删 |
| project/automations | `web/packages/constants/src/settings/project.ts:103` | 混合 | P2只留自动归档 |

### 5.4 PowerK 全部 72 个静态命令 ID

以下实测枚举配置/上下文命令中的静态 id；具体显示仍由权限、项目功能开关及当前上下文控制。所有保留命令在P4改原生导航；有工作项上下文的命令在P3去Epic分支。删功能命令按2.3连同快捷键删除。

| 命令ID | 证据（web/apps/web/相对） | 归类 | 设计处置 |
|---|---|---|---|
| `workspace_invites` | `core/components/power-k/config/account-commands.ts:42` | 1.1保留能力或其交互辅助 | 保留 |
| `sign_out` | `core/components/power-k/config/account-commands.ts:53` | 1.1保留能力或其交互辅助 | 保留 |
| `create_work_item` | `core/components/power-k/config/creation/command.ts:74` | 1.1保留能力或其交互辅助 | 保留 |
| `create_page` | `core/components/power-k/config/creation/command.ts:86` | 1.2文档页 | P2删 |
| `create_view` | `core/components/power-k/config/creation/command.ts:99` | 1.1保留能力或其交互辅助 | 保留 |
| `create_cycle` | `core/components/power-k/config/creation/command.ts:114` | 1.1保留能力或其交互辅助 | 保留 |
| `create_module` | `core/components/power-k/config/creation/command.ts:127` | 1.1保留能力或其交互辅助 | 保留 |
| `create_project` | `core/components/power-k/config/creation/command.ts:140` | 1.1保留能力或其交互辅助 | 保留 |
| `create_workspace` | `core/components/power-k/config/creation/command.ts:152` | 1.1工作区 | 保留；I5创建限制字段明确交M2 |
| `open_keyboard_shortcuts` | `core/components/power-k/config/help-commands.ts:22` | 1.1保留能力或其交互辅助 | 保留 |
| `open_plane_documentation` | `core/components/power-k/config/help-commands.ts:34` | 灰色品牌帮助 | P5删除Plane文档/论坛或改Nerve仓库 |
| `join_forum` | `core/components/power-k/config/help-commands.ts:47` | 灰色品牌帮助 | P5删除Plane文档/论坛或改Nerve仓库 |
| `report_bug` | `core/components/power-k/config/help-commands.ts:60` | 灰色品牌帮助 | P5删除Plane文档/论坛或改Nerve仓库 |
| `toggle_app_sidebar` | `core/components/power-k/config/miscellaneous-commands.ts:55` | 1.1保留能力或其交互辅助 | 保留 |
| `copy_current_page_url` | `core/components/power-k/config/miscellaneous-commands.ts:67` | 1.1保留能力或其交互辅助 | 保留 |
| `focus_top_nav_search` | `core/components/power-k/config/miscellaneous-commands.ts:79` | 1.1保留能力或其交互辅助 | 保留 |
| `open_workspace` | `core/components/power-k/config/navigation/commands.ts:94` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_home` | `core/components/power-k/config/navigation/commands.ts:110` | 灰色：首页容器，1.1未单列 | M1:68默认保留入口并P2重写正文，非1.1明列 |
| `nav_inbox` | `core/components/power-k/config/navigation/commands.ts:122` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_your_work` | `core/components/power-k/config/navigation/commands.ts:134` | 1.1工作项视图 | 保留gy；P2目标根页重定向assigned |
| `nav_account_settings` | `core/components/power-k/config/navigation/commands.ts:146` | 1.1个人设置 | 保留；C1要求P1改正式目标 |
| `open_project` | `core/components/power-k/config/navigation/commands.ts:157` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_projects_list` | `core/components/power-k/config/navigation/commands.ts:173` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_all_workspace_work_items` | `core/components/power-k/config/navigation/commands.ts:185` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_assigned_workspace_work_items` | `core/components/power-k/config/navigation/commands.ts:197` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_created_workspace_work_items` | `core/components/power-k/config/navigation/commands.ts:208` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_subscribed_workspace_work_items` | `core/components/power-k/config/navigation/commands.ts:219` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_workspace_analytics` | `core/components/power-k/config/navigation/commands.ts:231` | 1.2分析 | P2删 |
| `nav_workspace_drafts` | `core/components/power-k/config/navigation/commands.ts:243` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_workspace_archives` | `core/components/power-k/config/navigation/commands.ts:255` | 1.1保留能力或其交互辅助 | 保留 |
| `open_workspace_setting` | `core/components/power-k/config/navigation/commands.ts:269` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_workspace_settings` | `core/components/power-k/config/navigation/commands.ts:285` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_work_items` | `core/components/power-k/config/navigation/commands.ts:297` | 1.1保留能力或其交互辅助 | 保留 |
| `open_project_cycle` | `core/components/power-k/config/navigation/commands.ts:315` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_cycles` | `core/components/power-k/config/navigation/commands.ts:339` | 1.1保留能力或其交互辅助 | 保留 |
| `open_project_module` | `core/components/power-k/config/navigation/commands.ts:359` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_modules` | `core/components/power-k/config/navigation/commands.ts:383` | 1.1保留能力或其交互辅助 | 保留 |
| `open_project_view` | `core/components/power-k/config/navigation/commands.ts:403` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_views` | `core/components/power-k/config/navigation/commands.ts:425` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_pages` | `core/components/power-k/config/navigation/commands.ts:443` | 1.2文档页 | P2删 |
| `nav_project_intake` | `core/components/power-k/config/navigation/commands.ts:461` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_archives` | `core/components/power-k/config/navigation/commands.ts:479` | 1.1保留能力或其交互辅助 | 保留 |
| `open_project_setting` | `core/components/power-k/config/navigation/commands.ts:498` | 1.1保留能力或其交互辅助 | 保留 |
| `nav_project_settings` | `core/components/power-k/config/navigation/commands.ts:520` | 1.1保留能力或其交互辅助 | 保留 |
| `update_interface_theme` | `core/components/power-k/config/preferences-commands.ts:108` | 1.1个人偏好 | 保留基础主题；P2删自定义 |
| `update_timezone` | `core/components/power-k/config/preferences-commands.ts:123` | 1.1保留能力或其交互辅助 | 保留 |
| `update_start_of_week` | `core/components/power-k/config/preferences-commands.ts:138` | 1.1保留能力或其交互辅助 | 保留 |
| `update_interface_language` | `core/components/power-k/config/preferences-commands.ts:153` | 1.1个人偏好 | 保留；P1只剩两种语言 |
| `toggle_cycle_favorite` | `core/components/power-k/ui/pages/context-based/cycle/commands.ts:73` | 1.1保留能力或其交互辅助 | 保留 |
| `copy_cycle_url` | `core/components/power-k/ui/pages/context-based/cycle/commands.ts:88` | 1.1保留能力或其交互辅助 | 保留 |
| `add_remove_module_members` | `core/components/power-k/ui/pages/context-based/module/commands.tsx:105` | 1.1保留能力或其交互辅助 | 保留 |
| `change_module_status` | `core/components/power-k/ui/pages/context-based/module/commands.tsx:122` | 1.1保留能力或其交互辅助 | 保留 |
| `toggle_module_favorite` | `core/components/power-k/ui/pages/context-based/module/commands.tsx:139` | 1.1保留能力或其交互辅助 | 保留 |
| `copy_module_url` | `core/components/power-k/ui/pages/context-based/module/commands.tsx:154` | 1.1保留能力或其交互辅助 | 保留 |
| `toggle_page_lock` | `core/components/power-k/ui/pages/context-based/page/commands.ts:90` | 1.2文档页 | P2删 |
| `toggle_page_access` | `core/components/power-k/ui/pages/context-based/page/commands.ts:114` | 1.2文档页 | P2删 |
| `toggle_page_archive` | `core/components/power-k/ui/pages/context-based/page/commands.ts:139` | 1.2文档页 | P2删 |
| `toggle_page_favorite` | `core/components/power-k/ui/pages/context-based/page/commands.ts:161` | 1.2文档页 | P2删 |
| `copy_page_url` | `core/components/power-k/ui/pages/context-based/page/commands.ts:176` | 1.2文档页 | P2删 |
| `change_work_item_state` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:202` | 1.1保留能力或其交互辅助 | 保留 |
| `change_work_item_priority` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:222` | 1.1保留能力或其交互辅助 | 保留 |
| `change_work_item_assignees` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:242` | 1.1保留能力或其交互辅助 | 保留 |
| `assign_work_item_to_me` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:259` | 1.1保留能力或其交互辅助 | 保留 |
| `change_work_item_estimate` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:277` | 1.2估算 | P2删 |
| `add_work_item_to_cycle` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:297` | 1.1保留能力或其交互辅助 | 保留 |
| `add_work_item_to_modules` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:337` | 1.1保留能力或其交互辅助 | 保留 |
| `add_work_item_labels` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:369` | 1.1保留能力或其交互辅助 | 保留 |
| `subscribe_work_item` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:392` | 1.1保留能力或其交互辅助 | 保留 |
| `delete_work_item` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:407` | 1.1保留能力或其交互辅助 | 保留 |
| `copy_work_item_id` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:420` | 1.1保留能力或其交互辅助 | 保留 |
| `copy_work_item_title` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:433` | 1.1保留能力或其交互辅助 | 保留 |
| `copy_work_item_url` | `core/components/power-k/ui/pages/context-based/work-item/commands.ts:446` | 1.1保留能力或其交互辅助 | 保留 |


## 6. 对第 3 节十三条裁定的意见

| 裁定 | 意见 | 理由、边界与应补内容 |
|---|---|---|
| 3.1 编辑器去协作 | **同意，有保留边界** | binary/provider/实时事件的功能消费者随 Pages 删除；保留非协作内核。UniqueID 在 `web/packages/editor/src/extensions/extensions.ts:138` 为所有编辑器安装，`components/editors/editor-container.tsx:40` 的非协作节点定位也用 attrs.id；建议现在确定保留本地 ID，只去 provider。`copyMarkdownToClipboard` 被描述历史使用，不随导出删除。这里不是已发现设计要求误删 UniqueID，而是待核项已有答案 |
| 3.2 个人主页 | **有保留** | 三工作项标签和成员卡片方案可行；workspace wrapper 已预取成员（`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx:93`），member store `workspace-member.store.ts:239` 将 member.created_at 赋给 joining_date，确无时区。需明确加载中、请求失败、目标已退出/不存在的状态，不能把缺成员渲染成空名字。访客权限属于 M3 产品/后端矩阵，不应由 M1 前端重定向自行扩大或永久缩小；原 Plane profile API 要求 active membership |
| 3.3 固定侧栏 | **同意** | `plane-diff.md:46` 已删除 workspace_user_preferences；02fd956 将范围明确化是合理同步。固定默认顺序可用；项目导航的 workspace_user_properties、app rail 本地偏好、折叠/宽度及收藏都不是待删远端导航偏好 |
| 3.4 按工作项数的进度 | **同意** | 去估算后保留数量燃尽符合 V:53。AreaChart/recharts 仍被 `web/apps/web/core/components/core/sidebar/progress-chart.tsx:9` 使用。保留实际 Plane analytics URL 至 M6 合理；同时覆盖当前/已结束迭代 snapshot、模块与乐观更新，不能只改图表名称 |
| 3.5 services 修剪 | **同意** | 49 文件试删通过。3.5 的交付属于 P1；P3 交付范围又列 3.5 只是文档重复，别在 P3 再做一遍。URL helper/签名 URL 行为和现有测试保留，整包随 M2 PAT/M5 文件及剩余调用替换后删除 |
| 3.6 CSRF 留 M2 | **同意** | 这是传输替换顺序，M1 提前删会需要临时认证实现。02fd956 已同步 V:538 与 F:117，当前材料间没有 CSRF 冲突。原生邮箱密码端点在 Plane 中存在，去 email-check 不要求先建新后端。P3 保留 CSRF 时必须补 `**/auth/**` 请求断言；M2 必须同时删 Cookie/CSRF，不能无限延期 |
| 3.7 实例配置 | **有保留** | 删第三方/SMTP/AI/Unsplash/后台 setup 开关合理，但 config 不是只剩注册开关，见 I5。InstanceWrapper 的请求失败维护页与 is_setup_done 的未设置页是不同分支（`web/apps/web/core/lib/wrappers/instance-wrapper.tsx:39`、`:45`），前者保留。M2 仍负责最终 schema；M1 不应私定后端新字段 |
| 3.8 评论 access | **同意前端删除；表列由 M4 决策** | 上游公开 space 路径以 EXTERNAL 过滤（`plane/apps/api/plane/space/views/issue.py:96`、`:265`），删公开发布后此 UI 不再需要。它不是管理员/成员/访客评论权限；回复、表情、署名仍保留。M4 定是否删列并登记差异，不把“类型删了”当数据库已有决议 |
| 3.9 多选 | **反对现有判定方法，同意删除批量能力** | 有引用不等于有业务需求；当前 CE 整链 disabled，见 I3。应显式删工作项批量选择及透传，保留其他实际选择交互；若要新增键盘行导航，单独定义最小能力 |
| 3.10 外部 propel | **同意，有来源条件** | 它是现用第三方依赖，不是遗漏的工作区改名。保留锁定版本、AGPL 标识及只针对外部名字的例外可控；不应为品牌搜索零命中把它改名或删除。把“以后 vendor”写成有对应源版本的备选，不承诺 npm 构建物可无成本还原源码。产品负责人确认接受这一长期依赖即可 |
| 3.11 后续字段交接 | **有保留** | 领域划分正确，但当前列表不完整，必须写成按实际删除 diff 扩充的清单，见 I5 和 7.5。不能仅把缺字段错误交给未来生成类型发现，也不必 M1 提前改后端 |
| 3.12 旧重定向 | **反对现有 P1 实施清单** | 删除历史兼容符合原则，但仍有现用内部调用，C1 已复现。必须同步改终点或归入 P4；不能只修两条 sign-in 后全删 |
| 3.13 修改登录邮箱 | **同意删旧邮件流程，有产品保留意见** | `web/apps/web/core/components/core/modals/change-email-modal.tsx:113` 发验证码，`:74` 验证后更新并退出；确依赖邮件。设计已交 M2 决定替代，因此不是完全漏交接。但“个人资料保留”不自动授权永久禁止改邮箱；输入当前密码修改、管理员 CLI 修改或 v0 暂不提供，须负责人选，不能由 P3 删除者默定 |

D:102 声称所有裁定“都不涉及后端”，范围表述过满。**3.6 改交付时机，3.2 触及访客访问范围，3.3 取消一个偏好接口，3.7 定实例公开字段，3.8 影响列级语义，3.11 本身就是新接口约束，3.13 涉及账户能力。**其中 3.3 已有删表依据、3.6 已同步上级，不能重复报成新矛盾；其余也不是要求 M1 写后端，而是必须有负责人和后续决策点。已同步的文档不能单独证明产品负责人批准了所有新增产品取舍。

## 7. Phase 划分的评价与推荐拆法

### 7.1 保留五个 Phase，改几个边界，避免以文件数机械拆分

**推荐继续 P1 → P2 → P3 → P4 → P5 → 收尾。**P1 先清工具链/语言/无消费者 services 是有价值的；P2/P3 按功能依赖链删到各层，比按包切开更容易单独合并。路由迁移和改包名放后面，能避免为最终要删除的文件修参数和改 import。不是“Phase 越少越好”，也不是每碰一次公共文件就要另开 Phase。

| 方案 | 好处 | 代价与判断 |
|---|---|---|
| 原五 Phase，补边界 | 合并点清楚；每次的功能变化相对可解释；P4/P5 面更小 | P2/P3 共享文件会重复评审；**推荐**用同一 owner、完整依赖链任务和同一回归矩阵控制 |
| 合并 P2+P3 成一次大删除 | root/editor/issue types 有机会一次清干净 | 约670个文件删除（设计估计，须去重）加数百修改，审查/失败定位/撤回成本陡增；减少流程次数不等于减少实际推理量，不推荐整个合并 |
| 按 packages / app / store / types 分层 | 同包集中修改、减少文件冲突 | app 和 types 的删改耦合最强；为独立合并很容易制造临时 stub/兼容字段，违背原则，不推荐作为 Phase 边界 |
| 一次整体删除，再按 tsc 修复 | 找直接静态消费者快 | 会把保留入口删到“不再有人引用”，类型/knip反而全绿；只能用于一个已经核定的功能依赖链，不能作为全 M 算法 |
| 先 P4 路由，再删除 | 提前统一路由 API，后续无需碰 Next shim | 152 错误中包含将删除文件；AuthWrapper 和侧栏仍需在 P2/P3 重改；不值得。只把失效内链 C1 先修或延后同删 |
| 先改包名 | 早日统一品牌/命名 | 大量即将删掉的 import 和锁文件先被扰动，内容 diff 审查更难；保留末尾纯机械提交 |
| P5 包名与品牌揉成一个提交 | 少一个提交 | 搜索替换噪声遮住 Logo/文案语义变更；维持单独机械提交，随后品牌语义提交 |

重复修改候选的量级来自 `/tmp/m1-coupling-inventory.json`：扫描跟踪 TS/TSX，P2 正则并集 438 文件、P3 236、交集 **51**；本评审另以 `next/(link|navigation)|useAppRouter` 扫描得到的路由相关文本候选 359，其中与 P2 交集 87、与 P3 交集 51，三者交集 11；路由候选中 324 文件还含工作区包名。附加统计及完整模式记录在 `/tmp/m1-cross-phase-counts.json`。**这些是文本候选集合，不是“51 个文件确定改两遍”**：含注释、将被 P1/P2 整删的文件；实际 surviving 修改量需实施清单按路径去重。它说明先路由/先改名更可能扩大重复，而不是证明目前粒度不可用。

| 共享文件/边界 | 预计经过哪些 Phase | 控制方式 |
|---|---|---|
| `web/apps/web/app/routes/core.ts` | P1旧redirect（建议移出）、P2内容、P3平台、P4路由 | 唯一集成 owner；删除 route 同任务清导航/快捷键/空态/旧redirect |
| `core/store/root.store.ts` | P2内容 stores，P3多选/平台，P5包名 | 每次同时改 import、字段、constructor、resetOnSignOut；禁止多任务并行改此文件 |
| `core/store/workspace/index.ts` / sidebar | P2 home/nav 接线，P3 EE/平台，P4匹配，P5品牌 | P2把远端自定义侧栏及首页集中收口，不顺带删项目/本地偏好 |
| `core/store/issue/root.store.ts` / `use-issues.ts` / `use-issue-layout-store.ts` | P2内容引用，P3 EPIC/TEAM/type，P4参数，P5包名 | P3由一个任务一并改实例和分支，测每个保留上下文取对 store |
| cycle/module issue store + `distribution-update.ts` | P2去点数，P3企业分支，P5包名 | P2把数量算法和调用方一起验，P3不重写数量算法 |
| 列表/表格/工作项表单 | P2估算/Gantt，P3type/多选/模板，P4参数，P5包名 | 同任务删除传参链，避免几十次只删一个 prop 的零碎提交 |
| editor types/extensions/ref/wrappers | P2协作，P3 AI/EE（可前移），P5包名/品牌 MIME | 见 M2；保留同一编辑器行为验收，避免第二次从头人工猜 |
| 活动/通知 renderer | P2估算/page，P3type/epic/public，P5品牌 | 只删对应 case，保留 state/relation/cycle/module/comment renderer；默认分支不能掩盖误删 |
| types/constants/barrels、propel registry、翻译 JSON | P1/P2/P3，P5品牌 | 功能 owner 提供删除符号清单，唯一 owner 收口导出/注册；共用文件不用整文件关键词豁免 |

表内 `core/` 均位于 `web/apps/web/`；editor 位于 `web/packages/editor/`。这些是责任边界，不要求每个表格单元都产生独立任务。

### 7.2 推荐每个 Phase 的内部任务顺序与可合并条件

- **P1**：工具/构建遗留、两语言和键同步 → services 49 文件/barrels → lint 精确计数原型、关键词规则原型 → 开发监视/水合修复。旧 redirect 归 P4 或按 C1 完整迁移。services 提前删已有类型原型支持；i18n 清键不能仅搜 namespace。保留 URL 测试运行入口，Storybook 不必因此保留。knip 仍是报告，但三类依赖问题收敛到设计的 buffer 例外，不把 buffer 提前当 Node 内置删掉。
- **P2**：固定侧栏 → 全局分析+个人统计/个人主页 → 估算+数量 progress 全链 → Gantt/模块 timeline（先搬 REVERSE_RELATIONS）→ Pages+更新日志+搜索/收藏/资源分支 → 编辑器收敛 → 便签 → 首页固定最近访问 → 导出和集成一起删 → 统一 root/barrels/文案/资源收口。并非所有箭头都是硬依赖：真正必须的包括分析先于估算、Pages 先于 editor、Pages/便签先于首页、个人统计与分析先于 bar/pie 图表删除、导出与其 IntegrationService 同删。**新增约束**：Tour 的 views→pages 步骤要接到结束；profile 卡片/重定向先到位再删统计数据源；数量计算与旧 estimate 字段同任务；图标/扩展注册表和调用方同删。
- **P3**：认证+实例+邮件相关表单（包括 onboarding 和安全页）→ 公开发布+comment access → EPIC/TEAM/type+工作项批量选择+模板等空壳 → 余下 AI/Unsplash/EE → 计费/升级 → 死代码、接线、资源、knip 门禁。**新增约束**：实例字段与所有读它们的入口同时改；删 `EIssueServiceType` 多态时不能把所有 endpoint 改成 `issues`，保留描述历史及按编号查找确实使用 `work-items`（`web/apps/web/core/services/issue/work_item_version.service.ts:18`、`issue.service.ts:445`；Plane `apps/api/plane/app/urls/issue.py:267`）。共享 ProjectIssues 类仍供普通项目使用，不能随 TEAM 实例整类删。计费最终清理避免前面反复搬引用，但它引用的已删类型必须在对应 P2 任务同步处理，不能为等待 P3 保留空类型。
- **P4**：先消除各包 env 读取并正确处理 i18n 整个条件及类型 → 再去 define/dotenv；迁移 Link/hooks/命令上下文，参数守卫与 Navigate；最后删 shim/旧声明/强制 slash，并完成全导航匹配矩阵。生成路由类型与共用组件参数要分清；不用批量 `!`/`as string`。原 API URL helper 与 UI slash 分开处理，签名上传不能加斜杠。
- **P5**：包名纯机械提交+锁文件等价核对 → 品牌资产/文案/外链/本地存储键/组件名 → 可见品牌及版权来源检查。不在同一个提交升级依赖。包名核对允许根 `nerve`、应用 `web`、所有 `@nerve/*`，第三方 `@makeplane/propel` 保持原名。
- **收尾**：全清单和例外到期检查、双语死键核对、锁图/构建体积/警告规则统计、手册/改动清单/后续 handoff。不能等收尾才第一次定义首页或多选语义。

各 Phase 仍必须合并时通过类型/lint/format/build 和 S1–S4；**P1/P2 的 knip 是有明确剩余项的报告，P3 才全零并门禁**，这不是顺序陷阱。已有二级类型依赖可在同一功能提交里改消费者，不必为保持错误的文件边界加过渡层。本次只证明部分基线和原型可行，未运行未来 Phase 的整套代码。

### 7.3 最小保留行为验收矩阵

“静态 M1”表示不以新 Go 业务后端为完成前提，不等于只要页面停在错误页就能证明保留行为。以下核对不新增业务用户故事，也不要求搭完整旧后端。

| 所属 Phase | 最小断言 | 合理手段 |
|---|---|---|
| P1 | unsupported locale→en；中英文键集合；预渲染/首次客户端结构一致，无 #418；包源码改变能进入 dev 产物 | 键比较/构建；小浏览器核对；受控改一个包符号后观察更新，再恢复副本 |
| P2 首页/个人主页 | 最近访问有数据与空态；无 page；profile root 转 assigned；member/guest、目标不存在、成员接口失败不混为一个状态 | 最小页面 mock；保存断言脚本及 fixture |
| P2 editor | 工作项富文本输入保存、撤销/重做；lite 评论与 @成员；历史 HTML 查看/还原/复制 Markdown；图片/节点定位保留 | 可复用小编辑器 harness；不需要模拟 Yjs 服务器 |
| P2 progress | 创建/完成/取消/重开后的数量与进度；当前/已结束迭代 snapshot、模块；无点数切换 | 少量真实纯函数测试+一组渲染核对；覆盖数量守恒，避免按实现逐行写测试 |
| P2/P3 活动/通知/工作项 | state/relation/cycle/module/comment 动态；通知预览/全部已读；项目/迭代/模块/个人主页/归档选择正确 store；列表/看板/表格/日历仍有入口 | 代表性无 estimate/page/epic 的数据；断言保留显示与 store 选择 |
| P3 认证/实例 | 登录和注册邮箱可编辑；正确 POST 字段/CSRF/next_path；成功/失败回跳；关闭注册；实例请求失败；修改密码不被重置密码清理误伤 | 拦截 `**/auth/**` + `/api/**`，校验 request body 和最终 URL；未列出的请求立即失败 |
| P4 路由 | 未登录、未引导、已登录已引导/无工作区；next_path/back/replace；profile、设置、side rail、顶部通知有/无尾斜杠；query/hash不丢 | 同一 harness 增量用例；原生路由小测试可长期保留 |
| P5 | 图形/页标题/错误页/帮助链接无可见 Plane；PAT/Webhook设置仍可达 | 复用前面场景；S2继续不锁死启动页文本 |

“不写以后会被替换的代码”应限制不必要的产品实现，不应解释为不保存回归证据。稳定的纯计算/URL/编辑器行为值得提交小测试；旧接口 fixture 可以只放 review 的可运行附录并明确到 M2/M3 后用新客户端数据替换。**不建议**为了 M1 增加长期完整 Plane API mock 层、截图全集或新的兼容模型。

### 7.4 工具门禁、关键词及缓存的可行性

| 项 | 判断及必要条件 |
|---|---|
| 7.1 每包 warning 精确等于上限 | **可行**。直接使用锁定 oxlint JSON 的 diagnostics severity 计数，本次已逐包重算995；检查必须使用与 lint 相同的 cwd/config/忽略/依赖构建结果，先保留 error/进程失败语义再比较数值。上限仍从 package.json 单处读取。数字相等不能证明没新增某一条又删另一条，不能宣传为逐警告防回归 |
| 新检查与 Turbo 缓存 | 将“运行 oxlint+比较上限”做成同一个任务，成功结果整体缓存；root 检查脚本须进入该任务 inputs/globalDependencies，包 package.json、全局 lint config、锁文件/上游构建已纳入。别解析 errors-only 控制台日志，也别在 cache hit 时读未恢复的临时 JSON。P1 原型至少验证冷/热、少一警告、超一警告、改上限、改根脚本、改依赖包六类。当前 `check:lint.dependsOn=[^build]` 保留；`check:format.cache=false` 是 M0 已知正确选择，勿为省秒数又打开 |
| 7.2 knip 门禁+config hints | **可行**，typegen 后取消 web ignore 的原型未产生 unresolved。P1 删除 i18n 生成及对应 ignore，修 tailwind main；P3先typegen再knip，最后 `--treat-config-hints-as-errors`。使用真实 `index.css` 入口/exports 语义，不为压提示造空 JS。knip 目前由 Make 直跑，不需要 Turbo 缓存；若以后缓存，生成类型和配置必须进输入。按[官方 CLI](https://knip.dev/reference/cli)区分 report-only 与 hint error 选项 |
| 双语键一致性 | **可行，但必须双向**。现 `web/packages/i18n/scripts/sync-check.ts:96` 已算 missing/stale，`:172` 只因 missing 置失败；只删语言目录后原样接 CI 不够，应令 en/zh-CN 集合严格相等、两目录都存在，保留冲突检查。script、locale JSON 均应入任务 hash。修改后的具体脚本还未存在，本次未验证其实现 |
| 未引用翻译键 | **不能由 knip 或 en/zh-CN 相等证明**。整键+模板前缀是起点，还需枚举常量/动态参数、ICU插值及fallback namespace；“两边一起误删”照样通过同步。按功能提交记录删键与消费者对应；收尾做完整集合复核，对不能证明死的键先找所有变量来源，不能直接 delete。无需为此 M1另建一套复杂翻译编译器 |
| 锁文件 | 安装固定 Node/pnpm；依赖删除前后比对 importers 和存活包的 version/integrity/dependency edges，允许明确由移除引起的 peer suffix/孤儿包差异并解释。不能只比包数。同步查 catalog、overrides、allowBuilds，也查本文件实际还有的 minimumReleaseAgeExclude/peerDependencyRules/patches；删Storybook/masonry等后旧例外也应退出。P5重命名可在仓库根使用设计的文本等价 diff；命令应在未提交重命名时对 HEAD，提交后对父提交，别混淆基准 |
| web-dev | 10个包watch+web共11个persistent task，本次从scripts重算；并发至少12合理，P1须观察包变化而非只看Vite启动。去dotenv/define后P4另测开发预构建，不用一次production build代替 |
| CI耗时 | 基线冷lint31.256秒、build14.676秒，不是未来预算。每个可合并Phase跑全门禁；同任务局部迭代可先受影响包检查，收口全跑，避免每次删一个文件都重复全仓构建。无需借M1更换包管理/打包/状态工具 |

关键词重点修订表（基于本次实际命中，不把所有命中都判为待删）：

| 模式/范围 | 误报、漏报或边界 | 处理 |
|---|---|---|
| `TExtended` / `TAdditional` | 保留 extended-sidebar props、WorkItemFilters props；确有命中 | 用确切EE符号清单/调用链；别按前缀删除实用结构 |
| `(?i)plane` | 版权、第三方名、airplane/planet；URL测试样例也有 api.plane.so | 版权保留；第三方仅精确包名例外；测试样例可换中立域名，不能把整个同一行/文件全部排除 |
| analytics / wiki | tlds.ts含合法TLD；progress旧URL仍要用 | 仅TLD数据和具体旧URL例外；不要豁免整个progress service |
| timeline/epic 的 lookbehind | 内容PCRE可用，文件名ERE报错 | 文件名用独立合法模式/统一检查器；失败码必须失败 |
| TEAM 分支 | 现 enterprise 模式只有部分TEAM_*，裸 `EIssuesStoreType.TEAM`可漏 | 加精确符号/枚举值扫描；普通英文team不能全删 |
| `workItemTemplateId`、`recurring_work_items`、`WORKLOG`/`is_time_tracking_enabled` | 当前7.4未覆盖；有引用空壳、类型/动态注册、死文案不保证knip发现 | P3列入手工清单和精准模式；不可泛搜所有template（JS模板/估算模板等会误报） |
| Pages相关模式避开普通page | 避误报方向对，但实际Tour的Pages步骤/图片不一定含 pageId/page_view | 另验导览步骤和资产目录，views下一步改结束；不靠扩大到所有page来删路由分页 |
| 程序源码 `process.env` | 当前目标含Vite config的注入以及包内读取；NODE_ENV条件须语义改写 | i18n补 env 类型；Node工具若新增合法process.env读取，应按执行环境审查，不能无脑替换成import.meta |
| 只搜web、跳二进制 | 根锁文件/catalog/globalEnv遗漏；二进制品牌不识别 | 根依赖图单独验，图形资源以使用点+预览/实际页面验，不能宣称搜索替代可见品牌检查 |
| 各Phase允许后续文件 | 一个root/types/JSON同时属于多个Phase，按文件豁免太粗 | 以规则+符号/文本+owner+到期点登记；每Phase剔除已到期例外，收尾只余明确跨M旧接口例外 |

本机 Apple Git 能运行所有55个内容模式；**未证明任意macOS/Linux安装都同样支持PCRE**。选统一执行器并锁测试样例，比依赖“系统git应该一样”更可验收。

### 7.5 后续里程碑与保留代码的交接

3.11 应当是最低清单，实施 diff 才是最终依据。下表把遗漏补到领域；已有上级差异不是本报告要求 M1 新增后端开发。

| 接收 M | 已有交接 | 还应补/明确 |
|---|---|---|
| M2 账户/实例/PAT | 登录来源/autoset/邮箱验证/营销/计费字段，theme只留theme，CSRF/session，改邮箱决策 | 实例注册、创建工作区限制、文件上限及启动失败语义；is_bot/bot_type/user_type与其调用方按领域归属清除；旧头像/封面字段等已在差异清单的项保持链接；无邮件账户恢复/改邮箱方案 |
| M3 工作区项目 | page_view/close_in/default_state/anchor、侧栏偏好取消、profile访客权限、slug保留名单 | 项目estimate_id/is_issue_type_enabled/is_time_tracking_enabled等实际移除字段；workspace_user_properties与被删preferences表分清；token邀请链接是否保留、非成员profile处理；工作区/项目层external_source/id按既定差异处理 |
| M4 工作项 | estimate_point/type_id/is_epic/description_binary、comment access、issue-dates、搜索page | 描述历史/snapshot中的binary/点数字段；保留activity字段与删除估算/type/epic分支的取值约定；评论回复/表情/成员提及/关系功能保持；work-items历史端点在M1仍正确保留；外部来源字段及draft payload改造引用差异清单 |
| M5 文件 | PAGE_DESCRIPTION、剩余services包退出 | file_assets.page_id及page-only资源类型/授权分支；file_size_limit策略归属；编辑器图片、附件、头像/封面、签名URL行为仍要保留。services删除前检查URL helper的最后消费者，不能只数PAT/上传两处 |
| M6 迭代模块 | points/estimate_distribution，旧analytics URL例外到期 | total/completed/backlog/started/unstarted/cancelled等estimate字段、快照分布、burnup/burndown类型和空数据语义以实际diff全列；数量更新及已归档对象进度保留 |
| M7 协作 | 收藏/最近访问去page | 通知实体/活动类型中page/estimate/type/epic删除的实际枚举；保留member mention/工作项关注；最近访问改显式写入（差异清单:115），不是GET副作用；首页入口持续存在 |
| M8 开放发布 | 本设计没有接口改动，不必造新的M1后端接口 | 确认PAT/Webhook UI与developer分类保留；调用日志/OpenAPI页面当前没有迁入实现，属于M8新增；propel源码/第三方告知与构建体积记录交接 |

删除的原代码有固定 Plane 提交可追溯，并非永远找不回；真正昂贵的是公共接口/保留交互的消费者被一起删掉后再拼回来。因此 M1 要保护 `copyMarkdownToClipboard`、本地 UniqueID、用户mention、数量分布更新、非估算操作动态和正确的历史 endpoint；不是为以后可能用而保留协作、禁用多选、模板空壳、工时宣传。CSRF及旧API地址按明确M交接暂留是合理顺序，不应借此把本应M1删除的功能本体推到M2–M8。

## 8. 需要项目负责人决策的事项

以下列的是产品/范围决定；不要求负责人替实施者选择 ts-morph、守卫写法或测试目录。已获项目授权的决定不必再次机械审批，但设计需记录实际选项与责任，不能只用“可逆、不涉及后端”代替。

| 事项 | 选项与代价 | 推荐与决定时间 |
|---|---|---|
| 首页删完后给用户看什么 | 固定问候/空态/最近访问：最小且保留能力；迁到其他入口：首页更空，但用户路径和实现都增加变化 | **固定首页保留项目/工作项最近访问**，开工前确认；P2不能自行删掉唯一入口 |
| 个人主页访客能看什么 | 暂按原访问边界显示卡片+无权提示：改动最少；开放工作项标签：需要与项目权限一致；完全隐藏主页：更简单但减少成员信息入口 | M1采用设计的最小形态，**M3统一决定最终权限**；明确已退出/不存在成员状态，不在M1凭前端数据源扩大权限 |
| 没有邮件后如何改登录邮箱 | 当前密码确认：自助体验好但要设计身份校验/会话处理；管理员CLI：实现小但增加人工运维；v0暂不支持：最省事但减少资料能力 | **M1只删邮件流程；M2默认优先评估管理员CLI或安全自助方案**。需要负责人明确最终能力，不能把旧实现删除等同永久禁用 |
| 复制工作区邀请链接是否继续提供 | 仅系统内待办接受：最符合“不发邮件”，简单；仍允许复制token链接：便于外部传递，但要定义过期/撤销/登录后承接 | M1保留现入口到M3，**M3一次决定邀请契约**并清掉不用路径；不得随“项目邀请”清理误删整个工作区邀请 |
| 实例是否可禁止普通用户创建工作区、上传上限如何展示 | 保留公开配置字段：维持当前约束；删前端开关：UI更简单但仍需后端策略与失败提示 | **M1保留现用两字段及消费者**；M2/M5定新名和策略；若改变产品策略由负责人明确，而非清类型时删掉 |
| 长期保留 @makeplane/propel 吗 | 锁定外部包：M1工作量小、继续依赖上游供应；vendor对应源码：自主性强，但新增维护范围 | **当前保留0.3.0，记录对应源码与许可证**；无证据需要M1整包搬入。与可见品牌替换是两个决定 |
| 固定侧栏、仅数量进度、取消公开评论access、CSRF时机 | 侧栏/估算/公开发布上级已有删表/删功能决定；若反向保留，会重新引入存储/接口和范围。CSRF提前删则要中间传输实现 | **认可现方向**；3.8列级M4确认，3.6 M2替换时强制退出。现已同步的3.3/3.6不应反复阻塞实施等待同一授权 |

工作项模板、定时/重复工作项、工时记录没有现用完整能力，上级也未承诺。推荐明确作为死文案/企业空壳删除；若负责人要把它们加入 v0，属于新增产品范围，需另行修改总体设计和后端路线，不能靠此次清理顺便“实现”。

## 9. 处理建议清单

### 开工前必须改

1. 修 C1 的阶段责任：旧重定向随完整调用链迁移，不能按当前两条内链清单在P1全删。
2. 明确首页固定保留最近访问；把3.9改成工作项禁用多选整链删除，区分通知/普通下拉。
3. 把关键词表变成可执行、能辨别正则错误的规则；精确管理共享文件残留，补模板/工时/Tour及根依赖范围。
4. 把 editor/progress/工作项store/认证提交纳入最小行为矩阵；明确临时脚本证据可重跑，`**/auth/**`不能遗漏。
5. 明确3.7实例保留字段、3.11按实际diff补齐交接；统一无尾斜杠语义；修F:83完成条件。

### 可以并入对应的 Phase

- **P1**：已有13条URL测试接入口；lint JSON精确检查的缓存原型；双向翻译键检查；两语言回退；11 persistent tasks的dev监视与#418核对；49个services文件及工具遗留修剪。
- **P2**：首页/个人主页/侧栏保留行为；数量进度及snapshot；Pages→editor全链、UniqueID本地功能、Markdown历史复制、用户提及；Tour去Pages并接结束；是否吸收editor AI/EE收敛任务。
- **P3**：认证+实例字段同改；Epic/TEAM/type/多选/模板等同链删除；工时/重复文案/企业残留明确清点；公开发布与comment access；knip及配置提示门禁。
- **P4**：全部保留URL目标和菜单匹配、Auth Navigate分支、参数守卫、env条件与类型、签名URL边界；旧redirect完整退出。
- **P5/收尾**：机械包名独立提交，版权/外部依赖精确例外；来源记录、锁图/产物体积、死翻译键与全部handoff；修11/161等无功能影响的计数。

### 留给后续的哪个 M

M2负责Bearer/CSRF退出、账户恢复/改邮箱、PAT/实例新契约；M3负责profile访客权限、工作区邀请与slug名单/项目字段；M4负责comment.access列及工作项/历史/动态新契约；M5负责文件策略与services最后退出；M6负责数量progress新接口及analytics例外清零；M7负责通知/最近访问显式写入；M8负责尚无实现的日志/OpenAPI界面与发布来源告知。具体清单见7.5，M1结束时应有接收路径和关闭条件。

### 不建议做

不重做整个M1设计；不把P2/P3合成一次不可细审的大删除；不按包/层拆到需要临时stub才能合并；不把“knip认为有引用”等同“产品必须保留”；不先替即将删除的文件修原生路由或改品牌；不M1重写所有services/store；不为删除源码关键词vendor整个propel；不建立完整Plane假后端；不为了静态清零误删真实功能、随意扩大全文件豁免或把工具执行失败当零结果。

## 10. 处理结果（控制者核实与吸收）

**做法**：
- 每条发现都先对照代码核实。
  - 核实用的是只读的脚本：路由表、PowerK 命令和侧边栏常量的跳转目标、首页组件的挂载链、`useBulkOperationStatus`、实例字段的使用方、尾斜杠的比较、`sync-check.ts` 的失败条件、编辑器的 `UniqueID` 和 `copyMarkdownToClipboard`、`work-items` 地址、文案键数等。
  - 抽查到的事实都与报告一致，发现全部成立。
- 所有处理都吸收进 [M1 设计](../M1-design.md)，不另开补丁文档。
- 第 1–9 节保留 Codex 原稿的样子，处理结果以本节为准。

### 10.1 逐条处理

| 发现 | 核实 | 处理（M1 设计中的落点） |
|---|---|---|
| C1 旧重定向仍有现用入口 | 成立。命令面板 `nav_account_settings` 跳到 `/:workspaceSlug/settings/account`，侧边栏的数据分析 `href` 是 `/analytics/`，都靠重定向到达；共 11 条（不是 13 条） | 3.12 改写：数据分析的重定向随数据分析在 P2 删除，其余 10 条移到 P4，先把依赖它们的入口改成正式地址，再在同一提交里删重定向；P4 用真实路由表核对所有内部跳转目标（4.2）；P1 不再包含旧重定向 |
| I1 运行时验收有缺口，认证提交不在 `/api/**` 下 | 成立。原生表单提交到 `/auth/sign-in/`、`/auth/sign-up/`；`@plane/services` 的 `test` 脚本运行着保留地址工具的 13 个测试 | 新增 7.5 保留行为矩阵：稳定逻辑写进仓库的单元测试，前端 vitest 在 P1 接入 Makefile 和持续集成（总体设计 8.1、8.3 早已列出，M0 未接入）；其余用临时脚本，全文写进 review 附录，可重跑，后续 Phase 重跑相关场景；P3 同时拦截 `**/auth/**` 和 `**/api/**`；`test` 任务和脚本保留（第 8 节） |
| I2 首页的最近访问没有保留入口 | 成立。首页正文由 `HomeStore` 驱动，`RecentActivityWidget` 只经 `DashboardWidgets` 渲染 | 新增 3.14：首页固定显示问候、无项目空状态、最近访问和 peek，直接挂载；删小组件 store、管理面板和个性化请求。运行时核对覆盖有数据和空状态两种情况 |
| I3 多选是有引用、但整条被禁用的链 | 成立。`useBulkOperationStatus = () => false`，列表、表格、甘特图仍透传 `selectionHelpers`；涉及 24 个文件 | 3.9 改为明确裁定：P3 用一个任务删除批量操作、工作项 `MultipleSelectStore`、`useMultipleSelect`、选择框和 `selectionHelpers` 透传；通知和表单的多选下拉保留 |
| I4 关键词验收混用正则引擎，有误报和漏查 | 成立。`grep -E` 遇到 `(?<!…)` 报错退出（exit 2）；`TExtended`、`TAdditional` 命中保留的侧边栏和筛选组件 | 7.4 改为进仓库的"关键词守卫"（P1 实现，持续集成运行）：统一用 JavaScript `RegExp`，报错即失败；例外精确到规则、路径和符号，并有到期点，过期的例外报错；覆盖根目录的锁文件和工作区配置；规则随删除加入。种子规则去掉会误报的前缀，补上团队的确切枚举值、模板、工时、重复工作项和导览；更新日志的规则单独一行。总体设计 7.6、8.3 同步 |
| I5 实例保留字段和字段交接有缺口 | 成立。`is_workspace_creation_disabled` 有 6 处使用方，`file_size_limit` 有 1 处；差异清单删除了 `users.is_bot`、`bot_type`、`api_tokens.user_type` | 3.7 改为"删除 / 保留 / 等待重新定义"表：两个字段保留，维护页保留；3.11 改为种子清单，由 P2、P3 从实际改动导出完整清单写进 handoff，并补上 `is_bot` 等字段的读取方、`file_assets.page_id`、项目的 `estimate_id` 等字段和 M6 的估算字段 |
| I6 删掉自动补斜杠不等于地址统一不带斜杠 | 成立。顶部通知的跳转目标带 `/`，当前项的判断用的是 `includes("/notifications/")` | 4.1 定下地址约定：内部生成的地址不带结尾 `/`，当前项的判断对两种形式结果相同；按相等、`includes`、`startsWith`、正则逐个列出，范围包括应用栏和顶部通知；接口地址的尾斜杠不在其中 |
| M1 完成条件冲突、计数口径 | 成立。前端改动清单写着"oxlint 为零"；`applications.*` 实测 161 个叶子键；`pnpm -r ls` 还会列出根包 `nerve` | 前端改动清单的验收标准改为按基线；数字改为 11 条、161 个键；包名核对允许 `nerve` 和 `web`；构建的文件数只记口径 |
| M2 "编辑器包只改一次"与分期矛盾 | 成立。编辑器类型中的 AI 参数挂在协作文档编辑器上 | 3.1 和 P2 的顺序：编辑器包里的 AI 部分随 P2 的"编辑器收敛"任务删除，P3 只处理应用层的 AI；删掉"只改一次"的说法 |
| S1 来源、版权和体积的记录 | 成立 | 3.10：P5 记录 propel 锁定版本的完整性哈希、许可证和对应源码的位置，"并入源码"改写为需要先确认源码版本的备选；第 5 节：新文件和新图形按实际来源标注版权；7.6：P1 记录分类的体积基线，收尾对比 |

### 10.2 对第 6 节各条裁定意见的处理
- **3.1**：采纳。`UniqueID`（所有编辑器都安装，也用于节点定位）、`copyMarkdownToClipboard`（描述历史在用）、用户 @提及、图片和节点都明确保留；编辑器只去掉 `provider`。
- **3.2**：采纳。补上成员数据加载中、加载失败、目标不是成员几种状态；访客的最终规则交给 M3，M1 不扩大也不永久缩小访问范围。
- **3.5**：采纳。去掉 P3 交付物中重复的 3.5。
- **3.12**：按 C1 改写。
- **3.13**：采纳。写明最终的改邮箱能力由项目负责人在 M2 决定，M2 设计中要作为决策点提出。
- **其余**：3.3、3.4、3.6、3.8、3.10、3.11 的补充意见都已写进对应小节：
  - 进度的快照和乐观更新；
  - M2 必须删掉 CSRF；
  - 评论的 `access` 不是评论权限；
  - propel 的来源条件；
  - 按实际改动导出交接清单。
- **"都不涉及后端"的说法过满**：采纳。设计开头改为"M1 不做任何后端工作，其中几项会约束后续 M 的接口或产品能力"，并逐项写明负责的 M。

### 10.3 第 8 节需要项目负责人决策的事项

| 事项 | 处理 |
|---|---|
| 首页删完后显示什么 | 按报告的推荐定下（3.14）：固定显示最近访问等内容。这是保留总体设计 1.1 的"最近访问"，不是新的产品取舍 |
| 访客能看到的个人主页 | M1 保持现有的访问边界（3.2），最终规则由 M3 决定 |
| 没有邮件后如何修改登录邮箱 | M1 只删旧流程；最终能力留给项目负责人在 M2 决定（3.13） |
| 复制工作区邀请链接 | 新增 3.15：M1 不删，由 M3 设计邀请接口时一起决定 |
| 实例的两个保留字段 | M1 保留（3.7），最终名称和策略由 M2、M5 决定 |
| 长期保留 `@makeplane/propel` | 保留 0.3.0 并记录来源（3.10），已向项目负责人报备 |
| 固定侧边栏、只按数量计算进度、删除评论可见范围、CSRF 的时机 | 保持原方向 |

工作项模板、定时 / 重复工作项、工时记录都作为企业空壳或死文案删除（2.2），列入关键词守卫的种子规则。

### 10.4 同步的文档
- [M1 设计](../M1-design.md)：按以上各条修订。
- 总体设计 7.6、8.3：新增关键词守卫；vitest 在 M1/P1 接入。
- 前端改动清单第二节：
  - 首页一行写明固定显示的内容；
  - 旧重定向一行改为 11 条及删除的时机；
  - 企业版残留一行补上批量操作及多选、模板和工时的空壳；
  - 验收标准改为按基线。
