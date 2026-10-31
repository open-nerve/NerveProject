import { createLabel } from "../../fixtures/api";
import { createIssue, createIssueRelations } from "../../fixtures/issues";
import { createIssueLink } from "../../fixtures/links";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { refuseClipboardWrites } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { answerTo, transitionsEnded } from "../../fixtures/settings-pages";

for (const control of ["identifier", "detail", "link", "relation", "sub-item", "peek"] as const) {
  test(`C4 (page): ${control} copy is keyboard reachable and reports a refused clipboard write without success or navigation`, async ({
    api,
    page,
  }, info) => {
    const seed = await issueListProject(api, info);
    const source = await createIssue(api, seed.member.token, seed.project.id, { name: "Copy source" });
    if (control === "link")
      await createIssueLink(api, seed.member.token, source.id, { title: "Guide", url: "https://example.test/guide" });
    if (control === "relation") {
      const target = await createIssue(api, seed.member.token, seed.project.id, { name: "Related target" });
      await createIssueRelations(api, seed.member.token, source.id, {
        relation_type: "relates_to",
        issue_ids: [target.id],
      });
    }
    if (control === "sub-item")
      await createIssue(api, seed.member.token, seed.project.id, { name: "Child target", parent_id: source.id });

    const watch =
      control === "peek" ? (await openIssueList(page, seed)).watch : await openIssueDetail(page, seed, source);
    if (control === "peek") {
      await page.locator(`#issue-${source.id}`).click();
      await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
    }
    if (control === "relation") await page.getByRole("button", { name: "Relations 1", exact: true }).click();
    await refuseClipboardWrites(page);
    const before = page.url();
    const requestsBeforeCopy = watch.apiRequests.length;
    let copy;
    if (control === "relation" || control === "sub-item") {
      const options = page.getByRole("button", {
        name: `${control === "relation" ? "Related target" : "Child target"} options`,
        exact: true,
      });
      await options.focus();
      await expect(options).toBeFocused();
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
    await page.keyboard.press("Enter");
    await expect(page.getByText("Something went wrong. Please try again.", { exact: true })).toHaveCount(1);
    await expect(page.getByText(/^(Link copied|Work item (link|ID) copied to clipboard)$/)).toHaveCount(0);
    await expect(page).toHaveURL(before);
    if (control !== "peek")
      await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
    expect(
      watch.apiRequests.slice(requestsBeforeCopy).filter((request) => /^(POST|PATCH|DELETE) /.test(request))
    ).toEqual([]);
    await expectListPageQuiet(page, watch);
  });
}

test("C4: creation owns dismissal, nested draft confirmation owns its Escape and Tab loop, and date/label portals retain the outer form and its focus", async ({
  api,
  db,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  await createLabel(api, seed.member.token, seed.project.id, { name: "Visible label" });
  const { watch } = await openIssueList(page, seed);
  const create = page.getByRole("button", { name: "Add work item", exact: true });
  await create.focus();
  await page.keyboard.press("Enter");
  const outer = page.getByRole("dialog", { name: "Create new work item", exact: true });
  await expect(outer).toHaveCount(1);
  await expect(outer.getByRole("textbox", { name: "Title", exact: true })).toBeVisible();
  await transitionsEnded(outer);
  expect(await pageBehindHidden(page)).toBe(true);

  const date = outer.getByRole("button", { name: "Start date", exact: true });
  await date.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "Start date", exact: true })).toBeVisible();
  expect(await pageBehindHidden(page)).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "Start date", exact: true })).toHaveCount(0);
  await expect(modalDialog(page)).toHaveCount(1);
  await expect(date).toBeFocused();
  const labels = outer.getByRole("button", { name: "Labels", exact: true });
  await labels.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("option", { name: "Visible label", exact: true })).toBeVisible();
  expect(await pageBehindHidden(page)).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(modalDialog(page)).toHaveCount(1);
  await expect(labels).toBeFocused();
  // A clean form leaves directly, through the same decision used by Escape and its Discard button.
  await page.mouse.click(1, 1);
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(create).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(false);

  await create.click();
  const title = outer.getByRole("textbox", { name: "Title", exact: true });
  await title.fill("Keep this input");
  const discard = outer.getByRole("button", { name: "Discard", exact: true });
  await discard.focus();
  await page.keyboard.press("Enter");
  const inner = page.getByRole("dialog", { name: "Save this draft?", exact: true });
  await expect(inner).toHaveCount(1);
  await transitionsEnded(inner);
  expect(await pageBehindHidden(page)).toBe(true);
  const first = inner.getByRole("button", { name: "Discard", exact: true });
  const last = inner.getByRole("button", { name: "Save to Drafts", exact: true });
  await first.focus();
  await page.keyboard.press("Shift+Tab");
  await expect(last).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(first).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(inner).toHaveCount(0);
  await expect(modalDialog(page)).toHaveCount(1);
  await expect(title).toHaveValue("Keep this input");
  await expect(discard).toBeFocused();

  // The inner backdrop cancels only the confirmation; the dirty outer still owns the draft.
  await page.mouse.click(1, 1);
  await expect(inner).toHaveCount(1);
  await transitionsEnded(inner);
  await page.mouse.click(1, 1);
  await expect(inner).toHaveCount(0);
  await expect(modalDialog(page)).toHaveCount(1);
  await expect(title).toHaveValue("Keep this input");
  await title.focus();
  await expect(title).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(inner).toHaveCount(1);
  await inner.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(create).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(false);
  expect(
    watch.apiRequests.filter((request) => request.startsWith("POST /api/v0/projects/") || request.includes("/drafts"))
  ).toEqual([]);
  expect(await db.query("SELECT id FROM issues WHERE project_id=$1", [seed.project.id])).toEqual([]);
  await expectListPageQuiet(page, watch);
});

for (const dirty of [false, true]) {
  test(`C4: native parent search owns only its inner Escape and restores a ${dirty ? "dirty" : "clean"} creation form`, async ({
    api,
    page,
  }, info) => {
    const seed = await issueListProject(api, info);
    await createIssue(api, seed.member.token, seed.project.id, { name: "Candidate parent" });
    const { watch } = await openIssueList(page, seed);
    const create = page.getByRole("button", { name: "Add work item", exact: true });
    await create.click();
    const outer = page.getByRole("dialog", { name: "Create new work item", exact: true });
    const title = outer.getByRole("textbox", { name: "Title", exact: true });
    await expect(title).toBeVisible();
    if (dirty) await title.fill("Unsaved child");
    const parent = outer.getByRole("button", { name: "Add parent", exact: true });
    await parent.focus();
    const found = await answerTo(page, "GET", `/api/v0/workspaces/${seed.slug}/issue-search`, () =>
      page.keyboard.press("Enter")
    );
    expect(Object.fromEntries(new URL(found.url()).searchParams)).toEqual({ project_id: seed.project.id });
    const inner = page.getByRole("dialog", { name: "Add parent work item", exact: true });
    await expect(inner.getByRole("option").filter({ hasText: "Candidate parent" })).toBeVisible();
    expect(await pageBehindHidden(page)).toBe(true);
    await inner.getByRole("combobox").focus();
    await page.keyboard.press("Escape");
    await expect(modalDialog(page)).toHaveCount(1);
    await expect(inner).toHaveCount(0);
    await expect(parent).toBeFocused();
    await expect(title).toHaveValue(dirty ? "Unsaved child" : "");
    await expect(page.getByRole("dialog", { name: "Save this draft?", exact: true })).toHaveCount(0);
    await outer.getByRole("button", { name: "Discard", exact: true }).click();
    if (dirty)
      await page
        .getByRole("dialog", { name: "Save this draft?", exact: true })
        .getByRole("button", { name: "Discard", exact: true })
        .click();
    await expect(modalDialog(page)).toHaveCount(0);
    await expect(create).toBeFocused();
    expect(watch.apiRequests.filter((request) => request.startsWith("POST /api/v0/projects/"))).toEqual([]);
    await expectListPageQuiet(page, watch);
  });
}
