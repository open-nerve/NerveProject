/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { cn } from "@nerve/utils";

type Props = {
  children: ReactNode;
  className?: string;
};

function DefaultLayout({ children, className }: Props) {
  return <div className={cn("h-screen w-full overflow-hidden bg-surface-1", className)}>{children}</div>;
}

export default DefaultLayout;
