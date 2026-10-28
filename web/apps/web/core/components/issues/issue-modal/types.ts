/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { TIssue } from "@nerve/types";

/** Actual editable fields; resource relations and clone identity are separate context. */
export type IssueFormValues = Pick<
  TIssue,
  | "project_id"
  | "name"
  | "description_html"
  | "state_id"
  | "parent_id"
  | "priority"
  | "assignee_ids"
  | "label_ids"
  | "start_date"
  | "target_date"
>;
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
] as const;

export type IssueDescriptionLoadState = "loading" | "ready" | "error";
