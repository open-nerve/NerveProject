import type { Issue, IssueUpdate } from "@nerve/api-client";

import type { CustomMenu } from "@nerve/ui";

type TQuickActionPlacement = React.ComponentProps<typeof CustomMenu>["placement"];

export interface IQuickActionProps {
  parentRef: React.RefObject<HTMLElement | null>;
  issue: Issue;
  handleDelete: () => Promise<void>;
  /** Sends a change of the row's work item through the page's list. */
  handleUpdate?: (data: IssueUpdate) => Promise<void>;
  /**
   * Sends the removal of the row's work item from the page's cycle or module through the page's list, as its domain
   * sends it (a cycle's or a module's page's only, P6b spec 2.2, 2.5).
   */
  handleRemoveFromView?: () => Promise<void>;
  handleArchive?: () => Promise<void>;
  handleRestore?: () => Promise<void>;
  handleMoveToIssues?: () => Promise<void>;
  customActionButton?: React.ReactElement;
  portalElement?: HTMLDivElement | null;
  readOnly?: boolean;
  placements?: TQuickActionPlacement;
}

export type TRenderQuickActions = ({
  issue,
  parentRef,
  customActionButton,
  placement,
  portalElement,
}: {
  issue: Issue;
  parentRef: React.RefObject<HTMLElement | null>;
  customActionButton?: React.ReactElement;
  placement?: TQuickActionPlacement;
  portalElement?: HTMLDivElement | null;
}) => React.ReactNode;
