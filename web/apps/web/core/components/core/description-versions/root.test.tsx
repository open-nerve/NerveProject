// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig, type Cache } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { DescriptionVersionsRoot } from "./root";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } } }));
vi.mock("./dropdown", () => ({
  DescriptionVersionsDropdown: ({
    onVersionClick,
    versions,
  }: {
    onVersionClick: (id: string) => void;
    versions: { id: string }[];
  }) => (
    <>
      {versions.map((row) => (
        <button key={row.id} data-version={row.id} onClick={() => onVersionClick(row.id)}>
          {row.id}
        </button>
      ))}
    </>
  ),
}));
vi.mock("./modal", () => ({
  DescriptionVersionsModal: ({
    activeVersionDescription,
    activeVersionDetails,
    handleClose,
    handleNavigation,
    error,
    retry,
  }: {
    activeVersionDescription?: string;
    activeVersionDetails?: { id: string };
    handleClose: () => void;
    handleNavigation: (direction: "prev" | "next") => void;
    error?: unknown;
    retry: () => Promise<unknown>;
  }) => (
    <div role="dialog">
      <span data-active>{activeVersionDetails?.id}</span>
      <span data-html>{activeVersionDescription}</span>
      <button data-close onClick={handleClose}>
        close
      </button>
      <button data-next onClick={() => handleNavigation("next")}>
        next
      </button>
      {error ? (
        <div role="alert">
          <button data-retry onClick={() => void retry().catch(() => {})}>
            retry
          </button>
        </div>
      ) : null}
    </div>
  ),
}));
let renderer: Root;
let container: HTMLElement;
const reads: { version: string; resolve: (html: string) => void; reject: (e: unknown) => void }[] = [];
const readDescription = (version: string) =>
  new Promise<string>((resolve, reject) => reads.push({ version, resolve, reject }));
const versions = ["a", "b"].map((id) => ({ id, ownerId: "actor", lastSavedAt: "2026-10-30T00:00:00Z" }));
let cache: Cache<string>;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  reads.length = 0;
  cache = new Map();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
});
function render(scopeKey = "one") {
  return act(async () =>
    renderer.render(
      <SWRConfig value={{ provider: () => cache }}>
        <DescriptionVersionsRoot
          workspaceSlug="ws"
          projectId="A"
          scopeKey={scopeKey}
          versions={versions}
          entityInformation={{
            id: "x",
            createdAt: new Date(),
            createdByDisplayName: "actor",
            isRestoreDisabled: false,
          }}
          retryList={async () => {}}
          readDescription={readDescription}
          handleRestore={async () => {}}
        />
      </SWRConfig>
    )
  );
}
const click = (selector: string) => act(async () => container.querySelector<HTMLButtonElement>(selector)!.click());
it("switching quickly to a newer choice never lets a late old description cover it", async () => {
  await render();
  await click('[data-version="a"]');
  await click("[data-next]");
  expect(reads.map((row) => row.version)).toEqual(["a", "b"]);
  await act(async () => reads[1]!.resolve("new selection"));
  expect(container.querySelector("[data-active]")?.textContent).toBe("b");
  expect(container.querySelector("[data-html]")?.textContent).toBe("new selection");
  await act(async () => reads[0]!.resolve("late old HTML"));
  expect(container.querySelector("[data-html]")?.textContent).toBe("new selection");
});
it("close clears selection immediately and an old close timer cannot clear a reopened version", async () => {
  await render();
  await click('[data-version="a"]');
  await click("[data-close]");
  expect(container.querySelector('[role="dialog"]')).toBeNull();
  await click('[data-version="b"]');
  await act(async () => reads[1]!.resolve("reopened"));
  await act(async () => reads[0]!.resolve("closed"));
  expect(container.querySelector("[data-active]")?.textContent).toBe("b");
  expect(container.querySelector("[data-html]")?.textContent).toBe("reopened");
});
it("permission identity change closes the modal and cannot reuse its late answer", async () => {
  await render();
  await click('[data-version="a"]');
  await render("after-ABA");
  expect(container.querySelector('[role="dialog"]')).toBeNull();
  await click('[data-version="a"]');
  expect(reads).toHaveLength(2);
  await act(async () => reads[0]!.resolve("old permission"));
  expect(container.querySelector("[data-html]")?.textContent).toBe("");
  await act(async () => reads[1]!.resolve("current permission"));
  expect(container.querySelector("[data-html]")?.textContent).toBe("current permission");
});
it("a refused description can be retried without restoring or closing", async () => {
  await render();
  await click('[data-version="a"]');
  await act(async () => reads[0]!.reject(new Error("unavailable")));
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  await click("[data-retry]");
  expect(reads).toHaveLength(2);
  await act(async () => reads[1]!.resolve(""));
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(container.querySelector('[role="dialog"]')).not.toBeNull();
});
