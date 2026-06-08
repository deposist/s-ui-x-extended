import axios from 'axios'
import { clearCSRFToken, getCSRFToken } from '@/store/csrf'
import { getBaseUrl } from '@/plugins/base-url'

const api = axios.create({
    baseURL: getBaseUrl(),
    headers: {
        common: {
            'X-Requested-With': 'XMLHttpRequest',
        },
        post: {
            'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
        },
    },
})

const pendingRequests = new Map<string, AbortController>()
const DUPLICATE_ABORT_REASON = 'Duplicate request cancelled'

const isDedupeMethod = (method?: string) => {
    const m = (method ?? 'get').toLowerCase()
    // Only deduplicate idempotent reads. Mutating requests with identical
    // URLs but different bodies would otherwise cancel each other.
    return m === 'get' || m === 'head' || m === 'options'
}

const requestKey = (config: any) => {
    const params = config.params ? JSON.stringify(config.params) : ''
    return `${config.method}:${config.url}:${params}`
}

const normalizeURL = (url?: string) => (url ?? '').replace(/^\.\//, '').replace(/^\//, '')

const needsCSRFToken = (method?: string, url?: string) => {
    const m = (method ?? 'get').toLowerCase()
    if (!['post', 'put', 'patch', 'delete'].includes(m)) {
        return false
    }
    const normalized = normalizeURL(url)
    return normalized.startsWith('api/') && normalized !== 'api/login'
}

api.interceptors.request.use(
    async (config) => {
        if (isDedupeMethod(config.method)) {
            const key = requestKey(config)
            if (pendingRequests.has(key)) {
                pendingRequests.get(key)?.abort(DUPLICATE_ABORT_REASON)
            }
            const controller = new AbortController()
            config.signal = controller.signal
            pendingRequests.set(key, controller)
        }

        if (config.data instanceof FormData) {
            delete config.headers['Content-Type']
        }
        if (needsCSRFToken(config.method, config.url)) {
            config.headers['X-CSRF-Token'] = await getCSRFToken()
        }
        return config
    },
    (error) => Promise.reject(error),
)

api.interceptors.response.use(
    (response) => {
        if (isDedupeMethod(response.config.method)) {
            pendingRequests.delete(requestKey(response.config))
        }
        return response
    },
    (error) => {
        if (axios.isCancel(error) || error.code === 'ERR_CANCELED') {
            console.warn(error.message)
        } else if (error.config && isDedupeMethod(error.config.method)) {
            pendingRequests.delete(requestKey(error.config))
        }
        if (error.response?.status === 403 && error.response?.data?.msg === 'Invalid CSRF token') {
            clearCSRFToken()
        }
        return Promise.reject(error)
    }
)

export default api
