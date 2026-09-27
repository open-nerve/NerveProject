/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
// nerve imports
import type { ApiTokenCreate, ApiTokenCreated } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { renderFormattedDate, csvDownload } from "@nerve/utils";
// hooks
import { useApiTokens } from "@/hooks/store/user";
// local imports
import { CreateApiTokenForm } from "./form";
import { GeneratedTokenDetails } from "./generated-token-details";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function CreateApiTokenModal(props: Props) {
  const { isOpen, onClose } = props;
  // store hooks
  const { createToken } = useApiTokens();
  // states
  const [neverExpires, setNeverExpires] = useState<boolean>(false);
  // The new token with the token itself: only here, while the modal shows it (M2 design 7.7).
  const [generatedToken, setGeneratedToken] = useState<ApiTokenCreated | null>(null);
  const [wasOpen, setWasOpen] = useState(isOpen);
  // How many times an opening has ended, by a close or by leaving the page. A create belongs to the opening it
  // was made in: answered after that opening ended (Cancel stays clickable while it is out), it keeps no secret
  // and downloads no CSV; the list shows the token, which can be revoked there. A count, not an open flag, so a
  // reopening in the meantime does not take the answer as its own.
  const endedOpenings = useRef(0);
  const { t } = useTranslation();

  // Each opening starts with the form: set while rendering, so that no frame of an opening shows the token of
  // the last, not even one opened before the reset after closing has run.
  if (isOpen !== wasOpen) {
    setWasOpen(isOpen);
    if (isOpen) {
      setNeverExpires(false);
      setGeneratedToken(null);
    }
  }

  // Leaving the page ends the opening too.
  useEffect(
    () => () => {
      endedOpenings.current += 1;
    },
    []
  );

  const handleClose = () => {
    endedOpenings.current += 1;
    onClose();

    setTimeout(() => {
      setNeverExpires(false);
      setGeneratedToken(null);
    }, 350);
  };

  const downloadSecretKey = (data: ApiTokenCreated) => {
    const csvData = {
      Title: data.label,
      Description: data.description,
      Expiry: data.expired_at
        ? (renderFormattedDate(data.expired_at) ?? "")
        : t("workspace_settings.settings.api_tokens.never_expires"),
      "Secret key": data.token,
    };

    csvDownload(csvData, `secret-key-${Date.now()}`);
  };

  const handleCreateToken = async (data: ApiTokenCreate) => {
    const opening = endedOpenings.current;
    const created = await createToken(data);
    if (endedOpenings.current !== opening) return;
    setGeneratedToken(created);
    downloadSecretKey(created);
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={() => {}} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      {generatedToken ? (
        <GeneratedTokenDetails handleClose={handleClose} tokenDetails={generatedToken} />
      ) : (
        <CreateApiTokenForm
          handleClose={handleClose}
          neverExpires={neverExpires}
          toggleNeverExpires={() => setNeverExpires((prevData) => !prevData)}
          onSubmit={handleCreateToken}
        />
      )}
    </ModalCore>
  );
}
