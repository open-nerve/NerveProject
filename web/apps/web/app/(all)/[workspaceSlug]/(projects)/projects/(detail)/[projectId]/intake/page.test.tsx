// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { acme, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import Page from "./page";

// A project's intake page (P5a spec 2.5, 2.7, R19): by the route's project and the caller's role in it, never the
// router store's. Its intake turned off: the disabled page, whose way to turning it on is the project's admin's, and
// no intake (so no read of its list); else the intake, the page titled by the project's name. The project store and the
// permissions are the session's own, as nerve lists acme and its projects to u-me; the intake and the disabled page
// are stand-ins. @nerve/i18n's text is its key, then the values it was given.

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/components/inbox", () => ({ InboxIssueRoot: () => <p data-intake /> }));
vi.mock("@/components/empty-state/detailed-empty-state-root", () => ({
  DetailedEmptyState: (props: ComponentProps<typeof DetailedEmptyState>) => (
    <div data-disabled-page>
      <p>{props.title}</p>
      <button type="button" disabled={props.primaryButton?.disabled}>
        {props.primaryButton?.text}
      </button>
    </div>
  ),
}));

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  document.title = "";
  tab = await intakeTab();
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

const ADMIN = 20 as const;
const MEMBER = 15 as const;
/** The caller's acme lists WEB as fields say and OPS, its intake on, he its admin. */
async function webIs(fields: Partial<Project>) {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [
    projectOf("WEB", acme.id, fields),
    projectOf("OPS", acme.id, { intake_view: true, member_role: ADMIN }),
  ]);
}
const Intake = Page as unknown as (props: { params: Record<string, string> }) => React.JSX.Element;
const draw = () =>
  act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/acme/projects/p-web/intake"]}>
        <Intake params={{ workspaceSlug: "acme", projectId: "p-web" }} />
      </MemoryRouter>
    )
  );
const disabledPage = () => host.querySelector("[data-disabled-page]");
const enable = () => host.querySelector<HTMLButtonElement>("[data-disabled-page] button");

describe("the intake page", () => {
  it("shows the route's project's intake, titled by the project's name", async () => {
    await webIs({ intake_view: true, member_role: MEMBER });
    await draw();
    expect(host.querySelector("[data-intake]")).not.toBeNull();
    expect(disabledPage()).toBeNull();
    expect(document.title).toBe("inbox_issue.page_label WEB");
  });

  it.each([
    ["the project's admin may turn it on", ADMIN, false],
    ["a member may not", MEMBER, true],
  ] as const)("is disabled, its intake off, and shows no intake: %s", async (_, role, disabled) => {
    await webIs({ intake_view: false, member_role: role });
    await draw();
    expect(disabledPage()?.textContent).toContain("disabled_project.empty_state.inbox.title");
    expect(enable()?.disabled).toBe(disabled);
    expect(host.querySelector("[data-intake]")).toBeNull();
  });

  it("reads the route's project, not the router store's", async () => {
    await webIs({ intake_view: false, member_role: MEMBER });
    runInAction(() => tab.app.router.setQuery({ workspaceSlug: "acme", projectId: "p-ops" }));
    await draw();
    expect(disabledPage()).not.toBeNull();
    expect(enable()?.disabled).toBe(true);
    expect(host.querySelector("[data-intake]")).toBeNull();
  });
});
