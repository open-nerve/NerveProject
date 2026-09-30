import { createWorkspace, invite, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectWorkspaceDeleted } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W3, the workspace's settings (M3 design 2). The page version, with the
// session switch of 7.1, comes with the general page (P9); P4 adds the
// projects to the deletion's assertions.

test("W3 (API): the admin changes the workspace and deletes it with its members, invitations and settings at one moment; a member may do neither, and the slug never changes", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await invite(api, admin, slug, [{ email: emailFor(testInfo, "invitee"), role: 5 }]);
  const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body: { navigation_project_limit: 3 },
    headers: bearer(member),
  });
  expect(settings.response.status).toBe(200);

  const renamed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin" },
    headers: bearer(admin),
  });
  expect(renamed.response.status).toBe(200);
  expect(renamed.data).toMatchObject({
    slug,
    name: "Acme Corp",
    organization_size: "11-50",
    timezone: "Europe/Berlin",
    role: 20,
    total_members: 2,
  });
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS updated_by_the_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, adminEmail]
    )
  ).toEqual([{ name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin", updated_by_the_admin: true }]);

  // A member may not change it; a slug in the body is refused before anything is looked at.
  const byMember = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Mine" },
    headers: bearer(member),
  });
  expect(byMember.response.status).toBe(403);
  expect(byMember.error?.code).toBe("forbidden");
  const withSlug = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { slug: "other" } as never,
    headers: bearer(admin),
  });
  expect(withSlug.response.status).toBe(400);
  const memberDeletes = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(member),
  });
  expect(memberDeletes.response.status).toBe(403);
  expect(memberDeletes.error?.code).toBe("forbidden");

  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(admin) });
  expect(deleted.response.status).toBe(204);
  await expectWorkspaceDeleted(db, slug, adminEmail);
  const gone = await Promise.all(
    [admin, member].map((token) =>
      api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(token) })
    )
  );
  expect(gone.map((g) => [g.response.status, g.error?.code])).toEqual([
    [404, "workspace.not_found"],
    [404, "workspace.not_found"],
  ]);
});
