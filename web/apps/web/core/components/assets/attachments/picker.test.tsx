// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { contentAssetTab, imageFile } from "@/store/asset/fake-assets";
import { pickerDOM } from "../fake-image-picker";
import { AttachmentPicker } from "./picker";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof fixture>>;
const fixture = async () => {
  const mountedTab = await pickerDOM(await contentAssetTab());
  const view = mountedTab.app.asset.openContent(mountedTab.hosts.draft);
  return { ...mountedTab, view, upload: mountedTab.app.asset.createContentUpload(view, "attachment") };
};
const saved = vi.fn(async () => {});
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await fixture();
  saved.mockReset();
  await act(async () => tab.renderer.render(<AttachmentPicker upload={tab.upload} onUploaded={saved} />));
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const alert = () => document.querySelector<HTMLElement>('[role="alert"]');
it("accepts a native arbitrary file without preview, shows its original name and sends one exact chain", async () => {
  const file = imageFile("my.report.pdf", 4, "application/pdf");
  await tab.select([file]);
  expect(tab.create).not.toHaveBeenCalled();
  expect(document.body.textContent).toContain(file.name);
  expect(document.body.textContent).toContain("4 B");
  await act(async () => {
    tab.button("content_files.upload").click();
    tab.button("content_files.upload").click();
  });
  await act(async () => until(() => tab.puts.length === 1, "upload"));
  expect(tab.puts[0]!.file).toBe(file);
  expect(document.querySelector('[aria-live="polite"]')?.textContent).toBe("content_files.phases.uploading");
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => saved.mock.calls.length === 1, "one completion"));
  expect(tab.nerve.calls.map((call) => call.path)).toEqual(["/api/v0/assets", "/api/v0/assets/asset-1/complete"]);
  expect(saved).toHaveBeenCalledTimes(1);
  expect(tab.button("content_files.upload").disabled).toBe(true);
  expect(document.activeElement).toBe(document.querySelector('input[type="file"]'));
});
it.each(["multiple", "directory", "text"])(
  "rejects a %s drop with an accessible focused error and zero requests",
  async (kind) => {
    const file = imageFile("a.txt", 4, "text/plain");
    const event = new Event("drop", { bubbles: true, cancelable: true });
    Object.defineProperty(event, "dataTransfer", {
      value: {
        files: kind === "text" ? [] : kind === "multiple" ? [file, file] : [file],
        items: [
          {
            kind: kind === "text" ? "string" : "file",
            webkitGetAsEntry: () => ({ isDirectory: kind === "directory" }),
          },
        ],
      },
    });
    await act(async () => document.querySelector('[aria-label="content_files.drop"]')!.dispatchEvent(event));
    expect(alert()).not.toBeNull();
    expect(document.activeElement).toBe(alert());
    expect(tab.nerve.calls).toHaveLength(0);
  }
);
it("cancel stops PUT and never confirms or announces a late success", async () => {
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "PUT"));
  await tab.click("content_files.cancel_upload");
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  expect(tab.nerve.calls).toHaveLength(1);
  expect(saved).not.toHaveBeenCalled();
  expect(document.activeElement).toBe(document.querySelector('input[type="file"]'));
});
it("uncertain transfer checks the same preparation without another PUT, then missing object needs explicit new upload", async () => {
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "PUT"));
  await act(async () => tab.puts[0]!.fail());
  await tab.click("content_files.check_upload");
  await act(async () => until(() => saved.mock.calls.length === 1, "confirmation"));
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.calls).toHaveLength(2);
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 2, "second PUT"));
  await act(async () => tab.puts[1]!.fail());
  tab.nerve.respond = (call) =>
    call.path.endsWith("/complete") ? call.answer(problem(409, "asset.object_missing")) : tab.answer(call);
  await tab.click("content_files.check_upload");
  await act(async () => until(() => tab.upload.recovery === "restart", "missing object"));
  expect(alert()?.textContent).toBe("errors.asset_object_missing");
  tab.nerve.respond = tab.answer;
  await tab.click("content_files.upload_again");
  await act(async () => until(() => tab.puts.length === 3, "explicit restart"));
  expect(tab.nerve.calls.filter((call) => call.path === "/api/v0/assets")).toHaveLength(3);
});
it("confirmation refresh failure is already successful and has no second upload action", async () => {
  saved.mockImplementation(async () => {
    /* AttachmentResource owns its separate read error. */
  });
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "PUT"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => saved.mock.calls.length === 1, "confirmed upload"));
  expect(tab.button("content_files.upload").disabled).toBe(true);
});
it("retirement after complete reached the client cannot call the old page", async () => {
  let release: ((response: Response) => void) | undefined;
  let held: Response | undefined;
  tab.api.use({
    onResponse({ request, response }) {
      if (new URL(request.url).pathname.endsWith("/complete")) {
        held = response;
        return new Promise<Response>((resolve) => {
          release = resolve;
        });
      }
      return response;
    },
  });
  await tab.select();
  await tab.click("content_files.upload");
  await act(async () => until(() => tab.puts.length === 1, "PUT"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => !!held, "held completion"));
  await act(async () => {
    tab.view.dispose();
    release!(held!);
  });
  expect(saved).not.toHaveBeenCalled();
  expect(tab.upload.phase).toBe("disposed");
});
it("a malformed selection name is rejected before any service call", async () => {
  await tab.select([imageFile("bad/path.txt", 4, "text/plain")]);
  expect(alert()?.textContent).toBe("content_files.errors.name");
  expect(tab.nerve.calls).toHaveLength(0);
});
