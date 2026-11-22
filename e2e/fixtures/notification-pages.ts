import { expect, type Locator, type Page, type TestInfo } from "@playwright/test";

import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor, type Api } from "./api";
import { accountId, emailFor, login, named, type AuthTokens } from "./auth";
import { registerOnboarded } from "./settings-pages";

// The notifications page and the subscription's places (M7 P4 spec 2.6, 2.11), as a person uses them: their
// addresses, what a story finds on them, and the world of their stories. The helpers they share with the cycle pages'
// stories (toast, dismissToasts, recordApi, failedLoads, seenBy) are imported from cycle-pages.ts, not copied.

/**
 * One of the world's people: his address, his id, the tokens of the sign-in his first browser takes, and the access
 * token of another sign-in of his, for the API calls of the story, which the browser's refreshes leave alone.
 */
interface Person {
  email: string;
  id: string;
  tokens: AuthTokens;
  token: string;
}

/** Signs label up for this run of the test, onboarded and named name, and gives the person. */
export async function person(api: Api, testInfo: TestInfo, label: string, name: string): Promise<Person> {
  const email = emailFor(testInfo, label);
  const tokens = await registerOnboarded(api, email);
  const token = (await login(api, email)).access_token;
  await named(api, token, { display_name: name });
  return { email, id: await accountId(api, token), tokens, token };
}

/**
 * The world of the notification pages' stories: alice's workspace Acme and its project Web, secret (network 0: who
 * leaves it no longer sees it, and nerve answers what he asks of it as not found), of which bob and walt are members
 * too; each named, alice "Alice", bob "Bob", walt "Walt".
 */
export async function notificationWorld(api: Api, testInfo: TestInfo) {
  const [alice, bob, walt] = await Promise.all([
    person(api, testInfo, "alice", "Alice"),
    person(api, testInfo, "bob", "Bob"),
    person(api, testInfo, "walt", "Walt"),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, { email: bob.email, token: bob.token }, 15);
  await inviteAndAccept(api, alice.token, slug, { email: walt.email, token: walt.token }, 15);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB", network: 0 });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: walt.id, role: 15 },
  ]);
  return { alice, bob, walt, slug, web };
}

/** The address of the notifications page of the workspace of slug. */
export function notificationsPath(slug: string): string {
  return `/${slug}/notifications`;
}

/** The address of the detail page of the work item <identifier>-<sequence> of the workspace of slug. */
export function detailPath(slug: string, identifier: string, sequence: number): string {
  return `/${slug}/browse/${identifier}-${sequence}`;
}

/** The cards of the notifications page's list, in its order. */
function cardItems(page: Page): Locator {
  return page.locator("li[data-notification-id]");
}

/**
 * What each card of the notifications page's list says, in the list's order, as its main button shows it: the text of
 * each of its lines, the unread dot's words and the picture's letter among them, apart by " | ".
 */
export async function shownCards(page: Page): Promise<string[]> {
  return cardItems(page).evaluateAll((cards) =>
    cards.map((card) => {
      const main = card.querySelector("button");
      return (main?.innerText ?? "")
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line !== "")
        .join(" | ");
    })
  );
}

/** The card of the notifications page's list whose main button shows text. */
export function cardSaying(page: Page, text: string): Locator {
  return cardItems(page).filter({ has: page.getByRole("button").filter({ hasText: text }) });
}

/**
 * The notifications page's preview, beside its sidebar: the layout's outlet, the sidebar's next sibling (the sidebar
 * is the one of a quarter of the page's width from md up).
 */
function previewPane(page: Page): Locator {
  return page.locator('div[class*="md:w-3/12"] + div');
}

/**
 * Checks that the notifications page's preview shows that nothing is selected: its picture alone, no work item, no
 * control and no wait (never a blank pane, re-93 I4).
 */
export async function expectNothingSelected(page: Page): Promise<void> {
  const pane = previewPane(page);
  await expect(pane.locator("svg")).toBeVisible();
  await expect(pane.getByRole("button")).toHaveCount(0);
  await expect(pane.getByRole("status")).toHaveCount(0);
  await expect(pane).toHaveText("");
}
