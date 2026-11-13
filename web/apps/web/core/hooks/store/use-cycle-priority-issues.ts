/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext } from "react";
import { StoreContext } from "@/lib/store-context";
import type { CyclePriorityIssues } from "@/store/issue/cycle/priority-issues.store";

/**
 * The cycles' priority work items of the session's stores (IssueRootStore.cyclePriorityIssues): the one the root made,
 * never a component's own, as each binding stays in the root once made (re-66 M4).
 */
export const useCyclePriorityIssues = (): CyclePriorityIssues => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useCyclePriorityIssues must be used within StoreProvider");
  return context.issue.cyclePriorityIssues;
};
