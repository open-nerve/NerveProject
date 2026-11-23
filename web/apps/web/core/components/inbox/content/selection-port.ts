/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode, RefObject } from "react";

/** What every selection's port gives: where the selection leaves the work item, and the detail region's element. */
type Selection = {
  /**
   * Leaves the selected work item from for the work item to, or none (undefined), as an action on it ends or the
   * detail cannot show it; the focus goes with the selection (the landing of what shows it).
   */
  leave: (from: string, to: string | undefined) => void;
  /** The detail region's element, where the focus lands for the work item selected. */
  region: RefObject<HTMLElement | null>;
};

/**
 * The selection of a work item of a project's intake, as what shows it gives it to the detail region around it
 * (InboxContentRoot; P5b spec 2.5, R18): the intake page's, by its address (kind "page"), or the notifications page's
 * preview's, the work item embedded in it (kind "embed"; R16, R17). The detail region, its headers and its action model
 * read the selection through it alone, never the page's state (useIntakePageState), so that they need no intake page
 * around them.
 */
export type IntakeSelectionPort =
  | (Selection & {
      kind: "page";
      /**
       * The work item the selection leaves the work item id for once nerve confirms its triage (R14), asked as the
       * triage begins: the row after it in the list the page shows, else the row before; none for the only one, or one
       * the list does not show.
       */
      neighbour: (id: string) => string | undefined;
      /**
       * The page's navigation, as the header of the screen draws it: the previous and the next work items of its list
       * (IntakeNavigation), on a narrow screen after the button that shows or hides the list.
       */
      navigation: (screen: "wide" | "narrow") => ReactNode;
    })
  | (Selection & {
      kind: "embed";
      /**
       * The embed's close, its headers' named button (Close): the selection, the preview's notification, is cleared
       * (as leave clears it), the focus landing as the notifications page lands it (R26).
       */
      close: () => void;
    });
