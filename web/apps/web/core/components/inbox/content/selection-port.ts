/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode, RefObject } from "react";

/**
 * The selection of a work item of a project's intake, as what shows it gives it to the detail region around it
 * (InboxContentRoot; P5b spec 2.5, R18): the intake page's, by its address. The detail region, its headers and its
 * action model read the selection through it alone, never the page's state (useIntakePageState), so that they need no
 * intake page around them.
 */
export type IntakeSelectionPort = {
  /**
   * Leaves the selected work item from for the work item to, or none (undefined), as an action on it ends or the
   * detail cannot show it; the focus goes with the selection (the page's useIntakeLanding).
   */
  leave: (from: string, to: string | undefined) => void;
  /**
   * The work item the selection leaves the work item id for once nerve confirms its triage (R14), asked as the triage
   * begins: the row after it in the list the page shows, else the row before; none for the only one, or one the list
   * does not show.
   */
  neighbour: (id: string) => string | undefined;
  /** The detail region's element, where the page's landing gives the focus for the work item selected. */
  region: RefObject<HTMLElement | null>;
  /**
   * The page's navigation, as the header of the screen draws it: the previous and the next work items of its list
   * (IntakeNavigation), on a narrow screen after the button that shows or hides the list.
   */
  navigation: (screen: "wide" | "narrow") => ReactNode;
};
