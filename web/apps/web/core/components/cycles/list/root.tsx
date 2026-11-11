/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { Disclosure } from "@headlessui/react";
// nerve imports
import { ContentWrapper, ERowVariant } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { ListLayout } from "@/components/core/list";
// local imports
import { CyclePeekOverview } from "../cycle-peek-overview";
import { CycleListGroupHeader } from "./cycle-list-group-header";
import { CyclesListMap } from "./cycles-list-map";

type Props = {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
};

/** A project's cycles as rows (its list's sections, or its archive), and the preview of the cycle ?peekCycle names. */
export function CyclesList(props: Props) {
  const { workspaceSlug, projectId, children } = props;
  return (
    <ContentWrapper variant={ERowVariant.HUGGING} className="flex-row">
      <ListLayout>{children}</ListLayout>
      <CyclePeekOverview projectId={projectId} workspaceSlug={workspaceSlug} />
    </ContentWrapper>
  );
}

type SectionProps = {
  workspaceSlug: string;
  projectId: string;
  /** The status the section's icon shows. */
  status: "upcoming" | "completed";
  title: string;
  cycleIds: string[];
  defaultOpen?: boolean;
};

/** A section of a project's cycles list, named by its title: its header, which opens and closes it, and its rows. */
export function CycleListSection(props: SectionProps) {
  const { workspaceSlug, projectId, status, title, cycleIds, defaultOpen = false } = props;
  return (
    <Disclosure
      as="section"
      aria-label={title}
      className={cn("flex flex-shrink-0 flex-col", { "pb-7": status === "completed" })}
      defaultOpen={defaultOpen}
    >
      {({ open }) => (
        <>
          <Disclosure.Button className="sticky top-0 z-[2] w-full flex-shrink-0 cursor-pointer border-b border-subtle bg-layer-1">
            <CycleListGroupHeader title={title} type={status} count={cycleIds.length} showCount isExpanded={open} />
          </Disclosure.Button>
          <Disclosure.Panel>
            <CyclesListMap cycleIds={cycleIds} projectId={projectId} workspaceSlug={workspaceSlug} />
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  );
}
