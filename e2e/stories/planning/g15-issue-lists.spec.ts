import type { IssueListOrder } from "@nerve/api-client";
import type { TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import type { Database } from "../../fixtures/db";
import { validationFailed } from "../../fixtures/issue-planning";
import {
  createIssue,
  listIssues,
  listMemberIssues,
  memberIssuesListed,
  updateIssue,
  type IssueCreate,
  type IssuePage,
  type IssueQuery,
} from "../../fixtures/issues";
import { createModule, moduleCalls } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G15, a project's work items listed by their cycle and their modules (M6 design G15, M6 P3b spec 2.1, A1–A3), through
// the API; the pages come with P7. The values of one field are alternatives, none meaning no cycle or no module, and
// different fields intersect. A work item is in the group of its cycle and in one of each of its modules, none's group
// last. The names order both ways with none last, ties by id in the order's direction. A cycle or a module a work item
// has left does not count, an archived one does. The list of a member's work items orders by the names across his
// projects but refuses the filters and the groups. The admin calls with a personal access token, the member with the
// token of his sign-in and his personal access token.

/** The ids of items ascending: the order of a tie of their key, ascending. */
function ids(...items: { id: string }[]): string[] {
  return items.map((i) => i.id).toSorted();
}

/** The ids of items descending: the order of a tie of their key, descending. */
function idsDown(...items: { id: string }[]): string[] {
  return ids(...items).toReversed();
}

/** entries by their keys' order in a list's groups: the ids of their cycles or modules ascending, none's last. */
function inKeyOrder<T>(entries: [key: string, value: T][]): [string, T][] {
  return entries.toSorted(([a], [b]) => (a === "none" ? 1 : b === "none" ? -1 : a < b ? -1 : 1));
}

/** A group or a cell as G15 compares it: its key, its total, the ids of its first page of one, and whether more follow. */
type Shown = [key: string, total: number, first: string[], more: boolean];

/** The group or the cell of key whose work items are items, in their order, as a page of one shows it. */
function shown(key: string, items: { id: string }[]): Shown {
  return [key, items.length, items.slice(0, 1).map((i) => i.id), items.length > 1];
}

/** A group or a cell of a list's answer, as shown says it. */
function seen(g: { key: string; total: number; data?: { id: string }[]; next_cursor?: string | null }): Shown {
  return [g.key, g.total, (g.data ?? []).map((i) => i.id), typeof g.next_cursor === "string"];
}

/**
 * Reads the pages of a list of want, limit work items a page, each with the cursor of the one before: each has want's
 * next limit, the list's total and a cursor while more follow.
 */
async function expectPages(read: (cursor?: string) => Promise<IssuePage>, want: string[], limit: number) {
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
function memberships(db: Database, projectId: string) {
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
async function g15World(api: Api, testInfo: TestInfo) {
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

test("G15 (API): Web's work items filtered by Alpha, by Old, archived, by none, by none or Alpha, by Core or Docs, by Done, archived, by no module, and by a cycle with a label or a module, Left behind counted in neither Alpha nor Core it has left; grouped by cycle and by module, Fix the login in both Core's and Docs' groups, none's last, each group continued; laned by cycle and module, each cell continued; gil, made Web's guest, sees and counts only Plan, his own; the list of mem's work items refuses the filters and the groups", async ({
  api,
  db,
}, testInfo) => {
  const w = await g15World(api, testInfo);
  const { admin, gil, web, bug, alpha, beta, old, core, docs, done, fix, write, ship, idea, left, plan } = w;
  const listed = (query: IssueQuery, token = admin.token) =>
    listIssues(api, token, web.id, { order_by: "created_at", ...query });

  // The rows the lists answer from: Left behind's of Alpha and Core removed, Old and Done archived with Ship v1's.
  expect(await memberships(db, web.id), "Web's rows of its cycles and modules").toEqual([
    { kind: "cycle", target: "Alpha", archived: false, item: "Left behind", removed: true },
    { kind: "cycle", target: "Alpha", archived: false, item: "Write the docs", removed: false },
    { kind: "cycle", target: "Beta", archived: false, item: "Fix the login", removed: false },
    { kind: "cycle", target: "Beta", archived: false, item: "Plan", removed: false },
    { kind: "cycle", target: "Old", archived: true, item: "Ship v1", removed: false },
    { kind: "module", target: "Core", archived: false, item: "Fix the login", removed: false },
    { kind: "module", target: "Core", archived: false, item: "Left behind", removed: true },
    { kind: "module", target: "Core", archived: false, item: "Plan", removed: false },
    { kind: "module", target: "Docs", archived: false, item: "Fix the login", removed: false },
    { kind: "module", target: "Docs", archived: false, item: "Write the docs", removed: false },
    { kind: "module", target: "Done", archived: true, item: "Ship v1", removed: false },
  ]);

  // Filtered, in the order they were made: the values of a field alternatives, different fields intersecting.
  await (
    [
      ["Alpha", { cycle_id: [alpha.id] }, [write]],
      ["Old, archived", { cycle_id: [old.id] }, [ship]],
      ["no cycle", { cycle_id: ["none"] }, [idea, left]],
      ["no cycle or Alpha", { cycle_id: ["none", alpha.id] }, [write, idea, left]],
      ["Core or Docs", { module_id: [core.id, docs.id] }, [fix, write, plan]],
      ["Done, archived", { module_id: [done.id] }, [ship]],
      ["no module", { module_id: ["none"] }, [idea, left]],
      ["Beta or Old, and Bug", { cycle_id: [beta.id, old.id], label_id: [bug.id] }, [fix, ship]],
      ["no cycle, and Bug", { cycle_id: ["none"], label_id: [bug.id] }, [idea]],
      ["Beta and Docs", { cycle_id: [beta.id], module_id: [docs.id] }, [fix]],
      ["Alpha and Core", { cycle_id: [alpha.id], module_id: [core.id] }, []],
    ] satisfies [string, IssueQuery, { id: string }[]][]
  ).reduce(async (before, [name, query, want]) => {
    await before;
    await test.step(`filtered by ${name}`, async () => {
      expectIssuePage(
        await listed(query),
        want.map((i) => i.id)
      );
    });
  }, Promise.resolve());

  // Grouped, a work item a page: each group's total and first page, none's last; each group continued from its
  // cursor, within the filters of the first page. The list's total counts each work item once.
  const expectGroups = async (query: IssueQuery, groups: [string, { id: string }[]][], total: number) => {
    const page = await listed({ ...query, limit: 1 });
    expect(Object.keys(page).toSorted()).toEqual(["groups", "total"]);
    expect(page.total, "the list's total").toBe(total);
    expect(page.groups?.map(seen), "the groups").toEqual(inKeyOrder(groups).map(([key, items]) => shown(key, items)));
    await groups.reduce(async (before, [key, items]) => {
      await before;
      let cursor = page.groups?.find((g) => g.key === key)?.next_cursor ?? undefined;
      await items.slice(1).reduce(async (prior, item, n) => {
        await prior;
        if (!cursor) throw new Error(`the group ${key} has no next page`);
        const next = await listed({ ...query, limit: 1, group_key: key, cursor });
        expectIssuePage(next, [item.id], items.length, n < items.length - 2);
        cursor = next.next_cursor ?? undefined;
      }, Promise.resolve());
    }, Promise.resolve());
  };
  await test.step("grouped by cycle", () =>
    expectGroups(
      { group_by: "cycle_id" },
      [
        [alpha.id, [write]],
        [beta.id, [fix, plan]],
        [old.id, [ship]],
        ["none", [idea, left]],
      ],
      6
    ));
  await test.step("grouped by module", () =>
    expectGroups(
      { group_by: "module_id" },
      [
        [core.id, [fix, plan]],
        [docs.id, [fix, write]],
        [done.id, [ship]],
        ["none", [idea, left]],
      ],
      6
    ));
  await test.step("filtered by no cycle or Alpha, grouped by module", () =>
    expectGroups(
      { cycle_id: ["none", alpha.id], group_by: "module_id" },
      [
        [docs.id, [write]],
        ["none", [idea, left]],
      ],
      3
    ));

  // Laned: cycles' groups of modules' cells; the lanes' totals count each work item in each of its modules.
  await test.step("grouped by cycle, laned by module", async () => {
    const query: IssueQuery = { group_by: "cycle_id", sub_group_by: "module_id", limit: 1 };
    const lanes = await listed(query);
    expect(Object.keys(lanes).toSorted()).toEqual(["groups", "subgroup_totals", "total"]);
    expect(lanes.total).toBe(6);
    expect(lanes.subgroup_totals).toEqual(
      inKeyOrder<number>([
        [core.id, 2],
        [docs.id, 2],
        [done.id, 1],
        ["none", 2],
      ]).map(([key, total]) => ({ key, total }))
    );
    expect(lanes.groups?.map((g) => [g.key, g.total, g.subgroups?.map(seen)])).toEqual(
      inKeyOrder<[number, [string, { id: string }[]][]]>([
        [alpha.id, [1, [[docs.id, [write]]]]],
        [
          beta.id,
          [
            2,
            [
              [core.id, [fix, plan]],
              [docs.id, [fix]],
            ],
          ],
        ],
        [old.id, [1, [[done.id, [ship]]]]],
        ["none", [2, [["none", [idea, left]]]]],
      ]).map(([key, [total, cells]]) => [key, total, inKeyOrder(cells).map(([cell, items]) => shown(cell, items))])
    );
    await (
      [
        [beta.id, core.id, plan],
        ["none", "none", left],
      ] as const
    ).reduce(async (before, [group, cell, item]) => {
      await before;
      const cursor = lanes.groups?.find((g) => g.key === group)?.subgroups?.find((c) => c.key === cell)?.next_cursor;
      if (!cursor) throw new Error(`the cell ${group}/${cell} has no next page`);
      expectIssuePage(await listed({ ...query, group_key: group, sub_group_key: cell, cursor }), [item.id], 2);
    }, Promise.resolve());
  });

  // gil, made Web's guest, sees only Plan, which he made: his filters, groups and totals count only it.
  expect(
    await projectMemberWrites(api, web.id).change(
      admin.token,
      await projectMembershipOf(api, admin.token, web.id, gil.id),
      5
    ),
    "the admin makes gil Web's guest"
  ).toEqual({ status: 200 });
  expectIssuePage(await listed({ cycle_id: [beta.id] }, gil.token), [plan.id]);
  const gils = [await listed({ group_by: "cycle_id" }, gil.token), await listed({ group_by: "module_id" }, gil.token)];
  expect(
    gils.map((page) => [page.total, page.groups?.map(seen)]),
    "Web's work items as gil groups them by cycle and by module"
  ).toEqual([
    [1, [shown(beta.id, [plan])]],
    [1, [shown(core.id, [plan])]],
  ]);

  // The list of mem's work items has no cycles nor modules to filter or group by.
  const mine = (query: Omit<Parameters<typeof memberIssuesListed>[4], "relation">) =>
    memberIssuesListed(api, w.browser, w.slug, w.mem.id, { relation: "assigned", ...query });
  expect(
    [
      // @ts-expect-error -- the member's list declares no cycle filter; the real endpoint refuses it
      await mine({ cycle_id: [alpha.id] }),
      // @ts-expect-error -- nor a module filter
      await mine({ module_id: ["none"] }),
      await mine({ group_by: "cycle_id" }),
      await mine({ group_by: "module_id" }),
    ],
    "mem's work items by Alpha, by no module, grouped by cycle and by module"
  ).toEqual([
    validationFailed(["cycle_id", "not_allowed"]),
    validationFailed(["module_id", "not_allowed"]),
    validationFailed(["group_by", "not_allowed"]),
    validationFailed(["group_by", "not_allowed"]),
  ]);
});

test("G15 (API): by cycle name and by module name both ways, Web's list and the list of mem's work items across Web and Ops traverse real second pages: Alpha, Beta, Gamma, Old, archived, and Core, Delta, Docs, Done, archived, a work item by its least module; ties by id in the order's direction; Idea and Left behind, in none, last either way; mem's list alike to the token of his sign-in and his personal access token", async ({
  api,
}, testInfo) => {
  const w = await g15World(api, testInfo);
  const { fix, write, ship, idea, left, plan, deploy } = w;
  const web = {
    cycle_name: [write.id, ...ids(fix, plan), ship.id, ...ids(idea, left)],
    "-cycle_name": [ship.id, ...idsDown(fix, plan), write.id, ...idsDown(idea, left)],
    module_name: [...ids(fix, plan), write.id, ship.id, ...ids(idea, left)],
    "-module_name": [ship.id, write.id, ...idsDown(fix, plan), ...idsDown(idea, left)],
  } satisfies Partial<Record<IssueListOrder, string[]>>;
  const mine = {
    cycle_name: [write.id, ...ids(fix, plan), deploy.id, ship.id, ...ids(idea, left)],
    "-cycle_name": [ship.id, deploy.id, ...idsDown(fix, plan), write.id, ...idsDown(idea, left)],
    module_name: [...ids(fix, plan), deploy.id, write.id, ship.id, ...ids(idea, left)],
    "-module_name": [ship.id, write.id, deploy.id, ...idsDown(fix, plan), ...idsDown(idea, left)],
  } satisfies typeof web;
  await (Object.keys(web) as (keyof typeof web)[]).reduce(async (before, order_by) => {
    await before;
    await test.step(`Web's list by ${order_by}, two a page`, () =>
      expectPages(
        (cursor) => listIssues(api, w.admin.token, w.web.id, { order_by, limit: 2, cursor }),
        web[order_by],
        2
      ));
    await [w.browser, w.mem.token].reduce(async (prior, token) => {
      await prior;
      await test.step(`mem's work items by ${order_by}, three a page`, () =>
        expectPages(
          (cursor) =>
            listMemberIssues(api, token, w.slug, w.mem.id, { relation: "assigned", order_by, limit: 3, cursor }),
          mine[order_by],
          3
        ));
    }, Promise.resolve());
  }, Promise.resolve());
});
