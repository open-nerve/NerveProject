/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { FakeNerve, json, noContent, problem, type Endpoint } from "@/lib/auth/fake-nerve";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { draftOf, issueDetailOf, requestedIssue } from "@/store/issue/fake-issues";
import { WorkspaceDraftService } from "./workspace_draft.service";
const cases: {
  name: string;
  endpoint: Endpoint;
  run: (s: WorkspaceDraftService) => Promise<unknown>;
  answer?: unknown;
  body?: unknown;
  query?: [string, string][];
  status: number;
}[] = [
  {
    name: "list",
    endpoint: ["GET", "/api/v0/workspaces/ws/drafts"],
    run: (s) => s.list("ws", { cursor: "opaque+/=", limit: 3 }),
    query: [
      ["cursor", "opaque+/="],
      ["limit", "3"],
    ],
    answer: { data: [draftOf("d")], total: 80, next_cursor: "next" },
    status: 200,
  },
  {
    name: "create",
    endpoint: ["POST", "/api/v0/workspaces/ws/drafts"],
    run: (s) => s.create("ws", { project_id: null, payload: {} }),
    body: { project_id: null, payload: {} },
    answer: draftOf("d"),
    status: 201,
  },
  {
    name: "save",
    endpoint: ["PATCH", "/api/v0/drafts/d"],
    run: (s) => s.update("d", { payload: { name: "" } }),
    body: { payload: { name: "" } },
    answer: draftOf("d"),
    status: 200,
  },
  { name: "delete", endpoint: ["DELETE", "/api/v0/drafts/d"], run: (s) => s.delete("d"), status: 204 },
  {
    name: "publish",
    endpoint: ["POST", "/api/v0/drafts/d/publish"],
    run: (s) => s.publish("d"),
    answer: issueDetailOf("issue"),
    status: 201,
  },
];
beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());
it.each(cases)("native draft transport: $name", async ({ endpoint, run, body, query, answer, status }) => {
  const nerve = new FakeNerve();
  const sent = await requestedIssue(nerve, () => run(new WorkspaceDraftService(nerve.client())), endpoint);
  expect(sent.call.body).toEqual(body);
  expect(sent.call.queryEntries).toEqual(query ?? []);
  sent.call.answer(status === 204 ? noContent() : json(status, answer));
  expect(await sent.promise).toEqual(answer);
});
it.each(
  cases.flatMap((row) =>
    [403, 404, 409, 422].map((status) => ({ name: row.name, run: row.run, endpoint: row.endpoint, status }))
  )
)("keeps native refusal $name/$status", async ({ run, endpoint, status }) => {
  const nerve = new FakeNerve();
  const sent = await requestedIssue(nerve, () => run(new WorkspaceDraftService(nerve.client())), endpoint);
  sent.call.answer(problem(status, "draft.original"));
  await expect(sent.promise).rejects.toMatchObject({ status, problem: { code: "draft.original" } });
});
it.each(cases)("does not change session retirement: $name", async ({ run }) => {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const reason = new SessionChangedError();
  api.use({
    onRequest() {
      throw reason;
    },
  });
  await expect(run(new WorkspaceDraftService(api))).rejects.toBe(reason);
  expect(nerve.calls).toHaveLength(0);
});
