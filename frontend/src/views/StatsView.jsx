import { useEffect, useMemo, useState } from 'react'

const formatBytes = (value) => {
  let amount = Number(value) || 0
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let unit = 0
  while (amount >= 1024 && unit < units.length - 1) { amount /= 1024; unit++ }
  return `${unit > 0 && amount < 10 ? amount.toFixed(1) : Math.round(amount)} ${units[unit]}`
}

const formatSpeed = (value) => `${formatBytes(value)}/s`

const statusIcons = {
  globe: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18ZM3.6 9h16.8M3.6 15h16.8M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18',
  power: 'M12 3v9M18.4 6.6a8.5 8.5 0 1 1-12.8 0',
}

function StatusIcon({ path, active }) {
  return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={`h-4 w-4 ${active ? 'text-[#22c55e]' : 'text-[#9ca3af]'}`}><path d={path} /></svg>
}

const formatChartTime = (seconds) => {
  if (seconds === 0) return '0s'
  if (seconds < 60) return `-${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return `-${minutes}m${rest ? `${rest}s` : ''}`
}

function RealtimeSpeedChart({ samples, duration, onDurationChange, selectedSeries, onSelectedSeriesChange, t }) {
  const width = 720
  const height = 190
  const left = 58
  const right = 12
  const top = 12
  const bottom = 24
  const chartWidth = width - left - right
  const chartHeight = height - top - bottom
  const visibleSamples = samples.slice(-duration)
  const leadingSlots = Math.max(0, duration - visibleSamples.length)
  const series = [
    { key: 'up', label: t('totalUpload'), color: '#f59e0b' },
    { key: 'down', label: t('totalDownload'), color: '#22d3ee' },
    { key: 'proxyUp', label: t('proxyUpload'), color: '#a78bfa' },
    { key: 'proxyDown', label: t('proxyDownload'), color: '#6366f1' },
    { key: 'directUp', label: t('directUpload'), color: '#4ade80' },
    { key: 'directDown', label: t('directDownload'), color: '#10b981' },
  ]
  const activeSeries = series.filter(({ key }) => selectedSeries.includes(key))
  const maxValue = Math.max(1, ...visibleSamples.flatMap((sample) => activeSeries.map(({ key }) => sample[key] || 0)))
  const points = (key) => visibleSamples.map((sample, index) => {
    const x = left + ((leadingSlots + index) / Math.max(1, duration - 1)) * chartWidth
    const y = top + chartHeight - (sample[key] / maxValue) * chartHeight
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
  const current = visibleSamples.at(-1) || {}
  const durationLabel = duration === 60 ? t('oneMinute') : duration === 300 ? t('fiveMinutes') : t('thirtyMinutes')
  const toggleSeries = (key) => onSelectedSeriesChange(selectedSeries.includes(key) ? selectedSeries.filter((item) => item !== key) : [...selectedSeries, key])

  return <section className="mt-6 rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-5">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div><h2 className="text-xs font-semibold">{t('realtimeSpeed')}</h2><p className="mt-1 text-[9px] text-[var(--text-faint)]">{t('chartTimeRange', durationLabel)}</p></div>
      <div className="flex flex-wrap items-center justify-end gap-3 text-[10px] tabular-nums">
        <div className="flex rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] p-0.5">
          {[[60, t('oneMinute')], [300, t('fiveMinutes')], [1800, t('thirtyMinutes')]].map(([seconds, label]) => <button key={seconds} onClick={() => onDurationChange(seconds)} className={`rounded-md px-2 py-1 text-[9px] transition-colors ${duration === seconds ? 'bg-[var(--accent)] text-[var(--accent-text)]' : 'text-[var(--text-faint)] hover:text-[var(--text)]'}`}>{label}</button>)}
        </div>
      </div>
    </div>
    <div className="mt-3 flex flex-wrap gap-1.5">
      {series.map(({ key, label, color }) => {
        const active = selectedSeries.includes(key)
        return <button key={key} type="button" aria-pressed={active} onClick={() => toggleSeries(key)} className={`flex items-center gap-1.5 rounded-md border px-2 py-1 text-[9px] tabular-nums transition-colors ${active ? 'border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text)]' : 'border-transparent bg-[var(--bg-elevated)]/40 text-[var(--text-faint)] opacity-60'}`}>
          <i className="h-2 w-2 rounded-full" style={{ backgroundColor: color }} />{label} {formatSpeed(current[key] || 0)}
        </button>
      })}
    </div>
    <svg className="mt-4 h-48 w-full" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={t('realtimeSpeed')}>
      {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
        const y = top + chartHeight * ratio
        const value = maxValue * (1 - ratio)
        return <g key={ratio}><line x1={left} y1={y} x2={width - right} y2={y} stroke="var(--border)" strokeWidth="1" /><text x={left - 8} y={y + 3} textAnchor="end" fill="var(--text-faint)" fontSize="8">{formatSpeed(value)}</text></g>
      })}
      {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
        const x = left + ratio * chartWidth
        const secondsAgo = Math.round(duration * (1 - ratio))
        return <g key={ratio}><line x1={x} y1={top} x2={x} y2={top + chartHeight} stroke="var(--border)" strokeWidth="1" strokeDasharray="2 4" /><text x={x} y={height - 5} textAnchor={ratio === 0 ? 'start' : ratio === 1 ? 'end' : 'middle'} fill="var(--text-faint)" fontSize="8">{formatChartTime(secondsAgo)}</text></g>
      })}
      {activeSeries.map(({ key, color }) => <polyline key={key} points={points(key)} fill="none" stroke={color} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />)}
    </svg>
  </section>
}

export default function StatsView({ t, loadTraffic, speedSamples, speedDuration, onSpeedDurationChange, speedSeries, onSpeedSeriesChange, connected, canConnect, connectionBusy, proxyEnabled, proxyBusy, onToggleConnection, onToggleProxy }) {
  const [history, setHistory] = useState({ total: { uplink: 0, downlink: 0 }, daily: [], servers: [] })

  useEffect(() => {
    let active = true
    const refresh = () => loadTraffic().then((value) => {
      if (!active) return
      if (value?.history) setHistory(value.history)
    }).catch(() => {})
    refresh()
    const timer = window.setInterval(refresh, 1000)
    return () => { active = false; window.clearInterval(timer) }
  }, [loadTraffic])

  const days = useMemo(() => [...(history.daily || [])].slice(0, 14).reverse(), [history.daily])
  const dailySeries = [
    { key: 'up', label: t('totalUpload'), color: '#f59e0b', value: (day) => day.uplink || 0 },
    { key: 'down', label: t('totalDownload'), color: '#22d3ee', value: (day) => day.downlink || 0 },
    { key: 'proxyUp', label: t('proxyUpload'), color: '#a78bfa', value: (day) => day.proxy?.uplink || 0 },
    { key: 'proxyDown', label: t('proxyDownload'), color: '#6366f1', value: (day) => day.proxy?.downlink || 0 },
    { key: 'directUp', label: t('directUpload'), color: '#4ade80', value: (day) => day.direct?.uplink || 0 },
    { key: 'directDown', label: t('directDownload'), color: '#10b981', value: (day) => day.direct?.downlink || 0 },
  ]
  const maxDaily = Math.max(1, ...days.flatMap((day) => dailySeries.map(({ value }) => value(day))))
  const total = (history.total?.uplink || 0) + (history.total?.downlink || 0)

  return <main className="connect-pane flex-1 overflow-y-auto px-8 pb-8 pt-6">
    <div className="mx-auto max-w-5xl">
      <div className="flex flex-wrap items-start justify-between gap-5">
        <div>
          <span className="page-kicker inline-flex rounded-lg border border-violet-500/25 bg-violet-500/10 px-2.5 py-1.5 text-[9px] font-semibold tracking-[.14em] text-violet-300">STATS</span>
          <h1 className="mt-4 text-lg font-semibold">{t('trafficStats')}</h1>
          <p className="mt-2 text-[10px] text-[var(--text-faint)]">{t('trafficStatsHint')}</p>
        </div>
        <div className="mt-12 flex items-center gap-2">
          <button disabled={proxyBusy} onClick={onToggleProxy} className="flex min-h-9 items-center gap-2 whitespace-nowrap rounded-lg border border-[var(--border)] bg-[var(--bg-panel)] px-3 text-[10px] text-[var(--text-dim)] shadow-sm hover:bg-[var(--bg-hover)] disabled:opacity-50">
            <StatusIcon path={statusIcons.globe} active={proxyEnabled} />
            {proxyEnabled ? t('clearProxy') : t('applyProxy')}
          </button>
          <button disabled={connectionBusy || (!connected && !canConnect)} onClick={onToggleConnection} className="flex min-h-9 items-center gap-2 whitespace-nowrap rounded-lg border border-[var(--border)] bg-[var(--bg-panel)] px-3 text-[10px] text-[var(--text-dim)] shadow-sm hover:bg-[var(--bg-hover)] disabled:opacity-50">
            <StatusIcon path={statusIcons.power} active={connected} />
            {connected ? t('disconnectService') : t('connectService')}
          </button>
        </div>
      </div>

      <RealtimeSpeedChart samples={speedSamples} duration={speedDuration} onDurationChange={onSpeedDurationChange} selectedSeries={speedSeries} onSelectedSeriesChange={onSpeedSeriesChange} t={t} />

      <div className="mt-5 grid grid-cols-3 gap-3">
        {[[t('totalTraffic'), total, 'text-violet-400'], [t('upload'), history.total?.uplink, 'text-amber-400'], [t('download'), history.total?.downlink, 'text-cyan-400']].map(([label, value, color]) => <div key={label} className="rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-4"><div className="text-[9px] uppercase tracking-wider text-[var(--text-faint)]">{label}</div><div className={`mt-2 text-xl font-semibold tabular-nums ${color}`}>{formatBytes(value)}</div></div>)}
      </div>

      <div className="mt-3 grid grid-cols-2 gap-3">
        {[[t('proxyOutboundTraffic'), history.proxy, 'text-violet-400'], [t('directOutboundTraffic'), history.direct, 'text-emerald-400']].map(([label, value, color]) => <div key={label} className="rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-4">
          <div className="text-[9px] uppercase tracking-wider text-[var(--text-faint)]">{label}</div>
          <div className={`mt-2 text-lg font-semibold tabular-nums ${color}`}>{formatBytes((value?.uplink || 0) + (value?.downlink || 0))}</div>
          <div className="mt-2 flex gap-4 text-[9px] tabular-nums text-[var(--text-faint)]">
            <span className="text-amber-400">↑ {t('upload')} {formatBytes(value?.uplink)}</span>
            <span className="text-cyan-400">↓ {t('download')} {formatBytes(value?.downlink)}</span>
          </div>
        </div>)}
      </div>

      <section className="mt-5 rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-5">
        <h2 className="text-xs font-semibold">{t('dailyTraffic')}</h2>
        <div className="mt-3 flex flex-wrap gap-x-3 gap-y-1">
          {dailySeries.map(({ key, label, color }) => <span key={key} className="flex items-center gap-1.5 text-[8px] text-[var(--text-faint)]"><i className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: color }} />{label}</span>)}
        </div>
        {days.length ? <div className="mt-5 flex h-44 items-end gap-2">
          {days.map((day) => {
            return <div key={day.date} className="group flex min-w-0 flex-1 flex-col items-center justify-end gap-2">
              <div className="flex h-[120px] w-full items-end justify-center gap-px">
                {dailySeries.map(({ key, label, color, value }) => {
                  const amount = value(day)
                  return <div key={key} className="min-w-[2px] max-w-2 flex-1 rounded-t-sm" style={{ height: `${Math.max(amount > 0 ? 3 : 1, (amount / maxDaily) * 120)}px`, backgroundColor: color }} title={`${label}: ${formatBytes(amount)}`} />
                })}
              </div>
              <div className="truncate text-[8px] text-[var(--text-faint)]">{day.date.slice(5)}</div>
            </div>
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
