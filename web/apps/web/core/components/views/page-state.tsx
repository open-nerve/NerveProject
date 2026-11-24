/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
// local imports
import type { ViewListChoices } from "./view-list-order";
import { FIRST_CHOICES } from "./view-list-order";

/** A project's views list's page state, and the change the page makes to it. */
type ViewsPageState = {
  /** What the person chose of the list: its search, its filters and its order. */
  choices: ViewListChoices;
  /** Changes the choices as change says, from those the page shows now. */
  setChoices: (change: (choices: ViewListChoices) => ViewListChoices) => void;
};

const ViewsPageStateContext = createContext<ViewsPageState | undefined>(undefined);

/**
 * A project's views list's page state (P6a spec 2.3, R5): the search, the filters and the order the person chose of
 * the list. They are the page's alone, never a store's: the list's layout holds them, keyed by the route's project, so
 * that they end as the page is left or the project changes (a Back or a Forward too), and the page opens again on the
 * first choices (FIRST_CHOICES).
 */
export function ViewsPageStateProvider({ children }: { children: ReactNode }) {
  const [choices, setChoices] = useState(FIRST_CHOICES);
  const state = useMemo(() => ({ choices, setChoices }), [choices]);
  return <ViewsPageStateContext.Provider value={state}>{children}</ViewsPageStateContext.Provider>;
}

/** A project's views list's page state, of the list's layout (ViewsPageStateProvider). */
export function useViewsPageState(): ViewsPageState {
  const state = useContext(ViewsPageStateContext);
  if (state === undefined) throw new Error("useViewsPageState must be used within ViewsPageStateProvider");
  return state;
}
