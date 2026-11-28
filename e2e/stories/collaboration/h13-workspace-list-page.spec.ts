import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { recordApi, toast } from "../../fixtures/cycle-pages";
import { focusInDialog, modalDialog } from "../../fixtures/dialogs";
import { addCondition, displayPanel } from "../../fixtures/issue-filters-page";
import { createIssue, issueRead } from "../../fixtures/issues";
import { APP, WORKSPACE, valued } from "../../fixtures/mounts";
import { person } from "../../fixtures/notification-pages";
import { shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { pageMain, rowButton } from "../../fixtures/view-pages";
import {
  TABLE_PROPERTIES,
  catalogRead,
  columnHeader,
  defaultQuery,
  noteFocus,
  offered,
  orderBy,
  rowLabels,
  settled,
  switcher,
  workspaceListReads,
  workspaceViewPath,
  workspaceViewWorld,
  workspaceViewsPath,
  withoutRowProjects,
} from "../../fixtures/workspace-view-pages";

// H13's page version, the workspace's list of work items across its projects (M7 design 2, 3.7; M7/P6b spec 2.2–2.5,
// 2.7, 2.8, 2.15; R1–R5, R7; re-97 I1, I2, m1, m7): its four default views, each the workspace's list of the projects
// the caller is an active member of, unarchived, under its own term, flat and in the spreadsheet; each row changed in
// its own project as the caller's role there allows. Their display is the browser's, by account, workspace and view;
// their filters last for one entry. The labels they filter by and show are the workspace's label catalog's, read once.
// The total that leaves out the projects the caller is no member of and the archived ones is the API story's
// (h13-workspace-list.spec.ts); a page checks its rows. A saved view's page is h13-workspace-view-page.spec.ts's.

test("H13 (page): bob's All work items, opened anew, reads the app's start, the workspace's, the label catalog once and the workspace's list once, flat, the newest first, a hundred a page, and lists Web's and Ops' work items, not Lab's, of which he is no member, nor archived Shelf's, each row's labels named; its label filter offers the catalog's Bug and Infra; his label and project conditions read the list under them, send nothing and are gone once he has left and entered again; Assigned, Created and Subscribed, each entered once from the views list, read the list under their own term; a condition that leaves nothing offers Clear all filters", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  const issues = `/api/v0/workspaces/${w.slug}/issues`;

  // Opened anew: the app's start, the workspace's, the catalog once and the list once, under the default display.
  const page = await signedInPage(w.bob.tokens);
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, "all-issues"));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
  await settled(page, opened);
  const [listRead, ...more] = workspaceListReads(opened().requests, w.slug);
  expect(more).toEqual([]);
  expect(Object.fromEntries(listRead ?? [])).toEqual(defaultQuery());
  const others = withoutRowProjects(opened().requests, [w.web.id, w.ops.id]);
  expect(others.filter((request) => !request.startsWith(`GET ${issues}?`))).toEqual(
    [
      ...[...APP, ...WORKSPACE].map((request) => valued(request, { [w.slug]: "{slug}" })),
      catalogRead(w.slug),
    ].toSorted()
  );
  expect(await rowLabels(page)).toEqual({
    Crash: "Bug",
    Deploy: "Infra",
    Note: "Select labels",
    Typo: "Select labels",
  });

  // Label: the catalog's, Bug and Infra, not Lab's; Bug reads the list under it. Project: Web and Ops; Web besides.
  await addCondition(page, "Label");
  expect((await offered(page)).toSorted()).toEqual(["Bug", "Infra", "None"]);
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Bug", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await addCondition(page, "Projects");
  expect((await offered(page)).toSorted()).toEqual(["Ops", "Web"]);
  const read = await answerTo(page, "GET", issues, () =>
    page.getByRole("option", { name: "Web", exact: true }).click()
  );
  await page.keyboard.press("Escape");
  expect(Object.fromEntries(new URL(read.url()).searchParams)).toEqual({
    ...defaultQuery(),
    project_id: w.web.id,
    label_id: w.bug.id,
  });
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, opened);

  // Left for the views list and entered again: the conditions gone, the list read under none.
  await pageMain(page).getByRole("link", { name: "Views", exact: true }).click();
  await expect(page).toHaveURL(workspaceViewsPath(w.slug));
  const again = recordApi(page);
  await answerTo(page, "GET", issues, () => page.getByRole("link", { name: "All work items", exact: true }).click());
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
  await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);
  await settled(page, again);
  expect(workspaceListReads(again().requests, w.slug).map((query) => Object.fromEntries(query))).toEqual([
    defaultQuery(),
  ]);

  // Each other default view, entered once from the views list: its own term, its rows.
  const views = [
    ["Assigned", { assignee_id: "me" }, ["Crash"]],
    ["Created", { created_by_id: "me" }, ["Typo"]],
    ["Subscribed", { subscriber_id: "me" }, ["Crash", "Deploy", "Typo"]],
  ] as const;
  await views.reduce(async (before, [name, own, rows]) => {
    await before;
    await pageMain(page).getByRole("link", { name: "Views", exact: true }).click();
    await expect(page).toHaveURL(workspaceViewsPath(w.slug));
    const entered = recordApi(page);
    await page.getByRole("link", { name, exact: true }).click();
    await expect.poll(() => shownItems(page)).toEqual(rows);
    await settled(page, entered);
    expect(workspaceListReads(entered().requests, w.slug).map((query) => Object.fromEntries(query))).toEqual([
      defaultQuery(own),
    ]);
  }, Promise.resolve());

  // On Subscribed, High: nothing, and Clear all filters, which gives the three back.
  await addCondition(page, "Priority");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "High", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect(page.getByText("No matching results.", { exact: true })).toBeVisible();
  await answerTo(page, "GET", issues, () =>
    page.getByRole("button", { name: "Clear all filters", exact: true }).click()
  );
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Typo"]);

  expect(watch.apiRequests.filter((request) => !request.startsWith("GET "))).toEqual(["POST /api/v0/auth/refresh"]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H13 (page): bob's display of All work items is the browser's: Display offers no Cycle or Modules; Labels hidden, and the order by priority by its column's header, which reads the list again, send nothing and stay once he has left and entered again; Assigned, by the switcher, has its own display, and All work items, straight back by it, lists under his; his own workspace Beta, of no project, lists in the default display and offers its first project", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  const beta = slugFor(testInfo, "beta");
  await createWorkspace(api, w.bob.token, { name: "Beta", slug: beta });
  const issues = `/api/v0/workspaces/${w.slug}/issues`;
  const byPriority = defaultQuery({}, "-priority");

  const page = await signedInPage(w.bob.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, "all-issues"));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);

  // Display: the table's properties, no Cycle or Modules; Labels hidden, its column gone.
  const panel = await displayPanel(page);
  const properties = panel.locator("button[aria-pressed]");
  expect(await properties.allTextContents()).toEqual([...TABLE_PROPERTIES, "Show sub-work items"]);
  await panel.getByRole("button", { name: "Labels", exact: true }).click();
  await expect(panel.getByRole("button", { name: "Labels", exact: true })).toHaveAttribute("aria-pressed", "false");
  await page.keyboard.press("Escape");
  await expect(columnHeader(page, "Labels")).toHaveCount(0);
  // Priority's header, Urgent first: the list read under it.
  const ordered = await answerTo(page, "GET", issues, () => orderBy(page, "Priority", "Urgent None"));
  expect(Object.fromEntries(new URL(ordered.url()).searchParams)).toEqual(byPriority);
  await expect(columnHeader(page, "Priority")).toHaveAttribute("aria-sort", "descending");
  await settled(page, recorded);

  // Left for the views list and entered again: Labels hidden, the list read by priority.
  await pageMain(page).getByRole("link", { name: "Views", exact: true }).click();
  await expect(page).toHaveURL(workspaceViewsPath(w.slug));
  const entered = await answerTo(page, "GET", issues, () =>
    page.getByRole("link", { name: "All work items", exact: true }).click()
  );
  expect(Object.fromEntries(new URL(entered.url()).searchParams)).toEqual(byPriority);
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
  await expect(columnHeader(page, "Labels")).toHaveCount(0);
  await settled(page, recorded);

  // Assigned, by the switcher: its own display; straight back to All work items by it, once Assigned's reads are
  // answered: the list, under bob's display.
  await switcher(page, "All work items").click();
  const assigned = await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Assigned" }).click());
  expect(Object.fromEntries(new URL(assigned.url()).searchParams)).toEqual(defaultQuery({ assignee_id: "me" }));
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await expect(columnHeader(page, "Labels")).toBeVisible();
  await settled(page, recorded);
  await switcher(page, "Assigned").click();
  const back = await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "All work items" }).click());
  expect(Object.fromEntries(new URL(back.url()).searchParams)).toEqual(byPriority);
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
  await expect(columnHeader(page, "Labels")).toHaveCount(0);
  await expect(columnHeader(page, "Priority")).toHaveAttribute("aria-sort", "descending");
  await settled(page, recorded);

  // Beta, bob's, of no project: its list read in the default display, Labels shown; its first project offered.
  const betaRead = await answerTo(page, "GET", `/api/v0/workspaces/${beta}/issues`, () =>
    moveWithinApp(page, workspaceViewPath(beta, "all-issues"))
  );
  expect(Object.fromEntries(new URL(betaRead.url()).searchParams)).toEqual(defaultQuery());
  await expect(page.getByText("No project", { exact: true })).toBeVisible();
  const betaPanel = await displayPanel(page);
  await expect(betaPanel.getByRole("button", { name: "Labels", exact: true })).toHaveAttribute("aria-pressed", "true");
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Start your first project", exact: true }).click();
  await expect(modalDialog(page)).toHaveAccessibleName("Create project");
  await expect.poll(() => focusInDialog(page), "the focus in the dialog").toBe(true);
  await noteFocus(page, testInfo, "Create project's dialog");
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await settled(page, recorded);

  expect(watch.apiRequests.filter((request) => !request.startsWith("GET "))).toEqual(["POST /api/v0/auth/refresh"]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H13 (page): on bob's Assigned, Crash, his in Web, given to Alice instead stays listed while nerve has not answered, and leaves once it has: nothing is left, and the page offers a new work item", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  const change = `/api/v0/issues/${w.crash.id}`;

  const page = await signedInPage(w.bob.tokens);
  const watch = await watchPage(page);
  await page.goto(workspaceViewPath(w.slug, "assigned"));
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);

  // Alice besides: still bob's, still listed. Bob taken off, the answer held: listed until nerve answers.
  const crash = page.getByRole("table").getByRole("row", { name: /Options: Crash/ });
  await crash.getByRole("button", { name: "Bob Bob", exact: true }).click();
  expect(
    (await answerTo(page, "PATCH", change, () => page.getByRole("option", { name: "Alice" }).click())).status()
  ).toBe(200);
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  const off = await sentHeld(page, "PATCH", change, () => page.getByRole("option", { name: /^Bob/ }).click());
  expect(off.body).toEqual({ assignee_ids: [w.alice.id] });
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  expect((await off.release()).status()).toBe(200);
  await expect.poll(() => shownItems(page)).toEqual([]);
  await expect(page.getByText("No Views yet", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
  expect(await issueRead(api, w.alice.token, w.crash.id)).toMatchObject({
    status: 200,
    issue: { assignee_ids: [w.alice.id] },
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H13 (page): gus, Acme's guest, reads All work items, the catalog once, and lists Note, his own in Web, whose guests see their own alone, and Deploy, Ops', whose guests see every one, each row read-only; he is offered neither Add view nor Save view, and his switcher reads no views; Created's list read failing (Chromium), the page says so and its Retry reads it again", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  const issues = `/api/v0/workspaces/${w.slug}/issues`;

  const page = await signedInPage(w.gus.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, "all-issues"));
  await expect.poll(() => shownItems(page)).toEqual(["Deploy", "Note"]);
  await settled(page, recorded);
  expect(recorded().requests.filter((request) => request === catalogRead(w.slug))).toHaveLength(1);
  expect(workspaceListReads(recorded().requests, w.slug).map((query) => Object.fromEntries(query))).toEqual([
    defaultQuery(),
  ]);
  // Each row read-only: its state and priority stopped.
  const properties = page.getByRole("table").getByRole("button", { name: /^(State|Priority): / });
  await expect(properties).toHaveCount(4);
  await Promise.all((await properties.all()).map((button) => expect(button).toBeDisabled()));
  // No Add view; a condition offers no Save view.
  await expect(pageMain(page).getByRole("button", { name: "Add view", exact: true })).toHaveCount(0);
  await addCondition(page, "Priority");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Urgent", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect(page.getByText("No matching results.", { exact: true })).toBeVisible();
  await expect(rowButton(page, "Save view")).toHaveCount(0);
  // The switcher: the four default views, no read of the workspace's own.
  const opened = recordApi(page);
  await switcher(page, "All work items").click();
  expect(await offered(page)).toEqual(["All work items", "Assigned", "Created", "Subscribed"]);
  await page.keyboard.press("Escape");
  await settled(page, opened);
  expect(opened().requests).toEqual([]);

  // Created, its list read refused once by a route (Chromium alone, the page's code loaded): said, and Retry.
  const failures: string[] = [];
  const errors: string[] = [];
  if (browserName === "chromium") {
    let failed = false;
    await page.route(
      (url) => url.pathname === issues,
      async (route) => {
        if (failed) return route.fallback();
        failed = true;
        return route.fulfill({ status: 500, contentType: "application/problem+json", body: "{}" });
      }
    );
    await switcher(page, "All work items").click();
    await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Created" }).click());
    await expect(toast(page, "Something went wrong. Please try again.")).toBeVisible();
    await answerTo(page, "GET", issues, () => page.getByRole("button", { name: "Retry", exact: true }).click());
    await expect.poll(() => shownItems(page)).toEqual(["Note"]);
    await expect(page.getByRole("button", { name: "Retry", exact: true })).toHaveCount(0);
    failures.push(`500 GET ${issues}`);
    errors.push("Failed to load resource: the server responded with a status of 500 (Internal Server Error)");
  }

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([failures, [], []]);
  await expectQuietConsole(page, watch, { errors });
});

test("H13 (page): bob's All work items lists a hundred of his hundred and one work items across Web and Ops, and the last once the table is scrolled to its Load more, read by nerve's cursor under the same query", async ({
  api,
  signedInPage,
}, testInfo) => {
  const bob = await person(api, testInfo, "bob", "Bob");
  const slug = slugFor(testInfo);
  await createWorkspace(api, bob.token, { name: "Acme", slug });
  const [web, ops] = await Promise.all([
    createProject(api, bob.token, slug, { name: "Web", identifier: "WEB" }),
    createProject(api, bob.token, slug, { name: "Ops", identifier: "OPS" }),
  ]);
  if (!web || !ops) throw new Error("missing projects");
  await Promise.all(
    Array.from({ length: 101 }, (_, index) =>
      createIssue(api, bob.token, index % 2 ? ops.id : web.id, { name: `Item ${String(index).padStart(3, "0")}` })
    )
  );
  const issues = `/api/v0/workspaces/${slug}/issues`;

  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(slug, "all-issues"));
  const rows = page.getByRole("table").locator("tbody > tr");
  await expect(rows).toHaveCount(100);
  await settled(page, recorded);
  // Scrolled to the table's Load more, which a click would also take (WebKit loads the rest on the scroll alone, and
  // the button goes before a click lands): the rest, read once.
  const more = page.getByRole("button", { name: "Load more", exact: true });
  await expect(more).toBeEnabled();
  const next = await answerTo(page, "GET", issues, () => more.scrollIntoViewIfNeeded());
  await expect(rows).toHaveCount(101);
  await expect(more).toHaveCount(0);
  await settled(page, recorded);
  const reads = workspaceListReads(recorded().requests, slug).map((query) => Object.fromEntries(query));
  const cursor = new URL(next.url()).searchParams.get("cursor");
  expect(cursor).toBeTruthy();
  expect(reads).toEqual([defaultQuery(), { ...defaultQuery(), cursor }]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
