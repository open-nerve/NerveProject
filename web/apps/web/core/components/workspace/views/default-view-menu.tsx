/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { RefObject } from "react";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import type { DefaultWorkspaceViewKey } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { useQuickActionsFactory } from "@/components/common/quick-actions-factory";
// hooks
import { useCopyLink } from "@/hooks/use-copy-link";

type Props = {
  workspaceSlug: string;
  /** The default view: the key its address names, and its name's (M7/P6b spec 2.5). */
  view: { key: DefaultWorkspaceViewKey; i18n_label: string };
  /** The view's row, whose right-click opens the menu too; none in a header. */
  parentRef?: RefObject<HTMLElement | null>;
  customClassName?: string;
};

/**
 * A workspace's default view's quick actions (M7/P6b spec 2.8, 2.10, R12; re-97 I4): its menu, by a button named by the
 * view, and in a list by a right-click on the view's row (parentRef), opens its page in a new tab, which gets no hold
 * of this one, and copies its link (useCopyLink: said when copied, and when the browser does not let the page write the
 * clipboard). Nothing else: a default view is no view nerve holds, which anyone of the workspace has, and none changes
 * or deletes.
 */
export function DefaultViewMenu(props: Props) {
  const { workspaceSlug, view, parentRef, customClassName } = props;
  // nerve hooks
  const { t } = useTranslation();
  const factory = useQuickActionsFactory();
  const copyLink = useCopyLink();

  const link = `/${workspaceSlug}/workspace-views/${view.key}`;
  const items = [
    factory.createOpenInNewTabMenuItem(() => window.open(link, "_blank", "noopener,noreferrer")),
    factory.createCopyLinkMenuItem(() => void copyLink(link, t("common.link_copied_to_clipboard"))),
  ];

  return (
    <>
      {parentRef && <ContextMenu parentRef={parentRef} items={items} />}
      <CustomMenu
        ariaLabel={t("common.item_options", { name: t(view.i18n_label) })}
        customButton={<MoreHorizontalOutline className="size-4" aria-hidden="true" />}
        customButtonClassName={cn(getIconButtonStyling("tertiary", "lg"), customClassName)}
        placement="bottom-end"
        closeOnSelect
        maxHeight="lg"
      >
        {items.map((item) => (
          <CustomMenu.MenuItem key={item.key} onClick={() => item.action()} className="flex items-center gap-2">
            {item.icon && <item.icon className="h-3 w-3 flex-shrink-0" />}
            <h5>{item.title}</h5>
          </CustomMenu.MenuItem>
        ))}
      </CustomMenu>
    </>
  );
}
