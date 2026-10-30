// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeAll, beforeEach, expect, it, vi } from "vitest";
import { initPromise, setLanguage } from "@nerve/i18n";
import { CustomSearchSelect } from "@nerve/ui";
import { WorkItemFilterStore } from "@nerve/shared-state";
import { getStartDateFilterConfig } from "@nerve/utils";
import {
  COLLECTION_OPERATOR,
  EQUALITY_OPERATOR,
  FILTER_FIELD_TYPE,
  FILTER_NODE_TYPE,
  EIssuesStoreType,
} from "@nerve/types";
import type {
  IFilterOption,
  TSingleSelectFilterFieldConfig,
  TMultiSelectFilterFieldConfig,
  TWorkItemFilterProperty,
} from "@nerve/types";
import { heldChange } from "@/lib/auth/fake-tab";
import { FilterValueInput } from "./filter-value-input/root";
import { AddFilterDropdown } from "./add-filters/dropdown";
import { FilterItem } from "./filter-item/root";

vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: { start_of_the_week: 1 } }) }));

type Config = TSingleSelectFilterFieldConfig<string> | TMultiSelectFilterFieldConfig<string>;
const kinds = ["single", "multi"] as const;
const options = (name: string): IFilterOption<string>[] => [{ id: name, value: name, label: name }];
const config = (kind: (typeof kinds)[number], getOptions: Config["getOptions"]): Config =>
  kind === "single"
    ? { type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions }
    : { type: FILTER_FIELD_TYPE.MULTI_SELECT, getOptions, singleValueOperator: EQUALITY_OPERATOR.EXACT };
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const changed = vi.fn();
beforeAll(async () => {
  await initPromise;
});
beforeEach(async () => {
  await setLanguage("en");
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
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  changed.mockClear();
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function render(field: Config, value: string | string[] = [], id = "condition") {
  await act(async () =>
    root.render(
      <FilterValueInput
        filterFieldConfig={field}
        condition={{
          id,
          type: FILTER_NODE_TYPE.CONDITION,
          property: "label_id",
          operator: COLLECTION_OPERATOR.IN,
          value,
        }}
        onChange={changed}
      />
    )
  );
}
const popup = () => document.querySelector('[role="listbox"]');
const choose = () =>
  act(async () => {
    // Headless UI options select on mousedown to retain the search input's focus.
    document
      .querySelector<HTMLElement>('[role="option"]')!
      .dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
  });

it.each(kinds)(
  "%s opens an empty condition while options load and uses the answer without another click",
  async (kind) => {
    const held = heldChange<IFilterOption<string>[]>();
    const field = config(kind, () => held.sent);
    await render(field);
    expect(popup()).not.toBeNull();
    expect(popup()?.textContent).toContain("Loading options");
    await act(async () => held.answer(options("Ready")));
    expect(document.querySelector('[role="option"]')?.textContent).toContain("Ready");
    expect(document.querySelector<HTMLInputElement>('[role="combobox"]')?.placeholder).toBe("Search");
    await choose();
    expect(changed).toHaveBeenCalledWith(kind === "single" ? "Ready" : ["Ready"]);
  }
);

it.each(kinds)("%s ignores an old answer and finally while the new config is still loading", async (kind) => {
  const old = heldChange<IFilterOption<string>[]>(),
    current = heldChange<IFilterOption<string>[]>();
  await render(config(kind, () => old.sent));
  await render(config(kind, () => current.sent));
  await act(async () => old.answer(options("Old directory")));
  expect(document.body.textContent).not.toContain("Old directory");
  expect(popup()?.textContent).toContain("Loading options");
  await act(async () => current.answer(options("Current directory")));
  expect(document.querySelector('[role="option"]')?.textContent).toContain("Current directory");
});

it.each(kinds)(
  "%s shows a current loading refusal with retry and ignores the previous config's error",
  async (kind) => {
    const old = heldChange<IFilterOption<string>[]>(),
      current = heldChange<IFilterOption<string>[]>();
    let calls = 0;
    await render(config(kind, () => old.sent));
    await render(config(kind, () => (++calls === 1 ? current.sent : options("Retried"))));
    await act(async () => old.refuse(new Error("old refusal")));
    expect(document.querySelector('[role="alert"]')).toBeNull();
    await act(async () => current.refuse(new Error("current refusal")));
    expect(document.querySelector('[role="alert"]')?.textContent).toContain("Could not load options");
    const retry = [...document.querySelectorAll("button")].find((button) => button.textContent === "Retry");
    expect(retry).toBeDefined();
    await act(async () => {
      retry!.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
      retry!.click();
    });
    expect(document.querySelector('[role="alert"]')).toBeNull();
    expect(document.querySelector('[role="option"]')?.textContent).toContain("Retried");
    expect(calls).toBe(2);
  }
);

it.each(kinds)(
  "%s keeps an unavailable selected ID visible and preserves it when another value is chosen",
  async (kind) => {
    await render(config(kind, options("Known")), kind === "single" ? "unknown-id" : ["unknown-id"]);
    expect(host.textContent).toContain("Unavailable option");
    expect(changed).not.toHaveBeenCalled();
    await act(async () => host.querySelector("button")!.click());
    expect(document.querySelector('[role="option"]')?.textContent).toContain("Known");
    await choose();
    expect(changed).toHaveBeenCalledWith(kind === "single" ? "Known" : ["unknown-id", "Known"]);
  }
);

it.each(kinds.flatMap((kind) => (["en", "zh-CN"] as const).map((language) => ({ kind, language }))))(
  "uses real $language translations for $kind search, loading, and empty results",
  async ({ kind, language }) => {
    await act(async () => setLanguage(language));
    const held = heldChange<IFilterOption<string>[]>();
    await render(config(kind, () => held.sent));
    expect(popup()?.textContent).toContain(language === "en" ? "Loading options" : "正在加载选项");
    expect(document.querySelector<HTMLInputElement>('[role="combobox"]')?.placeholder).toBe(
      language === "en" ? "Search" : "搜索"
    );
    await act(async () => held.answer([]));
    expect(popup()?.textContent).toContain(language === "en" ? "No options found" : "未找到选项");
  }
);

it.each(["en", "zh-CN"] as const)(
  "uses real %s search text in the property and operator selectors",
  async (language) => {
    await act(async () => setLanguage(language));
    const filter = new WorkItemFilterStore().getOrCreateFilter({
      entityType: EIssuesStoreType.PROJECT,
      entityId: "selectors",
      initialExpression: { and: [{ start_date__exact: "2026-10-30" }] },
    });
    filter.configManager.registerAll([
      getStartDateFilterConfig<TWorkItemFilterProperty>("start_date")({ isEnabled: true }),
    ]);
    filter.configManager.setAreConfigsReady(true);
    await act(async () =>
      root.render(
        <AddFilterDropdown
          filter={filter}
          handleFilterSelect={changed}
          customButton={<span>Add filter</span>}
          buttonConfig={{ defaultOpen: true }}
        />
      )
    );
    expect(document.querySelector<HTMLInputElement>('[role="combobox"]')?.placeholder).toBe(
      language === "en" ? "Search" : "搜索"
    );
    expect(document.querySelector('[role="option"]')?.textContent).toContain("Start date");
    await act(async () =>
      root.render(<FilterItem filter={filter} condition={filter.allConditions[0]} showTransition={false} />)
    );
    const operator = [...host.querySelectorAll("button")].find(
      (button) => button.textContent === (language === "en" ? "is" : "是")
    );
    expect(operator).toBeDefined();
    await act(async () => operator!.click());
    expect(document.querySelector<HTMLInputElement>('[role="combobox"]')?.placeholder).toBe(
      language === "en" ? "Search" : "搜索"
    );
    expect(popup()?.textContent).toContain(language === "en" ? "between" : "介于");
  }
);

it("notifies one opening while a real search select rerenders with new options", async () => {
  const opened = vi.fn();
  const view = (ready: boolean) => (
    <CustomSearchSelect
      defaultOpen
      value=""
      onChange={changed}
      onOpen={opened}
      customButton={<span>Options</span>}
      options={ready ? [{ value: "x", query: "x", content: "Ready" }] : undefined}
    />
  );
  await act(async () => root.render(view(false)));
  await act(async () => root.render(view(true)));
  expect(popup()).not.toBeNull();
  expect(opened).toHaveBeenCalledTimes(1);
});
