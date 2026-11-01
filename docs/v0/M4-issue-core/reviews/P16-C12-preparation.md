# P16 / C12 准备与三浏览器自动回归

状态：T10 本地验证完成，准确检查点 CI 待核。本文是 **C12 准备与自动回归记录**，不是控制者最终全路径 C12 验收；后者随 M4 closeout 完成。基线、spec、plan 与阶段证据见 [P16 review](P16-web-cleanup-review.md)。

## 1. 范围、选择与运行方法

组合配置 [playwright.m4-cleanup.config.ts](../../../../e2e/playwright.m4-cleanup.config.ts) 复用原 base 的 testDir、globalSetup、数据库/进程夹具，汇入 list/filters/detail/collaboration 四份原配置的 match/grep，并增加 S2 的全部冷加载入口与 cleanup-interactions。显式声明 Chromium、Firefox、WebKit；不继承默认 Firefox 仅 A4 的限制。

原配置每引擎分别 11 / 22 / 44 / 43 项，合计 120 次选择、实际 test id 去重后 **110** 项；再增加原 detail 未选择的 10 条 S2 及本期 5 条编辑器页面，组合 **每引擎 125 项，共 375 项**。清单审计比较每引擎的 file/title 集合、真实 id 唯一性、所有原 id 仍在集合中，不凭题名前缀估计覆盖。`--list` JSON 的 skipped 只是未执行的枚举状态，不是一次成功运行。

本期共享选择器、编辑器包装层、菜单生命周期与 S2 入口的影响横跨原四组配置，故执行组合中的原 123 项/引擎及补充的 2 项嵌套菜单/引擎；三个引擎串行运行，测试中的数据库和进程仍由现有夹具管理。命令为：

```sh
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --list --reporter=json
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --project=chromium --workers=2
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --project=firefox --workers=2
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --project=webkit --workers=2
```

原始 argv/cwd/input hash、逐用例结果、构建摘要、截图/trace 在 `.superpowers/sdd/P16-web-cleanup/`。只失败时留截图和 trace；“无失败附件”不等于做了人工视觉检查。新配置额外按原 oxlint 全部规则显式扫描：临时规则副本仅移除 ignorePatterns，生产配置不变。

## 2. 页面与身份矩阵

下表每行适用于三个引擎；最终控制者都仍需在同一交付版本逐项核对。`焦点/锚点` 指现有 DOM 聚焦、回焦、弹层位置断言；它不等于所有像素级轮廓已检验。`轮廓` 列明确尚待人工检查，只有本期新增的三个控件检查了 computed outline、透明度、视口和各裁切祖先的几何边界。

| 页面/入口 | 身份与已有故事 | 自动键盘、右键与关闭/Back | 轮廓与最终控制者 |
|---|---|---|---|
| 项目列表 | member；I3/I4、C1/C3；S2 admin/member/guest/non-member | 分组/排序、属性选择器、行菜单、真实右键方向键/IME Escape/回焦；精确请求与拒绝 | 焦点/锚点自动；轮廓待 closeout |
| 看板 | member；I3/I6、C1 | 看板顺序、拖放确认/拒绝；菜单沿共用路径 | 最终键盘全路径/轮廓待 closeout |
| 日历 | member；I4/C1 | 日期区间、当前显示偏好；组件另核新周起始输入 | 最终菜单/轮廓待 closeout |
| 电子表格 | member；I3/I4、C1 | 列排序/显示和原生数据；共享选择器由 C3 保护 | 最终键盘全路径/轮廓待 closeout |
| 快速添加 | member；I7 | Enter/错误字段先说明再回焦、服务端拒绝保留；确认写 | 焦点自动；轮廓待 closeout |
| peek | member；C3/C4、I1/I2 | 加载中关闭/迟到读取、布局选择、标签、复制失败；回焦或旧触发器退役 | 焦点/弹层锚点自动；轮廓待 closeout |
| 富筛选与显示 | member；I5/C2；single/multi en/zh-CN 兼容夹具 | 搜索/清空/选中、0/空值组件合同、日期草稿、桌面/窄屏位置、Escape/回焦 | 焦点/锚点自动；轮廓待 closeout |
| 归档列表 | member；I15/C9；S2 admin/member | 归档/恢复、确认窗口 Tab/Escape、Back 不写、精确冷加载请求 | 焦点/锚点自动；轮廓待 closeout |
| 个人 assigned/created/subscribed | member；I19；S2 admin/member/guest | 三关系、局部偏好、访客无列表/工具栏/旧读取 | 最终键盘与轮廓待 closeout |
| 编号/项目详情 | member/guest/non-member；I1/I2/I20/I22、S2(detail) | 200/404/403、跨工作区错误地址、会话更替；编辑/拒绝/回焦 | 焦点自动；轮廓待 closeout |
| 创建与更新表单 | member；I1/I2/C4/C10/C11 | 表单不误提交，日期/标签/父项内层窗口隔离，IME、确认窗 Tab、Back | 焦点/弹层锚点自动；轮廓待 closeout |
| 子项/关联/链接 | member/guest；I8/I9/I10/C4/C5 | 搜索/创建/删除确认、复制失败、子项显示/筛选按钮；当前权限拒绝 | 焦点/弹层锚点自动；轮廓待 closeout |
| 删除 | member/guest；I17/I20/C7 | 当前工作项命令、确认窗、忙碌 Escape、写一次后离开 | 焦点自动；轮廓待 closeout |
| 描述编辑器 | member；I2-description、editor-loading、cleanup-interactions | 撤销重做/拒绝重试、按需加载/粘贴安全；继续输入原生 Tab、表后输入入库 | 新按钮轮廓几何通过；全编辑器人工待 closeout |
| 表格行/列菜单 | member；cleanup-interactions | Tab、具名触发器/面板、IME Escape、回焦、第二行/列键盘插入且入库、应用内 Back 退役 | 两触发器轮廓几何通过；草稿外层隔离与 peek 的两级 Escape 已核；全菜单人工待 closeout |
| 评论/表情/动态 | member/guest；I11/I12/I20/C6 | 读写权限/作者、编辑取消/拒绝、分页、筛选/排序、表情；确认写与焦点 | 焦点自动；轮廓待 closeout |
| 描述历史 | member；I13 | 首页/下一页、两版内容/恢复、当前路由验证后挂载 | 窗口与请求自动；轮廓待 closeout |
| 草稿创建/编辑/发布 | member、guest；I14/C9、S2 | 保存/重复提交/拒绝、复制迁移、发布、权限重载、关闭/Back；访客拒绝说明 | 焦点/隔离自动；轮廓待 closeout |
| 命令弹窗与顶部搜索宿主 | member；I18/C7 | 搜索、键盘/IME、当前对象、双宿主标签多选、旧范围迟到响应、Back/会话退役 | 焦点自动；轮廓待 closeout |
| 局域网 HTTP 复制 | member；C8 | 十二个实际复制入口，在无手势/非安全上下文说明失败、不假成功/写入/导航 | 保留“有手势可能成功”的既有限制；人工待 closeout |
| 设置等共享选择器消费者 | admin/member；C3、S2 | 自动化月份、项目日期筛选、API token 日历、workspace 菜单；表单隔离/回焦 | 焦点/锚点自动；轮廓待 closeout |

请求/控制台观察全部沿 `watchPage` / `expectListPageQuiet` / `expectQuietConsole`。S2 对冷加载按方法+path+query及重复数精确比对；其它故事对相应动作/拒绝/写后数据库和迟到响应断言。兼容筛选夹具使用实际共享 UI，但不冒充生产 API 故事。各故事已有的受控 403/422/503 或 LAN 浏览器警告保留明确来源，不增加任意网络/console 忽略。

## 3. 本期补充与有效失败

- 编辑器内容包装层原本无条件处理冒泡 focus，表格/图片等内部控件获得焦点后仍被 TipTap 的下一帧 focus 命令抢走。真实包装层的两项红例与页面事件堆栈确认根因；现在只处理 `target === currentTarget`，包装层自身的旧 tabIndex 聚焦合同仍有正例。
- 表格行/列控件补当前语言的名称、可见焦点、面板初始焦点与 Tab 管理；打开菜单明确选中其对应行/列，动作仍走原编辑器命令。颜色按钮补双语名称，rich/lite 共用一个翻译入口。
- 原插件在离开表格时丢掉 renderers，且将 destroy 写在不会被 ProseMirror 调用的顶层。现在由 Decoration.widget 的实际 DOM 挂载/销毁回调创建和释放 ReactRenderer，不为从未挂载的中间 transaction 提前造 renderer；只读时不提供可操作 decoration。真实组件六项红绿覆盖离开/只读/销毁与重新进入。
- 行/列菜单直接退役原本不会经过 onOpenChange(false)，因此 table 键盘标志残留。改为已挂载面板的 effect 成对注册和清理，六个离开/只读/销毁用例先红再绿；真实公开 ref 验证标志清零。
- 同一编辑器实例切换标签 context 后，原 widget 与 portal 的行列名、颜色名即时更新；真实 editor 整包 71 项通过。草稿弹窗的背景隔离、先关闭菜单再关闭草稿或 peek 三引擎各 2 项通过，未发现额外产品缺陷。测试初稿错误地用无尺寸外壳判断可见、要求已退役列表行回焦，已改按实际 heading 和菜单拥有者断言，原失败保留。
- Firefox 首轮 S2 的字体请求在相邻硬导航间被中止（资源本身 200），控制台记录 status=2152398850。trace 确认请求/导航时序后，visit 在下一次导航前等待 document.fonts.ready 并断言无 error 字体；原 console 精确观察保留。Firefox S2 五次与最终矩阵通过，没有加忽略项。
- 初始测试中的 Shift-Tab 在表格内部原本用于单元格导航；去标签后的文本本来不自动插空格；新挂载编辑器须先选择表格；打开面板须等其真实初始焦点后再选择动作；硬导航会正常刷新认证。上述测试预期/同步问题都保留原失败，不冒称产品红例；最后使用原生控件、已有应用内导航夹具和实际焦点/请求就绪信号。

## 4. 运行与反证收据

- `t10-complete-inventory-summary.json`：原配置 110 项去重，加 10 条 S2 和 5 条 cleanup；三个引擎各 125 个唯一实际 id，范围相同。M20 恢复默认 A4 grep 后 Firefox 为 0，断言按预期失败。
- `t10-matrix-r2-{chromium,firefox,webkit}`：各 123 passed，0 skipped/0 flaky。输入 SHA `fc5b6e41dbd16990a3aab8c045caa0cc6a92077e9544d48337a6462a54b6c8c1`。
- `t10-nested-final-{chromium,firefox,webkit}`：各补充 2 passed，0 skipped/0 flaky。输入 SHA `e48268d018d2f9a2942c61b4bc2a9ec3f48768dd370dcad59d0b63ab24ee1a44`；两次之间仅新增/扩展两个测试文件，产品及原用例行为不变。`t10-matrix-complete.json` 对三引擎 125 个 id 的并集、输入差异、逐例成功/无重试及 bin 一致性机器核验。
- 上述构建 bin SHA `145877f59635091780620fcaff922afd5938a7a665bd2ff2601927ef0cee5554`。新配置原规则显式扫描 `t10-config-lint` 为 0 诊断。
- M21 恢复旧冒泡 focus 行为后真实列按钮的 opacity 由 1 变 0，被原页面断言检出；`t10-M21-red` 保留红日志，恢复源码 SHA `dce7371c7a81127a038a78cfb868c70f43a9d0f989c0193bb2f6ab0eb5b86bd2`，重建及后续三引擎原用例通过。原始失败/恢复，不把测试同步错误计作有效变异。
- 最后测试修正后，`t10-i11-final-{chromium,firefox,webkit}` 各 10/10，`t10-font-final-{firefox,webkit}` 各 13/13；Chromium S2 由最终完整 E2E 284/284 覆盖。受测产品/bin 不变，新增严格文字断言及字体同步未放宽既有观察；详见阶段 review §20 和 `t10-f-r3-input-equivalence.json`。
- 完整 F 已通过，准确检查点 CI 在阶段 review §20–21 续记；当前不预标 CI。

## 5. Closeout 待办

- [ ] 控制者在最终 main 版本逐行完成三个浏览器全页面 C12，核焦点轮廓、裁切、键盘/右键、关闭/Back，以及真实请求/控制台。
- [ ] 将本期自动结果和最终人工结果分列，记录准确版本及未执行格，不能用清单或历史运行替代。
- [ ] 与 M4 设计 §3.23/§14、总体设计 9.4、规模数字和最终交接一起完成，P16 本阶段不提前关闭。
