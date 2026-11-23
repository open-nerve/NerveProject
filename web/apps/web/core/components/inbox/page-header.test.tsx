// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { acme, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import { IntakePageHeader } from "./page-header";

// The intake page's header (P5a spec 2.7, R21, R27; HO/M3-closeout.md:64): the route's project's breadcrumbs, the
// intake's named in the page's language; "Add work item" only while the project's intake is on, the project is not
// archived (nerve refuses a submission to it, project.archived) and the caller has a role in it, an admin's, a
// member's or a guest's, by the route's project, never the router store's. The project store and the permissions are
// the session's own, as nerve lists acme and its projects to u-me; the breadcrumbs of the project and the create modal
// are stand-ins. @nerve/i18n's text is its key.

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/inbox/modals/create-modal", () => ({ InboxIssueCreateModalRoot: () => null }));

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  tab = await intakeTab();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ADMIN = 20 as const;
const MEMBER = 15 as const;
const GUEST = 5 as const;
/** The caller's acme lists WEB as fields say (its intake on unless they say otherwise) and OPS, its intake off. */
async function webIs(fields: Partial<Project>) {
  const web = projectOf("WEB", acme.id, { intake_view: true, ...fields });
  const ops = projectOf("OPS", acme.id, { intake_view: false, member_role: ADMIN });
  const store = tab.app.projectRoot.project;
  await loadProjects(tab.nerve, store, acme, web.archived_at ? [ops] : [web, ops]);
  if (web.archived_at) await loadArchivedProjects(tab.nerve, store, acme, [web]);
}
/** The header of WEB's intake page, at its address. */
const draw = () =>
  act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/acme/projects/p-web/intake"]}>
        <Routes>
          <Route path="/:workspaceSlug/projects/:projectId/intake" element={<IntakePageHeader />} />
        </Routes>
      </MemoryRouter>
    )
  );
const add = () => [...host.querySelectorAll("button")].find((button) => button.textContent === "add_work_item");

describe("IntakePageHeader", () => {
  it.each([
    ["an admin", ADMIN],
    ["a member", MEMBER],
    ["a guest", GUEST],
  ] as const)("offers %s of the project Add work item while its intake is on", async (_, role) => {
    await webIs({ member_role: role });
    await draw();
    expect(add()).toBeDefined();
  });

  it.each<[string, Partial<Project>]>([
    ["while the project's intake is off", { intake_view: false, member_role: ADMIN }],
    ["to a caller of no role in the project", { member_role: null }],
    [
      "in an archived project, which takes no submission (R27)",
      { member_role: ADMIN, archived_at: "2026-11-01T00:00:00Z" },
    ],
  ])("offers no Add work item %s", async (_, fields) => {
    await webIs(fields);
    await draw();
    expect(add()).toBeUndefined();
  });

  it("names the intake's breadcrumb in the page's language, a link to the intake page", async () => {
    await webIs({ member_role: ADMIN });
    await draw();
    const crumb = [...host.querySelectorAll("a")].find((link) => link.textContent?.includes("sidebar.intake"));
    expect(crumb?.getAttribute("href")).toBe("/acme/projects/p-web/intake");
    expect(host.textContent).not.toContain("Intake");
  });

  it("reads the route's project, not the router store's", async () => {
    await webIs({ member_role: GUEST });
    runInAction(() => tab.app.router.setQuery({ workspaceSlug: "acme", projectId: "p-ops" }));
    await draw();
    expect(add()).toBeDefined();
  });
});
