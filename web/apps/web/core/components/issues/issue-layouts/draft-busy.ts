/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext } from "react";

/**
 * Whether the page's draft is being saved (P6a spec 2.7, R29): a view's page's Update view out (ViewStore.isUpdating,
 * which ProjectViewLayoutRoot provides for a project's view's page and WorkspaceViewLayoutRoot for a workspace's,
 * M7/P6b spec 2.9; a default view's page has no view, so there it is false). The ways deep in the list's layouts that
 * change the draft read it and stop meanwhile: a spreadsheet column's sort, the calendar's options, the empty list's
 * clearing of the filters. No other page provides it: there it is false, and they work as ever.
 */
export const DraftBusyContext = createContext(false);
