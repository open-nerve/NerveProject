/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import type { FC, FormEventHandler, RefObject } from "react";
import { useEffect } from "react";
import { observer } from "mobx-react";
import type { UseFormRegister } from "react-hook-form";
// nerve helpers
import { useOutsideClickDetector } from "@nerve/hooks";
// types

import { EIssueLayoutTypes } from "@nerve/types";
// components
import type { TQuickAddIssueForm } from "@/components/issues/issue-layouts/quick-add";
import {
  CalendarQuickAddIssueForm,
  KanbanQuickAddIssueForm,
  ListQuickAddIssueForm,
  SpreadsheetQuickAddIssueForm,
} from "@/components/issues/issue-layouts/quick-add";
// hooks
import { useProject } from "@/hooks/store/use-project";
import useKeypress from "@/hooks/use-keypress";

export type TQuickAddIssueFormRoot = {
  isOpen: boolean;
  layout: EIssueLayoutTypes;
  projectId: string;
  ref: RefObject<HTMLFormElement | null>;
  error: string | undefined;
  register: UseFormRegister<Pick<Issue, "name">>;
  onSubmit: FormEventHandler<HTMLFormElement>;
  onClose: () => void;
};

export const QuickAddIssueFormRoot = observer(function QuickAddIssueFormRoot(props: TQuickAddIssueFormRoot) {
  const { isOpen, layout, projectId, ref, error, register, onSubmit, onClose } = props;
  // store hooks
  const { getProjectById } = useProject();
  // derived values
  const projectDetail = getProjectById(projectId);
  // click detection
  useKeypress("Escape", onClose);
  useOutsideClickDetector(ref, onClose);
  // set focus on name input
  useEffect(() => {
    ref.current?.querySelector<HTMLInputElement>('input[name="name"]')?.focus();
  }, [ref]);

  if (!projectDetail) return <></>;

  const QUICK_ADD_ISSUE_FORMS: Record<EIssueLayoutTypes, FC<TQuickAddIssueForm>> = {
    [EIssueLayoutTypes.LIST]: ListQuickAddIssueForm,
    [EIssueLayoutTypes.KANBAN]: KanbanQuickAddIssueForm,
    [EIssueLayoutTypes.CALENDAR]: CalendarQuickAddIssueForm,
    [EIssueLayoutTypes.SPREADSHEET]: SpreadsheetQuickAddIssueForm,
  };

  const CurrentLayoutQuickAddIssueForm = QUICK_ADD_ISSUE_FORMS[layout] ?? null;

  if (!CurrentLayoutQuickAddIssueForm) return <></>;

  return (
    <CurrentLayoutQuickAddIssueForm
      ref={ref}
      isOpen={isOpen}
      projectDetail={projectDetail}
      error={error}
      register={register}
      onSubmit={onSubmit}
    />
  );
});
