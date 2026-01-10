import { IconRss, IconStack2, IconUser, IconUsers } from "@tabler/icons-react";
import { Outlet, useMatches } from "react-router";
import { TabList } from "./TabList.tsx";

type HandleType = {
  tabid?: string;
};

export const Home = () => {
  const matches = useMatches();
  const tabid =
    matches
      .filter((match) => Boolean((match.handle as HandleType)?.tabid))
      .map((match) => (match.handle as HandleType).tabid)
      .find(() => true) || "profile";

  return (
    <div>
      <header className="border-b relative z-20 w-full border-b border-slate-200 bg-white/90 shadow-lg shadow-slate-700/5 after:absolute after:left-0 after:top-full after:z-10 after:block after:h-px after:w-full after:bg-slate-200 lg:border-slate-200 lg:backdrop-blur-xs lg:after:hidden">
        <div className="relative mx-auto max-w-full px-6 lg:max-w-5xl xl:max-w-7xl 2xl:max-w-384">
          <nav
            aria-label="main navigation"
            className="flex h-14 items-stretch justify-between font-medium text-slate-700"
            role="navigation"
          >
            <a
              className="flex items-center gap-1.5 whitespace-nowrap py-3 text-lg focus:outline-hidden lg:flex-1"
              href="/"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="icon icon-tabler icon-tabler-shield-code"
                width="32"
                height="32"
                viewBox="0 0 24 24"
                strokeWidth="1.5"
                stroke="#10b981"
                fill="none"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                <path d="M12 21a12 12 0 0 1 -8.5 -15a12 12 0 0 0 8.5 -3a12 12 0 0 0 8.5 3a12 12 0 0 1 -.078 7.024" />
                <path d="M20 21l2 -2l-2 -2" />
                <path d="M17 17l-2 2l2 2" />
              </svg>
              Übergang
            </a>
          </nav>
        </div>
      </header>
      <TabList
        tabs={[
          {
            text: "Profile",
            url: "/",
            tabid: "profile",
            icon: IconUser,
          },
          {
            text: "Web Proxy",
            url: "/backends",
            tabid: "backends",
            icon: IconStack2,
          },
          {
            text: "MQTT Proxy",
            url: "/mqtt",
            tabid: "mqtt",
            icon: IconRss,
          },
          {
            text: "Manage Users",
            url: "/users",
            tabid: "users",
            icon: IconUsers,
          },
        ]}
        active={tabid}
      />
      <div className="mx-2">
        <Outlet />
      </div>
    </div>
  );
};
