import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";
import type { Database } from "./db";
import type { IssueQuery } from "./issues";

type View = components["schemas"]["View"];
type ViewCreate = components["schemas"]["ViewCreate"];
type ViewUpdate = components["schemas"]["ViewUpdate"];

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

/**
 * What a new view has of what its creation leaves out (ViewCreate; M7 P3b spec 2.4): no description, icon or filter;
 * the list's display defaults, the list layout without grouping, the newest first, sub-issues and empty groups shown,
 * the calendar by month without weekends; and every property shown.
 */
export const viewDefaults = {
  description: "",
  logo_props: {},
  filters: {},
  display_filters: {
    layout: "list",
    group_by: null,
    sub_group_by: null,
    order_by: "-created_at",
    sub_issue: true,
    show_empty_groups: true,
    calendar: { layout: "month", show_weekends: false },
  },
  display_properties: {
    assignee: true,
    start_date: true,
    due_date: true,
    labels: true,
    key: true,
    priority: true,
    state: true,
    sub_issue_count: true,
    link: true,
    attachment_count: true,
    created_on: true,
    updated_on: true,
    cycle: true,
    modules: true,
  },
} satisfies Pick<View, "description" | "logo_props" | "filters" | "display_filters" | "display_properties">;

/**
 * The calls on views whose answer a story compares, each by the caller of token: its status and the view or the views
 * it answers, or the problem's code and fields (M7 P3b spec 2.4).
 */
export function viewCalls(api: Api) {
  return {
    listInProject: async (token: string, projectId: string) => {
      const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/views", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      });
      return data ? { status: response.status, views: data.data } : answer(response, error);
    },
    listInWorkspace: async (token: string, slug: string) => {
      const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/views", {
        params: { path: { slug } },
        headers: bearer(token),
      });
      return data ? { status: response.status, views: data.data } : answer(response, error);
    },
    createInProject: async (token: string, projectId: string, body: ViewCreate) => {
      const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/views", {
        params: { path: { project_id: projectId } },
        body,
        headers: bearer(token),
      });
      return data ? { status: response.status, view: data } : answer(response, error);
    },
    createInWorkspace: async (token: string, slug: string, body: ViewCreate) => {
      const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/views", {
        params: { path: { slug } },
        body,
        headers: bearer(token),
      });
      return data ? { status: response.status, view: data } : answer(response, error);
    },
    read: async (token: string, id: string) => {
      const { data, error, response } = await api.GET("/api/v0/views/{view_id}", {
        params: { path: { view_id: id } },
        headers: bearer(token),
      });
      return data ? { status: response.status, view: data } : answer(response, error);
    },
    update: async (token: string, id: string, body: ViewUpdate) => {
      const { data, error, response } = await api.PATCH("/api/v0/views/{view_id}", {
        params: { path: { view_id: id } },
        body,
        headers: bearer(token),
      });
      return data ? { status: response.status, view: data } : answer(response, error);
    },
    remove: async (token: string, id: string) => {
      const { error, response } = await api.DELETE("/api/v0/views/{view_id}", {
        params: { path: { view_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** The view a call of viewCalls answered with status, which it must have. */
export function viewOf(answered: { status: number; view?: View }, status: number): View {
  expect(answered.status, JSON.stringify(answered)).toBe(status);
  if (!answered.view) {
    throw new Error(`answered ${status} without the view`);
  }
  return answered.view;
}

/**
 * The row of the view of id as stored (issue_views; M7 P3b spec 2.3): its project, null for a workspace's view; its
 * owner, the account that made it and the one that wrote it last, by id; whether it is deleted; and whether its moments
 * are a view's, written last no earlier than it was made and, deleted, deleted at that last write.
 */
export async function viewRow(db: Database, id: string) {
  const [row] = await db.query<{
    project_id: string | null;
    owner: string;
    made_by: string;
    written_by: string;
    deleted: boolean;
    stamped: boolean;
  }>(
    `SELECT project_id::text AS project_id, owned_by_id::text AS owner, created_by_id::text AS made_by,
            updated_by_id::text AS written_by, deleted_at IS NOT NULL AS deleted,
            updated_at >= created_at AND (deleted_at IS NULL OR deleted_at = updated_at) AS stamped
       FROM issue_views WHERE id = $1`,
    [id]
  );
  if (!row) {
    throw new Error(`no view ${id}`);
  }
  return row;
}

/**
 * The query a list of work items takes for the words of view, which groups nothing (H13; M7 P3b spec 2.4): each filter
 * key's alternatives, its order and whether it shows sub-issues. Its layout, empty groups, calendar and properties only
 * shape how a page shows the list.
 */
export function savedQuery(view: View) {
  const { group_by, sub_group_by, order_by, sub_issue } = view.display_filters;
  if (group_by !== null || sub_group_by !== null) {
    throw new Error(`the view ${view.name} groups its work items`);
  }
  return { ...view.filters, order_by: order_by as IssueQuery["order_by"], sub_issue: String(sub_issue) };
}
