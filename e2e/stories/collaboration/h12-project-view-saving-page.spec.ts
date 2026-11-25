import type { components } from "@nerve/api-client";
import type { Page } from "@playwright/test";

import { changeProject, createLabel, readProjectPreferences } from "../../fixtures/api";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { modalDialog } from "../../fixtures/dialogs";
import { cyclePath, dismissToasts, recordApi, toast } from "../../fixtures/cycle-pages";
import { createCycle, cyclePreferencesRead } from "../../fixtures/cycles";
import { addCondition } from "../../fixtures/issue-filters-page";
import { createIssue } from "../../fixtures/issues";
import { modulePath } from "../../fixtures/module-pages";
import { createModule, modulePreferencesRead } from "../../fixtures/modules";
import { bodiesSent, shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, moveWithinApp, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  deleteLabel,
  expectQueryHolds,
  listReads,
  pageMain,
  rowButton,
  viewPath,
  viewRead,
  viewWorld,
} from "../../fixtures/view-pages";
import { viewCalls, viewDefaults } from "../../fixtures/views";

// H12's page version, Save as (M7 P6a spec 2.4, 2.7, 2.11; R7, R8, R18, R28, R30): the view's page, the project's, a
// cycle's and a module's each offer Save view once the page's filters hold a condition, to those who may make a view; its
// dialog's name starts empty, and the new view keeps the page's conditions, the page's own term (a cycle's, a module's)
// and the display the page has confirmed, nerve's new view's page opening then. A layout chosen on a view's page, wide
// or narrow, changes its draft alone; a condition on what was deleted shows as unavailable and is saved back as it is.
// The view's page itself is h12-project-view-page.spec.ts's.

type View = components["schemas"]["View"];

/**
 * Saves what page shows as a new view of name by its filters row's Save view, its dialog's name empty to start with:
 * resolves with what it sent and nerve's new view, whose page then shows. The body is read as bodiesSent reads it,
 * routed in Chromium alone: a route in WebKit would make it fetch again the new view's page's code it preloaded.
 */
async function saveAs(page: Page, browserName: string, projectId: string, name: string) {
  await rowButton(page, "Save view").click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Create View");
  const field = dialog.getByRole("textbox", { name: "Title", exact: true });
  await expect(field).toHaveValue("");
  await field.fill(name);
  const views = `/api/v0/projects/${projectId}/views`;
  const bodies = await bodiesSent(page, browserName, "POST", views);
  const recorded = recordApi(page);
  const answer = await answerTo(page, "POST", views, () =>
    dialog.getByRole("button", { name: "Create View", exact: true }).click()
  );
  const [body] = bodies;
  expect(answer.status()).toBe(201);
  const made = (await answer.json()) as View;
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "View created successfully.")).toBeVisible();
  // the new view's page, its read of the view and of the list answered
  await expect(page).toHaveURL(new RegExp(`/views/${made.id}$`));
  await expect(pageMain(page).getByRole("button", { name, exact: true })).toBeVisible();
  await expect.poll(() => recorded().pending).toBe(0);
  expect(recorded().requests).toContain(viewRead(made.id));
  await dismissToasts(page);
  return { body, made };
}

/** Chooses the condition of the priority Urgent by the filters, the page's list read again under it. */
async function urgentOnly(page: Page, projectId: string): Promise<void> {
  await addCondition(page, "Priority");
  await answerTo(page, "GET", `/api/v0/projects/${projectId}/issues`, () =>
    page.getByRole("option", { name: "Urgent", exact: true }).click()
  );
  await page.keyboard.press("Escape");
}

test("H12 (page): on alice's view Hot, the table chosen wide and the calendar narrow change its draft alone, nothing sent; Save view's new view keeps its urgent condition and its calendar, its name empty to start with, and its page opens; Web's work items page offers Save view once it holds a condition, and the new view keeps it and the display Web's page has confirmed", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent" });
  await createIssue(api, w.alice.token, w.web.id, { name: "Typo", priority: "low" });
  const hot = await w.view(w.alice, { name: "Hot", filters: { priority: ["urgent"] } });

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  await page.goto(viewPath(w.slug, w.web.id, hot.id));
  await expect.poll(() => shownItems(page)).toEqual(["Crash"]);

  // The table, by the wide header's layouts; the calendar, by the narrow row's menu: the draft's alone.
  const pressed = (name: string) => pageMain(page).getByRole("button", { name: `${name} Layout`, exact: true });
  await pressed("Table").click();
  await expect(pressed("Table")).toHaveAttribute("aria-pressed", "true");
  const viewport = page.viewportSize();
  await page.setViewportSize({ width: 700, height: 720 });
  await pageMain(page).getByRole("button", { name: "Layout", exact: true }).click();
  await page.getByRole("menuitem", { name: "Calendar", exact: true }).click();
  if (viewport) await page.setViewportSize(viewport);
  await expect(pressed("Calendar")).toHaveAttribute("aria-pressed", "true");
  await expect(rowButton(page, "Update view")).toBeVisible();
  expect(watch.apiRequests.filter((request) => /^(PATCH|POST|DELETE) \/api\/v0\/(views|me)\//.test(request))).toEqual(
    []
  );

  // Save view: the condition and the calendar, nerve's default properties.
  const fromView = await saveAs(page, browserName, w.web.id, "Hot on the calendar");
  const calendar = { ...viewDefaults.display_filters, layout: "calendar" };
  expect(fromView.body).toEqual({
    name: "Hot on the calendar",
    description: "",
    logo_props: {},
    filters: { priority: ["urgent"] },
    display_filters: calendar,
    display_properties: viewDefaults.display_properties,
  });
  expect(await viewCalls(api).read(w.alice.token, fromView.made.id)).toEqual({ status: 200, view: fromView.made });

  // Web's work items: no Save view without a condition; with Urgent, the new view keeps it and Web's display.
  await moveWithinApp(page, `/${w.slug}/projects/${w.web.id}/issues`);
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await expect(rowButton(page, "Save view")).toHaveCount(0);
  await urgentOnly(page, w.web.id);
  const confirmed = await readProjectPreferences(api, w.alice.token, w.web.id);
  const fromProject = await saveAs(page, browserName, w.web.id, "Web's urgent");
  expect(fromProject.body).toEqual({
    name: "Web's urgent",
    description: "",
    logo_props: {},
    filters: { priority: ["urgent"] },
    display_filters: confirmed.display_filters,
    display_properties: confirmed.display_properties,
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): Sprint's page and Alpha's offer Save view once they hold a condition, and each new view keeps it, its cycle or its module, and the display the page has confirmed, its name empty to start with", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  expect(await changeProject(api, w.alice.token, w.web.id, { cycle_view: true, module_view: true })).toMatchObject({
    status: 200,
  });
  const sprint = await createCycle(api, w.alice.token, w.web.id, { name: "Sprint" });
  const alpha = await createModule(api, w.alice.token, w.web.id, { name: "Alpha" });
  await createIssue(api, w.alice.token, w.web.id, {
    name: "Crash",
    priority: "urgent",
    cycle_id: sprint.id,
    module_ids: [alpha.id],
  });
  await createIssue(api, w.alice.token, w.web.id, { name: "Typo", cycle_id: sprint.id, module_ids: [alpha.id] });

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  // Each page's progress sidebar closed, which lies over the right of its filters row.
  const details = async () => {
    const toggle = page.getByRole("button", { name: "Toggle details", exact: true });
    await expect(toggle).toHaveAttribute("aria-pressed", /true|false/);
    if ((await toggle.getAttribute("aria-pressed")) === "true") await toggle.click();
    await expect(toggle).toHaveAttribute("aria-pressed", "false");
  };
  const savedFrom = async (path: string, pages: "cycle" | "module", id: string, read: () => Promise<unknown>) => {
    await moveWithinApp(page, path);
    await details();
    await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
    await expect(rowButton(page, "Save view")).toHaveCount(0);
    await urgentOnly(page, w.web.id);
    const confirmed = (await read()) as {
      preferences: Pick<typeof viewDefaults, "display_filters" | "display_properties">;
    };
    const { body } = await saveAs(page, browserName, w.web.id, `Urgent in ${pages}`);
    expect(body).toEqual({
      name: `Urgent in ${pages}`,
      description: "",
      logo_props: {},
      filters: { priority: ["urgent"], ...(pages === "cycle" ? { cycle_id: [id] } : { module_id: [id] }) },
      display_filters: confirmed.preferences.display_filters,
      display_properties: confirmed.preferences.display_properties,
    });
  };
  await page.goto(`/${w.slug}/projects/${w.web.id}/issues`);
  await expect.poll(() => shownItems(page)).toEqual(["Crash", "Typo"]);
  await savedFrom(cyclePath(w.slug, w.web.id, sprint.id), "cycle", sprint.id, () =>
    cyclePreferencesRead(api, w.alice.token, sprint.id)
  );
  await savedFrom(modulePath(w.slug, w.web.id, alpha.id), "module", alpha.id, () =>
    modulePreferencesRead(api, w.alice.token, alpha.id)
  );

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H12 (page): alice's view Legacy, whose label was deleted, shows its condition's value as unavailable and lists under it as it is, offering no Update view; with Urgent besides, Update view saves the deleted label back as it was", async ({
  api,
  signedInPage,
}, testInfo) => {
  const w = await viewWorld(api, testInfo);
  const legacy = await createLabel(api, w.alice.token, w.web.id, { name: "Legacy" });
  await createIssue(api, w.alice.token, w.web.id, { name: "Crash", priority: "urgent" });
  const view = await w.view(w.alice, { name: "Legacy", filters: { label_id: [legacy.id] } });
  await deleteLabel(api, w.alice.token, legacy.id);

  const page = await signedInPage(w.alice.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  await page.goto(viewPath(w.slug, w.web.id, view.id));
  await expect(pageMain(page).getByText("Unavailable option", { exact: true })).toBeVisible();
  await expect.poll(() => listReads(recorded().requests, w.web.id).length).toBe(1);
  expectQueryHolds(listReads(recorded().requests, w.web.id)[0] ?? new URLSearchParams(), { label_id: [legacy.id] });
  await expect(rowButton(page, "Update view")).toHaveCount(0);

  await urgentOnly(page, w.web.id);
  const { body, answer } = await sentTo(page, "PATCH", `/api/v0/views/${view.id}`, () =>
    rowButton(page, "Update view").click()
  );
  expect(answer.status()).toBe(200);
  expect((body as { filters: unknown }).filters).toEqual({ label_id: [legacy.id], priority: ["urgent"] });
  await expect(toast(page, "View updated successfully.")).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
