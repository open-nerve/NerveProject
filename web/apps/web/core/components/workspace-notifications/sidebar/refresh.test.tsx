// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { acme, countOf, notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import { answer, cards, listRead, noteOf, renderList, settled } from "../fake-notification-list";
import { NotificationHeaderOptions } from "./header-options";

// The head's Refresh over the list it refreshes (P4 spec 2.4, 2.6, R7): the head and the list read the list by one
// key, one read as the page opens; Refresh reads the first page again through the store and gives the key that answer,
// so that a list whose read nerve refused shows the page once Refresh has it, the failure gone, with no read more; a
// refused Refresh says why and leaves the list as it was. The store and HTTP are real; SWR is, with the test's cache.

const fixture = vi.hoisted(() => ({
  session: { loginId: "one" },
  tab: undefined as unknown as Awaited<
    ReturnType<typeof import("@/store/notification/fake-notifications").notificationTab>
  >,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: fixture.session }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({
  useSession: () => ({ status: "signed-in", loginId: fixture.session.loginId }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => fixture.tab.workspaceRoot }));
vi.mock("@/hooks/store/use-notifications", () => ({ useNotifications: () => fixture.tab.store }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));

let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date("2026-11-20T12:00:00Z") });
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  toasts.length = 0;
  fixture.session.loginId = "one";
  fixture.tab = await notificationTab();
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

const nerve = () => fixture.tab.nerve;
const LIST = notificationRequests.list(acme);
const COUNT = notificationRequests.unreadCount(acme);
const failure = () => host.querySelector('[role="alert"]');
const refresh = () => host.querySelector<HTMLButtonElement>('button[aria-label="notification.options.refresh"]');

/** The list and the head's buttons at acme's address; nerve refuses the list's first read. */
async function refusedFirstRead() {
  await renderList(root, cache, <NotificationHeaderOptions />);
  (await listRead(nerve(), 0)).answer(problem(503, "server_busy"));
  await settled();
  expect(nerve().to(LIST)).toHaveLength(1);
  expect(failure()?.textContent).toContain("errors.server_busy");
}

describe("NotificationHeaderOptions' refresh, over the list it refreshes", () => {
  it("shows the first page once Refresh has it, the list's read failure gone, with no read more", async () => {
    await refusedFirstRead();
    await act(async () => refresh()?.click());
    const again = await listRead(nerve(), 1);
    await act(async () => until(() => nerve().to(COUNT).length === 1, "the count"));
    nerve()
      .to(COUNT)[0]
      ?.answer(json(200, countOf(1, 0)));
    await answer(again, pageOf([noteOf("One", 1)]));
    expect(failure()).toBeNull();
    expect(cards(host)).toEqual(["WEB-1 One"]);
    await settled();
    expect(nerve().to(LIST)).toHaveLength(2);
    expect(toasts).toEqual([]);
  });

  it("says why nerve refused Refresh, and the list's failure stays", async () => {
    await refusedFirstRead();
    await act(async () => refresh()?.click());
    (await listRead(nerve(), 1)).answer(problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(failure()?.textContent).toContain("errors.server_busy");
    expect(cards(host)).toEqual([]);
  });
});
