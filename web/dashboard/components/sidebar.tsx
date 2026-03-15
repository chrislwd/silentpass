'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const navItems = [
  { href: '/', label: 'Dashboard', icon: '◉' },
  { href: '/policies', label: 'Policies', icon: '⚙' },
  { href: '/coverage', label: 'Coverage', icon: '◎' },
  { href: '/logs', label: 'Logs', icon: '☰' },
  { href: '/billing', label: 'Billing', icon: '$' },
  { href: '/playground', label: 'Playground', icon: '>' },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-gray-900 text-white flex flex-col">
      <div className="p-6 border-b border-gray-700">
        <h1 className="text-xl font-bold">SilentPass</h1>
        <p className="text-xs text-gray-400 mt-1">Console</p>
      </div>
      <nav className="flex-1 p-4 space-y-1">
        {navItems.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                isActive
                  ? 'bg-primary-600 text-white'
                  : 'text-gray-300 hover:bg-gray-800 hover:text-white'
              }`}
            >
              <span className="text-lg">{item.icon}</span>
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="p-4 border-t border-gray-700">
        <div className="text-xs text-gray-500">Sandbox Environment</div>
        <div className="text-xs text-gray-400 mt-1 font-mono truncate">
          sk_test_sandbox_key_001
        </div>
      </div>
    </aside>
  );
}
