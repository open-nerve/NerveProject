# M3 工作区与项目设计：Codex 对抗性评审

## 1. 评审信息

| 项 | 内容 |
|---|---|
| 对象 | `.claude/worktrees/m3-design`，分支 `worktree-m3-design`，第二稿 `0d0f6ee56f08728cf2c6b7024d2045fae57582f8` |
| 第一稿 | `33fbc6a7`；两稿之间只有 `M3-design.md` 改动，1029 行增加、625 行删除 |
| 日期、评审者 | 2026-09-29；Codex。并行核查权限与接口、表结构与模块、前端与交接；报告作者复核发现及实验 |
| Nerve 基线 | 主项目保持 `main`，`f8cb7c22e2de97951c4dc17f36361ea22bb56f0e`；开始评审时指定 worktree 干净 |
| Plane 基线 | `/Users/xiaoruan/project/nerve-project/plane`，HEAD `02c19e1341d93141e8ad7b3278298adce208bafc`；另逐表核对 `tools/plane-schema/plane-v1.4.2-schema.sql` |
| 约束材料 | `v0-design.md`、`plane-diff.md`、`frontend-changes.md`、M2 设计的平台约定、M2 收尾 spec 附录 B、M3 的六份 handoff、M3 §17.1 首轮发现及修订 |
| 环境 | macOS Darwin 25.6.0 arm64；系统 Go 1.26.2，`go -C server version` 为 Go 1.27.1；Node 24.15.0；Apple Git 2.50.1；实验为 PostgreSQL 18.6，aarch64 Debian 镜像 |
| 改动范围 | 只新增本报告。没有修改设计、源码、配置或交接，没有切换主项目分支；按任务要求只提交本文件，不推送 |

下文 `M3:L` 指评审提交的 `docs/v0/M3-workspace-project/M3-design.md` 行号；`M2:L`、`总体:L` 分别指相应设计文档。`server/`、`web/`、`tools/` 路径均相对评审 worktree；`plane/` 指上表的独立只读源码目录。

证据严格分为三类：**已复现**限明确列出的实验，**读代码确认**指现有源码或快照，**设计推演**指尚未实现的 M3 规则组合。数据库实验实现的是设计的最小 SQL 模型，使用整数代替 UUID 便于观察，保留有关外键、成员状态和锁协议；它不是 M3 服务端或 HTTP 端到端测试。没有运行尚不存在的 M3 权限矩阵或页面验收。

### 1.1 实际执行的命令

只读检查包括 `git status --porcelain=v1`、`git rev-parse HEAD/main`、`git branch --show-current`、`git diff --stat/--name-only 33fbc6a7 0d0f6ee5`、两稿正文 diff、`rg`、`rg --files`、`nl -ba … | sed -n …`、`wc -l`，以及 Python 解析九个 `CREATE TABLE`、核对 Phase 任务数。读取了权限装饰器以外的视图内条件、序列化器和页面条件，没有只凭装饰器或设计引用作结论。

数据库只使用本次新建的容器，没有进入、停止、修改或删除任何已有容器：

```sh
docker ps --format '{{.Names}}\t{{.Image}}\t{{.Ports}}'
docker image ls --format '{{.Repository}}:{{.Tag}}'
docker run -d --rm --name nerve-m3-codex-review-20260929 \
  --network none --label codex.task=m3-adversarial-review \
  --tmpfs /var/lib/postgresql \
  -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.6
docker exec nerve-m3-codex-review-20260929 pg_isready
python3 /tmp/nerve-m3-codex-review.jngq4W/experiments.py
docker exec -i nerve-m3-codex-review-20260929 \
  psql -X -qAt -v ON_ERROR_STOP=1 -U postgres \
  < /tmp/nerve-m3-codex-review.jngq4W/logo.sql
```

容器未发布端口，没有挂载项目或已有数据库目录。实验结束后仅停止本次创建的容器，由 `--rm` 自动清理；脚本和结果留在上述临时目录。

另在临时目录做了 sqlc 对照及 Go overlay 实验，未改仓库文件：

```sh
CGO_ENABLED=0 go -C server tool -modfile=tools/go.mod sqlc compile \
  -f /tmp/m3-schema-review-bsvmqs0i/control.yaml
CGO_ENABLED=0 go -C server tool -modfile=tools/go.mod sqlc compile \
  -f /tmp/m3-schema-review-bsvmqs0i/cross_read.yaml
go -C server test -count=1 \
  -overlay=/tmp/m3-schema-review-bsvmqs0i/overlay.json \
  ./internal/modules/identity/adapter/http
go -C server test -count=1 ./internal/modules/identity/adapter/http
```

### 1.2 实验结果和边界

| 实验 | 实际结果 | 能证明什么 |
|---|---|---|
| S1：停用先枚举工作区，再等待正在接受的邀请；接受提交后，在停用提交前改角色、加入项目 | `initial_locked_workspace_ids=[]`；确认停用在邀请行上等待；另一请求读到账户仍有效；最终 `user_active=false, workspace_membership_active=false, project_membership_active=true, workspace_active_admins=0` | §3.6/3.9 的集合枚举与连带存在缺口，见 I-1 |
| S2：恢复成员后接受旧访客邀请 | A 离开前管理员数为 2；最终 `workspace_active_admins=0, workspace_role=5, project_role=20` | 无并发也能绕过管理员和访客角色保护，见 I-2 |
| S3：软删除项目未提交，另一会话用带 `deleted_at IS NULL` 的 `FOR SHARE` 取锁 | 确认等待；删除提交后返回 0 行 | 第二稿 M3:229 的重检说法成立 |
| S4：先持项目 N 锁，把第二状态设为默认；另一会话取同一个 N 锁后尝试守卫删除 | 删除 0 行；最终未删除状态 2 个，默认状态 1 个 | 第二稿的状态锁与默认守卫在此交错成立 |
| S5：按 M3:598 建 `logo_props` 临时表 | `{}`、`{"unexpected":true}`、`{"in_use":17,"emoji":[]}` 均插入成功 | 当前 CHECK 没有履行 M2 的键集合和值类型约定，见 M-5 |
| S6：sqlc 只装载工作区 schema，先查成员表，再联表 `users` | 前者退出 0；后者退出 1：`relation "users" does not exist` | 跨模块资料读取不能靠 workspace SQL 顺手 JOIN，见 M-1 |
| S7：只给加载后的 identity 契约增加两个 `sole_admin` 声明 | HTTP 包测试正文 `PASS`，TestMain 报两码未覆盖，退出 1；无 overlay 的同包测试退出 0 | bootstrap 的覆盖不能补另一个测试进程的记录，见 M-2 |

S1–S4 使用独立 `psql` 会话；阻塞通过 `pg_stat_activity.wait_event_type='Lock'` 确认后才释放对方，不以固定睡眠猜测交错。`experiments.py` 与 `results.txt` 保存完整 SQL 和输出。S5 的 SQL、输出为同目录 `logo.sql`、`logo-results.txt`。S6 及 S7 overlay 的命令和输出在另一临时目录的 `sqlc-result.txt`、`http-contract-overlay-result.txt`；无 overlay 的通过结果另见本次工具执行记录。

## 2. 抽查的具体说法

以下“符合”表示引用或规则有证据支持，不表示未来实现已经通过测试。

| # | 设计的说法 | 直接对照 | 判定 |
|---|---|---|---|
| 1 | M3:65 九张 Plane 表的列数 | 快照的 `CREATE TABLE`：workspaces:2563、workspace_members:2458、invites:2436、workspace_user_properties:2541、projects:2012、project_members:1913、project_user_properties:1972、states:2114、labels:1524 | **读代码确认**，依次为 14、17、13、14、36、16、15、18、15，准确 |
| 2 | 第 4 节的保留、删除及新增列 | 逐列对照快照、`plane/apps/api/plane/db/models/{workspace,project,state,label}.py` 和 `plane/apps/api/plane/db/mixins.py` | **读代码确认**，目标列数 10、10、11、10、23、11、11、14、12 成立；projects 删 14、加计数列 1。未发现遗漏的快照列 |
| 3 | M3:207–209 项目成员规则“照搬 Plane” | `plane/…/app/views/project/member.py:69–94,205–321`，特别是 234–238 的第二层管理员判断 | **读代码确认**，规则成立；已是成员时从“改角色”改为拒绝已明确登记 |
| 4 | M3:186 工作区管理员须同时是项目成员才能取得项目管理权限 | `plane/…/app/permissions/base.py:53–78`；`web/apps/web/core/store/user/permissions.store.ts:123–130` | **读代码确认**，两侧都先要求项目成员关系。首轮 M27 的修订有效 |
| 5 | M3:197–201 五项“服务端比页面宽松” | Plane project/base.py:314–347,382–397,428,436；state/base.py:46,61,105,113；issue/label.py:26–40；项目设置包装层、状态设置、member-select 的页面限制 | **读代码确认**，收紧的依据存在；不能把这些已登记差异重复列为缺陷 |
| 6 | M3:203 访客看不到成员邮箱，与 Plane 相同 | `plane/…/views/workspace/member.py:50–54`、`serializers/workspace.py:93–110` | **读代码确认**，成立，首轮 I9 已纠正 |
| 7 | M3:191/437 已归档项目不出现在状态列表 | `plane/…/views/workspace/state.py:20–27`、`views/state/base.py:34–39` | **读代码确认**，成立；状态写入的路径并不沿用这个归档过滤，允许写入也没有误读 |
| 8 | M3:228–229 锁父行后重读、锁条件重检软删除 | 实验 S3 | **已复现**，成立 |
| 9 | M3:279 “先邀请、后成员”足以处理停用与接受 | 实验 S1；M3:327–330 固定了先前枚举的工作区集合 | **已复现**，只能看见新的成员行，补不回工作区锁、管理员检查和项目连带范围，I-1 |
| 10 | M3:290 不能改自己保证工作区至少留一位管理员 | 实验 S2；Plane invite.py:188–195、管理命令:63–67 | **已复现**，遗漏接受旧邀请的角色覆盖，I-2 |
| 11 | M3:298 用与刷新令牌相同方式派生独立 MAC | `server/…/identity/adapter/signing/keys.go:16–17,52–57`、`mac.go:21–33` | **读代码确认**，已有实现确为 HKDF(seed, info) 和 16 字节 HMAC、`hmac.Equal`；新用途的独立 info 合理。没有把“读过算法”写成完整密码学证明 |
| 12 | M3:311/1222 访问日志不记录查询令牌 | `server/internal/platform/httpserver/middleware.go:117,165` | **读代码确认**，当前两处记录 `r.URL.Path`，不含 RawQuery；不据此保证反向代理、浏览器历史或未来错误信息也不记录 |
| 13 | M3:312 接受须 token 和邮箱，Plane 也检查邮箱 | `plane/…/views/workspace/invite.py:152–174` | **读代码确认**，成立；无邮箱验证下，对尚未注册受邀人的冒用风险仍如决策点 1 所述 |
| 14 | M3:359–363 恢复命令照 Plane，项目成员不会自动恢复 | `plane/…/management/commands/reactivate_workspace_member.py:51–90` | **读代码确认**，成立；恢复可发生在账户激活之前，端口契约需要相容，M-1 |
| 15 | M3:403–407 两个负责人 FK 改 SET NULL，owner 可删 | `plane/…/db/models/project.py:77–90`、workspace.py:131；快照及保留前端使用方 | **读代码确认**，CASCADE 来源真实，删除关系图与保留 FK 相符 |
| 16 | M3:413–420 标签不分大小写、两层、删除父标签连带 | Plane serializers/issue.py:361–386，标签拖放组件与 `label-utils.ts`，Label 模型的 parent CASCADE | **读代码确认**，依据成立；锁项目维护层级不变式的修订正确 |
| 17 | M3:424/430 删除 `is_triage`、新状态 sequence 为非分诊最大值加 15000 | `plane/…/db/models/state.py:24–68,117–128`，迁移 0063，创建项目 bulk_create 路径 | **读代码确认**，第二稿说法成立，不能再按第一稿问题报错 |
| 18 | M3:433–436 状态写入先锁项目、默认用两条语句 | 实验 S4 | **已复现**，此交错保住一个默认状态；未重新声称重跑了所有组与标签交错 |
| 19 | M3:598 logo_props 仅检查对象类型，内部形状交接口 | 实验 S5；M2:641、plane-diff.md:66 | **已复现**，仍缺上级要求的键集合和值类型，M-5 |
| 20 | M3:975–986 两段组合无环 | 现有 CredentialLock 依赖，按列出的构造顺序逐项展开 | **读代码确认＋设计推演**，现列依赖可无环；缺的是读取能力，M-1 |
| 21 | M3:852 两个跨模块错误码由 bootstrap 测试返回 | `apitest/problems.go:92–144`、HTTP TestMain；实验 S7 | **已复现**，不能据此满足 identity HTTP 包自己的双向门禁，M-2 |
| 22 | M3:1075 旧 ProjectService 只剩两个属性方法 | project.service.ts:119–130；三个保留搜索组件 | **读代码确认**，漏了 `projectIssuesSearch` 的去处，M-3 |
| 23 | M3:97 用错账户一打开就不提供接受 | M3:810 的公开结构、1084–1087 的 publicClient 和错误流程 | **设计推演**，数据不足，验收与流程矛盾，M-4 |
| 24 | M3 §7.11 已按 M2 收尾校准规模 | `M2-auth/specs/closeout.md:529–578`，M3 的文件组、测试、横向清理及 fixture 分项 | **读代码确认**，估算口径改善、约 380 文件加总无首轮错误；不是对未来工时的保证 |
| 25 | M3 §12 每段至多约 16 个任务，故事前置数据可经 API 准备 | 任务数 15/12/14/16/14/8/13/16/13/15/9；W/P 故事与 P1–P7 操作顺序 | **读代码确认＋设计推演**，现分段符合任务数上限，没有找到仍不可准备的故事；plan 尚不存在，不能验证其 1500 行上限 |

## 3. 发现列表

### Critical

未发现。

### Important

#### I-1｜停用的工作区集合在等待邀请时仍会增长，漏掉管理员检查和项目连带

**位置：** §3.6，M3:238、252、254、279；§3.9，327–330；关联 §3.7，285。

**证据：已复现（S1，Postgres 规范 SQL 模型）＋读代码确认。** 停用先按当前有效成员关系枚举并锁工作区，之后才删除邀请。接受邀请锁工作区和邀请，但不锁接收账户，因此能在枚举结束后增加新的工作区成员关系。M3:279 只证明后续 `UPDATE workspace_members` 的新快照能看见它，未证明先前的锁集合和检查也完整。

可复现交错如下。B 有一枚既有、有效、未过期的 PAT；B 尚不属于工作区 W；A 是 W 的管理员，W 有一个项目 P；B 持有加入 W 的管理员邀请 I。

| 步骤 | 接受邀请 A1 | 停用 D | B 的另外两个正常请求 |
|---|---|---|---|
| 1 | 锁 W（N）、I（FOR UPDATE），验证邀请 | | |
| 2 | | 锁 B 的账户并改成 inactive，尚未提交；枚举有效工作区得到空集合，工作区管理员检查为空 | |
| 3 | | 软删除 B 邮箱的邀请，等待 A1 的邀请行锁 | |
| 4 | 新建 B→W 的有效管理员关系，软删除 I，提交 | 等待结束；随后成员 UPDATE 尚未执行 | |
| 5 | | | 以既有 PAT 发新请求。未提交的停用仍读到 `users.is_active=true`；B 锁 W 后把 A 改成普通成员，合法，因为目标不是自己 |
| 6 | | | B 锁自己的有效工作区成员行（S）、P（N），加入 P 成为管理员，提交 |
| 7 | | 停用 B 的全部工作区成员关系；调用 `EndMemberships` 时传入步骤 2 的空集合；提交 | |

实测结果为 B 的账户和工作区成员关系无效、项目成员关系仍有效、W 的有效管理员数为 0。去掉步骤 5 也会单独留下“项目有效成员不再是工作区有效成员”的反例。

现有认证不会隐式阻塞步骤 5：`server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql:24–29` 对 user/PAT 作普通 SELECT；`identity/app/deactivate.go:96–108` 保留 PAT；`authenticate.go:125–139` 检查的是读到的 UserActive。PAT 的 last_used 写入也不取账户锁。这里没有声称停用后仍能发新请求，而是在停用事务提交前利用了合法的旧快照。

**为什么重要：** 这是第二稿明确承诺的不变式失败，不是一般的请求延迟。工作区会无人管理，项目成员子集不变式也不成立。首轮 I1/I7 修的是既有父行上的竞争，这次是此前根本没进入锁集合的父行。

**建议：** 接受邀请最先以 S 锁接收账户，并在锁内重读账户有效状态、当前邮箱，锁持有到事务结束，再按原顺序锁工作区和邀请。接受先拿到账户锁时，停用必须等接受提交，随后枚举包含新工作区；停用先拿锁时，接受等到后读到 inactive 并拒绝。创建工作区、恢复成员已有账户锁，此修改使三条增长工作区成员集合的路径采用同一协议。是否还要求复核凭证撤销状态可沿用 `CallerLock` 裁定，不能把仅重查 active/email 称为完整凭证复核。不要在锁过成员/项目后再补锁新工作区，那会破坏全局顺序。P6 的交错测试必须覆盖“原先不是此工作区成员”的接受，而不只覆盖已有工作区。

#### I-2｜恢复成员之后接受旧邀请，可以把唯一管理员改成访客且不降级项目角色

**位置：** §3.8，M3:313；§3.11，359–363；§3.6，252；与 §3.7，290 的保证冲突。

**证据：已复现（S2，顺序 SQL 模型）＋读代码确认。** 无需并发：

1. A、B 原为工作区管理员，尚无项目；A 移出 B，再给 B 发一份访客邀请。
2. 服务器管理员执行 `reactivate-member`，B 恢复为管理员，旧邀请仍未回应。
3. A 离开；此时有两名管理员，合法，离开时也没有项目连带可阻止。
4. B 创建项目，成为项目管理员。
5. B 接受旧访客邀请。按 §3.8，把已有成员关系设为有效并把角色覆盖成 5。

最终 `workspace_active_admins=0, workspace_role=5, project_role=20`。这条路径既没有“不能改自己”的检查，也没有调用 `DemoteToGuest`。

Plane 确实有这个组合：`plane/apps/api/plane/app/views/workspace/invite.py:188–195` 对存在的成员无条件改角色，不区分已有效与已离开；`db/management/commands/reactivate_workspace_member.py:63–67` 只恢复成员。因而“照搬 Plane”在此不能证明 Nerve 的额外不变式成立。

**为什么重要：** 即使 I-1 的锁修好，这个纯顺序问题仍存在。工作区管理员保底及“工作区访客不能拥有高项目角色”均可被邀请路径绕开。

**建议：** 在接受邀请时明确“已经是有效成员”的分支：推荐保留现角色并消费邀请，或用一个明确冲突拒绝接受；也可让恢复命令撤销过时邀请，但仍应定义有效成员再次接受的通用规则。若坚持覆盖角色，必须复用改角色及访客连带的全部规则，不能直接写 role。把恢复→离开→接受旧低角色邀请及项目角色两项加入验收，差异清单登记对 Plane 缺陷的修正。

#### I-3｜条件性决策风险：允许成员列出全部邀请，会暴露管理员邀请并形成提权路径

**位置：** 第 10 节决策点 4，M3:1438–1446；§9.2，1301；关联 §5.2，809 和 §3.8，317–320。

**证据：读代码确认＋设计推演。此项只在选择 B 且不补邀请读取范围时成立，推荐的 A 没有这个问题。** B 目前列出的保护是创建、修改邀请不能超过自己的角色；矩阵只把普通成员四个邀请操作改成允许。但 `WorkspaceInvitation` 含邮箱和可用 token。

假设管理员已邀请尚未注册的邮箱 X 为管理员。普通成员通过邀请列表得到 X 的邮箱和 token，再按决策点 1 的 A+(a) 以 X 注册、接受邀请，即取得一个管理员账户。攻击全程不创建、不修改那份邀请，角色上限检查不会触发。邮箱没有验证正是 M2 交接要求令牌的原因。

Plane 的读取范围也确实没有提供额外保护：`app/views/workspace/invite.py:39–49` 仅按 workspace slug 过滤；`app/permissions/workspace.py:66–71` 允许成员；`app/serializers/workspace.py:117–136` 返回全部字段和带 token 的链接。不能把“照 Plane 服务端”当作安全论证。

**为什么重要：** 负责人现在看到的 B 代价是“四行规则、一个检查、一个码和显示条件”，遗漏了对成员开放邀请凭证的权限边界。它不是仅让成员多一个邀请按钮。

**建议：** 保留 A 建议；若负责人选择 B，先裁定成员能读取、复制、修改、删除哪些邀请，尤其不得取得超出授权范围的管理员邀请凭证。列表、详情/修改返回结构及复制路径执行同一范围，测试上述未注册邮箱场景。把这个必要条件和代价写进选项表，再让负责人比较。

### Minor

#### M-1｜跨模块读端口未覆盖成员资料与恢复命令，Accounts 的有效性限制自相矛盾

**位置：** §6.5，M3:965–966；§6.6，982–990；关联 §5.1，744、§5.2，804–806、§3.11，359–364。

**证据：读代码确认＋设计推演；sqlc 边界另经 S6 复现。** 成员列表必须包含已无效的成员并内嵌姓名、邮箱等资料；现列出的账户资料读取能力只有按单个 id/邮箱锁“有效账户”的 `Accounts`。它既不是无锁批量资料读取，也无法返回停用账户。恢复命令又承诺允许恢复停用账户的成员关系、打印需要 activate 的提示，还要统计剩余无效项目成员数，三个 `ProjectCascade` 方法没有这个读取能力。

直接从 workspace JOIN users 不是现架构的实现补丁：`server/internal/archtest/rules_test.go:203–209` 禁止模块间导入，M2 的 sqlc schema 按模块装载；S6 的联表查询得到 `relation "users" does not exist`。现有 `identity/adapter/postgres/users.go:30` 的读取返回 identity 自己的领域类型，也不能直接拿来跨模块导入。

**为什么重要：** 详细组合图还不足以实现已经承诺的响应和命令；实现者容易因此扩 sqlc 范围、在 GET 中逐个锁账户，或意外拒绝停用账户的恢复路径。

**建议：** 补无锁批量公开资料读取端口；账户锁返回状态，由创建用例要求 active、恢复用例按已选语义处理 inactive；补项目成员统计读取。使用跨边界小值对象或组合根转换，保持模块边界。补这些依赖不要求推翻两段构造，也不要求把全部内部方法写成设计文档。

#### M-2｜跨模块错误码在 bootstrap 返回，不能满足 identity HTTP 包自己的契约门禁

**位置：** §5.3，M3:852；§9.4，1346；P6 完成线，1639。

**证据：已复现（S7）＋读代码确认。** `server/internal/platform/httpserver/apitest/problems.go:92–106` 的 answered 是当前进程的记录；128–144 在本包 TestMain 检查。`identity/adapter/http/handler_test.go:26` 自己调用 `apitest.Main(m, "identity")`，bootstrap 是另一个测试二进制。

只通过 Go overlay 在 `loadModule` 后把 `deactivateMe` 的空声明改成两码，不改任何仓库文件，目标 HTTP 包输出：

```text
PASS
api/modules/identity.yaml declares problem codes that no test answered through CheckResponse:
  deactivateMe: project.sole_admin
  deactivateMe: workspace.sole_admin
FAIL
```

无 overlay 的同一包测试通过。这个实验验证门禁的作用域，不声称已经实现 M3 的错误路径。

**为什么重要：** 按“由 bootstrap 返回”作为全部覆盖工作，P6 的双向契约完成线达不到。首轮 I4 的错误前缀修订是正确的，这里是另一项进程范围问题。

**建议：** bootstrap 保留真实组合检查；identity HTTP 包用已有 `fakeDeactivate.err` 增加两条 409 透传测试并走 `CheckResponse`。不放宽门禁，不跨模块导入业务实现。

#### M-3｜旧 ProjectService 缩到两个方法会丢失保留组件的搜索依赖

**位置：** §7.3，M3:1074–1075；关联 §7.10，1158、1165 及 §13.2，1795。

**证据：读代码确认＋设计推演。** `web/apps/web/core/services/project/project.service.ts:119–130` 的 `projectIssuesSearch` 仍被以下保留组件调用：

- `core/components/issues/parent-issues-list-modal.tsx:67–73`；
- `core/components/core/modals/existing-issues-list-modal.tsx:102–104`；
- `core/components/inbox/modals/select-duplicate.tsx:61–65`。

这些路径属于后续 M，但仍需编译。M3:1075 说旧 service “只剩”两个用户属性方法，没有给搜索方法或上述调用方去处；按字面删除会得到缺方法的类型错误。M2 的交接其实已按调用领域把搜索留给 M4/M7，见 M3 handoff `M2-closeout.md:33`。

**为什么重要：** P8 要在迁移后全体调用方能编译；这是跨里程碑边界遗漏，不是要求 M3 提前实现工作项搜索。

**建议：** 允许旧 service 保留该方法，或将其及调用方迁到归属 M4/M7 的旧搜索 service，并在交接里写清。把“只有两个方法需要某条关键词例外”与“类总共只剩两个方法”分开。

#### M-4｜W5 要求打开时识别错误邮箱，但公开预览没有所需信息

**位置：** 第 2 节 W5，M3:97；与 §5.2，810、§7.4，1084–1087 对照。

**证据：读代码确认＋设计推演。** W5 要求用另一个账户打开即显示邮箱不符、不提供接受；公开 `InvitationPreview` 没有邮箱或匹配标志，且页面用 `publicClient` 查询。两个登录账户对同一邀请得到相同预览，页面无法区分。§7.4 的实际机制是点接受后由 POST 返回 403。当前旧页面的邮箱来源是 `web/apps/web/app/(all)/workspace-invitations/page.tsx:51` 的预览 email，新契约已删除它。

**为什么重要：** W5 的页面完成线无法按现有契约实现，可能诱使实现者把邮箱重新塞进预览，破坏决策点 1 的选择。

**建议：** W5 改为“尝试接受后收到 403，再说明、隐藏或禁用接受并提供退出”，与 §7.4 一致。无需增加接口，也不需要返回被邀请邮箱。

#### M-5｜logo_props 的数据库约束没有遵守 M2 的对象 JSON 约定

**位置：** §4.6，M3:598；§5.2，813；上级 M2 §3.13，641、`plane-diff.md:66`。

**证据：已复现（S5）＋读约束确认。** M2 要求对象 JSON 在数据库检查对象类型、键集合和值类型；M3 只检查 `jsonb_typeof(logo_props)='object'`，把内部形状交接口。实测未知键、数字 `in_use`、数组 `emoji` 都通过。首轮 M8 裁定所有字段可选、`{}` 合法，并不等于允许未知键和错误类型。

**为什么重要：** 这是未登记的上级约定例外。接口有封闭结构，因此不把它评价为已经可经 API 写入坏数据的漏洞。

**建议：** 用允许字段缺省的 CASE/CHECK 校验键集合与出现时的类型；或者明确登记数据库仅保证对象类型的例外及理由。同步 §3.20 的上级修订安排。

## 4. 对决策点和架构问题的意见

### 4.1 四个产品决策

| 决策 | 意见 |
|---|---|
| 1：邀请凭证及关闭注册时自助注册 | A+(a)、预览不显示邮箱是可理解的取舍，第二稿已正确承认链接保密是主要控制。邮箱不是第二个已验证因素；拿到链接后，注册请求的允许/拒绝也能验证一个邮箱猜测。建议保留当前诚实的风险描述，明确预览隐藏邮箱只减少直接披露。A+(b) 的运维代价已列出；若将来从 (a) 改 (b)，代码改动虽小，已经因旧邀请注册的账户不会自动消失，需要另外决定是否处理存量 |
| 2：只保留链接 | 推荐 A 的理由成立。系统内按未验证邮箱列邀请，不应返回接受凭证。B 仍可作为纯通知列表，但确实增加低收益界面；没有发现必须保留第二条路的现有功能约束 |
| 3：访客个人主页 | A 与 Plane 页面一致，按此交给 M4 合理。若改选 B/C，“以后只改一行规则和页面判断”偏乐观：工作项查询还必须按项目可见性、`guest_view_all_features` 过滤并测试，不能用开放入口代替数据授权。这个工作已在 C 的选项格隐含，应在代价段说清 |
| 4：谁能邀请 | 支持 A。选 B 前必须修正 I-3 的凭证读取范围；“是否能改删别人邀请”不是唯一待定范围。修正后再比较成本，不能仅按四行规则估算 |

除此之外，没有找到必须由负责人重新裁定、却被默认为已批准的重大产品选择。创建工作区命令的 `--admin-email`、恢复命令、保留路由名单均有已写出的依据或登记。I-2 的“已是有效成员再接受邀请”需要补一个明确语义选择，它属于此前漏掉的状态转换。

### 4.2 七个架构问题

| 架构项 | 意见 |
|---|---|
| 11.1 派生邀请 MAC | 现有 signing 足以承载；域分离、128 位截断、常量时间比较和换钥失效的描述合理。独立邀请密钥的收益主要是独立轮换和故障范围，不应绝对描述为只有“JWT 换钥时链接不失效”。它仍是可以不选的运维权衡 |
| 11.2 加锁顺序 | 大方向和首轮修正正确；I-1 说明锁顺序正确不等于锁集合完整。接受邀请补接收账户锁后，再把新协议提升为后续 M 的规则 |
| 11.3 不分页、成员内嵌资料 | 已把两项与上级的差异显式交负责人；未来分页会影响客户端的说明正确。无硬集合上限，“由管理员决定大小”不等于存在技术上限；若接受 v0 例外，把大工作区响应大小和加载耗时留作实际验收即可，不凭没有基准的数据断言性能不合格 |
| 11.4 shared 纯取值规则 | 邮箱、网址检测、时区有真正的第二使用方，没有看到业务流程迁入 shared。时区继续使用既有 `time.LoadLocation`，已有文件读取例外，严格说不是完全无环境依赖的纯函数；不需要因此为它新增远程式端口 |
| 11.5 项目级标签 | 已回查保留的创建入口，未找到需要 project_id 为 null 的页面路径；修改上级唯一范围的安排明确。M7 的跨项目标签列表不意味着标签必须属于工作区 |
| 11.6 Provide/New | 当前给出的顺序可无环，CredentialLock 的存储依赖例外合理。按 M-1 补数据读取能力后再做组合检查；没有证据要求事后注入或双向模块 import |
| 11.7 错误码前缀 | 跨模块业务拒绝需要此例外，`forbidden` 平台化合理；保留“前缀属于已存在模块”的约束。M-2 修的是覆盖测试位置，不是要求撤回这个架构选择 |

### 4.3 前端实施时必须落实的两条现有约束

下面有直接代码风险，但 §3.1 和总体 §7.7 已经给出正确的总规则，因此不再把“设计没有逐文件重复规则”计成新的 Important。建议 P8–P10 spec 明列这些路径并加针对性验收：

1. **展示权限必须限制取数。** `workspace/settings/members-list.tsx:46–55` 无条件先取邀请、再取成员；79 行只控制显示。普通成员可以进入成员页，新矩阵却让邀请列表返回 403。需要只对管理员请求邀请，并使成员取数独立。`layouts/auth-layout/project-wrapper.tsx:70–95` 同时启动详情、偏好、标签、成员和状态；非成员的公开项目详情返回 200/`member_role:null`，后四项则都是 403。应等到确认有效项目成员再启用子资源请求，测试普通成员进入成员页及非成员直达项目加入页没有多余失败请求。不能仅把 fetcher 改成新地址。
2. **成功的迟到响应也要检查会话。** `project/delete-project-modal.tsx:62–69`、`workspace/delete-workspace-form.tsx:67–75` 在 await 后直接导航、提示。`SessionChangedError` 不覆盖已发出请求的迟到 200；旧 store 隔离也不会取消组件的旧闭包。按总体:593，在页面级副作用前检查 `inSession()`，增加“发出修改→另一标签页切账户→旧请求成功返回”的测试。M3 §7.1 引用了这条总规则，应落实到页面迁移，不能只测 store 换代。

## 5. 查过但没有发现问题的地方

### 5.1 权限、身份与状态

按 §9.2 的每个操作组，分别核对了 Plane 权限类、装饰器、视图内追加判断和保留页面。未发现第二稿矩阵中尚未登记的静态授权放宽：工作区有效性、前成员、项目有效成员、工作区管理员的项目例外、访客邮箱、私密项目、归档状态均有规则落点。停用账户在现有 identity 的 JWT/PAT 认证阶段被拒绝，下一请求的检查不依赖前端。

特别没有把以下情况当成新缺陷：项目管理员不能提拔同级管理员；移出项目成员比较原始项目角色时不额外给工作区管理员放宽；工作区管理员可看未加入项目但要先 join 才能管理；归档项目可读、状态列表为空而部分写入仍允许。它们分别有代码依据或已明确登记差异。

静态矩阵正确并不覆盖成员身份如何到达一个非法组合，I-1/I-2 正是转换路径的反例。本次没有把约 400 格全部实际运行一遍。

### 5.2 数据、事务与邀请令牌

九表逐列增删、保留类型、默认值来源、审计 FK、部分唯一索引、4.12 删除关系图均核对过。物理级联涉及的 workspace/project/parent 外键已有无条件索引；负责人 CASCADE 改 SET NULL 已登记；`profiles.last_workspace_id` 不加 FK 的读取交集理由成立。除 M-5 外，未找到需要另报的数据模型偏离。

项目 N 锁串行化标签集合和状态集合，修掉首轮标签成环与默认竞争的方向正确；S3/S4 对照通过。上下文共享事务有实际基础：`server/internal/platform/postgres/tx.go:26` 从 ctx 复用事务。未在现定的其余锁路径上找出另一个确定的死锁反例；不把“未找到”写成全局无死锁证明。

邀请 MAC 不存数据库、独立 info、删除后令牌失效、接受/忽略经邀请行锁单次回应，未发现新的密码学缺陷。注册关闭的有效邀请路径只给匹配邮箱开户，注册不自动接受的产品选择明确；链接与邮箱一同泄露可以抢先注册已写入决策点 1。公开预览的工作区名称、slug、角色及忽略状态是明示的披露；无效令牌统一 404、邮箱不符不返回目标邮箱，未找到除此以外确定的敏感字段泄露。

### 5.3 Phase、规模与六份交接

P1–P11 共 145 个任务，后端 92、前端 53，单段 8–16 个。P3 提供邀请接受后，P4 才准备多成员项目；成员退出/恢复故事在 P5，停用在 P6，状态拒绝在 P7。没有找到首轮 I2 那种故事前置数据尚无接口准备路径的问题。

规模估计已把测试、横向调用方、死代码清理、E2E fixture 分开，对照 M2 附录 B 的方向正确。P8 同时触及数据层、类型和大量调用方，仍是写 plan 时最需要测量的一段；当前没有 plan，不能证明一定超限，也不能因为表里写了 16 个任务就证明不会超过约 1500 行。

六份交接按原内容对照 §13.1，落点如下；“有落点”不是本次把 handoff 关闭：

| 交接 | 核对的项目 | 结论 |
|---|---|---|
| M2-closeout，14 节 | 邀请、落点、分代、last_workspace FK、创建开关、停用、可空引用、模块边界、删除图、表情数据、旧调用/类型、页大小、浏览器核对、下拉框/复制 | 均在 M3:1752–1765 有对应正文及 Phase。§1/§6 的关闭仍受 I-1/I-2/M-2 影响；§3 的 10 处 service、reaction 释放、loginId 已安排；无分页所以页大小交给 M4 符合原关闭条件 |
| M1-closeout | 死成员/prop、误归 41 行、oxlint、全仓规则清零、下调 cap | §7.9、P11/收尾有安排；最终仍须逐条消除或说明，不能只因“归别的 M”自动标误报 |
| M1-P2-trim-content | 项目字段、侧边栏偏好、保留名、个人主页 | §3.10/3.18、第 4/5 节及决策 3 有落点；个人主页的工作项 API 明确交 M4 |
| M1-P3-trim-platform | 删除字段/动态、项目成员、保留名、地址 | 项目部分在 P4/P8；视图和收集箱继续由 M7 同名交接约束，未被 M3 悄悄关闭 |
| M1-P4-router-native | 保留名来源、离开项目先成功后跳转 | §3.10、§7.6、P10 有落点；页面成功后的会话检查还须照总体 §7.7 做 |
| M0-P3-pagination-components | 已完成公共分页组件的状态 | 当前 done 有依据，M3 无分页使用者，收尾记 closed 的安排合理 |

表情选择器还核对了本机 frimousse 0.3.0 的地址拼接：给定 `emojibaseUrl` 后读取 locale 下 `data.json`、`messages.json`，设计的本站路径可行。没有发现需要放宽 CSP 的理由。没有运行真实浏览器或 M3 E2E，因此不把这项称为页面验证通过。

## 6. 结论

**第二稿仍需修订后进入相关实现：Critical 0；Important 2 个已复现的不变式问题，另有 1 个仅在选择决策点 4-B 时成立的提权风险；Minor 5。**

最需要先处理的是 I-1 的成员集合增长与 I-2 的旧邀请角色覆盖。它们都能绕过第二稿承诺的保护，且不属于 §17.1 已修发现的简单重复。建议先明确接受邀请的账户锁和“已是有效成员”语义，再写 P3/P6 spec；决策点 4 如保留 B 为可选项，必须补 I-3 的读取范围及成本。

权限矩阵的静态规则、九表大部分映射、两段构造、Phase 前置顺序和六份交接的落点，没有发现需要推翻的整体方向。报告没有改设计替负责人作决定；上述修正和决策仍由设计负责人落实。
