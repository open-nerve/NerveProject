/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useProjectMembershipChanges } from "./use-project-membership-changes";

// What a project's pages send for each change of a membership, and what they do with nerve's answer (M3 design 7.1,
// 7.6, M1-P4): the hook runs as a plain function, with stand-ins for the stores' changes, which nerve answers when the
// test says, for the router's navigate, and for the closing of the dialog that sent a removal or the leaving. Its
// session is fake-tab.ts's.

const page = vi.hoisted(() => ({
  updateMemberRole: vi.fn(),
  removeMemberFromProject: vi.fn(),
  leaveProject: vi.fn(),
  navigate: vi.fn(),
  closeDialog: vi.fn(),
}));
const web = projectOf("WEB", "w-acme", { member_role: 20 });
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) => (projectId === web.id ? web : undefined),
    leaveProject: page.leaveProject,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: { updateMemberRole: page.updateMemberRole, removeMemberFromProject: page.removeMemberFromProject },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const changes = () => useProjectMembershipChanges("acme", web.id);
/** Each change the pages make, by its store change: the role's, the removal's and the leaving's. */
const made = [
  { change: "a role", store: page.updateMemberRole, make: () => changes().changeRole("u-bob", 5) },
  { change: "a removal", store: page.removeMemberFromProject, make: () => changes().remove("u-bob", page.closeDialog) },
  { change: "the leaving", store: page.leaveProject, make: () => changes().leave(page.closeDialog) },
];

beforeEach(() => {
  signedIn();
  for (const store of [page.updateMemberRole, page.removeMemberFromProject, page.leaveProject]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  page.navigate.mockReset();
  page.closeDialog.mockReset();
  toasts.length = 0;
});

describe("useProjectMembershipChanges", () => {
  it("sends a member's new role as its number, to the membership's project", async () => {
    await changes().changeRole("u-bob", 5);
    expect(page.updateMemberRole.mock.calls).toEqual([[web.id, "u-bob", 5]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
  });

  it("removes a member of the project, and stays, its dialog closed once nerve has", async () => {
    const removal = heldChange<undefined>();
    page.removeMemberFromProject.mockReturnValueOnce(removal.sent);
    const removed = changes().remove("u-bob", page.closeDialog);
    expect(page.removeMemberFromProject.mock.calls).toEqual([[web.id, "u-bob"]]);
    expect(page.closeDialog).not.toHaveBeenCalled();
    removal.answer(undefined);
    await removed;
    expect([page.closeDialog.mock.calls.length, page.navigate.mock.calls, toasts]).toEqual([1, [], []]);
  });

  it("leaves the project, and closes its dialog and shows the workspace's projects only once nerve has made it, settling once they show", async () => {
    const leaving = heldChange<undefined>();
    page.leaveProject.mockReturnValueOnce(leaving.sent);
    const shown = heldChange<undefined>();
    page.navigate.mockReturnValueOnce(shown.sent);
    const left = changes().leave(page.closeDialog);
    const settled = settledYet(left);
    expect(page.leaveProject.mock.calls).toEqual([[web]]);
    expect([page.closeDialog.mock.calls, page.navigate.mock.calls]).toEqual([[], []]);
    leaving.answer(undefined);
    await vi.waitFor(() => expect(page.navigate.mock.calls).toEqual([["/acme/projects"]]));
    expect([page.closeDialog.mock.calls.length, settled()]).toEqual([1, false]);
    shown.answer(undefined);
    await left;
  });

  it.each(made)("shows nerve's reason when it refuses $change, and stays, its dialog open", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "project.sole_admin"));
    await make();
    expect([page.closeDialog.mock.calls, page.navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_sole_admin" }]);
  });

  describe.each(made)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        const making = make();
        switchAccount();
        settle(change);
        await making;
        expect([page.closeDialog.mock.calls, page.navigate.mock.calls, toasts]).toEqual([[], [], []]);
      }
    );
  });
});
