# 06 问题记录：持续集成上 W11 页面故事偶发失败

| 项 | 内容 |
|---|---|
| 状态 | **未解决，根因未明** |
| 首次出现 | main 的持续集成第 340 次（P3 的合并 `dfc9378b`，2026-10-24），`actions/runs/40018430648` |
| 之后的运行 | 第 341 次（`f25834bd`，`actions/runs/40121644406`）、第 342 次（`56d575fb`，`actions/runs/40122129200`）全部通过，W11 也通过；这两次相对 P3 的合并只多了本目录的文档，所以是偶发的 |
| 负责人的决定（2026-10-25） | 不阻塞主线，不专门去查。等它阻塞主线推进时（某次合并的持续集成因它失败、重跑也不过），由负责人协调 Codex 一起解决 |
| 记录人 | Claude：本文给出问题背景、已有的证据、排除过的方向、下次要收集的东西 |

## 1. 现象
- e2e 任务：125 个通过、1 个失败；server、web 两个任务通过。
- 失败的是 M3 的 W11 页面版本：`e2e/stories/workspace/w11-guest-bounds.spec.ts:101`，失败在第 190 行。
- 报错：

```text
Error: expect(locator).toHaveCount(expected) failed
Locator:  locator('[role="dialog"][aria-modal="true"]')
Expected: 1
Received: 0
Timeout:  5000ms
  14 × locator resolved to 0 elements

  188 |     await expect(theAdmin.page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();
  189 |     await theAdmin.page.keyboard.press(keys);
> 190 |     await expect(modalDialog(theAdmin.page)).toHaveCount(1);
      at endsOnBack (w11-guest-bounds.spec.ts:190:46)
      at w11-guest-bounds.spec.ts:195:10
```

- 第 195 行是 `endsOnBack("ControlOrMeta+k")`：管理员按 Ctrl+K 之后，命令面板 5 秒内一直没有出现。
- 报告里有：两个页面的失败截图、数据库快照、`error-context.md`、`trace.zip`。这些都在那次运行的 `playwright-report` 产物里，下载要负责人的权限。

## 2. 故事在这一步做什么（第 170–196 行）
这一段验证"切到另一个工作区再 Back 回来时，在原工作区打开的弹窗随它关掉"（v0 设计 7.7，按工作区键控的内容）。管理员有两个工作区 Acme（`slug`，有 Web、Secret 两个项目）和 Beta（`betaSlug`，没有项目）：

1. 第 176–182 行：
   1. `moveWithinApp` 到 Beta 的项目页，再到 Acme 的项目页；
   2. 点 "Add Project"，等到弹窗出现；
   3. Back 回 Beta 的项目页；
   4. `closedWithin` 确认弹窗一秒内关掉。
2. 第 186–196 行，`endsOnBack(keys)`：
   1. `moveWithinApp` 到 Acme 的项目页；
   2. 等 "Add Project" 可见；
   3. 按 `keys`；
   4. 期望出现一个模态弹窗；
   5. Back 回 Beta，确认它关掉。

   先用 `ControlOrMeta+k`（命令面板），再用 `ControlOrMeta+/`（快捷键列表）。

`moveWithinApp`（`e2e/fixtures/settings-pages.ts:273`）用 `history.pushState` 加一个 `popstate` 事件，让应用的路由自己切换。

## 3. 相关的应用代码
- `web/apps/web/app/(all)/[workspaceSlug]/layout.tsx`：工作区的内容以 `key={params.workspaceSlug}` 键控，换工作区时整个侧栏、头部、弹窗和页面重新挂载。
- `web/apps/web/app/(all)/[workspaceSlug]/(projects)/layout.tsx`：挂 `ProjectsAppPowerKProvider`。
- `web/apps/web/core/components/power-k/projects-app-provider.tsx`：
  - 第 41 行的 `useLayoutEffect` 在卸载时 `togglePowerKModal(false)`，即"面板随它的工作区关掉"；
  - 渲染面板的宿主 `ProjectsAppPowerKModalWrapper`。
- `web/apps/web/core/components/power-k/global-shortcuts.tsx`：
  - 第 62 行的 `useEffect`（被动效果，在绘制之后才运行）把 `ShortcutHandler.handleKeyDown` 挂到 `document` 的 `keydown`；
  - 卸载时摘掉。
- `web/apps/web/core/components/power-k/core/shortcut-handler.ts`：Cmd/Ctrl+K 总是 `openPalette()`，即 `togglePowerKModal(true)`，在判断"是否在输入框里"之前。
- `web/apps/web/core/store/power-k.store.ts`：
  - `isPowerKModalOpen` 是全局 store 的一个布尔值；
  - 没有任何条件会拒绝打开。
- `web/packages/ui/src/modals/modal-core.tsx`：
  - `open={isOpen && mounted}`，`mounted` 在第 71 行的 `useEffect` 里置真，即从挂载之后的那次渲染才打开；
  - Headless UI 的 `Dialog` 打开时带 `role="dialog"`、`aria-modal="true"`。
- 关掉面板的路径只有这些：
  - 用户的动作：Escape、点背景、面板里再按 Cmd/Ctrl+K、选中命令；
  - 宿主卸载。

  没有找到会自己关掉的路径。

## 4. 已经排除或没能复现的（2026-10-24/25，macOS）
- **与 P3 的改动无关**：
  - P3 在前端只改了 `error-messages.ts`、两份 `errors.json` 和生成的 `schema.gen.ts`，没有改这个故事、命令面板、`ModalCore` 和布局；
  - 第 336–339 次运行这个故事都通过。
- **照持续集成的方式跑整套**（`make build`，然后 `CI=1 playwright test --project=chromium --workers=2`）：一遍 124/124；连跑 3 遍（`--repeat-each=3`）372/372。I 系列故事连跑 5 遍 30/30。
- **CPU 降速**：
  - 把 W11 复制一份，用 CDP `Emulation.setCPUThrottlingRate` 给管理员的页面降速：4 倍、6 倍各 5/5 通过；10 倍时在别的步骤超时，没有意义。
  - 专门的探针（`palette-race.spec.ts`，见第 7 节）：降速 1、4、6 倍，切到 Acme 之后马上按键（等或不等 "Add Project"），面板都打开并保持打开，10/10。
- **Linux 的按键**：持续集成在 Linux 上，`ControlOrMeta` 是 Ctrl；本机 macOS 上是 Meta。改成 `Control+k`、`Control+/` 跑 5 遍，5/5 通过。按键时焦点在 `BODY`。
- **一个误导人的观察**：降速时，按键那一刻工作区切换按钮的文字还是 "Beta"，而 Acme 的项目 "Web" 已经显示。切换按钮的文字读的是 store 的当前工作区，晚于内容的挂载更新，所以它不能说明 Beta 的内容还在。

## 5. 没有证实的假设（按可能性）
1. **按键时页面上还是 Beta 的内容，键落在 Beta 的监听上。**
   - 等待条件 "Add Project" 可见在 Beta 的项目页上同样成立，分不出两个工作区。React Router 的导航若以过渡（transition）渲染，`moveWithinApp` 返回之后 Beta 的内容可能还在屏幕上。
   - 这时 Ctrl+K 由 Beta 的监听打开面板；随后 Acme 的内容提交，Beta 的宿主卸载，`togglePowerKModal(false)` 把面板关掉。
   - 面板只在两次提交之间存在，Playwright 的轮询都没赶上，就是"14 次都是 0 个"。
   - 本机没复现：本机上 `moveWithinApp` 返回时 Acme 的内容已经提交。
   - 若属实，问题在测试的等待条件：等 Acme 独有的东西，例如它的项目 "Web" 出现。第 177–178 行"Add Project"的点击有同样的隐患。
2. **键落在新内容已绘制、监听还没挂上的空隙里。** `global-shortcuts.tsx` 在被动效果里挂监听，绘制之后才运行。理论上在提交和被动效果之间，旧的监听也还在（卸载的清理也在被动效果里），所以键不会丢。若 trace 显示键确实丢了，可以考虑把挂监听改成 `useLayoutEffect`，让"页面画出来时它的快捷键已生效"成立，这是应用一侧的加固。
3. **焦点在一个拦下 `keydown` 冒泡的元素上，键没到 `document`。** 例如创建项目的弹窗被 Back 卸载之后，Headless UI 把焦点还给了别的元素。本机按键时焦点在 `BODY`。

## 6. 下次出现时收集什么
- 那次运行的 `playwright-report` 产物（负责人下载）。用 `pnpm exec playwright show-trace <trace.zip>` 打开管理员页面的 trace，看：
  - 第 188 行的等待通过时，以及第 189 行按键前后的 DOM 快照：是 Acme 还是 Beta 的内容？可以看侧栏里有没有 Acme 的项目 "Web"，页面上有没有项目卡片。
  - 按键之后有没有短暂出现 `[role="dialog"]`；
  - 控制台的报错；
  - 那段时间的网络请求。
- 失败截图 `test-failed-1.png`、`test-failed-2.png`：哪个是管理员的页面，画面停在哪个工作区。
- 同一次运行里别的故事有没有类似的"按键或点击之后弹窗没出现"。
- 若能重跑：只重跑失败的任务，看它是否稳定复现；是偶发的话，记下次数。

## 7. 复现的工具（本机，没有进仓库）
在 `~/.claude/projects/-Users-xiaoruan-project-nerve-project/carry-tools/m4-handoff/w11-probes/`，脚本里的路径指向已删除的 P3 工作树，用之前改成主目录：

| 文件 | 内容 |
|---|---|
| `e2e-ci-like.sh` | 照持续集成的方式 `make build` 之后跑 chromium、2 个 worker |
| `e2e-repeat-all.sh` | 整套 `--repeat-each=3` |
| `make-w11-probe.py`、`probe-w11.sh` | 复制一份 W11，给管理员的页面加 CDP 降速，跑 5 遍 |
| `make-w11-probe2.py`、`probe-keys.sh` | 复制一份 W11，按键改成 Linux 的 `Control+…`，记下按键时的焦点，跑 5 遍 |
| `palette-race.spec.ts` | 只测"切到另一个工作区、按 Ctrl+K、面板是否打开并保持"，可选降速、可选不等待 |

探针要放进 `e2e/stories/zz-probe/` 才能被 Playwright 找到，用完删除，不提交。
