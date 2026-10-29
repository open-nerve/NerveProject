/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export * from "./issue_comment";

/** A display entry refers to a native resource, never a second copy of that resource. */
export type TIssueActivityComment = {
  id: string;
  created_at: string;
} & (
  | { kind: "comment"; activity_type: "COMMENT" }
  | { kind: "activity"; activity_type: "ACTIVITY" | "STATE" | "ASSIGNEE" | "DEFAULT" }
);
