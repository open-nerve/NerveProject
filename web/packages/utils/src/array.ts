/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
import type { IIssueLabel, IIssueLabelTree } from "@nerve/types";

/**
 * @description Builds a tree structure from an array of labels
 * @param {IIssueLabel[]} array Array of labels
 * @param {any} parent Parent ID
 * @returns {IIssueLabelTree[]} Tree structure
 */
export const buildTree = (array: IIssueLabel[], parent = null) => {
  const tree: IIssueLabelTree[] = [];

  array.forEach((item: any) => {
    if (item.parent === parent) {
      const children = buildTree(array, item.id);
      item.children = children;
      tree.push(item);
    }
  });

  return tree;
};

/**
 * @description Returns valid keys from object whose value is not falsy
 * @param {any} obj Object to check
 * @returns {string[]} Array of valid keys
 * @example
 * getValidKeysFromObject({a: 1, b: 0, c: null}) // returns ['a']
 */
export const getValidKeysFromObject = (obj: any) => {
  if (!obj || isEmpty(obj) || typeof obj !== "object" || Array.isArray(obj)) return [];

  return Object.keys(obj).filter((key) => !!obj[key]);
};

/**
 * @description Sorts dropdown options with selected items appearing first
 * @param {T[]} options Array of dropdown options with value property
 * @param {string[] | string | null | undefined} selectedValues Selected value(s) - array for multi-select, string for single-select
 * @returns {T[]} Sorted array with selected items first
 * @example
 * const options = [{value: '1', label: 'A'}, {value: '2', label: 'B'}];
 * sortBySelectedFirst(options, ['2']) // returns [{value: '2', label: 'B'}, {value: '1', label: 'A'}]
 */
export const sortBySelectedFirst = <T extends { value: string | null }>(
  options: T[] | undefined,
  selectedValues: string[] | string | null | undefined
): T[] | undefined => {
  if (!options || options.length === 0) return options;

  // Normalize selectedValues to array for consistent handling
  const selectedSet = new Set(Array.isArray(selectedValues) ? selectedValues : selectedValues ? [selectedValues] : []);

  if (selectedSet.size === 0) return options;

  // Create a shallow copy to avoid mutating the original array
  return [...options].sort((a, b) => {
    const aSelected = a.value !== null && selectedSet.has(a.value);
    const bSelected = b.value !== null && selectedSet.has(b.value);

    // If both selected or both unselected, maintain original order
    if (aSelected === bSelected) return 0;

    // Selected items come first
    return aSelected ? -1 : 1;
  });
};

/**
 * @description Sorts dropdown options with current user first, then selected items, then unselected items
 * @param {T[]} options Array of dropdown options with value property
 * @param {string[] | string | null | undefined} selectedValues Selected value(s) - array for multi-select, string for single-select
 * @param {string | undefined} currentUserId ID of the current user to prioritize
 * @returns {T[]} Sorted array with current user first, then selected items, then unselected
 * @example
 * const options = [{value: 'user1'}, {value: 'user2'}, {value: 'user3'}];
 * sortByCurrentUserThenSelected(options, ['user2'], 'user3')
 * // returns [{value: 'user3'}, {value: 'user2'}, {value: 'user1'}]
 */
export const sortByCurrentUserThenSelected = <T extends { value: string | null }>(
  options: T[] | undefined,
  selectedValues: string[] | string | null | undefined,
  currentUserId: string | undefined
): T[] | undefined => {
  if (!options || options.length === 0) return options;

  // Normalize selectedValues to array for consistent handling
  const selectedSet = new Set(Array.isArray(selectedValues) ? selectedValues : selectedValues ? [selectedValues] : []);

  // Create a shallow copy to avoid mutating the original array
  return [...options].sort((a, b) => {
    const aIsCurrent = currentUserId && a.value === currentUserId;
    const bIsCurrent = currentUserId && b.value === currentUserId;

    // Current user always comes first
    if (aIsCurrent && !bIsCurrent) return -1;
    if (!aIsCurrent && bIsCurrent) return 1;
    if (aIsCurrent && bIsCurrent) return 0;

    // If neither is current user, sort by selection state
    const aSelected = a.value !== null && selectedSet.has(a.value);
    const bSelected = b.value !== null && selectedSet.has(b.value);

    // If both selected or both unselected, maintain original order
    if (aSelected === bSelected) return 0;

    // Selected items come before unselected
    return aSelected ? -1 : 1;
  });
};
