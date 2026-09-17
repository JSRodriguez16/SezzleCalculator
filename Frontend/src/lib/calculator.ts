export const operations = [
  { id: 'add', label: 'Suma', symbol: '+', description: 'Adicionar dos números.', aLabel: 'Primer número', bLabel: 'Segundo número' },
  { id: 'subtract', label: 'Resta', symbol: '−', description: 'Sustraer un número de otro.', aLabel: 'Primer número', bLabel: 'Segundo número' },
  { id: 'multiply', label: 'Multiplicar', symbol: '×', description: 'Multiplicar dos numeros.', aLabel: 'Primer número', bLabel: 'Segundo número' },
  { id: 'divide', label: 'Dividir', symbol: '÷', description: 'Dividir un número entre otro. El divisor debe ser distinto de cero.', aLabel: 'Dividendo', bLabel: 'Divisor' },
  { id: 'power', label: 'Potencia', symbol: 'xⁿ', description: 'Potenciar un número por otro.', aLabel: 'Base', bLabel: 'Exponente' },
  { id: 'sqrt', label: 'Raíz', symbol: '√', description: 'Calcula la raíz cuadrada de un número mayor o igual a cero.', aLabel: 'Número', bLabel: '' },
  { id: 'percentage', label: 'Porcentaje', symbol: '%', description: 'Calcula el porcentaje de una cantidad: cantidad × porcentaje ÷ 100.', aLabel: 'Cantidad', bLabel: 'Porcentaje' },
] as const

export type Operation = (typeof operations)[number]['id']
export type Calculation = { operation: Operation; a: number; b?: number; result: number }
export type CalculationInput = Omit<Calculation, 'result'>

export function isOperation(value: unknown): value is Operation {
  return operations.some((operation) => operation.id === value)
}

/** Acepta coma o punto decimal, sin separadores de miles ni conversiones parciales. */
export function parseOperand(value: string): number | null {
  const normalized = value.trim().replace(',', '.')
  if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(normalized)) return null
  const number = Number(normalized)
  return Number.isFinite(number) ? number : null
}

export function formatNumber(value: number): string {
  const magnitude = Math.abs(value)
  return new Intl.NumberFormat('es-CO', {
    maximumSignificantDigits: 15,
    notation: magnitude >= 1e15 || (magnitude > 0 && magnitude < 1e-7) ? 'scientific' : 'standard',
  }).format(Object.is(value, -0) ? 0 : value)
}

export function expression(input: CalculationInput): string {
  const a = formatNumber(input.a)
  const b = input.b === undefined ? '' : formatNumber(input.b)
  if (input.operation === 'sqrt') return `√${a}`
  if (input.operation === 'percentage') return `${b} % de ${a}`
  const symbol = input.operation === 'power' ? '^' : operations.find((operation) => operation.id === input.operation)!.symbol
  return `${a} ${symbol} ${b}`
}

export class CalculationError extends Error {
  constructor(message: string, public readonly code: string) {
    super(message)
    this.name = 'CalculationError'
  }
}

/** La API es la única responsable de calcular; el cliente valida el contrato recibido. */
export async function calculate(input: CalculationInput, signal?: AbortSignal): Promise<Calculation> {
  const body = input.operation === 'sqrt' ? { a: input.a } : { a: input.a, b: input.b }
  let response: Response
  try {
    response = await fetch(`/api/v1/${input.operation}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new CalculationError('No pudimos conectar. Comprueba tu conexión e inténtalo de nuevo.', 'NETWORK_ERROR')
  }
  let payload: unknown
  try {
    payload = await response.json()
  } catch {
    throw new CalculationError('El servicio no devolvió una respuesta válida. Inténtalo de nuevo.', 'INVALID_RESPONSE')
  }
  if (!response.ok) {
    const error = payload && typeof payload === 'object' && 'error' in payload ? payload.error : undefined
    if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
      throw new CalculationError(error.message, 'code' in error && typeof error.code === 'string' ? error.code : 'API_ERROR')
    }
    throw new CalculationError('No pudimos completar el cálculo. Inténtalo de nuevo.', 'API_ERROR')
  }
  if (!payload || typeof payload !== 'object' || !('result' in payload) || typeof payload.result !== 'number' || !Number.isFinite(payload.result) || !('operation' in payload) || payload.operation !== input.operation) {
    throw new CalculationError('El servicio no devolvió una respuesta válida. Inténtalo de nuevo.', 'INVALID_RESPONSE')
  }
  return { ...input, result: payload.result }
}
