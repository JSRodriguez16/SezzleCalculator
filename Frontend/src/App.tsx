import { useEffect, useRef, useState, type FormEvent } from 'react'
import { ArrowRight, Check, Copy, CornerDownLeft, History, LoaderCircle, RotateCcw, Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { calculate, expression, formatNumber, operations, parseOperand, type Calculation, type Operation } from '@/lib/calculator'
import { appendHistory, readHistory, saveHistory, type HistoryEntry } from '@/lib/history'

const requestTimeout = 15_000

export default function App() {
  const [operation, setOperation] = useState<Operation>('add')
  const [a, setA] = useState('')
  const [b, setB] = useState('')
  const [result, setResult] = useState<Calculation | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [history, setHistory] = useState(readHistory)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [notice, setNotice] = useState('')
  const request = useRef<AbortController | null>(null)
  const firstInput = useRef<HTMLInputElement>(null)
  const selected = operations.find((item) => item.id === operation)!

  useEffect(() => { saveHistory(history) }, [history])
  useEffect(() => () => { request.current?.abort() }, [])

  function clearFeedback() {
    setError('')
    setResult(null)
  }

  function selectOperation(value: Operation) {
    setOperation(value)
    clearFeedback()
  }

  function reset() {
    setA('')
    setB('')
    clearFeedback()
    firstInput.current?.focus()
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (request.current) return
    const first = parseOperand(a)
    const second = operation === 'sqrt' ? undefined : parseOperand(b)
    if (first === null || second === null) {
      setError('Escribe números válidos. Puedes usar coma o punto decimal, sin separadores de miles.')
      setResult(null)
      return
    }
    const controller = new AbortController()
    request.current = controller
    setBusy(true)
    setError('')
    setResult(null)
    let timedOut = false
    const timeout = window.setTimeout(() => { timedOut = true; controller.abort() }, requestTimeout)
    try {
      const calculation = await calculate({ operation, a: first, ...(second === undefined ? {} : { b: second }) }, controller.signal)
      setResult(calculation)
      setHistory((previous) => appendHistory(previous, calculation))
    } catch (failure) {
      if (timedOut) setError('El cálculo tardó demasiado. Inténtalo de nuevo.')
      else if (!controller.signal.aborted) setError(failure instanceof Error ? failure.message : 'Ocurrió un error inesperado.')
    } finally {
      window.clearTimeout(timeout)
      request.current = null
      setBusy(false)
    }
  }

  function reuse(entry: HistoryEntry) {
    setOperation(entry.operation)
    setA(String(entry.a))
    setB(entry.b === undefined ? '' : String(entry.b))
    clearFeedback()
    firstInput.current?.focus()
  }

  async function copy(entry: HistoryEntry) {
    try {
      await navigator.clipboard.writeText(String(entry.result))
      setCopiedId(entry.id)
      setNotice('Resultado copiado al portapapeles.')
    } catch {
      setNotice('No pudimos copiar el resultado. Puedes seleccionarlo y copiarlo manualmente.')
    }
  }

  return (
    <div className="app-shell">
      <a className="skip-link" href="#calculator">Ir a la calculadora</a>
      <header className="site-header">
        <div className="header-inner">
          <a className="brand" href="/" aria-label="Sezzle, inicio">
            <img src="/favicon.svg" alt="" width="34" height="34" />
            <span>sezzle</span>
          </a>
          </div>
      </header>

      <main className="main-content">

        <div className="workspace">
          <section aria-label="Calculadora" id="calculator" className="calculator-shell">
            <div className="calculator-display">
              <div className="display-heading"><span><span className="display-dot" /> CALCULADORA</span><span className="display-mode">{selected.label}</span></div>
              <div className="result-space" aria-live="polite" aria-atomic="true">
                <p className="display-expression">{busy ? 'Un momento, estamos calculando…' : result ? `${expression(result)} =` : 'Ingresa tus datos para empezar'}</p>
                <output className={`display-result ${result && formatNumber(result.result).length > 13 ? 'display-result-long' : ''}`} aria-label="Resultado">{busy ? <span className="result-pending">···</span> : result ? formatNumber(result.result) : '0'}</output>
              </div>
            </div>

            <form className="calculator-form" onSubmit={submit} noValidate>
              <fieldset className="operation-fieldset" disabled={busy}>
                <div className="operation-grid">
                  {operations.map((item) => (
                    <Button key={item.id} type="button" variant="ghost" className={`operation-button ${operation === item.id ? 'operation-active' : ''}`} aria-pressed={operation === item.id} aria-label={item.label} onClick={() => selectOperation(item.id)}>
                      <span className="operation-symbol" aria-hidden="true">{item.symbol}</span><span className="operation-label">{item.label}</span>
                    </Button>
                  ))}
                </div>
              </fieldset>

              <div className={`operand-grid ${operation === 'sqrt' ? 'operand-single' : ''}`}>
                <div className="operand-field">
                  <label htmlFor="operand-a">{selected.aLabel}</label>
                  <Input ref={firstInput} id="operand-a" name="a" inputMode="decimal" autoComplete="off" placeholder="0" maxLength={128} value={a} disabled={busy} aria-invalid={Boolean(error)} aria-describedby={error ? 'calculation-error number-hint' : 'number-hint'} onChange={(event) => { setA(event.target.value); clearFeedback() }} />
                </div>
                {operation !== 'sqrt' && <><span className="operand-symbol" aria-hidden="true">{operation === 'power' ? '^' : operation === 'percentage' ? '×' : selected.symbol}</span><div className="operand-field"><label htmlFor="operand-b">{selected.bLabel}</label><Input id="operand-b" name="b" inputMode="decimal" autoComplete="off" placeholder="0" maxLength={128} value={b} disabled={busy} aria-invalid={Boolean(error)} aria-describedby={error ? 'calculation-error number-hint' : 'number-hint'} onChange={(event) => { setB(event.target.value); clearFeedback() }} />{operation === 'percentage' && <span className="input-suffix" aria-hidden="true">%</span>}</div></>}
              </div>
              <p id="number-hint" className="number-hint">{selected.description}</p>
              {error && <div id="calculation-error" className="calculation-error" role="alert"><span className="error-mark" aria-hidden="true">!</span><p>{error}</p><Button type="button" variant="ghost" size="icon" aria-label="Cerrar error" onClick={() => setError('')}><X /></Button></div>}
              <div className="form-actions">
                <Button type="button" variant="outline" className="reset-button" onClick={reset} disabled={busy}><RotateCcw /> Limpiar</Button>
                <Button type="submit" className="calculate-button" disabled={busy}>{busy ? <LoaderCircle className="animate-spin" /> : null}{busy ? 'Calculando…' : 'Calcular'}{!busy && <ArrowRight />}</Button>
              </div>
              <p className="keyboard-hint"><CornerDownLeft size={12} /> También puedes presionar <kbd>Enter</kbd></p>
            </form>
          </section>

          <aside className="side-panel" aria-label="Historial y consejos">
            <Card className="history-card">
              <CardHeader className="history-header"><CardTitle><History size={18} /> Historial <span className="history-count">{history.length}</span></CardTitle><Button type="button" variant="ghost" size="icon" aria-label="Borrar historial" disabled={history.length === 0 || busy} onClick={() => { setHistory([]); setCopiedId(null) }}><Trash2 size={16} /></Button></CardHeader>
              <CardContent className="history-content">
                {history.length === 0 ? <div className="history-empty"><div className="history-illustration" aria-hidden="true"><span className="empty-line" /><span className="empty-line" /><span className="empty-line" /><div><History size={22} strokeWidth={1.6} /></div></div><p>Los cálculos que hagas aparecerán aquí.<br /></p><span className="empty-tag"></span></div> : <ol className="history-list" aria-label="Cálculos anteriores">{history.map((entry) => <li key={entry.id} className="history-item"><div className="history-item-top"><span>{operations.find((item) => item.id === entry.operation)!.label}</span><time dateTime={entry.createdAt}>{new Intl.DateTimeFormat('es-CO', { hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(entry.createdAt))}</time></div><p className="history-expression">{expression(entry)}</p><div className="history-result-row"><span className="history-result">{formatNumber(entry.result)}</span><div className="history-actions"><Button type="button" variant="ghost" size="icon" aria-label={`Reutilizar ${expression(entry)}`} title="Reutilizar cálculo" disabled={busy} onClick={() => reuse(entry)}><RotateCcw size={14} /></Button><Button type="button" variant="ghost" size="icon" aria-label={`Copiar resultado ${formatNumber(entry.result)}`} title={copiedId === entry.id ? 'Copiado' : 'Copiar resultado'} onClick={() => void copy(entry)}>{copiedId === entry.id ? <Check size={14} /> : <Copy size={14} />}</Button></div></div></li>)}</ol>}
              </CardContent>
              <div className="history-footer"><span className="history-storage-dot" /> El historial guarda los ultimos 20 calculos.</div>
            </Card>
            <p role="status" className="history-notice">{notice}</p>
          </aside>
        </div>
      </main>
    </div>
  )
}
