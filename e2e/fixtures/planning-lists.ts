import { expect, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "./api";
import { expectIssuePage } from "./assert/issue";
import { login, newAccount } from "./auth";
import { createCycle, cycleCalls } from "./cycles";
import type { Database } from "./db";
import { createIssue, updateIssue, type IssueCreate, type IssuePage } from "./issues";
import { createModule, moduleCalls } from "./modules";

// The world and the helpers of the stories of G15, a project's work items listed by their cycle and their modules
// (M6 design G15, M6 P3b spec 2.1, A1–A3): the ids a list orders, the groups and cells it answers, its pages, and the
// rows it answers from.

/** The ids of items ascending: the order of a tie of their key, ascending. */
export function ids(...items: { id: string }[]): string[] {
  return items.map((i) => i.id).toSorted();
}

/** The ids of items descending: the order of a tie of their key, descending. */
export function idsDown(...items: { id: string }[]): string[] {
  return ids(...items).toReversed();
}

/** entries by their keys' order in a list's groups: the ids of their cycles or modules ascending, none's last. */
export function inKeyOrder<T>(entries: [key: string, value: T][]): [string, T][] {
  return entries.toSorted(([a], [b]) => (a === "none" ? 1 : b === "none" ? -1 : a < b ? -1 : 1));
}

/** A group or a cell as G15 compares it: its key, its total, the ids of its first page of one, and whether more follow. */
type Shown = [key: string, total: number, first: string[], more: boolean];

/** The group or the cell of key whose work items are items, in their order, as a page of one shows it. */
export function shown(key: string, items: { id: string }[]): Shown {
  return [key, items.length, items.slice(0, 1).map((i) => i.id), items.length > 1];
}

/** A group or a cell of a list's answer, as shown says it. */
export function seen(g: { key: string; total: number; data?: { id: string }[]; next_cursor?: string | null }): Shown {
  return [g.key, g.total, (g.data ?? []).map((i) => i.id), typeof g.next_cursor === "string"];
}

/**
 * Reads the pages of a list of want, limit work items a page, each with the cursor of the one before: each has want's
 * next limit, the list's total and a cursor while more follow.
 */
export async function expectPages(read: (cursor?: string) => Promise<IssuePage>, want: string[], limit: number) {
  const starts = want.flatMap((_, n) => (n % limit === 0 ? [n] : []));
  let cursor: string | undefined;
  await starts.reduce(async (before, start, n) => {
    await before;
    const page = await read(cursor);
    expectIssuePage(page, want.slice(start, start + limit), want.length, n < starts.length - 1);
    cursor = page.next_cursor ?? undefined;
  }, Promise.resolve());
}

/**
 * The rows of the work items of the project of projectId in its cycles and its modules, removed ones too: the cycle's
 * or the module's name and whether it is archived, the work item's name and whether the row is removed, the cycles'
 * first, each kind by name.
 */
export function memberships(db: Database, projectId: string) {
  return db.query(
    `SELECT * FROM (
       SELECT 'cycle' AS kind, t.name AS target, t.archived_at IS NOT NULL AS archived, i.name AS item,
              r.deleted_at IS NOT NULL AS removed
         FROM cycle_issues r JOIN cycles t ON t.id = r.cycle_id JOIN issues i ON i.id = r.issue_id
        WHERE r.project_id = $1
       UNION ALL
       SELECT 'module', t.name, t.archived_at IS NOT NULL, i.name, r.deleted_at IS NOT NULL
         FROM module_issues r JOIN modules t ON t.id = r.module_id JOIN issues i ON i.id = r.issue_id
        WHERE r.project_id = $1) m
      ORDER BY kind, target COLLATE "C", item COLLATE "C"`,
    [projectId]
  );
}

/**
 * Acme's world of G15. Web has its cycles Alpha, Beta and Old and its modules Core, Docs and Done; Ops its cycle Gamma
 * and its module Delta; the admin turns the cycles and modules of both on. Each work item is assigned to mem, whose
 * work items the member's list lists, and made by the admin, but Plan, which gil, a member of Web, makes:
 * - Fix the login, labelled Bug, in Beta, Core and Docs;
 * - Write the docs, in Alpha and Docs;
 * - Ship v1, labelled Bug, in Old and Done, which the admin then archives;
 * - Idea, labelled Bug, in none;
 * - Left behind, in none, having left Alpha and Core;
 * - Plan, in Beta and Core;
 * - Deploy, Ops', in Gamma and Delta.
 */
export async function g15World(api: Api, testInfo: TestInfo) {
  const admin = await newAccount(api, testInfo, "admin");
  const mem = await newAccount(api, testInfo, "mem");
  const gil = await newAccount(api, testInfo, "gil");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, gil, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  const ops = await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: gil.id, role: 15 },
  ]);
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: mem.id, role: 15 }]);
  expect(
    [
      await changeProject(api, admin.token, web.id, { cycle_view: true, module_view: true }),
      await changeProject(api, admin.token, ops.id, { cycle_view: true, module_view: true }),
    ].map((changed) => changed.status),
    "the admin turns the cycles and modules of Web and Ops on"
  ).toEqual([200, 200]);
  const cycle = (projectId: string, name: string) => createCycle(api, admin.token, projectId, { name });
  const [alpha, beta, old, gamma] = [
    await cycle(web.id, "Alpha"),
    await cycle(web.id, "Beta"),
    await cycle(web.id, "Old"),
    await cycle(ops.id, "Gamma"),
  ];
  const module = (projectId: string, name: string) => createModule(api, admin.token, projectId, { name });
  const [core, docs, done, delta] = [
    await module(web.id, "Core"),
    await module(web.id, "Docs"),
    await module(web.id, "Done"),
    await module(ops.id, "Delta"),
  ];
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const item = (token: string, projectId: string, body: IssueCreate) =>
    createIssue(api, token, projectId, { assignee_ids: [mem.id], ...body });
  const fix = await item(admin.token, web.id, {
    name: "Fix the login",
    label_ids: [bug.id],
    cycle_id: beta.id,
    module_ids: [core.id, docs.id],
  });
  const write = await item(admin.token, web.id, { name: "Write the docs", cycle_id: alpha.id, module_ids: [docs.id] });
  const ship = await item(admin.token, web.id, {
    name: "Ship v1",
    label_ids: [bug.id],
    cycle_id: old.id,
    module_ids: [done.id],
  });
  const idea = await item(admin.token, web.id, { name: "Idea", label_ids: [bug.id] });
  const left = await item(admin.token, web.id, { name: "Left behind", cycle_id: alpha.id, module_ids: [core.id] });
  await updateIssue(api, admin.token, left.id, { cycle_id: null, module_ids: [] });
  const plan = await item(gil.token, web.id, { name: "Plan", cycle_id: beta.id, module_ids: [core.id] });
  const deploy = await item(admin.token, ops.id, { name: "Deploy", cycle_id: gamma.id, module_ids: [delta.id] });
  // Old, a draft, ends in 2025 and is archived; Done is completed and archived. Ship v1 stays in both.
  const cycles = cycleCalls(api);
  const modules = moduleCalls(api);
  expect(
    [
      await cycles.update(admin.token, old.id, { start_date: "2025-01-01", end_date: "2025-01-14" }),
      await cycles.archive(admin.token, old.id),
      await modules.update(admin.token, done.id, { status: "completed" }),
      await modules.archive(admin.token, done.id),
    ],
    "the admin ends Old and archives it, and completes Done and archives it"
  ).toEqual(Array(4).fill({ status: 200 }));
  const browser = (await login(api, mem.email)).access_token;
  return {
    admin,
    mem,
    gil,
    browser,
    slug,
    web,
    bug,
    alpha,
    beta,
    old,
    core,
    docs,
    done,
    fix,
    write,
    ship,
    idea,
    left,
    plan,
    deploy,
  };
}
