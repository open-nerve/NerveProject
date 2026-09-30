import {
  createWorkspace,
  slugFor,
  type Api,
  type WorkspacePreferences,
  type WorkspacePreferencesUpdate,
} from "../../fixtures/api";
import { expectPreferences } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W8, the project navigation's settings (M3 design 2, 3.18). The page
// version comes with the sidebar's "project navigation" dialog (P9).

const defaults: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 };

/** The answer of GET /api/v0/me/workspaces/{slug}/preferences, which must be 200. */
async function read(api: Api, token: string, slug: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `read the settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** The answer of PATCH /api/v0/me/workspaces/{slug}/preferences, which must be 200. */
async function change(api: Api, token: string, slug: string, body: WorkspacePreferencesUpdate): Promise<unknown> {
  const { data, error, response } = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `change the settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

test("W8 (API): the settings are the defaults and nothing is stored until the first change, which stores one row; later changes change it; each workspace has its own", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, pat, { name: "Acme", slug });
  await createWorkspace(api, pat, { name: "Other", slug: other });

  expect(await read(api, pat, slug)).toEqual(defaults);
  await expectPreferences(db, slug, email, null);

  const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };
  expect(await change(api, pat, slug, tabbed)).toEqual(tabbed);
  const row = await expectPreferences(db, slug, email, tabbed);
  // After a refresh the page reads them again.
  expect(await read(api, pat, slug)).toEqual(tabbed);

  // Showing every project: one field changes, the other stays, on the same row.
  expect(await change(api, pat, slug, { navigation_project_limit: 0 })).toEqual({
    ...tabbed,
    navigation_project_limit: 0,
  });
  expect(await expectPreferences(db, slug, email, { ...tabbed, navigation_project_limit: 0 })).toBe(row);

  // The other workspace keeps the defaults, and nothing is stored for it.
  expect(await read(api, pat, other)).toEqual(defaults);
  await expectPreferences(db, other, email, null);

  // Another account, not a member, reads nothing there.
  const stranger = (await register(api, emailFor(testInfo, "stranger"))).access_token;
  const refused = await api.GET("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    headers: bearer(stranger),
  });
  expect(refused.response.status).toBe(404);
  expect(refused.error?.code).toBe("workspace.not_found");
});
