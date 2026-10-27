import { Client } from "pg";

import {
  changeWorkspacePreferences,
  createLabel,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import { issueOwnedTable, projectTables, workspaceTables } from "../../fixtures/assert/catalog";
import { expectLabels, expectProjectCreated, expectProjectDeleted, type LabelRow } from "../../fixtures/assert/project";
import {
  expectInvitations,
  expectMembership,
  expectPreferences,
  expectWorkspaceDeleted,
} from "../../fixtures/assert/workspace";
import { accountId, bearer, emailFor, newAccount, register, registeredPAT } from "../../fixtures/auth";
import { furnishIssueRows } from "../../fixtures/furnish";
import { createDraft, draftRead, draftsListed } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// W3, the workspace's settings (M3 design 2): the API version. The page versions, with the session switch of 7.1,
// are in w3-workspace-settings-page.spec.ts.

test("W3 (API): the admin changes the workspace and deletes it with its members, invitations, settings and projects at one moment; a member may do neither, and the slug never changes", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registeredPAT(api, adminEmail)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await registeredPAT(api, memberEmail)).token;
  const declinerEmail = emailFor(testInfo, "decliner");
  const decliner = (await register(api, declinerEmail)).access_token;
  const inviteeEmail = emailFor(testInfo, "invitee");
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  // Two workspaces alike, each with the member, a pending and a declined invitation, the member's settings and a
  // project the admin created with the member its lead, which the member labelled; only Acme is deleted. Both exist
  // before either is furnished, and the member is Other's admin and Acme's member: each answer, role and row must be
  // the workspace's own, whichever row the database reads first.
  const [adminId, memberId] = await Promise.all([accountId(api, admin), accountId(api, member)]);
  const web = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  const otherWorkspace = await createWorkspace(api, admin, { name: "Other", slug: other });
  const acme = await createWorkspace(api, admin, { name: "Acme", slug });
  const tokens: string[] = [];
  const furnish = async ({ slug: target, id }: Workspace, memberRole: 15 | 20) => {
    expect(await inviteAndAccept(api, admin, target, { email: memberEmail, token: member }, memberRole)).toMatchObject({
      slug: target,
      role: memberRole,
      total_members: 2,
    });
    const created = await invite(api, admin, target, [
      { email: inviteeEmail, role: 5 },
      { email: declinerEmail, role: 15 },
    ]);
    tokens.push(...created.map((i) => i.token));
    const [, toDecline] = created;
    if (!toDecline) {
      throw new Error(`the invitations to ${target} were not created`);
    }
    const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
      params: { path: { invitation_id: toDecline.id } },
      body: { token: toDecline.token },
      headers: bearer(decliner),
    });
    expect(declined.response.status).toBe(204);
    await changeWorkspacePreferences(api, member, target, { navigation_project_limit: 3 });
    // The answer is the workspace's own new project as its creator sees it, first in both admins' sidebars.
    const project = await createProject(api, admin, target, {
      name: "Web",
      identifier: "web",
      project_lead_id: memberId,
    });
    expect(project).toMatchObject({
      workspace_id: id,
      identifier: "WEB",
      member_role: 20,
      sort_order: 65535,
      member_ids: [adminId, memberId],
    });
    // The member, Web's lead and so its admin, labels it: Bug, and UI under Bug.
    const bug = await createLabel(api, member, project.id, { name: "Bug", color: "#EF4444" });
    await createLabel(api, member, project.id, { name: "UI", parent_id: bug.id });
    return { project, bug };
  };
  const { project: otherWeb } = await furnish(otherWorkspace, 20);
  const { project: acmeWeb, bug: acmeBug } = await furnish(acme, 15);
  // Acme's Web has a work item too, the member's, assigned to him and labelled Bug, whose rows Acme's deletion must
  // write again as the admin's; Other's Web has none, as made (expectProjectCreated).
  await furnishIssueRows(api, member, slug, acmeWeb.id, memberId, acmeBug.id);
  const projectless = await createDraft(api, member, slug, { project_id: null, payload: { name: "Unassigned" } });
  /** Web's labels, as the member made them; deleted by the admin, as Acme's deletion writes them. */
  const webLabels = (deleted: boolean): LabelRow[] => [
    { name: "Bug", color: "#EF4444", parent: null, sort_order: 65535, deleted, by: deleted ? adminEmail : memberEmail },
    { name: "UI", color: "", parent: "Bug", sort_order: 75535, deleted, by: deleted ? adminEmail : memberEmail },
  ];
  await expectLabels(db, acmeWeb.id, webLabels(false));
  // The member reads Other as its admin, whatever becomes of Acme.
  const readOther = async () => {
    const read = await api.GET("/api/v0/workspaces/{slug}", {
      params: { path: { slug: other } },
      headers: bearer(member),
    });
    expect([read.response.status, read.data]).toEqual([
      200,
      expect.objectContaining({ name: "Other", slug: other, role: 20, total_members: 2 }),
    ]);
  };

  // A second admin changes it, so that the deletion must record its own author.
  const coAdminEmail = emailFor(testInfo, "co-admin");
  const coAdmin = (await registeredPAT(api, coAdminEmail)).token;
  await inviteAndAccept(api, admin, slug, { email: coAdminEmail, token: coAdmin }, 20);
  const renamed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin" },
    headers: bearer(coAdmin),
  });
  expect(renamed.response.status).toBe(200);
  expect(renamed.data).toMatchObject({
    slug,
    name: "Acme Corp",
    organization_size: "11-50",
    timezone: "Europe/Berlin",
    role: 20,
    total_members: 3,
  });
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS updated_by_the_co_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, coAdminEmail]
    )
  ).toEqual([
    { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin", updated_by_the_co_admin: true },
  ]);
  await readOther();

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

  // A project deleted before the workspace keeps its moment, and so do its rows: Old, with the member its lead, so
  // that each project table has a row of it.
  const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", project_lead_id: memberId });
  // The member moves Old in his sidebar, labels it and creates a work item in it, assigned to him and labelled Bug: his
  // display settings in it, its label and the work item's rows are his own writing, which Old's deletion must write
  // again as the admin's.
  const oldBug = await createLabel(api, member, old.id, { name: "Bug" });
  await furnishIssueRows(api, member, slug, old.id, memberId, oldBug.id);
  const moved = await api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: old.id } },
    body: { sort_order: 75535 },
    headers: bearer(member),
  });
  expect(moved.response.status).toBe(200);
  expect(
    await db.query(
      `SELECT s.sort_order, b.email AS by
         FROM project_user_properties s JOIN users u ON u.id = s.user_id JOIN users b ON b.id = s.updated_by_id
        WHERE s.project_id = $1 AND u.email = $2 AND s.deleted_at IS NULL`,
      [old.id, memberEmail]
    )
  ).toEqual([{ sort_order: 75535, by: memberEmail }]);
  await expectLabels(db, old.id, [
    { name: "Bug", color: "", parent: null, sort_order: 65535, deleted: false, by: memberEmail },
  ]);
  const oldBarrier = new Client({ connectionString: db.url, connectionTimeoutMillis: 10_000, query_timeout: 10_000 });
  await oldBarrier.connect();
  try {
    await oldBarrier.query("BEGIN");
    await oldBarrier.query("SET LOCAL lock_timeout='5s'");
    await oldBarrier.query(
      "SELECT id FROM draft_issues WHERE project_id=$1 AND deleted_at IS NULL ORDER BY id FOR UPDATE",
      [old.id]
    );
    const holder = (await oldBarrier.query<{ pid: number }>("SELECT pg_backend_pid() AS pid")).rows[0]?.pid;
    expect(holder).toEqual(expect.any(Number));
    const oldDeleted = await api.DELETE("/api/v0/projects/{project_id}", {
      params: { path: { project_id: old.id } },
      headers: bearer(admin),
    });
    expect(oldDeleted.response.status).toBe(204);
    await expect
      .poll(
        () =>
          db.query(
            `SELECT count(*)::int AS n FROM river_job WHERE kind='issue.delete_rows' AND args->>'scope'='project' AND args->>'id'=$1 AND state='running'`,
            [old.id]
          ),
        { timeout: 10_000 }
      )
      .toEqual([{ n: 1 }]);
    await expect
      .poll(
        () =>
          db.query(
            `SELECT count(*)::int AS n FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid)) AND wait_event_type='Lock'`,
            [holder]
          ),
        { timeout: 10_000 }
      )
      .toEqual([{ n: 1 }]);
    const oldDraftAudit = () =>
      db.query(
        `SELECT id, updated_at::text AS updated_at, deleted_at::text AS deleted_at, updated_by_id
         FROM draft_issues WHERE project_id = $1 ORDER BY id`,
        [old.id]
      );
    const oldDraftsBeforeWorkspaceDeletion = await db.query(
      `SELECT d.id, p.deleted_at::text AS updated_at, p.deleted_at::text AS deleted_at, p.updated_by_id
       FROM draft_issues d JOIN projects p ON p.id=d.project_id WHERE p.id=$1 ORDER BY d.id`,
      [old.id]
    );
    expect(oldDraftsBeforeWorkspaceDeletion).toEqual([
      {
        id: expect.any(String),
        updated_at: expect.any(String),
        deleted_at: expect.any(String),
        updated_by_id: adminId,
      },
    ]);

    const deleted = await api.DELETE("/api/v0/workspaces/{slug}", {
      params: { path: { slug } },
      headers: bearer(admin),
    });
    expect(deleted.response.status).toBe(204);
    const gone = await Promise.all(
      [admin, member].map((token) =>
        api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(token) })
      )
    );
    expect(gone.map((g) => [g.response.status, g.error?.code])).toEqual([
      [404, "workspace.not_found"],
      [404, "workspace.not_found"],
    ]);
    expect(
      await db.query("SELECT deleted_at IS NULL AS pending FROM draft_issues WHERE project_id=$1", [old.id])
    ).toEqual([{ pending: true }]);
    await oldBarrier.query("ROLLBACK");
    // The rows a story can delete before their workspace keep that moment: an invitation, when it is accepted (M3
    // design 3.8); Old and each row under it, with Old.
    await expectLabels(db, acmeWeb.id, webLabels(true));
    // Each of Web's labels carries Acme's deletion, its moment and its author: expectWorkspaceDeleted's counts cannot
    // tell a label stamped earlier, as Old's Bug, deleted before, gives the labels an earlier row.
    expect(
      await db.query(
        `SELECT l.name, l.deleted_at = w.deleted_at AS at_acmes_moment, l.updated_by_id = w.updated_by_id AS by_acmes_deleter
         FROM labels l JOIN workspaces w ON w.id = l.workspace_id WHERE l.project_id = $1 ORDER BY l.sort_order`,
        [acmeWeb.id]
      ),
      "when and by whom Web's labels were deleted"
    ).toEqual([
      { name: "Bug", at_acmes_moment: true, by_acmes_deleter: true },
      { name: "UI", at_acmes_moment: true, by_acmes_deleter: true },
    ]);
    // The invitations accepted before keep the moment they were answered; the declined one goes with the pending
    // one; the other workspace keeps everything.
    const invitations = (memberRole: number, withTheWorkspace: boolean) => [
      { email: memberEmail, role: memberRole, accepted: true, responded: true, deleted: true },
      { email: inviteeEmail, role: 5, accepted: false, responded: false, deleted: withTheWorkspace },
      { email: declinerEmail, role: 15, accepted: false, responded: true, deleted: withTheWorkspace },
    ];
    await expectInvitations(
      db,
      slug,
      adminEmail,
      [...invitations(15, true), { email: coAdminEmail, role: 20, accepted: true, responded: true, deleted: true }],
      tokens
    );
    await expectInvitations(db, other, adminEmail, invitations(20, false), tokens);
    await expectWorkspaceDeleted(db, slug, adminEmail, [
      "workspace_member_invites",
      "projects",
      ...(await projectTables(db)),
    ]);
    // Old and every row under it keep its deletion: its moment and its author, the admin.
    await expectProjectDeleted(db, old.id, adminEmail);
    await expect
      .poll(oldDraftAudit, { message: "the old project's drafts keep their exact earlier audit", timeout: 15_000 })
      .toEqual(oldDraftsBeforeWorkspaceDeletion);
    expect(
      await db.query(
        `SELECT d.project_id IS NULL AS projectless,
              d.deleted_at = w.deleted_at AND d.updated_at = w.deleted_at AS same_moment,
              d.updated_by_id = w.updated_by_id AND d.updated_by_id = $2::uuid AS same_actor
         FROM draft_issues d JOIN workspaces w ON w.id = d.workspace_id WHERE d.id = $1`,
        [projectless.id, adminId]
      )
    ).toEqual([{ projectless: true, same_moment: true, same_actor: true }]);
    await expectMembership(db, other, memberEmail, { role: 20, is_active: true });
    await expectProjectCreated(db, other, web, adminEmail, memberEmail, [
      { email: adminEmail, sort_order: 65535 },
      { email: memberEmail, sort_order: 65535 },
    ]);
    await expectPreferences(db, other, memberEmail, {
      navigation_control_preference: "ACCORDION",
      navigation_project_limit: 3,
    });
    await expectLabels(db, otherWeb.id, webLabels(false));
    await readOther();
  } finally {
    try {
      await oldBarrier.query("ROLLBACK");
    } finally {
      await oldBarrier.end();
    }
  }
});

// Catches a cascade that returns early for an empty project set and leaves nil-project drafts alive.
test("W3 (API): deleting a workspace with zero projects consumes its projectless draft with the workspace's exact audit and leaves no live row in the dynamic workspace directory", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  const workspace = await createWorkspace(api, admin.token, { name: "Empty", slug });
  const draft = await createDraft(api, admin.token, slug, { project_id: null, payload: { name: "No project yet" } });
  expect(await draftsListed(api, admin.token, slug)).toEqual({
    status: 200,
    page: { data: [draft], next_cursor: null, total: 1 },
  });
  expect(await db.query(`SELECT count(*)::int AS count FROM projects WHERE workspace_id = $1`, [workspace.id])).toEqual(
    [{ count: 0 }]
  );
  const tables = await workspaceTables(db);
  expect(tables).toContain("draft_issues");
  const before = new Map(
    await Promise.all(
      tables.map(async (table): Promise<[string, number]> => {
        const [row] = await db.query<{ count: number }>(
          `SELECT count(*)::int AS count FROM ${table} WHERE workspace_id = $1`,
          [workspace.id]
        );
        if (!row) throw new Error(`missing count for ${table}`);
        return [table, row.count];
      })
    )
  );
  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(admin.token),
  });
  expect(deleted.response.status).toBe(204);
  expect(await draftRead(api, admin.token, draft.id)).toEqual({ status: 404, code: "issue.draft_not_found" });
  const read = (table: string) =>
    db.query(
      `SELECT count(*)::int AS count,
       count(*) FILTER (WHERE r.deleted_at IS NULL)::int AS live,
       count(*) FILTER (WHERE r.deleted_at=w.deleted_at AND r.updated_at=w.deleted_at
                         AND r.updated_by_id=w.updated_by_id AND r.updated_by_id=$2::uuid)::int AS stamped
     FROM ${table} r JOIN workspaces w ON w.id=r.workspace_id WHERE w.id=$1`,
      [workspace.id, admin.id]
    );
  await Promise.all(
    tables
      .filter((name) => !issueOwnedTable(name))
      .map(async (table) => {
        expect(await read(table), `synchronous rows of ${table}`).toEqual([
          { count: before.get(table), live: 0, stamped: before.get(table) },
        ]);
      })
  );
  await Promise.all(
    tables.filter(issueOwnedTable).map(async (table) => {
      await expect
        .poll(() => read(table), { message: `issue rows of ${table}`, timeout: 15_000 })
        .toEqual([{ count: before.get(table), live: 0, stamped: before.get(table) }]);
    })
  );
});
