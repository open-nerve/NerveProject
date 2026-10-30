/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueCreate } from "@nerve/api-client";

import type { FC, FormEventHandler } from "react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { UseFormRegister } from "react-hook-form";
import { useForm } from "react-hook-form";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { AddOutline } from "@makeplane/propel/icons";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { followInSession, sessionGuard } from "@/lib/in-session";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";
import { useFormRefusal } from "@/lib/form-refusal";
import { useRefusedFocus } from "@/lib/refused-focus";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { EUserPermissions } from "@nerve/constants";
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
  error: string | undefined;
  register: UseFormRegister<QuickAddValues>;
  onSubmit: FormEventHandler<HTMLFormElement>;
};

export type TQuickAddIssueButton = {
  onClick: () => void;
};

type TQuickAddIssueRoot = {
  /** Business host identity, including equal-payload groups such as labels:none and assignees:none. */
  scopeKey?: string;
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
  const { rootIssueStore: root } = useIssueDetail();
  const role =
    workspaceSlug && projectId
      ? root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
      : undefined;
  const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const identity = root.permissionReads.identity(workspace?.id, projectId ? [projectId] : []);
  if (
    !workspaceSlug ||
    !projectId ||
    !root.active ||
    (role !== EUserPermissions.MEMBER && role !== EUserPermissions.ADMIN)
  )
    return null;
  return (
    <QuickAddIssueScope
      key={JSON.stringify([
        workspaceSlug,
        projectId,
        identity,
        props.prePopulatedData,
        props.isQuickAddOpen,
        props.layout,
        props.scopeKey,
      ])}
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
  const capture = useDetailScope(workspaceSlug, projectId, JSON.stringify([isOpen, prePopulatedData]));
  const refuse = useFormRefusal(["name"] as const, {});
  const focusAfterSuccess = useRef<(() => boolean) | null>(null);
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
    register,
    setError,
    formState: { errors, submitCount },
  } = useForm<QuickAddValues>({ defaultValues, shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  useLayoutEffect(() => {
    if (pending || !focusAfterSuccess.current) return;
    const current = focusAfterSuccess.current;
    focusAfterSuccess.current = null;
    queueMicrotask(() => {
      if (current()) formRef.current?.querySelector<HTMLInputElement>('input[name="name"]')?.focus();
    });
  }, [pending, submitCount, formRef]);

  useEffect(() => {
    if (isQuickAddOpen !== undefined) {
      setIsOpen(isQuickAddOpen);
    }
  }, [isQuickAddOpen]);

  useEffect(() => {
    if (!isOpen && !busy.current) reset({ ...defaultValues });
  }, [isOpen, reset]);

  const handleIsOpen = (nextOpen: boolean) => {
    if (!nextOpen && (busy.current || pending)) return;
    if (isQuickAddOpen !== undefined && setIsQuickAddOpen) {
      setIsQuickAddOpen(nextOpen);
    } else {
      setIsOpen(nextOpen);
    }
  };

  const onSubmitHandler = async (formData: QuickAddValues) => {
    const sameScope = capture();
    const sameSession = sessionGuard();
    const current = () => sameScope() && sameSession();
    if (busy.current || !quickAddCallback || !current()) return;
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
            if (!current()) return;
            setToast({
              type: TOAST_TYPE.SUCCESS,
              title: t("common.success"),
              message: t("issue.create.success"),
              actionItems: (
                <CreateIssueToastActionItems workspaceSlug={workspaceSlug} projectId={projectId} issueId={issue.id} />
              ),
            });
            reset(defaultValues);
            focusAfterSuccess.current = current;
          },
          failed: (error) => {
            if (!current() || error instanceof SessionChangedError) return;
            refuse(error, (field, message) => setError(field, { type: "server", message }, { shouldFocus: false }));
          },
        }
      );
    } finally {
      busy.current = false;
      if (mounted.current) setPending(false);
    }
  };

  return (
    <div className={cn(containerClassName, errors.name?.message ? `border-danger-strong bg-danger-subtle` : ``)}>
      {isOpen ? (
        <fieldset disabled={pending}>
          <QuickAddIssueFormRoot
            isOpen={isOpen}
            layout={layout}
            projectId={projectId}
            ref={formRef}
            error={errors.name?.message}
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
