// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TNameDescriptionLoader } from "@nerve/types";
import { ApiError } from "@/lib/api-error";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { IssueTitleInput } from "./title-input";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let mounted: boolean;
let requests: HeldChange<void>[];
const persist = vi.fn<(name: string) => Promise<void>>();
const control = () => host.querySelector("textarea")!;
const away = () => host.querySelector<HTMLButtonElement>("[data-away]")!;
function description() {
  const ids = control().getAttribute("aria-describedby")?.split(/\s+/) ?? [];
  return ids.map((id) => document.getElementById(id)?.textContent ?? "").join(" ");
}
function focuses() {
  const taken: { invalid: string | null; description: string }[] = [];
  control().addEventListener("focus", () =>
    taken.push({ invalid: control().getAttribute("aria-invalid"), description: description() })
  );
  return taken;
}
function Harness() {
  const [status, setStatus] = useState<TNameDescriptionLoader>("saved");
  return (
    <>
      <output>{status}</output>
      <IssueTitleInput
        workspaceSlug="ws"
        projectId="A"
        issueId="one"
        value="Original"
        setIsSubmitting={setStatus}
        issueOperations={{
          update: (_ws, _project, _id, patch) => persist(patch.name),
          prepareUpdate: () => (_ws, _project, _id, patch) => persist(patch.name),
        }}
      />
      <button data-away>Another control</button>
    </>
  );
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  requests = [];
  persist.mockReset().mockImplementation(() => {
    const held = heldChange<void>();
    requests.push(held);
    return held.sent;
  });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  mounted = true;
  await act(async () => root.render(<Harness />));
});
afterEach(async () => {
  if (mounted) await act(async () => root.unmount());
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function type(value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(control(), value);
    control().dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const debounce = () =>
  act(async () => {
    await vi.advanceTimersByTimeAsync(1500);
  });
const rejectedName = () =>
  new ApiError(422, {
    status: 422,
    code: "validation_failed",
    title: "Validation failed",
    errors: [{ field: "name", code: "invalid_format", message: "invalid title" }],
  });

it.each([
  ["   ", "form.title.required"],
  ["x".repeat(256), "errors.field.too_long"],
])("announcesRefusalBeforeFocus for a local invalid title (%s)", async (value, message) => {
  await type(value);
  await act(async () => away().focus());
  const taken = focuses();
  await debounce();
  expect(persist).not.toHaveBeenCalled();
  expect(control().value).toBe(value);
  expect(document.activeElement).toBe(control());
  expect(taken).toEqual([{ invalid: "true", description: message }]);
});
it("announcesRefusalBeforeFocus for the server's name field, and retries the retained value", async () => {
  await type("Retained title");
  await debounce();
  await act(async () => away().focus());
  const taken = focuses();
  await act(async () => requests[0]!.refuse(rejectedName()));
  expect(control().value).toBe("Retained title");
  expect(document.activeElement).toBe(control());
  expect(taken).toEqual([{ invalid: "true", description: "errors.field.invalid_format" }]);
  await act(async () =>
    [...host.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!.click()
  );
  expect(persist.mock.calls).toEqual([["Retained title"], ["Retained title"]]);
  await act(async () => requests[1]!.answer());
  expect(control().getAttribute("aria-invalid")).not.toBe("true");
  expect(host.querySelector("output")?.textContent).toBe("submitted");
});
it("doesNotMoveFocusWhileTyping after a refusal, and clears the obsolete field error", async () => {
  await type("Refused");
  await debounce();
  await act(async () => requests[0]!.refuse(rejectedName()));
  await act(async () => away().focus());
  const taken = focuses();
  await type("Fixed");
  expect(document.activeElement).toBe(away());
  expect(taken).toEqual([]);
  expect(control().getAttribute("aria-invalid")).not.toBe("true");
  expect(description()).toBe("");
  await debounce();
  await act(async () => requests[1]!.answer());
});
it("refuses an invalid pending title on unmount instead of flushing it through prepareUpdate", async () => {
  await type("   ");
  await act(async () => root.unmount());
  mounted = false;
  expect(persist).not.toHaveBeenCalled();
});
it("accepts 255 Unicode characters rather than counting their UTF-16 code units", async () => {
  const title = "😀".repeat(255);
  await type(title);
  await debounce();
  expect(persist).toHaveBeenCalledExactlyOnceWith(title);
  await act(async () => requests[0]!.answer());
  expect(control().getAttribute("aria-invalid")).not.toBe("true");
  expect(host.querySelector("output")?.textContent).toBe("submitted");
});
