# P5 main E2E 作业预算：一次重跑后的停点

> **负责人已批准 A（2026-11-04），实施与验证中。** 基线 main 为 `ce29aa00076130e2e805f50e2de08fc5d3ca2375`，本修复分支 `codex/m5-p5-ci-budget`。P5 功能与清理恢复夹具修复已合入；原main CI未全绿，分支/worktree和证据保留。P6 的 PC1 方案 A 已同步设计、spec、plan，产品尚未实施。

## 1. 为什么停下来

负责人此前明确要求：

> 如果它让某次合并的持续集成失败、重跑也不过，就停下来告诉我，我们一起解决。

本次 main 首轮发生已知 W11，主控已自行重跑一次。第二轮 W11 和全部用例均通过，但整个 E2E 作业因20分钟上限被取消，所以准确 main CI 仍未通过。此处是遵守上述特定停点，不把普通 CI 预算调整称为新的架构问题，也不据连续推进授权发起第三次重跑或自行调整上限。

## 2. 已核实的事实

| 运行 | 结果 | 证据 |
| --- | --- | --- |
| 修复分支 `defc8914`，CI 41740835449 attempt1 | server/web/e2e全绿；364/364，16.4m，无重试 | [分支运行](https://github.com/open-nerve/NerveProject/actions/runs/41740835449) |
| main `ce29aa00`，CI 41745515892 attempt1 | server/web成功；W11第190行弹窗数量expected1/received0，363 passed、1 failed（17.9m）；E2E终态cancelled | [首轮E2E](https://github.com/open-nerve/NerveProject/actions/runs/41745515892/job/126303655606) |
| 同一main与run，attempt2 | W11通过，364 passed（17.4m）、0 failed；E2E终态cancelled | [重跑E2E](https://github.com/open-nerve/NerveProject/actions/runs/41745515892/job/126312370586) |

第二轮的准确时间线（UTC）：

- 00:51:55：E2E job开始。
- 00:54:33：安装、浏览器及构建完成，进入E2E步骤，准备耗时2分38秒。
- 01:08:14：W11页面通过。
- 01:11:59.662：日志输出364 passed（17.4m），距job开始20分4.662秒。
- 01:12:00.488：日志记录作业被取消。

GitHub作业页的公开注解明确写明 **“The job has exceeded the maximum execution time of 20m0s”**，已保存原HTML。`.github/workflows/ci.yml` 的 `jobs.e2e.timeout-minutes` 当前确为20。由此可确认第二轮的取消原因是整个作业预算不足，不是W11再次失败或新用例断言失败。W11根因仍未修复，重跑通过不能替代其根因证明。

GitHub在第二轮为已成功的server/web分配了新job身份126312433250/126312371600，但保留原执行时间；再次下载的日志与首轮逐字节一致，未重新执行这两项。

两轮完整日志、run/jobs原件、第二轮公开注解HTML和两份原始Playwright ZIP均保留。首轮ZIP 21,919,197字节/701条目，第二轮2,750,153字节/670条目；公布的SHA和全部ZIP CRC均核验通过。P6预检目录复制16份相关原件、25,281,870字节，并逐hash/mode核对；源修复工作树原件保持不动。

## 3. 建议 A：仅将 E2E 作业总预算改为25分钟

拟修改范围明确限定为 `.github/workflows/ci.yml` 的 `jobs.e2e`：

```diff
   e2e:
     ...
-    timeout-minutes: 20
+    # 完整故事约17–18分钟，另留安装、浏览器、构建与收尾时间。
+    timeout-minutes: 25
```

当前364项故事本身约17–18分钟，整个job还包含安装、浏览器、构建和清理。20分钟的总预算已经被一次全通过运行实际超过，不能把全部20分钟都留给测试。25分钟为已测20分5秒总耗时保留约5分钟波动余量，并保持有限上限。

本方案不改任何 Playwright `timeout`、`expect`期限、重试次数、测试断言、workers、生产代码、服务端预算或权限；不屏蔽W11、不允许失败用例合并、不以此关闭任何尚未满足的验收。收益是避免全通过后被作业上限取消；代价是异常作业最坏可多占用5分钟runner。将来用例规模继续增长时应依据新实测另行评估，不能无证据持续加大预算。

方案 B 为保留20分钟，先优化或拆分E2E。它可能缩短反馈，但会扩展到套件调度、夹具生命周期和实物隔离验证，当前没有已证明的局部性能缺陷可直接修。仅再重跑同一20分钟配置不能修复已证实的预算与总耗时不匹配。综合范围、收益和回归风险，推荐 A。

## 4. 获准后的实施与验收

1. 从准确 main `ce29aa00` 建新的独立修复worktree，主目录继续保持main；同步本裁定，仅修改上述job预算及必要的中文执行记录。
2. 核完整差异及工作流结构，确认生产、测试断言与所有单项期限逐值不变；复用未改变输入的本地验证证据，不为单一作业预算重复执行全部本地大套件。
3. 一次fresh整分支deep-review，系统Git推送，核准确新分支server/web/e2e及完整日志；失败不自动放宽其他标准。
4. 分支门禁通过后按连续授权`--no-ff`合入main，准确新main CI三项全绿，才完成P5证据归档、正常分支/worktree清理。
5. P6同步最终main并回填预检与P5 A9，之后从T1实施已批准的资料图片方案。任何提前取消/重跑记录继续保留，P6与M5总体完成项不提前勾选。

## 5. 本机证据入口

- 修复原件：`/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m5-p5-ci-purge/.superpowers/sdd/P5-ci-purge-resume/`，重点为 `main-ci-attempt-1-failure.json`、`main-ci-attempt-2-failure.json`、`main-ci-rerun-request.json`。
- P6预检副本：`.superpowers/sdd/P6-profile-upload/precheck/p5-ci-budget/source-copy-manifest.json`，列出全部文件、字节、SHA及mode。原件尚未归档到稳定目录，因为main验收未通过。
- [清理恢复修复与执行收据](P5-ci-purge-resume.md)、[该修复独立审查](../../../reviews/re-43-M5P5集成修复审核.md)。

## 6. 批准后的执行记录

负责人回复“批准，go”，授权实施上述 A 方案并完成 P5 收尾后继续 P6。此前等待决定的表述为裁定前快照。

- [x] T1：从准确main新建隔离worktree，只将E2E作业总上限20改为25分钟，加实测依据注释。
- [x] T2：核工作流结构、全量差异、未变输入和证据；一次fresh整分支deep-review及主控复核，0 Critical/0 Important/0 Minor，见[re-44](../../../reviews/re-44-M5P5作业预算审核.md)。
- [ ] T3：准确最终分支CI三项全绿，再按连续授权noff合入main，确认准确main CI三项全绿。
- [ ] T4：P5及两次集成修复全部原件归档核验后正常清理；P6同步最终main，回填验收并从T1继续。

局部裁定：这是单一工作流预算调整，复用逐SHA不变的产品、测试和依赖验证，完整远端分支/main CI仍各跑一次；不新增复制配置常量的测试，不重跑未受影响的本地大套件。原有20分钟失败、W11失败及重跑通过原件全部保留，不能把扩大预算写成W11根因修复。实现和验证台账保存在 `.superpowers/sdd/P5-e2e-budget/`，后续准确身份与归档收据回填P6阶段记录。
