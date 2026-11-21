import type { MouseEvent, ReactNode } from "react";
import { navigate, usePath } from "./router";

const icon = (d: ReactNode) => (
  <svg
    viewBox="0 0 24 24"
    aria-hidden="true"
    className="size-6 fill-none stroke-current stroke-2 [stroke-linecap:round] [stroke-linejoin:round]"
  >
    {d}
  </svg>
);

const PLACES = [
  {
    path: "/",
    label: "Home",
    icon: icon(
      <>
        <path d="M3 11 12 4l9 7v9H3z" />
        <path d="M9 20v-6h6v6" />
      </>,
    ),
  },
  {
    path: "/collection",
    label: "Collection",
    icon: icon(
      <>
        <rect x="4" y="4" width="7" height="7" rx="2" />
        <rect x="13" y="4" width="7" height="7" rx="2" />
        <rect x="4" y="13" width="7" height="7" rx="2" />
        <rect x="13" y="13" width="7" height="7" rx="2" />
      </>,
    ),
  },
  {
    path: "/make-room",
    label: "Make room",
    icon: icon(
      <path d="M12 3l2.5 6.5L21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5z" />,
    ),
  },
  {
    path: "/rules",
    label: "Rules",
    icon: icon(<path d="M5 6h14M5 12h10M5 18h6" />),
  },
  {
    path: "/more",
    label: "More",
    icon: icon(
      <>
        <circle cx="12" cy="8" r="4" />
        <path d="M4 21c1.5-4 4.5-6 8-6s6.5 2 8 6" />
      </>,
    ),
  },
] as const;

/**
 * The main menu: a floating bar at the bottom on phones, a row at the top
 * on wider screens. Make room is raised in the middle on phones.
 */
export function Nav() {
  const path = usePath();
  const go = (e: MouseEvent<HTMLAnchorElement>, to: string) => {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    navigate(to);
  };
  return (
    <nav
      aria-label="Main"
      className="fixed inset-x-3 bottom-3 z-20 rounded-full bg-white/95 px-2 py-1.5 shadow-lg ring-1 ring-slate-200 backdrop-blur sm:static sm:inset-auto sm:rounded-2xl sm:px-1.5 sm:py-1.5 sm:shadow-sm dark:bg-slate-900/95 dark:ring-slate-700"
    >
      <ul className="flex items-end justify-around sm:items-center sm:justify-start sm:gap-1">
        {PLACES.map((p) => {
          const current = path === p.path;
          const raised = p.path === "/make-room";
          return (
            <li key={p.path}>
              <a
                href={p.path}
                aria-current={current ? "page" : undefined}
                onClick={(e) => go(e, p.path)}
                className={`flex min-w-14 flex-col items-center gap-0.5 whitespace-nowrap rounded-full px-2 py-1 text-[11px] font-semibold sm:flex-row sm:gap-2 sm:px-3 sm:py-2 sm:text-sm ${
                  current
                    ? "text-teal-800 dark:text-teal-300"
                    : "text-slate-500 hover:text-teal-800 dark:text-slate-400 dark:hover:text-teal-300"
                } ${current ? "sm:bg-teal-50 dark:sm:bg-slate-800" : ""}`}
              >
                <span
                  className={`grid place-items-center rounded-full ${
                    raised
                      ? "-mt-6 size-13 bg-teal-700 text-white shadow-md sm:mt-0 sm:size-auto sm:bg-transparent sm:text-current sm:shadow-none"
                      : current
                        ? "size-9 bg-teal-50 sm:size-auto sm:bg-transparent dark:bg-slate-800"
                        : "size-9 sm:size-auto"
                  }`}
                >
                  {p.icon}
                </span>
                {p.label}
              </a>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
