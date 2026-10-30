import type { IssueListOrder } from "@nerve/api-client";

import {
  addProjectMembers,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  readProjectPreferences,
  slugFor,
} from "../../fixtures/api";
import { expectIssueCollection, expectIssuePage } from "../../fixtures/assert/issue";
import { named, newAccount } from "../../fixtures/auth";
import { instantKey } from "../../fixtures/instant";
import { createIssue, issueRead, listIssues, searchIssues, updateIssue } from "../../fixtures/issues";
import { startLinkPage } from "../../fixtures/link-page";
import { createIssueLink, issueLinkDeleted } from "../../fixtures/links";
import { createState } from "../../fixtures/states";
import {
  issueListProject,
  saveListPreferences,
  openIssueList,
  expectListPageQuiet,
} from "../../fixtures/issue-list-page";
import { byId, changed, createdFor, expectIssueUpdated, madeBy } from "../../fixtures/assert/issue";
import { dragOnto } from "../../fixtures/project-pages";
import { closedWithin } from "../../fixtures/dialogs";
import { sentHeld } from "../../fixtures/settings-pages";
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

test("I4 (API): link counts 0/1/2/2 traverse two limit-two pages in both directions with same-direction UUID ties; deletion lowers fresh list, detail and search counts", async ({
  api,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const writer = await newAccount(api, info, "writer");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Link orders", slug });
  await inviteAndAccept(api, admin.token, slug, writer, 15);
  const project = await createProject(api, admin.token, slug, { name: "Links", identifier: "LINKS" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: writer.id, role: 15 }]);
  const zero = await createIssue(api, writer.token, project.id, { name: "Zero" });
  const one = await createIssue(api, writer.token, project.id, { name: "One" });
  const twoA = await createIssue(api, writer.token, project.id, { name: "Two A" });
  const twoB = await createIssue(api, writer.token, project.id, { name: "Two B" });
  const page = await startLinkPage();
  try {
    const removable = await createIssueLink(api, writer.token, one.id, { url: `${page.url}?one` });
    await [twoA, twoB].reduce(async (before, issue) => {
      await before;
      await createIssueLink(api, writer.token, issue.id, { url: `${page.url}?first` });
      // Counts include another author and links before or after crawler metadata arrives.
      await createIssueLink(api, admin.token, issue.id, { url: `${page.url}?second` });
    }, Promise.resolve());
    const tied = [twoA.id, twoB.id].toSorted();
    const orders = {
      link_count: [zero.id, one.id, ...tied],
      "-link_count": [...tied.toReversed(), one.id, zero.id],
    };
    await (["link_count", "-link_count"] as const).reduce(async (before, order_by) => {
      await before;
      await test.step(`${order_by}: two exact pages, no duplicated or missing identity`, async () => {
        const expected = orders[order_by];
        const first = await listIssues(api, writer.token, project.id, { order_by, limit: 2 });
        expectIssuePage(first, expected.slice(0, 2), 4, true);
        const second = await listIssues(api, writer.token, project.id, {
          order_by,
          limit: 2,
          cursor: first.next_cursor!,
        });
        expectIssuePage(second, expected.slice(2), 4);
        const rows = [...(first.data ?? []), ...(second.data ?? [])];
        expect(rows.map((row) => row.id)).toEqual(expected);
        expect(new Set(rows.map((row) => row.id)).size).toBe(4);
        expect(rows.map((row) => row.link_count)).toEqual(order_by === "link_count" ? [0, 1, 2, 2] : [2, 2, 1, 0]);
      });
    }, Promise.resolve());
    const before = await searchIssues(api, writer.token, slug, { q: `LINKS-${one.sequence_id}` });
    expectIssueCollection(before, [one.id]);
    expect(before.data[0]?.link_count).toBe(1);
    expect(await issueRead(api, writer.token, one.id)).toMatchObject({ status: 200, issue: { link_count: 1 } });
    expect(await issueLinkDeleted(api, writer.token, removable.id)).toEqual({ status: 204 });
    // Start new requests after the write; an earlier cursor does not freeze changing counts.
    const fresh = await listIssues(api, writer.token, project.id, { order_by: "link_count", limit: 2 });
    expectIssuePage(fresh, [zero.id, one.id].toSorted(), 4, true);
    expect(fresh.data?.map((row) => row.link_count)).toEqual([0, 0]);
    const detail = await issueRead(api, writer.token, one.id);
    expect(detail).toMatchObject({ status: 200, issue: { id: one.id, link_count: 0 } });
    const searched = await searchIssues(api, writer.token, slug, { q: `LINKS-${one.sequence_id}` });
    expectIssueCollection(searched, [one.id]);
    expect(searched.data[0]?.link_count).toBe(0);
    if (!("issue" in detail)) throw new Error("the linked issue detail is missing");
    expect(searched.data[0]?.link_count).toBe(detail.issue.link_count);
  } finally {
    await page.close();
  }
});

test("I4 (page): table label, assignee, child-count and due-date orders each traverse a real second page in both directions", async ({
  api,
  page,
  browserName,
}, info) => {
  test.setTimeout(90_000);
  const seed = await issueListProject(api, info);
  const { member, project, slug } = seed;
  const people = await Promise.all(
    ["Alpha", "Beta"].map(async (name) => {
      const who = await newAccount(api, info, name);
      await named(api, who.token, { display_name: name });
      await inviteAndAccept(api, member.token, slug, who, 15);
      await addProjectMembers(api, member.token, project.id, [{ member_id: who.id, role: 15 }]);
      return who;
    })
  );
  const labels = await Promise.all(
    ["Alpha", "Beta"].map((name) => createLabel(api, member.token, project.id, { name }))
  );
  const rows: { id: string; label: number | null; assignee: number | null; children: number; date: string | null }[] =
    [];
  await Array.from({ length: 101 }, (_, index) => index).reduce(async (prior, index) => {
    await prior;
    const label = index % 3 === 2 ? null : index % 3;
    const assignee = index % 3 === 0 ? null : (index % 3) - 1;
    const children = index === 0 ? 1 : index === 100 ? 2 : 0;
    const date = index % 3 === 1 ? null : index % 3 === 0 ? "2026-10-01" : "2026-10-03";
    const issue = await createIssue(api, member.token, project.id, {
      name: `Table ${String(index).padStart(3, "0")}`,
      label_ids: label === null ? [] : [labels[label]!.id],
      assignee_ids: assignee === null ? [] : [people[assignee]!.id],
      ...(date ? { target_date: date } : {}),
    });
    rows.push({ id: issue.id, label, assignee, children, date });
    await Promise.all(
      Array.from({ length: children }, (_, child) =>
        createIssue(api, member.token, project.id, { name: `Child ${index}-${child}`, parent_id: issue.id })
      )
    );
  }, Promise.resolve());
  const prefs = await readProjectPreferences(api, member.token, project.id);
  await saveListPreferences(api, seed, {
    filters: {},
    display_filters: { ...prefs.display_filters, layout: "spreadsheet", order_by: "-created_at", sub_issue: false },
    display_properties: {
      ...prefs.display_properties,
      labels: true,
      assignee: true,
      due_date: true,
      sub_issue_count: true,
    },
  });
  const { watch, queries } = await openIssueList(page, seed);
  const table = page.getByRole("table");
  const tableRows = table.locator("tbody > tr");
  const sorts = [
    { column: "Labels", key: "label", order: "label_name", ascending: "A Z", descending: "Z A" },
    { column: "Assignees", key: "assignee", order: "assignee_name", ascending: "A Z", descending: "Z A" },
    {
      column: "Sub-work item",
      key: "children",
      order: "sub_issues_count",
      ascending: "Least Most",
      descending: "Most Least",
    },
    { column: "Due date", key: "date", order: "target_date", ascending: "Old New", descending: "New Old" },
  ] as const;
  await sorts
    .flatMap((sort) => (["ascending", "descending"] as const).map((direction) => ({ sort, direction })))
    .reduce(async (prior, { sort, direction }) => {
      await prior;
      await test.step(`${sort.order} ${direction}: actual header and scroll continuation`, async () => {
        const descending = direction === "descending";
        const wanted = rows
          .toSorted((a, b) => {
            const left = a[sort.key],
              right = b[sort.key];
            if (left === null && right !== null) return 1;
            if (right === null && left !== null) return -1;
            const compared = left === right ? byId(a, b) : left! < right! ? -1 : 1;
            return descending ? -compared : compared;
          })
          .map((row) => row.id);
        const order = `${descending ? "-" : ""}${sort.order}`;
        const fromQuery = queries.length;
        const heading = table.getByRole("columnheader", { name: sort.column, exact: true });
        await heading.getByRole("button").click();
        await page.getByRole("menuitem", { name: sort[direction], exact: true }).click();
        await expect(heading).toHaveAttribute("aria-sort", direction);
        await expect
          .poll(() => queries.slice(fromQuery).some((query) => query.get("order_by") === order && !query.has("cursor")))
          .toBe(true);
        await expect(tableRows).toHaveCount(100);
        const observed = new Map<number, string>();
        await Array.from({ length: Math.ceil(wanted.length / 10) }, (_, step) => step * 10).reduce(
          async (previous, index) => {
            await previous;
            await tableRows.nth(index).scrollIntoViewIfNeeded();
            await expect(tableRows.nth(index).locator('td[id^="issue-"]')).toHaveAttribute(
              "id",
              `issue-${wanted[index]}`
            );
            const visible = await tableRows.evaluateAll((elements) =>
              elements.map((row, position) => ({ position, id: row.querySelector('td[id^="issue-"]')?.id.slice(6) }))
            );
            for (const entry of visible) if (entry.id) observed.set(entry.position, entry.id);
          },
          Promise.resolve()
        );
        await expect(tableRows).toHaveCount(101);
        expect(wanted.map((_, index) => observed.get(index))).toEqual(wanted);
        const continuation = queries.slice(fromQuery).filter((query) => query.has("cursor"));
        expect(continuation).toHaveLength(1);
        expect(Object.fromEntries(continuation[0]!)).toMatchObject({
          order_by: order,
          limit: "100",
          sub_issue: "false",
        });
        await expect(table.getByRole("button", { name: "Load more" })).toHaveCount(0);
      });
    }, Promise.resolve());
  await expectListPageQuiet(page, watch, browserName, 1);
});

test.describe("calendar in UTC+14", () => {
  test.use({ timezoneId: "Pacific/Kiritimati" });
  test("I4 (page): month, week and day ranges stay local; dragging writes only due date and adding existing retains a refused selection", async ({
    api,
    db,
    page,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    const { member, project } = seed;
    const local = await page.evaluate(() => ({
      year: new Date().getFullYear(),
      month: new Date().getMonth(),
      day: new Date().getDate(),
      weekday: new Date().getDay(),
    }));
    const year = local.year;
    const source = `${year}-12-16`,
      target = `${year}-12-17`,
      crowdedDate = `${year}-12-15`;
    const monthStart = new Date(Date.UTC(year, 11, 1));
    monthStart.setUTCDate(1 - monthStart.getUTCDay());
    const monthEnd = new Date(Date.UTC(year + 1, 0, 0));
    monthEnd.setUTCDate(monthEnd.getUTCDate() + 6 - monthEnd.getUTCDay());
    const monthRange = `${monthStart.toISOString().slice(0, 10)}..${monthEnd.toISOString().slice(0, 10)}`;
    const weekStart = new Date(Date.UTC(year, local.month, local.day - local.weekday));
    const weekEnd = new Date(Date.UTC(year, local.month, local.day - local.weekday + 6));
    const weekRange = `${weekStart.toISOString().slice(0, 10)}..${weekEnd.toISOString().slice(0, 10)}`;
    const moving = await createIssue(api, member.token, project.id, { name: "Move date", target_date: source });
    const available = await createIssue(api, member.token, project.id, { name: "Available undated" });
    await createIssue(api, member.token, project.id, { name: "Starts too late", start_date: `${year}-12-25` });
    const crowded: Awaited<ReturnType<typeof createIssue>>[] = [];
    await [0, 1, 2, 3, 4].reduce(async (prior, index) => {
      await prior;
      crowded.push(
        await createIssue(api, member.token, project.id, { name: `Crowded ${index}`, target_date: crowdedDate })
      );
    }, Promise.resolve());
    const prefs = await readProjectPreferences(api, member.token, project.id);
    await saveListPreferences(api, seed, {
      filters: {},
      display_filters: {
        ...prefs.display_filters,
        layout: "calendar",
        order_by: "sort_order",
        calendar: { layout: "month", show_weekends: true },
      },
    });
    const { watch, queries } = await openIssueList(page, seed);
    const searches: URLSearchParams[] = [];
    page.on("request", (request) => {
      const url = new URL(request.url());
      if (request.method() === "GET" && url.pathname === `/api/v0/workspaces/${seed.slug}/issue-search`)
        searches.push(url.searchParams);
    });
    await page.getByRole("button", { name: /^[A-Z][a-z]+ [0-9]{4}$/, exact: true }).click();
    await page.getByRole("button", { name: "Dec", exact: true }).click();
    await page.keyboard.press("Escape");
    await expect
      .poll(() => queries.some((query) => query.get("target_date") === monthRange && query.get("limit") === "4"))
      .toBe(true);
    const day = (date: string) => page.locator(`button[aria-label="${date}"]`).locator("..");
    const crowdedTile = day(crowdedDate);
    await expect(crowdedTile.locator('[id^="issue-"]')).toHaveCount(4);
    await crowdedTile.getByRole("button", { name: "Load more", exact: true }).click();
    await expect
      .poll(() =>
        crowdedTile.locator('[id^="issue-"]').evaluateAll((elements) => elements.map((element) => element.id.slice(6)))
      )
      .toEqual(crowded.map((issue) => issue.id));
    const continuation = queries.find((query) => query.has("cursor") && query.get("group_key") === crowdedDate);
    expect(continuation && Object.fromEntries(continuation)).toMatchObject({
      group_by: "target_date",
      target_date: monthRange,
      group_key: crowdedDate,
      limit: "4",
    });
    const sourceCard = day(source).locator(`[id="issue-${moving.id}"]`);
    const held = await sentHeld(page, "PATCH", `/api/v0/issues/${moving.id}`, () =>
      dragOnto(page, sourceCard, sourceCard, day(target), { x: 65, y: 35 })
    );
    expect(held.body).toEqual({ target_date: target });
    await expect(sourceCard).toBeVisible();
    await expect(day(target).locator(`[id="issue-${moving.id}"]`)).toHaveCount(0);
    expect(await held.release().then((response) => response.status())).toBe(200);
    await expect(day(target).locator(`[id="issue-${moving.id}"]`)).toBeVisible();
    await expect(sourceCard).toHaveCount(0);
    expect(watch.apiRequests.filter((request) => request === `PATCH /api/v0/issues/${moving.id}`)).toHaveLength(1);
    await expectIssueUpdated(db, moving.id, {
      ...madeBy(member.email, "Move date"),
      target_date: target,
      by: member.email,
      activity: [...createdFor(), changed("target_date", source, target, "updated the target date to")],
      wrote: 1,
    });

    await day(target).getByRole("button", { name: "Add work item", exact: true }).click();
    await page.getByRole("menuitem", { name: "Add existing work item", exact: true }).click();
    const dialog = page.getByRole("dialog", { name: "Add existing work item", exact: true });
    const option = dialog.getByRole("option", { name: /Available undated/ });
    await expect(option).toBeVisible();
    expect(searches).toHaveLength(1);
    expect(Object.fromEntries(searches[0]!)).toEqual({ project_id: project.id, target_date: "none" });
    await expect(dialog.getByRole("option", { name: /Move date|Crowded|Starts too late/ })).toHaveCount(0);
    await option.click();
    // A concurrent real change makes this selected date invalid; the refusal must keep the same selection and modal.
    await updateIssue(api, member.token, available.id, { start_date: `${year}-12-25` });
    await dialog.getByRole("button", { name: "Add selected work items", exact: true }).click();
    await expect.poll(() => watch.apiFailures).toEqual([`422 PATCH /api/v0/issues/${available.id}`]);
    await expect(dialog.getByRole("heading", { name: "Add existing work item", exact: true })).toBeVisible();
    expect(await closedWithin(page)).toBe(false);
    await expect(dialog.getByRole("button", { name: "Remove Available undated", exact: true })).toBeVisible();
    expect(await issueRead(api, member.token, available.id)).toMatchObject({
      status: 200,
      issue: { target_date: null },
    });
    await updateIssue(api, member.token, available.id, { start_date: null });
    const added = await sentHeld(page, "PATCH", `/api/v0/issues/${available.id}`, () =>
      dialog.getByRole("button", { name: "Add selected work items", exact: true }).click()
    );
    expect(added.body).toEqual({ target_date: target });
    await expect(dialog.getByRole("heading", { name: "Add existing work item", exact: true })).toBeVisible();
    expect(await closedWithin(page)).toBe(false);
    await expect(day(target).locator(`[id="issue-${available.id}"]`)).toHaveCount(0);
    expect(await added.release().then((response) => response.status())).toBe(200);
    await expect(dialog).toHaveCount(0);
    await expect(day(target).locator(`[id="issue-${available.id}"]`)).toBeVisible();
    expect(await issueRead(api, member.token, available.id)).toMatchObject({
      status: 200,
      issue: { target_date: target },
    });

    await page.getByRole("button", { name: "Options", exact: true }).click();
    await page.getByRole("button", { name: "Week layout", exact: true }).click();
    await page.keyboard.press("Escape");
    await expect
      .poll(() => queries.some((query) => query.get("target_date") === weekRange && query.get("limit") === "30"))
      .toBe(true);
    await expectListPageQuiet(page, watch, browserName, 1, {
      failures: [`422 PATCH /api/v0/issues/${available.id}`],
      errors:
        browserName === "chromium"
          ? ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"]
          : [],
    });
  });
});
