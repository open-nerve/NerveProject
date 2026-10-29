// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { IssueLabelSelect } from "./select/dropdown";
import { IssueLabelSelect as DetailLabels } from "./issue-detail/label/select/label-select";
import { LabelDropdown } from "./issue-layouts/properties/label-dropdown";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => tab.app.memberRoot }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => tab.app.label }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => tab.app.user.permission,
  useUser: () => ({ data: { id: "actor" } }),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
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
const noop = () => undefined;
const choices = [
  {
    name: "state",
    path: "states",
    view: () => <StateDropdown projectId="A" value={null} onChange={noop} buttonVariant="border-with-text" />,
  },
  {
    name: "member",
    path: "members",
    view: () => (
      <MemberDropdown projectId="A" value={null} multiple={false} onChange={noop} buttonVariant="border-with-text" />
    ),
  },
  { name: "modal labels", path: "labels", view: () => <IssueLabelSelect projectId="A" value={[]} onChange={noop} /> },
  {
    name: "detail labels",
    path: "labels",
    view: () => (
      <DetailLabels
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        values={[]}
        onSelect={noop}
        onAddLabel={async () => {
          throw new Error("Unexpected creation");
        }}
      />
    ),
  },
  {
    name: "workspace member",
    path: "workspace-members",
    view: () => <MemberDropdown value={null} multiple={false} onChange={noop} buttonVariant="border-with-text" />,
  },
  {
    name: "row labels",
    path: "labels",
    view: () => <LabelDropdown projectId="A" value={[]} onChange={noop} label="labels" />,
  },
];
it.each(choices)(
  "$name shows a current directory refusal, ends loading and retries only GET",
  async ({ path, view }) => {
    const endpoint = path === "workspace-members" ? "/api/v0/workspaces/ws/members" : `/api/v0/projects/A/${path}`;
    await act(async () => renderer.render(view()));
    await act(async () => {
      host.querySelector<HTMLButtonElement>("button")!.click();
      await until(() => tab.nerve.calls.length === 1, "directory GET");
    });
    expect(tab.nerve.calls.map((call) => [call.method, call.path])).toEqual([["GET", endpoint]]);
    await act(async () => {
      tab.nerve.calls[0]!.answer(problem(403, "forbidden"));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(document.body.querySelector('[role="alert"]')?.textContent).toContain("errors.forbidden");
    expect(host.textContent).not.toContain("common.loading");
    await act(async () => {
      [...document.body.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!.click();
      await until(() => tab.nerve.calls.length === 2, "retry GET");
      tab.nerve.calls[1]!.answer(json(200, { data: [] }));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(tab.nerve.calls.map((call) => [call.method, call.path])).toEqual(
      Array.from({ length: 2 }, () => ["GET", endpoint])
    );
    expect(document.body.querySelector('[role="alert"]')).toBeNull();
    expect(host.textContent).not.toContain("common.loading");
  }
);

it("state has a single native trigger button, so keyboard focus and hydration do not diverge", async () => {
  await act(async () => renderer.render(choices[0]!.view()));
  expect(host.querySelectorAll("button button")).toHaveLength(0);
});

it("does not display the old directory refusal after permission loss and recovery", async () => {
  await act(async () => renderer.render(choices[0]!.view()));
  await act(async () => {
    host.querySelector<HTMLButtonElement>("button")!.click();
    await until(() => tab.nerve.calls.length === 1, "states");
  });
  const old = tab.nerve.calls[0]!;
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    old.answer(problem(403, "forbidden"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(document.body.querySelector('[role="alert"]')).toBeNull();
});

it("the background state trigger forwards native attributes without leaking private presentation props", async () => {
  const errors = vi.spyOn(console, "error").mockImplementation(() => {});
  try {
    await act(async () =>
      renderer.render(
        <StateDropdown projectId="A" value={null} onChange={noop} buttonVariant="background-with-text" tabIndex={0} />
      )
    );
    expect(host.querySelector("button")?.tabIndex).toBe(0);
    expect(errors.mock.calls).toEqual([]);
  } finally {
    errors.mockRestore();
  }
});

it.each(["state", "member"])(
  "%s opens from its only native keyboard trigger and starts the directory read",
  async (kind) => {
    await act(async () =>
      renderer.render(
        kind === "state" ? (
          <StateDropdown projectId="A" value={null} onChange={noop} buttonVariant="border-with-text" tabIndex={0} />
        ) : (
          <MemberDropdown
            projectId="A"
            value={null}
            multiple={false}
            onChange={noop}
            buttonVariant="border-with-text"
            tabIndex={0}
          />
        )
      )
    );
    const button = host.querySelector<HTMLButtonElement>("button")!;
    expect(button.tabIndex).toBe(0);
    expect(host.querySelectorAll("button")).toHaveLength(1);
    await act(async () => {
      button.focus();
      button.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
      await until(() => tab.nerve.calls.length === 1, "keyboard directory read");
    });
    expect(tab.nerve.calls[0]!.path).toBe(`/api/v0/projects/A/${kind === "state" ? "states" : "members"}`);
  }
);
