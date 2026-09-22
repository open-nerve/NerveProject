# M1/P1 工具链、遗留物与多语言：评审记录

| 项 | 内容 |
|---|---|
| Phase | M1/P1 `web-hygiene` |
| 日期 | 2026-09-23 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P1-web-hygiene.md) / [plan](../plans/P1-web-hygiene.md) |
| 分支 | `worktree-m1-design`，从 `main` 的 `0e43771` 分出，提交从 `43f6ae9` 到本评审记录所在的提交 |
| 评审 | 各 Task 评审（sonnet，第 3 节）；整分支评审（opus，`0e43771..d671331`）：With fixes；修复轮后的范围复核（sonnet）：Approved |

## 1. 范围与结果

P1 是 M1 的第一个 Phase，清掉迁入时带进来的工具链遗留和部署遗留，只留中英文，把三条"靠人记着"的规则改成自动检查；不删除任何产品功能（spec 1）：

- **lint 上限自动核对**（spec 2.3）：新增 `tools/lint-cap.mjs`，13 个包的 `check:lint` 改为调用它，警告数必须**等于**上限。上限合计 **995 → 954**：

  | 包 | P1 之前 | P1 之后 |
  |---|---|---|
  | web | 779 | 777 |
  | editor | 75 | 75 |
  | utils | 34 | 34 |
  | ui | 32 | 31 |
  | propel | 59 | 29 |
  | services | 6 | 0 |
  | hooks | 4 | 4 |
  | i18n | 3 | 1 |
  | constants | 2 | 2 |
  | types | 1 | 1 |
  | shared-state、`@nerve/api-client`、`@nerve/e2e` | 0 | 0 |
  | **合计** | **995** | **954** |

- **关键词守卫**（spec 2.4）：新增 `tools/keywords.mjs` + `tools/keywords.json`，`make lint-web` 第一步运行它。P1 结束时 **12 条规则、0 条例外**，每条规则在删除对应内容的同一个 Task 中加入。
- **前端单元测试**：新增 `make test-web`（`turbo run test`），持续集成 `web` 任务运行它；services 原有 13 个测试，i18n 新增 10 个（`language.test.ts`）。
- **删除**：部署遗留（`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`、`serve` 及其脚本、`sw.js`/workbox、`public/favicon/` 等）；Storybook（配置、45 个 stories 及只为它们存在的文件）；knip 报出的未使用依赖（`buffer` 除外）及只为它们存在的文件；`packages/services` 中没有调用方的 49 个文件（**61 → 12 个文件**）；Next.js 垫片中的死文件（`image.tsx`、`script.tsx`、`next-script.d.ts`）。整个分支 `0e43771..3612074` 共 **751 个文件改动，+6642/-150925 行，691 个文件被删除**（`git diff --shortstat`）。
- **两条构建警告**消除（`MODULE_TYPELESS_PACKAGE_JSON`、`vite-tsconfig-paths` 提示）；每个包写自己的 `tsBuildInfoFile`。
- **React #418** 修复：`HydrateFallback` 不再读取主题，`LogoSpinner` 改为两张静态图片配 CSS `dark:` 变体选择。
- **多语言只留 en、zh-CN**（**21 → 2**，删除 532 个语言文件，剩 56 个）：新增 `toSupportedLanguage`，不支持的语言回退英文；`sync-check.ts` 改写为中英文键双向一致性检查，接入 `make lint-web`（`check:sync`）；删除翻译键生成脚本和 8 个企业版命名空间。
- **`make web-dev`** 改为 `turbo run dev --filter=web... --concurrency=12`，同时监视 web 依赖的 10 个包。
- **Task 9A（控制者追加）**：`tools/` 下的门禁脚本本身不受 `make lint-web` 的 lint/格式检查覆盖，新增根 `check:lint`/`check:format` 和 turbo 根任务 `//#check:lint`/`//#check:format`，递归覆盖 `tools/`，唯一豁免 `tools/plane-schema/**`。
- **收尾检查**：`make lint-web` **52** 个任务（含 2 个 `//#check:*` 根任务）；`make test-web` **12** 个任务；`make build-web` **11** 个任务；`make knip` 报告 **343** 处（Unused files 123 + Unused dependencies 1 + Unused exports 133 + Unused exported types 81 + Unused exported enum members 4 + Duplicate exports 1；Configuration hints 另计，随本地生成状态在 0–3 之间浮动）；构建体积从 1239 个文件/32,232,132 字节降到 684 个文件/27,161,767 字节（附录 5）；文档（前端改动清单、README、M0 设计、两份 M1 交接）已同步。

spec §4 的 10 项验收标准全部满足。第 3 项验收标准写的"过期的例外…以 1、1、2、2 退出"和第 8 项"资料语言为 `fr`"的措辞，在最终修复轮之后按下述方式满足，与原文字面略有不同但结论一致：

- `until` 到期的判定原来只在 Task 2 的例外 schema 里校验形状，真正的"到期即失败"是整分支评审后才补上的（规则文件新增顶层 `phase` 字段，第 4 节 Ruling）；spec 2.4 和第 4 节验收已在收尾提交 `3612074` 中同步这一点，验收标准本身也已改写为与实现一致。
- 第 8 项"资料语言为 `fr`"的浏览器核对不仅覆盖了 `profile.store.ts` 写入路径，修复轮还补上了 `use-translation.ts` 里最后一处未经 `toSupportedLanguage` 处理的 `as TLanguage` 强转（附录 4 引用），比 spec 原文列出的三处更完整。

## 2. 各 Task 的评审

| Task | 内容 | 提交范围 | 核实方式 |
|---|---|---|---|
| 1 | `tools/lint-cap.mjs`；13 个包 `check:lint` 改调用；P1 之前基线记录 | `686f89d`..`43f6ae9`（另有裁定后的纯文档提交 `b358e88`） | 实现者 sonnet：Step 3 遇阻——`oxfmt --check tools/` 连带检查了不符合格式的 `tools/plane-schema/README.md`，与 `lint-cap.mjs` 无关，NEEDS_CONTEXT。**控制者裁定**：格式检查范围收窄为 `tools/*.mjs tools/*.json`（Task 1 时只有 `tools/*.mjs`），`oxlint` 不变仍检查整个 `tools/`，`plane-schema` 不开例外也不改它的文件（spec 第 3 节第 13 项、计划"控制者评审补充"第 3 条）；实现者据裁定继续完成。评审（sonnet）Approved；两条 Minor 留给整分支评审：报告把 `lint-cap.mjs` 写成 66 行（实际 63 行）；`throw oxlint.error`、`d.labels[0]?.span` 两处理论上可能抛出未捕获异常（仍会以非 0 退出，不影响正确性） |
| 2 | 关键词守卫（`tools/keywords.mjs`+`.json`）；部署遗留；`serve`、`pnpm-workspace.yaml` 注释 | `b358e88`..`f40e2af`（实现 `6fd673c`；修复轮 1 `19663bf`+文档 `01266e4`；修复轮 2 `4eb7bc5`+文档 `f40e2af`） | 实现者 sonnet DONE（并发现 pnpm 误写入 `allowBuilds` 的一条杂项，控制者复核 `pnpm install` 空操作后确认不会复发）。评审（sonnet）Needs fixes，2 个 Important：(1) 已跟踪但工作区已删除的文件会跳过路径规则；(2) 同一文件里同一段原文出现多处时，一条例外会把它们全部盖住，后来新增的同样一处也会被顺带盖住。**控制者裁定**：(1) 不改——这条分支只有文件已从磁盘删除时才可达，复活的文件仍会命中路径规则，跳过已删除的文件是设计本意；(2) 从根源修——例外加可选 `count`（默认 1），实际命中数必须与它相等，与 lint 上限"必须相等"的做法一致，不引入行号（行号会随改动漂移）。修复轮 1（`19663bf`）后范围复核 Approved，带一条 Minor：两条键相同、`count` 也相同的例外会一起悄悄通过。**裁定**：从根源修——同一 `(rule, path, match)` 只能登记一条例外，重复登记以 2 退出（`4eb7bc5`），控制者直接复核 6 行差异 |
| 3 | 没有调用方的 turbo 任务和脚本；删除 Storybook；`make test-web` | `f40e2af`..`c52c45a` | 实现者 sonnet DONE_WITH_CONCERNS（81 个文件，−14196 行；propel 59→29、ui 32→31；pnpm 曾把一条 `'@swc/core': set this to true or false` 误插入 `allowBuilds`，实现者已移除，控制者重跑 `pnpm install` 确认不会复发）。评审（sonnet）Approved，无任何级别的发现 |
| 4 | 未使用的依赖及只为它们存在的文件；Next.js 垫片死文件 | `c52c45a`..`538a8b4` | 实现者 sonnet DONE（web 上限 777；knip 的依赖类报告只剩 editor 的 `buffer`）。评审（sonnet）Approved，无发现；评审者另行对每个被删文件和依赖做了孤儿导入的 grep 复核，未发现遗漏 |
| 5 | 修剪 `packages/services` | `538a8b4`..`d923efe` | 实现者 sonnet DONE（删 49 个文件，留 12 个，services 上限 0，13 个地址测试仍通过）。评审（sonnet）Approved，两条 Minor 留给后续：(a) `helpers/index.ts` 仍导出 `ensureAPITrailingSlash`，web 未导入——已存在的情况，归 P3 清零 knip 时处理；(b) `services-files` 规则的命中样本写了一个从未存在过的路径——样本只用于自检正则，判别力不受影响（后在整分支评审的修复轮中一并改为真实存在过的路径，见第 4 节） |
| 6 | 两条构建警告；每个包自己的增量编译文件 | `d923efe`..`ef29404` | 实现者 sonnet DONE（两条警告均消失，knip 的 `tailwind-config` 提示消失）。评审（sonnet）Approved，无发现；评审者独立确认 `tailwind-config` 唯一的使用者走 `exports` 子路径、`.turbo/` 在任意深度都被 `.gitignore` 挡住、没有残留的 `vite-tsconfig-paths` 引用 |
| 7 | 修复 React #418 | `ef29404`..`218574d` | 实现者 sonnet DONE（#418 消失，两张动画图片都在 DOM 中，CSS 决定显示哪一张；探测脚本前后输出已存档，供 review 附录使用）。评审（sonnet）Approved；评审者额外核对了编译后的 CSS（`dark:` 是 React 树之外的属性选择器，不会造成结构性不一致）和构建产物，唯一的 Minor（重复的 `alt` 文本）已核实并当场不作处理 |
| 8 | 只留中英文，不支持的语言回退英文 | `218574d`..`bbd5dc3` | 实现者 sonnet DONE（删除 532 个语言文件；`toSupportedLanguage` 带 10 个测试；`make test-web` 增至 12 个任务；浏览器探测 `fr`→`en`）。评审（sonnet）Approved；评审者逐一追踪了每个读取语言的位置（本地存储、资料获取、资料更新、命令面板）以及 `use-translation.ts` 中一处既有的 `as TLanguage` 强转，未发现未受保护的路径；一条计划本身要求、与相邻时区选择框行为一致的 Minor（资料加载完成前选择框显示"English"而非占位符） |
| 9 | 删除翻译键生成；中英文键一致性检查接入 `make lint-web`；删除死文案 | `bbd5dc3`..`c9d127c` | 实现者 sonnet DONE（生成脚本消失，`check:sync` 使 `lint-web` 增至 50 个任务，8 个企业版命名空间按语言各删除，i18n 上限 1）；额外发现并清理了本地残留的 `keys.generated.ts`（原被 `.gitignore` 挡住），一并移除 `.gitignore`/`.oxfmtrc.json`/`knip.jsonc` 中对应的特例。评审（sonnet）Approved；评审者确认 `sync-check.ts` 双向核对、命名空间从文件系统枚举（没有第二份硬编码列表）、缺失语言目录会失败；一条计划要求的 Minor：无效 JSON 会以未格式化的原始堆栈呈现（仍以非 0 退出） |
| 9A | 门禁覆盖 `tools/`（根 `check:lint`/`check:format`，`//#check:lint`/`//#check:format`） | `c9d127c`..`73eb04a`（实现 `383c5da`，由控制者修正 trailer；修复轮 `672f103`；文档 `73eb04a`） | 实现者 sonnet DONE（`lint-web` 50→52；实现者的缓存实验声称 turbo 只按根任务的命令文本哈希，`package.json` 未列入 `inputs`）。评审（sonnet）Approved，1 个 Important（举证性质）+2 个 Minor。**控制者裁定**：(1) 格式检查范围从 `tools/*.mjs tools/*.json` 顶层通配符改为整个 `tools/` 递归，唯一豁免 `tools/plane-schema/**` 写进 `.oxfmtrc.json` 的 `ignorePatterns`——门禁不能让以后 `tools/<子目录>/x.mjs` 逃过格式检查（oxlint 已经看得见）；(2) 缓存说法补一次对照实验：只改 `package.json` 里与门禁脚本无关的字段，观察是否命中缓存。修复轮（`672f103`）关闭了两条：嵌套探针证明旧的顶层通配确有缺口，现已用递归覆盖并把 `plane-schema` 的豁免声明在 `.oxfmtrc.json`；对照实验把缓存问题反过来定案——turbo 对根任务按**整个根 `package.json`** 哈希，不只是命令文本，`inputs` 因此仍保持 `["tools/**"]` 不变，报告改为写明实际验证到的结论 |
| 10 | `make web-dev` 同时监视各包；文档与交接；完整验收 | `73eb04a`..`d671331`（实现 `6f2059e`；控制者直接修复 `d671331`） | 实现者 sonnet DONE（`web-dev` 改为 11 个常驻任务；README、M0 设计、前端改动清单 1.3、两份交接均已记录 P1；含干净克隆的完整验收和 `make e2e` 5 个测试通过）。评审（sonnet）**spec ❌**：`M0-design.md` §6.3 的 CI 行还是修正前的措辞，与已经改过的 §6.1 矛盾。**控制者直接复核并修复**（`d671331`）：把 §6.3 的 `web` 任务描述同步为"也检查 `tools/` 下的脚本"。其余项（各上限、规则条数、knip 各类合计、语言集合、CI 步骤顺序）逐一对照真实的 `Makefile`/`turbo.json`/`package.json`/`ci.yml` 核实无误 |

## 3. 整分支评审（opus）：With fixes

评审范围 `0e43771..d671331`。0 Critical、2 Important、10 Minor。评审者独立复现了 52/12/11 的任务数、343 处 knip 发现、lint 上限表、一次强制冷构建和预渲染后的标记，并在一个临时仓库里用 13 种方式探测守卫，没有一次真实命中被当成退出码 0。

| # | 类别 | 发现 | 处理 |
|---|---|---|---|
| Important 1 | Important | `tools/keywords.mjs` 的样本自检只针对内容规则的 `content`/路径规则的 `path` 跑 `hit`/`miss`；内容规则真正用来选文件的 `files` 正则完全没有样本覆盖 | 已修（`d42408f`）：非路径规则新增 `samples.files.hit`/`samples.files.miss`，`loadRules()` 用同一套逻辑校验；5 条内容规则（`serve`、`storybook`、`next-script-image`、`i18n-key-generator`、`i18n-applications`）补上样本，其中 `storybook` 覆盖它 `files` 声明的全部 5 个候选根文件之一 |
| Important 2 | Important | 例外的 `until` 只在加载时校验*形状*（正则 `M\d+(/P\d+)?`），从未真正判断"是否到期"；spec 7.4 承诺的"到期即失败"没有兑现 | **裁定**：让工具兑现设计的承诺，而不是把设计改弱。已修（`d42408f`）：规则文件新增顶层 `phase`（当前 `"M1/P1"`），每条仍命中的例外都与它比较，到期的单独报告并计入退出码 1 的汇总行 |
| Minor（8 条，编号 3–9、11） | Minor | 注释错误（"多个例外可共享一个键"，实际重复例外自 `4eb7bc5` 起就会失败）；`count` 不符时的报错只给计数、不给具体命中；`services-files` 规则的命中样本路径从未存在过；`lint-cap.mjs` 的 JSON 解构在 `diagnostics` 缺失时不会在 `try` 内抛出，真正崩溃点在 `try/catch` 之外的裸 `TypeError`；`sync-check.ts` 不核对磁盘上的命名空间文件与 `NAMESPACES` 注册表是否一致，也没有显式处理"两种语言都没有任何命名空间文件"的极端情况；`use-translation.ts` 还有一处未经 `toSupportedLanguage` 的 `as TLanguage` 强转；`make lint-web` 的 turbo 调用没有 `--continue`，第一个失败的任务会隐藏其余任务的结果；knip 门禁化的时间点在 `Makefile`、`knip.jsonc`、`.github/workflows/ci.yml` 三处仍写"M1"而不是"M1/P3" | 全部已修（`d42408f`、`0136fc2`、`e56ae0c`、`246e4f4`），详见第 4 节 |
| Minor 10、12（延后） | Minor | 规则的作用范围有边界情况未覆盖；`tailwind-config`、`typescript-config` 没有 `check:format`/`check:lint`，也没有任务给仓库根目录自己的文件做格式检查 | 延后到本次 review 的移交事项（第 6 节），另有两张约 1 MB 的加载动画 GIF 留给 P5 |

### 3.1 修复后的复核（sonnet，范围复核）

修复提交 `d42408f`、`0136fc2`、`e56ae0c`、`246e4f4`（详细内容见 `.superpowers/sdd/P1-web-hygiene/final-fixes-report.md`）之后，Approved：九项发现全部真正关闭，没有引入新问题。评审者手工核对了 `phase` 比较的边界（相等、早于、晚于三种情况都用真实的例外注入验证）、逐一核对了新增的 5 组 `samples.files` 样本是否贴合每条规则真实的作用范围，并查阅 Turborepo 文档确认 `--continue` 仍然会让整体命令以最高的子任务退出码失败（`make lint-web` 依然会因为任何一个失败任务而失败）。控制者随后把文档同步进 `3612074`（M1 设计 7.4、spec 2.4 及验收标准）。

## 4. 控制者的裁定

| 裁定 | 内容 | 理由 |
|---|---|---|
| 1 | 一次性脚本目录改用会话临时目录 `$P1TMP`（`/private/tmp/claude-501/.../scratchpad/nerve-p1/`），不用 `/tmp/nerve-p1/` | 本次会话的沙箱不允许随意使用 `/tmp`；只影响路径，不影响逻辑 |
| 2 | 新增 Task 9A：门禁必须覆盖自己（`tools/lint-cap.mjs`、`tools/keywords.mjs` 本身不属于任何工作区包，原本不受 `make lint-web` 检查） | 门禁看不见自己是门禁自身的漏洞；代价是 `lint-web` 多两个任务（50→52） |
| 3 | `tools/` 格式检查的范围：先窄后宽。Task 1 时先用 `tools/*.mjs`（`oxfmt --check tools/` 会连带检查不符合格式的 `tools/plane-schema/README.md`）；Task 9A 复审后改为整个 `tools/` 递归，唯一豁免 `tools/plane-schema/**` 声明在 `.oxfmtrc.json` | 门禁看的是脚本本身；仓库的 Markdown 文档一直不做格式检查，`plane-schema` 是 M0 的快照工具，不单独开例外也不改它的文件。递归覆盖是因为手工命令用通配符是一次性的，常设门禁必须管住以后放在子目录里的脚本 |
| 4 | 例外必须覆盖确定的处数：加可选 `count`（默认 1），实际命中数必须与它相等，多了少了都失败；不引入行号 | 同一文件里同一段原文可能出现多处（例如锁文件里 `serve@14.2.5:`），原来按三元组匹配的例外会把它们一起盖住，后来新增的同样一处也会被顺带盖住，与"一条例外只覆盖一处"的设计意图不符。行号会随改动漂移，会逼着人频繁改例外 |
| 5 | 同一 `(rule, path, match)` 只能登记一条例外，重复登记以 2 退出 | Task 2 范围复核发现两条键相同、`count` 也相同的例外会一起悄悄通过——旧代码是靠巧合才拦住这种情况；把它变成显式规则，避免以后有人为同一处命中登记两条互不相关的理由 |
| 6 | 整分支评审后：把内容规则的 `files` 选择器纳入样本自检（新增 `samples.files.hit`/`miss`） | 守卫自己校验"规则能不能正确识别命中"，但从未验证过"规则到底在检查哪些文件"这件事；一个写错的 `files` 正则会让整条规则悄悄失效而不报错 |
| 7 | 整分支评审后：让 `until` 真正判定到期，而不是把设计文档的措辞改弱 | spec 7.4 已经承诺"例外到期即失败"；工具当时只校验格式，没有兑现这句话。裁定是让工具兑现承诺（加顶层 `phase` 字段，与到期比较），而不是删掉这句承诺 |
| 8 | 整分支评审后：Minor 3、4、5、6、7、8、9、11 一并在同一个修复轮中处理；Minor 10（规则作用范围的边界情况）和 Minor 12（`tailwind-config`/`typescript-config` 及仓库根目录自身文件没有格式检查）延后到本次 review 的移交事项，两张约 1 MB 的加载动画 GIF 留给 P5 | 8 条 Minor 都很小，而且都在门禁或门禁的可用性上，后续 Phase 会直接继承它们；延后的两条是范围性问题，不是本次改动引入的缺陷，适合放进移交事项让后续 Phase 心里有数而不是仓促打补丁 |

**判为不改的发现（附理由）**：

- Task 2 评审 Important 1——"已跟踪但工作区中已经删除的文件会跳过路径规则"：这条分支只有在文件已经从磁盘上被删除、但索引里还在时才可达；一个被复活的文件仍然存在于磁盘上，照常会命中它的路径规则。跳过工作区中已经删除的文件是 spec 2.4 写明的设计意图（"工作区中已经删掉、索引中还在的文件跳过"），不是遗漏。
- Task 5 评审 Minor (a)——`helpers/index.ts` 仍导出 web 未导入的 `ensureAPITrailingSlash`：这是修剪 `packages/services` 之前就存在的情况，不属于 P1 的改动范围，归入本次 review 的移交事项，等 P3 把 knip 改为门禁时一并清理（第 6 节）。

## 5. 附录（M1 设计 7.5）

### 5.1 Task 1 Step 9：lint-cap 的错误/参数核对与缓存核对

错误与参数（`src/lint-probe.ts` 是一个无法解析的临时文件）：

```
$ printf 'export const = ;\n' > web/packages/types/src/lint-probe.ts
$ pnpm --dir web/packages/types exec node ../../../tools/lint-cap.mjs 1; echo "exit=$?"
src/lint-probe.ts:1:14  syntax  Unexpected token
@plane/types: oxlint reports 1 error (listed above); errors are never allowed.
exit=1
$ rm web/packages/types/src/lint-probe.ts
$ pnpm --dir web/packages/types exec node ../../../tools/lint-cap.mjs; echo "exit=$?"
usage: node tools/lint-cap.mjs <cap>
exit=2
```

六种缓存情况（单独的缓存目录，冷启动之后逐一改动并恢复；23 个任务 = 13 个 `check:lint` + 它们依赖的 10 个包的构建）：

```
== cold
Tasks:    23 successful, 23 total
Cached:    0 cached, 23 total
== warm
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
== one warning fewer: the empty file that is the only warning of types is gone
@plane/types:check:lint: @plane/types: 0 oxlint warnings, fewer than the cap of 1. Lower the cap in the check:lint script of package.json to 0 in the same commit.
Tasks:    22 successful, 23 total
Cached:    6 cached, 23 total
Failed:    @plane/types#check:lint
== restored
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
== one warning more: an unused variable in web
web:check:lint: core/lint-probe.ts:2:9  eslint(no-unused-vars)  Variable 'unused' is declared but never used. Unused variables should start with a '_'.
web:check:lint: web: more oxlint warnings than the cap of 779. Fix the new warnings (every warning is listed above); the cap may only go down.
Tasks:    22 successful, 23 total
Cached:    22 cached, 23 total
Failed:    web#check:lint
== restored
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
== cap changed: types 1 -> 2
@plane/types:check:lint: @plane/types: 1 oxlint warning, fewer than the cap of 2. Lower the cap in the check:lint script of package.json to 1 in the same commit.
Tasks:    22 successful, 23 total
Cached:    6 cached, 23 total
Failed:    @plane/types#check:lint
== restored
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
== the script changed
Tasks:    23 successful, 23 total
Cached:    10 cached, 23 total
== restored
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
== a dependency changed: the source of hooks
Tasks:    23 successful, 23 total
Cached:    14 cached, 23 total
== restored
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
```

读法：types 的源码或 `package.json` 变了，types 的构建和依赖它的包全部重跑；只改 web 时只有 web 的 lint 重跑；改脚本（`lint-cap.mjs` 列为 `check:lint` 的输入）时 13 个 `check:lint` 全部重跑、10 个构建命中；改 hooks 时 hooks 和依赖它的 editor、propel、ui 的构建和 lint 以及 web 的 lint 重跑。每个 `restored` 都是满命中，最后 `git status --short` 没有输出。

### 5.2 Task 2 Step 12：关键词守卫的失败模式核对

```
== an exception that matches nothing
exit=1  keywords: 0 hits without an exception, 1 stale exception (tools/keywords.json).
== an invalid pattern
exit=2  keywords: rule deploy-files: "path" is not a valid regular expression: Invalid regular expression: /(unclosed/: Unterminated group
== a hit sample the pattern misses
exit=2  keywords: rule prettierignore does not match its hit sample "web/packages/ui/.prettierrc"
== the g flag
exit=2  keywords: rule serve: the flags of "content" may only be i, m, s, u
== an exception without an expiry
exit=2  keywords: exception {"rule":"serve","path":"a","match":"b","reason":"probe"} needs a "reason" and an "until" such as "M3" or "M1/P4"
== an untracked file that hits a rule
exit=1  keywords: 1 hit without an exception, 0 stale exceptions (tools/keywords.json).
serve  web/apps/web/keywords-probe/package.json:1  "serve\":"
== an unreadable file
exit=2  keywords: cannot read web/apps/web/keywords-probe/package.json: EACCES: permission denied, open 'web/apps/web/keywords-probe/package.json'
```

七种情形全部核对（无效正则、命中样本不命中、非法 flags、例外缺到期、未跟踪文件被检查、不可读文件、以及一个不再命中任何内容的过期例外）；每次注入之后都恢复，`git status --short` 与 `git diff HEAD -- tools/keywords.json tools/keywords.mjs` 均无输出。

### 5.3 Task 9 Step 10：中英文键一致性检查的双向性与失败传播

```
== as committed
en and zh-CN have the same namespaces and keys, each key defined once.
exit=0
== a key missing in zh-CN
accessibility: aria_labels.projects_sidebar.workspace_logo is missing in zh-CN
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== a key only in zh-CN
accessibility: aria_labels.probe is missing in en
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== one key in two namespaces
en: aria_labels.projects_sidebar.workspace_logo is defined in accessibility.json and home.json
zh-CN: aria_labels.projects_sidebar.workspace_logo is defined in accessibility.json and home.json
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== a key that is also a prefix
en: aria_labels.projects_sidebar is a key and also the prefix of other keys
zh-CN: aria_labels.projects_sidebar is a key and also the prefix of other keys
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== zh-CN missing
src/locales/zh-CN is missing
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
```

`make lint-web` 失败传播的核对（删掉一个 zh-CN 键之后）：

```
@plane/i18n:check:sync: accessibility: aria_labels.projects_sidebar.workspace_logo is missing in zh-CN
@plane/i18n:check:sync: en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
Failed:    @plane/i18n#check:sync
```

`make` 以非 0 退出（`make-exit=2`）；恢复文件后 `pnpm exec turbo run check:sync` 命中缓存（`Cached: 1 cached, 1 total`），证明任务的缓存键正确地取决于包内输入。

### 5.4 Task 7 探测脚本：React #418

脚本（`$P1TMP/hydration-probe.mjs`，配合 `$P1TMP/probe.sh` 启动的一次性静态服务器）：在亮色、暗色两种 `colorScheme` 下分别打开 `/` 和一个深层路径 `/acme/projects/x/issues`，统计控制台是否出现 #418，并拦截应用脚本单独读取预渲染的 `index.html` 里哪张图片可见。

修复前：

```
[light] /: #418 true
[light] /acme/projects/x/issues: #418 true
[light] prerendered index.html: data-theme=light, visible images []
[dark] /: #418 true
[dark] /acme/projects/x/issues: #418 true
[dark] prerendered index.html: data-theme=dark, visible images []
port 52906 is free
```

修复后：

```
[light] /: #418 false
[light] /acme/projects/x/issues: #418 false
[light] prerendered index.html: data-theme=light, visible images ["/assets/logo-spinner-light.gif"]
[dark] /: #418 false
[dark] /acme/projects/x/issues: #418 false
[dark] prerendered index.html: data-theme=dark, visible images ["/assets/logo-spinner-dark.gif"]
port 52976 is free
```

修复后的预渲染 HTML 片段（两张图片始终都在 DOM 中，只有 Tailwind 的 `dark:` 变体决定哪张可见）：

```html
<div class="relative flex h-screen w-full items-center justify-center bg-canvas">
  <div class="flex items-center justify-center">
    <img src="/assets/logo-spinner-light-DmBc4A-4.gif" alt="logo" class="h-6 w-auto object-contain sm:h-11 dark:hidden"/>
    <img src="/assets/logo-spinner-dark-Cc1AqekO.gif" alt="logo" class="hidden h-6 w-auto object-contain sm:h-11 dark:block"/>
  </div>
</div>
```

### 5.5 Task 8 探测脚本：语言回退到英文

脚本（`$P1TMP/lang-probe.mjs`，全文）：拦截 Plane 的启动接口，让一个资料语言分别为 `fr`、`zh-CN`、`en` 的用户登录，报告最终的 `<html lang>` 和 `localStorage.userLanguage`。

```js
// One-off (M1/P1 Task 8): signs a stubbed user in whose profile language is each of the given values, and
// reports the language the app then uses (<html lang>) and stores (localStorage userLanguage). The stubs
// answer the Plane endpoints the app calls at start; every other /api request gets 404. Run from the
// repository root (Playwright comes from e2e/). usage: node lang-probe.mjs <port> <language>...
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const [port, ...languages] = process.argv.slice(2);
const stubs = (language) => ({
  "/api/instances/": { instance: { is_setup_done: true }, config: { is_email_password_enabled: true } },
  "/api/users/me/": { id: "u1", email: "probe@example.com", display_name: "probe", first_name: "", last_name: "" },
  "/api/users/me/profile/": { id: "p1", user: "u1", language, is_onboarded: true, theme: {} },
  "/api/users/me/settings/": { id: "u1", email: "probe@example.com", workspace: {} },
  "/api/users/me/workspaces/": [],
});
const browser = await chromium.launch();
try {
  for (const language of languages) {
    const page = await browser.newPage();
    await page.route("**/api/**", (route) => {
      const json = stubs(language)[new URL(route.request().url()).pathname];
      return json ? route.fulfill({ json }) : route.fulfill({ status: 404, json: {} });
    });
    await page.goto(`http://127.0.0.1:${port}/`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1500);
    const result = await page.evaluate(() => ({
      lang: document.documentElement.lang,
      stored: localStorage.getItem("userLanguage"),
    }));
    console.log(`profile language ${JSON.stringify(language)}: <html lang>=${result.lang}, stored=${result.stored}`);
    await page.close();
  }
} finally {
  await browser.close();
}
```

修复后的输出（修复前，同一探测对 `fr` 会得到 `<html lang>=fr, stored=fr`，即资料里的原始值直接透传）：

```
profile language "fr": <html lang>=en, stored=en
profile language "zh-CN": <html lang>=zh-CN, stored=zh-CN
profile language "en": <html lang>=en, stored=en
port 53814 is free
```

### 5.6 Task 10 探测脚本：`make web-dev` 监视各包

脚本：`$P1TMP/web-dev-probe.sh` 在自己的进程组中启动 `make web-dev`，运行 `$P1TMP/dev-probe.mjs`（打开开发服务器首页，改 `@plane/constants` 的 `SITE_NAME`，不手动刷新，等待页面上的值变化，再恢复），结束时终止整个进程组并确认端口 3000 释放。

`Makefile` 改动之前（M0 的 `make web-dev` 只运行 web 自己的开发服务器）：

```
before: "Plane | Simple, extensible, open-source project management tool."
edited: "P1 watch probe" shown after not within 60 s; page loads: 0
restored: the old value shown after 1 ms
console errors about #418 or process: 0
all web-dev processes stopped
port 3000 is free
```

`Makefile` 改动之后：

```
before: "Plane | Simple, extensible, open-source project management tool."
edited: "P1 watch probe" shown after 1011 ms; page loads: 1
restored: the old value shown after 1006 ms
console errors about #418 or process: 0
all web-dev processes stopped
port 3000 is free
```

### 5.7 构建体积：Task 1 Step 2 对比 Task 10 Step 10

| 项 | P1 之前（`0f2b3e0`，Task 1 Step 2） | P1 之后（干净克隆，Task 10 Step 10） |
|---|---|---|
| JS | 1055 个文件，15,151,327 字节 | 505 个文件，10,270,415 字节 |
| CSS | 3 个文件，326,068 字节 | 3 个文件，322,306 字节 |
| 字体 | 34 个文件，6,849,620 字节 | 34 个文件，6,849,620 字节 |
| 其他 | 147 个文件，9,905,117 字节 | 142 个文件，9,719,426 字节 |
| 最大的 chunk | `assets/toolbar-Bnxh0Ws1.js`，1,821,937 字节 | `assets/toolbar-*.js`（内容哈希每次构建不同），1,821,937 字节 |
| 语言 chunk | 588 个（21 种语言 × 28 个命名空间） | 40 个（2 种语言 × 20 个命名空间） |
| **合计** | **1239 个文件，32,232,132 字节（30.7 MiB）** | **684 个文件，27,161,767 字节（25.9 MiB）** |

Task 10 Step 10 在干净克隆上的完整验收输出（含上表数字，CI `web` 任务全部步骤）：

```
Done in 5.5s using pnpm v11.10.0
gen-check-web ok
keywords: 12 rules, 0 exceptions, no hits.
 Tasks:    52 successful, 52 total
 Tasks:    12 successful, 12 total
Unused files (123)
Unused dependencies (1)
Unused exports (133)
Unused exported types (81)
Unused exported enum members (4)
Duplicate exports (1)
Configuration hints (1)
 Tasks:    11 successful, 11 total
js: 505 files, 10270415 bytes
css: 3 files, 322306 bytes
fonts: 34 files, 6849620 bytes
other: 142 files, 9719426 bytes
largest chunk: assets/toolbar-eiWISGVR.js, 1821937 bytes
locale chunks: 40
       0
```

（最大 chunk 的文件名中间是内容哈希，每次构建不同；字节数与 Task 1 Step 2 的基线比较值一致。`Configuration hints (1)` 是 web 的 `+types/`，取决于本地是否跑过 react-router 的类型生成，属于已记录的 0–3 条浮动范围。）文件数只作记录，不作为门禁（M1 设计 7.6）。

## 6. 交接与延后项

| 项 | 内容 | 去向 |
|---|---|---|
| (a) 守卫规则的作用范围边界 | `storybook` 规则的 `files` 不覆盖 `Makefile` 和 `.github/**`；`deploy-files` 的路径规则锚定在 `^web/`；`storybook-files` 对它自己 `storybook-static/` 分支没有命中样本。这些都是 P1 就有的范围，本次修复轮补的是"样本覆盖 `files` 本身"，不是把范围本身补全——后续 Phase 往这几条规则的作用范围之外删东西时，不要过度信任它们会拦住 | 后续删除相关内容的 Phase（P2–P5），新增规则或扩大范围时按 spec 2.4 的写法补样本 |
| (b) `tools/` 门禁本身的格式检查覆盖不到的两处 | `web/packages/tailwind-config`、`web/packages/typescript-config` 没有 `check:format`/`check:lint`；也没有任何任务对仓库根目录自身的文件（如 `Makefile`、根 `package.json` 以外的杂项）做格式检查。这是 M0 就存在的缺口，与 Task 9A 补 `tools/` 门禁的理由相同（门禁要覆盖自己） | 待后续 Phase 评估是否值得单独补齐（未指定具体 Phase，记录在案） |
| (c) 两张加载动画 GIF 体积 | `logo-spinner-light*.gif`、`logo-spinner-dark*.gif` 各约 1 MB，且都在预渲染阶段被 `<img>` 直接引用、随首屏一起加载（Task 7 的修复让两张图始终都在 DOM 中） | P5（品牌与包名阶段，届时会重新处理这两张图） |
| (d) `helpers/index.ts` 的孤儿导出 | 仍导出 `ensureAPITrailingSlash`，web 没有导入它（Task 5 评审 Minor (a)，判为 P1 范围之外） | P3（knip 改为门禁清零未使用代码时一并处理） |
| 标准提醒 | 其他已有的工作区如果本地构建过，会残留 `web/packages/i18n/src/types/keys.generated.ts`（Task 9 之后不再被 `.gitignore` 挡住，会出现在 `git status`、关键词守卫、格式检查和 knip 的扫描范围中）。执行 `rm -f web/packages/i18n/src/types/keys.generated.ts` 清理；守卫会直接指出这个文件，不会悄悄放过 | 所有其他已有的工作区/克隆 |

## 7. spec 第 3 节的裁定

第 3 节列出的与 M1 设计的 12 处差异，加上控制者补的第 13 项，全部按 spec 原文采纳（2026-09-23）：

1. **`.prettierignore` 的替代**：不是直接删，而是删除的同时把其中真正起作用的三条改写进 `.oxfmtrc.json` 的 `ignorePatterns`——因为 oxfmt 会读取当前目录下的 `.prettierignore`，直接删会让格式检查扫到构建产物而失败。
2. **`fix:format` 的保留**：M1 设计说删 `fix*`，但 `fix:format` 是 README 写明的修格式入口，按设计自己的"README 写明的入口保留"原则留下，`api-client`、`e2e` 补上这个脚本。
3. **`tailwind-config` 的入口**：删掉指向不存在文件的 `main`，只由已有的 `exports` 声明——这个包本来就没有 JS 入口。
4. **企业版死文案多删一个命名空间**：`editor`（29 个键，0 引用），按与其余 7 个命名空间相同的方法核对得出。
5. **额外删除的死代码和依赖**：Storybook 或已删依赖的专属文件（propel 的 5 个只被 stories 导入的文件和 `ToastStatic`、两包的 PostCSS 配置与样式、`use-font-face-observer.d.ts` 等）及 web 中未被引用的 `.gitignore`、`manifest.json`、`public/favicon/`——删除之后都成了死代码或没有使用者的依赖。
6. **pnpm 工作区失效条目**：另列出并删除 7 条覆盖项、1 条 allowBuilds、7 条 minimumReleaseAgeExclude，用一次性脚本按六类逐条核对，不靠人工判断。
7. **锁文件的一处间接副作用**：`tsx` 的 `get-tsconfig` 4.13.7→4.14.3，是 pnpm 重新解析时向图中已有版本去重的结果，没有新增包或版本，`tsx` 只用于 i18n 的 `check:sync`，由 `make lint-web` 覆盖。
8. **`LogoSpinner` 一并改写**：只改 `HydrateFallback` 不够，`LogoSpinner` 自己也按主题输出不同的 `src`；副作用是 `dark-contrast` 主题下改为显示暗色动画。
9. **守卫的引入顺序提前、范围来自工作区快照**：在 Task 2（早于所有删除）加入而非设计建议的 services 之后；文件范围是跟踪的文件加上未跟踪、未忽略的文件,跳过工作区中已删除的文件。
10. **`make test-web` 是独立入口**：`make test` 仍只运行 Go 测试，持续集成的 `server` 任务调它、不需要 Node；P1 同时给 `toSupportedLanguage` 加了 10 个测试。
11. **跨命名空间冲突检查两种都保留**：重名和"叶子又是前缀"两种检查都保留，对两种语言各做一次，按命名空间比较。
12. **lint 上限的具体做法**：`tools/lint-cap.mjs` + 各包 `check:lint` 中的上限 + turbo 输入，六种缓存情况均已实测（附录 5.1）。
13. **控制者补充项**：`tools/` 下的两个门禁脚本本身不受 `make lint-web` 检查，新增 Task 9A（根 `check:lint`/`check:format`，`lint-web` 50→52），范围在 Task 9A 执行时进一步裁定为递归覆盖整个 `tools/`，唯一豁免 `tools/plane-schema/**`（第 2、4 节）。

提交本 spec 和计划时，M1 设计第 6、7.4、8、9 节已同步差异 1–12；评审阶段（本次修复轮）又把 7.4 按第 4 节的两条整分支评审裁定重新同步了一次（`3612074`）。
