/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IntakeIssueCreate, IssueDetail } from "@nerve/api-client";
import type { FormEvent, Ref } from "react";
import { useCallback, useEffect, useImperativeHandle, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import { Switch } from "@makeplane/propel/components/switch";
import type { EditorRefApi } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { AlertModalCore, ModalTitle } from "@nerve/ui";
import { isEmptyHtmlString, renderFormattedPayloadDate } from "@nerve/utils";
// hooks
import { useIntake } from "@/hooks/store/use-intake";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";
import { useRefusedFocus } from "@/lib/refused-focus";
// local imports
import { intakeAddress } from "../../address";
import { InboxIssueDescription } from "./issue-description";
import { InboxIssueProperties } from "./issue-properties";
import { InboxIssueTitle } from "./issue-title";

/** What the form gives its modal: its close rule, which Escape and the backdrop go through (P5b spec 2.1, R4). */
export type InboxIssueCreateHandle = { requestClose: () => void };

type TInboxIssueCreateRoot = {
  workspaceSlug: string;
  projectId: string;
  handleModalClose: () => void;
  /** Called as the submission goes out and as it settles: the modal has no handleClose meanwhile (R28). */
  onSubmittingChange: (submitting: boolean) => void;
  closeRef?: Ref<InboxIssueCreateHandle>;
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

/** The longest name nerve takes, in characters (the contract's IntakeIssueCreate.name). */
const NAME_MAX = 255;

/** A name's length as nerve counts it (the contract; the server trims it first): its characters, trimmed. */
const nameLength = (name: string | undefined) => Array.from((name ?? "").trim()).length;

/**
 * Whether the form holds something the user would lose by closing it: a name (spaces alone are none), a description
 * with content (the editor writes an empty paragraph as it is built: isEmptyHtmlString), a priority, an assignee, a
 * label or a due date (R4, R25 (1)).
 */
const isChanged = (data: Partial<IssueDetail>) =>
  (data.name ?? "").trim() !== "" ||
  !isEmptyHtmlString(data.description_html ?? "") ||
  (data.priority ?? "none") !== "none" ||
  (data.assignee_ids ?? []).length > 0 ||
  (data.label_ids ?? []).length > 0 ||
  !!data.target_date;

/**
 * The form of the create modal of a project's intake (P5b spec 2.1). Its one close rule (requestClose), which the
 * modal's Escape and backdrop (handleClose) and Discard go through: not while the submission is out; while the
 * description's editor is built but not ready to discard, it says so (its ref null, the editor's chunk not built yet,
 * holds nothing to lose); a form with changes asks first (the discard confirmation), one without closes. A success
 * closes it directly (or empties it, Create more on). The name follows the contract's rule: a hint under it, too long
 * as one types, blank once a submit is refused; a refused submit, nerve's refusal of the name too, gives it the focus.
 * The follow-ups of the submission are the form's only while it is mounted (R29): Back and Forward leave the page
 * though the dialog cannot close, and a late answer is then another page's.
 */
export const InboxIssueCreateRoot = observer(function InboxIssueCreateRoot(props: TInboxIssueCreateRoot) {
  const { workspaceSlug, projectId, handleModalClose, onSubmittingChange, closeRef } = props;
  // router
  const navigate = useNavigate();
  // refs
  const descriptionEditorRef = useRef<EditorRefApi>(null);
  const submitBtnRef = useRef<HTMLButtonElement | null>(null);
  const mounted = useRef(false);
  // hooks
  const { submit } = useIntake();
  const showRefusal = useFormRefusal(["name"], {});
  const { getWorkspaceBySlug } = useWorkspace();
  const workspaceId = getWorkspaceBySlug(workspaceSlug)?.id;
  const { t } = useTranslation();
  // states
  const [createMore, setCreateMore] = useState<boolean>(false);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [discarding, setDiscarding] = useState(false);
  const [formData, setFormData] = useState<Partial<IssueDetail>>(defaultIssueData);
  // the name's refusal shown under it until the name is no longer blank: a blank submit's, or nerve's
  const [nameRefusal, setNameRefusal] = useState<string>();
  // each refused submit, which gives the refused name the focus
  const [refusedAttempt, setRefusedAttempt] = useState(0);
  const formRef = useRefusedFocus<HTMLFormElement>(refusedAttempt);
  const handleFormData = useCallback(
    <T extends keyof Partial<IssueDetail>>(issueKey: T, issueValue: Partial<IssueDetail>[T]) => {
      setFormData({
        ...formData,
        [issueKey]: issueValue,
      });
      if (issueKey === "name" && typeof issueValue === "string" && nameLength(issueValue) > 0)
        setNameRefusal(undefined);
    },
    [formData]
  );

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const setSubmitting = (submitting: boolean) => {
    setFormSubmitting(submitting);
    onSubmittingChange(submitting);
  };

  const sayEditorNotReady = () =>
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t("editor_is_not_ready_to_discard_changes") });

  const requestClose = () => {
    if (formSubmitting) return;
    const editor = descriptionEditorRef.current;
    if (editor && !editor.isEditorReadyToDiscard()) {
      sayEditorNotReady();
      return;
    }
    if (isChanged(formData)) setDiscarding(true);
    else handleModalClose();
  };
  useImperativeHandle(closeRef, () => ({ requestClose }));

  const refuseName = (message?: string) => {
    if (message !== undefined) setNameRefusal(message);
    setRefusedAttempt((attempt) => attempt + 1);
  };

  const handleFormSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!workspaceId) return;

    const length = nameLength(formData.name);
    if (length === 0 || length > NAME_MAX) {
      refuseName(length === 0 ? t("title_is_required") : undefined);
      return;
    }
    if (!descriptionEditorRef.current?.isEditorReadyToDiscard()) {
      sayEditorNotReady();
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
    setSubmitting(true);

    await followInSession(() => submit(scope, body), {
      done: (submitted) => {
        if (!mounted.current) return;
        if (!createMore) {
          navigate(intakeAddress(workspaceSlug, projectId, submitted.id, submitted.intake));
          handleModalClose();
        } else {
          descriptionEditorRef?.current?.clearEditor();
          setFormData(defaultIssueData);
          setNameRefusal(undefined);
        }
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t("issue_created_successfully") });
      },
      failed: (error) => {
        if (!mounted.current) return;
        showRefusal(error, (_name, message) => refuseName(message));
      },
    });
    setSubmitting(false);
  };

  const nameError = nameLength(formData.name) > NAME_MAX ? t("errors.field.too_long") : nameRefusal;

  if (!workspaceSlug || !projectId || !workspaceId) return <></>;
  return (
    <div className="flex w-full gap-2 bg-transparent">
      <div className="w-full rounded-lg">
        <form ref={formRef} onSubmit={handleFormSubmit} className="flex w-full flex-col">
          <div className="space-y-5 rounded-t-lg bg-surface-1 p-5">
            <div className="flex items-center justify-between gap-2">
              <ModalTitle className="text-18 font-medium text-secondary">{t("inbox_issue.modal.title")}</ModalTitle>
            </div>
            <div className="space-y-3">
              <InboxIssueTitle data={formData} handleData={handleFormData} error={nameError} />
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
            <div className="inline-flex items-center gap-1.5">
              <Switch
                size="sm"
                checked={createMore}
                onCheckedChange={(checked) => setCreateMore(checked)}
                disabled={formSubmitting}
                aria-label={t("create_more")}
              />
              <span className="text-11">{t("create_more")}</span>
            </div>
            <div className="flex items-center gap-3">
              <Button variant="secondary" size="lg" type="button" disabled={formSubmitting} onClick={requestClose}>
                {t("discard")}
              </Button>
              <Button variant="primary" ref={submitBtnRef} type="submit" loading={formSubmitting} size="lg">
                {formSubmitting ? t("creating") : t("create_work_item")}
              </Button>
            </div>
          </div>
        </form>
      </div>
      <AlertModalCore
        isOpen={discarding}
        variant="danger"
        title={t("inbox_issue.modals.discard.title")}
        content={t("inbox_issue.modals.discard.content")}
        primaryButtonText={{ loading: t("discard"), default: t("discard") }}
        secondaryButtonText={t("common.cancel")}
        isSubmitting={false}
        handleClose={() => setDiscarding(false)}
        handleSubmit={() => {
          setDiscarding(false);
          handleModalClose();
        }}
      />
    </div>
  );
});
