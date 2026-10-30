/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { forwardRef } from "react";
// nerve imports
import { RichTextEditorWithRef } from "@nerve/editor";
import type { EditorRefApi, IRichTextEditorProps, TFileHandler } from "@nerve/editor";
import type { MakeOptional } from "@nerve/types";
import { cn } from "@nerve/utils";
// components
import { EditorMentionsRoot } from "@/components/editor/embeds/mentions";
import { EditorMentionsError } from "@/components/editor/mentions-error";
// hooks
import { useEditorConfig, useEditorMention } from "@/hooks/editor";
import { useMember } from "@/hooks/store/use-member";
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";

type RichTextEditorWrapperProps = MakeOptional<
  Omit<IRichTextEditorProps, "fileHandler" | "mentionHandler">,
  "disabledExtensions" | "editable" | "getEditorMetaData"
> & {
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  issueSequenceId?: number;
} & (
    | {
        editable: false;
      }
    | {
        editable: true;
        uploadFile: TFileHandler["upload"];
        duplicateFile: TFileHandler["duplicate"];
      }
  );

export const RichTextEditor = forwardRef(function RichTextEditor(
  props: RichTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const {
    containerClassName,
    editable,
    workspaceSlug,
    workspaceId,
    projectId,
    disabledExtensions = [],
    ...rest
  } = props;
  // store hooks
  const members = useMember();
  const { getUserDetails } = members;
  // use editor mention
  const { fetchMentions, mentionScope } = useEditorMention({ workspaceSlug, projectId, editable, members });
  // editor config
  const { getEditorFileHandlers } = useEditorConfig();
  // parse content
  const { getEditorMetaData } = useParseEditorContent({
    projectId,
    workspaceSlug,
  });

  return (
    <RichTextEditorWithRef
      ref={ref}
      disabledExtensions={disabledExtensions}
      editable={editable}
      fileHandler={getEditorFileHandlers({
        projectId,
        uploadFile: editable ? props.uploadFile : async () => "",
        duplicateFile: editable ? props.duplicateFile : async () => "",
        workspaceId,
        workspaceSlug,
      })}
      getEditorMetaData={getEditorMetaData}
      mentionHandler={{
        searchCallback: fetchMentions,
        scope: mentionScope,
        renderError: (error, retry) => <EditorMentionsError error={error} retry={retry} />,
        renderComponent: EditorMentionsRoot,
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
