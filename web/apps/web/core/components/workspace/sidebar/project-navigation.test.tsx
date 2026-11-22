// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeCounts } from "@nerve/api-client";
import { FakeNerve, answered } from "@/lib/auth/fake-nerve";
import { countsOf } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { its, type Caller } from "@/store/user/fake-permissions";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// A project's items in the sidebar, as the intake's (P5a spec 2.5, R7, R21): the intake is offered by the intake entry
// rule (intakeEntryOn), and shows the project's pending work items beside its name while nerve counted some, the count
// in the link's name. The stores are a session's own (RootStore), over what nerve lists and counts in fake time; the
// fetch of the counts is the projects list's (useIntakeCounts), so the test makes it on the intake store. @nerve/i18n's
// text is its key, then the values it was given.

const held = vi.hoisted(() => ({ app: undefined as unknown as InstanceType<typeof RootStore> }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-app-theme", () => ({
  useAppTheme: () => ({
    isExtendedProjectSidebarOpened: false,
    toggleExtendedProjectSidebar: () => {},
    toggleSidebar: () => {},
  }),
}));
vi.mock("@/hooks/store/use-issue-detail", () => ({
  useIssueDetail: () => ({ issue: { getIssueIdByIdentifier: () => undefined, getIssueById: () => undefined } }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.app.user.permission }));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => held.app.issue.intake }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

const { RootStore } = await import("@/store/root.store");
const { ProjectNavigation } = await import("./project-navigation");

const acme = workspaceOf("acme");
let nerve: FakeNerve, host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});

/** The sidebar's items of web, in acme, of the caller's role in each; web's intake on. */
async function render(caller: Caller) {
  nerve = new FakeNerve();
  held.app = new RootStore(nerve.client());
  const workspace = { ...acme, role: caller.workspaceRole };
  const web = projectOf("WEB", acme.id, { intake_view: true, member_role: caller.projectRole });
  await loadWorkspaces(nerve, held.app.workspaceRoot, [workspace]);
  await loadProjects(nerve, held.app.projectRoot.project, workspace, [web]);
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/acme/projects/p-web/issues"]}>
        <Routes>
          <Route path="/:workspaceSlug/*" element={<ProjectNavigation workspaceSlug="acme" projectId="p-web" />} />
        </Routes>
      </MemoryRouter>
    )
  );
}
/** nerve gives counts as the pending work items of acme's intakes. */
const counted = (counts: IntakeCounts) =>
  act(async () => {
    await answered(
      nerve,
      () => held.app.issue.intake.fetchCounts(acme),
      ["GET", "/api/v0/workspaces/acme/intake-counts"],
      counts,
      "the counts"
    );
  });
const intake = () => host.querySelector<HTMLAnchorElement>('a[href="/acme/projects/p-web/intake"]');

describe("ProjectNavigation, the intake", () => {
  it("shows the project's pending count beside the intake, the count in the link's name", async () => {
    await render(its.admin);
    expect(intake()?.textContent).toBe("sidebar.intake");
    expect(intake()?.getAttribute("aria-label")).toBeNull();
    await counted(countsOf({ "p-web": 3, "p-api": 7 }));
    expect(intake()?.textContent).toBe("sidebar.intake3");
    expect(intake()?.getAttribute("aria-label")).toBe("inbox_issue.a11y.pending_count sidebar.intake 3");
  });

  it("shows no count while nothing is pending, or while nerve counted none of the project", async () => {
    await render(its.member);
    await counted(countsOf({ "p-web": 0 }));
    expect(intake()?.textContent).toBe("sidebar.intake");
    expect(intake()?.getAttribute("aria-label")).toBeNull();
    await counted(countsOf({ "p-api": 2 }));
    expect(intake()?.textContent).toBe("sidebar.intake");
    expect(intake()?.getAttribute("aria-label")).toBeNull();
  });

  it("offers the intake to the project's guest, with its count", async () => {
    await render(its.guest);
    await counted(countsOf({ "p-web": 1 }));
    expect(intake()?.getAttribute("aria-label")).toBe("inbox_issue.a11y.pending_count sidebar.intake 1");
  });

  it("offers no intake to the workspace's admin who has no role in the project, though its intake is on", async () => {
    await render(its.outsider);
    await counted(countsOf({ "p-web": 4 }));
    expect(host.querySelector("a")).toBeNull();
    expect(intake()).toBeNull();
  });
});
