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

