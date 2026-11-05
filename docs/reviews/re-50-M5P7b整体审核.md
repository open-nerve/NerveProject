# Deep Review：M5/P7b 工作项附件与编辑器图片

审查日期：2026-11-06。正式main基线`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`，冻结候选`0716b12f77468159726bd6c8d6e6cf154809de24`（T14检查点，分支CI [42079747995](https://github.com/open-nerve/NerveProject/actions/runs/42079747995) server/web/e2e首次即全绿）。按executing-plans执行一次fresh whole-branch审查：审者为只读子代理（显式指定最强档`fable`），主控（Claude Opus 5.5）处理结论与验证。没有逐Task派审者，也没有重复同范围复审。

## 1. 范围、证据与限制

审查范围为25个提交、298个文件（15603增、4076删）。审者先读review package的提交清单与统计，再按区域以`git diff`和整文件阅读手写逻辑：服务与字节下载、上传状态机与策略、分页/删除/读取资源、附件控件、详情/peek宿主与保留输入、评论、编辑器图片端口与应用适配、草稿文件与副本、`attachment_count`端到端、旧文件链删除、browse路由与认证中间件修复。生成物（`schema.gen.ts`、Go生成代码、`api/dist`）逐一对源核查；`server/migrations/sql/00038`的Up/Down与sqlc登记已核。台账中全部`Ruling:`逐条读过并给出评价。

审者只读：没有编辑、安装依赖、启动容器，也没有运行Vitest或E2E；每条发现来自读代码，无法以运行确认之处在原文中注明。完整审者原文`t15-review-raw.md`保存在本阶段ignored证据目录，随阶段归档。

## 2. 三视角与五项重点

| 视角 | 判断 |
|---|---|
| 逻辑 | 命令结果与只读核验处处分离：`UploadSession`的recovery只重发complete、仅`asset.object_missing`才重传；`ContentDeletion`同ID一个在途、非Problem/5xx记为未知、再次DELETE前先只读核验；确认后的读取失败成为“重新读取”而非命令失败。ContentView单调结束，A→B→A总是新view。内容权限来自M4角色与所有者，未复用P6项目封面策略；`canRemoveAttachment`不升格为可新增。保留输入随宿主scope创建与退休、在pending中存活，恢复后暂停并在新view重新核验图片。发现F1。 |
| 集成 | 生成物与源一致；`attachment_count`经OpenAPI、生成码、领域默认值、HTTP映射、两套词表、迁移00038与前端键列表端到端一致，排序只走服务端。四处跨M修复（`86c9f9f6`认证中间件重发、`370aba81`/`0d093f03` browse路由、`5ff2e4a2`宿主弹窗归属、`8d85db5a`表格插件计时器）均为页面、宿主或单文件层的根因修复，不改DetailResource、IssueVerification（仅新增只读`isMissing`）、PermissionReads或权限观察器；评论命令链新增的同步`onConfirmed`为既有唯一命令链上的追加。删除面完整，无兼容层残留。 |
| 测试 | 各Task的聚焦套件均有RED→GREEN收据与被杀死的变异；F8–F12、F16三浏览器`[local]`与Chromium `[s3]`页面故事带请求与旧流量监视、15个页面变异；页面发现的产品缺陷都在产品根因处修复。缺口（不阻塞合并）：离线到期续期（F1，已补）、节点视图在传输中重建（F2）、带文字的被拒图片批（F4）、删除核验404分支（F5）、恢复输入保存时的新上传（F6）、管理员删除他人评论文件的真实页面。 |

五项Review Focus：①丢失complete/DELETE回答、确认后读取失败、撤回期间保留输入——PASS（F5为唯一粗糙边）；②可见性、评论作者、无项目私有草稿、项目移动、A→B→A——PASS；③多节点同源、后台到期/重连、晚到onError、复制中Undo再Redo——PASS附Important F1（F2为Minor）；④新评论准备件/未知POST、草稿未保存/复制/发布失败——PASS（F3为Minor）；⑤纯非图片/混合文件、伪造剪贴板/外部src、旧复制监听、未来领域wrapper——PASS附Minor F4。

审者对局部裁定的评价：T9“含图片标识的输入在新view显式保存前保持暂停”、T11负责人A决定的端到端执行、T12复制为Markdown省略图片、T13“核对最新状态”不触发规范重读、真实撤销401、变异3/4在所改层杀死的如实记账、T14弹窗随正文、两个M4故事断言收窄而非删除，均认为正确；T10访客复制含图工作项走直连由服务端拒绝属可接受，提示可更友好但不是缺陷。

## 3. Important：F1 到期续期未送达后，图片停在失败直到逐张重试

位置：`web/apps/web/core/store/asset/content-read-entry.ts`（`wake()`只处理ready快照）、`content-reads.ts`（focus/online/visibilitychange唤醒全部条目）。

场景：带图片的详情或peek保持打开，设备休眠或断网超过服务端读取期限（65分钟）。唤醒时到期计时器往往先于网络恢复触发，那次唯一的续期请求未送达（fetch以TypeError失败），条目变为unavailable；随后网络恢复带来的online、focus、visibilitychange唤醒因条目不是ready而什么都不做。页面每张图停在“无法加载/重试”，只能逐张点重试或刷新。规格（内容资源子合同§87）把这些唤醒列为到期与重连时的续期入口，这次失败使其失效。审者评Important，主控按效果复核维持：这是“离开一阵再回来”的日常路径，结果看起来是页面坏了。

根因与修复：续期失败后条目没有区分“服务端已回答”与“请求根本未送达”。`ContentReadEntry`现记住“上次到期续期未送达”（只认浏览器fetch在网络失败时抛出的TypeError），之后每个唤醒事件再试一次，成功或得到服务端回答即结束；服务端的回答（如503）仍保持到用户显式重试；不另起计时器、不形成循环；图片加载错误的“一次自动重新解析”额度与显式重试不变。规格“到期仅触发一次重新授权”解释为一次真正到达服务端的重新授权；代价是设备持续离线时，每个唤醒事件每个资产多一次立即失败的请求。

验证：新增“未送达的续期由之后的唤醒再试，直到续期成功”，先红（focus唤醒后0个新请求）后绿（`t15-f1-red`→`t15-f1-green`）。变异“任何失败都可唤醒再试”起初存活：既有“到期被拒后停止自动续期”用例在派发focus后同步断言，而续期请求要到下一个任务才发出；该用例改为派发focus与online后推进时间再断言，变异即被杀死（`t15-f1-mut-any-failure-2`），实码14/14（`t15-f1-green-2`）。提交`6baaf7bf`。

## 4. 死成员：F7b

`LocalInput.suspended`与`IssueDetailInputState.suspended`只有自身测试读取。knip看不到类成员，T14 Step 1的死成员扫描因此漏掉这一类；主控按T14合同的未完成项处理而非润色，删除两个getter与测试中的两句断言，并对本阶段新增/修改文件的类成员补做读者扫描，除正则误报外再无无读者成员。issues与store/asset 82文件895项、web类型检查、改动文件oxlint零警告通过（`t15-f7-dead-getters`、`t15-types`）。提交`6058fe2b`。

## 5. Minor（交P8，未在本阶段修改）

- F2：上传或复制进行中拖动图片块（或Undo→Redo）会重建节点视图，清理取消传输并丢掉File，重挂载显示空“选择图片”且无提示（审者按ProseMirror视图匹配推断，未实跑）。
- F3：未知发布经核验已成为工作项后，关闭弹窗仍提示“草稿已保留”并链到草稿列表。
- F4：编辑器粘贴处理对被拒的图片批或非图片文件批返回true却不preventDefault，附带的文字或HTML走浏览器原生粘贴；修复前需先对齐规格“混合批保持事件未处理”的措辞。
- F5：删除结果未知、核验得404后，对话框仍可“删除文件”（只会再404），取消后“删除未能确认”的提示无处清除，直到控件重建。
- F6：“保存保留的输入”切换editable使编辑器重建，会取消恢复后新开始的图片上传。
- F7a：`ConfirmedContentNotice`在宿主生命期内一直显示最后一次收据。

## 6. Declined to judge 与主控裁定

| 审者暂不判断 | 主控裁定 |
|---|---|
| 新评论view在POST中途结束时，表单停在“未知”并保留文字，评论其实已发表 | 维持：规格“旧scope回调无效”；用户被提示先核对评论列表，不会重复发表；记P8 |
| 时钟偏差接近或超过65分钟TTL | 维持：P6既有能力校验规则、属环境；F1只对未送达的TypeError再试，`capabilityFailure`不在其列，不新增循环 |
| 非图片文件拖入时ProseMirror默认插入`text/uri-list`文字 | 维持：M4既有，规格要求非图片拖放不处理 |
| 每张评论卡一个ContentView与EditorAssets | 维持：每卡一条reaction与一次登记；P8容量演练实测 |
| `t13-content-local-4`两项间歇 | 维持T13记录：隔离重复与全量重跑均通过，T14三轮默认E2E与分支CI未现；P8三浏览器最终矩阵再观察 |
| M4规范getter在写后重读期间返回undefined | 维持：M4设计，规格§4.1据此设计宿主输入状态 |
| 下载`a.click()`后立即`revokeObjectURL` | 维持：F8下载SHA故事三浏览器实证 |
| `attachment_ids: []`与省略 | 维持T6合同：API规定二者等价 |

## 7. 合并判断

审者原结论With fixes；没有Critical、Structural或需要负责人决定的项。主控按效果复核后：F1维持Important并以有区分度的RED/GREEN与被杀死的变异闭环；F7b按T14合同未完成项删除；F2–F6、F7a为Minor，列入P8。修复后的完整七门禁与准确候选分支CI见下；本报告不替代它们或main CI。

主控最终本地复验（修复后HEAD `6058fe2b`，七项输入清单同一，digest `93f855b4…`前后不变）：gen-check 3秒；lint-go 2秒；`GOFLAGS=-p=1 make test`首跑只在`project/adapter/postgres`一包失败——testcontainers-go为该包启动会话收割器时收割器容器在就绪检查中消失，未执行任何业务断言（与M4/P7记录的Ryuk INFRA失败同类；Docker VM内存与磁盘无压力，无遗留测试容器）；按先例不改设置与期限，以同一输入完整重跑通过（`t15-test-2`，1000秒，61个测试包、21个无测试包，bootstrap 654秒）；lint-web 50任务/33缓存（web实际执行）；knip通过；test-web 16任务/15缓存（web实际执行337文件3816项）；e2e 415/415（212秒）。其他项目的15个容器前后一致，测试容器全部由收割器清理。准确候选分支CI与main集成为T15 Step 3及T16的前置，结果见[阶段记录§10](../v0/M5-files/reviews/P7b-content-assets-review.md#10-t15-整分支审查与修复)，本报告形成时尚未合并。
