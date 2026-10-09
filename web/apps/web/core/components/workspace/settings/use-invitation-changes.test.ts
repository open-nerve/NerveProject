/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { useInvitationChanges } from "./use-invitation-changes";

// What the members page sends for each change of an invitation, and what it does with nerve's answer (M3 design 7.1,
// 7.5): the hook runs as a plain function, with stand-ins for the store's changes, which nerve answers when the test
// says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ updateMemberInvitation: vi.fn(), deleteMemberInvitation: vi.fn() }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      updateMemberInvitation: page.updateMemberInvitation,
      deleteMemberInvitation: page.deleteMemberInvitation,
    },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Each change the page makes, by its store change: the role's and the deletion's. */
const changes = [
  {
    change: "a role",
    store: page.updateMemberInvitation,
    make: () => useInvitationChanges().changeRole("acme", "i-dan", 5),
  },
  {
    change: "a deletion",
    store: page.deleteMemberInvitation,
    make: () => useInvitationChanges().remove("acme", "i-dan"),
  },
];

beforeEach(() => {
  signedIn();
  for (const store of [page.updateMemberInvitation, page.deleteMemberInvitation]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  toasts.length = 0;
});

describe("useInvitationChanges", () => {
  it("sends a pending invitation's new role as its number, to the invitation's workspace", async () => {
    await useInvitationChanges().changeRole("acme", "i-dan", 5);
    expect([page.updateMemberInvitation.mock.calls, toasts]).toEqual([[["acme", "i-dan", { role: 5 }]], []]);
  });

  it("deletes an invitation of the workspace, and says so", async () => {
    await useInvitationChanges().remove("acme", "i-dan");
    expect(page.deleteMemberInvitation.mock.calls).toEqual([["acme", "i-dan"]]);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Invitation removed successfully." }]);
  });

  it.each(changes)("shows nerve's reason when it refuses $change", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "workspace.invitation_responded"));
    await make();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_invitation_responded" }]);
  });

  describe.each(changes)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        const made = make();
        switchAccount();
        settle(change);
        await made;
        expect(toasts).toEqual([]);
      }
    );
  });
});
