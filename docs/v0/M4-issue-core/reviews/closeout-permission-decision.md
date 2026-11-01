# M4 收尾 R06：项目目录缺席与旧详情的权限来源

日期：2026-11-01；复核：2026-11-02。状态：**负责人批准的方案 A 已实施并通过联合复核；R06 已关闭。全 M4 审查及 T13 最终验收仍进行中，不构成整期通过或合并许可。**

## 1. 问题与范围

同一用户打开过私有项目 A 的页面，项目详情已被缓存；之后用户失去 A 的成员资格，新的完整项目目录成功返回且不再包含 A。当前 `ProjectStore.getProjectById` 仍返回旧详情里的成员角色，M4 的权限观察器看不到变化，已加载工作项继续可见，失权前发出的迟到列表还会被接收。

这次只复现了**客户端缓存和迟到回包没有失效**，没有证明服务端允许失权后的新请求；服务端授权未修改。严重度 Major。涉及 M3 的项目目录事实及 M4 的权限范围失效，不能以清掉单个页面缓存掩盖。

## 2. 根因与现有测试缺口

1. `web/apps/web/core/store/project/project.store.ts` 的 `getProjectById` 在详情、未归档列表和已归档列表中选择最后回答的**存在项**。一个更新、更完整的列表中没有该项目时，没有产生能否定旧详情的事实。
2. 同文件 `knownProjectIds` 仍枚举旧详情，`user/permissions.store.ts` 继续从旧 `Project.member_role` 计算角色。
3. `issue/permission-reads.ts` 按批准的方案 A 观察现有项目 getter；上游 getter 未变，水位不变。IssueRootStore 的旧 token、canonical 和 list 都仍被认为有效。
4. 现有“项目列表移除项目”的测试没有先读取项目详情，因而目录移除确实会让 getter 变空，全部通过；新测试只多加一次真实 `fetchProject`，结果翻转。

M3 设计 §7.3 是项目详情与两份列表的事实来源；M4 P10b spec §2.6 明确权限观察的 owner 是 IssueRootStore，但依赖上游目录提供当前事实。问题在两者的交界处，不应再加一套 Issue 专用项目缓存。

## 3. 可复现证据

实施候选：`83543d5f29cda1e76fc99df3885ddc54ecd47a3f`，分支 `codex/m4-closeout`，收尾基线 `01e40e113e11365a4eeacc2794973de2751dde61`。

[复现补丁](closeout-project-absence-reproduction.patch) 仅增加一个参数化测试和一个已有 helper 导入；运行真实 RootStore、ProjectStore、IssueRootStore、Reconciled 和生成客户端，只在 HTTP 传输边界安排应答，不替换权限 getter 或失效方法。

| 条件 | 结果 |
|---|---|
| 未预读项目详情，完整项目目录移除 A | 通过：旧数据隐藏，旧 token 无效，迟到列表丢弃 |
| 先读过项目详情，再让相同目录移除 A | 失败：六个观察点均不符，包括旧项目仍可见、token 仍有效、已有工作项与列表未隐藏、迟到工作项和列表被接收 |

历史复现命令（在上述未修复候选的独立 worktree 应用补丁；当前修复分支已含正式测试，勿重复应用）：

```sh
git apply --check docs/v0/M4-issue-core/reviews/closeout-project-absence-reproduction.patch
git apply docs/v0/M4-issue-core/reviews/closeout-project-absence-reproduction.patch
pnpm --dir web/apps/web exec vitest run core/store/issue/helpers/base-issues.store.test.ts -t 'R06 project directory absence'
```

诊断时实际结果：1 passed、1 failed、51 skipped（`-t` 刻意只选新对照，非验收全量）；1.53 秒，退出码 1。失败发生在行为断言，未发生编译、导入或环境错误。停报前精确恢复原测试文件，重新执行该文件 51/51 通过，无 skip；这只证明当时还原成功，**不表示 R06 已修复**。当时补丁未进入正常测试发现范围；负责人批准后已纳入正式回归，后续结果见 §6。

原始证据根：工作树 `.superpowers/sdd/M4-closeout/`。

| 收据 | 输入 SHA256 | 日志 SHA256 |
|---|---|---|
| `t10-project-absence-red.json`，exit 1 | `8042e59a828ab69e80da95d6fbec223ac1fc6d98735a59f5a2f3b50177eae1b6` | `484dff2123b6818216fa01f47bb2788937c34306c4c6a1937ce18d3db37b1ac8` |
| `t10-project-absence-baseline.json`，exit 0 | `657599c0ace9ab4543461feb56f967fb69cc09196da018c24d47581fd15b7927` | `66b15e355f81f8eeef1af90194083e36432f9f642235dd26aef307c788a0e24b` |

两次运行输入均未改变。补丁 SHA256：`9351ba7b15978d0cfe538d05f3dbd0884a70b2f3b322998c86a444964d8e873c`；还原后的测试文件 SHA256：`f57b63c90f42201c0423e54c525b7daf02985717fb41c47c33033be0dea02435`，与上述实施候选完全一致。还原收据 `t10-project-absence-restored.json`。

## 4. 已批准方案 A：修正 ProjectStore 的事实选择

由项目目录的拥有者统一处理完整列表中的缺席：同一工作区、同一归档类别的成功完整列表若比某份旧项目记录更新，且已不含该项目，旧记录不能继续证明项目当前可见。项目消失只撤销当前可见性，不代表永久删除；以后新的有效详情或目录应答可以重新建立可见性。归档与未归档两份目录的覆盖范围必须分别处理。

IssueRootStore 仍是工作项权限失效的唯一 owner，沿既有观察链使旧 token、实体和列表失效；不新增端点、不增加第二项目缓存、不扩大删除水位、不放宽架构守卫。

负责人于 2026-11-01 回复“A：修正 ProjectStore 的事实来源，继续收尾（推荐）”。按以下顺序实施：

1. 先同步 M3 设计 §7.3、M4 设计权限条款、closeout spec/plan，写明完整目录缺席、归档类别、读取时序和恢复的语义。
2. 将诊断补丁转成正式回归，在 ProjectStore 本层再覆盖旧详情/完整列表两种顺序、归档与未归档切换、列表拒绝、其他工作区/项目、新详情恢复及已确认写入与回包交错。单个成功回包中的缺席不生成永久墓碑。
3. 根因修复放在 ProjectStore 的事实选择职责中；检查所有 getter 使用者和项目权限 getter。M4 继续使用现有权限观察接口。
4. 补齐 Issue 整合反例：whole/part/detail/verify、失权→恢复→旧答、先有详情和先有目录、无变化目录、公共项目仍可见但 member_role=null、管理员/成员/访客。新身份的读取才能恢复显示，旧答不能恢复旧身份数据。
5. 运行项目/权限/Issue 聚焦回归与相关页面故事，复审受影响全链；最终 T13 按新的输入补验 C12 权限格、运行完整 F 并取得准确分支 CI。其余 T10–T14 Step 1 继续，仍先不合并。

预期代价：这会改变共享 ProjectStore 的可见性选择，需要 M3/M4 的联合回归；归档/恢复时可能暂时隐藏旧记录直到新目录或详情证明当前状态。不能把任意列表失败或非完整分页当作撤权证据。

## 5. 历史停止位置与恢复

按负责人原话“架构级问题，比如架构回归，或者跨模块协同调整……停下之前先提交已完成的工作”，本次先提交 R01–R05 的已完成修复以及 R06 的诊断补丁、报告和推荐方案，再停在 T10 等待范围决定。

T1–T9 已完成；T10 未完成，T11–T13、T14 Step 1 尚待后续。C12 的既有三引擎记录保留，但最终输入绑定未完成；D18、总体设计完成标记、最终 main CI 和清理都仍待未来集成授权。主目录保持 main，当前工作分支不合并、不删除。

负责人已批准上述跨模块范围，现恢复 T10。此次批准不改变阶段完成后先不合并的边界。

## 6. 实施和联合复核

生产改动在 `project.store.ts` 和 `project-member.store.ts`。ProjectStore 统一裁定完整目录的正面/缺席事实、保留仅含 ID/归档类别/应答顺序的来源记录，并让所有项目 getter 与侧栏投影使用同一判定。目录及详情请求记录发出时已接受的目录、详情和完整写确认顺序；期间已接受的新事实优先，冷详情和工作区暂未加载亦同。快照直接读取已接受源的顺序，避免观察器在来源元数据续写前发出的新请求被误认为旧请求。新有效读取及缺席后发起的写入的完整答复可恢复，来源记录不保存第二份 Project，也不构成永久墓碑。

已确认写入的重放不能越过之后的独立事实；遇到新事实须替换相应项目或按缺席剔除，不能仅跳过旧修改而留下旧原始行。在途项目写入与成员角色/增删调用同一个 `withCurrentProject`，事实消失或改变后不把旧答覆回新事实，恢复后亦然。临时观察在成功/失败都释放，发送仍经各资源原有串行队列；无变化目录、失败和无关范围不抹去正常确认。退出、排序同样受此边界约束；永久删除仍按服务器的删除确认处理。IssueRootStore、权限签名、水位、服务端授权和架构守卫未改。

所有收据位于 `.superpowers/sdd/M4-closeout/`，每份含 HEAD、命令、输入清单 SHA、完整日志 SHA、耗时和 `inputUnchanged`；以下均已核实际输出。红例中的 skip 来自刻意 `-t` 聚焦，不作全量通过证据。

| 性质 / 对照 | 红例 | 修复复验 |
|---|---|---|
| 旧详情、完整缺席、归档类别、迟到详情及恢复 | `t10-project-visibility-red`：9 failed / 10 passed | `t10-project-visibility-green`：19 passed |
| 确认重放与迟到 update/join/archive/restore | `t10-project-confirmations-red`：5 failed | `t10-project-confirmations-green`：6 文件 130 passed |
| 新归档类别消失后不能回退旧类别 | `t10-project-class-red`：2 failed | `t10-project-class-green`：6 文件 132 passed |
| 侧栏/目录/编号 getter 同源 | `t10-project-projections-red`：2 failed / 5 passed | `t10-project-projections-green`：10 文件 214 passed |
| 成员角色旧确认不能覆盖恢复后的访客角色 | `t10-project-member-red`：1 failed | `t10-project-member-green`：10 文件 215 passed |
| 热详情下 whole/part/verify、权限恢复 | 原 HEAD 的 `t10-project-integration-red`：11 failed / 4 passed；其中两条详情初例因假传输悬挂，不以超时作反证 | 正式测试已允许错误分支的附属请求得到应答，`t10-project-detail-red` 的 2 例明确失败在旧数据被接受/不该发的附属请求；两次都精确恢复候选，见 `*-restored.json` |
| 工作区暂未加载时区分先前与新目录 | `t10-project-workspace-order-red`：1 failed / 1 passed | 完整前端套件与后续联合套件覆盖通过 |
| 退出/排序旧确认晚于重新加入 | `t10-project-other-writes-red`：2 failed | `t10-r06-joint-final`：62 文件 911 passed，6.48 秒，无 skip |
| 缺席后发起的成功完整写答应恢复项目（冷/热详情） | `t10-project-fresh-write-red`：2 failed | `t10-r06-joint-full-proof`：62 文件 913 passed |
| 归档/恢复并再次移除后，旧类别的目录不能复活项目 | `t10-project-directory-order-red`：4 failed | `t10-r06-joint-directory-proof`：62 文件 917 passed |
| 新恢复详情不能被更早发送的目录覆回旧角色 | `t10-project-positive-order-red`：2 failed | `t10-r06-joint-source-proof`：62 文件 919 passed |
| 由刚接受的目录观察器发出的新详情不能被误判为旧请求 | `t10-project-observer-order-red`：1 failed，管理员新答被替成访客旧事实 | `t10-r06-joint-observer-proof`：62 文件 920 passed，6.63 秒 |

早期检查与失败均保留：`t10-r06-web-full`、`t10-r06-lint-2`、`t10-r06-e2e` 曾通过，但其输入早于后续的退出/排序与目录时序修复，不能作为最后候选。首轮 lint 的 13 个测试非空断言和 4 处可空泛型错误已按现有惯例修正；末次补验发现一个 `dropped` 泛型推断缺口，显式指定 `Project`，未改变类型/守卫。完整前端还揭示旧 `work-item-level.test.tsx` 将所有请求自动回成工作项详情：目录返回形状不合法，原权限 ABA 断言并未验证真实角色切换。现限定详情假响应的 method/path，并分别断言两次目录读取无错误、值准确；8/8 DOM 测试通过。失败收据 `t10-r06-web-accepted-order` / `t10-r06-lint-accepted-order` 保留，不以它们作为产品反证。

最后候选的四条检查使用相同输入清单 SHA256 `a11ff0156991050d65a246228707c15c0b5e565b2a80d4795331806739549202`，运行中输入不变。完整正文随后只更新本次事实与设计说明；产品输入等价以逐文件清单核验。

| 检查 | 实际结果 | 收据 / 完整日志 SHA256 |
|---|---|---|
| `make test-web` | 18/18 task，17 缓存、web 实际运行；281 文件 3061 项，31.02 秒 | `t10-r06-web-complete` / `eda2f8575e4f2da147df98ed4c3d1cd27b02309bd2aafa8b5b78e7d889875485`；web 细项另存 `t10-r06-web-complete-web.log` |
| `make lint-web` | 54/54 task，11.47 秒；76 规则/13 例外无命中，1294 翻译键均有定义 | `t10-r06-lint-complete` / `5488c08098a1a6aac5ff9202b68dbd9248f714b51dfaef32e3ee964654ab3fd0` |
| `make knip` | exit 0，1.97 秒 | `t10-r06-knip-complete` / `204540c4aeb66000d0178a659a919dd26c4332797bf7016bb524dd480045b89e` |
| `make e2e` | 新构建 297/297，无意外 skip/flaky，117.76 秒 | `t10-r06-e2e-complete` / `a0e627884b01527c3c079802428770eb177accd7e5f50f677f63a5f520c2ecb2` |

修复文件、成员写入调用者、权限观察链、目录/侧栏投影和相关测试已全文复审。R06 的源头问题及读写交错回归已闭环；T13 仍须核最终产品输入、补三引擎 C12 权限格并完成完整 F，全期 T10 和分支终审也未完成。

历史停报提交 `0d1983f1f11fa51dda1ba521bbc87ccc005704f0` 的 [CI 41318564500](https://github.com/open-nerve/NerveProject/actions/runs/41318564500) attempt 1 已全绿：web `124883727326`、server `124883727420`、e2e `124886004338`。这只绑定停报提交；新修复的分支 CI、T13 最终完整 F/C12 输入及全期审查尚待完成。
