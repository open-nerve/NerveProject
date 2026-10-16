---
status: open
from: M3/closeout
to: M5
created: 2026-10-13
---

# M3 交给 M5：工作区图标、项目封面和头像，`organization_size` 的取值，附件的确认框

M3（工作区与项目）把工作区图标、项目封面和成员头像留成可为 `null` 的引用字段，等 M5 的文件存储（[M3 设计](../../M3-workspace-project/M3-design.md) 3.2）。下面每一节来自 M3 设计 13.2 的 M5 一行，和 M3 各 Phase、M3 收尾交给 M5 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。

## 1. 图标、封面和头像

- **做什么**：`workspaces.logo_asset_id`、`projects.cover_image_asset_id` 的迁移归各自的模块（`workspace`、`project`，文件名按总体设计 5.6）；工作区图标和项目封面的上传控件从 Plane 的源码加回；接口的 `Workspace.logo_url`、`Project.cover_image_url`、`MemberUser.avatar_url` 有真值（M3 设计 3.2；今天它们在接口中必有、总是 `null`）。
- **预设封面**：M3/P8b 删掉了上传预设封面的一步、随机封面和 28 张预设封面，只留显示用的默认图 `image_1.webp`（M3 设计 7.8）。它们随封面的上传加回，新建项目时的封面值照 M5 的 [M1 收尾交接](M1-closeout.md)"新建项目时的封面值"做。
- **关闭条件**：两个迁移和差异清单二·按表的两行；三个字段有真值的接口版本和页面版本的故事；预设封面和随机封面的取舍写进 M5 的设计。

## 2. `Workspace.organization_size` 收紧到数据库的取值

- **做什么**：数据库的 CHECK 只收前端的六个选项（`Just myself`、`2-10`、`11-50`、`51-200`、`201-500`、`500+`，`server/migrations/sql/00006_workspace_workspaces.sql`），请求（`WorkspaceCreate`、`WorkspaceUpdate`）引用同样六个值的枚举 `OrganizationSize`，回答的 `Workspace.organization_size` 却仍是可为 `null` 的任意字符串（`api/modules/workspace.yaml`）。把回答也改为引用 `OrganizationSize`（可为 `null`），前端读生成的类型。
- **为什么在 M5**：M3/P8a 修复轮的搁置项 P11 定为"下一个改 `api/modules/workspace.yaml` 的人做"，M5 加工作区图标时第一个改它。这是契约的改动（回答的类型变窄；库里的值都在六个值里）；负责人另有裁定时照裁定。
- **关闭条件**：`Workspace.organization_size` 引用 `OrganizationSize`，`make gen-check` 没有差异，读它的前端代码用生成的类型。

## 3. 附件的确认框和弹窗

- M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：弹窗打开时后面的页面 `inert`、`aria-hidden`；`AlertModalCore` 的确认框在请求路上时关不掉、取消不能用（M3/P11 spec 第 3 节第 2 条）。附件的删除确认框随之改变，M5 的页面能运行时核对：被拒绝时照 A-M2 留着、只在成功的跟进中关上，按钮的文字经 `t()` 传入（`@nerve/ui` 的默认值是英文，见 M8 的交接）。
- 共用组件的焦点轮廓在 M5 的页面上是否被 `overflow-hidden` 裁掉，用键盘核对（M3/P11 评审第 6 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，组件状态里的弹窗和状态不带过去。store 持有的弹窗开关由渲染它的宿主在卸载时关上（`useLayoutEffect` 的清理，像 `WorkspaceLevelModals`）；宿主在 key 之上时还要在范围结束时关上。附件的确认框、上传的弹窗和进度属于一个工作区或项目，放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）。
- **嵌套的 id 没有 key**（M3 收尾）：附件的确认框和上传的弹窗在工作项的页面上（`[issueId]`、`browse/[workItem]`），随 M4 给这些页面的 key 结束（M4 的交接第 6 节）；M5 自己加的、在一个 id 之下的页面同样以那个 id 给它的子树 key，并加一个后退的检查。
- **附件的错误只是红字**（M3 收尾，浏览器核对之后）：上传附件被拒绝时，`issues/attachment/attachment-upload.tsx:63` 在拖放区里显示 `<p className="text-center text-danger-primary">{fileError}</p>`，读屏不读它，焦点也不去它那里（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2 条的第三次扫描找到）。它不是一个字段的提示，是 M5 的上传。照 M2、M3 对不属于字段的错误的写法：经共用的拒绝提示（`core/hooks/use-refusal-toast.ts`，在不 `inert` 的提示区域里被读出），或像停用账户的弹窗那样以 `role="alert"` 显示（`account/deactivate-account-modal.tsx:91`）；文字经 `t()`。
- **关闭条件**：M5 的评审写明附件的确认框和上传弹窗的浏览器核对。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M5 一行；M3/P8a、P8b 的 spec 第 5 节，P8a 修复轮的搁置项 P11，M3/P11 评审第 6 节；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
