/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { ModalTitle } from "@nerve/ui";

type TInvitationFormProps = {
  title: string;
  description: React.ReactNode;
  children: React.ReactNode;
  onSubmit: () => void;
  /** The ref of the form's element (useWorkspaceInvitationActions' formRef). */
  formRef: React.Ref<HTMLFormElement>;
  actions: React.ReactNode;
  className?: string;
};

export const InvitationForm = observer(function InvitationForm(props: TInvitationFormProps) {
  const { title, description, children, actions, onSubmit, formRef, className } = props;

  return (
    <form ref={formRef} onSubmit={onSubmit} className={className}>
      <div className="space-y-4">
        <ModalTitle className="text-body-md-medium leading-6 text-primary">{title}</ModalTitle>
        <div className="text-body-xs-regular text-secondary">{description}</div>
        {children}
      </div>
      {actions}
    </form>
  );
});
