import { useEffect, useState } from 'react'

const defaultXraySettings = { enabled: false, assetDir: '~/.krab', useGHProxy: false, logLevel: 'debug', dnsHosts: {}, dnsServers: [] }
const lines = (values) => (values || []).join('\n')
const parseLines = (value) => value.split('\n').map((item) => item.trim()).filter(Boolean)
const formatHosts = (hosts) => Object.entries(hosts || {}).map(([host, target]) => `${host}=${target}`).join('\n')
const parseHosts = (value) => Object.fromEntries(value.split('\n').map((line) => {
  const separator = line.indexOf('=')
  return separator < 0 ? ['', ''] : [line.slice(0, separator).trim(), line.slice(separator + 1).trim()]
}).filter(([host, target]) => host && target))

export default function ProxySettingsView({ lang, t, version, onOpenAbout, config, onConfigChange, saveProxyConfig, loadGeoSettings, saveGeoSettings, updateGeoData, loadAutoStart, setAutoStart }) {
  const zh = lang === 'zh'
  const [busy, setBusy] = useState('')
  const [message, setMessage] = useState('')
  const [proxyMessage, setProxyMessage] = useState('')
  const [geo, setGeo] = useState(defaultXraySettings)
  const [dnsHosts, setDNSHosts] = useState('')
  const [dnsServers, setDNSServers] = useState('')
  const [autoStart, setAutoStartState] = useState(false)
  const [autoStartBusy, setAutoStartBusy] = useState(true)
  const [autoStartMessage, setAutoStartMessage] = useState('')

  useEffect(() => { loadGeoSettings().then((value) => {
    if (!value) return
    setGeo({ ...defaultXraySettings, ...value })
    setDNSHosts(formatHosts(value.dnsHosts))
    setDNSServers(lines(value.dnsServers))
  }).catch(() => {}) }, [loadGeoSettings])

  useEffect(() => {
    let active = true
    loadAutoStart().then((enabled) => {
      if (active) setAutoStartState(Boolean(enabled))
    }).catch((error) => {
      if (active) setAutoStartMessage(error?.message || String(error))
    }).finally(() => {
      if (active) setAutoStartBusy(false)
    })
    return () => { active = false }
  }, [loadAutoStart])

  const update = (key, value) => onConfigChange((current) => ({ ...current, [key]: value }))

  async function saveProxy() {
    setBusy('proxy'); setProxyMessage('')
    try {
      await saveProxyConfig(config)
      setProxyMessage(t('proxySettingsSaved'))
    }
    catch (error) { setProxyMessage(error?.message || String(error)) }
    finally { setBusy('') }
  }

  async function saveGeo() {
    setBusy('geo'); setMessage('')
    try {
      const current = await loadGeoSettings()
      await saveGeoSettings({ ...current, enabled: geo.enabled, assetDir: geo.assetDir, useGHProxy: geo.useGHProxy })
      setMessage(t('geoSettingsSaved'))
    }
    catch (error) { setMessage(error?.message || String(error)) }
    finally { setBusy('') }
  }

  async function updateGeo() {
    setBusy('geo-update'); setMessage('')
    try {
      const current = await loadGeoSettings()
      await saveGeoSettings({ ...current, assetDir: geo.assetDir, useGHProxy: geo.useGHProxy })
      await updateGeoData(geo.assetDir, geo.useGHProxy)
      setMessage(t('geoDataUpdated'))
    }
    catch (error) { setMessage(error?.message || String(error)) }
    finally { setBusy('') }
  }

  async function saveRuntime() {
    setBusy('runtime'); setMessage('')
    try {
      const current = await loadGeoSettings()
      await saveGeoSettings({ ...current, logLevel: geo.logLevel, dnsHosts: parseHosts(dnsHosts), dnsServers: parseLines(dnsServers) })
      setMessage(t('xraySettingsSaved'))
    }
    catch (error) { setMessage(error?.message || String(error)) }
    finally { setBusy('') }
  }

  async function toggleAutoStart(event) {
    const enabled = event.target.checked
    setAutoStartBusy(true); setAutoStartMessage('')
    try {
      await setAutoStart(enabled)
      setAutoStartState(enabled)
      setAutoStartMessage(enabled ? t('autoStartEnabled') : t('autoStartDisabled'))
    }
    catch (error) { setAutoStartMessage(error?.message || String(error)) }
    finally { setAutoStartBusy(false) }
  }

  return <main className="connect-pane flex-1 overflow-y-auto px-8 pb-8 pt-6">
    <div className="max-w-2xl mx-auto">
      <span className="page-kicker inline-flex rounded-lg border border-violet-500/25 bg-violet-500/10 px-2.5 py-1.5 text-[9px] font-semibold tracking-[.14em] text-violet-300">SETTINGS</span>
      <h1 className="text-lg font-semibold mt-4">{zh ? '系统配置' : 'System settings'}</h1>
      <p className="text-[10px] leading-4 text-[var(--text-faint)] mt-2">{zh ? '管理操作系统代理和 Xray GeoData 分流配置。' : 'Manage operating-system proxy and Xray GeoData routing settings.'}</p>

      <section className="mt-7">
        <div className="mb-3"><h2 className="text-[11px] font-semibold">{t('startupSettings')}</h2><p className="mt-1 text-[9px] text-[var(--text-faint)]">{t('startupSettingsHint')}</p></div>
        <div className="proxy-settings-card flex items-center justify-between gap-4 rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-5">
          <div><h3 className="text-[11px] font-medium">{t('launchAtLogin')}</h3><p className="mt-1 text-[9px] leading-4 text-[var(--text-faint)]">{t('launchAtLoginHint')}</p></div>
          <input type="checkbox" checked={autoStart} disabled={autoStartBusy} onChange={toggleAutoStart} aria-label={t('launchAtLogin')} />
        </div>
        {autoStartMessage && <div className="mt-3 text-[10px] text-[var(--text-dim)]">{autoStartMessage}</div>}
      </section>

      <section className="mt-10">
        <div className="mb-3"><h2 className="text-[11px] font-semibold">{zh ? '系统代理' : 'System proxy'}</h2><p className="text-[9px] text-[var(--text-faint)] mt-1">{zh ? '分别配置 HTTP、HTTPS 与 SOCKS5，点击设置后才会写入系统。' : 'Configure HTTP, HTTPS and SOCKS5 independently; changes apply only on request.'}</p></div>
        <div className="proxy-settings-card rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] divide-y divide-[var(--border)] overflow-hidden">
        {[['HTTP', 'httpHost', 'httpPort'], ['HTTPS', 'httpsHost', 'httpsPort'], ['SOCKS5', 'socksHost', 'socksPort']].map(([label, host, port]) =>
          <div key={label} className="proxy-settings-row grid grid-cols-[90px_1fr_120px] items-center gap-3 p-4 hover:bg-[var(--bg-hover)] transition-colors">
            <span className="proxy-protocol-label text-[10px] font-semibold text-violet-200">{label}</span>
            <input autoCapitalize="none" autoCorrect="off" spellCheck={false} value={config[host]} onChange={(e) => update(host, e.target.value)} className="rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] px-3 py-2 text-[11px] focus:outline-none focus:border-[var(--accent)]" placeholder="127.0.0.1" />
            <input type="text" inputMode="numeric" autoCapitalize="none" autoCorrect="off" spellCheck={false} value={config[port]} onChange={(e) => update(port, e.target.value)} className="rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] px-3 py-2 text-[11px] focus:outline-none focus:border-[var(--accent)]" />
          </div>)}
        </div>
        <div className="mt-4 flex items-center gap-3">
          <button disabled={Boolean(busy)} onClick={saveProxy} className="rounded-lg bg-[var(--accent)] px-4 py-2 text-[11px] text-white hover:bg-[var(--accent-hover)] disabled:opacity-50">{busy === 'proxy' ? t('saving') : t('saveProxySettings')}</button>
          {proxyMessage && <span className="text-[10px] text-[var(--text-dim)]">{proxyMessage}</span>}
        </div>
      </section>

      <section className="mt-10">
        <div className="mb-3"><h2 className="text-[11px] font-semibold">{zh ? 'GeoData 配置' : 'GeoData settings'}</h2><p className="text-[9px] text-[var(--text-faint)] mt-1">{zh ? '配置 Xray 的 geoip.dat 和 geosite.dat 分流资源。' : 'Configure the geoip.dat and geosite.dat routing assets used by Xray.'}</p></div>
        <div className="proxy-settings-card rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-5">
        <div className="flex items-center justify-between gap-4">
          <div><h3 className="text-[11px] font-medium">{zh ? '启用 GeoData 分流' : 'Enable GeoData routing'}</h3><p className="text-[9px] leading-4 text-[var(--text-faint)] mt-1">{zh ? '使用 geosite:cn、geoip:cn 和 geoip:private 规则进行直连分流。' : 'Route geosite:cn, geoip:cn and geoip:private directly.'}</p></div>
          <label className="flex items-center gap-2 text-xs text-[var(--text-dim)]"><input type="checkbox" checked={geo.enabled} onChange={(e) => setGeo({ ...geo, enabled: e.target.checked })} className="accent-[var(--accent)]" />{zh ? '启用' : 'Enable'}</label>
        </div>
        <label className="block mt-4 text-[11px] text-[var(--text-faint)]">{zh ? '资源目录（必须同时包含 geoip.dat 和 geosite.dat）' : 'Asset directory (must contain both geoip.dat and geosite.dat)'}</label>
        <input autoCapitalize="none" autoCorrect="off" spellCheck={false} value={geo.assetDir} onChange={(e) => setGeo({ ...geo, assetDir: e.target.value })} placeholder="~/.krab" className="mt-2 w-full rounded-lg bg-[var(--bg-elevated)] border border-[var(--border)] px-3 py-2 text-[11px] focus:border-[var(--accent)]" />
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button disabled={Boolean(busy)} onClick={saveGeo} className="rounded-lg bg-[var(--accent)] px-4 py-2 text-[11px] text-white hover:bg-[var(--accent-hover)] disabled:opacity-50">{busy === 'geo' ? t('saving') : t('saveGeoSettings')}</button>
          <button disabled={Boolean(busy)} onClick={updateGeo} className="rounded-lg border border-[var(--border-strong)] bg-[var(--bg-elevated)] px-4 py-2 text-[11px] text-[var(--text)] hover:bg-[var(--bg-hover)] disabled:opacity-50">{busy === 'geo-update' ? t('updatingGeoData') : t('updateGeoData')}</button>
          <label className="flex items-center gap-2 text-[10px] text-[var(--text-dim)]"><input type="checkbox" checked={Boolean(geo.useGHProxy)} onChange={(event) => setGeo({ ...geo, useGHProxy: event.target.checked })} className="accent-[var(--accent)]" />{t('useGHProxy')}</label>
        </div>
        </div>
      </section>

      <section className="mt-8">
        <div className="mb-3"><h2 className="text-[11px] font-semibold">{t('xrayRuntimeSettings')}</h2><p className="mt-1 text-[9px] text-[var(--text-faint)]">{t('xrayRuntimeSettingsHint')}</p></div>
        <div className="proxy-settings-card rounded-xl border border-[var(--border)] bg-[var(--bg-panel)] p-5">
          <label className="block text-[11px] text-[var(--text-faint)]">{t('logLevel')}</label>
          <select value={geo.logLevel || 'debug'} onChange={(event) => setGeo({ ...geo, logLevel: event.target.value })} className="mt-2 h-10 w-full rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] px-3 text-[11px] focus:border-[var(--accent)] focus:outline-none">
            {['debug', 'info', 'warning', 'error', 'none'].map((level) => <option key={level} value={level}>{level}</option>)}
          </select>

          <div className="mt-5 border-t border-[var(--border)] pt-5">
            <h3 className="text-[11px] font-medium">{t('dnsSettings')}</h3>
            <p className="mt-1 text-[9px] text-[var(--text-faint)]">{t('dnsSettingsHint')}</p>
            <div className="mt-4 grid grid-cols-2 gap-4">
              <label className="block"><span className="text-[11px] text-[var(--text-faint)]">{t('dnsHosts')}</span><span className="ml-2 text-[9px] text-[var(--text-faint)]">{t('dnsHostsHint')}</span><textarea autoCapitalize="none" autoCorrect="off" spellCheck={false} value={dnsHosts} onChange={(event) => setDNSHosts(event.target.value)} placeholder={'example.com=1.2.3.4\ndomain:example.org=example.com'} className="mt-2 h-28 w-full resize-none rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] px-3 py-2 font-mono text-[11px] leading-5 focus:border-[var(--accent)] focus:outline-none" /></label>
              <label className="block"><span className="text-[11px] text-[var(--text-faint)]">{t('dnsServers')}</span><span className="ml-2 text-[9px] text-[var(--text-faint)]">{t('onePerLine')}</span><textarea autoCapitalize="none" autoCorrect="off" spellCheck={false} value={dnsServers} onChange={(event) => setDNSServers(event.target.value)} placeholder={'1.1.1.1\n8.8.8.8\nhttps://dns.google/dns-query'} className="mt-2 h-28 w-full resize-none rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] px-3 py-2 font-mono text-[11px] leading-5 focus:border-[var(--accent)] focus:outline-none" /></label>
            </div>
          </div>
          <button disabled={Boolean(busy)} onClick={saveRuntime} className="mt-5 rounded-lg bg-[var(--accent)] px-4 py-2 text-[11px] text-white hover:bg-[var(--accent-hover)] disabled:opacity-50">{busy === 'runtime' ? t('saving') : t('saveXraySettings')}</button>
        </div>
      </section>
      {message && <div className="mt-4 text-xs text-[var(--text-dim)]">{message}</div>}
      <div className="mt-8 flex justify-center border-t border-[var(--border)] pt-6">
        <button onClick={onOpenAbout} className="rounded-lg border border-[var(--border-strong)] bg-[var(--bg-elevated)] px-4 py-2 text-[11px] text-[var(--text-dim)] hover:bg-[var(--bg-hover)] hover:text-[var(--text)]">
          {t('version')} {version ? `v${version.replace(/^v/, '')}` : '—'}
        </button>
      </div>
    </div>
  </main>
}
