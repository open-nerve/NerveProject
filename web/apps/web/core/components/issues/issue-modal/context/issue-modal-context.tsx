/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext } from "react";
// nerve imports
import type { Issue } from "@nerve/api-client";

export type TIssueModalContext = {
  allowedProjectIds: string[];
  selectedParentIssue: Issue | null;
  setSelectedParentIssue: React.Dispatch<React.SetStateAction<Issue | null>>;
};

export const IssueModalContext = createContext<TIssueModalContext | undefined>(undefined);
