// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode, type KeyboardEventHandler } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TPowerKCommandConfig, TPowerKContext } from "../../core/types";
import type { TPowerKCommandsListProps } from "./commands-list";
import { ProjectsAppPowerKModalWrapper } from "./wrapper";
import { ShortcutHandler } from "../../core/shortcut-handler";
import { PowerKCommandRegistry } from "../../core/registry";
const ui = vi.hoisted(() => ({
  setPage: vi.fn(),
  command: vi.fn(),
  context: vi.fn(),
  close: vi.fn(),
  retainExit: false,
}));
vi.mock("@/hooks/store/use-power-k", () => ({ usePowerK: () => ({ activePage: null, setActivePage: ui.setPage }) }));
vi.mock("@nerve/ui", () => ({
  EModalPosition: {},
  EModalWidth: {},
  ModalCore: ({ children, isOpen }: { children: ReactNode; isOpen: boolean }) =>
    isOpen || ui.retainExit ? children : null,
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
  ui.retainExit = false;
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
it("a retained closing command cannot execute an action, and reopening enables it again", async () => {
  const action = vi.fn();
  const command: TPowerKCommandConfig = {
    id: "action",
    i18n_title: "action",
    group: "general",
    type: "action",
    action,
    closeOnSelect: false,
    isEnabled: () => true,
    isVisible: () => true,
  };
  const Choices = ({ handleCommandSelect }: TPowerKCommandsListProps) => (
    <button data-testid="action" onClick={() => handleCommandSelect(command)}>
      Execute
    </button>
  );
  const show = (isOpen: boolean) =>
    act(async () =>
      root.render(
        <ProjectsAppPowerKModalWrapper
          commandsListComponent={Choices}
          context={context}
          isOpen={isOpen}
          onClose={ui.close}
        />
      )
    );
  const select = () => act(async () => (host.querySelector('[data-testid="action"]') as HTMLButtonElement).click());
  ui.retainExit = true;
  await show(true);
  await select();
  expect(action).toHaveBeenCalledTimes(1);
  action.mockClear();
  await show(false);
  await select();
  expect(action).not.toHaveBeenCalled();
  await show(true);
  await select();
  expect(action).toHaveBeenCalledTimes(1);
});
it.each(["ctrlKey", "metaKey"] as const)(
  "a closed host retained by its exit animation releases %s+K to the document opener",
  async (modifier) => {
    const reopened = vi.fn();
    const handler = new ShortcutHandler(new PowerKCommandRegistry(), () => context, reopened);
    document.addEventListener("keydown", handler.handleKeyDown);
    try {
      ui.retainExit = true;
      await render(true);
      await render(false);
      await act(async () => {
        host
          .querySelector("button")!
          .dispatchEvent(new KeyboardEvent("keydown", { key: "k", [modifier]: true, bubbles: true, cancelable: true }));
      });
      expect(reopened).toHaveBeenCalledExactlyOnceWith();
      expect(ui.close).not.toHaveBeenCalled();
    } finally {
      document.removeEventListener("keydown", handler.handleKeyDown);
      handler.destroy();
    }
  }
);
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
