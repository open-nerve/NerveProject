/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ComponentProps } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
import { projectMemberDetailsOf, projectOf } from "@/store/project/fake-projects";
import type { ConfirmProjectMemberRemove } from "./confirm-project-member-remove";
import { ProjectMemberListItem } from "./member-list-item";

// How the members page closes the dialog that asked for a removal or the caller's leaving (M3 design 7.1, 7.6; spec
// §3 #6, A-M2): only once nerve has made the change, in the session it was sent in. The rows render on the server for
// web, with stand-ins for the columns' hook, which shows the dialog of the row the test asks of and records its
// closing, for the dialog, which keeps the props it was given, and for the stores' changes, which nerve answers when
// the test says. The caller is me, an admin of web. Its session is fake-tab.ts's.

const web = projectOf("WEB", "w-acme", { member_role: 20 });
const page = vi.hoisted(() => ({
  removeMemberFromProject: vi.fn(),
  leaveProject: vi.fn(),
  navigate: vi.fn(),
  setRemoveMemberModal: vi.fn(),
}));
/** The row whose dialog shows, as the columns' hook gives it. */
const shownDialog = vi.hoisted((): { of: IProjectMemberDetails | null } => ({ of: null }));
/** The props the dialog was given, in the order rendered. */
const dialogs = vi.hoisted((): ComponentProps<typeof ConfirmProjectMemberRemove>[] => []);
vi.mock("@/components/projects/settings/useProjectColumns", () => ({
  useProjectColumns: () => ({
    columns: [],
    removeMemberModal: shownDialog.of,
    setRemoveMemberModal: page.setRemoveMemberModal,
  }),
}));
vi.mock("./confirm-project-member-remove", () => ({
  ConfirmProjectMemberRemove: (props: ComponentProps<typeof ConfirmProjectMemberRemove>) => {
    dialogs.push(props);
    return null;
  },
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "u-me" } }) }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) => (projectId === web.id ? web : undefined),
    leaveProject: page.leaveProject,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ project: { updateMemberRole: vi.fn(), removeMemberFromProject: page.removeMemberFromProject } }),
}));
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@nerve/ui", () => ({ Table: () => null }));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** Each request a dialog asks for: another's removal, from bob's row; the caller's leaving, from his own. */
const asked = [
  { request: "a removal", row: projectMemberDetailsOf(web, "bob", 15), store: page.removeMemberFromProject },
  { request: "the leaving", row: projectMemberDetailsOf(web, "me", 20), store: page.leaveProject },
];

/** Renders web's rows with the dialog of row showing, and confirms it: settles once the page has followed it. */
function confirm(row: IProjectMemberDetails) {
  shownDialog.of = row;
  dialogs.length = 0;
  renderToStaticMarkup(<ProjectMemberListItem memberDetails={[row]} projectId={web.id} workspaceSlug="acme" />);
  const [dialog] = dialogs;
  if (!dialog) throw new Error("the page showed no dialog");
  return dialog.onSubmit();
}

beforeEach(() => {
  signedIn();
  for (const store of [page.removeMemberFromProject, page.leaveProject]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  page.navigate.mockReset();
  page.setRemoveMemberModal.mockReset();
  toasts.length = 0;
});

describe("ProjectMemberListItem", () => {
  it.each(asked)("closes the dialog that asked for $request once nerve has made it", async ({ row, store }) => {
    await confirm(row);
    expect([store.mock.calls.length, page.setRemoveMemberModal.mock.calls, toasts]).toEqual([1, [[null]], []]);
  });

  it.each(asked)("keeps the dialog that asked for $request open when nerve refuses it", async ({ row, store }) => {
    store.mockRejectedValueOnce(refusal(403, "forbidden"));
    await confirm(row);
    expect(page.setRemoveMemberModal).not.toHaveBeenCalled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  describe.each(asked)("$request", ({ row, store }) => {
    it.each(lateSettlings)(
      "neither closes its dialog nor says anything when it $settles after another tab moved this one",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        const confirmed = confirm(row);
        switchAccount();
        settle(change);
        await confirmed;
        expect([page.setRemoveMemberModal.mock.calls, toasts]).toEqual([[], []]);
      }
    );
  });
});
