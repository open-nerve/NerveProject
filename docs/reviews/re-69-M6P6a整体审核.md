# M6 / P6a 整体审核

> 2026-11-13：Ready after fixes。
> - **目标**：分支 `codex/m6-p6a` 相对 main `aadaf61a5929b57d8aa53b93e5ae04b03cc046c6` 的全部修改（`aadaf61a..7d355551`，`45cb4482` 至 `7d355551` 的 18 个提交）；修复轮 F（`7b347f58` 起的六个提交，F1–F6）另经一次限定复审（§4）。
> - **依据**：[P6a spec](../v0/M6-cycles-modules/specs/P6a-web-modules.md)、[P6a plan](../v0/M6-cycles-modules/plans/P6a-web-modules.md)、[预检 re-68](re-68-M6P6a预检.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。
> - **阶段记录**：由 T16 在合入时写（P6a 的阶段记录），届时在此补链接。

## 1. 方法

一次 fresh 只读审查（`fable`，台账 Task 16），范围 `aadaf61a..7d355551`（T1–T15 的全部提交），不改动文件与 git 状态。它对照的依据见上；门的一致性对照 nerve 的 `access/domain/decide.go` 与 `lock.go`，文档的准确性对照差异清单与回执引用的 Plane 源码（主目录的只读 `plane/`）。

原始报告没有另存文件；结论、核对过的性质与各条发现由主控转述，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **数据层**：模块 store 的写入与重放正确；后续动作（提示、关闭弹窗、跳转、重读）只在 nerve 确认之后。
- **门**：每一处调用的取数门与写的门都与 nerve 的 `decide.go`、`lock.go` 一致。
- **会话与路由**：新代码不读 router store；SWR 只经 `useSessionSWR`。
- **删除**：旧的模块代码删除完整。
- **守卫**：`old-module-urls`；`project-address-route` 及其嵌套解构的一支；`copy-hook` 的例外降到 7；en 与 zh-CN 的文案键成对。
- **回归边界**：被改动的回归测试都在 plan 的“回归测试的边界”之内。
- **e2e**：模块的页面故事照 P5b 的教训写。
- **文档**：交接回执、差异清单、前端改动清单与其中 Plane 的引用准确。

## 3. 结论与处理

结论 **Ready after fixes**：没有 Critical、Important，三处 Minor 与一条观察，没有须负责人决定的事。修复轮 F 处理三处 Minor（F3–F5），另做主控在 T14、T15 复核时留下的两项（F1、F2），观察记入 M8 的交接（F6）：

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | `web/apps/web/core/components/modules/index.ts` 还转发 `./select`、`./form`、`./quick-actions`、`./progress-sidebar`，没有经它读的；它唯一的读者是 `modals/project-level.tsx` 的 `CreateUpdateModuleModal` | F3 照迭代的先例（没有 `cycles/index.ts`，按路径导入）删去桶文件与只被它转发的 `select/index.ts`，`project-level.tsx` 按路径导入（`a94a7b79`） |
| M2 | 模块页头的搜索框与迭代页头的约 45/57 行相同；`toggled` 在两个筛选根里一字不差 | F4 把 `toggled` 移到 `@nerve/utils` 的 `toggledOptions`（加单测）；迭代页头的测试不改断言即通过，于是把搜索框抽成 `core/components/planning/list-search.tsx` 的 `PlanningListSearch`，两个页头共用；两处各有让迭代与模块的测试同时失败的变异（`a41dd2b8`） |
| M3 | 模块的故事与夹具经 `e2e/fixtures/cycle-pages.ts` 的 `cycleMenu` 找模块的菜单，命名泄漏 | F5 改名 `itemMenu`，迭代与模块的调用者只改名字与导入行，不留别名（`7b2cb1db`） |
| 观察 | 模块一侧用了两份 `useLocalStorage`：`use-module-display.ts` 读 `@nerve/hooks`，`module_sidebar_collapsed` 与 `progress-tabs.tsx` 读应用的副本 `@/hooks/use-local-storage`；副本在 P6a 之前就有，是全应用的重复 | F6 记入 [M6-M8-11](../v0/M8-open-release/handoffs/M6-closeout.md#M6-M8-11) |
| 主控（T14） | G9 页面故事仍有自己的 `listedOn` 副本 | F1 改为导入夹具的 `listedOn`，`linkedModules` 去掉导出（`7b347f58`） |
| 主控（T15） | M4 交接 M6-04 的模块名字输入法、M3 交接第 3 节的共用组件焦点轮廓，T15 写回执时发现没有核对、交给了 P8 | F2 照 P5a 修复轮 F2：G9 的输入法页面故事 `g9-module-name-page.spec.ts`（Chromium 真组字、三引擎派发两种次序）；G19 故事三以 `focusOutline`（从 M4 C12 的 `visibleFocus` 抽出，M4 的故事改调它）量页头、行、卡片与详情页页头的 `Button`、`IconButton`、`getButtonStyling`、`CustomMenu`；RED 4 个、产品变异 4 个；spec §5、两份回执随之改写（`dbbefbd6`） |

F2 逐个 Tab 量下来，这四种共用控件之外还有焦点轮廓被裁掉或没有轮廓的控件（行的链接、侧栏链接一行的链接与 “Add link”、面包屑的链接；日期范围按钮与进度页签），迭代页同样，都是两域或全应用共用的件，超出修复轮，写进 [P6a spec](../v0/M6-cycles-modules/specs/P6a-web-modules.md) §5 的 P8。

## 4. 修复轮 F 的复审

（待写：修复轮 F 之后接着做一次限定复审，结论与处理由主控填写。）
