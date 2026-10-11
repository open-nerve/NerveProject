/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";

// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { AddOutline } from "@makeplane/propel/icons";

type TInvitationModalActionsProps = {
  isSubmitting?: boolean;
  handleClose: () => void;
  appendField: () => void;
};

export const InvitationModalActions = observer(function InvitationModalActions(props: TInvitationModalActionsProps) {
  const { isSubmitting = false, handleClose, appendField } = props;
  // store hooks
  const { t } = useTranslation();

  return (
    <div className="mt-5 flex items-center justify-between gap-2">
      <button
        type="button"
        className="flex items-center gap-1 bg-transparent py-2 pr-3 text-caption-md-medium text-accent-primary outline-accent-strong"
        onClick={appendField}
      >
        <AddOutline className="h-3.5 w-3.5" />
        {t("common.add_more")}
      </button>
      <div className="flex items-center gap-2">
        {/* Disabled while the invitations are out: nerve's answer is said in the form that sent them */}
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
          {t("cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
          {isSubmitting
            ? t("workspace_settings.settings.members.modal.button_loading")
            : t("workspace_settings.settings.members.modal.button")}
        </Button>
      </div>
    </div>
  );
});
