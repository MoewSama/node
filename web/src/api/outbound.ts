import { request } from '@/util/request'

export const getOutbounds = () => request('/api/outbound')
export const saveOutbound = (data: any) => request('/api/outbound/save', {
    method: 'PUT',
    body: JSON.stringify(data)
})
export const deleteOutbound = (id: number) => request(`/api/outbound/${id}`, {
    method: 'DELETE'
})
export const toggleOutbound = (id: number) => request(`/api/outbound/${id}/toggle`, {
    method: 'PUT'
})
