# M0/P4 Plane 表结构快照 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提供各个 M 建表时"照搬 Plane"的依据：`tools/plane-schema/` 下的提取脚本、临时环境、说明文档，以及提交到仓库的 Plane v1.4.2 表结构快照 `plane-v1.4.2-schema.sql`；新增 `make plane-schema`；用快照核对并更新差异清单。

**Architecture:** `extract.sh` 用 docker compose（项目名固定为 `nerve-plane-schema`，每条命令都用 `-p` 传入）启动一个数据目录在 tmpfs 上的 Postgres 15.7，在按摘要写死的 Plane v1.4.2 后端镜像中执行 `python manage.py migrate`，再用 Postgres 容器里的 `pg_dump --schema-only --no-owner --no-privileges` 导出，先写临时文件、成功后改名为快照；退出时（包括出错和中断）删除本项目的容器和网络。输入都已写死，重新运行得到逐字节相同的快照。快照是生成物，提交到仓库；持续集成不运行提取。

**Tech Stack:** Docker（Compose 2.22 或更高；本机 Docker 29.7.2、Compose 5.4.0）、bash 3.2、`postgres:15.7-alpine`、`makeplane/plane-backend:v1.4.2`、GNU Make 3.81。

**Spec:** `docs/v0/M0-foundation/specs/P4-plane-schema.md`（上级：`docs/v0/M0-foundation/M0-design.md`）

## Global Constraints

- **Docker 安全**：只通过 `make plane-schema`（即 `tools/plane-schema/extract.sh`）使用 Docker，另外只执行本计划中写明的只读查看命令（带 `nerve-plane-schema` 过滤条件的 `docker ps -a`、`docker network ls`）。本机运行着其他项目的容器（`plane-app-*`、`nerve-dev-db-1`、`agentforge-*`、`opennerve-*` 等），它们与本 Phase 无关：不要停止、重启、删除、进入或改动任何其他容器、数据卷、网络、镜像；不要执行 `docker rm`、`docker rmi`、`docker system prune`、`docker volume prune` 之类的命令。
- 镜像只按 `tools/plane-schema/compose.yaml` 中的"标签@摘要"使用，不要去掉摘要，也不要换版本。
- **快照不手写、不手改、不从本计划复制**：执行 `make plane-schema`，再提交它的输出。预期 11707 行、SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70`；对不上时停下来，说明 `compose.yaml` 或 `extract.sh` 与本计划不一致。
- 本 Phase 不改动 Go 代码、前端代码和持续集成，不需要执行 `make lint`、`make test`。
- Shell 脚本：`#!/usr/bin/env bash`，兼容 bash 3.2（macOS 自带；不用关联数组、`mapfile`、`${var,,}` 等 bash 4 的写法），缩进 2 个空格（`.editorconfig`），注释和提示用英文。YAML、Makefile 的注释和 Markdown 的正文用中文。
- 所有代码块都是完整的文件内容（"修改"步骤除外，它给出"把……替换为……"的原文），照原样写入，不要改动。**Makefile 的命令行以 Tab 开头**：写入后用 `grep -c "$(printf '\t')" Makefile` 核对（Task 1 给出预期值）；Tab 变成空格时 make 会报 `missing separator`。
- Makefile 必须兼容 macOS 自带的 GNU Make 3.81。
- 第一次运行 `make plane-schema` 会拉取约 205 MB 的镜像；之后每次约 1 分钟。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行。

## 文件结构

路径相对于仓库根目录。

| 文件 | 职责 | Task |
|---|---|---|
| `tools/plane-schema/compose.yaml` | 临时环境：Postgres 15.7 和 Plane v1.4.2 后端镜像 | 1 |
| `tools/plane-schema/extract.sh` | 提取脚本（可执行） | 1 |
| `Makefile`（修改） | 新增 `make plane-schema` | 1 |
| `tools/plane-schema/plane-v1.4.2-schema.sql`（生成） | 快照 | 1 |
| `tools/plane-schema/README.md` | 来源、内容、重新生成和升级的方法 | 2 |
| `README.md`（修改） | 新增"Plane 表结构快照"一节 | 2 |
| `docs/v0/plane-diff.md`（修改） | 按快照更新差异清单 | 3 |

---

### Task 1: 提取工具、`make plane-schema` 与快照

**Files:**
- Create: `tools/plane-schema/compose.yaml`、`tools/plane-schema/extract.sh`
- Modify: `Makefile`
- Generate: `tools/plane-schema/plane-v1.4.2-schema.sql`（`make plane-schema`）

**Interfaces:**
- Consumes: Docker（Compose 2.22 或更高）；Docker Hub 上的 `postgres:15.7-alpine`、`makeplane/plane-backend:v1.4.2`（按摘要）
- Produces:
  - `make plane-schema`：重新生成快照
  - `tools/plane-schema/plane-v1.4.2-schema.sql`：11707 行，110 张表。Task 2 在它上面核对 44 张表，Task 3 引用它的内容

- [ ] **Step 1: 写入 `tools/plane-schema/compose.yaml`**

```yaml
# 生成 Plane 表结构快照用的临时环境，只由 extract.sh 使用（见 README.md）。
# 镜像按"标签@摘要"写死：标签便于阅读，摘要保证每次拿到的是同一个镜像。
# 不映射任何主机端口；数据目录放在 tmpfs 中，不产生数据卷。
name: nerve-plane-schema

services:
  db:
    # 与 Plane v1.4.2 自带的 docker-compose.yml 相同的 Postgres 版本
    image: postgres:15.7-alpine@sha256:468d34fefd6338031787c7b8e94078975b3aaf4d66c7ead25c39cd3ba46a15c6
    environment:
      POSTGRES_USER: plane
      POSTGRES_PASSWORD: plane
      POSTGRES_DB: plane
    tmpfs:
      - /var/lib/postgresql/data
    healthcheck:
      # 走 TCP：初始化阶段的临时服务只监听 Unix 套接字，这时不算就绪
      test: ["CMD-SHELL", "pg_isready -h 127.0.0.1 -U plane -d plane"]
      interval: 1s
      timeout: 5s
      retries: 60

  migrator:
    image: makeplane/plane-backend:v1.4.2@sha256:90032ce088708889b60c00d491897916f4deb882facda27db59fd10fb68729ef
    command: ["python", "manage.py", "migrate", "--no-input"]
    environment:
      DATABASE_URL: postgresql://plane:plane@db:5432/plane
      # plane/celery.py 在导入时用 REDIS_URL 创建 Redis 客户端，值不能为空；
      # 客户端只在第一次执行命令时才连接，迁移过程中从不连接，所以指向一个不存在的主机
      REDIS_URL: redis://unused.invalid:6379/0
```

- [ ] **Step 2: 写入 `tools/plane-schema/extract.sh`，设为可执行**

```bash
#!/usr/bin/env bash
# Regenerates plane-v1.4.2-schema.sql: runs the Django migrations of the
# Plane v1.4.2 backend image against an empty Postgres 15.7 and dumps the
# resulting schema. Needs only Docker (Compose 2.22 or later); see README.md.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
out="$here/plane-v1.4.2-schema.sql"
tmp="$out.tmp"
# Passed with -p on every call: it wins over COMPOSE_PROJECT_NAME, so the
# cleanup below can never touch another compose project.
project=nerve-plane-schema

die() {
  echo "extract.sh: $*" >&2
  exit 1
}

compose() {
  docker compose --project-name "$project" --file "$here/compose.yaml" "$@"
}

cleanup() {
  rm -f "$tmp"
  compose down --volumes --remove-orphans >/dev/null 2>&1 ||
    echo "extract.sh: cleanup failed; run: docker compose -p $project down --volumes" >&2
}

command -v docker >/dev/null 2>&1 || die "docker not found; install Docker with Compose 2.22 or later"
docker info >/dev/null 2>&1 || die "cannot reach the Docker daemon; start Docker and retry"
docker compose version >/dev/null 2>&1 || die "docker compose not found; install Compose 2.22 or later"

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Leftovers of an interrupted run would otherwise be reused.
compose down --volumes --remove-orphans >/dev/null 2>&1 || true

echo "==> pulling images (only those not present locally)"
compose pull --quiet --policy missing || die "pulling the images failed"

echo "==> starting Postgres"
compose up --detach --wait db || die "Postgres did not become healthy"

echo "==> running the Plane migrations"
compose run --rm --no-deps migrator || die "the Plane migrations failed"

echo "==> dumping the schema"
compose exec -T db pg_dump --schema-only --no-owner --no-privileges --username plane --dbname plane >"$tmp" ||
  die "pg_dump failed"
mv "$tmp" "$out"

echo "==> wrote $out ($(wc -l <"$out" | tr -d ' ') lines)"
```

Run: `chmod +x tools/plane-schema/extract.sh && /bin/bash -n tools/plane-schema/extract.sh && echo syntax-ok`
Expected: `syntax-ok`（macOS 上 `/bin/bash` 是 3.2）

Run: `shasum -a 256 tools/plane-schema/compose.yaml tools/plane-schema/extract.sh`
Expected:

```
56825f38525f1874dd5f059535836cab5d25793b64928c5705c04eb0790da6c7  tools/plane-schema/compose.yaml
bf73bf30daea5d70cb92115208800b35bf34da2f4afca175547d9c99b445f704  tools/plane-schema/extract.sh
```

对不上时，逐字对照 Step 1、Step 2 的代码块（常见原因：缩进、行尾空格、文件末尾缺少换行）。

- [ ] **Step 3: 修改 `Makefile`（在末尾加上 `plane-schema`）**

把：

```makefile
.PHONY: test
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...
```

替换为：

```makefile
.PHONY: test
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...

.PHONY: plane-schema
plane-schema: ## 重新生成 Plane 表结构快照（需要 Docker，见 tools/plane-schema/README.md）
	tools/plane-schema/extract.sh
```

（这三行是 Makefile 的最后三行；替换后新的 `plane-schema` 成为最后一个命令，文件以换行结束。）

Run: `grep -c "$(printf '\t')" Makefile`
Expected: `25`

Run: `make`
Expected（颜色代码略去）：

```
  help           列出所有命令
  dev-db         启动开发数据库（Postgres 18），等待就绪
  dev-db-down    停止开发数据库，保留数据
  dev-db-reset   停止开发数据库并删除数据卷
  run            以 dev 配置运行后端（需先 make dev-db），Ctrl-C 停止
  tools          安装锁定版本的 golangci-lint 到 ./bin
  gen            重新生成全部代码：Go 接口层、api/dist、TS 客户端
  gen-go         由 api/ 生成 Go 接口层（只需要 Go）
  gen-web        打包 api/dist/openapi.yaml，生成 TS 客户端的类型（需要 Node）
  gen-check      重新生成全部代码，检查生成物已提交且没有差异
  gen-check-go   重新生成 Go 接口层并检查（持续集成 server 任务）
  gen-check-web  重新生成 api/dist 和 TS 类型并检查（持续集成 web 任务）
  lint           运行全部静态检查
  lint-go        运行 golangci-lint（server）
  lint-web       前端类型检查（需要 Node）
  test           运行 Go 测试（server，不用测试缓存）
  plane-schema   重新生成 Plane 表结构快照（需要 Docker，见 tools/plane-schema/README.md）
```

- [ ] **Step 4: 演示：Docker 不可用时，脚本给出说明并失败**

Run: `DOCKER_HOST=unix:///nonexistent.sock make plane-schema; echo "exit=$?"`
Expected（GNU Make 3.81 的输出；make 4.x 的最后一行会带上 `Makefile:<行号>:`）：

```
tools/plane-schema/extract.sh
extract.sh: cannot reach the Docker daemon; start Docker and retry
make: *** [plane-schema] Error 1
exit=2
```

- [ ] **Step 5: 生成快照**

Run: `make plane-schema`
Expected（约 1 分钟；本机没有镜像时先拉取约 205 MB）：

```
tools/plane-schema/extract.sh
==> pulling images (only those not present locally)
 Image …（每个镜像一行：本机已有时是 Skipped Image is already present locally，否则是 Pulling、Pulled）
==> starting Postgres
 Network nerve-plane-schema_default …
 Container nerve-plane-schema-db-1 …
 Container nerve-plane-schema-db-1 Healthy
==> running the Plane migrations
 Container nerve-plane-schema-migrator-run-<随机> …
Operations to perform:
  Apply all migrations: auth, contenttypes, db, django_celery_beat, license, sessions
Running migrations:
  Applying contenttypes.0001_initial... OK
  …（共 164 行 Applying … OK，中间夹着几行 Plane 的 JSON 日志）
  Applying sessions.0001_initial... OK
==> dumping the schema
==> wrote <仓库路径>/tools/plane-schema/plane-v1.4.2-schema.sql (11707 lines)
```

Run: `wc -l tools/plane-schema/plane-v1.4.2-schema.sql && shasum -a 256 tools/plane-schema/plane-v1.4.2-schema.sql`
Expected:

```
   11707 tools/plane-schema/plane-v1.4.2-schema.sql
4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70  tools/plane-schema/plane-v1.4.2-schema.sql
```

Run: `sed -n 1,6p tools/plane-schema/plane-v1.4.2-schema.sql`
Expected:

```
--
-- PostgreSQL database dump
--

-- Dumped from database version 15.7
-- Dumped by pg_dump version 15.7
```

- [ ] **Step 6: 确认没有留下容器、网络和临时文件**

Run: `docker ps -a --filter label=com.docker.compose.project=nerve-plane-schema --format '{{.Names}}'; docker network ls --filter name=nerve-plane-schema --format '{{.Name}}'; ls -A tools/plane-schema`
Expected（前两条命令没有输出）：

```
compose.yaml
extract.sh
plane-v1.4.2-schema.sql
```

- [ ] **Step 7: 核对改动范围**

Run: `git status --short -uall`
Expected:

```
 M Makefile
?? tools/plane-schema/compose.yaml
?? tools/plane-schema/extract.sh
?? tools/plane-schema/plane-v1.4.2-schema.sql
```

- [ ] **Step 8: 提交**

```bash
git add Makefile tools/plane-schema/compose.yaml tools/plane-schema/extract.sh tools/plane-schema/plane-v1.4.2-schema.sql
git ls-files -s tools/plane-schema | cut -c1-6
```

Expected（依次是 `compose.yaml`、`extract.sh`、快照；`extract.sh` 必须是 `100755`，否则回到 Step 2 执行 `chmod +x` 后重新 `git add`）：

```
100644
100755
100644
```

```bash
git commit -m "feat(tools): add the Plane v1.4.2 schema snapshot and its extraction tool

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: 说明文档；验收：重新生成一致、44 张表都在

**Files:**
- Create: `tools/plane-schema/README.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: Task 1 的 `make plane-schema` 和提交的快照
- Produces: 快照的来源和使用说明；M0 设计第 8 节 P4 的验收 1（重新运行得到一致的快照）和验收 2（44 张表都在）的证据

- [ ] **Step 1: 写入 `tools/plane-schema/README.md`**

````markdown
# Plane 表结构快照

`plane-v1.4.2-schema.sql` 是 Plane v1.4.2 在空库上跑完自带的全部 Django 迁移后，用 `pg_dump` 导出的表结构。Nerve 的每个 M 为自己的模块建表时，以它为起点，再按[差异清单](../../docs/v0/plane-diff.md)修改（[总体设计](../../docs/v0/v0-design.md) 5.6）。

这个文件是生成物，不要手改。文件末尾的空行也是 pg_dump 的原样输出（`git diff --check` 会提示 `new blank line at EOF`），不要删除。

## 快照信息

| 项 | 内容 |
|---|---|
| Plane 版本 | v1.4.2，标签提交 `5f7d92784c403f76284f0f16718f320221dc7fec` |
| 后端镜像 | `makeplane/plane-backend:v1.4.2@sha256:90032ce088708889b60c00d491897916f4deb882facda27db59fd10fb68729ef`，由标签提交构建 |
| Postgres | `postgres:15.7-alpine@sha256:468d34fefd6338031787c7b8e94078975b3aaf4d66c7ead25c39cd3ba46a15c6`（与 Plane 自带的 `docker-compose.yml` 相同），`pg_dump` 15.7 |
| 与参考源码的关系 | 仓库外的参考源码 `plane/` 是 `preview` 分支的 `02c19e1341d93141e8ad7b3278298adce208bafc`，比 v1.4.2 标签多 63 个提交。两者的 130 个迁移文件逐字节相同，所以快照同样是 `02c19e1` 的表结构 |
| 生成时间 | 2026-09-22。这是快照内容最后一次变化的日期；快照中不含时间戳，重新生成不会改变它 |
| 大小 | 11707 行，SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70` |

## 内容
- 110 张表：Plane 的 96 张业务表（`db` 应用 92 张，`license` 应用 4 张），加上 Django 和 Celery Beat 的 14 张系统表。
- 110 个主键、69 个唯一约束、557 个索引（其中 40 个是 `WHERE deleted_at IS NULL` 这类部分唯一索引）、474 个外键、18 个 CHECK、13 个 identity 列。
- 没有扩展、函数、触发器，也没有列默认值：Plane 的默认值和枚举都在 Python 代码里（`apps/api/plane/db/models/`）。
- 474 个外键全部是 `DEFERRABLE INITIALLY DEFERRED`，都没有 `ON DELETE`：这是 Django 的做法，级联删除由 Python 代码完成。
- 不做任何过滤。未保留的表和系统表也留在快照里：它说明的是"Plane 有什么"，保留的表上有哪些列和外键指向被砍掉的功能，也要对照它才看得出来。

## 重新生成

```bash
make plane-schema                        # 即 tools/plane-schema/extract.sh
git status --short -- tools/plane-schema # 没有输出：与提交的版本逐字节相同
```

- 只需要 Docker（含 Compose 2.22 或更高：脚本用到的 `pull --policy` 从这个版本开始提供）。第一次运行要拉取约 205 MB 的镜像，之后每次约 1 分钟。
- `extract.sh` 的步骤：拉取本机没有的镜像 → 启动 Postgres 并等到它就绪 → 在后端镜像中执行 `python manage.py migrate` → 在 Postgres 容器中执行 `pg_dump --schema-only --no-owner --no-privileges` → 写入快照。
- 结果是确定的：输入的两个镜像都按摘要写死，同样的输入得到逐字节相同的快照。已在 arm64 和 amd64 上核实。
- 持续集成不运行它：要从 Docker Hub 拉取约 205 MB 的镜像，而输入都已写死，快照不会自己变化。

## 实现要点
- **镜像写成"标签@摘要"**：标签便于阅读，摘要保证拿到的永远是同一个镜像，即使上游重新推送了同名标签。摘要是多架构索引的摘要，amd64 和 arm64 都能用。
- **迁移只需要数据库**：后端镜像只设两个环境变量。`DATABASE_URL` 指向 compose 中的 Postgres；`REDIS_URL` 只是因为 `plane/celery.py` 在导入时就创建 Redis 客户端，值不能为空，而客户端只在第一次执行命令时才连接，所以指向一个不存在的主机 `unused.invalid`。不需要 Redis、RabbitMQ、MinIO。
- **不影响本机的其他项目**：每条 compose 命令都带 `-p nerve-plane-schema`（它优先于环境变量 `COMPOSE_PROJECT_NAME`，所以清理时不会误删别的 compose 项目）；不映射任何主机端口；Postgres 的数据目录放在 tmpfs 中，不产生数据卷；脚本退出时（包括出错和 Ctrl-C）删除本项目的容器和网络。开始时也先清理一次，防止上次被中断后留下的容器被重用。
- **pg_dump 用 Postgres 容器里的**：版本与服务端一致，本机不需要安装 `psql`。15.7 早于 15.14 引入的 `\restrict` 随机行，所以不需要 `--restrict-key`；以后把 Postgres 升到 15.14 或更高时，要加上 `--restrict-key` 并给一个固定值，否则每次输出都不同。

## 升级 Plane 版本时
1. 修改 `compose.yaml` 中两个镜像的标签和摘要（摘要取 `docker buildx imagetools inspect <镜像>:<标签>` 输出的 `Digest`）；Postgres 跟随新版本 Plane 自带的 `docker-compose.yml`。
2. 修改 `extract.sh` 中的输出文件名，删除旧快照，重新生成。
3. 更新本文件的快照信息，并按新快照核对差异清单。

官方镜像拿不到时，可以用 Plane 源码中的 `apps/api/Dockerfile.api` 在对应的标签上自己构建镜像，替换 `compose.yaml` 中的 `migrator` 镜像（未验证）。
````

Run: `shasum -a 256 tools/plane-schema/README.md`
Expected: `b55c0f5e8bebe59dfba47440c9b4435c5dfe731b371513129511093811e07e59  tools/plane-schema/README.md`

- [ ] **Step 2: 修改 `README.md`（新增"Plane 表结构快照"一节）**

把：

```markdown
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（TS 类型检查）。

## 版权
```

替换为：

```markdown
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（TS 类型检查）。

## Plane 表结构快照

`tools/plane-schema/plane-v1.4.2-schema.sql` 是 Plane v1.4.2 的完整表结构，各个 M 为自己的模块建表时以它为起点，再按[差异清单](docs/v0/plane-diff.md)修改。它是生成物，不要手改；只有升级 Plane 基线时才需要用 `make plane-schema` 重新生成（需要 Docker），说明见 [tools/plane-schema/README.md](tools/plane-schema/README.md)。

## 版权
```

- [ ] **Step 3: 验收：重新生成，与提交的快照逐字节相同**

Run: `make plane-schema`
Expected: 与 Task 1 Step 5 相同，最后一行是 `==> wrote <仓库路径>/tools/plane-schema/plane-v1.4.2-schema.sql (11707 lines)`。

Run: `git status --short -- tools/plane-schema/plane-v1.4.2-schema.sql && shasum -a 256 tools/plane-schema/plane-v1.4.2-schema.sql`
Expected（`git status` 没有输出：快照与 Task 1 提交的版本逐字节相同）：

```
4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70  tools/plane-schema/plane-v1.4.2-schema.sql
```

- [ ] **Step 4: 验收：总体设计 5.2 的 44 张表都在快照中**

下面的表名按总体设计 5.2 的顺序排列（不含新增的 `auth_sessions`）。

Run:

```bash
n=0; missing=0; for t in users profiles api_tokens workspaces workspace_members workspace_member_invites workspace_user_properties projects project_members project_user_properties states labels issues issue_assignees issue_labels issue_relations issue_links issue_subscribers issue_mentions issue_comments issue_reactions comment_reactions issue_activities issue_versions issue_description_versions draft_issues cycles cycle_issues cycle_user_properties modules module_issues module_members module_links module_user_properties issue_views intakes intake_issues notifications user_favorites user_recent_visits webhooks webhook_logs api_activity_logs file_assets; do n=$((n+1)); grep -q "^CREATE TABLE public\.$t (" tools/plane-schema/plane-v1.4.2-schema.sql || { echo "missing: $t"; missing=$((missing+1)); }; done; echo "checked=$n missing=$missing"
```

Expected: `checked=44 missing=0`

Run: `grep -c '^CREATE TABLE public\.' tools/plane-schema/plane-v1.4.2-schema.sql`
Expected: `110`（96 张 Plane 业务表 + 14 张系统表）

- [ ] **Step 5: 核对改动范围，确认没有遗留**

Run: `docker ps -a --filter label=com.docker.compose.project=nerve-plane-schema --format '{{.Names}}'; docker network ls --filter name=nerve-plane-schema --format '{{.Name}}'; git status --short -uall`
Expected（前两条命令没有输出）：

```
 M README.md
?? tools/plane-schema/README.md
```

- [ ] **Step 6: 提交**

```bash
git add README.md tools/plane-schema/README.md
git commit -m "docs(tools): document the Plane schema snapshot

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 按快照更新差异清单

**Files:**
- Modify: `docs/v0/plane-diff.md`
- 核对（不改）：`docs/v0/M0-foundation/M0-design.md` 第 11 节

**Interfaces:**
- Consumes: Task 1 的快照；spec 2.7 的核对结果
- Produces: M0 设计第 8 节 P4 的验收 3（差异清单中的相关说明已更新）

- [ ] **Step 1: 修改基线一行（第 3 行）**

把：

```markdown
基线：Plane v1.4.2，提交 `02c19e1`。本清单记录 Nerve 在表结构、接口和行为上与 Plane 的每一处差异，在整个 v0 期间持续更新。
```

替换为：

```markdown
基线：Plane v1.4.2，提交 `02c19e1`（`preview` 分支，`package.json` 中的版本是 1.4.2；`v1.4.2` 标签指向 `5f7d927`，两者的迁移文件逐字节相同，见 [`tools/plane-schema/README.md`](../../tools/plane-schema/README.md)）。本清单记录 Nerve 在表结构、接口和行为上与 Plane 的每一处差异，在整个 v0 期间持续更新。
```

- [ ] **Step 2: 修改"列级别的细节"一行（第 6 行），链接到快照文件**

把：

```markdown
- 列级别的细节（每张表逐列核对），由**建这张表的 M** 在编写迁移时补全，起点是 M0 生成的 Plane 表结构快照（`tools/plane-schema/`）。之后的变更，也由改动它的 M 负责登记。
```

替换为：

```markdown
- 列级别的细节（每张表逐列核对），由**建这张表的 M** 在编写迁移时补全，起点是 M0/P4 生成的 Plane 表结构快照 [`tools/plane-schema/plane-v1.4.2-schema.sql`](../../tools/plane-schema/plane-v1.4.2-schema.sql)。之后的变更，也由改动它的 M 负责登记。
```

- [ ] **Step 3: 修改"一、未保留的表"的第一段（第 12 行），写全 14 张系统表**

把：

```markdown
Plane 共有 96 张业务表（`db` 应用 92 张，`license` 应用 4 张）。Nerve 保留 44 张，其余 52 张不保留。Django 和 Celery 的系统表（`django_migrations`、`django_content_type`、`auth_group`、`auth_permission`、`users_groups`、`users_user_permissions`、`django_celery_beat_*`）同样不保留。
```

替换为：

```markdown
Plane 共有 96 张业务表（`db` 应用 92 张，`license` 应用 4 张）。Nerve 保留 44 张，其余 52 张不保留。Django 和 Celery Beat 的 14 张系统表（`django_migrations`、`django_content_type`、`django_session`、`auth_group`、`auth_group_permissions`、`auth_permission`、`users_groups`、`users_user_permissions`，以及 6 张 `django_celery_beat_*`）同样不保留。96 + 14 = 110，正是表结构快照中的全部表（M0/P4 核对）。
```

- [ ] **Step 4: 修改"二、保留表的改动 · 全局"中默认值一行，补上证据**

把：

```markdown
| 默认值、非空约束、枚举检查（优先级、状态组、角色、邀请状态、收集箱状态等）写进数据库 | Plane 只在 Python 代码中处理 |
```

替换为：

```markdown
| 默认值、非空约束、枚举检查（优先级、状态组、角色、邀请状态、收集箱状态等）写进数据库 | Plane 只在 Python 代码中处理：快照中没有任何列默认值，CHECK 约束只有 18 个由 Django 正整数字段生成的 `>= 0` |
```

- [ ] **Step 5: 在"二 · 按表"中新增 `issues` 的 pg_trgm 索引**

把：

```markdown
| `issues` | **新增**唯一约束 `(project_id, sequence_id)` | Plane 只靠咨询锁保证编号不重复 |
```

替换为：

```markdown
| `issues` | **新增**唯一约束 `(project_id, sequence_id)` | Plane 只靠咨询锁保证编号不重复 |
| `issues` | **新增** `name` 的 pg_trgm 索引（需要 `pg_trgm` 扩展） | 标题模糊搜索（v0-design 6.9）；快照中没有任何扩展，`issues` 上只有外键列的 btree 索引 |
```

- [ ] **Step 6: 在"二 · 按表"中新增 `issue_comments` 删除 `description_id`**

把：

```markdown
| `issue_labels` | **新增**部分唯一约束 `(issue_id, label_id)` | Plane 在数据库层没有这个约束 |
```

替换为：

```markdown
| `issue_labels` | **新增**部分唯一约束 `(issue_id, label_id)` | Plane 在数据库层没有这个约束 |
| `issue_comments` | 删除 `description_id` 及其唯一约束 | 它是指向 `descriptions`（不保留，见一 B）的一对一外键 |
```

- [ ] **Step 7: 核对修改，并在快照中确认新写的说法**

Run: `grep -c 'plane-schema/' docs/v0/plane-diff.md; grep -c 'auth_group_permissions' docs/v0/plane-diff.md; grep -c 'pg_trgm' docs/v0/plane-diff.md; grep -c '^| `issue_comments`' docs/v0/plane-diff.md; ls tools/plane-schema/README.md tools/plane-schema/plane-v1.4.2-schema.sql`
Expected（两个链接的目标都存在）：

```
2
1
1
1
tools/plane-schema/README.md
tools/plane-schema/plane-v1.4.2-schema.sql
```

在快照中确认 Step 3–6 的说法：

Run: `S=tools/plane-schema/plane-v1.4.2-schema.sql; grep -cE '^CREATE TABLE public\.(django_|auth_|users_groups \(|users_user_permissions \()' $S; grep ' DEFAULT ' $S | grep -vc 'GENERATED BY DEFAULT AS IDENTITY'; grep -c 'CHECK (' $S; grep -c 'EXTENSION' $S; grep -c 'ADD CONSTRAINT issue_comments_description_id_' $S`
Expected（依次是：14 张系统表；除 identity 列以外没有任何 `DEFAULT`；18 个 CHECK；没有扩展；`description_id` 上有唯一约束和外键各一个）：

```
14
0
18
0
2
```

Run: `git diff --stat`
Expected:

```
 docs/v0/plane-diff.md | 10 ++++++----
 1 file changed, 6 insertions(+), 4 deletions(-)
```

- [ ] **Step 8: 核对 M0 设计文档的 Phase 进度表**

Run: `grep -n '^| P4 ' docs/v0/M0-foundation/M0-design.md`
Expected（spec 和 plan 提交时已经更新，这里不需要改动；review 链接在代码评审后补上）：

```
491:| P4 | plane-schema | 进行中 | [spec](specs/P4-plane-schema.md) | [plan](plans/P4-plane-schema.md) | — |
```

- [ ] **Step 9: 提交**

```bash
git add docs/v0/plane-diff.md
git commit -m "docs(M0/P4): update the Plane diff list from the schema snapshot

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 10: 推送并确认持续集成（由控制者执行）**

推送分支，确认 `CI` 的 `server` 和 `web` 两个任务照常通过（P4 没有改动 Go、前端和持续集成；持续集成不运行 `make plane-schema`）。

---

## 完成后

P4 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M0-foundation/reviews/P4-plane-schema-review.md`：
- 复述 spec 2.2 的结论（官方镜像可用、结果确定、44 张表都在）；
- 裁定 spec 第 3 节的差异，特别是第 8 项（总体设计表头的基线说明、P5 迁入前端的来源提交）；
- 按 spec 第 7 节建立 handoff：交给 M2 的放进 `docs/v0/M2-auth/handoffs/`，交给 M4 的放进 `docs/v0/M4-issue-core/handoffs/`，交给 P5 的放进 `docs/v0/M0-foundation/handoffs/`；
- 按 spec 第 3 节末尾同步 M0 设计第 5.3、6.1、12 节和总体设计的表头；
- 把 M0 设计文档中 P4 的状态改为"已完成"，补上 review 链接。
