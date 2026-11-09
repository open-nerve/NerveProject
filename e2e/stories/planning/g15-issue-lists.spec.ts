import type { IssueListOrder } from "@nerve/api-client";

import { projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { validationFailed } from "../../fixtures/issue-planning";
import { listIssues, listMemberIssues, memberIssuesListed, type IssueQuery } from "../../fixtures/issues";
import {
  expectPages,
  g15World,
  ids,
  idsDown,
  inKeyOrder,
  memberships,
  seen,
  shown,
} from "../../fixtures/planning-lists";
import { expect, test } from "../../fixtures/test";

// G15, a project's work items listed by their cycle and their modules (M6 design G15, M6 P3b spec 2.1, A1–A3), through
// the API; the pages come with P7. The values of one field are alternatives, none meaning no cycle or no module, and
// different fields intersect. A work item is in the group of its cycle and in one of each of its modules, none's group
// last. The names order both ways with none last, ties by id in the order's direction. A cycle or a module a work item
// has left does not count, an archived one does. The list of a member's work items orders by the names across his
// projects but refuses the filters and the groups. The admin calls with a personal access token, the member with the
// token of his sign-in and his personal access token.

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
