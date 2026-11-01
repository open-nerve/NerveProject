import { c12Evidence } from "../../fixtures/c12-evidence";
import { createLabel, changeRole, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { highlightCommand, openCommands, selectCommand } from "../../fixtures/power-k";
import { createIssueLink } from "../../fixtures/links";
import {
  collaborationLANPage,
  copyWithoutUserActivation,
  observeLANClipboard,
  expectLANClipboardWarnings,
} from "../../fixtures/issue-collaboration-page";
import { createState } from "../../fixtures/states";
import { modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { watchPage } from "../../fixtures/browser";
import { createIssueComment } from "../../fixtures/comments";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { createDraft, createIssue, createIssueRelations, updateIssue } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";
import { anotherTabSignsIn, bearer, shownNameOf, emailFor, accountId, password } from "../../fixtures/auth";
import { sentHeld, answerTo, moveWithinApp, registerOnboarded, transitionsEnded } from "../../fixtures/settings-pages";

test("C6: activity filters keep one choice, support Enter and Space, persist sorting and return focus on Escape", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Activity controls" });
  const first = await createIssueComment(api, seed.member.token, issue.id, "<p>First comment</p>");
  await updateIssue(api, seed.member.token, issue.id, { name: "Updated title" });
  const last = await createIssueComment(api, seed.member.token, issue.id, "<p>Last comment</p>");
  const watch = await openIssueDetail(page, seed, issue);
  const filter = page.getByRole("button", { name: "Filters", exact: true });
  const sort = page.getByRole("button", { name: "Toggle sort order", exact: true });
  const firstRow = page.locator(`[id="comment-${first.id}"]`);
  const timeline = firstRow.locator("..");
  const updated = page.getByText("set the name to Updated title.", { exact: true });
  const commentOrder = () => timeline.locator("xpath=./div[@id]").evaluateAll((rows) => rows.map((row) => row.id));
  await expect(firstRow).toBeVisible();
  await expect(updated).toBeVisible();
  expect(await commentOrder()).toEqual([`comment-${first.id}`, `comment-${last.id}`]);
  await expect(filter.locator("button")).toHaveCount(0);
  await expect(filter.locator("xpath=ancestor::button")).toHaveCount(0);
  await filter.focus();
  await page.keyboard.press("Enter");
  const updates = page.getByRole("button", { name: "Updates", exact: true });
  const comments = page.getByRole("button", { name: "Comments", exact: true });
  // Safari on macOS includes buttons in sequential navigation through Option-Tab.
  await page.keyboard.press(browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab");
  await expect(updates).toBeFocused();
  await c12Evidence(page, "B16", "timeline-filter-tab");
  await page.keyboard.press("Space");
  await expect(updates).toHaveAttribute("aria-pressed", "false");
  await expect(updated).toHaveCount(0);
  await page.getByRole("button", { name: "State", exact: true }).press("Enter");
  await page.getByRole("button", { name: "Assignee", exact: true }).press("Space");
  await expect(comments).toHaveAttribute("aria-disabled", "true");
  await comments.press("Enter");
  await expect(comments).toHaveAttribute("aria-pressed", "true");
  await expect(firstRow).toBeVisible();
  await comments.press("Escape");
  await expect(updates).toHaveCount(0);
  await expect(filter).toBeFocused();
  await sort.press("Space");
  await expect.poll(commentOrder).toEqual([`comment-${last.id}`, `comment-${first.id}`]);
  await page.reload();
  await expect(firstRow).toBeVisible();
  expect(await commentOrder()).toEqual([`comment-${last.id}`, `comment-${first.id}`]);
  await expect(updated).toHaveCount(0);
  await filter.press("Space");
  await expect(comments).toHaveAttribute("aria-pressed", "true");
  await expect(updates).toHaveAttribute("aria-pressed", "false");
  await updates.press("Enter");
  await expect(updated).toBeVisible();
  await updates.press("Escape");
  await expect(filter).toBeFocused();
  await sort.press("Enter");
  await expect.poll(commentOrder).toEqual([`comment-${first.id}`, `comment-${last.id}`]);
  await expectListPageQuiet(page, watch);
});

test("C6: reaction picker has one named button, keyboard focus returns on Escape, and combined emoji data stays on this origin", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Emoji controls" });
  const comment = await createIssueComment(api, seed.member.token, issue.id, "<p>Discuss emoji</p>");
  const watch = await openIssueDetail(page, seed, issue);
  const row = page.locator(`[id="comment-${comment.id}"]`);
  const name = `Add reaction to ${shownNameOf(seed.member.email)}’s comment`;
  const header = row.getByRole("button", { name, exact: true }).first();
  await expect(header).toBeVisible();
  await expect(row.locator("button button")).toHaveCount(0);
  await header.press("Enter");
  const search = page.getByRole("searchbox", { name: "Search", exact: true });
  await expect(search).toBeFocused();
  await search.press("Escape");
  await expect(search).toHaveCount(0);
  await expect(header).toBeFocused();
  await header.press("Space");
  await search.fill("woman technologist");
  const choice = page.getByRole("gridcell", { name: "Woman technologist", exact: true });
  await expect(choice).toBeVisible();
  await c12Evidence(page, "B16", "combined-emoji-keyboard-picker");
  const code = "128105-8205-128187";
  const path = `/api/v0/issue-comments/${comment.id}/reactions`;
  const held = await sentHeld(page, "POST", path, () => choice.press("Enter"));
  expect(held.body).toEqual({ reaction: code });
  await expect(header).toBeDisabled();
  await c12Evidence(page, "B16", "emoji-post-pending");
  await expect(row.getByRole("button", { name: /👩‍💻/ })).toHaveCount(0);
  expect((await held.release()).status()).toBe(201);
  await expect(row.getByRole("button", { name: "👩‍💻 1", exact: true })).toBeVisible();
  await expect(row.getByRole("button", { name, exact: true })).toHaveCount(2);
  const footer = row.getByRole("button", { name, exact: true }).last();
  await expect(footer).toBeEnabled();
  await footer.press("Space");
  await expect(search).toBeFocused();
  await search.press("Escape");
  await expect(footer).toBeFocused();
  const issueAdd = page.getByRole("button", { name: "Add reaction", exact: true });
  await expect(issueAdd.locator("button")).toHaveCount(0);
  await issueAdd.press("Enter");
  await search.fill("woman technologist");
  const issuePath = `/api/v0/issues/${issue.id}/reactions`;
  const issueHeld = await sentHeld(page, "POST", issuePath, () => choice.press("Space"));
  expect(issueHeld.body).toEqual({ reaction: code });
  expect((await issueHeld.release()).status()).toBe(201);
  await expect(page.getByRole("button", { name: "👩‍💻 1", exact: true })).toHaveCount(2);
  await c12Evidence(page, "B16", "emoji-confirmed-comment-and-issue");
  expect(watch.requestUrls.some((url) => new URL(url).pathname === "/assets/emojibase/15.3.2/en/data.json")).toBe(true);
  expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toHaveLength(1);
  expect(watch.apiRequests.filter((request) => request === `POST ${issuePath}`)).toHaveLength(1);
  await expectListPageQuiet(page, watch);
});

test("C9: creation saves an empty-title real editor draft, retains field refusals, and its nested prompt owns Escape and confirmed close", async ({
  api,
  page,
  db,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { watch } = await openIssueList(page, seed);
  const trigger = page.getByRole("button", { name: "Add work item", exact: true });
  await trigger.click();
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  const title = form.getByRole("textbox", { name: "Title", exact: true });
  const editor = form.locator('[contenteditable="true"]');
  await expect(editor).toBeVisible();
  await editor.fill("Keep unfinished content");
  const path = `/api/v0/workspaces/${seed.slug}/drafts`;
  let reject = true;
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      if (!reject || route.request().method() !== "POST") return route.fallback();
      reject = false;
      await route.fulfill({
        status: 422,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 422, code: "validation_failed", errors: [{ field: "name", code: "too_long" }] }),
      });
    }
  );
  expect(
    (
      await answerTo(page, "POST", path, () =>
        form.getByRole("button", { name: "Save to Drafts", exact: true }).click()
      )
    ).status()
  ).toBe(422);
  await expect(title).toHaveValue("");
  await expect(title).toHaveAttribute("aria-invalid", "true");
  await c12Evidence(page, "B18", "draft-field-refusal");
  await expect(title).toBeFocused();
  await expect(editor).toContainText("Keep unfinished content");
  const held = await sentHeld(page, "POST", path, () =>
    form.getByRole("button", { name: "Save to Drafts", exact: true }).dblclick()
  );
  expect(held.body).toMatchObject({
    project_id: seed.project.id,
    payload: { name: "", description_html: expect.stringContaining("Keep unfinished content") },
  });
  expect(held.body).not.toHaveProperty("skip_activity");
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(1);
  expect((await held.release()).status()).toBe(201);
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(trigger).toBeFocused();
  await trigger.click();
  await expect(editor).toBeVisible();
  await title.fill("Retain on inner Escape");
  await editor.fill("Save from close");
  await form.getByRole("switch", { name: "Create more", exact: true }).click();
  const discard = form.getByRole("button", { name: "Discard", exact: true });
  await discard.press("Enter");
  const prompt = page.getByRole("dialog", { name: "Save this draft?", exact: true });
  await expect(prompt.getByRole("heading", { name: "Save this draft?", exact: true })).toBeVisible();
  await c12Evidence(page, "B18", "nested-draft-save-prompt");
  expect(await pageBehindHidden(page)).toBe(true);
  await prompt.getByRole("button", { name: "Cancel", exact: true }).press("Escape");
  await expect(prompt).toHaveCount(0);
  await expect(discard).toBeFocused();
  await expect(title).toHaveValue("Retain on inner Escape");
  await expect(editor).toContainText("Save from close");
  await discard.click();
  const saved = await sentHeld(page, "POST", path, () =>
    prompt.getByRole("button", { name: "Save to Drafts", exact: true }).dblclick()
  );
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(2);
  expect((await saved.release()).status()).toBe(201);
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(trigger).toBeFocused();
  expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toHaveLength(3);
  expect(
    await db.query(
      "SELECT payload->>'name' AS name, payload->>'description_html' AS html FROM draft_issues WHERE workspace_id=$1 ORDER BY created_at",
      [seed.project.workspace_id]
    )
  ).toEqual([
    { name: "", html: expect.stringContaining("Keep unfinished content") },
    { name: "Retain on inner Escape", html: expect.stringContaining("Save from close") },
  ]);
  expect(await db.query("SELECT count(*)::int AS count FROM issues WHERE project_id=$1", [seed.project.id])).toEqual([
    { count: 0 },
  ]);
  await expectListPageQuiet(page, watch, {
    failures: [`422 POST ${path}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"]
        : [],
  });
});

test("C9: Back retires a pending draft save and its prompt before the confirmed old reply reaches another workspace form", async ({
  api,
  page,
  db,
}, info) => {
  const seed = await issueListProject(api, info);
  const beta = slugFor(info, "beta-drafts");
  await createWorkspace(api, seed.member.token, { name: "Beta drafts", slug: beta });
  const { watch } = await openIssueList(page, seed);
  await moveWithinApp(page, `/${beta}/drafts`);
  await expect(page.getByRole("button", { name: "Draft a work item", exact: true })).toBeVisible();
  await moveWithinApp(page, seed.path);
  await page.getByRole("button", { name: "Add work item", exact: true }).click();
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  await expect(form.locator('[contenteditable="true"]')).toBeVisible();
  await form.getByRole("textbox", { name: "Title", exact: true }).fill("Old draft confirmed later");
  await form.getByRole("button", { name: "Discard", exact: true }).click();
  const prompt = page.getByRole("dialog", { name: "Save this draft?", exact: true });
  const path = `/api/v0/workspaces/${seed.slug}/drafts`;
  const held = await sentHeld(page, "POST", path, () =>
    prompt.getByRole("button", { name: "Save to Drafts", exact: true }).click()
  );
  let released = false;
  try {
    await expect
      .poll(() =>
        db.query("SELECT count(*)::int AS count FROM draft_issues WHERE workspace_id=$1", [seed.project.workspace_id])
      )
      .toEqual([{ count: 1 }]);
    await page.goBack();
    await expect(page).toHaveURL(`/${beta}/drafts`);
    await expect(modalDialog(page)).toHaveCount(0);
    await page.getByRole("button", { name: "Draft a work item", exact: true }).click();
    const next = page.getByRole("dialog", { name: "Create a draft", exact: true });
    await next.getByRole("textbox", { name: "Title", exact: true }).fill("Keep new workspace input");
    expect((await held.release()).status()).toBe(201);
    released = true;
    await expect(next.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Keep new workspace input");
    await c12Evidence(page, "B18", "back-preserves-next-workspace-form");
    await expect(page.getByText("Draft created", { exact: true })).toHaveCount(0);
    expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toHaveLength(1);
    await expectListPageQuiet(page, watch);
  } finally {
    if (!released) await held.release();
  }
});

test("C9: a draft editor stays retired across member to guest to member permissions after real reloads", async ({
  api,
  page,
  db,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "draft-permission");
  const { access_token: token } = await registerOnboarded(api, email);
  const member = { email, token, id: await accountId(api, token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, member, 15);
  const watch = await watchPage(page);
  const path = `/${seed.slug}/drafts`;
  await page.goto(signInPath(path));
  expect(await submitSignIn(page, email, password)).toBe(200);
  await page.getByRole("button", { name: "Draft a work item", exact: true }).click();
  const form = modalDialog(page);
  await expect(form.locator('[contenteditable="true"]')).toBeVisible();
  await form.getByRole("textbox", { name: "Title", exact: true }).fill("Must not revive");
  await form.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(2);
  await changeRole(api, seed.member.token, seed.slug, member.id, 5);
  const readsBefore = watch.apiRequests.filter((request) => request.endsWith(`/workspaces/${seed.slug}/drafts`)).length;
  await page.reload();
  await expect(page.getByRole("button", { name: "Draft a work item", exact: true })).toBeDisabled();
  await expect(modalDialog(page)).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request.endsWith(`/workspaces/${seed.slug}/drafts`))).toHaveLength(
    readsBefore
  );
  await changeRole(api, seed.member.token, seed.slug, member.id, 15);
  await page.reload();
  const trigger = page.getByRole("button", { name: "Draft a work item", exact: true });
  await expect(trigger).toBeEnabled();
  await expect(modalDialog(page)).toHaveCount(0);
  await trigger.press("Enter");
  const reopened = page.getByRole("dialog", { name: "Create a draft", exact: true });
  await expect(reopened.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("");
  await c12Evidence(page, "B18", "permission-return-fresh-draft");
  await expect(reopened.locator('[contenteditable="true"]')).toBeVisible();
  await reopened.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(0);
  expect(
    await db.query("SELECT count(*)::int AS count FROM draft_issues WHERE workspace_id=$1", [seed.project.workspace_id])
  ).toEqual([{ count: 0 }]);
  await expectListPageQuiet(page, watch);
});

test("C9: real editor block-id migration in a copied draft is not an unsaved user edit", async ({
  api,
  page,
  db,
}, info) => {
  const seed = await issueListProject(api, info);
  const payload = { name: "Legacy draft", description_html: "<p>Before block IDs</p>" };
  const draft = await createDraft(api, seed.member.token, seed.slug, { payload });
  const { watch } = await openIssueList(page, seed);
  await moveWithinApp(page, `/${seed.slug}/drafts`);
  await page.locator(`[id="issue-${draft.id}"] button[aria-haspopup="menu"]:visible`).last().click();
  await page.getByRole("menuitem", { name: "Make a copy", exact: true }).click();
  const form = page.getByRole("dialog", { name: "Create a draft", exact: true });
  await expect(form.locator('[contenteditable="true"] p[data-id]')).toHaveText("Before block IDs");
  await expect(form.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Legacy draft (copy)");
  await c12Evidence(page, "B18", "legacy-draft-id-migration");
  await form.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(page.getByRole("dialog", { name: "Save this draft?", exact: true })).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => /^(POST|PATCH|DELETE) .*drafts/.test(request))).toEqual([]);
  expect(await db.query("SELECT payload FROM draft_issues WHERE workspace_id=$1", [seed.project.workspace_id])).toEqual(
    [{ payload }]
  );
  await expectListPageQuiet(page, watch);
});

test("C7: empty-query commands use the current issue, Chinese state selection respects IME, and deletion keeps its confirmation and focus", async ({
  api,
  page,
  db,
}, info) => {
  const seed = await issueListProject(api, info);
  const state = await createState(api, seed.member.token, seed.project.id, {
    name: "处理中",
    color: "#333333",
    group: "started",
  });
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Command target" });
  const watch = await openIssueDetail(page, seed, issue);
  const title = page.getByRole("textbox", { name: "Work item title", exact: true });
  await title.focus();
  await openCommands(page);
  await selectCommand(page, /^Change state/);
  const input = page.getByRole("combobox");
  await input.fill("处理");
  await expect(page.getByRole("option", { name: "处理中", exact: true })).toBeVisible();
  await c12Evidence(page, "B19", "chinese-state-command");
  const path = `/api/v0/issues/${issue.id}`;
  const writes = () => watch.apiRequests.filter((item) => item === `PATCH ${path}`);
  await input.dispatchEvent("keydown", { key: "Enter", isComposing: true, bubbles: true });
  await expect(input).toHaveValue("处理");
  expect(writes()).toHaveLength(0);
  const held = await sentHeld(page, "PATCH", path, () => selectCommand(page, /^处理中$/));
  expect(held.body).toEqual({ state_id: state.id });
  await expect(page.getByRole("combobox")).toHaveCount(0);
  await expect(page.getByText("处理中", { exact: true })).toHaveCount(0);
  expect((await held.release()).status()).toBe(200);
  await expect(page.getByRole("combobox")).toHaveCount(0);
  await expect
    .poll(
      async () =>
        (await db.query<{ state_id: string }>("SELECT state_id FROM issues WHERE id=$1", [issue.id]))[0]?.state_id
    )
    .toBe(state.id);
  await title.focus();
  await openCommands(page);
  await selectCommand(page, /^Delete/);
  const dialog = page.getByRole("dialog", { name: "Delete Work item", exact: true });
  await expect(dialog).toHaveCount(1);
  await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeVisible();
  await c12Evidence(page, "B13", "command-delete-confirmation");
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(title).toBeFocused();
  expect(watch.apiRequests.filter((item) => item === `DELETE ${path}`)).toHaveLength(0);
  await openCommands(page);
  await selectCommand(page, /^Delete/);
  const deletion = await sentHeld(page, "DELETE", path, () =>
    dialog.getByRole("button", { name: "Delete", exact: true }).click()
  );
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(1);
  expect((await deletion.release()).status()).toBe(204);
  await expect(page).toHaveURL(seed.path);
  expect(watch.apiRequests.filter((item) => item === `DELETE ${path}`)).toHaveLength(1);
  expect(watch.requestUrls.filter((url) => new URL(url).pathname.endsWith("/search"))).toEqual([]);
  await expectListPageQuiet(page, watch);
});

test("C7: keyboard theme selection waits for confirmation, reports native refusal, then reloads only after success", async ({
  api,
  signedInPage,
  db,
  browserName,
}, info) => {
  const email = emailFor(info, "command-theme");
  const page = await signedInPage(await registerOnboarded(api, email));
  const watch = await watchPage(page);
  await page.goto("/settings/profile/preferences");
  await expect(page.getByRole("button", { name: "System Preference", exact: true })).toBeVisible();
  await page.evaluate(() => {
    document.documentElement.dataset.commandDocument = "original";
  });
  let refused = false;
  const path = "/api/v0/me/profile";
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      if (route.request().method() !== "PATCH" || refused) return route.fallback();
      refused = true;
      await route.fulfill({
        status: 503,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 503, code: "server_busy" }),
      });
    }
  );
  await openCommands(page);
  await selectCommand(page, /^Change interface theme$/);
  // Keep the real exit transition in progress so reopening cannot accidentally wait for unmount.
  const panel = page.locator('[id^="headlessui-dialog-panel-"]');
  await transitionsEnded(panel);
  const slowExit = await page.addStyleTag({
    content: '[id^="headlessui-dialog-panel-"][data-leave] { transition-duration: 30s !important; }',
  });
  expect((await answerTo(page, "PATCH", path, () => selectCommand(page, /^Dark$/))).status()).toBe(503);
  await expect(page.getByText("The server is busy. Please try again later.", { exact: true })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expect(page.locator("html")).toHaveAttribute("data-command-document", "original");
  // Exercise both modifiers on every platform: cmdk also binds Control+K to Vim navigation.
  const reopenDuringExit = async (shortcut: "Control+k" | "Meta+k") => {
    await expect(panel).toHaveAttribute("data-leave", "");
    await selectCommand(page, /^Change interface theme$/);
    await expect(page.getByRole("option", { name: /^Change interface theme$/ })).toBeVisible();
    await page.keyboard.press(shortcut);
    await expect(page.getByRole("combobox")).toBeFocused();
    await expect(page.getByRole("combobox")).toHaveValue("");
    await expect(panel).not.toHaveAttribute("data-leave", "");
  };
  await reopenDuringExit("Control+k");
  await page.keyboard.press("Escape");
  await reopenDuringExit("Meta+k");
  await slowExit.evaluate((element) => {
    element.parentNode?.removeChild(element);
  });
  await selectCommand(page, /^Change interface theme$/);
  const held = await sentHeld(page, "PATCH", path, () => selectCommand(page, /^Dark$/));
  expect(held.body).toEqual({ theme: "dark" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expect(page.locator("html")).toHaveAttribute("data-command-document", "original");
  // Let the new document finish loading before Playwright's visibility probes can force its first layout.
  const [, confirmed] = await Promise.all([page.waitForEvent("load"), held.release()]);
  expect(confirmed.status()).toBe(200);
  await expect(page.locator("html")).not.toHaveAttribute("data-command-document", "original");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByRole("button", { name: "Dark", exact: true })).toBeVisible();
  await c12Evidence(page, "B21", "theme-confirmed-dark");
  expect(
    await db.query("SELECT p.theme FROM profiles p JOIN users u ON u.id=p.user_id WHERE u.email=$1", [email])
  ).toEqual([{ theme: "dark" }]);
  await expectListPageQuiet(page, watch, {
    failures: [`503 PATCH ${path}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 503 (Service Unavailable)"]
        : [],
  });
});

test("C7: a real theme success from the previous login cannot recolor or reload the new session", async ({
  api,
  signedInPage,
  context,
}, info) => {
  const first = emailFor(info, "theme-first"),
    next = emailFor(info, "theme-next");
  const page = await signedInPage(await registerOnboarded(api, first));
  const other = await registerOnboarded(api, next);
  expect(
    (await api.PATCH("/api/v0/me/profile", { body: { theme: "light-contrast" }, headers: bearer(other.access_token) }))
      .response.status
  ).toBe(200);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/preferences");
  await expect(page.getByRole("button", { name: "System Preference", exact: true })).toBeVisible();
  await openCommands(page);
  await selectCommand(page, /^Change interface theme$/);
  const held = await sentHeld(page, "PATCH", "/api/v0/me/profile", () => selectCommand(page, /^Dark$/));
  await anotherTabSignsIn(context, api, next);
  await expect(page.getByRole("button", { name: "Light high contrast", exact: true })).toBeVisible();
  await page.evaluate(() => {
    document.documentElement.dataset.commandDocument = "new-session";
  });
  const reload = page
    .waitForEvent("request", { predicate: (request) => request.isNavigationRequest(), timeout: 1000 })
    .then(
      () => true,
      () => false
    );
  expect((await held.release()).status()).toBe(200);
  expect(await reload).toBe(false);
  await expect(page.locator("html")).toHaveAttribute("data-command-document", "new-session");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light-contrast");
  await c12Evidence(page, "B21", "old-theme-success-keeps-new-session");
  await expect(page.getByText("Theme updated", { exact: true })).toHaveCount(0);
  await expectListPageQuiet(page, watch);
});

for (const control of [
  "identifier",
  "detail",
  "link",
  "relation",
  "sub-item",
  "peek",
  "comment",
  "layout",
  "creation",
  "command-id",
  "command-title",
  "command-url",
] as const) {
  for (const gesture of [false, true]) {
    test(
      gesture
        ? `C12: ${control} copy on LAN HTTP with a real keyboard gesture reports the native result without writes or navigation`
        : `C8: ${control} copy on LAN HTTP without a user gesture reports one refusal without success, writes or navigation`,
      async ({ api, db, browser, browserName }, info) => {
        const seed = await issueListProject(api, info);
        const source = await createIssue(api, seed.member.token, seed.project.id, { name: "LAN copy source" });
        const comment =
          control === "comment"
            ? await createIssueComment(api, seed.member.token, source.id, "<p>LAN discussion</p>")
            : undefined;
        if (control === "link")
          await createIssueLink(api, seed.member.token, source.id, {
            title: "Guide",
            url: "https://example.test/guide",
          });
        if (control === "relation") {
          const target = await createIssue(api, seed.member.token, seed.project.id, { name: "Related target" });
          await createIssueRelations(api, seed.member.token, source.id, {
            relation_type: "relates_to",
            issue_ids: [target.id],
          });
        }
        if (control === "sub-item")
          await createIssue(api, seed.member.token, seed.project.id, { name: "Child target", parent_id: source.id });
        const lan = await collaborationLANPage(browser, db, info);
        const page = lan.page;
        try {
          const watch = ["layout", "peek", "creation"].includes(control)
            ? (await openIssueList(page, seed)).watch
            : await openIssueDetail(page, seed, source);
          expect(new URL(page.url()).protocol).toBe("http:");
          expect(new URL(page.url()).hostname).not.toMatch(/^(localhost|127\.|\[?::1)/);
          expect(await page.evaluate(() => window.isSecureContext)).toBe(false);
          if (control === "creation") {
            await page.getByRole("button", { name: "Add work item", exact: true }).click();
            const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
            await form.getByRole("textbox", { name: "Title", exact: true }).fill("Created on LAN");
            await expect(form.locator('[contenteditable="true"]')).toBeVisible();
            expect(
              (
                await answerTo(page, "POST", `/api/v0/projects/${seed.project.id}/issues`, () =>
                  form.getByRole("button", { name: "Save", exact: true }).click()
                )
              ).status()
            ).toBe(201);
            await expect(form).toHaveCount(0);
          }
          if (control === "peek") {
            await page.locator(`#issue-${source.id}`).click();
            await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
          }
          if (control === "relation") {
            const relations = page.getByRole("button", { name: "Relations 1", exact: true });
            await relations.click();
            await transitionsEnded(relations.locator("../.."));
          }
          const before = page.url();
          const requestsBefore = watch.apiRequests.length;
          const nativeClipboard = await observeLANClipboard(page);
          const activate = async (target: Parameters<typeof copyWithoutUserActivation>[0]) => {
            if (gesture) await target.press("Enter");
            else await copyWithoutUserActivation(target);
          };
          if (control.startsWith("command-")) {
            await openCommands(page);
            const command = await highlightCommand(
              page,
              control === "command-id" ? /^Copy ID/ : control === "command-title" ? /^Copy title/ : /^Copy URL/
            );
            await activate(command);
          } else {
            let copy;
            if (["relation", "sub-item", "comment", "layout"].includes(control)) {
              const options =
                control === "comment"
                  ? page.locator(`[id="comment-${comment!.id}"]`).getByRole("button", {
                      name: `Options for ${shownNameOf(seed.member.email)}’s comment`,
                      exact: true,
                    })
                  : page.getByRole("button", {
                      name:
                        control === "layout"
                          ? `Options: ${source.name}`
                          : `${control === "relation" ? "Related target" : "Child target"} options`,
                      exact: true,
                    });
              await options.focus();
              await page.keyboard.press("Enter");
              copy = page.getByRole("menuitem", {
                name: control === "sub-item" ? "Copy work item link" : "Copy link",
                exact: true,
              });
            } else {
              copy = page.getByRole("button", {
                name:
                  control === "identifier"
                    ? `Copy ID: WEB-${source.sequence_id}`
                    : control === "link"
                      ? "Copy link: Guide"
                      : "Copy link",
                exact: true,
              });
            }
            await copy.focus();
            await expect(copy).toBeFocused();
            await activate(copy);
          }
          const calls = await nativeClipboard.evaluate((state) => state.calls);
          expect(calls).toHaveLength(1);
          const result = calls[0]!;
          expect(result).toMatchObject({
            command: "copy",
            secureContext: false,
            activeGesture: gesture,
            asyncClipboard: false,
          });
          expect(result.text).toEqual(expect.any(String));
          expect(result.text!.length).toBeGreaterThan(0);
          if (!gesture) expect(result.returned).toBe(false);
          await expect(page.getByText("Something went wrong. Please try again.", { exact: true })).toHaveCount(
            result.returned ? 0 : 1
          );
          const copyNotice = page
            .getByRole("region", { name: "Notifications", exact: true })
            .getByRole("dialog")
            .filter({
              has: page.getByText(
                /^(Link copied!?|Copied!|(?:Work item|Comment) (link|ID|title|URL) copied to clipboard\.?)$/
              ),
            });
          await expect(copyNotice).toHaveCount(result.returned ? 1 : 0);
          await c12Evidence(
            page,
            "B20",
            `${control}-${gesture ? "keyboard" : "no-gesture"}-${result.returned ? "copied" : "refused"}`
          );
          await expect(page).toHaveURL(before);
          expect(
            watch.apiRequests.slice(requestsBefore).filter((request) => /^(POST|PATCH|DELETE) /.test(request))
          ).toEqual([]);
          await info.attach("lan-http-clipboard", {
            body: JSON.stringify({ origin: lan.origin, control, gesture, native: result }, null, 2),
            contentType: "application/json",
          });
          const loginPath = ["layout", "peek", "creation"].includes(control)
            ? seed.path
            : `/${seed.slug}/browse/WEB-${source.sequence_id}`;
          const warnings = expectLANClipboardWarnings(
            watch,
            browserName,
            lan.origin,
            signInPath(loginPath),
            !result.returned,
            result.diagnosticSource
          );
          await expectListPageQuiet(page, watch, { warnings });
        } catch (error) {
          await info.attach("lan-http-failure", {
            body: await page.locator("body").ariaSnapshot(),
            contentType: "text/plain",
          });
          await info.attach("lan-http-failure-screen", { body: await page.screenshot(), contentType: "image/png" });
          throw error;
        } finally {
          await lan.close();
        }
      }
    );
  }
}

test("C7: top navigation keeps a multi-select command open for confirmed label changes and respects IME", async ({
  api,
  page,
  db,
}, info) => {
  const seed = await issueListProject(api, info);
  const [alpha, beta] = await Promise.all([
    createLabel(api, seed.member.token, seed.project.id, { name: "Alpha label" }),
    createLabel(api, seed.member.token, seed.project.id, { name: "Beta label" }),
  ]);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Top command labels" });
  const watch = await openIssueDetail(page, seed, issue);
  const input = page.getByPlaceholder("Search commands...", { exact: true });
  await input.click();
  await selectCommand(page, /^Add labels/, input);
  await input.fill("Alpha");
  const path = `/api/v0/issues/${issue.id}`;
  await input.dispatchEvent("keydown", { key: "Enter", isComposing: true, bubbles: true });
  expect(watch.apiRequests.filter((item) => item === `PATCH ${path}`)).toHaveLength(0);
  const first = await sentHeld(page, "PATCH", path, () => selectCommand(page, /^Alpha label$/, input));
  try {
    expect(first.body).toEqual({ label_ids: [alpha.id] });
    await expect(page.getByRole("option", { name: "Alpha label", exact: true })).toBeVisible();
  } finally {
    expect((await first.release()).status()).toBe(200);
  }
  await expect
    .poll(async () =>
      (
        await db.query<{ id: string }>(
          "SELECT label_id::text AS id FROM issue_labels WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY label_id",
          [issue.id]
        )
      ).map((row) => row.id)
    )
    .toEqual([alpha.id]);
  await input.fill("Beta");
  const second = await sentHeld(page, "PATCH", path, () => selectCommand(page, /^Beta label$/, input));
  try {
    expect(second.body).toEqual({ label_ids: [alpha.id, beta.id] });
    await expect(page.getByRole("option", { name: "Beta label", exact: true })).toBeVisible();
  } finally {
    expect((await second.release()).status()).toBe(200);
  }
  await expect
    .poll(async () =>
      (
        await db.query<{ id: string }>(
          "SELECT label_id::text AS id FROM issue_labels WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY label_id",
          [issue.id]
        )
      ).map((row) => row.id)
    )
    .toEqual([alpha.id, beta.id].toSorted());
  await expect(page.getByRole("option", { name: "Beta label", exact: true })).toBeVisible();
  await c12Evidence(page, "B19", "top-command-multi-label-confirmed");
  await input.press("Escape");
  await expect(page.locator("[cmdk-list]")).toHaveCount(0);
  await expect(input).toHaveValue("");
  expect(watch.apiRequests.filter((item) => item === `PATCH ${path}`)).toHaveLength(2);
  await input.click();
  await expect(page.getByRole("option", { name: /^Add labels/ })).toBeVisible();
  await input.press("Backspace");
  await expect(page.getByRole("option", { name: /^Add labels/ })).toHaveCount(0);
  await input.press("Escape");
  await input.click();
  await expect(page.getByRole("option", { name: /^Add labels/ })).toBeVisible();
  await input.press("Escape");
  await expectListPageQuiet(page, watch);
});
