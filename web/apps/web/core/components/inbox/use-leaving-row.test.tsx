// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { useIntakeLanding } from "./landing";
import { useLeavingRow } from "./use-leaving-row";

// Where the focus goes as a row of the intake page's list that has it leaves the list (P5a spec 2.8, R35; P4's
// useLeavingFocus): to the next row, else the previous, else the list's region, while the list is on the screen; to
// the detail's region while it is hidden (useIntakeLanding's land); never to the page's body. A row that had the focus
// and lost it to something else moves it not as it leaves, nor one that lost it to nothing once something else has
// it; nor does a row that never had it. The page here is a
// stand-in: the list's rows, which the test changes, beside the detail's region and a button outside the list.

const held = { list: true, setRows: undefined as ((rows: string[]) => void) | undefined };
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

function Page() {
  const [rows, setRows] = useState(["i-a", "i-b", "i-c"]);
  held.setRows = setRows;
  const landing = useIntakeLanding({ list: held.list, first: rows[0] });
  const leaving = useLeavingRow(rows, landing.land);
  return (
    <>
      <div onFocus={leaving.onFocus} onBlur={leaving.onBlur}>
        <button type="button" aria-label="filters" />
        <section ref={landing.list} tabIndex={-1} aria-label="list">
          {rows.map((id) => (
            <a key={id} href={`#${id}`} aria-label={id} data-intake-row={id}>
              {id}
            </a>
          ))}
        </section>
      </div>
      <section ref={landing.detail} tabIndex={-1} aria-label="detail" />
    </>
  );
}

let host: HTMLDivElement, root: Root;
beforeEach(async () => {
  held.list = true;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/acme/projects/p-web/intake?inboxIssueId=i-a"]}>
        <Routes>
          <Route path="/:workspaceSlug/projects/:projectId/intake" element={<Page />} />
        </Routes>
      </MemoryRouter>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const element = (name: string) => host.querySelector<HTMLElement>(`[aria-label="${name}"]`)!;
const focus = (name: string) => act(async () => element(name).focus());
/** Shows rows in the list, as a confirmation does, and lets its microtasks run. */
async function show(rows: string[]) {
  await act(async () => held.setRows!(rows));
  await act(async () => {
    await Promise.resolve();
  });
}
/** What has the focus: the page's body, or an element's name. */
const focused = () =>
  document.activeElement === document.body ? "body" : document.activeElement?.getAttribute("aria-label");

describe("useLeavingRow, the list on the screen", () => {
  it("a row with the focus leaves: the next row has it", async () => {
    await focus("i-b");
    await show(["i-a", "i-c"]);
    expect(focused()).toBe("i-c");
  });

  it("the last row with the focus leaves: the previous row has it", async () => {
    await focus("i-c");
    await show(["i-a", "i-b"]);
    expect(focused()).toBe("i-b");
  });

  it("the only row with the focus leaves: the list's region has it", async () => {
    await show(["i-b"]);
    await focus("i-b");
    await show([]);
    expect(focused()).toBe("list");
  });

  it("a row that lost the focus to something else leaves: the focus stays there", async () => {
    await focus("i-b");
    await focus("filters");
    await show(["i-a", "i-c"]);
    expect(focused()).toBe("filters");
  });

  it("a row that lost the focus to nothing, then something outside the list had it, leaves: the focus stays there", async () => {
    await focus("i-b");
    await act(async () => element("i-b").blur());
    await focus("detail");
    await show(["i-a", "i-c"]);
    expect(focused()).toBe("detail");
  });

  it("a row without the focus leaves: the focus stays on its row", async () => {
    await focus("i-a");
    await show(["i-a", "i-c"]);
    expect(focused()).toBe("i-a");
  });
});

describe("useLeavingRow, the list hidden (a narrow screen)", () => {
  it("a row with the focus leaves: the detail's region has it, never a row the screen does not show", async () => {
    held.list = false;
    await act(async () => held.setRows!(["i-a", "i-b", "i-c"]));
    await focus("i-b");
    await show(["i-a", "i-c"]);
    expect(focused()).toBe("detail");
  });
});
