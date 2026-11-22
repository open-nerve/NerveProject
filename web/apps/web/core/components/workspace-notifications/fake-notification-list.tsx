/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The notifications page's list in the tests of the list and of its cards (P4 spec 2.5, 2.6): at acme's address,
// under the page's state (read as it renders), with the test's SWR cache, its fetches made into the notification store
// of a tab of notificationTab against a fake nerve, which the test answers. A test file mocks the store hooks with
// that tab's, and the session with a signed-in one.

import { act } from "react";
import type { ReactNode } from "react";
import type { Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { SWRConfig } from "swr";
import { vi } from "vitest";
import type { Notification, NotificationPage } from "@nerve/api-client";
import type { Call, FakeNerve } from "@/lib/auth/fake-nerve";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { acme, notificationOf, notificationRequests } from "@/store/notification/fake-notifications";
import { NotificationPageStateProvider, useNotificationPageState } from "./page-state";
import { NotificationList } from "./sidebar/list";

/** The page's state as the page last rendered it. */
export const pageState = { current: undefined as unknown as ReturnType<typeof useNotificationPageState> };

function Read() {
  pageState.current = useNotificationPageState();
  return null;
}

/**
 * Renders the notifications page's list at acme's address, under the page's state, with the SWR cache given; head, a
 * part of the page's head, before it.
 */
export const renderList = (root: Root, cache: Map<string, unknown>, head?: ReactNode) =>
  act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
        <MemoryRouter initialEntries={[`/${acme.slug}/notifications`]}>
          <Routes>
            <Route
              path="/:workspaceSlug/notifications"
              element={
                <NotificationPageStateProvider>
                  <Read />
                  {head}
                  <NotificationList />
                </NotificationPageStateProvider>
              }
            />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );

/** The page changes its state as does says, and renders what follows. */
export const changeState = (does: (state: ReturnType<typeof useNotificationPageState>) => void) =>
  act(async () => does(pageState.current));

/** The read of acme's list nerve receives after the `before` reads it received so far. */
export async function listRead(nerve: FakeNerve, before: number): Promise<Call> {
  const path = notificationRequests.list(acme);
  await act(async () => until(() => nerve.to(path).length > before, "the list's read"));
  return nerve.to(path)[before]!;
}

/** Time passes for the answers given to settle into the stores and the page. */
export const settled = () => act(async () => vi.advanceTimersByTimeAsync(50));

/** nerve answers a request with body (200), and the page shows what follows. */
export async function answer(call: Call, body: NotificationPage | object) {
  call.answer(json(200, body));
  await settled();
}

/** The cards the list shows, each by its work item ("WEB-2 Two"). */
export const cards = (host: HTMLElement) =>
  [...host.querySelectorAll("li")].map((card) => card.textContent?.match(/WEB-\d+ [A-Z][a-z]+/)?.[0]);

/** The card of the work item ("WEB-2 Two"). */
export const cardOf = (host: HTMLElement, issue: string) =>
  [...host.querySelectorAll("li")].find((card) => card.textContent?.includes(issue));

/**
 * One of the caller's notifications in acme (notificationOf), of WEB-sequence, a work item named by name, its id
 * n-<name in lower case>, unless fields say otherwise.
 */
export function noteOf(name: string, sequence: number, fields: Partial<Notification> = {}): Notification {
  const base = notificationOf(name.toLowerCase());
  return {
    ...base,
    issue_id: `i-web-${sequence}`,
    data: { ...base.data, issue: { ...base.data.issue, id: `i-web-${sequence}`, name, sequence_id: sequence } },
    ...fields,
  };
}
