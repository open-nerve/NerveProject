import type { components } from "@nerve/api-client";
import { expect, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  answer,
  changeProject,
  changeRole,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "./api";
import { bearer, login, newAccount } from "./auth";
import type { Cycle } from "./cycles";
import { stateIds } from "./states";

// The calls and the expected numbers of the stories of a cycle's and a module's progress, its freeze and a cycle's
// transfer (M6 design G5–G7, G12, M6 P4 spec 2.1, 2.3, 2.7): the project's calendar day the stories count on, the
// three operations, and the progress a story expects, field by field as the API answers it.

export type Progress = components["schemas"]["Progress"];
export type CycleProgress = components["schemas"]["CycleProgress"];
type FrozenCounts = components["schemas"]["FrozenCounts"];
type AssigneeDistribution = components["schemas"]["AssigneeDistribution"];
type LabelDistribution = components["schemas"]["LabelDistribution"];
type Burndown = components["schemas"]["Burndown"];

/** The six counts of a progress: the work items, and those of each state group. */
export type Counts = Omit<FrozenCounts, "frozen_at">;

/** A state group a work item counts in. */
type Group = "backlog" | "unstarted" | "started" | "completed" | "cancelled";

/**
 * The time zone, of the Etc/GMT zones of whole hours, whose clock is nearest noon at now, within half an hour: a story
 * that counts on its project's calendar day, today's completions on it, runs some eleven hours from its midnight,
 * whenever it runs (the server runs on the wall clock). The POSIX names invert the sign: Etc/GMT-8 is eight hours
 * ahead of UTC. Noon is never more than twelve hours from UTC's hour, so the offsets from -12 to +12 suffice.
 */
function noonZone(now = new Date()): string {
  const ahead = Math.round(12 - (now.getUTCHours() + now.getUTCMinutes() / 60));
  return ahead === 0 ? "Etc/GMT" : ahead > 0 ? `Etc/GMT-${ahead}` : `Etc/GMT+${-ahead}`;
}

/** The calendar day, YYYY-MM-DD, days after day (before it when negative). */
function addDays(day: string, days: number): string {
  const at = new Date(`${day}T00:00:00Z`);
  at.setUTCDate(at.getUTCDate() + days);
  return at.toISOString().slice(0, 10);
}

/** The calendar day, YYYY-MM-DD, that it is now in the time zone zone. */
function todayIn(zone: string): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: zone, year: "numeric", month: "2-digit", day: "2-digit" }).format(
    new Date()
  );
}

/** The counts of work items in each group of groups, none in the others; the total their sum. */
export function counts(groups: Partial<Record<Group, number>>): Counts {
  const { backlog = 0, unstarted = 0, started = 0, completed = 0, cancelled = 0 } = groups;
  return {
    total_issues: backlog + unstarted + started + completed + cancelled,
    backlog_issues: backlog,
    unstarted_issues: unstarted,
    started_issues: started,
    completed_issues: completed,
    cancelled_issues: cancelled,
  };
}

/** The six counts of a progress, or of a list's row. */
export function countsOf(p: Counts): Counts {
  return {
    total_issues: p.total_issues,
    backlog_issues: p.backlog_issues,
    unstarted_issues: p.unstarted_issues,
    started_issues: p.started_issues,
    completed_issues: p.completed_issues,
    cancelled_issues: p.cancelled_issues,
  };
}

/** A row of a distribution as a story writes it: its assignee's or label's id, null for none, completed and pending. */
type Tally = [id: string | null, completed: number, pending: number];

/** rows by id ascending, the row of none last: the order a distribution answers in. */
function inIdOrder(rows: Tally[]): Tally[] {
  return rows.toSorted(([a], [b]) => (a === b ? 0 : a === null ? 1 : b === null ? -1 : a < b ? -1 : 1));
}

/** The distribution by assignee of rows, in its order. */
export function byAssignee(...rows: Tally[]): AssigneeDistribution[] {
  return inIdOrder(rows).map(([assignee_id, completed_issues, pending_issues]) => ({
    assignee_id,
    total_issues: completed_issues + pending_issues,
    completed_issues,
    pending_issues,
  }));
}

/** The distribution by label of rows, in its order. */
export function byLabel(...rows: Tally[]): LabelDistribution[] {
  return inIdOrder(rows).map(([label_id, completed_issues, pending_issues]) => ({
    label_id,
    total_issues: completed_issues + pending_issues,
    completed_issues,
    pending_issues,
  }));
}

/** The burndown from start to end of scope: a point each day, the first with left's first number, null once they end. */
export function burndown(start: string, end: string, scope: number, ...left: number[]): Burndown {
  const points: Burndown["points"] = [];
  for (let day = start; day <= end; day = addDays(day, 1)) {
    points.push({ date: day, remaining: left[points.length] ?? null });
  }
  return { start_date: start, end_date: end, scope, points };
}

/** The ids of rows ascending: a cycle's list row's assignee_ids. */
export function ascending(...ids: string[]): string[] {
  return ids.toSorted();
}

/** The call of getCycleProgress of the cycle of id by the caller of token. */
function getCycleProgress(api: Api, token: string, id: string) {
  return api.GET("/api/v0/cycles/{cycle_id}/progress", { params: { path: { cycle_id: id } }, headers: bearer(token) });
}

/** The progress of the cycle of id as the caller of token reads it (getCycleProgress). */
export async function cycleProgress(api: Api, token: string, id: string): Promise<CycleProgress> {
  const { data, error, response } = await getCycleProgress(api, token, id);
  expect(response.status, `read the progress of the cycle ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`cycleProgress ${id} answered 200 without the progress`);
  }
  return data;
}

/** The answer of getCycleProgress of the cycle of id by the caller of token: its status, or the problem's code. */
export async function cycleProgressAnswer(api: Api, token: string, id: string) {
  const { error, response } = await getCycleProgress(api, token, id);
  return answer(response, error);
}

/** The progress of the module of id as the caller of token reads it (getModuleProgress). */
export async function moduleProgress(api: Api, token: string, id: string): Promise<Progress> {
  const { data, error, response } = await api.GET("/api/v0/modules/{module_id}/progress", {
    params: { path: { module_id: id } },
    headers: bearer(token),
  });
  expect(response.status, `read the progress of the module ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`moduleProgress ${id} answered 200 without the progress`);
  }
  return data;
}

/**
 * The answer of a transfer of the work items of the cycle of from to the cycle of to by the caller of token
 * (transferCycleIssues): its status and how many moved, or the problem's code and fields.
 */
export async function transferred(api: Api, token: string, from: string, to: string) {
  const { data, error, response } = await api.POST("/api/v0/cycles/{cycle_id}/transfer-issues", {
    params: { path: { cycle_id: from } },
    body: { target_cycle_id: to },
    headers: bearer(token),
  });
  return data ? { status: response.status, transferred_issues: data.transferred_issues } : answer(response, error);
}

/** A transfer's answer of moved work items (transferred). */
export function moved(n: number) {
  return { status: 200, transferred_issues: n };
}

/**
 * Ends the cycle of id, current or ahead, by the caller of token: its days moved to start and end, in the past (M6 P4
 * spec 2.7; only a cycle that has ended refuses a change of its days). Answers the cycle as the change answers it.
 */
export async function endCycle(api: Api, token: string, id: string, start: string, end: string): Promise<Cycle> {
  const { data, error, response } = await api.PATCH("/api/v0/cycles/{cycle_id}", {
    params: { path: { cycle_id: id } },
    body: { start_date: start, end_date: end },
    headers: bearer(token),
  });
  expect(response.status, `end the cycle ${id} on ${end}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`endCycle ${id} answered 200 without the cycle`);
  }
  expect(data.status, `the cycle ${id} ended on ${end}`).toBe("completed");
  return data;
}

/**
 * Acme's world of the progress stories: the admin's workspace Acme; mem, its member and Web's, who calls with the
 * token of his sign-in (browser) too; walt, its admin and only Web's guest, who sees only his own work items there
 * (Web's guests do not see every work item by default; M4 design 3.4); Web, on the clock of noonZone at the start, its
 * cycles and modules turned on, its labels Bug and UI, and its five states, one of each group. day(n) is the calendar
 * day n days after Web's today then.
 */
export async function progressWorld(api: Api, testInfo: TestInfo) {
  const zone = noonZone();
  const today = todayIn(zone);
  const [admin, mem, walt] = [
    await newAccount(api, testInfo, "admin"),
    await newAccount(api, testInfo, "mem"),
    await newAccount(api, testInfo, "walt"),
  ];
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, walt, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB", timezone: zone });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: walt.id, role: 5 },
  ]);
  await changeRole(api, admin.token, slug, walt.id, 20);
  const on = await changeProject(api, admin.token, web.id, { cycle_view: true, module_view: true });
  expect(on.status, `the admin turns Web's cycles and modules on: ${JSON.stringify(on)}`).toBe(200);
  const ids = await stateIds(api, admin.token, web.id);
  const state = (name: string) => {
    const id = ids[name];
    if (!id) {
      throw new Error(`Web's states hold no ${name}`);
    }
    return id;
  };
  return {
    day: (n: number) => addDays(today, n),
    admin,
    mem,
    walt,
    slug,
    web,
    states: {
      backlog: state("Backlog"),
      todo: state("Todo"),
      started: state("In Progress"),
      done: state("Done"),
      cancelled: state("Cancelled"),
    },
    bug: await createLabel(api, admin.token, web.id, { name: "Bug" }),
    ui: await createLabel(api, admin.token, web.id, { name: "UI" }),
    browser: (await login(api, mem.email)).access_token,
  };
}
