---
status: open
from: M1/closeout
to: M5
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint、附件图标的标志、新建项目的封面值、复制资源的权限

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1350 个（成员 898、prop 452），属于本 M 领域（文件、附件、资源、封面、编辑器的图片上传）的有 **48 个**（成员 40、prop 8）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M5`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M5` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 694 个（web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

## 附件图标里的第三方标志

`app/assets/attachment/` 的 17 个文件类型图标里，4 个用了第三方的标志：`figma-icon.png`（Figma 的标志）、`pdf-icon.png`（Adobe Acrobat 的标志）、`csv-icon.png`（Excel 的 X 标志）、`excel-icon.png`（Google Sheets 的图标）。M1 收尾复看保留的图片时发现（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 2.4 D4、第 9 节第 6 条），附件归本 M，交本 M 处理。
- **怎样处理**：换成中性的文件类型图标（例如只写扩展名的图标，或图标库里的文件图标），新图标的来源登记在同目录的 `SOURCES.md`。
- **关闭条件**：`app/assets/attachment/` 里没有第三方的标志。

## 新建项目时的封面值

新建项目时（`projects/create/root.tsx` 的 `onSubmit`），前端先上传预设封面的副本，再创建项目，但 `POST …/projects/` 的 `cover_image_url` 仍是构建里预设封面的地址（`/assets/image_<n>-<hash>.webp`，每次构建都会变）；项目建好之后才登记副本（`POST …/bulk/`）、用副本的地址覆盖（`PATCH …/projects/<id>/`）。这两步有一步失败时，项目保存的是一个下次构建就失效的地址（上传本身失败时不创建项目）。这是 Plane 原有的行为，M1 收尾 T14 的浏览器核对记下了这几个请求的顺序（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.15、第 8 节）。
- **M3/P8b 之后**：生成的 `ProjectCreate`、`ProjectUpdate` 没有封面，M3/P8b 删除了上面这段流程：新建项目时的封面一步、随机封面、项目设置的封面选择器和 28 张预设封面（`app/assets/cover-images/` 的 `image_2`–`image_29`），只留没有封面时显示的默认图 `image_1.webp`（[M3 设计](../../M3-workspace-project/M3-design.md) 7.8、[P8b spec](../../M3-workspace-project/specs/P8b-web-project-data.md) 第 3 节第 3 条）。本 M 随 `projects.cover_image_asset_id` 把封面的上传和选择器从 Plane 的源码加回时，预设封面一并加回（图片照 M1/P5 的做法重画，登记在同目录的 `SOURCES.md`），并照下面的条件写封面值。
- **关闭条件**：本 M 的封面接口只保存上传后的资源（或本 M 加回的预设封面的编号），不保存构建路径；新建项目只写一次封面值。

## 复制资源的权限

在编辑器里复制、再粘贴时，剪贴板带编辑器自己的类型 `text/nerve-editor-html`。粘贴这个类型时，`src` 是资源 id（不以 `http` 开头）的每个 `image-component` 都被标为复制（`web/packages/editor/src/helpers/asset-duplication.ts`），图片节点随后经应用请求 `POST /api/assets/v2/workspaces/{workspaceSlug}/duplicate-assets/{asset_id}/`（`web/apps/web/core/services/file.service.ts` 的 `duplicateAsset`），请求体带目标的 `entity_type`、`entity_id`、`project_id`。任何网页都能在自己的复制事件里写这个类型，里面的资源 id 可以是别人的。M1 收尾 T17 只让这段 HTML 在惰性文档里解析（不再执行里面的脚本），复制的请求照旧；T17 的评审确认这是 Plane 原有的行为，不是收尾引入的（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 2.9 的 Critical 1、3.17）。资源归本 M。
- **怎样处理**：本 M 实现复制接口时，先核对请求者能读取源资源（它所在的工作区和项目），再复制；读不到的与不存在的一样回答，不透露资源是否存在。
- **关闭条件**：复制接口对请求者读不到的资源返回 404（或本项目对无权资源的约定），有测试覆盖这种情况。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节；2.4 D4、2.9 的 Critical 1、3.15、3.17、第 9 节第 6 条。

## 处理结果（M5/P2）

P2复制接口先检查源准备件申请者及宿主可见性，再检查目标；隐藏源与不存在均404，目标写权限不足不能抢先透露，F13与真实HTTP双端矩阵/锁交错已验证。绑定成员资源与编辑器复制接入仍归P5/P7，死成员/图标/页面清理未宣称完成，本交接保持open。证据见[P2阶段评审](../reviews/P2-assets-review.md)，整审/集成结果仍单独核验。

## 处理结果（M5/P4 T13核对）

新建项目封面API已满足只保存上传UUID或稳定cover-01..29编号、一次POST完成绑定；默认状态/Apply/签名/提交失败均整笔回滚，实际字节与并发抢占已验证，见[P4记录§15–19](../reviews/P4-profile-images-review.md#15-t8项目封面与一次创建)。29张素材、选择器和随机动作在P6，旧服务/编辑器复制接入在P5/P7，图标与死成员/告警最终关闭在P8；此交接继续open。

## 处理结果（M5/P5 T13核对）

绑定后的六类内容资产可以由宿主合法成员读取；复制仍先证明源可读，再核目标可写，源隐藏与不存在一致。local/S3的F11真实字节、双项目方向和T10源404优先于目标403的行为变异已验证，见[P5记录§16–20](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。这没有放开资料图片或未绑定准备件，也不代表P7编辑器消费者已接通。预设素材、中性图标、死成员与warning责任仍归P6–P8，本页保持open。

## 处理结果（M5/P6 T13核对）

项目封面已端到端实现29张自制SVG、稳定编号、主动随机和一次项目POST绑定，原构建URL/先建后补写不再进入资料流程；F7 local/S3及三浏览器页面均核实际字节/引用。消费清单56文件去除旧裸URL，新的图片显示仅依当前宿主与范围。死成员扫描使用附录A脚本和当前TypeScript，移除本段已改UserStore中仅测试读取的canPerformAnyCreateAction，保留仍有生产消费者的projectsWithCreatePermissions；其余本段命中均给具体调用链，见[P6评审§18](../reviews/P6-profile-upload-review.md#18-t13-验收与清理核对)。

旧FileService/EditorAssetStore及services包仍由P7迁移最后消费者并删除；附件图标、中性素材与全仓告警类别最后由P8核验。本交接保持open，历史48条不直接作为当前删除名单。

## 处理结果（M5/P7b T12核对）

旧FileService/EditorAssetStore及services包已随最后消费者迁移而删除；编辑器内复制粘贴图片改由EditorImages在服务器复制（T7–T9），不再请求`/api/assets/v2`。附件图标、中性素材与全仓告警类别仍由P8核验，本交接保持open。

## 处理结果（M5/P8）

死成员：T7按附录A重扫，`domains.mjs --rows M5`删除前56行（member 53、prop 3）；`e879c84d`删除只为死成员存在的代码后剩41行，全部是误报，逐行写明读者或写者（T7表），T11后重扫不变。oxlint：M5全期改动文件零警告；T8把`valid-typeof`与`no-constant-binary-expression`全仓清零并设为error，上限随之调低（`28be6724`）。附件图标：`app/assets/attachment/`的17个PNG没有任何引用，T7整目录删除（`1718210a`），因此不再有第三方标志。新建项目封面值：P4/P6起只保存上传资产或稳定预设编号，一次POST完成绑定。复制资源权限：源不可读与不存在同为404，先证明源可读再核目标可写（P2/P5，F13与F11在T12重跑）。证据见[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)的T7、T8与T12。条款全部完成，closeout改为closed。
