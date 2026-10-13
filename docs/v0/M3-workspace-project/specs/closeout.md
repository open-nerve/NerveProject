# M3/收尾 closeout：设计说明（spec）

| 项 | 内容 |
|---|---|
| 里程碑 | M3 工作区与项目（`docs/v0/M3-workspace-project`） |
| 日期 | 2026-10-13 |
| 状态 | 进行中：第 3 节的各条已由控制者裁定（2026-10-13，写在各条之后），第 3.2 节的三条负责人已答（2026-10-13）；预检之后修订一轮（2026-10-13：预检的发现、控制者对它们的裁定和负责人的答复都已写进本 spec 和计划） |
| 基点 | `49190525`（Merge M3/P11）；M3 从 `f8cb7c22`（M2 的修复轮合并之后）开始 |
| 上级 | [M3 设计](../M3-design.md) 第 12 节"收尾"、13.1、13.2、第 14、15 节、3.20、7.1、7.7、7.9、7.11、8.7、9.7（C11、C12）；[总体设计](../../v0-design.md) 7.7、9.4 |
| 前置交接 | P1–P11 每份 spec 的第 5 节和每份评审的第 6 节中交给收尾的各条；P8b–P11 修复轮的搁置清单；`frontend-changes.md`；M3 收到的 6 份交接（2.1） |
| 计划 | [plans/closeout.md](../plans/closeout.md)，16 个 Task |
| 原型 | `$M3TMP/closeout-proto`（`49190525` 的 `git archive` 副本），Task 0 之后逐个 Task 写成，每个 Task 之后的树存在 `$M3TMP/closeout-snap/T<n>`；工具、日志和计划的块在 `$M3TMP/closeout-tools/`。计划在另一个副本上从本 spec 和计划的提交逐 Task 复现（修订之后从修订的提交再一次），每个 Task 之后跑它的门禁，16 个都通过，最后的树与原型逐个文件相同（附录 A.9） |

`$M3TMP` 是 `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-m3`。

收尾按安全和行为改动看待每一处改动，不按重构看待：`ModalCore` 的重写碰到 M2–M8 的每一个弹窗，其中 31 个没有一个故事打开（附录 A.3），所以弹窗的每一条规则（后面的页面 `inert`、`aria-hidden`，Tab 走不出，请求在路上时关不掉，组字的 Escape 不关，焦点回到打开它的控件）各有一个会失败的端到端检查，在原型中由变异证明（附录 A.2）；离开工作区的入口逐个角色核对 nerve 的规则（唯一的管理员被拒绝，确认框留着，A-M2）；每一处可见的不同写进前端改动清单（W17，附录 A.8）。契约只改说明的文字（2.13 的 K24，不改形状）；Go 只改测试；没有新的 npm 包或 Go 模块，`@nerve/propel` 多一个对 `@nerve/utils` 的工作区内依赖（2.11）。

---

## 1. 目标

M3 设计第 12 节给收尾的三件事，和负责人加的两件：

1. **负责人的两件**：`ModalCore` 改用 Headless UI v2 的 `Dialog` 的打开方式（P10 的缺陷 ii：弹窗打开时后面的页面照旧能操作、能读到；负责人 2026-10-10），`AlertModalCore` 的 `tabIndex={1}` 随之去掉（P11 第 3 节第 3 条）；侧边栏的工作区菜单加"离开工作区"，每个角色都有（负责人 2026-10-11："菜单加入口，收尾做"）。
2. **各 Phase 留给收尾的事**：P1–P11 的 spec 第 5 节、评审第 6 节、修复轮的搁置清单、浏览器核对和前端改动清单中写着"收尾"的每一条，逐条分诊（2.4）：收尾做（一个 Task 和它的检查）、交给一个 M（写进它的 `handoffs/M3-closeout.md`，说明为什么属于它）、不做（写明理由）。文档写着"收尾决定"的，第 3 节给出建议，由控制者裁定。
3. **收尾的文档**：M3 收到的 6 份交接逐项写结论并关闭（2.1）；13.2 写成 M4–M8 的 5 份交接（第 4 节）；3.20 逐行、差异清单按第 4 节逐列核对（2.2、2.3）；README 按 8.7；规模估计与实际的对比（附录 A.6）；第 14 节的证据（2.15、附录 A.7）。第 14 节、第 15 节和总体设计 9.4 的状态翻转在控制者的评审提交里（M2 收尾的 D1），本 spec 和计划只准备证据。

完成线在第 5 节。

---

## 2. 分诊

### 2.1 M3 收到的 6 份交接

每份交接末尾加"处理结果（M3/收尾）"一节，`status` 改为 `closed`（Task 15）。

| 交接 | 条目 | 结论 | 证据 |
|---|---|---|---|
| `M0-P3-pagination-components.md` | 分页的组件名 | M3 的列表都是集合型的，不分页（M3 设计 3.12、11.3）；`Limit`、`Cursor`、`NextCursor` 只由 M2 的 `listApiTokens` 引用；页大小的规则移到 `shared` 一事原样交给 M4 | `api/modules/identity.yaml`；`apitest` 的组件名检查随 `make test` 通过；M4 的交接第 9 节 |
| `M1-P2-trim-content.md` | 项目字段、侧边栏偏好、保留的工作区地址、个人主页 | 四项都在 M3 做到 | P4a（`api/` 里没有 `close_in`、`default_state`、`page_view`、`estimate_id`、`/sidebar-preferences/`，收尾在 `49190525` 上 `git grep` 核对）；P2、P9、W8；P1、P8a；决策点 3 的 A、P11、C10。个人主页上工作项列表的规则行交给 M4（M4 的交接第 3 节） |
| `M1-P3-trim-platform.md` | 不再读的字段、项目成员、保留地址、地址的写法 | 四项都在 M3 做到 | P4a（`git grep` 没有 `anchor`、`is_form_enabled`）；P4b 的 `addProjectMembers`、P8b 的 `join`；P1、P8a；P4a、P8b |
| `M1-P4-router-native.md` | 保留的工作区名；离开项目的顺序 | 两项都做到 | `reserved_slugs.txt` 前后端一份（P1、P8a 的 vitest 和 `bootstrap` 的 Go 测试）；P10 的离开先等接口成功（vitest、P5）；收尾的离开工作区入口经同一个命令（2.7） |
| `M1-closeout.md` | 死成员和死 prop；oxlint | 两项都做到，剩下的每一行有出处 | P11 的 `--rows M3` 产品代码 80 → 54（P11 spec 附录 A.9）；收尾在最后一个代码 Task 之后重跑 218 → 225 行（附录 A.5）；误归的 41 行写进 M4、M6、M7 的交接；`always-return` 全仓清零（P11），上限再调低（web 303 → 295、`@nerve/ui` 15 → 12、`@nerve/propel` 11 → 10）；收尾改到的文件中仍有警告的 8 个：手改的 3 个改掉，其余 5 个按接收者写进 M4、M6、M7、M8 的交接（第 3 节第 2 条的裁定，Task 15） |
| `M2-closeout.md` | 十四节 | 十三节完成，第 12 节（页大小的规则）原样交给 M4；第 14 节的复制在 M4、M6、M7 的 M2 收尾交接中改为经 `useCopyLink`、`useCopyText` | 交接末尾的十四行表；P8a–P11 的评审；C4、C7、C8；M4 的交接第 5、9 节，M5 的交接第 1 节 |

另：M4–M8 的五份 `M1-closeout.md` 的"按规则清一类"写明 `always-return` 已由 M3/P11 全仓清零（P11 spec 第 5 节收尾一行，修复轮的 F-10）；M4、M6、M7 的 `M2-closeout.md` 的复制一节的关闭条件改为经 `useCopyLink`（`useCopyText`），并说 `copy-hook` 的例外随之删除（2.10）。

### 2.2 3.20 逐行

3.20 的 26 行都已在标出的 Phase 同步；收尾逐行找到落点（`$M3TMP/closeout-tools/inv/`，在 `49190525` 上只读核对）：总体设计的十处（1.1 的两处、3.4、3.5、3.6、4.2、5.3、6.2、6.3、6.5、7.7）都在，M2 设计 3.11 第 1 条的修订在，差异清单的 13 行见 2.3，前端改动清单 3.1、3.2 的 M3 一行和三行删除在，README 的"部署""安全""前端"三节有 8.7 的各条。核对中找到并由收尾补上的三处：

- README 中侧边栏的工作区菜单写着"切换、创建工作区、退出"：收尾加了"离开工作区"之后改为"切换、创建工作区、离开工作区、退出"（Task 3，与入口同一个提交）。
- README 的两处"（页面：P9；此前用接口）"（邀请、重新复制邀请链接）写于 P9 之前：改为"（或经接口）"（Task 16）；差异清单第四节邀请链接一行的"（页面：P9；…由 P8–P11 删除）"同样改为已发生的说法（Task 16）。
- 数据库的 `LC_CTYPE`：P7b spec 第 7 节说标签名不分大小写按数据库的 locale 折叠，部署说明写进 README 由收尾做（P7b 评审第 6 节把发布前的核对给 M8）。README 的"部署"一节加一条（Task 16），M8 的交接第 2 节核对。

"收尾"一行（总体设计 9.4 改为"已完成"）是控制者的评审提交（2.16）。

### 2.3 差异清单逐列

`$M3TMP/closeout-tools/inv/planediff.py` 把差异清单第二节 M3 的九张表与 M3 设计 4.2–4.10 逐列对照、把 4.11 的 32 行与差异清单第四节逐行对照（`logs/planediff-T16.txt`）：

- 九张表的列数都对得上（`workspaces` 14 → 10、`workspace_members` 17 → 10、`workspace_member_invites` 13 → 11、`workspace_user_properties` 14 → 10、`projects` 36 → 22 另加 `last_issue_sequence` 共 23、`project_members` 16 → 11、`project_user_properties` 15 → 11、`states` 18 → 14、`labels` 15 → 12）；设计中每个有差异的列和删除的列都在差异清单的那张表的行里，`projects`、`states` 的 `external_id`、`external_source` 由第二节"全局"一行写明（`plane-diff.md:58`）。
- 4.11 的 32 行中 31 行在第四节有同名的一行；"集合型的列表"由第三节的接口一行写明（`plane-diff.md:188`）。
- 收尾只改差异清单第四节邀请链接一行的 Phase 说法（2.2）。

### 2.4 各 Phase 留给收尾的事

来源：P1–P11 每份 spec 的第 5 节和评审的第 6 节（逐份读全，`$M3TMP/closeout-tools/read/`）、`rows.txt` 的 71 行（写着"收尾"的行，起点不是清单）、P8b–P11 的 `fix-wave-parked.md`、P9–P11 的浏览器核对、前端改动清单。"去处"一列：收尾做（Task 的编号），交给 M（交接的节），不做（理由），或第 3 节的条目（"收尾决定"的建议和规则的偏离，控制者已裁定）。

| # | 来源 | 事项 | 去处 |
|---|---|---|---|
| O1 | 负责人 2026-10-10；P10 的缺陷 ii；P11 第 3 节第 3 条、F-18 | `ModalCore` 改用 v2 的打开方式：后面的页面 `inert`、`aria-hidden`；`AlertModalCore` 的确认按钮和快捷键的搜索框不再是正的 `tabIndex`；保持 F-1、组字的 Escape；P6、P7 的删除确认框加 Tab 循环 | Task 1、2（2.5、2.6） |
| O2 | 负责人 2026-10-11；P11 第 3 节第 5 条 | 侧边栏的工作区菜单加"离开工作区"，每个角色；唯一的管理员被拒绝，确认框留着 | Task 3（2.7） |
| K1 | P9、P10、P11 的 spec 第 5 节和评审第 6 节；P9 的浏览器核对；P11 的 C9 | M3 页面上写死的英文（三份清单、`confirm-workspace-member-remove.tsx`、状态组的标题读成小写的键、设置侧边栏的标题、命令面板的占位、规模的选项、邀请页的标题、工作区菜单的角色、没有权限的页面标题、中文的"早上 晚上"） | Task 4、5、6（2.8）；收尾自己的扫描另找到的在 Task 12、13 |
| K2 | P9 第 5 节（搁置的 P6）、P10 第 5 节；P9 的存活变异 `T4f.M3a` | 角色名只有一个来源：`ROLE`、`getUserRole` 删除，五处经 `ROLE_DETAILS` 的 `i18n_title` | Task 6（2.8）；`T4f.M3a` 由 S7 的邀请页一测发现（变异 `T6.16`） |
| K3a | P11 第 5 节 | 状态、标签的颜色弹层（react-color 的 `TwitterPicker`）的十六进制输入框行内写了 `outline: none` | Task 8（2.9） |
| K3b | P11 第 5 节 | 没有名字的 27 处 propel `IconButton`（M2 1、M3 6、M4 8、M6 6、M7 6） | Task 7：在根上改，`aria-label` 必传（第 3 节第 1、4 条） |
| K3c | P11 第 5 节 | 看不到轮廓或 Tab 到不了的五处和 `member-columns.tsx:72` | M3 的四处 Task 8；`activity-filter.tsx` 交给 M4（交接第 7 节），`favorites-menu.tsx` 交给 M7（交接第 3 节） |
| K3d | P11 第 5 节 | 项目下拉框的根可以聚焦、没有名字；`issues/select/base.tsx` 同样 | 项目下拉框 Task 8；`issues/select/base.tsx` 交给 M4（交接第 7 节） |
| K3e | P11 第 5 节 | 侧边栏的两个外层手柄按钮 | 项目一行的 Task 8；收藏夹的交给 M7（交接第 3 节） |
| K3f | P11 第 5 节 | 状态的表单打开时焦点在名称上 | Task 8 |
| K3g | P10 评审第 6 节（浏览器核对） | 图标的连字（"home Launch"）和表情并进名字 | Task 7（`Logo` 的图标 `aria-hidden`） |
| K4 | P11 第 5 节、评审第 6 节 | `ButtonAvatars` 的 `showTooltip`（M3–M7 的 9 个文件 10 处，必传、没有读者） | Task 9（一次删） |
| K5 | P10 第 5 节、第 3 节第 8 条 | API 令牌弹窗和命令面板的复制经 `useCopyLink`；之后一条"复制只在 hook 里"的规则 | Task 9；规则带 20 个有期限的例外（第 3 节第 5 条） |
| K6 | P11 评审第 6 节（搁置的 B） | 五处 `error instanceof ApiError && error.problem?.code` 合成一个 `problemCode()` | Task 9 |
| K7 | P9 第 5 节（F-11） | 主题切换的 `setPromiseToast` 的失败一支与 `useRefusalToast` | Task 9（第 3 节第 6 条） |
| K8 | P9 评审第 6 节 | 关键词规则 `sign-out-toast` 的第二个写法 | Task 9（第 3 节第 7 条） |
| K9 | P10 评审第 6 节（浏览器核对） | `aria_labels.app_sidebar.*` 在两种语言中都没有定义 | Task 7：两个工作区菜单的、两个扩展侧边栏的四个键在两种语言中加上（Task 9 的 `tools/i18n-keys.mjs` 在 `49190525` 上找到的正是它们和一个日期筛选的键）；Task 7 为新的名字另加 `aria_labels` 的五个和 `sidebar` 的两个 |
| K10 | P10 第 5 节（F-11） | 输入法的 9 处守卫只查 `isComposing`：一个共用的判断加 `keyCode` 229 | Task 10（第 3 节第 8 条） |
| K11 | P10 评审第 6 节（浏览器核对） | `CustomSelect` 的按钮上按 Enter 提交外面的表单 | Task 10（第 3 节第 9 条） |
| K12 | P10 第 5 节（F-10）、P11 第 5 节 | 只读工作区的 36 处 router store 读（是否改读路由参数决定 `project-address-route` 的范围） | M3 的 19 个文件 Task 11 和规则 `workspace-address-route`；M4 4 个、M7 8 个、M8 3 个交给它们（第 3 节第 10 条） |
| K13 | P8b 评审第 6 节（PF-L5）、P9 第 5 节（PF16）、P10 第 5 节（T3-a）、P11 评审第 6 节 | 常驻的文案检查：没有读者的键（按完整的键路径）；读了却没有定义的键 | "读了却没有定义"由 Task 9 的 `tools/i18n-keys.mjs` 做，进 `make lint-web`；没有读者的键不做常驻检查（第 3 节第 11 条） |
| K14 | P10、P11 第 5 节（F-15） | `e2e/fixtures/assert/identity.ts`（485 行）拆分，`expectNoAddress` 移到 `browser.ts` | Task 14 |
| K15 | P7a 评审第 6 节（FW-a、FW-c） | `platform/jobs` 的测试：时间的余量改为期限；定期任务的测试按 `ScheduledAt` 排序 | Task 14 |
| K16 | P7a 评审第 6 节（FW-b） | 项目的 HTTP 测试中 `identifier_taken`、`name_taken` 的回答写成常量 | Task 14 |
| K17 | P7a 评审第 6 节（P12） | 故事中内联的 `createPAT(api, (await register(…` | Task 14：`registeredPAT(api, email)`，23 个故事 |
| K18 | P10 评审第 6 节 | `project-member.store.test.ts:43` 的资料改用 `userOf` | Task 14 |
| K19 | P10 第 5 节（F-10） | 用 `e2ekills.py` 重查之前各 Phase 的变异记录中端到端的"发现" | 收尾做（只是核对，附录 A.2）：P9 有一个假发现，同一个变异在别的运行中真被发现 |
| K20 | P10 第 5 节 | `webui` 给嵌入的文件加 `ETag` | 交给 M8（交接第 5 节；第 3 节第 12 条） |
| K21 | P10 第 5 节 | `@nerve/ui` 下拉框的 `any`（泛型改到 13 个调用方） | 交给 M4（交接第 8 节；第 3 节第 13 条） |
| K22 | P10 第 5 节（F-2） | `projectMemberIds` 的 getter 在 M4、M7 的读者改完之后删除 | 交给 M4（交接第 8 节）、M7（交接第 4 节），最后改完读者的那一个删除 getter（第 3 节第 14 条） |
| K23 | P8a 修复轮的搁置 P11 | `Workspace.organization_size` 收紧到数据库的 CHECK | 契约的改动：交给 M5（交接第 2 节；负责人 2026-10-13 确认，3.2 第 1 条） |
| K24 | P7a 第 5 节（A1） | 契约中说到工作项的句子扫一遍 | Task 14（只改说明的文字和由它生成的文件；第 3 节第 15 条） |
| K25 | P7b 评审第 6 节（B1） | 裁定 B1（契约按模块一个文件）写进设计 | Task 16（M3 设计 5.1） |
| K26 | P7b 第 5 节、评审第 6 节 | 部署说明写数据库的 `LC_CTYPE` | Task 16（README）；M8 核对（交接第 2 节） |
| K27 | P7b 评审第 6 节 | 架构师的复现中跑 gen-same | 不做：P7b 起每个 Phase 的复现都跑（本收尾的 `gates.sh` 的 `gen-same`） |
| K28 | P11 评审第 6 节 | Firefox、Safari 中的焦点 | C11 在 WebKit 和 Firefox 中各看一遍弹窗和离开的入口；其余页面交给 M8 的发布核对（交接第 4 节；第 3 节第 16 条） |
| K29 | P11 评审第 6 节 | `isEditable`、`disabled` 可选而默认值没有用 | 不做：评审写明"不算发现" |
| K30 | P9 评审第 6 节（浏览器核对） | 打开已拒绝的邀请链接的任何人都看到"You declined this invitation." | Task 4：改为不对读者说话的一句，"This invitation was declined." / "这份邀请已被拒绝。"（负责人 2026-10-13，3.2 第 2 条；W5，变异 `T4.14`） |
| K31 | P10 评审第 6 节 | 最后一个正的 `tabIndex`（M4、M6、M7）改完之后删除 `getTabIndex` 和 tab 序号的常量 | 交给 M4、M6、M7（交接的键盘一节：M4 第 7 节、M6 第 3 节、M7 第 3 节），最后一个删去正的 `tabIndex` 的 M 删除 |
| K32 | P11 评审第 6 节 | M4–M8 各自的 `AlertModalCore` 调用方的 A-M2；它们页面上 `overflow-hidden` 下的焦点轮廓 | 交给各 M（各交接） |

收尾自己找到的（每条写明落点）：

| # | 发现 | 去处 |
|---|---|---|
| F1 | 只在打开时才挂载的弹窗，后面的页面照旧能操作：Headless UI 的 `Dialog` 在挂载之后才找到它的主树（`MainTreeProvider` 的隐藏 `span` 经回调 ref 写入状态，晚于 `Dialog` 自己的 `inert` 副作用，而后者的依赖不再变）。31 处条件挂载（`inv/condmount.tsv`） | Task 1：`ModalCore` 从挂载之后的那次渲染打开（2.5） |
| F2 | Headless UI v2 的 `Dialog` 在没有 `autoFocus` 的元素时把焦点放在弹窗自己身上：API 令牌的新建弹窗打开时焦点不在 Title 上（Plane 原来也不在） | 不改（行为与原来相同）；C11 看一遍 |
| F3 | 焦点回到带提示的图标按钮时，提示随键盘焦点打开，下一次 Escape 先关提示 | 不改（提示的规则）；S6 核对这个顺序 |
| F4 | 侧边栏渲染两份（主侧边栏和浮出的副本）；副本透明、不接指针，但不 `inert`：Tab 和读屏都走到看不见的那一份；收起的侧边栏同样 | Task 8：不显示的一份 `inert`（S8） |
| F5 | 九处 `IconButton` 嵌在菜单自己的按钮里（按钮里有按钮，两个停留点） | M3 的项目一行 Task 8；M4 3、M6 2、M7 3 交给它们（交接第 7 节、M6 第 3 节、M7 第 3 节） |
| F6 | 侧边栏项目一行的 `Disclosure.Button` 名为"Open project menu"，每一行都一样 | Task 8：删去 `Disclosure.Button` 的 `aria-label`，按钮以项目为名（`Logo` 的图标 `aria-hidden`，K3g），展开箭头照旧名为打开、收起；S8 以项目的名字找到一行（变异 `T8.13`）；P1、P2 中与侧边栏的一行同名的卡片和面包屑改在页面内容中找（2.9） |
| F7 | Base UI 的 `Toast.Close` 是 `aria-hidden="true"` 而 `tabindex="0"`：Tab 能到、读屏读不到 | 交给 M8（交接第 4 节）：它在 Base UI 里，不在 nerve 的代码里 |
| F8 | `@nerve/ui` 的 `AlertModalCore` 的按钮默认值是英文（`@nerve/ui` 不依赖 i18n） | M3 的两个删除确认框和 M2 的令牌确认框传经 `t()` 的文字；其余调用方交给各 M，最后改为必传交给 M8（第 3 节第 17 条） |
| F9 | 日期都按英文格式化（首页的日期、日期筛选的弹窗、项目筛选中自定义日期的标签） | 交给 M8（交接第 3 节）：全应用一个决定 |
| F10 | 命令面板打开之后约 200 毫秒之内输入的字被清空（打开时的重置，Plane 原有） | 交给 M8（交接第 4 节） |
| F11 | 页头（每个工作区页面）的帮助菜单、账户菜单的按钮里又是按钮，帮助菜单的没有名字；通知链接没有名字、提示是英文"Inbox" | 帮助菜单、账户菜单交给 M8（交接第 4 节）；通知链接交给 M7（交接第 3 节） |
| F12 | `FilterHeader` 的展开按钮没有名字，`FilterOption` 的选中状态读屏读不到（M4 的组件，M3 的筛选也用） | 交给 M4（交接第 7 节） |
| F13 | 收集箱、收藏夹、视图、Webhook 的页面上的英文；`app/not-found.tsx` 的英文 | 交给 M7、M8（各交接） |
| F14 | `useProjectColumns`、`useMemberColumns` 的 `content` 表头从不渲染（`thRender`） | 不做：渲染的是 `thRender`，`content` 只是表格的数据结构 |
| F15 | 默认状态的名称（新项目的 Backlog、Todo 等）是 nerve 写进数据库的英文 | 不做：是数据，不是界面的文案；负责人 2026-10-13 决定照旧用英文，不交出（3.2 第 3 条） |
| F16 | 工作区的地址（第 12 条）之外，stores 的 router getter 照旧读 router store（`joinedProjectIds`、`workspaceProjectIds`、项目筛选的 `currentWorkspace*`、成员 store 的 `workspaceMemberIds` 等） | 不做：与 P10 对项目一侧的结论相同，store 的 getter 是读者的派生，页面经路由参数；写进 2.12 |
| F17 | 设置页的手机菜单（`settings/mobile/nav.tsx`，`relative z-50`）在手机的宽度画在每个 `ModalCore` 弹窗（`z-30`）之上；命令面板改经 `ModalCore` 之后也被它盖住（预检的 L8） | Task 2：菜单的容器、弹层和按钮降到 `z-20`，页面上的元素在弹窗的层之下（2.6）；M4 页面上 `z-50` 的元素写进 M4 的交接第 7 节，全应用的层叠顺序写进 M8 的交接第 2 节（Task 15） |

### 2.5 弹窗：`ModalCore` 的重写（Task 1）

**原来的样子**（`@nerve/ui` 的 `modals/modal-core.tsx`）：`Transition.Root` 包着 `Dialog`，`Dialog` 只在打开时挂载。Headless UI 2.2.10 的 `Dialog` 在挂载时把它的"主树"之外标为 `inert`，主树由 `MainTreeProvider` 的一个隐藏 `span` 经回调 ref 找到，晚于 `Dialog` 自己的 `inert` 副作用；那个副作用的依赖不再变，所以第一次打开时后面的页面既不 `inert` 也不隐藏，只在弹窗里的另一层（下拉框、表情）开关之后才变（P10 的缺陷 ii；原型的探针 `spike/zz-spike-dialogs.spec.ts`：建项目的弹窗中表情弹层关上之后，Tab 从弹窗走进后面的侧边栏）。

**改法**：`Dialog` 一直挂载，`open` 由调用方的 `isOpen` 决定；过渡改用 `DialogBackdrop`、`DialogPanel` 的 `transition`（`data-closed`、`data-leave` 的类，时长照旧 300/200 毫秒）。`ModalCore` 自己从挂载之后的那次渲染才打开（`mounted` 状态在副作用里设为真，`open={isOpen && mounted}`）：调用方只在打开时才挂载的弹窗（31 处，`inv/condmount.tsv`，例如成员页的移出确认框）也是先挂载、后打开，`Dialog` 这时已找到主树（F1）。组字时的 Escape：面板的 `onKeyDown` 在 Escape 是组字的时阻止默认，Headless UI 的 `useEscape` 因此不关（冒泡的阶段：弹窗里的控件先拿到它，选择的列表照旧收起自己）；Task 10 把判断换成共用的 `isComposing`（2.11）。`handleClose` 照旧可选：不传时 Escape 和点外面都不关（9 个调用方：M3 的建项目和邀请成员，M2 的 API 令牌的新建，其余是 M4、M6–M8 的，附录 A.3）。

**保持的规则，各有会失败的检查**（附录 A.2 的 `T1.*`）：

| 规则 | 检查 |
|---|---|
| 打开时后面的页面 `inert`、`aria-hidden`，关上之后都恢复 | `e2e/fixtures/dialogs.ts` 的新函数 `pageBehindHidden(page)`（主树的 `body` 子元素有 `inert` 和 `aria-hidden`）：P4（删除项目，第一次打开即隐藏）、P6、P7（删除确认框）、W7（成员页的移出确认框是条件挂载的，扣住移出的请求时仍隐藏，`T1.6`）、S6 |
| Tab 在弹窗里转 | P6、P7 的删除确认框：从取消起按 Tab，次序是删除、取消，再回到删除（`tabCycle`；原来确认按钮的 `tabIndex={1}` 让 Tab 先到它，走过最后一个就离开弹窗，`T1.2`） |
| 请求在路上时关不掉（F-1） | 已有的 `closedByEscape`、`closedByBackdropClick` 在扣住请求时（P4、P6、P7、W7，`T1.4`） |
| 组字时的 Escape 不关 | 已有的 `closedByComposingEscape`（P4 的删除项目，`T1.5`） |
| 关上之后焦点回到打开它的控件 | P4、P6：关上之后打开它的按钮有焦点 |
| Escape 和点外面关上（传了 `handleClose` 的） | 已有的 `closedByEscape`、`closedByBackdropClick`（`T1.3`） |

**弹窗之外渲染的东西**（brief 的"portals become inert"；附录 A.3）：Headless UI 只把主树标为 `inert`（`Dialog` 的 `useInertOthers` 只取主树的 `body` 子元素），`body` 的其余子元素被当作"根容器"：焦点陷阱和点外面的判断都允许它们。弹窗里打开的列表、选择器和提示经 `createPortal` 或 Base UI 的 portal 渲染成 `body` 的另一个子元素，所以照旧能点、能读。原型逐类核对（附录 A.3）：`CustomSearchSelect` 的面板（添加项目成员）、`CustomSelect` 的列表（邀请的角色、建项目的网络）、Base UI 的表情弹层（建项目）、提示的区域（`role="region"`、`aria-live`）都不 `inert`、不 `aria-hidden`，能点；弹窗留着时被拒绝的提示照旧显示并被读出（A-M2；`dialogs.ts` 的新函数 `announced` 看提示在不在 `aria-live` 的区域里）。

**W8 的一处检查跟着改**：W8 的页面版本原来在项目导航的弹窗打开时用 `getByRole` 读后面的侧边栏；后面的页面现在 `aria-hidden`，`getByRole` 读不到它，所以改为关上弹窗之后再读（检查改了，页面没有改）。

`AlertModalCore` 的确认按钮去掉 `tabIndex={1}`（`@nerve/ui` 的上限 15 → 14：它的那条 `tabindex-no-positive` 没有了）。

### 2.6 直接用 Headless UI 的三个弹窗（Task 2）

命令面板（`power-k/ui/modal/wrapper.tsx`）、快捷键（`power-k/ui/modal/shortcuts-root.tsx`）和切换账户（`onboarding/switch-account-modal.tsx`）原来直接用 Headless UI 的 `Dialog` 和 `Transition`，有 2.5 的同一个缺陷。三个都改经 `ModalCore`：

- 快捷键：搜索框去掉 `tabIndex={1}`（P10 的 F-18），打开时焦点照旧在搜索框上：`autoFocus` 照 Task 12 的项目筛选写明理由留着（`oxlint-disable-next-line`，第 3 节第 2、20 条），这个文件的两条警告都没有了，web 的上限 303 → 301。标题和搜索框的占位经 `t()`（已有的 `keyboard_shortcuts`，新键 `power_k.search_shortcuts`）。关闭按钮有名字（`common.close`），放在 `DialogTitle` 之外、标题的旁边：原来它在标题里，弹窗的名字是"Keyboard shortcuts Close"；S6 断言弹窗的名字正是标题（变异 `T2.5`）。标题随之左对齐（原来居中，附录 A.8）；
- 切换账户：退出的请求在路上时关不掉（F-1：`handleClose` 在 `switchingAccount` 时为空）；层级随 `ModalCore` `z-20` → `z-30`，宽度 `w-[40rem]` → `sm:max-w-2xl`（附录 A.8）。邮箱两边的说明原来由两个键拼成（`description_before_email`、`description_after_email`）：Task 4 改为一个带 `{email}` 的键，经 `Translated`（它在 Task 4 才有），两个旧键在两种语言中删除（vitest `switch-account-modal.test.tsx`，变异 `T4.12`）；
- 命令面板：层级随 `ModalCore` 是 `z-30`（原来它的 `Dialog` 是 `z-50`）；web 中 `z-40` 的只有看板拖动时的删除区和右键菜单的项（M4 的页面），都不与面板同时出现；`z-50` 的页面元素见下面的"层叠的顺序"。它只在打开时挂载，`ModalCore` 的 F1 修法同样适用。命令面板自己的命令文字是共用的英文，不在收尾的范围里（P11 的英文范围是 M3 的页面），写进 M8 的交接第 3 节（全应用的文案）。
- API 令牌的新建弹窗原来传 `handleClose={() => {}}`，与不传相同，删去。

新的故事 S6（`e2e/stories/smoke/s6-dialogs.spec.ts`）：命令面板和快捷键列表是模态的（打开时后面的页面 `inert`、隐藏，Tab 不出去，Escape 关上，页面恢复），快捷键弹窗的名字正是它的标题；在设置弹窗上再打开撤销令牌的确认框：设置弹窗在它后面 `inert`、隐藏，Escape 只关上面的一个，焦点回到原处，然后先关那个按钮的提示（F3），再关设置弹窗。A6 核对切换账户的弹窗在退出被扣住时关不掉。

两处没有写成变异，理由写在这里：提示不经 `BaseToast.Portal`（`AppProvider` 把它渲染在 `body` 下，它本来就是 `body` 的另一个子元素，从不在主树里）；命令面板只在打开时挂载（`ModalCore` 的 F1 修法已由 `T1.6` 看住）。

**层叠的顺序**（预检的 L8，F17）：每个 `ModalCore` 弹窗是 `z-30`；弹窗里的下拉框经 portal 渲染，也是 `z-30`，靠 DOM 的次序显示在弹窗之上，所以不抬高 `ModalCore`。规则：**页面上的元素在弹窗的层（`z-30`）之下**。设置页的手机菜单（`settings/mobile/nav.tsx`，容器 `relative z-50`）在手机的宽度画在每个弹窗之上（它 `inert`、`aria-hidden`，只是看得见）；命令面板原来是 `z-50`，改经 `ModalCore` 之后也被它盖住。收尾取修法 (b)：Task 2 把菜单的容器、弹层和按钮降到 `z-20`，注释写明这条规则。条件先核对过（`$M3TMP/closeout-tools/zprobe/`）：

- `zgrep.py`（`zgrep-before.txt`、`zgrep-after.txt`）：设置页的路由和它们用到的组件目录中 `z-20` 及以上的只有标签颜色的弹层（`z-20`，在 `z-10` 的 `Popover` 之内）、页头（`z-[27]` 和它里面的 `z-30`、`z-20`）、项目页标签导航的右键菜单（经 portal，`z-30`，不在设置页上）和这个菜单的三处 `z-50`（之后是 `z-20`）。个人设置是弹窗，没有这个菜单。
- 探针 `zz-zprobe.spec.ts`（跑完删除，不在 plan 中；`before.json`、`after.json`）：在 390×844 打开 12 个设置页（工作区的 general、成员、Webhook，项目的 general、成员、状态、标签、迭代、模块、视图、收集箱、自动化）的菜单，`elementFromPoint` 在弹层的 11×11 个点和按钮的中心都落在弹层里，`z-50` 时和 `z-20` 时相同；`z-20` 及以上的其余元素（顶部的加载进度条、页头）都不与弹层重叠。
- 截图（`shots/before-*.png`、`after-*.png`）：打开的菜单之前之后相同；命令面板和成员页的邀请弹窗打开时，之前菜单的按钮画在弹窗之上，之后被弹窗盖住。

S6 不加 `elementFromPoint` 的检查：后面的页面 `inert`，`elementFromPoint` 不命中 `inert` 的元素，菜单在 `z-50` 时它在按钮的中心照旧给出弹窗，这个检查看不出缺陷（探针核对过）；证据是截图。M4 的交接第 7 节写出 M4 页面上 `z-50` 及以上的元素和这条规则，M8 的交接第 2 节的发布核对加全应用的层叠顺序（Task 15）。

### 2.7 离开工作区的入口（Task 3）

**nerve 的规则**（`POST /api/v0/workspaces/{slug}/leave`，M3 设计 9.2、P5a）：每个有效成员都能离开，访客在内；唯一的管理员离开时 nerve 拒绝（`workspace.sole_admin`，409）；是某个有其他成员的项目的唯一管理员时也拒绝（`project.sole_admin`，409），整个离开回滚。后一条没有故事从离开的入口走到，由接口一层的 Go 测试看着（P5a、P5b）：`server/internal/bootstrap/ending_test.go` 的 `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（离开的一支得到 409 `project.sole_admin`，每张表一行不变），前一条是同一个文件的 `TestTheOnlyAdminCannotLeave`，两个问题码到 HTTP 的回答在 `workspace/adapter/http/members_test.go` 的 `TestLeaveWorkspaceRefusals`。页面经同一条拒绝的路径给出 nerve 的说法（`error-messages.ts` 把 `project.sole_admin` 映射为 `errors.project_sole_admin`），确认框留着，与 W11 的唯一管理员走的相同。

**页面**：侧边栏的工作区菜单（`workspace/sidebar/workspace-menu-root.tsx`，只在页头 `navigation/top-navigation-root.tsx` 用）加"离开工作区"一项，每个角色都有。它打开成员页离开时的同一个确认框（`ConfirmWorkspaceMemberRemove`，经新文件 `workspace/leave-workspace-modal.tsx`），经同一个命令（`useMembershipChanges` 的 `leave`）：成功之后只在发出它的会话里跟进（`followInSession`），去成员页离开之后去的地方（另一个工作区，或没有工作区时新建工作区的页面）；nerve 拒绝时确认框留着，原因在共用的拒绝提示里（A-M2）。确认框挂在菜单之外（菜单在选中一项时关上，挂在里面的确认框会跟着没有，`T3.5`）。菜单原来有一个没有调用方的"sidebar"写法，删除（216 → 205 行）。菜单的按钮在键盘取得焦点时显出 `@nerve/ui` 的选择框那样的内圈轮廓（Headless UI 的 `data-focus`；菜单的根裁掉按钮之外的部分，所以在边内），新建工作区、离开工作区、退出三项在方向键或指针选中时显出底色（`data-focus:bg-layer-transparent-hover`；原来只在指针悬停时，方向键选中的一项、Tab 到达的按钮都看不出来：Task 3 的实现者找到，控制者裁定在 Task 3 改），菜单中工作区的一项（`workspace/sidebar/dropdown-item.tsx`）同样：别的工作区的显出悬停的底色，当前工作区的从 `bg-layer-transparent-active` 深一级到 `selected`（原来当前的那一项选中时不变，而键盘打开菜单时先选中的正是它；控制者同样裁定在 Task 3 改）。

**A-M2 在成员页**：`useMembershipChanges` 的 `remove`、`leave` 和 `useInvitationChanges` 的删除收一个 `done`，只在成功的跟进中调用；`ConfirmWorkspaceMemberRemove` 不再自己关上。成员页的移出、离开和邀请的删除被拒绝时，确认框留着（原来关上；邀请的一行与成员的一行共用这个确认框，收尾找到）。离开的确认框的标题经 `t()`（`workspace_settings.settings.members.leave_title`）。

**确认框的对象**（Task 3 的修复轮，评审的 I1；负责人 2026-10-13 的 B，第 3.2 节第 4 条）：浏览器的后退、前进（工具栏、鼠标的侧键、Android 的返回手势）不受 `inert` 和弹窗的约束。`[workspaceSlug]` 的布局原来不以 slug 为 key，项目的两个布局不以项目为 key，从 /acme 后退到 /beta（或从一个项目后退到另一个项目的同一种页面）时其下的组件留着，开着的确认框也留着，作用于路由新指向的那一个：菜单的离开在 Beta 上问"离开工作区？"并离开 Beta，成员页的移出在 Beta 的成员页上移出，设置页的归档归档 Ops。修复轮 1 让离开和移出两个确认框各自记住打开时的工作区；它的清点（Task 3 的报告）在 M3 的弹窗中另找到 7 个同样的和 3 个靠输入名称挡住的（第 3.2 节第 4 条）。负责人选 B，在根上改一次：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`：页头和它的工作区菜单、`GlobalModals`、页面）以路由的 `workspaceSlug` 为 key；项目的页面（`(projects)/projects/(detail)/[projectId]/layout.tsx`）和项目的设置（`(settings)/settings/projects/[projectId]/layout.tsx`）整个返回的内容以路由的 `projectId` 为 key（项目页的标签栏和它的离开确认框在 `ProjectAuthWrapper` 之外，所以 key 在返回的根上，不只在 `Outlet` 上）。key 取自路由的参数，不取自 store；每个 key 有一句注释说明为什么（总体设计 7.7 的"状态按 id"：一次切换，后退、前进也是，不把弹窗或状态带进另一个工作区或项目）。换工作区、换项目时其下重新挂载，开着的确认框和组件的状态都不带过去。修复轮 1 的两处各自的记法撤回，菜单和成员页回到普通的状态。key 结束的是组件的状态；开关放在 store 里的弹窗在 key 之上，要由渲染它的宿主结束（修复轮 3，再评审的 I1；控制者的裁定：受范围约束的界面状态随它的 key 结束，store 持有的开关由宿主在卸载时关上）。`ProjectsAppPowerKProvider`（`(projects)/layout.tsx`、`(settings)/layout.tsx`）在工作区的 key 之下、项目的 key 之上，自己是命令面板的宿主，又渲染另外三个宿主，它们的开关都在命令面板的 store（`command-palette.store.ts`）里：M3 的 `modals/workspace-level.tsx` 的 `WorkspaceLevelModals` 渲染创建项目（`isCreateProjectModalOpen`；项目页的 Add Project、卡片列表、侧边栏的空列表、首页的没有项目、设置的没有项目、power-K、新手引导打开它），它的一个 `useLayoutEffect` 清理在它随工作区的内容卸载时、下一个工作区绘制之前关上它渲染的这一个开关（原来后退到 Beta 的项目页时这个确认框又打开，在 Beta 里创建项目：再评审的探查）；M4 的 `WorkItemLevelModals`（创建、删除工作项）和 M6 的 `ProjectLevelModals`（创建迭代、模块、视图）在项目的 key 之上，是它们的 Plane 文件，按名字交给它们（Task 15：把宿主移到项目的 key 之下或在项目的范围结束时关上开关，并在卸载时关上；接上页面时加后退的检查）。`GlobalModals` 的个人设置弹窗的开关也在 store 里，但它属于账户、不属于工作区，跟到别的工作区无害，不算 key 盖住的。命令面板（power-K，收尾的 Task 2 的弹窗）和它的快捷键列表的开关在 power-K 的 store（`power-k.store.ts`）里，原来开着时后退到别的工作区，它们在那里重新打开（修复轮 3 的探查），面板还可能显示它在前一个工作区找到的（搜索的结果、最近的页面、它的页），选中一项就在 Beta 的页面上作用于 acme。同一条规则（控制者的裁定：每个 store 持有的弹窗开关由它的宿主在卸载时关上）：`power-k/projects-app-provider.tsx` 的 `ProjectsAppPowerKProvider` 以同样的 `useLayoutEffect` 清理关上面板（`isPowerKModalOpen`）并清掉它的页（`activePage`，面板自己关上时由它的弹窗清掉，宿主卸载时弹窗已经不在），`power-k/global-shortcuts.tsx` 的 `GlobalShortcutsProvider` 关上快捷键列表（`isShortcutsListModalOpen`），注释写明这条规则。嵌套的 id（`[issueId]`、`[cycleId]`、`[moduleId]`、`[viewId]`、`browse/[workItem]`、`profile/[userId]` 等）没有 key，各 M 接上那一页时以页面自己的 id 给它的子树 key（交接写明）。M4–M8 的交接各加一句：属于一个工作区或项目的界面状态放在这些布局之下，不要放到它们之上（Task 15）。

**已经发出的请求**：切换之前已经发出的请求，切换之后照旧跟进。例如离开的请求在路上时后退到别的工作区，nerve 答应之后离开的跟进（`followInSession` 的 `navigate`）照旧执行，去成员页离开之后去的地方。这是接受的行为：请求的对象在发出时就定了（离开的是发出时的工作区），跟进只是它的结果；确认框在请求路上时关不掉（F-1），后退带走的只是确认框，不是请求。

**切换时的请求和看得到的不同**：经菜单换工作区、经侧边栏换项目、项目设置之间的移动和它们的后退，改动之前之后发出同一组请求，没有新的（W2，附录 A.4）；重新挂载带来的看得到的不同写在附录 A.8。

**测试**：两个 hook 的 vitest 照新的 `done`（成功时调用、被拒绝时不调用、换账户之后不调用）；W11 的页面版本：访客从菜单离开（键盘：按钮显出轮廓，`T3.9`；打开时先选中的当前工作区的一项深一级，`T3.12`；End、上、Enter 到达这一项，方向键选中的一项显出底色、别的不显，`T3.8`；扣住离开的回答时双击"离开"只发出一次请求，确认框的取消不可用，Escape 关不掉它，变异 `T3.7`），他的成员关系结束，页面去新建工作区；唯一的管理员从菜单离开被拒绝，原因在提示里，确认框留着，取消之后焦点回到菜单的按钮（`T3.10`：打开它的按钮在弹窗开着时重建，焦点就回不去），他的另一个工作区 Beta 的一项在方向键选中时显出底色（`T3.11`）；他从 Beta 经菜单到 acme 打开离开的确认框，后退到 Beta 时它关上、前进回来不再打开；从 Beta 的成员页经应用的链接到 acme 的成员页打开移出 gus 的确认框，一次后退两步回到 Beta 的成员页时它关上（两者看着工作区的 key：`T3.13` 去掉它，菜单的一例先失败；成员页的一例在同一个 key 下，没有自己的改法，原来的 `T3.14` 收进 `T3.13`；"前进回来不再打开"没有能单独让它失败的变异：key 一换，子树重新挂载，确认框的状态不在子树之外，后退时关上的确认框前进时只能是新的）；P3：Web 的设置的归档确认框开着，后退到 Ops 的设置（同一个布局）时关上（`T3.16`，项目设置的 key）；P8：访客在 Docs 的页头打开离开的确认框，后退到 Web 的工作项（同一个布局）时关上（`T3.17`，项目页面的 key）；W11 另从 Beta 的项目页到 acme 的（路由自己的移动）打开 Add Project 的创建确认框，后退到 Beta 的项目页时它关上（`T3.18`：宿主卸载时不关上，即修复轮 2 的样子）；又从 acme 的项目页以各自的键打开命令面板和快捷键列表，后退到 Beta 的项目页时各自关上（两个宿主，各一个检查：`T3.19` 去掉面板的宿主的清理，`T3.20` 去掉快捷键列表的）；W7：唯一的管理员在成员页离开被拒绝时确认框留着；W4：删除一个已被删除的邀请被拒绝时确认框留着。**W2（每个角色的挂载清单）**：菜单在管理员、唯一的管理员、成员、访客下都有这一项，首页的挂载清单不变，打开菜单不发请求（一次性的清单，附录 A.4；W7、W11 的页面版本看成员、唯一的管理员和访客；"打开菜单不发请求"由 S2 常驻看住：`mounts.ts` 的 `WORKSPACE_MENU`，`T3.15`）；W3（会话切换）：离开的跟进经 `followInSession`，hook 的 vitest 的"换账户之后"一行。

### 2.8 M3 页面上的英文和角色名（Task 4、5、6、12、13）

M3 页面上写死的英文都经 `t()`，en 和 zh-CN 各加同样的键（`make lint-web` 的键一致检查）。分五个 Task：工作区的页面（Task 4）、项目的弹窗和设置（Task 5）、状态和标签的设置页与角色名（Task 6）、项目列表页和它周围（Task 12）、项目设置的标题和其余页面（Task 13）。第 12、13 个 Task 由收尾自己的扫描找到（`inv/hardcoded.py`、`english2.py` 对 M3 的页面：P9–P11 的清单之外的英文）。

**一句话里带名字**：一个键是一整句，名字是占位（`{name}`），中文的语序由文案决定；名字照旧加粗。新文件 `components/common/translated.tsx`（`Translated`：把键的文字按占位切开，占位换成调用方给的节点，`Children.toArray` 给它们 key；vitest `translated.test.tsx`）。原来几处把英文的前后两半拼在名字两边。

**角色名只有一个来源**（K2）：`ROLE_DETAILS[role].i18n_title`；`@nerve/utils` 的 `ROLE`、`getUserRole`（`permission/role.ts`、`permission/index.ts`）删除。工作区成员的一列、邀请的一行、项目成员的下拉、编辑器的提及卡片（`editor/embeds/mentions/user.tsx`）、工作区菜单都经它。

**几处顺手的改动**：首页的问候原来由两个词拼成（"good"+"morning"，名字是英文语序的名和姓），中文的"good"也译作"早上"，读作"早上 下午""早上 晚上"：改为每个时段一个带 `{name}` 的键（"Good morning, {name}"、"早上好，{name}"），名字是显示名，与 Task 4 其余带名字的句子相同（S7，变异 `T4.13`）；已拒绝的邀请页的标题改为不对读者说话的一句（"This invitation was declined." / "这份邀请已被拒绝。"，负责人 2026-10-13，3.2 第 2 条；W5，变异 `T4.14`）；状态组的标题原来是小写的键（"backlog"），读屏读到的是键：改为组的名字（`STATE_GROUPS[g].i18n_label`，C9）；规模的选项和 general 页的规模经 `ORGANIZATION_SIZE_I18N_LABEL`（`@nerve/constants`）；两处装饰的图片 `alt=""`；忙时的"Leaving..."、"Joining..."改用共用的"Leaving"、"Joining"（W17）；项目卡片上访客看到的"Joined"、不是成员的人看到的"Join"按钮经 `t()`（`project_dialogs.joined`、`common.join`），设置的链接只有一个 `aria-hidden` 的图标，原来没有名字，现在以项目为名（"{name} settings"，与侧边栏的一行、"Remove X"同一个做法；`card.test.tsx`，变异 `T5.15`–`T5.17`、`T5.23`）；删除和离开项目时要输入的词只在常量 `CONFIRMATION` 里，输入框的提示和离开的不一致提示经 `{confirmation}` 说出它（原来英文的离开提示让人输入"leave project"，比对却要"Leave Project"；P4、P5，变异 `T5.20`、`T5.21`）；成员下拉框的"无"按页面上的词搜索（S7，变异 `T5.19`）；"Delete Label" → "Delete label"；邀请页、新建工作区、新手引导的标题经 `PageHead`（布局的英文 `meta` 删除）；提示区域的名字由 Base UI 的英文"Notifications"改为经 `t()`（`@nerve/propel` 的 `Toast` 收 `label`，`app/provider.tsx` 的 `AppToast`）。

**`AlertModalCore` 的按钮**：它的默认值是英文，`@nerve/ui` 不依赖 i18n（F8）。M3 的两个删除确认框（状态、标签）照 M2 的令牌确认框传经 `t()` 的文字；其余调用方在 M4–M8（第 3 节第 17 条）。

**项目的筛选**（Task 12）：负责人和成员两个筛选原是同一个文件的两份（`lead.tsx` 是 `members.tsx` 的副本），删去 `lead.tsx`，`FilterMembers` 收 `title`；筛选的名字由 `@nerve/constants` 的 `PROJECT_FILTER_I18N_LABEL` 给下拉框和已选筛选的一栏共用；网络和创建日期的选项按页面上的名字搜索；已选筛选的一栏中每个移除按钮有名字（"移除 X"）；成员筛选的按钮里不再嵌一个按钮（`CustomMenu` 的 `customButtonClassName`，Task 8 的同一个改法）。

**删除的**（W13）：`project/form-loader.tsx`（项目的包装层只在有项目时显示设置页，加载的分支走不到）、`project/settings/helper.tsx`（功能列表的开关包了一层，把每个开关都叫"Toggle project feature"，Task 13 直接用 `Switch` 并以功能为名）、两个只为 Plane 的英文写的键（`workspace_settings.token_not_copied` 等，2.10）、`@nerve/constants` 的一个没有读者的日期选项（2.10）。

**测试**：新的故事 S7（`s7-chinese.spec.ts`，5 个页面测试，Task 4 起逐 Task 加或扩展）：语言是中文的管理员看工作区的首页、菜单、项目页、general 页和成员页和它们的弹窗（Task 4：问候带显示名，快捷键弹窗的名字和搜索框是中文；之后的 Task 加侧边栏等）；项目的 general、成员、状态、标签页和它们的弹窗（Task 5、6）；邀请页（Task 6，`T6.16`）；筛选项目和成员（Task 12），同一个测试在 Task 13 加项目设置的标题、自定义范围和提示；新手引导的标题（Task 13）。vitest：改为读键的 6 个文件（`fake-i18n` 给键本身），新的 `project-dialog-texts.test.tsx`（加入、离开、移出的文字）和 `switch-account-modal.test.tsx`（切换账户的那句话是一个键，邮箱在其中；Task 4），`tab-navigation-visible-item.test.tsx`、`applied-filters/root.test.tsx`、`card.test.tsx` 加的。改了英文说法的故事（P2、P3、P5、W5）照新的说法（附录 A.8）。

### 2.9 名字和键盘（Task 7、8）

**`IconButton` 的名字在根上必传**（K3b，P11 的教训"共用组件的毛病在根上改"）：propel 的 `IconButtonProps` 的 `"aria-label"` 是必传的 `string`，`tsc` 让每个调用方传它。应用中 36 处（Task 7 改 28 处：M2 1、M3 6、M4 8、M6 6、M7 6，另有表情回应的按钮；Task 8 的项目一行删去 1 处），都有经 `t()` 的名字。表情回应的按钮收 `aria-label`（提示显示它），`EmojiReactionGroup` 收 `addReactionLabel`；回应的列表以表情为 `key`（原来以下标：propel 的那条 `no-array-index-key` 没有了，`@nerve/propel` 的上限 11 → 10）。只为名字改到的 Plane 文件 25 个（M2、M4–M7 和应用侧边栏的展开按钮），其中 22 个变长（第 3 节第 1 条的裁定，附录 A.5）。

**收尾的键盘清单**（P11 第 5 节，K3a–K3g，F4–F6）：

- 颜色弹层（状态的表单、标签的行内表单）经新文件 `components/common/color-picker.tsx`：react-color 的 `TwitterPicker` 的输入框样式里 `outline` 改回浏览器的（`"revert"`），键盘取得焦点时有轮廓（P6 核对）。
- 状态的表单打开时焦点在名称上（`setFocus("name")`，P6 核对）。
- 侧边栏的项目列表："+"按钮 `opacity-0` 加 `group-hover`、`focus-visible` 时显示（原来 `hidden`，Tab 到不了）；项目一行的菜单改用 `CustomMenu` 的省略号按钮和 `menuButtonOnClick`（菜单的按钮里不再嵌一个 `IconButton`，一个停留点；原来的 `actionSectionRef` 和它的点外面检测删除），行在 `focus-within` 时显示；拖动的手柄是 `span`，不是停留点；展开箭头 `tabIndex={-1}`（行自己的按钮做同一件事）。行自己的按钮（`Disclosure.Button`）原来名为"Open project menu"，每一行都一样（F6）：删去它的 `aria-label`，名字是项目的名字（`Logo` 的图标 `aria-hidden`，K3g），展开箭头照旧名为打开、收起。P1、P2 中按名字找的项目卡片和页头的面包屑因此与侧边栏的一行同名，改在页面内容中找（`e2e/fixtures/project-pages.ts` 的 `pageContent`）。
- 侧边栏渲染的两份（F4）：收起的主侧边栏、没有浮出的副本都 `inert`；从不设置的 `isHoveringTrigger` 状态和它的副作用删除（这个文件的一条 `exhaustive-deps` 没有了，web 的上限 301 → 300）。
- 项目成员页一行的菜单：省略号按钮名为"Options"（`common.options`），键盘取得焦点时显示（`focus-visible:opacity-100`），菜单项经 `t()`。
- 项目下拉框：停留点是它的按钮（`tabIndex` 给 `DropdownButton`，根不再可以聚焦；Headless UI 的 `ComboboxButton` 把自己设为 `tabIndex=-1`，所以里面的按钮才是停留点，与成员下拉框相同；vitest `dropdowns/project/base.test.tsx`）。
- 侧边栏的展开、收起和工作区菜单的按钮的名字原来是没有定义的键（K9）：两种语言加上；`Logo` 的图标连字和表情 `aria-hidden`（K3g），不并进链接的名字。

新的故事 S8（`s8-sidebar.spec.ts`）：副本隐藏时里面没有控件可到，收起时侧边栏没有；"+"和项目一行的菜单在 Tab 到达时显示、有焦点的标记，菜单是一个停留点；项目的一行以项目为名（变异 `T8.13`）；手柄不是停留点。Playwright 的 `getByRole` 不排除 `inert` 的元素，S8 用 `closest("[inert]")` 核对。

### 2.10 共用的写法和文案的检查（Task 9）

- **`showTooltip`**（K4）：`ButtonAvatars` 的 `showTooltip` 必传而没有读者，删除，10 处调用（9 个文件，M3–M7）。
- **复制只经 hook**（K5）：`core/hooks/use-copy-link.ts` 里一个不导出的 `useCopy(write, titleKey)`，导出 `useCopyLink`（原有）和新的 `useCopyText`；API 令牌弹窗（`useCopyText`，成功的提示照旧是"Success!"）和命令面板的"复制当前页面的地址"（`useCopyLink`；它的 `useCallback` 和一条 `eslint-disable` 随之删除）经它们；两条专用的失败文案删除（共用的提示）。关键词规则 `copy-hook`：web 应用的 `app/`、`core/` 中 `copy(Text|Url)ToClipboard(` 或 `navigator.clipboard` 只在这个 hook 里；20 个有期限的例外（M4 10、M6 5、M7 4、M8 1，`until` 是各 M 的收尾，第 3 节第 5 条：在那个 M 的收尾开始、规则文件的 `phase` 改为 `M<n>/closeout` 时到期，`tools/keywords.mjs` 的开头和 M1 设计 7.4 各加一句写明）；编辑器包不在范围内。
- **问题码只经 `problemCode()`**（K6）：`core/lib/api-error.ts` 加 `problemCode(error)`，五处 `error instanceof ApiError && error.problem?.code` 改用它（vitest）。
- **拒绝的文字只经一处**（K7）：`use-refusal-toast.ts` 导出 `useRefusalMessage`，`useRefusalToast` 和主题切换的 `setPromiseToast` 的失败一支都经它；关键词规则 `refusal-toast` 也看 `setPromiseToast(…errorMessageKey(` 的写法。
- **`sign-out-toast`**（K8）：规则加上 `{ … signOut … } = useUser()` 和 `useUser().signOut` 两种写法。
- **读了却没有定义的键**（K13）：新文件 `tools/i18n-keys.mjs`，进 `make lint-web`（和 knip 的入口）：web 应用的 `app/`、`core/` 与 `@nerve/constants`、`@nerve/types` 的源文件中 `t("…")`、`i18nKey`、`i18n_*` 的值和以已有顶层键开头的带点的字面量，每个在 en 中都有定义（模板的前缀只查前缀）；测试和替身不算。它在 `49190525` 上第一次运行就找到 K9 的四个键和 `@nerve/constants` 的 `filter.ts` 中一个没有读者、没有定义的 `date_filters.1_month_ago`（删除）。

关键词规则的 `phase` 写 `M3/closeout`。M3 设计 7.10 的规则表原来只列 P8a、P8b 的规则：Task 9 补上 P9 的三条（`workspaces-list-fetch`、`refusal-toast`、`sign-out-toast`）、P10 的两条（`project-leave`、`project-address-route`）和 `copy-hook`，Task 11 补上 `workspace-address-route`，每个 Task 结束时表是全的。A11 核对点击令牌复制到剪贴板并提示；S6 核对命令面板的复制。

### 2.11 输入法和选择上的 Enter（Task 10）

- **组字的判断只有一个**（K10）：`@nerve/utils` 的新文件 `keyboard.ts` 的 `isComposing(event)`：`event.isComposing` 或 `keyCode === 229`。Chromium、Firefox 在组字中的按键上 `isComposing` 为真；Safari 先结束组字，再交出 `keyCode` 229 的那个按键，只查 `isComposing` 的守卫拦不住它（P10 spec 第 5 节）。9 处守卫（`@nerve/ui` 的 `ModalCore`、`CustomSearchSelect`、`use-dropdown-key-down`，`@nerve/propel` 的表情选择器，web 的 `use-keypress`、命令面板等）都经它；`@nerve/propel` 因此依赖 `@nerve/utils`（`package.json` 加 `"@nerve/utils": "workspace:*"`，`pnpm-lock.yaml` 随之多 3 行；不是新的 npm 包）。命令面板的 `handleKeyDown` 开头加同一个守卫：搜索框为空时，结束组字的 Escape 原来关上面板。
- `@nerve/ui` 的 `use-dropdown-key-down` 的两条警告随之改掉（`selectActiveItem?.()`、依赖），`@nerve/ui` 的上限 14 → 12。
- **`CustomSelect` 上按 Enter 打开列表**（K11）：Headless UI 的 `ListboxButton` 在 Enter 上提交所在的表单（`attemptSubmit`）；添加项目成员的弹窗中在角色上按 Enter 就立即添加了。模块级的 `openOnEnter` 在 Enter 上先 `preventDefault`（Headless UI 合并的处理器先跑调用方的，被阻止的事件不再往下），再点它自己（`currentTarget.click()`）；`CustomSelect` 的两种按钮都用它。
- **测试**：`keyboard.test.ts`（两种顺序）；`dialogs.ts` 的 `composingEscape` 两种顺序都发（Chromium、Firefox 的 `isComposing`；Safari 的 `compositionend` 之后 `keyCode` 229 的 keydown，Chromium 照 `KeyboardEventInit` 的 `keyCode` 发出，原型里探查过）；S6：命令面板的搜索框中结束组字的 Escape 不关面板；P5：角色的选择上按 Enter 打开它的列表。

在真的 Safari 中没有核对：C11 在 WebKit 中看一遍（第 3 节第 16 条）。

### 2.12 工作区的地址（Task 11）

M3 页面上只读工作区的 19 个文件不再读路由 store 的 `currentWorkspace`（K12）：新文件 `core/hooks/use-route-workspace.ts`（`useRouteWorkspace()`：路由的 slug 经调用方的工作区列表 `getWorkspaceBySlug` 找到，没有时 `null`；vitest）。改到的：首页、工作区设置的 general 和成员页、个人主页的页头和侧边栏、首页的没有项目、项目页的根、设置侧边栏的页头、工作区菜单、`workspace-details`（它自己的一处查找也是）；工作区级的权限判断写明 slug（`allowPermissions(…, WORKSPACE, slug)`）：扩展侧边栏、个人主页的页头和布局、应用的侧边栏、成员页两处、没有项目两处、卡片列表、项目页的页头、成员的列、`useMemberColumns`、`workspace-details`、项目列表、快捷操作。项目页的根（`project/root.tsx`）的回调和副作用写全依赖（第 3 节第 2 条的裁定：副作用照旧只在挂载和换工作区时运行），web 的上限 300 → 298。

关键词规则 `workspace-address-route`（与项目一侧的 `project-address-route` 同一个写法）：这 19 个文件中没有 `currentWorkspace`，没有两个参数的 `WORKSPACE` 级 `allowPermissions`。M4 4 个、M7 8 个、M8 3 个文件同样只读工作区，交给它们（交接写明文件），对接时加进这条规则的范围（第 3 节第 10 条）。stores 里读 router store 的 getter 照旧（F16）：它们是读者的派生，页面的地址经路由参数，与 P10 对项目一侧的结论相同。

### 2.13 测试的整理和契约的说明（Task 14）

- **`identity.ts` 拆分**（K14）：会话的断言（`expectSignedIn`、`expectNewSession`、`expectAllSessionsRevoked`、`sessionOf` 等）移到新文件 `e2e/fixtures/assert/session.ts`（165 行），加 `sessionsOf(db, userId)`，`accountStateOf` 经它；`expectNoAddress` 移到 `e2e/fixtures/browser.ts`（F-15）；`identity.ts` 485 → 329 行。故事的导入随之改。
- **`platform/jobs` 的测试**（K15）：启动的期限 `startLimit` 2 秒把"River 起来了"与"没有起来"分开；定期任务的运行按 `ScheduledAt` 排序，每两次之间 1 秒；取消的上界是 `timeout + cancelGrace`；放弃的一条不再加 500 毫秒的余量。
- **项目的 HTTP 测试**（K16）：`identifierTakenJSON`、`nameTakenJSON` 两个常量，两张表都读它们。
- **`registeredPAT(api, email)`**（K17）：注册一个账户并给它一个 PAT，23 个故事用它（原来内联 `createPAT(api, (await register(…)).accessToken…)`）。
- **`userOf`**（K18）：`project-member.store.test.ts` 的资料来自它。
- **契约的说明**（K24，P7a 第 5 节的 A1）：`api/modules/project.yaml` 中说到工作项的五句（`identifier`、`default_assignee_id`、`guest_view_all_features`、两处 `archive_in`）改为说 nerve 对这些字段做什么（例如 `archive_in`："A number of months, 0–12, set for automatic archiving and kept as set; 0 for none."）。只改说明：形状、码和操作不变；`make gen` 之后 `api/dist/openapi.yaml`、`schema.gen.ts`、项目模块的 `server.gen.go` 随之变（plan 钉住它们的 SHA-256），`make gen-check` 没有差异（第 3 节第 15 条）。

`make test`（Go，42 个包）、`make lint-go` 在本 Task 跑。

### 2.14 交接和上级文档（Task 15、16）

- **Task 15**：2.1 的 6 份交接写结论、关闭；M4–M8 的 5 份 `handoffs/M3-closeout.md`（第 4 节）；M4–M8 的 5 份 `M1-closeout.md` 写 `always-return` 已清零；M4、M6、M7 的 `M2-closeout.md` 的复制一节改经 hook；M3 设计 13.2 的接收者一格链接到交接文件，表前加一句；M1 收尾交接的处理结果写上限的数（web 303 → 295、ui 15 → 12、propel 11 → 10）和仍有警告的 5 个文件的接收者。M4–M8 的交接另写（第 3 节第 2、5 条的裁定，2.6）：oxlint 留下的 5 个文件（M4 第 8 节 2 个、M6 第 4 节 1 个、M7 第 4 节 1 个、M8 第 4 节 `top-nav-power-k.tsx` 的 4 条）；关键词守卫的 `phase`（收尾开始时改为 `M<n>/closeout`，每个 Phase 开始时改为 `M<n>/P<k>`；M4 第 8 节、M6 第 4 节、M7 第 4 节、M8 第 1 节）；M4 第 7 节写 M4 页面上 `z-50` 及以上的元素和层叠的规则，M8 第 2 节的发布核对加全应用的层叠顺序，第 3 节写命令面板自己的文字。
- **Task 16**：README（"部署"加数据库的 locale，2.2 的两处"页面：P9"）；差异清单第四节邀请链接一行；M3 设计 5.1 写裁定 B1（契约按模块一个文件：`project.yaml` 1,348 行是写明的例外，由文件名得出模块的约定在 `Makefile` 和 `apitest`；K25）；前端改动清单：3.1 的 M3 一行写收尾，输入法一行（Safari 现在也拦得住）、P11 的键盘一行（36 处 `IconButton`、侧边栏的三处、成员页的菜单）照收尾之后的事实，3.2 加 7 行"完成于 M3/收尾"（附录 A.8）。

两个 Task 只改文档，没有变异；各跑 plan 的一次性脚本 `doclinks.py`：它改的文档中每个相对链接都指向存在的文件。收尾在这些文档中加的行里的路径（完整的和简写的）由原型另外核对，都找得到（附录 A.4）。

### 2.15 第 14 节逐项（证据在评审的提交里）

证据取自原型的最终树（附录 A.1 的最后一行）；评审的提交在收尾的头上重跑这些命令，照实写进第 14 节（计划的最后一个 Task 的说明）。

| # | 第 14 节 | 命令和原型上的输出 |
|---|---|---|
| 1 | 21 个故事的页面和接口版本、S1、S2 | `make e2e`：118 个通过（副本中 117 个和 S3，附录 A.1）；W1–W12、P1–P9 在 `e2e/stories/workspace/`、`project/`，S1、S2 在 `smoke/` |
| 2 | 权限矩阵、可见性、19 个交错测试、邀请和结束的成员关系的集成测试 | `make test`：42 个包 `ok`，没有 `FAIL`；`server/internal/bootstrap/permission_matrix_*_test.go`（13 个文件）、`project_visibility_test.go`、`interleaving_*_test.go`（15 个文件）和 `reactivation_races_test.go`、`invitation_rules_test.go` |
| 3 | 约定六的增长路径与停用的交错；降为访客的交错 | 同上：`interleaving_growth_test.go`、`interleaving_shrinking_test.go`、`reactivation_races_test.go`、`interleaving_roles_test.go` |
| 4 | `apitest.Main` 两个方向 | 同上：三个模块的 `adapter/http/handler_test.go`、`apitest/main_callers_test.go` |
| 5 | `access/domain` 的判定表、操作与规则表一一对应、矩阵的行 | 同上：`access/domain` 的 `TestDecide*`、`TestEveryRuleDecidesItsCells`、`TestAnActionWithoutARowHasNoRule`；`permission_matrix_coverage_test.go` |
| 6 | 持续集成的全部门禁 | `make lint-go`：两段 `0 issues.`；`make lint-web`：`keywords: 71 rules, 23 exceptions, no hits.`，`Tasks: 54 successful, 54 total`；`make knip`；`make test-web`：`Tasks: 16 successful, 16 total`；`make gen-check`：没有差异 |
| 7 | 决策点和第 11 节 | 已勾（2026-09-29） |
| 8 | 13.1 关闭、6 份交接 `closed`、13.2 的交接 | `grep -l -E '^status: (open|done)$' docs/v0/M3-workspace-project/handoffs/*.md` 没有输出；M4–M8 各有 `handoffs/M3-closeout.md`，13.2 的五行链接到它们（Task 15） |
| 9 | 3.20、差异清单、README | 2.2、2.3；README 的 8.7 各条和 locale 一条（Task 16） |
| 10 | C1–C10 写进各 Phase 的评审 | P9 评审（C1–C5）、P10 评审（C6–C8）、P11 评审（C9、C10）的第 2 节；C11、C12 是收尾的（9.7，Task 0） |
| 11 | `domains.mjs --rows M3`；oxlint 上限 | 225 行，产品代码 54 行和误归的 41 行逐行有出处（附录 A.5）；上限 web 295、ui 12、propel 10、utils 7、constants 1 |
| 12 | M3 的页面挂载时没有旧接口、没有无权的请求 | 附录 A.7 的页面与故事的表：每一页有一个 `watchPage` 的故事，断言旧接口请求、失败的请求、页面错误为空（S2 的四个角色另断言请求的清单） |
| 13 | `await` 之后的跳转、提示先核对会话 | `make test-web`：每个修改的 hook 的 vitest 中"换账户之后兑现"的一行（P9–P11 的 `lateSettlings`；收尾的离开经 `followInSession`，2.7）；`make e2e`：W3 的会话切换 |
| 14 | 规模的对比 | 附录 A.6 |
| 15 | 总体设计 9.4 | 评审的提交 |

### 2.16 状态翻转

- **第 15 节**：收尾一行在提交 spec 和计划时改为"进行中"，带两个链接（Task 0，本提交）；"已完成"和评审的链接在评审的提交里。
- **第 14 节**：每一项的勾和证据（2.15，评审时在收尾的头上重跑的输出）在评审的提交里。
- **总体设计 9.4**：M3 改为"已完成"在评审的提交里。
- **前端改动清单 3.1**：M3 一行的"收尾合并之后改为已完成"在评审的提交里改为"已完成"。
- **9.7**：控制者的两个浏览器核对在本提交加入（Task 0）：C11 重写之后的弹窗（打开时后面的页面 `inert`、`aria-hidden`，Tab 在弹窗里转，Escape、点外面关上而请求在路上时不关，组字的 Escape 不关，焦点回到打开它的控件，弹窗里的下拉框和选择器能用，被拒绝的提示在弹窗开着时被读出；焦点在 Firefox、Safari 中也看一遍）；C12 离开工作区的入口（四种角色各离开一次，键盘能到，唯一的管理员被拒绝时弹窗留着、原因被读出，落点同 W11，跟进只在离开的会话里，中文的文案）。收尾的评审也是控制者的。

---

## 3. 与设计的差异和建议（控制者已裁定，2026-10-13）

每一条写明事实、建议和理由，以及不照建议时的退路。第 1–3 条是规则的偏离，第 4–16 条是文档写着"收尾决定"的建议，第 17–22 条是交给 M4–M8 的决定和收尾自己的取舍。控制者 2026-10-13 的裁定（`$M3TMP/closeout-architect-rulings.md`）写在每条之后，预检（`$M3TMP/closeout-preflight/preflight.md`）对裁定的补充一并写进。

1. **`IconButton` 的名字必传让 25 个只为使用方改到的 Plane 文件改动，其中 22 个变长**（规则的偏离：只为使用方改到的 Plane 文件不变长）。Task 7 把 propel 的 `IconButton` 的 `aria-label` 改为必传（第 4 条），`tsc` 让每个调用方给名字。设置的 2 个（个人设置的弹窗 59 → 67，M2；工作区和个人设置共用的移动导航 55 → 56，Task 2 的层级另加一行注释，那是收尾的对象，2.6）、M4 的 10 个（评论的卡片 179 → 186、描述的版本弹窗 186 → 193、活动的排序 22 → 31 等）、M6 的 6 个、M7 的 3 个 Plane 文件和应用侧边栏的展开按钮（`sidebar/sidebar-toggle-button.tsx` 28 → 31：原来记在收尾的对象里，预检指出它也只为名字改到）共多 86 行；另 3 个同样只加名字而不变长（M4 的 `activity-filter.tsx`，M7 的通知的筛选菜单和页头选项的一项）。每个只加经 `t()` 的名字（`aria-label`；评论和工作项的回应另传 `addReactionLabel`，评论卡片的 `EmojiReactionButton` 收 `aria-label`），多数还要 `useTranslation` 的导入和取 `t` 的一行，oxfmt 把带名字的元素拆成多行（附录 A.5 逐个列出）。**建议：接受。** 名字是这些文件自己的行为（读屏原来读不到这些按钮），在根上改是 P11 的教训（共用组件的毛病在根上改），在每个 M 各补一遍就是同一个规则的几份实现，而新加的 `IconButton` 没有名字时 `tsc` 就拦住。**退路**：`aria-label` 留为可选，只给 M2、M3 的 7 处名字，其余 21 处写进各 M 的交接，再加一条关键词规则看住"没有名字的 `IconButton`"，例外带各 M 的期限。 **裁定：接受**（控制者 2026-10-13：在根上改共用组件是 P10、P11 立下的做法，退路是同一个规则在每个 M 各写一份再加守卫；条件是每个使用方的差异只加名字、`useTranslation` 的导入、`t` 的一行和 oxfmt 对那个元素的重排，预检逐个文件核对过 25 个；附录 A.5 记为写明的例外，列出 25 个文件和理由）。
2. **收尾改到的 8 个文件仍有 oxlint 警告**（规则的偏离：手改的文件结束时 0 条）。手改的 4 个：`navigation/top-nav-power-k.tsx`（Task 4 只改搜索的占位；4 条：两条 `exhaustive-deps`、可点击的 `div` 的 `click-events-have-key-events`、`prefer-tag-over-role`）、`power-k/ui/modal/shortcuts-root.tsx`（Task 2；2 → 1 条，剩搜索框的 `no-autofocus`）、`project/root.tsx`（Task 11 只改读工作区的三行；2 条 `exhaustive-deps`）、`@nerve/propel` 的 `emoji-reaction.tsx`（Task 7 加 `aria-label`；1 条 `no-array-index-key`）。只为 `IconButton` 的名字或 `showTooltip` 改到的 4 个：`description-versions/modal.tsx`（1 条 `tabindex-no-positive`，M4 的 `getTabIndex`）、`activity-filter.tsx`（2 条，可点击的 `div`）、`module-card-item.tsx`（5 条，P11 已交给 M6）、通知的筛选菜单（1 条 `no-map-spread`）。**建议**：后 4 个照 P11 的裁定 Q6 看待（同一个规则下改到的调用方，修它们会改 M4–M7 页面的 DOM 和副作用的时机，而无从核对），写进 M4、M6、M7 的交接（评审的提交加一行）；前 4 个在收尾的修复轮改掉：`project/root.tsx` 的两个副作用补上依赖（`isArchived`、`updateDisplayFilters`；它是 M3 的项目页，P4、S7 看着），快捷键的搜索框照 Task 12 的项目筛选写明 `autoFocus` 的理由（`oxlint-disable-next-line` 加理由），`emoji-reaction.tsx` 以表情为 key；`top-nav-power-k.tsx` 的四条改的是每个页面的页头（可点击的 `div` 改按钮改 DOM 和样式），交给 M8 的可访问性一节，与页头的帮助菜单、账户菜单一起改。**退路**：全部 8 个交给各自的 M（附录 A.5 的清单照抄进交接）。**裁定**（控制者 2026-10-13）：照建议，时间改为现在：手改的 3 个在修订中改掉，不留给修复轮（修复轮是给评审的发现的，计划写的文件里已知的警告是计划的事）；其余 5 个现在就写进交接。
   - `project/root.tsx`（Task 11）写全依赖：回调 `[clearAllFilters, clearAllAppliedDisplayFilters, isArchived, updateDisplayFilters, workspaceSlug]`，副作用 `[isArchived, updateDisplayFilters, workspaceSlug]`（原来是 `[pathname]`）。裁定的条件是说明依赖变时副作用做什么，预检的分析：`updateDisplayFilters` 是 `project_filter.store.ts` 的箭头字段，构造时经 `makeObservable` 包一次，引用不变；`ProjectRoot` 在项目列表和归档两个布局下各自挂载，归档的切换是一次跳转（卸载一个、挂载另一个），一次挂载之中 `isArchived` 不变，`pathname` 只随 slug 变。所以副作用照旧在挂载和换工作区时运行，不多取数、不多写 store；回调的依赖只改它的身份，并去掉一个陈旧的 `isArchived`。P4 的归档路径（列表 → 归档 → 恢复 → 列表）和 S7 的归档页看着它；行为不变，没有 W17 的行。web 的上限 300 → 298。
   - 快捷键的搜索框（Task 2）：`autoFocus` 照 Task 12 的项目筛选写明理由（`oxlint-disable-next-line`），抑制随之多一条（第 20 条）。
   - `emoji-reaction.tsx`（Task 7）以表情为 key，`@nerve/propel` 的上限 11 → 10。
   - 交出的 5 个由 Task 15 写进交接（不是评审的提交）：`description-versions/modal.tsx`、`activity-filter.tsx` 给 M4（第 8 节），`module-card-item.tsx` 在 M6（第 4 节，P11 已交给它），通知的筛选菜单给 M7（第 4 节），`top-nav-power-k.tsx` 给 M8（第 4 节：四条警告，与页头的帮助菜单、账户菜单一起改）。
   - 上限：web 303 → 301（Task 2）→ 300（Task 8）→ 298（Task 11）→ 295（Task 12），`@nerve/ui` 15 → 14（Task 1）→ 12（Task 10），`@nerve/propel` 11 → 10（Task 7）。
3. **Task 的切分：16 个**（brief 的上限是约 16 个、每个约 1,500 行）。原型原来是 15 个 Task；写计划时第 12 个（收尾的扫描找到的英文）量出 1,733 行块，加上说明会超过约 1,500 行，所以在原型中把它分成两个：Task 12（项目列表页、侧边栏、标签栏的英文）和 Task 13（项目设置的标题、功能的开关、自动归档、没有项目的设置页、新建工作区和新手引导的标题、提示的区域），各自的树都跑过门禁（附录 A.1）。之后的 Task 依次后移。最长的 Task 14 是 1,361 行（附录 A.9）。**建议：接受**（P11 第 3 节第 1 条的先例）。 **裁定：接受**（在上限上，不超过；P11 第 3 节第 1 条的先例）。
4. **K3b：`IconButton` 的名字在根上必传**（P11 第 5 节写"收尾决定"）。见第 1 条。另有两处随之：表情回应的按钮收 `aria-label`（提示显示它），图标的连字和表情 `aria-hidden`（K3g）。**建议：接受**（Task 7）。 **裁定：接受。**
5. **K5：`copy-hook` 规则现在就加，带 20 个有期限的例外**。复制只经 `useCopyLink`、`useCopyText`；M4 10 个、M6 5 个、M7 4 个、M8 1 个文件还自己复制，各是 `until: "M4/closeout"` 等的例外（不是"M4"：已有的例外写 M 的名字，在那个 M 开始时到期；这一条要在各 M 的收尾之前改完）。**建议：接受**（Task 9）；**退路**：规则等最后一个 M 改完再加，交接写明。 **裁定：接受**，条件是守卫的说明在加第一个这种例外的 Task 里写明这种写法和何时到期，接收的 M 的交接写明例外的文件和到期的时刻。Task 9 在 `tools/keywords.mjs` 的开头和 M1 设计 7.4 各加一句：`until` 为 `M<n>/closeout` 的例外在规则文件的 `phase` 改为 `M<n>/closeout` 时到期，每个 M 的收尾开始时改它（每个 Phase 开始时改为 `M<n>/P<k>`）。这句话要防的陷阱：M3 的 P1–P11 中 `phase` 一直是 `M2/closeout`，Task 9 才改为 `M3/closeout`，不改 `phase` 的 M 的例外不会到期。M4 第 8 节、M6 第 4 节、M7 第 4 节、M8 第 1 节写明各自的例外的文件、"收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M<n>/closeout`"和"每个 Phase 开始时改为 `M<n>/P<k>`"（Task 15）；M3 设计 7.10 的规则表补全（2.10）。
6. **K7：拒绝的文字只有一处**。主题切换的 `setPromiseToast` 的失败一支改经 `useRefusalMessage`，`refusal-toast` 规则也看这种写法。**建议：接受**（Task 9）。 **裁定：接受。**
7. **K8：`sign-out-toast` 规则的第二个写法**。加上解构和直接取 `signOut` 的两种。**建议：接受**（Task 9）。 **裁定：接受。**
8. **K10：组字的判断只有一个，含 `keyCode` 229**。`@nerve/utils` 的 `isComposing`，`@nerve/ui`、`@nerve/propel`、web 的 9 处守卫都经它；`@nerve/propel` 因此依赖 `@nerve/utils`（工作区内的依赖，`pnpm-lock.yaml` 多 3 行，不是新的 npm 包）。M4 的 Enter 守卫在静态上跟着变（同一个判断）。**建议：接受**（Task 10）。**退路**：各包各写一份判断（三份同一个规则）。 **裁定：接受。**
9. **K11：`CustomSelect` 上按 Enter 打开列表，不提交表单**。**建议：接受**（Task 10；W17）。 **裁定：接受。**
10. **K12：只读工作区的地址经路由**。M3 的 19 个文件改用 `useRouteWorkspace()`，新的关键词规则 `workspace-address-route` 看住它们；M4 4 个、M7 8 个、M8 3 个文件交给它们（交接写明文件和"对接时加进规则的范围"）；stores 的 router getter 不改（F16）。**建议：接受**（Task 11）。 **裁定：接受**（交接写明每个文件和"对接时加进规则的范围"）。
11. **K13：文案的常驻检查只做"读了却没有定义"**。`tools/i18n-keys.mjs` 进 `make lint-web`（它找到的一类正是 K9 的缺陷，不需要基线）。"没有读者的键"按完整的键路径：原型量出 en 的 1,330 个键中 73 个（`49190525`）、80 个（收尾之后：多的 7 个经模板读，如 `greeting.*`）没有一个源文件写出全名，其中 57 个是命令面板的模板读的；做成常驻检查要先给这 80 个一个白名单，而 `i18norphans.py` 逐 Phase 的核对（收尾之前之后没有新的孤儿键）已够。**建议：接受，不加没有读者的常驻检查**；`deadsym` 照旧是一次性的工具。 **裁定：接受。**
12. **K20：嵌入的文件的 `ETag` 交给 M8**。它是 Go 的改动，有没有用要在发布的性能核对时量（每个标签页一次 HEAD 和一次下载）。**建议：交给 M8**（交接第 5 节）。**退路**：收尾加一个小的 Go Task（`webui` 的处理和它的测试）。 **裁定：交给 M8**（要在发布的性能核对时量，那里是它的去处）。
13. **K21：`@nerve/ui` 下拉框的 `any` 交给 M4**。改成泛型要改 13 个调用方，多数在 M4–M7 的页面上，那里的故事还没有。**建议：交给 M4**（交接第 8 节）。 **裁定：接受。**
14. **K22：`projectMemberIds` 的 getter 交给 M4、M7**，最后改完读者的那一个删除它。**建议：接受**（M4 第 8 节、M7 第 4 节）。 **裁定：接受。**
15. **K24：契约中说到工作项的五句只改说明的文字**（brief："收尾不改契约，除非某一项证明需要"）。P7a 的裁定 A1 要扫一遍：`project.yaml` 的五个字段的说明原来说"工作项会怎样"（M4 才有工作项），改为说 nerve 对字段做什么。形状、码、操作都不变，`api/dist/openapi.yaml`、`schema.gen.ts`、项目模块的 `server.gen.go` 随 `make gen` 变（计划钉住 SHA-256），`make gen-check` 没有差异。**建议：接受**（Task 14）。 **裁定：接受。**
16. **K28：Firefox、Safari 中的焦点**。收尾的弹窗和离开的入口在 C11、C12 中由控制者在 WebKit 和 Firefox 中各看一遍（9.7 的两行写明）；其余页面交给 M8 的发布核对（交接第 4 节）。原型中只有 Chromium：Safari 的组字顺序由合成的事件核对（2.11）。**建议：接受**。 **裁定：接受。**
17. **`AlertModalCore` 的英文默认值交给 M8**（F8）。`@nerve/ui` 不依赖 i18n；M2、M3 的三个确认框传经 `t()` 的文字，M4–M7 的调用方由各 M 照做（交接写明），最后一个改完之后把这几个 prop 改为必传、删去默认值（M8 的交接第 3 节）。**建议：接受**。**退路**：现在就改为必传，M4–M7 的 9 个调用方传经 `t()` 的文字（它们的 Plane 文件变长，同第 1 条）。 **裁定：接受**（退路要多改 9 个 Plane 文件，而它们的文字 M4–M7 对接时会重写）。
18. **日期的本地化交给 M8**（F9）。首页的日期、日期筛选、自定义日期的标签都按英文格式化，是全应用的一个决定。**建议：交给 M8**（交接第 3 节）。 **裁定：接受。**
19. **页面上的其余可访问性交给各 M**：Base UI 的 `Toast.Close` Tab 能到、读屏读不到（F7，M8）；页头的帮助菜单、账户菜单的按钮里有按钮（F11，M8）；通知链接没有名字、提示是英文（F11，M7）；筛选的共用件的展开按钮和选中状态（F12，M4）；嵌套在菜单按钮里的 8 个 `IconButton`（F5，M4、M6、M7）；命令面板打开之后约 200 毫秒之内的输入被清空（F10，M8）。**建议：接受**（各交接写明文件和改法）。 **裁定：接受。**
20. **`project/dropdowns/filters/root.tsx` 的 `autoFocus` 写明理由留着**（Task 12）：下拉框打开时焦点在它的搜索框上，像组合框那样；这一条 `jsx-a11y/no-autofocus` 用 `oxlint-disable-next-line` 加理由（删除的 `lead.tsx` 带走了同样的一条；快捷键的搜索框照它多一条（第 2 条的裁定），命令面板的一条 `eslint-disable` 随 Task 9 删除，抑制的总数照旧 181，附录 A.4）。**建议：接受**。**退路**：去掉 `autoFocus`（打开筛选之后要再按一次 Tab 才到搜索框）。 **裁定：接受**（组合框的写法）；快捷键的搜索框照它（第 2 条的裁定）。
21. **README 现在写数据库的 `LC_CTYPE`**（K26）。P7b 的 spec 说由收尾写进部署说明，P7b 的评审把发布前的核对交给 M8；收尾写一条（Task 16），M8 照发布的部署文件核对。**建议：接受**。 **裁定：接受。**
22. **不做的**：K27（P7b 起每个复现都跑 gen-same）；K29（P11 的评审写明不算发现）；F14（`content` 表头只是表格的数据）；F2、F3（行为与原来相同，C11 看一遍）。**建议：接受**。 **裁定：接受。**

### 3.2 给负责人的（负责人 2026-10-13 已答：前三条照控制者的建议，第 4 条选 B）

1. **`Workspace.organization_size` 收紧到数据库的取值**（K23；P8a 修复轮的搁置 P11）：数据库的 CHECK 只收前端的六个选项，契约是任意字符串；收紧是契约的改动（响应引用一个 `OrganizationSize` 的枚举）。收尾不改契约，交给下一个改 `workspace.yaml` 的 M（M5 的交接第 2 节，它改工作区的图标）。请负责人确认这个去处，或另定。**负责人 2026-10-13：确认交给 M5**，M5 的交接第 2 节照旧。
2. **已拒绝的邀请链接对谁都说"You declined this invitation."**（K30；P9 的浏览器核对）：任何打开这个链接的人（不只是被邀请的人）都看到"你已拒绝"。nerve 的回答只说邀请已被拒绝，不说是谁打开的；页面的说法是产品问题（例如"这个邀请已被拒绝"）。**负责人 2026-10-13：改为不对读者说话的一句，收尾做。** `workspace_invitation.declined.title` 两种语言改为"This invitation was declined." / "这份邀请已被拒绝。"（Task 4，邀请页的英文在它里面；W5 照新的说法，变异 `T4.14`），说明的一句不变；W17 的一行（附录 A.8）和前端改动清单的收尾一行写明。
3. **默认状态的名称是英文的数据**（F15）：新项目的六个默认状态（Backlog、Todo、In Progress、Done、Cancelled 和分诊用的 Triage，`project/domain/state.go` 的 `DefaultStates`）由 nerve 以英文写进数据库，中文的用户看到英文的状态名。要不要按建项目的人的语言起名是产品问题。**负责人 2026-10-13：照旧用英文**：它们是用户会改名的数据，按语言起名要建项目的契约带上语言的信号。不改，不交出。
4. **确认框跟着后退、前进进入别的工作区或项目**（Task 3 的修复轮，评审的 I1 和它的清点；2.7）：浏览器的后退、前进不受弹窗的约束，布局不以工作区、项目为 key，开着的确认框留在原地、作用于路由新指向的那一个。M3 的弹窗中这样的有 9 个：菜单的离开工作区、成员页的移出和离开、邀请成员、新建项目（侧边栏、工作区级的弹窗、扩展侧边栏三个入口）、项目导航的设置（工作区级），添加项目成员、项目成员的移出和离开、设置页的归档、自动归档的月数（项目级）；另有 3 个靠输入名称挡住、只在名称相同时出错的：删除工作区、删除项目、页头的离开项目。**A**：每个确认框各自记住打开时的工作区或项目（修复轮 1 对离开、移出的做法），9 处各一个状态的形状、一次渲染时的重置、一个检查和一个变异，名称挡住的 3 处另算；M4–M8 的每个新弹窗都要记得照做，没有结构上的保证。**B**：在根上改一次，工作区的内容以 slug、项目的页面和设置以项目为 key，3 个 key 盖住这 12 个中开关是组件状态的（新建项目只有侧边栏和扩展侧边栏的入口是）和以后放在 key 之下的；新建项目由命令面板的 store 打开的入口（工作区级的弹窗：项目页的 Add Project、卡片列表、power-K 等）由它的宿主 `WorkspaceLevelModals` 卸载时的清理结束（修复轮 3，控制者的裁定）；代价是切换时其下重新挂载（量过：没有新的请求，附录 A.4；看得到的不同在附录 A.8）。**负责人 2026-10-13：B**，在根上改一次；修复轮 1 的两处各自的记法撤回；M4–M8 的交接写明界面状态放在 key 之下（Task 3 的修复轮 2）；store 持有的开关由宿主在卸载时关上（工作区级的弹窗、命令面板、快捷键列表），M4、M6 在项目的 key 之上的宿主按名字交出（修复轮 3）。

---

## 4. 交给 M4–M8 的

M3 设计 13.2 的每一行，连同各 Phase 的 spec 第 5 节、评审第 6 节交给后面的 M 的各条和收尾找到的，写进接收者的 `handoffs/M3-closeout.md`（Task 15）。每一节写做什么、为什么在这个 M、关闭条件和来处；13.2 的接收者一格链接到它。

| M | 文件 | 节 |
|---|---|---|
| M4 | [`docs/v0/M4-issue-core/handoffs/M3-closeout.md`](../../M4-issue-core/handoffs/M3-closeout.md) | 1 事件与删除工作区的连带；2 加锁的约定和方案 E；3 权限规则和错误码（个人主页的工作项列表在内）；4 状态、标签和项目的数据；5 显示设置与旧的 service；6 stores、会话和页面的取数；7 工作项页面的键盘、下拉框和弹窗（`ModalCore` 重写之后逐个核对、正的 `tabIndex`、嵌套的按钮、`issues/select/base.tsx`、`activity-filter.tsx`、筛选的共用件、`z-50` 及以上的页面元素和层叠的规则）；8 地址、复制、文案和留下的行（`workspace-address-route` 的 4 个文件、`copy-hook` 的 10 个例外、误归的 16 行、`@nerve/ui` 下拉框的 `any`、`projectMemberIds`、oxlint 留下的 2 个文件、守卫的 `phase`）；9 页大小的规则移到 `shared`（M2 收尾交接第 12 节，原样） |
| M5 | [`docs/v0/M5-files/handoffs/M3-closeout.md`](../../M5-files/handoffs/M3-closeout.md) | 1 图标、封面和头像；2 `organization_size`（3.2 第 1 条）；3 附件的确认框和弹窗 |
| M6 | [`docs/v0/M6-cycles-modules/handoffs/M3-closeout.md`](../../M6-cycles-modules/handoffs/M3-closeout.md) | 1 取数和 stores；2 数据（误归的 8 行）；3 键盘、弹窗和下拉框；4 地址、复制（5 个例外）、oxlint 和守卫的 `phase` |
| M7 | [`docs/v0/M7-collaboration/handoffs/M3-closeout.md`](../../M7-collaboration/handoffs/M3-closeout.md) | 1 接口和挂载时的取数；2 stores 和会话（误归的 17 行）；3 键盘、弹窗和下拉框（收藏的菜单和手柄、通知的链接）；4 地址（8 个文件）、复制（4 个例外）、文案和其余（`projectMemberIds`、通知的筛选菜单的 oxlint、守卫的 `phase`） |
| M8 | [`docs/v0/M8-open-release/handoffs/M3-closeout.md`](../../M8-open-release/handoffs/M3-closeout.md) | 1 Webhook（事件、页面、3 个文件的地址、1 个复制的例外、守卫的 `phase`）；2 发布之前的核对（表情的数据、README、locale、大列表的性能、全应用的层叠顺序）；3 全应用的文案（日期、`not-found`、propel 和 `AlertModalCore` 的默认值、命令面板自己的文字）；4 可访问性（`Toast.Close`、页头和它的搜索的 oxlint、Firefox 和 Safari、命令面板的输入）；5 嵌入的文件加 `ETag` |

第 3 节第 2 条交出的 5 个文件由 Task 15 写进交接（控制者的裁定：现在写，不在评审的提交里）：`description-versions/modal.tsx`、`activity-filter.tsx` 给 M4（第 8 节），`module-card-item.tsx` 在 M6（第 4 节，P11 已交给它），通知的筛选菜单给 M7（第 4 节），`top-nav-power-k.tsx` 给 M8（第 4 节）。

---

## 5. 验收标准（完成线）

- [ ] 计划的 16 个 Task 都已提交；每个 Task 之后 `make lint-web`、`make knip`、`make test-web`、`make build`、`make e2e` 通过（Task 14 另有 `make gen-check`、`make lint-go`、`make test`），worktree 中 `make e2e` 118 个全部通过（`49190525` 是 110 个）。
- [ ] 弹窗：`ModalCore` 是 Headless UI v2 的打开方式，打开时后面的页面 `inert`、`aria-hidden`（条件挂载的也是），Tab 在弹窗里转，F-1、组字的 Escape、焦点回到打开它的控件照旧；`AlertModalCore` 和快捷键的搜索框没有正的 `tabIndex`；命令面板、快捷键、切换账户经 `ModalCore`；页面上的元素在弹窗的层之下（设置页的手机菜单 `z-20`，2.6）；P4、P6、P7、W7、S6 的检查和变异 `T1.*`、`T2.*` 都被发现（附录 A.2）。
- [ ] 离开工作区：侧边栏的工作区菜单每个角色都有这一项，经成员页的确认框和命令；唯一的管理员被拒绝时确认框留着，取消之后焦点回到菜单的按钮；双击只发一次请求，请求在路上时确认框关不掉；菜单的按钮和方向键选中的一项（工作区和三个操作）在键盘下看得见；W11、W7、W4 的页面版本。
- [ ] 2.4 的每一条有去处：收尾做的有 Task 和检查，交出去的在接收者的交接里，不做的有理由，"收尾决定"的有第 3 节的裁定。
- [ ] 6 份收到的交接 `status: closed`，各有"处理结果（M3/收尾）"；M4–M8 各有 `handoffs/M3-closeout.md`，13.2 的五行链接到它们。
- [ ] 3.20 逐行、差异清单逐列核对过（2.2、2.3）；README 按 8.7 和 locale 一条；前端改动清单的收尾七行（附录 A.8）。
- [ ] 各包的 oxlint 上限照新的条数（web 295、ui 12、propel 10）；手改的文件结束时 0 条（`top-nav-power-k.tsx` 交给 M8），第 3 节第 2 条交出的 5 个文件写在接收者的交接里；抑制照旧 181 条。
- [ ] 计划在新的副本上从本 spec 和计划的提交（修订之后的那一个）逐 Task 复现，每个 Task 之后门禁通过，最后与原型逐个文件相同（附录 A.9）。
- [ ] 控制者的 C11、C12 和收尾的评审（评审的提交里第 14、15 节、9.4 和前端改动清单 3.1 的状态翻转，2.16）。

---

## 6. 风险

- **31 个弹窗没有故事打开**（都是 M4–M8 的：M4 9 个、M5 1 个、M6 9 个、M7 10 个、M8 2 个，附录 A.3）。重写只改 `ModalCore` 自己，调用方的 props 不变；Headless UI v2 的行为（主树之外的 `body` 子元素是根容器、焦点陷阱和点外面允许它们）对所有调用方一样，原型在 M3 的页面上逐类核对了弹窗里的四种 portal。工作项的大弹窗（下拉框、日期、编辑器）在 M4 对接时逐个核对（交接第 7 节）；C11 也在 M3 的页面上看一遍。**退路**：某个调用方在 M4–M8 中发现问题时，在那个调用方修，不回退 `ModalCore`（回退就回到缺陷 ii）。
- **Safari 和 Firefox 只由合成的事件和 C11 核对**：真实的输入法、真实的 Safari 焦点的差异可能还有（M8 的发布核对）。
- **条件挂载的修法依赖 Headless UI 的内部顺序**（`MainTreeProvider` 的回调 ref 晚于 `Dialog` 的副作用）：升级 Headless UI 之后，W7 的 `pageBehindHidden`（变异 `T1.6`）会先失败。
- **`copy-hook` 的例外有期限**：各 M 的收尾之前要改完它们的文件，否则关键词守卫失败；交接写明。到期靠各 M 照时改规则文件的 `phase`（守卫的说明和交接写明）。
- **页面上的层叠**：页面上的元素在弹窗的层（`z-30`）之下是一条约定，没有常驻的检查；M4 页面上的 `z-50` 元素写进 M4 的交接，M8 的发布核对全应用看一遍（2.6）。
- **翻译的质量**：新加的中文由收尾写，S7 只核对页面读的是键、是中文；用词由 C11、C12 和评审看一遍。

---

## 附录 A：原型的验证记录（2026-10-13）

### A.1 方法与门禁

- 原型 `$M3TMP/closeout-proto` 是 `49190525` 的 `git archive` 副本（`$M3TMP/closeout-tools/base-49190525.tar`，`pnpm install --frozen-lockfile` 之后）。Task 0（M3 设计第 15 节的收尾一行和 9.7 的 C11、C12，本提交）先写进副本，`T0` 是它之后的树。每个 Task 由一个脚本从上一个快照写出（`edit/T<n>.py`，共用 `sub.py`），照仓库的格式化（oxfmt）排版，存一份快照（`$M3TMP/closeout-snap/T1`…`T16`）。计划的块由快照之间的差异写出（`mkblocks.py`：16 个 Task，0 个问题；它把每个 Task 的块依次应用到上一个快照，与那个 Task 的快照逐字节比较）。代码或测试改过时，从改动的那个 Task 起重新写出之后的每个快照（`rerun.sh`）。原型中途把原来的第 12 个 Task 分成 Task 12、13（第 3 节第 3 条），之后的脚本、快照和变异的编号各加一（`edit/renumber_scripts.py`、`renumber_mutants.py`）。预检之后的修订（2026-10-13）改的是 Task 2–5、7–12、15、16 的脚本（`amend/a_t*.py` 改写 `edit/T<n>.py`），从 Task 2 起重新写出之后的每个快照（`chain.sh`）。重写时 `restore.py` 不动生成的文件，原型中是 Task 14 生成的那一份，于是一度被带进 `T2`–`T13` 的快照；`amend/fixgen.py` 把 `T1` 的放回，之后 `gensha.py` 的表与第一轮相同（只有 Task 14 改生成的文件，钉住的三个 SHA-256 不变）。
- 每个 Task 的树上跑它的门禁（`gates.sh`）：`make lint-web`、`make knip`、`make test-web`、`make build`、`make e2e`；Task 14 另有 gen-same（`make gen` 的输出与树中的生成文件相同，即 `make gen-check` 的核对）、`make lint-go`、`make test`；Task 15、16 只改文档，跑 `make lint-web` 和 plan 的 `doclinks.py`（A.4）。`49190525` 上：`make lint-web`（69 条规则、3 个例外）、`make knip`、`make test-web` 通过，`make e2e` 110 个中 109 个通过（S3 在副本中失败：它读构建的提交，副本不是 git 仓库；P11 附录 A.1 的 F4）。
- 最终原型上（修订之后，`logs/a-final.summary`、`a-final-build.summary`）：gen-same 0 处差异；`make lint-go` 两段 `0 issues.`；`make lint-web` 通过（关键词守卫 71 条规则、23 个例外、没有命中，turbo 54 个任务）；`make knip` 通过；`make test-web` 通过（16 个任务；web 应用 113 个测试文件、982 个测试，`49190525` 上 106 个测试文件、965 个测试）；`make build` 通过；`make test` 42 个包 `ok`、没有 `FAIL`；`make e2e` 118 个中 117 个通过（S3，F4）。
- 端到端的反复：收尾写的或改了检查的 14 个故事（S6、S7、S8、W4、W7、W8、W11、P1、P4–P7、A6、A11）在最终原型上各跑 3 次（`repeat.sh`，`logs/repeat-final3.log`）：36 个测试 108 次都通过（`repeat.sh` 的退出码 0）。修订之后，brief 点名的 S6、S8、W5、W11、P4 再各跑 3 次（`logs/repeat-a-final3.log`：10 个测试 30 次都通过），修订改了检查的 S7、P1、P2 也各 3 次（`logs/repeat-a-changed3.log`：12 个测试 36 次都通过）。

| Task | 改到的文件 | web 的测试文件 / 测试 | 端到端 | oxlint 上限（web / ui / propel） | 关键词守卫（规则 / 例外） |
|---|---|---|---|---|---|
| `T0` | – | 106 / 965 | 110 | 303 / 15 / 11 | 69 / 3 |
| 1 | 9 | 106 / 965 | 110 | 303 / 14 / 11 | 69 / 3 |
| 2 | 14 | 106 / 965 | 112 | 301 / 14 / 11 | 69 / 3 |
| 3 | 18 | 106 / 965 | 112 | 301 / 14 / 11 | 69 / 3 |
| 4 | 39 | 108 / 968 | 113 | 301 / 14 / 11 | 69 / 3 |
| 5 | 27 | 109 / 971 | 114 | 301 / 14 / 11 | 69 / 3 |
| 6 | 30 | 109 / 972 | 115 | 301 / 14 / 11 | 69 / 3 |
| 7 | 39 | 109 / 972 | 115 | 301 / 14 / 10 | 69 / 3 |
| 8 | 17 | 110 / 973 | 116 | 300 / 14 / 10 | 69 / 3 |
| 9 | 38 | 110 / 977 | 116 | 300 / 14 / 10 | 70 / 23 |
| 10 | 20 | 110 / 977 | 116 | 300 / 12 / 10 | 70 / 23 |
| 11 | 24 | 111 / 979 | 116 | 298 / 12 / 10 | 71 / 23 |
| 12 | 34 | 113 / 982 | 117 | 295 / 12 / 10 | 71 / 23 |
| 13 | 37 | 113 / 982 | 118 | 295 / 12 / 10 | 71 / 23 |
| 14 | 40（另有生成的 3 个） | 113 / 982 | 118 | 295 / 12 / 10 | 71 / 23 |
| 15 | 20 | 113 / 982 | 118 | 295 / 12 / 10 | 71 / 23 |
| 16 | 4 | 113 / 982 | 118 | 295 / 12 / 10 | 71 / 23 |

（测试数按 vitest 自己的计数，`it.each` 的每一行算一个（`vitestcounts.py`），只算 web 应用；`@nerve/utils` 另有 Task 10 的 `keyboard.test.ts`。一个文件可以由几个 Task 改到，所以改到的文件之和 410 大于 303：收尾改到 303 个文件（改 275 个、新建 23 个、删除 5 个），另有 Task 14 由 `make gen` 重新生成的 3 个。端到端的数是 `make e2e` 的测试数，副本中其中的 S3 失败（F4）。`@nerve/utils` 7、`@nerve/constants` 1 的上限不变。）

### A.2 变异

`mutants_closeout.py`：146 个变异（第一轮 136 个，修订加 10 个：`T2.5`、`T2.6`、`T3.7`、`T4.11`–`T4.14`、`T7.8`、`T8.13`、`T11.5`），每个只改一处或几处（Task 15、16 只改文档，没有变异）。`mut.py` 在最终原型上、在它写的每个检查上各跑一次（修订之后由 `amend/mutrun.sh` 重跑，`mut-results.json`，日志 `mut-logs-final/`）：146 个都被发现，没有存活的；每个写的每个检查都让它失败（`T9.11` 的 `check:sync` 是写明的对照，见下）。按发现它的层：只有端到端的 87 个、只有 vitest 的 30 个、只有静态的 11 个、只有 Go 的 5 个、vitest 和端到端的 10 个、静态和端到端的 3 个。按 Task 的表：Task 1 6 个、2 6 个、3 7 个、4 14 个、5 14 个、6 16 个、7 8 个、8 13 个、9 11 个、10 9 个、11 5 个、12 17 个、13 11 个、14 9 个。每个变异另在它自己的 Task 的快照上跑过（每个 Task 写完时，端到端在内；`mut-results-t<n>.json`，合并为 `mut-results-pertask.json`；修订改过的 Task 2–5、7–12 的变异由 `amend/mutrun.sh` 在新的快照上重跑，`logs/a-mut-pertask.log`）：146 个都在自己的 Task 的树上被发现，没有只由之后的 Task 才发现的（`mut-results-from.json` 为空）。每个变异要改的文字在它的 Task 的快照上和最终原型上都恰好出现一次（`mutcheck.py`：146 个都适用；有两处要改的文字在之后的 Task 中变了（`T1.5` 的组字判断由 Task 10 换成 `isComposing`，`T4.5` 的工作区由 Task 11 改读路由），它们照 P11 的写法给出两个版本的旧文字和新文字）。`e2ekills.py` 逐个核对端到端和 vitest 的 142 个"发现"，每一个是断言的失败，不是超时、容器或构建的失败：修订之后的最终运行 0 个可疑（`logs/a-mutrun.log`；第一轮的最终运行中 `T8.4` 在 S8 上的一次"发现"是容器启动的超时，重跑之后由 S8 的断言发现，超时的日志留在 `logs/T8.4-e2e.container-failure.log`）。

`T9.11`（侧边栏读的一个键在两种语言中都改了名）在 `check:sync`（两种语言的键一致）上存活，这是写明的对照：两种语言仍一致，键一致的检查看不到它，发现它的是新的 `tools/i18n-keys.mjs`（K13）。plan 的表只列发现它的检查。

按缺陷类别（一个变异可以被几层发现）：

| 类别 | 变异 | 静态 | vitest | Go | 端到端 | 存活 |
|---|---|---|---|---|---|---|
| 弹窗：后面的页面 `inert`、隐藏，焦点回来，Escape 和点外面（2.5、2.6） | 5 | 0 | 0 | 0 | 5 | 0 |
| 请求在路上时关不掉、只发一次（F-1） | 3 | 0 | 0 | 0 | 3 | 0 |
| 组字的 Escape | 1 | 0 | 0 | 0 | 1 | 0 |
| Tab 的次序和弹窗的名字：没有正的 `tabIndex`，菜单的项，快捷键的关闭按钮 | 5 | 2 | 0 | 0 | 5 | 0 |
| 被拒绝时弹窗留着（A-M2） | 3 | 0 | 2 | 0 | 3 | 0 |
| 入口的门：每个角色都有离开 | 1 | 0 | 0 | 0 | 1 | 0 |
| 文案：P9–P11 的清单和修订加的（2.8） | 51 | 2 | 18 | 0 | 33 | 0 |
| 文案：收尾的扫描找到的（Task 12、13） | 23 | 0 | 6 | 0 | 17 | 0 |
| 名字 | 8 | 1 | 3 | 0 | 7 | 0 |
| 键盘取得焦点时看得见；停留点（2.9） | 10 | 0 | 1 | 0 | 9 | 0 |
| 输入法和选择上的 Enter（2.11） | 9 | 0 | 2 | 0 | 9 | 0 |
| 共用的写法（2.10） | 7 | 0 | 5 | 0 | 3 | 0 |
| 规则：关键词守卫、`tools/i18n-keys.mjs` | 6 | 6 | 0 | 0 | 0 | 0 |
| oxlint 的上限：写明理由的抑制、依赖、key（第 3 节第 2 条） | 3 | 3 | 0 | 0 | 0 | 0 |
| 工作区的地址（2.12） | 2 | 0 | 2 | 0 | 1 | 0 |
| 测试的整理（2.13） | 9 | 0 | 1 | 5 | 3 | 0 |
| 合计 | 146 | 14 | 40 | 5 | 100 | 0 |

- **弹窗的 5 个和请求在路上的 3 个**：`Dialog` 只在打开时挂载（`T1.1`，P4 第一次打开时后面的页面不隐藏）、条件挂载的弹窗在挂载的那次渲染就打开（`T1.6`，W7 扣住移出的请求时后面的页面不隐藏：F1 的修法由它看住）、忽略 `handleClose`（`T1.3`）、快捷键弹窗不理 Escape 和背景（`T2.3`，S6）、离开的确认框挂在菜单里（`T3.5`，W11：菜单关上时它跟着没有）；确认框和切换账户的弹窗在请求路上时被 Escape 关上（`T1.4`、`T2.4`）；离开的确认框的按钮在请求路上时不禁用，双击发两次（`T3.7`，W11）。
- **键盘的 5 个、焦点的 10 个、名字的 8 个**：确认按钮和快捷键的搜索框回到 `tabIndex={1}`（`T1.2`、`T2.1`，P6、P7、S6 的 Tab 循环，也由 oxlint 的上限）、快捷键的关闭按钮没有名字或回到标题里（`T2.2`、`T2.5`，S6 断言弹窗的名字）、离开一项是菜单之外的按钮（`T3.4`）；颜色弹层的轮廓、状态表单的焦点、侧边栏的"+"、项目一行的菜单和手柄、副本和收起的侧边栏不 `inert`、成员页一行的菜单、项目下拉框的根（`T8.*`，S8、P6、P5 和 `base.test.tsx`）；项目页的搜索、表情并进链接的名字、筛选的移除按钮、功能的开关（`T7.1`、`T7.7`、`T8.7`、`T12.13`、`T12.16`、`T13.3`、`T13.4`）、侧边栏项目一行的名字（`T8.13`，S8）。
- **文案的 74 个**（P9–P11 的清单和修订加的 51 个，收尾的扫描 23 个；修订加的 4 个：快捷键弹窗的英文 `T4.11`、切换账户的两个半句 `T4.12`、问候的名和姓 `T4.13`、已拒绝的邀请页的旧说法 `T4.14`）：每个把一处写回英文、把占位放错、读没有定义的键或删去一个语言的键，由 S7（中文的页面）、改为读键的 vitest、`check:sync`（`T7.5`：一种语言少了键）或 `tools/i18n-keys.mjs`（`T9.11`）发现。
- **oxlint 的 3 个**（第 3 节第 2 条的裁定）：快捷键的搜索框的 `autoFocus` 不写理由（`T2.6`）、表情回应以下标为 key（`T7.8`，propel 的上限）、项目页的根的副作用只依赖地址（`T11.5`）：各让 oxlint 的条数超出上限，`make lint-web` 失败。
- **被拒绝时弹窗留着的 3 个、入口的门 1 个**：成员页的确认框在 nerve 回答之后关上（`T3.1`，W4、W7、W11）、离开不论 nerve 怎样回答都关上（`T3.2`）、邀请的删除同样（`T3.6`）；菜单只给管理员这一项（`T3.3`，W7、W11）。
- **输入法和 Enter 的 10 个**：`isComposing` 漏掉 Safari 的顺序或组字中的按键（`T10.1`、`T10.2`，`keyboard.test.ts`、P1）、各处守卫不经它、命令面板的空搜索框上组字的 Escape 关上面板（S6）、`CustomSelect` 上的 Enter 提交表单（`T10.8`、`T10.9`，P5）；组字的 Escape 关上弹窗（`T1.5`，P4）。
- **共用的写法 7 个、规则 6 个、地址 2 个**：`useCopy` 的写法和失败、`problemCode`、`useRefusalMessage`（vitest、A11、S6）；`copy-hook`、`refusal-toast`、`sign-out-toast` 和 `workspace-address-route`（两个）各有一个违反它的写法（关键词守卫），页面读一个没有定义的键（`T12.17`，`tools/i18n-keys.mjs`）；`useRouteWorkspace` 不给工作区或读 router store 的（`T11.1`、`T11.2`，vitest，前者也由 W3）。
- **测试整理的 9 个**（Task 14）：`jobs` 的期限和排序（`T14.1`–`T14.3`，Go 的 `platform/jobs`）、两个常量（`T14.4`、`T14.5`，Go 的项目 HTTP 测试）、移动之后的断言照旧失败（`expectNoAddress` 由 P9、`expectSignedIn` 由 A3、`sessionsOf` 由 A12，`T14.6`–`T14.8`）、`userOf`（`T14.9`，store 的测试）。`registeredPAT` 只是把 23 个故事的同一段写法收进一个函数，由那些故事的通过核对。
- **K19**（之前各 Phase 的端到端"发现"，`inv/k19.sh` 对 P8a–P11 的变异记录跑 `e2ekills_any.py`，`logs/k19.txt`）：P10、P11 的 0 个可疑；P9 逐 Task 的记录中 `T4.2` 在 W5 上的"发现"是容器启动的超时，同一个变异在 P9 的最终和修复轮的记录中由 W5 的断言真被发现，另由 vitest 发现；P8a、P8b 的日志目录是空的（记录只有结果），无从核对。之前的结论不变。
- **W18（每个新的 vitest 都能失败）**：`w18closeout.py` 把收尾新加的 23 个 web 的测试（标题在 `T0` 的同一个文件中没有的，`it.each` 按标题算一个；修订加了 `switch-account-modal.test.tsx` 的一个）对照每个变异的 vitest 日志：19 个被某个变异的运行失败（新的一个由 `T4.12`）。其余 4 个（成员页的移出、离开在成功时关上确认框的两行，`useRouteWorkspace` 的"没有时为 null"，`isComposing` 的"其余的按键不是输入法的"）另写探查变异 `W18.1`–`W18.4`（`mutants_w18.py`，只在最终原型上跑，不在 plan 的表中）：每个都让它守着的测试失败（`logs/mut-w18.log`，日志在 `mut-logs-w18/`）；之后 23 个测试都至少被一个变异或探查失败（修订之后重跑：`logs/a-mut-w18.log`、`a-w18closeout.txt`）。W3 的探查 `W3.1`–`W3.3`（确认框的 `done` 在 `followInSession` 的核对之外调用：成员页的移出、离开，邀请的删除）：每个都让那个 hook 的"换账户之后"的两行（被回答、被拒绝）和"被拒绝时确认框留着"的一行失败（`logs/mut-w18.log`）。

### A.3 弹窗的清单和 portal 的清扫

`inv/dialogs.py`（每个 `<ModalCore>`、`<AlertModalCore>` 元素、它的 props 和所属的 M，在 `49190525` 和最终原型上各跑一次，`inv/dialogs.tsv`、`dialogs-T16.tsv`）加上直接用 Headless UI 的三个，读故事得出哪个故事打开它：

- `49190525`：52 个弹窗：`ModalCore` 的调用方 36 个、`AlertModalCore` 13 个（另有 `alert-modal.tsx` 自己的一个），直接用 Headless UI 的 `Dialog` 3 个。调用方传的 props 都在 `isOpen`、`handleClose`、`position`、`width`、`className` 之内（`AlertModalCore` 的是 `handleClose`、`handleSubmit`、`isSubmitting`、`isOpen`、`title`、`content`、`primaryButtonText`、`secondaryButtonText`），`ModalCore` 的 props 不变，所以调用方不用改。不传 `handleClose` 的 8 个（建项目、邀请成员、工作项、迭代、模块、视图、收集箱的新建、Webhook 的新建）和传一个什么都不做的函数的 API 令牌的新建（Task 2 删去，与不传相同）Escape 和点外面都不关，照旧。
- 最终原型：命令面板、快捷键和切换账户改经 `ModalCore`（Task 2），52 个都经 `ModalCore`（39 个调用方）或 `AlertModalCore`（13 个）。离开工作区的入口复用成员页的确认框（`leave-workspace-modal.tsx` 只是把它挂在菜单之外），不是新的弹窗。
- 由故事打开的：`49190525` 上 18 个（M2 的 4 个、M3 的 14 个），收尾之后 21 个（S6 打开个人设置、命令面板和快捷键，A6 照旧打开切换账户）。**没有故事打开的 31 个都是 M4–M8 的**：M4 9 个、M5 1 个、M6 9 个、M7 10 个、M8 2 个，逐个写进各自的交接（第 4 节，M4 第 7 节等）。
- 条件挂载（`inv/condmount.py`，`condmount.tsv`）：110 处使用中 31 处只在打开时才挂载（例如成员页的移出确认框、项目卡片的加入和归档、建项目、日期筛选的 9 处），都由 F1 的修法覆盖，W7 的 `pageBehindHidden` 在条件挂载的那一个上核对（`T1.6`）。

| # | 文件（`web/apps/web/core/components/` 之下） | 种类 | M | 打开它的故事 | 弹窗里经 portal 渲染的 |
|---|---|---|---|---|---|
| 1 | `account/deactivate-account-modal.tsx` | `ModalCore` | M2 | A12、W9 | – |
| 2 | `api-token/delete-token-modal.tsx` | `AlertModalCore` | M2 | A11、S6 | – |
| 3 | `api-token/modal/create-token-modal.tsx` | `ModalCore` | M2 | A11 | `CustomSelect`、`DateDropdown`、提示 |
| 4 | `settings/profile/modal.tsx` | `ModalCore` | M2 | S6（收尾之前没有） | `CustomSelect`；令牌的两个弹窗在它上面打开 |
| 5 | `onboarding/switch-account-modal.tsx` | 直接用 `Dialog` → `ModalCore` | M2 | A6 | – |
| 6 | `automation/select-month-modal.tsx` | `ModalCore` | M3 | P3 | – |
| 7 | `labels/delete-label-modal.tsx` | `AlertModalCore` | M3 | P7 | – |
| 8 | `navigation/project-navigation-dialog.tsx` | `ModalCore` | M3 | W8 | – |
| 9 | `project-states/options/delete.tsx` | `AlertModalCore` | M3 | P6 | – |
| 10 | `project/add-project-members-modal.tsx` | `ModalCore` | M3 | P5 | `CustomSearchSelect`、`CustomSelect` |
| 11 | `project/archive-restore-modal.tsx` | `ModalCore` | M3 | P4 | – |
| 12 | `project/confirm-project-member-remove.tsx` | `ModalCore` | M3 | P5 | – |
| 13 | `project/create-project-modal.tsx` | `ModalCore`（没有 `handleClose`） | M3 | P1 | 表情选择器（Base UI）、`CustomSelect`、成员下拉框、提示 |
| 14 | `project/delete-project-modal.tsx` | `ModalCore` | M3 | P4 | – |
| 15 | `project/join-project-modal.tsx` | `ModalCore` | M3 | P2 | – |
| 16 | `project/leave-project-modal.tsx` | `ModalCore` | M3 | P5 | – |
| 17 | `workspace/confirm-workspace-member-remove.tsx` | `ModalCore` | M3 | W7、W11（收尾起也经菜单） | – |
| 18 | `workspace/delete-workspace-modal.tsx` | `ModalCore` | M3 | W3 | – |
| 19 | `workspace/members/invite-modal.tsx` | `ModalCore`（没有 `handleClose`） | M3 | W4 | `CustomSelect`（每行的角色） |
| 20 | `power-k/ui/modal/wrapper.tsx` | 直接用 `Dialog`（`z-50`）→ `ModalCore` | 共用 | S6（收尾之前没有） | cmdk 的列表（不经 portal） |
| 21 | `power-k/ui/modal/shortcuts-root.tsx` | 直接用 `Dialog` → `ModalCore` | 共用 | S6（收尾之前没有） | – |
| 22–30 | `core/description-versions/modal.tsx`、`core/filters/date-filter-modal.tsx`、`issues/archive-issue-modal.tsx`、`issues/confirm-issue-discard.tsx`、`issues/delete-issue-modal.tsx`、`issues/issue-detail/links/create-update-link-modal.tsx`、`issues/issue-modal/base.tsx`、`issues/parent-issues-list-modal.tsx`、`issues/workspace-draft/delete-modal.tsx` | `ModalCore` 7、`AlertModalCore` 2 | M4 | 没有 | 日期选择器；工作项的弹窗里有各种下拉框、编辑器；`confirm-issue-discard` 在工作项的弹窗上打开；三处正的 `tabIndex`（两个 `tabIndex={1}`，一个 `baseTabIndex`） |
| 31 | `issues/attachment/delete-attachment-modal.tsx` | `AlertModalCore` | M5 | 没有 | – |
| 32–40 | `core/modals/existing-issues-list-modal.tsx`、`cycles/archived-cycles/modal.tsx`、`cycles/delete-modal.tsx`、`cycles/modal.tsx`、`cycles/transfer-issues-modal.tsx`、`modules/archived-modules/modal.tsx`、`modules/delete-module-modal.tsx`、`modules/links/create-update-modal.tsx`、`modules/modal.tsx` | `ModalCore` 7、`AlertModalCore` 2 | M6 | 没有 | 日期范围、项目和成员的下拉框、`CustomSelect`；三处正的 `tabIndex`（两个 `tabIndex={1}`，一个 `baseTabIndex`） |
| 41–50 | `inbox/modals/create-modal/modal.tsx`、`inbox/modals/decline-issue-modal.tsx`、`inbox/modals/delete-issue-modal.tsx`、`inbox/modals/select-duplicate.tsx`、`inbox/modals/snooze-issue-modal.tsx`、`views/delete-view-modal.tsx`、`views/modal.tsx`、`workspace-notifications/.../snooze/modal.tsx`、`workspace/views/delete-view-modal.tsx`、`workspace/views/modal.tsx` | `ModalCore` 6、`AlertModalCore` 4 | M7 | 没有 | 下拉框、编辑器、日历、筛选 |
| 51–52 | `web-hooks/create-webhook-modal.tsx`、`web-hooks/delete-webhook-modal.tsx` | `ModalCore`、`AlertModalCore` | M8 | 没有 | – |

**portal 的清扫**：弹窗里经 portal 渲染的有三类：(a) `createPortal` 到 `body`（`CustomSelect` 的列表、`CustomSearchSelect` 的面板、`DateDropdown`、日期范围、`CustomMenu` 的子菜单）；(b) Base UI 的 portal（表情选择器的弹层、propel 的提示和菜单、提示的区域）；(c) `#context-menu-portal`（右键菜单，不在弹窗里）。每一个都是 `body` 的另一个子元素，在主树之外。Headless UI v2 的 `Dialog` 只把主树的 `body` 子元素标为 `inert`、`aria-hidden`（`useInertOthers`），`body` 的其余子元素被当作根容器，焦点陷阱和点外面的判断都允许它们。原型的探针（`spike/zz-spike-dialogs.spec.ts`、`zz-probe-cond.spec.ts`、`zz-probe-toast.spec.ts`，跑完删除，不在 plan 中；日志 `logs/e2eone-spike-base.log`、`e2eone-spike-v2.log`、`e2eone-probe.log`）：

- 改之前：第一次打开时 `body` 的子元素都不变（缺陷 ii）；主树只在弹窗里的另一层 Headless UI 开关之后才 `inert`；建项目的弹窗中表情弹层关上之后，Tab 从弹窗走进后面的侧边栏（29 次 Tab 中都在外面）。
- 改之后：主树的 `body` 子元素从第一次打开起 `inert`、`aria-hidden`；(a) 的面板和列表、(b) 的表情弹层和提示的区域（`role="region"`、`aria-live="polite"`）都不 `inert`、不 `aria-hidden`，能点；弹窗留着时被拒绝的提示照旧显示并在 `aria-live` 的区域里（`announced`）；取消关上，之后没有 `inert`，焦点回到打开它的按钮；表情弹层关上之后 Tab 在建项目的弹窗里转（29 次都在里面）。条件挂载的探针在改之后、加 `mounted` 之前仍看到后面的页面没有隐藏（F1），加了之后隐藏。
- brief 列的其余几种：**嵌套的弹窗**由 S6 核对（设置弹窗上的撤销确认框，2.6）；**导航关上的弹窗**：P4 的归档在 nerve 归档之后跳到项目列表，之后以 `getByRole` 找到侧边栏（树是 `aria-hidden` 时 `getByRole` 找不到它）：弹窗随跳转关上时，Headless UI 撤去 `inert`、`aria-hidden`；**`initialFocus`**：没有调用方传它（`ModalCore` 没有这个 prop），F2；**关上时卸载内容的弹窗**：`Dialog` 在 `open` 为假时照旧卸载面板（不用 `unmount={false}`），与原来的 `Transition.Root` 一样，表单的状态照旧随关上丢弃；**过渡的时长**照旧 300/200 毫秒，`closedWithin`、`transitionsEnded` 等的期限不变，用它们的故事照旧通过。
- 写成故事的：P1（建项目的表情弹层、网络的选择）、P5（添加成员的搜索面板和角色）、W4（邀请的角色）在弹窗里照旧能选，W11、W7、W4 的被拒绝的提示由 `announced` 核对；S6 的嵌套弹窗（设置弹窗上的撤销确认框）。M4–M8 的弹窗里的下拉框（日期、编辑器、工作项的各种下拉框）是同两类 portal，在各 M 对接时逐个核对（交接写明）。

### A.4 缺陷类的清扫

P8 的 web 清扫 W1–W20，按收尾改到的页面和弹窗收窄（brief）；不适用的写明为什么。

- **W1（旧接口回到挂载路径）**：收尾不加请求，也不改取数的路径。S2 的四个角色的请求清单照旧通过；A.7 的每一页由一个 `watchPage` 的故事断言没有旧接口的请求。
- **W2（每个角色的挂载清单）**：侧边栏的工作区菜单（`w2/zz-w2-menu.spec.ts`，跑完删除，不在 plan 中；`w2/before.json`、`after.json`）：五个账户（acme 的管理员即创建者、第二个管理员、成员、访客；solo 的唯一的管理员）各在自己的浏览器中打开工作区的首页，再打开工作区菜单，在 `T0` 和最终原型上各跑一次。首页的挂载清单之前之后相同，五个角色一样（`instance`、`me`、`me/profile`、`me/workspaces/{slug}/preferences`、`workspaces`、`workspaces/{slug}/members`、`workspaces/{slug}/projects?archived=false`、`workspaces/{slug}/states` 和登录的 `auth/refresh`），没有 400 以上的回答，没有发往别处的请求；打开菜单不发任何请求（之前之后都是）。唯一的不同是菜单的项：每个角色都多了"Leave workspace"（之前：工作区、新建工作区、退出；之后：工作区、新建工作区、离开工作区、退出）。W7、W11 的页面版本另以 `watchPage` 断言打开菜单之后没有失败的请求、旧接口和页面错误。**常驻**（Task 3 的修复轮，评审的 M2）：一次性的清单之外，"打开菜单不发请求"由 S2 看住：挂载清单所在的 `e2e/fixtures/mounts.ts` 加 `WORKSPACE_MENU`（空），S2 的四个角色（管理员即唯一的管理员、成员、访客、不是项目成员的成员）各在最后一页打开工作区菜单，断言此后的请求正是它（变异 `T3.15`：打开时再取一次工作区列表）。**切换**（Task 3 的修复轮 2，负责人的 B；一次性的探查 `zz-r2-switch.spec.ts`，不在 plan 中，跑完删除，`$SCRATCH/r2-probe/`）：管理员有 acme（项目 Web、Ops）和 Beta（项目 Docs），在改动之前（`e6c79afc` 的代码）和之后各跑一次，记下每一次切换之后的全部请求（等到 1.5 秒没有请求、再等 3 秒，越过 SWR 两秒的去重，免得重新挂载的重新校验被盖住）。经菜单从 acme 的首页换到 Beta 的首页：`PATCH me/profile`（菜单记下最后打开的工作区），`workspaces/{beta}/members`、`me/workspaces/{beta}/preferences`、`workspaces/{beta}/projects?archived=false`、`workspaces/{beta}/states`（工作区的包装层，在 key 之外）；后退到 acme 的首页：acme 的同样四个。经侧边栏从 Web 的工作项换到 Ops 的（手风琴和标签两种导航各一次）：`projects/{ops}`、`me/projects/{ops}/preferences`、`projects/{ops}/labels`、`projects/{ops}/members`、`projects/{ops}/states` 和 M4 的 `/api/workspaces/{acme}/projects/{ops}/user-properties/`（答 404，M4 的交接第 5 节）；后退到 Web 的：Web 的同样六个。项目设置之间（`moveWithinApp`，P3 的写法）从 Web 的 general 到 Ops 的：Ops 的五个项目的请求和 `timezones`；后退到 Web 的：Web 的五个和 `timezones`。每一次之前之后同一组请求，次数相同，没有新的（只有一次的 404 的回答与请求的先后不同）：工作区的包装层在 key 之外，项目的包装层的请求原来就随项目换了 key。
- **W3（会话切换）**：离开、移出和邀请的删除的跟进（关上确认框、去别处、提示）都在 `followInSession` 的 `done` 里（2.7）；两个 hook 的 vitest 的"换账户之后"的行（`lateSettlings`，每个修改各两行）在 `done` 改法之后照旧通过，W3 的探查（A.2）让它们失败；端到端的会话切换在 P9 的 W3。
- **W4（取数经 `useSessionSWR`）、W5（按权限取数）、W6（一个接一个）、W7（`sessionGuard()`）、W10、W11（状态、标签的顺序）、W15（保留名单）、W16（取数失败）、W19（M6、M7 的取数）**：不适用，收尾不改取数、stores 的写入和保留名单；T7、T9 在 M4–M7 的文件中只加名字或删去 `showTooltip`。
- **W8（落点）**：经菜单离开之后去成员页离开之后去的地方：W11 的访客（没有别的工作区）到新建工作区的页面；唯一的管理员被拒绝时留在原页（W11）。
- **W9（权限 store）**：工作区级的权限判断写明 slug（Task 11 的 19 个文件；关键词规则 `workspace-address-route` 看住两个参数的写法）；`T11.3`、`T11.4`。
- **W12（`as`、`any`、`!`）**：`castsall.py` 对收尾改到的每个源文件：没有新的 `as`、`any`、`!`；`e2e/fixtures/assert/session.ts:67` 的 `as SessionRow` 是从 `identity.ts` 移来的（Task 14，原来在 `identity.ts:68`）。抑制（`oxlint-disable`、`@ts-expect-error` 等，`suppressions.py`，`logs/a-suppressions.txt`）181 → 181：删除的 `lead.tsx` 和命令面板的 `eslint-disable` 各少一条，项目筛选和快捷键的搜索框的 `autoFocus` 各多一条带理由的（第 3 节第 2、20 条）。
- **W13（删除的代码）**：删去的 5 个文件（`project/form-loader.tsx`、`project/settings/helper.tsx`、`project/dropdowns/filters/lead.tsx`、`@nerve/utils` 的 `permission/role.ts`、`permission/index.ts`）、`ROLE`、`getUserRole`、工作区菜单的"sidebar"写法、`isHoveringTrigger`、`actionSectionRef`、`showTooltip`、8 个没有读者、只为 Plane 的英文写的或被一整句代替的键（切换账户的两个半句由 Task 4 删去）：`git grep`（副本中 `grep -r`）没有读者，`make knip` 通过。
- **W14（文案表）**：拒绝的文字只经 `useRefusalMessage`（K7）、问题码只经 `problemCode()`（K6）。
- **W17（看得到的不同）**：A.8。
- **W18**：A.2。
- **W20（可空的字段）**：`useRouteWorkspace()` 可以是 `null`，19 个文件都处理（`tsc`；页面在没有工作区时不渲染或给 `undefined`）。
- **文档的核对**：Task 15、16 各跑 plan 的 `doclinks.py`：它们改的 23 个文档中每个相对链接都指向存在的文件（本 spec 和 plan 在本提交里，逐 Task 复现时在副本中核对，A.9）。另对收尾在它改的 24 个文档（这 23 个和 Task 9 的 M1 设计）中加的行（`inv/addedpaths.py`，`T0` 到 `T16` 的差异，`amend/docsweep.py`，`logs/a-addedpaths.txt`）：392 行中的相对链接、以仓库顶层目录开头的路径 38 个、简写的 `.ts`、`.tsx` 路径 170 个都找得到，只有 `peek-overviews.tsx`（M3 设计 13.2 和 M7 的交接写它已随"最近"小部件删除）是历史。这些文档原有的行中还有不在的路径（已删去的文件的记录，构建和本地的产物，如 README 的 `config.local.yaml`、前端改动清单的 `keys.generated.ts`），是它们记下的历史，不改；所以 `inv/links.py`、`shortpaths.py` 的整篇核对是原型的清扫，不是 Task 的检查。
- **交接的清扫**（brief）：2.1 的每个结论写明证据（提交、测试、评审的节）；13.2 的五行各链接到一个接收者的 `M3-closeout.md`，每行的每一项在那个文件的某一节（第 4 节的表）；仍在生效的文档中写于之前的将来时的句子：README 的两处"（页面：P9；此前用接口）"、差异清单邀请链接一行的 Phase 说法由 Task 16 改为已发生的说法（2.2），M1 收尾交接的 oxlint 一句照收尾之后的事实（第 3 节第 2 条）。
- **Go 的清扫 19、36、45、46、50**（Task 14 只改测试）：19、36 不适用（没有新的方法或路径）；45：改过的 `jobs` 测试和项目 HTTP 测试的常量各有变异让它们失败（`T14.1`–`T14.5`）；46：`jobs` 的测试的每个等待有期限（`startLimit` 2 秒、`timeout + cancelGrace`、定期任务每两次之间 1 秒）；50：改过的测试的注释与代码逐句核对。
- **注释与代码一致**：每个 Task 新写或改动的注释、JSDoc 与它描述的代码逐句核对；交接和前端改动清单的行与最终的代码核对（Task 15、16）。

### A.5 文件的大小、Plane 文件的变长、oxlint 和死成员

- **文件的行数**（约 400 行；`sizes.py`，`logs/a-sizes.txt`）：收尾写的文件中最长的是 `s7-chinese.spec.ts` 318 行、`s6-dialogs.spec.ts` 112 行、`e2e/fixtures/dialogs.ts` 163 → 201 行、`assert/session.ts` 165 行；`assert/identity.ts` 485 → 329 行。超过约 400 行的：`p5-project-members.spec.ts` 392 → 404 行（Task 10 加的 Enter 一段，约 400 之内）；原来就超过的 `tools/keywords.json` 2,390 → 2,617 行、两种语言的 `common.json` 528 → 539 行、生成的文件和 `project.yaml`（行数不变）；Plane 的 `workspace/sidebar/projects-list-item.tsx` 452 → 438 行。
- **变长的 Plane 文件**（brief：只为使用方改到的不变长；`planegrow.py`，`logs/a-planegrow.txt`）：96 个变长。74 个是收尾的对象（行为改在它们里面：弹窗、离开的入口、文案、名字和键盘、地址、组字的守卫、层叠），例如 `delete-label-modal.tsx` 57 → 63、`mark-as-default.tsx` 46 → 54、项目的筛选（Task 12）。

  **写明的例外**（第 3 节第 1 条，控制者 2026-10-13 裁定）：propel 的 `IconButton` 的名字在根上必传，`tsc` 让每个调用方给名字，25 个 Plane 文件只为此改到（Task 7）：22 个变长、共 +86 行，3 个不变长。理由：名字是这些按钮自己的行为（读屏原来读不到它们），在共用组件的根上改一次，新加的没有名字的 `IconButton` 由 `tsc` 拦住；在每个 M 各补一遍是同一个规则的几份实现。允许的改动：名字（`aria-label`；评论和工作项的回应另传 `addReactionLabel`，评论卡片的 `EmojiReactionButton` 收 `aria-label`）、`useTranslation` 的导入、取 `t` 的一行和 oxfmt 对那个元素的重排；预检逐个文件核对过 25 个，没有别的改动。应用侧边栏的展开按钮原来记在收尾的对象里，预检指出它也只为名字改到，记在这里：

| M | 文件（行数：之前 → 之后） |
|---|---|
| M2、设置 | `settings/profile/modal.tsx` 59 → 67；`settings/mobile/nav.tsx` 55 → 56（名字；Task 2 的层级另加一行注释，到 57，那是收尾的对象，2.6） |
| 共用 | 应用侧边栏的展开按钮 `sidebar/sidebar-toggle-button.tsx` 28 → 31 |
| M4 | `comments/card/display.tsx` 179 → 186、`comments/comment-reaction.tsx` 91 → 94、`comments/quick-actions.tsx` 101 → 106、`core/description-versions/modal.tsx` 186 → 193、`issue-activity/sort-root.tsx` 22 → 31、`issue-detail-quick-actions.tsx` 165 → 171、`reactions/issue.tsx` 145 → 148、`quick-action-dropdowns/issue-detail.tsx` 338 → 342、`peek-overview/header.tsx` 238 → 244、`rich-filters/filters-toggle.tsx` 90 → 93；不变长：`issue-activity/activity-filter.tsx` 67 → 67 |
| M6 | 迭代的页头 252 → 253、模块的页头 246 → 249、`cycles-view-header.tsx` 132 → 133、`cycles/quick-actions.tsx` 182 → 184、`module-view-header.tsx` 193 → 194、`modules/quick-actions.tsx` 173 → 177 |
| M7 | `views/quick-actions.tsx` 132 → 136、`views/view-list-header.tsx` 124 → 127、通知的页头选项 86 → 88；不变长：通知的筛选菜单 `workspace-notifications/sidebar/filters/menu/root.tsx` 43 → 43、页头选项的一项 `…/header/options/menu-option/root.tsx` 93 → 93 |

  只为 `showTooltip`（Task 9）改到的 9 个文件都变短或不变（Task 9 改到的 Plane 文件中只有主题切换变长 91 → 92，它是 K7 的对象）。
- **oxlint**（`oxsnapfiles.py`，`logs/ox-final-amend.txt`）：上限 web 303 → 295（Task 2 快捷键的两条：`tabindex-no-positive` 改掉、`no-autofocus` 写明理由；Task 8 侧边栏的 `exhaustive-deps`；Task 11 项目页的根的 2 条 `exhaustive-deps`；Task 12 两个筛选文件的 3 条）、`@nerve/ui` 15 → 12（Task 1 `AlertModalCore` 的 `tabindex-no-positive`、Task 10 `use-dropdown-key-down` 的 2 条）、`@nerve/propel` 11 → 10（Task 7 表情回应的 `no-array-index-key`），各等于条数。收尾改到的文件中仍有警告的 5 个，照第 3 节第 2 条的裁定由 Task 15 写进接收者的交接：

| 文件 | 改它的 Task | 条数 | 规则 | 交给 |
|---|---|---|---|---|
| `navigation/top-nav-power-k.tsx` | 4（手改，只改搜索的占位） | 4 | `exhaustive-deps` 2、`click-events-have-key-events`、`prefer-tag-over-role` | M8 第 4 节（与页头的帮助菜单、账户菜单一起改） |
| `core/description-versions/modal.tsx` | 7（名字） | 1 | `tabindex-no-positive` | M4 第 8 节 |
| `issue-activity/activity-filter.tsx` | 7（名字） | 2 | `click-events-have-key-events`、`no-static-element-interactions` | M4 第 8 节 |
| `modules/module-card-item.tsx` | 9（`showTooltip`） | 5 | `consistent-function-scoping`、可点击的 `div` 4 | M6 第 4 节（P11 已交） |
| 通知的 `filters/menu/root.tsx` | 7（名字） | 1 | `no-map-spread` | M7 第 4 节 |

- **死成员和死 prop**（M1 收尾计划附录 A 的 `deadsym.mjs`、`domains.mjs`，`dead/run.sh`，副本经 `fakegit` 列出文件）：`deadsym`（修订之后重跑，`dead/run2.py`，`dead/a-final.*`）：死的 prop 362 → 368、成员 975 → 987、导出 2 → 2（修订多的一行是 `switch-account-modal.test.tsx` 的替身的 `children`，它在测试里，不在 M3 的行中）（之前的数是 P11 修复轮之后的，P11 spec 附录 A.9）。`--rows M3` 218 → 225（`inv/deadcmp.py`，`dead/final.m3` 对 P11 的）：产品代码的 54 行和按路径误归的 41 行与 P11 附录 A.9 的逐行相同（收尾没有加也没有删）；测试文件中的行 123 → 130，新的 7 行是新测试的列和替身的 props（`delete.test.tsx` 的 `label`，`card.test.tsx` 的 `children`、`label`，`project-dialog-texts.test.tsx` 的 `buttons`、`dialog`、`open`、`sentences`），测试自己读它们。

### A.6 规模：7.11 的估计与实际

**怎样量**（M2 收尾附录 B 的数法）：`f8cb7c2`（M3 开始）到 `49190525`（P11 的合并），`git diff --numstat --find-renames` 和 `--name-status` 在 worktree 中各一次（`$M3TMP/closeout-tools/measure/numstat.tsv`、`namestatus.tsv`），`f8cb7c2` 的 `git archive` 解开到 `measure/m3start/`；`measure/measure.py`：删掉的文件全算"删除"，新文件全算"新增"，改动的文件取 `min(+, −)` 为"重写"，其余为删除和新增；生成的文件、语言文件和包的清单另列；web 的文件按 7.11 的文件组归类（脚本里写明每组的路径规则）。收尾自己的改动另量（`inv/closeoutsize.py`，`T0` 到 `T16`）。

**前端**（`web/`，不含生成的文件、语言文件和清单）：

| 文件组 | 7.11 的行 | 文件 | 删除 | 重写 | 新增 | 净 | 7.11 估计的删除 / 重写 / 新增 |
|---|---|---|---|---|---|---|---|
| services | 695 | 13 | 567 | 10 | 256 | −311 | 600 / – / 350 |
| stores | 3,415 | 17 | 1,121 | 1,130 | 198 | −923 | 1,700 / 400 / 700 |
| store 的 hooks | 102 | 0 | 0 | 0 | 0 | 0 | 20 / 40 / – |
| wrappers | 371 | 2 | 189 | 56 | 0 | −189 | 80 / 150 / – |
| 顶层页面 | 475 | 4 | 242 | 95 | 53 | −189 | 250 / 150 / 50 |
| 设置页的路由 | 1,737 | 13 | 45 | 80 | 46 | +1 | 100 / 300 / – |
| 项目列表和归档的路由 | 142 | 0 | 0 | 0 | 0 | 0 | – / 50 / – |
| 工作区的组件 | 5,838 | 32 | 641 | 376 | 608 | −33 | 450 / 900 / 100 |
| 项目的组件 | 5,639 | 38 | 692 | 510 | 471 | −221 | 350 / 1,000 / 100 |
| 状态、标签的组件 | 2,243 | 21 | 349 | 474 | 152 | −197 | 50 / 500 / – |
| 设置的组件 | 482 | 7 | 35 | 98 | 7 | −28 | – / 100 / – |
| 新手引导的两步 | 911 | 6 | 610 | 73 | 49 | −561 | 300 / 200 / 50 |
| 成员、项目下拉框 | 865 | 7 | 245 | 149 | 11 | −234 | – / 200 / – |
| 类型 | 319 | 3 | 187 | 2 | 0 | −187 | 250 / – / – |
| 常量 | 251 | 1 | 50 | 2 | 0 | −50 | 60 / – / – |
| **组内小计** | 23,485 | **164** | **4,973** | **3,055** | **1,851** | **−3,122** | 约 4,200 / 4,000 / 1,350 |
| 组外 | – | 288 | 7,508 | 1,420 | 1,602 | −5,906 | 400 / 700 / 150 |
| 测试（vitest） | – | 117 | 12 | 47 | 12,197 | +12,185 | 新增 3,000 |
| **合计** | | **569** | **12,493** | **4,522** | **15,650** | **+3,157** | 约 4,600 / 4,700 / 1,500 + 测试 3,000 |
| 另列：生成的 `schema.gen.ts` | | 1 | 0 | 360 | 2,360 | +2,360 | 约 2,500 |
| 另列：语言文件 | | 20 | 150 | 66 | 162 | +12 | |
| 另列：包的清单 | | 6 | 0 | 5 | 6 | +6 | |

- 组外的 288 个文件中 59 个是删去的图片（`app/assets/cover-images` 58 个、`empty-state` 1 个，−4,622 行是 SVG 的文本行；预设的封面随 M5 的封面上传加回，M3 设计 13.2），其余是跟随类型和 store 的使用方、`promise(always-return)` 的清零、首页"最近"的删除、`@nerve/ui` 的下拉框、新的 `lib/`（`reconciled.ts`、`in-session.ts`、`form-refusal.ts`、`error-messages.ts`）和 `auth-layout`。
- 改到的文件 569 个，7.11 估约 380（330–430）：组内 164 个（估约 175），组外 288 个（估约 180，多出的是封面图片和 M1 收尾交接的清理），测试 117 个（估约 25）。
- **测试**是新增的大头（12,197 行，估 3,000，约 4 倍）：P8a–P11 的 hook 和 store 的测试按会话、权限和拒绝逐行写（`lateSettlings` 等），每个修改一张表。

**后端**：

| 项 | 7.11 的估计 | 实际 |
|---|---|---|
| 生产代码（不含生成的代码） | 约 12,000 行（10,000–15,400） | 170 个文件，+12,002 / −476，净 11,526 行：`project` 5,657、`workspace` 4,561、`identity` +422 / −218、`access` 364、`bootstrap` +293 / −44、`httpserver` +258 / −189、`shared` 191、`postgres` 177、`cmd` 76 |
| 测试 | 约 24,000 行（20,000–31,000） | 253 个文件，+43,271 / −422，净 42,849 行（1.8 倍；权限矩阵和交错测试） |
| 手写的 SQL | 约 1,500 行（查询约 1,200，迁移约 300） | 1,200 行（查询 896，迁移 304；授权 3） |
| 接口描述 | 约 2,200 行（`workspace.yaml` 约 950，`project.yaml` 约 1,250） | 2,219 行（`workspace.yaml` 871，`project.yaml` 1,348）；`identity.yaml` +46 / −8 |
| 生成的 Go | – | 27 个文件，净 9,950 行 |

**端到端**：故事 30 个文件，+6,644 / −71（21 个故事和 S1、S2 的页面版本；估约 2,900，约 2.3 倍：页面版本的键盘、请求在路上和会话的检查）；fixture 13 个文件，+1,962 / −13（估约 1,400）。测试数：`f8cb7c2` 49 个，`49190525` 110 个（估多约 50 个，实多 61 个），收尾之后 118 个。本机 `make e2e`：110 个约 29 秒，118 个约 34 秒。

**收尾自己**（`T0` 到 `T16`，修订之后，`logs/a-closeoutsize.txt`）：287 个文件（不含生成的 3 个和语言文件 16 个），+3,670 / −1,646，净 +2,024：web 应用 162 个文件 +1,175 / −1,071，它的测试 27 个 +558 / −82，web 的包 20 个 +131 / −103（另有 `keyboard.test.ts` +25），e2e 的故事 37 个 +779 / −141、fixture 6 个 +253 / −177，Go 的测试 3 个 +38 / −25，`tools/` 3 个 +305 / −10（`keywords.json` 的三条规则和 20 个例外、`i18n-keys.mjs`、`keywords.mjs` 的说明），文档 24 个 +392 / −28；语言文件 16 个 +348 / −24。

**对 M4 以后的估算**：后端的生产代码与估计相符（11,526 对约 12,000，M2 校准过的按用例估有效）；测试照旧是估得最差的（前端 4 倍、后端 1.8 倍、故事 2.3 倍），M4 起前端的测试按"每个修改一张会话、权限、拒绝的表"估；组外的清理（M1 收尾交接的规则、跟随类型的使用方）不随领域的文件数，单列。

### A.7 第 14 节第 12 条：每一页的挂载

`inv/watchpages.py`（`logs/watchpages-T16.txt`）在最终原型的故事中找每个 `watchPage` 之后打开的页面，按 M3 的页面归类；S2 另以一个循环在四个角色下打开 M3 的每一页并断言请求的清单；P6、P7 的页面以变量打开（状态、标签的设置页），也有 `watchPage`。每一行的故事断言旧接口的请求、失败的请求和页面错误为空（`[watch.apiFailures, watch.oldApiRequests, watch.pageErrors]`）：

| 页面 | `watchPage` 的故事 |
|---|---|
| `/`（根，落点） | S2、S5 |
| `/create-workspace` | W1、S7 |
| `/onboarding` | A5、A10、W1、S7 |
| `/workspace-invitations?…`（邀请链接） | W5 |
| `/{slug}`（首页） | W8、W11、S7 |
| `/{slug}/projects` | P1、P2、P5、S6、S7、S8 |
| `/{slug}/projects/archives` | P4、S7 |
| `/{slug}/projects/{project}/…`（项目的包装层） | P2、P8、S2 |
| `/{slug}/settings`（general） | W3、W4、S7 |
| `/{slug}/settings/members` | W4、S7 |
| `/{slug}/settings/projects`（没有项目） | S7 |
| `/{slug}/settings/projects/{project}`（general） | P2、P3、P4、S2、S7 |
| `…/{project}/members` | P3、S7 |
| `…/{project}/features/…` | P3、S7 |
| `…/{project}/states` | P6（变量）、S7 |
| `…/{project}/labels` | P7、S7 |
| `…/{project}/automations` | P3、S7 |
| `/{slug}/profile/{user}` | P9 |

### A.8 W17：看得到的不同

每一处看得到的不同写进前端改动清单（Task 16）：3.1 的 M3 一行写收尾做了什么，"输入法组字时的 Escape"一行和"共用组件的键盘焦点"一行照收尾之后的事实，3.2 加 7 行"完成于 M3/收尾"：弹窗、离开工作区、M3 页面上的文案、名字和键盘、输入法和选择上的 Enter、工作区的地址、共用的写法。

英文的说法改了的（`inv/englishchanges.py`、`literalkeys.py` 对 en 的值和故事中的字面量，`logs/englishchanges.txt`、`literalkeys.txt`；故事照新的说法改）：

| 原来 | 现在 | 在哪里 |
|---|---|---|
| "Leaving..."、"Joining..." | "Leaving"、"Joining" | 离开、加入项目的按钮忙时（P2、P5） |
| "Delete Label" | "Delete label" | 删除标签的确认框 |
| "…the state- X" | "…the state X" | 删除状态的确认框的正文（P6） |
| 状态组的标题读小写的键（"backlog"） | 组的名字（"Backlog"、"Canceled"…） | 状态设置页（P6；C9） |
| "General Settings" | "General settings" | 项目设置 |
| "{name} settings - Cycles" 等 | "{name} - Cycles" 等 | 项目设置各页的标题 |
| 已选筛选的标签"Access""created_at" 和下划线的键 | "Network""Created date" 和筛选的名字 | 项目列表的已选筛选（Task 12） |
| "No members" | "No Member Yet" | 项目卡片的成员（Task 12） |
| "Enter Months"；"Submit"、"Submitting..." | "Enter months"；"Update"、"Updating" | 自动归档的自定义范围（P3） |
| "No projects yet" | "No project" 和一句说明 | 设置的项目页（没有项目时，Task 13） |
| 每个开关都叫"Toggle project feature" | 以功能为名（"Cycles"…） | 项目的功能（P1、P3） |
| 已选筛选的移除按钮没有名字 | "Remove X" | 项目列表（Task 12） |
| 项目卡片上设置的链接没有名字（只有图标） | "{name} settings"（"Web settings"） | 项目列表的卡片（Task 5） |
| 离开项目的提示"Enter 'leave project'"（照它输入会被拒绝）；"…by typing the 'Leave Project'." | "Enter “Leave Project”"；"…by typing “Leave Project”." | 离开项目的弹窗（P5） |
| "Enter 'delete my project'" | "Enter “delete my project”" | 删除项目的弹窗（P4） |
| "Token copied to clipboard." 之外的专用失败文案 | 共用的失败提示 | API 令牌、命令面板的复制（Task 9） |
| 首页的问候由"Good"和"morning"等拼成，名字是名和姓 | 每个时段一个带 `{name}` 的句子（"Good morning, {name}"），名字是显示名 | 首页（中文的"good"也译作"早上"，原来读作"早上 下午""早上 晚上"，现在是"下午好，{name}"等；S7） |
| "You declined this invitation." | "This invitation was declined." | 已拒绝的邀请页（W5；负责人 2026-10-13，3.2 第 2 条） |
| 侧边栏项目一行的按钮都叫"Open project menu" | 以项目为名 | 侧边栏（S8；F6） |
| 快捷键弹窗的名字"Keyboard shortcuts Close"（关闭按钮在标题里） | "Keyboard shortcuts" | 快捷键弹窗（S6） |

删去的 en 键：`afternoon`、`evening`、`good`、`morning`、`onboarding.switch_account.description_before_email`、`description_after_email`（同一句的两半，现在是一个键）、`workspace_settings.token_not_copied`、`power_k.miscellaneous_actions.copy_current_page_url_toast_error`。

行为的不同（都在 3.2 的 7 行里）：弹窗打开时后面的页面不能操作、不被读出；Tab 在确认框里转；三个直接用 Headless UI 的弹窗改经 `ModalCore`：命令面板的层级 `z-50` → `z-30`，现在在提示、tooltip、表情的弹层和右键菜单之下（它们经 portal 渲染在主树之外，`z-50` 及以上），与别的弹窗一样，M4 页面上 `z-50` 的元素会画在它之上（交给 M4，2.6）；设置页的手机菜单 `z-50` → `z-20`，在每个弹窗之下（原来画在弹窗之上，2.6）；切换账户的层级 `z-20` → `z-30`、宽度 `w-[40rem]` → `sm:max-w-2xl`；快捷键的标题左对齐（原来居中），关闭按钮在标题旁；只在打开时才挂载的弹窗现在有打开的过渡（原来立即出现）；成员页的移出、离开和邀请的删除被拒绝时确认框留着；工作区菜单的"离开工作区"；工作区菜单的按钮在键盘取得焦点时显出内圈的轮廓，方向键选中的一项（工作区和三个操作）与指针悬停时一样显出底色，当前工作区的一项选中时深一级（原来都看不出；指针悬停在当前工作区的一项上时现在也深一级）；换工作区、换项目（后退、前进也是）时工作区的内容、项目的页面和设置重新挂载，开着的确认框关上，命令面板和快捷键列表也是（原来留着，确认框跟过去，作用于那个工作区或项目；负责人的 B 和控制者的裁定，2.7）；随之经菜单换工作区之后焦点在 `body` 上，像一次页面的加载（原来回到菜单的按钮；控制者裁定接受，全应用的焦点交给 M8 的交接第 4 节），窄屏（768 px 以下）上每一次重新挂载都把开着的侧边栏收起（`sidebar/sidebar-wrapper.tsx` 挂载时的副作用），所以手机上的后退会关上它，侧边栏的"Projects"一节重新展开（它的开关是组件的状态，写进 `localStorage` 却不读回；"Workspace"一节、每个项目的展开、侧边栏的收起和宽度在 `localStorage` 或 store 里，照旧留着），侧边栏和页面的滚动回到顶上（新的元素）；换项目时侧边栏在 key 之上，不变，项目的页头和页面重新挂载；`CustomSelect` 上的 Enter 打开列表、不提交表单；Safari 的组字顺序；侧边栏的副本、收起时 `inert`；侧边栏的"+"、项目一行的菜单在键盘取得焦点时显示，一行的菜单一个停留点，手柄不是停留点；成员页一行的菜单名为"Options"；项目下拉框的停留点是按钮；状态的表单打开时焦点在名称上；颜色弹层的输入框有焦点的轮廓；图标的连字和表情不并进名字。

### A.9 计划的块和逐 Task 复现

- **块的核对**：`mkblocks.py` 从 `T0` 起把每个 Task 的块依次应用，与那个 Task 的快照逐字节相同（16 个 Task，0 个问题）；`compose_plan.py` 核对每个有块的文件只给一次、文件结构的表列出计划改到的每个文件。Task 14 的生成文件不写成块，计划钉住 `make gen` 之后它们的 SHA-256 和行数（`gensha.py check`：只有 Task 14 改生成的文件，`49190525` 的与 Task 1–13 的相同）。
- **plan 的大小**：plan 共 13,122 行（修订之后；第一轮 12,760 行）：Task 1 之前（目标、约束、一次性的脚本、文件结构）228 行；各 Task 一节（"其中代码块"是代码块中的行，`plansize.py`）：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 | 360 | 284 | 9 |
| 2 | 679 | 586 | 14 |
| 3 | 835 | 729 | 18 |
| 4 | 1,130 | 954 | 39 |
| 5 | 1,101 | 963 | 27 |
| 6 | 1,045 | 895 | 30 |
| 7 | 849 | 681 | 39 |
| 8 | 604 | 495 | 17 |
| 9 | 1,115 | 943 | 38 |
| 10 | 474 | 356 | 20 |
| 11 | 647 | 525 | 24 |
| 12 | 1,154 | 990 | 34 |
| 13 | 817 | 650 | 37 |
| 14 | 1,361 | 1,164 | 40 |
| 15 | 611 | 522 | 20 |
| 16 | 112 | 72 | 4 |

  写 plan 之前的核对（brief 的"Phase size"）：按 13 组估出 13–14 个 Task、约 13,500 行，每个在约 1,450 行之内，在约 16 个、每个约 1,500 行之内，不拆。原型中实测 16 个 Task、12,760 行（修订之后 13,122 行，每个在约 1,500 行之内）；原来的第 12 个 Task（收尾的扫描找到的英文）量出 1,733 行块，加上说明会超过约 1,500 行，所以分成 Task 12、13（第 3 节第 3 条）；最长的 Task 14 是 1,361 行（代码块 1,164 行）。修订之后 `compose_plan.py` 另拒绝两种 plan：说明里一行像块的选择（以 `^` 开头）却不在 `@@BLOCKS …@@` 里（第一轮的 Task 3–8、12、13 正是这样，选择印成了说明，Step 1 空着，块都落在 Step 2 下：预检的 A-M1），和没有内容的一步；`amend/guarddemo.py` 核对第一轮的说明在 Task 3 被拒绝、加一个空的步被拒绝、现在的说明通过（`logs/a-guard-demo.txt`），`amend/stepcheck.py` 列出每一步给的文件：Task 1–13 的 Step 1 给它的非测试的 web 文件，Step 2 给其余的（`logs/a-stepcheck.txt`）。
- **逐 Task 复现（第一轮）**：`replay.py` 在本 spec 和 plan 的第一个提交 `fb5490c5` 的 `git archive` 副本上（`base-commit.tar` 解开到 `$M3TMP/closeout-replay`，`pnpm install --frozen-lockfile` 之后；plan 的一次性脚本 `doclinks.py` 照 plan 写到 `$M3TMP/closeout-plan/`），照 plan 逐个 Task 执行：写入块（`planapply.mjs`），核对 Go 的模块文件不变、`pnpm-lock.yaml` 与那个 Task 的快照相同，再执行这个 Task 的每个 `Run:`。副本不是 git 仓库，所以 `make lint-web` 照它的写法分三步运行（关键词守卫经 `kwreal.py` 运行仓库自己的 `tools/keywords.mjs`，`fakegit` 照 `git ls-files` 列出副本的文件；`tools/i18n-keys.mjs`；turbo 的四个检查），`make gen-check` 由 gen-same 代替（`make gen` 之前之后生成的文件逐字节相同），`make e2e` 只有 S3 因为读不到提交而失败，由脚本单独认出（F4）。16 个 Task 都通过（`replay-logs/summary.txt`）：

| Task | 写入的文件 | 检查 |
|---|---|---|
| 1 | 9 | `make lint-web`（69 条规则、3 个例外、没有命中；turbo 54 个任务）、`make knip`、`make test-web`（16 个任务）、`make build` 通过；`make e2e` 109 个通过（S3 除外） |
| 2–8 | 11、18、36、27、30、38、15 | 同 Task 1；`make e2e` 111、111、112、113、114、114、115 个通过 |
| 9 | 35 | 关键词守卫 70 条规则、23 个例外；`tools/i18n-keys.mjs` 读到的 1,261 个键都有定义；其余同上，115 个 |
| 10 | 20 | `pnpm install --frozen-lockfile` 通过；其余同 Task 9，115 个 |
| 11–13 | 22、34、37 | 71 条规则、23 个例外；读到的键 1,261、1,275、1,277 个；`make e2e` 115、116、117 个 |
| 14 | 40 | `make gen` 之后三个生成文件的 SHA-256 和行数与表相同；gen-same：39 个生成文件 0 处差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个包 `ok`、没有 `FAIL`；其余同 Task 13，117 个 |
| 15、16 | 20、4 | `make lint-web` 同上；`doclinks.py` 每个文件 `0 broken`，`broken: 0` |

  这一次复现用的 plan 是本 spec 第一个提交中的那一版（12,760 行，SHA-256 `2df6ca8a6346d890…`）。最后的副本与原型逐文件相同：`treediff.mjs closeout-replay closeout-proto`（原型在 `T16`，放进同一个提交的 spec 和 plan），3,255 个文件对 3,255 个，0 处差异（`logs/replay-all.log`）。本 spec 的第二个提交只改本 spec（本节的复现结果，2.4 的 K9 一行和 2.8 的 S7 的测试数），plan 不变。
- **修订之后的复现**：同样的做法，在修订的提交 `dfe41b2b` 的 `git archive` 副本上（`base-commit.tar`；第一轮的副本、日志和 `base-commit-fb5490c5.tar` 移到 `$M3TMP/closeout-v1/`）。16 个 Task 都通过（`replay-logs/summary.txt`，`replay-logs/replay-all.log`）：

| Task | 写入的文件 | 检查 |
|---|---|---|
| 1 | 9 | `make lint-web`（69 条规则、3 个例外、没有命中；turbo 54 个任务，oxlint 的条数各等于上限）、`make knip`、`make test-web`（16 个任务）、`make build` 通过；`make e2e` 109 个通过（S3 除外） |
| 2–8 | 14、18、39、27、30、39、17 | 同 Task 1；`make e2e` 111、111、112、113、114、114、115 个通过 |
| 9 | 38 | 关键词守卫 70 条规则、23 个例外；`tools/i18n-keys.mjs` 读到的 1,261 个键都有定义；其余同上，115 个 |
| 10 | 20 | `pnpm install --frozen-lockfile` 通过；其余同 Task 9，115 个 |
| 11–13 | 24、34、37 | 71 条规则、23 个例外；读到的键 1,261、1,275、1,277 个；`make e2e` 115、116、117 个 |
| 14 | 40 | `make gen` 之后三个生成文件的 SHA-256 和行数与表相同；gen-same：39 个生成文件 0 处差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个包 `ok`、没有 `FAIL`；其余同 Task 13，117 个 |
| 15、16 | 20、4 | `make lint-web` 同上；`doclinks.py` 每个文件 `0 broken`，`broken: 0` |

  这一次复现用的 plan 是修订的提交中的那一版（13,122 行，SHA-256 `71bcc9ddfd853970…`）。最后的副本与原型逐文件相同：`treediff.mjs closeout-replay closeout-proto`（原型在 `T16`，放进同一个提交的 spec 和 plan），3,256 个文件对 3,256 个，0 处差异（多的一个是 Task 4 的 `switch-account-modal.test.tsx`）。修订的第二个提交只改本 spec（本节的复现结果，2.6 的探针的路径和 `zgrep` 的一句），plan 不变。
