import { changeProject } from "../../fixtures/api";
import { login } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { createIssue } from "../../fixtures/issues";
import { notificationWorld, notificationsPath, shownCards } from "../../fixtures/notification-pages";
import { listNotifications, notificationsSent } from "../../fixtures/notifications";
import { stateIds } from "../../fixtures/states";
import { clockTest as test, expect } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// H2–H4's page version of nerve's own change (M7 design 3.3; P4 spec 2.11, R22, R24; re-93 m10, m13): a work item
// archived by nerve's auto-archive, which no one wrote, is said on its card to be archived by "Nerve", whose picture is
// its letter, never a member's or a deactivated user's. The clock is a process's of its own (clockTest's clockWorld,
// as I16 (page)): advanced by 744 hours, a month and more, the old process's tokens are void, so the receiver signs in
// anew against the new process. The archive is 31 days past the browser's clock, so how long ago is not asserted. This
// file runs clockTest alone (one kind of test to a file); the others of H2–H4 (page) are in
// h2-h4-notification-cards-page.spec.ts.

test("H2–H4 (page): Web archives its completed work items after a month: Old done, bob's, is archived by nerve a month and more later, and bob's notifications page says Nerve archived it, its picture N", async ({
  clockWorld: world,
  browser,
}, testInfo) => {
  // two nerve processes, one after the other, and the auto-archive's periodic job (as I16 (page))
  test.setTimeout(60_000);
  const before = world.current;
  const { alice, bob, slug, web } = await notificationWorld(before.api, testInfo);
  expect((await changeProject(before.api, alice.token, web.id, { archive_in: 1 })).status).toBe(200);
  const { Done } = await stateIds(before.api, alice.token, web.id);
  const old = await createIssue(before.api, alice.token, web.id, {
    name: "Old done",
    state_id: Done,
    assignee_ids: [bob.id],
  });
  await notificationsSent(before.api, bob.token, slug, 2);

  // A month and more later, a new process: bob, signed in anew, is sent nerve's archive of Old done.
  const advanced = await world.advance("744h");
  const tokens = await login(advanced.api, bob.email);
  await expect
    .poll(async () => (await listNotifications(advanced.api, tokens.access_token, slug)).data.length, {
      timeout: 20_000,
      message: "nerve's archive of Old done sent to bob",
    })
    .toBe(3);

  const other = await anotherBrowser(browser, advanced.nerve.baseURL, tokens);
  try {
    const page = other.page;
    const watch = await watchPage(page);
    await page.goto(notificationsPath(slug));
    // Each card but for how long ago it was sent.
    const shown = async () => (await shownCards(page)).map((card) => card.split(" | ").slice(0, -1).join(" | "));
    const at = `WEB-${old.sequence_id} Old done`;
    await expect
      .poll(shown)
      .toEqual([
        `Unread | N | Nerve archived the work item | ${at}`,
        `Unread | A | Alice assigned Bob | ${at}`,
        `Unread | A | Alice created the work item | ${at}`,
      ]);
    await expect(page.getByText("Deactivated user", { exact: true })).toHaveCount(0);
    expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(page, watch);
  } finally {
    await other.close();
  }
});
