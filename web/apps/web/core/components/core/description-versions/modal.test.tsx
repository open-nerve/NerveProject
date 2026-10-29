// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api-error";
import { DescriptionVersionsModal } from "./modal";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } } }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => ({ getWorkspaceBySlug: () => ({ id: "ws" }) }) }));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: ({ initialValue }: { initialValue: string }) => <div data-html>{initialValue}</div>,
}));
vi.mock("@nerve/ui", () => ({
  ModalCore: ({ children, isOpen }: React.PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
  EModalPosition: {},
  EModalWidth: {},
  Loader: { Item: () => <span>loading</span> },
}));
vi.mock("@nerve/propel/button", () => ({
  Button: ({ children, onClick, disabled }: React.PropsWithChildren<{ onClick?: () => void; disabled?: boolean }>) => (
    <button onClick={onClick} disabled={disabled}>
      {children}
    </button>
  ),
}));
vi.mock("@nerve/propel/icon-button", () => ({ IconButton: () => null }));
vi.mock("@makeplane/propel/components/avatar", () => ({ Avatar: () => null }));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: React.PropsWithChildren) => children,
}));
let renderer: Root;
let container: HTMLElement;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
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
  [...container.querySelectorAll<HTMLButtonElement>("button")].find(
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
  expect(container.querySelector("[data-html]")?.textContent).toBe("<p>chosen</p>");
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  expect(restoreButton().disabled).toBe(false);
});
it("empty HTML is a loaded value and success closes only after confirmation", async () => {
  let resolve!: () => void;
  const promise = new Promise<void>((yes) => {
    resolve = yes;
  });
  const restore = vi.fn(() => promise);
  await render("", restore);
  expect(container.querySelector("[data-html]")).not.toBeNull();
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
  expect(container.querySelector("[data-html]")?.textContent).toBe("cached HTML");
  expect(container.querySelector('[role="alert"]')?.textContent).toContain("common.retry");
  expect(restoreButton().disabled).toBe(true);
});
