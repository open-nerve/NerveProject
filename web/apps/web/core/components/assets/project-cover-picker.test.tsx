// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { PROJECT_COVERS, type ProjectCoverSelection } from "@/helpers/project-covers";
import { until } from "@/lib/auth/fake-time";
import { imageFile } from "@/store/asset/fake-assets";
import { ProjectCoverPicker } from "./project-cover-picker";
import { pickerTab } from "./fake-image-picker";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof pickerTab>>;
let changed: ReturnType<typeof vi.fn<(value: ProjectCoverSelection) => void>>;
const render = (value: ProjectCoverSelection = {}, disabled = false) =>
  act(async () =>
    tab.renderer.render(<ProjectCoverPicker upload={tab.upload} value={value} disabled={disabled} onChange={changed} />)
  );
const label = (id: string) => `project_cover.preset ${Number(id.slice(6))}`;
const preset = (id: string) => document.querySelector<HTMLButtonElement>(`button[aria-label="${label(id)}"]`);
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await pickerTab();
  changed = vi.fn();
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it("renders all presets as named native controls, never choosing on mount or rerender", async () => {
  await render();
  expect(document.querySelectorAll("img")).toHaveLength(29);
  expect(document.querySelectorAll('button[aria-pressed="false"]')).toHaveLength(29);
  expect(changed).not.toHaveBeenCalled();
  await render();
  expect(changed).not.toHaveBeenCalled();
  expect(tab.nerve.calls).toHaveLength(0);
  expect(tab.puts).toHaveLength(0);
});

it.each([
  { value: {}, selected: null },
  { value: { cover_image_asset_id: "uploaded" }, selected: null },
  { value: { cover_builtin_id: "cover-07" }, selected: "cover-07" },
  { value: { cover_builtin_id: null, cover_image_asset_id: null }, selected: null },
  { value: { cover_builtin_id: "unknown" }, selected: null },
])("leaves the controlled selection unchanged and marks only its known preset: $value", async ({ value, selected }) => {
  await render(value);
  expect(
    [...document.querySelectorAll('button[aria-pressed="true"]')].map((b) => b.getAttribute("aria-label"))
  ).toEqual(selected ? [label(selected)] : []);
  expect(changed).not.toHaveBeenCalled();
});

it("selecting a preset replaces an uploaded ID with only the stable preset ID; clear sends both null", async () => {
  await render({ cover_image_asset_id: "uploaded" });
  expect(preset("cover-04")).not.toBeNull();
  await tab.click(label("cover-04"));
  expect(changed.mock.calls).toEqual([[{ cover_builtin_id: "cover-04" }]]);
  await render(changed.mock.calls[0]?.[0]);
  expect(preset("cover-04")?.getAttribute("aria-pressed")).toBe("true");
  await tab.click("project_cover.remove");
  expect(changed.mock.calls[1]).toEqual([{ cover_builtin_id: null, cover_image_asset_id: null }]);
  expect(tab.nerve.calls).toHaveLength(0);
  expect(tab.puts).toHaveLength(0);
});

it.each([
  { random: 0, id: "cover-01" },
  { random: 0.5, id: "cover-15" },
  { random: 0.99999, id: "cover-29" },
])("only an explicit random click chooses $id once; rerender preserves it", async ({ random, id }) => {
  await render();
  const draw = vi.spyOn(Math, "random").mockReturnValue(random);
  await tab.click("project_cover.random");
  expect(changed.mock.calls).toEqual([[{ cover_builtin_id: id }]]);
  expect(draw).toHaveBeenCalledOnce();
  await render(changed.mock.calls[0]?.[0]);
  expect(draw).toHaveBeenCalledOnce();
  expect(changed).toHaveBeenCalledOnce();
  expect(JSON.stringify(changed.mock.calls)).not.toContain(PROJECT_COVERS[0]?.url);
});

it("switching to a preset cancels the old PUT and preview before publishing; late success cannot complete", async () => {
  await render();
  let pending: ReturnType<typeof tab.upload.start>;
  await act(async () => {
    tab.upload.select(imageFile());
    pending = tab.upload.start();
    await until(() => tab.puts.length === 1, "PUT");
  });
  changed.mockImplementation(() => {
    expect(tab.puts[0]?.signal?.aborted).toBe(true);
    expect(tab.upload.previewURL).toBeUndefined();
    expect(tab.upload.readyAsset).toBeUndefined();
  });
  await tab.click(label("cover-29"));
  expect(changed).toHaveBeenCalledExactlyOnceWith({ cover_builtin_id: "cover-29" });
  expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
  await act(async () => {
    tab.puts[0]?.answer(new Response(null, { status: 204 }));
    expect(await pending).toBeUndefined();
  });
  expect(tab.nerve.calls.map((c) => c.path)).toEqual(["/api/v0/assets"]);
});

it("binding or retired scope prevents preset, random and remove actions without cancelling the active selection", async () => {
  await act(async () => tab.upload.select(imageFile()));
  await render({}, true);
  expect(preset("cover-01")).not.toBeNull();
  await tab.click(label("cover-01"));
  await tab.click("project_cover.random");
  await tab.click("project_cover.remove");
  expect(changed).not.toHaveBeenCalled();
  expect(tab.upload.previewURL).toBe("blob:selection-1");
  await render();
  await act(async () => tab.app.dispose());
  expect([...document.querySelectorAll("button")].every((button) => button.disabled)).toBe(true);
  await tab.click(label("cover-01"));
  expect(changed).not.toHaveBeenCalled();
});
