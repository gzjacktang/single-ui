import { withPanelPath } from '@/util/request'

export const task = (id: string) => {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('token') ?? ''
  return new WebSocket(
    `${protocol}//${location.host}${withPanelPath(`/api/task/${id}`)}?token=${token}`,
  )
}
