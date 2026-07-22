import { request } from '@/util/request'

export const getUri = (data: { user: any; inbound: any }) =>
  request('/api/share/uri', {
    method: 'POST',
    body: JSON.stringify(data),
  })
