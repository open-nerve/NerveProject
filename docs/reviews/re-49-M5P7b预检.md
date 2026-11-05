# M5/P7b 工作项附件与编辑器图片预检

> 2026-11-05：预检通过，暂不实施。分支`codex/m5-p7b`，正式main基线`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`；P7a准确main CI 41869045846、归档及清理前置已核。

## 1. 范围与前置

本阶段接通工作项/评论/已有草稿的附件与正文图片、准备件恢复、含图复制、真实计数/排序/显示偏好，删除最后旧文件调用链及services包。[主spec](../v0/M5-files/specs/P7b-content-assets.md)、[资源子合同](../v0/M5-files/specs/P7b-content-lifecycle.md)、[编辑器子合同](../v0/M5-files/specs/P7b-editor-images.md)、[plan](../v0/M5-files/plans/P7b-content-assets.md)共同规定范围，不重建后端上传、绑定或发布协议。

P7a最终候选`756b90314009f00e487483729b5d19f76dbeb110`的[分支CI](https://github.com/open-nerve/NerveProject/actions/runs/41859931318)与noff main的[CI](https://github.com/open-nerve/NerveProject/actions/runs/41869045846)分别全绿。正式main的完整Git tree等于候选，4544个非文档输入与P7a最终门禁逐项相同；因此独立builder核查过的真实API/owner代码没有变化。归档及清理见[P7a阶段记录§13](../v0/M5-files/reviews/P7a-draft-preparations-review.md#13-t7-集成归档与接续)。

原P7预检的恢复缺口已由负责人批准A、拆P7a/P7b；P7a现在提供`listDraftAssetPreparations`与生成`DraftAssetPreparationPage`，本阶段按真实协议接收。原[re-46](re-46-M5P7预检.md)保留发现时结论，不改写历史。

## 2. 真实生产者与消费者核对

| 边界 | 已核事实与落点 |
|---|---|
| T1服务/下载 | 直接消费生成Asset、Content、AttachmentPage、DraftAssetPreparationPage；真实GET/DELETE/duplicate路径、limit/cursor/signal和204。字节下载使用无业务凭据的独立fetch、有限大小/时间/取消。 |
| T2上传/授权 | 现有UploadSession单一算法可提取消费方UploadPolicy；P6四用途保持，六种内容用途由当前host工厂映射，不新增new_issue。沿IssueRootStore唯一PermissionReads/registerResource。 |
| T3–T6资源/页面 | 已有附件分页、complete原子绑定、评论attachment_ids与计数刷新入口均存在；确认命令和读取失败分离，真实详情/peek和评论表单消费。输入跨卸载的局部缺口按下一节补实。 |
| T7–T9编辑器 | 应用声明实现当前宿主适配，包消费可选images小接口；包内extensions/hooks/raw-image和全部wrapper同Task迁移，旧TFileHandler不留空桩。业务HTML只存UUID，历史、自动保存、迁移语义保持。 |
| T10草稿与副本 | 真实Draft/null项目已有API支持；普通副本最终Issue，草稿副本最终Draft；含图明确提交后建一次目标，duplicate/save/publish沿原协议，未知结果先只读核验。 |
| T11计数 | 生成attachment_count及服务端正反排序已实现，不新增字段、不以当前页长度猜总数或逐行请求。 |
| T12删除旧链 | 最新消费者清单保留；URL规范化及原测试搬入web后，可删除services/file-type独有依赖，M6–M8不提供图片能力且不新增领域请求。 |
| T13证据 | 实际目录为stories/files、fixtures/file-story.ts，local/S3接收端和浏览器设施可复用；页面、三浏览器和有效变异必须新运行，P7a API证据不替代它们。 |
| T14–T16完成门 | 完整七门禁、准确分支CI、一次整审修复、noff/准确mainCI、全量归档和P8仅预检分别闭环。 |

私有草稿继续服从WorkspaceDraftIssues的工作区级集合寿命：asset可使用已有currentWorkspace，但不能在规范Draft撤回后私存DTO继续授权。项目成员资格不是私有草稿的新增条件；同工作区项目变化可能保守地要求重读，这是现有owner行为。issue/comment按明确项目范围保持独立。

## 3. 一次独立 builder 与局部裁定

审者`/root/m5_p7b_builder_check`只读准备合同与冻结候选代码，没有编辑、运行产品测试、启动数据库或安装依赖。其初始结论是**1项局部编排问题，不能直接宣称nothing important left**；未发现必须新增owner API或改变权限水位的必要性。本次没有再派同范围builder；以下为主控按实际源码裁定并修订合同后的结论。

问题：普通附件complete/DELETE后调用refreshAuthoritative，会令IssueVerification进入pending；规范Issue getter返回undefined，详情和peek卸载正文。现有useConfirmedAutosave的dirty/失败revision、评论HTML及准备件数组属于子组件本地，IssueDetailHost目前只管modals。省略afterWrite也不能避免显式读取pending。未发送评论、失败正文和图片恢复状态可能因此丢失。

裁定：在现有IssueDetailHost原scope中增加小型IssueDetailInputState，保存本地标题/正文/评论、编辑基准、已确认资产身份/命令结果；旧ContentView、队列、读句柄、URL和请求仍永久结束，新owner及资产重验后才允许编辑。冲突不自动覆盖，拒绝revision及未知POST/duplicate/DELETE不因重挂载自动重发。真实权限ABA/路由/root/关闭peek或owner确认删除/不可见（宿主读取401/403/404）清空；暂时读取失败只允许重读，正文写422仍保留输入，fresh owner只读时不能把未提交内容显示成规范事实。

根因是本地输入的寿命短于暂时卸载的子树。该调整只补当前页面编排，不修改IssueVerification的uncertain、PermissionReads、DetailResource或Reconciled通用语义；若实施必须改变它们，仍须先提交后上报。代价是增加小型受控输入状态及真实RootStore/详情/peek联合回归，传输中断可能需要用户明确恢复或重新选择文件。

落点已补入资源spec§4.1、主spec§3第11项/A2、editor§4、plan T3/T5/T6/T9/T13；标题实际也使用useConfirmedAutosave，已纳入。R13/R14详细转录及源码行号保存在P7a归档的`p7b-builder-check.md`。其余局部决定及收益/代价见[主spec§3](../v0/M5-files/specs/P7b-content-assets.md#3-局部裁定及与设计的补充)，没有架构待决项。

## 4. 计划覆盖与预检验证

16Task、48个开放实施Step、8个开放产品验收；A1→T1–T5/T11/T13，A2→T6/T9/T13，A3→T10/T13，A4→T7–T9/T13，A5贯穿T2–T10/T13，A6→T12/T14，A7→T14/T15，A8→T16。五项Review Focus各有拒绝/交错/未知结果反例，新增输入连续性必须通过联合链及有效变异证明。

`p7b-preflight-check.py`核准确main/祖先/候选树、P7a最终CI/归档清理、4544产品输入、文档本地链接、16Task/48Step/8A、无P7b实施brief与无依赖安装，并运行git diff --check；receipt保存于`.superpowers/sdd/P7b-content-assets/`。正式结论依据实际收据，不以静态检查冒称产品测试通过。

实际检查117个本地文档链接均有效，产品改动0、依赖安装0。首次检查器按行读取Git的转义文件名，将中文报告路径误判为非docs；改用NUL分隔的原始路径后检查通过，失败与成功输出均保留。这属于预检工具修正，不计作P7b产品RED/GREEN。

本轮无P7b原型、产品修改、依赖安装或新增页面运行。采用实际入口/数据/授权/消费者静态核查及一次独立builder替代预先完整实现原型；收益是避免两次实现，代价是运行时联动必须由T1–T13逐项RED/GREEN和真实页面验收证明，预检不能免除这些责任。

## 5. 接续边界

P7b预检通过，停在T1前；8项产品验收全部保持未勾选。P8/closeout继续承担最终F1–F18矩阵、默认部署/容量、P3-M1/P4-M1、全仓warning规则、具名copyMarkdownToClipboard风险与handoffs最终关闭。M5总体尚未完成。
