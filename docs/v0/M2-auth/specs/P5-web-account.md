# M2/P5 个人设置：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M2/P5 `web-account` |
| 日期 | 2026-09-27 |
| 状态 | 进行中 |
| 上级文档 | [M2 设计文档](../M2-design.md) 第 2（A7–A9、A11、A12）、3.5、3.20（P5 一行）、5.3、7.3、7.5、7.7–7.9、9.5、9.6（P5）、12（P5）、13.1、13.2、§16 节；[Codex 设计评审](../reviews/M2-design-codex-adversarial-review.md) M-6 |
| 前置交接 | [M1-closeout](../handoffs/M1-closeout.md)（死成员和死 prop、oxlint、主题下拉框）、[M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) 剩下的一项（`@nerve/services` 的 API 令牌旧地址）；[P4 评审记录](../reviews/P4-web-auth-review.md) 第 6 节交给 P5 的事项 |
| 计划 | [P5 plan](../plans/P5-web-account.md) |

本 spec 只写 M2 设计交给 P5 决定的东西：名字、签名、规则的细节、测试，以及原型证明了什么。规则本身以 M2 设计为准，这里引用节号，不重述。P5 建立在 P4 之上，沿用它的约定（每个会话一个 `RootStore`，service 从构造函数拿这一代的客户端，填 store 的 SWR 键带 `loginId`，`SessionChangedError` 的调用方规则；P4 spec 2.8、P4 评审第 3 节）。收尾的工作（10 份交接的结论、上级文档的总核对、给 M3–M8 的交接）不拉进来（第 5 节）。

## 1. 目标

按 M2 设计 12 节 P5 的目标：个人设置的四个标签页对接新接口；M2 领域的前端清理完毕。具体是：

- 设置页的下拉框在按钮旁展开（7.7 的根因，修在组件里），Escape 之后一次点击就再打开；
- PAT 的 service 和 store：属于会话的 `RootStore`，用这一代的客户端，逐页取完列表，令牌原文只交给创建它的弹窗；
- api-tokens 页：列表（创建时间、最后使用时间）、创建（1 周、1 个月、3 个月、1 年、自定义日期、永不过期；原文只显示一次，同时下载 CSV）、撤销；
- security 页在修改密码的表单下方列出 PAT，可以就地撤销，说明修改密码不撤销它们；
- preferences：时区列表来自 `GET /api/v0/timezones`；主题、语言、时区、每周第一天的失败按 `code` 提示；
- general 页的保存按字段显示错误；停用账户的弹窗按 Nerve 的行为改写文案；维护页不再暗示 nerve 没有启动；
- `@nerve/services` 删到只剩地址工具和文件工具；Plane 的 `IApiToken`、`TTimezoneObject` 删除；
- 死成员和死 prop（`domains.mjs --rows M2`）清到只剩误报；P4、P5 改到的文件 oxlint 清零；`no-unneeded-ternary` 全仓清零；上限调低；
- 关键词规则 3 条；
- A7–A9、A11、A12 的页面版本；
- 3.20 中 P5 的一行、README、两份交接的处理记录。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录，`web/apps/web/` 简写为 `web:`。"Task"是 plan 中负责它的任务；几个 Task 号表示先写过渡版本、最后一个 Task 写成最终版本。plan 的文件结构表逐个列出；共改动或新增 130 个文件（不算 plan、本 spec 和 M2 设计的进度表），删除 6 个。

| 路径 | 内容 | Task |
|---|---|---|
| `web/packages/ui/src/dropdowns/{custom-select,custom-search-select,combo-box,helper}.tsx`、`web:core/components/dropdowns/date.tsx` | 下拉框的定位和打开状态 | 1 |
| `web:core/services/api-token.service.ts`、`web:core/store/user/{api-token.store,api-token.store.test,index}.ts`、`web:core/store/workspace/index.ts`（`api-token.store.ts` 删除）、`web:core/lib/auth/fake-nerve.ts`、`web:core/lib/store-context.test.ts` | PAT 的 service 和 store | 2 |
| `web:core/components/api-token/`、`web:core/components/settings/profile/content/pages/api-tokens.tsx`、`web:core/components/ui/loader/settings/api-token.tsx`、`web:core/hooks/store/user/` | api-tokens 页 | 3 |
| `web/packages/services/`（`api.service.ts`、`developer/` 删除）、`web/packages/types/src/{api_token,timezone}.ts`（删除）、`web/packages/constants/src/fetch-keys.ts`、`pnpm-lock.yaml` | `@nerve/services` 和 Plane 的类型 | 3、6 |
| `web/packages/i18n/src/locales/{en,zh-CN}/{settings,workspace-settings,common}.json` | 文案 | 3、5、7 |
| `tools/keywords.json` | 3 条规则；`services-files` 收窄 | 3、6 |
| `e2e/fixtures/{settings-pages,browser}.ts`、`e2e/stories/identity/a{7,8,9,11,12}-*.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts` | fixture；页面版本 | 4–7 |
| `web:core/components/settings/profile/content/pages/security.tsx` | 安全页的 PAT 列表 | 5 |
| `web:core/services/timezone.service.ts`、`web:core/hooks/use-timezone.tsx`、`web:core/components/{appearance/theme-switcher,profile/start-of-week-preference}.tsx`、`preferences/language-and-timezone-list.tsx` | preferences | 6 |
| `web:core/components/settings/profile/content/pages/general/form.tsx`、`web:core/components/account/deactivate-account-modal.tsx`、`web:core/components/instance/maintenance-message.tsx` | general、停用账户、维护页 | 7 |
| `web:core/store/{user,issue}/…`、`web/packages/constants/src/settings/profile.ts`、5 个组件和包装（plan Task 8 逐个列出） | 死成员和死 prop | 8 |
| 10 个组件、`web/packages/utils/src/auth.ts`（plan Task 9） | P4、P5 改到的文件的 oxlint | 9 |
| 44 个文件（plan Task 10） | `no-unneeded-ternary` | 10 |
| `web/apps/web/package.json`、`web/packages/{ui,utils}/package.json` | oxlint 上限：web 551 → 457，ui 25 → 19，utils 18 → 16 | 1–3、6、8–10 |
| `docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M2-auth/handoffs/{M1-P3-trim-platform,M1-closeout}.md` | 文档同步、交接 | 11 |

### 2.2 依赖

- 不加任何 npm 包，Go 依赖不变，不改 `api/`、`server/` 和任何生成物。
- `@nerve/services` 去掉 `axios` 依赖（它的 axios 基类删除；web 和其他包仍用 axios）。`pnpm install` 之后 `pnpm-lock.yaml` 只少那个包的三行，plan 给出差异；复现时从前一个 lock 出发执行 `pnpm install`，得到的 lock 与 plan 逐字节相同。

### 2.3 下拉框（M2 设计 7.7；M1-closeout 交接）

- **根因**（原型中查明，附录 A 的 E1）：Headless UI 2.2.10 的 `Combobox.Options` 在渲染时把自己唯一的子元素克隆一份，换上自己的 ref（它的 `Frozen` 组件）。Plane 把 react-popper 的 `ref={setPopperElement}`、`style`、`attributes` 放在这个子元素上，这个 ref 从来没有被调用，`usePopper` 没有元素可定位，列表停在默认的 `position: absolute; left: 0; top: 0`，即页面的左上角（列表经 `createPortal` 挂在 `document.body` 下）。与界面语言无关：M1 收尾交接写的是 zh-CN，英文下一样（在 `cec4ec9` 的构建上实测：列表的 `style` 是 `left: 0px; top: 0px`，左上角在 (0, 4)，4 像素是列表自己的上边距）。
- **修复**：把 popper 的 ref、样式、属性和原来子元素的 `className` 放在列表元素本身（`Combobox.Options`，`CustomSelect` 改用 `Listbox` 之后是 `Listbox.Options`；`as="ul"`），子元素只留内容；Headless UI 转发这个 ref。修在三个组件：`CustomSelect`（主题、语言、每周第一天、api-tokens 的有效期）、`CustomSearchSelect`（时区）、`DateDropdown`（自定义有效期的日历）。它们都在 P5 的页面上。同样写法的其他 12 处在 M3–M6 的页面上，交给接上它们的 M（第 7 节）。
- **`CustomSearchSelect` 不设模态**（`modal={false}`）：Headless UI 2.2 的模态列表对"不包含输入框、按钮、列表的祖先的兄弟"设 `inert`；搜索框在列表里，于是列表的选项被设为 inert，点不了。Plane 的结构同样如此（原型实测：默认的 `modal` 下被设为 inert 的正是选项，`modal={false}` 下没有）。列表修到按钮旁之后，时区的选项才第一次真正可以点（A8）。`CustomSelect`（`Listbox`）、`DateDropdown` 没有输入框，保持默认的模态：列表打开时页面其余部分 inert、不能滚动。
- **打开状态**：Plane 的三个组件都有自己的 `isOpen`，Headless UI 另有一份，列表按组件的一份渲染，两份不同步。键盘打不开列表：Headless UI 的按钮处理 Enter、空格、方向键并停止冒泡，只打开它自己的一份。按钮上的 Escape 只关 Headless UI 的一份：`CustomSelect` 要点两次才再打开，`CustomSearchSelect`、`DateDropdown` 的列表是 `static` 的，关不掉（第 3 节第 3 条）。现在：
  - **`CustomSelect` 改用 Headless UI 的 `Listbox`**，打开状态只有它的一份：列表按 `Listbox` 的 `open`（render prop）渲染；组件的 `isOpen`、外层的按键处理、点击外部的检测、选项上只为关闭列表的点击处理都删除。没有搜索框的选择正是 `Listbox` 的用途：`Combobox` 只在输入框里处理方向键和 Enter，它的按钮固定 `tabIndex: -1`，没有输入框时键盘既到不了按钮，也移动不了、选不了选项；点击打开之后焦点不在任何控件上，Escape 无处可去。`Listbox` 的按钮在 Tab 顺序里；点击、空格、上下方向键打开，打开后焦点在列表上：方向键移动，Enter 或空格选中并关闭，输入字母跳到对应的名称；Escape、Tab、点击外部关闭，Escape 之后焦点回到按钮。按钮上的 Enter 不打开列表：在表单里它提交表单，与原生 `select` 相同（Headless UI 2.2 的 `ListboxButton`；api-tokens 创建弹窗的有效期按钮上按 Enter 即提交）。角色仍是 `listbox`、`option`，公开的 props 不变，使用者不改。
  - **`CustomSearchSelect` 仍用 `Combobox`**，打开状态只有 Headless UI 的一份：列表按 `Combobox` 的 `open` 渲染；组件的 `isOpen`、外层的按键处理、点击外部的检测、选项上的点击处理都删除。搜索框在列表里，列表打开时它取得焦点（ref 回调，不滚动），于是输入、方向键、Enter、Escape 都由 Headless UI 的输入框处理：点击按钮时 Headless UI 在 `pointerdown` 上 `preventDefault`，没有这一步焦点不在任何控件上。其余：
    - `onClose` 交给 `Combobox`；Headless UI 没有"打开"的回调，调用方的 `onOpen` 由随列表挂载的元素调用一次；
    - `defaultOpen`：`Combobox` 没有这个属性，组件挂载之后点一次自己的按钮（ref 保证只点一次）。它的使用者是 M4 的富筛选，没有在浏览器中核对（第 5、7 节）；
    - `Combobox` 在单选时把清空输入框当作选了 `null`。这里的输入框只筛选选项，所以 `null` 不交给调用方，清空搜索不改值（Plane 原来清空时区的搜索就发出 `PATCH /api/v0/me`，`user_timezone` 为 `null`）；
    - 搜索框的占位、没有结果、加载中的文字是可选的 prop（`searchPlaceholder`、`noResultsMessage`、`loadingMessage`），默认是原来的英文；时区选择由 Task 6 传入 `t()` 的文案；
    - 已知的限制：按钮不在 Tab 顺序里（Headless UI 的 `Combobox.Button` 固定 `tabIndex: -1`；`Combobox` 以常显的输入框为 Tab 停点，这里的搜索框却在列表里）。P5 之前也是这样。修法是结构性的（像 Popover 那样的按钮，`Combobox` 放在它的面板里），交给 M3（第 5、7 节）。
  - **`DateDropdown` 保持两份状态**：组件的 `isOpen` 决定是否渲染日历（`static`）；Headless UI 自己关闭时（按钮上的 Escape、点击外部），经 `ComboDropDown` 新加的 `onClose` prop 让 `isOpen` 跟着关；外层的按键处理保留（日历里的 Escape、Tab 关闭日历）。原因：`Combobox` 没有让组件关闭它的办法，而选中日期之后要关闭；它的主要使用者是 M4 的列表和表格单元格（带懒渲染），改成只有一份状态的结构（例如 Popover）会动到它们，现在核对不了。已知的限制，交给 M4（第 5、7 节）：
    - 键盘打不开日历，Tab 也到不了它的按钮（Headless UI 的按钮固定 `tabIndex: -1`，它处理 Enter、方向键并停止冒泡，只打开自己的一份）。P5 之前也是这样；
    - 两份状态会走散：日历里按 Tab 关闭时，Headless UI 的一份仍是打开的；下一次点击按钮时它的一份关、组件的一份开，日历显示着，这时点另一个下拉框，两个列表同时开着（浏览器中查明）。
- **oxlint**：`CustomSelect`、`CustomSearchSelect` 的 5 条警告随产生它们的代码删除（外层 `div` 的 `jsx_a11y/no-static-element-interactions`、选项的 `jsx_a11y/click-events-have-key-events`、`onClose && onClose()` 的 `no-unused-expressions`），没有禁用注释。`DateDropdown` 外层 `div` 的 `jsx_a11y/no-static-element-interactions` 用带理由的禁用注释：它不是控件，只接收从日历和按钮冒泡上来的按键（第 3 节第 11 条）。上限 ui 25 → 20，web 551 → 550。

### 2.4 PAT 的 service 和 store（M2 设计 7.5、13.2 的约定）

- `ApiTokenService(api: ApiClient)`：`list(cursor?)` 调 `GET /api/v0/me/api-tokens?limit=100[&cursor=…]`，`create(data)` 调 `POST /api/v0/me/api-tokens`，`revoke(tokenId)` 调 `DELETE /api/v0/api-tokens/{token_id}`，都经 `unwrap`。`api` 是 `RootStore` 交给 `UserStore` 的这一代会话的客户端；没有模块级的 service 或带令牌的客户端。
- `ApiTokenStore`（`core/store/user/api-token.store.ts`），`UserStore.apiTokens`：
  - `tokens: ApiToken[] | undefined`：取到之前是 `undefined`，页面据此显示加载中而不是"没有令牌"；列表的每一项只有生成的 `ApiToken` 的六个字段，没有原文；`observable.ref`，整个替换；
  - `fetchTokens()`：一页接一页取（每页 100 个，下一页带上一页的 `next_cursor`），全部取完才一次写入；任何一页失败就抛出，`tokens` 不变；`SessionChangedError`（加载中换了会话）不是失败，返回 `undefined`、不写入（后台加载忽略它）；
  - `createToken(data)`：返回带原文的 `ApiTokenCreated`，只交给调用方；列表加上显式构造的 `ApiToken`（六个字段，不是展开 `created`）；列表还没取过时不动它，否则只有新令牌的列表会被当作完整的列表显示；
  - `revokeToken(tokenId)`：nerve 答 204 之后才从列表去掉；拒绝时抛出，列表不变；换了会话时以 `SessionChangedError` 失败，请求不发出（用户的操作可以提示失败）。
- 工作区 store 下 Plane 的旧 `apiToken`（`store/workspace/api-token.store.ts`，没有代码读它）删除：Nerve 的令牌属于账户，不属于工作区。
- 测试：`api-token.store.test.ts` 5 个，对 `FakeNerve`（它的请求记录多了 `query`，测试核对 `limit` 和 `cursor` 的确切取值）；`store-context.test.ts` 加 1 个，照 P4 的做法证明换会话时 PAT store 的行为（plan Task 2 逐条写明）。

### 2.5 api-tokens 页、`@nerve/services`、Plane 的类型（M2 设计 7.5、7.7）

- **页面**（`settings/profile/content/pages/api-tokens.tsx`）：标题、说明、"添加令牌"；`ApiTokenList`（api-tokens 页和安全页共用）经 `useApiTokens()` 读 store，`useSWR(["API_TOKENS", loginId], fetchTokens)`，不在聚焦时重取，失败不自动重试：失败时显示"令牌没能加载"和"重试"，取数中显示加载中，空列表显示调用方给的空状态。
- **列表的每一项**：名称、说明；有效或已过期（`expired_at` 早于现在）；到期时间或"永不过期"；创建时间；最后使用时间或"从未使用"（8.5：便于认出不认识的令牌）；撤销按钮（`aria-label` "Revoke token"），确认之后调 `revokeToken`，失败按 `errorMessageKey` 提示。
- **创建**：名称（必填，255 个字符之内）、说明、有效期：`EXPIRY_PERIODS`（1 周、1 个月、3 个月、1 年）按日历加在现在上（date-fns 的 `add`）；"自定义日期"在日历里选一天，取那天、现在的时刻；"永不过期"开关（`expired_at: null`）。`expiryDate(choice, now, picked)` 是组件旁的纯函数，有单元测试。表单显示将要到期的时间。nerve 的字段错误（`label`、`expired_at`）显示在字段下，其余按 `needsErrorBanner` 提示。
- **原文只显示一次**：`createToken` 的返回值只放在弹窗组件的状态里；显示一次，同时下载 CSV（Plane 的行为：名称、说明、到期、原文）；关闭 350 毫秒之后清掉（Plane 的定时，等弹窗淡出）；每次打开都先回到表单，在渲染时就重置，重开得再快也不会有一帧显示上一个令牌（第 3 节第 4 条）。原文不进 store、存储、地址和控制台（A11 查七种形式）。
- **`@nerve/services`**：删除 Plane 的 axios 基类 `api.service.ts`（`web/` 下最后一个 `withCredentials: true`）和 `developer/`（旧的令牌服务，`/api/users/api-tokens/`）；只剩地址工具（带单元测试）和上传文件的元数据工具，整个包由 M5 删除。
- **Plane 的类型和常量**：`@nerve/types` 的 `api_token.ts`（`IApiToken`）、`timezone.ts`（`TTimezoneObject`、`TTimezones`，Task 6）删除；`@nerve/constants` 的 `API_TOKENS_LIST` 删除。`@nerve/*` 包声明了 `exports`，knip 把每个导出当作公开的；删掉使用者之后逐个查了它用过的导出，没有别的使用者的一并删除（附录 A 的死代码核对中 `export` 一类只剩 `cec4ec9` 就有的两行）。
- **文案**：`account_settings.api_tokens` 下新增列表、有效期、撤销的键（中英文）；只有旧页面用的 `workspace_settings.settings.api_tokens.title`、`.delete.error.message` 删除（原型核对：没有未使用的键，`cec4ec9` 也是 0 个）。

### 2.6 安全页（M2 设计 7.7、3.5；Codex M-6）

- 修改密码的表单下方一节"个人访问令牌"：标题、说明（修改密码不会撤销这些令牌：每个令牌在到期或在这里撤销之前一直有效）、`ApiTokenList`，没有令牌时一行"你没有个人访问令牌"。修改密码本身和它的错误处理是 P4 做的，不变。
- A7 的页面版本和接口版本调用同一个 `expectPasswordChanged`，只差 `survivingSession`：页面自己的会话延续（Codex M-6）。

### 2.7 preferences（M2 设计 5.3、7.7；前端改动清单 3.2）

- 时区：`TimezoneService.list()` 经 `publicClient` 调公开的 `GET /api/v0/timezones`（与实例信息一样不带令牌），返回生成的 `Timezone[]`；`use-timezone.tsx` 把同一时区的地点合成一个选项，选项的搜索文本含时区名、地点、GMT 和 UTC 偏移。Plane 的地址和 `TTimezoneObject` 删除。
- 主题、语言、时区、每周第一天：成功和失败的提示都经 `t()`；失败的文案按 `code` 取（`errorMessageKey`），不再是写死的英文。主题切换之后刷新页面的行为保留（7.7）。
- 页面反映 nerve 持有的值：主题在 nerve 应答成功之后才应用（随后刷新页面）；被拒绝时页面保持原来的主题、不刷新，提示说明原因。原来先应用、失败不撤回。
- 页面上的其余文案也经 `t()`：每周第一天的标题、说明和星期名（第 3 节第 15 条；命令面板的"更改一周的第一天"菜单同样用这些键，`START_OF_THE_WEEK_OPTIONS` 不再有英文的 `label`），时区列表的搜索框、"没有匹配"和按钮的"选择时区"（"加载中"在这里不会出现：加载时按钮是禁用的），主题的"正在更新""已更新""正在重新加载"（命令面板的主题提示用同样的键）。

### 2.8 general、停用账户、维护页（M2 设计 7.3、7.7；P4 spec 第 3 节第 14 条）

- **general 的保存**：`PATCH /api/v0/me`；成功提示"个人资料已更新"；`first_name`、`last_name`、`display_name` 的字段错误显示在字段下，其余在提示里（`fieldErrorKeys`、`needsErrorBanner`、`errorMessageKey`，P4 的错误文案表）。
- **停用账户**：确认弹窗的文案按 Nerve 的行为改写：所有地方退出，密码和 PAT 都不能用；什么都不删除，也不发邮件；要再用这个账户，请服务器管理员恢复（决策点 3）。成功时会话已由 `deactivateAccount` 结束（P4），登录页接管，显示"你的账户已停用"；失败按 `errorMessageKey` 提示。
- **维护页**（`InstanceWrapper` 读不到实例信息时）：标题"暂时无法连接 Nerve"，说明页面没能读取这台服务器的设置，会自动重试，服务器恢复应答后继续。原来的"Looks like Nerve didn't start up correctly!"在 nerve 只是暂时不可达时有误导。文案进 `common.json`；自动重试是 SWR 的默认行为，浏览器核对 C6 实测放开之后 5.3–10.4 秒自己到登录页。

### 2.9 死成员和死 prop（M2 设计 7.8；M1-closeout 交接）

`domains.mjs --rows M2`（M1 收尾 plan 附录 A 的工具，M1-closeout 交接写了用法）：

| 时点 | 行数 | 构成 |
|---|---|---|
| M1 收尾 | 66 | |
| P5 开始（`cec4ec9`） | 64 | 22 行测试替身和测试里的字面量；1 行 M3 的 prop；41 行死代码 |
| P5 结束 | 24 | 23 行测试替身和测试里的字面量（Task 2 给 `FakeNerve` 加的 `query` 多出 1 行）；1 行 M3 的 prop |

41 行死代码：Task 2 删掉工作区的旧 PAT store（12 行），Task 3 删掉 `@nerve/services` 的 axios 基类（3 行），Task 8 删掉其余 26 行（`settings.store` 的 `isLoading`、`error`、`TError`、`isScrolled`、`toggleIsScrolled`；`issue/profile` 两个 store 的 `userId`、`issueFilterService`、`currentView`、`quickAddIssue`、`appliedFilters`；`IUserPermissionStore` 接口上的 `fetchWorkspaceLevelProjectEntities`；个人设置两张表各写一遍的成员类型；5 个没人传的 prop）。`appliedFilters` 这一行只能连同 `IBaseIssueFilterStore` 上的声明和实现它的八个过滤 store 的 getter 一起删（第 3 节第 5 条）。剩下的 24 行逐行说明见附录 A.4。

### 2.10 oxlint（M2 设计 7.8；P4 spec 第 6 节）

- P5 改到的 116 个 web 文件、P4 改到的全部文件：没有警告。
- `eslint(no-unneeded-ternary)`：`cec4ec9` 有 54 条（web 53、ui 1），全部改为布尔表达式，全仓 0 条。
- 上限（`tools/lint-cap.mjs`，警告数必须正好等于上限）逐个 Task 调低：web 551 → 550（Task 1）→ 548（2）→ 543（3）→ 541（6）→ 537（8）→ 514（9）→ 457（10）；ui 25 → 20（1）→ 19（10）；utils 18 → 16（9）。每一步都是删掉带警告的代码之后的实际数。
- 清警告时的写法：`.then` 链改为 `await` 或返回里面的 promise；可点击的 `div` 改为 `<button type="button">`；带理由的禁用注释只用在两处：`DateDropdown` 外层的按键处理（2.3）和引导的资料步骤的 `autoFocus`（这一步只有这一个字段，打开就聚焦它）。两处清警告改了行为，第 3 节第 6、7 条。

### 2.11 关键词规则（M2 设计 7.9 中 P5 的部分）

| 规则 | 内容 | 在 `cec4ec9` 上的命中 | Task |
|---|---|---|---|
| `with-credentials` | `web/` 下的 `withCredentials:\s*true` | 1：`web/packages/services/src/api.service.ts` | 3 |
| `plane-api-token-urls` | `web/` 下的 `/api/users/api-tokens` | 4：`web/packages/services/src/developer/api-token.service.ts` | 3 |
| `plane-timezone-urls` | `web/` 下的 `/api/timezones/` | 1：`web:core/services/timezone.service.ts` | 6 |
| `services-files`（收窄） | 白名单去掉 `api.service.ts`、`developer/` | 3：这三个文件 | 3 |

规则 57 → 59（Task 3）→ 60（Task 6）；每条规则的 `samples` 各有命中和不命中的例子，守卫启动时自检。上传到对象存储的 `withCredentials: false`（M5）不命中。

### 2.12 端到端（M2 设计 2、9.5）

- **fixture**：
  - `e2e/fixtures/settings-pages.ts`：`registerOnboarded`、`answerTo`、`expectTokenGone`（令牌的七种形式：原文、去掉前缀的密文、两者的十六进制和 base64、密文字节的十六进制和 base64，都不在文档、地址、localStorage、sessionStorage 和页面的控制台输出里）、`expectListBesideButton`（页面上唯一的列表在按钮正下方或正上方 8 像素之内，左边缘或右边缘对齐 2 像素之内）、`submitPasswordChange`；
  - `e2e/fixtures/browser.ts`：`EMOJI_CHECK_WARNING`（第 3 节第 9 条），S2 和个人设置的故事共用。
- **页面版本**（接口版本不变；plan Task 4–7 逐条写明）：
  - **A11**（2 个）：创建、原文显示一次、CSV、关闭后七种形式都不在（再打开弹窗、经侧栏去安全页再回来、刷新之后都再核对）、撤销、数据库和 PAT 的实际效果；自定义日期的日历在按钮旁，Escape 关闭、一次点击再打开。第二个：另一个标签页以另一个账户登录，本标签页的列表换成那个账户的（缺陷类别：只有一个账户）。
  - **A7**：安全页列出 PAT 和说明；当前密码错、新密码太常见各显示在自己的字段下；成功之后页面的会话延续、别的会话结束、PAT 仍可用、旧密码登录 401。
  - **A8**：general 页的三个名字、preferences 页的时区（列表在按钮旁，Escape 之后再打开，搜索、选择，选中之后列表关闭）；刷新之后都在；没有发往旧接口的请求。
  - **A9**：主题（英文下列表在按钮旁；选 Dark，页面自己刷新）、语言（选简体中文，页面立即变中文，每周第一天的标题和按钮随之变中文：Sunday → 星期日）、中文下主题的列表在按钮旁（Escape 之后一次点击再打开）、中文下时区列表的搜索框和"未找到匹配项"、每周第一天：
    - 键盘：从语言的按钮按 Tab 到它的按钮，下箭头打开列表（在按钮旁，选项是中文的星期名；按钮上的 Enter 不打开列表，2.3），再按下箭头、Enter 从星期日改为星期一：`PATCH` 200，列表关闭，焦点回到按钮；空格再打开，Escape 关闭，焦点仍在按钮上；
    - 鼠标点已选中的选项（`aria-selected` 的星期一）：列表关闭，再发一次同样的 `PATCH`（200）；
    - 四次选择各发一个 `PATCH /api/v0/me/profile`，时区的搜索不发请求；数据库和刷新之后都对（按钮显示深色、简体中文、星期一）。
  - **A9 的第二个页面测试**（被拒绝的修改）：`page.route` 让 `PATCH /api/v0/me/profile` 回答 500（`internal_error`）；选 Dark：提示"Something went wrong on the server. Please try again."，页面的主题仍是原来的（`data-theme` 为 light），按钮仍是 System Preference，页面没有刷新；数据库不变；失败的请求和控制台的错误逐条点名（500 的 `PATCH` 一条，浏览器对它的报错一条）。
  - **A12**：确认弹窗的文案；204；回到带 `next_path` 的登录页并提示；浏览器没有记录；数据库里已停用、会话结束、PAT 保留；再登录 403 "This account is deactivated."；`nerve users activate` 之后登录进入引导，PAT 又能用。
- 每个内容断言之前先断言元素或请求存在；控制台只允许点名的消息（每次加载一条 `EMOJI_CHECK_WARNING`，故事造成的 422、403 的浏览器报错）。
- 测试数：35 → 41。新增的等待都有期限（`answerTo` 10 秒，下载 10 秒）。

### 2.13 文档与交接（M2 设计 3.20）

- 前端改动清单：3.1 的 M2 一行改为已完成；`packages/services` 一行写到 P5 删掉的部分；3.2 的错误处理一行写 P5 改完的个人设置。
- README：M2 能用的页面包括个人设置的四个标签页；PAT 泄露时在 api-tokens 页或安全页查看、撤销。
- 交接：M1-P3-trim-platform、M1-closeout 状态改为 `done`，各加"处理结果（M2/P5）"一节（第 7 节）。

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M2 设计的架构和前后端契约。第 5–8 条动到 M3、M4 领域的文件，第 9 条改了 13.2 中点名的一个 e2e 常量，请控制者裁定。

1. **下拉框的根因与语言无关，修在三个组件**：7.7 写"界面语言为 zh-CN 时"，点名 `custom-select.tsx`。根因在 Headless UI 的 `Combobox.Options`（2.3），英文下一样；同样的写法还在 `CustomSearchSelect`（时区）和 `DateDropdown`（自定义有效期），都在 P5 的页面上，一起修。其他 12 处在 M3–M6 的页面上，交给接上它们的 M（第 7 节）。建议 7.7 的描述随之更正。
2. **`CustomSearchSelect` 不设模态**：原来的选项被 Headless UI 设为 inert（2.3），Plane 的时区列表因此本来就点不了（列表在左上角，也没人发现）。只改这一个组件；`CustomSelect` 仍是模态，列表打开时页面其余部分 inert、不能滚动，这是 Headless UI 的默认行为，保留。
3. **下拉框的打开状态**：键盘打不开列表，Escape 之后要点两次才能再打开（`CustomSelect`），Escape 关不掉列表（`CustomSearchSelect`、`DateDropdown`），根子都是组件自己的 `isOpen` 与 Headless UI 的一份不同步。这是 Plane 原有的问题，7.7 没有写。主题下拉框是 7.7 点名要修好的控件，所以在同一个 Task 修（2.3）：`CustomSelect` 改用 `Listbox`，`CustomSearchSelect` 仍用 `Combobox`，两者的打开状态都只有 Headless UI 的一份；`DateDropdown` 保持两份，经 `ComboDropDown` 的 `onClose` 同步。`ComboDropDown` 的其他使用者（M3–M6 的下拉框）交给接上它们的 M（第 7 节）。
    - **控制者的裁定**（Task 1 在浏览器中查明：只跟随关闭的同步之后，点击打开的列表 Escape 仍关不掉，焦点不在任何控件上；`Combobox` 没有输入框时键盘移动不了、选不了选项；组件没有办法关闭 `Combobox`，选中日期之后日历不关）：`CustomSelect` 用 `Listbox`，按钮上的 Enter 提交所在的表单（原生 `select` 的行为）接受；`CustomSearchSelect` 只有一份状态，搜索框在打开时取得焦点；`DateDropdown` 保持 plan 的两份状态，M4 与其余 12 个下拉框一起重构；时区按钮不在 Tab 顺序里记为已知的限制，交给 M3。
4. **创建弹窗每次打开都先回到表单**：Plane 在关闭 350 毫秒之后才清掉原文；在这之前重开，弹窗会先显示上一个令牌。7.7 要求"只显示一次"，所以在渲染时按"刚打开"重置（React 的"属性变化时调整状态"写法），350 毫秒的清理保留（原文只在弹窗显示期间留在内存）。端到端测试控制不了这 350 毫秒，"不重置"的变异由浏览器核对 C3 发现（附录 A.7）。
5. **`appliedFilters` 的删除动到 8 个 M4 领域的过滤 store**：M2 的一行是 `issue/profile/filter.store.ts` 的 `appliedFilters`，它实现 `IBaseIssueFilterStore` 的声明，单删这一个类型检查不过。整个仓库没有代码读任何一个 `appliedFilters`，所以连同接口和其余 8 个 getter 一起删（Task 8）。
6. **周期下拉框的取数**（`dropdowns/cycle/cycle-options.tsx`，M4 领域）：清 `react-hooks/exhaustive-deps` 时发现原来的 `onOpen` 判断 `!cycleIds`，而 `cycleIds` 总是数组，于是从不取周期。改为打开时 `getProjectCycleIds(projectId)` 为空才取。这改了行为（没取过时会取），M4 接上时再核对。
7. **`Number(role) ?? GUEST`**（项目成员表，M3 领域）：`??` 永远不生效（`Number` 不返回空值），没有角色时得到 `NaN`；改为 `Number(role ?? GUEST)`，没有角色时按访客算。这是 `no-constant-binary-expression` 指出的缺陷。
8. **邀请页的 `fetchWorkspaces()` 拒绝**：P4 评审交给 M3 的一项（"没有返回它的 Promise，也没有处理拒绝"）。清 `promise/always-return` 时把它返回给外层的 `.catch`，失败时显示"出错了"。M3 对接邀请时不需要再改这一处；请控制者把 P4 评审的这一项记为在 P5 完成。
9. **`EMOJI_CHECK_WARNING`**：个人设置的每次加载都有 Chromium 的 `willReadFrequently` 提示，来源与 S2 的深链接相同（编辑器的 Emoji 节点让 `is-emoji-supported` 反复读画布），但路径不同：设置页的布局挂着命令面板（`ProjectsAppPowerKProvider`，M1 保留），它建了一个编辑器（原型用调用栈确认，附录 A 的 E3）。S2 原来有自己的 `emojiCanvasWarning`；两处共用一个常量，移到 `e2e/fixtures/browser.ts`，名为 `EMOJI_CHECK_WARNING`。M2 设计 13.2 交给 M4 的一行写的是 `emojiCanvasWarning`，请随之更正，并补上：设置页的命令面板在打开之前就建编辑器，M4 做编辑器的代码分割时一并处理，之后这个常量删除。
10. **general 页的保存、preferences 的失败提示在 P5 做**：7.7 没有逐项写，前端改动清单 3.2 写明"个人设置的其余部分（general 页的保存、preferences、安全页的 PAT 列表、api-tokens）在 P5"。按 P4 的错误文案表做（2.7、2.8）。
11. **"改到的文件清零"中的禁用注释**：7.8 要求改到的文件没有警告。`DateDropdown` 外层的按键处理（1 条）和引导的资料步骤（1 条）用带理由的 `oxlint-disable-next-line`，其余都改了代码（2.10）。
    - **控制者的裁定**：禁用注释只用在规则不适用、又没有代码改动能消除需要的地方。
    - 引导资料步骤的 `autoFocus` 接受。
    - 下拉框：`CustomSelect`、`CustomSearchSelect` 的打开状态只有 Headless UI 的一份（第 3 条），外层的按键处理和选项的点击处理随之删除，原来的 5 条警告没有了，不用禁用注释。`DateDropdown` 保持两份状态（第 3 条），外层 `div` 仍要接收日历里冒泡上来的 Escape、Tab，这一条禁用注释和理由保留。Task 9 没有下拉框的工作。

以下四条是控制者在执行前加的。它们原是第 5 节的"不改"和第 6 节的一个风险，都在 P5 负责的代码上，所以从根上修，落在拥有那个文件的 Task：

12. **PAT 列表的竞争**（Task 2）：取列表的请求在途中时创建或撤销完成，取回的旧列表不能覆盖它们：否则新令牌不显示，或者撤销了的令牌又出现，直到下一次取数。store 选择重取或合并，并说明理由。单元测试强制顺序：扣住列表的 GET，其间创建或撤销，再用旧的一页回答。另加"整个覆盖"的变异。
13. **CSV 按 RFC 4180 加引号**（Task 3）：名称或说明含逗号、引号、换行时，Plane 的 `csvDownload` 会错列。CSV 文本放进一个纯函数，写单元测试，`csvDownload` 调用它。PAT 创建弹窗和 webhook 的两个组件都用它，一起受益。
14. **撤销确认弹窗的按钮文案**（Task 3）：`AlertModalCore` 本来就接受 `primaryButtonText`、`secondaryButtonText`，P5 的调用处传入 `t()` 的文案，P5 的页面上不留写死的英文。
15. **每周第一天的标题和星期名**（Task 6）：按 `@nerve/constants` 现有的 i18n 键写法翻译（中英文），preferences 页在中文界面下不再是英文。

## 4. 验收标准（完成线，M2 设计 12 节 P5）

- [ ] 全部故事的页面版本和接口版本通过；此前的故事仍然通过（`make e2e` 共 41 个测试）。
- [ ] `domains.mjs --rows M2` 剩 24 行，每一行在附录 A.4 写明是误报，review 照录。
- [ ] `grep -rnE "withCredentials:\s*true|/api/users/api-tokens|/api/timezones/" web`（排除 `node_modules`、`dist`、`build`、`.turbo`）没有输出。
- [ ] P4、P5 改到的文件没有 oxlint 警告；`no-unneeded-ternary` 全仓 0 条；上限 web 457、ui 19、utils 16。
- [ ] web 单元测试 137 个通过（`--testTimeout=1000` 也通过）；PAT store 的 5 个、换会话的 1 个、有效期的 2 个是新的。
- [ ] 9.6 中 P5 的浏览器核对在合并前的代码上重跑（脚本对任意构建运行，见附录 A.6），写进 review。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20 中 P5 的一行、README 在同一次合并中写好；两份交接按第 7 节处理。
- [ ] e2e 任务的耗时写进 review（9.5、§16；附录 A.8 是本机的数）。

## 5. 不在 P5 范围内

- **收尾**：10 份交接逐项写结论；上级文档和差异清单的总核对；给 M3–M8 的交接（13.2）。
- **M3–M6**：同样写法的其他 12 个下拉框的定位和打开状态（第 7 节列出）；它们所在页面的其余对接。
- **M3**：`CustomSearchSelect` 的按钮不在 Tab 顺序里（2.3；P5 之前也是这样），修法是结构性的。
- **M4**：编辑器的代码分割，以及设置页的命令面板提前建编辑器（第 3 节第 9 条）。`DateDropdown` 改成只有一份打开状态的结构（2.3：键盘打不开日历，两份状态会走散）；`CustomSearchSelect` 的 `defaultOpen` 在接上富筛选时在浏览器中核对。
- **M5**：`@nerve/services` 剩下的文件工具和整个包；头像、封面的上传。
- 原来列在这里"不改"的三项（每周第一天的英文、撤销确认弹窗的英文按钮、CSV 不加引号），按控制者的裁定在 P5 修（第 3 节第 13–15 条）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 创建弹窗关闭后 350 毫秒内重开 | 在渲染时重置（第 3 节第 4 条）；端到端测试控制不了这段时间，由浏览器核对 C3 覆盖（附录 A.7 的"不重置"变异被 C3 发现，一次运行，重开在关闭后约 350 毫秒；时间更长时这个变异看不出来） |
| 列表加载中创建或撤销令牌 | 控制者裁定在 P5 修（第 3 节第 12 条）：取回的旧列表不覆盖途中完成的创建或撤销 |
| Headless UI 升级改变 `Combobox.Options` 的克隆或 inert 的规则 | A8、A9、A11 核对三个下拉框的位置、打开和关闭；变异"改回子元素上的 ref""恢复模态""不跟随关闭"都让它们失败 |
| 模态的 `CustomSelect` 打开时页面其余部分 inert | Headless UI 的默认；故事用 Escape 关闭，不点页面的其他地方 |
| `EMOJI_CHECK_WARNING` 是按原文写的第三方警告 | 依赖升级改了它时，更新 fixture；M4 去掉它（第 3 节第 9 条） |
| 持续集成（ubuntu）上的运行和耗时 | 原型只在 macOS 上运行；合并后看持续集成，写进 review |

## 7. 交接的处理

| 交接 | P5 处理的条目 | 留下的条目 | 状态 |
|---|---|---|---|
| M1-P3-trim-platform | `@nerve/services` 的 API 令牌旧地址：令牌服务和 axios 基类删除，PAT 经 store 用生成的客户端（Task 2、3）；关键词规则 `plane-api-token-urls`、`with-credentials` | — | done |
| M1-closeout | 死成员和死 prop（64 → 24，剩下的是误报，附录 A.4）；oxlint 在改到的文件中清零、`no-unneeded-ternary` 清零、上限调低（Task 8–10）；主题下拉框（Task 1，第 3 节第 1–3 条） | — | done |

评审和设计交给 P5 的事项：

| 事项 | 落在 |
|---|---|
| Codex M-6：A7 的页面版和 PAT 版的会话预期不同 | 2.6；A7 的页面版本（Task 5） |
| P4 评审第 6 节交给 P5 的全部（P4 spec 第 5 节） | 本 spec 第 1 节逐项；维护页的文案（2.8） |
| 13.2 的 stores 约定（service 从构造函数拿这一代的客户端、不经模块级客户端、SWR 键带 `loginId`） | 2.4；`store-context.test.ts` 的新测试；变异"一个 store 给所有会话""service 用第一个会话的客户端""客户端跟随标签页""SWR 键不带 `loginId`"都被发现（附录 A.7） |
| §16：e2e 的耗时 | 附录 A.8 |
| P4 评审交给 M3 的邀请页 `fetchWorkspaces()` | 在 P5 完成（第 3 节第 8 条） |

交给后续的事项（建议收尾写进 13.2）：

| 交给 | 事项 |
|---|---|
| M3–M6（接上组件的 M） | popper 的 ref 放在 `Combobox.Options` 唯一子元素上的其余 12 处（列表会停在页面左上角）：`web:core/components/dropdowns/{cycle/cycle-options,date-range,intake-state/base,member/member-options,module/module-options,priority,project/base,state/base}.tsx`、`web:core/components/issues/{issue-detail/label/select/label-select,issue-layouts/properties/label-dropdown,select/base}.tsx`、`web/packages/ui/src/dropdown/single-select.tsx`。它们的打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox` 并在列表打开时让输入框取得焦点，同 P5 的 `CustomSelect`、`CustomSearchSelect`；只跟随 Headless UI 的关闭（`onClose`）不够（2.3）。修法见 2.3 |
| M3 | `CustomSearchSelect` 的按钮不在 Tab 顺序里（Headless UI 的 `Combobox.Button` 固定 `tabIndex: -1`；P5 之前也是这样）：改成像 Popover 那样的按钮，`Combobox` 放在它的面板里（2.3）。M3 是第一个接上更多 `CustomSearchSelect` 的 M |
| M4 | 编辑器的代码分割，连同设置页的命令面板提前建编辑器；之后删除 `EMOJI_CHECK_WARNING`（第 3 节第 9 条） |
| M4 | `DateDropdown` 的两份打开状态（2.3）：键盘打不开日历；日历里按 Tab 关闭之后两份状态走散，再点另一个下拉框时两个列表同时开着。改成只有一份状态的结构（例如 Popover，选中日期时调它的 `close`），连同它在列表和表格单元格里的懒渲染（`ComboDropDown` 的 `renderByDefault`）。`CustomSearchSelect` 的 `defaultOpen`（挂载之后点一次按钮）没有在浏览器中核对，接上富筛选时核对 |

## 附录 A：原型验证记录（2026-09-27）

原型在 `$M2TMP/p5proto`（`cec4ec9` 的副本；Node 24.15.0、pnpm 11.10.0、Go 1.27.1（`toolchain`）、Docker 29.7.2、Playwright 1.63.0 的 Chromium）。plan 中的代码就是原型中运行过的代码，由脚本（`$M2TMP/p5tools/plan/assemble.mjs`）从原型文件原样拼入 plan：新文件和改动大的文件给完整内容，改动小的给对前一个版本的统一差异；拼接脚本用 `patch -p1` 把 113 个差异块逐个应用到前一个版本上，结果都与原型的文件逐字节相同。另一个脚本核对每个 Task 的文件快照：`cec4ec9` 与原型之间的每一处差异（130 个改动或新增的文件、6 个删除的文件）都由某个 Task 写出或删除。

### A.1 逐 Task 复现

在 `$M2TMP/p5stage`（`cec4ec9` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 11 个 Task 依次执行（`$M2TMP/p5tools/plan/replay-all.sh`）：放入这个 Task 写的文件（最终版本或过渡版本，即 plan 中的内容），删除它删除的文件；执行 plan 中的命令（Task 3 从前一个 lock 出发的 `pnpm install`，得到的 lock 与 plan 逐字节相同；Task 2、3、6、8、11 的 `grep` 核对，都没有多余的输出；Task 2 的 `vitest run --testTimeout=1000`；Task 8 的 `domains.mjs --rows M2`，与原型的 24 行相同；Task 10 的 `no-unneeded-ternary` 行数，0）；核对这个 Task 写的文件没有新的 `&lt;`、`&gt;`、`&amp;`；然后关键词守卫、前端检查、`make knip`、web 单元测试；Task 4–7、11 跑 `make e2e`；Task 11 另跑 `make lint-go` 和 `make test`。每个 Task 19–52 秒。11 个 Task 之后，复现的目录与原型逐文件相同（2636 个文件，不含依赖、构建产物和运行日志）。

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | `make gen`（副本不是 git 仓库，`make gen-check` 用不了） | 原型上重新生成的 `api/dist/openapi.yaml`、Go 接口层、`schema.gen.ts` 与 `cec4ec9` 的逐字节相同：P5 不改生成物 |
| 前端检查 | `turbo run check:types check:lint check:format check:sync`（54 个任务）；关键词守卫（真实的 `tools/keywords.mjs`，`git ls-files` 由遍历目录的替身代替）；`make knip`；`turbo run test`（16 个任务） | 每个 Task 都通过；oxlint 警告数每一步都等于上限（2.10）；关键词规则 57、59（Task 3）、60（Task 6）条，没有命中 |
| web 单元测试 | vitest | web 129 → 137 个（13 个文件）；`--testTimeout=1000` 下各包都通过（web 137、constants 11、editor 35、i18n 10、services 11、utils 43） |
| Go | `make lint-go`；`make test` | 0 issues ×2；32 个包全部通过（P5 不改 Go） |
| 端到端 | `make e2e`（Task 4、5、6、7、11） | 37、38、40、41、41 个测试中除 S3 外全部通过。S3 失败的原因与 P1–P4 相同：副本不是 git 仓库，`commit` 是 `unknown` |
| 页面故事重复 | `make build`，然后 `playwright test` A7、A8、A9、A11、A12、S2 `--grep "page\|S2" --repeat-each=4` | 36 个全部通过（12.6 秒） |
| 完成线 | `grep -rnE "withCredentials:\s*true\|/api/users/api-tokens\|/api/timezones/" web`（排除 `node_modules`、`dist`、`build`、`.turbo`、`.react-router`） | 没有输出 |
| 新规则在 `cec4ec9` 上 | 用原型的 `tools/keywords.json` 在 `cec4ec9` 上跑新的三条规则和收窄的 `services-files`（`$M2TMP/p5tools/kwbase.mjs`） | 分别 1、4、1、3 个命中（2.11） |
| 未使用的翻译键 | `$M2TMP/p5tools/i18nunused.mjs`（英文的每个键是否被 `web/` 的源码整键引用，或经模板前缀引用） | `cec4ec9` 和原型都是 0 个 |

### A.2 原型中定下的事实

- **E1** Headless UI 2.2.10 的 `internal/frozen.js`：`Frozen` 渲染时 `cloneElement(child, { ref })`，用自己的 ref 换掉子元素的 ref；`Combobox.Options` 用它包住唯一的子元素。在 `$M2TMP/p5scratch/frozen/`（只在这个目录里装了 jsdom，不进仓库）用 React 19.2.8 渲染：唯一子元素上的 ref 回调从未收到元素；子元素外再包一层时，里层的 ref 收到元素；`Combobox.Options` 自己的 ref 收到 `UL`。浏览器中（`$M2TMP/p5checks/explore-theme*.mjs`，`P=$M2TMP/p5base` 即 `cec4ec9` 的构建）修复之前主题列表的 `style` 是 `position: absolute; left: 0px; top: 0px`，`getBoundingClientRect()` 是 (0, 4, 192 × 180.5)，中英文一样；同一次运行中，Escape 之后的下一次点击在中英文下都没有打开列表（E4）；修复之后列表的 `data-popper-placement` 是 `bottom-end`，在按钮下方 4 像素。
- **E2** 同一目录的 `inert.mjs`：`Combobox` 默认 `modal` 为真，输入框在列表里时，被设为 `inert` 的正是放选项的元素；`modal={false}` 时没有元素被设为 `inert`。
- **E3** 设置页的 Canvas2D 提示（`$M2TMP/p5checks/explore-canvas.mjs`，经 CDP 的 `Log.entryAdded`）：四个标签页的每次加载各一条，文字与 S2 深链接的那一条相同，调用栈在 `is-emoji-supported` 里；设置页的布局挂着 `ProjectsAppPowerKProvider`，它建了编辑器。
- **E4** 时区列表的 Escape（`$M2TMP/p5checks/explore-escape.mjs`，修复打开状态之前）：列表打开后焦点在按钮上；第一次 Escape 之后列表还在，焦点移到搜索框；第二次才关上。主题列表 Escape 之后，下一次点击什么也不打开（浏览器核对 C2 的第一版发现）。
- **E5** 创建弹窗（修复之前）：关闭后 350 毫秒之内重开，弹窗先显示上一个令牌（浏览器核对 C3 的 `MutationObserver` 记下原文在关闭之后又出现一次，见 A.7）。
- **E6** `make e2e` 的耗时（A.8）。

### A.3 oxlint

`$M2TMP/p5tools/lintrows.mjs` 对每个 web 包跑 oxlint（与 `check:lint` 同一份配置），逐行列出警告：

| 范围 | `cec4ec9` | 原型 |
|---|---|---|
| P5 改到的 116 个 web 文件（`$M2TMP/p5tools/p5-changed-web.txt`，由 Task 快照得出） | — | 0 |
| P4 改到的 web 文件（`$M2TMP/p5tools/p4-changed.txt`，P4 spec 第 6 节说其中 6 个还有 12 条） | 12 | 0 |
| `eslint(no-unneeded-ternary)`，全仓 | 54（web 53、ui 1） | 0 |
| 各包合计 | web 551、ui 25、utils 18、editor 65、propel 16、hooks 3、constants 1、i18n 1（共 680） | web 457、ui 19、utils 16，其余不变（共 578） |

Task 8–10 改的文件（`$M2TMP/p5tools/t8-10files.txt`）在 `cec4ec9` 上共 87 条：`no-unneeded-ternary` 54 条；其余 33 条是 `jsx_a11y` 的 `click-events-have-key-events`、`no-static-element-interactions`、`prefer-tag-over-role` 9 条，`promise/always-return` 8 条，`react-hooks/exhaustive-deps`、`no-shadow`、`no-constant-binary-expression` 各 3 条，`no-useless-escape`、`jsx_a11y/img-redundant-alt` 各 2 条，`no-unused-expressions`、`no-empty-pattern`、`jsx_a11y/no-autofocus` 各 1 条。分到 Task：Task 8 4 条，Task 9 25 条，Task 10 58 条。Task 1–3、6 改的文件另有 15 条（web 8、ui 5，另 2 条随删除的旧 PAT store 去掉）。

### A.4 `domains.mjs --rows M2` 剩下的 24 行

`$M2TMP/p5tools/dead.sh`：`react-router typegen`，M1 收尾 plan 附录 A 的 `deadsym.mjs --tsv`（`git ls-files` 由遍历目录的替身代替），`domains.mjs --rows M2`。`export` 一类在 `cec4ec9` 和原型上都只有 `web/packages/api-client/test/client.typecheck.ts` 的两行（类型检查用的测试函数），P5 没有留下没人用的包导出。

| 行 | 为什么不是死代码 |
|---|---|
| `core/lib/auth/fake-browser.ts:24 tab`、`:42 write`、`:47 hold`、`:52 deliver`、`:82 held`（5 行） | 测试替身的成员，只由测试读取（`token-manager.tabs.test.ts`、`refresh-lock.test.ts`），`deadsym` 不计测试文件的读取 |
| `core/lib/auth/fake-nerve.ts:14 method`、`:17 query`、`:18 authorization`、`:19 body`、`:21 at`、`:23 aborted`、`:24 answer`、`:26 fail`、`:58 client`、`:63 to`、`:68 tokens`（11 行） | 假 nerve 的请求记录和方法，只由测试读取；`query` 是 P5 加的，`api-token.store.test.ts`、`store-context.test.ts` 核对它 |
| `core/lib/auth/fake-time.ts:21 value`、`:21 error`（2 行） | `track()` 的结果，只由测试读取 |
| `core/lib/auth/token-manager.test.ts:18 refresh_token`、`:18 login_id`、`:25 key`、`:25 held`、`:404 login_id`（5 行） | 测试里的字面量：记录由 `JSON.stringify` 读取，写入记录的 `key`、`held` 由断言读取 |
| `core/components/onboarding/root.tsx:22 invitations`（prop） | M3 加回邀请的取数时传入（M2 设计 3.1、13.2；P4 spec 2.11） |

### A.5 下拉框：修复前后

| 列表 | `cec4ec9` | 原型 |
|---|---|---|
| 主题（`CustomSelect`，中英文） | 在页面左上角 (0, 4) | 按钮下方 4 像素，右边缘对齐（C2） |
| 语言、每周第一天（`CustomSelect`） | 左上角 | 按钮下方或上方（A9、C2） |
| 时区（`CustomSearchSelect`） | 左上角，选项 inert、点不了 | 按钮下方，右边缘对齐，选项可点（A8、C2） |
| 自定义有效期（`DateDropdown`） | 左上角 | 按钮下方 4 像素，左边缘对齐（A11、C3） |
| Escape 之后再打开 | 主题要点两次；时区、日历 Escape 关不掉 | 一次 Escape 关闭，一次点击打开（A8、A9、A11、C2） |

### A.6 9.6 的浏览器核对

`make build`，然后 `node $M2TMP/p5checks/checks.mjs`（`lib.mjs` 在同一目录，由 P4 的核对脚本改来；不进仓库，review 附录按 9.6 收录执行时重跑的脚本全文和结果）。环境变量 `P` 指定要核对的构建（默认原型），`OUT` 指定截图和日志的目录；可以只跑列出的核对（`node checks.mjs C3-token-once`）。每次运行在开发库 `nerve-dev-db-1` 上建一个独立的库、结束时删除，启动一个 `bin/nerve`（随机端口）。结果是最后一次完整运行（全部通过，6 个）。

| 核对 | 9.6 的条目 | 做法 | 结果 |
|---|---|---|---|
| C1 | 个人设置的四个标签页 | 从侧栏依次点 Profile、Preferences、Security、Personal Access Tokens | 通过：地址依次是四个标签页，各自的标志性控件出现（名字输入框、主题按钮、"Change password"、"Add access token"），安全页有 PAT 一节；没有发往旧接口的请求、失败的请求、页面错误 |
| C2 | 主题下拉框在按钮旁（zh-CN、en） | 主题设为深色；英文下依次打开主题、每周第一天、时区；用页面把语言换成简体中文；中文下再打开主题、时区、每周第一天；每个列表 Escape 关闭，再点一次 | 通过：6 个列表都在按钮下方 4 像素，边缘对齐（0 像素）；每个一次 Escape 关闭，一次点击再打开；语言 `PATCH` 200，页面变中文。另记：每周第一天的标题和星期名在中文下仍是英文（第 5 节） |
| C3 | PAT 创建后只显示一次、CSV 下载 | 名称、说明；自定义日期的日历（位置）、选一天；生成；读下载的 CSV；装上 `MutationObserver` 记下原文在文档中的出现；关闭，立即重开 | 通过：201；日历在按钮下方 4 像素，左边缘对齐；原文显示；CSV 表头 `Title,Description,Expiry,Secret key`，含原文；关闭后约 346 毫秒重开，是空表单，原文再没出现（0 次）；文档、地址、两种存储、控制台都不含密文；列表有"deploy"和"Never used" |
| C4 | 安全页的字段错误和 PAT 列表 | 经接口建一个 PAT"cli"；当前密码错；新密码太常见；在安全页撤销"cli" | 通过：列出"cli"；两次 422，提示分别在当前密码、新密码字段下方 16 像素；撤销 204，显示"You have no personal access tokens."，这个 PAT 再用得到 401；失败的请求只有两个 422 |
| C5 | 停用账户 | general 页的"Deactivate account"；确认；登录；`nerve users activate`；再登录 | 通过：弹窗有"ask an administrator of this server to reactivate it"；204；到 `/?next_path=%2Fsettings%2Fprofile%2Fgeneral`，提示"Your account is deactivated."，浏览器没有记录；登录 403，"This account is deactivated."；`activated …: 0 API tokens are usable again`；再登录 200，进入 `/onboarding` |
| C6 | 维护页（P5 改的文案） | 实例请求答 503；打开首页；放开 | 通过：标题"Nerve cannot be reached right now"和说明；放开后 5.3–10.4 秒（SWR 的退避，几次运行）自己到登录页 |

C3 另对"创建弹窗打开时不重置"的构建跑过一次（`$M2TMP/p5tools/mutcheck.py`：改代码、`make build`、跑 C3、恢复、再构建）：原文在关闭之后又出现 1 次，C3 失败（A.7）。

### A.7 变异核对

`$M2TMP/p5tools/m_web.py`（`mutweb.py`：改一处代码，跑相关的单元测试，再恢复）和 `m_e2e.py`（`mute2e.py`：改代码，`make build`，跑相关的故事，再恢复），由 `mutall.sh` 依次运行；`mutcheck.py` 对故事发现不了的一个跑浏览器核对。共 33 个变异，按缺陷类别归类，每个按最后一次运行计：32 个被单元测试或故事发现，1 个只被浏览器核对发现。

| Task | A | C | D | F | K | 合计 |
|---|---|---|---|---|---|---|
| 1 下拉框 | 7 | | | | | 7 |
| 2 PAT store | 6 | 2 | 5 | 1 | | 14 |
| 3 api-tokens 页（故事在 Task 4） | 2 | | 1 | 6 | | 9 |
| 5 安全页 | | | | | 1 | 1 |
| 6 时区 | | | | | 1 | 1 |
| 7 停用账户 | 1 | | | | | 1 |
| 合计 | 16 | 2 | 6 | 7 | 2 | 33 |

类别：A 属性去掉了测试仍通过；C 假实现忽略参数；D 只有一个标签页或一个账户，看不出跨标签页、跨账户的问题；F 令牌出现在存储、地址、控制台、store 或再次打开的弹窗中；K 接线没人看。页面上的变异按它改的产品代码所属的 Task 计。

brief 要求的六个：

| 变异 | 结果 |
|---|---|
| 创建后原文也留下：进 store 的列表（`[created, …]`）/ sessionStorage / localStorage（base64）/ 地址（`?token=`） | `api-token.store.test.ts` 的"gives the token itself to the one who created it…"失败 / A11 的第一个页面测试失败（`expectTokenGone`：`sessionStorage holds …`、`localStorage holds <base64>`、`address holds …`） |
| 换会话之后列表是另一个账户的：一个 PAT store 给所有会话 / SWR 键不带 `loginId` | `store-context.test.ts` 的新测试失败，A11 的第二个页面测试失败（Y 的列表不出现）/ A11 的第二个页面测试失败（新 store 一直是加载中） |
| PAT store 不用这一代的客户端：service 用第一个会话的客户端 / 客户端跟随标签页此刻的会话 | `store-context.test.ts` 的新测试失败（Y 的列表取不到 / X 的第二页以 Y 的令牌发出） |
| 时区列表读 Plane 的地址（`fetch("/api/timezones/")`） | A8 的页面测试失败（没有"Beijing"；`oldApiRequests` 不空）；这一行也是 `plane-timezone-urls` 的内容 |
| 主题下拉框离开按钮（`custom-select.tsx` 换回 `cec4ec9` 的） | A9 的页面测试失败（`expectListBesideButton`） |
| 停用不结束会话（删掉 `deactivateAccount` 里的 `endSession`） | A12 的页面测试失败（页面停在 general） |

其余：

| 变异 | 结果 |
|---|---|
| 只取第一页 / 边取边显示 / 吞掉失败 / `SessionChangedError` 当作失败 / 不带 `limit` / 不带 `cursor` / 没取过的列表在创建时生成 / 撤销在答复之前改列表 / 撤销去掉的是第一个 | `api-token.store.test.ts` 或 `store-context.test.ts` 的对应测试失败 |
| 有效期按天数加（1 个月 = 30 天）/ 自定义日期取午夜 | `expiry.test.ts` 失败 |
| 控制台打出原文 / 关闭后不清掉原文（两处重置都去掉） | A11 的第一个页面测试失败 |
| 打开时不重置（只留 350 毫秒的清理） | 故事发现不了（A11 重开在 350 毫秒之后）；浏览器核对 C3 发现（A.6） |
| `CustomSearchSelect`、`DateDropdown` 换回 `cec4ec9` 的 / `CustomSearchSelect` 恢复模态 | A8 / A11 / A8 失败 |
| 三个下拉框不跟随 Headless UI 的关闭（去掉 `onClose`） | A9 / A8 / A11 失败 |
| 安全页不列出 PAT | A7 的页面测试失败 |

**原型中最初没被发现、改了测试或代码之后才被发现的**：

- **撤销去掉的是第一个**：测试的列表只有两个令牌、撤销的正好是第一个；改为三个令牌、撤销中间的一个。
- **有效期按天数加**：测试的"现在"是 9 月 27 日，加 30 天与加 1 个月相同；改为 10 月 27 日（10 月 31 天）。
- **关闭后不清掉原文**：第一稿的变异只去掉 350 毫秒的清理，A11 不重开弹窗，没发现；A11 加上"再打开是空表单"，代码加上打开时的重置（第 3 节第 4 条），变异改为两处都去掉之后被发现。
- **`DateDropdown` 换回 `cec4ec9` 的**：第一稿的 A11 只选 1 周，没打开日历；加上自定义日期的日历之后被发现。
- **打开状态不跟随 Headless UI**：浏览器核对 C2 的第一版在中文下点主题按钮，列表没打开（之前在英文下按过 Escape），由此查到 E4；修复之后 A8、A9、A11 加上"Escape 之后一次点击再打开"，三个变异都被发现。

**缺陷类别**（brief 列出的类别，对本 plan 逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试不失败（A） | store 的每条规则（逐页、一次写入、失败不写、创建不带原文、没取过不生成、答复之后才撤销）；有效期的日历算法；三个下拉框的位置和打开状态；停用结束会话 | 16 个 A 类变异都被发现；发现并修正四处（撤销、有效期、日历、打开状态） |
| 断言不可能失败 | "没有发往旧接口的请求"（A8、A9）先断言页面确实取了时区、确实发了 `PATCH`；`expectTokenGone` 先断言原文在关闭之前确实显示过、CSV 确实含原文；控制台的检查先写探针（P4 的 `expectQuietConsole`） | 时区读旧地址的变异让 A8 失败，证明断言是活的 |
| 假实现忽略参数（C） | `FakeNerve` 记下查询参数，测试核对 `{ limit: "100" }`、`{ limit: "100", cursor: "c-1" }` 的确切取值 | 2 个 C 类变异都被发现 |
| 一个标签页或一个账户（D） | `store-context.test.ts` 的新测试两个账户、会话切换在 X 的加载中途；A11 的第二个页面测试两个标签页、两个账户，各有自己的令牌 | 6 个 D 类变异都被发现 |
| 秘密的编码形式（F） | A11 查七种形式在文档、地址、两种存储、控制台，关闭后、换页回来、刷新之后；store 的列表逐字段相等并查密文；C3 用 `MutationObserver` 查重开时 | 7 个 F 类变异，6 个被故事或单元测试发现，1 个被 C3 发现 |
| 测试挂住 | 单元测试在假时钟上，`until` 有期限；故事的等待都有期限；浏览器核对每一步有期限 | 没有变异挂住 |
| 闸门在争用区段之外 | `store-context.test.ts` 在 X 的第一页请求停在假 nerve 里时切换会话，再放行答复 | "`SessionChangedError` 当作失败""客户端跟随标签页"都被发现 |
| 说明与代码不符 | 停用弹窗、维护页、安全页的说明文字、README、交接的处理结果、代码注释 | 逐句与代码核对；停用弹窗的每一句由 A12 和 C5 的实际效果核对（会话结束、PAT 恢复、没有删除） |
| 接线没人看（K） | 安全页挂上列表；时区的地址；`UserStore` 挂上 PAT store；`useApiTokens` | 2 个 K 类变异都被发现；`UserStore` 的接线由 D 类的变异覆盖 |
| 页面没有渲染被检查的东西 | 每个内容断言之前先断言元素或请求存在（列表项、"Never used"、按钮、`answerTo` 的回答） | 由上面各类的页面变异间接证明：断言都曾因变异而失败 |
| 删掉使用者后没人用的包导出 | `deadsym` 的 `export` 一类 | 与 `cec4ec9` 相同，只有两行测试函数（A.4） |
| 换会话之后以错误的身份写入 | PAT store 的 service 从构造函数拿这一代的客户端；`store-context.test.ts` 核对旧会话的撤销不发出 | "客户端跟随标签页"的变异让 X 的撤销以 Y 的令牌发出，被发现 |

### A.8 端到端的耗时（9.5、§16）

本机（Apple 芯片，macOS），原型的最终状态，`TURBO_FORCE=true make e2e`（前端照持续集成那样重新构建，8.7 秒）：整个命令 25 秒；Playwright 41 个测试 13.7 秒（M2 的 17 个故事文件都在其中）。持续集成的 e2e 任务另有安装依赖、构建 Go 和浏览器的时间，M0 时约 120 秒；合并后看持续集成的实际耗时，写进 review。

### A.9 没有证明的

- 持续集成（ubuntu）上的运行和耗时；S3 在 git 仓库中的结果。合并后看持续集成。
- 创建弹窗在关闭后 350 毫秒之内重开：C3 发现"不重置"的变异只跑过一次，重开发生在约 350 毫秒；时间更长时这个变异看不出来（第 6 节）。
- 其余 12 个同样写法的下拉框没有在浏览器中核对（它们在 M3–M6 的页面上）。
