export function useApi() {
  const trigger = async (operation: string) => {
    try {
      const res = await fetch(`/api/${operation}`, { method: 'POST' })
      if (!res.ok) {
        if (res.status === 409) {
          console.warn('Operation busy')
          return
        }
        console.error(`Operation failed: ${res.statusText}`)
      }
    } catch (err) {
      console.error('API Error:', err)
    }
  }

  return { trigger }
}