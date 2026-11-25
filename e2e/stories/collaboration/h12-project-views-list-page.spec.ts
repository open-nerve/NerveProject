import { expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import {
  chooseFromMenu,
  dismissToasts,
  entriesOf,
  menuEntries,
  recordApi,
  rowMenusKept,
  seenBy,
  toast,
} from "../../fixtures/cycle-pages";
import {
  closedByBackdropClick,
  closedByEscape,
  closedWithin,
  focusInDialog,
  modalDialog,
  sentWithin,
  shownWithin,
} from "../../fixtures/dialogs";
import { filterOption } from "../../fixtures/intake-pages";
import { APP, PROJECT, PROJECT_MEMBER, PROJECT_VIEWS, WORKSPACE, valued } from "../../fixtures/mounts";
import { moveWithinApp, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  listedViews,
  pageMain,
  viewPath,
  viewReads,
  viewWorld,
  viewsCrumb,
  viewsPath,
  viewsRead,
} from "../../fixtures/view-pages";
import { viewCalls, viewDefaults, viewOf, viewRow } from "../../fixtures/views";

// H12's page version, a project's views list (M7 design 3.7; P6a spec 2.2–2.4, 2.11; R4, R5, R11, R17, R28, R30): what
// each role is offered there, as nerve lets him (the matrix of spec 2.2): a new view and the empty list's New to the
// project's admins and members; a view's Edit to its owner, its Delete to its owner and the project's admins, its link
// and a new tab to anyone; a guest who sees his own work items alone lists his own views alone. The list's search, its
// owner and date filters and its order are the page's own, which read nothing. "Add view" opens the view's dialog,
// which holds the view's icon, name and description alone and sends the contract's default display; Edit sends what
// changed alone; Delete deletes. A project whose views are off shows its disabled page and reads none. The API version
// is h12-project-views.spec.ts; the view's page is h12-project-view-page.spec.ts's.

test("H12 (page): alice, Web's admin, opened anew, reads the app's start, the workspace's, the project's and its views alone, and is offered Add view, Edit of her own view and Delete of each; bob, a member, Add view, Edit and Delete of his own, alice's link and a new tab alone; gus, a guest who sees his own, lists his own view alone, its link and a new tab, no Add view, and Ops' empty list's New stopped; Lab, its views off, shows its disabled page, its button alice's, and reads no views", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  // gus's view, made while he is a member, then alice's and bob's; gus then Web's guest, and Ops' too.
  await w.view(w.gus, { name: "Gus's own" });
  await w.view(w.alice, { name: "Alice's board" });
  await w.view(w.bob, { name: "Bob's bugs" });
  await w.guest();
  const ops = await w.project("Ops", "OPS", true);
  await w.guest(ops);
  const lab = await w.project("Lab", "LAB", false);
  const all = ["Bob's bugs", "Alice's board", "Gus's own"];
  const [own, others] = [
    ["Edit", "Open in new tab", "Copy link", "Delete"],
    ["Open in new tab", "Copy link"],
  ];

  // alice's list, opened anew: the app's start, the workspace's, Web's, its resources and its views; the last changed
  // first; Add view; Edit of her own view alone, Delete of each.
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(viewsPath(w.slug, w.web.id));
  await expect.poll(() => listedViews(page)).toEqual(all);
  await expect.poll(opened).toEqual({
    pending: 0,
    requests: [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...PROJECT_VIEWS]
      .map((request) => valued(request, { [w.slug]: "{slug}", [w.web.id]: "{project}" }))
      .toSorted(),
  });
  await expect(viewsCrumb(page, w.slug, w.web.id)).toBeVisible();
  await expect(pageMain(page).getByRole("button", { name: "Add view", exact: true })).toBeVisible();
  expect(await menuEntries(page, "Alice's board")).toEqual(own);
  expect(await menuEntries(page, "Bob's bugs")).toEqual(["Open in new tab", "Copy link", "Delete"]);
  expect(await menuEntries(page, "Gus's own")).toEqual(["Open in new tab", "Copy link", "Delete"]);
  // Each row's right-click menu is the row's one other menu, inert while closed; a right-click opens it, its entries
  // its button's, and Escape closes it, the focus back on the row's link.
  expect(await rowMenusKept(page)).toEqual({ entries: 10, focusable: 0 });
  const bobsLink = page.getByRole("link", { name: "Bob's bugs", exact: true });
  await bobsLink.click({ button: "right" });
  expect(await entriesOf(page.getByRole("menu"))).toEqual(["Open in new tab", "Copy link", "Delete"]);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
  await expect(bobsLink).toBeFocused();

  // Ops' list, empty: its New offered her; Lab's, its views off: the disabled page, its button hers, no views read.
  await expect.poll(() => opened().pending).toBe(0);
  const moved = recordApi(page);
  await moveWithinApp(page, viewsPath(w.slug, ops.id));
  await expect(page.getByRole("button", { name: "Create view", exact: true })).toBeEnabled();
  await expect.poll(() => moved().pending).toBe(0);
  await moveWithinApp(page, viewsPath(w.slug, lab.id));
  await expect(page.getByText("Views are not enabled for the project.", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Manage features", exact: true })).toBeEnabled();
  await expect.poll(() => moved().pending).toBe(0);
  expect(viewReads(moved().requests)).toEqual([viewsRead(ops.id)]);

  // bob: Add view; Edit and Delete of his own; alice's link and a new tab alone. Lab's button is not his.
  await seenBy(browser, baseURL ?? "", api, w.bob.email, viewsPath(w.slug, w.web.id), async (other) => {
    await expect.poll(() => listedViews(other)).toEqual(all);
    await expect(pageMain(other).getByRole("button", { name: "Add view", exact: true })).toBeVisible();
    expect(await menuEntries(other, "Bob's bugs")).toEqual(own);
    expect(await menuEntries(other, "Alice's board")).toEqual(others);
    await moveWithinApp(other, viewsPath(w.slug, lab.id));
    await expect(other.getByRole("button", { name: "Manage features", exact: true })).toBeDisabled();
  });
  // gus: his own view alone, its link and a new tab; no Add view; Ops' New stopped.
  await seenBy(browser, baseURL ?? "", api, w.gus.email, viewsPath(w.slug, w.web.id), async (other) => {
    await expect.poll(() => listedViews(other)).toEqual(["Gus's own"]);
    await expect(pageMain(other).getByRole("button", { name: "Add view", exact: true })).toHaveCount(0);
    expect(await menuEntries(other, "Gus's own")).toEqual(others);
    await moveWithinApp(other, viewsPath(w.slug, ops.id));
    await expect(other.getByRole("button", { name: "Create view", exact: true })).toBeDisabled();
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): alice's list of Web's views, the last changed first, searched by name whatever its case, ordered by name both ways and by creation, filtered by bob as its owner and by a creation a week ago, which none matches, and cleared, reads nothing; its Filters button says whether it is open, and Escape gives the focus back to it", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  // Roadmap, then Bugs (bob's), then Board; Roadmap renamed last, so the last changed.
  const roadmap = await w.view(w.alice, { name: "Roadmap" });
  await w.view(w.bob, { name: "Bugs" });
  await w.view(w.alice, { name: "Board" });
  viewOf(await viewCalls(api).update(w.alice.token, roadmap.id, { name: "Release roadmap" }), 200);

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  await page.goto(viewsPath(w.slug, w.web.id));
  const listed = () => listedViews(page);
  await expect.poll(listed).toEqual(["Release roadmap", "Board", "Bugs"]);
  const local = recordApi(page);

  // The search, whatever its case; one that matches none says so; Escape empties it.
  await pageMain(page).getByRole("button", { name: "Search", exact: true }).click();
  const search = pageMain(page).getByPlaceholder("Search", { exact: true });
  await search.fill("BUG");
  await expect.poll(listed).toEqual(["Bugs"]);
  await search.fill("sprint");
  await expect(page.getByText("No matching results.", { exact: true })).toBeVisible();
  await search.press("Escape");
  await expect.poll(listed).toEqual(["Release roadmap", "Board", "Bugs"]);

  // The order: by name, the last first, then the first first; by creation, the first first.
  const order = async (current: string, item: string) => {
    await pageMain(page).getByRole("button", { name: current, exact: true }).click();
    await page.getByRole("menu").getByRole("menuitem", { name: item, exact: true }).click();
    await expect(page.getByRole("menu")).toHaveCount(0);
  };
  await order("Updated at", "Name");
  await expect.poll(listed).toEqual(["Release roadmap", "Bugs", "Board"]);
  await order("Name", "Ascending");
  await expect.poll(listed).toEqual(["Board", "Bugs", "Release roadmap"]);
  await order("Name", "Created at");
  await expect.poll(listed).toEqual(["Release roadmap", "Bugs", "Board"]);

  // The Filters button, named so, says it is closed, then open; bob as the owner lists his own; Escape closes it and
  // gives the focus back to it.
  const filters = pageMain(page).getByRole("button", { name: "Filters", exact: true });
  await expect(filters).toHaveAttribute("aria-expanded", "false");
  await filters.click();
  await expect(filters).toHaveAttribute("aria-expanded", "true");
  const panel = page.locator(`[id="${await filters.getAttribute("aria-controls")}"]`);
  await filterOption(panel, "Created by", /Bob$/).click();
  await expect.poll(listed).toEqual(["Bugs"]);
  await page.keyboard.press("Escape");
  await expect(filters).toHaveAttribute("aria-expanded", "false");
  await expect(filters).toBeFocused();
  // Created a week ago or more: none is. Clear all lists the three again.
  await filters.click();
  await filterOption(panel, "Created date", "1 week ago").click();
  await expect(page.getByText("No matching results.", { exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(filters).toBeFocused();
  await pageMain(page).getByRole("button", { name: "Clear all", exact: true }).click();
  await expect.poll(listed).toEqual(["Release roadmap", "Bugs", "Board"]);
  expect(local().requests, "what the list's own choices read").toEqual([]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): alice's Add view opens the dialog Create View, its name empty and focused, holding the view's icon, name and description alone; Escape, the backdrop and Cancel each close it; a blank name and one of 256 characters are refused under it, its focus, nothing sent; Sprint board, its answer held, keeps the dialog, Cancel stopped and Escape taken, the focus within; let through, the dialog closes and says so, the new view's page opens, its body nerve's default display and properties and no filters, as nerve reads it back, and Web's list shows it", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const triage = await w.view(w.alice, { name: "Triage" });
  // A view's page first, then the list by its crumb: the page's code is loaded before any route is (WebKit's preloads).
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const loaded = recordApi(page);
  await page.goto(viewPath(w.slug, w.web.id, triage.id));
  await expect(pageMain(page).getByRole("button", { name: "Triage", exact: true })).toBeVisible();
  await expect.poll(() => loaded().pending).toBe(0);
  await viewsCrumb(page, w.slug, w.web.id).click();
  await expect.poll(() => listedViews(page)).toEqual(["Triage"]);
  await expect.poll(() => loaded().pending).toBe(0);

  const addView = pageMain(page).getByRole("button", { name: "Add view", exact: true });
  const dialog = modalDialog(page);
  const name = dialog.getByRole("textbox", { name: "Title", exact: true });
  const open = async () => {
    await addView.click();
    await expect(dialog).toHaveAccessibleName("Create View");
    await expect(name).toHaveValue("");
  };
  // Its controls: the icon, the name, the description, Cancel and Create View; no layout and no display.
  await open();
  await expect(name).toBeFocused();
  expect(
    await dialog
      .locator("button, input, textarea, select, [role=combobox], [role=listbox]")
      .evaluateAll((controls) =>
        controls.map((one) => one.getAttribute("aria-label") || one.getAttribute("placeholder") || one.textContent)
      )
  ).toEqual(["View icon", "Title", "Description", "Cancel", "Create View"]);
  expect(await closedByEscape(page)).toBe(true);
  await open();
  expect(await closedByBackdropClick(page)).toBe(true);
  await open();
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await closedWithin(page)).toBe(true);

  // A blank name, then one of 256 characters: refused under it, its focus, nothing sent.
  const views = `/api/v0/projects/${w.web.id}/views`;
  const create = dialog.getByRole("button", { name: "Create View", exact: true });
  await open();
  const refused = async (typed: string, message: string) => {
    await name.fill(typed);
    expect(await sentWithin(page, "POST", views, () => create.click())).toBe(false);
    await expect(dialog.getByText(message, { exact: true })).toBeVisible();
    await expect(name).toHaveAttribute("aria-invalid", "true");
    await expect(name).toBeFocused();
  };
  await refused("   ", "Title is required");
  await refused("x".repeat(256), "Title should be less than 255 characters");

  // Sprint board, its answer held: the dialog stays, Cancel stopped, Escape taken, the focus within.
  await name.fill("Sprint board");
  await dialog.getByRole("textbox", { name: "Description", exact: true }).fill("This sprint's");
  const sent = await sentHeld(page, "POST", views, () => create.click());
  await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled();
  await page.keyboard.press("Escape");
  expect(await closedWithin(page)).toBe(false);
  expect(await focusInDialog(page)).toBe(true);
  const loading = recordApi(page);
  const answer = await sent.release();
  expect(answer.status()).toBe(201);
  const made = (await answer.json()) as Awaited<ReturnType<typeof w.view>>;
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "View created successfully.")).toBeVisible();
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, made.id));
  // The body: who the view is, and nerve's default display and properties; no filters.
  expect(sent.body).toEqual({
    name: "Sprint board",
    description: "This sprint's",
    logo_props: {},
    display_filters: viewDefaults.display_filters,
    display_properties: viewDefaults.display_properties,
  });
  expect(made).toEqual({
    ...viewDefaults,
    id: made.id,
    workspace_id: made.workspace_id,
    project_id: w.web.id,
    name: "Sprint board",
    description: "This sprint's",
    owned_by_id: w.alice.id,
    created_at: made.created_at,
    updated_at: made.created_at,
  });
  expect(await viewCalls(api).read(w.alice.token, made.id)).toEqual({ status: 200, view: made });
  await expect(pageMain(page).getByRole("button", { name: "Sprint board", exact: true })).toBeVisible();
  await expect.poll(() => loading().pending).toBe(0);
  await viewsCrumb(page, w.slug, w.web.id).click();
  await expect.poll(() => listedViews(page)).toEqual(["Sprint board", "Triage"]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): alice's Edit of her view Triage opens Update View with its name, and sends its new description alone; her Delete of Old, by its row's right-click menu, keeps the dialog while nerve's answer is held, then closes, says so and takes the row away, Old deleted by her as stored; bob's Bugs' link copied says so, and the browser refusing the clipboard, the page says it failed", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const old = await w.view(w.alice, { name: "Old" });
  const bugs = await w.view(w.bob, { name: "Bugs" });
  const triage = await w.view(w.alice, { name: "Triage" });

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  await page.goto(viewsPath(w.slug, w.web.id));
  const listed = () => listedViews(page);
  await expect.poll(listed).toEqual(["Triage", "Bugs", "Old"]);
  const dialog = modalDialog(page);

  // Edit: Update View, its name the view's; the new description alone is sent.
  await chooseFromMenu(page, "Triage", "Edit");
  await expect(dialog).toHaveAccessibleName("Update View");
  await expect(dialog.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Triage");
  await dialog.getByRole("textbox", { name: "Description", exact: true }).fill("Needs a look");
  const edited = await sentTo(page, "PATCH", `/api/v0/views/${triage.id}`, () =>
    dialog.getByRole("button", { name: "Update View", exact: true }).click()
  );
  expect([edited.body, edited.answer.status()]).toEqual([{ description: "Needs a look" }, 200]);
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await expect(page.getByText("Needs a look", { exact: true })).toBeVisible();
  await dismissToasts(page);

  // Delete, by the row's right-click menu: held, the dialog stays; let through, it closes, says so, the row goes.
  await page.getByRole("link", { name: "Old", exact: true }).click({ button: "right" });
  await page.getByRole("menu").getByRole("menuitem", { name: "Delete", exact: true }).click();
  const confirm = page.getByRole("dialog", { name: "Are you sure you want to delete this view?", exact: true });
  const deleted = await sentHeld(page, "DELETE", `/api/v0/views/${old.id}`, () =>
    confirm.getByRole("button", { name: "Delete", exact: true }).click()
  );
  await page.keyboard.press("Escape");
  expect(await closedWithin(page)).toBe(false);
  expect((await deleted.release()).status()).toBe(204);
  await expect(confirm).toHaveCount(0);
  await expect(toast(page, "View deleted successfully.")).toBeVisible();
  await expect.poll(listed).toEqual(["Triage", "Bugs"]);
  expect(await viewRow(db, old.id)).toEqual({
    project_id: w.web.id,
    owner: w.alice.id,
    made_by: w.alice.id,
    written_by: w.alice.id,
    deleted: true,
    stamped: true,
  });
  await dismissToasts(page);

  // Bugs' link copied: said so (Chromium lets the page write the clipboard once allowed, Firefox and WebKit a click);
  // refused by the browser: said it failed, never copied.
  if (browserName === "chromium") await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await chooseFromMenu(page, "Bugs", "Copy link");
  await expect(toast(page, "Link copied to clipboard")).toBeVisible();
  if (browserName === "chromium")
    expect(await page.evaluate(() => navigator.clipboard.readText())).toContain(viewPath(w.slug, w.web.id, bugs.id));
  await dismissToasts(page);
  await refuseClipboardWrites(page);
  await chooseFromMenu(page, "Bugs", "Copy link");
  await expect(toast(page, "Something went wrong. Please try again.")).toBeVisible();
  expect(await shownWithin(toast(page, "Link copied to clipboard"))).toBe(false);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
