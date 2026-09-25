---
status: open
from: M1/closeout
to: M8
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint、包的 license 字段、AGPL 的验收、中文覆盖

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1350 个（成员 898、prop 452），属于本 M 领域（Webhook）的有 **6 个**（成员 6、prop 0）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M8`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M8` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## 不属于某个领域的死成员和死 prop

共享的部分（`packages/propel`、`ui`、`types` 的通用类型、`utils`、`constants`、`hooks`、`i18n`，web 的通用组件、侧边栏、命令面板等）在 M1 收尾结束时有 **287 个**（成员 104、prop 183），列出方法同上，把 `--rows M8` 换成 `--rows shared`。M2–M7 改到这些文件时照上面的做法清掉；剩下的由 M8 在发布之前清完。

- **关闭条件**：发布之前，`domains.mjs … --rows shared` 列出的每一行都已消失，或者写进 M8 的 review（文件、名称、谁在读或传它）。

## 包的 `license` 字段

来自 Plane 的 13 个 `package.json`（web 应用和 12 个 `@nerve/*` 包）写的是 `"license": "AGPL-3.0"`，这是 SPDX 已经废弃的标识；Nerve 自己的根目录、`api-client`、`e2e` 写的是 `AGPL-3.0-only`，与根目录的 `LICENSE` 一致。它和版本号（M1/P5 的交接）是同一批发布元数据，M8 定版本号时一起改为 `AGPL-3.0-only`。

- **关闭条件**：`git grep -n '"license": "AGPL-3.0"' -- '*package.json'` 没有输出。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 694 个（web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
- **清零**：发布之前全部清零，然后 `.oxlintrc.json` 把警告改为错误、各包的上限删除（总体设计 7.6 的原始要求）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化；发布之前 oxlint 没有警告，警告即错误。

## AGPL 的三项义务和发往第三方的请求

Codex 对整个 M1 的对抗评审（[报告](../../M1-frontend-trim/reviews/M1-codex-adversarial-review.md)的 Important 3）指出，公开部署的验收还没有覆盖 AGPL-3.0 的三项具体义务：
- **第 5(a) 条**：修改过的作品要带着显著的修改说明和相关日期。README 有版权和许可证，[前端改动清单](../../frontend-changes.md)按类别记录了改动，但 M1 改过的 web 文件没有逐个的"修改及日期"标识，也没有文档说明集中的说明是否足够；
- **第 5(d) 条**：有交互界面的，界面上要显示适当的法律声明；原程序的界面本来没有这种声明时有例外；
- **第 13 条**：通过网络与运行版本交互的用户，要能取得这个运行版本的对应源码。帮助菜单打开仓库，但不能证明运行的是哪个提交，也不包括 `@makeplane/propel` 的对应源码（[M1/P5 的交接](M1-P5-brand.md)）。

另外有两类请求发往第三方的 jsDelivr（M1 之前就是这样）：
- 表情选择器（`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx` 的 `EmojiPicker.Root`，来自 `frimousse` 0.3.0）没有传 `emojibaseUrl`、`emojiVersion`，`frimousse` 于是从 `https://cdn.jsdelivr.net/npm/emojibase-data@latest` 取表情数据，版本不固定；
- 标注块的默认图标是 `https://cdn.jsdelivr.net/npm/emoji-datasource-apple/img/apple/64/1f4a1.png`（`web/packages/editor/src/extensions/callout/utils.ts:19`）。

条款怎样适用（例如集中的修改说明是否足够、5(d) 对原版界面的例外）是法律解读，由负责人在本 M 定。
- **时点**：任何对外的网络部署之前，最迟本 M 发布；propel 的对应源码（M1/P5 的交接）也按这个时点，不只看本 M 的排期。
- **关闭条件**：
  - 三条逐项验收，结论写进本 M 的 review，负责人的解读写明；
  - 页面上的源码入口指向运行的提交，并能取得完整的对应源码（含 propel）；
  - jsDelivr 的两类请求改为自托管的固定版本，或者明确告知用户并接受，写进 review。

## 界面文字的中文覆盖

M1 收尾的浏览器核对看到 zh-CN 界面上有英文（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 2.10 的 I2）。来源是 Plane 原有的、没有经过 `t()` 的英文字面量：`web/apps/web` 的 `.tsx` 里粗略找到约 457 处，最多的是工作项、项目、模块、导览、收集箱、迭代，例如 "First day of the week"、"Search commands..."、活动迭代卡片上各状态组的 "Completed"、"3 Work items"。另一种来源（zh-CN 的值与英文相同）已在收尾里译完，剩下的是不用翻译的 URL、ID、Webhooks 和示例邮箱。
- **谁改谁接**：M2–M7 把改到的界面文字接入 `t()`，中英文一起写；
- **关闭条件**：本 M 发布之前 zh-CN 界面上没有未翻译的界面文字，核对的方法（脚本或浏览器核对）写进本 M 的 review。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节，2.9 的 Important 3、2.10 的 I2。
