import {
  type Api,
  addProjectMembers,
  archiveProject,
  changeRole,
  createProject,
  createLabel,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { accountId, emailFor, named, newAccount, password } from "../../fixtures/auth";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import type { Page, TestInfo } from "@playwright/test";
import { watchPage } from "../../fixtures/browser";
import { addCondition, displayPanel } from "../../fixtures/issue-filters-page";
import { expectListPageQuiet, issueListProject, listGroup, openIssueList } from "../../fixtures/issue-list-page";
import { answerTo, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { createIssueComment } from "../../fixtures/comments";
import { createIssue, listMemberIssues, memberIssuesListed, type MemberIssueQuery } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

test("I19 (API): A reads B's three distinct relations across only A's live projects; A-only and C-only defeat wrong-target substitutions, own-only follows A and workspace guests are refused", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const a = await newAccount(api, info, "a"),
    b = await newAccount(api, info, "b"),
    c = await newAccount(api, info, "c");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await [a, b, c].reduce(async (before, who) => {
    await before;
    await inviteAndAccept(api, admin.token, slug, who, 15);
  }, Promise.resolve());
  const projects: Awaited<ReturnType<typeof createProject>>[] = [];
  await ["Visible", "Second", "Hidden", "Exited", "Archived", "Own"].reduce(async (before, name) => {
    await before;
    const p = await createProject(api, admin.token, slug, { name, identifier: name.toUpperCase(), network: 0 });
    await addProjectMembers(
      api,
      admin.token,
      p.id,
      [b, c, ...(name === "Hidden" ? [] : [a])].map((who) => ({ member_id: who.id, role: 15 as const }))
    );
    projects.push(p);
  }, Promise.resolve());
  const [visible, second, hidden, exited, archived, own] = projects;
  if (!visible || !second || !hidden || !exited || !archived || !own) throw new Error("missing projects");
  const assigned = await createIssue(api, c.token, visible.id, { name: "B assigned", assignee_ids: [b.id] });
  const created = await createIssue(api, b.token, visible.id, { name: "B created", assignee_ids: [c.id] });
  const subscribed = await createIssue(api, c.token, visible.id, { name: "B subscribed" });
  const comment = await createIssueComment(api, b.token, subscribed.id, "<p>Following</p>");
  expect(comment).toMatchObject({ actor_id: b.id, issue_id: subscribed.id });
  const aOnly = await createIssue(api, a.token, visible.id, { name: "A only", assignee_ids: [a.id] });
  const cOnly = await createIssue(api, c.token, visible.id, { name: "C only", assignee_ids: [c.id] });
  const across = await createIssue(api, b.token, second.id, { name: "B across", assignee_ids: [b.id] });
  await [hidden, exited, archived].reduce(async (before, p) => {
    await before;
    await createIssue(api, b.token, p.id, { name: "Not in caller scope", assignee_ids: [b.id] });
  }, Promise.resolve());
  expect(await projectMemberWrites(api, exited.id).leave(a.token)).toEqual({ status: 204 });
  await archiveProject(api, admin.token, archived.id);
  // The automatic subscriptions are intentional, not a fabricated disjoint three-way relation.
  await (
    [
      [assigned, [b.id, c.id]],
      [created, [b.id, c.id]],
      [subscribed, [b.id, c.id]],
      [aOnly, [a.id]],
      [cOnly, [c.id]],
    ] as const
  ).reduce(async (before, [item, subscribers]) => {
    await before;
    expect(
      (
        await db.query<{ subscriber_id: string }>(
          "SELECT subscriber_id::text FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY subscriber_id",
          [item.id]
        )
      ).map((row) => row.subscriber_id)
    ).toEqual([...subscribers].toSorted());
  }, Promise.resolve());
  const expected = {
    assigned: { [a.id]: [aOnly.id], [b.id]: [assigned.id, across.id], [c.id]: [created.id, cOnly.id] },
    created: { [a.id]: [aOnly.id], [b.id]: [created.id, across.id], [c.id]: [assigned.id, subscribed.id, cOnly.id] },
    subscribed: {
      [a.id]: [aOnly.id],
      [b.id]: [assigned.id, created.id, subscribed.id, across.id],
      [c.id]: [assigned.id, created.id, subscribed.id, cOnly.id],
    },
  };
  await (["assigned", "created", "subscribed"] as const).reduce(async (before, relation) => {
    await before;
    await [a, b, c].reduce(async (previous, target) => {
      await previous;
      const page = await listMemberIssues(api, a.token, slug, target.id, { relation });
      expect(page.total).toBe(expected[relation][target.id]!.length);
      expect(page.next_cursor).toBeNull();
      expect(page.data?.map((i) => i.id).toSorted()).toEqual(expected[relation][target.id]!.toSorted());
    }, Promise.resolve());
    const grouped = await listMemberIssues(api, a.token, slug, b.id, { relation, group_by: "project_id" });
    expect(grouped.total).toBe(expected[relation][b.id]!.length);
    expect(grouped.groups?.map((g) => g.key)).toEqual([visible.id, second.id].toSorted());
    expect(
      grouped.groups
        ?.flatMap((g) => g.data ?? [])
        .map((i) => i.id)
        .toSorted()
    ).toEqual(expected[relation][b.id]!.toSorted());
    expectIssuePage(
      await listMemberIssues(api, a.token, slug, b.id, { relation, project_id: [hidden.id, exited.id, archived.id] }),
      []
    );
  }, Promise.resolve());
  const ownItem = await createIssue(api, a.token, own.id, { name: "A own, B assigned", assignee_ids: [b.id] });
  const notOwn = await createIssue(api, c.token, own.id, { name: "C own, B assigned", assignee_ids: [b.id] });
  expect(
    await projectMemberWrites(api, own.id).change(
      admin.token,
      await projectMembershipOf(api, admin.token, own.id, a.id),
      5
    )
  ).toEqual({ status: 200 });
  await (["assigned", "subscribed"] as const).reduce(async (before, relation) => {
    await before;
    const query: MemberIssueQuery = { relation, project_id: [own.id] };
    expectIssuePage(await listMemberIssues(api, a.token, slug, b.id, query), [ownItem.id]);
    expectIssuePage(await listMemberIssues(api, b.token, slug, b.id, query), [notOwn.id, ownItem.id]);
  }, Promise.resolve());
  await changeRole(api, admin.token, slug, a.id, 5);
  await (["assigned", "created", "subscribed"] as const).reduce(async (before, relation) => {
    await before;
    expect(await memberIssuesListed(api, a.token, slug, b.id, { relation })).toEqual({
      status: 403,
      code: "forbidden",
    });
  }, Promise.resolve());
});

async function profileProject(api: Api, info: TestInfo) {
  const seed = await issueListProject(api, info);
  const a = seed.member;
  const onboard = async (label: string) => {
    const email = emailFor(info, label);
    const tokens = await registerOnboarded(api, email);
    const who = { id: await accountId(api, tokens.access_token), email, token: tokens.access_token };
    await named(api, who.token, { first_name: label, last_name: "Profile", display_name: label });
    return who;
  };
  const [b, c] = await Promise.all([onboard("Bob"), onboard("Carol")]);
  await inviteAndAccept(api, a.token, seed.slug, b, 15);
  await inviteAndAccept(api, a.token, seed.slug, c, 20);
  await addProjectMembers(api, a.token, seed.project.id, [
    { member_id: b.id, role: 15 },
    { member_id: c.id, role: 20 },
  ]);
  const projects: Awaited<ReturnType<typeof createProject>>[] = [];
  await ["Second", "Hidden", "Exited", "Archived", "Own"].reduce(async (before, name) => {
    await before;
    const project = await createProject(api, c.token, seed.slug, { name, identifier: name.toUpperCase(), network: 0 });
    await addProjectMembers(api, c.token, project.id, [
      { member_id: b.id, role: 15 },
      ...(name === "Hidden" ? [] : [{ member_id: a.id, role: 20 as const }]),
    ]);
    projects.push(project);
  }, Promise.resolve());
  const [second, hidden, exited, archived, own] = projects;
  if (!second || !hidden || !exited || !archived || !own) throw new Error("missing profile scope projects");
  const label = await createLabel(api, a.token, seed.project.id, { name: "Visible label" });
  const assigned = await createIssue(api, c.token, seed.project.id, {
    name: "B assigned",
    assignee_ids: [b.id],
    priority: "high",
    label_ids: [label.id],
    state_id: seed.states["In Progress"],
  });
  const created = await createIssue(api, b.token, seed.project.id, {
    name: "B created",
    assignee_ids: [c.id],
    priority: "high",
    label_ids: [label.id],
  });
  const subscribed = await createIssue(api, c.token, seed.project.id, {
    name: "B subscribed",
    priority: "high",
    label_ids: [label.id],
  });
  await createIssueComment(api, b.token, subscribed.id, "<p>Following</p>");
  const aOnly = await createIssue(api, a.token, seed.project.id, {
    name: "A only",
    assignee_ids: [a.id],
    priority: "low",
  });
  const cOnly = await createIssue(api, c.token, seed.project.id, {
    name: "C only",
    assignee_ids: [c.id],
    priority: "low",
  });
  const across = await createIssue(api, b.token, second.id, {
    name: "B across",
    assignee_ids: [b.id],
    priority: "medium",
  });
  await [hidden, exited, archived].reduce(async (before, project) => {
    await before;
    await createIssue(api, b.token, project.id, { name: `Hidden scope ${project.name}`, assignee_ids: [b.id] });
  }, Promise.resolve());
  expect(await projectMemberWrites(api, exited.id).leave(a.token)).toEqual({ status: 204 });
  await archiveProject(api, c.token, archived.id);
  const ownItem = await createIssue(api, a.token, own.id, {
    name: "A own B assigned",
    assignee_ids: [b.id],
    priority: "high",
  });
  const notOwn = await createIssue(api, c.token, own.id, {
    name: "C own B assigned",
    assignee_ids: [b.id],
    priority: "high",
  });
  expect(
    await projectMemberWrites(api, own.id).change(c.token, await projectMembershipOf(api, c.token, own.id, a.id), 5)
  ).toEqual({ status: 200 });
  return {
    ...seed,
    a,
    b,
    c,
    label,
    second,
    hidden,
    exited,
    archived,
    own,
    assigned,
    created,
    subscribed,
    aOnly,
    cOnly,
    across,
    ownItem,
    notOwn,
  };
}
function memberPath(slug: string, id: string) {
  return `/api/v0/workspaces/${slug}/members/${id}/issues`;
}
function profileReads(page: Page, path: string) {
  const queries: URLSearchParams[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname === path) queries.push(url.searchParams);
  });
  return queries;
}
async function profileRows(page: Page, ids: readonly string[]) {
  await expect
    .poll(() =>
      page
        .locator('[id^="issue_"]')
        .evaluateAll((rows) => [...new Set(rows.map((row) => row.id.split("_")[1]))].toSorted())
    )
    .toEqual([...ids].toSorted());
}
async function profileGroup(page: Page, name: string, path: string) {
  const panel = await displayPanel(page);
  const group = panel.getByRole("button", { name: "Group by", exact: true }).locator("../..");
  const button = group.getByRole("button", { name, exact: true });
  const answer = await answerTo(page, "GET", path, () => button.click());
  await expect(button).toHaveAttribute("aria-pressed", "true");
  await page.keyboard.press("Escape");
  expect(answer.status()).toBe(200);
  return answer;
}

test("I19 (page): three real relation tabs and both layouts group only the caller's visible scope, including an admin's own-only project", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await profileProject(api, info);
  const { slug, b } = seed;
  const path = memberPath(slug, b.id);
  const queries = profileReads(page, path);
  const { watch } = await openIssueList(page, { ...seed, path: `/${slug}/profile/${b.id}/assigned` });
  const relations = [
    ["Assigned", "assigned", [seed.assigned.id, seed.across.id, seed.ownItem.id]],
    ["Created", "created", [seed.created.id, seed.across.id]],
    [
      "Subscribed",
      "subscribed",
      [seed.assigned.id, seed.created.id, seed.subscribed.id, seed.across.id, seed.ownItem.id],
    ],
  ] as const;
  await relations.reduce(async (before, [title, relation, ids]) => {
    await before;
    if (relation !== "assigned") await page.getByRole("link", { name: title, exact: true }).click();
    await expect(page).toHaveURL(`/${slug}/profile/${b.id}/${relation}`);
    await profileRows(page, ids);
    await (["List Layout", "Board Layout"] as const).reduce(async (prior, layout) => {
      await prior;
      const button = page.getByRole("button", { name: layout, exact: true });
      if ((await button.getAttribute("aria-pressed")) !== "true") {
        expect((await answerTo(page, "GET", path, () => button.click())).status()).toBe(200);
      }
      await [
        ["Project", "project_id"],
        ["State groups", "state_group"],
        ["Priority", "priority"],
        ["Labels", "label_id"],
      ].reduce(async (previous, [name, group]) => {
        await previous;
        const answer = await profileGroup(page, name!, path);
        expect(await answer.json()).toMatchObject({ total: ids.length });
        await profileRows(page, ids);
        expect(Object.fromEntries(queries.at(-1)!)).toMatchObject({ relation, group_by: group });
        await expect(page.getByText(seed.notOwn.name, { exact: true })).toHaveCount(0);
        await expect(page.getByText(/^Hidden scope /)).toHaveCount(0);
      }, Promise.resolve());
    }, Promise.resolve());
  }, Promise.resolve());
  expect(
    watch.apiRequests.filter((request) => request.startsWith("PATCH ") && request.endsWith("/preferences"))
  ).toEqual([]);
  expect(watch.apiRequests.filter((request) => /GET \/api\/v0\/projects\/[^/]+\/labels$/.test(request))).toEqual([]);
  await expectListPageQuiet(page, watch, browserName, 1);
});

test("I19 (page): a created group keeps its relation and project restriction on page two, reload and shared views while another target and account retain their own settings", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await profileProject(api, info);
  const { slug, b, c, member, project } = seed;
  const rows: string[] = [];
  await Array.from({ length: 51 }, (_, index) => index).reduce(async (before, index) => {
    await before;
    rows.push(
      (
        await createIssue(api, b.token, project.id, {
          name: `Created page ${index + 1}`,
          assignee_ids: [c.id],
          priority: "high",
        })
      ).id
    );
  }, Promise.resolve());
  const path = memberPath(slug, b.id);
  const queries = profileReads(page, path);
  const url = `/${slug}/profile/${b.id}/created`;
  const { watch } = await openIssueList(page, { ...seed, path: url });
  await expect(page.getByRole("link", { name: "Created", exact: true })).toHaveAttribute("aria-current", "page");
  await addCondition(page, "Projects");
  await answerTo(page, "GET", path, () => page.getByRole("option", { name: "Web", exact: true }).click());
  await page.keyboard.press("Escape");
  await addCondition(page, "Priority");
  await answerTo(page, "GET", path, () => page.getByRole("option", { name: "High", exact: true }).click());
  await page.keyboard.press("Escape");
  const grouped = await profileGroup(page, "Priority", path);
  expect(await grouped.json()).toMatchObject({ total: 52, groups: [{ key: "high", total: 52 }] });
  const high = listGroup(page, "High");
  await expect(high.locator('[class~="group/list-header"]')).toHaveText("High52");
  await expect(high.locator('[id^="issue_"]')).toHaveCount(50);
  expect(
    (await answerTo(page, "GET", path, () => high.getByRole("button", { name: /^Load more/ }).click())).status()
  ).toBe(200);
  await profileRows(page, [seed.created.id, ...rows]);
  await expect(high.getByRole("button", { name: /^Load more/ })).toHaveCount(0);
  expect(Object.fromEntries(queries.find((query) => query.has("cursor"))!)).toMatchObject({
    relation: "created",
    group_by: "priority",
    group_key: "high",
    project_id: project.id,
    priority: "high",
    limit: "50",
  });
  await page.reload();
  await expect(listGroup(page, "High").locator('[class~="group/list-header"]')).toHaveText("High52");
  expect(Object.fromEntries(queries.at(-1)!)).toMatchObject({
    relation: "created",
    group_by: "priority",
    project_id: project.id,
    priority: "high",
  });
  await page.getByRole("link", { name: "Assigned", exact: true }).click();
  await profileRows(page, [seed.assigned.id]);
  expect(Object.fromEntries(queries.at(-1)!)).toMatchObject({
    relation: "assigned",
    group_by: "priority",
    project_id: project.id,
    priority: "high",
  });
  // The current user's real sidebar route changes the target without injecting preferences.
  await page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: "Your work", exact: true })
    .click();
  await expect(page).toHaveURL(`/${slug}/profile/${member.id}/assigned`);
  await profileRows(page, [seed.aOnly.id]);
  await expect(page.getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);
  await page.goBack();
  await expect(page).toHaveURL(`/${slug}/profile/${b.id}/assigned`);
  await profileRows(page, [seed.assigned.id]);
  await expect(page.getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(2);
  // Sign out and sign in through the real UI in the SAME context; local preferences remain in storage.
  await page.getByRole("button", { name: "Open workspace switcher", exact: true }).click();
  await page.getByRole("menuitem", { name: "Sign out", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "Email", exact: true })).toBeVisible();
  await page.goto(signInPath(`/${slug}/profile/${b.id}/assigned`));
  expect(await submitSignIn(page, c.email, password)).toBe(200);
  await expect(page).toHaveURL(`/${slug}/profile/${b.id}/assigned`);
  await expect(page.locator(`[id^="issue_${seed.across.id}_"]`)).toBeAttached();
  await expect(page.locator(`[id^="issue_${seed.notOwn.id}_"]`)).toBeAttached();
  await expect(page.getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);
  await expectListPageQuiet(page, watch, browserName, 3);
});

test("I19 (page): view and target navigation end the old display panel and late member read", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await profileProject(api, info);
  const path = memberPath(seed.slug, seed.b.id);
  const { watch } = await openIssueList(page, { ...seed, path: `/${seed.slug}/profile/${seed.b.id}/assigned` });
  await profileRows(page, [seed.assigned.id, seed.across.id, seed.ownItem.id]);
  const oldPanel = await displayPanel(page);
  const oldId = await oldPanel.getAttribute("id");
  const held = await sentHeld(page, "GET", path, () =>
    page.getByRole("link", { name: "Created", exact: true }).click()
  );
  await expect(page.locator(`[id="${oldId}"]`)).toHaveCount(0);
  await page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: "Your work", exact: true })
    .click();
  await expect(page).toHaveURL(`/${seed.slug}/profile/${seed.a.id}/assigned`);
  await profileRows(page, [seed.aOnly.id]);
  expect((await held.release()).status()).toBe(200);
  await profileRows(page, [seed.aOnly.id]);
  await expect(page.getByRole("button", { name: "Display", exact: true })).toHaveAttribute("aria-expanded", "false");
  await page.goBack();
  await expect(page).toHaveURL(`/${seed.slug}/profile/${seed.b.id}/created`);
  await profileRows(page, [seed.created.id, seed.across.id]);
  await expect(page.locator(`[id="${oldId}"]`)).toHaveCount(0);
  await expectListPageQuiet(page, watch, browserName, 1);
});

test("I19 (page): a label selected from a visible detail remains a real restriction after a cold profile reload without fetching label directories", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await profileProject(api, info);
  const path = memberPath(seed.slug, seed.b.id);
  const queries = profileReads(page, path);
  const { watch } = await openIssueList(page, { ...seed, path: `/${seed.slug}/profile/${seed.b.id}/assigned` });
  await page.locator(`#issue-${seed.assigned.id}`).click();
  const peek = page.locator("#full-screen-portal");
  const labels = peek.getByRole("button", { name: `Labels: ${seed.assigned.name}`, exact: true });
  await labels.click();
  await expect(page.getByRole("option", { name: seed.label.name, exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await peek.getByRole("button", { name: "Close the peek view", exact: true }).click();
  await addCondition(page, "Label");
  expect(
    (
      await answerTo(page, "GET", path, () => page.getByRole("option", { name: seed.label.name, exact: true }).click())
    ).status()
  ).toBe(200);
  await page.keyboard.press("Escape");
  await profileRows(page, [seed.assigned.id]);
  const labelReads = watch.apiRequests.filter((request) => /GET \/api\/v0\/projects\/[^/]+\/labels$/.test(request));
  expect(labelReads).toEqual([`GET /api/v0/projects/${seed.project.id}/labels`]);
  await page.reload();
  await profileRows(page, [seed.assigned.id]);
  expect(Object.fromEntries(queries.at(-1)!)).toMatchObject({ relation: "assigned", label_id: seed.label.id });
  await expect(page.getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(1);
  expect(watch.apiRequests.filter((request) => /GET \/api\/v0\/projects\/[^/]+\/labels$/.test(request))).toEqual(
    labelReads
  );
  await expectListPageQuiet(page, watch, browserName, 2);
});

test("I19 (page): a workspace guest has neither toolbar nor member issue read on desktop or mobile", async ({
  api,
  signedInPage,
  browserName,
}, info) => {
  const seed = await profileProject(api, info);
  const tokens = await registerOnboarded(api, emailFor(info, "profile-guest"));
  await inviteAndAccept(
    api,
    seed.a.token,
    seed.slug,
    { email: emailFor(info, "profile-guest"), token: tokens.access_token },
    5
  );
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await [
    { width: 1280, height: 800 },
    { width: 390, height: 844 },
  ].reduce(async (before, viewport) => {
    await before;
    await page.setViewportSize(viewport);
    await page.goto(`/${seed.slug}/profile/${seed.b.id}/assigned`);
    await expect(page.getByText("You do not have the permission to access this page.", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Display", exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Add filter", exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: /^(List|Board) Layout$/ })).toHaveCount(0);
    await expect(page.getByRole("link", { name: /^(Assigned|Created|Subscribed)$/ })).toHaveCount(0);
  }, Promise.resolve());
  expect(watch.apiRequests.filter((request) => /GET .*\/members\/[^/]+\/issues$/.test(request))).toEqual([]);
  await expectListPageQuiet(page, watch, browserName, 2);
});
