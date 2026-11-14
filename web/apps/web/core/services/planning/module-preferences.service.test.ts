/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The caller's own list settings on a module's page (P6b spec 2.4), against a fake nerve: one method per operation.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ModulePreferences } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle } from "@/lib/auth/fake-time";
import { preferencesOf } from "@/store/project/fake-projects";
import { ModulePreferencesService } from "./module-preferences.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const M1 = "019a0000-0000-7000-8000-0000000000d1";
const { display_filters, display_properties } = preferencesOf();
const saved: ModulePreferences = { filters: { priority: ["high"] }, display_filters, display_properties };

it("reads the module page's settings by the module's id and gives nerve's answer as it is", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(json(200, saved));
  const service = new ModulePreferencesService(nerve.client());
  expect((await settle(service.get(M1), "the settings")).value).toEqual(saved);
  expect(nerve.calls.map(({ method, path, body }) => ({ method, path, body }))).toEqual([
    { method: "GET", path: `/api/v0/me/modules/${M1}/preferences`, body: undefined },
  ]);
});

it("sends the fields it is given, each whole, and gives the page's settings as nerve changed them", async () => {
  const nerve = new FakeNerve();
  const changed: ModulePreferences = { ...saved, filters: { cycle_id: ["none"] } };
  nerve.respond = (call) => call.answer(json(200, changed));
  const service = new ModulePreferencesService(nerve.client());
  const body = { filters: { cycle_id: ["none"] }, display_properties: { ...display_properties, priority: false } };
  expect((await settle(service.update(M1, body), "the change")).value).toEqual(changed);
  expect(nerve.calls.map(({ method, path, body: sent }) => ({ method, path, body: sent }))).toEqual([
    { method: "PATCH", path: `/api/v0/me/modules/${M1}/preferences`, body },
  ]);
});

it("hands nerve's refusal to the caller", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) =>
    call.answer(call.method === "GET" ? problem(404, "planning.module_not_found") : problem(400, "bad_request"));
  const service = new ModulePreferencesService(nerve.client());
  const read = await settle(service.get(M1), "the read");
  expect(read.error).toBeInstanceOf(ApiError);
  expect(read.error).toMatchObject({ status: 404, problem: { code: "planning.module_not_found" } });
  const change = await settle(service.update(M1, { filters: {} }), "the change");
  expect(change.error).toMatchObject({ status: 400, problem: { code: "bad_request" } });
});
