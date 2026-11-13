import { expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  archivedCyclesPath,
  chooseFromMenu,
  cyclePagesWorld,
  cycleSection,
  cyclesPath,
  dismissToasts,
  failedLoads,
  itemMenu,
  linkedCycles,
  menuEntries,
  openCyclesList,
  recordApi,
  sectionToggle,
  seenBy,
  toast,
  turnCycles,
} from "../../fixtures/cycle-pages";
import { createCycle } from "../../fixtures/cycles";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  shownWithin,
} from "../../fixtures/dialogs";
import { APP, WORKSPACE, valued } from "../../fixtures/mounts";
import { answerTo, moveWithinApp, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G8's page version (M6 P5a spec 2.5, 2.9, R11, R12): a project's ended cycles archived from its list, its archive of
// cycles, a cycle's preview there, and its restore, by its member in his browser; and the archive while it holds no
// cycle, which S2 covered until the archive read nerve's API (P5a T9). The API version is g8-archive-cycle.spec.ts;
// both assert the same rows.

/** rows with the cycle of name changed as to says. */
function changed(rows: CycleRow[], name: string, to: Partial<CycleRow>): CycleRow[] {
  return rows.map((c) => (c.name === name ? { ...c, ...to } : c));
}

test("G8 (page): Web's archive of cycles, empty, says so at once and asks nerve's API alone; of Q1 and Q2, ended, Now, current, and Later, a draft, mem may archive neither Now nor Later; Q1's archive waits for nerve and goes, said once its row has gone; Q2's, refused as Web's cycles were turned off meanwhile, stays, saying why and never success, then goes; the archive lists Q2 then Q1, Q1 previewed as nerve reads it; Q1's restore is said once nerve has restored it, Q2's refused says why; Web's guest and a workspace admin who is its guest may archive and restore none", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // mem's page, then two other browsers, each signed in anew
  test.setTimeout(60_000);
  const { admin, mem, gus, wanda, slug, web, token } = await cyclePagesWorld(api, testInfo);
  // Q1 and Q2 ended in 2025; Now runs from 2026 to the end of 2098; Later has no days.
  const q1 = await createCycle(api, token, web.id, { name: "Q1", start_date: "2025-01-01", end_date: "2025-03-31" });
  const q2 = await createCycle(api, token, web.id, { name: "Q2", start_date: "2025-04-01", end_date: "2025-06-30" });
  const now = await createCycle(api, token, web.id, { name: "Now", start_date: "2026-01-01", end_date: "2098-12-31" });
  await createCycle(api, token, web.id, { name: "Later" });
  const row = (name: string, start: string | null, end: string | null, sort_order: number): CycleRow => ({
    name,
    description: "",
    start,
    end,
    sort_order,
    archived: false,
    deleted: false,
    owner: admin.email,
    by: admin.email,
  });
  let rows = [
    row("Q1", "2025-01-01", "2025-03-31", 65535),
    row("Q2", "2025-04-01", "2025-06-30", 55535),
    row("Now", "2026-01-01", "2098-12-31", 45535),
    row("Later", null, null, 35535),
  ];
  await expectCycles(db, web.id, rows);

  // Web's archive of cycles holds none: mem's page says so as soon as nerve has answered, never a placeholder that
  // waits for good. It asks nerve's API alone, for what a workspace's page loads and the archive (S2's archived cycles
  // entry, M6 P5a T9): the archive is not one of the project's pages that read the project and its own resources.
  const page = await signedInPage(mem.tokens);
  const watch = await watchPage(page);
  const recorded = recordApi(page);
  const archive = archivedCyclesPath(slug, web.id);
  await page.goto(archive);
  await expect(page.getByText("No archived cycles yet", { exact: true })).toBeVisible();
  await expect(page.getByRole("main").last().locator(".animate-pulse")).toHaveCount(0);
  const expected = [...APP, ...WORKSPACE, `GET /api/v0/projects/{project}/cycles?archived=true`].map((request) =>
    valued(request, { [slug]: "{slug}", [web.id]: "{project}" })
  );
  await expect.poll(recorded).toEqual({ pending: 0, requests: expected.toSorted() });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);

  // On Web's list, opened from the sidebar, Now, in progress, and Later, a draft, offer their archive disabled, with
  // the reason; Q1 its archive.
  await openCyclesList(page, slug, web.id);
  await expect.poll(() => linkedCycles(cycleSection(page, "Active cycle"))).toEqual([now.id]);
  await sectionToggle(page, "Completed cycle").click();
  await expect.poll(() => linkedCycles(cycleSection(page, "Completed cycle"))).toEqual([q2.id, q1.id]);
  const reasoned = page.getByRole("menu").getByRole("menuitem", { name: /^Archive/ });
  const archiveOffered = async (name: string) => {
    await itemMenu(page, name).click();
    const offered = [(await reasoned.isDisabled()) ? "disabled" : "enabled", (await reasoned.textContent()) ?? ""];
    await page.keyboard.press("Escape");
    await expect(reasoned).toHaveCount(0);
    return offered;
  };
  expect([await archiveOffered("Now"), await archiveOffered("Later"), await archiveOffered("Q1")]).toEqual([
    ["disabled", "ArchiveOnly completed cycles can be archived"],
    ["disabled", "ArchiveOnly completed cycles can be archived"],
    ["enabled", "Archive"],
  ]);

  // Q1's archive, named by it, waits for nerve: Escape does not close it, Cancel is disabled. Once nerve has archived
  // Q1, the dialog closes and says so, after Q1's row has gone.
  await chooseFromMenu(page, "Q1", "Archive");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Archive cycle Q1");
  const confirm = dialog.getByRole("button", { name: "Archive", exact: true });
  const q1Archive = `/api/v0/cycles/${q1.id}/archive`;
  const archiving = await sentHeld(page, "POST", q1Archive, () => confirm.click());
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
  expect((await archiving.release()).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "Your archives can be found in project archives.")).toBeVisible();
  await expect.poll(() => linkedCycles(cycleSection(page, "Completed cycle"))).toEqual([q2.id]);
  rows = changed(rows, "Q1", { archived: true, by: mem.email });
  await expectCycles(db, web.id, rows);

  // Q2's archive asked for, Web's cycles turned off meanwhile: nerve refuses it, said beside the dialog, which stays and
  // says no success; Cancel closes it. Turned on again, Q2 is archived.
  await dismissToasts(page);
  await chooseFromMenu(page, "Q2", "Archive");
  await expect(dialog).toHaveAccessibleName("Archive cycle Q2");
  await turnCycles(api, token, web, false);
  const q2Archive = `/api/v0/cycles/${q2.id}/archive`;
  expect((await sentTo(page, "POST", q2Archive, () => confirm.click())).answer.status()).toBe(409);
  const refusal = toast(page, "Cycles are turned off in this project. Turn them on to change them.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  expect(await shownWithin(toast(page, "Your archives can be found in project archives."))).toBe(false);
  await expectCycles(db, web.id, rows);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await closedWithin(page)).toBe(true);
  await turnCycles(api, token, web, true);
  await chooseFromMenu(page, "Q2", "Archive");
  expect((await answerTo(page, "POST", q2Archive, () => confirm.click())).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  rows = changed(rows, "Q2", { archived: true, by: mem.email });
  await expectCycles(db, web.id, rows);

  // The archive lists Q2, archived last, then Q1. Q1's row previews it, read as nerve has it, ended; closed, the
  // address has no preview again.
  await dismissToasts(page);
  await page.goto(archive);
  const listedThere = () => linkedCycles(page.getByRole("main").last());
  await expect.poll(listedThere).toEqual([q2.id, q1.id]);
  await page.locator(`a[href$="/cycles/${q1.id}"]`).click();
  await expect(page).toHaveURL(`${archive}?peekCycle=${q1.id}`);
  const previewed = page.getByRole("heading", { name: "Q1", exact: true });
  await expect(previewed).toBeVisible();
  await expect(previewed.locator("xpath=..")).toContainText("Completed");
  await expect(page.getByRole("button", { name: "Date range", exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Close", exact: true }).click();
  await expect(page).toHaveURL(archive);

  // Q1 restored from its menu: said once nerve has restored it, and its row goes from the archive.
  const q1Restore = `/api/v0/cycles/${q1.id}/unarchive`;
  await itemMenu(page, "Q1").click();
  const restoring = await sentHeld(page, "POST", q1Restore, () =>
    page.getByRole("menuitem", { name: "Restore", exact: true }).click()
  );
  expect(await shownWithin(toast(page, "The cycle has been restored."))).toBe(false);
  expect((await restoring.release()).status()).toBe(200);
  await expect(toast(page, "The cycle has been restored.")).toBeVisible();
  await expect(page).toHaveURL(archive);
  await expect.poll(listedThere).toEqual([q2.id]);
  rows = changed(rows, "Q1", { archived: false, by: mem.email });
  await expectCycles(db, web.id, rows);

  // Q2's restore, Web's cycles turned off meanwhile: nerve refuses it, said, never success; Q2 stays archived.
  await dismissToasts(page);
  await turnCycles(api, token, web, false);
  const q2Restore = `/api/v0/cycles/${q2.id}/unarchive`;
  await itemMenu(page, "Q2").click();
  const refused = await sentTo(page, "POST", q2Restore, () =>
    page.getByRole("menuitem", { name: "Restore", exact: true }).click()
  );
  expect(refused.answer.status()).toBe(409);
  await expect(refusal).toBeVisible();
  expect(await shownWithin(toast(page, "The cycle has been restored."))).toBe(false);
  await expect.poll(listedThere).toEqual([q2.id]);
  await expectCycles(db, web.id, rows);
  await turnCycles(api, token, web, true);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 POST ${q2Archive}`, `409 POST ${q2Restore}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409, 409) });

  // Web's guest, and a workspace admin who is its guest, see the list and the archive and may archive or restore none.
  // Their sidebars have no link to the cycles (a guest's has none, as Plane's): the list is loaded by its address, and
  // the archive reached within the app.
  const neither = async (other: typeof page) => {
    await sectionToggle(other, "Completed cycle").click();
    await expect.poll(() => linkedCycles(cycleSection(other, "Completed cycle"))).toEqual([q1.id]);
    expect(await menuEntries(other, "Q1")).toEqual(["Open in new tab", "Copy link"]);
    await moveWithinApp(other, archive);
    await expect.poll(() => linkedCycles(other.getByRole("main").last())).toEqual([q2.id]);
    expect(await menuEntries(other, "Q2")).toEqual(["Open in new tab", "Copy link"]);
  };
  await seenBy(browser, baseURL ?? "", api, gus.email, cyclesPath(slug, web.id), neither);
  await seenBy(browser, baseURL ?? "", api, wanda.email, cyclesPath(slug, web.id), neither);
});
