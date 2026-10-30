// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, host, renderer, changed, choices, posts, open, enter } from "./fake-issue-properties";
import { act } from "react";
import { expect, it, vi } from "vitest";
import { labelOf } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { IssuePropertyLabels } from "./issue-layouts/properties/labels";
import { ModalCore, ModalTitle } from "@nerve/ui";
import { until } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
it.each(choices)(
  "$name opens by keyboard, positions the whole popup, selects existing labels without creating, and restores the same trigger",
  async (choice) => {
    const existing = labelOf(tab.project, "Existing", 1);
    tab.nerve.respond = (call) => {
      if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [existing] }));
    };
    await act(async () => renderer.render(choice.view()));
    const trigger = host.querySelector<HTMLButtonElement>("button")!;
    expect(trigger.tabIndex).toBe(0);
    expect(host.querySelector('div[tabindex="0"]')).toBeNull();
    await act(async () => {
      trigger.focus();
      trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
      await vi.advanceTimersByTimeAsync(20);
    });
    await act(async () => until(() => tab.nerve.calls.length === 1, "keyboard label directory"));
    const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
    expect(input).not.toBeNull();
    expect(document.activeElement).toBe(input);
    const list = document.querySelector('[role="listbox"]')!;
    expect(list.closest("[data-popper-placement]")).not.toBeNull();
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Existing");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      input.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
      input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
    });
    expect(changed).toHaveBeenCalledExactlyOnceWith([existing.id]);
    expect(posts()).toHaveLength(0);
    await act(async () => {
      input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
      await vi.advanceTimersByTimeAsync(50);
    });
    expect(document.querySelector('[role="listbox"]')).toBeNull();
    expect(document.activeElement).toBe(trigger);
  }
);

it.each(choices)(
  "$name keeps outer ModalCore background isolation while searching and dismissing the inner list",
  async (choice) => {
    await act(async () =>
      renderer.render(
        <ModalCore isOpen>
          <ModalTitle>Edit labels</ModalTitle>
          {choice.view()}
        </ModalCore>
      )
    );
    const trigger = document.querySelector<HTMLButtonElement>('[role="dialog"] button')!;
    await act(async () => {
      trigger.focus();
      trigger.click();
      await vi.advanceTimersByTimeAsync(20);
    });
    const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
    expect(input).not.toBeNull();
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Search");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(host.getAttribute("aria-hidden")).toBe("true");
    expect(document.querySelector('[role="dialog"]')?.getAttribute("aria-hidden")).not.toBe("true");
    await act(async () => {
      input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
      await vi.advanceTimersByTimeAsync(50);
    });
    expect(document.querySelector('[role="listbox"]')).toBeNull();
    expect(document.querySelector('[role="dialog"]')).not.toBeNull();
    expect(host.getAttribute("aria-hidden")).toBe("true");
  }
);

it("the real list labels trigger names its work item and has no unnamed root Tab stop", async () => {
  await act(async () =>
    renderer.render(<IssuePropertyLabels projectId="A" value={[]} onChange={changed} issueName="Fix race" />)
  );
  const trigger = host.querySelector<HTMLButtonElement>("button")!;
  expect(trigger.getAttribute("aria-label")).toBe("common.labels: Fix race");
  expect(host.querySelector('div[tabindex="0"]')).toBeNull();
});

it.each(choices)("$name does not attach a creation from a dismissed popup to its next opening", async (choice) => {
  await open(choice);
  await enter();
  const old = posts()[0]!;
  await act(async () => {
    document
      .querySelector("input")!
      .dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  await act(async () => {
    host.querySelector<HTMLButtonElement>("button")!.click();
    await vi.advanceTimersByTimeAsync(20);
  });
  const nextInput = document.querySelector<HTMLInputElement>("input")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(nextInput, "Next query");
    nextInput.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => {
    old.answer(json(201, labelOf(tab.project, "Old created", 1)));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(changed).not.toHaveBeenCalled();
  expect(nextInput.value).toBe("Next query");
  expect(toasts).toHaveLength(0);
  expect(document.querySelector('[role="listbox"]')).not.toBeNull();
});

it("searching a grouped child keeps its parent heading and selects that child without creating a duplicate", async () => {
  const parent = labelOf(tab.project, "Parent", 1);
  const child = labelOf(tab.project, "Needle", 2, { parent_id: parent.id });
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [parent, child] }));
  };
  await open(choices[0]!);
  const input = document.querySelector<HTMLInputElement>("input")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Needle");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  expect(document.querySelector('[role="listbox"]')?.textContent).toContain("Parent");
  expect(document.querySelectorAll('[role="option"]')).toHaveLength(1);
  await act(async () => {
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  expect(changed).toHaveBeenCalledExactlyOnceWith([child.id]);
  expect(posts()).toHaveLength(0);
});

it("a current row label replacement refusal is visible once and leaves the confirmed selection unchanged", async () => {
  const existing = labelOf(tab.project, "Existing", 1);
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [existing] }));
  };
  changed.mockImplementation(async () => {
    throw new Error("refused label change");
  });
  await open(choices[1]!);
  const input = document.querySelector<HTMLInputElement>("input")!;
  await act(async () => {
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  expect(changed).toHaveBeenCalledExactlyOnceWith([existing.id]);
  expect(toasts).toHaveLength(1);
  expect(document.querySelector('[role="option"]')?.getAttribute("aria-selected")).toBe("false");
});

it("a refused row replacement after its popup closed cannot toast into the next opening", async () => {
  const existing = labelOf(tab.project, "Existing", 1);
  const write = heldChange<void>();
  changed.mockReturnValue(write.sent);
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/labels")) call.answer(json(200, { data: [existing] }));
  };
  await open(choices[1]!);
  const input = document.querySelector<HTMLInputElement>("input")!;
  await act(async () => {
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  expect(changed).toHaveBeenCalledTimes(1);
  await act(async () => {
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(50);
  });
  await act(async () => {
    host.querySelector<HTMLButtonElement>("button")!.click();
    await vi.advanceTimersByTimeAsync(20);
  });
  await act(async () => write.refuse(new Error("late refusal")));
  expect(toasts).toHaveLength(0);
  expect(document.querySelector('[role="listbox"]')).not.toBeNull();
});
