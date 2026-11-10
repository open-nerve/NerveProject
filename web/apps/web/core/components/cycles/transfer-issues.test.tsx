// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { dialog, settle } from "./fake-cycle-pages";
import { TransferPages, cyclesOf, transferCycles, where } from "./fake-transfer-pages";

// The transfer of an ended cycle's unfinished work items (P5a spec 2.2, 2.5, R8, R12): its two entries, its row's
// "transfer N work items" and the banner of its page, each for a caller who writes the project's cycles, of an ended
// cycle not archived with work items unfinished by its live counts, frozen or not; and its dialog, its page's
// (CycleActionsHost), which offers the project's cycles that may take them and waits for nerve: it closes and says how
// many nerve moved only once nerve has (none is an answer too); refused, it stays and says why, never success. The
// caller's role is the permission store's own (fake-permissions.ts); its pages and cycles fake-transfer-pages.tsx's.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof transferCycles>,
  permissions: undefined as IUserPermissionStore | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("./modal", () => ({ CycleCreateUpdateModal: () => null }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  signedIn();
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

/**
 * The caller at a page of web's cycles: "cycles", or the page of the cycle of a name (cycles/<name>); the cycles are
 * web's (cyclesOf), or those of names alone. Gives them by name.
 */
async function at(page: string, caller: Caller = its.admin, names?: string[]) {
  const { permissions, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole);
  held.permissions = permissions;
  const rows = cyclesOf(web).filter((row) => names?.includes(row.name) ?? true);
  held.cycles = transferCycles(rows);
  const path = page.replace(/<(\w+)>/, (_, name: string) => rows.find((row) => row.name === name)?.id ?? name);
  await act(async () => root.render(<TransferPages path={`/${acme.slug}/projects/${web.id}/${path}`} />));
  await settle();
  return Object.fromEntries(rows.map((row) => [row.name, row.id]));
}

/** Each row's transfer, as "name: what it says", of the rows that offer one. */
const entries = () =>
  [...host.querySelectorAll<HTMLElement>("[data-row]")].flatMap((row) => {
    const button = row.querySelector("button");
    return button ? [`${row.dataset.row}: ${button.textContent}`] : [];
  });
const bannerButton = () =>
  [...host.querySelectorAll("button")].find((button) => button.textContent === "project_cycles.transfer.title");
const click = async (button: HTMLElement | null | undefined) => {
  await act(async () => button?.click());
  await settle();
};
const openFromRow = (name: string) => click(host.querySelector<HTMLElement>(`[data-row="${name}"] button`));
const title = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;
const closeButton = () => dialog()?.querySelector<HTMLButtonElement>('button[aria-label="close"]');
const targetButtons = () => [...(dialog()?.querySelectorAll<HTMLButtonElement>("li button") ?? [])];
/** The cycles the dialog offers, each as its button shows it: its name, then its status. */
const targets = () =>
  targetButtons().map((button) => [...button.querySelectorAll("span")].map((span) => span.textContent).join(" "));
const chooseTarget = (name: string) =>
  click(targetButtons().find((button) => button.querySelector("span")?.textContent === name));
async function search(query: string) {
  const input = dialog()?.querySelector<HTMLInputElement>('input[aria-label="project_cycles.transfer.search"]');
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, query);
    input?.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const escape = async () => {
  await act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await settle();
};
const said = (count: number, target: string) => ({
  type: "success",
  title: "project_cycles.action.transfer.success.title",
  message: `project_cycles.action.transfer.success.description ${count} ${target}`,
});

it.each([
  { caller: its.admin, offered: true },
  { caller: its.member, offered: true },
  { caller: its.memberAdmin, offered: true },
  { caller: its.guest, offered: false },
  { caller: its.guestAdmin, offered: true },
  { caller: its.outsider, offered: false },
])(
  "offers $caller.who a row's transfer of an ended cycle's unfinished work items, by its live counts: $offered",
  async ({ caller, offered }) => {
    await at("cycles", caller);
    // frozen or not; not of a cycle archived, not ended, or with none unfinished now, though frozen with some
    const ended = ["Sprint: project_cycles.transfer.entry 4", "Frozen: project_cycles.transfer.entry 2"];
    expect(entries()).toEqual(offered ? ended : []);
  }
);

it.each([
  { cycle: "Sprint", caller: its.admin, offered: true },
  { cycle: "Frozen", caller: its.member, offered: true },
  { cycle: "Sprint", caller: its.guest, offered: false },
  { cycle: "Sprint", caller: its.outsider, offered: false },
  { cycle: "Done", caller: its.admin, offered: false },
  { cycle: "Shelved", caller: its.admin, offered: false },
  { cycle: "Now", caller: its.admin, offered: false },
])("shows $caller.who the transfer's banner on the page of $cycle: $offered", async ({ cycle, caller, offered }) => {
  await at(`cycles/<${cycle}>`, caller);
  expect(host.textContent?.includes("project_cycles.transfer.completed")).toBe(offered);
  expect(bannerButton() !== undefined).toBe(offered);
});

it("offers the project's cycles not archived nor ended, in nerve's order, never itself; by name", async () => {
  const ids = await at("cycles");
  await openFromRow("Sprint");
  expect(title()).toBe("project_cycles.transfer.title");
  const statuses = ["Now project_cycles.status.in_progress", "Next project_cycles.status.yet_to_start"];
  expect(targets()).toEqual([...statuses, "Idea project_cycles.status.draft"]);
  expect([...(dialog()?.querySelectorAll<HTMLElement>("[tabindex]") ?? [])].filter((el) => el.tabIndex > 0)).toEqual(
    []
  );
  // nerve read it again: its days have changed, so that it has not ended; still not its own target
  await held.cycles.reread(ids.Sprint, { status: "upcoming" });
  expect(targets()).toEqual([...statuses, "Idea project_cycles.status.draft"]);
  await search("NE");
  expect(targets()).toEqual(["Next project_cycles.status.yet_to_start"]);
  await search("zzz");
  expect(targets()).toEqual([]);
  expect(dialog()?.textContent).toContain("project_cycles.no_matching_cycles");
  expect(dialog()?.textContent).not.toContain("project_cycles.transfer.no_target");
});

it("says that no cycle can take them, to make one first", async () => {
  await at("cycles", its.admin, ["Sprint", "Done", "Later"]);
  await openFromRow("Sprint");
  expect(targets()).toEqual([]);
  expect(dialog()?.textContent).toContain("project_cycles.transfer.no_target");
});

it("waits for nerve, nothing closing it meanwhile; once nerve has moved them, closes and says how many", async () => {
  const ids = await at("cycles");
  await openFromRow("Sprint");
  await chooseTarget("Next");
  expect(held.cycles.transfer.mock.calls).toEqual([[ids.Sprint, ids.Next]]);
  expect([dialog() === null, toasts]).toEqual([false, []]);
  await escape();
  expect(dialog()).not.toBeNull();
  expect(closeButton()?.disabled).toBe(true);
  expect(targetButtons().map((button) => button.disabled)).toEqual([true, true, true]);
  await chooseTarget("Now");
  expect(held.cycles.transfer).toHaveBeenCalledOnce();
  await held.cycles.answer(4);
  expect(toasts).toEqual([said(4, "Next")]);
  expect(dialog()).toBeNull();
});

it("says so when nerve moved none: an answer too", async () => {
  await at("cycles");
  await openFromRow("Frozen");
  await chooseTarget("Idea");
  await held.cycles.answer(0);
  expect(toasts).toEqual([said(0, "Idea")]);
  expect(dialog()).toBeNull();
});

it.each([
  {
    refused: "422 of its target",
    error: refusal(422, "validation_failed", [{ field: "target_cycle_id", code: "not_allowed" }]),
    why: "errors.validation_failed",
  },
  { refused: "409 archived", error: refusal(409, "planning.archived"), why: "errors.planning_archived" },
  {
    refused: "409 target ended",
    error: refusal(409, "planning.cycle_completed"),
    why: "errors.planning_cycle_completed",
  },
  {
    refused: "409 not ended",
    error: refusal(409, "planning.cycle_not_completed"),
    why: "errors.planning_cycle_not_completed",
  },
  {
    refused: "409 cycles off",
    error: refusal(409, "planning.cycles_disabled"),
    why: "errors.planning_cycles_disabled",
  },
  { refused: "409 project archived", error: refusal(409, "project.archived"), why: "errors.project_archived" },
  { refused: "403", error: refusal(403, "forbidden"), why: "errors.forbidden" },
  { refused: "503", error: refusal(503, "server_busy"), why: "errors.server_busy" },
])("keeps the dialog when nerve refuses the transfer ($refused), saying why, never success", async (refused) => {
  const ids = await at("cycles");
  await openFromRow("Sprint");
  await chooseTarget("Now");
  await held.cycles.refuse(refused.error);
  const why = { type: "error", title: "toast.error", message: refused.why };
  expect(toasts).toEqual([why]);
  expect(dialog()).not.toBeNull();
  expect(closeButton()?.disabled).toBe(false);
  // chosen again, made: it closes then, and only then says so
  await chooseTarget("Next");
  await held.cycles.answer(4);
  expect(held.cycles.transfer.mock.calls).toEqual([
    [ids.Sprint, ids.Now],
    [ids.Sprint, ids.Next],
  ]);
  expect(toasts).toEqual([why, said(4, "Next")]);
  expect(dialog()).toBeNull();
});

it("follows nerve's answer on its page, though the row's transfer went as its counts were read again", async () => {
  const ids = await at("cycles");
  await openFromRow("Sprint");
  await chooseTarget("Next");
  await held.cycles.reread(ids.Sprint, { backlog_issues: 0, unstarted_issues: 0, started_issues: 0 });
  expect(entries()).toEqual(["Frozen: project_cycles.transfer.entry 2"]);
  await held.cycles.answer(4);
  expect(toasts).toEqual([said(4, "Next")]);
  expect(dialog()).toBeNull();
});

it("transfers again what is unfinished after a transfer", async () => {
  const ids = await at("cycles");
  await openFromRow("Sprint");
  await chooseTarget("Next");
  await held.cycles.answer(4);
  // read again after it: one is back in a state not started
  await held.cycles.reread(ids.Sprint, { backlog_issues: 0, unstarted_issues: 1, started_issues: 0 });
  expect(entries()).toEqual(["Sprint: project_cycles.transfer.entry 1", "Frozen: project_cycles.transfer.entry 2"]);
  await openFromRow("Sprint");
  await chooseTarget("Now");
  await held.cycles.answer(1);
  expect(held.cycles.transfer.mock.calls).toEqual([
    [ids.Sprint, ids.Next],
    [ids.Sprint, ids.Now],
  ]);
  expect(toasts).toEqual([said(4, "Next"), said(1, "Now")]);
});

it("transfers from the banner of a frozen cycle's page, whose banner goes once none is left", async () => {
  const ids = await at("cycles/<Frozen>");
  expect(host.textContent).toContain("project_cycles.transfer.completed");
  await click(bannerButton());
  expect(title()).toBe("project_cycles.transfer.title");
  await chooseTarget("Idea");
  expect(held.cycles.transfer.mock.calls).toEqual([[ids.Frozen, ids.Idea]]);
  await held.cycles.answer(2);
  await held.cycles.reread(ids.Frozen, { started_issues: 0 });
  expect(toasts).toEqual([said(2, "Idea")]);
  expect(dialog()).toBeNull();
  expect(bannerButton()).toBeUndefined();
});

const settlings = [
  { settles: "answered", settle: () => held.cycles.answer(4) },
  { settles: "refused", settle: () => held.cycles.refuse(refusal(403, "forbidden")) },
];

it.each(settlings)("follows nothing of a transfer $settles once a Back has left its page", async ({ settle: by }) => {
  await at("cycles");
  await openFromRow("Sprint");
  await chooseTarget("Next");
  await act(async () => where.navigate?.("/acme/projects/elsewhere/issues"));
  await settle();
  await by();
  expect(toasts).toEqual([]);
  expect(host.textContent).toBe("elsewhere");
});

it.each(settlings)(
  "follows nothing of a transfer $settles after another tab moved this one to another account",
  async ({ settle: by }) => {
    await at("cycles");
    await openFromRow("Sprint");
    await chooseTarget("Next");
    switchAccount();
    await by();
    expect(toasts).toEqual([]);
  }
);
