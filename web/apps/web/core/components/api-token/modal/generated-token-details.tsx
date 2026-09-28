/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ApiTokenCreated } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { CopyOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// ui
import { renderFormattedDate, renderFormattedTime, copyTextToClipboard } from "@nerve/utils";
// types
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  handleClose: () => void;
  /** The new token, with the token itself, which shows here this once. */
  tokenDetails: ApiTokenCreated;
};

export function GeneratedTokenDetails(props: Props) {
  const { handleClose, tokenDetails } = props;
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const copyApiToken = (token: string) => {
    void copyTextToClipboard(token)
      .then(() =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: `${t("success")}!`,
          message: t("workspace_settings.token_copied"),
        })
      )
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: `${t("error")}!`,
          message: t("workspace_settings.token_not_copied"),
        })
      );
  };

  return (
    <div className="w-full p-5">
      <div className="w-full space-y-3 text-wrap">
        <h3 className="text-16 leading-6 font-medium text-primary">{t("workspace_settings.key_created")}</h3>
        <p className="text-13 text-placeholder">{t("workspace_settings.copy_key")}</p>
      </div>
      <button
        type="button"
        onClick={() => copyApiToken(tokenDetails.token)}
        className="mt-4 flex w-full items-center justify-between truncate rounded-md border-[0.5px] border-subtle px-3 py-2 text-13 font-medium outline-none"
      >
        <span className="truncate pr-2">{tokenDetails.token}</span>
        <Tooltip label={t("account_settings.api_tokens.copy")} disabled={isMobile}>
          <CopyOutline className="h-4 w-4 flex-shrink-0 text-placeholder" />
        </Tooltip>
      </button>
      <div className="mt-6 flex items-center justify-between">
        <p className="text-11 text-placeholder">
          {tokenDetails.expired_at
            ? t("account_settings.api_tokens.expires_at", {
                date: renderFormattedDate(tokenDetails.expired_at),
                time: renderFormattedTime(tokenDetails.expired_at),
              })
            : t("workspace_settings.settings.api_tokens.never_expires")}
        </p>
        <Button variant="secondary" onClick={handleClose}>
          {t("close")}
        </Button>
      </div>
    </div>
  );
}
