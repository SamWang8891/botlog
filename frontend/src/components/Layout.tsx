import { NavLink, Outlet } from 'react-router-dom';

function NavItem({ to, label }: { to: string; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `px-4 py-2 text-sm font-medium transition-all duration-200 border-b-2 ${
          isActive
            ? 'text-neon-blue border-neon-blue glow-blue'
            : 'text-gray-400 border-transparent hover:text-neon-green hover:border-neon-green/50'
        }`
      }
    >
      {label}
    </NavLink>
  );
}

export default function Layout() {
  return (
    <div className="min-h-screen bg-dark-900 scanlines">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-dark-900/90 backdrop-blur border-b border-dark-600">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 rounded-full bg-neon-green animate-pulse-glow" />
              <h1 className="text-lg sm:text-xl font-bold text-neon-green glow-green tracking-wider">
                BOTLOG
              </h1>
              <span className="hidden sm:inline text-xs text-dark-500 font-mono">
                // real-time bot traffic monitor
              </span>
            </div>

            {/* Navigation */}
            <nav className="flex gap-1">
              <NavItem to="/" label="LIVE" />
              <NavItem to="/stats" label="STATS" />
            </nav>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <Outlet />
      </main>
    </div>
  );
}
