/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { createRoutesStub } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { IProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import LabelsPage from "./labels/page";
import StatesPage from "./states/page";

// Whom a project's state and label settings show what (M3 design 2 P6, P7, W11, 9.2): its states and its labels to
// those nerve lets list them, but its guests, whom each page tells they may not (state.list and label.list: its
// members, whatever their role; design 2 W11); the controls that change them to those nerve lets change them
// (state.create, label.create and the others: its admins, and its members who are the workspace's admins, its guests
// among them). A page renders on the server at its route, through React Router's stub, which gives it the route's
// params, with stand-ins for its parts that write what they show: the list of groups writes the states it is given and
// a label its name, each whether they have their controls, the label list's heading and its empty state show their
// buttons. The caller's permissions and projects are the stores' own, over acme and its project web as nerve lists
// them to him (fake-permissions.ts), while the address the router store holds names no project: the page reads its
// project from its route. The state and label stores give web's and none of another project's, so that a page that
// reads another's shows its loader.

const caller = vi.hoisted(
  (): { permissions?: IUserPermissionStore; projects?: IProjectStore; projectId?: string } => ({})
);
/** web's states, by group, as the state store groups them. */
const grouped = vi.hoisted(() => ({ backlog: [{ name: "Backlog" }], started: [{ name: "In Progress" }] }));
/** web's one label, as the label store gives it. */
const bug = vi.hoisted(() => ({ id: "l-bug", name: "Bug", parent_id: null }));
/** web's labels as the label store lists them: Bug, but in the test of a project with none. */
const listed = vi.hoisted(() => ({ labels: [bug] }));
beforeEach(() => {
  listed.labels = [bug];
});
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => caller.permissions }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => caller.projects }));
vi.mock("@/hooks/store/use-project-state", () => ({
  useProjectState: () => ({
    getGroupedProjectStates: (projectId: string) => (projectId === caller.projectId ? grouped : undefined),
  }),
}));
vi.mock("@/hooks/store/use-label", () => ({
  useLabel: () => ({
    getProjectLabels: (projectId: string) => (projectId === caller.projectId ? listed.labels : undefined),
    getProjectLabelsTree: (projectId: string) =>
      projectId === caller.projectId
        ? listed.labels.map((label) => ({ id: label.id, name: label.name, parent_id: label.parent_id, children: [] }))
        : undefined,
  }),
}));
vi.mock("@/components/labels/project-setting-label-item", () => ({
  ProjectSettingLabelItem: ({ label, isEditable }: { label: { name: string }; isEditable: boolean }) =>
    `${label.name}${isEditable ? " and its controls" : ""}`,
}));
vi.mock("@/components/labels/use-label-changes", () => ({ useLabelChanges: () => ({}) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/components/project-states/group-list", () => ({
  GroupList: ({
    groupedStates,
    isEditable,
  }: {
    groupedStates: Record<string, { name: string }[]>;
    isEditable: boolean;
  }) =>
    `${Object.values(groupedStates)
      .flat()
      .map((state) => state.name)
      .join(", ")}${isEditable ? " and their controls" : ""}`,
}));
vi.mock("@/components/auth-screens/not-authorized-view", () => ({ NotAuthorizedView: () => "not authorized" }));
vi.mock("@/components/core/page-title", () => ({ PageHead: ({ title }: { title?: string }) => `[${title}]` }));
vi.mock("@/components/settings/content-wrapper", () => ({
  SettingsContentWrapper: ({ children }: { children: unknown }) => children,
}));
vi.mock("@/components/settings/heading", () => ({ SettingsHeading: ({ control }: { control?: unknown }) => control }));
vi.mock("@nerve/propel/button", () => ({ Button: ({ children }: { children: unknown }) => `[${children}]` }));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateCompact: ({ actions }: { actions?: { label: string }[] }) =>
    (actions ?? []).map((action) => `[${action.label}]`).join(""),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** A route of React Router's stub, whose component gets the route's props as a route module's does. */
type StubRoute = Parameters<typeof createRoutesStub>[0][number];

/**
 * Signs the caller in as nerve lists acme and web to him (callerInWeb), moves the router store's address to acme with
 * no project, and opens the page at its route under web's settings: the text it shows him, and web's name.
 */
async function openedBy({ workspaceRole, projectRole }: Caller, route: string, Component: StubRoute["Component"]) {
  const { permissions, router, projects, acme, web } = await callerInWeb(workspaceRole, projectRole);
  Object.assign(caller, { permissions, projects, projectId: web.id });
  router.setQuery({ workspaceSlug: acme.slug });
  const Routes = createRoutesStub([{ path: `/:workspaceSlug/settings/projects/:projectId/${route}`, Component }]);
  const markup = renderToStaticMarkup(
    <Routes initialEntries={[`/${acme.slug}/settings/projects/${web.id}/${route}`]} />
  );
  return { text: markup.replace(/<[^>]*>/g, ""), name: web.name };
}

describe("the state settings page", () => {
  it.each([its.admin, its.memberAdmin, its.guestAdmin])("shows $who the states and their controls", async (row) => {
    const { text, name } = await openedBy(row, "states", StatesPage);
    expect(text).toBe(`[${name} - common.states]Backlog, In Progress and their controls`);
  });

  it("shows its member the states, and no control", async () => {
    const { text, name } = await openedBy(its.member, "states", StatesPage);
    expect(text).toBe(`[${name} - common.states]Backlog, In Progress`);
  });

  it("tells its guest he may not see them", async () => {
    const { text } = await openedBy(its.guest, "states", StatesPage);
    expect(text).toBe("not authorized");
  });
});

describe("the label settings page", () => {
  it.each([its.admin, its.memberAdmin, its.guestAdmin])(
    "shows $who the labels, their controls and its button",
    async (row) => {
      const { text, name } = await openedBy(row, "labels", LabelsPage);
      expect(text).toBe(`[${name} - common.labels][common.add_label]Bug and its controls`);
    }
  );

  it("shows its member the labels, and no control", async () => {
    const { text, name } = await openedBy(its.member, "labels", LabelsPage);
    expect(text).toBe(`[${name} - common.labels]Bug`);
  });

  it("tells its guest he may not see them", async () => {
    const { text } = await openedBy(its.guest, "labels", LabelsPage);
    expect(text).toBe("not authorized");
  });

  it("shows the button of its empty state, with no label, to those who may add one alone", async () => {
    listed.labels = [];
    const admin = await openedBy(its.admin, "labels", LabelsPage);
    const member = await openedBy(its.member, "labels", LabelsPage);
    expect([admin.text, member.text]).toEqual([
      `[${admin.name} - common.labels][common.add_label][settings_empty_state.labels.cta_primary]`,
      `[${member.name} - common.labels]`,
    ]);
  });
});
