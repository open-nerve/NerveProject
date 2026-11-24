/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { IssueUpdate } from "@nerve/api-client";

/** The modules a change adds a work item to and takes it out of; none is in both. */
export type ModuleSetChange = { add?: readonly string[]; remove?: readonly string[] };

/** The members a change assigns a work item to and unassigns it from; none is in both. */
export type AssigneeSetChange = { add?: readonly string[]; remove?: readonly string[] };

/**
 * A change of a work item as the lists, the rows, the spreadsheet and the detail send it (P7a spec 2.2, R3): an
 * update's fields, but its modules as additions and removals, applied in the write's turn to the set the work item then
 * holds, never as a whole set, so that a value carrying one, as a work item's fields do, is not a change. Its
 * assignees go either way (P5b spec 2.6, R20): a dropdown's pick as additions and removals, applied so too, or a whole
 * set worked out in the turn (a drop) or chosen in a form; never both. The cycle is one field and goes as given.
 */
export type IssueChange = Omit<IssueUpdate, "module_ids" | "assignee_ids"> & {
  module_ids?: never;
  modules?: ModuleSetChange;
} & (
    | { assignee_ids?: IssueUpdate["assignee_ids"]; assignees?: never }
    | { assignee_ids?: never; assignees: AssigneeSetChange }
  );
