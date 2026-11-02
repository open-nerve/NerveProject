// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { Toast, setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { EmojiReactionGroup } from "@nerve/propel/emoji-reaction";
import { Tooltip } from "@nerve/propel/tooltip";

let root: Root;
let host: HTMLElement;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  document.body.replaceChildren();
  vi.unstubAllGlobals();
});

it.each(["Close", "关闭"])("names the toast close button as %s and lets it dismiss the message", async (closeLabel) => {
  const props = { theme: "light" as const, label: "Notifications", closeLabel };
  await act(async () => root.render(<Toast {...props} />));
  await act(async () => {
    setToast({ type: TOAST_TYPE.SUCCESS, title: "Saved feedback" });
  });
  const button = document.body.querySelector<HTMLButtonElement>("button")!;
  expect(button).not.toBeNull();
  expect(button.getAttribute("aria-label")).toBe(closeLabel);
  await act(async () => button.click());
  await vi.waitFor(() => expect(document.body.textContent).not.toContain("Saved feedback"));
});

it("a reaction reports users beyond the display limit", async () => {
  await act(async () =>
    root.render(
      <EmojiReactionGroup
        reactions={[{ emoji: "👍", count: 6, reacted: true, users: ["Ada", "Bo", "Cal", "Di", "Eve", "Fay"] }]}
        maxDisplayUsers={2}
        addReactionLabel="Add reaction"
        showAddButton={false}
      />
    )
  );
  const button = host.querySelector("button")!;
  expect(button).not.toBeNull();
  await act(async () => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", bubbles: true }));
    button.focus();
  });
  await vi.waitFor(() => expect(document.body.textContent).toContain("Ada, Bo"));
  expect(document.body.textContent).toContain("Ada, Bo and 4 more");
});

it.each([false, true])("a reaction exposes reacted=%s as its selected state", async (reacted) => {
  await act(async () =>
    root.render(
      <EmojiReactionGroup
        reactions={[{ emoji: "👍", count: 1, reacted }]}
        addReactionLabel="Add reaction"
        showAddButton={false}
      />
    )
  );
  expect(host.querySelector("button")?.getAttribute("aria-pressed")).toBe(String(reacted));
});

it("keeps the tooltip popup role separate from its positioner and accepts block content", async () => {
  await act(async () =>
    root.render(
      <Tooltip tooltipContent={<div>Reaction people</div>}>
        <button type="button" aria-describedby="existing-description">
          Reaction
        </button>
      </Tooltip>
    )
  );
  await act(async () => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", bubbles: true }));
    host.querySelector("button")!.focus();
  });
  await vi.waitFor(() => expect(document.body.textContent).toContain("Reaction people"));
  const popup = document.querySelector('[role="tooltip"]');
  expect(popup?.textContent).toBe("Reaction people");
  expect(popup?.id).toBeTruthy();
  expect(host.querySelector("button")?.getAttribute("aria-describedby")).toContain(popup!.id);
  expect(host.querySelector("button")?.getAttribute("aria-describedby")).toContain("existing-description");
  expect(document.querySelector("p div")).toBeNull();
});
