/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormData } from "./types";
import { useState } from "react";
import { isEmpty } from "lodash-es";
import { observer } from "mobx-react";
// Nerve imports

import { isEmptyHtmlString } from "@nerve/utils";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { useTranslation } from "@nerve/i18n";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
// hooks

// local imports
import { ConfirmIssueDiscard } from "../confirm-issue-discard";
import { IssueFormRoot } from "./form";
import type { IssueFormProps } from "./form";

interface DraftIssueProps extends IssueFormProps {
  isOpen: boolean;
  promptToSaveDraft: boolean;
  isBusy: () => boolean;
  changesMade: IssueFormData | null;
  onChange: (formData: IssueFormData | null) => void;
}

export const DraftIssueLayout = observer(function DraftIssueLayout(props: DraftIssueProps) {
  const {
    changesMade,
    data,
    onChange,
    onClose,
    descriptionLoadState = "ready",
    isOpen,
    promptToSaveDraft,
    isBusy,
    editorRef,
  } = props;
  const { t } = useTranslation();
  // states
  const [issueDiscardModal, setIssueDiscardModal] = useState(false);
  // store hooks

  const sanitizeChanges = (): IssueFormData => {
    const sanitizedChanges = { ...changesMade };
    Object.entries(sanitizedChanges).forEach(([key, value]) => {
      const issueKey = key as keyof IssueFormData;
      if (value === null || value === undefined || value === "") delete sanitizedChanges[issueKey];
      if (typeof value === "object" && isEmpty(value)) delete sanitizedChanges[issueKey];
      if (Array.isArray(value) && value.length === 0) delete sanitizedChanges[issueKey];
      if (issueKey === "project_id") delete sanitizedChanges.project_id;
      if (issueKey === "priority" && value && value === "none") delete sanitizedChanges.priority;
      if (
        issueKey === "description_html" &&
        changesMade?.description_html &&
        isEmptyHtmlString(changesMade.description_html)
      )
        delete sanitizedChanges.description_html;
    });
    return sanitizedChanges;
  };

  const requestClose = () => {
    if (isBusy()) return;
    if (descriptionLoadState === "ready" && editorRef.current && !editorRef.current.isEditorReadyToDiscard()) {
      setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("editor_is_not_ready_to_discard_changes") });
      return;
    }
    // If the user is updating an existing work item, we don't need to show the discard modal
    if (data?.id || !promptToSaveDraft) {
      onClose();
      setIssueDiscardModal(false);
    } else {
      if (changesMade) {
        const sanitizedChanges = sanitizeChanges();
        if (isEmpty(sanitizedChanges)) {
          onClose();
          setIssueDiscardModal(false);
        } else setIssueDiscardModal(true);
      } else {
        onClose();
        setIssueDiscardModal(false);
      }
    }
  };

  const handleCreateDraftIssue = async () => {
    if (!changesMade || isBusy() || descriptionLoadState !== "ready") return;
    if (!editorRef.current?.isEditorReadyToDiscard()) {
      setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("editor_is_not_ready_to_discard_changes") });
      return;
    }
    try {
      await props.onSubmit(changesMade, true);
    } catch {
      /* The command owner retains the form and reports the refusal. */
    }
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={requestClose}
      position={EModalPosition.TOP}
      width={props.isDuplicateModalOpen ? EModalWidth.VIXL : EModalWidth.XXXXL}
      className="rounded-lg !bg-transparent shadow-none transition-[width] ease-linear"
    >
      <ConfirmIssueDiscard
        isOpen={issueDiscardModal}
        handleClose={() => setIssueDiscardModal(false)}
        onConfirm={handleCreateDraftIssue}
        canSave={descriptionLoadState === "ready"}
        onDiscard={() => {
          if (isBusy()) return;
          onChange(null);
          setIssueDiscardModal(false);
          onClose();
        }}
      />
      <IssueFormRoot {...props} onClose={requestClose} />
    </ModalCore>
  );
});
