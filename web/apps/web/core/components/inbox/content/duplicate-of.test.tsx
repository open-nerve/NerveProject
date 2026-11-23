// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import type { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";
import { acme, detailOf, loadProjectList, writesTab } from "@/store/issue/intake/fake-intake";
import { IntakeDuplicateOf } from "./duplicate-of";

// The work item a work item of a project's intake duplicates (P5a spec 2.5, 2.9, R17; re-94 m3): read once, by the
// read a work item detail makes of its parent (fetchIssueCore), keyed by the caller's permissions, and named by the
// project's identifier, its number and its name, linked to it. Nerve refusing that read (the work item gone, or not
// the caller's to see) leaves the row out, and leaves the project's held lists as they were: no error of that read,
// no cell to read again (a verification, refreshAuthoritative, would give them both). The stores, SWR and the router
// are real; nerve is a fake that answers each request as a test says.

const fixture = vi.hoisted(() => ({ fetches: [] as unknown[] }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/lib/use-session-swr", async (original) => {
  const real = await original<typeof import("@/lib/use-session-swr")>();
  return {
    ...real,
    useSessionSWR: (...args: Parameters<typeof real.useSessionSWR>) => {
      fixture.fetches.push(args[0]);
      return real.useSessionSWR(...args);
    },
  };
});

let tab: Awaited<ReturnType<typeof writesTab>>;
let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
/** The address the page is at. */
let path: string | undefined;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  fixture.fetches = [];
  tab = await writesTab();
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

function Where() {
  path = useLocation().pathname;
  return null;
}
/** The row of the work item that duplicateToId names, in p-web's intake page. */
async function shown(duplicateToId: string | null) {
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
        <MemoryRouter initialEntries={["/acme/projects/p-web/intake"]}>
          <Routes>
            <Route
              path="*"
              element={
                <>
                  <IntakeDuplicateOf workspaceSlug="acme" projectId="p-web" duplicateToId={duplicateToId} />
                  <Where />
                </>
              }
            />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
const requests = () => tab.nerve.calls.map((call) => [call.method, call.path]);
const READ = ["GET", "/api/v0/issues/i-t"];
const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));
/** The read of the work item duplicated, out as the row is drawn, and the only request. */
async function readOut() {
  await settled();
  expect(requests()).toEqual([READ]);
  return tab.nerve.calls[0]!;
}
const link = () => host.querySelector("a");
/** What a held list shows: its error, and whether any of its cells is stale (to be read again). */
const listState = (list: IssueListBinding) => ({
  error: list.error,
  stale: [...(list.value?.groups.values() ?? [])].some(
    (group) => !!group.stale || [...(group.subgroups?.values() ?? [])].some((cell) => !!cell.stale)
  ),
});

it("reads the work item duplicated once, keyed by the caller's permissions, and names it by its number and name, linked", async () => {
  await shown("i-t");
  const call = await readOut();
  expect(link(), "nothing while it is read").toBeNull();
  const identity = tab.app.issue.permissionReads.identity(acme.id, ["p-web"], true);
  expect(fixture.fetches).toContainEqual(["intake-duplicate", identity, "acme", "p-web", "i-t"]);
  call.answer(json(200, detailOf("t", { intake: null, sequence_id: 12, name: "Original" })));
  await settled();
  expect(host.textContent).toContain("issue.relation.duplicate");
  expect(link()?.textContent).toBe("WEB-12 · Original");
  expect(link()?.getAttribute("href")).toBe("/acme/browse/WEB-12");
  expect(requests(), "one read").toEqual([READ]);
  await act(async () => link()!.click());
  expect(path).toBe("/acme/browse/WEB-12");
});

it.each([
  { status: 403, code: "forbidden" },
  { status: 404, code: "issue.not_found" },
  { status: 500, code: "server.internal" },
])(
  "leaves the row out once nerve refuses the read ($status), and a held project list without its error or a stale cell",
  async ({ status, code }) => {
    // p-web's work items, i-t among them, held as a project list
    const list = await loadProjectList(tab, {}, issuePageOf([issueOf("i-t", "p-web", acme.id)]));
    tab.nerve.calls.length = 0;
    const before = listState(list);
    expect(before).toEqual({ error: undefined, stale: false });
    await shown("i-t");
    (await readOut()).answer(problem(status, code));
    await settled();
    expect(host.textContent).toBe("");
    expect(listState(list)).toEqual(before);
    expect(requests(), "no read of the list again").toEqual([READ]);
  }
);

it("takes the row away once a later read of the work item duplicated is refused, it since gone", async () => {
  await shown("i-t");
  (await readOut()).answer(json(200, detailOf("t", { intake: null, sequence_id: 12, name: "Original" })));
  await settled();
  expect(link()?.textContent).toBe("WEB-12 · Original");
  // drawn again later, as another selection's: it reads the work item again
  await act(async () => root.render(<></>));
  await act(async () => void (await vi.advanceTimersByTimeAsync(5_000)));
  await shown("i-t");
  await settled();
  expect(requests()).toEqual([READ, READ]);
  tab.nerve.calls[1]!.answer(problem(404, "issue.not_found"));
  await settled();
  expect(host.textContent).toBe("");
});

it("reads nothing and draws nothing for a work item that duplicates none", async () => {
  await shown(null);
  await settled();
  expect(fixture.fetches).toContain(null);
  expect(requests()).toEqual([]);
  expect(host.textContent).toBe("");
});
