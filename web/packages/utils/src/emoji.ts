/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * Groups reactions by a specified key
 * @param {any[]} reactions - Array of reaction objects
 * @param {string} key - Key to group reactions by
 * @returns {Object} Object with reactions grouped by the specified key
 */
export const groupReactions: (reactions: any[], key: string) => { [key: string]: any[] } = (
  reactions: any,
  key: string
) => {
  if (!Array.isArray(reactions)) {
    console.error("Expected an array of reactions, but got:", reactions);
    return {};
  }

  const groupedReactions = reactions.reduce(
    (acc: any, reaction: any) => {
      if (!reaction || typeof reaction !== "object" || !Object.prototype.hasOwnProperty.call(reaction, key)) {
        console.warn("Skipping undefined reaction or missing key:", reaction);
        return acc; // Skip undefined reactions or those without the specified key
      }

      if (!acc[reaction[key]]) {
        acc[reaction[key]] = [];
      }
      acc[reaction[key]].push(reaction);
      return acc;
    },
    {} as { [key: string]: any[] }
  );

  return groupedReactions;
};
