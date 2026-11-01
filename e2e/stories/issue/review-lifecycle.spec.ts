import { addProjectMembers, changeProject, inviteAndAccept } from "../../fixtures/api";
import { bearer, newAccount } from "../../fixtures/auth";
import { c12Evidence } from "../../fixtures/c12-evidence";
import { deferred } from "../../fixtures/deferred";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { createIssue, issueArchived, issueRead } from "../../fixtures/issues";
import { answerTo, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

for (const status of [403, 404]) {
  test(`C12 (review): a hot ID remains at its address when the current read answers ${status}`, async ({
    api,
    page,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Hot ID refusal" });
    const { watch } = await openIssueList(page, seed);
    const idPath = `${seed.path}/${issue.id}`;
    await moveWithinApp(page, idPath);
    await expect(page).toHaveURL(`/${seed.slug}/browse/WEB-${issue.sequence_id}`);
    await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
    await moveWithinApp(page, seed.path);
    await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
    const path = `/api/v0/issues/${issue.id}`;
    const received = deferred();
    const released = deferred();
    await page.route(`**${path}`, async (route) => {
      if (route.request().method() !== "GET") return route.fallback();
      received.resolve();
      await released.promise;
      await route.fulfill({
        status,
        contentType: "application/problem+json",
        body: JSON.stringify({
          type: "about:blank",
          title: status === 403 ? "Forbidden" : "Not Found",
          status,
          code: status === 403 ? "forbidden" : "not_found",
        }),
      });
    });
    try {
      await moveWithinApp(page, idPath);
      await received.promise;
      await expect(page).toHaveURL(idPath);
      expect((await answerTo(page, "GET", path, async () => released.resolve())).status()).toBe(status);
      await expect(page.getByRole("button", { name: "Retry", exact: true })).toBeVisible();
      await expect(page).toHaveURL(idPath);
      await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveCount(0);
      await c12Evidence(page, "B10", `review-hot-id-${status}`);
      await expectListPageQuiet(page, watch, {
        failures: [`${status} GET ${path}`],
        errors:
          browserName === "firefox"
            ? []
            : [
                `Failed to load resource: the server responded with a status of ${status} (${status === 403 ? "Forbidden" : "Not Found"})`,
              ],
      });
    } finally {
      released.resolve();
    }
  });
}

for (const reason of ["archive", "guest"] as const) {
  test(`C12 (review): ${reason} preserves parent navigation and read-only widgets in all three peek layouts`, async ({
    api,
    page,
  }, info) => {
    const seed = await issueListProject(api, info);
    const parent = await createIssue(api, seed.member.token, seed.project.id, { name: "Read-only parent" });
    const issue = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Read-only child",
      parent_id: parent.id,
      state_id: seed.states.Done,
    });
    const child = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Readable descendant",
      parent_id: issue.id,
    });
    let member = seed.member;
    let path = seed.path;
    if (reason === "archive") {
      expect((await issueArchived(api, seed.member.token, issue.id)).status).toBe(200);
      path = `/${seed.slug}/projects/${seed.project.id}/archives/issues`;
    } else {
      member = await newAccount(api, info, "readonly-guest");
      expect(
        (await api.PATCH("/api/v0/me/profile", { headers: bearer(member.token), body: { is_onboarded: true } }))
          .response.status
      ).toBe(200);
      await inviteAndAccept(api, seed.member.token, seed.slug, member, 5);
      await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: member.id, role: 5 }]);
      expect(
        (await changeProject(api, seed.member.token, seed.project.id, { guest_view_all_features: true })).status
      ).toBe(200);
    }
    const { watch } = await openIssueList(page, { ...seed, member, path });
    await page.locator(`#issue-${issue.id}`).click();
    const peek = page.locator("#full-screen-portal");
    await ["Side Peek", "Modal", "Full Screen"].reduce(async (prior, layout) => {
      await prior;
      await peek.getByRole("button", { name: "Toggle peek view layout", exact: true }).click();
      await page.getByRole("option", { name: layout, exact: true }).click();
      await expect(peek.getByText(issue.name, { exact: true })).toBeVisible();
      await expect(peek.getByText(child.name, { exact: true })).toBeVisible();
      await expect(peek.getByRole("button", { name: "State: Done", exact: true })).toBeDisabled();
      await expect(peek.getByRole("button", { name: "State: Backlog", exact: true })).toBeDisabled();
      await expect(peek.getByRole("button", { name: "Add relation", exact: true })).toHaveCount(0);
      await expect(peek.getByRole("button", { name: "Add link", exact: true })).toHaveCount(0);
      await expect(peek.getByRole("button", { name: "Add sub-work item", exact: true })).toHaveCount(0);
      await peek.getByRole("button", { name: "Sibling work items", exact: true }).click();
      await expect(page.getByRole("menu")).toBeVisible();
      await expect(page.getByRole("menuitem", { name: "Remove parent work item", exact: true })).toHaveCount(0);
      await c12Evidence(page, "B12", `review-${reason}-${layout}`);
      await page.keyboard.press("Escape");
    }, Promise.resolve());
    expect(await issueRead(api, seed.member.token, issue.id)).toMatchObject({ issue: { parent_id: parent.id } });
    expect(watch.apiRequests.filter((request) => /^(PATCH|POST|DELETE) \/api\/v0\/issues\//.test(request))).toEqual([]);
    await expectListPageQuiet(page, watch);
  });
}

test("C12 (review): a real description anchor scrolls into view and a retired load leaves the destination alone", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Description anchor",
    description_html:
      Array.from({ length: 50 }, (_, i) => `<p data-id="row-${i}">Paragraph ${i}</p>`).join("") +
      '<p data-id="anchor-target">Anchor target</p>',
  });
  const { watch } = await openIssueList(page, seed);
  const path = `/${seed.slug}/browse/WEB-${issue.sequence_id}#anchor-target`;
  await moveWithinApp(page, path);
  const anchor = page.locator('[data-id="anchor-target"]');
  await expect(anchor).toBeInViewport();
  await c12Evidence(page, "B14", "review-real-description-anchor");
  await moveWithinApp(page, seed.path);
  await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
  const held = await sentHeld(page, "GET", `/api/v0/issues/${issue.id}`, () => moveWithinApp(page, path));
  await moveWithinApp(page, `/${seed.slug}/projects`);
  await expect(page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();
  expect((await held.release()).status()).toBe(200);
  await expect(page).toHaveURL(`/${seed.slug}/projects`);
  await expect(page.locator('[data-id="anchor-target"]')).toHaveCount(0);
  await c12Evidence(page, "B14", "review-retired-description-anchor");
  await expectListPageQuiet(page, watch);
});
