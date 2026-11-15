/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
import { useProject } from "@/hooks/store/use-project";
import { listWordOn, type ListWord } from "@/lib/planning-switch";

/**
 * Whether the route's list page offers a word of its options (P7a spec 2.5, R11; P7b spec 2.5, R8): on a page of a
 * project (its list, a cycle's, a module's: the route names the project) a word of the planning only while the route's
 * project has its domain turned on, the page's own domain too; on a page of no project (the member view, whose rows are
 * of many projects) every word the page's options give. The display, grouping and sub-grouping panels offer by it, the
 * project's mobile header through the same panels.
 */
export function useRouteListWords(): (word: ListWord) => boolean {
  const { projectId } = useParams();
  const { getProjectById } = useProject();
  const project = projectId ? getProjectById(projectId) : undefined;
  return (word) => !projectId || listWordOn(project, word);
}
