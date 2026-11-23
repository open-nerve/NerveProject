import {
  addProjectMembers,
  changeProject,
  createLabel,
  createProject,
  projectMemberWrites,
  type Api,
} from "../../fixtures/api";
import { notificationStates } from "../../fixtures/assert/notification";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { commentDeleted, commentUpdated, createIssueComment } from "../../fixtures/comments";
import { failedLoads, recordApi, toast, turnCycles } from "../../fixtures/cycle-pages";
import { createCycle } from "../../fixtures/cycles";
import { submitIntakeIssue } from "../../fixtures/intake";
import { expectHeaderOf, shownStatus } from "../../fixtures/intake-pages";
import {
  createIssue,
  createIssueRelations,
  issueArchived,
  issueDeleted,
  issueRelationDeleted,
  issueUnarchived,
  listIssueRelations,
  updateIssue,
  type Issue,
} from "../../fixtures/issues";
import { createIssueLink, issueLinkDeleted, updateIssueLink } from "../../fixtures/links";
import { turnModules } from "../../fixtures/module-pages";
import { createModule } from "../../fixtures/modules";
import { APP, WORKSPACE, valued } from "../../fixtures/mounts";
import {
  cardSaying,
  expectNothingSelected,
  notificationWorld,
  notificationsPath,
  shownCards,
} from "../../fixtures/notification-pages";
import { notificationsSent } from "../../fixtures/notifications";
import { holdAnswer } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { fileDeleted, storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

// H2–H4's page version (M7 design 2, 3.2–3.4; P4 spec 2.6, 2.11, R11, R12, R22, re-93 I4): the notifications of real
// writes as their receiver reads them on the notifications page. Each card says who made the change, a member by his
// name and his picture's letter, and what, by the kind of change and its values (describeChange; a removed value
// arrives as ""), with the work item's reference and name as they were when it was sent, and how long ago. A
// description's change, a cycle's and a module's have no card; a mention is in the mentions tab alone, instead of its
// comment's card; the other end of a relation is named only to whom it shows (H4); a deleted work item's cards are no
// longer listed; an excerpt is plain text. An intake item's notification embeds its intake detail (P5b), closed by Close.
// Who has left a project, before the page's refresh, opens one of its cards: nerve answers that notification as not
// found, the toast says so, the card leaves and the preview shows nothing selected, never a blank pane; the refresh
// lists none of the project's. The API version is h2-h4-notifications.spec.ts; the parent's change whose two ends he
// sees neither of, and a field the card has no words for, are describeChange's unit cells. Nerve's own change (R24) is
// h2-h4-auto-archive-page.spec.ts.

/** How long ago each card of this story was sent, as the page says it: the story reads them within half a minute. */
const NOW = "less than a minute ago";

/** The reference and name of item, as a card shows the work item, by name, its name then. */
function ref(item: Pick<Issue, "sequence_id">, name: string, identifier = "WEB"): string {
  return `${identifier}-${item.sequence_id} ${name}`;
}

/** What shownCards gives of a card of Alice's change, saying what, of the work item of reference at, unread. */
function alices(what: string, at: string, excerpt?: string, unread = true): string {
  return [...(unread ? ["Unread"] : []), "A", `Alice ${what}`, ...(excerpt ? [excerpt] : []), at, NOW].join(" | ");
}

/**
 * Alice's writes on the work items bob holds, Fix his assignment, each of which sends him one notification at least,
 * and those that send him none: Fix's description, cycle and module. A notification's work item and excerpt are as the
 * fan-out reads them, after its write has answered: so Fix's creation is sent before it is renamed, and each comment's
 * before it changes (sent waits until bob is listed that many).
 */
async function alicesWrites(
  api: Api,
  w: Awaited<ReturnType<typeof notificationWorld>>,
  sent: (n: number) => Promise<void>
) {
  const { alice, bob, walt, slug, web } = w;
  const token = alice.token;
  // Ops, alice's secret project, which bob is no member of: its Secret is a work item he does not see.
  const ops = await createProject(api, token, slug, { name: "Ops", identifier: "OPS", network: 0 });
  const secret = await createIssue(api, token, ops.id, { name: "Secret" });
  await Promise.all([turnCycles(api, token, web, true), turnModules(api, token, web, true)]);
  const [cycle, alpha, states, bug] = await Promise.all([
    createCycle(api, token, web.id, { name: "Sprint" }),
    createModule(api, token, web.id, { name: "Alpha" }),
    stateIds(api, token, web.id),
    createLabel(api, token, web.id, { name: "Bug" }),
  ]);
  // Spare, bob's, deleted once he has its two notifications, beside Rate limit's two.
  const spare = await createIssue(api, token, web.id, { name: "Spare", assignee_ids: [bob.id] });
  await sent(4);
  expect(await issueDeleted(api, token, spare.id)).toEqual({ status: 204 });
  // Fix, bob's, renamed once he has its two notifications: theirs keep its name then.
  const fix = await createIssue(api, token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await sent(4);
  const [parent, docs, tests, deploy] = await Promise.all(
    ["Parent", "Docs", "Tests", "Deploy"].map((name) => createIssue(api, token, web.id, { name }))
  );
  if (!parent || !docs || !tests || !deploy) throw new Error("Fix's other ends were not made");
  const change = (body: Parameters<typeof updateIssue>[3]) => updateIssue(api, token, fix.id, body);
  const relate = (relation_type: "relates_to" | "duplicate" | "blocking" | "blocked_by", other: Issue) =>
    createIssueRelations(api, token, fix.id, { relation_type, issue_ids: [other.id] });
  const unrelate = async (other: Issue) => {
    const relation = (await listIssueRelations(api, token, fix.id)).find((r) => r.issue.id === other.id);
    expect(await issueRelationDeleted(api, token, relation?.id ?? "")).toEqual({ status: 204 });
  };
  // one change after another, each its own write
  await (
    [
      { description_html: "<p>Steps</p>" },
      { cycle_id: cycle.id },
      { module_ids: [alpha.id] },
      { name: "Fix the login" },
      { priority: "high" },
      { state_id: states.Todo },
      { start_date: "2026-10-17" },
      { target_date: "2026-10-20" },
      { start_date: null },
      { target_date: null },
      { label_ids: [bug.id] },
      { label_ids: [] },
      { assignee_ids: [bob.id, walt.id] },
      { assignee_ids: [bob.id] },
      { parent_id: parent.id },
      { parent_id: null },
    ] satisfies Parameters<typeof updateIssue>[3][]
  ).reduce(async (previous, body) => {
    await previous;
    await change(body);
  }, Promise.resolve());
  await relate("relates_to", parent);
  await relate("relates_to", secret);
  await relate("duplicate", docs);
  await relate("blocking", tests);
  await relate("blocked_by", deploy);
  await unrelate(secret);
  await unrelate(parent);
  await change({ state_id: states.Done });
  expect((await issueArchived(api, token, fix.id)).status).toBe(200);
  expect((await issueUnarchived(api, token, fix.id)).status).toBe(200);
  await sent(27);
  const looks = await createIssueComment(api, token, fix.id, "<p>Looks <strong>good</strong></p>");
  await sent(28);
  expect((await commentUpdated(api, token, looks.id, "<p>Looks <em>good</em> now</p>")).status).toBe(200);
  await sent(29);
  expect(await commentDeleted(api, token, looks.id)).toEqual({ status: 204 });
  const link = await createIssueLink(api, token, fix.id, { url: "https://example.test/guide" });
  await updateIssueLink(api, token, link.id, { url: "https://example.test/guide-v2" });
  expect(await issueLinkDeleted(api, token, link.id)).toEqual({ status: 204 });
  const file = await storedFile(api, token, "ISSUE_ATTACHMENT", { kind: "issue", id: fix.id });
  expect(await fileDeleted(api, token, file.id)).toEqual({ status: 204 });
  // Two mentions of bob, in Fix's description and in a comment on it: his mentions tab's alone.
  const mention = `<mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`;
  await change({ description_html: `<p>Steps</p><p>See ${mention}</p>` });
  await createIssueComment(api, token, fix.id, `<p>Hello</p>${mention}`);
  return { fix, parent, docs, tests, deploy };
}

test("H2–H4 (page): bob's notifications page lists a card for each of Alice's changes of Fix, by her name and her picture's letter, in words of the change and its values, with Fix's reference and name as they were, and none of its description, cycle and module; the mentions in the mentions tab alone; another end named only where he sees it; none of a deleted work item; an intake item's card embeds its intake detail, without triage; once he has left Web, the card he opens is not found, said once, gone, the preview back to nothing selected, and the refresh lists none of Web's", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const w = await notificationWorld(api, testInfo);
  const { alice, bob, slug, web } = w;
  // API, a project of alice's of which bob is a member too, and its Rate limit, his: the cards of another project.
  const apiProject = await createProject(api, alice.token, slug, { name: "API", identifier: "API" });
  await addProjectMembers(api, alice.token, apiProject.id, [{ member_id: bob.id, role: 15 }]);
  const rate = await createIssue(api, alice.token, apiProject.id, { name: "Rate limit", assignee_ids: [bob.id] });
  const sent = async (count: number) => {
    await notificationsSent(api, bob.token, slug, count);
  };
  const { fix, parent, docs, tests, deploy } = await alicesWrites(api, w, sent);
  // Crash on save, submitted to Web's intake by alice for bob.
  expect((await changeProject(api, alice.token, web.id, { intake_view: true })).status).toBe(200);
  const crash = await submitIntakeIssue(api, alice.token, web.id, { name: "Crash on save", assignee_ids: [bob.id] });
  const [all] = await Promise.all([
    notificationsSent(api, bob.token, slug, 37),
    notificationsSent(api, bob.token, slug, 2, { tab: "mentions" }),
  ]);

  // The page opened anew asks for the app's, the workspace's and the all tab's first page alone; the tabs say their
  // unread.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  const requests = recordApi(page);
  await page.goto(notificationsPath(slug));
  const [fixed, login] = [ref(fix, "Fix"), ref(fix, "Fix the login")];
  const fixCards = [
    alices("removed an attachment", login),
    alices("added an attachment", login),
    alices("removed the link https://example.test/guide-v2", login),
    alices("changed a link to https://example.test/guide-v2", login),
    alices("added the link https://example.test/guide", login),
    alices("deleted a comment", login),
    alices("edited a comment", login, "Looks good now"),
    alices("commented", login, "Looks good"),
    alices("restored the work item", login),
    alices("archived the work item", login),
    alices("set the state to Done", login),
    alices(`marked the work item as no longer related to ${ref(parent, "Parent")}`, login),
    alices("marked the work item as no longer related to another work item", login),
    alices(`marked the work item as blocked by ${ref(deploy, "Deploy")}`, login),
    alices(`marked the work item as blocking ${ref(tests, "Tests")}`, login),
    alices(`marked the work item as a duplicate of ${ref(docs, "Docs")}`, login),
    alices("marked the work item as related to another work item", login),
    alices(`marked the work item as related to ${ref(parent, "Parent")}`, login),
    alices(`made the work item no longer a child of ${ref(parent, "Parent")}`, login),
    alices(`set the parent to ${ref(parent, "Parent")}`, login),
    alices("unassigned Walt", login),
    alices("assigned Walt", login),
    alices("removed the label Bug", login),
    alices("added the label Bug", login),
    alices("removed the due date", login),
    alices("removed the start date", login),
    alices("set the due date to Oct 20, 2026", login),
    alices("set the start date to Oct 17, 2026", login),
    alices("set the state to Todo", login),
    alices("set the priority to High", login),
    alices("renamed the work item to Fix the login", login),
    alices("assigned Bob", fixed),
    alices("created the work item", fixed),
  ];
  const crashCards = [
    alices("assigned Bob", ref(crash, "Crash on save")),
    alices("created the work item", ref(crash, "Crash on save")),
  ];
  const rateCards = [
    alices("assigned Bob", ref(rate, "Rate limit", "API")),
    alices("created the work item", ref(rate, "Rate limit", "API")),
  ];
  await expect.poll(() => shownCards(page)).toEqual([...crashCards, ...fixCards, ...rateCards]);
  expect(requests()).toEqual({
    pending: 0,
    requests: [
      ...APP,
      ...WORKSPACE.map((request) => valued(request, { [slug]: "{slug}" })),
      `GET /api/v0/workspaces/${slug}/notifications?tab=all&archived=false&snoozed=false&limit=100`,
    ].toSorted(),
  });
  const tabs = page.getByRole("tab");
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 37 unread");
  await expect(tabs.nth(1)).toHaveAccessibleName("Mentions, 2 unread");
  // The excerpt is the comment's plain text: none of its markup.
  await expect(cardSaying(page, "Alice commented").locator("strong, em")).toHaveCount(0);

  // The mentions tab: alice's two mentions of bob, his alone, the comment's with its plain text.
  await page.getByRole("tab", { name: "Mentions, 2 unread", exact: true }).click();
  await expect
    .poll(() => shownCards(page))
    .toEqual([alices("mentioned you in a comment", login, "Hello"), alices("mentioned you in the description", login)]);

  // Crash on save's card, in the intake, opened: bob's notification read, its intake detail embedded in the preview,
  // pending, with its copy and its opening and the embed's Close (the toast's close is named so too: the preview's),
  // no plain peek; bob, a member who did not submit it, has no triage. The embed reads the work item after the
  // preview's read: its fetch, whose GET of the work item comes twice (the third and the fourth here: the fetch reads
  // it again as its sub-work items' parent), and its description's versions; bob leaves Web below once they and the
  // rest have answered (re-95 m12).
  await page.getByRole("tab", { name: "All, 37 unread", exact: true }).click();
  const crashNote = all.find((n) => n.issue_id === crash.id && n.data.change.field === "assignees");
  const crashPath = `/api/v0/issues/${crash.id}`;
  let crashReads = 0;
  const embedReads = [
    page.waitForResponse(
      (r) => r.request().method() === "GET" && new URL(r.url()).pathname === crashPath && ++crashReads === 4
    ),
    page.waitForResponse((r) => crashReads >= 3 && new URL(r.url()).pathname === `${crashPath}/description-versions`),
  ];
  await cardSaying(page, "Alice assigned Bob").filter({ hasText: "Crash on save" }).getByRole("button").first().click();
  const preview = page.getByRole("region", { name: "Preview", exact: true });
  await expect(preview.getByRole("button", { name: "Close", exact: true })).toBeVisible();
  await expectHeaderOf(page, crash);
  await expect(shownStatus(page, "Pending")).toBeVisible();
  await expect(preview.getByRole("button", { name: "Copy work item link", exact: true })).toBeVisible();
  await expect(preview.getByRole("link", { name: "Open work item", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^(Accept|Decline)$/ })).toHaveCount(0);
  await expect
    .poll(() => notificationStates(db, [crashNote?.id ?? ""]))
    .toEqual([{ read: true, archived: false, snoozed: false, by_receiver: true }]);
  await expect
    .poll(() => shownCards(page))
    .toContain(alices("assigned Bob", ref(crash, "Crash on save"), undefined, false));

  await Promise.all(embedReads);
  await expect.poll(() => requests().pending).toBe(0);
  // bob leaves Web (in another sign-in of his) and opens its card of the priority, before the page's refresh: the
  // preview reads its work item, not found, and says so, with a retry, while nerve's answer to marking the
  // notification read is held; then that answer, not found too, is said in a toast, once, the card leaves and the
  // preview shows nothing selected.
  expect(await projectMemberWrites(api, web.id).leave(bob.token)).toEqual({ status: 204 });
  const priorityNote = all.find((n) => n.issue_id === fix.id && n.data.change.field === "priority");
  const release = await holdAnswer(page, "PATCH", `/api/v0/notifications/${priorityNote?.id}`);
  const refused = page.waitForResponse((r) => new URL(r.url()).pathname === `/api/v0/issues/${fix.id}`);
  await cardSaying(page, "Alice set the priority to High").getByRole("button").first().click();
  expect((await refused).status()).toBe(404);
  const failure = page.getByRole("alert").filter({ hasText: "The work item does not exist, or you cannot see it." });
  await expect(failure.getByRole("button", { name: "Retry", exact: true })).toBeVisible();
  await release();
  await expect(toast(page, "The notification does not exist, or you cannot see it.")).toHaveCount(1);
  await expect(cardSaying(page, "Alice set the priority to High")).toHaveCount(0);
  await expectNothingSelected(page);
  // The refresh: Rate limit's cards alone.
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect.poll(() => shownCards(page)).toEqual(rateCards);

  expect([watch.apiFailures.toSorted(), watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/issues/${fix.id}`, `404 PATCH /api/v0/notifications/${priorityNote?.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404, 404) });
});
