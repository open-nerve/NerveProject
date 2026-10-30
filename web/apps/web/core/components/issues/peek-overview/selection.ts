/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IIssueDetail, TPeekIssue } from "@/store/issue/issue-details/root.store";

// DOM ownership stays in the presentation layer; the store keeps only the business selection.
const triggers = new WeakMap<TPeekIssue, HTMLElement>();
export function openIssuePeek(detail: IIssueDetail, selection: TPeekIssue, trigger?: HTMLElement) {
  if (trigger) triggers.set(selection, trigger);
  detail.setPeekIssue(selection);
}
export function restoreIssuePeekFocus(selection: TPeekIssue | undefined) {
  const trigger = selection && triggers.get(selection);
  if (trigger?.isConnected && !trigger.closest('[inert], [aria-hidden="true"]')) trigger.focus();
}
