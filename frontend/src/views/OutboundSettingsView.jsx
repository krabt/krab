import { useEffect, useMemo, useState } from 'react'

const empty = { rules: [] }
const lines = (value) => (value || []).join('\n')
const list = (value) => value.split(/[,\r\n]/).map((item) => item.trim()).filter(Boolean)
const sameList = (left, right) => left.length === right.length && left.every((item, index) => item === right[index])
const ruleID = () => `rule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
const createRule = (name) => ({ id: ruleID(), name, enabled: true, whitelistDomains: [], whitelistIPs: [], blacklistDomains: [], blacklistIPs: [], proxyDomains: [], proxyIPs: [] })

function RuleField({ label, hint, placeholder, value, onChange }) {
  const [draft, setDraft] = useState(() => lines(value))

  useEffect(() => {
    if (!sameList(list(draft), value || [])) setDraft(lines(value))
  }, [value, draft])

  return <label className="block">
    <span className="text-[11px] font-medium">{label}</span>
    <span className="ml-2 text-[9px] text-[var(--text-faint)]">{hint}</span>
    <textarea autoCapitalize="none" autoCorrect="off" spellCheck={false} value={draft} placeholder={placeholder} onChange={(event) => { setDraft(event.target.value); onChange(list(event.target.value)) }} className="mt-2 h-28 w-full resize-none rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] px-3 py-2 font-mono text-[11px] leading-5 placeholder:text-[var(--text-faint)]/60 focus:border-[var(--accent)] focus:outline-none" />
  </label>
}

const itemCount = (rule) => ['whitelistDomains', 'whitelistIPs', 'blacklistDomains', 'blacklistIPs', 'proxyDomains', 'proxyIPs'].reduce((total, key) => total + (rule?.[key]?.length || 0), 0)

export default function OutboundSettingsView({ lang, t, loadSettings, saveRule, deleteRule }) {
  const zh = lang === 'zh'
  const [settings, setSettings] = useState(empty)
  const [selectedID, setSelectedID] = useState('')
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => {
    loadSettings().then((value) => {
      const next = value?.rules ? value : empty
      setSettings(next)
      setSelectedID('')
    }).catch(() => {})
  }, [loadSettings])

  const selected = useMemo(() => settings.rules.find((rule) => rule.id === selectedID), [settings.rules, selectedID])
  const updateRule = (values) => setSettings((current) => ({ ...current, rules: current.rules.map((rule) => rule.id === selectedID ? { ...rule, ...values } : rule) }))
  const addRule = () => {
    const rule = createRule(zh ? `规则 ${settings.rules.length + 1}` : `Rule ${settings.rules.length + 1}`)
    setSettings((current) => ({ ...current, rules: [...current.rules, rule] }))
    setSelectedID(rule.id)
    setMessage('')
  }
  const removeRule = async () => {
    if (!selected) return
    setBusy(true); setMessage('')
    try {
      await deleteRule(selected.id)
      const index = settings.rules.findIndex((rule) => rule.id === selectedID)
      const rules = settings.rules.filter((rule) => rule.id !== selectedID)
      setSettings((current) => ({ ...current, rules }))
      setSelectedID(rules[Math.min(index, rules.length - 1)]?.id || '')
      setMessage(zh ? '规则已删除' : 'Rule deleted')
    } catch (error) { setMessage(error?.message || String(error)) }
    finally { setBusy(false) }
  }
  async function saveSelectedRule() {
    if (!selected) return
    setBusy(true); setMessage('')
    try {
      await saveRule(selected)
      setMessage(zh ? `“${selected.name}”已保存，下次连接时生效` : `“${selected.name}” saved; it applies on the next connection`)
    } catch (error) { setMessage(error?.message || String(error)) }
    finally { setBusy(false) }
  }

  return <main className="connect-pane flex-1 overflow-y-auto px-8 pb-8 pt-6">
    <div className="mx-auto max-w-6xl">
      <div className="flex items-end justify-between gap-4">
        <div>
          <span className="page-kicker inline-flex rounded-lg border border-violet-500/25 bg-violet-500/10 px-2.5 py-1.5 text-[9px] font-semibold tracking-[.14em] text-violet-300">OUTBOUND RULES</span>
          <h1 className="mt-4 text-lg font-semibold">{t('outboundRules')}</h1>
          <p className="mt-2 text-[10px] text-[var(--text-faint)]">{zh ? '可创建多个规则组；每个规则都可以独立配置域名与 IP 黑白名单。规则按照列表顺序应用。' : 'Create multiple rule groups, each with its own domain and IP allow/block lists. Rules apply in list order.'}</p>
        </div>
        <button onClick={addRule} className="shrink-0 rounded-lg bg-[var(--accent)] px-4 py-2 text-[11px] text-white hover:bg-[var(--accent-hover)]">＋ {zh ? '新增规则' : 'Add rule'}</button>
      </div>

      <div className="mt-6 grid min-h-[480px] grid-cols-[220px_minmax(0,1fr)] overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--bg-panel)]">
        <aside className="border-r border-[var(--border)] bg-[var(--bg-elevated)]/40 p-3">
          <div className="mb-2 px-2 text-[9px] font-medium uppercase tracking-[.14em] text-[var(--text-faint)]">{zh ? `${settings.rules.length} 条规则` : `${settings.rules.length} rules`}</div>
          <div className="space-y-1.5">
            {settings.rules.map((rule, index) => <button key={rule.id} aria-expanded={rule.id === selectedID} onClick={() => setSelectedID((current) => current === rule.id ? '' : rule.id)} className={`w-full rounded-xl border px-3 py-3 text-left transition-colors ${rule.id === selectedID ? 'border-violet-500/45 bg-violet-500/10' : 'border-transparent hover:border-[var(--border)] hover:bg-[var(--bg-hover)]'}`}>
              <div className="flex items-center gap-2"><span className={`h-1.5 w-1.5 rounded-full ${rule.enabled ? 'bg-emerald-400' : 'bg-[var(--text-faint)]'}`} /><span className="min-w-0 flex-1 truncate text-[11px] font-medium">{rule.name || `${zh ? '规则' : 'Rule'} ${index + 1}`}</span><span className={`text-[10px] text-[var(--text-faint)] transition-transform ${rule.id === selectedID ? 'rotate-180' : ''}`}>⌄</span></div>
              <div className="mt-1 pl-3.5 text-[9px] text-[var(--text-faint)]">{itemCount(rule)} {zh ? '个匹配项' : 'entries'}</div>
            </button>)}
          </div>
          {settings.rules.length === 0 && <div className="whitespace-pre-line px-2 py-8 text-center text-[10px] leading-5 text-[var(--text-faint)]">{zh ? '暂无规则\n点击右上角新增' : 'No rules yet.\nUse Add rule above.'}</div>}
        </aside>

        <section className="p-5">
          {selected ? <>
            <div className="flex items-center gap-3 border-b border-[var(--border)] pb-5">
              <input autoCapitalize="none" autoCorrect="off" spellCheck={false} value={selected.name} onChange={(event) => updateRule({ name: event.target.value })} placeholder={zh ? '规则名称' : 'Rule name'} className="h-10 min-w-0 flex-1 rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] px-3 text-[12px] font-medium focus:border-[var(--accent)] focus:outline-none" />
              <label className="flex h-10 cursor-pointer items-center gap-2 rounded-lg border border-[var(--border)] px-3 text-[10px] text-[var(--text-dim)]"><input type="checkbox" checked={selected.enabled} onChange={(event) => updateRule({ enabled: event.target.checked })} className="accent-[var(--accent)]" />{zh ? '启用' : 'Enabled'}</label>
              <button disabled={busy} onClick={saveSelectedRule} className="h-10 rounded-lg bg-[var(--accent)] px-4 text-[10px] font-medium text-white hover:bg-[var(--accent-hover)] disabled:opacity-50">{busy ? (zh ? '保存中…' : 'Saving…') : (zh ? '保存规则' : 'Save rule')}</button>
              <button disabled={busy} onClick={removeRule} className="h-10 rounded-lg border border-red-500/25 px-3 text-[10px] text-red-400 hover:bg-red-500/10 disabled:opacity-50">{zh ? '删除' : 'Delete'}</button>
            </div>
            <div className="mt-5 space-y-3">
              <section className="rounded-xl border border-emerald-500/20 bg-[var(--bg-elevated)]/35 p-4">
                <h2 className="text-xs font-semibold text-emerald-400">{zh ? '白名单 · 直连' : 'Whitelist · Direct'}</h2>
                <div className="mt-4 grid grid-cols-2 gap-4"><RuleField label={zh ? '域名' : 'Domains'} hint={t('commaOrLineSeparated')} placeholder={'example.com\ndomain:example.org\nfull:api.example.net\nregexp:^.+\\.internal$'} value={selected.whitelistDomains} onChange={(value) => updateRule({ whitelistDomains: value })} /><RuleField label={zh ? 'IP 网段' : 'IP ranges'} hint={t('commaOrLineSeparated')} placeholder={'192.168.1.10\n10.0.0.0/8\n2001:db8::1\n2001:db8::/32'} value={selected.whitelistIPs} onChange={(value) => updateRule({ whitelistIPs: value })} /></div>
              </section>
              <section className="rounded-xl border border-red-500/20 bg-[var(--bg-elevated)]/35 p-4">
                <h2 className="text-xs font-semibold text-red-400">{zh ? '黑名单 · 阻断' : 'Blacklist · Block'}</h2>
                <div className="mt-4 grid grid-cols-2 gap-4"><RuleField label={zh ? '域名' : 'Domains'} hint={t('commaOrLineSeparated')} placeholder={'ads.example.com\ndomain:tracker.example\nfull:blocked.example.net\nregexp:^ad[0-9]+\\.example$'} value={selected.blacklistDomains} onChange={(value) => updateRule({ blacklistDomains: value })} /><RuleField label={zh ? 'IP 网段' : 'IP ranges'} hint={t('commaOrLineSeparated')} placeholder={'203.0.113.8\n203.0.113.0/24\n2001:db8:bad::1\n2001:db8:bad::/48'} value={selected.blacklistIPs} onChange={(value) => updateRule({ blacklistIPs: value })} /></div>
              </section>
              <section className="rounded-xl border border-violet-500/20 bg-[var(--bg-elevated)]/35 p-4">
                <h2 className="text-xs font-semibold text-violet-400">{t('proxyRules')}</h2>
                <div className="mt-4 grid grid-cols-2 gap-4"><RuleField label={zh ? '域名' : 'Domains'} hint={t('commaOrLineSeparated')} placeholder={'google.com\ndomain:github.com\nfull:api.example.net\nregexp:^.+\\.example$'} value={selected.proxyDomains} onChange={(value) => updateRule({ proxyDomains: value })} /><RuleField label={zh ? 'IP 网段' : 'IP ranges'} hint={t('commaOrLineSeparated')} placeholder={'8.8.8.8\n1.1.1.1\n203.0.113.0/24\n2001:db8::/32'} value={selected.proxyIPs} onChange={(value) => updateRule({ proxyIPs: value })} /></div>
              </section>
            </div>
          </> : <div className="flex h-full flex-col items-center justify-center text-center"><div className="text-sm font-medium">{settings.rules.length ? (zh ? '选择一条规则展开编辑' : 'Select a rule to expand it') : (zh ? '创建第一条出站规则' : 'Create your first outbound rule')}</div><div className="mt-2 text-[10px] text-[var(--text-faint)]">{settings.rules.length ? (zh ? '再次点击已展开的规则即可收起。' : 'Click the open rule again to collapse it.') : (zh ? '每条规则都包含独立的黑名单和白名单。' : 'Each rule has independent allow and block lists.')}</div>{settings.rules.length === 0 && <button onClick={addRule} className="mt-5 rounded-lg bg-[var(--accent)] px-4 py-2 text-[11px] text-white">{zh ? '新增规则' : 'Add rule'}</button>}</div>}
        </section>
      </div>

      {message && <div className="mt-4 text-[10px] text-[var(--text-dim)]">{message}</div>}
    </div>
  </main>
}
