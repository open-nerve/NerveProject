/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useUser } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";

/** How many of the members the filter shows until the person asks for them all. */
const FIRST = 5;

type Props = {
  /** What the members filter, in the page's language: a module's lead, or its members. */
  title: string;
  appliedFilters: readonly string[] | undefined;
  /** Turns the member on, or off if on. */
  handleUpdate: (memberId: string) => void;
  /** The members the filter offers, in the order to offer them. */
  memberIds: readonly string[];
  searchQuery: string;
};

/**
 * The options of the filters' menu for the members who lead a module, or are its members (P6a spec 2.4, R14): the
 * members given, in their order, whose names the menu's search finds, five of them until the person asks for all;
 * the caller as "you".
 */
export const FilterMembers = observer(function FilterMembers(props: Props) {
  const { title, appliedFilters, handleUpdate, memberIds, searchQuery } = props;
  // states
  const [showAll, setShowAll] = useState(false);
  const [previewEnabled, setPreviewEnabled] = useState(true);
  // hooks
  const { t } = useTranslation();
  const imageWorkspace = useRouteWorkspace();
  const { getUserDetails } = useMember();
  const { data: currentUser } = useUser();

  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const searched = searchQuery.toLowerCase();
  const members = memberIds.flatMap((memberId) => {
    const member = getUserDetails(memberId);
    return member?.display_name.toLowerCase().includes(searched) ? [member] : [];
  });
  const shown = showAll ? members : members.slice(0, FIRST);

  return (
    <>
      <FilterHeader
        title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {members.length > 0 ? (
            <>
              {shown.map((member) => (
                <FilterOption
                  key={member.id}
                  isChecked={!!appliedFilters?.includes(member.id)}
                  onClick={() => handleUpdate(member.id)}
                  icon={
                    <ProfileAvatar
                      alt={member.display_name}
                      fallback={member.display_name[0]?.toUpperCase()}
                      source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: member.id }}
                      size="xs"
                    />
                  }
                  title={currentUser?.id === member.id ? t("you") : member.display_name}
                />
              ))}
              {members.length > FIRST && (
                <button
                  type="button"
                  className="ml-8 text-11 font-medium text-accent-primary"
                  onClick={() => setShowAll(!showAll)}
                >
                  {t(showAll ? "common.view_less" : "common.view_all")}
                </button>
              )}
            </>
          ) : (
            <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
          )}
        </div>
      )}
    </>
  );
});
