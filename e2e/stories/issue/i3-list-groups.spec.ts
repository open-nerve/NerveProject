import {
  addProjectMembers,
  readProjectPreferences,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { byId, expectIssuePage } from "../../fixtures/assert/issue";
import { bearer, newAccount } from "../../fixtures/auth";
import { createIssue, issuesListed, listIssues, updateIssue, type IssueQuery } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

import {
  issueListProject,
  saveListPreferences,
  openIssueList,
  listGroup,
  expectListOrder,
  expectListPageQuiet,
} from "../../fixtures/issue-list-page";

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
  // Use the public client directly: CSV alternatives and repeated dates must
  // work without a fixture-only query serializer.
  const alternatives = await api.GET("/api/v0/projects/{project_id}/issues", {
    params: { path: { project_id: project.id }, query: { label_id: [a.id, c.id], order_by: "sort_order" } },
    headers: bearer(x.token),
  });
  expect(alternatives.response.status, JSON.stringify(alternatives.error)).toBe(200);
  if (!alternatives.data) throw new Error("CSV list omitted page");
  expectIssuePage(alternatives.data, [p1.id, pc.id, p2.id, pa.id], 4);
  await updateIssue(api, x.token, p1.id, { start_date: "2026-10-05" });
  await updateIssue(api, x.token, pc.id, { start_date: "2026-10-15" });
  await updateIssue(api, x.token, pa.id, { start_date: "2026-10-25" });
  await updateIssue(api, x.token, p2.id, { start_date: "2026-11-05" });
  await updateIssue(api, x.token, p0.id, { label_ids: [a.id], start_date: "2026-09-25" });
  const dated = await api.GET("/api/v0/projects/{project_id}/issues", {
    params: {
      path: { project_id: project.id },
      query: { label_id: [a.id, c.id], start_date: ["2026-10-01..", "..2026-10-31"], order_by: "sort_order" },
    },
    headers: bearer(x.token),
  });
  expect(dated.response.status, JSON.stringify(dated.error)).toBe(200);
  if (!dated.data) throw new Error("dated CSV list omitted page");
  expectIssuePage(dated.data, [p1.id, pc.id, pa.id], 3);
  expect(
    await issuesListed(api, x.token, project.id, {
      // @ts-expect-error -- a future sort is rejected by the real endpoint
      order_by: "unknown_count",
    })
  ).toEqual({ status: 422, code: "validation_failed", errors: [{ field: "order_by", code: "not_allowed" }] });
});

test("I3 (page): real state continuation, saved A intersection C and label-priority lanes keep order, complete labels and unique totals", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member, project, states } = seed;
  const a = await createLabel(api, member.token, project.id, { name: "Label A" });
  const c = await createLabel(api, member.token, project.id, { name: "Label C" });
  const labels = [a, c].toSorted(byId).map((label) => label.id);
  const rows: Awaited<ReturnType<typeof createIssue>>[] = [];
  await Array.from({ length: 51 }, (_, index) => index).reduce(async (prior, index) => {
    await prior;
    rows.push(
      await createIssue(api, member.token, project.id, {
        name: `Both ${String(index + 1).padStart(2, "0")}`,
        state_id: states.Todo,
        label_ids: labels,
        priority: "high",
      })
    );
  }, Promise.resolve());
  const cOnly = await createIssue(api, member.token, project.id, {
    name: "C only outside A",
    state_id: states.Todo,
    label_ids: [c.id],
    priority: "high",
  });
  const aOnly = await createIssue(api, member.token, project.id, {
    name: "A only",
    state_id: states.Done,
    label_ids: [a.id],
    priority: "low",
  });
  const prefs = await readProjectPreferences(api, member.token, project.id);
  const display = {
    ...prefs.display_filters,
    layout: "list" as const,
    group_by: "state_id" as const,
    sub_group_by: null,
    order_by: "sort_order" as const,
    show_empty_groups: false,
  };
  await saveListPreferences(api, seed, {
    filters: {},
    display_filters: display,
    display_properties: { ...prefs.display_properties, labels: true },
  });
  const { watch, queries } = await openIssueList(page, seed);
  const todo = listGroup(page, "Todo");
  await expectListOrder(
    todo,
    rows.slice(0, 50).map((row) => row.id)
  );
  await expect(todo.locator('[class~="group/list-header"]')).toHaveText("Todo52");
  const labelsButton = todo
    .getByRole("button", { name: "Labels: Both 01", exact: true })
    .filter({ hasText: "Label A" });
  await labelsButton.click();
  await expect(page.getByRole("option", { name: "Label A", exact: true })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("option", { name: "Label C", exact: true })).toHaveAttribute("aria-selected", "true");
  await page.keyboard.press("Escape");
  await expect(labelsButton).toBeFocused();
  await todo.getByRole("button", { name: "Load more" }).focus();
  await page.keyboard.press("Enter");
  await expectListOrder(
    todo,
    [...rows, cOnly].map((row) => row.id)
  );
  await expect(todo.getByRole("button", { name: "Load more" })).toHaveCount(0);
  expect(Object.fromEntries(queries.find((query) => query.has("cursor"))!)).toMatchObject({
    group_by: "state_id",
    group_key: states.Todo,
    limit: "50",
    order_by: "sort_order",
  });
  await expectListOrder(listGroup(page, "Done"), [aOnly.id]);

  await saveListPreferences(api, seed, {
    filters: { label_id: [a.id] },
    display_filters: { ...display, group_by: "label_id" },
  });
  await page.reload();
  const cGroup = listGroup(page, "Label C");
  await expectListOrder(
    cGroup,
    rows.slice(0, 50).map((row) => row.id)
  );
  await cGroup.getByRole("button", { name: "Load more" }).click();
  await expectListOrder(
    cGroup,
    rows.map((row) => row.id)
  );
  await expect(cGroup.locator('[class~="group/list-header"]')).toHaveText("Label C51");
  const cQuery = queries.find((query) => query.get("group_key") === c.id && query.has("cursor"));
  expect(cQuery && Object.fromEntries(cQuery)).toMatchObject({
    label_id: a.id,
    group_by: "label_id",
    group_key: c.id,
    order_by: "sort_order",
    limit: "50",
  });
  const confirmed = await listIssues(api, member.token, project.id, {
    label_id: [a.id],
    group_by: "label_id",
    group_key: c.id,
    order_by: "sort_order",
    limit: 100,
  });
  expect(confirmed.data?.map((row) => row.label_ids)).toEqual(rows.map(() => labels));
  // Change layout through the actual toolbar; then reload the saved grouping for a real swimlane page.
  const boardRead = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      url.pathname === `/api/v0/projects/${project.id}/issues` &&
      url.searchParams.get("group_by") === "label_id" &&
      url.searchParams.get("limit") === "30"
    );
  });
  await page.getByRole("button", { name: "Board Layout", exact: true }).click();
  expect(await (await boardRead).finished()).toBeNull();
  await expect
    .poll(async () => (await readProjectPreferences(api, member.token, project.id)).display_filters.layout)
    .toBe("kanban");
  await saveListPreferences(api, seed, {
    filters: {},
    display_filters: { ...display, layout: "kanban", group_by: "priority", sub_group_by: "label_id" },
  });
  await page.reload();
  await expect(page.getByRole("button", { name: "Label A 52", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Label C 52", exact: true })).toBeVisible();
  expect(
    queries.some(
      (query) =>
        query.get("group_by") === "priority" && query.get("sub_group_by") === "label_id" && query.get("limit") === "10"
    )
  ).toBe(true);
  await expectListPageQuiet(page, watch);
});
