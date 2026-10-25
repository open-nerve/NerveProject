import assert from "node:assert/strict";
import { test } from "node:test";

import { createClient } from "../src/index.ts";

// Only transport is replaced: the public client's URL construction remains real.
function capture(options = {}) {
  const requests = [];
  const client = createClient({
    baseUrl: "http://nerve.test",
    ...options,
    fetch: async (request) => {
      requests.push(new URL(request.url));
      return Response.json({ total: 0, data: [], next_cursor: null });
    },
  });
  return { client, requests };
}

// Removing the route's CSV adaptation must break this real outgoing request.
test("public list client sends CSV alternatives once and repeated date conditions separately", async () => {
  const { client, requests } = capture();
  await client.GET("/api/v0/projects/{project_id}/issues", {
    params: {
      path: { project_id: "project" },
      query: { priority: ["high", "low"], start_date: ["2026-10-01..", "..2026-10-31"] },
    },
  });
  assert.deepEqual(requests[0].searchParams.getAll("priority"), ["high,low"]);
  assert.deepEqual(requests[0].searchParams.getAll("start_date"), ["2026-10-01..", "..2026-10-31"]);
});

test("both list routes serialize each CSV field without changing atoms or control parameters", async () => {
  const { client, requests } = capture();
  const query = Object.freeze({
    state_id: Object.freeze(["state-a", "state-b"]),
    state_group: Object.freeze(["started", "completed"]),
    priority: Object.freeze(["high", "none"]),
    label_id: Object.freeze(["label-a", "none"]),
    assignee_id: Object.freeze(["me", "none"]),
    created_by_id: Object.freeze(["me", "user-a"]),
    mention_id: Object.freeze(["me", "user-b"]),
    subscriber_id: Object.freeze(["user-a", "user-b"]),
    target_date: Object.freeze(["2026-10-01..", "..2026-10-31"]),
    sub_issue: "false",
    archived: "true",
    group_by: "label_id",
    sub_group_by: "priority",
    group_key: "label-a",
    sub_group_key: "high",
    order_by: "-created_at",
    cursor: "v1_+=/?&日本語",
    limit: 25,
  });
  await client.GET("/api/v0/projects/{project_id}/issues", {
    params: { path: { project_id: "project" }, query },
  });
  assert.deepEqual(Object.fromEntries(requests[0].searchParams), {
    state_id: "state-a,state-b",
    state_group: "started,completed",
    priority: "high,none",
    label_id: "label-a,none",
    assignee_id: "me,none",
    created_by_id: "me,user-a",
    mention_id: "me,user-b",
    subscriber_id: "user-a,user-b",
    target_date: "..2026-10-31",
    sub_issue: "false",
    archived: "true",
    group_by: "label_id",
    sub_group_by: "priority",
    group_key: "label-a",
    sub_group_key: "high",
    order_by: "-created_at",
    cursor: "v1_+=/?&日本語",
    limit: "25",
  });
  assert.deepEqual(requests[0].searchParams.getAll("target_date"), ["2026-10-01..", "..2026-10-31"]);
  for (const key of [
    "state_id",
    "state_group",
    "priority",
    "label_id",
    "assignee_id",
    "created_by_id",
    "mention_id",
    "subscriber_id",
  ])
    assert.equal(requests[0].searchParams.getAll(key).length, 1, key);

  await client.GET("/api/v0/workspaces/{slug}/members/{user_id}/issues", {
    params: {
      path: { slug: "acme", user_id: "member" },
      query: {
        relation: "assigned",
        project_id: ["p-a", "p-b"],
        state_group: ["started", "completed"],
        priority: ["low", "none"],
        label_id: ["label-a", "none"],
        start_date: ["2026-10-01..", "..2026-10-31"],
      },
    },
  });
  assert.deepEqual(
    [...requests[1].searchParams],
    [
      ["relation", "assigned"],
      ["project_id", "p-a,p-b"],
      ["state_group", "started,completed"],
      ["priority", "low,none"],
      ["label_id", "label-a,none"],
      ["start_date", "2026-10-01.."],
      ["start_date", "..2026-10-31"],
    ]
  );
});

test("list defaults omit empty values and preserve scalar escaping and zero", async () => {
  const { client, requests } = capture();
  await client.GET("/api/v0/projects/{project_id}/issues", {
    params: {
      path: { project_id: "project" },
      query: {
        priority: [],
        label_id: undefined,
        start_date: [],
        target_date: null,
        limit: 0,
        cursor: "",
        group_key: "a&b=日本語",
      },
    },
  });
  assert.deepEqual(
    [...requests[0].searchParams],
    [
      ["limit", "0"],
      ["cursor", ""],
      ["group_key", "a&b=日本語"],
    ]
  );
});

test("request method shares GET list defaults while other routes and methods retain existing serialization", async () => {
  const { client, requests } = capture();
  const init = { params: { path: { project_id: "project" }, query: { priority: ["high", "low"] } } };
  await client.request("get", "/api/v0/projects/{project_id}/issues", init);
  await client.POST("/api/v0/projects/{project_id}/issues", { ...init, body: { name: "new" } });
  await client.GET("/api/v0/instance", { params: { query: { priority: ["high", "low"] } } });
  assert.deepEqual(
    requests.map((url) => url.searchParams.getAll("priority")),
    [["high,low"], ["high", "low"], ["high", "low"]]
  );
});

test("explicit client and request serializer overrides remain authoritative", async () => {
  const path = "/api/v0/projects/{project_id}/issues";
  const init = { params: { path: { project_id: "project" }, query: { priority: ["high", "low"] } } };
  const { client, requests } = capture({ querySerializer: () => "custom=client" });
  await client.GET(path, init);
  await client.GET(path, { ...init, querySerializer: () => "custom=request" });
  await client.request("get", path, { ...init, querySerializer: () => "custom=generic" });
  assert.deepEqual(
    requests.map((url) => url.search),
    ["?custom=client", "?custom=request", "?custom=generic"]
  );

  const objects = capture({ querySerializer: { array: { style: "pipeDelimited", explode: false } } });
  await objects.client.GET(path, init);
  await objects.client.GET(path, { ...init, querySerializer: { array: { style: "form", explode: true } } });
  const defaults = capture();
  await defaults.client.GET(path, { ...init, querySerializer: { array: { style: "form", explode: true } } });
  assert.deepEqual(objects.requests[0].searchParams.getAll("priority"), ["high|low"]);
  assert.deepEqual(objects.requests[1].searchParams.getAll("priority"), ["high", "low"]);
  assert.deepEqual(defaults.requests[0].searchParams.getAll("priority"), ["high", "low"]);
});
