/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { SITE_NAME } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// hooks
import { useMember } from "@/hooks/store/use-member";

/** Who made a change, as a page shows him: his name, and whether he is a member, one a page may link to. */
type ActorName = { name: string; member: boolean };

/**
 * Who made a change, as every page shows him (P4 spec R22): no account (null) is nerve itself, an automatic archive's
 * maker, by the site's name; an account the workspace's members do not list (they list the ended ones too,
 * getUserDetails) is a deactivated user, never shown by its id; else the member, by his display name. Read in an
 * observer: the members are the store's.
 */
export function useActorName(actorId: string | null): ActorName {
  const { t } = useTranslation();
  const { getUserDetails } = useMember();
  if (actorId === null) return { name: SITE_NAME, member: false };
  const user = getUserDetails(actorId);
  return user ? { name: user.display_name, member: true } : { name: t("common.deactivated_user"), member: false };
}
