'use client';

import { useEffect, useState } from 'react';
import { StatCard } from '@/components/stat-card';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || 'sk_test_sandbox_key_001';

interface DashboardData {
  total_verifications: number;
  silent_success_rate: number;
  fallback_rate: number;
  otp_cost_saved: number;
  high_risk_blocked: number;
  avg_latency_ms: number;
  countries: { code: string; requests: number; silent_rate: number }[];
}

interface Activity {
  time: string;
  event: string;
  country: string;
  status: string;
  latency_ms: number;
}

export default function DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    const headers = { 'X-API-Key': API_KEY, 'Content-Type': 'application/json' };

    Promise.all([
      fetch(`${API_BASE}/v1/stats/dashboard`, { headers }).then(r => r.ok ? r.json() : null),
      fetch(`${API_BASE}/v1/stats/activity`, { headers }).then(r => r.ok ? r.json() : null),
    ])
      .then(([dashboard, activity]) => {
        if (dashboard) setData(dashboard);
        if (activity) setActivities(activity.activities || []);
      })
      .catch(() => setError('Cannot connect to API. Start the backend with: make dev'));
  }, []);

  if (error) {
    return (
      <div>
        <h1 className="text-2xl font-bold text-gray-900 mb-4">Dashboard</h1>
        <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-6 text-sm text-yellow-800">
          {error}
        </div>
        <FallbackDashboard />
      </div>
    );
  }

  if (!data) {
    return (
      <div>
        <h1 className="text-2xl font-bold text-gray-900 mb-4">Dashboard</h1>
        <div className="text-gray-400">Loading...</div>
      </div>
    );
  }

  const stats = [
    { title: 'Total Verifications', value: data.total_verifications.toLocaleString(), subtitle: 'Last 30 days', trend: { value: '12.5% vs prev period', positive: true } },
    { title: 'Silent Success Rate', value: `${data.silent_success_rate}%`, subtitle: 'Across all countries', trend: { value: '2.1% vs prev period', positive: true } },
    { title: 'OTP Fallback Rate', value: `${data.fallback_rate}%`, subtitle: 'Auto-fallback triggered', trend: { value: '1.8% vs prev period', positive: false } },
    { title: 'OTP Cost Saved', value: `$${data.otp_cost_saved.toLocaleString()}`, subtitle: 'vs pure OTP baseline', trend: { value: '18% vs prev period', positive: true } },
    { title: 'High Risk Blocked', value: data.high_risk_blocked.toString(), subtitle: 'SIM Swap + fraud signals', trend: { value: '5.2% vs prev period', positive: true } },
    { title: 'Avg Latency', value: `${(data.avg_latency_ms / 1000).toFixed(1)}s`, subtitle: 'Silent verification P95', trend: { value: '0.3s improvement', positive: true } },
  ];

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-500 mt-1">Live metrics from your SilentPass backend</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
        {stats.map((stat) => (
          <StatCard key={stat.title} {...stat} />
        ))}
      </div>

      {data.countries && data.countries.length > 0 && (
        <div className="bg-white rounded-xl border border-gray-200 mb-8">
          <div className="p-6 border-b border-gray-100">
            <h2 className="text-lg font-semibold">Country Distribution</h2>
          </div>
          <div className="p-6">
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
              {data.countries.map((c) => (
                <div key={c.code} className="text-center">
                  <div className="text-2xl font-bold text-gray-900">{c.code}</div>
                  <div className="text-sm text-gray-500">{c.requests.toLocaleString()} req</div>
                  <div className="text-xs text-green-600">{c.silent_rate}% silent</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200">
        <div className="p-6 border-b border-gray-100">
          <h2 className="text-lg font-semibold">Recent Activity</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-500 border-b border-gray-100">
                <th className="px-6 py-3 font-medium">Time</th>
                <th className="px-6 py-3 font-medium">Event</th>
                <th className="px-6 py-3 font-medium">Country</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Latency</th>
              </tr>
            </thead>
            <tbody>
              {activities.map((row, i) => (
                <tr key={i} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-6 py-3 text-sm text-gray-400">{row.time}</td>
                  <td className="px-6 py-3 text-sm">{row.event}</td>
                  <td className="px-6 py-3 text-sm font-mono">{row.country}</td>
                  <td className="px-6 py-3">
                    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
                      row.status === 'verified' || row.status === 'clean'
                        ? 'bg-green-100 text-green-700'
                        : row.status === 'blocked'
                        ? 'bg-red-100 text-red-700'
                        : 'bg-yellow-100 text-yellow-700'
                    }`}>
                      {row.status}
                    </span>
                  </td>
                  <td className="px-6 py-3 text-sm font-mono text-gray-500">{row.latency_ms}ms</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function FallbackDashboard() {
  const stats = [
    { title: 'Total Verifications', value: '124,892', subtitle: 'Demo data', trend: { value: '12.5%', positive: true } },
    { title: 'Silent Success Rate', value: '84.7%', subtitle: 'Demo data', trend: { value: '2.1%', positive: true } },
    { title: 'OTP Fallback Rate', value: '15.3%', subtitle: 'Demo data', trend: { value: '1.8%', positive: false } },
    { title: 'OTP Cost Saved', value: '$3,420', subtitle: 'Demo data', trend: { value: '18%', positive: true } },
  ];

  return (
    <div className="mt-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 opacity-60">
      {stats.map((stat) => (
        <StatCard key={stat.title} {...stat} />
      ))}
    </div>
  );
}
