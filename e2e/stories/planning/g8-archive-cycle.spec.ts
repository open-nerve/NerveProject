import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { cycleMoments, expectCycles, type CycleRow } from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls, listCycles, readCycle } from "../../fixtures/cycles";
import { expect, test } from "../../fixtures/test";

// G8, archiving a project's cycles and restoring them (M6 design 3.3, M6 P1 spec R3): a completed cycle alone, its
// project's archived catalog, the last archived first, and the list without it. The admin calls with a personal
// access token, the member with the token of his sign-in. A work item's cycle comes with P3.

/** rows with the cycle of name changed as to says. */
function changed(rows: CycleRow[], name: string, to: Partial<CycleRow>): CycleRow[] {
  return rows.map((c) => (c.name === name ? { ...c, ...to } : c));
}

test("G8 (API): of Web's Q1 and Q2, ended, Now, current, and Later, a draft, Now and Later may not be archived nor may a guest archive Q1; the member archives Q1, then Q2, a second archive of Q1 writing nothing; they leave Web's list for its archived catalog, Q2 first, and Q1 is still read; the admin restores Q1, a second restore writing nothing, and it is listed again in its place", async ({
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
  const on = await changeProject(api, admin.token, web.id, { cycle_view: true });
  expect(on.status, `the admin turns Web's cycles on: ${JSON.stringify(on)}`).toBe(200);
  const browser = (await login(api, mem.email)).access_token;
  const { archive, unarchive } = cycleCalls(api);

  // Q1 and Q2 ended in 2025; Now runs from 2026 to the end of 2098; Later has no days. Each is created before the one
  // before it.
  const q1 = await createCycle(api, admin.token, web.id, {
    name: "Q1",
    start_date: "2025-01-01",
    end_date: "2025-03-31",
  });
  const q2 = await createCycle(api, admin.token, web.id, {
    name: "Q2",
    start_date: "2025-04-01",
    end_date: "2025-06-30",
  });
  const now = await createCycle(api, admin.token, web.id, {
    name: "Now",
    start_date: "2026-01-01",
    end_date: "2098-12-31",
  });
  const later = await createCycle(api, admin.token, web.id, { name: "Later" });
  expect([q1.status, q2.status, now.status, later.status], "the cycles' statuses").toEqual([
    "completed",
    "completed",
    "current",
    "draft",
  ]);
  let cycles: CycleRow[] = [
    {
      name: "Q1",
      description: "",
      start: "2025-01-01",
      end: "2025-03-31",
      sort_order: 65535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
    {
      name: "Q2",
      description: "",
      start: "2025-04-01",
      end: "2025-06-30",
      sort_order: 55535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
    {
      name: "Now",
      description: "",
      start: "2026-01-01",
      end: "2098-12-31",
      sort_order: 45535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
    {
      name: "Later",
      description: "",
      start: null,
      end: null,
      sort_order: 35535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
  ];

  // Refused, each changing nothing: Now, current, and Later, a draft, are not completed; the guest may not archive.
  expect(
    [await archive(browser, now.id), await archive(admin.token, later.id), await archive(guest.token, q1.id)],
    "Now, Later, and the guest's archive of Q1"
  ).toEqual([
    { status: 409, code: "planning.cycle_not_completed" },
    { status: 409, code: "planning.cycle_not_completed" },
    { status: 403, code: "forbidden" },
  ]);
  await expectCycles(db, web.id, cycles);

  // mem archives Q1, archived and last changed by him at one moment; archiving it again answers it as it is and
  // writes nothing (R3); then he archives Q2.
  expect(await archive(browser, q1.id), "mem archives Q1").toEqual({ status: 200 });
  cycles = changed(cycles, "Q1", { archived: true, by: mem.email });
  await expectCycles(db, web.id, cycles);
  const archivedQ1 = await cycleMoments(db, web.id);
  expect(await archive(admin.token, q1.id), "the admin archives Q1 again").toEqual({ status: 200 });
  expect(await cycleMoments(db, web.id), "the cycles' last changes after Q1's second archive").toEqual(archivedQ1);
  expect(await archive(browser, q2.id), "mem archives Q2").toEqual({ status: 200 });
  cycles = changed(cycles, "Q2", { archived: true, by: mem.email });
  await expectCycles(db, web.id, cycles);

  // Web's list holds Later and Now; its archived catalog Q2, archived last, then Q1, the same with either of mem's
  // tokens; Q1 is read by its id, archived at its last change.
  const lists = async (token: string) => [
    (await listCycles(api, token, web.id)).map((c) => c.name),
    (await listCycles(api, token, web.id, true)).map((c) => c.name),
  ];
  expect(await lists(browser), "Web's list and archived catalog, as mem lists them").toEqual([
    ["Later", "Now"],
    ["Q2", "Q1"],
  ]);
  expect(await lists(mem.token), "the same with mem's personal access token").toEqual(await lists(browser));
  const read = await readCycle(api, browser, q1.id);
  expect(
    [read.status, read.archived_at !== null && read.archived_at === read.updated_at],
    "Q1 as mem reads it"
  ).toEqual(["completed", true]);

  // The admin restores Q1, last changed by him; restoring it again writes nothing. It is listed in its place, Q2
  // alone archived.
  expect(await unarchive(admin.token, q1.id), "the admin restores Q1").toEqual({ status: 200 });
  cycles = changed(cycles, "Q1", { archived: false, by: admin.email });
  await expectCycles(db, web.id, cycles);
  const restoredQ1 = await cycleMoments(db, web.id);
  expect(await unarchive(browser, q1.id), "mem restores Q1 again").toEqual({ status: 200 });
  expect(await cycleMoments(db, web.id), "the cycles' last changes after Q1's second restore").toEqual(restoredQ1);
  expect(await lists(browser), "Web's list and archived catalog, Q1 restored").toEqual([
    ["Later", "Now", "Q1"],
    ["Q2"],
  ]);
});
