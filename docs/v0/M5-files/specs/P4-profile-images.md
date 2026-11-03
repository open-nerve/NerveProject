# M5 / P4：资料图片、业务引用与原子绑定

> 预检通过，T1–T4实现与聚焦回归完成，T5待实施。分支codex/m5-p4，基线main5807a0d6e369c486e98e4cb00389d8b309322221；P3精确main CI41591764345全绿、归档清理后核验通过。一次fresh builder check无重要未决项，主控plan自检及生成/lint/迁移/架构基线通过，见[re-38](../../../reviews/re-38-M5P4预检.md)。
> 输入：M5-design §3/§4/§6/§7/§12 P4，M2/M3/M1-closeout交接；P2-access/recovery/protected-purge合同。负责人授权连续M5，局部取舍自行裁定；重大新风险先提交后上报。

## 1. 目标、范围与方案

让用户头像/个人封面、工作区图标、项目封面成为真实业务引用；一次项目创建请求完成封面绑定，已有成员资料能返回有权限的短期图片URL。关闭F5–F7的API部分及organization_size响应枚举；页面控件、29张实际预设素材与会话缓存归P6，工作项/评论/草稿绑定归P5。

采用所属模块的明确字段和小消费接口：owner拥有授权、当前引用、事务、业务写入；asset只锁/校验/改自己的资产，签发只用现有CPU-only ReadSigner。bootstrap逐值转换并显式构造。正式实施合同还包括[读取与绑定锁序](P4-image-access.md)及[反向引用清理](P4-reference-purge.md)，三份文档一起预检。

比较过：①所属事务内锁/校验→业务行→绑定，引用失效由owner端口解除（采用）；②公开任意资产绑定/依赖前端创建后补PATCH（会有半成品、越权与错误事务边界，拒绝）；③反向FK自动SET NULL（丢失明确owner清理职责，不能证明实际解除流程，拒绝）。采用①为主控在既定设计内的实现裁定；新增引用必须随同物理清理和守卫交付，不能只加列。

## 2. 交付物与数据边界

| 位置 | 责任 |
|---|---|
| migrations/sql/00035_identity_users_avatar_asset.sql | users.avatar_asset_id、cover_image_asset_id，nullable UUID，具名即时非deferred NO ACTION FK及各自无条件索引 |
| migrations/sql/00036_workspace_workspaces_logo_asset.sql | workspaces.logo_asset_id，同一FK/索引规则 |
| migrations/sql/00037_project_projects_cover_image_asset.sql | projects.cover_image_asset_id同上；cover_builtin_id nullable text，精确cover-01..cover-29，最多一个非空的CHECK |
| 各owner domain、app、postgres/queries、HTTP | 只读写本模块列，sqlc scope归owner；patch区分省略/null/值；现有操作的完整响应一致 |
| asset/app/image_bindings.go、image_reads.go | 通用资产侧的批量绑定校验/更新及已证明引用的图片签发；不调用owner授权、不重进Access.Within |
| 各owner app/image_ports.go、image_reads.go、adapter/postgres/image_references.go | 消费方小接口、读取编排、纯本表解除失效引用；不会跨模块JOIN |
| bootstrap/asset_images.go、asset_references.go | 按用途逐值转换、构造与解除顺序；不在模块之间import或传任意事务闭包 |
| api/modules/identity.yaml、workspace.yaml、project.yaml及生成物 | 字段、三态输入、完整错误码；organization_size枚举 |
| bootstrap组合测试、三owner/asset测试、e2e/stories/files/f5–f7-* | 真实认证/上传/引用/字节/权限/事务/删除证据；前置通过API |

没有新表、新依赖或全局安装；不新增生产默认容器、驱动迁移、任意URL输入、通用attach/detach/restore API。Plane只读对照root下models/user.py、workspace.py、project.py的nullable资源字段及URL property；本项目不用旧URL fallback、不保存构建地址，明确NO ACTION与owner解除的差异登记到plane-diff.md。修改的手写文件约400行内，按职责拆，生成物由make gen产出。

## 3. API形状与字段语义

已有URL字段保持名称，真实返回短期URL或null。每个资产URL增加对应的 `avatar_url_expires_at` / `cover_image_url_expires_at` / `logo_url_expires_at`，nullable date-time；URL与期限同时有值或同时null，使用P2/P3相同最短5分钟/最长65分钟、实际签名秒精确一致的规则，不另造时钟。数据库不保存URL/期限；签发零业务DML。

| DTO/输入 | 增量 |
|---|---|
| User | 必有nullable avatar_asset_id、cover_image_asset_id；现有两个URL及各自expires_at真实 |
| UserUpdate | 可省略、可null的avatar_asset_id、cover_image_asset_id；省略保留，null移除 |
| Workspace | 必有nullable logo_asset_id、logo_url_expires_at；organization_size改为OrganizationSize或null（六个既有值） |
| WorkspaceUpdate | 可省略、可null logo_asset_id |
| WorkspaceCreate | 不加logo字段：准备logo必须已有workspace；创建回答自然为null，后续Update绑定 |
| MemberUser | avatar_url真实，增加nullable avatar_url_expires_at；不开放独立他人profile/asset UUID探测接口；邮箱规则保持 |
| Project | 必有nullable cover_image_asset_id、cover_builtin_id、cover_image_url_expires_at |
| ProjectCreate / ProjectUpdate | 两个可空cover字段；详见下面互斥规则 |
| InvitationPreview | 已有workspace_logo_url真实，增加workspace_logo_url_expires_at；仅有效既有邀请链接证明可读其固定图标，详见access合同 |

OrganizationSize精确为Just myself、2-10、11-50、51-200、201-500、500+；请求规则不变，数据库既有CHECK不放宽。前端消费生成类型，不手写镜像或适配回旧Plane形状。

预设封面：有效ID为cover-01到cover-29；只保存编号。P4的cover_image_url专指上传图片，选择builtin时为null且expires_at为null；P6按builtin编号解析本项目静态素材，不返回尚不存在的构建路径。随机由P6的用户动作选择一个ID，后端不随机。

项目patch：省略两个字段保持；显式null只清对应字段；非null的asset自动清builtin，非null的builtin自动清asset并软删旧资产；同一个请求两个字段都非null为422，两个字段依请求字段合同顺序报告not_allowed。两者null清空；builtin空串/越界/大小写不符拒绝；不能通过URL字符串填任一字段。创建时同一规则，默认无封面。

## 4. 准备件校验、替换和原子性

owner先执行既有认证/结构/字段验证和权限顺序。资产可见性不能反过来提前泄露owner。锁下使用owner真实当前字段，客户端不提交“旧资产ID”。资产锁取所有旧/新ID并集、去重、UUID升序，一次收集；同时更新头像和封面不能分两轮追锁。

新候选必须属于当前actor、未删除、purpose精确匹配、完整scope匹配、uploaded且有active final、bound_at为空，且没有超过准备件7天期限（CreatedAt严格早于锁后业务Now减7天才过期，恰好7天仍可绑定，和P2清理一致）。上传签名15分钟过期不代替这项检查，已完成对象的允许绑定期限也不是读URL的运行时期限。

拒绝顺序：owner原有401/404/403/归档判定 → 所有指名新候选的不可见（不存在、其他申请者、已删、过期、完整范围不符均asset.not_found） → 同一允许范围内的用途不符（422对应字段not_allowed）→ 未上传asset.not_uploaded → 同范围已被其他引用绑定asset.bound。头像/封面字段按固定请求顺序；批量中隐藏件优先于其他候选状态，不能由另一件的409推测隐藏件。范围已迁到另一个project的准备件也为404，不回取那个project的权限/锁。基础设施错误原样传播。

- 同一字段重复提交当前ID：允许已绑定的**当前**引用，核对目的用途/宿主；不因当前管理员不是旧申请者而拒绝，不刷新bound_at/资产审计。当前ID不允许被搬到别的字段/宿主。
- 替换/移除：在同一事务清/改owner字段，再软删旧资产，保留不可变申请者；已更早删除的旧件不重写删除审计。不会在HTTP请求中删对象，最终仍按60天与恢复协议处理。
- 所有新绑定/旧软删/owner修改共享最后资产锁后的业务At和actor；签名用独立runtime时钟。普通名称更新而图片没变不刷新资产审计。
- 新项目：生成新ID，W(S)/创建权限/成员锁 → 预先锁并校验候选 → 读同一At → INSERT project时一次写入最终cover字段，再插入成员/偏好/默认状态 → 写候选scope由new_project变为project、ProjectID填新ID、bound_at=At → 签发响应。全在一个事务里；project的cover FK已指向存在且锁住的准备件，资产父FK再指向本事务新project，提交前两侧一致。新行没有其他事务可锁，不在资产锁后追已存在父行。唯一约束/成员/默认状态/绑定/响应签名/提交任一步失败，project及全部附属行、资产范围/审计全部回滚。响应丢失后的再次创建遵循已有名字/标识唯一性，不新增幂等键或二次PATCH。
- 公开DELETE资产仍不能删除已绑定头像/封面/logo；准备件仅申请者，generic Get/ResolveContent也不扩大成成员按任意UUID读取用户图片的接口。有权读owner的成员经owner回复取图片。

## 5. 消费端口与构造

精确锁图、值和方法见[P4-image-access](P4-image-access.md)。owner声明ImageBindings、ImageReader，bootstrap转换为asset/app的ImageBindingRequest/ImageReference；不用共享“全能文件服务”。

锁/校验和应用分两步，均使用ctx同一事务：`LockImageBindings(ctx, request) (time.Time,error)` 返回资产最后锁之后的At；`ApplyImageBindings(ctx, request, at) error` 只在相同已锁集合上重读/校验并写，不开事务、不进门、不读owner、不做对象I/O。两步用于让新project先有真实FK目标，同时所有新行使用同一At；不能将Apply作为不加锁的后门。adapter已有LockAssets拒绝无事务，补实际负例。

Provide仍可由pool单独构造自有rows和绑定能力；签发能力由 `asset.NewImageReader(Provided, ReadSigner, RuntimeNow)` 在bootstrap提前构造，只依赖asset自有存储和已启动storage，不依赖尚未New的owner。owner.New接完整小端口一次构造；最后asset.New仍接owner完整Access，不补注入、不动态找服务。绑定时Clock由显式构造输入传入资产绑定组件；具体公开外观可拆成 `asset.NewImageBindings(Provided, Clock)`，不改变Provide(pool)签名。

图片签发不做外部HEAD/GET；P2确认已经验证final。读端口只接受owner从持锁/授权查询得到的引用，校验asset purpose/完整scope/bound/uploaded/final/未删，批量锁后CPU签名。失效引用回答null而非替另一资产签发；数据库/签名故障原样失败，不能吞成无头像。asset不回头锁账户/父级。

## 6. 删除、清理与守卫

同时实施[P4-reference-purge](P4-reference-purge.md)：反向引用FK也为即时NO ACTION；对象回收和资产墓碑选择仍有界。候选先读、按owner顺序锁并条件解除当前引用、最后删eligible资产，整个数据库阶段原子提交，外部对象I/O仍在外。

保留P2全部父保护、分页前进、双时钟/晚到对象和精确23503分类。原始asset→issue→project→workspace→identity顺序不反转；asset墓碑阶段先经明确端口调用identity→workspace→project的引用解除，各owner SQL只操作自己表。目录从真实FK枚举全部反向边，逐项与owner声明/实际解除者对应；未知NO ACTION、CASCADE偷删owner、SET NULL绕过owner、漏引用、漏解除步骤都失败，无通用豁免。

## 7. 验证及预检限制

F5用户头像/封面、F6工作区图标/成员头像、F7项目创建/替换/预设使用真实API准备件与实际local图片字节；同一授权/绑定核心用真实S3兼容服务做完整程序参数化主路径与拒绝/回滚，不能只假Signer。跨浏览器上传已有P3覆盖，P4无新页面控件，P6补F5–F7页面和过期缓存。

真实PostgreSQL交错：密码重置/停用与签发、workspace撤权/删除与读取/绑定、project私有化/撤权/归档与绑定、profile替换与成员读取、双创建抢同准备件、旧引用回收与替换。观察实际PID、SQL/blocker，至少双ID顺序，-race -count=5；正常请求不能40P01，物理purge保留既有重试分类，不吞成blocked。阶段规模记录至少1000个图片引用的批量查询/签发与取消资源释放，列表不截断、不N+1（分块规则见access）。

15项以上实际性质变异，逐项有具名断言RED，不计编译失败、0 tests或伪造返回成功。完整七项、一次deep-review三视角及一次fresh整分支审查、局部修复RED/GREEN、准确最终分支和main CI、完整ignored证据归档分别留证。

已做的规划探针在P3冻结归档的payload/superpowers/sdd/P3-s3-storage中保存：双向NO ACTION、解除回滚、61/59天/有key区分、并发替换后的条件解除、清完后前进。1测试通过（1.985s，命令4.284s）；首次有参多SQL语句42601属于探针搭建错误，原件保留。此探针不是真实模块、并发或守卫验收，正式T1/T4/T10仍须验证产品SQLC/约束、真实目录和锁交错，不能以探针替代。

## 8. 完成标准

- [ ] A1：三owner迁移、索引/互斥CHECK、sqlc归属和Plane差异精确；空库Up/Down及已有P3数据升级通过，不宣称带绑定业务数据的降级无损。
- [ ] A2：profile两字段批量校验/锁序、替换/移除/同ID幂等、过期和跨范围拒绝，故障原子回滚。
- [ ] A3：workspace logo与organization_size生成枚举、管理员写/成员读/guest邮箱遮盖保持。
- [ ] A4：project自定义/预设/清除互斥、new_project一次创建绑定，附属行与候选全部同事务。
- [ ] A5：所有返回User/Workspace/Project/MemberUser的操作和既有邀请预览均有真实URL+期限合同；授权、签发、零业务DML与批量行为验证。
- [ ] A6：反向引用清理/父NO ACTION/有界游标/异常前进/删除审计/真实目录守卫及反例全闭环。
- [ ] A7：F5–F7 API/local实物、真实S3主路径/回滚及必要HTTP/PG并发五轮、≥15性质变异和1000引用证据。
- [ ] A8：完整七项、deep-review/一次fresh整审及修复复核、准确最后分支CI成功，失败原件保留。
- [ ] A9：授权noff、精确main CI全绿、归档hash/bytes/mode/membership/unknown=0、正常分支/worktree清理；下一段P5 spec/plan预检完成。

## 9. 不在本段与局部取舍

P5完成issue/comment/draft绑定/计数/事件/迁移，P6/P7页面、静态素材、缓存/失效、编辑器，P8规模/故障和交接收敛。M2/M3/M1 handoff保持open，记录P4 API结果；不把M5整体标完成。P3-M1测试夹具启动和清理同时失败时丢原cause、P1焦点Minor仍具名交P8，本段不借机修共享UI；2026-11-03依负责人“测试用完清理容器”的指示，对实际复现的pgtest启动失败遗留作局部修复，见阶段记录§11；P3-M1的S3双故障仍归P8。

主控裁定与代价：使用锁内CPU签发使已撤权请求不能新签发，代价是图片读短暂持有账户/父/资产锁；整集合接口保留现状，按有限块处理资产而不截断数据，后续容量仍需测。预设只返回稳定ID，页面素材P6前仍显示已有默认图。既有邀请预览沿M3的有效MAC链接能力只读它自己的logo，不能泛化到workspace或其他asset；撤销先提交即拒绝新签发，已发URL仍受最长65分钟bearer边界。
