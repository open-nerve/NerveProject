/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { DraftCreate, IssueDraftPayload } from "@nerve/api-client";
/** Form clears are instructions to omit a payload key; only the top-level project accepts wire null. */
export type DraftEdits = { [K in keyof IssueDraftPayload]?: IssueDraftPayload[K] | null } & {
  project_id?: string | null;
};
const keys = [
  "name",
  "description_html",
  "state_id",
  "priority",
  "assignee_ids",
  "label_ids",
  "parent_id",
  "start_date",
  "target_date",
] as const;
export function draftPayload(before: IssueDraftPayload, edits: DraftEdits = {}): IssueDraftPayload {
  const payload: IssueDraftPayload = {};
  for (const key of keys) {
    if (key === "assignee_ids" || key === "label_ids") {
      const value = Object.hasOwn(edits, key) ? edits[key] : before[key];
      if (value !== null && value !== undefined) payload[key] = value;
    } else {
      const value = Object.hasOwn(edits, key) ? edits[key] : before[key];
      if (value !== null && value !== undefined) payload[key] = value;
    }
  }
  return payload;
}
export function draftInput(edits: DraftEdits, before: IssueDraftPayload = {}): DraftCreate {
  return {
    ...(Object.hasOwn(edits, "project_id") ? { project_id: edits.project_id } : {}),
    payload: draftPayload(before, edits),
  };
}
