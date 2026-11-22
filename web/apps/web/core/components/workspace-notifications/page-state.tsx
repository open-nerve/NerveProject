/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import type { NotificationTab, NotificationType } from "@nerve/api-client";
// store
import type { NotificationFilters } from "@/store/notification/filters";
import { DEFAULT_FILTERS } from "@/store/notification/filters";

/** The notification the page previews: its own, its work item and that work item's project. */
export type NotificationSelection = { notificationId: string; issueId: string; projectId: string };

/** The notifications page's state, and the changes the page makes to it. */
type NotificationPageState = {
  filters: NotificationFilters;
  selected: NotificationSelection | undefined;
  /** Shows the tab; the all tab's types and the selection stay as they are. */
  setTab: (tab: NotificationTab) => void;
  /** Lists the type's notifications too, or no longer, in the all tab. */
  toggleType: (type: NotificationType) => void;
  /** Lists every type's notifications in the all tab. */
  clearTypes: () => void;
  /** Lists the unread ones alone, or the read ones too. */
  toggleUnread: () => void;
  /** Lists the archived ones alone, the snoozed no longer; or no longer the archived. */
  toggleArchived: () => void;
  /** Lists the snoozed ones alone, the archived no longer; or no longer the snoozed. */
  toggleSnoozed: () => void;
  /** Previews the notification, in place of any other. */
  select: (selection: NotificationSelection) => void;
  /** Previews nothing, if the notification named is still the one previewed. */
  clearSelection: (notificationId: string) => void;
};

const NotificationPageStateContext = createContext<NotificationPageState | undefined>(undefined);

/** The filters with the type on if it was off, off if it was on. */
const toggled = (filters: NotificationFilters, type: NotificationType): NotificationFilters => ({
  ...filters,
  types: filters.types.includes(type) ? filters.types.filter((one) => one !== type) : [...filters.types, type],
});

/**
 * The notifications page's state (P4 spec 2.5, R3): its tab, its filters and the notification it previews. They are
 * the page's alone, never a store's: the page's layout holds them, under the layout of the route's workspace, which is
 * keyed by its slug, so that they end as the page is left or the workspace changes (W6), a Back or a Forward too, and
 * the page opens again on DEFAULT_FILTERS with nothing selected. The archived and the snoozed are never both listed:
 * nerve takes both as equalities (R25), so the page never asks for both. The selection holds its own work item and
 * project, so that it stays as the tab and the filters change; only the notification it names clears it, as the preview
 * closes, its project's read is refused, or the notification leaves the lists (notification.not_found).
 */
export function NotificationPageStateProvider({ children }: { children: ReactNode }) {
  const [filters, setFilters] = useState<NotificationFilters>(DEFAULT_FILTERS);
  const [selected, setSelected] = useState<NotificationSelection>();
  // the changes, the same functions for as long as the page is open: each changes the state as it is when it runs
  const changes = useMemo<Omit<NotificationPageState, "filters" | "selected">>(
    () => ({
      setTab: (tab) => setFilters((current) => ({ ...current, tab })),
      toggleType: (type) => setFilters((current) => toggled(current, type)),
      clearTypes: () => setFilters((current) => ({ ...current, types: [] })),
      toggleUnread: () => setFilters((current) => ({ ...current, unread: !current.unread })),
      toggleArchived: () => setFilters((current) => ({ ...current, archived: !current.archived, snoozed: false })),
      toggleSnoozed: () => setFilters((current) => ({ ...current, snoozed: !current.snoozed, archived: false })),
      select: (selection) => setSelected(selection),
      clearSelection: (notificationId) =>
        setSelected((current) => (current?.notificationId === notificationId ? undefined : current)),
    }),
    []
  );
  const state = useMemo(() => ({ filters, selected, ...changes }), [filters, selected, changes]);
  return <NotificationPageStateContext.Provider value={state}>{children}</NotificationPageStateContext.Provider>;
}

/** The notifications page's state, of the page's layout (NotificationPageStateProvider). */
export function useNotificationPageState(): NotificationPageState {
  const state = useContext(NotificationPageStateContext);
  if (state === undefined)
    throw new Error("useNotificationPageState must be used within NotificationPageStateProvider");
  return state;
}
