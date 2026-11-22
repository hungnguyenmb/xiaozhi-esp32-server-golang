import axios from 'axios'
import { ElMessage } from 'element-plus'
import { localizeApiFeedback } from '../locales/server'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    localizeApiFeedback(response.data, response.config?.url)
    return response
  },
  (error) => {
    localizeApiFeedback(error.response?.data, error.config?.url)
    const silentError = error.config?.silentError === true
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    } else if (!silentError) {
      ElMessage.error(error.response?.data?.error || 'Yêu cầu không thành công')
    }
    return Promise.reject(error)
  }
)

export default api
