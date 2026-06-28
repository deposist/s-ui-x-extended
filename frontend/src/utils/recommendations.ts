export type RecommendationPath = string | Array<string | number>
export type RecommendationMode = 'create' | 'edit' | 'both' | (string & {})

export interface RecommendationContext<TModel = unknown> {
  mode?: RecommendationMode
  model: TModel
  type?: string
  unavailableTypes?: string[]
  [key: string]: unknown
}

export interface RecommendationSpec<TModel = unknown> {
  id?: string
  label: string
  description?: string
  path: RecommendationPath
  value: unknown | ((context: RecommendationContext<TModel>) => unknown)
  mode?: RecommendationMode
  onlyIfEmpty?: boolean
  when?: (context: RecommendationContext<TModel>) => boolean
}

export interface ResolvedRecommendation<TModel = unknown> extends RecommendationSpec<TModel> {
  currentValue: unknown
  recommendedValue: unknown
  applicable: boolean
}

export function pathToArray(path: RecommendationPath): Array<string | number> {
  if (Array.isArray(path)) return path
  if (path.length === 0) return []

  return path.split('.').filter(Boolean).map((part) => {
    const index = Number(part)
    return Number.isInteger(index) && String(index) === part ? index : part
  })
}

export function getByPath<T = unknown>(target: unknown, path: RecommendationPath): T | undefined {
  return pathToArray(path).reduce<unknown>((current, part) => {
    if (current == null) return undefined
    return (current as Record<string | number, unknown>)[part]
  }, target) as T | undefined
}

export function deepClone<T>(value: T): T {
  if (value == null || typeof value !== 'object') return value

  try {
    if (typeof structuredClone === 'function') return structuredClone(value)
  } catch {
    // Vue may wrap recommendation specs from component data() in reactive proxies.
    // structuredClone cannot clone those proxies, while JSON cloning is enough for
    // recommendation payloads because they are plain config values.
  }

  return JSON.parse(JSON.stringify(value)) as T
}

export function isEmptyRecommendationValue(value: unknown): boolean {
  if (value == null) return true
  if (typeof value === 'string') return value.length === 0
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.keys(value as object).length === 0
  return false
}

export function setByPath<T extends object>(
  target: T,
  path: RecommendationPath,
  value: unknown,
  options: { cloneValue?: boolean; createParents?: boolean } = {},
): T {
  const parts = pathToArray(path)
  const { cloneValue = true, createParents = true } = options
  if (parts.length === 0) return target

  let current = target as Record<string | number, unknown>
  for (let index = 0; index < parts.length - 1; index += 1) {
    const part = parts[index]
    const nextPart = parts[index + 1]
    const next = current[part]

    if (next == null || typeof next !== 'object') {
      if (!createParents) return target
      current[part] = typeof nextPart === 'number' ? [] : {}
    }

    current = current[part] as Record<string | number, unknown>
  }

  current[parts[parts.length - 1]] = cloneValue ? deepClone(value) : value
  return target
}

function modeMatches<TModel>(spec: RecommendationSpec<TModel>, context: RecommendationContext<TModel>): boolean {
  if (spec.mode == null || spec.mode === 'both' || context.mode == null) return true
  return spec.mode === context.mode
}

export function resolveRecommendationValue<TModel>(
  spec: RecommendationSpec<TModel>,
  context: RecommendationContext<TModel>,
): unknown {
  return typeof spec.value === 'function'
    ? (spec.value as (context: RecommendationContext<TModel>) => unknown)(context)
    : spec.value
}

export function isRecommendationVisible<TModel>(
  spec: RecommendationSpec<TModel>,
  context: RecommendationContext<TModel>,
): boolean {
  return modeMatches(spec, context) && (spec.when?.(context) ?? true)
}

export function resolveRecommendations<TModel>(
  specs: RecommendationSpec<TModel>[],
  context: RecommendationContext<TModel>,
): ResolvedRecommendation<TModel>[] {
  return specs
    .filter((spec) => isRecommendationVisible(spec, context))
    .map((spec) => {
      const currentValue = getByPath(context.model, spec.path)
      const onlyIfEmpty = spec.onlyIfEmpty ?? true

      return {
        ...spec,
        currentValue,
        recommendedValue: deepClone(resolveRecommendationValue(spec, context)),
        applicable: !onlyIfEmpty || isEmptyRecommendationValue(currentValue),
      }
    })
}

export function applyRecommendation<TModel extends object>(
  model: TModel,
  spec: RecommendationSpec<TModel>,
  context: RecommendationContext<TModel>,
  options: { force?: boolean } = {},
): TModel {
  const currentValue = getByPath(model, spec.path)
  const onlyIfEmpty = spec.onlyIfEmpty ?? true

  if (!isRecommendationVisible(spec, context)) return model
  if (!options.force && onlyIfEmpty && !isEmptyRecommendationValue(currentValue)) return model

  return setByPath(model, spec.path, resolveRecommendationValue(spec, context), { cloneValue: true })
}

export function applyRecommendations<TModel extends object>(
  model: TModel,
  specs: RecommendationSpec<TModel>[],
  context: RecommendationContext<TModel>,
  options: { force?: boolean } = {},
): TModel {
  for (const spec of specs) applyRecommendation(model, spec, context, options)
  return model
}
