import { readProjectPreferences } from "../../fixtures/api";
import { expectModuleRows, lastWrite } from "../../fixtures/assert/issue-planning";
import { anotherTabSignsIn } from "../../fixtures/auth";
import { followAccessToken } from "../../fixtures/browser";
import { toast } from "../../fixtures/cycle-pages";
import { modalDialog } from "../../fixtures/dialogs";
import { displayPanel } from "../../fixtures/issue-filters-page";
import { expectListOrder, listGroup, scrollToLastListRow } from "../../fixtures/issue-list-page";
import { ascendingIds } from "../../fixtures/issue-planning";
import { issueRead, listIssues, type Issue } from "../../fixtures/issues";
import { turnModules } from "../../fixtures/module-pages";
import { changeModulePreferences, modulePreferencesRead } from "../../fixtures/modules";
import {
  bodiesSent,
  chooseFromRow,
  deleteAttachment,
  dialogReads,
  headerCount,
  listQueries,
  listRow,
  loadedAnew,
  moduleListWorld,
  openListPage,
  quickAddButton,
  quietEachTime,
  rowEntries,
  rowOptions,
  shownItems,
  tableOrder,
  weekdayThisMonth,
  workItemLayoutButton,
} from "../../fixtures/planning-list-pages";
import { expectPages, ids, idsDown, memberships } from "../../fixtures/planning-lists";
import { moveWithinApp } from "../../fixtures/settings-pages";
import { storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

// G15, a module's page's work items (M6 P6b spec 2.3–2.7, 2.10; M4 handoffs M6-01, M6-02; M5 handoff M5-M6-02), as
// mem uses them in his browser: the module's own work items in each layout under the page's own preferences, a work
// item of two modules on both their pages, an archived module's read-only, nothing read while the project's modules
// are off; the pages of a list, its counts after a write across its groups and an archiving, its attachments; the
// page's new work items joining the module in one request each, a removal from the module that leaves the work item's
// other modules as they are, an archived one too, a row's edit dialog that outlives the page's check of it and changes
// no module, and an archived module's page offering none of these. Each page ends without an unexpected failure
// (expectListPageQuiet). The world is made through the API, nerve's answers are read back through it and the database.

test("G15 (page): the module Now's page lists only Now's work items in the list, the board, the calendar and the table, each layout saved as the page's own, and Shared, of Now and Other, on both their pages; its grouping and a property, changed, stay when the page is loaded again and Web's own preferences do not change; Old's page, archived, lists its own read-only; with Web's modules off Now's page, loaded again, reads neither its preferences nor its work items", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const due = weekdayThisMonth();
  const [now, other, old] = [await w.module("Now"), await w.module("Other"), await w.module("Old")];
  await w.item("Plan", { module_ids: [now.id], target_date: due });
  const build = await w.item("Build", { module_ids: [now.id], target_date: due, priority: "high" });
  await w.item("Shared", { module_ids: [now.id, other.id], target_date: due });
  await w.item("Elsewhere", { module_ids: [other.id], target_date: due });
  await w.item("Loose", { target_date: due });
  await w.item("Shipped", { module_ids: [old.id], target_date: due });
  await w.archive(old.id);
  const token = w.mem.tokens.access_token;
  const projectPreferences = await readProjectPreferences(api, token, w.web.id);
  // Now's page as nerve keeps it for mem: its layout, its grouping and whether it shows the work items' IDs
  const saved = async () => {
    const read = await modulePreferencesRead(api, token, now.id);
    if (!("preferences" in read)) throw new Error(`Now's preferences: ${JSON.stringify(read)}`);
    const { display_filters, display_properties } = read.preferences;
    return { layout: display_filters.layout, group_by: display_filters.group_by, id: display_properties.key };
  };

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(now.id));
  const quiet = quietEachTime(page, watch);
  // Each layout lists Now's three work items, each in that layout's own place for it (a list's group, a board's column
  // of its state, the calendar's day of its date, the table), and nerve keeps the layout as the page's own.
  const places = {
    List: page.locator('[class~="group/list-header"]').locator("../.."),
    Board: page.locator(`[id^="${build.state_id}__"]`),
    Calendar: page.locator(`button[aria-label="${due}"]`).locator("xpath=preceding-sibling::div[1]"),
    Table: page.getByRole("table"),
  };
  await (
    [
      ["List", "list"],
      ["Board", "kanban"],
      ["Calendar", "calendar"],
      ["Table", "spreadsheet"],
    ] as const
  ).reduce(async (before, [layout, key]) => {
    await before;
    await test.step(`the ${layout.toLowerCase()} layout`, async () => {
      if ((await workItemLayoutButton(page, layout).getAttribute("aria-pressed")) !== "true")
        await workItemLayoutButton(page, layout).click();
      await expect(workItemLayoutButton(page, layout)).toHaveAttribute("aria-pressed", "true");
      await expect(places[layout].filter({ has: rowOptions(page, "Build") })).toHaveCount(1);
      await expect(places[layout].filter({ has: rowOptions(page, "Plan") })).toHaveCount(1);
      await expect(places[layout].filter({ has: rowOptions(page, "Shared") })).toHaveCount(1);
      await expect.poll(() => shownItems(page)).toEqual(["Build", "Plan", "Shared"]);
      await expect.poll(async () => (await saved()).layout).toBe(key);
      await quiet();
    });
  }, Promise.resolve());

  // The list grouped by priority, without the work items' IDs: as the page's own preferences, which it shows when
  // loaded again, in another browser of mem's.
  await workItemLayoutButton(page, "List").click();
  const panel = await displayPanel(page);
  const groupBy = panel.getByRole("button", { name: "Group by", exact: true });
  if ((await groupBy.getAttribute("aria-expanded")) !== "true") await groupBy.click();
  await groupBy.locator("../..").getByRole("button", { name: "Priority", exact: true }).click();
  await panel.getByRole("button", { name: "ID", exact: true }).click();
  await expect(panel.getByRole("button", { name: "ID", exact: true })).toHaveAttribute("aria-pressed", "false");
  await page.keyboard.press("Escape");
  await expect.poll(saved).toEqual({ layout: "list", group_by: "priority", id: false });
  await loadedAnew(browser, baseURL ?? "", api, w.mem, w.path(now.id), async (again) => {
    await expect(workItemLayoutButton(again, "List")).toHaveAttribute("aria-pressed", "true");
    await expectListOrder(listGroup(again, "High"), [build.id]);
    await expect.poll(() => shownItems(again)).toEqual(["Build", "Plan", "Shared"]);
    await expect((await displayPanel(again)).getByRole("button", { name: "ID", exact: true })).toHaveAttribute(
      "aria-pressed",
      "false"
    );
    await again.keyboard.press("Escape");
  });
  expect(await readProjectPreferences(api, token, w.web.id), "Web's own preferences").toEqual(projectPreferences);
  await quiet();

  // Other's page: Shared again, beside Other's own work item.
  await moveWithinApp(page, w.path(other.id));
  await expect.poll(() => shownItems(page)).toEqual(["Elsewhere", "Shared"]);
  await quiet();

  // Old's page, archived: its own work item, which offers no change; no new work item.
  await moveWithinApp(page, w.path(old.id));
  await expect.poll(() => shownItems(page)).toEqual(["Shipped"]);
  expect(await rowEntries(page, "Shipped")).toEqual(["Open in new tab", "Copy link"]);
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toHaveCount(0);
  await expect(quickAddButton(page)).toHaveCount(0);
  await quiet();

  // Web's modules off: Now's page, loaded again, shows that, and reads neither its preferences nor its work items.
  await turnModules(api, w.token, w.web, false);
  await moveWithinApp(page, w.path(now.id));
  // the page has not read Web again: it shows Now's work items, read before Web's modules were turned off
  await expect.poll(() => shownItems(page)).toEqual(["Build", "Plan", "Shared"]);
  await quiet();
  await loadedAnew(browser, baseURL ?? "", api, w.mem, w.path(now.id), async (again, seen) => {
    await expect(again.getByText("Modules are not enabled for the project.", { exact: true })).toBeVisible();
    await expect(workItemLayoutButton(again, "List")).toHaveCount(0);
    expect(
      seen.apiRequests.filter(
        (request) => request.endsWith(`/projects/${w.web.id}/issues`) || request.includes("/me/modules/")
      )
    ).toEqual([]);
  });
});

test("G15 (page): the module Now's 51 work items, two with real attachments (1 and 3), page by attachments both ways in the list, each once, as nerve pages them; the table shows each count and sorts by it, and after a sign-in anew shows the counts an added and a deleted attachment leave; a state changed across the list's groups and an archiving change the counts as nerve answers", async ({
  api,
  context,
  signedInPage,
}, testInfo) => {
  test.setTimeout(90_000);
  const w = await moduleListWorld(api, testInfo);
  const [now, other] = [await w.module("Now"), await w.module("Other")];
  const todo = w.states.Todo ?? "";
  const made = await Array.from({ length: 51 }, (_, n) => `W${String(n + 1).padStart(2, "0")}`).reduce(
    async (before, name) => [...(await before), await w.item(name, { module_ids: [now.id], state_id: todo })],
    Promise.resolve([] as Issue[])
  );
  await w.item("Loose", { state_id: todo });
  await w.item("Elsewhere", { module_ids: [other.id], state_id: todo });
  const [one, three] = [made[6]!, made[22]!];
  const attach = (issue: Issue, name: string) =>
    storedFile(
      api,
      w.token,
      "ISSUE_ATTACHMENT",
      { kind: "issue", id: issue.id },
      Buffer.from(name),
      "text/plain",
      name
    );
  await attach(one, "one.txt");
  const threes = [await attach(three, "a.txt"), await attach(three, "b.txt"), await attach(three, "c.txt")];
  const zeros = made.filter((issue) => issue !== one && issue !== three);
  const down = [three.id, one.id, ...idsDown(...zeros)];
  const up = [...ids(...zeros), one.id, three.id];
  // the API pages the module's list by attachments both ways, each work item once, the last page without a cursor
  const token = w.mem.tokens.access_token;
  await expectPages(
    (cursor) =>
      listIssues(api, token, w.web.id, { module_id: [now.id], order_by: "-attachment_count", limit: 50, cursor }),
    down,
    50
  );
  await expectPages(
    (cursor) =>
      listIssues(api, token, w.web.id, { module_id: [now.id], order_by: "attachment_count", limit: 50, cursor }),
    up,
    50
  );
  // Now's page as nerve keeps it for mem
  const preferences = async () => {
    const read = await modulePreferencesRead(api, token, now.id);
    if (!("preferences" in read)) throw new Error(`Now's preferences: ${JSON.stringify(read)}`);
    return read.preferences;
  };
  const { display_filters, display_properties } = await preferences();
  await changeModulePreferences(api, token, now.id, {
    display_filters: {
      ...display_filters,
      layout: "list",
      group_by: "state_id",
      order_by: "-attachment_count",
      show_empty_groups: false,
    },
    display_properties: { ...display_properties, attachment_count: true },
  });

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(now.id));
  const queries = listQueries(page, w.web.id);
  const quiet = quietEachTime(page, watch);
  const group = listGroup(page, "Todo");
  // Each way: the first 50, then the rest from the group's cursor, each once; no more to load.
  const pages = async (want: string[]) => {
    await expectListOrder(group, want.slice(0, 50));
    await expect(group.locator('[class~="group/list-header"]')).toHaveText("Todo51");
    await scrollToLastListRow(group);
    await group.getByRole("button", { name: "Load more" }).click();
    await expectListOrder(group, want);
    await expect(group.getByRole("button", { name: "Load more" })).toHaveCount(0);
  };
  // the rows of the two first, with their counts
  await expect(listRow(page, three.id)).toContainText("Attachments3");
  await expect(listRow(page, one.id)).toContainText("Attachments1");
  await pages(down);
  expect(Object.fromEntries(queries.find((query) => query.has("cursor"))!)).toMatchObject({
    module_id: now.id,
    group_by: "state_id",
    group_key: todo,
    order_by: "-attachment_count",
    limit: "50",
  });
  await quiet();

  // The table sorted by its header, the fewest attachments first: the page's own order, which nerve keeps, and by
  // which the list then pages the other way. Sorted back, the most first, the table shows each count.
  await workItemLayoutButton(page, "Table").click();
  const table = page.getByRole("table");
  const heading = table.getByRole("columnheader", { name: "Attachments", exact: true });
  const sortBy = async (label: string, direction: string, first: string[]) => {
    await heading.getByRole("button").click();
    await page.getByRole("menuitem", { name: label, exact: true }).click();
    await expect(heading).toHaveAttribute("aria-sort", direction);
    await expect.poll(async () => (await tableOrder(page)).slice(0, 2)).toEqual(first);
  };
  await sortBy("Least Most", "ascending", up.slice(0, 2));
  await expect.poll(async () => (await preferences()).display_filters.order_by).toBe("attachment_count");
  await workItemLayoutButton(page, "List").click();
  await pages(up);
  await quiet();
  await workItemLayoutButton(page, "Table").click();
  await sortBy("Most Least", "descending", down.slice(0, 2));
  const cell = (issue: Issue) => table.locator("tr").filter({ has: page.locator(`td[id="issue-${issue.id}"]`) });
  await expect(cell(three)).toContainText("3 attachments");
  await expect(cell(one)).toContainText("1 attachment");
  // the first without attachments, in the order shown
  await expect(cell(zeros.find((issue) => issue.id === down[2])!)).toContainText("0 attachments");
  await quiet();

  // One more attachment of the one, one fewer of the three; mem signs in anew in another tab, and the page reads the
  // counts nerve now gives.
  const accessToken = followAccessToken(page);
  await attach(one, "two.txt");
  await deleteAttachment(api, w.token, threes[0]!.id);
  const before = accessToken();
  await (await anotherTabSignsIn(context, api, w.mem.email)).close();
  await expect(cell(one)).toContainText("2 attachments");
  await expect(cell(three)).toContainText("2 attachments");
  expect(accessToken(), "the page reads as the new sign-in").not.toBe(before);
  await quiet();

  // The list again: the first row moved from Todo to Done, then archived; the groups and the page count as nerve does.
  await workItemLayoutButton(page, "List").click();
  const first = await group
    .locator('[id^="issue_"]')
    .first()
    .evaluate((row) => row.id.split("_")[1]);
  const moved = made.find((issue) => issue.id === first)!;
  await listRow(page, moved.id).getByRole("button", { name: "State: Todo", exact: true }).click();
  await page.getByRole("option", { name: "Done", exact: true }).click();
  await expect(group.locator('[class~="group/list-header"]')).toHaveText("Todo50");
  await expect(listGroup(page, "Done").locator('[class~="group/list-header"]')).toHaveText("Done1");
  await expect(headerCount(page, "Now")).toHaveText("51");
  await scrollToLastListRow(listGroup(page, "Done"));
  await chooseFromRow(page, moved.name, "Archive");
  await modalDialog(page).getByRole("button", { name: "Archive", exact: true }).click();
  await expect(listRow(page, moved.id)).toHaveCount(0);
  await expect(listGroup(page, "Done")).toHaveCount(0);
  await expect(group.locator('[class~="group/list-header"]')).toHaveText("Todo50");
  await expect(headerCount(page, "Now")).toHaveText("50");
  expect((await listIssues(api, token, w.web.id, { module_id: [now.id], limit: 1 })).total).toBe(50);
  await quiet();
});

test("G15 (page): on the module Now's page the header's new work item and a quick one each send one request, with Now, and join it; Leave, in Now, Other and Gone, archived, removed from Now, sends Other and Gone and leaves Now alone; Keep's edit dialog stays open with Keep while the page checks it, and changes its name, never its modules; Gone's page, archived, offers neither a new work item, a removal nor an edit", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const [now, other, gone] = [await w.module("Now"), await w.module("Other"), await w.module("Gone")];
  const keep = await w.item("Keep", { module_ids: [now.id, other.id] });
  const leave = await w.item("Leave", { module_ids: [now.id, other.id, gone.id] });
  const joined = await lastWrite(db, leave.id);
  await w.archive(gone.id);

  const { page, watch } = await openListPage(signedInPage, w.mem, w.path(now.id));
  const quiet = quietEachTime(page, watch);
  const created = `/api/v0/projects/${w.web.id}/issues`;
  const bodies = await bodiesSent(page, browserName, "POST", created);
  await expect.poll(() => shownItems(page)).toEqual(["Keep", "Leave"]);

  // The header's new work item: the dialog keeps to Web, and its one request names Now.
  const from = watch.apiRequests.length;
  await page.getByRole("button", { name: "Add work item", exact: true }).click();
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  await expect(form.getByRole("button", { name: "Project: Web", exact: true })).toBeDisabled();
  await form.getByRole("textbox", { name: "Title", exact: true }).fill("Made in the header");
  await form.getByRole("button", { name: "Save", exact: true }).click();
  await expect(form).toHaveCount(0);
  // A quick one at the list's end, by its name and Enter.
  await quickAddButton(page).click();
  await page.getByRole("textbox", { name: "Work item title", exact: true }).fill("Made quickly");
  await page.keyboard.press("Enter");
  await expect.poll(() => shownItems(page)).toEqual(["Keep", "Leave", "Made in the header", "Made quickly"]);
  expect(bodies.map((body) => (body as { module_ids?: string[] }).module_ids)).toEqual([[now.id], [now.id]]);
  expect(
    watch.apiRequests.slice(from).filter((request) => !request.startsWith("GET ")),
    "the writes of the two"
  ).toEqual([`POST ${created}`, `POST ${created}`]);

  // Leave removed from Now: one change to the modules it is in but Now, Gone's kept though it is archived; its row
  // leaves and the page says so; in the database Leave's row of Now alone is removed, by mem, at that change.
  const changes = await bodiesSent(page, browserName, "PATCH", `/api/v0/issues/${leave.id}`);
  await chooseFromRow(page, "Leave", "Remove from module");
  await expect(toast(page, "Work item removed from the module successfully.")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual(["Keep", "Made in the header", "Made quickly"]);
  expect(
    (await memberships(db, w.web.id)).filter((row) => row.kind === "module"),
    "Web's rows of modules: the two new ones joined Now, Leave's of Now removed, its others kept"
  ).toEqual([
    { kind: "module", target: "Gone", archived: true, item: "Leave", removed: false },
    { kind: "module", target: "Now", archived: false, item: "Keep", removed: false },
    { kind: "module", target: "Now", archived: false, item: "Leave", removed: true },
    { kind: "module", target: "Now", archived: false, item: "Made in the header", removed: false },
    { kind: "module", target: "Now", archived: false, item: "Made quickly", removed: false },
    { kind: "module", target: "Other", archived: false, item: "Keep", removed: false },
    { kind: "module", target: "Other", archived: false, item: "Leave", removed: false },
  ]);
  const removal = await lastWrite(db, leave.id);
  expect(removal.by, "who took Leave out of Now").toBe(w.mem.email);
  await expectModuleRows(db, leave.id, [
    { target: "Gone", added: joined.at, removed: null, adder: w.admin.email, by: w.admin.email },
    { target: "Now", added: joined.at, removed: removal.at, adder: w.admin.email, by: w.mem.email },
    { target: "Other", added: joined.at, removed: null, adder: w.admin.email, by: w.admin.email },
  ]);
  expect(changes).toEqual([{ module_ids: ascendingIds(other, gone) }]);
  await quiet();

  // Keep's edit dialog: it reads Keep, its sub-issues' read checks Keep and the list reads its first page again, the
  // row off the page meanwhile; the dialog stays open with Keep, and its one change renames Keep, with no modules:
  // Keep stays in Now and Other.
  const edits = await bodiesSent(page, browserName, "PATCH", `/api/v0/issues/${keep.id}`);
  await dialogReads(page, keep.id, w.web.id, () => chooseFromRow(page, "Keep", "Edit"));
  const title = modalDialog(page).getByRole("textbox", { name: "Title", exact: true });
  await expect(title).toHaveValue("Keep");
  await title.fill("Keep it");
  await modalDialog(page).getByRole("button", { name: "Update", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(0);
  await expect.poll(() => shownItems(page)).toEqual(["Keep it", "Made in the header", "Made quickly"]);
  expect(edits, "the edit's one change").toHaveLength(1);
  expect(edits[0]).toMatchObject({ name: "Keep it" });
  expect(edits[0]).not.toHaveProperty("module_ids");
  const kept = await issueRead(api, w.token, keep.id);
  expect("issue" in kept ? kept.issue.module_ids : kept, "Keep's modules").toEqual(ascendingIds(now, other));
  await quiet();

  // Gone's page, archived: Leave, kept in it, offers only to be opened or linked; no new work item, in the header or
  // the list.
  await moveWithinApp(page, w.path(gone.id));
  await expect.poll(() => shownItems(page)).toEqual(["Leave"]);
  expect(await rowEntries(page, "Leave")).toEqual(["Open in new tab", "Copy link"]);
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toHaveCount(0);
  await expect(quickAddButton(page)).toHaveCount(0);
  await quiet();
});
