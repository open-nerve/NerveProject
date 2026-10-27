import type { IssueListOrder } from "@nerve/api-client";

import {
  addProjectMembers,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { named, newAccount } from "../../fixtures/auth";
import { instantKey } from "../../fixtures/instant";
import { createIssue, listIssues, updateIssue } from "../../fixtures/issues";
import { createState } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

test("I4 (API): all twenty-one orders traverse real second and terminal pages with distinct keys; calendar ranges AND and missing dates stay separate", async ({
  api,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  const people: Awaited<ReturnType<typeof newAccount>>[] = [];
  await ["Alpha", "Beta", "Zebra"].reduce(async (before, name) => {
    await before;
    const who = await newAccount(api, info, name);
    await named(api, who.token, { display_name: name });
    await inviteAndAccept(api, admin.token, slug, who, 15);
    await addProjectMembers(api, admin.token, project.id, [{ member_id: who.id, role: 15 }]);
    people.push(who);
  }, Promise.resolve());
  const [alpha, beta, zebra] = people;
  if (!alpha || !beta || !zebra) throw new Error("missing named assignees");
  const labels: Awaited<ReturnType<typeof createLabel>>[] = [];
  const states: Awaited<ReturnType<typeof createState>>[] = [];
  await ["Alpha", "Beta", "Zebra"].reduce(async (before, name) => {
    await before;
    labels.push(await createLabel(api, admin.token, project.id, { name }));
    states.push(await createState(api, admin.token, project.id, { name, color: "#123456", group: "unstarted" }));
  }, Promise.resolve());
  const [la, lb, lz] = labels,
    [sa, sb, sz] = states;
  if (!la || !lb || !lz || !sa || !sb || !sz) throw new Error("missing sort catalogs");
  const p = await createIssue(api, admin.token, project.id, {
    name: "P",
    state_id: sz.id,
    priority: "high",
    label_ids: [lz.id, la.id],
    assignee_ids: [zebra.id, alpha.id],
    start_date: "2026-10-02",
    target_date: "2026-10-03",
  });
  const q = await createIssue(api, admin.token, project.id, {
    name: "Q",
    state_id: sa.id,
    priority: "low",
    label_ids: [lb.id],
    assignee_ids: [beta.id],
    start_date: "2026-10-01",
  });
  const r = await createIssue(api, admin.token, project.id, {
    name: "R",
    state_id: sb.id,
    priority: "urgent",
    target_date: "2026-10-01",
  });
  expect(instantKey(p.created_at) < instantKey(q.created_at)).toBe(true);
  expect(instantKey(q.created_at) < instantKey(r.created_at)).toBe(true);
  await updateIssue(api, admin.token, p.id, { sort_order: 30 });
  await updateIssue(api, admin.token, q.id, { sort_order: 10 });
  await updateIssue(api, admin.token, r.id, { sort_order: 20 });
  await [p, p, r].reduce(async (before, parent) => {
    await before;
    await createIssue(api, admin.token, project.id, { name: "Child", parent_id: parent.id });
  }, Promise.resolve());
  const uq = await updateIssue(api, admin.token, q.id, { name: "Q updated" });
  const up = await updateIssue(api, admin.token, p.id, { name: "P updated" });
  const ur = await updateIssue(api, admin.token, r.id, { name: "R updated" });
  expect(instantKey(uq.updated_at) < instantKey(up.updated_at)).toBe(true);
  expect(instantKey(up.updated_at) < instantKey(ur.updated_at)).toBe(true);
  // Independent expected identities: no production comparator and no list response as the oracle.
  // All three have zero links: only the fixture UUIDs determine these ties.
  // Never sort the server response when asserting any order.
  const idOrder = [p.id, q.id, r.id].toSorted();
  const orders = {
    sort_order: [q.id, r.id, p.id],
    created_at: [p.id, q.id, r.id],
    "-created_at": [r.id, q.id, p.id],
    updated_at: [q.id, p.id, r.id],
    "-updated_at": [r.id, p.id, q.id],
    start_date: [q.id, p.id, r.id],
    "-start_date": [p.id, q.id, r.id],
    target_date: [r.id, p.id, q.id],
    "-target_date": [p.id, r.id, q.id],
    priority: [q.id, p.id, r.id],
    "-priority": [r.id, p.id, q.id],
    state_name: [q.id, r.id, p.id],
    "-state_name": [p.id, r.id, q.id],
    label_name: [p.id, q.id, r.id],
    "-label_name": [q.id, p.id, r.id],
    assignee_name: [p.id, q.id, r.id],
    "-assignee_name": [q.id, p.id, r.id],
    sub_issues_count: [q.id, r.id, p.id],
    "-sub_issues_count": [p.id, r.id, q.id],
    link_count: idOrder,
    "-link_count": idOrder.toReversed(),
  } satisfies Record<IssueListOrder, string[]>;
  expect(Object.keys(orders)).toHaveLength(21);
  await (Object.keys(orders) as IssueListOrder[]).reduce(async (before, order_by) => {
    await before;
    await test.step(`${order_by}: second page, terminal page, exact identities`, async () => {
      const wanted = orders[order_by];
      const seen: string[] = [];
      let cursor: string | undefined;
      await wanted.reduce(async (prior, id, n) => {
        await prior;
        const page = await listIssues(api, admin.token, project.id, { order_by, limit: 1, sub_issue: "false", cursor });
        expectIssuePage(page, [id], 3, n < 2);
        seen.push(...(page.data ?? []).map((row) => row.id));
        expect(page.data?.[0]?.sub_issues_count).toBe(
          page.data?.[0]?.id === p.id ? 2 : page.data?.[0]?.id === r.id ? 1 : 0
        );
        expect(page.data?.[0]?.link_count).toBe(0);
        cursor = page.next_cursor ?? undefined;
      }, Promise.resolve());
      expect(seen).toEqual(wanted);
      expect(new Set(seen).size).toBe(3);
    });
  }, Promise.resolve());
  const range = await listIssues(api, admin.token, project.id, {
    sub_issue: "false",
    target_date: ["2026-10-01..2026-10-31"],
    group_by: "target_date",
  });
  expect(range.total).toBe(2);
  expect(range.groups?.map((g) => [g.key, g.total, g.data?.map((i) => i.id), g.next_cursor])).toEqual([
    ["2026-10-01", 1, [r.id], null],
    ["2026-10-03", 1, [p.id], null],
  ]);
  expectIssuePage(
    await listIssues(api, admin.token, project.id, {
      sub_issue: "false",
      target_date: ["2026-10-01..2026-10-31", "..2026-10-02"],
    }),
    [r.id]
  );
  expectIssuePage(await listIssues(api, admin.token, project.id, { sub_issue: "false", target_date: ["none"] }), [
    q.id,
  ]);
  expectIssuePage(await listIssues(api, admin.token, project.id, { sub_issue: "false", start_date: ["none"] }), [r.id]);
  expectIssuePage(
    await listIssues(api, admin.token, project.id, {
      sub_issue: "false",
      target_date: ["2026-10-01..2026-10-02", "2026-10-03..2026-10-04"],
    }),
    []
  );
});
