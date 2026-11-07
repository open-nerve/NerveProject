import { changeProject, createProject, createWorkspace, slugFor, type Api } from "../../fixtures/api";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, listCycles, readCycle } from "../../fixtures/cycles";
import { expect, test } from "../../fixtures/test";

// G2, a cycle's status (M6 design 3.2): read from its days on its project's calendar day, never stored. Kiritimati
// (UTC+14) and Pago Pago (UTC-11) are 25 hours apart, so Kiritimati's day is always one or two after Pago Pago's,
// whatever the hour the story runs at.

const kiritimati = "Pacific/Kiritimati";
const pagoPago = "Pacific/Pago_Pago";

/** The calendar day, YYYY-MM-DD, that it is now in the time zone zone. */
function today(zone: string): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: zone, year: "numeric", month: "2-digit", day: "2-digit" }).format(
    new Date()
  );
}

/** The status of a cycle of one day, day, on the calendar day on. */
function statusOn(day: string, on: string): string {
  return on < day ? "upcoming" : on === day ? "current" : "completed";
}

/**
 * The names and statuses of the cycles of each project of projectIds, as the caller of each token lists them, then
 * reads each, with the calendar day of zone they were read on; read again when that day ended while they were read.
 */
async function statuses(
  api: Api,
  tokens: string[],
  projectIds: string[],
  zone: string
): Promise<{ day: string; answers: [string, string][][][] }> {
  const day = today(zone);
  const answers = await Promise.all(
    projectIds.map((projectId) =>
      Promise.all(
        tokens.map(async (token) => {
          const listed = await listCycles(api, token, projectId);
          const read = await Promise.all(listed.map((c) => readCycle(api, token, c.id)));
          return [listed, read].map((cycles) => cycles.map((c): [string, string] => [c.name, c.status]));
        })
      ).then((each) => each.flat())
    )
  );
  return today(zone) === day ? { day, answers } : statuses(api, tokens, projectIds, zone);
}

test("G2 (API): Atoll keeps Kiritimati's time and Samoa Pago Pago's; each has a Backlog without days, a cycle of Kiritimati's day and one of Pago Pago's; Atoll's are a draft, current and completed, Samoa's a draft, upcoming and current, the same in their lists and read one by one, with a personal access token and with the token of a sign-in; Samoa moved to Kiritimati's time reads its own as Atoll's", async ({
  api,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const atoll = await createProject(api, admin.token, slug, { name: "Atoll", identifier: "ATL", timezone: kiritimati });
  const samoa = await createProject(api, admin.token, slug, { name: "Samoa", identifier: "SAM", timezone: pagoPago });
  const browser = (await login(api, admin.email)).access_token;
  const kiritimatiDay = today(kiritimati);
  const pagoPagoDay = today(pagoPago);
  expect(kiritimatiDay > pagoPagoDay, `Kiritimati's ${kiritimatiDay} after Pago Pago's ${pagoPagoDay}`).toBe(true);
  await Promise.all(
    [atoll, samoa].map(async (project) => {
      const on = await changeProject(api, admin.token, project.id, { cycle_view: true });
      expect(on.status, `turn ${project.name}'s cycles on: ${JSON.stringify(on)}`).toBe(200);
      // Created in this order, each before the one before it: listed Pago Pago's, Kiritimati's, Backlog.
      await createCycle(api, admin.token, project.id, { name: "Backlog" });
      await createCycle(api, admin.token, project.id, {
        name: "Kiritimati's day",
        start_date: kiritimatiDay,
        end_date: kiritimatiDay,
      });
      await createCycle(api, admin.token, project.id, {
        name: "Pago Pago's day",
        start_date: pagoPagoDay,
        end_date: pagoPagoDay,
      });
    })
  );

  // Each project reads the days on its own calendar day; the list and each read agree, through either token. On the
  // days they were created Atoll's are completed, current and a draft, Samoa's current, upcoming and a draft; the
  // story reads them on the days it reads them on, a day possibly ended since.
  const [atollRead, samoaRead] = await Promise.all([
    statuses(api, [admin.token, browser], [atoll.id], kiritimati),
    statuses(api, [admin.token, browser], [samoa.id], pagoPago),
  ]);
  const want = (day: string): [string, string][] => [
    ["Pago Pago's day", statusOn(pagoPagoDay, day)],
    ["Kiritimati's day", statusOn(kiritimatiDay, day)],
    ["Backlog", "draft"],
  ];
  expect(
    [atollRead.answers, samoaRead.answers],
    `Atoll's cycles as listed and read on ${atollRead.day}, Samoa's on ${samoaRead.day}`
  ).toEqual([[Array(4).fill(want(atollRead.day))], [Array(4).fill(want(samoaRead.day))]]);
  // The admin moves Samoa to Kiritimati's time: its cycles, their days unchanged, read as Atoll's.
  const moved = await changeProject(api, admin.token, samoa.id, { timezone: kiritimati });
  expect(moved.status, `move Samoa to Kiritimati's time: ${JSON.stringify(moved)}`).toBe(200);
  const [atollNow, samoaNow] = (await statuses(api, [admin.token, browser], [atoll.id, samoa.id], kiritimati)).answers;
  expect(samoaNow, "Samoa's cycles on Kiritimati's time").toEqual(atollNow);
});
