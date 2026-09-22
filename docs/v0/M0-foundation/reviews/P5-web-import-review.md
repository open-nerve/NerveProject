# M0/P5 前端迁入与内嵌：评审记录

| 项 | 内容 |
|---|---|
| Phase | M0/P5 `web-import` |
| 日期 | 2026-09-22 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P5-web-import.md) / [plan](../plans/P5-web-import.md) |
| 分支 | `worktree-m0-p5-web-import`，从 `main` 的 `83c8b49` 分出，提交从 `859f813` 到本评审记录所在的提交 |

## 1. 范围与结果

P5 把 Plane 的前端原样迁入 `web/`，能安装、检查、构建，并编进 `nerve`（spec 1）：

- `web/apps/web` 和 12 个 packages，从 Plane 提交 `02c19e1` 原样迁入；`patches/react-color@2.19.3.patch`。
- 根目录的 pnpm 工作区、turbo 配置：与已有的 `@redocly/cli`、`@nerve/api-client` 合并；`pnpm-lock.yaml` 以 Plane 的锁文件为起点生成。
- lint 警告基线（只降不升），类型检查、lint、格式检查统一由 turbo 驱动。
- Vite 开发服务器把 `/api` 转发给本地 Go 服务，`make web-dev`。
- `server/internal/platform/webui`：内嵌 `dist/`，按缓存策略提供静态文件，单页应用回退到 `index.html`。
- `make build` 构建出内嵌前端的 `bin/nerve`；CI 的 `web` 任务加上 lint 基线、格式检查和前端构建。
- 前端改动清单登记来源提交和迁入时的改动。

spec §4 的 8 项验收标准全部满足：迁入的 13 个目录和 `patches/` 与 `02c19e1` 树对象相同；`pnpm install --frozen-lockfile` 在干净克隆上成功；`make lint-web`、`make build`、`make lint-go`、`make test` 全部通过；手工验证的响应符合 §2.9 的规则表；CI 的 `server`、`web` 两个任务都通过；前端改动清单和三个交给 P5 的 handoff 都已处理。

## 2. 原型验证的结论（spec 2.2）

- **迁入的内容**：`apps/web` 加 12 个包（constants、editor、hooks、i18n、propel、services、shared-state、types、ui、utils、tailwind-config、typescript-config）；13 个目录和 `patches/` 的 git 树对象与 `02c19e1` 中对应的树完全相同。
- **锁文件**：以 Plane 的 `pnpm-lock.yaml` 为起点生成，13 个 importers 与 Plane 逐行相同；三次独立生成结果都是 13835 行，SHA-256 `d677458d…18ca`。
- **构建产物**：`web/apps/web/build/client/` 1239 个文件、34 MB；内嵌后的 `bin/nerve` 50.5 MB（不含前端时 18.0 MB）。
- **lint 警告基线**：合计 1005 条（web 779、editor 75、propel 59、utils 34、ui 32、services 6、hooks 4、i18n 3、constants 2、types 1、shared-state 0、api-client 0），远低于 Plane 原有的上限。

## 3. 各 Task 的评审

| Task | 内容 | 提交范围 | 核实方式 |
|---|---|---|---|
| 1 | 迁入 Plane 的 web app 和 12 个包、`patches/`、锁文件 | 859f813..48e1a63 | 实现者 sonnet；评审（sonnet），clean：14/14 目录树对象与 `02c19e1` 相同，锁文件 13835 行、SHA `d677458d…`，importer 切片与 Plane 相同，没有 fetch 残留 |
| 2 | lint 警告基线、turbo 驱动的检查任务 | 48e1a63..4599a3d | 实现者 sonnet；评审（sonnet），clean：警告上限与表格一致，配置文件 = Plane 原文加已登记的改动，Makefile 该提交时有 25 行 Tab |
| 3 | Vite 开发代理 `/api` → `127.0.0.1:8080` | 4599a3d..58c0d10 | 实现者 sonnet；评审（sonnet），clean：`vite.config.ts` 唯一改动就是代理块，改动清单逐行核对，来源 SHA 已记录 |
| 4 | `platform/webui`：内嵌与单页应用回退 | 58c0d10..027b78a | 实现者 sonnet；评审（sonnet），clean：文件逐字节比对，规则顺序符合 spec 2.9，没有目录遍历或列目录，挂载不遮蔽探针和 `/api`；Minor 已 parked（`assets/` 外嵌套点号路径缺测试，走同一代码路径） |
| 5 | `make build`：内嵌前端并编译 `bin/nerve` | 027b78a..106f887 | 实现者 sonnet；评审（sonnet），clean：recipe 在 Make 3.81、BSD find/cp 上可移植，失败会传播，`.gitignore` 覆盖全部产物；确认 turbo 的共享 worktree 缓存是真实存在的 |
| 6 | CI `web` 任务、README、关闭 3 个交接给 P5 的 handoff | 106f887..0bb0d01 | 实现者 sonnet；评审（sonnet），clean：CI 步骤顺序、缓存接线、`if`、超时经 YAML 解析核实；3 个 handoff 的处理结果对照代码验证；没有版本注入的说法 |

Task 1 另有两件事：
- 实现提交 `7e01453` 的 trailer 写错了 `Claude Sonnet 5`，amend 为 `48e1a63` 修正。
- Ruling 4：实现者在沙箱中用 `git fetch --update-shallow <plane> 02c19e1` 取源码，把共享仓库变成了 shallow（`.git/shallow` 指向一个任何引用都到不了的 Plane 提交）。控制者移除了 `.git/shallow`，`git fsck --connectivity-only` 干净，未引用的 Plane 提交对象留给 gc 回收；此后实现者的指令改为禁止 `fetch`，并把 trailer 字符串写死。

## 4. 整分支评审（opus）：With fixes，没有代码缺陷

评审范围 `83c8b49..0bb0d01`（评审包 3896 行；排除逐字节复制的 Plane 源码、锁文件、`patches/`、plan，保留各包的配置文件）。

| # | 类别 | 发现 | 处理 |
|---|---|---|---|
| Important 1 | Important | spec §4.7 需要一次暖缓存的 CI 运行，此前只记录了冷运行 `35695878413` | 控制者在合并前把收尾提交推到分支上，补充暖缓存运行的记录（第 6 节） |
| Important 2 | Important | 本地建 `web/apps/web/.env` 会被 `vite.config.ts` 的 dotenv 悄悄加载，把 `VITE_API_BASE_URL` 打进构建产物，破坏同源部署 | 已修（docs(M0/P5): address the final review）：README 加"不要建立 `.env`"的说明；M1 handoff 改为整体删除 `.env.example`，并复查 dotenv 加载 |
| Minor 1 | Minor | spec 101 行措辞：两条 postcss-selector-parser 的 overrides 不是"目标包不在依赖图中"，包本身在图里，只是版本区间选择器匹配不到任何一个版本 | 已修：改写措辞，把这两条与另外 11 条"包确实不在图中"的条目分开描述 |
| Minor 2 | Minor | README:61 从仓库根目录执行 `NERVE_ENV=dev bin/nerve serve` 读不到 `server/configs/config.local.yaml`（`localConfigFile` 相对当前工作目录解析） | 已修：改为 `cd server && NERVE_ENV=dev ../bin/nerve serve` |
| Minor 3 | Minor | README:63 的 `--output-logs=errors-only` 看不到警告数，也没写怎么修格式 | 已修：补充 `pnpm --filter <pkg> run check:lint`（打印 `Found N warnings`）和 `pnpm exec turbo run fix:format` |
| Minor 4 | Minor | frontend-changes §二缺 M1 删除部署遗留（`Dockerfile.*`、`caddy/`、`.dockerignore`、`serve`、`sw.js`/workbox、`.env.example`）的计划中行 | 已修：补充 4 行 |
| Minor 5 | Minor | `pnpm-workspace.yaml` 里有几处过时的 Plane 英文注释（`.npmrc` 设置、Express 4、runtime images） | 已修：扩大 M1 handoff 的范围，把这些注释一并列入 |
| Minor 6 | Minor | `make build` 之后，`make run` 和 `go test` 会继续内嵌旧的前端构建 | 已修：README 加清理命令 `find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete` |
| Minor 7 | Minor | 没有测试 `assets/` 外的嵌套点号路径；`/readyz` 只被间接检查 | Parked：和已有的隐藏文件测试走同一代码路径，测试价值低 |

### 4.1 修复后的复核（控制者）

修复提交 `8e462ad` 只改文档（47 行），由控制者直接复核：Important 2 和 Minor 1–4、6 全部处理到位（README 的 `.env` 提醒、在 `server/` 下运行 `bin/nerve`、查看警告数和修格式的命令、清掉旧构建产物；spec 2.4 的 postcss-selector-parser 措辞；前端改动清单 M1 的四行计划）。Minor 5 写进了 M1 的交接（`a7d0ec5`）。没有引入新问题。

## 5. 控制者裁定

| Ruling | 内容 |
|---|---|
| 1 | spec §3 第 1–16 项按原样接受：不迁入 `.npmrc`；catalog/overrides/allowBuilds/globalEnv 只保留作用于迁入的包的条目，以锁文件为证据；lint 上限改为实测值；`check:format` 并入 `lint-web`；不加 `web.enabled`；未构建时用英文提示、404；`assets/` 缺失返回 404，隐藏文件不提供；`public/` 文件 `no-cache`；`build-web` 拆分；CI `web` 任务只到 `build-web`，完整 `make build` 留给 P6 的 `e2e`；turbo 缓存只在任务内部使用；`patches/` 放根目录；不建 Vite 模式文件；构建产物路径不变；M0 期间首页显示 Plane 的启动错误页；锁文件以 Plane 的为起点——每条都有原型证据支撑 |
| 2 | `make build` 的版本注入（S3）留给 P6：P6 需要时自己加 `-ldflags -X …buildinfo.version`，P5→P6 的 handoff 已写明 |
| 3 | 执行分工：六个实现者都用 sonnet（要跑 pnpm/turbo/docker/server 等重命令，且需要对差异做判断）；评审都用 sonnet；整分支评审用 opus。评审包排除逐字节复制的 Plane 源码和锁文件，改用树对象比对和 Task 1 的锁文件核对来把关 |
| 4 | Task 1 的沙箱变通方法把共享仓库变成了 shallow；控制者移除 `.git/shallow` 并核实 `fsck` 干净；此后实现者指令禁止 `fetch`，并把 trailer 字符串写死 |
| 5 | 修复轮 = Important 2 + Minor 1–6，作为纯文档改动（README、spec §2.4 措辞、frontend-changes §二 计划中行）合并成一个提交，由收尾阶段的 agent 完成；Minor 7 parked（同一代码路径，只是测试缺口）；Important 1 靠把收尾提交推到分支上、在合并前补记暖缓存运行来满足 |

## 6. CI 证据

- **`35695878413`**（提交 `0bb0d01`，冷 pnpm 缓存）：成功。`web` 142 秒（install 12 秒、gen-check 2 秒、lint-web 87 秒、build-web 24 秒、cache save 5 秒）；`server` 45 秒。
- **`35698392213`**（提交 `a7d0ec5`，暖 pnpm 缓存，spec 4.7 的第二次运行）：成功。`web` 116 秒（恢复缓存 4 秒、install 11 秒、gen-check 2 秒、lint-web 68 秒、build-web 18 秒）；`server` 38 秒。命中缓存的依据：恢复步骤 4 秒（冷运行时未命中，1 秒），保存步骤 0 秒（主键命中时不再保存；冷运行时保存用了 5 秒）。
- 缓存按分支（ref）隔离：合并后 `main` 上的第一次运行仍是冷缓存。

## 7. 移交事项

**本次创建的 handoff（3 个）：**

| handoff | 去向 | 内容 |
|---|---|---|
| [M0-P5-frontend-trim-notes](../../M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md) | M1 | 部署遗留和未用依赖的删除范围、`.env.example` 整体删除、`pnpm-workspace.yaml` 的过时注释、重新测警告基线、React #418 等已知行为差异 |
| [M0-P5-frontend-api-notes](../../M2-auth/handoffs/M0-P5-frontend-api-notes.md) | M2 | 前端仍调用 Plane 接口的现状、`webui` 对非 `/api` 路径一律回答 `index.html`（含 `/.well-known/*`）、安全响应头待定 |
| [P5-web-import-p6-notes](../handoffs/P5-web-import-p6-notes.md) | M0/P6 | `e2e/` 需要自己的检查脚本、turbo 严格环境模式、构建产物是否要跨 job 传递、按 ref 隔离的缓存、版本注入归属、`.env` 的注意事项 |

**本次关闭的 handoff（3 个，均在 Task 6 处理）：**

- [P2-server-platform-p5-webui-mount](../handoffs/P2-server-platform-p5-webui-mount.md)：`webui` 挂载为不带方法的 `/`，只处理 GET/HEAD，未加入 `web.enabled`。
- [P3-api-contract-p5-notes](../handoffs/P3-api-contract-p5-notes.md)：lint/格式检查排除生成文件，`lint-web` 改由 turbo 驱动，保留 `@redocly/cli`，api-client 的 `typescript` 改为 `catalog:`。
- [P4-plane-schema-p5-notes](../handoffs/P4-plane-schema-p5-notes.md)：前端从 `02c19e1` 迁入，复制前核实提交存在，前端改动清单记下完整 SHA。
