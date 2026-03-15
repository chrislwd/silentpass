const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || 'sk_test_sandbox_key_001';

interface RequestOptions {
  method?: string;
  body?: unknown;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const res = await fetch(`${API_BASE}/v1${path}`, {
    method: opts.method || 'GET',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
    },
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || `API error: ${res.status}`);
  }

  return res.json();
}

// Verification
export interface CreateSessionParams {
  app_id: string;
  phone_number: string;
  country_code: string;
  verification_type: 'silent' | 'silent_or_otp' | 'otp_only';
  use_case: 'signup' | 'login' | 'transaction' | 'phone_change';
  device_context?: { ip_address?: string; user_agent?: string };
  callback_url?: string;
}

export interface SessionResponse {
  session_id: string;
  recommended_action: string;
  expires_at: string;
}

export interface SilentVerifyResponse {
  status: 'verified' | 'fallback_required' | 'failed';
  confidence_score?: number;
  telco_signal?: string;
  token?: string;
}

export interface OTPSendResponse {
  delivery_status: string;
  resend_after_seconds: number;
}

export interface OTPCheckResponse {
  status: 'verified' | 'failed';
  token?: string;
  attempts_left: number;
}

export interface SIMSwapResponse {
  sim_swap_detected: boolean;
  last_change_time?: string;
  risk_level: 'low' | 'medium' | 'high';
  recommendation: 'allow' | 'challenge' | 'block' | 'review';
}

export interface VerdictResponse {
  verdict: 'allow' | 'challenge' | 'block' | 'review';
  risk_level: 'low' | 'medium' | 'high';
  reasons?: string[];
  action_required?: string;
}

export const api = {
  createSession: (params: CreateSessionParams) =>
    request<SessionResponse>('/verification/session', { method: 'POST', body: params }),

  silentVerify: (sessionId: string) =>
    request<SilentVerifyResponse>('/verification/silent', { method: 'POST', body: { session_id: sessionId } }),

  sendOTP: (sessionId: string, channel: string, locale?: string) =>
    request<OTPSendResponse>('/verification/otp/send', { method: 'POST', body: { session_id: sessionId, channel, locale } }),

  checkOTP: (sessionId: string, code: string) =>
    request<OTPCheckResponse>('/verification/otp/check', { method: 'POST', body: { session_id: sessionId, code } }),

  checkSIMSwap: (phoneNumber: string, countryCode: string) =>
    request<SIMSwapResponse>('/risk/sim-swap', { method: 'POST', body: { phone_number: phoneNumber, country_code: countryCode } }),

  getVerdict: (sessionId: string, params?: { verification_result?: string; sim_swap_result?: SIMSwapResponse; policy_id?: string }) =>
    request<VerdictResponse>('/risk/verdict', { method: 'POST', body: { session_id: sessionId, ...params } }),

  health: () => request<{ status: string; service: string; storage: string }>('/../health'),
};
