import {
  addProjectMembers,
  archiveProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
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
