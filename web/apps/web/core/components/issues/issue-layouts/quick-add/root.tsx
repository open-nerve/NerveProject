/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueCreate } from "@nerve/api-client";

import type { FC } from "react";
import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { UseFormRegister } from "react-hook-form";
import { useForm } from "react-hook-form";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { AddOutline } from "@makeplane/propel/icons";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { followInSession } from "@/lib/in-session";
import type { Project } from "@nerve/api-client";
import type { EIssueLayoutTypes } from "@nerve/types";
import { cn } from "@nerve/utils";
// local imports
import { QuickAddIssueFormRoot } from "./form";
import { CreateIssueToastActionItems } from "../../create-issue-toast-action-items";

type QuickAddValues = Pick<Issue, "name">;
type QuickAddPrefill = Partial<
  Pick<Issue, "state_id" | "priority" | "assignee_ids" | "label_ids" | "start_date" | "target_date" | "parent_id">
>;

export type TQuickAddIssueForm = {
  ref: React.RefObject<HTMLFormElement | null>;
  isOpen: boolean;
  projectDetail: Project;
  hasError: boolean;
  register: UseFormRegister<QuickAddValues>;
  onSubmit: () => void;
};

export type TQuickAddIssueButton = {
  onClick: () => void;
};

type TQuickAddIssueRoot = {
  isQuickAddOpen?: boolean;
  layout: EIssueLayoutTypes;
  prePopulatedData?: QuickAddPrefill;
  QuickAddButton?: FC<TQuickAddIssueButton>;
  customQuickAddButton?: React.ReactNode;
  containerClassName?: string;
  setIsQuickAddOpen?: (isOpen: boolean) => void;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
};

const defaultValues: QuickAddValues = {
  name: "",
};

export const QuickAddIssueRoot = observer(function QuickAddIssueRoot(props: TQuickAddIssueRoot) {
  const { workspaceSlug, projectId } = useParams();
  if (!workspaceSlug || !projectId) return null;
  return (
    <QuickAddIssueScope
      key={`${workspaceSlug}/${projectId}`}
      {...props}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
    />
  );
});

const QuickAddIssueScope = observer(function QuickAddIssueScope(
  props: TQuickAddIssueRoot & { workspaceSlug: string; projectId: string }
) {
  const {
    isQuickAddOpen,
    layout,
    prePopulatedData,
    QuickAddButton,
    customQuickAddButton,
    containerClassName = "",
    setIsQuickAddOpen,
    quickAddCallback,
    workspaceSlug,
    projectId,
  } = props;
  // i18n
  const { t } = useTranslation();
  // states
  const [isOpen, setIsOpen] = useState(isQuickAddOpen ?? false);
  const [pending, setPending] = useState(false);
  const busy = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  // form info
  const {
    reset,
    handleSubmit,
    setFocus,
    register,
    formState: { errors },
  } = useForm<QuickAddValues>({ defaultValues });

  useEffect(() => {
    if (isQuickAddOpen !== undefined) {
      setIsOpen(isQuickAddOpen);
    }
  }, [isQuickAddOpen]);

  useEffect(() => {
    if (!isOpen && !busy.current) reset({ ...defaultValues });
  }, [isOpen, reset]);

  // oxlint-disable-next-line no-shadow
  const handleIsOpen = (isOpen: boolean) => {
    if (!isOpen && (busy.current || pending)) return;
    if (isQuickAddOpen !== undefined && setIsQuickAddOpen) {
      setIsQuickAddOpen(isOpen);
    } else {
      setIsOpen(isOpen);
    }
  };

  const onSubmitHandler = async (formData: QuickAddValues) => {
    if (busy.current || !quickAddCallback) return;
    busy.current = true;
    setPending(true);
    try {
      // Only caller-owned editable values enter the existing create input contract.
      const { state_id, priority, assignee_ids, label_ids, start_date, target_date, parent_id } =
        prePopulatedData ?? {};
      await followInSession(
        () =>
          quickAddCallback(projectId, {
            ...formData,
            ...(state_id !== undefined && { state_id }),
            ...(priority !== undefined && { priority }),
            ...(assignee_ids !== undefined && { assignee_ids }),
            ...(label_ids !== undefined && { label_ids }),
            ...(start_date != null && { start_date }),
            ...(target_date != null && { target_date }),
            ...(parent_id != null && { parent_id }),
          }).then((issue) => {
            if (!issue) throw new Error(t("common.error.message"));
            return issue;
          }),
        {
          done: (issue) => {
            setToast({
              type: TOAST_TYPE.SUCCESS,
              title: t("common.success"),
              message: t("issue.create.success"),
              actionItems: (
                <CreateIssueToastActionItems workspaceSlug={workspaceSlug} projectId={projectId} issueId={issue.id} />
              ),
            });
            if (mounted.current) reset(defaultValues);
          },
          failed: (error) => {
            setToast({
              type: TOAST_TYPE.ERROR,
              title: t("common.error.label"),
              message: error instanceof Error ? error.message : t("common.error.message"),
            });
          },
        }
      );
    } finally {
      busy.current = false;
      if (mounted.current) setPending(false);
    }
  };

  return (
    <div
      className={cn(
        containerClassName,
        errors && errors?.name && errors?.name?.message ? `border-danger-strong bg-danger-subtle` : ``
      )}
    >
      {isOpen ? (
        <fieldset disabled={pending}>
          <QuickAddIssueFormRoot
            isOpen={isOpen}
            layout={layout}
            projectId={projectId}
            // oxlint-disable-next-line no-unneeded-ternary
            hasError={errors && errors?.name && errors?.name?.message ? true : false}
            setFocus={setFocus}
            register={register}
            onSubmit={handleSubmit(onSubmitHandler)}
            onClose={() => handleIsOpen(false)}
          />
          {pending && <div role="status">{t("issue.adding")}</div>}
        </fieldset>
      ) : (
        <>
          {QuickAddButton && <QuickAddButton onClick={() => handleIsOpen(true)} />}
          {customQuickAddButton && <>{customQuickAddButton}</>}
          {!QuickAddButton && !customQuickAddButton && (
            <button
              className="flex w-full cursor-pointer items-center gap-2 bg-layer-transparent px-2 py-3 hover:bg-layer-transparent-hover"
              onClick={() => handleIsOpen(true)}
            >
              <AddOutline className="h-3.5 w-3.5" />
              <span className="text-13 font-medium">{t("issue.new")}</span>
            </button>
          )}
        </>
      )}
    </div>
  );
});
