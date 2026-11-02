import { c12Evidence } from "../../fixtures/c12-evidence";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { createIssue } from "../../fixtures/issues";
import { answerTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

test("C12 (review): named alignment and link controls preserve selection, keyboard focus and saved content", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Named link controls",
    description_html: "<p>Link words</p>",
  });
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  const path = `/api/v0/issues/${issue.id}`;
  await editor.click();
  await page.keyboard.press("ControlOrMeta+A");
  const center = page.getByRole("button", { name: "Center align", exact: true });
  await expect(center).toBeVisible();
  const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
  await center.focus();
  await page.keyboard.press(tab);
  await page.keyboard.press(`Shift+${tab}`);
  await expect(center).toBeFocused();
  await expect(center).toHaveCSS("outline-style", /^(auto|solid)$/);
  await c12Evidence(page, "B14", "review-alignment-keyboard");
  expect((await answerTo(page, "PATCH", path, () => page.keyboard.press("Enter"))).status()).toBe(200);
  await expect(editor.locator("p")).toHaveCSS("text-align", "center");
  await page.getByRole("button", { name: "Link", exact: true }).click();
  const url = page.getByRole("textbox", { name: "Link URL", exact: true });
  await expect(url).toBeFocused();
  await url.fill("https://example.com/selected");
  await c12Evidence(page, "B14", "review-link-url-name");
  expect((await answerTo(page, "PATCH", path, () => page.keyboard.press("Enter"))).status()).toBe(200);
  await expect(editor.locator('a[href="https://example.com/selected"]')).toHaveText("Link words");
  await page.reload();
  await expect(editor.locator("p")).toHaveCSS("text-align", "center");
  await expect(editor.locator('a[href="https://example.com/selected"]')).toHaveText("Link words");
  await editor.click();
  await page.keyboard.press("ControlOrMeta+A");
  await page.getByRole("button", { name: "Link", exact: true }).click();
  await expect(url).toHaveValue("https://example.com/selected");
  expect(
    (
      await answerTo(page, "PATCH", path, () => page.getByRole("button", { name: "Remove link", exact: true }).click())
    ).status()
  ).toBe(200);
  await expect(editor.locator("a")).toHaveCount(0);
  await expect(editor).toHaveText("Link words");
  await c12Evidence(page, "B14", "review-link-removed-kept-alignment");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): actual plain-text paste still converts bare URLs and emoji shortcodes", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Paste additions" });
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await editor.click();
  expect(
    (
      await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () =>
        editor.evaluate((element) => {
          const clipboardData = new DataTransfer();
          clipboardData.setData("text/plain", "Visit https://example.com and :smile:");
          const paste = new ClipboardEvent("paste", { bubbles: true, cancelable: true });
          // Firefox discards ClipboardEvent constructor data; substitute the data boundary explicitly.
          Object.defineProperty(paste, "clipboardData", { value: clipboardData });
          element.dispatchEvent(paste);
        })
      )
    ).status()
  ).toBe(200);
  await expect(editor.locator('a[href="https://example.com"]')).toBeVisible();
  await expect(editor.locator('[data-type="emoji"]')).toHaveText("😄");
  await page.reload();
  await expect(editor.locator('a[href="https://example.com"]')).toBeVisible();
  await expect(editor.locator('[data-type="emoji"]')).toHaveText("😄");
  await c12Evidence(page, "B14", "review-url-emoji-paste-reopened");
  await expectListPageQuiet(page, watch);
});
