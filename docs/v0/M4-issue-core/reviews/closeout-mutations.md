# M4 收尾：有效反证逐项证据

2026-11-02。审计输入 `d66dc91d29ca5b03bc90b727da2c3fa63058eeca`；关键性质 15 项、局部修复补充 9 项，共 24 个有效变异。所有临时源码精确恢复。最终产品输入、完整 F 与 C12 仍由 T13 收口。

原始根为 `.superpowers/sdd/M4-closeout/`。`t10-mutation-manifest.json` 逐项保存原态/变异/恢复源码 SHA256、失败收据/日志 SHA256、正常回归收据、历史测试及探针的当前等价核验。失败必须来自实际行为断言，不计编译、夹具或命令错误。

## 1. 关键性质（K01–K15）

| ID | 性质 | 有效失败收据 | 恢复后通过 |
|---|---|---|---|
| K01 | 工作区成员锁必须阻塞 FOR SHARE | `t2-mutant-remove-lock` | `t2-after` |
| K02 | 周期 kind 必须注册 worker | `t3-mutant-periodic-worker` | `t3-registration-restored` |
| K03 | 投递 kind 必须注册 worker | `t3-mutant-enqueued-worker` | `t3-registration-restored` |
| K04 | purge 必须实际清除过期业务行 | `t9-purge-negative-no-work` | `t9-purge` |
| K05 | purge 不得越过保留窗口 | `t9-purge-negative-cutoff` | `t9-purge` |
| K06 | 规模结果必须来自边界后的新任务 | `t9-purge-negative-old-boundary` | `t9-purge` |
| K07 | 访客可见性Grant保持本人范围 | `t10-mutant-16-own-only` | `t10-key-green-policy` |
| K08 | 创建者离开公开项目后不能沿创作者身份写入 | `t10-mutant-17-creator` | `t10-key-green-policy` |
| K09 | 每个项目写闸门并发上限不能多放一人 | `t10-mutant-18-gate-capacity` | `t10-key-green-policy` |
| K10 | 连续同作者描述活动合并而不是新增 | `t10-mutant-19-description-merge` | `t10-key-green-activity` |
| K11 | 列表排序统计不含已删链接 | `t10-mutant-20b-link-count` | `t10-key-green-list` |
| K12 | 数值游标不经float64丢失大整数 | `t10-mutant-21-cursor-precision` | `t10-key-green-list` |
| K13 | DNS任何一个候选为私网时不拨号 | `t10-mutant-22-dns-private` | `t10-key-green-policy` |
| K14 | 带凭据URL在解析阶段拒绝不查DNS | `t10-mutant-23-url-credentials` | `t10-key-green-policy` |
| K15 | 旧会话写确认不得发布到已退役工作项根 | `t10-mutant-24-session-retire` | `t10-key-green-session` |

## 2. 局部修复补充（不替代关键性质计数）

| ID | 性质 | 有效失败收据 | 恢复后通过 |
|---|---|---|---|
| U01 | 时长组合预算不能被整数溢出绕过 | `t10-mutant-07-budget` | `t10-mutants-green-config` |
| U02 | 有限极值排序仍输出有限中点 | `t10-mutant-08-midpoint` | `t10-mutants-green-app` |
| U03 | 周起始日确认保留独立的跨月周导航 | `t10-mutant-09-week` | `t10-mutants-green-app` |
| U04 | 访客个人筛选清空与工作项创建权限分开 | `t10-mutant-10-guest-filter` | `t10-mutants-green-app` |
| U05 | 访客泳道表头不显示创建入口 | `t10-mutant-11-swimlane` | `t10-mutants-green-app` |
| U06 | 单日期模式不受隐藏范围上限约束 | `t10-mutant-12-date-single` | `t10-mutants-green-app` |
| U07 | 默认背景命令保留文字颜色 | `t10-mutant-13-background` | `t10-mutants-green-editor` |
| U08 | 编辑器建议不消费输入法确认键 | `t10-mutant-14-ime` | `t10-mutants-green-editor` |
| U09 | Markdown粘贴只解析一次并保留字面符号 | `t10-mutant-15-paste` | `t10-mutants-green-editor` |

## 3. 未捕获项、证据修正与限制

- `t10-mutant-20-link-count`：单看排序未捕获墓碑计数，因为所有值都增加 1 后顺序不变。此轮不计有效；改用已有 `TestFlatIssueListCountOrder/link_count_limit2`，真实 count/key 为 1 而期望 0，形成 K11 的有效业务反证。未新增镜像测试。
- `20b` 控制脚本原来查找 `count/key` 文本，而实际失败写作 `flat position … Key:[49] … want 0`（49 是字符 `1` 的字节）。原始 result 的 false 保留，人工复核收据单列 `controller-verification.json`，不改失败日志或把脚本误匹配记为产品故障。
- K01–K06 是本次收尾 T2/T3/T9 已执行的独立反证；当前重新核对目标源码、实际覆盖测试、pgtest 探针与 purge 相关源码 hash，未重复运行未变的相同测量。R01 只修失败启动的清理，R02 不改变正常配置；原规模夹具正常路径等价。最终 F 仍独立执行。
- K07–K15 恢复后运行 access/gate/outbound 全包 race、描述活动 race、精确计数/游标 race、会话写确认 89 项；U01–U09 恢复后配置 race、应用 69 项、编辑器 29 项全部通过。7 份正常收据共同输入 `a3dbd198cfe275f23b1a464f149dedf470d0470c23717496a3ff3193e7fddb41`。
- 候选并非必须逐个变异；未抽中的原候选仍由各自真实正向、交错及错误测试覆盖，不计本表反证数量。DNS/凭据反证用测试本地服务器或受控 resolver，不向外部危险目标拨号。
