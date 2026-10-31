import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  projectMembershipOf,
  slugFor,
} from "../../fixtures/api";
import { accountId, emailFor, password } from "../../fixtures/auth";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { watchPage } from "../../fixtures/browser";
import { createIssue } from "../../fixtures/issues";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { answerTo, moveWithinApp, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

test("I22 (page): a real creation is committed before another tab signs out and signs a different account in; the old reply cannot toast, navigate, project its hidden row or close the new account's form", async ({
  api,
  db,
  page,
  context,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "next-account");
  const tokens = await registerOnboarded(api, email);
  const next = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, next, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: next.id, role: 15 }]);
  const nextOwn = await createIssue(api, next.token, seed.project.id, { name: "Next account's visible work" });
  expect(
    (await changeProject(api, seed.member.token, seed.project.id, { guest_view_all_features: false })).status
  ).toBe(200);
  expect(
    await projectMemberWrites(api, seed.project.id).change(
      seed.member.token,
      await projectMembershipOf(api, seed.member.token, seed.project.id, next.id),
      5
    )
  ).toEqual({ status: 200 });
  const beta = slugFor(info, "beta");
  await createWorkspace(api, next.token, { name: "Beta", slug: beta });
  const ops = await createProject(api, next.token, beta, { name: "Ops", identifier: "OPS" });
  const nextItem = await createIssue(api, next.token, ops.id, { name: "Next account's own project" });
  const { watch } = await openIssueList(page, seed);
  const second = await context.newPage();
  const secondWatch = await watchPage(second);
  await second.goto(seed.path);
  await expect(second.getByRole("button", { name: "Open workspace switcher", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add work item", exact: true }).click();
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  const oldName = "Held creation belongs only to the former account";
  await form.getByRole("textbox", { name: "Title", exact: true }).fill(oldName);
  const path = `/api/v0/projects/${seed.project.id}/issues`;
  const held = await sentHeld(page, "POST", path, () =>
    form.getByRole("button", { name: "Save", exact: true }).click()
  );
  let released = false;
  try {
    expect(held.body).toMatchObject({ name: oldName });
    await expect
      .poll(() =>
        db.query("SELECT created_by_id::text, updated_by_id::text FROM issues WHERE project_id=$1 AND name=$2", [
          seed.project.id,
          oldName,
        ])
      )
      .toEqual([{ created_by_id: seed.member.id, updated_by_id: seed.member.id }]);
    await expect(form.getByRole("button", { name: "Saving", exact: true })).toBeDisabled();
    await second.getByRole("button", { name: "Open workspace switcher", exact: true }).click();
    expect(
      (
        await answerTo(second, "POST", "/api/v0/auth/logout", () =>
          second.getByRole("menuitem", { name: "Sign out", exact: true }).click()
        )
      ).status()
    ).toBe(204);
    await expect(page).toHaveURL(signInPath(seed.path));
    await expect(second).toHaveURL(signInPath(seed.path));
    await expect(form).toHaveCount(0);
    expect(await submitSignIn(second, next.email, password)).toBe(200);
    await expect(page).toHaveURL(seed.path);
    await expect(page.locator(`#issue-${nextOwn.id}`)).toBeVisible();
    await expect(page.getByText(oldName, { exact: true })).toHaveCount(0);
    await page.getByRole("button", { name: "Open workspace switcher", exact: true }).click();
    await page.getByRole("menu").getByRole("menuitem", { name: /Beta/ }).click();
    await expect(page).toHaveURL(`/${beta}`);
    const betaPath = `/${beta}/projects/${ops.id}/issues`;
    await moveWithinApp(page, betaPath);
    await expect(page.locator(`#issue-${nextItem.id}`)).toBeVisible();
    await page.getByRole("button", { name: "Add work item", exact: true }).click();
    await form.getByRole("textbox", { name: "Title", exact: true }).fill("Keep next account's input");
    expect((await held.release()).status()).toBe(201);
    released = true;
    await expect(form.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Keep next account's input");
    await expect(page).toHaveURL(betaPath);
    await expect(page.getByText("Work item created successfully", { exact: true })).toHaveCount(0);
    await expect(page.getByText(oldName, { exact: true })).toHaveCount(0);
    expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toHaveLength(1);
    expect(await db.query("SELECT name FROM issues WHERE project_id=$1 ORDER BY sequence_id", [ops.id])).toEqual([
      { name: nextItem.name },
    ]);
    await expectListPageQuiet(page, watch);
    await expectListPageQuiet(second, secondWatch);
  } finally {
    if (!released) await held.release();
    await second.close();
  }
});

test("I22 (page): Back across workspace, project and issue ends creation, deletion, link and peek owners; Forward does not revive them", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const item = await createIssue(api, seed.member.token, seed.project.id, { name: "Retired detail" });
  const neighbor = await createIssue(api, seed.member.token, seed.project.id, { name: "Previous detail" });
  const ops = await createProject(api, seed.member.token, seed.slug, { name: "Ops", identifier: "OPS" });
  const other = await createIssue(api, seed.member.token, ops.id, { name: "Other project detail" });
  const beta = slugFor(info, "beta");
  await createWorkspace(api, seed.member.token, { name: "Beta", slug: beta });
  const betaProject = await createProject(api, seed.member.token, beta, { name: "Beta work", identifier: "BETA" });
  const betaItem = await createIssue(api, seed.member.token, betaProject.id, { name: "Beta list is mounted" });
  const { watch } = await openIssueList(page, seed);
  await expect(page.locator(`#issue-${item.id}`)).toBeVisible();
  const detail = `/${seed.slug}/browse/WEB-${item.sequence_id}`;
  const cases = [
    {
      kind: "create",
      previous: `/${beta}/projects/${betaProject.id}/issues`,
      previousItem: betaItem,
      current: seed.path,
    },
    {
      kind: "peek",
      previous: `/${beta}/projects/${betaProject.id}/issues`,
      previousItem: betaItem,
      current: seed.path,
    },
    {
      kind: "delete",
      previous: `/${seed.slug}/browse/WEB-${neighbor.sequence_id}`,
      previousItem: neighbor,
      current: detail,
    },
    { kind: "link", previous: `/${seed.slug}/browse/OPS-${other.sequence_id}`, previousItem: other, current: detail },
  ] as const;
  await cases.reduce(async (previousCase, entry) => {
    await previousCase;
    // pushState changes the address before React commits the route. Establish each actual page, including
    // Back/Forward destinations, so an action cannot open an old host that the pending navigation then retires.
    const expectPreviousPage = async () => {
      await expect(page).toHaveURL(entry.previous);
      if (entry.kind === "create" || entry.kind === "peek")
        await expect(page.locator(`#issue-${entry.previousItem.id}`)).toBeVisible();
      else
        await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(
          entry.previousItem.name
        );
    };
    const expectCurrentPage = async () => {
      await expect(page).toHaveURL(entry.current);
      if (entry.kind === "create" || entry.kind === "peek")
        await expect(page.locator(`#issue-${item.id}`)).toBeVisible();
      else await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(item.name);
    };
    await moveWithinApp(page, entry.previous);
    await expectPreviousPage();
    await moveWithinApp(page, entry.current);
    await expectCurrentPage();
    if (entry.kind === "create") {
      await page.getByRole("button", { name: "Add work item", exact: true }).click();
      await page
        .getByRole("dialog", { name: "Create new work item", exact: true })
        .getByRole("textbox", { name: "Title", exact: true })
        .fill("Retired unsaved input");
    } else if (entry.kind === "peek") {
      await page.locator(`#issue-${item.id}`).click();
      await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
    } else if (entry.kind === "delete") {
      await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
      await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
      await expect(page.getByRole("dialog", { name: "Delete Work item", exact: true })).toHaveCount(1);
    } else {
      await page.getByRole("button", { name: "Add link", exact: true }).click();
      await expect(page.getByRole("dialog", { name: "Add link", exact: true })).toHaveCount(1);
    }
    await page.goBack();
    await expectPreviousPage();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
    await page.goForward();
    await expectCurrentPage();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
  }, Promise.resolve());
  expect(
    watch.apiRequests.filter((request) =>
      /^(POST|PATCH|DELETE) \/api\/v0\/(issues|projects\/[^/]+\/issues)/.test(request)
    )
  ).toEqual([]);
  await expectListPageQuiet(page, watch);
});
