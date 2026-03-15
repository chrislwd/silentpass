'use client';

interface CountryCoverage {
  country: string;
  code: string;
  operators: string[];
  silentVerify: boolean;
  simSwap: boolean;
  avgLatency: string;
  successRate: string;
  status: 'live' | 'beta' | 'planned';
}

const coverageData: CountryCoverage[] = [
  { country: 'Indonesia', code: 'ID', operators: ['Telkomsel', 'XL Axiata', 'Indosat'], silentVerify: true, simSwap: true, avgLatency: '1.1s', successRate: '87%', status: 'live' },
  { country: 'Thailand', code: 'TH', operators: ['AIS', 'DTAC', 'TrueMove H'], silentVerify: true, simSwap: true, avgLatency: '0.9s', successRate: '89%', status: 'live' },
  { country: 'Philippines', code: 'PH', operators: ['Globe', 'Smart'], silentVerify: true, simSwap: false, avgLatency: '1.3s', successRate: '82%', status: 'beta' },
  { country: 'Malaysia', code: 'MY', operators: ['Maxis', 'Celcom', 'Digi'], silentVerify: true, simSwap: true, avgLatency: '1.0s', successRate: '85%', status: 'live' },
  { country: 'Singapore', code: 'SG', operators: ['Singtel', 'StarHub', 'M1'], silentVerify: true, simSwap: true, avgLatency: '0.7s', successRate: '92%', status: 'live' },
  { country: 'Vietnam', code: 'VN', operators: ['Viettel', 'Mobifone'], silentVerify: true, simSwap: false, avgLatency: '1.4s', successRate: '78%', status: 'beta' },
  { country: 'Brazil', code: 'BR', operators: ['Claro', 'Vivo', 'TIM'], silentVerify: true, simSwap: true, avgLatency: '1.6s', successRate: '76%', status: 'beta' },
  { country: 'Mexico', code: 'MX', operators: ['Telcel', 'AT&T MX'], silentVerify: false, simSwap: false, avgLatency: '-', successRate: '-', status: 'planned' },
];

export default function CoveragePage() {
  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Coverage</h1>
        <p className="text-gray-500 mt-1">Country and operator capability matrix</p>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="text-left text-sm text-gray-500 border-b border-gray-200 bg-gray-50">
              <th className="px-6 py-3 font-medium">Country</th>
              <th className="px-6 py-3 font-medium">Operators</th>
              <th className="px-6 py-3 font-medium text-center">Silent Verify</th>
              <th className="px-6 py-3 font-medium text-center">SIM Swap</th>
              <th className="px-6 py-3 font-medium">Avg Latency</th>
              <th className="px-6 py-3 font-medium">Success Rate</th>
              <th className="px-6 py-3 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            {coverageData.map((row) => (
              <tr key={row.code} className="border-b border-gray-50 hover:bg-gray-50">
                <td className="px-6 py-4">
                  <div className="font-medium">{row.country}</div>
                  <div className="text-xs text-gray-400 font-mono">{row.code}</div>
                </td>
                <td className="px-6 py-4 text-sm text-gray-600">
                  {row.operators.join(', ')}
                </td>
                <td className="px-6 py-4 text-center">
                  {row.silentVerify ? (
                    <span className="text-green-500 text-lg">&#10003;</span>
                  ) : (
                    <span className="text-gray-300 text-lg">&#10007;</span>
                  )}
                </td>
                <td className="px-6 py-4 text-center">
                  {row.simSwap ? (
                    <span className="text-green-500 text-lg">&#10003;</span>
                  ) : (
                    <span className="text-gray-300 text-lg">&#10007;</span>
                  )}
                </td>
                <td className="px-6 py-4 text-sm font-mono">{row.avgLatency}</td>
                <td className="px-6 py-4 text-sm font-mono">{row.successRate}</td>
                <td className="px-6 py-4">
                  <span
                    className={`px-2 py-0.5 rounded text-xs font-medium ${
                      row.status === 'live'
                        ? 'bg-green-100 text-green-700'
                        : row.status === 'beta'
                        ? 'bg-yellow-100 text-yellow-700'
                        : 'bg-gray-100 text-gray-500'
                    }`}
                  >
                    {row.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
