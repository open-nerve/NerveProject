import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle } from "../../fixtures/cycles";
import { createIssue, listIssues } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";
import { createProjectView, savedQuery, viewCalls, viewDefaults, viewOf, viewRow } from "../../fixtures/views";

// H12, a project's saved views (M7 design 2, 3.7; P3b spec 2.2, 2.4, 2.5): the API version. A project's views are off
// until its admin turns them on; then its member makes a view, the defaults filling what he leaves out, his, as stored;
// lists it, reads it, renames it and saves words in it, which the project's list takes back as its query. Only its
// owner changes it, and a member deletes only his own, the project's admin another's too, the deletion his as stored.
// Turned off, the views refuse every write and are read as before. A guest who sees only his own work items lists only
// his own views, and another's answers him as an id that never was, as every view of a private project does a member
// of the workspace who is not the project's. A personal access token and a browser session alike. Each word a view
// stores, the refusals' order, the workspace admin's cases, an archived project, the list's order, the deletion window
// and the interleavings are the bootstrap tests' (TestProjectViewsThroughTheApp, TestViewWritesInterleave) and the
// permission matrix's; the pages are P6a's.

test("H12 (API): Web's views, off until alice, its admin, turns them on; bob, its member, makes a view, the defaults filling the rest, lists it, reads it, renames it and saves urgent work items of Sprint in it, which Web's list takes back; carl, a member, neither changes nor deletes it; gus, a guest who sees only his own, lists his own alone and olga, no member of Web, private, sees none, bob's answering each as an id that never was; turned off, Web's views refuse the writes and are read as before; alice deletes bob's view, the deletion hers as stored; with personal access tokens and browser sessions alike", async ({
  api,
  db,
}, testInfo) => {
  // Each account with its personal access token (newAccount) and a browser session besides.
  const signedIn = async (label: string) => {
    const account = await newAccount(api, testInfo, label);
    return { ...account, session: (await login(api, account.email)).access_token };
  };
  const [alice, bob, carl, gus, olga] = await Promise.all([
    signedIn("alice"),
    signedIn("bob"),
    signedIn("carl"),
    signedIn("gus"),
    signedIn("olga"),
  ]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, carl, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 15);
  await inviteAndAccept(api, alice.token, slug, olga, 15);
  // alice's Web, private: bob, carl and gus its members, olga not.
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB", network: 0 });
  await addProjectMembers(
    api,
    alice.token,
    web.id,
    [bob, carl, gus].map(({ id }) => ({ member_id: id, role: 15 as const }))
  );
  const views = viewCalls(api);
  const notFound = { status: 404, code: "navigation.view_not_found" };
  const forbidden = { status: 403, code: "forbidden" };
  const disabled = { status: 409, code: "navigation.views_disabled" };

  // Web's views are off by default; alice turns them on, and its cycles.
  expect(await views.createInProject(bob.token, web.id, { name: "Early" }), "a view while Web's are off").toEqual(
    disabled
  );
  expect(await changeProject(api, alice.token, web.id, { issue_views_view: true, cycle_view: true })).toMatchObject({
    status: 200,
  });
  // gus makes a view of his own, then alice makes him Web's guest: Web does not show its guests everything.
  const gusView = await createProjectView(api, gus.token, web.id, { name: "Gus's" });
  const gusMembership = await projectMembershipOf(api, alice.token, web.id, gus.id);
  expect(await projectMemberWrites(api, web.id).change(alice.token, gusMembership, 5), "gus, Web's guest").toEqual({
    status: 200,
  });
  // alice's work items of Web, for a view's words to filter: urgent and in Sprint, a cycle of Web; urgent alone; in
  // Sprint alone. bob's list shows the three.
  const sprint = await createCycle(api, alice.token, web.id, { name: "Sprint" });
  const both = await createIssue(api, alice.token, web.id, {
    name: "Urgent in Sprint",
    priority: "urgent",
    cycle_id: sprint.id,
  });
  const urgent = await createIssue(api, alice.token, web.id, { name: "Urgent", priority: "urgent" });
  const planned = await createIssue(api, alice.token, web.id, { name: "In Sprint", cycle_id: sprint.id });
  expectIssuePage(await listIssues(api, bob.token, web.id), [planned.id, urgent.id, both.id]);
  const absent = randomUUID();

  // With each account's credential of the kind given, a round of a view of bob's, from its making to its deletion.
  const round = async (credential: "token" | "session") => {
    const as = (account: typeof alice) => account[credential];

    // bob makes a view, its name alone given, the rest the defaults: his, of Web, as stored.
    const made = viewOf(await views.createInProject(as(bob), web.id, { name: "  Bob's  " }), 201);
    expect(made, "bob's view as made").toEqual({
      ...viewDefaults,
      id: made.id,
      workspace_id: acme.id,
      project_id: web.id,
      name: "Bob's",
      owned_by_id: bob.id,
      created_at: made.created_at,
      updated_at: made.created_at,
    });
    const his = {
      project_id: web.id,
      owner: bob.id,
      made_by: bob.id,
      written_by: bob.id,
      deleted: false,
      stamped: true,
    };
    expect(await viewRow(db, made.id), "bob's view as stored").toEqual(his);
    expect(await views.listInProject(as(bob), web.id), "Web's views").toEqual({ status: 200, views: [made, gusView] });
    expect(await views.read(as(bob), made.id), "bob's view, read").toEqual({ status: 200, view: made });

    // He renames it and saves urgent work items of Sprint in it, which Web's list takes back as its query.
    const words = { priority: ["urgent"], cycle_id: [sprint.id] };
    const saved = viewOf(await views.update(as(bob), made.id, { name: "Urgent in Sprint", filters: words }), 200);
    expect(saved, "bob's view renamed").toEqual({
      ...made,
      name: "Urgent in Sprint",
      filters: words,
      updated_at: saved.updated_at,
    });
    expect(saved.updated_at, "its last change").not.toBe(made.updated_at);
    expect(await viewRow(db, made.id), "bob's view renamed, as stored").toEqual(his);
    expectIssuePage(await listIssues(api, as(bob), web.id, savedQuery(saved)), [both.id]);

    // carl, a member of Web, neither changes bob's view nor deletes it.
    expect(
      [await views.update(as(carl), made.id, { name: "Carl's" }), await views.remove(as(carl), made.id)],
      "carl changes and deletes bob's view"
    ).toEqual([forbidden, forbidden]);
    // gus lists his own alone; olga, no member of Web, private, sees none. bob's view, read, changed and deleted,
    // answers each as an id that never was.
    expect(await views.listInProject(as(gus), web.id), "Web's views, as gus lists them").toEqual({
      status: 200,
      views: [gusView],
    });
    expect(await views.listInProject(as(olga), web.id), "Web's views, as olga lists them").toEqual({
      status: 404,
      code: "project.not_found",
    });
    await Promise.all(
      (
        [
          [gus, "gus"],
          [olga, "olga"],
        ] as const
      ).map(async ([account, who]) => {
        const each = async (id: string) => [
          await views.read(as(account), id),
          await views.update(as(account), id, { name: "Taken" }),
          await views.remove(as(account), id),
        ];
        const answers = await each(made.id);
        expect(answers, `bob's view, as ${who} reads, changes and deletes it`).toEqual([notFound, notFound, notFound]);
        expect(answers, `bob's view, as an id that never was, to ${who}`).toEqual(await each(absent));
      })
    );

    // Turned off, Web's views refuse bob's writes, and are read as before.
    expect(await changeProject(api, as(alice), web.id, { issue_views_view: false })).toMatchObject({ status: 200 });
    expect(
      [
        await views.createInProject(as(bob), web.id, { name: "Off" }),
        await views.update(as(bob), made.id, { name: "Off" }),
        await views.remove(as(bob), made.id),
      ],
      "bob's writes, Web's views off"
    ).toEqual([disabled, disabled, disabled]);
    expect(await views.listInProject(as(bob), web.id), "Web's views, off").toEqual({
      status: 200,
      views: [saved, gusView],
    });
    expect(await views.read(as(bob), made.id), "bob's view, off").toEqual({ status: 200, view: saved });
    expect(await changeProject(api, as(alice), web.id, { issue_views_view: true })).toMatchObject({ status: 200 });

    // alice, Web's admin, deletes bob's view: deleted by her at its last write, it answers as one that never was.
    expect(await views.remove(as(alice), made.id), "alice deletes bob's view").toEqual({ status: 204 });
    expect(await viewRow(db, made.id), "bob's view deleted, as stored").toEqual({
      ...his,
      written_by: alice.id,
      deleted: true,
    });
    expect(await views.read(as(bob), made.id), "bob's view, deleted").toEqual(notFound);
    expect(await views.listInProject(as(bob), web.id), "Web's views, bob's deleted").toEqual({
      status: 200,
      views: [gusView],
    });
  };

  await test.step("with personal access tokens", () => round("token"));
  await test.step("with browser sessions", () => round("session"));
});
