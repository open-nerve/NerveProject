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

// G3, changing and deleting a project's cycles (M6 P1 spec 2.3, R4, R5): the name, the description, the days, the
// place; a completed cycle's place alone; nothing of an archived one; deleting for the project's admins and a
// cycle's creator. The admin calls with a personal access token, the member with the token of his sign-in.

/** rows with the cycle of name changed as to says. */
function changed(rows: CycleRow[], name: string, to: Partial<CycleRow>): CycleRow[] {
  return rows.map((c) => (c.name === name ? { ...c, ...to } : c));
}

test("G3 (API): Web's member renames his Alpha Sprint A, describes it and moves its days over its own; an end on Beta's first day, a start after its end and a guest's change are refused; an empty change and one to the same name write nothing; Done, completed, takes a place and refuses its name, even the same; archived, it refuses any change; the member may not delete Beta, the admin's, and deletes Sprint A, after which it is not found; the admin deletes Done, completed and archived, and Beta and Done are listed no more", async ({
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
  const { update, archive, remove, read } = cycleCalls(api);

  // Alpha, mem's, and Beta, the admin's, are in January and February 2099, ahead; Done, the admin's, ended in 2025.
  const alpha = await createCycle(api, browser, web.id, {
    name: "Alpha",
    start_date: "2099-01-01",
    end_date: "2099-01-14",
  });
  const beta = await createCycle(api, admin.token, web.id, {
    name: "Beta",
    start_date: "2099-02-01",
    end_date: "2099-02-14",
  });
  const done = await createCycle(api, admin.token, web.id, {
    name: "Done",
    start_date: "2025-01-01",
    end_date: "2025-01-14",
  });
  let cycles: CycleRow[] = [
    {
      name: "Alpha",
      description: "",
      start: "2099-01-01",
      end: "2099-01-14",
      sort_order: 65535,
      archived: false,
      deleted: false,
      owner: mem.email,
      by: mem.email,
    },
    {
      name: "Beta",
      description: "",
      start: "2099-02-01",
      end: "2099-02-14",
      sort_order: 55535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
    {
      name: "Done",
      description: "",
      start: "2025-01-01",
      end: "2025-01-14",
      sort_order: 45535,
      archived: false,
      deleted: false,
      owner: admin.email,
      by: admin.email,
    },
  ];
  await expectCycles(db, web.id, cycles);

  // mem renames Alpha Sprint A and describes it; then moves its days to the 5th to the 20th, over its own: a cycle's
  // own days are no overlap. Each answers the cycle as changed.
  expect(await update(browser, alpha.id, { name: "Sprint A", description: "Checkout" }), "mem renames Alpha").toEqual({
    status: 200,
  });
  expect(
    await update(browser, alpha.id, { start_date: "2099-01-05", end_date: "2099-01-20" }),
    "mem moves Sprint A's days"
  ).toEqual({ status: 200 });
  cycles = changed(cycles, "Alpha", {
    name: "Sprint A",
    description: "Checkout",
    start: "2099-01-05",
    end: "2099-01-20",
  });
  await expectCycles(db, web.id, cycles);
  const sprintA = await readCycle(api, browser, alpha.id);
  expect(
    [sprintA.name, sprintA.description, sprintA.start_date, sprintA.end_date, sprintA.status, sprintA.updated_by_id],
    "Sprint A as mem reads it"
  ).toEqual(["Sprint A", "Checkout", "2099-01-05", "2099-01-20", "upcoming", mem.id]);

  // Refused, each changing nothing: Sprint A's end moved to Beta's first day, the 1st of February; its start alone
  // moved after its end, the problem at the start; the guest's change of Beta's name.
  expect(
    [
      await update(browser, alpha.id, { end_date: "2099-02-01" }),
      await update(browser, alpha.id, { start_date: "2099-01-21" }),
      await update(guest.token, beta.id, { name: "Guest's" }),
    ],
    "Sprint A's end on Beta's first day, its start after its end, the guest's change"
  ).toEqual([
    { status: 409, code: "planning.cycle_dates_overlap" },
    { status: 422, code: "validation_failed", errors: [{ field: "start_date", code: "not_allowed" }] },
    { status: 403, code: "forbidden" },
  ]);
  await expectCycles(db, web.id, cycles);

  // An empty change and one to Beta's own name answer it as it is and write nothing: not its last change, nor who
  // made it.
  const before = await cycleMoments(db, web.id);
  expect(
    [await update(browser, beta.id, {}), await update(browser, beta.id, { name: "Beta" })],
    "mem's empty change of Beta, and his change to its name"
  ).toEqual([{ status: 200 }, { status: 200 }]);
  expect(await cycleMoments(db, web.id), "the cycles' last changes").toEqual(before);
  await expectCycles(db, web.id, cycles);

  // Done ended: mem moves it to the top; its name, another or its own, is refused (R5).
  expect(await update(browser, done.id, { sort_order: 1000 }), "mem moves Done").toEqual({ status: 200 });
  cycles = changed(cycles, "Done", { sort_order: 1000, by: mem.email });
  expect(
    [
      await update(browser, done.id, { name: "Finished" }),
      await update(admin.token, done.id, { name: "Done", sort_order: 2000 }),
    ],
    "Done renamed, and given its own name with a place"
  ).toEqual([
    { status: 409, code: "planning.cycle_completed" },
    { status: 409, code: "planning.cycle_completed" },
  ]);
  await expectCycles(db, web.id, cycles);
  // The admin archives Done: then nothing of it changes, an empty change refused too.
  expect(await archive(admin.token, done.id), "the admin archives Done").toEqual({ status: 200 });
  cycles = changed(cycles, "Done", { archived: true, by: admin.email });
  expect(
    [await update(browser, done.id, { sort_order: 3000 }), await update(browser, done.id, {})],
    "archived Done moved, and changed by nothing"
  ).toEqual([
    { status: 409, code: "planning.archived" },
    { status: 409, code: "planning.archived" },
  ]);
  await expectCycles(db, web.id, cycles);

  // mem may not delete Beta, the admin's, and deletes Sprint A, his own: deleted by him, read no more, a second
  // deletion and a change not found.
  expect(await remove(browser, beta.id), "mem deletes Beta").toEqual({ status: 403, code: "forbidden" });
  await expectCycles(db, web.id, cycles);
  expect(await remove(browser, alpha.id), "mem deletes Sprint A").toEqual({ status: 204 });
  cycles = changed(cycles, "Sprint A", { deleted: true });
  await expectCycles(db, web.id, cycles);
  expect(
    [
      await read(browser, alpha.id),
      await update(browser, alpha.id, { name: "Sprint A2" }),
      await remove(browser, alpha.id),
      await remove(admin.token, alpha.id),
    ],
    "deleted Sprint A read, changed, deleted again by mem and by the admin"
  ).toEqual(Array(4).fill({ status: 404, code: "planning.cycle_not_found" }));
  await expectCycles(db, web.id, cycles);
  // The admin deletes Done, completed and archived: a deletion asks neither.
  expect(await remove(admin.token, done.id), "the admin deletes Done").toEqual({ status: 204 });
  cycles = changed(cycles, "Done", { deleted: true });
  await expectCycles(db, web.id, cycles);
  expect(
    [
      (await listCycles(api, browser, web.id)).map((c) => c.name),
      (await listCycles(api, mem.token, web.id, true)).map((c) => c.name),
    ],
    "Web's cycles and its archived ones, as mem lists them with either token"
  ).toEqual([["Beta"], []]);
});
