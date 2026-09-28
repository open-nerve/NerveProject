---
status: closed
from: M1/P3
to: M2
created: 2026-09-24
---

# 认证与实例配置：前端剩下的形态，以及 M2 必须接手的几件事

M1/P3 删掉了第三方登录、验证码登录、找回 / 重置 / 设置密码、"检查邮箱"步骤、管理后台和所有邮件发送（Task 1）。登录页和注册页各剩一个"邮箱 + 密码"表单，模式由路由决定（`/` 是登录，`/sign-up` 是注册），邮箱可以编辑。

## 必须由 M2 完成

- **删掉 Cookie 会话和 CSRF**（M1 设计 3.6，不能再延期）。前端仍这样提交：
  - 登录、注册：原生表单 POST 到 `/auth/sign-in/`、`/auth/sign-up/`，带 `csrfmiddlewaretoken`（`web/apps/web/core/components/account/auth-forms/password.tsx`）；
  - 退出：`AuthService` 临时拼一个带 `csrfmiddlewaretoken` 的表单（`web/apps/web/core/services/auth.service.ts`）；
  - 修改密码：请求头 `X-CSRFTOKEN`（`web/apps/web/core/services/user.service.ts`）。

  M2 用令牌管理器重写登录页时，这四处一起换掉。
- **认证错误就地显示。** Plane 的认证接口把所有错误都重定向到 `/?error_code=…`。所以现在注册失败会回到登录页，错误提示显示在登录表单上方，用户要再点"注册"（P3 spec 第 3 节第 2 条，有意不为此加按错误码切换模式的代码）。错误提示里的"去注册""去登录"链接会带上地址里的 `email`。M2 的新接口直接返回错误，不再重定向，这一段行为随之消失。
- **修改登录邮箱的最终形态**（M1 设计 3.13，决策点）。旧的"向新邮箱发验证码"流程已删除。M2 设计时要让项目负责人在三者中选一个：输入当前密码后自助修改；只能由管理员用命令行修改；v0 暂不提供。

## 实例配置

`IInstanceInfo` 只剩 `config`；`IInstanceConfig`（`web/packages/types/src/instance/base.ts`）从 16 个字段变为 4 个：

| 字段 | 读取方 | 待定 |
|---|---|---|
| `enable_signup` | 登录页页头的"注册"链接 | 名称由 M2 定 |
| `is_workspace_creation_disabled` | 创建工作区页、新手引导、工作区菜单、命令面板 | — |
| `file_size_limit` | `use-file-size` | 名称和策略由 M2 或 M5 定 |
| `is_self_managed` | 新手引导：为真时跳过"角色""用途"两步 | **去留由 M2 定**：Nerve 只有自托管一种形态，这个字段恒为真时，要连同新手引导的两个步骤一起决定（这是产品决定，不是死代码，M1 没有按恒真化简） |

实例请求失败时显示维护页；管理后台的配置类型已全部删除，实例配置改由配置文件和命令行管理。

## 前端不再读的用户字段

`is_password_autoset`、`last_login_medium`、`has_marketing_email_consent`、`billing_address_country`、`billing_address`、`has_billing_address`，邮件通知偏好的 5 个开关，第三方账户（`provider`、`provider_account_id`）。安全页只剩"输入旧密码修改密码"。

## 前端不再调用的地址

`/auth/email-check/`、`/auth/magic-*`、`/auth/forgot-password/`、`/auth/reset-password/`、`/auth/set-password/`、第三方登录的全部地址、`/api/users/me/notification-preferences/`、`/api/users/me/email/generate-code/` 和修改邮箱、`/api/users/me/instance-admin/`（`is_instance_admin`）。

## 其他

- `packages/services` 的 API 令牌 `retrieve`（GET）和 `destroy`（DELETE）的地址没有结尾斜杠（`/api/users/api-tokens/${tokenId}`，靠 Django 的 `APPEND_SLASH` 重定向）。M2 重新定义令牌接口时一并处理。
- 个人设置只剩 `general`、`security`、`preferences`、`api-tokens` 四个标签页，由 `web/packages/constants/src/navigation.test.ts` 守住。

## 关闭条件

M2 合并时：

- `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出，登录、注册、退出、修改密码都走令牌管理器；
- 认证错误由接口直接返回，前端不再从 `/?error_code=…` 读取；
- 修改登录邮箱的三选一有项目负责人的裁定，写进 M2 设计；
- `IInstanceConfig` 的 4 个字段在 `api/` 里各有定义或已删除，`is_self_managed` 连同新手引导的两个步骤有结论，`enable_signup` 的名称已定；
- "不再读的用户字段""不再调用的地址"两节列出的都不出现在 M2 的接口描述里；
- API 令牌的地址随新接口统一结尾斜杠。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。

## 处理结果（M2/P3a）

- **实例配置**（接口完成，M2 设计 5.3）：`InstanceInfo` 加上 `signup_enabled`（替代 `enable_signup`）、`workspace_creation_enabled`（`is_workspace_creation_disabled` 取反）、`file_size_limit`，取自配置；`is_self_managed` 不定义，新手引导的"角色""用途"两步随之删除（M2 设计 3.19，前端在 M2/P4）。
- **不再读的用户字段、不再调用的地址**（接口完成）：两节列出的字段和地址都不出现在 `api/` 的接口描述中。
- **令牌的地址**（完成）：`/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}`，结尾都不带 `/`。
- **修改登录邮箱**：负责人已裁定只能由管理员用命令行修改（M2 设计决策点 1），命令 `nerve users set-email` 在 M2/P3b。

仍未处理，状态保持 `open`：Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（M2/P4）；`set-email` 命令（M2/P3b）。

来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。

## 处理结果（M2/P3b）

- **修改登录邮箱**（完成）：`nerve users set-email --email <旧邮箱> --new-email <新邮箱>`（M2 设计决策点 1、3.17）。新邮箱按注册时的规则规范化和校验，改写 `users.email`，结束该账户的全部会话（`email_changed`），PAT 不撤销；新邮箱已被别的账户使用、或与旧邮箱相同时，退出码为 1，数据库不变。端到端 A16 覆盖。接口和界面都没有修改邮箱的入口，`updateMe` 的说明写明这个命令。

仍未处理，状态保持 `open`：Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（M2/P4）。

来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。

## 处理结果（M2/P4）

- **Cookie 会话和 CSRF**（完成）：四处都已换掉。登录、注册由 `useUser().signIn`、`signUp` 经 `AuthService` 用 JSON 调 `POST /api/v0/auth/login`、`register`（不带令牌的 `publicClient`），令牌交给令牌管理器；退出由令牌管理器在续期用的那把锁下调 `POST /api/v0/auth/logout`；修改密码调 `POST /api/v0/me/change-password`，访问令牌由认证中间件加上。web 的 axios 基类删掉 `withCredentials` 和 401 拦截。`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出，关键词规则 `csrf`（不区分大小写）和 `plane-auth-urls` 看住。
- **认证错误就地显示**（完成，M2 设计 7.3）：接口直接返回 problem，登录、注册表单把它显示在表单上方或对应字段下方（`weak_password` 显示规则，`common_password` 显示"密码太常见"），不跳转，输入的内容留在表单里；表单上方的提示按 `problem.code` 取文案，字段下方的按字段错误的 `code` 取（`helpers/authentication.helper.ts`；单元测试对着 `openapi.yaml` 核对两张表）。前端不再读 `/?error_code=…`，Plane 的错误码枚举 `EAuthenticationErrorCodes`（原在 `helpers/authentication.helper.tsx`）和 `@nerve/types` 的 `auth.ts` 删除，关键词规则 `auth-error-code` 看住。A2、A3、A15 的页面版本覆盖。
- **实例配置**（完成）：`IInstanceConfig`、`IInstanceInfo` 删除，instance store 直接用生成的 `InstanceInfo`；注册链接读 `signup_enabled`，创建工作区的四处读 `workspace_creation_enabled`，`use-file-size` 读 `file_size_limit`。新手引导的"角色""用途"两步连同 `is_self_managed` 删除（M2 设计 3.19），关键词规则 `is-self-managed` 看住。

仍未处理，状态保持 `open`：`packages/services` 的 API 令牌服务仍调旧地址 `/api/users/api-tokens/…`（没有结尾斜杠的 `retrieve`、`destroy` 也在其中），M2/P5 把 api-tokens 标签页改接 `/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}` 时删除。

来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/P5）

- **API 令牌的旧地址**（完成）：`@nerve/services` 的令牌服务 `developer/` 和它的 axios 基类 `api.service.ts` 删除，包里只剩地址工具和文件工具（M2 设计 7.5）。api-tokens 页和 security 页的令牌列表经 PAT store（`core/store/user/api-token.store.ts`）用生成的客户端调 `GET`、`POST /api/v0/me/api-tokens` 和 `DELETE /api/v0/api-tokens/{token_id}`；store 属于会话的 `RootStore`，用这个会话的客户端。关键词规则 `plane-api-token-urls`（`/api/users/api-tokens`）和 `with-credentials`（`withCredentials:\s*true`）看住。

全部处理完，状态改为 `done`。

来源：[M2/P5 spec](../specs/P5-web-account.md) 第 7 节。

## 处理结果（M2/收尾）

逐条核对关闭条件（`d97c513`）：

- **CSRF**：`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；登录、注册、退出、修改密码都走令牌管理器（M2/P4）。
- **认证错误**：接口直接返回 problem，前端不读 `/?error_code=…`（关键词规则 `auth-error-code`，M2/P4）。
- **修改登录邮箱**：负责人裁定为 B，`nerve users set-email`（M2/P3b，A16）。
- **实例配置**：`InstanceInfo` 有 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（`api/modules/instance.yaml:83`）；`is_self_managed` 连同新手引导的两步删除（关键词规则 `is-self-managed`，M2/P4）。
- **不再读的用户字段、不再调用的地址**（原文要求收尾再核对一次）：`git grep -n -i -E 'is_password_autoset|last_login_medium|has_marketing_email_consent|billing_address|has_billing_address|property_change|state_change|issue_completed|comment|[^_]mention|provider|notification|email-check|magic|forgot|reset-password|set-password|generate-code|change-email|instance-admin|is_instance_admin|google|github|gitlab|gitea|[^0]/auth/' d97c513 -- api` 没有输出（退出码 1）。接口描述共 15 个操作、12 个路径，都在 `/api/v0/auth/`、`/api/v0/me`、`/api/v0/api-tokens/{token_id}`、`/api/v0/instance`、`/api/v0/timezones` 下。
- **令牌的地址**：`/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}`，结尾都没有 `/`（M2/P3a）。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
