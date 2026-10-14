export function panelAddress(config: { Domain?: string; Port?: number; Path?: string }, login = false) {
  const url = new URL(window.location.origin)
  url.protocol = config.Domain ? 'https:' : 'http:'
  if (config.Domain) url.hostname = config.Domain
  url.port = String(config.Port || window.location.port || (config.Domain ? 443 : 80))
  url.pathname = `${config.Path || ''}${login ? '/login' : ''}`
  return url.toString().replace(/\/$/, '')
}
