/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A part of the intake page's sidebar that changes the page's filters (P5a spec 2.6, 2.7), in its tests: rendered at
// p-web's intake address, in the tab the address names, under the page's state (read as it renders), inside a form
// whose submit the test gives. A test that opens a menu uses fake timers (the menus' transitions) and stubs
// Element.getAnimations, which jsdom lacks.

import { act } from "react";
import type { ReactNode } from "react";
import type { Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import { IntakePageStateProvider, useIntakePageState } from "../page-state";

/** The page's state as the page last rendered it. */
export const filterState = { current: undefined as unknown as ReturnType<typeof useIntakePageState> };

function Read() {
  filterState.current = useIntakePageState();
  return null;
}

/** Renders part at p-web's intake page in the tab search names, under the page's state, in a form whose submit is given. */
export const renderFilterPart = (
  root: Root,
  part: ReactNode,
  submitted: () => void = () => {},
  search = "?currentTab=open"
) =>
  act(async () =>
    root.render(
      // keyed by the address: another address is another page, its state its own
      <MemoryRouter key={search} initialEntries={[`/acme/projects/p-web/intake${search}`]}>
        <Routes>
          <Route
            path="/:workspaceSlug/projects/:projectId/intake"
            element={
              <IntakePageStateProvider>
                <Read />
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    submitted();
                  }}
                >
                  {part}
                </form>
              </IntakePageStateProvider>
            }
          />
        </Routes>
      </MemoryRouter>
    )
  );

/** The page changes its filters by fields, from those it holds. */
export const changeFilters = (fields: Partial<IntakeFilters>) =>
  act(async () => filterState.current.setFilters((current) => ({ ...current, ...fields })));
