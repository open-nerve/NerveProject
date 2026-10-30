/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Draft, IssueDraftPayload } from "@nerve/api-client";
import { ISSUE_PRIORITIES } from "@nerve/constants";
import type { IssueFormData } from "./types";
/** Editable display values, not an Issue DTO. Unknown draft priorities remain in the stored payload until edited. */
export function draftFormData(draft: Draft): IssueFormData {
  return { ...draftPayloadFields(draft.payload), id: draft.id, project_id: draft.project_id };
}
export function draftPayloadFields(payload: IssueDraftPayload): IssueFormData {
  return {
    name: payload.name,
    description_html: payload.description_html,
    state_id: payload.state_id,
    priority: ISSUE_PRIORITIES.find((row) => row.key === payload.priority)?.key,
    assignee_ids: payload.assignee_ids,
    label_ids: payload.label_ids,
    parent_id: payload.parent_id,
    start_date: payload.start_date,
    target_date: payload.target_date,
  };
}
