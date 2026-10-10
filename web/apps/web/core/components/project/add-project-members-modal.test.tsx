/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectMember, ProjectMembersAdd } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { projectMemberOf, projectOf } from "@/store/project/fake-projects";
import { AddProjectMembersModal } from "./add-project-members-modal";

// Whom the project's add-members modal offers, and what it does with nerve's answer (M3 design 2 P5, 3.5, 7.1, 7.6):
// the modal renders on the server with stand-ins for the UI kit's controls (fake-controls.ts), which keep the props
// they were given, and for the member stores: of acme's members, bob is web's already and sid's membership ended;
// ann, wes (its admin) and gus (its guest) are not web's. nerve answers the adding at once, unless the test refuses
// or holds it. Its session is fake-tab.ts's.

const web = projectOf("WEB", "w-acme");
const acme = [
  membershipOf("ann"),
  membershipOf("bob"),
  membershipOf("sid", { is_active: false }),
  membershipOf("wes", { role: 20 }),
  membershipOf("gus", { role: 5 }),
];
const page = vi.hoisted(() => ({ bulkAddMembersToProject: vi.fn(), onClose: vi.fn() }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      getProjectMemberDetails: (userId: string) => (userId === "u-bob" ? {} : null),
      bulkAddMembersToProject: page.bulkAddMembersToProject,
    },
    workspace: {
      workspaceMemberIds: acme.map((membership) => membership.member.id),
      getWorkspaceMemberDetails: (userId: string) => acme.find((membership) => membership.member.id === userId) ?? null,
    },
  }),
}));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** Renders the modal and gives its member select. */
function render() {
  emptyShown();
  renderToStaticMarkup(<AddProjectMembersModal isOpen onClose={page.onClose} projectId={web.id} />);
  const [members] = shown.searchSelects;
  if (!members) throw new Error("the modal showed no member select");
  return members;
}

/** Renders the modal, picks ann, and submits it: settles once the modal has followed nerve's answer. */
function addAnn() {
  render().onChange("u-ann");
  return submitModalForm();
}

beforeEach(() => {
  signedIn();
  page.bulkAddMembersToProject.mockReset();
  page.bulkAddMembersToProject.mockImplementation((_projectId: string, _data: ProjectMembersAdd) =>
    Promise.resolve([])
  );
  page.onClose.mockReset();
  toasts.length = 0;
});

describe("AddProjectMembersModal", () => {
  it("offers the workspace's active members who are not the project's", () => {
    expect(render().options?.map((option) => option.value)).toEqual(["u-ann", "u-wes", "u-gus"]);
  });

  it("adds them, then closes and says so", async () => {
    await addAnn();
    expect(page.bulkAddMembersToProject.mock.calls).toEqual([
      [web.id, { members: [{ member_id: "u-ann", role: 15 }] }],
    ]);
    expect(page.onClose).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Members added successfully." }]);
  });

  it("stays open, keeps the picks and shows nerve's reason when it refuses them", async () => {
    page.bulkAddMembersToProject.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "members[0].member_id", code: "duplicate" }])
    );
    await addAnn();
    expect(page.onClose).not.toHaveBeenCalled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.validation_failed" }]);
    // the form still has ann as a member: submitted again, it sends her again
    await submitModalForm();
    const addingAnn = [web.id, { members: [{ member_id: "u-ann", role: 15 }] }];
    expect(page.bulkAddMembersToProject.mock.calls).toEqual([addingAnn, addingAnn]);
  });

  // nerve answers the adding with ann's membership: a close that waited for it, past the session's check, would show
  it.each(lateSettlingsAnswering([projectMemberOf(web, "ann")]))(
    "neither closes nor says anything when the adding $settles after another tab moved this one",
    async ({ settle }) => {
      const adding = heldChange<ProjectMember[]>();
      page.bulkAddMembersToProject.mockReturnValueOnce(adding.sent);
      const added = addAnn();
      // the form's checks come first: the adding leaves once they pass
      await vi.waitFor(() => expect(page.bulkAddMembersToProject).toHaveBeenCalled());
      switchAccount();
      settle(adding);
      await added;
      expect([page.onClose.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
