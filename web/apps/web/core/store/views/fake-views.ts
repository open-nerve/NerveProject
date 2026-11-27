/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The views of a project as the tests of its views' pages see them (P6a spec 2.3): views as nerve gives them, the
// requests of them, and the caller's roles in acme and its projects web and ops, as the permission store gives them
// (fake-permissions.ts), with a views' store over a fake nerve, which answers each request when the test says. The
// caller is u-me.

import type { Project, View, Workspace } from "@nerve/api-client";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { preferencesOf, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { permissionsOver, type Caller } from "@/store/user/fake-permissions";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { ViewStore } from "./view.store";

const { display_filters, display_properties } = preferencesOf();

/** The caller, as useUser gives him. */
export const ME = "u-me";

/** A view of the project as nerve gives it, named name, owned by owner, as fields say otherwise. */
export const viewOf = (project: Project, name: string, owner: string, fields: Partial<View> = {}): View => ({
  id: `v-${project.identifier.toLowerCase()}-${name.toLowerCase().replace(/\s/g, "-")}`,
  workspace_id: project.workspace_id,
  project_id: project.id,
  name,
  description: "",
  logo_props: {},
  filters: {},
  display_filters,
  display_properties,
  owned_by_id: owner,
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
  ...fields,
});

/** A workspace's own view as nerve gives it, of no project (M7/P6b spec 2.6): as viewOf's, its id of the slug. */
export const workspaceViewOf = (workspace: Workspace, name: string, owner: string, fields: Partial<View> = {}): View =>
  viewOf(projectOf(workspace.slug.toUpperCase(), workspace.id), name, owner, { project_id: null, ...fields });

/** The requests of the views: a project's, where it lists them; a view's, where it is deleted. */
export const viewRequests = {
  list: (project: Project) => `/api/v0/projects/${project.id}/views`,
  at: (view: View) => `/api/v0/views/${view.id}`,
};

/**
 * The caller of a role in acme and in its projects web (as fields say otherwise) and ops, each nerve listed: his
 * permissions, the projects' store, and a views' store over a fake nerve of its own. Nerve answers the lists in fake
 * time: the test runs under vitest's fake timers.
 */
export async function viewsTab(caller: Caller, fields: Partial<Project> = {}) {
  const acme = workspaceOf("acme", { role: caller.workspaceRole });
  const role = { member_role: caller.projectRole, guest_view_all_features: caller.guestViewAll ?? false };
  const web = projectOf("WEB", acme.id, { ...role, ...fields });
  const ops = projectOf("OPS", acme.id, role);
  const { projects, permissions } = await permissionsOver([acme], [web, ops]);
  const nerve = new FakeNerve();
  const views = new ViewStore({ projectRoot: { project: projects } } as unknown as RootStore, nerve.client());
  return { acme, web, ops, projects, permissions, nerve, views };
}

/** The store reads the project's views, and nerve lists these. */
export const loadViews = (nerve: FakeNerve, store: ViewStore, project: Project, views: View[]) =>
  answered(
    nerve,
    () => store.fetchProjectViews(project.id),
    ["GET", viewRequests.list(project)],
    { data: views },
    "views"
  );
