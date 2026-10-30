// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, afterEach, it, expect, vi } from "vitest";
import { CustomSelect, ModalCore, ModalTitle } from "@nerve/ui";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { PriorityDropdown } from "./priority";
import { ProjectDropdownBase } from "./project/base";
import { WorkItemStateDropdownBase } from "./state/base";
import { MemberDropdownBase } from "./member/base";
import { DateRangeDropdown } from "./date-range";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/user", () => ({
  useUserProfile: () => ({ data: { start_of_the_week: 1 } }),
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ workspace: { isUserSuspended: () => false }, getUserDetails: () => undefined }),
}));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const changed = vi.fn(),
  closed = vi.fn(),
  opened = vi.fn();
const project = projectOf("WEB", "ws");
const todo = stateOf(project, "Todo", "unstarted", 1);
const factories = {
  priority: (disabled = false) => (
    <PriorityDropdown
      buttonVariant="border-with-text"
      value="low"
      onChange={changed}
      onClose={closed}
      disabled={disabled}
      renderByDefault={false}
    />
  ),
  project: (disabled = false) => (
    <ProjectDropdownBase
      buttonVariant="border-with-text"
      value={null}
      onChange={changed}
      disabled={disabled}
      multiple={false}
      projectIds={[project.id, "missing"]}
      getProjectById={(id) => (id === project.id ? project : undefined)}
    />
  ),
  state: (disabled = false) => (
    <WorkItemStateDropdownBase
      buttonVariant="border-with-text"
      value={null}
      onChange={changed}
      onClose={closed}
      onDropdownOpen={opened}
      disabled={disabled}
      renderByDefault={false}
      projectId={project.id}
      stateIds={[todo.id]}
      getStateById={(id) => (id === todo.id ? todo : undefined)}
      getStatePercentageInGroup={() => undefined}
    />
  ),
};
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
  opened.mockClear();
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
async function press(element: Element, key: string, composing = false) {
  await act(async () => {
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, isComposing: composing });
    element.dispatchEvent(down);
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true, isComposing: composing });
    element.dispatchEvent(up);
    if (
      !composing &&
      !down.defaultPrevented &&
      !up.defaultPrevented &&
      ["Enter", " "].includes(key) &&
      element instanceof HTMLButtonElement
    )
      element.click();
  });
  await tick();
}
async function type(input: HTMLInputElement, value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const day = (date: string) => document.querySelector<HTMLButtonElement>(`[data-day="${date}"] button`)!;
const list = () => document.querySelector<HTMLElement>('[role="listbox"]');
for (const [kind, control] of Object.entries(factories)) {
  it(`${kind}: a single named native trigger opens without hover, search owns focus, positioning owns the whole list and Escape restores the trigger`, async () => {
    await render(control());
    expect(host.querySelector("button button")).toBeNull();
    const button = host.querySelector<HTMLButtonElement>("button")!;
    expect(button.tabIndex).toBe(0);
    await act(async () => button.focus());
    await press(button, "Enter");
    const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
    expect(input).not.toBeNull();
    expect(document.activeElement).toBe(input);
    expect(list()).not.toBeNull();
    const positioned = list()!.closest<HTMLElement>("[data-popper-placement]");
    expect(positioned).not.toBeNull();
    expect(positioned!.style.position).toBe("absolute");
    await type(input, "zzzz");
    expect(document.querySelectorAll('[role="option"]')).toHaveLength(0);
    await type(input, "");
    expect(document.querySelectorAll('[role="option"]')).toHaveLength(kind === "priority" ? 5 : 1);
    await press(input, "Escape", true);
    expect(list()).not.toBeNull();
    await press(input, "Escape");
    expect(list()).toBeNull();
    expect(document.activeElement).toBe(button);
    expect(button.getAttribute("aria-expanded")).toBe("false");
    await press(button, " ");
    expect(list()).not.toBeNull();
    await press(document.querySelector('input[role="combobox"]')!, "ArrowDown");
    await press(document.querySelector('input[role="combobox"]')!, "Enter");
    expect(changed).toHaveBeenCalledTimes(1);
    expect(list()).toBeNull();
    if (kind === "state") expect(opened).toHaveBeenCalledTimes(2);
  });
  it(`${kind}: opening its list in ModalCore does not release or replace the outer background isolation`, async () => {
    await render(
      <ModalCore isOpen handleClose={closed}>
        <ModalTitle>Edit properties</ModalTitle>
        {control()}
        <button>After</button>
      </ModalCore>
    );
    const button = document.querySelector<HTMLButtonElement>('[role="dialog"] button')!;
    await act(async () => {
      button.focus();
      button.click();
    });
    await tick();
    expect(list()).not.toBeNull();
    expect(host.getAttribute("aria-hidden")).toBe("true");
    await type(document.querySelector<HTMLInputElement>('input[role="combobox"]')!, "e");
    expect(host.getAttribute("aria-hidden")).toBe("true");
    expect(document.querySelector('[role="dialog"]')?.getAttribute("aria-hidden")).not.toBe("true");
    await press(document.querySelector('input[role="combobox"]')!, "Escape");
    expect(list()).toBeNull();
    expect(document.querySelector('[role="dialog"]')).not.toBeNull();
    expect(host.getAttribute("aria-hidden")).toBe("true");
  });
  it(`${kind}: disabled leaves one disabled trigger and no popup`, async () => {
    await render(control(true));
    const button = host.querySelector<HTMLButtonElement>("button")!;
    expect(button.disabled).toBe(true);
    await act(async () => button.click());
    expect(list()).toBeNull();
    expect(changed).not.toHaveBeenCalled();
  });
}

it("date range uses one calendar panel and keeps the confirmed display after rejected selection or clearing", async () => {
  const value = { from: new Date(2026, 9, 20), to: new Date(2026, 9, 25) };
  await render(
    <DateRangeDropdown
      buttonVariant="border-with-text"
      value={value}
      onSelect={changed}
      isClearable
      renderByDefault={false}
      minDate={new Date(2026, 9, 19)}
      maxDate={new Date(2026, 9, 31)}
    />
  );
  expect(host.querySelector("button button")).toBeNull();
  const button = host.querySelector<HTMLButtonElement>("button")!;
  const confirmedText = button.textContent;
  await act(async () => button.focus());
  await press(button, "Enter");
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();

  expect(day("2026-10-18").disabled).toBe(true);
  await act(async () => day("2026-10-30").click());
  expect(changed).toHaveBeenCalledTimes(1);
  expect(button.textContent).toBe(confirmedText);
  await press(day("2026-10-30"), "Escape");
  expect(document.querySelector('[role="dialog"]')).toBeNull();
  expect(document.activeElement).toBe(button);
  const clear = host.querySelector<HTMLButtonElement>(
    'button[aria-label="common.clear_date: project_cycles.date_range"]'
  )!;
  await act(async () => clear.click());
  expect(changed.mock.calls.at(-1)).toEqual([{ from: undefined, to: undefined }]);
  expect(button.textContent).toBe(confirmedText);
});

it("project multiple selection preserves its array, skips missing or filtered directory entries, and stays open", async () => {
  const other = projectOf("API", "ws");
  await render(
    <ProjectDropdownBase
      buttonVariant="border-with-text"
      multiple
      value={[project.id]}
      onChange={changed}
      projectIds={[project.id, other.id, "missing"]}
      getProjectById={(id) => [project, other].find((item) => item.id === id)}
      renderCondition={(id) => id !== "missing"}
    />
  );
  const trigger = host.querySelector<HTMLButtonElement>("button")!;
  await act(async () => trigger.click());
  await tick();
  expect(document.querySelectorAll('[role="option"]')).toHaveLength(2);
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find((item) =>
    item.textContent?.includes("API")
  )!;
  await act(async () => {
    option.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    option.click();
  });
  expect(changed.mock.calls).toEqual([[[project.id, other.id]]]);
  expect(list()).not.toBeNull();
});

it.each([false, true])("date range defaultOpen and disabled preserve the %s portal choice", async (portal) => {
  const value = { from: new Date(2026, 9, 20), to: new Date(2026, 9, 25) };
  const control = (disabled: boolean) => (
    <DateRangeDropdown
      buttonVariant="border-with-text"
      value={value}
      defaultOpen
      renderInPortal={portal}
      disabled={disabled}
    />
  );
  await render(control(false));
  await tick();
  const panel = document.querySelector('[role="dialog"]')!;
  expect(panel).not.toBeNull();
  expect(host.contains(panel)).toBe(!portal);
  await render(control(true));
  await tick();
  expect(document.querySelector('[role="dialog"]')).toBeNull();
  expect(host.querySelector<HTMLButtonElement>("button")?.disabled).toBe(true);
});

it("a CustomSelect trigger does not open or submit on either IME Enter ordering", async () => {
  await render(
    <CustomSelect value="side" label="Layout" onChange={changed}>
      <CustomSelect.Option value="side">Side</CustomSelect.Option>
    </CustomSelect>
  );
  const button = host.querySelector<HTMLButtonElement>("button")!;
  await act(async () => button.focus());
  await press(button, "Enter", true);
  expect(list()).toBeNull();
  await act(async () => {
    button.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", keyCode: 229, bubbles: true, cancelable: true }));
  });
  await tick();
  expect(list()).toBeNull();
  expect(changed).not.toHaveBeenCalled();
  await press(button, "Enter");
  expect(list()).not.toBeNull();
});

it("a deferred icon-only member control stays named and preserves its native trigger across keyboard open and close", async () => {
  await render(
    <MemberDropdownBase
      value={[]}
      multiple
      memberIds={[]}
      getUserDetails={() => undefined}
      onChange={changed}
      buttonVariant="border-without-text"
      placeholder="Assignees"
      renderByDefault={false}
    />
  );
  const trigger = host.querySelector<HTMLButtonElement>("button")!;
  expect(trigger.textContent).toContain("Assignees");
  await act(async () => trigger.focus());
  await press(trigger, "Enter");
  expect(host.querySelector("button")).toBe(trigger);
  const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
  expect(document.activeElement).toBe(input);
  await press(input, "Escape");
  expect(document.activeElement).toBe(trigger);
});
it("a state directory refresh preserves its selected button label while the spinner is present", async () => {
  await render(
    <WorkItemStateDropdownBase
      buttonVariant="border-with-text"
      value={todo.id}
      onChange={changed}
      projectId={project.id}
      stateIds={[todo.id]}
      getStateById={() => todo}
      getStatePercentageInGroup={() => undefined}
      isInitializing
    />
  );
  expect(host.querySelector("button")?.textContent).toContain("Todo");
});
