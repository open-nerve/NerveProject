/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pageSettled } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { DeactivateAccountModal } from "./deactivate-account-modal";

// What the deactivation dialog does with nerve's answer (M3 design 7.1, 7.5, W9): the dialog renders on the server,
// with stand-ins for its modal and buttons (fake-controls.ts), and the test confirms through the confirm button. The
// user store's deactivation is a stand-in, which resolves whether it ended the tab's session.

const store = vi.hoisted(() => ({ deactivateAccount: vi.fn<() => Promise<boolean>>() }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ deactivateAccount: store.deactivateAccount }) }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the open dialog and confirms it, as its confirm button would; gives onClose once the dialog has followed. */
async function confirm() {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeactivateAccountModal isOpen onClose={onClose} />);
  shown.buttons.find((button) => button.variant === "error-fill")?.onClick?.({ preventDefault: () => {} });
  await pageSettled();
  return onClose;
}

beforeEach(() => {
  store.deactivateAccount.mockReset();
  toasts.length = 0;
  emptyShown();
});

describe("DeactivateAccountModal", () => {
  it("says the account is deactivated, and closes, once the deactivation ended the tab's session", async () => {
    store.deactivateAccount.mockResolvedValueOnce(true);
    const onClose = await confirm();
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "account_deactivated" }]);
    expect(onClose.mock.calls).toHaveLength(1);
  });

  it("says nothing, and stays, when it ended no session of the tab's: another tab had moved it to another account", async () => {
    store.deactivateAccount.mockResolvedValueOnce(false);
    const onClose = await confirm();
    expect([toasts, onClose.mock.calls]).toEqual([[], []]);
  });

  // the reason the dialog then shows is W9's to check: the render does not show what is set after it
  it("stays open, with no toast, when nerve refuses", async () => {
    store.deactivateAccount.mockRejectedValueOnce(refusal(409, "workspace.sole_admin"));
    const onClose = await confirm();
    expect([toasts, onClose.mock.calls]).toEqual([[], []]);
  });
});
