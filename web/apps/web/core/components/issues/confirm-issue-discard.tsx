/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
// ui
import { Button } from "@nerve/propel/button";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useTranslation } from "@nerve/i18n";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  onDiscard: () => void;
  onConfirm: () => Promise<void>;
  canSave?: boolean;
};

export function ConfirmIssueDiscard(props: Props) {
  const { isOpen, handleClose, onDiscard, onConfirm, canSave = true } = props;
  const { t } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const busy = useRef(false);

  const onClose = () => {
    if (!busy.current) handleClose();
  };

  const handleDeletion = async () => {
    if (busy.current || !canSave) return;
    busy.current = true;
    setIsLoading(true);
    try {
      await onConfirm();
    } finally {
      busy.current = false;
      setIsLoading(false);
    }
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <div className="px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <div className="sm:flex sm:items-start">
          <div className="mt-3 text-center sm:mt-0 sm:text-left">
            <ModalTitle className="text-16 leading-6 font-medium text-primary">
              {t("issue.draft.save_prompt.title")}
            </ModalTitle>
            <div className="mt-2">
              <p className="text-13 text-secondary">{t("issue.draft.save_prompt.description")}</p>
            </div>
          </div>
        </div>
      </div>
      <div className="flex justify-between gap-2 p-4 sm:px-6">
        <div>
          <Button
            variant="secondary"
            onClick={() => {
              if (!busy.current) onDiscard();
            }}
            disabled={isLoading}
          >
            {t("discard")}
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            {t("cancel")}
          </Button>
          <Button variant="primary" onClick={handleDeletion} loading={isLoading} disabled={isLoading || !canSave}>
            {isLoading ? t("saving") : t("save_to_drafts")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
