// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it } from "vitest";
import type { IFilterOption, TSingleSelectFilterFieldConfig } from "@nerve/types";
import { FILTER_FIELD_TYPE } from "@nerve/types";
import { heldChange } from "@/lib/auth/fake-tab";
import { useFilterSelectOptions } from "./use-filter-select-options";

type Config = TSingleSelectFilterFieldConfig<string>;
const options = (label: string): IFilterOption<string>[] => [{ id: label, label, value: label }];
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const rendered: string[] = [];
function Probe({ config }: { config: Config }) {
  const { options: directory, loading, error, retry } = useFilterSelectOptions(config);
  const text = loading ? "loading" : error !== undefined ? "error" : directory.map((option) => option.label).join(",");
  rendered.push(text);
  return (
    <>
      <output>{text}</output>
      <button onClick={retry}>retry</button>
    </>
  );
}
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  host = document.createElement("div");
  root = createRoot(host);
  rendered.length = 0;
});
afterEach(async () => {
  await act(async () => root.unmount());
});
const render = (config: Config) => act(async () => root.render(<Probe config={config} />));

it.each(["success", "error"] as const)("ignores old %s after a new directory has already arrived", async (settling) => {
  const old = heldChange<IFilterOption<string>[]>();
  await render({ type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions: () => old.sent });
  await render({ type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions: options("Current") });
  expect(host.querySelector("output")?.textContent).toBe("Current");
  rendered.length = 0;
  await act(async () => (settling === "success" ? old.answer(options("Old")) : old.refuse(new Error("Old"))));
  expect(host.querySelector("output")?.textContent).toBe("Current");
  expect(rendered).not.toContain("loading");
  expect(rendered).not.toContain("error");
  expect(rendered).not.toContain("Old");
});

it("hides a previous array immediately and captures synchronous throws as retryable reads", async () => {
  await render({ type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions: options("Old") });
  const held = heldChange<IFilterOption<string>[]>();
  let calls = 0;
  const next: Config = {
    type: FILTER_FIELD_TYPE.SINGLE_SELECT,
    getOptions: () => {
      if (++calls === 1) throw null;
      return held.sent;
    },
  };
  rendered.length = 0;
  await render(next);
  expect(rendered).not.toContain("Old");
  expect(host.querySelector("output")?.textContent).toBe("error");
  await act(async () => host.querySelector("button")!.click());
  expect(host.querySelector("output")?.textContent).toBe("loading");
  await act(async () => held.answer(options("Retried")));
  expect(host.querySelector("output")?.textContent).toBe("Retried");
  expect(calls).toBe(2);
});

it.each(["success", "error"] as const)("does not publish a late %s after the owner unmounts", async (settling) => {
  const old = heldChange<IFilterOption<string>[]>();
  await render({ type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions: () => old.sent });
  await act(async () => root.render(null));
  await render({ type: FILTER_FIELD_TYPE.SINGLE_SELECT, getOptions: options("New owner") });
  rendered.length = 0;
  await act(async () => (settling === "success" ? old.answer(options("Old")) : old.refuse(new Error("Old"))));
  expect(host.querySelector("output")?.textContent).toBe("New owner");
  expect(rendered).toEqual([]);
});
