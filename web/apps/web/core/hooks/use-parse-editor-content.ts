/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
// helpers
import type { TCustomComponentsMetaData } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";

type TArgs = {
  workspaceSlug: string;
};

export const useParseEditorContent = (args: TArgs) => {
  const { workspaceSlug } = args;
  // store hooks
  const { getUserDetails } = useMember();

  const getEditorMetaData = useCallback(
    (htmlContent: string): TCustomComponentsMetaData => {
      const parser = new DOMParser();
      const doc = parser.parseFromString(htmlContent, "text/html");
      // process user mentions
      const userMentions: TCustomComponentsMetaData["user_mentions"] = [];
      const mentionComponents = doc.querySelectorAll("mention-component");
      mentionComponents.forEach((element) => {
        const id = element.getAttribute("entity_identifier");
        if (id) {
          const userDetails = getUserDetails(id);
          const originUrl = typeof window !== "undefined" && (window.location.origin ?? "");
          const path = `${workspaceSlug}/profile/${id}`;
          const url = `${originUrl}/${path}`;
          if (userDetails) {
            userMentions.push({
              id,
              display_name: userDetails.display_name,
              url,
            });
          }
        }
      });

      return {
        // A content image has no stable address: its reads are short-lived grants that must never leave the page,
        // so exported Markdown omits images (the copy-as-Markdown risk is named for P8).
        file_assets: [],
        user_mentions: userMentions,
      };
    },
    [getUserDetails, workspaceSlug]
  );

  return {
    getEditorMetaData,
  };
};
