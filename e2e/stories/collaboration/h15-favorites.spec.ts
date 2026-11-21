import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  archiveProject,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { bearer, login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import {
  createFavorite,
  favoriteCalls,
  favoriteOf,
  favoriteRows,
  storedAs,
  type Favorite,
  type FavoriteCreate,
} from "../../fixtures/favorites";
import { validationFailed } from "../../fixtures/issue-planning";
import { createModule } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";
import { createProjectView, createWorkspaceView } from "../../fixtures/views";

// H15, one's favorites (M7 design 2, 3.8, 3.10; P3c spec 2.2–2.5, 2.7, R4, R7–R10): the API version. A member of a
// workspace favors a project, a cycle, a module, a project's view and a view of the workspace's own, each his, at the
// top, one step past his last, of the project the server finds from its target and answered with its target's name
// and icon; and makes a folder, its name without its white space. Favored again, a target answers the favorite he has
// and writes nothing. A target that is not there, another workspace's, one he does not see and an archived one are one
// refusal alike. The workspace's guest may not list or write favorites; one who is none of its members finds no
// workspace. A folder's name is his alone in any case, made or renamed; a folder holds no folder, and a parent is one
// of his folders. He moves a favorite in and out of a folder and places it; a name is a folder's. Another's favorite
// and one of a target he no longer sees answer him as an id that never was; out of a project his favorites of it are
// kept and not listed, and listed again once he joins it. A folder is deleted with what it holds, at one moment. A
// personal access token and a browser session alike. The list's ties, the toggles, the project-level guest cells, a
// change of nothing, both refusals in one answer, the refusals' order, the deletion window and the interleavings are
// the bootstrap tests' (TestFavoritesOfEveryKindThroughTheApp, TestFavoriteWritesInterleaveWithArchives) and the
// permission matrix's; the pages are P7's.

// The icons of Web, of alice's view of it and of bob's view of acme's own.
const webLogo = { in_use: "emoji", emoji: { value: "128640" } } as const;
const alicesLogo = { in_use: "icon", icon: { name: "star", color: "#ff0000" } } as const;
const acmesLogo = { in_use: "emoji", emoji: { value: "9889" } } as const;

/** The body of a favorite of the target of entity_id, of entity_type. */
function target(entity_type: FavoriteCreate["entity_type"], entity_id: string): FavoriteCreate {
  return { entity_type, entity_id };
}

test("H15 (API): bob, acme's member, favors his folder Daily, Web, its cycle Sprint, its module Core, alice's view of Web and his view of acme's own, each of the project the server finds and named and shown as its target is, and favored again a target answers the one he has, writing nothing; a cycle not there, Side and Away of Elsewhere, Vault, private, Old and Shelf, archived, and alice's view to walt, who sees only his own, are one refusal alike; gus, acme's guest, may not list or write favorites, and olga, none of acme, finds no workspace; bob's folders keep their names without white space, his alone in any case, made or renamed, hold no folder and are none of alice's or his favorites of targets; he moves Web's in and out of Work and places it last, a name a folder's alone; alice's folder answers him as an id that never was; out of Web he lists none of his favorites of it, kept, until he joins it again; Box is deleted with what it holds at one moment; with personal access tokens and browser sessions alike", async ({
  api,
  db,
}, testInfo) => {
  // Each account with its personal access token (newAccount) and a browser session besides.
  const signedIn = async (label: string) => {
    const account = await newAccount(api, testInfo, label);
    return { ...account, session: (await login(api, account.email)).access_token };
  };
  const [alice, bob, gus, walt, olga] = await Promise.all([
    signedIn("alice"),
    signedIn("bob"),
    signedIn("gus"),
    signedIn("walt"),
    signedIn("olga"),
  ]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 15);
  await inviteAndAccept(api, alice.token, slug, walt, 15);
  // alice's Web, public, with an icon and its cycles, modules and views turned on: bob its member, walt its guest, who
  // sees only his own work items there (Web does not show its guests everything). Its cycles Sprint and Next, its
  // module Core, alice's view of it, with an icon, and Old, a cycle that has ended, which she archives.
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB", logo_props: webLogo });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: walt.id, role: 5 },
  ]);
  const turnedOn = { cycle_view: true, module_view: true, issue_views_view: true };
  expect(await changeProject(api, alice.token, web.id, turnedOn)).toMatchObject({ status: 200 });
  const sprint = await createCycle(api, alice.token, web.id, { name: "Sprint" });
  const next = await createCycle(api, alice.token, web.id, { name: "Next" });
  const core = await createModule(api, alice.token, web.id, { name: "Core" });
  const alicesView = await createProjectView(api, alice.token, web.id, { name: "Alice's", logo_props: alicesLogo });
  const old = await createCycle(api, alice.token, web.id, {
    name: "Old",
    start_date: "2025-01-01",
    end_date: "2025-01-14",
  });
  expect(await cycleCalls(api).archive(alice.token, old.id), "alice archives Old").toEqual({ status: 200 });
  // bob's view of acme's own, with an icon; Vault, alice's private project, hers alone; Shelf, whose member bob is,
  // archived. Elsewhere, bob's own workspace, with his project Side and his view Away of its own.
  const acmesView = await createWorkspaceView(api, bob.token, slug, { name: "Acme's", logo_props: acmesLogo });
  const vault = await createProject(api, alice.token, slug, { name: "Vault", identifier: "VLT", network: 0 });
  const shelf = await createProject(api, alice.token, slug, { name: "Shelf", identifier: "SHF" });
  await addProjectMembers(api, alice.token, shelf.id, [{ member_id: bob.id, role: 15 }]);
  await archiveProject(api, alice.token, shelf.id);
  const elsewhere = slugFor(testInfo, "elsewhere");
  await createWorkspace(api, bob.token, { name: "Elsewhere", slug: elsewhere });
  const side = await createProject(api, bob.token, elsewhere, { name: "Side", identifier: "SID" });
  const away = await createWorkspaceView(api, bob.token, elsewhere, { name: "Away" });
  // alice's folder in acme; and gus's, made while he is acme's member, before alice makes him its guest.
  const alicesFolder = await createFavorite(api, alice.token, slug, { entity_type: "folder", name: "Alice's" });
  const gusFolder = await createFavorite(api, gus.token, slug, { entity_type: "folder", name: "Gus's" });
  await changeRole(api, alice.token, slug, gus.id, 5);

  const favorites = favoriteCalls(api);
  const refused = validationFailed(["entity_id", "not_allowed"]);
  const noParent = validationFailed(["parent_id", "not_allowed"]);
  const taken = { status: 409, code: "navigation.favorite_folder_name_taken" };
  const notFound = { status: 404, code: "navigation.favorite_not_found" };
  const forbidden = { status: 403, code: "forbidden" };
  const absent = randomUUID();

  // With each account's credential of the kind given, a round of bob's favorites in acme, from their making to their
  // deletion: each round starts with none.
  const round = async (credential: "token" | "session") => {
    const as = (account: typeof alice) => account[credential];
    // bob's favorites of ids as stored, in that order.
    const stored = async (...ids: string[]) => {
      const rows = await favoriteRows(db, bob.id, acme.id);
      return ids.map((id) => rows.find((row) => row.id === id));
    };
    const listed = async (...want: Favorite[]) =>
      expect(await favorites.list(as(bob), slug), "bob's favorites in acme").toEqual({ status: 200, favorites: want });
    // A change and a deletion of the favorite of id by account, each answered.
    const reach = async (account: typeof alice, id: string) => [
      await favorites.update(as(account), id, { sequence: 1 }),
      await favorites.remove(as(account), id),
    ];

    // bob favors each kind: his, at the top, each one step of 10000 past his last from 65535, of the project the
    // server finds from its target, none for acme's own view and a folder, named and shown as its target is now, a
    // cycle and a module with no icon, a folder with its own name without its white space.
    const made: Favorite[] = [];
    const favor = async (body: FavoriteCreate, project: string | null, name: string, logo: object) => {
      const f = favoriteOf(await favorites.create(as(bob), slug, body), 201);
      expect(f, `bob's favorite of ${name}`).toEqual({
        id: f.id,
        workspace_id: acme.id,
        project_id: project,
        entity_type: body.entity_type,
        entity_id: body.entity_id ?? null,
        parent_id: null,
        sequence: 65535 + 10000 * made.length,
        name,
        logo_props: logo,
        created_at: f.created_at,
      });
      made.push(f);
      return f;
    };
    const daily = await favor({ entity_type: "folder", name: "  Daily  " }, null, "Daily", {});
    const webs = await favor({ entity_type: "project", entity_id: web.id }, web.id, "Web", webLogo);
    const sprints = await favor({ entity_type: "cycle", entity_id: sprint.id }, web.id, "Sprint", {});
    const cores = await favor({ entity_type: "module", entity_id: core.id }, web.id, "Core", {});
    const alices = await favor({ entity_type: "view", entity_id: alicesView.id }, web.id, "Alice's", alicesLogo);
    const acmes = await favor({ entity_type: "view", entity_id: acmesView.id }, null, "Acme's", acmesLogo);
    await listed(acmes, alices, cores, sprints, webs, daily);
    expect(await stored(...made.map((f) => f.id)), "bob's favorites as stored").toEqual(
      made.map((f) => storedAs(f, bob.id))
    );

    // Favored again, at the top or into Daily, Sprint answers his favorite of it as it is, and nothing is written.
    const rows = await favoriteRows(db, bob.id, acme.id);
    expect(
      [
        await favorites.create(as(bob), slug, { entity_type: "cycle", entity_id: sprint.id }),
        await favorites.create(as(bob), slug, { entity_type: "cycle", entity_id: sprint.id, parent_id: daily.id }),
      ],
      "Sprint favored again"
    ).toEqual([
      { status: 200, favorite: sprints },
      { status: 200, favorite: sprints },
    ]);
    expect(await favoriteRows(db, bob.id, acme.id), "bob's favorites, Sprint favored again").toEqual(rows);

    // A cycle that is not there; Side and Away, Elsewhere's; Vault, which he is none of; Old and Shelf, archived; and
    // alice's view to walt, who sees only his own in Web: one refusal alike, and nothing is written.
    expect(
      [
        await favorites.create(as(bob), slug, target("cycle", absent)),
        await favorites.create(as(bob), slug, target("project", side.id)),
        await favorites.create(as(bob), slug, target("view", away.id)),
        await favorites.create(as(bob), slug, target("project", vault.id)),
        await favorites.create(as(bob), slug, target("cycle", old.id)),
        await favorites.create(as(bob), slug, target("project", shelf.id)),
        await favorites.create(as(walt), slug, target("view", alicesView.id)),
      ],
      "the targets bob and walt may not favor"
    ).toEqual(Array(7).fill(refused));
    expect(
      [await favoriteRows(db, bob.id, acme.id), await favoriteRows(db, walt.id, acme.id)],
      "bob's and walt's favorites, the refusals'"
    ).toEqual([rows, []]);

    // gus, acme's guest, may neither list, make, change nor delete a favorite, his folder kept; olga, none of acme's
    // members, finds no workspace.
    expect(
      [
        await favorites.list(as(gus), slug),
        await favorites.create(as(gus), slug, { entity_type: "folder", name: "Guest's" }),
        await favorites.update(as(gus), gusFolder.id, { name: "Guest's" }),
        await favorites.remove(as(gus), gusFolder.id),
      ],
      "gus lists, makes, changes and deletes favorites"
    ).toEqual([forbidden, forbidden, forbidden, forbidden]);
    expect(await favoriteRows(db, gus.id, acme.id), "gus's folder").toEqual([storedAs(gusFolder, gus.id)]);
    const nowhere = { status: 404, code: "workspace.not_found" };
    expect(
      [
        await favorites.list(as(olga), slug),
        await favorites.create(as(olga), slug, { entity_type: "folder", name: "Olga's" }),
      ],
      "olga lists and makes favorites in acme"
    ).toEqual([nowhere, nowhere]);

    // His folder Work keeps its name without its white space. Another of his named so in any case is refused, made
    // or renamed, and an empty name; renamed in another case, Work is not. A folder in a folder is refused, made or
    // moved.
    const work = favoriteOf(await favorites.create(as(bob), slug, { entity_type: "folder", name: "  Work  " }), 201);
    expect(work, "bob's folder Work").toEqual({
      ...daily,
      id: work.id,
      sequence: 125535,
      name: "Work",
      created_at: work.created_at,
    });
    const home = favoriteOf(await favorites.create(as(bob), slug, { entity_type: "folder", name: "Home" }), 201);
    expect(
      [
        await favorites.create(as(bob), slug, { entity_type: "folder", name: "work" }),
        await favorites.create(as(bob), slug, { entity_type: "folder", name: " WORK " }),
        await favorites.update(as(bob), home.id, { name: "wORK" }),
        await favorites.create(as(bob), slug, { entity_type: "folder", name: "   " }),
        await favorites.create(as(bob), slug, { entity_type: "folder", name: "Inner", parent_id: work.id }),
        await favorites.update(as(bob), home.id, { parent_id: work.id }),
      ],
      "bob's folders named as Work, empty, and in Work"
    ).toEqual([taken, taken, taken, validationFailed(["name", "too_short"]), noParent, noParent]);
    const renamed = favoriteOf(await favorites.update(as(bob), work.id, { name: "WORK" }), 200);
    expect(renamed, "Work renamed in another case").toEqual({ ...work, name: "WORK" });
    // A parent he may not name, alice's folder or his favorite of Web, which is none, is refused, made or moved.
    expect(
      [
        await favorites.create(as(bob), slug, { entity_type: "cycle", entity_id: next.id, parent_id: alicesFolder.id }),
        await favorites.create(as(bob), slug, { entity_type: "cycle", entity_id: next.id, parent_id: webs.id }),
        await favorites.update(as(bob), sprints.id, { parent_id: alicesFolder.id }),
        await favorites.update(as(bob), sprints.id, { parent_id: webs.id }),
      ],
      "bob's favorites in alice's folder and in his favorite of Web"
    ).toEqual([noParent, noParent, noParent, noParent]);

    // He moves his favorite of Web into Work, as his list shows, and out again, then places it last; a name is a
    // folder's alone.
    const inWork = favoriteOf(await favorites.update(as(bob), webs.id, { parent_id: work.id }), 200);
    expect(inWork, "Web's in Work").toEqual({ ...webs, parent_id: work.id });
    await listed(home, renamed, acmes, alices, cores, sprints, inWork, daily);
    expect(await favorites.update(as(bob), webs.id, { parent_id: null }), "Web's out of Work").toEqual({
      status: 200,
      favorite: webs,
    });
    const last = favoriteOf(await favorites.update(as(bob), webs.id, { sequence: 0.5 }), 200);
    expect(last, "Web's placed last").toEqual({ ...webs, sequence: 0.5 });
    await listed(home, renamed, acmes, alices, cores, sprints, daily, last);
    expect(await favorites.update(as(bob), webs.id, { name: "Site" }), "Web's renamed").toEqual(
      validationFailed(["name", "not_allowed"])
    );
    expect(await stored(webs.id, work.id), "Web's and Work as stored").toEqual([
      storedAs(last, bob.id),
      storedAs(renamed, bob.id),
    ]);

    // alice's folder answers bob's change and deletion as an id that never was, and his Daily answers her so.
    expect(await reach(bob, alicesFolder.id), "alice's folder, to bob").toEqual([notFound, notFound]);
    expect(await reach(bob, alicesFolder.id), "alice's folder, as an id that never was").toEqual(
      await reach(bob, absent)
    );
    expect(await reach(alice, daily.id), "bob's Daily, to alice").toEqual(await reach(alice, absent));

    // bob leaves Web: his favorites of Web, Sprint, Core and alice's view are kept and not listed, and answer as ids
    // that never were; joined again, he lists them as before.
    expect(await projectMemberWrites(api, web.id).leave(as(bob)), "bob leaves Web").toEqual({ status: 204 });
    await listed(home, renamed, acmes, daily);
    expect(await stored(sprints.id, cores.id, alices.id, webs.id), "bob's favorites of Web's, out of it").toEqual([
      storedAs(sprints, bob.id),
      storedAs(cores, bob.id),
      storedAs(alices, bob.id),
      storedAs(last, bob.id),
    ]);
    expect(await reach(bob, sprints.id), "his favorite of Sprint, out of Web").toEqual(await reach(bob, absent));
    const joined = await api.POST("/api/v0/projects/{project_id}/join", {
      params: { path: { project_id: web.id } },
      headers: bearer(as(bob)),
    });
    expect(joined.response.status, "bob joins Web again").toBe(200);
    await listed(home, renamed, acmes, alices, cores, sprints, daily, last);

    // Box, into which he moves his favorites of Web and of acme's view, is deleted with them at one moment, by him;
    // he lists none of the three, the rest kept as they were, and Box answers as an id that never was.
    const box = favoriteOf(await favorites.create(as(bob), slug, { entity_type: "folder", name: "Box" }), 201);
    const held = [
      favoriteOf(await favorites.update(as(bob), webs.id, { parent_id: box.id }), 200),
      favoriteOf(await favorites.update(as(bob), acmes.id, { parent_id: box.id }), 200),
    ];
    const kept = [daily, sprints, cores, alices, home].map((f) => f.id).concat(work.id);
    const keptRows = await stored(...kept);
    expect(await favorites.remove(as(bob), box.id), "bob deletes Box").toEqual({ status: 204 });
    const [boxRow] = await stored(box.id);
    expect(boxRow?.deleted, "Box's deletion").toEqual(expect.any(String));
    expect(await stored(box.id, ...held.map((f) => f.id)), "Box and what it held, deleted").toEqual(
      [box, ...held].map((f) => storedAs(f, bob.id, boxRow?.deleted ?? null))
    );
    expect(await stored(...kept), "bob's other favorites, Box deleted").toEqual(keptRows);
    await listed(home, renamed, alices, cores, sprints, daily);
    expect(await reach(bob, box.id), "Box, deleted").toEqual(await reach(bob, absent));

    // He deletes the rest, for the next round to make them again.
    expect(await Promise.all(kept.map((id) => favorites.remove(as(bob), id))), "bob deletes the rest").toEqual(
      kept.map(() => ({ status: 204 }))
    );
    await listed();
  };

  await test.step("with personal access tokens", () => round("token"));
  await test.step("with browser sessions", () => round("session"));
});
