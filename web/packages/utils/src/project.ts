/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { sortBy } from "lodash-es";
// nerve imports
import type { Project } from "@nerve/api-client";
import type { TProjectDisplayFilters, TProjectFilters, TProjectOrderByOptions } from "@nerve/types";
// local imports
import { getDate } from "./datetime";
import { satisfiesDateFilter } from "./filter";

export const projectIdentifierSanitizer = (identifier: string): string =>
  identifier.replace(/[^ÇŞĞIİÖÜA-Za-z0-9]/g, "");

/**
 * @description filters projects based on the filter
 * @param {Project} project
 * @param {TProjectFilters} filters
 * @param {TProjectDisplayFilters} displayFilters
 * @returns {boolean}
 */
export const shouldFilterProject = (
  project: Project,
  displayFilters: TProjectDisplayFilters,
  filters: TProjectFilters
): boolean => {
  let fallsInFilters = true;
  Object.keys(filters).forEach((key) => {
    const filterKey = key as keyof TProjectFilters;
    if (filterKey === "access" && filters.access && filters.access.length > 0)
      fallsInFilters = fallsInFilters && filters.access.includes(`${project.network}`);
    if (filterKey === "lead" && filters.lead && filters.lead.length > 0)
      fallsInFilters = fallsInFilters && filters.lead.includes(`${project.project_lead_id}`);
    if (filterKey === "members" && filters.members && filters.members.length > 0) {
      fallsInFilters = fallsInFilters && filters.members.some((memberId) => project.member_ids.includes(memberId));
    }
    if (filterKey === "created_at" && filters.created_at && filters.created_at.length > 0) {
      const createdDate = getDate(project.created_at);
      filters.created_at.forEach((dateFilter) => {
        fallsInFilters = fallsInFilters && !!createdDate && satisfiesDateFilter(createdDate, dateFilter);
      });
    }
  });
  if (displayFilters.my_projects && !project.member_role) fallsInFilters = false;
  if (displayFilters.archived_projects && !project.archived_at) fallsInFilters = false;
  if (project.archived_at) fallsInFilters = displayFilters.archived_projects ? fallsInFilters : false;

  return fallsInFilters;
};

/**
 * @description orders projects based on the orderByKey
 * @param {Project[]} projects
 * @param {TProjectOrderByOptions | undefined} orderByKey
 * @returns {Project[]}
 */
export const orderProjects = (projects: Project[], orderByKey: TProjectOrderByOptions | undefined): Project[] => {
  let orderedProjects: Project[] = [];
  if (projects.length === 0) return orderedProjects;

  if (orderByKey === "sort_order") orderedProjects = sortBy(projects, [(p) => p.sort_order]);
  if (orderByKey === "name") orderedProjects = sortBy(projects, [(p) => p.name.toLowerCase()]);
  if (orderByKey === "-name") orderedProjects = sortBy(projects, [(p) => p.name.toLowerCase()]).toReversed();
  if (orderByKey === "created_at") orderedProjects = sortBy(projects, [(p) => p.created_at]);
  if (orderByKey === "-created_at") orderedProjects = sortBy(projects, [(p) => !p.created_at]);
  if (orderByKey === "members_length") orderedProjects = sortBy(projects, [(p) => p.member_ids.length]);
  if (orderByKey === "-members_length") orderedProjects = sortBy(projects, [(p) => p.member_ids.length]).toReversed();

  return orderedProjects;
};
