/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import {
  AtOutline,
  CalendarOutline,
  DueDateOutline,
  LabelsOutline,
  MembersOutline,
  PriorityOutline,
  ProjectsOutline,
  StartDateOutline,
  StateOutline,
  UserOutline,
} from "@makeplane/propel/icons";
// nerve imports
import { Avatar } from "@makeplane/propel/components/avatar";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { PriorityIcon, StateGroupIcon } from "@nerve/propel/icons";
import type { MemberUser, Project, State, Label } from "@nerve/api-client";
import type { TFilterConfig, TWorkItemFilterProperty } from "@nerve/types";
import { useTranslation } from "@nerve/i18n";

import {
  getAssigneeFilterConfig,
  getCreatedAtFilterConfig,
  getCreatedByFilterConfig,
  getFileURL,
  getLabelFilterConfig,
  getMentionFilterConfig,
  getSubscriberFilterConfig,
  getPriorityFilterConfig,
  getProjectFilterConfig,
  getStartDateFilterConfig,
  getStateFilterConfig,
  getStateGroupFilterConfig,
  getTargetDateFilterConfig,
  getUpdatedAtFilterConfig,
  isLoaderReady,
} from "@nerve/utils";
// store hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";

export type TWorkItemFiltersEntityProps = {
  workspaceSlug: string;
  labelIds?: string[];
  memberIds?: string[];
  projectId?: string;
  projectIds?: string[];
  stateIds?: string[];
};

export type TUseWorkItemFiltersConfigProps = {
  allowedFilters: TWorkItemFilterProperty[];
} & TWorkItemFiltersEntityProps;

export type TWorkItemFiltersConfig = {
  areAllConfigsInitialized: boolean;
  configs: TFilterConfig<TWorkItemFilterProperty>[];
};

export const useWorkItemFiltersConfig = (props: TUseWorkItemFiltersConfigProps): TWorkItemFiltersConfig => {
  const { allowedFilters, labelIds, memberIds, projectIds, stateIds } = props;
  const { t } = useTranslation();
  const meLabel = t("common.filter_options.me");
  const noneLabel = t("common.none");
  const subscriberLabel = t("common.filter_options.subscribers");
  // store hooks
  const { loader: projectLoader, getProjectById } = useProject();
  const { getLabelById } = useLabel();
  const { getStateById } = useProjectState();
  const { getUserDetails } = useMember();
  // derived values
  const filtersToShow = useMemo(() => new Set(allowedFilters), [allowedFilters]);
  const members: MemberUser[] | undefined = useMemo(
    () =>
      memberIds
        ? (memberIds.map((memberId) => getUserDetails(memberId)).filter((member) => member) as MemberUser[])
        : undefined,
    [memberIds, getUserDetails]
  );
  const workItemStates: State[] | undefined = useMemo(
    () =>
      stateIds ? (stateIds.map((stateId) => getStateById(stateId)).filter((state) => state) as State[]) : undefined,
    [stateIds, getStateById]
  );
  const workItemLabels: Label[] | undefined = useMemo(
    () =>
      labelIds ? (labelIds.map((labelId) => getLabelById(labelId)).filter((label) => label) as Label[]) : undefined,
    [labelIds, getLabelById]
  );
  const projects = useMemo(
    () => (projectIds ? (projectIds.map((id) => getProjectById(id)).filter((listed) => listed) as Project[]) : []),
    [projectIds, getProjectById]
  );
  const areAllConfigsInitialized = useMemo(() => isLoaderReady(projectLoader), [projectLoader]);

  /**
   * Checks if a filter is enabled based on the filters to show.
   * @param key - The filter key.
   * @param level - The level of the filter.
   * @returns True if the filter is enabled, false otherwise.
   */
  const isFilterEnabled = useCallback((key: TWorkItemFilterProperty) => filtersToShow.has(key), [filtersToShow]);

  // state group filter config
  const stateGroupFilterConfig = useMemo(
    () =>
      getStateGroupFilterConfig<TWorkItemFilterProperty>("state_group")({
        isEnabled: isFilterEnabled("state_group"),
        filterIcon: StateOutline,
        getOptionIcon: (stateGroupKey) => <StateGroupIcon stateGroup={stateGroupKey} />,
      }),
    [isFilterEnabled]
  );

  // state filter config
  const stateFilterConfig = useMemo(
    () =>
      getStateFilterConfig<TWorkItemFilterProperty>("state_id")({
        isEnabled: isFilterEnabled("state_id") && workItemStates !== undefined,
        filterIcon: StateOutline,
        getOptionIcon: (state) => <StateGroupIcon stateGroup={state.group} color={state.color} />,
        states: workItemStates ?? [],
      }),
    [isFilterEnabled, workItemStates]
  );

  // label filter config
  const labelFilterConfig = useMemo(
    () =>
      getLabelFilterConfig<TWorkItemFilterProperty>("label_id")({
        isEnabled: isFilterEnabled("label_id") && workItemLabels !== undefined,
        filterIcon: LabelsOutline,
        labels: workItemLabels ?? [],
        additionalOptions: [{ id: "none", value: "none", label: noneLabel }],
        getOptionIcon: (color) => (
          <span className="flex size-2.5 flex-shrink-0 rounded-full" style={{ backgroundColor: color }} />
        ),
      }),
    [isFilterEnabled, workItemLabels, noneLabel]
  );

  // assignee filter config
  const assigneeFilterConfig = useMemo(
    () =>
      getAssigneeFilterConfig<TWorkItemFilterProperty>("assignee_id")({
        isEnabled: isFilterEnabled("assignee_id") && members !== undefined,
        filterIcon: MembersOutline,
        members: members ?? [],
        additionalOptions: [
          { id: "me", value: "me", label: meLabel },
          { id: "none", value: "none", label: noneLabel },
        ],
        getOptionIcon: (memberDetails) => (
          <Avatar
            alt={memberDetails.display_name}
            fallback={memberDetails.display_name?.[0]?.toUpperCase()}
            src={getFileURL(memberDetails.avatar_url)}
            size="2xs"
          />
        ),
      }),
    [isFilterEnabled, members, meLabel, noneLabel]
  );

  // mention filter config
  const mentionFilterConfig = useMemo(
    () =>
      getMentionFilterConfig<TWorkItemFilterProperty>("mention_id")({
        isEnabled: isFilterEnabled("mention_id") && members !== undefined,
        filterIcon: AtOutline,
        members: members ?? [],
        additionalOptions: [{ id: "me", value: "me", label: meLabel }],
        getOptionIcon: (memberDetails) => (
          <Avatar
            alt={memberDetails.display_name}
            fallback={memberDetails.display_name?.[0]?.toUpperCase()}
            src={getFileURL(memberDetails.avatar_url)}
            size="2xs"
          />
        ),
      }),
    [isFilterEnabled, members, meLabel]
  );

  // created by filter config
  const createdByFilterConfig = useMemo(
    () =>
      getCreatedByFilterConfig<TWorkItemFilterProperty>("created_by_id")({
        isEnabled: isFilterEnabled("created_by_id") && members !== undefined,
        filterIcon: UserOutline,
        members: members ?? [],
        additionalOptions: [{ id: "me", value: "me", label: meLabel }],
        getOptionIcon: (memberDetails) => (
          <Avatar
            alt={memberDetails.display_name}
            fallback={memberDetails.display_name?.[0]?.toUpperCase()}
            src={getFileURL(memberDetails.avatar_url)}
            size="2xs"
          />
        ),
      }),
    [isFilterEnabled, members, meLabel]
  );

  const subscriberFilterConfig = useMemo(
    () => ({
      ...getSubscriberFilterConfig<TWorkItemFilterProperty>("subscriber_id")({
        isEnabled: isFilterEnabled("subscriber_id") && members !== undefined,
        filterIcon: MembersOutline,
        members: members ?? [],
        additionalOptions: [{ id: "me", value: "me", label: meLabel }],
      }),
      label: subscriberLabel,
    }),
    [isFilterEnabled, members, meLabel, subscriberLabel]
  );

  // priority filter config
  const priorityFilterConfig = useMemo(
    () =>
      getPriorityFilterConfig<TWorkItemFilterProperty>("priority")({
        isEnabled: isFilterEnabled("priority"),
        filterIcon: PriorityOutline,
        getOptionIcon: (priority) => <PriorityIcon priority={priority} />,
      }),
    [isFilterEnabled]
  );

  // start date filter config
  const startDateFilterConfig = useMemo(
    () =>
      getStartDateFilterConfig<TWorkItemFilterProperty>("start_date")({
        isEnabled: isFilterEnabled("start_date"),
        filterIcon: StartDateOutline,
      }),
    [isFilterEnabled]
  );

  // target date filter config
  const targetDateFilterConfig = useMemo(
    () =>
      getTargetDateFilterConfig<TWorkItemFilterProperty>("target_date")({
        isEnabled: isFilterEnabled("target_date"),
        filterIcon: DueDateOutline,
      }),
    [isFilterEnabled]
  );

  // created at filter config
  const createdAtFilterConfig = useMemo(
    () =>
      getCreatedAtFilterConfig<TWorkItemFilterProperty>("created_at")({
        isEnabled: isFilterEnabled("created_at"),
        filterIcon: CalendarOutline,
      }),
    [isFilterEnabled]
  );

  // updated at filter config
  const updatedAtFilterConfig = useMemo(
    () =>
      getUpdatedAtFilterConfig<TWorkItemFilterProperty>("updated_at")({
        isEnabled: isFilterEnabled("updated_at"),
        filterIcon: CalendarOutline,
      }),
    [isFilterEnabled]
  );

  // project filter config
  const projectFilterConfig = useMemo(
    () =>
      getProjectFilterConfig<TWorkItemFilterProperty>("project_id")({
        isEnabled: isFilterEnabled("project_id") && projects !== undefined,
        filterIcon: ProjectsOutline,
        projects: projects,
        getOptionIcon: (option) => <Logo logo={option.logo_props} size={12} />,
      }),
    [isFilterEnabled, projects]
  );

  return {
    areAllConfigsInitialized,
    configs: [
      stateFilterConfig,
      stateGroupFilterConfig,
      assigneeFilterConfig,
      priorityFilterConfig,
      projectFilterConfig,
      mentionFilterConfig,
      subscriberFilterConfig,
      labelFilterConfig,
      startDateFilterConfig,
      targetDateFilterConfig,
      createdAtFilterConfig,
      updatedAtFilterConfig,
      createdByFilterConfig,
    ],
  };
};
