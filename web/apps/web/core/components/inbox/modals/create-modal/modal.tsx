/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
// nerve imports
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// local imports
import { InboxIssueCreateRoot, type InboxIssueCreateHandle } from "./create-root";

type TInboxIssueCreateModalRoot = {
  workspaceSlug: string;
  projectId: string;
  modalState: boolean;
  handleModalClose: () => void;
};

/**
 * The create modal of a project's intake (P5b spec 2.1): Escape and its backdrop go through the form's close rule
 * (handleClose → requestClose), and while the form's submission is out the dialog has no handleClose at all (R28), as
 * AlertModalCore's: Escape then leaves the focus where it is, and the backdrop closes nothing.
 */
export function InboxIssueCreateModalRoot(props: TInboxIssueCreateModalRoot) {
  const { workspaceSlug, projectId, modalState, handleModalClose } = props;
  const form = useRef<InboxIssueCreateHandle>(null);
  const [submitting, setSubmitting] = useState(false);

  return (
    <ModalCore
      isOpen={modalState}
      handleClose={submitting ? undefined : () => form.current?.requestClose()}
      position={EModalPosition.TOP}
      width={EModalWidth.XXXXL}
      className="rounded-lg !bg-transparent shadow-none transition-[width] ease-linear"
    >
      <InboxIssueCreateRoot
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        handleModalClose={handleModalClose}
        onSubmittingChange={setSubmitting}
        closeRef={form}
      />
    </ModalCore>
  );
}
