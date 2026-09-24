/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
// helpers
import { getEditorAssetSrc } from "@nerve/utils";
import type { TCustomComponentsMetaData } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";

type TArgs = {
  projectId?: string;
  workspaceSlug: string;
};

export const useParseEditorContent = (args: TArgs) => {
  const { projectId, workspaceSlug } = args;
  // store hooks
  const { getUserDetails } = useMember();

  const getEditorMetaData = useCallback(
    (htmlContent: string): TCustomComponentsMetaData => {
      const parser = new DOMParser();
      const doc = parser.parseFromString(htmlContent, "text/html");
      const filesMetaData: TCustomComponentsMetaData["file_assets"] = [];
      // process image components
      const imageComponents = doc.querySelectorAll("image-component");
      imageComponents.forEach((element) => {
        const src = element.getAttribute("src");
        if (src) {
          const assetSrc = src.startsWith("http")
            ? src
            : getEditorAssetSrc({
                assetId: src,
                projectId,
                workspaceSlug,
              });
          if (assetSrc) {
            filesMetaData.push({
              id: src,
              name: src,
              url: assetSrc,
            });
          }
        }
      });
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
        file_assets: filesMetaData,
        user_mentions: userMentions,
      };
    },
    [getUserDetails, projectId, workspaceSlug]
  );

  return {
    getEditorMetaData,
  };
};
