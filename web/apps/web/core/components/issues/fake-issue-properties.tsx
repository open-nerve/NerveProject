// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, vi, type Mock } from "vitest";
import { EUserPermissions } from "@nerve/constants";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { projectOf, loadProjects } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { IssueLabelSelect } from "./select/dropdown";
import { IssueLabel } from "./issue-detail/label/root";
import { LabelDropdown } from "./issue-layouts/properties/label-dropdown";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }), useNavigate: () => vi.fn() }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => tab.app.label }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/use-project-inbox", () => ({
  useProjectInbox: () => ({ getIssueInboxByIssueId: () => undefined }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
export let tab: Awaited<ReturnType<typeof issueStoreTab>>;
export let host: HTMLDivElement;
export let renderer: ReturnType<typeof createRoot>;
export const changed: Mock<(ids: string[]) => void | Promise<void>> = vi.fn();
export const choices = [
  { name: "modal", view: (project = "A") => <IssueLabelSelect projectId={project} value={[]} onChange={changed} /> },
  {
    name: "row",
    view: (project = "A") => <LabelDropdown projectId={project} value={[]} onChange={changed} label="labels" />,
  },
  {
    name: "detail",
    view: (project = "A") => (
      <IssueLabel workspaceSlug="ws" projectId={project} issueId="x" disabled={false} onLabelUpdate={changed} />
    ),
  },
];
beforeEach(async () => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  changed.mockReset();
  toasts.length = 0;
  tab = await issueStoreTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: EUserPermissions.ADMIN },
    projectOf("B", tab.workspace.id, { id: "B", member_role: EUserPermissions.ADMIN }),
  ]);
  tab.nerve.calls.length = 0;
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [] }));
  };
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
export const namedButton = (name: string) =>
  [...document.querySelectorAll("button")].find((item) => item.textContent === name)!;
export const posts = () => tab.nerve.calls.filter((call) => call.method === "POST");
export const open = async (choice: (typeof choices)[number]) => {
  await act(async () => renderer.render(choice.view()));
  await act(async () => {
    host.querySelector<HTMLButtonElement>("button")!.click();
    await until(() => tab.nerve.calls.length === 1, "labels GET");
    await vi.advanceTimersByTimeAsync(20);
  });
};
export const enter = async () => {
  await act(async () => {
    const input = document.querySelector<HTMLInputElement>("input")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New name");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => {
    document.querySelector("input")!.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
    await until(() => posts().length === 1, "label POST");
  });
};
