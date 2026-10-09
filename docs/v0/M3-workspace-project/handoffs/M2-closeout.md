---
status: open
from: M2/closeout
to: M3
created: 2026-09-28
---

# M2 交给 M3：邀请与注册、登录后的落点、停用的端口、stores 按会话分代，以及 M2 改到却走不到的页面

M2（账户认证）做完了注册、登录、续期、退出、个人设置、PAT 和 `nerve users` 命令，前端有了令牌管理器和按会话分代的 stores（[M2 设计](../../M2-auth/M2-design.md)）。M2 没有工作区的接口，下面这些由 M3 接着做。每一节来自 M2 设计 13.2 中接收者含 M3 的一行（多个 M 的行只取 M3 的部分）；收尾重扫 M2 设计时另找到几项，放进相应的节：第 2 节的 `/api/users/me/settings/` 和挂载时的规则是否往后延伸，第 6 节的加锁顺序，第 8 节的 `Authorizer` 端口，第 11 节的 `owner` 类型和守卫的收紧。为什么这样定，看链接的节，这里写做什么、代码在哪、M2 做到哪一步、怎样算完。

## 1. 邀请：凭链接中的令牌接受；关闭注册时持有邀请的人仍可注册（交负责人确认）

- **做什么**：接受工作区邀请时，不能只靠邮箱匹配，要凭邀请链接中的令牌（M1 设计 3.15 留下的路径）；注册关闭时（prod 的默认，M2 设计决策点 2），持有有效邀请的人仍可注册。
- **为什么**：v0 没有邮件服务，邮箱未经验证。只靠邮箱匹配，谁先用别人的邮箱注册，谁就能接受发给那个邮箱的邀请。Plane 把任何未删除的工作区邀请都算上（`plane/apps/api/plane/authentication/adapter/base.py:102-120`）。prod 默认关闭注册不能代替接受邀请时的身份证明。
- **这是产品改动**：它改变总体设计 1.1"系统内接受邀请"和 4.2"被邀请的邮箱始终可以注册"的做法，**由 M3 的设计交负责人确认**。
- **代码在哪**：注册是否开放由端口 `SignupPolicy` 决定（`server/internal/modules/identity/app/ports.go:353`），现在由 `bootstrap` 接成配置开关 `signupSwitch`（`server/internal/bootstrap/app.go:191-194`）；注册用例在 `server/internal/modules/identity/app/register.go`。M3 扩展 `SignupPolicy` 的实现和注册请求（请求体加上邀请令牌），不另加端口（M2 设计第一稿评审 M16）。签发邀请令牌按账户行锁（M2 设计 3.5）。
- **关闭条件**：负责人的确认写进 M3 设计；只凭邮箱匹配的接受被拒绝，凭有效令牌的成功；`auth.signup_enabled = false` 时，带有效邀请令牌的注册成功、不带的得到"注册已关闭"；这些都有接口版本和页面版本的故事。

## 2. 登录后的落点与新手引导的取数

- **做什么**：在新接口上加回 M2 删掉的取数：`AuthenticationWrapper`（`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`）的落点数据（"上次的工作区"、工作区列表），新手引导页（`web/apps/web/app/(all)/onboarding/page.tsx`）的工作区和邀请。
- **为什么**：M2 的规则是"M2 能到达的页面挂载时不请求 M3 以后的旧接口"（M2 设计 3.1）。所以完成引导的用户登录后直接去 `/create-workspace`，新手引导不再预取，`OnboardingRoot` 的 `invitations` 取默认的空数组。
- **注意**：加回时，工作区取数的 SWR fetcher 要 `return`（或 `await`）`fetchWorkspaces()`（`web/apps/web/core/store/workspace/index.ts:141`）的 Promise。M2 删掉的原 fetcher（原在 `onboarding/page.tsx:33-37`）没有，失败会成为未处理的 Promise 拒绝。邀请页 `web/apps/web/app/(all)/invitations/page.tsx` 的同类缺陷 P5 已改：Promise 交给外层的 `.catch`，失败时提示（M2/P5 spec 第 3 节第 8 条）。SWR 键带上 `loginId`（第 3 节）。
- **Plane 的 `/api/users/me/settings/` 还在**（M2 设计 7.5：工作区数据留给 M3）：`web/apps/web/core/services/user.service.ts:43` 的 `currentUserSettings` 仍调它，nerve 答 404；`web/apps/web/core/store/user/settings.store.ts` 的 `fetchCurrentUserSettings` 把回答存成 `IUserSettings`，工作区 store 的 `getWorkspaceRedirectionUrl`（`web/apps/web/core/store/workspace/index.ts:93-99`）按其中的 `last_workspace_slug`、`fallback_workspace_slug` 算落点。创建、加入、删除工作区和移出成员之后有四处 `await fetchCurrentUserSettings()`（`web/apps/web/core/components/` 下的 `onboarding/steps/workspace/{create,join-invites}.tsx`、`workspace/delete-workspace-form.tsx`、`workspace/settings/members-list-item.tsx`）。M3 让落点由 `profiles.last_workspace_id` 和工作区列表算出，删掉这个方法、`IUserSettings` 的工作区部分和这四处调用，或者给它们新接口；不改的话，工作区接上之后这些 `await` 因 404 抛出，例如新手引导建好工作区却报失败。
- **守卫**：A3、A10 的页面测试用 `watchPage`（`e2e/fixtures/browser.ts`）断言这些页面挂载时没有旧接口请求（`oldApiRequests`）、没有失败的接口请求。加回的取数走 `/api/v0`，这些断言照旧通过；失败说明还在调旧地址。
- **挂载时的规则是否往后延伸**：M2 设计 3.1 的规则只管 M2 能到达的页面。工作区接上之后，工作区的页面挂载时会请求 M4–M8 的旧接口（例如首页的最近访问，M7）；是否对 M3 能到达的页面照这条规则做（`watchPage` 的 `oldApiRequests` 能守住），写进 M3 设计。
- **关闭条件**：完成引导的用户登录后落到上次的工作区，没有时按 Plane 的规则落到第一个工作区或 `/create-workspace`；新手引导的工作区、邀请两步用新接口取数；A3、A10 的断言不改仍通过；取数失败时页面没有未处理的拒绝（故事或单元测试）；`currentUserSettings` 不再调 Plane 的地址（改接或删除），上面四处不再因它失败；README"前端"一节的"M2 中看到的页面"一条随之改写；M3 的设计写明 M2 设计 3.1 挂载时的规则是否延伸到 M3 能到达的页面。

## 3. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M3 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。规则只写在那里；下面是 M3 的范围，在 `d97c513` 上数出。
  - **模块级的 service 实例**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 共 37 处，按调用的接口分，M3 的领域 10 处：`WorkspaceService` 8 处（`web/apps/web/app/(all)/invitations/page.tsx`、`web/apps/web/app/(all)/workspace-invitations/page.tsx`、`web/apps/web/core/components/account/auth-forms/auth-header.tsx`、`web/apps/web/core/components/onboarding/steps/team/root.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/{create,join-invites}.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/core/store/user/permissions.store.ts`），`ProjectService` 1 处（`web/apps/web/core/components/project/form.tsx`），`ProjectMemberService` 1 处（`web/apps/web/core/services/project/project-member.service.ts`）。同是这两个类的另外 9 处调的是搜索、编辑器的 @提及 和最近访问，在 [M4](../../M4-issue-core/handoffs/M2-closeout.md) 第 12 节、[M7](../../M7-collaboration/handoffs/M2-closeout.md) 第 2 节。它们现在经 web 的 axios 基类（`web/apps/web/core/services/api.service.ts`，不带令牌）调 Plane 的旧地址，nerve 答 404。接上新接口时改为 store 构造时用这一代的客户端建。
  - **注册在沿用的 `router` 上的反应**：`web/apps/web/core/store/project/project_filter.store.ts:63` 的 `reaction`。它只做本地的同步更新、不发请求，但不随退役的一代释放，每换一次会话留下一代 stores 的内存。
  - **SWR 键**：工作区、项目、成员的 SWR 键带上 `loginId`。
- **M2 做了什么**：P4 建立分代（`web/apps/web/core/lib/store-context.tsx`、`web/apps/web/core/store/root.store.ts`、`web/apps/web/core/lib/auth/api-client.ts` 的 `apiFor`），P5 的 PAT store（`web/apps/web/core/store/user/api-token.store.ts`）照做（M2/P4 spec 2.8、M2/P5 spec 2.4）。
- **关闭条件**：M3 合并时，上面的 `git grep` 中 M3 领域的 10 处都已消失（或是只调公开操作的 `publicClient`）；`project_filter` 的反应随退役的一代释放，有单元测试（换代之后改路由，旧一代的反应不再运行）；M3 新加的 SWR 键带 `loginId`。

## 4. `profiles.last_workspace_id` 的外键

- **做什么**：决定是否给 `profiles.last_workspace_id` 补外键 `REFERENCES workspaces ON DELETE SET NULL`。
- **为什么**：M2 建 `profiles` 时还没有 `workspaces`，这一列照搬 Plane，是可空、不是外键的 `uuid`（M2 设计 4.3）。
- **怎样补**：迁移归 `identity`：文件名 `<v>_identity_profiles_last_workspace_fk.sql`，列在 `server/sqlc.yaml` 中 `identity` 的条目里，版本号大于建 `workspaces` 的迁移（总体设计 5.6，M2 设计 3.14）。`TestSQLCSchemaScope`（`server/internal/archtest/sqlc_test.go`）核对 `ALTER TABLE` 的所有者。
- **关闭条件**：M3 设计写明补或不补和理由；补的话迁移照上面命名，`server/migrations/schema_test.go` 核对约束名，差异清单二·按表登记。

## 5. `workspace_creation_enabled` 的执行

- **做什么**：配置 `workspace.creation_enabled` 为 `false` 时，创建工作区的接口拒绝；决定是否提供创建工作区的命令（Nerve 没有实例管理员，M2 设计 3.16）。
- **M2 做了什么**：配置和实例接口有这个字段（`server/internal/modules/instance/domain/info.go:22`，`GET /api/v0/instance` 的 `workspace_creation_enabled`），前端四处读它（`/create-workspace`、新手引导的创建工作区、工作区菜单、命令面板）。服务端还没有执行它。
- **关闭条件**：接口在关闭时拒绝，有错误码和测试；是否提供命令写进 M3 设计。

## 6. 停用的端口（决策点 3 已裁定为 A）

- **做什么**：给停用用例加上它声明的端口，在停用的同一个事务里调用：唯一管理员时拒绝；停用成员关系；删除发给这个邮箱的邀请。给 `deactivateMe`（`api/modules/identity.yaml`）声明对应的错误码。
- **为什么**：M2 的停用只处理账户自己的数据（会话、新手引导，M2 设计 3.5）。唯一管理员的检查按 Plane 的本意做，修正它查询的缺陷（Plane 的检查从不拒绝，M2 设计决策点 3），登记差异清单第四节。
- **代码在哪**：`server/internal/modules/identity/app/deactivate.go`：自助停用（`POST /api/v0/me/deactivate`）走 `Execute`，`nerve users deactivate` 走 `ExecuteByEmail`，两者都在账户行锁下调同一个 `deactivate`；端口在 `deactivate` 里调用，两条路就都经过它。端口在 `identity/app` 声明，由 M3 的模块实现，`bootstrap` 接上（总体设计 6.2）。
- **加锁顺序**：端口在账户行锁之内被调用。M2 的全局加锁顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens`，停用按它依次写（M2 设计 3.5）；成员关系、邀请的写入排在这个顺序的哪里，由 M3 定下，写进 M3 设计（M2 设计 6.4 的停用一条："另按 M3 定下的顺序"）。
- **注意**：`nerve users deactivate` 也走这个用例，而命令行的组合没有 River 客户端（`server/internal/bootstrap/users.go:22-26`）。端口的实现如果投递任务，先要有"只投递"的 River 客户端：见 [M4 的交接](../../M4-issue-core/handoffs/M2-closeout.md)第 1 节，那一项随之提前到 M3。
- **关闭条件**：三件事在停用的事务里完成，唯一管理员时停用被拒绝，接口和命令两条路都有测试；`deactivateMe` 的错误码已声明，`apitest` 两个方向的核对通过；差异清单第四节"停用账户"一行更新；M3 的写入在全局加锁顺序中的位置写进 M3 设计。

## 7. 可空的引用字段（M2 设计 3.2）

- **做什么**：工作区图标、项目封面、成员头像随实体进入接口，M5 之前为 `null`；`IUserLite.avatar_url`（`web/packages/types/src/users.ts:24`，现在是 `string`）改为可为 `null`。读取它们的代码不动。
- **为什么**：M5 才有文件存储。先返回 `null`，免得 M3 删掉字段、M5 再加回，前端改两次。只适用于本来允许为空的引用字段。
- **关闭条件**：这些字段在 M3 的接口描述中可为 `null`；`IUserLite` 改用生成的类型，或已改为可空。

## 8. 模块边界：`Authorizer` 端口、sqlc 与 `TestSQLCSchemaScope`

- **做什么**：
  1. `access` 模块读成员表：在 `app` 层声明端口，还是作为 `TestSQLCSchemaScope` 中写明理由的例外（M2 设计 3.14）。
  2. `TestSQLCSchemaScope` 的所有者规则用正则识别 `ALTER TABLE`，漏掉带引号的标识符，以及别的模块的表上 `ALTER` 以外的 DDL（`CREATE INDEX … ON users`、`CREATE TRIGGER`、`DROP TABLE`）。M3 是第一个有跨模块外键的 M，补上这几种写法和它们的反例（M2/P1 评审 M5）。
  3. `Authorizer` 端口由 M3 加入 `internal/shared`（M2 设计 3.3；总体设计 6.2 列了它）。M2 的 `shared` 有 `Actor`、领域错误、`TxManager` 和游标的封套（`server/internal/shared/`），权限检查的端口还没有。
- **代码在哪**：`server/internal/archtest/sqlc_test.go`、`server/internal/archtest/sqlc_cases_test.go`、`server/sqlc.yaml`、`server/internal/shared/`。
- **关闭条件**：M3 设计写明第 1 件的选择；第 2 件的每种写法都有一个反例，规则漏掉它时测试失败；`Authorizer` 在 `shared` 声明，由 M3 的模块实现，`shared` 仍只依赖标准库（架构测试规则 10）。

## 9. 物理删除与跨模块外键的关系图（M3 的部分）

- **做什么**：M3 的每张新表照 M2 设计 3.13 搬 Django 模型的 `on_delete`，并在 M2 设计 4.7 的图上延伸，写明物理删除时每条外键的去向。
- **为什么**：Plane `project.py:77-89` 的项目负责人是 `CASCADE`：物理删除一个账户会连带删除项目。M2 只停用、不删除账户；M3 写明允许物理删除的范围。M4、M6、M7 各自延伸自己的表（它们的交接）。
- **关闭条件**：M3 设计有延伸后的图，每条指向 `users` 的外键写明去向；项目负责人的 `CASCADE` 有结论（照搬，或改为 `SET NULL`、`RESTRICT` 并登记差异）。

## 10. CSP：表情选择器的数据从本站提供

- **做什么**：表情选择器（`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx` 的 `EmojiPicker.Root`，来自 `frimousse`）运行时从 `cdn.jsdelivr.net/npm/emojibase-data` 下载数据。改为随前端一起构建、从本站提供（设置 `emojibaseUrl`，固定版本）。
- **为什么**：页面的 CSP 是 `connect-src 'self'`（`server/internal/platform/webui/csp.go:42`），这个下载现在被挡住。自托管的 Nerve 应当在没有外网时也能用，也不把使用情况告诉第三方（M2 设计 8.3）。项目图标最先用到它，所以在 M3。
- **与 M8 的关系**：[M8 的 M1 收尾交接](../../M8-open-release/handoffs/M1-closeout.md)"AGPL 的三项义务和发往第三方的请求"列了同一个请求；M3 改完之后，那一项对它只剩核对。
- **关闭条件**：打开表情选择器时页面不请求 `cdn.jsdelivr.net`，没有 CSP 违规（故事的 `cspViolations` 为空，或浏览器核对）；CSP 没有为它放开外部来源。

## 11. M2 留下的 M3 调用和类型

- 新手引导的创建工作区、加入工作区、邀请成员三步：界面照常出现，提交调 Plane 的旧接口，M3 之前得到 404（M2 设计 3.1）。
- `web/apps/web/core/services/user.service.ts` 中留下的 `leaveWorkspace`、`joinProject`、`leaveProject`（第 71–87 行）：改接新接口。`joinProject` 的关键词守卫例外见 [M1-P3 的交接](M1-P3-trim-platform.md)"项目成员"一节。
- `IUserLite` 的 `is_bot`（`web/packages/types/src/users.ts:29`）：Nerve 不区分人和机器人（总体设计 0.2），删除。
- 时区接口 `GET /api/v0/timezones`（公开操作，`web/apps/web/core/services/timezone.service.ts`）也供工作区和项目设置使用，不另建。
- 工作区类型的 `owner`（`web/packages/types/src/workspace.ts:20`）：M2 删掉 `IUser` 时改为 `IUserLite`，M3 对接工作区时按接口再定（M2 设计 7.5）。
- **守卫的收紧**：规则 `plane-user-urls`（`tools/keywords.json`）有意不禁止整个 `/api/users/me/`，因为 `joinProject`、工作区列表、邀请和 `/settings/` 这些 M3 的旧调用还在用它下面的地址（M2 设计 7.9）。M3 迁走 `/api/users/me/…` 下最后一个旧调用之后，把规则收紧为整个 `/api/users/me/`，`samples` 随之改。
- **关闭条件**：三步用新接口；这三个方法改用生成的客户端或删除；`is_bot` 删除；工作区、项目设置的时区选择用这个接口；`owner` 的类型取自生成的类型；`plane-user-urls` 已收紧（或 M3 的 review 写明还有哪个旧调用留在 `/api/users/me/` 下、交给哪个 M）。

## 12. 页大小的规则移到 `shared`

- **做什么**：`limit` 的 1–100、默认 50（`api/common.yaml` 的 `Limit`）现在在 `identity/domain`（`server/internal/modules/identity/domain/api_token.go:128-140` 的 `PageSize`），因为 M2 只有 PAT 列表一个使用者（M2/P3a spec 第 3 节第 8 条）。第二个分页列表出现时移到 `internal/shared`，两个列表共用。
- **同一处的提醒**：游标不签名（总体设计 3.4），改成另一个合格的位置照样可用。列表只能让游标决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：第二个分页列表合并时，页大小的规则在 `shared`，两个列表都用它；M3 没有分页列表时，M3 的 review 写明，把本节写进下一个 M 的交接。

## 13. P5 改到、M2 的页面走不到的地方（接上时核对）

它们在工作区、项目的路由上，M2 没有工作区的接口，故事和浏览器核对都到不了。M3 接上这些路由时核对：

- `WorkspaceAuthWrapper`（`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`）"找不到工作区"界面的退出登录改为 `<button type="button">`，名称和提示都经 `t("sign_out")`：Tab 能到，Enter、空格退出。
- 它和 `ProjectAuthWrapper`（`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`）删掉了没有调用方传入的 `isLoading`：工作区或权限加载时仍显示加载中，项目页在项目详情取到之后才显示。
- `ProfileSidebar`（`web/apps/web/core/components/profile/sidebar.tsx`）的窗口大小监听只注册一次，经 ref 读当前的折叠状态（原来读第一次渲染时的值）：宽屏下折叠，再把窗口拉过 768 像素，它重新展开。
- `WorkspaceLogo`（`web/apps/web/core/components/workspace/logo.tsx`）渲染 `<span>`：它在工作区菜单和邀请页的按钮里，按钮只能含短语内容，显示方式由它的 `grid` 类决定。P5 在邀请页上比对过改前改后的盒子和截图，M3 接上工作区菜单和卡片时核对。
- 新手引导加入工作区一步（`web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx`）的每个邀请由可点击的 `div` 改为 `<label>` 连着它的 `Checkbox`：点一行、在勾选框上按空格都勾选，Tab 能到勾选框。P5 用两条假的邀请比对过改前改后的截图（相同），M3 加回邀请的取数时核对。
- 这一步、邀请成员一步和导览的文案 P5 改经 `t()`，M3 接上时在中文下看一遍。
- **关闭条件**：M3 的浏览器核对逐条写明结果。

## 14. 下拉框和复制到剪贴板（M3 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。修法和根因见 M2/P5 spec 2.3。

- **`CustomSearchSelect` 的按钮不在 Tab 顺序里**（`web/packages/ui/src/dropdowns/custom-search-select.tsx`；Headless UI 的 `Combobox.Button` 固定 `tabIndex: -1`，P5 之前也是这样）：改成像 Popover 那样的按钮，`Combobox` 放在它的面板里。个人设置的时区选择就是它。
- **12 个同样写法的下拉框中 M3 的一个**：`dropdowns/member/member-options.tsx`（项目负责人，`projects/create/attributes.tsx`）。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。只跟随关闭（`onClose`）不够：点击打开之后焦点不在任何控件上，Escape 关不掉。其余 11 个在 M4、M6、M7 的交接。
- **P5 重建的 `CustomSelect`**（`web/packages/ui/src/dropdowns/custom-select.tsx`，改用 `Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M3 页面上的 9 个调用方：`workspace/create-workspace-form.tsx`（在 M2 已有的 `/create-workspace` 页上）、`workspace/{invite-modal/fields,settings/invitations-list-item,settings/member-columns,settings/workspace-details}.tsx`、`project/{add-project-members-modal,form,settings/member-columns}.tsx`、`projects/create/attributes.tsx`。
- **`CustomSearchSelect` 在 M3 的调用方**（只有 Headless UI 的一份打开状态，清空搜索不再交出 `null`，`onOpen` 每次打开调一次）：`navigation/project-header.tsx`、`project/{member-select,add-project-members-modal}.tsx`，以及项目页头上的 `BreadcrumbNavigationSearchDropdown`（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`，它的 `onOpen`）。
- **复制到剪贴板**：P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M3 的 3 处调用没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝：`workspace/sidebar/projects-list.tsx`、`project/card.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`。加上处理，经 `t()` 提示失败，像 `api-token/modal/generated-token-details.tsx`。
- **关闭条件**：M3 的 review 逐项写明：`CustomSearchSelect` 能用 Tab 到达、用键盘打开；`member-options` 的列表在按钮旁展开；列出的调用方在浏览器中核对过；3 处复制都处理失败。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M3 的各行；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

## 处理结果（M3/P1）

- **第 4 节 `profiles.last_workspace_id` 的外键**（完成）：不补外键，理由写在 M3 设计 3.14；没有代码改动。
- **第 5 节 `workspace_creation_enabled` 的执行**（完成）：开关关闭时，经过认证、格式正确的 `createWorkspace` 请求答 403 `workspace.creation_disabled`，用例在看请求的值之前先查开关（`server/internal/modules/workspace/app/create_workspace.go`），`api/modules/workspace.yaml` 声明这个码；用例测试、`workspace` 的 HTTP 测试、组合测试的两种开关值、权限矩阵中关闭开关的一行和 W1 都核对它。命令写进 M3 设计 3.11 并实现：`nerve workspaces create --slug --name --admin-email` 不看开关（W10）。
- **第 7 节 可空的引用字段**（部分）：`Workspace.logo_url` 在接口中必有、可为 `null`，M5 之前总是 `null`（`api/modules/workspace.yaml`）；`cover_image_url`、`MemberUser.avatar_url` 随 P2、P4，`IUserLite` 随 P8；本节保持 `open`。
- **第 8 节 模块边界**（完成）：第 1 件选端口：`access` 在 `app/ports.go` 声明 `WorkspaceRoles`，由 `workspace` 的存储实现，`bootstrap` 接上（M3 设计 6.5、6.6），不是 `TestSQLCSchemaScope` 的例外。第 2 件：`TestSQLCSchemaScope` 认出带引号的名字和 `public.` 前缀、`ALTER TABLE`、`CREATE [UNIQUE] INDEX`、`CREATE TRIGGER`、`ALTER TRIGGER`、`DROP TRIGGER`、`DROP TABLE` 和改名，每种都有反例（`server/internal/archtest/sqlc_cases_test.go`），去掉任何一种的变异都让它失败。第 3 件：`Authorizer` 在 `server/internal/shared/authorize.go` 声明，由 `access` 实现，`shared` 仍只依赖标准库。`ProjectAccess` 在 P4 照同一写法。
- **第 12 节 页大小的规则**：M3 的列表都是集合型的，不分页（M3 设计 3.12；P1 的 `listWorkspaces` 答 `{"data": [...]}`）。按关闭条件由 P1 的 review 写明，本节在 M3 收尾时原样写进 M4 的交接（M3 设计 13.2）。

仍未处理，状态保持 `open`：第 1–3、6 节，第 7 节的其余部分，第 9–11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 P1 的 review 和 M3 的收尾。

来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

## 处理结果（M3/P2）

- **第 7 节 可空的引用字段**（部分）：`MemberUser.avatar_url` 在接口中必有、可为 `null`，M5 之前总是 `null`（`api/modules/workspace.yaml`；`listWorkspaceMembers`、`updateWorkspaceMember` 的答复）；`cover_image_url` 随 P4，`IUserLite` 随 P8；本节保持 `open`。

来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

## 处理结果（M3/P3）

- **第 1 节 邀请与注册**（接口一侧完成）：负责人的裁定写在 M3 设计第 10 节（决策点 1、2、4）。接受、忽略要登录，请求体带链接的令牌（`nrv_inv_` 加 22 个字符，由签名密钥派生，不存库，`server/internal/modules/workspace/domain/token.go`）；令牌对、登录账户的邮箱与邀请的邮箱相同时接受成功（200；还不是有效成员的，按邀请的角色成为成员）；不带令牌 400，令牌不对与邀请不存在同一个 404，邮箱不一致 403 `workspace.invitation_email_mismatch`，回答不含被邀请的邮箱。创建邀请最先以 `CallerLock` 锁住邀请人的账户行、在锁下复核凭证（`server/internal/modules/workspace/app/create_invitations.go`）；重置密码先提交的，创建 401、没有邀请（交错 9 `TestInvitingAndResettingThePassword`）。接受最先以 `FOR SHARE` 锁住调用者的账户行，锁下重读 `is_active` 和邮箱（`server/internal/modules/workspace/app/respond_invitation.go`；交错 12 `TestAcceptingAndChangingTheAddress`）；已是有效成员时只消费邀请，成员关系和角色不变（`TestAcceptWorkspaceInvitation`）。`auth.signup_enabled = false` 时，带有效邀请、邮箱相同的注册成功，不带的和邀请无效的各种情况都是 403 `identity.signup_disabled`（`server/internal/bootstrap/signup_policy.go`，`TestRegisteringWithAnInvitationWhileSignupIsOff`）。W5、W6 的接口版本（`e2e/stories/workspace/w5-invitation-link.spec.ts`、`w6-sign-up-by-invitation.spec.ts`）核对这些。页面一侧（邀请页、注册页带着邀请回到邀请页，W5、W6 的页面版本）在 P9，本节保持 `open`。

仍未处理，状态保持 `open`：第 1 节的页面一侧（P9）；第 2、3、6 节，第 7 节的其余部分，第 9–11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 M3 的收尾。

来源：[M3/P3 spec](../specs/P3-invitations.md) 第 7 节。

## 处理结果（M3/P4a）

- **第 7 节 可空的引用字段**（部分）：`Project.cover_image_url` 在接口中必有、可为 `null`，M5 之前总是 `null`（`api/modules/project.yaml`；`listProjects`、`createProject`、`getProject` 的答复）；`IUserLite` 随 P8，本节保持 `open`。
- **第 9 节 物理删除与跨模块外键的关系图**（完成）：图在 M3 设计 4.12，每条指向 `users` 的外键写明去向；项目负责人、默认负责人由 Plane 的 `CASCADE` 改为 `ON DELETE SET NULL`（`server/migrations/sql/00010_project_projects.sql`），登记在差异清单二·按表的 `projects` 各行和第四节"项目负责人、默认负责人"。

仍未处理，状态保持 `open`：第 1 节的页面一侧（P9）；第 2、3、6 节，第 7 节的其余部分，第 10、11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 M3 的收尾。

来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

## 处理结果（M3/P6）

- **第 6 节 停用的端口**（完成）：`identity/app` 声明 `MembershipDeactivator`（`DeactivateMemberships(ctx, userID, email)`），`deactivate`（`server/internal/modules/identity/app/deactivate.go`）在撤销会话之后调用它，自助停用（`Execute`）和 `nerve users deactivate`（`ExecuteByEmail`）都经过这里，邮箱取自锁下的账户行。它由 `workspace` 的 `Deactivator` 实现（`server/internal/modules/workspace/app/deactivate_memberships.go`），`bootstrap` 接上：服务用 `workspace.New` 的 `Deactivator()`，命令行用 `workspace.NewDeactivator` 和 `project.NewCascade`（`server/internal/bootstrap/users.go`）。三件事在停用的事务里：按 id 锁住他所在的全部工作区之后，他是某个工作区或项目唯一的有效管理员、那里还有别的有效成员时拒绝（409 `workspace.sole_admin`、`project.sole_admin`，数据库不变）；删除发给他邮箱的全部邀请（待接受的和已忽略的），和他是唯一有效成员的工作区的待接受邀请，删除之前由一条语句按 id 锁住它们，删除只写锁住的那些（P6 终审的 I1 和裁定 F-1，M3 设计 3.9）；结束他在全部工作区和项目的成员关系，行和角色留着。`deactivateMe` 声明这两个码，`identity` 自己的 HTTP 测试返回过它们，`apitest` 两个方向的核对通过；接口和命令两条路各有组合测试（`server/internal/bootstrap/deactivation_test.go`、`deactivation_locks_test.go`、`deactivation_races_test.go`、`deactivation_crossed_test.go`）和故事 W9；差异清单第四节"停用账户"一行已更新；这些写入在全局加锁顺序中的位置写在 M3 设计 3.6（账户行、工作区、邀请、工作区成员、项目、项目成员）。
- **第 6 节的"注意"**（M4 交接第 1 节的"只投递"River 客户端）：停用的端口不投递任务，命令行的组合仍没有 River 客户端，这一项不提前到 M3，照原计划留在 M4（M3 设计 3.9、13.1）。

仍未处理，状态保持 `open`：第 1 节的页面一侧（P9）；第 2、3 节，第 7 节的其余部分，第 10、11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 M3 的收尾。

来源：[M3/P6 spec](../specs/P6-deactivation.md) 第 7 节。

## 处理结果（M3/P8b）

P8a 没有在这里记处理结果；它做到的部分（[P8a spec](../specs/P8a-web-workspace-data.md) 第 3 节的交接表和第 7 节）一并记在下面。

- **第 2 节 登录后的落点与新手引导的取数**（P8a 的部分完成）：落点由 `landingPath`（按调用者的工作区列表和资料的 `last_workspace_id`）算出，`useLanding` 做出全部判断，`AuthenticationWrapper` 照它渲染，`use-landing.test.ts` 守着；`currentUserSettings`、`IUserSettings`、设置 store 和创建、加入、删除工作区与移出成员之后的四处 `await fetchCurrentUserSettings()` 删除；取数失败不是未处理的拒绝（`useSessionSWR` 交回 store 的 Promise）（P8a 的 Task 1、3–5，P8a spec 第 3 节的交接表）。页面一侧（W2 的页面版本、新手引导的两步、README 的说明）在 P9。
- **第 3 节 stores 按会话分代**（完成）：`git grep` 中 M3 领域的 10 处都已消失：`WorkspaceService` 8 处在 P8a，`project/form.tsx` 的 `ProjectService`（Task 5）和 `ProjectMemberService`（Task 4、6）2 处在 P8b；剩下的 22 处都是 M4–M7 的（[P8b spec](../specs/P8b-web-project-data.md) 附录 A.6）。`project_filter` 跟随 `router` 的反应由 `RootStore.dispose()` 随退役的一代释放，`store-context.test.ts` 核对换代之后它不再运行（P8a；P8b 的修正轮把项目列表的筛选按工作区的 id 存之后，这个测试先载入两代各自的工作区列表）。M3 的会话取数都经 `useSessionSWR`，键带 `loginId`（P8a spec 2.1、P8b spec 2.12），根目录 `.oxlintrc.json` 的 `no-restricted-imports` 看住范围内的文件不从 `swr` 导入值。
- **第 7 节 可空的引用字段**（完成）：`IUserLite` 换成生成的 `MemberUser`，`avatar_url` 和 `Workspace.logo_url` 读作 `null`（P8a）；`Project.cover_image_url` 读作 `null`，没有封面时显示默认图（P8b 的 Task 1、2，W20 的 `t2-cover-null`）。
- **第 11 节 M2 留下的 M3 调用和类型**（代码一侧完成；页面一侧在 P9、P10）：`leaveWorkspace` 改调新接口（P8a）；`UserService.joinProject`、`leaveProject` 删除，加入、离开项目经 `ProjectsService.join`、`leave` 和项目 store 的 `joinProject`、`leaveProject`（P8b 的 Task 4，P8b spec 第 3 节第 4 条）；`is_bot` 删除，M4 的活动类型中留下的两处随 M4（P8a spec 第 5 节）；工作区的 `owner` 不再存在（P8a）；新手引导的加入一步随决策点 2 删除，创建、邀请两步发到 nerve（P8a，页面行为在 P9）；时区经 `GET /api/v0/timezones`，工作区和项目的 general 页挂载时都取它（P8b spec 附录 A.5）。留给页面的：新手引导的创建、邀请两步的页面行为（P9），工作区、项目 general 页的时区选择的核对（P9、P10）；本节随 P10 关闭（M3 设计 12 节 P10 的"关闭"、13.1），`is_bot` 在 M4 的活动类型中的两处随 M4。`plane-user-urls` 收紧为整个 `/api/users/` 和 `/api/instances/`（P8b 的 Task 11），`web/apps/web` 中 `/api/users/` 下的调用都已不在（P8b spec 附录 A.4）；P8b 的修正轮给它加上 `/api/users/me/` 之外的命中样例（`/api/users/last-visited-workspace/`），规则退回只禁 `/api/users/me/` 时样例失败（变异 `fw-p15-users-me-only`）。

仍未处理，状态保持 `open`：第 1 节的页面一侧、第 2 节的页面一侧和第 13 节（P9）；第 11 节的页面一侧（P9、P10）；第 10、14 节（P10，M3 设计 7.7）；第 12 节等 M3 的收尾。

来源：[M3/P8b spec](../specs/P8b-web-project-data.md) 第 7 节；[M3/P8a spec](../specs/P8a-web-workspace-data.md) 第 7 节。

## 处理结果（M3/P9）

- **第 1 节 邀请与注册**（页面一侧完成）：邀请链接的页面由 `invitationView`（`web/apps/web/core/components/workspace/invitation-view.ts`）决定显示什么：链接缺 id 或令牌、nerve 按链接找不到时无效；没有登录时给登录、注册，两者都带着链接回来；登录之后接受或忽略（`use-invitation-answer.ts`），邮箱不一致时只给退出登录；接受之后写上次打开的工作区再进入。注册凭链接中的邀请（注册关闭时也能注册），登录页和注册页之间的链接保留链接的参数。W5、W6 的页面版本守着（[M3/P9 spec](../specs/P9-web-workspace-pages.md) 2.4、2.7）。
- **第 2 节 登录后的落点与新手引导的取数**（页面一侧完成）：新手引导取调用者的工作区列表，由它决定步骤，已有工作区的人资料一步之后就完成；创建一步与 `/create-workspace` 共用 `useCreateWorkspace`，邀请一步发往刚建的工作区；完成引导不再把列表中的第一个工作区写成上次打开的。W2、W1、W5 的页面版本分别守着切换、创建、接受之后写的 `last_workspace_id`，W2 的还守着删除、离开之后的落点；README 的"前端"一节写明能用的页面（spec 2.2、2.4、2.5、2.6、2.11）。
- **第 11 节 M2 留下的 M3 调用和类型**（页面一侧中工作区的部分完成）：新手引导的创建、邀请两步（spec 2.6）；工作区 general 页的时区随修改发出（spec 2.1）。项目的部分随 P10。
- **第 13 节**（P9 的部分）：`WorkspaceAuthWrapper` 的"找不到工作区"界面和退出登录失败的提示经 `t()`（spec 2.10）；新手引导的邀请一步换成成员页的表单（spec 2.6）。逐条的浏览器核对是 M3/P9 评审的 C4。
- **M2 的 `UserStore.deactivateAccount`**（`web/apps/web/core/store/user/index.ts`，M3/P9 改了签名）：交回 `Promise<boolean>`，即 `tokenManager.endSession(loginId)` 的回答：停用应答时标签页的记录是否仍是发出停用的那个会话（是时结束它）；另一个标签页先退出、或把这个标签页换到别的账户时为 `false`。它唯一的调用方停用弹窗只在为 `true` 时说已停用并关闭；nerve 拒绝时它照旧拒绝（spec 2.9，第 3 节第 3 条）。

仍未处理，状态保持 `open`：第 11 节的页面一侧（P10）；第 13 节等 C4 写进 M3/P9 的评审；第 10、14 节（P10，M3 设计 7.7）；第 12 节等 M3 的收尾。

来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。
