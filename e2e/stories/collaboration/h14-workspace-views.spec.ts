import { randomUUID } from "node:crypto";

import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import { validationFailed } from "../../fixtures/issue-planning";
import { createIssue, listWorkspaceIssues } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";
import { savedQuery, viewCalls, viewDefaults, viewOf, viewRow } from "../../fixtures/views";

// H14, a workspace's own saved views (M7 design 2, 3.7, 3.10; P3b spec 2.2, 2.4, 2.5, R1), and H13's saved workspace
// view: the API version. A member of a workspace makes a view of no project, the defaults filling what he leaves out,
// his, as stored; lists it, reads it and saves words in it, which the workspace's list of work items across its
// projects takes back as its query (H13); deletes it. A workspace view holds the workspace list's words: no cycle. Only
// its owner changes it, and a member deletes only his own, the workspace's admin another's too, the deletion his as
// stored. The workspace's guest has no saved workspace view: his list, his creation and his read are refused. A
// personal access token and a browser session alike. Each word a view stores, the refusals' order and the
// interleavings are the bootstrap tests' (TestWorkspaceViewsThroughTheApp, TestViewWritesInterleave) and the
// permission matrix's; the pages are P6b's.

test("H14 (API): bob, acme's member, makes a view of acme's own, the defaults filling the rest, lists it, reads it and saves in it his top-level work items of Alpha, the oldest first, which acme's list across its projects takes back (H13), and deletes it; a workspace view takes no cycle; carl, a member, neither changes nor deletes bob's view, and alice, acme's admin, deletes carl's, each deletion its deleter's as stored; gus, acme's guest, may neither list, make nor read one; with personal access tokens and browser sessions alike", async ({
  api,
  db,
}, testInfo) => {
  // Each account with its personal access token (newAccount) and a browser session besides.
  const signedIn = async (label: string) => {
    const account = await newAccount(api, testInfo, label);
    return { ...account, session: (await login(api, account.email)).access_token };
  };
  const [alice, bob, carl, gus] = await Promise.all([
    signedIn("alice"),
    signedIn("bob"),
    signedIn("carl"),
    signedIn("gus"),
  ]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, carl, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 5);
  // alice's Alpha and Web, bob a member of each, and her work items there, one after another: in Alpha, bob's first,
  // its child, also his, another of no one's, and bob's second; in Web, bob's. acme's list shows him the five.
  const alpha = await createProject(api, alice.token, slug, { name: "Alpha", identifier: "ALP" });
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, alpha.id, [{ member_id: bob.id, role: 15 }]);
  await addProjectMembers(api, alice.token, web.id, [{ member_id: bob.id, role: 15 }]);
  const first = await createIssue(api, alice.token, alpha.id, { name: "Alpha first", assignee_ids: [bob.id] });
  const child = await createIssue(api, alice.token, alpha.id, {
    name: "Alpha child",
    parent_id: first.id,
    assignee_ids: [bob.id],
  });
  const other = await createIssue(api, alice.token, alpha.id, { name: "Alpha other" });
  const second = await createIssue(api, alice.token, alpha.id, { name: "Alpha second", assignee_ids: [bob.id] });
  const webs = await createIssue(api, alice.token, web.id, { name: "Web's", assignee_ids: [bob.id] });
  expectIssuePage(await listWorkspaceIssues(api, bob.token, slug), [webs.id, second.id, other.id, child.id, first.id]);
  const views = viewCalls(api);
  const notFound = { status: 404, code: "navigation.view_not_found" };
  const forbidden = { status: 403, code: "forbidden" };

  // With each account's credential of the kind given, a round of a view of bob's and one of carl's, from their making
  // to their deletion.
  const round = async (credential: "token" | "session") => {
    const as = (account: typeof alice) => account[credential];

    // bob makes a view, its name alone given, the rest the defaults: his, of acme and no project, as stored.
    const made = viewOf(await views.createInWorkspace(as(bob), slug, { name: "  Mine  " }), 201);
    expect(made, "bob's view as made").toEqual({
      ...viewDefaults,
      id: made.id,
      workspace_id: acme.id,
      project_id: null,
      name: "Mine",
      owned_by_id: bob.id,
      created_at: made.created_at,
      updated_at: made.created_at,
    });
    const his = { project_id: null, owner: bob.id, made_by: bob.id, written_by: bob.id, deleted: false, stamped: true };
    expect(await viewRow(db, made.id), "bob's view as stored").toEqual(his);
    const carls = viewOf(await views.createInWorkspace(as(carl), slug, { name: "Carl's" }), 201);
    expect(await views.listInWorkspace(as(bob), slug), "acme's views").toEqual({ status: 200, views: [carls, made] });
    expect(await views.read(as(bob), made.id), "bob's view, read").toEqual({ status: 200, view: made });

    // H13: he saves in it his top-level work items of Alpha, the oldest first. Read back, its words are acme's list's
    // query, which shows those alone, in that order.
    const words = { filters: { project_id: [alpha.id], assignee_id: [bob.id] } };
    const display = { ...viewDefaults.display_filters, order_by: "created_at", sub_issue: false };
    const saved = viewOf(await views.update(as(bob), made.id, { ...words, display_filters: display }), 200);
    expect(saved, "bob's view with his words").toEqual({
      ...made,
      ...words,
      display_filters: display,
      updated_at: saved.updated_at,
    });
    expect(saved.updated_at, "its last change").not.toBe(made.updated_at);
    const stored = viewOf(await views.read(as(bob), made.id), 200);
    expect(stored, "bob's view read back").toEqual(saved);
    expectIssuePage(await listWorkspaceIssues(api, as(bob), slug, savedQuery(stored)), [first.id, second.id]);

    // A cycle is a project's: a workspace view takes none.
    expect(
      await views.createInWorkspace(as(bob), slug, { name: "Sprint", filters: { cycle_id: [randomUUID()] } }),
      "a workspace view of a cycle"
    ).toEqual(validationFailed(["filters.cycle_id", "not_allowed"]));

    // carl neither changes bob's view nor deletes it; gus, acme's guest, may neither list, make nor read one.
    expect(
      [await views.update(as(carl), made.id, { name: "Carl's" }), await views.remove(as(carl), made.id)],
      "carl changes and deletes bob's view"
    ).toEqual([forbidden, forbidden]);
    expect(
      [
        await views.listInWorkspace(as(gus), slug),
        await views.createInWorkspace(as(gus), slug, { name: "Gus's" }),
        await views.read(as(gus), made.id),
      ],
      "gus lists, makes and reads acme's views"
    ).toEqual([forbidden, forbidden, forbidden]);

    // alice, acme's admin, deletes carl's view, and bob his own: each deleted by its deleter at its last write, they
    // answer as views that never were, and are listed no more.
    expect(await views.remove(as(alice), carls.id), "alice deletes carl's view").toEqual({ status: 204 });
    expect(await views.remove(as(bob), made.id), "bob deletes his view").toEqual({ status: 204 });
    expect([await viewRow(db, carls.id), await viewRow(db, made.id)], "the two views deleted, as stored").toEqual([
      { ...his, owner: carl.id, made_by: carl.id, written_by: alice.id, deleted: true },
      { ...his, deleted: true },
    ]);
    expect([await views.read(as(bob), carls.id), await views.read(as(bob), made.id)], "the two, deleted").toEqual([
      notFound,
      notFound,
    ]);
    expect(await views.listInWorkspace(as(bob), slug), "acme's views, the two deleted").toEqual({
      status: 200,
      views: [],
    });
  };

  await test.step("with personal access tokens", () => round("token"));
  await test.step("with browser sessions", () => round("session"));
});
