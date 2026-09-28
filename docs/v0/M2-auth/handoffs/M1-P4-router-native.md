---
status: closed
from: M1/P4
to: M2
created: 2026-09-24
---

# 路由与跳转：`next_path`、认证包装和接口地址

M1/P4 去掉了 Next.js 兼容层：web 应用只用 React Router 的写法，没有前端环境变量，接口一律用相对路径（同源部署）。认证相关的部分只做了"让它在原生路由下正确工作"的最小改动（M1 设计 4.2），下面几件事由 M2 接手。

## 必须由 M2 完成

- **服务端校验 `next_path`。** 前端只在跳转前校验：`AuthenticationWrapper` 读一次 `next_path`、修剪首尾空白，用 `@nerve/utils` 的 `isValidNextPath` 判断（只接受以单个 `/` 开头的站内路径，拒绝 `//`、`\` 和任何协议，带单元测试），然后跳到校验过的那个值。但登录、注册表单把地址里的原值作为隐藏字段提交（`web/apps/web/core/components/account/auth-forms/password.tsx` 的 `<input type="hidden" name="next_path">`），由服务端发出最后的跳转。M2 的 Go 处理器必须用同一条规则校验：修剪后以单个 `/` 开头，没有 `//`、`\`，没有控制字符；不合格时按没有 `next_path` 处理。
- **重写 `AuthenticationWrapper`**（`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`）。P4 只把 6 处渲染时跳转改为 `<Navigate replace />`，被守卫挡住的地址不留在历史里。令牌管理器替换 Cookie 会话时一并重做。
- **401 处理**（`web/apps/web/core/services/api.service.ts`）：`currentPath ? … : ""` 恒为真（`pathname` 不会为空）。令牌管理器替换这段时不要照搬。

## 由 M2 决定

- `next_path` 只带 `pathname`，不带原地址的查询参数和片段（基线如此）。登录之后回到带筛选或定位到评论的地址，需要把它们也带上。
- 修改密码页（`settings/profile/content/pages/security.tsx`）把错误断言为 `Error & { error_code?: string }`，Plane 实际返回的是数字，靠 `err.error_code?.toString()` 才能与枚举匹配（P4 Task 7 删空转换时因此保留了这一处）。认证错误的契约由 OpenAPI 生成时，类型要与实际取值一致，这个断言和转换随之去掉。
- 登录、注册表单和退出直接提交到 `/auth/…`（相对地址），不再有基础地址。

## 关闭条件

M2 合并时：

- Go 的登录、注册处理器按本文件的规则校验 `next_path`，测试覆盖以 `//`、`\` 开头、带协议和带控制字符的值（都按没有 `next_path` 处理）；
- `AuthenticationWrapper` 和 401 处理随令牌管理器重写，没有照搬恒为真的判断；
- "由 M2 决定"的三项在 M2 的设计或 review 里各有结论。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。

## 处理结果（M2/P4）

- **服务端校验 `next_path`**（按"服务端没有跳转"关闭，M2 设计 3.18）：M2 的登录、注册是 JSON 接口，服务端不发出跳转，也不接收 `next_path`；表单不再提交它（隐藏字段随表单的重写删除）。跳转只发生在浏览器里，唯一的关口是 `@nerve/utils` 的 `isValidNextPath`：任何位置都没有控制字符（`\u0000`–`\u001f`、`\u007f`；浏览器会从地址里丢掉制表符和换行，`/\t/evil.example` 会变成 `//evil.example`）和反斜杠，修剪首尾空白后以 `/` 开头、不以 `//` 开头，按本站地址解析后主机不变。单元测试覆盖 `//`、`\`、`/\`、协议和控制字符；A3 的页面版本打开四个不合格的 `next_path`，登录后都落到 `/create-workspace`。
- **`AuthenticationWrapper`**（完成，M2 设计 7.4）：随令牌管理器重写。会话状态来自令牌管理器（`useSession`）；未登录而在需要账户的页面时 `<Navigate to={signInPath(…)} replace />`；已登录而在登录页、注册页时，去合格的 `next_path`，没有时按是否完成新手引导去 `/onboarding` 或 `/create-workspace`（需要账户的页面仍先要求完成新手引导）；"会话暂不可用"时显示可重试的界面，地址和记录都不变，只有令牌管理器排定了重试时说明里才说页面会自动重试。认证相关的跳转只在这一处发生：停用账户、切换账户的弹窗不再自己跳转。
- **401 处理**（完成，M2 设计 7.1、7.2）：web 的 axios 基类删掉 401 拦截和 `withCredentials`，恒为真的 `currentPath` 判断没有照搬；401 由 web 的认证中间件（`web/apps/web/core/lib/auth/auth-middleware.ts`，挂在 `@nerve/api-client` 的客户端上）处理：续期一次、重发一次，只有续期得到 401 或重发后仍是 401 才结束会话；标签页已不在请求所属的会话时，请求以 `SessionChangedError` 结束，不续期、不重发。
- **"由 M2 决定"的三项**：
  - `next_path` 带上查询参数和片段（M2 设计 3.18）：`AuthenticationWrapper` 把 `pathname + search + hash` 交给 `signInPath`，它整体 `encodeURIComponent`；单元测试和 A3 的页面版本（`/settings/profile/general?tab=x#y` 登录后原样回到）覆盖。
  - 修改密码页的错误断言（完成）：`Error & { error_code?: string }` 和 `toString()` 删除；错误是 `ApiError`，按 `problem.code`（`identity.current_password_incorrect` 显示在当前密码下方）和 `problem.errors`（`new_password` 的规则）显示，类型来自接口描述。
  - 表单提交的地址（完成）：登录、注册、退出不再提交到 `/auth/…`，改为经 `@nerve/api-client` 的客户端调 `/api/v0/auth/…`，关键词规则 `plane-auth-urls` 看住。

全部处理完，状态改为 `done`。

来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

三项在 M2/P4 完成，收尾在 `d97c513` 上核对：`next_path` 的关口是 `@nerve/utils` 的 `isValidNextPath`（`web/packages/utils/src/url.ts`），单元测试 `web/packages/utils/src/next-path.test.ts` 随 `make test-web` 通过，A3 的页面测试打开四个不合格的 `next_path`，登录后都落到 `/create-workspace`（`make e2e`）；`AuthenticationWrapper` 和 401 处理（`web/apps/web/core/lib/auth/auth-middleware.ts`）随令牌管理器重写；"由 M2 决定"的三项在 M2 设计 3.18、7.2、7.7 有结论。状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
