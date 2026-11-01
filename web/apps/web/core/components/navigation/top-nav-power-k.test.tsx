// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig, type SWRConfiguration } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { ShortcutHandler } from "@/components/power-k/core/shortcut-handler";
import { TopNavPowerK } from "./top-nav-power-k";

const ui = vi.hoisted(() => ({
  params: { workspaceSlug: "ws", projectId: "A" },
  navigate: vi.fn(),
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "search" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "search" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/use-power-k", () => ({ usePowerK: () => tab.app.powerK }));
// The unrelated command categories are not part of this search-host contract.
vi.mock("@/components/power-k/ui/pages", () => ({ PowerKModalPagesList: () => null }));
vi.mock("@/components/power-k/ui/pages/context-based", () => ({ PowerKContextBasedPagesList: () => null }));
vi.mock("react-router", async (original) => {
  const { useSyncExternalStore } = await import("react");
  return {
    ...(await original<typeof import("react-router")>()),
    useParams: () =>
      useSyncExternalStore(
        (notify) => {
          window.addEventListener("routechange", notify);
          return () => window.removeEventListener("routechange", notify);
        },
        () => ui.params
      ),
    useNavigate: () => ui.navigate,
  };
});
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

let tab: Awaited<ReturnType<typeof issueStoreTab>>, root: Root, host: HTMLDivElement, config: SWRConfiguration;
const scrollDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
const input = () => host.querySelector<HTMLInputElement>('input[type="text"]')!;
const searches = () => tab.nerve.calls.filter((call) => call.path.endsWith("/issue-search"));
const options = () => [...host.querySelectorAll('[role="option"]')].map((element) => element.textContent);
const tick = (ms = 0) => act(async () => vi.advanceTimersByTimeAsync(ms));
async function type(value: string) {
  await act(async () => {
    input().dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    input().focus();
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input(), value);
    input().dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const key = (value: string, details: KeyboardEventInit = {}) =>
  act(async () =>
    input().dispatchEvent(new KeyboardEvent("keydown", { key: value, bubbles: true, cancelable: true, ...details }))
  );
const answer = (id: string, project = "A", index = searches().length - 1) =>
  act(async () => searches()[index]!.answer(json(200, { data: [{ ...issueOf(id, project), sequence_id: 9 }] })));
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  // oxlint-disable-next-line no-extend-native -- jsdom lacks scrolling; actual keyboard focus is verified in I18.
  HTMLElement.prototype.scrollIntoView = vi.fn();
  ui.params = { workspaceSlug: "ws", projectId: "A" };
  ui.navigate.mockReset();
  tab = await issueStoreTab();
  const other = projectOf("B", tab.workspace.id, { id: "B" });
  tab.nerve.respond = (call) => {
    if (/workspaces\/[^/]+\/projects$/.test(call.path)) call.answer(json(200, { data: [tab.project, other] }));
    else if (call.path.endsWith("/states")) {
      const project = tab.app.projectRoot.project.getProjectById(call.path.split("/").at(-2))!;
      call.answer(json(200, { data: [stateOf(project, "Todo", "unstarted", 1, { id: "state" })] }));
    }
  };
  config = { provider: () => new Map(), dedupingInterval: 0 };
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <SWRConfig value={config}>
        <TopNavPowerK />
      </SWRConfig>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  // oxlint-disable-next-line no-extend-native -- restore the original test-environment descriptor.
  if (scrollDescriptor) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", scrollDescriptor);
  else Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

it("the top search sends its own native query while the modal is closed and keeps server matches", async () => {
  expect(tab.app.powerK.isPowerKModalOpen).toBe(false);
  await type("0009");
  await tick(500);
  expect(searches()).toHaveLength(1);
  expect(searches()[0]!.query).toEqual({ q: "0009", project_id: "A" });
  await answer("Unrelated title");
  expect(options()).toEqual(["A-9Unrelated title"]);
  await key("Enter", { isComposing: true });
  expect(ui.navigate).not.toHaveBeenCalled();
  await key("Enter", { keyCode: 229 });
  expect(ui.navigate).not.toHaveBeenCalled();
  await key("Enter");
  expect(ui.navigate).toHaveBeenCalledExactlyOnceWith("/ws/browse/A-9");
  expect(input().value).toBe("");
  expect(options()).toEqual([]);
  expect(tab.app.powerK.isPowerKModalOpen).toBe(false);
});

it("a route change closes the top host and retires its held response before reopening", async () => {
  await type("old");
  await tick(500);
  expect(searches()).toHaveLength(1);
  await act(async () => {
    ui.params = { workspaceSlug: "ws", projectId: "B" };
    window.dispatchEvent(new Event("routechange"));
  });
  expect(input().value).toBe("");
  expect(host.querySelector("[cmdk-list]")).toBeNull();
  await answer("old answer", "A", 0);
  expect(options()).toEqual([]);
  await type("old");
  await tick(500);
  expect(searches()).toHaveLength(2);
  expect(searches()[1]!.query).toEqual({ q: "old", project_id: "B" });
  await answer("current answer", "B", 1);
  expect(options()).toEqual(["B-9current answer"]);
});

it("arrow keys select later native results without moving focus out of the search input", async () => {
  await type("match");
  await tick(500);
  await act(async () =>
    searches()[0]!.answer(
      json(200, {
        data: [
          { ...issueOf("first", "A"), sequence_id: 9 },
          { ...issueOf("second", "A"), sequence_id: 10 },
        ],
      })
    )
  );
  await key("ArrowDown");
  expect(document.activeElement).toBe(input());
  expect(host.querySelector('[role="option"][aria-selected="true"]')?.textContent).toBe("A-10second");
  await key("ArrowUp");
  expect(document.activeElement).toBe(input());
  expect(host.querySelector('[role="option"][aria-selected="true"]')?.textContent).toBe("A-9first");
});

it.each(["ctrlKey", "metaKey"] as const)(
  "top search consumes closing %s+K before document opens the modal",
  async (modifier) => {
    const handler = new ShortcutHandler(
      tab.app.powerK.commandRegistry,
      () => ({
        params: ui.params,
        activeCommand: null,
        activeContext: null,
        shouldShowContextBasedActions: true,
        setShouldShowContextBasedActions: () => {},
        navigate: ui.navigate,
        closePalette: () => {},
        setActiveCommand: () => {},
        setActivePage: tab.app.powerK.setActivePage,
      }),
      () => tab.app.powerK.togglePowerKModal(true)
    );
    document.addEventListener("keydown", handler.handleKeyDown);
    try {
      await type("held");
      await key("k", { [modifier]: true });
      expect(input().value).toBe("");
      expect(tab.app.powerK.isPowerKModalOpen).toBe(false);
    } finally {
      document.removeEventListener("keydown", handler.handleKeyDown);
      handler.destroy();
    }
  }
);
