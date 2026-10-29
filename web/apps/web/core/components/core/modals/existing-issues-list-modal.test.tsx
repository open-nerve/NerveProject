// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import type { ISearchIssueResponse } from "@nerve/types";
import { heldChange } from "@/lib/auth/fake-tab";
import { ExistingIssuesListModal } from "./existing-issues-list-modal";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } } }));
vi.mock("@/services/project", () => ({ ProjectService: vi.fn(function ProjectService() {}) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/components/issues/issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("@nerve/propel/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
  }: React.PropsWithChildren<{ onClick?: React.MouseEventHandler<HTMLButtonElement>; disabled?: boolean }>) => (
    <button onClick={onClick} disabled={disabled}>
      {children}
    </button>
  ),
}));
vi.mock("@nerve/ui", () => ({
  EModalPosition: { CENTER: "center" },
  EModalWidth: { XXL: "xxl" },
  ModalCore: ({ children }: React.PropsWithChildren) => <div role="dialog">{children}</div>,
  Loader: Object.assign(() => null, { Item: () => null }),
}));
const item: ISearchIssueResponse = {
  id: "one",
  name: "Selected",
  project_id: "A",
  project__identifier: "A",
  project__name: "A",
  sequence_id: 1,
  start_date: null,
  state__color: "#ffffff",
  state__name: "Todo",
  workspace__slug: "ws",
};
const search = async () => [item];
let renderer: Root;
let container: HTMLElement;
let pending: ReturnType<typeof heldChange<void>>;
const close = vi.fn();
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  close.mockReset();
  pending = heldChange<void>();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  vi.useRealTimers();
});
const render = (project = "A", read = search) =>
  act(async () =>
    renderer.render(
      <ExistingIssuesListModal
        workspaceSlug="ws"
        projectId={project}
        isOpen
        handleClose={close}
        searchParams={{ issue_id: project }}
        selectedWorkItemIds={[item.id]}
        workItemSearchServiceCallback={read}
        handleOnSubmit={() => pending.sent}
      />
    )
  );
const send = () =>
  act(async () =>
    [...container.querySelectorAll("button")]
      .find((button) => button.textContent === "issue.select.add_selected")!
      .click()
  );
it("retains the selection and handles a failed submit without closing", async () => {
  await render();
  await send();
  await act(async () => pending.refuse(new Error("refused")));
  expect(close).not.toHaveBeenCalled();
  expect(
    [...container.querySelectorAll("button")].find((button) => button.textContent === "issue.select.add_selected")
      ?.disabled
  ).toBe(false);
  expect(container.querySelector<HTMLInputElement>('input[type="checkbox"]')?.checked).toBe(true);
});
it("a submit from the previous project does not close the current modal", async () => {
  await render();
  await send();
  await render("B");
  await act(async () => pending.answer());
  expect(close).not.toHaveBeenCalled();
});
it("shows a search refusal and retries only the read", async () => {
  const read = vi.fn().mockRejectedValueOnce(new Error("unavailable")).mockResolvedValue([item]);
  await render("A", read);
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  await act(async () =>
    [...container.querySelectorAll("button")].find((b) => b.textContent === "common.retry")!.click()
  );
  expect(read).toHaveBeenCalledTimes(2);
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(container.textContent).toContain("Selected");
  expect(close).not.toHaveBeenCalled();
});
it("a search from an old project cannot replace the new project's rows", async () => {
  const old = heldChange<ISearchIssueResponse[]>();
  await render("A", () => old.sent);
  await render("B", async () => [{ ...item, name: "Current project" }]);
  await act(async () => old.answer([{ ...item, name: "Old project" }]));
  expect(container.textContent).toContain("Current project");
  expect(container.textContent).not.toContain("Old project");
});
