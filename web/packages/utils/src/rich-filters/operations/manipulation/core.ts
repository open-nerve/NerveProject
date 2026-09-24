/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type {
  TFilterConditionPayload,
  TFilterExpression,
  TFilterGroupNode,
  TFilterProperty,
  TFilterValue,
} from "@nerve/types";
// local imports
import { createAndGroupNode } from "../../factories/nodes/core";
import { getGroupChildren } from "../../types";
import { isAndGroupNode, isConditionNode, isGroupNode } from "../../types/core";
import { shouldUnwrapGroup } from "../../validators/shared";

/**
 * Adds an AND condition to the filter expression.
 * @param expression - The current filter expression
 * @param condition - The condition to add
 * @returns The updated filter expression
 */
export const addAndCondition = <P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  condition: TFilterExpression<P>
): TFilterExpression<P> => {
  // if no expression, set the new condition
  if (!expression) {
    return condition;
  }
  // if the expression is a condition, convert it to an AND group
  if (isConditionNode(expression)) {
    return createAndGroupNode([expression, condition]);
  }
  // if the expression is a group, and the group is an AND group, add the new condition to the group
  if (isGroupNode(expression) && isAndGroupNode(expression)) {
    expression.children.push(condition);
    return expression;
  }
  // if the expression is a group, but not an AND group, create a new AND group and add the new condition to it
  if (isGroupNode(expression) && !isAndGroupNode(expression)) {
    return createAndGroupNode([expression, condition]);
  }
  // Throw error for unexpected expression type
  console.error("Invalid expression type", expression);
  return expression;
};

/**
 * Updates a node in the filter expression.
 * Uses recursive tree traversal with proper type handling.
 * @param expression - The filter expression to update
 * @param targetId - The id of the node to update
 * @param updates - The updates to apply to the node
 */
export const updateNodeInExpression = <P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  targetId: string,
  updates: Partial<TFilterConditionPayload<P, TFilterValue>>
) => {
  // Helper function to recursively update nodes
  const updateNode = (node: TFilterExpression<P>): void => {
    if (node.id === targetId) {
      if (!isConditionNode<P, TFilterValue>(node)) {
        console.warn("updateNodeInExpression: targetId matched a group; ignoring updates");
        return;
      }
      Object.assign(node, updates);
      return;
    }

    if (isGroupNode(node)) {
      const children = getGroupChildren(node);
      children.forEach((child) => updateNode(child));
    }
  };

  updateNode(expression);
};

/**
 * Unwraps a group if it meets the unwrapping criteria, otherwise returns the group.
 * @param group - The group node to potentially unwrap
 * @param preserveNotGroups - Whether to preserve NOT groups even with single children
 * @returns The unwrapped child or the original group
 */
export const unwrapGroupIfNeeded = <P extends TFilterProperty>(
  group: TFilterGroupNode<P>,
  preserveNotGroups = true
) => {
  if (shouldUnwrapGroup(group, preserveNotGroups)) {
    const children = getGroupChildren(group);
    return children[0];
  }
  return group;
};
