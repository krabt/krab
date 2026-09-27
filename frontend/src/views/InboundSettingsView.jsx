const defaultProxyConfig = {
  httpPort: 5889,
  httpsPort: 5889,
  socksPort: 5888,
}

export default function InboundSettingsView({ t, config }) {
  const current = { ...defaultProxyConfig, ...(config || {}) }
  const httpPorts = [...new Set([current.httpPort, current.httpsPort])].join(', ')
  const inbounds = [
    { protocol: 'HTTP', address: '127.0.0.1', port: httpPorts, configuration: t('readOnly') },
    { protocol: 'SOCKS5', address: '127.0.0.1', port: current.socksPort, configuration: t('readOnly') },
  ]

  return <main className="connect-pane flex-1 overflow-y-auto px-8 pb-8 pt-6">
    <div className="mx-auto max-w-4xl">
      <span className="page-kicker inline-flex rounded-lg border border-violet-500/25 bg-violet-500/10 px-2.5 py-1.5 text-[9px] font-semibold tracking-[.14em] text-violet-300">INBOUNDS</span>
      <h1 className="mt-4 text-lg font-semibold">{t('inboundRules')}</h1>
      <p className="mt-2 text-[10px] text-[var(--text-faint)]">{t('inboundRulesHint')}</p>

      <div className="mt-6 overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--bg-panel)]">
        <div className="grid grid-cols-[1fr_1.4fr_1fr_90px] gap-4 border-b border-[var(--border)] bg-[var(--bg-elevated)]/50 px-5 py-3 text-[9px] font-medium uppercase tracking-wider text-[var(--text-faint)]">
          <span>{t('protocol')}</span>
          <span>{t('listenAddress')}</span>
          <span>{t('port')}</span>
          <span>{t('configuration')}</span>
        </div>
        <div className="divide-y divide-[var(--border)]">
          {inbounds.map((inbound) => <div key={inbound.protocol} className="grid grid-cols-[1fr_1.4fr_1fr_90px] items-center gap-4 px-5 py-4 text-[11px]">
            <span className="font-semibold text-violet-300">{inbound.protocol}</span>
            <span className="font-mono text-[var(--text-dim)]">{inbound.address}</span>
            <span className="font-mono tabular-nums text-[var(--text)]">{inbound.port}</span>
            <span className="w-fit rounded-full border border-[var(--border)] bg-[var(--bg-elevated)] px-2 py-1 text-[9px] text-[var(--text-faint)]">{inbound.configuration}</span>
          </div>)}
        </div>
      </div>
      <p className="mt-4 text-[10px] text-[var(--text-faint)]">{t('inboundPortsFromSystemSettings')}</p>
    </div>
  </main>
}
