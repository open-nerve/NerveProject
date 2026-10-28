// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { CalendarQuickAddIssueActions } from "./quick-add-issue-actions";

const calls = vi.hoisted(() => ({ update: vi.fn(() => new Promise<never>(() => {})) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "project" }),
}));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => ({ updateIssue: calls.update }) }));
vi.mock("@nerve/propel/toast", () => ({ setPromiseToast: vi.fn() }));
vi.mock("../quick-add", () => ({ QuickAddIssueRoot: () => null }));
vi.mock("@/components/core/modals/existing-issues-list-modal", () => ({
  ExistingIssuesListModal: ({
    searchParams,
    handleOnSubmit,
  }: {
    searchParams: { target_date: string };
    handleOnSubmit: (issues: { id: string }[]) => void;
  }) => <button onClick={() => handleOnSubmit([{ id: "issue" }])}>{searchParams.target_date}</button>,
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
it("sets the selected existing project's work item date", async () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  await act(async () => root.render(<CalendarQuickAddIssueActions prePopulatedData={{ target_date: "2026-11-01" }} />));
  expect(container.textContent).toBe("none");
  await act(async () => container.querySelector("button")?.click());
  expect(calls.update).toHaveBeenCalledWith("ws", "project", "issue", { target_date: "2026-11-01" });
  await act(async () => root.unmount());
});
