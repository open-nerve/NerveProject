# M4 收尾 R06：项目目录缺席与旧详情的权限来源

日期：2026-11-01。状态：**已复现，待负责人决策；未实施修复，不通过最终验收。**

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

复现命令（先在该分支应用补丁）：

```sh
git apply --check docs/v0/M4-issue-core/reviews/closeout-project-absence-reproduction.patch
git apply docs/v0/M4-issue-core/reviews/closeout-project-absence-reproduction.patch
pnpm --dir web/apps/web exec vitest run core/store/issue/helpers/base-issues.store.test.ts -t 'R06 project directory absence'
```

实际结果：1 passed、1 failed、51 skipped（`-t` 刻意只选新对照，非验收全量）；1.53 秒，退出码 1。失败发生在行为断言，未发生编译、导入或环境错误。测试结束后已精确恢复原测试文件，重新执行该文件 51/51 通过，无 skip；这只证明还原成功，**不表示 R06 已修复**。补丁暂不加入正常测试发现范围，保留为待决定的诊断附件。

原始证据根：工作树 `.superpowers/sdd/M4-closeout/`。

| 收据 | 输入 SHA256 | 日志 SHA256 |
|---|---|---|
| `t10-project-absence-red.json`，exit 1 | `8042e59a828ab69e80da95d6fbec223ac1fc6d98735a59f5a2f3b50177eae1b6` | `484dff2123b6818216fa01f47bb2788937c34306c4c6a1937ce18d3db37b1ac8` |
| `t10-project-absence-baseline.json`，exit 0 | `657599c0ace9ab4543461feb56f967fb69cc09196da018c24d47581fd15b7927` | `66b15e355f81f8eeef1af90194083e36432f9f642235dd26aef307c788a0e24b` |

两次运行输入均未改变。补丁 SHA256：`9351ba7b15978d0cfe538d05f3dbd0884a70b2f3b322998c86a444964d8e873c`；还原后的测试文件 SHA256：`f57b63c90f42201c0423e54c525b7daf02985717fb41c47c33033be0dea02435`，与上述实施候选完全一致。还原收据 `t10-project-absence-restored.json`。

## 4. 推荐方案 A：修正 ProjectStore 的事实选择

由项目目录的拥有者统一处理完整列表中的缺席：同一工作区、同一归档类别的成功完整列表若比某份旧项目记录更新，且已不含该项目，旧记录不能继续证明项目当前可见。项目消失只撤销当前可见性，不代表永久删除；以后新的有效详情或目录应答可以重新建立可见性。归档与未归档两份目录的覆盖范围必须分别处理。

IssueRootStore 仍是工作项权限失效的唯一 owner，沿既有观察链使旧 token、实体和列表失效；不新增端点、不增加第二项目缓存、不扩大删除水位、不放宽架构守卫。

批准后按以下顺序实施：

1. 先同步 M3 设计 §7.3、M4 设计权限条款、closeout spec/plan，写明完整目录缺席、归档类别、读取时序和恢复的语义。
2. 将诊断补丁转成正式回归，在 ProjectStore 本层再覆盖旧详情/完整列表两种顺序、归档与未归档切换、列表拒绝、其他工作区/项目、新详情恢复及已确认写入与回包交错。单个成功回包中的缺席不生成永久墓碑。
3. 根因修复放在 ProjectStore 的事实选择职责中；检查所有 getter 使用者和项目权限 getter。M4 继续使用现有权限观察接口。
4. 补齐 Issue 整合反例：whole/part/detail/verify、失权→恢复→旧答、先有详情和先有目录、无变化目录、公共项目仍可见但 member_role=null、管理员/成员/访客。新身份的读取才能恢复显示，旧答不能恢复旧身份数据。
5. 运行项目/权限/Issue 聚焦回归与相关页面故事，复审受影响全链；最终 T13 按新的输入补验 C12 权限格、运行完整 F 并取得准确分支 CI。其余 T10–T14 Step 1 继续，仍先不合并。

预期代价：这会改变共享 ProjectStore 的可见性选择，需要 M3/M4 的联合回归；归档/恢复时可能暂时隐藏旧记录直到新目录或详情证明当前状态。不能把任意列表失败或非完整分页当作撤权证据。

## 5. 停止位置

按负责人原话“架构级问题，比如架构回归，或者跨模块协同调整……停下之前先提交已完成的工作”，本次先提交 R01–R05 的已完成修复以及 R06 的诊断补丁、报告和推荐方案，再停在 T10 等待范围决定。

T1–T9 已完成；T10 未完成，T11–T13、T14 Step 1 尚待后续。C12 的既有三引擎记录保留，但最终输入绑定未完成；D18、总体设计完成标记、最终 main CI 和清理都仍待未来集成授权。主目录保持 main，当前工作分支不合并、不删除。
