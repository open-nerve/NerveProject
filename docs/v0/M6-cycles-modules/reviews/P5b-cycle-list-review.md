# M6/P5b 阶段评审记录

> 整分支审查 Ready to merge（[re-67](../../../reviews/re-67-M6P5b整体审核.md)），修复轮 F 经限定复审（Ready to merge，五处 Minor 以措辞与 spec §5 处理，re-67 §4）；最终候选的代码 `ebeb2cab`，其后只有本记录所在的提交（文档与页头的一处注释）。合入、main CI 与归档清理见 §7（由 P6 的 T1 回填）。基线 main `0b8efcd701edf7e73a37d5bb82ad176a7795b910`（P5a 的 noff，main CI 42995581409 首次通过）。

## 1. 评审方式与范围

P5b 交付设计 §12 P5 一行中迭代页的工作项列表：列表引擎的 `EIssuesStoreType.CYCLE`（项目列表加 `cycle_id`）、成员判断认得迭代与模块、按页的偏好基类与迭代页的偏好（`/me/cycles/{id}/preferences`）、四种布局、页头、空态、已结束与已归档的只读、本页新建（只在迭代可加入时带 `cycle_id`）、移出迭代与写后重读进度、进行中卡片的优先级页签、卡片与侧栏的行点击，以及 G15、G19 的迭代页列表部分。“添加已有工作项”、迭代下拉框与列表词表在 P7（R11）。

- **调研**：P5a 合入前，两个只读调研代理在 P5a 的树上写出现状笔记（M4 的列表引擎全程；Plane 的迭代页列表与 nerve 的接口契约），随台账归档。
- **界定**：[spec](../specs/P5b-cycle-list.md)（R1–R11、A1–A8）与 [plan](../plans/P5b-cycle-list.md)。
- **预检**：实施前一次 fresh 只读预检（[re-66](../../../reviews/re-66-M6P5b预检.md)），结论为修正后可实施：两处 High（迭代关闭时详情页的列表区、日历子树的十处读）、六处 Medium、九处 Low 在实施前改入，Task 由 14 个拆为 16 个。
- **实施**：T1、T15 的文档与各 Task 由实现代理完成，主控逐 Task 复核并记台账；T13 在真浏览器里发现 M4 的一处缺陷（§3 第一条），在本段修复。
- **门禁与审查**：T10 一次检查点（七项门禁与分支 CI）；T16 一次 fresh 只读整分支审查（[re-67](../../../reviews/re-67-M6P5b整体审核.md)），修复轮 F 另经一次限定复审；最终候选的七项门禁与三种引擎的页面配置。

## 2. 验收状态

A1–A7 满足（[spec §4](../specs/P5b-cycle-list.md#4-验收标准)已勾选），A8 待合入后由 P6 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 引擎：CYCLE 的 store、动作、路由映射、空态；成员判断、`fields()`、写入放行 `cycle_id` | T2、T5、T8 |
| A2 | 偏好：基类重构后项目页不变；迭代页偏好的读、改、被拒、较旧的读不盖较新的 | T3、T4、T8 |
| A3 | 页面：四种布局、页头、空态、只读、新建一个请求、移出迭代、拖拽、被拒说明、写后重读进度 | T6–T10、F3 |
| A4 | 优先级页签与行点击、侧栏切换、预览里不可点 | T11、T12、F4 |
| A5 | G15、G19 的迭代页列表部分（三种引擎） | T13、T14、F1、F2、F5 |
| A6 | 守卫：`project-address-route`、告警上限 53 不变、knip、关键词、按本页分组 | 各 Task |
| A7 | 文档与交接回执 | T1、T15、F7、F8、T16 |

## 3. 局部裁定

台账中 128 条 `Ruling:`，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **M4 的缺陷，本段修复**（T13，主控裁定）：列表行菜单的“修改”“复制一份”打开后约一秒自行关闭，项目页、迭代页、个人页都如此。打开弹窗读工作项详情，子项的读取对该工作项做父项核验（M4 P12 D22），M4 的列表在核验期间有意把该行移出（`getIssueById` 隐藏未决项、`IssueLayoutHOC` 以骨架替换布局），弹窗渲染在行内而随行卸载。照 M4 自己的先例（re-17 R1、P13/re-19 R2）把两个弹窗移到加载门之上的 `IssueListCommandHost`（`a4735ee7`）；改核验语义或核验中仍显示行的两种修法改 M4 的规则（D22、re-29 R08），不取。记入 [M4 交接](../handoffs/M4-closeout.md)。
- **工作区管理员兼访客**（T14，主控裁定）：只看自己的工作项与数目，但可新建、可移出迭代——照 spec §2.6（新建给项目页的 `allowPermissions([ADMIN, MEMBER])`，网页以工作区管理员为项目 ADMIN，M4 I20）与 nerve 的 `access/domain/decide.go`；plan 的一句写过了头，已改正。
- **全局新建**（T5 之后的主控裁定）：迭代页上的全局新建只在迭代可加入（迭代打开、未结束、未归档）时带 `cycle_id`，否则照项目页新建。
- **优先级页签的 store**（T11）：独立的小类 `CyclePriorityIssues`，根构造时建一个、每个迭代一个绑定，不继承 `BaseIssuesStore`，不碰本页的当前列表（换迭代时不闪出上一个迭代的行）；只在页签打开时取数，失败说明并可重试（与 Plane 的差异记入差异清单）。
- **行点击**（T12）：卡片的行替换本页筛选、确认后才进入、一次一个；优先级行经导航状态在迭代页自己的预览宿主下打开预览，进入后以 `replace` 取走；侧栏行只在本页筛选实例存在时可点（与 Plane 的两个条件在 nerve 里等价）。
- **页头的控件**（F3，re-67 M8）：布局、筛选与显示在本页偏好读到之后才出现，与正文同一条件；项目页同因、M4 起如此，记入 spec §5 的 P6。
- **菜单工厂的接口**（F4，re-67 M7）：列表行传 `openDialog(kind)`，详情与归档保留两个 setter，删去以 ref 记“要修改的工作项”的适配。
- 其余见台账。

## 4. 实施记录

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `a23270ea`、`13d02edb` | 16 个 Task；预检 re-66 的修正 |
| T1 回填与阶段 | `5d9b1e2e` | P5a 阶段记录 §7、spec A8、设计 §14、M6-M8-05 的补充 |
| T2 成员判断与写入 | `a91035ea` | `issueMembership` 认得 `cycle_id`、`module_id`，`fields()`、白名单；7 个变异失败 |
| T3 偏好的基类 | `b205756f` | `PagePreferencesFilter`，项目页的既有测试不改断言全过；7 个变异失败 |
| T4 迭代页偏好 | `781ffefc` | `CyclePreferencesService`、`CycleIssuesFilter`；8 个变异失败 |
| T5 列表引擎 | `5f530b8b`、`0013acec` | `PagePreferencesIssues`、`CycleIssues`、迭代页的动作与路由映射；10 个变异失败；全局新建的裁定 |
| T6 读本页筛选 | `a46cf5e9` | `usePageIssueFilters`、日历子树与拖拽；6 个变异失败 |
| T7 只读 | `fd26d88e` | 四个 base root 的 `readOnly`；14 次变异失败 |
| T8 布局根与页面 | `77d52b4b` | `CycleLayoutRoot`、四个迭代根、空态、迭代关闭时停用、偏好跟进；16 个变异失败 |
| T9 页头与新建 | `641f507b`、`6c3a6f6a` | 数目、布局、筛选、显示、新建（锁定项目）；13 个变异失败；spec §5 补记 M4 的页头数目缺陷 |
| T10 快捷操作与进度 | `0dcdfe5f`、`00cf6c38` | 移出迭代、写后重读 `/progress`；11 个变异失败；检查点 |
| T11 优先级页签 | `7051ee4d` | `CyclePriorityIssues` 与卡片的第三个页签；11 个变异失败 |
| T12 行点击 | `99bf286b` | `useEnterFiltered`、`PeekOnEntry`、侧栏切换；15 个变异失败 |
| T13 页面故事一 | `a4735ee7`、`f4a37007` | M4 缺陷的修复（5 个变异失败）；G15 三个故事（6 个 e2e 变异失败） |
| T14 页面故事二 | `b90dda81` | G19 四个故事（14 个 e2e 变异失败） |
| T15 文档 | `8b11bc63` | M3/M4/M5 交接回执、差异清单、前端改动清单、设计、M8 交接 |
| 修复轮 F | `6512df33`–`ebeb2cab` | §5 的两处三引擎竞争（F1、F2）与 re-67 的 M1–M9（F3–F7）；F8 改正 M6-M8-07 的措辞、新增 M6-M8-09 |
| T16 记录 | 本记录所在提交 | re-67（含修复轮的限定复审）、本记录、spec §4 的勾选与 §2.6、§5、plan 的状态与第 185 行、设计 §14、前端改动清单的一句、M6-M8-07 的再现、`header.tsx` 的注释 |

## 5. 门禁与 CI

- **检查点**（T10 之后，`00cf6c38`）：
  - `t10-gate`：`make gen-check`、`make lint-go`、`make test`（本机 894 秒通过）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 445 过、1 败：W7（`w7-member-management.spec.ts:44`）测试一侧的 Postgres “too many clients already”，即 [M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03)，只重跑这一项通过（`t10-gate-e2e-w7-rerun`）。
  - 分支 CI 43047769115：web、server (0)、server (1) 成功；e2e 两次取消。attempt 1：`playwright install --with-deps` 的 apt 自 15:43:38Z 起解析不了 `azure.archive.ubuntu.com`，16:12:29Z 起陷入 “Tried to start delayed item … but failed” 的循环直到作业期限，一条测试没跑；按“查明是环境原因、只重跑一次”重跑 e2e。attempt 2：同一原因，这一步用了 30 分钟，只剩 3 分半给测试（已跑的都过）。不再重跑，交 [M6-M8-06](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-06)；T10 的代码由下一次推送覆盖。
  - 分支 CI 43070304124（`f4a37007`，T13 之后推送）全绿：web、server (0)、server (1)、e2e 130704599852（作业 20 分 41 秒，装浏览器 28 秒，镜像已恢复）。
- **最终候选之前**（`8b11bc63`，`t16-gate`）：前六项通过（本机 `make test` 这次通过）；build+e2e 452 过、1 败：M5 的 F11 (page) [s3] 的 `net::ERR_ABORTED GET /api/v0/assets/{复制品}/content`。trace 显示第一次被拒的粘贴之后的撤销与第二次被拒的粘贴各让编辑器重读一遍已有图片，第二遍的内容读取发出 9.8 ms 后未答即被中止；P5b 没有改编辑器、资产、详情页与该夹具，只重跑该文件一次通过（`t16-gate-e2e-f11-rerun`），交 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07)。
- **三种引擎的页面配置**（`t16-three-engines`，`8b11bc63`）：54 过、3 败，根因都在测试：
  - G3 (page) 在 firefox 与 webkit：P4 的 `dismissToasts` 数到通知后点它的关闭，恰逢通知约 5 秒的自动关闭（Base UI 默认），按钮在退场中“not stable”、随后脱离 DOM，click 等到 60 秒超时。F1 改为一次按下所有未在退场的通知的关闭、再等全部消失；插入等待的确定性复现下旧版失败、新版通过。
  - G15 故事二在 firefox：初次载入 1.8 秒后的 `page.reload()` 之后约 10 秒，Firefox 报根预加载的字体未用。F2 的探针显示任何一次再次载入都会如此（不论相隔多久），故事二改以表格表头的排序（页面自己的偏好写入）代替重载，故事一的“再次载入后仍在”改由同一人另开的浏览器核对；`--repeat-each=5` 全过。其余三引擎故事的再次载入同属潜在间歇，交 [M6-M8-09](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-09)。
  - 修复轮之后 `f-final-three-engines`（`35215d46`）57 条全过。
- **最终候选**：`t16-final-gate`（`ebeb2cab699e15fb7cae13b9a611189c0c0c2cc8`）：`make gen-check`、`make lint-go`、`make test`（本机通过）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 452 过、1 败：F11 (page) [local]，与 `t16-gate` 同一签名、同一过程（第二次被拒后的撤销让编辑器重读已有图片，内容读取在故事粘贴普通文本 0.5 ms 后发出、11.3 ms 后中止）。把内容读取经路由延迟 150 ms 不能复现（`t16-red-f11-undo-race` 通过），所以不能简单归为“载入未完、下一次粘贴中止它”，未证实的修法不上；P5b 在该详情页只走到 `fields()` 的计算。依 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 的处理办法只重跑该文件一次通过（`t16-final-gate-e2e-f11-rerun`，7 例），再现与否定的实验补记其中。
  - 本记录所在提交只改文档与 `header.tsx` 的一处注释，另跑 `make lint-web`、`make knip`、`make test-web`、文档链接与关键词（收据 `t16-record-*`），其余以准确 HEAD 的分支 CI 为准。
- **记录提交的分支 CI**：`496d905d58228ce81d83045ff0d319e18eff642b` 的 43098103752 自 2026-11-11T20:47:23Z 起一直 queued、没有建立任何作业（到 2026-11-13T02:41Z 约 30 小时；仓库公开、同一并发组里没有别的运行）。不能取消、也不能重跑未完成的运行；workflow 的 `concurrency`（分支上 `cancel-in-progress: true`）让同一分支的新推送取代它，所以以这条补记作为新的提交推送。
- **最终候选的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-67](../../../reviews/re-67-M6P5b整体审核.md)）逐项核对了：

- 数据层与写入（成员判断、偏好的读写次序与整字段替换、一次新建、移出与写后重读进度、优先级页签的绑定）；
- 会话与页面的范围（挂载键、迟到的回答、退役后不写、不读 router store）；
- 权限与只读、行点击；
- M4 共用代码的回归与回归边界；
- 测试的诚实性（另由三个 fresh 审者分头核对）与文档的准确性。

结论 Ready to merge：没有 Critical、Important，十一处 Minor。修复轮 F 处理 M1–M9（文档三处、测试强度、菜单工厂的接口、页头控件、注释），M10 记为 M8 的观察，M11 不需动作；修复轮的限定复审结论见 re-67 §4。

## 7. 集成、归档与接续

由 P6 的 T1 按归档里的台账、CI 收据与 `cleanup.log` 回填。

**接续**：P6（网页的模块）。
