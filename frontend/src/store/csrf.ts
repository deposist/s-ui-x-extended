import axios from 'axios'
import { getBaseUrl } from '@/plugins/base-url'

let csrfToken: string | null = null
let csrfTokenPromise: Promise<string> | null = null
let csrfTokenGeneration = 0

export const getCSRFToken = async () => {
  if (csrfToken) {
    return csrfToken
  }
  if (!csrfTokenPromise) {
    const generation = csrfTokenGeneration
    csrfTokenPromise = axios.get('api/csrf', {
      baseURL: getBaseUrl(),
      headers: {
        'X-Requested-With': 'XMLHttpRequest',
      },
    }).then((response) => {
      if (response.data?.success === false && typeof response.data?.msg === 'string' && response.data.msg.length > 0) {
        throw new Error(response.data.msg)
      }
      const token = response.data?.obj?.token
      if (typeof token !== 'string' || token.length === 0) {
        throw new Error('CSRF token was not returned')
      }
      if (generation === csrfTokenGeneration) {
        csrfToken = token
      }
      return token
    }).finally(() => {
      if (generation === csrfTokenGeneration) {
        csrfTokenPromise = null
      }
    })
  }
  return csrfTokenPromise
}

export const clearCSRFToken = () => {
  csrfTokenGeneration++
  csrfToken = null
  csrfTokenPromise = null
}
