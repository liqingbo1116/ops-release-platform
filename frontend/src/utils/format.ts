export function formatDateTime(value: string): string {
  return value.replace('T', ' ').replace('+08:00', '').slice(0, 16)
}

export function joinCapabilities(value: string[]): string {
  return value.join(' / ')
}
