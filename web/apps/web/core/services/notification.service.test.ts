/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The caller's notifications (P4 spec 2.1), against a fake nerve: one method per operation, nerve's answer as it is.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle } from "@/lib/auth/fake-time";
import { countOf } from "@/store/notification/fake-notifications";
import { NotificationService } from "./notification.service";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

it("reads the caller's unread counts in a workspace by its slug and gives nerve's answer as it is", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(json(200, countOf(3, 1)));
  const service = new NotificationService(nerve.client());
  expect((await settle(service.unreadCount("acme"), "the count")).value).toEqual(countOf(3, 1));
  expect(nerve.calls.map(({ method, path, query, body }) => ({ method, path, query, body }))).toEqual([
    { method: "GET", path: "/api/v0/workspaces/acme/notifications/unread-count", query: {}, body: undefined },
  ]);
});

it("hands nerve's refusal to the caller", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(problem(404, "workspace.not_found"));
  const service = new NotificationService(nerve.client());
  const read = await settle(service.unreadCount("gone"), "the count");
  expect(read.error).toBeInstanceOf(ApiError);
  expect(read.error).toMatchObject({ status: 404, problem: { code: "workspace.not_found" } });
});
