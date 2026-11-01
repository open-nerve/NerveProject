# M4 / P16 `web-cleanup` 执行计划

> 准备期；尚未授权实现。基线 `313728cbc61003d7ed072c1e1e191e1cc2a3d75a`，工作树 `<repo>/.codex/worktress/codex/m4-p16`，分支 `codex/m4-p16`。以 [spec](../specs/P16-web-cleanup.md) 为权威，以下步骤均未执行。

**目标：** 以真实读者、类型和运行证据完成 M4 前端清理，准备 C12，保持已批准的领域边界。

**结构：** 不增状态所有者/依赖/后端协议；沿原共享 UI、原 AST 脚本、lint-cap、关键词 samples 和既有页面夹具收口。C=`web/apps/web/core`，P=`web/packages`，E=`e2e`；全部文件组及当前生产读者见 [inventory](../reviews/P16-web-cleanup-inventory.md)。

## Global Constraints

- 主目录一直 main；仅 P16 worktree 实施。plane/refer 只读，不动已有容器/Docker、不全局安装、不引未点名依赖、不 stash/clean/reset --hard、不取凭据。详细中文提交，系统 `/usr/bin/git` 推送；阶段完成先不合并。
- 本次只写文档。实施 T1 起才把 phase 改 M4/P16，最终 closeout 另改。规则/样本/忽略范围/到期日不放宽，cap只降；有意义的类型负例不能运行到应用中。
- 计划保持约1500行以内且≤16Task；新生产文件约400行内、单Task含测试约1500行作为复核参考，十一Task估计约6000–10000行总改动（含删除、ledger与测试，非承诺必须改够）；超出按职责先拆到≤16，仍超才按设计D2上报分段。不因大批量重命名把不同职责塞入大组件。
- IssueRootStore权限/确认写、会话代际/路由owner、原生生成API、M7 view/版本/成员目录合同保持；测试复用已有帮手，数据库只作断言，不重造完整世界。
- 名称/提示新文案 en/zh-CN；键盘复用isComposing。局部root cause自行修，架构/协议/跨领域责任新变化先提交已完成工作再报告。
- 原基线已有绿性质先核基线，新增缺口才有首次红；以有效变异证明关键断言。纯声明类型负例预期是指定TS错误，不能把任意编译失败充当行为红。

## Review Focus

1. 泛型是否贯穿 options/value/onChange，单/多值判别是否被 any/boolean/断言重新抹平；对象、0、空串、null与数组是否保留真实语义。
2. dead ledger是否每行具名读者，动态/第三方/测试说明是否可查；历史52行是否真正追溯；删除是否带走整个独占链。
3. no-shadow重命名是否误改key/闭包/effect，改动文件是否零告警，cap是否按实际数同步下降。
4. 共享 UI 的真实键盘、IME、回焦/portal、M2/M3与M6/M7消费者是否受影响；测试是否只 mock掉了被验组件。
5. 守卫真实范围、S2精确请求和三浏览器实际执行；P16准备/阶段完成/M4closeout三者是否混淆。

## 验证与证据协议

所有命令在 P16 worktree 根。U：`pnpm --filter web exec vitest run <任务明确文件>`；UI：`pnpm --filter @nerve/ui exec vitest run <src/...>`。各 Task 用 `pnpm --filter <实际改动包> check:types`，跨包改后先 `pnpm exec turbo run build --filter=web^... --force` 更新依赖dist，再核web；`git diff --check`和改动文件oxlint必须通过。静态审查和复用证据不冒称新运行。

页面 B：`make build` 后 `pnpm --filter @nerve/e2e exec playwright test <精确故事> --project=chromium --grep '<实际标题>'`。三个引擎用既有四个M4专用配置或T10的新组合配置，显式 `--project` 与 `--list` 核非零；数据库/testcontainers套件串行。既有watchPage、expectQuietConsole、expectListPageQuiet、recordFocuses、pageBehindHidden、modalDialog及API/auth/clock夹具不复制、不放宽。

T2、T5、T10 的产品push检查点及T11最终，串行完整 F（没有新修改/失败/未解风险不重复同套）：
~~~sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
~~~
每次保留argv/cwd/input SHA或dirty逐文件hash/raw/exit/executed/skip、变异补丁/红断言/恢复hash，放 `.superpowers/sdd/P16-web-cleanup/`。W11按06处理，非W11先看实际日志；已登录浏览器/连接器可取GitHub日志，不提取凭据。合并CI的W11重跑仍失败即报告。

## Task 1：焦点索引与实施入口（约150–400行）

**Files：** P/utils/src/tab-indices.ts、P/constants/src/tab-indices.ts及barrel，P/ui/src/dropdowns/{custom-menu,helper}.tsx，inventory所列七个消费者，tools/keywords.json，M6/M7现有handoff。
**Interfaces：** getTabIndex/ETabIndices/CustomMenu.buttonTabIndex维持存活读者合同；仅无读者的独立分支可删。

- [ ] Step 1：重验main基线/分支、当前消费者；建立实施台账，phase改M4/P16并执行关键词检查，不改例外到期日。
- [ ] Step 2：逐七文件核getTabIndex实际调用、枚举组、CustomMenu直接tabIndex和barrel；区分M4已删路径、M6/M7仍在路径。
- [ ] Step 3：记录存活共享代码与最后删除条件，更新M6/M7现有交接子项；只删确无消费者的独立成员，不改未来领域键盘行为。
- [ ] Step 4：类型/关键词/受影响shared UI测试；静态负例M01在审计输入加回一个M4正tabIndex读者必须被标出，恢复。若零产品改动不伪造运行修复。
- [ ] Step 5：写A1、当前路径和两域归属；提交 `refactor(M4/P16): 核清焦点索引的最后消费者与删除条件`。

## Task 2：共享下拉框泛型与消费者（约800–1400行）

**Files：** spec§2.2、inventory的32生产文件及类型读者；新增 P/ui/src/dropdowns/select-contract.typecheck.tsx、select-contract.test.tsx（两个文件各有单一用途）；既有 C/components/dropdowns/basic-dropdowns.test.tsx、rich-filters选择器测试。
**Interfaces：** Select<T>任意值/可空当前值/回调T；SearchSelect<T extends string|number>按multiple字面量区分T与T[]；菜单展开无参，item为真实MouseEvent；共同字段、portal、键盘合同不变。

- [ ] Step 1：先读实际UI+消费者与现有测试；写类型正负例（string/number/主题对象、单多值/错误回调/错误option）及真实组件 `preservesEmptyAndZeroSelections`、`multiSelectionStaysOpenAndReturnsAnArray`、`objectSelectionReturnsTheOriginalThemeOption`。类型fixture由tsc实际include，不运行负例。
- [ ] Step 2：执行UI类型和组件测试，确认负例确实因旧any不报错而触发unused expect-error，运行缺口给具体值/回调失败；已有正常键盘绿性质保持。记录原始baseline，不把no-test-found当红。
- [ ] Step 3：实现强类型合同并逐32文件适配；泛型贯穿options/common.ts，不使用宽泛boolean/any/unknown双断言。保持单选内部null不外发、0/空串、对象、不提交表单、多选数组/关闭策略；同步删真实死参数/分支及改动文件warning、下调cap。
- [ ] Step 4：UI测试、web实际消费者测试及全包类型；B list-interactions C3的设置/表单/peek入口、filter-archive-profile-interactions C2选择器；M02错值类型、M03错单多回调、M04单选吞0、M05多选选后关闭、M06空态回调、M07改对象值各由对应编译/行为断言检出并恢复。串行F并固定候选输入。
- [ ] Step 5：逐调用路径记兼容结论、A2/类型fixture入口与局限；提交 `refactor(M4/P16): 贯通共享下拉框的选择值与回调类型` 后push检查点，核远端HEAD为该提交并记录CI。

## Task 3：全仓no-shadow升错（约450–1000行）

**Files：** `.oxlintrc.json`，inventory列出的27个显式告警文件及8个屏蔽文件（共35）、其所属测试/package.json；不改工具规则实现。
**Interfaces：** 仅重命名局部绑定，wire/property key/公共API不改；涉及原有其它warning的局部修复保留运行语义。

- [ ] Step 1：对所有包/tools/E取oxlint JSON，按code枚举显示38条与8处屏蔽隐藏的8条（当前合46条/33文件，连同2处无效屏蔽共35文件）；解除屏蔽后刷新清单并核同文件其它warning，逐行确认内外绑定与shorthand键。
- [ ] Step 2：先跑对应现有测试/类型；为闭包或组件实际缺口补 `shadowedCallbackKeepsItsOriginalValue` 等拥有者测试，不为单纯重命名复制实现。固定改前输入和实际告警。
- [ ] Step 3：根因清理绑定和所碰文件其它告警，删掉全部8处旧屏蔽（含无效两处），根规则设no-shadow为error；每包按实际数同提交下调cap，无ignore/override/disable豁免。隐藏8条从未计入原cap，不把移除屏蔽后的暂时增加当成抬cap依据。
- [ ] Step 4：逐包真实扫描no-shadow 0、无error，专项及全规则disable扫描0；M08在一次性合法TS输入中放嵌套遮蔽必须诊断error，M09不遮蔽正例通过；M10涉及序列化的重命名若变键由现有消费者/新增必要断言检出；全部恢复再核类型/测试/cap。
- [ ] Step 5：写A3、35文件实际处理与无屏蔽/全仓扫描/升错证据；提交 `refactor(M4/P16): 清零变量遮蔽并将规则提升为错误`。

## Task 4：死成员与历史52行（约700–1500行）

**Files：** 原M1 closeout plan附录三个脚本只复制到ignored证据目录；当前M4候选生产/测试文件；新增 reviews/P16-dead-members.md，M1/M3子回执。
**Interfaces：** 保持原AST分类、第三方回调/公开ref/原生DTO/表格列合同，只删无读者链。

- [ ] Step 1：typegen后运行原deadsym --tsv、domains --rows M4；逐项标生产/测试/第三方/误归属，核与准备453候选的差异，不拿历史数量冻结清单。
- [ ] Step 2：沿symbol追真实读者/写入及唯一用途分支；历史52行从原报告/版本逐名映射到现状，零候选不等于整文件已删，WorkspaceService view四方法逐一保留。
- [ ] Step 3：删除真实dead及独占参数/导出/分支；保留项每行具名读取位置/动态机制和相应测试。公开EditorRefApi与TipTap插件等不以脚本零read删合同；只因测试方便而留的多余fixture字段也须收掉。
- [ ] Step 4：新AST逐行ledger对账、knip、类型和对应行为测试；M11审计输入缺一行必须失败，M12一个真实动态协议/列回调去除由对应消费者测试失败。审计断言和产品断言分列；恢复后再次生成完整终态。
- [ ] Step 5：写A4、各kind/文件计数、历史52逐名落点与新旧identity映射；提交 `refactor(M4/P16): 清理死成员并核实动态合同的真实读者`。

## Task 5：告警终态与精确cap（约700–1400行）

**Files：** inventory计划告警范围与实际新增改动文件；各包package.json。C/components/{dropdowns/layout,issues/issue-layouts/calendar/issue-block-root,issues/issue-detail-widgets/sub-issues/title-actions}.tsx、P/editor存活菜单/插件为关键行为读取点；其它具体文件逐项台账，不把未知文件列为已改。
**Interfaces：** 保持键盘/焦点/effect和未启用的图片上传边界；lint-cap严格相等不变。

- [ ] Step 1：刷新所有包diagnostics，合并domains M4、M3基线53a6a05b至本期的TS/TSX改动集合及T2/T3触及文件；核准备103条显示+8条隐藏诊断、63文件（含无效屏蔽）的实际剩余，不能只按本分支增量漏掉M4早期改动。
- [ ] Step 2：逐实际行为缺口补拥有者测试：可交互项Enter/Space可达、无嵌套button、layout/effect取当前输入、菜单/编辑器不开新增网络。数组/局部绑定等纯机械清理复用既有回归，不为每行重命名单独造测试。
- [ ] Step 3：修真正交互语义、effect或数据转换；真dead移除整链；按每包实际数同提交降低cap，改动文件告警清零。新UI文字双语，不增加role空壳/空handler/disable。
- [ ] Step 4：逐文件oxlint零、全包数等于cap、no-shadow仍0/error、类型与所属组件/编辑器/HTML安全回归；M13交互失去键盘、M14旧闭包忽略新输入由具体断言检出；cap一少一多的现成lint-cap诊断以临时参数验证不变。串行F并固定候选输入。
- [ ] Step 5：写A5、基线→终值表和具体删除/修复证据；提交 `refactor(M4/P16): 收口前端告警并同步降低各包上限` 后push检查点，核远端HEAD和CI。

## Task 6：关键词终态（约150–350行）

**Files：** tools/keywords.json现有三规则及samples，具名searchWorkspace/projectIssuesSearch扫描；既有tools/keywords.mjs保持策略。
**Interfaces：** old-issue-urls的八类匹配范围、negation-filters全名和copy-hook各领域到期日不变。

- [ ] Step 1：读phase与当前exceptions，确认旧URL/否定筛选0例外、M4复制0、其它copy10条准确归属；匹配规则外的M6/M7旧地址另列所有者，不称全应用零旧接口。
- [ ] Step 2：执行node tools/keywords.mjs及既有samples；已有终态绿先记录，不为清理任务强造生产改动。
- [ ] Step 3：只删除已不匹配的真实陈旧例外或补必要样本；当前0条即合法，不添加虚假M5/M6/M7例外，保持正则/范围/期限。
- [ ] Step 4：M15审计输入加入M4旧工作项地址、M16恢复否定字段必须被真实规则检出；原生/api/v0和合法同名片段通过。M4复制及旧搜索具名扫描复核，恢复后整份守卫绿。
- [ ] Step 5：写A6、具体命中/例外数量与适用范围；提交 `test(M4/P16): 固定旧地址与筛选守卫的最终边界`。

## Task 7：成员getter最后读者（约120–350行）

**Files：** C/store/member/project/project-member.store.ts及测试，C/components/issues/issue-layouts/utils.tsx；inventory所列M7三文件和现有handoff。
**Interfaces：** getProjectMemberIds(projectId,false)显式读取保持；M7仍读projectMemberIds getter，保留存活合同。

- [ ] Step 1：rg所有同名读写，区分getter、interface/computed注册、测试和M4局部数组，确认M4历史两读者已无。
- [ ] Step 2：核M7 mobile-header/view-list-header/filter-selection的实际路径与用法，运行现有project成员store测试与类型，不能只grep名字断言0。
- [ ] Step 3：不删存活getter；记录M7最后迁移读者后连同interface/computed注册与专属测试更新的条件；不改成员目录权限/会话。
- [ ] Step 4：M17静态审计区分一个M4显式参数方法调用和一个恢复的隐式getter读取，后者失败；字节恢复，成员store和受影响消费者回归。
- [ ] Step 5：写A7/M7子交接；提交 `docs(M4/P16): 明确成员隐式getter的最后迁移责任`。

## Task 8：前端说明与子交接终稿（约250–500行）

**Files：** README.md前端节、docs/v0/frontend-changes.md的M4行、docs/v0/plane-diff.md对应行，M1-P3/M1-closeout/M3-closeout及M6/M7输入子回执。
**Interfaces：** 文档只表述真实已合并/本期已验证的能力，历史失败与输入路径不覆盖丢失。

- [ ] Step 1：按spec§2.8逐句核README当前P12/P13等历史字样、两清单与P12–P15集成证据，列需更新行。
- [ ] Step 2：对照实际路由/请求合同/测试，区分原生M4能力、未接通领域、增量窗口/HTTP复制等既有限制。
- [ ] Step 3：完成本期文档和子项接收；保持八份M4顶部open、设计§14/总体9.4未完成；不生成最终M4-closeout来冒充收尾。
- [ ] Step 4：静态相对链接/章节/文件核验，diff逐行审；A1–A7证据与本期状态一致，尚未完成的A9–A11仍未勾。
- [ ] Step 5：写A8，提交 `docs(M4/P16): 更新原生工作项能力与前端清理说明`。

## Task 9：测试缺口与S2终稿（约650–1200行）

**Files：** E/stories/smoke/s2-web-app.spec.ts；既有issue-list-page/issue-detail-page/browser/auth/issue/issue-list夹具；T2–T5发现的对应组件测试。
**Interfaces：** followRequests/expectExactly保持精确方法+path+query与重复数，前置经API，不SQL写；四身份与404/403规则不改。

- [ ] Step 1：建立主入口→实际故事/请求表；在S2现有REQUESTS之外补admin/member归档/个人三关系/草稿，保留guest不读个人列表及草稿拒绝、project-non-member不读项目工作项；详情/命令动作与既有S2(detail)/I18/C7互链。
- [ ] Step 2：跑B stories/smoke/s2-web-app.spec.ts，缺口先有明确路径/预期请求差异；先读生产合同推expected，不把watch实际输出自动照抄成白名单。
- [ ] Step 3：补入口分支/夹具必要字段与拥有者测试；请求精确、拒绝有可见说明、无旧地址/未处理拒绝/未说明console；不重复实现已有功能，不改权限规则。
- [ ] Step 4：S2全文件与受影响I故事通过；M18偷偷增加列表GET/重复GET、M19访客个人页误取列表由具体请求断言检出；恢复再绿，API写后确认与DOM就绪不用sleep代替。
- [ ] Step 5：写A9、四身份/全部M4入口覆盖和实际case数；提交 `test(M4/P16): 补齐最终页面入口与精确请求检查`。

## Task 10：C12三浏览器准备与影响回归（约300–650行）

**Files：** 新 reviews/P16-C12-preparation.md；必要时新E/playwright.m4-cleanup.config.ts、E/package.json脚本；既有四个M4config及对应故事/fixtures只按缺口补充。
**Interfaces：** 继承base的testDir/setup/fixtures/process，不复制Playwright世界；三project同scope，默认A4限制不带入。

- [ ] Step 1：逐spec§2.10页面/身份/焦点/右键/Back列矩阵，映射现有C1–C11/I/S2，重复test id去重；已有题名C12不等于全部最终C12。
- [ ] Step 2：明确组合配置并--list，检查Chromium/Firefox/WebKit各非零、标题和范围相同；没有执行的页面格标待收尾，不写通过。
- [ ] Step 3：只补当前改动暴露的焦点裁切/菜单/右键缺口，保留真实控件与数据库/请求观察；框架已有fixture直接复用。
- [ ] Step 4：make build后串行三引擎受影响入口；记录各格实际run/skip/截图/trace和console，新config用显式文件路径补oxlint；若根ignorePatterns跳过，按P15已证实方法复制完全相同规则到临时配置、只去掉ignorePatterns后扫描该文件（--no-ignore本身不能覆盖配置排除；生产配置不变）；M20恢复默认Firefox A4 grep由清单断言失败，M21关键焦点/右键行为失效由真实页面断言失败。恢复；串行F并固定候选输入。
- [ ] Step 5：写A10，清楚标“C12准备及受影响自动回归完成，控制者最终全路径C12待closeout”；提交 `test(M4/P16): 准备三浏览器收尾矩阵并核共享控件影响` 后push检查点，核远端HEAD和CI。

## Task 11：整分支deep-review与阶段交付（约350–800行文档，修复另计）

**Files：** 全分支baseline..candidate，spec/plan/review/inventory/dead与C12表、设计§15、输入子回执，新增docs/reviews/re-NN-M4P16整体审核.md（届时取下一个编号）。
**Interfaces：** 复用既有资源/权限/会话，不把预检结论当实现批准。

- [ ] Step 1：固定候选输入、更新AST/改动文件警告/全仓no-shadow/cap/关键词终态，A1–A10逐条要求与实际证据对账；未影响旧证据须hash和input等价证明才能复用。
- [ ] Step 2：由主控按deep-review顺序审逻辑、集成、覆盖，读跨Task调用链和测试；至少抽十五个有意义负例/变异并复核红失败原因与恢复。无授权不另起并行审者。
- [ ] Step 3：局部根因修复并补必要验证，按修复范围复审；结构/高风险问题先提交现有成果再报负责人，保持未完成状态。
- [ ] Step 4：最终串行F（同输入已有最终F且无新变化不重复），逐引擎影响回归、文档/所有输入验真、完整证据manifest；提交/推送后核准确branch HEAD三CI job全绿，不用较早检查点代替。
- [ ] Step 5：全部条件成立才勾A11与阶段完成、记录review/deep-review/CI/raw入口；提交 `docs(M4/P16): 记录前端清理终审与阶段验收` 并推送，最终文档提交也须准确HEAD三CI job全绿，提交后收据留在证据目录，不能以此前候选绿代替。先不合并，不清理worktree，不启动closeout实现。

## 变异与证据索引

M01/T1消费者审计；M02–M07/T2类型与真实选择；M08–M10/T3升错/序列化；M11–M12/T4逐行审计/动态读者；M13–M14/T5键盘/effect；M15–M16/T6关键词；M17/T7隐式getter；M18–M19/T9请求/访客；M20–M21/T10浏览器清单/真实控件。若某性质没有产品变化、原有覆盖已证实，核原始hash与输入等价后复用，不能强造不适用变异或以任意TS失败替代行为证据。最终明确逐条在哪层被检出。

准备期仅源码/测试阅读、静态盘点、文档审查；没有运行以上Task、未新增产品代码、未更改phase。实施授权后从T1开始。
