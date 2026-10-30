// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ModalCore, ModalTitle } from "@nerve/ui";
import { CreateApiTokenForm } from "@/components/api-token/modal/form";
import { DateDropdown } from "./date";
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: { start_of_the_week: 1 } }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const changed = vi.fn(),
  closed = vi.fn();
beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date(2026, 9, 30, 12));
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  changed.mockClear();
  closed.mockClear();
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
const popup = () =>
  document.querySelector('[role="dialog"][aria-label="Due date"]') ?? document.querySelector('[role="listbox"]');
const trigger = () =>
  (host.querySelector<HTMLButtonElement>('button[aria-label="Due date"]') ??
    host.querySelector<HTMLButtonElement>("button"))!;
async function render(props: Partial<ComponentProps<typeof DateDropdown>> = {}) {
  await act(async () =>
    root.render(
      <>
        <DateDropdown
          buttonVariant="border-with-text"
          placeholder="Due date"
          value="2026-10-30"
          onChange={changed}
          onClose={closed}
          {...props}
        />
        <button>After</button>
      </>
    )
  );
}
async function press(element: Element, key: string, composing = false) {
  await act(async () => {
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, isComposing: composing });
    element.dispatchEvent(down);
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true, isComposing: composing });
    element.dispatchEvent(up);
    // jsdom does not perform a native button's keyboard default action. The browser stories use real keypresses.
    if (
      !composing &&
      !down.defaultPrevented &&
      !up.defaultPrevented &&
      (key === "Enter" || key === " ") &&
      element instanceof HTMLButtonElement
    )
      element.click();
  });
}

const day = (n: string) => document.querySelector<HTMLButtonElement>(`[data-day="${n}"] button`)!;
it.each(["Enter", " "])(
  "a lazy date trigger opens by %s without pointer preparation, then Escape restores it",
  async (key) => {
    await render({ renderByDefault: false });
    expect(host.querySelector("button button")).toBeNull();
    expect(popup()).toBeNull();
    expect(trigger()).not.toBeNull();
    await act(async () => trigger().focus());
    await press(trigger(), key);
    expect(popup()).not.toBeNull();
    expect(trigger().getAttribute("aria-expanded")).toBe("true");
    await press(day("2026-10-30"), "Escape", true);
    expect(popup()).not.toBeNull();
    await press(day("2026-10-30"), "Escape");
    await act(async () => vi.advanceTimersByTimeAsync(50));
    expect(popup()).toBeNull();
    expect(document.activeElement).toBe(trigger());
    expect(closed).toHaveBeenCalledTimes(1);
    await press(trigger(), key);
    expect(popup()).not.toBeNull();
  }
);
it("selection closes the one popup and next opening is immediately usable", async () => {
  await render();
  await act(async () => {
    trigger().focus();
    trigger().click();
  });
  await act(async () => day("2026-10-31").click());
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(changed).toHaveBeenCalledTimes(1);
  expect(changed.mock.calls[0]![0]).toEqual(new Date(2026, 9, 31));
  expect(popup()).toBeNull();
  expect(closed).toHaveBeenCalledTimes(1);
  await act(async () => {
    trigger().focus();
    trigger().click();
  });
  expect(popup()).not.toBeNull();
});
it("defaultOpen, disabled days, disabled trigger and closeOnSelect retain their public contracts", async () => {
  await render({
    defaultOpen: true,
    minDate: new Date(2026, 9, 30),
    maxDate: new Date(2026, 9, 31),
    closeOnSelect: false,
  });
  expect(popup()).not.toBeNull();
  expect(day("2026-10-29").disabled).toBe(true);
  expect(day("2026-11-01").disabled).toBe(true);
  await act(async () => day("2026-10-31").click());
  expect(changed).toHaveBeenCalledTimes(1);
  expect(popup()).not.toBeNull();
  await render({ disabled: true });
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(trigger().disabled).toBe(true);
  expect(popup()).toBeNull();
});
it("pointer dismissal and focus leaving the calendar close its only state", async () => {
  await render();
  await act(async () => {
    trigger().focus();
    trigger().click();
  });
  const after = [...host.querySelectorAll("button")].find((node) => node.textContent === "After")!;
  await act(async () => {
    after.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    after.click();
    after.focus();
  });
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(popup()).toBeNull();
  await act(async () => {
    trigger().focus();
    trigger().click();
  });
  expect(popup()).not.toBeNull();
  await act(async () => after.focus());
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(popup()).toBeNull();
});
it("opening the adjacent picker dismisses the previous calendar instead of leaving a second panel", async () => {
  await act(async () =>
    root.render(
      <>
        <DateDropdown buttonVariant="border-with-text" placeholder="Due date" value={null} onChange={changed} />
        <DateDropdown buttonVariant="border-with-text" placeholder="Start date" value={null} onChange={changed} />
      </>
    )
  );
  await act(async () => {
    trigger().focus();
    trigger().click();
  });
  expect(popup()).not.toBeNull();
  const other = (host.querySelector<HTMLButtonElement>('button[aria-label="Start date"]') ??
    host.querySelectorAll<HTMLButtonElement>("button:not(button button)")[1])!;
  await act(async () => {
    other.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    other.click();
  });
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(popup()).toBeNull();
  expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(1);
});
function InsideModal() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button onClick={() => setOpen(true)}>Edit</button>
      <ModalCore isOpen={open} handleClose={() => setOpen(false)}>
        <ModalTitle>Edit dates</ModalTitle>
        <DateDropdown buttonVariant="border-with-text" placeholder="Due date" value="2026-10-30" onChange={changed} />
      </ModalCore>
    </>
  );
}
it("a calendar inside ModalCore stays usable and does not release the outer background", async () => {
  await act(async () => root.render(<InsideModal />));
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  const dateTrigger = (document.querySelector<HTMLButtonElement>('button[aria-label="Due date"]') ??
    document.querySelector<HTMLButtonElement>('[role="dialog"][aria-modal="true"] button'))!;
  await act(async () => dateTrigger.click());
  expect(host.getAttribute("aria-hidden")).toBe("true");
  await act(async () => day("2026-10-31").click());
  expect(changed).toHaveBeenCalledTimes(1);
  expect(host.getAttribute("aria-hidden")).toBe("true");
  expect(document.querySelector('[role="dialog"][aria-modal="true"]')).not.toBeNull();
});

it("the single native trigger has a name and the calendar is not a listbox", async () => {
  await render();
  expect(trigger().getAttribute("aria-label")).toBe("Due date");
  await act(async () => {
    trigger().focus();
    trigger().click();
  });
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.querySelector('[role="dialog"][aria-label="Due date"]')).not.toBeNull();
});

it("clearing is a named keyboard button that emits once without opening the calendar", async () => {
  await render();
  expect(document.activeElement).not.toBe(trigger());
  const clear = host.querySelector<HTMLButtonElement>('button[aria-label="common.clear_date: Due date"]')!;
  await act(async () => clear.focus());
  await press(clear, " ");
  expect(changed.mock.calls).toEqual([[null]]);
  expect(popup()).toBeNull();
  expect(closed).not.toHaveBeenCalled();
});

it("the real API token form keeps tomorrow as its minimum and submits the custom day at the current local time", async () => {
  const submitted = vi.fn<ComponentProps<typeof CreateApiTokenForm>["onSubmit"]>(async () => undefined);
  await act(async () =>
    root.render(
      <ModalCore isOpen handleClose={() => undefined}>
        <CreateApiTokenForm
          handleClose={() => undefined}
          neverExpires={false}
          toggleNeverExpires={() => undefined}
          onSubmit={submitted}
        />
      </ModalCore>
    )
  );
  const input = document.querySelector<HTMLInputElement>('input[aria-label="title"]')!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "deploy");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  const expiry = [...document.querySelectorAll<HTMLButtonElement>("button")].find((b) =>
    b.textContent?.includes("account_settings.api_tokens.expiry.set")
  )!;
  await act(async () => {
    expiry.focus();
    expiry.click();
  });
  const custom = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find((option) =>
    option.textContent?.includes("account_settings.api_tokens.expiry.custom")
  )!;
  await act(async () => custom.click());
  const date = document.querySelector<HTMLButtonElement>(
    'button[aria-label="account_settings.api_tokens.expiry.set_date"]'
  )!;
  await act(async () => {
    date.focus();
    date.click();
  });
  expect(day("2026-10-30").disabled).toBe(true);
  expect(day("2026-10-31").disabled).toBe(false);
  await act(async () => day("2026-10-31").click());
  await act(async () => vi.advanceTimersByTimeAsync(50));
  const submit = document.querySelector<HTMLButtonElement>('button[type="submit"]')!;
  await act(async () => submit.click());
  expect(submitted).toHaveBeenCalledTimes(1);
  expect(submitted.mock.calls[0]?.[0]).toEqual({
    label: "deploy",
    description: "",
    expired_at: new Date(2026, 9, 31, 12).toISOString(),
  });
  expect(host.getAttribute("aria-hidden")).toBe("true");
});
