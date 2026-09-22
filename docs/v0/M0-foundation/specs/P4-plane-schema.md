# M0/P4 Plane 表结构快照：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M0/P4 `plane-schema` |
| 日期 | 2026-09-22 |
| 状态 | 已批准 |
| 上级文档 | [M0 设计文档](../M0-design.md) 第 2、5.3、6.1、6.3、8、12 节；[v0 总体设计](../../v0-design.md) 5.1、5.2、5.6 节；[差异清单](../../plane-diff.md) |
| 前置交接 | 无（`handoffs/` 中没有交给 P4 的事项） |

## 1. 目标
提供各个 M 建表时"照搬 Plane"的依据：Plane v1.4.2 在空库上跑完自带的全部 Django 迁移之后的完整表结构。
- 提取脚本 `tools/plane-schema/extract.sh`：只需要 Docker，从零生成快照；重新运行得到逐字节相同的结果；
- 快照 `tools/plane-schema/plane-v1.4.2-schema.sql`，提交到仓库；
- README 写明来源（Plane 版本、提交、镜像摘要）、生成时间、内容、重新生成和升级的方法；
- `make plane-schema`；
- 用快照核对总体设计 5.2 的 44 张表和差异清单，更新差异清单中的相关说明。

## 2. 交付物

### 2.1 文件总览
路径都相对于仓库根目录。"生成"表示由命令生成并提交，不手改。

| 路径 | 内容 |
|---|---|
| `tools/plane-schema/compose.yaml` | 临时环境：Postgres 15.7 和 Plane v1.4.2 后端镜像（2.4） |
| `tools/plane-schema/extract.sh` | 提取脚本，可执行（2.5） |
| `tools/plane-schema/plane-v1.4.2-schema.sql` | 快照（生成，2.6） |
| `tools/plane-schema/README.md` | 来源、内容、重新生成、升级方法（2.9） |
| `Makefile` | 新增 `plane-schema`（2.8） |
| `README.md` | 新增"Plane 表结构快照"一节（2.9） |
| `docs/v0/plane-diff.md` | 按快照更新（2.9） |

P4 不改动 Go 代码、前端代码和持续集成。

### 2.2 原型验证：结论与证据

**结论：官方镜像可用，不需要"从源码用 Python 执行迁移"的备选方案。** 在仓库外的临时目录里，按 2.4、2.5 的最终写法完整运行了提取流程。本 spec 和计划中的数值都来自这些运行。

| 问题 | 结论 | 证据 |
|---|---|---|
| `makeplane/plane-backend:v1.4.2` 能否拉取和运行 | 能。多架构索引中有 amd64 和 arm64 | arm64（本机原生）约 40 秒跑完迁移；amd64（在 arm64 上模拟）约 65 秒 |
| 迁移需要哪些服务和环境变量 | 只要 Postgres。环境变量只有 `DATABASE_URL`，外加一个非空的 `REDIS_URL` | 不设 `REDIS_URL` 时，`plane/__init__.py` 导入 `plane/celery.py`，后者在导入时执行 `redis.Redis.from_url(None)`，报 `AttributeError: 'NoneType' object has no attribute 'startswith'`。设为不存在的主机 `redis://unused.invalid:6379/0` 后迁移成功：Redis 客户端只在第一次执行命令时才连接，整个迁移过程从未连接。迁移文件中没有调用 Celery 任务或缓存。`SECRET_KEY` 为空时 Plane 用随机值，与表结构无关；RabbitMQ、MinIO 都不需要 |
| 执行了哪些迁移 | 164 个 | auth 12、contenttypes 2、db 122、django_celery_beat 21、license 6、sessions 1。`django_celery_results` 装了但不在 `INSTALLED_APPS` 中，没有它的表 |
| 输出是否确定 | 确定 | 从零完整运行 6 次（其中 1 次先删除本机的后端镜像、重新拉取；1 次用 amd64 镜像），输出逐字节相同：11707 行，379933 字节，SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70` |
| `\restrict` 随机行 | 不会出现 | pg_dump 15.7 的 `--help` 中没有 `--restrict-key`，输出中也没有 `\restrict`（它由 15.14 引入）。头部随版本变化的只有两行（`Dumped from database version 15.7`、`Dumped by pg_dump version 15.7`），版本随镜像写死 |
| 44 张表是否都在 | 都在，名字全部正确 | 2.7 |
| 镜像由哪个提交构建 | `v1.4.2` 标签 `5f7d927` | 2.3 |
| 脚本能否在 macOS 上运行 | 能 | `#!/usr/bin/env bash` 在本机解析到 `/bin/bash` 3.2.57；除了 docker，只用到 `dirname`、`mv`、`rm`、`wc`、`tr` |
| 出错和中断时是否清理 | 是 | Docker 不可用、拉取失败（不存在的标签）、迁移失败（去掉 `REDIS_URL`）三种情况以退出码 1 结束，并给出一行说明；Ctrl-C、SIGTERM 分别以 130、143 结束。五种情况都不留下容器、网络、数据卷和临时文件，也不写出快照 |
| `-p` 是否必要 | 是 | 设置 `COMPOSE_PROJECT_NAME=plane-app` 时，不带 `-p` 的 `docker compose -f compose.yaml config` 得到 `name: plane-app`（环境变量覆盖了文件中的 `name:`），这时清理用的 `down --volumes --remove-orphans` 会删掉另一个项目的容器；带上 `-p nerve-plane-schema` 后不受影响 |

### 2.3 Plane 的提交：标签与参考源码
- **`v1.4.2` 标签**指向 `5f7d92784c403f76284f0f16718f320221dc7fec`（`release: v1.4.2 #9632`，2026-08-23，`master` 分支上的合并提交）。后端镜像 `v1.4.2` 由它构建：
  - 镜像的 Python 是 3.12.10，与标签中 `Dockerfile.api` 的 `python:3.12.10-alpine` 一致（`preview` 已改为 3.12.12）；
  - 镜像的 `plane/settings/common.py` 中还有 PostHog 的配置，`requirements/base.txt` 中是 `djangorestframework==3.17.1`，都与标签一致（`preview` 已删除 PostHog，DRF 是 3.17.2）；
  - 镜像创建于 2026-08-23T14:36:03Z，比标签提交晚 3 分钟。
- **参考源码**：总体设计和差异清单写的"Plane v1.4.2，提交 `02c19e1`"，是仓库外参考源码 `plane/` 的提交，即 `preview` 分支的 `02c19e1341d93141e8ad7b3278298adce208bafc`（2026-09-21）。它的 `package.json` 版本也是 1.4.2，但比 `v1.4.2` 标签多 63 个提交（两者的共同祖先是 `v1.4.2-rc1`），其中包括前端升级到 React 19 和 React Router 8、pnpm 升级到 11.10。
  - M0 设计第 1 节锁定的前端版本来自 `02c19e1`，而不是标签：`02c19e1` 中是 React Router 8.3.0、pnpm 11.10.0，`v1.4.2` 标签中是 React Router 7.17/7.18、pnpm 11.3.0。所以项目实际的代码基线就是 `02c19e1`，只是"v1.4.2"这个说法不够准确。
- **两者的表结构相同**：
  - 镜像中的迁移目录（128 个迁移文件和 2 个 `__init__.py`，共 130 个文件）与 `plane/` 中的逐字节相同：按路径排序后拼接，SHA-256 都是 `85e5c6124354fb193411934680b475e976992ae4a191155ad38bc01ea621105a`；
  - 在 GitHub 上对比两个提交的 `apps/api` 目录：迁移文件和模型文件没有任何差异；有差异的依赖只是 DRF 的补丁版本和删除 `posthog`，都不带迁移。
- 所以快照同时是 `5f7d927` 和 `02c19e1` 的表结构。README 写明两个提交；差异清单的基线一行补充这层关系（2.9）。总体设计的表头是否同样补充，请控制者裁定（第 3 节第 8 项）。

### 2.4 临时环境：`tools/plane-schema/compose.yaml`
- 项目名 `nerve-plane-schema`：文件中写 `name:`，便于手工查看；脚本另外用 `-p` 传入（2.5）。
- **`db`**：
  - `postgres:15.7-alpine@sha256:468d34fefd6338031787c7b8e94078975b3aaf4d66c7ead25c39cd3ba46a15c6`，与 Plane v1.4.2 自带的 `docker-compose.yml` 和 `deployments/cli/community/docker-compose.yml` 相同；
  - 账号 `plane`、密码 `plane`、库 `plane`；
  - 数据目录 `/var/lib/postgresql/data` 放在 tmpfs 中。镜像为这个目录声明了 `VOLUME`，挂上 tmpfs 之后不再生成匿名卷（已核实容器没有任何卷挂载）；
  - 健康检查 `pg_isready -h 127.0.0.1 -U plane -d plane`：走 TCP。初始化阶段的临时服务只监听 Unix 套接字，不会被误判为就绪。
- **`migrator`**：
  - `makeplane/plane-backend:v1.4.2@sha256:90032ce088708889b60c00d491897916f4deb882facda27db59fd10fb68729ef`；
  - 命令 `python manage.py migrate --no-input`。`manage.py` 默认使用 `plane.settings.production`，与 Plane 自己的部署相同；
  - 环境变量只有 `DATABASE_URL` 和 `REDIS_URL`（2.2）；
  - 不用 Plane 的 `bin/docker-entrypoint-migrator.sh`：它先执行 `wait_for_db` 再迁移，这里由健康检查保证数据库就绪。
- 不映射任何主机端口，不声明数据卷；两个服务只在 compose 的默认网络中互相访问。
- **镜像写成"标签@摘要"**：标签便于阅读；摘要是多架构索引的摘要，同一个摘要在 amd64、arm64 上分别取到对应的镜像。写死摘要的理由：
  - "重新运行得到一致的快照"要求输入不变。标签可以被重新推送：Plane `preview` 的 `Dockerfile.api` 已经加入 `APK_SECURITY_PATCH` 参数，专门用来重建镜像；
  - 本机已经有这个摘要的镜像时，不再访问 Docker Hub（2.5 的 `--policy missing`）；
  - 代价：升级时标签和摘要要一起改。README 写明取摘要的命令。

### 2.5 提取脚本：`tools/plane-schema/extract.sh`
**步骤**：
1. 检查 `docker` 命令、Docker 服务、`docker compose`；任何一项不可用都退出，并说明原因（例如 `extract.sh: cannot reach the Docker daemon; start Docker and retry`）。
2. 设置退出时的清理：删除临时文件，执行 `down --volumes --remove-orphans`。`INT`、`TERM` 转为 `exit 130`、`exit 143`，保证清理一定执行。
3. 先清理一次，防止上次被中断后留下的容器被重用。
4. `pull --quiet --policy missing`：只拉取本机没有的镜像；失败时说明 `pulling the images failed`。
5. `up --detach --wait db`：启动 Postgres，等到健康检查通过。
6. `run --rm --no-deps -T migrator`：执行迁移，输出原样打印（164 行 `Applying …`）。`-T` 不分配终端：否则在终端中运行时，compose 把终端切到原始模式，Ctrl-C 被转发进容器，脚本会以"迁移失败"（退出码 1）结束，而不是 130。
7. `exec -T db pg_dump --schema-only --no-owner --no-privileges --username plane --dbname plane`，写入临时文件 `plane-v1.4.2-schema.sql.tmp`，成功后再改名为快照。任何一步失败都不会留下写了一半的快照。
8. 打印快照的路径和行数。

**写法**：
- `#!/usr/bin/env bash`，`set -euo pipefail`；兼容 bash 3.2（macOS 自带）；不需要本机的 `psql`、Python。
- 每条 compose 命令都带 `--project-name nerve-plane-schema`：命令行参数优先于环境变量 `COMPOSE_PROJECT_NAME`，也优先于文件中的 `name:`，所以清理只会作用于本项目（证据见 2.2）。
- 输出路径由脚本所在的目录决定，在任何目录下运行结果都一样。
- 需要 Compose 2.22 或更高：`pull --policy` 从 v2.22.0 开始提供（已对照 docker/compose 各版本的源码）。本机是 Docker 29.7.2、Compose 5.4.0。
- 缩进 2 个空格（`.editorconfig`）；注释和提示用英文，与 Go 代码一致。
- **中断的两种情况**（都已验证）：
  - Ctrl-C：整个前台进程组都收到 SIGINT，正在运行的 compose 命令随之结束，脚本立即以 130 退出并清理（原型中在第 57 个迁移时中断；评审修复加上 `-T` 后，又用 `script` 分配的伪终端在迁移中途按 Ctrl-C 核实）。
  - 只给脚本进程发 SIGTERM：bash 要等当前的前台命令结束才执行 trap，所以会等迁移跑完（约 40 秒）再以 143 退出并清理，不写出快照。

### 2.6 快照：`tools/plane-schema/plane-v1.4.2-schema.sql`
- pg_dump 的原样输出，不做任何后处理：11707 行，379933 字节，SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70`。
- **不过滤**：未保留的表和 Django、Celery Beat 的系统表都留在快照里。
  - 快照回答的是"Plane 有什么"。保留的表上有哪些列和外键指向被砍掉的功能，要对照完整的快照才看得出来：本次核对就是这样发现 `issue_comments.description_id` 的（2.7）。
  - 过滤就要把"保留哪些表"写进工具，而这份清单属于差异清单，会随设计调整；工具只负责如实导出。
  - pg_dump 的原样输出最容易复现和核对。
- **内容**：
  - 110 张表：Plane 的 96 张业务表（`db` 应用 92 张，`license` 应用 4 张），加上 14 张系统表：`django_migrations`、`django_content_type`、`django_session`、`auth_group`、`auth_group_permissions`、`auth_permission`、`users_groups`、`users_user_permissions`，以及 6 张 `django_celery_beat_*`。
  - 110 个主键、69 个唯一约束、557 个索引（其中 40 个是部分唯一索引，38 个的条件是 `deleted_at IS NULL`，另外 2 个在 `labels` 上）、474 个外键、18 个 CHECK、13 个 identity 列（系统表和 `project_identifiers` 的整数主键）。
  - 没有扩展、函数、触发器、视图、自定义类型，也没有列默认值。
- **格式**：只有 ASCII 字符，没有行尾空格，没有 CRLF，符合 `.editorconfig`。文件末尾有一个空行（pg_dump 的输出如此），`git diff --check` 会报 `new blank line at EOF`。不处理：快照保持 pg_dump 的原样；仓库中没有执行这项检查的钩子或持续集成步骤。
- **生成时间**：快照中没有时间戳，重新生成不会改变它。README 中的"生成时间"是快照内容最后一次变化的日期（2026-09-22）。

### 2.7 用快照核对设计文档
- **总体设计 5.2 的 44 张表**：全部存在，名字全部是 Plane 模型真实的 `db_table`，没有需要更正的表名。核对命令见计划 Task 2。
- **差异清单一的 52 张未保留的表**：全部存在。44 + 52 = 96 张业务表，加上 14 张系统表，正好是快照中的 110 张表，没有遗漏，也没有多出。
- **差异清单二点名的列**：每一列都在快照中，例如 `users.is_bot`、`api_tokens.token`、`projects.close_in`、`issues.description_binary`、`issue_views.query`，以及类型为 `date` 的 `issues.archived_at`。
- **差异清单中关于约束的说法**，与快照一致：
  - `labels` 的名称唯一有两个部分唯一索引：`project_id` 为空时是 `(name)`（整个实例范围，没有限定在工作区内），不为空时是 `(project_id, name)`；
  - `issue_labels` 没有唯一约束；`issues` 没有 `(project_id, sequence_id)` 唯一约束；
  - `cycle_issues` 的唯一是 `(cycle_id, issue_id)`，不是 `(issue_id)`；
  - `unique_together (..., deleted_at)` 的写法确实存在，例如 `cycle_issues_issue_id_cycle_id_deleted_at_93e8fecd_uniq`。
- **发现的遗漏**（在 2.9 中补上）：
  1. 差异清单一列出的系统表缺了 `django_session`、`auth_group_permissions`。
  2. 保留的表中，指向未保留的表的外键共 7 个；其中 `issue_comments.description_id → descriptions`（带唯一约束，是一对一关系）没有写进差异清单二。另外 6 个都已登记：`issues` 和 `draft_issues` 的 `estimate_point_id`、`type_id`，`projects.estimate_id`，`file_assets.page_id`。
  3. 总体设计 6.9 已定的"工作项标题建 pg_trgm 索引"没有写进差异清单。Plane 没有这个索引：快照中没有任何扩展，`issues` 上只有外键列的 btree 索引。

### 2.8 `make plane-schema` 与持续集成
- **`make plane-schema`**：执行 `tools/plane-schema/extract.sh`，放在 Makefile 的最后。`make help` 显示为"重新生成 Plane 表结构快照（需要 Docker，见 tools/plane-schema/README.md）"。
  - 理由：Makefile 是所有命令的统一入口（M0 设计 6.1），`make help` 中能看到；它只是转发，不重复脚本的逻辑。
  - 不另加检查命令：是否与提交的版本一致，用 `git status --short -- tools/plane-schema` 就能看出（README 写明）。
- **持续集成不运行提取**：
  - 每次都要从 Docker Hub 拉取约 205 MB 的镜像（压缩后：后端约 110 MB，Postgres 约 96 MB），再跑约 1 分钟，还受 Docker Hub 匿名拉取频率的限制；
  - 两个输入都按摘要写死，仓库中没有别的东西影响输出，快照不会自己变化。只有升级 Plane 时才重新生成，那是一次在本地完成、经过评审的改动。
- **不加检查快照的测试**（例如在 Go 测试中确认 44 张表都在）：
  - 没有代码读取快照，文件本身也不会变；
  - 它唯一的变化途径是升级 Plane，那时保留的表清单本身也要重新评审，这样的测试只会跟着改；
  - 44 张表在本 Phase 的验收中核对一次（第 4 节）。

### 2.9 文档
- **`tools/plane-schema/README.md`**：
  - 快照信息：Plane 版本和标签提交、两个镜像的"标签@摘要"、与参考源码 `02c19e1` 的关系、生成时间、行数和 SHA-256；
  - 内容概要（2.6）；
  - 重新生成：`make plane-schema`，再用 `git status` 确认没有变化；只需要 Docker；持续集成不运行；
  - 实现要点：摘要、只需要两个环境变量、`-p`、tmpfs、不映射端口、清理、`--restrict-key`；
  - 升级 Plane 版本的步骤；官方镜像拿不到时的办法（用 Plane 源码的 `Dockerfile.api` 自己构建，未验证）。
- **根目录 `README.md`**：在"接口与代码生成"和"版权"之间新增"Plane 表结构快照"一节，两句话：快照是什么、怎样重新生成。
- **`docs/v0/plane-diff.md`**（M0 设计第 8 节 P4 的验收"差异清单中的相关说明已更新"）：
  1. 基线一行：补充 `02c19e1` 是 `preview` 分支、`v1.4.2` 标签是 `5f7d927`、两者的迁移文件逐字节相同，链接到 README（基线本身不变）。
  2. 开头"列级别的细节……起点是……快照"：改为链接到快照文件。
  3. 一：系统表写全 14 张，补上 `django_session`、`auth_group_permissions`；注明 96 + 14 = 110，与快照一致。
  4. 二·全局中"默认值、非空约束、枚举检查写进数据库"一行：补上证据，快照中没有任何列默认值，CHECK 约束只有 18 个由 Django 正整数字段生成的 `>= 0`。
  5. 二·按表：新增 `issue_comments` 删除 `description_id` 及其唯一约束（指向不保留的 `descriptions` 的一对一外键）。
  6. 二·按表：新增 `issues` 的 `name` 的 pg_trgm 索引（需要 `pg_trgm` 扩展）。

## 3. 与上级设计的差异和补充（请控制者裁定）

| # | 上级设计 | P4 的做法 | 理由 |
|---|---|---|---|
| 1 | M0 5.3：Postgres 15（Plane 使用的版本） | `postgres:15.7-alpine`，按"标签@摘要"写死 | Plane v1.4.2 自带的两个 compose 文件都用这个版本；摘要保证每次拿到同一个镜像（2.4） |
| 2 | M0 5.3：Plane v1.4.2 的后端镜像 | `makeplane/plane-backend:v1.4.2`，同样按"标签@摘要"写死 | 标签可以被重新推送；"重新运行得到一致的快照"要求输入不变（2.4） |
| 3 | M0 5.3：在后端镜像里执行 Django 迁移 | 直接执行 `python manage.py migrate --no-input`，不用 Plane 的 `docker-entrypoint-migrator.sh`；只启动 Postgres，`REDIS_URL` 指向不存在的主机 | 数据库就绪由健康检查保证，`wait_for_db` 多余；迁移只用到数据库，Redis 客户端从未连接（2.2） |
| 4 | M0 5.3、第 12 节：镜像拿不到时，改为用 Python 在 Plane 源码目录里执行迁移，P4 核实 | 不实现备选方案；README 记下"用 Plane 源码的 `Dockerfile.api` 自己构建镜像"作为恢复办法（未验证） | 镜像在 amd64、arm64 上都能用；用 Python 执行需要在本机安装 Python 和 Plane 的依赖，违反"除了 Docker、Go、Node 不装别的"；不写用不上的代码 |
| 5 | M0 5.3：`pg_dump --schema-only --no-owner --no-privileges` | 参数不变；pg_dump 在 Postgres 容器中执行 | 版本与服务端一致，本机不需要 `psql`；15.7 早于 `\restrict`，不需要 `--restrict-key`（2.2） |
| 6 | M0 5.3：快照文件 | 不做过滤：保留系统表和未保留的表 | 快照是"Plane 有什么"的依据；保留清单属于差异清单；原样输出最容易核对（2.6） |
| 7 | M0 5.3：README 写明"生成时间" | 生成时间是快照内容最后一次变化的日期；另外写明两个镜像的摘要、行数、SHA-256，以及标签提交与参考源码提交的关系 | pg_dump 的输出中没有时间戳，重新生成不会改变快照（2.6） |
| 8 | 总体设计表头、差异清单：Plane v1.4.2，提交 `02c19e1` | 快照注明来自 `v1.4.2` 标签（`5f7d927`）构建的镜像；差异清单的基线一行补充两者的关系，基线本身不变 | `02c19e1` 是 `preview` 分支，比标签多 63 个提交；迁移文件逐字节相同，快照对两者都成立；M0 锁定的前端版本也来自 `02c19e1`（2.3）。**请裁定**：总体设计的表头是否同样补充；建议保持 `02c19e1` 为代码基线，P5 从它迁入前端（第 7 节） |
| 9 | M0 6.1 的命令表 | 新增 `make plane-schema` | Makefile 是统一入口；只转发给脚本（2.8） |
| 10 | M0 6.3 持续集成 | 不运行提取，也不加检查快照的测试 | 拉取约 205 MB 镜像、约 1 分钟；输入写死，快照不会自己变化；没有代码读取它（2.8） |
| 11 | 差异清单一：系统表列了 7 项 | 写全 14 张：补上 `django_session`、`auth_group_permissions` | 按快照核对（2.7） |
| 12 | 差异清单二 | 新增两行：`issue_comments` 删除 `description_id` 及其唯一约束；`issues` 新增 `name` 的 pg_trgm 索引 | 前者指向不保留的 `descriptions`；后者是总体设计 6.9 已定的差异，Plane 没有（2.7） |

控制者裁定后，评审阶段把 M0 设计第 5.3 节（镜像、迁移命令、备选方案的结论，链接本 spec）、第 6.1 节（`make plane-schema`）、第 12 节（"Plane 后端镜像无法获取"一行改为"已由 M0/P4 验证并解除"），以及总体设计表头的基线说明（按第 8 项的裁定）同步更新。总体设计 5.2 的 44 个表名全部正确，不需要改动。

## 4. 验收标准
1. **重新运行得到一致的快照**（M0 设计第 8 节 P4 验收 1）：`make plane-schema` 从零运行成功，写出 11707 行、SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70` 的快照；提交后再运行一次，`git status --short -- tools/plane-schema` 没有输出。
2. **44 张表都在**（验收 2）：计划 Task 2 的核对命令输出 `checked=44 missing=0`；快照中共 110 张表。
3. **差异清单已更新**（验收 3）：2.9 列出的 6 处都已修改。
4. **不留下任何东西**：每次运行结束后（包括失败），`docker ps -a --filter label=com.docker.compose.project=nerve-plane-schema` 和 `docker network ls --filter name=nerve-plane-schema` 都没有输出，`tools/plane-schema/` 下没有 `.tmp` 文件；本机其他 compose 项目的容器不受影响。
5. **Docker 不可用时给出说明**：`DOCKER_HOST=unix:///nonexistent.sock make plane-schema` 输出 `extract.sh: cannot reach the Docker daemon; start Docker and retry`，退出码非 0。
6. **命令与文件**：`make` 列出 `plane-schema`；Makefile 在 GNU Make 3.81 下可用；`extract.sh` 在 git 中的模式是 `100755`；`/bin/bash -n` 检查通过。
7. **持续集成通过**：P4 不改动持续集成，推送后 `server`、`web` 两个任务照常通过。

## 5. 不在 P4 范围内
- 按快照建表：由各个 M 为自己的模块编写迁移（总体设计 5.6），从 M2 开始。
- 逐列核对差异清单：由建这张表的 M 完成。
- 从源码构建后端镜像的备选方案（第 3 节第 4 项）。
- 在持续集成中运行提取或检查快照（2.8）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| Docker Hub 上的镜像被删除，或摘要不再可用 | 快照已经提交，日常开发不依赖重新生成；本机已有的镜像不受影响（`--policy missing`）；README 写明恢复办法：用 Plane 源码的 `Dockerfile.api` 在对应标签上构建 |
| 以后把 Postgres 镜像升到 15.14 或更高，pg_dump 会输出随机的 `\restrict` 行，每次结果都不同 | README 写明：升级时加上 `--restrict-key` 并给一个固定值 |
| 有人用编辑器保存了快照（例如去掉了末尾的空行） | README 写明快照是生成物、不要手改；`make plane-schema` 能恢复原样；`git status` 能看出改动 |
| 只给脚本进程发 SIGTERM 时，要等当前步骤结束才清理 | bash 的行为：前台命令结束后才执行 trap。最多等一次迁移（约 40 秒），清理照常完成，不写出快照（已验证）；Ctrl-C 立即生效 |
| 两个工作区同时运行 `extract.sh` | 两者共用项目名，会互相清理对方的容器。这个命令只在升级 Plane 时手工运行，一次只运行一份 |
| Compose 低于 2.22 时没有 `pull --policy` | 拉取这一步失败，compose 自己会报 `unknown flag`，脚本接着说明拉取失败；README 写明版本要求 |

## 7. 移交给后续阶段的事项（评审时建立 handoff）

| 交给 | 事项 |
|---|---|
| M2 | 第一次按快照建表时，一次定下以下约定，以后各个 M 照做：（1）Plane 的 474 个外键全部是 `DEFERRABLE INITIALLY DEFERRED`，都没有 `ON DELETE`（Django 在 Python 中处理级联）：Nerve 的外键是否保留 `DEFERRABLE`，每类关系用什么 `ON DELETE`，要与"连带删除在同一事务中同步完成"（总体设计 5.5）一起考虑；（2）约束和索引名带 Django 的哈希后缀（例如 `label_workspace_id_c4c9ae5a`、`accounts_user_id_7f1e1f1e_fk_users_id`、`…_uniq`）：沿用，还是改为可读的命名规则；（3）快照中没有列默认值，也没有枚举 CHECK：默认值和取值范围要从 Plane 源码的 `apps/api/plane/db/models/` 中读取，再写进数据库（差异清单二·全局）；（4）快照中的 14 张系统表不照搬 |
| M2 | 94 张业务表的主键是没有默认值的 `id uuid`，与"由应用生成 UUIDv7"（总体设计 5.5）一致；`sessions`（主键 `session_key`）、`project_identifiers`（整数 identity 主键）是例外，两张都不保留 |
| M4 | `issues.name` 的 pg_trgm 索引要先 `CREATE EXTENSION pg_trgm`（Plane 没有任何扩展），由 M4 的迁移创建，并确认 Postgres 18 镜像中有这个扩展 |
| P5 | 参考源码 `plane/` 是 `preview` 分支的 `02c19e1`，不是 `v1.4.2` 标签（`5f7d927`）。M0 设计第 1 节锁定的 React Router 8.3、pnpm 11.10 都来自 `02c19e1`（标签中是 React Router 7.18、pnpm 11.3.0），所以前端必须从 `02c19e1` 迁入。按第 3 节第 8 项的裁定，在前端改动清单中写明迁入的来源提交 |
