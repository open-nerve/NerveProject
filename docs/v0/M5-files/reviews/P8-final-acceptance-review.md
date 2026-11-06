# M5/P8 阶段评审记录

> 2026-11-06：正式预检通过，暂不实施。分支`codex/m5-p8`，基线main`2b2c6f33e3a93536742d5f28b74eb959ba73d6d1`（P7b noff）。P7b准确分支CI 42109712732、main CI 42115811156及归档/正常清理已核，见[P7b阶段记录§11](P7b-content-assets-review.md#11-t16-集成归档与接续)。

## 1. 评审方式与范围

[主spec](../specs/P8-final-acceptance.md)与[plan](../plans/P8-final-acceptance.md)界定16个Task。一次fresh builder对照main实际代码逐Task核查，主控按既有事实、局部裁定、需负责人决定分类闭环；正式报告见[re-51](../../../reviews/re-51-M5P8预检.md)。

这是实施前预检；P8整分支deep-review、门禁与集成尚未开始，不冒充同一轮完成。

## 2. 验收状态

A1–A8全部开放。P1–P7b的阶段证据只作前置与目录来源；最终F1–F18矩阵、演练、演示和三浏览器结果必须在本阶段最终产品候选上新运行。

## 3. 局部裁定

主spec§3的11条为绑定合同：粘贴三态、传输随节点而非节点视图、核验不重建编辑器、删除核验404出口与收据寿命、副本与链接表单提示、删除无读者代码与素材、两条告警规则清零、演练只组合既有夹具、演示照README的prod步骤、S3页面变体沿P6/P7b在Chromium、夹具用例落点。

预检补实的取舍与代价：

- Undo撤掉在途图片的插入时取消且不写回，Redo恢复的节点显示可见的已取消状态、不自动重发；代价是用户需重新选择文件，换来不重发未知结果。
- 粘贴的被拒图片批改为拦下并整批不插入（P7b子合同原写“不吞”），普通文件批交回ProseMirror，拖放不变；代价是混合粘贴中的文字需另行粘贴。
- 链接表单的成功提示以宿主scope判定，子项/关联不新增提示；M4代码的本地修复单独提交、可回退。
- 演示用prod配置并给e2e夹具加不注入测试覆盖的显式选项；S3页面变体不新增三浏览器config，浏览器差异由F4三浏览器覆盖。

没有架构待决项，也没有需要负责人决定的问题。

## 4. 预检发现与处理

一次fresh builder对照main实际代码逐Task核查，初始结论“需先解决6项”，另1项提醒，全部为既有事实或局部裁定：Undo→Redo与取消规则矛盾（P8-01）、子项/关联本无成功提示（P8-02）、默认配置关闭注册与夹具注入test覆盖（P8-03）、现有config无三浏览器S3页面变体（P8-04）、原用例断言相反合同（P8-05）、夹具用例落点未指明（P8-06）、链接检查不是仓库工具（P8-07）。逐项裁定与代价见[re-51§3](../../../reviews/re-51-M5P8预检.md#3-一次独立-builder-与局部裁定)，均已写入spec§3与plan步骤。审者没有看修订后的文档，因此不冒称独立复审通过；主控合同自检与预检收据闭环。

## 5. 限制与证据

预检只读源码与文档，没有安装依赖、运行产品测试、启动数据库或修改产品；页面、演练与演示的运行证据在实施中产生。原始builder报告与预检输入清单保存在本阶段ignored目录`.superpowers/sdd/P8-final-acceptance/`，随阶段归档。输入交接保持open，P8只写最终处理结果，closeout才关闭。

## 6. T1 证据与交接目录

### 6.1 F1–F18

交付阶段按[M5设计§12](../M5-design.md#12-分阶段推进)；Go测试按文件归类（`server/internal/bootstrap/`下除注明者），最终以T12的`GOFLAGS=-p=1 make test`全量运行为准；故事均在`e2e/stories/files/`，其中用`fileStory`/`pageFileStory`注册的同时有`[local]`与`[s3]`变体（F4–F12、F16的大部分故事）。阶段记录中“F1–F7”也用作P7b整审发现编号，本目录只按用户故事编号归类。

| F | 交付阶段 | 现有Go测试 | 现有故事 | P8最终落点 |
|---|---|---|---|---|
| F1 | P2 API、P3 S3、P6/P7b页面 | asset_lifecycle、asset_complete、asset_transfer、asset_contract | f1-upload（local；S3由F4与S3适配器合同覆盖） | T12：Go全量、默认e2e |
| F2 | P1 local、P3 S3 | asset_transfer_slow（`TestRawStorageDefaultSlowUpload`）、asset_transfer、asset_mac；`modules/asset/adapter/http/{transfer,budget}`、`adapter/local/sign`、`adapter/s3/fault` | 无（设计即为Go慢流） | T10：local与S3慢流演练、页面慢流（Chromium限速）；T12重跑原测试 |
| F3 | P1、P2 | asset_complete_races、asset_complete_failure、asset_complete_io；`adapter/local/seal`、`adapter/s3/object_real` | f3-publication | T12 |
| F4 | P3 | `adapter/s3/*`（presign_integration、object_real、endpoint_origin、inventory、capacity、fault）、asset_s3_lifecycle、asset_s3_recovery | f4-s3-storage | T12：`m5-s3`三浏览器 |
| F5 | P4 API、P6页面 | asset_image_account_first、asset_image_auth、asset_image_credentials、asset_image_references、asset_image_storage、asset_image_io | f5-profile-images、f5-profile-images-page | T12：默认e2e、`m5-profile`三浏览器 |
| F6 | P4、P6 | asset_image_catalog、asset_image_member_races、asset_image_invitation_races、invitation_image_locks | f6-workspace-images、f6-workspace-logo-page | T12 |
| F7 | P4、P6 | asset_image_creation_races、asset_image_parent_races | f7-project-covers、f7-project-cover-page | T12 |
| F8 | P5 API、P7b页面 | attachment_access、attachment_lists、attachment_publication、attachment_events、attachment_rollback、attachment_transaction、attachment_read_effects、attachment_read_snapshot、asset_attachment_capacity | f8-issue-attachments、f8-issue-attachments-page | T12：默认e2e、`m5-content`三浏览器；T5回归 |
| F9 | P5、P7b | asset_content、asset_content_read、asset_content_races、attachment_subscriber | f9-comment-assets、f9-comment-files-page | T12 |
| F10 | P5、P7a、P7b | draft_assets、draft_assets_atomic、draft_assets_races、asset_draft_preparations（含concurrency/recovery/snapshot） | f10-draft-assets、f10-draft-files-page、f10-draft-preparations | T12；T6①回归 |
| F11 | P5、P7b | asset_duplicate、asset_duplicate_races、asset_copy_content | f11-content-copy、f11-editor-images-page | T12；T2/T3回归 |
| F12 | P5、P7b | asset_content_storage；`adapter/content/validate`、`adapter/postgres/{content_bindings,image_bindings,content_sets}` | f12-image-identity、f12-image-history-page | T12 |
| F13 | P2、P5 | asset_access_ports、asset_host_matrix、asset_content_permissions、attachment_access、assets_credentials_interleaving、assets_membership_interleaving、assets_late_host_interleaving | f13-permissions | T12 |
| F14 | P2、P5 | asset_deletion、asset_deletion_rollback、asset_content_deletion、assets_scope_rollback、asset_parent_ports、asset_parent_reader | f14-deletion | T10演练；T12 |
| F15 | P2、P4、P5 | assets_cleanup、asset_purge、asset_purge_wiring、asset_retention、asset_content_retention、asset_image_retention、asset_content_purge_process、asset_image_purge_process；`adapter/postgres/cleanup_*` | f15-cleanup | T10演练；T12 |
| F16 | P6、P7b | web单元（`core/store/asset`、`core/components/issues`宿主与详情） | f16-content-lifetime-page、f16-profile-upload-lifetime | T12：`m5-content`、`m5-profile`三浏览器；T3/T4回归 |
| F17 | P6、P7b | web单元（附件对话框、上传控件） | f17-profile-upload-accessibility；P7b页面故事中的可访问性断言 | T12；T5回归 |
| F18 | P8 | 关键词守卫、knip | 无 | T11默认部署演示、T7删除面、T11扫描 |

缺口与补位：F2的S3慢速与页面慢速上传在[M0加固交接](../handoffs/M0-hardening-http-timeouts.md)中仍记为“后续阶段”，P3/P6/P7b没有补写结果，由T10补（S3演练中的慢PUT与Chromium限速的页面上传，后者大于1 MiB且超过15秒，独立config不进默认套件）；F18由T11新建；其余F在T12最终产品候选上重跑。

### 6.2 输入交接逐条

| 交接 | 条款 | 当前事实 | P8动作 |
|---|---|---|---|
| M0加固 | 逐请求放宽读写期限；不调大全局；慢体完成/不发体断开的测试；大小上限与413 | P1 raw路由已验证；P2业务JSON仍1 MiB/15秒 | T10补S3与页面慢流；T13写最终结果 |
| M1-P2 | 无`PAGE_DESCRIPTION`；无`page_id` | P2表与生成用途无此二者，P7b删`EFileAssetType` | T13写最终结果 |
| M1-P3 | 资源类型用生成类型；评论附件项目级；封面只内置/上传；编辑器附件结论；`file_size_limit`名称与策略 | 生成`AssetPurpose`；评论附件随项目；封面预设+上传；编辑器只处理图片、`ACCEPTED_ATTACHMENT_MIME_TYPES`已删；`files.size_limit`经实例配置下发 | T2细化粘贴语义；T13写最终结果 |
| M1收尾 | 死成员/死prop | 历史48条不作当前名单 | T7按附录A `--rows M5`逐行闭环 |
| | oxlint：改到的文件零警告、清一类规则、cap下调 | P7b改动文件零警告 | T8两条规则清零并设error；M5全期改动文件零警告 |
| | 附件图标第三方标志 | 17个PNG无引用 | T7整目录删除 |
| | 新建项目封面值 | P4/P6一次POST、稳定编号 | T13写最终结果 |
| | 复制资源权限 | P2/P5源不可读与不存在同为404 | T13写最终结果 |
| M2收尾 | §1头像封面（两列、签名地址、两处上传控件、关键词规则） | P4 API、P6页面 | T13写最终结果 |
| | §2上传与中间件（>1 MiB/>15秒、`file_size_limit`执行、CSP、整程序测试） | P1/P3；`use-file-size.ts`已删，读取点为`core/hooks/editor/editor-assets.ts`的`root.instance.config.file_size_limit` | T10慢流；T13写最终结果与实际读取点 |
| | §3删除`@nerve/services` | P7b完成 | T13写最终结果 |
| | §4 stores按会话分代 | P6/P7b：`FileService`单例清零、SWR键带loginId | T13引证 |
| | §5 `e2e/fixtures/storage.ts` | P2完成 | T13写最终结果 |
| M3收尾 | §1图标封面头像 | P4/P6 | T13写最终结果 |
| | §2 `organization_size` | P4响应收紧 | T13写最终结果 |
| | §3附件确认框/弹窗、焦点轮廓、key布局与后退、可访问错误 | P6资料弹窗、P7b附件对话框与页面故事 | T5/T6修复；T12最终三浏览器；T13写明 |
| M4收尾 | M5-01附件接口与页面 | P5/P7b | T12；T13写最终结果 |
| | 旧复制入口具名风险 | 无产品调用方 | T7删除方法，T13写“已删除” |
| | M5-02草稿同事务迁移与图片标识 | P5/P7b | T12；T13写最终结果 |
| | M5-03删除图、事件与物理清理 | P2/P5 | T10演练；T13写最终结果与事件交接 |
| | M5-04首个投递任务的CLI | 仍无投递任务的CLI，`NewInserter`只在服务组合 | T13原文转交M6/M7/M8 |
| | P7b T14注记“链接/子项/关联提示被跳过” | 只有链接表单有成功提示 | T6修链接；T13更正注记 |

每条都有落点，没有既未完成又无法转交的条款。

### 6.3 承接项

| 来源 | 项 | 落点 |
|---|---|---|
| re-50 §5 | F2拖动中传输被取消、F6保存保留输入重建编辑器 | T3、T4 |
| | F3未知发布后“草稿已保留” | T6① |
| | F4被拒文件批的原生粘贴 | T2 |
| | F5删除核验404死路、F7a收据常驻 | T5 |
| re-50 §6 | 新评论view在POST中途结束；时钟偏差；非图片拖放文字；`a.click()`后立即revoke；`attachment_ids: []` | 维持原裁定，T13在限制与交接中如实写明 |
| | 每张评论卡一个ContentView与EditorAssets的成本 | T12实测（含图评论较多的详情页，记录请求数与耗时） |
| | `t13-content-local-4`两项间歇 | T12隔离重复运行 |
| P7b记录§9–§11 | M6/M7/M8领域接线 | T13交接 |
| | P3-M1与S3夹具子进程吞错 | T9 |
| | P4-M1 `WorkspacePatch.Logo` | T7 |
| | 复制为Markdown省略图片（签名读取不外泄） | T13写为已知限制；旧复制方法T7删除 |
| | M4链接提示 | T6② |
| | testcontainers收割器竞态与Docker停顿 | 环境：同输入重跑并保留原件，不改预算 |
| | CI apt慢与E2E作业时长逼近25分钟 | 观察并记录；调整预算或拆分作业须负责人批准 |

## 7. 实施记录

T1记录提交为`96329131`：§6目录；M0加固交接中仍记为后续阶段的S3与页面慢速上传补入T10，re-50交P8实测的评论卡成本补入T12（两条Ruling见台账）。

T2（re-50 F4）：`drop.ts`的`handleDOMEvents.paste`在普通与被拒两种批次都返回已处理却不拦下，浏览器原生粘贴把附带的文字/HTML绕过ProseMirror写进文档。剪贴板文件改为普通/拒绝/接受三态：被拒的图片批整批不插入、提示一次并拦下；纯非图片文件交回ProseMirror的粘贴管线；拖放与无图片能力的路径不变（应用中新建表单与未打开收集箱的用例依赖后者）。原用例对被拒批断言`defaultPrevented=false`，随合同改写；P7b编辑器子合同§3与§5两处同改。RED：普通批文字未插入、5种被拒批未拦下（`t2-red-2`；`t2-red`是新worktree未构建依赖包的环境失败，不计）；GREEN 28/28（`t2-green`）；变异“被拒批不拦下”杀死5例、“普通批返回true”杀死1例，源码按SHA恢复（`t2-mut-rejected-unprevented`、`t2-mut-ordinary-true`）。编辑器包14文件137项、类型、改动文件oxlint与格式通过；重建后受影响的应用测试6文件63项通过（`t2-editor-all`、`t2-editor-types`、`t2-web-affected-2`）。关键词phase改为`M5/P8`，守卫76规则14例外无命中。

T3（re-50 F2）：传输原由节点视图持有，视图卸载即中止并`cancel(id)`、丢掉File，完成按旧`getPos`写回。拖动块越过视图匹配范围、包进或解除引用块都会重建视图，进行中的上传与复制因此静默丢失。新增`custom-image/transfers.ts`：传输记录（控制器、阶段、结果）按节点id放在扩展存储，视图只订阅；视图卸载后，当前任务结束时文档里已无该id且无src的节点才取消；完成按id写回所有等待中的同id节点；编辑器销毁与`images.signal`结束时全部取消。Undo撤掉在途插入即取消，Redo恢复的节点显示“传输已取消，请重新选择”（新增en/zh-CN文案）而不自动重发；未知复制失败在重挂载后仍显示失败与重试。

排查RED时发现拖动后节点id改变：UniqueID插件在paste事件置“去ID”标志、只在transformPasted复位，被图片处理器接管的粘贴留下它，此后编辑器内部拖动的块被换新id——即“粘贴图片后拖动”这条最常见路径在任何按id归属的设计下都会被取消。`bdc6909d`单独修复：drop按本次拖放重新判定标志并清拖动来源（Plane带来的编辑器插件的本地修复，与`8d85db5a`同类，可单独回退）；RED为段落内部拖动后换新id（`t3-uid-red`），GREEN 17/17（`t3-uid-green`），变异“只置位不复位”被杀死（`t3-uid-mut-stale-flag`）。

传输RED（`t3-red-2`）：复制拖动与包裹块的传输被取消、Redo后无提示；上传拖动失败于上述id问题。GREEN 35/35（`t3-green-2`，含新增“确认只写回自己的节点”）。变异：“视图卸载即取消”杀死3例，“Redo后自动重发”杀死1例，“按开始时位置写回”起初存活——`paste.ts`按`completed`给同id节点补src掩盖了单图错误写入，新增两图用例（另一张在途图片移到原位置）后杀死（`t3-mut-start-position-2`）。编辑器包14文件145项、类型、改动文件oxlint 0/0与格式；重建后应用受影响19文件174项（`t3-web-affected`），i18n键同步与使用检查通过。应用适配`editor-assets.ts`无需改动。

T4（re-50 F6）：`use-editor.ts`以`[editable, images.signal]`重建编辑器，“保存保留的输入”核验期间把`editable`切成false，旧编辑器随之销毁，`transfers.destroy()`中止恢复后新开始的上传、丢掉File；核验又把在途图片抛出的“未完成”显示成“图片不可用”。`description-input/root.tsx`改为：`editable`只看`disabled`与`replacing`，编辑器包在`display: contents`的容器里，核验期间`inert`并`aria-busy`（与评论表单提交期间的做法一致）；有未完成图片时点保存不核验，由既有“未完成”状态报告，传输完成后再点保存即核验并恢复自动保存。`onChange`不在核验期间丢弃：核验结果本就只在HTML未变时生效（`live()`），丢弃会让核验期间完成的上传只写进编辑器、不进本地描述。

RED（`t4-red`）：两例都在第一处断言失败——上传中点保存后编辑器元素被换掉；核验等待读取时可编辑编辑器消失（`tab.editor()`为null）。GREEN首跑（`t4-green`）暴露两处测试自身步骤与既有界面不符：恢复后的保存被422拒绝时入口是“重试”而非“保存保留的输入”，被拒图片节点显示自身错误而不在DOM中带id——改为点重试、断言节点显示`errors.asset_not_found`并断言被扣住的读取正是该图片，产品代码未因此改动；`t4-green-2` 20/20。变异：“核验时切换editable”杀死2、“未完成也核验”杀死1、“去掉inert”杀死1、“去掉aria-busy”杀死1，源码SHA均恢复。`vitest related`覆盖root.tsx的28文件366项（`t4-web-related`）、应用类型（`t4-web-types`）、改动文件oxlint 0/0与格式、关键词检查通过。

恢复版本（`replacing`）仍切换`editable`：恢复整体替换正文，在途传输本应随之结束；只有失败的恢复会多余地结束它们，而让恢复期间保持同一编辑器还需要`change()`接受恢复期间的传输写回（现由`owner.replacing`丢弃）——不属§3第3条，记入台账待最终审查评估。

T5（re-50 F5、F7a）：删除回答丢失后的核验原先把404与200一样记为“已核验”，对话框随即重新允许“删除文件”（只会再得404），该ID也留在`uncertain`，附件区的“删除结果尚不确定”在控件重建前一直显示；`ConfirmedContentNotice`则在宿主整个寿命内显示最后一次收据。现在`ContentDeletion.verify()`返回仍可见的不确定ID：得404的ID离开`uncertain`，不生成收据、不伪造204、不另发请求；附件与草稿准备件资源新增`verifyDeletion()`，刷新控件的`retryRead()`仍只读。对话框据此进入“文件已不可见、无需删除”状态（新增en/zh-CN键`content_files.delete_gone`），焦点移到这段说明，只留“关闭”。收据：`IssueDetailInputState.beginCommand()`在同一宿主有传输上线（选择器`onBusyChange(true)`）或删除请求发出时清除收据与待归还焦点；重挂载的附件控件只在焦点确实丢失时调用`takeFocus()`，取回焦点即清除收据——用户在重读期间把焦点移到别处时，收据保留到下一个命令。404分支不触发Issue权威重读：那次重读会撤回正文、带走对话框的说明并丢失焦点，而无收据可以归还它；附件数随下一次规范读取更新，与其他成员的并发修改相同。

RED（`t5-red-2`）：两项单元断言（取回焦点后收据仍在、`beginCommand`不存在）与六项宿主用例（detail/peek各三：404后等不到“不可见”说明；焦点归还后收据仍显示；新删除命令开始后旧收据仍显示）。首跑`t5-red`里另有一项既有peek焦点用例失败，原因是本次新用例中途失败时把聚焦的按钮留在页面上——改为`onTestFinished`清理后不再出现，不计产品RED。GREEN（`t5-green`）38/38。变异：“404仍留uncertain”杀死2、“收据常驻”杀死3、“得404仍允许再确认”杀死2、“删除不清收据”“上传不清收据”“焦点未丢也取走”各杀死2，源码SHA均恢复。`vitest related`覆盖九个改动源文件的254文件3020项（`t5-web-related`），核验得200后可显式再确认的既有用例照旧通过；应用类型、i18n同步与键使用、改动文件oxlint 0/0与格式、关键词检查通过。

T6①（re-50 F3）：带图片的普通副本先建目标草稿、复制图片再发布；发布回答丢失时弹窗显示“可能已建成工作项，先核对草稿”，核验发现草稿已不在时显示“已不在草稿中，先核对项目的工作项”。但关闭弹窗的提示只看`copy.target`，一律说“副本保留在草稿中，打开草稿完成或删除”——已发布时这会把用户引向不存在的草稿，并可能重建一个重复的工作项。`issue-modal/base.tsx`的`finishClose`改为按副本状态提示：核验得已发布时提示核对工作项（`content_files.copy.published`，不附草稿链接）；发布结果未核验时提示可能已建成（`content_files.copy.unknown_publish`，附草稿链接）；其余仍是“保留在草稿中”。未新增文案。

RED（`t6a-red`）：未核验与已发布两例关闭后都得到“保留在草稿中”。GREEN（`t6a-green`）10/10；变异“已发布仍提示保留”“未核验仍提示保留”各杀死1。`vitest related`覆盖base.tsx的47文件545项（`t6a-web-related`）、应用类型、改动文件oxlint 0/0与格式、关键词检查通过。

T6②（M4链接提示，本地修复）：链接创建/更新表单的成功提示原先由表单自身的scope守卫；写入之后的规范重读撤回正文，表单随之卸载，确认到达时提示被跳过——链接已建成，用户却没有任何反馈。`create-update-link-modal.tsx`在提交时取所在详情宿主的输入状态（`useIssueDetailInput()`）与其`epoch`：宿主仍current时提示一次，宿主已结束（换工作项、换路由、权限变化）不提示；关闭表单仍只在表单自己current时进行，被撤下的表单不再调用关闭。写入与列表结果不变。子项/关联经选择弹窗提交，M4起只有失败提示，不新增成功提示；P7b记录§“P8未完成事项”与M4交接中“链接/子项/关联提示被跳过”的说法已更正，并登记在`frontend-changes`的M4详情一行。

RED（`t6b-red`）：表单被写后重读撤下后确认、表单实例被关闭后确认，两例都没有提示。既有用例“已关闭实例的确认不清除也不关闭重开的表单”原断言没有提示，那是§3第5条要取代的表单scope规则，改为恰好一次提示、重开的表单仍不被清除或关闭；新增“宿主结束后确认不提示”守住另一侧。GREEN（`t6b-green`）7/7；变异“提示随表单寿命”杀死2、“总是提示”杀死1、“表单不在也关闭”杀死3。`vitest related`覆盖该表单的30文件372项（`t6b-web-related`）、应用类型、改动文件oxlint 0/0与格式、关键词检查通过。

T7（删除而非转交）：`1718210a`删除三处无读者代码。`EditorRefApi.copyMarkdownToClipboard`（editor的`helpers/editor-ref.ts`与`types/editor.ts`，`dist`随构建）只有描述版本弹窗测试里一个断言它未被调用的绑定，一并删除；后续复制走`getMarkDown`与`useCopyText`。`web/apps/web/app/assets/attachment/`的17个PNG（含4个第三方标志）全仓无引用（`git grep`含字符串、动态路径与样式），整目录删除。`WorkspacePatch.Logo`无赋值者也无读者：删除后`go build ./...`、`go vet`与workspace包测试（`t7a-go-workspace`，含postgres适配器）通过；同文件`Workspace.Logo`仍在使用，未动。editor类型与14文件145项、重建editor后web类型、描述版本测试3文件15项、`make knip`通过。M4交接中复制方法的具名风险与M1交接的附件图标条件由T13按“已删除”关闭。

死成员：按M1收尾计划附录A取出三个脚本（SHA-256：`deadsym.mjs` 3d85f090…、`lib/program.mjs` b2c52513…、`domains.mjs` 5f76ea3b…，TypeScript 5.8.3），`react-router typegen`后全量扫描，原始输出保存在P8工作区（`deadsym/dead-before.tsv`、`rows-M5-before.txt`及删除后的`-after-1`）。删除前合计prop 344、member 874、export 5，`--rows M5`为member 53、prop 3；第二个提交删除只为死成员存在的代码：`ContentReads.retry`与`invalidate`（无调用方；`readWith`的强制参数随之只剩false，并回`read`；`ContentReadEntry.invalidate`只被后者调用，一并删除），`PagedContent.retryRead`（两个子类各自定义，基类实现无人调用），editor的`TEditorAsset`（整个`types/asset.ts`与其再导出），`ProfileImageLease.key`（只被一处测试断言读取，断言随之删除），`ContentConfirmation.asset`（收据只读kind与id；输入状态的`Receipt`改为直接用`ContentConfirmation`，去掉重复类型），`CoverImage.fallbackUrl`（无人传入），以及T3自己引入、只在本文件使用的`ImageTransferState`导出。删除后合计prop 343、member 859、export 3，全量逐行对比没有新出现的行（无孤儿）；M5剩41行，均为误报：

| 文件 | 成员 | 读者（或写者） |
|---|---|---|
| `components/assets/project-cover.tsx` | `kind` | `source`原样传给`ProfileImage`，由`useProfileImage`经`ProfileImageSource`读取 |
| `issue-detail-widgets/attachments/host.test.tsx` | `children`、`removeRoutePeekId`、`data`、`next_cursor` | 模拟的`Link`与peek页头读前两者（由被测组件经模块模拟写入）；页对象经`json(200, page)`序列化后由附件列表客户端读取 |
| `issue-layouts/attachment-count.test.tsx` | `children`、`as`、`observe`、`unobserve`、`disconnect`、`order`、`shown` | 模拟的`RenderIfVisible`读前两者；`QuietObserver`的方法由列表续页代码经全局观察器接口调用；后两者由同文件`mount()`读取 |
| `lib/asset-download.test.ts`、`lib/asset-upload.test.ts` | `href`、`name`、`attached`、`url`、`init` | 记录的点击与请求由同文件断言（`toEqual`整体比较或逐项读取） |
| `services/asset.service.test.ts` | `name`、`method`、`path`、`body`、`query`、`answer`、`run` | 表驱动用例由同文件读取，`name`由vitest的`$name`标题插值读取 |
| `store/asset/content-read-entry.ts` | `stamp`（非ready分支）、`error`（ready分支） | 判别联合的同一字段：生产在收窄后读取另一分支的同名成员（`content-read-entry.ts`的loaded/failed、`hooks/editor/image-read.ts`的stamp与非ready分支的error），测试经联合整体读取 |
| `store/asset/fake-assets.ts` | `file`、`signal`、`fail` | 假PUT记录，由各测试读取 |
| `store/asset/upload-policy.ts` | `purpose`（三个分支） | `upload-session.ts`把冻结的`target`展开进创建资产的请求体，经生成的请求类型读取 |
| propel `empty-state/assets/asset-registry.tsx`、`helper.tsx` | `className`（注册表五处类型字面量与插图props） | 插图组件经`TIllustrationAssetProps`读取（25处），`getCompactAsset`/`getDetailedAsset`经注册表类型传入 |
| `spreadsheet/columns/attachment-column.tsx` | prop `issue` | `spreadsheet/issue-column.tsx`经`SPREADSHEET_COLUMNS`以`TSpreadsheetColumn`写入 |

验证：editor类型与14文件145项、重建后web类型、`vitest related`覆盖八个改动源文件的254文件3024项（`t7b-web-related`）、`make knip`、改动文件oxlint 0/0与格式、关键词检查通过。

T8（两条规则清零）：全仓按包按规则的原始统计保存在P8工作区`lint/before`与`lint/after`。命中全在web：`valid-typeof` 4处、`no-constant-binary-expression` 5处，即`lib/local-storage.ts`三处与`hooks/use-local-storage.tsx`一处的`typeof window === undefined ||`（`typeof`的结果是字符串，与`undefined`值比较恒假），以及`components/settings/helper.ts`的`` `/${accessKey}` || "" ``（模板字符串恒真）。删除死分支前先补三份小测试（存储的读写删与空值、无`window`时不抛错并返回空；钩子从已存JSON起步、写回、清除后由自身事件回到初始值、坏JSON被丢弃；设置路径的工作区与访问键），删除前后均9/9（`t8-before-2`、`t8-after-2`）；变异“去掉无window守卫”被杀死。随后在根`.oxlintrc.json`的`rules`把两条设为error，web的cap 67→58，其余包不变；重新加回一处命中时web的`check:lint`以“errors are never allowed”失败（`t8-mut-hit-returns`）。`make lint-web`首跑因新钩子测试的泛型推断成字面量`false`而类型失败，补类型参数后50/50通过（`t8-make-lint-web-2`）；M5全期（`3db22f06..HEAD`）改动的395个web与tools文件逐包oxlint零诊断（`t8-m5-changed-lint`）。

T9（S3夹具的两个错误）：子进程`s3-container.ts`的catch原先只回送固定字符串“owned S3 container failed”，父进程再换成通用的“S3 child startup failed”，P7b T14第三轮CI失败时只能看到这句；`startS3`启动失败后`await close()`，清理若也失败，抛出的就只剩清理错误。现在子进程把原错误的name、message与stack经IPC回送，父进程重建为`cause`；`startS3`与`closeProcess`的双失败各以`AggregateError`保留两者（沿`e2e/fixtures/test.ts`先例）；清理函数作为参数注入，默认仍是`removeS3Container`；等待期限不变。为在`s3.ts`层毫秒级、不启动容器地让启动失败，`startS3`的选项除清理函数外还接受`auth`——子进程拒收空的身份文件是唯一在任何容器之前失败的路径。先加注入点（行为不变）再写用例。`catch`里抛出的`AggregateError`以`oxlint-disable-next-line preserve-caught-error`注明原因：被捕获的错误就是聚合错误的第一个，该规则不读`AggregateError`的选项参数。

RED（`t9-red`）：子进程拒收启动时只有通用信息、没有`cause`；启动与清理双失败时只剩清理错误。GREEN（`t9-green-3`为最终代码）；变异“清理错误盖过启动错误”“子进程回送固定信息”均被杀死。用例追加在`stories/smoke/s3-lifecycle.spec.ts`，各约0.18秒，整份文件5/5（30.1→30.3秒），默认套件时长不变；真实容器回归：`t9-lifecycle`含既有三条容器用例，`test:m5-s3`的F4在Chromium、Firefox、WebKit各1条（`t9-m5-s3`，3/3）；e2e类型、lint（cap 0）、格式通过。`closeProcess`自身的双失败（子进程退出出错且清理失败）同样聚合，但没有用例：子进程的派生失败无法不加新注入点地制造。

T5更正（T10期间发现）：T5把plan“`takeFocus()`归还焦点后的正文重挂载时清除”读成了“取回焦点即清除”，于是“已上传/已删除”只在规范重读撤下正文的那一刻可见，正文回来、焦点归还的同一提交里就消失——视觉上一闪而过，读屏可能来不及读出；T10写慢速页面演练时，F8等页面故事断言提示可见，暴露了这一点。改为：取回焦点只把收据记为已送达，提示随此刻显示的正文保留；该正文下一次被撤下（再一次规范重读）时，已送达的收据随之结束；同一宿主开始新命令时照旧立即结束。用户在重读期间把焦点移到别处的情形不变（不调用`takeFocus()`，收据留到下一个命令）。实现只在`IssueDetailInputState`：`takeFocus()`置`delivered`，宿主对规范项在/不在的既有反应在“不在”且已送达时结束提示。RED（`t5b-red`）三项：单元断言取回焦点后收据仍在、detail/peek宿主用例断言焦点归还后提示仍显示。GREEN（`t5b-green-2`）38/38；变异“取回焦点即清除”“撤下时不清除”各杀死3；`vitest related`覆盖`detail-input-state.ts`的32文件419项（`t5b-web-related`）。

同时，T5对删除核验404的改变使P7b的F8页面故事过时：它在“核验最新状态”后断言“删除文件”重新可用、再发一次DELETE得幂等的204。按§3第4条改为断言说明获得焦点、对话框只剩“关闭”、关闭后“删除结果未知”的提示消失、DELETE仍只有一次、没有“已删除”收据、服务端附件数为49。此路径不触发Issue权威重读（T5 Ruling），页面上附件区标题的数字在下一次规范读取前仍是50，故事不断言它。重建后Chromium上F8两个变异（`t5b-f8`）、F9与F16页面故事（`t5b-page-stories`中除F8外17项）通过。

T10（全链路清理与慢流演练）：新增显式开启的`TestFilesCleanupDrill`（`NERVE_FILES_DRILL=1`，沿`NERVE_PURGE_SCALE`先例；默认`make test`跳过，默认CI时长不变），文件`server/internal/bootstrap/files_cleanup_drill{,_fixture}_test.go`，子测试local与s3。演练用生产组合`newAppWithFactories`，只经测试自有的工厂观察与注入：连接池挂查询追踪记录分页、资产工厂取得存储句柄、重启后以运行时钟+2小时模拟租约已过（同`restartAssetS3Cleanup`）、需要时以既有`imageDeleteFault`让一个对象删除失败；业务时钟经既有test环境配置`clock.offset`随重启推进。没有产品开关，没有只供测试的API。配置在`testConfig`上只改三项合法取值：`files.cleanup_batch=2`、`cleanup.purge_batch=2`（让每次清扫与purge都要走多页）、`jobs.periodic_workers=1`（启动时RunOnStart任务按注册顺序先清扫后purge，每一步状态可精确断言）。为让S3复用，既有夹具抽出`newContentAssetTreeAt`（按驱动给出直传base与成功状态，原`newContentAssetTree`委托它）与`addContentMember`，既有调用方行为不变。

演练对象：工作区A（acme）含六类内容用途（议题/评论/草稿的附件与描述图片，均经API绑定）、工作区图标与项目封面、一个未绑定准备件（草稿描述图片，草稿准备件列表中唯一一项）、一次失败复制（声明为PNG的非图片附件复制为议题描述图片：422 `asset.content_invalid`，目标行保留一个候选记录；源文件先过图片检查失败，候选对象从未写出——有对象的失败复制恢复由`TestDraftAssetPreparationsRecoverFailedCopy`证明，本次重跑）、一次未完成上传（申请后从未PUT）、一次慢流上传（3 MiB，按`pacedAssetBody`的160 KiB/s直传：local 19.116628625s、S3 19.125008375s，均>15秒；local在其传输预算内；完成后经API读回字节一致），项目封面最后创建。邻域：bob的工作区B（同样六类内容用途）与alice的头像。A共13行。每步精确断言：

1. 第一次重启（业务时钟不动）：A中11个已上传文件的暂存对象全部回收、失败复制的候选记录移除；13行无墓碑、无候选，只有未完成上传保留其暂存记录；11个final对象的SHA-256与行内记录一致（慢流文件另与上传字节的摘要一致）。此后取邻域基线：B的文件行、宿主行（工作区/项目/议题/评论/草稿）、alice头像引用与7个对象的摘要。
2. 删除A（204）后立即：读A的文件内容404、以A的文件为源复制404、在A内申请上传404、草稿准备件列表404 `asset.not_found`。项目、工作区草稿与`asset.delete_scope`任务各一次完成且沿用原At/By（`waitWorkspaceDeletion`）。13行等于上一步，只有删除时间/更新时间为工作区的`deleted_at`、删除人/更新人为alice；5个宿主行带同一`deleted_at`；字节全在；图标/封面引用不变；邻域不变。
3. 第8天（重启，+8天）：墓碑不被7天规则重盖，未完成上传那条从未写出的暂存记录因删除而回收，其余逐字段不变；字节全在；宿主、引用、邻域不变。
4. 第61天（重启，再+53天），封面final对象的删除在该组合中失败：清扫第1次尝试以“specific image key unavailable”失败，第2次完成，输出`failed=1`；purge一次完成：扫描41、删除39、受阻2（封面所在项目与工作区）。A只剩封面一行，删除审计不变、final处于reclaiming；其余10个final对象已删、封面字节仍在；议题、评论、草稿已purge，项目与工作区保留；工作区图标引用已解除（null），项目封面引用保留；邻域不变。
5. 下一周期（同日再重启）：清扫与purge各一次无错误；A无任何文件行，封面对象已删，项目与工作区已purge；邻域不变。

两种存储结果相同（local {'neighbour': 7, 'objects_reclaimed': 22, 'workspace_a': 13}；S3相同），清扫第61天输出与purge计数一致。分页与时间：每个`ListRecovery`/`ListScope`页LIMIT 2且不超过2行、每个`Purge…`页LIMIT 2且不超过2行；local上`ListRecovery`共49页（43页满）、`ListScope` 8页（6页满）、`PurgeAssetCandidates` 12页（6页满）；清扫、范围删除与purge任务共12次运行，最长262.133ms（S3 285.078ms），预算10s。原件：`t10-drill-local-2`、`t10-drill-s3-1`（各含证据JSON）。

变异（local，均被杀死）：删除审计丢失删除人（`t10-mut-audit-by`，删除一步失败）、final对象不等60天就回收（`t10-mut-purge-early`，第8天失败）、对象删除失败被吞掉而记录照删（`t10-mut-failure-swallowed`，第61天无错误而失败）、7天规则也盖已绑定文件（`t10-mut-seven-days-bound`，邻域检查失败）。

已有阶段测试证明的性质不在演练里重证，重跑原测试（`t10-phase-reruns`，31个顶层测试全部通过，另含资产模块8个包的全部测试）：`TestADeletionRefusedAtItsCommitChangesNoRow`、`TestAFailedProjectsStepRollsTheDeletionBack`、`TestAssetConfiguredRetentionWired`、`TestAssetDeletionFollowsCommentAndDraft`、`TestAssetDeletionFollowsIssueTree`、`TestAssetDraftPublicationDoesNotDeleteAssets`、`TestAssetParentDeletionReader`、`TestAssetParentEventRollsBackAfterSynchronousMark`、`TestAssetParentJobsUseOriginalTransaction`、`TestAssetParentLocksAndFacts`、`TestAssetProtectedParentDoesNotBlockWiredPurge`、`TestAssetProtectedParentsProgressAndConverge`、`TestAssetS3Lifecycle`、`TestAssetScopeDeletionRollback`、`TestAssetSweepRestarts`、`TestContentAssetDeletion`、`TestContentAssetRetention`、`TestContentAssetRetentionProcess`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestDeletionViolationsCatchesEachGap`、`TestDraftAssetPreparationsCancelReleasesSnapshot`、`TestDraftAssetPreparationsCompletedAndHistoricalImages`、`TestDraftAssetPreparationsOtherWorkspaceUnaffected`、`TestDraftAssetPreparationsRecoverFailedCopy`、`TestDraftAssetPreparationsRecoverLostAnswer`、`TestDraftAssetPreparationsSnapshot`、`TestImagePurgeProcess`、`TestImageReferenceRetention`、`TestPurgeCursorRiverSnoozeAndRestart`、`TestPurgeDeletedRunsWithRuntimeRole`、`TestRawStorageDefaultSlowUpload`。

页面慢流（T1裁定补入）：`e2e/drills/f1-slow-page-upload.spec.ts`，独立配置`e2e/playwright.m5-slow.config.ts`（testDir为drills，只Chromium）与脚本`test:m5-slow`，不进默认套件。Chromium经DevTools把上行限到72 KiB/s，详情页附件区上传1,310,720字节：local与S3的PUT各约17.8秒、从选择文件到计数变为1约18.4秒，均>15秒；页面给出“File uploaded.”，服务端字节一致（`t10-slow-page-3`，含报告JSON）。演练等待附件计数变为1（90秒）后再读取请求，因为夹具对请求的5秒轮询短于这次传输。T5更正与F8页面故事更新即在写这条演练时发现，见上节。

T10期间另发现：M4的显式规模测试`TestPurgeScale`的查询追踪只认以`-- name: Purge`开头、带3个参数的DELETE；M5/P2起purge协议改为`…Candidates`查询加`DeletePurge…`删除，它会把候选查询记为错误并漏掉父行删除，M5期间该测试从未开启（P5/P6评审记录）。T12显式运行确认，确认后只修测试追踪。

