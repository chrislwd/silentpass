'use client';

import { useEffect, useState } from 'react';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || 'sk_test_sandbox_key_001';

interface BillingEntry {
  product: string;
  country: string;
  calls: number;
  successful: number;
  unit_price: number;
  total: number;
}

interface BillingData {
  entries: BillingEntry[];
  total_cost: number;
  period: string;
  currency: string;
}

export default function BillingPage() {
  const [data, setData] = useState<BillingData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch(`${API_BASE}/v1/billing/summary`, {
      headers: { 'X-API-Key': API_KEY },
    })
      .then(r => r.ok ? r.json() : null)
      .then(d => { if (d) setData(d); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  const entries = data?.entries || [];
  const totalCost = data?.total_cost || 0;
  const period = data?.period || '2026-03';

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Billing</h1>
          <p className="text-gray-500 mt-1">Usage and cost breakdown for {period}</p>
        </div>
        <div className="text-right">
          <div className="text-sm text-gray-500">Current Period Total</div>
          <div className="text-3xl font-bold text-gray-900">
            ${totalCost.toLocaleString('en-US', { minimumFractionDigits: 2 })}
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-6 text-gray-400">Loading...</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-500 border-b border-gray-200 bg-gray-50">
                <th className="px-6 py-3 font-medium">Product</th>
                <th className="px-6 py-3 font-medium">Country</th>
                <th className="px-6 py-3 font-medium text-right">API Calls</th>
                <th className="px-6 py-3 font-medium text-right">Successful</th>
                <th className="px-6 py-3 font-medium text-right">Unit Price</th>
                <th className="px-6 py-3 font-medium text-right">Total</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((row, i) => (
                <tr key={i} className="border-b border-gray-50 hover:bg-gray-50 text-sm">
                  <td className="px-6 py-3 font-medium">{row.product}</td>
                  <td className="px-6 py-3 font-mono">{row.country}</td>
                  <td className="px-6 py-3 text-right font-mono">{row.calls.toLocaleString()}</td>
                  <td className="px-6 py-3 text-right font-mono">{row.successful.toLocaleString()}</td>
                  <td className="px-6 py-3 text-right font-mono text-gray-500">${row.unit_price.toFixed(3)}</td>
                  <td className="px-6 py-3 text-right font-mono font-medium">${row.total.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="bg-gray-50 font-semibold text-sm">
                <td className="px-6 py-3" colSpan={5}>Total</td>
                <td className="px-6 py-3 text-right font-mono">
                  ${totalCost.toLocaleString('en-US', { minimumFractionDigits: 2 })}
                </td>
              </tr>
            </tfoot>
          </table>
        )}
      </div>
    </div>
  );
}
