import { expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import {
  chooseFromMenu,
  dismissToasts,
  entriesOf,
  itemMenu,
  menuEntries,
  recordApi,
  rowMenusKept,
  seenBy,
  toast,
} from "../../fixtures/cycle-pages";
import { modalDialog, sentWithin, shownWithin } from "../../fixtures/dialogs";
import { APP, WORKSPACE, valued } from "../../fixtures/mounts";
import { shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { listedViews, pageMain } from "../../fixtures/view-pages";
import { viewCalls, viewDefaults, viewRow } from "../../fixtures/views";
import {
  recordShown,
  settled,
  switcher,
  workspaceViewContent,
  workspaceViewPath,
  workspaceViewsPath,
  workspaceViewsRead,
  workspaceViewsWorld,
} from "../../fixtures/workspace-view-pages";

// H14's page version, a workspace's views list (M7 design 3.7; M7/P6b spec 2.7, 2.10, 2.11, 2.15; R12–R15, R29, R30):
// the four default views, anyone's of the workspace, then the workspace's own views, the newest first, for its admins
// and members alone, who read them as the page opens; its guest reads none. What each role is offered there is what
// nerve lets him (spec 2.7's matrix): Add view to the admins and members; a view's Edit to its owner, its Delete to its
// owner and the workspace's admins, its link and a new tab to those who see it; a default view's link and a new tab
// to anyone. The search is the page's own, whatever its case. Add view makes a view of the workspace's own, of no
// project, in the spreadsheet (WORKSPACE_VIEW_CONTENT); Edit sends what changed alone; a deletion from the view's own
// page goes to the list, never through "does not exist". The API version is h14-workspace-views.spec.ts.

/** The four default views, by their names, as the list shows them first. */
const DEFAULTS = ["All work items", "Assigned", "Created", "Subscribed"];
/** The entries of a view's menu for its owner, and for one who may neither change nor delete it. */
const [OWN, SEEN] = [
  ["Edit", "Open in new tab", "Copy link", "Delete"],
  ["Open in new tab", "Copy link"],
];

test("H14 (page): alice, Acme's admin, opened anew, reads the app's start, the workspace's and its views once, and lists the four default views, then Acme's own, the newest first; Add view hers, Edit of her own view and Delete of each, a default view's link and a new tab; each row's right-click menu its button's, inert while closed; the search, whatever its case, keeps the rows whose names hold it and says when none does, reading nothing; bob, a member, Add view, Edit and Delete of his own, alice's link and a new tab alone; gus, the guest, the default views alone, no Add view, no views read", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewsWorld(api, testInfo);
  await w.view(w.alice, { name: "Alice's board" });
  await w.view(w.bob, { name: "Bob's bugs" });
  const all = [...DEFAULTS, "Bob's bugs", "Alice's board"];

  // Opened anew: the app's start, the workspace's and its views, once; the default views, then the newest first.
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(workspaceViewsPath(w.slug));
  await expect.poll(() => listedViews(page)).toEqual(all);
  await settled(page, opened);
  expect(opened().requests).toEqual(
    [
      ...[...APP, ...WORKSPACE].map((request) => valued(request, { [w.slug]: "{slug}" })),
      workspaceViewsRead(w.slug),
    ].toSorted()
  );
  await expect(pageMain(page).getByRole("button", { name: "Add view", exact: true })).toBeVisible();
  // Each row's menu, named by its view: Edit of her own, Delete of each; a default view's link and a new tab alone.
  expect(await menuEntries(page, "Alice's board")).toEqual(OWN);
  expect(await menuEntries(page, "Bob's bugs")).toEqual(["Open in new tab", "Copy link", "Delete"]);
  expect(await menuEntries(page, "All work items")).toEqual(SEEN);
  // Each row's right-click menu, its one other, inert while closed; a right-click opens it, its button's entries, and
  // Escape closes it, the focus back on the row's link.
  expect(await rowMenusKept(page)).toEqual({ entries: 4 * 2 + 4 + 3, focusable: 0 });
  const rightClicked = async (name: string) => {
    const link = page.getByRole("link", { name, exact: true });
    await link.click({ button: "right" });
    const entries = await entriesOf(page.getByRole("menu"));
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menu")).toHaveCount(0);
    await expect(link).toBeFocused();
    return entries;
  };
  expect(await rightClicked("Bob's bugs")).toEqual(["Open in new tab", "Copy link", "Delete"]);
  expect(await rightClicked("Assigned")).toEqual(SEEN);

  // The search, whatever its case and its spaces around: a view of Acme's own, a default view by its name; one that
  // matches none says so; emptied, every row back. It reads nothing.
  const local = recordApi(page);
  const search = pageMain(page).getByRole("textbox", { name: "Search", exact: true });
  await search.fill("BUG");
  await expect.poll(() => listedViews(page)).toEqual(["Bob's bugs"]);
  await search.fill(" assi ");
  await expect.poll(() => listedViews(page)).toEqual(["Assigned"]);
  await search.fill("sprint");
  await expect(pageMain(page).getByText("No matches found", { exact: true })).toBeVisible();
  expect(await listedViews(page)).toEqual([]);
  await search.fill("");
  await expect.poll(() => listedViews(page)).toEqual(all);
  expect(local().requests, "what the search read").toEqual([]);

  // bob, a member: Add view; Edit and Delete of his own; alice's link and a new tab alone.
  await seenBy(browser, baseURL ?? "", api, w.bob.email, workspaceViewsPath(w.slug), async (other) => {
    await expect.poll(() => listedViews(other)).toEqual(all);
    await expect(pageMain(other).getByRole("button", { name: "Add view", exact: true })).toBeVisible();
    expect(await menuEntries(other, "Bob's bugs")).toEqual(OWN);
    expect(await menuEntries(other, "Alice's board")).toEqual(SEEN);
  });
  // gus, the guest: the default views alone, their link and a new tab; no Add view; once the app's start and the
  // workspace's reads are out, none of Acme's own views among them.
  await seenBy(browser, baseURL ?? "", api, w.gus.email, workspaceViewsPath(w.slug), async (other, seen) => {
    await expect.poll(() => listedViews(other)).toEqual(DEFAULTS);
    await expect(pageMain(other).getByRole("button", { name: "Add view", exact: true })).toHaveCount(0);
    expect(await menuEntries(other, "Created")).toEqual(SEEN);
    const shell = [...APP, ...WORKSPACE].map((request) => valued(request, { [w.slug]: "{slug}" }).replace(/\?.*$/, ""));
    await expect.poll(() => shell.filter((request) => !seen.apiRequests.includes(request))).toEqual([]);
    expect(seen.apiRequests.filter((request) => request === workspaceViewsRead(w.slug))).toEqual([]);
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H14 (page): bob's Add view opens Create View, its name empty and focused; a blank name and one of 256 characters are refused under it, its focus, nothing sent; Plan, sent, closes the dialog and says so, the new view's page opening: its body the table's display and every property, no filters, a view of Acme's own, of no project, his as stored and as nerve reads it back, listed first of Acme's own", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewsWorld(api, testInfo);
  const board = await w.view(w.alice, { name: "Alice's board" });

  // A view's page first, then the list by its crumb: both pages' code loaded before any route is (WebKit's preloads).
  const page = await signedInPage(w.bob.tokens);
  const watch = await watchPage(page);
  const loaded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, board.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, loaded);
  await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/views`, () =>
    pageMain(page).getByRole("link", { name: "Views", exact: true }).click()
  );
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Alice's board"]);
  await settled(page, loaded);

  // Add view: Create View, its name empty and focused.
  await pageMain(page).getByRole("button", { name: "Add view", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create View");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toHaveValue("");
  await expect(name).toBeFocused();
  // A blank name, then one of 256 characters: refused under it, its focus, nothing sent.
  const views = `/api/v0/workspaces/${w.slug}/views`;
  const create = dialog.getByRole("button", { name: "Create View", exact: true });
  const refused = async (typed: string, message: string) => {
    await name.fill(typed);
    expect(await sentWithin(page, "POST", views, () => create.click())).toBe(false);
    await expect(dialog.getByText(message, { exact: true })).toBeVisible();
    await expect(name).toHaveAttribute("aria-invalid", "true");
    await expect(name).toBeFocused();
  };
  await refused("   ", "Title is required");
  await refused("x".repeat(256), "Title should be less than 255 characters");

  // Plan: sent, the dialog closes and says so, its page opens. Its body: who it is, the table's display and every
  // property, no filters (WORKSPACE_VIEW_CONTENT); the view Acme's own, of no project, bob's.
  await name.fill("Plan");
  await dialog.getByRole("textbox", { name: "Description", exact: true }).fill("Next quarter's");
  const sent = await sentTo(page, "POST", views, () => create.click());
  expect(sent.answer.status()).toBe(201);
  expect(sent.body).toEqual({ name: "Plan", description: "Next quarter's", logo_props: {}, ...workspaceViewContent });
  const made = (await sent.answer.json()) as typeof board;
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "View created successfully.")).toBeVisible();
  await expect(page).toHaveURL(workspaceViewPath(w.slug, made.id));
  await expect(switcher(page, "Plan")).toBeVisible();
  expect(made).toEqual({
    ...viewDefaults,
    ...workspaceViewContent,
    id: made.id,
    workspace_id: w.acme.id,
    project_id: null,
    name: "Plan",
    description: "Next quarter's",
    owned_by_id: w.bob.id,
    created_at: made.created_at,
    updated_at: made.created_at,
  });
  expect(await viewRow(db, made.id)).toEqual({
    project_id: null,
    owner: w.bob.id,
    made_by: w.bob.id,
    written_by: w.bob.id,
    deleted: false,
    stamped: true,
  });
  expect(await viewCalls(api).read(w.bob.token, made.id)).toEqual({ status: 200, view: made });
  // Back on the list by the crumb: Plan first of Acme's own.
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, loaded);
  await pageMain(page).getByRole("link", { name: "Views", exact: true }).click();
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Plan", "Alice's board"]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H14 (page): alice's deletion of Old from its own page's menu goes to Acme's views, saying so and never that Old does not exist; there, her Edit of Triage sends its new name alone; her Delete of bob's Bugs, by its row's right-click menu, says so and takes the row away, Bugs deleted by her as stored; Triage's link and Assigned's copied say so, and the browser refusing the clipboard, Triage's says it failed", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewsWorld(api, testInfo);
  const old = await w.view(w.alice, { name: "Old" });
  const bugs = await w.view(w.bob, { name: "Bugs" });
  const triage = await w.view(w.alice, { name: "Triage" });

  // Old's page; its header's menu deletes it: the dialog closes, says so, and the list shows, the page never saying
  // that Old does not exist on its way there.
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, old.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await settled(page, recorded);
  await itemMenu(page, "Old").click();
  expect(await entriesOf(page.getByRole("menu"))).toEqual(OWN);
  await page.getByRole("menu").getByRole("menuitem", { name: "Delete", exact: true }).click();
  const confirm = page.getByRole("dialog", { name: "Are you sure you want to delete this view?", exact: true });
  const notFound = await recordShown(page, "View does not exist");
  const gone = await answerTo(page, "DELETE", `/api/v0/views/${old.id}`, () =>
    confirm.getByRole("button", { name: "Delete", exact: true }).click()
  );
  expect(gone.status()).toBe(204);
  await expect(toast(page, "View deleted successfully.")).toBeVisible();
  await expect(page).toHaveURL(workspaceViewsPath(w.slug));
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Triage", "Bugs"]);
  expect(await notFound(), "Old's page said it does not exist").toBe(false);
  expect(await viewRow(db, old.id)).toMatchObject({ project_id: null, deleted: true, written_by: w.alice.id });
  await settled(page, recorded);
  await dismissToasts(page);

  // Edit of Triage: Update View, its name the view's; the new name alone is sent.
  await chooseFromMenu(page, "Triage", "Edit");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Update View");
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(name).toHaveValue("Triage");
  await name.fill("Triage board");
  const edited = await sentTo(page, "PATCH", `/api/v0/views/${triage.id}`, () =>
    dialog.getByRole("button", { name: "Update View", exact: true }).click()
  );
  expect([edited.body, edited.answer.status()]).toEqual([{ name: "Triage board" }, 200]);
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Triage board", "Bugs"]);

  // bob's Bugs, by its row's right-click menu: the dialog closes, says so, and the row goes, Bugs deleted by her.
  await page.getByRole("link", { name: "Bugs", exact: true }).click({ button: "right" });
  await page.getByRole("menu").getByRole("menuitem", { name: "Delete", exact: true }).click();
  const deleted = await answerTo(page, "DELETE", `/api/v0/views/${bugs.id}`, () =>
    confirm.getByRole("button", { name: "Delete", exact: true }).click()
  );
  expect(deleted.status()).toBe(204);
  await expect(confirm).toHaveCount(0);
  await expect(toast(page, "View deleted successfully.")).toBeVisible();
  await expect.poll(() => listedViews(page)).toEqual([...DEFAULTS, "Triage board"]);
  expect(await viewRow(db, bugs.id)).toEqual({
    project_id: null,
    owner: w.bob.id,
    made_by: w.bob.id,
    written_by: w.alice.id,
    deleted: true,
    stamped: true,
  });

  // Triage's link and Assigned's, copied: said so (Chromium lets the page write the clipboard once allowed, Firefox
  // and WebKit a click); refused by the browser: said it failed, never copied.
  const copied = async (view: string, path: string) => {
    await chooseFromMenu(page, view, "Copy link");
    await expect(toast(page, "Link copied to clipboard")).toBeVisible();
    if (browserName === "chromium") expect(await page.evaluate(() => navigator.clipboard.readText())).toContain(path);
    await dismissToasts(page);
  };
  if (browserName === "chromium") await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await copied("Triage board", workspaceViewPath(w.slug, triage.id));
  await copied("Assigned", workspaceViewPath(w.slug, "assigned"));
  await refuseClipboardWrites(page);
  await chooseFromMenu(page, "Triage board", "Copy link");
  await expect(toast(page, "Something went wrong. Please try again.")).toBeVisible();
  expect(await shownWithin(toast(page, "Link copied to clipboard"))).toBe(false);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
