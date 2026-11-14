import {
  cycleJoined,
  cycleLeft,
  cycleMoved,
  expectCycleRows,
  expectPlanningActivity,
  lastWrite,
  wrote,
} from "../../fixtures/assert/issue-planning";
import { watchPage } from "../../fixtures/browser";
import { recordApi } from "../../fixtures/cycle-pages";
import { modalDialog, shownWithin } from "../../fixtures/dialogs";
import { expectListPageQuiet } from "../../fixtures/issue-list-page";
import { bodiesSent, cycleListWorld, rowOptions, shownItems } from "../../fixtures/planning-list-pages";
import { answerTo, expectListBesideButton, moveWithinApp, transitionsEnded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G4's page version (M6 P7a spec 2.4–2.6, 2.11, R2, R4, R5, R14; M2 handoff 5(a), M3 handoff 3): a work item's cycle
// as mem uses it in his browser. Its detail's cycle dropdown, opened by the keyboard beside its button, offers No cycle
// and the cycles it may join, never an ended or an archived one, and sends each pick alone; a row of the project's list
// names the archived cycle its work item is in, the project's archive read once; and a cycle page's new work item
// starts in that cycle, the form's pick winning over the page. The API version is g4-issue-cycle.spec.ts; both assert
// the same rows. The world is made through the API, nerve's answers read back through the database.

/** The day offset days from today, YYYY-MM-DD: a cycle over today is in progress, whatever day the story runs. */
function dayFromToday(offset: number): string {
  return new Date(Date.now() + offset * 86_400_000).toISOString().slice(0, 10);
}

test("G4 (page): on Fix the login's page its cycle, opened by the keyboard beside its button with its search focused, offers No cycle, Next and Now, in nerve's order, neither Done, ended, nor Old, archived; typed Now picks it in one change, Next moves it there and No cycle takes it out, each one change of its cycle alone, and Escape gives the button the focus back; Web's list names Legacy's cycle Old, archived, Web's archive read once; Now's page's new work item starts in Now, is moved to Next in the dialog and is created in Next alone, off Now's list", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await cycleListWorld(api, testInfo);
  // Now is in progress and Next ahead; Done ended, and Old too, which Legacy joined before and which is archived.
  const now = await w.cycle("Now", { start_date: dayFromToday(-7), end_date: dayFromToday(7) });
  const next = await w.cycle("Next", { start_date: "2099-01-01", end_date: "2099-01-14" });
  await w.cycle("Done", { start_date: "2025-02-01", end_date: "2025-02-14" });
  const old = await w.cycle("Old");
  await w.item("Legacy", { cycle_id: old.id });
  await w.end(old.id, true);
  await w.item("Plan", { cycle_id: now.id });
  const fix = await w.item("Fix the login");

  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  const requests = recordApi(page);
  const path = `/api/v0/issues/${fix.id}`;
  const changes = await bodiesSent(page, browserName, "PATCH", path);
  await page.goto(`/${w.slug}/browse/WEB-${fix.sequence_id}`);
  const cycle = (name: string) => page.getByRole("button", { name: `Cycle: ${name}`, exact: true });
  const options = page.getByRole("option");

  // Opened by the keyboard, once the sidebar shows the work item's creator above it (so that nothing moves the button
  // from under its list), the list is beside its button, the one list open, its search focused; it offers No cycle
  // and the cycles Fix the login may join, in nerve's order; typed, it narrows to Now.
  await expect(page.getByText("Created by", { exact: true })).toBeVisible();
  await cycle("No cycle").focus();
  await page.keyboard.press("Enter");
  await expectListBesideButton(page, cycle("No cycle"));
  await expect(page.getByRole("listbox")).toHaveCount(1);
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
  await expect(options).toHaveText(["No cycle", "Next", "Now"]);
  await page.keyboard.type("Now");
  await expect(options).toHaveText(["Now"]);

  // Each pick is one change of the cycle alone, shown once nerve has it: Now, then Next, then No cycle.
  const pick = async (from: string, to: string, sent: number) => {
    if (sent > 0) await cycle(from).click();
    await page.getByRole("option", { name: to, exact: true }).click();
    await expect(cycle(to)).toBeVisible();
    expect(changes, `the changes after ${to}`).toHaveLength(sent + 1);
    return (await lastWrite(db, fix.id)).at;
  };
  const joined = await pick("No cycle", "Now", 0);
  const moved = await pick("Now", "Next", 1);
  const left = await pick("Next", "No cycle", 2);
  expect(changes).toEqual([{ cycle_id: now.id }, { cycle_id: next.id }, { cycle_id: null }]);
  await cycle("No cycle").focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("listbox")).toHaveCount(1);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(cycle("No cycle")).toBeFocused();
  await expectCycleRows(db, fix.id, [
    { target: "Now", added: joined, removed: moved, adder: w.mem.email, by: w.mem.email },
    { target: "Next", added: moved, removed: left, adder: w.mem.email, by: w.mem.email },
  ]);
  await expectPlanningActivity(db, fix.id, [
    ...wrote(w.mem.email, joined, cycleJoined("Now")),
    ...wrote(w.mem.email, moved, cycleMoved("Now", "Next")),
    ...wrote(w.mem.email, left, cycleLeft("Next")),
  ]);

  // Web's list: Legacy's row names its cycle Old, captioned archived, once Web's archive is read, once. A row's control
  // is named by its value, its caption and its row, each a box of its own, which the name parts by a space as well.
  await moveWithinApp(page, w.projectPath);
  await expect(page.getByRole("button", { name: /^Cycle: Old ?, Archived ?, Legacy$/ })).toBeVisible();
  expect(
    requests().requests.filter((request) => request === `GET /api/v0/projects/${w.web.id}/cycles?archived=true`),
    "the reads of Web's archive of cycles"
  ).toHaveLength(1);

  // Now's page's new work item: its dialog's cycle starts as Now; opened there, it offers the same cycles, and Next
  // picked is what the one creation sends; the work item is not Now's, and nerve has it in Next.
  await moveWithinApp(page, w.path(now.id));
  await expect.poll(() => shownItems(page)).toEqual(["Plan"]);
  const created = `/api/v0/projects/${w.web.id}/issues`;
  const creations = await bodiesSent(page, browserName, "POST", created);
  await page.getByRole("button", { name: "Add work item", exact: true }).click();
  const form = modalDialog(page);
  // the dialog's list opens beside its button once the dialog has come in, its description's editor above the fields
  await expect(form.locator('[contenteditable="true"]')).toBeVisible();
  await transitionsEnded(form);
  await form.getByRole("button", { name: "Cycle: Now", exact: true }).click();
  await expectListBesideButton(page, form.getByRole("button", { name: "Cycle: Now", exact: true }));
  await expect(options).toHaveText(["No cycle", "Next", "Now"]);
  await page.getByRole("option", { name: "Next", exact: true }).click();
  await expect(form.getByRole("button", { name: "Cycle: Next", exact: true })).toBeVisible();
  await form.getByRole("textbox", { name: "Title", exact: true }).fill("Made on Now's page");
  const answer = await answerTo(page, "POST", created, () =>
    form.getByRole("button", { name: "Save", exact: true }).click()
  );
  expect(answer.status()).toBe(201);
  const made = (await answer.json()) as { id: string };
  await expect(form).toHaveCount(0);
  expect(creations).toHaveLength(1);
  expect(creations[0]).toMatchObject({ name: "Made on Now's page", cycle_id: next.id });
  expect(await shownWithin(rowOptions(page, "Made on Now's page")), "the new work item on Now's page").toBe(false);
  expect(await shownItems(page)).toEqual(["Plan"]);
  const creation = await lastWrite(db, made.id);
  await expectCycleRows(db, made.id, [
    { target: "Next", added: creation.at, removed: null, adder: w.mem.email, by: w.mem.email },
  ]);
  await expectListPageQuiet(page, watch);
});
