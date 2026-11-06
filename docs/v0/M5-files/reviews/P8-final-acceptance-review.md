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

两种存储结果相同（local {'neighbour': 7, 'objects_reclaimed': 22, 'workspace_a': 13}；S3相同），清扫第61天输出与purge计数一致。分页与时间：每个`ListRecovery`/`ListScope`页LIMIT 2且不超过2行；每条`Purge…`语句的LIMIT参数都是2，其中候选查询（`Purge…Candidates`）每页不超过2行，单语句分页（`Purge<表>`）只回一行汇总，追踪器不观测它删了几行，删除数由purge输出与每一步的逐行快照核对（T15审查更正）；local上`ListRecovery`共49页（43页满）、`ListScope` 8页（6页满）、`PurgeAssetCandidates` 12页（6页满）；清扫、范围删除与purge任务共12次运行，最长262.133ms（S3 285.078ms），预算10s。原件：`t10-drill-local-2`、`t10-drill-s3-1`（各含证据JSON）。

变异（local，均被杀死）：删除审计丢失删除人（`t10-mut-audit-by`，删除一步失败）、final对象不等60天就回收（`t10-mut-purge-early`，第8天失败）、对象删除失败被吞掉而记录照删（`t10-mut-failure-swallowed`，第61天无错误而失败）、7天规则也盖已绑定文件（`t10-mut-seven-days-bound`，邻域检查失败）。

已有阶段测试证明的性质不在演练里重证，重跑原测试（`t10-phase-reruns`，31个顶层测试全部通过，另含资产模块8个包的全部测试）：`TestADeletionRefusedAtItsCommitChangesNoRow`、`TestAFailedProjectsStepRollsTheDeletionBack`、`TestAssetConfiguredRetentionWired`、`TestAssetDeletionFollowsCommentAndDraft`、`TestAssetDeletionFollowsIssueTree`、`TestAssetDraftPublicationDoesNotDeleteAssets`、`TestAssetParentDeletionReader`、`TestAssetParentEventRollsBackAfterSynchronousMark`、`TestAssetParentJobsUseOriginalTransaction`、`TestAssetParentLocksAndFacts`、`TestAssetProtectedParentDoesNotBlockWiredPurge`、`TestAssetProtectedParentsProgressAndConverge`、`TestAssetS3Lifecycle`、`TestAssetScopeDeletionRollback`、`TestAssetSweepRestarts`、`TestContentAssetDeletion`、`TestContentAssetRetention`、`TestContentAssetRetentionProcess`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestDeletionViolationsCatchesEachGap`、`TestDraftAssetPreparationsCancelReleasesSnapshot`、`TestDraftAssetPreparationsCompletedAndHistoricalImages`、`TestDraftAssetPreparationsOtherWorkspaceUnaffected`、`TestDraftAssetPreparationsRecoverFailedCopy`、`TestDraftAssetPreparationsRecoverLostAnswer`、`TestDraftAssetPreparationsSnapshot`、`TestImagePurgeProcess`、`TestImageReferenceRetention`、`TestPurgeCursorRiverSnoozeAndRestart`、`TestPurgeDeletedRunsWithRuntimeRole`、`TestRawStorageDefaultSlowUpload`。

页面慢流（T1裁定补入）：`e2e/drills/f1-slow-page-upload.spec.ts`，独立配置`e2e/playwright.m5-slow.config.ts`（testDir为drills，只Chromium）与脚本`test:m5-slow`，不进默认套件。Chromium经DevTools把上行限到72 KiB/s，详情页附件区上传1,310,720字节：local与S3的PUT各约17.8秒、从选择文件到计数变为1约18.4秒，均>15秒；页面给出“File uploaded.”，服务端字节一致（`t10-slow-page-3`，含报告JSON）。演练等待附件计数变为1（90秒）后再读取请求，因为夹具对请求的5秒轮询短于这次传输。T5更正与F8页面故事更新即在写这条演练时发现，见上节。

T10期间另发现：M4的显式规模测试`TestPurgeScale`的查询追踪只认以`-- name: Purge`开头、带3个参数的DELETE；M5/P2起purge协议改为`…Candidates`查询加`DeletePurge…`删除，它会把候选查询记为错误并漏掉父行删除，M5期间该测试从未开启（P5/P6评审记录）。T12显式运行确认，确认后只修测试追踪。

T11（默认部署演示，F18）：`e2e/fixtures/deployment.ts`的`deploymentTest`照README“部署”节部署一个实例：在测试Postgres上新建一个空库（不取模板），用Node `crypto`生成Ed25519私钥（PKCS#8 PEM，权限600），以`NERVE_ENV=prod`执行`nerve migrate up`，再`nerve users create --email …`（密码经标准输入）建第一个账户，然后`nerve serve`。`startNerve`与`runNerve`新增显式选项`deployment`：不注入`NERVE_ENV=test`，也不注入测试用的传输容量（每分钟10000、突发1000）；默认关闭，其他故事的环境不变。故事`e2e/stories/files/f18-default-deployment.spec.ts`由新配置`e2e/playwright.m5-demo.config.ts`（Chromium、Firefox、WebKit）与脚本`test:m5-demo`运行；基础配置以`testIgnore`把它留在默认故事之外（`playwright test --list`共417项、108个文件，不含F18）。

全部环境变量（serve）：`NERVE_ENV=prod`；`NERVE_DATABASE__URL`（夹具给出，指向那个空库，带`application_name=nerve`）；`NERVE_AUTH__JWT__PRIVATE_KEY_FILE`（上述私钥文件）；`NERVE_FILES__LOCAL__ROOT`（本实例独占的目录）；`NERVE_SERVER__ADDR`（首次`127.0.0.1:0`，重启时为同一个`127.0.0.1:端口`）；`NERVE_SERVER__ADDR_FILE`（nerve写出实际地址的文件，夹具据此找到端口）。`migrate up`与`users create`只有前四个。此外只继承运行者环境中非`NERVE_`开头的变量。nerve启动时记录的有效配置（“configuration loaded”，数据库地址由nerve自己遮蔽）：env prod；server读30秒、写60秒、请求15秒、请求体1 MiB；注册关闭；`auto_migrate`关闭、连接池10；业务时钟偏移0；files为local，单文件5 MiB、上传授权15分钟、传输限速每分钟60次突发20、最低速率128 KiB/s、宽限15秒、并发16、清理每小时一次、每批100、预算10秒——除上面的变量外全是基础配置的默认值。故事先断言这条记录。

故事：第一个账户经API标记已完成引导，并建工作区、项目和工作项（主机经API备好，与既有页面故事相同）；浏览器经登录页登录（200），随后每个文件都由页面写入：头像（个人设置“Change avatar”→“Upload and save”，`PATCH /api/v0/me` 200，头像解码显示）；工作区图标（工作区设置，`PATCH /api/v0/workspaces/deployed` 200，切换器图标解码显示）；项目封面先选预设“Cover 29”（`cover_builtin_id: cover-29`），再上传图片（`cover_image_asset_id`）；附件（详情页附件区选择文件，“File uploaded.”，计数变为1，浏览器下载的SHA-256与原字节一致）；正文粘贴一个PNG文件（保存的描述带这张图片的标识，图片解码显示）；带文件的评论（发表201，评论的附件区下载，SHA-256一致）。服务端六个文件（`avatar.png`、`logo.png`、`cover.png`、`report.txt`、`pasted.png`、`comment-note.txt`）均已绑定、未删除，行内SHA-256与原字节一致，独占目录中的对象字节一致，经授权的API读取一致。随后页面停在`about:blank`，进程在同一地址、库、密钥与目录上重启；六行与重启前逐字段相同；页面会话延续，头像、切换器图标、项目设置中的上传封面、正文图片均解码显示，附件与评论文件再次下载，SHA-256一致。`watchPage`：无`/api/v0`之外的请求、无CSP违规、无未处理异常、无失败的接口请求；每个浏览器184个接口请求。

结果（`t11-browsers-3`，原件含报告JSON与`t11-artifacts/<浏览器>/nerve-{0,1}.log`）：chromium 12.5秒、firefox 15.1秒、webkit 14.0秒，3/3通过。RED（`t11-red`）：选项加入前夹具注入了测试容量，配置断言得到每分钟10000、突发1000而失败；加入后通过。变异：夹具在部署模式下仍注入测试容量（`t11-mut-test-capacity`，配置断言失败），重启换到另一个文件根（`t11-mut-other-root`，重启后第一次经授权读取失败），均被杀死。开发中对故事本身的更正（均非产品问题）：时钟偏移在日志中是数字0；描述保存的请求体按`bodiesSentTo`的说明经路由读取（响应所属的请求不带网页客户端发出的请求体）；切换器图标须在工作区页面上看；离开页面前等网络空闲（否则Firefox把被这次导航截断的读取记为`NS_BINDING_ABORTED`、WebKit报“Frame load interrupted”）；重启期间页面停在`about:blank`。

Step 2扫描：产品源码中没有旧文件路由（`/api/assets`、`assets/v2`、`file-assets`）——两处命中都在测试替身`core/store/asset/fake-assets.ts`中，它定义这些旧路由以断言从未被调用；`web/packages/services`不存在；关键词守卫当前阶段M5/P8，14条例外全部在M6及以后到期，无命中；死成员按附录A重扫（脚本SHA与T7相同）：`domains.mjs --rows M5`仍是T7表中的41行（35个符号），无新增、无消失（`deadsym/dead-after-2.tsv`、`rows-M5-after-2.txt`）。`/api/v0`之外的路径字面量只剩M6/M7领域尚未迁移的服务（周期、模块、视图、通知、intake、收藏、webhook），演示中三浏览器均未请求。README“部署”节的本地文件存储一条删去“用户申请上传与附件页面随后续阶段接入”的过时说法，写明默认配置下六类文件的字节都在文件根中并指向演示；“端到端测试”一节写入`test:m5-demo`、`test:m5-slow`与Go清理演练的运行方法。

T12（F1–F18最终矩阵）：最终产品候选的产品代码即T11的`24d2e507`，本Task只增两个测试提交：`54e07f76`（`TestPurgeScale`按M5清理协议观察，见下）与`da82f250`（F11页面故事更正与内容页夹具`quiet()`，见下）。按plan的顺序逐个运行、同一时刻只有一组容器套件（原件均在P8工作区，每次运行含收据、输入清单与日志，Playwright另有JSON报告与独立的`-results`目录）：`make build build-s3-fixture`（t12-build）；默认故事（e2e按`make e2e`的同一命令，只加JSON报告与独立输出目录）t12-e2e-3 417/417（211秒）；`m5-profile`三浏览器42/42（t12-m5-profile）；`m5-content`三浏览器local 63/63（t12-m5-content-4）；`m5-s3`（F4三浏览器）3/3（t12-m5-s3）；`m5-demo`三浏览器3/3（t12-m5-demo-2）；`m5-slow`（local/S3）2/2（t12-m5-slow-2）；`NERVE_FILES_DRILL=1`清理演练local与s3通过（t12-drill，79秒）；`NERVE_PURGE_SCALE=1`通过（t12-purge-scale-final）；`GOFLAGS='-p=1 -v' make test`（-v只为留下逐测试行）61个包全过、0失败、7项跳过均为显式开启的测量/演练或子进程辅助（t12-go，997秒）。另在M5代码上重跑M4的`TestTheDeletionsAtTheScaleTarget`（NERVE_MEASURE_SCALE=1，t12-deletion-scale）：原样通过，单项目删除任务7.6秒、五项目14.3秒，处理期间邻工作区PATCH最大68ms（M4记为6.4秒、14.6秒、576ms）。

输入一致性：t12-e2e-3、t12-m5-content-4与两次F11隔离运行的输入清单与`da82f250`的工作树逐文件一致；t12-m5-profile、t12-m5-s3、t12-drill、t12-purge-scale-final、t12-go运行于`54e07f76`，与最终树只差F11故事与内容页夹具两个e2e文件——前两者的故事不导入该夹具，后三者不读e2e文件；t12-go的“输入有变”即运行期间改了这两个文件。电脑重启中止了t12-m5-content-2（exit 137，无收据，原件保留）；重启后Docker Desktop自动启动了其他项目的容器（未触碰），此后的运行与其共享主机资源，只影响绝对耗时。

| F | Go（T1目录所列文件中的测试，t12-go） | 故事与演练（本次运行） |
|---|---|---|
| F1 | 4个文件7项全过 | F1 (API)（默认，Chromium）；F1 (page, slow network)（m5-slow，Chromium local/S3） |
| F2 | 7个文件12项全过 | 无故事（Go慢流）；页面慢流见F1行；`TestFilesCleanupDrill`慢流上传local/S3（t12-drill） |
| F3 | 5个文件7项全过 | F3 (API)（默认） |
| F4 | 8个文件8项全过 | F4 (browser)（m5-s3三浏览器＋默认Chromium） |
| F5 | 6个文件9项全过 | F5 (API)；F5 (page)两项（m5-profile三浏览器local＋默认Chromium local/S3） |
| F6 | 4个文件5项全过 | F6 (API)；F6 (page)（m5-profile三浏览器＋默认local/S3） |
| F7 | 2个文件2项全过 | F7 (API)；F7 (page)（m5-profile三浏览器＋默认local/S3） |
| F8 | 9个文件12项全过 | F8 (API)（默认local/S3）；F8 (page)（m5-content三浏览器local＋默认local/S3） |
| F9 | 4个文件4项全过 | F9 (API)；F9 (page)（同上） |
| F10 | 7个文件26项全过 | F10 (API)三项（默认local/S3）；F10 (page)三项（m5-content三浏览器＋默认） |
| F11 | 3个文件7项全过 | F11 (API)；F11 (page)（m5-content三浏览器＋默认local/S3；本节更正后另隔离3轮15/15） |
| F12 | 5个文件15项全过 | F12 (API)；F12 (page)两项（m5-content三浏览器＋默认local/S3） |
| F13 | 7个文件10项全过 | F13 (API)（默认） |
| F14 | 6个文件9项全过 | F14 (API)（默认）；`TestFilesCleanupDrill`删除与立即404（t12-drill local/S3） |
| F15 | 12个文件16项全过 | F15 (API)（默认）；`TestFilesCleanupDrill`第8/61天与故障重试（t12-drill）；`TestPurgeScale`（t12-purge-scale-final） |
| F16 | web单元在T14门禁 | F16 (page)二十二项：资料页9项（m5-profile三浏览器＋默认）、内容页13项（m5-content三浏览器＋默认，两项另含S3） |
| F17 | web单元在T14门禁 | F17 (page)（m5-profile三浏览器＋默认）；页面故事中的可访问性断言 |
| F18 | 关键词守卫、knip在T14门禁 | F18 (page)（m5-demo三浏览器） |

F11更正：T2把普通文件的粘贴交回ProseMirror的粘贴管线（§3第1条），但T2只以编辑器与应用单元测试验证，F11页面故事直到本Task才重跑，仍按P7b合同断言“粘贴事件未被拦下”；ProseMirror管线中Plane的代码块插件（`codeBlockVSCodeHandlerCustom`）对代码块外的每次粘贴先`preventDefault`，于是t12-e2e（Chromium local/S3）与t12-m5-content（三浏览器）一致失败于同一断言。这是故事过时，不是产品缺陷：故事改为断言合同本身——普通文件的拖放仍不拦下；带着文件名文字的普通文件粘贴由ProseMirror插入文字并保存，不上传、无提示。更正后的首次全量（t12-e2e-2）在Chromium local暴露故事自身的换页竞态：保存后的规范重读重挂载正文并再次读取图片，故事随即换页，`GET …/content`被中止（`net::ERR_ABORTED`）；Playwright的`networkidle`在早已加载的单页应用上立即返回，不能表达这一等待，内容页夹具因此新增`quiet()`（无`/api/v0`请求在途且500毫秒无活动），F11在换页前调用。隔离3轮：默认local/S3 6/6、m5-content三浏览器9/9（t12-f11-default、t12-f11-content）；变异“普通批返回true”（t12-mut-f11-ordinary-true，重建后运行）在等待含文件名的保存处超时被杀死，恢复后重建（t12-build-2，缓存命中原产物）。

间歇项（re-50§6/P7b `t13-content-local-4`）：Firefox F16换工作区故事隔离重复5/5（t12-repeat-f16-firefox），三次三浏览器全量均通过。WebKit F8隔离重复5/5（t12-repeat-f8-webkit），但在t12-m5-content-3全量中复现一次，签名与P7b相同——点击“Choose or replace file”后180秒内没有`filechooser`事件；该控件是可见的原生`<input type="file">`加`<label htmlFor>`，点击与系统选择器之间没有应用代码，不存在应用异步丢失用户激活的路径。结论：自动化层（WebKit在满载下的选择器拦截）间歇，紧接的全量重跑63/63（t12-m5-content-4），记入T14/CI观察；默认CI不跑WebKit。

评论卡成本（re-50§6交P8实测）：一次性测量故事（未提交，脚本存于P8工作区`t12-comment-cards.spec.ts`，`t12-comment-cards`）在同一工作项详情页各打开1条与40条带图评论，计时到全部评论图片解码：local 818ms/895ms、S3 726ms/858ms；接口请求26/105，每张图恰好一次元数据读取与一次内容授权读取（`GET /api/v0/assets/:id`与`…/content`各40），没有重复读取。成本随图片线性增长、每卡无额外请求，维持re-50的“每卡一条reaction与一次登记”。

清理规模（`54e07f76`，Ruling见台账）：M4的`TestPurgeScale`假定一次运行、无snooze，追踪器只认M4的三参DELETE，在M5的有界清理上必然失败（t12-purge-scale：`State:available Snoozes:1`）。按M5协议更新：一次attempt可分多次预算内运行，job观察器在离开running的同一UPDATE中读出每次运行的自报输出；追踪器认三种语句形状，参数按类型读取；受保护表逐表核对DELETE命令标签；分页表以各次运行自报删除总数对旧范围行数，并先核实旧范围之外无可清理行，推出无一行被级联删除；预算取消的在途语句只在紧随其后的就是它的重做时接受。变异：先清`issue_comments`使`comment_reactions`被级联删除1行（t12-mut-cascade，只被总量核对杀死：660025≠660026）；每次运行从头开始并换截止时间（t12-mut-no-resume，被重做规则杀死）。诊断（t12-diag-plans）发现原夹具在播种后、软删与老化前ANALYZE，`deleted_at`统计为全NULL，规划器每页读遍全部墓碑，直到autovacuum下一轮分析；夹具改为老化后ANALYZE。最终一次运行5.0秒删除660,026行（M4同数据单次12.2秒）；统计陈旧时仍有界并续跑完成（t12-purge-scale-3：6次运行、5条被取消的语句均重做、合计52秒）。

其余：`TestAllPeriodicJobsRun`的独立清单仍是M4的四种周期任务，未含`asset.cleanup`（十余个默认测试等待其启动运行），记为Minor。CI的e2e余量（约21.2/20.5分钟，预算25分钟）在T14的准确检查点CI复核；本地默认故事417项3.5分钟。

T13（文档同步与交接）：七份输入交接各写“处理结果（M5/P8）”，保持open，由收尾改为closed。M0加固：P2留下的S3与页面慢流由T10补齐。M1-P2/P3：逐项结论。M1收尾：死成员`--rows M5`由56行降为41行，余下全为误报并逐行写明读者；两条告警规则清零；附件图标整目录删除。M2：§2的`use-file-size.ts`已删，实际读取点为`core/hooks/editor/editor-assets.ts`读`root.instance.config.file_size_limit`，超限413 `asset.too_large`；另记`86c9f9f6`。M3：弹窗、焦点与可访问错误的核对。M4：M5-01/02/03的最终证据；旧复制方法写“已删除”而不是“已修复”；P7b T14段已只涉链接表单（T6②）；M5-04原文整段转交。

三份`M5-closeout`：[M6](../../M6-cycles-modules/handoffs/M5-closeout.md)写资产用途与宿主接入的六步（契约与迁移、访问、父读取、删除、富文本图片、purge登记）、迭代/模块列表中的附件计数、有界可续跑的清理协议与`TestPurgeScale`认的三种语句形状。[M7](../../M7-collaboration/handoffs/M5-closeout.md)写收集箱的现状与接入条件、附件事件（名字、载荷、发出点、写入事务内同步投递且没有outbox）与通知的责任、只读显示与复制为Markdown。[M8](../../M8-open-release/handoffs/M5-closeout.md)写local/S3的部署条件与CSP、`files.*`配置表（含共享清理预算放在`files.cleanup_budget`的命名提醒）、备份与实例独占（清理删除早于24小时且无行引用的对象，恢复或预发实例不能指向生产存储）、Webhook、发布前不在默认CI中的验收（`test:m5-demo`、`test:m5-slow`、三浏览器矩阵、清理演练与两项规模测试）及仍开放的观察（404后附件区标题计数、陈旧统计下的清理、多热项目公平、WebKit间歇、`TestAllPeriodicJobsRun`清单）。M5-04的原文三份各一，嵌入时与M4交接逐字核对。资产面的事实先由一次只读检索汇总，再逐项对照源码核实：事件名与载荷、订阅、草稿发布时的事件、计数SQL、413、CLI没有投递客户端。

README：新增“文件（M5）”与“S3存储”两条；M4条目的“不启用图片上传和附件”改为指向M5；CSP一条补S3来源；本地存储的备份一条补清理与实例独占。plane-diff：`users`、`workspaces`、`projects`三行由“暂不建，由M5加入”改为P4迁移00035–00037的实际列与约束。frontend-changes：M5进度行；块ID插件`bdc6909d`登记在表格插件`8d85db5a`旁。M5设计：§13写出三份交接的落点，§15更新P8状态。re-32的两处与re-41的一处断链已修复。`doc-links.py`：102份文档、769个相对链接、0断；关键词守卫76条规则、14个例外，无命中。服务端配置样例注释已与M5一致，未改。

T14（七门禁与检查点）：在`460a5464`上串行运行七门禁，全部exit 0、每次运行前后输入清单一致：`make gen-check`（t14-gen-check，5秒）、`make lint-go`（17秒）、`GOFLAGS=-p=1 make test`（t14-test，61个包全过，991秒；Go测试`-count=1`不用缓存）、`make lint-web`（13秒）、`make knip`、`make test-web`（44秒）、`make e2e`（417/417，211秒）。turbo缓存如实记录：lint-web 50项任务中31项、test-web 16项中9项、e2e的前端构建10项全部由缓存回放——缓存键是任务输入，回放的是同一输入上次成功的输出；Go与e2e故事本身不经缓存。容器快照（每个门禁前后`docker ps -a`，只读）：只出现e2e会话自己的testcontainers收割容器，随后自行退出；重启后自动运行的其他项目容器未触碰。七门禁没有失败，也没有环境重跑。

检查点`634bc8eb`的CI（42159271084）server作业失败两项，都是测试与启动时周期任务的竞争，不是产品缺陷。`TestAttachmentCapacity/1001`的取消读取在`file_assets`上持ACCESS EXCLUSIVE锁，随后等应用连接池的占用归零；启动时的`asset.cleanup`仍在同一连接池中排在这把锁后，占用因而不归零（本地诊断t14-diag-capacity：被占用的连接正在执行`LockAssets`、等待表锁）。`TestRequestLockProbeWaitsForItsFixtureAfterUpstreamProject`持上游项目的行锁时，启动时的`issue.trim_description_versions`（逐个锁住所有未删项目）排在它后面并先拿到行的元组锁，请求的阻塞者因而是这个任务而不是测试的上游（CI日志中该后端的SQL为`LockIssueProject`；测试项目`archive_in`为0，排除了自动归档）。M5新增了第五个启动时运行的周期任务，测试配置只有两个周期worker，启动运行可能拖到测试世界建好之后。按本包既有写法（26个文件在加锁前等相关启动任务完成一次），两项测试在加锁前分别等`asset.cleanup`与`cleanup.purge_deleted`、`issue.trim_description_versions`完成；这些任务只由周期计划投递，测试中间隔1小时，完成一次后不会再出现。修复前本地5次失败1次（t14-ci-flakes-local），修复后20次全过（t14-ci-flakes-green）；去掉等待的变异在本机共25次未复现（t14-mut-capacity-nowait、t14-mut-capacity-nowait-full），竞争依赖时序，机理由诊断确认。该检查点的失败作业重跑一次（attempt 2），server通过。其他在应用世界里持项目行锁或`file_assets`表锁的锁测试理论上同样暴露，CI未观察到，写入M8交接。

P8预检提交`1f4da708`的CI（42126297643）e2e有1项失败、414项通过：F7（page）[s3]在点击“Add Project”时超时90秒。CI保留的轨迹显示页面自身的19个资源加载以`net::ERR_NETWORK_CHANGED`失败（运行器上发生网络变化；推测与并行的S3故事启动容器时Docker新建网络接口有关，未证实），应用停在启动画面。同一故事在T12、T14的全量和m5-profile三浏览器中都通过。该运行未重跑，其后提交的CI覆盖了同一输入范围。资源加载失败后应用停在启动画面，既不提示也不重试，这是观察到的行为，写入M8交接。

修复后的检查点`dd0eb2ff`（dd0eb2fffd05798813c426a67d973196acda04f8）的[分支CI 42162936213](https://github.com/open-nerve/NerveProject/actions/runs/42162936213)未通过，T14 Step 3未完成。attempt 1：server 127712757842与web 127712757601 success（两项锁测试的修复在CI上通过）；e2e 127717088966 416/417——F10（page）[s3]的收尾检查发现一条`net::ERR_NETWORK_CHANGED POST /api/v0/auth/refresh`。按T14 Step 2以同一输入重跑一次（attempt 2）：e2e 127724824125 416/417——F7（page）[s3]在点击“Add Project”时超时90秒，与预检CI的失败相同。三次失败的CI轨迹签名一致：某一时刻该浏览器中全部在途请求（包括发往127.0.0.1的页面脚本与API）同时以`ERR_NETWORK_CHANGED`失败（F10在05:01:28.18–.30，F7在05:30:25.5；F7两次都因此停在启动画面）。这是浏览器网络栈的整体事件，不是应用逻辑。两次检查点失败的时刻都在另一个worker的[s3]故事拆除其SeaweedFS容器前后约2.5秒以内（05:01:25.85与05:30:25.85结束）。每个[s3]故事在Docker默认bridge网络上启动并拆除自己的容器；Linux运行器上容器启停增删宿主网络接口，Chromium据此判定网络变化（推断，未在运行器上直接证实；macOS的Docker在虚拟机中，本地从未复现）。P8 T9只改了S3夹具的错误传递，容器生命周期自P3/P6未变，这是M5 S3 e2e夹具既有的暴露，不是P8引入；P8的四次完整e2e运行中出现三次（P7b的CI未见）。原检查点`634bc8eb`的重跑中server通过，e2e因新推送被CI的并发规则取消。按负责人规则（CI失败、重跑也不过即停下报告）停在此处：根因修复需要改共享的S3测试夹具，或调整CI重试策略，均待负责人决定。

负责人于2026-11-06选择修S3测试夹具。改为：`global-setup.ts`在任何浏览器启动前与PostgreSQL一起启动本次运行唯一的SeaweedFS服务，经环境变量交给测试worker；每个S3世界从这个服务取自己的桶（与每个世界取自己的数据库同一做法），关闭时清空并删除自己的桶，服务在全局拆除时才删除。S3夹具自身的生命周期测试（`smoke/s3-lifecycle.spec.ts`，其中会真实启停容器）移入`s3-fixture`项目，Chromium与Firefox项目依赖它，先于任何浏览器跑完。测试辅助程序新增两个动作：`delete-bucket`（分页清空后删除桶，桶已不存在视为已删）与`head-bucket`（只有确切404才是不存在）。SeaweedFS每个有数据的桶约占7个卷、删除桶即释放（本地探针：`volume.max=32`时第6个桶写入即失败，删除后40轮建桶/写入/删桶全部成功），`volume.max`由32改为256（约36个桶同时有数据）。这正是P3 spec“单服务多世界可共用服务但不共用桶/CORS/DB，多桶要预留实际卷容量”所允许的形态；桶、CORS与数据库仍为每个世界独有。证据：辅助程序Go测试先红后绿，两个变异（只清空第一页、任何检查失败都读成不存在）被杀死（t14s-mut-one-page、t14s-mut-any-absent-2）；夹具测试先红（t14s-red：世界仍自起容器，容器清单多出一个）后绿（t14s-green-1，5/5），变异“关闭时不删桶”被两项测试杀死（t14s-mut-close-keeps-buckets）；默认E2E 417/417（t14s-e2e，189秒，此前约3.5分钟）、`test:m5-s3` 3/3、`test:m5-slow` 2/2、`test:m5-profile` 42/42，运行后没有遗留的测试容器；`make lint-go` 0、e2e类型/lint/格式通过。本机是macOS（Docker在虚拟机中），无法复现原失败；修复是否消除CI上的网络变化，由准确HEAD的CI与其后两轮main CI观察。

共用S3服务的提交`2234d114`的[分支CI 42175455705](https://github.com/open-nerve/NerveProject/actions/runs/42175455705)中web通过，server失败一项、e2e因此跳过（共用S3的修复尚未在CI上运行）。失败的是`TestContentAssetRaces/reverse_references_cancel=true`：请求已持有低位资产行的FOR UPDATE、正等高位资产的测试锁，测试用行锁探针确认它的锁模式时，探针没有直接等待请求的事务，而是排在该行的元组锁后面，2秒内未得到预期等待。排在前面的是另一个等待这一行的会话：启动时的`asset.cleanup`在对象清点阶段对存储中每个对象的资产行执行`LockAssets … FOR UPDATE`，包括这两个刚上传的图片，迟到时正排在请求后面。这是P8在CI上第三次遇到同类竞争（启动时运行的任务拖到测试世界建好之后）；本包有15个文件对`file_assets`加锁而不等待启动清理。先试了在共用的世界构造`newIssueWorldOnRequests`中让所有世界都等五个启动任务各完成一次：bootstrap包在25分钟期限处超时（t14w-bootstrap，平时约11分钟）。实测原因是启动任务通常在应用启动后1–3秒才完成，晚于世界建好（单个世界不等0.2–1.3秒、等则1.4–3.5秒，t14w-time-nowait），三百多处世界各等一次就使包的时间翻倍，等于改CI预算，不可取。最终只让观察型世界等待（`newIssueWorldOnRequests`带请求追踪时，即`newObservedIssueWorld*`与四处直接调用，共31处）：这类世界专为把每个锁等待归到它的请求而建，正是不能容忍启动任务插队的测试；`TestRequestLockProbeWaitsForItsFixtureAfterUpstreamProject`（普通世界加观察路由）与`TestAttachmentCapacity`（`startImageAPIWorld`）各自调用同一个`waitForStartupRuns`。三项受影响测试各重复5次全过（t14w-focused-2），bootstrap包全量通过（t14w-bootstrap-2，818秒；T12/T14时约650秒，观察型世界在多项测试中按用例循环建立，实际等待次数远多于31处），仍在CI的20分钟包期限与25分钟作业期限内；`make lint-go` 0。本机无法复现这次CI上的迟到，机理由CI日志与清点阶段的源码对应确认。

T15（整分支审查）：冻结候选`634bc8eb`，一次fresh只读审查（`fable`），报告见[re-52](../../../reviews/re-52-M5P8整体审核.md)。结论Ready to merge: Yes，没有Critical或Important；五项Review Focus全部PASS（第5项附Minor 1）。主控按效果复核后不需要修复轮：上面T10那句关于`Purge…`页的措辞按实际观测改写；其余Minor（演练追踪器对单语句分页只记结果行、`ImageTransfers`的中止监听不移除、`verify()`多ID部分失败时对话框混乱、未完成时恢复点击无反馈、404后的标题计数、启动时已非当前节点的`queued`）作为deferred记入台账与负责人报告；declined-to-judge十项逐条维持既有裁定。冻结候选的CI首次server失败两项测试，由T14 Step 2按根因修复（`dd0eb2ff`），不触及审查过的产品代码。
