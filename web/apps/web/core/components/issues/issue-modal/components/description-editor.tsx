/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormValues, IssueDescriptionChangeSource } from "../types";
import React, { useEffect } from "react";
import { observer } from "mobx-react";
import type { Control } from "react-hook-form";
import { Controller } from "react-hook-form";
// nerve imports
import type { EditorRefApi } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import { Loader } from "@nerve/ui";
import { getDescriptionPlaceholderI18n } from "@nerve/utils";
// components
import { RichTextEditor } from "@/components/editor/rich-text";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
type TIssueDescriptionEditorProps = {
  control: Control<IssueFormValues>;
  isDraft: boolean;
  descriptionHtmlData: string | undefined;
  editorRef: React.MutableRefObject<EditorRefApi | null>;
  submitBtnRef: React.MutableRefObject<HTMLButtonElement | null>;
  workspaceSlug: string;
  projectId: string | null;
  handleDescriptionHTMLDataChange: (descriptionHtmlData: string, source?: IssueDescriptionChangeSource) => void;
};

// The shared editor contract requires these callbacks for editable content. Images are absent from
// this schema; a future accidental caller must reject rather than claim an upload succeeded.
const imageUnavailable = async (): Promise<never> => {
  throw new Error("Images are not available for work items.");
};

export const IssueDescriptionEditor = observer(function IssueDescriptionEditor(props: TIssueDescriptionEditorProps) {
  const {
    control,
    isDraft,
    descriptionHtmlData,
    editorRef,
    submitBtnRef,
    workspaceSlug,
    projectId,
    handleDescriptionHTMLDataChange,
  } = props;
  // i18n
  const { t } = useTranslation();
  // store hooks
  const { getWorkspaceBySlug } = useWorkspace();
  const workspaceId = getWorkspaceBySlug(workspaceSlug)?.id ?? "";
  useEffect(() => {
    if (descriptionHtmlData !== undefined) handleDescriptionHTMLDataChange(descriptionHtmlData);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [descriptionHtmlData]);

  return (
    <div className="relative rounded-lg border-[0.5px] border-subtle-1 bg-layer-2">
      {descriptionHtmlData === undefined || (!isDraft && !projectId) ? (
        <Loader className="max-h-64 min-h-[120px] space-y-2 overflow-hidden rounded-md border border-subtle p-3 py-2 pt-3">
          <Loader.Item width="100%" height="26px" />
          <div className="flex items-center gap-2">
            <Loader.Item width="26px" height="26px" />
            <Loader.Item width="400px" height="26px" />
          </div>
          <div className="flex items-center gap-2">
            <Loader.Item width="26px" height="26px" />
            <Loader.Item width="400px" height="26px" />
          </div>
          <Loader.Item width="80%" height="26px" />
          <div className="flex items-center gap-2">
            <Loader.Item width="50%" height="26px" />
          </div>
        </Loader>
      ) : (
        <Controller
          name="description_html"
          control={control}
          render={({ field: { value } }) => (
            <RichTextEditor
              editable
              disabledExtensions={["image"]}
              id="issue-modal-editor"
              initialValue={value ?? ""}
              value={descriptionHtmlData}
              workspaceSlug={workspaceSlug}
              workspaceId={workspaceId}
              projectId={projectId ?? undefined}
              onChange={(_description, description_html, options) => {
                handleDescriptionHTMLDataChange(description_html, options?.isMigrationUpdate ? "migration" : "edit");
              }}
              onEnterKeyPress={() => submitBtnRef?.current?.click()}
              ref={editorRef}
              placeholder={(isFocused, description) => t(getDescriptionPlaceholderI18n(isFocused, description))}
              containerClassName="pt-3 min-h-[120px]"
              uploadFile={imageUnavailable}
              duplicateFile={imageUnavailable}
            />
          )}
        />
      )}
    </div>
  );
});
