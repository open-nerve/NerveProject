# M6 / P7a 整体审核

> 2026-11-14：Ready to merge；主控仍以修复轮 F 处理三处 Minor（同 P6b 的先例），修复轮 F 经限定复审 Ready to merge（§4）。
> - **目标**：分支 `codex/m6-p7a` 相对 main `7868a6b496d2587e2ed2cac7360c3ca6a86963be`（P6b 合入后 CI 的修复 noff）的全部修改（`7868a6b4..01a10e66`，`ab4ef06b` 至 `01a10e66` 的 21 个提交，含并入那次修复的 `4b93ffd0`，163 个文件）；修复轮 F（`f113759f` 起的四个提交，F1–F4）另经一次限定复审（§4）。
> - **依据**：[P7a spec](../v0/M6-cycles-modules/specs/P7a-work-item-planning.md)、[P7a plan](../v0/M6-cycles-modules/plans/P7a-work-item-planning.md)、[预检 re-72](re-72-M6P7a预检.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。
> - **阶段记录**：[P7a 阶段记录](../v0/M6-cycles-modules/reviews/P7a-work-item-planning-review.md)。

## 1. 方法

一次 fresh 只读审查（`fable`，台账 Task 16），范围 `7868a6b4..01a10e66`（spec 与 plan、并入的修复与 T1–T15 的全部提交），不改动文件与 git 状态。它与最终候选 `01a10e66` 的本机门禁（`t16-gate`、`t16-m6-pages`）并行，没有跑构建与测试，引用的收据是台账的。它读了整份差异、spec、plan、re-72、主控的 D1–D10 与台账，对照接口合同（`api/modules/issue.yaml` 的 `IssueCreate`、`IssueUpdate` 与草稿的 payload）读写入单元与每个新的面、承重用例的新测试与两个 e2e 故事；文档引用的 Plane 源码在主目录的只读 `plane/` 抽查 11 处。

原始报告没有另存文件；结论、核对过的性质与各条发现照它转述，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **写入**：模块的增减在写队列轮到时以所持的集合算（`issue-writes.ts` 的 `changeIssue`，`issue-write-fields.ts` 逐个点名字段，`modulesAfter` 保留次序与已归档、页面不认识的 id），被拒之后下一个写重读所持的；`IssueChange` 的 `module_ids?: never` 有以 `Partial<IssueDetail>` 的值写的类型测试；P6b 的移出经同一命令，它的测试不改。修改发出表单起点到终值的差（`moduleSetChange`，没有变化时不带，P6b R7 仍成立）；迭代是固定字段，`null` 照发。草稿照 re-72 F6（`draftInput` 不改；契约里草稿与 `IssueCreate` 的 `cycle_id` 不可为 `null`、`IssueUpdate` 的可为，R6 对）。复制与调用者给的值等两份列表答到才过滤，读被拒时说明并可重试；表单优先（弹窗经 `createIssueAsGiven`，快速新建照旧注入）；命令宿主的新建按项目持有、按 `isPlanningPageStoreType` 锁定项目、`DIALOG_STORES` 覆盖四种。
- **会话分代与迟到的回答**：选项经 `useDirectoryRead`（以 `useDetailScope` 与权限身份为主、请求计数、吞掉 `SessionChangedError`），下拉框以项目为 key；名字与列表经 `useSessionSWR`（键带 `loginId`），由 `usePlanningReads.listed` 把门，键与包装层、归档页共用，取数用当代 store 的方法；写都经 `run`、`useDetailCommand`、`useDetailScope`。没有发现过时的写。
- **界面**：选项与值分开（已结束、已归档的迭代在按钮上带状态与说明、不作选项；已归档的已选模块排在选项之后、带说明、只能去掉；不重排，R13）；各面的开关都经 `planning-switch.ts`；可访问性（一个原生的触发，面板上的 popper 引用，`modal={false}`，搜索框取得焦点，按钮里是 `span`，以行为名，表格的格子关上时焦点回到格子，字段的 `invalid`、`describedBy`，R17 的静态表）；只读照传。
- **设计质量**：改到的最大的产品文件 `form.tsx` 372 行、`all-properties.tsx` 345、`issue-writes.ts` 302、`command-host.tsx` 255、`create-command.tsx` 181，根 320 未动；没有值的循环导入；没有开关、桩或兼容（`handleUpdate`、`WritesRoot.workspaceSlug`、五个没有读者的 i18n 键已删，`usePlanningListsAnswered` 是被替换而不是留着）；类型由 `IssueUpdate`、`IssueDetail` 推出；守卫（`project-address-route` 加 `issue-writes.ts` 与 33 个文件并有样例，swr 的限制，i18n 两种语言同提交）。
- **测试**：真实的 store 与 FakeNerve，替身只限编辑器、通知与无关的侧栏；plan 列的变异在台账里都有失败的收据（T2 2、T3 6、T4 3、T5 8、T6 10、T7 13、T8 9、T9 11+2、T10 15、T11 9、T12 17、T13 11、T14 6），存活的两个（T2 由快照取范围、T4 新建的请求体）补了钉住的测试，并以同一变异 RED；改过的回归断言都有理由（R17、R15、两个页头测试的显示属性、`root.store.planning.test.ts:146` 只改参数形状、`close-policy.test.tsx` 的 `module_ids: []`）；e2e 不用 `page.reload()`，锚是确认的状态、扣住的回答与记下的请求体；G4 走遍 spec §2.11 的各项，G11 只差一项（M3）。
- **文档**：`plane-diff.md` 引的 Plane 行号抽查 11 处都对；`frontend-changes.md`、设计 §8、§12、§14 与交接回执（M2 §4、§5(a)，M3 §3，M4 M6-02，M6-M8-01、-03、-08，新的 M6-M8-13）与代码和台账一致，行数也对。

## 3. 结论与处理

结论 **Ready to merge**：没有 Critical、Important，三处 Minor 与六条观察，没有须负责人决定的事（对负责人的只有 T16 推送前的 e2e 估算与上界，照 30 分钟的规则，台账 T16 记）。主控仍以修复轮 F 处理三处 Minor（同 P6b 的先例），六条观察不需动作：

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | 两份列表在途时保存会发未核过的调用者的值：`create-command.tsx:96` 在 `reads.out` 时照原样留着调用者给的值，`:160-161` 有值且开关开着就发 `cycle_id`、`module_ids`；`form.tsx:341-365` 没有把 `planningLoading` 交给 `IssueFormFooter`，两栏转圈时“保存”可按。从已结束的迭代、已归档的模块复制的快手会因没见过的值收到 409（T12 的裁定只管列表被拒）；nerve 挡住坏数据，只关体验 | F1 照主控裁定加载中禁用主按钮，不取“加载中不发两键”（那会静默丢掉持有的值，违 re-72 F7）：`form.tsx` 的 `planningPending`（加载中且表单的项目仍是打开时的，与两栏同一条件）交给两栏与页脚，主按钮的 `disabled` 并上它；“存为草稿”不变。新测试一例（列表在途时禁用、点它不发，答到之后可按、只发选项），变异“去掉禁用条件”失败；`close-policy.test.tsx` 的替身另答两份列表、`fake-form.tsx` 给每个测试自己的 SWR 缓存，断言不改（`f113759f`） |
| M2 | 草稿的项目换走又换回时，保存丢掉存下的迭代与模块：`form.tsx:176-180` 以 `shouldDirty` 清空，`form-submission.ts:87-92` 在 `moved` 为假时也照改动发出（台账 T13 记过，保存前两栏显示已清空） | F2 照主控裁定：`project_id` 回到表单的默认（打开时的）项目时，两栏以 `resetField` 回到存下的值，不算改动、不发。新测试一例（换到 B 再换回 A，两栏显示存下的，保存的 payload 带着它们），变异两个（去掉两行 `resetField`、只去掉模块一行）都失败（`72c39da1`） |
| M3 | G11 没走 spec §2.11 的“Gamma 只能去掉”：`g11-issue-modules-page.spec.ts:84-90` 只核 Gamma 列为已选，接着移出的是 Alpha（组件一级由 `dropdown.test.tsx:149`、`sidebar.test.tsx` 与变异 `t7-mut-archived-not-listed` 钉住） | F3 在修改弹窗那一步之后加一步：列表里该行的模块控件打开、点已归档的 Gamma，只发一个 `PATCH` `{module_ids: [Delta]}`；nerve 确认之后控件只名 Delta，开着的列表不再列出 Gamma；数据库 Gamma 一行此刻移除，动态多一条。e2e 变异（点已归档的已选项不发移出，能编译、能构建）让故事在这一步失败，期望错误的 RED 证实“不再列出”的断言；三种引擎、chromium `--repeat-each=3` 通过；G11 本机 chromium 2.6 秒（T14 为 2.6 秒）（`9b4a5f77`） |
| 观察 | T11（修改时关掉开关 → 409 的提示）与 T12（内容复制的弹窗开着时关掉开关 → 发布被拒、草稿留着，`draft-planning.test.tsx:351` 钉住） | 在设计 §3.3 B 之内并有记录，不需动作 |
| 观察 | `useListRead` 每次调用都建两个目录（`use-planning-names.ts:77`） | 微不足道，不需动作 |
| 观察 | `create-command.tsx` 内容大多是新的，仍留着 Plane 的文件头 | 照 T4 的裁定，不需动作 |
| 观察 | `row-planning-properties.tsx:33`、`:53` 一行停用两条 oxlint 规则 | 照 `all-properties.tsx:119` 的先例，不需动作 |
| 观察 | 已持有的列表在挂上时再验证一次（台账 T10） | 不闪（`held` 为真即不算在途），不需动作 |
| 观察 | M6-M8-13 的范围写得对：不是本段引入，交 M8 | 不需动作 |

F4 写本记录，并在 spec §5 加修复轮的清单、改 spec 与 plan 的状态、设计 §14 的 P7a 一行（`docs` 提交，即本记录所在的提交）。

修复轮中实现者的局部裁定（台账 Task 16 修复轮 F 的 Ruling，交限定复审核对）：
- F1 的禁用条件与两栏的加载同一个（表单换到别的项目之后，它的值是那个项目的、经那个项目的选项选的，保存不等打开时项目的列表）；“存为草稿”不随列表禁用（主控的裁定只说主按钮；草稿存下什么都可以，发布时由 nerve 核对、拒绝落到字段，spec §2.8、R17）。
- F2 以项目一栏不是改动（RHF 的默认值即 `data.project_id`）判断“回到打开时的项目”，与 `form-submission.ts` 的 `moved` 同一起点。
- F3 的一步放在行的模块控件上、修改之后：放在预览里要重开预览并等它到齐，放在修改之前会改掉 spec 那一行写着的修改请求体 {Gamma, Delta}；行与预览共用 `ModuleDropdownBase`。e2e 变异不取 `t7-mut-archived-not-listed`（它让故事在旧的一步就失败）与“点 Gamma 发整集合”（没有在途的写时与轮到时算的 {Delta} 相同，等价存活）。

## 4. 修复轮 F 的复审

修复轮 F（`f113759f..52c04d69`，4 个提交，F1–F4）经一次 fresh 限定复审（`fable`，范围 `01a10e66..52c04d69`，只读、没有跑构建与测试，与最终候选 `52c04d69` 的七项门禁并行）。原始报告没有另存，结论与核对过的性质照它转述。逐项核对：

- **范围**：F1–F4 各只做了提交所说的，关掉 §3 的 M1–M3；主控接受的三项取舍（“存为草稿”在列表在途时仍可用、F3 在行上、F1 的测试准备改动）照实现。
- **F1**：`form.tsx` 一处算出 `planningPending` 交给两栏与页脚，页脚主按钮的原生 `disabled` 并上它，没有省掉任何键。弹窗读列表的项目（`base.tsx` 的 `activeProjectId`）就是表单打开时的项目，所以条件恰为“在途的是表单所在项目的列表”；修改与草稿从不读列表、主按钮不受影响；“存为草稿”与“加入项目”不变。主按钮是表单唯一的提交按钮，禁用时 Enter 的隐式提交与描述编辑器的 `submitBtnRef.click()` 都不提交。产品代码没有动 store、SWR 的键与配置。
- **F2**：项目一栏回到默认值时 RHF 去掉它的改动标记，`resetField` 把两栏恢复为存下的值并去掉改动，`form-submission.ts` 不再发这两键；与提交件的 `moved` 同以 `data.project_id` 为起点；新建的表单走另一支，不变；挂上时的那一次是幂等的。
- **F3**：新的一步在修改之后，保住 spec §2.11 写着的修改请求体 {Gamma, Delta}；修改那一步的数据库核对不变，只提成常量复用；没有 `page.reload()`，锚是 nerve 确认的名字与记下的请求体；时长记在 §3 与 spec §5。
- **测试**：RED、GREEN 与变异的收据都在台账；F1 的三处测试准备的改动不掩盖行为（`copied()` 等描述的编辑器出现，与原意等价，旧的等法在保存要等列表之后会一直等；`close-policy.test.tsx` 只多答两份空列表；`fake-form.tsx` 每个测试一份 SWR 缓存，是同目录两个测试已有的写法）；没有削弱的断言。
- **文档**：本记录与审查者的结论相符，抬头的数（21 个提交、163 个文件）核过，各处链接都能打开。

结论 **Ready to merge**：没有 Critical、Important、Minor，六条观察都不需动作，没有须负责人决定的事：

| 编号 | 观察 | 处理 |
|---|---|---|
| 观察 | F1 只守按钮：`form-submission.ts:106`、`create-command.tsx:126` 不看 `planningLoading`；浏览器里没有绕过禁用的默认按钮的提交，测试里直接派发 `submit` 事件仍会发出 | 只是纵深防御，加上会超出裁定，不需动作 |
| 观察 | F2 的一个既有角落（F2 没有扩大它）：草稿的表单挂上时打开时的项目就关着某个域，那一栏的 `Controller` 从未登记，`resetField` 对它无效，换走又换回仍会发那个域的 `null` 或 `[]`；挂上时开着、之后才关的照常 | 在设计 §3.3 B 接受的范围之内（§3 的第一条观察），不需动作 |
| 观察 | `planning-copy.test.tsx:229-230` 在 jsdom 里点禁用的按钮什么也不做，“不发”由 `disabled` 推出 | 证明这道守卫的是变异 `f1-mut-save-not-disabled`，不需动作 |
| 观察 | `close-policy.test.tsx` 的 `pendingCannotCloseOrResubmit` 在真实计时器下靠两份列表在点“保存”之前答到 | 假的 nerve 同步作答，结果确定；若间歇失败，签名是 “expected writes to have length 2”，不需动作 |
| 观察 | §3 M1 引的 `form.tsx:341-365` 是 F1 之前的行号（现为 `:348-373`） | 历史引用，不需动作 |
| 观察 | F1 禁用的保存没有文字说明原因 | 正上方两栏的 `role="status"` 加载即提示，不需动作 |
