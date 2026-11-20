# M7 / P2c 阶段记录：Triage 宿主的资产判定

> 状态：已完成，合入 main（noff `7dfaa8a1`），§9 为审查与集成的记录。spec 与 plan 见 [P2c spec](../specs/P2c-intake-assets.md)、[P2c plan](../plans/P2c-intake-assets.md)，预检 [re-87](../../../reviews/re-87-M7P2c预检.md)（修正后通过，4 处重要、12 处次要全部采纳）。基线 main `f25cccad`（P2b 的 noff）。

## 1. 交付

- 资产的四个写（`createAsset`、`completeAsset`、`duplicateAsset` 的目标、`deleteAsset`）在宿主是“在收集箱里的工作项”时按 `intake.update` 判定，非管理员须是提交者（与 P2b 的内容修改共用 `refuseNonSubmitter`）；锁下复用 `refuseIntakeOff`（归档 → 关着）与 `CheckOpen`（拒绝、重复 409）；评论宿主、资产的读、已接受的工作项照旧。
- `Locks.seeChoosing`：读一次工作项、按工作项选动作、再判定（`seeAs` 委托给它）。
- 资产的项目读一次交回具名结构 `AssetProject{Archived, IntakeView}`。
- `asset.yaml` 四个写加 `issue.intake_disabled`、`issue.intake_closed`，各一句说明。
- 证据：单测（含交叉格）、真实宿主矩阵 48 格、H11 的 HTTP 组合测试、封口表三行与一格真实的锁等待、`createAsset` 的权限矩阵一行、e2e 的接口故事。

## 2. Task 与提交

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `44ab1091` | spec、plan、预检 re-87；设计 §14；P2b 阶段记录 §9 |
| T2 | `959a2c9f` | 两次纯搬移：`Locks.seeChoosing`、`refuseNonSubmitter` |
| T3 | `d95a0827` | 资产的项目读带开关（`AssetProject`） |
| T4 | `9b1af055` | 契约的两个码与说明、处理器测试两行 |
| T5 | `f3b7fea0` | issue 一侧的判定；文档注释 |
| T6 | `79e4be74` | 真实宿主矩阵 |
| T7 | `c4ad6c8a` | H11 的 HTTP 组合测试 |
| T8 | `bfa60ba6` | 交错与权限矩阵的一行 |
| T9 | `c61258e0` | e2e 故事（H11 的资产，PAT 与会话对等） |
| T10 | `92a0e11b` | 文档与交接 |
| 审查修复 | `9b5426b4`、`0a48086f`、`1880d4e2` | 整分支审查的 m2、m3、m4 与检查点 2 暴露的项目写锁守卫的归类 |

## 3. 局部裁定（实施中）

- T2：`intakeEditor` 改为“条目行 → `refuseNonSubmitter` → 非管理员的访客字段”，与原来的 `switch` 等价（以访客身份加入的工作区管理员 `ProjectAdmin` 为真，所以访客字段的判断加了“非管理员”）。
- T3：端口与类型都叫 `AssetProject`，字段叫 `IntakeView`（与 `IssueProject`、`LockedProject` 和列名一致）；三处编译跟随的断言不加开关（意思不变），开关由新的 adapter 测试与转换子测试钉住；历史文档里的旧名 `AssetProjectArchived` 不改。
- T5：一个谓词 `inIntake`（工作项宿主、写的意图、`InTriage()`）决定选动作、提交者规则与锁下的判定；`refuseIntakeOff` 保留 `LockedProject` 参数，在 `Check` 里就地转换，不动 P2a、P2b 的四个调用者；“开关先于提交者规则”没有能编译的变异（开关只在 `Check` 读得到），由两个交叉格钉住。
- T6：矩阵按意图（读、准备、删除）× 用途，另加局外人一列。
- T7：“附件挂上”就是确认上传（这套接口没有另外的挂接操作）；访客提交者删掉管理员在他条目上已绑定的附件（spec §2.5）一并测了；`waitForStartupRuns` 在造好数据后、第一个子测试前等一次（每个拒绝都比对整张 `file_assets`）。
- 检查点 2：`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 确定失败——T8 给 `createAsset` 加的矩阵行用工作项表的列（`issueColumns`），守卫 `writesOnAProject` 按“行名项目表的列”把它算成项目上的写。资产的写不是项目模块锁表里的写：它走资产模块自己的锁序（调用者、宿主的工作区 `FOR SHARE`、它们的项目、草稿，M5 P2 spec §2），由 `assets_lock_order_test.go` 钉住，`projectWrites` 的那套测法不适用。修法：守卫按标签排除资产的写（与 issue、planning 同一个机制），守卫自己的形状测试加一格（asset 标签、行名项目表的列也不算）。**偏离 plan 的允许清单**（改了已有的守卫），去掉排除的变异让形状测试失败。（修复的第一版 `9b5426b4` 按行的列表排除、并把原因写成“经 issue 的 `Locks`、不锁工作区”，限定复审 R1 指出原因不对——资产的写实际先 `FOR SHARE` 工作区——改为按标签排除、写明真正的原因。）
- T8：锁等待的一格放在 `TestAssetLateHostInterleaving`（宿主在 Inspect 之后变化正是它测的），锁的持有者是真实的拒绝；接受那一行先把 bob 降为访客（表里写死了 bob，成员非提交者到不了 `Check`）。

## 4. 向负责人说明

- **四条收窄（设计 §3.6 已批准，R1）**，只对在收集箱里的工作项（待处理、暂缓、拒绝、重复）的宿主：① 访客提交者从 403 变为可以给自己的条目加图片与附件；② 不是提交者的成员从可以变为 403；③ 拒绝、重复的条目上资产的写对所有人（含管理员）都答 409 `issue.intake_closed`，已绑定的附件只能随删除整个条目一起去掉；④ 收集箱关着时资产的写对所有人都答 409 `issue.intake_disabled`。另：接受之后照 `issue.update`，访客提交者不能再删自己没绑定的预备资产。
- **与 Plane 的不同**（plane-diff 已登记）：Plane 服务端只按项目角色、不看条目状态与创建者；已绑定附件的删除 Plane 是管理员或上传者，Nerve 是宿主允许写的人。
- **最终分支 CI 暴露的 M4 产品缺陷（已按根因修）**：M4 的 I2 页面故事在 CI 上失败一次——工作项详情的标签选择器在一次选择的写还没结束（PATCH 之后还要刷新动态与评论）时拒收下一次选择，但 Headless UI 的选项不跟随 Combobox 的禁用，选项看上去可点、点了却被悄悄丢掉。修法：选择器忙时每个选项也禁用（`aria-disabled`），点击会等到可选；详情、列表行、新建表单三处的标签选择器共用这个底座，一并修好（`1880d4e2`，单测与 e2e 的确定性复现、去掉修法的变异、I2 连跑 10 遍）。这不是已知的偶发，没有重跑。
- **两路**：见 §6。两路测试分配的问题（第 0 路的负担）已在 P2b 提出，等负责人决定。

## 5. 本地门禁与 CI

- 检查点 1（`c4ad6c8a`）：本机七项全过（cp1）；分支 CI 44620044098 attempt 1 全绿。
- 检查点 2（`c61258e0`）：gen-check、lint-go 通过，第 0 路的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 确定失败（见 §3），修后由最终候选补全七项。

## 6. 测试时间与两路

| 时点 | 本机 bootstrap 第 0 路 | 第 1 路 | 合计 | CI server (0) Test | CI server (1) Test |
|---|---|---|---|---|---|
| P2b 基线（P2b fin；main CI 44488525189） | 552.0 秒 | 519.5 秒 | 1071.5 | 814 秒 | 642 秒 |
| 检查点 1（`c4ad6c8a`） | 523.8 | 560.8 | 1084.6 | 708 | 662 |
| 检查点 2 / 最终候选（fin2 `0a48086f`、fin3 `612d77c9`） | 532.4 / 518.2 | 545.9 / 555.7 | 1078.3 / 1073.9 | 719（最终分支 CI） | 706 |
| main（`7dfaa8a1`，main CI 44647295601） | | | | 705 | 670 |

P2c 只加了一个 bootstrap 顶层名（T7 的 `TestIntakeItemAssetsThroughTheApp`），它让 366 个测试换路，T6 提交前算好的清单与实际逐行相同；新测试自己约 2–3 秒，宿主矩阵 +0.15 秒，`TestAssetLateHostInterleaving` +4.4 秒。检查点 1 的 CI 上第 0 路 −106 秒、第 1 路 +20 秒（对合入 P2b 后的 main），都在 90 秒以内；那一次的运行机器正常（`issue/adapter/http` 92.8 秒）。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- `server/internal/bootstrap/ports_test.go` 522 行（基线 486，P2c 加 36），超过约 400 行，下次动它之前先拆。
- 负责人的多选（`MemberDropdown` → `CustomSearchSelect`）没有忙的守卫，下一个集合从渲染出的值算：PATCH 的回答到了、页面还没 `confirmChange` 的那几个微任务里再点，会发出旧的集合（错的请求体，不是超时）；没有观察到，没有改，交 P5/M8 看。
- `writesOnAProject` 按标签排除工作项（issue）、计划（planning）与资产（asset）的写；资产的写有自己的锁序与锁序测试。以后加别的按宿主授权、自有锁序的写，照这一条。

## 8. 文档与交接（T10）

- plane-diff：收集箱条目的资产一行。
- 硬规则 03：§4 加 P2c 一段（`inIntake` 与判定的组成）；§5 加一条（新的顶层测试名让后面的换路，预先算好）。
- 设计 §3.6（“附件按它们自己的规则照常可写”与资产一句的矛盾改写）、§3.11（关着时条目上资产的写也 409）；§14 的 P2c 一行在合入后的下一阶段 T1 改为已完成。
- M5-M7-01：服务端的处理结果；页面与三浏览器的页面故事仍待 P5。

## 9. 整分支审查与集成

- 检查点 2（`c61258e0`）：第 0 路的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 确定失败（§3），检查点 2 的分支 CI 被修复的推送取代。
- 整分支审查（fable，只读，`f25cccad..c61258e0`）：Ready to merge，没有严重、重要的，5 处次要：m1 阶段记录的待填（在此回填）；m2 T7 的子测试按次序成故事（写明）；m3 拒绝的子测试加重复删除、重复确认两格；m4 设计 §3.10 的 `intake.update` 一行补资产；m5 `bootstrap/ports_test.go` 522 行（§7）。与守卫的修复一起在 `9b5426b4`。
- 限定复审（`9b5426b4`）：R1 重要——守卫修复写的原因不对（资产的写走自己的锁序、先 `FOR SHARE` 工作区，不是“经 issue 的 `Locks`、不锁工作区”），主控在代码里核实后改为按标签 `asset` 排除并写明真正的原因（`0a48086f`）；R2、R3 随之消失，R4 §7 的路径更正。
- 最终候选 fin（`9b5426b4`）：六项通过，F11 (page) 以 M6-M8-07 的签名失败一次（随 R1 的修改整套重跑，不单独重跑）；fin2（`0a48086f`）七项全过。
- 分支 CI 44635307447（`0a48086f`）failure：e2e (1) 的 M4 I2 页面故事——标签选择器吞掉快速的第二次点击（§4），不是已知的偶发，没有重跑；按根因在 `1880d4e2` 修（选项随忙禁用），阶段记录 `612d77c9`；fin3 七项全过；分支 CI 44642775608 attempt 1 全绿（server (0) Test 719 秒、server (1) 706 秒、web Unit 493 秒、e2e 520/622 秒）。
- `--no-ff` 合入 main `7dfaa8a1`（父 `f25cccad`、`612d77c9`，树与分支相同）；main CI 44647295601 attempt 1 全绿：server (0) Test 705 秒、server (1) 670 秒、web Unit 490 秒、e2e 647/607 秒。
- 归档 `.superpowers/sdd/M7-P2c-integration-20261120`（3095 个文件、unknown 0，清理后复核一致）；P2c 的 worktree、本地与远端分支已删除（先核实已含在 main）。
