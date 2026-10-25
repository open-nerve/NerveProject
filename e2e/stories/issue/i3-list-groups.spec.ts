import {
  addProjectMembers,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { byId, expectIssuePage } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { createIssue, issuesListed, listIssues, updateIssue, type IssueQuery } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

test("I3 (API): state groups and priority-label lanes count unique issues; complete arrays survive double expansion and A-filter C-group continuation", async ({
  api,
}, info) => {
  const x = await newAccount(api, info, "x");
  const y = await newAccount(api, info, "y");
  const slug = slugFor(info);
  await createWorkspace(api, x.token, { name: "Acme", slug });
  await inviteAndAccept(api, x.token, slug, y, 15);
  const project = await createProject(api, x.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, x.token, project.id, [{ member_id: y.id, role: 15 }]);
  const a = await createLabel(api, x.token, project.id, { name: "A" });
  const c = await createLabel(api, x.token, project.id, { name: "C" });
  const states = await stateIds(api, x.token, project.id);
  const todo = states.Todo;
  const done = states.Done;
  if (!todo || !done) throw new Error("expected Todo and Done");
  const both = [a, c].toSorted(byId).map((v) => v.id);
  const people = [x, y].toSorted(byId).map((v) => v.id);
  const rows: Awaited<ReturnType<typeof createIssue>>[] = [];
  await (
    [
      ["p1", both, people, todo, "high", 10],
      ["pC", [c.id], [x.id], todo, "high", 20],
      ["p2", both, [x.id], todo, "high", 30],
      ["pA", [a.id], [y.id], done, "low", 40],
      ["p0", [], [], done, "low", 50],
    ] as const
  ).reduce(async (before, [name, labels, assignees, state, priority, sort]) => {
    await before;
    const issue = await createIssue(api, x.token, project.id, {
      name,
      label_ids: [...labels],
      assignee_ids: [...assignees],
      state_id: state,
      priority,
    });
    await updateIssue(api, x.token, issue.id, { sort_order: sort });
    rows.push(issue);
  }, Promise.resolve());
  const [p1, pc, p2, pa, p0] = rows;
  if (!p1 || !pc || !p2 || !pa || !p0) throw new Error("missing fixture issue");
  const read = (query: IssueQuery) =>
    listIssues(api, x.token, project.id, { order_by: "sort_order", limit: 1, ...query });
  const grouped = await read({ group_by: "state_id" });
  expect(Object.keys(grouped).toSorted()).toEqual(["groups", "total"]);
  expect(grouped.total).toBe(5);
  expect(grouped.groups?.map((g) => g.key)).toEqual([todo, done].toSorted());
  await (
    [
      [todo, [p1.id, pc.id, p2.id]],
      [done, [pa.id, p0.id]],
    ] as const
  ).reduce(async (before, [key, ids]) => {
    await before;
    const group = grouped.groups?.find((g) => g.key === key);
    expect(group).toMatchObject({ total: ids.length, data: [{ id: ids[0] }], next_cursor: expect.any(String) });
    let cursor = group?.next_cursor;
    await ids.slice(1).reduce(async (previous, id, index) => {
      await previous;
      if (!cursor) throw new Error("group omitted a real cursor");
      const page = await read({ group_by: "state_id", group_key: key, cursor });
      expectIssuePage(page, [id], ids.length, index < ids.length - 2);
      cursor = page.next_cursor;
    }, Promise.resolve());
  }, Promise.resolve());
  const lanes = await read({ group_by: "priority", sub_group_by: "label_id" });
  expect(Object.keys(lanes).toSorted()).toEqual(["groups", "subgroup_totals", "total"]);
  expect(lanes.total).toBe(5);
  expect(lanes.groups?.map((g) => [g.key, g.total])).toEqual([
    ["high", 3],
    ["low", 2],
  ]);
  const labelKeys = [a.id, c.id].toSorted();
  expect(lanes.subgroup_totals).toEqual([...labelKeys.map((key) => ({ key, total: 3 })), { key: "none", total: 1 }]);
  await (
    [
      [
        "high",
        [
          [a.id, [p1.id, p2.id]],
          [c.id, [p1.id, pc.id, p2.id]],
        ],
      ],
      [
        "low",
        [
          [a.id, [pa.id]],
          ["none", [p0.id]],
        ],
      ],
    ] as const
  ).reduce(async (before, [priority, cells]) => {
    await before;
    const group = lanes.groups?.find((g) => g.key === priority);
    expect(group && Object.keys(group).toSorted()).toEqual(["key", "subgroups", "total"]);
    expect(group?.subgroups?.map((g) => g.key)).toEqual(cells.map(([key]) => key).toSorted());
    await cells.reduce(async (previous, [key, ids]) => {
      await previous;
      const cell = group?.subgroups?.find((g) => g.key === key);
      expect(cell).toMatchObject({
        total: ids.length,
        data: [{ id: ids[0] }],
        next_cursor: ids.length > 1 ? expect.any(String) : null,
      });
      let cursor = cell?.next_cursor;
      await ids.slice(1).reduce(async (prior, id, index) => {
        await prior;
        if (!cursor) throw new Error("cell omitted cursor");
        const page = await read({
          group_by: "priority",
          sub_group_by: "label_id",
          group_key: priority,
          sub_group_key: key,
          cursor,
        });
        expectIssuePage(page, [id], ids.length, index < ids.length - 2);
        cursor = page.next_cursor;
      }, Promise.resolve());
    }, Promise.resolve());
  }, Promise.resolve());
  const expanded = await listIssues(api, x.token, project.id, {
    group_by: "label_id",
    sub_group_by: "assignee_id",
    limit: 10,
  });
  for (const label of both)
    for (const person of people) {
      const item = expanded.groups
        ?.find((g) => g.key === label)
        ?.subgroups?.find((g) => g.key === person)
        ?.data.find((i) => i.id === p1.id);
      expect(item).toMatchObject({ label_ids: both, assignee_ids: people });
      expect(item).not.toHaveProperty("description_html");
    }
  const filtered = await read({ label_id: [a.id], group_by: "label_id" });
  expect(filtered.total).toBe(3);
  const cGroup = filtered.groups?.find((g) => g.key === c.id);
  expect(cGroup).toMatchObject({ total: 2, data: [{ id: p1.id }] });
  if (!cGroup?.next_cursor) throw new Error("C group needs second page");
  const second = await read({ label_id: [a.id], group_by: "label_id", group_key: c.id, cursor: cGroup.next_cursor });
  expectIssuePage(second, [p2.id], 2);
  expect(second.data?.[0]).toMatchObject({ label_ids: both, assignee_ids: [x.id] });
  expect(
    await issuesListed(api, x.token, project.id, {
      // @ts-expect-error -- a future sort is rejected by the real endpoint
      order_by: "link_count",
    })
  ).toEqual({ status: 422, code: "validation_failed", errors: [{ field: "order_by", code: "not_allowed" }] });
});
