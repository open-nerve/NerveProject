# Plane 表结构快照

`plane-v1.4.2-schema.sql` 是 Plane v1.4.2 在空库上跑完自带的全部 Django 迁移后，用 `pg_dump` 导出的表结构。Nerve 的每个 M 为自己的模块建表时，以它为起点，再按[差异清单](../../docs/v0/plane-diff.md)修改（[总体设计](../../docs/v0/v0-design.md) 5.6）。

这个文件是生成物，不要手改。文件末尾的空行也是 pg_dump 的原样输出（`git diff --check` 会提示 `new blank line at EOF`），不要删除。

## 快照信息

| 项 | 内容 |
|---|---|
| Plane 版本 | v1.4.2，标签提交 `5f7d92784c403f76284f0f16718f320221dc7fec` |
| 后端镜像 | `makeplane/plane-backend:v1.4.2@sha256:90032ce088708889b60c00d491897916f4deb882facda27db59fd10fb68729ef`，由标签提交构建 |
| Postgres | `postgres:15.7-alpine@sha256:468d34fefd6338031787c7b8e94078975b3aaf4d66c7ead25c39cd3ba46a15c6`（与 Plane 自带的 `docker-compose.yml` 相同），`pg_dump` 15.7 |
| 与参考源码的关系 | 仓库外的参考源码 `plane/` 是 `preview` 分支的 `02c19e1341d93141e8ad7b3278298adce208bafc`，比 v1.4.2 标签多 63 个提交。两者的迁移目录（128 个迁移文件和 2 个 `__init__.py`）逐字节相同，所以快照同样是 `02c19e1` 的表结构 |
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
- 结果是确定的：输入的两个镜像都按摘要写死，同样的输入得到逐字节相同的快照。已在 arm64 上核实，并在 arm64 主机上模拟 amd64 核实。
- 持续集成不运行它：要从 Docker Hub 拉取约 205 MB 的镜像，而输入都已写死，快照不会自己变化。

## 实现要点
- **镜像写成"标签@摘要"**：标签便于阅读，摘要保证拿到的永远是同一个镜像，即使上游重新推送了同名标签。摘要是多架构索引的摘要，amd64 和 arm64 都能用。
- **迁移只需要数据库**：后端镜像只设两个环境变量。`DATABASE_URL` 指向 compose 中的 Postgres；`REDIS_URL` 只是因为 `plane/celery.py` 在导入时就创建 Redis 客户端，值不能为空，而客户端只在第一次执行命令时才连接，所以指向一个不存在的主机 `unused.invalid`。不需要 Redis、RabbitMQ、MinIO。
- **不影响本机的其他项目**：每条 compose 命令都带 `-p nerve-plane-schema`（它优先于环境变量 `COMPOSE_PROJECT_NAME`，所以清理时不会误删别的 compose 项目）；不映射任何主机端口；Postgres 的数据目录放在 tmpfs 中，不产生数据卷；脚本退出时（包括出错和 Ctrl-C）删除本项目的容器和网络。开始时也先清理一次，防止上次被中断后留下的容器被重用。
- **pg_dump 用 Postgres 容器里的**：版本与服务端一致，本机不需要安装 `psql`。15.7 早于 15.14 引入的 `\restrict` 随机行，所以不需要 `--restrict-key`；以后把 Postgres 升到 15.14 或更高时，要加上 `--restrict-key` 并给一个固定值，否则每次输出都不同。

## 升级 Plane 版本时
1. 修改 `compose.yaml` 中两个镜像的标签和摘要（摘要取 `docker buildx imagetools inspect <镜像>:<标签>` 输出的 `Digest`）；Postgres 跟随新版本 Plane 自带的 `docker-compose.yml`。
2. 修改 `extract.sh` 中的输出文件名，删除旧快照，重新生成。
3. 用 `grep -rn 'plane-v1.4.2-schema' --exclude-dir=node_modules .` 找出引用旧文件名的地方（根目录 README、差异清单等），一并改为新文件名。
4. 更新本文件的快照信息，按新快照核对差异清单；差异清单、总体设计和前端改动清单中的基线说明一起更新。

官方镜像拿不到时，可以用 Plane 源码中的 `apps/api/Dockerfile.api` 在对应的标签上自己构建镜像，替换 `compose.yaml` 中的 `migrator` 镜像（未验证）。
