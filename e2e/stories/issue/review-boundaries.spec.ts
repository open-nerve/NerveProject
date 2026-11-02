import {
  addProjectMembers,
  changeProject,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  readProjectPreferences,
} from "../../fixtures/api";
import { accountId, bearer, emailFor, newAccount } from "../../fixtures/auth";
import { createCommentReaction, createIssueComment, createIssueReaction } from "../../fixtures/comments";
import { c12Evidence } from "../../fixtures/c12-evidence";
import { deferred } from "../../fixtures/deferred";
import { modalDialog } from "../../fixtures/dialogs";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import {
  expectListPageQuiet,
  issueListProject,
  openIssueList,
  saveListPreferences,
} from "../../fixtures/issue-list-page";
import { createIssue } from "../../fixtures/issues";
import { answerTo, holdAnswer, moveWithinApp, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

test.describe("New York date boundaries", () => {
  test.use({ timezoneId: "America/New_York" });
  test("C12 (review): relative and custom child dates retain local inclusive boundaries and single-date freedom", async ({
    api,
    page,
  }, info) => {
    await page.clock.setFixedTime(new Date("2026-10-30T12:00:00-04:00"));
    const seed = await issueListProject(api, info);
    const parent = await createIssue(api, seed.member.token, seed.project.id, { name: "Local date parent" });
    const children = await ["2026-11-05", "2026-11-06", "2026-11-07", "2026-12-15", "2026-12-16"].reduce(
      async (prior, date) => {
        const rows = await prior;
        return [
          ...rows,
          await createIssue(api, seed.member.token, seed.project.id, {
            name: `Due ${date}`,
            parent_id: parent.id,
            target_date: date,
          }),
        ];
      },
      Promise.resolve([] as Awaited<ReturnType<typeof createIssue>>[])
    );
    const watch = await openIssueDetail(page, seed, parent);
    const filters = page.getByRole("button", { name: "Filters", exact: true }).first();
    await expect(page.locator(`#issue-${children[0]!.id}`)).toBeVisible();
    await filters.click();
    await page.getByText("1 week from now", { exact: true }).last().click();
    await page.keyboard.press("Escape");
    await expect(page.locator(`#issue-${children[0]!.id}`)).toHaveCount(0);
    await expect(page.locator(`#issue-${children[1]!.id}`)).toBeVisible();
    await expect(page.locator(`#issue-${children[2]!.id}`)).toBeVisible();
    await c12Evidence(page, "B12", "review-new-york-relative-inclusion");
    await filters.click();
    await page.getByText("1 week from now", { exact: true }).last().click();
    await page.getByText("Custom", { exact: true }).last().click();
    const modal = page.getByRole("dialog", { name: "Due date", exact: true });
    await modal.locator("select.rdp-months_dropdown").first().selectOption("11");
    const day = modal.locator('[data-day="2026-12-15"] button').first();
    await expect(day).toBeDisabled();
    await modal.getByRole("button", { name: "Due date range", exact: true }).click();
    await page.getByRole("option", { name: "Due date before", exact: true }).click();
    await expect(day).toBeEnabled();
    await day.click();
    await c12Evidence(page, "B12", "review-before-beyond-hidden-end");
    await modal.getByRole("button", { name: "Apply", exact: true }).click();
    await page.keyboard.press("Escape");
    await expect(page.locator(`#issue-${children[3]!.id}`)).toBeVisible();
    await expect(page.locator(`#issue-${children[4]!.id}`)).toHaveCount(0);
    // Remove the custom before condition, then choose the corresponding after boundary.
    await filters.click();
    await page.getByText("Custom", { exact: true }).last().click();
    await page.getByText("Custom", { exact: true }).last().click();
    await modal.getByRole("button", { name: "Due date range", exact: true }).click();
    await page.getByRole("option", { name: "Due date after", exact: true }).click();
    await modal.locator("select.rdp-months_dropdown").selectOption("11");
    await expect(day).toBeEnabled();
    await day.click();
    await modal.getByRole("button", { name: "Apply", exact: true }).click();
    await page.keyboard.press("Escape");
    await expect(page.locator(`#issue-${children[2]!.id}`)).toHaveCount(0);
    await expect(page.locator(`#issue-${children[3]!.id}`)).toBeVisible();
    await expect(page.locator(`#issue-${children[4]!.id}`)).toBeVisible();
    await c12Evidence(page, "B12", "review-after-inclusive-boundary");
    await page.clock.setFixedTime(new Date("2026-11-02T12:00:00-05:00"));
    await filters.click();
    await page.getByText("Custom", { exact: true }).last().click();
    await page.getByText("Custom", { exact: true }).last().click();
    await expect(modal.locator('[data-day="2026-11-02"]')).toHaveAttribute("data-selected", "true");
    await c12Evidence(page, "B12", "review-new-day-new-filter-default");
    await modal.getByRole("button", { name: "Cancel", exact: true }).click();
    await expectListPageQuiet(page, watch);
  });

  for (const date of ["0001-01-01", "0099-12-31"]) {
    test(`C12 (review): an early-year date ${date} survives another field save and can be cleared without a 1900-year offset`, async ({
      api,
      db,
      page,
    }, info) => {
      const seed = await issueListProject(api, info);
      const issue = await createIssue(api, seed.member.token, seed.project.id, {
        name: "Early date",
        target_date: date,
      });
      const watch = await openIssueDetail(page, seed, issue);
      const property = page.getByText("Due date", { exact: true }).locator("../..");
      const control = property.getByRole("button");
      await expect(control).toContainText(date.startsWith("0001") ? "0001" : "0099");
      // The shared calendar deliberately offers only +/-30 years. Preserve an API-supplied
      // historic date while editing another field; do not pretend its calendar can choose year 1.
      const title = page.getByRole("textbox", { name: "Work item title", exact: true });
      await expect(page.getByText("created the work item.", { exact: true })).toBeVisible();
      // Saving refreshes both collaboration collections. Observe those reads before the
      // write, then consume their bodies before reloading this document.
      const refreshed = ["activities", "comments"].map((collection) =>
        page.waitForResponse(
          (response) =>
            response.request().method() === "GET" &&
            new URL(response.url()).pathname === `/api/v0/issues/${issue.id}/${collection}`
        )
      );
      expect(
        (await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () => title.fill("Early date retained"))).status()
      ).toBe(200);
      await Promise.all(
        refreshed.map(async (answer) => {
          const response = await answer;
          expect(response.status()).toBe(200);
          expect(await response.finished()).toBeNull();
        })
      );
      expect(await db.query("SELECT target_date::text FROM issues WHERE id=$1", [issue.id])).toEqual([
        { target_date: date },
      ]);
      await page.reload();
      await expect(title).toHaveValue("Early date retained");
      await expect(control).toContainText(date.startsWith("0001") ? "0001" : "0099");
      await c12Evidence(page, "B10", `review-early-year-${date.slice(0, 4)}`);
      await property.hover();
      expect(
        (
          await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, () =>
            property.getByRole("button", { name: "Clear date: Add due date", exact: true }).click()
          )
        ).status()
      ).toBe(200);
      expect(await db.query("SELECT target_date FROM issues WHERE id=$1", [issue.id])).toEqual([{ target_date: null }]);
      await expectListPageQuiet(page, watch);
    });
  }
});

test("C12 (review): mobile layout, display and grouping refusals are reported once and successful retry follows confirmed preferences", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Mobile preference row" });
  const prefs = await readProjectPreferences(api, seed.member.token, seed.project.id);
  await saveListPreferences(api, seed, {
    display_filters: { ...prefs.display_filters, layout: "list", group_by: null },
  });
  await page.setViewportSize({ width: 390, height: 844 });
  const { watch, queries } = await openIssueList(page, seed);
  // The URL can settle before lazy route modules finish loading. Reach the actual
  // row before routing disables the HTTP cache used by those module preloads.
  await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
  const path = `/api/v0/me/projects/${seed.project.id}/preferences`;
  let refuse = true;
  await page.route(`**${path}`, async (route) => {
    if (route.request().method() !== "PATCH" || !refuse) return route.fallback();
    await route.fulfill({
      status: 403,
      contentType: "application/problem+json",
      body: JSON.stringify({ type: "about:blank", title: "Forbidden", status: 403, code: "forbidden" }),
    });
  });
  const actions = [
    async () => {
      await page.getByRole("button", { name: "Layout", exact: true }).click();
      await page.getByRole("menuitem", { name: "Board", exact: true }).click();
    },
    async () => {
      await page.getByRole("button", { name: "Display", exact: true }).click();
      await page
        .getByRole("button", { name: "Display Properties", exact: true })
        .locator("../..")
        .getByRole("button", { name: "Priority", exact: true })
        .click();
    },
    async () => {
      await page.getByRole("button", { name: "Display", exact: true }).click();
      await page
        .getByRole("button", { name: "Group by", exact: true })
        .locator("../..")
        .getByRole("button", { name: "Priority", exact: true })
        .click();
    },
  ];
  await actions.reduce(async (prior, action, index) => {
    await prior;
    expect((await answerTo(page, "PATCH", path, action)).status()).toBe(403);
    await expect(page.getByText("Your role does not allow this.", { exact: true })).toHaveCount(1);
    await c12Evidence(page, "B07", `review-mobile-refusal-${index}`);
    await page.keyboard.press("Escape");
    // Menu selection restores focus on a later frame. Wait for that close
    // postcondition before handing focus to the toast viewport with F6.
    await expect(page.getByRole("button", { name: index === 0 ? "Layout" : "Display", exact: true })).toBeFocused();
    await page.keyboard.press("F6");
    await page
      .getByRole("region", { name: "Notifications", exact: true })
      .getByRole("button", { name: "Close", exact: true })
      .click();
    await expect(page.getByText("Your role does not allow this.", { exact: true })).toHaveCount(0);
    expect(await readProjectPreferences(api, seed.member.token, seed.project.id)).toMatchObject({
      display_filters: { layout: "list", group_by: null },
      display_properties: prefs.display_properties,
    });
  }, Promise.resolve());
  refuse = false;
  expect((await answerTo(page, "PATCH", path, actions[2]!)).status()).toBe(200);
  await page.keyboard.press("Escape");
  await expect.poll(() => queries.at(-1)?.get("group_by")).toBe("priority");
  await c12Evidence(page, "B01", "review-mobile-confirmed-grouping");
  await expectListPageQuiet(page, watch, {
    failures: Array(3).fill(`403 PATCH ${path}`),
    errors:
      browserName === "firefox"
        ? []
        : Array(3).fill("Failed to load resource: the server responded with a status of 403 (Forbidden)"),
  });
});

for (const language of ["en", "zh-CN"] as const) {
  test(`C12 (review): ${language} feedback exposes reaction counts, pressed state, tooltip and a working named toast close`, async ({
    api,
    page,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    if (language === "zh-CN")
      expect(
        (await api.PATCH("/api/v0/me/profile", { headers: bearer(seed.member.token), body: { language } })).response
          .status
      ).toBe(200);
    const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Reaction feedback" });
    const comment = await createIssueComment(api, seed.member.token, issue.id, "<p>Comment reaction</p>");
    await createIssueReaction(api, seed.member.token, issue.id);
    await createCommentReaction(api, seed.member.token, comment.id);
    await [0, 1, 2, 3, 4].reduce(async (prior, index) => {
      await prior;
      const person = await newAccount(api, info, `reaction-${index}`);
      await inviteAndAccept(api, seed.member.token, seed.slug, person, 15);
      await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: person.id, role: 15 }]);
      await createIssueReaction(api, person.token, issue.id);
    }, Promise.resolve());
    // The ordinary sign-in helper's title lookup is translated; use the read-only text argument to select the value host.
    const watch =
      language === "en" ? await openIssueDetail(page, seed, issue) : (await openIssueList(page, seed)).watch;
    if (language === "zh-CN") await moveWithinApp(page, `/${seed.slug}/browse/WEB-${issue.sequence_id}`);
    const reaction = page.getByRole("button", { name: "👍 6", exact: true });
    await expect(reaction).toHaveAttribute("aria-pressed", "true");
    const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
    await reaction.focus();
    await page.keyboard.press(tab);
    await page.keyboard.press(`Shift+${tab}`);
    await expect(reaction).toBeFocused();
    await expect(reaction).toHaveCSS("outline-style", /^(auto|solid)$/);
    const tooltip = page.getByRole("tooltip").filter({ hasText: "and 1 more" });
    await expect(tooltip).toBeVisible();
    expect((await reaction.getAttribute("aria-describedby"))?.split(" ")).toContain(await tooltip.getAttribute("id"));
    await c12Evidence(page, "B16", `review-${language}-reaction-tooltip`);
    await page.keyboard.press("Escape");
    await expect(tooltip).toHaveCount(0);
    await page.locator(`#comment-${comment.id}`).getByRole("button", { name: "👍 1", exact: true }).focus();
    await expect(page.getByRole("tooltip")).toBeVisible();
    await c12Evidence(page, "B16", `review-${language}-comment-tooltip`);
    await page.keyboard.press("Escape");
    const path = `/api/v0/issues/${issue.id}/reactions/128077`;
    expect((await answerTo(page, "DELETE", path, () => reaction.click())).status()).toBe(204);
    await expect(page.getByRole("button", { name: "👍 5", exact: true })).toHaveAttribute("aria-pressed", "false");
    // A real copy action supplies a toast; only the clipboard boundary is substituted.
    await page.evaluate(() =>
      Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: async () => {} } })
    );
    await page.getByRole("button", { name: language === "en" ? "Copy link" : "复制链接", exact: true }).click();
    // Base UI exposes toast actions after keyboard focus enters the viewport (F6).
    await page.keyboard.press("F6");
    const close = page
      .getByRole("region")
      .getByRole("button", { name: language === "en" ? "Close" : "关闭", exact: true });
    await expect(close).toHaveCount(1);
    // F6 focuses the region; its first Tab enters the toast root, the next reaches Close.
    await page.keyboard.press(tab);
    await page.keyboard.press(tab);
    await expect(close).toBeFocused();
    await expect(close).toHaveCSS("outline-style", /^(auto|solid)$/);
    await c12Evidence(page, "B21", `review-${language}-toast-close`);
    await page.keyboard.press("Enter");
    await expect(close).toHaveCount(0);
    await expectListPageQuiet(page, watch);
  });
}

test("C12 (review): a full catalog revokes a private hot project's visibility before an old issue read returns", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  expect((await changeProject(api, seed.member.token, seed.project.id, { network: 0 })).status).toBe(200);
  const email = emailFor(info, "private-reader");
  const tokens = await registerOnboarded(api, email);
  const member = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, member, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: member.id, role: 15 }]);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Private hot facts" });
  await page.clock.install();
  const { watch } = await openIssueList(page, { ...seed, member });
  const trigger = page.locator(`#issue-${issue.id}`);
  await expect(trigger).toBeVisible();
  const request = deferred();
  const release = await holdAnswer(page, "GET", `/api/v0/issues/${issue.id}`, 1, request.resolve);
  await trigger.click();
  await request.promise;
  expect(
    await projectMemberWrites(api, seed.project.id).remove(
      seed.member.token,
      await projectMembershipOf(api, seed.member.token, seed.project.id, member.id)
    )
  ).toEqual({ status: 204 });
  await moveWithinApp(page, `/${seed.slug}/projects`);
  await expect(page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();
  // Expire only the browser's SWR deduplication window; business data and server time stay real.
  await page.clock.fastForward(3_000);
  const catalogPath = `/api/v0/workspaces/${seed.slug}/projects`;
  const catalogAnswer = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return (
      response.request().method() === "GET" &&
      url.pathname === catalogPath &&
      url.searchParams.get("archived") === "false"
    );
  });
  await page.keyboard.press("ControlOrMeta+k");
  await expect(modalDialog(page).getByRole("combobox")).toBeFocused();
  const catalog = await catalogAnswer;
  expect(catalog.status()).toBe(200);
  expect((await catalog.json()).data).toEqual([]);
  await expect(
    page.getByRole("complementary", { name: "Main sidebar", exact: true }).locator(`a[href="${seed.path}"]`)
  ).toHaveCount(0);
  expect((await answerTo(page, "GET", `/api/v0/issues/${issue.id}`, release)).status()).toBe(200);
  expect(
    (
      await answerTo(page, "GET", `/api/v0/workspaces/${seed.slug}/issue-search`, () =>
        modalDialog(page).getByRole("combobox").fill(issue.name)
      )
    ).status()
  ).toBe(200);
  await expect(page.getByRole("option").filter({ hasText: issue.name })).toHaveCount(0);
  await c12Evidence(page, "B10", "review-private-catalog-late-issue");
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("complementary", { name: "Main sidebar", exact: true }).locator(`a[href="${seed.path}"]`)
  ).toHaveCount(0);
  await expectListPageQuiet(page, watch);
});
