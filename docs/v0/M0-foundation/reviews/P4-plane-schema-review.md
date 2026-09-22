# M0/P4 Plane 表结构快照：评审记录

| 项 | 内容 |
|---|---|
| Phase | M0/P4 `plane-schema` |
| 日期 | 2026-09-22 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P4-plane-schema.md) / [plan](../plans/P4-plane-schema.md) |
| 分支 | `worktree-m0-p4-plane-schema`，从 `main` 的 `cd2fada` 分出，提交从 `622859e` 到本评审记录所在的提交 |

## 1. 范围与结果

P4 提供 Plane v1.4.2 在空库上跑完自带的全部 Django 迁移之后的完整表结构快照，作为以后各个 M 建表时"照搬 Plane"的依据（spec 1）：

- 临时环境 `tools/plane-schema/compose.yaml`：Postgres 15.7 和 Plane v1.4.2 后端镜像，两个镜像都写成"标签@摘要"。
- 提取脚本 `tools/plane-schema/extract.sh`：只需要 Docker，从零生成快照，重新运行得到逐字节相同的结果。
- 快照 `tools/plane-schema/plane-v1.4.2-schema.sql`：11707 行，379933 字节，SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70`，已提交到仓库。
- `tools/plane-schema/README.md`：来源、内容、重新生成、升级方法。
- `make plane-schema`。
- `docs/v0/plane-diff.md`：按快照核对并更新。

M0 §8 P4 的三项验收标准全部满足：重新运行得到一致的快照；快照中能找到总体设计 5.2 列出的全部 44 张表；差异清单中的相关说明已更新。

## 2. 原型验证的结论

- **官方镜像可用**：`makeplane/plane-backend:v1.4.2` 在 amd64、arm64 上都能拉取运行；迁移只需要 Postgres 和一个非空的 `REDIS_URL`（客户端只在第一次执行命令时才连接，迁移全程从未连接）。不需要"从源码用 Python 执行迁移"的备选方案（spec 2.2）。
- **确定性**：从零完整运行 6 次（其中 1 次先删除本机镜像重新拉取，1 次用 amd64 镜像），输出逐字节相同；Task 执行期间和修复轮又各重新生成一次，SHA 始终是 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70`（spec 2.2；progress.md 的 fix wave 记录）。
- **44/96/110 的表检查**：总体设计 5.2 列出的 44 张保留表全部存在，名字全部正确；96 张业务表（44 保留 + 52 不保留）加 14 张系统表，正好是快照中的 110 张表，没有遗漏也没有多出（spec 2.7）。
- **标签与 preview 的关系**：`v1.4.2` 标签指向 `5f7d927`，仓库外参考源码 `plane/` 是 `preview` 分支的 `02c19e1`，比标签多 63 个提交；两者的迁移目录（128 个迁移文件 + 2 个 `__init__.py`）逐字节相同（SHA-256 `85e5c612…`），所以快照同时是两个提交的表结构（spec 2.3）。

## 3. 各 Task 的评审

| Task | 内容 | 提交范围 | 核实方式 |
|---|---|---|---|
| 1 | `compose.yaml`、`extract.sh`、快照 | 622859e..2d90bf5 | 实现者 sonnet；评审（sonnet），clean：文件哈希与 brief 一致，摘要写死，不映射主机端口，tmpfs，清理只作用于本项目 |
| 2+3 | `tools/plane-schema/README.md`、根 README、`docs/v0/plane-diff.md` | 2d90bf5..5eab98d | 批处理，实现者 haiku；一次评审（sonnet），clean：快照相关的事实逐条用 grep 复核，链接全部可解析 |

## 4. 整分支评审（opus）：With fixes

| # | 类别 | 发现 | 处理 |
|---|---|---|---|
| Important 1 | Important | `extract.sh:47` 的 `compose run` 在真实终端中运行时会分配 TTY，Ctrl-C 被转发进容器，脚本以退出码 1（"the Plane migrations failed"）结束，而不是 130 | 已修（`0e223bb`）：加 `-T`。修复后用 `script(1)` 分配的伪终端，在迁移进行到约 30 秒时按 Ctrl-C 核实：退出码 130，容器、网络、临时文件都没有留下 |
| Minor 2 | Minor | README 的升级步骤应加一步：grep 并更新引用 `plane-v1.4.2-schema` 的地方，包括差异清单、总体设计、前端改动清单里的基线行 | 已修（`0e223bb`） |
| Minor 3 | Minor | 从未检查 Compose 版本：Compose 2.0–2.21 时错误只说"pulling the images failed"；根目录 README 只写"Compose v2" | 已修（`0e223bb`）：拉取失败的说明补充"needs Compose 2.22 or later"；根 README 注明 Compose 2.22 或更高 |
| Minor 4 | Minor | 没有 `.gitattributes`：Windows 上开启 autocrlf 会把快照转成 CRLF，SHA-256 核对失效 | 已修（`0e223bb`）：新增 `.gitattributes`，`tools/plane-schema/*.sql -text linguist-generated=true` |
| Minor 5 | Minor | 导出的 `CDPATH` 含 `.` 时会破坏 `here=` 的赋值 | 已修（`0e223bb`）：改为 `cd -- … >/dev/null` |
| Minor 6 | Minor | 恢复提示缺少 `-f compose.yaml`；`kill -9` 留下的 `.tmp` 文件在下次运行时不会被清理 | 已修（`0e223bb`）：恢复提示补上 `-f`；脚本开始时先 `rm -f "$tmp"` |
| Minor 7 | Minor | 措辞：迁移目录"130 个文件"其实是 128 个迁移文件加 2 个 `__init__.py`；amd64 的验证是在 arm64 上模拟的；spec 的命令列表漏列 `dirname` | 已修（`0e223bb`）：spec §2.2/§2.3/§2.5 的措辞更正 |
| Minor 8 | Minor | spec 缺口：Ctrl-C 的验证方式在 spec 里没写清楚（对应 Important 1）；§3 的收尾清单缺了 `docs/v0/frontend-changes.md:3` 的基线行，M0 §10 L479 的"M1 或 M2"已经过时（P4 还要移交给 M4 和 P5） | spec 文字已修（`0e223bb`）；文档缺口在本次收尾处理：M0-design、v0-design、frontend-changes 的同步，以及 M2/M4/M0-P5 三个 handoff |

### 4.1 修复后的复核（sonnet）

复核范围是修复提交 `0e223bb`（`5eab98d..0e223bb`）。结论：第 1–8 项全部处理到位（第 8 项中属于文档收尾的部分，在本次收尾完成），修复本身没有引入新问题。
- 复核者独立核实：`bash -n` 通过，`extract.sh` 仍是 `100755`；`.gitattributes` 对快照生效（`text: unset`、`linguist-generated: true`）；设置 `CDPATH=.` 时 `here` 仍然正确；快照 SHA-256 仍是 `4080c81e…`，11707 行。

## 5. 控制者裁定

| Ruling | 内容 |
|---|---|
| 1 | spec §3 第 1–12 项按原样接受：镜像按标签@摘要写死、直接执行 `manage.py migrate` 并用虚拟 `REDIS_URL`、不实现 Python 备选方案、快照不过滤、README 的"生成时间"是内容最后变化的日期、`make plane-schema`、持续集成不运行提取也不加检查快照的测试、系统表写全 14 张、差异清单新增两行——每条都有原型证据支撑，且在 M0 §5.3 的既定范围内 |
| 2 | 基线（spec §3 第 8 项）：保持 `02c19e1`（`preview`）为代码基线；`v1.4.2` 标签是 `5f7d927`，迁移文件逐字节相同；v0 设计表头在阶段收尾时补充与差异清单一致的说明；P5 从 `02c19e1` 迁入前端 |
| 3 | 快照保留 `pg_dump` 的原样输出（包括末尾空行）：原样输出最容易复现和核对，仓库中没有检查这项的钩子或持续集成步骤 |
| 4 | 实现分工：Task 1 单独执行（sonnet）；Task 2+3 批处理（haiku），一次评审；评审统一用 sonnet；整分支评审用 opus |
| 5 | 修复轮范围 = Important 1 + Minor 2–7 + spec 中 Ctrl-C/§2.2/§2.3 的措辞；Minor 8 的文档缺口和全部 doc-sync/handoff 事项放到阶段收尾处理 |

## 6. 持续集成证据

- **`35689184964`**（提交 `5eab98d`，Task 1–3）：成功。`server`、`web` 两个任务的全部步骤通过。
- **`35690530470`**（提交 `0e223bb`，修复轮）：成功。两个任务的全部步骤通过。

## 7. 移交事项

**本次创建并安装的 handoff（3 个）：**

| handoff | 去向 | 内容 |
|---|---|---|
| [M0-P4-schema-conventions](../../M2-auth/handoffs/M0-P4-schema-conventions.md) | M2 | 第一次按快照建表的约定（外键、命名、默认值/CHECK、系统表、Django 索引产物）、94 张表 UUID 主键的例外 |
| [M0-P4-pg-trgm](../../M4-issue-core/handoffs/M0-P4-pg-trgm.md) | M4 | `issues.name` 的 pg_trgm 索引，以及转给 M8 的部署要求 |
| [P4-plane-schema-p5-notes](../handoffs/P4-plane-schema-p5-notes.md) | M0/P5 | 前端来源提交是 `02c19e1`；`plane/` 在移动的 `preview` 分支上，迁入前要锁定提交并记录 SHA |

**本次关闭的 handoff：** 无。spec 的"前置交接"为空（`handoffs/` 中没有交给 P4 的事项）。
