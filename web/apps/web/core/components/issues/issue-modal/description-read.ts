/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormData, IssueDescriptionLoadState } from "./types";
import { useEffect, useRef, useState } from "react";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";

type DescriptionReadInput = {
  isOpen: boolean;
  workspaceSlug: string | undefined;
  projectId: string | null | undefined;
  data: IssueFormData | undefined;
  sourceIssueId: string | undefined;
  fetchIssueDetails: boolean;
  isDraft: boolean;
};

/** The modal's description: the given HTML, or the edited or copied work item's as nerve answers it. */
export function useDescriptionRead({
  isOpen,
  workspaceSlug,
  projectId,
  data,
  sourceIssueId,
  fetchIssueDetails,
  isDraft,
}: DescriptionReadInput) {
  const descriptionRequest = useRef(0);
  const [description, setDescription] = useState<string | undefined>(undefined);
  const [descriptionLoadState, setDescriptionLoadState] = useState<IssueDescriptionLoadState>("loading");
  const [descriptionAttempt, setDescriptionAttempt] = useState(0);
  const { fetchIssue } = useIssueDetail();

  useEffect(() => {
    const request = ++descriptionRequest.current;
    let active = true;
    const isCurrentRequest = () => active && descriptionRequest.current === request;
    setDescription(undefined);
    setDescriptionLoadState("loading");
    if (!isOpen || !workspaceSlug) return;

    const loadDescription = async () => {
      try {
        let html = data?.description_html ?? "<p></p>";
        const issueId = data?.id ?? sourceIssueId;
        if (issueId !== undefined && fetchIssueDetails && !isDraft) {
          if (!projectId) throw new Error("Missing source project");
          const response = await fetchIssue(workspaceSlug, projectId, issueId);
          if (typeof response?.description_html !== "string") throw new Error("Missing source description");
          html = response.description_html;
        }
        if (isCurrentRequest()) {
          setDescription(html);
          setDescriptionLoadState("ready");
        }
      } catch {
        if (isCurrentRequest()) setDescriptionLoadState("error");
      }
    };
    // This effect owns both success and failure, including requests retired by close/source changes.
    void loadDescription();
    return () => {
      active = false;
    };
  }, [
    isOpen,
    workspaceSlug,
    projectId,
    data?.id,
    data?.description_html,
    sourceIssueId,
    fetchIssueDetails,
    isDraft,
    fetchIssue,
    descriptionAttempt,
  ]);

  return {
    description,
    setDescription,
    descriptionLoadState,
    retryDescription: () => setDescriptionAttempt((attempt) => attempt + 1),
    /** Retires the request on the wire, so that its answer never reaches a later opening. */
    retireDescription: () => {
      descriptionRequest.current++;
    },
  };
}
