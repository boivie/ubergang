import { Icon } from "@tabler/icons-react";
import { Link } from "react-router";

export interface Tab {
  text: string;
  tabid: string;
  url: string;
  icon: Icon;
}

export interface TabListProps {
  tabs: Tab[];
  active: string;
}

interface TabProps {
  tab: Tab;
  active: boolean;
}

export function Tab({ tab, active }: TabProps) {
  if (active) {
    return (
      <li role="presentation">
        <Link
          className="inline-flex items-center justify-center w-full h-10 gap-2 px-5 -mb-px text-sm font-medium tracking-wide transition duration-300 border-b-2 rounded-t focus-visible:outline-hidden whitespace-nowrap border-emerald-500 hover:border-emerald-600 focus:border-emerald-700 text-emerald-500 hover:text-emerald-600 focus:text-emerald-700 hover:bg-emerald-50 focus:bg-emerald-50 disabled:cursor-not-allowed disabled:border-slate-500 stroke-emerald-500 hover:stroke-emerald-600 focus:stroke-emerald-700"
          role="tab"
          to={tab.url}
          aria-selected="true"
        >
          <tab.icon size={20} />
          <span className="order-2 hidden md:block">{tab.text}</span>
        </Link>
      </li>
    );
  }

  return (
    <li role="presentation">
      <Link
        className="inline-flex items-center justify-center w-full h-10 gap-2 px-5 -mb-px text-sm font-medium tracking-wide transition duration-300 border-b-2 border-transparent rounded-t focus-visible:outline-hidden justify-self-center hover:border-emerald-500 focus:border-emerald-600 whitespace-nowrap text-slate-700 stroke-slate-700 hover:bg-emerald-50 hover:text-emerald-500 focus:stroke-emerald-600 focus:bg-emerald-50 focus:text-emerald-600 hover:stroke-emerald-600 disabled:cursor-not-allowed disabled:text-slate-500"
        role="tab"
        to={tab.url}
      >
        <tab.icon size={20} />
        <span className="order-2 hidden md:block">{tab.text}</span>
      </Link>
    </li>
  );
}

export function TabList({ tabs, active }: TabListProps) {
  return (
    <section className="max-w-full mb-3" aria-multiselectable="false">
      <ul
        className="flex items-center overflow-x-auto overflow-y-hidden border-b border-slate-200"
        role="tablist"
      >
        {tabs.map((t) => (
          <Tab key={t.tabid} tab={t} active={t.tabid == active} />
        ))}
      </ul>
    </section>
  );
}
