---
status: open
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

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
