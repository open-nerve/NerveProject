---
status: open
from: M2/closeout
to: M4
created: 2026-09-28
---

# M2 交给 M4：命令行的只投递 River 客户端、游标与请求体检查、清理任务、编辑器和下拉框

M2（账户认证）做完了注册、登录、会话、PAT、`nerve users` 命令和第一个 River 定时任务（[M2 设计](../../M2-auth/M2-design.md)）。下面这些是 M2 定下的约定第一次落到工作项上的地方，或 M2 改到而要由 M4 接上的代码。每一节来自 M2 设计 13.2 中接收者含 M4 的一行（多个 M 的行只取 M4 的部分），另有 M0-P6 交接剩下的一项（第 13 节）和收尾在 13.2 之外找到的几项：第 2 节的游标不签名，第 4 节的生成器遇到不支持的写法（M2 设计 §16）、第一个 `date` 字段和 map 型对象，第 6 节的定时任务，第 8 节的 `worker-src`、`frame-src`。

## 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）

- **负责人的要求**：2026-09-26 批准把它推迟到 M4 时说，"做好记录，以后别漏了"。M4 用不到它，也要把这一节原样转进下一个 M 的交接，不能随本交接关闭而丢掉。
- **做什么**：第一个"用例会投递 River 任务、又能从命令行触发"的命令加入时，`platform/jobs` 提供一个只投递的客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`），`bootstrap` 的命令组合把它交给这些用例。
- **为什么**：命令行的组合只有连接池和 `identity` 的管理用例，没有 River 客户端（`server/internal/bootstrap/users.go:22-26`，M2 设计 3.17）。服务用的客户端（`server/internal/platform/jobs/jobs.go:70` 的 `New`）配队列、启动 worker，命令行不能用它。用例在事务里用 `InsertTx` 投递（M2 设计 3.15）；命令行上没有客户端时，任务要么没投递，要么用例在运行中失败。M2 的命令都不投递任务，所以推迟（M2 设计 3.15、3.17、13.2）。
- **可能早于 M4**：M3 给停用加端口（[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 6 节），`nerve users deactivate` 也走这个用例；端口的实现投递任务时，这一项提前到 M3。
- **建议的守法**：投递任务的端口在用例的构造参数里必填，命令的组合漏了它，在构造时就失败，而不是运行到一半；`server/internal/bootstrap/users_test.go` 这样的组合测试覆盖每个命令。
- **关闭条件**：M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。

## 2. 列表的游标

- **做什么**：工作项按 `sort_order`、优先级或日期排序，每种排序定义自己的游标载荷，最后以 `id` 保证稳定（M2 设计 3.12）；不复用 PAT 列表的 `(created_at, id)`。
- **代码在哪**：游标的封套在 `server/internal/shared/cursor.go`（`EncodeCursor`、`DecodeCursor`：版本、base64url 编码，解不开是 400，总体设计 3.4）；PAT 列表（`identity` 的 `listApiTokens`）是载荷的范例。
- **游标不签名**：改成另一个合格的位置照样可用。载荷只决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：每种排序有自己的载荷和测试（翻页不重复、不遗漏，排序键相同时按 `id`）；改过的游标只能在调用者看得到的数据里换起点，有测试。

## 3. 自动归档读 `updated_at`

- **做什么**：自动归档按"超过 `archive_in × 30` 天未更新"判断（总体设计 5.5），读 `updated_at`。它由用例的时钟显式写入（M2 设计 3.13，差异清单二·全局），测试用固定时钟（`server/internal/platform/clock/clocktest`）。
- **关闭条件**：写工作项的每个用例都用时钟写 `updated_at`，测试断言它等于固定时钟；自动归档的测试推进固定时钟，不等真实的时间。

## 4. 请求体检查的两处延伸和草稿发布

- **路径的排序**：字段错误的路径现在按字典序排序（`tags[10]` 在 `tags[2]` 之前，`server/internal/platform/httpserver/bodyshape/bodyshape.go` 的 `Check` 末尾的 `slices.SortFunc`），改为按数组下标的数值排序。
- **数的范围**：不限类型的节点（`{}`、开放对象、没有 `items` 的数组）不看数的范围，`1e400` 这类 float64 放不下的数仍然得到解码器笼统的 400，改为在边界上报出。
- 两者都在第一个带数组或开放对象请求体的操作到来时处理（M2/P1 评审）；在那之前客户端不能依赖路径的顺序。
- **生成器**（`server/tools/bodyshapegen`）遇到不支持的 schema 写法时失败并说明原因；遇到它的 M 带着测试扩展生成器，不放过（M2 设计 §16）。
- **map 型对象**：契约的写法检查要求每个 object schema 设 `additionalProperties: false`（`server/internal/platform/httpserver/apitest/rules_test.go` 的 `closedObject`；codex-fixes 起组件和内联的都查，它是 M2 设计 3.11 不变式②的前提），`type: object, additionalProperties: {type: string}` 这样的 map 型 schema 会被判为违规（M0/P3 评审 C3，M0-P3 交接第 5 节；M2 没有用到）。第一个需要它的 M（可能早于 M4）给 `closedObject` 加一个例外分支和一个反例用例，并在 3.11 的不变式②写明解码器怎样读 map 的键。
- **结构检查的代价**（M2 设计 3.11，codex-fixes）：路径按段压栈、一个请求体最多报 16 个问题，`bodyshape` 的 `TestCheckCostsAboutTheBody` 守住。按表检查每层把对象的成员复制一次，代价是请求体乘 schema 的层数；M2 的请求体 schema 最深两层、不递归。第一个递归的请求体 schema 让它变成请求体乘请求体的深度：那时按表检查改为一遍读完，代价测试加上这种请求体。第一个数组或开放 map 的请求体 schema 到来时，代价测试同样加上它。
- **第一个 `format: date` 的字段**（工作项的开始、截止日期）：`bodyshape` 的格式检查器按生成的 Go 类型登记，现在只有 `time.Time` 和标准库的 `uuid.UUID`；M2 没有 `date` 字段，生成器遇到 `date` 就失败。M4 选定它的 Go 类型（不能是 `openapi_types.Date`：它所在的包导入 `github.com/google/uuid`），带着测试登记检查器（M2 设计 3.11）。
- **草稿发布**复用同一个结构检查：发布时，草稿的 `payload` 按"创建工作项"的请求 schema 走 `bodyshape` 的校验（M2 设计 3.11），与创建工作项走同一条路（总体设计 5.4，差异清单第四节"草稿发布"）。
- **关闭条件**：两处延伸各有单元测试（`tags[2]` 排在 `tags[10]` 之前；开放对象里的 `1e400` 得到带路径的 400）；同一个坏的请求体，草稿发布与创建工作项给出相同的 400；`date` 的检查器已登记，不合格的日期得到带路径的 400，有测试；第一个 map 型 schema 出现时，`closedObject` 有例外分支和反例用例（或者本 M 的 review 写明没有用到）；数组、开放 map 或递归的请求体 schema 出现时，`TestCheckCostsAboutTheBody` 有这种请求体，递归的由一遍读完的按表检查通过（或者本 M 的 review 写明没有用到）。

## 5. 60 天物理清理与删除关系图（M4 的部分）

- **做什么**：60 天物理清理（总体设计 5.5）把软删除的 `api_tokens` 纳入（撤销 PAT 就是软删除，M2 设计 4.4）。`issue_activities` 对工作项、评论的外键是 `DO_NOTHING`（照搬时不写 `ON DELETE`），先删工作项会被它挡住：先删除或置空这些引用，或者登记为 `SET NULL` 的差异。
- M4 的每张新表照 M2 设计 3.13 搬 `on_delete`，在 M2 设计 4.7 的图上延伸，写明物理删除时每条外键的去向（M3、M6、M7 各自延伸自己的表）。
- **关闭条件**：清理任务的集成测试覆盖软删除超过 60 天的 PAT 和带操作动态的工作项；M4 设计有延伸的图。

## 6. 定时任务和事件订阅者

- **漏注册 worker 没有响亮的迹象**（M2/P3b 评审第 7 节；收尾评审对照 `river@v0.47.0` 更正）：runner 只在 River 的 `Start` 失败时像数据库不可达一样无限重试、每次一条 WARN "jobs did not start; trying again"（`server/internal/platform/jobs/jobs.go:136`），而 `Start` 因配置失败只发生在一个 worker 都没有的时候（`river@v0.47.0/client.go:1102-1104`）。`identity` 总是注册 `identity.cleanup_expired_sessions` 的 worker，所以 M4 的模块给定时任务漏了自己的 worker 时，River 照常启动，leader 照常投递（定时任务的投递不核对 worker，`river@v0.47.0/periodic_job.go:254-260`），执行时 River 记 ERROR "jobexecutor.JobExecutor: Unhandled job kind"（`river@v0.47.0/internal/jobexecutor/job_executor.go:215`）并重试到放弃；`bootstrap` 漏接整个 `jobs.Job` 时什么都不记。M4 是第一个加业务定时任务的 M（自动归档、60 天清理）：加一个测试，漏注册时失败（M2 的做法是 `bootstrap` 的测试等 `identity.cleanup_expired_sessions` 在 `river_job` 中 `completed`，`server/internal/bootstrap/jobs_test.go:45`，M2 设计 3.15），或者让 `platform/jobs` 在构造时拒绝 kind 没有 worker 的定时任务。
- **事件订阅者的写法**：事务内投递用 `InsertTx(ctx, tx, …)`，回滚的事务不留任务（M2 设计 3.15 的 spike）；订阅者怎样写由第一个需要它的 M 定，就是 M4。
- **服务角色的权限**（M2 codex-fixes，M2 设计 6.1）：M4 的迁移加了表、视图或序列，或者 River 升级（总体设计 5.2）加了它们，同一个提交把授权加进 `deploy/runtime-grants.sql`（函数和类型靠 PUBLIC 的默认权限，收回时同样要加）。`server/internal/bootstrap/runtime_role_test.go` 的 `TestTheGrantsFileCoversEveryRelationAndFunction` 在少了或多了时失败；`TestTheRuntimeRoleServesWithTheGrantsFile` 用恰好这些权限运行 nerve（就绪、River 的清理任务、River 的索引重建），M4 的第一个业务定时任务在它读写的表上照样要有权限。
- **关闭条件**：漏注册 worker 的变异让测试失败；事件订阅者的写法写进 M4 设计；`deploy/runtime-grants.sql` 覆盖 M4 的表，上面两个测试通过。

## 7. 编辑器的代码分割

- **做什么**：未登录时直接打开需要登录的页面，页面模块先加载了编辑器（tiptap 的 Emoji 节点和 `is-emoji-supported`），包装层才跳到登录页。把编辑器拆出去，到用它的页面才加载。个人设置页的布局挂着命令面板（`ProjectsAppPowerKProvider`，`web/apps/web/app/(all)/settings/profile/layout.tsx`），它在打开之前就建了一个编辑器，一并处理（M2/P5 spec 第 3 节第 9 条）。
- 之后删除 e2e 对 `is-emoji-supported` 的 Chromium 警告（`willReadFrequently`）的预期 `EMOJI_CHECK_WARNING`（`e2e/fixtures/browser.ts`；S2 的深链接和个人设置的故事用它）。
- **关闭条件**：`EMOJI_CHECK_WARNING` 和它的使用都删除，S2 和个人设置的故事仍断言控制台没有别的警告，并通过。

## 8. CSP：编辑器 callout 的默认表情图；`worker-src`、`frame-src`

- **做什么**：`web/packages/editor/src/extensions/callout/utils.ts:19` 的默认表情图来自 `cdn.jsdelivr.net/npm/emoji-datasource-apple`，被页面 CSP 的 `img-src 'self' data: blob:`（`server/internal/platform/webui/csp.go:40`）挡住。改为本站资源或原生表情；核对表情回应（它也用表情选择器，数据由 M3 改为本站提供，[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 10 节）。
- **与 M8 的关系**：[M8 的 M1 收尾交接](../../M8-open-release/handoffs/M1-closeout.md)列了同一个请求；M4 改完之后，那一项对它只剩核对。
- **`worker-src`、`frame-src`**：页面的 CSP 没有写这两条，按 `default-src 'self'` 落回（M2/P4 评审第 4 节）。编辑器或它的扩展要用 `blob:` 的 worker（M4），收集箱或视图要嵌 iframe（M7，[M7 的交接](../../M7-collaboration/handoffs/M2-closeout.md)第 6 节）时，在 `server/internal/platform/webui/csp.go` 只加需要的来源，有测试。
- **关闭条件**：插入 callout、打开表情回应时没有 CSP 违规，页面不请求 `cdn.jsdelivr.net`；M4 的 review 写明编辑器是否用 `blob:` 的 worker，用的话 `csp.go` 只为它加了 `worker-src` 的来源，有测试。

## 9. 命令面板的主题命令

- **做什么**：`web/apps/web/core/components/power-k/config/preferences-commands.ts` 先 `setTheme`，再发 `PATCH /api/v0/me/profile`，失败时不撤回，页面停在 nerve 没有的主题；失败的提示是固定的文案，不按 `code`。P5 起 preferences 页（`web/apps/web/core/components/appearance/theme-switcher.tsx`）在 nerve 应答成功之后才应用主题（M2/P5 spec 2.7）；M4 处理命令面板时照样改（应答成功、且标签页仍在发出修改时的会话才 `setTheme`，同 `theme-switcher.tsx` 的 `inSession()`；总体设计 7.7）。
- **关闭条件**：命令面板的主题命令在应答成功之后才应用，失败时按 `code` 提示，有故事或浏览器核对。

## 10. 下拉框（M4 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。根因和修法见 M2/P5 spec 2.3。

- **`DateDropdown`**（`dropdowns/date.tsx`）要只有一份打开状态的结构，例如 Popover（选中日期时调它的 `close`），连同它在列表和表格单元格里的懒渲染（`ComboDropDown` 的 `renderByDefault`）。P5 保留了两份状态，经 `onClose` 同步：键盘打不开日历；日历里按 Tab 关闭之后两份状态走散，再点另一个下拉框时两个列表同时开着。api-tokens 页的自定义有效期也用它。
- **`CustomSearchSelect` 的 `defaultOpen`**（挂载之后点一次按钮）没有在浏览器中核对，接上富筛选时核对。
- **12 个同样写法的下拉框中 M4 的 7 个**：`dropdowns/{date-range,priority,project/base,state/base}.tsx`、`issues/{issue-detail/label/select/label-select,issue-layouts/properties/label-dropdown,select/base}.tsx`。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。`date-range`、`project/base` 也在 M6 的周期、模块表单里，M6 接上时核对 M4 的修改。
- **P5 重建的 `CustomSelect`**（`Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M4 的 3 个调用方：`automation/auto-archive-automation.tsx`、`core/filters/date-filter-select.tsx`、`issues/peek-overview/header.tsx`。
- **`CustomSearchSelect` 在 M4 的调用方**（一份打开状态，清空搜索不再交出 `null`，`onOpen` 每次打开调一次）：`rich-filters/{add-filters/dropdown,filter-item/root,filter-value-input/select/single,filter-value-input/select/multi}.tsx`。
- **关闭条件**：M4 的 review 逐项写明：`DateDropdown` 能用键盘打开，只有一份状态；7 个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。

## 11. 复制到剪贴板（M4 的 6 处）

- P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M4 的 6 处没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝（路径在 `web/apps/web/core/components/` 下）：`issues/issue-layouts/quick-action-dropdowns/helper.tsx`、`issues/peek-overview/header.tsx`、`issues/issue-detail/issue-activity/helper.tsx`（`.then` 在一个不等待它的 `try` 里）、`issues/issue-detail/links/link-item.tsx`（不等待，照样提示已复制）、`issues/issue-detail-widgets/{relations/helper.tsx,sub-issues/helper.ts}`（`await` 了，调用方不处理）。
- **关闭条件**：6 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。

## 12. stores 按会话分代（规则在总体设计 7.7）和 M2 留下的调用

- **做什么**：M4 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M4 的范围（`d97c513` 上数出）：
  - **注册在沿用的 `router` 上的反应**：`IssueRootStore` 的 `autorun`（`web/apps/web/core/store/issue/root.store.ts:170`），随退役的一代释放。
  - **模块级的 service 实例**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中按调用的接口属于 M4 的 13 处：`IssueService` 2、`WorkItemVersionService` 2、`WorkspaceDraftService` 1；搜索的 4 处，`ProjectService` 的 `projectIssuesSearch` 3 处（`web/apps/web/core/components/core/modals/existing-issues-list-modal.tsx`、`web/apps/web/core/components/issues/parent-issues-list-modal.tsx`、`web/apps/web/core/components/inbox/modals/select-duplicate.tsx`）和 `WorkspaceService` 的 `searchWorkspace` 1 处（`web/apps/web/core/components/power-k/ui/modal/search-menu.tsx`）；编辑器 @提及 的 `WorkspaceService.searchEntity` 4 处（`web/apps/web/core/components/editor/lite-text/editor.tsx`、`web/apps/web/core/components/editor/rich-text/description-input/root.tsx`、`web/apps/web/core/components/issues/issue-modal/components/description-editor.tsx`、`web/apps/web/core/components/inbox/modals/create-modal/issue-description.tsx`）。
  - **`store-context.tsx` 导出的 `rootStore` 的三处同步读取**：`web/apps/web/core/components/issues/issue-layouts/utils.tsx`，以及 `web/apps/web/core/store/command-palette.store.ts`、`web/apps/web/core/store/issue/issue.store.ts` 中只读、不发请求的两处。它们接上新接口之后仍只能同步读取；要发请求就改经自己的 `RootStore`。
  - **SWR 键**：工作项、列表、评论等的 SWR 键带上 `loginId`。
- **M2 留下的调用**：`web/apps/web/core/services/user.service.ts:52` 的 `getUserProfileIssues` 调 Plane 的旧地址，改接新接口或删除。
- **关闭条件**：`IssueRootStore` 的反应随退役的一代释放，有单元测试；上面的 `git grep` 中 M4 的 13 处都已消失；三处同步读取不发请求；`getUserProfileIssues` 改接或删除。

## 13. e2e 的 `clock.ts`（M0-P6 交接剩下的一项）

- **做什么**：可注入的时钟用于"时间流逝"类故事（自动归档），按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/clock.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。nerve 这一侧的时钟怎样在测试中注入（例如 test 配置下的时钟），由 M4 设计定。
- **关闭条件**：`clock.ts` 按这个写法加入，自动归档的故事用它，不等真实的时间。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M4 的各行；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
