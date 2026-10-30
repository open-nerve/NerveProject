import {
  addProjectMembers,
  createProject,
  createWorkspace,
  inviteAndAccept,
  readProjectPreferences,
  slugFor,
} from "../../fixtures/api";
import {
  changed,
  createdFor,
  expectIssuePage,
  expectIssuePosition,
  expectIssueUpdated,
  madeBy,
} from "../../fixtures/assert/issue";
import { accountId, emailFor, newAccount } from "../../fixtures/auth";
import { createIssue, issueRead, listIssues, updateIssue } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import {
  issueListProject,
  saveListPreferences,
  openIssueList,
  expectListPageQuiet,
  expectListOrder,
} from "../../fixtures/issue-list-page";
import { registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { dragOnto } from "../../fixtures/project-pages";
import { expect, test } from "../../fixtures/test";

test("I6 (API): one state-and-position PATCH moves a card between Done neighbors, completes it at the write and records only one state activity", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const member = await newAccount(api, info, "member");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, member, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
  const states = await stateIds(api, member.token, project.id);
  const todo = states.Todo,
    done = states.Done;
  if (!todo || !done) throw new Error("expected Todo and Done");
  const left = await createIssue(api, admin.token, project.id, { name: "Left", state_id: done });
  const right = await createIssue(api, admin.token, project.id, { name: "Right", state_id: done });
  const move = await createIssue(api, admin.token, project.id, { name: "Move", state_id: todo });
  await updateIssue(api, admin.token, left.id, { sort_order: 100 });
  await updateIssue(api, admin.token, right.id, { sort_order: 300 });
  await updateIssue(api, admin.token, move.id, { sort_order: 900 });
  const read = () => listIssues(api, member.token, project.id, { group_by: "state_id", order_by: "sort_order" });
  expect((await read()).groups?.map((g) => [g.key, g.total])).toEqual(
    [
      [todo, 1],
      [done, 2],
    ].toSorted((a, b) => String(a[0]).localeCompare(String(b[0])))
  );
  const moved = await updateIssue(api, member.token, move.id, { state_id: done, sort_order: 200 });
  expect(moved).toMatchObject({
    state_id: done,
    sort_order: 200,
    completed_at: expect.any(String),
    updated_by_id: member.id,
  });
  expect(await issueRead(api, member.token, move.id)).toMatchObject({ status: 200, issue: moved });
  await expectIssuePosition(db, move.id, { state: done, sort: 200, from: todo, actor: member.id });
  await expectIssueUpdated(db, move.id, {
    ...madeBy(admin.email, "Move"),
    state: "Done",
    completed: true,
    by: member.email,
    activity: [...createdFor(), changed("state", "Todo", "Done", "updated the state to", "Done", "Todo")],
    wrote: 1,
  });
  const after = await read();
  expect(after.total).toBe(3);
  expect(after.groups?.map((g) => [g.key, g.total])).toEqual([[done, 3]]);
  expect(after.groups?.[0]?.data?.map((i) => i.id)).toEqual([left.id, move.id, right.id]);
  expectIssuePage(await listIssues(api, member.token, project.id, { state_id: [todo] }), []);
});

test("I6 (page): a member moves between Done cards once, reorders below a middle card and at the tail, and a refused move stays put after reload", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const admin = seed.member;
  const email = emailFor(info, "board-member");
  const tokens = await registerOnboarded(api, email);
  const member = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  await inviteAndAccept(api, admin.token, seed.slug, member, 15);
  await addProjectMembers(api, admin.token, seed.project.id, [{ member_id: member.id, role: 15 }]);
  const todo = seed.states.Todo!,
    done = seed.states.Done!;
  const left = await createIssue(api, admin.token, seed.project.id, { name: "Left", state_id: done });
  const right = await createIssue(api, admin.token, seed.project.id, { name: "Right", state_id: done });
  const move = await createIssue(api, admin.token, seed.project.id, { name: "Move", state_id: todo });
  await updateIssue(api, admin.token, left.id, { sort_order: 100 });
  await updateIssue(api, admin.token, right.id, { sort_order: 300 });
  await updateIssue(api, admin.token, move.id, { sort_order: 900 });
  const memberSeed = { ...seed, member };
  const prefs = await readProjectPreferences(api, member.token, seed.project.id);
  await saveListPreferences(api, memberSeed, {
    filters: {},
    display_filters: {
      ...prefs.display_filters,
      layout: "kanban",
      group_by: "state_id",
      sub_group_by: null,
      order_by: "sort_order",
      show_empty_groups: true,
    },
  });
  const { watch } = await openIssueList(page, memberSeed);
  const doneColumn = page.locator(`[id^="${done}__"]`);
  const todoColumn = page.locator(`[id^="${todo}__"]`);
  const card = (id: string) => page.locator(`[id^="issue_${id}_"]`);
  await expectListOrder(doneColumn, [left.id, right.id]);
  await expectListOrder(todoColumn, [move.id]);
  const path = `/api/v0/issues/${move.id}`;
  const held = await sentHeld(page, "PATCH", path, () =>
    dragOnto(page, card(move.id), card(move.id), card(right.id), { x: 100, y: 4 })
  );
  expect(held.body).toEqual({ state_id: done, sort_order: 200 });
  await expectListOrder(doneColumn, [left.id, right.id]);
  await expectListOrder(todoColumn, [move.id]);
  expect(await held.release().then((response) => response.status())).toBe(200);
  await expectListOrder(doneColumn, [left.id, move.id, right.id]);
  await expectListOrder(todoColumn, []);
  await expect(
    doneColumn.locator("../..").locator(".sticky").getByText("Done", { exact: true }).locator("..")
  ).toHaveText("Done3");
  await expect(
    todoColumn.locator("../..").locator(".sticky").getByText("Todo", { exact: true }).locator("..")
  ).toHaveText("Todo0");
  expect(watch.apiRequests.filter((request) => request === `PATCH ${path}`)).toHaveLength(1);
  await expectIssuePosition(db, move.id, { state: done, sort: 200, from: todo, actor: member.id });
  await expectIssueUpdated(db, move.id, {
    ...madeBy(admin.email, "Move"),
    state: "Done",
    completed: true,
    by: member.email,
    activity: [...createdFor(), changed("state", "Todo", "Done", "updated the state to", "Done", "Todo")],
    wrote: 1,
  });

  const middleBox = await card(move.id).boundingBox();
  if (!middleBox) throw new Error("missing middle card");
  const reordered = await sentHeld(page, "PATCH", `/api/v0/issues/${left.id}`, () =>
    dragOnto(page, card(left.id), card(left.id), card(move.id), { x: 100, y: middleBox.height - 4 })
  );
  expect(reordered.body).toEqual({ sort_order: 250 });
  await expectListOrder(doneColumn, [left.id, move.id, right.id]);
  expect(await reordered.release().then((response) => response.status())).toBe(200);
  await expectListOrder(doneColumn, [move.id, left.id, right.id]);
  const columnBox = await doneColumn.boundingBox();
  if (!columnBox) throw new Error("missing Done column");
  const tail = await sentHeld(page, "PATCH", `/api/v0/issues/${left.id}`, () =>
    dragOnto(page, card(left.id), card(left.id), doneColumn, { x: 100, y: columnBox.height - 65 })
  );
  expect(tail.body).toEqual({ sort_order: 65835 });
  expect(await tail.release().then((response) => response.status())).toBe(200);
  await expectListOrder(doneColumn, [move.id, right.id, left.id]);

  const refusedPath = `/api/v0/issues/${right.id}`;
  await page.route(`**${refusedPath}`, async (route) => {
    if (route.request().method() !== "PATCH") return route.fallback();
    await route.fulfill({
      status: 403,
      contentType: "application/problem+json",
      body: JSON.stringify({ type: "about:blank", title: "Forbidden", status: 403, code: "forbidden" }),
    });
  });
  await dragOnto(page, card(right.id), card(right.id), todoColumn, { x: 100, y: 60 });
  await expect.poll(() => watch.apiFailures).toEqual([`403 PATCH ${refusedPath}`]);
  await expectListOrder(doneColumn, [move.id, right.id, left.id]);
  await expectListOrder(todoColumn, []);
  await expect(page.getByText("Your role does not allow this.", { exact: true })).toBeVisible();
  await page.reload();
  await expectListOrder(doneColumn, [move.id, right.id, left.id]);
  await expectListOrder(todoColumn, []);
  expect(await issueRead(api, member.token, right.id)).toMatchObject({
    status: 200,
    issue: { state_id: done, sort_order: 300 },
  });
  await expectListPageQuiet(page, watch, browserName, 2, {
    failures: [`403 PATCH ${refusedPath}`],
    errors:
      browserName === "chromium"
        ? ["Failed to load resource: the server responded with a status of 403 (Forbidden)"]
        : [],
  });
});
