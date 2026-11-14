/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import type { IssueDetail } from "@nerve/api-client";
import type { ModuleSetChange } from "@nerve/types";

/** Actual editable fields; resource relations and clone identity are separate context. */
export type IssueFormValues = Pick<
  IssueDetail,
  | "name"
  | "description_html"
  | "state_id"
  | "parent_id"
  | "priority"
  | "assignee_ids"
  | "label_ids"
  | "start_date"
  | "target_date"
  | "cycle_id"
  | "module_ids"
> & { project_id: string | null };
export type IssueFormData = Partial<IssueFormValues> & { id?: string };
export const ISSUE_FORM_FIELDS = [
  "project_id",
  "name",
  "description_html",
  "state_id",
  "parent_id",
  "priority",
  "assignee_ids",
  "label_ids",
  "start_date",
  "target_date",
  "cycle_id",
  "module_ids",
] as const;
export type IssueFormField = (typeof ISSUE_FORM_FIELDS)[number];

export type IssueDescriptionLoadState = "loading" | "ready" | "error";
export type IssueDescriptionChangeSource = "load" | "migration" | "edit";
/** What the form says of a submission beside its values, for its command; never part of a draft's payload. */
export type IssueFormSubmitOptions = {
  isDescriptionMigration?: boolean;
  closeAfterSave?: boolean;
  /** A work item's edit's modules: the change from the form's start to the set it holds (P7a spec 2.6, R3). */
  modules?: ModuleSetChange;
  onRefused?: (error: unknown) => void;
};
