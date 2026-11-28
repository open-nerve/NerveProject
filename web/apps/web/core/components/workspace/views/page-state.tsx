/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";

/** A workspace's views list's page state, and the change the page makes to it. */
type WorkspaceViewsPageState = {
  /** The text the names of the rows shown hold, whatever its case and its spaces around. */
  search: string;
  setSearch: (search: string) => void;
};

const WorkspaceViewsPageStateContext = createContext<WorkspaceViewsPageState | undefined>(undefined);

/**
 * A workspace's views list's page state (M7/P6b spec 2.10, R13): its search alone (the list has no filter and no
 * order). It is the page's, never a store's: the list's layout holds it, keyed by the route's workspace, so
 * that it ends as the page is left or the workspace changes, and the page opens again with none.
 */
export function WorkspaceViewsPageStateProvider({ children }: { children: ReactNode }) {
  const [search, setSearch] = useState("");
  const state = useMemo(() => ({ search, setSearch }), [search]);
  return <WorkspaceViewsPageStateContext.Provider value={state}>{children}</WorkspaceViewsPageStateContext.Provider>;
}

/** A workspace's views list's page state, of the list's layout (WorkspaceViewsPageStateProvider). */
export function useWorkspaceViewsPageState(): WorkspaceViewsPageState {
  const state = useContext(WorkspaceViewsPageStateContext);
  if (state === undefined) throw new Error("useWorkspaceViewsPageState must be used within its provider");
  return state;
}
