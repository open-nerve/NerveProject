// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { acme, countOf, notificationRequests, notificationTab, pageOf } from "@/store/notification/fake-notifications";
import { answer, cards, changeState, listRead, noteOf, renderList, settled } from "../fake-notification-list";
import { NotificationHeaderOptions } from "./header-options";

// The head's Refresh and mark all read over the list they read again (P4 spec 2.4, 2.6, R7, R9): the head and the list
// read the list by one key, one read as the page opens; each reads the first page again through the store and gives
// the key that answer, so that a list whose read nerve refused shows the page once it has it, the failure gone, with
// no read more; a refused Refresh says why and leaves the list as it was; mark all read reads it again whether nerve
// marked or refused, saying which. The store and HTTP are real; SWR is, with the test's cache.

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
// the card's custom snooze dialog reads the profile (DateDropdown); it is never opened here
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: undefined }) }));

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
const markAll = () =>
  host.querySelector<HTMLButtonElement>('button[aria-label="notification.options.mark_all_as_read"]');
const MARK_ALL = notificationRequests.markAllRead(acme);

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

describe("NotificationHeaderOptions' mark all read, over the list it reads again", () => {
  it.each([
    [
      "marked",
      () => json(200, { marked: 0 }),
      { type: "success", title: "toast.success", message: "notification.toasts.all_read 0" },
    ],
    [
      "refused",
      () => problem(503, "server_busy"),
      { type: "error", title: "toast.error", message: "errors.server_busy" },
    ],
  ])("shows the first page it reads again once nerve has %s, the list's read failure gone", async (_, reply, toast) => {
    await refusedFirstRead();
    await act(async () => markAll()?.click());
    await act(async () => until(() => nerve().to(MARK_ALL).length === 1, "mark all read"));
    nerve().to(MARK_ALL)[0]?.answer(reply());
    const again = await listRead(nerve(), 1);
    await act(async () => until(() => nerve().to(COUNT).length === 1, "the count"));
    nerve()
      .to(COUNT)[0]
      ?.answer(json(200, countOf(0, 0)));
    await answer(again, pageOf([noteOf("One", 1, { read_at: "2026-11-20T12:00:00Z" })]));
    expect(toasts).toEqual([toast]);
    expect(failure()).toBeNull();
    expect(cards(host)).toEqual(["WEB-1 One"]);
    await settled();
    expect(nerve().to(LIST)).toHaveLength(2);
  });
});

describe("NotificationHeaderOptions' late answers, once the page shows other filters (whole-branch m2)", () => {
  /**
   * The list and the head at acme's address, its first page read; then, once press has sent its request, the archived
   * view, whose read nerve refuses.
   */
  async function archivedViewRefusedWhile(press: () => HTMLButtonElement | null, sent: () => boolean) {
    await renderList(root, cache, <NotificationHeaderOptions />);
    await answer(await listRead(nerve(), 0), pageOf([noteOf("One", 1)]));
    await act(async () => press()?.click());
    await act(async () => until(sent, "the head's request"));
    await changeState((state) => state.toggleArchived());
    const archived = nerve()
      .to(LIST)
      .find((call) => call.query.archived === "true");
    archived?.answer(problem(503, "server_busy"));
    await settled();
    expect(failure()?.textContent).toContain("errors.server_busy");
  }

  it("mark all read reads the filters it marked again into the store alone: the archived view's failure stays", async () => {
    await archivedViewRefusedWhile(markAll, () => nerve().to(MARK_ALL).length === 1);
    nerve()
      .to(MARK_ALL)[0]
      ?.answer(json(200, { marked: 1 }));
    await act(async () => until(() => nerve().to(LIST).length === 3, "the marked filters' first page"));
    expect(nerve().to(LIST)[2]?.query.archived).toBe("false");
    await answer(nerve().to(LIST)[2]!, pageOf([noteOf("One", 1, { read_at: "2026-11-20T12:00:00Z" })]));
    nerve()
      .to(COUNT)[0]
      ?.answer(json(200, countOf(0, 0)));
    await settled();
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "notification.toasts.all_read 1" }]);
    expect(failure()?.textContent).toContain("errors.server_busy");
    expect(cards(host)).toEqual([]);
  });

  it("Refresh gives its first page to the filters it was sent for: the archived view's failure stays", async () => {
    await archivedViewRefusedWhile(refresh, () => nerve().to(LIST).length === 2);
    await answer(nerve().to(LIST)[1]!, pageOf([noteOf("One", 1)]));
    nerve()
      .to(COUNT)[0]
      ?.answer(json(200, countOf(1, 0)));
    await settled();
    expect(failure()?.textContent).toContain("errors.server_busy");
    expect(cards(host)).toEqual([]);
  });
});
