import axios from 'axios'
import api from './api'
import { i18n } from '@/locales'
import router from '@/router'
import { push } from 'notivue'
import { clearCSRFToken } from '@/store/csrf'

let invalidLoginHandled = false
let loginSuccessTimestamp = 0

export const markLoginSuccess = () => {
  loginSuccessTimestamp = Date.now()
  invalidLoginHandled = false
}

export interface Msg {
  success: boolean
  msg: string
  obj: any | null
}

function _handleMsg(msg: any): void {
  if (!isMsg(msg)) {
    return
  }
  if(msg.msg){
    if (!msg.success && msg.msg == "Invalid login") {
      // After a successful login, stale cached assets can still send
      // unauthenticated requests. If we get "Invalid login" within 10s
      // of a successful login, force a hard reload to fetch fresh
      // index.html instead of looping back to /login.
      if (loginSuccessTimestamp > 0 && Date.now() - loginSuccessTimestamp < 10000) {
        loginSuccessTimestamp = 0
        // A plain reload() may be served from the browser's bfcache or HTTP cache,
        // so force a real navigation to a cache-busting URL.
        const url = new URL(window.location.href)
        url.searchParams.set('_', String(Date.now()))
        window.location.replace(url.toString())
        return
      }
      if (!invalidLoginHandled) {
        invalidLoginHandled = true
        // Suppress the error notification when already on the login
        // page — getting "Invalid login" while not authenticated is
        // expected, not an error worth surfacing.
        if (router.currentRoute.value.path !== '/login') {
          push.error({
            title: i18n.global.t('invalidLogin'),
          })
        }
        localLogout()
      }
      return
    }
    if (msg.success) {
      push.success({
        message: i18n.global.t('success') + ": " + i18n.global.t('actions.' + msg.msg),
      })
    } else {
      push.error({
        title: i18n.global.t('failed'),
        message: msg.msg
      })
    }
  }
}

export const localLogout = () => {
  clearCSRFToken()
  router.push('/login')
}

export const resetInvalidLoginHandling = () => {
  invalidLoginHandled = false
}

export const logout = async () => {
  // POST so the request carries the CSRF token (a GET logout was forgeable
  // cross-site). Clear the local token only AFTER the request, otherwise the
  // CSRF interceptor would have nothing to attach.
  const response = await HttpUtils.post('api/logout', null)
  clearCSRFToken()
  if(response.success){
    router.push('/login')
  }
}

function _describePayload(data: unknown): string {
  if (typeof data === 'object' && data !== null) {
    for (const key of ['message', 'error', 'detail'] as const) {
      const value = Object.hasOwn(data, key) ? (data as Record<string, unknown>)[key] : undefined
      if (typeof value === 'string' && value !== '') return value
    }

    try {
      return JSON.stringify(data)
    } catch {
      return String(data)
    }
  }

  return String(data)
}

function _respToMsg(resp: any): Msg {
  const data = resp.data
  if (data == null) {
    return { success: true, msg: "", obj: null }
  }
  if (isMsg(data)) {
    return { success: data.success, msg: data.msg, obj: data.obj ?? null }
  }
  return { success: false, msg: _describePayload(data), obj: null }
}

function _errorToMsg(error: any): Msg {
  if (axios.isCancel(error) || error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError') {
    return { success: false, msg: "", obj: null }
  }
  if (error?.response?.data) {
    return _respToMsg(error.response)
  }
  if (error?.message === 'Invalid login') {
    return { success: false, msg: 'Invalid login', obj: null }
  }
  return { success: false, msg: error.toString(), obj: null }
}

function isMsg(obj: unknown): obj is Msg {
  if (obj === null || typeof obj !== 'object') return false
  return Object.hasOwn(obj, 'success') && Object.hasOwn(obj, 'msg') && Object.hasOwn(obj, 'obj')
}
  
const HttpUtils = {
  async get(url: string, data: object = {}, options: any = {}): Promise<Msg> {
    let msg: Msg
    try {
        const resp = await api.get(url, { params: data, ...options })
        msg = _respToMsg(resp)
    } catch (e: any) {
        msg = _errorToMsg(e)
    }
    _handleMsg(msg)
    return msg
  },
  async post(url: string, data: object | null, options: any = undefined): Promise<Msg> {
    let msg: Msg
    try {
        const resp = await api.post(url, data, options)
        msg = _respToMsg(resp)
    } catch (e: any) {
        msg = _errorToMsg(e)
    }
    _handleMsg(msg)
    return msg
  },
}

export default HttpUtils
