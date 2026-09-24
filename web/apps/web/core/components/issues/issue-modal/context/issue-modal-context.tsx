/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext } from "react";
// nerve imports
import type { ISearchIssueResponse } from "@nerve/types";

export type TIssueModalContext = {
  allowedProjectIds: string[];
  selectedParentIssue: ISearchIssueResponse | null;
  setSelectedParentIssue: React.Dispatch<React.SetStateAction<ISearchIssueResponse | null>>;
};

export const IssueModalContext = createContext<TIssueModalContext | undefined>(undefined);
