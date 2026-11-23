import { projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { intakeRowOf, notificationRowsOf, triageActivities } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dismissToasts, failedLoads, recordApi, toast } from "../../fixtures/cycle-pages";
import { triageIntakeIssue } from "../../fixtures/intake";
import {
  expectHeaderOf,
  identifierOf,
  intakeDetail,
  intakeEntry,
  intakePath,
  intakeRow,
  intakeWorld,
  rowName,
  selecting,
  shownRows,
  shownStatus,
  submitNamed,
  triagePath,
} from "../../fixtures/intake-pages";
import { createIssue } from "../../fixtures/issues";
import { answerTo, bodiesSentTo, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H10's page version, its acceptance (M7 design 2, 3.5; P5a spec 2.4, 2.8, 2.12, R4, R12, R14, R28, R29, R35; re-94
// I2, m9): a work item of a project's intake accepted by its admin, on the intake page, through M4's issue modal. Its
// submit triages the work item first, then sends what the form changed as M4's update; the selection moves to the next
// row only once that update is confirmed and the modal closed, whatever the form changed, nothing too. Accepted, the
// work item is in the project's list, which the app shows without a reload, the work items still in the intake not;
// its submitter is sent its new state. Accepting a work item another admin accepted meanwhile is refused: no update
// follows, the modal closes, and the header, read again, says it is accepted. The other triages are
// h10-intake-triage-page.spec.ts's; the API version is h10-intake-triage.spec.ts.

/** How a live intake row reads, neither snoozed nor a duplicate. */
const live = { snoozed_till: null, duplicate_to_id: null };

test("H10 (page): alice, Web's admin, goes from its work items to its intake by the sidebar and accepts the work item selected through the modal, its priority changed to High: one triage, then one update with the priority, the selection moving to the next row only once that update is confirmed and the modal closed; she accepts that one unchanged, the selection moving on; back on Web's work items, without a reload, the two are listed once each, the pending one not; each has one acceptance in its activity, and bob, who submitted them, is sent their new state and the priority", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const [dark, slow, crash] = await submitNamed(api, bob.token, web.id, ["Dark mode", "Slow search", "Crash on save"]);

  // Web's work items first, the window marked, then its intake by the sidebar: the newest selected.
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await expect(page.locator(`#issue-${fix.id}`)).toBeVisible();
  await page.evaluate(() => Object.assign(window, { h10: "loaded once" }));
  await intakeEntry(page, slug, web.id, 3).click();
  await expect.poll(() => shownRows(page)).toEqual([crash, slow, dark].map(rowName));
  await expect(page).toHaveURL(selecting(crash));
  // Its detail drawn before any request is read on its way: a route set while the page still loads the detail's code
  // makes WebKit fetch that code again, its preloads unused, and warn (as in h20-intake-late-pages.spec.ts;
  // cp0fix-red-h10-load, under load)
  await expectHeaderOf(page, crash);

  // Accepted through the modal, its state Web's default, its priority changed: the triage, then, held, the update
  // carrying the priority alone with the description the form always sends; meanwhile the modal says the work item is
  // accepted and stays, the selection kept. Once the update is confirmed: the next row focused, as the triage ends, and
  // selected, the modal closed, the update said.
  const triaged = await bodiesSentTo(page, "POST", (path) => path.endsWith("/triage"));
  await intakeDetail(page).getByRole("button", { name: "Accept", exact: true }).click();
  const moving = page.getByRole("dialog", { name: `Move ${identifierOf(crash)} to project work items`, exact: true });
  await expect(moving.getByRole("button", { name: "State: Backlog", exact: true })).toBeVisible();
  await moving.getByRole("button", { name: "Priority: None", exact: true }).click();
  await page.getByRole("option", { name: "High", exact: true }).click();
  const update = await sentHeld(page, "PATCH", `/api/v0/issues/${crash.id}`, () =>
    moving.getByRole("button", { name: "Add to project", exact: true }).click()
  );
  expect(triaged).toEqual([{ status: "accepted" }]);
  expect(update.body).toEqual({ description_html: "<p></p>", priority: "high" });
  const accepted = page.getByRole("dialog", {
    name: `${identifierOf(crash)} is accepted; your changes to it are not saved yet`,
    exact: true,
  });
  await expect(accepted).toHaveCount(1);
  await expect(page).toHaveURL(selecting(crash));
  expect((await update.release()).status()).toBe(200);
  await expect(intakeRow(page, slow)).toBeFocused();
  await expect(page).toHaveURL(selecting(slow));
  await expect(accepted).toHaveCount(0);
  await expect(toast(page, "Work item updated successfully")).toBeVisible();
  await expect.poll(() => shownRows(page)).toEqual([slow, dark].map(rowName));

  // Slow search accepted unchanged: whatever M4's update then does, nothing fails, and the selection moves on.
  await dismissToasts(page);
  await expectHeaderOf(page, slow);
  await intakeDetail(page).getByRole("button", { name: "Accept", exact: true }).click();
  const unchanged = page.getByRole("dialog", { name: `Move ${identifierOf(slow)} to project work items`, exact: true });
  await unchanged.getByRole("button", { name: "Add to project", exact: true }).click();
  await expect(intakeRow(page, dark)).toBeFocused();
  await expect(page).toHaveURL(selecting(dark));
  await expect(unchanged).toHaveCount(0);
  expect(triaged).toEqual([{ status: "accepted" }, { status: "accepted" }]);

  // Back on Web's work items, by the sidebar, the page never loaded again: the two accepted, Crash on save high, and
  // Fix the login, each once; Dark mode, pending, not (P5a spec 6).
  await page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: "Work items", exact: true })
    .and(page.locator(`a[href="/${slug}/projects/${web.id}/issues"]`))
    .click();
  await expect
    .poll(() => page.locator('[id^="issue-"]').evaluateAll((rows) => rows.map((row) => row.id)))
    .toEqual([crash, slow, fix].map((issue) => `issue-${issue.id}`));
  await expect(
    page.locator(`#issue-${crash.id}`).getByRole("button", { name: "Priority: high", exact: true })
  ).toBeVisible();
  expect(await page.evaluate(() => (window as Window & { h10?: string }).h10)).toBe("loaded once");

  // Each accepted by alice, one activity of hers saying so; bob, who submitted them, sent their state by her, and
  // Crash on save's priority.
  const acceptance = { verb: "accepted", old_value: "pending", new_value: "accepted", actor: alice.id };
  expect(await Promise.all([crash, slow].map((issue) => intakeRowOf(db, issue.id)))).toEqual([
    { ...live, status: "accepted", written_by: alice.id },
    { ...live, status: "accepted", written_by: alice.id },
  ]);
  expect(await Promise.all([crash, slow].map((issue) => triageActivities(db, issue.id)))).toEqual([
    [acceptance],
    [acceptance],
  ]);
  const state = { sender: "created", by: alice.id, field: "state", verb: "updated" };
  await expect
    .poll(() => Promise.all([crash, slow].map((issue) => notificationRowsOf(db, issue.id, bob.id))), {
      timeout: 10_000,
    })
    .toEqual([
      [
        { ...state, old_value: "Triage", new_value: "Backlog" },
        { ...state, field: "priority", old_value: "none", new_value: "high" },
      ],
      [{ ...state, old_value: "Triage", new_value: "Backlog" }],
    ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H10 (page): alice's acceptance through the modal, open while odo, Web's other admin, accepts the work item, is refused: no update is sent, she is told, the modal closes, and the header, read again, says the work item is accepted and offers no triage, the selection and the focus staying in its detail", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, odo, slug, web } = await intakeWorld(api, testInfo);
  const membership = await projectMembershipOf(api, alice.token, web.id, odo.id);
  expect(await projectMemberWrites(api, web.id).change(alice.token, membership, 20), "odo made Web's admin").toEqual({
    status: 200,
  });
  const [dark, crash] = await submitNamed(api, bob.token, web.id, ["Dark mode", "Crash on save"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([crash, dark].map(rowName));
  await expect(page).toHaveURL(selecting(crash));

  // The modal open, its priority changed; odo accepts Crash on save meanwhile; then her submit: nerve refuses the
  // triage, and no update follows.
  await intakeDetail(page).getByRole("button", { name: "Accept", exact: true }).click();
  const moving = page.getByRole("dialog", { name: `Move ${identifierOf(crash)} to project work items`, exact: true });
  await moving.getByRole("button", { name: "Priority: None", exact: true }).click();
  await page.getByRole("option", { name: "Urgent", exact: true }).click();
  await triageIntakeIssue(api, odo.token, crash.id, { status: "accepted" });
  const updates = await bodiesSentTo(page, "PATCH", `/api/v0/issues/${crash.id}`);
  const sent = recordApi(page);
  const refused = await answerTo(page, "POST", triagePath(crash), () =>
    moving.getByRole("button", { name: "Add to project", exact: true }).click()
  );
  expect([refused.status(), ((await refused.json()) as { code: string }).code]).toEqual([
    409,
    "issue.intake_transition_not_allowed",
  ]);
  await expect(toast(page, "The intake work item cannot move from its status to the one you chose.")).toBeVisible();
  await expect(moving).toHaveCount(0);
  await expect.poll(() => sent().pending).toBe(0);
  expect(updates).toEqual([]);

  // Read again: accepted, nothing to triage; still selected, the focus on its detail; gone from the open tab's list.
  await expectHeaderOf(page, crash);
  await expect(shownStatus(page, "Accepted")).toBeVisible();
  await expect(intakeDetail(page).getByRole("button", { name: /^(Accept|Decline|Options)$/ })).toHaveCount(0);
  await expect(page).toHaveURL(selecting(crash));
  await expect(intakeDetail(page)).toBeFocused();
  await expect.poll(() => shownRows(page)).toEqual([rowName(dark)]);
  expect(await intakeRowOf(db, crash.id)).toEqual({ ...live, status: "accepted", written_by: odo.id });
  expect(await triageActivities(db, crash.id)).toEqual([
    { verb: "accepted", old_value: "pending", new_value: "accepted", actor: odo.id },
  ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 POST ${triagePath(crash)}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
});
