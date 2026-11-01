// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, afterEach, it, expect, vi } from "vitest";
import { CustomSelect } from "./custom-select";
import { CustomSearchSelect } from "./custom-search-select";
import { CustomMenu } from "./custom-menu";

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const changed = vi.fn(),
  opened = vi.fn(),
  closed = vi.fn();
beforeEach(() => {
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
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  changed.mockClear();
  opened.mockClear();
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
async function render(content: ReactNode) {
  await act(async () => root.render(content));
}
async function tick() {
  await act(async () => vi.advanceTimersByTimeAsync(50));
}
async function click(element: HTMLElement) {
  await act(async () => {
    element.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    element.click();
  });
  await tick();
}
function trigger() {
  return host.querySelector<HTMLButtonElement>("button")!;
}
function option(text: string) {
  const found = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === text
  );
  if (!found) throw new Error(`Missing option: ${text}`);
  return found;
}
function list() {
  return document.querySelector('[role="listbox"]');
}
async function type(value: string) {
  const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function key(element: HTMLElement, pressed: string, keyCode = 0) {
  await act(async () =>
    element.dispatchEvent(new KeyboardEvent("keydown", { key: pressed, keyCode, bubbles: true, cancelable: true }))
  );
  await tick();
}

it.each([0, ""])(
  "preservesEmptyAndZeroSelections: %s is selected, emitted once and closes the search popup",
  async (value) => {
    await render(
      <CustomSearchSelect
        value={null}
        options={[{ value, query: "Pick", content: "Pick" }]}
        onChange={changed}
        onOpen={opened}
        onClose={closed}
        label="Select"
      />
    );
    const button = trigger();
    await click(button);
    await click(option("Pick"));
    expect(changed.mock.calls).toEqual([[value]]);
    expect(list()).toBeNull();
    expect(opened).toHaveBeenCalledTimes(1);
    expect(closed).toHaveBeenCalledTimes(1);
    expect(document.activeElement).toBe(button);
  }
);

it.each([null, [0]])(
  "multiSelectionStaysOpenAndReturnsAnArray from %j without mutating the caller's array",
  async (initial) => {
    const options = [
      { value: 0, query: "Zero", content: "Zero" },
      { value: 1, query: "One", content: "One" },
    ];
    function Multi() {
      const [value, setValue] = useState(initial);
      return (
        <CustomSearchSelect
          multiple
          value={value}
          options={options}
          label="Members"
          onChange={(values: number[]) => {
            changed(values);
            setValue(values);
          }}
          onOpen={opened}
          onClose={closed}
        />
      );
    }
    await render(<Multi />);
    await click(trigger());
    await click(option("One"));
    const picked = initial ? [0, 1] : [1];
    expect(changed.mock.calls).toEqual([[picked]]);
    expect(changed.mock.calls[0][0]).not.toBe(initial);
    expect(initial).toEqual(initial ? [0] : null);
    expect(list()).not.toBeNull();
    await click(option("One"));
    expect(changed.mock.calls.at(-1)).toEqual([initial ?? []]);
    expect(list()).not.toBeNull();
    expect(opened).toHaveBeenCalledTimes(1);
    expect(closed).not.toHaveBeenCalled();
  }
);

it("clearing only the search never emits a null selection or closes the popup", async () => {
  await render(
    <CustomSearchSelect
      value="one"
      options={[{ value: "one", query: "One", content: "One" }]}
      onChange={changed}
      onClose={closed}
      label="Search"
    />
  );
  await click(trigger());
  await type("missing");
  expect(document.querySelectorAll('[role="option"]')).toHaveLength(0);
  await type("");
  expect(changed).not.toHaveBeenCalled();
  expect(option("One")).not.toBeNull();
  expect(closed).not.toHaveBeenCalled();
  expect(list()).not.toBeNull();
});

it("objectSelectionReturnsTheOriginalThemeOption and nullable current selection does not transform it", async () => {
  const theme = { name: "Dark", colors: ["black", "white"] };
  await render(
    <CustomSelect value={null} label="Theme" onChange={changed}>
      <CustomSelect.Option value={theme}>Dark</CustomSelect.Option>
    </CustomSelect>
  );
  await click(trigger());
  await click(option("Dark"));
  expect(changed).toHaveBeenCalledTimes(1);
  expect(changed.mock.calls[0][0]).toBe(theme);
  expect(list()).toBeNull();
});

it.each([false, 0, ""])("CustomSelect preserves a falsy value %s without submitting its form", async (value) => {
  const submit = vi.fn();
  await render(
    <form
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <CustomSelect value={null} label="Value" onChange={changed}>
        <CustomSelect.Option value={value}>Pick</CustomSelect.Option>
      </CustomSelect>
    </form>
  );
  const button = trigger();
  await act(async () => button.focus());
  await key(button, "Enter", 229);
  expect(list()).toBeNull();
  await key(button, "Enter");
  expect(list()).not.toBeNull();
  await click(option("Pick"));
  expect(changed.mock.calls).toEqual([[value]]);
  expect(submit).not.toHaveBeenCalled();
});

it("loading, disabled options, footer, defaultOpen and query reset retain their contracts", async () => {
  const control = (options?: { value: number; query: string; content: string; disabled?: boolean }[]) => (
    <CustomSearchSelect
      value={null}
      options={options}
      label="Options"
      onChange={changed}
      defaultOpen
      loadingMessage="Loading choices"
      noResultsMessage="No choice"
      footerOption={<button type="button">Footer</button>}
      onOpen={opened}
      onClose={closed}
    />
  );
  await render(control());
  await tick();
  expect(list()?.textContent).toContain("Loading choices");
  expect(list()?.textContent).toContain("Footer");
  await render(control([{ value: 0, query: "Disabled", content: "Disabled", disabled: true }]));
  await click(option("Disabled"));
  expect(changed).not.toHaveBeenCalled();
  await type("absent");
  expect(list()?.textContent).toContain("No choice");
  const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
  await key(input, "Escape", 229);
  expect(list()).not.toBeNull();
  await key(input, "Escape");
  expect(list()).toBeNull();
  await click(trigger());
  expect(option("Disabled")).not.toBeNull();
  expect(opened).toHaveBeenCalledTimes(2);
  expect(closed).toHaveBeenCalledTimes(1);
});

it("a menu's opening notification has no arguments and its item receives the clicked button event", async () => {
  let eventTarget: HTMLButtonElement | null = null;
  await render(
    <CustomMenu label="Menu" menuButtonOnClick={opened}>
      <CustomMenu.MenuItem
        onClick={(event) => {
          eventTarget = event.currentTarget;
          changed(event.type);
        }}
      >
        Action
      </CustomMenu.MenuItem>
    </CustomMenu>
  );
  await click(trigger());
  const button = document.querySelector<HTMLButtonElement>('[role="menuitem"]')!;
  await click(button);
  expect(opened.mock.calls).toEqual([[]]);
  expect(changed.mock.calls).toEqual([["click"]]);
  expect(eventTarget).toBe(button);
});
