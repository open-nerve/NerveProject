// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation, useNavigationType, useSearchParams } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { useIntakeLanding, type IntakeLanding } from "./landing";

// Where the intake page's focus lands as its selection leaves a work item (P5a spec 2.8, R14, R35): with the list on
// the screen, on the row selected, else on the list's region; with the list hidden (a narrow screen, a row pressed),
// on what the screen shows instead, never on a row or a list it does not: the detail's region around the work item
// selected, else, nothing selected nor to be, the list's toggle; nothing selected but a first row the page selects
// next, on that one's detail once it shows it. The page here is a stand-in: the list's rows, the detail's region while
// a work item is selected, the toggle while none is, and the page's own selection of its first row.

const held = {
  landing: undefined as IntakeLanding | undefined,
  list: true,
  first: undefined as string | undefined,
  where: undefined as { search: string; type: string } | undefined,
};
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

function Page() {
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  held.where = { search: location.search, type: useNavigationType() };
  const selected = params.get("inboxIssueId") ?? undefined;
  const landing = useIntakeLanding({ list: held.list, first: held.first });
  held.landing = landing;
  // the page selects its first row once nothing is (IN/root.tsx), in a later commit than the one that showed nothing
  // selected, as a browser's router does (a transition): here, a task later
  useEffect(() => {
    if (selected !== undefined || held.first === undefined) return;
    const first = held.first;
    const later = setTimeout(() => setParams({ inboxIssueId: first }, { replace: true }));
    return () => clearTimeout(later);
  }, [selected, setParams]);
  return (
    <>
      <section ref={landing.list} tabIndex={-1} aria-label="list">
        {["i-a", "i-b", "i-c"].map((id) => (
          <a key={id} href={`#${id}`} aria-label={id} aria-current={id === selected ? "true" : undefined}>
            {id}
          </a>
        ))}
      </section>
      {selected === undefined ? (
        <button ref={landing.toggle} type="button" aria-label="toggle" />
      ) : (
        <section ref={landing.detail} tabIndex={-1} aria-label="detail" data-selected={selected} />
      )}
    </>
  );
}

let host: HTMLDivElement, root: Root;
beforeEach(() => {
  held.list = true;
  held.first = undefined;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

async function render(selected: string) {
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={[`/acme/projects/p-web/intake?inboxIssueId=${selected}`]}>
        <Routes>
          <Route path="/:workspaceSlug/projects/:projectId/intake" element={<Page />} />
        </Routes>
      </MemoryRouter>
    )
  );
}
/** Leaves from for to, as an action's end does, and lets the page settle: its microtasks, then its selection of a row. */
async function leave(from: string, to: string | undefined) {
  await act(async () => held.landing!.leave(from, to));
  await act(async () => {
    await Promise.resolve();
  });
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve));
  });
  await act(async () => {
    await Promise.resolve();
  });
}
const selected = () => new URLSearchParams(held.where!.search).get("inboxIssueId");
/** What has the focus: the page's body, or an element's name. */
const focused = () =>
  document.activeElement === document.body ? "body" : document.activeElement?.getAttribute("aria-label");

describe("useIntakeLanding, the list on the screen", () => {
  it("lands on the row the selection leaves for, by a replace", async () => {
    await render("i-b");
    await leave("i-b", "i-c");
    expect([selected(), held.where!.type, focused()]).toEqual(["i-c", "REPLACE", "i-c"]);
  });

  it("lands on the list's region as the selection is cleared, a first row selected after", async () => {
    held.first = "i-a";
    await render("i-b");
    await leave("i-b", undefined);
    expect([selected(), focused()]).toEqual(["i-a", "list"]);
  });
});

describe("useIntakeLanding, the list hidden (a narrow screen)", () => {
  it("lands on the detail's region around the work item the selection leaves for, never on its hidden row", async () => {
    held.list = false;
    await render("i-b");
    await leave("i-b", "i-c");
    expect([selected(), focused()]).toEqual(["i-c", "detail"]);
    expect(document.activeElement?.getAttribute("data-selected")).toBe("i-c");
  });

  it("cleared, a first row the page selects next: lands on that one's detail once shown, not on the toggle it removes", async () => {
    held.list = false;
    held.first = "i-a";
    await render("i-b");
    await leave("i-b", undefined);
    expect([selected(), focused()]).toEqual(["i-a", "detail"]);
    expect(document.activeElement?.getAttribute("data-selected")).toBe("i-a");
  });

  it("cleared, no row left to select: lands on the list's toggle, never on the hidden list", async () => {
    held.list = false;
    await render("i-b");
    await leave("i-b", undefined);
    expect([selected(), focused()]).toEqual([null, "toggle"]);
  });
});
