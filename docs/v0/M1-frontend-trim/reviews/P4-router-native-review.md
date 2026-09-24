# M1/P4 去掉 Next.js 兼容层：评审记录

| 项 | 内容 |
|---|---|
| Phase | M1/P4 `router-native` |
| 日期 | 2026-09-24 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P4-router-native.md) / [plan](../plans/P4-router-native.md)（控制者评审补充在 plan 的 Tasks 之前，`6693e6c`） |
| 分支 | `worktree-m1-p4-router-native`，从 `main` 的 `9437a6e` 分出，提交从 `4280748` 到本评审记录所在的提交 |
| 评审 | 各 Task 评审（sonnet，第 2 节）；整分支评审（opus，`9437a6e..23e580f`）：Ready to merge with fixes；修复轮后的范围复核（sonnet）：Approved，没有新的发现 |

## 1. 范围与结果

P4 让 web 应用只用 React Router 自己的写法：去掉 Next.js 兼容层和前端环境变量，路由参数按真实的类型处理，渲染时跳转改为 `<Navigate replace />`，去掉强制结尾 `/`，删除 Plane 的旧地址重定向，并加一个进仓库的路由匹配测试（M1 设计 4.1、4.2、3.12）。

| Task | 去掉的 | 改成的 |
|---|---|---|
| 1 | 各包对环境变量的读取；`constants/src/endpoints.ts`；服务的基础地址 | 接口一律用相对路径（同源部署）；停用账户的提示改为"请联系管理员"；i18n 读 `import.meta.env.DEV` |
| 2 | dotenv、`define: { "process.env": … }`、`.env.example`、`turbo.json` 的 `VITE_*` | 守卫 `frontend-env`；顶层 `phase` 改为 `M1/P4` |
| 3 | `next/link`、`useRouter` / `useAppRouter`、`usePathname`、垫片的 `useSearchParams`、跳转的 `setTimeout` 延迟；包装里放过 `//host` 的 `isValidURL` | React Router 的 `Link to`、`useLocation`、`useSearchParams`、`useNavigate`；认证包装的 6 处 `<Navigate replace />`；`isValidNextPath`（带单元测试） |
| 4 | `next/navigation` 的 `useParams`、垫片、类型声明和 Vite 别名 | 路由页头从布局取参数；38 个共享组件按种类处理（guard、render、pass、widen、type、delete）；页头里恒为真的判断删掉 |
| 5 | 10 条旧地址重定向和它们的模块 | 5 个入口改为正式地址（停用功能页的按钮、命令面板的"账户设置"）；`app/routes/navigation.test.ts` |
| 6 | 强制结尾 `/` 的中间件、`ensureTrailingSlash`、`app/compat/next/`、`nextjs.json`、设置的 12 个没有调用方的 `highlight` | 应用里的地址不带 `/`；当前项一律用 `useMatch`、`matchPath`、`NavLink` |
| 7 | 对字符串的空转换 `.toString()`：web 699 处，constants 10 处，utils 2 处；router store 的 Node 类型 `ParsedUrlQuery` | 参数的类型是 React Router 给的 `string \| undefined`；删除留下的无效回退值、`x: x`、`x ? x : undefined` 一并收掉 |
| 修复轮 | 未修剪的 `next_path`；router store 的 4 个没有读取方的 getter；约 26 个别名、2 个退化的三元式；唯一的相对跳转目标；`!.env.example` | 包装跳到它校验过的值；三处自动跳转替换当前地址 |

数字（P3 结束 `9437a6e` → P4 结束 `6f190a1`）：

| 项 | P3 结束 | P4 结束 |
|---|---|---|
| 关键词守卫 | 42 条规则、4 条例外，顶层 `phase` 为 `M1/P3` | **45 条规则**、4 条例外（`M3`、`M6`、2 条 `M9`），顶层 `phase` 为 `M1/P4` |
| `make lint-web` / `make test-web` / `make build-web` | 52 / 15 / 11 | 52 / **16** / 11 |
| `pnpm exec turbo run check:types` | 23 | 23 |
| vitest 测试 | 63（constants 11、editor 16、i18n 10、services 13、utils 13） | **77**（web 5、constants 11、editor 16、i18n 10、services 11、utils 24） |
| lint 上限合计 | 711 | 711，不变 |
| `make knip` | 门禁，零 | 零 |
| 每种语言的文案键 / 无引用的键 | 1617 / 498 | 1617 / 498，不变 |
| 环境变量的读取（`process.env`、`VITE_*`、`dotenv`，行数） | 27 | **0** |
| 兼容层的引用（`next/link`、`next/navigation`、`useAppRouter`、`ensureTrailingSlash`、`compat/next`，行数） | 396（269 个文件） | **0** |
| 渲染时跳转（`render.mjs`） | 6 | **0** |
| 路由参数的类型错误（改为 React Router 的类型时） | 102 处，43 个文件 | 0；没有新增的非空断言和 `as string` |
| 内部跳转目标（`targets.mjs`）/ 以 `/` 结尾的 | 351 / 39 | 343 / **0** |
| 路径字面量（`pathlits.mjs`）/ 落不到页面的 | 228 / 9 | 210 / 4（都是后面还要拼接的前缀；多出的 1 个是修复轮写成绝对地址的设置链接） |
| 以 `/` 结尾的路径字面量（`slashes.mjs`） | 62 | 1（`utils/url.ts` 的 `"//"`，不是路径） |
| web 应用里的 `.toString()` | 729 | 30（都不是字符串：boolean 11、number 5、`string \| null` 4、对象和其他 7、宽联合 1、`?.` 保护后续的链 1、类型断言 1） |
| 旧地址重定向 | 10 条路由、10 个模块 | 0 |
| 构建体积 js | 422 个 / 6,881,280 字节 | 397 个 / 6,849,609 字节 |
| 构建体积 css / fonts / other | 3 / 296,988；25 / 3,755,608；112 / 7,910,515 | 3 / 296,988；25 / 3,755,608；112 / 7,910,263 |
| 最大 chunk / 语言 chunk | `use-parse-editor-content` 1,379,329 / 34 | `use-parse-editor-content` 1,379,322 / 34 |

- **改动规模**：`9437a6e..6f190a1`（不含本评审记录所在的提交）共 **423 个文件改动，+3190/−2602 行，20 个文件被删除、5 个新增**（`git diff --shortstat`）。不算文档，是 417 个文件、+1933/−2597 行；新增的 5 个文件是 spec、plan、路由测试、`vitest.config.ts` 和 `next-path.test.ts`。
- **依赖**：catalog 删 `dotenv`；web 删 `dotenv`、加开发依赖 `vitest`；i18n 加开发依赖 `vite`（catalog 版本）；锁文件删 1 个包（`dotenv`），与 spec 2.2 的预期相同。
- **保留行为的核对**（M1 设计 7.5 的 P4 一行、4.2）：
  - 进仓库的测试：路由匹配测试 5 个，`isValidNextPath` 的测试 11 个（附录 6.1）；
  - 临时核对脚本：A 组（登录状态、`next_path`、后退和替换）、B 组（重跑 P2、P3 的 5 段脚本，个人设置页的命令面板）、C 组（当前项、地址、片段、旧地址、自动跳转）。在 `6f190a1` 的构建上：9 段脚本共 403 项全部通过（第 9 项 106、第 10–12 和 15 项 69、A 组 61、命令面板 15、P2 探测 1–3 为 37 / 17 / 24、P3 探测 1–2 为 46 / 28），没有未列出的请求、未捕获的页面错误或 `navigate()` 的警告（附录 6.2–6.4）；
  - 反向对照：在基点 `9437a6e` 的构建上跑 C 组，失败的正好是 P4 改变的行为（附录 6.5）。
- **文档**：`docs/v0/frontend-changes.md` 1.4 节和第二节"Next.js 兼容垫片"一行（Task 6，本评审提交补上 Task 7 和修复轮）；M0 的两份交接（Task 6）；spec 的几处说法和第 7 节（本评审提交）；M1 设计 12 节 P4 一行在本评审提交中更新。

### spec 第 4 节验收标准

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | 每个 Task 一个提交，`check:types` 23、`make lint-web` 52、`make build-web` 11，`make knip` 零 | **满足**。7 个 Task 提交（`cd732ee`、`acdffc9`、`c7a871b`、`b12e88b`、`397a54c`、`16d88b5`、`c347ebd`），Task 4 的跟进提交 `5a209d5`，Task 7 的跟进提交 `d180574`、`f6d0ef0`、`23e580f`，修复轮的 5 个提交；每个提交结束时门禁都通过 |
| 2 | `make lint-web`：45 条规则、4 条例外；警告数等于上限（711）；`alts.mjs M1/P4` 为 0 | **满足**。每个提交结束时每个包的警告数等于上限，合计 711；修复轮一度多出的一条在同一个提交里改掉（第 4 节裁定 3）；`alts.mjs M1/P4` 为 0 |
| 3 | `make test-web` 16 个任务；vitest 66 个；stderr 只有编辑器的 5 行 | **满足**，vitest 77 个（多出的 11 个是 `isValidNextPath` 的测试，第 4 节裁定 3）。web 的路由测试、utils 的测试 stderr 为 0 字节 |
| 4 | 没有 `process.env`、`VITE_*`、`dotenv`；`.env.example` 不存在；开发服务器没有 "process is not defined" | **满足**。整分支评审 grep：只剩 `import.meta.env.DEV` / `PROD`；Task 2 之后开发服务器上实测为 0 |
| 5 | 兼容层的文件和引用不存在 | **满足** |
| 6 | `render.mjs` 为 0；包装只用 `<Navigate replace />` | **满足**。修复轮之后包装读一次 `next_path`、修剪后校验和跳转（Q2） |
| 7 | 没有新增的非空断言和 `as string`；每个参数守卫写明种类和挂载证据 | **满足**。Task 4 的 38 个组件按种类处理；Task 7 与修复轮删掉了 router store 唯一的 `as TProfileViews` 断言 |
| 8 | `routes/core.ts` 没有旧地址重定向；路由测试 5 个通过，改回旧目标时失败 | **满足**。Task 5、6 都做过"改坏再恢复"；修复轮让测试也排除 `.test.tsx`，并把唯一的相对目标改为绝对地址 |
| 9 | `targets.mjs --summary` 的 `trailing 0`；当前项没有按文本比较整个地址的判断 | **满足**。343 个目标，0 个以 `/` 结尾。2.7 表里的每一处都改为 `useMatch`、`matchPath` 或 `NavLink`；Task 6 的 slash matrix 203 行 0 处不同；C 组第 9 项两种形式的结果相同 |
| 10 | 孤儿和约定导出的核对 | **满足**。`symref`、`keyref` 为 0，`headers.sh`、`dangling` 没有输出；`route-exports.mjs 9437a6e HEAD` 只列删除的文件和 `app/layout.tsx` 的 `clientMiddleware` |
| 11 | 控制者的浏览器核对全部通过，写进附录；S1–S4 通过；持续集成的 `server`、`web`、`e2e` 通过 | **满足**。浏览器核对见附录 6。分支推送后，持续集成 run 35974880927 在 `6f190a1` 上通过（合并后在 `main` 上再跑一次）：`server` 38 秒，`web` 109 秒（Lint 一步 55 秒，Unit tests 12 秒，Unused code 3 秒），`e2e` 105 秒（S1–S4 所在的 E2E 一步 15 秒） |
| 12 | `frontend-changes.md`、M0 的两份交接同步；M1 设计 12 节更新 | **满足** |

## 2. 各 Task 的评审

- 实现者和修复轮都用 Opus 5.5，各 Task 的评审者用 sonnet，整分支评审用 opus。
- 原型（`$P4TMP/proto`，spec 2.2）只作为对照答案，不整体搬运：每个 Task 逐步执行，每一处与原型的差异都在报告里说明。
- Task 3、4 各改一百多个文件，评审包按"机械替换 / 逐处判断"拆成两份（控制者评审补充第 4 条）。机械那一份由控制者先在基点的干净克隆上重放脚本，确认提交的差异与脚本自己写出的差异逐文件相同，评审者因此只需判断每一类替换的意思是否不变。
- 每份报告回答六个风险点：渲染时跳转、查询参数和片段、参数守卫不掩盖真实的缺参、路由模块的约定导出、共享可变状态、带与不带结尾 `/` 时的当前项。

| Task | 内容 | 提交 | 核实方式 |
|---|---|---|---|
| 1 | 各包不再读环境变量 | `cd732ee` | 实现者 DONE（49 个文件，+36/−291）。环境变量读取 27 → 15（剩下的 15 处由 Task 2 的 `define` 之前的顺序决定，spec 2.2 结论 1）；services 的测试 13 → 11，去掉的只是基础地址的行为。与原型 `479609f` 的差异：停用账户的提示写成普通字符串（控制者评审补充第 3 条）；`normalizeAPIRequestURL`、`getFileURL` 的注释措辞。<br>**裁定**：god-mode 规则命中样本里的 `web/packages/constants/src/endpoints.ts` 保留——命中样本可以指向已删的代码（它说明规则抓什么，路径仍匹配文件正则），只有不命中样本必须引用保留的代码。<br>评审 Approved，所有核对钉在 `cd732ee` 上（当时 Task 2 已在同一工作区开始） |
| 2 | 去掉 `process.env` 注入、dotenv 和 `.env.example` | `acdffc9` | 实现者 DONE（8 个文件），与原型 `6fad945` 在这 8 个文件上相同。守卫 43 条规则（新增 `frontend-env`，按文件范围排除 Node 端的 `e2e/`、`tools/`）、顶层 `phase` 改为 `M1/P4`。开发服务器上"process is not defined"为 0。<br>**观察**：开发服务器打印 66 条"Module "path"/"fs"/"url"/"source-map-js" has been externalized for browser compatibility"，基点和原型上相同，与 P4 无关，交 M1 收尾的依赖梳理。<br>评审 Approved。Minor：根目录 `.gitignore` 的 `!.env.example` 在唯一的 `.env.example` 删除后成了死配置。**裁定**：进修复轮；`.env`、`.env.*` 保留，防止提交本地密钥 |
| 3 | `Link`、location、search params 和 `navigate`；渲染时跳转改为 `<Navigate>` | `c7a871b` | 实现者 DONE_WITH_CONCERNS（140 个文件，+545/−563）。codemod 改 129 个文件；渲染时跳转 6 → 0（认证包装的 6 处都改为 `<Navigate … replace />`）；守卫 45 条规则。控制者评审补充的两处修正：`next_path` 改用 `@plane/utils` 的 `isValidNextPath`（删掉放行 `//host`、`javascript:` 的本地 `isValidURL`），并为它文档里的 9 个例子加了单元测试（utils 13 → 22）；两个错误链接的空模板字符串写成 `""`。<br>实现者的三个疑问与**裁定**：<br>- 没有结尾 `/` 的目标会经过布局的 308 中间件，中间件用不带片段的 `request.url` 重定向，`#sub-issues` 在 Task 6 之前丢失：暂时现象，Task 6 删中间件后 C 组核对片段；这个片段本身没有元素读取，交 M4；<br>- `isValidNextPath` 检查修剪后的值，包装却跳到未修剪的值（`next_path=+/foo` 落到站内的"页面不存在"，不是开放重定向）：包装必须跳到它验证过的那个值，进修复轮，探测加 A3.6；<br>- 链接不再被补上 `/`（外部链接、附件地址）：正确，采纳。<br>评审包拆成两份：<br>- 机械部分（125 个文件）：控制者在 `acdffc9` 的干净克隆上重放 codemod，改的是同样的 129 个文件，122 个文件的差异与提交逐字节相同，另 3 个只差 oxfmt 的换行。评审者逐类核对替换的意思，列出 17 处"跳转之后还有代码"的调用点，去掉 `setTimeout` 延迟后都不改变行为（数据路由的 `navigate` 本身是异步的，不会同步卸载组件）。Approved，两条 Minor：codemod 的两个分支在这批文件里没有真实用例（不是缺陷）；`leave-project-modal.tsx` 先跳转再调用离开项目的接口，这是 Plane 原有的顺序，交 M3。<br>- 手改部分（15 个文件）：6 个 `<Navigate>` 与基点的 6 处重定向一一对应；命令面板两个上下文的构造处都传 `navigate`；两条新守卫规则有真实的命中和不命中样本。Approved，没有发现。<br>控制者 A 组核对：见附录 |
| 4 | 路由参数取自 React Router | `b12e88b`、`5a209d5` | 实现者 DONE_WITH_CONCERNS（198 个文件，+493/−423）。类型错误 102 → 63 → 0，三份日志与原型逐字节相同；挂载证据（`t4-mounts.txt`）与备份相同；38 个共享组件按种类处理（guard 6、render 15、guard+render 1、pass 7、widen 4、type 3、delete 2），没有新增的 `as string` 或非空断言。与原型 `e168fdc` 的唯一差异：`workspaceInfoBySlug` 的 JSDoc 随放宽的类型改为 `string \| undefined`。<br>**裁定**：<br>- 页头改为从布局取参数之后，页头里对必填路由参数的判断恒为真（本 Task 造成 5 处，Shim 下类型上已恒真的约 20 处）：同一位实现者在跟进提交 `5a209d5` 里删除，共 25 处，`.toString()` 留给 Task 7；<br>- spec 2.5 和第 3 节第 4 条说基点在个人设置页打开的工作项弹窗"提交时抛错"，不成立：实现者读出提交只会提前返回（`base.tsx:339`），表单随后清空；控制者的 B 组探测发现命令面板在那里根本不提供"新建工作项"（直接打开和从工作区进入都一样，`n` `i` 也打不开），因为这个命令要求有当前工作区。弹窗的 render 守卫保留（它让类型成立，入口本来就不存在），文字在本评审提交中更正，M4 的交接改为"弹窗挂在个人设置页上但没有入口"；<br>- 三个页头 `= props` 上方的 `// router` 注释保留：值仍是路由参数。<br>评审包拆成两份：<br>- 机械部分（135 个文件）：控制者在 `c7a871b` 的干净克隆上重放 `imports.mjs`，132 个只被它改动的文件与提交逐字节相同，正是实现者列出的那一组。评审者确认 Shim 的 `useParams` 原来就是直接转发，这些文件的运行时值不变；约 60 处插值路由参数的地方都有上游的判断、可选链、同名的 prop 或路由嵌套保证。Approved；Minor 一条（命名），不处理。<br>- 逐处判断部分（63 个文件）加跟进提交：12 对布局 / 页头传的正好是页头原来读的参数，都是必填段（`globalViewId` 除外，类型为可选）；`return null` 都在最后一个 hook 之后；弹窗的 `moduleId` 真值表与基点相同；跟进提交删掉的每一处都是必填段的 prop。Approved，没有发现。评审者提到未改动的 `projects-list.tsx` 里基点就有的 `={"JOINED"}`：全仓库有二十多个文件带这种写法，都是基点的，**裁定**交 M1 收尾统一处理 |
| 5 | 每个内部跳转落在页面上；删除旧地址重定向；路由匹配测试 | `397a54c` | 实现者 DONE_WITH_CONCERNS（21 个文件，+195/−208），本 Task 自己的差异与原型 `a3a957b` 只差 `vitest.config.ts` 的一行注释。落不到页面的路径字面量 228 / 9 → 217 / 5；路由测试 4 个，连跑 4 次 stderr 为 0 字节；故意改坏两个目标，测试恰好列出这两处，恢复后差异逐字节相同。<br>实现者的两个问题与**裁定**：<br>- 在 vitest 4.1.11、React Router 8.3.0、knip 6.37.0 上实测，没有 `vitest.config.ts` 时 vitest 加载 `vite.config.ts`、运行 React Router 插件，测试仍然通过，knip 也通过：spec 2.6 给的理由和原型的教训 4 不成立。文件保留（它让应用的构建插件不进测试），注释改为实话；spec 的措辞在本评审提交中更正；<br>- 5 个只有一段的旧地址（`/sign-in`、`/signin`、`/login`、`/register`、`/profile`）匹配 `/:workspaceSlug`，已登录时显示工作区包装的"Workspace not found"，未登录时先去 `/?next_path=…`：这是路由表对任何未知单段地址的如实回答，采纳，C 组第 12 项按此断言。`login`、`register` 不在 `RESTRICTED_URLS` 里：交 M3，保留的工作区名应该正好是应用的顶层路由段。<br>评审 Approved。评审者从 `397a54c` 的 `git archive` 实际运行了路由测试（常量那一个测试按源码推理，因为 Task 6 当时正在改常量）：扫描 1311 个源文件（`app/` 131、`core/` 1177、`helpers/` 3），找到 249 个目标；自己重做了"改坏再恢复"，失败的正好是同样两处；测试只做同步的文件读取和集合判断，结果与文件顺序无关。Minor：`sourceFiles` 排除 `.test.ts` 和 `.d.ts`，没有排除 `.test.tsx`（目前一个也没有）。**裁定**：本 Phase 新写的代码，进修复轮 |
| 6 | 去掉强制结尾 `/`，当前项用 React Router 匹配；垫片的最后部分 | `16d88b5` | 实现者 DONE（56 个文件：删 2、改 54，+164/−249，与原型 `270f8e6` 相同；除哈希和行号外，代码与原型逐字节相同，文档有 7 行不同）。以 `/` 结尾的跳转目标 35 → 0（合计 343）；以 `/` 结尾的路径字面量只剩 `url.ts` 的 `"//"`；路径字面量 209 / 4（都是前缀）；web 的路由测试 5 个，"改坏再恢复"与 Task 5 相同；没有带 `end` 的 `NavLink`；`lintdiff` 只有 `project/root.tsx` 的两行（教训 7）；构建体积 js 397 个 / 6,858,461 字节，比原型多 183 字节（`isValidNextPath` 进了 utils 的 `dist`，减去 `5a209d5` 删掉的判断）。<br>风险点 6 的证据：实现者的 `slash-matrix.mjs` 把每个新的当前项判断交给安装的 `matchPath` 和 `NavLink` 的规则，对每个样本地址的两种形式各算一次，再与基点对带 `/` 形式的判断比较：203 行，0 处不同，真假两种答案都有。<br>实现者更正了原型文档里 7 行不符合本分支的文字，采纳：dotenv 也离开了 web 的开发依赖；`layout.preload.tsx` 列为 Next.js 的遗留；旧地址重定向一行（按钮叫"管理功能"、命令面板叫"转到账号设置"、旧目标带工作区段、测试的真实范围）；垫片一行记下 `isValidNextPath`；`.env` 一行里停用账户的措辞；M0-P5 交接的第 1 项；M0-P6 交接注明 P3 已把 knip 改为门禁。<br>评审 Approved，没有发现。评审者钉在 `16d88b5` 上核对：<br>- `layout.tsx` 没有中间件和多余的导入，`app/compat/next/` 为空；<br>- 在 `app/`、`core/`、`helpers/`、`constants` 里扫以 `/` 结尾的字面量，只有 `password.tsx` 表单的 `action`（接口地址，保留）；<br>- 2.7 的每一行都读了源码：hook 都在条件 `return` 之前（`sidebar-item` 在权限判断的 `return null` 之前调用 `useMatch`）；首页、项目 `end: true`，设置的"常规"精确匹配、其余按前缀；`project-navigation` 的 `matchPath` 模式由真实的参数值拼成，不含 `:` 或 `*`；<br>- `slash-matrix` 里 `NavLink` 的规则与 react-router 8.3.0 的 `lib/dom/lib.js` 逐字节相同；`tab-navigation-root` 删掉的跳转从来不会执行（没有路由服务项目根地址）；新加的"不以 `/` 结尾"断言不是空断言。<br>评审者还指出 `use-workspace-paths.ts` 唯一的调用方是 `app-rail-hoc.tsx`，随应用栏一起是死的（第 4 节裁定 9，交收尾）。<br>控制者在 `16d88b5` 的构建上提前跑了一次 C 组：除 A3.6（交修复轮）外全部通过 |
| 7 | 删掉对字符串的空转换 `.toString()` | `c347ebd`、`d180574`、`f6d0ef0`、`23e580f` | 没有原型（控制者评审补充第 1 条）。实现者 DONE_WITH_CONCERNS：`c347ebd`（203 个文件，+563/−784：web 202 个，加守卫的一个样本行）。脚本先建带类型检查的程序（ts-morph，TS 5.9，4322 个文件，0 个错误）。之前 729 处（219 个文件）；删掉 671 处（202 个文件）：接收者为 `string` 的 `x.toString()` 467、`x?.toString()` 71，`string \| undefined` 且在链尾的 `x?.toString()` 133；留下 58 处。`verify.mjs`：455 对，0 处不符，673 处调用（另 2 处在守卫样本里）；三份故意改坏的差异各报 1 处不符（反向对照）。<br>**裁定**：<br>- 两处偏离字面规则，采纳，因为规则的本意是"没有任何表达式的值改变"：`x?.toString().<后续>` 在 `?.` 保护后面的链时保留（实现者第一次运行把 `workItem?.toString().split("-")[0]` 改成了 `workItem.split(…)`，TS18048，改正规则后在干净的树上重跑）；经类型断言得到的接收者保留（`security.tsx` 的 `err.error_code?.toString()`：断言说是字符串，Plane 发的是数字，转换才让枚举查找成立，交 M2）；<br>- 守卫 `bulk-operations` 的不命中样本改为 `add-project-members-modal.tsx:86` 的现状（不命中样本引用保留的代码）；<br>- 留下的 58 处里有 28 处，根因是 `router.store.ts` 的 `query` 仍是 Node 的 `ParsedUrlQuery`（Next.js 的遗留，实际由 React Router 的参数填入）和两个 `*-issue-properties` hook 的 `string \| string[] \| undefined` 参数。这是 P4 的范围，由同一位实现者做跟进提交。<br>跟进提交：<br>- A `d180574`：`router.store.ts` 改用本地的 `TRouteParams = Record<string, string \| undefined>`（store 不导入 react-router，`StoreWrapper` 传入的 `Readonly<Params<string>>` 可以赋给它），两个 hook 的 13 个参数改为 `string \| undefined`，逐个核对调用方，没有传数组的；<br>- B `f6d0ef0`：在新类型下重跑，删掉 40 处（router store 12、两个 hook 16、constants 的 `fetch-keys.ts` 10、utils 的 `emoji.ts` 2），`verify.mjs` 32 对，0 处不符。编辑器 callout 的 `logo-selector.tsx:34` 保留：属性声明为 `string`，运行时是数字（TipTap 用 `fromString` 解析它，propel 的 `stringToEmoji` 在 `try` 之外调用 `.split`，删掉转换会让标注块的图标崩溃），脚本加了 `RUNTIME_NOT_STRING` 表，之前的 671 处重新核对，没有来自 TipTap 节点属性的；<br>- C `23e580f`：收掉删除留下的结构。15 处对 `string` 的 `?? ""` / `\|\| ""`；19 处 `x: x` 改为简写；30 处 `x ? x : undefined\|null` 中 `x` 是 React Router 路由参数的（脚本核对绑定和导入）：28 处改为按原名解构、删掉别名，2 处改为 `x ?? null`。<br>最后留下的：web 30 处（boolean 11、number 5、`string \| null` 4、对象和其他 7、宽联合 1、`?.` 保护后续的链 1、类型断言 1），packages 8 处；各处干跑都删不掉任何一处。<br>评审 Approved，没有 Critical 或 Important：<br>- 在 `23e580f` 上干跑 `tostring.mjs`（0 个类型错误、0 处可删），按字节复现 `verify.mjs` 的数字，并把一个新增行里的变量改名做反向对照，报出不符；<br>- 从声明、收窄追到运行时的值，逐个核对约 40 处，全部成立；跟进 C 的 64 处全部核对（解构没有遮蔽，简写只有 `x: x`，被删的回退值左边是非联合的 `string`）；四个提交都没有动任何依赖数组。<br>两条 Minor 都是记账（评审包 `review-t7-follow.diff` 的范围是 `c347ebd..23e580f`，比标签多含提交 A；"packages 8"包含 propel 的一处 `toString(16)`，它带参数，规则本来就不会碰），不处理。<br>控制者在 `23e580f` 的构建上正式跑 C 组：与 Task 6 的构建结果相同，Task 7 没有改变探测看得到的行为 |
| 修复轮 | 整分支评审的 Q1–Q4、M1–M4（第 3 节） | `cdad109`、`b13d2a2`、`192be3e`、`3f5d781`、`6f190a1` | 实现者 DONE（5 个提交，17 个文件，+45/−102）。第 4 个提交做到一半时网络断开，会话重启后原样恢复同一位实现者（它从自己的记录继续），没有丢失工作，也没有用 `git restore`。每个提交都跑了 `check:types`、孤儿核对（基点 `23e580f`）、守卫、`alts`、`render.mjs`、`lintdiff` 和上限。<br>- 提交 1：包装读一次 `next_path` 并修剪，校验和跳转用同一个值；`next-path.test.ts` 加制表符、换行两个例子（utils 24）。登录、注册表单的隐藏字段原样提交给服务端，客户端不校验也不跳转，不改（交 M2）；<br>- 提交 2：删掉 4 个没有读取方的 getter（连同接口成员、`makeObservable` 条目、`as TProfileViews` 断言和它的导入），8 处 `this.query?.` 改为 `this.query.`。证据：`git grep -P` 在 HEAD 和基点都找不到读取方，同一模式对在用的参数命中 16 个文件（正向对照）；`useRouterParams()` 唯一的调用方只用 `setQuery`；<br>- 提交 3：三处自动跳转加 `{ replace: true }`，逐处写明触发条件；语法树扫描确认从 effect 里调用的 `navigate` 正好 4 处，现在全是替换；收集箱的标签点击仍是 `push`；<br>- 提交 4：25 个别名、2 个退化的三元式，外加实现者扫描发现的同形 `module_ids`；<br>- 提交 5：`no-projects.tsx` 的绝对地址；路由测试的文件过滤（`.d.ts` 用 `endsWith`，正则写法会多一条 `prefer-string-starts-ends-with`，web 超过上限）；`.gitignore`；控制者读提交 3 时加的一处：浏览页跳转里 `if (data?.is_intake)` 之内的 `data?.id` 改为 `data.id`。<br>最后：门禁 23 / 52 / 16 / 11，knip 零，vitest 77，跳转目标 343 / 0，路径字面量 210 / 4（多出的 1 个是新的绝对地址，落在页面上），上限 711 不变。<br>**裁定**：多收的 `module_ids`、`endsWith` 的写法、`moduleId ?? ""` 保留回退（组件挂在 6 个没有 `:moduleId` 的路由上）都采纳；实现者列出的基点写法（`viewId as TProfileViews`、3 个导入常量的别名、一个没有插值的模板字面量）交收尾（第 7 节）。<br>范围复核：见 3.1 |

## 3. 整分支评审（opus）：Ready to merge with fixes

评审范围 `9437a6e..23e580f`，结果为 0 Critical、1 Important、7 Minor，另外核对了控制者排进修复轮的 4 项（Q1–Q4）。评审者读了整个 Phase 的差异（按核心、非核心和 `-U0` 拆开），以及 `AuthenticationWrapper`、`api.service.ts` 的 401 处理、`isValidNextPath` 和它的测试、`router.store.ts`、2.7 表里的每一处当前项判断、Task 7 的别名所在处。评审者复跑了 spec 第 4 节的工具，并写了两个新的只读脚本：

- `hooks-after-return.mjs`（TypeScript 语法树）：在全部 web 文件里找"提前 `return` 之后才调用的 hook"，结果为 0；它自带的样本文件证明它能发现这种写法；
- `ternary.mjs`：新增行里的退化结构，找到 2 处（M2）。

评审者的结论是：没有开放重定向（React Router 的 `encodeLocation` 让每个通过校验的 `next_path` 都留在站内，React 19 拦下 `javascript:` 的 `href`），没有渲染时跳转，没有提前 `return` 之后的 hook，没有兼容层的残留，没有依赖结尾 `/` 的当前项判断。

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| I1 | Important | spec 7.1 给 M2 的一行过时：还写着 `isValidURL` 只拒绝 `http`、`https`、`ftp`（Task 3 已换成 `isValidNextPath`）；漏了服务端那一半：登录、注册表单把地址里的原值提交给 `/auth/sign-in/`、`/auth/sign-up/`，由服务端跳转 | 本评审提交重写这一行，并写进 M2 的交接：服务端用同一条规则校验 `next_path` |
| Q1 | 控制者 | 根目录 `.gitignore` 的 `!.env.example` 在唯一的 `.env.example` 删除后成了死配置（Task 2 评审） | 已修（`6f190a1`） |
| Q2 | 控制者，评审者扩大 | 包装跳到未修剪的 `next_path`，而 `isValidNextPath` 校验的是修剪后的值。评审者指出开头的制表符、换行也能通过：`%20` 落到站内的"页面不存在" `/%20/probe-ws/projects`；控制者的探测又发现 `%0A`、`%09` 落到 `/projects`，因为未修剪的值不以 `/` 开头，React Router 把它当相对路径，得到 `/\n/probe-ws/projects`，解析地址时去掉换行，`//probe-ws/projects` 被读成主机名加 `/projects`。只取路径，所以仍在站内，但地址是错的 | 已修（`cdad109`）：读一次、修剪，校验和跳转用同一个值；`next-path.test.ts` 加制表符、换行两个例子（utils 22 → 24）。A3.6–A3.8 通过 |
| Q3 | 控制者 | 路由测试的 `sourceFiles` 没有排除 `.test.tsx`（Task 5 评审） | 已修（`6f190a1`）：排除 `\.(test\|spec)\.tsx?$` 和 `\.d\.ts$` |
| Q4 | 控制者，评审者改进 | `router.store.ts` 的 `profileViewId` 用 `as TProfileViews` 断言，getter 里的 `this.query?.` 是死的 `?.`。评审者发现 `profileViewId`、`peekId`、`issueId`、`inboxId` 四个 getter 在基点和现在都没有读取方 | 已修（`b13d2a2`）：删掉四个 getter（断言随之消失）和死的 `?.` |
| M1 | Minor | Task 7 留下约 26 个只转存一下的别名（`const workspaceSlug = routerWorkspaceSlug;`，`use-issues-actions.tsx` 有 8 组） | 已修（`3f5d781`）：实现者用脚本扫全部 `const a = b;`，55 处里 25 处是 Task 7 写下的别名，全部删掉，参数按原名解构；其余 30 处是 Plane 基点的写法，大多是有意的角色名，只转存导入常量的 3 处交收尾 |
| M2 | Minor | 新增行里两处退化的三元式（`moduleId != undefined ? moduleId : ""`、`data?.cycle_id ? data?.cycle_id : …`） | 已修（`3f5d781`）；实现者扫本 Phase 全部新增行，又找到同形的 `module_ids` 一处，一并收掉。`moduleId ?? ""` 的回退保留：组件挂在 6 个没有 `:moduleId` 的路由上 |
| M3 | Minor | `no-projects.tsx` 的 `link: "settings"` 是应用里唯一的相对跳转目标，路由测试看不见它（行为是对的：React Router 把它解析为 `/:ws/settings`，基点的垫片会送到坏掉的 `/settings/`） | 已修（`6f190a1`）：写成绝对地址，与旁边的一项一致 |
| M4 | Minor | 三处页面到达时自动做的跳转仍是 `push`，后退会回到马上又跳走的地址：工作项浏览页遇到收集箱中的工作项时跳到收集箱，项目设置跳到第一个项目，收集箱侧边栏跳到第一项（基点如此；`inbox/content/root.tsx` 已经是替换） | 已修（`192be3e`）：三处都改为 `{ replace: true }`。控制者加了 C15 的三项后退核对，在 `23e580f` 上都失败，修复后通过 |
| M5 | Minor | 一些文件里 `react-router` 的导入放在 `// hooks` 分组里，没有和其他外部导入放在一起 | 不改（第 4 节裁定 10） |
| M6 | Minor | `frontend-changes.md` 1.4 节没有记 Task 7（router store 的参数类型、web 包里的 `.toString()`） | 本评审提交补上 |
| M7 | Minor | `api.service.ts` 的 401 处理里 `currentPath ? … : ""` 恒为真（基点如此） | 写进 M2 的交接：令牌管理器替换这段时不要照搬 |

修复轮合计（`23e580f..6f190a1`）：5 个提交，17 个文件，+45/−102 行。每个提交都通过 `check:types`、孤儿核对、守卫、`render.mjs` 和 lint 上限；最后 `make lint-web` 52、`make test-web` 16、`make build-web` 11 个任务通过，`make knip` 为零，vitest 77 个。

### 3.1 修复后的复核（sonnet，范围复核）

修复轮 `23e580f..6f190a1` 之后，范围复核的结论是 **Approved**：Q1–Q4、M1–M4 和控制者加的 `data.id` 全部为 Fixed，没有新的发现。复核者对每一项都从源码自己取证：

- **Q2**：包装只读一次、修剪后的值，校验和跳转用它；`auth-root.tsx` 读原值只为交给 `password.tsx` 的隐藏字段，客户端不校验也不跳转（交 M2）；`isValidNextPath` 调用 `url.trim()`，会去掉制表符和换行，所以两个新测试确实覆盖了修复。
- **Q4**：自己用 `git grep -P` 在 web 和整个仓库里找四个成员，没有一处读 router store 的；`.issueId` 的命中都属于工作项详情 store 里无关的 `peekIssue`；`query` 只由 `setQuery(useParams())` 整体赋值，不会是 `undefined`。
- **M1、M2**：重跑实现者的别名扫描，剩下 30 处，与报告相同，都是基点的写法，没有一处涉及路由参数；`a ? a : b` 与 `a || b` 对任何值都相同（包括 `""` 和 `[]`），不依赖类型。重跑三元式扫描，只剩路由测试第 94 行一处误报（条件里有 `i === 0 &&`，不是退化结构）。
- **M4**：三处都在页面到达时的 `useEffect` 里触发，收集箱的标签点击仍是 `push`；复核者又在 `app/`、`core/` 里找了其他含 `navigate(` 的 effect 和处理函数，都是用户的操作（点击、提交、Esc），不需要替换。
- **复跑**：`check:types` 23 个任务，`keywords: 45 rules, 4 exceptions, no hits.`，web 的路由测试 5 个、utils 24 个，`render-time navigations: 0`；修复轮新增的行里没有 `as string`、非空断言、`={"`、`${"` 和没有插值的模板字面量。

修复轮之后，控制者在 `6f190a1` 的构建上重跑了 A、B、C 三组核对（附录 6）：9 段脚本共 403 项全部通过，其中修复轮加的 A3.7、A3.8 和第 15 项的三个场景在修复轮之前的 `23e580f` 上失败（附录 6.5），修复后通过。

## 4. 控制者的裁定

执行中的裁定都按"能自己定的就自己定"的原则做出：只要符合 SOLID、从根源解决、不打补丁，就不升级给用户。没有一项属于架构级、跨模块或意料之外的高风险，因此都没有升级。

1. **原型只是对照答案**（P3 裁定 1）。每个 Task 逐步执行，报告与原型逐文件比较。Task 2、5、6 的代码与原型相同（Task 5 只差一行注释，Task 6 只差文档）；Task 3、4 的机械部分由控制者在基点的干净克隆上重放脚本，确认提交的差异就是脚本写出的差异。与原型不同的地方都来自执行前的控制者评审（`next_path` 的校验、Task 7）或实现者发现的原型错误（第 5 节）。
2. **孤儿以前一个 Task 的真实提交为基点**（P3 裁定 2），`infile-orphans`、`symref`、`keyref`、`dangling`、`headers.sh` 每个 Task 都跑。
3. **数字按实测。** lint 上限合计 711，整个 Phase 不变（修复轮的路由测试一度多出一条 `prefer-string-starts-ends-with`，改用 `endsWith` 之后回到 566）。测试 77 个，不是 spec 写的 66 个：`isValidNextPath` 的测试 Task 3 加 9 个、修复轮加 2 个。空转换按类型数：Task 7 删掉 web 的 671 处，跟进提交再删 40 处，不是 spec 按路由参数数的 520 行。
4. **空转换 `.toString()` 在 P4 删**（控制者评审补充第 1 条，推翻 spec 第 5 节）。它是 Next.js 的 `useParams` 返回 `string | string[]` 时留下的写法，属于 P4 要去掉的兼容层习惯；Task 4 之后参数有了真实的类型，才能用类型判断哪些调用是空操作。执行中发现 router store 的 `query` 和两个 hook 的参数仍是 Next.js 时代的类型，由跟进提交改正，再在新类型下删。
5. **开放重定向从根源修**（控制者评审补充第 2 条）。包装原来的 `isValidURL` 放过 `//host`、`javascript:`；改用 `@plane/utils` 里本来就有、却没人用的 `isValidNextPath`，并为它加单元测试。修复轮让包装跳到它校验过的那个值（Q2）。服务端的校验交 M2（I1）。
6. **自动跳转一律替换当前地址。** spec 第 3 节第 2 条让守卫跳转用 `replace`；整分支评审发现另外三处页面到达时的跳转仍是 `push`（M4）。它们都不是用户的操作，后退回到的地址会马上再跳走，所以修复轮也改为替换。codemod 按规则保留了 Plane 的 `push`，这是它应该做的：语义的改变由人判断，不由脚本做。
7. **测试输出**（P3 裁定 6）：只接受编辑器包那 5 行 `prosemirror-codemark` 的 sourcemap 警告。web 的路由测试、utils 的测试 stderr 都是 0 字节。
8. **`vitest.config.ts` 保留，说法更正。** Task 5 实测，没有它时测试和 knip 也都通过，spec 2.6 和原型教训 4 的理由不成立。保留的理由是让测试不加载应用的构建插件，与编辑器包相同；文字在本评审提交中更正。
9. **从不渲染的应用栏交收尾，不在 P4 删。** 写 C 组时发现：`AppRailVisibilityProvider` 的 `isEnabled` 默认为假，唯一的使用处（工作区布局）不传它，所以应用栏从不挂载。这是一个藏在开关后面的功能，按"删除，不隐藏"应当整条删掉，但它不是路由的事，范围也要先量清（`app-rail-root.tsx`、`items-root.tsx`、`app-rail-hoc.tsx`、`core/lib/app-rail/`、布局里的 provider、`content-wrapper` 的分支、偏好的 setter、`@plane/types` 里的类型、`use-workspace-paths.ts`）。Task 6 仍按计划改了它的当前项判断，保证"每个当前项判断都用 React Router"这一句成立；C 组去掉了"应用栏的设置是当前项"一项，因为观察不到，顶部通知用的是同一个组件，已核对。
10. **导入分组不重排**（M5）。codemod 把每个 `react-router` 导入放在文件原来 `next/*` 导入的位置；为此重排约 180 个文件只是外观上的改动，读代码的人得不到好处。
11. **单段的旧地址显示"Workspace not found"**（Task 5）。`/sign-in`、`/login` 等匹配 `/:workspaceSlug`，这是路由表对任何未知单段地址的如实回答，符合 M1 设计 3.12"旧地址不保留"。保留名单与顶层路由段不一致交 M3。
12. **控制者的探测发现了评审没有发现的问题**（P3 裁定 11）：
    - B 组第 8 项证明 spec 2.5 和第 3 节第 4 条对弹窗的说法不成立（Task 4）；
    - C 组发现应用栏从不渲染（裁定 9）；
    - A3.7、A3.8 证明 Q2 不只是"落到 404"，未修剪的值会被解析到另一个站内地址；
    - C 组 13 项重跑时，P2、P3 探测里 `a[href="/probe-ws/notifications/"]` 的选择器在 Task 6 之后失效，"红点消失"的等待因此空过、计数读得太早。这是探测的问题，P4 的副本已改正（附录 6.3）。

    探测在 Task 3、4、6、7 之后和修复轮之后各跑一次，并在基点的构建上做了反向对照（附录 6.5）。
13. **执行过程。** 实现者有几次在一条 Bash 命令里串了只读的 git 命令或把 `cd` 与其他命令写在一起，harness 拦下了其中几次；控制者自己也有一次在只读查询里用 `;` 串了两个 `git diff --shortstat`。历史都没有变化。之后的派发都重申了"一条命令只做一件 git 事"。

## 5. 计划缺陷

计划由原型写成，执行时发现以下缺陷，都已在对应 Task 或修复轮中修正：

| Task | 缺陷 | 实际做法 |
|---|---|---|
| 全部 | 测试总数 66、`.toString()` 按路由参数数的 520 行 | 以实测为准（第 4 节裁定 3） |
| 2 | 删掉 `.env.example` 后，`.gitignore` 的 `!.env.example` 没有删 | 修复轮（Q1） |
| 3 | 包装校验修剪后的 `next_path`，却跳到未修剪的值 | 修复轮（Q2），探测 A3.6–A3.8 |
| 4 | spec 2.5、第 3 节第 4 条说基点在个人设置页打开的弹窗"提交时抛错"：基点只是提前返回，而且那里没有打开它的入口 | 守卫保留（让类型成立），文字更正，M4 的交接改写 |
| 4 | 页头改为从布局取参数后，页头里对必填参数的判断恒为真 | 跟进提交 `5a209d5` 删掉 25 处 |
| 5 | `vitest.config.ts` 的理由（"否则失败""knip 才把测试文件算作入口"）不成立 | 文件保留，注释、spec 2.6 和教训 4 更正 |
| 5 | 路由测试没有排除 `.test.tsx` | 修复轮（Q3） |
| 5 | `no-projects.tsx` 的相对目标 `"settings"` 路由测试看不见 | 修复轮（M3） |
| 6 | C 组"应用栏的设置是当前项"观察不到：应用栏从不渲染 | 去掉这一项，应用栏交收尾（第 4 节裁定 9） |
| 7 | 按字面规则会改掉 `?.` 保护的链（TS18048）和类型断言得到的接收者；编辑器 callout 的属性运行时是数字 | 规则按"值不变"收窄，加 `RUNTIME_NOT_STRING` 表 |
| 7 | 28 处留下的调用来自 Next.js 时代的参数类型 | 跟进提交 A、B |
| 7 | 删掉转换后留下无效的回退值、`x: x`、`x ? x : undefined`，以及约 26 个别名和 2 个退化的三元式 | 跟进提交 C；别名和三元式由修复轮收掉（M1、M2） |
| 3（codemod） | 三处页面到达时的自动跳转按 Plane 的写法保留为 `push` | 修复轮改为替换（M4），探测 C15 |
| 4（router store） | 四个没有读取方的 getter，其中一个用断言掩盖类型 | 修复轮删除（Q4） |
| 7 | 工作项弹窗 `module_ids` 的退化三元式（与 M2 同形，整分支评审没有列出） | 修复轮的实现者扫描本 Phase 全部新增行时发现，一并收掉 |

## 6. 附录（M1 设计 7.5）

临时核对脚本不进仓库（M1 设计 7.5），全文、假数据、运行命令、断言和输出写在这里。脚本都在 `$P4TMP`（`/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p4`）下，跑在 `$P4TMP/probe-app` 里：那是本仓库的一个克隆，检出到被测的提交，由 `setup-probe-app.sh <提交>` 安装依赖、执行 `make build-web`。做法沿用 P3 评审附录 6：node 起一个静态服务器提供 `web/apps/web/build/client`（单页应用的回退到 `index.html`），用 e2e 包里的 Playwright 驱动 Chromium，路径以 `/api/`、`/auth/` 开头的请求全部由脚本里的桩回答；**当前场景没有列出的请求一律算失败**，每个场景还检查"没有未捕获的页面错误"和"控制台没有 `You should call navigate() in a React.useEffect()`"（plan 的"控制者的浏览器核对"）。下面的输出都来自修复轮之后 `6f190a1` 的构建（`run-c.sh fix`），反向对照的输出来自基点 `9437a6e` 和修复轮之前的 `23e580f`。

### 6.1 进仓库的测试

- `web/apps/web/app/routes/navigation.test.ts`（Task 5 新增 4 个，Task 6 加 1 个）：读源码里写出的每个路径和导航常量，用 `matchRoutes` 匹配真实的路由表（spec 2.6）。
- `web/packages/utils/src/next-path.test.ts`（Task 3 新增 9 个，修复轮加 2 个）：`isValidNextPath` 的每个例子各一个测试。

在 `6f190a1` 上的输出（`pnpm --dir web/apps/web exec vitest run --reporter=verbose`，utils 包同样，它的输出也包括原有的 13 个测试；路径前缀缩写为 `<repo>`）：

```

 RUN  v4.1.11 <repo>/web/apps/web

 ✓ app/routes/navigation.test.ts > internal navigation > finds the paths written in the web app 0ms
 ✓ app/routes/navigation.test.ts > internal navigation > lands every path written in the web app on a page 47ms
 ✓ app/routes/navigation.test.ts > internal navigation > writes every in-app path without a trailing slash 0ms
 ✓ app/routes/navigation.test.ts > internal navigation > lands every entry of the navigation constants on a page, without a trailing slash 4ms
 ✓ app/routes/navigation.test.ts > internal navigation > sends a path that no page serves to page not found 2ms

 Test Files  1 passed (1)
      Tests  5 passed (5)
   Start at  16:18:27
   Duration  1.42s (transform 107ms, setup 0ms, import 1.31s, tests 54ms, environment 0ms)
```

```

 RUN  v4.1.11 <repo>/web/packages/utils

 ✓ src/next-path.test.ts > isValidNextPath > accepts a path on this site 1ms
 ✓ src/next-path.test.ts > isValidNextPath > accepts a path with several segments 0ms
 ✓ src/next-path.test.ts > isValidNextPath > accepts a path surrounded by whitespace 0ms
 ✓ src/next-path.test.ts > isValidNextPath > accepts a path after a tab 0ms
 ✓ src/next-path.test.ts > isValidNextPath > accepts a path after a newline 0ms
 ✓ src/next-path.test.ts > isValidNextPath > rejects an absolute address 0ms
 ✓ src/next-path.test.ts > isValidNextPath > rejects a protocol-relative address 0ms
 ✓ src/next-path.test.ts > isValidNextPath > rejects a javascript: address 0ms
 ✓ src/next-path.test.ts > isValidNextPath > rejects an empty string 0ms
 ✓ src/next-path.test.ts > isValidNextPath > rejects a path that does not start with a slash 0ms
 ✓ src/next-path.test.ts > isValidNextPath > rejects a path that starts with a backslash 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > is zero without a cycle 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > is zero for an empty cycle 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > leaves the cancelled work items out of the total 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > counts the started work items as well when asked to 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > reports 100 once everything that was not cancelled is done 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > reports zero when every work item was cancelled 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > prefers the snapshot of a completed cycle over the live counts 0ms
 ✓ src/progress.test.ts > the optimistic count update > adds a new work item to the total and to its own state group 1ms
 ✓ src/progress.test.ts > the optimistic count update > moves the count between state groups when a work item is completed 0ms
 ✓ src/progress.test.ts > the optimistic count update > moves it back when the work item is reopened 0ms
 ✓ src/progress.test.ts > the optimistic count update > takes a removed work item out of the total 0ms
 ✓ src/progress.test.ts > the optimistic count update > keeps the assignee and label counts of a module in step 0ms
 ✓ src/progress.test.ts > the optimistic count update > takes the completed work item off the burn-down chart on the day it was completed 0ms

 Test Files  2 passed (2)
      Tests  24 passed (24)
   Start at  16:18:29
   Duration  534ms (transform 61ms, setup 0ms, import 502ms, tests 5ms, environment 0ms)
```

### 6.2 A 组：几种登录状态的跳转、`next_path`、后退和替换（plan 第 1–6 项）

#### 1. 目的

核对 `AuthenticationWrapper` 的 6 处跳转改为 `<Navigate replace />` 之后，每种登录状态落在正确的地址，被挡住的地址不留在历史里；`next_path` 只接受站内路径，前后带空白、制表符或换行的站内路径跳到修剪后的地址；`navigate(-1)` 的"返回"和删除 Webhook 之后的替换。第 6 项（退出登录以表单提交 `/auth/sign-out/`）由 B 组重跑的 P3 认证脚本第 9 项核对。

#### 2. 共用部分（`$P4TMP/probe-a/lib.mjs`，A、B、C 三组都用它）

```js
// One-off (M1/P4 browser checks, plan "控制者的浏览器核对", M1 design 4.2 "运行时核对"): what the P4 probe scripts
// share. `probe(run)` serves the built web app (web/apps/web/build/client, SPA fallback to its index.html) on a free
// port, starts Chromium and calls `run`; each `scenario` gets a fresh browser context in which every request whose
// path starts with /api/ or /auth/ goes to the scenario's stubs. A request the scenario does not list gets 404 and
// fails its "no unlisted request" check; every scenario also fails on an uncaught page error and on React Router's
// "You should call navigate() in a React.useEffect()" warning.
// Addresses are compared without a trailing slash; until Task 6 removes the forced one, scenarios open addresses
// with it (SLASH=/, the default) because the base's slash redirect adds a history entry. After Task 6: SLASH= STRICT=1.
// Run the probes from the repository root of the build (Playwright comes from e2e/).
// `node <probe>.mjs <text>` runs only the scenarios whose name contains <text>.
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");

// ---------------------------------------------------------------- fake data
export const WS = "probe-ws";
export const ME = {
  id: "u1", email: "probe@example.com", display_name: "probe", first_name: "Probe", last_name: "User",
  avatar_url: "", cover_image_url: null, is_active: true,
};
export const PROJECT = {
  id: "p1", name: "Probe Project", identifier: "PRB", sort_order: 65535, logo_props: {}, member_role: 20,
  archived_at: null, workspace: "w1", cycle_view: true, module_view: true, issue_views_view: true, inbox_view: true,
};
export const WORKSPACE = { id: "w1", slug: WS, name: "Probe WS", total_members: 1, role: 20, owner: ME };
const CSRF = "probe-csrf-token";
const config = { enable_signup: true, is_workspace_creation_disabled: false, file_size_limit: 5242880, is_self_managed: true };

export class Reply {
  constructor(status, json, headers = {}) {
    Object.assign(this, { status, json, headers });
  }
}

export const anonymousStubs = () => ({
  "GET /api/instances/": { config },
  "GET /api/users/me/": new Reply(401, { error_code: 5000, error_message: "AUTHENTICATION_FAILED" }),
  "GET /auth/get-csrf-token/": { csrf_token: CSRF },
});
// A signed-in user; `onboarded` and `workspaces` vary per scenario.
export const signedInStubs = ({ onboarded = true, workspaces = [WORKSPACE] } = {}) => ({
  "GET /api/instances/": { config },
  "GET /api/users/me/": ME,
  "GET /api/users/me/profile/": {
    id: "pr1", user: "u1", language: "en", is_onboarded: onboarded, is_tour_completed: true, theme: {},
    onboarding_step: { profile_complete: onboarded, workspace_create: onboarded, workspace_invite: onboarded, workspace_join: onboarded },
  },
  "GET /api/users/me/settings/": {
    id: "u1", email: ME.email,
    workspace: {
      last_workspace_id: workspaces.length ? "w1" : null, last_workspace_slug: workspaces.length ? WS : null,
      fallback_workspace_id: workspaces.length ? "w1" : null, fallback_workspace_slug: workspaces.length ? WS : null,
      invites: 0,
    },
  },
  "GET /api/users/me/workspaces/": workspaces,
  "GET /auth/get-csrf-token/": { csrf_token: CSRF },
});
// What the workspace layout, the sidebar and the settings pages of probe-ws ask for.
export const workspaceStubs = () => ({
  [`GET /api/workspaces/${WS}/workspace-members/me/`]: { id: "wm-u1", member: "u1", workspace: "w1", role: 20, is_active: true },
  [`GET /api/users/me/workspaces/${WS}/project-roles/`]: { p1: 20 },
  [`GET /api/workspaces/${WS}/projects/`]: [PROJECT],
  [`GET /api/workspaces/${WS}/projects/details/`]: [PROJECT],
  [`GET /api/workspaces/${WS}/members/`]: [{ id: "wm-u1", member: ME, role: 20, is_active: true }],
  [`GET /api/workspaces/${WS}/states/`]: [],
  [`GET /api/workspaces/${WS}/user-favorites/`]: [],
  [`GET /api/workspaces/${WS}/user-properties/`]: { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 },
  [`GET /api/workspaces/${WS}/users/notifications/unread/`]: { total_count: 0, mention_unread_notifications_count: 0 },
  // the workspace home
  [`GET /api/workspaces/${WS}/recent-visits/`]: [],
});
// What the cycles list of project p1 asks for.
export const projectStubs = () => {
  const p = `/api/workspaces/${WS}/projects/${PROJECT.id}`;
  return {
    [`GET ${p}/`]: PROJECT,
    [`GET ${p}/cycles/`]: [],
    [`GET ${p}/project-members/me/`]: { id: "pm1", member: "u1", role: 20, original_role: 20 },
    [`GET ${p}/members/`]: [{ id: "pm1", member: "u1", role: 20, original_role: 20 }],
    [`GET ${p}/user-properties/`]: { preferences: { navigation: { default_tab: "work_items", hide_in_more_menu: [] } } },
    [`GET ${p}/issue-labels/`]: [],
    [`GET ${p}/states/`]: [],
    [`GET ${p}/intake-state/`]: {},
    [`GET ${p}/modules/`]: [],
    [`GET ${p}/views/`]: [],
    [`GET /api/workspaces/${WS}/modules/`]: [],
  };
};

// ---------------------------------------------------------------- static server over the build
const root = path.resolve("web/apps/web/build/client");
const shell = path.join(root, "index.html");
const types = {
  ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json",
  ".svg": "image/svg+xml", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".jpg": "image/jpeg",
  ".ico": "image/x-icon", ".woff": "font/woff", ".woff2": "font/woff2", ".ttf": "font/ttf",
};
const server = http.createServer((req, res) => {
  let file = path.join(root, decodeURIComponent(new URL(req.url, "http://x").pathname));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = shell;
  res.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream" });
  fs.createReadStream(file).pipe(res);
});

// ---------------------------------------------------------------- checks
const results = [];
export const check = (name, ok, detail = "") => {
  results.push({ name, ok });
  console.log(ok ? `PASS ${name}` : `FAIL ${name}: ${detail}`);
};
export const visible = async (locator, timeout = 15000) => {
  try {
    await locator.first().waitFor({ state: "visible", timeout });
    return true;
  } catch {
    return false;
  }
};
const NAVIGATE_WARNING = /navigate\(\) in a React\.useEffect/;
// `SLASH=` (empty) after Task 6; until then "/".
export const SLASH = process.env.SLASH ?? "/";
// STRICT=1 (group C item 13, after Task 6): an address must match exactly — a trailing slash is a failure, not noise.
export const STRICT = process.env.STRICT === "1";
export const strip = (p) => (STRICT ? p : p.replace(/(.)\/$/, "$1"));
export const here = (page) => {
  const url = new URL(page.url());
  return { path: strip(url.pathname), next: url.searchParams.get("next_path"), search: url.search };
};

let browser;
export let base;
const only = process.argv[2];
export async function scenario(name, stubs, fn) {
  if (only && !name.includes(only)) return;
  console.log(`\n== ${name}`);
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "en-US" });
  const page = await context.newPage();
  const ctx = { requests: [], unstubbed: [], pageErrors: [], navigateWarnings: [] };
  page.on("pageerror", (e) => ctx.pageErrors.push(e.message.split("\n")[0]));
  page.on("console", (m) => {
    if (NAVIGATE_WARNING.test(m.text())) ctx.navigateWarnings.push(m.text().split("\n")[0]);
  });
  await context.route(
    (url) => url.pathname.startsWith("/api/") || url.pathname.startsWith("/auth/"),
    async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      const key = `${request.method()} ${url.pathname}`;
      ctx.requests.push({ key, search: url.search, body: request.postData() });
      let stub = stubs[key];
      if (typeof stub === "function") stub = stub(request, ctx);
      try {
        if (stub === undefined) {
          ctx.unstubbed.push(key);
          return await route.fulfill({ status: 404, json: {} });
        }
        const reply = stub instanceof Reply ? stub : new Reply(200, stub);
        if (reply.json === undefined) return await route.fulfill({ status: reply.status, headers: reply.headers });
        return await route.fulfill({ status: reply.status, json: reply.json, headers: reply.headers });
      } catch {
        // the context closed while the reply was pending
      }
    }
  );
  try {
    await fn(page, ctx);
    check(`${name}: no request outside the scenario's list`, ctx.unstubbed.length === 0, JSON.stringify([...new Set(ctx.unstubbed)]));
    check(`${name}: no uncaught page error`, ctx.pageErrors.length === 0, JSON.stringify(ctx.pageErrors));
    check(`${name}: no "call navigate() in a React.useEffect()" warning`, ctx.navigateWarnings.length === 0, JSON.stringify(ctx.navigateWarnings));
  } catch (error) {
    check(`${name}: ran to the end`, false, error.message.split("\n")[0]);
  } finally {
    await context.close();
  }
}
export const goto = (page, p) => page.goto(`${base}${p}`, { waitUntil: "domcontentloaded" });
// Waits until the address (without a trailing slash) is `want`, then prints where the page landed.
export const settle = async (page, want, timeout = 20000) => {
  const ok = await page
    .waitForURL((u) => strip(u.pathname) === want, { timeout })
    .then(() => true, () => false);
  await page.waitForTimeout(500);
  const url = new URL(page.url());
  console.log(`  (at ${url.pathname}${url.search})`);
  return ok;
};

export function probe(run) {
  server.listen(0, "127.0.0.1", async () => {
    base = `http://127.0.0.1:${server.address().port}`;
    browser = await chromium.launch();
    try {
      await run();
    } finally {
      await browser.close();
      server.close();
    }
    const failed = results.filter((r) => !r.ok).length;
    console.log(`\n${results.length - failed} passed, ${failed} failed`);
    process.exitCode = failed ? 1 : 0;
  });
}
```

#### 脚本全文（`$P4TMP/probe-a/redirects.mjs`）

```js
// One-off (M1/P4 browser checks, group A: plan "控制者的浏览器核对"): the redirects of AuthenticationWrapper for each
// sign-in state, next_path (including the unsafe values it must ignore), going back and replacing. The server, the
// stubs and the per-scenario checks are in lib.mjs.
// Run from the repository root of the build. usage: node redirects.mjs [scenario name filter]
import {
  PROJECT, Reply, SLASH, WS, anonymousStubs, base, check, goto, here, probe, projectStubs, scenario, settle, signedInStubs,
  strip, visible, workspaceStubs,
} from "./lib.mjs";

const WEBHOOK = {
  id: "wh1", url: "https://hooks.example.com/probe", is_active: true, project: true, issue: true, module: false,
  cycle: false, issue_comment: false, created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z",
};
const signInFormShown = (page) => visible(page.locator("form input#email"), 20000);

probe(async () => {
  // A1. Signed out, a workspace address sends you to sign in with next_path, and the guarded address is replaced:
  //     going back returns to the page before it, not to the address that would redirect again.
  await scenario("signed out: a workspace address", anonymousStubs(), async (page) => {
    await goto(page, `/sign-up${SLASH}`);
    await visible(page.locator("form input#email"));
    await goto(page, `/${WS}/projects${SLASH}`);
    const landed = await settle(page, "/");
    const { next } = here(page);
    check("A1.1 lands on / with next_path=/probe-ws/projects", landed && strip(next ?? "") === `/${WS}/projects`, `${page.url()}`);
    check("A1.2 shows the sign-in form", await signInFormShown(page));
    await page.goBack({ waitUntil: "domcontentloaded" });
    const back = await settle(page, "/sign-up", 10000);
    check("A1.3 going back returns to /sign-up (the guarded address was replaced)", back, page.url());
  });

  // A2. Not onboarded: a workspace address goes to /onboarding. Signed out: /onboarding goes to sign in.
  await scenario("signed in, not onboarded", {
    ...signedInStubs({ onboarded: false }),
    ...workspaceStubs(),
    "GET /api/users/me/workspaces/invitations/": [],
  }, async (page) => {
    await goto(page, `/${WS}/projects`);
    check("A2.1 a workspace address goes to /onboarding", await settle(page, "/onboarding"), page.url());
  });
  await scenario("signed out: /onboarding", anonymousStubs(), async (page) => {
    await goto(page, "/onboarding");
    const landed = await settle(page, "/");
    check("A2.2 /onboarding goes to / with next_path=/onboarding", landed && strip(here(page).next ?? "") === "/onboarding", page.url());
  });

  // A3. Onboarded: / goes to the last workspace; a safe next_path is followed; unsafe ones are ignored.
  const onboarded = { ...signedInStubs(), ...workspaceStubs() };
  const cases = [
    ["A3.1 / goes to the last workspace", "/", `/${WS}`],
    ["A3.2 a site path in next_path is followed", `/?next_path=/${WS}/projects`, `/${WS}/projects`],
    ["A3.3 an absolute URL in next_path is ignored", "/?next_path=https://example.com/x", `/${WS}`],
    ["A3.4 a protocol-relative URL in next_path is ignored", "/?next_path=//example.com/x", `/${WS}`],
    ["A3.5 a javascript: URL in next_path is ignored", `/?next_path=${encodeURIComponent("javascript:alert(1)")}`, `/${WS}`],
    // isValidNextPath accepts a path with surrounding spaces (it checks the trimmed value); the app must go to that
    // trimmed path, not to the untrimmed text (fix wave, Task 3 concern 2).
    ["A3.6 a site path with a leading space in next_path is followed", `/?next_path=%20/${WS}/projects`, `/${WS}/projects`],
    // the same with a leading newline and tab (final review, Q2 widened)
    ["A3.7 a site path with a leading newline in next_path is followed", `/?next_path=%0A/${WS}/projects`, `/${WS}/projects`],
    ["A3.8 a site path with a leading tab in next_path is followed", `/?next_path=%09/${WS}/projects`, `/${WS}/projects`],
  ];
  for (const [name, from, to] of cases) {
    await scenario(name, onboarded, async (page) => {
      await goto(page, from);
      const ok = await settle(page, to);
      check(name, ok && new URL(page.url()).origin === base, page.url());
    });
  }

  // A4. Onboarded without a workspace: / goes to /create-workspace.
  await scenario("onboarded, no workspace", {
    ...signedInStubs({ workspaces: [] }),
    "GET /api/users/me/workspaces/invitations/": [],
  }, async (page) => {
    await goto(page, "/");
    check("A4.1 / goes to /create-workspace", await settle(page, "/create-workspace"), page.url());
  });

  // A5. Going back (navigate(-1)) and replacing (navigate(x, { replace: true })).
  await scenario("back and replace", {
    ...onboarded,
    [`GET /api/workspaces/${WS}/webhooks/`]: [WEBHOOK],
    [`GET /api/workspaces/${WS}/webhooks/${WEBHOOK.id}/`]: WEBHOOK,
    [`DELETE /api/workspaces/${WS}/webhooks/${WEBHOOK.id}/`]: new Reply(204),
    "GET /api/users/me/workspaces/invitations/": [],
  }, async (page) => {
    // The create-workspace form's "Go back" button goes back one entry. Until Task 6 the addresses carry the trailing
    // slash: the base's trailing-slash redirect adds a history entry, so going back from an address opened without
    // it lands on the slashless address, which redirects forward again (group C checks that this is gone).
    await goto(page, `/${WS}/settings/webhooks${SLASH}`);
    await settle(page, `/${WS}/settings/webhooks`);
    await goto(page, `/create-workspace${SLASH}`);
    await settle(page, "/create-workspace");
    await page.getByRole("button", { name: /go back/i }).click();
    check("A5.1 \"Go back\" returns to the previous page", await settle(page, `/${WS}/settings/webhooks`), page.url());
    // Deleting a webhook replaces its page with the list: going back does not return to the deleted webhook.
    await page.getByRole("link", { name: /hooks\.example\.com/ }).first().click();
    await settle(page, `/${WS}/settings/webhooks/${WEBHOOK.id}`);
    await page.getByRole("button", { name: "Danger zone" }).click();
    await page.getByRole("button", { name: "Delete webhook" }).click();
    await page.getByRole("dialog").getByRole("button", { name: /delete/i }).click();
    check("A5.2 deleting the webhook lands on the list", await settle(page, `/${WS}/settings/webhooks`), page.url());
    await page.goBack({ waitUntil: "domcontentloaded" });
    await page.waitForTimeout(1000);
    check("A5.3 going back does not return to the deleted webhook", !here(page).path.endsWith(WEBHOOK.id), page.url());
  });

  // The breadcrumb's back ("…", shown only at 640 px or narrower) goes back one entry.
  await scenario("breadcrumb back", {
    ...onboarded,
    ...projectStubs(),
  }, async (page) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await goto(page, `/${WS}/projects${SLASH}`);
    await settle(page, `/${WS}/projects`);
    await goto(page, `/${WS}/projects/${PROJECT.id}/cycles${SLASH}`);
    await settle(page, `/${WS}/projects/${PROJECT.id}/cycles`);
    const back = page.locator("span.text-secondary", { hasText: /^\.\.\.$/ });
    check("A5.4 the breadcrumb shows its back item at phone width", await visible(back));
    // At phone width the app sidebar starts open and closes on a mouse down outside it, which moves the header
    // ~217 px left between mouse down and up, so that first click is lost (the same on the base build). Close the
    // sidebar with a click on the page's empty lower right first.
    const before = await back.first().boundingBox();
    await page.mouse.click(380, 800);
    await page.waitForTimeout(500);
    const after = await back.first().boundingBox();
    console.log(`  the item moved from x=${before?.x} to x=${after?.x} when the sidebar closed`);
    await back.first().click();
    check("A5.4 the breadcrumb's back returns to the previous page", await settle(page, `/${WS}/projects`), page.url());
  });
});
```

#### 3. 假数据

一个已登录、已完成引导的用户 `u1`，工作区 `probe-ws`，项目 `p1`（`lib.mjs` 的 `signedInStubs`、`workspaceStubs`、`projectStubs`）；未登录时 `/api/users/me/` 回答 401；A2 的用户未完成引导；A4 的用户没有工作区；A5 另有一个 Webhook。

#### 4. 运行命令

```bash
cd $P4TMP/probe-app   # 检出到 6f190a1，已 make build-web
SLASH= STRICT=1 node $P4TMP/probe-a/redirects.mjs
```

`SLASH=`、`STRICT=1`：Task 6 之后地址一律不带结尾 `/` 打开，比较地址时结尾 `/` 算失败（plan 第 13 项）。Task 3、4 之后的运行用默认的 `SLASH=/`，因为基点的中间件会把地址补上 `/` 并多出一条历史记录。

#### 5. 输出

```

== signed out: a workspace address
  (at /?next_path=/probe-ws/projects)
PASS A1.1 lands on / with next_path=/probe-ws/projects
PASS A1.2 shows the sign-in form
  (at /sign-up)
PASS A1.3 going back returns to /sign-up (the guarded address was replaced)
PASS signed out: a workspace address: no request outside the scenario's list
PASS signed out: a workspace address: no uncaught page error
PASS signed out: a workspace address: no "call navigate() in a React.useEffect()" warning

== signed in, not onboarded
  (at /onboarding)
PASS A2.1 a workspace address goes to /onboarding
PASS signed in, not onboarded: no request outside the scenario's list
PASS signed in, not onboarded: no uncaught page error
PASS signed in, not onboarded: no "call navigate() in a React.useEffect()" warning

== signed out: /onboarding
  (at /?next_path=/onboarding)
PASS A2.2 /onboarding goes to / with next_path=/onboarding
PASS signed out: /onboarding: no request outside the scenario's list
PASS signed out: /onboarding: no uncaught page error
PASS signed out: /onboarding: no "call navigate() in a React.useEffect()" warning

== A3.1 / goes to the last workspace
  (at /probe-ws)
PASS A3.1 / goes to the last workspace
PASS A3.1 / goes to the last workspace: no request outside the scenario's list
PASS A3.1 / goes to the last workspace: no uncaught page error
PASS A3.1 / goes to the last workspace: no "call navigate() in a React.useEffect()" warning

== A3.2 a site path in next_path is followed
  (at /probe-ws/projects)
PASS A3.2 a site path in next_path is followed
PASS A3.2 a site path in next_path is followed: no request outside the scenario's list
PASS A3.2 a site path in next_path is followed: no uncaught page error
PASS A3.2 a site path in next_path is followed: no "call navigate() in a React.useEffect()" warning

== A3.3 an absolute URL in next_path is ignored
  (at /probe-ws)
PASS A3.3 an absolute URL in next_path is ignored
PASS A3.3 an absolute URL in next_path is ignored: no request outside the scenario's list
PASS A3.3 an absolute URL in next_path is ignored: no uncaught page error
PASS A3.3 an absolute URL in next_path is ignored: no "call navigate() in a React.useEffect()" warning

== A3.4 a protocol-relative URL in next_path is ignored
  (at /probe-ws)
PASS A3.4 a protocol-relative URL in next_path is ignored
PASS A3.4 a protocol-relative URL in next_path is ignored: no request outside the scenario's list
PASS A3.4 a protocol-relative URL in next_path is ignored: no uncaught page error
PASS A3.4 a protocol-relative URL in next_path is ignored: no "call navigate() in a React.useEffect()" warning

== A3.5 a javascript: URL in next_path is ignored
  (at /probe-ws)
PASS A3.5 a javascript: URL in next_path is ignored
PASS A3.5 a javascript: URL in next_path is ignored: no request outside the scenario's list
PASS A3.5 a javascript: URL in next_path is ignored: no uncaught page error
PASS A3.5 a javascript: URL in next_path is ignored: no "call navigate() in a React.useEffect()" warning

== A3.6 a site path with a leading space in next_path is followed
  (at /probe-ws/projects)
PASS A3.6 a site path with a leading space in next_path is followed
PASS A3.6 a site path with a leading space in next_path is followed: no request outside the scenario's list
PASS A3.6 a site path with a leading space in next_path is followed: no uncaught page error
PASS A3.6 a site path with a leading space in next_path is followed: no "call navigate() in a React.useEffect()" warning

== A3.7 a site path with a leading newline in next_path is followed
  (at /probe-ws/projects)
PASS A3.7 a site path with a leading newline in next_path is followed
PASS A3.7 a site path with a leading newline in next_path is followed: no request outside the scenario's list
PASS A3.7 a site path with a leading newline in next_path is followed: no uncaught page error
PASS A3.7 a site path with a leading newline in next_path is followed: no "call navigate() in a React.useEffect()" warning

== A3.8 a site path with a leading tab in next_path is followed
  (at /probe-ws/projects)
PASS A3.8 a site path with a leading tab in next_path is followed
PASS A3.8 a site path with a leading tab in next_path is followed: no request outside the scenario's list
PASS A3.8 a site path with a leading tab in next_path is followed: no uncaught page error
PASS A3.8 a site path with a leading tab in next_path is followed: no "call navigate() in a React.useEffect()" warning

== onboarded, no workspace
  (at /create-workspace)
PASS A4.1 / goes to /create-workspace
PASS onboarded, no workspace: no request outside the scenario's list
PASS onboarded, no workspace: no uncaught page error
PASS onboarded, no workspace: no "call navigate() in a React.useEffect()" warning

== back and replace
  (at /probe-ws/settings/webhooks)
  (at /create-workspace)
  (at /probe-ws/settings/webhooks)
PASS A5.1 "Go back" returns to the previous page
  (at /probe-ws/settings/webhooks/wh1)
  (at /probe-ws/settings/webhooks)
PASS A5.2 deleting the webhook lands on the list
PASS A5.3 going back does not return to the deleted webhook
PASS back and replace: no request outside the scenario's list
PASS back and replace: no uncaught page error
PASS back and replace: no "call navigate() in a React.useEffect()" warning

== breadcrumb back
  (at /probe-ws/projects)
  (at /probe-ws/projects/p1/cycles)
PASS A5.4 the breadcrumb shows its back item at phone width
  the item moved from x=284.59375 to x=67.59375 when the sidebar closed
  (at /probe-ws/projects)
PASS A5.4 the breadcrumb's back returns to the previous page
PASS breadcrumb back: no request outside the scenario's list
PASS breadcrumb back: no uncaught page error
PASS breadcrumb back: no "call navigate() in a React.useEffect()" warning

61 passed, 0 failed
```

### 6.3 B 组：参数（plan 第 7、8 项）

#### 1. 目的

第 7 项：Task 4 改了这些页面取参数的地方，重跑 P2 评审附录的探测 1–3 和 P3 评审附录的探测 1–2，结果应与 P3 结束时相同。第 8 项：在个人设置页打开命令面板，没有未捕获的错误，也打不开工作项弹窗（spec 第 3 节第 4 条）。

#### 2. 重跑的 P2、P3 脚本

脚本全文见 [P2 评审记录](P2-trim-content-review.md)附录和 [P3 评审记录](P3-trim-platform-review.md)附录 6.2、6.3。Task 3 之后链接不再被补上 `/`，Task 6 之后应用里的地址都不带 `/`，所以其中 4 个用 P4 的副本跑（第 13 项），副本与原文的差别只在地址的写法：

P2 探测 1 的副本（`$P4TMP/probe-b/p2probe1.mjs`，由 `adapt-p2probe1.py` 改写）与原文的差异：

```diff
--- nerve-p2/probe-1/home-profile-sidebar.mjs
+++ nerve-p4/probe-b/p2probe1.mjs
@@ -8,2 +8,4 @@
 // usage: node home-profile-sidebar.mjs
+// M1/P4 copy (probe-b/p2probe1.mjs): link hrefs are selected and compared without the forced trailing slash (the
+// Link shim is gone since P4 Task 3); STRICT=1 (group C, after Task 6) accepts only the slashless form.
 import { createRequire } from "node:module";
@@ -15,2 +17,6 @@
 const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
+const STRICT = process.env.STRICT === "1";
+// A link by its address: without the trailing slash, and until Task 6 also with it.
+const byHref = (h) => (STRICT ? `a[href="${h}"]` : `a[href="${h}"], a[href="${h}/"]`);
+const hrefForm = (h) => (h === "/" || STRICT ? h : h.replace(/\/$/, ""));
 
@@ -191,3 +197,3 @@
     );
-    const workItem = main.locator(`a[href="/${WS}/browse/PRB-7/"]`);
+    const workItem = main.locator(byHref(`/${WS}/browse/PRB-7`));
     const workItemOk = (await visible(workItem)) && /PRB-7[\s\S]*Probe work item/.test(await workItem.innerText());
@@ -195,3 +201,3 @@
       await main.innerText());
-    const project = main.locator(`a[href="/${WS}/projects/p1/issues"]`);
+    const project = main.locator(byHref(`/${WS}/projects/p1/issues`));
     const projectOk = (await visible(project)) && /PRB[\s\S]*Probe Project/.test(await project.innerText());
@@ -215,10 +221,11 @@
       as.map((a) => ({ href: a.getAttribute("href"), text: a.innerText.trim() })));
-    const navItems = anchors.filter((a) => !a.href.startsWith(`/${WS}/projects/p1`));
+    const hrefs = anchors.map((a) => ({ ...a, href: hrefForm(a.href) }));
+    const navItems = hrefs.filter((a) => !a.href.startsWith(`/${WS}/projects/p1`));
     const expected = [
-      { href: `/${WS}/`, text: "Home" },
-      { href: `/${WS}/profile/u1/`, text: "Your work" },
-      { href: `/${WS}/drafts/`, text: "Drafts" },
-      { href: `/${WS}/projects/`, text: "Projects" },
-      { href: `/${WS}/workspace-views/all-issues/`, text: "Views" },
-      { href: `/${WS}/projects/archives/`, text: "Archives" },
+      { href: `/${WS}`, text: "Home" },
+      { href: `/${WS}/profile/u1`, text: "Your work" },
+      { href: `/${WS}/drafts`, text: "Drafts" },
+      { href: `/${WS}/projects`, text: "Projects" },
+      { href: `/${WS}/workspace-views/all-issues`, text: "Views" },
+      { href: `/${WS}/projects/archives`, text: "Archives" },
     ];
@@ -233,3 +240,3 @@
     for (const item of expected) {
-      const link = sidebar.locator(`a[href="${item.href}"]`).first();
+      const link = sidebar.locator(byHref(item.href)).first();
       await link.hover();
@@ -280,3 +287,3 @@
     // 4. The sidebar's "Your work" lands on the assigned tab of one's own profile.
-    await sidebar.locator(`a[href="/${WS}/profile/u1/"]`).click();
+    await sidebar.locator(byHref(`/${WS}/profile/u1`)).first().click();
     const landed = await landsOn(page, new RegExp(`/${WS}/profile/u1/assigned/?$`));
```

#### 脚本全文（`$P4TMP/probe-c/adapt-p2p3.py`）

```py
"""One-off (M1/P4 group C item 13: "P2、P3 脚本里带结尾 `/` 的预期地址改为不带"): P4 copies of P2's probe 3 and P3's
lists and auth probes, with every app address written without the trailing slash the base forced: the addresses the
probes open, the ones they expect (regexes that tolerated `\\/?$` become exact), the inbox link they select, and the
stubbed server redirect after sign-in (it redirects to next_path, which the app now sends without the slash).
/api/ and /auth/ endpoints keep their slash. The archived P2/P3 scripts are untouched.
usage: python3 adapt-p2p3.py   (writes probe-c/p2probe3.mjs, p3lists.mjs, p3auth.mjs)"""
import pathlib
import re

here = pathlib.Path(__file__).parent
root = here.parent.parent
SOURCES = {
    "p2probe3.mjs": root / "nerve-p2/probe-3/activity-notifications-lists-probe.mjs",
    "p3lists.mjs": root / "nerve-p3/probe-lists/lists-detail-probe.mjs",
    "p3auth.mjs": root / "nerve-p3/probe-auth/auth-instance.mjs",
}
# a string or template literal holding an app address (not /api/, /auth/) that ends in "/"
LITERAL = re.compile(r'(["`])((?:\$\{base\})?/(?:probe-ws|\$\{WS\})(?:/[^"`\s]*?)?)/\1')
TOLERANT = re.compile(r"\\/\?\$/")

for name, src in SOURCES.items():
    text = src.read_text()
    lines = text.split("\n")
    changed = []
    for i, line in enumerate(lines):
        new = LITERAL.sub(lambda m: f"{m.group(1)}{m.group(2)}{m.group(1)}", line)
        new = TOLERANT.sub("$/", new)
        if new != line:
            changed.append((i + 1, line.strip(), new.strip()))
            lines[i] = new
    note = (
        f"// M1/P4 copy (probe-c/{name}, made by adapt-p2p3.py from {src.relative_to(root)}): app addresses are written\n"
        "// and expected without the trailing slash the base forced (P4 Task 6); /api/ and /auth/ endpoints keep theirs.\n"
    )
    out = lines[0] + "\n" + note + "\n".join(lines[1:]) if lines[0].startswith("//") else note + "\n".join(lines)
    (here / name).write_text(out)
    print(f"== {name}: {len(changed)} lines")
    for n, old, new in changed:
        print(f"  {n}: {old}\n   -> {new}")
```

`p2probe3.mjs` 与原文的差异：

```diff
--- nerve-p2/probe-3/activity-notifications-lists-probe.mjs
+++ nerve-p4/probe-c/p2probe3.mjs
@@ -1,2 +1,4 @@
 // One-off (M1/P2, design 7.5 row "P2、P3 动态、通知、工作项列表"): serves the built web app
+// M1/P4 copy (probe-c/p2probe3.mjs, made by adapt-p2p3.py from nerve-p2/probe-3/activity-notifications-lists-probe.mjs): app addresses are written
+// and expected without the trailing slash the base forced (P4 Task 6); /api/ and /auth/ endpoints keep theirs.
 // (web/apps/web/build/client, SPA fallback to index.html) on a free port, signs in a stubbed, onboarded user
@@ -298,3 +300,3 @@
     const { page, close } = await openPage("activity");
-    await page.goto(`${base}/probe-ws/projects/p1/issues/i1/`);
+    await page.goto(`${base}/probe-ws/projects/p1/issues/i1`);
     let section;
@@ -302,3 +304,3 @@
     await check("activity: the work item opens from its project path (redirects to /browse/PRB-1/)", async () => {
-      await page.waitForURL(/\/probe-ws\/browse\/PRB-1\/?$/, { timeout: 15000 });
+      await page.waitForURL(/\/probe-ws\/browse\/PRB-1$/, { timeout: 15000 });
       section = page.locator("div.text-h5-medium", { hasText: /^Activity$/ }).locator("xpath=../..");
@@ -337,3 +339,3 @@
     const { page, close } = await openPage("notifications");
-    const inboxDot = page.locator('a[href="/probe-ws/notifications/"] span.bg-danger-primary');
+    const inboxDot = page.locator('a[href="/probe-ws/notifications"] span.bg-danger-primary');
     const allTab = page.locator("div.cursor-pointer", { hasText: /^All/ }).first();
@@ -341,3 +343,3 @@
     const card = (identifier, title) => page.getByText(new RegExp(`^${identifier}\\s${title}$`));
-    await page.goto(`${base}/probe-ws/notifications/`);
+    await page.goto(`${base}/probe-ws/notifications`);
     await check("notifications: the page lists both stubbed notifications with readable text", async () => {
@@ -386,11 +388,11 @@
   const CONTEXTS = [
-    { name: "project", url: "/probe-ws/projects/p1/issues/", titles: [0, 1, 2, 3],
+    { name: "project", url: "/probe-ws/projects/p1/issues", titles: [0, 1, 2, 3],
       request: (r) => r.path === `${P}/issues/` && !r.query.get("cycle") && !r.query.get("module") },
-    { name: "cycle", url: "/probe-ws/projects/p1/cycles/c1/", titles: [0, 1],
+    { name: "cycle", url: "/probe-ws/projects/p1/cycles/c1", titles: [0, 1],
       request: (r) => r.path === `${P}/issues/` && r.query.get("cycle") === "c1" },
-    { name: "module", url: "/probe-ws/projects/p1/modules/m1/", titles: [0, 2],
+    { name: "module", url: "/probe-ws/projects/p1/modules/m1", titles: [0, 2],
       request: (r) => r.path === `${P}/issues/` && r.query.get("module") === "m1" },
-    { name: "profile", url: "/probe-ws/profile/u1/", finalUrl: /\/probe-ws\/profile\/u1\/assigned\/?$/, titles: [3],
+    { name: "profile", url: "/probe-ws/profile/u1", finalUrl: /\/probe-ws\/profile\/u1\/assigned$/, titles: [3],
       request: (r) => r.path === `${W}/user-issues/u1/` && r.query.get("assignees") === "u1" },
-    { name: "archives", url: "/probe-ws/projects/p1/archives/issues/", titles: [4],
+    { name: "archives", url: "/probe-ws/projects/p1/archives/issues", titles: [4],
       request: (r) => r.path === `${P}/archived-issues/` },
@@ -419,3 +421,3 @@
     const { page, close } = await openPage("layouts");
-    await page.goto(`${base}/probe-ws/projects/p1/issues/`);
+    await page.goto(`${base}/probe-ws/projects/p1/issues`);
     await page.getByText(TITLES[0], { exact: true }).first().waitFor({ timeout: 15000 });
```

`p3lists.mjs` 与原文的差异：

```diff
--- nerve-p3/probe-lists/lists-detail-probe.mjs
+++ nerve-p4/probe-c/p3lists.mjs
@@ -1,2 +1,4 @@
 // One-off (M1/P3, design 7.5 row "P2、P3 动态、通知、工作项列表", P3 spec 2.13; built on M1/P2's probe 3, whose checks
+// M1/P4 copy (probe-c/p3lists.mjs, made by adapt-p2p3.py from nerve-p3/probe-lists/lists-detail-probe.mjs): app addresses are written
+// and expected without the trailing slash the base forced (P4 Task 6); /api/ and /auth/ endpoints keep theirs.
 // 1-4 are unchanged apart from the ones marked P3): serves the built web app
@@ -322,3 +324,3 @@
     const { page, requests, close } = await openPage("activity");
-    await page.goto(`${base}/probe-ws/projects/p1/issues/i1/`);
+    await page.goto(`${base}/probe-ws/projects/p1/issues/i1`);
     let section;
@@ -326,3 +328,3 @@
     await check("activity: the work item opens from its project path (redirects to /browse/PRB-1/)", async () => {
-      await page.waitForURL(/\/probe-ws\/browse\/PRB-1\/?$/, { timeout: 15000 });
+      await page.waitForURL(/\/probe-ws\/browse\/PRB-1$/, { timeout: 15000 });
       section = page.locator("div.text-h5-medium", { hasText: /^Activity$/ }).locator("xpath=../..");
@@ -380,3 +382,3 @@
     const { page, close } = await openPage("notifications");
-    const inboxDot = page.locator('a[href="/probe-ws/notifications/"] span.bg-danger-primary');
+    const inboxDot = page.locator('a[href="/probe-ws/notifications"] span.bg-danger-primary');
     const allTab = page.locator("div.cursor-pointer", { hasText: /^All/ }).first();
@@ -384,3 +386,3 @@
     const card = (identifier, title) => page.getByText(new RegExp(`^${identifier}\\s${title}$`));
-    await page.goto(`${base}/probe-ws/notifications/`);
+    await page.goto(`${base}/probe-ws/notifications`);
     await check("notifications: the page lists both stubbed notifications with readable text", async () => {
@@ -429,11 +431,11 @@
   const CONTEXTS = [
-    { name: "project", url: "/probe-ws/projects/p1/issues/", titles: [0, 1, 2, 3],
+    { name: "project", url: "/probe-ws/projects/p1/issues", titles: [0, 1, 2, 3],
       request: (r) => r.path === `${P}/issues/` && !r.query.get("cycle") && !r.query.get("module") },
-    { name: "cycle", url: "/probe-ws/projects/p1/cycles/c1/", titles: [0, 1],
+    { name: "cycle", url: "/probe-ws/projects/p1/cycles/c1", titles: [0, 1],
       request: (r) => r.path === `${P}/issues/` && r.query.get("cycle") === "c1" },
-    { name: "module", url: "/probe-ws/projects/p1/modules/m1/", titles: [0, 2],
+    { name: "module", url: "/probe-ws/projects/p1/modules/m1", titles: [0, 2],
       request: (r) => r.path === `${P}/issues/` && r.query.get("module") === "m1" },
-    { name: "profile", url: "/probe-ws/profile/u1/", finalUrl: /\/probe-ws\/profile\/u1\/assigned\/?$/, titles: [3],
+    { name: "profile", url: "/probe-ws/profile/u1", finalUrl: /\/probe-ws\/profile\/u1\/assigned$/, titles: [3],
       request: (r) => r.path === `${W}/user-issues/u1/` && r.query.get("assignees") === "u1" },
-    { name: "archives", url: "/probe-ws/projects/p1/archives/issues/", titles: [4],
+    { name: "archives", url: "/probe-ws/projects/p1/archives/issues", titles: [4],
       request: (r) => r.path === `${P}/archived-issues/` },
@@ -462,3 +464,3 @@
     const { page, close } = await openPage("layouts");
-    await page.goto(`${base}/probe-ws/projects/p1/issues/`);
+    await page.goto(`${base}/probe-ws/projects/p1/issues`);
     await page.getByText(TITLES[0], { exact: true }).first().waitFor({ timeout: 15000 });
@@ -522,3 +524,3 @@
     STUBS[`${W}/user-favorites/f4/group/`] = []; // the folder's contents
-    await page.goto(`${base}/probe-ws/projects/p1/issues/`);
+    await page.goto(`${base}/probe-ws/projects/p1/issues`);
     await check("favourites (P3): the sidebar lists a project, view, cycle, module and folder favourite, each with an icon", async () => {
```

`p3auth.mjs` 与原文的差异：

```diff
--- nerve-p3/probe-auth/auth-instance.mjs
+++ nerve-p4/probe-c/p3auth.mjs
@@ -1,2 +1,4 @@
 // One-off (M1/P3 retained-behaviour probe 1, M1 design 7.5 row "P3 认证、实例", P3 spec 2.13): serves the built web
+// M1/P4 copy (probe-c/p3auth.mjs, made by adapt-p2p3.py from nerve-p3/probe-auth/auth-instance.mjs): app addresses are written
+// and expected without the trailing slash the base forced (P4 Task 6); /api/ and /auth/ endpoints keep theirs.
 // app (web/apps/web/build/client, SPA fallback to its index.html) on a free port and checks the sign-in and sign-up
@@ -170,3 +172,3 @@
   }, async (page, ctx) => {
-    await goto(page, `/?next_path=${encodeURIComponent(`/${WS}/projects/`)}`);
+    await goto(page, `/?next_path=${encodeURIComponent(`/${WS}/projects`)}`);
     check("1.1 / shows the sign-in form: email, password, no confirmation, \"Go to workspace\"",
@@ -196,3 +198,3 @@
       body.csrfmiddlewaretoken === CSRF && body.email === "typed@example.com" &&
-        body.password === "Wrong-password-1" && body.next_path === `/${WS}/projects/`,
+        body.password === "Wrong-password-1" && body.next_path === `/${WS}/projects`,
       JSON.stringify(body));
@@ -215,3 +217,3 @@
       Object.assign(stubs, signedInStubs());
-      return redirectTo(`/${WS}/projects/`);
+      return redirectTo(`/${WS}/projects`);
     };
@@ -219,3 +221,3 @@
   })(), async (page, ctx) => {
-    await goto(page, `/?next_path=${encodeURIComponent(`/${WS}/projects/`)}`);
+    await goto(page, `/?next_path=${encodeURIComponent(`/${WS}/projects`)}`);
     await visible(emailInput(page), 20000);
@@ -231,3 +233,3 @@
       fields(formPost(ctx, "/auth/sign-in/")[0]).csrfmiddlewaretoken === CSRF &&
-        fields(formPost(ctx, "/auth/sign-in/")[0]).next_path === `/${WS}/projects/`,
+        fields(formPost(ctx, "/auth/sign-in/")[0]).next_path === `/${WS}/projects`,
       JSON.stringify(fields(formPost(ctx, "/auth/sign-in/")[0])));
@@ -373,3 +375,3 @@
   })(), async (page, ctx) => {
-    await goto(page, `/${WS}/projects/`);
+    await goto(page, `/${WS}/projects`);
     await visible(page.getByText("Probe Project"), 20000);
```

P2 探测 2（应用中的编辑器）没有断言带 `/` 的地址，原样重跑。副本里的通知链接选择器原来是 `a[href="/probe-ws/notifications/"]`，Task 6 之后顶部通知的链接不带 `/`，原文的选择器会找不到它，"红点消失"的等待因此空过；副本改正了这一处（第 4 节裁定 12）。

#### 脚本全文（`$P4TMP/probe-b/palette.mjs`）

```js
// One-off (M1/P4 browser checks, group B item 8: plan "控制者的浏览器核对"; spec section 3 item 4): the command palette
// on the profile settings (/settings/profile/general), a page without a workspace in its address. No uncaught error;
// when the palette offers "New work item" there, choosing it opens no work-item modal (the base opened one whose
// submit threw, because the modal read the workspace from the address). Two ways in: loaded directly, and reached
// in-app from a workspace page through the palette's "Go to account settings" (the stores then still hold the
// workspace's projects). The server, the stubs and the per-scenario checks are in ../probe-a/lib.mjs.
// Run from the repository root of the build. usage: node palette.mjs [scenario name filter]
import { SLASH, WS, check, goto, probe, scenario, settle, signedInStubs, visible, workspaceStubs } from "../probe-a/lib.mjs";

const profileStubs = () => ({
  ...signedInStubs(),
  ...workspaceStubs(),
  "GET /api/users/me/workspaces/invitations/": [],
});

const openPalette = async (page) => {
  await page.keyboard.press("ControlOrMeta+k");
  const input = page.locator("[cmdk-input]");
  return visible(input, 5000);
};
const offered = async (page) => (await page.locator("[cmdk-item]").allInnerTexts()).map((t) => t.trim().split("\n")[0]);
const workItemModal = (page) => page.getByRole("dialog").locator('input[name="name"]');

// The palette on the profile settings; `label` says how the page was reached.
async function onProfileSettings(page, label) {
  check(`B8 ${label}: the palette opens`, await openPalette(page));
  const items = await offered(page);
  console.log(`  offered: ${JSON.stringify(items)}`);
  const create = page.locator("[cmdk-item]", { hasText: "New work item" });
  if ((await create.count()) === 0) {
    check(`B8 ${label}: "New work item" is not offered, so no modal can open from the palette`, true);
  } else {
    await create.first().click();
    await page.waitForTimeout(1500);
    check(`B8 ${label}: choosing "New work item" opens no work-item modal`, !(await workItemModal(page).isVisible()), "the modal opened");
  }
  // The same command's key sequence ("n" then "i") with the palette closed.
  await page.keyboard.press("Escape");
  await page.waitForTimeout(300);
  await page.locator("body").click({ position: { x: 5, y: 5 } });
  await page.keyboard.press("n");
  await page.keyboard.press("i");
  await page.waitForTimeout(1500);
  check(`B8 ${label}: the key sequence n i opens no work-item modal`, !(await workItemModal(page).isVisible()), "the modal opened");
}

probe(async () => {
  await scenario("profile settings loaded directly", profileStubs(), async (page) => {
    await goto(page, `/settings/profile/general${SLASH}`);
    check("B8.1 lands on the profile settings", await settle(page, "/settings/profile/general"), page.url());
    await page.waitForTimeout(1000);
    await onProfileSettings(page, "loaded directly");
  });

  await scenario("profile settings reached from the workspace", profileStubs(), async (page) => {
    await goto(page, `/${WS}/projects${SLASH}`);
    await settle(page, `/${WS}/projects`);
    await page.waitForTimeout(1000);
    check("B8.2 the palette opens on the workspace", await openPalette(page));
    await page.locator("[cmdk-input]").fill("account settings");
    await page.locator("[cmdk-item]", { hasText: "Go to account settings" }).first().click();
    check("B8.2 \"Go to account settings\" lands on the profile settings", await settle(page, "/settings/profile/general"), page.url());
    await page.waitForTimeout(1000);
    await onProfileSettings(page, "reached from the workspace");
  });
});
```

#### 3. 运行命令

```bash
cd $P4TMP/probe-app   # 检出到 6f190a1
bash $P4TMP/probe-c/run-c.sh fix   # 全部三组；B 组的部分如下
STRICT=1 node $P4TMP/probe-b/p2probe1.mjs
node $P4TMP/../nerve-p2/probe-2/editor-probe.mjs   # P2 的原文
node $P4TMP/probe-c/p2probe3.mjs
node $P4TMP/probe-c/p3auth.mjs
node $P4TMP/probe-c/p3lists.mjs
SLASH= STRICT=1 node $P4TMP/probe-b/palette.mjs
```

#### 脚本全文（`$P4TMP/probe-c/run-c.sh`）

```bash
#!/bin/bash
# One-off (M1/P4 browser checks, group C: plan "控制者的浏览器核对" items 9–14): on the build in $P4TMP/probe-app (or
# the directory given), run the current-item (9) and address (10–12) probes, then group A and group B item 7 again
# with every app address expected without a trailing slash (13): redirects with SLASH= STRICT=1, P2 probe 1's P4 copy
# with STRICT=1, P2 probe 2 as archived (it expects no slash-ended address), and the P4 copies of P2 probe 3 and the
# P3 probes (adapt-p2p3.py), plus the palette probe (item 8).
# Outputs go to $P4TMP/probe-c/<name>-<tag>.txt. usage: bash run-c.sh <tag> [build dir]
set -u
P=/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad
C=$P/nerve-p4/probe-c
TAG=$1
cd "${2:-$P/nerve-p4/probe-app}"
echo "build at $(git rev-parse --short HEAD)"
run() {
  local name=$1
  shift
  env "$@" > "$C/$name-$TAG.txt" 2>&1
  local code=$?
  local fails
  fails=$(grep -c "^FAIL" "$C/$name-$TAG.txt")
  echo "$name: exit $code, PASS $(grep -c '^PASS' "$C/$name-$TAG.txt"), FAIL $fails, last: $(tail -1 "$C/$name-$TAG.txt")"
  [ "$fails" -gt 0 ] && grep "^FAIL" "$C/$name-$TAG.txt" | head -12 | cut -c1-260
  return 0
}
run current node "$C/current.mjs"
run addresses node "$C/addresses.mjs"
run redirects SLASH= STRICT=1 node "$P/nerve-p4/probe-a/redirects.mjs"
run palette SLASH= STRICT=1 node "$P/nerve-p4/probe-b/palette.mjs"
run p2probe1 STRICT=1 node "$P/nerve-p4/probe-b/p2probe1.mjs"
run p2probe2 node "$P/nerve-p2/probe-2/editor-probe.mjs"
run p2probe3 node "$C/p2probe3.mjs"
run p3auth node "$C/p3auth.mjs"
run p3lists node "$C/p3lists.mjs"
```

#### 4. 输出

P2 探测 1（首页、个人主页、侧边栏）：

```
serving web/apps/web/build/client (SPA fallback: index.html) on port 51410

== home with recent visits
PASS 1.1 home greets the signed-in user
PASS 1.2 recents list the visited work item (PRB-7, its name, its link)
PASS 1.3 recents list the visited project (identifier, name, link to its work items)
PASS 1.4 the recents request asks for all entities (no entity_name)
PASS 1.5 with a joined project and two members the quickstart guide stays hidden
PASS 2.3a home (with data) has no manage-widgets button, quick links or stickies
PASS 6.1 the sidebar shows exactly Home, Your work, Drafts | Projects, Views, Archives, in this order
PASS 6.2a the sidebar offers no customize, pin, unpin or more control
PASS 6.2b hovering a sidebar item shows no control and no item can be dragged
PASS 3.1 the home page and the sidebar have no link to /pages and no "Pages" item
PASS 3.2 the project's sidebar navigation (accordion) is Work items, Cycles, Modules, Views, Intake, with no Pages
PASS 3.3 with the project's navigation open there is still no /pages link and no "Pages" item
PASS 6.3 the sidebar preferences button opens the project navigation dialog
PASS 6.4 choosing Tabbed Navigation saves {navigation_control_preference: TABBED}
PASS 6.5 the dialog closes on Escape
  (landed on /probe-ws/profile/u1/assigned)
PASS 4.1 the sidebar's Your work lands on /probe-ws/profile/u1/assigned
PASS 4.2 one's own profile is labelled Your work in the breadcrumb and Assigned is a tab

== project in tabbed navigation
PASS 3.4 the project's tabs are Work items, Cycles, Modules, Views, Intake, with no Pages
PASS 3.5 the project page has no link to /pages and no "Pages" item

== home, empty
PASS 2.1 with no recent visits the recents empty state shows
PASS 2.2 with no projects the no-projects empty state (quickstart guide) shows, with Create a project
PASS 2.3b home (empty) has no manage-widgets button, quick links or stickies
PASS 2.4 filtering recents by Projects asks for entity_name=project and shows the projects empty state

== profile redirect
  (landed on /probe-ws/profile/u2/assigned?probe=1)
PASS 4.3 /probe-ws/profile/u2?probe=1 lands on /probe-ws/profile/u2/assigned?probe=1
  (landed on /probe-ws/profile/u2/assigned)
PASS 4.4 /probe-ws/profile/u2 lands on /probe-ws/profile/u2/assigned, titled "Profile - Assigned"

== profile card: active member
PASS 5a card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"
PASS 5a neither the not-a-member nor the load-failed message shows, no "undefined"

== profile card: guest
PASS 5b card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"
PASS 5b neither the not-a-member nor the load-failed message shows, no "undefined"

== profile card: absent from the list
PASS 5c card shows "This user is not a member of this workspace."
PASS 5c no name and no "Could not load this workspace's members."; breadcrumb is "Work", no "undefined"

== profile card: inactive
PASS 5d card shows "This user is not a member of this workspace."
PASS 5d no name and no "Could not load this workspace's members."; breadcrumb is "Work", no "undefined"

== profile card: members request fails (500)
PASS 5e card shows "Could not load this workspace's members."
PASS 5e no name and no "This user is not a member of this workspace."; breadcrumb is "Work", no "undefined"

== profile card: members still loading
PASS 5f while the members load (before the delayed reply): a loading skeleton, no name, no not-a-member, no load-failed, no "undefined"
PASS 5g once they arrive the skeleton gives way to the card of Ada Lovelace

== unstubbed requests (answered 404)
home with recent visits:
  GET /api/workspaces/probe-ws/users/notifications/unread/
project in tabbed navigation:
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/projects/p1/issue-labels/ x2
  GET /api/workspaces/probe-ws/projects/p1/members/ x2
  GET /api/workspaces/probe-ws/projects/p1/states/ x2
  GET /api/workspaces/probe-ws/projects/p1/intake-state/ x2
  GET /api/workspaces/probe-ws/projects/p1/cycles/ x2
  GET /api/workspaces/probe-ws/modules/ x2
  GET /api/workspaces/probe-ws/projects/p1/modules/ x2
  GET /api/workspaces/probe-ws/projects/p1/views/ x2
home, empty:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile redirect:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: active member:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: guest:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: absent from the list:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: inactive:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: members request fails (500):
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: members still loading:
  GET /api/workspaces/probe-ws/users/notifications/unread/

== console errors
home with recent visits: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
project in tabbed navigation: 18 from unstubbed requests, 0 from the deliberate 500, 0 other
home, empty: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile redirect: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: active member: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: guest: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: absent from the list: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: inactive: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: members request fails (500): 2 from unstubbed requests, 1 from the deliberate 500, 0 other
profile card: members still loading: 2 from unstubbed requests, 0 from the deliberate 500, 0 other

port 51410 is free; browser connected: false

37 passed, 0 failed
```

P2 探测 2（应用中的编辑器）：

```
  info: /probe-ws/browse/P1-1#b-alpha-2 lands on /probe-ws/browse/P1-1#b-alpha-2
PASS S6 control: without a block id in the URL the block to locate starts below the fold
PASS S6 the image node renders through the asset endpoint
PASS S6 the stored block id is kept in the editor and the #id link scrolls to that block
PASS S1 typing in the description sends it in description_html
  sent description_html: <p class="editor-paragraph-block" data-id="b-alpha-1">Existing alpha text</p><p class="editor-paragraph-block" data-id="0b62b2a8-3528-4e37-8322-5b1b93e7973a">Typed by probe</p><image-component data-id="img-1" src="asset-1" id="img-1" width="120px" height="60px" aspectratio="2" alignment="left" status="uploaded"></image-component>[30 filler paragraphs, ids b-fill-0..29]<p class="editor-paragraph-block" data-id="b-alpha-2">Second block to locate</p>
PASS S1 every block of the sent description carries a unique data-id
PASS S2 Cmd/Ctrl+Z removes the typed text
PASS S2 Shift+Cmd/Ctrl+Z brings it back
  info: saves sent after undo+redo: [{"has_typed_text":false}]; editor shows the typed text: true
PASS S5 the versions dropdown lists the stubbed version
PASS S5 viewing a version renders its content read-only
  clipboard: "## Version heading\n\nVersion **bold** text\n"
PASS S5 copy Markdown puts the version's Markdown on the clipboard
  sent description_html: <h2 class="editor-heading-block" data-id="v-h">Version heading</h2><p class="editor-paragraph-block" data-id="v-p">Version <strong>bold</strong> text</p>
PASS S5 restore puts the version back and sends its description_html
PASS S3 an existing comment renders in a read-only editor
PASS S4 typing @ in the comment box lists the stubbed member
PASS S4 picking the member puts an @mia mention in the comment box
  sent comment_html: <p class="editor-paragraph-block" data-id="49db3413-c8cd-4725-b8d9-0c12d18d40e0">Hello <mention-component id="a71c1c74-6d11-465b-b37f-8c6f73337750" entity_identifier="u2" entity_name="user_mention"></mention-component> </p>
PASS S4 the created comment carries a mention node for that member and shows it
  sent: {"description_html":"<p class=\"editor-paragraph-block\" data-id=\"b039c608-09d7-4f8d-a1b2-165792c89c2c\">Legacy gamma text without an id</p>","skip_activity":"true"}
PASS S3 the parent opened after it (peek) still gets ids: the id-only migration update goes out
  sent description_html: <p class="editor-paragraph-block" data-id="b039c608-09d7-4f8d-a1b2-165792c89c2c">Legacy gamma text without an id</p><p class="editor-paragraph-block" data-id="7b2171c1-fc13-4e84-8163-7fa8c6967064">Typed after read-only</p>
PASS S3 editing that description after the read-only editor keeps a unique data-id on every block

unstubbed requests (404): 8
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/user-favorites/
  GET /api/workspaces/probe-ws/projects/p1/user-properties/
  GET /api/workspaces/probe-ws/projects/p1/intake-state/
  GET /api/workspaces/probe-ws/projects/p1/cycles/
  GET /api/workspaces/probe-ws/modules/
  GET /api/workspaces/probe-ws/projects/p1/modules/
  GET /api/workspaces/probe-ws/projects/p1/views/
console errors on page A0 P1-1: 20 caused by unstubbed requests, 0 others
  [unstubbed] 16x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page A P1-1: 16 caused by unstubbed requests, 0 others
  [unstubbed] 12x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page B P1-2: 15 caused by unstubbed requests, 0 others
  [unstubbed] 11x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
port 51474 is free
17 passed, 0 failed
```

P2 探测 3（动态、通知、工作项列表）：

```
PASS activity: the work item opens from its project path (redirects to /browse/PRB-1/)
PASS activity: 5 property entries (created, state, relation, cycle, module), each with actor and text
PASS activity: state entry reads "otto set the state to In Progress."
PASS activity: relation entry reads "otto marked this work item is blocking work item PRB-2."
PASS activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1
PASS activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1
PASS activity: comment entry shows its author and text
PASS activity: no raw i18n key, undefined, null or NaN in the activity section
PASS notifications: the page lists both stubbed notifications with readable text
PASS notifications: before reading, the inbox shows the unread dot and All shows 2
PASS notifications: opening one sends its read request and All drops to 1
PASS notifications: opening one shows the work item preview (title and description)
PASS notifications: "mark all as read" posts {snoozed:false, archived:false}
PASS notifications: after mark all read, the inbox dot, the All count and the card dots are gone
PASS store: project page lists exactly ["Project item alpha","Cycle item gamma","Module item delta","Assigned item epsilon"]
PASS store: cycle page lists exactly ["Project item alpha","Cycle item gamma"]
PASS store: module page lists exactly ["Project item alpha","Module item delta"]
PASS store: profile page lists exactly ["Assigned item epsilon"]
PASS store: archives page lists exactly ["Archived item zeta"]
PASS layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)
PASS layouts: switching to Board Layout saves layout=kanban and renders it without an error
PASS layouts: switching to Table Layout saves layout=spreadsheet and renders it without an error
PASS layouts: switching to Calendar Layout saves layout=calendar and renders it without an error
PASS layouts: switching to List Layout saves layout=list and renders it without an error

unstubbed requests (0):

browser console errors per scenario:
  activity: 0 (from unstubbed requests: 0, other: 0)
  notifications: 0 (from unstubbed requests: 0, other: 0)
  store-project: 0 (from unstubbed requests: 0, other: 0)
  store-cycle: 0 (from unstubbed requests: 0, other: 0)
  store-module: 0 (from unstubbed requests: 0, other: 0)
  store-profile: 0 (from unstubbed requests: 0, other: 0)
  store-archives: 0 (from unstubbed requests: 0, other: 0)
  layouts: 0 (from unstubbed requests: 0, other: 0)
port 51495 is free
browser processes started by this run: 3; still running: none

all checks passed
```

P3 探测 1（认证与实例）：

```

== sign-in fails
PASS 1.1 / shows the sign-in form: email, password, no confirmation, "Go to workspace"
PASS 1.1b the document title names the sign-in page
PASS 1.2 no sign-in code, forgot-password, third-party or admin entry on the page
PASS 1.3 sign-up is enabled, so the header links to /sign-up
PASS 1.4 the email field is editable and its clear button empties it
  (landed on /?error_code=5065&email=typed%40example.com)
PASS 1.5 the form posts once, as a page navigation, to /auth/sign-in/
PASS 1.6 the post carries the CSRF token, the typed email, the password and next_path
PASS 1.7 the CSRF token was fetched before the post
PASS 1.8 the failure lands on / with "Authentication failed" above the sign-in form, the email kept
PASS 1.9 no request to a deleted sign-in method or step
PASS sign-in fails: no request outside the scenario's list
PASS sign-in fails: no uncaught page error

== sign-in succeeds
  (landed on /probe-ws/projects)
PASS 2.1 after a successful sign-in the next path opens and stays open (no bounce back to /)
PASS 2.2 the post carried the CSRF token and next_path
PASS sign-in succeeds: no request outside the scenario's list
PASS sign-in succeeds: no uncaught page error

== sign-up fails
PASS 3.1 /sign-up shows email, password, confirmation and "Create account"
PASS 3.2 the header links straight to the sign-in page (/), titled "Sign up"
PASS 3.3 a weak password shows the strength banner and posts nothing
  (landed on /?error_code=5035&email=new%40example.com)
PASS 3.4 the sign-up post carries the CSRF token, the typed email and the password
PASS 3.5 the failure lands on / as "Authentication failed" above the sign-in form (accepted change, M2)
PASS sign-up fails: no request outside the scenario's list
PASS sign-up fails: no uncaught page error

== error links between the forms
PASS 4.1 "No account found" links to the sign-up page with the email
  (landed on /sign-up?email=nobody%40example.com)
PASS 4.2 following it opens the sign-up form with the email filled in
PASS 4.3 "already registered" links to the sign-in page with the email
PASS error links between the forms: no request outside the scenario's list
PASS error links between the forms: no uncaught page error

== unknown error code
PASS 5.1 /?error_code=toString shows the sign-in form without an error banner
PASS unknown error code: no request outside the scenario's list
PASS unknown error code: no uncaught page error

== sign-up disabled
PASS 6.1 with enable_signup off the header has no "Sign up" link and no "new to" line
PASS sign-up disabled: no request outside the scenario's list
PASS sign-up disabled: no uncaught page error

== instance request fails
PASS 7.1 a failed instance request shows the maintenance view and no sign-in form
PASS instance request fails: no request outside the scenario's list
PASS instance request fails: no uncaught page error

== password change
PASS 8.1 the profile settings sidebar offers exactly Profile, Security, Preferences, Personal Access Tokens
PASS 8.2 the security page has old, new and confirmation password fields
PASS 8.3 the change posts old and new password with the X-CSRFToken header and reports success
PASS 8.4 no request to a deleted account endpoint (set-password, notification preferences, email change)
PASS password change: no request outside the scenario's list
PASS password change: no uncaught page error

== sign-out
  (landed on /)
PASS 9.1 sign-out posts the CSRF token as a page navigation and lands on the sign-in form
PASS sign-out: no request outside the scenario's list
PASS sign-out: no uncaught page error

46 passed, 0 failed
```

P3 探测 2（动态、通知、工作项列表与详情）：

```
PASS activity: the work item opens from its project path (redirects to /browse/PRB-1/)
PASS activity: 5 property entries (created, state, relation, cycle, module), each with actor and text
PASS activity: state entry reads "otto set the state to In Progress."
PASS activity: relation entry reads "otto marked this work item is blocking work item PRB-2."
PASS activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1
PASS activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1
PASS activity: comment entry shows its author and text
PASS activity: no raw i18n key, undefined, null or NaN in the activity section
PASS activity (P3): the lookup by sequence uses /work-items/PRB-1/, the description history /work-items/i1/description-versions/
PASS mentions (P3): typing @ in the comment editor searches users only and lists them
PASS notifications: the page lists both stubbed notifications with readable text
PASS notifications: before reading, the inbox shows the unread dot and All shows 2
PASS notifications: opening one sends its read request and All drops to 1
PASS notifications: opening one shows the work item preview (title and description)
PASS notifications: "mark all as read" posts {snoozed:false, archived:false}
PASS notifications: after mark all read, the inbox dot, the All count and the card dots are gone
PASS store: project page lists exactly ["Project item alpha","Cycle item gamma","Module item delta","Assigned item epsilon"]
PASS store: cycle page lists exactly ["Project item alpha","Cycle item gamma"]
PASS store: module page lists exactly ["Project item alpha","Module item delta"]
PASS store: profile page lists exactly ["Assigned item epsilon"]
PASS store: archives page lists exactly ["Archived item zeta"]
PASS layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)
PASS layouts: switching to Board Layout saves layout=kanban and renders it without an error
PASS layouts: switching to Table Layout saves layout=spreadsheet and renders it without an error
PASS layouts: switching to Calendar Layout saves layout=calendar and renders it without an error
PASS layouts: switching to List Layout saves layout=list and renders it without an error
PASS cover picker (P3): "Change cover" opens Images and Upload only, and nothing asks /api/unsplash/
PASS favourites (P3): the sidebar lists a project, view, cycle, module and folder favourite, each with an icon

unstubbed requests (0):

browser console errors per scenario:
  activity: 0 (from unstubbed requests: 0, other: 0)
  notifications: 0 (from unstubbed requests: 0, other: 0)
  store-project: 0 (from unstubbed requests: 0, other: 0)
  store-cycle: 0 (from unstubbed requests: 0, other: 0)
  store-module: 0 (from unstubbed requests: 0, other: 0)
  store-profile: 0 (from unstubbed requests: 0, other: 0)
  store-archives: 0 (from unstubbed requests: 0, other: 0)
  layouts: 0 (from unstubbed requests: 0, other: 0)
  cover-picker: 0 (from unstubbed requests: 0, other: 0)
  favourites: 0 (from unstubbed requests: 0, other: 0)
port 51608 is free
browser processes started by this run: 3; still running: none

all checks passed
```

第 8 项（个人设置页的命令面板）：

```

== profile settings loaded directly
  (at /settings/profile/general)
PASS B8.1 lands on the profile settings
PASS B8 loaded directly: the palette opens
  offered: ["New workspace","Workspace invites","Sign out","Toggle app sidebar","Copy current page URL","Focus search input","Change interface theme","Change timezone","Change first day of week","Change interface language","Open keyboard shortcuts","Open Plane documentation","Join our Forum","Report a bug"]
PASS B8 loaded directly: "New work item" is not offered, so no modal can open from the palette
PASS B8 loaded directly: the key sequence n i opens no work-item modal
PASS profile settings loaded directly: no request outside the scenario's list
PASS profile settings loaded directly: no uncaught page error
PASS profile settings loaded directly: no "call navigate() in a React.useEffect()" warning

== profile settings reached from the workspace
  (at /probe-ws/projects)
PASS B8.2 the palette opens on the workspace
  (at /settings/profile/general)
PASS B8.2 "Go to account settings" lands on the profile settings
PASS B8 reached from the workspace: the palette opens
  offered: ["New workspace","Workspace invites","Sign out","Toggle app sidebar","Copy current page URL","Focus search input","Change interface theme","Change timezone","Change first day of week","Change interface language","Open keyboard shortcuts","Open Plane documentation","Join our Forum","Report a bug"]
PASS B8 reached from the workspace: "New work item" is not offered, so no modal can open from the palette
PASS B8 reached from the workspace: the key sequence n i opens no work-item modal
PASS profile settings reached from the workspace: no request outside the scenario's list
PASS profile settings reached from the workspace: no uncaught page error
PASS profile settings reached from the workspace: no "call navigate() in a React.useEffect()" warning

15 passed, 0 failed
```

### 6.4 C 组：当前项、地址、片段、旧地址和自动跳转（plan 第 9–13 项，修复轮的第 15 项）

#### 1. 目的

第 9 项：每个导航组件的当前项，每个地址分别以不带和带结尾 `/` 两种形式打开，结果相同且正确。第 10 项：加载不带 `/` 的地址不多出历史记录；点侧边栏、设置侧边栏、顶部通知、面包屑，一次导航就落在不带 `/` 的地址。第 11 项：评论链接的 `#comment-cm1` 保留并高亮评论；收集箱的查询参数保留；个人主页的重定向保留查询参数。第 12 项：旧地址落到"页面不存在"（单段的落到"Workspace not found"，Task 5 的裁定）；停用迭代时的按钮落到功能设置；命令面板的"账户设置"一次到达。第 13 项：A 组和 B 组第 7 项在严格模式下重跑（6.2、6.3）。第 15 项（修复轮加，整分支评审 M4）：三处页面到达时的自动跳转之后，后退回到之前的页面。

#### 2. 假数据

P3 探测 2 的假数据原样取出（`extract-p3data.py` 生成 `p3data.mjs`：项目 `p1`、迭代 `c1`、模块 `m1`、工作项 `i1`–`i5`、评论 `cm1`、两条通知），`stubs.mjs` 再加一个由 `u1` 创建的工作项和 C 组页面额外请求的 6 个地址。第 11、15 项的收集箱另有工作项 `i9`。

#### 脚本全文（`$P4TMP/probe-c/extract-p3data.py`）

```py
"""One-off (M1/P4 group C): the fake data of P3's lists probe as an importable module, so group C serves the same
project, cycle, module, work items, comments and notifications. Takes the section between the "fake data" and
"harness" banners of nerve-p3/probe-lists/lists-detail-probe.mjs verbatim and exports its names.
usage: python3 extract-p3data.py   (writes probe-c/p3data.mjs)"""
import pathlib
import re

here = pathlib.Path(__file__).parent
src = (here / "../../nerve-p3/probe-lists/lists-detail-probe.mjs").resolve().read_text().split("\n")
start = next(i for i, l in enumerate(src) if l.startswith("// ---") and "fake data" in l)
end = next(i for i, l in enumerate(src) if l.startswith("// ---") and "harness" in l)
body = src[start:end]
names = [m.group(1) for l in body if (m := re.match(r"const (\w+) =", l))]
out = [
    "// Generated by extract-p3data.py from nerve-p3/probe-lists/lists-detail-probe.mjs (its fake data section,",
    "// verbatim). STUBS keys: \"METHOD path\" answers that method, a bare path answers GET; functions get (query, request).",
    *body,
    f"export {{ {', '.join(names)} }};",
    "",
]
(here / "p3data.mjs").write_text("\n".join(out))
print(f"lines {start + 1}-{end} of the P3 probe; exports: {', '.join(names)}")
```

#### 脚本全文（`$P4TMP/probe-c/stubs.mjs`）

```js
// One-off (M1/P4 group C): the stubs of the group C scenarios, in ../probe-a/lib.mjs's form ("METHOD path" keys,
// functions get (request, ctx)). P3's lists probe's stubs (p3data.mjs; bare keys answer GET, functions get (query,
// request)) plus what the pages group C opens ask for beyond them.
import { signedInStubs } from "../probe-a/lib.mjs";
import { STUBS, W, P, ISSUES, emptyPage, issue, paginated, project } from "./p3data.mjs";

const fromP3 = () =>
  Object.fromEntries(
    Object.entries(STUBS).map(([key, value]) => [
      /^[A-Z]+ /.test(key) ? key : `GET ${key}`,
      typeof value === "function" ? (request) => value(new URL(request.url()).searchParams, request) : value,
    ])
  );

// profile "created": one work item the signed-in user created (P3's are all created by u2)
const CREATED = issue("i6", 6, "Created item eta", { created_by: "u1" });
const userIssues = (q) => {
  const list = [...ISSUES, CREATED].filter(
    (i) =>
      (!q.get("assignees") || i.assignee_ids.includes(q.get("assignees"))) &&
      (!q.get("created_by") || i.created_by === q.get("created_by")) &&
      !q.get("subscriber")
  );
  return paginated(list, q);
};

export const stubs = (extra = {}) => ({
  ...signedInStubs(),
  ...fromP3(),
  [`GET ${W}/user-issues/u1/`]: (request) => userIssues(new URL(request.url()).searchParams),
  // the pages group C opens beyond P3's
  [`GET ${W}/recent-visits/`]: [],
  [`GET ${W}/projects/details/`]: [project],
  [`GET ${W}/draft-issues/`]: { ...emptyPage, next_cursor: "30:1:0", prev_cursor: "30:-1:1", count: 0, total_count: 0 },
  [`GET ${W}/invitations/`]: [],
  [`GET ${P}/archived-modules/`]: [],
  "GET /api/timezones/": { timezones: [{ value: "UTC", label: "UTC", gmt_offset: "+00:00" }] },
  ...extra,
});
```

#### 脚本全文（`$P4TMP/probe-c/current.mjs`）

```js
// One-off (M1/P4 browser checks, group C item 9: plan "控制者的浏览器核对"; spec 2.7): which item each navigation
// component shows as current, with every address opened both without and with a trailing slash — the answers must be
// the expected ones and the same for both forms. Components and how they mark the current item (their classes are
// unchanged by P4, only the decision is): the workspace sidebar and a project's navigation items
// ("!bg-layer-transparent-active" on SidebarNavItem), the settings sidebars ("bg-layer-transparent-selected"),
// the app rail's settings and the top bar's inbox ("!text-icon-primary" on the icon), the profile tabs and the
// archive tabs ("border-accent-strong"). The data is P3's lists probe's (p3data.mjs); the server and the per-scenario
// checks are in ../probe-a/lib.mjs.
// Run from the repository root of the build. usage: node current.mjs [scenario name filter]
import { check, goto, probe, scenario, strip } from "../probe-a/lib.mjs";
import { stubs } from "./stubs.mjs";

// label lists of the current items, per component
const current = {
  sidebar: (page) =>
    page.locator("#main-sidebar a[href]").evaluateAll((as) =>
      as.filter((a) => a.querySelector('[class~="!bg-layer-transparent-active"]') || a.matches('[class~="!bg-layer-transparent-active"]'))
        .map((a) => a.innerText.trim().split("\n")[0])),
  settings: (page) =>
    page.locator('[class~="bg-layer-transparent-selected"]').evaluateAll((els) =>
      els.filter((e) => !e.closest("[data-app-rail], .app-rail") && e.innerText.trim()).map((e) => e.innerText.trim().split("\n")[0])),
  rail: (page) =>
    page.locator('a[href]:has([class~="!text-icon-primary"])').evaluateAll((as) => as.map((a) => a.getAttribute("href"))),
  tabs: (page) =>
    page.locator('a[href]:has(span[class~="border-accent-strong"])').evaluateAll((as) => as.map((a) => a.innerText.trim())),
};

// [name, address, component, expected current labels (or hrefs for the rail), extra check]
const CASES = [
  ["sidebar: home", "/probe-ws", "sidebar", ["Home"]],
  ["sidebar: projects", "/probe-ws/projects", "sidebar", ["Projects"]],
  ["sidebar: archives", "/probe-ws/projects/archives", "sidebar", ["Archives"]],
  ["sidebar: your work", "/probe-ws/profile/u1/assigned", "sidebar", ["Your work"]],
  ["sidebar: drafts", "/probe-ws/drafts", "sidebar", ["Drafts"]],
  ["profile tabs: created", "/probe-ws/profile/u1/created", "tabs", ["Created"], async (page) => {
    const text = await page.locator("main, body").first().innerText();
    return { ok: text.includes("Created item eta") && !text.includes("Assigned item epsilon"), detail: "lists the created work item only" };
  }],
  ["workspace settings: general", "/probe-ws/settings", "settings", ["General"]],
  ["workspace settings: members", "/probe-ws/settings/members", "settings", ["Members"]],
  ["project settings: general", "/probe-ws/settings/projects/p1", "settings", ["General"]],
  ["project settings: labels", "/probe-ws/settings/projects/p1/labels", "settings", ["Labels"]],
  // The plan's "app rail: settings is current on /probe-ws/settings/members" cannot be observed: the app rail is never
  // rendered (AppRailVisibilityProvider's isEnabled defaults to false and its only use passes none), so there is no
  // case for it; the dead app rail goes to the M1 closeout. The top bar's inbox uses the same item component.
  ["top bar: inbox", "/probe-ws/notifications", "rail", ["/probe-ws/notifications"]],
  ["project navigation: cycles", "/probe-ws/projects/p1/cycles/c1", "sidebar", ["Cycles"]],
  ["archive tabs: modules", "/probe-ws/projects/p1/archives/modules", "tabs", ["Modules"]],
];

const normalise = (component, list) => (component === "rail" ? list.map(strip) : list);

probe(async () => {
  for (const [name, address, component, expected, extra] of CASES) {
    for (const form of [address, `${address}/`]) {
      await scenario(`${name} (${form})`, stubs(), async (page) => {
        await goto(page, form);
        await page.waitForLoadState("networkidle").catch(() => {});
        await page.waitForTimeout(1500);
        const got = normalise(component, await current[component](page));
        console.log(`  at ${new URL(page.url()).pathname}; current ${component}: ${JSON.stringify(got)}`);
        check(`C9 ${name} (${form}): current ${component} is ${JSON.stringify(expected)}`,
          JSON.stringify(got) === JSON.stringify(expected), JSON.stringify(got));
        if (extra) {
          const { ok, detail } = await extra(page);
          check(`C9 ${name} (${form}): ${detail}`, ok, (await page.locator("body").innerText()).slice(0, 400));
        }
      });
    }
  }
});
```

#### 脚本全文（`$P4TMP/probe-c/addresses.mjs`）

```js
// One-off (M1/P4 browser checks, group C items 10–12: plan "控制者的浏览器核对"; spec 2.6, 2.7):
//   10. loading an address without a trailing slash adds no history entry, and in-app navigation (the sidebar, the
//       settings sidebar, the top bar's inbox, a breadcrumb) lands on an address without one in a single navigation;
//   11. the query and the fragment survive: a comment link (#comment-cm1) keeps its fragment and highlights the
//       comment, with and without a trailing slash; the intake page keeps currentTab/inboxIssueId and opens that
//       item; the profile redirect keeps the query;
//   12. legacy addresses land on "page not found"; the disabled cycles page's button goes to the cycles feature
//       settings; the palette's "Go to account settings" goes straight to /settings/profile/general;
//   15. (fix wave, final review M4) the redirects a page makes on arrival replace its address: going back returns to
//       the page before, not to the address that redirects forward again.
// The data is P3's lists probe's (p3data.mjs); the server and the per-scenario checks are in ../probe-a/lib.mjs.
// Run from the repository root of the build. usage: node addresses.mjs [scenario name filter]
import { Reply, check, goto, probe, scenario, settle, visible } from "../probe-a/lib.mjs";
import { P, W, emptyPage, issue, project } from "./p3data.mjs";
import { stubs } from "./stubs.mjs";

const NOT_FOUND = "Sorry, the page you are looking for cannot be found.";
const WORKSPACE_NOT_FOUND = "Workspace not found";

// Every main-frame address the page goes through while `action` runs (Playwright reports history API navigations
// as frame navigations too).
async function addressesDuring(page, action, settleMs = 2000) {
  const seen = [];
  const listener = (frame) => {
    if (frame === page.mainFrame()) seen.push(new URL(frame.url()));
  };
  page.on("framenavigated", listener);
  await action();
  await page.waitForTimeout(settleMs);
  page.off("framenavigated", listener);
  return seen.map((u) => `${u.pathname}${u.search}${u.hash}`);
}

// A click that must land on `want` in exactly one navigation, without a trailing slash at any point.
async function clickLands(page, label, locator, want) {
  const seen = await addressesDuring(page, () => locator.first().click());
  const path = new URL(page.url()).pathname;
  check(`C10 ${label}: lands on ${want} in one navigation, no trailing slash`,
    path === want && seen.length === 1 && !seen.some((a) => /.\/($|\?|#)/.test(a)), JSON.stringify(seen));
}

probe(async () => {
  // 10.0 loading an address without the slash adds no history entry (the base's slash redirect added one)
  await scenario("C10 page load", stubs(), async (page) => {
    await goto(page, "/probe-ws/projects");
    await settle(page, "/probe-ws/projects");
    const { length, path } = await page.evaluate(() => ({ length: history.length, path: location.pathname }));
    check("C10.0 /probe-ws/projects loads as itself and adds one history entry", path === "/probe-ws/projects" && length === 2,
      JSON.stringify({ length, path }));
  });

  // 10.1–10.4 in-app navigation
  await scenario("C10 in-app navigation", stubs(), async (page) => {
    await goto(page, "/probe-ws/projects");
    await settle(page, "/probe-ws/projects");
    await page.waitForTimeout(1000);
    await clickLands(page, "sidebar Drafts", page.locator("#main-sidebar a", { hasText: "Drafts" }), "/probe-ws/drafts");
    await clickLands(page, "top bar inbox", page.locator('a[href^="/probe-ws/notifications"]'), "/probe-ws/notifications");
    await goto(page, "/probe-ws/settings");
    await settle(page, "/probe-ws/settings");
    await page.waitForTimeout(1000);
    await clickLands(page, "settings sidebar Members", page.locator("a", { hasText: /^Members$/ }), "/probe-ws/settings/members");
    await goto(page, `/probe-ws/projects/p1/cycles/c1`);
    await settle(page, "/probe-ws/projects/p1/cycles/c1");
    await page.waitForTimeout(1500);
    // the breadcrumb's "Cycles" (BreadcrumbLink: an <a> around a rounded-sm div), not the project navigation's items
    // of the same name (sidebar-like <a> elements, in and outside #main-sidebar)
    await clickLands(page, "breadcrumb Cycles", page.locator("a:has(> div.rounded-sm)", { hasText: /^Cycles$/ }),
      "/probe-ws/projects/p1/cycles");
  });

  // 11.1 a comment link keeps its fragment and highlights the comment, with and without the slash
  for (const form of ["/probe-ws/browse/PRB-1#comment-cm1", "/probe-ws/browse/PRB-1/#comment-cm1"]) {
    await scenario(`C11 comment fragment (${form})`, stubs(), async (page) => {
      await goto(page, form);
      const card = page.locator("#comment-cm1");
      await visible(card, 20000);
      await page.waitForTimeout(1500);
      const url = new URL(page.url());
      const highlighted = await page.evaluate(() => {
        const el = document.getElementById("comment-cm1");
        return Boolean(el && (el.matches('[class~="border-accent-strong"]') || el.querySelector('[class~="border-accent-strong"]') || el.closest('[class~="border-accent-strong"]')));
      });
      check(`C11.1 ${form}: the address keeps #comment-cm1`, url.hash === "#comment-cm1", page.url());
      check(`C11.1 ${form}: the comment is highlighted (border-accent-strong)`, highlighted, "not highlighted");
    });
  }

  // 11.2 the intake page keeps its query and opens that item
  const intakeProject = { ...project, inbox_view: true };
  const i9 = issue("i9", 9, "Intake item iota");
  const inboxIssue = { id: "ii9", status: -2, issue: i9, snoozed_till: null, duplicate_to: null, source: "IN_APP", created_by: "u2" };
  const intakeStubs = {
    [`GET ${P}/`]: intakeProject,
    [`GET ${W}/projects/`]: [intakeProject],
    [`GET ${P}/inbox-issues/`]: { next_cursor: "10:1:0", prev_cursor: "10:-1:1", next_page_results: false, prev_page_results: false, total_count: 1, count: 1, total_pages: 1, total_results: 1, results: [inboxIssue] },
    [`GET ${P}/inbox-issues/i9/`]: inboxIssue,
    [`GET ${P}/intake-work-items/i9/description-versions/`]: { next_page_results: false, prev_page_results: false, results: [], total_pages: 0, cursor: "", next_cursor: null, prev_cursor: null, page_count: 0 },
    [`GET ${P}/issues/i9/reactions/`]: [],
    [`GET ${P}/issues/i9/history/`]: [],
    [`GET /api/assets/v2/workspaces/probe-ws/projects/p1/issues/i9/attachments/`]: [],
  };
  await scenario("C11 intake query", stubs(intakeStubs), async (page) => {
    await goto(page, "/probe-ws/projects/p1/intake?currentTab=open&inboxIssueId=i9");
    await page.waitForTimeout(3000);
    const url = new URL(page.url());
    check("C11.2 the intake address keeps currentTab=open&inboxIssueId=i9",
      url.pathname === "/probe-ws/projects/p1/intake" && url.searchParams.get("currentTab") === "open" && url.searchParams.get("inboxIssueId") === "i9", page.url());
    check("C11.2 the intake page opens i9", await visible(page.getByText("Intake item iota"), 10000), (await page.locator("body").innerText()).slice(0, 300));
  });

  // 11.3 the profile redirect keeps the query
  await scenario("C11 profile query", stubs(), async (page) => {
    await goto(page, "/probe-ws/profile/u1?x=1");
    await page.waitForTimeout(2500);
    const url = new URL(page.url());
    check("C11.3 /probe-ws/profile/u1?x=1 lands on /probe-ws/profile/u1/assigned?x=1",
      url.pathname === "/probe-ws/profile/u1/assigned" && url.search === "?x=1", page.url());
  });

  // 12.1 legacy addresses land on "page not found" at their own address. A one-segment one (/sign-in, /login)
  // matches /:workspaceSlug like any unknown one-segment address, so a signed-in visitor gets the workspace
  // wrapper's "Workspace not found" there instead (Task 5 Q2, accepted).
  const legacyCases = [
    ["/sign-in", WORKSPACE_NOT_FOUND],
    ["/login", WORKSPACE_NOT_FOUND],
    ["/probe-ws/settings/account", NOT_FOUND],
    ["/probe-ws/projects/p1/inbox", NOT_FOUND],
  ];
  for (const [legacy, text] of legacyCases) {
    // an unknown one-segment address is a workspace slug: the workspace layout asks for its states and user
    // properties before it shows "Workspace not found"; a real server answers 404 for a workspace that doesn't exist
    const slug = legacy.slice(1);
    const unknownWorkspace = text === WORKSPACE_NOT_FOUND
      ? {
          [`GET /api/workspaces/${slug}/states/`]: new Reply(404, { error: "Workspace not found" }),
          [`GET /api/workspaces/${slug}/user-properties/`]: new Reply(404, { error: "Workspace not found" }),
        }
      : {};
    await scenario(`C12 legacy ${legacy}`, stubs(unknownWorkspace), async (page) => {
      await goto(page, legacy);
      await page.waitForTimeout(2500);
      const path = new URL(page.url()).pathname;
      check(`C12.1 ${legacy} stays and shows "${text}"`,
        path === legacy && (await visible(page.getByText(text), 5000)), `${page.url()}`);
    });
  }

  // 12.2 the disabled cycles page's button goes to the cycles feature settings
  await scenario("C12 disabled cycles", stubs({
    [`GET ${P}/`]: { ...project, cycle_view: false },
    [`GET ${W}/projects/`]: [{ ...project, cycle_view: false }],
  }), async (page) => {
    await goto(page, "/probe-ws/projects/p1/cycles");
    await page.getByRole("button", { name: "Manage features" }).first().click({ timeout: 20000 });
    await page.waitForTimeout(2500);
    const path = new URL(page.url()).pathname;
    check("C12.2 the disabled cycles page's button lands on /probe-ws/settings/projects/p1/features/cycles",
      path === "/probe-ws/settings/projects/p1/features/cycles" && !(await page.getByText(NOT_FOUND).isVisible()), page.url());
  });

  // 15 (final review M4) the automatic redirects replace: going back from where they land returns to the page before,
  // not to the address that redirects forward again
  const backFrom = async (page, label, from, lands) => {
    await goto(page, "/probe-ws/projects");
    await settle(page, "/probe-ws/projects");
    await page.waitForTimeout(1000);
    await goto(page, from);
    const ok = await page.waitForURL((u) => lands(u), { timeout: 20000 }).then(() => true, () => false);
    await page.waitForTimeout(1000);
    check(`C15 ${label}: ${from} redirects`, ok, page.url());
    await page.goBack({ waitUntil: "domcontentloaded" });
    await page.waitForTimeout(2500);
    check(`C15 ${label}: going back returns to /probe-ws/projects`, new URL(page.url()).pathname === "/probe-ws/projects", page.url());
  };
  await scenario("C15 settings/projects redirect", stubs(), async (page) => {
    await backFrom(page, "settings/projects → first project", "/probe-ws/settings/projects", (u) => u.pathname === "/probe-ws/settings/projects/p1");
  });
  await scenario("C15 intake first item", stubs(intakeStubs), async (page) => {
    await backFrom(page, "intake → first item", "/probe-ws/projects/p1/intake?currentTab=open",
      (u) => u.pathname === "/probe-ws/projects/p1/intake" && u.searchParams.get("inboxIssueId") === "i9");
  });
  // the browse page renders the item before its effect sends it to the intake page
  await scenario("C15 browse intake item", stubs({
    ...intakeStubs,
    [`GET ${W}/work-items/PRB-9/`]: { ...i9, is_intake: true },
    [`GET ${P}/issues/i9/sub-issues/`]: { sub_issues: [], state_distribution: {} },
    [`GET ${P}/issues/i9/issue-relation/`]: { blocking: [], blocked_by: [], duplicate: [], relates_to: [] },
    [`GET ${P}/work-items/i9/description-versions/`]: { ...emptyPage, cursor: "", next_cursor: null, prev_cursor: null, page_count: 0 },
  }), async (page) => {
    await backFrom(page, "browse of an intake item → intake", "/probe-ws/browse/PRB-9",
      (u) => u.pathname === "/probe-ws/projects/p1/intake" && u.searchParams.get("inboxIssueId") === "i9");
  });

  // 12.3 the palette's "Go to account settings" goes straight to the profile settings
  await scenario("C12 palette account settings", stubs({ "GET /api/users/me/workspaces/invitations/": [] }), async (page) => {
    await goto(page, "/probe-ws/projects");
    await settle(page, "/probe-ws/projects");
    await page.waitForTimeout(1000);
    await page.keyboard.press("ControlOrMeta+k");
    await page.locator("[cmdk-input]").fill("account settings");
    const seen = await addressesDuring(page, () =>
      page.locator("[cmdk-item]", { hasText: "Go to account settings" }).first().click(), 2500);
    check("C12.3 \"Go to account settings\" goes straight to /settings/profile/general",
      new URL(page.url()).pathname === "/settings/profile/general" && seen.length === 1, JSON.stringify(seen));
  });
});
```

#### 3. 运行命令

```bash
cd $P4TMP/probe-app   # 检出到 6f190a1
node $P4TMP/probe-c/current.mjs
node $P4TMP/probe-c/addresses.mjs
```

#### 4. 输出

第 9 项：

```

== sidebar: home (/probe-ws)
  at /probe-ws; current sidebar: ["Home"]
PASS C9 sidebar: home (/probe-ws): current sidebar is ["Home"]
PASS sidebar: home (/probe-ws): no request outside the scenario's list
PASS sidebar: home (/probe-ws): no uncaught page error
PASS sidebar: home (/probe-ws): no "call navigate() in a React.useEffect()" warning

== sidebar: home (/probe-ws/)
  at /probe-ws/; current sidebar: ["Home"]
PASS C9 sidebar: home (/probe-ws/): current sidebar is ["Home"]
PASS sidebar: home (/probe-ws/): no request outside the scenario's list
PASS sidebar: home (/probe-ws/): no uncaught page error
PASS sidebar: home (/probe-ws/): no "call navigate() in a React.useEffect()" warning

== sidebar: projects (/probe-ws/projects)
  at /probe-ws/projects; current sidebar: ["Projects"]
PASS C9 sidebar: projects (/probe-ws/projects): current sidebar is ["Projects"]
PASS sidebar: projects (/probe-ws/projects): no request outside the scenario's list
PASS sidebar: projects (/probe-ws/projects): no uncaught page error
PASS sidebar: projects (/probe-ws/projects): no "call navigate() in a React.useEffect()" warning

== sidebar: projects (/probe-ws/projects/)
  at /probe-ws/projects/; current sidebar: ["Projects"]
PASS C9 sidebar: projects (/probe-ws/projects/): current sidebar is ["Projects"]
PASS sidebar: projects (/probe-ws/projects/): no request outside the scenario's list
PASS sidebar: projects (/probe-ws/projects/): no uncaught page error
PASS sidebar: projects (/probe-ws/projects/): no "call navigate() in a React.useEffect()" warning

== sidebar: archives (/probe-ws/projects/archives)
  at /probe-ws/projects/archives; current sidebar: ["Archives"]
PASS C9 sidebar: archives (/probe-ws/projects/archives): current sidebar is ["Archives"]
PASS sidebar: archives (/probe-ws/projects/archives): no request outside the scenario's list
PASS sidebar: archives (/probe-ws/projects/archives): no uncaught page error
PASS sidebar: archives (/probe-ws/projects/archives): no "call navigate() in a React.useEffect()" warning

== sidebar: archives (/probe-ws/projects/archives/)
  at /probe-ws/projects/archives/; current sidebar: ["Archives"]
PASS C9 sidebar: archives (/probe-ws/projects/archives/): current sidebar is ["Archives"]
PASS sidebar: archives (/probe-ws/projects/archives/): no request outside the scenario's list
PASS sidebar: archives (/probe-ws/projects/archives/): no uncaught page error
PASS sidebar: archives (/probe-ws/projects/archives/): no "call navigate() in a React.useEffect()" warning

== sidebar: your work (/probe-ws/profile/u1/assigned)
  at /probe-ws/profile/u1/assigned; current sidebar: ["Your work"]
PASS C9 sidebar: your work (/probe-ws/profile/u1/assigned): current sidebar is ["Your work"]
PASS sidebar: your work (/probe-ws/profile/u1/assigned): no request outside the scenario's list
PASS sidebar: your work (/probe-ws/profile/u1/assigned): no uncaught page error
PASS sidebar: your work (/probe-ws/profile/u1/assigned): no "call navigate() in a React.useEffect()" warning

== sidebar: your work (/probe-ws/profile/u1/assigned/)
  at /probe-ws/profile/u1/assigned/; current sidebar: ["Your work"]
PASS C9 sidebar: your work (/probe-ws/profile/u1/assigned/): current sidebar is ["Your work"]
PASS sidebar: your work (/probe-ws/profile/u1/assigned/): no request outside the scenario's list
PASS sidebar: your work (/probe-ws/profile/u1/assigned/): no uncaught page error
PASS sidebar: your work (/probe-ws/profile/u1/assigned/): no "call navigate() in a React.useEffect()" warning

== sidebar: drafts (/probe-ws/drafts)
  at /probe-ws/drafts; current sidebar: ["Drafts"]
PASS C9 sidebar: drafts (/probe-ws/drafts): current sidebar is ["Drafts"]
PASS sidebar: drafts (/probe-ws/drafts): no request outside the scenario's list
PASS sidebar: drafts (/probe-ws/drafts): no uncaught page error
PASS sidebar: drafts (/probe-ws/drafts): no "call navigate() in a React.useEffect()" warning

== sidebar: drafts (/probe-ws/drafts/)
  at /probe-ws/drafts/; current sidebar: ["Drafts"]
PASS C9 sidebar: drafts (/probe-ws/drafts/): current sidebar is ["Drafts"]
PASS sidebar: drafts (/probe-ws/drafts/): no request outside the scenario's list
PASS sidebar: drafts (/probe-ws/drafts/): no uncaught page error
PASS sidebar: drafts (/probe-ws/drafts/): no "call navigate() in a React.useEffect()" warning

== profile tabs: created (/probe-ws/profile/u1/created)
  at /probe-ws/profile/u1/created; current tabs: ["Created"]
PASS C9 profile tabs: created (/probe-ws/profile/u1/created): current tabs is ["Created"]
PASS C9 profile tabs: created (/probe-ws/profile/u1/created): lists the created work item only
PASS profile tabs: created (/probe-ws/profile/u1/created): no request outside the scenario's list
PASS profile tabs: created (/probe-ws/profile/u1/created): no uncaught page error
PASS profile tabs: created (/probe-ws/profile/u1/created): no "call navigate() in a React.useEffect()" warning

== profile tabs: created (/probe-ws/profile/u1/created/)
  at /probe-ws/profile/u1/created/; current tabs: ["Created"]
PASS C9 profile tabs: created (/probe-ws/profile/u1/created/): current tabs is ["Created"]
PASS C9 profile tabs: created (/probe-ws/profile/u1/created/): lists the created work item only
PASS profile tabs: created (/probe-ws/profile/u1/created/): no request outside the scenario's list
PASS profile tabs: created (/probe-ws/profile/u1/created/): no uncaught page error
PASS profile tabs: created (/probe-ws/profile/u1/created/): no "call navigate() in a React.useEffect()" warning

== workspace settings: general (/probe-ws/settings)
  at /probe-ws/settings; current settings: ["General"]
PASS C9 workspace settings: general (/probe-ws/settings): current settings is ["General"]
PASS workspace settings: general (/probe-ws/settings): no request outside the scenario's list
PASS workspace settings: general (/probe-ws/settings): no uncaught page error
PASS workspace settings: general (/probe-ws/settings): no "call navigate() in a React.useEffect()" warning

== workspace settings: general (/probe-ws/settings/)
  at /probe-ws/settings/; current settings: ["General"]
PASS C9 workspace settings: general (/probe-ws/settings/): current settings is ["General"]
PASS workspace settings: general (/probe-ws/settings/): no request outside the scenario's list
PASS workspace settings: general (/probe-ws/settings/): no uncaught page error
PASS workspace settings: general (/probe-ws/settings/): no "call navigate() in a React.useEffect()" warning

== workspace settings: members (/probe-ws/settings/members)
  at /probe-ws/settings/members; current settings: ["Members"]
PASS C9 workspace settings: members (/probe-ws/settings/members): current settings is ["Members"]
PASS workspace settings: members (/probe-ws/settings/members): no request outside the scenario's list
PASS workspace settings: members (/probe-ws/settings/members): no uncaught page error
PASS workspace settings: members (/probe-ws/settings/members): no "call navigate() in a React.useEffect()" warning

== workspace settings: members (/probe-ws/settings/members/)
  at /probe-ws/settings/members/; current settings: ["Members"]
PASS C9 workspace settings: members (/probe-ws/settings/members/): current settings is ["Members"]
PASS workspace settings: members (/probe-ws/settings/members/): no request outside the scenario's list
PASS workspace settings: members (/probe-ws/settings/members/): no uncaught page error
PASS workspace settings: members (/probe-ws/settings/members/): no "call navigate() in a React.useEffect()" warning

== project settings: general (/probe-ws/settings/projects/p1)
  at /probe-ws/settings/projects/p1; current settings: ["General"]
PASS C9 project settings: general (/probe-ws/settings/projects/p1): current settings is ["General"]
PASS project settings: general (/probe-ws/settings/projects/p1): no request outside the scenario's list
PASS project settings: general (/probe-ws/settings/projects/p1): no uncaught page error
PASS project settings: general (/probe-ws/settings/projects/p1): no "call navigate() in a React.useEffect()" warning

== project settings: general (/probe-ws/settings/projects/p1/)
  at /probe-ws/settings/projects/p1/; current settings: ["General"]
PASS C9 project settings: general (/probe-ws/settings/projects/p1/): current settings is ["General"]
PASS project settings: general (/probe-ws/settings/projects/p1/): no request outside the scenario's list
PASS project settings: general (/probe-ws/settings/projects/p1/): no uncaught page error
PASS project settings: general (/probe-ws/settings/projects/p1/): no "call navigate() in a React.useEffect()" warning

== project settings: labels (/probe-ws/settings/projects/p1/labels)
  at /probe-ws/settings/projects/p1/labels; current settings: ["Labels"]
PASS C9 project settings: labels (/probe-ws/settings/projects/p1/labels): current settings is ["Labels"]
PASS project settings: labels (/probe-ws/settings/projects/p1/labels): no request outside the scenario's list
PASS project settings: labels (/probe-ws/settings/projects/p1/labels): no uncaught page error
PASS project settings: labels (/probe-ws/settings/projects/p1/labels): no "call navigate() in a React.useEffect()" warning

== project settings: labels (/probe-ws/settings/projects/p1/labels/)
  at /probe-ws/settings/projects/p1/labels/; current settings: ["Labels"]
PASS C9 project settings: labels (/probe-ws/settings/projects/p1/labels/): current settings is ["Labels"]
PASS project settings: labels (/probe-ws/settings/projects/p1/labels/): no request outside the scenario's list
PASS project settings: labels (/probe-ws/settings/projects/p1/labels/): no uncaught page error
PASS project settings: labels (/probe-ws/settings/projects/p1/labels/): no "call navigate() in a React.useEffect()" warning

== top bar: inbox (/probe-ws/notifications)
  at /probe-ws/notifications; current rail: ["/probe-ws/notifications"]
PASS C9 top bar: inbox (/probe-ws/notifications): current rail is ["/probe-ws/notifications"]
PASS top bar: inbox (/probe-ws/notifications): no request outside the scenario's list
PASS top bar: inbox (/probe-ws/notifications): no uncaught page error
PASS top bar: inbox (/probe-ws/notifications): no "call navigate() in a React.useEffect()" warning

== top bar: inbox (/probe-ws/notifications/)
  at /probe-ws/notifications/; current rail: ["/probe-ws/notifications"]
PASS C9 top bar: inbox (/probe-ws/notifications/): current rail is ["/probe-ws/notifications"]
PASS top bar: inbox (/probe-ws/notifications/): no request outside the scenario's list
PASS top bar: inbox (/probe-ws/notifications/): no uncaught page error
PASS top bar: inbox (/probe-ws/notifications/): no "call navigate() in a React.useEffect()" warning

== project navigation: cycles (/probe-ws/projects/p1/cycles/c1)
  at /probe-ws/projects/p1/cycles/c1; current sidebar: ["Cycles"]
PASS C9 project navigation: cycles (/probe-ws/projects/p1/cycles/c1): current sidebar is ["Cycles"]
PASS project navigation: cycles (/probe-ws/projects/p1/cycles/c1): no request outside the scenario's list
PASS project navigation: cycles (/probe-ws/projects/p1/cycles/c1): no uncaught page error
PASS project navigation: cycles (/probe-ws/projects/p1/cycles/c1): no "call navigate() in a React.useEffect()" warning

== project navigation: cycles (/probe-ws/projects/p1/cycles/c1/)
  at /probe-ws/projects/p1/cycles/c1/; current sidebar: ["Cycles"]
PASS C9 project navigation: cycles (/probe-ws/projects/p1/cycles/c1/): current sidebar is ["Cycles"]
PASS project navigation: cycles (/probe-ws/projects/p1/cycles/c1/): no request outside the scenario's list
PASS project navigation: cycles (/probe-ws/projects/p1/cycles/c1/): no uncaught page error
PASS project navigation: cycles (/probe-ws/projects/p1/cycles/c1/): no "call navigate() in a React.useEffect()" warning

== archive tabs: modules (/probe-ws/projects/p1/archives/modules)
  at /probe-ws/projects/p1/archives/modules; current tabs: ["Modules"]
PASS C9 archive tabs: modules (/probe-ws/projects/p1/archives/modules): current tabs is ["Modules"]
PASS archive tabs: modules (/probe-ws/projects/p1/archives/modules): no request outside the scenario's list
PASS archive tabs: modules (/probe-ws/projects/p1/archives/modules): no uncaught page error
PASS archive tabs: modules (/probe-ws/projects/p1/archives/modules): no "call navigate() in a React.useEffect()" warning

== archive tabs: modules (/probe-ws/projects/p1/archives/modules/)
  at /probe-ws/projects/p1/archives/modules/; current tabs: ["Modules"]
PASS C9 archive tabs: modules (/probe-ws/projects/p1/archives/modules/): current tabs is ["Modules"]
PASS archive tabs: modules (/probe-ws/projects/p1/archives/modules/): no request outside the scenario's list
PASS archive tabs: modules (/probe-ws/projects/p1/archives/modules/): no uncaught page error
PASS archive tabs: modules (/probe-ws/projects/p1/archives/modules/): no "call navigate() in a React.useEffect()" warning

106 passed, 0 failed
```

第 10–12、15 项：

```

== C10 page load
  (at /probe-ws/projects)
PASS C10.0 /probe-ws/projects loads as itself and adds one history entry
PASS C10 page load: no request outside the scenario's list
PASS C10 page load: no uncaught page error
PASS C10 page load: no "call navigate() in a React.useEffect()" warning

== C10 in-app navigation
  (at /probe-ws/projects)
PASS C10 sidebar Drafts: lands on /probe-ws/drafts in one navigation, no trailing slash
PASS C10 top bar inbox: lands on /probe-ws/notifications in one navigation, no trailing slash
  (at /probe-ws/settings)
PASS C10 settings sidebar Members: lands on /probe-ws/settings/members in one navigation, no trailing slash
  (at /probe-ws/projects/p1/cycles/c1)
PASS C10 breadcrumb Cycles: lands on /probe-ws/projects/p1/cycles in one navigation, no trailing slash
PASS C10 in-app navigation: no request outside the scenario's list
PASS C10 in-app navigation: no uncaught page error
PASS C10 in-app navigation: no "call navigate() in a React.useEffect()" warning

== C11 comment fragment (/probe-ws/browse/PRB-1#comment-cm1)
PASS C11.1 /probe-ws/browse/PRB-1#comment-cm1: the address keeps #comment-cm1
PASS C11.1 /probe-ws/browse/PRB-1#comment-cm1: the comment is highlighted (border-accent-strong)
PASS C11 comment fragment (/probe-ws/browse/PRB-1#comment-cm1): no request outside the scenario's list
PASS C11 comment fragment (/probe-ws/browse/PRB-1#comment-cm1): no uncaught page error
PASS C11 comment fragment (/probe-ws/browse/PRB-1#comment-cm1): no "call navigate() in a React.useEffect()" warning

== C11 comment fragment (/probe-ws/browse/PRB-1/#comment-cm1)
PASS C11.1 /probe-ws/browse/PRB-1/#comment-cm1: the address keeps #comment-cm1
PASS C11.1 /probe-ws/browse/PRB-1/#comment-cm1: the comment is highlighted (border-accent-strong)
PASS C11 comment fragment (/probe-ws/browse/PRB-1/#comment-cm1): no request outside the scenario's list
PASS C11 comment fragment (/probe-ws/browse/PRB-1/#comment-cm1): no uncaught page error
PASS C11 comment fragment (/probe-ws/browse/PRB-1/#comment-cm1): no "call navigate() in a React.useEffect()" warning

== C11 intake query
PASS C11.2 the intake address keeps currentTab=open&inboxIssueId=i9
PASS C11.2 the intake page opens i9
PASS C11 intake query: no request outside the scenario's list
PASS C11 intake query: no uncaught page error
PASS C11 intake query: no "call navigate() in a React.useEffect()" warning

== C11 profile query
PASS C11.3 /probe-ws/profile/u1?x=1 lands on /probe-ws/profile/u1/assigned?x=1
PASS C11 profile query: no request outside the scenario's list
PASS C11 profile query: no uncaught page error
PASS C11 profile query: no "call navigate() in a React.useEffect()" warning

== C12 legacy /sign-in
PASS C12.1 /sign-in stays and shows "Workspace not found"
PASS C12 legacy /sign-in: no request outside the scenario's list
PASS C12 legacy /sign-in: no uncaught page error
PASS C12 legacy /sign-in: no "call navigate() in a React.useEffect()" warning

== C12 legacy /login
PASS C12.1 /login stays and shows "Workspace not found"
PASS C12 legacy /login: no request outside the scenario's list
PASS C12 legacy /login: no uncaught page error
PASS C12 legacy /login: no "call navigate() in a React.useEffect()" warning

== C12 legacy /probe-ws/settings/account
PASS C12.1 /probe-ws/settings/account stays and shows "Sorry, the page you are looking for cannot be found."
PASS C12 legacy /probe-ws/settings/account: no request outside the scenario's list
PASS C12 legacy /probe-ws/settings/account: no uncaught page error
PASS C12 legacy /probe-ws/settings/account: no "call navigate() in a React.useEffect()" warning

== C12 legacy /probe-ws/projects/p1/inbox
PASS C12.1 /probe-ws/projects/p1/inbox stays and shows "Sorry, the page you are looking for cannot be found."
PASS C12 legacy /probe-ws/projects/p1/inbox: no request outside the scenario's list
PASS C12 legacy /probe-ws/projects/p1/inbox: no uncaught page error
PASS C12 legacy /probe-ws/projects/p1/inbox: no "call navigate() in a React.useEffect()" warning

== C12 disabled cycles
PASS C12.2 the disabled cycles page's button lands on /probe-ws/settings/projects/p1/features/cycles
PASS C12 disabled cycles: no request outside the scenario's list
PASS C12 disabled cycles: no uncaught page error
PASS C12 disabled cycles: no "call navigate() in a React.useEffect()" warning

== C15 settings/projects redirect
  (at /probe-ws/projects)
PASS C15 settings/projects → first project: /probe-ws/settings/projects redirects
PASS C15 settings/projects → first project: going back returns to /probe-ws/projects
PASS C15 settings/projects redirect: no request outside the scenario's list
PASS C15 settings/projects redirect: no uncaught page error
PASS C15 settings/projects redirect: no "call navigate() in a React.useEffect()" warning

== C15 intake first item
  (at /probe-ws/projects)
PASS C15 intake → first item: /probe-ws/projects/p1/intake?currentTab=open redirects
PASS C15 intake → first item: going back returns to /probe-ws/projects
PASS C15 intake first item: no request outside the scenario's list
PASS C15 intake first item: no uncaught page error
PASS C15 intake first item: no "call navigate() in a React.useEffect()" warning

== C15 browse intake item
  (at /probe-ws/projects)
PASS C15 browse of an intake item → intake: /probe-ws/browse/PRB-9 redirects
PASS C15 browse of an intake item → intake: going back returns to /probe-ws/projects
PASS C15 browse intake item: no request outside the scenario's list
PASS C15 browse intake item: no uncaught page error
PASS C15 browse intake item: no "call navigate() in a React.useEffect()" warning

== C12 palette account settings
  (at /probe-ws/projects)
PASS C12.3 "Go to account settings" goes straight to /settings/profile/general
PASS C12 palette account settings: no request outside the scenario's list
PASS C12 palette account settings: no uncaught page error
PASS C12 palette account settings: no "call navigate() in a React.useEffect()" warning

69 passed, 0 failed
```

#### 5. 没有覆盖的，以及原因

- **应用栏的当前项**：应用栏从不渲染（第 4 节裁定 9）。顶部通知用的是同一个组件，已核对。
- **退出登录**：由 P3 认证脚本第 9 项核对（6.3）。
- **只检查英文**，桌面宽度 1440×900；A5.4 的面包屑返回在 390 像素宽时核对（它只在 640 像素以下显示）。
- P2、P3 脚本原有的"没有覆盖"各项仍然成立。

### 6.5 反向对照

为了证明这些脚本能发现 P4 改变的行为，控制者在基点 `9437a6e` 的构建上跑了 C 组全部（`run-c.sh base`）。第 9 项 106 项全部通过，符合预期：基点的中间件把地址统一成带 `/` 的形式，文本比较按那个形式写；它核对的是 P4 没有改坏当前项。其余失败的正好是 P4 改变的行为：第 10 项（加载时多一条历史记录，四次导航都落在带 `/` 的地址）；第 11 项（不带 `/` 时丢掉片段、评论不高亮，查询参数前被插进 `/`）；第 12 项（`/sign-in`、`/login` 被重定向到 `/create-workspace`，旧的账户设置地址被重定向，停用页的按钮落到 `…/features/`，命令面板经过重定向）；严格模式的 A 组和 P2、P3 副本（地址带 `/`、收件箱链接）。P2 探测 2 和 P3 认证副本在基点也通过：它们不断言依赖结尾 `/` 的地址。

```
current: PASS 106, FAIL 0
addresses: PASS 39, FAIL 15
FAIL C10.0 /probe-ws/projects loads as itself and adds one history entry: {"length":3,"path":"/probe-ws/projects/"}
FAIL C10 sidebar Drafts: lands on /probe-ws/drafts in one navigation, no trailing slash: ["/probe-ws/drafts/"]
FAIL C10 top bar inbox: lands on /probe-ws/notifications in one navigation, no trailing slash: ["/probe-ws/notifications/"]
FAIL C10 settings sidebar Members: lands on /probe-ws/settings/members in one navigation, no trailing slash: ["/probe-ws/settings/members/"]
FAIL C10 breadcrumb Cycles: lands on /probe-ws/projects/p1/cycles in one navigation, no trailing slash: ["/probe-ws/projects/p1/cycles/"]
FAIL C11.1 /probe-ws/browse/PRB-1#comment-cm1: the address keeps #comment-cm1: http://127.0.0.1:55478/probe-ws/browse/PRB-1/
FAIL C11.1 /probe-ws/browse/PRB-1#comment-cm1: the comment is highlighted (border-accent-strong): not highlighted
FAIL C11.2 the intake address keeps currentTab=open&inboxIssueId=i9: http://127.0.0.1:55478/probe-ws/projects/p1/intake/?currentTab=open&inboxIssueId=i9
FAIL C11.3 /probe-ws/profile/u1?x=1 lands on /probe-ws/profile/u1/assigned?x=1: http://127.0.0.1:55478/probe-ws/profile/u1/assigned/?x=1
FAIL C12.1 /sign-in stays and shows "Workspace not found": http://127.0.0.1:55478/create-workspace/
FAIL C12.1 /login stays and shows "Workspace not found": http://127.0.0.1:55478/create-workspace/
FAIL C12.1 /probe-ws/settings/account stays and shows "Sorry, the page you are looking for cannot be found.": http://127.0.0.1:55478/settings/profile/general/
FAIL C12.1 /probe-ws/projects/p1/inbox stays and shows "Sorry, the page you are looking for cannot be found.": http://127.0.0.1:55478/probe-ws/projects/p1/intake/
FAIL C12.2 the disabled cycles page's button lands on /probe-ws/settings/projects/p1/features/cycles: http://127.0.0.1:55478/probe-ws/settings/projects/p1/features/
FAIL C12.3 "Go to account settings" goes straight to /settings/profile/general: ["/settings/profile/general/"]
redirects: PASS 33, FAIL 16
FAIL A1.1 lands on / with next_path=/probe-ws/projects: http://127.0.0.1:55641/?next_path=/probe-ws/projects/
FAIL A1.3 going back returns to /sign-up (the guarded address was replaced): http://127.0.0.1:55641/?next_path=/probe-ws/projects/
FAIL A2.1 a workspace address goes to /onboarding: http://127.0.0.1:55641/onboarding/
FAIL A2.2 /onboarding goes to / with next_path=/onboarding: http://127.0.0.1:55641/?next_path=/onboarding/
FAIL A3.1 / goes to the last workspace: http://127.0.0.1:55641/probe-ws/
FAIL A3.2 a site path in next_path is followed: http://127.0.0.1:55641/probe-ws/projects/
FAIL A3.3 an absolute URL in next_path is ignored: http://127.0.0.1:55641/probe-ws/
FAIL A3.4 a protocol-relative URL in next_path is ignored: http://127.0.0.1:55641/x/
FAIL A3.4 a protocol-relative URL in next_path is ignored: no request outside the scenario's list: ["GET /api/workspaces/x/states/","GET /api/workspaces/x/user-properties/"]
FAIL A3.5 a javascript: URL in next_path is ignored: http://127.0.0.1:55641/javascript:alert(1)/javascript:alert(1)/
FAIL A3.5 a javascript: URL in next_path is ignored: no request outside the scenario's list: ["GET /api/workspaces/javascript:alert(1)/states/","GET /api/workspaces/javascript:alert(1)/user-properties/"]
FAIL A3.6 a site path with a leading space in next_path is followed: http://127.0.0.1:55641/http:/127.0.0.1:55641/probe-ws/projects/http:/127.0.0.1:55641/probe-ws/projects/
FAIL A4.1 / goes to /create-workspace: http://127.0.0.1:55641/create-workspace/
FAIL A5.1 "Go back" returns to the previous page: http://127.0.0.1:55641/create-workspace/
FAIL back and replace: ran to the end: locator.click: Timeout 30000ms exceeded.
FAIL A5.4 the breadcrumb's back returns to the previous page: http://127.0.0.1:55641/probe-ws/projects/p1/cycles/
palette: PASS 14, FAIL 1
FAIL B8.2 "Go to account settings" lands on the profile settings: http://127.0.0.1:56054/settings/profile/general/
p2probe1: PASS 26, FAIL 3
FAIL 1.2 recents list the visited work item (PRB-7, its name, its link): Home
FAIL 6.1 the sidebar shows exactly Home, Your work, Drafts | Projects, Views, Archives, in this order: [{"href":"/probe-ws/","text":"Home"},{"href":"/probe-ws/profile/u1/","text":"Your work"},{"href":"/probe-ws/drafts/","text":"Drafts"},{"href":"/probe-ws/proj
FAIL home with recent visits: ran to the end: locator.hover: Timeout 30000ms exceeded.
p2probe2: PASS 17, FAIL 0
p2probe3: PASS 13, FAIL 11
FAIL activity: the work item opens from its project path (redirects to /browse/PRB-1/): page.waitForURL: Timeout 15000ms exceeded.
FAIL activity: 5 property entries (created, state, relation, cycle, module), each with actor and text: got 0: []
FAIL activity: state entry reads "otto set the state to In Progress.": not among []
FAIL activity: relation entry reads "otto marked this work item is blocking work item PRB-2.": not among []
FAIL activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1: not among []
FAIL activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1: not among []
FAIL activity: comment entry shows its author and text: Cannot read properties of undefined (reading 'locator')
FAIL activity: no raw i18n key, undefined, null or NaN in the activity section: Cannot read properties of undefined (reading 'innerText')
FAIL notifications: before reading, the inbox shows the unread dot and All shows 2: locator.waitFor: Timeout 10000ms exceeded.
FAIL notifications: after mark all read, the inbox dot, the All count and the card dots are gone: All tab reads "All\n1"
FAIL store: profile page lists exactly ["Assigned item epsilon"]: page.waitForURL: Timeout 15000ms exceeded.
p3auth: PASS 46, FAIL 0
p3lists: PASS 16, FAIL 12
FAIL activity: the work item opens from its project path (redirects to /browse/PRB-1/): page.waitForURL: Timeout 15000ms exceeded.
FAIL activity: 5 property entries (created, state, relation, cycle, module), each with actor and text: got 0: []
FAIL activity: state entry reads "otto set the state to In Progress.": not among []
FAIL activity: relation entry reads "otto marked this work item is blocking work item PRB-2.": not among []
FAIL activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1: not among []
FAIL activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1: not among []
FAIL activity: comment entry shows its author and text: Cannot read properties of undefined (reading 'locator')
FAIL activity: no raw i18n key, undefined, null or NaN in the activity section: Cannot read properties of undefined (reading 'innerText')
FAIL mentions (P3): typing @ in the comment editor searches users only and lists them: Cannot read properties of undefined (reading 'locator')
FAIL notifications: before reading, the inbox shows the unread dot and All shows 2: locator.waitFor: Timeout 10000ms exceeded.
FAIL notifications: after mark all read, the inbox dot, the All count and the card dots are gone: All tab reads "All\n1"
FAIL store: profile page lists exactly ["Assigned item epsilon"]: page.waitForURL: Timeout 15000ms exceeded.
```

修复轮加的 A3.7、A3.8 和第 15 项，在修复轮之前的 `23e580f` 上跑（`run-new.sh cur`）：A3.6–A3.8 和第 15 项的三个"后退"都失败，其余通过。A3.7、A3.8 落在 `/projects`：未修剪的值被当作相对路径（第 3 节 Q2）。

```

== A3.1 / goes to the last workspace
  (at /probe-ws)
PASS A3.1 / goes to the last workspace
PASS A3.1 / goes to the last workspace: no request outside the scenario's list
PASS A3.1 / goes to the last workspace: no uncaught page error
PASS A3.1 / goes to the last workspace: no "call navigate() in a React.useEffect()" warning

== A3.2 a site path in next_path is followed
  (at /probe-ws/projects)
PASS A3.2 a site path in next_path is followed
PASS A3.2 a site path in next_path is followed: no request outside the scenario's list
PASS A3.2 a site path in next_path is followed: no uncaught page error
PASS A3.2 a site path in next_path is followed: no "call navigate() in a React.useEffect()" warning

== A3.3 an absolute URL in next_path is ignored
  (at /probe-ws)
PASS A3.3 an absolute URL in next_path is ignored
PASS A3.3 an absolute URL in next_path is ignored: no request outside the scenario's list
PASS A3.3 an absolute URL in next_path is ignored: no uncaught page error
PASS A3.3 an absolute URL in next_path is ignored: no "call navigate() in a React.useEffect()" warning

== A3.4 a protocol-relative URL in next_path is ignored
  (at /probe-ws)
PASS A3.4 a protocol-relative URL in next_path is ignored
PASS A3.4 a protocol-relative URL in next_path is ignored: no request outside the scenario's list
PASS A3.4 a protocol-relative URL in next_path is ignored: no uncaught page error
PASS A3.4 a protocol-relative URL in next_path is ignored: no "call navigate() in a React.useEffect()" warning

== A3.5 a javascript: URL in next_path is ignored
  (at /probe-ws)
PASS A3.5 a javascript: URL in next_path is ignored
PASS A3.5 a javascript: URL in next_path is ignored: no request outside the scenario's list
PASS A3.5 a javascript: URL in next_path is ignored: no uncaught page error
PASS A3.5 a javascript: URL in next_path is ignored: no "call navigate() in a React.useEffect()" warning

== A3.6 a site path with a leading space in next_path is followed
  (at /%20/probe-ws/projects)
FAIL A3.6 a site path with a leading space in next_path is followed: http://127.0.0.1:59800/%20/probe-ws/projects
PASS A3.6 a site path with a leading space in next_path is followed: no request outside the scenario's list
PASS A3.6 a site path with a leading space in next_path is followed: no uncaught page error
PASS A3.6 a site path with a leading space in next_path is followed: no "call navigate() in a React.useEffect()" warning

== A3.7 a site path with a leading newline in next_path is followed
  (at /projects)
FAIL A3.7 a site path with a leading newline in next_path is followed: http://127.0.0.1:59800/projects
FAIL A3.7 a site path with a leading newline in next_path is followed: no request outside the scenario's list: ["GET /api/workspaces/projects/states/","GET /api/workspaces/projects/user-properties/"]
PASS A3.7 a site path with a leading newline in next_path is followed: no uncaught page error
PASS A3.7 a site path with a leading newline in next_path is followed: no "call navigate() in a React.useEffect()" warning

== A3.8 a site path with a leading tab in next_path is followed
  (at /projects)
FAIL A3.8 a site path with a leading tab in next_path is followed: http://127.0.0.1:59800/projects
FAIL A3.8 a site path with a leading tab in next_path is followed: no request outside the scenario's list: ["GET /api/workspaces/projects/states/","GET /api/workspaces/projects/user-properties/"]
PASS A3.8 a site path with a leading tab in next_path is followed: no uncaught page error
PASS A3.8 a site path with a leading tab in next_path is followed: no "call navigate() in a React.useEffect()" warning

27 passed, 5 failed
```

```

== C15 settings/projects redirect
  (at /probe-ws/projects)
PASS C15 settings/projects → first project: /probe-ws/settings/projects redirects
FAIL C15 settings/projects → first project: going back returns to /probe-ws/projects: http://127.0.0.1:60006/probe-ws/settings/projects/p1
PASS C15 settings/projects redirect: no request outside the scenario's list
PASS C15 settings/projects redirect: no uncaught page error
PASS C15 settings/projects redirect: no "call navigate() in a React.useEffect()" warning

== C15 intake first item
  (at /probe-ws/projects)
PASS C15 intake → first item: /probe-ws/projects/p1/intake?currentTab=open redirects
FAIL C15 intake → first item: going back returns to /probe-ws/projects: http://127.0.0.1:60006/probe-ws/projects/p1/intake?currentTab=open&inboxIssueId=i9
PASS C15 intake first item: no request outside the scenario's list
PASS C15 intake first item: no uncaught page error
PASS C15 intake first item: no "call navigate() in a React.useEffect()" warning

== C15 browse intake item
  (at /probe-ws/projects)
PASS C15 browse of an intake item → intake: /probe-ws/browse/PRB-9 redirects
FAIL C15 browse of an intake item → intake: going back returns to /probe-ws/projects: http://127.0.0.1:60006/probe-ws/projects/p1/intake?currentTab=open&inboxIssueId=i9
PASS C15 browse intake item: no request outside the scenario's list
PASS C15 browse intake item: no uncaught page error
PASS C15 browse intake item: no "call navigate() in a React.useEffect()" warning

12 passed, 3 failed
```

## 7. 交接与延后项

交给后续 M 的事项写进对应 M 的 `handoffs/M1-P4-router-native.md`：

| M | 事项 |
|---|---|
| M2 | - **服务端必须校验 `next_path`**：登录、注册表单把地址里的原值提交给 `/auth/sign-in/`、`/auth/sign-up/`，服务端用与 `isValidNextPath` 相同的规则校验（修剪后以单个 `/` 开头，没有 `//`、`\`、控制字符）；<br>- `AuthenticationWrapper` 只做了最小改动，令牌管理器重写时一并重做；<br>- 401 处理里恒为真的判断；<br>- `next_path` 只带 `pathname`；<br>- 修改密码页把数字错误码断言为字符串 |
| M3 | - `RESTRICTED_URLS` 应正好是应用的顶层路由段（`login`、`register` 不在里面，有些列出的词已不是路由）；<br>- 离开项目的弹窗先跳转、后调用接口 |
| M4 | - 工作项弹窗挂在个人设置页上但没有入口，是否去掉由 M4 定；<br>- 表格"子工作项"列的 `#sub-issues` 经加载器重定向丢失，没有元素读取它 |

交给 M1 后续 Phase 和收尾：

| 去向 | 事项 |
|---|---|
| P5 | 无新增事项 |
| M1 收尾 | - **从不渲染的应用栏**（第 4 节裁定 9）：整条删除，范围在收尾时量清，包括 `use-workspace-paths.ts`；<br>- **`={"…"}` 的写法**：Task 4 评审时数过，二十多个文件里有，都是基点的；P2、P3 的规则只收新增行里的，收尾统一收掉；<br>- **开发服务器的 66 条外部化警告**（`Module "path"/"fs"/"url"/"source-map-js" has been externalized for browser compatibility`），基点、原型、本分支相同，归到依赖复核；<br>- **编辑器 callout 图标属性的类型**：`data-emoji-unicode` 声明为 `string`，运行时是数字（`logo-selector.tsx:34` 因此保留 `.toString()`）；<br>- **没有调用方的代码**：`useProjectIssueProperties` 返回的 5 个 fetcher 只有 `fetchCycles` 有调用方；`@plane/utils` 的 `convertHexEmojiToDecimal`、`emojiCodeToUnicode` 没有调用方（对象成员和包导出，knip 看不见）；<br>- **修复轮留下的基点写法**：`use-issues-actions.tsx` 的 `viewId as TProfileViews`（共享的 `IssueActions` 接口把 `viewId` 定为 `string`；根治是在路由处把个人主页的视图段按 profile views 收窄一次，再把 `TProfileViews` 往下传，这要改各种 store 共用的 hook 接口）；三个只给导入常量起别名的 `const`（`lite-text/toolbar.tsx:32`、`issue-layouts/utils.tsx:243`、`:258`）；`no-projects.tsx:115` 没有插值的模板字面量；<br>- **路由测试的宽松之处**（spec 第 3 节第 8 条）：要收紧就得给跳转目标的表达式加取值推断；<br>- **构建体积**：写进收尾评审，对比数据见第 1 节 |

## 8. spec 第 3 节的裁定

第 3 节的 14 项在执行前全部采纳（plan 的"控制者评审补充"），执行中的落实情况：

1. **第 3、4 步各拆成两个 Task**：照此执行，另加 Task 7。评审包按"机械 / 逐处判断"拆开，两类问题分别看得清。
2. **守卫跳转一律 `replace`**：照此执行，A1.3 断言后退不回到被挡住的地址。修复轮把同样的原则推广到三处自动跳转（第 4 节裁定 6）。
3. **路由自己的页头从布局取参数**：照此执行，12 对布局和页头；跟进提交删掉页头里恒为真的判断。
4. **工作项弹窗在没有工作区的页面不渲染**：守卫照此执行，理由更正（第 5 节），交 M4。
5. **停用功能页的按钮改到各自的功能设置页**：照此执行，C12.2 断言。
6. **侧边栏用 `useMatch`，不用 `NavLink` 的 `end`**：照此执行，C 组第 9 项的 13 个场景、两种形式共 106 项全部通过。
7. **`settings/helper.ts` 保留**：照此执行，Task 6 评审确认它按段切分，结尾 `/` 不影响结果。
8. **路由匹配测试的口径**：照此执行，两条反向断言和"改坏再恢复"在 Task 5、6 都做过；是否收紧交收尾。
9. **`frontend-env` 按文件范围排除 Node 端代码**：照此执行，没有登记新的例外。
10. **`globalEnv` 留 `NODE_ENV`**：照此执行。
11. **`getFileURL` 保留为 `path || undefined`**：照此执行。
12. **`normalizeAPIRequestURL` 的签名变了**：照此执行，单元测试 5 → 3 个（services 13 → 11）。
13. **收掉的死代码**：照此执行；修复轮又删了 router store 的 4 个 getter。
14. **M0 交接的收尾写在 Task 6 的文档里**：照此执行，`M0-P5-frontend-trim-notes` 为 `closed`，`M0-P6-knip-notes` 保持 `open`。
