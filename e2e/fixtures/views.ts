import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

type View = components["schemas"]["View"];
type ViewCreate = components["schemas"]["ViewCreate"];

/**
 * Creates a view in the project of projectId through the navigation API, as the caller of token, an admin or a member
 * of it, once its views are turned on (M7 P3b spec 2.4).
 */
export async function createProjectView(api: Api, token: string, projectId: string, body: ViewCreate): Promise<View> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/views", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the view ${body.name} in ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createProjectView ${body.name} answered 201 without the view`);
  }
  return data;
}

/**
 * Creates a view of the workspace of slug's own, of no project, through the navigation API, as the caller of token, an
 * admin or a member of it (M7 P3b spec 2.4).
 */
export async function createWorkspaceView(api: Api, token: string, slug: string, body: ViewCreate): Promise<View> {
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/views", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the view ${body.name} of ${slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createWorkspaceView ${body.name} answered 201 without the view`);
  }
  return data;
}
