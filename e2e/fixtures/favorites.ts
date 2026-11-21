import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

type Favorite = components["schemas"]["Favorite"];
type FavoriteCreate = components["schemas"]["FavoriteCreate"];

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
