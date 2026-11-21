import {
  addProjectMembers,
  archiveProject,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { bearer, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import {
  createFavorite,
  favoriteCalls,
  favoriteRows,
  storedAs,
  targetStamp,
  type Favorite,
  type FavoriteCreate,
} from "../../fixtures/favorites";
import { createModule, moduleCalls } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";
import { createProjectView, createWorkspaceView, viewCalls } from "../../fixtures/views";

// H16, what follows a favorite's target (M7 design 2, 3.8, 6.1; P3c spec 2.6, R3, R10–R12): the API version. Two
// accounts favor the same targets. A cycle's and a module's deletion and archive, a project's view's deletion and a
// workspace's view's delete every account's favorites of it, at the moment and by the writer of the target's own
// deletion or archive, and neither lists them; unarchived, a target brings none back. A project's archive deletes so
// every account's favorites of it and of its cycles, modules and views, their folders and another project's kept;
// unarchived, it brings none back. Deleted, a project's favorites are listed no more at once, and its deletion's job
// deletes them at its moment, by its deleter. The favorites of the targets no step touches stay as they were. A cycle
// or a project archived again, a malformed event, the job's workspace step, the deletion window and the interleavings
// are the bootstrap tests' (TestFavoritesFollowTheirTargets, TestFavoritesOfEveryKindThroughTheApp,
// TestFavoriteWritesInterleaveWithArchives); the archive before the deletion is P4's (R12).

/** The lists of favored, alice's and bob's in turn as the steps favor them: alice's first, each the last made first. */
function listedAs(favored: Favorite[]) {
  return [0, 1].map((turn) => ({ status: 200, favorites: favored.filter((_, n) => n % 2 === turn).toReversed() }));
}

test("H16 (API): alice and bob favor the same targets; bob deletes his cycle and his module and archives alice's that have ended, and alice deletes bob's view of Web and his view of acme's own: each deletes both accounts' favorites of it at its own moment, by its writer, and unarchived brings none back; alice archives Arc, which deletes so both accounts' favorites of Arc and of its cycle, module and view, their folders Pins and their favorites of Web's kept, and unarchived brings none back; alice deletes Doomed, whose favorites neither lists at once, and its job deletes them at its moment, by her; the favorites of Web's targets stay as they were", async ({
  api,
  db,
}, testInfo) => {
  const [alice, bob] = await Promise.all([newAccount(api, testInfo, "alice"), newAccount(api, testInfo, "bob")]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  const favorites = favoriteCalls(api);
  const [cycles, modules, views] = [cycleCalls(api), moduleCalls(api), viewCalls(api)];

  // A project of alice's in acme, bob its member, its cycles, modules and views turned on.
  const project = async (name: string, identifier: string) => {
    const made = await createProject(api, alice.token, slug, { name, identifier });
    await addProjectMembers(api, alice.token, made.id, [{ member_id: bob.id, role: 15 }]);
    const turnedOn = { cycle_view: true, module_view: true, issue_views_view: true };
    expect(await changeProject(api, alice.token, made.id, turnedOn)).toMatchObject({ status: 200 });
    return made;
  };
  // alice's and bob's new favorites of the target of entity_id, or their folders of name, hers first.
  const bothFavor = async (body: FavoriteCreate) => [
    await createFavorite(api, alice.token, slug, body),
    await createFavorite(api, bob.token, slug, body),
  ];
  // The favorites of favored as stored, in that order.
  const stored = async (favored: Favorite[]) => {
    const rows = [...(await favoriteRows(db, alice.id, acme.id)), ...(await favoriteRows(db, bob.id, acme.id))];
    return favored.map(({ id }) => rows.find((row) => row.id === id));
  };
  // favored, alice's and bob's in turn as bothFavor makes them, as favoriteRows reads them: live, each its owner's; or
  // taken away at the moment and by the writer of stamp.
  const live = (favored: Favorite[]) => favored.map((f, n) => storedAs(f, n % 2 === 0 ? alice.id : bob.id));
  const takenAway = (favored: Favorite[], stamp: { at: string | null; by: string }) =>
    favored.map((f) => storedAs(f, stamp.by, stamp.at));
  // Each one's list in acme, alice's first.
  const lists = async () => [await favorites.list(alice.token, slug), await favorites.list(bob.token, slug)];

  // Web, and the favorites both keep through every step: of Web, its cycle Kept, its module Kept, bob's view Kept of
  // it and his view Kept of acme's own.
  const web = await project("Web", "WEB");
  const keptCycle = await createCycle(api, bob.token, web.id, { name: "Kept" });
  const keptModule = await createModule(api, bob.token, web.id, { name: "Kept" });
  const keptView = await createProjectView(api, bob.token, web.id, { name: "Kept" });
  const keptOwn = await createWorkspaceView(api, bob.token, slug, { name: "Kept" });
  const kept = [
    ...(await bothFavor({ entity_type: "project", entity_id: web.id })),
    ...(await bothFavor({ entity_type: "cycle", entity_id: keptCycle.id })),
    ...(await bothFavor({ entity_type: "module", entity_id: keptModule.id })),
    ...(await bothFavor({ entity_type: "view", entity_id: keptView.id })),
    ...(await bothFavor({ entity_type: "view", entity_id: keptOwn.id })),
  ];
  expect(await stored(kept), "the favorites kept, as stored").toEqual(live(kept));
  // What each one lists after each step: the favorites kept, and from Arc's archive on, each one's Pins.
  let shown = kept;
  expect(await lists(), "each one's favorites").toEqual(listedAs(shown));

  // Each target's deletion or archive, by its writer: it deletes both accounts' favorites of it as its own row was
  // written, at its moment and by him, and neither lists them; unarchived by alice, it brings none back.
  const ended = { start_date: "2025-01-01", end_date: "2025-01-14" };
  const steps = [
    {
      name: "bob deletes his cycle",
      kind: "cycle",
      table: "cycles",
      column: "deleted_at",
      writer: bob,
      make: async () => (await createCycle(api, bob.token, web.id, { name: "Mine" })).id,
      takeAway: (id: string) => cycles.remove(bob.token, id),
    },
    {
      name: "bob deletes his module",
      kind: "module",
      table: "modules",
      column: "deleted_at",
      writer: bob,
      make: async () => (await createModule(api, bob.token, web.id, { name: "Mine" })).id,
      takeAway: (id: string) => modules.remove(bob.token, id),
    },
    {
      name: "bob archives alice's cycle that has ended",
      kind: "cycle",
      table: "cycles",
      column: "archived_at",
      writer: bob,
      make: async () => (await createCycle(api, alice.token, web.id, { name: "Ended", ...ended })).id,
      takeAway: (id: string) => cycles.archive(bob.token, id),
      bringBack: (id: string) => cycles.unarchive(alice.token, id),
    },
    {
      name: "bob archives alice's completed module",
      kind: "module",
      table: "modules",
      column: "archived_at",
      writer: bob,
      make: async () => (await createModule(api, alice.token, web.id, { name: "Ended", status: "completed" })).id,
      takeAway: (id: string) => modules.archive(bob.token, id),
      bringBack: (id: string) => modules.unarchive(alice.token, id),
    },
    {
      name: "alice deletes bob's view of Web",
      kind: "view",
      table: "issue_views",
      column: "deleted_at",
      writer: alice,
      make: async () => (await createProjectView(api, bob.token, web.id, { name: "Mine" })).id,
      takeAway: (id: string) => views.remove(alice.token, id),
    },
    {
      name: "alice deletes bob's view of acme's own",
      kind: "view",
      table: "issue_views",
      column: "deleted_at",
      writer: alice,
      make: async () => (await createWorkspaceView(api, bob.token, slug, { name: "Mine" })).id,
      takeAway: (id: string) => views.remove(alice.token, id),
    },
  ] as const;
  for (const step of steps) {
    // eslint-disable-next-line no-await-in-loop -- Each step lists what the steps before it left.
    await test.step(step.name, async () => {
      const target = await step.make();
      const favored = await bothFavor({ entity_type: step.kind, entity_id: target });
      expect(await step.takeAway(target), step.name).toEqual({ status: step.column === "deleted_at" ? 204 : 200 });
      const stamp = await targetStamp(db, step.table, step.column, target);
      expect(stamp, `the ${step.column} of ${target}`).toEqual({ at: expect.any(String), by: step.writer.id });
      expect(await stored(favored), "both favorites, taken away").toEqual(takenAway(favored, stamp));
      expect(await lists(), "each one's favorites").toEqual(listedAs(shown));
      if ("bringBack" in step) {
        expect(await step.bringBack(target), `alice unarchives ${target}`).toEqual({ status: 200 });
        expect(await stored(favored), "both favorites, unarchived").toEqual(takenAway(favored, stamp));
        expect(await lists(), "each one's favorites, unarchived").toEqual(listedAs(shown));
      }
    });
  }

  await test.step("alice archives Arc", async () => {
    // Arc, with its cycle, its module and bob's view of it, each favored by both, as Arc is; and each one's folder
    // Pins. Archived, Arc deletes the eight at its moment, by alice; the folders are kept, as each one's list shows.
    const arc = await project("Arc", "ARC");
    const cycle = await createCycle(api, alice.token, arc.id, { name: "Arc's" });
    const module = await createModule(api, alice.token, arc.id, { name: "Arc's" });
    const view = await createProjectView(api, bob.token, arc.id, { name: "Arc's" });
    const favored = [
      ...(await bothFavor({ entity_type: "project", entity_id: arc.id })),
      ...(await bothFavor({ entity_type: "cycle", entity_id: cycle.id })),
      ...(await bothFavor({ entity_type: "module", entity_id: module.id })),
      ...(await bothFavor({ entity_type: "view", entity_id: view.id })),
    ];
    const pins = await bothFavor({ entity_type: "folder", name: "Pins" });
    await archiveProject(api, alice.token, arc.id);
    const stamp = await targetStamp(db, "projects", "archived_at", arc.id);
    expect(stamp, "Arc's archive").toEqual({ at: expect.any(String), by: alice.id });
    expect(await stored(favored), "the favorites of Arc's, taken away").toEqual(takenAway(favored, stamp));
    expect(await stored(pins), "each one's Pins").toEqual(live(pins));
    shown = [...kept, ...pins];
    expect(await lists(), "each one's favorites, Arc archived").toEqual(listedAs(shown));
    const unarchived = await api.POST("/api/v0/projects/{project_id}/unarchive", {
      params: { path: { project_id: arc.id } },
      headers: bearer(alice.token),
    });
    expect(unarchived.response.status, "alice unarchives Arc").toBe(200);
    expect(await stored(favored), "the favorites of Arc's, unarchived").toEqual(takenAway(favored, stamp));
    expect(await lists(), "each one's favorites, Arc unarchived").toEqual(listedAs(shown));
  });

  await test.step("alice deletes Doomed", async () => {
    // Doomed and its cycle, each favored by both. Deleted, neither lists them at once; its job deletes them at its
    // moment, by alice.
    const doomed = await project("Doomed", "DMD");
    const cycle = await createCycle(api, alice.token, doomed.id, { name: "Doomed's" });
    const favored = [
      ...(await bothFavor({ entity_type: "project", entity_id: doomed.id })),
      ...(await bothFavor({ entity_type: "cycle", entity_id: cycle.id })),
    ];
    const deleted = await api.DELETE("/api/v0/projects/{project_id}", {
      params: { path: { project_id: doomed.id } },
      headers: bearer(alice.token),
    });
    expect(deleted.response.status, "alice deletes Doomed").toBe(204);
    expect(await lists(), "each one's favorites, Doomed deleted").toEqual(listedAs(shown));
    const stamp = await targetStamp(db, "projects", "deleted_at", doomed.id);
    expect(stamp, "Doomed's deletion").toEqual({ at: expect.any(String), by: alice.id });
    await expect
      .poll(() => stored(favored), { message: "the favorites of Doomed's, deleted by its job", timeout: 15_000 })
      .toEqual(takenAway(favored, stamp));
  });

  expect(await stored(kept), "the favorites kept, after every step").toEqual(live(kept));
});
