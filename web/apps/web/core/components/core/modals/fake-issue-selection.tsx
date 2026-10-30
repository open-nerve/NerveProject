// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { vi } from "vitest";
import { StoreContext } from "@/lib/store-context";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { stateOf } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";

/** Real session stores and selector DOM shared by the calendar, detail widget and selection tests. */
export async function issueSelectionTab() {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
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
  const tab = await issueStoreTab();
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/states")) {
      const project = tab.app.projectRoot.project.getProjectById(call.path.split("/").at(-2))!;
      call.answer(json(200, { data: [stateOf(project, "Todo", "unstarted", 1, { id: "state" })] }));
    }
  };
  const container = document.createElement("div");
  document.body.append(container);
  const renderer = createRoot(container);
  return {
    ...tab,
    render: (children: ReactNode) =>
      act(async () => renderer.render(<StoreContext.Provider value={tab.app}>{children}</StoreContext.Provider>)),
    close: async () => {
      await act(async () => renderer.unmount());
      container.remove();
      tab.app.dispose();
      vi.restoreAllMocks();
      vi.unstubAllGlobals();
      vi.useRealTimers();
    },
    searches: () => tab.nerve.to("/api/v0/workspaces/ws/issue-search"),
  };
}
export function selectionButton(text: string): HTMLButtonElement {
  const button = [...document.querySelectorAll("button")].find((element) => element.textContent === text);
  if (!button) throw new Error(`Missing button: ${text}`);
  return button;
}
export const clickSelection = (selector: string) =>
  act(async () => {
    const element = document.querySelector<HTMLElement>(selector);
    if (!element) throw new Error(`Missing selector: ${selector}`);
    element.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    element.dispatchEvent(new MouseEvent("mouseup", { bubbles: true, cancelable: true, button: 0 }));
    element.click();
  });
export const searchSelection = (text: string) =>
  act(async () => {
    const input = document.querySelector<HTMLInputElement>('[role="combobox"]')!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
