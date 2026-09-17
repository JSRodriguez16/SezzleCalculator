import { isOperation, type Calculation } from '@/lib/calculator'

export const HISTORY_KEY = 'sezzle-calculator-history-v1'
export const HISTORY_LIMIT = 20
export type HistoryEntry = Calculation & { id: string; createdAt: string }

function isHistoryEntry(value: unknown): value is HistoryEntry {
  if (!value || typeof value !== 'object') return false
  const entry = value as Record<string, unknown>
  return typeof entry.id === 'string' && typeof entry.createdAt === 'string' && Number.isFinite(Date.parse(entry.createdAt)) && isOperation(entry.operation) && typeof entry.a === 'number' && Number.isFinite(entry.a) && typeof entry.result === 'number' && Number.isFinite(entry.result) && (entry.operation === 'sqrt' ? entry.b === undefined : typeof entry.b === 'number' && Number.isFinite(entry.b))
}

export function readHistory(): HistoryEntry[] {
  try {
    const saved: unknown = JSON.parse(localStorage.getItem(HISTORY_KEY) || '[]')
    return Array.isArray(saved) ? saved.filter(isHistoryEntry).slice(0, HISTORY_LIMIT) : []
  } catch {
    return []
  }
}

export function saveHistory(history: HistoryEntry[]): void {
  // Navegación privada, cuotas o almacenamiento desactivado no bloquean los cálculos.
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(history.slice(0, HISTORY_LIMIT)))
  } catch { /* El historial continúa disponible durante esta sesión. */ }
}

export function appendHistory(history: HistoryEntry[], calculation: Calculation): HistoryEntry[] {
  // randomUUID requiere un contexto seguro; el identificador solo distingue filas locales.
  const id = typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  const entry = { ...calculation, id, createdAt: new Date().toISOString() }
  return [entry, ...history].slice(0, HISTORY_LIMIT)
}
