// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderFormattedDate } from "@nerve/utils";
import { refusal } from "@/lib/fake-refusal";
import { InboxIssueSnoozeModal } from "./snooze-issue-modal";

// The snooze modal of a work item of a project's intake (P5a spec 2.8, R13): named by its title (ModalTitle); a day of
// the caller's calendar, tomorrow at the earliest and at first, or the day of the snooze held while that is tomorrow
// or later; it sends that day's first moment in the browser's zone (snoozeInstant) and waits for nerve, closed by
// neither Escape nor Cancel meanwhile; nerve's refusal of the moment (must_be_future) shows under the day, which takes
// the focus, and any other is the action model's to say. The dialog is real (Headless UI); its submission is the
// test's. @nerve/i18n's text is its key.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: { start_of_the_week: 1 } }) }));

type Submit = (snoozedTill: string, underField: (error: unknown) => boolean) => Promise<void>;
const submit = vi.fn<Submit>();
const close = vi.fn();
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  // 25 November 2026, 15:10 in the browser's zone
  vi.useFakeTimers({ now: new Date(2026, 10, 25, 15, 10) });
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  submit.mockReset().mockResolvedValue(undefined);
  close.mockReset();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(400)));
const render = async (value: string | null = null) => {
  await act(async () => root.render(<InboxIssueSnoozeModal value={value} onClose={close} onSubmit={submit} />));
  await settled();
};
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');
const dateButton = () =>
  dialog()?.querySelector<HTMLButtonElement>('button[aria-label="inbox_issue.modals.snooze.day"]') ?? null;
const buttonSaying = (text: string) =>
  [...(dialog()?.querySelectorAll("button") ?? [])].find((button) => button.textContent === text);
const day = (date: string) => document.querySelector<HTMLButtonElement>(`[data-day="${date}"] button`);
const describedBy = (control: HTMLElement) =>
  document.getElementById(control.getAttribute("aria-describedby") ?? "")?.textContent;
const escape = () =>
  act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
async function snooze() {
  await act(async () => buttonSaying("inbox_issue.actions.snooze")!.click());
  await settled();
}
async function openCalendar() {
  await act(async () => dateButton()!.click());
  await settled();
}

describe("the snooze modal (R13)", () => {
  it("is named by its title and starts at tomorrow, its earliest day: today is no choice", async () => {
    await render();
    expect(document.getElementById(dialog()!.getAttribute("aria-labelledby")!)?.textContent).toBe(
      "inbox_issue.modals.snooze.title"
    );
    expect(dateButton()?.textContent).toContain(renderFormattedDate(new Date(2026, 10, 26)));
    await openCalendar();
    expect(day("2026-11-25")?.disabled).toBe(true);
    expect(day("2026-11-26")?.disabled).toBe(false);
  });

  it("sends tomorrow's first moment in the browser's zone, unless another day is chosen", async () => {
    await render();
    await snooze();
    expect(submit).toHaveBeenCalledExactlyOnceWith(new Date(2026, 10, 26).toISOString(), expect.any(Function));
    submit.mockClear();
    await openCalendar();
    await act(async () => day("2026-11-28")!.click());
    await settled();
    await snooze();
    expect(submit).toHaveBeenCalledExactlyOnceWith(new Date(2026, 10, 28).toISOString(), expect.any(Function));
  });

  it("starts at the held snooze's day while it is tomorrow or later, at tomorrow once it is not", async () => {
    await render(new Date(2026, 11, 2, 9, 30).toISOString());
    expect(dateButton()?.textContent).toContain(renderFormattedDate(new Date(2026, 11, 2)));
    await snooze();
    expect(submit).toHaveBeenCalledExactlyOnceWith(new Date(2026, 11, 2).toISOString(), expect.any(Function));
    await act(async () => root.unmount());
    root = createRoot(host);
    await render(new Date(2026, 10, 25, 23, 0).toISOString());
    expect(dateButton()?.textContent).toContain(renderFormattedDate(new Date(2026, 10, 26)));
  });

  it("shows nerve's must_be_future under the day, which takes the focus, the dialog open", async () => {
    const taken: boolean[] = [];
    submit.mockImplementation(async (_, underField) => {
      taken.push(underField(refusal(422, "validation_failed", [{ field: "snoozed_till", code: "must_be_future" }])));
      throw refusal(422, "validation_failed");
    });
    await render();
    await snooze();
    expect(taken).toEqual([true]);
    expect(dateButton()?.getAttribute("aria-invalid")).toBe("true");
    expect(describedBy(dateButton()!)).toBe("errors.field.must_be_future");
    expect(document.activeElement).toBe(dateButton());
    expect(close).not.toHaveBeenCalled();
  });

  it("leaves any other refusal to the action model, nothing under the day", async () => {
    const taken: boolean[] = [];
    submit.mockImplementation(async (_, underField) => {
      taken.push(underField(refusal(403, "forbidden")));
      throw refusal(403, "forbidden");
    });
    await render();
    await snooze();
    expect(taken).toEqual([false]);
    expect(dateButton()?.hasAttribute("aria-invalid")).toBe(false);
    expect(dialog()).not.toBeNull();
  });

  it("cannot be closed while nerve snoozes, by Escape or Cancel; once refused, Cancel closes it", async () => {
    let refuse: ((error: unknown) => void) | undefined;
    submit.mockImplementation(() => new Promise((_, reject) => (refuse = reject)));
    await render();
    await snooze();
    await escape();
    expect(buttonSaying("common.cancel")?.disabled).toBe(true);
    expect(close).not.toHaveBeenCalled();
    await act(async () => refuse!(refusal(503, "server_busy")));
    await settled();
    await act(async () => buttonSaying("common.cancel")!.click());
    expect(close).toHaveBeenCalledOnce();
  });
});
