// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DateFilterModal } from "@/components/core/filters/date-filter-modal";
import { press } from "@/components/workspace-notifications/sidebar/fake-header-menus";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { acme, intakeTab, listPath, pageOf } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { IntakePageStateProvider, useIntakePageState } from "../page-state";
import { useIntakeList } from "../use-intake-list";
import { IntakeFiltersMenu } from "./menu";

// The filters' menu of the list of the intake page's tab (P5a spec 2.2 C1–C3, 2.7, R9, R10, R23; M3 closeout's
// FiltersDropdown): a button named Filters that says whether its menu is open, Escape closing it and giving the focus
// back. Each value chosen asks nerve for exactly the list it names, each parameter once, two values one CSV parameter
// (the client's serializer, as the page sends it); the statuses are the tab's own, the last chosen staying; the members
// are the route project's, by its explicit ids; a preset names days of the caller's calendar as it is chosen, the
// custom dates' dialog after, before or between days, one range a field. The list's read (useIntakeList), the stores,
// SWR and HTTP are real; the members and labels are the test's, and so is the custom dates' dialog (its own tests are
// date-filter-modal's). @nerve/i18n's text is its key, then the values it was given.

const held = vi.hoisted(() => ({
  dialog: undefined as undefined | ComponentProps<typeof DateFilterModal>,
  memberIdsOf: [] as unknown[][],
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => tab.store }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/hooks/store/use-label", () => ({
  useLabel: () => ({
    getProjectLabels: (projectId: string) =>
      projectId === "p-web" ? [{ id: "l-bug", name: "Bug", color: "#f00" }] : undefined,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    getUserDetails: (id: string) =>
      ({
        "u-ann": { id: "u-ann", display_name: "Ann" },
        "u-bob": { id: "u-bob", display_name: "Bob" },
        "u-cat": { id: "u-cat", display_name: "Cat" },
        "u-router": { id: "u-router", display_name: "Router" },
      })[id],
    project: {
      // the route project's members, a guest (Cat) too; another project's, none
      getProjectMemberIds: (projectId: string, includeGuestUsers: boolean) => {
        held.memberIdsOf.push([projectId, includeGuestUsers]);
        return projectId === "p-web" ? ["u-ann", "u-bob", "u-cat"] : ["u-router"];
      },
      // the address's project, as the router store names it: another one
      projectMemberIds: ["u-router"],
    },
  }),
}));
vi.mock("@/components/core/filters/date-filter-modal", async () => {
  const { useEffect } = await import("react");
  return {
    // the dialog as the menu last drew it, open; none once the menu takes it away
    DateFilterModal: (props: ComponentProps<typeof DateFilterModal>) => {
      held.dialog = props.isOpen ? props : undefined;
      useEffect(
        () => () => {
          held.dialog = undefined;
        },
        []
      );
      return null;
    },
  };
});

const Page = observer(function Page() {
  const { filters } = useIntakePageState();
  useIntakeList("p-web", filters);
  return <IntakeFiltersMenu projectId="p-web" />;
});

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  // half an hour before the caller's midnight, west of UTC: nerve's UTC day is already the next
  vi.useFakeTimers({ now: new Date("2026-11-24T07:30:00Z") });
  vi.stubEnv("TZ", "America/Los_Angeles");
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  held.dialog = undefined;
  held.memberIdsOf = [];
  tab = await intakeTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [
    projectOf("WEB", acme.id, { intake_view: true, member_role: 20 }),
  ]);
  tab.nerve.calls.length = 0;
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  vi.useRealTimers();
});

const render = (search = "?currentTab=open") =>
  act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
        <MemoryRouter initialEntries={[`/acme/projects/p-web/intake${search}`]}>
          <Routes>
            <Route
              path="/:workspaceSlug/projects/:projectId/intake"
              element={
                <IntakePageStateProvider>
                  <Page />
                </IntakePageStateProvider>
              }
            />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));
const reads = () => tab.nerve.to(listPath("p-web"));
/**
 * The query of the list's read nerve receives after the `before` reads it received so far, each parameter in turn;
 * nerve answers it, a list of no work item, so that a later read of the same filters is a read of its own.
 */
async function sent(before: number): Promise<[string, string][]> {
  await act(async () => until(() => reads().length > before, `list read ${before + 1}`));
  const read = reads()[before]!;
  read.answer(json(200, pageOf([])));
  await settled();
  return read.queryEntries;
}
const filtersButton = () => host.querySelector<HTMLButtonElement>("button[aria-expanded]")!;
/** The menu's filter named title: its options, each by its name, and whether it is chosen. */
const section = (title: string) =>
  [...host.querySelectorAll<HTMLElement>("div.py-2")].find((found) =>
    found.querySelector("button[aria-expanded]")?.getAttribute("aria-label")?.startsWith(title)
  )!;
const options = (title: string) =>
  [...section(title).querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].map((option) => ({
    option,
    label: option.querySelector(".flex-grow")?.textContent,
    pressed: option.getAttribute("aria-pressed"),
  }));
const option = (title: string, label: string) => {
  const found = options(title).find((shownOption) => shownOption.label === label);
  expect(found, `${title}: ${label}`).toBeDefined();
  return found!.option;
};
const shown = (title: string) => options(title).map(({ label, pressed }) => [label, pressed]);
/** The menu opened by its button's Enter, the first read of the list answered. */
async function opened(search?: string) {
  await render(search);
  await sent(0);
  await press(filtersButton(), "Enter");
}
const pick = (title: string, label: string) => act(async () => option(title, label).click());
const first = (status = "pending"): [string, string] => ["status", status];
const newest: [string, string][] = [
  ["order_by", "-created_at"],
  ["limit", "50"],
];
const STATUS = "inbox_issue.filters.status";
const CREATED = "inbox_issue.filters.created_at";
const UPDATED = "inbox_issue.filters.updated_at";

describe("IntakeFiltersMenu's button", () => {
  it("is named Filters and says whether its menu is open; Escape closes it, the focus back on it", async () => {
    await render();
    expect(filtersButton().getAttribute("aria-label")).toBe("common.filters");
    expect(filtersButton().getAttribute("aria-expanded")).toBe("false");
    await press(filtersButton(), "Enter");
    expect(filtersButton().getAttribute("aria-expanded")).toBe("true");
    await press(host.querySelector<HTMLInputElement>('input[aria-label="common.filter_options.search"]')!, "Escape");
    expect(filtersButton().getAttribute("aria-expanded")).toBe("false");
    expect(document.activeElement).toBe(filtersButton());
  });
});

describe("IntakeFiltersMenu's filters", () => {
  it.each([
    { title: STATUS, label: "inbox_issue.status.snoozed.title", query: [first("pending,snoozed"), ...newest] },
    { title: "common.priority", label: "urgent", query: [first(), ["priority", "urgent"], ...newest] },
    { title: "common.assignees", label: "Bob", query: [first(), ["assignee_id", "u-bob"], ...newest] },
    { title: "common.created_by", label: "Cat", query: [first(), ["created_by_id", "u-cat"], ...newest] },
    { title: "common.labels", label: "Bug", query: [first(), ["label_id", "l-bug"], ...newest] },
    {
      title: CREATED,
      label: "inbox_issue.filters.presets.today",
      query: [first(), ["created_at", "2026-11-23"], ...newest],
    },
    {
      title: UPDATED,
      label: "inbox_issue.filters.presets.yesterday",
      query: [first(), ["updated_at", "2026-11-22"], ...newest],
    },
  ])(
    "asks nerve for exactly the list a value of $title names, each parameter once",
    async ({ title, label, query }) => {
      await opened();
      await pick(title, label);
      expect(await sent(1)).toEqual(query);
      expect(shown(title).find(([shownLabel]) => shownLabel === label)).toEqual([label, "true"]);
    }
  );

  it("sends two values of a filter as one parameter, never repeated (C2)", async () => {
    await opened();
    await pick("common.assignees", "Ann");
    await sent(1);
    await pick("common.assignees", "Cat");
    expect(await sent(2)).toEqual([first(), ["assignee_id", "u-ann,u-cat"], ...newest]);
    await pick("common.priority", "high");
    await pick("common.priority", "low");
    expect(await sent(4)).toEqual([first(), ["priority", "high,low"], ["assignee_id", "u-ann,u-cat"], ...newest]);
  });
});

describe("IntakeFiltersMenu's statuses", () => {
  it("offers the open tab's pending and snoozed; the last one chosen stays, asking nerve nothing more (R9)", async () => {
    await opened();
    expect(shown(STATUS)).toEqual([
      ["inbox_issue.status.pending.title", "true"],
      ["inbox_issue.status.snoozed.title", "false"],
    ]);
    await pick(STATUS, "inbox_issue.status.pending.title");
    await settled();
    expect(reads()).toHaveLength(1);
    expect(shown(STATUS)[0]).toEqual(["inbox_issue.status.pending.title", "true"]);
  });

  it("offers the closed tab's three final statuses, each turned off but the last", async () => {
    await opened("?currentTab=closed");
    expect(shown(STATUS)).toEqual([
      ["inbox_issue.status.accepted.title", "true"],
      ["inbox_issue.status.declined.title", "true"],
      ["inbox_issue.status.duplicate.title", "true"],
    ]);
    await pick(STATUS, "inbox_issue.status.declined.title");
    expect(await sent(1)).toEqual([first("accepted,duplicate"), ...newest]);
    await pick(STATUS, "inbox_issue.status.accepted.title");
    expect(await sent(2)).toEqual([first("duplicate"), ...newest]);
    await pick(STATUS, "inbox_issue.status.duplicate.title");
    await settled();
    expect(reads()).toHaveLength(3);
  });
});

describe("IntakeFiltersMenu's members", () => {
  it("offers the route project's members, guests too, by its explicit ids: never the address's (R23)", async () => {
    await opened();
    expect([shown("common.assignees"), shown("common.created_by")]).toEqual([
      [
        ["Ann", "false"],
        ["Bob", "false"],
        ["Cat", "false"],
      ],
      [
        ["Ann", "false"],
        ["Bob", "false"],
        ["Cat", "false"],
      ],
    ]);
    expect(held.memberIdsOf.every(([projectId, guests]) => projectId === "p-web" && guests === true)).toBe(true);
  });
});

describe("IntakeFiltersMenu's days", () => {
  it("names a preset's days of the caller's calendar as it is chosen, one range a field (R10, C3)", async () => {
    await opened();
    await pick(CREATED, "inbox_issue.filters.presets.last_7_days");
    expect(await sent(1)).toEqual([first(), ["created_at", "2026-11-16..2026-11-23"], ...newest]);
    // another preset takes its place: one range
    await pick(CREATED, "inbox_issue.filters.presets.last_30_days");
    expect(await sent(2)).toEqual([first(), ["created_at", "2026-10-24..2026-11-23"], ...newest]);
    expect(shown(CREATED).filter(([, pressed]) => pressed === "true")).toEqual([
      ["inbox_issue.filters.presets.last_30_days", "true"],
    ]);
    // chosen again, it turns off
    await pick(CREATED, "inbox_issue.filters.presets.last_30_days");
    expect(await sent(3)).toEqual([first(), ...newest]);
  });

  it("takes the custom dates' dialog's after, before and between as a.., ..b and a..b, the field's one range", async () => {
    await opened();
    const custom = async (values: string[]) => {
      await pick(UPDATED, "common.filter_options.custom");
      expect(held.dialog?.title).toBe(UPDATED);
      await act(async () => {
        held.dialog!.onSelect(values);
        held.dialog!.handleClose();
      });
      expect(held.dialog).toBeUndefined();
    };
    await custom(["2026-11-01;after"]);
    expect(await sent(1)).toEqual([first(), ["updated_at", "2026-11-01.."], ...newest]);
    expect(shown(UPDATED).at(-1)).toEqual(["common.filter_options.custom", "true"]);
    // chosen again, the custom days turn off
    await pick(UPDATED, "common.filter_options.custom");
    expect(await sent(2)).toEqual([first(), ...newest]);
    await custom(["2026-11-20;before"]);
    expect(await sent(3)).toEqual([first(), ["updated_at", "..2026-11-20"], ...newest]);
    await pick(UPDATED, "inbox_issue.filters.presets.today");
    expect(await sent(4)).toEqual([first(), ["updated_at", "2026-11-23"], ...newest]);
    await custom(["2026-11-01;after", "2026-11-20;before"]);
    expect(await sent(5)).toEqual([first(), ["updated_at", "2026-11-01..2026-11-20"], ...newest]);
  });
});
