// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode } from "react";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import type { ContentHost } from "@/store/asset/content-view";
import { StoreContext } from "@/lib/store-context";
import { until } from "@/lib/auth/fake-time";
import { contentAssetTab } from "@/store/asset/fake-assets";
import { pickerDOM } from "@/components/assets/fake-image-picker";
import { AttachmentPicker } from "@/components/assets/attachments/picker";
import { useContentUpload, useContentView } from "./use-content-assets";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof fixture>>;
const fixture = async () => pickerDOM(await contentAssetTab());
const saved = vi.fn(async () => {});
const Host = observer(function Host({ host }: { host: ContentHost | undefined }) {
  const view = useContentView(host);
  const upload = useContentUpload(view, "attachment");
  return upload ? (
    <AttachmentPicker key={JSON.stringify(host)} upload={upload} onUploaded={saved} />
  ) : (
    <p>Unavailable</p>
  );
});
const render = (host: ContentHost | undefined) =>
  act(async () =>
    tab.renderer.render(
      <StrictMode>
        <StoreContext.Provider value={tab.app}>
          <Host host={host} />
        </StoreContext.Provider>
      </StrictMode>
    )
  );
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await fixture();
  saved.mockReset();
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
it("StrictMode and host changes retire the old transfer and only the new host can publish", async () => {
  await render(tab.hosts.draft);
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "draft PUT"));
  await render(tab.hosts.new_comment);
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  expect(saved).not.toHaveBeenCalled();
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 2, "comment PUT"));
  await act(async () => tab.puts[1]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => saved.mock.calls.length === 1, "current result"));
  expect(tab.nerve.calls.filter((call) => call.path === "/api/v0/assets").map((call) => call.body)).toMatchObject([
    { scope: { kind: "draft", id: tab.draft.id } },
    { scope: { kind: "new_comment", id: tab.issue.id } },
  ]);
});
it("a missing canonical owner produces no upload and unmount ends every active chain", async () => {
  await render({ ...tab.hosts.issue, issueID: "unread" });
  expect(document.querySelector('input[type="file"]')).toBeNull();
  expect(tab.nerve.calls).toHaveLength(0);
  await render(tab.hosts.draft);
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "PUT"));
  await render(undefined);
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
});
