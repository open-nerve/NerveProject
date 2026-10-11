// What the pages ask nerve for as they load: S2's REQUESTS is built of these lists, and a page's story that checks what
// it loads reads them here rather than writing its own.

/**
 * What a page asks nerve for as it loads, signed in, as "<method> <path>" with the query when there is one, {slug} for
 * the workspace's slug and {project} for the project's id: the app's start, on every page (M2 design 7.1, M3 design
 * 7.4); the workspace wrapper's, on every page of a workspace, whatever the role; the project's read, on every page of
 * a project; and the project's own resources, only once that read says the caller is a member of the project (M3
 * design 7.1). No list has an address of M6's or M7's (cycles, modules, views, the intake's triage state, favourites,
 * the unread notifications, recents; M3 design 3.1), nor one outside /api/v0 (M2 design 3.1). These lists are the
 * record of what the pages load: a phase that adds a fetch to a page adds it here (the P8b spec's appendix A.5 is a
 * copy, as of P8b).
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
];
export const PROJECT = ["GET /api/v0/projects/{project}"];
export const PROJECT_MEMBER = [
  "GET /api/v0/me/projects/{project}/preferences",
  "GET /api/v0/projects/{project}/labels",
  "GET /api/v0/projects/{project}/members",
  "GET /api/v0/projects/{project}/states",
];
/**
 * A general settings page lists the time zones (M2 design 5.3): the project's, for its members; the workspace's, for
 * its admins and members (the workspace's settings show a guest no general page, M3 design 9.2).
 */
export const GENERAL = ["GET /api/v0/timezones"];
/** The workspace's projects page lists its archived projects too, its own fetch (useArchivedProjectsFetch), for every role. */
export const ARCHIVED = ["GET /api/v0/workspaces/{slug}/projects?archived=true"];
/** The members page lists the workspace's invitations for an admin alone, as nerve shows them to no one else. */
export const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];

/** path with the values of names in place of their names: the address of a page of REQUESTS, or a request of a list. */
export function valued(path: string, names: Record<string, string>): string {
  return Object.entries(names).reduce((shown, [value, name]) => shown.replaceAll(name, value), path);
}
