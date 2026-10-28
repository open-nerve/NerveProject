import type { TestInfo } from "@playwright/test";
import { escapeIdentifier } from "pg";

import { createLabel, createProject, createWorkspace, slugFor, type Api } from "../../fixtures/api";
import { accountId, bearer, createPAT, emailFor, login, newAccount } from "../../fixtures/auth";
import { keysBelow, projectTables, workspaceTables } from "../../fixtures/assert/catalog";
import { expectIssueDeleted } from "../../fixtures/assert/issue";
import { expectProjectDeleted } from "../../fixtures/assert/project";
import { expectWorkspaceDeleted } from "../../fixtures/assert/workspace";
import { archiveWatermark, completedPeriodicJobs } from "../../fixtures/auto-archive";
import { createCommentReaction, createIssueComment, createIssueReaction } from "../../fixtures/comments";
import type { Database } from "../../fixtures/db";
import { furnishIssueRows, furnishWorkspace } from "../../fixtures/furnish";
import { createIssue, createIssueRelations, issueDeleted, issueRead } from "../../fixtures/issues";
import { completedLinkCrawl, createIssueLink } from "../../fixtures/links";
import { clockTest as test, expect } from "../../fixtures/test";

type Account = Awaited<ReturnType<typeof newAccount>>;
type Row = { id: string; deleted_at: string | null; [column: string]: unknown };
type Snapshot = { table: string; rows: Row[] };

/** JSON keeps every column, including PostgreSQL's full timestamp precision. */
async function rowsById(db: Database, table: string, ids: string[]): Promise<Row[]> {
  return (
    await db.query<{ row: Row }>(
      `SELECT to_jsonb(t) AS row FROM ${escapeIdentifier(table)} t WHERE id=ANY($1::uuid[]) ORDER BY id`,
      [ids]
    )
  ).map(({ row }) => row);
}

async function snapshot(db: Database, ids: Map<string, Set<string>>): Promise<Snapshot[]> {
  return Promise.all(
    [...ids].map(async ([table, values]) => {
      expect(values.size, `${table} has actual furnished IDs`).toBeGreaterThan(0);
      const rows = await rowsById(db, table, [...values]);
      expect(rows.length, `${table} IDs still exist before purge`).toBe(values.size);
      return { table, rows };
    })
  );
}

/** Collect before parents disappear: a later JOIN to a missing parent proves nothing about its children. */
async function issueIDs(db: Database, root: string) {
  const keys = await keysBelow(db, "issues");
  const ids = new Map<string, Set<string>>([["issues", new Set([root])]]);
  const populated = new Set<string>();
  let grew = true;
  while (grew) {
    grew = false;
    for (const key of keys) {
      // Each edge consumes IDs discovered by earlier edges; repeat until the FK closure stops growing.
      // eslint-disable-next-line no-await-in-loop
      const rows = await db.query<{ id: string }>(
        `SELECT id FROM ${escapeIdentifier(key.child)} WHERE ${escapeIdentifier(key.column)}=ANY($1::uuid[])`,
        [[...(ids.get(key.parent) ?? [])]]
      );
      if (rows.length) populated.add(`${key.child}.${key.column}`);
      const children = ids.get(key.child) ?? new Set<string>();
      for (const { id } of rows) {
        if (!children.has(id)) grew = true;
        children.add(id);
      }
      ids.set(key.child, children);
    }
  }
  expect(
    [...populated].toSorted(),
    "every real FK edge has rows, including both relation ends and comment reactions"
  ).toEqual(keys.map((key) => `${key.child}.${key.column}`).toSorted());
  expect(ids.get("issues")?.size, "root, child and grandchild").toBe(3);
  return ids;
}

async function parentIDs(db: Database, table: "projects" | "workspaces", id: string) {
  const tables = table === "projects" ? await projectTables(db) : await workspaceTables(db);
  const column = table === "projects" ? "project_id" : "workspace_id";
  const ids = new Map<string, Set<string>>([[table, new Set([id])]]);
  await Promise.all(
    tables.map(async (child) => {
      const rows = await db.query<{ id: string }>(`SELECT id FROM ${escapeIdentifier(child)} WHERE ${column}=$1`, [id]);
      expect(rows.length, `${table}/${id}: ${child} is furnished through API`).toBeGreaterThan(0);
      ids.set(child, new Set(rows.map((row) => row.id)));
    })
  );
  expect(tables).toContain("draft_issues");
  expect(tables).toContain("comment_reactions");
  return ids;
}

/** Furnish leaves link crawling asynchronous; stabilize it before any full-row comparison. */
async function finishLinks(db: Database, ids: Map<string, Set<string>>) {
  await Promise.all(
    [...(ids.get("issue_links") ?? [])].map(async (id) => {
      const [link] = await db.query<{ url: string }>("SELECT url FROM issue_links WHERE id=$1", [id]);
      if (!link) throw new Error(`missing furnished link ${id}`);
      await completedLinkCrawl(db, id, link.url);
    })
  );
}

async function seed(api: Api, db: Database, info: TestInfo, admin: Account, member: Account, age: string) {
  const token = admin.token;
  const keeper = await createWorkspace(api, token, { name: `${age} keeper`, slug: slugFor(info, `${age}-keep`) });
  const project = await createProject(api, token, keeper.slug, { name: "Independent issue", identifier: "ISS" });
  const usedLabel = await createLabel(api, token, project.id, { name: "Attached" });
  const label = await createLabel(api, token, project.id, { name: "Independent label" });
  const pat = await createPAT(api, token, { label: `${age} independent PAT` });
  const description = `<mention-component entity_name="user_mention" entity_identifier="${admin.id}"></mention-component>`;
  const root = await createIssue(api, token, project.id, {
    name: `${age} root`,
    description_html: description,
    assignee_ids: [admin.id],
    label_ids: [usedLabel.id],
  });
  const child = await createIssue(api, token, project.id, { name: "Child", parent_id: root.id });
  await createIssue(api, token, project.id, { name: "Grandchild", parent_id: child.id });
  const comment = await createIssueComment(api, token, root.id, "<p>Physical second-level reaction</p>");
  await createCommentReaction(api, token, comment.id);
  await createIssueReaction(api, token, root.id);
  const link = await createIssueLink(api, token, root.id, { url: `http://169.254.169.254/furnish/${root.id}` });
  await completedLinkCrawl(db, link.id, link.url);
  async function relate(relation_type: "relates_to" | "blocking") {
    const peer = await createIssue(api, token, project.id, { name: relation_type });
    await createIssueRelations(api, token, root.id, { relation_type, issue_ids: [peer.id] });
  }
  await relate("relates_to");
  await relate("blocking");
  const doomed = await createProject(api, token, keeper.slug, { name: "Project target", identifier: "DEL" });
  const projectLabel = await createLabel(api, token, doomed.id, { name: "Bug" });
  await furnishIssueRows(api, token, keeper.slug, doomed.id, admin.id, projectLabel.id);
  const workspace = await createWorkspace(api, token, { name: "Workspace target", slug: slugFor(info, `${age}-del`) });
  await furnishWorkspace(api, token, workspace.slug, member, emailFor(info, `${age}-pending`));
  // A neighboring project remains live even when already 61 days old.
  const neighbor = await createProject(api, token, keeper.slug, { name: "Neighbor", identifier: "LIVE" });
  const neighborLabel = await createLabel(api, token, neighbor.id, { name: "Keep" });
  await furnishIssueRows(api, token, keeper.slug, neighbor.id, admin.id, neighborLabel.id);
  const scopes = [
    await issueIDs(db, root.id),
    await parentIDs(db, "projects", doomed.id),
    await parentIDs(db, "workspaces", workspace.id),
    new Map([
      ["labels", new Set([label.id])],
      ["api_tokens", new Set([pat.id])],
    ]),
  ];
  const neighborIDs = await parentIDs(db, "projects", neighbor.id);
  await Promise.all([...scopes, neighborIDs].map((ids) => finishLinks(db, ids)));
  // These two parents are independent of the deleted project/workspace targets.
  const parents = new Map([
    ["workspaces", new Set([keeper.id])],
    ["projects", new Set([project.id])],
  ]);
  return { root, doomed, workspace, scopes, parents, neighborIDs, label, pat };
}

type Seed = Awaited<ReturnType<typeof seed>>;

async function remove(api: Api, db: Database, admin: Account, data: Seed) {
  const headers = bearer(admin.token);
  expect(await issueDeleted(api, admin.token, data.root.id)).toEqual({ status: 204 });
  await expectIssueDeleted(db, data.root.id, admin.email);
  expect(
    (
      await api.DELETE("/api/v0/labels/{label_id}", {
        params: { path: { label_id: data.label.id } },
        headers,
      })
    ).response.status
  ).toBe(204);
  expect(
    (
      await api.DELETE("/api/v0/api-tokens/{token_id}", {
        params: { path: { token_id: data.pat.id } },
        headers,
      })
    ).response.status
  ).toBe(204);
  expect(
    (
      await api.DELETE("/api/v0/projects/{project_id}", {
        params: { path: { project_id: data.doomed.id } },
        headers,
      })
    ).response.status
  ).toBe(204);
  await expectProjectDeleted(db, data.doomed.id, admin.email);
  expect(
    (
      await api.DELETE("/api/v0/workspaces/{slug}", {
        params: { path: { slug: data.workspace.slug } },
        headers,
      })
    ).response.status
  ).toBe(204);
  await expectWorkspaceDeleted(db, data.workspace.slug, admin.email, ["workspace_member_invites"]);
  const projectIds = [...data.scopes[2]!.get("projects")!, data.doomed.id];
  await expect
    .poll(
      () =>
        db.query(
          `SELECT args->>'scope' AS scope, args->>'id' AS id, state::text FROM river_job
     WHERE kind='issue.delete_rows' AND ((args->>'scope'='project' AND args->>'id'=ANY($1::text[]))
       OR (args->>'scope'='workspace_drafts' AND args->>'id'=$2)) ORDER BY scope, id`,
          [projectIds, data.workspace.id]
        ),
      { timeout: 15_000 }
    )
    .toEqual([
      ...projectIds.toSorted().map((id) => ({ scope: "project", id, state: "completed" })),
      { scope: "workspace_drafts", id: data.workspace.id, state: "completed" },
    ]);
}

async function capture(db: Database, scopes: Map<string, Set<string>>[]) {
  return (await Promise.all(scopes.map((ids) => snapshot(db, ids)))).flat();
}

async function unchanged(db: Database, before: Snapshot[]) {
  await Promise.all(
    before.map(async ({ table, rows }) => {
      expect(
        await rowsById(
          db,
          table,
          rows.map((row) => row.id)
        ),
        `${table}: retained IDs and every column`
      ).toEqual(rows);
    })
  );
}

// A skipped PAT purge leaves an old ID behind; a broad cutoff removes recent rows or live neighboring chains.
test("I21 (API): a fresh purge after +61d physically removes old FK chains and PATs while recent and live rows survive", async ({
  clockWorld: world,
}, info) => {
  test.setTimeout(120_000);
  const db = world.db;
  const admin = await newAccount(world.current.api, info, "purge-admin");
  const member = await newAccount(world.current.api, info, "purge-member");
  const old = await seed(world.current.api, db, info, admin, member, "old");
  await remove(world.current.api, db, admin, old);
  const oldRows = await capture(db, old.scopes);
  for (const { table, rows } of oldRows) {
    expect(
      rows.every((row) => row.deleted_at !== null),
      `${table}: every old target has a real tombstone`
    ).toBe(true);
  }
  const oldNeighbors = await capture(db, [old.parents, old.neighborIDs]);
  const previous = world.current.nerve;
  await world.advance("1464h");
  expect(previous.exitCode, "the previous process exited before its replacement").toBe(0);
  const api = world.current.api;
  const recent = await seed(api, db, info, admin, member, "recent");
  await remove(api, db, admin, recent);
  const live = await seed(api, db, info, admin, member, "live");
  const recentRows = await capture(db, recent.scopes);
  for (const { table, rows } of recentRows) {
    expect(
      rows.every((row) => row.deleted_at !== null),
      `${table}: recent targets were really deleted`
    ).toBe(true);
  }
  const retained = await capture(db, [
    recent.parents,
    recent.neighborIDs,
    ...live.scopes,
    live.parents,
    live.neighborIDs,
  ]);
  // Expired old sessions do not imply deleted accounts: establish a new login in this process generation.
  await Promise.all(
    [admin, member].map(async (account) => {
      const session = await login(api, account.email);
      expect(await accountId(api, session.access_token)).toBe(account.id);
    })
  );
  const watermark = await archiveWatermark(db); // after the final successful API write, never rounded through Date
  await expect
    .poll(
      async () => (await completedPeriodicJobs(db, "cleanup.purge_deleted", watermark, world.current.startedAt)).length,
      { timeout: 15_000 }
    )
    .toBeGreaterThan(0);
  const fresh = await completedPeriodicJobs(db, "cleanup.purge_deleted", watermark, world.current.startedAt);
  await info.attach("purge-directory-and-attempts", {
    body: JSON.stringify(
      {
        watermark,
        startedAt: world.current.startedAt,
        fresh,
        old: oldRows,
        recent: recentRows,
        retained,
        oldNeighbors,
      },
      null,
      2
    ),
    contentType: "application/json",
  });
  console.log("I21 fresh purge", JSON.stringify({ watermark, startedAt: world.current.startedAt, fresh }));
  await Promise.all(
    oldRows.map(async ({ table, rows }) => {
      expect(
        await rowsById(
          db,
          table,
          rows.map((row) => row.id)
        ),
        `${table}: old saved IDs are physically absent`
      ).toEqual([]);
    })
  );
  await unchanged(db, recentRows);
  await unchanged(db, retained);
  await unchanged(db, oldNeighbors);
  expect(await db.query("SELECT id FROM users WHERE id=ANY($1::uuid[]) ORDER BY id", [[admin.id, member.id]])).toEqual(
    [admin.id, member.id].toSorted().map((id) => ({ id }))
  );
  expect((await issueRead(api, admin.token, live.root.id)).status).toBe(200);
  expect(await issueRead(api, admin.token, recent.root.id)).toEqual({ status: 404, code: "issue.not_found" });
});
