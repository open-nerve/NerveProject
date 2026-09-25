import type { TIssue } from "@nerve/types";
import type { CustomMenu } from "@nerve/ui";

type TQuickActionPlacement = React.ComponentProps<typeof CustomMenu>["placement"];

export interface IQuickActionProps {
  parentRef: React.RefObject<HTMLElement | null>;
  issue: TIssue;
  handleDelete: () => Promise<void>;
  handleUpdate?: (data: TIssue) => Promise<void>;
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
  issue: TIssue;
  parentRef: React.RefObject<HTMLElement | null>;
  customActionButton?: React.ReactElement;
  placement?: TQuickActionPlacement;
  portalElement?: HTMLDivElement | null;
}) => React.ReactNode;
