export default function LogsView({ lang, log, loading, live, onRefresh, onToggleLive }) {
  const zh = lang === 'zh'
  return <main className="connect-pane flex-1 overflow-y-auto px-8 pb-8 pt-6">
    <div className="max-w-4xl mx-auto">
      <span className="page-kicker inline-flex rounded-lg border border-violet-500/25 bg-violet-500/10 px-2.5 py-1.5 text-[9px] font-semibold tracking-[.14em] text-violet-300">LOGS</span>
      <div className="mt-4 flex items-end justify-between gap-4">
        <div><h1 className="text-lg font-semibold">{zh ? '运行日志' : 'Runtime logs'}</h1><p className="mt-2 text-[10px] text-[var(--text-faint)]">{zh ? '查看 Xray-core 最近的运行和错误日志。' : 'Inspect recent Xray-core runtime and error output.'}</p></div>
        <div className="flex items-center gap-2">
          <button aria-pressed={live} onClick={onToggleLive} className={`flex items-center gap-2 rounded-lg border px-4 py-2 text-[11px] transition-colors ${live ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-400' : 'border-[var(--border)] bg-[var(--bg-panel)] hover:bg-[var(--bg-hover)]'}`}>
            <i className={`h-1.5 w-1.5 rounded-full ${live ? 'animate-pulse bg-emerald-400' : 'bg-[var(--text-faint)]'}`} />
            {zh ? '实时' : 'Live'}
          </button>
          <button disabled={loading} onClick={onRefresh} className="rounded-lg border border-[var(--border)] bg-[var(--bg-panel)] px-4 py-2 text-[11px] hover:bg-[var(--bg-hover)] disabled:opacity-50">{loading ? (zh ? '刷新中…' : 'Refreshing…') : (zh ? '刷新日志' : 'Refresh')}</button>
        </div>
      </div>
      <div className="mt-6 min-h-[360px] rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] overflow-hidden">
        <div className="flex items-center gap-1.5 border-b border-[var(--border)] px-4 py-3"><i className="h-2 w-2 rounded-full bg-red-400/80" /><i className="h-2 w-2 rounded-full bg-amber-400/80" /><i className="h-2 w-2 rounded-full bg-emerald-400/80" /><span className="ml-2 text-[9px] text-[var(--text-faint)]">xray-core.log</span></div>
        <pre className="max-h-[calc(100vh-240px)] min-h-[320px] overflow-auto whitespace-pre-wrap break-all p-4 font-mono text-[10px] leading-5 text-[var(--text-dim)]">{log || (zh ? '暂无日志' : 'No log output')}</pre>
      </div>
    </div>
  </main>
}
