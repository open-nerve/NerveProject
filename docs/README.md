# Nerve 文档约定

本目录存放 Nerve 的全部设计与过程文档。任何人（或 Agent）开始工作前，先读本文件，再读当前版本的总体设计。

## 目录结构

```
docs/
  README.md                        本文件：文档约定
  v0/                              版本目录：v0 = 启动首版（M0–M8）
    v0-design.md                   版本总体设计 + 各里程碑（M）进度表
    plane-diff.md                  与 Plane 的差异清单（表结构 / 接口 / 行为）
    frontend-changes.md            前端改动清单（相对 Plane 前端）
    M<n>-<slug>/                   里程碑目录，例如 M0-foundation
      M<n>-design.md               里程碑设计总文档（开始该 M 时创建）
      specs/                       各 Phase 的设计说明
      plans/                       各 Phase 的实施计划
      reviews/                     各 Phase 的评审记录
      handoffs/                    交接事项（提醒、TODO、移交给后续阶段的工作）
```

## 层级与节奏

- **版本（v0、v1…）**：一个版本 = 一组里程碑。版本总体设计定义范围、架构和路线。
- **里程碑（M0、M1…）**：一个 M 一个 M 串行推进。开始某个 M 时，先在它的根目录写 `M<n>-design.md`，补充这个 M 的细节：用户故事清单、Phase 划分、Phase 进度表。
- **Phase（P1、P2…）**：M 内部再切分成多个 Phase，串行推进。每个 Phase 依次产出 spec → plan → 实现 → review。需要移交的事项写成 handoff。

## 命名规则

- 目录名、文件名用英文 kebab-case；正文用中文，技术名词保留英文。
- 同一个 Phase 在四个子目录中使用**同一个前缀和 slug**，便于对照：
  - `specs/P2-list-engine.md`
  - `plans/P2-list-engine.md`
  - `reviews/P2-list-engine-review.md`（第二轮评审加 `-r2`，依此类推）
  - `handoffs/P2-list-engine-<主题>.md`
- Phase 编号只在所属 M 内部有效，跨文档引用写成 `M4/P2`。

## 进度追踪（唯一来源）

- `v0-design.md` 中的"里程碑进度表"记录 M0–M8 的状态。
- 每个 `M<n>-design.md` 中的"Phase 进度表"记录本 M 各 Phase 的状态，并链接到对应的 spec / plan / review。
- 状态取值：`未开始` / `进行中` / `已完成`。

## handoff 规则

每个 handoff 文档开头写明：

```yaml
status: open        # open | done
from: M2/P3         # 在哪里产生
to: M4              # 应由哪个阶段处理
created: 2026-09-22
```

- **移交给其他 M 的事项，直接放进目标 M 的 `handoffs/` 目录**，并在来源 Phase 的 review 中链接过去。这样开始那个 M 时，就能看到它要接手的事项。
- 开始一个 M 之前，先处理它 `handoffs/` 中所有 `open` 的事项。
- 一个 M 完成前，不能留有 `open` 的 handoff。每一项要么处理完（改为 `done`），要么明确移交给后续 M。

## 其他

- 空目录放 `.gitkeep`，保证目录结构能提交进 git。
- 参考代码（Plane 上游源码等）放在仓库根目录的 `plane/`、`refer/` 中，已加入 `.gitignore`，不进仓库。
