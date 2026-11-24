/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { AssigneeSetChange } from "@nerve/types";

/**
 * The assignees a change makes of the set held: its removals taken out, its additions not in it put after, the ids it
 * does not name kept in their order (P5b spec 2.6, R20; the modules' rule, modulesAfter in planning-directory.ts). The
 * write unit makes a change's request of it in the work item's turn (issue-write-fields.ts).
 */
export function assigneesAfter(held: readonly string[], change: AssigneeSetChange): string[] {
  const removed = new Set(change.remove);
  const assignees = held.filter((assignee) => !removed.has(assignee));
  for (const assignee of change.add ?? []) if (!assignees.includes(assignee)) assignees.push(assignee);
  return assignees;
}

/**
 * The change that makes the set picked of the set a dropdown showed as it was picked: the members picked not shown
 * added, those shown not picked taken out (moduleSetChange's rule, in planning-directory.ts). A dropdown of an existing
 * work item's assignees sends its pick so, and the write works the whole set out in its turn from the set then held,
 * so a pick made while another one's change is out keeps that one (Q34).
 */
export function assigneeSetChange(shown: readonly string[], picked: readonly string[]): AssigneeSetChange {
  return {
    add: picked.filter((assignee) => !shown.includes(assignee)),
    remove: shown.filter((assignee) => !picked.includes(assignee)),
  };
}
