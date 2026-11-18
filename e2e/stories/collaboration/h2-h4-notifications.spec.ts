import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { named, newAccount } from "../../fixtures/auth";
import { createIssueComment } from "../../fixtures/comments";
import { createIssue, issueSubscribed, updateIssue, type Issue } from "../../fixtures/issues";
import {
  listNotifications,
  notificationsListed,
  notificationsSent,
  type Notification,
} from "../../fixtures/notifications";
import { expect, test } from "../../fixtures/test";

// H2–H4, the notifications of real writes as their receivers list them (M7 design 2, 3.2–3.4; P1c spec 2.3, 2.4): the
// API version. A write sends its notifications from a job after it answers, each to a subscriber of the work item but
// its writer, as its creator, else as an assignee, else as a subscriber, and to whom it mentions only as mentioned;
// each answered with who wrote it, the work item as it was then and the change. The all tab lists every sender but
// mentioned, the mentions tab the mentions alone, type a subset of the all tab's senders, and pages by created_at and
// id walk the list once. The other writes, the recheck, the other ends, every filter and refusal, and the paging's
// edges are the bootstrap tests'.

/** The key of a notification among one receiver's in this story: its work item and the field it changed. */
function subject(note: { issue_id: string; data: { change: { field: string } } }): string {
  return `${note.issue_id} ${note.data.change.field}`;
}

/** notes in the order of their subjects, to compare a receiver's notifications whatever their order. */
function bySubject<T extends Parameters<typeof subject>[0]>(notes: T[]): T[] {
  return notes.toSorted((a, b) => subject(a).localeCompare(subject(b)));
}

/** The ids of notes, in their order. */
function ids(notes: Notification[]): string[] {
  return notes.map((n) => n.id);
}

test("H2–H4 (API): an admin's creation of a work item with an assignee, her change of another that a member subscribes to and her comment mentioning him are listed to each receiver with their sender, writer, work item and change; the member's comment to her alone; a mention only in the mentions tab, and instead of his subscription's; type selects senders, and pages of two walk the list", async ({
  api,
}, testInfo) => {
  const [alice, bob, walt] = await Promise.all([
    newAccount(api, testInfo, "alice"),
    newAccount(api, testInfo, "bob"),
    newAccount(api, testInfo, "walt"),
  ]);
  await Promise.all([named(api, bob.token, { display_name: "Bob" }), named(api, walt.token, { display_name: "Walt" })]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, walt, 15);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: walt.id, role: 15 },
  ]);

  // alice creates Fix for bob and Move for walt: each its creator and its assignee subscribe to it.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  const move = await createIssue(api, alice.token, web.id, { name: "Move", assignee_ids: [walt.id] });
  // A fan-out reads the work item's subscribers when it runs: Move's creation is sent before bob subscribes to it.
  await notificationsSent(api, walt.token, slug, 2);
  expect(await issueSubscribed(api, bob.token, move.id), "bob subscribes to Move").toEqual({ status: 204 });
  await updateIssue(api, alice.token, move.id, { name: "Move it" });
  const mention = `<p>Hello</p><mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`;
  const hello = await createIssueComment(api, alice.token, move.id, mention);
  const onIt = await createIssueComment(api, bob.token, fix.id, "<p>On it</p>");
  // Each write's notifications are written at once: these are of all six.
  await Promise.all([
    notificationsSent(api, bob.token, slug, 3),
    notificationsSent(api, bob.token, slug, 1, { tab: "mentions" }),
    notificationsSent(api, walt.token, slug, 4),
    notificationsSent(api, alice.token, slug, 1),
  ]);

  // A notification, unread, not archived nor snoozed, of the work item item, in Web, by the account of by, its change
  // of change's keys, every other key of the change null.
  const sent = (
    sender: Notification["sender"],
    by: string,
    item: Issue,
    name: string,
    change: { field: string; verb: string; [key: string]: unknown }
  ) => ({
    id: expect.any(String),
    workspace_id: acme.id,
    project_id: web.id,
    issue_id: item.id,
    sender,
    triggered_by_id: by,
    data: {
      issue: {
        id: item.id,
        name,
        sequence_id: item.sequence_id,
        identifier: "WEB",
        state_name: "Backlog",
        state_group: "backlog",
      },
      change: {
        old_value: null,
        new_value: null,
        old_id: null,
        new_id: null,
        old_issue: null,
        new_issue: null,
        comment_id: null,
        excerpt: null,
        ...change,
      },
    },
    read_at: null,
    archived_at: null,
    snoozed_till: null,
    created_at: expect.any(String),
  });
  const created = { field: "issue", verb: "created" };
  const renamed = { field: "name", verb: "updated", old_value: "Move", new_value: "Move it" };
  const commented = { field: "comment", verb: "created", comment_id: hello.id, excerpt: "Hello" };
  // The caller of token's two tabs.
  const tabs = ({ token }: { token: string }) =>
    Promise.all([
      listNotifications(api, token, slug).then((page) => page.data),
      listNotifications(api, token, slug, { tab: "mentions" }).then((page) => page.data),
    ]);
  const [[bobs, bobsMentions], [walts, waltsMentions], [alices, alicesMentions]] = await Promise.all([
    tabs(bob),
    tabs(walt),
    tabs(alice),
  ]);
  // bob: Fix's creation and his assignment, as its assignee; Move's new name, as its subscriber; alice's comment on
  // Move only as its mention of him, in the mentions tab; nothing of his own comment, though he is Fix's assignee.
  expect(bySubject(bobs), "bob's all tab").toEqual(
    bySubject([
      sent("assigned", alice.id, fix, "Fix", created),
      sent("assigned", alice.id, fix, "Fix", { field: "assignees", verb: "created", new_value: "Bob", new_id: bob.id }),
      sent("subscribed", alice.id, move, "Move it", renamed),
    ])
  );
  expect(bobsMentions, "bob's mentions tab").toEqual([
    sent("mentioned", alice.id, move, "Move it", { ...commented, verb: "mentioned" }),
  ]);
  // walt, Move's assignee: each of alice's writes of Move, her comment as a comment.
  expect(bySubject(walts), "walt's all tab").toEqual(
    bySubject([
      sent("assigned", alice.id, move, "Move", created),
      sent("assigned", alice.id, move, "Move", {
        field: "assignees",
        verb: "created",
        new_value: "Walt",
        new_id: walt.id,
      }),
      sent("assigned", alice.id, move, "Move it", renamed),
      sent("assigned", alice.id, move, "Move it", commented),
    ])
  );
  expect(waltsMentions, "walt's mentions tab").toEqual([]);
  // alice, creator of both, nothing of her own writes: bob's comment on her Fix alone.
  expect(alices, "alice's all tab").toEqual([
    sent("created", bob.id, fix, "Fix", { field: "comment", verb: "created", comment_id: onIt.id, excerpt: "On it" }),
  ]);
  expect(alicesMentions, "alice's mentions tab").toEqual([]);

  // type selects the all tab's senders, in the list's order; the mentions tab takes none.
  await Promise.all(
    ["assigned", "subscribed", "created", "created,subscribed"].map(async (type) => {
      const senders = type.split(",");
      expect(ids((await listNotifications(api, bob.token, slug, { type })).data), `bob's type=${type}`).toEqual(
        ids(bobs.filter((n) => senders.includes(n.sender)))
      );
    })
  );
  expect(await notificationsListed(api, bob.token, slug, { tab: "mentions", type: "assigned" })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "type", code: "not_allowed" }],
  });

  // Newest first: Move's new name, then Fix's two of one moment, by id; pages of two walk them once, the edge between
  // the two of one moment.
  expect(bobs.map(subject)[0], "bob's newest").toBe(subject({ issue_id: move.id, data: { change: renamed } }));
  expect(bobs[2]?.created_at, "Fix's creation and bob's assignment, of one moment").toBe(bobs[1]?.created_at);
  const first = await listNotifications(api, bob.token, slug, { limit: 2 });
  expect(first.next_cursor, "the first page of two's cursor").not.toBeNull();
  const second = await listNotifications(api, bob.token, slug, { limit: 2, cursor: first.next_cursor ?? "" });
  expect(second.next_cursor, "the second page of two's cursor").toBeNull();
  expect(ids([...first.data, ...second.data]), "bob's pages of two").toEqual(ids(bobs));
});
