# M1/P3 账户、平台与死代码：评审记录

| 项 | 内容 |
|---|---|
| Phase | M1/P3 `trim-platform` |
| 日期 | 2026-09-24 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P3-trim-platform.md) / [plan](../plans/P3-trim-platform.md)（控制者评审补充在 plan 的 Tasks 之前，`c333971`） |
| 分支 | `worktree-m1-p3-trim-platform`，从 `main` 的 `6d9692b` 分出，提交从 `b85b378` 到本评审记录所在的提交 |
| 评审 | 各 Task 评审（sonnet，第 2 节）；整分支评审（opus，`6d9692b..93635a2`）：Ready to merge with fixes；修复轮后的范围复核（sonnet）：Approved（Minor 一条，已由 `e405bd2` 处理） |

## 1. 范围与结果

P3 删掉 M1 设计 2.2 列出的账户与平台功能、企业版残留和 Plane 自身的死代码，把 knip 改为门禁。保留功能的唯一入口逐条保住（spec 2.13）：

| Task | 删掉的功能 | 保住或改写的保留功能 |
|---|---|---|
| 1 | 第三方登录、验证码登录、找回 / 重置 / 设置密码、"检查邮箱"步骤、管理后台和"实例未完成设置"页、所有邮件发送（通知偏好、修改登录邮箱、营销邮件同意） | 登录页、注册页各剩一个"邮箱 + 密码"表单，模式由路由决定，邮箱可以编辑；CSRF 和会话不动（交 M2）；实例配置剩 4 个字段，请求失败显示维护页；安全页只剩"输入旧密码修改密码"；个人设置剩 4 个标签页（仓库内测试） |
| 2 | 项目和视图的公开发布、space 应用的地址、公开页的类型、评论的"内部 / 外部"可见范围 | 草稿的"发布为工作项"不受影响 |
| 3 | Epic、团队、工作项类型、模板、批量操作与多选、工时、重复工作项、工作流和项目更新的空壳 | 工作项的 service 类型整个删除，描述历史和按编号查找的 `/work-items/` 地址写成字面量；共享的 `ProjectIssues` 保留；`StateOption` 搬到状态下拉框；表单的多选下拉框保留 |
| 4 | AI 助手、Plane AI 的入口、Unsplash、遥测的实例字段 | 图片选择器只剩"图片""上传"，打开时不再发请求 |
| 5 | 计费设置页、升级弹窗和徽标、套餐常量、"联系销售"、付费套餐的保留地址 | 版本号仍在帮助菜单；注销账户的说明不再提计费；工作区设置剩 3 个标签页（仓库内测试） |
| 6 | 企业版扩展点：`extended` 路由表、编辑器的 `flaggedExtensions` / `extendedEditorProps`、"additional"一类的 hook 和映射、`Base*` 类的别名、富文本筛选里空的扩展一半 | 原来的作用直接写在用到的地方；斜杠菜单的顺序不变（新测试）；@提及只搜用户；收藏的实体类型改为五种的联合类型；项目"邀请"改为"添加成员" |
| 7 | knip 报告的 64 个未使用文件和全部未使用的导出、类型；P2 评审转来的清单；Plane 不路由的 8 个 service 方法；集成和导入的文案与图片 | knip 改为门禁（先生成路由类型、配置提示也算错误）；链接扩展不再注册多余的 linkify 协议，去掉了一处模块级的共享状态；`@plane/types` 改用 React 库的 tsconfig |
| 修复轮 | 编辑器里 CE 中为空的 `ADDITIONAL_*` 扩展点；批量归档和打不开的批量删除；本 Phase 造成的 4 个成员孤儿；以企业版功能命名的文案、store 状态和插图（第 3 节） | 错误提示里的链接带上填写的邮箱（F1）；恢复 P2 误删的 `cycles_description`，项目功能说明改为字面量引用 |

数字（P2 结束 `6d9692b` → P3 结束 `e405bd2`；构建体积和探测在 `8eb07ac` 上测得，之后只改了守卫的一个样本）：

| 项 | P2 结束 | P3 结束 |
|---|---|---|
| 关键词守卫 | 24 条规则、14 条例外（11 条 `M1/P3`、1 条 `M6`、2 条 `M9`） | **42 条规则、4 条例外**（`M3`、`M6`、2 条 `M9`），顶层 `phase` 为 `M1/P3` |
| `make lint-web` / `make test-web` / `make build-web` | 52 / 15 / 11 | 52 / 15 / 11 |
| `pnpm exec turbo run check:types` | 23 | 23 |
| 仓库内测试 | constants 5 个、editor 14 个 | constants **11** 个、editor **16** 个 |
| lint 上限合计 | 809 | **711** |
| — web / editor / utils | 658 / 67 / 27 | 566 / 65 / 25 |
| — constants / types | 2 / 1 | 1 / 0 |
| — ui / propel / hooks / i18n | 28 / 21 / 4 / 1 | 不变 |
| `make knip` | 只出报告：未使用文件 85、导出 108、类型 61、枚举成员 4 | **门禁，零** |
| 每种语言的文案键 | 2179 | 1617 |
| 无引用的键（严格方法） | 969 | 498 |
| 图片（无引用 / 全部） | 158 / 286 | 136 / 251 |
| web 的 service 调用 / Plane 不路由的 | 262 / 51（多数是多态地址造成的误报） | 230 / 4（都只差结尾斜杠，分别交 M2、M3、M6） |
| 构建体积 js | 439 个 / 7,319,739 字节 | 422 个 / 6,881,280 字节 |
| 构建体积 css / fonts / other | 3 / 305,244；25 / 3,755,608；122 / 8,043,211 | 3 / 296,988；25 / 3,755,608；112 / 7,910,515 |
| 最大 chunk / 语言 chunk | `use-editor-flagging` 1,381,850 / 34 | `use-parse-editor-content` 1,379,329 / 34 |

- **改动规模**：`6d9692b..e405bd2`（不含本评审记录所在的提交）共 **782 个文件改动，+7909/−27993 行，299 个文件被删除**（`git diff --shortstat`）。不算文档，是 778 个文件、+2532/−27982 行。增加的行主要是 spec、plan、守卫规则的样本和新测试。
- **依赖**：没有增删依赖，锁文件没有变化（spec 第 3 节第 14 条）。整个 Phase knip 都没有报告未使用的依赖。
- **P3 内部发现并修复的两处回归**：
  - Task 1 的错误链接（本 Phase 引入）。错误提示里"去注册""去登录"两个链接还按被删的"检查邮箱"步骤写：一个指向永远是登录页的 `/`，一个经过会丢掉查询参数的旧地址重定向。认证探测发现，Task 1 修复轮改正；链接不带邮箱的问题（F1）由修复轮 `a5403bd` 解决。两位评审者都没有发现它（第 4 节裁定 11）。
  - P2 误删的 `cycles_description`（P2 引入）。项目功能列表用后缀拼出说明的键，P2 的键核对看不见它。整分支评审发现，修复轮 `eb4bdff` 恢复并改为字面量引用（第 4 节裁定 10）。
- **保留行为的核对**（M1 设计 7.5、spec 2.13）：两处进仓库的测试（附录 6.1）和两段临时核对脚本（附录 6.2、6.3，共 74 项检查），在 `8eb07ac` 的构建上全部通过。两段脚本在基点 `6d9692b` 的构建上做了反向对照（附录 6.4）：认证脚本有 12 项失败，探测 2 有 1 项失败，失败的正好是 P3 改变的行为。
- **文档**：`docs/v0/frontend-changes.md` 第二节的 10 行在 Task 7 改为"已完成 | M1/P3"；M1 设计 12 节 P3 一行在本评审提交中更新。

### spec 第 4 节验收标准

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | 7 个 Task 各一个提交，`check:types` 23 | **满足**。7 个 Task 提交（`60875cb`、`9315086`、`cc7fbe3`、`ea3574f`、`bc4b57c`、`5d27988`、`93635a2`），另有修复轮的 8 个提交和守卫样本的 1 个提交（`e405bd2`），每个提交结束时 23 个任务通过 |
| 2 | `make lint-web`：42 条规则、4 条例外，52 个任务，警告数等于上限；`alts.mjs` 没有输出 | **满足**。上限合计 **711**，低于 spec 预计的 713（第 4 节裁定 3）；`alts.mjs M1/P3` 输出 0 |
| 3 | `make test-web` 15（constants 11、editor 16），`make build-web` 11 | **满足**。编辑器测试的输出只剩 `prosemirror-codemark` 的 5 行 sourcemap 警告，记录在案（第 4 节裁定 6） |
| 4 | `make knip` 以门禁形式通过 | **满足**。先运行 `react-router typegen`，不带 `--no-exit-code`，带 `--treat-config-hints-as-errors`；`knip.jsonc` 没有 web 的 `ignoreUnresolved`；持续集成的 `web` 任务以"Unused code"一步执行它；反向对照（故意留一个未使用的导出）退出码为 2 |
| 5 | 守卫：本 Phase 的 18 条规则没有未登记的命中；4 条例外精确，`until` 为 M3、M6 或 M9 | **满足**。M9 的两条（`tlds.ts` 的 `analytics`、`wiki`）写明了顶级域名列表必须完整 |
| 6 | 中英文键集合相同；`keyref.mjs orphaned 6d9692b` 列出的键都已删除；以被删功能命名的键没有剩下 | **修复轮之后满足**。整分支评审发现 34 个 `group_syncing` 键、3 个 SSO 身份键和 `common.publish` 仍在，修复轮一并删除了它们和其他以企业版功能命名的键（第 4 节裁定 4） |
| 7 | `keyref orphaned` 0；`assets-orphaned` 没有输出；`symref orphaned` 只列 `TIssueSearchResponse`；`headers.sh` 没有输出；没有 `${"` | **满足**，在 `8eb07ac` 上对 `6d9692b` 重新核对（`e405bd2` 只改了 `tools/keywords.json` 的一个样本） |
| 8 | `endpoints.py` 只剩 4 处 | **满足**。230 个调用、4 处不路由，都只差结尾斜杠（第 7 节） |
| 9 | M1 设计 7.5 中 P3 的几行核对过 | **满足**，见附录 6 |
| 10 | S1–S4 通过；持续集成 `server`、`web`、`e2e` 通过 | **满足**。分支推送后，持续集成 run 35906156066 在 `8eb07ac` 上通过（合并后在 `main` 上再跑一次）：`server` 42 秒，`web` 129 秒（Lint 一步 72 秒，Unit tests 13 秒，Unused code 5 秒），`e2e` 104 秒（S1–S4 所在的 E2E 一步 15 秒） |
| 11 | `frontend-changes.md` 第二节同步；M1 设计 12 节更新 | **满足** |

## 2. 各 Task 的评审

- 实现者和修复轮都用 Opus 5.5，各 Task 的评审者用 sonnet，整分支评审用 opus。
- 原型（`$P3TMP/proto`，spec 2.2）只作为对照答案，不整体搬运（控制者评审补充第 1 条）：每个 Task 逐步执行，每一处与原型的差异都在报告里说明。原型的错误和遗漏就是这样发现的（第 5 节）。
- 孤儿检查以前一个 Task 的真实提交为基点。从 Task 2 起加上 `infile-orphans.mjs`（同文件内的使用者），从 Task 4 起加上两条插值残渣的 grep（第 4 节裁定 2）。
- 每份报告回答四个风险点：常量化简保留了哪一支；有没有引入共享状态（P2 的 `UniqueID` 教训）；删掉的扩展点在社区版中确实为空；路由模块的约定导出没有动。

| Task | 内容 | 提交 | 核实方式 |
|---|---|---|---|
| 1 | 认证、实例配置、邮件 | `60875cb` | 实现者 DONE_WITH_CONCERNS（106 个文件）。**裁定**：<br>- 原型用 `errorCodeMessages[errorCode]` 直接查地址里的错误码，`?error_code=toString` 会查到 `Object.prototype` 的成员，让登录页崩溃；实现者先判断是否为枚举成员，保持基点行为，采纳（原型错）；<br>- 以被删页面命名的 `instance-not-ready.webp`、`instance-setup-done.webp` 随之删除。<br>评审 Approved，一条 Minor（`isButtonDisabled` 写成 `cond ? false : true`）。<br>**控制者的探测发现了评审漏掉的回归**：<br>- 错误提示里的两个链接还按被删的"检查邮箱"步骤写。"没有账户，去注册"指向 `/?email=`，现在那里永远是登录页，用户原地打转；"已注册，去登录"指向 `/sign-in?email=`，旧地址的重定向会丢掉查询参数；<br>- 页头的两个 `pageTitle` 写反了（Plane 自己的问题，登录页标题是"Sign up"）。<br>修复轮（amend）：两个链接指向 `/sign-up?email=`、`/?email=`；注册页头的"登录"指向 `/`，不经过留给 P4 的 `/sign-in` 重定向；标题复原；`isButtonDisabled` 取反，去掉了基点的一条 `no-unneeded-ternary` 警告，web 上限 630 → 629。<br>重跑探测仍有 3 项失败：链接不带邮箱，因为 `AuthRoot` 调 `authErrorHandler` 时没有传地址里的 `email`，这个参数从来收不到值。Task 2 已经在这个提交上开始，所以不再 amend，记为 **F1**，放进修复轮（第 3 节） |
| 2 | 公开发布、评论的可见范围 | `9315086` | 实现者 DONE（49 个文件）。**差异**：`symref.mjs orphaned` 看不见同一文件内的使用者，`issue_comment.ts` 里只被 `TIssuePublicComment` 使用的 7 个类型是本 Task 的孤儿，原型漏删了。<br>**裁定**：<br>- 采纳，并把实现者写的 `infile-orphans.mjs` 加进之后每个 Task 的孤儿检查。控制者对 Task 1、2 跑了一遍，92 行全部是 `defined now: 0`。收尾的包导出基线因此从 355 变为 350；<br>- 收集箱详情对 `FORMS` 来源显示的"Intake Form user"保留，它对应后端的一个来源取值，交 M4。<br>评审 Approved，一条 Minor：工具栏外层的 `div` 只包一个 `div`，但两者样式不同（滚动容器和工具栏行），不是退化结构，不改 |
| 3 | Epic、团队、工作项类型、批量操作与多选、模板、工作流 | `cc7fbe3` | 实现者 DONE_WITH_CONCERNS（266 个文件）。与原型的 7 处差异全部采纳，其中 3 处是原型的错误，逐步执行才发现：<br>- 表格表头列的判断写错了优先级，除一列外的所有列标题都会隐藏；<br>- 原型的 `bulk.py` 把 `:bg-accent-primary/10` 粘到 14 个文件的类名后面，破坏了 `rounded-none`、`hover:bg-layer-1`、`p-2`、`after:absolute` 等类；<br>- `displayProperties?.key &&` 改变了列表块的行为（基点的 `issue_type` 恒为真）。<br>控制者在 HEAD 上 grep 被粘连的类名和 `selected-issue-row`，都没有。以工作流命名的 6 个键随之删除（裁定 4）。<br>差异有 860 KB，按数据层、界面层拆成两个评审包，两位评审者都 Approved：<br>- 数据层评审者复现了 `endpoints.py` 的 243 / 13（issue.service 的 4 处不路由的调用在本 Task 之前就是这样）。Minor：`EFileAssetType` 的 `INITIATIVE_DESCRIPTION`、`PROJECT_DESCRIPTION` 在基点就已无引用，也不在本 Task 的词汇内。**裁定**：接受，它们和团队的资源类型在同一个枚举片段里、同为企业版类型，挪到 Task 7 只会多留两行死代码；<br>- 界面层评审者抽查了 30 多处常量化简，都保留了假的那一支。Minor 1：化简留下了插值外壳（没有插值的模板字符串、`{"…"}` 属性、`` `${t(…)}` ``），`${"` 的 grep 看不到它们。**裁定**：本 Task 的修复轮收掉 10 处；从 Task 4 起孤儿检查加两条 grep（裁定 2）。<br>控制者在 `9183e15` 的构建上原样重跑 P2 的探测 3：24/24 |
| 4 | AI 助手、Plane AI、Unsplash、遥测 | `ea3574f` | 实现者 DONE_WITH_CONCERNS（37 个文件）。差异：<br>- 一处悬空的类型断言注释；<br>- `EditorRefApi.setEditorValueAtCursorPosition` 和 `insert-content-at-cursor-position.ts`：唯一的调用方是被删的 AI 处理器，`symref` 看不见对象方法。<br>原型两处都漏了。<br>**裁定**：<br>- 工作项表单描述编辑器下方的 AI 行（`p-3`）在社区版里只可能是空的，删除，记为可见变化（第 4 节裁定 8）；<br>- `posthog`、`intercom` 的命中样本保留：M1 设计 7.4 点名了这两个词，规则的 `why` 写明样本是 Plane 的 SDK 接线。<br>评审 Approved。封面帮助函数里的两处退化（`if (http) return "uploaded_asset"; return "uploaded_asset";` 和一个不可达的 `return imageUrl;`）在本 Task 编辑的函数里，由修复轮收掉，任何输入的结果都不变 |
| 5 | 计费和升级提示 | `bc4b57c` | 实现者 DONE（60 个文件）。计费文案删完之后，21 条例外随之过期删除，顶层 `phase` 改为 `M1/P3`，守卫变为 39 条规则、3 条例外。<br>**裁定**：<br>- `features-list.tsx` 只剩一个子元素的 fragment 收掉（本 Task 改动的 JSX）；<br>- 侧边栏底部那一栏只放着版本徽标，随徽标删除，记为可见变化，版本号仍在帮助菜单里；<br>- 改动文件里基点就已死的代码（从不被读的 `isEnabled: true`、不可达的 `featureItem?.href` 分支、注释掉的 `renderChildren`、空的 FEATURES 设置分组）交给 Task 7。<br>评审 Approved，任何级别都没有发现 |
| 6 | 企业版扩展点；项目"邀请"改为添加成员 | `5d27988` | 实现者 DONE（122 个文件）。控制者评审补充第 5 条要求证明每个被删的扩展点在社区版中确实为空，实现者的回答是：<br>- 一张 29 行的表，每行给出 `bc4b57c` 上的文件和行号，说明它为空、恒定还是只做转发；<br>- 在基点和现在，按四组 `disabledExtensions` 打印斜杠菜单、核心扩展列表和富文本附加项，结果相同（只差图片项上两个从不被读的字段）。<br>**裁定**：<br>- 收藏的文件夹保留 logo：基点经由 additional hook 传入 `entity_data.logo_props`，原型的写法会丢掉它；<br>- Task 7 接手的基点死代码：`BlockMenu` / `EditorBubbleMenu` 多余的 prop、`drop.ts` 的空 `else if`、通知兜底里恒真的判断、按 `string` 取键的 `FAVORITE_ITEM_LINKS`；<br>- `useFiltersOperatorConfigs`（恒定值，属于否定运算符的显示层）交 M4。<br>评审 Approved。评审者在 `bc4b57c` 上复核了 12 个以上的扩展点，确认斜杠菜单的顺序不变，运行了新测试（2 个）和编辑器包的全部测试（16 个）。控制者探测 2：28/28 |
| 7 | Plane 自身的死代码、P2 评审的清单、knip 门禁 | `93635a2` | 实现者 DONE_WITH_CONCERNS（242 个文件）。knip 报告的 64 个未使用文件，以及未使用的导出和类型，经过 4 轮全部删完。`make knip` 改为门禁，反向对照时（故意留一个未使用的导出）退出码为 2。<br>P2 评审的清单：<br>- 链接扩展不再注册 linkify 的自定义协议。"already initialized"的输出消失，同时去掉了一处模块级的共享状态：原来每次创建编辑器都 `registerCustomProtocol`，任何一个编辑器销毁都会 `reset()` 所有编辑器的 linkify；<br>- `NodeHighlightPlugin`、"适应宽度"、空的 FEATURES 分组、点名的 5 个包导出。<br>Task 2、5、6 转来的基点死代码：<br>- `PROJECT_FEATURES_LIST` 只剩渲染读的 `key` 和 `property`；<br>- 通知兜底里恒真的判断；<br>- `FAVORITE_ITEM_LINKS` 按联合类型定类型，只有一个取值的 `itemLevel` 删除。<br>Plane 不路由的 8 个 service 方法删除；`CommentCreate.projectId` 改为必填。<br>原型漏掉的 `TIssueFilterKeys` 和 4 张 `imports-*.webp` 一并删除。<br>**裁定**：<br>- `is-emoji-supported` 在编辑器的 vitest 配置里外部化，改变的是测试加载它的方式，不是过滤输出，采纳；<br>- `prosemirror-codemark` 剩下的 5 行 sourcemap 警告，作为记录在案的例外（第 4 节裁定 6）；<br>- 企业版 SSO 组同步的 `group_syncing` 键记为 **F2**（报告写 40 个，整分支评审逐个数是 34 个）。<br>差异按应用层、其余部分拆成两个评审包，都 Approved，没有 Critical 或 Important。评审者运行了 `make knip`（退出码 0），并确认路由模块的差异为空。探测：探测 2 28/28，认证探测 43/46（F1 的 3 项） |

## 3. 整分支评审（opus）：Ready to merge with fixes

评审范围 `6d9692b..93635a2`，结果为 0 Critical、3 Important、9 Minor，另外核对了控制者在执行中记下的两项 F1、F2。评审者读了整个 Phase 的差异（`-U0`，新增 2384 行）和各接缝处的完整代码，复跑了 spec 第 4 节的全部工具，还写了一组只读脚本，找 knip、`symref`、`infile-orphans` 都看不见的东西：

- 新增行里的退化结构（AST）、悬空的导入分组注释、只剩一个子元素的 fragment；
- 只剩写入、没有读取的对象成员（`memberorph`、`memberdead`），没有调用方再传的 prop（`proporph`、`propdead`）；
- 约 70 个被删功能的词在代码和文案里的全部命中（`vocab.sh`）。

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| F1 | 控制者 | 错误提示里的链接不带邮箱：`AuthRoot` 调 `authErrorHandler` 时没有传地址里的 `email`，这个参数从来收不到值 | 已修（`a5403bd`）：传入 `emailParam`，并加进 effect 的依赖。安全页的调用不传：Plane 的修改密码接口只返回 3 种错误码（5138、5135、5021），消息里都没有链接，而且页面只显示字符串消息。认证探测第 4.1–4.3 项通过 |
| F2 | 控制者 | 企业版 SSO 组同步的 `group_syncing` 键仍在（评审者逐个数是 34 个，不是 40 个） | 已修（`ec26ed3`），连同 SSO 身份的 3 个键和 M6、M8 的文案 |
| I1 | Important | 编辑器里 CE 中为空的扩展点没有删：`ADDITIONAL_EXTENSIONS`（编辑器和 utils 各一份空枚举）、`ADDITIONAL_BLOCK_NODE_TYPES`（空数组）、`ADDITIONAL_ASSETS_META_DATA_RECORD`（空对象），以及从来没人设置的斜杠命令 `badge` | 已修（`0dc56bf`）：资源映射只以 `CORE_EXTENSIONS` 为键，`UniqueID` 直接读 `BLOCK_NODE_TYPES`。实现者证明了没有代码写 `options.types`：`UniqueID` 不经 `configure` 使用，每处读取都只是 `includes` 或传给 `addGlobalAttributes`，TipTap 的 `mergeDeep` 也是复制。这几个符号加进 `enterprise-shells` 规则 |
| I2 | Important | 本 Phase 删掉了唯一使用者、而工具看不见的 4 个成员：`RelationStore.getRelationByIssueIdRelationType`、`WorkspaceRootStore.getWorkspaceById`、只剩写入的 `ISlashCommandItem.commandKey`、只剩一个取值的 `FilterOption.activePulse` | 已修（`2526a22`），连同 `makeObservable` 条目和恒为假的脉冲点分支。孤儿的口径从此也覆盖对象成员和 prop |
| I3 | Important | 批量操作的链路没有删完：批量归档在基点和现在都没有调用方；命令面板的批量删除弹窗在基点、现在和 Plane 上游都打不开（没有代码把开关设为打开），spec 2.5 保留它的前提不成立 | 已修（`47e28b1`），两者都删：弹窗和它的条目组件、命令面板 store 的开关、8 个 store 的 `removeBulkIssues` / `archiveBulkIssues`、两个 service 方法和一个文案键。控制者裁定删除：M1 设计 3.9 要求整条删除，没有把它列为例外；弹窗本来打不开，行为不变。`bulk-operations` 规则去掉"不含批量删除"的说明，并覆盖被删的符号；不命中的样本取自保留的真实代码（`bulkAddMembersToProject`、`updateBulkFilters`、`markAllNotificationsAsRead`）。web 上限 567 → 566 |
| M1 | Minor | 日历根组件里 `const storeType = fallbackStoreType`，是化简 `isEpic` 后留下的别名 | 已修（`7ecdd06`） |
| M2 | Minor | 关联组件里只包一个子元素的 fragment（Task 3 改写过这段） | 已修（`7ecdd06`） |
| M3 | Minor | 没人传的 prop：工作项表单为企业版模板准备的 `showActionButtons`、`dataResetProperties`，项目创建页头的 `handleTemplateSelect` 等 4 个，`ProjectFeaturesList` 恒为真的 `isAdmin` 和不读的 `_featureKey` | 已修（`2526a22`），每个分支按默认值化简，行为不变；`isAdmin` 化简后 `ProjectFeatureToggle.disabled` 也没人传了，一并删除 |
| M4 | Minor | `drop.ts` 里附件分类的结果没人读；`TEditorCommands` 里的 `"external-embed"` 没人用 | 已修（`7ecdd06`）。`"attachment"` 也删了：基点的编辑器没有附件节点，它只是企业版的预留（第 4 节裁定 9）；上传 hook 的 `type` 随之只剩一个取值，删除。每种拖入的文件处理方式与基点相同 |
| M5 | Minor | `ExtendedAppHeader` 是 CE / EE 接缝的一半 | 已修（`7ecdd06`）：它有自己的 hook 和渲染，不是只做转发，所以改名为 `HeaderWithSidebarToggle`（`git mv`），不内联 |
| M6 | Minor | `common.publish` 以被删的发布功能命名，键工具因为图标名 `"publish"` 这个字面量把它算作有引用 | 已修（`ec26ed3`） |
| M7 | Minor | `WorkflowsPropertyIcon` 和图标注册表的 `property.workflows` 以被删的工作流命名 | 已修（`ec26ed3`）。实现者先证明了没有代码在运行时拼出注册表的键 |
| M8 | Minor | 不在 M1 设计 2.2 清单上的企业版残留：主题 store 里举措和项目概览侧边栏的状态（没人读，还写两个 localStorage 键）、项目 store 里里程碑和概览的折叠区（三个从不调用的 action）、企业版设置页和里程碑的文案、propel 的举措和模板插图 | 已修（`ec26ed3`）。后续（`8eb07ac`）又删了同一类、没有使用者的 `customer`、`update`、`dashboard` 插图 |
| M9 | Minor | 键工具的盲区：项目功能列表用 `` `${featureItem.key}_description` `` 拼键，3 个在用的键被算作无引用；`cycles_description` 在两种语言里都缺失；收集箱的说明写的是被删的收集表单 | 已修（`eb4bdff`）：`cycles_description` 是 P2 误删的，恢复原文；四个说明改为常量里的 `i18n_description` 字面量；收集箱的两处说明改为只描述保留的用法。全仓库扫描没有字面量前缀的 `t()`：只剩一处，它拼出的键都存在（交收尾） |

修复轮之外，控制者从实现者列出的"交收尾"清单里又拿回三项，因为它们是本轮改过的代码的残留，或与 M8 同类（`8eb07ac`）：只为已删的企业版子类存在的 `getCoreModalsState` 并进唯一的调用方 `isAnyModalOpen`；`project-feature-update.tsx` 悬空的导入分组注释；上面说的三张插图。

修复轮合计（`93635a2..8eb07ac`）：8 个提交，69 个文件，+149/−2230 行。每个提交都通过 `check:types`、`make lint-web`、`make knip` 和孤儿核对；最后 `make test-web` 15 个任务、`make build-web` 11 个任务通过。

### 3.1 修复后的复核（sonnet，范围复核）

修复轮 `93635a2..8eb07ac` 之后，范围复核的结论是 **Approved**：F1、F2、I1–I3、M1–M9 和后续的三项全部为 Fixed，修复本身没有引入 Critical 或 Important 问题。复核者对每一项都自己取证：

- **F1**：`auth-root.tsx` 传入 `emailParam ?? undefined`，effect 的依赖是 `[error_code, emailParam]`。按 `authentication.helper.tsx` 核对了安全页不传邮箱的理由：只有"用户已存在""用户不存在"这两个登录、注册的错误码会渲染带邮箱的链接，修改密码的错误码都不会。
- **I1**：`UniqueID` 只在 `extensions.ts` 以不经 `configure` 的方式使用；`plugin.ts`、`utils.ts` 对 `options.types` 只解构和读取，从不赋值。`CORE_ASSETS_META_DATA_RECORD` 和 `NodeFileMapType` 只以 `CORE_EXTENSIONS` 为键。
- **I3**：`isAnyModalOpen` 判断的弹窗集合与之前相同，只少了批量删除；8 个被删的符号在 `8eb07ac` 上没有任何命中。
- **M3、M4**：逐个核对了按默认值化简的分支（`showActionButtons` 为真时的操作行、`dataResetProperties` 为空时只在挂载时重置、`isClosable` 为真时的关闭按钮、封面选择器直接用 `field.onChange`）。`drop.ts` 接收的 MIME 类型不变，附件类型的拖入仍像基点一样被吞掉；上传 hook 去掉 `type` 之后，它原来唯一的取值是 `"image"`。
- **M9**：`cycles_description` 的中英文与 `e3760cb` 删掉的原文逐字节相同，位置也相同。复核者自己扫描了没有字面量前缀的 `t()`，结果与实现者相同：只剩 `issue-description.tsx:79` 一处（交收尾）。复核者另外认为 `profile-view.tsx` 按模板读取的 `profile.empty_state.*` 没有对应的键；控制者核对后不成立：这些键在 `settings.json` 里（`assigned`、`created`、`subscribed` 各有 `title`、`description`，中英文都有），P2 的探测 1 也看到了"No work items are assigned to you"。
- **复跑**：`node tools/keywords.mjs`（42 条规则、4 条例外、没有命中），`alts.mjs M1/P3`（0），i18n 的 `check:sync`，编辑器测试 16 个（输出只有那 5 行 sourcemap 警告），constants 测试 11 个，`make knip`（退出码 0），`make lint-web`（52 个任务）。修复轮新增的行里没有无插值的模板字符串、`={"…"}` 或新的导入分组注释。

复核者提出一条 Minor：`enterprise-shells` 规则有一个不命中样本引用的是 `ExtendedAppHeader` 的导入，而修复轮已经把它改名，这个样本不再是仓库里的真实代码。守卫本身不受影响（它只检查样本不命中）。控制者把它换成扩展侧边栏里真实存在的 `ExtendedSidebarWrapper` 导入（`e405bd2`），守卫和 `alts.mjs` 复跑通过。

修复轮之后，控制者在 `8eb07ac` 的构建上重跑了两段临时核对脚本：认证 46 项、探测 2 共 28 项，全部通过，没有未列出的请求；探测 2 另外确认端口已释放、没有残留的浏览器进程。附录 6.2、6.3 的输出就是这次重跑的结果。

## 4. 控制者的裁定

执行中的裁定都按"能自己定的就自己定"的原则做出：只要符合 SOLID、从根源解决、不打补丁，就不升级给用户。没有一项属于架构级、跨模块或意料之外的高风险，因此都没有升级。

1. **原型只是对照答案。** 逐步执行，允许逐文件参考原型、使用架构师写的精确替换脚本，但不允许整体 checkout、cherry-pick、read-tree 或 apply；每一处差异都要说明。代价是 Task 慢一些。收获是发现了原型的十几处错误和遗漏（第 5 节），其中 Task 1 的错误码查找、Task 3 的表头优先级和类名粘连，如果整体搬运都会直接进入主分支。
2. **孤儿的口径。** 以前一个 Task 的真实提交为基点，不用原型的提交。P2 的"本 Task 删除的代码是某个符号唯一的使用者"在 P3 补了两种 `symref` 看不见的情况：
   - 同一文件内的使用者（Task 2 起用 `infile-orphans.mjs`，列出的每一行都必须是 `defined now: 0`）；
   - 常量化简留下的插值外壳（Task 4 起在新增的行上 grep 没有插值的模板字符串，以及单个插值和花括号字符串属性）。
3. **数字按实测**（P2 裁定 2）。上限只降不升，计划里的数字只作参考。最终与 spec 的差异都有来源：
   - lint 上限合计 711，不是 713：Task 1 修复轮去掉了基点的一条 `no-unneeded-ternary`，修复轮删掉的批量删除弹窗带走一条 `promise(always-return)`；
   - 无引用的键 498，不是 570：Task 3 删了 6 个以工作流命名的键；修复轮删了 63 个标为无引用的键（第 3 节），另有 3 个项目功能说明改为字面量引用后不再算作无引用；
   - 无引用的图片 136 张，不是 142 张：Task 1 的 2 张、Task 7 的 4 张（修复轮删的是 propel 的 TSX 插图，不计入图片数）；
   - 基线就没人用的包导出 347 个，不是 355 个：Task 2 的 5 个同文件孤儿类型，Task 7 的 `TIssueFilterKeys` 等，修复轮的 `PROJECT_MILESTONES`。Task 7 报告写的是 349，修复轮的实现者在 `93635a2` 上用同一脚本测得 348，差的 1 个没有追查，以最终实测的 347 为准。
4. **以被删功能命名的文案和图片**（P2 裁定 3 延续到 P3）：随功能删，即使基点就已无引用。本 Phase 因此多删了：
   - 键：Task 3 的 6 个工作流键；修复轮的 64 个，即 SSO 组同步 34 个、身份（SSO）3 个、企业版的工作区设置页（运行器、项目状态、项目、关联）15 个、里程碑 8 个、项目概览 3 个、发布 1 个；
   - 图片：Task 1 的 2 张实例设置图，Task 7 的 4 张导入图；
   - propel 的插图 `initiative`、`template`、`customer`、`update`、`dashboard` 和图标 `WorkflowsPropertyIcon`（修复轮）。
5. **退化结构。** 本 Phase 的删除造成的退化结构，由造成它的 Task 收掉：Task 3 的插值外壳、Task 4 封面帮助函数里的重复 return、Task 5 只剩一个子元素的 fragment。改动文件里基点就存在的死代码交给 Task 7 统一删（第 2 节 Task 5、6 两行）。Task 2 工具栏的两层 `div` 样式不同，不算退化结构。
6. **测试输出。** 测试输出要干净：声明测试环境可以，过滤日志不行。
   - linkify 的"already initialized"：Task 7 从根源去掉（不再注册多余的协议）。
   - `is-emoji-supported` 的 sourcemap 警告：Task 7 在编辑器的 vitest 配置里把它外部化，改变的是测试加载它的方式，不是过滤。
   - `prosemirror-codemark` 剩下的 5 行：原因是这个依赖的打包方式。`mainFields: ["module","main"]` 让它解析到 `dist/esm/*.js`，Vitest 把 `esm` 目录下的文件内联交给 Vite 转换；Vite 读取文件末尾的 `sourceMappingURL`，逐个补 `sourcesContent`，而它发布的 map 指向包里没有的 `../../src/*.ts`，于是警告。这 5 行在 `6d9692b` 就存在。合法的办法都试过了：
     - 外部化：它的 ESM 构建导入 `./plugin` 时不带扩展名，Node 加载失败；
     - 预打包（optimizer）：预打包带进自己的一份 `prosemirror-state`，又出现 P2 的"两份实例"问题（`RangeError: Adding different instances of a keyed plugin`）；
     - `fallbackCJS`：没有效果。

     剩下的办法要么是过滤（不允许），要么是替换或内置这个包（这是依赖决定，不在 P3 的范围内）。**裁定**：记录在案，作为"测试输出干净"的一个例外，交给 M1 收尾的依赖复核（查新版本是否带 `sourcesContent`）。判断错了的代价是一个包的测试日志里有 5 行已知的警告。
7. **SDD 工作区不能让 knip 多报。** Task 1 时 knip 把 `.superpowers/sdd/P3-trim-platform/p3tmp/` 下备份的 18 个 `.mjs` 脚本算作未使用文件（knip 不读嵌套的 `.gitignore`）。不加 knip 的忽略项，而是从根源改：备份改为 `p3tmp.tar`，目录删除。否则 Task 7 的 knip 门禁在本地会失败。
8. **可见的变化**，都随被删功能而来：
   - 注册失败后回到登录页，错误提示显示在登录表单上方（spec 第 3 节第 2 条，交 M2）；
   - 错误提示的两个链接改为指向各自模式的路由，并带上填写的邮箱；
   - 登录页、注册页的标题复原（修正 Plane 自己写反的问题）；
   - 工作项表单描述编辑器下方的空白 AI 行删除（少了 24px 内边距）；
   - 侧边栏底部只放版本徽标的那一栏删除，版本号仍在帮助菜单里；
   - 图片选择器只有"图片""上传"两个标签页，打开时不再请求 `/api/unsplash/`；
   - 列表、看板、表格没有选择框；
   - 项目的"邀请成员"改名为"添加成员"。
9. **M1 设计 3.1 的"附件节点"。** 设计要求保留"图片和附件的节点"，但基点的编辑器里没有附件节点（它只在企业版里存在），只有为它预留的 `"attachment"` 命令类型、上传参数和 `drop.ts` 里的空分支。修复轮删掉的是这些预留，没有删除任何节点；拖入非图片文件时编辑器的行为与基点相同（交 M5）。
10. **P2 的一处回归在 P3 发现并修复。** P2 Task 2（`e3760cb`）删掉了 `cycles_description`，但项目功能列表用 `` `${featureItem.key}_description` `` 拼出这个键，于是项目创建后"功能"一步里迭代那一行显示原始键名。原因是严格方法看不见只有后缀的拼接。修复轮恢复了中英文原文，并把四个说明改成常量里的字面量（`i18n_description`），键工具从此能看见它们。全仓库只有另一处没有字面量前缀的 `t()` 调用，它拼出的键都存在（交收尾）。
11. **控制者的探测会发现评审发现不了的问题。** Task 1 的错误链接回归，两位评审者都没有发现，是认证探测发现的。F1 也来自探测。探测在 Task 1、3、5、6、7 之后各跑一次，并在基点的构建上做了反向对照（附录 6.4）。
12. **执行过程。** Task 3、Task 7 的实现者有几次在一条 Bash 命令里串了只读的 git 命令，历史没有变化；之后的派发都重申了"一条命令只做一件 git 事"。

## 5. 计划缺陷

计划由原型写成，执行时发现以下缺陷，都已在对应 Task 或修复轮中修正，不需要改动计划文件本身：

| Task | 缺陷 | 实际做法 |
|---|---|---|
| 全部 | 预期的 lint 上限、键数、图片数、包导出数 | 以实测为准（第 4 节裁定 3） |
| 1 | 原型用 `errorCodeMessages[errorCode]` 查地址里的错误码 | 先判断是否为枚举成员；否则 `?error_code=toString` 会让登录页崩溃 |
| 1 | 错误提示的两个链接还按被删的"检查邮箱"步骤写（`/?email=`、`/sign-in?email=`） | 指向 `/sign-up?email=`、`/?email=`（修复轮），邮箱由 F1 带上 |
| 1 | 页头的两个 `pageTitle` 写反（Plane 的问题，原型照搬） | 复原 |
| 1 | `authErrorHandler` 的 `email` 参数从来收不到值 | F1：`AuthRoot` 传入地址里的 `email` |
| 1 | 以被删页面命名的 2 张实例图片没有删 | 删除 |
| 2 | 孤儿检查只用 `symref.mjs orphaned`，看不见同一文件内的使用者 | 删掉 `issue_comment.ts` 的 7 个类型；之后每个 Task 加 `infile-orphans.mjs` |
| 3 | 表格表头列的判断优先级写错 | 按基点的逻辑保留假的那一支 |
| 3 | `bulk.py` 把 `:bg-accent-primary/10` 粘到 14 个文件的类名上 | 逐个文件按基点的类名修改 |
| 3 | `displayProperties?.key &&` 改变了列表块的行为 | 保留基点行为（`issue_type` 恒为真时的那一支） |
| 3 | 提交前的检查只 grep `${"`，看不到其他插值外壳 | 修复轮收掉 10 处，之后加两条 grep |
| 3 | 以工作流命名的 6 个键没有删 | 删除 |
| 4 | 漏掉 `setEditorValueAtCursorPosition` 和 `insert-content-at-cursor-position.ts` | 删除 |
| 4 | 封面帮助函数的重复 return 和不可达 return | 修复轮收掉 |
| 6 | 收藏的文件夹会丢掉 logo | 保留基点行为 |
| 7 | 漏掉 `TIssueFilterKeys` 和 4 张 `imports-*.webp` | 删除 |
| 7 | 漏掉 34 个 `group_syncing` 键（Task 7 报告写 40，整分支评审逐个数是 34） | F2 |
| 3 | spec 2.5 保留命令面板的 `BulkDeleteIssuesModal`，前提是它能打开；批量归档的整条链路也没有列入 | 两者在基点都没有入口，修复轮删除（I3）；M1 设计 3.9 本来就要求整条删除 |
| 6 | 编辑器里在社区版中为空的 `ADDITIONAL_*` 常量和枚举、斜杠命令的 `badge` 没有列进扩展点清单 | 修复轮删除，并加进 `enterprise-shells` 规则（I1） |
| 全部 | 孤儿检查只看导出和文件，看不见对象成员和 prop | 修复轮删除本 Phase 造成的 4 个（I2）；基点就没人用的交收尾 |
| 全部 | 以被删功能命名的文案只按本 Phase 规则的词汇找，漏了 SSO 身份、企业版设置页、里程碑、项目概览 | 修复轮一并删除（第 4 节裁定 4） |

## 6. 附录（M1 设计 7.5）

临时核对脚本不进仓库（M1 设计 7.5），全文、假数据、运行命令、断言和输出写在这里。两段脚本都在 `$P3TMP`（`/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p3`）下，跑在 `$P3TMP/probe-app` 里：那是本仓库的一个克隆，检出到被测的提交，由 `setup-probe-app.sh <提交>` 安装依赖、执行 `make build-web`。脚本用 node 起一个静态服务器提供 `web/apps/web/build/client`（单页应用的回退到 `index.html`），用 e2e 包里的 Playwright 驱动 Chromium，路径以 `/api/`、`/auth/` 开头的请求全部由脚本里的桩回答；**当前场景没有列出的请求一律算失败**（spec 2.13）。

### 6.1 两处进仓库的测试

- `web/packages/constants/src/navigation.test.ts`：P2 的 5 个，加上 Task 1、5、7 各 2 个（个人设置、工作区设置的标签页各在一个分组里，工作区设置没有空分组，保留地址不重复）。
- `web/packages/editor/src/extensions/slash-commands/command-items-list.test.ts`（Task 6 新增）：图片项紧跟代码块；禁用图片时没有图片项。

在 `8eb07ac` 上的输出（`pnpm --dir web/packages/constants exec vitest run --reporter=verbose`，编辑器包同样；路径前缀缩写为 `<repo>`）：

```
 RUN  v4.1.11 <repo>/web/packages/constants

 ✓ src/navigation.test.ts > the workspace sidebar is a fixed list > shows these items above the workspace group, in this order 1ms
 ✓ src/navigation.test.ts > the workspace sidebar is a fixed list > shows these items inside the workspace group, in this order 0ms
 ✓ src/navigation.test.ts > the workspace sidebar is a fixed list > gives every item a label, a link and at least one role 0ms
 ✓ src/navigation.test.ts > the profile page > keeps exactly the three work-item tabs 0ms
 ✓ src/navigation.test.ts > the profile page > points each tab at its own route 0ms
 ✓ src/navigation.test.ts > the profile settings > keeps exactly these tabs 0ms
 ✓ src/navigation.test.ts > the profile settings > shows every tab in exactly one sidebar group 0ms
 ✓ src/navigation.test.ts > the workspace settings > keep exactly these tabs 0ms
 ✓ src/navigation.test.ts > the workspace settings > show every tab in exactly one sidebar group 0ms
 ✓ src/navigation.test.ts > the workspace settings > have no empty sidebar group 0ms
 ✓ src/navigation.test.ts > the reserved workspace addresses > list each address once 0ms

 Test Files  1 passed (1)
      Tests  11 passed (11)
```

```
 RUN  v4.1.11 <repo>/web/packages/editor

 ✓ src/extensions/slash-commands/command-items-list.test.ts > the slash command list > offers an image right after the code block 1ms
 ✓ src/extensions/slash-commands/command-items-list.test.ts > the slash command list > offers no image when the editor disables images 0ms
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/index.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/plugin.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/utils.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/inputRules.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/actions.js" points to missing source files
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > installs no collaboration extension 1ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > keeps undo and redo local, in the history of the starter kit 0ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > switches the history off when the caller asks it to 0ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > keeps the nodes the description, the comments and the history view need 0ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > leaves the image out when the caller disables it 0ms
 ✓ src/extensions/extensions.test.ts > the toolbar > has one set of items, since one editor type is left 0ms
 ✓ src/extensions/extensions.test.ts > the toolbar > offers no item that only the deleted page editor showed 0ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > takes typed text and gives it back as HTML and as Markdown 57ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > undoes and redoes typing with the keyboard 14ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > shows a description read-only, and edits it once it is editable again 21ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > inserts a user mention that is saved, read back and copied as Markdown 38ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > inserts an image from the toolbar, and keeps an uploaded image through a save 17ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > gives every block node an id, keeps it through a save and gives new blocks new ones 19ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > creates editors one after another in one process: editable, read-only, editable 21ms

 Test Files  3 passed (3)
      Tests  16 passed (16)
```

编辑器输出里的 5 行 `Sourcemap for …` 是 `prosemirror-codemark` 的打包方式造成的，基点就有，记录在案（第 4 节裁定 6）。

### 6.2 临时核对脚本 1：认证与实例（7.5 "P3 认证、实例"）

#### 1. 目的

核对 Task 1 改写的登录、注册表单，实例的几种状态，以及保留的账户操作：表单提交的字段、CSRF 令牌、`next_path` 和结果跳转；失败后回到 `/` 并显示错误；错误提示里两个链接的去向和邮箱；关闭注册时页头没有"注册"；实例请求失败时显示维护页；修改密码带 CSRF 请求头提交；个人设置只剩 4 个标签页；退出带 CSRF 令牌提交。

#### 2. 脚本全文（`$P3TMP/probe-auth/auth-instance.mjs`）

```js
// One-off (M1/P3 retained-behaviour probe 1, M1 design 7.5 row "P3 认证、实例", P3 spec 2.13): serves the built web
// app (web/apps/web/build/client, SPA fallback to its index.html) on a free port and checks the sign-in and sign-up
// forms, the instance states and the account flows that P3 Task 1 rewrote. Every request whose path starts with
// /api/ or /auth/ goes to the stubs below; a request the running scenario does not list gets 404 and fails the
// scenario's "no unlisted request" check. Form posts (sign-in, sign-up, sign-out) are browser navigations: the stub
// records the urlencoded body and answers with the redirect Plane's Django views send. Each scenario runs in a
// fresh browser context. Run from the repository root (Playwright comes from e2e/).
// usage: node auth-instance.mjs
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");

// ---------------------------------------------------------------- fake data
const WS = "probe-ws";
const ME = {
  id: "u1", email: "probe@example.com", display_name: "probe", first_name: "Probe", last_name: "User",
  avatar_url: "", cover_image_url: null, is_active: true,
};
const PROJECT = {
  id: "p1", name: "Probe Project", identifier: "PRB", sort_order: 65535, logo_props: {}, member_role: 20,
  archived_at: null, workspace: "w1", cycle_view: true, module_view: true, issue_views_view: true, inbox_view: true,
};
// What an instance sends once P3 is done: the four fields the web app still reads (M1 design 3.7, P3 spec 2.11).
const config = (extra = {}) => ({
  enable_signup: true, is_workspace_creation_disabled: false, file_size_limit: 5242880, is_self_managed: true, ...extra,
});
const CSRF = "probe-csrf-token";

class Reply {
  constructor(status, json, headers = {}) {
    Object.assign(this, { status, json, headers });
  }
}
const redirectTo = (location) => new Reply(302, undefined, { location });

// Nobody signed in: the instance, the CSRF token and a 401 for the current user.
const anonymousStubs = (instanceConfig = config()) => ({
  "GET /api/instances/": { config: instanceConfig },
  "GET /api/users/me/": new Reply(401, { error_code: 5000, error_message: "AUTHENTICATION_FAILED" }),
  "GET /auth/get-csrf-token/": { csrf_token: CSRF },
});
// An onboarded admin of probe-ws with project p1.
const signedInStubs = () => ({
  "GET /api/instances/": { config: config() },
  "GET /api/users/me/": ME,
  "GET /api/users/me/profile/": { id: "pr1", user: "u1", language: "en", is_onboarded: true, is_tour_completed: true, theme: {} },
  "GET /api/users/me/settings/": { id: "u1", email: ME.email, workspace: { last_workspace_slug: WS } },
  "GET /api/users/me/workspaces/": [{ id: "w1", slug: WS, name: "Probe WS", total_members: 1, role: 20 }],
  [`GET /api/workspaces/${WS}/workspace-members/me/`]: { id: "wm-u1", member: "u1", workspace: "w1", role: 20, is_active: true },
  [`GET /api/users/me/workspaces/${WS}/project-roles/`]: { p1: 20 },
  [`GET /api/workspaces/${WS}/projects/`]: [PROJECT],
  [`GET /api/workspaces/${WS}/members/`]: [{ id: "wm-u1", member: ME, role: 20, is_active: true }],
  [`GET /api/workspaces/${WS}/states/`]: [],
  [`GET /api/workspaces/${WS}/user-favorites/`]: [],
  [`GET /api/workspaces/${WS}/user-properties/`]: { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 },
  [`GET /api/workspaces/${WS}/projects/details/`]: [PROJECT],
  [`GET /api/workspaces/${WS}/users/notifications/unread/`]: { total_count: 0, mention_unread_notifications_count: 0 },
  "GET /auth/get-csrf-token/": { csrf_token: CSRF },
});

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
const check = (name, ok, detail = "") => {
  results.push({ name, ok });
  console.log(ok ? `PASS ${name}` : `FAIL ${name}: ${detail}`);
};
const visible = async (locator, timeout = 10000) => {
  try {
    await locator.first().waitFor({ state: "visible", timeout });
    return true;
  } catch {
    return false;
  }
};
const bodyText = (page) => page.locator("body").innerText();
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const REMOVED_AUTH = /email-check|magic|unique-code|forgot-password|reset-password|set-password|oauth|google|github|gitlab|gitea/i;
const REMOVED_TEXT = /unique code|forgot (your )?password|continue with (google|github|gitlab|gitea)|sign in with|god mode|set a password/i;

let browser;
let base;
async function scenario(name, stubs, fn) {
  console.log(`\n== ${name}`);
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "en-US" });
  const page = await context.newPage();
  const ctx = { requests: [], unstubbed: [], pageErrors: [] };
  page.on("pageerror", (e) => ctx.pageErrors.push(e.message.split("\n")[0]));
  await context.route(
    (url) => url.pathname.startsWith("/api/") || url.pathname.startsWith("/auth/"),
    async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      const key = `${request.method()} ${url.pathname}`;
      ctx.requests.push({
        key, search: url.search, body: request.postData(), headers: request.headers(),
        navigation: request.isNavigationRequest(),
      });
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
    check(`${name}: no request outside the scenario's list`, ctx.unstubbed.length === 0, JSON.stringify(ctx.unstubbed));
    check(`${name}: no uncaught page error`, ctx.pageErrors.length === 0, JSON.stringify(ctx.pageErrors));
  } catch (error) {
    check(`${name}: ran to the end`, false, error.message.split("\n")[0]);
  } finally {
    await context.close();
  }
}
const goto = (page, p) => page.goto(`${base}${p}`, { waitUntil: "domcontentloaded" });
const landsOn = async (page, pattern) => {
  const ok = await page.waitForURL(pattern, { timeout: 20000 }).then(() => true, () => false);
  const url = new URL(page.url());
  console.log(`  (landed on ${url.pathname}${url.search})`);
  return ok;
};
const formPost = (ctx, pathname) => ctx.requests.filter((r) => r.key === `POST ${pathname}`);
const fields = (request) => Object.fromEntries(new URLSearchParams(request?.body ?? ""));
const emailInput = (page) => page.locator("form input#email");
const submitButton = (page) => page.locator("form button[type=submit]");
// A link's target as "<path without the trailing slash>?<query>": the web app's Link adds a trailing slash.
const target = (href) => {
  if (!href) return `${href}`;
  const url = new URL(href, "http://x");
  return `${url.pathname.replace(/(.)\/$/, "$1")}${url.search}`;
};
const signInFormShown = async (page) =>
  (await visible(emailInput(page), 20000)) &&
  (await page.locator("form input#confirm-password").count()) === 0 &&
  /^go to workspace$/i.test((await submitButton(page).innerText()).trim());

// ---------------------------------------------------------------- scenarios
async function run() {
  // 1. The sign-in page: one email and password form, the email editable, the post carries the CSRF token and
  //    next_path; a failure comes back to / with the error above the form.
  await scenario("sign-in fails", {
    ...anonymousStubs(),
    "POST /auth/sign-in/": redirectTo(`/?error_code=5065&email=${encodeURIComponent("typed@example.com")}`),
  }, async (page, ctx) => {
    await goto(page, `/?next_path=${encodeURIComponent(`/${WS}/projects/`)}`);
    check("1.1 / shows the sign-in form: email, password, no confirmation, \"Go to workspace\"",
      await signInFormShown(page), await bodyText(page).catch(() => ""));
    check("1.1b the document title names the sign-in page", /^Sign in\b/.test(await page.title()), await page.title());
    const text = await bodyText(page);
    check("1.2 no sign-in code, forgot-password, third-party or admin entry on the page",
      !REMOVED_TEXT.test(text), `${text.match(REMOVED_TEXT)}`);
    const header = page.getByRole("link", { name: "Sign up" });
    check("1.3 sign-up is enabled, so the header links to /sign-up",
      (await visible(header)) && target(await header.getAttribute("href")) === "/sign-up",
      `${await header.getAttribute("href").catch(() => "none")}`);
    await emailInput(page).fill("first@example.com");
    await page.getByRole("button", { name: /clear email/i }).click();
    const cleared = await emailInput(page).inputValue();
    await emailInput(page).fill("typed@example.com");
    check("1.4 the email field is editable and its clear button empties it",
      cleared === "" && (await emailInput(page).inputValue()) === "typed@example.com", `after clear: "${cleared}"`);
    await page.locator("form input#password").fill("Wrong-password-1");
    await submitButton(page).click();
    const landed = await landsOn(page, (url) => url.pathname === "/" && url.searchParams.get("error_code") === "5065");
    const post = formPost(ctx, "/auth/sign-in/");
    const body = fields(post[0]);
    check("1.5 the form posts once, as a page navigation, to /auth/sign-in/",
      post.length === 1 && post[0].navigation, JSON.stringify(post.map((p) => p.key)));
    check("1.6 the post carries the CSRF token, the typed email, the password and next_path",
      body.csrfmiddlewaretoken === CSRF && body.email === "typed@example.com" &&
        body.password === "Wrong-password-1" && body.next_path === `/${WS}/projects/`,
      JSON.stringify(body));
    check("1.7 the CSRF token was fetched before the post",
      ctx.requests.findIndex((r) => r.key === "GET /auth/get-csrf-token/") <
        ctx.requests.findIndex((r) => r.key === "POST /auth/sign-in/"),
      JSON.stringify(ctx.requests.map((r) => r.key)));
    check("1.8 the failure lands on / with \"Authentication failed\" above the sign-in form, the email kept",
      landed && (await visible(page.getByRole("alert").filter({ hasText: "Authentication failed. Please try again." }))) &&
        (await signInFormShown(page)) && (await emailInput(page).inputValue()) === "typed@example.com",
      await bodyText(page));
    check("1.9 no request to a deleted sign-in method or step",
      !ctx.requests.some((r) => REMOVED_AUTH.test(r.key)), JSON.stringify(ctx.requests.map((r) => r.key)));
  });

  await scenario("sign-in succeeds", (() => {
    const stubs = { ...anonymousStubs() };
    // After the post the session exists: every later request sees the signed-in user.
    stubs["POST /auth/sign-in/"] = () => {
      Object.assign(stubs, signedInStubs());
      return redirectTo(`/${WS}/projects/`);
    };
    return stubs;
  })(), async (page, ctx) => {
    await goto(page, `/?next_path=${encodeURIComponent(`/${WS}/projects/`)}`);
    await visible(emailInput(page), 20000);
    await emailInput(page).fill(ME.email);
    await page.locator("form input#password").fill("Right-password-1");
    await submitButton(page).click();
    const landed = await landsOn(page, (url) => url.pathname.startsWith(`/${WS}/projects`));
    await sleep(2000);
    const stays = new URL(page.url()).pathname.startsWith(`/${WS}/projects`);
    check("2.1 after a successful sign-in the next path opens and stays open (no bounce back to /)",
      landed && stays && (await visible(page.getByText("Probe Project"), 20000)), page.url());
    check("2.2 the post carried the CSRF token and next_path",
      fields(formPost(ctx, "/auth/sign-in/")[0]).csrfmiddlewaretoken === CSRF &&
        fields(formPost(ctx, "/auth/sign-in/")[0]).next_path === `/${WS}/projects/`,
      JSON.stringify(fields(formPost(ctx, "/auth/sign-in/")[0])));
  });

  // 3. The sign-up page: email editable, a weak password never posts, a failure lands on / as an error above the
  //    sign-in form (P3 spec section 3 item 2: the mode comes from the route; handed to M2).
  await scenario("sign-up fails", {
    ...anonymousStubs(),
    "POST /auth/sign-up/": redirectTo(`/?error_code=5035&email=${encodeURIComponent("new@example.com")}`),
  }, async (page, ctx) => {
    await goto(page, "/sign-up");
    const confirm = page.locator("form input#confirm-password");
    check("3.1 /sign-up shows email, password, confirmation and \"Create account\"",
      (await visible(emailInput(page), 20000)) && (await visible(confirm)) &&
        /^create account$/i.test((await submitButton(page).innerText()).trim()),
      await bodyText(page).catch(() => ""));
    const signIn = page.getByRole("link", { name: "Sign in" });
    check("3.2 the header links straight to the sign-in page (/), titled \"Sign up\"",
      (await visible(signIn)) && target(await signIn.getAttribute("href")) === "/" && /^Sign up\b/.test(await page.title()),
      `${await signIn.getAttribute("href").catch(() => "none")} / ${await page.title()}`);
    await emailInput(page).fill("new@example.com");
    await page.locator("form input#password").fill("weak");
    await confirm.fill("weak");
    await submitButton(page).click();
    await sleep(1000);
    check("3.3 a weak password shows the strength banner and posts nothing",
      formPost(ctx, "/auth/sign-up/").length === 0 &&
        (await visible(page.getByText(/password/i).filter({ hasText: /strong|strength|criteria|weak/i }))),
      await bodyText(page));
    await page.locator("form input#password").fill("Strong-password-2026!");
    await confirm.fill("Strong-password-2026!");
    await submitButton(page).click();
    const landed = await landsOn(page, (url) => url.pathname === "/" && url.searchParams.get("error_code") === "5035");
    const body = fields(formPost(ctx, "/auth/sign-up/")[0]);
    check("3.4 the sign-up post carries the CSRF token, the typed email and the password",
      body.csrfmiddlewaretoken === CSRF && body.email === "new@example.com" && body.password === "Strong-password-2026!",
      JSON.stringify(body));
    check("3.5 the failure lands on / as \"Authentication failed\" above the sign-in form (accepted change, M2)",
      landed && (await visible(page.getByRole("alert").filter({ hasText: "Authentication failed. Please try again." }))) &&
        (await signInFormShown(page)),
      await bodyText(page));
  });

  // 4. The error links between the two forms point at the route that decides the mode.
  await scenario("error links between the forms", anonymousStubs(), async (page) => {
    await goto(page, `/?error_code=5060&email=${encodeURIComponent("nobody@example.com")}`);
    const createOne = page.getByRole("alert").getByRole("link", { name: "Create one" });
    check("4.1 \"No account found\" links to the sign-up page with the email",
      (await visible(createOne, 20000)) &&
        target(await createOne.getAttribute("href")) === `/sign-up?email=${encodeURIComponent("nobody@example.com")}`,
      `${await createOne.getAttribute("href").catch(() => "none")}`);
    await createOne.click();
    check("4.2 following it opens the sign-up form with the email filled in",
      (await landsOn(page, (url) => url.pathname.replace(/\/$/, "") === "/sign-up")) &&
        (await visible(page.locator("form input#confirm-password"))) &&
        (await emailInput(page).inputValue()) === "nobody@example.com",
      await bodyText(page));
    await goto(page, `/sign-up?error_code=5030&email=${encodeURIComponent("taken@example.com")}`);
    const signInLink = page.getByRole("alert").getByRole("link", { name: "Sign In" });
    check("4.3 \"already registered\" links to the sign-in page with the email",
      (await visible(signInLink, 20000)) &&
        target(await signInLink.getAttribute("href")) === `/?email=${encodeURIComponent("taken@example.com")}`,
      `${await signInLink.getAttribute("href").catch(() => "none")}`);
  });

  // 5. An error_code that is not a code (it comes from the address) shows nothing and breaks nothing.
  await scenario("unknown error code", anonymousStubs(), async (page) => {
    await goto(page, "/?error_code=toString");
    check("5.1 /?error_code=toString shows the sign-in form without an error banner",
      (await signInFormShown(page)) && (await page.getByRole("alert").count()) === 0, await bodyText(page));
  });

  // 6. Sign-up disabled: no sign-up link in the header.
  await scenario("sign-up disabled", anonymousStubs(config({ enable_signup: false })), async (page) => {
    await goto(page, "/");
    await visible(emailInput(page), 20000);
    const text = await bodyText(page);
    check("6.1 with enable_signup off the header has no \"Sign up\" link and no \"new to\" line",
      (await page.getByRole("link", { name: "Sign up" }).count()) === 0 && !/new to/i.test(text), text);
  });

  // 7. The instance request fails: the maintenance view, no form.
  await scenario("instance request fails", {
    ...anonymousStubs(),
    "GET /api/instances/": new Reply(500, { error: "down" }),
  }, async (page) => {
    await goto(page, "/");
    check("7.1 a failed instance request shows the maintenance view and no sign-in form",
      (await visible(page.getByText(/didn.t start up correctly/), 20000)) && (await emailInput(page).count()) === 0,
      await bodyText(page).catch(() => ""));
  });

  // 8. Signed in: the profile settings keep four tabs; the password change asks for the old password and posts with
  //    the CSRF header.
  await scenario("password change", {
    ...signedInStubs(),
    "POST /auth/change-password/": {},
  }, async (page, ctx) => {
    await goto(page, "/settings/profile/security");
    await visible(page.locator("input[name=old_password], #old_password"), 20000);
    // The tabs are buttons in the settings sidebar, one block per category (the class names are the ones
    // item-categories.tsx renders); the workspace list below them is not a tab.
    const security = page.getByRole("button", { name: "Security", exact: true }).first();
    await visible(security);
    const tabs = await security
      .locator("xpath=ancestor::div[contains(@class,'gap-y-4')][1]/div[contains(@class,'shrink-0')]//button")
      .evaluateAll((bs) => bs.map((b) => b.innerText.trim()).filter(Boolean));
    check("8.1 the profile settings sidebar offers exactly Profile, Security, Preferences, Personal Access Tokens",
      JSON.stringify(tabs.toSorted()) ===
        JSON.stringify(["Profile", "Security", "Preferences", "Personal Access Tokens"].toSorted()),
      JSON.stringify(tabs));
    const inputs = page.locator("main input[type=password], main input[type=text]");
    check("8.2 the security page has old, new and confirmation password fields", (await inputs.count()) === 3,
      `${await inputs.count()} inputs`);
    await inputs.nth(0).fill("Old-password-1!");
    await inputs.nth(1).fill("New-password-2026!");
    await inputs.nth(2).fill("New-password-2026!");
    await page.getByRole("button", { name: "Change password" }).click();
    const done = await visible(page.getByText("Password changed successfully."), 15000);
    const post = ctx.requests.filter((r) => r.key === "POST /auth/change-password/");
    let json = {};
    try {
      json = JSON.parse(post[0]?.body ?? "{}");
    } catch {}
    check("8.3 the change posts old and new password with the X-CSRFToken header and reports success",
      done && post.length === 1 && post[0].headers["x-csrftoken"] === CSRF &&
        json.old_password === "Old-password-1!" && json.new_password === "New-password-2026!",
      JSON.stringify({ post: post.map((p) => ({ headers: p.headers, body: p.body })) }));
    check("8.4 no request to a deleted account endpoint (set-password, notification preferences, email change)",
      !ctx.requests.some((r) => /set-password|notification-preferences|email\/generate-code|instance-admin/.test(r.key)),
      JSON.stringify(ctx.requests.map((r) => r.key)));
  });

  // 9. Signed in: sign-out posts the CSRF token.
  await scenario("sign-out", (() => {
    const stubs = { ...signedInStubs() };
    stubs["POST /auth/sign-out/"] = () => {
      Object.assign(stubs, anonymousStubs());
      return redirectTo("/");
    };
    return stubs;
  })(), async (page, ctx) => {
    await goto(page, `/${WS}/projects/`);
    await visible(page.getByText("Probe Project"), 20000);
    const avatar = page.locator("#main-sidebar, nav, aside").getByText("P", { exact: true });
    const menus = page.getByRole("button").filter({ has: page.getByText("P", { exact: true }) });
    await (await menus.count() ? menus.last() : avatar.last()).click();
    await page.getByRole("menuitem", { name: /sign out/i }).or(page.getByText(/^sign out$/i)).first().click();
    const landed = await landsOn(page, (url) => url.pathname === "/");
    const body = fields(formPost(ctx, "/auth/sign-out/")[0]);
    check("9.1 sign-out posts the CSRF token as a page navigation and lands on the sign-in form",
      landed && body.csrfmiddlewaretoken === CSRF && formPost(ctx, "/auth/sign-out/")[0]?.navigation &&
        (await signInFormShown(page)),
      JSON.stringify(body));
  });
}

// ---------------------------------------------------------------- main
server.listen(0, "127.0.0.1", async () => {
  base = `http://127.0.0.1:${server.address().port}`;
  browser = await chromium.launch();
  try {
    await run();
  } finally {
    await browser.close();
    server.close();
  }
  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length} passed, ${failed.length} failed`);
  process.exitCode = failed.length ? 1 : 0;
});
```

#### 3. 假数据

写在脚本开头（`fake data` 一段）：

- 实例：P3 之后前端还读的 4 个字段（`enable_signup: true`、`is_workspace_creation_disabled: false`、`file_size_limit`、`is_self_managed: true`），各场景按需覆盖；
- 未登录：实例、CSRF 令牌 `probe-csrf-token`、`/api/users/me/` 返回 401；
- 已登录：用户 `u1`（`probe@example.com`），已完成新手引导，是工作区 `probe-ws` 的管理员，有项目 `p1`（`PRB`）；工作区页面加载时的成员、状态、收藏、用户属性、未读通知等只给最小数据；
- 表单提交（登录、注册、退出）是浏览器的页面跳转：桩记下 urlencoded 的请求体，按 Plane 的 Django 视图回一个重定向（失败时 `/?error_code=…&email=…`）；
- 每个场景在全新的浏览器上下文里运行（1440×900，`en-US`）。

#### 4. 运行命令

```sh
bash $P3TMP/setup-probe-app.sh 8eb07ac
cd $P3TMP/probe-app && node ../probe-auth/auth-instance.mjs
```

#### 5. 检查项

每个场景的编号检查见输出；另外每个场景都检查"没有列出之外的请求"和"没有未捕获的页面错误"，场景中途失败时记为"ran to the end"失败。九个场景：登录失败、登录成功、注册失败、两个表单之间的错误链接、未知错误码（`?error_code=toString`）、关闭注册、实例请求失败、修改密码、退出。

#### 6. 在 `8eb07ac` 的构建上的输出

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
  (landed on /probe-ws/projects/)
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
  (landed on /sign-up/?email=nobody%40example.com)
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

#### 7. 没有覆盖的，以及原因

- **注册成功之后的新手引导不在其中**：`is_self_managed` 控制的两个步骤由 M2 决定去留（spec 2.11），P3 没有改它们。
- **后端的 CSRF 校验没有被执行**：桩只核对令牌是否随表单提交、是否在提交前取过；Django 的真实校验由 M2 连同会话一起替换。
- **只检查英文**：`zh-CN` 没有加载。
- **只在桌面宽度**（1440×900）。
- **工作区邀请的两条路径**不在其中：P3 没有改它们（M1 设计 3.15）。

### 6.3 临时核对脚本 2：动态、通知、工作项列表与详情（7.5 "P2、P3 动态、通知、工作项列表"）

#### 1. 目的

在 P2 的临时核对脚本 3（动态、通知、各上下文的工作项列表、四种布局）上加 P3 的几项：描述历史和按编号查找用 `/work-items/` 地址；任何布局都没有选择框；图片选择器只有"图片""上传"，不请求 `/api/unsplash/`；@提及只搜用户；收藏的五种实体都有图标。

#### 2. 脚本全文（`$P3TMP/probe-lists/lists-detail-probe.mjs`）

```js
// One-off (M1/P3, design 7.5 row "P2、P3 动态、通知、工作项列表", P3 spec 2.13; built on M1/P2's probe 3, whose checks
// 1-4 are unchanged apart from the ones marked P3): serves the built web app
// (web/apps/web/build/client, SPA fallback to index.html) on a free port, signs in a stubbed, onboarded user
// of workspace probe-ws / project p1, and checks in a real browser that
//   1. a work item's activity shows its state, relation, cycle, module and comment entries as readable text;
//   2. the notifications page lists the stubbed notifications, opening one shows the work item preview and
//      "mark all as read" posts the right request and clears the unread count and indicator;
//   3. the project, cycle, module, profile ("assigned") and archive pages each list the work items of THAT
//      context (every context has its own titles, so a wrong issue store shows the wrong titles or none);
//   4. the project work-item view offers exactly the list, board, table and calendar layouts and each renders;
// and, for P3:
//   1b. the description history uses /work-items/…/description-versions/ and the lookup by sequence uses
//       /workspaces/…/work-items/PRB-1/ (M1 design 2.2: these two keep the work-items address);
//   4b. no layout shows a selection checkbox (bulk operations and multi-select are gone, design 3.9);
//   5. the cover picker offers the preset images and upload only, and never requests /api/unsplash/;
//   6. an @mention in the comment editor searches users only (query_type=user_mention);
//   7. the sidebar favourites show every kept entity type (project, view, cycle, module, folder) with an icon.
// Every /api or /auth request the stubs do not answer is a failure (P3 spec 2.13).
// The stubs answer the Plane endpoints these pages call, with minimal data (no estimates, no pages, no epics);
// every other /api or /auth request gets 404 and is listed at the end. Run from the repository root
// (Playwright comes from e2e/). usage: node lists-detail-probe.mjs
import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import net from "node:net";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const ROOT = path.resolve("web/apps/web/build/client");

// ---------------------------------------------------------------- fake data
const W = "/api/workspaces/probe-ws";
const P = `${W}/projects/p1`;
const now = Date.now();
const iso = (t) => new Date(t).toISOString();
const localDate = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
// a weekday of the current month (the calendar hides weekends by default)
const dueDate = (() => {
  const d = new Date();
  const shift = { 0: 1, 6: -1 }[d.getDay()] ?? 0;
  const moved = new Date(d.getFullYear(), d.getMonth(), d.getDate() + shift);
  if (moved.getMonth() !== d.getMonth()) moved.setDate(moved.getDate() - 3 * Math.sign(shift));
  return localDate(moved);
})();
const users = {
  u1: { id: "u1", display_name: "probe", first_name: "Probe", last_name: "User", avatar_url: "", is_bot: false },
  u2: { id: "u2", display_name: "otto", first_name: "Otto", last_name: "Other", avatar_url: "", is_bot: false },
};
const states = [
  ["s-backlog", "Backlog", "backlog"],
  ["s-todo", "Todo", "unstarted"],
  ["s-progress", "In Progress", "started"],
  ["s-done", "Done", "completed"],
  ["s-cancel", "Cancelled", "cancelled"],
].map(([id, name, group], i) => ({
  id, name, group, color: "#888888", default: i === 1, description: "", project_id: "p1",
  sequence: (i + 1) * 1000, workspace_id: "w1", order: i,
}));
const project = {
  id: "p1", name: "Probe Project", identifier: "PRB", sort_order: 1, logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  member_role: 20, archived_at: null, workspace: "w1", cycle_view: true, issue_views_view: true, module_view: true,
  inbox_view: false, network: 2, created_at: "2026-09-01T00:00:00Z", members: ["u1", "u2"], is_member: true, anchor: null,
};
const counts = { total_issues: 2, completed_issues: 0, backlog_issues: 0, started_issues: 1, unstarted_issues: 1, cancelled_issues: 0 };
const cycle = {
  id: "c1", name: "Sprint One", description: "", project_id: "p1", workspace_id: "w1", owned_by_id: "u1", sort_order: 1,
  start_date: iso(now - 7 * 86400000), end_date: iso(now + 7 * 86400000), status: "current", archived_at: null,
  is_favorite: false, view_props: { filters: {} }, project_detail: { id: "p1" }, assignee_ids: [], ...counts,
};
const module = {
  id: "m1", name: "Module One", description: "", description_text: null, description_html: null, workspace_id: "w1",
  project_id: "p1", lead_id: null, member_ids: [], is_favorite: false, sort_order: 1, view_props: { filters: {} },
  status: "in-progress", archived_at: null, start_date: null, target_date: null,
  created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z", ...counts,
};
const issue = (id, seq, name, extra = {}) => ({
  id, sequence_id: seq, name, sort_order: seq * 1000, state_id: "s-todo", priority: "medium", label_ids: [],
  assignee_ids: [], sub_issues_count: 0, attachment_count: 0, link_count: 0, project_id: "p1", parent_id: null,
  cycle_id: null, module_ids: [], type_id: null, created_at: "2026-09-10T00:00:00Z", updated_at: "2026-09-10T00:00:00Z",
  start_date: null, target_date: dueDate, completed_at: null, archived_at: null, created_by: "u2", updated_by: "u2",
  is_draft: false, ...extra,
});
// the project's work items: alpha is in the cycle and the module, gamma only in the cycle, delta only in the
// module, epsilon is assigned to the signed-in user; zeta is archived. Alpha's description is stored the way the
// editor saves it (paragraph with a node id); without the id the editor's one-off id migration PATCHes it on view
const ISSUES = [
  issue("i1", 1, "Project item alpha", {
    state_id: "s-progress", cycle_id: "c1", module_ids: ["m1"],
    description_html: '<p class="editor-paragraph-block" data-id="5f1c2a4e-1111-4a2b-9c3d-000000000001">Alpha description text</p>',
  }),
  issue("i2", 2, "Cycle item gamma", { cycle_id: "c1" }),
  issue("i3", 3, "Module item delta", { module_ids: ["m1"] }),
  issue("i4", 4, "Assigned item epsilon", { assignee_ids: ["u1"] }),
];
const ARCHIVED = [issue("i5", 5, "Archived item zeta", { state_id: "s-done", archived_at: "2026-09-15T00:00:00Z" })];
const TITLES = [...ISSUES, ...ARCHIVED].map((i) => i.name);
// list endpoints answer like Plane: filtered by the context's query parameters, grouped by group_by/sub_group_by
const GROUP_PROP = {
  state_id: "state_id", priority: "priority", labels__id: "label_ids", assignees__id: "assignee_ids",
  cycle_id: "cycle_id", issue_module__module_id: "module_ids", target_date: "target_date",
  project_id: "project_id", created_by: "created_by",
};
const groupKeys = (i, g) => {
  const v = i[GROUP_PROP[g]];
  return Array.isArray(v) ? (v.length ? v : ["None"]) : [v ?? "None"];
};
const groupBy = (list, g) => {
  const out = {};
  for (const i of list) for (const k of groupKeys(i, g)) (out[k] ??= []).push(i);
  return out;
};
const wrap = (list, results = list) => ({ results, total_results: list.length });
const paginated = (list, q) => {
  const [g, sg] = [q.get("group_by"), q.get("sub_group_by")];
  const nest = (arr) =>
    sg ? Object.fromEntries(Object.entries(groupBy(arr, sg)).map(([k, sub]) => [k, wrap(sub)])) : arr;
  const results = g ? Object.fromEntries(Object.entries(groupBy(list, g)).map(([k, arr]) => [k, wrap(arr, nest(arr))])) : list;
  return {
    grouped_by: g, sub_grouped_by: sg, next_cursor: "100:1:0", prev_cursor: "100:-1:1", next_page_results: false,
    prev_page_results: false, total_count: list.length, count: list.length, total_pages: 1, extra_stats: null,
    results, total_results: list.length,
  };
};
const filtered = (q) =>
  ISSUES.filter(
    (i) =>
      (!q.get("cycle") || i.cycle_id === q.get("cycle")) &&
      (!q.get("module") || i.module_ids.includes(q.get("module"))) &&
      (!q.get("assignees") || i.assignee_ids.includes(q.get("assignees"))) &&
      (!q.get("created_by") || i.created_by === q.get("created_by")) &&
      (!q.get("priority") || q.get("priority").split(",").includes(i.priority)) &&
      !q.get("subscriber")
  );
const activity = (id, minutesAgo, extra) => ({
  id, workspace: "w1", workspace_detail: { id: "w1", slug: "probe-ws", name: "Probe WS" }, project: "p1",
  project_detail: { id: "p1", identifier: "PRB", name: "Probe Project" }, issue: "i1",
  issue_detail: { id: "i1", sequence_id: 1, name: "Project item alpha" }, actor: "u2", actor_detail: users.u2,
  created_at: iso(now - minutesAgo * 60000), updated_at: iso(now - minutesAgo * 60000), verb: "updated", field: null,
  old_value: "", new_value: "", old_identifier: null, new_identifier: null, comment: "", epoch: 0,
  issue_comment: null, attachments: [], ...extra,
});
const ACTIVITIES = [
  activity("a0", 60, { verb: "created" }),
  activity("a1", 50, { field: "state", old_value: "Todo", new_value: "In Progress", old_identifier: "s-todo", new_identifier: "s-progress" }),
  activity("a2", 40, { field: "blocking", new_value: "PRB-2", new_identifier: "i2" }),
  activity("a3", 30, { verb: "created", field: "cycles", new_value: "Sprint One", new_identifier: "c1" }),
  activity("a4", 20, { verb: "created", field: "modules", new_value: "Module One", new_identifier: "m1" }),
];
const COMMENTS = [
  {
    ...activity("cm1", 10, {}), comment_html: "<p>Probe comment on alpha</p>", comment_stripped: "Probe comment on alpha",
    comment_json: {}, comment_reactions: [], access: "INTERNAL", external_id: null, external_source: null,
  },
];
const notification = (id, item, issueActivity, minutesAgo) => ({
  id, title: item.name, entity_identifier: item.id, entity_name: "issue", message_html: "<p></p>",
  sender: "in_app:issue_activities:updated", receiver: "u1", triggered_by: "u2", triggered_by_details: users.u2,
  read_at: null, archived_at: null, snoozed_till: null, is_inbox_issue: false, is_mentioned_notification: false,
  workspace: "w1", project: "p1", created_at: iso(now - minutesAgo * 60000), updated_at: iso(now - minutesAgo * 60000),
  created_by: "u2", updated_by: "u2",
  data: {
    issue: { id: item.id, sequence_id: item.sequence_id, identifier: "PRB", name: item.name, state_name: "Todo", state_group: "unstarted" },
    issue_activity: { id: `na-${id}`, actor: "u2", issue_comment: null, ...issueActivity },
  },
});
const NOTIFICATIONS = [
  notification("n1", ISSUES[0], { field: "state", verb: "updated", old_value: "Todo", new_value: "In Progress" }, 5),
  notification("n2", ISSUES[1], { field: "comment", verb: "created", old_value: "", new_value: "<p>Looks good to me</p>" }, 3),
];
const filterProps = (layout) => ({ rich_filters: {}, display_filters: { layout }, display_properties: {} });
const emptyPage = { next_page_results: false, prev_page_results: false, results: [], total_pages: 0 };

// "METHOD path" keys answer that method only; bare path keys answer GET only; functions get (query, request)
const STUBS = {
  "/api/instances/": { instance: { is_setup_done: true }, config: { is_email_password_enabled: true } },
  "/api/users/me/": { id: "u1", email: "probe@example.com", display_name: "probe", first_name: "Probe", last_name: "User" },
  "/api/users/me/profile/": { id: "p1", user: "u1", language: "en", is_onboarded: true, theme: {} },
  "/api/users/me/settings/": { id: "u1", email: "probe@example.com", workspace: {} },
  "/api/users/me/workspaces/": [{ id: "w1", slug: "probe-ws", name: "Probe WS", role: 20 }],
  "/api/users/me/workspaces/probe-ws/project-roles/": { p1: 20 },
  [`${W}/workspace-members/me/`]: { id: "wm1", member: "u1", role: 20, workspace: "w1", view_props: {}, default_props: {}, draft_issue_count: 0 },
  [`${W}/members/`]: Object.values(users).map((member, i) => ({ id: `wm${i + 1}`, member, role: 20, is_active: true, created_at: "2026-09-01T00:00:00Z" })),
  [`${W}/projects/`]: [project],
  [`${W}/states/`]: states,
  [`${W}/labels/`]: [],
  [`${W}/cycles/`]: [cycle],
  [`${W}/modules/`]: [module],
  [`${W}/user-favorites/`]: [],
  [`${W}/user-properties/`]: { ...filterProps("list"), navigation_project_limit: 10, navigation_control_preference: "ACCORDION" },
  [`${W}/users/notifications/unread/`]: { total_unread_notifications_count: 2, mention_unread_notifications_count: 0 },
  [`${W}/users/notifications/`]: { ...emptyPage, next_cursor: "30:1:0", prev_cursor: "30:-1:1", count: 2, total_count: 2, total_pages: 1, results: NOTIFICATIONS },
  [`POST ${W}/users/notifications/n1/read/`]: { ...NOTIFICATIONS[0], read_at: iso(now) },
  [`POST ${W}/users/notifications/mark-all-read/`]: {},
  [`${W}/user-issues/u1/`]: (q) => paginated(filtered(q), q),
  [`${P}/`]: project,
  [`${P}/project-members/me/`]: { id: "pm1", member: "u1", role: 20, original_role: 20, created_at: "2026-09-01T00:00:00Z" },
  [`${P}/members/`]: Object.keys(users).map((member, i) => ({ id: `pm${i + 1}`, member, role: 20, original_role: 20, created_at: "2026-09-01T00:00:00Z" })),
  [`${P}/user-properties/`]: { ...filterProps("list"), sort_order: 1, preferences: { navigation: { default_tab: "work_items", hide_in_more_menu: [] } } },
  [`PATCH ${P}/user-properties/`]: (q, request) => request.postDataJSON(),
  [`${P}/issue-labels/`]: [],
  [`${P}/states/`]: states,
  [`${P}/intake-state/`]: states[0],
  [`${P}/views/`]: [],
  [`${P}/cycles/`]: [cycle],
  [`${P}/cycles/c1/`]: cycle,
  [`${P}/cycles/c1/user-properties/`]: filterProps("list"),
  [`${P}/cycles/c1/progress/`]: counts,
  [`${P}/cycles/c1/analytics/`]: { assignees: [], labels: [], completion_chart: {} },
  [`${P}/cycles/c1/cycle-issues/`]: (q) => paginated(filtered(q).filter((i) => i.cycle_id === "c1"), q),
  [`${P}/modules/`]: [module],
  [`${P}/modules/m1/`]: module,
  [`${P}/modules/m1/user-properties/`]: filterProps("list"),
  [`${P}/issues/`]: (q) => paginated(filtered(q), q),
  [`${P}/archived-issues/`]: (q) => paginated(ARCHIVED, q),
  [`${P}/issues/i1/`]: ISSUES[0],
  [`${P}/issues/i1/meta/`]: { project_identifier: "PRB", sequence_id: 1 },
  [`${W}/work-items/PRB-1/`]: ISSUES[0],
  [`${P}/issues/i1/history/`]: (q) => (q.get("activity_type") === "issue-comment" ? COMMENTS : ACTIVITIES),
  [`${P}/issues/i1/sub-issues/`]: { sub_issues: [], state_distribution: {} },
  [`${P}/issues/i1/issue-relation/`]: { blocking: [ISSUES[1]], blocked_by: [], duplicate: [], relates_to: [] },
  [`${P}/work-items/i1/description-versions/`]: { ...emptyPage, cursor: "", next_cursor: null, prev_cursor: null, page_count: 0 },
  // P3: the mention search answers users only
  [`${W}/entity-search/`]: (q) => ({ user_mention: q.get("query_type") === "user_mention" ? [users.u1, users.u2].map((u) => ({ member__id: u.id, member__display_name: u.display_name, member__avatar_url: "" })) : [] }),
};
// P3: one favourite of every kept entity type; the folder holds nothing
const favourite = (id, entity_type, name, extra = {}) => ({
  id, name, entity_type, entity_identifier: extra.entity_identifier ?? null, entity_data: extra.entity_data ?? null,
  is_folder: entity_type === "folder", sequence: 65535 * (Number(id.slice(1)) + 1), parent: null, workspace_id: "w1",
  project_id: extra.project_id ?? null,
});
const FAVOURITES = [
  favourite("f0", "project", "Probe Project", { entity_identifier: "p1", project_id: "p1", entity_data: { id: "p1", name: "Probe Project", logo_props: project.logo_props } }),
  favourite("f1", "view", "Probe View", { entity_identifier: "v1", project_id: "p1", entity_data: { id: "v1", name: "Probe View", logo_props: {} } }),
  favourite("f2", "cycle", "Sprint One", { entity_identifier: "c1", project_id: "p1", entity_data: { id: "c1", name: "Sprint One" } }),
  favourite("f3", "module", "Module One", { entity_identifier: "m1", project_id: "p1", entity_data: { id: "m1", name: "Module One" } }),
  favourite("f4", "folder", "Probe Folder"),
];

// ---------------------------------------------------------------- harness
const TYPES = { ".js": "text/javascript", ".css": "text/css", ".html": "text/html", ".json": "application/json", ".svg": "image/svg+xml", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".jpg": "image/jpeg", ".ico": "image/x-icon", ".woff2": "font/woff2" };
const server = http.createServer((req, res) => {
  let file = path.join(ROOT, decodeURIComponent(new URL(req.url, "http://x").pathname));
  if (!file.startsWith(ROOT + path.sep) || !fs.existsSync(file) || fs.statSync(file).isDirectory())
    file = path.join(ROOT, "index.html"); // React Router SPA mode emits index.html as the shell
  res.writeHead(200, { "content-type": TYPES[path.extname(file)] ?? "application/octet-stream" });
  fs.createReadStream(file).pipe(res);
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const port = server.address().port;
const base = `http://127.0.0.1:${port}`;

const unstubbed = [];
const consoleErrors = {};
let failures = 0;
const norm = (s) => s.replace(/\s+/g, " ").trim();
const fail = (detail) => {
  throw new Error(detail);
};
async function check(name, fn) {
  try {
    await fn();
    console.log(`PASS ${name}`);
  } catch (error) {
    failures += 1;
    console.log(`FAIL ${name}: ${error.message.split("\n")[0]}`);
  }
}

let browser;
let started = [];
// the browser and its helpers are descendants of this process: they are listed before the browser closes and
// checked at the end, so the run proves it leaves no process behind
const descendants = (pid) => {
  let children = [];
  try {
    children = execFileSync("pgrep", ["-P", String(pid)]).toString().split("\n").filter(Boolean).map(Number);
  } catch {} // pgrep exits 1 when there are none
  return children.flatMap((child) => [child, ...descendants(child)]);
};
const alive = (pid) => {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
};
async function openPage(scenario) {
  const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
  const page = await context.newPage();
  const requests = [];
  consoleErrors[scenario] ??= [];
  page.on("console", (m) => m.type() === "error" && consoleErrors[scenario].push(m.text().split("\n")[0]));
  page.on("pageerror", (e) => consoleErrors[scenario].push(`pageerror: ${String(e.message).split("\n")[0]}`));
  const handler = (route) => {
    const request = route.request();
    const url = new URL(request.url());
    requests.push({ method: request.method(), path: url.pathname, query: url.searchParams, body: request.postData() });
    let stub = STUBS[`${request.method()} ${url.pathname}`] ?? (request.method() === "GET" ? STUBS[url.pathname] : undefined);
    if (typeof stub === "function") stub = stub(url.searchParams, request);
    if (stub === undefined) {
      unstubbed.push(`[${scenario}] ${request.method()} ${url.pathname}${url.search}`);
      return route.fulfill({ status: 404, json: {} });
    }
    return route.fulfill({ json: stub });
  };
  await page.route("**/api/**", handler);
  await page.route("**/auth/**", handler);
  return { page, requests, close: () => context.close() };
}
const waitForRequest = (page, method, pathname, predicate = () => true) =>
  page.waitForRequest((r) => r.method() === method && new URL(r.url()).pathname === pathname && predicate(r), { timeout: 15000 });
const ERROR_PAGE = "Looks like something went wrong";

try {
  browser = await chromium.launch();

  // ---------------------------------------------------------- 1. activity
  {
    const { page, requests, close } = await openPage("activity");
    await page.goto(`${base}/probe-ws/projects/p1/issues/i1/`);
    let section;
    let entries = [];
    await check("activity: the work item opens from its project path (redirects to /browse/PRB-1/)", async () => {
      await page.waitForURL(/\/probe-ws\/browse\/PRB-1\/?$/, { timeout: 15000 });
      section = page.locator("div.text-h5-medium", { hasText: /^Activity$/ }).locator("xpath=../..");
      await section.getByText("Probe comment on alpha").waitFor({ timeout: 15000 });
      entries = (await section.locator("div.w-full.truncate.text-secondary").allInnerTexts()).map(norm);
    });
    await check("activity: 5 property entries (created, state, relation, cycle, module), each with actor and text", async () => {
      if (entries.length !== 5) fail(`got ${entries.length}: ${JSON.stringify(entries)}`);
      const blank = entries.filter((e) => !/^otto \S.{8,}/.test(e));
      if (blank.length) fail(`entries without text: ${JSON.stringify(blank)}`);
    });
    // each entry is "<actor> <text> <time ago>"; the cycle and module names also link to their pages
    const entry = (name, text, href) =>
      check(`activity: ${name} entry reads "${text}"${href ? ` and links to ${href}` : ""}`, async () => {
        if (!entries.some((e) => e.startsWith(`${text} `))) fail(`not among ${JSON.stringify(entries)}`);
        if (href && !(await section.locator(`a[href="${href}"]`).count())) fail(`no link to ${href}`);
      });
    await entry("state", "otto set the state to In Progress.");
    await entry("relation", "otto marked this work item is blocking work item PRB-2.");
    await entry("cycle", "otto added this work item to the cycle Sprint One", "/probe-ws/projects/p1/cycles/c1");
    await entry("module", "otto added this work item to the module Module One", "/probe-ws/projects/p1/modules/m1");
    await check("activity: comment entry shows its author and text", async () => {
      const text = norm(await section.locator('div[id="cm1"]').first().innerText());
      if (!/^O otto commented .* Probe comment on alpha$/.test(text)) fail(JSON.stringify(text));
    });
    await check("activity: no raw i18n key, undefined, null or NaN in the activity section", async () => {
      const text = norm(await section.innerText());
      const bad = text.match(/\b(undefined|null|NaN)\b|\b[a-z][a-z0-9_]*(\.[a-z0-9_]+)+\b/g);
      if (bad) fail(`${JSON.stringify(bad)} in ${JSON.stringify(text)}`);
    });
    await check("activity (P3): the lookup by sequence uses /work-items/PRB-1/, the description history /work-items/i1/description-versions/", async () => {
      const paths = requests.map((r) => r.path);
      const want = [`${W}/work-items/PRB-1/`, `${P}/work-items/i1/description-versions/`];
      const missing = want.filter((p) => !paths.includes(p));
      if (missing.length) fail(`not requested: ${JSON.stringify(missing)}`);
      const wrong = paths.filter((p) => /\/issues\/(PRB-1|i1\/description-versions)\/$/.test(p));
      if (wrong.length) fail(`issues-addressed instead: ${JSON.stringify(wrong)}`);
    });
    // 6. an @mention in the comment editor searches users only
    await check("mentions (P3): typing @ in the comment editor searches users only and lists them", async () => {
      const editor = section.locator(".ProseMirror[contenteditable=true]").last();
      await editor.click();
      const search = waitForRequest(page, "GET", `${W}/entity-search/`);
      await page.keyboard.type("@ott");
      const request = await search;
      const types = new URL(request.url()).searchParams.get("query_type");
      if (types !== "user_mention") fail(`query_type=${types}`);
      await page.getByText("otto", { exact: true }).last().waitFor({ timeout: 10000 });
    });
    await close();
  }

  // ---------------------------------------------------------- 2. notifications
  {
    const { page, close } = await openPage("notifications");
    const inboxDot = page.locator('a[href="/probe-ws/notifications/"] span.bg-danger-primary');
    const allTab = page.locator("div.cursor-pointer", { hasText: /^All/ }).first();
    const unreadDots = page.locator("div.border-b.border-subtle div.absolute.rounded-full.bg-accent-primary");
    const card = (identifier, title) => page.getByText(new RegExp(`^${identifier}\\s${title}$`));
    await page.goto(`${base}/probe-ws/notifications/`);
    await check("notifications: the page lists both stubbed notifications with readable text", async () => {
      await card("PRB-1", "Project item alpha").waitFor({ timeout: 15000 });
      await card("PRB-2", "Cycle item gamma").waitFor();
      const text = norm(await page.locator("body").innerText());
      for (const s of ["otto updated state to In Progress.", "otto commented Looks good to me."])
        if (!text.includes(s)) fail(`missing "${s}"`);
    });
    await check("notifications: before reading, the inbox shows the unread dot and All shows 2", async () => {
      await inboxDot.waitFor({ timeout: 10000 });
      if (norm(await allTab.innerText()) !== "All 2") fail(`All tab reads ${JSON.stringify(await allTab.innerText())}`);
      if ((await unreadDots.count()) !== 2) fail(`${await unreadDots.count()} unread card dots`);
    });
    const markRead = waitForRequest(page, "POST", `${W}/users/notifications/n1/read/`);
    await card("PRB-1", "Project item alpha").click();
    await check("notifications: opening one sends its read request and All drops to 1", async () => {
      await markRead;
      await page.waitForFunction(() => {
        const tab = [...document.querySelectorAll("div.cursor-pointer")].find((d) => /^All/.test(d.innerText));
        return tab && tab.innerText.replace(/\s+/g, " ").trim() === "All 1";
      }, null, { timeout: 10000 });
    });
    await check("notifications: opening one shows the work item preview (title and description)", async () => {
      await page.waitForFunction(() => document.querySelector("#title-input")?.value === "Project item alpha", null, { timeout: 15000 });
      await page.getByText("Alpha description text").waitFor({ timeout: 15000 });
    });
    await check('notifications: "mark all as read" posts {snoozed:false, archived:false}', async () => {
      const button = page.locator("div.h-header", { hasText: "Inbox" }).locator("button").first();
      await button.hover();
      await page.getByText("Mark all as read", { exact: true }).waitFor({ timeout: 5000 });
      const request = waitForRequest(page, "POST", `${W}/users/notifications/mark-all-read/`);
      await button.click();
      const body = (await request).postDataJSON();
      if (JSON.stringify(body) !== JSON.stringify({ snoozed: false, archived: false })) fail(`body ${JSON.stringify(body)}`);
    });
    await check("notifications: after mark all read, the inbox dot, the All count and the card dots are gone", async () => {
      await inboxDot.waitFor({ state: "detached", timeout: 10000 });
      if (norm(await allTab.innerText()) !== "All") fail(`All tab reads ${JSON.stringify(await allTab.innerText())}`);
      if (await unreadDots.count()) fail(`${await unreadDots.count()} unread card dots left`);
    });
    await close();
  }

  // ---------------------------------------------------------- 3. work-item store per context
  const CONTEXTS = [
    { name: "project", url: "/probe-ws/projects/p1/issues/", titles: [0, 1, 2, 3],
      request: (r) => r.path === `${P}/issues/` && !r.query.get("cycle") && !r.query.get("module") },
    { name: "cycle", url: "/probe-ws/projects/p1/cycles/c1/", titles: [0, 1],
      request: (r) => r.path === `${P}/issues/` && r.query.get("cycle") === "c1" },
    { name: "module", url: "/probe-ws/projects/p1/modules/m1/", titles: [0, 2],
      request: (r) => r.path === `${P}/issues/` && r.query.get("module") === "m1" },
    { name: "profile", url: "/probe-ws/profile/u1/", finalUrl: /\/probe-ws\/profile\/u1\/assigned\/?$/, titles: [3],
      request: (r) => r.path === `${W}/user-issues/u1/` && r.query.get("assignees") === "u1" },
    { name: "archives", url: "/probe-ws/projects/p1/archives/issues/", titles: [4],
      request: (r) => r.path === `${P}/archived-issues/` },
  ];
  for (const ctx of CONTEXTS) {
    const { page, requests, close } = await openPage(`store-${ctx.name}`);
    const expected = ctx.titles.map((i) => TITLES[i]);
    await check(`store: ${ctx.name} page lists exactly ${JSON.stringify(expected)}`, async () => {
      await page.goto(`${base}${ctx.url}`);
      if (ctx.finalUrl) await page.waitForURL(ctx.finalUrl, { timeout: 15000 });
      await page.getByText(expected[0], { exact: true }).first().waitFor({ timeout: 15000 });
      await page.waitForTimeout(1000);
      const shown = [];
      for (const title of TITLES)
        if (await page.getByText(title, { exact: true }).filter({ visible: true }).count()) shown.push(title);
      if (JSON.stringify(shown) !== JSON.stringify(expected)) fail(`shown ${JSON.stringify(shown)}`);
      const listCalls = requests.filter((r) => /\/(issues|archived-issues|user-issues\/u1)\/$/.test(r.path));
      if (!listCalls.some(ctx.request))
        fail(`no matching list request; list requests: ${JSON.stringify(listCalls.map((r) => `${r.path}?${r.query}`))}`);
    });
    await close();
  }

  // ---------------------------------------------------------- 4. layouts
  {
    const { page, close } = await openPage("layouts");
    await page.goto(`${base}/probe-ws/projects/p1/issues/`);
    await page.getByText(TITLES[0], { exact: true }).first().waitFor({ timeout: 15000 });
    const layoutButtons = page.locator('button[aria-label$=" Layout"]');
    await check("layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)", async () => {
      const labels = (await layoutButtons.evaluateAll((els) => els.map((e) => e.getAttribute("aria-label")))).sort();
      const want = ["Board Layout", "Calendar Layout", "List Layout", "Table Layout"];
      if (JSON.stringify(labels) !== JSON.stringify(want)) fail(`labels ${JSON.stringify(labels)}`);
      const html = await page.content();
      if (/gantt|timeline/i.test(html)) fail(`"gantt" or "timeline" in the page: ${html.match(/.{40}(gantt|timeline).{40}/i)?.[0]}`);
    });
    const LAYOUTS = [
      ["Board Layout", "kanban", '[id="s-progress__null"] [id="issue-i1"]', '[id="s-todo__null"] [id="issue-i2"]'],
      ["Table Layout", "spreadsheet", 'table td[id="issue-i1"]', 'table td[id="issue-i4"]'],
      ["Calendar Layout", "calendar", 'a[id="issue-i1"]', 'div.grid > div:text-is("Mon")'],
      ["List Layout", "list", 'a[id="issue-i1"]', 'a[id="issue-i4"]'],
    ];
    for (const [label, key, ...markers] of LAYOUTS) {
      await check(`layouts: switching to ${label} saves layout=${key} and renders it without an error`, async () => {
        const errorsBefore = consoleErrors.layouts.filter((e) => e.startsWith("pageerror")).length;
        const saved = waitForRequest(page, "PATCH", `${P}/user-properties/`, (r) => r.postDataJSON()?.display_filters?.layout === key);
        await page.locator(`button[aria-label="${label}"]`).click();
        await saved;
        for (const marker of markers) await page.locator(marker).first().waitFor({ timeout: 15000 });
        if (key !== "spreadsheet" && (await page.locator("table td[id^='issue-']").count())) fail("table rows still shown");
        if ((await page.locator("body").innerText()).includes(ERROR_PAGE)) fail("error page shown");
        // P3: bulk operations and multi-select are gone, so no layout offers a selection checkbox
        const boxes = await page.locator("main input[type=checkbox], main [role=checkbox]").count();
        if (boxes) fail(`${boxes} selection checkbox(es) in the ${key} layout`);
        const errorsAfter = consoleErrors.layouts.filter((e) => e.startsWith("pageerror"));
        if (errorsAfter.length > errorsBefore) fail(`page error: ${errorsAfter.at(-1)}`);
      });
    }
    await close();
  }

  // ---------------------------------------------------------- 5. cover picker (P3)
  {
    const { page, requests, close } = await openPage("cover-picker");
    await page.goto(`${base}/settings/profile/general`);
    await check("cover picker (P3): \"Change cover\" opens Images and Upload only, and nothing asks /api/unsplash/", async () => {
      const button = page.getByRole("button", { name: "Change cover" });
      await button.waitFor({ timeout: 15000 });
      await button.click();
      const tabs = page.getByRole("tab");
      await tabs.first().waitFor({ timeout: 10000 });
      const names = (await tabs.allInnerTexts()).map(norm);
      if (JSON.stringify(names) !== JSON.stringify(["Images", "Upload"])) fail(`tabs ${JSON.stringify(names)}`);
      await page.waitForTimeout(1000);
      const unsplash = requests.filter((r) => /unsplash/i.test(r.path));
      if (unsplash.length) fail(`requested ${JSON.stringify(unsplash.map((r) => r.path))}`);
      if (/unsplash/i.test(await page.locator("body").innerText())) fail("the word Unsplash is on the page");
    });
    await close();
  }

  // ---------------------------------------------------------- 7. favourites (P3)
  {
    const { page, close } = await openPage("favourites");
    STUBS[`${W}/user-favorites/`] = FAVOURITES;
    STUBS[`${W}/user-favorites/f4/group/`] = []; // the folder's contents
    await page.goto(`${base}/probe-ws/projects/p1/issues/`);
    await check("favourites (P3): the sidebar lists a project, view, cycle, module and folder favourite, each with an icon", async () => {
      const sidebar = page.locator("#main-sidebar");
      const toggle = sidebar.getByText("Favorites", { exact: true }).first();
      await toggle.waitFor({ timeout: 15000 });
      await toggle.click();
      await sidebar.getByText("Probe Folder", { exact: true }).first().waitFor({ timeout: 15000 });
      const missing = [];
      for (const name of FAVOURITES.map((f) => f.name)) {
        const item = sidebar.getByText(name, { exact: true }).last();
        if (!(await item.count())) {
          missing.push(`${name}: not shown`);
          continue;
        }
        const row = item.locator("xpath=ancestor::*[.//svg or .//img or .//span[contains(@class,'emoji')]][1]");
        if (!(await row.count())) missing.push(`${name}: no icon`);
      }
      if (missing.length) fail(missing.join("; "));
    });
    STUBS[`${W}/user-favorites/`] = [];
    await close();
  }
} finally {
  started = descendants(process.pid);
  await browser?.close();
  await new Promise((resolve) => server.close(resolve));
}

console.log(`\nunstubbed requests (${unstubbed.length}):`);
for (const r of [...new Set(unstubbed)]) console.log(`  ${r}`);
console.log("\nbrowser console errors per scenario:");
for (const [scenario, errors] of Object.entries(consoleErrors)) {
  const expected = unstubbed.length ? errors.filter((e) => /404|Failed to load resource/.test(e)) : [];
  const other = errors.filter((e) => !expected.includes(e));
  console.log(`  ${scenario}: ${errors.length} (from unstubbed requests: ${expected.length}, other: ${other.length})`);
  for (const e of [...new Set(other)]) console.log(`    other: ${e.slice(0, 200)}`);
}
const probe = net.createServer();
await new Promise((resolve, reject) => probe.once("error", reject).listen(port, "127.0.0.1", resolve));
await new Promise((resolve) => probe.close(resolve));
console.log(`port ${port} is free`);
await new Promise((resolve) => setTimeout(resolve, 500));
const left = started.filter(alive);
console.log(`browser processes started by this run: ${started.length}; still running: ${left.length ? left.join(", ") : "none"}`);
if (left.length) failures += 1;
if (unstubbed.length) failures += 1; // P3 spec 2.13: a request the probe does not list is a failure
console.log(`\n${failures ? `${failures} check(s) FAILED` : "all checks passed"}`);
process.exit(failures ? 1 : 0);
```

#### 3. 假数据

写在脚本开头（`fake data` 一段）：

- 工作区 `probe-ws`、项目 `p1`（`PRB`）、用户 `u1`（probe，已登录）和 `u2`（otto）、五个状态组各一个状态、迭代 `c1`（Sprint One）、模块 `m1`（Module One）；
- 工作项的标题各不相同，项目、迭代、模块、"分配给他的"、归档五个上下文各列出不同的组合，所以 store 选错时显示的标题就不对；
- 工作项 `i1` 的动态：创建、状态、关联、迭代、模块各一条，一条评论；没有估算、文档页、Epic、工时、类型类的动态；
- 两条通知；
- 实体搜索只在 `query_type=user_mention` 时返回两个用户；
- 收藏：项目、视图、迭代、模块各一个，加一个文件夹（打开后由 `user-favorites/f4/group/` 返回）。

#### 4. 运行命令

```sh
bash $P3TMP/setup-probe-app.sh 8eb07ac
cd $P3TMP/probe-app && node ../probe-lists/lists-detail-probe.mjs
```

#### 5. 检查项

见输出。标 `(P3)` 的是本 Phase 加的；"没有选择框"并在四个布局的检查里（`main` 里没有 `input[type=checkbox]` 或 `[role=checkbox]`）。

#### 6. 在 `8eb07ac` 的构建上的输出

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
port 49345 is free
browser processes started by this run: 3; still running: none

all checks passed
```

#### 7. 没有覆盖的，以及原因

- **封面的上传没有真的执行**：只核对选择器的两个标签页和没有 Unsplash 请求；上传走 M5 要重写的资源接口。
- **收藏只核对显示和图标**，没有点击跳转：链接映射按联合类型定类型，编译器保证每种都有（Task 7）。
- **只检查英文**，桌面宽度（1600×1000）。
- P2 脚本 3 原有的"没有覆盖"各项仍然成立（见 [P2 评审记录](P2-trim-content-review.md)附录）。

### 6.4 反向对照：基点 `6d9692b` 的构建

为了证明两段脚本能发现 P3 要改变的行为，控制者在 Task 7 期间，在基点 `6d9692b` 的构建上各跑了一次。

认证脚本在 29 项中有 12 项失败（场景中途失败时后面的项不再运行），正好是 P3 改变的行为：基点是先填邮箱的"检查邮箱"步骤而不是一个表单（所以后面的场景填不了密码，中途失败）；页头标题写反；注册页头的"登录"指向 `/sign-in`；错误提示的链接；个人设置多一个"通知"标签页；退出后落在"检查邮箱"步骤。

```

== sign-in fails
FAIL 1.1 / shows the sign-in form: email, password, no confirmation, "Go to workspace": New to Plane?
Sign up
No authentication methods available
Please contact your administrator to enable authentication for your instance.
Join 10,000+ teams building with Plane
FAIL 1.1b the document title names the sign-in page: Sign up - Plane
PASS 1.2 no sign-in code, forgot-password, third-party or admin entry on the page
PASS 1.3 sign-up is enabled, so the header links to /sign-up
FAIL sign-in fails: ran to the end: locator.fill: Timeout 30000ms exceeded.

== sign-in succeeds
FAIL sign-in succeeds: ran to the end: locator.fill: Timeout 30000ms exceeded.

== sign-up fails
FAIL 3.1 /sign-up shows email, password, confirmation and "Create account": Already have an account?
Sign in
No authentication methods available
Please contact your administrator to enable authentication for your instance.
Join 10,000+ teams building with Plane
FAIL 3.2 the header links straight to the sign-in page (/), titled "Sign up": /sign-in/ / Sign in - Plane
FAIL sign-up fails: ran to the end: locator.fill: Timeout 30000ms exceeded.

== error links between the forms
FAIL 4.1 "No account found" links to the sign-up page with the email: none
FAIL error links between the forms: ran to the end: locator.click: Timeout 30000ms exceeded.

== unknown error code
FAIL 5.1 /?error_code=toString shows the sign-in form without an error banner: New to Plane?
Sign up
No authentication methods available
Please contact your administrator to enable authentication for your instance.
Join 10,000+ teams building with Plane
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
FAIL 8.1 the profile settings sidebar offers exactly Profile, Security, Preferences, Personal Access Tokens: ["Profile","Preferences","Notifications","Security","Personal Access Tokens"]
PASS 8.2 the security page has old, new and confirmation password fields
PASS 8.3 the change posts old and new password with the X-CSRFToken header and reports success
PASS 8.4 no request to a deleted account endpoint (set-password, notification preferences, email change)
PASS password change: no request outside the scenario's list
PASS password change: no uncaught page error

== sign-out
  (landed on /)
FAIL 9.1 sign-out posts the CSRF token as a page navigation and lands on the sign-in form: {"csrfmiddlewaretoken":"probe-csrf-token"}
PASS sign-out: no request outside the scenario's list
PASS sign-out: no uncaught page error

17 passed, 12 failed
```

探测 2 只有图片选择器一项失败：基点每次打开都请求 `/api/unsplash/?query=`。其余 P3 的检查（`/work-items/` 地址、没有选择框、@提及、收藏图标）在基点也通过：它们是 P3 必须保持不变的行为，而 P3 确实没有改变它们。

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
FAIL cover picker (P3): "Change cover" opens Images and Upload only, and nothing asks /api/unsplash/: requested ["/api/unsplash/"]
PASS favourites (P3): the sidebar lists a project, view, cycle, module and folder favourite, each with an icon

unstubbed requests (1):
  [cover-picker] GET /api/unsplash/?query=

browser console errors per scenario:
  activity: 0 (from unstubbed requests: 0, other: 0)
  notifications: 0 (from unstubbed requests: 0, other: 0)
  store-project: 0 (from unstubbed requests: 0, other: 0)
  store-cycle: 0 (from unstubbed requests: 0, other: 0)
  store-module: 0 (from unstubbed requests: 0, other: 0)
  store-profile: 0 (from unstubbed requests: 0, other: 0)
  store-archives: 0 (from unstubbed requests: 0, other: 0)
  layouts: 0 (from unstubbed requests: 0, other: 0)
  cover-picker: 1 (from unstubbed requests: 1, other: 0)
  favourites: 0 (from unstubbed requests: 0, other: 0)
port 53590 is free
browser processes started by this run: 3; still running: none

2 check(s) FAILED
```

## 7. 交接与延后项

交给后续 M 的事项写进对应 M 的 `handoffs/M1-P3-trim-platform.md`：

| M | 事项 |
|---|---|
| M2 | - **必须删掉 Cookie 会话和 CSRF**：登录、注册、退出仍提交 `csrfmiddlewaretoken`，修改密码仍带 `X-CSRFTOKEN`；<br>- 认证错误就地显示：现在注册失败回到 `/`，显示在登录表单上方；<br>- 修改登录邮箱的最终形态（M1 设计 3.13，决策点）；<br>- 实例配置剩 4 个字段，`is_self_managed` 连同新手引导的两个步骤由 M2 决定，`enable_signup` 的名称由 M2 定；<br>- 不再读的用户字段、不再调用的认证和账户地址；<br>- API 令牌的 `retrieve`、`destroy` 地址没有结尾斜杠 |
| M3 | - 不再读 `IProject.anchor`、视图的发布设置、收集箱的 `anchors` / `is_form_enabled`；项目动态不再显示企业版功能开关的变化；<br>- 项目成员只有"从工作区成员中添加"一种方式；加入公开项目仍调用 `/projects/invitations/`（守卫例外 `until: M3`）；<br>- `RESTRICTED_URLS` 剩下的 Plane 产品词由 M3 与后端的保留名单一起定；<br>- `checkProjectIdentifierAvailability` 的地址没有结尾斜杠 |
| M4 | - 工作项不再读 `is_epic`、`type_id`，动态没有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`，评论没有 `access`；<br>- 描述历史和按编号查找仍用 `/work-items/`，其余是 `/issues/`；<br>- 没有批量操作：批量删除和批量归档的接口不再调用，命令面板的批量删除弹窗在基点就打不开，已删除；<br>- 富文本筛选的显示层和 `useFiltersOperatorConfigs`（恒定值）由 M4 决定去留；<br>- 收集箱的来源：前端只创建 `IN_APP`，`FORMS`、`EMAIL` 只在显示时读到，由 M4 决定保留哪些 |
| M5 | - `EFileAssetType` 剩 8 种；评论的附件总是项目级上传；封面图没有 Unsplash；<br>- 编辑器没有附件节点，但仍把 PDF 等文件的拖入和粘贴算作"已处理"、什么也不插入（与基点相同），由 M5 决定；<br>- `file_size_limit` 的名称和策略 |
| M6 | - 迭代进度地址 `…/analytics?type=` 没有结尾斜杠，守卫例外 `until: M6` 随进度接口的替换一起消失；<br>- `TCycleProgress`、`IWorkspaceProgressResponse` 已删除 |
| M7 | - 收藏的 `entity_type` 只有 `project`、`view`、`cycle`、`module`、`folder`；<br>- 通知卡片不再显示工时，通知内容的兜底映射已并入卡片 |

交给 M1 后续 Phase 和收尾：

| 去向 | 事项 |
|---|---|
| P4 | - `AuthenticationWrapper` 的 6 处渲染时跳转（M1 设计 4.2）；<br>- `routes/core.ts` 末尾的 10 条旧地址重定向。其中 `/sign-in` 的重定向会丢掉查询参数（Task 1 的探测发现），P3 让页内链接直接指向 `/` 和 `/sign-up`，不再经过它；P4 替换重定向时要保留查询参数；<br>- `next/navigation`、`next/link` 等兼容层（认证页也在用 `useSearchParams`） |
| P5 | - 61 个文件里的 `// plane web imports`、`// plane web components` 等导入分组注释（Plane 拆分 CE / EE 时的目录名）。这是 P3 结束时的实测：spec 按原型写的是 63 个，本次执行到 Task 7 结束时是 62 个，修复轮又改掉了 `project-feature-update.tsx` 里的；<br>- 页面标题的" - Plane"、注册页的"Create your Plane account"、"Join 10,000+ teams building with Plane"等品牌文字 |
| M1 收尾 | **死资源**（严格方法，P3 结束时）：498 个无引用的键、136 张无引用的图片、347 个没人用的包导出（`symref.mjs unused web/packages/`）。<br>**键工具的盲区**，删键之前先处理：<br>- 只有后缀的拼接看不见，P3 已把项目功能说明改为字面量，全仓库另有一处 `` t(`${getDescriptionPlaceholderI18n(…)}`) ``，它拼出的键都存在，外层的模板字符串多余；<br>- 键名与无关的字符串字面量相同时会被算作有引用（例如图标名 `"publish"`），这类"有引用"的键要人工看一遍。<br>**knip 看不见的死成员和死 prop**：整分支评审的启发式脚本列出 182 个从不使用的对象成员（其中 56 个是 store 或 service 的方法）和 205 个没人传的 prop，多数在基点就是死的；脚本在 `$P3TMP/final-review/`（`memberdead.mjs`、`propdead.mjs`，备份在 `p3tmp.tar`）。已知的几个：`TrailingNode` 的导出、`useProjectIssueProperties` 的 `fetchStates`、`IssueFormRoot` 挂载时的数据重置（可能是空操作，P3 为保证行为不变而保留）。<br>**依赖复核**：`prosemirror-codemark` 发布的 sourcemap 缺源文件，编辑器测试输出里有 5 行警告；查新版本是否带 `sourcesContent`，或者决定替换它（第 4 节裁定 6）。<br>**`M9` 例外**：重新核对两条（`tlds.ts` 的 `analytics`、`wiki`）。<br>**构建体积**：写进收尾评审，对比数据见第 1 节 |

## 8. spec 第 3 节的裁定

第 3 节的 15 项在执行前全部采纳（控制者评审 `c333971`），执行中的落实情况：

1. **第 6 步拆成两个 Task**：照此执行。Task 6 改扩展点和项目成员，Task 7 删死代码并把 knip 改为门禁；knip 门禁最后加入，之后没有 Task 再造出孤儿。
2. **注册失败后显示在登录页**：照此执行，认证探测第 3.5 项断言这一行为（交 M2）。执行中另外发现错误提示的两个链接还按被删的"检查邮箱"步骤写，已修正（第 2 节 Task 1、F1）。
3. **工作项的 service 类型整个删除**：照此执行。`/work-items/` 的两个地址写成字面量，探测 2 的"activity (P3)"一项断言它们。
4. **基点就已死、带着本 Task 词汇的代码由本 Task 删**：照此执行（8 处）。执行中把同样的思路推广到文案和图片（第 4 节裁定 4）。
5. **`useEditorFlagging` 之外的扩展点也在 P3 删**：照此执行，`enterprise-shells` 规则按确切符号列出，Task 6 的 29 行扩展点表证明它们在社区版中为空。
6. **收藏的实体类型改为联合类型**：照此执行，Task 7 又让 `FAVORITE_ITEM_LINKS` 以它为键。探测 2 断言五种收藏都有图标（交 M7）。
7. **表格菜单的"适应宽度"删除**：照此执行（Task 7）。
8. **`is_self_managed` 保留到 M2**：照此执行（交 M2）。
9. **基线就没人用的包导出交给收尾**：照此执行。执行中基线从 355 变为 349（第 4 节裁定 3）。
10. **类型包改用 React 库的 tsconfig**：照此执行，6 个文件改为 `import type` 导入 React 类型。
11. **富文本筛选的显示层留给 M4**：照此执行，`useFiltersOperatorConfigs` 一并交 M4。
12. **四个 store 文件改名**：照此执行，`git mv` 与类名一致，导入方在同一个提交里改完。
13. **`links/root.tsx` 改名为 `links/types.ts`**：照此执行。
14. **不删依赖**：照此执行。整个 Phase knip 没有报告未使用的依赖，锁文件没有变化。
15. **CSRF 不动**：照此执行，认证探测断言登录、注册、修改密码、退出都带 CSRF 令牌（交 M2）。
