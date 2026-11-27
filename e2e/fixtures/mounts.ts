// What the pages ask nerve for as they load, and the workspace menu as it opens: S2's REQUESTS is built of these lists,
// S2 opens the menu after them, and a page's story that checks what it loads reads them here rather than writing its
// own.

/**
 * What a page asks nerve for as it loads, signed in, as "<method> <path>" with the query when there is one, {slug} for
 * the workspace's slug and {project} for the project's id: the app's start, on every page (M2 design 7.1, M3 design
 * 7.4); the workspace's, on every page of a workspace, whatever the role: its wrapper's, and the caller's unread
 * notifications its top navigation counts (M7 P4, useUnreadNotificationCount); the project's read, on every page of a
 * project; and the project's own resources, only once that read says the caller is a member of the project (M3 design
 * 7.1). Of M6's and M7's addresses only a project's cycles and its modules (PROJECT_CYCLES, M6 P5a; PROJECT_MODULES, M6
 * P6a), the workspace's unread notifications (WORKSPACE, M7 P4) and the pending work items of its intakes
 * (INTAKE_COUNTS, M7 P5a) are read as every page of a project loads; a project's views are read only on the pages that
 * use them (PROJECT_VIEWS, M7 P6a), the workspace's label catalog only on the pages that list work items across its
 * projects, each behind its gate (WORKSPACE_LABELS, M7 P6b), and none of the others (the intake's triage state,
 * favourites, recents; M3 design 3.1), nor one outside /api/v0 (M2 design 3.1). These lists are the record of what the
 * pages load: a phase that adds a fetch to a page adds it here (the P8b spec's appendix A.5 is a copy, as of P8b).
 */
export const APP = [
  "POST /api/v0/auth/refresh",
  "GET /api/v0/instance",
  "GET /api/v0/me",
  "GET /api/v0/me/profile",
  "GET /api/v0/workspaces",
];
export const WORKSPACE = [
  "GET /api/v0/me/workspaces/{slug}/preferences",
  "GET /api/v0/workspaces/{slug}/members",
  "GET /api/v0/workspaces/{slug}/projects?archived=false",
  "GET /api/v0/workspaces/{slug}/states",
  "GET /api/v0/workspaces/{slug}/notifications/unread-count",
];
export const PROJECT = ["GET /api/v0/projects/{project}"];
export const PROJECT_MEMBER = [
  "GET /api/v0/me/projects/{project}/preferences",
  "GET /api/v0/projects/{project}/labels",
  "GET /api/v0/projects/{project}/members",
  "GET /api/v0/projects/{project}/states",
];
/**
 * A project whose cycles are on: its members' pages read its cycles too, where the project's own resources are read
 * (M6 P5a spec 2.2, use-project-fetch.ts). A new project's cycles are off, so the lists above leave it out.
 */
export const PROJECT_CYCLES = ["GET /api/v0/projects/{project}/cycles"];
/**
 * A project whose modules are on: its members' pages read its modules too, where its cycles are read (M6 P6a spec 2.2,
 * use-project-fetch.ts). A new project's modules are off, as its cycles are.
 */
export const PROJECT_MODULES = ["GET /api/v0/projects/{project}/modules"];
/**
 * A workspace one of whose projects has its intake turned on, not archived, the caller a member of it, an admin, a
 * member or a guest: where the app's sidebar lists the projects (a workspace's pages but its settings and its
 * notifications), the pending work items of the caller's intakes are read too, once (M7 P5a spec 2.5, R7;
 * useIntakeCounts). A new project's intake is off, so the lists above leave it out.
 */
export const INTAKE_COUNTS = ["GET /api/v0/workspaces/{slug}/intake-counts"];
/**
 * A project whose views are on, to a caller the ways into them are offered to (an admin, a member or a guest): its
 * views list reads them as it loads, and a view's page and the command palette once their switcher or menu opens (M7
 * P6a spec 2.1, R20; the key ["PROJECT_VIEWS", project]). The project's shell does not read them, so the lists above
 * leave it out; a view's page reads its view instead ("GET /api/v0/views/{id}").
 */
export const PROJECT_VIEWS = ["GET /api/v0/projects/{project}/views"];
/**
 * The workspace's label catalog, the labels of its projects the caller is a member of, unarchived (M7 P6b spec 2.2,
 * R17; useWorkspaceLabelCatalog, the key ["WORKSPACE_LABELS", workspace]): the workspace's view page, the profile's
 * work items and the drafts page each read it once as they load, each behind its own gate (the drafts page and the
 * profile's work items for the workspace's admins and members only). The workspace's shell does not read it, so the
 * lists above leave it out.
 */
export const WORKSPACE_LABELS = ["GET /api/v0/workspaces/{slug}/labels"];
/**
 * A general settings page lists the time zones (M2 design 5.3): the project's, for its members; the workspace's, for
 * its admins and members (the workspace's settings show a guest no general page, M3 design 9.2).
 */
export const GENERAL = ["GET /api/v0/timezones"];
/** The workspace's projects page lists its archived projects too, its own fetch (useArchivedProjectsFetch), for every role. */
export const ARCHIVED = ["GET /api/v0/workspaces/{slug}/projects?archived=true"];
/** The members page lists the workspace's invitations for an admin alone, as nerve shows them to no one else. */
export const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];
/**
 * The workspace menu asks nerve for nothing as it opens, whatever the role (M3 closeout, W2): its entries are the
 * workspaces the app's start loaded (APP), and its leaving asks the dialog's question before it sends anything.
 */
export const WORKSPACE_MENU: string[] = [];
/** path with the values of names in place of their names: the address of a page of REQUESTS, or a request of a list. */
export function valued(path: string, names: Record<string, string>): string {
  return Object.entries(names).reduce((shown, [value, name]) => shown.replaceAll(name, value), path);
}
