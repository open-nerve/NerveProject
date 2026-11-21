import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";
import type { Database } from "./db";

export type Favorite = components["schemas"]["Favorite"];
export type FavoriteCreate = components["schemas"]["FavoriteCreate"];
type FavoriteUpdate = components["schemas"]["FavoriteUpdate"];

/**
 * Creates a favorite of the caller of token in the workspace of slug through the navigation API, of a target or a
 * folder, which must be new (201): an admin or a member of the workspace, who sees the target, and has no undeleted
 * favorite of it yet (M7 P3c spec 2.4, 2.5).
 */
export async function createFavorite(api: Api, token: string, slug: string, body: FavoriteCreate): Promise<Favorite> {
  const what = `the ${body.entity_type} ${body.entity_id ?? body.name}`;
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/favorites", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `favor ${what} in ${slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createFavorite of ${what} answered 201 without the favorite`);
  }
  return data;
}

/**
 * Makes the caller of token a folder of name in the workspace of slug and favors the view of viewId in it, through the
 * navigation API: of a workspace's view, two favorites of no project, the second in the first (M7 P3c spec 2.3, 2.5).
 */
export async function favorViewInFolder(
  api: Api,
  token: string,
  slug: string,
  name: string,
  viewId: string
): Promise<void> {
  const folder = await createFavorite(api, token, slug, { entity_type: "folder", name });
  const held = await createFavorite(api, token, slug, { entity_type: "view", entity_id: viewId, parent_id: folder.id });
  expect(held.parent_id, `the view ${viewId} favored in the folder ${name} of ${slug}`).toBe(folder.id);
}

/**
 * The calls on favorites whose answer a story compares, each by the caller of token: its status and the favorite or
 * the favorites it answers, or the problem's code and fields (M7 P3c spec 2.4).
 */
export function favoriteCalls(api: Api) {
  return {
    list: async (token: string, slug: string) => {
      const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/favorites", {
        params: { path: { slug } },
        headers: bearer(token),
      });
      return data ? { status: response.status, favorites: data.data } : answer(response, error);
    },
    create: async (token: string, slug: string, body: FavoriteCreate) => {
      const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/favorites", {
        params: { path: { slug } },
        body,
        headers: bearer(token),
      });
      return data ? { status: response.status, favorite: data } : answer(response, error);
    },
    update: async (token: string, id: string, body: FavoriteUpdate) => {
      const { data, error, response } = await api.PATCH("/api/v0/favorites/{favorite_id}", {
        params: { path: { favorite_id: id } },
        body,
        headers: bearer(token),
      });
      return data ? { status: response.status, favorite: data } : answer(response, error);
    },
    remove: async (token: string, id: string) => {
      const { error, response } = await api.DELETE("/api/v0/favorites/{favorite_id}", {
        params: { path: { favorite_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** The favorite a call of favoriteCalls answered with status, which it must have. */
export function favoriteOf(answered: { status: number; favorite?: Favorite }, status: number): Favorite {
  expect(answered.status, JSON.stringify(answered)).toBe(status);
  if (!answered.favorite) {
    throw new Error(`answered ${status} without the favorite`);
  }
  return answered.favorite;
}

/** A favorite as stored (favoriteRows). */
interface FavoriteRow {
  id: string;
  type: string;
  identifier: string | null;
  project: string | null;
  parent: string | null;
  name: string | null;
  sequence: number;
  deleted: string | null;
  writer: string;
  stamped: boolean;
}

/**
 * The favorites of the account of userId in the workspace of workspaceId as stored (user_favorites; M7 P3c spec 2.3),
 * deleted ones too, the first made first: each one's id; its kind, its target's id and its project, a folder's none
 * and a workspace's view's no project; its folder; a folder's own name, null for the others; its sequence; the moment
 * it was deleted, null while it is not; the account that wrote it last; and whether its moments are a favorite's, made
 * by its owner, written last no earlier than it was made and, deleted, deleted at that last write. Read-only.
 */
export async function favoriteRows(db: Database, userId: string, workspaceId: string): Promise<FavoriteRow[]> {
  return db.query<FavoriteRow>(
    `SELECT id::text AS id, entity_type AS type, entity_identifier::text AS identifier, project_id::text AS project,
            parent_id::text AS parent, name, sequence, deleted_at::text AS deleted, updated_by_id::text AS writer,
            created_by_id = user_id AND updated_at >= created_at
              AND (deleted_at IS NULL OR deleted_at = updated_at) AS stamped
       FROM user_favorites WHERE user_id = $1 AND workspace_id = $2 ORDER BY created_at, id`,
    [userId, workspaceId]
  );
}

/**
 * The row favoriteRows reads of favorite as it was answered: undeleted, written last by the account of writerId, its
 * owner, or what took it away when deleted is the moment.
 */
export function storedAs(favorite: Favorite, writerId: string, deleted: string | null = null): FavoriteRow {
  return {
    id: favorite.id,
    type: favorite.entity_type,
    identifier: favorite.entity_id,
    project: favorite.project_id,
    parent: favorite.parent_id,
    name: favorite.entity_type === "folder" ? favorite.name : null,
    sequence: favorite.sequence,
    deleted,
    writer: writerId,
    stamped: true,
  };
}

/**
 * The moment the target of id, a row of table, was deleted or archived (column), null while it is not, and the account
 * that wrote it last: what the cascade of its favorites takes from it (M7 P3c spec 2.6, R3). Read-only.
 */
export async function targetStamp(
  db: Database,
  table: "projects" | "cycles" | "modules" | "issue_views",
  column: "archived_at" | "deleted_at",
  id: string
): Promise<{ at: string | null; by: string }> {
  const [row] = await db.query<{ at: string | null; by: string }>(
    `SELECT ${column}::text AS at, updated_by_id::text AS by FROM ${table} WHERE id = $1`,
    [id]
  );
  if (!row) {
    throw new Error(`no ${table} row ${id}`);
  }
  return row;
}
