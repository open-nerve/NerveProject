/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
// types
import type { IIssueDetail } from "@/store/issue/issue-details/root.store";

export const useIssueDetail = (): IIssueDetail => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useIssueDetail must be used within StoreProvider");
  return context.issue.issueDetail;
};

/** Each actual detail/peek host supplies its own UI state; there is no global fallback. */
export const IssueDetailModalContext = createContext<
  import("@/store/issue/issue-details/modals").IssueDetailModals | undefined
>(undefined);
export function useIssueDetailModals() {
  const modals = useContext(IssueDetailModalContext);
  if (!modals) throw new Error("Issue detail modals require their detail or peek host");
  return modals;
}
