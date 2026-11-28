import { randomUUID } from "node:crypto";

import type { components } from "@nerve/api-client";
import type { Page } from "@playwright/test";

import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dismissToasts, failedLoads, menuEntries, recordApi, seenBy, toast } from "../../fixtures/cycle-pages";
import { focusInDialog, modalDialog } from "../../fixtures/dialogs";
import { addCondition } from "../../fixtures/issue-filters-page";
import { createIssue } from "../../fixtures/issues";
import { APP, WORKSPACE, valued } from "../../fixtures/mounts";
import { bodiesSent, shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { expectQueryHolds, pageMain, rowButton, turnViews, viewRead } from "../../fixtures/view-pages";
import { createProjectView, createWorkspaceView, savedQuery, viewCalls } from "../../fixtures/views";
import {
  catalogRead,
  expectPageControls,
  noteFocus,
  offered,
  orderBy,
  settled,
  switcher,
  workspaceListReads,
  workspaceViewContent,
  workspaceViewPath,
  workspaceViewWorld,
  workspaceViewsPath,
  workspaceViewsRead,
  withoutRowProjects,
} from "../../fixtures/workspace-view-pages";

// H13's page version, a saved workspace view's page (M7 design 2, 3.7; M7/P6b spec 2.7–2.9, 2.15; R8, R10, R11, R29,
// R30): opened, it reads the view once, its own read, and the workspace's label catalog once, then the workspace's list
// under the view's words; its filters and its order change the page's draft alone, which Update view saves, whole as
// the draft is in its turn, for the view's owner, waiting for nerve while every way to change the draft stops. Save
// view makes a view of the workspace's own of what a page shows, a default view's own term with it, in the spreadsheet.
// A view held in the list layout is drawn as the table and saved back as it is. A view nerve has not, a project's view
// and any view to the workspace's guest, who reads none, does not exist, with the way back to the workspace's views.
// The default views are h13-workspace-list-page.spec.ts's.

type View = components["schemas"]["View"];

/** The words a story's change of a view's page saves, in nerve's words: its filters and its display as given. */
function saved(filters: View["filters"], order = "-created_at") {
  return {
    filters,
    display_filters: { ...workspaceViewContent.display_filters, order_by: order },
    display_properties: workspaceViewContent.display_properties,
  };
}

/** What a workspace view's page says of a view that does not exist to its viewer (R29). */
function notFound(page: Page) {
  return page.getByText("View does not exist", { exact: true });
}

/** The address of the API's view of id, which a view's page reads. */
function viewPath(id: string): string {
  return `/api/v0/views/${id}`;
}

/** Chooses the condition of option for the filter of name on page, the list read again under it. */
async function choose(page: Page, slug: string, name: string, option: string): Promise<void> {
  await addCondition(page, name);
  await answerTo(page, "GET", `/api/v0/workspaces/${slug}/issues`, () =>
    page.getByRole("option", { name: option, exact: true }).click()
  );
  await page.keyboard.press("Escape");
}

test("H13 (page): alice's view Hot, opened anew, reads the app's start, the workspace's, the view once, the catalog once and the workspace's list under its words, listing Crash and Outage; her Bug condition and the order by priority send nothing and offer Update view; its answer held, the button says Confirming and the conditions, Add filter, Clear all, Save view, Display and the columns' order stop; let through, it says so, the controls back, the view the draft's words and display as nerve reads it back, no read following; bob, a member, is offered neither Update view nor Edit; her next change, Hot deleted elsewhere meanwhile, is refused: said why, the controls and Update view kept", async ({
  api,
  baseURL,
  browser,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  await createIssue(api, w.alice.token, w.ops.id, { name: "Outage", priority: "urgent" });
  const hot = await createWorkspaceView(api, w.alice.token, w.slug, {
    name: "Hot",
    filters: { priority: ["urgent"] },
    ...workspaceViewContent,
  });
  const issues = `/api/v0/workspaces/${w.slug}/issues`;
  const change = `/api/v0/views/${hot.id}`;

  // Opened anew: the view once, the catalog once, the list once under its words.
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, hot.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Outage"]);
  await settled(page, recorded);
  const [listRead, ...more] = workspaceListReads(recorded().requests, w.slug);
  expect(more).toEqual([]);
  expectQueryHolds(listRead ?? new URLSearchParams(), { ...savedQuery(hot), limit: "100" });
  const others = withoutRowProjects(recorded().requests, [w.web.id, w.ops.id]);
  expect(others.filter((request) => !request.startsWith(`GET ${issues}?`))).toEqual(
    [
      ...[...APP, ...WORKSPACE].map((request) => valued(request, { [w.slug]: "{slug}" })),
      catalogRead(w.slug),
      viewRead(hot.id),
    ].toSorted()
  );
  // The switcher reads Acme's own views once it opens, and only then, and offers them after the four default views.
  const switched = recordApi(page);
  await switcher(page, "Hot").click();
  await settled(page, switched);
  expect(await offered(page)).toEqual(["All work items", "Assigned", "Created", "Subscribed", "Hot"]);
  await page.keyboard.press("Escape");
  expect(switched().requests).toEqual([workspaceViewsRead(w.slug)]);

  // Bug and the order by priority: the draft's, the list read under them, nothing sent; Update view offered.
  await choose(page, w.slug, "Label", "Bug");
  await answerTo(page, "GET", issues, () => orderBy(page, "Priority", "Urgent None"));
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await expect(rowButton(page, "Update view")).toBeVisible();
  await settled(page, recorded);
  const updating = recordApi(page);

  // Update view, its answer held: Confirming, every way to change the draft stopped; let through, said, gone.
  const update = await sentHeld(page, "PATCH", change, () => rowButton(page, "Update view").click());
  await expect(rowButton(page, "Confirming")).toBeDisabled();
  await expectPageControls(page, false, 2);
  const answer = await update.release();
  expect(answer.status()).toBe(200);
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await expect(rowButton(page, "Update view")).toHaveCount(0);
  await expectPageControls(page, true, 2);
  const body = saved({ priority: ["urgent"], label_id: [w.bug.id] }, "-priority");
  expect(update.body).toEqual(body);
  const updated = (await answer.json()) as View;
  expect(updated).toEqual({ ...hot, ...body, updated_at: updated.updated_at });
  expect(await viewCalls(api).read(w.alice.token, hot.id)).toEqual({ status: 200, view: updated });
  await settled(page, updating);
  expect(updating().requests).toEqual([`PATCH ${change}`]);
  await dismissToasts(page);

  // bob, a member: his own change offers him no Update view; the view's menu, no Edit.
  await seenBy(browser, baseURL ?? "", api, w.bob.email, workspaceViewPath(w.slug, hot.id), async (other) => {
    await expect.poll(() => shownItems(other)).toEqual(["Crash"]);
    await other.getByRole("button", { name: "Remove filter", exact: true }).last().click();
    await expect.poll(() => shownItems(other)).toEqual(["Crash", "Outage"]);
    await expect(rowButton(other, "Update view")).toHaveCount(0);
    expect(await menuEntries(other, "Hot")).toEqual(["Open in new tab", "Copy link"]);
  });

  // Bug taken off, Hot then deleted by alice elsewhere: held, the controls stop; refused, said why, the controls back,
  // Update view and the draft's list kept.
  await answerTo(page, "GET", issues, () =>
    pageMain(page).getByRole("button", { name: "Remove filter", exact: true }).last().click()
  );
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Outage"]);
  await expect(rowButton(page, "Update view")).toBeVisible();
  await settled(page, recorded);
  expect(await viewCalls(api).remove(w.alice.token, hot.id)).toEqual({ status: 204 });
  const refused = await sentHeld(page, "PATCH", change, () => rowButton(page, "Update view").click());
  await expectPageControls(page, false, 1);
  expect((await refused.release()).status()).toBe(404);
  await expect(toast(page, "The view does not exist, or you cannot see it.")).toBeVisible();
  await expectPageControls(page, true, 1);
  await expect(rowButton(page, "Update view")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Outage"]);
  expect(refused.body).toEqual(saved({ priority: ["urgent"] }, "-priority"));

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`404 PATCH ${change}`], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404) });
});

test("H13 (page): bob's view Plain, made through the API in the list layout, is drawn as the table, and the Update view of its order keeps the list layout; on Assigned, Save view comes with his Urgent condition, its name empty to start with, and the new view keeps the condition, Assigned's own term and the table, its page opening then", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  const plain = await createWorkspaceView(api, w.bob.token, w.slug, { name: "Plain" });
  expect(plain.display_filters.layout).toBe("list");
  const issues = `/api/v0/workspaces/${w.slug}/issues`;

  // Plain: the table; Urgent, ordered by priority, Update view sends the list layout back as it was.
  const page = await signedInPage(w.bob.tokens);
  const watch = await watchPage(page);
  await page.goto(workspaceViewPath(w.slug, plain.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
  await expect(page.getByRole("table")).toBeVisible();
  await choose(page, w.slug, "Priority", "Urgent");
  await answerTo(page, "GET", issues, () => orderBy(page, "Priority", "Urgent None"));
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  const change = `/api/v0/views/${plain.id}`;
  const patches = await bodiesSent(page, browserName, "PATCH", change);
  const answer = await answerTo(page, "PATCH", change, () => rowButton(page, "Update view").click());
  expect(answer.status()).toBe(200);
  expect(patches).toHaveLength(1);
  expect(patches[0]).toEqual({
    filters: { priority: ["urgent"] },
    display_filters: { ...plain.display_filters, order_by: "-priority" },
    display_properties: plain.display_properties,
  });
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await dismissToasts(page);

  // Assigned: no Save view until a condition; Urgent, then Save view: a new view of Acme's own.
  await answerTo(page, "GET", issues, () => moveWithinApp(page, workspaceViewPath(w.slug, "assigned")));
  await expect(switcher(page, "Assigned")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await expect(rowButton(page, "Save view")).toHaveCount(0);
  await choose(page, w.slug, "Priority", "Urgent");
  await rowButton(page, "Save view").click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create View");
  await expect.poll(() => focusInDialog(page), "the focus in the dialog").toBe(true);
  await noteFocus(page, testInfo, "Save view's dialog");
  const field = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(field).toHaveValue("");
  await field.fill("Mine urgent");
  const views = `/api/v0/workspaces/${w.slug}/views`;
  const bodies = await bodiesSent(page, browserName, "POST", views);
  const recorded = recordApi(page);
  const made = await answerTo(page, "POST", views, () =>
    dialog.getByRole("button", { name: "Create View", exact: true }).click()
  );
  expect(made.status()).toBe(201);
  const view = (await made.json()) as View;
  expect(bodies).toEqual([
    {
      name: "Mine urgent",
      description: "",
      logo_props: {},
      ...saved({ priority: ["urgent"], assignee_id: ["me"] }),
    },
  ]);
  await expect(toast(page, "View created successfully.")).toBeVisible();
  await expect(page).toHaveURL(workspaceViewPath(w.slug, view.id));
  await expect(switcher(page, "Mine urgent")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, recorded);
  expect(recorded().requests.filter((request) => request === viewRead(view.id))).toHaveLength(1);
  expect(await viewCalls(api).read(w.bob.token, view.id)).toEqual({ status: 200, view });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H13 (page): a view nerve does not have and Web's project view, at a workspace view's address, do not exist, and Go to views goes back to the workspace's views; a read that failed (Chromium) says so and Retry reads it again; gus, the guest, opening bob's view's address, is told it does not exist and reads no view", async ({
  api,
  baseURL,
  browser,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await workspaceViewWorld(api, testInfo);
  const mine = await createWorkspaceView(api, w.bob.token, w.slug, { name: "Mine", ...workspaceViewContent });
  await turnViews(api, w.alice.token, w.web, true);
  const webView = await createProjectView(api, w.alice.token, w.web.id, { name: "Web view" });
  const absent = randomUUID();

  // A view nerve has not: it does not exist; Go to views, the workspace's views.
  const page = await signedInPage(w.bob.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(workspaceViewPath(w.slug, mine.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
  await settled(page, recorded);
  await answerTo(page, "GET", viewPath(absent), () => moveWithinApp(page, workspaceViewPath(w.slug, absent)));
  await expect(notFound(page)).toBeVisible();
  await expect(switcher(page, "Mine")).toHaveCount(0);
  await answerTo(page, "GET", `/api/v0/workspaces/${w.slug}/views`, () =>
    page.getByRole("button", { name: "Go to views", exact: true }).click()
  );
  await expect(page).toHaveURL(workspaceViewsPath(w.slug));
  await expect(page.getByRole("link", { name: "Mine", exact: true })).toBeVisible();
  await settled(page, recorded);
  // Web's view, which nerve gives at its id: not the workspace's own, so not found either.
  await answerTo(page, "GET", viewPath(webView.id), () => moveWithinApp(page, workspaceViewPath(w.slug, webView.id)));
  await expect(notFound(page)).toBeVisible();
  await settled(page, recorded);
  expect(recorded().requests.filter((request) => request === viewRead(webView.id))).toHaveLength(1);

  // A read that failed (Chromium alone, a route that answers it 500 once): why, and Retry reads it again.
  const failures: string[] = [];
  const errors = failedLoads(browserName, 404);
  if (browserName === "chromium") {
    let failed = false;
    await page.route(
      (url) => url.pathname === `/api/v0/views/${mine.id}`,
      async (route) => {
        if (failed) return route.fallback();
        failed = true;
        return route.fulfill({ status: 500, contentType: "application/problem+json", body: "{}" });
      }
    );
    await moveWithinApp(page, workspaceViewPath(w.slug, mine.id));
    await expect(page.getByRole("alert")).toBeVisible();
    await page.getByRole("button", { name: "Retry", exact: true }).click();
    await expect.poll(() => shownItems(page)).toEqual(["Crash", "Deploy", "Note", "Typo"]);
    failures.push(`500 GET /api/v0/views/${mine.id}`);
    errors.push("Failed to load resource: the server responded with a status of 500 (Internal Server Error)");
  }
  await settled(page, recorded);

  // gus, the guest: Mine does not exist to him, and he reads no view.
  await seenBy(browser, baseURL ?? "", api, w.gus.email, workspaceViewPath(w.slug, mine.id), async (other, seen) => {
    await expect(notFound(other)).toBeVisible();
    await expect(pageMain(other).getByRole("link", { name: "Views", exact: true })).toBeVisible();
    expect(seen.apiRequests.filter((request) => request.startsWith("GET /api/v0/views/"))).toEqual([]);
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/views/${absent}`, ...failures],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors });
});
