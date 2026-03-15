'use client';

import { useEffect, useState } from 'react';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || 'sk_test_sandbox_key_001';

interface LogEntry {
  id: string;
  session_id: string;
  timestamp: string;
  method: string;
  country_code: string;
  upstream_provider: string;
  result: string;
  latency_ms: number;
  error_code: string;
}

export default function LogsPage() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);

  const fetchLogs = (q?: string) => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);

    fetch(`${API_BASE}/v1/logs?${params}`, {
      headers: { 'X-API-Key': API_KEY },
    })
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        if (data?.logs) setLogs(data.logs);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => { fetchLogs(); }, []);

  const handleSearch = (val: string) => {
    setSearch(val);
    fetchLogs(val);
  };

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Logs</h1>
        <p className="text-gray-500 mt-1">Request trace and upstream response logs</p>
      </div>

      <div className="mb-4">
        <input
          type="text"
          placeholder="Search by session ID, method, country, or result..."
          value={search}
          onChange={(e) => handleSearch(e.target.value)}
          className="w-full md:w-96 px-4 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-6 text-gray-400">Loading...</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-500 border-b border-gray-200 bg-gray-50">
                <th className="px-4 py-3 font-medium">Session</th>
                <th className="px-4 py-3 font-medium">Timestamp</th>
                <th className="px-4 py-3 font-medium">Method</th>
                <th className="px-4 py-3 font-medium">Country</th>
                <th className="px-4 py-3 font-medium">Upstream</th>
                <th className="px-4 py-3 font-medium">Result</th>
                <th className="px-4 py-3 font-medium">Latency</th>
                <th className="px-4 py-3 font-medium">Error</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id} className="border-b border-gray-50 hover:bg-gray-50 text-sm">
                  <td className="px-4 py-3 font-mono text-primary-600">{log.session_id}</td>
                  <td className="px-4 py-3 text-gray-500">{log.timestamp}</td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 rounded bg-gray-100 text-xs font-medium">{log.method}</span>
                  </td>
                  <td className="px-4 py-3 font-mono">{log.country_code}</td>
                  <td className="px-4 py-3 text-gray-500">{log.upstream_provider}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      log.result === 'verified' || log.result === 'clean' || log.result === 'sent'
                        ? 'bg-green-100 text-green-700'
                        : log.result === 'block'
                        ? 'bg-red-100 text-red-700'
                        : 'bg-yellow-100 text-yellow-700'
                    }`}>
                      {log.result}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-mono text-gray-500">{log.latency_ms}ms</td>
                  <td className="px-4 py-3 font-mono text-red-400 text-xs">{log.error_code}</td>
                </tr>
              ))}
              {logs.length === 0 && (
                <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-400">No logs found</td></tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
