/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
import { Breadcrumbs } from "@nerve/ui";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  label?: string;
  href?: string;
  icon?: React.ReactNode;
  disableTooltip?: boolean;
  isLast?: boolean;
};

const IconWrapper = React.memo(function IconWrapper({ icon }: { icon: React.ReactNode }) {
  return <div className="flex size-4 items-center justify-center overflow-hidden !text-16">{icon}</div>;
});

IconWrapper.displayName = "IconWrapper";

const LabelWrapper = React.memo(function LabelWrapper({ label }: { label: React.ReactNode }) {
  return <div className="relative line-clamp-1 block max-w-[150px] truncate overflow-hidden text-primary">{label}</div>;
});

LabelWrapper.displayName = "LabelWrapper";

const BreadcrumbContent = React.memo(function BreadcrumbContent({
  icon,
  label,
}: {
  icon?: React.ReactNode;
  label?: React.ReactNode;
}) {
  if (!icon && !label) return null;

  return (
    <>
      {icon && <IconWrapper icon={icon} />}
      {label && <LabelWrapper label={label} />}
    </>
  );
});

BreadcrumbContent.displayName = "BreadcrumbContent";

const ItemWrapper = React.memo(function ItemWrapper({
  children,
  ...props
}: React.ComponentProps<typeof Breadcrumbs.ItemWrapper>) {
  return <Breadcrumbs.ItemWrapper {...props}>{children}</Breadcrumbs.ItemWrapper>;
});

ItemWrapper.displayName = "ItemWrapper";

export const BreadcrumbLink = observer(function BreadcrumbLink(props: Props) {
  const { href, label, icon, disableTooltip = false, isLast = false } = props;
  const { isMobile } = usePlatformOS();

  const itemWrapperProps = useMemo(
    (): Omit<React.ComponentProps<typeof ItemWrapper>, "children"> => ({
      label,
      disableTooltip: isMobile || disableTooltip,
      type: href && href !== "" ? "link" : "text",
      isLast,
    }),
    [href, label, isMobile, disableTooltip, isLast]
  );

  const content = useMemo(() => <BreadcrumbContent icon={icon} label={label} />, [icon, label]);

  if (href) {
    // The link is as high as its crumb (h-full of the item's h-6), not its wrapper's padded content, so the breadcrumbs'
    // box, which hides what overflows it, holds the link whole; the keyboard's ring is inside its edge.
    return (
      <Link
        to={href}
        className="h-full rounded-sm focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong"
      >
        <ItemWrapper {...itemWrapperProps}>{content}</ItemWrapper>
      </Link>
    );
  }

  return <ItemWrapper {...itemWrapperProps}>{content}</ItemWrapper>;
});
