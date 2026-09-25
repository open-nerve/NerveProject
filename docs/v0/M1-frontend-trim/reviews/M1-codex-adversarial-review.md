# M1 前端瘦身对抗性评审

## 1. 评审信息

| 项 | 实测值 |
|---|---|
| 对象 | main，c493255ee8c7f418567bacbd8f025de6adbf18e2；范围 0e43771..c493255 |
| 收尾状态 | first-parent 只有 P1 57fccb6、P2 6d9692b、P3 9437a6e、P4 96d8c1d、P5 c493255 五个合并；没有 Merge M1 closeout。M1-design.md:676 也记为“未开始”。本报告把收尾交付物及依赖收尾的标准记为“未完成（收尾进行中）”，不列为新发现 |
| 日期、评审者 | 2026-09-25；Codex（GPT-6） |
| 运行环境 | macOS Darwin 25.6.0 arm64；系统 Go 1.26.2，server/ 中由 go.mod 自动选择 Go 1.27.1；Node 24.15.0；pnpm 11.10.0；Docker client/server 29.7.2 |
| 范围核实 | git diff --stat 0e43771 HEAD 为 3165 文件、+37376/−234756 行；server/、api/ 零改动，e2e/ 只有 package.json 5 行 P1 工具链变更 |
| 已运行，均退出 0 | pnpm install --frozen-lockfile；make gen-check；make lint-web（48 规则、5 例外；52/52 任务）；make lint（Go 0 issue）；make knip；make test-web 与强制无缓存的 turbo test（16/16 任务、77 个测试）；make build-web；make build；强制无缓存的 turbo build（11/11 任务）；make test；Playwright Chromium 安装；make e2e（S1–S4，5/5）；107 个独立构建图片原尺寸目视；propel 25 个插画及 122 个静态图标渲染目视 |
| 特殊运行结果 | make web-dev 能在 3000 端口回答页面，开发浏览器有 22 条 sanitize-html 依赖引入的浏览器兼容警告；因未另起 Go 开发后端，/api/instances/ 被 Vite 代理后返回 502。该进程已停止。缓存构建的临时陈旧文件试验见 Important 1，试验文件与嵌入资源均已删除并重新构建 |
| 未独立运行 | P1–P5 review 附录的整套浏览器脚本及 0e43771 基线反向脚本：它们依赖当时仓库外的临时脚本、假数据与基点构建，当前未直接复用；本轮细读了 P2、P3、P5 三份脚本，并独立做数学复算、真实构建和端到端测试。未在 Linux 主机上重跑门禁。未因 Docker 或网络受阻跳过上述 make 命令 |

证据口径：文件位置为仓库相对路径加行号；“已复现”只指本轮真实运行的现象；未运行完整产品场景的结论标为“读代码确认”或“推断”。首次 make build-web 命中 Turbo 缓存时，旧目录有 689 文件、21,848,544 字节；强制无缓存重建后为 534 文件、17,306,349 字节，恰与 P5 review:42-47 的分类合计一致。前者不能用于 M1 体积结论。

## 2. 总体结论

**修完列出的 Critical 后可以开始 M2。** 这以既定的 M1 收尾完成并验收为前提，收尾未合并属于当前里程碑状态而非新发现。独立运行的安装、生成、静态检查、knip、前后端测试、构建及 S1–S4 均通过；删除边界、原生路由和可见品牌也有较完整的代码落点。但编辑器把来自剪贴板的自定义 HTML 在清洗前送入离屏 innerHTML，Chromium 中可触发脚本执行（Critical 1）。这条入口可由其他网站在用户复制时写入，应在 M2 开始前封闭。

“M1 已全部完成”目前不成立，因为收尾尚未合并。另有缓存构建携带陈旧资源、取消工作项时进度定义分叉、可见截图含已删功能、发布合规交接缺项，分别需要在后续指定时点解决。现有 CI 全绿不能替代这些判断：S2 只加载首页和深层地址，P2 的进度探测假数据把取消数设为 0。

## 3. 各领域评估

| 领域 | 评分 / 5 | 主要依据 | 最大风险 |
|---|---:|---|---|
| 范围闭环 | 3 | 五个 Phase 已合并；server/api 零 diff；收尾未合并 | 收尾验收与交接未关闭 |
| 删除的正确性（保留功能完整） | 3 | 十类删除链路抽查、77 个测试；P2 进度与图片例外 | 取消项进度错误、旧功能截图仍可见 |
| 路由 | 4 | 原生 route 表、navigation.test、恶意 next_path 输入与 S2 深层路径 | 服务端 next_path 语义留给 M2 |
| 安全 | 2 | 自定义剪贴板 XSS 已复现；其余窗口风险已记录待收尾 | 跨站复制后在 Nerve 源执行脚本 |
| 合规与品牌 | 3 | 逐看构建中 107 个独立图片及 propel 25 个插画、122 个静态图标、品牌变量与版权头；AGPL 发布清单仍缺 | 对应源码与法律通知的发布验收 |
| 质量门禁 | 4 | 关键词、lint、knip、vitest 变异均有反向失败 | 已知格式范围与 knip 通配导出盲区 |
| 测试 | 3 | 77 个单测、5 个 E2E 通过，P2/P3/P5 附录细读 | 保留功能矩阵的场景数据过窄 |
| 构建与工程 | 3 | 干净构建体积与 P5 一致，构建和 e2e 通过 | Turbo 命中时遗留文件被嵌入 |
| 代码质量 | 3 | 大规模删除后仍可检查/构建；少量公开孤儿与旧注释 | 删除边界随 M2–M8 改写漂移 |
| 文档 | 3 | 各 Phase 的证据链详尽；收尾缺席与编辑器边界矛盾 | 已完成的叙述被当作最终验收 |

## 4. 新发现

### Critical 1：自定义剪贴板 HTML 在清洗前触发脚本

- 类别：安全。位置：web/packages/editor/src/props.ts:38-46；web/packages/editor/src/helpers/paste-asset.ts:10-12,28-29；web/packages/editor/src/hooks/use-editor.ts:52-64。
- 证据：CoreEditorProps 的 handlePaste 从 text/nerve-editor-html 取原始字符串并调用 processAssetDuplication；后者执行 tempDiv.innerHTML = htmlContent，且第二次处理还会再次赋值。Chromium 实验：网页的 copy 事件用 clipboardData.setData 写入自定义类型，跨来源页面的 paste 事件仍读到完整值；将值设为离屏 div.innerHTML 后，图片加载失败触发 onerror，div.isConnected 为 false 时也执行。最小载荷为带无效 src 和 onerror 的 img 标签。仓库内没有 Content-Security-Policy 配置命中。普通 text/html 粘贴不走此分支；旧基线有相同 sink，但 M1 改动并验收了编辑器及剪贴板类型，现有 review 没记录这一风险。
- 可信度：已复现剪贴板传递和离屏 DOM 执行；Nerve 编辑器接线由代码确认，未用完整业务页面重放攻击。
- 影响：用户从攻击者页面复制内容并粘贴到 Nerve 富文本编辑器时，脚本可在 Nerve 源运行，访问页面数据并发起同源请求。用户交互是利用前提；不需要攻击者控制 Nerve 服务。
- 处理建议（权衡）：
  1. **推荐，M2 开始前**：先对自定义 HTML 做适合编辑器节点的白名单清洗（事件属性、脚本与危险 URL 一律去掉），用 inert template 处理图片节点，再交给 pasteHTML；加跨来源自定义 MIME 的浏览器回归。收益是保留本地复制与图片复制；代价是维护一份很小的编辑器允许节点规则，并核对自定义节点属性；风险是过窄规则可能丢格式，需要以现有节点样本验收。这是封闭实测执行入口所必需的复杂度。
  2. 删除自定义 MIME 的粘贴分支，退回普通 HTML/文本粘贴。收益是改动小、封口快；代价是同一编辑器之间的资源复制语义会退化；风险是图片复制行为变化，需明确接受并回归。
  3. 不处理：收益是零即时改动；代价是每次接入富文本时承担额外安全核查；风险是跨来源脚本执行入口持续存在，不可接受。

### Important 1：缓存命中的前端构建会把陈旧资源带进 Go 二进制

- 类别：工程。位置：web/apps/web/package.json:9；turbo.json:16-18；Makefile:126-134；server/internal/platform/webui/embed.go:10-15；server/internal/platform/webui/handler.go:58-85。
- 证据：本轮在主目录进行的试验：初始 build/client 有 689 文件、21,848,544 字节；强制构建清理后是 534 文件、17,306,349 字节。随后只在被忽略的 build/client/assets/ 写入 52 字节的 m1-stale-asset-probe.txt，执行 make build-web，输出 11/11 cached、退出 0，文件仍在；执行 make build 后同名文件出现在 webui/dist/assets/。Handler 对已有 assets/ 文件直接 serveFile，且 Go 使用 go:embed all:dist。清除试验文件并再次 make build 后，两处均不存在；写本报告前 git status 为空。
- 可信度：已复现。旧的 155 个文件具体来源未逐个鉴别；不能断言其中仍有 Plane Logo，但可以断言构建会携带无引用或旧版本的资源。
- 影响：在同一工作区反复构建或按 README 用 make build 打包时，旧 JS、图片或其他生成资源可随新版本嵌入并被 HTTP 直接访问；P5 在干净克隆的品牌/体积结果不能覆盖这种构建路径。全新 CI 工作区风险较低。
- 处理建议（权衡）：
  1. **推荐，并入 M2，任何对外部署前**：在 make build-web 调用 Turbo 前清空仅限 web/apps/web/build/client 的生成目录；缓存恢复和真实构建都从空目录开始。收益是部署产物与当前缓存键一致；代价约一条受限清理命令和缓存复核；风险是 Turbo 缓存恢复若不完整会漏文件，须以冷/热两次构建确认。
  2. 只在 make build 复制前按构建清单筛选文件。收益是可避免旧资源；代价是维护清单和相对路径逻辑；风险是新资源未列入清单而漏打包，比生成目录清理复杂。
  3. 不处理：收益是零构建脚本变更；代价是部署时需人工辨别旧资源；风险是本地/增量部署继续把旧资源嵌入，对于已发生的 155 文件差异与品牌目标不合适。

### Important 2：当前迭代进度把取消项同时计入分子并排除分母

- 类别：Bug、删除。位置：web/apps/web/core/components/cycles/active-cycle/progress.tsx:36-38,54-68；web/apps/web/core/components/cycles/active-cycle/root.tsx:79-84；web/packages/utils/src/cycle.ts:73-104；web/packages/utils/src/progress.test.ts:26-28；docs/v0/M1-frontend-trim/reviews/P2-trim-content-review.md:1819。
- 证据：当前迭代卡片使用 (completed+cancelled)/(total−cancelled)，而 P2 改写并测试的共享进度定义是 completed/(total−cancelled)。代入 total=10、completed=3、cancelled=2，卡片计算 62.5% 且写“5/8 closed”，共享函数为 38%；当完成 8、取消 2 时传给 LinearProgress 的值为 125%，propel 组件会钳制到 100%，但页面文案写“10/8 closed”。P2 浏览器假数据的 cancelled_issues 是 0，没走到这个分支。卡片公式在 0e43771 已存在；本发现针对 M1 改写同域定义后未核对保留页面的承诺，不把旧公式单独当作 M1 新增 bug。
- 可信度：读代码确认，并独立复算；当前业务页因接口尚未对接，未用真实服务端渲染。
- 影响：M6 实现迭代时，进度条、列表及文字互相矛盾；组件虽钳制超过 100% 的输入，用户仍会看到不可能的“10/8 closed”并误判完成度。
- 处理建议（权衡）：
  1. **推荐，M6 迭代进度验收前**：当前迭代卡片复用 calculateCycleProgress，并让文字使用同一个“完成 / 非取消总数”口径；补 cancelled>0 和全完成样本。收益是只有一份进度定义；代价是改一处卡片及测试；风险是若产品另想把“取消”算已关闭需重新定口径，但 M1-design.md:158-168 和现有测试已给出排除口径。
  2. 把共享函数改成“完成或取消均算关闭”并统一所有页面。收益是保留当前卡片的“关闭”意图；代价是改设计、测试、快照和 M6 接口约定；风险是与“取消项排除”裁定冲突，回归范围大。
  3. 不处理：收益是省去本轮改动，代价是 M6 后期再迁移两个进度口径；风险是错误完成度随新接口固化。

### Important 3：公开部署的 AGPL 验收尚未覆盖三个具体义务

- 类别：合规、闭环。位置：LICENSE:91-97,196-221,540-550；README.md:98-102；docs/v0/M8-open-release/handoffs/M1-P5-brand.md:8-31；web/apps/web/core/components/workspace/sidebar/help-section/root.tsx:45。
- 证据：[GNU AGPL-3.0 正文](https://www.gnu.org/licenses/agpl.en.html)第 5(a) 款要求修改说明及相关日期，第 5(d) 款规定交互界面的适当法律声明并附原程序界面的例外，第 13 条要求网络交互用户能取得对应源码。README 有版权和许可证链接，前端改动清单记录了修改类别，帮助菜单打开仓库；但抽查的 1395 个 M1 修改过的 web 文件没有单独的“修改及日期”标识，当前文档也未说明集中声明是否足够。M8 handoff 明确覆盖 propel 对应源码、版本和 OG 地址，却未把第 5(a)、5(d)、13 条的验收、原版例外判定及运行版本与源码的对应关系列为关闭条件。propel 的源码缺口本身是已知项，不在此重复报告为新发现。
- 可信度：文档与代码覆盖缺口已确认；具体法定义务的适用方式、特别是 5(d) 的原版界面例外，属于法律解释，需项目负责人核定。本轮不声称本地未发布提交已发生侵权。
- 影响：若 M8 之前做对外演示或部署，单靠仓库链接不能证明运行版本及 propel 的 Corresponding Source 都可取得；发布时也可能漏掉所需的修改说明和法律声明。
- 处理建议（权衡）：
  1. **推荐，任何外部网络部署前且最迟 M8**：给 M8 handoff 增加 5(a)、5(d)、13 三条逐项验收；由负责人核定集中修改说明是否足够、原版是否触发 5(d) 例外，并验证页面的源码入口指向运行提交及完整对应源码。收益是可执行的发布检查；代价是一次来源和法律核定；风险是核定后仍可能需要补界面或源码材料，但比预先给数千文件机械加头更可控。
  2. 现在逐文件加修改日期和界面法律菜单。收益是可能满足最严格解释；代价是大规模噪声改动；风险是 5(d) 例外与对应源码尚未核定，容易做错且不能单靠文件头解决第 13 条。
  3. 不处理：收益是短期省事；代价是发布前仍须临时核来源和通知；风险是对外网络交互或分发时无法给出完整合规证据，不可作为发布决定。

### Important 4：保留页面的图片仍展示已删除的估算和数据分析入口

- 类别：删除、闭环。位置：web/apps/web/app/assets/empty-state/disabled-feature/intake-dark.webp；同目录 intake-light.webp；web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx:15-16,42-52；web/apps/web/core/components/empty-state/detailed-empty-state-root.tsx:93；web/apps/web/app/assets/onboarding/cycles.webp；web/apps/web/core/components/onboarding/tour/root.tsx:13,48-53,141。
- 证据：对构建里的 intake-dark、intake-light（源图均为 2631×1200）逐张按原始像素目视，工作项属性明确印着 “Estimate 2 Months”，还有 PL-145/PL-146 示例编号。项目收集箱关闭时，page.tsx:45-52 直接把这些图传给 DetailedEmptyState，用户会看到。导览的 cycles.webp（1920×1080）顶部还印着 “Analytics” 按钮，而 P2 已删除数据分析入口。两张收集箱图与导览图按保留功能命名，所以关键词守卫和 P2 review:141-143 按“被删功能命名的图片”清理的方法均看不到这些像素。P5 浏览器脚本的图像组只比较八张已发现品牌图的哈希，并未检验这里的内容。
- 可信度：已按原始像素目视构建资源并读代码确认挂载条件；未在真实项目数据下拍摄该禁用页。
- 影响：禁用收集箱页及导览向用户展示不存在的估算属性和数据分析按钮，删除承诺在可见图片层没有闭环。用户可能按图片寻找功能，M3/M6 的界面验收也会被旧截图误导。PL 编号本身不足以证明 Plane 品牌仍在，不把它报作独立品牌缺陷。
- 处理建议（权衡）：
  1. **推荐，M1 收尾且 M2 开始前**：对仍挂载的图逐张移除已删功能的可见字段与按钮，并核对暗色、亮色及导览裁切；收益是保留有用的示意图并兑现删除边界，代价是少量位图重制，风险是修改图中文字/阴影时产生痕迹，需要原尺寸目视。
  2. 用不含旧界面的简单自有插画替换这三张图；收益是从根上避免旧产品 UI 再次漏入，代价是设计和布局调整较多，风险是空状态信息量降低。
  3. 不处理；收益是零即时改动，代价是后续每次复核都要解释旧图，风险是用户持续看到不存在的功能，不符合 M1 的可见删除目标。

### Minor 1：P1 明确移交 P3 的公开孤儿导出没有收口

- 类别：闭环、质量。位置：docs/v0/M1-frontend-trim/reviews/P1-web-hygiene-review.md:390；web/packages/services/src/helpers/url.ts:22,36-38；web/packages/services/src/helpers/index.ts:7；web/packages/services/src/index.ts:9。
- 证据：P1 review 要求 P3 清理 ensureAPITrailingSlash 的孤儿导出；P3 spec、plan、review 没有另行裁定。当前仅服务包自身实现和测试引用该符号，两个 barrel 仍公开它；函数本身被 normalizeAPIRequestURL 调用，不能删。可信度：读代码确认。
- 影响：公开 API 面比实际使用面大，给 M2/M5 接口重写增加一个不必要的兼容期待；运行功能不受影响。
- 处理建议（权衡）：① **推荐，收尾或并入 M2**：取消公开导出、保留内部函数；收益是 API 面与使用面一致，代价一处 barrel 变更，风险是外部未知调用者可能引用，但 M1 尚未公开承诺此符号。② 等 M5 删除整个 services 包时一并消失；收益是零即时改动，代价是交接延后，风险是 M2 将它误当稳定 API。③ 不处理；收益是避免一行变更，代价是持续保留孤儿导出，风险是未来调用方误用。

### Minor 2：编辑器“附件节点保留”的设计叙述与实际边界相反

- 类别：文档。位置：docs/v0/M1-frontend-trim/M1-design.md:131-136；docs/v0/M5-files/handoffs/M1-P3-trim-platform.md:12-16；web/packages/editor/src/plugins/drop.ts:35,63。
- 证据：设计 3.1 写“保留图片和附件的节点”；M5 handoff 说明社区版没有附件节点，P3 已删附件命令；当前编辑器仍有附件 MIME 和拖粘被视为“已处理”的分支。可信度：读代码确认，并与 0e43771 的附件命令类型作过对照。
- 影响：M5 可能误以为只需对接已有附件节点，或把“工作项附件区”与“编辑器内附件”混为一谈。非图片拖粘被吞是 handoff 已知问题，不作为新 bug 重报。
- 处理建议（权衡）：① **推荐，收尾**：把设计 3.1 改成“图片节点保留；附件由工作项附件区承担，编辑器内是否支持由 M5 决定”；收益是与 handoff 一致，代价仅文档编辑，风险很低。② 现在实现编辑器附件节点；收益是满足设计字面要求，代价是超出 M1 的接口和编辑器工作，风险是 M5 再返工。③ 不处理；收益是省文档时间，代价是 M5 继续核来源，风险是错误估算附件工作。

## 5. 闭环核对表

### 5.1 M1-design 0.1 的五个目标与 0.2 范围

| 项 | 判定 | 独立证据 |
|---|---|---|
| 0.1-1 已删功能、企业残留和死代码删到各层 | 部分成立 | 下表十类抽查主体删除；Important 4 是仍在使用的图中的旧功能，另有收尾认领的 133 张无引用图、482 个疑似无引用文案键（P5 review:40-47,1095,1100），不是最终闭环 |
| 0.1-2 保留功能完整并核过 7.5 | 部分成立 | 多数保留入口有代码和旧脚本落点；Important 2 证明取消项进度未核到，Critical 1 是编辑器保留路径的安全缺口，其他矩阵行也有未独立重跑的场景 |
| 0.1-3 Next 垫片删除、原生 React Router | 成立 | app/routes/core.ts:7-20 原生 route/layout/index；rg 对 compat/next、next/navigation、useAppRouter、ensureTrailingSlash 零命中；navigation.test.ts 存在，强制构建成功 |
| 0.1-4 品牌、包名与两语言 | 成立（本轮检查范围） | app/root.tsx:61-96 用 SITE_NAME；constants/src/metadata.ts:7-9 为 Nerve；107 个独立图片、propel 25 个插画和 122 个静态图标目视未见 Plane 标志；en 与 zh-CN 各 17 JSON、1599 叶键，pnpm 工作区内包名为 @nerve/* |
| 0.1-5 类型、knip、lint、关键词、构建 | 成立 | make lint、make knip、强制测试、强制构建均退出 0；tools/keywords.mjs 输出 48 rules、5 exceptions、no hits |
| 0.2 删除与兼容层、品牌、语言、工具链纳入范围 | 部分成立 | P1–P5 的代码落点存在；收尾未合并，范围终态尚不成立 |
| 0.2 接口对接、Bearer、services 整包与后端留后续 M | 成立 | server/api 零 diff；services 仍在；auth-forms/password.tsx 保留旧提交协议，M2 handoff 接收 |
| 0.2 新 E2E 故事与 oxlint 清零留后续 | 成立 | CI 仍跑 S1–S4；当前 lint 上限合计 711；清零计划属于未开始的收尾 |

### 5.2 删除深度抽查：P2 与 P3 的十类功能

检查方式：在当前 web/ 的路由、组件、store、service、类型、常量、命令/快捷键/导览、本地存储和文案中搜下列确切符号；对照 git diff --name-status 0e43771 HEAD、tools/keywords.json 对应规则和锁文件。关键词“零命中”是守卫范围内的结果，不把它误当成图片或未被规则覆盖的语义已经消失。保留的地址或词在表中写明。

| 功能 | 判定 | 跨层证据与残留判断 |
|---|---|---|
| P2 文档页/协作 | 成立 | 原 pages 路由、page store/service/type、Yjs 等依赖删除；对 description_binary、y-prosemirror、page_view 的 web 源码搜索零；tools/keywords.json:519 规则覆盖路径/内容，命令与导览的 pages 分支已删。编辑器图片、提及与描述历史仍在；附件节点的文档矛盾见 Minor 2 |
| P2 估算 | 部分成立 | estimates 设置路由、字段与交互 UI、点数切换及依赖删除；web 文本源码 estimat/估算 零未登记命中，规则在 tools/keywords.json:405；禁用收集箱图像仍印 Estimate 2 Months（Important 4），迭代数量口径另见 Important 2 |
| P2 甘特/时间线 | 成立 | 路由/布局、EIssueLayoutTypes.GANTT 与组件服务删除；gantt 和 issue-dates 源码零命中，规则在 tools/keywords.json:438；propel 的 view_timeline 是材料图标名，不是布局入口 |
| P2 数据分析 | 部分成立 | analytics 路由、侧栏、store、文案图表删除；tools/keywords.json:288；导览 cycles.webp 仍印 Analytics 按钮（Important 4）；cycle.service.ts:13 的 /analytics?type=issues 是保留的 Plane 进度接口，到 M6 换掉，utils/tlds.ts:59 是顶级域名数据 |
| P3 公开发布 | 成立 | apps/space 未迁入；publish 路由/组件/类型及评论 access UI 删除；tools/keywords.json:860,906 查确切符号；草稿“发布为工作项”的 publish 语义保留。新手引导截图里的 Public 是旧示例项目可见性文字，不能凭该词认定公开发布功能仍可用 |
| P3 管理后台 | 成立 | apps/admin 未迁入；god-mode、instance-admin、is_setup_done 等网页路径/分支零命中；实例请求失败的 MaintenanceView 保留，types/src/instance/base.ts:11-16 只留四字段 |
| P3 Epic/工作项类型 | 成立 | 工作项 store 的 Epic 多态分支、类型切换/组件和入口删除；tools/keywords.json:934 有规则；描述历史仍用 work-items 地址，是 M4 已记交接，不是 Epic 残留 |
| P3 工作项批量操作/多选 | 成立 | MultipleSelectStore、useMultipleSelect、selectionHelpers 在 web 零命中；tools/keywords.json:1028；project-member.store.ts:71 的 bulkAddMembersToProject 是项目成员添加，不是工作项多选，旧注释可改但不作为删除失败 |
| P3 认证多余路径 | 成立（按 M1 边界） | OAuth、验证码、找回/重置/设置密码界面删；邮箱密码和修改密码留；M2 接手传输方式。关闭注册的直达 /sign-up 行为在基线也存在，本轮只把 P3 浏览器脚本覆盖不足记在 5.4 |
| P3 AI/Unsplash/遥测 | 成立 | 关键词对应规则、package 删除与封面上传入口核对；Unsplash/遥测 SDK 不在构建主机清单；callout 的 jsDelivr emoji 图片不属于 Unsplash，见第 7 节 |

额外搜索没有发现新引入的功能开关、空实现 hook 或恒 false 的工作项多选链。结构退化并未最终清零：P4/P5 review 已把从不渲染的应用栏、单项菜单、死成员和 482 个疑似死键移交收尾，当前未开始。本轮没有把这些已记录事项重复列为新发现。锁文件中 vitest 4.1.11、jsdom 30.1.1、@makeplane/propel 0.3.0 解析为精确版本（pnpm-lock.yaml:5808,4225,1672）；allowBuilds 没扩大，pnpm install --frozen-lockfile 通过。

### 5.3 第 3 节十五条设计裁定

| 裁定 | 判定 | 证据 |
|---|---|---|
| 3.1 编辑器只去协作、保留本地能力 | 部分成立 | editor/src 无 Yjs/协作扩展，extensions.test.ts 保证图片/提及等；附件节点叙述与 M5 handoff 冲突（Minor 2），自定义 HTML 粘贴有 Critical 1 |
| 3.2 个人主页转分配给他的 | 成立 | app/routes/redirects/core/profile-index.tsx:12-19；core/components/profile/use-profile-member.ts:13-45 处理成员加载、缺失和失败；profile/[userId]/layout.tsx:31-34,55-64 处理访客权限 |
| 3.3 固定侧栏导航 | 成立 | constants/src/workspace.ts:146,168 固定列表，constants/src/navigation.test.ts:19-45 校验顺序 |
| 3.4 进度按工作项数 | 部分成立 | 点数切换已删，数量函数与乐观更新测试存在；当前迭代保留卡片在取消项上不一致（Important 2） |
| 3.5 services 只修剪 | 成立 | web/packages/services 保留令牌、地址、文件工具；整包移交 M5；公开孤儿导出是 Minor 1 |
| 3.6 认证只删界面、传输交 M2 | 成立 | auth-forms/password.tsx:120-142；M2-auth/handoffs/M1-P3-trim-platform.md:12-34 |
| 3.7 实例保留四字段及维护页 | 成立 | types/src/instance/base.ts:11-16；instance-wrapper.tsx:39 |
| 3.8 评论可见范围随公开发布删除 | 成立 | 原 access UI/类型无调用，M4-issue-core/handoffs/M1-P3-trim-platform.md:16 接手接口字段 |
| 3.9 多选整条删除 | 成立 | 上表的三组确切符号零命中，通知/普通下拉不属于该功能 |
| 3.10 propel 作为锁定依赖 | 成立（源码待 M8） | pnpm-workspace.yaml:22 锁定 0.3.0；M8-open-release/handoffs/M1-P5-brand.md:10-19 记来源缺口 |
| 3.11 按 M 交接字段 | 部分成立 | M2–M8 共有 17 份 M1 handoff，均 open；例如 M4-issue-core/handoffs/M1-P4-router-native.md:10-18 给出两项决策，但没有可验证的关闭条件，最终核对应在收尾 |
| 3.12 删旧地址重定向 | 成立 | core.ts 路由中无旧 redirect 模块；navigation.test.ts 扫内部目标；401 丢 query/hash 是已知 M2 交接 |
| 3.13 邮箱修改留给 M2 决定 | 成立 | 验证码界面已删，M2-auth/handoffs/M1-P3-trim-platform.md:21 记录决策 |
| 3.14 首页保留最近访问 | 成立 | home/root.tsx:40-46；home-body.tsx:24-30 有数据与空态分支 |
| 3.15 邀请链接保留至 M3 | 成立 | app/routes/core.ts:30-33 保留 workspace-invitations；M3 handoff 有关闭决策 |

### 5.4 第 9 节 Phase、9.7 交接及 7.5 行为矩阵

| 第 9 节交付物及验收 | 判定 | 证据 |
|---|---|---|
| P1 工具链、遗留物、多语言、lint 上限 | 部分成立 | 合并 57fccb6；P1 阶段为 12 规则/0 例外，当前为 48/5；两语言各 1599 键、lint 上限核对及开发监视可运行；格式范围与构建兼容警告仍有已知限制 |
| P2 内容类删除、进度与编辑器、handoff | 部分成立 | 合并 6d9692b；删除主体及 tests 落地，但 7.5 的取消项进度分支不成立，保留图片仍显示估算与 Analytics（Important 4） |
| P3 平台删除、knip 门禁、handoff | 部分成立 | 合并 9437a6e；删除主体和 knip 门禁成立，CI web job:68-72 调用；P1 明确交给 P3 的孤儿导出仍在（Minor 1），认证脚本覆盖盲点见下 |
| P4 原生路由、无环境注入、README | 成立 | 合并 96d8c1d；app/routes/core.ts、navigation.test.ts；web 源码仅 import.meta.env.DEV/PROD，非环境配置；深层路径 S2 通过 |
| P5 品牌、包名、来源记录 | 成立（本轮图片检查范围） | 合并 c493255；构建 107 个独立图片原尺寸目视、propel 25 个插画及 122 个静态图标渲染目视、SITE_NAME、en/zh-CN 文案和锁定记录 |
| 收尾关键词例外、死键、锁、体积、oxlint 计划、文档 | 未完成（收尾进行中） | first-parent 无 closeout；M1-design.md:623-629,676；closeout spec/plan/review 当前不存在 |

| 9.7 两份 M0 交接事项 | 判定 | 证据 |
|---|---|---|
| 部署遗留、serve、sw/workbox、Turbo 死任务 | 成立 | M0-P5-frontend-trim-notes.md:1-18 标 closed，P1 删除目录与相关依赖；构建缓存陈旧文件另见 Important 1 |
| catalog、overrides、allowBuilds 锁核 | 部分成立 | P1–P5 review 各有解析核对；最终一次属于未合并收尾；本轮 frozen install 通过且 allowBuilds 未扩大 |
| dotenv、define process.env、README | 成立 | P4 删除；web 源码 grep 无 process.env/dotenv；README.md:60-65 描述同源 |
| lint 基线与清零计划 | 部分成立 | 当前合计 711、自动核对有效；按规则清零计划属于收尾 |
| React #418、tailwind-config 模块、vite-tsconfig-paths、web-dev | 部分成立 | P1 修复与浏览器探测记录；本轮 web-dev 启动并渲染，未重做 #418 的基点反向探测 |
| 结尾斜杠 | 成立 | 内部目标导航测试、Go webui/handler.go:54-65 SPA 回退与 S2 深层路径；接口尾斜杠不在该裁定范围 |
| knip 配置提示与 i18n ignoreUnresolved | 成立 | make knip --treat-config-hints-as-errors 退出 0 |
| api-client/e2e 上限 0 | 成立 | 当前 package.json 脚本与 lint-cap.mjs 核对，make lint-web 通过 |
| S2 不断言文字、knip 路径不随包名改 | 成立 | e2e/stories/smoke/s2-web-app.spec.ts；knip.jsonc 保持路径配置，P5 未改变 S2 口径 |

| 7.5 行 | 判定 | 本轮核对与脚本有效性 |
|---|---|---|
| P1 语言回退、键、#418、包源码监视 | 部分成立 | 两语言各 1599 键、make lint-web 通过；web-dev 运行了，但未改包符号做热更新，也未重跑 #418 视觉反向探测 |
| P2 首页、个人主页、侧栏 | 部分成立 | 挂载链与四状态代码可定位；P2 脚本附录写了有数据/空态断言，本轮未完整重跑 |
| P2 编辑器输入/保存/撤销/提及/历史/图片 | 部分成立 | editor tests 16 个与 P2 脚本覆盖部分本地能力；自定义粘贴有 Critical 1，附件边界见 Minor 2 |
| P2 进度 | 不成立 | Important 2；P2 review:1819 假数据取消数为 0，无法证明取消分支 |
| P2/P3 动态、通知、store 上下文、四布局 | 部分成立 | P2 review:2147-2210 的探测确有请求及可见标题断言、基点反向数据会失败；use-issues.ts:75-113 对项目/迭代/模块/主页/归档分派；未跑全部业务数据 |
| P3 认证与实例 | 部分成立 | P3 review:378-505 同时拦 /api 与 /auth，确实断言表单体与导航；关闭注册只查首页隐藏注册链接（:576-583），未直访 /sign-up；成功 next_path 是桩固定 302，不能证明后端规则 |
| P4 路由 | 部分成立 | next-path 恶意输入与导航测试通过，S2 深层路径通过；完整 P4 九段脚本本轮未重跑 |
| P5 品牌、PAT、Webhook | 部分成立 | P5 脚本 DOM 文字/属性和基点反向对照有效，C 组图片哈希差异本身不足以证明无 Logo；本轮独立目视了构建图片；PAT/Webhook 页只核了路由落点，未登录操作 |

三份脚本的可证与不可证之处另记：P2 的通知/store 探测确有断言，但取消数 0；P3 的 POST/CSRF/next_path 字段断言确实触发，后端目标由桩固定；P5 的 177 项包含 DOM 文案与品牌属性，C 组只比较“图像字节不同于 Plane 原图”，一像素变化也会过，所以本轮另作逐图目视。基点反向失败的记录见 P3 review:1443-1452、P5 review:1009-1078；本轮未重放，不能把报告里的反向结果说成本轮实测。

### 5.5 第 11 节九条完成标准

| 条目 | 判定 | 证据 |
|---|---|---|
| 1. P1–P5 加收尾均有 spec/plan/review | 未完成（收尾进行中） | 前五份俱全；收尾三份不存在 |
| 2. 类型、knip、lint、格式、vitest CI | 成立 | 本轮 make lint、knip、强制 test 退出 0；CI:60-72 均调用 |
| 3. 守卫跨 M 例外与无死键 | 未完成（收尾进行中） | 守卫 48/5 无未登记命中；死键最终判定及 M9 例外复核属收尾 |
| 4. 7.5 逐行核对 | 不成立 | P2 取消项分支未覆盖且结果矛盾；其他行部分独立验证 |
| 5. make build 与 S1–S4 | 成立 | 本轮 make build 退出 0；make e2e 5/5 |
| 6. 双语言、Nerve 品牌与包名 | 成立（本轮检查范围） | 各 17 JSON、构建图联系表、应用标题 Nerve、pnpm 工作区包名；propel 是许可的第三方名 |
| 7. oxlint 清零计划、体积对比 | 未完成（收尾进行中） | P5 只有阶段数字；最终对比/计划未有 closeout review |
| 8. M1 handoff closed，后续 M handoff 有接收项 | 未完成（收尾进行中） | M0→M1 两份 front matter 为 closed；后续 17 份 open，最终关闭条件核对属于收尾 |
| 9. 前端清单与总体设计状态同步 | 未完成（收尾进行中） | frontend-changes.md 第二、四节已有 P1–P5 内容；v0-design.md:667 的 M1 仍“进行中”，与当前真实状态一致 |

### 5.6 设计对抗评审第 10 节“已吸收”事项与 M1 自己的 handoff

| 原项 | 判定 | 代码核对 |
|---|---|---|
| C1 旧跳转入口 | 成立 | P4 删旧 redirect 后 navigation.test.ts 扫正式内部地址 |
| I1 运行时矩阵及 /auth 拦截 | 部分成立 | P3 脚本拦 /auth 与 /api，但 P2 取消进度及 P3 禁注册直达存在盲区 |
| I2 首页最近访问 | 成立 | home/root.tsx 与 home-body.tsx 直接挂载 |
| I3 多选整链 | 成立 | MultipleSelectStore/useMultipleSelect/selectionHelpers 零命中 |
| I4 JS 正则守卫/精确例外 | 成立 | tools/keywords.mjs 使用 RegExp；本轮错误正则、count、stale、until 变异均以非零退出 |
| I5 实例字段与跨 M | 部分成立 | types/src/instance/base.ts:11-16 和 M2–M8 handoff 已有；关闭条件待收尾 |
| I6 斜杠与当前项 | 部分成立 | 路由测试与 S2 通过；完整菜单/通知/后退脚本本轮未重跑 |
| M1 完成条件与计数口径 | 成立 | M1-design.md:654-665 仍为未勾状态；当前上限 711、测试 77，未误写为 oxlint 清零 |
| M2 编辑器 AI 分期 | 成立 | P2 editor 包协作/AI 删除，P3 应用 AI 删除；关键词规则生效 |
| S1 来源、版权、体积 | 部分成立 | propel 来源与新文件头有记录；最终体积对比属于未合并收尾 |
| 10.2 编辑器本地 UniqueID、Markdown 历史复制、图片/提及 | 部分成立 | editor/src/extensions/extensions.ts:120 保留 UniqueID，helpers/editor-ref.ts:76 和 description-versions/modal.tsx:67-69 保留复制调用，图片/提及扩展测试通过；自定义粘贴有 Critical 1 |
| 10.2 成员状态、CSRF 时机、评论 access、邮箱决策 | 成立（交接边界） | use-profile-member.ts:13-45 与 profile/[userId]/layout.tsx:31-64 分处理加载/失败和访客；password.tsx:141 保留 CSRF 至 M2；M4 评论 access 和 M2 改邮箱在 handoff 中 |
| 10.2 其余 3.3、3.4、3.5、3.10、3.11、3.12 的改写 | 部分成立 | 固定导航、propel 来源、删除旧跳转和分领域 handoff 见 5.3；3.4 的取消数分支为 Important 2，handoff 关闭条件待收尾 |
| M1 handoff：M0/P5 | 成立 | handoffs/M0-P5-frontend-trim-notes.md:1-18 为 closed，P1/P4/P5 有代码落点；最后锁核仍属收尾 |
| M1 handoff：M0/P6 | 成立 | handoffs/M0-P6-knip-notes.md:1-20 为 closed；make knip、lint 上限与 CI 实跑 |

### 5.7 门禁、路由、安全和运行专项试验

门禁变异在仓库外的 /tmp/nerve-m1-gates.onqAt2/repo 进行。该目录由 git clone --shared 建立，按冻结锁文件离线安装；每项试验在 finally 恢复原文件或删除探针。临时克隆最终 git status --porcelain 为空，主目录当时只新增本报告。下表的退出码是外层命令的退出码，Turbo/Make 可能把子命令的失败码汇总成 2。

| 对抗试验 | 步骤与实际结果 | 判断 |
|---|---|---|
| 关键词：未跟踪文本 | 新建 web/apps/web/app/zz-keyword-probe.tsx 写 estimate，make lint-web 退出 2，内层守卫报告 1 hit；文件名改为 *.stories.tsx，node tools/keywords.mjs 退出 1 | 正向拦截成立 |
| 关键词：大小写与包名 | 同文件分别写 GaNtT 和 @plane/propel，守卫分别退出 1；后者报 2 hits | 大小写和品牌规则生效 |
| 关键词：范围边界 | 在 server/internal/、e2e/、根 *.config.mjs 各写 estimate，守卫均退出 0；web 文本前插 NUL 退出 0；写进被忽略的 build/client/ 退出 0 | 与 tools/keywords.mjs:5-9,127-153 明说的 git 范围、二进制跳过一致。它守护被选中的源码，不证明整个仓库文本零命中 |
| 关键词：规则及例外 | 把 estimates.files.source 缩为只含 web/packages/types/，样本检查退出 2；首条例外 count 改 2、match 改成不存在的值、until 改 M1/P5，各退出 1；把正则改成未闭合的 [，退出 2 | 自检、数量、过期和错误正则均能失败；tools/keywords.mjs:34-44,87-103,107-123 是实现 |
| lint 上限与缓存 | utils 上限 25→26 和 25→24 均使 make lint-web 退出 2，分别报实测数低于、高于上限；在 tools/lint-cap.mjs 插入 process.exit(7)，外层退出 2；把 .oxlintrc.json 的三类 warn 关闭，实测 0 也因低于上限退出 2。这些任务都显示 cache miss | 双向精确核对成立，make lint-web 的 --continue 未吞失败；turbo.json:3,12-26 把配置和脚本纳入输入 |
| 锁文件依赖缓存 | 用 turbo run check:lint --dry=json 对照，pnpm-lock 与 workspace 的 oxlint 1.51.0→1.51.1 同步变异后，utils hash 从 23f86636be41d45b 变成 c7528775ea38f434，根任务和 web 任务也变 | 改依赖不会误命中旧 lint 缓存 |
| knip | 新建未用的 utils/src/zz-unused-knip-probe.ts，以及给 authentication-wrapper 加未用导出，make knip 分别退出 2；给 knip.jsonc 加不存在的 ignore，退出 2 并提示 Remove from ignore；在 utils/src/string.ts 加未用导出则退出 0 | 门禁对文件和普通导出有效；通配 barrel src/index.ts:31 使最后一类公开符号看作可使用，这是 P3 review:1566-1587 已记的盲区 |
| vitest | 把 utils/src/url.ts:296 的斜杠判断改成恒假，make test-web 退出 2，next-path.test.ts:53 的用例失败；恢复后 77 测试通过 | 该校验并非无效测试；编辑器测试仍打印 prosemirror-codemark 缺源文件的 5 行 sourcemap 警告（P3/P4 已知） |
| 格式范围 | tailwind-config 的 postcss 配置和 typescript-config 的 base.json 各放一个格式错，单文件 oxfmt --check 退出 1，make lint-web 却退出 0 | P1 review:388 已明确记录这两包和根目录的覆盖缺口；此处证实仍在 |

tools/keywords.mjs 与 tools/lint-cap.mjs 没有单独的单元测试文件；本轮用上表的故障注入核了它们最关键的失败语义。两脚本使用 Node 和 git 的通用接口，未见只限 macOS 的分支；本轮实跑环境是 macOS，Linux 行为只能以 .github/workflows/ci.yml:79-112 的 Ubuntu 配置和已有 CI 记录作间接证据，未独立在 Linux 上重跑。

路由恶意输入在已安装的 utils 校验函数及 Chromium 的 URL 解析下核过。//evil.com、/\evil.com、%2F%2Fevil.com、javascript:、data: 及大小写变体被 isValidNextPath 拒绝；/%5Cevil.com、/%2F%2Fevil.com、/%255Cevil.com 和带内部制表符、换行或 U+2028 的站内路径中有些被接受，但浏览器解析后的 origin 没离开本地源，本轮未复现开放重定向或脚本协议跳转。auth wrapper 在 authentication-wrapper.tsx:31,56-59 使用修剪后的同一值；登录和注册表单在 password.tsx:142 把原始 next_path 交给服务端，这项服务端校验已明确交 M2。对 app/、core/ 中的 useParams 调用与相邻断言做搜索，未见成批用 params.x! 或 as string 压掉缺参；部分通用组件仍靠父路由的挂载条件保证参数，不能只据类型检查断言所有错误地址安全。自动跳转用了 replace；api.service.ts:41-42 的 401 分支只保留 pathname，查询与片段丢失，P4 review 已记。Go 的 handler.go:54-65 对有无尾斜杠都走相同 SPA 回退；make e2e 的 S2 深层路径通过。旧单段 /login、/register 落到工作区段而非旧重定向，符合设计 3.12。

安全静态复核覆盖 web 中全部 window.open、target="_blank"、dangerouslySetInnerHTML 和 innerHTML 调用。显式 target="_blank" 的链接带 rel；未传第三参数的 window.open 在编辑器链接、附件、图片和全屏预览中仍有，P5 review:1094 已交收尾，需在任何外部部署前补上。其余 innerHTML 命中为内置 SVG 字符串或 Critical 1 的粘贴路径；不能因前者而淡化后者。编辑器对普通链接的协议限制有默认 http/https 配置；本轮未在完整业务编辑器里分别走点击、Markdown 导入、图片 src 的所有不可信输入路径，所以不声称这些路径全面安全。浏览器存储抽查见 theme.store.ts、module_filter.store.ts、i18n/core/instance.ts、stale-asset-error.ts：键值主要是主题、过滤器、语言和 __nerve_chunk_reload，没有发现直接写入 JWT/PAT 的代码；泛型 use-local-storage 的动态键不能仅靠这次静态抽查穷尽。

版权来源抽查：git diff --name-only --diff-filter=A 0e43771 HEAD -- web 列出 29 个新文件，其中 19 个 TS/TSX/配置代码逐个看文件头：13 个 Nerve 新写文件有 OpenNerve 和 AGPL-3.0-only 两行，6 个复述 Plane 代码的文件保留 Plane 原声明。web 全树当前有 17 个带 OpenNerve 声明的代码、SVG 或来源文件；三张新 SVG 头部也有该声明，PNG/ICO 在 app/assets/brand/SOURCES.md 登记；根 LICENSE 完整。P5 spec:225-228 对 web 外 tools/、e2e/ 和 api-client 的文件头另有根许可证裁定，所以不能把 tools/keywords.mjs 无文件头当作 M1 违约。修改说明及网络源码的发布闭环仍见 Important 3。

图像审计从强制构建的 client 列出 107 个独立图片：20 SVG、28 WebP、29 PNG、29 JPEG、1 ICO。位图用原始分辨率逐张看，SVG/ICO 用 macOS sips 在仓库外转 PNG 后按固有尺寸看；另从 web/packages/propel/dist/empty-state/index.js 导出 25 个 Illustration，经 React SSR 渲染 SVG，替换调色板变量后逐张看。web/packages/propel/src/icons/ 有 146 个文件、131 个 TSX，名称/全文搜索 plane、logo、brand 只得到版权头和颜色类；将 dist/icons/index.js 的 122 个静态 SVG 导出成索引图目视，五个默认缩放不易辨认的图标再放到 128×128 单看；剩下的是包装或动态选择已有图标的代码。上述图片和图标均未见 Plane 名称或 Logo；旧功能截图见 Important 4。issues.webp 中的 Theme customization 是示例任务标题，不能据此认定自定义主题入口还在。propel 插画源码只随 className 改动，未见按主题改变几何的分支。局限是本轮未在真实登录态逐页确认大图的裁切比例和每个图标的主题呈现；禁用收集箱图片由 DetailedEmptyState 原比例显示，导览用 object-cover。

对强制构建后的 client 逐文件提取 URL 主机，主动第三方资源中可确认 cdn.jsdelivr.net：editor/src/extensions/callout/utils.ts:20 的默认表情图片，以及打包的 Frimousse 表情数据默认地址（emojibase-data@latest）；这两项为 M1 前已有的依赖行为。github.com/open-nerve 是帮助菜单用户点击后打开；w3.org、React Router 文档等其他字面量来自 SVG 命名空间、依赖文本或源映射，不能据此认作页面主动连接。未在产物里找到 Plane API、Unsplash、PostHog、Sentry、Intercom 的主动地址。保留的 jsDelivr 请求意味着使用表情/标注块时仍可能向第三方暴露客户端地址，发布时需纳入 M8 隐私与依赖核查，见第 7 节。

构建及 E2E：P1 记录的 M0 基线是 1239 文件、32,232,132 字节；本轮强制无缓存构建是 534 文件、17,306,349 字节，少 705 文件、14,925,783 字节，与 P5 阶段数字一致。make build、make e2e 均退出 0，S1–S4 为 5/5。S2 只证明首页、深层 SPA 回退和同源请求，不访问进度、编辑器和业务菜单，所以它不能证明大规模删除的保留功能。make web-dev 的浏览器里记录 22 条 path/fs/url/source-map-js 的 externalized for browser compatibility 警告；依赖链从 sanitize-html 及其打包依赖进入开发端，与 P4 基点相同，待收尾的依赖复核；页面仍渲染 Nerve。开发后端未启动时 /api/instances/ 返回 502 是该次试验的环境状态，不计为产品缺陷。

按 P1 review:346-354 相同的扩展名分类口径，本轮强制构建的 JS 为 397 文件/6,818,887 字节，CSS 为 3/296,816，字体为 25/3,755,608，其他为 109/6,435,038；最大 chunk 是 use-parse-editor-content，1,379,320 字节。这些数字逐项等于 P5 review:42-47。初始缓存目录的 689/21,848,544 不等于 P5 数字，其原因和可嵌入风险见 Important 1。

### 5.8 v0-design 1.1 保留功能的入口

这里核的是 M1 是否保留可到达的前端入口，不把尚未接入 /api/v0 的旧数据操作称为端到端通过。侧栏常量及其顺序见 web/packages/constants/src/workspace.ts:146-187，路由表见 web/apps/web/app/routes/core.ts:15-300；其余非路由入口按组件或设置导航定位。7.5 的实际行为状态仍以 5.4 为准。

| v0 保留领域 | 入口判定 | 入口及限制 |
|---|---|---|
| 账户与认证 | 部分成立 | core.ts:15-25 保留登录、注册、引导；settings/profile 路由在 :298-300，constants/src/settings/profile.ts:25-61 有资料、偏好、安全、PAT；实际认证传输交 M2 |
| 工作区 | 成立（入口） | core.ts:21-33 创建与工作区邀请，:51-52 工作区首页，:219-223 成员设置；角色判断仍在 sidebar 常量 :151,158,164 |
| 项目 | 成立（入口） | core.ts:99-116 项目和工作项，:238-267 项目成员及迭代/模块/视图/收集箱开关，:269-277 状态/标签 |
| 状态与标签 | 成立（入口） | core.ts:269-277 两个设置页保留；新状态和层级标签的写入未单独端到端验证 |
| 工作项 | 成立（入口） | core.ts:106-117 项目列表与详情，:54-57 按编号浏览；描述、关系、附件的可见组件与 7.5 编辑器/动态行一起核，Critical 1 是粘贴例外 |
| 四布局与筛选 | 部分成立 | core.ts:106-112 的工作项列表页仍在，P2 review:2147-2210 的旧脚本查过列表/看板/表格/日历；本轮未以真实新接口切换四布局 |
| 评论与动态 | 部分成立 | 工作项详情路由 core.ts:113-117 与 P2/P3 活动、通知脚本有挂载/显示证据；回复、表情和每类操作本轮未完整重跑 |
| 历史版本 | 部分成立 | web/apps/web/core/components/core/description-versions/modal.tsx:67-69 的 Markdown 复制入口在，P2 脚本查过查看/还原；快照数据待 M4 |
| 草稿 | 成立（入口） | sidebar 常量 workspace.ts:160-165 与 core.ts:59-62 保留草稿页；发布成工作项的服务端结果待 M4 |
| 各类归档 | 部分成立 | workspace.ts:182-187 侧栏归档，core.ts:87-92,176-206 有项目、工作项、迭代、模块的归档页；自动归档规则是后端 M4/M6 的责任 |
| 迭代 | 部分成立 | core.ts:119-133 列表/详情；进度卡片取消项不一致（Important 2），燃尽及转移需 M6 新接口数据 |
| 模块 | 成立（入口） | core.ts:135-149 列表/详情，:257-259 功能开关；负责人、成员和模块链接行为未用新接口核 |
| 项目与工作区视图 | 成立（入口） | core.ts:78-85 工作区视图，:151-165 项目视图；workspace.ts:176-180 保留侧栏入口 |
| 需求收集箱 | 部分成立 | core.ts:167-173 收集箱页和 :265-267 开关；禁用态图片有 Important 4，接受/拒绝等后台行为待 M4 |
| 通知 | 部分成立 | core.ts:64-67 有通知页；P2 review 脚本查过预览和全已读，用户提及及服务端送达待 M7 |
| 收藏与最近访问 | 成立（入口） | home/root.tsx:40-46、home-body.tsx:24-30 保留最近访问；侧栏收藏及五类图标由 P3/P5 浏览器脚本查过，M7 换接口 |
| 文件 | 部分成立 | 工作项详情路由、editor/src/extensions/extensions.ts:123-127 保留图片节点；M5 handoff 明说编辑器附件节点不存在，附件区/头像/封面的新服务端行为待 M5 |
| 搜索 | 部分成立 | 命令面板和工作项父子/关系选择器仍有入口，P3/P4 脚本核过路由目标；编号搜索与 @成员结果要随 M4/M7 接口复核 |
| 开放能力 | 部分成立 | core.ts:224-231 有 Webhook；settings/profile 的 api-tokens 页见 constants/src/settings/profile.ts:44-61；接口调用日志和 OpenAPI 页面是 M8 新增，不属于 M1 应保留的 Plane 入口 |

## 6. 已知事项复核

当前没有 closeout spec、分诊表或 review，因此无法复核“收尾做 / 交后续 M / 不做”的最终裁定；下表依据 P1–P5 review 第 6/7 节和现有 handoff。判“同意”只表示该去向在 M1 的阶段边界内合理，不代表事项已经关闭。判“不同意”时说明时间或关闭条件应改在哪里。

| 已记事项与来源 | 判断 | 理由及现状 |
|---|---|---|
| P1 的 Storybook、deploy-files 规则范围；忽略文件和二进制跳过（P1 review:383-387） | 同意边界，反对把它称为全仓零命中 | tools/keywords.mjs:127-153 与本轮变异结果吻合；扩规则应在删除触及新范围时做，无须扫描所有生成文件 |
| P1 的 tailwind-config、typescript-config 和根格式缺口（P1 review:388） | 同意留收尾评估 | 本轮格式变异证实；低影响，但最终质量门禁说明须如实限制范围 |
| P1→P3 的 ensureAPITrailingSlash 导出（P1 review:390） | 不同意“已由 P3 收口” | 当前仍公开，见 Minor 1；只去掉 barrel 导出即可 |
| P2 的用户 custom 主题值回退（P2 review:2677-2690，M2 handoff） | 同意并入 M2 | 新用户模型和保存协议由 M2 定；M1 保留亮色占位不阻断使用 |
| P2 的描述保存撤销/重做竞态（P2 review:2685，M4 handoff） | 同意 M4，但须用真编辑器回归 | 旧基点的 1.5 秒延迟竞态已重现于 review；M4 重写描述保存正是修复点，M1 不再加第二套保存逻辑 |
| P2/P3 的文案键、无引用图片、动态 t() 键和 knip 看不到的对象成员（P2 review:2694；P3 review:1577-1586；P5 review:1095,1100） | 同意交收尾 | P5 记 482 个疑似死键、133 张死图，不会随实际构建发布，但仍违反“删掉”；动态键必须按实际来源核对，不能机械删除 |
| P3 的 login/register 失败回到登录页、Cookie/CSRF、实例 is_self_managed（P3 review:1569-1573，M2 handoff） | 同意并入 M2 | 新 JWT/PAT 与实例模型会替换旧表单协议；关闭条件应含失败提示位置、关闭注册直达、next_path 服务端校验 |
| P3 的非图片拖粘被消费但不插入（P3 review:1575，M5 handoff） | 同意 M5 | 旧社区版没有编辑器附件节点；M5 先决定是否支持编辑器内附件，设计叙述需按 Minor 2 改正 |
| P3/P4 的编辑器测试 sourcemap 五行和开发端 externalized 警告（P3 review:1585；P4 review:2358） | 同意收尾调查，反对静默接受到发布 | 本轮冷测试与开发浏览器均重见；目前是噪音而非编译失败，先查来源，若不影响客户端再留下明确结论 |
| P4 的 API 401 只带 pathname、服务端 next_path 原值（P4 review:2345-2350，M2 handoff） | 同意 M2，必须在认证对接前 | 前端拒绝本轮恶意目标；password.tsx:142 仍原值提交，不能把前端校验当作后端保护 |
| P4 的 RESTRICTED_URLS 与真实顶层段不齐（P4 review:2349，M3 handoff） | 同意 M3 | /login 等旧单段现在按工作区 slug 解析，M3 有工作区创建和保留名上下文 |
| P3 的富文本否定筛选显示层、收集箱 FORMS/EMAIL 来源（M4-issue-core/handoffs/M1-P3-trim-platform.md:28-31） | 同意 M4 | 工作项筛选与收集箱来源由 M4 新领域模型决定；提前删掉显示分支可能丢失既有数据语义，届时应按 OpenAPI 枚举核对 |
| P4 的个人设置页工作项弹窗无入口、#sub-issues 丢片段（M4-issue-core/handoffs/M1-P4-router-native.md:10-18） | 同意 M4 | 前者是无入口的挂载，随工作项弹窗梳理删除；后者目前没有读取片段的元素，M4 若保留跳到子工作项应同时实现锚点和定位测试 |
| P4 的永不渲染应用栏、无用路径 hook、退化 JSX/别名、路由测试宽松（P4 review:2353-2361） | 同意交收尾 | 这是“删掉，不隐藏”的直接对象；不应把未挂载树继续搬进 M2 的领域改写。路由测试的表达式推断可不做，保留浏览器核对弥补 |
| P5 的无 noopener 的窗口调用（P5 review:1094） | 同意收尾修，但不同意外部部署后再修 | 用户写的链接和附件可打开新标签；当前未收尾，部署前应按 review 列出的五处逐一收敛 |
| P5 的 29 张封面照片与导览真人/姓名来源（P5 review:1096） | 同意交负责人裁定；现状未关闭 | 未见授权证据，也没有 closeout 分诊表中的负责人决定；无法取得来源就换为自有或可许可材料 |
| P5 的 pnpm-workspace.yaml 三处 brand 例外直到 M9（P5 review:1097；tools/keywords.json:1630-1635） | 部分同意 | 对上游包 @makeplane/propel 的引用有来源理由；until M9 超过 v0 分期，收尾应逐条确认 count=3 和发布时的归属，守卫自身已核对数量 |
| P2/P3 的 tlds.ts 中 analytics/wiki 两个 M9 例外（P2 review:2696；P3 review:1585） | 同意收尾复核 | 两词是顶级域名数据而非产品功能；可保留精确例外，但需核 tlds.ts 当前来源和两条 count，不能把它们并入 propel 的三个品牌例外 |
| P3 的 347 个未用包导出、182 个疑似死对象成员、205 个未传 prop（P3 review:1577-1585） | 同意收尾逐项定性 | knip 通过不能证明对象成员或通配导出有用；TrailingNode、useProjectIssueProperties 的 fetcher、动态 t() 外壳均在原 review 的收尾清单，不能先按总数机械删 |
| P4 的 callout 数字 emoji 属性、5 个无调用 fetcher、viewId 断言与退化别名/模板（P4 review:2358） | 同意收尾 | 这些是原基点或路由改写暴露的窄点；callout 的数值转字符串有运行时理由，不能一并机械删，其他应按消费者收掉 |
| P5 的导入分组错标、约 20 条失效守卫样本路径（P5 review:1098-1099） | 同意收尾 | 它们不改变运行时行为，但样本若引用从未存在的路径会降低自检解释力；P5 已列出具体文件，收尾应换为实际保留路径 |
| P5 的 EmptySpace 死属性/错误页 alt、Go webui 假文件名（P5 review:1100-1101） | 同意收尾 | 前者属于静态工具盲区，后者测试仍通过但注释误导；删无调用属性并让夹具或注释与真实生成布局一致即可 |
| P5 的 propel 对应源码、版本 1.4.2 与相对 OG 图片（P5 review:1088，M8 handoff） | 同意 M8 作为发布门槛，不同意外部演示前仍无源码 | 版本和绝对 OG 地址依赖公开发布信息；AGPL 对应源码需在任何网络提供或分发前核定，不能只看 M8 排期 |
| P5 对短生命周期存储键、剪贴板 MIME 不做迁移（P5 review:138） | 同意 | 旧键不再读取，旧 MIME 会走常规粘贴；风险是一次性体验差异，不值得兼容层 |
| P5 对系统操作者、收集箱机器人特殊显示交 M4（P5 review:1087） | 同意 | 新工作项/收集箱模型应当统一人类与智能体的显示，不宜在 M1 留 Plane 机器人特例 |

P1–P5 review 已标“已修”的醒目问题也抽样反查：P2 恢复 cycles_description 双语言键、P3 恢复收藏 logo、P4 next_path 修剪值与跳转值一致、P5 八张图中的像素 Logo/Plane Design 已去掉；当前代码和本轮目视支持这些结论。Minor 1 是“交后续 Phase”却没落地，Important 2 是原有脚本数据把危险分支排除；两者不能被旧 review 的 Approved 覆盖。

## 7. 对 M2–M8 的前瞻风险

| 里程碑 | M1 留下的衔接点、代价与收益 |
|---|---|
| M2 认证/用户与 API 客户端 | 现有 services、store 仍传 Plane 形状，auth-forms/password.tsx 仍是 Cookie/CSRF 表单。按 handoff 逐领域改成 OpenAPI 生成类型和同源 /api/v0 客户端，届时一并核服务端 next_path、401 的 query/hash、禁注册直达。先封 Critical 1，避免新认证数据暴露于现有编辑器脚本入口；现在全面改 services 会重复 M2–M5 的领域工作 |
| M3 工作区和项目 | RESTRICTED_URLS 与路由段不一，邀请链接和 /projects/invitations/ 暂存。M3 处理 slug 冲突、邀请权限、离开项目先跳再请求；提前只改保留名单没有完整工作区规则，收益小 |
| M4 工作项、评论和描述 | 共享 issue store 用项目/迭代/模块/主页/归档的条件分派，M1 删了 Epic/多选支路，但仍带旧类型和描述保存竞态。M4 应用领域 API 类型时以路由上下文与真实数据覆盖每个分支，补撤销/重做、动态与通知场景；现在另造兼容层会违反单一来源 |
| M5 文件 | services 中上传/资源类型和编辑器图片复制仍在，编辑器附件节点其实不存在（Minor 2）。M5 决定工作项附件区和富文本附件是否都要支持，然后移除旧 services 的相应方法；先把设计文档对齐可避免误估接口 |
| M6 迭代/模块 | analytics 地址暂时在 cycle.service.ts:13，守卫例外到 M6。按 Important 2 把当前迭代卡片与共享数量函数收成一个定义，再替换进度接口；本轮差异在取消数为 0 的假数据中被遮住，M6 回归要有取消项和全部完成的样本 |
| M7 通知/收藏/最近访问 | 首页最近访问和五类收藏入口还在，通知卡片旧多态已经收窄。M7 改生成类型和接口时要核 URL 落点、未读数及每类图标；不要恢复已删 page/epic 类型的兜底 |
| M8 公开发布 | propel 0.3.0 锁定但对应源码未齐，版本显示仍是 1.4.2，OG 相对地址、照片来源和 AGPL 5(a)/5(d)/13 验收待定（Important 3）。如果 M8 前有外部部署，发布检查必须提前。表情的 jsDelivr 最新数据与默认图片还会发第三方请求，需决定自托管固定版本还是明确告知并接受；自托管增加资源维护量，但可稳定供应链并减少隐私外传 |

质量门禁的长期成本：711 条 oxlint 警告的精确上限确实会在增加和减少时失败，适合作为过渡基线；收尾尚无按包按规则清零计划，M2 起每个改写领域可逐包降低上限，避免为追求“零”而先重写无关 Plane 代码。关键词守卫的 48 条规则和 5 个例外在当前 Phase 自检有效；后续删掉 analytics、邀请接口等例外时应同时删规则例外并更新样本。它按路径和二进制性质有边界，不能当作图片品牌、运行时 XSS 或保留功能的替代检查。knip 的公共通配导出盲区则适合在 M2–M5 改包 API 时人工核查，无需另建一套复杂静态分析器。

## 8. 做得好的地方

- P1–P5 的历史边界清楚，server/、api/ 没有被 M1 顺手改动；安装、生成、检查、测试、构建和 E2E 均可复跑。
- 原生路由、尾斜杠回退和 next_path 的前端校验在本轮恶意输入中没有出现开放重定向；P4 已把服务端责任准确移交。
- P5 对可见图片做过修复。本轮独立按原尺寸目视 107 个构建位图、SVG、ICO 文件，并渲染检查 propel 的 25 个内联插画与 122 个静态图标；应用标题和可达帮助链接没有发现 Plane 的可见品牌。
- 关键词、lint 上限、knip 和 next_path 测试对多种故意破坏都实际失败；这些门禁的有效范围可从代码和变异结果精确说明。

## 9. 处理建议清单

| 时点 | 行动 |
|---|---|
| M2 开始前必须做 | Critical 1：封闭自定义剪贴板 HTML 在清洗前进入 innerHTML 的入口，并以跨来源复制、粘贴做浏览器回归；Important 4：在 M1 收尾清除可见图片里的 Estimate 与 Analytics 旧功能 |
| 可以并入 M2 | Important 1：构建前清除限定的生成目录，确认缓存恢复完整；Minor 1：去掉无调用方的公开导出；M2 认证按现有 handoff 校验服务端 next_path 与关闭注册直达 |
| 随后续 M 处理 | Important 2：M6 统一进度口径；Important 3：外部网络部署前、最迟 M8 补发布合规验收；Minor 2：M1 收尾修设计叙述，并在 M5 决定编辑器附件。已知的 window.open、死资源、照片来源及 oxlint 清零计划由 M1 收尾处理，实际部署前须核其关闭 |
| 不建议做 | 为旧剪贴板 MIME、旧地址、已删功能加兼容层：P5/P4 的不迁移和删除裁定有合理依据；为让 knip 推断通配导出而新增第二套复杂工具：在领域包 API 改写时核查即可 |

## 10. 处理结果（控制者核实与处理）

**做法**：
- 这份报告评审的是 `c493255`（P5 合并之后、收尾之前）。收尾当时已经在做，报告的发现由收尾的架构师对照代码逐条核实（[收尾 spec](../specs/closeout.md) 2.9），7 条新发现全部成立。控制者按"M2 开始之前必须做的在 M1 内做完"的原则给出分诊，负责人 2026-09-25 批准按建议处理。
- 能在 M1 内修的都在收尾里修了，每一条都有进仓库的测试或临时核对脚本，并在基线 `c493255` 上做了反向对照（修复前失败、修复后通过）；属于后续 M 的写进对应 M 的收尾交接，带可核对的关闭条件。
- 第 1–9 节保留 Codex 原稿的样子，处理结果以本节为准。收尾的完整记录在[收尾评审](closeout-review.md)。

### 10.1 逐条处理

| 发现 | 核实 | 处理 |
|---|---|---|
| Critical 1 自定义剪贴板 HTML 在清洗前触发脚本 | 成立。`processAssetDuplication` 在活动文档里新建 `div`、写 `innerHTML` 来标记要复制的图片；`div` 没有挂到页面上，但元素属于页面，`<img onerror>` 照样执行。ProseMirror 自己按 `text/html` 粘贴的路径本来就在分离的文档里解析 | **收尾 T17**（`7836c7a`）：在 `DOMParser` 的惰性文档里标记，之后仍交给 ProseMirror 按 schema 解析；编辑器的新测试在旧代码上失败；浏览器核对（收尾 plan 第 14 项）在基线上 `window.__xss` 为 `true`，修复后为 `false`，同编辑器之间复制图片仍复制资源。任务评审专门做了对抗性的审查，没有找到别的入口。评审另外确认一件 Plane 原有的事：任何网页都能在这个剪贴板类型里写别人的资源 id，让应用请求复制它，所以复制接口要核对权限，**交 M5**（M5 收尾交接"复制资源的权限"） |
| Important 1 缓存命中的前端构建带进陈旧资源 | 成立。Turbo 命中缓存时只把记录下来的文件写回，不删目录里多出来的文件；冷构建时 Vite 自己清空，所以只在命中缓存时出现 | **收尾 T18**（`fc3069a`，跟进 `e070065`）：`make build-web` 在 Turbo 之前清空 `web/apps/web/build/client`（`WEB_CLIENT`，`make build` 复制时也读它）。`prove.sh` 在旧的 `Makefile` 上重现（热构建之后事先放进去的文件还在），修复后冷、热两次构建的 531 个文件逐字节相同、都没有那个文件 |
| Important 2 活动迭代卡片把取消的工作项算进分子 | 成立。卡片是 (完成 + 取消) / (总数 − 取消)，取消 2、完成 8、共 10 时是 125% 和 "10/8 closed"；迭代列表用的共享函数 `calculateCycleProgress` 是完成 / (总数 − 取消) | **收尾 T19**（`c84f26d`，跟进 `f9d1ab2`）：卡片的进度条用 `calculateCycleProgress`，文字和取消的说明走 `t()`（中英文）；浏览器核对（收尾 plan 第 15 项）在基线上是 "5/8 Work items closed"、62.5，修复后是 "3/8 work items completed"、38。迭代侧边栏、模块的进度口径和共享函数读快照的条件**交 M6**（M6 收尾交接"进度的口径"） |
| Important 3 AGPL 的三项义务 | 成立（文档和代码的覆盖缺口）；条款怎样适用是法律解读 | **交 M8**：M8 收尾交接"AGPL 的三项义务和发往第三方的请求"：第 5(a)、5(d)、13 条逐项验收，页面上的源码入口指向运行的提交和完整的对应源码（含 propel），jsDelivr 的两类请求自托管或明确告知；时点是任何对外的网络部署之前，最迟 M8 发布；解读由负责人定 |
| Important 4 保留页面的图片展示已删的估算和数据分析 | 成立。收尾复看保留的全部截图，又找到甘特图、时间线的布局图标和 "Public" 标签，共 9 张图里 10 处 | **收尾 T20**（`ff555bf`）：10 处按背景涂掉，WebP 按文件大小最接近原图的质量重新编码；12 张截图按原尺寸重新看过；`intake-light.webp` 一处用左边一列重复填，保住卡片的底边。浏览器核对（第 13 项）比较页面拿到的字节与仓库里的文件 |
| Minor 1 `ensureAPITrailingSlash` 的公开导出 | 成立。分诊时依据收尾 spec 里的一格以为 T4 已经解决，架构师核对之后更正：两个 `export *` 桶文件仍公开它（收尾 spec 第 9 节第 7 条） | **收尾的修复轮 FW15**（`df4cbf7`）：`services` 的 `helpers/index.ts` 只按名字转出 `normalizeAPIRequestURL`；函数本身保留（`normalizeAPIRequestURL` 调用它，测试也用它） |
| Minor 2 设计 3.1 的"附件节点保留" | 成立 | **收尾评审的提交**：[M1 设计](../M1-design.md) 3.1 改为"图片节点保留；附件由工作项的附件区承担，编辑器里要不要支持由 M5 决定"，与 M5 的 P3 交接一致 |

### 10.2 第 6 节里"不同意"或有保留的各条

| 事项 | 处理 |
|---|---|
| P1→P3 的 `ensureAPITrailingSlash` | 同 Minor 1（FW15） |
| 死文案、死图片、动态 `t()` 键和对象成员 | 收尾删掉 522 个没有引用的键（T4 的 22 个、T11 的 500 个，动态键按来源逐个核对）、141 张不显示的图（T12 的 133 张、T15 的 8 张）、512 个没人用的包导出（T4，三轮）；对象成员和 prop（1350 个）按领域交 M2–M8，每份交接带列出的命令和关闭条件（收尾 spec 第 4 节第 4 条） |
| 编辑器测试的 sourcemap 五行和开发端的外部化警告（"反对静默接受"） | 都查到了来源并去掉：`prosemirror-codemark` 的补丁删掉指向未发布源文件的注释（T9），测试输出干净；外部化警告来自 `sanitize-html` 带进浏览器端的 postcss，删掉它、HTML 工具改用 `DOMParser`（T8），开发服务器的警告从浏览器 44 条、服务端 44 行降为 0 |
| `window.open` 没有 `noopener`（"不同意外部部署后再修"） | 收尾 T2 修完：16 处都带 `noopener,noreferrer`（原来 12 处没有），守卫规则 `window-open` 看住；浏览器核对第 4 项在新页面里核对 `window.opener` 为 `null` |
| 封面照片、导览截图的来源（"现状未关闭"） | 负责人裁定替换：29 张封面换成 Nerve 自己画的图（T14），截图里的真人头像和人名换掉（T15），保留的其余图片按能看清 20 px 细节的尺寸复看一遍 |
| `pnpm-workspace.yaml` 的三处 `brand` 例外（`until: M9`） | 逐条重新核对：三处都是记录配置来源的注释，理由成立，在 v0 内不会消失，保留（收尾 spec 3.6） |
| `tlds.ts` 的两条 M9 例外 | 随 `tlds.ts` 删除（T4，没有读取方） |
| propel 的对应源码（"不同意外部演示前仍无源码"） | 采纳：M8 收尾交接写明 propel 的对应源码与第 13 条一起，时点是任何对外的网络部署之前，不只看 M8 的排期 |
| 其余"同意"的各条 | 按报告的判断处理，落点见[收尾 spec](../specs/closeout.md) 第 2 节的分诊表 |

### 10.3 第 9 节的时点

报告建议"M2 开始前必须做"的两条（Critical 1、Important 4）和"可以并入 M2"的两条（Important 1、Minor 1）都在 M1 收尾里做完；Important 2 的卡片在收尾修掉，其余口径交 M6；Important 3 交 M8，时点按报告的建议；Minor 2 在收尾评审的提交里改。报告第 7 节提到的 oxlint 清零计划写进收尾 spec 3.3 和 M2–M8 的收尾交接。

### 10.4 同步的文档
- [M1 设计](../M1-design.md) 3.1、11 节、12 节；[总体设计](../../v0-design.md) 9.4（M1 已完成）。
- M5、M6、M8 的收尾交接（见 10.1）；M2 的收尾交接另加个人设置主题下拉框的位置（收尾浏览器核对顺带发现，Plane 原有）。
- [前端改动清单](../../frontend-changes.md) 1.6 节 T17–T20 和两次修复轮各行。
