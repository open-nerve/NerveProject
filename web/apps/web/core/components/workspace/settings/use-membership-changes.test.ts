/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useMembershipChanges } from "./use-membership-changes";

// What the members page sends for each change of a membership, and what it does with nerve's answer (M3 design 7.1,
// 7.5): the hook runs as a plain function, with stand-ins for the stores' changes, which nerve answers when the test
// says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({
  updateMember: vi.fn(),
  removeMemberFromWorkspace: vi.fn(),
  leaveWorkspace: vi.fn(),
  navigate: vi.fn(),
}));
const acme = workspaceOf("acme", { role: 20 });
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) => (slug === acme.slug ? acme : null),
    leaveWorkspace: page.leaveWorkspace,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: { updateMember: page.updateMember, removeMemberFromWorkspace: page.removeMemberFromWorkspace },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Each change the page makes, by its store change: the role's, the removal's and the leaving's. */
const changes = [
  { change: "a role", store: page.updateMember, make: () => useMembershipChanges("acme").changeRole("u-bob", 5) },
  {
    change: "a removal",
    store: page.removeMemberFromWorkspace,
    make: () => useMembershipChanges("acme").remove("u-bob"),
  },
  { change: "the leaving", store: page.leaveWorkspace, make: () => useMembershipChanges("acme").leave() },
];

beforeEach(() => {
  signedIn();
  for (const store of [page.updateMember, page.removeMemberFromWorkspace, page.leaveWorkspace]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  page.navigate.mockClear();
  toasts.length = 0;
});

describe("useMembershipChanges", () => {
  it("sends a member's new role as its number, to the membership's workspace", async () => {
    await useMembershipChanges("acme").changeRole("u-bob", 5);
    expect(page.updateMember.mock.calls).toEqual([["acme", "u-bob", { role: 5 }]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
  });

  it("removes a member of the workspace, and stays", async () => {
    await useMembershipChanges("acme").remove("u-bob");
    expect(page.removeMemberFromWorkspace.mock.calls).toEqual([["acme", "u-bob"]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
  });

  it("leaves the workspace of the address, then lands at the root", async () => {
    await useMembershipChanges("acme").leave();
    expect([page.leaveWorkspace.mock.calls, page.navigate.mock.calls]).toEqual([[[acme]], [["/"]]]);
  });

  it.each(changes)("shows nerve's reason when it refuses $change, and stays", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "workspace.sole_admin"));
    await make();
    expect(page.navigate.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_sole_admin" }]);
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
        expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
      }
    );
  });
});
