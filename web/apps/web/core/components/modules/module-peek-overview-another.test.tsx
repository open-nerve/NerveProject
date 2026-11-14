// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { renderFormattedPayloadDate } from "@nerve/utils";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { moduleProgressOf, moduleRowOf } from "@/store/planning/fake-modules";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { listedModules } from "./fake-module-lists";
import { ModulePeekOverview, usePeekModule } from "./module-peek-overview";

// The preview of one module, then of another in the same panel (P6a spec 2.6; v0 design 7.7): the list previews
// another module by changing ?peekModule, and the other's preview starts afresh, as a module's own page does (its
// layout is keyed by the module): nothing of the module before — a day picked alone, a change of its days still out,
// the tab of its progress — carries into it. The sidebar is the real one (its links a stand-in); the modules a
// stand-in store's, whose change of a module's days goes to nerve, which settles it when the test says; the caller
// writes the project's modules.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("./fake-module-lists").listedModules>,
  /** The props the dates' picker was last given. */
  dates: undefined as ComponentProps<typeof DateRangeDropdown> | undefined,
  /** How the page previews another module (usePeekModule's). */
  peek: undefined as ((moduleId: string | null) => void) | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ canWriteModules: () => true }) }));
vi.mock("@/hooks/use-planning-reads", () => ({ usePlanningReads: () => ({ readable: true, listed: true }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ project: { getProjectMemberDetails: () => null } }),
}));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({ getProjectLabels: () => [] }) }));
// the module page's filters, whose values a row of the progress toggles (P6b): none here, so the rows are plain
vi.mock("@/hooks/store/work-item-filters/use-work-item-filters", () => ({
  useWorkItemFilters: () => ({ getFilter: () => undefined, updateFilterValueFromSidebar: () => undefined }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
vi.mock("@/components/dropdowns/member/dropdown", () => ({ MemberDropdown: () => null }));
vi.mock("./progress-sidebar/sidebar-links", () => ({ ModuleSidebarLinks: () => null }));
// the dates' picker keeps its props, so that a test reads the days it shows and picks a range as its calendar would
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => {
    held.dates = props;
    return <button type="button" aria-label="dates" disabled={props.disabled} />;
  },
}));

const web = projectOf("WEB", "id-acme");
const draft = moduleOf(web, "Draft");
const alpha = moduleOf(web, "Alpha", { status: "planned", start_date: "2026-11-16", target_date: "2026-11-22" });
const beta = moduleOf(web, "Beta", { status: "planned", start_date: "2026-11-23", target_date: "2026-11-29" });

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
function Peeker() {
  held.peek = usePeekModule()[1];
  return null;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  signedIn();
  localStorage.clear();
  toasts.length = 0;
  held.modules = listedModules();
  held.modules.show("modules", web.id, [moduleRowOf(draft), moduleRowOf(alpha), moduleRowOf(beta)]);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

/** The list of web's modules, previewing module. */
async function previewing(module: Module) {
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/acme/projects/${web.id}/modules?peekModule=${module.id}`]}>
          <ModulePeekOverview workspaceSlug="acme" projectId={web.id} />
          <Peeker />
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
/** The list previews another module, in the same panel. */
const previewAnother = (module: Module) => act(async () => held.peek?.(module.id));
/** nerve answers the last read of a module's progress: of no work items. */
const progressRead = () => held.modules.answer("progress", moduleProgressOf());
/** A day of November 2026, as the calendar gives it: the local midnight of the day. */
const november = (day: number) => new Date(2026, 10, day);
/** The person picks a day alone, the range's first. */
const pickFirst = (day: Date) => act(async () => held.dates?.onSelect?.({ from: day, to: undefined }));
/** The person picks a day after the range's start: the calendar ends the range it shows on it. */
const pickEnd = (day: Date) => act(async () => held.dates?.onSelect?.({ from: held.dates?.value.from, to: day }));
const shownDays = () => {
  const { from, to } = held.dates?.value ?? {};
  return `${renderFormattedPayloadDate(from) ?? "-"} → ${renderFormattedPayloadDate(to) ?? "-"}`;
};
const datesEnabled = () => held.dates?.disabled !== true;
const name = () => host.querySelector("h4")?.textContent;
const selectedTab = () => host.querySelector('[role="tab"][aria-selected="true"]')?.textContent;
const tab = (title: string) =>
  [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')].find((one) => one.textContent === title);

it("previews another module afresh: a day picked alone for the module before is not the other's", async () => {
  await previewing(draft);
  await pickFirst(november(16));
  expect(shownDays()).toBe("2026-11-16 → -");
  await previewAnother(beta);
  expect(name()).toBe("Beta");
  expect(shownDays()).toBe("2026-11-23 → 2026-11-29");
  expect(held.modules.update).not.toHaveBeenCalled();
  // the person ends Beta's range: its own start goes with it
  await pickEnd(november(30));
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(beta.id, {
    start_date: "2026-11-23",
    target_date: "2026-11-30",
  });
});

it("previews another module afresh: the module before's change of days still out is not the other's", async () => {
  await previewing(alpha);
  await pickFirst(november(17));
  await pickEnd(november(25));
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(alpha.id, {
    start_date: "2026-11-17",
    target_date: "2026-11-25",
  });
  expect(datesEnabled()).toBe(false);
  await previewAnother(beta);
  // Beta's days, which the person may change while Alpha's change is out
  expect(shownDays()).toBe("2026-11-23 → 2026-11-29");
  expect(datesEnabled()).toBe(true);
  // nerve makes Alpha's change: Beta's preview does not say so
  await act(async () =>
    held.modules.sent[0]?.answer({ ...alpha, start_date: "2026-11-17", target_date: "2026-11-25" })
  );
  expect(toasts).toEqual([]);
  expect(datesEnabled()).toBe(true);
});

it("previews another module afresh: the tab of its progress is the other's own", async () => {
  localStorage.setItem(`module-progress-tab-${beta.id}`, JSON.stringify("stat-states"));
  await previewing(beta);
  await progressRead();
  expect(selectedTab()).toBe("common.states");
  await previewAnother(alpha);
  await progressRead();
  // Alpha has no tab kept: the first tab shown is the default
  expect(selectedTab()).toBe("common.assignees");
  await act(async () => tab("common.labels")?.click());
  expect(selectedTab()).toBe("common.labels");
  // Beta's progress is in the store already: its preview shows it at once, at the tab kept for Beta
  await previewAnother(beta);
  expect(name()).toBe("Beta");
  expect(selectedTab()).toBe("common.states");
});
