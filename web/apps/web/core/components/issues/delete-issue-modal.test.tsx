// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { noContent, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { DeleteIssueModal } from "./delete-issue-modal";
const binding = vi.hoisted(() => ({
  tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined,
  close: vi.fn(),
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-issues", () => ({
  useIssues: () => ({ issueMap: binding.tab!.app.issue.issues.issuesMap }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { id: "actor" } }),
  useUserPermissions: () => ({ allowPermissions: () => true }),
}));
vi.mock("@nerve/ui", () => ({
  AlertModalCore: ({ handleSubmit, isSubmitting }: { handleSubmit: () => Promise<void>; isSubmitting: boolean }) => (
    <button disabled={isSubmitting} onClick={() => void handleSubmit()}>
      Delete
    </button>
  ),
}));
let renderer: Root;
let container: HTMLElement;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  toasts.length = 0;
  binding.close.mockReset();
  binding.tab = await issueStoreTab();
  for (const id of ["one", "two"])
    binding.tab.app.issue.issues.acceptRead(issueDetailOf(id), binding.tab.app.issue.captureRead());
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  binding.tab!.app.dispose();
  vi.useRealTimers();
});
const render = (id: string) =>
  act(async () =>
    renderer.render(
      <DeleteIssueModal
        isOpen
        handleClose={binding.close}
        data={issueDetailOf(id)}
        onSubmit={() => binding.tab!.app.issue.deleteIssue(id, binding.tab!.app.issue.scope("ws", "A"))}
      />
    )
  );
it("keeps the delete confirmation open and retryable on refusal", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  expect(container.querySelector("button")!.disabled).toBe(true);
  await act(async () => binding.tab!.nerve.calls[0]!.answer(problem(403, "forbidden")));
  expect(binding.close).not.toHaveBeenCalled();
  expect(container.querySelector("button")!.disabled).toBe(false);
  expect(toasts).toHaveLength(1);
  expect(toasts[0]?.type).toBe("error");
});
it("closes once after a confirmed deletion", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  expect(binding.close).not.toHaveBeenCalled();
  await act(async () => binding.tab!.nerve.calls[0]!.answer(noContent()));
  expect(binding.close).toHaveBeenCalledTimes(1);
});
it("does not close a new issue's modal when the old deletion completes", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  await render("two");
  await act(async () => binding.tab!.nerve.calls[0]!.answer(noContent()));
  expect(binding.close).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
