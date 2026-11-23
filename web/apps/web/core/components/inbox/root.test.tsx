// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Link, MemoryRouter, Route, Routes, useLocation, useNavigate, useNavigationType } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeIssuePage } from "@nerve/api-client";
import type { EmptyStateCompact } from "@nerve/propel/empty-state";
import type { Call } from "@/lib/auth/fake-nerve";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { acme, intakeIssueOf, intakeTab, listPath, pageOf } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import type { InboxContentRoot } from "./content";
import { IntakePageStateProvider } from "./page-state";
import { InboxIssueRoot } from "./root";
import type { IntakeSidebar } from "./sidebar/root";

// The intake page's body (P5a spec 2.6, 2.7, R5, R14): the sidebar beside the selected work item's detail. The
// selection is the address's (?inboxIssueId): a row's press is the user's (a push), so that Back returns to the work
// item selected before; with none selected and the list read with rows, the first is selected by a replace, the page's
// own doing. The whole layout waits for the page's first read; a later read, of another tab or other filters, waits
// in the list alone, the tabs kept. On a narrow screen with nothing selected, a named button shows or hides the list.
// The sidebar and the detail are stand-ins here (the sidebar's rows links to their addresses); the list's read, the
// stores, SWR and HTTP are real, with the test's cache. @nerve/i18n's text is its key.

const held = vi.hoisted(() => ({
  where: undefined as { search: string; type: string } | undefined,
  navigate: undefined as unknown as (to: string | number) => void,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateCompact: (props: ComponentProps<typeof EmptyStateCompact>) => <p data-empty-state>{props.title}</p>,
}));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => tab.store }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("./content", () => ({
  InboxContentRoot: (props: ComponentProps<typeof InboxContentRoot>) => <p data-detail={props.inboxIssueId} />,
}));
vi.mock("./sidebar/root", () => ({
  IntakeSidebar: (props: ComponentProps<typeof IntakeSidebar>) => (
    <ul data-sidebar>
      {props.list.page?.data.map((row) => (
        <li key={row.id}>
          <Link to={`/acme/projects/p-web/intake?currentTab=open&inboxIssueId=${row.id}`}>{row.name}</Link>
        </li>
      ))}
    </ul>
  ),
}));

function Where() {
  const location = useLocation();
  held.where = { search: location.search, type: useNavigationType() };
  held.navigate = useNavigate() as never;
  return null;
}

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
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
                  <Where />
                  <InboxIssueRoot workspaceSlug="acme" projectId="p-web" />
                </IntakePageStateProvider>
              }
            />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));
async function listRead(before: number): Promise<Call> {
  await act(async () => until(() => tab.nerve.to(listPath("p-web")).length > before, `list read ${before + 1}`));
  return tab.nerve.to(listPath("p-web"))[before]!;
}
async function answer(call: Call, page: IntakeIssuePage) {
  call.answer(json(200, page));
  await settled();
}
const one = intakeIssueOf("One", { sequence_id: 1 });
const two = intakeIssueOf("Two", { sequence_id: 2 });
const sidebar = () => host.querySelector("[data-sidebar]");
const detail = () => host.querySelector<HTMLElement>("[data-detail]")?.dataset.detail;
const link = (name: string) => [...host.querySelectorAll("a")].find((found) => found.textContent === name)!;
const toggle = () => host.querySelector<HTMLButtonElement>('button[aria-label="inbox_issue.a11y.list_toggle"]');

describe("InboxIssueRoot", () => {
  it("waits for the page's first read with the whole layout's placeholder; a later read waits in the list alone", async () => {
    await render();
    const first = await listRead(0);
    expect(sidebar()).toBeNull();
    expect(host.querySelector('[role="status"]')?.textContent).toContain("common.loading");
    await answer(first, pageOf([one]));
    expect(sidebar()).not.toBeNull();
    await act(async () => held.navigate("/acme/projects/p-web/intake?currentTab=closed"));
    await listRead(1);
    expect(sidebar()).not.toBeNull();
  });

  it("selects the list's first row by a replace once it is read with rows and nothing is selected", async () => {
    await render();
    await answer(await listRead(0), pageOf([one, two]));
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-One", type: "REPLACE" });
    expect(detail()).toBe("i-One");
  });

  it("keeps the selection the address names, selecting nothing else", async () => {
    await render("?currentTab=open&inboxIssueId=i-Two");
    await answer(await listRead(0), pageOf([one, two]));
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Two", type: "POP" });
    expect(detail()).toBe("i-Two");
  });

  it("holds the selection in the address: a row's press is a push, and Back returns to the work item selected before", async () => {
    await render();
    await answer(await listRead(0), pageOf([one, two]));
    await act(async () => link("Two").click());
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Two", type: "PUSH" });
    expect(detail()).toBe("i-Two");
    await act(async () => held.navigate(-1));
    expect(held.where?.search).toBe("?currentTab=open&inboxIssueId=i-One");
    expect(detail()).toBe("i-One");
  });

  it("selects nothing in a list of no rows, saying to select a work item", async () => {
    await render();
    await answer(await listRead(0), pageOf([]));
    expect(held.where).toEqual({ search: "?currentTab=open", type: "POP" });
    expect(detail()).toBeUndefined();
    expect(host.querySelector("[data-empty-state]")?.textContent).toBe("project_empty_state.intake_main.title");
  });

  it("offers, on a narrow screen with nothing selected, a named button that shows and hides the list", async () => {
    await render();
    await answer(await listRead(0), pageOf([]));
    expect(toggle()?.getAttribute("aria-expanded")).toBe("true");
    await act(async () => toggle()!.click());
    expect(toggle()?.getAttribute("aria-expanded")).toBe("false");
    await act(async () => toggle()!.click());
    expect(toggle()?.getAttribute("aria-expanded")).toBe("true");
  });
});
