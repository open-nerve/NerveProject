import { createHash } from "node:crypto";

import { createClient, type components } from "@nerve/api-client";
import { expect, type TestInfo } from "@playwright/test";

import { bearer } from "./auth";

/** The typed Nerve API client, generated from api/dist/openapi.yaml. */
export type Api = ReturnType<typeof createClient>;

export type Workspace = components["schemas"]["Workspace"];
export type WorkspaceCreate = components["schemas"]["WorkspaceCreate"];
export type WorkspacePreferences = components["schemas"]["WorkspacePreferences"];
export type WorkspacePreferencesUpdate = components["schemas"]["WorkspacePreferencesUpdate"];
export type WorkspaceInvitation = components["schemas"]["WorkspaceInvitation"];
export type WorkspaceMember = components["schemas"]["WorkspaceMember"];
export type InvitationCreate = components["schemas"]["InvitationCreate"];
export type Project = components["schemas"]["Project"];
export type ProjectCreate = components["schemas"]["ProjectCreate"];
export type ProjectUpdate = components["schemas"]["ProjectUpdate"];
export type ProjectMember = components["schemas"]["ProjectMember"];
export type ProjectMemberNew = components["schemas"]["ProjectMemberNew"];
export type ProjectPreferences = components["schemas"]["ProjectPreferences"];
export type ProjectPreferencesUpdate = components["schemas"]["ProjectPreferencesUpdate"];

/** Returns a client for the nerve at baseURL. */
export function createApi(baseURL: string): Api {
  return createClient({ baseUrl: baseURL });
}

/**
 * A slug of this run of this test, as emailFor gives an address: the tests
 * of a worker share its database, and --repeat-each runs a test again in
 * the same worker. It is label and 16 hexadecimal digits, well within the
 * 48 characters of a slug.
 */
export function slugFor(testInfo: TestInfo, label = "w"): string {
  const run = `${testInfo.testId}-${testInfo.repeatEachIndex}-${testInfo.retry}`;
  return `${label}-${createHash("sha256").update(run).digest("hex").slice(0, 16)}`;
}

/** Creates a workspace with the bearer token given, its caller the admin (M3 design 3.11), and returns it. */
export async function createWorkspace(api: Api, token: string, body: WorkspaceCreate): Promise<Workspace> {
  const { data, error, response } = await api.POST("/api/v0/workspaces", { body, headers: bearer(token) });
  expect(response.status, `create the workspace ${body.slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createWorkspace ${body.slug} answered 201 without the workspace`);
  }
  return data;
}

/** Invites the addresses of invitations to the workspace of slug with the bearer token given, an admin's, and returns the invitations in that order. */
export async function invite(
  api: Api,
  token: string,
  slug: string,
  invitations: InvitationCreate[]
): Promise<WorkspaceInvitation[]> {
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    body: { invitations },
    headers: bearer(token),
  });
  expect(response.status, `invite to ${slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`invite to ${slug} answered 201 without the invitations`);
  }
  return data.data;
}

/** Accepts invitation with the bearer token given, the invitee's, and returns the workspace as the new member reads it. */
export async function accept(api: Api, token: string, invitation: WorkspaceInvitation): Promise<Workspace> {
  const { data, error, response } = await api.POST("/api/v0/workspace-invitations/{invitation_id}/accept", {
    params: { path: { invitation_id: invitation.id } },
    body: { token: invitation.token },
    headers: bearer(token),
  });
  expect(response.status, `accept the invitation of ${invitation.email}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`accept answered 200 without the workspace`);
  }
  return data;
}

/**
 * Makes the account of member.email, whose bearer token is member.token, a member of the workspace of slug with
 * role: its admin, with adminToken, invites the address and the account accepts. The one way a workspace gets a
 * second member (M3 design 12 constraint 1). Returns the workspace as the new member reads it.
 */
export async function inviteAndAccept(
  api: Api,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  role: InvitationCreate["role"]
): Promise<Workspace> {
  const [invitation] = await invite(api, adminToken, slug, [{ email: member.email, role }]);
  if (!invitation) {
    throw new Error(`invite ${member.email} to ${slug} answered no invitation`);
  }
  return accept(api, member.token, invitation);
}

/** Lists the memberships of the workspace of slug, ended ones too, with the bearer token given, a member's. */
export async function listMembers(api: Api, token: string, slug: string): Promise<WorkspaceMember[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/members", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `list the members of ${slug}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`listMembers of ${slug} answered 200 without the members`);
  }
  return data.data;
}

/** The id of the membership of the account of memberId in the workspace of slug, as the caller of token lists it. */
export async function membershipOf(api: Api, token: string, slug: string, memberId: string): Promise<string> {
  const membership = (await listMembers(api, token, slug)).find((m) => m.member.id === memberId);
  if (!membership) {
    throw new Error(`no membership of ${memberId} in ${slug}`);
  }
  return membership.id;
}

/** Creates a project in the workspace of slug with the bearer token given, an admin's or a member's, and returns it. */
export async function createProject(api: Api, token: string, slug: string, body: ProjectCreate): Promise<Project> {
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the project ${body.identifier} in ${slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createProject ${body.identifier} answered 201 without the project`);
  }
  return data;
}

/**
 * Adds the accounts of members, each an active member of the project's workspace, to the project of projectId with
 * the bearer token given, an admin's of the project, and returns their memberships in that order. The one way a
 * project gets a member with a role of the admin's choice (M3 design 3.5).
 */
export async function addProjectMembers(
  api: Api,
  token: string,
  projectId: string,
  members: ProjectMemberNew[]
): Promise<ProjectMember[]> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: projectId } },
    body: { members },
    headers: bearer(token),
  });
  expect(response.status, `add members to ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`addProjectMembers to ${projectId} answered 201 without the members`);
  }
  return data.data;
}

/**
 * Runs make, which makes a story's projects, between two projects of another workspace of the caller of token, one
 * made before them and one after: a query that loses its project's id and reads the first row in id order, either
 * way, reads a project of another workspace, never one make made: First, or an earlier test's project in the worker's
 * database, ascending; Last descending, until the story makes another project. A project of another workspace is not
 * found. Returns what make returns.
 */
export async function amidAnotherWorkspace<T>(
  api: Api,
  token: string,
  testInfo: TestInfo,
  make: () => Promise<T>
): Promise<T> {
  const slug = slugFor(testInfo, "elsewhere");
  await createWorkspace(api, token, { name: "Elsewhere", slug });
  await createProject(api, token, slug, { name: "First", identifier: "FIRST" });
  const made = await make();
  await createProject(api, token, slug, { name: "Last", identifier: "LAST" });
  return made;
}
