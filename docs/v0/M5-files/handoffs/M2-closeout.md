---
status: closed
from: M2/closeout
to: M5
created: 2026-09-28
---

# M2 交给 M5：头像和封面、上传与按路由的中间件、删除 `@nerve/services`

M2（账户认证）的接口里，头像、封面是可为 `null` 的地址，M2 没有文件存储（[M2 设计](../../M2-auth/M2-design.md) 3.2）。下面这些由 M5 接上。第 1、2 节来自 M2 设计 13.2 给 M5 的两行，第 3 节来自 M2/P5 spec 第 5 节"不在 P5 范围内"，第 4 节是 13.2"M3 及以后有 stores 的 M"一行中 M5 的部分，第 5 节是 M0-P6 交接剩下的一项。

## 1. 头像和封面

- **做什么**：加入 `users.avatar_asset_id`、`cover_image_asset_id`；`User.avatar_url`、`cover_image_url` 开始返回签名地址；按新的上传协议加回 general 页和新手引导资料步骤的上传控件（M2 设计 3.2）。
- **迁移的范例**（M2 设计 3.14，总体设计 5.6）：先在 M5 的模块里建 `file_assets`（`<v>_<模块>_file_assets.sql`），再写 `<v+1>_identity_users_avatar_asset.sql` 给 `users` 加列和外键；后者归 `identity` 的 sqlc 条目（`server/sqlc.yaml`）。`TestSQLCSchemaScope` 核对 `ALTER TABLE` 的所有者，它的用例里就有这个文件名（`server/internal/archtest/sqlc_cases_test.go:30-32`）。
- **M2 做了什么**：建 `users` 时不建这两列（差异清单二·按表），接口的两个地址一直是 `null`。M2/P4 删掉了头像、封面的上传控件和 Plane 的上传路径（`cb2487c9`），`EFileAssetType`（`web/packages/types/src/enums.ts:22-29`）的 `USER_AVATAR`、`USER_COVER` 随之删除，现在是 6 种。关键词规则 `plane-user-assets`（`tools/keywords.json`）挡住 Plane 的 `/api/assets/v2/user-assets/`、`/api/users/file-assets/`、`UserImageUploadModal` 和 `USER_AVATAR`、`USER_COVER`（M2 设计 7.9）。
- **加回时**：资源类型按 M5 的协议定。沿用 `USER_AVATAR`、`USER_COVER` 这两个名字时，从规则的 `content` 中去掉 `\bUSER_(?:AVATAR|COVER)\b` 一支，`samples` 随之改；Plane 的两个地址和弹窗名仍然挡住。
- **关闭条件**：两列和外键照上面的两个迁移加入，差异清单二·按表更新；两个地址返回签名地址；两处上传控件用新协议；`node tools/keywords.mjs` 通过，`plane-user-assets` 仍挡住 Plane 的地址。

## 2. 上传与按路由的中间件

- **做什么**：`/api/v0` 的每个操作都经过模块级的 `Middlewares`：请求体上限 `server.max_body_bytes`（默认 1 MiB）和请求期限 `server.request_timeout`（默认 15 秒；默认值在 `server/configs/config.yaml:15,17`，交给平台的字段在 `server/internal/platform/httpserver/api.go:42-43`），它们会让 `/api/v0` 下的上传失败。M5 在平台加按操作的放宽设置，或者把上传放在 `/api/v0` 之外（例如直传对象存储），并按 M2 设计 3.6 的整程序测试处理：写进接口描述，或在设计中说明（M2 设计控制者复核 m7）。
- **`file_size_limit` 的执行**：名称 M2 已定（`api/modules/instance.yaml:106`，前端 `web/apps/web/core/hooks/use-file-size.ts` 读它），值来自配置；上传时由 M5 执行。
- **CSP**：`img-src`、`connect-src`（`server/internal/platform/webui/csp.go:40,42`）加上存储的来源。只有对象存储这种本来就在外部的来源才加（M2 设计 8.3）。
- **与 [M0 加固的交接](M0-hardening-http-timeouts.md)的关系**：那一项管连接上的读写时限（`http.ResponseController`），这一项管按路由的请求体上限和请求期限，两者都要放宽。
- **关闭条件**：大于 1 MiB、耗时超过 15 秒的上传都成功（测试）；超过 `file_size_limit` 的被拒绝；整程序测试（`server/internal/bootstrap/contract_test.go`）仍通过；CSP 只为存储放开来源，有测试。

## 3. 删除 `@nerve/services`

- **做什么**：M5 重写文件的接口调用之后，删除整个 `web/packages/services`（总体设计 7.1，[前端改动清单](../../frontend-changes.md) 3.1 的最后一行）。
- **现在还剩**：上传文件的元数据工具（`web/packages/services/src/file/helper.ts` 的 `getFileMetaDataForUpload`、`generateFileUploadPayload`，调用方是 `web/apps/web/core/services/file.service.ts` 和 `web/apps/web/core/services/issue/issue_attachment.service.ts`），以及地址规范化（`web/packages/services/src/helpers/url.ts` 的 `normalizeAPIRequestURL`，带单元测试）。
- **注意**：地址规范化的调用方是 web 的 axios 基类 `web/apps/web/core/services/api.service.ts`，还没对接新接口的领域经它调 Plane 的旧地址（M5 之后是 M6–M8 的领域）。删包时它还有调用方，就把规范化连同测试挪进 web（例如 `web/apps/web/core/services/` 下），或者证明那时已没有旧地址要补结尾的 `/`。
- **守卫**：规则 `services-files`（`tools/keywords.json`，只允许包里现在的几个文件）随包删除；包不存在之后，这条规则删掉，或改为挡住包名重新出现。
- **关闭条件**：`web/packages/services` 不存在，任何 `package.json` 都不依赖 `@nerve/services`；`make knip`、`make lint-web`、`make test-web` 通过；前端改动清单 3.1 的那一行改为已完成。

## 4. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M5 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M5 的范围（`d97c513` 上数出）：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `FileService` 8 处。它们现在经 web 的 axios 基类调 Plane 的旧地址。
- **关闭条件**：这 8 处都已消失；M5 新加的 SWR 键带 `loginId`。

## 5. e2e 的 `storage.ts`（M0-P6 交接剩下的一项）

- **做什么**：检查文件是否真的写入本地存储，按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/storage.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。
- **关闭条件**：`storage.ts` 照这个写法加入，上传的故事用它核对存储里的文件。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M5 的各行；[M2/P5 spec](../../M2-auth/specs/P5-web-account.md) 第 5 节；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

## 处理结果（M5/P2）

P2已有资产表、生成AssetPurpose和真实业务/字节两段协议；按§1调整关键词的裸用途分支，旧地址/弹窗/旧枚举实际反例仍失败。§5的e2e/fixtures/storage.ts已用显式token、生成类型与独立无凭据raw传输实现。用户反向字段、头像封面UI、会话store、services删净及S3 CSP仍在P3–P7，本交接保持open。证据见[P2阶段评审](../reviews/P2-assets-review.md)，整审/集成结果仍单独核验。


## 处理结果（M5/P3 T7）

§2的S3 CSP来源已由webui.Options接收实际签名origin，仅img/connect精确扩展；local旧policy与其他安全头保持。三浏览器F4、CORS负例及实际通配变异证明边界；IPv6字面量因浏览器CSP不兼容在配置时拒绝，DNS可解析到IPv6。§5的独立S3世界与Go辅助程序补齐实物SHA、失败回收。证据见[P3阶段记录§14](../reviews/P3-s3-storage-review.md)。头像封面UI、反向引用、services删除和会话分代尚未到期，本交接保持open；大文件/慢流及阶段全量门禁也不由本Task提前代签。


## 处理结果：M5/P4 T5（局部）

§1的两个用户反向字段、三态更新和签名URL/到期对已在真实API接通；锁下账户与资产一笔更新，签名或提交失败整笔回滚。PublicProfiles提供头像引用事实，成员头像仍由P4/T7接入。证据见[P4阶段记录§12](../reviews/P4-profile-images-review.md#12-t5用户资料的原子图片与真实读取)。本交接仍open：P6上传控件和会话store、P7旧services清理、P8最终验收尚未完成，当前聚焦验证不能代签后续工作。

## 处理结果：M5/P4 T10–T11（局部）

全部个人、工作区、成员/邀请和项目图片API的五轮真实权限交错已通过；F5–F7 API故事与local/S3同合同的真实字节、范围/审计和延期提交回滚也已验证，见[P4阶段记录§17–18](../reviews/P4-profile-images-review.md#17-t10真实授权与锁交错)。本交接仍open：P4清理恢复/规模、整审和集成未完成；页面、素材与会话缓存仍在P6–P8，不把API证据代作页面验收。

## 处理结果：M5/P4 T12（局部）

四类图片的61/59天保留期、四key存储故障保留引用与父行、真实新OS进程继续原River游标，以及1000/1001引用冷统计读取和取消资源释放均已验证；31项行为变异和最终race证据见[P4记录§19](../reviews/P4-profile-images-review.md#19-t12保留期真实进程恢复和冷统计容量)。本交接保持open，完整七项/整审/集成正在推进；页面、预设素材和缓存仍归P6–P8。

## P6 预检的删除责任澄清（交接仍 open）

当前最后旧 FileService/helper 消费者属于 P7 编辑器/评论范围。P6 只给新资料页面接 AssetService 和独立无凭据传输；P7 连同最后消费者删除 services 包并迁移仍有用的 URL 规范化。M5 总体删净验收和关键词规则不延期，原交接正文保留；见 [P6 预检](../../../reviews/re-42-M5P6预检.md)。

## 处理结果（M5/P6 T13核对）

§1个人资料/引导头像与封面通过新AssetService和原生File上传接通，独立资产字段更新、替换/移除、刷新/重登录及真实local/S3字节验收完成。§4本段新资料资源按loginId和权限代次持有；换账号、取消、权限ABA和旧读新写在真实页面与RootStore联合回归通过。§5继续复用普通storage函数和现有拥有资源的fixture，WebKit额外接收端观察不增加产品接口。详见[P6评审§16–18](../reviews/P6-profile-upload-review.md#16-t11-真实页面双存储与字节)。

§3及旧文件服务剩余会话单例仍属于P7最后编辑器/评论消费者，不把本段局部迁移当整包删除；M5最终清理/门禁由P8汇总，本页保持open。

## 处理结果（M5/P7b T12核对）

§3完成：`web/packages/services`整包删除，任何`package.json`不再依赖`@nerve/services`，pnpm按仓库版本移除独有依赖`file-type`及其传递依赖（未手写锁文件）。地址规范化连同原单元测试移到`web/apps/web/core/lib/api-request-url.ts`，旧APIService只改导入位置、行为不变；上传元数据工具随旧文件链删除。规则`services-files`改为挡住该目录任何文件重新出现。`make knip`、`make lint-web`、`make test-web`通过，前端改动清单3.1对应行已改为已完成。其余各节的最终汇总仍由P8收尾，本页保持open。

## 处理结果（M5/P7b T14核对）

T13真实页面发现M2认证中间件的缺陷（不是本页的交接项，因属M2的代码在此记录）：401后续期一次，再以`options.fetch(copy)`重发请求副本；openapi-fetch交出的是冻结options对象上的浏览器原生fetch，以options为接收者调用时浏览器在副本发出前抛Illegal invocation，M2设计7.1的“续期一次并重发一次”在真实浏览器从未生效（单元测试的FakeNerve fetch是箭头函数，未暴露）。`86c9f9f6`取出后单独调用，FakeNerve客户端的fetch改为与浏览器一样拒绝非window接收者，六个既有重发用例RED→GREEN；F16页面故事在三浏览器证明一次401经续期重发后读取成功、输入与收据保留。按本地缺陷处理（单文件、无接口变更），可单独回退。本页其余各节不变，保持open。

## 处理结果（M5/P8）

§1头像与个人封面：两列与外键、签名地址、个人资料与引导两处上传控件均已接通（P4/P6），`plane-user-assets`仍挡住Plane的地址与弹窗名。§2上传与中间件：文件字节不经`/api/v0`的1 MiB/15秒中间件，走逐请求放宽期限的raw路由；大于1 MiB、超过15秒的上传在T10的local/S3演练与页面慢流中成功；超过`files.size_limit`的上传被拒（413）；整程序契约测试在T12重跑通过；CSP只为S3的实际签名来源放开`img-src`/`connect-src`（P3 T7）。`use-file-size.ts`已删，实际读取点为`core/hooks/editor/editor-assets.ts`读`root.instance.config.file_size_limit`。§3 `@nerve/services`整包删除（P7b）。§4 `FileService`单例清零，M5新加的SWR键带`loginId`（P6/P7b）。§5 `e2e/fixtures/storage.ts`照fixture写法加入并被上传故事使用（P2）。另记P7b T14修复的M2认证中间件401续期重发缺陷`86c9f9f6`。最终证据见[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)的T10与T12。条款全部完成，closeout改为closed。

## 关闭（M5/closeout，2026-11-06）

§1–§5全部完成，见上面的处理结果与[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)。§1要求的[差异清单二·按表](../../plane-diff.md#按表)已更新：`users`一行记有M5/P4的迁移00035（头像与个人封面两列及外键，归`identity`）。没有转交项，`status`改为`closed`。原文与各阶段处理结果保留不改。
