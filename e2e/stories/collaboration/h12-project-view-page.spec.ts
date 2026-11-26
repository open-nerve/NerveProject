import { randomUUID } from "node:crypto";

import type { Page } from "@playwright/test";

import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { failedLoads, menuEntries, recordApi, seenBy, toast } from "../../fixtures/cycle-pages";
import { addCondition } from "../../fixtures/issue-filters-page";
import { createIssue } from "../../fixtures/issues";
import { APP, PROJECT, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
import { shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, holdAnswer, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import {
  expectDraftControls,
  expectQueryHolds,
  layoutButtons,
  listReads,
  pageMain,
  rowButton,
  turnViews,
  viewPath,
  viewRead,
  viewReads,
  viewWorld,
  viewsCrumb,
  viewsPath,
  viewsRead,
} from "../../fixtures/view-pages";
import { savedQuery, viewCalls, viewDefaults } from "../../fixtures/views";

// H12's page version, a project's view's page (M7 design 3.7; P6a spec 2.2, 2.6, 2.7, 2.11; R2, R6, R9, R10, R29):
// opened, it reads the view once, its own read whatever SWR keeps, and then the project's work items under the view's
// words; its filters change the page's draft alone, which Update view saves for the view's owner, waiting for nerve
// while every way to change the draft stops; the draft dies with the page. Its header gives the crumbs to the project's
// views in each of its states (reading, not found, failed, off), and the view's controls once it shows the view. Save
// as is h12-project-view-saving-page.spec.ts's, the layouts h12-project-view-layouts-page.spec.ts's.

/** Waits for the reads page has out to be answered (recordApi's), as a story does before it moves or changes nerve. */
async function settled(page: Page, recorded: () => { pending: number }): Promise<void> {
  await expect.poll(() => recorded().pending, "the reads out answered").toBe(0);
}

test("H12 (page): alice's view Hot, opened anew, reads the app's start, the workspace's, the project's, the view once and Web's work items under its words, listing Crash alone; its Display button says whether it is open, and Escape gives the focus back to it; its switcher reads Web's views only once it opens; her filters' change sends nothing and offers Update view; left for Web's views and entered again, twice, the draft gone, it reads the view once more each time and lists Crash alone again; on a narrow screen its own row gives the layouts and the display", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent" });
  await createIssue(api, w.alice.token, w.web.id, { name: "Typo", priority: "low" });
  const hot = await w.view(w.alice, { name: "Hot", filters: { priority: ["urgent"] } });

  // Opened anew: the app's start, the workspace's, Web's, its resources, the view once, and Web's list under the
  // view's words with the list's own options.
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(viewPath(w.slug, w.web.id, hot.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await settled(page, opened);
  const [listRead] = listReads(opened().requests, w.web.id);
  expect(opened().requests.filter((request) => !request.includes("/issues?"))).toEqual(
    [
      ...[...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER].map((request) =>
        valued(request, { [w.slug]: "{slug}", [w.web.id]: "{project}" })
      ),
      viewRead(hot.id),
    ].toSorted()
  );
  expect(listReads(opened().requests, w.web.id)).toHaveLength(1);
  expectQueryHolds(listRead ?? new URLSearchParams(), { ...savedQuery(hot), limit: "100" });
  // The header: the crumbs to Web's views and the view's switcher, the layouts, the list's pressed.
  await expect(viewsCrumb(page, w.slug, w.web.id)).toBeVisible();
  await expect(pageMain(page).getByRole("button", { name: "Hot", exact: true })).toBeVisible();
  await expect(layoutButtons(page)).toHaveCount(4);
  await expect(pageMain(page).getByRole("button", { name: "List Layout", exact: true })).toHaveAttribute(
    "aria-pressed",
    "true"
  );

  // Display, named so: closed, then open; Escape closes it, the focus back on it.
  const display = pageMain(page).getByRole("button", { name: "Display", exact: true });
  await expect(display).toHaveAttribute("aria-expanded", "false");
  await display.click();
  await expect(display).toHaveAttribute("aria-expanded", "true");
  await page.keyboard.press("Escape");
  await expect(display).toHaveAttribute("aria-expanded", "false");
  await expect(display).toBeFocused();

  // The switcher reads Web's views once it opens, and only then, and lists them.
  const switched = recordApi(page);
  await pageMain(page).getByRole("button", { name: "Hot", exact: true }).click();
  await expect(page.getByRole("option", { name: "Hot" })).toBeVisible();
  await page.keyboard.press("Escape");
  await settled(page, switched);
  expect(switched().requests).toEqual([viewsRead(w.web.id)]);

  // The filters' change is the draft's: the list read again, nothing sent, Update view offered. Left for Web's views
  // and entered again, the draft is gone: the view read once more, Crash alone; then again: read once more. Whether an
  // entry falls within SWR's dedupe window of the read before (in which a mount's plain read would not read it again)
  // is the real time's, not the story's: use-view-page-read.test.tsx pins an entry within it (review m2).
  const change = async () => {
    await answerTo(page, "GET", `/api/v0/projects/${w.web.id}/issues`, () =>
      pageMain(page).getByRole("button", { name: "Remove filter", exact: true }).click()
    );
    await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
    await expect(rowButton(page, "Update view")).toBeVisible();
  };
  const reenter = async () => {
    const again = recordApi(page);
    await viewsCrumb(page, w.slug, w.web.id).click();
    const link = page.getByRole("link", { name: "Hot", exact: true });
    await expect(link).toBeVisible();
    await settled(page, again);
    await answerTo(page, "GET", `/api/v0/views/${hot.id}`, () => link.click());
    await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
    await settled(page, again);
    expect(again().requests.filter((request) => request === viewRead(hot.id))).toHaveLength(1);
    await expect(pageMain(page).getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);
    await expect(rowButton(page, "Update view")).toHaveCount(0);
  };
  await change();
  await reenter();
  await reenter();
  expect(watch.apiRequests.filter((request) => !request.startsWith("GET ")).toSorted()).toEqual([
    "POST /api/v0/auth/refresh",
  ]);

  // A narrow screen: the page's own row gives the layouts and the display.
  await page.setViewportSize({ width: 700, height: 720 });
  await expect(pageMain(page).getByRole("button", { name: "Layout", exact: true })).toBeVisible();
  await expect(pageMain(page).getByRole("button", { name: "Display", exact: true })).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): the header of Web's view's page gives the crumbs to Web's views in each state of the page, and the view's switcher, layouts and display once it shows the view: while the view is read; for a view nerve does not have, which says so and goes back to the views; for Ops' view, which is not Web's; for a read that failed, which Retry reads again (Chromium); and for Lab, its views off, which reads none", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const urgent = await w.view(w.alice, { name: "Urgent" });
  const ops = await w.project("Ops", "OPS", true);
  const opsView = await w.view(w.alice, { name: "Ops view" }, ops.id);
  const lab = await w.project("Lab", "LAB", false);
  const absent = randomUUID();

  // The view's page, then Web's views: their code loaded before any route is (WebKit's preloads).
  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  const crumb = viewsCrumb(page, w.slug, w.web.id);
  await page.goto(viewPath(w.slug, w.web.id, urgent.id));
  const switcher = pageMain(page).getByRole("button", { name: "Urgent", exact: true });
  const display = pageMain(page).getByRole("button", { name: "Display", exact: true });
  await expect(display).toBeVisible();
  await answerTo(page, "GET", `/api/v0/projects/${w.web.id}/views`, () => crumb.click());
  await settled(page, recorded);
  /** The header shows the crumbs alone: Web's views, no view's switcher, layouts or display. */
  const crumbsAlone = async () => {
    await expect(crumb).toBeVisible();
    await expect(switcher).toHaveCount(0);
    await expect(layoutButtons(page)).toHaveCount(0);
    await expect(display).toHaveCount(0);
  };

  // Read: while the view's answer is held, the crumbs alone; let through, the view's controls.
  const release = await holdAnswer(page, "GET", `/api/v0/views/${urgent.id}`);
  await page.getByRole("link", { name: "Urgent", exact: true }).click();
  await expect(page).toHaveURL(viewPath(w.slug, w.web.id, urgent.id));
  await expect(pageMain(page).getByRole("button", { name: "Add view", exact: true })).toHaveCount(0);
  await crumbsAlone();
  await release();
  await expect(switcher).toBeVisible();
  await expect(display).toBeVisible();
  await settled(page, recorded);

  // A view nerve does not have: not found, the crumbs alone; its button goes back to Web's views.
  await moveWithinApp(page, viewPath(w.slug, w.web.id, absent));
  await expect(page.getByText("View does not exist", { exact: true })).toBeVisible();
  await crumbsAlone();
  await page.getByRole("button", { name: "View other views", exact: true }).click();
  await expect(page).toHaveURL(viewsPath(w.slug, w.web.id));
  await settled(page, recorded);
  // Ops' view, under Web: not Web's, so not found either.
  await moveWithinApp(page, viewPath(w.slug, w.web.id, opsView.id));
  await expect(page.getByText("View does not exist", { exact: true })).toBeVisible();
  await crumbsAlone();
  await settled(page, recorded);

  // A read that failed (Chromium alone, a route that answers it 500 once): why, the crumbs alone; Retry reads again.
  const failures: string[] = [];
  const errors = failedLoads(browserName, 404);
  if (browserName === "chromium") {
    let failed = false;
    await page.route(
      (url) => url.pathname === `/api/v0/views/${urgent.id}`,
      async (route) => {
        if (failed) return route.fallback();
        failed = true;
        return route.fulfill({ status: 500, contentType: "application/problem+json", body: "{}" });
      }
    );
    await moveWithinApp(page, viewPath(w.slug, w.web.id, urgent.id));
    await expect(page.getByRole("alert")).toBeVisible();
    await crumbsAlone();
    await page.getByRole("button", { name: "Retry", exact: true }).click();
    await expect(display).toBeVisible();
    failures.push(`500 GET /api/v0/views/${urgent.id}`);
    errors.push("Failed to load resource: the server responded with a status of 500 (Internal Server Error)");
  }
  await settled(page, recorded);

  // Lab, its views off: the disabled page, its crumbs alone, no view read.
  const toLab = recordApi(page);
  await moveWithinApp(page, viewPath(w.slug, lab.id, absent));
  await expect(page.getByText("Views are not enabled for the project.", { exact: true })).toBeVisible();
  await expect(viewsCrumb(page, w.slug, lab.id)).toBeVisible();
  await expect(display).toHaveCount(0);
  await settled(page, toLab);
  expect(viewReads(toLab().requests)).toEqual([]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/views/${absent}`, ...failures],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors });
});

test("H12 (page): alice's view Mine: her urgent condition is the draft's, offering Update view; its answer held, the button says Confirming and the conditions, Add filter, Clear all, Save view, the layouts and the display stop; let through, it says so and goes, the controls back, nerve's view the page's words and display, as nerve reads it back; bob, a member, is offered neither Update view nor Edit; her Backlog condition, Web's views turned off meanwhile, is refused: said why, the controls back, the draft and Update view kept", async ({
  api,
  baseURL,
  browser,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const states = await stateIds(api, w.alice.token, w.web.id);
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent", state_id: states.Backlog });
  await createIssue(api, w.alice.token, w.web.id, { name: "Typo", priority: "low" });
  const mine = await w.view(w.alice, { name: "Mine" });
  const change = `/api/v0/views/${mine.id}`;
  const issues = `/api/v0/projects/${w.web.id}/issues`;

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  await page.goto(viewPath(w.slug, w.web.id, mine.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  const recorded = recordApi(page);

  // Urgent: the draft's, the list read under it, nothing sent; Update view offered.
  await addCondition(page, "Priority");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Urgent", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  await expect(rowButton(page, "Update view")).toBeVisible();
  await settled(page, recorded);
  const before = recorded().requests.length;

  // Update view, its answer held: Confirming, and every way to change the draft stopped; let through, said, gone.
  const saved = await sentHeld(page, "PATCH", change, () => rowButton(page, "Update view").click());
  await expect(rowButton(page, "Confirming")).toBeDisabled();
  await expectDraftControls(page, false, 1);
  const answer = await saved.release();
  expect(answer.status()).toBe(200);
  await expect(toast(page, "View updated successfully.")).toBeVisible();
  await expect(rowButton(page, "Update view")).toHaveCount(0);
  await expect(rowButton(page, "Confirming")).toHaveCount(0);
  await expectDraftControls(page, true, 1);
  const body = {
    filters: { priority: ["urgent"] },
    display_filters: viewDefaults.display_filters,
    display_properties: viewDefaults.display_properties,
  };
  expect(saved.body).toEqual(body);
  const updated = await answer.json();
  expect(updated).toEqual({ ...mine, ...body, updated_at: updated.updated_at });
  expect(await viewCalls(api).read(w.alice.token, mine.id)).toEqual({ status: 200, view: updated });
  // nerve's answer the page's words as they were: no read follows it.
  await settled(page, recorded);
  expect(recorded().requests.slice(before)).toEqual([`PATCH ${change}`]);

  // bob, a member: his own condition offers him no Update view; the view's menu, no Edit.
  await seenBy(browser, baseURL ?? "", api, w.bob.email, viewPath(w.slug, w.web.id, mine.id), async (other) => {
    await expect.poll(() => shownItems(other)).toEqual(["Crash"]);
    await other.getByRole("button", { name: "Remove filter", exact: true }).click();
    await expect.poll(() => shownItems(other)).toEqual(["Crash", "Typo"]);
    await expect(rowButton(other, "Update view")).toHaveCount(0);
    expect(await menuEntries(other, "Mine")).toEqual(["Open in new tab", "Copy link"]);
  });

  // Backlog besides, Web's views then turned off by alice elsewhere: held, the controls stop; refused, said why, the
  // controls back, the draft's two conditions, its list and Update view kept.
  await addCondition(page, "State");
  await answerTo(page, "GET", issues, () => page.getByRole("option", { name: "Backlog", exact: true }).click());
  await page.keyboard.press("Escape");
  await expect(rowButton(page, "Update view")).toBeVisible();
  await settled(page, recorded);
  await turnViews(api, w.alice.token, w.web, false);
  const refused = await sentHeld(page, "PATCH", change, () => rowButton(page, "Update view").click());
  await expectDraftControls(page, false, 2);
  expect((await refused.release()).status()).toBe(409);
  await expect(toast(page, "Views are turned off in this project. Turn them on to change them.")).toBeVisible();
  await expectDraftControls(page, true, 2);
  await expect(rowButton(page, "Update view")).toBeVisible();
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);
  expect(refused.body).toEqual({ ...body, filters: { priority: ["urgent"], state_id: [states.Backlog] } });
  expect(await viewCalls(api).read(w.alice.token, mine.id)).toEqual({ status: 200, view: updated });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`409 PATCH ${change}`], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
});
