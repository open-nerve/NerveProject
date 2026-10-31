// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act, useImperativeHandle } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { EditorRefApi } from "@nerve/editor";
import { heldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { ApiError } from "@/lib/api-error";
import { DescriptionVersionsModal } from "./modal";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } } }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => ({ getWorkspaceBySlug: () => ({ id: "ws" }) }) }));
const binding = vi.hoisted(() => ({ ready: true, legacyCopy: vi.fn() }));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: ({
    initialValue,
    ref,
  }: {
    initialValue: string;
    ref: React.Ref<Pick<EditorRefApi, "getMarkDown" | "isEditorReadyToDiscard" | "copyMarkdownToClipboard">>;
  }) => {
    useImperativeHandle(ref, () => ({
      getMarkDown: () => "# Chosen\n\n**bold**\n",
      isEditorReadyToDiscard: () => binding.ready,
      copyMarkdownToClipboard: binding.legacyCopy,
    }));
    return <div data-html>{initialValue}</div>;
  },
}));
vi.mock("@makeplane/propel/components/avatar", () => ({ Avatar: () => null }));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: React.PropsWithChildren) => children,
}));
let renderer: Root;
let container: HTMLElement;
const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");
let clipboard: ReturnType<typeof heldChange<void>>;
let write: ReturnType<typeof vi.fn<(text: string) => Promise<void>>>;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  binding.ready = true;
  binding.legacyCopy.mockClear();
  toasts.length = 0;
  clipboard = heldChange<void>();
  write = vi.fn(() => clipboard.sent);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: write } });
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  clipboard.answer();
  if (clipboardDescriptor) Object.defineProperty(navigator, "clipboard", clipboardDescriptor);
  else Reflect.deleteProperty(navigator, "clipboard");
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const close = vi.fn();
function render(html: string | undefined, restore: (html: string) => Promise<unknown>, error?: unknown) {
  close.mockClear();
  return act(async () =>
    renderer.render(
      <DescriptionVersionsModal
        activeVersionDescription={html}
        activeVersionDetails={{ id: "v", ownerId: "actor", lastSavedAt: "2026-10-30T00:00:00Z" }}
        handleClose={close}
        handleNavigation={() => {}}
        handleRestore={restore}
        isNextDisabled={false}
        isPrevDisabled={false}
        isOpen
        isRestoreDisabled={false}
        projectId="A"
        workspaceSlug="ws"
        error={error}
        retry={async () => {}}
      />
    )
  );
}
const restoreButton = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent === "common.actions.restore"
  )!;
it.each([403, 422])("keeps the dialog and HTML when restoration is refused (%s)", async (status) => {
  // Catch on the test-owned promise prevents a deliberately old fire-and-forget implementation from leaking to Vitest.
  let reject!: (error: unknown) => void;
  const promise = new Promise<void>((_, no) => {
    reject = no;
  });
  void promise.catch(() => {});
  const restore = vi.fn(() => promise);
  await render("<p>chosen</p>", restore);
  await act(async () => restoreButton().click());
  expect(close).not.toHaveBeenCalled();
  expect(restoreButton().disabled).toBe(true);
  await act(async () => reject(new ApiError(status, { code: "forbidden", title: "refused", status })));
  expect(close).not.toHaveBeenCalled();
  expect(document.querySelector("[data-html]")?.textContent).toBe("<p>chosen</p>");
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  expect(restoreButton().disabled).toBe(false);
});
it("empty HTML is a loaded value and success closes only after confirmation", async () => {
  let resolve!: () => void;
  const promise = new Promise<void>((yes) => {
    resolve = yes;
  });
  const restore = vi.fn(() => promise);
  await render("", restore);
  expect(document.querySelector("[data-html]")).not.toBeNull();
  await act(async () => restoreButton().click());
  expect(restore).toHaveBeenCalledWith("");
  expect(close).not.toHaveBeenCalled();
  await act(async () => resolve());
  expect(close).toHaveBeenCalledOnce();
});
it("unread HTML cannot be restored as an invented empty description", async () => {
  const restore = vi.fn(async () => {});
  await render(undefined, restore);
  expect(restoreButton().disabled).toBe(true);
  await act(async () => restoreButton().click());
  expect(restore).not.toHaveBeenCalled();
});

it("keeps a revalidation error visible beside cached HTML with a read-only retry", async () => {
  await render("cached HTML", async () => {}, new Error("read refused"));
  expect(document.querySelector("[data-html]")?.textContent).toBe("cached HTML");
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("common.retry");
  expect(restoreButton().disabled).toBe(true);
});

it("names the real modal with ModalTitle and has no positive tab order", async () => {
  await render("chosen", async () => {});
  const dialog = document.querySelector('[role="dialog"]')!;
  const label = dialog.getAttribute("aria-labelledby");
  expect(label).not.toBeNull();
  expect(document.getElementById(label!)?.textContent).toBe("description_versions.title");
  expect([...dialog.querySelectorAll<HTMLElement>("[tabindex]")].filter((node) => node.tabIndex > 0)).toHaveLength(0);
});
it.each([true, false])("markdownCopyWaitsAndReportsRefusalOnce (accepted=%s)", async (accepted) => {
  await render("chosen", async () => {});
  const copy = document.querySelector<HTMLButtonElement>('[aria-label="common.actions.copy_markdown"]')!;
  await act(async () => {
    copy.click();
    copy.click();
  });
  expect.soft(write.mock.calls).toEqual([["# Chosen\n\n**bold**\n"]]);
  expect.soft(toasts).toHaveLength(0);
  expect(copy.disabled).toBe(true);
  await act(async () =>
    accepted ? clipboard.answer() : clipboard.refuse(new DOMException("denied", "NotAllowedError"))
  );
  expect(toasts).toHaveLength(1);
  expect(toasts[0]?.type).toBe(accepted ? "success" : "error");
  expect(binding.legacyCopy).not.toHaveBeenCalled();
  expect(copy.disabled).toBe(false);
});
it("does not invent Markdown before the editor is ready", async () => {
  binding.ready = false;
  await render("chosen", async () => {});
  const copy = document.querySelector<HTMLButtonElement>('[aria-label="common.actions.copy_markdown"]')!;
  await act(async () => copy.click());
  expect(write).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
  await render(undefined, async () => {});
  expect(copy.disabled).toBe(true);
});
it("historyRefusalStaysOpenAndLateReadCannotReplaceSelection holds focus while restore is pending", async () => {
  const pending = heldChange<void>();
  await render("chosen", () => pending.sent);
  await act(async () => {
    restoreButton().focus();
    restoreButton().click();
  });
  const dialog = document.querySelector<HTMLElement>('[role="dialog"]')!;
  const event = new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true });
  await act(async () => dialog.dispatchEvent(event));
  expect(close).not.toHaveBeenCalled();
  expect(event.defaultPrevented).toBe(true);
  await act(async () => pending.refuse(new Error("refused")));
  expect(close).not.toHaveBeenCalled();
});
