# P7 预检裁定：找不到身份的未完成文件阻止草稿发布

> 2026-11-04，负责人已明确选择 **A：补恢复入口，分 P7a/P7b**。本次只修订文档并完成P7a预检，不实施P7。原上报提交88d5ae30保留发现时事实；本页基线为已合入main的 `963636757fe5c5a8c10d5781b43d1481ba6f32a2`，产品树与全绿P6候选8e46ab55一致。P6 main CI/归档独立记录，不能由此页代签。

## 1. 用户会遇到什么

用户编辑一份草稿，上传图片时网络中断，或复制图片时存储返回失败。用户取消这个图片节点、重新打开草稿甚至重新上传成功后，原草稿仍可能报“文件尚未完成”，无法发布。

原因是服务器已经保存了一条未完成资产，但错误回答没有交付它的 ID。现有页面和附件列表都找不到这条记录，无法指名删除。原七天未绑定件清理最终可回收，但需要严格超过期限且清理实际执行；不是用户当场可用的恢复方式。

这不会损坏 P6 的资料上传：资料绑定不要求宿主的全部准备件完成。本问题在 P7 首次把同一协议接入草稿发布时成为发布阻塞。

## 2. 已核实的证据及边界

| 事实 | 源码证据 |
|---|---|
| 发布枚举草稿全部未删资产，不只正文引用 | `server/internal/modules/asset/adapter/postgres/queries/content.sql:16`，`DraftAssetIDs` |
| 任一活资产未完成便拒绝整个发布 | `server/internal/modules/asset/app/draft_asset_rules.go:28`，`q.Publish && !isPublished` |
| 申请先提交记录再返回新ID/上传能力 | `server/internal/modules/asset/app/prepare.go:48–80`；`api/modules/asset.yaml` prepare描述 |
| 复制先提交独立pending目标，再执行外部封存；失败仅返回error | `server/internal/modules/asset/app/duplicate.go:42–85、102–120` |
| 附件列表只列uploaded且bound，不能找到pending准备件 | `server/internal/modules/asset/adapter/postgres/queries/attachments.sql:1–11` |
| 已知ID可显式删除私有准备件；未知ID没有发现入口 | `api/modules/asset.yaml` GET/DELETE按id及完整paths；`asset/app/delete.go` |
| 原资料会话取消/重新选文件清空prepared；重新申请也弃旧ID | `web/apps/web/core/store/asset/upload-session.ts:116–121、221–244` |

一次 fresh builder 独立核得相同结论，原报告见 `.superpowers/sdd/P7-content-assets/builder-questions.md`；正式预检报告记录静态范围及限制。

主控新运行既有 `TestDraftAssetTransferPublicationStates`（`go test -p 1 -count=1 -run '^TestDraftAssetTransferPublicationStates$' -v ./internal/modules/asset/app`），三个子项全部通过：pending能随项目移动、pending阻止发布、已发布final携落败candidate仍允许发布。3.298秒，1707个Go输入前后hash一致，无Docker、无产品修改。raw/receipt/inputs保存在本worktree `.superpowers/sdd/P7-content-assets/draft-pending-existing-contract*`。

这次只执行了既有单元合同，**没有新执行“HTTP回包丢失→页面发布失败”的端到端复现**；未知ID路径是上述实际事务/响应/查询合同的静态推导。P5已有HTTP测试 `TestDraftAssetPublishPendingBlocksWholeGraph` 明确通过指名DELETE移除pending才允许发布，本次只读该测试，不冒称重新运行。

## 3. 推荐 A：补作者可发现、可明确丢弃准备件的恢复入口

保留P5“全部活资产完成才发布”的安全边界，补窄查询 `GET /api/v0/drafts/{draft_id}/asset-preparations`：仅当前作者、活工作区/草稿可访问，沿既有宿主授权和读快照；只返回该草稿尚未绑定且未删除的资产元数据，pending/uploaded均包含，分页有界，不返回对象key或签名URL，不枚举他人准备件。精确DTO/游标/拒绝边界现已固定在[P7a spec](../specs/P7a-draft-preparations.md)，API源及生成物等获准实施才修改，不能假定已存在。

页面在用户明确恢复时读取准备件，显示文件名、状态与时间，用户可指名丢弃不需要的未完成件，复用既有DELETE软删除及对象恢复记录。不能按文件名猜、静默批量删除或在卸载时后台DELETE；其他标签页仍在上传的文件也不能自动丢弃。已知ID显式取消的恢复也经同一路径，未知ID/失败复制不再漏掉。

若请求、取消与发布交错，仍以服务器锁下重验为准：删除后迟到complete不能复活；新准备件在检查后出现可继续拒绝发布，页面明确重读，而不是无限自动重试。删除保留原物理清理期限和候选记录，不改变NO ACTION、P5迁移集合、版本历史或S3能力边界。

**为什么推荐：** 根因是“有恢复记录却无法让合法作者发现并处理”，补齐这个能力可同时覆盖已知ID取消、prepare响应丢失、明确失败duplicate及跨刷新恢复。单改前端保存已知ID只能覆盖其中一部分；发布时自动忽略pending则改变既有数据完整性承诺。

**代价与范围：** 新增只读API/生成类型、asset SQL与宿主端口组合测试、权限/分页/并发回归及恢复UI；不加表/依赖/平行权限观察器。P7原计划已16Task，建议拆为 **P7a草稿准备件恢复协议**（小阶段，先API与安全合同），再 **P7b内容文件页面**（承接当前P7草案）；阶段各自预检/实现/整审/noff/main CI，不挤掉验证。

## 4. 其他可行方向

**B：修改草稿发布合同，在发布事务中显式放弃未完成准备件。** 需要同时定义哪些件可放弃、用户如何确认、其他标签页正在传输时的结果、迟到complete与候选回收，并改P5原“pending一律阻止发布”的合同及测试。好处是少一个恢复列表；代价是发布动作可能舍弃用户仍想保留的文件，跨模块事务/数据语义变更更大。本页不默认批准此方向。

继续原协议、只修已知ID或等七天GC都没有覆盖未知ID即时恢复，不作为“根因已解决”的推荐方案。

## 5. 必须补齐的验收

已按批准A同步M5设计/P5接收合同/P7a与P7b spec、plan；当前仅预检，暂不实施。P7a验证真实API发现未知ID并恢复发布，P7b补真实页面取消/选择/刷新/多标签。至少覆盖申请提交后丢回包、复制提交目标后存储失败、无项目草稿、作者与不可见/不存在同形、分页全量、显式丢弃与complete/发布交错、删后旧PUT不复活、已绑定历史图不误删、邻域草稿不受影响，不能只测已知ID的DELETE。

## 6. 另一个局部预检问题

现有普通工作项“创建副本”最后应返回Issue，现有草稿副本应返回Draft；不能把二者都停在新草稿。当前草案只写“含图先取得真实目标草稿”，没有完整规定发布时机与部分失败/取消后的恢复。

主控采用以下局部方向，已写入P7b合同/T10：打开副本弹窗不写数据库；用户明确提交含图副本时才创建一次真实目标草稿，逐图复制并保存UUID；普通副本在同一次用户提交流程中调用既有publish，只有拿到Issue才成功关闭/回调，草稿副本止于Draft。部分失败保留并明确展示同一目标草稿与已完成映射，后续重试复用该目标；用户取消不假称未产生持久记录，提供回到保留草稿的路径。未知创建/发布结果先核验，不自动再建。无图副本保持原直接创建路径。

这是当前API可支持的局部编排，不新增普通工作项准备scope；实现会核实际两个quick-action入口、工作区/项目权限与原字段/错误语义。它同样依赖第3节解决失败复制残留pending，故当前不实施。若后续发现其结果与既有授权无法兼容，再保存实证上报，不能临时放宽权限。

## 7. 决定后的状态

原P7预检的 **Needs Decision** 在负责人选择A后关闭为“已裁定并拆分”，不改写历史原报告。P7a的新协议做一次独立预检，P7b保留已同步恢复责任的延后草案，正式基线/预检待P7a集成。所有产品验收仍未打勾，P6门禁独立继续；当前无API/产品实现或发布规则放宽。当前worktree/分支改为codex/m5-p7a，原P7 ignored台账一并保留至阶段归档。
