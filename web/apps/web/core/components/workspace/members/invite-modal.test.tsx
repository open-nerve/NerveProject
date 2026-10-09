/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, pick, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { SendWorkspaceInvitationModal } from "./invite-modal";

// What the invitation form sends, and what it does with nerve's answer (M3 design 2 W4, 7.1): the modal renders on
// the server, with stand-ins for its inputs, selects and buttons (fake-controls.ts), which keep the props they were
// given; the test types an address and picks a role as a person would, and submits the form. Its session is
// fake-tab.ts's.

const page = vi.hoisted(() => ({ invite: vi.fn(), onClose: vi.fn() }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// The form's title is headless UI's Dialog.Title, which renders only inside its Dialog: the modal's stand-in has none.
vi.mock("@headlessui/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@headlessui/react")>()),
  Dialog: { Title: ({ children }: { children?: ReactNode }) => children },
}));

/** Renders the form, types email in its one row, picks the role labelled role, and submits it. */
async function invite(email: string, role: string) {
  renderToStaticMarkup(<SendWorkspaceInvitationModal isOpen onClose={page.onClose} invite={page.invite} />);
  const [address] = shown.inputs;
  const [roles] = shown.selects;
  if (!address || !roles) throw new Error("the form showed no row");
  address.onChange({ target: { value: email } });
  pick(roles, role);
  await submitModalForm();
}

beforeEach(() => {
  signedIn();
  page.invite.mockReset();
  page.invite.mockResolvedValue([]);
  page.onClose.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("SendWorkspaceInvitationModal", () => {
  it("sends its rows as nerve's WorkspaceInvitationsCreate, each role a number; then closes, and says so", async () => {
    await invite("dave@example.com", "Guest");
    expect(page.invite.mock.calls).toEqual([[{ invitations: [{ email: "dave@example.com", role: 5 }] }]]);
    expect(page.onClose).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([
      {
        type: "success",
        title: "toast.success",
        message: "workspace_settings.settings.members.invitations_sent_successfully",
      },
    ]);
  });

  it("sends nothing for an address that is none", async () => {
    await invite("dave", "Member");
    expect([page.invite.mock.calls, page.onClose.mock.calls, toasts]).toEqual([[], [], []]);
  });

  // the message the form then shows under the row is W4's to check: the render does not show what is set after it
  it("stays open, with no toast, when nerve refuses a row", async () => {
    page.invite.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "invitations[0].email", code: "duplicate" }])
    );
    await invite("dave@example.com", "Member");
    expect([page.onClose.mock.calls, toasts]).toEqual([[], []]);
  });

  it("stays open, and shows nerve's reason, when it refuses the invitations", async () => {
    page.invite.mockRejectedValueOnce(refusal(403, "forbidden"));
    await invite("dave@example.com", "Member");
    expect(page.onClose.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "neither closes nor speaks when the invitations' answer $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const sent = heldChange<undefined>();
      page.invite.mockReturnValueOnce(sent.sent);
      const submitted = invite("dave@example.com", "Member");
      await vi.waitFor(() => expect(page.invite).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(sent);
      await submitted;
      expect([page.onClose.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
