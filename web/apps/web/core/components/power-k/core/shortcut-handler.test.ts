// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { PowerKCommandRegistry } from "./registry";
import { ShortcutHandler } from "./shortcut-handler";
import type { TPowerKContext } from "./types";

const consume = (event: KeyboardEvent) => {
  event.preventDefault();
  event.stopPropagation();
};

it.each(["ctrlKey", "metaKey"] as const)(
  "%s+K consumed by the document-root host cannot reopen the palette; a later free key still opens it",
  (modifier) => {
    let opened = 0;
    const context: TPowerKContext = {
      params: { workspaceSlug: "ws" },
      activeCommand: null,
      activeContext: null,
      shouldShowContextBasedActions: true,
      setShouldShowContextBasedActions: () => {},
      navigate: () => {},
      closePalette: () => {},
      setActiveCommand: () => {},
      setActivePage: () => {},
    };
    const handler = new ShortcutHandler(
      new PowerKCommandRegistry(),
      () => context,
      () => {
        opened++;
      }
    );
    // Production hydrates at document: stopping propagation there cannot stop another document listener.
    const press = (details: KeyboardEventInit = {}) =>
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: "k", [modifier]: true, bubbles: true, cancelable: true, ...details })
      );
    document.addEventListener("keydown", consume);
    document.addEventListener("keydown", handler.handleKeyDown);
    try {
      press();
      expect(opened).toBe(0);
      document.removeEventListener("keydown", consume);
      press({ isComposing: true });
      press({ keyCode: 229 });
      expect(opened).toBe(0);
      press();
      expect(opened).toBe(1);
    } finally {
      document.removeEventListener("keydown", consume);
      document.removeEventListener("keydown", handler.handleKeyDown);
      handler.destroy();
    }
  }
);
