import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls, listCycles } from "../../fixtures/cycles";
import { expect, test } from "../../fixtures/test";

// G1, creating a project's cycles (M6 design 2, M6 P1 spec 2.3): a draft from a name alone, a cycle with both of its
// days, the days and their overlap refused, and the cycles' switch. The admin calls with a personal access token,
// the member with the token of his sign-in, as the browser does.

test("G1 (API): Web's cycles are off, so its member's Backlog is refused; its admin turns them on; the member creates Backlog, a draft, from its name, at the first place; the admin creates Sprint 1, the 1st to the 14th of March 2025, before it, and the member Sprint 2, the day after it ends, before both; a blank name, a start or an end alone, an end before the start, days sharing Sprint 2's last, days sharing archived Sprint 1's first and a guest's cycle are refused, each changing nothing; of two cycles of the same days created at once one is; the member lists the same cycles with either of his tokens", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const [mem, guest] = await Promise.all(["mem", "guest"].map((label) => newAccount(api, testInfo, label)));
  if (!mem || !guest) {
    throw new Error("the accounts were not registered");
  }
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, guest, 5);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  // mem's token of his sign-in, the browser's, beside his personal access token.
  const browser = (await login(api, mem.email)).access_token;
  const { create, archive } = cycleCalls(api);

  // A new project's cycles are off (M6 design 3.6): mem's Backlog is refused, and nothing is written.
  expect(await create(browser, web.id, { name: "Backlog" }), "mem's Backlog, cycles off").toEqual({
    status: 409,
    code: "planning.cycles_disabled",
  });
  await expectCycles(db, web.id, []);
  const on = await changeProject(api, admin.token, web.id, { cycle_view: true });
  expect(on.status, `the admin turns Web's cycles on: ${JSON.stringify(on)}`).toBe(200);

  // mem creates Backlog from its name alone, without its surrounding spaces: a draft, at a project's first place,
  // owned by him.
  const backlog = await createCycle(api, browser, web.id, { name: "  Backlog " });
  expect(
    [
      backlog.name,
      backlog.description,
      backlog.start_date,
      backlog.end_date,
      backlog.status,
      backlog.sort_order,
      backlog.project_id,
      backlog.workspace_id === web.workspace_id,
      backlog.owned_by_id,
      backlog.created_by_id,
      backlog.updated_by_id,
      backlog.archived_at,
    ],
    "Backlog as created"
  ).toEqual(["Backlog", "", null, null, "draft", 65535, web.id, true, mem.id, mem.id, mem.id, null]);
  // The admin creates Sprint 1 with its days and a description, before Backlog; mem creates Sprint 2 from the day
  // after Sprint 1 ends, before both: the days of two cycles may touch without sharing one.
  const sprint1 = await createCycle(api, admin.token, web.id, {
    name: "Sprint 1",
    description: "Sign-in",
    start_date: "2025-03-01",
    end_date: "2025-03-14",
  });
  const sprint2 = await createCycle(api, browser, web.id, {
    name: "Sprint 2",
    start_date: "2025-03-15",
    end_date: "2025-03-28",
  });
  expect(
    [sprint1.status, sprint1.sort_order, sprint1.owned_by_id, sprint2.status, sprint2.sort_order, sprint2.owned_by_id],
    "Sprint 1 and Sprint 2 as created, their days past"
  ).toEqual(["completed", 55535, admin.id, "completed", 45535, mem.id]);
  let cycles: CycleRow[] = [
    {
      name: "Backlog",
      description: "",
      start: null,
      end: null,
      sort_order: 65535,
      archived: false,
      deleted: false,
      owner: mem.email,
      by: mem.email,
    },
    {
      name: "Sprint 1",
      description: "Sign-in",
      start: "2025-03-01",
      end: "2025-03-14",
      sort_order: 55535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
    {
      name: "Sprint 2",
      description: "",
      start: "2025-03-15",
      end: "2025-03-28",
      sort_order: 45535,
      archived: false,
      deleted: false,
      owner: mem.email,
      by: mem.email,
    },
  ];
  await expectCycles(db, web.id, cycles);

  // The admin archives Sprint 1, which ended: an archived cycle keeps its days from the others' (M6 design 3.3).
  expect(await archive(admin.token, sprint1.id), "the admin archives Sprint 1").toEqual({ status: 200 });
  cycles = cycles.map((c) => (c.name === "Sprint 1" ? { ...c, archived: true } : c));
  await expectCycles(db, web.id, cycles);

  // Refused, each changing nothing: a name of spaces; a start alone and an end alone; an end before the start; days
  // sharing the 28th of March, Sprint 2's last; days sharing the 1st of March, archived Sprint 1's first; the
  // guest's cycle, of days of their own.
  expect(
    [
      await create(browser, web.id, { name: "   " }),
      await create(browser, web.id, { name: "Sprint 3", start_date: "2025-04-01" }),
      await create(admin.token, web.id, { name: "Sprint 3", end_date: "2025-04-14" }),
      await create(browser, web.id, { name: "Sprint 3", start_date: "2025-04-14", end_date: "2025-04-01" }),
      await create(browser, web.id, { name: "Sprint 3", start_date: "2025-03-28", end_date: "2025-04-10" }),
      await create(admin.token, web.id, { name: "Sprint 0", start_date: "2025-02-15", end_date: "2025-03-01" }),
      await create(guest.token, web.id, { name: "Guest's", start_date: "2025-06-01", end_date: "2025-06-14" }),
    ],
    "a blank name, a start alone, an end alone, an end before the start, Sprint 2's last day, Sprint 1's first, the guest's"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "name", code: "too_short" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "end_date", code: "required" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "start_date", code: "required" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "end_date", code: "not_allowed" }] },
    { status: 409, code: "planning.cycle_dates_overlap" },
    { status: 409, code: "planning.cycle_dates_overlap" },
    { status: 403, code: "forbidden" },
  ]);
  await expectCycles(db, web.id, cycles);

  // The admin and mem create Sprint 3, of the same days, at once: the second, decided after the first under the
  // project's lock, sees it (M6 design 6).
  const raced = await Promise.all([
    create(admin.token, web.id, { name: "Sprint 3", start_date: "2025-04-01", end_date: "2025-04-14" }),
    create(browser, web.id, { name: "Sprint 3", start_date: "2025-04-01", end_date: "2025-04-14" }),
  ]);
  expect(
    raced.toSorted((a, b) => a.status - b.status),
    "the admin's and mem's Sprint 3, created at once"
  ).toEqual([{ status: 201 }, { status: 409, code: "planning.cycle_dates_overlap" }]);
  const winner = raced[0]?.status === 201 ? admin.email : mem.email;
  cycles = [
    ...cycles,
    {
      name: "Sprint 3",
      description: "",
      start: "2025-04-01",
      end: "2025-04-14",
      sort_order: 35535,
      archived: false,
      deleted: false,
      owner: winner,
      by: winner,
    },
  ];
  await expectCycles(db, web.id, cycles);

  // mem lists Web's cycles, archived Sprint 1 aside, by their places, the same with either of his tokens.
  const listed = await listCycles(api, browser, web.id);
  expect(
    listed.map((c) => [c.name, c.status]),
    "Web's cycles as mem lists them"
  ).toEqual([
    ["Sprint 3", "completed"],
    ["Sprint 2", "completed"],
    ["Backlog", "draft"],
  ]);
  expect(await listCycles(api, mem.token, web.id), "Web's cycles as mem's personal access token lists them").toEqual(
    listed
  );
});
