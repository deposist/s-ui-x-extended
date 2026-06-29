import { InTypes } from '@/types/inbounds'

export type TlsTemplateKind = 'tls' | 'reality'

const inboundRealityTemplateTypes = new Set<string>([
  InTypes.VLESS,
  InTypes.Trojan,
])

export function tlsTemplateKind(tlsConfig: any): TlsTemplateKind {
  return tlsConfig?.server?.reality != null ? 'reality' : 'tls'
}

export function inboundAllowedTlsTemplateKinds(inboundType: string): TlsTemplateKind[] {
  return inboundRealityTemplateTypes.has(inboundType) ? ['tls', 'reality'] : ['tls']
}

export function isInboundTlsTemplateCompatible(inboundType: string, tlsConfig: any): boolean {
  return inboundAllowedTlsTemplateKinds(inboundType).includes(tlsTemplateKind(tlsConfig))
}
