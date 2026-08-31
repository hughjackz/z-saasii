import axios from 'axios'
import router from '@/router/index.js'

const http = axios.create({
  baseURL: '/api',
  timeout: 15000
})

// Attach JWT from memory (never localStorage)
let _token = null
let _onUnauthorized = null
export function setToken(t) { _token = t }
export function clearToken() { _token = null }
export function getToken() { return _token }
export function onUnauthorized(cb) { _onUnauthorized = cb }

http.interceptors.request.use(cfg => {
  if (_token) cfg.headers.Authorization = `Bearer ${_token}`
  return cfg
})

http.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      clearToken()
      _onUnauthorized?.()
      router.push('/login')
    }
    return Promise.reject(err.response?.data || err)
  }
)
 
// Auth
export const auth = {
  login: (username, password) => http.post('/auth/login', { username, password }),
  logout: () => http.post('/auth/logout'),
  me: () => http.get('/auth/me')
}

// Devices
export const devices = {
  list: (params) => http.get('/devices', { params }),
  get: id => http.get(`/devices/${id}`),
  create: data => http.post('/devices', data),
  update: (id, data) => http.put(`/devices/${id}`, data),
  remove: id => http.delete(`/devices/${id}`)
}

// OCPP Configuration
export const ocppConfig = {
  getAll: deviceId => http.get(`/ocpp/${deviceId}/configuration`),
  getKeys: (deviceId, keys) => http.post(`/ocpp/${deviceId}/configuration/get`, { keys }),
  setKeys: (deviceId, configs) => http.post(`/ocpp/${deviceId}/configuration/set`, { configs })
}

// OCPP Transactions
export const transactions = {
  list: (deviceId, params) => http.get(`/ocpp/${deviceId}/transactions`, { params }),
  active: deviceId => http.get(`/ocpp/${deviceId}/transactions/active`),
  remoteStart: (deviceId, data) => http.post(`/ocpp/${deviceId}/remote-start`, data),
  remoteStop: (deviceId, data) => http.post(`/ocpp/${deviceId}/remote-stop`, data)
}

// OCPP Maintenance
export const maintenance = {
  hardReset: deviceId => http.post(`/ocpp/${deviceId}/reset`, { type: 'Hard' }),
  softReset: deviceId => http.post(`/ocpp/${deviceId}/reset`, { type: 'Soft' }),
  getLog: (deviceId, data) => http.post(`/ocpp/${deviceId}/get-log`, data),
  firmwareUpdate: (deviceId, formData) =>
    http.post(`/ocpp/${deviceId}/firmware-update`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
}

// OCPP PnC (Plug & Charge)
export const pnc = {
  getInstalledCerts: (deviceId, type) =>
    http.post(`/ocpp/${deviceId}/get-installed-certificate-ids`, { certificateType: type }),
  deleteCert: (deviceId, certName) => http.post(`/ocpp/${deviceId}/delete-certificate`, { certName }),
  installCert: (deviceId, certNames, certType) =>
    http.post(`/ocpp/${deviceId}/install-certificate`, { certNames, certType }),
  triggerCsr: deviceId => http.post(`/ocpp/${deviceId}/trigger-message`, { requestedMessage: 'SignCertificate' }),
  signCertificate: (deviceId, data) => http.post(`/ocpp/${deviceId}/sign-certificate`, data),
  certificateSigned: (deviceId, data) => http.post(`/ocpp/${deviceId}/certificate-signed`, data),
  contractCertGroup: (deviceId, certs) => http.post(`/ocpp/${deviceId}/contract-cert-group`, { certs })
}

// Users
export const users = {
  list: (params) => http.get('/users', { params }),
  listCPOPs: () => http.get('/users/cpops'),
  listCPOMs: (parentId) => http.get('/users/cpoms', { params: { parent: parentId } }),
  get: id => http.get(`/users/${id}`),
  create: data => http.post('/users', data),
  update: (id, data) => http.put(`/users/${id}`, data),
  remove: id => http.delete(`/users/${id}`),
  updatePermissions: (id, permissions) => http.put(`/users/${id}/permissions`, { permissions })
}

// Certificates library
export const certs = {
  list: (params) => http.get('/certificates', { params }),
  upload: (formData) =>
    http.post('/certificates', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }),
  getContent: (id) => http.get(`/certificates/${id}/content`),
  rename: (id, name) => http.put(`/certificates/${id}`, { name }),
  remove: id => http.delete(`/certificates/${id}`)
}

// ID Tags
export const idtags = {
  list: (params) => http.get('/idtags', { params }),
  create: data => http.post('/idtags', data),
  update: (id, data) => http.put(`/idtags/${id}`, data),
  remove: id => http.delete(`/idtags/${id}`)
}

// VDV261
export const vdv = {
  listProfiles: () => http.get('/vdv/profiles'),
  createProfile: (data) => http.post('/vdv/profiles', data),
  updateProfile: (id, data) => http.put(`/vdv/profiles/${id}`, data),
  deleteProfile: (id) => http.delete(`/vdv/profiles/${id}`),
  listCarInfos: () => http.get('/vdv/carinfos'),
  createCarInfo: (data) => http.post('/vdv/carinfos', data),
  updateCarInfo: (id, data) => http.put(`/vdv/carinfos/${id}`, data),
  deleteCarInfo: (id) => http.delete(`/vdv/carinfos/${id}`),
  getSettings: () => http.get('/vdv/settings'),
  updateSettings: (data) => http.post('/vdv/settings', data),
  restartService: () => http.post('/vdv/settings/restart'),
  uploadCert: (formData) => http.post('/vdv/settings/upload-cert', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
}

// Events
export const events = {
  queryLogs: (params) => http.get('/events/logs', { params })
}

// Charging Profiles
export const profiles = {
  list: () => http.get('/profiles'),
  upload: (formData) =>
    http.post('/profiles', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }),
  rename: (id, name) => http.put(`/profiles/${id}`, { name }),
  remove: id => http.delete(`/profiles/${id}`)
}
