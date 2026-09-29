import { createApi, createWorkspace, slugFor, type Api } from "../../fixtures/api";
import { countWorkspaces, expectNoWorkspaceAdded, expectWorkspaceCreated } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W1, create a workspace (M3 design 2, 3.10, 3.11). The page version comes
// with the onboarding and /create-workspace pages.

/** The answer of GET /api/v0/workspace-slugs/{slug}. */
async function availability(api: Api, token: string, slug: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/workspace-slugs/{slug}", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `check ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

test("W1 (API): creating a workspace makes the caller its admin and only member; a taken or reserved slug, or creation switched off, adds nothing", async ({
  api,
  db,
  nerveWith,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const slug = slugFor(testInfo);
  expect(await availability(api, pat, slug)).toEqual({ available: true });

  const body = { name: "Acme", slug, organization_size: "2-10", timezone: "Asia/Shanghai" } as const;
  const created = await createWorkspace(api, pat, body);

  expect(created).toMatchObject({ ...body, logo_url: null, role: 20, total_members: 1 });
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
  expect(await availability(api, pat, slug)).toEqual({ available: false, reason: "taken" });
  expect(await availability(api, pat, "settings")).toEqual({ available: false, reason: "reserved" });

  const before = await countWorkspaces(db);
  const taken = await api.POST("/api/v0/workspaces", { body: { name: "Acme again", slug }, headers: bearer(pat) });
  expect(taken.response.status).toBe(409);
  expect(taken.error?.code).toBe("workspace.slug_taken");
  const reserved = await api.POST("/api/v0/workspaces", {
    body: { name: "Settings", slug: "settings" },
    headers: bearer(pat),
  });
  expect(reserved.response.status).toBe(422);
  expect(reserved.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "slug", code: "not_allowed" },
  ]);
  // A nerve with creation switched off, on the same database: the personal
  // access token works there too.
  const closed = createApi((await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" })).baseURL);
  const refused = await closed.POST("/api/v0/workspaces", {
    body: { name: "Beta", slug: slugFor(testInfo, "beta") },
    headers: bearer(pat),
  });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("workspace.creation_disabled");
  await expectNoWorkspaceAdded(db, before);
  // No refusal changed the workspace created first.
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
});
