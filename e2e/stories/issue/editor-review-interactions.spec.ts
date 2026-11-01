import type { Locator, Page } from "@playwright/test";
import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { named, newAccount } from "../../fixtures/auth";
import { c12Evidence } from "../../fixtures/c12-evidence";
import { composingKey, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { createIssue, issueArchived, updateIssue } from "../../fixtures/issues";
import { answerTo, transitionsEnded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

async function tabBackTo(page: Page, control: Locator, browserName: string) {
  const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
  await control.focus();
  await page.keyboard.press(tab);
  await page.keyboard.press(`Shift+${tab}`);
  await expect(control).toBeFocused();
  await expect(control).toHaveCSS("opacity", "1");
  await expect(control).toHaveCSS("outline-style", /^(auto|solid)$/);
}

test("C12 (review): keyboard and pointer block actions target their own block, retain focus and disappear in read-only detail", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Block ownership",
    state_id: seed.states.Done,
    description_html: "<p>First block</p><p>Second block</p>",
  });
  const path = `/api/v0/issues/${issue.id}`;
  const migration = page.waitForResponse(
    (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === path
  );
  const watch = await openIssueDetail(page, seed, issue);
  expect((await migration).status()).toBe(200);
  const container = page.locator(`[id="editor-container-${issue.id}"]`);
  const editor = container.locator('[contenteditable="true"]');
  const handle = container.getByRole("button", { name: "Block actions", exact: true });
  const menu = page.getByRole("group", { name: "Block actions", exact: true });
  await editor.getByText("Second block", { exact: true }).click();
  await page.mouse.move(0, 0);
  await tabBackTo(page, handle, browserName);
  await c12Evidence(page, "B14", "review-block-handle-native-tab");
  await page.keyboard.press("Enter");
  await expect(menu).toHaveCount(1);
  const remove = menu.getByRole("button", { name: "Delete", exact: true });
  await expect(remove).toBeFocused();
  await composingKey(remove, "Escape");
  await expect(menu).toBeVisible();
  await c12Evidence(page, "B14", "review-block-menu-ime");
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expect(handle).toBeFocused();
  expect(
    (
      await answerTo(page, "PATCH", path, async () => {
        await page.keyboard.press("Enter");
        await menu.getByRole("button", { name: "Duplicate", exact: true }).click();
      })
    ).status()
  ).toBe(200);
  await expect(editor.locator(":scope > p")).toHaveText(["First block", "Second block", "Second block"]);
  // Pointer focus must not jump from the hovered first block back to the keyboard's prior block.
  await editor.getByText("First block", { exact: true }).hover();
  const firstBox = await editor.getByText("First block", { exact: true }).boundingBox();
  const handleBox = await handle.boundingBox();
  if (!firstBox || !handleBox) throw new Error("missing first block or its handle");
  expect(Math.abs(firstBox.y - handleBox.y)).toBeLessThan(40);
  expect(
    (
      await answerTo(page, "PATCH", path, async () => {
        await handle.click();
        await menu.getByRole("button", { name: "Duplicate", exact: true }).click();
      })
    ).status()
  ).toBe(200);
  await expect(editor.locator(":scope > p")).toHaveText(["First block", "First block", "Second block", "Second block"]);
  expect(await db.query("SELECT description_stripped FROM issues WHERE id=$1", [issue.id])).toEqual([
    { description_stripped: "First blockFirst blockSecond blockSecond block" },
  ]);
  await c12Evidence(page, "B14", "review-pointer-duplicates-first-block");
  expect((await issueArchived(api, seed.member.token, issue.id)).status).toBe(200);
  await page.reload();
  await expect(container.locator('[contenteditable="true"]')).toHaveCount(0);
  await expect(menu).toHaveCount(0);
  await expect(container.getByRole("button", { name: "Block actions", exact: true })).not.toBeVisible();
  await c12Evidence(page, "B14", "review-archived-editor-no-block-action");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): editor suggestions keep composition keys and leave another field's Enter alone", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Suggestion ownership" });
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await editor.fill("/head");
  const slash = page.locator("#slash-command");
  await expect(slash).toBeVisible();
  await composingKey(editor, "Enter");
  await composingKey(editor, "Escape");
  await expect(slash).toBeVisible();
  await expect(editor).toHaveText("/head");
  await c12Evidence(page, "B14", "review-slash-composition");
  const title = page.getByRole("textbox", { name: "Work item title", exact: true });
  await title.focus();
  await title.press("Enter");
  await expect(editor).toHaveText("/head");
  await expect(editor.locator("h1,h2,h3")).toHaveCount(0);
  await editor.focus();
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await expect(slash).toHaveCount(0);
  await expect(editor.locator("h1,h2,h3")).toHaveCount(1);
  await editor.fill(":smi");
  const emoji = page.locator("#emoji-item-0");
  await expect(emoji).toBeVisible();
  await composingKey(editor, "Enter");
  await composingKey(editor, "Escape");
  await expect(emoji).toBeVisible();
  await expect(editor).toHaveText(":smi");
  await c12Evidence(page, "B14", "review-emoji-composition");
  await page.keyboard.press("ArrowDown");
  expect(
    (await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () => page.keyboard.press("Enter"))).status()
  ).toBe(200);
  await expect(emoji).toHaveCount(0);
  await expect(editor.locator('[data-type="emoji"]')).toHaveCount(1);
  await expectListPageQuiet(page, watch);
});

test("C12 (review): a slash default-background command keeps the next text's color through save and reload", async ({
  api,
  db,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Default background",
    description_html: '<p><span data-text-color="gray" data-background-color="peach">Before</span></p>',
  });
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await editor.click();
  await page.keyboard.press("ControlOrMeta+End");
  await page.keyboard.insertText(" /Default");
  const menu = page.locator("#slash-command");
  await expect(menu.getByText("Default background", { exact: true })).toBeVisible();
  await c12Evidence(page, "B14", "review-default-background-menu");
  await menu.getByText("Default background", { exact: true }).click();
  await expect(menu).toHaveCount(0);
  expect(
    (await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () => page.keyboard.insertText("After"))).status()
  ).toBe(200);
  const after = editor.locator("span").filter({ hasText: /^After$/ });
  await expect(after).toHaveAttribute("data-text-color", "gray");
  await expect(after).not.toHaveAttribute("data-background-color");
  await expect
    .poll(
      async () =>
        (await db.query<{ description_html: string }>("SELECT description_html FROM issues WHERE id=$1", [issue.id]))[0]
          ?.description_html
    )
    .toContain('data-text-color="gray">After</span>');
  await page.reload();
  await expect(after).toHaveAttribute("data-text-color", "gray");
  await expect(after).not.toHaveAttribute("data-background-color");
  await c12Evidence(page, "B14", "review-default-background-saved");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): history Markdown copies literal punctuation and nested tasks, then a real paste saves their meaning", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const other = await newAccount(api, info, "markdown-author");
  await named(api, other.token, { first_name: "Ada", last_name: "Team", display_name: "*Ada* [team]" });
  await inviteAndAccept(api, seed.member.token, seed.slug, other, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: other.id, role: 15 }]);
  const original =
    "<p>*literal* **bold** ~~strike~~ `code` [label](https://example.com) &lt;not html&gt;</p>" +
    "<p><em>Italic <code>keep()</code></em> <u><strong>underlined bold</strong></u></p>" +
    `<p><mention-component entity_name="user_mention" entity_identifier="${other.id}"></mention-component></p>` +
    '<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Parent</p><p>Second paragraph</p>' +
    '<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Child</p></li></ul></li></ul>';
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Markdown meaning",
    description_html: original,
  });
  await updateIssue(api, other.token, issue.id, { description_html: "<p>Current revision</p>" });
  const watch = await openIssueDetail(page, seed, issue);
  const calls = await page.evaluateHandle(() => {
    const copied: string[] = [];
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (text: string) => void copied.push(text) },
    });
    return copied;
  });
  await page.getByRole("button", { name: /^Last edited by/ }).click();
  await page.getByRole("menuitem").last().click();
  const history = page.getByRole("dialog", { name: "Description history", exact: true });
  await expect(history.getByText("Second paragraph", { exact: true })).toBeVisible();
  await history.getByRole("button", { name: "Copy markdown", exact: true }).click();
  await expect.poll(() => calls.jsonValue()).toHaveLength(1);
  const [markdown] = await calls.jsonValue();
  expect(markdown).toContain("\\*literal\\*");
  expect(markdown).toContain("[x]");
  expect(markdown).toContain("[ ]");
  await c12Evidence(page, "B17", "review-markdown-history");
  await page.keyboard.press("Escape");
  await expect(history).toHaveCount(0);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await editor.click();
  await page.keyboard.press("ControlOrMeta+A");
  // Substitute clipboard data on a synthetic paste event; the mounted parser and save path stay real.
  expect(
    (
      await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () =>
        editor.evaluate((element, text) => {
          const clipboardData = new DataTransfer();
          clipboardData.setData("text/plain", text);
          const paste = new ClipboardEvent("paste", { bubbles: true, cancelable: true });
          // Firefox discards ClipboardEvent constructor data; substitute the data boundary explicitly.
          Object.defineProperty(paste, "clipboardData", { value: clipboardData });
          element.dispatchEvent(paste);
        }, markdown!)
      )
    ).status()
  ).toBe(200);
  const verify = async () => {
    const literal = editor.locator(":scope > p").first();
    await expect(literal).toHaveText("*literal* **bold** ~~strike~~ `code` [label](https://example.com) <not html>");
    await expect(literal.locator("em,strong,s,code")).toHaveCount(0);
    const mention = editor.getByRole("link", { name: "@*Ada* [team]", exact: true });
    await expect(mention).toBeVisible();
    await expect(mention.locator("em,strong")).toHaveCount(0);
    await expect(editor.locator("code")).toHaveText("keep()");
    await expect(editor.locator("strong")).toHaveText("underlined bold");
    await expect(editor.getByRole("checkbox")).toHaveCount(2);
    await expect(editor.getByRole("checkbox").first()).toBeChecked();
    await expect(editor.getByRole("checkbox").last()).not.toBeChecked();
    await expect(editor.getByRole("listitem").first().getByRole("listitem")).toHaveCount(1);
    await expect(editor.getByText("Second paragraph", { exact: true })).toBeVisible();
  };
  await verify();
  await page.reload();
  await verify();
  await c12Evidence(page, "B14", "review-markdown-reloaded");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): creation keeps its drag-disabled editor isolated from the mounted detail editor and preserves IME dismissal", async ({
  api,
  db,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Background editor",
    description_html: "<p>Background stays unchanged</p>",
  });
  const watch = await openIssueDetail(page, seed, issue);
  const background = page.locator(`[id="editor-container-${issue.id}"]`);
  await expect(background.locator('[contenteditable="true"]')).toBeVisible();
  await page.getByRole("button", { name: "Copy link", exact: true }).focus();
  await page.keyboard.press("c");
  const outer = page.getByRole("dialog", { name: "Create new work item", exact: true });
  const editor = outer.locator('[contenteditable="true"]');
  await expect(editor).toBeVisible();
  await editor.fill("Modal block");
  expect(await page.locator('[contenteditable="true"]').count()).toBeGreaterThanOrEqual(2);
  // The actual creation host does not enable dragDrop. It must not borrow the detail's handle or menu.
  await expect(outer.locator("#drag-handle")).toHaveCount(0);
  expect(await background.evaluate((element) => !!element.closest('[inert], [aria-hidden="true"]'))).toBe(true);
  await composingKey(editor, "Escape");
  await expect(outer).toHaveCount(1);
  await expect(editor).toBeVisible();
  await expect(editor).toBeFocused();
  await expect(page.getByRole("dialog", { name: "Save this draft?", exact: true })).toHaveCount(0);
  await expect(page.getByRole("group", { name: "Block actions", exact: true })).toHaveCount(0);
  await c12Evidence(page, "B14", "review-two-editors-creation-isolation");
  // ProseMirror owns Escape in its contenteditable; the outer form's field supplies the dismissal gesture.
  await outer.getByRole("textbox", { name: "Title", exact: true }).focus();
  await page.keyboard.press("Escape");
  const confirmation = page.getByRole("dialog", { name: "Save this draft?", exact: true });
  await expect(confirmation.getByRole("heading", { name: "Save this draft?", exact: true })).toBeVisible();
  await confirmation.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(0);
  expect(await pageBehindHidden(page)).toBe(false);
  expect(await db.query("SELECT description_stripped FROM issues WHERE id=$1", [issue.id])).toEqual([
    { description_stripped: "Background stays unchanged" },
  ]);
  await expectListPageQuiet(page, watch);
});

test("C12 (review): code copy is reachable by native Tab with a named tooltip and copies that code", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Accessible code copy",
    description_html: "<pre><code>const answer = 42;</code></pre>",
  });
  const watch = await openIssueDetail(page, seed, issue);
  // Only the clipboard boundary is substituted; real DOM focus, tooltip and click handler are exercised.
  const calls = await page.evaluateHandle(() => {
    const copied: string[] = [];
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (text: string) => void copied.push(text) },
    });
    return copied;
  });
  const copy = page
    .locator(`[id="editor-container-${issue.id}"]`)
    .getByRole("button", { name: "Copy code", exact: true });
  await page.mouse.move(0, 0);
  await tabBackTo(page, copy, browserName);
  const tooltip = page.getByRole("tooltip", { name: "Copy code", exact: true });
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toHaveAttribute("id", /.+/);
  expect(await copy.getAttribute("aria-describedby")).toBe(await tooltip.getAttribute("id"));
  await c12Evidence(page, "B14", "review-code-copy-keyboard-tooltip");
  await page.keyboard.press("Enter");
  await expect.poll(() => calls.jsonValue()).toEqual(["const answer = 42;"]);
  await expect(page.locator(`[id="editor-container-${issue.id}"] code`)).toHaveText("const answer = 42;");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): named bubble formatting preserves text color while clearing its background", async ({
  api,
  db,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Bubble colors",
    description_html: "<p>Selected words</p>",
  });
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await editor.click();
  await page.keyboard.press("ControlOrMeta+A");
  const bold = page.getByRole("button", { name: "Bold", exact: true });
  await expect(bold).toBeVisible();
  await bold.click();
  await expect(bold).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "Color", exact: true }).click();
  await page.getByRole("button", { name: "Text color: Pink", exact: true }).click();
  await page.getByRole("button", { name: "Background color: Green", exact: true }).click();
  await expect(editor.locator("[data-text-color='pink']")).toBeVisible();
  await expect(editor.locator("[data-background-color='green']")).toBeVisible();
  await c12Evidence(page, "B14", "review-named-color-controls");
  expect(
    (
      await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () =>
        page.getByRole("button", { name: "Clear background color", exact: true }).click()
      )
    ).status()
  ).toBe(200);
  await expect(editor.locator("[data-text-color='pink']")).toBeVisible();
  await expect(editor.locator("[data-background-color='green']")).toHaveCount(0);
  await expect(editor.locator("strong")).toHaveText("Selected words");
  const saved = await db.query<{ description_html: string }>("SELECT description_html FROM issues WHERE id=$1", [
    issue.id,
  ]);
  expect(saved[0]?.description_html).toContain('data-text-color="pink"');
  expect(saved[0]?.description_html).not.toContain('data-background-color="green"');
  await page.keyboard.press("Escape");
  await transitionsEnded(page.locator("body"));
  await c12Evidence(page, "B14", "review-cleared-background-kept-text-color");
  await expectListPageQuiet(page, watch);
});
