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
export type InvitationCreate = components["schemas"]["InvitationCreate"];

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
 * Makes the account of email, whose bearer token is memberToken, a member of the workspace of slug with role:
 * its admin, with adminToken, invites the address and the account accepts. The one way a workspace gets a
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
