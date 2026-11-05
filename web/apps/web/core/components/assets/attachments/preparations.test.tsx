// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { contentAssetOf, contentAssetTab } from "@/store/asset/fake-assets";
import { pickerDOM, waitForPickerDOM } from "../fake-image-picker";
import { DraftPreparations } from "./preparations";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof fixture>>;
async function fixture() {
  const mountedTab = await pickerDOM(await contentAssetTab());
  const view = mountedTab.app.asset.openContent(mountedTab.hosts.draft);
  return {
    ...mountedTab,
    view,
    resource: mountedTab.app.asset.draftPreparations(view),
    path: `/api/v0/drafts/${mountedTab.draft.id}/asset-preparations`,
  };
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await fixture();
  await act(async () => tab.renderer.render(<DraftPreparations resource={tab.resource} />));
});
afterEach(async () => {
  await tab?.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function open() {
  await tab.click("content_files.preparations.open");
  await act(async () => until(() => tab.nerve.to(tab.path).length === 1, "preparation read"));
  tab.nerve.to(tab.path)[0]!.answer(
    json(200, {
      data: [
        { ...contentAssetOf("pending-id", tab.view.host, "description"), state: "pending", is_bound: false },
        { ...contentAssetOf("uploaded-id", tab.view.host, "attachment"), is_bound: false },
      ],
      next_cursor: null,
    })
  );
  await waitForPickerDOM(() => document.body.textContent?.includes("pending-id.png") ?? false, "preparation rows");
}
it("only opens on demand and distinguishes native status, purpose, size and creation time", async () => {
  expect(tab.nerve.calls).toHaveLength(0);
  await open();
  const body = document.body.textContent!;
  expect(body).toContain("content_files.preparations.warning");
  expect(body).toContain("content_files.states.pending");
  expect(body).toContain("content_files.states.uploaded");
  expect(body).toContain("content_files.purposes.DRAFT_ISSUE_DESCRIPTION");
  expect(body).toContain("4 B");
  expect(document.querySelector("time")?.getAttribute("dateTime")).toBe("2026-11-01T00:00:00Z");
  expect(document.querySelector("a[href]")).toBeNull();
});
it("discards only the chosen ID after an explicit warning, and keeps bound-image refusal open", async () => {
  await open();
  await tab.click("content_files.discard pending-id.png");
  await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "discard dialog");
  expect(document.querySelector('[role="dialog"]')?.textContent).toContain(
    "content_files.preparations.confirm_warning"
  );
  await tab.click("content_files.confirm_discard");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/pending-id").length === 1, "discard"));
  tab.nerve.to("/api/v0/assets/pending-id")[0]!.answer(problem(409, "asset.bound"));
  await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "bound refusal");
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
  expect(document.querySelector('[role="alert"]')?.textContent).toBe("errors.asset_bound");
  expect(tab.resource.rows).toHaveLength(2);
  expect(tab.nerve.to("/api/v0/assets/uploaded-id")).toHaveLength(0);
});
it("a confirmed discard closes even if its list refresh fails and offers only rereading", async () => {
  await open();
  await tab.click("content_files.discard pending-id.png");
  await waitForPickerDOM(() => !!document.querySelector('[role="dialog"]'), "dialog");
  await tab.click("content_files.confirm_discard");
  await act(async () => until(() => tab.nerve.to("/api/v0/assets/pending-id").length === 1, "discard"));
  tab.nerve.to("/api/v0/assets/pending-id")[0]!.answer(noContent());
  await act(async () => until(() => tab.nerve.to(tab.path).length === 2, "refresh"));
  tab.nerve.to(tab.path)[1]!.answer(problem(503, "server_busy"));
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "confirmed closed");
  expect(tab.resource.rows.map((row) => row.id)).toEqual(["uploaded-id"]);
  await tab.click("content_files.refresh");
  await act(async () => until(() => tab.nerve.to(tab.path).length === 3, "retry"));
  expect(tab.nerve.to("/api/v0/assets/pending-id")).toHaveLength(1);
});
