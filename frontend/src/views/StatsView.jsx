import { useEffect, useMemo, useState } from 'react'

const formatBytes = (value) => {
  let amount = Number(value) || 0
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let unit = 0
  while (amount >= 1024 && unit < units.length - 1) { amount /= 1024; unit++ }
  return `${unit > 0 && amount < 10 ? amount.toFixed(1) : Math.round(amount)} ${units[unit]}`
}

export default function StatsView({ t, loadTraffic }) {
  const [history, setHistory] = useState({ total: { uplink: 0, downlink: 0 }, daily: [], servers: [] })

  useEffect(() => {
    let active = true
    const refresh = () => loadTraffic().then((value) => {
      if (active && value?.history) setHistory(value.history)
    }).catch(() => {})
    refresh()
    const timer = window.setInterval(refresh, 2000)
    return () => { active = false; window.clearInterval(timer) }
  }, [loadTraffic])

  const days = useMemo(() => [...(history.daily || [])].slice(0, 14).reverse(), [history.daily])
  const maxDaily = Math.max(1, ...days.map((day) => (day.uplink || 0) + (day.downlink || 0)))
  const total = (history.total?.uplink || 0) + (history.total?.downlink || 0)

  return <main className="connect-pane flex-1 overflow-y-auto px-8 pb-8 pt-6">
    <div className="mx-auto max-w-5xl">
      <span className="page-kicker inline-flex rounded-lg border border-violet-500/25 bg-violet-500/10 px-2.5 py-1.5 text-[9px] font-semibold tracking-[.14em] text-violet-300">STATS</span>
      <h1 className="mt-4 text-lg font-semibold">{t('trafficStats')}</h1>
      <p className="mt-2 text-[10px] text-[var(--text-faint)]">{t('trafficStatsHint')}</p>

      <div className="mt-6 grid grid-cols-3 gap-3">
        {[[t('totalTraffic'), total, 'text-violet-400'], [t('upload'), history.total?.uplink, 'text-amber-400'], [t('download'), history.total?.downlink, 'text-cyan-400']].map(([label, value, color]) => <div key={label} className="rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-4"><div className="text-[9px] uppercase tracking-wider text-[var(--text-faint)]">{label}</div><div className={`mt-2 text-xl font-semibold tabular-nums ${color}`}>{formatBytes(value)}</div></div>)}
      </div>

      <section className="mt-5 rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-5">
        <h2 className="text-xs font-semibold">{t('dailyTraffic')}</h2>
        {days.length ? <div className="mt-5 flex h-44 items-end gap-2">
          {days.map((day) => {
            const amount = (day.uplink || 0) + (day.downlink || 0)
            return <div key={day.date} className="group flex min-w-0 flex-1 flex-col items-center justify-end gap-2"><div className="text-[8px] text-[var(--text-faint)] opacity-0 transition-opacity group-hover:opacity-100">{formatBytes(amount)}</div><div className="w-full max-w-10 rounded-t-md bg-gradient-to-t from-violet-600 to-cyan-400" style={{ height: `${Math.max(4, (amount / maxDaily) * 120)}px` }} /><div className="truncate text-[8px] text-[var(--text-faint)]">{day.date.slice(5)}</div></div>
          })}
        </div> : <div className="py-14 text-center text-[10px] text-[var(--text-faint)]">{t('noTrafficHistory')}</div>}
      </section>

      <section className="mt-5 overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--bg-panel)]">
        <div className="border-b border-[var(--border)] px-5 py-4 text-xs font-semibold">{t('trafficByServer')}</div>
        {(history.servers || []).length ? <div className="divide-y divide-[var(--border)]">{history.servers.map((server) => <div key={server.id} className="grid grid-cols-[minmax(0,1fr)_110px_110px_110px] items-center gap-3 px-5 py-3 text-[10px]"><span className="truncate font-medium">{server.name}</span><span className="text-right tabular-nums text-amber-400">↑ {formatBytes(server.uplink)}</span><span className="text-right tabular-nums text-cyan-400">↓ {formatBytes(server.downlink)}</span><span className="text-right tabular-nums text-[var(--text-dim)]">{formatBytes((server.uplink || 0) + (server.downlink || 0))}</span></div>)}</div> : <div className="py-12 text-center text-[10px] text-[var(--text-faint)]">{t('noTrafficHistory')}</div>}
      </section>
    </div>
  </main>
}
