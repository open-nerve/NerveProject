/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type {
  TAllAvailableOperatorsForDisplay,
  TFilterConditionNode,
  TFilterConditionNodeForDisplay,
  TFilterExpression,
  TFilterGroupNode,
  TFilterProperty,
  TFilterValue,
} from "@nerve/types";
// local imports
import { isConditionNode, isGroupNode } from "../../types/core";
import { getGroupChildren } from "../../types/shared";
import { getDisplayOperator } from "./shared";

/**
 * Generic tree visitor function type
 */
type TreeVisitorFn<P extends TFilterProperty, T> = (
  expression: TFilterExpression<P>,
  parent?: TFilterGroupNode<P>,
  depth?: number
) => T | null;

/**
 * Tree traversal modes
 */
enum TreeTraversalMode {
  /** Visit all nodes depth-first */
  ALL = "ALL",
  /** Visit only condition nodes */
  CONDITIONS = "CONDITIONS",
  /** Visit only group nodes */
  GROUPS = "GROUPS",
}

/**
 * Generic tree traversal utility that visits nodes based on the specified mode.
 * This eliminates code duplication in tree walking functions.
 *
 * @param expression - The expression to traverse
 * @param visitor - Function to call for each visited node
 * @param mode - Traversal mode to determine which nodes to visit
 * @param parent - Parent node (used internally for recursion)
 * @param depth - Current depth (used internally for recursion)
 * @returns Array of results from the visitor function (nulls are filtered out)
 */
const traverseExpressionTree = <P extends TFilterProperty, T>(
  expression: TFilterExpression<P> | null,
  visitor: TreeVisitorFn<P, T>,
  mode: TreeTraversalMode = TreeTraversalMode.ALL,
  parent?: TFilterGroupNode<P>,
  depth: number = 0
): T[] => {
  if (!expression) return [];

  const results: T[] = [];

  // Determine if we should visit this node based on the mode
  const shouldVisit =
    mode === TreeTraversalMode.ALL ||
    (mode === TreeTraversalMode.CONDITIONS && isConditionNode(expression)) ||
    (mode === TreeTraversalMode.GROUPS && isGroupNode(expression));

  if (shouldVisit) {
    const result = visitor(expression, parent, depth);
    if (result !== null) {
      results.push(result);
    }
  }

  // Recursively traverse children for group nodes
  if (isGroupNode(expression)) {
    const children = getGroupChildren(expression);
    for (const child of children) {
      const childResults = traverseExpressionTree(child, visitor, mode, expression, depth + 1);
      results.push(...childResults);
    }
  }

  return results;
};

/**
 * Finds a node by its ID in the filter expression tree.
 * Uses the generic tree traversal utility for better maintainability.
 * @param expression - The filter expression to search in
 * @param targetId - The ID of the node to find
 * @returns The found node or null if not found
 */
export const findNodeById = <P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  targetId: string
): TFilterExpression<P> | null => {
  const results = traverseExpressionTree(
    expression,
    (node) => (node.id === targetId ? node : null),
    TreeTraversalMode.ALL
  );

  // Return the first match (there should only be one with unique IDs)
  return results.length > 0 ? results[0] : null;
};

/**
 * Extracts all conditions from a filter expression.
 * Uses the generic tree traversal utility for better maintainability and consistency.
 * @param expression - The filter expression to extract conditions from
 * @returns An array of filter conditions
 */
export const extractConditions = <P extends TFilterProperty>(
  expression: TFilterExpression<P>
): TFilterConditionNode<P, TFilterValue>[] =>
  traverseExpressionTree(expression, (node) => (isConditionNode(node) ? node : null), TreeTraversalMode.CONDITIONS);

/**
 * Extracts all conditions from a filter expression, including their display operators.
 * @param expression - The filter expression to extract conditions from
 * @returns An array of filter conditions with their display operators
 */
export const extractConditionsWithDisplayOperators = <P extends TFilterProperty>(
  expression: TFilterExpression<P>
): TFilterConditionNodeForDisplay<P, TFilterValue>[] => {
  // First extract all raw conditions
  const rawConditions = extractConditions(expression);

  // Transform operators using the extended helper
  return rawConditions.map((condition) => {
    const displayOperator = getDisplayOperator(condition.operator, expression, condition.id);
    return {
      ...condition,
      operator: displayOperator,
    };
  });
};

/**
 * Finds all conditions by property and operator.
 * @param expression - The filter expression to search in
 * @param property - The property to find the conditions by
 * @param operator - The operator to find the conditions by
 * @returns An array of conditions that match the property and operator
 */
export const findConditionsByPropertyAndOperator = <P extends TFilterProperty>(
  expression: TFilterExpression<P>,
  property: P,
  operator: TAllAvailableOperatorsForDisplay
): TFilterConditionNodeForDisplay<P, TFilterValue>[] => {
  const conditions = extractConditionsWithDisplayOperators(expression);
  return conditions.filter((condition) => condition.property === property && condition.operator === operator);
};
