import type { IntakeIssueCreate, IssueDetail } from "@nerve/api-client";
// oxlint-disable jsx_a11y/prefer-tag-over-role
// oxlint-disable jsx_a11y/click-events-have-key-events
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FormEvent } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { ETabIndices } from "@nerve/constants";
import type { EditorRefApi } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";

import { Switch } from "@makeplane/propel/components/switch";
import { renderFormattedPayloadDate, getTabIndex } from "@nerve/utils";
// hooks
import { useIntake } from "@/hooks/store/use-intake";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useNavigate } from "react-router";
import useKeypress from "@/hooks/use-keypress";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { InboxIssueDescription } from "./issue-description";
import { InboxIssueProperties } from "./issue-properties";
import { InboxIssueTitle } from "./issue-title";

type TInboxIssueCreateRoot = {
  workspaceSlug: string;
  projectId: string;
  handleModalClose: () => void;
  isDuplicateModalOpen: boolean;
  handleDuplicateIssueModal: (value: boolean) => void;
};

const defaultIssueData: Partial<IssueDetail> = {
  name: "",
  description_html: "",
  priority: "none",
  label_ids: [],
  assignee_ids: [],
  start_date: renderFormattedPayloadDate(new Date()),
  target_date: "",
};

export const InboxIssueCreateRoot = observer(function InboxIssueCreateRoot(props: TInboxIssueCreateRoot) {
  const { workspaceSlug, projectId, handleModalClose } = props;
  // router
  const navigate = useNavigate();
  // refs
  const descriptionEditorRef = useRef<EditorRefApi>(null);
  const submitBtnRef = useRef<HTMLButtonElement | null>(null);
  const formRef = useRef<HTMLFormElement | null>(null);
  const modalContainerRef = useRef<HTMLDivElement | null>(null);
  // hooks
  const { submit } = useIntake();
  const refusalToast = useRefusalToast();
  const { getWorkspaceBySlug } = useWorkspace();
  const workspaceId = getWorkspaceBySlug(workspaceSlug)?.id;
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  // states
  const [createMore, setCreateMore] = useState<boolean>(false);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [formData, setFormData] = useState<Partial<IssueDetail>>(defaultIssueData);
  const handleFormData = useCallback(
    <T extends keyof Partial<IssueDetail>>(issueKey: T, issueValue: Partial<IssueDetail>[T]) => {
      setFormData({
        ...formData,
        [issueKey]: issueValue,
      });
    },
    [formData]
  );

  const { getIndex } = getTabIndex(ETabIndices.INTAKE_ISSUE_FORM, isMobile);

  const handleEscKeyDown = (event: KeyboardEvent) => {
    if (descriptionEditorRef.current?.isEditorReadyToDiscard()) {
      handleModalClose();
    } else {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Editor is still processing changes. Please wait before proceeding.",
      });
      event.preventDefault(); // Prevent default action if editor is not ready to discard
    }
  };

  useKeypress("Escape", handleEscKeyDown);

  useEffect(() => {
    const formElement = formRef?.current;
    const modalElement = modalContainerRef?.current;

    if (!formElement || !modalElement) return;

    const resizeObserver = new ResizeObserver(() => {
      modalElement.style.maxHeight = `${formElement?.offsetHeight}px`;
    });

    resizeObserver.observe(formElement);

    return () => {
      resizeObserver.disconnect();
    };
  }, [formRef, modalContainerRef]);

  const handleFormSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!workspaceId) return;

    if (!descriptionEditorRef.current?.isEditorReadyToDiscard()) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Editor is still processing changes. Please wait before proceeding.",
      });
      return;
    }

    // nerve takes a day due, or no key for none (IntakeIssueCreate, C10)
    const body: IntakeIssueCreate = {
      name: formData.name || "",
      description_html: formData.description_html || "<p></p>",
      priority: formData.priority || "none",
      label_ids: formData.label_ids || [],
      assignee_ids: formData.assignee_ids || [],
      ...(formData.target_date ? { target_date: formData.target_date } : {}),
    };
    const scope = { workspaceSlug, projectID: projectId, workspaceID: workspaceId };
    setFormSubmitting(true);

    await followInSession(() => submit(scope, body), {
      done: (submitted) => {
        if (!createMore) {
          navigate(`/${workspaceSlug}/projects/${projectId}/intake?currentTab=open&inboxIssueId=${submitted.id}`);
          handleModalClose();
        } else {
          descriptionEditorRef?.current?.clearEditor();
          setFormData(defaultIssueData);
        }
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: `Success!`,
          message: "Work item created successfully.",
        });
      },
      failed: refusalToast,
    });
    setFormSubmitting(false);
  };

  const isTitleLengthMoreThan255Character = formData?.name ? formData.name.length > 255 : false;

  if (!workspaceSlug || !projectId || !workspaceId) return <></>;
  return (
    <div className="flex w-full gap-2 bg-transparent">
      <div className="w-full rounded-lg">
        <form ref={formRef} onSubmit={handleFormSubmit} className="flex w-full flex-col">
          <div className="space-y-5 rounded-t-lg bg-surface-1 p-5">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-18 font-medium text-secondary">{t("inbox_issue.modal.title")}</h3>
            </div>
            <div className="space-y-3">
              <InboxIssueTitle
                data={formData}
                handleData={handleFormData}
                isTitleLengthMoreThan255Character={isTitleLengthMoreThan255Character}
              />
              <InboxIssueDescription
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                workspaceId={workspaceId}
                data={formData}
                handleData={handleFormData}
                editorRef={descriptionEditorRef}
                containerClassName="bg-layer-2 border-[0.5px] border-subtle-1 py-3 min-h-[150px]"
                onEnterKeyPress={() => submitBtnRef?.current?.click()}
              />
              <InboxIssueProperties projectId={projectId} data={formData} handleData={handleFormData} />
            </div>
          </div>
          <div className="flex items-center justify-between gap-2 rounded-b-lg border-t-[0.5px] border-subtle bg-surface-1 px-5 py-4">
            <div
              className="inline-flex cursor-pointer items-center gap-1.5"
              onClick={() => setCreateMore((prevData) => !prevData)}
              role="button"
              tabIndex={getIndex("create_more")}
            >
              <Switch size="sm" checked={createMore} onCheckedChange={() => {}} aria-label={t("create_more")} />
              <span className="text-11">{t("create_more")}</span>
            </div>
            <div className="flex items-center gap-3">
              <Button
                variant="secondary"
                size="lg"
                type="button"
                onClick={() => {
                  if (descriptionEditorRef.current?.isEditorReadyToDiscard()) {
                    handleModalClose();
                  } else {
                    setToast({
                      type: TOAST_TYPE.ERROR,
                      title: "Error!",
                      message: "Editor is still processing changes. Please wait before proceeding.",
                    });
                  }
                }}
                tabIndex={getIndex("discard_button")}
              >
                {t("discard")}
              </Button>
              <Button
                variant="primary"
                ref={submitBtnRef}
                type="submit"
                loading={formSubmitting}
                disabled={isTitleLengthMoreThan255Character}
                tabIndex={getIndex("submit_button")}
                size="lg"
              >
                {formSubmitting ? t("creating") : t("create_work_item")}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
});
