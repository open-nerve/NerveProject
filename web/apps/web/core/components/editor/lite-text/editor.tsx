/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
// nerve imports
import { LiteTextEditorWithRef } from "@nerve/editor";
import type { EditorRefApi, ILiteTextEditorProps, TFileHandler } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import type { MakeOptional } from "@nerve/types";
import { cn, isCommentEmpty } from "@nerve/utils";
// components
import { EditorMentionsRoot } from "@/components/editor/embeds/mentions";
import { EditorMentionsError } from "@/components/editor/mentions-error";
import { IssueCommentToolbar } from "@/components/editor/lite-text/toolbar";
// hooks
import { useEditorConfig, useEditorMention } from "@/hooks/editor";
import { useMember } from "@/hooks/store/use-member";
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";
// services

type LiteTextEditorWrapperProps = MakeOptional<
  Omit<ILiteTextEditorProps, "fileHandler" | "mentionHandler">,
  "disabledExtensions" | "getEditorMetaData"
> & {
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  showSubmitButton?: boolean;
  isSubmitting?: boolean;
  showToolbarInitially?: boolean;
  parentClassName?: string;
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

function isMutableRefObject<T>(ref: React.ForwardedRef<T>): ref is React.MutableRefObject<T | null> {
  return !!ref && typeof ref === "object" && "current" in ref;
}

export const LiteTextEditor = React.forwardRef(function LiteTextEditor(
  props: LiteTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const { t } = useTranslation();
  const {
    containerClassName,
    editable,
    workspaceSlug,
    workspaceId,
    projectId,
    showSubmitButton = true,
    isSubmitting = false,
    showToolbarInitially = true,
    parentClassName = "",
    placeholder = t("issue.comments.placeholder"),
    disabledExtensions = [],
    showPlaceholderOnEmpty = true,
    ...rest
  } = props;
  // states
  const [isFocused, setIsFocused] = useState(showToolbarInitially);
  const [editorRef, setEditorRef] = useState<EditorRefApi | null>(null);
  // store hooks
  const members = useMember();
  const { getUserDetails } = members;
  // parse content
  const { getEditorMetaData } = useParseEditorContent({
    projectId,
    workspaceSlug,
  });
  // use editor mention
  const { fetchMentions, mentionScope } = useEditorMention({ workspaceSlug, projectId, editable, members });
  // editor config
  const { getEditorFileHandlers } = useEditorConfig();
  // derived values
  const isEmpty = isCommentEmpty(props.initialValue);

  return (
    <div
      className={cn(
        "relative rounded-sm border border-subtle",
        {
          "p-3": editable,
        },
        parentClassName
      )}
      onFocus={() => !showToolbarInitially && setIsFocused(true)}
      onBlur={() => !showToolbarInitially && setIsFocused(false)}
    >
      {/* Editor and confirmation toolbar keep their existing layout. */}
      <div>
        {/* Main Editor - always rendered once */}
        <div>
          <LiteTextEditorWithRef
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
            handleEditorReady={(ready) => {
              if (ready) {
                setEditorRef(isMutableRefObject<EditorRefApi>(ref) ? ref.current : null);
              }
            }}
            mentionHandler={{
              searchCallback: fetchMentions,
              scope: mentionScope,
              renderError: (error, retry) => <EditorMentionsError error={error} retry={retry} />,
              renderComponent: EditorMentionsRoot,
              getMentionedEntityDetails: (id) => ({
                display_name: getUserDetails(id)?.display_name ?? "",
              }),
            }}
            placeholder={placeholder}
            showPlaceholderOnEmpty={showPlaceholderOnEmpty}
            containerClassName={cn(containerClassName, "relative", {
              "p-2": !editable,
            })}
            {...rest}
          />
        </div>
      </div>

      {/* Full Toolbar - conditionally rendered */}
      {editable && (
        <div
          className={cn(
            "origin-top overflow-hidden transition-all duration-300 ease-out",
            isFocused ? "mt-3 max-h-[200px] scale-y-100 opacity-100" : "invisible max-h-0 scale-y-0 opacity-0"
          )}
        >
          <IssueCommentToolbar
            executeCommand={(item) => {
              // TODO: update this while toolbar homogenization
              // @ts-expect-error type mismatch here
              editorRef?.executeMenuItemCommand({
                itemKey: item.itemKey,
                ...item.extraProps,
              });
            }}
            handleSubmit={(e) => rest.onEnterKeyPress?.(e)}
            isCommentEmpty={isEmpty}
            isSubmitting={isSubmitting}
            editorRef={editorRef}
            showSubmitButton={showSubmitButton}
            submitButtonText="common.comment"
          />
        </div>
      )}
    </div>
  );
});

LiteTextEditor.displayName = "LiteTextEditor";
