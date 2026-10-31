// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type PropsWithChildren } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { IssueDetailHost } from "./detail-host";
import { IssueParentSelectRoot } from "./parent-select-root";
import { useIssueOperations } from "./issue-detail/use-issue-operations";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "parent" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "parent" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({
  useParams: () => ({ workspaceSlug: "ws" }),
  Link: ({ children }: PropsWithChildren) => <span>{children}</span>,
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("./issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ModalTitle: ({ children }: PropsWithChildren) => <h3>{children}</h3>,
  ModalCore: ({ children, isOpen }: PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
function Host() {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="x">
      <IssueParentSelectRoot workspaceSlug="ws" projectId="A" issueId="x" issueOperations={issueOperations} />
    </IssueDetailHost>
  );
}
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (call.path === "/api/v0/workspaces/ws/issue-search")
      call.answer(json(200, { data: [{ ...issueDetailOf("next"), name: "Next parent" }] }));
    else if (call.path === "/api/v0/issues/next") call.answer(json(200, issueDetailOf("next")));
    else call.answer(json(200, { data: [], next_cursor: null }));
  };
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
  await act(async () => renderer.render(<Host />));
  await act(async () => host.querySelector<HTMLButtonElement>('button[aria-label="issue.add.parent"]')!.click());
  await act(async () => {
    const input = host.querySelector("input")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Next");
    input.dispatchEvent(new Event("input", { bubbles: true }));
    await vi.advanceTimersByTimeAsync(500);
  });
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const select = async () => {
  await act(async () => {
    const input = host.querySelector("input")!;
    input.focus();
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }));
    await vi.advanceTimersByTimeAsync(20);
  });
  await act(async () =>
    host
      .querySelector<HTMLElement>('[role="option"]')!
      .dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }))
  );
};
it.each([200, 403, 422])("waits for parent confirmation and retains search on refusal (%s)", async (status) => {
  await select();
  expect(host.querySelector('[role="dialog"]')).not.toBeNull();
  await act(async () => until(() => tab.nerve.calls.some((call) => call.method === "PATCH"), "parent write"));
  expect(host.querySelector('[role="dialog"]')).not.toBeNull();
  await select();
  const writes = tab.nerve.calls.filter((call) => call.method === "PATCH");
  expect(writes).toHaveLength(1);
  await act(async () =>
    writes[0]!.answer(
      status === 200 ? json(200, { ...issueDetailOf("x"), parent_id: "next" }) : problem(status, "forbidden")
    )
  );
  if (status === 200) expect(host.querySelector('[role="dialog"]')).toBeNull();
  else {
    expect(host.querySelector<HTMLInputElement>("input")?.value).toBe("Next");
    expect(host.querySelector('[role="dialog"]')).not.toBeNull();
    expect(toasts.at(-1)?.message).toBe("errors.forbidden");
    expect(tab.app.issue.issues.getIssueById("x")?.parent_id).toBeNull();
  }
});
