# M5：文件

> 状态：P1–P5已完成集成归档。P5最终main f5f4ce94的CI41772399410全绿，三组分支/worktree及5053件证据归档核验完成。P6的PC1方案A已批准，最终基线预检通过，T1–T5完成，T6接续；P6/M5总体仍未完成。见[P5阶段记录](reviews/P5-issue-assets-review.md)和[P6阶段记录](reviews/P6-profile-upload-review.md)。
> 起点：`main@3db22f06ad9e82a483c05c8078801af00a5cae02`（M4 最终 main CI 41397218533，三个作业成功）。
> 设计、P1–P4分支均已合入并归档；P5已noff合入main，因一次重跑仍未全绿，按负责人特定停点等待预算裁定；当前文档工作分支codex/m5-p6。负责人2026-11-02已授权按既有节奏连续推进，常规设计、预检、实现、deep-review、`--no-ff`集成、CI和归档无需逐次询问。重大风险、架构回归或关键分歧，先提交已完成工作，再上报。

## 1. 目标、范围与输入

M5 使头像、个人封面、工作区图标、项目封面、工作项附件和编辑器图片经真实文件服务工作。默认仍是一个 Go 进程加 PostgreSQL，文件存在本机磁盘；选择 S3 时前端直传对象存储。业务数据只保存资产身份，不保存会过期的签名地址或前端构建路径。

范围包含上传申请、PUT、确认、读取、复制、软删除、未完成上传清理和 60 天物理清理；保留 M4 的权限、事务、内容标识、会话代次与删除图不变式。不恢复 Page、企业版资源、Unsplash，不借文件接线开放尚未实现的 M6/M7 接口。

前置依据是[总体设计 §6.7](../v0-design.md#67-后台任务与文件存储)、[执行方法](../claude_to_gpt/02-method.md)、[硬规则](../claude_to_gpt/03-rules-and-pitfalls.md)。全部输入交接及逐条落点见 §13；原文保留，不把过时的代码数量当成当前事实。

### 1.1 设计起点事实

以下为 M5 设计起点快照。P1 已实现 asset 本地存储与 raw 传输，当前整审/CI/集成状态见 [P1 review](reviews/P1-local-storage-review.md)；file_assets 与业务入口仍由后续阶段承担。

- 尚无 `asset` 后端模块和 `file_assets` 迁移。用户、工作区、项目的 URL 字段存在但没有真实文件引用。
- `FileService` / `FileUploadService` 仍调用旧 Plane 路由和 multipart POST。当前编辑器资产 store、编辑器配置、旧评论创建和 inbox 创建处仍有消费者；实现时重新扫描，不沿用旧交接的实例数量。
- `EFileAssetType` 目前只有 `COMMENT_DESCRIPTION`、`ISSUE_ATTACHMENT`、`ISSUE_DESCRIPTION`、`DRAFT_ISSUE_DESCRIPTION` 四种。交接中的五种、六种是历史数量。
- 编辑器的 `image-component[src]` 已以资产 ID 表示内部图片，运行时再解析显示地址；M4 页面尚未开放图片和附件区。M5 延续这个身份模型。
- `PublishDraft.Execute` 已将创建工作项和软删草稿放在同一事务内；M5 在此边界补资产迁移，不在前端发布后搬迁。
- `plane/.../asset/v2.py` 的直接标记已上传、通用 workspace 权限、宽松 restore 和批量绑定只作行为参考；不继承它们缺少实物验证或细粒度授权的实现。

## 2. 用户故事

每条有 API/存储或页面证据。`F` 是 M5 的故事编号，不替换 M4 已完成的 `I` / `C` 编号。

| 编号 | 行为及有区分度的拒绝路径 |
|---|---|
| F1 | 申请、真实 PUT、确认、读取；文件字节相同，重复确认不重复绑定、不增加动态 |
| F2 | 大于 1 MiB 且传输超过普通 API 的 15 秒仍能成功；超限、长度不符、类型伪造、失效或篡改签名被拒绝 |
| F3 | 确认后重放旧 PUT、确认并发、对象变化、崩溃重试不能替换已发布字节；失败对象可清理 |
| F4 | local 和真实 S3 兼容服务通过同一契约；S3 CORS、浏览器自动发送 Content-Length、签名请求头真实验证 |
| F5 | 个人资料及引导上传头像/封面，替换、移除、拒绝、重新登录和旧请求迟到均保持正确账号归属 |
| F6 | 管理员更换工作区图标；非管理员不能写，成员列表头像来自有权读取的资料 |
| F7 | 新建项目上传或选择封面，一次创建请求绑定最终封面；失败不创建半成品，随机选择保存稳定编号 |
| F8 | 工作项附件上传、列表、下载、删除；计数、排序和显示属性来自真实活附件 |
| F9 | 评论图片/附件跟随评论及工作项权限；创建前准备、发送失败后重试、评论删除均不误绑其他评论 |
| F10 | 草稿图片/附件保存、刷新、换项目、发布；并发双发布只有一份工作项，任一迁移失败全部回滚 |
| F11 | 编辑器上传、粘贴、复制、撤销重做与重新打开；复制源不可见和不存在同样 404，目标无写权不复制 |
| F12 | 换图/调序影响内容标识，不能利用 skip_activity 跳过动态/版本；只补合法 ID 保留 M4 迁移语义 |
| F13 | 访客仅自己的工作项、私有项目、撤权、归档、软删除，以及祖先删除期间的资产读取/写入边界 |
| F14 | 删除工作项/评论/草稿/项目/工作区立即阻止新的资产访问授权，异步任务保留原 At/By、失败可重试且不误伤邻域 |
| F15 | 未完成上传、未绑定准备件、失败复制/确认、超 60 天墓碑及孤立对象得到有限批次清理；对象删除失败保留重试依据 |
| F16 | 换账号、工作区、项目、工作项、Back/Forward 与撤权使请求/进度/弹窗失效，迟到回包不写入新宿主 |
| F17 | 上传/删除拒绝可被读屏读出，键盘焦点完整，请求中不能重复提交或意外关闭，成功后正确返回焦点 |
| F18 | 无外部存储配置的演示实例可用；无旧文件路由、死服务包、到期关键词豁免或未经说明的遗留成员 |

## 3. 业务模型与权限

### 3.1 资产、准备件和对象的区别

`file_assets` 是一个业务资产，拥有稳定 UUID、申请者、用途、可选父范围、声明大小/类型及审计。实际存储对象的随机 key 由服务端产生，文件名仅作展示与下载名称，不参与路径。

资产状态为 `pending → uploaded → deleted`。`uploaded` 表示实物已验证且发布对象已经封存；不表示已经挂入业务字段。新项目/新评论尚无 ID 时可以产生**准备件**：必须有申请者、用途和已有工作区/项目/工作项边界，只有申请者可读写，不能进入成员可见列表。宿主创建在同一事务里绑定它，不能跨用途、跨申请者或跨范围挪用。

业务父字段绑定与实物上传完成分开。头像/封面更换经所属模块的更新接口提交；失败保持原引用。资产已绑定后不提供任意改 owner 的通用 API。草稿发布是唯一的草稿到工作项整体迁移入口。

### 3.2 用途与授权

资源用途使用生成的枚举：`USER_AVATAR`、`USER_COVER`、`WORKSPACE_LOGO`、`PROJECT_COVER`、`ISSUE_ATTACHMENT`、`ISSUE_DESCRIPTION`、`COMMENT_DESCRIPTION`、`COMMENT_ATTACHMENT`、`DRAFT_ISSUE_DESCRIPTION`、`DRAFT_ISSUE_ATTACHMENT`。当前四种值与新增值一起由 OpenAPI 生成；不保留手工镜像。

| 用途 | 范围与读写规则 |
|---|---|
| 用户头像/封面 | 写自己的资料；已绑定头像/封面的解析只由有权返回该用户资料的接口进行，准备件仅申请者 |
| 工作区图标 | 工作区管理员写，活成员随工作区资料读；工作区删除后不能再签发 |
| 项目封面 | 沿用项目更新权限及可见性；准备件须具备工作区创建项目权限 |
| 工作项附件/描述 | 沿用工作项读取/更新的授权次序；guest 不能靠 project membership 绕过创建者边界 |
| 评论图片/附件 | 读随可见工作项，写随评论创建/编辑/删除规则；未发表准备件仅申请者 |
| 草稿附件/描述 | 仅草稿作者，遵守 workspace 与未删除项目边界；发布重新执行目标项目创建权限 |

认证失败 401；资源不可见、已删或不存在统一 404；先证明可见再返回明确写权限不足 403。归档项目/工作项按现有领域写限制拒绝，读取允许。实体类型不能由客户端伪造为“静态公共”类型绕过权限。文件 key、UUID 和旧签名 URL 都不作为新的业务授权依据。

申请、确认、绑定、复制、删除均重新验证权限。复制先源读取、再目标写入；源不可读与不存在使用同一响应形状，不做跨工作区公开探测。目标合法不代表源合法。

### 3.3 签名地址的边界

数据库和 HTML 保存 UUID，不保存签名 URL。浏览器读取时通过已认证的业务 API 得到短期 URL；返回 `expires_at`。到期时重新授权后解析，运行时替换 URL 不触发业务内容保存。

按总体设计，读 URL 是短期 bearer capability：**已签发的 S3 URL 在到期前可能继续有效**，权限撤回阻止新签发，页面立即清空旧范围。不能把 M4 的页面/请求失效写成 S3 存储能即时收回已发出的 URL。读 URL 不放入日志、动态或持久化 HTML；业务 JSON 仍遵守既有 no-store，原始文件响应采用 private 且不超过 URL 有效期的缓存（local按读取时刻计算剩余max-age；S3为private,no-store，避免签发时固化的max-age在晚取件时越过期限）；页面不把能力 URL 分享为永久链接。需要即时收回每次字节读取将改变既定 S3 直读架构，出现这种新要求须上报。

读 URL 的到期时间取下一个 UTC 整点，最短不足 5 分钟时取再下一个整点，因此最长 65 分钟；签名时间使用当前时间，不伪造 SDK 时钟。页面按 `expires_at` 复用结果，不能仅凭取整宣称每次 S3 签名字符串相同。上传授权默认 15 分钟，不按小时取整。

## 4. 数据、引用和删除图

### 4.1 迁移所有权

以 `tools/plane-schema/plane-v1.4.2-schema.sql` 的 `file_assets` 为起点。asset 迁移拥有 `file_assets`，保留 UUID、审计、attributes、用途、父引用、上传/删除标记及 storage_metadata 的业务作用。差异登记到 `plane-diff.md`：删除 page/外部导入字段及冗余 entity_identifier；大小改精确非负 bigint；key 使用内部字段而不是 Django FileField；增加上传期限及绑定状态的明确约束。`is_deleted` 与 `deleted_at` 若保留兼容双字段，必须 DB CHECK 一致，不允许两个事实来源。

用途和父字段组合有 CHECK，父 ID 由服务端从授权结果填写。资产表只引用已存在的用户/工作区/项目/工作项/评论/草稿，不反向跨模块 JOIN。所属模块查询其父事实，asset 只查询自己的表。

沿用 `file_assets.storage_metadata` 保存带 schema_version 的有界存储状态：一个 staging、一个 final、最多三个未回收的 seal/copy 候选；每项含随机 key、阶段、期限、字节数/ETag/校验摘要。Go 使用具名结构，SQL 约束对象形状和候选数，不接收客户端 metadata，不堆无界历史数组。对象 key 只能由该资产生成，所有状态修改持有同一资产行锁；达到候选上限先回收或明确返回暂忙，不能继续追加。它是存储恢复记录，不增加总体设计 §5.2 之外的业务表或通用 outbox。上传先提交 staging 状态再返回签名，seal/copy 先提交候选状态再写外部存储。

后续加列分别归 owner：`identity.users.avatar_asset_id/cover_image_asset_id`、`workspace.workspaces.logo_asset_id`、`project.projects.cover_image_asset_id`。外键目标是 asset；所有者引用被替换/移除后旧资产才可回收。项目另有稳定的 `cover_builtin_id`，与 `cover_image_asset_id` 互斥；可均为空。迁移顺序、sqlc scope 和 Plane 差异同步验证。

### 4.2 首张表就延伸删除覆盖

asset 表合入的同一阶段更新真实删除 fixture、目录/外键守卫、`furnishIssueRows`、工作项及父范围删除故事。不能留到页面阶段才补；不能给新表加守卫豁免。

同步删除工作项/评论/草稿时，asset 在原事务处理其拥有的资产行；项目/工作区的同步删除只使父范围立即不可见并投递 asset 自己的有界任务。普通工作项子树删除已有逐项 `issue.deleted`；父级批量删除不依赖那些事件。`issue/adapter/postgres/delete_rows.go` 继续只操作 issue 表，由删除协作边界通知 asset，不向它塞 asset SQL。

事件任务沿用原 At/By，重试不覆写更早删除审计。物理清理先移除外部对象并在锁下确认 metadata 已无待清理 key，然后移除无引用的资产墓碑，最后才允许相应父行被 purge。失败保留记录，下轮重试；不能先删 DB 行再忘记对象。跨 owner 的 purge 顺序由 bootstrap 明列，循环引用通过所属模块清空失效引用解除，不关闭外键。

**2026-11-02 负责人裁定 A（T4父分页/持久游标与T5真实资产保护、已知对象恢复及联合目录已实施）：** `file_assets` 的业务父引用采用即时 `NO ACTION`，防止任一直达或传递 CASCADE 抹掉恢复记录。asset 对象/墓碑清理先于父 owner；父清理须有界识别和跳过受阻候选，仍推进无关候选。覆盖 comment/draft/issue、issue 自引用、states→issues、project/workspace 全部路径；不把首批删除 0 行误当候选耗尽。守卫精确验证受保护 FK、清理者、父候选协议和顺序，原任意 RESTRICT/NO ACTION、漏表/漏 FK 负例仍拒绝；增加移除数据库保护/跳过依赖验证的反例。不是把 NO ACTION 直接列为安全 action。

收益是协议遗漏或对象故障不会丢掉唯一恢复依据；代价是相关父墓碑在对象故障时继续保留，不能靠 CASCADE 按时强删。所有对象 I/O 在事务外，父 owner 小端口与 asset 自有 SQL 在 bootstrap 接合；完整协议、锁图和前进证明在 [P2 spec](specs/P2-assets.md) 及其 access/recovery/protected-purge 子合同固定，完整预检记录见 [re-34](../../reviews/re-34-M5P2预检.md)。[问题、探针与已批准方案](reviews/P2-purge-contract-decision.md)保留替代 C（全锁内 owner 协议，无数据库兜底）和未采纳理由。


**P4引用合同（实现、完整门禁、整审、准确分支/main CI及归档清理均完成，证据见P4阶段记录§22）：** 三owner分别加入四个图片引用即时NO ACTION FK与索引，project另有cover-01..29互斥字段。owner事务拥有账号首锁和W/P授权、收齐old/new资产集合；asset端口只锁/改自有行，最后锁后一个At。新项目先锁/校验准备件，INSERT一次写最终cover，再迁移asset范围，任何失败全部回滚。所有返回资料/组织/项目的操作统一在现有授权边界内CPU签发，公开邀请只沿原MAC链接证明固定logo，不开放任意资产读取；成员头像不在W锁后锁目标账号。批量读取保留整集合，资产按全局UUID顺序、每SQL至多1000条；T12冷统计1000/1001引用最终race实测611/560ms，14/15条SQL，保持零存储I/O。

反向引用清理页最多min(batch,1000)：候选先读，identity→workspace→project各自有序锁行并按当前引用条件解除，再由asset复核cutoff/metadata并DELETE，一次提交，删0也按扫描位置前进。保留原asset→issue→project→workspace→identity顺序、父保护及未知FK负例；目录逐一核四条反向边和具名解除者，无通用NO ACTION豁免。完整边界、代价和验收见[P4主spec](specs/P4-profile-images.md)、[绑定/读取](specs/P4-image-access.md)、[引用清理](specs/P4-reference-purge.md)。

## 5. 存储契约与确认协议

### 5.1 消费方端口

`asset/app` 声明小接口，分为上传签发、对象检查/封存、读取签发、复制、删除及清单。domain 仅有值和规则，不能依赖 SDK、文件系统、HTTP 或数据库。local/s3 都在 `asset/adapter`；bootstrap 显式选择，配置错误启动即失败，不运行时悄悄回退。

共享契约测试经实际 HTTP PUT/GET 检验，不用一个返回成功的 S3 假实现冒充适配器证明。必须覆盖签名方法、长度、类型、过期、篡改、并发、条件复制、封存不可改、读取/删除幂等、缺失与后端故障的区分。

### 5.2 三步上传与不可变发布

1. **申请**：普通 JSON API 在事务里授权并创建 pending 资产和 staging 对象记录；写入声明大小/规范化 MIME/期限。提交后返回资产 ID 与 `{method: PUT, url, headers, expires_at}`；headers为按名称排序、无重复名称的`[{name,value}]`，每项对象封闭，调用者逐项设置请求头，内部存储端口仍用字典。申请超过限制直接拒绝，不能像 Plane 用 min 悄悄截小。
2. **传输**：前端只使用返回的精确目的地址和必要头发送 File 字节，不附加 API bearer token 或 cookies；不能把任意调用方 URL交给带认证的 API client。大小和类型签入授权；Content-Length 由浏览器根据 File 自动设置，不尝试设置浏览器禁用头。
3. **确认准备**：重新授权，确认 pending 且未进入清理，建立随机的 seal 候选 key 记录。外部 I/O 期间不持有业务事务、行锁或数据库连接。
4. **实物检查与封存**：检查 staging 存在、精确大小、声明类型及内容约束。以 ETag/版本条件把相同对象复制到本次独有的 seal key；再检查封存结果。图片做受限头部/尺寸/格式验证，附件下载不依赖客户端 MIME 作为安全依据。
5. **提交结果**：再次锁父范围、资产并重新授权；仅一名胜者把该 seal 对象登记为最终对象并置 uploaded。同一资产重复确认返回同一结果，竞争失败的候选进入清理；任意客户端可重放的 PUT 只指向 staging，不能写最终对象。

不能使用所有确认共享的固定 seal key，否则先输掉 DB 竞争的慢复制仍可能覆盖胜者。存储超时/DB 提交失败/进程退出都保留候选记录；重试检查已有结果，不凭“请求超时”假定对象未写入。

本协议不宣称数据库和对象存储有分布式原子事务。对业务可见性用最终 DB 提交控制，对外部残留用对象记录和对账回收控制。禁止签发 staging/candidate 的公开读取地址。

### 5.3 local

默认数据目录下的独立资产根目录，服务生成 key 并使用根约束的文件系统操作，拒绝 traversal、编码路径变体、符号链接逃逸、目录/设备文件。写入独立临时对象目录（字节文件和元数据），验证长度后原子发布；不将半文件暴露为成功对象。local 对同 key 的第二次 PUT 返回冲突，不覆写已经完整的对象；S3 staging 可被重放覆写，两者共同合同只承诺最终 seal 不变，客户端不能依赖 staging 可以重写。失败重传重新申请 key；响应丢失可先确认已有上传。签名绑定 method、key、size、type、expiry、用途和版本，用恒定时间比较。

复用 `identity.Keys.MAC(purpose)` 已提供的用途隔离能力，bootstrap 为上传/读取分别派生 `asset-upload` / `asset-read` 的 MAC，asset 只接收 Tag/Verify 小接口，不 import identity，也不接触原始密钥。生产继续要求既有持久 signing key；同一密钥重启仍能验证，dev/test 使用既有临时密钥时重启使旧 URL 失效。轮换签名密钥同步使这些 URL 失效，不新增另一套密钥文件和发现逻辑。

原始传输端点 `PUT /static/local/{key}`、`GET/HEAD /static/local/{key}` 在普通 JSON API 之外，仍使用 request ID、安全头、日志与有限并发/速率限制；不能以这一例外跳过整个程序路由清单核对。每个请求用 ResponseController 设置读写期限，预算由声明大小、配置最低传输速度和固定余量计算，设置明确最大值；超额体由 MaxBytesReader 拒绝，空闲/总期限到达结束连接。其他 API 默认的 1 MiB/15 秒和连接全局默认保持不变。

传输空闲以活跃方向的实际字节进度判断：上传读侧、下载写侧；GET/HEAD 用于检测断连的后台读只设总期限，不能因没有新请求数据打断健康下载。期限使用真实系统单调时钟，取消时实际中断 socket，进度续期不能覆盖取消。`size_limit` 限制新上传；旧对象读取按可信大小计算预算，并受显式最长传输时限约束。

读取有正确 Content-Type、Content-Length、Content-Disposition 和 nosniff。普通附件一律 attachment；inline 仅限服务确认的 JPEG/PNG/WebP/GIF。支持标准 HEAD、单段 Range/If-Range 与过期检查，拒绝不支持的多范围；每次先验证授权签名再处理条件缓存请求。含签名 query 的完整地址不记日志，错误不泄露磁盘路径。

`/static` 路径树明确拒绝 ServeMux 的 `..`/重复斜杠重定向，避免签名 query 被带到其他端点；未知路径返回 404 no-store Problem。匹配具体端点后的无效签名仍为统一 403。平台提供按具名路径树声明的路由策略，逐段识别规范化中途进入再离开的路径，普通 API/页面路径保持原规则。raw 路由与 asset 专用错误码登记在 `api/storage-local.json`，全程序路由、源码错误码及中英文消息按精确目录验证，不进入业务 OpenAPI 的 bearer/publicOperations 合同。

`/static` 是 M3 已预留的文件根：static 从 reserved 转入 server，保留名字总集合不变，storage 工作区不迁移。构建/public 不占该树，`/assets` 继续专属构建资源；Vite 的具名开发传输插件原样流式转发 `/static`，拥有取消/关闭和安全错误出口，不让默认代理日志记录签名 query。Go 继续负责验签与预算；具体合同见 P1 spec §2.7。

### 5.4 s3 与兼容性

采用 AWS SDK for Go v2：root/aws v1.47.5、credentials v1.20.13、service/s3 v1.114.6及其9个必须间接模块（[P3 spec §2](specs/P3-s3-storage.md#2-结构依赖和资源所有权)列全）。不手写SigV4、不引入aws/config/默认凭据发现链。只从Nerve已验证的files.s3配置注入显式凭据、region、bucket、endpoint、path_style和专属prefix；秘密通过环境或*_file，日志及错误遮盖。

P3真实服务固定[SeaweedFS4.52](https://github.com/seaweedfs/seaweedfs/releases/tag/4.52)，镜像`chrislusf/seaweedfs@sha256:57877e32bb90073b37dc109b438ba66e84fd5ed9ced5f5c04f56444ef243ce59`；源/各平台摘要及7组协议探针见P3 spec/re-36。不代表其他品牌或AWS云已实测。生产默认仍local，不增加默认对象服务容器；只在测试自建/清理随机凭据、桶、数据与端口，已有Docker/容器不动。

SDK首次实际消费时再改server模块锁；测试专用server/tools/s3fixture可消费同三SDK版本及必要间接模块，提供Node测试桶/CORS/实物断言，不注册到nerve CLI、不引入JS SDK。既有testcontainers-go v0.44.0及Node testcontainers12.1.0仅按实际消费转直接依赖，不升级。除此以外无新增依赖授权。P2的官方x/image/webp v0.47.0继续保持；前端不增加上传框架。

S3来源主机使用DNS或IPv4；浏览器CSP不能精确允许IPv6字面量，因此配置提前拒绝字面量，可使用解析到IPv6的DNS，不改变服务端监听能力。只支持从未启用版本控制的普通私有桶，不支持Enabled/Suspended/Object Lock/Express/access point；启动在operation_timeout总期限内只读HeadBucket/GetBucketVersioning，未知或拒绝即失败，不写生产探针。部署保持模式、prefix独占且无外部删除活文件规则；否则普通Delete可能仅留下delete marker而失去旧字节恢复依据。版本化协议和跨后端迁移未实现，不能直接改配置当迁移或自动fallback。

真实合同要求长度/类型绑定的SigV4 PUT、HEAD/GET/DELETE、源CopySourceIfMatch加目的IfNoneMatch=*的原子COPY及有限有序ListObjectsV2；不退无条件复制。官方HTTPPresignerV4扩展点仅在正式签名前按真实signingTime计算精确期限，随后由官方Signer签名；不伪造时钟或改签后host。错误/200内嵌错误、迟到实物、双时钟和分页恢复沿P2协议，不能把请求取消当服务端已停止。 协议正文在SDK解析前有界（普通64KiB，清单64KiB+limit×16KiB），成功Open仍流式；错误读取/解析失败保持后端故障，不凭HTTP状态确认缺对象或冲突。

验证后先统一端点主机名大小写/十进制端口并省略默认端口，拒绝浏览器另释的非标准数字主机，再由官方EndpointResolverV2计算实际bucket origin；不改签后URL，独立transport只准该origin、无环境代理/redirect、系统TLS、SDK RetryMaxAttempts=1；不改credential-free link-preview outbound客户端。bootstrap选择驱动与资源所有权，asset app小端口和domain纯净；S3不注册local raw handler，/static保留树与完整路由守卫仍按两driver验证。

浏览器CORS只允许实例frontend origin、PUT/GET/HEAD、Content-Type和明确暴露头；CSP只加实际存储origin到img-src/connect-src，不用通配。F4必须真实三浏览器File/Blob验证长度、类型、跨origin/匿名/CSP及实物，不以Go HTTP替代。详细配置上限、秘密文件规则、部署约束、测试/容量和代价见[P3 spec](specs/P3-s3-storage.md)与[plan](plans/P3-s3-storage.md)。

## 6. API 与错误

先写 `api/modules/asset.yaml` 及全程序契约，再生成 Go/TS。所有业务 API 使用 `/api/v0`、严格请求体和 Problem。初始路由设计：

| 操作 | 语义 |
|---|---|
| `POST /api/v0/assets` | 用途、宿主/准备范围、name、size、content_type；返回 201、id 和 upload |
| `POST /api/v0/assets/{id}/complete` | 验证上传，返回资产 DTO；重复为同一资产，不重复业务写 |
| `GET /api/v0/assets/{id}` | 授权后返回元数据及上传/绑定状态，不泄露内部 key |
| `POST /api/v0/assets/{id}/duplicate` | 已认证源读取 + 目标写；建立独立对象和资产，失败记录供清理 |
| `GET /api/v0/assets/{id}/content` | 已认证读取，返回 `{url, expires_at}`；不通过 GET 修改资产/使用计数 |
| `DELETE /api/v0/assets/{id}` | 可删除的准备件/附件软删；已绑定的头像/封面须先经 owner 更新引用 |
| `GET /api/v0/issues/{id}/attachments` | 仅活附件，游标分页；在 issue 授权快照内调用 asset 的列表端口 |
| `GET /api/v0/drafts/{id}/attachments` | 作者的活草稿附件；最终路径须与已生成 draft 路由命名一致 |

评论的附件作为评论 DTO 的受限列表或独立分页端点，由 P5 预检按现有评论分页上限选择并写死，不能出现一次加载无界文件集合。新评论提交携带已完成准备资产 ID；新项目/资料/工作区更新携带 asset ID（或 builtin ID），不接受任意 URL。

统一错误包括结构非法 400、认证 401、可见但拒绝 403、不可见 404、状态冲突 409、体超限 413、类型不支持 415、业务字段非法 422、限流 429、存储暂不可用 503；保持已有 bodyshape/shared.Invalid 的 400/422 分工。具体 code 由资产域具名并在契约的 x-problem-codes、中文/英文错误翻译及全程序负例同时覆盖。不存在对象与存储连接失败不能都映成 404；签名失效与资产权限丢失不能假报上传成功。

普通文件允许任意非空字节作为附件，最大值用 `files.size_limit`，不按扩展名认可可执行内容；下载强制 attachment、规范化安全文件名。新文件图片用途只允许栅格格式，不接受 SVG/HTML 或 data/blob 作为文件来源。M4 清洗器已保留安全的远程 src；M5 不静默抹掉存量 HTML，也不把保留值当文件授权。只有规范 UUID 才进入资产解析；非资产 src 在页面显示不可用占位、不发远程请求、不由服务端自动抓取，原值仍参与有序内容标识。新上传入口只生成资产 ID，清洗的既有安全 URL 合同不因文件接入变成任意拒绝。编辑器不再接收自己无法插入的非图片 MIME，普通附件走附件区，不吞掉文件拖放。

## 7. 跨模块协作与锁

沿用小接口由使用方声明、Provide/New 两段构造、bootstrap 唯一接线。asset 不 import identity/workspace/project/issue；issue 不 import asset。已有 `shared.TxManager` 在 ctx 传同一事务，不另开假嵌套事务。

跨模块读取通过 owner 提供的能力：用户身份/资料可见性、工作区存在及成员授权、项目锁/可见性、工作项/评论/草稿的读写边界。不能复制一份权限判断到 asset。业务 owner 的 app 声明资产校验/引用端口，只有“已授权的当前宿主 + 此 actor 可用的已上传 ID”通过；一般副作用继续通过同步事件通知，再在原事务 InsertTx 投递任务。

关键绑定/迁移是业务一致性的一部分：校验、锁住资产、修改所属字段/归属、业务行写入、动态、事件及任务在同一事务中成功或回滚。端口只能操作资产自己的表，不含存储网络 I/O；不能在模块间回调执行任意闭包或暴露连接池。P2/P4/P5 的 spec 分别画出完整调用和锁序，先于实现核对。

总序继承 M4：无连接进入写门 → workspace 稳定边界 → project（跨项目升序）→ issue/评论/草稿既有锁序 → asset（UUID 升序）。获取 asset 锁后不追逐另一个项目/宿主锁；观察到宿主变化返回冲突并重新发起。确认封存和复制的存储 I/O 在事务之外，最终提交必须重验父范围和权限。

草稿发布复用当前 `W(S) → P(N) → D(U)`，先执行唯一创建核心的 prepare（相关校验、清洗和事务内序号分配，尚未写业务行），再收齐全部活资产及当前引用并按 UUID 顺序锁定；最后一把资产锁后读取唯一 At，再 write 工作项、迁移资产、软删草稿及发事件。迁移资产的 draft_id 清空、issue_id 填新 ID、用途由 draft 对应到 issue，时间/操作者与创建/软删草稿一致。竞争发布、改项目、上传完成、父删除都要实锁交错测试，不能靠固定 sleep。

草稿保存保持 M4 的结构400与业务值延后校验，资产校验只消费同一清洗器的图片投影，不改写原 payload。迁移实现显式依赖已有事件发布器，全部资产写成功后按 UUID 发普通附件 added；issue 消费端口不变，动态仍在同一事务。

## 8. 内容、历史与附件计数

服务端 sanitizer 留在 `issue/adapter/content`，app 使用小接口，domain 纯净。对 `image-component[src]` 中的规范 UUID 检查资产用途/范围；其他安全 src 按 §6 保留但不能获得内部图片解析权，raw img 同样不触发远程加载。既有文本/提及/链接/复选框标识继续使用，增加有序图片身份序列，换图与换序都造成内容变化。

编辑器中的显示 URL 只在 node view 内解析；`src` 始终是稳定 ID。图片渲染请求按会话、宿主、节点身份失效，A 图片的慢响应不能写到 B 节点。上传未成功时不保存假成功图片；保存/发布遇到未完成上传要明确阻止并展示可访问错误。

删除编辑器节点仅修改文档，不立即删除对象，撤销/重做及历史版本仍能解析。已绑定的描述图片保留到宿主删除；v0 不为每次编辑做跨版本垃圾回收，代价是移除图片仍占存储。这一保守规则保护 M4 的历史版本，不把最近 20 个版本之外的裁剪等同于可以删除仍活宿主的图片。未绑定准备件可以按期限清理。

`Issue.attachment_count` 只数该工作项活的 `ISSUE_ATTACHMENT`，不把描述图片、评论附件或草稿准备件算入。追加 ±attachment_count 排序和“附件”显示项时同时更新 SQL、游标比较器、生成 DTO、store 与界面；不先返回虚假 0。计数批量读取，不做每项一次查询；跨请求新增/删除仍可能影响游标位置，与 link_count 边界一致。

所有返回 Issue 的读取将已授权结果与附件计数放在同一只读快照。详情、按编号读取、直接子项列表原先没有事务，P5 复用已有 ReadTx，在首个宿主读取前进入；列表、关系和搜索沿已有快照，写返回沿原写事务。计数只接收最终返回的去重 ID，不查 lookahead；group/subgroup 合并各 cell 后批量统计。创建核心与 Locks.seen、Listing 各自显式注入小接口，不用 nil 或端口错误回落为 0。代价是三处旧读取增加一个短只读事务；拒绝顺序、不取写闸门和行锁的约束保持有效。

资产新增/删除工作项附件时产生具名事件 `asset.attachment_added` / `asset.attachment_deleted`，共享事件 envelope 的 Entity 指向 asset、Project/Workspace/Actor/OccurredAt 真实，Changes 带关联 issue ID；同一事务投递或拒绝。是否展示工作项动态由 P5 按 M4 的动作规则固定：每次真实增删各一条，重复 complete/delete 无额外行。项目批量清理不制造逐个用户动作动态。通知/Webhook 收到精确事件责任交接，不先实现通知。

**P5落实（T1–T15、整审、最终main CI及归档清理已完成）：** 评论附件走独立分页端点，创建最多100个attachment_ids；计数由asset在相同读取快照提供完整非零目录给issue SQL排序，普通页批量计返回ID，不跨模块JOIN、不保存冗余计数。草稿迁移保留已有bound，当前正文所用描述才新增绑定；从未引用的准备图片继续私有、可删和原7天清理，不能因发布自动公开。P5落实API/生成类型/比较器和现有store消费，P6/P7按§12挂页面；详见[P5主spec](specs/P5-issue-assets.md)、[绑定合同](specs/P5-content-binding.md)、[读取合同](specs/P5-attachment-reads.md)。

## 9. 前端与视觉

P6 新 AssetService 作为当前会话 API client 的消费者；旧 FileService 随 P7 最后编辑器/评论调用者迁移一并删除；不在模块顶层 new 带会话实例。原始 PUT 使用不带业务认证的独立传输器，支持 AbortSignal、准备/上传/确认阶段进度、超时和可区分的拒绝；独立 fetch 不提供字节进度，不能编造百分比；取消不能吞错后继续 complete。store 以 session/loginId + workspace/project/issue/draft/comment scope 管理，复用 IssueRootStore 的权限失效入口，不另造平行权限缓存。

**P6 PC1（负责人 2026-11-04 批准 A）：** 复用 IssueRootStore 时，现有 PermissionReads 同时记录规范项目可见性与现有有效角色的变化，补齐 member_role=null 的合法封面消费者；工作区匹配、当前 ProjectStore 事实和原 R06 规则仍是来源。目录缺席使旧生命周期永久失效，恢复后须新读；水位不是工作项授权，UserPermissionStore 和后端权限不变。不得另建 Asset 权限观察器或用历史详情复活封面；T2/T3/T12 联合回归及行为变异后验收，见[决策记录](reviews/P6-public-project-lifetime-decision.md)。

保持现有路由 key，上传进度/弹窗在所属子树内；unmount 取消请求和 debounce，代次不符的进度、成功、失败均不能更新新宿主。SWR key 含 loginId，缓存 URL 包含到期时间，撤权清掉已解析地址。请求中按钮与关闭行为、拒绝保留、成功关闭、返回焦点和读屏通知按 M3 规则。

头像/封面控件从只读 Plane 源码对照恢复，用新合同，不恢复旧上传地址或旧弹窗类名。项目保留预设和随机选择：制作本项目自己的 29 个静态封面（稳定 `cover-01` 至 `cover-29`，具体视觉允许实现者决定），登记 SOURCES；随机只在用户主动选择时选一个稳定 ID，不在后端/渲染时不停变。预设无需上传副本；自定义封面先上传准备件，再随创建一次绑定。不能保存 Vite hash 路径。

附件图标替换 figma/pdf/csv/excel 第三方标志，使用现有图标或自制中性 SVG 并登记来源。不改 shared 动效/弹窗框架；如果发现新的跨模块基础库缺陷，按风险纪律处理。

P7 最后旧文件消费者迁移时删除整个 `web/packages/services` 及包依赖；P6 不导入旧上传 helper，避免为提前删包而跨入编辑器实现；仍供 M6/M7 旧 APIService 使用的 URL 规范化连测试搬到 web 内，保持那些未实施功能的现状。死成员扫描重新运行，逐项核实真实消费者；不依据历史 48 个数量做机械删除。修改文件 oxlint 为零，并完成至少一个尚未清零规则的全仓清零，不能重复领取 M4 已清的 no-shadow。

公开旧 `copyMarkdownToClipboard` 不能直接复用；新消费者走已受控复制通道。若需要修复旧入口，成功/false/异常/无事件/卸载/换编辑器的监听释放与反馈一起测试，否则保持无消费者并把风险具名交接。

## 10. 清理、运行与观察

默认未完成申请、已完成但未绑定准备件 7 天清理；上传授权过期后不能开始传输，已在授权期内完成的对象允许在资产清理前确认，确认仍需重新授权、验证实物和唯一发布。签名/网络期限用运行时钟，7 天/60 天业务保留用既有业务时钟，两者不混用；清理须同时满足对象授权/传输的运行期限。已软删资产严格超过 `cleanup.purge_after`（默认60天）进入 purge，final对象回收与数据库行使用同一既有配置，沿用业务时钟，worker 调度用真实时钟。每批有限行数、每次有限时间，继续工作使用既有 snooze/retry 机制，原审计保持。

staging 在授权到期及传输预算结束前不能被视为永久消失；旧 PUT 可能正在路上。对象清理与专属 namespace 的分页对账处理后到对象，未记录/无活引用且超过安全宽限的对象才可删。对账只能看实例独占的前缀，绝不清理桶中别的项目数据；部署者必须配置独占前缀，测试明确放邻域对象验证。候选和失败操作保留记录至清理成功，不能仅依靠桶生命周期规则。

新增配置集中在 `files`，强类型、未知键拒绝、启动校验、样例与日志遮盖同时更新。默认 5 MiB 限额沿用实例 `file_size_limit`，更改配置同步前后端；超大整数、预算乘加溢出、零/负值及不合法 origin 拒绝启动。首期不支持 multipart/resumable，最大配置上限 100 MiB，超限不悄悄拆传。该上限与封存/复制、内存/磁盘预算测试绑定。

记录请求 ID、asset ID、阶段、字节数、耗时和错误类别；不记录签名 query、密钥、用户私有文件名/正文或客户端完整可执行错误。健康检查不通过写对象探测改变数据；启动验证配置及能力，实际存储故障使操作明确失败。

M5 不为关闭交接人为添加 CLI。若第一条会投 River 任务的命令在此期间出现，使用已有 NewInserter，不启动 workers，真实运行角色测提交/回滚任务；否则原文转交下一 M。

## 11. 验证策略

单元覆盖纯规则；adapter 契约覆盖真实字节与后端错误；bootstrap 组合覆盖完整认证、权限、事务、事件、SQL/外键、作用域与全程序 API；Playwright 页面覆盖浏览器实际 PUT、真实磁盘/对象字节和持久化结果。`e2e/fixtures/storage.ts` 先提供普通函数，再薄封装 fixture，不能只断言 DOM 上出现文件名。

负例至少包括：另一账号同文件名、跨私有项目、guest 自己/他人、管理员兼 guest、已归档/已删除、源可读但目标不可写和反向、同大小不同字节、PUT 迟到、copy 成功而 DB 失败、清理失败再试、父删除先/后、草稿双发布、换宿主后旧成功/错误。每个性质有真实可杀死的定向变异，不用只镜像实现的测试。

local/S3 共享适配器合同及最少浏览器场景；Chromium/Firefox/WebKit 验证上传与展示、复制、拖放、取消和慢请求。完整项目七项本地门禁先于有产品变化的检查点推送：gen/gen-check、lint-go、test、lint-web、knip、test-web、build+e2e（具体命令按 Makefile 和 CI）。不并跑容器重套件、不改变旧容器。

每阶段准确分支 SHA 的 CI 及 `--no-ff` main SHA 的 CI 分开记。已知 W11 偶发只按既有规则处理：重跑通过不能写成根因已修；重跑仍失败时停下上报。归档保存原件、摘要、哈希、字节数、模式和集合对账，未知文件为零才常规清理分支/worktree。

## 12. 分阶段推进

每段先 spec、plan、预检，再逐 Task 实现及检查，结束做 deep-review 三视角和修复复审。每段最多约 16 Task，超限在实施前拆分，不压缩验收。后续阶段只在前置事实落实后写可执行 plan；下表是边界，不是假定全部方案已经过预检。

| Phase | 交付边界 | 必须同时完成的验证 |
|---|---|---|
| P1 | 存储小接口、local 实物传输、签名、配置/限流/期限、共享合同骨架；不开放业务申请 API，不建未接生命周期的表 | F2/F3 的 local 合同、路径/缓存/重启/并发、全程序传输路由清单；默认 API 限制不变 |
| P2 | asset 迁移、对象记录、申请/确认/读取/删除/复制、全部宿主权限端口及删除/清理基础 | F1/F3/F13/F14/F15 API+DB；首张表同步删除图与目录守卫、锁序、无连接存储 I/O |
| P3 | s3 适配器、真实兼容服务合同、CORS/CSP、配置和部署样例 | F4；长度/类型签名、条件封存、失败/重试、相同字节和孤立对象对账；local 回归 |
| P4 | 用户/工作区/项目真实引用字段、稳定预设封面 ID、一次创建绑定、资料 URL；organization_size 收紧 | F5–F7 API；所属迁移/事务回滚、旧引用回收、成员头像、非法/过期准备件 |
| P5 | 工作项/评论/草稿引用、附件计数/排序、草稿迁移、图片清洗与内容标识、事件 | F8–F15 API+DB；并发双发布、历史/撤销图片、完整删除图与事务事件；架构守卫不放宽 |
| P6 | 新上传传输器与会话 store、用户/工作区/项目页面和预设封面 | F5–F7/F16/F17 页面；真实上传、失败恢复、键盘/读屏、跨会话及跨路由迟到结果 |
| P7 | 工作项/评论附件、草稿与编辑器图片页面、列表计数/排序/显示、旧文件消费者及 services 包删除 | F8–F12/F16/F17 页面；伪造剪贴板、取消/过期/重试、撤权、Back/Forward、旧复制入口条件 |
| P8 | 文件全链路容量/清理演练、所有交接复核、死成员和 warning 收敛、默认部署演示与三浏览器验收 | F1–F18 最终矩阵、精确数据证据、deep-review、M6–M8 交接及收尾材料 |
| closeout | 首次 noff 合收尾并等 main 绿，再补完成勾选/总体 9.4，经第二次 noff/main 绿，最终归档清理 | §14 每条证据、全部输入关闭、精确两轮 main CI 与归档清单 |

P8测试设施故障演练同时承接P3-M1：S3测试夹具启动与清理同时失败时应保留两个错误及原cause，当前只传播清理错误；这是诊断Minor，见[re-37](../../reviews/re-37-M5P3整体审核.md#3-minor与未发现项)，未声称已修。

P4-M1维护项归P8死成员/接口清理：WorkspacePatch.Logo是未消费的内部输入字段，当前HTTP不会赋值，不影响实际图片更新；P4整审只记录，不声称已移除。见[re-39](../../reviews/re-39-M5P4整体审核.md#p4-m1维护minor递交p8)。

P1 起第一个实现提交更新关键词 phase 与到期例外；纯设计分支不提前改 phase。以后到期文件必须替换/删除，不能仅推迟日期。

## 13. 输入交接与后续交接

当前所有输入仍为 open，只有实际验收或明确转交完成才关闭，原文与处理结果分开。

| 输入 | 条款落点 | 关闭阶段 |
|---|---|---|
| [M0-hardening-http-timeouts](handoffs/M0-hardening-http-timeouts.md) | §5.3、F2，逐请求读写期限和真实慢流 | P1/P3 |
| [M1-P2-trim-content](handoffs/M1-P2-trim-content.md) | §3/§6/§9，四种当前事实、无 Page/企业类型、编辑器不吞非图片、文件限额、无 Unsplash | P7 |
| [M1-P3-trim-platform](handoffs/M1-P3-trim-platform.md) | §3/§6/§9，资源类型和服务替换、现有 MIME/封面残留逐条复核 | P7 |
| [M1-closeout](handoffs/M1-closeout.md) | §9，死成员/prop、一个新 warning 清零、中性图标、自有预设封面和一次绑定、源复制权限 | P8 |
| [M2-closeout](handoffs/M2-closeout.md) | §4/§5/§9/§11，identity FK、头像页面、上传中间件与 CSP、services 包删除、会话和 storage fixture | P8 |
| [M3-closeout](handoffs/M3-closeout.md) | §4/§6/§9，logo/cover/avatar、organization_size、预设取舍、弹窗/焦点/可访问错误 | P8 |
| [M4-closeout](handoffs/M4-closeout.md) | §4/§7/§8/§9/§10，附件计数与页面、草稿原子迁移、图片标识、父删除/purge、首个任务 CLI、旧 copy 方法风险 | P8 |

收尾分别给 M6/M7/M8 写 `handoffs/M5-closeout.md`：明确资产端口/生命周期与作用域要求、真实支持的存储协议、事件名字和载荷、清理/通知责任、尚未触发的 CLI 条款原文。未复用且未修复的 copy 方法风险继续具名转交，不能因没有消费者写成“已修复”。M5 自己承诺的可用功能不得靠转交逃避完成标准。

## 14. 完成标准

仅在收尾 first main CI 成功后逐条勾选，并附最终证据；本节现在全未完成。

- [ ] F1–F18 每条有与失败路径相区分的 API/字节/页面证据，未把 mock 当真实存储。
- [ ] local 默认可运行；HMAC/路径/期限/超限/并发/重启验证通过。
- [ ] s3 真实兼容服务与 local 共用合同通过，配置/CORS/CSP和明确支持边界有证据。
- [ ] 已确认对象不可被旧上传 URL 覆盖；并发、崩溃、未知外部结果和孤立对象清理有证明。
- [ ] 全部资源用途、宿主/角色矩阵、复制源与目标、准备件绑定授权通过。
- [ ] users/workspaces/projects 引用、成员头像、稳定封面一次写入、organization_size 契约完整。
- [ ] 工作项/评论/草稿附件和真实 attachment_count、排序/显示全部可用。
- [ ] 草稿发布的创建、资产迁移、软删与审计同事务；唯一发布与交错失败证明通过。
- [ ] 图片清洗、有序 src 内容标识、skip_activity、历史/撤销与持久身份通过。
- [ ] 所有删除入口、新表目录/外键图、有限批次、原 At/By、60 天 purge 和存储重试通过。
- [ ] 文件传输/渲染/store/弹窗在换身份、路由、撤权和迟到回包时正确失效。
- [ ] 中文/英文错误、键盘/读屏/焦点、请求中/拒绝/成功/取消的交互通过三浏览器核验。
- [ ] 无旧文件 API、死 services 包、到期 keyword 例外；死成员逐条核实、改动文件 oxlint 为零及一个新规则全仓清零。
- [ ] Plane 差异、前端改动、配置/部署说明、素材 SOURCES 和原始证据均已同步。
- [ ] 所有输入 handoff closed 并有落点；M6/M7/M8 收到 M5-closeout，首个 CLI 与 copy 风险未丢失。
- [ ] 最终本地七项、deep-review 和修复复审、精确分支/main CI 均通过；历史失败诚实保留。
- [ ] 全阶段与收尾 noff 合入，完成标记第二次 noff 后 main 全绿，总体 §9.4 标完成。
- [ ] 原始证据完整归档并验证 hash/bytes/mode/membership，分支和 worktree 正常清理、未知文件为零。

## 15. 进度与裁定

| 阶段 | 状态 | spec / plan / review |
|---|---|---|
| 设计与 P1 预检 | 已完成 | [预检记录](../../reviews/re-31-M5设计与P1预检.md) |
| P1 | 已完成：noff main 85dc4087，CI 41471671630 全绿，792 文件归档核验并清理 | [spec](specs/P1-local-storage.md) / [plan](plans/P1-local-storage.md) / [阶段记录](reviews/P1-local-storage-review.md) / [整审](../../reviews/re-33-M5P1整体审核.md) |
| P2 | 已完成：noff main2d1bcf0c，CI41556463795全绿，2191文件归档核验并清理 | [spec](specs/P2-assets.md) / [plan](plans/P2-assets.md) / [整审](../../reviews/re-35-M5P2整体审核.md) / [阶段记录](reviews/P2-assets-review.md) / [裁定](reviews/P2-purge-contract-decision.md) |
| P3 | 已完成：noff main5807a0d6，CI41591764345全绿，2251文件归档核验并清理 | [spec](specs/P3-s3-storage.md) / [plan](plans/P3-s3-storage.md) / [预检](../../reviews/re-36-M5P3预检.md) / [阶段记录](reviews/P3-s3-storage-review.md) |
| P4 | 已完成：noff main0f93100a，CI41641099230全绿，2301文件归档核验并清理 | [spec](specs/P4-profile-images.md) / [plan](plans/P4-profile-images.md) / [整审](../../reviews/re-39-M5P4整体审核.md) / [阶段记录](reviews/P4-profile-images-review.md) |
| P5 | 已完成；最终main f5f4ce94、CI41772399410三job全绿，5053文件归档核验且三组分支/worktree清理完成 | [spec](specs/P5-issue-assets.md) / [plan](plans/P5-issue-assets.md) / [预检](../../reviews/re-40-M5P5预检.md) / [整审](../../reviews/re-41-M5P5整体审核.md) / [阶段记录](reviews/P5-issue-assets-review.md) |
| P6 | PC1已批准A并同步合同；最终main基线预检通过，T1–T5完成，T6接续 | [spec](specs/P6-profile-upload.md) / [plan](plans/P6-profile-upload.md) / [预检](../../reviews/re-42-M5P6预检.md) / [决策](reviews/P6-public-project-lifetime-decision.md) |
| P7–P8、closeout | 未开始 | 待前置阶段完成后逐段展开 |

当前可自行决定的局部取舍及代价：编辑器只处理图片，普通文件走附件区；预设封面使用稳定编号，不上传构建素材副本；描述图片保留到宿主删除，代价是占用空间；封存复制增加一次存储操作，换取确认后字节不被旧 PUT 改写。以上均在既定文件模块范围内，设计预检已核对一致性，具体实现按阶段合同验证。

设计审查已修正的衔接：复用已有用途 MAC；保持 M4 的安全 URL 清洗合同；业务 JSON 仍 no-store；签名/传输使用运行时钟；外部操作记录放入有界 storage_metadata，不增加业务表。

P1预算/MAC/Range、P2状态表/锁图/清理/owner端口、P3 SDK与双存储合同、P4绑定/读取/反向清理合同均已随实现验证并归档，见各阶段记录。P5 T1–T15完成：内容身份、绑定/迁移、精确事件/动态、分页、真实计数与完整全局排序、三宿主调用方及权限/字节/清理/容量均有证据，本地七门禁、整审和最终main CI闭环。集成期间修复测试夹具并按批准将E2E作业预算20改为25分钟；最终main f5f4ce94的CI41772399410三job成功、E2E364/364正常收尾。历史W11失败和超时原件全部保留，5053文件归档逐hash核验，三组分支/worktree清理且归档再核通过，见[P5§22.2](reviews/P5-issue-assets-review.md#222-最终main全绿与归档清理)。P6 PC1已批准A并同步设计、spec和plan；最终基线预检通过，T1–T5完成，T6接续。若事实证明既定架构无法满足安全或一致性要求，提交当前文档和证据后上报。

## 16. 协议核验来源

- [AWS：预签名 URL](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html)：有效期内可重复使用，PUT 可替换同 key；因此不能以 HEAD 后置 uploaded 作为不可变发布的充分条件。
- [AWS：CopyObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CopyObject.html)：源 ETag 条件与复制响应语义；候选需独立 key，失败要保留外部操作记录。
- [官方 SDK presign 合同测试](https://github.com/aws/aws-sdk-go-v2/blob/main/service/s3/internal/customizations/presign_test.go)：签名头行为参考；最终以锁定版本与真实浏览器/服务实测为验收。

这些是设计的技术依据，不是已完成的 Nerve 实现或测试结果。不得在最终验收中用文档引用替代本项目证据。


### P4/T7邀请预览锁核对（2026-11-03）

原P4 access合同把撤回邀请误认为持工作区N；M3实际是工作区S后邀请U，停用账户还会直接锁并删除发给其邮箱的邀请。预览签发沿本模块现有父→子顺序补工作区S→固定邀请行S→资产锁，锁下重验未删除/未接受和固定工作区。此为实现既定有效链接能力的局部合同修正，不改变M3写入协议或扩大公开读取范围。代价是CPU签名和提交期间，操作同一邀请需等待短事务；T7/T10验证两方向及错token零查询。详见[P4读取合同](specs/P4-image-access.md#4-所有响应入口)。
