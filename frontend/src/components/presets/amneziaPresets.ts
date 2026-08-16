// AmneziaWG obfuscation presets (junk parameters only).
//
// Values follow the amneziawg-installer ADVANCED.md preset table
// (github.com/bivlked/amneziawg-installer/blob/main/ADVANCED.md):
//   default (Balanced): Jc 3-6, Jmin 40-89, Jmax = Jmin+50..250
//   mobile:             Jc 3 (fixed), Jmin 30-50, Jmax = Jmin+20..80
// A preset fills Jc/Jmin/Jmax only. H1-H4 are never touched: they come from
// the server-side crypto/rand generator (stage 1) and stay unique per
// endpoint regardless of the junk profile.

export type AmneziaPresetId = 'balanced' | 'mobile' | 'custom'

export interface AmneziaJunkParams {
  jc: number
  jmin: number
  jmax: number
}

export interface AmneziaPreset {
  id: AmneziaPresetId
  titleKey: string
  descriptionKey: string
  // Absent for 'custom': selecting it changes nothing and keeps manual values.
  junk?: AmneziaJunkParams
}

export const amneziaPresetCatalog: AmneziaPreset[] = [
  {
    id: 'balanced',
    titleKey: 'types.amnezia.presets.balanced.title',
    descriptionKey: 'types.amnezia.presets.balanced.description',
    // Midpoint of the ADVANCED.md default distribution (Jc 3-6, Jmin 40-89,
    // Jmax = Jmin + 50..250). Randomize regenerates these within the same
    // distribution server-side when this preset is active.
    junk: { jc: 4, jmin: 40, jmax: 90 },
  },
  {
    id: 'mobile',
    titleKey: 'types.amnezia.presets.mobile.title',
    descriptionKey: 'types.amnezia.presets.mobile.description',
    // ADVANCED.md mobile preset (Yota/Tele2/Megafon empirics): fixed Jc=3
    // with a narrow junk size window so carrier DPI heuristics on packet
    // bursts are not triggered.
    junk: { jc: 3, jmin: 40, jmax: 70 },
  },
  {
    id: 'custom',
    titleKey: 'types.amnezia.presets.custom.title',
    descriptionKey: 'types.amnezia.presets.custom.description',
  },
]

export function amneziaPresetById(id: AmneziaPresetId): AmneziaPreset {
  const preset = amneziaPresetCatalog.find(item => item.id === id)
  if (!preset) throw new Error(`unknown amnezia preset: ${id}`)
  return preset
}

// Applies a preset to the amnezia options object in place. Only junk
// parameters are written; header and padding fields are left untouched.
// 'custom' is a no-op by design.
export function applyAmneziaPreset(amnezia: Record<string, unknown>, id: AmneziaPresetId): void {
  const preset = amneziaPresetById(id)
  if (!preset.junk) return
  amnezia.jc = preset.junk.jc
  amnezia.jmin = preset.junk.jmin
  amnezia.jmax = preset.junk.jmax
}

// Detects which preset the current junk values correspond to. The preset is
// deliberately not persisted (extra keys in the amnezia options would leak
// into the sing-box endpoint config), so the selector state is derived.
export function detectAmneziaPreset(amnezia: Record<string, unknown> | undefined | null): AmneziaPresetId {
  if (!amnezia) return 'custom'
  for (const preset of amneziaPresetCatalog) {
    if (!preset.junk) continue
    if (amnezia.jc === preset.junk.jc && amnezia.jmin === preset.junk.jmin && amnezia.jmax === preset.junk.jmax) {
      return preset.id
    }
  }
  return 'custom'
}

// AWG 3.0 timing defaults seeded into a freshly enabled amnezia profile.
// Values follow the ranges the official Amnezia client generates for new 3.0
// configs (amneziawg-go UAPI ranges, seconds; max_handshake_attempts is a
// count). content_padding_addition 0 means no extra padding; the wireguard
// server fills the junk/header fields separately. HeaderProtectionKey is not
// defaulted: it is a server-side key that must match the client and requires
// S1-S4 >= 12.
export const amneziaTimingDefaults: Record<string, string | number> = {
  content_padding_addition: 0,
  rekey_after_time: '120-180',
  rekey_timeout: '1-5',
  reject_after_time: '90-120',
  keepalive_timeout: '5-10',
  max_handshake_attempts: '20-30',
}

// Applies the AWG 3.0 timing defaults to the amnezia options object in place,
// leaving any values the operator already set untouched.
export function applyAmneziaTimingDefaults(amnezia: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(amneziaTimingDefaults)) {
    if (amnezia[key] === undefined || amnezia[key] === null) {
      amnezia[key] = value
    }
  }
}

// One curated "recommended" profile: balanced junk midpoints, padding values
// that keep the four padded packet sizes distinct and stay >= 12 so header
// protection can be enabled later without touching S again, the stock init
// packet prefix, and fixed vanilla-WireGuard timings (ordered so a session is
// always rekeyed before it can be rejected).
export const amneziaRecommendedParams: Record<string, unknown> = {
  jc: 4,
  jmin: 40,
  jmax: 90,
  s1: 15,
  s2: 18,
  s3: 16,
  s4: 20,
  i1: '<b 0x01020304><r 8>',
  content_padding_addition: 0,
  rekey_after_time: 120,
  rekey_timeout: 5,
  reject_after_time: 180,
  keepalive_timeout: 10,
  max_handshake_attempts: 18,
}

// Applies the recommended profile in place. Header fields (H1-H4) are left
// alone: they are per-endpoint random by design and must stay unique.
export function applyAmneziaRecommended(amnezia: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(amneziaRecommendedParams)) {
    amnezia[key] = value
  }
}
