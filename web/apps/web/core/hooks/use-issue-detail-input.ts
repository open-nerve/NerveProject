/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useContext } from "react";
import type { IssueDetailInputState } from "@/components/issues/detail-input-state";
export const IssueDetailInputContext = createContext<IssueDetailInputState | undefined>(undefined);
/** Optional for reusable editors outside a detail; those keep their original local lifetime. */
export function useIssueDetailInput() {
  return useContext(IssueDetailInputContext);
}
