/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { TFilterProperty } from "@nerve/types";
import { EQUALITY_OPERATOR, COLLECTION_OPERATOR } from "@nerve/types";
// local imports
import type { TCreateFilterConfig, TCreateUserFilterParams } from "../../../rich-filters";
import { createFilterConfig, createOperatorConfigEntry, getMemberMultiSelectConfig } from "../../../rich-filters";

// ------------ Assignee filter ------------

/**
 * Assignee filter specific params
 */
type TCreateAssigneeFilterParams = TCreateUserFilterParams;

/**
 * Get the assignee filter config
 * @template K - The filter key
 * @param key - The filter key to use
 * @returns A function that takes parameters and returns the assignee filter config
 */
export const getAssigneeFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreateAssigneeFilterParams> =>
  (params: TCreateAssigneeFilterParams) =>
    createFilterConfig<P>({
      id: key,
      label: "Assignees",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getMemberMultiSelectConfig(updatedParams, EQUALITY_OPERATOR.EXACT)
        ),
      ]),
    });

// ------------ Mention filter ------------

/**
 * Mention filter specific params
 */
type TCreateMentionFilterParams = TCreateUserFilterParams;

/**
 * Get the mention filter config
 * @template K - The filter key
 * @param key - The filter key to use
 * @returns A function that takes parameters and returns the mention filter config
 */
export const getMentionFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreateMentionFilterParams> =>
  (params: TCreateMentionFilterParams) =>
    createFilterConfig<P>({
      id: key,
      label: "Mentions",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getMemberMultiSelectConfig(updatedParams, EQUALITY_OPERATOR.EXACT)
        ),
      ]),
    });

// ------------ Created by filter ------------

/** Subscribers use the existing user selection vocabulary and native subscriber_id field. */
export const getSubscriberFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreateUserFilterParams> =>
  (params) =>
    createFilterConfig<P>({
      id: key,
      label: "Subscribers",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getMemberMultiSelectConfig(updatedParams, EQUALITY_OPERATOR.EXACT)
        ),
      ]),
    });

/**
 * Created by filter specific params
 */
type TCreateCreatedByFilterParams = TCreateUserFilterParams;

/**
 * Get the created by filter config
 * @template K - The filter key
 * @param key - The filter key to use
 * @returns A function that takes parameters and returns the created by filter config
 */
export const getCreatedByFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreateCreatedByFilterParams> =>
  (params: TCreateCreatedByFilterParams) =>
    createFilterConfig<P>({
      id: key,
      label: "Created by",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getMemberMultiSelectConfig(updatedParams, EQUALITY_OPERATOR.EXACT)
        ),
      ]),
    });
