'use client';

import { useEffect, useState } from 'react';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || 'sk_test_sandbox_key_001';

interface Policy {
  id: string;
  name: string;
  use_case: string;
  strategy: string;
  sim_swap_action: string;
  countries: string[];
  priority: number;
  active: boolean;
}

export default function PoliciesPage() {
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchPolicies = () => {
    fetch(`${API_BASE}/v1/policies`, {
      headers: { 'X-API-Key': API_KEY },
    })
      .then(r => r.ok ? r.json() : null)
      .then(data => {
        if (data?.policies) setPolicies(data.policies);
        setLoading(false);
      })
      .catch(() => {
        // Fallback demo data
        setPolicies([
          { id: 'pol_001', name: 'Signup - Silent First', use_case: 'signup', strategy: 'silent_or_otp', sim_swap_action: 'challenge', countries: ['ID', 'TH', 'PH', 'MY'], priority: 10, active: true },
          { id: 'pol_002', name: 'Login - Low Friction', use_case: 'login', strategy: 'silent', sim_swap_action: 'challenge', countries: ['ID', 'TH', 'PH', 'MY', 'SG'], priority: 10, active: true },
          { id: 'pol_003', name: 'Transaction - Strict', use_case: 'transaction', strategy: 'silent_or_otp', sim_swap_action: 'block', countries: ['ID', 'TH'], priority: 10, active: true },
        ]);
        setLoading(false);
      });
  };

  useEffect(() => { fetchPolicies(); }, []);

  const togglePolicy = async (id: string, active: boolean) => {
    await fetch(`${API_BASE}/v1/policies/${id}`, {
      method: 'PUT',
      headers: { 'X-API-Key': API_KEY, 'Content-Type': 'application/json' },
      body: JSON.stringify({ active: !active }),
    });
    fetchPolicies();
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Policies</h1>
          <p className="text-gray-500 mt-1">Configure verification strategies per use case</p>
        </div>
        <button className="bg-primary-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-primary-700 transition-colors">
          Create Policy
        </button>
      </div>

      {loading ? (
        <div className="text-gray-400">Loading...</div>
      ) : (
        <div className="space-y-4">
          {policies.map((policy) => (
            <div key={policy.id} className="bg-white rounded-xl border border-gray-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <h3 className="font-semibold text-gray-900">{policy.name}</h3>
                  <button
                    onClick={() => togglePolicy(policy.id, policy.active)}
                    className={`px-2 py-0.5 rounded text-xs font-medium cursor-pointer transition-colors ${
                      policy.active ? 'bg-green-100 text-green-700 hover:bg-green-200' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
                    }`}
                  >
                    {policy.active ? 'Active' : 'Inactive'}
                  </button>
                </div>
                <span className="text-xs font-mono text-gray-400">{policy.id.substring(0, 8)}</span>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div>
                  <div className="text-gray-400 mb-1">Use Case</div>
                  <div className="font-medium">{policy.use_case}</div>
                </div>
                <div>
                  <div className="text-gray-400 mb-1">Strategy</div>
                  <div className="font-medium">{policy.strategy.replace(/_/g, ' ')}</div>
                </div>
                <div>
                  <div className="text-gray-400 mb-1">SIM Swap Action</div>
                  <div className="font-medium capitalize">{policy.sim_swap_action}</div>
                </div>
                <div>
                  <div className="text-gray-400 mb-1">Countries</div>
                  <div className="font-medium font-mono">{policy.countries?.join(', ') || '-'}</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
