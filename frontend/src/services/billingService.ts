import { http } from './httpClient'
import type { BalanceResponse, TopUpResponse, Transaction, TopUpMethod } from '@/types/api'

// Confirmed real endpoints, all in backend/internal/api/billing_handlers.go.
export async function getBalance(): Promise<BalanceResponse> {
  return http.get<BalanceResponse>('/billing/balance', true)
}

export async function topUp(amountPaise: number, method: TopUpMethod): Promise<TopUpResponse> {
  return http.post<TopUpResponse>('/billing/topup', { amount_paise: amountPaise, method }, true)
}

export async function getTransactions(): Promise<Transaction[]> {
  return http.get<Transaction[]>('/billing/transactions', true)
}
