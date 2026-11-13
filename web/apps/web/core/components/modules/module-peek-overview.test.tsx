// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, useLocation, type Location } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { refusal } from "@/lib/fake-refusal";
import { listedModules } from "./fake-module-lists";
import { ModulePeekOverview } from "./module-peek-overview";

// The preview of a module on its project's list and archive (P6a spec 2.6, R3, R12): ?peekModule names it; the preview
// reads the module, its progress and its links by the session's store, for a caller who reads the project's modules
// (usePlanningReads' readable: an archived module's alike, nerve reads it, and whether the modules are turned on or
// not), and shows its progress sidebar, or why nerve could not read one of them, with a retry. Closing it takes it out
// of the address. The modules are a stand-in store's; the gate a stand-in; the sidebar a stand-in that says whose it
// is and closes it.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("./fake-module-lists").listedModules>,
  readable: true,
  gated: [] as string[],
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/use-planning-reads", () => ({
  usePlanningReads: (workspaceSlug: string, projectId: string, domain: string) => {
    held.gated.push(`${domain} of ${workspaceSlug}/${projectId}`);
    return { readable: held.readable, listed: held.readable };
  },
}));
vi.mock("./progress-sidebar", () => ({
  ModuleDetailsSidebar: (props: { moduleId: string; projectId: string; handleClose: () => void }) => (
    <aside aria-label={`${props.moduleId} of ${props.projectId}`}>
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
  Object.assign(held, { modules: listedModules(), readable: true, gated: [] });
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
        <MemoryRouter initialEntries={[`/acme/projects/p-web/modules${search}`]}>
          <ModulePeekOverview workspaceSlug="acme" projectId="p-web" />
          <Where />
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
const sidebar = () => host.querySelector("aside")?.getAttribute("aria-label");
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);

it("reads the module it previews, its progress and its links by the session's store, and shows its sidebar", async () => {
  await previewing("?peekModule=m-auth");
  expect(held.gated).toContain("modules of acme/p-web");
  expect(held.modules.asked()).toEqual(["module of m-auth", "progress of m-auth", "links of m-auth"]);
  expect(sidebar()).toBe("m-auth of p-web");
});

it("reads and shows nothing for a caller who does not read the project's modules", async () => {
  held.readable = false;
  await previewing("?peekModule=m-auth");
  expect(held.modules.asked()).toEqual([]);
  expect(host.innerHTML).toBe("");
});

it("reads and shows nothing without a module to preview", async () => {
  await previewing("");
  expect(held.modules.asked()).toEqual([]);
  expect(host.innerHTML).toBe("");
});

it("closes by the sidebar's close, the address keeping the rest of its query", async () => {
  await previewing("?layout=list&peekModule=m-auth");
  await act(async () => button("close")?.click());
  expect(location?.search).toBe("?layout=list");
  expect(sidebar()).toBeUndefined();
});

it.each(["module", "progress", "links"])(
  "says why nerve could not read the module's %s, waiting no more, and reads it again on retry",
  async (what) => {
    await previewing("?peekModule=m-gone");
    await held.modules.refuse(what, refusal(503, "server_busy"));
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
    expect(sidebar()).toBeUndefined();
    await act(async () => button("common.retry")?.click());
    expect(held.modules.asked()).toEqual([
      "module of m-gone",
      "progress of m-gone",
      "links of m-gone",
      `${what} of m-gone`,
    ]);
    await held.modules.answer(what, what === "links" ? [] : {});
    expect(host.querySelector('[role="alert"]')).toBeNull();
    expect(sidebar()).toBe("m-gone of p-web");
  }
);

it("closes while it says nerve does not find the module", async () => {
  await previewing("?peekModule=m-gone");
  await held.modules.refuse("module", refusal(404, "planning.module_not_found"));
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.planning_module_not_found");
  await act(async () => button("close")?.click());
  expect(location?.search).toBe("");
  expect(host.innerHTML).toBe("");
});
