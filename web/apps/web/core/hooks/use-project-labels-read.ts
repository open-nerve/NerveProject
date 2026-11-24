/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useLabel } from "@/hooks/store/use-label";
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * The read of the labels of the project of projectId into the label store (M3 design 7.1), its one definition: by the
 * one key every reader shares, so that SWR makes one read for them all; none for no project (undefined). The labels a
 * work item holds show by that store (LabelListItem draws only a label it holds), so whatever shows a work item reads
 * them: the project's pages, through their wrapper, a member's of a project not archived (use-project-fetch.ts); the
 * notifications page's preview, outside any project's pages, once the caller holds a role in the work item's project
 * (P5b T9). A label dropdown reads them as it opens.
 */
export function useProjectLabelsRead(projectId: string | undefined): void {
  const { fetchProjectLabels } = useLabel();
  useSessionSWR(projectId ? ["PROJECT_LABELS", projectId] : null, (id) => fetchProjectLabels(id));
}
