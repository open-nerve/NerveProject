# M4/P14 详情与弹窗：准备与预检记录

日期：2026-10-31。状态：**已授权实施，逐任务推进中**。基线 `8bb4bbcb52a224ace4cb2f9d383bca9b97a022bf`；[spec](../specs/P14-web-issue-detail.md)、[plan](../plans/P14-web-issue-detail.md)、[文档预检](../../../reviews/re-20-M4P14预检审核.md)。§1–7 保留文档预检时点，实施结果从 §8 续记；A 条款待完整证据再勾选。phase 已进入 M4/P14。

## 1. 前置与方法

P13 合并、main CI、归档与清理的最终事实见 [P13 review §15](P13-web-filters-archives-profile-review.md#15-授权合并main-ci与归档清理2026-10-31)。本期从该准确 main 建 codex/m4-p14，主目录一直 main。原生 create_worktree 无法指定负责人约定的路径与分支，因此沿 using-git-worktrees 用系统 git worktree add；没有安装依赖或改产品源码。

按 02 §3 以现有源码、生成合同、测试和消费者静态核对替代原型/逐 Task 产品重放，不虚构红绿、生成 hash 或执行数量。brainstorming 的 fresh builder-check 独立读取 spec 与关键搜索/链接/编辑器合同，初次报告 nothing important left；主控随后按 deep-review 的逻辑、集成、覆盖三个视角顺序复核，不能算三个独立审者。独立补查 spec/plan 提出 R5，修订后的定点回读确认 resolved；具体阅读范围和静态结果保存在准备原件。

主项目 `.superpowers/sdd/P14-preparation-20261031/` 保存 builder-check.md、consumer-inventory.json、source-review-inventory.json、precheck-static.json、document-links.json；准确提交/远端核对另以 precheck-delivery.json 后置记录，不循环引用自己的提交 SHA。未来清理必须把本准备根与实施 worktree 的 `.superpowers/sdd/P14-web-issue-detail/` 一起归档。

## 2. 验收映射

| 标准 | 静态依据/实施任务 | 当前产品状态 |
|---|---|---|
| A1 | 原生 id/编号路由、header/body 与 detail-host/WorkItemLevelModals，T1/T14 | 未实施 |
| A2 | title-input、label-select、FieldError/useRefusedFocus、I2，T2 | 未实施 |
| A3 | 既有 confirmed autosave/boolean flag；弹窗 migration 缺口，T3/T4 | 未实施 |
| A4 | create/edit base/form、项目/标题字段和 I1，T4 | 未实施 |
| A5 | ModalCore、draft discard、document Escape，T5/T6 | 未实施 |
| A6 | issue-search 合同、原生父项全部消费者、I8，T6 | 未实施 |
| A7 | ExistingIssuesListModal、relations、M7 查重/父项适配，T6/T7 | 未实施 |
| A8 | link resource、URL 字段、metadata 空/终态、data favicon、I10，T8/T15 | 未实施 |
| A9 | 必需 delete onSubmit、子树语义/目录断言 I17，T9 | 未实施 |
| A10 | editor 三条 runtime 引入、settings host、emoji dead 检测，T10 | 未实施 |
| A11 | 五扩展死序列化、HTML fixture 与实际 build CSP，T11 | 未实施 |
| A12 | M4 image 配置、已删除入口、P12 表格链接与剩余计数入口，T12/T13 | 未实施 |
| A13 | OwnOnly、S2 403/404、真实双标签页已确认旧答，T14 | 未实施 |
| A14 | copy hooks 真实签名/反馈与六消费者，T15 | 未实施 |
| A15 | F、显式三 project、非零执行、至少16有效变异/恢复、dead/oxlint，T16 | 未实施 |
| A16 | 最终基线..HEAD deep-review、修复复审、准确分支 CI，T16 | 本次仅文档预检 |

## 3. 准备决定

D1–D8 见 spec §3，理由和代价逐项列出。核心是复用已有 root/确认/宿主能力、无源 id 的新建父项普通搜索、有源 id 才用 for=parent；元数据 UI owner 有期限回读，不增加常驻全局轮询；五处无人使用的 Markdown 序列化按 P14 既定裁定删除但保留 P4 安全 fixture；M7 仅共享选择器读取/类型适配，M5 图片和 P15 草稿/评论等边界不扩大。

T13 表格链接 P12 已完成，回归加实际剩余 all-properties 子项计数入口的片段修复。ProjectService 当前只有两处 search 读者，迁移后删除整个空类，不保留到 P15。它们是源码事实导致的任务落点校准，不是新架构或协议变更。

## 4. 预检发现与修订

| 编号 | 严重度/根因 | 文稿修订与复核 |
|---|---|---|
| R1 | Major，局部计划遗漏：只列 lazy 两个 editor wrapper，lite-text barrel 的 toolbar→TOOLBAR_ITEMS 仍能提前加载 editor | spec §2.9、plan T10 明确第三条 runtime 链、同一延迟边界/删除零读者导出，并由真实构建 Network 验 C10 |
| R2 | Major，旧设计路径已过时：T13 初稿把 P12 已修好的表格列当缺陷并预期行为红 | 标记 ceb154a9 为已有输入；表格补绿回归，真正剩余 #sub-issues 在 all-properties.tsx，T13 增其真实消费者/无效片段测试，不制造假红 |
| R3 | Major，复制接口不够具体：hook 本身捕获拒绝并提示，消费者若 await 后另提示成功仍会假成功 | spec §2.10/plan T15 写两参数 Promise<void>、hook 单独负责提示；relations/sub-issues 已是 hook，操作类型和调用一起改，不留 void 假接口 |
| R4 | Minor，任务留下查路径/空服务条件：现有 detail-host/use-copy-link 测试应直接指定，ProjectService 只剩方法本身 | 所有既有/新增目标明确；不存在的 row-scope 猜测改为新增 all-properties.test；T7 删除整服务/barrel/专用类型，P15 仅回归零残留 |
| R5 | Important，独立补查发现任务顺序：T5 要验真实父项页面，但 T6 才替换打开即发出的旧搜索接口 | T5 先完成关闭策略/嵌套单测及其他原生弹窗页面；父项 C4 页面放 T6 Step 4 随原生查询闭合，A5 映射同步，不扩大旧 API 失败白名单；独立定点回读已确认 resolved |

以上只修订计划，没有改变源码。独立 builder 的 R5 是局部任务依赖，由主控修订，无需负责人重新裁定；后续读者盘点和主控三视角用于收紧可执行性。当前没有未决架构级问题。实现若发现新风险，仍按纪律先提交再上报。

## 5. 与原设计/原型方法的差异

保留 16 Task 与所有 I/C 完成线。P11 已有跳转/描述合同、P12 已有选择器/表格 browse、P13 已有归档权限分别作为输入；M4 design §12/P14 同步这些事实和实际剩余路径。EMOJI 引用数量以当前 consumer-inventory 为准备快照，历史计数不再作为未来删除门。

没有实施原型、逐 Task 重放、浏览器产品测试或七项门禁。本次静态检查只证明文档结构、路径/调用/合同、链接、phase/产品 diff 及 P13 集成证据对应正确；不把 P13 测试/CI 填为 P14 的产品通过。

静态脚本核定 16 个任务、80 个未执行步骤、16 条未勾选验收，路径区分既有文件与计划新建文件；新增/变更文档无意外失效链接，设计中 4 个原有 M5–M8 最终 closeout 链接仍明确待收尾创建。产品 diff 为空，phase 保持 M4/P13，主目录 main 与准确基线一致，P13 CI/清理回执及归档清单哈希相符。最终文件哈希和实际检查数量见 precheck-static.json，提交后远端身份见 precheck-delivery.json。

## 6. 移交与下一步

预检完成后等待负责人启动，从 T1 开始；本轮不实现。阶段最终需固定准确基线至最终 HEAD 做 deep-review，局部修复复审、准确分支 CI 全绿后先保留分支，等负责人安排集成。

M1/M2/M3 输入子项映射见 spec §7，顶层 handoff 仍 open。P15 接时间线/评论/表情/历史/草稿/搜索和余下复制，P16/收尾接全局清理/C12/设计 §14/总体设计 9.4/M5–M8 closeout。不以本期准备关闭整个 M4。

## 7. 已知限制与证据保留

P13 main 第一次 CI 为已知 W11 偶发；UI 日志与一次失败作业重跑证据保留在 P13 集成根，最终结论看 §1 指向的后置收据。P14 未执行产品测试，未声称上述潜在实现缺口已修复。公开 CI metadata 可核 SHA 与作业结论，详细日志/重跑沿已授权浏览器，不提取凭据。

## 8. 实施接续（2026-10-31）

负责人已授权完成 P14、阶段末 deep-review 和局部修复，先不合并。实施证据根为本 worktree `.superpowers/sdd/P14-web-issue-detail/`；每次 run.py 回执记录 cwd/argv、HEAD 与 dirty diff/新增文件哈希、raw/exit，准备根继续保留。

### T1 入口与共同路由身份

编号读取仍沿原 useSessionSWR 和 root；仅将这一次读取移到路由 layout，经 Outlet context 给正文、经已证明 id 给页头，取消页头自行查询编号缓存。共同 Fragment 身份含原编号、workspace、服务端解析后的项目/工作项、既有权限 identity 与 loginId。没有新权限观察器、第二缓存或后端协议。T1 将 phase 改为 M4/P14，76 规则/24 例外检查通过，无到期延期。

原生 id replace、401/403/404 拒绝、A→B→A、真实页头删除框、加载/拒绝不露缓存操作和权限 ABA 与既有详情宿主回归共 23 项；基线原宿主 13 项绿。行为红 t1-red-final 的两项分别为缓存页头操作和共同边界内局部状态残留；前期夹具/类型错误单独保留，不当性质红。M01 去共同 key 检出后逐字节恢复。共同 key 的独立局部状态用宿主探针隔离验证，实际菜单/删除框另用真实组件，不能冒称浏览器证明；页面与会话 A1 剩余归 T14/T16。

局部决定：路由上下文只传已有 SWR 读取结果，不创建业务读模型；代价是正文依赖自己的 layout，已由真实嵌套路由测试核对。编号索引仍有 WorkItemLevelModals、命令面板和导航真实读者，保留。原生 id 后的第二次 core GET 来自已有 sub-issues 父项权威核验（调用栈诊断已留存），不误当路由重复请求删除。
