// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { attachmentOf, contentAssetOf, contentAssetTab } from "@/store/asset/fake-assets";
import { pickerDOM, waitForPickerDOM } from "../fake-image-picker";
import { Attachments } from "./root";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof fixture>>;
async function fixture() {
  const mountedTab = await pickerDOM(await contentAssetTab());
  const view = mountedTab.app.asset.openContent(mountedTab.hosts.draft);
  return {
    ...mountedTab,
    view,
    resource: mountedTab.app.asset.attachments(view),
    path: `/api/v0/drafts/${mountedTab.draft.id}/attachments`,
  };
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await fixture();
});
afterEach(async () => {
  await tab?.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function render() {
  await act(async () =>
    tab.renderer.render(
      <StoreContext.Provider value={tab.app}>
        <Attachments view={tab.view} resource={tab.resource} />
      </StoreContext.Provider>
    )
  );
  await act(async () => until(() => tab.nerve.to(tab.path).length === 1, "attachment read"));
  tab.nerve.to(tab.path)[0]!.answer(json(200, { data: [attachmentOf("file-a")], next_cursor: null }));
  await waitForPickerDOM(() => document.body.textContent?.includes("file-a.txt") ?? false, "attachment rendered");
}
async function open() {
  await tab.click("content_files.remove file-a.txt");
  await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "delete dialog");
}
const escape = () =>
  act(async () =>
    document.activeElement!.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
it("only reads the first page on mount, shows native metadata with neutral controls and no permanent capability links", async () => {
  await render();
  expect(document.body.textContent).toContain("4 B");
  expect(document.querySelector("button button")).toBeNull();
  expect(document.querySelector("a[href]")).toBeNull();
  expect(document.querySelector("img")).toBeNull();
  expect(tab.nerve.calls).toHaveLength(1);
});
it("rejected deletion stays open with its row and a focused error; another explicit attempt can succeed", async () => {
  await render();
  await open();
  await tab.click("content_files.confirm_delete");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
  tab.nerve.to("/api/v0/assets/file-a")[0]!.answer(problem(403, "forbidden"));
  await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "refusal");
  expect(document.activeElement).toBe(document.querySelector('[role="alert"]'));
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  expect(tab.resource.rows).toHaveLength(1);
  await tab.click("content_files.confirm_delete");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 2, "second explicit delete"));
  tab.nerve.to("/api/v0/assets/file-a")[1]!.answer(noContent());
  await act(async () => until(() => tab.nerve.to(tab.path).length === 2, "refresh"));
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [], next_cursor: null }));
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "closed");
  expect(document.activeElement).toBe(tab.button("content_files.refresh"));
  expect(tab.resource.rows).toEqual([]);
});
it("busy deletion rejects Escape/backdrop and double submit, then refresh failure stays a read-only action", async () => {
  await render();
  await open();
  await act(async () => {
    tab.button("content_files.confirm_delete").click();
    tab.button("content_files.confirm_delete").click();
  });
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "one delete"));
  await escape();
  const backdrop = document.querySelector<HTMLElement>('[data-headlessui-state][class*="bg-backdrop"]')!;
  await act(async () => {
    backdrop.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    backdrop.click();
  });
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  tab.nerve.to("/api/v0/assets/file-a")[0]!.answer(noContent());
  await act(async () => until(() => tab.nerve.to(tab.path).length === 2, "refresh"));
  tab.nerve.to(tab.path)[1]!.answer(problem(503, "server_busy"));
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "confirmed closed");
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  await tab.click("content_files.refresh");
  await act(async () => until(() => tab.nerve.to(tab.path).length === 3, "read retry"));
  expect(tab.nerve.to("/api/v0/assets/file-a")).toHaveLength(1);
});
it("download starts only on click, shares double clicks and releases its object URL", async () => {
  await render();
  const anchor = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/assets/file-a")
      call.answer(json(200, { ...contentAssetOf("file-a", tab.view.host, "attachment"), name: "file-a.txt" }));
    else if (call.path.endsWith("/content"))
      call.answer(
        json(200, {
          url: "/assets/storage/local/file-a?grant=private",
          expires_at: new Date(Date.now() + 60_000).toISOString(),
        })
      );
  };
  await act(async () => {
    tab.button("content_files.download file-a.txt").click();
    tab.button("content_files.download file-a.txt").click();
  });
  await act(async () => until(() => tab.puts.length === 1, "raw GET"));
  await act(async () => tab.puts[0]!.answer(new Response(new Uint8Array(4))));
  await waitForPickerDOM(() => anchor.mock.calls.length === 1, "download");
  expect(tab.create).toHaveBeenCalledTimes(1);
  expect(tab.revoke).toHaveBeenCalledTimes(1);
  expect(tab.puts).toHaveLength(1);
});
it("view retirement closes the dialog and old deletion cannot announce into the next page", async () => {
  await render();
  await open();
  await tab.click("content_files.confirm_delete");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
  await act(async () => {
    tab.view.dispose();
    tab.renderer.render(<button type="button">next</button>);
  });
  const next = tab.button("next");
  await act(async () => next.focus());
  tab.nerve.to("/api/v0/assets/file-a")[0]!.answer(noContent());
  await act(async () => vi.advanceTimersByTimeAsync(20));
  expect(document.activeElement).toBe(next);
  expect(tab.nerve.to(tab.path)).toHaveLength(1);
});
it("an ordinary attachment upload confirms once and a failed list refresh can only be reread", async () => {
  await render();
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/complete"))
      call.answer(
        json(200, { ...tab.assets.get("asset-1"), state: "uploaded", is_bound: true, width: null, height: null })
      );
    else if (call.path === tab.path) call.answer(problem(503, "server_busy"));
    else tab.answer(call);
  };
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "original transfer"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await waitForPickerDOM(
    () => tab.nerve.to(tab.path).length === 2 && !!document.querySelector('[role="alert"]'),
    "read refusal"
  );
  expect(tab.button("content_files.upload").disabled).toBe(true);
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(1);
  tab.nerve.respond = (call) => {
    if (call.path === tab.path) call.answer(json(200, { data: [attachmentOf("asset-1")], next_cursor: null }));
  };
  await tab.click("content_files.refresh");
  await waitForPickerDOM(() => document.body.textContent?.includes("asset-1.txt") ?? false, "fresh list");
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.calls.filter((call) => call.method === "POST")).toHaveLength(2);
});
it("an unknown deletion forces read-only verification before a second explicit attempt", async () => {
  await render();
  await open();
  await tab.click("content_files.confirm_delete");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 1, "delete"));
  tab.nerve.to("/api/v0/assets/file-a")[0]!.fail();
  await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "unknown");
  expect(tab.button("content_files.confirm_delete").disabled).toBe(true);
  expect(document.querySelector('[role="dialog"]')?.textContent).toContain("content_files.unknown_delete");
  await tab.click("content_files.verify");
  await act(async () => until(() => tab.nerve.to(tab.path).length === 2, "verify list"));
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [], next_cursor: null }));
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/file-a").length === 2, "verify ID"));
  tab.nerve.to("/api/v0/assets/file-a")[1]!.answer(json(200, contentAssetOf("file-a", tab.view.host, "attachment")));
  await waitForPickerDOM(() => !tab.button("content_files.confirm_delete").disabled, "explicit choice enabled");
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  expect(tab.nerve.to("/api/v0/assets/file-a").map((call) => call.method)).toEqual(["DELETE", "GET"]);
});
