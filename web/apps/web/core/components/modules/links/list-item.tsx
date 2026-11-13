/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { CopyOutline, DeleteOutline, EditOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import type { ModuleLink } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { calculateTimeAgo, getIconForLink } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useCopyText } from "@/hooks/use-copy-link";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  link: ModuleLink;
  /** Whether the caller changes and removes the link. */
  editable: boolean;
  /** Whether its removal is out: it is not asked again until nerve has answered. */
  removing: boolean;
  onEdit: () => void;
  onRemove: () => void;
};

const ACTION = "grid place-items-center rounded-sm p-1 text-secondary hover:bg-layer-transparent-hover";

/**
 * A link of a module (P6a spec 2.8, R10): its icon, the one nerve keeps inline in its metadata (a data:image), else
 * one by its address, never one the browser would fetch from elsewhere (favicon_url); its title, else its page's,
 * else its address, which opens in a new tab with no hold on this one; when it was added, and by whom while that
 * account is there. Every caller copies its address as it is; the caller who writes the module changes and removes it.
 */
export const ModuleLinkItem = observer(function ModuleLinkItem(props: Props) {
  const { link, editable, removing, onEdit, onRemove } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getUserDetails } = useMember();
  // hooks
  const { isMobile } = usePlatformOS();
  const copyText = useCopyText();
  // derived values
  const icon = link.metadata.favicon?.startsWith("data:image/") ? link.metadata.favicon : undefined;
  const Icon = getIconForLink(link.url);
  const name = link.title || link.metadata.title || link.url;
  const adder = link.created_by_id === null ? undefined : getUserDetails(link.created_by_id);
  const ago = calculateTimeAgo(link.created_at);

  return (
    <li className="relative flex flex-col rounded-md bg-layer-3 p-2.5">
      <div className="flex w-full items-start justify-between gap-2">
        <div className="flex items-start gap-2 truncate">
          <span className="py-1">
            {icon ? (
              <img src={icon} alt="" className="size-3 shrink-0" />
            ) : (
              <Icon className="size-3 shrink-0 stroke-2 text-tertiary group-hover:text-primary" />
            )}
          </span>
          <Tooltip label={link.url} layout="stacked" disabled={isMobile}>
            <a href={link.url} target="_blank" rel="noopener noreferrer" className="cursor-pointer truncate text-11">
              {name}
            </a>
          </Tooltip>
        </div>
        <div className="z-1 flex shrink-0 items-center">
          {editable && (
            <button type="button" aria-label={t("common.edit_item", { name })} className={ACTION} onClick={onEdit}>
              <EditOutline className="size-3 stroke-[1.5]" />
            </button>
          )}
          <button
            type="button"
            aria-label={`${t("common.actions.copy_link")}: ${name}`}
            className={ACTION}
            onClick={() => void copyText(link.url, t("common.link_copied_to_clipboard"))}
          >
            <CopyOutline className="size-3 stroke-[1.5]" />
          </button>
          {editable && (
            <button
              type="button"
              aria-label={t("common.delete_item", { name })}
              className={ACTION}
              disabled={removing}
              onClick={onRemove}
            >
              <DeleteOutline className="size-3 stroke-[1.5]" />
            </button>
          )}
        </div>
      </div>
      <p className="mt-0.5 px-5 text-11 text-tertiary">
        {adder
          ? t("project_module.link.added_by", { time: ago, name: adder.display_name })
          : t("project_module.link.added_at", { time: ago })}
      </p>
    </li>
  );
});
