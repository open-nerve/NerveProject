/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { CloseCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { ApiToken } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// components
import { DeleteApiTokenModal } from "@/components/api-token/delete-token-modal";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  token: ApiToken;
};

/** A token as lists show it: never the token itself, but when it was created and last used (M2 design 8.5). */
export function ApiTokenListItem(props: Props) {
  const { token } = props;
  // states
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const expired = token.expired_at !== null && new Date(token.expired_at).getTime() <= Date.now();
  const at = (time: string) => ({ date: renderFormattedDate(time), time: renderFormattedTime(time) });

  return (
    <>
      <DeleteApiTokenModal isOpen={deleteModalOpen} onClose={() => setDeleteModalOpen(false)} tokenId={token.id} />
      <div className="relative flex flex-col justify-center border-b border-subtle py-3">
        {/* Always shown, not on hover alone: the keyboard and touch reach it too. */}
        <Tooltip label={t("account_settings.api_tokens.revoke")} disabled={isMobile}>
          <button
            type="button"
            onClick={() => setDeleteModalOpen(true)}
            aria-label={t("account_settings.api_tokens.revoke")}
            className="absolute right-4 grid place-items-center"
          >
            <CloseCircleOutline className="h-4 w-4 text-danger-primary" />
          </button>
        </Tooltip>
        <div className="flex w-4/5 items-center">
          <h5 className="truncate text-13 font-medium">{token.label}</h5>
          <span
            className={`${
              expired ? "bg-layer-1 text-placeholder" : "bg-success-subtle text-success-primary"
            } ml-2 flex h-4 max-h-fit items-center rounded-xs px-2 text-11 font-medium`}
          >
            {expired ? t("account_settings.api_tokens.expired") : t("account_settings.api_tokens.active")}
          </span>
        </div>
        <div className="mt-1 flex w-full flex-col justify-center">
          {token.description.trim() !== "" && (
            <p className="mb-1 max-w-[70%] text-13 break-words">{token.description}</p>
          )}
          <p className="text-11 leading-6 text-placeholder">
            {token.expired_at === null
              ? t("workspace_settings.settings.api_tokens.never_expires")
              : t(`account_settings.api_tokens.${expired ? "expired_at" : "expires_at"}`, at(token.expired_at))}
          </p>
          <p className="text-11 leading-6 text-placeholder">
            {t("account_settings.api_tokens.created_at", { date: renderFormattedDate(token.created_at) })}
            {" · "}
            {token.last_used === null
              ? t("account_settings.api_tokens.never_used")
              : t("account_settings.api_tokens.last_used", at(token.last_used))}
          </p>
        </div>
      </div>
    </>
  );
}
