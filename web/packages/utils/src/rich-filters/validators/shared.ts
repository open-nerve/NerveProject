/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { TFilterGroupNode, TFilterProperty } from "@nerve/types";
// local imports
import { getGroupChildren } from "../types/shared";

/**
 * Determines if a group should be unwrapped based on the number of children and group type.
 * @param group - The group node to check
 * @returns True if the group should be unwrapped, false otherwise
 */
export const shouldUnwrapGroup = <P extends TFilterProperty>(group: TFilterGroupNode<P>) => {
  const children = getGroupChildren(group);

  // Never unwrap groups with multiple children
  if (children.length !== 1) {
    return false;
  }

  // Unwrap AND groups with a single child
  return true;
};
