// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useConfirmedAutosave } from "./use-confirmed-autosave";
import type { AutosaveRetention, AutosaveSnapshot } from "./autosave-retention";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let snapshot: AutosaveSnapshot<string> | undefined;
let canFlush: boolean;
const save = vi.fn<(value: string) => Promise<void>>();
const retention: AutosaveRetention<string> = {
  read: () => snapshot,
  store: (value) => {
    snapshot = { ...value };
  },
  get canFlushOnUnmount() {
    return canFlush;
  },
};
const same = (a: string, b: string) => a === b;
function Input({ initial = "original" }: { initial?: string }) {
  const input = useConfirmedAutosave({ initial, same, save, status: () => undefined, retention });
  return (
    <>
      <input value={input.value} onChange={(event) => input.change(event.target.value)} />
      <output>{input.conflict ? "conflict" : input.needsResume ? "preserved" : "ready"}</output>
      <button onClick={input.resume}>resume</button>
      <button onClick={input.discard}>discard</button>
      {input.failure && <span role="alert">refused</span>}
    </>
  );
}
beforeEach(() => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  snapshot = undefined;
  canFlush = true;
  save.mockReset().mockResolvedValue();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});
const render = async (shown: boolean, initial = "original") =>
  act(async () => root.render(shown ? <Input initial={initial} /> : null));
const type = async (value: string) =>
  act(async () => {
    const input = host.querySelector("input")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
const click = async (text: string) =>
  act(async () => [...host.querySelectorAll("button")].find((b) => b.textContent === text)!.click());
const wait = async () => act(async () => vi.advanceTimersByTimeAsync(3000));
it("retains an unattempted input across owner verification, without flushing or resending on remount", async () => {
  await render(true);
  await type("keep local");
  canFlush = false;
  await render(false);
  expect(save).not.toHaveBeenCalled();
  canFlush = true;
  await render(true);
  await wait();
  expect(host.querySelector("input")!.value).toBe("keep local");
  expect(host.querySelector("output")!.textContent).toBe("preserved");
  expect(save).not.toHaveBeenCalled();
  await click("resume");
  expect(save).toHaveBeenCalledExactlyOnceWith("keep local");
  expect(host.querySelector("output")!.textContent).toBe("ready");
});
it("editing a preserved input on an unchanged base resumes the ordinary debounce", async () => {
  await render(true);
  await type("keep local");
  canFlush = false;
  await render(false);
  canFlush = true;
  await render(true);
  expect(host.querySelector("output")!.textContent).toBe("preserved");
  await type("keep local, amended");
  await wait();
  expect(save).toHaveBeenCalledExactlyOnceWith("keep local, amended");
  expect(host.querySelector("output")!.textContent).toBe("ready");
});
it("preserves a refused revision and never retries it merely because the body mounts", async () => {
  save.mockRejectedValueOnce(new Error("refused"));
  await render(true);
  await type("keep refusal");
  await wait();
  canFlush = false;
  await render(false);
  canFlush = true;
  await render(true);
  await wait();
  expect(host.querySelector("input")!.value).toBe("keep refusal");
  expect(host.querySelector('[role="alert"]')).not.toBeNull();
  expect(save).toHaveBeenCalledTimes(1);
  await click("resume");
  expect(save).toHaveBeenCalledTimes(2);
});
it("a changed canonical base preserves local text and requires an explicit conflict decision", async () => {
  await render(true);
  await type("mine");
  canFlush = false;
  await render(false);
  canFlush = true;
  await render(true, "theirs");
  expect(host.querySelector("input")!.value).toBe("mine");
  expect(host.querySelector("output")!.textContent).toBe("conflict");
  await type("mine amended");
  await wait();
  expect(save).not.toHaveBeenCalled();
  await click("resume");
  expect(save).toHaveBeenCalledExactlyOnceWith("mine amended");
});
it("choosing the latest canonical base discards only the local input and makes no write", async () => {
  await render(true);
  await type("mine");
  canFlush = false;
  await render(false);
  canFlush = true;
  await render(true, "theirs");
  await click("discard");
  await wait();
  expect(host.querySelector("input")!.value).toBe("theirs");
  expect(save).not.toHaveBeenCalled();
  expect(snapshot?.dirty).toBe(false);
});
it("a fresh read proving the local value is already saved removes the resume action without another write", async () => {
  await render(true);
  await type("mine");
  canFlush = false;
  await render(false);
  canFlush = true;
  await render(true, "mine");
  await wait();
  expect(host.querySelector("output")!.textContent).toBe("ready");
  expect(save).not.toHaveBeenCalled();
});
it("a later ordinary canonical update exposes a conflict before the debounce may overwrite it", async () => {
  await render(true);
  await type("mine");
  await render(true, "theirs");
  await wait();
  expect(host.querySelector("output")!.textContent).toBe("conflict");
  expect(save).not.toHaveBeenCalled();
});
