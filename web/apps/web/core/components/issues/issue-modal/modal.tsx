/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import type { IssueFormData } from "./types";
import React from "react";
import { observer } from "mobx-react";
// nerve imports
import type { EIssuesStoreType, TWorkspaceDraftIssue } from "@nerve/types";
// local imports
import { CreateUpdateIssueModalBase } from "./base";
import { IssueModalProvider } from "./provider";

interface CommonIssuesModalProps {
  data?: IssueFormData;
  sourceIssueId?: string;
  isOpen: boolean;
  onClose: () => void;
  beforeFormSubmit?: () => Promise<void>;
  withDraftIssueWrapper?: boolean;
  storeType?: EIssuesStoreType;
  fetchIssueDetails?: boolean;
  moveToIssue?: boolean;
  modalTitle?: string;
  primaryButtonText?: {
    default: string;
    loading: string;
  };
  isProjectSelectionDisabled?: boolean;
  allowedProjectIds?: string[];
  showActionItemsOnUpdate?: boolean;
}

export type IssuesModalProps = CommonIssuesModalProps &
  (
    | { isDraft: true; onSubmit?: (res: TWorkspaceDraftIssue) => Promise<void> }
    | { isDraft?: false; onSubmit?: (res: Issue) => Promise<void> }
  );

export const CreateUpdateIssueModal = observer(function CreateUpdateIssueModal(props: IssuesModalProps) {
  if (!props.isOpen) return null;
  return (
    <IssueModalProvider allowedProjectIds={props.allowedProjectIds}>
      <CreateUpdateIssueModalBase {...props} />
    </IssueModalProvider>
  );
});
