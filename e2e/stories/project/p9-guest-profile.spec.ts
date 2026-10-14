import { createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { accountId, emailFor, named } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectNoAddress, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// P9, a guest on a member's profile (M3 design 2, decision point 3, A: Plane's page): the card, and of the work items
// only that he may not see them, on anyone's profile, his own too. Page only: the work items' list is M4's, which tests
// its rule (13.2); the card's data is the members' list, which W11's API version covers. S2 holds what the page loads.

test("P9 (page): a guest opens a member's profile: its card, without the address, and in place of the work items that he may not see them, which the page does not ask for; his own profile the same", async ({
  api,
  signedInPage,
}, testInfo) => {
  const adaEmail = emailFor(testInfo, "ada");
  const gusEmail = emailFor(testInfo, "gus");
  const [ada, gus] = await Promise.all([registerOnboarded(api, adaEmail), registerOnboarded(api, gusEmail)]);
  await named(api, ada.access_token, { first_name: "Ada", last_name: "Lovelace", display_name: "ada" });
  await named(api, gus.access_token, { first_name: "Gus", last_name: "Grant", display_name: "gus" });
  const slug = slugFor(testInfo);
  await createWorkspace(api, ada.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, ada.access_token, slug, { email: gusEmail, token: gus.access_token }, 5);
  const [adaId, gusId] = await Promise.all([accountId(api, ada.access_token), accountId(api, gus.access_token)]);

  const page = await signedInPage(gus);
  const watch = await watchPage(page);
  /** Opens the profile of id, which goes to its first tab, and checks what it shows of him named name. */
  const profileOf = async (id: string, name: string, display: string) => {
    await page.goto(`/${slug}/profile/${id}`);
    await expect(page).toHaveURL(`/${slug}/profile/${id}/assigned`);
    await expect(page.getByRole("heading", { level: 4, name }), name).toBeVisible();
    await expect(page.getByText(`(${display})`, { exact: true }), name).toBeVisible();
    await expect(page.getByText("You do not have the permission to access this page."), name).toBeVisible();
    // no tab of the work items: the assigned, the created, the subscribed
    await expect(page.getByRole("link", { name: /^(Assigned|Created|Subscribed)$/ }), name).toHaveCount(0);
    await expectNoAddress(page, [adaEmail, gusEmail], `${name}'s profile`);
  };
  await profileOf(adaId, "Ada Lovelace", "ada");
  await profileOf(gusId, "Gus Grant", "gus");
  // Nothing asked of the work items, by nerve's API or by the old frontend's.
  expect(watch.apiRequests.filter((request) => request.includes("issues"))).toEqual([]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});
