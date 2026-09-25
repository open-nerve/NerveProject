# M1 收尾：评审记录

| 项 | 内容 |
|---|---|
| Phase | M1 收尾 `closeout`（M1 的最后一个 Phase；它合并之后 M1 完成） |
| 日期 | 2026-09-25 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/closeout.md) / [plan](../plans/closeout.md)（控制者评审补充和各次裁定在 plan 的 Tasks 之前） |
| 分支 | `worktree-m1-closeout`，从 `main` 的 `c493255` 分出；`26cdb33` 把 `main` 上 Codex 对整个 M1 的对抗评审报告（`d2bd4fc`）合并进来 |
| 评审 | 各 Task 评审（sonnet，第 2 节）；整分支评审（opus，`c493255..26cdb33`）：Ready to merge with fixes，0 Critical、0 Important、7 Minor（第 3 节）；修复轮之后的范围复核（sonnet，`26cdb33..5557c50`）：Approved，没有新的发现（3.1）；Codex 对整个 M1 的对抗评审（评审对象 `c493255`）：7 条新发现，处理结果写在[报告](M1-codex-adversarial-review.md)第 10 节（本评审第 4 节） |

## 1. 范围与结果

收尾把 P1–P5 的评审留给它的事项做完（M1 设计 9 节）：knip、tsc 看得见的死代码和没人用的包导出删完，死文案、死图片删完，守卫认识收尾、每个分支都有命中样本；`window.open` 加固；`sanitize-html` 换成浏览器自带的 `DOMParser`；测试输出和开发服务器的噪声从依赖上去掉；来源查不到的封面照片和截图里的真人换掉；Codex 对整个 M1 的对抗评审找到的 4 个问题在 M1 内修掉，其余写进后续 M 的交接。

| Task | 去掉的 | 改成的 |
|---|---|---|
| 1 | 守卫不认识 `M<n>/closeout`；42 个分支没有命中样本，50 个不命中样本引用不存在的代码 | 收尾的 Phase 由工具检查到期；新增 41 个命中样本，不命中样本替换 60 个、删 5 个，都引用保留的代码（跟进 `483ca9d` 让 5 个样本落在规则停止命中的地方） |
| 2 | 16 处 `window.open` 里 12 处没有 `noopener` | 都带 `noopener,noreferrer`；规则 `window-open` |
| 3 | 从不渲染的应用栏（8 个文件） | 规则 `app-rail`、`app-rail-files` |
| 4 | 512 个没有别的文件读取的包导出（三轮，166 个文件），`tlds.ts` 和它的两条例外，因此没人用的 22 个键，propel 的 `cmdk` 依赖 | — |
| 5 | P2–P4 评审点名的死代码；菜单项里的 `window.close()` | base-ui 自己关闭菜单；`no-restricted-globals` 禁止 `close` 等全局名 |
| 6 | 80 处 `={"…"}` 等退化结构、208 行悬空和 4 对重复的导入分组注释、装饰插图的错误 `alt`、`TPlacement` 的类型洞 | `react/jsx-curly-brace-presence` 为错误级别；`alt=""` |
| 7 | 标注块属性按数字读取（`128161` 丢了字符串） | 按 HTML 里的字符串读取，带测试 |
| 8 | `sanitize-html` 和只有它用的 16 个包 | HTML 工具改用 `DOMParser`；通知预览不再显示 `&amp;`；开发服务器的外部化警告 44 → 0 |
| 9 | 编辑器测试的 5 行 sourcemap 警告 | `prosemirror-codemark` 的 pnpm 补丁删掉 12 行悬空的 `sourceMappingURL` 注释 |
| 10 | 格式检查不覆盖两个配置包和根目录的配置 | `check:format` 覆盖它们（根目录的 `fix:format` 在修复轮补上） |
| 11 | 500 个没有代码引用的文案键；主题选项的两个标签来源 | 主题的名字只来自 `THEME_OPTIONS` |
| 12 | 133 张没有被导入的图片（6.1 MiB） | — |
| 13 | — | 前端改动清单 1.6 节、README、17 份交接补关闭条件、7 份新交接 |
| 14 | 29 张来源和许可查不到的封面照片（JPEG） | Nerve 自己画的 29 张抽象图（SVG 源文件和 WebP） |
| 15 | 11 张保留的图里 77 个照片头像和真实的人名；8 张不显示的图 | 字母头像和中性的名字 |
| 16 | `handler_test.go` 编造的夹具文件名 | 构建真实产出的文件名 |
| 修复轮 | 任务评审和浏览器核对发现的 7 个问题（FW1–FW10） | 见 spec 3.16 和本记录第 2 节 |
| 17 | 粘贴编辑器自己的剪贴板类型时在活动文档里解析（Codex Critical 1） | 在 `DOMParser` 的惰性文档里标记要复制的图片 |
| 18 | 命中 Turbo 缓存的构建带着陈旧的文件（Codex Important 1） | `make build-web` 先清空 `web/apps/web/build/client`（跟进 `e070065`：`make build` 复制时也读 `WEB_CLIENT`） |
| 19 | 活动迭代卡片把取消的工作项算进分子（Codex Important 2） | 用共享的 `calculateCycleProgress`，文字走 `t()`（跟进 `f9d1ab2`：分组行是带 key 的列表项） |
| 20 | 保留的截图里 10 处已删的功能（Codex Important 4） | 按背景涂掉 |
| 整分支评审之后的修复轮 | 5 个没翻译的中文值（FW14）、`services` 公开的 `ensureAPITrailingSlash`（FW15，Codex Minor 1）、一个错标的导入分组注释（FW16）、根目录格式检查的文件列表写了两份（FW17） | 见第 3 节 |

数字（收尾之前 `c493255` → 收尾结束）。逐项的口径和每个 Task 结束时的中间值在 [spec](../specs/closeout.md) 3.2，这里只列主要的几项：

| 项 | 收尾之前 | 收尾结束 |
|---|---|---|
| 关键词守卫 | 48 条规则、5 条例外，顶层 `phase` 为 `M1/P5` | **51 条规则、3 条例外**（`analytics` M6、`project-invitations` M3、`brand` M9），顶层 `phase` 为 `M1/closeout`；`alts.mjs`、`missquote.mjs`、`missreal.mjs` 都是 0 |
| `make lint-web` / `check:types` / `make test-web` / `make build-web` | 52 / 23 / 16 / 11 | 54 / 23 / 16 / 11 |
| vitest | 77 个测试，stderr 有编辑器的 5 行 | **101 个**，没有 stderr |
| oxlint 上限合计 | 711 | **694**（3.3 的表在第 9 节） |
| 包导出 / 死成员 / 死 prop（`deadsym.mjs`） | 426 / 1150 / 638 | **2 / 898 / 452**（2 个导出是 `api-client` 的类型测试；成员和 prop 交 M2–M8） |
| 每种语言的键 / 没有引用的键 | 1599 / 482 | **1079 / 0** |
| 图片 / 没有引用的 | 246 / 133 | **134 / 0** |
| 导入分组注释：悬空 / 重复 | 208 / 4 | 0 / 0 |
| 不带 `noopener` 的 `window.open` | 12 | 0 |
| 开发服务器的外部化警告（浏览器 / 服务端） | 44 / 44 | 0 / 0 |
| 构建体积合计 | 534 个文件，17,306,349 字节 | **531 个，14,763,091 字节**（P1 之前的 32,232,132 字节少 54.2%） |
| 锁文件 | — | 删 17 个包，不新增、不升级 |

- **改动规模**：`c493255..5557c50`（本评审提交之前）`893 files changed, 8993 insertions(+), 13031 deletions(-)`，其中 Codex 报告 +354 行。不算 `docs/` 是 **865 个文件，+5956 / −13029 行**：删 345 个文件，新增 65 个（29 张封面的 SVG 和 WebP、3 个 `SOURCES.md`、3 个测试文件、1 个补丁）。本分支自己的提交 43 个：20 个 Task、3 个跟进、修复轮 8 个代码提交、整分支评审之后的修复轮 4 个，文档提交 8 个。
- **依赖**：删 `sanitize-html` 2.17.7、`@types/sanitize-html` 和只有它们用的 15 个包，propel 不再依赖 `cmdk`（web 应用自己仍依赖它）；`utils` 的开发依赖加 `jsdom`（锁文件里已有的同一版本）；`prosemirror-codemark@0.4.2` 带补丁。没有新增或升级的包（spec 3.5，整分支评审复核）。
- **保留行为的核对**（M1 设计 7.5）：控制者的浏览器核对跑了 4 轮（T5、T8、修复轮、T20 之后），每轮都在基线 `c493255` 的构建上做反向对照。第 4 轮在 `ff555bf` 的构建上 **1147 项全部通过**；基线上 1067 项通过、80 项失败，失败的正好是收尾改掉的行为（附录 7.2）。它包括重跑的 P2–P5 的场景（P4 的 9 段 403 项、P5 的 `brand.mjs` 177 项）。整分支评审之后的修复轮只改了 5 个中文值、一个包的导出、一行注释和根目录的脚本，没有再跑浏览器核对（7.2）。
- **文档**：前端改动清单 1.6 节 30 行；README；P2–P5 给 M2–M8 的 17 份交接补关闭条件；M2–M8 各一份收尾交接（死成员和死 prop、oxlint，以及各自的几项）；Codex 报告的处理结果；M1 设计 3.1、11 节、12 节，总体设计 9.4（本评审提交）。

### spec 5.2 的验收标准

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | 20 个 Task 各一个提交（另有 3 个跟进），修复轮 10 个提交；每个提交结束时五个门禁通过 | **满足**。每个 Task 的实现者跑了五个门禁；控制者另在本分支的 31 个代码提交上逐个重算中间值（`branchrun.mjs`，与 spec 3.2 的表一致，两处差别是文档写错了，已改，见第 6 节）。整分支评审之后的修复轮 4 个代码提交各自跑了门禁 |
| 2 | `keywords: 51 rules, 3 exceptions, no hits.`；上限合计 694；`alts`、`missquote`、`missreal` 为 0 | **满足**（整分支评审复跑） |
| 3 | `make test`、`make lint-go` 通过；`t16/names.mjs` 为 0 | **满足**（T16） |
| 4 | vitest 101 个，没有 stderr；T17 的新用例在旧代码上失败 | **满足** |
| 5 | 每个 Task 的孤儿核对 | **满足**。`symref`、`dangling`、`keyref orphaned` 都是 0，`headers.sh` 没有输出；`deadorph` 只有登记的 4 行（T7 的 `parseHTML`、FW4 的 `TLogoProps.emoji.url`、`9a98148` 的 `JSONContent.content`、`.text`）；`infile-orphans` 的每一行都以 `defined now: 0)` 结尾（T3 12 行、T4 166 行、T5 4 行） |
| 6 | `deadsym` 2 / 898 / 452；键 1079；图片 0 / 134；注释 0 / 0；`jsx-curly-brace-presence` 0 | **满足** |
| 7 | 锁文件与 3.5 相同；`devwarn.mjs` 为 0；构建体积与 3.4 相同 | **满足**。整分支评审在 `26cdb33` 上重新构建，531 个文件、14,763,098 字节，与 3.4 当时的一列逐项相同；FW14 之后是 14,763,091 |
| 8 | `t18/prove.sh` 冷、热构建都没有事先放进去的文件，旧 `Makefile` 上热构建留下它 | **满足**（T18，评审复核日志） |
| 9 | 浏览器核对全部通过，反向对照在基线上失败的正好是收尾修掉的；S1–S4、持续集成通过 | **满足**。浏览器核对见附录 7。持续集成在本分支和合并之后的 `main` 上各跑一次 |
| 10 | 前端改动清单 1.6 节；README、交接 | **满足**。清单 30 行（spec 5.2 写的 26 行加上整分支评审之后修复轮的 4 行） |
| 11 | 新画的和改过的图片按能看清 20 px 细节的尺寸看过，哈希与 plan 相同 | **满足**。T20 的 `intake-light.webp` 按裁定与原型不同，plan 写的是本分支的哈希 |

## 2. 各 Task 的评审

- 实现者都用 Opus 5.5，各 Task 的评审者用 sonnet，整分支评审用 opus。下一个 Task 的实现者与上一个 Task 的评审同时开始，评审都钉在被评审的提交上。
- 原型（`$COTMP/proto`：T1–T13、T14–T16、T17–T20 三批）是对照答案，不整体搬运：每个 Task 逐步执行，报告与原型逐文件比较，每处差异写明原因（第 5 节裁定 1）。
- 每份报告回答 plan 的风险点：保留的东西没有失去最后一个读取方、依赖没变、没有新的共享可变状态、链接加固、版权声明按来源。

| Task | 提交 | 核实方式与评审 |
|---|---|---|
| 1 | `aea47e4`、`483ca9d` | 实现者 DONE，树与原型 `c9eeb28` 相同。控制者要求把 5 个不命中样本移到规则停止命中的地方（跟进提交）。评审 1 个 Important：命中样本 `import Image from "next/image";` 在仓库历史里没有原文。**裁定**保留：P1 为 `next/script`、`next/image` 两个垫片一起设计了这条规则，`next/image` 的垫片存在但没有导入方，所以不存在真实的导入行；这是登记的 3 个示意样本之一（第 5 节裁定 2） |
| 2 | `1cfc629` | 实现者 DONE（原型 `5eb9494`）。评审 Approved，没有发现：12 处都不读返回值 |
| 3 | `4e1893c` | 实现者 DONE（原型 `7b802a8`）。评审 Approved；提出 propel 右键菜单的 `Trigger`、`Content` 仍从 base-ui 继承 `className`（交修复轮，FW1） |
| 4 | `7f189da` | 实现者 DONE：原型 `cccf8be` 加 6 处手改（一个多余的行尾注释、两个分组标签、一个标题、一个空行、两处写着已删的 `THEMES` 的注释），因此 `infile-orphans` 是 166 行，原型是 165。评审 Approved，没有发现（按类别抽查，另找脚本的副作用） |
| 5 | `4cf483f` | 实现者 DONE（原型 `30c798d` 加 T4 的 6 处）。评审 Approved；另记一件交 M4 的事：工作项弹窗的描述不读 `isMigrationUpdate` |
| 6 | `20ac5b6` | 实现者 DONE：原型 `17becf9` 加 4 处只改注释的差异（悬空的 `//`、一个 `//hooks`）。评审 Approved；另有 4 行只有 `//` 的标签和一个错标（交修复轮，FW3） |
| 7 | `acdbf50` | 实现者 DONE（标注块的文件与原型 `344ab76` 相同）；`deadorph` 1 行是 `parseHTML`（已知，测试证明被读取）。评审 Approved；提出本地存储的图标不经检查就被信任（交修复轮，FW4） |
| 8 | `d25f08c` | 实现者 DONE（与原型 `ec6c3e5` 逐字节相同）：`devwarn.mjs` 基线浏览器 44 条、服务端 44 行，之后 0 / 0；实测 `DOMParser` 的文档是惰性的；对编辑器的 HTML，`isEmptyHtmlString` 新旧结果相同。评审 Approved；草稿弹窗只把 `img` 算作内容，确认是基线就有的问题（交修复轮，FW5）；标注块的 Markdown 序列化不转义（交 M4） |
| 9 | `98ecc86` | 实现者 DONE（补丁和锁文件与原型 `2d51151` 相同；注释改正三处）。评审 Approved；Minor：补丁后 `utils.js` 末尾没有换行，生成文件，不影响 |
| 10 | `263aef9` | 实现者 DONE（原型 `3cb038f`；`make lint-web` 54 个任务）。评审 **Changes requested**（Important）：根目录没有 `fix:format`，README 说的"格式化每个包"不成立（修复轮 FW7） |
| 11 | `f9fb015` | 实现者 DONE（文案、`themes.ts`、主题开关与原型 `b13b3d8` 相同；`i18n-applications` 的不命中样本改到规则读取的文件）。评审 Approved：抽查 55 个删掉的键，没有活的 |
| 12 | `4797772` | 实现者 DONE（133 个删除与原型 `3ebfe70` 相同；不带缓存的构建通过）。评审 Approved |
| 13 | `300eff0` | 实现者 DONE（26 个文档文件；前端改动清单改正 5 处，让它描述本分支）。评审 Approved；"512 个导出：274 删、283 不导出"的单位不一致（修复轮的文档提交） |
| 14 | `89244e3` | 实现者 DONE（与原型 `131e1de` 逐字节相同）。评审 Approved：SVG 不在上传允许的类型里，保存的是上传副本的地址 |
| 15 | `e349924` | 实现者 DONE（图片、`SOURCES.md`、`root.tsx` 与原型 `f997e8a` 相同；M5 交接里新建项目的请求顺序改正）。评审 Approved |
| 16 | `985675a` | 实现者 DONE（与原型 `3c64669` 相同；变异测试证明 `/icons` 保住了回退分支的检验能力）。评审 Approved |
| 修复轮 | `1896bca`、`c167005`、`28bedba`、`773b3d8`、`6f2b484`、`4ea6397`、`827823c`、`6048ad3`、`4be6cba`、`9a98148` | FW1 propel 右键菜单不接受 `className`；FW3 空的和错的导入分组标签；FW4 标注块检查本地存储里的图标（14 个测试）；FW5 草稿弹窗和 `isCommentEmpty` 用同一份"算作内容的标签"；FW6 个人主页窄屏的菜单按钮显示标签名；FW7 根目录的 `fix:format`；FW10 `isCommentEmpty` 走不到的 JSON 分支删除；`9a98148` 把 `JSONContent` 的三个字段恢复（第 5 节裁定 5）；两个文档提交。每个提交跑了门禁和孤儿核对，由整分支评审一并审 |
| 17 | `7836c7a` | 实现者 DONE（编辑器的 4 个文件与原型 `bad3483` 相同）：新用例在旧代码上失败，Chromium 里的复制探测 22 / 0（基线 21 / 1，C3 执行了 `onerror`）。评审 Approved（对抗性审查）：逐条追了每个粘贴和拖放的入口；schema 只留声明过的属性；链接拒绝 `javascript:` 等；重新序列化之后 ProseMirror 解析的结果不变（表格、代码块、自定义元素、实体）。另确认复制资源的接口要核对权限（交 M5） |
| 18 | `fc3069a`、`e070065` | 实现者 DONE_WITH_CONCERNS（`Makefile` 与原型 `b85f861` 相同）：`prove.sh` 在旧 `Makefile` 上重现，修复后冷、热构建相同；实际跑了 `make build`。控制者要求 `make build` 的复制也读 `WEB_CLIENT`（跟进提交）。实现者更正原型的说法"持续集成里清空是空操作"（e2e 任务的第二次构建走热路径）。评审 Approved；评审者说 `rm -rf` 没有参数时会失败；控制者在 macOS 上实测 `rm -f` 没有参数时退出码为 0（GNU coreutils 的 `rm -f` 同样），实现者的说法成立 |
| 19 | `c84f26d`、`f9d1ab2` | 实现者 DONE_WITH_CONCERNS（与原型 `a994925` 只差收掉一层 `<span>`）：四种计数下卡片的数字用真实的函数和文案算过。控制者要求收掉分组行没有 key 的片段和它外面的空 `div`（跟进提交，React 的 key 警告 1 → 0，web 的上限 566 → 565）。评审 Approved；卡片、侧边栏、模块的口径不一致和快照的条件交 M6 |
| 20 | `ff555bf` | 实现者 **BLOCKED**：原型 `intake-light.webp` 那一框用纯白填，抹掉了卡片底边 650 px，在卡片外的透明处画了白块，应用里看得见。控制者看过三张对照图，裁定改为重复框左边的一列（第 5 节裁定 6），其余 8 张与原型逐字节相同。实现者看了 20 张前后对照和 12 张图的全部条带，没有别的已删功能。评审 Approved，没有发现（自己裁剪、看了 10 个框和全部条带） |
| 整分支评审之后的修复轮 | `e171d64`、`df4cbf7`、`8a6bea5`、`73f8c36`、`5557c50` | 见第 3 节 |

文档提交：`47ce708`（spec、plan）、`4a354c9`（控制者评审补充）、`dcefebe`（T14–T16 加入 plan）、`827823c`、`4be6cba`（修复轮）、`5ed69fc`（T17–T20 加入 plan、Codex 的分诊）、`14684db`（T17–T20 的清单行和数字）、`5557c50`（整分支评审之后）。

## 3. 整分支评审（opus）：Ready to merge with fixes

评审范围 `c493255..26cdb33`，结果 **0 Critical、0 Important、7 Minor**。评审者读了每个代码文件的差异，逐条核对了 spec、前端改动清单和交接里的数十条具体说法，并复跑门禁：`check:types --force` 23、守卫、vitest 101（编辑器 35、utils 29）、`make knip`、`check:sync`、各包上限（694）、在 `26cdb33` 上构建（531 个文件、14,763,098 字节，与 spec 3.4 相同）、`linttable.mjs`（与 spec 3.3 逐字节相同）；从 `c493255` 起的全部孤儿核对。评审者自己写了几个只读脚本（`$COTMP/final-review/`）：
- `delkeys-check.mjs`、`litkeys.mjs`：删掉的 522 个键与现在每个拼出键的模板、每个等于某个删掉的键的字面量比对，没有活的；159 个不是字面量的 `t()` 参数都追到常量；
- `hitreal.mjs`：T1 新增的 41 个命中样本在历史里都有原文，只有登记的 3 个示意样本例外（由此发现 R7）；
- `degen-touched.mjs`：本分支加上或改过的行里没有退化结构；`nervelabels.mjs`、`shrunk.mjs`：导入分组标签只剩 R2 一处。

安全方面：`web/` 里没有 `dangerouslySetInnerHTML`；剩下写 `innerHTML` 的只有编辑器里的常量 SVG；`paste-asset.ts` 读的是 `DOMParser` 文档的 `body.innerHTML`，ProseMirror 会在自己分离的文档里再解析一次；`pasteHTML` 发出的合成粘贴事件没有 `clipboardData`，处理函数返回 `false`，不会重入。

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| R1 | Minor（文档） | M2 的 P2 交接引用 T4 删掉的 `THEMES` | 已修（`5557c50`）：引用现在的注释，列出 `THEME_OPTIONS` 的 5 个值 |
| R2 | Minor | `workspace/content-wrapper.tsx` 的 `// nerve imports` 标着一个本地导入（T3 删掉 `cn` 和应用栏的导入之后留下） | 已修（FW16 `8a6bea5`）：改为 `// components` |
| R3 | Minor（一个值一个来源） | 根目录 `package.json` 的 `check:format`、`fix:format` 各写一份 7 个路径的列表，README 第三份 | 已修（FW17 `73f8c36`）：`check:format` 是 `pnpm run fix:format --check`，README 指向脚本。修复轮的简报当时认为两份是"仓库的惯例"，是错的：各包写的是 `.`，不是列表（第 5 节裁定 7） |
| R4 | Minor（文档） | spec 第 6 节"160 个只有一个子元素的片段"，T19 的跟进删掉一个，是 159 | 已修（`5557c50`） |
| R5 | Minor（文档） | spec 3.2 说 T19 的文字"页面上没有跑"；第 4 轮浏览器核对其实跑了（23 / 0）。这项核对也不在 spec 3.14 和 plan 的核对清单里 | 已修（`5557c50`）：改正那一行；3.14 加上这项和第 4 轮的合计；plan 的核对清单加第 15 项，反向对照改为第 16 项 |
| R6 | Minor（文档） | spec 3.1 的 T15 写的是重放的数字（+63 / −95），提交是 +64 / −96 | 已修（`5557c50`） |
| R7 | Minor（文档） | spec 3.6 "命中样本都取自真实代码"只对 T1 新增的 41 个成立；P1–P5 写的 39 个在本仓库的历史里没有原样的一行 | 已修（`5557c50`）：限定为 T1 的 41 个；另写明 39 个里 23 个是 P5 改名之前的行，16 个本仓库没有（第 5 节裁定 2） |

评审者还指出控制者给出的第 4 轮浏览器核对的合计算错了（写成 928 / 0、基线 836 / 92：基线 B 组的日志把每行失败打印了两次，合计也漏了两段）。按每个脚本的摘要重算是本分支 1147 / 0、基线 1067 / 80，本记录和 spec 3.14 用的是重算的数字。

整分支评审之后的修复轮（`26cdb33..5557c50`，4 个代码提交和 1 个文档提交）另外做了两项已定的事：
- **FW14**（`e171d64`）：zh-CN `common` 的 5 个值还是英文（个人、项目、工作区设置侧边栏的分组标题），译为"您的个人资料""开发者""工作结构""执行""管理"，用的是文案里已有的说法；与英文相同的值 11 → 6，剩下的是不用翻译的 URL、ID、Webhooks 和示例邮箱。构建的 zh-CN 语言 chunk 少 7 字节；
- **FW15**（`df4cbf7`，Codex Minor 1）：`@nerve/services` 的 `helpers/index.ts` 只按名字转出 `normalizeAPIRequestURL`，包的公开导出 5 → 4；`url.ts` 保留两个导出（测试用 `ensureAPITrailingSlash`）。

文档提交 `5557c50` 还改了控制者逐提交重算时发现的两处（第 6 节）。修复轮之后：门禁 23 / 54 / 16 / 11、knip 0；vitest 101；`deadsym` 2 / 898 / 452，`deadorph` 对 `26cdb33` 为 0；上限 694；守卫和样本的核对都是 0。

### 3.1 修复后的复核（sonnet，范围复核）

修复轮 `26cdb33..5557c50` 之后，范围复核的结论是 **Approved**，没有新的发现。复核者对每一项自己取证：
- **FW14**：5 个词在 zh-CN 的其他文案里都有同样的用法（"您的个人资料""管理员""您没有执行此操作的权限"），称呼是"您"；两种语言都是 1079 个键；提交只改这一个文件。
- **FW15**：`ensureAPITrailingSlash` 只在 `url.ts` 和它的测试里（测试直接从 `./url` 导入）；包的公开导出是 4 个名字；`app-rail` 规则引用 `url.ts` 的不命中样本仍然成立。
- **FW16**：只改一行注释；`labels-all.mjs` 为 `dangling 0, repeat 0`。
- **FW17**：复核者在 HEAD 上自己跑 `pnpm run check:format`，通过（9 个文件）；`turbo run check:format --dry=json` 列出 `//#check:format`；实现者的反向测试日志显示弄乱 `turbo.json` 之后检查以 1 失败、`fix:format` 修好、检查再通过；README 不再抄列表。
- **文档**：逐项核对多于要求的 12 条，都相符：R1 的 5 个值；R4 的 `frag1 159`、`tpl 197`；R5 的第 4 轮合计，从原始日志逐段重算（本分支 A 组 768 加其余 379 = 1147 / 0，基线 759 / 9 加 308 / 71 = 1067 / 80）；R6 的 +64 / −96 和 M5 交接的 +13 / −2；R7 的 41 / 39 / 23 / 16（重跑 `hitreal.mjs`、`hitplane.mjs`）；T4 的 166 和 `9a98148` 的 `deadorph`（`branchrun.txt`）；3.16 修复轮一段每个提交的行数；3.4 的 14,763,091；第 12–16 项的编号前后一致。
- **门禁**：`check:types` 23、`make knip` 0、守卫、`missquote`、`missreal` 为 0；修复轮前后 `deadsym` 的 1352 行逐行相同（`deadorph` 为 0）。

## 4. Codex 对整个 M1 的对抗评审

Codex 评审的是 `c493255`（P5 合并之后、收尾之前），报告在 `main` 的 `d2bd4fc`，由 `26cdb33` 合并进本分支。结论是"修完列出的 Critical 后可以开始 M2"。7 条新发现由收尾的架构师对照代码逐条核实，全部成立；控制者的分诊因为有一条 Critical 先报给负责人，负责人批准按建议处理（spec 2.9）。

| 发现 | 处理 |
|---|---|
| Critical 1 自定义剪贴板 HTML 在清洗前触发脚本 | **收尾 T17** 修掉；复制资源接口的权限交 M5 |
| Important 1 缓存命中的构建带进陈旧资源 | **收尾 T18** 修掉 |
| Important 2 活动迭代卡片的进度口径 | **收尾 T19** 修掉卡片；侧边栏、模块和快照的口径交 M6 |
| Important 3 AGPL 第 5(a)、5(d)、13 条和 jsDelivr 的请求 | **交 M8**，时点是任何对外的网络部署之前 |
| Important 4 保留的截图显示已删的功能 | **收尾 T20** 修掉（复看后共 10 处，比报告多甘特图、时间线的图标和 "Public" 标签） |
| Minor 1 `ensureAPITrailingSlash` 的公开导出 | **收尾修复轮 FW15** |
| Minor 2 M1 设计 3.1 的"附件节点保留" | **本评审提交**改写 |

逐条的核实、报告第 6 节里"不同意"的各条和第 9 节时点的处理写在报告的第 10 节。报告第 9 节"M2 开始前必须做"的两条和"可以并入 M2"的两条都在 M1 内做完了。

## 5. 控制者的裁定

执行中的裁定都按"能自己定的就自己定"的原则做出：符合 SOLID、从根源解决、不打补丁。升级给负责人的只有两件：spec 第 9 节的待裁定（封面和截图的来源、M1 的终点等，执行前）和 Codex 评审的分诊（有一条 Critical）。

1. **原型只是对照答案。** 每个 Task 逐步执行、逐文件比较。与原型不同的地方都写明原因，实现者发现的原型问题一律按实际改：T1 的 5 个样本、T4 的 6 处手改、T6 的 4 处注释、T9 的注释、T11 的样本位置、T13 前端改动清单的 5 处、T15 的请求顺序、T18 关于持续集成的说法、T19 的 `<span>`、T20 的 `intake-light.webp`。
2. **命中样本取自真实代码，示意的写法登记。** T1 新增的 41 个命中样本里只有 3 个是示意的写法，因为守卫看住的写法在仓库里从来没有被跟踪的实例：`import Image from "next/image";`、`storybook-static/index.html`、`web/apps/web/public/stickies.yaml`。P1–P5 写的命中样本里有 39 个找不到原样的一行（23 个是 P5 改名之前的行，16 个本仓库没有）；它们仍然证明规则命中那种写法，不改（整分支评审 R7）。不命中样本全部引用保留的代码，并且在规则读取的文件里（`missreal.mjs`）。
3. **跟进提交由同一位实现者做，改的是本 Task 造成或暴露的东西。** T1 的样本位置、T18 的第二个路径来源、T19 的片段和空 `div`。
4. **收尾途中发现的基线问题：在保留功能上、在收尾的路上的，在收尾修；属于某个 M 的领域的，交那个 M。** 修掉的：草稿弹窗不问就丢掉只有图片或提及的草稿（FW5，数据丢失）、个人主页窄屏显示文案键（FW6）、本地存储里的图标不经检查就被信任（FW4）、React 的 key 警告（T19 的跟进）。交出去的：工作项弹窗的 `isMigrationUpdate`、标注块的 Markdown 序列化（M4）；复制资源的权限、附件图标里的第三方标志、新建项目的封面值（M5）；进度的口径（M6）；主题下拉框的位置（M2）；没有经过 `t()` 的界面文字（M8）。
5. **描述外部格式的类型保持完整。** FW10 删掉 `isCommentEmpty` 走不到的 JSON 分支时，把 `JSONContent` 里只有它读的三个字段也删了；`JSONContent` 描述的是 TipTap 的 JSON，只写一半的字段比有死字段更糟，所以 `9a98148` 恢复它们，与它唯一的使用者 `comment_json` 一起交 M4。
6. **T20 的 `intake-light.webp`。** 原型的纯白把卡片底边抹掉了 650 px，在卡片外的透明处画了白块；控制者看了原型的裁切、实现者的另一种填法和应用里的截图之后，裁定那一框改为重复左边一列（带着卡片的白、底边和下面的透明），其余 9 个框照原型。plan、spec 写的是本分支的字节和哈希。
7. **一个值一个来源，包括"惯例"。** 修复轮的简报以"各包的脚本写自己的目标"为由接受了根目录的两份文件列表，整分支评审指出各包写的是 `.`，不是列表；FW17 收成一份。这条裁定是控制者的错误判断，已改。
8. **`deadorph` 的已知例外。** 脚本只看有类型的读取：`parseHTML` 由 TipTap 读（T7 的测试证明），`TLogoProps.emoji.url` 由运行时的检查读（FW4），`JSONContent.content`、`.text` 按裁定 5 保留。
9. **死成员和死 prop 不在收尾删**（spec 第 4 节第 4 条）：1350 处大多是 Plane 接口类型的字段，M2 起按 OpenAPI 生成的类型整体替换；按领域交 M2–M8，共享的 287 个交 M8，每份交接带列出的命令和关闭条件。
10. **封面和截图**（spec 第 9 节第 1 条，负责人执行前裁定）：来源和许可查不到的不随 Nerve 发布，功能保留，所以换成自己画的封面、去掉截图里的真人；保留的其余图片按能看清 20 px 细节的尺寸复看一遍；附件图标里的第三方标志交 M5。
11. **浏览器核对里测不到的就换一种测法。** 生产构建没有 React 的 key 警告的文字，所以 T19 的核对读 React 自己记录的列表 key；进度条的 100 在基线上也通过（propel 把 125 截到 100），能区分两者的是文字。
12. **执行过程。** 两位实现者各有一次把 `cd` 和只读命令写在一起，没有动到仓库；两个子代理的会话因为网络断开中止，它们的工作在断开之前已经完成，没有重做。控制者在报告第 4 轮浏览器核对的合计时算错了一次（第 3 节），已按每个脚本的摘要改正。

**spec 第 9 节的七条待裁定**（执行前由控制者裁定，第 1、3 条报负责人）：
1. 封面照片和导览截图的来源：替换（T14、T15），保留的其余图片全部复看（裁定 10）；
2. `handler_test.go` 的夹具名：收尾做（T16）；
3. M1 的终点：收尾合并即 M1 完成；已经启动的 Codex 评审的发现分诊进收尾，状态在本评审提交里改（M1 设计 11 节、12 节，总体设计 9.4）；
4. 死成员和死 prop 按领域交 M2–M8（裁定 9）；
5. 删除的图片按 300×200 的联系表看是否足够：足够，删掉的图不再显示给任何人；保留和改过的图按能看清 20 px 细节的尺寸看；
6. 附件图标里的第三方标志：交 M5；
7. Codex Minor 1：放进整分支评审之后的修复轮（FW15）。

## 6. 计划缺陷

| Task | 缺陷 | 实际做法 |
|---|---|---|
| 1 | 5 个不命中样本不在规则停止命中的地方 | 跟进提交 `483ca9d` |
| 4 | 脚本留下写着已删 `THEMES` 的注释和几处格式；plan 写 `infile-orphans` 165 | 实现者手改 6 处；本分支是 166（`5557c50` 改 spec、plan） |
| 6 | 脚本留下悬空的 `//` 和 `//hooks` | 实现者改 4 处注释；另有 4 行由修复轮 FW3 收掉 |
| 8 | `devwarn.mjs` 要仓库的绝对路径、要先 `make build-web`；spec 没写 `<script>`、`<style>`、`<textarea>`、`<option>` 里文字的变化 | `dcefebe` 写进 plan、spec |
| 9 | 补丁的注释写错（"每个 JS 文件"、5 条警告不是 12 条） | 实现者改正 |
| 10 | 根目录没有 `fix:format` | 修复轮 FW7，FW17 收成一份列表 |
| 11 | `i18n-applications` 的不命中样本改到了规则不读取的文件 | 改到规则读取的 `workspace-settings.json` |
| 12 | 按文件名查漏掉 6 张与在用的图同名的死图 | T15 删除，T15 起按路径查（`assetpaths.mjs`） |
| 13 | "512 个导出：274 删、283 不导出"单位不一致 | `827823c`：274 个声明删掉、238 个名字不再导出 |
| 15 | M5 交接里新建项目的请求顺序写错；spec 用了重放的行数 | 实现者改正；`5557c50` |
| 18 | 说持续集成里清空"是空操作"（e2e 任务的第二次构建会清空再恢复）；复制的路径写了两遍 | 提交信息、`14684db`；跟进提交 `e070065` |
| 19 | 分组行没有 key 的片段和空 `div` 没在原型里收掉 | 跟进提交 `f9d1ab2` |
| 20 | `intake-light.webp` 的纯白抹掉卡片底边；"其余 8 张相差不到 600 字节"（`views.webp` 是 812） | 裁定改填法；`14684db` 改为"不到 1,000 字节" |
| Codex 分诊 | 以为 Minor 1 已由 T4 解决 | 架构师核对后更正，修复轮 FW15 |
| 浏览器核对 | 没有 T19 的卡片一项 | 第 4 轮加上（`active-cycle.mjs`），`5557c50` 写进 plan 第 15 项 |
| spec 3.2、3.6、第 6 节 | T19 "页面上没有跑"；"命中样本都取自真实代码"；160 个片段 | 整分支评审 R5、R7、R4，`5557c50` |
| spec 3.2 的孤儿段落 | `deadorph` 报 `content`、`text` 写在 FW10（实际在 `9a98148`） | 控制者逐提交重算时发现，`5557c50` |

## 7. 附录（M1 设计 7.5）

临时核对脚本不进仓库（M1 设计 7.5），全文、运行命令和输出写在这里。

### 7.1 做法

脚本在 `$COTMP/probe/` 下（`$COTMP` = `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-co`），跑在 `$COTMP/probe-app` 里：本仓库的一个克隆，检出到被测的提交，`make build-web` 之后由 node 起静态服务器提供 `web/apps/web/build/client`（单页应用的回退到 `index.html`），用 e2e 包里的 Playwright 驱动 Chromium。路径以 `/api/`、`/auth/` 开头的请求全部由脚本里的桩回答，**当前场景没有列出的请求一律算失败**，每个场景还检查没有未捕获的页面错误。反向对照跑在基线 `c493255` 的构建上（`$COTMP/base`）。

- **A 组**（重跑前面的 Phase）：1. P4 的 9 段（当前项、地址、跳转、命令面板，以及改编的 P2、P3 探测）；2. P5 的 `brand.mjs`；3. 在 1、2 走过的每个页面上找原样显示的键和加载失败的图片（`a3hook.mjs`），另加个人主页的标签在 480 px 和 1440 px 下的显示（`a3-profile.mjs`，第 1 轮发现 FW6 之后加的）。
- **B 组**（收尾自己的改动，plan 的第 4–15 项）：`window.open`、`window.close()`、通知预览、评论和草稿的是否为空、标注块、主题名、工作项表单的迭代、空状态的插图（`closeout-b.mjs`）；预设封面（第 12 项）、改过的图（第 13 项）、粘贴（第 14 项）、活动迭代卡片（第 15 项）。
- **C 组**：在基线上跑同样的脚本（plan 第 16 项）。

### 7.2 四轮的结果

| 轮 | 被测提交 | 本分支 | 基线 `c493255` | 这一轮发现的 |
|---|---|---|---|---|
| 1 | `4cf483f`（T5 之后） | A1 403、A2 177 全部通过；A3 179 / 9；B4 91 / 0、B5 9 / 0、B10 14 / 0、B11 24 / 0 | A3 同样的 9 项失败；B4 55 / 36、B5 8 / 1 | A3 的 9 项是个人主页窄屏显示 `profile.tabs.assigned` 这样的键，基线就有（FW6） |
| 2 | `d25f08c`（T8 之后） | A 同上；B 183 / 0 | B 141 / 42（B4、B5、B6 的 `&amp;`、B8 的数字和 `&amp;`） | B7 记录到：只有图片或只有提及的草稿关闭时不问就丢掉，两个构建都是（FW5） |
| 3 | `9a98148`（修复轮之后） | A1 403、A2 177、A3 188、个人主页 18，全部通过；B 316 / 0（含第 7 项草稿 27、第 9 项主题名 18、第 12 项封面 32、第 13 项图片 81） | A3 179 / 9、个人主页 14 / 4；B 257 / 59 | 个人设置的主题下拉框画在左上角、zh-CN 界面有英文，基线相同（交 M2、M8，spec 2.10） |
| 4 | `ff555bf`（T20 之后） | **1147 项全部通过** | **1067 / 80** | 没有新的问题 |

第 4 轮逐项（本分支 / 基线）：A1 403 / 403；A2 177 / 177；A3 188 / 179（基线失败 9，个人主页的键）；个人主页 18 / 14（失败 4）；B4 91 / 55（36）；B5 9 / 8（1）；B6 6 / 4（2）；B7 27 / 25（2）；B8 14 / 11（3）；B9 18 / 16（2）；B10 14 / 14；B11 24 / 24；第 12 项 32 / 30（2）；第 13 项 81 / 70（11）；第 14 项 22 / 21（1）；第 15 项 23 / 16（7）。基线上失败的每一项都是收尾改掉的行为：`window.open` 没有 `noopener`、菜单项调用 `window.close()`、预览显示 `&amp;`、草稿不问就丢、标注块的属性是数字和 `&amp;`、Power K 的主题名是英文（en 的 "System preference" 小写的 p 也不对）、封面是 JPEG、11 张图与现在不同、`onerror` 执行了、卡片显示 "5/8 Work items closed" 和 62.5、分组行的片段没有 key。基线上也通过的项（A1、A2、B10、B11 和各项里核对"没有弄坏"的部分）说明收尾没有弄坏它们。

控制者看了第 4 轮的截图，包括 T19 的卡片（本分支 "3/8 work items completed"、进度 38%，zh-CN 的标题和说明是中文；基线 "5/8 Work items closed"、62.5%）。zh-CN 下各状态组的 "Completed"、"3 Work items" 仍是英文，那是没有经过 `t()` 的字面量，交 M8。

整分支评审之后的修复轮没有再跑浏览器核对：FW14 只改 5 个中文值（构建里的 zh-CN 语言 chunk），FW15 只收窄一个包的导出（构建不变），FW16 只改注释，FW17 只改根目录的脚本。它们由门禁、测试和范围复核覆盖。

### 7.3 脚本全文

脚本在 `$COTMP/probe/` 下。P4、P5 的脚本（第 1、2 项）的全文在 [P4 评审](P4-router-native-review.md)和 [P5 评审](P5-brand-review.md)的附录里；收尾重跑的是它们的副本，只改了一处：本地服务器监听 `PROBE_PORT` 给的端口（原来是 `0`，由系统分配），好让一轮核对的十几个服务器各用固定的端口、互不冲突。改动的 8 个文件每个都只有这一行（`diff` 核对）。

#### 7.3.1 `run4.sh`：第 4 轮的总入口

对一个构建依次跑全部核对，写出每项的日志和摘要。

```bash
#!/bin/bash
# One-off (M1/closeout browser checks, probe run 4): every probe of the closeout's browser checks on the build in
# <repo>, labelled <which> (branch or base). Outputs: $COTMP/probe/run4-<which>-<probe>.log, screenshots in
# $COTMP/probe/shots/run4-<which>-<probe>/. Ports: <port base> + 1 … + 10 for group A (run-a.sh), then + 11 … + 17.
#   a            A1 (P4's nine segments), A2 (P5's brand.mjs), A3 (a3hook scans; a3sum.mjs): run-a.sh, whose summary
#                heads the log, followed by each segment's full output and a3sum's;
#   a3-profile   the profile tabs at 480 and 1440 px (a3-profile.mjs);
#   b            items 4–11 (closeout-b.mjs, through run-b.sh);
#   t14-covers   item 12, the architect's covers-probe.mjs; t15-pictures: item 13, pictures-probe.mjs;
#   t17-copy     item 14, the implementer's copy-probe.mjs: the three run from byte-identical copies under
#                $COTMP/probe/arch/, whose relative imports reach the port-adapted library copies under $COTMP/probe/;
#   t19-active-cycle  the T19 check (active-cycle.mjs).
# usage: bash run4.sh <which> <repo> <port base>
set -u
C=/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-co/probe
WHICH=$1
REPO=$2
PORT=$3
L=$C/run4-$WHICH
S=$C/shots/run4-$WHICH
head=$(cat "$REPO/.git/HEAD")
case $head in "ref: "*) head="$head = $(cat "$REPO/.git/${head#ref: }" 2>/dev/null || grep " ${head#ref: }\$" "$REPO/.git/packed-refs")" ;; esac
echo "run 4, $WHICH: build at $head ($REPO)"

# group A
bash "$C/run-a.sh" "run4-$WHICH" "$REPO" "$PORT" > "$L-a.log" 2>&1
for name in current addresses redirects palette p2probe1 p2probe2 p2probe3 p3auth p3lists brand a3; do
  printf '\n##### %s\n' "$name" >> "$L-a.log"
  cat "$C/logs/run4-$WHICH/$name.txt" >> "$L-a.log"
done

# the rest: one script each, from the repository root of the build
cd "$REPO"
one() {
  local name=$1 port=$2
  shift 2
  env PROBE_PORT="$port" OUT="$S-$name" node "$@" > "$L-$name.log" 2>&1
  echo "$name: exit $?, PASS $(grep -c '^PASS' "$L-$name.log"), FAIL $(grep -c '^FAIL' "$L-$name.log"), last: $(tail -1 "$L-$name.log")"
}
one a3-profile $((PORT + 11)) "$C/a3-profile.mjs"
bash "$C/run-b.sh" "run4-$WHICH" "$REPO" $((PORT + 12)) > "$L-b.log.head" 2>&1
cat "$L-b.log.head" "$C/logs/run4-$WHICH/closeout-b.txt" > "$L-b.log"
rm -f "$L-b.log.head"
echo "b: PASS $(grep -c '^PASS' "$C/logs/run4-$WHICH/closeout-b.txt"), FAIL $(grep -c '^FAIL' "$C/logs/run4-$WHICH/closeout-b.txt"), last: $(tail -1 "$L-b.log")"
one t14-covers $((PORT + 13)) "$C/arch/t14/probe/covers-probe.mjs"
one t15-pictures $((PORT + 14)) "$C/arch/t15/probe/pictures-probe.mjs"
one t17-copy $((PORT + 15)) "$C/arch/t17/probe/copy-probe.mjs"
one t19-active-cycle $((PORT + 16)) "$C/active-cycle.mjs"
```

#### 7.3.2 `run-a.sh`：A 组（1、2、3）

起 P4 的 9 段、P5 的 `brand.mjs` 和 A3 的扫描，每段一个端口。

```bash
#!/bin/bash
# One-off (M1/closeout browser checks, plan "控制者的浏览器核对" group A): on the build in <repo> (a clone with
# web/apps/web/build/client built), run
#   A1  P4's nine segments, as $P4TMP/probe-c/run-c.sh runs them (same scripts, same environment variables);
#   A2  P5's brand.mjs, groups A, B and C;
#   A3  every probe of A1 and A2 under a3hook.mjs, which scans each page they visit for raw i18n keys and broken
#       pictures (a3sum.mjs turns the scans into checks).
# The scripts are copies under $COTMP/probe/nerve-p{2,4,5}/ (the originals stay untouched); the copies differ only in
# the port their static server listens on (PROBE_PORT, <port base> + n, instead of a free port the system picks).
# Outputs go to $COTMP/probe/logs/<tag>/. usage: bash run-a.sh <tag> <repo> <port base>
set -u
C=/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-co/probe
TAG=$1
PORT=$3
OUT=$C/logs/$TAG
mkdir -p "$OUT"
rm -f "$OUT"/a3-*.jsonl
cd "$2"
head=$(cat .git/HEAD)
case $head in "ref: "*) head="$head = $(cat ".git/${head#ref: }" 2>/dev/null || grep " ${head#ref: }\$" .git/packed-refs)" ;; esac
echo "build at $head ($2)"
n=0
run() {
  local name=$1
  shift
  n=$((n + 1))
  env PROBE_PORT=$((PORT + n)) A3_OUT="$OUT/a3-$name.jsonl" A3_NAME="$name" "$@" > "$OUT/$name.txt" 2>&1
  local code=$?
  local fails
  fails=$(grep -c "^FAIL" "$OUT/$name.txt")
  echo "$name: exit $code, PASS $(grep -c '^PASS' "$OUT/$name.txt"), FAIL $fails, last: $(tail -1 "$OUT/$name.txt")"
  [ "$fails" -gt 0 ] && grep "^FAIL" "$OUT/$name.txt" | head -12 | cut -c1-260
  return 0
}
HOOK=(--import "$C/a3hook.mjs")
# A1: run-c.sh's nine segments
run current node "${HOOK[@]}" "$C/nerve-p4/probe-c/current.mjs"
run addresses node "${HOOK[@]}" "$C/nerve-p4/probe-c/addresses.mjs"
run redirects SLASH= STRICT=1 node "${HOOK[@]}" "$C/nerve-p4/probe-a/redirects.mjs"
run palette SLASH= STRICT=1 node "${HOOK[@]}" "$C/nerve-p4/probe-b/palette.mjs"
run p2probe1 STRICT=1 node "${HOOK[@]}" "$C/nerve-p4/probe-b/p2probe1.mjs"
run p2probe2 node "${HOOK[@]}" "$C/nerve-p2/probe-2/editor-probe.mjs"
run p2probe3 node "${HOOK[@]}" "$C/nerve-p4/probe-c/p2probe3.mjs"
run p3auth node "${HOOK[@]}" "$C/nerve-p4/probe-c/p3auth.mjs"
run p3lists node "${HOOK[@]}" "$C/nerve-p4/probe-c/p3lists.mjs"
# A2: P5's brand.mjs, all three groups
run brand OUT="$C/shots/brand-$TAG" node "${HOOK[@]}" "$C/nerve-p5/probe/brand.mjs"
# A3
node "$C/a3sum.mjs" "$OUT"/a3-*.jsonl > "$OUT/a3.txt" 2>&1
echo "a3: exit $?, PASS $(grep -c '^PASS' "$OUT/a3.txt"), FAIL $(grep -c '^FAIL' "$OUT/a3.txt"), last: $(tail -1 "$OUT/a3.txt")"
grep "^FAIL" "$OUT/a3.txt" | head -30 | cut -c1-300
```

#### 7.3.3 `a3hook.mjs`：A3 的扫描

由 `run-a.sh` 注入 P4、P5 的脚本：在它们走过的每个页面上找原样显示的键和加载失败的图片。

```js
// One-off (M1/closeout browser checks, plan "控制者的浏览器核对" item 3): loaded with `node --import` in front of every
// group A probe (P4's nine segments, P5's brand.mjs), so item 3 looks at exactly the pages items 1 and 2 visit, with
// their stubs, without editing those scripts. It wraps chromium.launch: every browser context the probe opens gets an
// init script that, in the top frame, scans each settled state of the page (400 ms after the DOM last changed, at
// least every 2 s while it keeps changing, and once more when the probe closes the context or the browser) for
//   key    a text node (or an input's placeholder) whose trimmed value matches ^[a-z0-9_-]+(\.[a-z0-9_-]+)+$: an
//          i18n key shown as is;
//   img    an <img> that finished loading with naturalWidth 0: a broken picture (an <img> still loading at the
//          final scan is waited for, up to 5 s, and reported as "pending" if it never finishes);
// and reports each scan (the path, the counts of text nodes and pictures looked at, what it found) through a binding
// to $A3_OUT, one JSON line per scan. It prints nothing, so the probe's own PASS/FAIL output is unchanged;
// a3sum.mjs turns the lines into checks.
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const OUT = process.env.A3_OUT;
const NAME = process.env.A3_NAME ?? "probe";
if (!OUT) throw new Error("a3hook: set A3_OUT");

function monitor() {
  if (window.top !== window) return;
  const KEY = /^[a-z0-9_-]+(\.[a-z0-9_-]+)+$/;
  const SKIP = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "TEMPLATE"]);
  const visible = (el) => !!el && el.getClientRects().length > 0 && getComputedStyle(el).visibility !== "hidden";
  const scan = async (final) => {
    if (!document.documentElement) return;
    const keys = [];
    let texts = 0;
    const walker = document.createTreeWalker(document.documentElement, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      const node = walker.currentNode;
      if (SKIP.has(node.parentElement?.tagName)) continue;
      const value = node.nodeValue.trim();
      if (!value) continue;
      texts++;
      if (KEY.test(value)) keys.push({ text: value, visible: visible(node.parentElement) });
    }
    for (const el of document.querySelectorAll("input[placeholder], textarea[placeholder]")) {
      const value = el.getAttribute("placeholder").trim();
      if (KEY.test(value)) keys.push({ text: value, placeholder: true, visible: visible(el) });
    }
    let imgs = [...document.querySelectorAll("img")];
    if (final) {
      const until = Date.now() + 5000;
      while (imgs.some((i) => !i.complete) && Date.now() < until) await new Promise((r) => setTimeout(r, 100));
      imgs = [...document.querySelectorAll("img")];
    }
    const broken = [];
    const pending = [];
    for (const img of imgs) {
      const src = (img.currentSrc || img.getAttribute("src") || "").slice(0, 160);
      if (!img.complete) pending.push({ src, loading: img.loading });
      else if (img.naturalWidth === 0) broken.push({ src, alt: img.getAttribute("alt"), visible: visible(img) });
    }
    const loaded = imgs.filter((i) => i.complete && i.naturalWidth > 0).length;
    try {
      await window.__a3report({ path: location.pathname, final, texts, imgs: imgs.length, loaded, keys, broken, pending: final ? pending : [] });
    } catch {
      // the page is going away
    }
  };
  let timer = null;
  let first = 0;
  const schedule = () => {
    const now = Date.now();
    if (!first) first = now;
    clearTimeout(timer);
    // a settled state: 400 ms without a change, or every 2 s while the page keeps changing
    timer = setTimeout(() => {
      first = 0;
      scan(false);
    }, now - first > 2000 ? 0 : 400);
  };
  new MutationObserver(schedule).observe(document, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ["src"] });
  window.__a3scan = () => scan(true);
}

let contexts = 0;
const write = (line) => fs.appendFileSync(OUT, `${JSON.stringify(line)}\n`);
const finalScan = async (context) => {
  for (const page of context.pages()) {
    try {
      await Promise.race([
        page.evaluate(() => window.__a3scan?.()),
        new Promise((r) => setTimeout(r, 8000)),
      ]);
    } catch {
      // closed or navigating
    }
  }
};
const originalLaunch = chromium.launch.bind(chromium);
chromium.launch = async (...args) => {
  const browser = await originalLaunch(...args);
  const originalNewContext = browser.newContext.bind(browser);
  const open = new Set();
  browser.newContext = async (...cargs) => {
    const context = await originalNewContext(...cargs);
    const id = ++contexts;
    open.add(context);
    await context.exposeBinding("__a3report", (_source, scanResult) => write({ script: NAME, context: id, ...scanResult }));
    await context.addInitScript(monitor);
    const originalClose = context.close.bind(context);
    context.close = async (...closeArgs) => {
      if (open.delete(context)) await finalScan(context);
      return originalClose(...closeArgs);
    };
    return context;
  };
  const originalBrowserClose = browser.close.bind(browser);
  browser.close = async (...closeArgs) => {
    for (const context of [...open]) {
      open.delete(context);
      await finalScan(context);
    }
    return originalBrowserClose(...closeArgs);
  };
  return browser;
};
```

#### 7.3.4 `a3sum.mjs`：A3 的汇总

把 A3 每段的记录合并成通过、失败的计数。

```js
// One-off (M1/closeout browser checks, plan item 3): turns the scans a3hook.mjs wrote (one JSON line per scan) into
// checks. A page is a (probe, path) pair; every page items 1 and 2 visited gets two checks over all its scans:
//   no raw key   no text node or placeholder matched ^[a-z0-9_-]+(\.[a-z0-9_-]+)+$ in any scan of it;
//   pictures     every <img> that finished loading had naturalWidth > 0, and none was still loading at the final scan.
// usage: node a3sum.mjs <scans.jsonl>...
import fs from "node:fs";

const pages = new Map();
for (const file of process.argv.slice(2)) {
  for (const line of fs.readFileSync(file, "utf8").split("\n")) {
    if (!line) continue;
    const s = JSON.parse(line);
    const id = `${s.script} ${s.path}`;
    let p = pages.get(id);
    if (!p) pages.set(id, (p = { scans: 0, texts: 0, imgs: 0, loaded: 0, keys: new Map(), broken: new Map(), pending: new Map() }));
    p.scans++;
    p.texts = Math.max(p.texts, s.texts);
    p.imgs = Math.max(p.imgs, s.imgs);
    p.loaded = Math.max(p.loaded, s.loaded);
    for (const k of s.keys) p.keys.set(`${k.placeholder ? "placeholder " : ""}${k.text}${k.visible ? "" : " (hidden)"}`, true);
    for (const b of s.broken) p.broken.set(`${b.src} alt=${b.alt}${b.visible ? "" : " (hidden)"}`, true);
    for (const b of s.pending) p.pending.set(`${b.src} loading=${b.loading}`, true);
  }
}
let passed = 0;
let failed = 0;
const check = (name, ok, detail) => {
  if (ok) passed++;
  else failed++;
  console.log(ok ? `PASS ${name}` : `FAIL ${name}: ${detail}`);
};
for (const [id, p] of [...pages].sort(([a], [b]) => a.localeCompare(b))) {
  console.log(`\n== ${id} (${p.scans} scans, up to ${p.texts} text nodes, ${p.imgs} pictures of which ${p.loaded} loaded)`);
  check(`${id}: no raw i18n key`, p.keys.size === 0, JSON.stringify([...p.keys.keys()]));
  check(`${id}: every picture loaded`, p.broken.size === 0 && p.pending.size === 0, JSON.stringify({ broken: [...p.broken.keys()], pending: [...p.pending.keys()] }));
}
console.log(`\n${pages.size} pages; ${passed} passed, ${failed} failed`);
process.exitCode = failed ? 1 : 0;
```

#### 7.3.5 `a3-selftest.mjs`：A3 的自检

先证明扫描能发现原样的键和坏图（只在第 1 轮跑过）。

```js
// One-off (M1/closeout browser checks, plan item 3): the control of a3hook.mjs / a3sum.mjs. Run under the hook, it
// opens one page and adds to it a text node that is an i18n key ("workspace_projects.no_such_key"), an <img> whose
// address serves no picture (/no-such-picture.png: the static server answers the app's index.html) and a good one
// (the app's icon, /icons/icon-192x192.png). a3sum.mjs must then fail the page's two checks, naming exactly the
// added key and the bad picture. usage: (from the repository root of the build)
//   PROBE_PORT=<port> A3_OUT=<file> A3_NAME=selftest node --import a3hook.mjs a3-selftest.mjs && node a3sum.mjs <file>
import { check, goto, probe, scenario, visible } from "./nerve-p4/probe-a/lib.mjs";
import { stubs } from "./nerve-p4/probe-c/stubs.mjs";

probe(async () => {
  await scenario("A3 control", stubs(), async (page) => {
    await goto(page, "/probe-ws/projects");
    check("A3 control: the projects page is up", await visible(page.getByText("Probe Project").first()));
    await page.evaluate(() => {
      const span = document.createElement("span");
      span.textContent = "workspace_projects.no_such_key";
      document.body.append(span);
      for (const src of ["/no-such-picture.png", "/icons/icon-192x192.png"]) {
        const img = document.createElement("img");
        img.src = src;
        document.body.append(img);
      }
    });
    await page.waitForTimeout(1500);
  });
});
```

#### 7.3.6 `a3-profile.mjs`：个人主页的标签（FW6）

窗口 480 px 和 1440 px 下个人主页顶部栏显示标签的名字，不显示文案键。

```js
// One-off (M1/closeout browser checks, plan item 3, follow-up): A3 found the text node "profile.tabs.assigned" (and
// "…created") on the profile pages, hidden at the probes' 1440 px. The profile header's small-screen menu button
// (md:hidden) prints its `type` prop, and the layout passes it the tab's i18n key (PROFILE_TABS[].i18n_label), not
// the translation. This shows the page at 480 px, where that button is the one on screen, and at 1440 px; at 480 px
// it also checks that the button says the tab's English name.
// usage: PROBE_PORT=<port> OUT=<screenshot dir> node a3-profile.mjs   (from the repository root of the build)
import fs from "node:fs";
import path from "node:path";
import { check, goto, probe, scenario, visible } from "./nerve-p4/probe-a/lib.mjs";
import { stubs } from "./nerve-p4/probe-c/stubs.mjs";

const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });

probe(async () => {
  for (const [tab, key, label] of [
    ["assigned", "profile.tabs.assigned", "Assigned"],
    ["created", "profile.tabs.created", "Created"],
  ])
    for (const width of [480, 1440])
      await scenario(`profile ${tab} at ${width} px`, stubs(), async (page) => {
        await page.setViewportSize({ width, height: 900 });
        await goto(page, `/probe-ws/profile/u1/${tab}`);
        await visible(page.getByText("Probe User").first());
        await page.waitForTimeout(1500);
        const raw = page.getByText(key, { exact: true });
        const shown = await raw.isVisible();
        console.log(`  (the raw key "${key}" is ${shown ? "on screen" : "not on screen"}; in the DOM: ${await raw.count()})`);
        await page.screenshot({ path: path.join(out, `profile-${tab}-${width}.png`) });
        check(`profile ${tab} at ${width} px: the page does not show the raw key "${key}" (it should say "${label}")`, !shown, `"${key}" is on screen`);
        if (width < 768) {
          // (run 3) the small-screen menu button names the tab (its text may be cut off on screen, not in the DOM)
          const button = page.locator("button:visible").filter({ hasText: new RegExp(`^\\s*${label}\\s*$`) });
          check(`profile ${tab} at ${width} px: the header's menu button says "${label}"`, (await button.count()) > 0, `no visible button says "${label}"`);
        }
      });
});
```

#### 7.3.7 `closeout-b.mjs`：B 组第 4–11 项

`window.open`、`window.close`、通知预览、评论和草稿的是否为空、标注块、主题名、工作项表单的迭代、空状态的插图。

```js
// One-off (M1/closeout browser checks, plan "控制者的浏览器核对" group B items 4–11; group C item 12 runs it on the
// base build): what the closeout changed (T2, T4, T5; T7, T8 from the second run; T11 and the fix round from the third).
//   4   window.open (T2): the twelve call sites, each through the UI: the editor's link click, the attachment list,
//       the image toolbar's download, the full-screen preview's download and "open image in new tab", and "Open in new
//       tab" on a cycle, a module, a project view, a work item, a project card, a default and a saved workspace view.
//       Each: one call, its third argument names noopener and noreferrer, the new tab (popup event) has a null
//       window.opener and an empty document.referrer;
//   5   window.close (T5): tabbed project navigation, the window narrowed until the tabs overflow, an item chosen in
//       the overflow menu: the tab opens, the menu closes, window.close is not called;
//   6   notification preview (T8): a comment and a description change of "<p>Tom &amp; Jerry &lt;3</p>" read
//       "Tom & Jerry <3" in the notifications list;
//   7   "is it empty" (T8): the comment editor's Comment button and Enter for an empty comment, spaces, non-breaking
//       spaces, line breaks, hard breaks, text, only an image, only a mention (empty ones send nothing, the others
//       POST the editor's HTML); the draft modal's Discard for the same inputs (text, only an image and only a
//       mention ask "Save this draft?", the empty inputs close without asking; the fix round's 773b3d8);
//   8   callout (T7, T8): a saved callout with data-emoji-unicode="128161" shows 💡 and keeps the string; a callout
//       inserted with /callout takes the emoji stored in localStorage, its URL with a plain & (read from the node and
//       from the editor's HTML output: the URL is never rendered into the page);
//   9   theme names (T11): with the profile's language zh-CN, and then en, Power K's theme menu and the profile
//       settings' theme switch name the five themes in that language;
//   10  the work item form's cycles (T5): opened from the workspace home and inside the project, the cycle dropdown
//       lists the project's cycle and GET …/cycles/ is sent (from the home, by the form, before the dropdown opens);
//   11  empty-state illustrations (T4): the project list, the cycle list and the intake page show theirs.
// The server, the stubs and the per-scenario checks (no request outside the scenario's list, no uncaught page error,
// no navigate() warning) are P4's (nerve-p4/probe-a/lib.mjs, copied), the data P4's group C data (p3data.mjs,
// stubs.mjs). Screenshots go to $OUT. Run from the repository root of the build.
// usage: PROBE_PORT=<port> OUT=<screenshot dir> node closeout-b.mjs [scenario name filter]
import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { check, goto, probe, scenario, visible } from "./nerve-p4/probe-a/lib.mjs";
import { COMMENTS, ISSUES, P, W, cycle, emptyPage, filterProps, filtered, notification, paginated, project, users } from "./nerve-p4/probe-c/p3data.mjs";
import { stubs } from "./nerve-p4/probe-c/stubs.mjs";

const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });
const shot = (page, name) => page.screenshot({ path: path.join(out, `${name}.png`) });
// P3's profile has no is_tour_completed: the workspace home would open the product tour over everything
const TOUR_DONE = {
  "GET /api/users/me/profile/": { id: "p1", user: "u1", language: "en", is_onboarded: true, is_tour_completed: true, theme: {} },
};

// ---------------------------------------------------------------- 4. window.open (T2)
// window.open is wrapped: every call's arguments are recorded, then the original runs (the tab really opens).
const recordOpen = () => {
  window.__opens = [];
  const original = window.open;
  window.open = function (...args) {
    window.__opens.push(args.map((a) => (a === undefined ? null : String(a))));
    return original.apply(this, args);
  };
};
// What a new tab loads is answered here, so the tab has a document to ask about its opener and referrer: a page of
// the app (an "open in new tab" address) gets a small landing page instead of the whole app (whose requests would
// belong to no scenario), another site (a member's link) the same, and a file under /api/assets/ its bytes (below).
const LANDING = "<!doctype html><title>landing</title><p>new tab</p>";
const landOn = (context, match) =>
  context.route(match, (route) => route.fulfill({ status: 200, contentType: "text/html", body: LANDING }));
// the app's own address `path`: only while the new tab opens (newTabs), so the scenario's page loads the real app
const landOnApp = async (context, path) => {
  const match = (url) => url.pathname === path;
  const handler = (route) =>
    route.request().resourceType() === "document"
      ? route.fulfill({ status: 200, contentType: "text/html", body: LANDING })
      : route.fallback();
  await context.route(match, handler);
  return () => context.unroute(match, handler);
};
const OPEN_IN_NEW_TAB = "Open in new tab";
// a second project the user has not joined: its card offers "Open in new tab"
const P2 = { ...project, id: "p2", name: "Other Project", identifier: "OTH", is_member: false, member_role: null, members: ["u2"] };
const WITH_P2 = {
  [`GET ${W}/projects/`]: [project, P2],
  [`GET ${W}/projects/details/`]: [project, P2],
};
const WORKSPACE_VIEW = {
  id: "wv1", name: "Probe Workspace View", description: "", access: 1, workspace: "w1", workspace_id: "w1",
  owned_by: "u1", created_by: "u1", logo_props: {}, is_locked: false, is_favorite: false, sort_order: 1,
  rich_filters: {}, display_filters: { layout: "spreadsheet" }, display_properties: {}, filters: {}, query: {},
  created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z",
};
const WORKSPACE_VIEWS = {
  [`GET ${W}/views/`]: [WORKSPACE_VIEW],
  [`GET ${W}/views/${WORKSPACE_VIEW.id}/`]: WORKSPACE_VIEW,
  // the views' work items (P3's list answer: filtered by the query, grouped as asked)
  [`GET ${W}/issues/`]: (request) => paginated(filtered(new URL(request.url()).searchParams), new URL(request.url()).searchParams),
};
// ui's ContextMenu: every row keeps one mounted (transparent while closed); the open one is the overlay that takes
// pointer events (the active cycle has two, one over the other: the last is on top)
const openContextMenu = (page) => page.locator('div.pointer-events-auto.opacity-100 > [data-context-menu="true"]');
const VIEW = {
  id: "v1", name: "Probe View", description: "", access: 1, project: "p1", project_id: "p1", workspace: "w1",
  workspace_id: "w1", owned_by: "u1", created_by: "u1", logo_props: {}, is_locked: false, is_favorite: false,
  sort_order: 1, rich_filters: {}, display_filters: { layout: "list" }, display_properties: {},
  created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z",
};
// a 64x48 PNG, for the image in the description (the editor loads it from /api/assets/)
const png = (() => {
  const [w, h] = [64, 48];
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type), data]);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(zlib.crc32(body));
    return Buffer.concat([len, body, crc]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr.set([8, 2, 0, 0, 0], 8);
  const row = Buffer.concat([Buffer.from([0]), Buffer.alloc(w * 3, Buffer.from([0x3a, 0x7b, 0xd5]))]);
  const raw = Buffer.concat(Array.from({ length: h }, () => row));
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ]);
})();
const ASSET = "5f1c2a4e-3333-4a2b-9c3d-000000000003";
const ASSETS = `/api/assets/v2/workspaces/probe-ws/projects/p1`;
const IMAGE_URL = `${ASSETS}/${ASSET}/`;
const IMAGE_DOWNLOAD_URL = `${ASSETS}/download/${ASSET}/`;
const ATTACHMENT_URL = `${ASSETS}/issues/i1/attachments/att1/`;
// the files under /api/assets/ this scenario serves; any other /api/assets/ request falls through to lib.mjs's
// handler, which answers 404 and fails the scenario's "no request outside the list"
const serveAssets = (context, served) =>
  context.route(
    (url) => url.pathname.startsWith("/api/assets/"),
    (route) => {
      const { pathname } = new URL(route.request().url());
      const file = {
        [IMAGE_URL]: { contentType: "image/png", body: png },
        [IMAGE_DOWNLOAD_URL]: { contentType: "image/png", body: png },
        [ATTACHMENT_URL]: { contentType: "text/plain", body: "the attachment's bytes\n" },
      }[pathname];
      if (!file) return route.fallback();
      served.push(`${route.request().method()} ${pathname}`);
      return route.fulfill({ status: 200, ...file });
    }
  );

// Runs `action`, then reports the window.open calls it made and every tab it opened (Playwright's popup event):
// the tab's address, whether its window.opener is null and its document.referrer. `landing`: an address of the app
// the new tab gets the landing page for.
async function newTabs(page, action, landing) {
  const popups = [];
  const onPopup = (p) => popups.push(p);
  page.on("popup", onPopup);
  const stopLanding = landing ? await landOnApp(page.context(), landing) : null;
  const before = await page.evaluate(() => window.__opens.length);
  await action();
  await page.waitForFunction((n) => window.__opens.length > n, before, { timeout: 5000 }).catch(() => {});
  await page.waitForTimeout(1500);
  page.off("popup", onPopup);
  await stopLanding?.();
  const calls = await page.evaluate((n) => window.__opens.slice(n), before);
  const tabs = [];
  for (const p of popups) {
    await p.waitForLoadState("domcontentloaded").catch(() => {});
    tabs.push(
      await p
        .evaluate(() => ({ url: location.href, openerIsNull: window.opener === null, referrer: document.referrer }))
        .catch((e) => ({ url: p.url(), error: e.message.split("\n")[0] }))
    );
    await p.close();
  }
  return { calls, tabs };
}
// The checks of one call site: one window.open call to `url`, its third argument names noopener and noreferrer; the
// new tab opened at `url`, and in it window.opener is null and document.referrer is empty.
function checkOpen(page, site, { calls, tabs }, url) {
  const detail = JSON.stringify({ calls, tabs });
  const call = calls[0] ?? [];
  // addresses compared absolute, a relative one taken against the opening page
  const target = (u) => {
    try {
      return new URL(u, page.url()).href;
    } catch {
      return String(u);
    }
  };
  check(`B4 ${site}: one window.open call, to ${url}`, calls.length === 1 && target(call[0]) === target(url), detail);
  const features = (call[2] ?? "").split(",").map((f) => f.trim());
  check(`B4 ${site}: the third argument has noopener and noreferrer`, features.includes("noopener") && features.includes("noreferrer"), detail);
  const tab = tabs.find((t) => t.url && target(t.url) === target(url));
  check(`B4 ${site}: the new tab opens (popup event)`, !!tab, detail);
  check(`B4 ${site}: in the new tab window.opener is null`, !!tab && tabs.every((t) => t.openerIsNull === true), detail);
  check(`B4 ${site}: in the new tab document.referrer is empty`, !!tab && tabs.every((t) => t.referrer === ""), detail);
}

// PRB-1 with a member's link and an image in its description, and one attachment
const LINK = "https://example.com";
const DESCRIPTION =
  `<p class="editor-paragraph-block" data-id="5f1c2a4e-1111-4a2b-9c3d-000000000001">Read <a href="${LINK}" target="_blank" rel="noopener noreferrer nofollow">${LINK}</a> first</p>` +
  `<image-component src="${ASSET}" id="5f1c2a4e-1111-4a2b-9c3d-000000000002" width="256px" height="192px" aspectRatio="1.3333333333333333" alignment="left" status="uploaded"></image-component>` +
  `<p class="editor-paragraph-block" data-id="5f1c2a4e-1111-4a2b-9c3d-000000000004">After the image</p>`;
const ATTACHMENT = {
  id: "att1", asset_url: ATTACHMENT_URL, attributes: { name: "report.txt", size: 1234, type: "text/plain" },
  created_by: "u2", updated_at: "2026-09-20T00:00:00Z", created_at: "2026-09-20T00:00:00Z", issue_id: "i1", project_id: "p1",
};
const withFiles = { ...ISSUES[0], description_html: DESCRIPTION, attachment_count: 1, issue_attachments: [ATTACHMENT] };
const WORK_ITEM_PAGE = { [`GET ${W}/work-items/PRB-1/`]: withFiles, [`GET ${P}/issues/i1/`]: withFiles };

// ---------------------------------------------------------------- 5. window.close (T5)
const TABBED = { ...filterProps("list"), navigation_project_limit: 10, navigation_control_preference: "TABBED" };

// ---------------------------------------------------------------- 6. notification preview (T8)
// a comment and a description change whose HTML holds the entities &amp; and &lt; (the list renders both through
// stripAndTruncateHTML: the comment by sanitizeCommentForNotification, the description directly)
const TOM_HTML = "<p>Tom &amp; Jerry &lt;3</p>";
const TOM_TEXT = "Tom & Jerry <3";
const TOM_NOTIFICATIONS = [
  notification("n6c", ISSUES[1], { field: "comment", verb: "created", old_value: "", new_value: TOM_HTML }, 3),
  notification("n6d", ISSUES[0], { field: "description", verb: "updated", old_value: "", new_value: TOM_HTML }, 5),
];
const NOTIFICATIONS_PAGE = {
  [`GET ${W}/users/notifications/`]: {
    ...emptyPage, next_cursor: "30:1:0", prev_cursor: "30:-1:1", count: 2, total_count: 2, total_pages: 1, results: TOM_NOTIFICATIONS,
  },
};
// every text node of the page that mentions Jerry
const jerryTexts = (page) =>
  page.evaluate(() => {
    const found = [];
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) if (walker.currentNode.nodeValue.includes("Jerry")) found.push(walker.currentNode.nodeValue.trim());
    return found;
  });

// ---------------------------------------------------------------- 7. "is it empty" (T8)
// The tiptap editor keeps its instance on its ProseMirror element (dom.editor, @tiptap/core 2.26): the probe reads
// the editor's HTML through it, clears it between cases, and inserts the image (there is no upload to stub). Text,
// spaces, non-breaking spaces, line breaks and the mention go in through the keyboard and the mention list.
const COMMENT_EDITOR = "#editor-container-add_comment_i1 .ProseMirror";
const MODAL_EDITOR = "#editor-container-issue-modal-editor .ProseMirror";
const editorHTML = (page, selector) => page.locator(selector).evaluate((el) => el.editor.getHTML());
const clearEditor = (page, selector) => page.locator(selector).evaluate((el) => el.editor.commands.clearContent(true));
const IMAGE_NODE = `<image-component src="${ASSET}" id="5f1c2a4e-7777-4a2b-9c3d-000000000007" width="256px" height="192px" aspectRatio="1.3333333333333333" alignment="left" status="uploaded"></image-component>`;
// [case, what the author does in the focused editor `selector`, whether it is content]
const EMPTINESS_CASES = [
  ["empty", async () => {}, false],
  ["only spaces", async (page) => page.keyboard.type("   "), false],
  ["only non-breaking spaces", async (page) => page.keyboard.insertText("   "), false],
  // Shift+Enter starts a new paragraph in these editors; a <br> (hard break) is inserted by its command
  ["only line breaks", async (page) => { for (let i = 0; i < 3; i++) await page.keyboard.press("Shift+Enter"); }, false],
  ["only hard breaks (<br>)", async (page, selector) =>
    page.locator(selector).evaluate((el) => { for (let i = 0; i < 3; i++) el.editor.commands.setHardBreak(); }), false],
  ["text", async (page) => page.keyboard.type("Looks good"), true],
  ["only an image", async (page, selector) => page.locator(selector).evaluate((el, html) => el.editor.commands.insertContent(html), IMAGE_NODE), true],
  ["only a mention", async (page) => {
    await page.keyboard.type("@ott");
    const option = page.locator('[id^="mention-item-"]', { hasText: "otto" }).first();
    await option.waitFor({ state: "visible", timeout: 10000 });
    await option.click();
  }, true],
];
let commentSeq = 0;
const COMMENT_PAGE = {
  // a created comment answers with what was sent
  [`POST ${P}/issues/i1/comments/`]: (request) => ({
    ...COMMENTS[0], id: `cm-new-${++commentSeq}`, comment_html: request.postDataJSON().comment_html, actor: "u1", actor_detail: users.u1,
    created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
  }),
};

// ---------------------------------------------------------------- 8. callout (T7, T8)
// PRB-1's description holds a callout the way the editor saves one: its emoji's code point is the string "128161"
const BULB_URL = "https://cdn.jsdelivr.net/npm/emoji-datasource-apple/img/apple/64/1f4a1.png";
const CALLOUT_DESCRIPTION =
  `<p class="editor-paragraph-block" data-id="5f1c2a4e-8888-4a2b-9c3d-000000000001">Before the callout</p>` +
  `<div data-block-type="callout-component" data-logo-in-use="emoji" data-emoji-unicode="128161" data-emoji-url="${BULB_URL}" id="5f1c2a4e-8888-4a2b-9c3d-000000000002">` +
  `<p class="editor-paragraph-block" data-id="5f1c2a4e-8888-4a2b-9c3d-000000000003">Heads up</p></div>` +
  `<p class="editor-paragraph-block" data-id="5f1c2a4e-8888-4a2b-9c3d-000000000004">After the callout</p>`;
const withCallout = { ...ISSUES[0], description_html: CALLOUT_DESCRIPTION };
const CALLOUT_PAGE = { [`GET ${W}/work-items/PRB-1/`]: withCallout, [`GET ${P}/issues/i1/`]: withCallout };
// the callout logo a member picked last (the editor keeps it in localStorage): an emoji whose address has a query
const GRIN_URL = "https://cdn.example.com/emoji/1f600.png?size=64&set=apple";
const STORED_LOGO = JSON.stringify({ in_use: "emoji", emoji: { value: "128512", url: GRIN_URL } });
// the attributes of every callout node in the editor, from its document and from its HTML output (parsed back)
const callouts = (page, selector) =>
  page.locator(selector).evaluate((el) => {
    const fromDoc = [];
    el.editor.state.doc.descendants((node) => {
      if (node.type.name === "calloutComponent") fromDoc.push(node.attrs);
    });
    const html = el.editor.getHTML();
    const parsed = new DOMParser().parseFromString(html, "text/html");
    const fromHtml = [...parsed.querySelectorAll('div[data-block-type="callout-component"]')].map((div) =>
      Object.fromEntries([...div.attributes].map((a) => [a.name, a.value]))
    );
    return { fromDoc, fromHtml, html };
  });

// ---------------------------------------------------------------- 9. theme names (T11)
// The UI language comes from the user's profile: the profile store calls @nerve/i18n's setLanguage(profile.language),
// which changes i18next's language, stores it under localStorage "userLanguage" and sets <html lang>. The probe
// stubs the profile's language (and theme "system", so the switch shows its current value) and waits for <html lang>.
const THEME_NAMES = {
  en: ["System Preference", "Light", "Dark", "Light high contrast", "Dark high contrast"],
  "zh-CN": ["系统偏好", "浅色", "深色", "浅色高对比度", "深色高对比度"],
};
// Power K's command that opens its theme page (power_k.preferences_actions.update_theme)
const CHANGE_THEME = { en: "Change interface theme", "zh-CN": "更改界面主题" };
const profileIn = (language) => ({
  "GET /api/users/me/profile/": { id: "p1", user: "u1", language, is_onboarded: true, is_tour_completed: true, theme: { theme: "system" } },
  "GET /api/users/me/workspaces/invitations/": [],
});

// ---------------------------------------------------------------- 10. the work item form's cycles (T5)
// the cycle dropdown's option list: the innermost element holding both "No cycle" and the cycle's name (the list
// page behind the modal shows the cycle's name in its rows too)
const cycleOptions = (page) =>
  page
    .locator("div")
    .filter({ has: page.getByText("No cycle", { exact: true }) })
    .filter({ has: page.getByText(cycle.name, { exact: true }) })
    .last();

// ---------------------------------------------------------------- 11. empty states (T4)
const NO_PROJECTS = {
  [`GET ${W}/projects/`]: [],
  [`GET ${W}/projects/details/`]: [],
  "GET /api/users/me/workspaces/probe-ws/project-roles/": {},
};
const NO_CYCLES = { [`GET ${P}/cycles/`]: [], [`GET ${W}/cycles/`]: [] };
const INTAKE_PROJECT = { ...project, inbox_view: true };
const INTAKE = {
  [`GET ${W}/projects/`]: [INTAKE_PROJECT],
  [`GET ${W}/projects/details/`]: [INTAKE_PROJECT],
  [`GET ${P}/`]: INTAKE_PROJECT,
  [`GET ${P}/inbox-issues/`]: { ...emptyPage, next_cursor: "10:1:0", prev_cursor: "10:-1:1", count: 0, total_count: 0, total_results: 0 },
};
// the <svg> elements of at least 40x40 px in the empty state titled `title`: propel's EmptyStateDetailed and
// EmptyStateCompact both put the illustration and the text in one container, the ancestor with max-w-[25rem]
const illustrationNextTo = (page, title) =>
  page.evaluate((title) => {
    const own = [...document.querySelectorAll("body *")].find((el) => el.childElementCount === 0 && el.textContent.trim() === title);
    const container = own?.closest('[class*="max-w-[25rem]"]');
    if (!container) return { title: !!own, container: false, svgs: [] };
    const svgs = [...container.querySelectorAll("svg")]
      .map((s) => {
        const r = s.getBoundingClientRect();
        return { width: Math.round(r.width), height: Math.round(r.height), shapes: s.querySelectorAll("path, rect, circle, ellipse, line, polygon, polyline").length };
      })
      .filter((s) => s.width >= 40 && s.height >= 40);
    return { title: true, container: true, svgs };
  }, title);

probe(async () => {
  await scenario("B4 work item page", stubs({ ...TOUR_DONE, ...WORK_ITEM_PAGE }), async (page) => {
    const context = page.context();
    await context.addInitScript(recordOpen);
    const served = [];
    await serveAssets(context, served);
    await landOn(context, "https://example.com/**");
    await goto(page, "/probe-ws/browse/PRB-1");
    const editor = page.locator(".ProseMirror").first();
    const link = editor.locator(`a[href^="${LINK}"]`);
    check("B4 the description shows the member's link", await visible(link));
    const image = editor.locator("img").first();
    check("B4 the description shows the image", await visible(image));
    await page.waitForTimeout(1000);
    await shot(page, "B4-1-work-item");
    // 1. the editor opens a link on a click (custom-link's clickHandler)
    checkOpen(page, "editor link", await newTabs(page, () => link.click()), `${LINK}/`);
    // 2. the attachment list opens the file
    const attachment = page.getByRole("button", { name: /report\.txt/ });
    check("B4 the attachment list shows report.txt", await visible(attachment));
    checkOpen(page, "attachment list", await newTabs(page, () => attachment.click()), ATTACHMENT_URL);
    // 3. the image toolbar's download (the toolbar shows while the pointer is over the image)
    await image.hover();
    const toolbarDownload = page.locator('.editor-container button[aria-label="Download image"]');
    check("B4 the image toolbar shows its download button", await visible(toolbarDownload));
    await shot(page, "B4-2-image-toolbar");
    checkOpen(page, "image toolbar download", await newTabs(page, () => toolbarDownload.click()), IMAGE_DOWNLOAD_URL);
    // 4, 5. the full-screen preview's download and "open image in new tab"
    await image.hover();
    await page.locator('button[aria-label="View image in full screen"]').click();
    const viewer = page.locator('[aria-label="Fullscreen image viewer"]');
    check("B4 the full-screen preview opens", await visible(viewer));
    await shot(page, "B4-3-full-screen");
    checkOpen(page, "full-screen download", await newTabs(page, () => viewer.locator('button[aria-label="Download image"]').click()), IMAGE_DOWNLOAD_URL);
    checkOpen(page, "full-screen open original", await newTabs(page, () => viewer.locator('button[aria-label="Open image in new tab"]').click()), IMAGE_URL);
    console.log(`  (files served: ${JSON.stringify([...new Set(served)])})`);
  });

  // 6–12. "Open in new tab" in the menus of a project card, a cycle, a module, a project view, the work item list,
  // a default workspace view and a saved workspace view
  await scenario("B4 open in new tab: project lists", stubs({ ...TOUR_DONE, [`GET ${P}/views/`]: [VIEW] }), async (page) => {
    const context = page.context();
    await context.addInitScript(recordOpen);
    for (const [site, address, row, url] of [
      ["cycle", "/probe-ws/projects/p1/cycles", cycle.name, "/probe-ws/projects/p1/cycles/c1"],
      ["module", "/probe-ws/projects/p1/modules", "Module One", "/probe-ws/projects/p1/modules/m1"],
      ["project view", "/probe-ws/projects/p1/views", VIEW.name, "/probe-ws/projects/p1/views/v1"],
      ["work item", "/probe-ws/projects/p1/issues", ISSUES[0].name, "/probe-ws/browse/PRB-1"],
    ]) {
      await goto(page, address);
      const target = page.getByText(row, { exact: true }).first();
      check(`B4 ${site}: ${address} lists "${row}"`, await visible(target));
      await page.waitForTimeout(500);
      await target.click({ button: "right" });
      const item = openContextMenu(page).getByRole("button", { name: OPEN_IN_NEW_TAB }).last();
      check(`B4 ${site}: its context menu has "${OPEN_IN_NEW_TAB}"`, await visible(item, 5000));
      await shot(page, `B4-4-${site.replace(/ /g, "-")}-menu`);
      checkOpen(page, site, await newTabs(page, () => item.click(), url), url);
    }
  });

  // the project card offers "Open in new tab" for a project the user has not joined
  await scenario("B4 open in new tab: project card", stubs({ ...TOUR_DONE, ...WITH_P2 }), async (page) => {
    const context = page.context();
    await context.addInitScript(recordOpen);
    await goto(page, "/probe-ws/projects");
    const card = page.locator("a", { hasText: P2.name }).first();
    check(`B4 project card: /probe-ws/projects shows the card of "${P2.name}" (not joined)`, await visible(card));
    await page.waitForTimeout(500);
    await card.click({ button: "right" });
    const item = openContextMenu(page).getByRole("button", { name: OPEN_IN_NEW_TAB }).last();
    check(`B4 project card: its context menu has "${OPEN_IN_NEW_TAB}"`, await visible(item, 5000));
    await shot(page, "B4-5-project-card-menu");
    checkOpen(page, "project card", await newTabs(page, () => item.click(), "/probe-ws/projects/p2/issues"), "/probe-ws/projects/p2/issues");
  });

  // the workspace views header: a default view's menu and a saved view's menu
  await scenario("B4 open in new tab: workspace views", stubs({ ...TOUR_DONE, ...WORKSPACE_VIEWS }), async (page) => {
    const context = page.context();
    await context.addInitScript(recordOpen);
    for (const [site, address, url] of [
      ["default workspace view", "/probe-ws/workspace-views/assigned", "/probe-ws/workspace-views/assigned"],
      ["saved workspace view", `/probe-ws/workspace-views/${WORKSPACE_VIEW.id}`, `/probe-ws/workspace-views/${WORKSPACE_VIEW.id}`],
    ]) {
      await goto(page, address);
      await page.waitForTimeout(1500);
      await shot(page, `B4-6-${site.replace(/ /g, "-")}`);
      // the view's menu: the right-most menu button of the page header (the ⋯ after "Add view"; the top bar above
      // it ends at y 40)
      let trigger = null;
      let right = -1;
      for (const b of await page.locator("button[aria-haspopup]").all()) {
        const box = await b.boundingBox();
        if (box && box.y > 40 && box.y < 85 && box.x > right) [trigger, right] = [b, box.x];
      }
      check(`B4 ${site}: ${address} has the view's menu in its header`, !!trigger);
      await trigger.click();
      const item = page.getByRole("menuitem", { name: OPEN_IN_NEW_TAB });
      check(`B4 ${site}: its menu has "${OPEN_IN_NEW_TAB}"`, await visible(item, 5000));
      await shot(page, `B4-6-${site.replace(/ /g, "-")}-menu`);
      checkOpen(page, site, await newTabs(page, () => item.click(), url), url);
    }
  });

  await scenario("B5 tab overflow menu", stubs({ [`GET ${W}/user-properties/`]: TABBED }), async (page) => {
    // window.close is a counter: the base's propel MenuItem called the global close() on every click
    await page.context().addInitScript(() => {
      window.__closeCalls = 0;
      window.close = () => {
        window.__closeCalls++;
      };
    });
    // the page loads at 1440 px, then the window narrows to 480 px: the sidebar collapses (below 768 px it does on a
    // resize) and the tab bar still has no room for every tab (at 700 px all four fit once the sidebar is gone)
    await goto(page, "/probe-ws/projects/p1/issues");
    const header = page.locator("div.h-header").first();
    const workItemsTab = header.locator('a[href="/probe-ws/projects/p1/issues"]', { hasText: "Work items" });
    check("B5 the project navigation is tabbed (the Work items tab is in the header)", await visible(workItemsTab));
    await page.waitForTimeout(1000);
    await page.setViewportSize({ width: 480, height: 900 });
    await page.waitForTimeout(2000);
    const tabX = (await workItemsTab.first().boundingBox())?.x ?? Infinity;
    // the overflow menu's trigger: the menu button to the right of the tabs (the one left of them is the project's)
    let trigger = null;
    for (const b of await header.locator('button[aria-haspopup="menu"]').all())
      if (((await b.boundingBox())?.x ?? -1) > tabX) trigger = b;
    check("B5 the tabs overflow: an overflow menu button follows the tabs", !!trigger);
    await shot(page, "B5-1-tabs");
    await trigger.click();
    // the popup this trigger opened (propel Menu's popup carries data-main-menu)
    const menu = page.locator('[data-main-menu="true"]:visible').filter({ has: page.locator('[role="menuitem"]') });
    const items = (await visible(menu.locator('[role="menuitem"]'))) ? (await menu.locator('[role="menuitem"]').allInnerTexts()).map((t) => t.trim()) : [];
    // which tabs overflow depends on the measured widths; the menu holds the last ones of Cycles, Modules, Views
    const TABS = ["Cycles", "Modules", "Views"];
    check("B5 the overflow menu opens and lists the tabs that do not fit",
      items.length > 0 && JSON.stringify(items) === JSON.stringify(TABS.slice(TABS.length - items.length)), JSON.stringify(items));
    await shot(page, "B5-2-menu-open");
    const popup = await menu.first().elementHandle();
    const choice = items[0];
    const want = `/probe-ws/projects/p1/${choice.toLowerCase()}`;
    await menu.locator('[role="menuitem"]', { hasText: choice }).click();
    const landed = await page.waitForURL((u) => u.pathname === want, { timeout: 10000 }).then(() => true, () => false);
    await page.waitForTimeout(1000);
    await shot(page, "B5-3-after-choice");
    check(`B5 choosing ${choice} goes to ${want}`, landed, page.url());
    const stillOpen = await popup.evaluate((el) => el.isConnected && el.getClientRects().length > 0).catch(() => false);
    check("B5 the overflow menu closes", !stillOpen, "the menu is still open");
    const closeCalls = await page.evaluate(() => window.__closeCalls);
    check("B5 window.close() was not called (count 0)", closeCalls === 0, `count ${closeCalls}`);
  });

  // ---------------------------------------------------------------- 6. notification preview (T8)
  await scenario("B6 notification preview", stubs({ ...TOUR_DONE, ...NOTIFICATIONS_PAGE }), async (page) => {
    await goto(page, "/probe-ws/notifications");
    check("B6 the notifications list shows the comment notification", await visible(page.getByText("commented", { exact: false }).first()));
    await page.waitForTimeout(1000);
    const texts = await jerryTexts(page);
    console.log(`  (text nodes that mention Jerry: ${JSON.stringify(texts)})`);
    check(`B6 the comment and the description notification preview "${TOM_HTML}" as ${JSON.stringify(TOM_TEXT)}`,
      texts.length === 2 && texts.every((t) => t === TOM_TEXT), JSON.stringify(texts));
    check("B6 no preview shows an entity (&amp; or &lt;) as text", texts.every((t) => !/&(amp|lt|gt);/.test(t)), JSON.stringify(texts));
    await shot(page, "B6-notifications");
  });

  // ---------------------------------------------------------------- 7. "is it empty" (T8)
  // The comment editor under PRB-1's activity: for each case, is the Comment button enabled, and does a click on it
  // or Enter in the editor send POST …/comments/ (with the editor's HTML)?
  await scenario("B7 comment editor", stubs({ ...TOUR_DONE, ...COMMENT_PAGE }), async (page, ctx) => {
    const served = [];
    await serveAssets(page.context(), served);
    await goto(page, "/probe-ws/browse/PRB-1");
    const editor = page.locator(COMMENT_EDITOR);
    check("B7 the comment editor is there", await visible(editor));
    const button = page.getByRole("button", { name: "Comment", exact: true });
    const posts = () => ctx.requests.filter((r) => r.key === `POST ${P}/issues/i1/comments/`);
    for (const [name, fill, content] of EMPTINESS_CASES) {
      await clearEditor(page, COMMENT_EDITOR);
      await editor.click();
      await fill(page, COMMENT_EDITOR);
      await page.waitForTimeout(700);
      const html = await editorHTML(page, COMMENT_EDITOR);
      const enabled = await button.isEnabled();
      const before = posts().length;
      await button.click({ force: true });
      await page.waitForTimeout(1000);
      if (!content) {
        await editor.press("Enter");
        await page.waitForTimeout(1000);
      }
      const sent = posts().slice(before);
      console.log(`  (${name}: the editor's HTML ${JSON.stringify(html)}; Comment button ${enabled ? "enabled" : "disabled"}; ${sent.length} POST)`);
      if (content) {
        check(`B7 comment "${name}": the Comment button is enabled`, enabled, html);
        check(`B7 comment "${name}": a click sends one POST …/comments/ with the editor's HTML`,
          sent.length === 1 && JSON.parse(sent[0].body).comment_html === html, JSON.stringify(sent.map((r) => r.body)));
      } else {
        check(`B7 comment "${name}": the Comment button is disabled`, !enabled, html);
        check(`B7 comment "${name}": neither a click nor Enter sends a request`, sent.length === 0, JSON.stringify(sent.map((r) => r.body)));
      }
      if (name === "only a mention") await shot(page, "B7-1-comment-mention");
    }
  });

  // The create work item modal, title left empty, only the description filled: does Discard ask "Save this draft?"
  // (the description counts as content) or close the modal (it counts as empty)? Runs 1–2 recorded the image-only
  // and mention-only answers without asserting them (the draft layout passed ["img"] while the editor writes
  // <image-component> and <mention-component>, older than T8); since the fix round (773b3d8) both must ask.
  await scenario("B7 draft modal", stubs(TOUR_DONE), async (page) => {
    const served = [];
    await serveAssets(page.context(), served);
    await goto(page, "/probe-ws/projects/p1/issues");
    const answers = {};
    for (const [name, fill, content] of EMPTINESS_CASES) {
      await page.getByRole("button", { name: "Add work item" }).click();
      const editor = page.locator(MODAL_EDITOR);
      if (!(await visible(editor))) throw new Error("the create work item modal did not open");
      await editor.click();
      await fill(page, MODAL_EDITOR);
      await page.waitForTimeout(700);
      const html = await editorHTML(page, MODAL_EDITOR);
      await page.locator("form").getByRole("button", { name: "Discard", exact: true }).click();
      const asks = await visible(page.getByText("Save this draft?", { exact: true }), 2000);
      answers[name] = asks ? "asks to save a draft" : "closes without asking";
      console.log(`  (draft "${name}" (${content ? "content" : "empty"}): Discard ${answers[name]}; the description's HTML ${JSON.stringify(html)})`);
      if (name === "only an image") await shot(page, "B7-2-draft-image");
      if (asks) await page.getByRole("dialog").getByRole("button", { name: "Discard", exact: true }).click();
      await page.waitForTimeout(700);
      if (await page.locator(MODAL_EDITOR).isVisible()) throw new Error(`the modal is still open after the ${name} case`);
    }
    for (const name of ["text", "only an image", "only a mention"])
      check(`B7 draft "${name}": Discard asks to save a draft`, answers[name] === "asks to save a draft", JSON.stringify(answers));
    check("B7 draft: empty, spaces, non-breaking spaces, line breaks and hard breaks close without asking",
      ["empty", "only spaces", "only non-breaking spaces", "only line breaks", "only hard breaks (<br>)"].every((n) => answers[n] === "closes without asking"),
      JSON.stringify(answers));
  });

  // ---------------------------------------------------------------- 8. callout (T7, T8)
  await scenario("B8 callout from the description", stubs({ ...TOUR_DONE, ...CALLOUT_PAGE }), async (page) => {
    await goto(page, "/probe-ws/browse/PRB-1");
    const block = page.locator(".editor-callout-component").first();
    check("B8 the description shows the callout", await visible(block));
    await page.waitForTimeout(1000);
    const icon = (await block.locator("button").first().innerText()).trim();
    check(`B8 the callout's icon is 💡 (data-emoji-unicode="128161")`, icon === "💡", JSON.stringify(icon));
    const { fromDoc } = await callouts(page, ".ProseMirror:has(.editor-callout-component)");
    // T7: the node keeps the string the HTML holds (TipTap's default parsing made it the number 128161)
    check(`B8 (T7) the callout node's data-emoji-unicode is the string "128161"`,
      fromDoc.length === 1 && fromDoc[0]["data-emoji-unicode"] === "128161", JSON.stringify(fromDoc.map((a) => a["data-emoji-unicode"])));
    await shot(page, "B8-1-callout");
  });

  // a callout inserted with "/callout" in the create modal's description takes the logo last picked (localStorage)
  await scenario("B8 new callout takes the stored logo", stubs(TOUR_DONE), async (page) => {
    await page.context().addInitScript((logo) => localStorage.setItem("editor-calloutComponent-logo", logo), STORED_LOGO);
    await goto(page, "/probe-ws/projects/p1/issues");
    await page.getByRole("button", { name: "Add work item" }).click();
    const editor = page.locator(MODAL_EDITOR);
    check("B8 the create work item modal opens", await visible(editor));
    await editor.click();
    await page.keyboard.type("/callout");
    await page.waitForTimeout(700);
    await page.keyboard.press("Enter");
    const block = page.locator("#editor-container-issue-modal-editor .editor-callout-component").first();
    check("B8 /callout inserts a callout", await visible(block));
    await page.waitForTimeout(700);
    const icon = (await block.locator("button").first().innerText()).trim();
    check("B8 the new callout's icon is the stored emoji 😀 (128512)", icon === "😀", JSON.stringify(icon));
    const { fromDoc, fromHtml, html } = await callouts(page, MODAL_EDITOR);
    console.log(`  (node data-emoji-url ${JSON.stringify(fromDoc.map((a) => a["data-emoji-url"]))}; in the HTML output ${JSON.stringify(fromHtml.map((a) => a["data-emoji-url"]))})`);
    console.log(`  (the editor's HTML: ${html})`);
    // the URL is not rendered into the page (the icon is the emoji character); it lives in the node and the HTML
    const inDom = await page.evaluate((host) =>
      [...document.querySelectorAll("*")].some((el) => [...el.attributes].some((a) => a.value.includes(host))), "cdn.example.com");
    console.log(`  (the stored emoji URL ${inDom ? "is" : "is not"} in any attribute of the page)`);
    check(`B8 the new callout node's data-emoji-url is ${GRIN_URL} (a plain &)`,
      fromDoc.length === 1 && fromDoc[0]["data-emoji-url"] === GRIN_URL, JSON.stringify(fromDoc.map((a) => a["data-emoji-url"])));
    check(`B8 the editor's HTML output, parsed back, holds the same address (a plain &, no &amp;)`,
      fromHtml.length === 1 && fromHtml[0]["data-emoji-url"] === GRIN_URL, JSON.stringify(fromHtml.map((a) => a["data-emoji-url"])));
    await shot(page, "B8-2-new-callout");
  });

  // ---------------------------------------------------------------- 9. theme names (T11)
  for (const language of ["zh-CN", "en"]) {
    const want = THEME_NAMES[language];
    await scenario(`B9 theme names (${language})`, stubs(profileIn(language)), async (page) => {
      // Power K, on the workspace's projects page
      await goto(page, "/probe-ws/projects");
      const lang = await page.waitForFunction((l) => document.documentElement.lang === l, language, { timeout: 15000 }).then(() => true, () => false);
      check(`B9 (${language}) the UI language follows the profile (<html lang="${language}">)`, lang);
      await visible(page.getByText("Probe Project").first());
      await page.waitForTimeout(500);
      await page.keyboard.press("ControlOrMeta+k");
      check(`B9 (${language}) Power K opens`, await visible(page.locator("[cmdk-input]"), 5000));
      const command = page.locator("[cmdk-item]", { hasText: CHANGE_THEME[language] }).first();
      check(`B9 (${language}) Power K offers "${CHANGE_THEME[language]}"`, await visible(command, 5000));
      await command.click();
      await page.waitForTimeout(700);
      const menu = (await page.locator("[cmdk-item]").allInnerTexts()).map((t) => t.trim());
      console.log(`  (Power K theme menu: ${JSON.stringify(menu)})`);
      await shot(page, `B9-1-power-k-${language}`);
      check(`B9 (${language}) Power K's theme menu names the themes ${JSON.stringify(want)}`, JSON.stringify(menu) === JSON.stringify(want), JSON.stringify(menu));
      await page.keyboard.press("Escape");
      await page.keyboard.press("Escape");

      // the profile settings' theme switch: its current value and its five options
      await goto(page, "/settings/profile/preferences");
      const current = page.getByRole("button", { name: want[0] });
      const shown = await visible(current, 15000);
      const label = shown ? (await current.first().innerText()).trim() : (await page.locator("main, body").first().innerText()).slice(0, 300);
      check(`B9 (${language}) the settings' theme switch shows the current theme as "${want[0]}"`, shown, JSON.stringify(label));
      if (shown) await current.first().click();
      const options = page.getByRole("option");
      await visible(options, 5000);
      const names = (await options.allInnerTexts()).map((t) => t.trim());
      console.log(`  (settings theme switch: current ${JSON.stringify(label)}; options ${JSON.stringify(names)})`);
      await shot(page, `B9-2-settings-${language}`);
      check(`B9 (${language}) the switch's options name the themes ${JSON.stringify(want)}`, JSON.stringify(names) === JSON.stringify(want), JSON.stringify(names));
    });
  }

  // ---------------------------------------------------------------- 10. the work item form's cycles (T5)
  // Opened from the workspace home, the form's project (p1) is not the route's (none): the form itself fetches the
  // project's cycles (T5: the cycle store's fetchAllCycles instead of useProjectIssueProperties' fetchCycles).
  await scenario("B10 work item form cycles, from the workspace home", stubs(TOUR_DONE), async (page, ctx) => {
    const cycleRequests = () => ctx.requests.filter((r) => r.key === `GET ${P}/cycles/`).length;
    await goto(page, "/probe-ws");
    const newWorkItem = page.getByRole("button", { name: "New work item" }).first();
    check("B10 the workspace home is up", await visible(newWorkItem));
    await page.waitForTimeout(1500);
    check(`B10 the workspace home itself does not ask for the project's cycles`, cycleRequests() === 0, `${cycleRequests()} requests`);
    await newWorkItem.click();
    const modal = page.locator("form").filter({ hasText: "Create new work item" });
    check("B10 the create work item modal opens, on project p1", await visible(modal) && (await modal.textContent()).includes(project.name));
    await page.waitForTimeout(1500);
    check(`B10 the form fetches the project's cycles (GET ${P}/cycles/) before the cycle dropdown opens`, cycleRequests() > 0,
      JSON.stringify(ctx.requests.map((r) => r.key)));
    await shot(page, "B10-1-modal");
    await modal.getByRole("button", { name: "Cycle", exact: true }).last().click();
    check(`B10 the cycle dropdown lists the project's cycle ("${cycle.name}")`, await visible(cycleOptions(page)));
    await shot(page, "B10-2-cycles");
  });

  // the same form opened inside the project (the route's project is the form's)
  await scenario("B10 work item form cycles, inside the project", stubs(TOUR_DONE), async (page, ctx) => {
    await goto(page, "/probe-ws/projects/p1/issues");
    await page.getByRole("button", { name: "Add work item" }).click();
    const modal = page.locator("form").filter({ hasText: "Create new work item" });
    check("B10 (in project) the create work item modal opens", await visible(modal));
    await page.waitForTimeout(1000);
    await modal.getByRole("button", { name: "Cycle", exact: true }).last().click();
    check(`B10 (in project) the cycle dropdown lists the project's cycle ("${cycle.name}")`, await visible(cycleOptions(page)));
    check(`B10 (in project) GET ${P}/cycles/ was requested`, ctx.requests.some((r) => r.key === `GET ${P}/cycles/`));
    await shot(page, "B10-3-in-project");
  });

  // ---------------------------------------------------------------- 11. empty-state illustrations (T4)
  // The empty states draw their illustration as an inline <svg> (propel's asset registry, by assetKey), not an <img>:
  // the check finds the empty state by its title and asks for an <svg> of at least 40x40 px with shapes in it next
  // to the title; every <img> on the page must have loaded as well. The intake page has two: the list's (with the
  // default "Status: Pending" filter it is the "no matching results" one, assetKey "search") and the main pane's
  // (assetKey "intake").
  for (const [name, extra, address, titles] of [
    ["project list", NO_PROJECTS, "/probe-ws/projects", ["No active projects"]],
    ["cycle list", NO_CYCLES, "/probe-ws/projects/p1/cycles", ["Group and timebox your work in Cycles."]],
    ["intake", INTAKE, "/probe-ws/projects/p1/intake", ["No matching results.", "Select an Intake work item to view its details"]],
  ])
    await scenario(`B11 ${name} empty state`, stubs({ ...TOUR_DONE, ...extra }), async (page) => {
      await goto(page, address);
      for (const title of titles) {
        const heading = page.getByText(title, { exact: true });
        check(`B11 ${name}: ${address} shows the empty state "${title}"`, await visible(heading));
        await page.waitForTimeout(1000);
        const found = await illustrationNextTo(page, title);
        check(`B11 ${name}: "${title}" shows its illustration (an <svg> of at least 40x40 px with shapes)`,
          found.svgs.some((s) => s.width >= 40 && s.height >= 40 && s.shapes >= 5), JSON.stringify(found));
        console.log(`  (illustration next to "${title}": ${JSON.stringify(found.svgs)})`);
        // the probe's own control: with the illustration taken out of the page, the same check finds none
        const removed = await page.evaluate((title) => {
          const own = [...document.querySelectorAll("body *")].find((el) => el.childElementCount === 0 && el.textContent.trim() === title);
          const svgs = [...(own?.closest('[class*="max-w-[25rem]"]')?.querySelectorAll("svg") ?? [])].filter((s) => s.getBoundingClientRect().width >= 40);
          const parked = svgs.map((s) => [s, s.parentNode, s.nextSibling]);
          svgs.forEach((s) => s.remove());
          window.__parked = parked;
          return svgs.length;
        }, title);
        const without = await illustrationNextTo(page, title);
        await page.evaluate(() => window.__parked.forEach(([s, parent, next]) => parent.insertBefore(s, next)));
        check(`B11 ${name}: (probe control) with its ${removed} illustration(s) taken out, the check finds none`,
          removed > 0 && without.container && without.svgs.length === 0, JSON.stringify({ removed, without }));
      }
      const images = await page.evaluate(() =>
        [...document.querySelectorAll("img")].map((i) => ({ src: i.getAttribute("src"), complete: i.complete, naturalWidth: i.naturalWidth })));
      check(`B11 ${name}: every <img> on the page loaded (naturalWidth > 0; ${images.length} on the page)`,
        images.every((i) => i.complete && i.naturalWidth > 0), JSON.stringify(images));
      await shot(page, `B11-${name.replace(/ /g, "-")}`);
    });
});
```

#### 7.3.8 `run-b.sh`：B 组的入口

在给定端口上跑 `closeout-b.mjs`。

```bash
#!/bin/bash
# One-off (M1/closeout browser checks, plan "控制者的浏览器核对" group B, and group C item 12 on the base build): runs
# closeout-b.mjs on the build in <repo>. Output: $COTMP/probe/logs/<tag>/closeout-b.txt, screenshots in
# $COTMP/probe/shots/b-<tag>/. usage: bash run-b.sh <tag> <repo> <port>
set -u
C=/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-co/probe
TAG=$1
OUT=$C/logs/$TAG
mkdir -p "$OUT"
cd "$2"
head=$(cat .git/HEAD)
case $head in "ref: "*) head="$head = $(cat ".git/${head#ref: }" 2>/dev/null || grep " ${head#ref: }\$" .git/packed-refs)" ;; esac
echo "build at $head ($2)"
env PROBE_PORT="$3" OUT="$C/shots/b-$TAG" node "$C/closeout-b.mjs" > "$OUT/closeout-b.txt" 2>&1
echo "closeout-b: exit $?, PASS $(grep -c '^PASS' "$OUT/closeout-b.txt"), FAIL $(grep -c '^FAIL' "$OUT/closeout-b.txt"), last: $(tail -1 "$OUT/closeout-b.txt")"
grep "^FAIL" "$OUT/closeout-b.txt" | cut -c1-200
```

#### 7.3.9 `covers-probe.mjs`：第 12 项（T14 的预设封面）

架构师写的原件的逐字节副本（`cmp` 核对）。

```js
// Closeout T14 browser check: the new preset covers in the app's cover slots, on the build of the repository at the
// current directory (P5's probe library: a static server over web/apps/web/build/client, stubbed /api/ and /auth/).
// - A1: the projects page with eight projects, each on a different preset cover (the project card);
// - A2: the create-project modal (its header shows a random preset), the cover picker with all 29 presets, and the
//   header after picking one; then the form is submitted and the preset's upload request is recorded: the app
//   uploads a copy of the preset, so its detected type must be image/webp and its name the preset's file name;
// - A3: the project settings (the h-44 cover over the full width), A4: the profile settings and A5: the user menu
//   (which falls back to the default cover, image_1).
// Screenshots go to $OUT. Every scenario fails on an unlisted request, an uncaught page error and an image on the
// page that failed to load.
// usage: OUT=<dir> node covers-probe.mjs [scenario filter]   (from the repository root of the build)
import fs from "node:fs";
import path from "node:path";
import { Reply, WS, check, goto, probe, scenario, settle, visible } from "../../../nerve-p5/probe/lib.mjs";
import { P, W, project } from "../../../nerve-p4/probe-c/p3data.mjs";
import { stubs } from "../../../nerve-p4/probe-c/stubs.mjs";

const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });
const shot = (target, name) => target.screenshot({ path: path.join(out, `${name}.png`) });

// the build's address of each preset, by number
const assets = fs.readdirSync("web/apps/web/build/client/assets");
const preset = (n) => {
  // .jpg too, so that the negative control on a build with the old photos runs to the end (and fails there)
  const f = assets.find((a) => new RegExp(`^image_${n}-[\\w-]+\\.(?:webp|jpg)$`).test(a));
  if (!f) throw new Error(`no built preset ${n}`);
  return `/assets/${f}`;
};
const PRESETS = Array.from({ length: 29 }, (_, i) => preset(i + 1));
check("the build has 29 WebP presets and no cover JPEG", PRESETS.every((p) => p.endsWith(".webp")) && !assets.some((a) => /^image_\d+-.*\.jpg$/.test(a)), JSON.stringify(PRESETS.slice(0, 2)));

// every <img> on the page that is complete but has no pixels
const brokenImages = (page) =>
  page.evaluate(() => [...document.querySelectorAll("img")].filter((i) => i.complete && i.naturalWidth === 0).map((i) => i.getAttribute("src")));
const noBroken = async (page, name) => {
  const broken = await brokenImages(page);
  check(`${name}: no image failed to load`, broken.length === 0, JSON.stringify(broken));
};
const coverSrcs = (page) =>
  page.evaluate(() => [...document.querySelectorAll("img")].map((i) => i.getAttribute("src")).filter((s) => /\/assets\/image_\d+-/.test(s ?? "")));

const projects = Array.from({ length: 8 }, (_, i) => ({
  ...project,
  id: i === 0 ? project.id : `p${i + 1}`,
  name: `Cover ${[1, 5, 9, 13, 17, 21, 25, 29][i]}`,
  identifier: `CV${i + 1}`,
  cover_image_url: PRESETS[[1, 5, 9, 13, 17, 21, 25, 29][i] - 1],
  sort_order: 65535 * (i + 1),
}));

probe(async () => {
  await scenario("A1 project cards", stubs({ [`GET ${W}/projects/`]: projects, [`GET ${W}/projects/details/`]: projects }), async (page) => {
    await goto(page, `/${WS}/projects`);
    await settle(page, `/${WS}/projects`);
    await visible(page.getByText("Cover 29"));
    await page.waitForTimeout(1500);
    const shown = await coverSrcs(page);
    check("A1 the cards show the eight presets", projects.every((p) => shown.includes(p.cover_image_url)), JSON.stringify(shown));
    await noBroken(page, "A1");
    await shot(page, "a1-project-cards");
  });

  const uploads = [];
  const writes = [];
  const record = (reply) => (request) => {
    writes.push(`${request.method()} ${new URL(request.url()).pathname} ${request.postData() ?? ""}`.slice(0, 300));
    return reply;
  };
  const created = { ...project, id: "pnew", name: "Cover probe", identifier: "COVER" };
  const ASSET = `/api/assets/v2/workspaces/${WS}/asset-1/`;
  await scenario(
    "A2 create project and the picker",
    stubs({
      // the upload chain: signed URL, the object store's POST, the upload status; then the project and its cover
      [`POST /api/assets/v2/workspaces/${WS}/`]: (request) => {
        uploads.push(JSON.parse(request.postData() ?? "{}"));
        return { upload_data: { url: "/api/probe-object-store/", fields: { key: "k" } }, asset_id: "asset-1", asset_url: ASSET };
      },
      "POST /api/probe-object-store/": record(new Reply(204, undefined)),
      [`PATCH ${ASSET}`]: record({}),
      // the stored copy, as the asset store would serve it (here: the preset the copy was made from)
      [`GET ${ASSET}`]: new Reply(302, undefined, { location: PRESETS[16] }),
      [`POST ${W}/projects/`]: record(created),
      // (the bulk address repeats the project id: entity and project)
      [`POST /api/assets/v2/workspaces/${WS}/projects/pnew/pnew/bulk/`]: record({}),
      [`PATCH ${W}/projects/pnew/`]: record({ ...created, cover_image_url: ASSET }),
    }),
    async (page) => {
      await goto(page, `/${WS}/projects`);
      await settle(page, `/${WS}/projects`);
      await page.getByRole("button", { name: /Add project/i }).first().click();
      const header = page.getByAltText("Project cover image");
      const open = await visible(header);
      if (!open) await shot(page, "a2-debug");
      check("A2 the modal's header shows a preset", open, "");
      const first = await header.getAttribute("src");
      check("A2 the header's preset is one of the 29", PRESETS.includes(first), first);
      await page.waitForTimeout(800);
      await shot(page, "a2-create-modal");
      await page.getByRole("button", { name: /Change cover/i }).click();
      const tiles = page.getByAltText(/^Cover image \d+$/);
      await visible(tiles);
      await page.waitForTimeout(1500);
      check("A2 the picker offers 29 presets", (await tiles.count()) === 29, String(await tiles.count()));
      const srcs = await tiles.evaluateAll((els) => els.map((e) => e.getAttribute("src")));
      check("A2 the picker's presets are the built WebP files, in order", JSON.stringify(srcs) === JSON.stringify(PRESETS), JSON.stringify(srcs.slice(0, 3)));
      await noBroken(page, "A2 picker");
      const panel = page.locator("div.md\\:h-\\[36rem\\]").first();
      await shot(panel, "a2-picker-top");
      await tiles.nth(28).scrollIntoViewIfNeeded();
      await page.waitForTimeout(800);
      await shot(panel, "a2-picker-bottom");
      await tiles.nth(16).scrollIntoViewIfNeeded();
      await tiles.nth(16).click();
      await page.waitForTimeout(800);
      check("A2 the header shows the picked preset 17", (await header.getAttribute("src")) === PRESETS[16], await header.getAttribute("src"));
      await shot(page, "a2-picked-17");
      await page.getByPlaceholder(/project name/i).fill("Cover probe");
      await page.getByRole("button", { name: /^Create project$/i }).click();
      await page.waitForTimeout(2000);
      console.log(`  upload requests: ${JSON.stringify(uploads)}`);
      const u = uploads[0] ?? {};
      check("A2 the preset is uploaded as a WebP copy under its file name", uploads.length === 1 && u.type === "image/webp" && u.name === PRESETS[16].split("/").pop() && u.size > 0, JSON.stringify(uploads));
      console.log(`  writes: ${JSON.stringify(writes.map((w) => w.replace(/\s+/g, " ").slice(0, 160)), null, 1)}`);
      check("A2 the project stores the uploaded copy's address, not the preset's", writes.some((w) => w.startsWith(`PATCH ${W}/projects/pnew/`) && w.includes(`"cover_image_url":"${ASSET}"`)), "");
      await shot(page, "a2-after-create");
    }
  );

  const withCover = { ...project, cover_image_url: PRESETS[6] };
  await scenario("A3 project settings", stubs({ [`GET ${W}/projects/`]: [withCover], [`GET ${W}/projects/details/`]: [withCover], [`GET ${P}/`]: withCover }), async (page) => {
    await goto(page, `/${WS}/settings/projects/${project.id}`);
    await page.waitForTimeout(2500);
    const shown = await coverSrcs(page);
    check("A3 the settings show preset 7", shown.includes(PRESETS[6]), JSON.stringify(shown));
    await noBroken(page, "A3");
    await shot(page, "a3-project-settings");
  });

  await scenario("A4 profile settings and A5 user menu", stubs({ "GET /api/users/me/": { ...stubs()["GET /api/users/me/"], cover_image_url: PRESETS[21] } }), async (page) => {
    await goto(page, "/settings/profile/general");
    await page.waitForTimeout(2500);
    const shown = await coverSrcs(page);
    check("A4 the profile shows preset 22", shown.includes(PRESETS[21]), JSON.stringify(shown));
    await noBroken(page, "A4");
    await shot(page, "a4-profile-settings");
  });

  await scenario("A5 user menu, no cover", stubs(), async (page) => {
    await goto(page, `/${WS}/projects`);
    await settle(page, `/${WS}/projects`);
    await page.waitForTimeout(1000);
    const menu = page.locator("[data-headlessui-state] button, button").filter({ has: page.locator("img, span") });
    await page.getByRole("button", { name: /probe/i }).first().click().catch(() => menu.first().click());
    await page.waitForTimeout(1000);
    const shown = await coverSrcs(page);
    check("A5 the user menu falls back to the default preset (1)", shown.includes(PRESETS[0]), JSON.stringify(shown));
    await noBroken(page, "A5");
    await shot(page, "a5-user-menu");
  });
});
```

#### 7.3.10 `pictures-probe.mjs`：第 13 项（T15、T20 改过的图）

架构师写的原件的逐字节副本。页面拿到的字节与仓库里的文件比较，所以 T20 改过的图也在核对之内。

```js
// Closeout T15 browser check: the retouched pictures where the app shows them, on the build of the repository at the
// current directory (P5's probe library: a static server over web/apps/web/build/client, stubbed /api/ and /auth/).
// - D1: the four "feature disabled" pages of a project (cycles, modules, views, intake with the project's flag off),
//   light and dark: the page shows <feature>-<theme>.webp, the bytes it was served are the repository's file, and
//   they are not the file of c493255 (the closeout's base: the photographs);
// - D2: the product tour, step by step: each step shows its picture, served as the repository has it; the three
//   retouched ones differ from c493255's (modules.webp has no avatar and does not change);
// - D3: the cycles page with a completed cycle and no active one: the active-cycle section shows its empty state
//   ("No active cycle", propel's drawing) and asks for no active-cycle picture (T15 deletes the two it never showed).
// Screenshots go to $OUT. Every scenario fails on an unlisted request, an uncaught page error and an image on the
// page that failed to load. Negative control: on the base's build D1 and D2 fail "differs from c493255's".
// usage: OUT=<dir> node pictures-probe.mjs [scenario filter]   (from the repository root of the build)
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { WS, check, goto, probe, scenario, visible } from "../../../nerve-p5/probe/lib.mjs";
import { P, W, cycle, iso, now, project } from "../../../nerve-p4/probe-c/p3data.mjs";
import { stubs } from "../../../nerve-p4/probe-c/stubs.mjs";

const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });
const shot = (page, name) => page.screenshot({ path: path.join(out, `${name}.png`) });
const A = "web/apps/web/app/assets";
const BASE = "/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-co/base/web/apps/web/app/assets";
const sha = (file) => crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex");

// every <img> on the page whose src names `stem` (its hashed build name), with the SHA-256 of the bytes it was served
const shownImages = (page, stem) =>
  page.evaluate(async (stem) => {
    const found = [];
    for (const img of document.querySelectorAll("img")) {
      const src = img.getAttribute("src") ?? "";
      if (!new RegExp(`/${stem}-[\\w-]+\\.webp$`).test(src)) continue;
      const buf = await (await fetch(img.src)).arrayBuffer();
      const hash = [...new Uint8Array(await crypto.subtle.digest("SHA-256", buf))].map((b) => b.toString(16).padStart(2, "0")).join("");
      found.push({ src, sha: hash });
    }
    return found;
  }, stem);
const brokenImages = (page) =>
  page.evaluate(() => [...document.querySelectorAll("img")].filter((i) => i.complete && i.naturalWidth === 0).map((i) => i.getAttribute("src")));
const noBroken = async (page, name) => {
  const broken = await brokenImages(page);
  check(`${name}: no image failed to load`, broken.length === 0, JSON.stringify(broken));
};
// the page shows the picture `rel`, served as the repository has it; `changed`: and not as c493255 had it
const checkPicture = async (page, name, rel, changed) => {
  const stem = rel.split("/").pop().replace(/\.webp$/, "");
  let shown = [];
  for (let i = 0; i < 30 && !shown.length; i++) {
    shown = await shownImages(page, stem);
    if (!shown.length) await page.waitForTimeout(500);
  }
  const want = sha(path.join(A, rel));
  check(`${name}: shows ${stem}, served as the repository has it`, shown.length > 0 && shown.every((i) => i.sha === want), JSON.stringify(shown));
  if (changed) {
    const before = sha(path.join(BASE, rel));
    check(`${name}: ${stem} differs from c493255's`, shown.length > 0 && shown.every((i) => i.sha !== before), JSON.stringify(shown.map((i) => i.sha.slice(0, 16))));
  }
};
const theme = (page, t) => page.addInitScript((t) => localStorage.setItem("theme", t), t);

probe(async () => {
  for (const t of ["light", "dark"]) {
    for (const [feature, flag] of [["cycles", "cycle_view"], ["modules", "module_view"], ["views", "issue_views_view"], ["intake", "inbox_view"]]) {
      const off = { ...project, [flag]: false };
      await scenario(`D1 ${feature} disabled (${t})`, stubs({ [`GET ${P}/`]: off, [`GET ${W}/projects/`]: [off], [`GET ${W}/projects/details/`]: [off] }), async (page) => {
        await theme(page, t);
        await goto(page, `/${WS}/projects/${project.id}/${feature}`);
        await checkPicture(page, `D1 ${feature} disabled (${t})`, `empty-state/disabled-feature/${feature}-${t}.webp`, true);
        await page.waitForTimeout(500);
        await noBroken(page, `D1 ${feature} disabled (${t})`);
        await shot(page, `d1-${feature}-disabled-${t}`);
      });
    }
  }

  await scenario("D2 product tour", stubs({
    "GET /api/users/me/profile/": { id: "pr1", user: "u1", language: "en", is_onboarded: true, is_tour_completed: false, theme: {} },
  }), async (page) => {
    await goto(page, `/${WS}`);
    const start = page.getByRole("button", { name: "Take a Product Tour" });
    check("D2 the tour's welcome shows", await visible(start, 20000));
    await start.click();
    const steps = [["Plan with work items", "issues", true], ["Move with cycles", "cycles", true], ["Break into modules", "modules", false], ["Views", "views", true]];
    for (const [k, [title, stem, changed]] of steps.entries()) {
      check(`D2 step ${k + 1} "${title}" shows`, await visible(page.getByAltText(title, { exact: true }), 10000));
      await checkPicture(page, `D2 step ${k + 1}`, `onboarding/${stem}.webp`, changed);
      await page.waitForTimeout(500);
      await noBroken(page, `D2 step ${k + 1}`);
      await shot(page, `d2-tour-${k + 1}-${stem}`);
      if (k < steps.length - 1) await page.getByRole("button", { name: "Next", exact: true }).click();
    }
  });

  const done = { ...cycle, status: "completed", start_date: iso(now - 21 * 86400000), end_date: iso(now - 14 * 86400000) };
  for (const t of ["light", "dark"]) {
    await scenario(`D3 no active cycle (${t})`, stubs({ [`GET ${P}/cycles/`]: [done], [`GET ${W}/cycles/`]: [done], [`GET ${P}/cycles/c1/`]: done }), async (page) => {
      await theme(page, t);
      await goto(page, `/${WS}/projects/${project.id}/cycles`);
      check(`D3 (${t}) the active-cycle section shows`, await visible(page.getByText("Active cycle", { exact: true }), 20000));
      check(`D3 (${t}) its empty state shows`, await visible(page.getByText("No active cycle", { exact: true }), 10000));
      await page.waitForTimeout(1000);
      const active = await page.evaluate(() => [...document.querySelectorAll("img")].map((i) => i.getAttribute("src") ?? "").filter((s) => /\/active-(light|dark)-/.test(s)));
      check(`D3 (${t}) no active-cycle picture on the page`, active.length === 0, JSON.stringify(active));
      await noBroken(page, `D3 no active cycle (${t})`);
      await shot(page, `d3-no-active-cycle-${t}`);
    });
  }
});
```

#### 7.3.11 `copy-probe.mjs`：第 14 项（T17 的粘贴）

T17 的实现者写的原件的逐字节副本。

```js
// Closeout T17 browser check: copying between Nerve editors still duplicates an uploaded image, on the build of the
// repository at the current directory (P5's probe library: a static server over web/apps/web/build/client, stubbed
// /api/ and /auth/). Work item PRB-1's description holds text and an uploaded image (its src is the asset's id).
// - C1: select the description, copy it with the keyboard, paste it into the comment editor with the keyboard: the
//   clipboard carries text/nerve-editor-html, the app asks the server to duplicate the asset once, and the comment
//   editor shows the text and an image whose src is the new asset's id;
// - C2 (what deleting the custom-type branch would leave, measured without changing the code): the same copied
//   HTML pasted as text/html only. No duplication is asked for, and the comment's image keeps the description's asset.
// Screenshots go to $OUT.
// usage: OUT=<dir> node copy-probe.mjs [scenario filter]   (from the repository root of the build)
import fs from "node:fs";
import path from "node:path";
import { check, goto, probe, scenario, visible } from "../../../nerve-p5/probe/lib.mjs";
import { ISSUES, P, W } from "../../../nerve-p4/probe-c/p3data.mjs";
import { stubs } from "../../../nerve-p4/probe-c/stubs.mjs";

const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });
const shot = (target, name) => target.screenshot({ path: path.join(out, `${name}.png`) });

const SOURCE = "a0000000-0000-4000-8000-000000000001";
const COPY = "a0000000-0000-4000-8000-000000000002";
// a picture the build ships, served for either asset
const picture = `/assets/${fs.readdirSync("web/apps/web/build/client/assets").find((a) => /^image_1-[\w-]+\.(?:webp|jpg)$/.test(a))}`;
const alpha = {
  ...ISSUES[0],
  description_html:
    '<p class="editor-paragraph-block" data-id="5f1c2a4e-1111-4a2b-9c3d-000000000001">Alpha description text</p>' +
    `<image-component data-id="5f1c2a4e-1111-4a2b-9c3d-000000000002" id="5f1c2a4e-1111-4a2b-9c3d-000000000002" src="${SOURCE}" width="200px" height="auto" aspectRatio="1.5" alignment="left" status="uploaded"></image-component>`,
};
const duplications = [];
const saves = [];
const scenarioStubs = () =>
  stubs({
    [`GET ${P}/issues/i1/`]: alpha,
    [`GET ${W}/work-items/PRB-1/`]: alpha,
    // the description editor saves what it shows once (see "saves" below)
    [`PATCH ${P}/issues/i1/`]: (request) => {
      saves.push(request.postData());
      return alpha;
    },
    [`POST /api/assets/v2/workspaces/probe-ws/duplicate-assets/${SOURCE}/`]: (request) => {
      duplications.push(request.postData());
      return { asset_id: COPY };
    },
  });

// the image nodes an editor shows, by the asset id each points to
const imageSources = (editor) =>
  editor.evaluate((el) => [...el.querySelectorAll("image-component, [data-type=imageComponent], img")].map((i) => i.getAttribute("src")));

probe(async () => {
  await scenario("C1 copy and paste between editors", scenarioStubs(), async (page, ctx) => {
    await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
    await page.context().route(/\/api\/assets\/v2\/workspaces\/probe-ws\/projects\/p1\/[\w-]+\/$/, (route) =>
      route.fulfill({ status: 302, headers: { location: picture } })
    );
    await goto(page, `/probe-ws/browse/PRB-1`);
    const desc = page.locator("#editor-container-i1 .ProseMirror");
    check("C1 the description editor shows", await visible(desc, 20000));
    await page.waitForTimeout(1500);
    await shot(page, "c1-description");
    await desc.click();
    await page.keyboard.press("ControlOrMeta+a");
    await page.keyboard.press("ControlOrMeta+c");
    const box = page.locator("#editor-container-add_comment_i1 .ProseMirror");
    check("C1 the comment editor shows", await visible(box, 10000));
    await box.click();
    await page.keyboard.press("ControlOrMeta+v");
    await page.waitForTimeout(2500);
    const text = await box.innerText();
    check("C1 the comment editor has the description's text", /Alpha description text/.test(text), text);
    check("C1 the app asked once to duplicate the uploaded image", duplications.length === 1, JSON.stringify(duplications));
    const html = await box.evaluate((el) => el.innerHTML);
    check("C1 the pasted image points to the new asset", html.includes(COPY) || (await imageSources(box)).some((s) => s?.includes(COPY)), html.slice(0, 400));
    await shot(page, "c1-pasted");
    console.log(`  requests: ${JSON.stringify(ctx.requests.filter((r) => /assets/.test(r.key)).map((r) => r.key))}`);
    console.log(`  description saves: ${JSON.stringify(saves.map((s) => s?.slice(0, 300)))}`);
  });

  duplications.length = 0;
  await scenario("C2 the same copy pasted as text/html only", scenarioStubs(), async (page, ctx) => {
    await page.context().route(/\/api\/assets\/v2\/workspaces\/probe-ws\/projects\/p1\/[\w-]+\/$/, (route) =>
      route.fulfill({ status: 302, headers: { location: picture } })
    );
    await goto(page, `/probe-ws/browse/PRB-1`);
    const desc = page.locator("#editor-container-i1 .ProseMirror");
    check("C2 the description editor shows", await visible(desc, 20000));
    await page.waitForTimeout(1500);
    await desc.click();
    await page.keyboard.press("ControlOrMeta+a");
    const copied = await desc.evaluate((el) => {
      const dt = new DataTransfer();
      el.dispatchEvent(new ClipboardEvent("copy", { clipboardData: dt, bubbles: true, cancelable: true }));
      return { types: [...dt.types], html: dt.getData("text/html") };
    });
    console.log(`  copied types: ${JSON.stringify(copied.types)}`);
    const box = page.locator("#editor-container-add_comment_i1 .ProseMirror");
    check("C2 the comment editor shows", await visible(box, 10000));
    await box.click();
    await box.evaluate((el, html) => {
      const dt = new DataTransfer();
      dt.setData("text/html", html);
      el.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true, cancelable: true }));
    }, copied.html);
    await page.waitForTimeout(2500);
    const html = await box.evaluate((el) => el.innerHTML);
    console.log(`  duplication requests: ${duplications.length}`);
    check("C2 without the custom type no duplication is asked for", duplications.length === 0, JSON.stringify(duplications));
    check("C2 the pasted image keeps the description's asset", html.includes(SOURCE) || (await imageSources(box)).some((s) => s?.includes(SOURCE)), html.slice(0, 400));
    await shot(page, "c2-pasted-html-only");
    console.log(`  requests: ${JSON.stringify(ctx.requests.filter((r) => /assets/.test(r.key)).map((r) => r.key))}`);
  });

  // C3 (the negative control): a page of another origin can write the editor's clipboard type in its copy event.
  // Paste HTML that carries an <img> whose onerror runs a script into the comment editor. At the fix the paste is
  // parsed in an inert document, so the handler never runs (window.__xss stays false); on c493255 the old code sets
  // it as innerHTML of a live-document div, the image fails to load and the handler runs (window.__xss becomes true).
  await scenario("C3 a cross-origin clipboard payload does not run a script", scenarioStubs(), async (page, ctx) => {
    await page.addInitScript(() => {
      window.__xss = false;
    });
    // the editor resolves any image it holds (the description's, and the "x" of the payload); answer them so the
    // scenario's "no unlisted request" check stays about the paste, not about image resolution
    await page.route(/\/api\/assets\//, (route) => route.fulfill({ status: 302, headers: { location: picture } }));
    await goto(page, `/probe-ws/browse/PRB-1`);
    const box = page.locator("#editor-container-add_comment_i1 .ProseMirror");
    check("C3 the comment editor shows", await visible(box, 20000));
    await box.click();
    await page.waitForTimeout(300);
    await box.evaluate((el) => {
      const payload = '<p>pwned?</p><img src="x" onerror="window.__xss = true">';
      const dt = new DataTransfer();
      dt.setData("text/nerve-editor-html", payload);
      el.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true, cancelable: true }));
    });
    await page.waitForTimeout(1500);
    const ran = await page.evaluate(() => window.__xss === true);
    console.log(`  window.__xss after the paste: ${ran}`);
    check("C3 the clipboard payload's onerror did not run", ran === false, `window.__xss=${ran}`);
    check("C3 the text was still pasted", /pwned\?/.test(await box.innerText()), await box.innerText());
    check("C3 no image with an event handler is in the editor", !/onerror/i.test(await box.evaluate((el) => el.innerHTML)), "found onerror");
    await shot(page, "c3-payload");
    void ctx;
  });
});
```

#### 7.3.12 `active-cycle.mjs`：T19 的活动迭代卡片

第 4 轮新加：有取消的工作项时卡片的文字、进度条的值、取消的说明、各状态组的行和它们的 React 列表键（中英文）。

```js
// One-off (M1/closeout browser checks, probe run 4, T19): the active-cycle card on a project's cycles page, for a
// cycle with cancelled work items. The progress counts only completed work items against the closable ones (total
// minus cancelled), as calculateCycleProgress does; the header and the note come from i18n keys; the group rows are
// keyed list items.
//   A (total 10: completed 3, cancelled 2, started 3, unstarted 1, backlog 1), en: the header reads "3/8 work items
//     completed", the bar's aria-valuenow is 38, the note reads "2 cancelled work items are excluded from this
//     report.", the rows show completed 3, started 3, unstarted 1, backlog 1, and every item of the rows' list has
//     its own React key; zh-CN: the header "3/8 个工作项已完成", the note "报告中已排除 2 个已取消的工作项。";
//   B (total 10: completed 8, cancelled 2), en: the header reads "8/8 work items completed" and the bar is at 100.
// The keys: the builds are production React, which never prints "Each child in a list should have a unique key"
// (neither build's assets hold that text), so a console check would pass on any build. The probe reads what that
// warning is about from React's fibers instead: the list the card's map() returns is one Fragment fiber under the
// rows' container, and each of its children must carry a key of its own. The console's key warnings are still
// counted and printed. The server, the stubs and the per-scenario checks (no request outside the scenario's list,
// no uncaught page error, no navigate() warning) are P4's (nerve-p4/probe-a/lib.mjs, copied), the data P4's group C
// data (p3data.mjs, stubs.mjs; its cycle c1 is the project's current cycle). Screenshots go to $OUT.
// usage: PROBE_PORT=<port> OUT=<screenshot dir> node active-cycle.mjs [scenario name filter]   (from the repository
// root of the build)
import fs from "node:fs";
import path from "node:path";
import { check, goto, probe, scenario, visible } from "./nerve-p4/probe-a/lib.mjs";
import { P, W, cycle, project } from "./nerve-p4/probe-c/p3data.mjs";
import { stubs } from "./nerve-p4/probe-c/stubs.mjs";

const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });

const COUNTS = {
  A: { total_issues: 10, completed_issues: 3, cancelled_issues: 2, started_issues: 3, unstarted_issues: 1, backlog_issues: 1 },
  B: { total_issues: 10, completed_issues: 8, cancelled_issues: 2, started_issues: 0, unstarted_issues: 0, backlog_issues: 0 },
};
const profile = (language) => ({
  "GET /api/users/me/profile/": { id: "p1", user: "u1", language, is_onboarded: true, is_tour_completed: true, theme: { theme: "light" } },
});
// the current cycle c1 with the counts, in the lists and the details, and as its progress (which the card merges in)
const cycleStubs = (counts, language) =>
  stubs({
    [`GET ${P}/cycles/`]: [{ ...cycle, ...counts }],
    [`GET ${W}/cycles/`]: [{ ...cycle, ...counts }],
    [`GET ${P}/cycles/c1/`]: { ...cycle, ...counts },
    [`GET ${P}/cycles/c1/progress/`]: counts,
    ...profile(language),
  });

// the card, its parts, and the keys of its rows' list (see the header)
const readCard = (bar) =>
  bar.evaluate((el) => {
    const card = el.closest('div[class*="min-h-[17rem]"]');
    const header = card.querySelector("h3")?.parentElement?.querySelector(":scope > span");
    const rows = [...card.querySelectorAll("div.cursor-pointer")].map((row) => ({
      group: row.querySelector("span.capitalize")?.textContent,
      count: Number.parseInt(row.querySelector(":scope > span:last-child")?.textContent ?? "", 10),
    }));
    const list = card.querySelector("div.cursor-pointer")?.closest("div.flex.flex-col.gap-5");
    const note = list ? [...list.children].find((c) => c.tagName === "SPAN")?.textContent.trim() : undefined;
    const fiberOf = (node) => node[Object.keys(node).find((k) => k.startsWith("__reactFiber$"))];
    const items = [];
    const container = list && fiberOf(list);
    const array = container?.child; // the Fragment fiber React makes for the array map() returns
    for (let f = array?.tag === 7 ? array.child : null; f; f = f.sibling)
      items.push({ key: f.key, type: f.tag === 7 ? "Fragment" : typeof f.type === "string" ? f.type : "component" });
    const indicator = el.querySelector("[style*='width']");
    return {
      header: header?.textContent.trim(),
      valueNow: el.getAttribute("aria-valuenow"),
      width: indicator ? (indicator.getBoundingClientRect().width / el.querySelector("[style*='width']").parentElement.getBoundingClientRect().width) * 100 : null,
      rows,
      note,
      list: array ? { tag: array.tag, key: array.key, items } : null,
    };
  });

const onCyclesPage = async (page, name, counts, language) => {
  const keyWarnings = [];
  page.on("console", (m) => {
    if (/unique "key"/.test(m.text())) keyWarnings.push(m.text().split("\n")[0]);
  });
  await goto(page, `/probe-ws/projects/${project.id}/cycles`);
  await page.waitForFunction((lang) => document.documentElement.lang === lang, language, { timeout: 20000 }).catch(() => {});
  // (the cycle's row above the cards has a "Cycle progress" bar too)
  const box = page.locator('div[class*="min-h-[17rem]"]').filter({ has: page.getByRole("progressbar", { name: "Cycle progress" }) });
  const bar = box.getByRole("progressbar", { name: "Cycle progress" });
  check(`${name}: the active-cycle card shows its progress bar`, await visible(bar, 20000));
  await page.waitForTimeout(1500);
  const card = await readCard(bar);
  console.log(`  (the card: ${JSON.stringify(card)})`);
  console.log(`  (console key warnings: ${keyWarnings.length}${keyWarnings.length ? ` ${JSON.stringify(keyWarnings)}` : ""})`);
  await box.screenshot({ path: path.join(out, `${name}-card.png`) });
  await page.screenshot({ path: path.join(out, `${name}-page.png`) });
  return card;
};

probe(async () => {
  await scenario("T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en)", cycleStubs(COUNTS.A, "en"), async (page) => {
    const card = await onCyclesPage(page, "T19-A-en", COUNTS.A, "en");
    check('T19 A (en) the header reads "3/8 work items completed"', card.header === "3/8 work items completed", JSON.stringify(card.header));
    check("T19 A (en) the progress bar's aria-valuenow is 38", card.valueNow === "38", `aria-valuenow ${card.valueNow}, fill ${card.width?.toFixed(1)}%`);
    check('T19 A (en) the note reads "2 cancelled work items are excluded from this report."', card.note === "2 cancelled work items are excluded from this report.", JSON.stringify(card.note));
    const want = [["completed", 3], ["started", 3], ["unstarted", 1], ["backlog", 1]];
    check("T19 A (en) the rows show completed 3, started 3, unstarted 1, backlog 1", JSON.stringify(card.rows.map((r) => [r.group, r.count])) === JSON.stringify(want), JSON.stringify(card.rows));
    const keys = card.list?.items.map((i) => i.key) ?? [];
    check(
      "T19 A (en) each item of the rows' list has a React key of its own (what the dev-mode key warning checks)",
      keys.length > 0 && keys.every((k) => k !== null) && new Set(keys).size === keys.length,
      JSON.stringify(card.list)
    );
  });

  await scenario("T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN)", cycleStubs(COUNTS.A, "zh-CN"), async (page) => {
    const card = await onCyclesPage(page, "T19-A-zh-CN", COUNTS.A, "zh-CN");
    check("T19 A (zh-CN) the UI language follows the profile", (await page.evaluate(() => document.documentElement.lang)) === "zh-CN");
    check('T19 A (zh-CN) the header reads "3/8 个工作项已完成"', card.header === "3/8 个工作项已完成", JSON.stringify(card.header));
    check('T19 A (zh-CN) the note reads "报告中已排除 2 个已取消的工作项。"', card.note === "报告中已排除 2 个已取消的工作项。", JSON.stringify(card.note));
    check("T19 A (zh-CN) the progress bar's aria-valuenow is 38", card.valueNow === "38", `aria-valuenow ${card.valueNow}`);
  });

  await scenario("T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en)", cycleStubs(COUNTS.B, "en"), async (page) => {
    const card = await onCyclesPage(page, "T19-B-en", COUNTS.B, "en");
    check('T19 B (en) the header reads "8/8 work items completed"', card.header === "8/8 work items completed", JSON.stringify(card.header));
    check("T19 B (en) the progress bar's aria-valuenow is 100", card.valueNow === "100", `aria-valuenow ${card.valueNow}, fill ${card.width?.toFixed(1)}%`);
  });
});
```

### 7.4 第 4 轮的输出

#### 7.4.1 本分支 `ff555bf`：摘要

```text
run 4, branch: build at ff555bf4cf945ea7bfcba613a0338803548b7b33 ($COTMP/probe-app)
a3-profile: exit 0, PASS 18, FAIL 0, last: 18 passed, 0 failed
b: PASS 203, FAIL 0, last: 203 passed, 0 failed
t14-covers: exit 0, PASS 32, FAIL 0, last: 32 passed, 0 failed
t15-pictures: exit 0, PASS 81, FAIL 0, last: 81 passed, 0 failed
t17-copy: exit 0, PASS 22, FAIL 0, last: 22 passed, 0 failed
t19-active-cycle: exit 0, PASS 23, FAIL 0, last: 23 passed, 0 failed

[exited with code 0]
```

#### 7.4.2 基线 `c493255`：摘要

```text
run 4, base: build at ref: refs/heads/worktree-m1-closeout = c493255ee8c7f418567bacbd8f025de6adbf18e2 ($COTMP/base)
a3-profile: exit 1, PASS 14, FAIL 4, last: 14 passed, 4 failed
b: PASS 157, FAIL 46, last: 157 passed, 46 failed
t14-covers: exit 1, PASS 30, FAIL 2, last: 30 passed, 2 failed
t15-pictures: exit 1, PASS 70, FAIL 11, last: 70 passed, 11 failed
t17-copy: exit 1, PASS 21, FAIL 1, last: 21 passed, 1 failed
t19-active-cycle: exit 1, PASS 16, FAIL 7, last: 16 passed, 7 failed

[exited with code 0]
```

#### 7.4.3 本分支：个人主页的标签

```text

== profile assigned at 480 px
  (the raw key "profile.tabs.assigned" is not on screen; in the DOM: 0)
PASS profile assigned at 480 px: the page does not show the raw key "profile.tabs.assigned" (it should say "Assigned")
PASS profile assigned at 480 px: the header's menu button says "Assigned"
PASS profile assigned at 480 px: no request outside the scenario's list
PASS profile assigned at 480 px: no uncaught page error
PASS profile assigned at 480 px: no "call navigate() in a React.useEffect()" warning

== profile assigned at 1440 px
  (the raw key "profile.tabs.assigned" is not on screen; in the DOM: 0)
PASS profile assigned at 1440 px: the page does not show the raw key "profile.tabs.assigned" (it should say "Assigned")
PASS profile assigned at 1440 px: no request outside the scenario's list
PASS profile assigned at 1440 px: no uncaught page error
PASS profile assigned at 1440 px: no "call navigate() in a React.useEffect()" warning

== profile created at 480 px
  (the raw key "profile.tabs.created" is not on screen; in the DOM: 0)
PASS profile created at 480 px: the page does not show the raw key "profile.tabs.created" (it should say "Created")
PASS profile created at 480 px: the header's menu button says "Created"
PASS profile created at 480 px: no request outside the scenario's list
PASS profile created at 480 px: no uncaught page error
PASS profile created at 480 px: no "call navigate() in a React.useEffect()" warning

== profile created at 1440 px
  (the raw key "profile.tabs.created" is not on screen; in the DOM: 0)
PASS profile created at 1440 px: the page does not show the raw key "profile.tabs.created" (it should say "Created")
PASS profile created at 1440 px: no request outside the scenario's list
PASS profile created at 1440 px: no uncaught page error
PASS profile created at 1440 px: no "call navigate() in a React.useEffect()" warning

18 passed, 0 failed
```

#### 7.4.4 本分支：B 组

```text
build at ff555bf4cf945ea7bfcba613a0338803548b7b33 ($COTMP/probe-app)
closeout-b: exit 0, PASS 203, FAIL 0, last: 203 passed, 0 failed

== B4 work item page
PASS B4 the description shows the member's link
PASS B4 the description shows the image
PASS B4 editor link: one window.open call, to https://example.com/
PASS B4 editor link: the third argument has noopener and noreferrer
PASS B4 editor link: the new tab opens (popup event)
PASS B4 editor link: in the new tab window.opener is null
PASS B4 editor link: in the new tab document.referrer is empty
PASS B4 the attachment list shows report.txt
PASS B4 attachment list: one window.open call, to /api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/
PASS B4 attachment list: the third argument has noopener and noreferrer
PASS B4 attachment list: the new tab opens (popup event)
PASS B4 attachment list: in the new tab window.opener is null
PASS B4 attachment list: in the new tab document.referrer is empty
PASS B4 the image toolbar shows its download button
PASS B4 image toolbar download: one window.open call, to /api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/
PASS B4 image toolbar download: the third argument has noopener and noreferrer
PASS B4 image toolbar download: the new tab opens (popup event)
PASS B4 image toolbar download: in the new tab window.opener is null
PASS B4 image toolbar download: in the new tab document.referrer is empty
PASS B4 the full-screen preview opens
PASS B4 full-screen download: one window.open call, to /api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/
PASS B4 full-screen download: the third argument has noopener and noreferrer
PASS B4 full-screen download: the new tab opens (popup event)
PASS B4 full-screen download: in the new tab window.opener is null
PASS B4 full-screen download: in the new tab document.referrer is empty
PASS B4 full-screen open original: one window.open call, to /api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/
PASS B4 full-screen open original: the third argument has noopener and noreferrer
PASS B4 full-screen open original: the new tab opens (popup event)
PASS B4 full-screen open original: in the new tab window.opener is null
PASS B4 full-screen open original: in the new tab document.referrer is empty
  (files served: ["GET /api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","GET /api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","GET /api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/"])
PASS B4 work item page: no request outside the scenario's list
PASS B4 work item page: no uncaught page error
PASS B4 work item page: no "call navigate() in a React.useEffect()" warning

== B4 open in new tab: project lists
PASS B4 cycle: /probe-ws/projects/p1/cycles lists "Sprint One"
PASS B4 cycle: its context menu has "Open in new tab"
PASS B4 cycle: one window.open call, to /probe-ws/projects/p1/cycles/c1
PASS B4 cycle: the third argument has noopener and noreferrer
PASS B4 cycle: the new tab opens (popup event)
PASS B4 cycle: in the new tab window.opener is null
PASS B4 cycle: in the new tab document.referrer is empty
PASS B4 module: /probe-ws/projects/p1/modules lists "Module One"
PASS B4 module: its context menu has "Open in new tab"
PASS B4 module: one window.open call, to /probe-ws/projects/p1/modules/m1
PASS B4 module: the third argument has noopener and noreferrer
PASS B4 module: the new tab opens (popup event)
PASS B4 module: in the new tab window.opener is null
PASS B4 module: in the new tab document.referrer is empty
PASS B4 project view: /probe-ws/projects/p1/views lists "Probe View"
PASS B4 project view: its context menu has "Open in new tab"
PASS B4 project view: one window.open call, to /probe-ws/projects/p1/views/v1
PASS B4 project view: the third argument has noopener and noreferrer
PASS B4 project view: the new tab opens (popup event)
PASS B4 project view: in the new tab window.opener is null
PASS B4 project view: in the new tab document.referrer is empty
PASS B4 work item: /probe-ws/projects/p1/issues lists "Project item alpha"
PASS B4 work item: its context menu has "Open in new tab"
PASS B4 work item: one window.open call, to /probe-ws/browse/PRB-1
PASS B4 work item: the third argument has noopener and noreferrer
PASS B4 work item: the new tab opens (popup event)
PASS B4 work item: in the new tab window.opener is null
PASS B4 work item: in the new tab document.referrer is empty
PASS B4 open in new tab: project lists: no request outside the scenario's list
PASS B4 open in new tab: project lists: no uncaught page error
PASS B4 open in new tab: project lists: no "call navigate() in a React.useEffect()" warning

== B4 open in new tab: project card
PASS B4 project card: /probe-ws/projects shows the card of "Other Project" (not joined)
PASS B4 project card: its context menu has "Open in new tab"
PASS B4 project card: one window.open call, to /probe-ws/projects/p2/issues
PASS B4 project card: the third argument has noopener and noreferrer
PASS B4 project card: the new tab opens (popup event)
PASS B4 project card: in the new tab window.opener is null
PASS B4 project card: in the new tab document.referrer is empty
PASS B4 open in new tab: project card: no request outside the scenario's list
PASS B4 open in new tab: project card: no uncaught page error
PASS B4 open in new tab: project card: no "call navigate() in a React.useEffect()" warning

== B4 open in new tab: workspace views
PASS B4 default workspace view: /probe-ws/workspace-views/assigned has the view's menu in its header
PASS B4 default workspace view: its menu has "Open in new tab"
PASS B4 default workspace view: one window.open call, to /probe-ws/workspace-views/assigned
PASS B4 default workspace view: the third argument has noopener and noreferrer
PASS B4 default workspace view: the new tab opens (popup event)
PASS B4 default workspace view: in the new tab window.opener is null
PASS B4 default workspace view: in the new tab document.referrer is empty
PASS B4 saved workspace view: /probe-ws/workspace-views/wv1 has the view's menu in its header
PASS B4 saved workspace view: its menu has "Open in new tab"
PASS B4 saved workspace view: one window.open call, to /probe-ws/workspace-views/wv1
PASS B4 saved workspace view: the third argument has noopener and noreferrer
PASS B4 saved workspace view: the new tab opens (popup event)
PASS B4 saved workspace view: in the new tab window.opener is null
PASS B4 saved workspace view: in the new tab document.referrer is empty
PASS B4 open in new tab: workspace views: no request outside the scenario's list
PASS B4 open in new tab: workspace views: no uncaught page error
PASS B4 open in new tab: workspace views: no "call navigate() in a React.useEffect()" warning

== B5 tab overflow menu
PASS B5 the project navigation is tabbed (the Work items tab is in the header)
PASS B5 the tabs overflow: an overflow menu button follows the tabs
PASS B5 the overflow menu opens and lists the tabs that do not fit
PASS B5 choosing Modules goes to /probe-ws/projects/p1/modules
PASS B5 the overflow menu closes
PASS B5 window.close() was not called (count 0)
PASS B5 tab overflow menu: no request outside the scenario's list
PASS B5 tab overflow menu: no uncaught page error
PASS B5 tab overflow menu: no "call navigate() in a React.useEffect()" warning

== B6 notification preview
PASS B6 the notifications list shows the comment notification
  (text nodes that mention Jerry: ["Tom & Jerry <3","Tom & Jerry <3"])
PASS B6 the comment and the description notification preview "<p>Tom &amp; Jerry &lt;3</p>" as "Tom & Jerry <3"
PASS B6 no preview shows an entity (&amp; or &lt;) as text
PASS B6 notification preview: no request outside the scenario's list
PASS B6 notification preview: no uncaught page error
PASS B6 notification preview: no "call navigate() in a React.useEffect()" warning

== B7 comment editor
PASS B7 the comment editor is there
  (empty: the editor's HTML "<p class=\"editor-paragraph-block\"></p>"; Comment button disabled; 0 POST)
PASS B7 comment "empty": the Comment button is disabled
PASS B7 comment "empty": neither a click nor Enter sends a request
  (only spaces: the editor's HTML "<p class=\"editor-paragraph-block\" data-id=\"45006288-d510-4f1b-b6ba-5e145a2b7f5a\">   </p>"; Comment button disabled; 0 POST)
PASS B7 comment "only spaces": the Comment button is disabled
PASS B7 comment "only spaces": neither a click nor Enter sends a request
  (only non-breaking spaces: the editor's HTML "<p class=\"editor-paragraph-block\" data-id=\"4b89899d-6aa4-416e-8716-2887f79dc102\">&nbsp;&nbsp;&nbsp;</p>"; Comment button disabled; 0 POST)
PASS B7 comment "only non-breaking spaces": the Comment button is disabled
PASS B7 comment "only non-breaking spaces": neither a click nor Enter sends a request
  (only line breaks: the editor's HTML "<p class=\"editor-paragraph-block\" data-id=\"72b4f083-939c-4825-9d03-9d031fec144b\"></p><p class=\"editor-paragraph-block\" data-id=\"734f80da-70e3-4d01-a4af-e8cb782f2166\"></p><p class=\"editor-paragraph-block\" data-id=\"2180a000-59de-42cc-b127-eada736788b3\"></p><p class=\"editor-paragraph-block\" data-id=\"5c0dd108-4f86-4a92-b6fc-d50f30286fbd\"></p>"; Comment button disabled; 0 POST)
PASS B7 comment "only line breaks": the Comment button is disabled
PASS B7 comment "only line breaks": neither a click nor Enter sends a request
  (only hard breaks (<br>): the editor's HTML "<p class=\"editor-paragraph-block\" data-id=\"1350bb39-2ff3-4871-b73b-2423b5c4987c\"><br><br><br></p>"; Comment button disabled; 0 POST)
PASS B7 comment "only hard breaks (<br>)": the Comment button is disabled
PASS B7 comment "only hard breaks (<br>)": neither a click nor Enter sends a request
  (text: the editor's HTML "<p class=\"editor-paragraph-block\" data-id=\"7b668c2c-cabe-4919-9625-ed66b7e9d45b\">Looks good</p>"; Comment button enabled; 1 POST)
PASS B7 comment "text": the Comment button is enabled
PASS B7 comment "text": a click sends one POST …/comments/ with the editor's HTML
  (only an image: the editor's HTML "<image-component data-id=\"5f1c2a4e-7777-4a2b-9c3d-000000000007\" src=\"5f1c2a4e-3333-4a2b-9c3d-000000000003\" id=\"5f1c2a4e-7777-4a2b-9c3d-000000000007\" width=\"256px\" height=\"192px\" aspectratio=\"1.3333333333333333\" alignment=\"left\" status=\"uploaded\"></image-component>"; Comment button enabled; 1 POST)
PASS B7 comment "only an image": the Comment button is enabled
PASS B7 comment "only an image": a click sends one POST …/comments/ with the editor's HTML
  (only a mention: the editor's HTML "<p class=\"editor-paragraph-block\" data-id=\"4eeddf15-7af5-4e6a-a45e-34de4c6e8397\"><mention-component id=\"6c942b5e-945e-4553-8cdc-8bed8dac20d5\" entity_identifier=\"u2\" entity_name=\"user_mention\"></mention-component> </p>"; Comment button enabled; 1 POST)
PASS B7 comment "only a mention": the Comment button is enabled
PASS B7 comment "only a mention": a click sends one POST …/comments/ with the editor's HTML
PASS B7 comment editor: no request outside the scenario's list
PASS B7 comment editor: no uncaught page error
PASS B7 comment editor: no "call navigate() in a React.useEffect()" warning

== B7 draft modal
  (draft "empty" (empty): Discard closes without asking; the description's HTML "<p class=\"editor-paragraph-block\"></p>")
  (draft "only spaces" (empty): Discard closes without asking; the description's HTML "<p class=\"editor-paragraph-block\" data-id=\"2a3f8877-b878-47ca-87e4-2502d85847ec\">   </p>")
  (draft "only non-breaking spaces" (empty): Discard closes without asking; the description's HTML "<p class=\"editor-paragraph-block\" data-id=\"7dacd66b-3ce9-44ba-b554-0c9b82a8dd49\">&nbsp;&nbsp;&nbsp;</p>")
  (draft "only line breaks" (empty): Discard closes without asking; the description's HTML "<p class=\"editor-paragraph-block\" data-id=\"00b7638b-38b0-4fb7-92a0-6ab51c0dc85f\"><br><br><br></p>")
  (draft "only hard breaks (<br>)" (empty): Discard closes without asking; the description's HTML "<p class=\"editor-paragraph-block\" data-id=\"34e4713f-9695-41a7-aef8-2ffef75161b8\"><br><br><br></p>")
  (draft "text" (content): Discard asks to save a draft; the description's HTML "<p class=\"editor-paragraph-block\" data-id=\"daae69ff-d050-4767-8ddf-f555ef350265\">Looks good</p>")
  (draft "only an image" (content): Discard asks to save a draft; the description's HTML "<image-component data-id=\"5f1c2a4e-7777-4a2b-9c3d-000000000007\" src=\"5f1c2a4e-3333-4a2b-9c3d-000000000003\" id=\"5f1c2a4e-7777-4a2b-9c3d-000000000007\" width=\"256px\" height=\"192px\" aspectratio=\"1.3333333333333333\" alignment=\"left\" status=\"uploaded\"></image-component>")
  (draft "only a mention" (content): Discard asks to save a draft; the description's HTML "<p class=\"editor-paragraph-block\" data-id=\"abe5b730-0e1d-47a7-a868-f66456f94432\"><mention-component id=\"0e881a77-d8ee-43ca-9286-8c314cf888a6\" entity_identifier=\"u2\" entity_name=\"user_mention\"></mention-component> </p>")
PASS B7 draft "text": Discard asks to save a draft
PASS B7 draft "only an image": Discard asks to save a draft
PASS B7 draft "only a mention": Discard asks to save a draft
PASS B7 draft: empty, spaces, non-breaking spaces, line breaks and hard breaks close without asking
PASS B7 draft modal: no request outside the scenario's list
PASS B7 draft modal: no uncaught page error
PASS B7 draft modal: no "call navigate() in a React.useEffect()" warning

== B8 callout from the description
PASS B8 the description shows the callout
PASS B8 the callout's icon is 💡 (data-emoji-unicode="128161")
PASS B8 (T7) the callout node's data-emoji-unicode is the string "128161"
PASS B8 callout from the description: no request outside the scenario's list
PASS B8 callout from the description: no uncaught page error
PASS B8 callout from the description: no "call navigate() in a React.useEffect()" warning

== B8 new callout takes the stored logo
PASS B8 the create work item modal opens
PASS B8 /callout inserts a callout
PASS B8 the new callout's icon is the stored emoji 😀 (128512)
  (node data-emoji-url ["https://cdn.example.com/emoji/1f600.png?size=64&set=apple"]; in the HTML output ["https://cdn.example.com/emoji/1f600.png?size=64&set=apple"])
  (the editor's HTML: <div data-id="03846a24-632d-4419-9bf9-6396eeae4724" id="03846a24-632d-4419-9bf9-6396eeae4724" data-emoji-unicode="128512" data-emoji-url="https://cdn.example.com/emoji/1f600.png?size=64&amp;set=apple" data-logo-in-use="emoji" data-background="" data-block-type="callout-component"><p class="editor-paragraph-block" data-id="d216c64e-09be-4115-afb6-7e160e018eaa"></p></div>)
  (the stored emoji URL is not in any attribute of the page)
PASS B8 the new callout node's data-emoji-url is https://cdn.example.com/emoji/1f600.png?size=64&set=apple (a plain &)
PASS B8 the editor's HTML output, parsed back, holds the same address (a plain &, no &amp;)
PASS B8 new callout takes the stored logo: no request outside the scenario's list
PASS B8 new callout takes the stored logo: no uncaught page error
PASS B8 new callout takes the stored logo: no "call navigate() in a React.useEffect()" warning

== B9 theme names (zh-CN)
PASS B9 (zh-CN) the UI language follows the profile (<html lang="zh-CN">)
PASS B9 (zh-CN) Power K opens
PASS B9 (zh-CN) Power K offers "更改界面主题"
  (Power K theme menu: ["系统偏好","浅色","深色","浅色高对比度","深色高对比度"])
PASS B9 (zh-CN) Power K's theme menu names the themes ["系统偏好","浅色","深色","浅色高对比度","深色高对比度"]
PASS B9 (zh-CN) the settings' theme switch shows the current theme as "系统偏好"
  (settings theme switch: current "系统偏好"; options ["系统偏好","浅色","深色","浅色高对比度","深色高对比度"])
PASS B9 (zh-CN) the switch's options name the themes ["系统偏好","浅色","深色","浅色高对比度","深色高对比度"]
PASS B9 theme names (zh-CN): no request outside the scenario's list
PASS B9 theme names (zh-CN): no uncaught page error
PASS B9 theme names (zh-CN): no "call navigate() in a React.useEffect()" warning

== B9 theme names (en)
PASS B9 (en) the UI language follows the profile (<html lang="en">)
PASS B9 (en) Power K opens
PASS B9 (en) Power K offers "Change interface theme"
  (Power K theme menu: ["System Preference","Light","Dark","Light high contrast","Dark high contrast"])
PASS B9 (en) Power K's theme menu names the themes ["System Preference","Light","Dark","Light high contrast","Dark high contrast"]
PASS B9 (en) the settings' theme switch shows the current theme as "System Preference"
  (settings theme switch: current "System Preference"; options ["System Preference","Light","Dark","Light high contrast","Dark high contrast"])
PASS B9 (en) the switch's options name the themes ["System Preference","Light","Dark","Light high contrast","Dark high contrast"]
PASS B9 theme names (en): no request outside the scenario's list
PASS B9 theme names (en): no uncaught page error
PASS B9 theme names (en): no "call navigate() in a React.useEffect()" warning

== B10 work item form cycles, from the workspace home
PASS B10 the workspace home is up
PASS B10 the workspace home itself does not ask for the project's cycles
PASS B10 the create work item modal opens, on project p1
PASS B10 the form fetches the project's cycles (GET /api/workspaces/probe-ws/projects/p1/cycles/) before the cycle dropdown opens
PASS B10 the cycle dropdown lists the project's cycle ("Sprint One")
PASS B10 work item form cycles, from the workspace home: no request outside the scenario's list
PASS B10 work item form cycles, from the workspace home: no uncaught page error
PASS B10 work item form cycles, from the workspace home: no "call navigate() in a React.useEffect()" warning

== B10 work item form cycles, inside the project
PASS B10 (in project) the create work item modal opens
PASS B10 (in project) the cycle dropdown lists the project's cycle ("Sprint One")
PASS B10 (in project) GET /api/workspaces/probe-ws/projects/p1/cycles/ was requested
PASS B10 work item form cycles, inside the project: no request outside the scenario's list
PASS B10 work item form cycles, inside the project: no uncaught page error
PASS B10 work item form cycles, inside the project: no "call navigate() in a React.useEffect()" warning

== B11 project list empty state
PASS B11 project list: /probe-ws/projects shows the empty state "No active projects"
PASS B11 project list: "No active projects" shows its illustration (an <svg> of at least 40x40 px with shapes)
  (illustration next to "No active projects": [{"width":160,"height":160,"shapes":31}])
PASS B11 project list: (probe control) with its 1 illustration(s) taken out, the check finds none
PASS B11 project list: every <img> on the page loaded (naturalWidth > 0; 0 on the page)
PASS B11 project list empty state: no request outside the scenario's list
PASS B11 project list empty state: no uncaught page error
PASS B11 project list empty state: no "call navigate() in a React.useEffect()" warning

== B11 cycle list empty state
PASS B11 cycle list: /probe-ws/projects/p1/cycles shows the empty state "Group and timebox your work in Cycles."
PASS B11 cycle list: "Group and timebox your work in Cycles." shows its illustration (an <svg> of at least 40x40 px with shapes)
  (illustration next to "Group and timebox your work in Cycles.": [{"width":160,"height":165,"shapes":32}])
PASS B11 cycle list: (probe control) with its 1 illustration(s) taken out, the check finds none
PASS B11 cycle list: every <img> on the page loaded (naturalWidth > 0; 0 on the page)
PASS B11 cycle list empty state: no request outside the scenario's list
PASS B11 cycle list empty state: no uncaught page error
PASS B11 cycle list empty state: no "call navigate() in a React.useEffect()" warning

== B11 intake empty state
PASS B11 intake: /probe-ws/projects/p1/intake shows the empty state "No matching results."
PASS B11 intake: "No matching results." shows its illustration (an <svg> of at least 40x40 px with shapes)
  (illustration next to "No matching results.": [{"width":80,"height":80,"shapes":7}])
PASS B11 intake: (probe control) with its 1 illustration(s) taken out, the check finds none
PASS B11 intake: /probe-ws/projects/p1/intake shows the empty state "Select an Intake work item to view its details"
PASS B11 intake: "Select an Intake work item to view its details" shows its illustration (an <svg> of at least 40x40 px with shapes)
  (illustration next to "Select an Intake work item to view its details": [{"width":80,"height":80,"shapes":10}])
PASS B11 intake: (probe control) with its 1 illustration(s) taken out, the check finds none
PASS B11 intake: every <img> on the page loaded (naturalWidth > 0; 0 on the page)
PASS B11 intake empty state: no request outside the scenario's list
PASS B11 intake empty state: no uncaught page error
PASS B11 intake empty state: no "call navigate() in a React.useEffect()" warning

203 passed, 0 failed
```

#### 7.4.5 本分支：第 12 项

```text
PASS the build has 29 WebP presets and no cover JPEG

== A1 project cards
  (at /probe-ws/projects)
PASS A1 the cards show the eight presets
PASS A1: no image failed to load
PASS A1 project cards: no request outside the scenario's list
PASS A1 project cards: no uncaught page error
PASS A1 project cards: no "call navigate() in a React.useEffect()" warning

== A2 create project and the picker
  (at /probe-ws/projects)
PASS A2 the modal's header shows a preset
PASS A2 the header's preset is one of the 29
PASS A2 the picker offers 29 presets
PASS A2 the picker's presets are the built WebP files, in order
PASS A2 picker: no image failed to load
PASS A2 the header shows the picked preset 17
  upload requests: [{"entity_identifier":"","entity_type":"PROJECT_COVER","name":"image_17-CQ3Mr1HZ.webp","size":12900,"type":"image/webp"}]
PASS A2 the preset is uploaded as a WebP copy under its file name
  writes: [
 "POST /api/probe-object-store/ ------WebKitFormBoundarynPBrEfJomREVDdZi Content-Disposition: form-data; name=\"key\" k ------WebKitFormBoundarynPBrEfJomREVDdZi Con",
 "PATCH /api/assets/v2/workspaces/probe-ws/asset-1/ {}",
 "POST /api/workspaces/probe-ws/projects/ {\"cover_image_url\":\"/assets/image_17-CQ3Mr1HZ.webp\",\"description\":\"\",\"logo_props\":{\"in_use\":\"emoji\",\"emoji\":{\"value\":\"89",
 "POST /api/assets/v2/workspaces/probe-ws/projects/pnew/pnew/bulk/ {\"asset_ids\":[\"asset-1\"]}",
 "PATCH /api/workspaces/probe-ws/projects/pnew/ {\"cover_image_url\":\"/api/assets/v2/workspaces/probe-ws/asset-1/\"}"
]
PASS A2 the project stores the uploaded copy's address, not the preset's
PASS A2 create project and the picker: no request outside the scenario's list
PASS A2 create project and the picker: no uncaught page error
PASS A2 create project and the picker: no "call navigate() in a React.useEffect()" warning

== A3 project settings
PASS A3 the settings show preset 7
PASS A3: no image failed to load
PASS A3 project settings: no request outside the scenario's list
PASS A3 project settings: no uncaught page error
PASS A3 project settings: no "call navigate() in a React.useEffect()" warning

== A4 profile settings and A5 user menu
PASS A4 the profile shows preset 22
PASS A4: no image failed to load
PASS A4 profile settings and A5 user menu: no request outside the scenario's list
PASS A4 profile settings and A5 user menu: no uncaught page error
PASS A4 profile settings and A5 user menu: no "call navigate() in a React.useEffect()" warning

== A5 user menu, no cover
  (at /probe-ws/projects)
PASS A5 the user menu falls back to the default preset (1)
PASS A5: no image failed to load
PASS A5 user menu, no cover: no request outside the scenario's list
PASS A5 user menu, no cover: no uncaught page error
PASS A5 user menu, no cover: no "call navigate() in a React.useEffect()" warning

32 passed, 0 failed
```

#### 7.4.6 本分支：第 13 项

```text

== D1 cycles disabled (light)
PASS D1 cycles disabled (light): shows cycles-light, served as the repository has it
PASS D1 cycles disabled (light): cycles-light differs from c493255's
PASS D1 cycles disabled (light): no image failed to load
PASS D1 cycles disabled (light): no request outside the scenario's list
PASS D1 cycles disabled (light): no uncaught page error
PASS D1 cycles disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 modules disabled (light)
PASS D1 modules disabled (light): shows modules-light, served as the repository has it
PASS D1 modules disabled (light): modules-light differs from c493255's
PASS D1 modules disabled (light): no image failed to load
PASS D1 modules disabled (light): no request outside the scenario's list
PASS D1 modules disabled (light): no uncaught page error
PASS D1 modules disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 views disabled (light)
PASS D1 views disabled (light): shows views-light, served as the repository has it
PASS D1 views disabled (light): views-light differs from c493255's
PASS D1 views disabled (light): no image failed to load
PASS D1 views disabled (light): no request outside the scenario's list
PASS D1 views disabled (light): no uncaught page error
PASS D1 views disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 intake disabled (light)
PASS D1 intake disabled (light): shows intake-light, served as the repository has it
PASS D1 intake disabled (light): intake-light differs from c493255's
PASS D1 intake disabled (light): no image failed to load
PASS D1 intake disabled (light): no request outside the scenario's list
PASS D1 intake disabled (light): no uncaught page error
PASS D1 intake disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 cycles disabled (dark)
PASS D1 cycles disabled (dark): shows cycles-dark, served as the repository has it
PASS D1 cycles disabled (dark): cycles-dark differs from c493255's
PASS D1 cycles disabled (dark): no image failed to load
PASS D1 cycles disabled (dark): no request outside the scenario's list
PASS D1 cycles disabled (dark): no uncaught page error
PASS D1 cycles disabled (dark): no "call navigate() in a React.useEffect()" warning

== D1 modules disabled (dark)
PASS D1 modules disabled (dark): shows modules-dark, served as the repository has it
PASS D1 modules disabled (dark): modules-dark differs from c493255's
PASS D1 modules disabled (dark): no image failed to load
PASS D1 modules disabled (dark): no request outside the scenario's list
PASS D1 modules disabled (dark): no uncaught page error
PASS D1 modules disabled (dark): no "call navigate() in a React.useEffect()" warning

== D1 views disabled (dark)
PASS D1 views disabled (dark): shows views-dark, served as the repository has it
PASS D1 views disabled (dark): views-dark differs from c493255's
PASS D1 views disabled (dark): no image failed to load
PASS D1 views disabled (dark): no request outside the scenario's list
PASS D1 views disabled (dark): no uncaught page error
PASS D1 views disabled (dark): no "call navigate() in a React.useEffect()" warning

== D1 intake disabled (dark)
PASS D1 intake disabled (dark): shows intake-dark, served as the repository has it
PASS D1 intake disabled (dark): intake-dark differs from c493255's
PASS D1 intake disabled (dark): no image failed to load
PASS D1 intake disabled (dark): no request outside the scenario's list
PASS D1 intake disabled (dark): no uncaught page error
PASS D1 intake disabled (dark): no "call navigate() in a React.useEffect()" warning

== D2 product tour
PASS D2 the tour's welcome shows
PASS D2 step 1 "Plan with work items" shows
PASS D2 step 1: shows issues, served as the repository has it
PASS D2 step 1: issues differs from c493255's
PASS D2 step 1: no image failed to load
PASS D2 step 2 "Move with cycles" shows
PASS D2 step 2: shows cycles, served as the repository has it
PASS D2 step 2: cycles differs from c493255's
PASS D2 step 2: no image failed to load
PASS D2 step 3 "Break into modules" shows
PASS D2 step 3: shows modules, served as the repository has it
PASS D2 step 3: no image failed to load
PASS D2 step 4 "Views" shows
PASS D2 step 4: shows views, served as the repository has it
PASS D2 step 4: views differs from c493255's
PASS D2 step 4: no image failed to load
PASS D2 product tour: no request outside the scenario's list
PASS D2 product tour: no uncaught page error
PASS D2 product tour: no "call navigate() in a React.useEffect()" warning

== D3 no active cycle (light)
PASS D3 (light) the active-cycle section shows
PASS D3 (light) its empty state shows
PASS D3 (light) no active-cycle picture on the page
PASS D3 no active cycle (light): no image failed to load
PASS D3 no active cycle (light): no request outside the scenario's list
PASS D3 no active cycle (light): no uncaught page error
PASS D3 no active cycle (light): no "call navigate() in a React.useEffect()" warning

== D3 no active cycle (dark)
PASS D3 (dark) the active-cycle section shows
PASS D3 (dark) its empty state shows
PASS D3 (dark) no active-cycle picture on the page
PASS D3 no active cycle (dark): no image failed to load
PASS D3 no active cycle (dark): no request outside the scenario's list
PASS D3 no active cycle (dark): no uncaught page error
PASS D3 no active cycle (dark): no "call navigate() in a React.useEffect()" warning

81 passed, 0 failed
```

#### 7.4.7 本分支：第 14 项

```text

== C1 copy and paste between editors
PASS C1 the description editor shows
PASS C1 the comment editor shows
PASS C1 the comment editor has the description's text
PASS C1 the app asked once to duplicate the uploaded image
PASS C1 the pasted image points to the new asset
  requests: ["POST /api/assets/v2/workspaces/probe-ws/duplicate-assets/a0000000-0000-4000-8000-000000000001/"]
  description saves: ["{\"description_html\":\"<p class=\\\"editor-paragraph-block\\\" data-id=\\\"5f1c2a4e-1111-4a2b-9c3d-000000000001\\\">Alpha description text</p><image-component data-id=\\\"5f1c2a4e-1111-4a2b-9c3d-000000000002\\\" src=\\\"a0000000-0000-4000-8000-000000000001\\\" id=\\\"5f1c2a4e-1111-4a2b-9c3d-000000000002\\\" width=\\\"200px"]
PASS C1 copy and paste between editors: no request outside the scenario's list
PASS C1 copy and paste between editors: no uncaught page error
PASS C1 copy and paste between editors: no "call navigate() in a React.useEffect()" warning

== C2 the same copy pasted as text/html only
PASS C2 the description editor shows
  copied types: ["text/plain","text/html","text/nerve-editor-html"]
PASS C2 the comment editor shows
  duplication requests: 0
PASS C2 without the custom type no duplication is asked for
PASS C2 the pasted image keeps the description's asset
  requests: []
PASS C2 the same copy pasted as text/html only: no request outside the scenario's list
PASS C2 the same copy pasted as text/html only: no uncaught page error
PASS C2 the same copy pasted as text/html only: no "call navigate() in a React.useEffect()" warning

== C3 a cross-origin clipboard payload does not run a script
PASS C3 the comment editor shows
  window.__xss after the paste: false
PASS C3 the clipboard payload's onerror did not run
PASS C3 the text was still pasted
PASS C3 no image with an event handler is in the editor
PASS C3 a cross-origin clipboard payload does not run a script: no request outside the scenario's list
PASS C3 a cross-origin clipboard payload does not run a script: no uncaught page error
PASS C3 a cross-origin clipboard payload does not run a script: no "call navigate() in a React.useEffect()" warning

22 passed, 0 failed
```

#### 7.4.8 本分支：T19 的卡片

```text

== T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en)
PASS T19-A-en: the active-cycle card shows its progress bar
  (the card: {"header":"3/8 work items completed","valueNow":"38","width":37.99713612637997,"rows":[{"group":"completed","count":3},{"group":"started","count":3},{"group":"unstarted","count":1},{"group":"backlog","count":1}],"note":"2 cancelled work items are excluded from this report.","list":{"tag":7,"key":null,"items":[{"key":"completed","type":"div"},{"key":"started","type":"div"},{"key":"unstarted","type":"div"},{"key":"backlog","type":"div"}]}})
  (console key warnings: 0)
PASS T19 A (en) the header reads "3/8 work items completed"
PASS T19 A (en) the progress bar's aria-valuenow is 38
PASS T19 A (en) the note reads "2 cancelled work items are excluded from this report."
PASS T19 A (en) the rows show completed 3, started 3, unstarted 1, backlog 1
PASS T19 A (en) each item of the rows' list has a React key of its own (what the dev-mode key warning checks)
PASS T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en): no request outside the scenario's list
PASS T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en): no uncaught page error
PASS T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en): no "call navigate() in a React.useEffect()" warning

== T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN)
PASS T19-A-zh-CN: the active-cycle card shows its progress bar
  (the card: {"header":"3/8 个工作项已完成","valueNow":"38","width":37.99713612637997,"rows":[{"group":"completed","count":3},{"group":"started","count":3},{"group":"unstarted","count":1},{"group":"backlog","count":1}],"note":"报告中已排除 2 个已取消的工作项。","list":{"tag":7,"key":null,"items":[{"key":"completed","type":"div"},{"key":"started","type":"div"},{"key":"unstarted","type":"div"},{"key":"backlog","type":"div"}]}})
  (console key warnings: 0)
PASS T19 A (zh-CN) the UI language follows the profile
PASS T19 A (zh-CN) the header reads "3/8 个工作项已完成"
PASS T19 A (zh-CN) the note reads "报告中已排除 2 个已取消的工作项。"
PASS T19 A (zh-CN) the progress bar's aria-valuenow is 38
PASS T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN): no request outside the scenario's list
PASS T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN): no uncaught page error
PASS T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN): no "call navigate() in a React.useEffect()" warning

== T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en)
PASS T19-B-en: the active-cycle card shows its progress bar
  (the card: {"header":"8/8 work items completed","valueNow":"100","width":100,"rows":[{"group":"completed","count":8}],"note":"2 cancelled work items are excluded from this report.","list":{"tag":7,"key":null,"items":[{"key":"completed","type":"div"}]}})
  (console key warnings: 0)
PASS T19 B (en) the header reads "8/8 work items completed"
PASS T19 B (en) the progress bar's aria-valuenow is 100
PASS T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en): no request outside the scenario's list
PASS T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en): no uncaught page error
PASS T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en): no "call navigate() in a React.useEffect()" warning

23 passed, 0 failed
```

#### 7.4.9 基线：个人主页的标签

```text

== profile assigned at 480 px
  (the raw key "profile.tabs.assigned" is on screen; in the DOM: 1)
FAIL profile assigned at 480 px: the page does not show the raw key "profile.tabs.assigned" (it should say "Assigned"): "profile.tabs.assigned" is on screen
FAIL profile assigned at 480 px: the header's menu button says "Assigned": no visible button says "Assigned"
PASS profile assigned at 480 px: no request outside the scenario's list
PASS profile assigned at 480 px: no uncaught page error
PASS profile assigned at 480 px: no "call navigate() in a React.useEffect()" warning

== profile assigned at 1440 px
  (the raw key "profile.tabs.assigned" is not on screen; in the DOM: 1)
PASS profile assigned at 1440 px: the page does not show the raw key "profile.tabs.assigned" (it should say "Assigned")
PASS profile assigned at 1440 px: no request outside the scenario's list
PASS profile assigned at 1440 px: no uncaught page error
PASS profile assigned at 1440 px: no "call navigate() in a React.useEffect()" warning

== profile created at 480 px
  (the raw key "profile.tabs.created" is on screen; in the DOM: 1)
FAIL profile created at 480 px: the page does not show the raw key "profile.tabs.created" (it should say "Created"): "profile.tabs.created" is on screen
FAIL profile created at 480 px: the header's menu button says "Created": no visible button says "Created"
PASS profile created at 480 px: no request outside the scenario's list
PASS profile created at 480 px: no uncaught page error
PASS profile created at 480 px: no "call navigate() in a React.useEffect()" warning

== profile created at 1440 px
  (the raw key "profile.tabs.created" is not on screen; in the DOM: 1)
PASS profile created at 1440 px: the page does not show the raw key "profile.tabs.created" (it should say "Created")
PASS profile created at 1440 px: no request outside the scenario's list
PASS profile created at 1440 px: no uncaught page error
PASS profile created at 1440 px: no "call navigate() in a React.useEffect()" warning

14 passed, 4 failed
```

#### 7.4.10 基线：第 12 项

```text
FAIL the build has 29 WebP presets and no cover JPEG: ["/assets/image_1-8ncaj2f5.jpg","/assets/image_2-Cbu4qct1.jpg"]

== A1 project cards
  (at /probe-ws/projects)
PASS A1 the cards show the eight presets
PASS A1: no image failed to load
PASS A1 project cards: no request outside the scenario's list
PASS A1 project cards: no uncaught page error
PASS A1 project cards: no "call navigate() in a React.useEffect()" warning

== A2 create project and the picker
  (at /probe-ws/projects)
PASS A2 the modal's header shows a preset
PASS A2 the header's preset is one of the 29
PASS A2 the picker offers 29 presets
PASS A2 the picker's presets are the built WebP files, in order
PASS A2 picker: no image failed to load
PASS A2 the header shows the picked preset 17
  upload requests: [{"entity_identifier":"","entity_type":"PROJECT_COVER","name":"image_17-fngcXhhl.jpg","size":57385,"type":"image/jpeg"}]
FAIL A2 the preset is uploaded as a WebP copy under its file name: [{"entity_identifier":"","entity_type":"PROJECT_COVER","name":"image_17-fngcXhhl.jpg","size":57385,"type":"image/jpeg"}]
  writes: [
 "POST /api/probe-object-store/ ------WebKitFormBoundaryt76pafDwofF0rAnf Content-Disposition: form-data; name=\"key\" k ------WebKitFormBoundaryt76pafDwofF0rAnf Con",
 "PATCH /api/assets/v2/workspaces/probe-ws/asset-1/ {}",
 "POST /api/workspaces/probe-ws/projects/ {\"cover_image_url\":\"/assets/image_17-fngcXhhl.jpg\",\"description\":\"\",\"logo_props\":{\"in_use\":\"emoji\",\"emoji\":{\"value\":\"898",
 "POST /api/assets/v2/workspaces/probe-ws/projects/pnew/pnew/bulk/ {\"asset_ids\":[\"asset-1\"]}",
 "PATCH /api/workspaces/probe-ws/projects/pnew/ {\"cover_image_url\":\"/api/assets/v2/workspaces/probe-ws/asset-1/\"}"
]
PASS A2 the project stores the uploaded copy's address, not the preset's
PASS A2 create project and the picker: no request outside the scenario's list
PASS A2 create project and the picker: no uncaught page error
PASS A2 create project and the picker: no "call navigate() in a React.useEffect()" warning

== A3 project settings
PASS A3 the settings show preset 7
PASS A3: no image failed to load
PASS A3 project settings: no request outside the scenario's list
PASS A3 project settings: no uncaught page error
PASS A3 project settings: no "call navigate() in a React.useEffect()" warning

== A4 profile settings and A5 user menu
PASS A4 the profile shows preset 22
PASS A4: no image failed to load
PASS A4 profile settings and A5 user menu: no request outside the scenario's list
PASS A4 profile settings and A5 user menu: no uncaught page error
PASS A4 profile settings and A5 user menu: no "call navigate() in a React.useEffect()" warning

== A5 user menu, no cover
  (at /probe-ws/projects)
PASS A5 the user menu falls back to the default preset (1)
PASS A5: no image failed to load
PASS A5 user menu, no cover: no request outside the scenario's list
PASS A5 user menu, no cover: no uncaught page error
PASS A5 user menu, no cover: no "call navigate() in a React.useEffect()" warning

30 passed, 2 failed
```

#### 7.4.11 基线：第 13 项

```text

== D1 cycles disabled (light)
PASS D1 cycles disabled (light): shows cycles-light, served as the repository has it
FAIL D1 cycles disabled (light): cycles-light differs from c493255's: ["89774bbf8f315a23"]
PASS D1 cycles disabled (light): no image failed to load
PASS D1 cycles disabled (light): no request outside the scenario's list
PASS D1 cycles disabled (light): no uncaught page error
PASS D1 cycles disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 modules disabled (light)
PASS D1 modules disabled (light): shows modules-light, served as the repository has it
FAIL D1 modules disabled (light): modules-light differs from c493255's: ["df27316c122312ee"]
PASS D1 modules disabled (light): no image failed to load
PASS D1 modules disabled (light): no request outside the scenario's list
PASS D1 modules disabled (light): no uncaught page error
PASS D1 modules disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 views disabled (light)
PASS D1 views disabled (light): shows views-light, served as the repository has it
FAIL D1 views disabled (light): views-light differs from c493255's: ["2630a44feb7289a5"]
PASS D1 views disabled (light): no image failed to load
PASS D1 views disabled (light): no request outside the scenario's list
PASS D1 views disabled (light): no uncaught page error
PASS D1 views disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 intake disabled (light)
PASS D1 intake disabled (light): shows intake-light, served as the repository has it
FAIL D1 intake disabled (light): intake-light differs from c493255's: ["60ffa8fa546f7f5f"]
PASS D1 intake disabled (light): no image failed to load
PASS D1 intake disabled (light): no request outside the scenario's list
PASS D1 intake disabled (light): no uncaught page error
PASS D1 intake disabled (light): no "call navigate() in a React.useEffect()" warning

== D1 cycles disabled (dark)
PASS D1 cycles disabled (dark): shows cycles-dark, served as the repository has it
FAIL D1 cycles disabled (dark): cycles-dark differs from c493255's: ["8cae8b1e5f17ae88"]
PASS D1 cycles disabled (dark): no image failed to load
PASS D1 cycles disabled (dark): no request outside the scenario's list
PASS D1 cycles disabled (dark): no uncaught page error
PASS D1 cycles disabled (dark): no "call navigate() in a React.useEffect()" warning

== D1 modules disabled (dark)
PASS D1 modules disabled (dark): shows modules-dark, served as the repository has it
FAIL D1 modules disabled (dark): modules-dark differs from c493255's: ["d600f02f12815a22"]
PASS D1 modules disabled (dark): no image failed to load
PASS D1 modules disabled (dark): no request outside the scenario's list
PASS D1 modules disabled (dark): no uncaught page error
PASS D1 modules disabled (dark): no "call navigate() in a React.useEffect()" warning

== D1 views disabled (dark)
PASS D1 views disabled (dark): shows views-dark, served as the repository has it
FAIL D1 views disabled (dark): views-dark differs from c493255's: ["7f0710ab95e9fcfc"]
PASS D1 views disabled (dark): no image failed to load
PASS D1 views disabled (dark): no request outside the scenario's list
PASS D1 views disabled (dark): no uncaught page error
PASS D1 views disabled (dark): no "call navigate() in a React.useEffect()" warning

== D1 intake disabled (dark)
PASS D1 intake disabled (dark): shows intake-dark, served as the repository has it
FAIL D1 intake disabled (dark): intake-dark differs from c493255's: ["e572356508c3206e"]
PASS D1 intake disabled (dark): no image failed to load
PASS D1 intake disabled (dark): no request outside the scenario's list
PASS D1 intake disabled (dark): no uncaught page error
PASS D1 intake disabled (dark): no "call navigate() in a React.useEffect()" warning

== D2 product tour
PASS D2 the tour's welcome shows
PASS D2 step 1 "Plan with work items" shows
PASS D2 step 1: shows issues, served as the repository has it
FAIL D2 step 1: issues differs from c493255's: ["fe3248a02087c363"]
PASS D2 step 1: no image failed to load
PASS D2 step 2 "Move with cycles" shows
PASS D2 step 2: shows cycles, served as the repository has it
FAIL D2 step 2: cycles differs from c493255's: ["d026c49c5733d076"]
PASS D2 step 2: no image failed to load
PASS D2 step 3 "Break into modules" shows
PASS D2 step 3: shows modules, served as the repository has it
PASS D2 step 3: no image failed to load
PASS D2 step 4 "Views" shows
PASS D2 step 4: shows views, served as the repository has it
FAIL D2 step 4: views differs from c493255's: ["6ee12edc68cc4589"]
PASS D2 step 4: no image failed to load
PASS D2 product tour: no request outside the scenario's list
PASS D2 product tour: no uncaught page error
PASS D2 product tour: no "call navigate() in a React.useEffect()" warning

== D3 no active cycle (light)
PASS D3 (light) the active-cycle section shows
PASS D3 (light) its empty state shows
PASS D3 (light) no active-cycle picture on the page
PASS D3 no active cycle (light): no image failed to load
PASS D3 no active cycle (light): no request outside the scenario's list
PASS D3 no active cycle (light): no uncaught page error
PASS D3 no active cycle (light): no "call navigate() in a React.useEffect()" warning

== D3 no active cycle (dark)
PASS D3 (dark) the active-cycle section shows
PASS D3 (dark) its empty state shows
PASS D3 (dark) no active-cycle picture on the page
PASS D3 no active cycle (dark): no image failed to load
PASS D3 no active cycle (dark): no request outside the scenario's list
PASS D3 no active cycle (dark): no uncaught page error
PASS D3 no active cycle (dark): no "call navigate() in a React.useEffect()" warning

70 passed, 11 failed
```

#### 7.4.12 基线：第 14 项

```text

== C1 copy and paste between editors
PASS C1 the description editor shows
PASS C1 the comment editor shows
PASS C1 the comment editor has the description's text
PASS C1 the app asked once to duplicate the uploaded image
PASS C1 the pasted image points to the new asset
  requests: ["POST /api/assets/v2/workspaces/probe-ws/duplicate-assets/a0000000-0000-4000-8000-000000000001/"]
  description saves: ["{\"description_html\":\"<p class=\\\"editor-paragraph-block\\\" data-id=\\\"5f1c2a4e-1111-4a2b-9c3d-000000000001\\\">Alpha description text</p><image-component data-id=\\\"5f1c2a4e-1111-4a2b-9c3d-000000000002\\\" src=\\\"a0000000-0000-4000-8000-000000000001\\\" id=\\\"5f1c2a4e-1111-4a2b-9c3d-000000000002\\\" width=\\\"200px"]
PASS C1 copy and paste between editors: no request outside the scenario's list
PASS C1 copy and paste between editors: no uncaught page error
PASS C1 copy and paste between editors: no "call navigate() in a React.useEffect()" warning

== C2 the same copy pasted as text/html only
PASS C2 the description editor shows
  copied types: ["text/plain","text/html","text/nerve-editor-html"]
PASS C2 the comment editor shows
  duplication requests: 0
PASS C2 without the custom type no duplication is asked for
PASS C2 the pasted image keeps the description's asset
  requests: []
PASS C2 the same copy pasted as text/html only: no request outside the scenario's list
PASS C2 the same copy pasted as text/html only: no uncaught page error
PASS C2 the same copy pasted as text/html only: no "call navigate() in a React.useEffect()" warning

== C3 a cross-origin clipboard payload does not run a script
PASS C3 the comment editor shows
  window.__xss after the paste: true
FAIL C3 the clipboard payload's onerror did not run: window.__xss=true
PASS C3 the text was still pasted
PASS C3 no image with an event handler is in the editor
PASS C3 a cross-origin clipboard payload does not run a script: no request outside the scenario's list
PASS C3 a cross-origin clipboard payload does not run a script: no uncaught page error
PASS C3 a cross-origin clipboard payload does not run a script: no "call navigate() in a React.useEffect()" warning

21 passed, 1 failed
```

#### 7.4.13 基线：T19 的卡片

```text

== T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en)
PASS T19-A-en: the active-cycle card shows its progress bar
  (the card: {"header":"5/8 Work items closed","valueNow":"62.5","width":62.49711303062497,"rows":[{"group":"completed","count":3},{"group":"started","count":3},{"group":"unstarted","count":1},{"group":"backlog","count":1}],"note":"2 cancelled work items are excluded from this report.","list":{"tag":7,"key":null,"items":[{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"}]}})
  (console key warnings: 0)
FAIL T19 A (en) the header reads "3/8 work items completed": "5/8 Work items closed"
FAIL T19 A (en) the progress bar's aria-valuenow is 38: aria-valuenow 62.5, fill 62.5%
PASS T19 A (en) the note reads "2 cancelled work items are excluded from this report."
PASS T19 A (en) the rows show completed 3, started 3, unstarted 1, backlog 1
FAIL T19 A (en) each item of the rows' list has a React key of its own (what the dev-mode key warning checks): {"tag":7,"key":null,"items":[{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"}]}
PASS T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en): no request outside the scenario's list
PASS T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en): no uncaught page error
PASS T19-A-en active cycle: 3 completed, 2 cancelled of 10 (en): no "call navigate() in a React.useEffect()" warning

== T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN)
PASS T19-A-zh-CN: the active-cycle card shows its progress bar
  (the card: {"header":"5/8 Work items closed","valueNow":"62.5","width":62.49711303062497,"rows":[{"group":"completed","count":3},{"group":"started","count":3},{"group":"unstarted","count":1},{"group":"backlog","count":1}],"note":"2 cancelled work items are excluded from this report.","list":{"tag":7,"key":null,"items":[{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"}]}})
  (console key warnings: 0)
PASS T19 A (zh-CN) the UI language follows the profile
FAIL T19 A (zh-CN) the header reads "3/8 个工作项已完成": "5/8 Work items closed"
FAIL T19 A (zh-CN) the note reads "报告中已排除 2 个已取消的工作项。": "2 cancelled work items are excluded from this report."
FAIL T19 A (zh-CN) the progress bar's aria-valuenow is 38: aria-valuenow 62.5
PASS T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN): no request outside the scenario's list
PASS T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN): no uncaught page error
PASS T19-A-zh-CN active cycle: 3 completed, 2 cancelled of 10 (zh-CN): no "call navigate() in a React.useEffect()" warning

== T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en)
PASS T19-B-en: the active-cycle card shows its progress bar
  (the card: {"header":"10/8 Work items closed","valueNow":"100","width":100,"rows":[{"group":"completed","count":8}],"note":"2 cancelled work items are excluded from this report.","list":{"tag":7,"key":null,"items":[{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"},{"key":null,"type":"Fragment"}]}})
  (console key warnings: 0)
FAIL T19 B (en) the header reads "8/8 work items completed": "10/8 Work items closed"
PASS T19 B (en) the progress bar's aria-valuenow is 100
PASS T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en): no request outside the scenario's list
PASS T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en): no uncaught page error
PASS T19-B-en active cycle: 8 completed, 2 cancelled of 10 (en): no "call navigate() in a React.useEffect()" warning

16 passed, 7 failed
```

#### 7.4.14 基线：B 组（只列失败的行）

```text
== B4 work item page
FAIL B4 editor link: the third argument has noopener and noreferrer: {"calls":[["https://example.com/","_blank"]],"tabs":[{"url":"https://example.com/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/"}]}
FAIL B4 editor link: in the new tab window.opener is null: {"calls":[["https://example.com/","_blank"]],"tabs":[{"url":"https://example.com/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/"}]}
FAIL B4 editor link: in the new tab document.referrer is empty: {"calls":[["https://example.com/","_blank"]],"tabs":[{"url":"https://example.com/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/"}]}
FAIL B4 attachment list: the third argument has noopener and noreferrer: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 attachment list: in the new tab window.opener is null: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 attachment list: in the new tab document.referrer is empty: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/issues/i1/attachments/att1/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 image toolbar download: the third argument has noopener and noreferrer: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 image toolbar download: in the new tab window.opener is null: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 image toolbar download: in the new tab document.referrer is empty: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 full-screen download: the third argument has noopener and noreferrer: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 full-screen download: in the new tab window.opener is null: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 full-screen download: in the new tab document.referrer is empty: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/download/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 full-screen open original: the third argument has noopener and noreferrer: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 full-screen open original: in the new tab window.opener is null: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
FAIL B4 full-screen open original: in the new tab document.referrer is empty: {"calls":[["/api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/api/assets/v2/workspaces/probe-ws/projects/p1/5f1c2a4e-3333-4a2b-9c3d-000000000003/","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/browse/PRB-1"}]}
== B4 open in new tab: project lists
FAIL B4 cycle: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/projects/p1/cycles/c1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/cycles/c1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/cycles"}]}
FAIL B4 cycle: in the new tab window.opener is null: {"calls":[["/probe-ws/projects/p1/cycles/c1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/cycles/c1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/cycles"}]}
FAIL B4 cycle: in the new tab document.referrer is empty: {"calls":[["/probe-ws/projects/p1/cycles/c1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/cycles/c1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/cycles"}]}
FAIL B4 module: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/projects/p1/modules/m1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/modules/m1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/modules"}]}
FAIL B4 module: in the new tab window.opener is null: {"calls":[["/probe-ws/projects/p1/modules/m1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/modules/m1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/modules"}]}
FAIL B4 module: in the new tab document.referrer is empty: {"calls":[["/probe-ws/projects/p1/modules/m1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/modules/m1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/modules"}]}
FAIL B4 project view: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/projects/p1/views/v1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/views/v1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/views"}]}
FAIL B4 project view: in the new tab window.opener is null: {"calls":[["/probe-ws/projects/p1/views/v1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/views/v1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/views"}]}
FAIL B4 project view: in the new tab document.referrer is empty: {"calls":[["/probe-ws/projects/p1/views/v1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p1/views/v1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/views"}]}
FAIL B4 work item: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/browse/PRB-1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/browse/PRB-1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/issues"}]}
FAIL B4 work item: in the new tab window.opener is null: {"calls":[["/probe-ws/browse/PRB-1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/browse/PRB-1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/issues"}]}
FAIL B4 work item: in the new tab document.referrer is empty: {"calls":[["/probe-ws/browse/PRB-1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/browse/PRB-1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects/p1/issues"}]}
== B4 open in new tab: project card
FAIL B4 project card: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/projects/p2/issues","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p2/issues","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects"}]}
FAIL B4 project card: in the new tab window.opener is null: {"calls":[["/probe-ws/projects/p2/issues","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p2/issues","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects"}]}
FAIL B4 project card: in the new tab document.referrer is empty: {"calls":[["/probe-ws/projects/p2/issues","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/projects/p2/issues","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/projects"}]}
== B4 open in new tab: workspace views
FAIL B4 default workspace view: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/workspace-views/assigned","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/workspace-views/assigned","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/workspace-views/assigned"}]}
FAIL B4 default workspace view: in the new tab window.opener is null: {"calls":[["/probe-ws/workspace-views/assigned","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/workspace-views/assigned","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/workspace-views/assigned"}]}
FAIL B4 default workspace view: in the new tab document.referrer is empty: {"calls":[["/probe-ws/workspace-views/assigned","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/workspace-views/assigned","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/workspace-views/assigned"}]}
FAIL B4 saved workspace view: the third argument has noopener and noreferrer: {"calls":[["/probe-ws/workspace-views/wv1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/workspace-views/wv1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/workspace-views/wv1"}]}
FAIL B4 saved workspace view: in the new tab window.opener is null: {"calls":[["/probe-ws/workspace-views/wv1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/workspace-views/wv1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/workspace-views/wv1"}]}
FAIL B4 saved workspace view: in the new tab document.referrer is empty: {"calls":[["/probe-ws/workspace-views/wv1","_blank"]],"tabs":[{"url":"http://127.0.0.1:3462/probe-ws/workspace-views/wv1","openerIsNull":false,"referrer":"http://127.0.0.1:3462/probe-ws/workspace-views/wv1"}]}
== B5 tab overflow menu
FAIL B5 window.close() was not called (count 0): count 1
== B6 notification preview
FAIL B6 the comment and the description notification preview "<p>Tom &amp; Jerry &lt;3</p>" as "Tom & Jerry <3": ["Tom &amp; Jerry &lt;3","Tom &amp; Jerry &lt;3"]
FAIL B6 no preview shows an entity (&amp; or &lt;) as text: ["Tom &amp; Jerry &lt;3","Tom &amp; Jerry &lt;3"]
== B7 comment editor
== B7 draft modal
FAIL B7 draft "only an image": Discard asks to save a draft: {"empty":"closes without asking","only spaces":"closes without asking","only non-breaking spaces":"closes without asking","only line breaks":"closes without asking","only hard breaks (<br>)":"closes without asking","text":"asks to save a draft","only an image":"closes without asking","only a mention":"closes without asking"}
FAIL B7 draft "only a mention": Discard asks to save a draft: {"empty":"closes without asking","only spaces":"closes without asking","only non-breaking spaces":"closes without asking","only line breaks":"closes without asking","only hard breaks (<br>)":"closes without asking","text":"asks to save a draft","only an image":"closes without asking","only a mention":"closes without asking"}
== B8 callout from the description
FAIL B8 (T7) the callout node's data-emoji-unicode is the string "128161": [128161]
== B8 new callout takes the stored logo
FAIL B8 the new callout node's data-emoji-url is https://cdn.example.com/emoji/1f600.png?size=64&set=apple (a plain &): ["https://cdn.example.com/emoji/1f600.png?size=64&amp;set=apple"]
FAIL B8 the editor's HTML output, parsed back, holds the same address (a plain &, no &amp;): ["https://cdn.example.com/emoji/1f600.png?size=64&amp;set=apple"]
== B9 theme names (zh-CN)
FAIL B9 (zh-CN) Power K's theme menu names the themes ["系统偏好","浅色","深色","浅色高对比度","深色高对比度"]: ["System preference","Light","Dark","Light high contrast","Dark high contrast"]
== B9 theme names (en)
FAIL B9 (en) Power K's theme menu names the themes ["System Preference","Light","Dark","Light high contrast","Dark high contrast"]: ["System preference","Light","Dark","Light high contrast","Dark high contrast"]
== B10 work item form cycles, from the workspace home
== B10 work item form cycles, inside the project
== B11 project list empty state
== B11 cycle list empty state
== B11 intake empty state
157 passed, 46 failed
```

#### 7.4.15 A 组（本分支和基线的开头摘要）

```text
build at ff555bf4cf945ea7bfcba613a0338803548b7b33 ($COTMP/probe-app)
current: exit 0, PASS 106, FAIL 0, last: 106 passed, 0 failed
addresses: exit 0, PASS 69, FAIL 0, last: 69 passed, 0 failed
redirects: exit 0, PASS 61, FAIL 0, last: 61 passed, 0 failed
palette: exit 0, PASS 15, FAIL 0, last: 15 passed, 0 failed
p2probe1: exit 0, PASS 37, FAIL 0, last: 37 passed, 0 failed
p2probe2: exit 0, PASS 17, FAIL 0, last: 17 passed, 0 failed
p2probe3: exit 0, PASS 24, FAIL 0, last: all checks passed
p3auth: exit 0, PASS 46, FAIL 0, last: 46 passed, 0 failed
p3lists: exit 0, PASS 28, FAIL 0, last: all checks passed
brand: exit 0, PASS 177, FAIL 0, last: 177 passed, 0 failed
a3: exit 0, PASS 188, FAIL 0, last: 94 pages; 188 passed, 0 failed
```

```text
build at ref: refs/heads/worktree-m1-closeout = c493255ee8c7f418567bacbd8f025de6adbf18e2 ($COTMP/base)
current: exit 0, PASS 106, FAIL 0, last: 106 passed, 0 failed
addresses: exit 0, PASS 69, FAIL 0, last: 69 passed, 0 failed
redirects: exit 0, PASS 61, FAIL 0, last: 61 passed, 0 failed
palette: exit 0, PASS 15, FAIL 0, last: 15 passed, 0 failed
p2probe1: exit 0, PASS 37, FAIL 0, last: 37 passed, 0 failed
p2probe2: exit 0, PASS 17, FAIL 0, last: 17 passed, 0 failed
p2probe3: exit 0, PASS 24, FAIL 0, last: all checks passed
p3auth: exit 0, PASS 46, FAIL 0, last: 46 passed, 0 failed
p3lists: exit 0, PASS 28, FAIL 0, last: all checks passed
brand: exit 0, PASS 177, FAIL 0, last: 177 passed, 0 failed
a3: exit 1, PASS 179, FAIL 9, last: 94 pages; 179 passed, 9 failed
FAIL addresses /probe-ws/profile/u1/assigned: no raw i18n key: ["profile.tabs.assigned (hidden)"]
FAIL current /probe-ws/profile/u1/assigned: no raw i18n key: ["profile.tabs.assigned (hidden)"]
FAIL current /probe-ws/profile/u1/assigned/: no raw i18n key: ["profile.tabs.assigned (hidden)"]
FAIL current /probe-ws/profile/u1/created: no raw i18n key: ["profile.tabs.created (hidden)"]
FAIL current /probe-ws/profile/u1/created/: no raw i18n key: ["profile.tabs.created (hidden)"]
FAIL p2probe1 /probe-ws/profile/u1/assigned: no raw i18n key: ["profile.tabs.assigned (hidden)"]
FAIL p2probe1 /probe-ws/profile/u2/assigned: no raw i18n key: ["profile.tabs.assigned (hidden)"]
FAIL p2probe3 /probe-ws/profile/u1/assigned: no raw i18n key: ["profile.tabs.assigned (hidden)"]
FAIL p3lists /probe-ws/profile/u1/assigned: no raw i18n key: ["profile.tabs.assigned (hidden)"]
```

## 8. 交接与延后项

每一项都写在对应 M 的 `handoffs/M1-closeout.md`（或更早 Phase 的交接），带可核对的关闭条件；总表在 spec 8.1。

| M | 事项 |
|---|---|
| M2–M8 | 本 M 领域里的死成员和死 prop（`domains.mjs --rows M<n>`）；oxlint 的清理（谁改谁清、按规则清一类，第 9 节）；P2–P5 写给它们的 17 份交接各补了关闭条件 |
| M2 | 另有个人设置的主题下拉框画在左上角（Plane 原有，本评审提交写进） |
| M4 | 另有 `viewId as TProfileViews`；工作项弹窗的描述不读 `isMigrationUpdate`；标注块的 Markdown 序列化不转义；`JSONContent` 和 `comment_json` 一起处理 |
| M5 | 另有新建项目时写进封面值的构建路径；附件图标里的第三方标志；复制资源的接口核对请求者能否读取源资源（T17 的评审）；编辑器里要不要支持附件（P3 的交接，M1 设计 3.1） |
| M6 | 另有进度的口径：迭代侧边栏、模块的列表、排序和卡片，`calculateCycleProgress` 读快照的条件 |
| M8 | 另有共享部分的 287 个死成员和死 prop；oxlint 发布前清零、警告改为错误；13 个 `package.json` 的 `license`；AGPL 第 5(a)、5(d)、13 条的验收和 jsDelivr 的两类请求（时点是任何对外的网络部署之前，本评审提交写进）；界面文字的中文覆盖（本评审提交写进）；P5 交的 propel 对应源码、版本号、分享图的地址 |

M1 自己的两份交接（`M0-P5-frontend-trim-notes`、`M0-P6-knip-notes`）在收尾之前已经 `closed`。

记录在案、不改的：`make build` 的 `find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete`，只有在命令行把 `WEBUI_DIST` 覆盖为空时，GNU `find` 才会从当前目录删起（T18 的实现者指出；与 Makefile 的其他变量一样，需要有意覆盖）；`prosemirror-codemark` 补丁之后 `utils.js` 末尾没有换行（生成文件）。

## 9. oxlint 的清零计划、构建体积、锁文件（M1 设计 7.3、7.6、9）

**oxlint**：收尾结束时 694 个警告、39 条规则，每个包的警告数等于上限（`tools/lint-cap.mjs` 自动核对）：web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1；`api-client`、`services`、`shared-state`、`types`、`e2e` 没有警告。按包、按规则（`linttable.mjs`，与 `tools/lint-cap.mjs` 同样在每个包目录里 `oxlint --format=json .`；整分支评审复跑，与 [spec](../specs/closeout.md) 3.3 逐字节相同）：

| 规则 | web | constants | editor | hooks | i18n | propel | ui | utils | 合计 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `eslint(no-shadow)` | 136 | 1 | 22 |  |  | 8 | 3 | 1 | 171 |
| `eslint-plugin-jsx-a11y(no-static-element-interactions)` | 65 |  | 9 |  |  | 1 | 4 |  | 79 |
| `eslint-plugin-promise(always-return)` | 75 |  | 1 |  |  |  |  |  | 76 |
| `eslint-plugin-jsx-a11y(click-events-have-key-events)` | 61 |  | 5 |  |  | 1 | 4 |  | 71 |
| `eslint(no-unneeded-ternary)` | 55 |  |  |  |  |  | 1 |  | 56 |
| `eslint-plugin-react-hooks(exhaustive-deps)` | 35 |  | 1 |  |  |  | 7 |  | 43 |
| `eslint-plugin-react(no-array-index-key)` | 19 |  |  |  |  | 3 | 1 |  | 23 |
| `eslint-plugin-unicorn(consistent-function-scoping)` | 15 |  | 4 | 2 |  | 1 |  |  | 22 |
| `eslint-plugin-jsx-a11y(no-autofocus)` | 17 |  | 3 |  |  | 1 |  |  | 21 |
| `eslint(no-unused-expressions)` | 13 |  |  |  |  |  | 2 | 1 | 16 |
| `eslint-plugin-unicorn(prefer-set-has)` | 9 |  |  |  |  | 1 |  | 2 | 12 |
| `eslint-plugin-jsx-a11y(tabindex-no-positive)` | 9 |  |  |  |  |  | 1 |  | 10 |
| `eslint-plugin-import(no-named-as-default)` |  |  | 8 |  |  |  |  |  | 8 |
| `eslint-plugin-unicorn(no-array-sort)` | 4 |  |  |  |  |  |  | 4 | 8 |
| `eslint(no-constant-binary-expression)` | 8 |  |  |  |  |  |  |  | 8 |
| `oxc(no-map-spread)` | 4 |  | 2 |  |  |  |  | 1 | 7 |
| `eslint(no-useless-escape)` |  |  |  |  |  |  |  | 6 | 6 |
| `oxc(const-comparisons)` | 5 |  |  | 1 |  |  |  |  | 6 |
| `eslint(preserve-caught-error)` | 5 |  |  |  |  |  |  |  | 5 |
| `eslint-plugin-import(no-unassigned-import)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint-plugin-jsx-a11y(img-redundant-alt)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint-plugin-jsx-a11y(prefer-tag-over-role)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint(no-useless-catch)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint(valid-typeof)` | 4 |  |  |  |  |  |  |  | 4 |
| `eslint-plugin-import(no-named-as-default-member)` | 2 |  |  |  | 1 |  |  |  | 3 |
| `eslint-plugin-unicorn(no-array-reverse)` |  |  | 1 |  |  |  |  | 2 | 3 |
| `oxc(no-accumulating-spread)` | 2 |  | 1 |  |  |  |  |  | 3 |
| `eslint-plugin-unicorn(no-single-promise-in-promise-methods)` | 2 |  |  |  |  |  |  |  | 2 |
| `eslint-plugin-unicorn(no-useless-spread)` | 1 |  | 1 |  |  |  |  |  | 2 |
| `eslint-plugin-unicorn(prefer-add-event-listener)` |  |  | 2 |  |  |  |  |  | 2 |
| `eslint-plugin-unicorn(prefer-array-flat-map)` |  |  | 2 |  |  |  |  |  | 2 |
| `eslint(no-empty-pattern)` | 2 |  |  |  |  |  |  |  | 2 |
| `eslint-plugin-jsx-a11y(alt-text)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint-plugin-jsx-a11y(anchor-is-valid)` |  |  |  |  |  |  | 1 |  | 1 |
| `eslint-plugin-react(jsx-no-constructed-context-values)` |  |  |  |  |  |  | 1 |  | 1 |
| `eslint-plugin-unicorn(no-instanceof-builtins)` |  |  |  |  |  |  |  | 1 | 1 |
| `eslint-plugin-unicorn(prefer-array-find)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint(no-new)` |  |  | 1 |  |  |  |  |  | 1 |
| `eslint(no-unsafe-optional-chaining)` | 1 |  |  |  |  |  |  |  | 1 |
| **合计** | 565 | 1 | 65 | 3 | 1 | 16 | 25 | 18 | **694** |

**清零计划**（写进 M2–M8 各自的收尾交接）：
1. **谁改谁清**：每个 M 改到的文件，在该 M 结束时没有 oxlint 警告；
2. **按规则清一类**：每个 M 另外按规则集中清掉至少一类，优先能机械修复的 `no-shadow`、`promise/always-return`、`no-unneeded-ternary`，写进该 M 的 review；
3. **M8 发布之前清零**，之后 `.oxlintrc.json` 把警告改为错误、删掉各包的上限（总体设计 7.6 的原始要求）。
每一步上限随之调低。

**构建体积**（P1 的方法：先删掉 `web/apps/web/build`，再 `make build-web`，然后按类别统计）：

| 项 | P1 之前（`0f2b3e0`） | 收尾之前（`c493255`） | 收尾之后（`73f8c36`） | 对 P1 之前 |
|---|---|---|---|---|
| JS | 1055 个，15,151,327 字节 | 397 个，6,818,887 | 396 个，6,586,775 | −56.5% |
| CSS | 3 个，326,068 | 3 个，296,816 | 3 个，293,801 | −9.9% |
| 字体 | 34 个，6,849,620 | 25 个，3,755,608 | 25 个，3,755,608 | −45.2% |
| 其他 | 147 个，9,905,117 | 109 个，6,435,038 | 107 个，4,126,907 | −58.3% |
| 最大的 chunk | `toolbar-*.js` 1,821,937 | `use-parse-editor-content-*.js` 1,379,320 | 同名 1,378,580 | |
| **合计** | **1239 个，32,232,132 字节（30.7 MiB）** | **534 个，17,306,349（16.5 MiB）** | **531 个，14,763,091（14.1 MiB）** | **−54.2%** |

收尾让构建再小 2,543,258 字节：JS 少 232,112（死组件、`sanitize-html`），"其他"少 2,308,131（封面从 JPEG 换成 WebP、删掉活动迭代的两张图、T20 重新编码的截图），CSS 少 3,015。逐个 Task 的变化在 spec 3.4。

**锁文件**（`lock-diff.mjs`）：除删除以外的变化只有 `prosemirror-codemark@0.4.2` 的补丁哈希（T9）和 `utils` 开发依赖的 `jsdom`（锁文件里已有的 30.1.1，T8）。删掉的 17 个包是 `sanitize-html` 2.17.7、`@types/sanitize-html` 和只有它们用的 15 个，以及 `catalogs` 里的两条（T8）；propel 的 `cmdk` 依赖（T4）。没有新增或升级的包。
