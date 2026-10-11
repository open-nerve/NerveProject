import type { Page, Request, Response, TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  archiveProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Project,
} from "../../fixtures/api";
import { accountId, emailFor, register, type AuthTokens } from "../../fixtures/auth";
import { signInPath } from "../../fixtures/auth-pages";
import {
  EMOJI_CHECK_WARNING,
  expectQuietConsole,
  requestsElsewhere,
  watchPage,
  type PageWatch,
} from "../../fixtures/browser";
import { APP, ARCHIVED, GENERAL, INVITATIONS, PROJECT, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

/** A page of the frontend's router, not a file: nerve answers it with index.html. */
const deepLink = "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues";

interface Visit {
  document: Response;
  watch: PageWatch;
  /** Static resources that loaded, as "<status> <url>". */
  loaded: string[];
  /** Static resources that failed: an HTTP error, or refused or aborted by the browser. */
  failed: string[];
  /** Requests to any origin other than nerve's; the frontend is served same-origin. */
  elsewhere: string[];
}

function isStatic(url: string): boolean {
  return !new URL(url).pathname.startsWith("/api/");
}

/**
 * Opens path, signed out, and waits until the app has started: nerve has answered the one request the app
 * makes as it starts, for the instance's settings, and the sign-in page shows. Not networkidle: a request
 * whose answer the page never reads keeps the network busy, and the wait would last until the test's
 * timeout instead of failing on what the page sent.
 */
async function open(page: Page, path: string): Promise<Visit> {
  const watch = await watchPage(page);
  const loaded: string[] = [];
  const failed: string[] = [];
  page.on("response", (res) => {
    if (isStatic(res.url())) {
      (res.status() < 400 ? loaded : failed).push(`${res.status()} ${res.url()}`);
    }
  });
  page.on("requestfailed", (req) => {
    if (isStatic(req.url())) {
      failed.push(`${req.failure()?.errorText} ${req.url()}`);
    }
  });
  const [document] = await Promise.all([
    page.goto(path),
    page.waitForResponse((res) => new URL(res.url()).pathname === "/api/v0/instance", { timeout: 10_000 }),
  ]);
  if (!document) {
    throw new Error(`no document response for ${path}`);
  }
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  return { document, watch, loaded, failed, elsewhere: requestsElsewhere(page, watch) };
}

/**
 * Signed out, the app asks nerve for the instance's settings only: without a refresh token it neither
 * refreshes nor asks for /me (M2 design 7.1), so no API call fails; the page's Content-Security-Policy
 * blocks nothing of it (8.3); and its console has no error, and no warning but thirdPartyWarnings.
 */
async function expectQuietSignedOut(
  page: Page,
  watch: PageWatch,
  thirdPartyWarnings: readonly string[] = []
): Promise<void> {
  expect(watch.apiRequests).toEqual(["GET /api/v0/instance"]);
  expect(watch.apiFailures).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, { warnings: thirdPartyWarnings });
}

test("S2: a user opens the home page in a browser", async ({ page }) => {
  const { document, watch, loaded, failed, elsewhere } = await open(page, "/");

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(document.headers()["content-security-policy"]).toMatch(/^default-src 'self'; script-src 'self' 'sha256-/);
  expect(loaded).toContainEqual(expect.stringMatching(/^200 .*\/assets\/[^/]+\.js$/));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
  await expectQuietSignedOut(page, watch);
});

test("S2: a user opens a deep link directly", async ({ page, request }) => {
  const { document, watch, failed, elsewhere } = await open(page, deepLink);

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(await document.body()).toEqual(await (await request.get("/")).body());
  // Signed out, the page behind the sign-in goes to the sign-in page, which comes back to it.
  await expect(page).toHaveURL(signInPath(deepLink));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
  await expectQuietSignedOut(page, watch, [EMOJI_CHECK_WARNING]);
});

/** The accounts of the workspace S2 signs in to: its admin; a member and a guest of it in its project too; a member of it who is not. */
const ACCOUNTS = ["admin", "member", "guest", "project non-member"] as const;
type Account = (typeof ACCOUNTS)[number];

/**
 * The pages each account opens, in order, each as its path ({slug} and {project} for their values) and the requests it
 * makes as it loads: "/", which lands on the workspace's home; the workspace's projects page, whose call of its own
 * fetch no other check holds; the workspace's general settings and its members; then the project's settings.
 */
const REQUESTS: Record<Account, [path: string, requests: string[]][]> = {
  admin: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE, ...INVITATIONS]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
  ],
  member: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
  ],
  guest: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
  ],
  "project non-member": [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT]],
  ],
};

/**
 * Makes S2's workspace through the API: its admin creates it and a public project; a member and a guest of the
 * workspace join the project with the same roles; another member of the workspace does not. Returns the tokens of
 * each account, which have not been used in a browser yet.
 */
async function acme(
  api: Api,
  testInfo: TestInfo
): Promise<{ slug: string; project: Project; tokens: Record<Account, AuthTokens> }> {
  const signUp = async (label: string) => {
    const email = emailFor(testInfo, label);
    return { email, tokens: await registerOnboarded(api, email) };
  };
  const admin = await signUp("admin");
  const member = await signUp("member");
  const guest = await signUp("guest");
  const outsider = await signUp("outsider");
  const token = admin.tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const project = await createProject(api, token, slug, { name: "Web", identifier: "WEB", network: 2 });
  const join = (who: { email: string; tokens: AuthTokens }, role: 5 | 15) =>
    inviteAndAccept(api, token, slug, { email: who.email, token: who.tokens.access_token }, role);
  await join(member, 15);
  await join(guest, 5);
  await join(outsider, 15);
  await addProjectMembers(api, token, project.id, [
    { member_id: await accountId(api, member.tokens.access_token), role: 15 },
    { member_id: await accountId(api, guest.tokens.access_token), role: 5 },
  ]);
  const tokens = {
    admin: admin.tokens,
    member: member.tokens,
    guest: guest.tokens,
    "project non-member": outsider.tokens,
  };
  return { slug, project, tokens };
}

/** An API request's path and query, or undefined for a request of anything else. */
function apiPath(request: Request): string | undefined {
  const { pathname, search } = new URL(request.url());
  return pathname.startsWith("/api/") ? `${pathname}${search}` : undefined;
}

/**
 * Follows the API requests of page from now on, each as "<method> <path>" (with its query) with the names of names in
 * place of their values. between(from, to) gives the requests from the index from to the index to (the end when it is
 * left out), sorted, and how many of all the requests so far have not ended (answered or failed): a test polls for
 * none pending and its list, so that a wait that times out prints the difference; next() is the index of the next
 * request.
 */
function followRequests(page: Page, names: Record<string, string>) {
  const requests: string[] = [];
  let ended = 0;
  page.on("request", (request) => {
    const path = apiPath(request);
    if (path !== undefined) {
      const named = Object.entries(names).reduce((shown, [value, name]) => shown.replaceAll(value, name), path);
      requests.push(`${request.method()} ${named}`);
    }
  });
  const end = (request: Request) => {
    if (apiPath(request) !== undefined) {
      ended += 1;
    }
  };
  page.on("requestfinished", end);
  page.on("requestfailed", end);
  return {
    next: () => requests.length,
    between: (from: number, to?: number) => ({
      pending: requests.length - ended,
      requests: requests.slice(from, to).toSorted(),
    }),
  };
}

/**
 * The end of a story that loads one page: waits until the requests followed are exactly expected, none pending; then
 * no API request failed or went to an old address, the Content-Security-Policy blocked nothing and the page threw
 * nothing; its console is quiet but for warnings; and, read once more after all that, nothing came after the list was
 * whole.
 */
async function expectExactly(
  page: Page,
  watch: PageWatch,
  requests: ReturnType<typeof followRequests>,
  expected: string[],
  warnings: readonly string[]
): Promise<void> {
  const whole = { pending: 0, requests: expected.toSorted() };
  await expect.poll(() => requests.between(0)).toEqual(whole);
  expect([watch.apiFailures, watch.oldApiRequests, watch.cspViolations, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings });
  expect(requests.between(0)).toEqual(whole);
}

for (const account of ACCOUNTS) {
  const visits = REQUESTS[account];
  test(`S2: the workspace's ${account} signs in and opens ${visits.map(([path]) => path).join(", then ")}`, async ({
    api,
    signedInPage,
  }, testInfo) => {
    const { slug, project, tokens } = await acme(api, testInfo);
    const page = await signedInPage(tokens[account]);
    const watch = await watchPage(page);
    const names = { [slug]: "{slug}", [project.id]: "{project}" };
    const requests = followRequests(page, names);

    /** Opens path and waits until its requests are requested, none pending: gives the index of its first and its list. */
    const visit = async (path: string, requested: string[]) => {
      const load = { from: requests.next(), expected: requested.toSorted() };
      await page.goto(valued(path, names));
      if (path === "/") await expect(page).toHaveURL(`/${slug}`);
      await expect.poll(() => requests.between(load.from)).toEqual({ pending: 0, requests: load.expected });
      return load;
    };
    // the pages load one after another
    const loads = await visits.reduce<Promise<{ from: number; expected: string[] }[]>>(
      (before, [path, requested]) => before.then(async (loaded) => [...loaded, await visit(path, requested)]),
      Promise.resolve([])
    );
    // What the wrapper of the project's settings, the last page, shows once nerve's read has answered: the join
    // screen to one who is no member of the project, the project's own pages to its members (M3 design 3.19, 7.6).
    const join = page.getByRole("button", { name: "Join project" });
    if (account === "project non-member") await expect(join).toBeVisible();
    else await expect(join).toHaveCount(0);

    expect(watch.apiFailures).toEqual([]);
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    // The hint is logged once for each load (EMOJI_CHECK_WARNING).
    await expectQuietConsole(page, watch, { warnings: visits.map(() => EMOJI_CHECK_WARNING) });
    // Nothing came after any list was whole.
    for (const [index, { from, expected }] of loads.entries()) {
      expect(requests.between(from, loads[index + 1]?.from)).toEqual({ pending: 0, requests: expected });
    }
  });
}

test("S2: the admin of two workspaces opens a project of one at the other's address: not found, the read alone", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { slug, tokens } = await acme(api, testInfo);
  const beta = `${slug}-beta`;
  await createWorkspace(api, tokens.admin.access_token, { name: "Beta", slug: beta });
  const lab = await createProject(api, tokens.admin.access_token, beta, { name: "Lab", identifier: "LAB", network: 2 });
  const page = await signedInPage(tokens.admin);
  const watch = await watchPage(page);
  const requests = followRequests(page, { [beta]: "{beta}", [slug]: "{slug}", [lab.id]: "{project}" });

  // a project counts in the address's workspace alone (M3 design 3.19): its own resources are not fetched here
  await page.goto(`/${slug}/settings/projects/${lab.id}`);
  await expect(page.getByText("Project not found")).toBeVisible();
  await expectExactly(page, watch, requests, [...APP, ...WORKSPACE, ...PROJECT], [EMOJI_CHECK_WARNING]);
});

test("S2: the admin of a project opens it archived: the archived screen, and the project's read alone, not what its members read", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { slug, project, tokens } = await acme(api, testInfo);
  await archiveProject(api, tokens.admin.access_token, project.id);
  const page = await signedInPage(tokens.admin);
  const watch = await watchPage(page);
  const requests = followRequests(page, { [slug]: "{slug}", [project.id]: "{project}" });
  await page.goto(`/${slug}/projects/${project.id}/issues`);
  await expect(page.getByText("This project is archived")).toBeVisible();
  await expectExactly(page, watch, requests, [...APP, ...WORKSPACE, ...PROJECT], [EMOJI_CHECK_WARNING]);
});

test("S2: a newcomer opens /, which sends him to the onboarding: it asks for his workspaces, as the app does, and no more", async ({
  api,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await register(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  const requests = followRequests(page, {});
  await page.goto("/");
  await expect(page).toHaveURL("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();
  await expectExactly(page, watch, requests, APP, []);
});

test("S2: a member of a workspace opens his profile's settings directly: they ask for his workspaces, as the app does, which their sidebar lists, and no more", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug: slugFor(testInfo) });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  const requests = followRequests(page, {});
  await page.goto("/settings/profile/general");
  await expect(page.getByRole("link", { name: "Acme" })).toBeVisible();
  await expectExactly(page, watch, requests, APP, [EMOJI_CHECK_WARNING]);
});
