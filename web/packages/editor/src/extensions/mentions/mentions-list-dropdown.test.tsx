/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createRef } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TMentionSection } from "@/types";
import type { CommandListInstance } from "@/helpers/tippy";
import { MentionsListDropdown } from "./mentions-list-dropdown";
declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
function pending() {
  let resolve!: (value: TMentionSection[]) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<TMentionSection[]>((done, failed) => {
    resolve = done;
    reject = failed;
  });
  return { promise, resolve, reject };
}
const section = (title: string): TMentionSection[] => [
  { key: "users", items: [{ id: title, title, entity_identifier: title, entity_name: "user_mention", icon: null }] },
];
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
const command = vi.fn();
const onClose = vi.fn();
const search = vi.fn<(query: string) => Promise<TMentionSection[]>>();
const ref = createRef<CommandListInstance>();
beforeEach(() => {
  vi.useFakeTimers();
  command.mockClear();
  onClose.mockClear();
  search.mockReset();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});
const render = (query: string) =>
  act(async () =>
    root.render(
      <MentionsListDropdown
        ref={ref}
        query={query}
        command={command}
        searchCallback={search}
        onClose={onClose}
        renderError={(_error, retry) => (
          <div role="alert">
            Directory refused<button onClick={retry}>Retry</button>
          </div>
        )}
      />
    )
  );
const delay = () =>
  act(async () => {
    await vi.advanceTimersByTimeAsync(300);
  });
it("keeps the newer results when an older query resolves last", async () => {
  const old = pending();
  const fresh = pending();
  search.mockReturnValueOnce(old.promise).mockReturnValueOnce(fresh.promise);
  await render("old");
  await delay();
  await render("new");
  await delay();
  await act(async () => fresh.resolve(section("Newest")));
  expect(host.textContent).toContain("Newest");
  await act(async () => old.resolve(section("Stale")));
  expect(host.textContent).toContain("Newest");
  expect(host.textContent).not.toContain("Stale");
});
it("does not finish the new loading state while its query is still debouncing", async () => {
  const old = pending();
  search.mockReturnValueOnce(old.promise);
  await render("old");
  await delay();
  await render("new");
  await act(async () => old.resolve(section("Stale")));
  expect(host.textContent).toContain("Loading...");
  expect(host.textContent).not.toContain("Stale");
});
it("does not select a previous suggestion with Enter while a new query is pending", async () => {
  search.mockResolvedValueOnce(section("Old"));
  await render("old");
  await delay();
  await render("new");
  await act(async () => {
    ref.current!.onKeyDown({ event: new KeyboardEvent("keydown", { key: "Enter" }) });
  });
  expect(command).not.toHaveBeenCalled();
});
it("shows a current refusal and retries the read without presenting no results", async () => {
  search.mockRejectedValueOnce(new Error("refused")).mockResolvedValueOnce(section("Recovered"));
  await render("a");
  await delay();
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("Directory refused");
  expect(host.textContent).not.toContain("No results");
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  await delay();
  expect(search.mock.calls).toEqual([["a"], ["a"]]);
  expect(host.textContent).toContain("Recovered");
});
it("ignores failures from a previous query and from an unmounted dropdown", async () => {
  const old = pending();
  const fresh = pending();
  search.mockReturnValueOnce(old.promise).mockReturnValueOnce(fresh.promise);
  await render("old");
  await delay();
  await render("new");
  await delay();
  await act(async () => old.reject(new Error("old refused")));
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(host.textContent).toContain("Loading...");
  await act(async () => root.render(null));
  await act(async () => fresh.reject(new Error("retired refused")));
  expect(host.textContent).toBe("");
  expect(onClose).not.toHaveBeenCalled();
});

it("ends already rendered suggestions on host scope loss, including selection before React can rerender", async () => {
  let valid = true;
  let invalidate: (() => void) | undefined;
  const scope = {
    capture: () => () => valid,
    subscribe: (listener: () => void) => {
      invalidate = listener;
      return () => {
        invalidate = undefined;
      };
    },
  };
  search.mockResolvedValueOnce(section("Old scope"));
  await act(async () =>
    root.render(
      <MentionsListDropdown
        ref={ref}
        query=""
        command={command}
        searchCallback={search}
        onClose={onClose}
        scope={scope}
      />
    )
  );
  await delay();
  expect(host.textContent).toContain("Old scope");
  valid = false;
  await act(async () => ref.current?.onKeyDown({ event: new KeyboardEvent("keydown", { key: "Enter" }) }));
  expect(command).not.toHaveBeenCalled();
  await act(async () => invalidate?.());
  expect(onClose).toHaveBeenCalled();
  expect(host.textContent).not.toContain("Old scope");
});
