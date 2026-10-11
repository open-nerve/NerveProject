/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { ComponentType, SVGProps } from "react";
import { useState } from "react";
import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { Avatar } from "@makeplane/propel/components/avatar";
import { ChevronDownOutline, DeactivatedUserOutline } from "@makeplane/propel/icons";
// nerve imports
import { EPillSize, EPillVariant, Pill } from "@nerve/propel/pill";
import type { MemberUser } from "@nerve/api-client";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { CustomSearchSelect } from "@nerve/ui";
// helpers
import { cn, getFileURL, sortByCurrentUserThenSelected } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useUser } from "@/hooks/store/user";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import { ButtonAvatars } from "./avatar";
import type { MemberDropdownProps } from "./types";

type TMemberDropdownBaseProps = {
  getUserDetails: (userId: string) => MemberUser | undefined;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  memberIds?: string[];
  onClose?: () => void;
  onDropdownOpen?: () => void;
  optionsClassName?: string;
  renderByDefault?: boolean;
} & MemberDropdownProps;

/**
 * Picks members: a CustomSearchSelect (@nerve/ui), whose list, with its search, opens beside its button, the open
 * state Headless UI's alone, the search taking the focus as the list opens on a desktop (M3 design 7.7). A dropdown
 * its caller defers (renderByDefault false: a row of a list on a desktop) is its button alone until the pointer
 * first comes over it, as ComboDropDown (@nerve/ui) does: no select and no popper for each row.
 */
export const MemberDropdownBase = observer(function MemberDropdownBase(props: TMemberDropdownBaseProps) {
  const { t } = useTranslation();
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
    getUserDetails,
    hideIcon = false,
    icon,
    memberIds,
    onClose,
    onDropdownOpen,
    optionsClassName = "",
    placeholder = t("members"),
    placement,
    renderByDefault = true,
    showTooltip = false,
    showUserDetails = false,
    tabIndex,
    tooltipContent,
    value,
  } = props;
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { data: currentUser } = useUser();
  const {
    workspace: { isUserSuspended },
  } = useMember();
  const { isMobile } = usePlatformOS();
  const [rendered, setRendered] = useState(renderByDefault);

  // what the button says: the member picked, how many are, or the placeholder
  const getDisplayName = () => {
    if (Array.isArray(value)) {
      if (value.length === 0) return placeholder;
      if (value.length === 1) return getUserDetails(value[0])?.display_name || placeholder;
      return showUserDetails ? `${value.length} ${t("members").toLocaleLowerCase()}` : "";
    }
    return showUserDetails && value ? getUserDetails(value)?.display_name || placeholder : placeholder;
  };

  // the button: alone while the dropdown is deferred, then in the select's button
  const button = (
    <DropdownButton
      // shown as active while the list is open (Headless UI's data-open on the select's button)
      className={cn("text-11 group-data-open:bg-layer-transparent-active", buttonClassName)}
      isActive={false}
      tooltipHeading={placeholder}
      tooltipContent={tooltipContent ?? `${value?.length ?? 0} ${value?.length !== 1 ? t("assignees") : t("assignee")}`}
      showTooltip={showTooltip}
      variant={buttonVariant}
      renderToolTipByDefault={renderByDefault}
      tabIndex={tabIndex}
    >
      {!hideIcon && <ButtonAvatars showTooltip={showTooltip} userIds={value} icon={icon} />}
      {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
        <span className="flex-grow truncate text-left text-body-xs-medium leading-5">{getDisplayName()}</span>
      )}
      {dropdownArrow && (
        <ChevronDownOutline className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
      )}
    </DropdownButton>
  );
  if (!rendered) {
    return (
      <div className="flex h-full items-center" onMouseEnter={() => setRendered(true)}>
        {button}
      </div>
    );
  }

  // the value and its handler as the select takes them: one member, or many
  const selection:
    | { multiple: false; value: string | null; onChange: (val: string | null) => void }
    | { multiple: true; value: string[]; onChange: (val: string[]) => void } = props.multiple
    ? { multiple: true, value: props.value, onChange: props.onChange }
    : { multiple: false, value: props.value, onChange: props.onChange };

  const options = sortByCurrentUserThenSelected<ICustomSearchSelectOption>(
    memberIds?.map((userId) => {
      const userDetails = getUserDetails(userId);
      const suspended = isUserSuspended(userId, workspaceSlug);
      return {
        value: userId,
        query: `${userDetails?.display_name} ${userDetails?.first_name} ${userDetails?.last_name}`,
        content: (
          <div className="flex items-center gap-2">
            <div className="w-4">
              {suspended ? (
                <DeactivatedUserOutline className="h-3.5 w-3.5 text-placeholder" />
              ) : (
                <Avatar
                  alt={userDetails?.display_name}
                  fallback={userDetails?.display_name?.[0]?.toUpperCase()}
                  src={getFileURL(userDetails?.avatar_url ?? "")}
                  size="xs"
                />
              )}
            </div>
            <span className={cn("flex-grow truncate", suspended ? "text-placeholder" : "")}>
              {currentUser?.id === userId ? t("you") : userDetails?.display_name}
            </span>
            {suspended && (
              <Pill variant={EPillVariant.DEFAULT} size={EPillSize.XS} className="border-none">
                Suspended
              </Pill>
            )}
          </div>
        ),
        disabled: suspended,
      };
    }),
    value,
    currentUser?.id
  );

  return (
    <CustomSearchSelect
      {...selection}
      options={options}
      onOpen={onDropdownOpen}
      onClose={onClose}
      disabled={disabled}
      // Tab stops once, at the DropdownButton in it, a button of its own, as in every dropdown of dropdowns/, where
      // the caller's tabIndex puts it: its Enter and Space reach the select's button, which opens the list
      tabIndex={-1}
      placement={placement}
      // As before the select was under it: the dropdown shrinks in its row (an inbox row's assignees), and its
      // button has no hover tint of its own behind the DropdownButton's (several callers turn that one off).
      className={cn("h-full shrink", className)}
      customButtonClassName={cn(
        "clickable group block h-full w-auto max-w-full outline-none hover:bg-transparent",
        buttonContainerClassName
      )}
      // the list's width, and the z-index a caller gives it
      optionsClassName={cn("w-48", optionsClassName)}
      searchPlaceholder={t("search")}
      noResultsMessage={t("no_matching_results")}
      loadingMessage={t("loading")}
      // on a phone the search takes no focus as the list opens: the keyboard would cover the list
      focusSearchOnOpen={!isMobile}
      // the list keeps 12 pixels inside the window
      popperModifiers={[{ name: "preventOverflow", options: { padding: 12 } }]}
      customButton={button}
    />
  );
});
