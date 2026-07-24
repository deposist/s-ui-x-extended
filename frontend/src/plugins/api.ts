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
    (response) => response,
    (error) => {
        if (axios.isCancel(error) || error.code === 'ERR_CANCELED') {
            console.warn(error.message)
        }
        if (error.response?.status === 403 && error.response?.data?.msg === 'Invalid CSRF token') {
            clearCSRFToken()
        }
        return Promise.reject(error)
    }
)

export default api
