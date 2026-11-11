// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, useLocation, type Location } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { refusal } from "@/lib/fake-refusal";
import { listedCycles } from "./fake-cycle-lists";
import { CyclePeekOverview } from "./cycle-peek-overview";

// The preview of a cycle on its project's list and archive (P5a spec 2.5, R11): ?peekCycle names it; the preview reads
// the cycle and its progress by the session's store (the reads of a cycle's page, an archived one's alike: nerve
// reads archived cycles) and shows its progress sidebar, or why nerve could not read it, with a retry. The cycles are
// a stand-in store's; the sidebar a stand-in that says whose it is and closes it.

const held = vi.hoisted(() => ({ cycles: undefined as unknown as ReturnType<typeof listedCycles> }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("./progress-sidebar", () => ({
  CycleDetailsSidebar: (props: { cycleId: string; projectId: string; handleClose: () => void }) => (
    <aside aria-label={`${props.cycleId} of ${props.projectId}`}>
      <button type="button" onClick={props.handleClose}>
        close
      </button>
    </aside>
  ),
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
let location: Location | undefined;
function Where() {
  location = useLocation();
  return null;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  held.cycles = listedCycles();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

async function previewing(search: string) {
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/acme/projects/p-web/cycles${search}`]}>
          <CyclePeekOverview workspaceSlug="acme" projectId="p-web" />
          <Where />
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
const sidebar = () => host.querySelector("aside")?.getAttribute("aria-label");
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);

it("reads the cycle it previews and its progress by the session's store, and shows the cycle's sidebar", async () => {
  await previewing("?peekCycle=cold");
  expect(held.cycles.asked()).toEqual(["cycle of cold", "progress of cold"]);
  expect(sidebar()).toBe("cold of p-web");
});

it("reads and shows nothing without a cycle to preview", async () => {
  await previewing("");
  expect(held.cycles.asked()).toEqual([]);
  expect(host.innerHTML).toBe("");
});

it("closes by the sidebar's close, the address keeping the rest of its query", async () => {
  await previewing("?layout=list&peekCycle=cold");
  await act(async () => button("close")?.click());
  expect(location?.search).toBe("?layout=list");
  expect(sidebar()).toBeUndefined();
});

it.each(["cycle", "progress"])(
  "says why nerve could not read the cycle's %s, and reads it again on retry",
  async (what) => {
    await previewing("?peekCycle=gone");
    await held.cycles.refuse(what, refusal(503, "server_busy"));
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
    expect(sidebar()).toBeUndefined();
    await act(async () => button("common.retry")?.click());
    expect(held.cycles.asked()).toEqual(["cycle of gone", "progress of gone", `${what} of gone`]);
    await held.cycles.answer(what, {});
    expect(host.querySelector('[role="alert"]')).toBeNull();
    expect(sidebar()).toBe("gone of p-web");
  }
);

it("closes while it says nerve does not find the cycle", async () => {
  await previewing("?peekCycle=gone");
  await held.cycles.refuse("cycle", refusal(404, "planning.cycle_not_found"));
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.planning_cycle_not_found");
  await act(async () => button("close")?.click());
  expect(location?.search).toBe("");
  expect(host.innerHTML).toBe("");
});
