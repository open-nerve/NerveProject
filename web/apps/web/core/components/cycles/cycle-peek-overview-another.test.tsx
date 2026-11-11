// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, CycleUpdate } from "@nerve/api-client";
import { renderFormattedPayloadDate } from "@nerve/utils";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { heldChange, signedIn, type HeldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { progressOf, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, projectOf } from "@/store/project/fake-projects";
import { CyclePeekOverview, usePeekCycle } from "./cycle-peek-overview";
import { listedCycles } from "./fake-cycle-lists";

// The preview of one cycle, then of another in the same panel (P5a spec 2.5; v0 design 7.7): the list previews
// another cycle by changing ?peekCycle, and the other's preview starts afresh, as a cycle's own page does (its layout
// is keyed by the cycle): nothing of the cycle before — a day picked alone, a change of its days still out, the tab of
// its progress — carries into it. The sidebar is the real one; the cycles a stand-in store's, whose change of a
// cycle's days goes to nerve, which settles it when the test says; the caller writes the project's cycles.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof previewedCycles>,
  /** The props the dates' picker was last given. */
  dates: undefined as ComponentProps<typeof DateRangeDropdown> | undefined,
  /** How the page previews another cycle (usePeekCycle's). */
  peek: undefined as ((cycleId: string | null) => void) | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ canWriteCycles: () => true }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ project: { getProjectMemberDetails: () => null } }),
}));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({ getProjectLabels: () => [] }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
// the dates' picker keeps its props, so that a test reads the days it shows and picks a range as its calendar would
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => {
    held.dates = props;
    return <button type="button" aria-label="dates" disabled={props.disabled} />;
  },
}));

const web = projectOf("WEB", "id-acme");
const draft = cycleOf(web, "Draft");
const alpha = cycleOf(web, "Alpha", { status: "upcoming", start_date: "2026-11-16", end_date: "2026-11-22" });
const beta = cycleOf(web, "Beta", { status: "upcoming", start_date: "2026-11-23", end_date: "2026-11-29" });

/**
 * The cycle store as the preview reads it (listedCycles; the project's list holds the three cycles), with its change of
 * a cycle's days, which goes to nerve; nerve settles each when the test says (sent).
 */
function previewedCycles() {
  const cycles = listedCycles();
  cycles.show("cycles", web.id, [rowOf(draft), rowOf(alpha), rowOf(beta)]);
  const sent: HeldChange<Cycle>[] = [];
  const update = vi.fn((_cycleId: string, _body: CycleUpdate) => {
    const change = heldChange<Cycle>();
    sent.push(change);
    return change.sent;
  });
  return { ...cycles, update, sent };
}

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
function Peeker() {
  held.peek = usePeekCycle()[1];
  return null;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  signedIn();
  localStorage.clear();
  toasts.length = 0;
  held.cycles = previewedCycles();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

/** The list of web's cycles, previewing cycle. */
async function previewing(cycle: Cycle) {
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/acme/projects/${web.id}/cycles?peekCycle=${cycle.id}`]}>
          <CyclePeekOverview workspaceSlug="acme" projectId={web.id} />
          <Peeker />
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
/** The list previews another cycle, in the same panel. */
const previewAnother = (cycle: Cycle) => act(async () => held.peek?.(cycle.id));
/** nerve answers the last read of a cycle's progress: of no work items. */
const progressRead = (cycle: Cycle) => held.cycles.answer("progress", progressOf(cycle));
/** A day of November 2026, as the calendar gives it: the local midnight of the day. */
const november = (day: number) => new Date(2026, 10, day);
/** The person picks a day alone, the range's first. */
const pickFirst = (day: Date) => act(async () => held.dates?.onSelect?.({ from: day, to: undefined }));
/** The person picks a day after the range's start: the calendar ends the range it shows on it (react-day-picker). */
const pickEnd = (day: Date) => act(async () => held.dates?.onSelect?.({ from: held.dates?.value.from, to: day }));
/** The days the picker shows, as nerve writes them ("-" for none). */
const shownDays = () => {
  const { from, to } = held.dates?.value ?? {};
  return `${renderFormattedPayloadDate(from) ?? "-"} → ${renderFormattedPayloadDate(to) ?? "-"}`;
};
const datesEnabled = () => held.dates?.disabled !== true;
const name = () => host.querySelector("h4")?.textContent;
const selectedTab = () => host.querySelector('[role="tab"][aria-selected="true"]')?.textContent;
const tab = (title: string) =>
  [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')].find((one) => one.textContent === title);

it("previews another cycle afresh: a day picked alone for the cycle before is not the other's", async () => {
  await previewing(draft);
  await pickFirst(november(16));
  expect(shownDays()).toBe("2026-11-16 → -");
  await previewAnother(beta);
  expect(name()).toBe("Beta");
  expect(shownDays()).toBe("2026-11-23 → 2026-11-29");
  expect(held.cycles.update).not.toHaveBeenCalled();
  // the person ends Beta's range: its own start goes with it
  await pickEnd(november(30));
  expect(held.cycles.update).toHaveBeenCalledExactlyOnceWith(beta.id, {
    start_date: "2026-11-23",
    end_date: "2026-11-30",
  });
});

it("previews another cycle afresh: the cycle before's change of days still out is not the other's", async () => {
  await previewing(alpha);
  await pickFirst(november(17));
  await pickEnd(november(25));
  expect(held.cycles.update).toHaveBeenCalledExactlyOnceWith(alpha.id, {
    start_date: "2026-11-17",
    end_date: "2026-11-25",
  });
  expect(datesEnabled()).toBe(false);
  await previewAnother(beta);
  // Beta's days, which the person may change while Alpha's change is out
  expect(shownDays()).toBe("2026-11-23 → 2026-11-29");
  expect(datesEnabled()).toBe(true);
  // nerve makes Alpha's change: Beta's preview does not say so
  await act(async () => held.cycles.sent[0]?.answer({ ...alpha, start_date: "2026-11-17", end_date: "2026-11-25" }));
  expect(toasts).toEqual([]);
  expect(datesEnabled()).toBe(true);
});

it("previews another cycle afresh: the tab of its progress is the other's own", async () => {
  localStorage.setItem(`cycle-progress-tab-${beta.id}`, JSON.stringify("stat-states"));
  await previewing(beta);
  await progressRead(beta);
  expect(selectedTab()).toBe("common.states");
  await previewAnother(alpha);
  await progressRead(alpha);
  // Alpha has no tab kept: the first tab shown is the default
  expect(selectedTab()).toBe("common.assignees");
  await act(async () => tab("common.labels")?.click());
  expect(selectedTab()).toBe("common.labels");
  // Beta's progress is in the store already: its preview shows it at once, at the tab kept for Beta
  await previewAnother(beta);
  expect(name()).toBe("Beta");
  expect(selectedTab()).toBe("common.states");
});
