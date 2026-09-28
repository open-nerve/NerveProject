/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import useSWR from "swr";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// components
import { ApiTokenListItem } from "@/components/api-token/token-list-item";
import { APITokenSettingsLoader } from "@/components/ui/loader/settings/api-token";
// hooks
import { useApiTokens } from "@/hooks/store/user";
// lib
import { useSession } from "@/lib/auth/use-session";

type Props = {
  /** What shows when the account has no token. */
  empty: ReactNode;
};

/**
 * The personal access tokens of the account of the tab's session, each of which can be revoked here: the
 * api-tokens page and the security page (M2 design 7.7) show it.
 */
export const ApiTokenList = observer(function ApiTokenList(props: Props) {
  const { empty } = props;
  // store hooks
  const session = useSession();
  const { tokens, fetchTokens } = useApiTokens();
  const { t } = useTranslation();

  // The list is fetched for each session, into the store of its own RootStore (M2 design 7.1): another account
  // or another sign-in is another key, and nothing of the list before shows.
  const { error, isValidating, mutate } = useSWR(
    session.status === "signed-in" ? ["API_TOKENS", session.loginId] : null,
    () => fetchTokens(),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );

  if (tokens === undefined)
    return error && !isValidating ? (
      <div className="flex items-center gap-3 py-3 text-13 text-secondary">
        <span role="alert">{t("account_settings.api_tokens.load_failed")}</span>
        <Button variant="secondary" size="sm" onClick={() => void mutate()}>
          {t("common.retry")}
        </Button>
      </div>
    ) : (
      <APITokenSettingsLoader />
    );
  if (tokens.length === 0) return <>{empty}</>;
  return (
    <div>
      {tokens.map((token) => (
        <ApiTokenListItem key={token.id} token={token} />
      ))}
    </div>
  );
});
