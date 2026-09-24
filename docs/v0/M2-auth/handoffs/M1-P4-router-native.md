---
status: open
from: M1/P4
to: M2
created: 2026-09-24
---

# 路由与跳转：`next_path`、认证包装和接口地址

M1/P4 去掉了 Next.js 兼容层：web 应用只用 React Router 的写法，没有前端环境变量，接口一律用相对路径（同源部署）。认证相关的部分只做了"让它在原生路由下正确工作"的最小改动（M1 设计 4.2），下面几件事由 M2 接手。

## 必须由 M2 完成

- **服务端校验 `next_path`。** 前端只在跳转前校验：`AuthenticationWrapper` 读一次 `next_path`、修剪首尾空白，用 `@plane/utils` 的 `isValidNextPath` 判断（只接受以单个 `/` 开头的站内路径，拒绝 `//`、`\` 和任何协议，带单元测试），然后跳到校验过的那个值。但登录、注册表单把地址里的原值作为隐藏字段提交（`web/apps/web/core/components/account/auth-forms/password.tsx` 的 `<input type="hidden" name="next_path">`），由服务端发出最后的跳转。M2 的 Go 处理器必须用同一条规则校验：修剪后以单个 `/` 开头，没有 `//`、`\`，没有控制字符；不合格时按没有 `next_path` 处理。
- **重写 `AuthenticationWrapper`**（`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`）。P4 只把 6 处渲染时跳转改为 `<Navigate replace />`，被守卫挡住的地址不留在历史里。令牌管理器替换 Cookie 会话时一并重做。
- **401 处理**（`web/apps/web/core/services/api.service.ts`）：`currentPath ? … : ""` 恒为真（`pathname` 不会为空）。令牌管理器替换这段时不要照搬。

## 由 M2 决定

- `next_path` 只带 `pathname`，不带原地址的查询参数和片段（基线如此）。登录之后回到带筛选或定位到评论的地址，需要把它们也带上。
- 修改密码页（`settings/profile/content/pages/security.tsx`）把错误断言为 `Error & { error_code?: string }`，Plane 实际返回的是数字，靠 `err.error_code?.toString()` 才能与枚举匹配（P4 Task 7 删空转换时因此保留了这一处）。认证错误的契约由 OpenAPI 生成时，类型要与实际取值一致，这个断言和转换随之去掉。
- 登录、注册表单和退出直接提交到 `/auth/…`（相对地址），不再有基础地址。

来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。
