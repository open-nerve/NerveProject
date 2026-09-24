/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import { useEffect, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { UseFormRegister } from "react-hook-form";
import { useForm } from "react-hook-form";
// plane imports
import { useTranslation } from "@nerve/i18n";
import { AddOutline } from "@makeplane/propel/icons";
import { setPromiseToast } from "@nerve/propel/toast";
import type { IProject, TIssue, EIssueLayoutTypes } from "@nerve/types";
import { cn, createIssuePayload } from "@nerve/utils";
// local imports
import { QuickAddIssueFormRoot } from "./form";
import { CreateIssueToastActionItems } from "../../create-issue-toast-action-items";

export type TQuickAddIssueForm = {
  ref: React.RefObject<HTMLFormElement | null>;
  isOpen: boolean;
  projectDetail: IProject;
  hasError: boolean;
  register: UseFormRegister<TIssue>;
  onSubmit: () => void;
};

export type TQuickAddIssueButton = {
  onClick: () => void;
};

type TQuickAddIssueRoot = {
  isQuickAddOpen?: boolean;
  layout: EIssueLayoutTypes;
  prePopulatedData?: Partial<TIssue>;
  QuickAddButton?: FC<TQuickAddIssueButton>;
  customQuickAddButton?: React.ReactNode;
  containerClassName?: string;
  setIsQuickAddOpen?: (isOpen: boolean) => void;
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
};

const defaultValues: Partial<TIssue> = {
  name: "",
};

export const QuickAddIssueRoot = observer(function QuickAddIssueRoot(props: TQuickAddIssueRoot) {
  const {
    isQuickAddOpen,
    layout,
    prePopulatedData,
    QuickAddButton,
    customQuickAddButton,
    containerClassName = "",
    setIsQuickAddOpen,
    quickAddCallback,
  } = props;
  // i18n
  const { t } = useTranslation();
  // router
  const { workspaceSlug, projectId } = useParams();
  // states
  const [isOpen, setIsOpen] = useState(isQuickAddOpen ?? false);
  // form info
  const {
    reset,
    handleSubmit,
    setFocus,
    register,
    formState: { errors, isSubmitting },
  } = useForm<TIssue>({ defaultValues });

  useEffect(() => {
    if (isQuickAddOpen !== undefined) {
      setIsOpen(isQuickAddOpen);
    }
  }, [isQuickAddOpen]);

  useEffect(() => {
    if (!isOpen) reset({ ...defaultValues });
  }, [isOpen, reset]);

  // oxlint-disable-next-line no-shadow
  const handleIsOpen = (isOpen: boolean) => {
    if (isQuickAddOpen !== undefined && setIsQuickAddOpen) {
      setIsQuickAddOpen(isOpen);
    } else {
      setIsOpen(isOpen);
    }
  };

  const onSubmitHandler = async (formData: TIssue) => {
    if (isSubmitting || !workspaceSlug || !projectId) return;

    reset({ ...defaultValues });

    const payload = createIssuePayload(projectId, {
      // oxlint-disable-next-line unicorn/no-useless-fallback-in-spread
      ...(prePopulatedData ?? {}),
      ...formData,
    });

    if (quickAddCallback) {
      const quickAddPromise = quickAddCallback(projectId, { ...payload });
      setPromiseToast<any>(quickAddPromise, {
        loading: t("issue.adding"),
        success: {
          title: t("common.success"),
          message: () => t("issue.create.success"),
          actionItems: (data) => (
            // TODO: Translate here
            <CreateIssueToastActionItems workspaceSlug={workspaceSlug} projectId={projectId} issueId={data.id} />
          ),
        },
        error: {
          title: t("common.error.label"),
          message: (err) => err?.message || t("common.error.message"),
        },
      });

      await quickAddPromise;
    }
  };

  if (!projectId) return null;

  return (
    <div
      className={cn(
        containerClassName,
        errors && errors?.name && errors?.name?.message ? `border-danger-strong bg-danger-subtle` : ``
      )}
    >
      {isOpen ? (
        <QuickAddIssueFormRoot
          isOpen={isOpen}
          layout={layout}
          prePopulatedData={prePopulatedData}
          projectId={projectId}
          // oxlint-disable-next-line no-unneeded-ternary
          hasError={errors && errors?.name && errors?.name?.message ? true : false}
          setFocus={setFocus}
          register={register}
          onSubmit={handleSubmit(onSubmitHandler)}
          onClose={() => handleIsOpen(false)}
        />
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
