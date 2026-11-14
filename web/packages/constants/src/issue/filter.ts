/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  IIssueFilterOptions,
  ILayoutDisplayFiltersOptions,
  TIssueActivityComment,
  TIssueGroupByOptions,
  TWorkItemFilterProperty,
} from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { ISSUE_DISPLAY_PROPERTIES_KEYS, SUB_ISSUES_DISPLAY_PROPERTIES_KEYS } from "./common";

export enum EIssueFilterType {
  FILTERS = "rich_filters",
  DISPLAY_FILTERS = "display_filters",
  DISPLAY_PROPERTIES = "display_properties",
  KANBAN_FILTERS = "kanban_filters",
}

export type TSupportedFilterTypeForUpdate =
  | EIssueFilterType.DISPLAY_FILTERS
  | EIssueFilterType.DISPLAY_PROPERTIES
  | EIssueFilterType.KANBAN_FILTERS;

type TFiltersLayoutOptions = {
  [layoutType: string]: ILayoutDisplayFiltersOptions;
};

type TFilterPropertiesByPageType = {
  filters: TWorkItemFilterProperty[];
  layoutOptions: TFiltersLayoutOptions;
};

type TIssueFiltersToDisplayByPageType = {
  [pageType: string]: TFilterPropertiesByPageType;
};

/** The project page's options: those of its work items' list, of which a cycle's and a module's page are derived. */
const PROJECT_ISSUES: TFilterPropertiesByPageType = {
  filters: [
    "priority",
    "state_group",
    "state_id",
    "cycle_id",
    "module_id",
    "assignee_id",
    "mention_id",
    "subscriber_id",
    "created_by_id",
    "label_id",
    "start_date",
    "target_date",
  ],
  layoutOptions: {
    list: {
      display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
      display_filters: {
        group_by: ["state", "priority", "labels", "assignees", "created_by", "cycle", "module", null],
        order_by: ["sort_order", "-created_at", "-updated_at", "start_date", "-priority", "target_date"],
      },
      extra_options: {
        access: true,
        values: ["show_empty_groups", "sub_issue"],
      },
    },
    kanban: {
      display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
      display_filters: {
        group_by: ["state", "priority", "labels", "assignees", "created_by", "cycle", "module"],
        sub_group_by: ["state", "priority", "labels", "assignees", "created_by", "cycle", "module", null],
        order_by: ["sort_order", "-created_at", "-updated_at", "start_date", "-priority", "target_date"],
      },
      extra_options: {
        access: true,
        values: ["show_empty_groups", "sub_issue"],
      },
    },
    calendar: {
      display_properties: ["key"],
      display_filters: {},
      extra_options: {
        access: true,
        values: ["sub_issue"],
      },
    },
    spreadsheet: {
      display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
      display_filters: {
        order_by: ["sort_order", "-created_at", "-updated_at", "start_date", "-priority"],
      },
      extra_options: {
        access: true,
        values: ["sub_issue"],
      },
    },
  },
};

/**
 * A cycle's or a module's page's options (P5b R9, P6b R12, P7b A8): the project page's without the page's own filter
 * and grouping, every other option the same. The page lists its own cycle's or module's work items only: filtering or
 * grouping by its own would name it everywhere, and nerve refuses the page's own key on the page's preferences.
 */
function withoutOwn(
  options: TFilterPropertiesByPageType,
  filter: TWorkItemFilterProperty,
  grouping: TIssueGroupByOptions
): TFilterPropertiesByPageType {
  const groupings = (keys: TIssueGroupByOptions[]) => keys.filter((key) => key !== grouping);
  const layoutOptions: TFiltersLayoutOptions = {};
  for (const [layout, { display_filters, ...rest }] of Object.entries(options.layoutOptions)) {
    // a layout with no grouping keeps none: the display panel shows a section for each key its options have
    const { group_by, sub_group_by, ...ordering } = display_filters;
    layoutOptions[layout] = {
      ...rest,
      display_filters: {
        ...ordering,
        ...(group_by && { group_by: groupings(group_by) }),
        ...(sub_group_by && { sub_group_by: groupings(sub_group_by) }),
      },
    };
  }
  return { filters: options.filters.filter((key) => key !== filter), layoutOptions };
}

export const ISSUE_DISPLAY_FILTERS_BY_PAGE: TIssueFiltersToDisplayByPageType = {
  profile_issues: {
    filters: ["priority", "state_group", "label_id", "project_id", "start_date", "target_date"],
    layoutOptions: {
      list: {
        display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
        display_filters: {
          group_by: ["state_detail.group", "priority", "project", "labels", null],
          order_by: ["sort_order", "-created_at", "-updated_at", "start_date", "-priority"],
        },
        extra_options: {
          access: true,
          values: ["show_empty_groups", "sub_issue"],
        },
      },
      kanban: {
        display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
        display_filters: {
          group_by: ["state_detail.group", "priority", "project", "labels"],
          order_by: ["sort_order", "-created_at", "-updated_at", "start_date", "-priority"],
        },
        extra_options: {
          access: true,
          values: ["show_empty_groups"],
        },
      },
    },
  },
  archived_issues: {
    filters: [
      "priority",
      "state_group",
      "state_id",
      "assignee_id",
      "created_by_id",
      "label_id",
      "start_date",
      "target_date",
    ],
    layoutOptions: {
      list: {
        display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
        display_filters: {
          group_by: ["state", "priority", "labels", "assignees", "created_by", null],
          order_by: ["sort_order", "-created_at", "-updated_at", "start_date", "-priority"],
        },
        extra_options: {
          access: true,
          values: ["show_empty_groups"],
        },
      },
    },
  },
  my_issues: {
    filters: [
      "priority",
      "state_group",
      "label_id",
      "assignee_id",
      "created_by_id",
      "project_id",
      "start_date",
      "target_date",
    ],
    layoutOptions: {
      spreadsheet: {
        display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
        display_filters: {
          order_by: [],
        },
        extra_options: {
          access: true,
          values: ["sub_issue"],
        },
      },
      list: {
        display_properties: ISSUE_DISPLAY_PROPERTIES_KEYS,
        display_filters: {},
        extra_options: {
          access: false,
          values: [],
        },
      },
    },
  },
  issues: PROJECT_ISSUES,
  cycle_issues: withoutOwn(PROJECT_ISSUES, "cycle_id", "cycle"),
  module_issues: withoutOwn(PROJECT_ISSUES, "module_id", "module"),
  sub_work_items: {
    filters: ["priority", "state_id", "assignee_id", "start_date", "target_date"],
    layoutOptions: {
      list: {
        display_properties: SUB_ISSUES_DISPLAY_PROPERTIES_KEYS,
        display_filters: {
          order_by: ["-created_at", "-updated_at", "start_date", "-priority"],
          group_by: ["state", "priority", "assignees", null],
        },
        extra_options: {
          access: true,
          values: ["sub_issue"],
        },
      },
    },
  },
};

export const ISSUE_STORE_TO_FILTERS_MAP: Partial<Record<EIssuesStoreType, TFilterPropertiesByPageType>> = {
  [EIssuesStoreType.PROJECT]: ISSUE_DISPLAY_FILTERS_BY_PAGE.issues,
  [EIssuesStoreType.CYCLE]: ISSUE_DISPLAY_FILTERS_BY_PAGE.cycle_issues,
  [EIssuesStoreType.MODULE]: ISSUE_DISPLAY_FILTERS_BY_PAGE.module_issues,
};

export const SUB_WORK_ITEM_AVAILABLE_FILTERS_FOR_WORK_ITEM_PAGE: (keyof IIssueFilterOptions)[] = [
  "priority",
  "state",
  "assignees",
  "start_date",
  "target_date",
];

export enum EActivityFilterType {
  ACTIVITY = "ACTIVITY",
  COMMENT = "COMMENT",
  STATE = "STATE",
  ASSIGNEE = "ASSIGNEE",
  DEFAULT = "DEFAULT",
}

export type TActivityFilters = EActivityFilterType;

type TActivityFilterOptionsKey = Exclude<TActivityFilters, EActivityFilterType.DEFAULT>;

export const ACTIVITY_FILTER_TYPE_OPTIONS: Record<TActivityFilterOptionsKey, { labelTranslationKey: string }> = {
  [EActivityFilterType.ACTIVITY]: {
    labelTranslationKey: "common.updates",
  },
  [EActivityFilterType.COMMENT]: {
    labelTranslationKey: "common.comments",
  },
  [EActivityFilterType.STATE]: {
    labelTranslationKey: "common.state",
  },
  [EActivityFilterType.ASSIGNEE]: {
    labelTranslationKey: "common.assignee",
  },
};

export type TActivityFilterOption = {
  key: TActivityFilters;
  labelTranslationKey: string;
  isSelected: boolean;
  onClick: () => void;
};

export const defaultActivityFilters: TActivityFilters[] = [
  EActivityFilterType.ACTIVITY,
  EActivityFilterType.COMMENT,
  EActivityFilterType.STATE,
  EActivityFilterType.ASSIGNEE,
];

export const filterActivityOnSelectedFilters = (
  activities: TIssueActivityComment[],
  filters: TActivityFilters[]
): TIssueActivityComment[] =>
  activities.filter((activity) => {
    if (activity.activity_type === EActivityFilterType.DEFAULT) return true;
    return filters.includes(activity.activity_type as TActivityFilters);
  });

export const BASE_ACTIVITY_FILTER_TYPES = [
  EActivityFilterType.ACTIVITY,
  EActivityFilterType.STATE,
  EActivityFilterType.ASSIGNEE,
  EActivityFilterType.DEFAULT,
];
