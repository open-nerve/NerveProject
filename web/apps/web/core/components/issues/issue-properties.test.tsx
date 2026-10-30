// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EUserPermissions } from "@nerve/constants";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { projectOf, labelOf, loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { IssueLabelSelect } from "./select/dropdown";
import { IssueLabel } from "./issue-detail/label/root";
import { LabelDropdown } from "./issue-layouts/properties/label-dropdown";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => tab.app.label }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/use-project-inbox", () => ({
  useProjectInbox: () => ({ getIssueInboxByIssueId: () => undefined }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
const changed = vi.fn();
const choices = [
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
  vi.useRealTimers();
});
const posts = () => tab.nerve.calls.filter((call) => call.method === "POST");
const open = async (choice: (typeof choices)[number]) => {
  await act(async () => renderer.render(choice.view()));
  await act(async () => {
    host.querySelector<HTMLButtonElement>("button")!.click();
    await until(() => tab.nerve.calls.length === 1, "labels GET");
    await vi.advanceTimersByTimeAsync(20);
  });
};
const enter = async () => {
  await act(async () => {
    const input = host.querySelector<HTMLInputElement>("input")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New name");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => {
    host.querySelector("input")!.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
    await until(() => posts().length === 1, "label POST");
  });
};
it.each(choices)("$name places duplicate-name refusal under the name without attaching or closing", async (choice) => {
  await open(choice);
  await enter();
  expect(changed).not.toHaveBeenCalled();
  await act(async () => {
    posts()[0]!.answer(problem(409, "project.label_name_taken"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(host.querySelector<HTMLInputElement>("input")?.value).toBe("New name");
  expect(host.querySelector('input[aria-invalid="true"]')).not.toBeNull();
  expect(host.textContent).toContain("errors.project_label_name_taken");
  expect(toasts).toHaveLength(0);
  expect(changed).not.toHaveBeenCalled();
  expect(tab.app.label.getProjectLabels("A")).toEqual([]);
});
it.each(choices)("$name attaches only the confirmed label", async (choice) => {
  await open(choice);
  await enter();
  expect(changed).not.toHaveBeenCalled();
  const confirmed = labelOf(tab.project, "New name", 1);
  await act(async () => {
    posts()[0]!.answer(json(201, confirmed));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(changed).toHaveBeenCalledExactlyOnceWith([confirmed.id]);
  expect(posts()).toHaveLength(1);
});
it.each(choices)("$name ignores an old project's ordinary refusal after its scope changed", async (choice) => {
  await open(choice);
  await enter();
  await act(async () => renderer.render(choice.view("B")));
  await act(async () => {
    posts()[0]!.answer(problem(403, "forbidden"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(toasts).toHaveLength(0);
  expect(changed).not.toHaveBeenCalled();
  expect(host.textContent).not.toContain("errors.forbidden");
});
it("without a project, offers only fetched labels of still-visible projects with no reads", async () => {
  const held = labelOf(tab.project, "Held A", 1);
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [held] }));
  };
  expect((await settle(tab.app.label.fetchProjectLabels("A"), "held labels")).error).toBeUndefined();
  tab.nerve.calls.length = 0;
  await act(async () => renderer.render(<IssueLabelSelect projectId={undefined} value={[]} onChange={changed} />));
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  expect(host.textContent).toContain("Held A");
  expect(tab.nerve.calls).toHaveLength(0);
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  });
  expect(host.textContent).not.toContain("Held A");
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/labels"))).toHaveLength(0);
});

it.each(choices)("$name ignores a confirmed label from the previous project", async (choice) => {
  await open(choice);
  await enter();
  await act(async () => renderer.render(choice.view("B")));
  await act(async () => {
    posts()[0]!.answer(json(201, labelOf(tab.project, "Confirmed old", 1)));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(changed).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
it.each(choices)("$name sends one creation while repeated Enter presses are pending", async (choice) => {
  await open(choice);
  await enter();
  await act(async () => {
    host.querySelector("input")!.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
  });
  await act(async () => {
    posts()[0]!.answer(json(201, labelOf(tab.project, "Only once", 1)));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(posts()).toHaveLength(1);
  expect(changed).toHaveBeenCalledTimes(1);
});
it("an admin permission loss and recovery cannot revive a pending label refusal", async () => {
  await open(choices[0]!);
  await enter();
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    posts()[0]!.answer(problem(409, "project.label_name_taken"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(host.textContent).not.toContain("errors.project_label_name_taken");
  expect(toasts).toHaveLength(0);
});

it("a confirmed new label whose issue PATCH is refused remains unselected and preserves the query", async () => {
  await open({
    name: "detail-native",
    view: () => <IssueLabel workspaceSlug="ws" projectId="A" issueId="x" disabled={false} />,
  });
  await enter();
  const label = labelOf(tab.project, "New name", 1);
  await act(async () => posts()[0]!.answer(json(201, label)));
  await act(async () => until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "attach confirmed label"));
  expect(tab.app.issue.issues.getIssueById("x")?.label_ids).toEqual([]);
  await act(async () => tab.nerve.calls.find((call) => call.method === "PATCH")!.answer(problem(403, "forbidden")));
  await act(async () => vi.advanceTimersByTimeAsync(20));
  expect(host.querySelector<HTMLInputElement>("input")?.value).toBe("New name");
  expect(tab.app.issue.issues.getIssueById("x")?.label_ids).toEqual([]);
  expect(toasts.filter((toast) => toast.message === "errors.forbidden")).toHaveLength(1);
});
