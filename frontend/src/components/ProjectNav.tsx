"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { useLocale } from "./LocaleProvider";

const tabs = [
  { key: "dashboard", labelKey: "nav.dashboard", path: "" },
  { key: "income", labelKey: "nav.income_daily", path: "/income-daily" },
  { key: "actual", labelKey: "nav.actual_expenses", path: "/actual-expenses" },
  { key: "members", labelKey: "nav.members", path: "/members" },
  { key: "taxes", labelKey: "nav.taxes", path: "/tax-rules" },
  { key: "rules", labelKey: "nav.rules", path: "/profit-share-rules" },
  { key: "my-income", labelKey: "nav.my_income", path: "/my-income" },
  { key: "forecast", labelKey: "nav.forecast", path: "/forecast" }
];

export function ProjectNav({ projectId }: { projectId: string }) {
  const pathname = usePathname();
  const { t } = useLocale();

  return (
    <nav className="tabs project-tabs">
      {tabs.map((tab) => {
        const href = `/projects/${projectId}${tab.path}`;
        const active = pathname === href;
        return (
          <Link key={tab.key} href={href} className={active ? "active" : ""}>
            {t(tab.labelKey)}
          </Link>
        );
      })}
    </nav>
  );
}
