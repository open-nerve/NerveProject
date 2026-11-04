/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { forwardRef } from "react";
// nerve imports
import { RichTextEditorWithRef } from "@nerve/editor";
import type { EditorRefApi } from "@nerve/editor";
import type { RichTextEditorWrapperProps } from "./types";
import { cn } from "@nerve/utils";
// components
import { useEditorLabels } from "@/components/editor/use-editor-labels";
import { EditorMentionsRoot } from "@/components/editor/embeds/mentions";
import { EditorMentionsError } from "@/components/editor/mentions-error";
// hooks
import { useEditorMention } from "@/hooks/editor";
import { useMember } from "@/hooks/store/use-member";
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";

export const RichTextEditor = forwardRef(function RichTextEditor(
  props: RichTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const {
    containerClassName,
    editable = true,
    workspaceSlug,
    workspaceId,
    projectId,
    disabledExtensions = [],
    ...rest
  } = props;
  const labels = useEditorLabels();
  // store hooks
  const members = useMember();
  const { getUserDetails } = members;
  // use editor mention
  const { fetchMentions, mentionScope } = useEditorMention({ workspaceSlug, projectId, editable, members });
  // parse content
  const { getEditorMetaData } = useParseEditorContent({
    projectId,
    workspaceSlug,
  });

  return (
    <RichTextEditorWithRef
      labels={labels}
      ref={ref}
      disabledExtensions={disabledExtensions}
      editable={editable}
      getEditorMetaData={getEditorMetaData}
      mentionHandler={{
        searchCallback: fetchMentions,
        scope: mentionScope,
        renderError: (error, retry) => <EditorMentionsError error={error} retry={retry} />,
        renderComponent: (mention) => (
          <EditorMentionsRoot
            {...mention}
            workspaceID={workspaceId}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
          />
        ),
        getMentionedEntityDetails: (id) => ({
          display_name: getUserDetails(id)?.display_name ?? "",
        }),
      }}
      {...rest}
      containerClassName={cn("relative pb-3 pl-3", containerClassName)}
    />
  );
});

RichTextEditor.displayName = "RichTextEditor";
