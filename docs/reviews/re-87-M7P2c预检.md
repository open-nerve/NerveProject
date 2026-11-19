# M7 / P2c 预检

> 2026-11-19：结论 **修正后通过**（Ready after fixes）。没有严重问题；4 处重要、12 处次要。合同的每一项在 `f25cccad` 上都做得出来，与设计 §3.6、§3.10、§12、§15 E 没有矛盾，调研的 DP-1–12 核下来都成立。重要问题落在 spec §1 / R1 的收窄说明不全、验收缺拒绝次序的交叉格与对应变异、plan 的交错一项没有定形且不在允许清单里、新顶层测试名对两路的影响没有预先算。不需要改设计的取舍。须向负责人说明的只有一件：收窄比 spec 现在写的多两条（见 I1）。
> - **目标**：[P2c spec](../v0/M7-collaboration/specs/P2c-intake-assets.md)、[P2c plan](../v0/M7-collaboration/plans/P2c-intake-assets.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2（H11）、§3.6、§3.10、§3.11、§12、§15，台账 `research/p2c-facts.md`，[P2b spec](../v0/M7-collaboration/specs/P2b-intake-list-edit-delete.md)、[P2b 阶段记录](../v0/M7-collaboration/reviews/P2b-intake-list-edit-delete-review.md)、[re-86](re-86-M7P2b预检.md) 与[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §3–§5。
> - **基线**：main `f25cccad`（P2b 的 noff），worktree `codex/m7-p2c`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git、不起容器、不跑测试、不跑 `go build`。对照代码核了 issue 一侧的资产宿主判定（`Inspect` / `Check` / `read`）、`Locks.seeAs` 与 `decide`、`intakeEditor`、`refuseIntakeOff`、`CheckOpen`、`InTriage`、资产模块的 `Access.Within` 与四个写的用例、内容绑定、bootstrap 的宿主分派与错误转换、project 的 `AssetProjectArchived`（SQL、adapter、接口、转发、假实现）、项目锁的 SQL、access 的规则、资产契约与逐码的处理器测试、权限矩阵的资产行与目标守卫、真实宿主矩阵、两种交错的先例、分路脚本、e2e 的夹具。下文路径相对 worktree 根，`si/` = `server/internal/`，`ia/` = `si/modules/issue/app/`，`aa/` = `si/modules/asset/app/`，`bs/` = `si/bootstrap/`。

没有核实的：

- 各测试的实际耗时与两路 CI 的余量（没有运行；数字取自 P2b 阶段记录 `:55`）。
- sqlc 对加宽后的查询生成的类型；`make gen-check` 是否覆盖 sqlc 的生成物（按先例推定）。
- `issue/app` 假世界能否直接摆出“有条目行的工作项 + 各种授予”（P2b 的 `update_intake_test.go` 已经这样用，没有逐个读假实现）。
- e2e 两片各自的现状时间与新故事落在哪一片。
- 调研 §8 写的 Plane 的行为（没有读 `plane/`）；硬规则 `03:137,148` 以外的行号。

## 2. 发现

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §1、R1；plan T10（阶段记录“向负责人说明”） | **对 main 的收窄只写了一半。** §1 与 R1 说的两条（访客提交者从不能到能、非提交者的成员从能到不能）是准确的。但今天 `Check` 对工作项宿主只判项目归档与工作项归档，不看开关、不看条目状态，所以 P2c 还收窄了两条，而且连管理员一起：① 拒绝、重复的条目上，四个写对**所有人**（管理员、成员提交者）从 2xx 变成 409 `issue.intake_closed`——已绑定的附件从此只能靠删除整个条目去掉；② 收集箱关着时，在收集箱里的工作项上四个写对所有人 409 `issue.intake_disabled`（此时删除条目也是 409）。另有一条放宽之后的边角没有写：访客提交者的条目被接受后，他自己未绑定的预备资产也删不掉（回到 `issue.update`，403），与 §2.5 的“已关闭没有例外”同理，靠到期回收。这些都在设计 §12 “同一时段、资产路径读开关”的授权之内，不是新的取舍，但 R1 是给负责人看的那句话，不能比实际窄。 | `ia/asset_access.go:56-84`（今天的 `Check`：没有开关与状态）、`:106-109`（`issue.update`）；`si/modules/access/domain/rules.go` 的 `issue.update`（管理员、成员）；`aa/delete.go:26-41`（宿主先于行的判定）；设计 `:298` | §1 与 R1 补成四条：访客提交者得到；非提交者的成员失去；拒绝、重复之后所有人（含管理员）不能再写、已绑定的附件只能随条目删除；开关关着时所有人不能写。§2.5 加“接受之后访客提交者删不掉自己的预备资产（403），照到期回收”。T10 的阶段记录 §4 照这四条写。 |
| I2 | spec §4；plan“性质与变异”表 | **拒绝次序没有交叉格，变异表少了调研列出的一半。** spec §2.2 定了两串次序，§4 的验收只有各拒绝单独成立的格；硬规则要求“拒绝的次序在两个交叉方向上都钉住”，plan 自己的约束是“每个可见、守卫、锁的性质至少一个变异”。plan 的表有 6 个性质，调研的候选有 9 个，缺的正是次序与两个“不变”：归档先于开关、开关先于时段（两者都成立的格）；403 先于 409（非提交者的成员 × 已关闭、× 开关关）；只看自己的访客对别人的条目 404 `asset.not_found` 且与随机 id 逐值相同（先于任何 403）；时段不在 `Inspect`（放进去之后“关闭 + 开关关”会先答 `intake_closed`）；`notFound` 仍是 `ErrIssueNotFound`（换成 `ErrIntakeNotFound` 会以未声明的 `issue.intake_not_found` 透出）；读不变（`duplicateAsset` 的源、`getAsset`、`resolveAssetContent`、`getIssueAttachments` 在关闭、开关关时照常）。另缺两格：存的状态是 `snoozed` 而时间已到的条目照开着（`CheckOpen` 只看存的状态、不读时钟）；评论宿主的组合格（非提交者的成员在已关闭、开关关的条目上经评论加附件照常 201）。 | spec §2.2、§4；plan 的表；台账 `p2c-facts.md:243`（九个候选）；`ia/asset_access.go:64-66`（读的意图在项目读之前返回）、`:72-76`；`si/modules/issue/module_assets.go:55-57`（`AssetHostNotFound` 不含 `ErrIntakeNotFound`）；`bs/asset_access.go:106-114`（其余错误原样透出）；`si/modules/issue/domain/intake.go` 的 `CheckOpen`；硬规则 03 §5 | spec §4 加一条“次序的交叉格”：归档 × 开关关、开关关 × 已关闭、非提交者 × 已关闭、非提交者 × 开关关、只看自己的访客 × 别人的条目（与随机 id 逐值相同）；加“到期的暂缓照开着”“读在关闭、开关关时不变”“评论宿主在组合一层不变”三格。plan 的表补上：次序颠倒（两处）、时段进 `Inspect`、`notFound` 换值、读被拦，各标所在 Task（单测 T5，HTTP 的在 T7）。 |
| I3 | plan T8、允许清单；spec §4（“资产写在锁下等待时”） | **交错一项没有定形，哪一种都不在允许清单里。** 现有两种做法：① 受控封存（`TestAssetLateHostInterleaving` 的表：`completeAsset` 的两个事务之间宿主变了，第二个事务的 `Check` 照新状态答），加三格即可，但那是改已有测试的表，允许清单只写了宿主矩阵与权限矩阵；它也不是“在锁下等待”，是先后发生。② 真的锁等待：P2a/P2b 的 `racingWrite` / `serializeAtStatement` 只会造 issue 模块的路由（`newIssueRoute…`），资产写不经它；资产一侧的先例是整个应用加 `w.observe` 的探针（`TestAssetMembershipInterleaving`），等待点是 `LockIssueProjects`（复数，在调用者的账户锁与工作区的共享锁之后），不是 P2b 用的 `LockIssueProject`。plan 的“语句名的锁探针”没有说用哪一套、探哪条语句、是新顶层测试还是已有的表。 | `bs/assets_late_host_interleaving_test.go:15-113`；`bs/intake_edit_interleaving_test.go:29-41`（`editsIntake` 经 issue 路由）、`bs/intake_interleaving_test.go:60`（`serializeAtStatement`）；`bs/assets_membership_interleaving_test.go:18-36`；`aa/access.go:76-102`（账户锁 → 工作区 → 项目 → `Check`）；`si/modules/project/adapter/postgres/queries/issue_ports.sql:13-22` | spec §4 与 plan T8 定下来。建议：三格（拒绝、关开关、接受）用受控封存加进 `TestAssetLateHostInterleaving` 的表（不加顶层名、有恢复记录的断言），允许清单加一条“该表加行，不改已有行”；“资产写与分诊、开关在同一把项目锁上串行”另用一格真的锁等待（照 `TestAssetMembershipInterleaving` 的做法，探 `LockIssueProjects`），或写明理由不做（资产取的就是 `LockIssueProjects`，M5 的锁序测试已钉）。spec 的措辞改成与选定的做法相符。 |
| I4 | plan Global Constraints 的测试时间一条、T6–T8 | **新顶层测试名对两路的影响可以事先算出来，plan 没有安排。** bootstrap 现有 776 个顶层测试，按名字排序后单双号分路；P2b 自己的测试约 10 秒，却因两个新名字让第 0 路 +128 秒（799 秒，67%）。P2c 的 T7（HTTP 组合）、T8（交错）若各加一个顶层名，排在它们之后的测试再换一次路；第 0 路离 80% 的提醒线只有约 160 秒，而它还跑 `bootstrap` 以外的全部包（本阶段的 `issue/app` 单测、资产处理器的 8 个子测试、project 的 adapter 测试都落在这一路，量很小但都在这一路）。plan 只说“少加顶层测试”，没有说加几个、排在哪里、换路之后各路是哪些测试。 | `tools/go-test-shard.sh:15-17,25-29`；P2b 阶段记录 `:55`；设计 `:258`；`bs/asset_host_matrix_test.go:57`、`bs/assets_late_host_interleaving_test.go:15`（可以承接子测试的已有顶层测试） | plan 写明新顶层测试名的个数与名字（目标：T6 进 `TestAssetHostAccessRealMatrix`、T8 的三格进已有的表、权限矩阵的一行本就不加名字，只有 T7 一个新名字）。T7 定名之后、检查点 1 之前，用 `go test -list` 与脚本同一条 `awk` 列出改名前后两路的清单（只列名字，不跑测试），把换路的测试记进台账；预计某一路超过 90 秒或接近八成时照设计 §9 先报告。不靠改名调路的规矩不变。 |

### 次要

| # | 位置 | 发现与证据 | 修正 |
|---|---|---|---|
| m1 | spec §2.4；plan T3 | “加宽”没有定形。方法名 `AssetProjectArchived` 之后同时交回开关，名不副实；`(archived, intakeView, found bool, err)` 三个并排的布尔容易传反。调研 DP-4 建议过改名并交回结构体，spec 没有采纳也没有否决。 | spec 定名与形状（如 `AssetProjectState`，回答 `{Archived, IntakeView}` 与 `found`），sqlc 的查询名、`si/modules/project/module.go:132` 与 `:120-131` 的注释、`ia/asset_access.go:38-42` 的注释随之。 |
| m2 | spec §2.2、R2 | 开关与时段的两处判定没有说复用已有的函数。次序“归档 → 开关”已经在 `refuseIntakeOff`（`ia/submit_intake.go:92-100`，收 `LockedProject`），时段在 `IntakeFacts.CheckOpen`；资产路径若各写一遍就是近似副本。两者的文档注释今天只说提交、分诊与内容修改，§2.6 的清单里没有它们。 | 写明 `Check` 用 `CheckOpen`，归档与开关的次序与 `refuseIntakeOff` 共用一处（把它改成收两个布尔或收加宽后的值）；§2.6 的注释清单加 `refuseIntakeOff`、`CheckOpen`、`AssetProjectReader`。 |
| m3 | plan T5（“共用的提交者规则（从 `intakeEditor` 抽出）”）、T2 | 抽出提交者规则是纯搬移，却与行为改动放在同一个 Task；spec §2.3 的“先单独提交纯搬移”只说了 `Locks`。混在一起，“`update_intake_test.go`、`update_intake_refusals_test.go` 断言不改”就没有独立的证据。 | 抽出移到 T2，作为第二个纯搬移提交（调研的 Task 2 本来如此）；T5 只加调用。 |
| m4 | plan T4（“Go 文档注释加宽”） | T4 在行为（T5）之前就把 `ErrIntakeDisabled`、`ErrIntakeClosed`、`ActionUpdateIntake`、`rules.go` 的注释写成“资产也答”，这一个提交里句子比代码宽（硬规则 03 §5“文档的坑”）。契约的说明先行是契约先行的固有代价，Go 注释不必。 | Go 注释移到 T5；T4 只留契约、生成物与处理器的两行。 |
| m5 | spec §2.6 | 两个错误的**消息**没有说改不改。资产路径上用户看到的是 “…it can no longer be edited in the intake.” 与 “…turn it on to submit or triage work items.”（`si/modules/issue/domain/errors.go:82-86,97-101`）；P2b 的修改、删除已经沿用后一句。网页按码取文案，不受影响。 | 写一句“消息不改（与 P2b 同）”，或一并放宽；二选一，免得审查时当成漏改。 |
| m6 | spec §2.6（契约的说明） | `deleteAsset` 的说明写 “Repeated authorized deletion on a still-visible host returns 204”，`completeAsset` 写 “Repeated authorized confirmation returns the existing result”（`api/modules/asset.yaml:96-103,119-130`）。P2c 之后，在已关闭或开关关着的收集箱宿主上重复的删除、确认答 409（宿主的判定先于行，`aa/delete.go:26-41`、`aa/complete.go:38-45`）。“authorized”勉强盖得住，但新加的那一句要点明。 | 四个操作新加的一句写全：管理员与提交者、待处理或暂缓、开关开着；已关闭或关着时重复的请求也是 409。 |
| m7 | spec §2.1（`docs/` 一行）、plan T10 | 文档清单比调研 §9 少：设计 §3.11 `:159` 的开关清单（“提交、分诊、条目的修改与删除”）要加资产的写；设计 §3.6 `:109` 前半句“评论、表情、链接、**附件**按它们自己的规则照常可写”与后半句相抵，要把附件从前半句拿掉并加时段与开关；M5 交接 M5-M7-01 的“处理结果（M7/P2c）”。 | T10 点名这三处；spec §2.1 的 `docs/` 一行加“设计 §3.6、§3.11，M5-M7-01 的处理结果”。 |
| m8 | plan T6；spec §4（“宿主矩阵……四个写 × 角色与提交者”） | 真实宿主矩阵按意图（读、准备、删除）判，不按四个操作；它的世界里只有 alice（管理员）、bob（成员）、walt（只看自己的访客）、olga（外人），收集箱默认关着，末尾的子测试会归档项目。“加一列”实际要：`w.intakeOn`、经提交造条目、再加访客提交者、能看全部的访客、非提交者的成员，并排在归档的子测试之前。 证据 `bs/asset_host_matrix_test.go:57-63,75-76,126-136`；`bs/intake_fixture_test.go`（`intakeOn`：项目建成时关着）。 | spec 的措辞改成“准备、删除两种意图 × 两种用途 × 角色与提交者”；T6 写明种子与位置，作为该顶层测试的子测试。 |
| m9 | plan T6（检查点 1） | 检查点 1 放在 T6，HTTP 组合（T7）在它之后。T6 不经 HTTP，新码第一次经 `CheckResponse` 核对是在 T7；P2b 的检查点 1 放在组合测试之后。 | 检查点 1 移到 T7 之后（或 T8 之后），检查点 2 仍在 e2e。 |
| m10 | plan T3（“adapter 测试与变异”） | 加宽后的值经 `bs/ports.go:373-375` 转发，这一处没有测试：`fakeIssueProjects.AssetProjectArchived` 只记一笔 `"archive state"`，没有任何断言经 `issueProjects` 问它（`bs/asset_parent_ports_test.go:18-21`）；三处真实数据库的断言（`:76,82,163`）直接问 `project.Provide(...).IssueProjects`。转发时丢掉开关的变异要到 T7 才会失败。project 的 adapter 目录里今天没有这条读的测试。 | T3 加一格经 `issueProjects{projects: f}` 的转发断言（两个值都对得上），并写明 adapter 的测试放在哪个包；三处已有断言补上开关的值。变异“转发丢开关”列入表。 |
| m11 | plan T8（权限矩阵的一行） | 目标守卫只核路径里的 `{id}` 是本列的私有资产（`bs/permission_matrix_targets_test.go:101-103`，按工作区的列取），所以以工作项为宿主的行只能是 `createAsset`（spec 已经只写了这一个，成立）；它的宿主 id 在请求体里，守卫看不到。 | 该行加 `check`：回答的 `scope` 是本列的工作项；plan 写明 `columns: issueColumns`、`prepare: inTheIntake`，以及为什么其余三个操作不进矩阵。 |
| m12 | spec §2.2（`Inspect` 的次序）；plan T9、检查点 2 | ① “资产不在 404 `asset.not_found`”应是“宿主不在或看不到（含访客规则）”；资产行看不到的 404 在宿主之前，是资产模块的。② e2e 的新故事若是新文件会改变两片的划分；检查点 2 只写了“两路各自的增量”，没有写 e2e 两片。`e2e/fixtures/storage.ts:93-110` 的 `storedFile` 只走成功，403、409 的格要用回答式的调用。 | ① 改措辞。② T9 写明故事并入已有的 `h9-…` 还是新文件；检查点 2 记 e2e 两片的时间。 |

## 3. 逐项回答

1. **可行性**（都成立）：
   - `Locks` 的变体：`seeAs` 是“`GetIssue` → `decide`”两步（`ia/lock.go:230-243`），调用者三处（`lock.go:141` 的 `precheck`、`:226` 的 `see`、`ia/description_versions.go:96`）加资产一处；在两步之间插入“按工作项选动作”，`seeAs` 以常量的选择委托，是纯搬移，`NewLocks` 的签名不变、没有构造处要跟着改。读的行是 `GetIssue` 的，带 `Intake`，可以问 `InTriage()`（`si/modules/issue/domain/issue.go:113-115`）。
   - 选动作不泄漏：成员本来读得到条目；访客规则的 404 在 `decide` 里先于角色（`lock.go:283-297`），两种动作下外人都是同一个 404。
   - 加宽 `AssetProjectArchived`：唯一调用者 `ia/asset_access.go:67`；SQL 是一条按 id 的读（`queries/assets.sql:1-2`），加一列即可；跟随的是 `ia/asset_access_test.go:15-22`、`bs/asset_parent_ports_test.go:18-21` 两个假实现与 `:76,82,163` 三处断言，与 spec 说的数目相符。其他实现 `project.IssueProjects` 的假实现只有 `fakeIssueProjects` 一个。
   - 锁下的值：资产取的项目锁是 `LockIssueProjects`（`FOR NO KEY UPDATE`，`issue_ports.sql:13-22`），与分诊、项目修改同一把行锁；`Check` 在它之后重读工作项与项目，读到的就是锁下的状态与开关。
   - 提交者规则：`intakeEditor` 中间两步（`ia/update_intake.go:94-98`）可以原样抽出；资产路径不经 `hasIntakeRow`（`InTriage()` 已蕴含条目行），不会答 `issue.intake_not_found`。`read` 里的 `check` 变量在 `Inspect` 与 `Check` 各跑一次，拿到的是那一次判定的授予。
   - 只对在收集箱里的 `AssetIssue` 宿主判开关与时段：`Check` 对评论宿主同样拿到工作项（`ia/asset_access.go:140-144`），所以“判到评论宿主”的变异是真实的风险，限定 `q.Kind == AssetIssue` 是必要的。
   - 次序与 `aa/access.go` 相符：每个宿主先 `Inspect`（`:55`），闸门、锁之后按语义次序逐个 `Check`（`:98-102`），锁漏掉的父 404 在其后（`:103-105`），资产行在更后。spec §2.2 的 `Check` 一串与 `ia/asset_access.go:57-82` 的结构一一对应。
   - 码：四个写都以 `PrepareAsset` 或 `RemoveAsset` 到 `Check`（`aa/prepare.go:48`、`complete.go:38,89`、`duplicate.go:42,91`、`delete.go:31`），两个新码在四个操作上都到得了；`forbidden` 四个都已声明（`asset.yaml:53,106,132,168`）；issue 的错误经 `assetHostError` 原样透出。处理器测试的两行照 `business_errors_test.go:84-85` 的形状用 `shared.NewError` 造，不需要改处理器，T4 单独就是绿的。
   - `AssetHostNotFound`：保持 `notFound = ErrIssueNotFound`，项目不在的分支照旧（`ia/asset_access.go:71-76`）。
2. **行为**：
   - `duplicateAsset` 的源：读的意图，动作 `issue.read`；`Check` 在 `q.Intent == AssetRead` 处返回（`:64-66`），到不了开关与时段。不受影响。
   - 描述图片的绑定：`ContentBinder.checked` 只核放置（本人为本工作项准备、用途对、已发布；已绑定的保持原申请人），没有宿主的判定（`aa/content_bindings.go:46-60`、`content_binding_rules.go:9-35`）；`updateIntakeIssue` 已经经修改核心绑定。P2c 不用动这条路；访客提交者只卡在准备。
   - “已关闭”只指拒绝、重复：已接受的不是 `InTriage()`，走 `issue.update`；`CheckOpen` 对已接受的分支在这条路上到不了。锁下以这一次读到的为准：等待中被接受，管理员与成员成功、访客提交者 403；等待中被拒绝，409。
   - 开关的次序与 P2a/P2b 一致：归档 → 开关 → 关闭；非提交者的 403 在它们之前，与内容修改相同。
   - 到期的暂缓：存的状态仍是 `snoozed`，`InTriage()` 为真、`CheckOpen` 放行，不读时钟；正确，缺一格测试（I2）。
3. **与设计**：没有矛盾。图片与附件同一个规则、访客提交者也能加附件，是设计 `:109` 的字面；时段与开关是 `:298` 的字面；读照 `issue.read` 与 `:108` 一致；评论宿主的附件、图片照评论规则（非提交者的成员仍可经评论加附件，开关关着、已关闭时也可），与“评论照常可写”一致。要跟着改字的见 m7。
4. **缺的**：
   - 行为上会变红的已有测试：没有找到。`bs/intake_*_test.go`、`issue_intake_*_test.go` 不碰资产；`seedIntake` 写的是已接受的行；宿主矩阵与单测的工作项都没有条目行；e2e 的收集箱故事不碰资产。
   - 按动作、操作驱动的守卫：不加动作、不加操作，`bs/actions_test.go`、写锁表（只认 `issue` 标签，`bs/issue_write_locks_test.go:193-203`）、删除窗口、路由（`bs/asset_contract_test.go`）都不变；`InTriage` 没有按调用者清点的守卫。
   - 权限矩阵：资产行今天只在工作区宿主上（`bs/permission_matrix_assets_test.go:37-63`）；加行的限制见 m11。
   - e2e：`fixtures/intake.ts`、`fixtures/storage.ts` 现成，默认的夹具带本机存储；见 m12。
   - 允许清单的缺口见 I3；测试时间见 I4。
5. **plan**：T4 先于 T5 是对的（T7 起的每个请求都经 `CheckResponse`；`apitest.Main(m, "asset")` 要求声明的码当场有处理器测试回答，契约与两行同一 Task 也是对的）。没有红窗口：T2、T3 是搬移与编译跟随，T4 自己转绿，T5 之后没有已有测试变红。10 个 Task 现实；T5 最重（判定、单测、五个以上变异），按 m3、m4 调整后仍是 10 个。检查点的位置见 m9，变异见 I2，交错见 I3。
6. **§1 的收窄说明**：写出的两条准确（今天访客提交者 403、非提交者的成员 2xx，`ia/asset_access.go:106-109`），但不全，见 I1。

## 4. 核实为正确的事实（摘要）

- `AssetProjectArchived` 只有 `ia/asset_access.go:67` 一个调用者；资产模块的 app 层不需要改，宿主请求只带范围与意图。
- `intake.update` 的规则是项目的管理员、成员、访客，没有 `AllowCreator`（`rules.go` 的 `intake.update` 一行）；提交者规则在用例里，`Grant.ProjectAdmin` 来自那一次判定。
- 两个码的网页文案已有（`web/apps/web/core/lib/error-messages.ts` 与两份 `errors.json`）。
- 处理器测试的表按“码 × 操作”列，新两行各四个操作；资产的处理器包不引 issue 的 domain，照 `:83-85` 的写法造错误。
- 已绑定的附件按宿主删除（`aa/access_assets.go` 的 `assetVisibleFor`）：开着的收集箱宿主上，访客提交者能删管理员传的；已绑定的描述图片仍是 409 `asset.bound`，在宿主的判定之后。
- bootstrap 包现有 776 个顶层测试；第 0 路另跑 `bootstrap` 以外的全部包与 `server/tools`（`tools/go-test-shard.sh:25-29`）。
- 资产的锁序不变：`Check` 只多读已锁的那一行的一列，不加锁、不加语句。

## 5. 处理

待主控。

## 处理（主控，2026-11-19）

全部采纳，改入 spec 与 plan：

- I1：spec §1 列出四条收窄（访客提交者可写、非提交者 403、拒绝与重复对所有人 409、收集箱关着对所有人 409）与“接受之后访客提交者不能删自己的预备资产”；阶段记录向负责人逐条说明。
- I2：plan 的变异表补齐（次序、403 先于 409、只看自己的 404、时段不进 `Inspect`、`notFound`、读不变）；spec §4 加交叉格、到期的暂缓与评论宿主的格。
- I3：交错定为封口表加三行（允许清单写明）加一格真实地在 `LockIssueProjects` 上等待（照 `assets_membership_interleaving_test.go`）。
- I4：新的顶层名只有 T7 的一个，T6、T8 为已有测试的子测试或表行；T6 之后按分路算法算好换路清单。
- m1–m12：资产的项目读交回具名结构；复用 `refuseIntakeOff` 与 `CheckOpen`；提交者规则的抽出并入 T2 的纯搬移；Go 文档注释挪到 T5；错误文字不变；重复删除、重复完成的说明；T10 加设计 §3.11、§3.6 `:109` 的矛盾与 M5-M7-01；宿主矩阵的世界；检查点 1 挪到 T7 之后；bootstrap 转换的测试；矩阵行只能是 `createAsset` 且带 `check`；`Inspect` 的措辞、e2e 的文件与片的时间。
