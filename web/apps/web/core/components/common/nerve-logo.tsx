/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { cn } from "@nerve/utils";
// assets
import lockupOnDark from "@/app/assets/brand/lockup-on-dark.svg?url";
import lockup from "@/app/assets/brand/lockup.svg?url";
import mark from "@/app/assets/brand/mark.svg?url";

// The brand is a set of files in app/assets/brand/ (SOURCES.md there): a new logo replaces the files, not this code.
// The images keep their own proportions, so give them a height and an automatic width.

type TProps = {
  className?: string;
};

/** The Nerve mark. It reads on light and dark backgrounds alike. */
export function NerveLogo({ className }: TProps) {
  return <img src={mark} alt="Nerve" className={className} />;
}

/**
 * The Nerve lockup: the mark and the wordmark. The wordmark follows the theme through CSS, as the theme is known
 * only in the browser; `onColor` keeps the light wordmark for a coloured background in either theme.
 */
export function NerveLockup({ className, onColor = false }: TProps & { onColor?: boolean }) {
  if (onColor) return <img src={lockupOnDark} alt="Nerve" className={className} />;
  return (
    <>
      <img src={lockup} alt="Nerve" className={cn(className, "dark:hidden")} />
      <img src={lockupOnDark} alt="Nerve" className={cn(className, "hidden dark:block")} />
    </>
  );
}
