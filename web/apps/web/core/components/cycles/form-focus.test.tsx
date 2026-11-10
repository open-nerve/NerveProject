// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ModalCore, ModalTitle } from "@nerve/ui";
import { CycleForm } from "./form";
import { ModuleForm } from "../modules/form";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/user/user-user", () => ({
  useUser: () => ({ projectsWithCreatePermissions: { project: true } }),
}));
// Unrelated asynchronous fields are not involved in the dialog's initial focus contract.
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: () => <button type="button">Project</button>,
}));
vi.mock("@/components/dropdowns/member/dropdown", () => ({
  MemberDropdown: () => <button type="button">Member</button>,
}));
vi.mock("@/components/dropdowns/date-range", () => ({ DateRangeDropdown: () => <button type="button">Dates</button> }));
vi.mock("@/components/modules", () => ({ ModuleStatusSelect: () => <button type="button">Status</button> }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
for (const kind of ["cycle", "module"] as const) {
  it(`${kind}: the existing modal owns initial name focus even with a preceding project selector`, async () => {
    const submitted = vi.fn(),
      closed = vi.fn();
    const props = {
      handleFormSubmit: submitted,
      handleClose: closed,
      status: false,
      projectId: "project",
      setActiveProject: vi.fn(),
    };
    await act(async () =>
      root.render(
        <ModalCore isOpen handleClose={closed}>
          <ModalTitle>{kind}</ModalTitle>
          {kind === "cycle" ? (
            <CycleForm projectId="project" editing={false} busy={false} onSubmit={submitted} onCancel={closed} />
          ) : (
            <ModuleForm {...props} />
          )}
        </ModalCore>
      )
    );
    await act(async () => vi.advanceTimersByTimeAsync(100));
    const input = document.querySelector<HTMLInputElement>('input[name="name"]');
    expect(input).not.toBeNull();
    expect(document.activeElement).toBe(input);
    expect(document.querySelector('[role="dialog"]')?.contains(input)).toBe(true);
    expect(submitted).not.toHaveBeenCalled();
    expect(closed).not.toHaveBeenCalled();
  });
}
