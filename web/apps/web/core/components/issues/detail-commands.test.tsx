// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState, type ReactNode, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until, track } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { IssueDetailRoot, type TIssueOperations } from "./issue-detail/root";
import { IssueParentSelectRoot } from "./parent-select-root";
import { useIssueOperations } from "./issue-detail/use-issue-operations";
import type { IssueParentSelect } from "./issue-detail/parent-select";
import { IssuePeekOverview } from "./peek-overview/root";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "commands" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "commands" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
  useIssueDetailModals: () => ({ toggleParentIssueModal: closed }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ issueDetailSidebarCollapsed: false }) }));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: tab.app.issue.projectIssues }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => undefined }));
vi.mock("./issue-detail/main-content", () => ({ IssueMainContent: (props: ProbeProps) => <Probe {...props} /> }));
vi.mock("./issue-detail/sidebar", () => ({ IssueDetailsSidebar: () => null }));
vi.mock("./peek-overview/view", () => ({ IssueView: (props: ProbeProps) => <Probe {...props} /> }));

const closed = vi.fn();
vi.mock("./issue-detail/parent-select", () => ({
  IssueParentSelect: (props: ComponentProps<typeof IssueParentSelect>) => (
    <>
      <button
        onClick={() =>
          void props
            .handleParentIssue("next")
            .then(() => closed(null))
            .catch(() => undefined)
        }
      >
        parent
      </button>
      <button onClick={() => void props.handleRemoveSubIssue("ws", "A", "next", "x")}>remove</button>
    </>
  ),
}));
function ParentHarness() {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return <IssueParentSelectRoot workspaceSlug="ws" projectId="A" issueId="x" issueOperations={issueOperations} />;
}
type ProbeProps = { issueId: string; issueOperations: TIssueOperations };
const operations = new Map<string, TIssueOperations>();
function Probe({ issueId, issueOperations }: ProbeProps) {
  operations.set(issueId, issueOperations);
  const [outcome, setOutcome] = useState("idle");
  return (
    <button
      onClick={() => {
        setOutcome("pending");
        void issueOperations.update("ws", "A", issueId, { name: "changed" }).then(
          () => setOutcome("confirmed"),
          () => setOutcome("refused")
        );
      }}
    >
      {issueId}:{outcome}
    </button>
  );
}
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
const cache = () => new Map();
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  for (const id of ["x", "next"]) tab.app.issue.issues.acceptRead(issueDetailOf(id), tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (/^\/api\/v0\/issues\/[^/]+$/.test(call.path))
      call.answer(json(200, issueDetailOf(call.path.split("/").at(-1)!)));
    else call.answer(json(200, { data: [], next_cursor: null }));
  };
  operations.clear();
  toasts.length = 0;
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
const render = async (view: ReactNode) =>
  act(async () => root.render(<SWRConfig value={{ provider: cache }}>{view}</SWRConfig>));
async function mount(kind: "detail" | "peek") {
  if (kind === "detail") await render(<IssueDetailRoot workspaceSlug="ws" projectId="A" issueId="x" />);
  else {
    tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "x" });
    await render(<IssuePeekOverview hostScope="list" />);
    await act(async () =>
      until(
        () =>
          tab.app.issue.issueDetail.subIssues.subIssuesByIssueId("x") !== undefined &&
          tab.app.issue.issueDetail.activity.getActivitiesByIssueId("x") !== undefined,
        "peek reads"
      )
    );
  }
  tab.nerve.calls.length = 0;
}
for (const kind of ["detail", "peek"] as const) {
  it.each([403, 422])(`${kind} keeps the caller refused and reports the native problem (%s)`, async (status) => {
    await mount(kind);
    await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
    await act(async () => until(() => tab.nerve.calls.length === 1, "one PATCH"));
    expect(host.textContent).toBe("x:pending");
    expect(tab.app.issue.issues.getIssueById("x")?.name).toBe("x");
    await act(async () => tab.nerve.calls[0]!.answer(problem(status, "forbidden")));
    await act(async () => until(() => host.textContent !== "x:pending", "settled UI"));
    expect(host.textContent).toBe("x:refused");
    expect(toasts.at(-1)?.message).toBe("errors.forbidden");
    expect(tab.app.issue.issues.getIssueById("x")?.name).toBe("x");
    expect(tab.nerve.calls).toHaveLength(1);
  });
  it(`${kind} follows one confirmation without duplicate activity reads`, async () => {
    await mount(kind);
    const result = track(operations.get("x")!.update("ws", "A", "x", { name: "changed" }));
    await act(async () => until(() => tab.nerve.calls.length === 1, "PATCH"));
    await act(async () =>
      tab.nerve.calls[0]!.answer(
        json(200, { ...issueDetailOf("x"), name: "changed", updated_at: "2026-10-30T09:00:00Z" })
      )
    );
    await act(async () => until(() => result.settled, "confirmed update"));
    expect(result.error).toBeUndefined();
    expect(tab.app.issue.issues.getIssueById("x")?.name).toBe("changed");
    expect(tab.nerve.to("/api/v0/issues/x/activities")).toHaveLength(kind === "peek" ? 1 : 0);
  });
  it(`${kind} does not toast or resolve an old caller successfully after unmount`, async () => {
    await mount(kind);
    const result = track(operations.get("x")!.update("ws", "A", "x", { name: "changed" }));
    await act(async () => until(() => tab.nerve.calls.length === 1, "old PATCH"));
    await render(null);
    await act(async () => tab.nerve.calls[0]!.answer(problem(403, "forbidden")));
    await act(async () => until(() => result.settled, "old refusal"));
    expect(result.error).toBeDefined();
    expect(toasts).toEqual([]);
  });
}
it("detail fetch leaves a native refusal available to its consumer", async () => {
  await mount("detail");
  tab.nerve.respond = undefined;
  const result = track(operations.get("x")!.fetch("ws", "A", "x"));
  await act(async () => until(() => tab.nerve.calls.length === 1, "detail GET"));
  await act(async () => tab.nerve.calls[0]!.answer(problem(403, "forbidden")));
  await act(async () => until(() => result.settled, "refused read"));
  expect(result.error).toMatchObject({ status: 403 });
  expect(tab.nerve.calls).toHaveLength(1);
});

it("the parent caller uses the confirmed core result without repeating child or parent reads", async () => {
  closed.mockClear();
  await render(<ParentHarness />);
  tab.nerve.calls.length = 0;
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  await act(async () => until(() => tab.nerve.calls.length === 1, "parent PATCH"));
  await act(async () => tab.nerve.calls[0]!.answer(json(200, { ...issueDetailOf("x"), parent_id: "next" })));
  await act(async () => until(() => closed.mock.calls.length === 1, "parent modal closes"));
  expect(tab.nerve.calls.map((call) => [call.method, call.path])).toEqual([
    ["PATCH", "/api/v0/issues/x"],
    ["GET", "/api/v0/issues/next"],
    ["GET", "/api/v0/projects/A/states"],
  ]);
});
it("removing a child releases its loader on refusal and reports the native failure", async () => {
  await render(<ParentHarness />);
  tab.nerve.calls.length = 0;
  await act(async () => host.querySelectorAll<HTMLButtonElement>("button")[1]!.click());
  await act(async () => until(() => tab.nerve.calls.length === 1, "remove PATCH"));
  await act(async () => host.querySelectorAll<HTMLButtonElement>("button")[1]!.click());
  expect(tab.nerve.calls).toHaveLength(1);
  await act(async () => tab.nerve.calls[0]!.answer(problem(403, "forbidden")));
  await act(async () => until(() => toasts.length > 0, "remove refused"));

  expect(toasts.at(-1)?.message).toBe("errors.forbidden");
  expect(tab.nerve.calls).toHaveLength(1);
});
