'use client';

import { useState } from 'react';
import { api, type SessionResponse, type SilentVerifyResponse, type OTPSendResponse, type OTPCheckResponse, type SIMSwapResponse, type VerdictResponse } from '@/lib/api';

type StepResult = SessionResponse | SilentVerifyResponse | OTPSendResponse | OTPCheckResponse | SIMSwapResponse | VerdictResponse | { error: string };

interface Step {
  name: string;
  status: 'pending' | 'running' | 'success' | 'error';
  result?: StepResult;
}

export default function PlaygroundPage() {
  const [phone, setPhone] = useState('+6281234567890');
  const [country, setCountry] = useState('ID');
  const [useCase, setUseCase] = useState<'signup' | 'login' | 'transaction' | 'phone_change'>('signup');
  const [otpCode, setOtpCode] = useState('000000');
  const [sessionId, setSessionId] = useState('');
  const [steps, setSteps] = useState<Step[]>([]);
  const [running, setRunning] = useState(false);

  const addStep = (name: string): number => {
    const idx = steps.length;
    setSteps(prev => [...prev, { name, status: 'running' }]);
    return idx;
  };

  const updateStep = (idx: number, status: 'success' | 'error', result: StepResult) => {
    setSteps(prev => prev.map((s, i) => i === idx ? { ...s, status, result } : s));
  };

  const runFullFlow = async () => {
    setSteps([]);
    setRunning(true);
    let currentSessionId = '';

    try {
      // Step 1: Create session
      const s1Idx = 0;
      setSteps([{ name: 'Create Session', status: 'running' }]);
      const session = await api.createSession({
        app_id: 'playground',
        phone_number: phone,
        country_code: country,
        verification_type: 'silent_or_otp',
        use_case: useCase,
      });
      currentSessionId = session.session_id;
      setSessionId(currentSessionId);
      setSteps(prev => prev.map((s, i) => i === s1Idx ? { ...s, status: 'success', result: session } : s));

      // Step 2: Silent verify
      setSteps(prev => [...prev, { name: 'Silent Verify', status: 'running' }]);
      const silent = await api.silentVerify(currentSessionId);
      setSteps(prev => prev.map((s, i) => i === 1 ? { ...s, status: 'success', result: silent } : s));

      // Step 3: If fallback needed, send OTP
      if (silent.status === 'fallback_required') {
        setSteps(prev => [...prev, { name: 'Send OTP (fallback)', status: 'running' }]);
        const otpSend = await api.sendOTP(currentSessionId, 'sms');
        setSteps(prev => prev.map((s, i) => i === 2 ? { ...s, status: 'success', result: otpSend } : s));
      }

      // Step 4: SIM Swap check
      const simIdx = steps.length;
      setSteps(prev => [...prev, { name: 'SIM Swap Check', status: 'running' }]);
      const simSwap = await api.checkSIMSwap(phone, country);
      setSteps(prev => prev.map((s, i) => i === prev.length - 1 ? { ...s, status: 'success', result: simSwap } : s));

      // Step 5: Verdict
      setSteps(prev => [...prev, { name: 'Risk Verdict', status: 'running' }]);
      const verdict = await api.getVerdict(currentSessionId, {
        verification_result: silent.status,
        sim_swap_result: simSwap,
      });
      setSteps(prev => prev.map((s, i) => i === prev.length - 1 ? { ...s, status: 'success', result: verdict } : s));

    } catch (err) {
      setSteps(prev => {
        const last = prev.length - 1;
        return prev.map((s, i) => i === last ? { ...s, status: 'error', result: { error: String(err) } } : s);
      });
    }

    setRunning(false);
  };

  const runOTPCheck = async () => {
    if (!sessionId) return;
    setSteps(prev => [...prev, { name: 'Check OTP', status: 'running' }]);
    try {
      const result = await api.checkOTP(sessionId, otpCode);
      setSteps(prev => prev.map((s, i) => i === prev.length - 1 ? { ...s, status: 'success', result } : s));
    } catch (err) {
      setSteps(prev => prev.map((s, i) => i === prev.length - 1 ? { ...s, status: 'error', result: { error: String(err) } } : s));
    }
  };

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">API Playground</h1>
        <p className="text-gray-500 mt-1">Test the full verification flow against the live backend</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Input panel */}
        <div className="space-y-6">
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="font-semibold mb-4">Parameters</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-500 mb-1">Phone Number</label>
                <input
                  type="text" value={phone} onChange={e => setPhone(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
              <div>
                <label className="block text-sm text-gray-500 mb-1">Country Code</label>
                <select
                  value={country} onChange={e => setCountry(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  {['ID', 'TH', 'PH', 'MY', 'SG', 'VN', 'BR', 'MX'].map(c => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-500 mb-1">Use Case</label>
                <select
                  value={useCase} onChange={e => setUseCase(e.target.value as typeof useCase)}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  <option value="signup">Signup</option>
                  <option value="login">Login</option>
                  <option value="transaction">Transaction</option>
                  <option value="phone_change">Phone Change</option>
                </select>
              </div>
              <button
                onClick={runFullFlow} disabled={running}
                className="w-full bg-primary-600 text-white py-2 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 transition-colors"
              >
                {running ? 'Running...' : 'Run Full Verification Flow'}
              </button>
            </div>
          </div>

          {sessionId && (
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <h2 className="font-semibold mb-4">OTP Verification</h2>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm text-gray-500 mb-1">Session ID</label>
                  <div className="text-xs font-mono text-gray-400 truncate">{sessionId}</div>
                </div>
                <div>
                  <label className="block text-sm text-gray-500 mb-1">OTP Code</label>
                  <input
                    type="text" value={otpCode} onChange={e => setOtpCode(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary-500"
                    placeholder="Enter OTP code (sandbox: 000000)"
                  />
                </div>
                <button
                  onClick={runOTPCheck}
                  className="w-full bg-gray-800 text-white py-2 rounded-lg text-sm font-medium hover:bg-gray-900 transition-colors"
                >
                  Verify OTP
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Results panel */}
        <div className="space-y-3">
          <h2 className="font-semibold text-gray-900">Flow Steps</h2>
          {steps.length === 0 && (
            <div className="text-sm text-gray-400">Run a flow to see results here</div>
          )}
          {steps.map((step, i) => (
            <div key={i} className="bg-white rounded-xl border border-gray-200 p-4">
              <div className="flex items-center gap-2 mb-2">
                <span className={`w-2 h-2 rounded-full ${
                  step.status === 'success' ? 'bg-green-500' :
                  step.status === 'error' ? 'bg-red-500' :
                  step.status === 'running' ? 'bg-yellow-500 animate-pulse' :
                  'bg-gray-300'
                }`} />
                <span className="text-sm font-medium">{step.name}</span>
                <span className={`text-xs px-1.5 py-0.5 rounded ${
                  step.status === 'success' ? 'bg-green-100 text-green-700' :
                  step.status === 'error' ? 'bg-red-100 text-red-700' :
                  step.status === 'running' ? 'bg-yellow-100 text-yellow-700' :
                  'bg-gray-100 text-gray-500'
                }`}>
                  {step.status}
                </span>
              </div>
              {step.result && (
                <pre className="text-xs font-mono bg-gray-50 rounded-lg p-3 overflow-x-auto max-h-48">
                  {JSON.stringify(step.result, null, 2)}
                </pre>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
