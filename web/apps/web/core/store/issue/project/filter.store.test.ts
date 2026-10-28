/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { workItemFiltersAdapter } from "@nerve/shared-state";
import { toProjectFilters } from "@/lib/issue-filters";
import { EIssueFilterType } from "@nerve/constants";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { preferencesOf } from "@/store/project/fake-projects";
import { issueStoreTab, requestedIssue } from "../fake-issues";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

it("writesOneWholePreferenceField using the last confirmed answer", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const initial = preferencesOf();
  const loading = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  loading.call.answer(json(200, initial));
  await loading.promise;
  const first = store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { priority: false });
  const second = store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { state: false });
  await until(() => nerve.calls.length === 2, "first patch");
  expect(nerve.calls[1]?.body).toEqual({ display_properties: { ...initial.display_properties, priority: false } });
  expect(store.getIssueFilters("A")?.displayProperties?.priority).toBe(true);
  const confirmed = { ...initial.display_properties, priority: false, labels: false };
  nerve.calls[1]?.answer(json(200, preferencesOf({ display_properties: confirmed })));
  await first;
  await until(() => nerve.calls.length === 3, "second patch");
  expect(nerve.calls[2]?.body).toEqual({ display_properties: { ...confirmed, state: false } });
  nerve.calls[2]?.answer(json(200, preferencesOf({ display_properties: { ...confirmed, state: false } })));
  await second;
  expect(store.getIssueFilters("A")?.displayProperties).toMatchObject({ priority: false, labels: false, state: false });
});

it("keeps confirmed filters when a patch is refused and sends them explicitly on the list", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const initial = preferencesOf({ filters: { priority: ["high"] } });
  const loading = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  loading.call.answer(json(200, initial));
  await loading.promise;
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  const patch = await requestedIssue(
    nerve,
    () => store.updateFilterExpression("ws", "A", { and: [{ priority__in: "low" }] }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  expect(patch.call.body).toEqual({ filters: { priority: ["low"] } });
  patch.call.answer(problem(403, "forbidden"));
  await expect(patch.promise).rejects.toBeDefined();
  expect(store.getIssueFilters("A")?.richFilters).toEqual({ and: [{ priority__in: "high" }] });
  const list = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(list.call.query).toMatchObject({ priority: "high" });
  list.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
  await list.promise;
});

it("keeps confirmed values when an older read arrives and when the session retires", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const initial = preferencesOf();
  const first = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  first.call.answer(json(200, initial));
  await first.promise;
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  const patch = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { priority: false }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  patch.call.answer(
    json(200, preferencesOf({ display_properties: { ...initial.display_properties, priority: false } }))
  );
  await patch.promise;
  read.call.answer(json(200, initial));
  await read.promise;
  expect(store.getIssueFilters("A")?.displayProperties?.priority).toBe(false);
  const last = await requestedIssue(
    nerve,
    () => store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { state: false }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  app.dispose();
  last.call.answer(json(200, preferencesOf({ display_properties: { ...initial.display_properties, state: false } })));
  await expect(last.promise).rejects.toBeDefined();
  expect(store.getIssueFilters("A")?.displayProperties?.state).toBe(true);
});

it("translates display vocabulary between controls and persisted preferences", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const initial = preferencesOf({
    display_filters: { ...preferencesOf().display_filters, group_by: "state_id", order_by: "-state_name" },
  });
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, initial));
  await read.promise;
  expect(store.getIssueFilters("A")?.displayFilters).toMatchObject({ group_by: "state", order_by: "-state__name" });
  const patch = await requestedIssue(
    nerve,
    () =>
      store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_FILTERS, {
        group_by: "labels",
        order_by: "assignees__first_name",
      }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  const display = { ...initial.display_filters, group_by: "label_id", order_by: "assignee_name" };
  expect(patch.call.body).toEqual({ display_filters: display });
  patch.call.answer(json(200, preferencesOf({ display_filters: display })));
  await patch.promise;
  expect(store.getIssueFilters("A")?.displayFilters).toMatchObject({
    group_by: "labels",
    order_by: "assignees__first_name",
  });
});

it("rejects a changed session without sending or changing the confirmed preferences", async () => {
  const { app, nerve, api } = tab;
  const store = app.issue.projectIssuesFilter;
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, preferencesOf()));
  await read.promise;
  FakeNerve.replaceSession(api);
  await expect(
    store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_PROPERTIES, { state: false })
  ).rejects.toBeDefined();
  expect(nerve.calls).toHaveLength(1);
  expect(store.getIssueFilters("A")?.displayProperties?.state).toBe(true);
});

it("replaces filters whole instead of retaining removed conditions", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const initial = preferencesOf({ filters: { priority: ["high"] } });
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, initial));
  await read.promise;
  const patch = await requestedIssue(
    nerve,
    () => store.updateFilterExpression("ws", "A", { and: [{ state_id__in: "019a0000-0000-7000-8000-000000000001" }] }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  expect(patch.call.body).toEqual({ filters: { state_id: ["019a0000-0000-7000-8000-000000000001"] } });
  patch.call.answer(json(200, preferencesOf({ filters: { state_id: ["019a0000-0000-7000-8000-000000000001"] } })));
  await patch.promise;
  expect(store.getIssueFilters("A")?.richFilters).toEqual({
    and: [{ state_id__in: "019a0000-0000-7000-8000-000000000001" }],
  });
});

it("F01 saves and reads subscriber/date AND filters through UI and the actual list query, then clears them", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, preferencesOf()));
  await read.promise;
  const filters = {
    subscriber_id: ["me"],
    label_id: ["none", "019a0000-0000-7000-8000-000000000001"],
    target_date: ["2026-10-01..", "..2026-10-31"],
  };
  const patch = await requestedIssue(
    nerve,
    () =>
      store.updateFilterExpression("ws", "A", {
        and: [
          { subscriber_id__in: "me" },
          { label_id__in: "none,019a0000-0000-7000-8000-000000000001" },
          { target_date__range: "2026-10-01," },
          { target_date__range: ",2026-10-31" },
        ],
      }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  expect(patch.call.body).toEqual({ filters });
  patch.call.answer(json(200, preferencesOf({ filters })));
  await patch.promise;
  const reload = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  reload.call.answer(json(200, preferencesOf({ filters })));
  await reload.promise;
  const internal = workItemFiltersAdapter.toInternal(store.getIssueFilters("A")!.richFilters);
  if (!internal) throw new Error("Missing UI expression");
  expect(toProjectFilters(workItemFiltersAdapter.toExternal(internal))).toEqual(filters);
  const list = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(list.call.query).toMatchObject({ subscriber_id: "me", label_id: "none,019a0000-0000-7000-8000-000000000001" });
  expect(list.call.queryEntries.filter(([key]) => key === "target_date")).toEqual([
    ["target_date", "2026-10-01.."],
    ["target_date", "..2026-10-31"],
  ]);
  list.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
  await list.promise;
  const clear = await requestedIssue(nerve, () => store.updateFilterExpression("ws", "A", {}), [
    "PATCH",
    "/api/v0/me/projects/A/preferences",
  ]);
  expect(clear.call.body).toEqual({ filters: {} });
  clear.call.answer(json(200, preferencesOf()));
  await clear.promise;
  expect(store.getIssueFilters("A")?.richFilters).toEqual({});
  const cleared = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(cleared.call.query).toEqual({ limit: "20", order_by: "-created_at", sub_issue: "true" });
  cleared.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
  await cleared.promise;
});

it("fix1 I1/I3 canonicalizes saved UUIDs and clears year zero on real preference write/read/query", async () => {
  const { app, nerve } = tab;
  const store = app.issue.projectIssuesFilter;
  const lower = "019a0000-0000-7000-8000-000000000001";
  const upper = "019A0000-0000-7000-8000-000000000001";
  const filters = {
    state_id: [lower],
    label_id: ["none", lower],
    assignee_id: ["me", lower],
    created_by_id: [lower],
    mention_id: [lower],
    subscriber_id: [lower],
    priority: ["high"],
  };
  const read = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  read.call.answer(json(200, preferencesOf()));
  await read.promise;
  const patch = await requestedIssue(
    nerve,
    () =>
      store.updateFilterExpression("ws", "A", {
        and: [
          { state_id__in: upper },
          { label_id__in: `none,${upper}` },
          { assignee_id__in: `me,${upper}` },
          { created_by_id__in: upper },
          { mention_id__in: upper },
          { subscriber_id__in: upper },
          { start_date__exact: "0000-01-01" },
          { target_date__range: "0000-01-01," },
          { priority__in: "high" },
        ],
      }),
    ["PATCH", "/api/v0/me/projects/A/preferences"]
  );
  expect.soft(patch.call.body).toEqual({ filters });
  patch.call.answer(json(200, preferencesOf({ filters })));
  await patch.promise;
  const reload = await requestedIssue(nerve, () => store.fetchFilters("ws", "A"), [
    "GET",
    "/api/v0/me/projects/A/preferences",
  ]);
  reload.call.answer(
    json(200, preferencesOf({ filters: { ...filters, state_id: [upper], start_date: ["0000-01-01"] } }))
  );
  await reload.promise;
  expect.soft(store.getIssueFilters("A")?.richFilters).toEqual({
    and: [
      { state_id__in: lower },
      { priority__in: "high" },
      { label_id__in: `none,${lower}` },
      { assignee_id__in: `me,${lower}` },
      { created_by_id__in: lower },
      { mention_id__in: lower },
      { subscriber_id__in: lower },
    ],
  });
  const list = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(list.call.query).toEqual({
    limit: "20",
    order_by: "-created_at",
    sub_issue: "true",
    state_id: lower,
    priority: "high",
    label_id: `none,${lower}`,
    assignee_id: `me,${lower}`,
    created_by_id: lower,
    mention_id: lower,
    subscriber_id: lower,
  });
  list.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
  await list.promise;
});
