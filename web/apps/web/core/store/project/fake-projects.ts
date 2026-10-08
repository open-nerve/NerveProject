/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The caller's projects, for the tests of the stores that read them, against a fake nerve: nerve's records, the
// fetches that load them into a store, and a tab whose stores have the caller's workspaces and projects.

import type {
  Label,
  Project,
  ProjectMember,
  ProjectPreferences,
  ProjectRole,
  State,
  StateGroup,
  Workspace,
} from "@nerve/api-client";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
import { ProjectRootStore } from "@/store/project";
import type { IProjectStore } from "@/store/project/project.store";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";

/**
 * A project of the workspace workspaceId names, as nerve gives it: the identifier names it; public, with the caller
 * as its member, first in his sidebar, unless fields say otherwise.
 */
export function projectOf(identifier: string, workspaceId: string, fields: Partial<Project> = {}): Project {
  return {
    id: `p-${identifier.toLowerCase()}`,
    workspace_id: workspaceId,
    name: identifier,
    description: "",
    identifier,
    network: 2,
    project_lead_id: null,
    default_assignee_id: null,
    cycle_view: true,
    module_view: true,
    issue_views_view: true,
    intake_view: false,
    guest_view_all_features: false,
    archive_in: 0,
    archived_at: null,
    logo_props: {},
    timezone: "UTC",
    cover_image_url: null,
    member_role: 15,
    sort_order: 1000,
    member_ids: [],
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/**
 * The caller's settings in a project as nerve gives them: its default tab bar, and the project's place in his
 * sidebar, unless fields say otherwise.
 */
export function preferencesOf(fields: Partial<ProjectPreferences> = {}): ProjectPreferences {
  return { navigation: { default_tab: "work_items", hide_in_more_menu: [] }, sort_order: 65535, ...fields };
}

/** A state of the project as nerve gives it: the name names it in its project; not its default unless fields say so. */
export function stateOf(
  project: Project,
  name: string,
  group: StateGroup,
  sequence: number,
  fields: Partial<State> = {}
): State {
  return {
    id: `s-${project.identifier}-${name}`,
    workspace_id: project.workspace_id,
    project_id: project.id,
    name,
    description: "",
    color: "#60646C",
    group,
    default: false,
    sequence,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** A label of the project as nerve gives it: the name names it in its project; at the top unless fields say otherwise. */
export function labelOf(project: Project, name: string, sortOrder: number, fields: Partial<Label> = {}): Label {
  return {
    id: `l-${project.identifier}-${name}`,
    workspace_id: project.workspace_id,
    project_id: project.id,
    parent_id: null,
    name,
    color: "#F59E0B",
    sort_order: sortOrder,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** A membership of the project as nerve lists it: the name names the member and the membership; a member's role. */
export function projectMemberOf(project: Pick<Project, "id">, name: string, role: ProjectRole = 15): ProjectMember {
  return {
    id: `pm-${project.id}-${name}`,
    project_id: project.id,
    member_id: `u-${name}`,
    role,
    created_at: "2026-10-01T09:00:00Z",
  };
}

/** The store fetches the workspace's projects that are not archived, and nerve lists these. */
export function loadProjects(
  nerve: FakeNerve,
  store: IProjectStore,
  workspace: Pick<Workspace, "id" | "slug">,
  projects: Project[]
) {
  return answered(
    nerve,
    () => store.fetchProjects(workspace),
    ["GET", `/api/v0/workspaces/${workspace.slug}/projects`],
    { data: projects },
    "the projects"
  );
}

/** The store fetches the workspace's archived projects, and nerve lists these. */
export function loadArchivedProjects(
  nerve: FakeNerve,
  store: IProjectStore,
  workspace: Pick<Workspace, "id" | "slug">,
  projects: Project[]
) {
  return answered(
    nerve,
    () => store.fetchArchivedProjects(workspace),
    ["GET", `/api/v0/workspaces/${workspace.slug}/projects`],
    { data: projects },
    "the archived projects"
  );
}

/** The store reads the project alone, and nerve gives read, the project unless it says. */
export function loadProject(nerve: FakeNerve, store: IProjectStore, project: Project, read: Project = project) {
  const path = `/api/v0/projects/${project.id}`;
  return answered(nerve, () => store.fetchProject(project.id), ["GET", path], read, "the read");
}

/** A workspace of the caller's, and its projects that are not archived as nerve listed them, unless none were fetched. */
type Listed = { workspace: Workspace; projects?: Project[] };

/**
 * A tab whose stores of the workspaces and the projects are the pages': nerve listed the caller's two workspaces,
 * here's and elsewhere's, and the projects of each that the tab fetched. Its address is here's first project's, or
 * here's workspace's when it fetched none; the requests so far are forgotten. Two workspaces, so that a store that
 * keeps a workspace's records where another's are fails a test (P8a's review, its third lesson).
 */
export async function projectTab(here: Listed, elsewhere: Listed) {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  // after the stores, whose reactions follow the address (the projects page's filters)
  router.setQuery({ workspaceSlug: here.workspace.slug, projectId: here.projects?.[0]?.id });
  await loadWorkspaces(nerve, workspaceRoot, [here.workspace, elsewhere.workspace]);
  if (here.projects) await loadProjects(nerve, projectRoot.project, here.workspace, here.projects);
  if (elsewhere.projects) await loadProjects(nerve, projectRoot.project, elsewhere.workspace, elsewhere.projects);
  nerve.calls.length = 0;
  return { nerve, api, router, workspaceRoot, projectRoot };
}
