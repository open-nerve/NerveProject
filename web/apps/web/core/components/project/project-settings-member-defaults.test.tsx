/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectRole, ProjectUpdate, WorkspaceRole } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import type { ProjectToggleField } from "@/store/project/project.store";
import { callerInWeb } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ProjectSettingsMemberDefaults } from "./project-settings-member-defaults";

// What the project's member defaults send (v0 design 7.7: a change whose body depends on what the store holds is
// built in its turn), what the page says of them (M3 design 7.1), and whom it lets change them (3.4): the page renders
// on the server with stand-ins for the member selects and the guests' switch (fake-controls.ts), which keep the props
// they were given, and for the project store, whose changes nerve has not answered yet unless the test says. Its
// session is fake-tab.ts's; the caller's permissions are the store's own, over acme and web as nerve lists them to him
// (fake-permissions.ts): web's admin, unless the test says.

type Select = { onChange: (value: string) => void; isDisabled?: boolean };
const page = vi.hoisted(() => {
  const selects: Select[] = [];
  // nerve has not answered: a change stays out, unless a test answers it
  const updateProject = vi.fn((_projectId: string, _data: ProjectUpdate): Promise<unknown> => new Promise(() => {}));
  const toggleProject = vi.fn(
    (_projectId: string, _field: ProjectToggleField): Promise<unknown> => new Promise(() => {})
  );
  return { selects, updateProject, toggleProject };
});
const caller = vi.hoisted(() => {
  const held: { permissions?: IUserPermissionStore } = {};
  return held;
});
const web: Project = projectOf("WEB", "id-acme", { project_lead_id: "u-bob", default_assignee_id: "u-cat" });
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: () => web,
    updateProject: page.updateProject,
    toggleProject: page.toggleProject,
  }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => caller.permissions }));
vi.mock("./member-select", () => ({
  MemberSelect: (props: Select) => {
    page.selects.push(props);
    return null;
  },
}));
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** The page's controls, as render gives them. */
type Shown = ReturnType<typeof render>;

/** Renders the page: gives its two member selects, the lead's then the default assignee's, and its switch. */
function render() {
  page.selects.length = 0;
  emptyShown();
  renderToStaticMarkup(<ProjectSettingsMemberDefaults workspaceSlug="acme" projectId={web.id} />);
  const [lead, assignee] = page.selects;
  const [guests] = shown.switches;
  if (!lead || !assignee || !guests) throw new Error("the page showed no member selects or no switch");
  return { lead, assignee, guests };
}

beforeEach(async () => {
  signedIn();
  caller.permissions = (await callerInWeb(15, 20)).permissions;
  page.updateProject.mockClear();
  page.toggleProject.mockClear();
  toasts.length = 0;
});

describe("the project's member defaults", () => {
  it("changes the lead, then the default assignee, each alone: nerve keeps the other as it has it", () => {
    const { lead, assignee } = render();
    lead.onChange("u-ann");
    // the lead's change is still out: nerve has not answered it
    assignee.onChange("none");
    expect(page.updateProject.mock.calls).toEqual([
      [web.id, { project_lead_id: "u-ann" }],
      [web.id, { default_assignee_id: null }],
    ]);
  });

  it("turns the guests' view with the store's toggle, from nerve's last answer, not the switch's value", () => {
    const { guests } = render();
    guests.onCheckedChange(true);
    expect(page.toggleProject.mock.calls).toEqual([[web.id, "guest_view_all_features"]]);
    expect(page.updateProject).not.toHaveBeenCalled();
  });

  it("says a change is made, and nerve's reason for refusing one", async () => {
    page.updateProject.mockResolvedValueOnce(web);
    page.updateProject.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "project_lead_id", code: "not_allowed" }])
    );
    const { lead } = render();
    lead.onChange("u-ann");
    lead.onChange("u-gus");
    await vi.waitFor(() => expect(toasts).toHaveLength(2));
    expect(toasts).toEqual([
      { type: "success", title: "success!", message: "project_settings.general.toast.success" },
      { type: "error", title: "toast.error", message: "errors.validation_failed" },
    ]);
  });

  // each of the page's three changes, sent before another tab moved this one and settled after; nerve answers with the
  // project, as it would
  describe.each([
    { change: "the lead's change", send: ({ lead }: Shown) => lead.onChange("u-ann"), store: page.updateProject },
    {
      change: "the default assignee's change",
      send: ({ assignee }: Shown) => assignee.onChange("u-ann"),
      store: page.updateProject,
    },
    {
      change: "the guests' view's change",
      send: ({ guests }: Shown) => guests.onCheckedChange(true),
      store: page.toggleProject,
    },
  ])("$change", ({ send, store }) => {
    it.each(lateSettlingsAnswering(web))(
      "says nothing when it $settles after another tab moved this one",
      async ({ settle }) => {
        const change = heldChange<Project>();
        store.mockReturnValueOnce(change.sent);
        send(render());
        switchAccount();
        settle(change);
        await pageSettled();
        expect(toasts).toEqual([]);
      }
    );
  });

  // whom nerve lets change the project (project.update: its admins, and its members who are the workspace's admins)
  it.each<{ who: string; workspaceRole: WorkspaceRole; projectRole: ProjectRole | null; changes: boolean }>([
    { who: "its admin", workspaceRole: 15, projectRole: 20, changes: true },
    { who: "its member who is the workspace's admin", workspaceRole: 20, projectRole: 15, changes: true },
    { who: "its member", workspaceRole: 15, projectRole: 15, changes: false },
    { who: "its guest", workspaceRole: 5, projectRole: 5, changes: false },
    { who: "the workspace's admin who is not its member", workspaceRole: 20, projectRole: null, changes: false },
  ])("lets $who change them when nerve does", async ({ workspaceRole, projectRole, changes }) => {
    caller.permissions = (await callerInWeb(workspaceRole, projectRole)).permissions;
    const { lead, assignee, guests } = render();
    expect([lead.isDisabled, assignee.isDisabled, guests.disabled]).toEqual([!changes, !changes, !changes]);
  });
});
