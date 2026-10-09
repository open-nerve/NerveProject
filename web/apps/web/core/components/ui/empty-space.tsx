/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentType, ReactNode, SVGProps } from "react";
import { Link } from "react-router";
import { ChevronRightOutline } from "@makeplane/propel/icons";

type EmptySpaceProps = {
  title: string;
  description: string;
  children: ReactNode;
};

function EmptySpace({ title, description, children }: EmptySpaceProps) {
  return (
    <div className="max-w-lg">
      <h2 className="text-16 font-medium text-primary">{title}</h2>
      <div className="mt-1 text-13 text-secondary">{description}</div>
      <ul role="list" className="mt-6 divide-y divide-subtle-1 border-t border-b border-subtle">
        {children}
      </ul>
    </div>
  );
}

/** An item of an EmptySpace's list: a button that does action, or a link to href. */
type EmptySpaceItemProps = { title: string; Icon: ComponentType<SVGProps<SVGSVGElement>> } & (
  | { action: () => void; href?: undefined }
  | { href: string; action?: undefined }
);

function EmptySpaceItem({ title, Icon, action, href }: EmptySpaceItemProps) {
  const spaceItem = (
    <div className="group relative flex items-center space-x-3 py-4">
      <div className="flex-shrink-0">
        <span className="inline-flex h-10 w-10 items-center justify-center rounded-lg bg-accent-primary">
          <Icon className="h-6 w-6 text-on-color" aria-hidden="true" />
        </span>
      </div>
      <div className="min-w-0 flex-1 text-secondary">
        <div className="text-13 font-medium group-hover:text-primary">{title}</div>
      </div>
      <div className="flex-shrink-0 self-center">
        <ChevronRightOutline className="h-5 w-5 text-secondary group-hover:text-primary" aria-hidden="true" />
      </div>
    </div>
  );

  return (
    <li>
      {href === undefined ? (
        <button type="button" className="w-full cursor-pointer text-left" onClick={action}>
          {spaceItem}
        </button>
      ) : (
        <Link to={href} className="block">
          {spaceItem}
        </Link>
      )}
    </li>
  );
}

export { EmptySpace, EmptySpaceItem };
