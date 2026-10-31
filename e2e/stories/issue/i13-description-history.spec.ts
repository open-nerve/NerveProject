import {
  addProjectMembers,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { accountId, emailFor, newAccount } from "../../fixtures/auth";
import {
  createIssue,
  descriptionPage,
  descriptionVersion,
  issueArchived,
  issueUpdated,
  updateIssue,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { instantKey } from "../../fixtures/instant";
import { expect, test } from "../../fixtures/test";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { answerTo, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { pageBehindHidden, focusInDialog } from "../../fixtures/dialogs";

// I13 creates independent history by changing author, without manipulating time.
test("I13 (API): versions paginate newest first and details restore through ordinary PATCH; same-author saves merge and archived issues refuse restoration", async ({
  api,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const ann = await newAccount(api, testInfo, "ann");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, ann, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: ann.id, role: 15 }]);
  const issue = await createIssue(api, admin.token, project.id, {
    name: "History",
    description_html: "<p>Initial</p>",
  });
  const initial = (await descriptionPage(api, admin.token, issue.id)).data[0];
  expect(initial).toMatchObject({ issue_id: issue.id, owned_by_id: admin.id });
  if (!initial) throw new Error("missing initial version");
  await updateIssue(api, ann.token, issue.id, { description_html: "<p>Ann draft</p>" });
  const draft = (await descriptionPage(api, admin.token, issue.id)).data[0];
  await updateIssue(api, ann.token, issue.id, { description_html: "<p>Ann final</p>" });
  const merged = await descriptionPage(api, admin.token, issue.id);
  expect(merged.data.map((row) => row.id)).toEqual([draft?.id, initial.id]);
  expect(instantKey(merged.data[0]?.last_saved_at ?? "") > instantKey(draft?.last_saved_at ?? "")).toBe(true);
  await updateIssue(api, admin.token, issue.id, { description_html: "<p>Admin final</p>" });
  const first = await descriptionPage(api, ann.token, issue.id, undefined, 1);
  expect(first.next_cursor).not.toBeNull();
  const second = await descriptionPage(api, ann.token, issue.id, first.next_cursor ?? undefined, 1);
  const third = await descriptionPage(api, ann.token, issue.id, second.next_cursor ?? undefined, 1);
  expect(third.next_cursor).toBeNull();
  expect([first.data[0]?.owned_by_id, second.data[0]?.owned_by_id, third.data[0]?.owned_by_id]).toEqual([
    admin.id,
    ann.id,
    admin.id,
  ]);
  expect([first.data[0]?.id, second.data[0]?.id, third.data[0]?.id]).toEqual([
    expect.any(String),
    draft?.id,
    initial.id,
  ]);
  const historical = await descriptionVersion(api, ann.token, initial.id);
  expect(historical).toMatchObject({ ...initial, description_html: "<p>Initial</p>" });
  const beforeRestore = await descriptionPage(api, admin.token, issue.id, undefined, 100);
  expect(
    (await updateIssue(api, admin.token, issue.id, { description_html: historical.description_html })).description_html
  ).toBe(historical.description_html);
  const afterRestore = await descriptionPage(api, admin.token, issue.id, undefined, 100);
  expect(afterRestore.data.map((row) => row.id)).toEqual(beforeRestore.data.map((row) => row.id));
  const restoredVersion = afterRestore.data[0];
  if (!restoredVersion) throw new Error("no restored version");
  expect((await descriptionVersion(api, ann.token, restoredVersion.id)).description_html).toBe(
    historical.description_html
  );
  const { Done: done } = await stateIds(api, admin.token, project.id);
  if (!done) throw new Error("no Done state");
  await updateIssue(api, admin.token, issue.id, { state_id: done });
  expect((await issueArchived(api, admin.token, issue.id)).status).toBe(200);
  expect(await issueUpdated(api, ann.token, issue.id, { description_html: "<p>Ann final</p>" })).toEqual({
    status: 409,
    code: "issue.archived",
  });
  expect(await descriptionPage(api, admin.token, issue.id, undefined, 100)).toEqual(afterRestore);
  expect(await descriptionVersion(api, ann.token, initial.id)).toEqual(historical);
});

test("I13 (page): history loads a second page, copies actual Markdown after settlement, and restores only after a confirmed ordinary PATCH", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const other = await newAccount(api, info, "history-other");
  await inviteAndAccept(api, seed.member.token, seed.slug, other, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: other.id, role: 15 }]);
  const original = "<h1>Original</h1><p>Keep <strong>bold</strong>.</p>";
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "History controls",
    description_html: original,
  });
  // Different authors create real independent versions without waiting or changing database time.
  await Array.from({ length: 2 }, (_, index) => index).reduce(async (previous, index) => {
    await previous;
    await updateIssue(api, index % 2 === 0 ? other.token : seed.member.token, issue.id, {
      description_html: `<p>Revision ${index + 1}</p>`,
    });
  }, Promise.resolve());
  const versions = await descriptionPage(api, seed.member.token, issue.id, undefined, 100);
  expect(versions.data).toHaveLength(3);
  // The worker keeps only 20 versions. Request small real pages instead of racing retention with 51 versions.
  await page.route(
    (url) => url.pathname === `/api/v0/issues/${issue.id}/description-versions`,
    async (route) => {
      const url = new URL(route.request().url());
      url.searchParams.set("limit", "2");
      await route.continue({ url: url.toString() });
    }
  );
  const watch = await openIssueDetail(page, seed, issue);
  await page.getByRole("button", { name: "Load more", exact: true }).click();
  await expect(page.getByRole("button", { name: "Load more", exact: true })).toHaveCount(0);
  const trigger = page.getByRole("button", { name: /^Last edited by/ });
  await trigger.press("Enter");
  await expect(page.getByRole("menuitem")).toHaveCount(3);
  await page.getByRole("menuitem").last().press("Enter");
  const modal = page.getByRole("dialog", { name: "Description history", exact: true });
  await expect(modal.getByRole("heading", { name: "Original", exact: true })).toBeVisible();
  expect(await pageBehindHidden(page)).toBe(true);
  expect(
    await modal
      .locator("[tabindex]")
      .evaluateAll((nodes) => nodes.filter((node) => node instanceof HTMLElement && node.tabIndex > 0).length)
  ).toBe(0);
  const clipboard = await page.evaluateHandle(() => {
    const calls: string[] = [];
    let finish!: () => void;
    const held = new Promise<void>((resolve) => {
      finish = resolve;
    });
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: (text: string) => {
          calls.push(text);
          return held;
        },
      },
    });
    return { calls, finish: () => finish() };
  });
  const copy = modal.getByRole("button", { name: "Copy markdown", exact: true });
  await copy.dblclick();
  expect(await clipboard.evaluate((state) => state.calls)).toEqual(["# Original\n\nKeep **bold**.\n"]);
  await expect(copy).toBeDisabled();
  await expect(page.getByText("Markdown copied to clipboard.", { exact: true })).toHaveCount(0);
  await clipboard.evaluate((state) => state.finish());
  await expect(page.getByText("Markdown copied to clipboard.", { exact: true })).toHaveCount(1);
  await expect(copy).toBeEnabled();
  const path = `/api/v0/issues/${issue.id}`;
  let refused = false;
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      if (route.request().method() !== "PATCH" || refused) return route.fallback();
      refused = true;
      await route.fulfill({
        status: 403,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 403, code: "forbidden", title: "Forbidden" }),
      });
    }
  );
  const restore = modal.getByRole("button", { name: "Restore", exact: true });
  expect((await answerTo(page, "PATCH", path, () => restore.click())).status()).toBe(403);
  await expect(modal.getByRole("alert")).toBeVisible();
  await expect(modal.getByRole("heading", { name: "Original", exact: true })).toBeVisible();
  const held = await sentHeld(page, "PATCH", path, () => restore.dblclick());
  expect(held.body).toMatchObject({ description_html: original });
  await expect(restore).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(modal).toHaveCount(1);
  await expect(modal.getByRole("heading", { name: "Original", exact: true })).toBeVisible();
  expect(await focusInDialog(page)).toBe(true);
  expect((await held.release()).status()).toBe(200);
  await expect(modal).toHaveCount(0);
  await expect(trigger).toBeFocused();
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await expect(editor.getByRole("heading", { name: "Original", exact: true })).toBeVisible();
  expect((await descriptionPage(api, seed.member.token, issue.id, undefined, 100)).data.map((row) => row.id)).toEqual(
    versions.data.map((row) => row.id)
  );
  expect(watch.apiRequests.filter((request) => request === `PATCH ${path}`)).toHaveLength(2);
  await expectListPageQuiet(page, watch, {
    failures: [`403 PATCH ${path}`],
    errors:
      browserName === "chromium"
        ? ["Failed to load resource: the server responded with a status of 403 (Forbidden)"]
        : [],
  });
});

test("I13 (page): archived history keeps list and detail refusal retries separate and has no restore action", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Archived history",
    state_id: seed.states.Done,
    description_html: "<p>Archived text</p>",
  });
  expect((await issueArchived(api, seed.member.token, issue.id)).status).toBe(200);
  const version = (await descriptionPage(api, seed.member.token, issue.id)).data[0]!;
  const paths = [`/api/v0/issues/${issue.id}/description-versions`, `/api/v0/issue-description-versions/${version.id}`];
  const refused = new Set<string>();
  await page.route(
    (url) => paths.includes(url.pathname),
    async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (refused.has(path)) return route.fallback();
      refused.add(path);
      await route.fulfill({
        status: 500,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 500, code: "internal_error", title: "Unavailable" }),
      });
    }
  );
  const watch = await openIssueDetail(page, seed, issue, issue.name);
  await expect(page.getByRole("alert")).toBeVisible();
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  const trigger = page.getByRole("button", { name: /^Last edited by/ });
  await expect(trigger).toBeEnabled();
  await trigger.press("Space");
  await page.getByRole("menuitem").press("Enter");
  const modal = page.getByRole("dialog", { name: "Description history", exact: true });
  await expect(modal.getByRole("alert")).toBeVisible();
  await expect(modal.getByRole("button", { name: "Copy markdown", exact: true })).toBeDisabled();
  await modal.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(modal.getByText("Archived text", { exact: true })).toBeVisible();
  await expect(modal.getByRole("button", { name: "Restore", exact: true })).toHaveCount(0);
  await expect(modal.getByRole("button", { name: "Copy markdown", exact: true })).toBeEnabled();
  await page.keyboard.press("Escape");
  await expect(modal).toHaveCount(0);
  await expect(trigger).toBeFocused();
  expect(watch.apiRequests.filter((request) => /^(POST|PATCH|DELETE) .*\/issues\//.test(request))).toEqual([]);
  await expectListPageQuiet(page, watch, {
    failures: paths.map((path) => `500 GET ${path}`),
    errors:
      browserName === "chromium"
        ? paths.map(() => "Failed to load resource: the server responded with a status of 500 (Internal Server Error)")
        : [],
  });
});

test("I13 (page): a project guest reads his own visible item without any history request or control", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "history-guest");
  const tokens = await registerOnboarded(api, email);
  const guest = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, guest, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: guest.id, role: 15 }]);
  const issue = await createIssue(api, guest.token, seed.project.id, {
    name: "Guest's item",
    description_html: "<p>Guest text</p>",
  });
  expect(
    await projectMemberWrites(api, seed.project.id).change(
      seed.member.token,
      await projectMembershipOf(api, seed.member.token, seed.project.id, guest.id),
      5
    )
  ).toEqual({ status: 200 });
  const watch = await openIssueDetail(page, { ...seed, member: guest }, issue, issue.name);
  await expect(page.getByText("Guest text", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: /^Last edited by/ })).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request.includes("description-versions"))).toEqual([]);
  await expectListPageQuiet(page, watch);
});
