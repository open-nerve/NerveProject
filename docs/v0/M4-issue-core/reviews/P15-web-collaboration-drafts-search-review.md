# M4/P15 协作、草稿与命令搜索：预检记录

日期：2026-10-31。状态：文档预检通过；尚未实施。基线 main 7a81ea18d8e9d80f079e4e30ae0e90b11be23ef9，分支 codex/m4-p15；[spec](../specs/P15-web-collaboration-drafts-search.md)、[plan](../plans/P15-web-collaboration-drafts-search.md)、[deep-review 预检](../../../reviews/re-23-M4P15预检审核.md)。

## 1. 前置与方法

P14 授权合并、main 准确 CI、I19/I3 集成测试修复及两工作树归档清理见 [P14 review §15–16](P14-web-issue-detail-review.md#16-授权集成main-ci-与归档清理2026-10-31)。本期只新建独立 worktree，主目录保持 main。原生 worktree 工具不能指定负责人要求的精确路径，沿 using-git-worktrees 使用系统 git worktree add。

按 02 §3 用源码、生成合同、消费者和测试静态核对替代产品原型及逐 Task 重放。没有安装 P15 依赖、改产品、执行 P15 测试或提前进入 phase。brainstorming 的独立 builder 全文读 spec/plan，并查评论授权、草稿发布、搜索和 clock，未发现关键未决问题；主控按 deep-review 三视角顺序核验，不是三位审者。最后路径及基线文字修订由主控负责。

## 2. 验收映射

下表均是实施后的验收落点，本次不勾选产品完成。

| 标准 | Task / 实际依据 | 状态 |
|---|---|---|
| A1 | T1，activity store/order、IssueUser、I12 | 待实施/回归 |
| A2 | T2，activity-filter/filter-root/sort-root、C6 | 待实施 |
| A3 | T3，CommentCard/QuickActions、后端评论授权、I11/I20 | 待实施 |
| A4 | T4，issue/comment reactions、picker、C6 CSP | 待实施 |
| A5 | T5，原生版本资源/restore、ModalTitle、getMarkDown、I13 | 待实施 |
| A6 | T6，draftInput、WorkspaceDraftIssues、个人草稿页、I14 | 待实施 |
| A7 | T7，P14 dirty/readiness 与 ModalCore、C9 草稿 | 待实施 |
| A8 | T8，工作项命令聚合与原生确认写、C7 | 待实施 |
| A9 | T9，theme-switcher sessionGuard、四文件 route 守卫 | 待实施 |
| A10 | T10，useIssueSearch/IssueSearchService、显式项目目录、I18 | 待实施 |
| A11 | T11，13 历史实例、WorkspaceService view 保留 | 待实施 |
| A12 | T12，ClockWorld/auto-archive fresh 水位、I16 page | 待实施 |
| A13 | T13，四复制消费者与 T5 Markdown、C8 LAN HTTP | 待实施 |
| A14 | T14，显式三个 browser、C6–C9、晚答与 Back | 待实施 |
| A15 | T15，本期 dead/lint/关键词、真实受影响文件 | 待实施 |
| A16 | T16，完整 F、整分支 deep-review、准确分支 CI | 待实施 |

## 3. 准备决定

D1–D8 详见 spec §3。命令原粗项拆为 T8 工作项动作与 T9 主题/创建路由，共 16 个 Task；其它顺延，完成线未削减。每 Task 约 300–1450 行含测试，超出先局部拆职责，不能把未接通领域挪进本期。

沿现有 root 权限所有权、确认队列与 owner；已满足的来源/actor/草稿能力只回归，版本 Markdown 用既有同步 getMarkDown，复制 hook 单独负责反馈。命令根页空 q 保留原命令，非空 q 原生搜索，项目在显式目录本地过滤。M7 服务与 P16/收尾边界不扩大。局部决定均有错判代价和测试落点；架构新发现仍先提交再上报。

## 4. 预检发现与修订

[re-23](../../../reviews/re-23-M4P15预检审核.md) 逐项记录 R1–R9：复制返回/失败、payload 整替换与发布顺序、clock 新进程凭据、M7 搜索/视图边界、查询 owner 与目录、LAN HTTP、精确路径/守卫、已有能力的真实回归、T8/T9 与 T10 搜索迁移的先后依赖。已全部修订文稿。独立 builder 未提出新的阻塞决定；主控最后修正 power-k/core/types.ts 的歧义，并全文核对 commands-list/search-menu 后把根页非空查询的页面验收明确留在 T10/T14。这两项最终修订由主控复核。

没有确认需要改变服务端协议、数据模型、权限所有权或跨模块责任的问题。这里的“通过”只代表计划可执行，不代表未来实现无缺陷。

## 5. 与设计和方法的差异

原设计 15 粗任务细化成 16 个，不改变范围；设计 §12、§15 同步。plan 提供具体 Files/Interfaces、步骤、命令、期望行为与变异，不提前伪造尚未生成代码的 hash、测试次数或整段原型。授权实现后逐 Task 保存实际输入、失败层、恢复和回归。

本期最后模块实例为 search-menu 的 WorkspaceService；此前 13 路径的实际清理归属为 P11 九处、P12 一处、P14 两处、P15 一处，不照旧分配重复施工。公开 editor ref、M7 四 view 方法、旧领域适配继续保持明确合同。

## 6. 移交事项

M1-P3、M1-P5、M2/M3 closeout 只准备本期落点，顶部 status 继续 open；本次不追加虚假的实施关闭证据。T15/T16 实施后追加各子项事实；M4 真正收尾才统一 closed、逐条勾设计 §14、改总体设计 9.4 和建立 M5–M8 的 M4-closeout。

P16 全局 dead/no-shadow、C12 和总守卫；M5 文件/图片；M6/M7 自身页面、收集箱来源/系统创建者、版本旧 DTO、M7 lazy pending discard；其它领域搜索均保留原归属。

## 7. 已知限制与待验证

评论和菜单键盘、管理员删别人、表情 CSP、完整草稿 UI、搜索乱序和 cmdk、LAN HTTP 剪贴板、三浏览器都仍待产品实现和测试。远端旧编辑/删评论超出增量窗口须关闭重开才读取；原 API 空 q 最近项不等于命令首页要列近期项。没有扩大产品承诺或将已有测试当作这些行为的替代证据。

## 8. 静态预检与准备交付

主目录 .superpowers/sdd/P15-preparation-20261031/ 留存 builder-check.md、source-review-inventory.json、path-inventory.json、document-links.json、precheck-static.json 及 raw。只核文档、路径、合同、现有基线和关闭证据；16 个 A、16 Task 共 80 步全部未勾，tools/keywords.json 仍 M4/P14。

静态预检 exit 0：本分支只改 16 份 Markdown 文档；88 个确定路径中 73 个已存在、15 个明确待新增；386 个本地链接没有新增错误。设计中原有的 M5–M8 四个未来 M4-closeout 链接保持待收尾，不伪造文件消除检查。主目录 main 与 origin/main 均为上述基线且干净，P14 准确 main CI 和清理回执及三个归档 manifest 的 hash 已核对。最终文稿复核输出另存 precheck-static-final.log。

准确文档提交和远端 HEAD 以 precheck-delivery.json 后置记录，主目录仍是通过 CI 的 main，P15 分支/工作树保留给后续 T1。最终阶段整分支审查与 CI 在实施后执行，本次不宣称已完成。
