import { request, withPanelPath } from '@/util/request'

export interface TaskStatus {
  logs: Array<{ time: string; level: string; message: string }>
  next: number
  done: boolean
  error: string
}

export const getTaskStatus = (id: string, after = 0): Promise<TaskStatus> =>
  request(`/api/task/${encodeURIComponent(id)}/status?after=${after}`, { signal: AbortSignal.timeout(10000) })

export const task = (id: string) => {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('token') ?? ''
  return new WebSocket(
    `${protocol}//${location.host}${withPanelPath(`/api/task/${id}`)}?token=${token}`,
  )
}
