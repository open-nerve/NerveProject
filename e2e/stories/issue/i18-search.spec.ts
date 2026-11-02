import { c12Evidence } from "../../fixtures/c12-evidence";
import { readFileSync } from "node:fs";
import type { TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { expectIssueCollection } from "../../fixtures/assert/issue";
import { newAccount, accountId, emailFor } from "../../fixtures/auth";
import { issueListProject, expectListPageQuiet, openIssueList } from "../../fixtures/issue-list-page";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { openCommands } from "../../fixtures/power-k";
import { deferred } from "../../fixtures/deferred";
import { answerTo, holdAnswer, moveWithinApp, registerOnboarded } from "../../fixtures/settings-pages";
import { instantKey } from "../../fixtures/instant";
import {
  createIssue,
  createIssueRelations,
  issueRelationDeleted,
  searchIssues,
  updateIssue,
} from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// Every precondition is an API write; P15 pages exercise the same native search contract.
async function searchWorld(api: Api, info: TestInfo) {
  const admin = await newAccount(api, info, "admin");
  const member = await newAccount(api, info, "member");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Search stories", slug });
  await inviteAndAccept(api, admin.token, slug, member, 15);
  return { admin, member, slug };
}

test("I18 (API): visible same-sequence issues rank before title hits; hidden higher scores cannot consume the limit, and shared Unicode identifier cases select the exact visible project", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const fixture = JSON.parse(
    readFileSync(new URL("../../../api/testdata/identifier-fold.json", import.meta.url), "utf8")
  ) as { cases: { name: string; identifiers: string[]; asked: string; want: string | null }[] };
  const folded = fixture.cases.find((row) => row.name === "I and İ stored, ı asked: I, its upper case");
  const dotted = fixture.cases.find((row) => row.name === "I and İ stored, İ asked");
  if (!folded || !dotted) throw new Error("the shared Unicode identifier examples are missing");
  const projects: Awaited<ReturnType<typeof createProject>>[] = [];
  await folded.identifiers.reduce(async (before, identifier) => {
    await before;
    const project = await createProject(api, admin.token, slug, { name: identifier, identifier, network: 0 });
    await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
    projects.push(project);
  }, Promise.resolve());
  const [plain, dottedProject] = projects;
  if (!plain || !dottedProject) throw new Error("missing visible projects");
  const a = await createIssue(api, member.token, plain.id, { name: "Orbit release" });
  const b = await createIssue(api, member.token, dottedProject.id, { name: "Orbit release" });
  expect(a.sequence_id).toBe(b.sequence_id);
  expect(instantKey(a.created_at) < instantKey(b.created_at)).toBe(true);
  const titleHit = await createIssue(api, member.token, plain.id, { name: "1" });
  const hidden = await createProject(api, admin.token, slug, { name: "Hidden", identifier: "SECRET", network: 0 });
  await createIssue(api, admin.token, hidden.id, { name: "1" });
  await createIssue(api, admin.token, hidden.id, { name: "Orbit" });
  // Equal titles have equal similarity; their known creation order settles the numeric branch.
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "1" }), [b.id, a.id, titleHit.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "1", limit: 1 }), [b.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "Orbit", limit: 1 }), [b.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "1", project_id: plain.id }), [
    a.id,
    titleHit.id,
  ]);
  expect(plain.identifier).toBe(folded.want);
  expect(dottedProject.identifier).toBe(dotted.want);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: `${folded.asked}-${a.sequence_id}` }), [a.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: `${dotted.asked}-${b.sequence_id}` }), [b.id]);
});

test("I18 (API): own-only hides another author's matching issue and parent; parent and sub-issue selectors traverse a five-level tree through that hidden intermediate node", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const project = await createProject(api, admin.token, slug, { name: "Tree", identifier: "TREE" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
  expect((await changeProject(api, admin.token, project.id, { guest_view_all_features: false })).status).toBe(200);
  const root = await createIssue(api, member.token, project.id, { name: "Tree" });
  const source = await createIssue(api, member.token, project.id, { name: "Tree", parent_id: root.id });
  const hidden = await createIssue(api, admin.token, project.id, { name: "Tree", parent_id: source.id });
  const deep = await createIssue(api, member.token, project.id, { name: "Tree", parent_id: hidden.id });
  const leaf = await createIssue(api, member.token, project.id, { name: "Tree", parent_id: deep.id });
  const allowed = await createIssue(api, member.token, project.id, { name: "Tree" });
  await createIssue(api, admin.token, project.id, { name: "Tree" });
  expect(
    await projectMemberWrites(api, project.id).change(
      admin.token,
      await projectMembershipOf(api, admin.token, project.id, member.id),
      5
    )
  ).toEqual({ status: 200 });
  await changeRole(api, admin.token, slug, member.id, 5);
  const all = await searchIssues(api, member.token, slug, { q: "Tree" });
  expectIssueCollection(all, [allowed.id, leaf.id, deep.id, source.id, root.id]);
  expect(all.data.find((row) => row.id === deep.id)?.parent_id).toBeNull();
  expect(all.data.find((row) => row.id === source.id)?.parent_id).toBe(root.id);
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: "Tree", for: "parent", issue_id: source.id }),
    [allowed.id]
  );
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: "Tree", for: "sub_issue", issue_id: leaf.id }),
    [allowed.id]
  );
  // These otherwise matching numeric/identifier branches cannot bypass the exclusions either.
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: `TREE-${leaf.sequence_id}`, for: "parent", issue_id: source.id }),
    []
  );
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: `${root.sequence_id}`, for: "sub_issue", issue_id: leaf.id }),
    []
  );
});

test("I18 (API): relation selection excludes every live stored type in both directions before limit and returns a target again after its relation is deleted", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const project = await createProject(api, admin.token, slug, { name: "Relations", identifier: "REL" });
  const other = await createProject(api, admin.token, slug, { name: "Other", identifier: "OTHER" });
  await [project, other].reduce(async (before, p) => {
    await before;
    await addProjectMembers(api, admin.token, p.id, [{ member_id: member.id, role: 15 }]);
  }, Promise.resolve());
  const source = await createIssue(api, member.token, project.id, { name: "Related" });
  const allowed = await createIssue(api, member.token, other.id, { name: "Related" });
  const relations: {
    relation: Awaited<ReturnType<typeof createIssueRelations>>[number];
    target: Awaited<ReturnType<typeof createIssue>>;
  }[] = [];
  const directions = (["relates_to", "duplicate", "blocked_by"] as const).flatMap((relation_type) =>
    [false, true].map((reverse) => ({ relation_type, reverse }))
  );
  await directions.reduce(async (before, { relation_type, reverse }) => {
    await before;
    const target = await createIssue(api, member.token, other.id, { name: "Related" });
    // blocked_by is stored as requested; unlike blocking it does not swap endpoints.
    const [relation] = await createIssueRelations(api, member.token, reverse ? target.id : source.id, {
      relation_type,
      issue_ids: [reverse ? source.id : target.id],
    });
    if (!relation) throw new Error("the new relation is missing");
    relations.push({ relation, target });
  }, Promise.resolve());
  const query = { q: "Related", for: "relation", issue_id: source.id };
  expectIssueCollection(await searchIssues(api, member.token, slug, query), [allowed.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { ...query, limit: 1 }), [allowed.id]);
  const restored = relations.at(-1)!;
  expect(await issueRelationDeleted(api, member.token, restored.relation.id)).toEqual({ status: 204 });
  expectIssueCollection(await searchIssues(api, member.token, slug, query), [restored.target.id, allowed.id]);
});

test("I18 (API): two- and three-character Chinese contains queries and literal symbols work; empty words use updated order and date none independently", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const project = await createProject(api, admin.token, slug, { name: "Words", identifier: "WORDS" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
  const short = await createIssue(api, member.token, project.id, { name: "中文" });
  const long = await createIssue(api, member.token, project.id, { name: "中文词" });
  const dated = await createIssue(api, member.token, project.id, {
    name: "Dated",
    start_date: "2026-10-01",
    target_date: "2026-10-02",
  });
  const literal = await createIssue(api, member.token, project.id, { name: "literal %_\\" });
  const descriptionOnly = await createIssue(api, member.token, project.id, {
    name: "Other",
    description_html: "<p>中文词 literal %_\\</p>",
    target_date: "2026-10-03",
  });
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "中文" }), [short.id, long.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "中文词" }), [long.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "%_\\" }), [literal.id]);
  const changed = await updateIssue(api, member.token, short.id, { priority: "high" });
  expect(instantKey(changed.updated_at) > instantKey(descriptionOnly.updated_at)).toBe(true);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: " \t " }), [
    short.id,
    descriptionOnly.id,
    literal.id,
    dated.id,
    long.id,
  ]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "", target_date: ["none"] }), [
    short.id,
    literal.id,
    long.id,
  ]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { start_date: ["none"] }), [
    short.id,
    descriptionOnly.id,
    literal.id,
    long.id,
  ]);
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { target_date: ["2026-10-01..2026-10-31", "..2026-10-02"], limit: 1 }),
    [dated.id]
  );
});

test("I18 (page): native command search keeps relevance order, exact Unicode and literal queries, workspace scope and local projects", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const other = await createProject(api, seed.member.token, seed.slug, { name: "Command Catalog", identifier: "CAT" });
  const a = await createIssue(api, seed.member.token, seed.project.id, { name: "Orbit release" });
  const b = await createIssue(api, seed.member.token, other.id, { name: "Orbit release" });
  const titleHit = await createIssue(api, seed.member.token, seed.project.id, { name: "1" });
  const chinese = await createIssue(api, seed.member.token, seed.project.id, { name: "中文" });
  const literal = await createIssue(api, seed.member.token, seed.project.id, { name: "Literal %_\\" });
  const watch = await openIssueDetail(page, seed, a);
  const path = `/api/v0/workspaces/${seed.slug}/issue-search`;
  const queries: URLSearchParams[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname === path) queries.push(url.searchParams);
  });
  await openCommands(page);
  expect(queries).toEqual([]);
  const input = page.getByRole("combobox");
  const results = page.locator('[role="option"][data-value^="issue-"]');
  await input.fill("1");
  await expect(results).toHaveCount(2);
  expect(await results.evaluateAll((rows) => rows.map((row) => row.getAttribute("data-value")))).toEqual([
    `issue-${a.id}`,
    `issue-${titleHit.id}`,
  ]);
  expect(queries.at(-1)?.get("project_id")).toBe(seed.project.id);
  await page.getByRole("switch", { name: "Workspace level", exact: true }).click();
  await expect(results).toHaveCount(3);
  expect(await results.evaluateAll((rows) => rows.map((row) => row.getAttribute("data-value")))).toEqual([
    `issue-${b.id}`,
    `issue-${a.id}`,
    `issue-${titleHit.id}`,
  ]);
  expect(queries.at(-1)?.has("project_id")).toBe(false);
  const directoryBefore = watch.apiRequests.filter(
    (request) => request === `GET /api/v0/workspaces/${seed.slug}/projects`
  ).length;
  const numeric = "0001";
  const ranked = (await searchIssues(api, seed.member.token, seed.slug, { q: numeric })).data;
  expect(ranked.map((issue) => issue.id)).toEqual([b.id, a.id]);
  expect(ranked.every((issue) => !issue.name.includes(numeric))).toBe(true);
  await [
    { q: "中文", ids: [chinese.id] },
    { q: "%_\\", ids: [literal.id] },
    { q: "WEB-1", ids: [a.id] },
    { q: numeric, ids: ranked.map((issue) => issue.id) },
  ].reduce(async (previous, { q, ids }) => {
    await previous;
    await input.fill(q);
    await expect(results).toHaveCount(ids.length);
    await expect
      .poll(() => results.evaluateAll((rows) => rows.map((row) => row.getAttribute("data-value"))))
      .toEqual(ids.map((id) => `issue-${id}`));
    expect(queries.at(-1)?.get("q")).toBe(q);
    await c12Evidence(page, "B19", `modal-query-${q}`);
    expect(queries.at(-1)?.has("project_id")).toBe(false);
  }, Promise.resolve());
  await input.fill("cAtAloG");
  const projectResult = page.getByRole("option", { name: "Command Catalog", exact: true });
  await expect(projectResult).toBeVisible();
  await expect(results).toHaveCount(0);
  expect(
    watch.apiRequests.filter((request) => request === `GET /api/v0/workspaces/${seed.slug}/projects`)
  ).toHaveLength(directoryBefore);
  await input.press("Enter");
  await expect(page).toHaveURL(`/${seed.slug}/projects/${other.id}/issues`);
  await expectListPageQuiet(page, watch);
});

test("I18 (page): search errors offer GET retry and a delayed previous query cannot replace the current results", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const first = await createIssue(api, seed.member.token, seed.project.id, { name: "First search" });
  const next = await createIssue(api, seed.member.token, seed.project.id, { name: "Second search" });
  const watch = await openIssueDetail(page, seed, first);
  const path = `/api/v0/workspaces/${seed.slug}/issue-search`;
  let refused = false;
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      if (refused) return route.fallback();
      refused = true;
      await route.fulfill({
        status: 500,
        contentType: "application/problem+json",
        body: JSON.stringify({ status: 500, code: "internal_error" }),
      });
    }
  );
  await openCommands(page);
  const input = page.getByRole("combobox");
  await input.fill("First search");
  const alert = page.getByRole("alert");
  await expect(alert).toContainText("Something went wrong on the server. Please try again.");
  await c12Evidence(page, "B19", "search-get-refusal");
  await expect(page.getByRole("option", { name: /^No results found/ })).toHaveCount(0);
  expect(
    (
      await answerTo(page, "GET", path, () => alert.getByRole("button", { name: "Retry", exact: true }).click())
    ).status()
  ).toBe(200);
  const firstResult = page.locator(`[role="option"][data-value="issue-${first.id}"]`);
  await expect(firstResult).toBeVisible();
  const release = await holdAnswer(page, "GET", path);
  const sent = page.waitForRequest(
    (request) => new URL(request.url()).pathname === path && new URL(request.url()).searchParams.get("q") === "WEB-1"
  );
  await input.fill("WEB-1");
  await expect(firstResult).toHaveCount(0);
  await sent;
  await input.fill("WEB-2");
  const current = page.locator(`[role="option"][data-value="issue-${next.id}"]`);
  await expect(current).toBeVisible();
  expect((await answerTo(page, "GET", path, release)).status()).toBe(200);
  await expect(current).toBeVisible();
  await expect(firstResult).toHaveCount(0);
  await input.press("Enter");
  await expect(page).toHaveURL(`/${seed.slug}/browse/WEB-${next.sequence_id}`);
  expect(watch.apiRequests.filter((request) => /^(PATCH|POST|DELETE) .*issues/.test(request))).toEqual([]);
  await expectListPageQuiet(page, watch, {
    failures: [`500 GET ${path}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 500 (Internal Server Error)"]
        : [],
  });
});

for (const role of [5, 20] as const) {
  test(`I18 (page): OwnOnly search uses the returned visibility for a workspace role ${role} and project guest`, async ({
    api,
    page,
  }, info) => {
    const seed = await issueListProject(api, info);
    const email = emailFor(info, "own-only");
    const tokens = await registerOnboarded(api, email);
    const member = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
    await inviteAndAccept(api, seed.member.token, seed.slug, member, 15);
    await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: member.id, role: 15 }]);
    const own = await createIssue(api, member.token, seed.project.id, { name: "Visible needle" });
    const hidden = await createIssue(api, seed.member.token, seed.project.id, { name: "Hidden needle" });
    expect(
      await projectMemberWrites(api, seed.project.id).change(
        seed.member.token,
        await projectMembershipOf(api, seed.member.token, seed.project.id, member.id),
        5
      )
    ).toEqual({ status: 200 });
    await changeRole(api, seed.member.token, seed.slug, member.id, role);
    const watch = await openIssueDetail(page, { ...seed, member }, own, role === 5 ? own.name : undefined);
    await openCommands(page);
    await expect(page.getByRole("option", { name: /^Copy title/ })).toBeVisible();
    if (role === 5) await expect(page.getByRole("option", { name: /^Change state/ })).toHaveCount(0);
    const input = page.getByRole("combobox");
    await input.fill("needle");
    await expect(page.locator(`[role="option"][data-value="issue-${own.id}"]`)).toBeVisible();
    await expect(page.locator(`[role="option"][data-value="issue-${hidden.id}"]`)).toHaveCount(0);
    await input.fill(`WEB-${hidden.sequence_id}`);
    await expect(page.getByRole("option", { name: /^No results found/ })).toBeVisible();
    await c12Evidence(page, "B19", `own-only-no-result-role-${role}`);
    await expectListPageQuiet(page, watch);
  });
}

// A held real search answer cannot revive a result after project A -> B -> Back to A.
test("I18 (page): Back and reopening the same query retire the previous project search answer", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const old = await createIssue(api, seed.member.token, seed.project.id, { name: "Old needle" });
  const beta = await createProject(api, seed.member.token, seed.slug, { name: "Beta", identifier: "BETA" });
  const betaIssue = await createIssue(api, seed.member.token, beta.id, { name: "Other project" });
  const watch = await openIssueDetail(page, seed, old);
  const path = `/api/v0/workspaces/${seed.slug}/issue-search`;
  const fetched = deferred<void>();
  // Install interception before the modal starts its font request; switching Chromium's
  // interception on mid-request can leave document.fonts.ready pending during capture.
  const release = await holdAnswer(page, "GET", path, 1, () => fetched.resolve());
  await openCommands(page);
  await page.getByRole("combobox").fill("needle");
  await fetched.promise;
  let released = false;
  try {
    await updateIssue(api, seed.member.token, old.id, { name: "Removed title" });
    const current = await createIssue(api, seed.member.token, seed.project.id, { name: "Current needle" });
    await moveWithinApp(page, `/${seed.slug}/projects/${beta.id}/issues`);
    await expect(page.locator(`#issue-${betaIssue.id}`)).toBeVisible();
    await expect(page.getByRole("combobox")).toHaveCount(0);
    await page.goBack();
    await expect(page).toHaveURL(`/${seed.slug}/browse/WEB-${old.sequence_id}`);
    await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toBeVisible();
    await openCommands(page);
    await page.getByRole("combobox").fill("needle");
    const result = page.locator(`[role="option"][data-value="issue-${current.id}"]`);
    await expect(result).toBeVisible();
    const previous = await answerTo(page, "GET", path, release);
    released = true;
    expect(previous.status()).toBe(200);
    expect((await previous.json()).data.map((row: { id: string }) => row.id)).toEqual([old.id]);
    await expect(result).toBeVisible();
    await expect(page.locator(`[role="option"][data-value="issue-${old.id}"]`)).toHaveCount(0);
    await c12Evidence(page, "B19", "modal-back-late-answer-retired");
    await page.getByRole("combobox").press("ControlOrMeta+k");
    await expect(page.getByRole("combobox")).toHaveCount(0);
    await openCommands(page);
    await expect(page.getByRole("option", { name: /^Change state/ })).toBeVisible();
    expect(watch.apiRequests.filter((request) => /^(POST|PATCH|DELETE) .*issues/.test(request))).toEqual([]);
    await expectListPageQuiet(page, watch);
  } finally {
    if (!released) await release();
  }
});

test("I18 (page): top navigation owns native search, IME and late answers across Back while the modal stays closed", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const other = await createProject(api, seed.member.token, seed.slug, { name: "Other search", identifier: "OTHER" });
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Native title" });
  const old = await createIssue(api, seed.member.token, seed.project.id, { name: "Late old" });
  const { watch } = await openIssueList(page, seed);
  const input = page.getByPlaceholder("Search commands...", { exact: true });
  const results = page.locator('[role="option"][data-value^="issue-"]');
  await input.fill("0001");
  await expect(results).toHaveCount(1);
  await expect(results).toHaveAttribute("data-value", `issue-${issue.id}`);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await input.dispatchEvent("keydown", { key: "Enter", isComposing: true, bubbles: true });
  await input.dispatchEvent("keydown", { key: "Enter", keyCode: 229, bubbles: true });
  await expect(page).toHaveURL(seed.path);
  await input.press("Enter");
  await expect(page).toHaveURL(`/${seed.slug}/browse/${seed.project.identifier}-${issue.sequence_id}`);
  await expect(input).toHaveValue("");
  await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
  const path = `/api/v0/workspaces/${seed.slug}/issue-search`;
  const received = deferred<void>();
  const release = await holdAnswer(page, "GET", path, 1, () => received.resolve());
  let released = false;
  try {
    await input.fill("Late");
    await received.promise;
    await updateIssue(api, seed.member.token, old.id, { name: "Retired" });
    const fresh = await createIssue(api, seed.member.token, seed.project.id, { name: "Late fresh" });
    await moveWithinApp(page, `/${seed.slug}/projects/${other.id}/issues`);
    await expect(page.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
    await expect(input).toHaveValue("");
    await expect(page.locator("[cmdk-list]")).toHaveCount(0);
    await page.goBack();
    await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
    await input.fill("Late");
    await expect(results).toHaveCount(1);
    await expect(results).toHaveAttribute("data-value", `issue-${fresh.id}`);
    await c12Evidence(page, "B19", "top-back-late-answer-retired");
    const response = await answerTo(page, "GET", path, release);
    released = true;
    expect(response.status()).toBe(200);
    expect((await response.json()).data.map((row: { id: string }) => row.id)).toEqual([old.id]);
    await expect(results).toHaveAttribute("data-value", `issue-${fresh.id}`);
    await c12Evidence(page, "B19", "top-back-late-answer-retired");
    await input.press("ControlOrMeta+k");
    await expect(page.locator("[cmdk-list]")).toHaveCount(0);
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await input.click();
    await expect(input).toHaveValue("");
    await expect(results).toHaveCount(0);
    await input.fill("Native title");
    await expect(results).toHaveAttribute("data-value", `issue-${issue.id}`);
    await page.getByRole("button", { name: "Clear search", exact: true }).click();
    await expect(input).toHaveValue("");
    await expect(results).toHaveCount(0);
    await input.press("Escape");
    expect(watch.apiRequests.filter((request) => /^(POST|PATCH|DELETE) .*issues/.test(request))).toEqual([]);
    await expectListPageQuiet(page, watch);
  } finally {
    if (!released) await release();
  }
});
