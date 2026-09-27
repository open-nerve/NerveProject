# M2/P5 个人设置 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 个人设置的四个标签页（general 含停用账户、preferences、security 含 PAT 列表、api-tokens）都用新接口；PAT 的 store 属于会话的 `RootStore`，用这一代的客户端；设置页的下拉框在按钮旁展开；维护页不再暗示 nerve 没有启动；`@nerve/services` 只剩地址和文件工具，Plane 剩下的令牌、时区类型删除；M1 收尾交接的死成员、死 prop、oxlint 在 M2 的范围内清完，`no-unneeded-ternary` 全仓清零；A7–A9、A11、A12 的页面版本通过。

**Architecture:** PAT 的 service（`core/services/api-token.service.ts`）从构造函数拿到客户端；PAT store（`core/store/user/api-token.store.ts`）挂在 `UserStore` 下，`RootStore` 建它时传入这一代会话的客户端，所以换会话就换 store，旧会话的请求以 `SessionChangedError` 结束，不会以新账户的身份发出。列表逐页取完才显示；创建的响应带令牌原文，只交给调用方（弹窗），store 的列表不含原文。页面组件经 `useApiTokens()` 读 store，SWR 的键带 `loginId`。时区列表经 `publicClient` 调公开的 `GET /api/v0/timezones`。下拉框的根因在 Headless UI 2.2 的 `Combobox.Options`：它克隆唯一的子元素时换上自己的 ref，放在子元素上的 react-popper 的 ref 从未被设置，列表停在页面左上角；改为把 popper 的 ref、样式和属性放在 `Combobox.Options` 本身，并让组件自己的打开状态跟随 Headless UI 的关闭。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、react-hook-form 7.84、Headless UI 2.2.10、react-popper 2.3.0、date-fns 4.1、openapi-fetch 0.17.0、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1、golangci-lint 2.13.2（Go 代码不变）。

**Spec:** `docs/v0/M2-auth/specs/P5-web-account.md`（上级：`docs/v0/M2-auth/M2-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 包，Go 依赖不变。Task 3 从 `@nerve/services` 删掉 `axios` 依赖（web 和其他包仍用它）：改完 `web/packages/services/package.json` 之后执行 `pnpm install`，`pnpm-lock.yaml` 的变化只有 Task 3 差异块中的三行；不一致时停下来。
- **每个 Task 提交前**：`make lint-web`（关键词守卫；各包的类型检查、oxlint 警告数等于上限、格式、中英文翻译键一致）、`make knip`、`make test-web` 都通过。oxlint 上限（各包 `package.json` 的 `check:lint`）：web 在 Task 1、2、3、6、8、9、10 依次调为 550、548、543、541、537、514、457；ui 在 Task 1、10 调为 20、19；utils 在 Task 9 调为 16。`tools/lint-cap.mjs` 要求警告数正好等于上限，这些数就是删掉带警告的代码之后的结果；对不上时停下来，说明某处与本 plan 不一致。Task 4–7 另执行 `make e2e`（它们改故事）。Task 11 另执行 `make lint-go`（两段都是 `0 issues.`）、`make test`，提交之后执行 `make gen-check`（本 phase 不改生成物，它应当没有输出差异）。
- **没有生成的文件**：本 phase 不改 `api/`、`server/`，也不改任何生成物；`make gen` 之后 `git status` 应当干净。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合；用绝对路径或 `pnpm -C <目录>`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**（负责人的长期规则）：
  - 前端没有兼容层：接口的数据结构直接用 `@nerve/api-client` 生成的类型，不写转换；Plane 的手写实体类型随对接删除；
  - 删掉的东西删干净：没有开关、占位、注释掉的代码；关键词守卫在删除的同一个提交里加上规则（M1 设计 7.4）；删掉一个使用者时，查它用过的每个 `@nerve/*` 导出，没有别的使用者的一并删除（knip 看不见包的公开导出）；
  - 沿用现有的写法：MobX store、M1 留下的 React Router 写法、P4 的会话约定（service 从构造函数拿这一代的客户端，没有模块级的带令牌客户端；store 只经自己的 `RootStore` 找兄弟 store；填 store 的 SWR 键带 `loginId`；`SessionChangedError` 不退出、不跳转，后台加载忽略它，用户的操作可以提示失败）、`t()` 取文案（中英文的键一致）；
  - 一个文件只做一件事，不超过约 400 行；
  - 单元测试凡是声称顺序或竞争的，都用假时钟或手动的闸门强制顺序，不用 sleep；每个等待都有期限；单元测试不在自己的时限内冷加载大的模块图（在文件顶层导入，或在带期限的 `beforeAll` 里）；没有 DOM 测试环境（不加 jsdom），组件由端到端测试和浏览器核对覆盖，值得单元测试的逻辑放进组件旁的纯函数；
  - Go、TS 代码的注释用英文；中文文档照本 plan 原样。
- **工具的坑**：Edit、Write 工具曾把字面的 `<` 写成 `&lt;`。写完 TSX、TS 文件后 `grep -rn "&lt;\|&gt;\|&amp;"` 这次改过的文件，应当没有输出。
- **代码块**：标为"新文件"或"完整内容"的块是**完整的文件内容**，照原样写入，不要改动（原型中逐字节运行过）；标为"差异"的块是对这个文件当前版本（`cec4ec9` 或前一个 Task 写的版本）的统一差异，照差异修改，改完的文件与原型逐字节相同。差异块可以存成文件后在仓库根目录用 `patch -p1 < <文件>` 应用（块里的路径是 `a/<路径>`、`b/<路径>`；拼 plan 的脚本已用 `patch -p1` 逐个核对过，113 个差异块都得出原型中的文件）。
- **删除**：标为"删除"的文件用 `git rm` 删掉，之后用给出的 `grep` 核对没有残留的引用。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再写成最终版本；每个过渡版本都在原型的逐 Task 复现中运行过（spec 附录 A）。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录，`web/apps/web/` 简写为 `web:`。"过渡"表示这个 Task 写过渡版本，后面的 Task 写最终版本；"删除"表示这个 Task 删掉它。

| 文件 | 职责 | Task |
|---|---|---|
| `web/packages/ui/src/dropdowns/custom-select.tsx`、`custom-search-select.tsx`、`combo-box.tsx`；`web:core/components/dropdowns/date.tsx`（修改） | 列表在按钮旁展开；打开状态跟随 Headless UI 的关闭；搜索下拉框不设 inert | 1 |
| `web/apps/web/package.json`（修改） | oxlint 上限 | 1、2、3、6、8、9、10 |
| `web/packages/ui/package.json`（修改） | oxlint 上限 | 1、10 |
| `web:core/services/api-token.service.ts`、`web:core/store/user/api-token.store.ts`、`api-token.store.test.ts` | PAT 的接口调用和 store | 2 |
| `web:core/store/user/index.ts`、`web:core/store/workspace/index.ts`（修改）；`web:core/store/workspace/api-token.store.ts`（删除） | PAT store 挂到 `UserStore` | 2 |
| `web:core/lib/auth/fake-nerve.ts`、`web:core/lib/store-context.test.ts`（修改） | 假 nerve 记下查询参数；换会话时 PAT store 的测试 | 2 |
| `web:core/components/api-token/`（`token-list.tsx`、`modal/expiry.ts`、`modal/expiry.test.ts` 新建；其余修改）、`web:core/components/ui/loader/settings/api-token.tsx`、`web:core/components/settings/profile/content/pages/api-tokens.tsx`、`web:core/hooks/store/user/`（修改、新建 `user-api-tokens.ts`） | api-tokens 页 | 3 |
| `web/packages/services/`（删除 `api.service.ts`、`developer/`；修改 `package.json`、`src/index.ts`）、`pnpm-lock.yaml`（修改） | `@nerve/services` 只剩地址和文件工具 | 3 |
| `web/packages/types/src/api_token.ts`（删除）、`index.ts`（修改）；`web/packages/constants/src/fetch-keys.ts`（修改） | Plane 的令牌类型和 SWR 键 | 3（`index.ts` 过渡）、6 |
| `web/packages/i18n/src/locales/{en,zh-CN}/settings.json`、`workspace-settings.json`（修改） | 文案 | 3（`settings.json` 过渡）、5 |
| `tools/keywords.json`（修改） | P5 的三条规则；`services-files` 收窄 | 3（过渡）、6 |
| `e2e/fixtures/settings-pages.ts` | 个人设置的故事的操作和断言 | 4（过渡）、5 |
| `e2e/fixtures/browser.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`（修改） | `EMOJI_CHECK_WARNING`，S2 和个人设置的故事共用 | 4 |
| `e2e/stories/identity/a11-api-tokens.spec.ts`（修改） | A11 的页面版本 | 4 |
| `web:core/components/settings/profile/content/pages/security.tsx`、`e2e/stories/identity/a7-change-password.spec.ts`（修改） | 安全页的 PAT 列表；A7 的页面版本 | 5 |
| `web:core/services/timezone.service.ts`、`web:core/hooks/use-timezone.tsx`（修改）；`web/packages/types/src/timezone.ts`（删除） | 时区列表 | 6 |
| `web:core/components/{appearance/theme-switcher,profile/start-of-week-preference}.tsx`、`settings/profile/content/pages/preferences/language-and-timezone-list.tsx`（修改） | preferences 的错误提示 | 6 |
| `e2e/stories/identity/a8-update-me.spec.ts`、`a9-preferences.spec.ts`（修改） | A8、A9 的页面版本 | 6 |
| `web:core/components/settings/profile/content/pages/general/form.tsx`、`account/deactivate-account-modal.tsx`、`instance/maintenance-message.tsx`（修改）；`web/packages/i18n/src/locales/{en,zh-CN}/common.json`（修改） | general 页的保存、停用账户、维护页 | 7 |
| `e2e/stories/identity/a12-deactivate.spec.ts`（修改） | A12 的页面版本 | 7 |
| `web:core/store/{user/settings,user/permissions,issue/profile/filter,issue/profile/issue}.store.ts`、`web:core/store/issue/*/filter.store.ts`、`issue/helpers/issue-filter-helper.store.ts`、`web/packages/constants/src/settings/profile.ts`、`web:core/components/{auth-screens/not-authorized-view,profile/sidebar,settings/profile/heading}.tsx`、`web:core/layouts/auth-layout/{project,workspace}-wrapper.tsx`（修改） | 死成员和死 prop | 8 |
| `web:app/(all)/invitations/page.tsx`、`web:core/components/{core/image-picker-popover,issues/issue-detail/reactions/issue,onboarding/steps/profile/root,onboarding/switch-account-modal,project/form,project/settings/member-columns,workspace/settings/member-columns}.tsx`、`web/packages/utils/src/auth.ts`、`web/packages/utils/package.json`（修改） | P4、P5 改到的文件的 oxlint 警告 | 9 |
| 44 个文件（Task 10 列出，其中 `web:core/components/dropdowns/cycle/cycle-options.tsx`、`web:core/components/modules/modal.tsx` 另有两处警告）（修改） | `no-unneeded-ternary` 清零（54 条） | 10 |
| `docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M2-auth/handoffs/{M1-P3-trim-platform,M1-closeout}.md`（修改） | 文档同步、交接 | 11 |

---

### Task 1: 下拉框在按钮旁展开（7.7 的根因），打开状态跟随 Headless UI

**Files:**
- Modify: `web/packages/ui/src/dropdowns/custom-select.tsx`、`custom-search-select.tsx`、`combo-box.tsx`
- Modify: `web/apps/web/core/components/dropdowns/date.tsx`
- Modify（过渡）: `web/apps/web/package.json`（oxlint 上限 551 → 550）、`web/packages/ui/package.json`（25 → 20）

**Interfaces:**
- 根因（spec 2.3）：Headless UI 2.2 的 `Combobox.Options` 渲染时克隆自己唯一的子元素，换上自己的 ref（`Frozen` 组件）。Plane 把 react-popper 的 `ref={setPopperElement}`、`style`、`attributes` 放在这个子元素上，这个 ref 从来没有被调用，popper 没有元素可定位，列表停在 `position: absolute; left: 0; top: 0`，即页面左上角。与界面语言无关（M1 收尾交接写的是 zh-CN，英文下一样）。
- 修复：三个组件都把 `ref={setPopperElement}`、`style={styles.popper}`、`{...attributes.popper}` 和原来子元素的 `className` 放在 `Combobox.Options` 本身（`as="ul"`；Headless UI 转发这个 ref），子元素只留内容。`popperElement` 的类型改为 `HTMLElement`。
- `CustomSearchSelect` 另设 `modal={false}`：Headless UI 2.2 的模态列表把"不包含输入框、按钮、列表的祖先的兄弟"都设为 inert，搜索框在列表里，于是列表的选项被设为 inert，点不了（Plane 的结构也是这样；spec 2.3）。`CustomSelect`、`DateDropdown` 没有输入框，保持默认。
- 打开状态：三个组件都有自己的 `isOpen`，Headless UI 另有一份。按钮上的 Escape 由 Headless UI 处理并停止冒泡，只关了它自己的一份：列表消失（或者对 `static` 的列表，不消失），组件的 `isOpen` 仍为真，下一次点击只把它设为假，要点第二次才打开。三个组件把 `onClose` 交给 `Combobox`（`ComboDropDown` 加上 `onClose` prop，透传给 `Combobox`），Headless UI 自己关闭时组件的状态跟着关。
- 使用者：preferences 的主题、语言、时区、每周第一天（Task 6 的 A8、A9 页面版本核对）；api-tokens 的有效期和自定义日期（Task 4 的 A11 页面版本核对）。M3–M6 的其他下拉框有同样的写法，不在本 Task（spec 第 7 节交接）。

**Tests:** 没有 DOM 测试环境，下拉框由 Task 4、6 的页面版本（`expectListBesideButton`：列表在按钮正下方或正上方 8 像素之内，左边缘或右边缘对齐 2 像素之内；Escape 关闭之后一次点击就再打开）和浏览器核对 C2、C3 覆盖。本 Task 只跑现有的检查。

- [ ] **Step 1: `CustomSelect`**

`web/packages/ui/src/dropdowns/custom-select.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/ui/src/dropdowns/custom-select.tsx
+++ b/web/packages/ui/src/dropdowns/custom-select.tsx
@@ -41,7 +41,7 @@
   } = props;
   // states
   const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
-  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
+  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
   const [isOpen, setIsOpen] = useState(false);
   // refs
   const dropdownRef = useRef<HTMLDivElement | null>(null);
@@ -66,6 +66,7 @@
 
   return (
     <DropdownContext.Provider value={closeDropdown}>
+      {/* oxlint-disable-next-line jsx_a11y/no-static-element-interactions -- the keys come from the button inside, which Headless UI makes the control; this div only sees them bubble */}
       <Combobox
         as="div"
         ref={dropdownRef}
@@ -77,6 +78,9 @@
         }}
         className={cn("relative flex-shrink-0 text-left", className)}
         onKeyDown={handleKeyDown}
+        // Headless UI also closes on its own (Escape on the button, a click outside). isOpen follows: else it stays
+        // true with no list shown, and the next click only sets it to false, so the list takes a second click.
+        onClose={closeDropdown}
         disabled={disabled}
       >
         <>
@@ -118,26 +122,29 @@
         </>
         {isOpen &&
           createPortal(
-            <Combobox.Options as="ul" data-prevent-outside-click>
+            // Popper places the list itself, whose ref Headless UI forwards. A ref on the list's one child would
+            // never be set: Combobox.Options clones that child with a ref of its own (Headless UI 2.2), so the
+            // list would stay where the page begins, at its top left.
+            <Combobox.Options
+              as="ul"
+              data-prevent-outside-click
+              ref={setPopperElement}
+              className={cn(
+                "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap focus:outline-none",
+                optionsClassName
+              )}
+              style={styles.popper}
+              {...attributes.popper}
+            >
               <div
-                className={cn(
-                  "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap focus:outline-none",
-                  optionsClassName
-                )}
-                ref={setPopperElement}
-                style={styles.popper}
-                {...attributes.popper}
+                className={cn("space-y-1 overflow-y-scroll", {
+                  "max-h-60": maxHeight === "lg",
+                  "max-h-48": maxHeight === "md",
+                  "max-h-36": maxHeight === "rg",
+                  "max-h-28": maxHeight === "sm",
+                })}
               >
-                <div
-                  className={cn("space-y-1 overflow-y-scroll", {
-                    "max-h-60": maxHeight === "lg",
-                    "max-h-48": maxHeight === "md",
-                    "max-h-36": maxHeight === "rg",
-                    "max-h-28": maxHeight === "sm",
-                  })}
-                >
-                  {children}
-                </div>
+                {children}
               </div>
             </Combobox.Options>,
             document.body
@@ -161,6 +168,7 @@
   }, [closeDropdown]);
 
   return (
+    // oxlint-disable-next-line jsx_a11y/click-events-have-key-events -- Headless UI makes this an option and picks it with the keys; the click only closes the list when the option was already picked
     <Combobox.Option
       as="li"
       value={value}
```

- [ ] **Step 2: `CustomSearchSelect`**

`web/packages/ui/src/dropdowns/custom-search-select.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox } from "@headlessui/react";
import { ChevronDownOutline, InfoOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import React, { useRef, useState } from "react";
import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
import { useOutsideClickDetector } from "@nerve/hooks";
// local imports
import { Tooltip } from "@nerve/propel/tooltip";
import { useDropdownKeyDown } from "../hooks/use-dropdown-key-down";
import { cn } from "../utils";
import type { ICustomSearchSelectProps } from "./helper";

export function CustomSearchSelect(props: ICustomSearchSelectProps) {
  const {
    customButtonClassName = "",
    buttonClassName = "",
    className = "",
    chevronClassName = "",
    customButton,
    placement,
    disabled = false,
    footerOption,
    input = false,
    label,
    maxHeight = "md",
    multiple = false,
    noChevron = false,
    onChange,
    options,
    onOpen,
    onClose,
    optionsClassName = "",
    value,
    tabIndex,
    noResultsMessage = "No matches found",
    defaultOpen = false,
  } = props;
  const [query, setQuery] = useState("");

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const [isOpen, setIsOpen] = useState(defaultOpen);
  // refs
  const dropdownRef = useRef<HTMLDivElement | null>(null);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
  });

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const comboboxProps: any = {
    value,
    onChange,
    disabled,
  };

  if (multiple) comboboxProps.multiple = true;

  const openDropdown = () => {
    setIsOpen(true);
    if (referenceElement) referenceElement.focus();
    if (onOpen) onOpen();
  };

  const closeDropdown = () => {
    setIsOpen(false);
    onClose?.();
  };

  const handleKeyDown = useDropdownKeyDown(openDropdown, closeDropdown, isOpen);
  useOutsideClickDetector(dropdownRef, closeDropdown);

  const toggleDropdown = () => {
    if (isOpen) closeDropdown();
    else openDropdown();
  };

  return (
    // oxlint-disable-next-line jsx_a11y/no-static-element-interactions -- the keys come from the button inside, which Headless UI makes the control; this div only sees them bubble
    <Combobox
      as="div"
      ref={dropdownRef}
      tabIndex={tabIndex}
      className={cn("relative flex-shrink-0 text-left", className)}
      onKeyDown={handleKeyDown}
      // Headless UI also closes on its own (Escape on the button, a click outside): isOpen follows, as in
      // custom-select.tsx; else the list, static, stays open.
      onClose={closeDropdown}
      {...comboboxProps}
    >
      {({ open }: { open: boolean }) => {
        if (open && onOpen) onOpen();

        return (
          <>
            {customButton ? (
              <Combobox.Button as={React.Fragment}>
                <button
                  ref={setReferenceElement}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between gap-1 text-11",
                    {
                      "cursor-not-allowed text-secondary": disabled,
                      "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                    },
                    customButtonClassName
                  )}
                  onClick={toggleDropdown}
                >
                  {customButton}
                </button>
              </Combobox.Button>
            ) : (
              <Combobox.Button as={React.Fragment}>
                <button
                  ref={setReferenceElement}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between gap-1 rounded-sm border-[0.5px] border-strong",
                    {
                      "px-3 py-2 text-13": input,
                      "px-2 py-1 text-11": !input,
                      "cursor-not-allowed text-secondary": disabled,
                      "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                    },
                    buttonClassName
                  )}
                  onClick={toggleDropdown}
                >
                  {label}
                  {!noChevron && !disabled && (
                    <ChevronDownOutline className={cn("h-3 w-3 flex-shrink-0", chevronClassName)} aria-hidden="true" />
                  )}
                </button>
              </Combobox.Button>
            )}
            {isOpen &&
              createPortal(
                // Popper places the list itself, whose ref Headless UI forwards: a ref on the list's one child would
                // never be set (custom-select.tsx), and the list would stay at the top left of the page. Not modal:
                // a modal list makes inert whatever holds none of the input, the button and the list, going up
                // from each (Headless UI 2.2); the search input is in the list, so that would be the options.
                <Combobox.Options
                  as="ul"
                  data-prevent-outside-click
                  static
                  modal={false}
                  ref={setPopperElement}
                  className={cn(
                    "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                    optionsClassName
                  )}
                  style={styles.popper}
                  {...attributes.popper}
                >
                  <div className="mx-2 flex items-center gap-1.5 rounded-sm border border-subtle px-2">
                    <SearchOutline className="h-3.5 w-3.5 text-placeholder" />
                    <Combobox.Input
                      className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      placeholder="Search"
                      displayValue={(assigned: any) => assigned?.name}
                    />
                  </div>
                  <div
                    className={cn("vertical-scrollbar mt-2 scrollbar-xs space-y-1 overflow-y-scroll px-2", {
                      "max-h-96": maxHeight === "2xl",
                      "max-h-80": maxHeight === "xl",
                      "max-h-60": maxHeight === "lg",
                      "max-h-48": maxHeight === "md",
                      "max-h-36": maxHeight === "rg",
                      "max-h-28": maxHeight === "sm",
                    })}
                  >
                    {filteredOptions ? (
                      filteredOptions.length > 0 ? (
                        filteredOptions.map((option) => (
                          // oxlint-disable-next-line jsx_a11y/click-events-have-key-events -- Headless UI makes this an option and picks it with the keys; the click only closes the list
                          <Combobox.Option
                            as="li"
                            key={option.value}
                            value={option.value}
                            className={({ active }) =>
                              cn(
                                "flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none",
                                {
                                  "bg-layer-transparent-hover": active,
                                  "cursor-not-allowed text-placeholder opacity-60": option.disabled,
                                }
                              )
                            }
                            onClick={() => {
                              if (!multiple) closeDropdown();
                            }}
                            disabled={option.disabled}
                          >
                            {({ selected }) => (
                              <>
                                <span className="flex-grow truncate">{option.content}</span>
                                {selected && <TickOutline className="h-3.5 w-3.5 flex-shrink-0" />}
                                {option.tooltip && (
                                  <>
                                    {typeof option.tooltip === "string" ? (
                                      <Tooltip tooltipContent={option.tooltip}>
                                        <InfoOutline className="h-3.5 w-3.5 flex-shrink-0 cursor-pointer text-secondary" />
                                      </Tooltip>
                                    ) : (
                                      option.tooltip
                                    )}
                                  </>
                                )}
                              </>
                            )}
                          </Combobox.Option>
                        ))
                      ) : (
                        <p className="px-1.5 py-1 text-placeholder italic">{noResultsMessage}</p>
                      )
                    ) : (
                      <p className="px-1.5 py-1 text-placeholder italic">Loading...</p>
                    )}
                  </div>
                  {footerOption}
                </Combobox.Options>,
                document.body
              )}
          </>
        );
      }}
    </Combobox>
  );
}
```

- [ ] **Step 3: `ComboDropDown` 的 `onClose`；`DateDropdown`**

`web/packages/ui/src/dropdowns/combo-box.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/ui/src/dropdowns/combo-box.tsx
+++ b/web/packages/ui/src/dropdowns/combo-box.tsx
@@ -17,6 +17,8 @@
   onChange?: (value: any) => void;
   disabled?: boolean | undefined;
   onKeyDown?: KeyboardEventHandler<HTMLDivElement> | undefined;
+  /** Called when Headless UI closes the combobox on its own (Escape on the button, a click outside). */
+  onClose?: () => void;
   multiple?: boolean;
   renderByDefault?: boolean;
   button: ReactNode;
```

`web/apps/web/core/components/dropdowns/date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/dropdowns/date.tsx
+++ b/web/apps/web/core/components/dropdowns/date.tsx
@@ -78,7 +78,7 @@
   const startOfWeek = data?.start_of_the_week;
   // popper-js refs
   const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
-  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
+  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
   // popper-js init
   const { styles, attributes } = usePopper(referenceElement, popperElement, {
     placement: placement ?? "bottom-start",
@@ -163,6 +163,7 @@
   );
 
   return (
+    // oxlint-disable-next-line jsx_a11y/no-static-element-interactions -- the keys come from the button inside, which Headless UI makes the control; this div only sees them bubble
     <ComboDropDown
       as="div"
       ref={dropdownRef}
@@ -173,38 +174,44 @@
           if (!isOpen) handleKeyDown(e);
         } else handleKeyDown(e);
       }}
+      // Headless UI also closes on its own (Escape on the button, a click outside): isOpen follows, as in
+      // @nerve/ui's custom-select.tsx; else the calendar stays open.
+      onClose={handleClose}
       button={comboButton}
       disabled={disabled}
       renderByDefault={renderByDefault}
     >
       {isOpen &&
         createPortal(
-          <Combobox.Options as="ul" data-prevent-outside-click static>
-            <div
-              className={cn(
-                "z-30 my-1 overflow-hidden rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200",
-                optionsClassName
-              )}
-              ref={setPopperElement}
-              style={styles.popper}
-              {...attributes.popper}
-            >
-              <Calendar
-                className="rounded-md border border-subtle p-3"
-                captionLayout="dropdown"
-                selected={getDate(value)}
-                defaultMonth={getDate(value)}
-                onSelect={(date: Date | undefined) => {
-                  dropdownOnChange(date ?? null);
-                }}
-                showOutsideDays
-                initialFocus
-                disabled={disabledDays}
-                mode="single"
-                fixedWeeks
-                weekStartsOn={startOfWeek}
-              />
-            </div>
+          // Popper places the list itself, whose ref Headless UI forwards: a ref on the list's one child would never
+          // be set (@nerve/ui's custom-select.tsx), and the calendar would stay at the top left of the page.
+          <Combobox.Options
+            as="ul"
+            data-prevent-outside-click
+            static
+            ref={setPopperElement}
+            className={cn(
+              "z-30 my-1 overflow-hidden rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200",
+              optionsClassName
+            )}
+            style={styles.popper}
+            {...attributes.popper}
+          >
+            <Calendar
+              className="rounded-md border border-subtle p-3"
+              captionLayout="dropdown"
+              selected={getDate(value)}
+              defaultMonth={getDate(value)}
+              onSelect={(date: Date | undefined) => {
+                dropdownOnChange(date ?? null);
+              }}
+              showOutsideDays
+              initialFocus
+              disabled={disabledDays}
+              mode="single"
+              fixedWeeks
+              weekStartsOn={startOfWeek}
+            />
           </Combobox.Options>,
           document.body
         )}
```

- [ ] **Step 4: oxlint 上限**

web 少 1 条（`date.tsx`），ui 少 5 条（`custom-select.tsx` 2 条、`custom-search-select.tsx` 3 条）：`onClose && onClose()` 改为 `onClose?.()`；外层 `div` 的 `jsx_a11y/no-static-element-interactions` 和选项的 `jsx_a11y/click-events-have-key-events` 用带理由的禁用注释（按键由 Headless UI 的按钮和选项处理，外层只看到冒泡上来的按键；选项的点击只负责关闭列表）。

`web/apps/web/package.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 551",
+    "check:lint": "node ../../../tools/lint-cap.mjs 550",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

`web/packages/ui/package.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/ui/package.json
+++ b/web/packages/ui/package.json
@@ -16,7 +16,7 @@
   "scripts": {
     "build": "tsdown",
     "dev": "tsdown --watch --no-clean",
-    "check:lint": "node ../../../tools/lint-cap.mjs 25",
+    "check:lint": "node ../../../tools/lint-cap.mjs 20",
     "check:types": "tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt ."
```

- [ ] **Step 5: 检查**

Run: `grep -rn "&lt;\|&gt;\|&amp;" web/packages/ui/src/dropdowns web/apps/web/core/components/dropdowns/date.tsx`
Expected: 没有输出。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（web 的 oxlint 上限 550，ui 20）。

- [ ] **Step 6: 提交**

```bash
git add web/packages/ui web/apps/web/core/components/dropdowns/date.tsx web/apps/web/package.json
```
```bash
git commit -m "fix(M2/P5): dropdowns open beside their button, and reopen after Escape

Headless UI 2.2's Combobox.Options clones its one child with a ref of
its own, so react-popper's ref on that child was never set and the list
stayed at the page's top left, in any language. CustomSelect,
CustomSearchSelect and DateDropdown put popper's ref, style and
attributes on Combobox.Options itself, which forwards the ref.
CustomSearchSelect's list is not modal: a modal one made its own
options inert, the search box being inside it. Each dropdown's open
state now follows the closes Headless UI makes itself (Escape on the
button): before, the next click only undid the stale state.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 检查都通过；三个组件里没有放在子元素上的 popper ref。列表的位置由 Task 4、6 的故事证明。

---

### Task 2: PAT 的 service 和 store

**Files:**
- Create: `web/apps/web/core/services/api-token.service.ts`、`web/apps/web/core/store/user/api-token.store.ts`、`api-token.store.test.ts`
- Modify: `web/apps/web/core/store/user/index.ts`、`web/apps/web/core/store/workspace/index.ts`
- Delete: `web/apps/web/core/store/workspace/api-token.store.ts`
- Modify: `web/apps/web/core/lib/auth/fake-nerve.ts`、`web/apps/web/core/lib/store-context.test.ts`
- Modify（过渡）: `web/apps/web/package.json`（oxlint 上限 550 → 548）

**Interfaces:**
- Produces（spec 2.4，M2 设计 7.5）：
  - `class ApiTokenService { constructor(api: ApiClient); list(cursor?): Promise<ApiTokenPage>; create(data: ApiTokenCreate): Promise<ApiTokenCreated>; revoke(tokenId): Promise<void> }`：`GET /api/v0/me/api-tokens?limit=100[&cursor=…]`、`POST /api/v0/me/api-tokens`、`DELETE /api/v0/api-tokens/{token_id}`，都经 `unwrap`；`api` 是 `RootStore` 传下来的这一代会话的客户端，没有模块级的实例。
  - `interface IApiTokenStore { tokens: ApiToken[] | undefined; fetchTokens(): Promise<ApiToken[] | undefined>; createToken(data): Promise<ApiTokenCreated>; revokeToken(tokenId): Promise<void> }`，`class ApiTokenStore(api: ApiClient)`：
    - `tokens`：新的在前，只有列表的六个字段，没有令牌原文；取到之前是 `undefined`（不是空数组，页面据此显示加载中，而不是"没有令牌"）；
    - `fetchTokens`：逐页取（每页 100 个，按上一页的 `next_cursor`），全部取完才一次写入；任何一页失败就抛出，`tokens` 不变；`SessionChangedError`（会话在加载中换了）不算失败，返回 `undefined`，不写入；
    - `createToken`：返回带令牌原文的 `ApiTokenCreated`，只交给调用方；列表加上一个显式构造的 `ApiToken`（六个字段），列表还没取过时不动它（否则只有新令牌的列表会被当成完整列表显示）；
    - `revokeToken`：nerve 答 204 之后才从列表去掉；拒绝时抛出，列表不变。
  - `UserStore.apiTokens: IApiTokenStore`，构造时 `new ApiTokenStore(api)`；工作区 store 的旧 `apiToken` 和它的文件删除（Plane 的 PAT 挂在工作区下，Nerve 的属于账户）。
  - `FakeNerve` 的每个请求多记一项 `query: Record<string, string>`（地址的查询参数），测试据此核对 `limit`、`cursor`。
- 使用者：Task 3 的 api-tokens 页、Task 5 的安全页（经 `useApiTokens()`）。

**Tests:**
- `api-token.store.test.ts`，5 个，对假 nerve：逐页取、每页带上一页的 cursor、全部取完才显示（中途 `tokens` 仍是 `undefined`）；任何一页失败时抛出 `ApiError`、不显示部分列表；创建时原文只在返回值里，列表的新项与列表的六个字段逐字段相等（`toEqual` 多一个字段也失败）；列表没取过时创建不生成列表；撤销在 nerve 答复之前不改列表，拒绝时保留，成功时只去掉那一个（三个中间的一个）。
- `store-context.test.ts` 加 1 个：X 的列表加载到一半时另一个标签页登录 Y，本标签页跟随：X 的第一页在跟随之后才答复，第二页不再请求（不以 X 也不以 Y 的身份），加载安静地放弃、返回 `undefined`；Y 的会话有自己的 PAT store（与 X 的不是同一个对象），它先用 Y 的刷新令牌续期，再以 Y 的访问令牌取第一页，查询参数只有 `limit=100`（没有 X 的 cursor）；X 的页面上发起的撤销不发出，以 `SessionChangedError` 失败，Y 的列表不变。

- [ ] **Step 1: 假 nerve 记下查询参数**

`web/apps/web/core/lib/auth/fake-nerve.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/lib/auth/fake-nerve.ts
+++ b/web/apps/web/core/lib/auth/fake-nerve.ts
@@ -13,6 +13,8 @@
 export type Call = {
   method: string;
   path: string;
+  /** The parameters of the query string. */
+  query: Record<string, string>;
   authorization: string | null;
   body: unknown;
   /** When the request arrived, on the fake clock. */
@@ -33,13 +35,15 @@
   /** The fetch of the client under test: reads the body as fetch does, parks the request until answered or failed. */
   fetch = async (request: Request): Promise<Response> => {
     const text = await request.text();
+    const url = new URL(request.url);
     return new Promise<Response>((resolve, reject) => {
       const onAbort = () => reject(new DOMException("The operation was aborted.", "AbortError"));
       if (request.signal.aborted) return onAbort();
       request.signal.addEventListener("abort", onAbort);
       this.calls.push({
         method: request.method,
-        path: new URL(request.url).pathname,
+        path: url.pathname,
+        query: Object.fromEntries(url.searchParams),
         authorization: request.headers.get("Authorization"),
         body: text === "" ? undefined : (JSON.parse(text) as unknown),
         at: Date.now(),
```

- [ ] **Step 2: service 和 store**

`web/apps/web/core/services/api-token.service.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ApiTokenCreate, ApiTokenCreated, ApiTokenPage } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The account's personal access tokens (M2 design 7.5). */
export class ApiTokenService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** A page of the account's tokens, newest first, as large as nerve gives: the one after cursor, or the first. */
  async list(cursor?: string): Promise<ApiTokenPage> {
    return unwrap(await this.api.GET("/api/v0/me/api-tokens", { params: { query: { limit: 100, cursor } } }));
  }

  /** Creates a token; the answer carries the token itself, this once. */
  async create(data: ApiTokenCreate): Promise<ApiTokenCreated> {
    return unwrap(await this.api.POST("/api/v0/me/api-tokens", { body: data }));
  }

  async revoke(tokenId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/api-tokens/{token_id}", { params: { path: { token_id: tokenId } } }));
  }
}
```

`web/apps/web/core/store/user/api-token.store.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, ApiToken, ApiTokenCreate, ApiTokenCreated } from "@nerve/api-client";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
// services
import { ApiTokenService } from "@/services/api-token.service";

export interface IApiTokenStore {
  /** The account's tokens, newest first, as lists show them: never the token itself. Undefined until fetched. */
  tokens: ApiToken[] | undefined;
  fetchTokens: () => Promise<ApiToken[] | undefined>;
  createToken: (data: ApiTokenCreate) => Promise<ApiTokenCreated>;
  revokeToken: (tokenId: string) => Promise<void>;
}

/**
 * The personal access tokens of the account of a session (M2 design 7.5). Its service sends with the client of
 * that session, which the RootStore of the session hands down: a new session has a store of its own.
 */
export class ApiTokenStore implements IApiTokenStore {
  tokens: ApiToken[] | undefined = undefined;
  private readonly service: ApiTokenService;

  constructor(api: ApiClient) {
    makeObservable(this, {
      tokens: observable.ref,
      fetchTokens: action,
      createToken: action,
      revokeToken: action,
    });
    this.service = new ApiTokenService(api);
  }

  /**
   * @description fetches every page of the account's tokens and shows them all at once. A change of session
   * while they load is no failure: the new session's store fetches its own (store-context.tsx), and this one
   * gives undefined.
   * @returns {Promise<ApiToken[] | undefined>}
   */
  fetchTokens = async (): Promise<ApiToken[] | undefined> => {
    try {
      const tokens = await this.listFrom(undefined);
      runInAction(() => {
        this.tokens = tokens;
      });
      return tokens;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /** The page after cursor (undefined: the first) and every page after it, one after another. */
  private async listFrom(cursor: string | undefined): Promise<ApiToken[]> {
    const page = await this.service.list(cursor);
    return page.next_cursor === null ? page.data : [...page.data, ...(await this.listFrom(page.next_cursor))];
  }

  /**
   * @description creates a token and returns it with the token itself, for the one time it is shown; the list
   * gets the token as lists show it, without the token itself. Fails when nerve refuses.
   * @returns {Promise<ApiTokenCreated>}
   */
  createToken = async (data: ApiTokenCreate): Promise<ApiTokenCreated> => {
    const created = await this.service.create(data);
    const listed: ApiToken = {
      id: created.id,
      label: created.label,
      description: created.description,
      expired_at: created.expired_at,
      last_used: created.last_used,
      created_at: created.created_at,
    };
    runInAction(() => {
      if (this.tokens) this.tokens = [listed, ...this.tokens];
    });
    return created;
  };

  /**
   * @description revokes a token, which then leaves the list; fails when nerve refuses
   * @returns {Promise<void>}
   */
  revokeToken = async (tokenId: string): Promise<void> => {
    await this.service.revoke(tokenId);
    runInAction(() => {
      if (this.tokens) this.tokens = this.tokens.filter((token) => token.id !== tokenId);
    });
  };
}
```

- [ ] **Step 3: 挂到 `UserStore`，删掉工作区的旧 store**

`web/apps/web/core/store/user/index.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/user/index.ts
+++ b/web/apps/web/core/store/user/index.ts
@@ -30,6 +30,8 @@
 import type { IUserProfileStore } from "@/store/user/profile.store";
 import { ProfileStore } from "@/store/user/profile.store";
 // local imports
+import type { IApiTokenStore } from "./api-token.store";
+import { ApiTokenStore } from "./api-token.store";
 import type { IUserSettingsStore } from "./settings.store";
 import { UserSettingsStore } from "./settings.store";
 
@@ -40,6 +42,7 @@
   userProfile: IUserProfileStore;
   userSettings: IUserSettingsStore;
   permission: IUserPermissionStore;
+  apiTokens: IApiTokenStore;
   // actions
   fetchCurrentUser: () => Promise<User | undefined>;
   updateCurrentUser: (data: UserUpdate) => Promise<User>;
@@ -60,6 +63,7 @@
   userProfile: IUserProfileStore;
   userSettings: IUserSettingsStore;
   permission: IUserPermissionStore;
+  apiTokens: IApiTokenStore;
   // service
   userService: UserService;
   authService: AuthService;
@@ -72,6 +76,7 @@
     this.userProfile = new ProfileStore(store, api);
     this.userSettings = new UserSettingsStore(api);
     this.permission = new UserPermissionStore(store, api);
+    this.apiTokens = new ApiTokenStore(api);
     // service
     this.userService = new UserService(api);
     this.authService = new AuthService();
@@ -82,6 +87,7 @@
       userProfile: observable,
       userSettings: observable,
       permission: observable,
+      apiTokens: observable,
       // actions
       fetchCurrentUser: action,
       updateCurrentUser: action,
```

`web/apps/web/core/store/workspace/index.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/workspace/index.ts
+++ b/web/apps/web/core/store/workspace/index.ts
@@ -14,8 +14,6 @@
 // store
 import type { RootStore } from "@/store/root.store";
 // sub-stores
-import type { IApiTokenStore } from "./api-token.store";
-import { ApiTokenStore } from "./api-token.store";
 import type { IWebhookStore } from "./webhook.store";
 import { WebhookStore } from "./webhook.store";
 
@@ -45,7 +43,6 @@
   ) => Promise<void>;
   // sub-stores
   webhook: IWebhookStore;
-  apiToken: IApiTokenStore;
 }
 
 export class WorkspaceRootStore implements IWorkspaceRootStore {
@@ -60,7 +57,6 @@
   user;
   // sub-stores
   webhook: IWebhookStore;
-  apiToken: IApiTokenStore;
 
   constructor(_rootStore: RootStore) {
     makeObservable(this, {
@@ -90,7 +86,6 @@
     this.user = _rootStore.user;
     // sub-stores
     this.webhook = new WebhookStore(_rootStore);
-    this.apiToken = new ApiTokenStore(_rootStore);
   }
 
   /**
```

```bash
git rm web/apps/web/core/store/workspace/api-token.store.ts
```

Run: `grep -rn "workspace/api-token.store\|\.apiToken\b\|IApiTokenStore" web/apps/web/core web/apps/web/app`
Expected: 只有 `core/store/user/index.ts` 和 `core/store/user/api-token.store.ts` 里的 `IApiTokenStore`（`.apiToken\b` 没有输出）。

- [ ] **Step 4: 测试**

`web/apps/web/core/store/user/api-token.store.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiToken, ApiTokenCreated } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { ApiTokenStore } from "@/store/user/api-token.store";

// The personal access tokens of an account (M2 design 7.5), against a fake nerve. That the store sends as the
// session of its RootStore, and that a new session starts with a store of its own, is store-context.test.ts.

const LIST = "/api/v0/me/api-tokens";

/** A token as lists show it, the n-th. */
function listed(n: number): ApiToken {
  return {
    id: `00000000-0000-4000-8000-00000000000${n}`,
    label: `token ${n}`,
    description: "",
    expired_at: null,
    last_used: null,
    created_at: `2026-09-2${n}T00:00:00Z`,
  };
}

/** The token itself of a new token: nrv_pat_ and 43 characters (M2 design 3.4). */
const SECRET = "nrv_pat_Q2hhcmxlcyBCYWJiYWdlIGFuZCBBZGEgTG92ZWxhY2U";

function created(n: number): ApiTokenCreated {
  return { ...listed(n), token: SECRET };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ApiTokenStore", () => {
  it("fetches the pages one after another, each after the cursor of the last, and shows the whole list at once", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());

    await until(() => nerve.calls.length === 1, "the first page");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: LIST });
    expect(nerve.calls[0]?.query).toEqual({ limit: "100" });
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2)], next_cursor: "c-1" }));
    await until(() => nerve.calls.length === 2, "the second page");
    // Nothing shows until the list is whole.
    expect(store.tokens).toBeUndefined();
    expect(nerve.calls[1]).toMatchObject({ method: "GET", path: LIST });
    expect(nerve.calls[1]?.query).toEqual({ limit: "100", cursor: "c-1" });
    nerve.calls[1]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    expect(fetched.value).toEqual([listed(3), listed(2), listed(1)]);
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
    expect(nerve.calls).toHaveLength(2);
  });

  it("fails when a page fails, and shows no part of the list", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());

    await until(() => nerve.calls.length === 1, "the first page");
    nerve.calls[0]?.answer(json(200, { data: [listed(2)], next_cursor: "c-1" }));
    await until(() => nerve.calls.length === 2, "the second page");
    nerve.calls[1]?.answer(problem(503, "server_busy"));
    await until(() => fetched.settled, "the failure");

    expect(fetched.error).toBeInstanceOf(ApiError);
    expect(store.tokens).toBeUndefined();
  });

  it("gives the token itself to the one who created it, and lists the new token without it", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the list");
    nerve.calls[0]?.answer(json(200, { data: [listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    const body = { label: "token 2", description: "ci", expired_at: "2026-10-04T00:00:00Z" };
    const creating = track(store.createToken(body));
    await until(() => nerve.calls.length === 2, "the creation");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: LIST, body });
    nerve.calls[1]?.answer(json(201, created(2)));
    await until(() => creating.settled, "the new token");

    expect(creating.value).toEqual(created(2));
    // The list has the new token first, with the fields lists show and no other (toEqual fails on a field
    // more): never the token itself.
    expect(store.tokens).toEqual([listed(2), listed(1)]);
    expect(JSON.stringify(store.tokens)).not.toContain(SECRET.slice("nrv_pat_".length));
  });

  it("leaves a list it has not fetched to the fetch, when it creates a token", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const creating = track(store.createToken({ label: "token 1" }));
    await until(() => nerve.calls.length === 1, "the creation");
    nerve.calls[0]?.answer(json(201, created(1)));
    await until(() => creating.settled, "the new token");

    expect(creating.value).toEqual(created(1));
    // A list of the new token alone would show as the whole list.
    expect(store.tokens).toBeUndefined();
  });

  it("takes a token off the list once nerve revoked it, and keeps it when nerve refuses", async () => {
    const nerve = new FakeNerve();
    const store = new ApiTokenStore(nerve.client());
    const fetched = track(store.fetchTokens());
    await until(() => nerve.calls.length === 1, "the list");
    nerve.calls[0]?.answer(json(200, { data: [listed(3), listed(2), listed(1)], next_cursor: null }));
    await until(() => fetched.settled, "the list");

    const refused = track(store.revokeToken(listed(2).id));
    await until(() => nerve.calls.length === 2, "the first revocation");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: `/api/v0/api-tokens/${listed(2).id}` });
    nerve.calls[1]?.answer(problem(404, "identity.api_token_not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);

    // The token in the middle: only it leaves.
    const revoked = track(store.revokeToken(listed(2).id));
    await until(() => nerve.calls.length === 3, "the second revocation");
    // Until nerve answers, the token is still the account's.
    expect(store.tokens).toEqual([listed(3), listed(2), listed(1)]);
    nerve.calls[2]?.answer(noContent());
    await until(() => revoked.settled, "the revocation");
    expect(revoked.error).toBeUndefined();
    expect(store.tokens).toEqual([listed(3), listed(1)]);
  });
});
```

`web/apps/web/core/lib/store-context.test.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/lib/store-context.test.ts
+++ b/web/apps/web/core/lib/store-context.test.ts
@@ -21,6 +21,7 @@
 const LOGOUT = "/api/v0/auth/logout";
 const ME = "/api/v0/me";
 const PROFILE = "/api/v0/me/profile";
+const TOKENS = "/api/v0/me/api-tokens";
 const X = "0123456789abcdef0123456789abcdef";
 /** The sessions of other accounts, which other tabs sign in to. */
 const Y = "fedcba9876543210fedcba9876543210";
@@ -29,6 +30,15 @@
 const Z = "abababababababababababababababab";
 /** The record a sign-in as loginId writes. */
 const record = (loginId: string) => JSON.stringify({ refresh_token: `rt-${loginId}`, login_id: loginId });
+/** A personal access token of the account of loginId, as lists show it. */
+const tokenOf = (loginId: string) => ({
+  id: `${loginId.slice(0, 8)}-0000-4000-8000-000000000000`,
+  label: `token of ${loginId}`,
+  description: "",
+  expired_at: null,
+  last_used: null,
+  created_at: "2026-09-27T00:00:00Z",
+});
 
 /** The page's theme and language, each time something set them. */
 const page = vi.hoisted(() => ({ themes: [] as string[], languages: [] as string[] }));
@@ -226,6 +236,50 @@
     expect(nerve.to(PROFILE)).toEqual([]);
     expect(tm.state).toEqual({ status: "signed-in", loginId: Y });
     expect(storage.data.get(AUTH_KEY)).toBe(record(Y));
+  });
+
+  it("gives each session the tokens of its own account: a load cut by the switch stops, the new list loads as the new account", async () => {
+    const { nerve, context, SessionChangedError, signedIn, follow } = await load();
+    await signedIn();
+    const x = context.rootStore;
+    // The api-tokens page of X's session loads X's list, a page at a time.
+    const listedX = track(x.user.apiTokens.fetchTokens());
+    await until(() => nerve.calls.length === 1, "X's first page");
+    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: TOKENS, authorization: "Bearer at-1" });
+    // Another tab signs in as Y while the page is out; this tab follows.
+    await follow(Y);
+    nerve.calls[0]?.answer(json(200, { data: [tokenOf(X)], next_cursor: "c-1" }));
+    await until(() => listedX.settled || nerve.calls.length > 1, "X's list, or another request");
+
+    // X's next page is not asked for, as X or as Y; the load gives up quietly and shows nothing.
+    expect(listedX).toEqual({ settled: true, value: undefined });
+    expect(x.user.apiTokens.tokens).toBeUndefined();
+    await vi.advanceTimersByTimeAsync(1_000);
+    expect(nerve.calls).toHaveLength(1);
+
+    // Y's session has a store of its own, which shows nothing until it has Y's list, loaded as Y.
+    const y = context.rootStore;
+    expect(y.user.apiTokens).not.toBe(x.user.apiTokens);
+    expect(y.user.apiTokens.tokens).toBeUndefined();
+    const listedY = track(y.user.apiTokens.fetchTokens());
+    await until(() => nerve.calls.length === 2, "Y's refresh");
+    expect(nerve.calls[1]).toMatchObject({ path: REFRESH, body: { refresh_token: `rt-${Y}` } });
+    nerve.calls[1]?.answer(json(200, nerve.tokens()));
+    await until(() => nerve.calls.length === 3, "Y's list");
+    expect(nerve.calls[2]).toMatchObject({ method: "GET", path: TOKENS, authorization: "Bearer at-2" });
+    // Y's first page: no cursor of X's list.
+    expect(nerve.calls[2]?.query).toEqual({ limit: "100" });
+    nerve.calls[2]?.answer(json(200, { data: [tokenOf(Y)], next_cursor: null }));
+    await until(() => listedY.settled, "Y's list");
+    expect(y.user.apiTokens.tokens).toEqual([tokenOf(Y)]);
+
+    // A revocation from X's page, which the tab no longer shows, is not sent: not with Y's token.
+    const revoked = track(x.user.apiTokens.revokeToken(tokenOf(X).id));
+    await until(() => revoked.settled || nerve.calls.length > 3, "the answer, or a request");
+    expect(revoked.error).toBeInstanceOf(SessionChangedError);
+    await vi.advanceTimersByTimeAsync(1_000);
+    expect(nerve.calls).toHaveLength(3);
+    expect(y.user.apiTokens.tokens).toEqual([tokenOf(Y)]);
   });
 
   it("gives the code that reads the stores outside the components the RootStore of the session now", async () => {
```

- [ ] **Step 5: oxlint 上限**

工作区的旧 store 带走 2 条警告。

`web/apps/web/package.json`（对 Task 1 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 550",
+    "check:lint": "node ../../../tools/lint-cap.mjs 548",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 6: 检查**

Run: `pnpm -C web/apps/web exec vitest run core/store/user core/lib/store-context.test.ts`
Expected: `api-token.store.test.ts` 5 个、`store-context.test.ts` 5 个测试通过。

Run: `pnpm -C web/apps/web exec vitest run --testTimeout=1000`
Expected: 通过（P4 的 CI 教训：测试不在自己的时限内冷加载 store 的模块图）。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（web 的 oxlint 上限 548）。

- [ ] **Step 7: 提交**

```bash
git add web/apps/web/core/services/api-token.service.ts web/apps/web/core/store web/apps/web/core/lib web/apps/web/package.json
```
```bash
git commit -m "feat(M2/P5): the personal access token store, on the session's client

ApiTokenService takes the client of its RootStore's session by
constructor; ApiTokenStore hangs off UserStore, so a new session has a
store of its own and a load cut by the switch ends in
SessionChangedError, never sent as the new account. The list is fetched
page by page and shown once whole; a created token's secret goes only
to the caller, never into the list. The workspace's old token store
goes: tokens belong to the account.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 6 个新测试通过；它们用假 nerve 的手动答复和假时钟决定先后，没有 sleep。

---

### Task 3: api-tokens 页；`@nerve/services` 只剩地址和文件工具

**Files:**
- Create: `web/apps/web/core/components/api-token/token-list.tsx`、`modal/expiry.ts`、`modal/expiry.test.ts`、`web/apps/web/core/hooks/store/user/user-api-tokens.ts`
- Modify: `web/apps/web/core/components/api-token/modal/create-token-modal.tsx`、`modal/form.tsx`、`modal/generated-token-details.tsx`、`delete-token-modal.tsx`、`token-list-item.tsx`
- Modify: `web/apps/web/core/components/settings/profile/content/pages/api-tokens.tsx`、`web/apps/web/core/components/ui/loader/settings/api-token.tsx`、`web/apps/web/core/hooks/store/user/index.ts`
- Modify: `web/packages/i18n/src/locales/{en,zh-CN}/workspace-settings.json`；（过渡）`settings.json`
- Delete: `web/packages/services/src/api.service.ts`、`web/packages/services/src/developer/`、`web/packages/types/src/api_token.ts`
- Modify: `web/packages/services/package.json`、`src/index.ts`、`pnpm-lock.yaml`、`web/packages/constants/src/fetch-keys.ts`；（过渡）`web/packages/types/src/index.ts`、`tools/keywords.json`
- Modify（过渡）: `web/apps/web/package.json`（oxlint 上限 548 → 543）

**Interfaces:**
- Produces（spec 2.5，M2 设计 7.7）：
  - `useApiTokens(): IApiTokenStore`（`hooks/store/user/`），读 `StoreContext` 的 `user.apiTokens`。
  - `ApiTokenList({ empty })`：api-tokens 页和安全页（Task 5）共用。`useSWR(signed-in ? ["API_TOKENS", loginId] : null, fetchTokens, { revalidateOnFocus: false, shouldRetryOnError: false })`；`tokens === undefined` 时：取数失败（并且不在重试中）显示"令牌没能加载"和重试按钮，否则显示加载中；空列表显示 `empty`；否则逐个 `ApiTokenListItem`。
  - `ApiTokenListItem`：名称、说明、有效或已过期（按 `expired_at` 与现在比较）、到期时间、创建时间、最后使用时间（"从未使用"），撤销按钮（`aria-label` 为"撤销令牌"）。
  - 有效期：`EXPIRY_PERIODS`（1 周、1 个月、3 个月、1 年）、`TExpiryChoice`（加 `"custom"`）、`expiryDate(choice, now, picked)`：按日历加上期间（date-fns 的 `add`），自定义日期取所选那天、现在的时刻；未选日期时 `undefined`。表单另有"永不过期"开关（`expired_at: null`）。
  - 创建弹窗：`createToken` 的返回值只放在弹窗组件的状态里；显示一次，同时下载 CSV（Plane 的行为）；关闭之后 350 毫秒清掉；每次打开都先回到表单（在渲染时重置，重开得再快也不会有一帧显示上一个令牌）。表单的字段错误按 `fieldErrorKeys` 显示在 `label`（名称）、`expired_at`（有效期）下，其余按 `needsErrorBanner` 提示。
  - 撤销弹窗：`revokeToken`，失败按 `errorMessageKey` 提示。
  - `@nerve/services`：删除 Plane 的 axios 基类 `api.service.ts`（最后一个 `withCredentials: true`）和 `developer/`（旧的令牌服务，`/api/users/api-tokens/`）；`package.json` 去掉 `axios`；只剩地址工具和上传文件的元数据工具。`@nerve/types` 删除 `api_token.ts`（`IApiToken`）；`@nerve/constants` 删除 `API_TOKENS_LIST`。它们没有别的使用者（按 Global Constraints 查过）。
  - 关键词规则（`tools/keywords.json`，顶层 `phase` 改为 `M2/P5`）：`with-credentials`（`withCredentials:\s*true`，`web/` 下）、`plane-api-token-urls`（`/api/users/api-tokens`）；`services-files` 的白名单去掉 `api.service.ts` 和 `developer/`。
  - 文案：`account_settings.api_tokens` 下新增 `load_failed`、`active`、`expired`、`expires_at`、`expired_at`、`created_at`、`last_used`、`never_used`、`revoke`、`copy`、`expiry.*`；删除只有旧页面用的 `workspace_settings.settings.api_tokens.title`、`.delete.error.message`。
- 使用者：Task 4 的 A11；Task 5 的安全页用 `ApiTokenList`。

**Tests:** `expiry.test.ts`，2 个，本地时间，任何时区都成立：从 10 月 27 日起按日历加 1 周、1 个月、3 个月、1 年（10 月 31 天、11 月 30 天，按天数加 30 天会错）；自定义日期取所选那天、现在的时刻，没选时 `undefined`。页面由 Task 4 的 A11 页面版本覆盖。

- [ ] **Step 1: store 的 hook**

`web/apps/web/core/hooks/store/user/user-api-tokens.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
// types
import type { IApiTokenStore } from "@/store/user/api-token.store";

/** The personal access tokens of the account of the tab's session now. */
export const useApiTokens = (): IApiTokenStore => useContext(StoreContext).user.apiTokens;
```

`web/apps/web/core/hooks/store/user/index.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export * from "./user-user";
export * from "./user-user-profile";
export * from "./user-user-settings";
export * from "./user-permissions";
export * from "./user-api-tokens";
```

- [ ] **Step 2: 列表**

`web/apps/web/core/components/api-token/token-list.tsx`（新文件）：

```tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import useSWR from "swr";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// components
import { ApiTokenListItem } from "@/components/api-token/token-list-item";
import { APITokenSettingsLoader } from "@/components/ui/loader/settings/api-token";
// hooks
import { useApiTokens } from "@/hooks/store/user";
// lib
import { useSession } from "@/lib/auth/use-session";

type Props = {
  /** What shows when the account has no token. */
  empty: ReactNode;
};

/**
 * The personal access tokens of the account of the tab's session, each of which can be revoked here: the
 * api-tokens page and the security page (M2 design 7.7) show it.
 */
export const ApiTokenList = observer(function ApiTokenList(props: Props) {
  const { empty } = props;
  // store hooks
  const session = useSession();
  const { tokens, fetchTokens } = useApiTokens();
  const { t } = useTranslation();

  // The list is fetched for each session, into the store of its own RootStore (M2 design 7.1): another account
  // or another sign-in is another key, and nothing of the list before shows.
  const { error, isValidating, mutate } = useSWR(
    session.status === "signed-in" ? ["API_TOKENS", session.loginId] : null,
    () => fetchTokens(),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );

  if (tokens === undefined)
    return error && !isValidating ? (
      <div className="flex items-center gap-3 py-3 text-13 text-secondary">
        <span role="alert">{t("account_settings.api_tokens.load_failed")}</span>
        <Button variant="secondary" size="sm" onClick={() => void mutate()}>
          {t("common.retry")}
        </Button>
      </div>
    ) : (
      <APITokenSettingsLoader />
    );
  if (tokens.length === 0) return <>{empty}</>;
  return (
    <div>
      {tokens.map((token) => (
        <ApiTokenListItem key={token.id} token={token} />
      ))}
    </div>
  );
});
```

`web/apps/web/core/components/api-token/token-list-item.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { CloseCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { ApiToken } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// components
import { DeleteApiTokenModal } from "@/components/api-token/delete-token-modal";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  token: ApiToken;
};

/** A token as lists show it: never the token itself, but when it was created and last used (M2 design 8.5). */
export function ApiTokenListItem(props: Props) {
  const { token } = props;
  // states
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const expired = token.expired_at !== null && new Date(token.expired_at).getTime() <= Date.now();
  const at = (time: string) => ({ date: renderFormattedDate(time), time: renderFormattedTime(time) });

  return (
    <>
      <DeleteApiTokenModal isOpen={deleteModalOpen} onClose={() => setDeleteModalOpen(false)} tokenId={token.id} />
      <div className="group relative flex flex-col justify-center border-b border-subtle py-3">
        <Tooltip label={t("account_settings.api_tokens.revoke")} disabled={isMobile}>
          <button
            type="button"
            onClick={() => setDeleteModalOpen(true)}
            aria-label={t("account_settings.api_tokens.revoke")}
            className="absolute right-4 hidden place-items-center group-hover:grid"
          >
            <CloseCircleOutline className="h-4 w-4 text-danger-primary" />
          </button>
        </Tooltip>
        <div className="flex w-4/5 items-center">
          <h5 className="truncate text-13 font-medium">{token.label}</h5>
          <span
            className={`${
              expired ? "bg-layer-1 text-placeholder" : "bg-success-subtle text-success-primary"
            } ml-2 flex h-4 max-h-fit items-center rounded-xs px-2 text-11 font-medium`}
          >
            {expired ? t("account_settings.api_tokens.expired") : t("account_settings.api_tokens.active")}
          </span>
        </div>
        <div className="mt-1 flex w-full flex-col justify-center">
          {token.description.trim() !== "" && (
            <p className="mb-1 max-w-[70%] text-13 break-words">{token.description}</p>
          )}
          <p className="text-11 leading-6 text-placeholder">
            {token.expired_at === null
              ? t("workspace_settings.settings.api_tokens.never_expires")
              : t(`account_settings.api_tokens.${expired ? "expired_at" : "expires_at"}`, at(token.expired_at))}
          </p>
          <p className="text-11 leading-6 text-placeholder">
            {t("account_settings.api_tokens.created_at", { date: renderFormattedDate(token.created_at) })}
            {" · "}
            {token.last_used === null
              ? t("account_settings.api_tokens.never_used")
              : t("account_settings.api_tokens.last_used", at(token.last_used))}
          </p>
        </div>
      </div>
    </>
  );
}
```

`web/apps/web/core/components/ui/loader/settings/api-token.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { range } from "lodash-es";

/** The rows of a list of tokens, while it loads. */
export function APITokenSettingsLoader() {
  return (
    <div className="divide-y-[0.5px] divide-subtle-1">
      {range(2).map((i) => (
        <div key={i} className="flex flex-col gap-2 py-3">
          <div className="flex items-center gap-2">
            <span className="h-5 w-28 rounded-sm bg-layer-1" />
            <span className="h-5 w-16 rounded-sm bg-layer-1" />
          </div>
          <span className="h-5 w-36 rounded-sm bg-layer-1" />
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 3: 有效期**

`web/apps/web/core/components/api-token/modal/expiry.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { add } from "date-fns";

/** The periods a new token may last (M2 design 7.7); it may also end on a day picked, or never. */
export const EXPIRY_PERIODS = [
  { key: "1_week", period: { weeks: 1 } },
  { key: "1_month", period: { months: 1 } },
  { key: "3_months", period: { months: 3 } },
  { key: "1_year", period: { years: 1 } },
] as const;

/** A period's key, or "custom" for a day picked. */
export type TExpiryChoice = (typeof EXPIRY_PERIODS)[number]["key"] | "custom";

/**
 * When a token created at now expires: the period chosen after now, or the day picked (a date at midnight, as
 * the date picker gives it) at now's time of day. Undefined for "custom" before a day is picked.
 */
export function expiryDate(choice: TExpiryChoice, now: Date, picked: Date | null): Date | undefined {
  if (choice !== "custom") return add(now, EXPIRY_PERIODS.find((p) => p.key === choice)?.period ?? {});
  if (!picked) return undefined;
  return add(picked, { hours: now.getHours(), minutes: now.getMinutes(), seconds: now.getSeconds() });
}
```

`web/apps/web/core/components/api-token/modal/expiry.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { expiryDate } from "./expiry";

// When a new token expires (M2 design 7.7). The dates are the local time's, as the page's, so the test holds in
// any time zone, across a change of summer time too.

/** 27 October 2026, 10:20:30, local time: a month of 31 days, then one of 30. */
const now = new Date(2026, 9, 27, 10, 20, 30);

describe("expiryDate", () => {
  it("adds the period chosen to now, by the calendar", () => {
    expect(expiryDate("1_week", now, null)).toEqual(new Date(2026, 10, 3, 10, 20, 30));
    expect(expiryDate("1_month", now, null)).toEqual(new Date(2026, 10, 27, 10, 20, 30));
    expect(expiryDate("3_months", now, null)).toEqual(new Date(2027, 0, 27, 10, 20, 30));
    expect(expiryDate("1_year", now, null)).toEqual(new Date(2027, 9, 27, 10, 20, 30));
  });

  it("ends a token on the day picked at now's time of day, and not before a day is picked", () => {
    expect(expiryDate("custom", now, new Date(2026, 10, 3))).toEqual(new Date(2026, 10, 3, 10, 20, 30));
    expect(expiryDate("custom", now, null)).toBeUndefined();
  });
});
```

- [ ] **Step 4: 创建和撤销的弹窗**

`web/apps/web/core/components/api-token/modal/form.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { add } from "date-fns";
import { Controller, useForm } from "react-hook-form";
import { CalendarOutline } from "@makeplane/propel/icons";
// nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import type { ApiTokenCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// ui
import { Switch } from "@makeplane/propel/components/switch";
import { CustomSelect } from "@nerve/ui";
import { cn, renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// components
import { DateDropdown } from "@/components/dropdowns/date";
// helpers
import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/helpers/authentication.helper";
// local imports
import type { TExpiryChoice } from "./expiry";
import { EXPIRY_PERIODS, expiryDate } from "./expiry";

type Props = {
  handleClose: () => void;
  neverExpires: boolean;
  toggleNeverExpires: () => void;
  /** Creates the token; fails as nerve refuses. */
  onSubmit: (data: ApiTokenCreate) => Promise<void>;
};

type TFormValues = {
  label: string;
  description: string;
  expiry: TExpiryChoice | null;
};

const defaultValues: TFormValues = {
  label: "",
  description: "",
  expiry: null,
};

/** The fields of ApiTokenCreate whose errors show under the form's fields. */
const FIELDS = ["label", "expired_at"] as const;

export function CreateApiTokenForm(props: Props) {
  const { handleClose, neverExpires, toggleNeverExpires, onSubmit } = props;
  // states
  const [customDate, setCustomDate] = useState<Date | null>(null);
  // form
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    setError,
    watch,
  } = useForm<TFormValues>({ defaultValues });
  // hooks
  const { t } = useTranslation();

  const handleFormSubmit = async (data: TFormValues) => {
    const expiresAt = data.expiry === null ? undefined : expiryDate(data.expiry, new Date(), customDate);
    if (!neverExpires && expiresAt === undefined) {
      setError("expiry", { type: "manual", message: t("account_settings.api_tokens.expiry.required") });
      return;
    }
    try {
      await onSubmit({
        label: data.label,
        description: data.description,
        expired_at: neverExpires ? null : expiresAt?.toISOString(),
      });
    } catch (error) {
      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
      const fields = fieldErrorKeys(error);
      if (fields.label !== undefined) setError("label", { type: "manual", message: t(fields.label) });
      if (fields.expired_at !== undefined) setError("expiry", { type: "manual", message: t(fields.expired_at) });
      if (needsErrorBanner(error, FIELDS))
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    }
  };

  const tomorrow = add(new Date(), { days: 1 });
  const expiry = watch("expiry");
  const expiresAt = expiry === null ? undefined : expiryDate(expiry, new Date(), customDate);

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)}>
      <div className="space-y-5 p-5">
        <h3 className="text-18 font-medium text-secondary">
          {t("workspace_settings.settings.api_tokens.create_token")}
        </h3>
        <div className="space-y-3">
          <div className="space-y-1">
            <Controller
              control={control}
              name="label"
              rules={{
                required: t("title_is_required"),
                maxLength: {
                  value: 255,
                  message: t("title_should_be_less_than_255_characters"),
                },
                validate: (val) => val.trim() !== "" || t("title_is_required"),
              }}
              render={({ field: { value, onChange } }) => (
                <Field name="input" invalid={Boolean(errors.label)}>
                  <InputGroup size="2xl">
                    <Input
                      size="2xl"
                      type="text"
                      value={value}
                      onChange={onChange}
                      placeholder={t("title")}
                      aria-label={t("title")}
                    />
                  </InputGroup>
                </Field>
              )}
            />
            {errors.label && <span className="text-11 text-danger-primary">{errors.label.message}</span>}
          </div>
          <Controller
            control={control}
            name="description"
            render={({ field: { value, onChange } }) => (
              <Field name="description" invalid={Boolean(errors.description)}>
                <TextAreaGroup resize="none">
                  <TextArea
                    size="lg"
                    surface="field"
                    autoResize
                    maxRows={8}
                    value={value}
                    onChange={onChange}
                    placeholder={t("description")}
                    aria-label={t("description")}
                  />
                </TextAreaGroup>
              </Field>
            )}
          />
          <div className="space-y-1">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Controller
                  control={control}
                  name="expiry"
                  render={({ field: { onChange, value } }) => (
                    <CustomSelect
                      customButton={
                        <div
                          className={cn(
                            "flex h-7 items-center gap-2 rounded-sm border-[0.5px] border-strong px-2 py-0.5",
                            {
                              "text-placeholder": neverExpires,
                            }
                          )}
                        >
                          <CalendarOutline className="h-3 w-3" />
                          {t(`account_settings.api_tokens.expiry.${value ?? "set"}`)}
                        </div>
                      }
                      value={value}
                      onChange={onChange}
                      disabled={neverExpires}
                    >
                      {EXPIRY_PERIODS.map((option) => (
                        <CustomSelect.Option key={option.key} value={option.key}>
                          {t(`account_settings.api_tokens.expiry.${option.key}`)}
                        </CustomSelect.Option>
                      ))}
                      <CustomSelect.Option value="custom">
                        {t("account_settings.api_tokens.expiry.custom")}
                      </CustomSelect.Option>
                    </CustomSelect>
                  )}
                />
                {expiry === "custom" && (
                  <div className="h-7">
                    <DateDropdown
                      value={customDate}
                      onChange={(date) => setCustomDate(date)}
                      minDate={tomorrow}
                      icon={<CalendarOutline className="h-3 w-3" />}
                      buttonVariant="border-with-text"
                      placeholder={t("account_settings.api_tokens.expiry.set_date")}
                      disabled={neverExpires}
                    />
                  </div>
                )}
              </div>
              {!neverExpires && expiresAt && (
                <span className="text-11 text-placeholder">
                  {t("account_settings.api_tokens.expires_at", {
                    date: renderFormattedDate(expiresAt),
                    time: renderFormattedTime(expiresAt),
                  })}
                </span>
              )}
            </div>
            {!neverExpires && errors.expiry && (
              <span className="text-11 text-danger-primary">{errors.expiry.message}</span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <label className="flex cursor-pointer items-center gap-1.5">
          <Switch size="sm" checked={neverExpires} onCheckedChange={toggleNeverExpires} />
          <span className="text-11">{t("workspace_settings.settings.api_tokens.never_expires")}</span>
        </label>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={handleClose}>
            {t("cancel")}
          </Button>
          <Button variant="primary" type="submit" loading={isSubmitting}>
            {isSubmitting
              ? t("workspace_settings.settings.api_tokens.generating")
              : t("workspace_settings.settings.api_tokens.generate_token")}
          </Button>
        </div>
      </div>
    </form>
  );
}
```

`web/apps/web/core/components/api-token/modal/create-token-modal.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import type { ApiTokenCreate, ApiTokenCreated } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { renderFormattedDate, csvDownload } from "@nerve/utils";
// hooks
import { useApiTokens } from "@/hooks/store/user";
// local imports
import { CreateApiTokenForm } from "./form";
import { GeneratedTokenDetails } from "./generated-token-details";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function CreateApiTokenModal(props: Props) {
  const { isOpen, onClose } = props;
  // store hooks
  const { createToken } = useApiTokens();
  // states
  const [neverExpires, setNeverExpires] = useState<boolean>(false);
  // The new token with the token itself: only here, while the modal shows it (M2 design 7.7).
  const [generatedToken, setGeneratedToken] = useState<ApiTokenCreated | null>(null);
  const [wasOpen, setWasOpen] = useState(isOpen);
  const { t } = useTranslation();

  // Each opening starts with the form: set while rendering, so that no frame of an opening shows the token of
  // the last, not even one opened before the reset after closing has run.
  if (isOpen !== wasOpen) {
    setWasOpen(isOpen);
    if (isOpen) {
      setNeverExpires(false);
      setGeneratedToken(null);
    }
  }

  const handleClose = () => {
    onClose();

    setTimeout(() => {
      setNeverExpires(false);
      setGeneratedToken(null);
    }, 350);
  };

  const downloadSecretKey = (data: ApiTokenCreated) => {
    const csvData = {
      Title: data.label,
      Description: data.description,
      Expiry: data.expired_at
        ? (renderFormattedDate(data.expired_at)?.replace(",", " ") ?? "")
        : t("workspace_settings.settings.api_tokens.never_expires"),
      "Secret key": data.token,
    };

    csvDownload(csvData, `secret-key-${Date.now()}`);
  };

  const handleCreateToken = async (data: ApiTokenCreate) => {
    const created = await createToken(data);
    setGeneratedToken(created);
    downloadSecretKey(created);
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={() => {}} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      {generatedToken ? (
        <GeneratedTokenDetails handleClose={handleClose} tokenDetails={generatedToken} />
      ) : (
        <CreateApiTokenForm
          handleClose={handleClose}
          neverExpires={neverExpires}
          toggleNeverExpires={() => setNeverExpires((prevData) => !prevData)}
          onSubmit={handleCreateToken}
        />
      )}
    </ModalCore>
  );
}
```

`web/apps/web/core/components/api-token/modal/generated-token-details.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ApiTokenCreated } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { CopyOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// ui
import { renderFormattedDate, renderFormattedTime, copyTextToClipboard } from "@nerve/utils";
// types
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  handleClose: () => void;
  /** The new token, with the token itself, which shows here this once. */
  tokenDetails: ApiTokenCreated;
};

export function GeneratedTokenDetails(props: Props) {
  const { handleClose, tokenDetails } = props;
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const copyApiToken = (token: string) => {
    void copyTextToClipboard(token).then(() =>
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: `${t("success")}!`,
        message: t("workspace_settings.token_copied"),
      })
    );
  };

  return (
    <div className="w-full p-5">
      <div className="w-full space-y-3 text-wrap">
        <h3 className="text-16 leading-6 font-medium text-primary">{t("workspace_settings.key_created")}</h3>
        <p className="text-13 text-placeholder">{t("workspace_settings.copy_key")}</p>
      </div>
      <button
        type="button"
        onClick={() => copyApiToken(tokenDetails.token)}
        className="mt-4 flex w-full items-center justify-between truncate rounded-md border-[0.5px] border-subtle px-3 py-2 text-13 font-medium outline-none"
      >
        <span className="truncate pr-2">{tokenDetails.token}</span>
        <Tooltip label={t("account_settings.api_tokens.copy")} disabled={isMobile}>
          <CopyOutline className="h-4 w-4 flex-shrink-0 text-placeholder" />
        </Tooltip>
      </button>
      <div className="mt-6 flex items-center justify-between">
        <p className="text-11 text-placeholder">
          {tokenDetails.expired_at
            ? t("account_settings.api_tokens.expires_at", {
                date: renderFormattedDate(tokenDetails.expired_at),
                time: renderFormattedTime(tokenDetails.expired_at),
              })
            : t("workspace_settings.settings.api_tokens.never_expires")}
        </p>
        <Button variant="secondary" onClick={handleClose}>
          {t("close")}
        </Button>
      </div>
    </div>
  );
}
```

`web/apps/web/core/components/api-token/delete-token-modal.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// ui
import { AlertModalCore } from "@nerve/ui";
// helpers
import { errorMessageKey } from "@/helpers/authentication.helper";
// hooks
import { useApiTokens } from "@/hooks/store/user";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  tokenId: string;
};

export function DeleteApiTokenModal(props: Props) {
  const { isOpen, onClose, tokenId } = props;
  // store hooks
  const { revokeToken } = useApiTokens();
  // states
  const [deleteLoading, setDeleteLoading] = useState<boolean>(false);
  const { t } = useTranslation();

  const handleClose = () => {
    onClose();
    setDeleteLoading(false);
  };

  const handleDeletion = async () => {
    setDeleteLoading(true);
    try {
      await revokeToken(tokenId);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("workspace_settings.settings.api_tokens.delete.success.title"),
        message: t("workspace_settings.settings.api_tokens.delete.success.message"),
      });
      handleClose();
    } catch (error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("workspace_settings.settings.api_tokens.delete.error.title"),
        message: t(errorMessageKey(error)),
      });
      setDeleteLoading(false);
    }
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={handleDeletion}
      isSubmitting={deleteLoading}
      isOpen={isOpen}
      title={t("workspace_settings.settings.api_tokens.delete.title")}
      content={<>{t("workspace_settings.settings.api_tokens.delete.description")} </>}
    />
  );
}
```

- [ ] **Step 5: 页面**

`web/apps/web/core/components/settings/profile/content/pages/api-tokens.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EmptyStateCompact } from "@nerve/propel/empty-state";
// components
import { CreateApiTokenModal } from "@/components/api-token/modal/create-token-modal";
import { ApiTokenList } from "@/components/api-token/token-list";
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";

export function APITokensProfileSettings() {
  // states
  const [isCreateTokenModalOpen, setIsCreateTokenModalOpen] = useState(false);
  // translation
  const { t } = useTranslation();

  return (
    <div className="size-full">
      <CreateApiTokenModal isOpen={isCreateTokenModalOpen} onClose={() => setIsCreateTokenModalOpen(false)} />
      <ProfileSettingsHeading
        title={t("account_settings.api_tokens.title")}
        description={t("account_settings.api_tokens.description")}
        control={
          <Button variant="primary" size="lg" onClick={() => setIsCreateTokenModalOpen(true)}>
            {t("workspace_settings.settings.api_tokens.add_token")}
          </Button>
        }
      />
      <div className="mt-7">
        <ApiTokenList
          empty={
            <EmptyStateCompact
              assetKey="token"
              assetClassName="size-20"
              title={t("settings_empty_state.tokens.title")}
              description={t("settings_empty_state.tokens.description")}
              actions={[
                {
                  label: t("settings_empty_state.tokens.cta_primary"),
                  onClick: () => {
                    setIsCreateTokenModalOpen(true);
                  },
                },
              ]}
              align="start"
              rootClassName="py-20"
            />
          }
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 6: 文案**

`web/packages/i18n/src/locales/en/settings.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/en/settings.json
+++ b/web/packages/i18n/src/locales/en/settings.json
@@ -6,7 +6,27 @@
     },
     "api_tokens": {
       "title": "Personal Access Tokens",
-      "description": "Generate secure API tokens to integrate your data with external systems and applications."
+      "description": "Generate secure API tokens to integrate your data with external systems and applications.",
+      "load_failed": "The tokens could not be loaded.",
+      "active": "Active",
+      "expired": "Expired",
+      "expires_at": "Expires {date} at {time}",
+      "expired_at": "Expired {date} at {time}",
+      "created_at": "Created {date}",
+      "last_used": "Last used {date} at {time}",
+      "never_used": "Never used",
+      "revoke": "Revoke token",
+      "copy": "Copy secret key",
+      "expiry": {
+        "1_week": "1 week",
+        "1_month": "1 month",
+        "3_months": "3 months",
+        "1_year": "1 year",
+        "custom": "Custom date",
+        "set": "Set expiration date",
+        "set_date": "Set date",
+        "required": "Choose when the token expires, or that it never does."
+      }
     }
   },
   "profile": {
```

`web/packages/i18n/src/locales/zh-CN/settings.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/zh-CN/settings.json
+++ b/web/packages/i18n/src/locales/zh-CN/settings.json
@@ -6,7 +6,27 @@
     },
     "api_tokens": {
       "title": "个人访问令牌",
-      "description": "生成安全的 API 令牌，将您的数据与外部系统和应用程序集成。"
+      "description": "生成安全的 API 令牌，将您的数据与外部系统和应用程序集成。",
+      "load_failed": "无法加载令牌。",
+      "active": "有效",
+      "expired": "已过期",
+      "expires_at": "{date} {time} 过期",
+      "expired_at": "已于 {date} {time} 过期",
+      "created_at": "创建于 {date}",
+      "last_used": "最后使用于 {date} {time}",
+      "never_used": "从未使用",
+      "revoke": "撤销令牌",
+      "copy": "复制密钥",
+      "expiry": {
+        "1_week": "1 周",
+        "1_month": "1 个月",
+        "3_months": "3 个月",
+        "1_year": "1 年",
+        "custom": "自定义日期",
+        "set": "设置过期时间",
+        "set_date": "选择日期",
+        "required": "请选择令牌何时过期，或设为永不过期。"
+      }
     }
   },
   "profile": {
```

`web/packages/i18n/src/locales/en/workspace-settings.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/en/workspace-settings.json
+++ b/web/packages/i18n/src/locales/en/workspace-settings.json
@@ -92,7 +92,6 @@
         }
       },
       "api_tokens": {
-        "title": "Access Tokens",
         "add_token": "Add access token",
         "create_token": "Create token",
         "never_expires": "Never expires",
@@ -106,8 +105,7 @@
             "message": "The token has been successfully deleted"
           },
           "error": {
-            "title": "Error!",
-            "message": "The token could not be deleted"
+            "title": "Error!"
           }
         }
       }
```

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/zh-CN/workspace-settings.json
+++ b/web/packages/i18n/src/locales/zh-CN/workspace-settings.json
@@ -92,7 +92,6 @@
         }
       },
       "api_tokens": {
-        "title": "API 令牌",
         "add_token": "添加访问令牌",
         "create_token": "创建令牌",
         "never_expires": "永不过期",
@@ -106,8 +105,7 @@
             "message": "API 令牌已成功删除"
           },
           "error": {
-            "title": "错误！",
-            "message": "无法删除 API 令牌"
+            "title": "错误！"
           }
         }
       }
```

- [ ] **Step 7: `@nerve/services`、`@nerve/types`、`@nerve/constants` 的旧代码**

```bash
git rm web/packages/services/src/api.service.ts web/packages/services/src/developer/index.ts web/packages/services/src/developer/api-token.service.ts web/packages/types/src/api_token.ts
```

`web/packages/services/src/index.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export * from "./file";
export * from "./helpers";
```

`web/packages/services/package.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/services/package.json
+++ b/web/packages/services/package.json
@@ -23,7 +23,6 @@
   "dependencies": {
     "@nerve/constants": "workspace:*",
     "@nerve/types": "workspace:*",
-    "axios": "catalog:",
     "file-type": "catalog:"
   },
   "devDependencies": {
```

Run: `pnpm install`
Expected: `pnpm-lock.yaml` 的变化只有下面这一处：

`pnpm-lock.yaml`（对 `cec4ec9` 的差异）：

```diff
--- a/pnpm-lock.yaml
+++ b/pnpm-lock.yaml
@@ -849,9 +849,6 @@
       '@nerve/types':
         specifier: workspace:*
         version: link:../types
-      axios:
-        specifier: 1.18.1
-        version: 1.18.1(supports-color@10.2.2)
       file-type:
         specifier: 'catalog:'
         version: 21.3.3(supports-color@10.2.2)
```

`web/packages/types/src/index.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/types/src/index.ts
+++ b/web/packages/types/src/index.ts
@@ -4,7 +4,6 @@
  * See the LICENSE file for details.
  */
 
-export * from "./api_token";
 export * from "./calendar";
 export * from "./charts";
 export * from "./common";
```

`web/packages/constants/src/fetch-keys.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/constants/src/fetch-keys.ts
+++ b/web/packages/constants/src/fetch-keys.ts
@@ -93,9 +93,6 @@
 // Issues
 export const ISSUE_DETAILS = (issueId: string) => `ISSUE_DETAILS_${issueId.toUpperCase()}`;
 
-// api-tokens
-export const API_TOKENS_LIST = `API_TOKENS_LIST`;
-
 // project level keys
 export const PROJECT_DETAILS = (_workspaceSlug: string, projectId: string) =>
   `PROJECT_DETAILS_${projectId.toUpperCase()}`;
```

Run: `grep -rn "APITokenService\|IApiToken\b\|API_TOKENS_LIST\|withCredentials: true\|/api/users/api-tokens" web --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build --exclude-dir=.turbo`
Expected: 没有输出。

- [ ] **Step 8: 关键词规则**

`tools/keywords.json`（对 `cec4ec9` 的差异）：

```diff
--- a/tools/keywords.json
+++ b/tools/keywords.json
@@ -1,5 +1,5 @@
 {
-  "phase": "M2/P4",
+  "phase": "M2/P5",
   "rules": [
     {
       "id": "deploy-files",
@@ -150,17 +150,18 @@
     {
       "id": "services-files",
       "phase": "M1/P1",
-      "why": "packages/services 只留 web 用到的令牌服务、地址工具和上传元数据工具，整包在 M5 删除；新的接口调用用生成的客户端",
+      "why": "packages/services 只留 web 用到的地址工具和上传元数据工具，整包在 M5 删除；新的接口调用用生成的客户端。M2/P5 删掉了 Plane 的 axios 基类 api.service.ts 和令牌服务 developer/（PAT 改由 web 的 store 经生成的客户端调用）",
       "path": {
-        "source": "^web/packages/services/(?!(?:package\\.json|tsconfig\\.json|tsdown\\.config\\.ts|src/(?:index|api\\.service)\\.ts|src/helpers/(?:index|url|url\\.test)\\.ts|src/developer/(?:index|api-token\\.service)\\.ts|src/file/(?:index|helper)\\.ts)$)",
+        "source": "^web/packages/services/(?!(?:package\\.json|tsconfig\\.json|tsdown\\.config\\.ts|src/index\\.ts|src/helpers/(?:index|url|url\\.test)\\.ts|src/file/(?:index|helper)\\.ts)$)",
         "flags": ""
       },
       "samples": {
-        "hit": ["web/packages/services/src/issue/sites-issue.service.ts", "web/packages/services/src/live.service.ts"],
-        "miss": [
-          "web/packages/services/src/helpers/url.test.ts",
+        "hit": [
+          "web/packages/services/src/issue/sites-issue.service.ts",
+          "web/packages/services/src/api.service.ts",
           "web/packages/services/src/developer/api-token.service.ts"
-        ]
+        ],
+        "miss": ["web/packages/services/src/helpers/url.test.ts", "web/packages/services/src/file/helper.ts"]
       }
     },
     {
@@ -1919,8 +1920,59 @@
             "web/apps/web/helpers/authentication.helper.ts"
           ],
           "miss": ["server/internal/shared/error.go", "docs/v0/M2-auth/M2-design.md"]
+        }
+      }
+    },
+    {
+      "id": "with-credentials",
+      "phase": "M2/P5",
+      "why": "Plane 靠 Cookie 认证，axios 实例都带 withCredentials: true；Nerve 的令牌在 Authorization 头里，不发 Cookie（M2 设计 7.2）。最后一处是 @nerve/services 的 api.service.ts，随 PAT 的重写删除（M2 设计 7.9）。上传到对象存储的 withCredentials: false 属于 M5，不命中",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "withCredentials:\\s*true",
+        "flags": ""
+      },
+      "samples": {
+        "hit": ["      withCredentials: true,", "    this.axiosInstance = create({ withCredentials:true });"],
+        "miss": ["      withCredentials: false,"],
+        "files": {
+          "hit": ["web/packages/services/src/api.service.ts", "web/apps/web/core/services/api.service.ts"],
+          "miss": ["docs/v0/M2-auth/M2-design.md", "server/internal/platform/httpapi/cors.go"]
         }
       }
+    },
+    {
+      "id": "plane-api-token-urls",
+      "phase": "M2/P5",
+      "why": "Plane 的个人访问令牌地址 /api/users/api-tokens/；web 的 PAT store 改调 /api/v0/me/api-tokens 和 /api/v0/api-tokens/{token_id}（M2 设计 7.5、7.9）",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "/api/users/api-tokens",
+        "flags": ""
+      },
+      "samples": {
+        "hit": [
+          "    return this.get(`${API_BASE_URL}/api/users/api-tokens/`)",
+          "    return this.delete(`/api/users/api-tokens/${tokenId}/`)"
+        ],
+        "miss": [
+          "    return unwrap(await this.api.GET(\"/api/v0/me/api-tokens\", { params: { query: { limit: 100, cursor } } }));",
+          "    unwrap(await this.api.DELETE(\"/api/v0/api-tokens/{token_id}\", { params: { path: { token_id: tokenId } } }));"
+        ],
+        "files": {
+          "hit": [
+            "web/packages/services/src/developer/api-token.service.ts",
+            "web/apps/web/core/services/api-token.service.ts"
+          ],
+          "miss": ["docs/v0/M2-auth/M2-design.md", "e2e/stories/identity/a11-api-tokens.spec.ts"]
+        }
+      }
     }
   ],
   "exceptions": [
```

- [ ] **Step 9: oxlint 上限**

旧的令牌页面组件带走 5 条警告。

`web/apps/web/package.json`（对 Task 2 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 548",
+    "check:lint": "node ../../../tools/lint-cap.mjs 543",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 10: 检查**

Run: `grep -rn "&lt;\|&gt;\|&amp;" web/apps/web/core/components/api-token web/apps/web/core/components/settings/profile/content/pages/api-tokens.tsx`
Expected: 没有输出。

Run: `pnpm -C web/apps/web exec vitest run core/components/api-token`
Expected: `expiry.test.ts` 2 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（关键词守卫 59 条规则、没有命中；web 的 oxlint 上限 543）。

- [ ] **Step 11: 提交**

```bash
git add -A web/apps/web web/packages tools/keywords.json pnpm-lock.yaml
```
```bash
git commit -m "feat(M2/P5): the api-tokens page on the token store; @nerve/services cut down

The page lists the account's tokens with when each was created and last
used, creates one (a week, a month, three months, a year, a day picked
or never), shows it once with its CSV, and revokes. ApiTokenList is
keyed by the session's loginId. @nerve/services loses Plane's axios
base class, the last withCredentials: true, and its old token service;
Plane's IApiToken and API_TOKENS_LIST go with them. Keyword rules
with-credentials and plane-api-token-urls keep them out.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 检查都通过；`web/` 里没有 `withCredentials: true` 和 `/api/users/api-tokens`。

---

### Task 4: A11 的页面版本

**Files:**
- Create（过渡）: `e2e/fixtures/settings-pages.ts`
- Modify: `e2e/fixtures/browser.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`
- Modify: `e2e/stories/identity/a11-api-tokens.spec.ts`

**Interfaces:**
- `EMOJI_CHECK_WARNING`（`e2e/fixtures/browser.ts`）：Chromium 的 `willReadFrequently` 提示，页面每次加载编辑器时一条，不是应用自己的。S2 原来有自己的 `emojiCanvasWarning`（深链接），个人设置的每次加载也有（设置页的命令面板建了编辑器；spec 第 3 节第 9 条），两处共用这一个常量，故事按加载次数列出。
- Produces（`e2e/fixtures/settings-pages.ts`，Task 5 加上修改密码的操作）：
  - `registerOnboarded(api, email)`：经接口注册并完成引导，设置页才打开；
  - `answerTo(page, method, path, act)`：`act` 让页面发出的那个请求的响应（10 秒期限）；
  - `expectTokenGone(page, token, log)`：令牌的七种形式（原文、去掉前缀的密文、两者的十六进制和 base64、密文字节的十六进制和 base64）都不在文档、地址、localStorage、sessionStorage 和页面的控制台输出里；
  - `expectListBesideButton(page, button)`：页面上唯一的列表在按钮正下方或正上方（8 像素之内），左边缘或右边缘对齐（2 像素之内）。
- 断言先确认元素或请求存在，再断言它的内容（缺陷类别："页面没渲染所以通过"）。

**Tests:**（A11 的页面版本，2 个；API 版本不变）
- 创建：名称、说明；有效期先选"自定义日期"，日历在按钮旁展开，Escape 关闭，一次点击再打开，选一天之后显示到期时间；再改回 1 周，生成：201；数据库里的哈希等于原文的 SHA-256；原文显示一次，CSV 下载并含原文。关闭之后：原文不在页面上；列表有新令牌、"从未使用"；再打开弹窗是空表单，原文仍不在；`expectTokenGone`；经侧栏去安全页再回来、刷新之后都再核对一次。用 PAT 调 `GET /api/v0/me` 成功；在页面上撤销：204，列表空，数据库 `deleted_at` 已填，再用它得到 401。没有失败的请求、页面错误；控制台只有两次加载各一条 `EMOJI_CHECK_WARNING`。
- 两个账户：X 的标签页显示 X 的令牌；另一个标签页以 Y 登录（`writeRecord`），本标签页跟随，重新取列表（200），显示 Y 的令牌、没有 X 的，地址仍是 api-tokens 页。

- [ ] **Step 1: fixture**

`e2e/fixtures/browser.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/e2e/fixtures/browser.ts
+++ b/e2e/fixtures/browser.ts
@@ -99,6 +99,16 @@
 }
 
 /**
+ * Chromium's hint when a script reads a canvas back often, which a page logs once for each load that brings the
+ * editor: tiptap builds the editor's Emoji node, which asks is-emoji-supported about each emoji version (a canvas
+ * and getImageData each time). A deep link signed out loads it before the page goes to the sign-in (S2); a
+ * settings page's command palette (ProjectsAppPowerKProvider, kept since M1) creates an editor on every load. A
+ * hint about a third party's code, not an error of the app: stories name it in expectQuietConsole, once a load.
+ */
+export const EMOJI_CHECK_WARNING =
+  "Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently";
+
+/**
  * Checks that page logged no error and no warning since watch began (M2 design 9.6), such as React Router's
  * "navigate() should be called in useEffect". It logs a probe of each kind first and expects to find it,
  * so that a watch that does not hear the console cannot pass. expected names what page logs that is not
```

`e2e/stories/smoke/s2-web-app.spec.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/e2e/stories/smoke/s2-web-app.spec.ts
+++ b/e2e/stories/smoke/s2-web-app.spec.ts
@@ -1,21 +1,12 @@
 import type { Page, Response } from "@playwright/test";
 
 import { signInPath } from "../../fixtures/auth-pages";
-import { expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
+import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
 import { expect, test } from "../../fixtures/test";
 
 /** A page of the frontend's router, not a file: nerve answers it with index.html. */
 const deepLink = "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues";
 
-/**
- * Chromium's hint when a script reads a canvas back often, which the deep link logs once: its page's module
- * loads the editor before the page goes to the sign-in, and tiptap builds the editor's Emoji node as the
- * module loads, which asks is-emoji-supported about each emoji version (a canvas and getImageData each
- * time). A hint about a third party's code, not an error of the app.
- */
-const emojiCanvasWarning =
-  "Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently";
-
 interface Visit {
   document: Response;
   watch: PageWatch;
@@ -107,5 +98,5 @@
   await expect(page).toHaveURL(signInPath(deepLink));
   expect(failed).toEqual([]);
   expect(elsewhere).toEqual([]);
-  await expectQuietSignedOut(page, watch, [emojiCanvasWarning]);
+  await expectQuietSignedOut(page, watch, [EMOJI_CHECK_WARNING]);
 });
```

`e2e/fixtures/settings-pages.ts`（新文件）：

```ts
import { expect, type Locator, type Page, type Response } from "@playwright/test";

import type { Api } from "./api";
import { bearer, register, type AuthTokens } from "./auth";

// The personal settings (M2 design 7.7), as a person uses them. Each load of a settings page logs
// EMOJI_CHECK_WARNING (browser.ts), which the stories name.

/** Signs email up through the API, onboarded so that the settings open, and returns the session's tokens. */
export async function registerOnboarded(api: Api, email: string): Promise<AuthTokens> {
  const tokens = await register(api, email);
  const { response } = await api.PATCH("/api/v0/me/profile", {
    body: { is_onboarded: true },
    headers: bearer(tokens.access_token),
  });
  expect(response.status, "onboarded").toBe(200);
  return tokens;
}

/** Resolves with nerve's answer to the request of method to path that act makes page send. */
export async function answerTo(page: Page, method: string, path: string, act: () => Promise<void>): Promise<Response> {
  const [response] = await Promise.all([
    page.waitForResponse((res) => res.request().method() === method && new URL(res.url()).pathname === path, {
      timeout: 10_000,
    }),
    act(),
  ]);
  return response;
}

/**
 * The forms a token could take where it must not be: as it is, the secret after its prefix, and those as hex
 * and as base64, and the secret's bytes as hex and as base64.
 */
function tokenForms(token: string): string[] {
  const secret = token.slice("nrv_pat_".length);
  const bytes = Buffer.from(secret, "base64url");
  return [
    token,
    secret,
    Buffer.from(token).toString("hex"),
    Buffer.from(token).toString("base64"),
    Buffer.from(secret).toString("base64"),
    bytes.toString("hex"),
    bytes.toString("base64"),
  ];
}

/**
 * Checks that page holds token in none of its forms (tokenForms): not in the document, its address, its
 * localStorage or sessionStorage, nor in what the page logged, which log collects.
 */
export async function expectTokenGone(page: Page, token: string, log: readonly string[]): Promise<void> {
  const held = await page.evaluate(() => {
    const [local, session] = [localStorage, sessionStorage].map((area) =>
      Array.from({ length: area.length }, (_, i) => `${area.key(i)}=${area.getItem(area.key(i) ?? "")}`).join("\n")
    );
    return {
      document: document.documentElement.outerHTML,
      address: window.location.href,
      localStorage: local ?? "",
      sessionStorage: session ?? "",
    };
  });
  const where = { ...held, console: log.join("\n") };
  for (const form of tokenForms(token)) {
    for (const [place, text] of Object.entries(where)) {
      expect(text.includes(form), `${place} holds ${form}`).toBe(false);
    }
  }
}

/**
 * Checks that the one list open on page shows beside button, which opened it (M2 design 7.7): right under it or
 * over it, and lined up with it, not where the page begins.
 */
export async function expectListBesideButton(page: Page, button: Locator): Promise<void> {
  const list = page.getByRole("listbox");
  await expect(list).toBeVisible();
  const [at, beside] = await Promise.all([list.boundingBox(), button.boundingBox()]);
  expect(at, "the list's box").not.toBeNull();
  expect(beside, "the button's box").not.toBeNull();
  if (at === null || beside === null) return;
  // Under the button, or over it when the page has no room below (Popper's flip); the list's margin is 4 px.
  const under = Math.abs(at.y - (beside.y + beside.height));
  const over = Math.abs(beside.y - (at.y + at.height));
  expect(Math.min(under, over), "from the button to the list, under it or over it").toBeLessThan(8);
  // The left edges lined up (bottom-start, as the date's calendar opens) or the right edges (bottom-end, as the
  // settings' selects open).
  const left = Math.abs(at.x - beside.x);
  const right = Math.abs(at.x + at.width - (beside.x + beside.width));
  expect(Math.min(left, right), "between the left edges or the right edges").toBeLessThan(2);
}
```

- [ ] **Step 2: 故事**

`e2e/stories/identity/a11-api-tokens.spec.ts`（完整内容）：

```ts
import { readFile } from "node:fs/promises";

import type { components } from "@nerve/api-client";

import { expectTokenStored } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, newRecord, register, writeRecord } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, expectListBesideButton, expectTokenGone, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A11, personal access tokens (M2 design 2).

const weekMs = 7 * 24 * 60 * 60 * 1000;

test("A11 (page): a token created on the page shows once, and nowhere after; the list shows it without it, and revokes it", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await registerOnboarded(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  const log: string[] = [];
  page.on("console", (message) => log.push(message.text()));
  await page.goto("/settings/profile/api-tokens");
  await expect(page.getByText("No Personal token yet")).toBeVisible();

  // Created with a name, a description and a week. A day of one's own is picked in a calendar that opens beside
  // its button, and that Escape closes; the week replaces it.
  await page.getByRole("button", { name: "Add access token" }).first().click();
  await page.getByLabel("Title").fill("deploy");
  await page.getByLabel("Description").fill("ci");
  await page.getByRole("button", { name: "Set expiration date" }).click();
  await page.getByRole("option", { name: "Custom date" }).click();
  // The dropdown's button, which holds the button of its label.
  const day = page.getByRole("button", { name: "Set date" }).first();
  await day.click();
  await expectListBesideButton(page, day);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await day.click();
  await expectListBesideButton(page, day);
  await page.getByRole("grid").getByRole("button", { disabled: false }).last().click();
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(page.getByText(/^Expires .+ at /)).toBeVisible();
  await page.getByRole("button", { name: "Custom date" }).click();
  await page.getByRole("option", { name: "1 week" }).click();
  const downloaded = page.waitForEvent("download", { timeout: 10_000 });
  const answer = await answerTo(page, "POST", "/api/v0/me/api-tokens", () =>
    page.getByRole("button", { name: "Generate token" }).click()
  );
  expect(answer.status()).toBe(201);
  const created = (await answer.json()) as components["schemas"]["ApiTokenCreated"];
  expect(created).toMatchObject({ label: "deploy", description: "ci", last_used: null });
  const createdAt = Date.now();
  expect(Math.abs(new Date(created.expired_at ?? "").getTime() - (createdAt + weekMs))).toBeLessThan(60_000);
  await expectTokenStored(db, created.id, created.token, created.expired_at ?? "");

  // The token shows this once, and the CSV downloaded has it.
  await expect(page.getByText(created.token)).toBeVisible();
  const csv = await readFile(await (await downloaded).path(), "utf8");
  expect(csv).toContain(created.token);

  // Closed, the list has the new token, without the token itself: nowhere on the page, in its address, its
  // storage or its console; nor after going to another tab and back, nor after a reload.
  await page.getByRole("button", { name: "Close" }).click();
  await expect(page.getByText(created.token)).toHaveCount(0);
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  // Opened again, the modal asks for a new token.
  await page.getByRole("button", { name: "Add access token" }).first().click();
  await expect(page.getByLabel("Title")).toHaveValue("");
  await expect(page.getByText(created.token)).toHaveCount(0);
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByLabel("Title")).toHaveCount(0);
  await expect(page.getByText("Never used")).toBeVisible();
  await expectTokenGone(page, created.token, log);
  await page.getByRole("button", { name: "Security" }).click();
  await expect(page).toHaveURL("/settings/profile/security");
  await page.getByRole("button", { name: "Personal Access Tokens" }).click();
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  await expectTokenGone(page, created.token, log);
  await page.reload();
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  await expectTokenGone(page, created.token, log);

  // The token works until it is revoked on the page, which then lists it no more.
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(200);
  await page.getByText("deploy", { exact: true }).hover();
  await page.getByRole("button", { name: "Revoke token" }).click();
  const revoked = await answerTo(page, "DELETE", `/api/v0/api-tokens/${created.id}`, () =>
    page.getByRole("button", { name: "Delete" }).click()
  );
  expect(revoked.status()).toBe(204);
  await expect(page.getByText("No Personal token yet")).toBeVisible();
  const [row] = await db.query<{ deleted_at: Date | null }>("SELECT deleted_at FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(row?.deleted_at).not.toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(401);

  expect(watch.apiFailures).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  // Two loads of the page: the first, and the reload.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

test("A11 (page): the list is the account's own: when another tab signs another account in, it shows that account's", async ({
  api,
  context,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const tabA = await signedInPage(await registerOnboarded(api, x));
  const watch = await watchPage(tabA);
  await createPAT(api, (await login(api, x)).access_token, { label: "token of x" });
  await createPAT(api, (await registerOnboarded(api, y)).access_token, { label: "token of y" });
  await tabA.goto("/settings/profile/api-tokens");
  await expect(tabA.getByText("token of x")).toBeVisible();

  // Tab B keeps a sign-in of Y as the token manager does; tab A follows, and its list is Y's.
  const tabB = await context.newPage();
  await tabB.goto("/site.webmanifest.json");
  const listed = answerTo(tabA, "GET", "/api/v0/me/api-tokens", async () => {
    await writeRecord(tabB, newRecord(await login(api, y)));
  });
  expect((await listed).status()).toBe(200);
  await expect(tabA.getByText("token of y")).toBeVisible();
  await expect(tabA.getByText("token of x")).toHaveCount(0);
  await expect(tabA).toHaveURL("/settings/profile/api-tokens");

  expect(watch.apiFailures).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(tabA, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("A11 (API): a token creates, lists page by page and revokes another; a revoked or expired token fails", async ({
  api,
  db,
}, testInfo) => {
  const admin = await createPAT(api, (await register(api, emailFor(testInfo))).access_token, { label: "admin" });
  const expiredAt = new Date(Date.now() + weekMs).toISOString();

  const created = await createPAT(api, admin.token, { label: "deploy", description: "ci", expired_at: expiredAt });

  expect(created).toMatchObject({ label: "deploy", description: "ci", last_used: null });
  expect(created.token).toMatch(/^nrv_pat_[A-Za-z0-9_-]{43}$/);
  await expectTokenStored(db, created.id, created.token, expiredAt);

  // The list, a token a page, newest first, never shows a token itself.
  const list = async (cursor?: string): Promise<components["schemas"]["ApiToken"][]> => {
    const { data, response } = await api.GET("/api/v0/me/api-tokens", {
      params: { query: { limit: 1, cursor } },
      headers: bearer(admin.token),
    });
    expect(response.status).toBe(200);
    const page = data?.data ?? [];
    return data?.next_cursor ? [...page, ...(await list(data.next_cursor))] : page;
  };
  const listed = await list();
  expect(listed.map((t) => t.id)).toEqual([created.id, admin.id]);
  expect(JSON.stringify(listed)).not.toContain(created.token);

  // The token authenticates, and its use is recorded.
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(200);
  const [used] = await db.query<{ last_used: Date | null }>("SELECT last_used FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(used?.last_used).not.toBeNull();

  // Revoked, it stops at once and leaves the list.
  const revoked = await api.DELETE("/api/v0/api-tokens/{token_id}", {
    params: { path: { token_id: created.id } },
    headers: bearer(admin.token),
  });
  expect(revoked.response.status).toBe(204);
  const [row] = await db.query<{ deleted_at: Date | null }>("SELECT deleted_at FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(row?.deleted_at).not.toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(401);
  expect((await list()).map((t) => t.id)).toEqual([admin.id]);

  // Another account's token is not found, and it keeps working.
  const other = await createPAT(api, (await register(api, emailFor(testInfo, "other"))).access_token);
  const foreign = await api.DELETE("/api/v0/api-tokens/{token_id}", {
    params: { path: { token_id: other.id } },
    headers: bearer(admin.token),
  });
  expect(foreign.response.status).toBe(404);
  expect(foreign.error?.code).toBe("identity.api_token_not_found");
  expect((await api.GET("/api/v0/me", { headers: bearer(other.token) })).response.status).toBe(200);

  // A token past its expiry fails too.
  const old = await createPAT(api, admin.token, { label: "old" });
  await db.query("UPDATE api_tokens SET expired_at = now() - interval '1 minute' WHERE id = $1", [old.id]);
  expect((await api.GET("/api/v0/me", { headers: bearer(old.token) })).response.status).toBe(401);
});
```

- [ ] **Step 3: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

Run: `make e2e`
Expected: 全部通过（A11 的三个测试：两个页面版本、一个 API 版本）。

- [ ] **Step 4: 提交**

```bash
git add e2e
```
```bash
git commit -m "test(M2/P5): A11's page versions: a token shows once, and the list is the account's own

A token created on the page shows once with its CSV, and afterwards is
in none of seven forms in the document, the address, the storage or the
console, not after a reload, nor in the modal opened again; the
calendar of a day picked opens beside its button. When another tab
signs another account in, the list is that account's. S2 and the
settings' stories share EMOJI_CHECK_WARNING, which every load of a
settings page logs too.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过。

---

### Task 5: 安全页的 PAT 列表；A7 的页面版本

**Files:**
- Modify: `web/apps/web/core/components/settings/profile/content/pages/security.tsx`
- Modify: `web/packages/i18n/src/locales/{en,zh-CN}/settings.json`
- Modify: `e2e/fixtures/settings-pages.ts`、`e2e/stories/identity/a7-change-password.spec.ts`

**Interfaces:**
- Produces（spec 2.6，M2 设计 7.7、3.5）：修改密码的表单下方一节"个人访问令牌"：标题、说明"修改密码不会撤销这些令牌：每个令牌在到期或在这里撤销之前一直有效"，`ApiTokenList`（Task 3，可以就地撤销），没有令牌时一行"你没有个人访问令牌"。修改密码的错误处理是 P4 做的，不变。
- `submitPasswordChange(page, current, next)`（fixture）：填当前密码、新密码两次，提交，返回 nerve 对那一个请求的状态码。
- A7 的页面版本和 API 版本只差在"会话是否延续"（Codex M-6）：页面自己的会话延续，其他会话结束，PAT 不受影响。

**Tests:**（A7 的页面版本，1 个；API 版本不变）账户另有一个会话和一个 PAT"deploy"。安全页列出"deploy"和那句说明；当前密码错：422，当前密码字段下"The current password is wrong."；新密码太常见：422，新密码字段下"This password is too common"，前一条消失；数据库没变。改成功：204，"Password changed successfully."；页面的刷新令牌仍有效，别的会话结束（`expectPasswordChanged`）；刷新之后仍在安全页、仍列出"deploy"；PAT 仍能调 `GET /api/v0/me`，旧密码登录 401。失败的请求只有两个 422，控制台只有它们的两条和两次加载的 `EMOJI_CHECK_WARNING`。

- [ ] **Step 1: 安全页**

`web/apps/web/core/components/settings/profile/content/pages/security.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/security.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/security.tsx
@@ -18,6 +18,7 @@
 import { PasswordStrengthIndicator } from "@nerve/ui";
 import { getPasswordStrength } from "@nerve/utils";
 // components
+import { ApiTokenList } from "@/components/api-token/token-list";
 import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
 // helpers
 import { errorMessageKey, fieldErrorKeys } from "@/helpers/authentication.helper";
@@ -263,6 +264,15 @@
           </div>
         </div>
       </form>
+      <section className="mt-12 flex flex-col gap-4">
+        <ProfileSettingsHeading
+          title={t("account_settings.security.api_tokens.title")}
+          description={t("account_settings.security.api_tokens.description")}
+        />
+        <ApiTokenList
+          empty={<p className="text-13 text-placeholder">{t("account_settings.security.api_tokens.empty")}</p>}
+        />
+      </section>
     </div>
   );
 });
```

`web/packages/i18n/src/locales/en/settings.json`（对 Task 3 版本的差异）：

```diff
--- a/web/packages/i18n/src/locales/en/settings.json
+++ b/web/packages/i18n/src/locales/en/settings.json
@@ -27,6 +27,13 @@
         "set_date": "Set date",
         "required": "Choose when the token expires, or that it never does."
       }
+    },
+    "security": {
+      "api_tokens": {
+        "title": "Personal access tokens",
+        "description": "Changing your password does not revoke these tokens: each one works until it expires or you revoke it here.",
+        "empty": "You have no personal access tokens."
+      }
     }
   },
   "profile": {
```

`web/packages/i18n/src/locales/zh-CN/settings.json`（对 Task 3 版本的差异）：

```diff
--- a/web/packages/i18n/src/locales/zh-CN/settings.json
+++ b/web/packages/i18n/src/locales/zh-CN/settings.json
@@ -27,6 +27,13 @@
         "set_date": "选择日期",
         "required": "请选择令牌何时过期，或设为永不过期。"
       }
+    },
+    "security": {
+      "api_tokens": {
+        "title": "个人访问令牌",
+        "description": "修改密码不会撤销这些令牌：每个令牌在过期或您在此撤销之前一直有效。",
+        "empty": "您没有个人访问令牌。"
+      }
     }
   },
   "profile": {
```

- [ ] **Step 2: fixture 和故事**

`e2e/fixtures/settings-pages.ts`（对 Task 4 版本的差异）：

```diff
--- a/e2e/fixtures/settings-pages.ts
+++ b/e2e/fixtures/settings-pages.ts
@@ -91,3 +91,17 @@
   const right = Math.abs(at.x + at.width - (beside.x + beside.width));
   expect(Math.min(left, right), "between the left edges or the right edges").toBeLessThan(2);
 }
+
+/**
+ * Fills the security page's form, which page shows, with the current password and a new one typed twice, and
+ * submits it: resolves with the status of nerve's answer to the one change it sends.
+ */
+export async function submitPasswordChange(page: Page, current: string, next: string): Promise<number> {
+  await page.locator("#old_password").fill(current);
+  await page.locator("#new_password").fill(next);
+  await page.locator("#confirm_password").fill(next);
+  const answer = await answerTo(page, "POST", "/api/v0/me/change-password", () =>
+    page.getByRole("button", { name: "Change password" }).click()
+  );
+  return answer.status();
+}
```

`e2e/stories/identity/a7-change-password.spec.ts`（完整内容）：

```ts
import { accountOf, expectPasswordChanged, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded, submitPasswordChange } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A7, changing the password (M2 design 2). The page keeps its own session,
// which a token has not: the two versions differ in that alone, which the
// shared assertion takes as survivingSession (Codex M-6).

const newPassword = "N3w-Passw0rd!";

/** What the browser logs of an answer of 422, which the page shows under a field. */
const refusedResourceError =
  "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)";

test("A7 (page): the security page changes the password, keeps its own session, and lists the tokens it leaves alone", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  // Another session of the account, which the change ends, and a token, which it leaves (M2 design 3.5).
  await login(api, email);
  const pat = await createPAT(api, tokens.access_token, { label: "deploy" });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/security");

  // The page lists the account's tokens, and says the change leaves them.
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  await expect(page.getByText("Changing your password does not revoke these tokens", { exact: false })).toBeVisible();
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // A wrong current password shows under its field; a new password too common, under the new one. Nothing
  // changes.
  expect(await submitPasswordChange(page, "Wr0ng-password", newPassword)).toBe(422);
  await expect(page.getByText("The current password is wrong.")).toBeVisible();
  expect(await submitPasswordChange(page, password, "Password1!")).toBe(422);
  await expect(page.getByText("This password is too common")).toBeVisible();
  await expect(page.getByText("The current password is wrong.")).toHaveCount(0);
  expect(await accountOf(db, email)).toEqual(before);

  expect(await submitPasswordChange(page, password, newPassword)).toBe(204);
  await expect(page.getByText("Password changed successfully.")).toBeVisible();
  // The page's session goes on; every other ends; the token is as it was.
  const held = (await recordOf(page))?.refresh_token;
  expect(held, "the page's refresh token").toBeDefined();
  await expectPasswordChanged(db, before, tokensBefore, held);
  await page.reload();
  await expect(page).toHaveURL("/settings/profile/security");
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  const old = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(old.response.status).toBe(401);

  expect(watch.apiFailures).toEqual(["422 POST /api/v0/me/change-password", "422 POST /api/v0/me/change-password"]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    errors: [refusedResourceError, refusedResourceError],
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
  });
});

test("A7 (API): a personal access token changes the password; every session ends, the token goes on", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const change = (current: string) =>
    api.POST("/api/v0/me/change-password", {
      body: { current_password: current, new_password: newPassword },
      headers: bearer(pat.token),
    });
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // A wrong current password: its problem, and nothing changes.
  const wrong = await change("Wr0ng-password");
  expect(wrong.response.status).toBe(422);
  expect(wrong.error?.code).toBe("identity.current_password_incorrect");
  expect(await accountOf(db, email)).toEqual(before);

  const changed = await change(password);
  expect(changed.response.status).toBe(204);
  // A token has no session of its own: every session ends (M2 design 3.5).
  await expectPasswordChanged(db, before, tokensBefore);

  // The token goes on; the old password no longer signs in, the new one does.
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  const old = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(old.response.status).toBe(401);
  const renewed = await api.POST("/api/v0/auth/login", { body: { email, password: newPassword } });
  expect(renewed.response.status).toBe(200);
});
```

- [ ] **Step 3: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

Run: `make e2e`
Expected: 全部通过。

- [ ] **Step 4: 提交**

```bash
git add web/apps/web/core/components/settings/profile/content/pages/security.tsx web/packages/i18n e2e
```
```bash
git commit -m "feat(M2/P5): the security page lists the tokens a password change leaves

Under the change-password form, the account's personal access tokens,
revocable in place, and a line that changing the password does not
revoke them. A7's page version: each refusal under its field, the
page's own session goes on while every other ends, and the token still
works.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过。

---

### Task 6: preferences：时区列表来自 `GET /api/v0/timezones`，错误按 `code` 提示；A8、A9 的页面版本

**Files:**
- Modify: `web/apps/web/core/services/timezone.service.ts`、`web/apps/web/core/hooks/use-timezone.tsx`
- Delete: `web/packages/types/src/timezone.ts`；Modify: `web/packages/types/src/index.ts`
- Modify: `tools/keywords.json`
- Modify: `web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`、`web/apps/web/core/components/profile/start-of-week-preference.tsx`、`web/apps/web/core/components/appearance/theme-switcher.tsx`
- Modify: `e2e/stories/identity/a8-update-me.spec.ts`、`a9-preferences.spec.ts`
- Modify（过渡）: `web/apps/web/package.json`（oxlint 上限 543 → 541）

**Interfaces:**
- Produces（spec 2.7，M2 设计 5.3、7.7）：
  - `class TimezoneService { list(): Promise<Timezone[]> }`：`publicClient.GET("/api/v0/timezones")`（公开的接口，和实例信息一样不带令牌），经 `unwrap`，返回 `data`。`use-timezone.tsx` 用生成的 `Timezone`，把同一时区的地点合成一个选项（`Map`），选项的内容函数移到模块级。Plane 的 `TTimezoneObject` 删除。
  - 关键词规则 `plane-timezone-urls`（`/api/timezones/`）。
  - 主题、语言、时区、每周第一天：成功的提示用 `t("toast.success")` 和已有的 `power_k.preferences_actions.toast.*.success` 文案；失败的提示用 `t(errorMessageKey(error))`（frontend-changes 3.2：按 `code` 取文案）。行为不变：主题切换之后刷新页面（7.7）。

**Tests:**（页面版本各 1 个；API 版本不变）
- A8：general 页改名、姓、显示名，保存：`PATCH /api/v0/me` 200。preferences 页的时区按钮"UTC"：列表在按钮旁展开；Escape 关闭，一次点击再打开；搜"Asia/Shanghai"，选"Beijing"：200。数据库的四个字段和 `updated_at` 已变；刷新之后两个页面都显示新值。没有失败的请求、没有旧地址的请求（`oldApiRequests` 为空，时区不再走 `/api/timezones/`）。
- A9：主题"System Preference"的列表在按钮旁展开，选"Dark"：200，页面自己刷新；语言"English"的列表在按钮旁，选"简体中文"：200，页面立即变成中文；中文下主题"深色"的列表在按钮旁，Escape 关闭，一次点击再打开；每周第一天"Sunday"的列表在按钮旁，选"Monday"：200。数据库的资料是 `{theme: "dark", language: "zh-CN", start_of_the_week: 1}`；刷新之后都显示。控制台只有三次加载的 `EMOJI_CHECK_WARNING`。

- [ ] **Step 1: 时区列表**

`web/apps/web/core/services/timezone.service.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Timezone } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
import { publicClient } from "@/lib/auth/api-client";

export class TimezoneService {
  /** The time zones to choose from, with their offsets now (M2 design 5.3): public, like the instance. */
  async list(): Promise<Timezone[]> {
    return unwrap(await publicClient.GET("/api/v0/timezones")).data;
  }
}
```

`web/apps/web/core/hooks/use-timezone.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
import type { Timezone } from "@nerve/api-client";
// services
import { TimezoneService } from "@/services/timezone.service";

const timezoneService = new TimezoneService();

// One option for each time zone: the labels of the places that share it, together.
const groupTimezones = (timezones: Timezone[]): Timezone[] => {
  const grouped = new Map<string, Timezone>();
  for (const timezone of timezones) {
    const existing = grouped.get(timezone.value);
    grouped.set(timezone.value, existing ? { ...existing, label: `${existing.label}, ${timezone.label}` } : timezone);
  }
  return Array.from(grouped.values());
};

// An option's content: the offset, then the places.
const timezoneLabel = (timezone: Timezone) => (
  <div className="flex gap-1.5">
    <span className="text-placeholder">{timezone.utc_offset}</span>
    <span className="text-secondary">{timezone.label}</span>
  </div>
);

const useTimezone = () => {
  // fetching the timezone from the server
  const {
    data: timezones,
    isLoading: timezoneIsLoading,
    error: timezonesError,
  } = useSWR("TIMEZONES_LIST", () => timezoneService.list(), {
    refreshInterval: 0,
  });

  // derived values
  const isDisabled = timezoneIsLoading || timezonesError || !timezones;

  const options = [
    ...groupTimezones(timezones ?? []).map((timezone) => ({
      value: timezone.value,
      query: `${timezone.value} ${timezone.label}, ${timezone.gmt_offset}, ${timezone.utc_offset}`,
      content: timezoneLabel(timezone),
    })),
    {
      value: "UTC",
      query: "utc, coordinated universal time",
      content: "UTC",
    },
  ];

  const selectedTimezone = (value: string | undefined) => options.find((option) => option.value === value)?.content;

  return {
    timezones: options,
    isLoading: timezoneIsLoading,
    error: timezonesError,
    disabled: isDisabled,
    selectedValue: selectedTimezone,
  };
};

export default useTimezone;
```

```bash
git rm web/packages/types/src/timezone.ts
```

`web/packages/types/src/index.ts`（对 Task 3 版本的差异）：

```diff
--- a/web/packages/types/src/index.ts
+++ b/web/packages/types/src/index.ts
@@ -28,7 +28,6 @@
 export * from "./search";
 export * from "./settings";
 export * from "./state";
-export * from "./timezone";
 export * from "./users";
 export * from "./utils";
 export * from "./view-props";
```

Run: `grep -rn "TTimezoneObject\|/api/timezones/\|timezoneService.fetch" web --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build --exclude-dir=.turbo`
Expected: 没有输出。

`tools/keywords.json`（对 Task 3 版本的差异）：

```diff
--- a/tools/keywords.json
+++ b/tools/keywords.json
@@ -1973,6 +1973,27 @@
           "miss": ["docs/v0/M2-auth/M2-design.md", "e2e/stories/identity/a11-api-tokens.spec.ts"]
         }
       }
+    },
+    {
+      "id": "plane-timezone-urls",
+      "phase": "M2/P5",
+      "why": "Plane 的时区列表地址 /api/timezones/；web 改用生成的客户端调用公开的 /api/v0/timezones（M2 设计 5.3、7.9）",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "/api/timezones/",
+        "flags": ""
+      },
+      "samples": {
+        "hit": ["    return this.get(`/api/timezones/`)", "    return this.get(\"/api/timezones/\")"],
+        "miss": ["    return unwrap(await publicClient.GET(\"/api/v0/timezones\")).data;"],
+        "files": {
+          "hit": ["web/apps/web/core/services/timezone.service.ts", "web/apps/web/core/hooks/use-timezone.tsx"],
+          "miss": ["docs/v0/M2-auth/M2-design.md", "server/internal/modules/instance/app/list_timezones.go"]
+        }
+      }
     }
   ],
   "exceptions": [
```

- [ ] **Step 2: preferences 的提示**

`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
@@ -14,6 +14,8 @@
 import { TimezoneSelect } from "@/components/global";
 import { StartOfWeekPreference } from "@/components/profile/start-of-week-preference";
 import { SettingsControlItem } from "@/components/settings/control-item";
+// helpers
+import { errorMessageKey } from "@/helpers/authentication.helper";
 // hooks
 import { useUser, useUserProfile } from "@/hooks/store/user";
 
@@ -33,16 +35,12 @@
       try {
         await updateCurrentUser({ user_timezone: value });
         setToast({
-          title: "Success!",
-          message: "Timezone updated successfully",
+          title: t("toast.success"),
+          message: t("power_k.preferences_actions.toast.timezone.success"),
           type: TOAST_TYPE.SUCCESS,
         });
-      } catch (_error) {
-        setToast({
-          title: "Error!",
-          message: "Failed to update timezone",
-          type: TOAST_TYPE.ERROR,
-        });
+      } catch (error) {
+        setToast({ title: t("toast.error"), message: t(errorMessageKey(error)), type: TOAST_TYPE.ERROR });
       }
     };
 
@@ -50,16 +48,12 @@
       try {
         await updateUserProfile({ language: value });
         setToast({
-          title: "Success!",
-          message: "Language updated successfully",
+          title: t("toast.success"),
+          message: t("power_k.preferences_actions.toast.generic.success"),
           type: TOAST_TYPE.SUCCESS,
         });
-      } catch (_error) {
-        setToast({
-          title: "Error!",
-          message: "Failed to update language",
-          type: TOAST_TYPE.ERROR,
-        });
+      } catch (error) {
+        setToast({ title: t("toast.error"), message: t(errorMessageKey(error)), type: TOAST_TYPE.ERROR });
       }
     };
 
```

`web/apps/web/core/components/profile/start-of-week-preference.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/profile/start-of-week-preference.tsx
+++ b/web/apps/web/core/components/profile/start-of-week-preference.tsx
@@ -8,10 +8,13 @@
 // nerve imports
 import type { StartOfTheWeek } from "@nerve/api-client";
 import { START_OF_THE_WEEK_OPTIONS } from "@nerve/constants";
+import { useTranslation } from "@nerve/i18n";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
 import { CustomSelect } from "@nerve/ui";
 // components
 import { SettingsControlItem } from "@/components/settings/control-item";
+// helpers
+import { errorMessageKey } from "@/helpers/authentication.helper";
 // hooks
 import { useUserProfile } from "@/hooks/store/user";
 
@@ -23,13 +26,18 @@
 }) {
   // hooks
   const { data: userProfile, updateUserProfile } = useUserProfile();
+  const { t } = useTranslation();
 
   const handleStartOfWeekChange = async (val: StartOfTheWeek) => {
     try {
       await updateUserProfile({ start_of_the_week: val });
-      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success", message: "First day of the week updated successfully" });
-    } catch (_error) {
-      setToast({ type: TOAST_TYPE.ERROR, title: "Update failed", message: "Please try again later." });
+      setToast({
+        type: TOAST_TYPE.SUCCESS,
+        title: t("toast.success"),
+        message: t("power_k.preferences_actions.toast.generic.success"),
+      });
+    } catch (error) {
+      setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
     }
   };
 
```

`web/apps/web/core/components/appearance/theme-switcher.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/appearance/theme-switcher.tsx
+++ b/web/apps/web/core/components/appearance/theme-switcher.tsx
@@ -15,6 +15,8 @@
 // components
 import { ThemeSwitch } from "@/components/core/theme/theme-switch";
 import { SettingsControlItem } from "@/components/settings/control-item";
+// helpers
+import { errorMessageKey } from "@/helpers/authentication.helper";
 // hooks
 import { useUserProfile } from "@/hooks/store/user";
 
@@ -51,8 +53,8 @@
             message: () => "Reloading to apply changes...",
           },
           error: {
-            title: "Error!",
-            message: () => "Failed to update theme. Please try again.",
+            title: t("toast.error"),
+            message: (error) => t(errorMessageKey(error)),
           },
         });
         // Wait for the promise to resolve, then reload after showing toast
@@ -62,7 +64,7 @@
         console.error("Error updating theme:", error);
       }
     },
-    [setTheme, updateUserTheme]
+    [setTheme, updateUserTheme, t]
   );
 
   if (!userProfile) return null;
```

- [ ] **Step 3: oxlint 上限**

`use-timezone.tsx` 的两条警告随改写消失（`unicorn/consistent-function-scoping`：组件内定义的函数；`no-unsafe-optional-chaining`：展开 `?.` 的结果）。

`web/apps/web/package.json`（对 Task 3 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 543",
+    "check:lint": "node ../../../tools/lint-cap.mjs 541",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 4: 故事**

`e2e/stories/identity/a8-update-me.spec.ts`（完整内容）：

```ts
import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, expectListBesideButton, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A8, changing the names and the time zone (M2 design 2).

test("A8 (page): the general page changes the names, the preferences page the time zone; both hold after a reload", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  const before = await accountOf(db, email);
  await page.goto("/settings/profile/general");

  await page.locator("#first_name").fill("Ada");
  await page.locator("#last_name").fill("Lovelace");
  await page.locator("#display_name").fill("ada");
  const saved = await answerTo(page, "PATCH", "/api/v0/me", () =>
    page.getByRole("button", { name: "Save changes" }).click()
  );
  expect(saved.status()).toBe(200);

  // The time zone, from nerve's list (GET /api/v0/timezones), in the list that opens beside its button.
  await page.getByRole("button", { name: "Preferences" }).click();
  const timezone = page.getByRole("button", { name: "UTC" });
  await timezone.click();
  await expectListBesideButton(page, timezone);
  // Escape closes it, and the next click opens it again.
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await timezone.click();
  await expectListBesideButton(page, timezone);
  await page.getByPlaceholder("Search").fill("Asia/Shanghai");
  const zoned = await answerTo(page, "PATCH", "/api/v0/me", () =>
    page.getByRole("option", { name: "Beijing" }).click()
  );
  expect(zoned.status()).toBe(200);

  const change = { first_name: "Ada", last_name: "Lovelace", display_name: "ada", user_timezone: "Asia/Shanghai" };
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());

  // After a reload, both pages show what nerve has.
  await page.reload();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();
  await page.getByRole("button", { name: "Profile" }).click();
  await expect(page.locator("#first_name")).toHaveValue("Ada");
  await expect(page.locator("#last_name")).toHaveValue("Lovelace");
  await expect(page.locator("#display_name")).toHaveValue("ada");

  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

test("A8 (API): a personal access token changes the names and the time zone; null is a 400", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const before = await accountOf(db, email);
  const change = { first_name: "Ada", last_name: "Lovelace", display_name: "ada", user_timezone: "Asia/Shanghai" };

  const { data, response } = await api.PATCH("/api/v0/me", { body: change, headers: bearer(pat.token) });

  expect(response.status).toBe(200);
  expect(data).toMatchObject(change);
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());

  // null breaks the contract: the platform's 400, and nothing changes.
  const nulled = await api.PATCH("/api/v0/me", {
    // @ts-expect-error -- null for a name breaks the contract on purpose
    body: { first_name: null },
    headers: bearer(pat.token),
  });
  expect(nulled.response.status).toBe(400);
  expect(nulled.error?.code).toBe("bad_request");
  expect(nulled.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "first_name", code: "invalid_format" },
  ]);
  expect(await accountOf(db, email)).toEqual(after);
});
```

`e2e/stories/identity/a9-preferences.spec.ts`（完整内容）：

```ts
import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, expectListBesideButton, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A9, changing the preferences (M2 design 2), with the theme's list beside
// its button in both languages (7.7, 9.6).

const profileOf = async (db: Parameters<typeof accountOf>[0], email: string) =>
  db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [
    (await accountOf(db, email)).id,
  ]);

test("A9 (page): the preferences page changes the theme, the language and the first day of the week, each list beside its button; they hold after a reload", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  await page.goto("/settings/profile/preferences");

  // In English, the theme's list opens beside its button. Dark: the page reloads itself to apply it.
  const theme = page.getByRole("button", { name: "System Preference" });
  await theme.click();
  await expectListBesideButton(page, theme);
  const reloaded = page.waitForEvent("load", { timeout: 10_000 });
  const themed = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "Dark", exact: true }).click()
  );
  expect(themed.status()).toBe(200);
  await reloaded;

  // The language: the page is in Chinese at once.
  const language = page.getByRole("button", { name: "English" });
  await language.click();
  await expectListBesideButton(page, language);
  const spoken = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "简体中文" }).click()
  );
  expect(spoken.status()).toBe(200);
  await expect(page.getByText("语言和时间")).toBeVisible();

  // In Chinese, the theme's list opens beside its button too; Escape closes it, and the next click opens it again.
  const themeZh = page.getByRole("button", { name: "深色" });
  await themeZh.click();
  await expectListBesideButton(page, themeZh);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await themeZh.click();
  await expectListBesideButton(page, themeZh);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);

  // The first day of the week.
  const week = page.getByRole("button", { name: "Sunday" });
  await week.click();
  await expectListBesideButton(page, week);
  const started = await answerTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("option", { name: "Monday" }).click()
  );
  expect(started.status()).toBe(200);

  const change = { theme: "dark", language: "zh-CN", start_of_the_week: 1 };
  expect(await profileOf(db, email)).toEqual([change]);
  // After a reload, the page shows them all, in Chinese.
  await page.reload();
  await expect(page.getByRole("button", { name: "深色" })).toBeVisible();
  await expect(page.getByRole("button", { name: "简体中文" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Monday" })).toBeVisible();

  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  // Three loads: the first, the theme's own reload, and the last.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
  });
});

test("A9 (API): a personal access token changes the theme, the language and the first day of the week", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const change = { theme: "dark", language: "zh-CN", start_of_the_week: 1 } as const;

  const { data, response } = await api.PATCH("/api/v0/me/profile", { body: change, headers: bearer(pat.token) });

  expect(response.status).toBe(200);
  expect(data).toMatchObject(change);
  expect(await profileOf(db, email)).toEqual([change]);
  const read = await api.GET("/api/v0/me/profile", { headers: bearer(pat.token) });
  expect(read.data).toEqual(data);
});
```

- [ ] **Step 5: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（关键词守卫 60 条规则、没有命中；web 的 oxlint 上限 541）。

Run: `make e2e`
Expected: 全部通过。

- [ ] **Step 6: 提交**

```bash
git add -A web/apps/web web/packages/types tools/keywords.json e2e
```
```bash
git commit -m "feat(M2/P5): preferences on the new API: time zones from /api/v0/timezones

The time zone list comes from nerve's public GET /api/v0/timezones with
the generated Timezone; Plane's address and TTimezoneObject go, and the
keyword rule plane-timezone-urls keeps the address out. Theme,
language, time zone and first day of the week say why a change failed
from the problem's code. A8's and A9's page versions check each list
beside its button, in English and in Chinese, and reopening after
Escape.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过；`web/` 里没有 `/api/timezones/`。

---

### Task 7: general 页的保存、停用账户、维护页的文案；A12 的页面版本

**Files:**
- Modify: `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`、`web/apps/web/core/components/instance/maintenance-message.tsx`
- Modify: `web/packages/i18n/src/locales/{en,zh-CN}/common.json`
- Modify: `e2e/stories/identity/a12-deactivate.spec.ts`

**Interfaces:**
- Produces（spec 2.8，M2 设计 7.3、7.7；P4 spec 第 3 节第 14 条）：
  - general 页的保存：`PATCH /api/v0/me`；成功提示"个人资料已更新"；nerve 拒绝时，`first_name`、`last_name`、`display_name` 的字段错误显示在字段下（`fieldErrorKeys`），其余（`needsErrorBanner`）在提示里，文案按 `code` 取。
  - 停用账户：确认弹窗的文案按 Nerve 的行为改写（所有地方退出；密码和 PAT 都不能用；什么都不删除，也不发邮件；由服务器管理员恢复）；成功后会话已由 `deactivateAccount` 结束（P4），登录页接管并显示"你的账户已停用"；失败按 `errorMessageKey` 提示。
  - 维护页（`InstanceWrapper` 读不到实例信息时）：标题"暂时无法连接 Nerve"，说明页面没能读取这台服务器的设置，会自动重试，服务器恢复应答后继续。不再说 nerve 没有启动或在维护（nerve 暂时不可达时也显示它）。文案进 `common.json`（`maintenance.title`、`maintenance.description`）。
  - 新文案键：`profile_updated`、`account_deactivated`、`maintenance.*`；改写 `deactivate_account_description`、`deactivate_your_account_description`。

**Tests:**（A12 的页面版本，1 个；API 版本不变）账户另有一个会话和一个 PAT。general 页点"Deactivate account"：弹窗含"ask an administrator of this server to reactivate it"；确认：`POST /api/v0/me/deactivate` 204；页面回到 `signInPath("/settings/profile/general")`，显示"Your account is deactivated."，浏览器里没有记录；数据库里账户已停用、会话全部结束、PAT 保留（`expectDeactivated`）。再登录：403，"This account is deactivated."。管理员 `nerve users activate`：输出"1 API tokens are usable again"；登录 200，进入 `/onboarding`（停用让引导重来）；PAT 又能用。失败的请求只有那个 403。

- [ ] **Step 1: general 页的保存**

`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
@@ -14,13 +14,15 @@
 import type { User, UserUpdate } from "@nerve/api-client";
 import { useTranslation } from "@nerve/i18n";
 import { Button } from "@nerve/propel/button";
-import { setPromiseToast } from "@nerve/propel/toast";
+import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
 
 import { getFileURL } from "@nerve/utils";
 // components
 import { DeactivateAccountModal } from "@/components/account/deactivate-account-modal";
 import { CoverImage } from "@/components/common/cover-image";
 import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
+// helpers
+import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/helpers/authentication.helper";
 // hooks
 import { useUser } from "@/hooks/store/user";
 // utils
@@ -37,6 +39,9 @@
   user: User;
 };
 
+/** The fields of UserUpdate the form has, whose errors show under them. */
+const FIELDS = ["first_name", "last_name", "display_name"] as const;
+
 export const GeneralProfileSettingsForm = observer(function GeneralProfileSettingsForm(props: Props) {
   const { user } = props;
   // states
@@ -49,6 +54,7 @@
     handleSubmit,
     watch,
     control,
+    setError,
     formState: { errors },
   } = useForm<TUserProfileForm>({
     defaultValues: {
@@ -69,19 +75,21 @@
       display_name: formData?.display_name,
     };
 
-    const updatePromise = updateCurrentUser(userPayload).finally(() => setIsLoading(false));
-
-    setPromiseToast(updatePromise, {
-      loading: "Updating...",
-      success: {
-        title: "Success!",
-        message: () => `Profile updated successfully.`,
-      },
-      error: {
-        title: "Error!",
-        message: () => `There was some error in updating your profile. Please try again.`,
-      },
-    });
+    try {
+      await updateCurrentUser(userPayload);
+      setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t("profile_updated") });
+    } catch (error) {
+      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
+      const fields = fieldErrorKeys(error);
+      for (const field of FIELDS) {
+        const key = fields[field];
+        if (key !== undefined) setError(field, { type: "manual", message: t(key) });
+      }
+      if (needsErrorBanner(error, FIELDS))
+        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
+    } finally {
+      setIsLoading(false);
+    }
   };
 
   return (
```

- [ ] **Step 2: 停用账户、维护页**

`web/apps/web/core/components/account/deactivate-account-modal.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/account/deactivate-account-modal.tsx
+++ b/web/apps/web/core/components/account/deactivate-account-modal.tsx
@@ -11,6 +11,8 @@
 import { DeleteOutline } from "@makeplane/propel/icons";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
 import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
+// helpers
+import { errorMessageKey } from "@/helpers/authentication.helper";
 // hooks
 import { useUser } from "@/hooks/store/user";
 
@@ -38,21 +40,20 @@
 
     await deactivateAccount()
       .then(() => {
+        // The session has ended: the sign-in page takes over (AuthenticationWrapper), and shows this.
         setToast({
           type: TOAST_TYPE.SUCCESS,
-          title: "Success!",
-          message: "Account deactivated successfully.",
+          title: t("toast.success"),
+          message: t("account_deactivated"),
         });
         handleClose();
         return;
       })
       .catch((error: unknown) => {
-        // An ApiError's message is its problem's detail or title, else the HTTP status; the other failures (the
-        // session changed or is unavailable, no network) are errors with a message as well.
         setToast({
           type: TOAST_TYPE.ERROR,
-          title: "Error!",
-          message: error instanceof Error ? error.message : undefined,
+          title: t("toast.error"),
+          message: t(errorMessageKey(error)),
         });
       })
       .finally(() => setIsDeactivating(false));
```

`web/apps/web/core/components/instance/maintenance-message.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";

/**
 * What the page says when it cannot load the instance's settings (InstanceWrapper): nerve did not answer, or
 * answered with a failure, maybe only for a moment. The request is tried again by itself (SWR), and the page
 * carries on once it succeeds.
 */
export function MaintenanceMessage() {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-2.5">
      <h1 className="text-left text-18 font-semibold text-primary">{t("maintenance.title")}</h1>
      <span className="text-left text-14 font-medium text-secondary">{t("maintenance.description")}</span>
    </div>
  );
}
```

`web/packages/i18n/src/locales/en/common.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/en/common.json
+++ b/web/packages/i18n/src/locales/en/common.json
@@ -26,7 +26,7 @@
   "saving": "Saving",
   "save_changes": "Save changes",
   "deactivate_account": "Deactivate account",
-  "deactivate_account_description": "When deactivating an account, all of the data and resources within that account will be permanently removed and cannot be recovered.",
+  "deactivate_account_description": "Signs you out everywhere; your password and personal access tokens stop working until an administrator of this server reactivates the account. Nothing is deleted.",
   "preferences": "Preferences",
   "language_and_time": "Language & Time",
   "create_workspace": "Create workspace",
@@ -123,7 +123,7 @@
   "you_can_see_here_if_someone_invites_you_to_a_workspace": "You can see here if someone invites you to a workspace",
   "back_to_home": "Back to home",
   "deactivate_your_account": "Deactivate your account",
-  "deactivate_your_account_description": "Once deactivated, you can't be assigned work items. To reactivate your account, you will need an invite to a workspace at this email address.",
+  "deactivate_your_account_description": "You will be signed out everywhere, and neither your password nor your personal access tokens will work. Nothing is deleted and no email is sent: to use the account again, ask an administrator of this server to reactivate it.",
   "deactivating": "Deactivating",
   "confirm": "Confirm",
   "draft_created": "Draft created",
@@ -467,5 +467,11 @@
   "accordion_navigation_control": "Accordion sidebar navigation",
   "horizontal_navigation_bar": "Tabbed Navigation",
   "show_limited_projects_on_sidebar": "Show limited projects on sidebar",
-  "enter_number_of_projects": "Enter number of projects"
+  "enter_number_of_projects": "Enter number of projects",
+  "profile_updated": "Your profile is updated.",
+  "account_deactivated": "Your account is deactivated.",
+  "maintenance": {
+    "title": "Nerve cannot be reached right now",
+    "description": "The page could not load this server's settings. It tries again by itself and carries on once the server answers."
+  }
 }
```

`web/packages/i18n/src/locales/zh-CN/common.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/zh-CN/common.json
+++ b/web/packages/i18n/src/locales/zh-CN/common.json
@@ -26,7 +26,7 @@
   "saving": "保存中",
   "save_changes": "保存更改",
   "deactivate_account": "停用账号",
-  "deactivate_account_description": "停用账号后，该账号内的所有数据和资源将被永久删除且无法恢复。",
+  "deactivate_account_description": "在所有地方退出登录；在这台服务器的管理员重新启用账户之前，密码和个人访问令牌都无法使用。数据不会被删除。",
   "preferences": "偏好设置",
   "language_and_time": "语言和时间",
   "create_workspace": "创建工作区",
@@ -123,7 +123,7 @@
   "you_can_see_here_if_someone_invites_you_to_a_workspace": "如果有人邀请您加入工作区，您可以在这里看到",
   "back_to_home": "返回首页",
   "deactivate_your_account": "停用您的账户",
-  "deactivate_your_account_description": "一旦停用，您将无法被分配工作项。要重新激活您的账户，您需要收到发送到此电子邮件地址的工作区邀请。",
+  "deactivate_your_account_description": "您将在所有地方退出登录，密码和个人访问令牌都将无法使用。数据不会被删除，也不会发送邮件：如需再次使用此账户，请联系这台服务器的管理员重新启用。",
   "deactivating": "正在停用",
   "confirm": "确认",
   "draft_created": "草稿已创建",
@@ -467,5 +467,11 @@
   "accordion_navigation_control": "折叠侧边栏导航",
   "horizontal_navigation_bar": "标签式导航",
   "show_limited_projects_on_sidebar": "在侧边栏显示有限数量的项目",
-  "enter_number_of_projects": "输入项目数量"
+  "enter_number_of_projects": "输入项目数量",
+  "profile_updated": "个人资料已更新。",
+  "account_deactivated": "您的账户已停用。",
+  "maintenance": {
+    "title": "暂时无法连接 Nerve",
+    "description": "页面没能读取这台服务器的设置。它会自动重试，服务器恢复应答后继续。"
+  }
 }
```

- [ ] **Step 3: 故事**

`e2e/stories/identity/a12-deactivate.spec.ts`（完整内容）：

```ts
import { randomUUID } from "node:crypto";

import type { Api } from "../../fixtures/api";
import { accountOf, accountStateOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { formAlert, signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { bearer, createPAT, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";

// A12, deactivating an account (M2 design 2, decision 3), with the
// administrator's nerve users activate and deactivate.

/** Finishes onboarding, so that starting it over shows. */
async function onboard(api: Api, token: string): Promise<void> {
  const { response } = await api.PATCH("/api/v0/me/profile", {
    body: {
      onboarding_step: { profile_complete: true, workspace_create: true, workspace_invite: true, workspace_join: true },
      is_onboarded: true,
      is_tour_completed: true,
      last_workspace_id: randomUUID(),
    },
    headers: bearer(token),
  });
  expect(response.status).toBe(200);
}

test("A12 (page): the general page deactivates the account once confirmed; the session ends, sign-in says so, and nerve users activate lets it in again", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const pat = await createPAT(api, tokens.access_token);
  // Another session of the account, which ends too.
  await login(api, email);
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // The confirmation says what deactivating does, as nerve does it (decision 3).
  await page.getByRole("button", { name: "Deactivate account" }).click();
  await expect(page.getByText("ask an administrator of this server to reactivate it", { exact: false })).toBeVisible();
  const deactivated = await answerTo(page, "POST", "/api/v0/me/deactivate", () =>
    page.getByRole("button", { name: "Confirm" }).click()
  );
  expect(deactivated.status()).toBe(204);

  // The page is back at sign-in, which comes back to the general page, and the browser keeps no session.
  await expect(page).toHaveURL(signInPath("/settings/profile/general"));
  await expect(page.getByText("Your account is deactivated.")).toBeVisible();
  expect(await recordOf(page)).toBeNull();
  await expectDeactivated(db, before, tokensBefore);

  // Signing in again: the account is deactivated.
  expect(await submitSignIn(page, email, password)).toBe(403);
  await expect(formAlert(page)).toHaveText("This account is deactivated.");

  // The administrator activates it: it signs in, into onboarding, which deactivating started over.
  expect(await nerveUsers(db, ["activate", "--email", email])).toBe(
    `activated ${email}: 1 API tokens are usable again\n`
  );
  expect(await submitSignIn(page, email, password)).toBe(200);
  await expect(page).toHaveURL("/onboarding");
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);

  expect(watch.apiFailures).toEqual(["403 POST /api/v0/auth/login"]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
    warnings: [EMOJI_CHECK_WARNING],
  });
});

test("A12 (API): a token deactivates the account; nerve users activate brings it and its tokens back, deactivate does as the API did", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  await onboard(api, pat.token);
  // Another account, which nothing here changes.
  const other = emailFor(testInfo, "other");
  await register(api, other);
  const otherBefore = await accountStateOf(db, other);
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  const deactivated = await api.POST("/api/v0/me/deactivate", { headers: bearer(pat.token) });
  expect(deactivated.response.status).toBe(204);
  await expectDeactivated(db, before, tokensBefore);
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);

  // Nothing authenticates as the account: the token fails, the password is refused.
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(401);
  const refused = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("identity.account_deactivated");

  // The administrator activates it: the same token and password work again.
  expect(await nerveUsers(db, ["activate", "--email", email])).toBe(
    `activated ${email}: 1 API tokens are usable again\n`
  );
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  await login(api, email);

  // The administrator's deactivate leaves the database as the API did.
  await onboard(api, pat.token);
  const again = await accountOf(db, email);
  expect(await nerveUsers(db, ["deactivate", "--email", email])).toBe(`deactivated ${email}: revoked 1 sessions\n`);
  await expectDeactivated(db, again, tokensBefore);
  expect(await accountStateOf(db, other), "the other account").toEqual(otherBefore);
});
```

- [ ] **Step 4: 检查**

Run: `grep -rn "&lt;\|&gt;\|&amp;" web/apps/web/core/components/settings/profile/content/pages/general/form.tsx web/apps/web/core/components/account web/apps/web/core/components/instance`
Expected: 没有输出。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

Run: `make e2e`
Expected: 全部通过。

- [ ] **Step 5: 提交**

```bash
git add web/apps/web/core/components web/packages/i18n e2e
```
```bash
git commit -m "feat(M2/P5): general's save and deactivation say what nerve does; maintenance copy

The general page shows a refusal under the field it is about, anything
else in a toast, from the problem's code. The deactivation modal says
what nerve does: signed out everywhere, password and tokens stop,
nothing deleted, an administrator reactivates. The maintenance page no
longer implies nerve failed to start: it could not be reached, and the
page carries on once it answers. A12's page version.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过。

---

### Task 8: 死成员和死 prop（`domains.mjs --rows M2`）

**Files:**
- Modify: `web/apps/web/core/store/user/settings.store.ts`、`permissions.store.ts`
- Modify: `web/apps/web/core/store/issue/profile/filter.store.ts`、`profile/issue.store.ts`、`helpers/issue-filter-helper.store.ts`，以及 `web/apps/web/core/store/issue/{archived,cycle,module,project-views,project,workspace-draft,workspace}/filter.store.ts`
- Modify: `web/packages/constants/src/settings/profile.ts`
- Modify: `web/apps/web/core/components/auth-screens/not-authorized-view.tsx`、`profile/sidebar.tsx`、`settings/profile/heading.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`workspace-wrapper.tsx`
- Modify（过渡）: `web/apps/web/package.json`（oxlint 上限 541 → 537）

**Interfaces:**（spec 2.9，M2 设计 7.8；M1 收尾交接）`domains.mjs --rows M2` 在本 phase 开始时 64 行（M1 收尾时 66 行）：22 行测试替身、1 行 M3 的 prop、41 行死代码。Task 2、3 随改写删掉 15 行（工作区的旧 PAT store 12 行、`@nerve/services` 的 axios 基类 3 行），本 Task 删掉其余 26 行；Task 2 给假 nerve 加的 `query` 多出 1 行测试替身。剩 24 行，都不是死代码（spec 附录 A.4 逐行说明：23 行是测试替身和测试里的字面量，1 行是 M3 加回取数时传入的 prop）。本 Task 删除的：
- `IUserSettingsStore` 的 `isLoading`、`error`（及 `TError`）、`isScrolled`、`toggleIsScrolled`；`fetchCurrentUserSettings` 失败时直接抛出，返回类型不再含 `undefined`；
- `IUserPermissionStore` 接口上的 `fetchWorkspaceLevelProjectEntities`（实现保留，只有这个 store 自己调用它）；
- `issue/profile` 的 `userId`、`issueFilterService`、接口上的 `currentView`、`quickAddIssue`；
- `IBaseIssueFilterStore.appliedFilters` 和实现它的八个过滤 store 的 `appliedFilters` getter：M2 的一行（profile 的）只能连同接口一起删，没有任何读者；
- 个人设置的两张表各写一遍的成员类型合成 `TProfileSettingsTab`；
- 没人传的 prop：`NotAuthorizedView` 的 `actionButton`，`ProfileSidebar`、`ProfileSettingsHeading` 的 `className`，`ProjectAuthWrapper`、`WorkspaceAuthWrapper` 的 `isLoading`。
- 同一批文件里的 4 条 oxlint 警告：`ProfileSidebar` 的窗口大小监听只注册一次，改为经 ref 读当前的折叠状态（`react-hooks/exhaustive-deps`）；`WorkspaceAuthWrapper` 的"退出"改为 `<button type="button" aria-label="Sign out">`（两条 `jsx_a11y`）；`workspace/filter.store.ts` 内层的 `_filters` 改名 `view`（`no-shadow`）。

**Tests:** 行为不变；现有的单元测试和 `make e2e` 覆盖。

- [ ] **Step 1: 用户的 store**

`web/apps/web/core/store/user/settings.store.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient } from "@nerve/api-client";
import type { IUserSettings } from "@nerve/types";
// services
import { UserService } from "@/services/user.service";

export interface IUserSettingsStore {
  // observables
  data: IUserSettings;
  sidebarCollapsed: boolean;
  // actions
  fetchCurrentUserSettings: (bustCache?: boolean) => Promise<IUserSettings>;
  toggleSidebar: (collapsed?: boolean) => void;
}

export class UserSettingsStore implements IUserSettingsStore {
  // observables
  sidebarCollapsed: boolean = true;
  data: IUserSettings = {
    id: undefined,
    email: undefined,
    workspace: {
      last_workspace_id: undefined,
      last_workspace_slug: undefined,
      last_workspace_name: undefined,
      last_workspace_logo: undefined,
      fallback_workspace_id: undefined,
      fallback_workspace_slug: undefined,
      invites: undefined,
    },
  };
  // services
  userService: UserService;

  constructor(api: ApiClient) {
    makeObservable(this, {
      // observables
      data: observable,
      sidebarCollapsed: observable.ref,
      // actions
      fetchCurrentUserSettings: action,
      toggleSidebar: action,
    });
    // services
    this.userService = new UserService(api);
  }

  // actions
  toggleSidebar = (collapsed?: boolean) => {
    this.sidebarCollapsed = collapsed ?? !this.sidebarCollapsed;
  };

  // actions
  /**
   * @description fetches user profile information
   * @returns {Promise<IUserSettings>}
   */
  fetchCurrentUserSettings = async (bustCache: boolean = false) => {
    const userSettings = await this.userService.currentUserSettings(bustCache);
    runInAction(() => {
      this.data = userSettings;
    });
    return userSettings;
  };
}
```

`web/apps/web/core/store/user/permissions.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/user/permissions.store.ts
+++ b/web/apps/web/core/store/user/permissions.store.ts
@@ -38,7 +38,6 @@
     workspaceSlug: string,
     projectId?: string
   ) => EUserPermissions | undefined;
-  fetchWorkspaceLevelProjectEntities: (workspaceSlug: string, projectId: string) => void;
   allowPermissions: (
     allowPermissions: ETempUserRole[],
     level: TUserPermissionsLevel,
```

- [ ] **Step 2: 工作项的 store**

`web/apps/web/core/store/issue/profile/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/profile/filter.store.ts
+++ b/web/apps/web/core/store/issue/profile/filter.store.ts
@@ -22,15 +22,12 @@
 } from "@nerve/types";
 import { EIssuesStoreType } from "@nerve/types";
 import { handleIssueQueryParamsByLayout } from "@nerve/utils";
-import { IssueFiltersService } from "@/services/issue_filter.service";
 import type { IBaseIssueFilterStore } from "../helpers/issue-filter-helper.store";
 import { IssueFilterHelperStore } from "../helpers/issue-filter-helper.store";
 // types
 import type { IIssueRootStore } from "../root.store";
 
 export interface IProfileIssuesFilter extends IBaseIssueFilterStore {
-  // observables
-  userId: string;
   //helper actions
   getFilterParams: (
     options: IssuePaginationOptions,
@@ -53,30 +50,23 @@
 
 export class ProfileIssuesFilter extends IssueFilterHelperStore implements IProfileIssuesFilter {
   // observables
-  userId: string = "";
   filters: { [userId: string]: IIssueFilters } = {};
   // root store
   rootIssueStore: IIssueRootStore;
-  // services
-  issueFilterService;
 
   constructor(_rootStore: IIssueRootStore) {
     super();
     makeObservable(this, {
       // observables
-      userId: observable.ref,
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilters: action,
     });
     // root store
     this.rootIssueStore = _rootStore;
-    // services
-    this.issueFilterService = new IssueFiltersService();
   }
 
   get issueFilters() {
@@ -86,13 +76,6 @@
     return this.getIssueFilters(userId);
   }
 
-  get appliedFilters() {
-    const userId = this.rootIssueStore.userId;
-    if (!userId) return undefined;
-
-    return this.getAppliedFilters(userId);
-  }
-
   getIssueFilters(userId: string) {
     const displayFilters = this.filters[userId] || undefined;
     if (isEmpty(displayFilters)) return undefined;
@@ -134,7 +117,6 @@
   );
 
   fetchFilters = async (workspaceSlug: string, userId: string) => {
-    this.userId = userId;
     const _filters = this.handleIssuesLocalFilters.get(EIssuesStoreType.PROFILE, workspaceSlug, userId, undefined);
 
     const richFilters: TWorkItemFilterExpression = _filters?.rich_filters;
```

`web/apps/web/core/store/issue/profile/issue.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/profile/issue.store.ts
+++ b/web/apps/web/core/store/issue/profile/issue.store.ts
@@ -18,7 +18,6 @@
 
 export interface IProfileIssues extends IBaseIssuesStore {
   // observable
-  currentView: TProfileViews;
   viewFlags: ViewFlags;
   // actions
   setViewId: (viewId: TProfileViews) => void;
@@ -46,8 +45,6 @@
   createIssue: (workspaceSlug: string, projectId: string, data: Partial<TIssue>) => Promise<TIssue>;
   updateIssue: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => Promise<void>;
   archiveIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
-
-  quickAddIssue: undefined;
 }
 
 export class ProfileIssues extends BaseIssuesStore implements IProfileIssues {
@@ -219,7 +216,4 @@
   // Using aliased names as they cannot be overridden in other stores
   updateIssue = this.issueUpdate;
   archiveIssue = this.issueArchive;
-
-  // Setting them as undefined as they can not performed on profile issues
-  quickAddIssue = undefined;
 }
```

`web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts
+++ b/web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts
@@ -38,7 +38,6 @@
   // observables
   filters: Record<string, IIssueFilters>;
   //computed
-  appliedFilters: Partial<Record<TIssueParams, string | boolean>> | undefined;
   issueFilters: IIssueFilters | undefined;
 }
 
```

`web/apps/web/core/store/issue/archived/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/archived/filter.store.ts
+++ b/web/apps/web/core/store/issue/archived/filter.store.ts
@@ -68,7 +68,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilters: action,
@@ -84,13 +83,6 @@
     if (!projectId) return undefined;
 
     return this.getIssueFilters(projectId);
-  }
-
-  get appliedFilters() {
-    const projectId = this.rootIssueStore.projectId;
-    if (!projectId) return undefined;
-
-    return this.getAppliedFilters(projectId);
   }
 
   getIssueFilters(projectId: string) {
```

`web/apps/web/core/store/issue/cycle/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/cycle/filter.store.ts
+++ b/web/apps/web/core/store/issue/cycle/filter.store.ts
@@ -70,7 +70,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilters: action,
@@ -86,13 +85,6 @@
     if (!cycleId) return undefined;
 
     return this.getIssueFilters(cycleId);
-  }
-
-  get appliedFilters() {
-    const cycleId = this.rootIssueStore.cycleId;
-    if (!cycleId) return undefined;
-
-    return this.getAppliedFilters(cycleId);
   }
 
   getIssueFilters(cycleId: string) {
```

`web/apps/web/core/store/issue/module/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/module/filter.store.ts
+++ b/web/apps/web/core/store/issue/module/filter.store.ts
@@ -70,7 +70,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilters: action,
@@ -86,13 +85,6 @@
     if (!moduleId) return undefined;
 
     return this.getIssueFilters(moduleId);
-  }
-
-  get appliedFilters() {
-    const moduleId = this.rootIssueStore.moduleId;
-    if (!moduleId) return undefined;
-
-    return this.getAppliedFilters(moduleId);
   }
 
   getIssueFilters(moduleId: string) {
```

`web/apps/web/core/store/issue/project-views/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/project-views/filter.store.ts
+++ b/web/apps/web/core/store/issue/project-views/filter.store.ts
@@ -75,7 +75,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilters: action,
@@ -92,13 +91,6 @@
     if (!viewId) return undefined;
 
     return this.getIssueFilters(viewId);
-  }
-
-  get appliedFilters() {
-    const viewId = this.rootIssueStore.viewId;
-    if (!viewId) return undefined;
-
-    return this.getAppliedFilters(viewId);
   }
 
   getIssueFilters(viewId: string) {
```

`web/apps/web/core/store/issue/project/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/project/filter.store.ts
+++ b/web/apps/web/core/store/issue/project/filter.store.ts
@@ -68,7 +68,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilterExpression: action,
@@ -85,13 +84,6 @@
     if (!projectId) return undefined;
 
     return this.getIssueFilters(projectId);
-  }
-
-  get appliedFilters() {
-    const projectId = this.rootIssueStore.projectId;
-    if (!projectId) return undefined;
-
-    return this.getAppliedFilters(projectId);
   }
 
   getIssueFilters(projectId: string) {
```

`web/apps/web/core/store/issue/workspace-draft/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/workspace-draft/filter.store.ts
+++ b/web/apps/web/core/store/issue/workspace-draft/filter.store.ts
@@ -68,7 +68,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // actions
       fetchFilters: action,
       updateFilters: action,
@@ -84,13 +83,6 @@
     if (!workspaceSlug) return undefined;
 
     return this.getIssueFilters(workspaceSlug);
-  }
-
-  get appliedFilters() {
-    const workspaceSlug = this.rootIssueStore.workspaceSlug;
-    if (!workspaceSlug) return undefined;
-
-    return this.getAppliedFilters(workspaceSlug);
   }
 
   getIssueFilters(workspaceSlug: string) {
```

`web/apps/web/core/store/issue/workspace/filter.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/issue/workspace/filter.store.ts
+++ b/web/apps/web/core/store/issue/workspace/filter.store.ts
@@ -72,7 +72,6 @@
       filters: observable,
       // computed
       issueFilters: computed,
-      appliedFilters: computed,
       // fetch actions
       fetchFilters: action,
       updateFilters: action,
@@ -117,11 +116,6 @@
     return this.getIssueFilters(viewId);
   }
 
-  get appliedFilters() {
-    const viewId = this.rootIssueStore.globalViewId;
-    return this.getAppliedFilters(viewId);
-  }
-
   getFilterParams = computedFn(
     (
       options: IssuePaginationOptions,
@@ -171,13 +165,13 @@
 
     // Get the view details if the view is not a static view
     if (STATIC_VIEW_TYPES.includes(viewId) === false) {
-      const _filters = await this.issueFilterService.getViewDetails(workspaceSlug, viewId);
-      richFilters = _filters?.rich_filters;
-      displayFilters = this.computedDisplayFilters(_filters?.display_filters, {
+      const view = await this.issueFilterService.getViewDetails(workspaceSlug, viewId);
+      richFilters = view?.rich_filters;
+      displayFilters = this.computedDisplayFilters(view?.display_filters, {
         layout: EIssueLayoutTypes.SPREADSHEET,
         order_by: "-created_at",
       });
-      displayProperties = this.computedDisplayProperties(_filters?.display_properties);
+      displayProperties = this.computedDisplayProperties(view?.display_properties);
     }
 
     // override existing order by if ordered by manual sort_order
```

- [ ] **Step 3: 常量和组件**

`web/packages/constants/src/settings/profile.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/constants/src/settings/profile.ts
+++ b/web/packages/constants/src/settings/profile.ts
@@ -22,13 +22,13 @@
   [PROFILE_SETTINGS_CATEGORY.DEVELOPER]: "common.developer",
 };
 
-const PROFILE_SETTINGS: Record<
-  TProfileSettingsTabs,
-  {
-    key: TProfileSettingsTabs;
-    i18n_label: string;
-  }
-> = {
+/** A tab of the profile settings, as the sidebar lists it. */
+type TProfileSettingsTab = {
+  key: TProfileSettingsTabs;
+  i18n_label: string;
+};
+
+const PROFILE_SETTINGS: Record<TProfileSettingsTabs, TProfileSettingsTab> = {
   general: {
     key: "general",
     i18n_label: "profile.actions.profile",
@@ -49,10 +49,7 @@
 
 export const PROFILE_SETTINGS_TABS: TProfileSettingsTabs[] = Object.keys(PROFILE_SETTINGS) as TProfileSettingsTabs[];
 
-export const GROUPED_PROFILE_SETTINGS: Record<
-  PROFILE_SETTINGS_CATEGORY,
-  { key: TProfileSettingsTabs; i18n_label: string }[]
-> = {
+export const GROUPED_PROFILE_SETTINGS: Record<PROFILE_SETTINGS_CATEGORY, TProfileSettingsTab[]> = {
   [PROFILE_SETTINGS_CATEGORY.YOUR_PROFILE]: [
     PROFILE_SETTINGS["general"],
     PROFILE_SETTINGS["preferences"],
```

`web/apps/web/core/components/auth-screens/not-authorized-view.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// assets
import { cn } from "@nerve/utils";
import ProjectNotAuthorizedImg from "@/app/assets/auth/project-not-authorized.svg?url";
import Unauthorized from "@/app/assets/auth/unauthorized.svg?url";
import WorkspaceNotAuthorizedImg from "@/app/assets/auth/workspace-not-authorized.svg?url";
// layouts
import DefaultLayout from "@/layouts/default-layout";

type Props = {
  section?: "settings" | "general";
  isProjectView?: boolean;
  className?: string;
};

export const NotAuthorizedView = observer(function NotAuthorizedView(props: Props) {
  const { section = "general", isProjectView = false, className } = props;

  // assets
  const settingAsset = isProjectView ? ProjectNotAuthorizedImg : WorkspaceNotAuthorizedImg;
  const asset = section === "settings" ? settingAsset : Unauthorized;

  return (
    <DefaultLayout className={cn("bg-surface-1", className)}>
      <div className="flex h-full w-full flex-col items-center justify-center gap-y-5 text-center">
        <div className="h-44 w-72">
          <img src={asset} className="h-[176px] w-[288px] object-contain" alt="" />
        </div>
        <h1 className="text-18 font-medium text-primary">Oops! You are not authorized to view this page</h1>
      </div>
    </DefaultLayout>
  );
});
```

`web/apps/web/core/components/profile/sidebar.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/profile/sidebar.tsx
+++ b/web/apps/web/core/components/profile/sidebar.tsx
@@ -13,7 +13,7 @@
 import { IconButton } from "@nerve/propel/icon-button";
 import { EditOutline } from "@makeplane/propel/icons";
 import { Loader } from "@nerve/ui";
-import { cn, renderFormattedDate, getFileURL } from "@nerve/utils";
+import { renderFormattedDate, getFileURL } from "@nerve/utils";
 // hooks
 import { useAppTheme } from "@/hooks/store/use-app-theme";
 import { useCommandPalette } from "@/hooks/store/use-command-palette";
@@ -21,12 +21,7 @@
 // local imports
 import { useProfileMember } from "./use-profile-member";
 
-type TProfileSidebar = {
-  className?: string;
-};
-
-export const ProfileSidebar = observer(function ProfileSidebar(props: TProfileSidebar) {
-  const { className = "" } = props;
+export const ProfileSidebar = observer(function ProfileSidebar() {
   // refs
   const ref = useRef<HTMLDivElement>(null);
   // router
@@ -34,6 +29,9 @@
   // store hooks
   const { data: currentUser } = useUser();
   const { profileSidebarCollapsed, toggleProfileSidebar } = useAppTheme();
+  // the state now, for the resize listener, which is added once
+  const collapsed = useRef(profileSidebarCollapsed);
+  collapsed.current = profileSidebarCollapsed;
   const { toggleProfileSettingsModal } = useCommandPalette();
   const profileMember = useProfileMember(workspaceSlug ?? "", userId ?? "");
   const { t } = useTranslation();
@@ -51,7 +49,7 @@
       if (window && window.innerWidth < 768) {
         toggleProfileSidebar(true);
       }
-      if (window && profileSidebarCollapsed && window.innerWidth >= 768) {
+      if (window && collapsed.current && window.innerWidth >= 768) {
         toggleProfileSidebar(false);
       }
     };
@@ -59,7 +57,7 @@
     window.addEventListener("resize", handleToggleProfileSidebar);
     handleToggleProfileSidebar();
     return () => window.removeEventListener("resize", handleToggleProfileSidebar);
-  }, []);
+  }, [toggleProfileSidebar]);
 
   const renderContent = () => {
     if (profileMember.status === "loading")
@@ -125,10 +123,7 @@
   return (
     <div
       ref={ref}
-      className={cn(
-        `vertical-scrollbar fixed z-5 scrollbar-md h-full w-full shrink-0 overflow-hidden overflow-y-auto border-l border-subtle bg-surface-1 shadow-raised-200 transition-all md:relative md:w-[300px]`,
-        className
-      )}
+      className="vertical-scrollbar fixed z-5 scrollbar-md h-full w-full shrink-0 overflow-hidden overflow-y-auto border-l border-subtle bg-surface-1 shadow-raised-200 transition-all md:relative md:w-[300px]"
       style={profileSidebarCollapsed ? { marginLeft: `${window?.innerWidth || 0}px` } : {}}
     >
       {renderContent()}
```

`web/apps/web/core/components/settings/profile/heading.tsx`（完整内容）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

type Props = {
  control?: React.ReactNode;
  description?: React.ReactNode;
  title?: React.ReactNode;
};

export function ProfileSettingsHeading({ control, description, title }: Props) {
  return (
    <div className="flex flex-col items-start justify-between gap-4 md:flex-row md:items-center">
      <div className="flex flex-col items-start gap-1">
        {title && <h6 className="text-h6-medium text-primary">{title}</h6>}
        {description && <p className="text-body-xs-regular text-tertiary">{description}</p>}
      </div>
      {control}
    </div>
  );
}
```

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
+++ b/web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
@@ -38,11 +38,10 @@
   workspaceSlug: string;
   projectId: string;
   children: ReactNode;
-  isLoading?: boolean;
 }
 
 export const ProjectAuthWrapper = observer(function ProjectAuthWrapper(props: IProjectAuthWrapper) {
-  const { workspaceSlug, projectId, children, isLoading: isParentLoading = false } = props;
+  const { workspaceSlug, projectId, children } = props;
   // states
   const [isJoiningProject, setIsJoiningProject] = useState(false);
   // store hooks
@@ -125,7 +124,7 @@
     joinProject(workspaceSlug, projectId).finally(() => setIsJoiningProject(false));
   };
 
-  const isProjectLoading = (isParentLoading || isProjectDetailsLoading) && !projectDetailsError;
+  const isProjectLoading = isProjectDetailsLoading && !projectDetailsError;
 
   if (isProjectLoading) return null;
 
```

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
+++ b/web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
@@ -41,11 +41,10 @@
 
 interface IWorkspaceAuthWrapper {
   children: ReactNode;
-  isLoading?: boolean;
 }
 
 export const WorkspaceAuthWrapper = observer(function WorkspaceAuthWrapper(props: IWorkspaceAuthWrapper) {
-  const { children, isLoading: isParentLoading = false } = props;
+  const { children } = props;
   // router params
   const { workspaceSlug } = useParams();
   // store hooks
@@ -125,7 +124,7 @@
   };
 
   // if list of workspaces are not there then we have to render the spinner
-  if (isParentLoading || allWorkspaces === undefined || loader) {
+  if (allWorkspaces === undefined || loader) {
     return (
       <div className="grid h-full place-items-center rounded-lg border border-subtle p-4">
         <div className="flex flex-col items-center gap-3 text-center">
@@ -146,14 +145,16 @@
             </div>
             <div className="relative flex items-center gap-2">
               <div className="text-13 font-medium">{currentUser?.email}</div>
-              <div
+              <button
+                type="button"
+                aria-label="Sign out"
                 className="relative flex h-6 w-6 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm hover:bg-layer-1"
                 onClick={handleSignOut}
               >
                 <Tooltip label="Sign out" alignOffset={8} disabled={isMobile}>
                   <LogOutOutline width={14} height={14} />
                 </Tooltip>
-              </div>
+              </button>
             </div>
           </div>
           <div className="relative flex h-full w-full flex-grow flex-col items-center justify-center space-y-3">
```

- [ ] **Step 4: oxlint 上限**

`web/apps/web/package.json`（对 Task 6 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 541",
+    "check:lint": "node ../../../tools/lint-cap.mjs 537",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 5: 检查**

Run: `grep -rnw "get appliedFilters\|appliedFilters: computed\|appliedFilters: Partial\|toggleIsScrolled\|actionButton" web/apps/web/core web/apps/web/app`
Expected: 没有输出。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（web 的 oxlint 上限 537）。

Run: M1 收尾交接写的 `domains.mjs … --rows M2`（M1 收尾 plan 附录 A 的 `deadsym.mjs`、`domains.mjs`，不进仓库）
Expected: 24 行，与 spec 附录 A.4 的清单相同。

- [ ] **Step 6: 提交**

```bash
git add web/apps/web/core web/packages/constants web/apps/web/package.json
```
```bash
git commit -m "refactor(M2/P5): delete M2's dead members and props (M1-closeout handoff)

UserSettingsStore's isLoading, error and isScrolled; the profile issue
stores' userId, filter service, currentView and quickAddIssue;
appliedFilters, which nothing read, from the base filter store and the
eight filter stores; the props nobody passes (NotAuthorizedView's
actionButton, the profile sidebar's and heading's className, the auth
wrappers' isLoading). The 24 rows left are test doubles and a prop M3
passes again.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 检查都通过；`domains.mjs --rows M2` 剩 24 行，与 spec 附录 A.4 相同。

---

### Task 9: oxlint：P4、P5 改到的文件清零

**Files:**
- Modify: `web/apps/web/app/(all)/invitations/page.tsx`
- Modify: `web/apps/web/core/components/core/image-picker-popover.tsx`、`issues/issue-detail/reactions/issue.tsx`、`onboarding/steps/profile/root.tsx`、`onboarding/switch-account-modal.tsx`、`project/form.tsx`、`project/settings/member-columns.tsx`、`workspace/settings/member-columns.tsx`
- Modify: `web/packages/utils/src/auth.ts`、`web/packages/utils/package.json`（oxlint 上限 18 → 16）
- Modify（过渡）: `web/apps/web/package.json`（oxlint 上限 537 → 514）

**Interfaces:**（spec 2.10，M2 设计 7.8；P4 spec 第 6 节）P4 留下警告的文件和 P5 前面几个 Task 没有顺带清掉的文件，共 25 条：
- `promise/always-return`：`.then` 链改为 `async`/`await` 或返回里面的 promise（邀请页、`switch-account-modal`、`project/form`）；
- `jsx_a11y` 的 `click-events-have-key-events`、`no-static-element-interactions`、`prefer-tag-over-role`：可点击的 `div` 改为 `<button type="button">`（邀请页的工作区卡片加 `aria-pressed`；封面图片；工作区成员的移除），项目成员的移除把 `onClick` 放到 `CustomMenu.MenuItem` 上；`img-redundant-alt`：替代文本不再含"image"；
- `no-autofocus`：引导的资料步骤只有这一个字段，打开就聚焦它，用带理由的禁用注释；
- `react-hooks/exhaustive-deps`：表情回应的列表不再 `useMemo`（依赖里缺 `getReactionUsers`，而它每次渲染都变）；
- `no-constant-binary-expression`：`Number(x?.role) ?? GUEST` 的 `??` 永远不生效，改为 `Number(x?.role ?? GUEST)`（没有角色时按访客算，原来得到 `NaN`）；
- `no-shadow`、`no-empty-pattern`、`no-useless-escape`：改名、`() =>`、正则里的 `\[` 改为 `[`（字符类里等价，`auth.test.ts` 覆盖）。

**Tests:** 行为不变（`Number(… ?? GUEST)` 除外，见上）；现有的单元测试和 `make e2e` 覆盖。

- [ ] **Step 1: 邀请页、引导、切换账户**

`web/apps/web/app/(all)/invitations/page.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/app/(all)/invitations/page.tsx
+++ b/web/apps/web/app/(all)/invitations/page.tsx
@@ -77,17 +77,15 @@
     workspaceService
       .joinWorkspaces({ invitations: invitationsRespond })
       .then(() => {
-        mutate(USER_WORKSPACES_LIST);
+        void mutate(USER_WORKSPACES_LIST);
         const firstInviteId = invitationsRespond[0];
         const redirectWorkspace = invitations?.find((i) => i.id === firstInviteId)?.workspace;
         // the workspace opened last is a best-effort preference: the joined workspace opens whether nerve saves it or not
-        updateUserProfile({ last_workspace_id: redirectWorkspace?.id })
+        return updateUserProfile({ last_workspace_id: redirectWorkspace?.id })
           .catch(() => undefined)
           .then(() => {
             setIsJoiningWorkspaces(false);
-            fetchWorkspaces().then(() => {
-              navigate(`/${redirectWorkspace?.slug}`);
-            });
+            return fetchWorkspaces().then(() => navigate(`/${redirectWorkspace?.slug}`));
           });
       })
       .catch((_err) => {
@@ -126,9 +124,11 @@
                     const isSelected = invitationsRespond.includes(invitation.id);
 
                     return (
-                      <div
+                      <button
+                        type="button"
                         key={invitation.id}
-                        className={`flex cursor-pointer items-center gap-2 rounded-sm border px-3.5 py-5 ${
+                        aria-pressed={isSelected}
+                        className={`flex w-full cursor-pointer items-center gap-2 rounded-sm border px-3.5 py-5 text-left ${
                           isSelected ? "border-accent-strong" : "border-subtle hover:bg-layer-1"
                         }`}
                         onClick={() => handleInvitation(invitation, isSelected ? "withdraw" : "accepted")}
@@ -147,7 +147,7 @@
                         <span className={`flex-shrink-0 ${isSelected ? "text-accent-primary" : "text-secondary"}`}>
                           <TickCircleOutline className="h-5 w-5" />
                         </span>
-                      </div>
+                      </button>
                     );
                   })}
                 </div>
```

`web/apps/web/core/components/onboarding/steps/profile/root.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/steps/profile/root.tsx
+++ b/web/apps/web/core/components/onboarding/steps/profile/root.tsx
@@ -115,6 +115,7 @@
                 type="text"
                 value={value}
                 onChange={(e) => onChange(e.target.value)}
+                // oxlint-disable-next-line jsx_a11y/no-autofocus -- the step asks for this one field, which it opens on
                 autoFocus
                 className={cn(
                   "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary transition-all duration-200 placeholder:text-placeholder focus:border-transparent focus:ring-2 focus:ring-accent-strong focus:outline-none",
```

`web/apps/web/core/components/onboarding/switch-account-modal.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/switch-account-modal.tsx
+++ b/web/apps/web/core/components/onboarding/switch-account-modal.tsx
@@ -37,19 +37,19 @@
   const handleSwitchAccount = async () => {
     setSwitchingAccount(true);
 
-    await signOut()
-      .then(() => {
-        setTheme("system");
-        handleClose();
-      })
-      .catch(() =>
-        setToast({
-          type: TOAST_TYPE.ERROR,
-          title: "Error!",
-          message: "Failed to sign out. Please try again.",
-        })
-      )
-      .finally(() => setSwitchingAccount(false));
+    try {
+      await signOut();
+      setTheme("system");
+      handleClose();
+    } catch {
+      setToast({
+        type: TOAST_TYPE.ERROR,
+        title: "Error!",
+        message: "Failed to sign out. Please try again.",
+      });
+    } finally {
+      setSwitchingAccount(false);
+    }
   };
 
   return (
```

- [ ] **Step 2: 其余组件**

`web/apps/web/core/components/core/image-picker-popover.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/core/image-picker-popover.tsx
+++ b/web/apps/web/core/components/core/image-picker-popover.tsx
@@ -150,17 +150,18 @@
                     <TabsPanel value="images">
                       <div className="grid grid-cols-4 gap-4">
                         {Object.values(STATIC_COVER_IMAGES).map((imageUrl, index) => (
-                          <div
+                          <button
+                            type="button"
                             key={imageUrl}
                             className="relative col-span-2 aspect-video md:col-span-1"
                             onClick={() => handleStaticImageSelect(imageUrl)}
                           >
                             <img
                               src={imageUrl}
-                              alt={`Cover image ${index + 1}`}
+                              alt={`Cover ${index + 1}`}
                               className="absolute top-0 left-0 h-full w-full cursor-pointer rounded-sm object-cover transition-opacity hover:opacity-80"
                             />
-                          </div>
+                          </button>
                         ))}
                       </div>
                     </TabsPanel>
@@ -185,7 +186,7 @@
                               <>
                                 <img
                                   src={image ? URL.createObjectURL(image) : getCoverImageDisplayURL(value, "")}
-                                  alt="image"
+                                  alt="Cover preview"
                                   className="h-full w-full rounded-lg object-cover"
                                 />
                               </>
```

`web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
+++ b/web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
@@ -99,19 +99,17 @@
   };
 
   // Transform reactions data to Propel EmojiReactionType format
-  const reactions: EmojiReactionType[] = useMemo(() => {
-    if (!reactionIds) return [];
+  const reactions: EmojiReactionType[] = reactionIds
+    ? Object.keys(reactionIds)
+        .filter((reaction) => reactionIds[reaction]?.length > 0)
+        .map((reaction) => ({
+          emoji: stringToEmoji(reaction),
+          count: reactionIds[reaction].length,
+          reacted: userReactions.includes(reaction),
+          users: getReactionUsers(reaction),
+        }))
+    : [];
 
-    return Object.keys(reactionIds)
-      .filter((reaction) => reactionIds[reaction]?.length > 0)
-      .map((reaction) => ({
-        emoji: stringToEmoji(reaction),
-        count: reactionIds[reaction].length,
-        reacted: userReactions.includes(reaction),
-        users: getReactionUsers(reaction),
-      }));
-  }, [reactionIds, userReactions]);
-
   const handleReactionClick = (emoji: string) => {
     if (disabled) return;
     // Convert emoji back to decimal string format for the API
```

`web/apps/web/core/components/project/form.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project/form.tsx
+++ b/web/apps/web/core/components/project/form.tsx
@@ -93,13 +93,13 @@
   const handleUpdateChange = async (payload: Partial<IProject>) => {
     if (!workspaceSlug || !project) return;
     return updateProject(workspaceSlug, project.id, payload)
-      .then(() => {
+      .then(() =>
         setToast({
           type: TOAST_TYPE.SUCCESS,
           title: t("toast.success"),
           message: t("project_settings.general.toast.success"),
-        });
-      })
+        })
+      )
       .catch((err) => {
         try {
           // Handle the new error format where codes are nested in arrays under field names
@@ -187,14 +187,11 @@
       return;
     }
 
-    if (project.identifier !== formData.identifier)
-      await projectService
-        .checkProjectIdentifierAvailability(workspaceSlug, payload.identifier ?? "")
-        .then(async (res) => {
-          if (res.exists) setError("identifier", { message: t("common.identifier_already_exists") });
-          else await handleUpdateChange(payload);
-        });
-    else await handleUpdateChange(payload);
+    if (project.identifier !== formData.identifier) {
+      const res = await projectService.checkProjectIdentifierAvailability(workspaceSlug, payload.identifier ?? "");
+      if (res.exists) setError("identifier", { message: t("common.identifier_already_exists") });
+      else await handleUpdateChange(payload);
+    } else await handleUpdateChange(payload);
     setTimeout(() => {
       setIsLoading(false);
     }, 300);
@@ -435,8 +432,8 @@
                 <>
                   <TimezoneSelect
                     value={value}
-                    onChange={(value: string) => {
-                      onChange(value);
+                    onChange={(timezone: string) => {
+                      onChange(timezone);
                     }}
                     error={Boolean(errors.timezone)}
                     buttonClassName="!border-subtle !shadow-none font-medium rounded-md"
```

`web/apps/web/core/components/project/settings/member-columns.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project/settings/member-columns.tsx
+++ b/web/apps/web/core/components/project/settings/member-columns.tsx
@@ -46,7 +46,7 @@
 
   return (
     <Disclosure>
-      {({}) => (
+      {() => (
         <div className="group relative">
           <div className="flex w-72 items-center gap-2">
             <div className="flex flex-1 items-center gap-x-2 gap-y-2">
@@ -76,11 +76,8 @@
                 optionsClassName="p-1.5"
                 placement="bottom-end"
               >
-                <CustomMenu.MenuItem>
-                  <div
-                    className="flex cursor-pointer items-center gap-x-1 font-medium text-danger-primary"
-                    onClick={() => setRemoveMemberModal(rowData)}
-                  >
+                <CustomMenu.MenuItem onClick={() => setRemoveMemberModal(rowData)}>
+                  <div className="flex cursor-pointer items-center gap-x-1 font-medium text-danger-primary">
                     <CircleMinus className="size-3.5 flex-shrink-0" />
                     {rowData.member?.id === currentUser?.id ? "Leave " : "Remove "}
                   </div>
@@ -112,17 +109,17 @@
   const roleLabel = ROLE[rowData.original_role ?? EUserPermissions.GUEST];
   const isCurrentUser = currentUser?.id === rowData.member.id;
   const isRowDataWorkspaceAdmin = [EUserPermissions.ADMIN].includes(
-    Number(getWorkspaceMemberDetails(rowData.member.id)?.role) ?? EUserPermissions.GUEST
+    Number(getWorkspaceMemberDetails(rowData.member.id)?.role ?? EUserPermissions.GUEST)
   );
   const isCurrentUserWorkspaceAdmin = currentUser
     ? [EUserPermissions.ADMIN].includes(
-        Number(getWorkspaceMemberDetails(currentUser.id)?.role) ?? EUserPermissions.GUEST
+        Number(getWorkspaceMemberDetails(currentUser.id)?.role ?? EUserPermissions.GUEST)
       )
     : false;
   const currentProjectRole = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
 
   const isCurrentUserProjectAdmin = currentProjectRole
-    ? ![EUserPermissions.MEMBER, EUserPermissions.GUEST].includes(Number(currentProjectRole) ?? EUserPermissions.GUEST)
+    ? ![EUserPermissions.MEMBER, EUserPermissions.GUEST].includes(Number(currentProjectRole))
     : false;
 
   // logic
```

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/workspace/settings/member-columns.tsx
+++ b/web/apps/web/core/components/workspace/settings/member-columns.tsx
@@ -88,20 +88,13 @@
                 popoverClassName="justify-end"
                 buttonClassName="outline-none	origin-center rotate-90 size-8 aspect-square flex-shrink-0 grid place-items-center opacity-0 group-hover:opacity-100 transition-opacity"
                 render={() => (
-                  <div
-                    role="button"
-                    tabIndex={0}
+                  <button
+                    type="button"
                     className="flex cursor-pointer items-center gap-x-3"
                     onClick={() => setRemoveMemberModal(rowData)}
-                    onKeyDown={(e) => {
-                      if (e.key === "Enter" || e.key === " ") {
-                        e.preventDefault();
-                        setRemoveMemberModal(rowData);
-                      }
-                    }}
                   >
                     <DeleteOutline className="size-3.5 align-middle" /> {id === currentUser?.id ? "Leave " : "Remove "}
-                  </div>
+                  </button>
                 )}
               />
             )}
@@ -153,11 +146,11 @@
           render={({ field: { value } }) => (
             <CustomSelect
               value={value as EUserPermissions}
-              onChange={async (value: EUserPermissions) => {
+              onChange={async (role: EUserPermissions) => {
                 if (!workspaceSlug) return;
                 try {
                   await updateMember(workspaceSlug, rowData.member.id, {
-                    role: value as unknown as EUserPermissions,
+                    role: role as unknown as EUserPermissions,
                   });
                 } catch (err: unknown) {
                   const error = err as { error?: string | string[] };
```

- [ ] **Step 3: `@nerve/utils`**

`web/packages/utils/src/auth.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/utils/src/auth.ts
+++ b/web/packages/utils/src/auth.ts
@@ -30,7 +30,7 @@
   const hasUpperCase = /[A-Z]/.test(password);
   const hasLowerCase = /[a-z]/.test(password);
   const hasDigit = /[0-9]/.test(password);
-  const hasSpecialChar = /[!@#$%^&*()\-_+=\[\]{}|;:'",.<>?/]/.test(password);
+  const hasSpecialChar = /[!@#$%^&*()\-_+=[\]{}|;:'",.<>?/]/.test(password);
 
   if (hasUpperCase && hasLowerCase && hasDigit && hasSpecialChar) {
     return E_PASSWORD_STRENGTH.STRENGTH_VALID;
@@ -72,6 +72,6 @@
   {
     key: "special",
     label: "Min 1 special character",
-    isValid: /[!@#$%^&*()\-_+=\[\]{}|;:'",.<>?/]/.test(password),
+    isValid: /[!@#$%^&*()\-_+=[\]{}|;:'",.<>?/]/.test(password),
   },
 ];
```

`web/packages/utils/package.json`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/utils/package.json
+++ b/web/packages/utils/package.json
@@ -15,7 +15,7 @@
   "scripts": {
     "build": "tsdown",
     "dev": "tsdown --watch --no-clean",
-    "check:lint": "node ../../../tools/lint-cap.mjs 18",
+    "check:lint": "node ../../../tools/lint-cap.mjs 16",
     "check:types": "tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 4: oxlint 上限**

`web/apps/web/package.json`（对 Task 8 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 537",
+    "check:lint": "node ../../../tools/lint-cap.mjs 514",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 5: 检查**

Run: `grep -rn "&lt;\|&gt;\|&amp;" "web/apps/web/app/(all)/invitations/page.tsx" web/apps/web/core/components/core/image-picker-popover.tsx web/apps/web/core/components/project web/apps/web/core/components/workspace/settings/member-columns.tsx`
Expected: 没有输出。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（web 的 oxlint 上限 514，utils 16）。

Run: `pnpm -C web/apps/web exec oxlint --format unix <P4、P5 改到的每个 web 文件>`（清单见 spec 附录 A.3）
Expected: 没有警告。

- [ ] **Step 6: 提交**

```bash
git add "web/apps/web/app/(all)/invitations/page.tsx" web/apps/web/core/components web/packages/utils web/apps/web/package.json
```
```bash
git commit -m "refactor(M2/P5): no oxlint warning left in the files P4 and P5 touch

Promise chains become awaits or return their promises; clickable divs
become buttons; the onboarding profile step keeps its autofocus with a
reason; Number(role) ?? GUEST, whose fallback never applied, becomes
Number(role ?? GUEST); shadowed names, an empty pattern and useless
escapes go. The caps drop: web 514, utils 16.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 检查都通过。

---

### Task 10: `no-unneeded-ternary` 清零

**Files:**
- Modify: 下面 Step 1 的 44 个文件
- Modify: `web/apps/web/package.json`（oxlint 上限 514 → 457）、`web/packages/ui/package.json`（20 → 19）

**Interfaces:**（spec 2.10，M2 设计 7.8）54 条都是 `x ? true : false` 一类：条件本身是布尔值（比较、`length > 0`）时直接用它，否则 `!!x`；嵌套的两处（`storedValue ? (storedValue === true ? true : false) : false`、周期状态的判断）化成一个等价的比较。web 53 条、ui 1 条。同一批文件里另有 4 条：
- `dropdowns/cycle/cycle-options.tsx`：打开时的副作用列出全部依赖（`react-hooks/exhaustive-deps`）并去掉 `&&` 表达式语句（`no-unused-expressions`）。原来的 `onOpen` 判断 `!cycleIds`，而 `cycleIds` 总是数组，于是从不取周期；改为 `!getProjectCycleIds(projectId)`：没取过时取（spec 第 3 节第 6 条）。
- `modules/modal.tsx`：两个 `.then`/`.catch` 链改为 `try`/`catch`（`promise/always-return`）。

**Tests:** 行为不变（`cycle-options` 除外，见上）；现有的单元测试和 `make e2e` 覆盖。

- [ ] **Step 1: 44 个文件**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page.tsx
+++ b/web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page.tsx
@@ -39,7 +39,7 @@
     cycleId,
   });
   // derived values
-  const isSidebarCollapsed = storedValue ? (storedValue === true ? true : false) : false;
+  const isSidebarCollapsed = storedValue === true;
   const cycle = getCycleById(cycleId);
   const project = getProjectById(projectId);
   const pageTitle = project?.name && cycle?.name ? `${project?.name} - ${cycle?.name}` : undefined;
```

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx
+++ b/web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx
@@ -32,7 +32,7 @@
   // const { issuesFilter } = useIssues(EIssuesStoreType.MODULE);
   // local storage
   const { setValue, storedValue } = useLocalStorage("module_sidebar_collapsed", "false");
-  const isSidebarCollapsed = storedValue ? (storedValue === "true" ? true : false) : false;
+  const isSidebarCollapsed = storedValue === "true";
   // fetching module details
   const { error } = useSWR(`CURRENT_MODULE_DETAILS_${moduleId}`, () =>
     fetchModuleDetails(workspaceSlug, projectId, moduleId)
```

`web/apps/web/core/components/common/filters/created-at.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/common/filters/created-at.tsx
+++ b/web/apps/web/core/components/common/filters/created-at.tsx
@@ -33,7 +33,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -64,7 +64,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/common/filters/created-by.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/common/filters/created-by.tsx
+++ b/web/apps/web/core/components/common/filters/created-by.tsx
@@ -74,7 +74,7 @@
                   return (
                     <FilterOption
                       key={`member-${member.id}`}
-                      isChecked={appliedFilters?.includes(member.id) ? true : false}
+                      isChecked={!!appliedFilters?.includes(member.id)}
                       onClick={() => handleUpdate(member.id)}
                       icon={
                         <Avatar
```

`web/apps/web/core/components/cycles/archived-cycles/header.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/cycles/archived-cycles/header.tsx
+++ b/web/apps/web/core/components/cycles/archived-cycles/header.tsx
@@ -30,7 +30,7 @@
   const { currentProjectArchivedFilters, archivedCyclesSearchQuery, updateFilters, updateArchivedCyclesSearchQuery } =
     useCycleFilter();
   // states
-  const [isSearchOpen, setIsSearchOpen] = useState(archivedCyclesSearchQuery !== "" ? true : false);
+  const [isSearchOpen, setIsSearchOpen] = useState(archivedCyclesSearchQuery !== "");
   // outside click detector hook
   useOutsideClickDetector(inputRef, () => {
     if (isSearchOpen && archivedCyclesSearchQuery.trim() === "") setIsSearchOpen(false);
```

`web/apps/web/core/components/cycles/cycles-view-header.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/cycles/cycles-view-header.tsx
+++ b/web/apps/web/core/components/cycles/cycles-view-header.tsx
@@ -32,7 +32,7 @@
   const { currentProjectFilters, searchQuery, updateFilters, updateSearchQuery } = useCycleFilter();
   const { t } = useTranslation();
   // states
-  const [isSearchOpen, setIsSearchOpen] = useState(searchQuery !== "" ? true : false);
+  const [isSearchOpen, setIsSearchOpen] = useState(searchQuery !== "");
   // outside click detector hook
   useOutsideClickDetector(inputRef, () => {
     if (isSearchOpen && searchQuery.trim() === "") setIsSearchOpen(false);
```

`web/apps/web/core/components/cycles/dropdowns/filters/end-date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/cycles/dropdowns/filters/end-date.tsx
+++ b/web/apps/web/core/components/cycles/dropdowns/filters/end-date.tsx
@@ -33,7 +33,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -64,7 +64,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/cycles/dropdowns/filters/start-date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/cycles/dropdowns/filters/start-date.tsx
+++ b/web/apps/web/core/components/cycles/dropdowns/filters/start-date.tsx
@@ -32,7 +32,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -63,7 +63,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/cycles/dropdowns/filters/status.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/cycles/dropdowns/filters/status.tsx
+++ b/web/apps/web/core/components/cycles/dropdowns/filters/status.tsx
@@ -40,7 +40,7 @@
             filteredOptions.map((status) => (
               <FilterOption
                 key={status.value}
-                isChecked={appliedFilters?.includes(status.value) ? true : false}
+                isChecked={!!appliedFilters?.includes(status.value)}
                 onClick={() => handleUpdate(status.value)}
                 title={t(status.i18n_title)}
               />
```

`web/apps/web/core/components/dropdowns/cycle/cycle-options.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/dropdowns/cycle/cycle-options.tsx
+++ b/web/apps/web/core/components/dropdowns/cycle/cycle-options.tsx
@@ -52,13 +52,10 @@
   const { isMobile } = usePlatformOS();
 
   useEffect(() => {
-    if (isOpen) {
-      onOpen();
-      if (!isMobile) {
-        inputRef.current && inputRef.current.focus();
-      }
-    }
-  }, [isOpen, isMobile]);
+    if (!isOpen) return;
+    if (workspaceSlug && !getProjectCycleIds(projectId)) void fetchAllCycles(workspaceSlug, projectId);
+    if (!isMobile) inputRef.current?.focus();
+  }, [isOpen, isMobile, workspaceSlug, projectId, getProjectCycleIds, fetchAllCycles]);
 
   // popper-js init
   const { styles, attributes } = usePopper(referenceElement, popperElement, {
@@ -76,13 +73,9 @@
   const cycleIds = (getProjectCycleIds(projectId) ?? [])?.filter((cycleId) => {
     const cycleDetails = getCycleById(cycleId);
     if (currentCycleId && currentCycleId === cycleId) return false;
-    return cycleDetails?.status ? (cycleDetails?.status.toLowerCase() != "completed" ? true : false) : true;
+    return cycleDetails?.status?.toLowerCase() !== "completed";
   });
 
-  const onOpen = () => {
-    if (workspaceSlug && !cycleIds) fetchAllCycles(workspaceSlug, projectId);
-  };
-
   const searchInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
     if (query !== "" && e.key === "Escape") {
       e.stopPropagation();
```

`web/apps/web/core/components/inbox/inbox-filter/filters/date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/inbox/inbox-filter/filters/date.tsx
+++ b/web/apps/web/core/components/inbox/inbox-filter/filters/date.tsx
@@ -44,7 +44,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = filterValue?.filter((f) => isDate(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
 
   const handleCustomDate = () => {
@@ -78,7 +78,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={filterValue?.includes(option.value) ? true : false}
+                  isChecked={!!filterValue?.includes(option.value)}
                   onClick={() => handleInboxIssueFilters(filterKey, handleFilterValue(option.value))}
                   title={option.name}
                   multiple={false}
```

`web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx
+++ b/web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx
@@ -61,7 +61,7 @@
                 {filteredOptions.slice(0, itemsToRender).map((label) => (
                   <FilterOption
                     key={label?.id}
-                    isChecked={filterValue?.includes(label?.id) ? true : false}
+                    isChecked={!!filterValue?.includes(label?.id)}
                     onClick={() => handleInboxIssueFilters("labels", handleFilterValue(label.id))}
                     icon={<LabelIcons color={label.color} />}
                     title={label.name}
```

`web/apps/web/core/components/inbox/inbox-filter/filters/members.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/inbox/inbox-filter/filters/members.tsx
+++ b/web/apps/web/core/components/inbox/inbox-filter/filters/members.tsx
@@ -82,7 +82,7 @@
                   return (
                     <FilterOption
                       key={`members-${member.id}`}
-                      isChecked={filterValue?.includes(member.id) ? true : false}
+                      isChecked={!!filterValue?.includes(member.id)}
                       onClick={() => handleInboxIssueFilters(filterKey, handleFilterValue(member.id))}
                       icon={
                         <Avatar
```

`web/apps/web/core/components/inbox/inbox-filter/filters/priority.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/inbox/inbox-filter/filters/priority.tsx
+++ b/web/apps/web/core/components/inbox/inbox-filter/filters/priority.tsx
@@ -47,7 +47,7 @@
             filteredOptions.map((priority) => (
               <FilterOption
                 key={priority.key}
-                isChecked={filterValue?.includes(priority.key) ? true : false}
+                isChecked={!!filterValue?.includes(priority.key)}
                 onClick={() => handleInboxIssueFilters("priority", handleFilterValue(priority.key))}
                 icon={<PriorityIcon priority={priority.key} className="h-3.5 w-3.5" />}
                 title={priority.title}
```

`web/apps/web/core/components/inbox/inbox-filter/filters/status.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/inbox/inbox-filter/filters/status.tsx
+++ b/web/apps/web/core/components/inbox/inbox-filter/filters/status.tsx
@@ -58,7 +58,7 @@
             filteredOptions.map((status) => (
               <FilterOption
                 key={status.key}
-                isChecked={filterValue?.includes(status.status) ? true : false}
+                isChecked={!!filterValue?.includes(status.status)}
                 onClick={() => handleStatusFilterSelect(status.status)}
                 icon={<InboxStatusIcon type={status.status} className="h-3.5 w-3.5" />}
                 title={t(status.i18n_title)}
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx
@@ -49,7 +49,7 @@
         return (
           <FilterOption
             key={option.key}
-            isChecked={selectedExtraOptions?.[option.key] ? true : false}
+            isChecked={!!selectedExtraOptions?.[option.key]}
             onClick={() => handleUpdate(option.key, !selectedExtraOptions?.[option.key])}
             title={t(option.titleTranslationKey)}
           />
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/order-by.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/order-by.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/order-by.tsx
@@ -40,7 +40,7 @@
           {ISSUE_ORDER_BY_OPTIONS.filter((option) => orderByOptions.includes(option.key)).map((orderBy) => (
             <FilterOption
               key={orderBy?.key}
-              isChecked={activeOrderBy === orderBy?.key ? true : false}
+              isChecked={activeOrderBy === orderBy?.key}
               onClick={() => handleUpdate(orderBy.key)}
               title={t(orderBy.titleTranslationKey)}
               multiple={false}
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/sub-group-by.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/sub-group-by.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/sub-group-by.tsx
@@ -46,7 +46,7 @@
             return (
               <FilterOption
                 key={subGroupBy?.key}
-                isChecked={selectedSubGroupBy === subGroupBy?.key ? true : false}
+                isChecked={selectedSubGroupBy === subGroupBy?.key}
                 onClick={() => handleUpdate(subGroupBy.key)}
                 title={t(subGroupBy.titleTranslationKey)}
                 multiple={false}
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/assignee.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/assignee.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/assignee.tsx
@@ -74,7 +74,7 @@
                   return (
                     <FilterOption
                       key={`assignees-${member.id}`}
-                      isChecked={appliedFilters?.includes(member.id) ? true : false}
+                      isChecked={!!appliedFilters?.includes(member.id)}
                       onClick={() => handleUpdate(member.id)}
                       icon={
                         <Avatar
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/due-date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/due-date.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/due-date.tsx
@@ -32,7 +32,7 @@
 
   const isCustomDateSelected = () => {
     const isCustomFateApplied = appliedFilters?.filter((f) => f.includes("-")) || [];
-    return isCustomFateApplied.length > 0 ? true : false;
+    return isCustomFateApplied.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -63,7 +63,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/priority.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/priority.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/priority.tsx
@@ -42,7 +42,7 @@
             filteredOptions.map((priority) => (
               <FilterOption
                 key={priority.key}
-                isChecked={appliedFilters?.includes(priority.key) ? true : false}
+                isChecked={!!appliedFilters?.includes(priority.key)}
                 onClick={() => handleUpdate(priority.key)}
                 icon={<PriorityIcon priority={priority.key} className="h-3.5 w-3.5" />}
                 title={priority.title}
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/project.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/project.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/project.tsx
@@ -65,7 +65,7 @@
                 {sortedOptions.slice(0, itemsToRender).map((project) => (
                   <FilterOption
                     key={`project-${project.id}`}
-                    isChecked={appliedFilters?.includes(project.id) ? true : false}
+                    isChecked={!!appliedFilters?.includes(project.id)}
                     onClick={() => handleUpdate(project.id)}
                     icon={
                       <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/start-date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/start-date.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/start-date.tsx
@@ -31,7 +31,7 @@
 
   const isCustomDateSelected = () => {
     const isCustomFateApplied = appliedFilters?.filter((f) => f.includes("-")) || [];
-    return isCustomFateApplied.length > 0 ? true : false;
+    return isCustomFateApplied.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -62,7 +62,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state-group.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state-group.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state-group.tsx
@@ -49,7 +49,7 @@
               {filteredOptions.slice(0, itemsToRender).map((stateGroup) => (
                 <FilterOption
                   key={stateGroup.key}
-                  isChecked={appliedFilters?.includes(stateGroup.key) ? true : false}
+                  isChecked={!!appliedFilters?.includes(stateGroup.key)}
                   onClick={() => handleUpdate(stateGroup.key)}
                   icon={<StateGroupIcon stateGroup={stateGroup.key} />}
                   title={stateGroup.label}
```

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
+++ b/web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
@@ -58,7 +58,7 @@
                 {sortedOptions.slice(0, itemsToRender).map((state) => (
                   <FilterOption
                     key={state.id}
-                    isChecked={appliedFilters?.includes(state.id) ? true : false}
+                    isChecked={!!appliedFilters?.includes(state.id)}
                     onClick={() => handleUpdate(state.id)}
                     icon={
                       <StateGroupIcon
```

`web/apps/web/core/components/modules/archived-modules/header.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/archived-modules/header.tsx
+++ b/web/apps/web/core/components/modules/archived-modules/header.tsx
@@ -39,7 +39,7 @@
     workspace: { workspaceMemberIds },
   } = useMember();
   // states
-  const [isSearchOpen, setIsSearchOpen] = useState(archivedModulesSearchQuery !== "" ? true : false);
+  const [isSearchOpen, setIsSearchOpen] = useState(archivedModulesSearchQuery !== "");
   // outside click detector hook
   useOutsideClickDetector(inputRef, () => {
     if (isSearchOpen && archivedModulesSearchQuery.trim() === "") setIsSearchOpen(false);
```

`web/apps/web/core/components/modules/dropdowns/filters/lead.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/dropdowns/filters/lead.tsx
+++ b/web/apps/web/core/components/modules/dropdowns/filters/lead.tsx
@@ -74,7 +74,7 @@
                   return (
                     <FilterOption
                       key={`lead-${member.id}`}
-                      isChecked={appliedFilters?.includes(member.id) ? true : false}
+                      isChecked={!!appliedFilters?.includes(member.id)}
                       onClick={() => handleUpdate(member.id)}
                       icon={
                         <Avatar
```

`web/apps/web/core/components/modules/dropdowns/filters/members.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/dropdowns/filters/members.tsx
+++ b/web/apps/web/core/components/modules/dropdowns/filters/members.tsx
@@ -74,7 +74,7 @@
                   return (
                     <FilterOption
                       key={`member-${member.id}`}
-                      isChecked={appliedFilters?.includes(member.id) ? true : false}
+                      isChecked={!!appliedFilters?.includes(member.id)}
                       onClick={() => handleUpdate(member.id)}
                       icon={
                         <Avatar
```

`web/apps/web/core/components/modules/dropdowns/filters/start-date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/dropdowns/filters/start-date.tsx
+++ b/web/apps/web/core/components/modules/dropdowns/filters/start-date.tsx
@@ -33,7 +33,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -64,7 +64,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/modules/dropdowns/filters/status.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/dropdowns/filters/status.tsx
+++ b/web/apps/web/core/components/modules/dropdowns/filters/status.tsx
@@ -41,7 +41,7 @@
             filteredOptions.map((status) => (
               <FilterOption
                 key={status.value}
-                isChecked={appliedFilters?.includes(status.value) ? true : false}
+                isChecked={!!appliedFilters?.includes(status.value)}
                 onClick={() => handleUpdate(status.value)}
                 icon={<ModuleStatusIcon status={status.value} />}
                 title={t(status.i18n_label)}
```

`web/apps/web/core/components/modules/dropdowns/filters/target-date.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/dropdowns/filters/target-date.tsx
+++ b/web/apps/web/core/components/modules/dropdowns/filters/target-date.tsx
@@ -33,7 +33,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -64,7 +64,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple
```

`web/apps/web/core/components/modules/modal.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/modal.tsx
+++ b/web/apps/web/core/components/modules/modal.tsx
@@ -57,45 +57,44 @@
     if (!workspaceSlug || !projectId) return;
 
     const selectedProjectId = payload.project_id ?? projectId;
-    await createModule(workspaceSlug, selectedProjectId, payload)
-      .then((_res) => {
-        handleClose();
-        setToast({
-          type: TOAST_TYPE.SUCCESS,
-          title: "Success!",
-          message: "Module created successfully.",
-        });
-      })
-      .catch((err) => {
-        setToast({
-          type: TOAST_TYPE.ERROR,
-          title: "Error!",
-          message: err?.detail ?? err?.error ?? "Module could not be created. Please try again.",
-        });
+    try {
+      await createModule(workspaceSlug, selectedProjectId, payload);
+      handleClose();
+      setToast({
+        type: TOAST_TYPE.SUCCESS,
+        title: "Success!",
+        message: "Module created successfully.",
       });
+    } catch (err: unknown) {
+      const error = err as { detail?: string; error?: string } | undefined;
+      setToast({
+        type: TOAST_TYPE.ERROR,
+        title: "Error!",
+        message: error?.detail ?? error?.error ?? "Module could not be created. Please try again.",
+      });
+    }
   };
 
   const handleUpdateModule = async (payload: Partial<IModule>) => {
     if (!workspaceSlug || !projectId || !data) return;
 
     const selectedProjectId = payload.project_id ?? projectId;
-    await updateModuleDetails(workspaceSlug, selectedProjectId, data.id, payload)
-      .then((_res) => {
-        handleClose();
-
-        setToast({
-          type: TOAST_TYPE.SUCCESS,
-          title: "Success!",
-          message: "Module updated successfully.",
-        });
-      })
-      .catch((err) => {
-        setToast({
-          type: TOAST_TYPE.ERROR,
-          title: "Error!",
-          message: err?.detail ?? err?.error ?? "Module could not be updated. Please try again.",
-        });
+    try {
+      await updateModuleDetails(workspaceSlug, selectedProjectId, data.id, payload);
+      handleClose();
+      setToast({
+        type: TOAST_TYPE.SUCCESS,
+        title: "Success!",
+        message: "Module updated successfully.",
       });
+    } catch (err: unknown) {
+      const error = err as { detail?: string; error?: string } | undefined;
+      setToast({
+        type: TOAST_TYPE.ERROR,
+        title: "Error!",
+        message: error?.detail ?? error?.error ?? "Module could not be updated. Please try again.",
+      });
+    }
   };
 
   const handleFormSubmit = async (formData: Partial<IModule>) => {
@@ -138,7 +137,7 @@
       <ModuleForm
         handleFormSubmit={handleFormSubmit}
         handleClose={handleClose}
-        status={data ? true : false}
+        status={!!data}
         projectId={activeProject ?? ""}
         setActiveProject={setActiveProject}
         data={data}
```

`web/apps/web/core/components/modules/module-view-header.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/module-view-header.tsx
+++ b/web/apps/web/core/components/modules/module-view-header.tsx
@@ -50,7 +50,7 @@
   const { t } = useTranslation();
 
   // states
-  const [isSearchOpen, setIsSearchOpen] = useState(searchQuery !== "" ? true : false);
+  const [isSearchOpen, setIsSearchOpen] = useState(searchQuery !== "");
 
   // handlers
   const handleFilters = useCallback(
```

`web/apps/web/core/components/modules/progress-sidebar/issue-progress.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/modules/progress-sidebar/issue-progress.tsx
+++ b/web/apps/web/core/components/modules/progress-sidebar/issue-progress.tsx
@@ -64,7 +64,7 @@
   if (!moduleDetails) return <></>;
   return (
     <div className="space-y-4 border-t border-subtle px-3 py-4">
-      <Disclosure defaultOpen={isModuleDateValid ? true : false}>
+      <Disclosure defaultOpen={!!isModuleDateValid}>
         {({ open }) => (
           <div className="space-y-6">
             {/* progress bar header */}
```

`web/apps/web/core/components/project-states/options/delete.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project-states/options/delete.tsx
+++ b/web/apps/web/core/components/project-states/options/delete.tsx
@@ -30,7 +30,7 @@
   const [isDeleteModal, setIsDeleteModal] = useState(false);
   const [isDelete, setIsDelete] = useState(false);
   // derived values
-  const isDeleteDisabled = state.default ? true : totalStates === 1 ? true : false;
+  const isDeleteDisabled = state.default || totalStates === 1;
 
   const handleDeleteState = async () => {
     if (isDeleteDisabled) return;
```

`web/apps/web/core/components/project-states/state-item-title.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project-states/state-item-title.tsx
+++ b/web/apps/web/core/components/project-states/state-item-title.tsx
@@ -66,7 +66,7 @@
           <div className="flex-shrink-0 text-11 transition-all">
             <StateMarksAsDefault
               stateId={state.id}
-              isDefault={state.default ? true : false}
+              isDefault={state.default}
               markStateAsDefaultCallback={props.stateOperationsCallbacks.markStateAsDefault}
             />
           </div>
```

`web/apps/web/core/components/project-states/state-item.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project-states/state-item.tsx
+++ b/web/apps/web/core/components/project-states/state-item.tsx
@@ -45,7 +45,7 @@
   const [isDraggedOver, setIsDraggedOver] = useState(false);
   const [closestEdge, setClosestEdge] = useState<string | null>(null);
   // derived values
-  const isDraggable = totalStates === 1 ? false : true;
+  const isDraggable = totalStates !== 1;
   const commonStateItemListProps = {
     stateCount: totalStates,
     state: state,
```

`web/apps/web/core/components/project/dropdowns/filters/access.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project/dropdowns/filters/access.tsx
+++ b/web/apps/web/core/components/project/dropdowns/filters/access.tsx
@@ -42,7 +42,7 @@
             filteredOptions.map((access) => (
               <FilterOption
                 key={access.key}
-                isChecked={appliedFilters?.includes(`${access.key}`) ? true : false}
+                isChecked={!!appliedFilters?.includes(`${access.key}`)}
                 onClick={() => handleUpdate(`${access.key}`)}
                 icon={<ProjectNetworkIcon iconKey={access.iconKey} />}
                 title={t(access.i18n_label)}
```

`web/apps/web/core/components/project/dropdowns/filters/created-at.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
+++ b/web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
@@ -32,7 +32,7 @@
 
   const isCustomDateSelected = () => {
     const isValidDateSelected = appliedFilters?.filter((f) => isInDateFormat(f.split(";")[0])) || [];
-    return isValidDateSelected.length > 0 ? true : false;
+    return isValidDateSelected.length > 0;
   };
   const handleCustomDate = () => {
     if (isCustomDateSelected()) {
@@ -63,7 +63,7 @@
               {filteredOptions.map((option) => (
                 <FilterOption
                   key={option.value}
-                  isChecked={appliedFilters?.includes(option.value) ? true : false}
+                  isChecked={!!appliedFilters?.includes(option.value)}
                   onClick={() => handleUpdate(option.value)}
                   title={option.name}
                   multiple={false}
```

`web/apps/web/core/components/project/dropdowns/filters/lead.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project/dropdowns/filters/lead.tsx
+++ b/web/apps/web/core/components/project/dropdowns/filters/lead.tsx
@@ -74,7 +74,7 @@
                   return (
                     <FilterOption
                       key={`lead-${member.id}`}
-                      isChecked={appliedFilters?.includes(member.id) ? true : false}
+                      isChecked={!!appliedFilters?.includes(member.id)}
                       onClick={() => handleUpdate(member.id)}
                       icon={
                         <Avatar
```

`web/apps/web/core/components/project/dropdowns/filters/members.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/components/project/dropdowns/filters/members.tsx
+++ b/web/apps/web/core/components/project/dropdowns/filters/members.tsx
@@ -74,7 +74,7 @@
                   return (
                     <FilterOption
                       key={`member-${member.id}`}
-                      isChecked={appliedFilters?.includes(member.id) ? true : false}
+                      isChecked={!!appliedFilters?.includes(member.id)}
                       onClick={() => handleUpdate(member.id)}
                       icon={
                         <Avatar
```

`web/apps/web/core/lib/wrappers/store-wrapper.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/lib/wrappers/store-wrapper.tsx
+++ b/web/apps/web/core/lib/wrappers/store-wrapper.tsx
@@ -39,7 +39,7 @@
    */
   useEffect(() => {
     const localValue = localStorage && localStorage.getItem("app_sidebar_collapsed");
-    const localBoolValue = localValue ? (localValue === "true" ? true : false) : false;
+    const localBoolValue = localValue === "true";
     if (localValue && sidebarCollapsed === undefined) toggleSidebar(localBoolValue);
   }, [sidebarCollapsed, setTheme, toggleSidebar]);
 
```

`web/apps/web/core/store/notifications/workspace-notifications.store.ts`（对 `cec4ec9` 的差异）：

```diff
--- a/web/apps/web/core/store/notifications/workspace-notifications.store.ts
+++ b/web/apps/web/core/store/notifications/workspace-notifications.store.ts
@@ -141,9 +141,9 @@
           }
         } else {
           if (this.filters.snoozed) {
-            return n.snoozed_till ? true : false;
+            return !!n.snoozed_till;
           } else if (this.filters.archived) {
-            return n.archived_at ? true : false;
+            return !!n.archived_at;
           } else {
             return true;
           }
```

`web/packages/ui/src/form-fields/password/indicator.tsx`（对 `cec4ec9` 的差异）：

```diff
--- a/web/packages/ui/src/form-fields/password/indicator.tsx
+++ b/web/packages/ui/src/form-fields/password/indicator.tsx
@@ -25,7 +25,7 @@
   const criteria = getPasswordCriteria(password);
   const strengthInfo = getStrengthInfo(strength);
 
-  const isPasswordMeterVisible = isFocused ? true : strength === E_PASSWORD_STRENGTH.STRENGTH_VALID ? false : true;
+  const isPasswordMeterVisible = isFocused || strength !== E_PASSWORD_STRENGTH.STRENGTH_VALID;
 
   if ((!password && !showCriteria) || !isPasswordMeterVisible) {
     return null;
```

- [ ] **Step 2: oxlint 上限**

`web/apps/web/package.json`（对 Task 9 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 514",
+    "check:lint": "node ../../../tools/lint-cap.mjs 457",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

`web/packages/ui/package.json`（对 Task 1 版本的差异）：

```diff
--- a/web/packages/ui/package.json
+++ b/web/packages/ui/package.json
@@ -16,7 +16,7 @@
   "scripts": {
     "build": "tsdown",
     "dev": "tsdown --watch --no-clean",
-    "check:lint": "node ../../../tools/lint-cap.mjs 20",
+    "check:lint": "node ../../../tools/lint-cap.mjs 19",
     "check:types": "tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt ."
```

- [ ] **Step 3: 检查**

Run: `pnpm -C web/apps/web exec oxlint --format unix . | grep -c no-unneeded-ternary`
Expected: `0`（`web/packages/ui` 下同样是 `0`）。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（web 的 oxlint 上限 457，ui 19）。

- [ ] **Step 4: 提交**

```bash
git add web/apps/web web/packages/ui
```
```bash
git commit -m "refactor(M2/P5): no unneeded ternary left

The 54 x ? true : false and their kin become boolean expressions. In
the same files: the cycle dropdown's open effect lists its dependencies
and now fetches the project's cycles when they are not loaded (its
onOpen checked an array that was always there, and never did); the
module modal awaits instead of chaining. The caps drop: web 457, ui 19.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 检查都通过；全仓没有 `no-unneeded-ternary` 警告。

---

### Task 11: 文档

**Files:**
- Modify: `docs/v0/frontend-changes.md`、`README.md`
- Modify: `docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md`、`M1-closeout.md`

**Interfaces:**（M2 设计 3.20 的 P5 行）
- `frontend-changes.md`：`packages/services` 一行写到 P5 删掉的部分；3.1 的 M2 一行改为"已完成"；3.2 的错误处理一行写 P5 改完的个人设置。
- `README.md`：M2 能用的页面包括个人设置的四个标签页；PAT 泄露时在 api-tokens 页或安全页查看、撤销。
- 交接：`M1-P3-trim-platform.md`（API 令牌的旧地址）、`M1-closeout.md`（死成员和死 prop、oxlint、主题下拉框）都处理完，状态改为 `done`，各加"处理结果（M2/P5）"一节。

**Tests:** 无。

- [ ] **Step 1: 文档**

`docs/v0/frontend-changes.md`（对 `cec4ec9` 的差异）：

```diff
--- a/docs/v0/frontend-changes.md
+++ b/docs/v0/frontend-changes.md
@@ -15,7 +15,7 @@
 | 来源提交 | Plane `02c19e1341d93141e8ad7b3278298adce208bafc`（`preview` 分支，`package.json` 中的版本是 1.4.2） |
 | 迁入方式 | M0/P5 用 `git archive` 按上面的完整提交复制。迁入时 13 个目录与 Plane 中对应目录的 git 树对象完全相同，之后的每一处改动都登记在本清单中（见 [P5 spec](M0-foundation/specs/P5-web-import.md) 2.3） |
 | 使用 | `apps/web` → `web/apps/web`；packages 中的 types、constants、ui、propel、editor、i18n、hooks、utils、shared-state、tailwind-config、typescript-config → `web/packages/<包名>`；`patches/react-color@2.19.3.patch` → 仓库根目录的 `patches/` |
-| 暂时使用 | `packages/services`：web 的令牌设置页和文件工具函数依赖它。M1/P1 删掉了其中没有调用方的 49 个文件，只剩令牌服务、地址规范化（带单元测试）和上传文件的元数据工具；M2（PAT）和 M5（文件）重写对应的接口调用后，将它删除 |
+| 暂时使用 | `packages/services`：web 的 axios 基类和文件工具函数依赖它。M1/P1 删掉了其中没有调用方的 49 个文件；M2/P5 删掉了令牌服务和 Plane 的 axios 基类（`api.service.ts`、`developer/`，PAT 改由 web 的 store 经生成的客户端调用），只剩地址规范化（带单元测试）和上传文件的元数据工具；M5 重写文件的接口调用后，将它删除 |
 | 不使用 | `apps/admin`、`apps/space`、`apps/live`、`apps/api`、`apps/proxy`、`packages/logger`、`packages/decorators`、`packages/codemods`（已核实 web 及其依赖的包都不引用它们）；根目录的 `.npmrc`（pnpm 11 只从 `.npmrc` 读取认证和仓库地址，其中的其他设置都不起作用）；husky、lint-staged（Git 钩子）和 react-doctor |
 | 第三方依赖 | `@makeplane/propel` 0.3.0：Plane 发布在 npm 上的设计系统，AGPL-3.0-only，版本锁定（`pnpm-workspace.yaml` 的 catalog，不跟随发布更新）。tarball 的完整性哈希 `sha512-nGhiE42vLQVvv7NZOcQYKARJiTXyKDSSTcHOzPdtFpa+WkSz9B91jw6i+3Ikz5cpkv/aEKjOTJF+cByw2zv5ZQ==`（`pnpm-lock.yaml`）。源码：npm 上这个版本带 SLSA 来源证明，记录它由 `github.com/makeplane/propel` 的 `packages/propel`、提交 `0a31b1529c0f249a058e59ade313d8b34ea8f964` 经 `.github/workflows/release.yml` 构建；这个仓库不公开（M1/P5 时访问为 404）。包里的 source map 带着 1376 个源文件中 1375 个的全文。以后要自己修改设计令牌时，按 [M1 设计](M1-frontend-trim/M1-design.md) 3.10 评估并入源码 |
 
@@ -198,7 +198,7 @@
 
 | 领域 | 所属 M | 状态 |
 |---|---|---|
-| 认证、用户、实例配置、PAT；令牌管理器 | M2 | 进行中 |
+| 认证、用户、实例配置、PAT；令牌管理器 | M2 | 已完成 |
 | 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 计划中 |
 | 工作项、列表（分页和分组的新结构）、子任务、关联、链接、评论、表情回应、操作动态、搜索、历史版本、草稿、工作项归档 | M4 | 计划中 |
 | 文件、附件、编辑器图片（上传改为 `{method, url, headers}` 形式的 PUT） | M5 | 计划中 |
@@ -214,7 +214,7 @@
 | `core/store/issue/helpers/base-issues.store.ts` 等列表相关 store | 使用新接口的分页结构（不透明游标 `next_cursor`）和分组结构（`groups` 数组），不再按"已加载条数 ÷ 每页条数"拼页码游标 | 新接口的分页和分组设计 | 计划中 | |
 | 用户和认证相关的 store | 登录、退出、续期改走令牌管理器 | 认证改为 Bearer 令牌 | 已完成 | M2/P4 |
 | 登录、注册、退出、修改密码的提交方式 | 删除 CSRF 令牌和 Django 会话的表单提交，改走令牌管理器 | 认证改为 Bearer 令牌；CSRF 是传输方式的一部分，和它的替代品一起删除（[M1 设计](M1-frontend-trim/M1-design.md) 3.6） | 已完成 | M2/P4 |
-| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误，停用账户的弹窗显示它的 `detail` 或 `title`。其余的随各自对接新接口改：个人设置的其余部分（general 页的保存、preferences、安全页的 PAT 列表、api-tokens）在 P5（[M2 设计](M2-auth/M2-design.md) 12 节、7.7）；M3–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
+| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）：general 页的保存和 api-tokens 页的创建把字段错误显示在字段下方，其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户的失败都在提示中，文案按 `code` 取（`helpers/authentication.helper.ts` 的 `errorMessageKey`）。M3–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
 | 文件上传相关的 store 和调用方 | 预签名 POST 改为 `{method, url, headers}` 形式的 PUT | 文件存储改为 PUT 上传 | 计划中 | |
 | 迭代和模块的归属 | 通过工作项的 `cycle_id`、`module_ids` 字段修改，不再调用单独的接口 | 接口设计 | 计划中 | |
 
```

`README.md`（对 `cec4ec9` 的差异）：

```diff
--- a/README.md
+++ b/README.md
@@ -65,7 +65,7 @@
 - **构建**：`make build` 构建前端，复制到 `server/internal/platform/webui/dist/`，编译出内嵌前端的 `bin/nerve`。运行时要在 `server/` 目录下，`config.local.yaml` 才能生效：`cd server && NERVE_ENV=dev ../bin/nerve serve`，之后打开 http://127.0.0.1:8080 。`dist/` 中只提交了 `.gitkeep`，没有构建过前端时页面上只有一句提示。
 - **清掉旧的构建产物**：`make build` 之后，`make run` 和 `go test` 都会继续内嵌这份构建。要去掉它：`find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete`（与 Makefile 里 `make build` 自己的清理命令相同）。
 - **登录状态**：会话由令牌管理器（`web/apps/web/core/lib/auth/`）保管，不用 Cookie：注册、登录得到的令牌交给它，续期和退出由它完成。访问令牌只在内存里；刷新令牌和 `login_id` 存在 localStorage 的 `nerve.auth` 中，同一个浏览器的标签页共用这一个会话，一次只有一个标签页在续期（`navigator.locks`，没有它时用 localStorage 的租约，见"部署"一节的"HTTPS"一条）。带令牌的请求得到 401 时，认证中间件（同一目录）续期一次、重发一次；只有续期得到 401、或者重发后仍是 401 才结束会话，页面回到登录页。续期超时（8 秒）、断网或得到 401 以外的错误（如 429、5xx）时会话保留：打开或刷新页面时的第一次续期这样失败，页面显示"会话暂不可用"的界面（"Cannot reach the server for now"），按 `Retry-After` 或退避自动重试，也可以点"Try again"；使用中的续期这样失败，需要它的请求失败，不自动重试，其中取账户的请求失败时也显示这个界面，但只能手动重试。会话每次变化（本标签页或另一个标签页登录、退出，会话结束），stores 都按新会话重建；属于旧会话的请求以 `SessionChangedError` 结束，不再发出，也不续期重发。
-- **M2 中看到的页面**：登录、注册；新手引导（三步中只有资料一步能用，工作区、邀请两步要用 M3 的接口）；完成了新手引导的账户登录后的落点 `/create-workspace`（创建工作区在 M3）；个人设置的 general 和 security。工作区相关的页面和操作还在调用 Plane 的接口，Nerve 返回 404，从 M3 起对接；个人设置的 preferences（时区列表）和 api-tokens 在 M2/P5 对接。
+- **M2 中看到的页面**：登录、注册；新手引导（三步中只有资料一步能用，工作区、邀请两步要用 M3 的接口）；完成了新手引导的账户登录后的落点 `/create-workspace`（创建工作区在 M3）；个人设置的四个标签页（general、preferences、security、api-tokens）。工作区相关的页面和操作还在调用 Plane 的接口，Nerve 返回 404，从 M3 起对接。
 - **lint 警告数等于上限**：每个包的 `check:lint` 脚本是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，它运行 oxlint，要求警告数正好等于上限，有任何错误都失败。警告多了，`make lint-web` 失败并列出这个包的全部警告：修掉新增的那几条，上限只能调低。警告少了（修掉了警告，或者删掉了带警告的代码），同样失败，并给出应调低到的数值：在同一个提交里把上限改成这个数。`make lint-web` 只打印失败任务的输出；要看某个包的全部警告，执行 `pnpm --filter <包名> exec oxlint .`。
 - **关键词守卫**：`make lint-web` 的第一步是 `node tools/keywords.mjs`，规则在 `tools/keywords.json`（M1 设计 7.4）。它检查 git 列出的文件（包括还没 `git add` 的新文件），命中规则、又没有登记例外就失败，并列出规则、文件、行号和命中的原文；规则或文件读取有问题时以 2 退出。删掉一个功能时，在同一个提交里加上它的规则（每条规则带理由和命中、不命中的样本）。确实要保留的命中登记为例外：规则、文件、命中的原文、理由和到期的 M 或 Phase，一条例外只覆盖一处；例外不再命中任何内容时守卫会提醒删掉它。`tools/` 下的脚本本身也由 `make lint-web` 检查：oxlint 不允许警告（根目录 `package.json` 的 `check:lint`）；oxfmt 检查它们和根目录的工具链配置的格式，文件列表只写在根目录 `package.json` 的 `fix:format` 里，`check:format` 就是带 `--check` 运行它。
 - **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包，以及根目录 `fix:format` 列出的文件。
@@ -118,7 +118,7 @@
   - `activate --email <邮箱>`：恢复账户，没有过期的 PAT **重新可用**，输出它们的个数。
 
   `set-email` 和 `activate` 都不是账户被盗后的恢复手段：怀疑账户被盗时，另外执行 `reset-password`。
-- **令牌泄露后的恢复**：刷新令牌存在浏览器的 localStorage 里，页面上的 XSS 能读出它，换来访问令牌后创建一个永不过期的 PAT（创建 PAT 不要求输入密码）。这个 PAT 不受退出、修改密码和会话 30 天期限的影响，还能再创建 PAT，所以泄露的影响不以 30 天为限（M2 设计 8.5）。怀疑泄露时：查看账户的 PAT 列表（`GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，逐个撤销；或者由服务器管理员执行 `nerve users reset-password --email <邮箱>`，它结束该账户的全部会话、撤销全部 PAT。
+- **令牌泄露后的恢复**：刷新令牌存在浏览器的 localStorage 里，页面上的 XSS 能读出它，换来访问令牌后创建一个永不过期的 PAT（创建 PAT 不要求输入密码）。这个 PAT 不受退出、修改密码和会话 30 天期限的影响，还能再创建 PAT，所以泄露的影响不以 30 天为限（M2 设计 8.5）。怀疑泄露时：查看账户的 PAT 列表（个人设置的 api-tokens 页或 security 页，或 `GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，逐个撤销；或者由服务器管理员执行 `nerve users reset-password --email <邮箱>`，它结束该账户的全部会话、撤销全部 PAT。
 - **令牌的密钥扫描**：个人访问令牌以 `nrv_pat_` 开头，刷新令牌以 `nrv_rt_` 开头，但前缀不会让代码托管平台自动识别它们。要让平台发现提交里泄露的令牌，在它的密钥扫描中加自定义规则，例如 GitHub 仓库或组织设置的 Secret scanning → Custom patterns（需要平台提供这项功能）：个人访问令牌 `nrv_pat_[A-Za-z0-9_-]{43}`，刷新令牌 `nrv_rt_[A-Za-z0-9_-]{91}`。
 - **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。
 - **停机**：收到 SIGTERM 或 SIGINT 后，nerve 依次停下：先是 HTTP 优雅停机，不再接受新连接，处理中的请求最多等 `server.shutdown_timeout`（默认 20 秒）；然后是后台任务（River），不再取新任务，正在运行的任务最多再等 `jobs.shutdown_timeout`（默认 10 秒），到期取消它们，再等 1 秒；最后关闭连接池，最多等 5 秒。前两步到期时 nerve 记 ERROR 并以退出码 1 退出，连接池到期只记 WARN。默认配置下最坏约 36 秒；实测空闲的 nerve 只要几到二十几毫秒。容器或进程管理器的停止宽限期要比 36 秒长：Docker 默认只等 10 秒（Compose 用 `stop_grace_period` 调大），Kubernetes 默认 30 秒（`terminationGracePeriodSeconds`）；也可以调小这几个期限。宽限期不够时，nerve 在收尾中被 SIGKILL（再收到一次 SIGTERM 或 SIGINT 时也立即退出）：处理中的请求被切断，正在运行的任务留在 `running`，River 在它开始运行 1 小时后才把它当作卡住的任务重新排入重试。
```

- [ ] **Step 2: 交接**

`docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md`（对 `cec4ec9` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
+++ b/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
@@ -1,5 +1,5 @@
 ---
-status: open
+status: done
 from: M1/P3
 to: M2
 created: 2026-09-24
@@ -89,3 +89,11 @@
 仍未处理，状态保持 `open`：`packages/services` 的 API 令牌服务仍调旧地址 `/api/users/api-tokens/…`（没有结尾斜杠的 `retrieve`、`destroy` 也在其中），M2/P5 把 api-tokens 标签页改接 `/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}` 时删除。
 
 来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
+
+## 处理结果（M2/P5）
+
+- **API 令牌的旧地址**（完成）：`@nerve/services` 的令牌服务 `developer/` 和它的 axios 基类 `api.service.ts` 删除，包里只剩地址工具和文件工具（M2 设计 7.5）。api-tokens 页和 security 页的令牌列表经 PAT store（`core/store/user/api-token.store.ts`）用生成的客户端调 `GET`、`POST /api/v0/me/api-tokens` 和 `DELETE /api/v0/api-tokens/{token_id}`；store 属于会话的 `RootStore`，用这个会话的客户端。关键词规则 `plane-api-token-urls`（`/api/users/api-tokens`）和 `with-credentials`（`withCredentials:\s*true`）看住。
+
+全部处理完，状态改为 `done`。
+
+来源：[M2/P5 spec](../specs/P5-web-account.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-closeout.md`（对 `cec4ec9` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-closeout.md
+++ b/docs/v0/M2-auth/handoffs/M1-closeout.md
@@ -1,5 +1,5 @@
 ---
-status: open
+status: done
 from: M1/closeout
 to: M2
 created: 2026-09-25
@@ -35,3 +35,9 @@
 - **关闭条件**：本 M 的浏览器核对写明个人设置的主题下拉框在它的按钮旁展开（zh-CN 和 en 各一次）。
 
 来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节，2.10 的 I1。
+
+## 处理结果（M2/P5）
+
+- **死成员和死 prop**（完成）：`domains.mjs … --rows M2` 在 M1 收尾时 66 行，P5 开始时 64 行，结束时 24 行。删掉的有 PAT 的旧 store（`store/workspace/api-token.store.ts`，随 PAT store 重写）、`@nerve/services` 的 axios 基类、`settings.store` 的 `isLoading`、`error`、`isScrolled`、`toggleIsScrolled`，`issue/profile` 两个 store 的 `userId`、`issueFilterService`、`currentView`、`quickAddIssue`，`IUserPermissionStore` 的 `fetchWorkspaceLevelProjectEntities`，`IBaseIssueFilterStore` 和实现它的九个过滤 store 的 `appliedFilters`（M2 的一行只能连同接口一起删），个人设置两张表各写一遍的成员类型（合成一个类型），以及 5 个没人传的 prop（`NotAuthorizedView` 的 `actionButton`、`ProfileSidebar` 和 `ProfileSettingsHeading` 的 `className`、两个认证包装的 `isLoading`）。剩下的 24 行都不是死代码：23 行是测试替身和测试里的字面量（`fake-browser.ts`、`fake-nerve.ts`、`fake-time.ts` 的成员只由测试读取；`token-manager.test.ts` 的记录字面量由 `JSON.stringify` 读取），`OnboardingRoot` 的 `invitations` 由 M3 加回取数时传入（M2 设计 3.1、13.2）。逐行的理由在 [M2/P5 spec](../specs/P5-web-account.md) 附录 A。
+- **oxlint**（完成）：P5 改到的 116 个 web 文件和 P4 改到的文件都没有警告；`eslint(no-unneeded-ternary)` 全仓清零（54 条）。各包上限：web 551 → 457，ui 25 → 19，utils 18 → 16（逐个任务的数值见 [M2/P5 plan](../plans/P5-web-account.md)）。
+- **主题下拉框**（完成）：根因在 Headless UI 2.2 的 `Combobox.Options`，与语言无关：它克隆自己唯一的子元素时换上自己的 ref，放在子元素上的 react-popper 的 ref 从来没有被设置，列表停在 `position: absolute; left: 0; top: 0`，即页面左上角。`CustomSelect`、`CustomSearchSelect` 和 `DateDropdown` 改为把 popper 的 ref、样式和属性放在 `Combobox.Options` 本身（Headless UI 转发这个 ref）；`CustomSearchSelect` 另外设 `modal={false}`，否则搜索框所在的列表里的选项被设为 inert，点不了。三个组件还把自己的打开状态跟随 Headless UI 自己做的关闭（按钮上的 Escape、点击外部，`onClose`）：以前按 Escape 之后，下一次点击什么也不打开。A9 的页面版本在中英文下各核对一次主题列表在按钮旁展开、Escape 之后一次点击就再打开，A8 核对时区列表，A11 核对自定义有效期的日历；浏览器核对 C2、C3 在中英文下核对主题、时区、每周第一天三个列表和日历。
```

- [ ] **Step 3: 检查**

Run: `make lint-web`、`make knip`、`make test-web`、`make lint-go`、`make test`
Expected: 都通过；`make lint-go` 两段都是 `0 issues.`。

Run: `grep -rnE "withCredentials:\s*true|/api/users/api-tokens|/api/timezones/" web --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build --exclude-dir=.turbo`
Expected: 没有输出（M2 设计 12 节 P5 的完成标准）。

Run: `make e2e`
Expected: 全部通过。

- [ ] **Step 4: 提交**

```bash
git add docs/v0/frontend-changes.md README.md docs/v0/M2-auth/handoffs
```
```bash
git commit -m "docs(M2/P5): frontend changes, README and the handoffs P5 closes

frontend-changes: M2 done; @nerve/services down to the address and
file utilities; the settings' errors. README: the four settings tabs,
and where to look at and revoke tokens. The M1-P3 and M1-closeout
handoffs are done, each with what P5 did.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过，没有差异。

**Done when:** 检查都通过；两个交接的状态是 `done`。
