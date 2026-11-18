import type { TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { login, newAccount } from "../../fixtures/auth";
import { createIssueComment } from "../../fixtures/comments";
import { createIssue } from "../../fixtures/issues";
import {
  allNotificationsMarked,
  listNotifications,
  notificationsSent,
  notificationUpdated,
  unreadNotifications,
  type Notification,
} from "../../fixtures/notifications";
import { expect, test } from "../../fixtures/test";

// H5–H7, a receiver's notifications as he reads and changes them (M7 design 2, 3.4; P1c spec 2.3, 2.5, 2.7): the API
// version. Each of read, archived and snoozed_till changes one state of his own notification, alike with a personal
// access token and his browser session; the read, archived and snoozed filters and the unread counts of the two tabs
// follow, the counts leaving out the archived and the snoozed; a snooze ends by itself; marking all read marks the
// unread of a tab and its filters alone. What he reads now decides: removed from a project, he is listed, counted and
// lets change none of its notifications, which are kept as they were. Another's notification is refused as one that
// never was, before a value is. The other refusals, the moments, the batches, the other cases of what he reads and the
// interleavings are the bootstrap tests'.

/** The ids of notes, in their order. */
function ids(notes: Notification[]): string[] {
  return notes.map((n) => n.id);
}

/** A notification of field, the one of notes. */
function noteOf(notes: Notification[], field: string): Notification {
  const note = notes.find((n) => n.data.change.field === field);
  if (!note) {
    throw new Error(`no notification of ${field} among ${JSON.stringify(notes)}`);
  }
  return note;
}

/**
 * A workspace of this run, alice its admin, bob a member, and its project Web, of which alice is the admin and bob a
 * member, with his membership of it.
 */
async function acmeWeb(api: Api, testInfo: TestInfo) {
  const [alice, bob] = await Promise.all([newAccount(api, testInfo, "alice"), newAccount(api, testInfo, "bob")]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  const [membership] = await addProjectMembers(api, alice.token, web.id, [{ member_id: bob.id, role: 15 }]);
  if (!membership) {
    throw new Error("bob was not added to Web");
  }
  return { alice, bob, slug, web, membership };
}

const notFound = { status: 404, code: "notification.not_found" };

test("H5–H7 (API): a member reads, archives, snoozes and unsnoozes his notifications and marks one unread, alike with a personal access token and his browser session, and the filters and the two tabs' unread counts follow; a snooze ends by itself; marking the all tab read leaves the mentions and the archived unread; a past snooze is refused, and another's notification as none", async ({
  api,
}, testInfo) => {
  const { alice, bob, slug, web } = await acmeWeb(api, testInfo);
  // alice's Fix for bob, her comment on it and her comment mentioning him: three in his all tab, one in his mentions.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await createIssueComment(api, alice.token, fix.id, "<p>Looks good</p>");
  await createIssueComment(
    api,
    alice.token,
    fix.id,
    `<p>Hello</p><mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`
  );
  const pat = { what: "his personal access token", token: bob.token };
  const session = { what: "his browser session", token: (await login(api, bob.email)).access_token };
  const [all, mentions] = await Promise.all([
    notificationsSent(api, bob.token, slug, 3),
    notificationsSent(api, bob.token, slug, 1, { tab: "mentions" }),
  ]);
  const [looks, creation, assignment, mention] = [
    noteOf(all, "comment"),
    noteOf(all, "issue"),
    noteOf(all, "assignees"),
    noteOf(mentions, "comment"),
  ];
  /** The notifications of all that are among notes, in the list's order. */
  const listed = (...notes: Notification[]) => all.filter((n) => notes.includes(n));

  // bob's unread counts are counts, alike with each credential, and his lists of the queries given those given.
  const shows = async (
    counts: { all: number; mentions: number },
    lists: [NonNullable<Parameters<typeof listNotifications>[3]>, Notification[]][]
  ) => {
    await Promise.all([
      ...[pat, session].map(async ({ what, token }) =>
        expect(await unreadNotifications(api, token, slug), `bob's unread counts with ${what}`).toEqual(counts)
      ),
      ...lists.map(async ([query, notes]) =>
        expect(
          ids((await listNotifications(api, bob.token, slug, query)).data),
          `bob's ${JSON.stringify(query)}`
        ).toEqual(ids(notes))
      ),
    ]);
  };
  // bob's change of note with the credential given, answered 200: the notification as changed.
  const change = async (
    { what, token }: typeof pat,
    note: Notification,
    body: Parameters<typeof notificationUpdated>[3]
  ) => {
    const changed = await notificationUpdated(api, token, note.id, body);
    expect(
      changed.status,
      `${JSON.stringify(body)} of ${note.data.change.field} with ${what}: ${JSON.stringify(changed)}`
    ).toBe(200);
    if (!("notification" in changed)) {
      throw new Error(`the change of ${note.id} answered 200 without the notification`);
    }
    return changed.notification;
  };

  expect((await listNotifications(api, session.token, slug)).data, `bob's all tab with ${session.what}`).toEqual(all);
  await shows({ all: 3, mentions: 1 }, [
    [{ read: "false" }, all],
    [{ tab: "mentions" }, [mention]],
  ]);

  // Read with his session, and again with his token: one moment, the first's.
  const read = await change(session, looks, { read: true });
  expect(read, "the comment read").toEqual({ ...looks, read_at: expect.any(String) });
  expect(await change(pat, looks, { read: true }), "the comment read again").toEqual(read);
  await shows({ all: 2, mentions: 1 }, [
    [{ read: "true" }, [looks]],
    [{ read: "false" }, listed(creation, assignment)],
  ]);

  // Archived with his token: listed only with archived=true, and no more counted, though unread.
  expect(await change(pat, assignment, { archived: true }), "the assignment archived").toEqual({
    ...assignment,
    archived_at: expect.any(String),
  });
  await shows({ all: 1, mentions: 1 }, [
    [{}, listed(looks, creation)],
    [{ archived: "true" }, [assignment]],
  ]);

  // Snoozed for an hour with his session: listed only with snoozed=true, and no more counted, till then.
  const hour = 60 * 60 * 1000;
  const till = new Date(Date.now() + hour).toISOString();
  const snoozed = await change(session, creation, { snoozed_till: till });
  expect(snoozed, "the creation snoozed").toEqual({ ...creation, snoozed_till: expect.any(String) });
  expect(Date.parse(snoozed.snoozed_till ?? ""), "the creation's snoozed_till").toBe(Date.parse(till));
  await shows({ all: 0, mentions: 1 }, [
    [{}, [looks]],
    [{ snoozed: "true" }, [creation]],
  ]);

  // A snooze till a moment past is refused on his own; alice's change of his, with that body or another, is refused as
  // a change of no notification is.
  const past = new Date(Date.now() - hour).toISOString();
  expect(
    await notificationUpdated(api, pat.token, creation.id, { snoozed_till: past }),
    "bob's snooze till the past"
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "snoozed_till", code: "must_be_future" }],
  });
  const refused = [{ read: true }, { snoozed_till: past }].flatMap((body) => [
    { what: `alice's ${JSON.stringify(body)} of bob's`, id: creation.id, body },
    { what: `alice's ${JSON.stringify(body)} of none`, id: crypto.randomUUID(), body },
  ]);
  await Promise.all(
    refused.map(async ({ what, id, body }) =>
      expect(await notificationUpdated(api, alice.token, id, body), what).toEqual(notFound)
    )
  );

  // Unsnoozed with his token: as it was.
  expect(await change(pat, creation, { snoozed_till: null }), "the creation unsnoozed").toEqual(creation);
  await shows({ all: 1, mentions: 1 }, [
    [{}, listed(looks, creation)],
    [{ snoozed: "true" }, []],
  ]);

  // Marking the all tab read with his session marks its unread that are neither archived nor snoozed, the creation
  // alone: the mention, of the other tab, and the archived assignment stay unread.
  expect(await allNotificationsMarked(api, session.token, slug, { tab: "all" }), "the all tab marked read").toEqual({
    status: 200,
    marked: 1,
  });
  await shows({ all: 0, mentions: 1 }, [
    [{ read: "false" }, []],
    [{ tab: "mentions", read: "false" }, [mention]],
    [{ archived: "true", read: "false" }, [assignment]],
  ]);

  // Unread again with his token.
  expect(await change(pat, looks, { read: false }), "the comment unread").toEqual(looks);
  await shows({ all: 1, mentions: 1 }, [[{ read: "false" }, [looks]]]);

  // Snoozed till a second from now with his session, the comment is counted again once that second is over, with no
  // write: not before, as nerve reads the snooze at the moment of each read.
  const soon = Date.now() + 1000;
  await change(session, looks, { snoozed_till: new Date(soon).toISOString() });
  await expect
    .poll(() => unreadNotifications(api, pat.token, slug), { intervals: [50, 100], message: "the snooze over" })
    .toEqual({ all: 1, mentions: 1 });
  expect(Date.now(), "counted again only once its snooze is over").toBeGreaterThanOrEqual(soon);
  await shows({ all: 1, mentions: 1 }, [
    [{ read: "false" }, [looks]],
    [{ snoozed: "true" }, []],
  ]);
});

test("H5 (API): a member removed from a project is listed, counted, and lets change and mark read none of its notifications, those of the workspace's other project still; added back, he is listed them again, unread as they were", async ({
  api,
}, testInfo) => {
  const { alice, bob, slug, web, membership } = await acmeWeb(api, testInfo);
  const ops = await createProject(api, alice.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, alice.token, ops.id, [{ member_id: bob.id, role: 15 }]);
  await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await createIssue(api, alice.token, ops.id, { name: "Plan", assignee_ids: [bob.id] });
  const all = await notificationsSent(api, bob.token, slug, 4);
  const [webs, opss] = [all.filter((n) => n.project_id === web.id), all.filter((n) => n.project_id === ops.id)];
  expect([webs.length, opss.length], "bob's notifications of Web and of Ops").toEqual([2, 2]);
  expect(await unreadNotifications(api, bob.token, slug), "bob's unread counts in both").toEqual({
    all: 4,
    mentions: 0,
  });

  expect(
    await projectMemberWrites(api, web.id).remove(alice.token, membership.id),
    "alice removes bob from Web"
  ).toEqual({
    status: 204,
  });
  expect(ids((await listNotifications(api, bob.token, slug)).data), "bob's notifications out of Web").toEqual(
    ids(opss)
  );
  expect(await unreadNotifications(api, bob.token, slug), "bob's unread counts out of Web").toEqual({
    all: 2,
    mentions: 0,
  });
  await Promise.all(
    webs.map(async (note) =>
      expect(await notificationUpdated(api, bob.token, note.id, { read: true }), "bob's read of one of Web").toEqual(
        notFound
      )
    )
  );
  expect(await allNotificationsMarked(api, bob.token, slug, {}), "bob marks all read out of Web").toEqual({
    status: 200,
    marked: 2,
  });

  await addProjectMembers(api, alice.token, web.id, [{ member_id: bob.id, role: 15 }]);
  const back = (await listNotifications(api, bob.token, slug)).data;
  expect(ids(back), "bob's notifications back in Web").toEqual(ids(all));
  expect(
    back.filter((n) => n.project_id === web.id),
    "Web's, as they were"
  ).toEqual(webs);
  expect(ids((await listNotifications(api, bob.token, slug, { read: "true" })).data), "the read, Ops'").toEqual(
    ids(opss)
  );
  expect(await unreadNotifications(api, bob.token, slug), "bob's unread counts back in Web").toEqual({
    all: 2,
    mentions: 0,
  });
});
