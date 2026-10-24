# M4/P4 描述与协作：评审与裁定记录

| 项 | 内容 |
|---|---|
| Phase | M4/P4 `issue-collaboration` |
| 日期 | 2026-10-25 |
| 状态 | **实施前暂停**：A1 的清洗实现放置与基础架构规则冲突，待负责人裁定；不是阶段验收或 Claude 终审通过记录 |
| spec / plan | [spec 草案](../specs/P4-issue-collaboration.md) / [plan 草案](../plans/P4-issue-collaboration.md) |
| 分支 | `codex/m4-p4` |
| main 基线 | `56d575fb8ff6575880d1edef65219728a81f2b12` |
| 工作树 | `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m4-p4`；主目录保持 `main` |

## 1. 评审方式

- 按[交接入口](../../claude_to_gpt/README.md)的顺序读取 01、03、02、04、05、06，核对 M4 设计的 P4–收尾、完成标准、进度表，以及 P3 的 spec、plan、review 样板和此前交给 P4 的事项。
- 依交接 02 §3 调整计划产出方式：保留文件、接口、验收、执行命令与关键性质的变异计划，不把尚未运行的原型或逐块重放写成已验证。架构子任务负责 spec/plan 草案，主任务核对既有代码与基线。本次暂停保存 spec 和 16 个任务纲要；逐任务完整接口、命令与可执行性自检尚未完成，留待 A1 裁定后补齐。
- 在准备 Task 1 时发现 A1，停止进入预检和产品实现。当前没有 Task 实现、逐 Task 评审、整分支终审或 P4 的变异结果。下列测试是起步基线和架构放置探针。

## 2. 验收标准核对

### 2.1 已运行的检查

| 检查 | 实际结果 | 证据范围 |
|---|---|---|
| `pnpm install --frozen-lockfile` | 退出 0；锁文件未变 | 工作树安装；没有全局安装或新增依赖 |
| `make test` | 退出 0；server 和 server/tools 的测试通过 | P3 起步基线；日志 `/tmp/nerve-m4-p4-baseline-test.log` |
| `make test-web` | 退出 0；17/17 个任务成功，全部命中 Turbo 缓存 | 基线缓存结果；日志 `/tmp/nerve-m4-p4-baseline-web.log` |
| `make lint-web` | 退出 0；54/54 个任务成功，其中 37 个命中缓存 | 基线；日志 `/tmp/nerve-m4-p4-baseline-lint-web.log` |
| `node tools/keywords.mjs`（启动时） | 72 条规则、23 条例外，无命中 | 已把 phase 改为 `M4/P4` |
| 架构放置合成图探针 | domain 放置失败；adapter 放置通过 | §3/A1；只验证依赖方向，未安装或运行 bluemonday |
| 移除临时探针后 `go -C server test -count=1 ./internal/archtest` | 退出 0；1.481s | 现有架构守卫未改动 |

### 2.2 尚未执行的 P4 验收

S3、描述/提及/版本、评论/表情/动态、权限矩阵增量、交错 14/15、接口故事、目录驱动删除增量、关键性质的实际变异、完整门禁与 Claude 整分支终审都未执行。M4 设计 §14 不新增勾选；P4 不标已完成。

## 3. 准备阶段的决定与待裁定问题

### A1：清洗库应放在哪一层（待负责人裁定）

**问题。** M4 设计要求在 `issue/domain/sanitize.go` 使用 bluemonday；M0 已建立并由测试强制执行的规则，则禁止 domain、app 和 shared 直接或间接依赖任何第三方库。按 M4 的文件放置直接实现会使架构检查失败；若为它放宽守卫，会改变已经建立的纯层边界。清洗库本身已获设计批准，冲突是放置位置，不是依赖许可。

**依据。**

- [M4 设计](../M4-design.md) §3.7、§6.2、§12/P4 Task 1、§16 明确指向 `issue/domain/sanitize.go` 或 `issue/domain`；§6.1 批准 bluemonday 和其传递依赖，并要求 issue 接受现有 archtest 检查。
- [M0 设计](../../M0-foundation/M0-design.md) §3 的模块边界表和 §3.7 的第 2、10 条及传递检查禁止该依赖。
- 现有 `server/internal/archtest/rules_test.go` 的 `innerLayersArePure` / `isInfrastructure` 拒绝直接导入；`purity_test.go` 的 `TestPureLayersReachNoInfrastructure` 拒绝间接导入。把依赖转藏在 shared 也不成立。

**实际验证。** 临时测试调用现有 `check(graph)` 和传递检查函数，输入以下两张合成依赖图，不增加第三方依赖、不改守卫：

```text
现设计：issue/domain → github.com/microcosm-cc/bluemonday
候选：  issue（module.go）→ issue/app、issue/adapter/content
        issue/adapter/content → issue/app、issue/domain、bluemonday、x/net/html
        issue/app → issue/domain
```

运行 `go -C server test -count=1 ./internal/archtest -run '^TestP4SanitizerPlacementProbe$' -v`，退出 1：

```text
design_domain_placement: FAIL
internal/modules/issue/domain imports github.com/microcosm-cc/bluemonday:
domain and app import only the standard library (not net/http or database/sql),
their own module's inner layers and internal/shared
proposed_adapter_placement: PASS
```

探针随即移除，原有 archtest 重跑通过。源文件副本、完整输出与重跑日志保存在工作树的 `.superpowers/sdd/P4-issue-collaboration/`，分别为 `p4_sanitizer_probe_test.go.txt`、`sanitizer-placement-probe.log`、`archtest-after-probe.log`。这是依赖图证据，不代表清洗实现或安全验证已完成。

**推荐 A。** 调整 M4 对清洗的层次安排，保持现有架构规则：

1. domain 只保存清洗结果、内容标识等纯值及规则。
2. app 声明由用例消费的 `HTMLSanitizer` 小接口。
3. `issue/adapter/content` 集中实现清洗、解析和内容投影，承接 bluemonday 与 `x/net/html`，放行表仍只有一份。
4. `issue/module.go` 构造实现并注入用例；bootstrap 继续负责模块组合。描述和评论的产品行为不变，S3 和安全验证仍是第一项实现验收。
5. 同步 M4 设计中上述位置及 spec/plan，再进入预检；不修改或豁免现有架构守卫。

这沿用现有 identity 的做法：`app.PasswordHasher` 声明接口，`adapter/argon2` 引用第三方库，`module.go` 组装。代价是增加一个窄接口和适配器，并修订 M4 设计的层次说明。

**备选 B。** 保留 domain 清洗实现，为第三方依赖放宽纯层规则。它需要修改基础架构约束及直接/传递守卫，会影响以后对纯层的判断，**不推荐**。

**裁定状态：未裁定。** 按[负责人纪律](../../claude_to_gpt/03-rules-and-pitfalls.md) §1，先提交已完成准备工作，再上报并停下。推荐 A 不视为已批准，也不提前实施。

### 已形成的局部计划调整

- 五张新表按首个可用接口的任务加入，同步删除连带和 `furnishIssueRows`，避免用临时豁免绕过目录驱动删除断言。
- 为满足 P4 读目录与查询在同一快照内，计划把设计已批准的 `ReadOnlySnapshot` 从 P5 前移；它必须与现有 `LockTimeout` 的事务开始路径兼容，交给 P5 去重。
- 提及资格须按被提及成员自身的可见性判定，不能沿用发起人的 `OwnOnly`；草案拟在既有 `ProjectMembers` 目录职责内细化，尚未实现或独立预检。不得改变 P2 对 `ProjectGate.Lock` 返回值的约定，发现额外跨模块协议变化时仍按负责人纪律处理。

上述安排只是保留下来的草案，不构成通过预检的结论。

## 4. 整分支评审的发现与处理

尚未送 Claude 终审。A1 是实施前发现，待负责人先裁定。后续完成 P4 后，按用户要求交付分支、基线、spec、plan 和本记录，等待负责人安排 Claude 对整分支终审；处理发现后才能合并。

## 5. 最终实现与 plan 的差异

尚无产品实现，不能填写最终差异。当前已提交的启动变化只有 `tools/keywords.json` 的 phase 和工作树忽略规则（`05841220`）。spec/plan 均明确是未执行草案；获裁定后应统一修订，避免并存两套清洗接口或放置方案。

## 6. 移交事项

- **恢复 P4：** 读取 A1 的负责人回复，记录裁定；统一修订设计/spec/plan，完成独立预检后从 Task 1 开始。不能把本文的基线检查算成 P4 的验收结果。
- **P5：** 若采用前移方案，复用 P4 落地的 `ReadOnlySnapshot`，不要重复实现；其验证结果在实际完成后补入正式交接。
- **Claude：** 本分支仍处于准备阶段，尚未到阶段终审节点。
- M4 的各份交接维持当前状态，不因计划覆盖而提前关闭；M5–M8 的 M4 收尾交接尚未产生。

## 7. 已知限制

- P4 尚未实现，M4 远未达到收尾完成条件；没有推送、PR 或合并，没有新 main CI 结论。
- W11 既有偶发失败按 [06](../../claude_to_gpt/06-ci-w11-flake.md) 的裁定不调查；未来某次合并被它阻塞且重跑仍失败时再上报。
- 草案没有经过独立预检；后续任务的接口、拆分和文件名仍须在 A1 裁定后核定。
