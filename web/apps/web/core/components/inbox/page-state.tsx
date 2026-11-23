/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useSearchParams } from "react-router";
// store
import type { IntakeFilters, IntakeTab } from "@/store/issue/intake/filters";
import { defaultFilters } from "@/store/issue/intake/filters";

/** The intake page's state, and the change the page makes to it. */
type IntakePageState = {
  /** The filters and the order of the tab the address names (?currentTab), as the page lists its work items. */
  filters: IntakeFilters;
  /** Changes the filters of the tab shown, as change says, from those it shows now. */
  setFilters: (change: (filters: IntakeFilters) => IntakeFilters) => void;
};

const IntakePageStateContext = createContext<IntakePageState | undefined>(undefined);

/** The tab the address's ?currentTab names: the closed tab by its name, the open tab for any other value or none. */
const tabOf = (value: string | null): IntakeTab => (value === "closed" ? "closed" : "open");

/**
 * The intake page's state (P5a spec 2.6, R5): the filters and the order of the tab the address names. They are the
 * page's alone, never a store's: the intake's layout holds them, under the layouts of the route's project and
 * workspace, each keyed by its id, so that they end as the page is left or the project or the workspace changes, and
 * the page opens again on the tab's first filters (defaultFilters). The tab and the selection are the address's, so
 * that Back and Forward go through them; the filters are keyed by the tab: a change of tab, by the tabs, a link or a
 * Back, returns to that tab's own first filters, the selection changing within the tab keeps them.
 */
export function IntakePageStateProvider({ children }: { children: ReactNode }) {
  const [params] = useSearchParams();
  const tab = tabOf(params.get("currentTab"));
  const [held, setHeld] = useState<IntakeFilters>(() => defaultFilters(tab));
  // keyed by the tab: once the address names another tab, the filters held are of a tab no longer shown
  if (held.tab !== tab) setHeld(defaultFilters(tab));
  const filters = held.tab === tab ? held : defaultFilters(tab);
  const setFilters = useCallback(
    (change: (current: IntakeFilters) => IntakeFilters) =>
      setHeld((current) => change(current.tab === tab ? current : defaultFilters(tab))),
    [tab]
  );
  const state = useMemo(() => ({ filters, setFilters }), [filters, setFilters]);
  return <IntakePageStateContext.Provider value={state}>{children}</IntakePageStateContext.Provider>;
}

/** The intake page's state, of the intake's layout (IntakePageStateProvider). */
export function useIntakePageState(): IntakePageState {
  const state = useContext(IntakePageStateContext);
  if (state === undefined) throw new Error("useIntakePageState must be used within IntakePageStateProvider");
  return state;
}
