# M5 / P2 资产用例与生命周期实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检阻断草稿，不可执行。仅记录规划输入和隔离验证；没有开始 T1。

**Goal：** 在 P1 实物存储合同上接入资产业务与完整生命周期。

**Architecture：** asset 自有数据与对象状态；权限由宿主模块提供，bootstrap 唯一接线，外部存储 I/O 不进入数据库事务。P2 首张表的物理删除约束正在与 M4 的父级 CASCADE 合同联合核对，未决前不冻结迁移、清理端口或实施任务。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/River/local；图片验证的官方 WebP 依赖只作预检，尚未安装。

**Spec：** [P2-assets](../specs/P2-assets.md)。

## Global Constraints

- 主目录保持 main；仅在 codex/m5-p2 worktree 准备，已快进 P1 合并提交 85dc4087；其准确 main CI 尚待核，不提前作为通过的实施基线。
- 不修改产品、迁移、守卫或依赖来绕过未决问题；隔离数据库探针与源码原件进入本计划证据目录，执行后移除临时探针源文件。
- 物理对象删除失败必须保留恢复记录；不能以父级 CASCADE 抹掉它，也不能未经决策放宽现有守卫。
- 重大架构冲突先提交已完成文档与证据摘要，再上报；未决时不标预检通过。

## Review Focus

本段先核清理合同：超过 60 天的父墓碑遇到未处理子资产、外部删除失败、SET NULL 与用途 CHECK、有限批次与邻域前进；结论写入预检记录。完整锁序、状态机、API、图片限制和不超过 16 个 Task 的可执行计划在此问题裁定后补齐，不把本草稿当已通过预检。
