---
status: open
from: M0/P6
to: M8
created: 2026-09-22
---

# M8 确定发布流程时，与版本号和持续集成相关的注意事项

## 发布时版本号的来源

- P6 加入了版本号注入机制（`Makefile`：`VERSION ?= 0.1.0-dev`，`GO_LDFLAGS` 用 `-X …buildinfo.version=$(VERSION)`），但没有决定**发布时** `VERSION` 从哪里来（命令行手填？git 标签？）。M8 定下来源和发布流程（例如 `make build VERSION=0.1.0`，或从 git 标签派生）。
- `VERSION` 的默认值写了两处：`Makefile` 的 `VERSION ?= 0.1.0-dev` 和 `server/internal/platform/buildinfo` 里的默认版本号，改版本号时容易只改一处。`Makefile` 的注释已经写明两者应当相同；把这两处默认值合一（比如让其中一处读取另一处，或者只保留一处，交给构建脚本传参）。
- 持续集成靠注入与默认值不同的 `VERSION`（`0.0.0-ci.<运行编号>`）来发现注入失效（S3 核对）；合一默认值时不要破坏这个检测手段。

## `VERSION` 的校验和引用

- `Makefile:120,128` 的 `VERSION` 目前不加引号（P6 最终评审 Minor 4，延后到 M8）。补上引号，并加语义化版本校验：发布时如果 `VERSION` 包含空格或 shell 特殊字符，现在会被 `-ldflags` 原样展开，出错信息不直观。

## 必须通过的检查（required checks）

- 把 `server`、`web`、`e2e` 设为分支保护的必须检查之前，要按 M0 设计 6.3 的说明处理：GitHub 把被 `if` 条件跳过的任务也标记为 Success，同仓库分支的 PR 会跳过这三个任务。要么去掉现在的跳过条件，要么另加一个 `if: always()` 的汇总任务，把汇总任务设为必须检查。
- `e2e` 任务通过 `needs: [server, web]` 依赖另外两个任务，这个跳过条件会经 `needs` 传导过去，一并处理。

## Docker Hub 拉取频率限制

- 端到端测试拉取两个镜像：`postgres:18.6`（fixture 的数据库）和 `testcontainers/ryuk:0.14.0`（testcontainers 的回收容器，负责清理运行中途崩溃留下的容器）。两者都可能撞上 Docker Hub 匿名拉取的频率限制，目前没有负责人跟进；出现限流时，在持续集成中登录 Docker Hub 或改用镜像缓存（M0 设计第 12 节、P6 spec 第 6 节风险表）。

来源：[M0/P6 评审记录](../../M0-foundation/reviews/P6-e2e-ci-review.md)。
