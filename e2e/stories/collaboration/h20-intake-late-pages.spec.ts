import type { Page } from "@playwright/test";

import { changeProject, createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { intakeRowOf } from "../../fixtures/assert/intake";
import { anotherTabSignsIn } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { recordApi } from "../../fixtures/cycle-pages";
import { shownWithin } from "../../fixtures/dialogs";
import { expectLanded, focusedName } from "../../fixtures/intake-focus";
import {
  expectHeaderOf,
  intakeDetail,
  intakeEntry,
  intakeListRead,
  intakeListReads,
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
import { holdAnswer, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H20's page version for the intake page's late answers (M7 design 2, H20; P5a spec 2.6, 2.8, R14, R35; v0 design
// 7.7): what the page sent belongs to its selection, its project, its workspace and its session. A decline answered
// after Back or Forward changed the selection moves the selection nowhere (R14) and says nothing; its row, which had
// the focus back as its modal went, leaves the list and gives the focus to the next row, never to the page's body
// (R35). A decline answered after a move to another project's intake, and a workspace's counts and list answered
// after a move to another workspace's, change and say nothing there; back, they are read again. Answers late for a
// session another tab ended change and say nothing on the other account's page. Every request is /api/v0's. The keys
// and names are h20-intake-keys-pages.spec.ts's; the focus as a triage ends, h20-intake-pages.spec.ts's.

/** The decline modal of the page's detail, and the header's button that opens it. */
const declining = (page: Page) => page.getByRole("dialog", { name: "Decline work item", exact: true });
const declineButton = (page: Page) => intakeDetail(page).getByRole("button", { name: "Decline", exact: true });

/** A toast on page: the page said something. */
const anyToast = (page: Page) => page.getByLabel("Notifications", { exact: true }).getByRole("dialog");

/** How a declined intake row reads: declined by who, neither snoozed nor a duplicate. */
const declinedBy = (id: string) => ({ status: "declined", snoozed_till: null, duplicate_to_id: null, written_by: id });

test("H20 (page): alice goes Back while a decline is out, by the keyboard and by the pointer: the selection stays where Back put it once nerve answers, and the declined row, which had the focus back, leaves the list giving it to the next row, never to the page's body; Forward finds the declined one, its header saying so", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [a1, a2, a3, a4, a5] = await submitNamed(api, bob.token, web.id, ["A1", "A2", "A3", "A4", "A5"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([a5, a4, a3, a2, a1].map(rowName));
  await expect(page).toHaveURL(selecting(a5));

  // By the keyboard: A4 chosen by Enter on its row (a push), declined by Enter, nerve's answer held. Back: A5 selected,
  // the modal gone with A4's detail, A4's row given the focus back. Let through: A4 leaves the list, A5 still selected,
  // the focus on A3's row.
  await intakeRow(page, a4).focus();
  await page.keyboard.press("Enter");
  await expectHeaderOf(page, a4);
  await declineButton(page).focus();
  await page.keyboard.press("Enter");
  const byKeyboard = await sentHeld(page, "POST", triagePath(a4), async () => {
    await declining(page).getByRole("button", { name: "Decline", exact: true }).focus();
    await page.keyboard.press("Enter");
  });
  await page.goBack();
  await expect(page).toHaveURL(selecting(a5));
  await expect(declining(page)).toHaveCount(0);
  await expect(intakeRow(page, a4)).toBeFocused();
  expect((await byKeyboard.release()).status()).toBe(200);
  await expect.poll(() => shownRows(page)).toEqual([a5, a3, a2, a1].map(rowName));
  expect(await shownWithin(anyToast(page))).toBe(false);
  await expect.poll(() => focusedName(page)).toBe(`link ${rowName(a3)}`);
  await expectLanded(intakeRow(page, a3));
  await expect(page).toHaveURL(selecting(a5));

  // Forward: A4, declined, its header saying so and offering no triage.
  await page.goForward();
  await expect(page).toHaveURL(selecting(a4));
  await expect(shownStatus(page, "Declined")).toBeVisible();
  await expect(intakeDetail(page).getByRole("button", { name: /^(Accept|Decline|Options)$/ })).toHaveCount(0);

  // By the pointer: A2 pressed and declined, held; Back to A4. Let through: A2 leaves, A4 still selected; the focus on
  // A1's row, the next, where the row had it (WebKit's press of a link does not focus it: the detail kept the focus).
  await intakeRow(page, a2).click();
  await expectHeaderOf(page, a2);
  await declineButton(page).click();
  const byPointer = await sentHeld(page, "POST", triagePath(a2), () =>
    declining(page).getByRole("button", { name: "Decline", exact: true }).click()
  );
  await page.goBack();
  await expect(page).toHaveURL(selecting(a4));
  expect((await byPointer.release()).status()).toBe(200);
  await expect.poll(() => shownRows(page)).toEqual([a5, a3, a1].map(rowName));
  const [name, landing] =
    browserName === "webkit"
      ? ["region Intake work item", intakeDetail(page)]
      : [`link ${rowName(a1)}`, intakeRow(page, a1)];
  await expect.poll(() => focusedName(page)).toBe(name);
  await expectLanded(landing);
  await expect(page).toHaveURL(selecting(a4));
  expect(await Promise.all([a4, a2].map((issue) => intakeRowOf(db, issue.id)))).toEqual([
    declinedBy(alice.id),
    declinedBy(alice.id),
  ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice's decline of a Web work item, answered after a move to Ops' intake, moves nothing there; back on Web, its list is read again; Acme's counts and Web's closed list, answered after a move to Beta's intake, change nothing of Beta's list, tab or count; back on Web, its closed list is read again", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [w1, w2, w3] = await submitNamed(api, bob.token, web.id, ["W1", "W2", "W3"]);
  const ops = await createProject(api, alice.token, slug, { name: "Ops", identifier: "OPS" });
  expect((await changeProject(api, alice.token, ops.id, { intake_view: true })).status).toBe(200);
  const [o1] = await submitNamed(api, alice.token, ops.id, ["O1"]);
  const beta = await createWorkspace(api, alice.token, { name: "Beta", slug: slugFor(testInfo, "beta") });
  const core = await createProject(api, alice.token, beta.slug, { name: "Core", identifier: "CORE" });
  expect((await changeProject(api, alice.token, core.id, { intake_view: true })).status).toBe(200);
  const [c1] = await submitNamed(api, alice.token, core.id, ["C1"]);
  const counts = `/api/v0/workspaces/${slug}/intake-counts`;

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const answered = (path: string) =>
    page.waitForResponse((r) => r.request().method() === "GET" && new URL(r.url()).pathname === path);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([w3, w2, w1].map(rowName));
  await expect(page).toHaveURL(selecting(w3));

  // W3 declined, nerve's answer held; a move to Ops' intake, which lists and selects O1. Let through, its read again
  // and Acme's counts in its turn: Ops' selection, list and tab as they were.
  await declineButton(page).click();
  const held = await sentHeld(page, "POST", triagePath(w3), () =>
    declining(page).getByRole("button", { name: "Decline", exact: true }).click()
  );
  await moveWithinApp(page, intakePath(slug, ops.id));
  const opsRows = [`OPS-${o1.sequence_id} O1`];
  await expect.poll(() => shownRows(page)).toEqual(opsRows);
  await expect(page).toHaveURL(selecting(o1));
  const counted = answered(counts);
  expect((await held.release()).status()).toBe(200);
  await counted;
  expect(await shownWithin(anyToast(page))).toBe(false);
  expect(await shownRows(page)).toEqual(opsRows);
  await expect(page).toHaveURL(selecting(o1));
  await expect(page.getByRole("tab", { name: "Open, 1 work items", exact: true })).toHaveAttribute(
    "aria-selected",
    "true"
  );

  // Back on Web: its list read again, without W3; W3 still selected, declined.
  const reads = recordApi(page);
  await page.goBack();
  await expect.poll(() => shownRows(page)).toEqual([w2, w1].map(rowName));
  expect(intakeListReads(reads().requests)).toContain(
    intakeListRead(web.id, "status=pending&order_by=-created_at&limit=50")
  );
  await expect(page).toHaveURL(selecting(w3));
  await expect(shownStatus(page, "Declined")).toBeVisible();

  // Acme's counts held after W2's decline, then Web's closed list, held; a move to Beta's intake, which lists and
  // selects Core's C1, its tab and its count its own. Let through: Beta's as they were.
  const releaseCounts = await holdAnswer(page, "GET", counts);
  const releaseList = await holdAnswer(page, "GET", `/api/v0/projects/${web.id}/intake-issues`);
  await intakeRow(page, w2).click();
  await expectHeaderOf(page, w2);
  await declineButton(page).click();
  const sent = recordApi(page);
  await declining(page).getByRole("button", { name: "Decline", exact: true }).click();
  await expect(intakeRow(page, w1)).toBeFocused();
  await page.getByRole("tab", { name: "Closed", exact: true }).click();
  await expect
    .poll(() => sent().requests)
    .toEqual(
      expect.arrayContaining([
        `GET ${counts}`,
        intakeListRead(web.id, "status=accepted,declined,duplicate&order_by=-created_at&limit=50"),
      ])
    );
  await moveWithinApp(page, intakePath(beta.slug, core.id));
  const coreRows = [`CORE-${c1.sequence_id} C1`];
  const coreTab = page.getByRole("tab", { name: "Open, 1 work items", exact: true });
  await expect.poll(() => shownRows(page)).toEqual(coreRows);
  await expect(page).toHaveURL(selecting(c1));
  await expect(coreTab).toHaveAttribute("aria-selected", "true");
  await expect(intakeEntry(page, beta.slug, core.id, 1)).toBeVisible();
  const late = Promise.all([answered(counts), answered(`/api/v0/projects/${web.id}/intake-issues`)]);
  await releaseCounts();
  await releaseList();
  await late;
  expect(await shownWithin(anyToast(page))).toBe(false);
  expect(await shownRows(page)).toEqual(coreRows);
  await expect(page).toHaveURL(selecting(c1));
  await expect(coreTab).toHaveAttribute("aria-selected", "true");
  await expect(intakeEntry(page, beta.slug, core.id, 1)).toBeVisible();

  // Back on Web's closed tab: its list read again, W3 and W2 declined; Web's count, one.
  const again = recordApi(page);
  await page.goBack();
  await expect.poll(() => shownRows(page)).toEqual([w3, w2].map(rowName));
  expect(intakeListReads(again().requests)).toContain(
    intakeListRead(web.id, "status=accepted,declined,duplicate&order_by=-created_at&limit=50")
  );
  await expect(intakeEntry(page, slug, web.id, 1)).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice's count read and decline, answered after another tab signs bob in, say and change nothing on his page: his list, his count and the selection stay as he reads them", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [x1, x2, x3, x4] = await submitNamed(api, bob.token, web.id, ["X1", "X2", "X3", "X4"]);
  const counts = `/api/v0/workspaces/${slug}/intake-counts`;

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([x4, x3, x2, x1].map(rowName));
  await expect(intakeEntry(page, slug, web.id, 4)).toBeVisible();
  // X4's detail drawn before any answer is held: a route set while the page still loads its detail's code makes WebKit
  // fetch that code again, its preloads unused, and warn (T12's lesson; t13-webkit-preload-red, under load)
  await expectHeaderOf(page, x4);

  // X4 declined: Acme's counts read again in its turn, held (three pending then); X3 declined, nerve's answer held.
  const releaseCounts = await holdAnswer(page, "GET", counts);
  const sent = recordApi(page);
  await declineButton(page).click();
  await declining(page).getByRole("button", { name: "Decline", exact: true }).click();
  await expect(intakeRow(page, x3)).toBeFocused();
  await expect.poll(() => sent().requests).toEqual(expect.arrayContaining([`GET ${counts}`]));
  await expectHeaderOf(page, x3);
  await declineButton(page).click();
  const held = await sentHeld(page, "POST", triagePath(x3), () =>
    declining(page).getByRole("button", { name: "Decline", exact: true }).click()
  );

  // Another tab signs bob in, whose page this becomes: X3 still selected, declined; his list X2 and X1; his count two.
  await (await anotherTabSignsIn(context, api, bob.email)).close();
  await expect.poll(() => shownRows(page)).toEqual([x2, x1].map(rowName));
  await expect(page).toHaveURL(selecting(x3));
  await expect(shownStatus(page, "Declined")).toBeVisible();
  await expect(intakeEntry(page, slug, web.id, 2)).toBeVisible();

  // alice's answers let through: on his page, nothing of hers: the selection, the list and the count his.
  const late = page.waitForResponse((r) => new URL(r.url()).pathname === counts);
  await releaseCounts();
  await late;
  expect((await held.release()).status()).toBe(200);
  expect(await shownWithin(anyToast(page))).toBe(false);
  expect(await shownWithin(intakeEntry(page, slug, web.id, 3))).toBe(false);
  await expect(intakeEntry(page, slug, web.id, 2)).toBeVisible();
  await expect(page).toHaveURL(selecting(x3));
  expect(await shownRows(page)).toEqual([x2, x1].map(rowName));
  expect(await Promise.all([x4, x3].map((issue) => intakeRowOf(db, issue.id)))).toEqual([
    declinedBy(alice.id),
    declinedBy(alice.id),
  ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
