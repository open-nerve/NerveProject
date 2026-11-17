import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import {
  createIssue,
  issueRead,
  issueSubscribed,
  issueUnsubscribed,
  listIssues,
  listMemberIssues,
} from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// H1, subscribing to a work item and unsubscribing from it (M7 design 2, 3.1; P1a spec 2.3, 2.4): the API version.
// Who may read a work item subscribes to it and unsubscribes from it, each 204 without a body, a second time too; the
// work item in full says whether its caller subscribes (is_subscribed), and the lists of work items by subscriber
// (subscriber_id, and a member's relation=subscribed) follow. A subscription is a row of the work item's subscribers,
// made by its subscriber; an unsubscription deletes it, by her, at its moment. The refusals, the locks and the
// interleavings are the bootstrap tests' and the permission matrix's.

/**
 * The rows of the account of accountId among the subscribers of the work item of issueId, the deleted ones too, in the
 * order they were made: whether it is live; whether it is of the work item's workspace and project, and made and last
 * written by its subscriber; and whether its moments are a subscription's, made and last written at one (live), or an
 * unsubscription's, deleted at its last write, after it was made.
 */
async function subscriptionRows(db: Database, issueId: string, accountId: string) {
  return db.query<{ live: boolean; hers: boolean; stamped: boolean }>(
    `SELECT s.deleted_at IS NULL AS live,
            s.workspace_id = i.workspace_id AND s.project_id = i.project_id
              AND s.created_by_id = s.subscriber_id AND s.updated_by_id = s.subscriber_id AS hers,
            CASE WHEN s.deleted_at IS NULL THEN s.updated_at = s.created_at
                 ELSE s.deleted_at = s.updated_at AND s.updated_at > s.created_at END AS stamped
       FROM issue_subscribers s JOIN issues i ON i.id = s.issue_id
      WHERE s.issue_id = $1 AND s.subscriber_id = $2 ORDER BY s.created_at, s.id`,
    [issueId, accountId]
  );
}

test("H1 (API): a member subscribes to a work item and unsubscribes from it, each twice, alike with a personal access token and her browser session; is_subscribed, the subscriber_id list and her subscribed list follow, and each subscription is a row of hers, which her unsubscription deletes", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const ann = await newAccount(api, testInfo, "ann");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, ann, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: ann.id, role: 15 }]);
  // The admin's Fix the login, which subscribes him, its creator, and not ann.
  const fix = await createIssue(api, admin.token, web.id, { name: "Fix the login" });
  const pat = { what: "her personal access token", token: ann.token };
  const session = { what: "her browser session", token: (await login(api, ann.email)).access_token };

  // Whether ann subscribes to Fix, as each of her credentials reads it, and as the lists by subscriber show it.
  const follows = async (subscribed: boolean) => {
    const [byPAT, bySession] = await Promise.all([pat, session].map(({ token }) => issueRead(api, token, fix.id)));
    const read = { status: 200, issue: { id: fix.id, is_subscribed: subscribed } };
    expect(byPAT, `Fix read with ${pat.what}`).toMatchObject(read);
    expect(bySession, `Fix read with ${session.what}`).toMatchObject(read);
    const listed = subscribed ? [fix.id] : [];
    expectIssuePage(await listIssues(api, admin.token, web.id, { subscriber_id: [ann.id] }), listed);
    expectIssuePage(await listMemberIssues(api, ann.token, slug, ann.id, { relation: "subscribed" }), listed);
  };
  const live = { live: true, hers: true, stamped: true };
  const deleted = { live: false, hers: true, stamped: true };

  // With the credential given, ann subscribes, and again, then unsubscribes, and again: one row of hers, live, then
  // deleted, after her rows from before.
  const round = async ({ what, token }: typeof pat, before: (typeof deleted)[]) => {
    const step = async (write: typeof issueSubscribed, name: string, last: typeof live) => {
      expect(await write(api, token, fix.id), `${name} with ${what}`).toEqual({ status: 204 });
      expect(await subscriptionRows(db, fix.id, ann.id), `rows once ${name} with ${what}`).toEqual([...before, last]);
    };
    await step(issueSubscribed, "subscribed", live);
    await step(issueSubscribed, "subscribed again", live);
    await follows(true);
    await step(issueUnsubscribed, "unsubscribed", deleted);
    await step(issueUnsubscribed, "unsubscribed again", deleted);
    await follows(false);
  };

  await follows(false);
  expect(await subscriptionRows(db, fix.id, ann.id)).toEqual([]);
  await round(pat, []);
  await round(session, [deleted]);
});
