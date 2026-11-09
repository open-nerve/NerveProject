import type {
  CyclePreferences,
  CyclePreferencesUpdate,
  IssueListDisplayFilters,
  ModulePreferencesUpdate,
} from "@nerve/api-client";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { pageSettingsRows, type PageSettingsRow } from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls, cyclePreferencesChanged, cyclePreferencesRead } from "../../fixtures/cycles";
import { validationFailed } from "../../fixtures/issue-planning";
import { createModule, moduleCalls, modulePreferencesChanged, modulePreferencesRead } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G15, each account's display settings on a cycle's and a module's page (M6 design G15, M6 P3b spec 2.3, 2.4, 2.5,
// R6, R8, A5), through the API and in the database; the pages come with P5 and P6. Each caller reads and changes his own
// row, the defaults while he has none; the page's own filter key is refused by the body's shape, a word no list has at
// its field; a deleted cycle's or module's rows are deleted with it. The admin calls with a personal access token, the
// member with the token of his sign-in and his personal access token, alike.

/** A change of the display settings on a cycle's or a module's page: either page's body, the page's own key too. */
type PageSettingsChange = CyclePreferencesUpdate & ModulePreferencesUpdate;

/** Filters of a change of the display settings on either page. */
type PageFilters = NonNullable<PageSettingsChange["filters"]>;

/** The page of a cycle or a module, as G15 reads and changes the display settings on it. */
interface SettingsPage {
  /** The name of its cycle or module. */
  name: string;
  /** The table of its rows, and that of its cycle or module. */
  table: Parameters<typeof pageSettingsRows>[1];
  targets: "cycles" | "modules";
  id: string;
  /** The answers of the reads and the changes of the settings on it by the caller of token. */
  read: (token: string) => Promise<unknown>;
  change: (token: string, body: PageSettingsChange) => Promise<unknown>;
  /** Its own filter key, which its body refuses, and the field the refusal names. */
  own: PageFilters;
  ownField: string;
  /** The other page's filter key, one of its filters. */
  other: PageFilters;
  /** The admin's deletion of its cycle or module, and the code a page of a deleted one is answered. */
  remove: () => Promise<unknown>;
  notFound: string;
}

/** The display settings on a page of a caller who has stored none (M6 P3b spec 2.1, 2.4). */
const defaults = {
  filters: {},
  display_filters: {
    layout: "list",
    group_by: null,
    sub_group_by: null,
    order_by: "-created_at",
    sub_issue: true,
    show_empty_groups: true,
    calendar: { layout: "month", show_weekends: false },
  },
  display_properties: {
    assignee: true,
    start_date: true,
    due_date: true,
    labels: true,
    key: true,
    priority: true,
    state: true,
    sub_issue_count: true,
    link: true,
    attachment_count: true,
    created_on: true,
    updated_on: true,
    cycle: true,
    modules: true,
  },
} satisfies CyclePreferences;

/** The answer of a read or a change of the display settings on a page that are preferences. */
function answered(preferences: unknown) {
  return { status: 200, preferences };
}

test("G15 (API): mem's display settings on Sprint's and Launch's pages are the defaults, alike to either of his tokens, a read writing no row; his first change stores the defaults and it, a second replaces only the field it gives; the page's own filter key is refused by the body's shape and words no list has at their fields, his row as it was; Web's guest has his own row beside mem's; the admin deletes Sprint and Launch, and their rows are deleted with them, at that moment, by him", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const mem = await newAccount(api, testInfo, "mem");
  const guest = await newAccount(api, testInfo, "guest");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, guest, 5);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  const on = await changeProject(api, admin.token, web.id, { cycle_view: true, module_view: true });
  expect(on.status, `the admin turns Web's cycles and modules on: ${JSON.stringify(on)}`).toBe(200);
  const browser = (await login(api, mem.email)).access_token;
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint" });
  const launch = await createModule(api, admin.token, web.id, { name: "Launch" });

  const pages: SettingsPage[] = [
    {
      name: "Sprint",
      table: "cycle_user_properties",
      targets: "cycles",
      id: sprint.id,
      read: (token) => cyclePreferencesRead(api, token, sprint.id),
      change: (token, body) => cyclePreferencesChanged(api, token, sprint.id, body),
      own: { cycle_id: ["none"] },
      ownField: "filters.cycle_id",
      other: { module_id: [launch.id] },
      remove: () => cycleCalls(api).remove(admin.token, sprint.id),
      notFound: "planning.cycle_not_found",
    },
    {
      name: "Launch",
      table: "module_user_properties",
      targets: "modules",
      id: launch.id,
      read: (token) => modulePreferencesRead(api, token, launch.id),
      change: (token, body) => modulePreferencesChanged(api, token, launch.id, body),
      own: { module_id: [launch.id] },
      ownField: "filters.module_id",
      other: { cycle_id: ["none"] },
      remove: () => moduleCalls(api).remove(admin.token, launch.id),
      notFound: "planning.module_not_found",
    },
  ];
  const kept = new Map<string, PageSettingsRow[]>();
  await pages.reduce(async (before, page) => {
    await before;
    const rows = () => pageSettingsRows(db, page.table, page.id);

    // The defaults, to either of mem's tokens; reading them writes no row.
    expect(
      [await page.read(browser), await page.read(mem.token)],
      `${page.name}'s settings as mem reads them with the token of his sign-in and his personal access token`
    ).toEqual([answered(defaults), answered(defaults)]);
    expect(await rows(), `${page.name}'s rows after mem's reads`).toEqual([]);

    // mem's first change, his filters: stored with the defaults, his row made and written at that moment, by him.
    // A filter of each kind, an empty one kept, and the other page's key.
    const filters = { priority: ["high"], label_id: [], assignee_id: ["me"], ...page.other };
    const first = { ...defaults, filters };
    expect(await page.change(browser, { filters }), `mem's first change on ${page.name}'s page`).toEqual(
      answered(first)
    );
    const [made] = await rows();
    if (!made) throw new Error(`${page.name}: mem's first change wrote no row`);
    expect(await rows(), `${page.name}'s rows after mem's first change`).toEqual([
      { user: mem.email, ...first, by: mem.email, created: made.created, updated: made.created, deleted: null },
    ]);

    // His second change, his display properties: the filters stay; his row written again, at a later moment.
    const properties = { ...defaults.display_properties, cycle: false, labels: false };
    const second = { ...first, display_properties: properties };
    expect(
      await page.change(mem.token, { display_properties: properties }),
      `mem's second change on ${page.name}'s page`
    ).toEqual(answered(second));
    expect(
      [await page.read(mem.token), await page.read(browser)],
      `${page.name}'s settings as mem reads them, changed`
    ).toEqual([answered(second), answered(second)]);
    const [changed] = await rows();
    expect(await rows(), `${page.name}'s rows after mem's second change`).toEqual([
      { user: mem.email, ...second, by: mem.email, created: made.created, updated: changed?.updated, deleted: null },
    ]);
    expect(changed?.updated, `the moment of mem's second change on ${page.name}'s page`).not.toBe(made.created);

    // Refused, his row as it was: the page's own key, by the body's shape, with a filter it allows; words no list has,
    // each at its field.
    const display: IssueListDisplayFilters = { ...defaults.display_filters, group_by: "project_id" };
    expect(
      [
        await page.change(browser, { filters: { ...page.own, priority: ["low"] } }),
        await page.change(mem.token, { filters: { label_id: ["wrong"] }, display_filters: display }),
      ],
      `mem's changes on ${page.name}'s page by its own key, and by wrong words`
    ).toEqual([
      { status: 400, code: "bad_request", errors: [{ field: page.ownField, code: "not_allowed" }] },
      validationFailed(["filters.label_id[0]", "not_allowed"], ["display_filters.group_by", "not_allowed"]),
    ]);
    const mems = await rows();
    expect(mems, `${page.name}'s rows after the refusals`).toEqual([
      { user: mem.email, ...second, by: mem.email, created: made.created, updated: changed?.updated, deleted: null },
    ]);

    // Web's guest, who sees only his own work items there: the defaults, then his own row beside mem's.
    const kanban = { ...defaults.display_filters, layout: "kanban", group_by: "state_id" };
    expect(await page.read(guest.token), `${page.name}'s settings as the guest reads them`).toEqual(answered(defaults));
    expect(
      await page.change(guest.token, { display_filters: kanban }),
      `the guest's first change on ${page.name}'s page`
    ).toEqual(answered({ ...defaults, display_filters: kanban }));
    const [guests] = await rows();
    if (!guests) throw new Error(`${page.name}: the guest's change wrote no row`);
    expect(await rows(), `${page.name}'s rows after the guest's change`).toEqual([
      {
        user: guest.email,
        ...defaults,
        display_filters: kanban,
        by: guest.email,
        created: guests.created,
        updated: guests.created,
        deleted: null,
      },
      ...mems,
    ]);
    expect(await page.read(browser), `${page.name}'s settings as mem reads them after the guest's`).toEqual(
      answered(second)
    );
    kept.set(page.name, await rows());
  }, Promise.resolve());

  // The admin deletes Sprint and Launch: each one's rows deleted with it, at its deletion's moment, by him; the page
  // of a deleted one is not found.
  await pages.reduce(async (before, page) => {
    await before;
    expect(await page.remove(), `the admin deletes ${page.name}`).toEqual({ status: 204 });
    const [deletion] = await db.query<{ at: string | null }>(
      `SELECT deleted_at::text AS at FROM ${page.targets} WHERE id = $1`,
      [page.id]
    );
    expect(deletion?.at, `${page.name}'s deletion's moment`).toEqual(expect.any(String));
    expect(await pageSettingsRows(db, page.table, page.id), `${page.name}'s rows after its deletion`).toEqual(
      (kept.get(page.name) ?? []).map(({ user, filters, display_filters, display_properties, created }) => ({
        user,
        filters,
        display_filters,
        display_properties,
        by: admin.email,
        created,
        updated: deletion?.at,
        deleted: deletion?.at,
      }))
    );
    expect(
      [await page.read(browser), await page.change(mem.token, { filters: {} })],
      `${page.name}'s settings read and changed by mem once it is deleted`
    ).toEqual([
      { status: 404, code: page.notFound },
      { status: 404, code: page.notFound },
    ]);
  }, Promise.resolve());
});
