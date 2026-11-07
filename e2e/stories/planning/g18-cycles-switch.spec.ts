import {
  addProjectMembers,
  archiveProject,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls, listCycles, readCycle } from "../../fixtures/cycles";
import { expect, test } from "../../fixtures/test";

// G18, a project's cycles turned off and on again, and its archive (M6 design 3.6, M6 P1 spec R7): the writes are
// refused, the reads answer as before, and what was written is all there again. The admin calls with a personal
// access token, the member with the token of his sign-in.

test("G18 (API): with Web's cycles off, every write of its member and its admin on them is refused and changes nothing, while its list, its archived catalog and each cycle are read as before; turned on again, its cycles are all there and the member renames Draft; Web archived, the writes are refused again and the reads still answer", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const mem = await newAccount(api, testInfo, "mem");
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: mem.id, role: 15 }]);
  const turn = async (on: boolean) => {
    const changed = await changeProject(api, admin.token, web.id, { cycle_view: on });
    expect(changed.status, `the admin turns Web's cycles ${on ? "on" : "off"}: ${JSON.stringify(changed)}`).toBe(200);
  };
  await turn(true);
  const browser = (await login(api, mem.email)).access_token;
  const calls = cycleCalls(api);

  // Old ended in 2025 and is archived; Sprint, the admin's, ended in 2025 too; Draft, mem's, has no days.
  const old = await createCycle(api, admin.token, web.id, {
    name: "Old",
    start_date: "2025-01-01",
    end_date: "2025-01-14",
  });
  expect(await calls.archive(admin.token, old.id), "the admin archives Old").toEqual({ status: 200 });
  const sprint = await createCycle(api, admin.token, web.id, {
    name: "Sprint",
    start_date: "2025-02-01",
    end_date: "2025-02-14",
  });
  const draft = await createCycle(api, browser, web.id, { name: "Draft" });
  let cycles: CycleRow[] = [
    {
      name: "Draft",
      description: "",
      start: null,
      end: null,
      sort_order: 45535,
      archived: false,
      deleted: false,
      owner: mem.email,
      by: mem.email,
    },
    {
      name: "Old",
      description: "",
      start: "2025-01-01",
      end: "2025-01-14",
      sort_order: 65535,
      archived: true,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
    {
      name: "Sprint",
      description: "",
      start: "2025-02-01",
      end: "2025-02-14",
      sort_order: 55535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
  ];
  await expectCycles(db, web.id, cycles);
  /** What each token reads of Web's cycles: its list, its archived catalog, and each cycle by its id. */
  const reads = async (token: string) => [
    await listCycles(api, token, web.id),
    await listCycles(api, token, web.id, true),
    ...(await Promise.all([old, sprint, draft].map((c) => readCycle(api, token, c.id)))),
  ];
  const before = await reads(browser);

  /** Each write on Web's cycles, by mem's sign-in or the admin, each one a write that would pass were it allowed. */
  const writes = () =>
    Promise.all([
      calls.create(browser, web.id, { name: "New" }),
      calls.update(browser, draft.id, { name: "Renamed" }),
      calls.update(admin.token, sprint.id, { sort_order: 1000 }),
      calls.archive(admin.token, sprint.id),
      calls.unarchive(admin.token, old.id),
      calls.remove(browser, draft.id),
      calls.remove(admin.token, sprint.id),
    ]);

  // Off: each write refused, nothing written; the reads answer as before, with either of mem's tokens.
  await turn(false);
  expect(await writes(), "the writes on Web's cycles, off").toEqual(
    Array(7).fill({ status: 409, code: "planning.cycles_disabled" })
  );
  await expectCycles(db, web.id, cycles);
  expect(await reads(browser), "Web's cycles read by mem's sign-in, off").toEqual(before);
  expect(await reads(mem.token), "Web's cycles read by mem's personal access token, off").toEqual(before);

  // On again: all there, as before; mem renames Draft.
  await turn(true);
  expect(await reads(browser), "Web's cycles read by mem, on again").toEqual(before);
  expect(await calls.update(browser, draft.id, { name: "Renamed" }), "mem renames Draft").toEqual({ status: 200 });
  cycles = cycles.map((c) => (c.name === "Draft" ? { ...c, name: "Renamed" } : c));
  await expectCycles(db, web.id, cycles);

  // Web archived, its cycles on: each write refused as the project's, nothing written; the reads still answer.
  const renamed = await reads(browser);
  await archiveProject(api, admin.token, web.id);
  expect(await writes(), "the writes on archived Web's cycles").toEqual(
    Array(7).fill({ status: 409, code: "project.archived" })
  );
  await expectCycles(db, web.id, cycles);
  expect(await reads(browser), "archived Web's cycles read by mem").toEqual(renamed);
});
