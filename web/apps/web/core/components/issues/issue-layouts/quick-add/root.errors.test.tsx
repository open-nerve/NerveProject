// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssueLayoutTypes } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { issueStoreTab, issueDetailOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { QuickAddIssueRoot } from "./root";
const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("../../create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
class NoResize {
  observe() {}
  unobserve() {}
  disconnect() {}
}
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
let away: HTMLButtonElement;
let writes: Promise<unknown>[];
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal("ResizeObserver", NoResize);
  tab = await issueStoreTab();
  bound.app = tab.app;
  toasts.length = 0;
  writes = [];
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
  away = document.createElement("button");
  away.textContent = "Other control";
  document.body.append(away);
});
afterEach(async () => {
  await act(async () => view.unmount());
  tab.app.dispose();
  host.remove();
  away.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const create: NonNullable<ComponentProps<typeof QuickAddIssueRoot>["quickAddCallback"]> = (project, body) => {
  const result = tab.app.issue.createIssue(tab.app.issue.scope("ws", project!), body);
  writes.push(result);
  return result;
};
const mount = (props: Partial<ComponentProps<typeof QuickAddIssueRoot>> = {}) =>
  act(async () =>
    view.render(
      <StoreContext.Provider value={tab.app}>
        <QuickAddIssueRoot isQuickAddOpen layout={EIssueLayoutTypes.LIST} quickAddCallback={create} {...props} />
      </StoreContext.Provider>
    )
  );
const input = () => host.querySelector("input")!;
const submit = () =>
  host.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
async function type(value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input(), value);
    input().dispatchEvent(new Event("input", { bubbles: true }));
  });
}
function described(control: HTMLInputElement) {
  return (control.getAttribute("aria-describedby")?.split(" ") ?? [])
    .map((id) => document.getElementById(id)?.textContent ?? "")
    .join(" ");
}
async function focusEvents() {
  const events: { invalid: string | null; description: string }[] = [];
  input().addEventListener("focus", () =>
    events.push({ invalid: input().getAttribute("aria-invalid"), description: described(input()) })
  );
  await act(async () => away.focus());
  return events;
}
async function answer(response: Response) {
  await act(async () => {
    tab.nerve.calls.at(-1)!.answer(response);
    await Promise.allSettled(writes);
  });
}
it.each(Object.values(EIssueLayoutTypes))(
  "%s blank name is described before focus and valid editing clears it",
  async (layout) => {
    await mount({ layout });
    await type("   ");
    const events = await focusEvents();
    await act(async () => submit());
    expect(tab.nerve.calls).toHaveLength(0);
    expect(events).toEqual([{ invalid: "true", description: "issue.title.required" }]);
    expect(document.activeElement).toBe(input());
    await type("Real name");
    expect(input().getAttribute("aria-invalid")).not.toBe("true");
    expect(described(input())).toBe("");
  }
);
it("server field refusal is described before focus; success resets and focuses only after pending ends", async () => {
  await mount();
  await type("Sent name");
  await act(async () => submit());
  await until(() => tab.nerve.calls.length === 1, "create");
  const events = await focusEvents();
  await answer(
    json(422, { status: 422, code: "validation_failed", errors: [{ field: "name", code: "too_long", message: "" }] })
  );
  expect(events).toEqual([{ invalid: "true", description: "errors.field.too_long" }]);
  expect(input().value).toBe("Sent name");
  expect(input().matches(":disabled")).toBe(false);
  expect(toasts).toHaveLength(0);
  await type("Short");
  await act(async () => submit());
  await until(() => tab.nerve.calls.length === 2, "retry");
  await act(async () => away.focus());
  await answer(json(201, { ...issueDetailOf("created"), name: "Short" }));
  expect(input().value).toBe("");
  expect(document.activeElement).toBe(input());
  expect(input().matches(":disabled")).toBe(false);
  expect(toasts.map((t) => t.type)).toEqual(["success"]);
});
it("permission refusal uses the problem message without a false name error", async () => {
  await mount();
  await type("Keep me");
  await act(async () => submit());
  await until(() => tab.nerve.calls.length === 1, "create");
  await answer(problem(403, "forbidden"));
  expect(input().getAttribute("aria-invalid")).not.toBe("true");
  expect(input().value).toBe("Keep me");
  expect(toasts.map((t) => t.message)).toEqual(["errors.forbidden"]);
});
it.each([{ isComposing: true }, { isComposing: false, keyCode: 229 }])(
  "IME Enter stays in composition %s",
  async (event) => {
    await mount();
    await type("输入中");
    const key = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true, ...event });
    await act(async () => input().dispatchEvent(key));
    expect(key.defaultPrevented).toBe(true);
    expect(tab.nerve.calls).toHaveLength(0);
    expect(input().value).toBe("输入中");
  }
);
it.each(["close", "group", "same-prefill-group", "permission"] as const)(
  "%s change invalidates old submit followers, including permission ABA",
  async (change) => {
    await mount({ scopeKey: "labels:none", prePopulatedData: { state_id: "old" } });
    await type("Old owner");
    await act(async () => submit());
    await until(() => tab.nerve.calls.length === 1, "old create");
    if (change === "permission") {
      await act(async () => tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: 5 })));
      expect(host.querySelector("form")).toBeNull();
      await act(async () =>
        tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: 15 }))
      );
    } else
      await mount(
        change === "close"
          ? { isQuickAddOpen: false }
          : change === "same-prefill-group"
            ? { scopeKey: "assignees:none", prePopulatedData: { state_id: "old" } }
            : { prePopulatedData: { state_id: "new" } }
      );
    await answer(json(201, issueDetailOf("old-confirmed")));
    expect(toasts).toHaveLength(0);
    expect(tab.nerve.calls).toHaveLength(1);
    if (change === "close") expect(host.querySelector("form")).toBeNull();
    else expect(input().value).toBe("");
  }
);
