import { slugFor } from "../../fixtures/api";
import { countWorkspaces, expectNoWorkspaceAdded, expectWorkspaceCreated } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";
import { nerveWorkspaces, nerveWorkspacesFails } from "../../fixtures/workspaces";

// W10, the administrator creates a workspace (M3 design 2, 3.11): how
// workspaces come to be while creation is switched off. There is no page
// version.

test("W10: nerve workspaces create makes the account of the address the workspace's admin, creation switched off or not; a taken slug, an unknown or a deactivated account adds nothing", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const deactivated = emailFor(testInfo, "deactivated");
  await register(api, deactivated);
  await nerveUsers(db, ["deactivate", "--email", deactivated]);
  const slug = slugFor(testInfo);

  const line = await nerveWorkspaces(
    db,
    ["create", "--slug", slug, "--name", "Acme", "--admin-email", email.toUpperCase()],
    { NERVE_WORKSPACE__CREATION_ENABLED: "false" }
  );

  expect(line).toBe(`created workspace ${slug} with admin ${email}\n`);
  const id = await expectWorkspaceCreated(db, email, { name: "Acme", slug, organization_size: null, timezone: "UTC" });
  // The account sees it through the API, as its admin and only member.
  const { data, response } = await api.GET("/api/v0/workspaces", { headers: bearer(pat) });
  expect(response.status).toBe(200);
  expect(data?.data).toMatchObject([{ id, slug, name: "Acme", role: 20, total_members: 1 }]);

  const before = await countWorkspaces(db);
  await nerveWorkspacesFails(
    db,
    ["create", "--slug", slug, "--name", "Acme again", "--admin-email", email],
    "A workspace with this slug exists."
  );
  await nerveWorkspacesFails(
    db,
    ["create", "--slug", slugFor(testInfo, "beta"), "--name", "Beta", "--admin-email", emailFor(testInfo, "nobody")],
    "No account has this e-mail address."
  );
  await nerveWorkspacesFails(
    db,
    ["create", "--slug", slugFor(testInfo, "beta"), "--name", "Beta", "--admin-email", deactivated],
    "The account is deactivated."
  );
  await expectNoWorkspaceAdded(db, before);
});
