/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import type { RefObject } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { EditorRefApi } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";

import { getDescriptionPlaceholderI18n } from "@nerve/utils";
// components
import { RichTextEditor } from "@/components/editor/rich-text/editor";
type TInboxIssueDescription = {
  containerClassName?: string;
  workspaceSlug: string;
  projectId: string;
  workspaceId: string;
  data: Partial<IssueDetail>;
  handleData: (
    issueKey: keyof Partial<IssueDetail>,
    issueValue: Partial<IssueDetail>[keyof Partial<IssueDetail>]
  ) => void;
  editorRef: RefObject<EditorRefApi | null>;
  onEnterKeyPress?: (e?: any) => void;
};

export const InboxIssueDescription = observer(function InboxIssueDescription(props: TInboxIssueDescription) {
  const { containerClassName, workspaceSlug, projectId, workspaceId, data, handleData, editorRef, onEnterKeyPress } =
    props;
  // i18n
  const { t } = useTranslation();

  return (
    <RichTextEditor
      editable
      id="inbox-modal-editor"
      initialValue={!data?.description_html || data?.description_html === "" ? "<p></p>" : data?.description_html}
      ref={editorRef}
      workspaceSlug={workspaceSlug}
      workspaceId={workspaceId}
      projectId={projectId}
      dragDropEnabled={false}
      onChange={(_description: object, description_html: string) => handleData("description_html", description_html)}
      placeholder={(isFocused, description) => t(getDescriptionPlaceholderI18n(isFocused, description))}
      containerClassName={containerClassName}
      onEnterKeyPress={onEnterKeyPress}
      // A field of the form: its wrapper is no Tab stop of its own (-1; the editor itself is the description's stop),
      // and the editor lets Tab and Shift+Tab move on outside lists (list-keymap.ts), where it would swallow them
      tabIndex={-1}
    />
  );
});
