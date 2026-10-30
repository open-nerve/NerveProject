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
- **结构检查的代价**（M2 设计 3.11，codex-fixes）：路径随读随写、只留前 257 字节，报出的路径最长 256 字节（更长的截短，以 `…` 结尾），同一个问题只报一次，一个请求体最多报 16 个问题；`bodyshape` 的 `TestCheckCostsAboutTheBody` 和整程序测试 `TestTheAnswerToABrokenBodyStaysSmall` 守住。M4 的路径（`tags[10].name`）远不到 256 字节，截短碰不到它们。按表检查每层把对象的成员复制一次，代价是请求体乘 schema 的层数；M2 的请求体 schema 最深两层、不递归。第一个递归的请求体 schema 让它变成请求体乘请求体的深度：那时按表检查改为一遍读完，代价测试加上这种请求体。第一个数组或开放 map 的请求体 schema 到来时，代价测试同样加上它。
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
- **关闭条件**：6 处都改经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`，M3/P10；复制文字的是同一文件的 `useCopyText`，M3 收尾），失败时经 `t()` 提示；关键词规则 `copy-hook` 中它们的例外随之删除（M3 收尾，[M3 的交接](M3-closeout.md)）。

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

## 处理结果（M4/P1b）

- **第 3 节，`updated_at`**（P1a、P1b 的写完成）：`createIssue`（P1a）和 `updateIssue`（P1b）把用例的时钟在项目 N 之下读到的一个时刻写进它写的每一行：修改写下工作项的 `updated_at`、`updated_by_id`，新加的负责人、标签、关注和每一行动态的 `created_at`、`updated_at`，去掉的负责人、标签的 `deleted_at`、`updated_at`。用例的测试用固定的时钟（`issue/app` 测试的 `clockAt`）断言这些列等于它；组合的测试断言它落在请求之内（`TestUpdatingAWorkItemStampsItsRows`）；端到端的 I2 断言修改写下的每一行是工作项的 `updated_at`、是修改者的。

仍未处理，状态保持 `open`：第 3 节其余写工作项的用例（M4 的 P2–P4、P7）和自动归档推进固定时钟的测试（M4 的 P6）；第 1、2、4–13 节，按 M4 设计 13.1 的落点。

来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

## 处理结果（M4/P2）

- **第 3 节，`updated_at`**（P2 的写完成）：`deleteIssue`、`archiveIssue`、`unarchiveIssue` 和带日期的 `updateIssue` 把用例的时钟在项目 N 之下读到的一个时刻写进它写的每一行：删除写下工作项和它的负责人、标签、关注、动态的 `deleted_at`、`updated_at`，归档、恢复写下工作项的 `archived_at`（恢复为空）、`updated_at` 和那一行动态的 `created_at`、`updated_at`。用例的测试用固定的时钟（`clockAt`）断言交给仓储的时刻等于它，仓储的测试断言这些列等于交给它的时刻；组合的测试断言它落在请求之内（`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、`TestArchivingAWorkItemStampsItsRows`）；端到端的 I15 断言归档、恢复的那一行动态是工作项的 `updated_at`、是写它的人的，删除的每一行同一时刻、同一个删除者（`expectIssueDeleted`）。
- **第 4 节，路径的排序**（完成）：字段错误的路径按数组下标的数值排序（`bodyshape` 的 `comparePaths`，`tags[2]` 在 `tags[10]` 之前，一个全序）；单元测试 `TestComparePaths` 和 `Check` 的表测试（`ids[2]` 在 `ids[10]` 之前）；M2 设计 3.11 改写。
- **第 4 节，数的范围**（完成）：不限类型的节点（`{}`、开放对象未声明的成员、没有 `items` 的数组的元素）里 float64 放不下的数在它的路径上报 `invalid_format`（`ambiguity.go` 的 `numbers`，一遍读完那个值）；`Check` 的表测试在这三种节点里各放一个 `1e400`；M2 设计 3.11 改写。
- **第 4 节，第一个 `format: date` 的字段**（完成）：Go 类型是 `shared.Date`，五个模块的 `oapi-codegen.yaml` 都映射过去；`bodyshape` 登记 `FormatDate`（检查器照 `shared.ParseDate` 的规则读，平台不导入 `shared`），生成器认它；`issue` 的 HTTP 适配器的 `TestDatesPassTheBodyCheckAsTheyDecode` 核对检查器与生成的解码器对同一组写法结论相同。不合格的日期得到带路径的 400：`bodyshape` 的 `TestCheck`（日期写错的 `on`）和 `TestFormatDate`；`apitest` 为契约的每个日期字段生成一个不合格的请求体用例（`TestBodyCasesOfDates`），整程序测试 `TestBodiesThatBreakTheStructureAnswer400` 把它们发给 `createIssue`、`updateIssue`。
- **第 4 节，结构检查的代价**（数组完成）：`TestCheckCostsAboutTheBody` 加上 M4 的三个请求体：200,000 个不是 uuid 的字符串的数组（每个元素一个格式问题）、1 MiB 的错误日期、不限类型的值里 9,990 层数组之下一个 float64 放不下的数；数组的请求体（M3 的 `hide_in_more_menu`、P1a 的 `assignee_ids` 一类）的代价由它守住。开放 map 和递归的请求体 schema：M4/P2 没有用到。
- **第 4 节，map 型对象**：M4/P2 没有用到（契约里没有 `additionalProperties` 为 schema 的对象），`closedObject` 不改。

仍未处理，状态保持 `open`：第 3 节其余写工作项的用例（M4 的 P3、P4、P7）和自动归档推进固定时钟的测试（M4 的 P6）；第 4 节的草稿发布（M4 的 P8）、map 型对象和递归的请求体 schema（第一个用到它们的 Phase，或 M4 的 review 写明没有用到）；第 1、2、5–13 节，按 M4 设计 13.1 的落点。

来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

## 处理结果（M4/P3）

- **第 3 节，`updated_at`**（P3 的写完成）：改父工作项的 `updateIssue`，`addSubIssues`、`createIssueRelations`、`deleteIssueRelation` 和删除子树的 `deleteIssue` 在取得全部锁之后读用例的时钟（跨项目的写在涉及的每个项目的 N 之下），把这一个时刻写进它写的每一行：移动的子工作项的 `updated_at`、`updated_by_id` 和它的一行动态；关联行的 `created_at`、`updated_at`（删除时 `deleted_at`、`updated_at`），两端工作项的 `updated_at`、`updated_by_id` 和各自的一行动态；子树里每个工作项和它们的负责人、标签、关联、关注、动态的 `deleted_at`、`updated_at`。用例的测试用固定的时钟（`clockAt`）断言交给仓储的时刻等于它，仓储的测试断言这些列等于交给它的时刻；组合的测试断言加关联、删除关联、删除子树一次写的每一行是同一个时刻、是写它的人（`TestWorkItemsAreRelatedAcrossProjects`、`TestARelationIsDeletedFromEitherWorkItem`、`TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`）；加子工作项一次写的行是同一个时刻，由单元的 `TestAddSubIssuesPutsTheWorkItemsUnderIt`（交给仓储的移动、它们的动态和回答的 `updated_at` 都是固定时钟的那一刻）和端到端的 I8 断言，组合的 `TestWorkItemsArePutUnderAWorkItem` 只断言移动的每一个和它的一行动态是写它的人的；端到端的 I8、I9 断言一次写在工作项下的行（`expectIssueUpdated` 的 `wrote`：P3 的每次写是它的一行动态）是工作项的 `updated_at`、是写它的人的，关联行由 `expectRelations` 断言是做它的人做的、删除它的人最后写的，I15、I17 断言删除的每一行同一时刻、同一个删除者（`expectIssueDeleted`）。
- **第 4 节，map 型对象和递归的请求体 schema**：M4/P3 没有用到。P3 的请求体（`SubIssuesAdd`、`IssueRelationsCreate`，`IssueCreate`、`IssueUpdate` 加的 `parent_id`）都是封闭的对象，数组的元素是 `uuid` 的字符串，代价由 P2 加进 `TestCheckCostsAboutTheBody` 的 uuid 数组守住；`closedObject` 不改。

仍未处理，状态保持 `open`：第 3 节其余写工作项的用例（M4 的 P4、P7）和自动归档推进固定时钟的测试（M4 的 P6）；第 4 节的草稿发布（M4 的 P8）、map 型对象和递归的请求体 schema（第一个用到它们的 Phase，或 M4 的 review 写明没有用到）；第 1、2、5–13 节，按 M4 设计 13.1 的落点。

来源：[M4/P3 spec](../specs/P3-issue-hierarchy.md) 第 7 节。

## 处理结果（M4/P4）

- §2：T2严格三种游标，T3评论、T10描述历史、T12动态分别以自己的排序键/最终id分页；合法篡改只能换可见数据的起点。T14真实HTTP篡改/权限补证；P5其他工作项排序仍需自有载荷，不复用PAT。
- §3：描述、评论、工作项表情用锁下同一时刻更新对应审计；评论表情按设计不推进主工作项/评论审计。T13真实锁等待后推进Clock、T14真实SQL回滚、T15 API审计断言已验证；P6自动归档/P7其余写仍待后续。
- §4：P4请求都是闭合对象，没有开放map或递归schema，不放宽closedObject；date/下标/数字范围沿P2已完成证据，草稿发布留P8。
- §5/6：五表grants与目录删除在T3/5/6/9同批落地，comment FK物理CASCADE纳入图；P4同步软删保留旧删除戳/邻居。60天清理（含PAT）、每工作项最新20版保留、项目/工作区异步删除仍是P9；P6事件/worker完整性/只投递客户端，未实现或运行这些任务。
- 余项保留§1负责人原话和原样转交条件；§7–13编辑器按需加载、CSP/表情图、上传、下拉框、复制hook、stores分代、clock.ts均依原M4设计13.1落点继续；本Phase没有以S3往返替代页面或CSP核对。

状态保持 `open`；本次仅完成P4接收部分，待M4真正收尾逐项核对关闭条件。证据：[P4 spec §4/§7](../specs/P4-issue-collaboration.md#4-验收标准)、[P4 review §2/§6](../reviews/P4-issue-collaboration-review.md#2-验收标准核对)。

## 处理结果（M4/P5）

- §2：T3/T5–7/T9/T10/T12的19排序使用各自定型键和最终id的canonical载荷；合法篡改只换可见范围的起点。CSV/none/单日/重复日期AND、PageSize和闭合响应沿既有规范；三形状与各层去重total、组格续页原筛选、RO/RR同快照及空页补total均有实际SQL/HTTP/变异证据。
- §3：T8 sort_order全链仍经原写锁、闸门、锁下时钟，同次state+sort只一条state动态；skip true/false互斥及新旧交错race五遍已验证。T14历史写/读与边界实际回归，T15 I6验证审计与落库；不把列表读引入写锁或闸门。
- §4：T13三设置GET/PATCH使用闭合typed对象、形状400/词值422，未新增开放map或递归schema；T13a旧1–25→26→Down/Up、T13b四方向race、T14写隔离与T15 I5各有证据。T12最终32轮保留真实索引、完整排序/temp和跨项目/成员相关扫描成本，交M8，不用池参数读回替代旧删除性能。
- 仍开放：§1原话/原样转交，草稿P8、自动归档P6/其余写P7、清理P9、§7–13页面/编辑器/CSP/上传/下拉/复制/stores/clock.ts各原落点；P10b共享issue-order.json的生产码点比较器/vitest和旧筛选store迁移、P11详情数据及P12–15页面均未由P5关闭。

仅登记本输入的P5承接，`status: open`保持；M4总体、T16内部整分支审查/最终分支CI与负责人终审/集成另待收口，不合并。证据：[P5 spec §2/§7](../specs/P5-issue-list.md#2-交付物)、[P5 review §2](../reviews/P5-issue-list-review.md#2-验收标准逐条核对)。

## 处理结果（M4/P6）

§3自动归档从 `updated_at` 与同源业务时钟读阈值；§6只投递客户端的原事务回滚、worker/周期两类漏登记、运行角色grants和真实completed已由T1/T7–T11、T15验证；§13独立 `clock.ts` 及I16真实时间水位由T13/T14验证。§1命令行只投递原文保留，M4收尾转M5，当前CLI只接纯业务时钟、不构造jobs。P7/P9及页面/编辑器原落点仍开放；本handoff总体 `status: open`。证据：[P6 review §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)、[§6](../reviews/P6-issue-events-review.md#6-移交事项)。

## 处理结果（M4/P7，2026-10-27）

§3链接三写经原Locks.run：拒绝前置、锁下重读/授权/归档/noop/唯一，再取一次Clock，先issue审计再link，活动和事件同刻同写者；任一失败原事务回滚。worker单句id+URL+未删除守卫只更新metadata，不推进issue/link审计、不加活动/事件。§6首个生产subscriber通过既有issue_inserter/InsertTx投递，worker登记与grants、真实受限角色CRUD+crawl已验证；T15/T16分别记录准确源码的验证，不把配置3attempt声称为已注入真实DB故障观察三次重试。

§1命令只投递原文及收尾原样转M5要求保留，CLI仍不构造出站/jobs。P8草稿、P9清理/版本保留、§7–12页面与其他原落点继续开放。

本回执只关闭P7对应子责任，顶部`status: open`保持；M4整体、P8/P9/P11/P14/P15及M5–M8最终closeout不提前关闭。证据：[P7 review §2](../reviews/P7-issue-links-review.md#2-验收标准逐条核对)、[行为矩阵](../reviews/P7-issue-links-behavior-matrix.md)、[整体审核](../../../reviews/re-05-M4P7整体审核.md)。本波独立定向复审与最终分支CI状态在P7 review单列，未合并main。

## 处理结果（M4/P8，2026-10-27）

§2草稿游标已在T2/T4验证：[created_at,id]canonical共享编码，另一owner/随机id合法游标只是位置，页与total仍仅自己，非法400；不签名、不回查游标ID。§4草稿发布读取锁下存储payload，由adapter复用create bodyshape和mapper，与直接创建以同canonical字节比较结构400/字段、业务422同路径，>16字段错误截断有反例；API封闭schema与成本守卫未放宽。T5/T6共享唯一创建核心、同事务/actor/At，事件/soft-delete/描述版本三故障逐表回滚；T7 gate无连接/事务和真实PID交错，T15最终race65/65。§6部署说明明确runtime经PUBLIC EXECUTE使用扩展函数、收回后显式授权，关系/全函数目录守卫保留。§1 CLI原文、其他页面/清理/后续M5–M8责任保持原归属，顶部status仍open。

准确产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6`，首次整审 `166b9efd..10f44790`；[P8 review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)列A条款/真实RUN/变异恢复/四次七门禁及准确产品checkpoint CI，[整体审核](../../../reviews/re-07-M4P8整体审核.md)列三镜头/186文件覆盖与历史D1。文档增量独评及最终准确HEAD CI由后续提交和ignored收据追踪，未合并、不清理；不以本期子项已处理改写历史关闭条件或M4整体状态。

## 处理结果（M4/P9，2026-10-28）

| 原输入 | 本期处理结果 | 证据/仍开放责任 |
|---|---|---|
| §5物理清理/关系图 | PAT、带活动/评论/二级反应的issue以及M3目录已有真实61d回收，59d/精确cutoff/活行保留；活动/评论子链CASCADE，不再沿用历史DO_NOTHING阻挡判断 | T1–4/T13 I21保存ID物理直查，T15真实FK反例；不清账户，session仍原任务 |
| §6任务/订阅者 | purge/trim四周期目录实际completed，delete_rows请求kind登记、原事务投递失败回滚、受限角色与真实重试/snooze/耗尽已有证据 | T4/5/8–13；未来通知/Webhook订阅仍所属阶段 |
| §1只投递CLI | P9仅移除Cascade的issue过渡依赖，无新投递命令/dispatcher/issue.New/jobs/InsertTx接线 | 首个真实投递命令仍M5，既有NewInserter交接保留 |

§2游标、§3updated_at、§4请求体/草稿、§7–12编辑器/CSP/主题/下拉/复制/会话与§13时钟继续按历史回执/所属阶段追踪，不能因本表全关。

证据：[P9 review](../reviews/P9-purge-review.md)及[spec §7/附录D](../specs/P9-purge.md)；最终产品 `e003e96625ba51264d3aeb77729729d53a770103`。最终F/准确分支CI、T16文档独评在review分别列明。本文件 `status: open` 保持到M4收尾，历史原文/先前回执保留；不因P9后端承接关闭整份交接。


## 处理结果（M4/P10a，2026-10-28）

第12节本期子项：核心已直接采用生成Issue/IssueDetail，nullable actor/parent、真实Detail描述与局部表单/请求边界准确化；四类列表/筛选根和部分无读者路由/目录镜像、方法/状态已删除。T7删除部分自读autorun镜像不等于IssueRootStore会话治理整项完成；剩余autorun严格模式历史输出继续保留，未关闭store-context警告性质线。

P10b仍承担列表/会话分代、router派生、三处同步读取边界与SWR loginId；P11及P12–P15依设计分配迁模块级service、profile旧接口及各页面。T5/R15新增资源拒绝由真实Base错误/重试和关闭/换来源ownership闭合，只覆盖新增链，不代表所有拒绝/会话性质已验。证据与明确余项见[P10a移交](../reviews/P10a-web-issue-types-review.md#6-移交事项)，顶层status保持open。

本期Task级完成至T8提交8081112492a2fc744a43b3e814c6d36ff4ba3959；A-01/A-02代码终审修复已独评并提交5f7085bce4aa5103a0133fa982847d274efa6e47；文稿最后独评已通过，最终交付HEAD分支CI待提交后核验，准确产品5f708 CI已核成功见review§8，不把本段子项写成M4最终交付或集成。


整审A-01无消费者子项两域预取与A-02退休列表独占乐观分布链已根因清理并独评；未改会话/资源/领域协议，真实目录/统计/进度保留，不扩大本输入其余关闭范围。

## P10b 本段处理（2026-10-29）

§12 工作项 router autorun 改为计算属性；权限观察器在 retire 释放。三处同步读取、列表 SWR loginId、共享排序 fixture 与比较器由 T3/T10 验证；详情 SWR及模块 service 按 P11–P15 原分配，P11核旧 profile 方法删除。 证据见 [P10b review](../reviews/P10b-web-list-data-review.md)；整审及N2限定修复复审已闭环，产品候选2e1264aa完整F及准确CI40896656404 attempt1三任务通过，实施与终审完成、集成待办；纯文稿HEAD CI由主控另核；整份交接 status 保持 open。


## 处理结果（M4/P11，2026-10-30）

§12原七处模块实例加归档header/issueId跳转两处前移实例共九处已消失，四个提及入口由本代成员目录读取；详情/版本/草稿SessionSWR键有login与权限身份，资源和队列继续只消费IssueRootStore失效水位，退役与ABA/旧宿主都有回归。getUserProfileIssues已于P10b删除；三处同步root读取未改为异步发请求。剩余四处旧搜索实例按P12/P14/P15迁移，13实例总门未提前关闭。

依据[P11 review§9/12/14](../reviews/P11-web-detail-data-review.md#9-最终验收映射)和原始迁移审计；完整剪贴板/CSP/页面故事仍由后续阶段承担，顶层status保持open，P11尚未合并。

## 处理结果（M4/P12，2026-10-30）

- §10 DateDropdown在T8改为单Popover状态，列表/表格键盘可装载，单日期与范围相邻切换不会互抢焦；api-tokens真实自定义有效期表单及A11回归通过。七个点名控件的实际浮层ref、单状态、具名触发器和外层Modal隔离分别由T9/T10/C3核验；Calendar网格用Popover，离散搜索选项复用CustomSearchSelect，不机械套Listbox。CustomSelect的peek、自动归档、日期筛选三个真实入口有C3兼容证据。最终T16三浏览器各11项已通过，WebKit使用Docker恢复后的新收据，不据T15旧绿充数。
- §10剩余：CustomSearchSelect的defaultOpen及rich-filters四入口完整筛选验收留P13；M6/M7共享props兼容不算领域页面验收。仅本段明确测试的控件性质已处理，不关闭整个§10或顶层交接。
- §12 ExistingIssuesListModal的模块级ProjectService已删除，三调用改用本代root持有的IssueSearchService；原helpers拆为显式scope目录函数且保持同步零HTTP，M04/M05和路由回归捕获错范围/偷发请求。parent、inbox查重和命令搜索余下实例/旧类型继续由P14/P15负责；不虚报13实例全零。
- §7编辑器分割和§11复制hook仍按P14/P15完成；本段watchPage只精确期待已有Canvas提示，不掩盖其它console/API失败。实施、范围与最终交付核准规则见[P12 review§23–25](../reviews/P12-web-list-pages-review.md)和[整审](../../../reviews/re-17-M4P12整体审核.md)。顶层status保持open。
