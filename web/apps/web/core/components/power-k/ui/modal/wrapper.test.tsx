// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode, type KeyboardEventHandler } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TPowerKContext } from "../../core/types";
import { ProjectsAppPowerKModalWrapper } from "./wrapper";
import { ShortcutHandler } from "../../core/shortcut-handler";
import { PowerKCommandRegistry } from "../../core/registry";
const ui = vi.hoisted(() => ({ setPage: vi.fn(), command: vi.fn(), context: vi.fn(), close: vi.fn() }));
vi.mock("@/hooks/store/use-power-k", () => ({ usePowerK: () => ({ activePage: null, setActivePage: ui.setPage }) }));
vi.mock("@nerve/ui", () => ({
  EModalPosition: {},
  EModalWidth: {},
  ModalCore: ({ children, isOpen }: { children: ReactNode; isOpen: boolean }) => (isOpen ? children : null),
}));
vi.mock("cmdk", () => ({
  Command: Object.assign(
    ({ children, onKeyDown }: { children: ReactNode; onKeyDown: KeyboardEventHandler }) => (
      <div role="dialog" aria-label="Commands" onKeyDown={onKeyDown}>
        {children}
      </div>
    ),
    {
      List: ({ children }: { children: ReactNode }) => children,
    }
  ),
}));
vi.mock("./footer", () => ({ PowerKModalFooter: () => null }));
vi.mock("./header", () => ({
  PowerKModalHeader: ({ searchTerm, onSearchChange }: { searchTerm: string; onSearchChange: (s: string) => void }) => (
    <button onClick={() => onSearchChange("fresh query")}>{searchTerm || "empty"}</button>
  ),
}));
let root: Root, host: HTMLDivElement;
const context: TPowerKContext = {
  params: { workspaceSlug: "ws" },
  activeCommand: null,
  activeContext: null,
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: ui.context,
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: ui.command,
  setActivePage: ui.setPage,
};
const Empty = () => null;
const render = (isOpen: boolean) =>
  act(async () =>
    root.render(
      <ProjectsAppPowerKModalWrapper
        commandsListComponent={Empty}
        context={context}
        isOpen={isOpen}
        onClose={ui.close}
      />
    )
  );
beforeEach(() => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.clearAllMocks();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});
it("a closed host has no delayed reset capable of erasing the newly opened query", async () => {
  await render(false);
  await act(async () => vi.advanceTimersByTimeAsync(50));
  await render(true);
  await act(async () => host.querySelector("button")!.click());
  expect(host.textContent).toBe("fresh query");
  await act(async () => vi.advanceTimersByTimeAsync(200));
  expect(host.textContent).toBe("fresh query");
});
it("closing retires the query and command before an immediate reopen", async () => {
  await render(true);
  await act(async () => host.querySelector("button")!.click());
  await render(false);
  await render(true);
  expect(host.textContent).toBe("empty");
  expect(ui.command).toHaveBeenCalledWith(null);
  expect(ui.setPage).toHaveBeenCalledWith(null);
});
it.each(["ctrlKey", "metaKey"] as const)(
  "closing with %s+K is consumed before the real document shortcut handler",
  async (modifier) => {
    const reopened = vi.fn();
    const handler = new ShortcutHandler(new PowerKCommandRegistry(), () => context, reopened);
    document.addEventListener("keydown", handler.handleKeyDown);
    try {
      await render(true);
      await act(async () => {
        host.querySelector("button")!.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "k",
            [modifier]: true,
            bubbles: true,
            cancelable: true,
          })
        );
      });
      expect(ui.close).toHaveBeenCalledExactlyOnceWith();
      expect(reopened).not.toHaveBeenCalled();
    } finally {
      document.removeEventListener("keydown", handler.handleKeyDown);
      handler.destroy();
    }
  }
);
