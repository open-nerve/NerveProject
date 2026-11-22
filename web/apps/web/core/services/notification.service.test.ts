/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The caller's notifications (P4 spec 2.1), against a fake nerve: one method per operation, nerve's answer as it is.
import { afterEach, beforeEach, expect, expectTypeOf, it, vi } from "vitest";
import type { NotificationMarkAllRead, operations } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle } from "@/lib/auth/fake-time";
import { countOf, notificationOf, pageOf } from "@/store/notification/fake-notifications";
import { type NotificationListQuery, NotificationService } from "./notification.service";

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

it("lists the caller's notifications in a workspace by its slug, with the query as it is: strings, the page size a number", async () => {
  expectTypeOf<NotificationListQuery>().toEqualTypeOf<
    NonNullable<operations["listNotifications"]["parameters"]["query"]>
  >();
  expectTypeOf<NotificationListQuery["limit"]>().toEqualTypeOf<number | undefined>();
  expectTypeOf<Omit<NotificationListQuery, "limit">>().toEqualTypeOf<{
    tab?: string;
    type?: string;
    read?: string;
    archived?: string;
    snoozed?: string;
    cursor?: string;
  }>();
  const nerve = new FakeNerve();
  const page = pageOf([notificationOf("renamed")], "c-2");
  nerve.respond = (call) => call.answer(json(200, page));
  const service = new NotificationService(nerve.client());
  const query: NotificationListQuery = {
    tab: "all",
    type: "assigned,created",
    read: "false",
    archived: "false",
    snoozed: "false",
    limit: 100,
    cursor: "opaque+/=",
  };
  expect((await settle(service.list("acme", query), "the list")).value).toEqual(page);
  expect(nerve.calls.map(({ method, path, queryEntries, body }) => ({ method, path, queryEntries, body }))).toEqual([
    {
      method: "GET",
      path: "/api/v0/workspaces/acme/notifications",
      queryEntries: [
        ["tab", "all"],
        ["type", "assigned,created"],
        ["read", "false"],
        ["archived", "false"],
        ["snoozed", "false"],
        ["limit", "100"],
        ["cursor", "opaque+/="],
      ],
      body: undefined,
    },
  ]);
});

it("hands nerve's refusal of a list to the caller", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(problem(422, "validation_failed"));
  const service = new NotificationService(nerve.client());
  const read = await settle(service.list("acme", { tab: "mentions", type: "assigned" }), "the list");
  expect(read.error).toBeInstanceOf(ApiError);
  expect(read.error).toMatchObject({ status: 422, problem: { code: "validation_failed" } });
});

it("changes the caller's notification with the body as it is: null ends its snooze (T1); gives nerve's answer", async () => {
  const nerve = new FakeNerve();
  const changed = notificationOf("one", { read_at: "2026-11-10T09:00:00Z" });
  nerve.respond = (call) => call.answer(json(200, changed));
  const service = new NotificationService(nerve.client());
  const change = await settle(service.update("n-one", { read: true, snoozed_till: null }), "the change");
  expect(change.value).toEqual(changed);
  expect(nerve.calls.map(({ method, path, query, body }) => ({ method, path, query, body }))).toEqual([
    { method: "PATCH", path: "/api/v0/notifications/n-one", query: {}, body: { read: true, snoozed_till: null } },
  ]);
});

it("marks the caller's unread notifications in a workspace read, the type an array as given; gives how many", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) => call.answer(json(200, { marked: 4 }));
  const service = new NotificationService(nerve.client());
  const filters: NotificationMarkAllRead = {
    tab: "all",
    type: ["assigned", "created"],
    archived: false,
    snoozed: false,
  };
  const marking = await settle(service.markAllRead("acme", filters), "the marking");
  expect(marking.value).toEqual({ marked: 4 });
  expect(nerve.calls.map(({ method, path, query, body }) => ({ method, path, query, body }))).toEqual([
    {
      method: "POST",
      path: "/api/v0/workspaces/acme/notifications/mark-all-read",
      query: {},
      body: { tab: "all", type: ["assigned", "created"], archived: false, snoozed: false },
    },
  ]);
});

it("hands nerve's refusals of the changes to the caller", async () => {
  const nerve = new FakeNerve();
  nerve.respond = (call) =>
    call.answer(call.method === "PATCH" ? problem(404, "notification.not_found") : problem(500, "internal_error"));
  const service = new NotificationService(nerve.client());
  const change = await settle(service.update("n-gone", { archived: true }), "the change");
  expect(change.error).toBeInstanceOf(ApiError);
  expect(change.error).toMatchObject({ status: 404, problem: { code: "notification.not_found" } });
  const marking = await settle(service.markAllRead("acme", {}), "the marking");
  expect(marking.error).toBeInstanceOf(ApiError);
  expect(marking.error).toMatchObject({ status: 500, problem: { code: "internal_error" } });
});
