import { useEffect, useRef, useCallback } from 'react'

/**
 * usePolling — periodically invokes a callback at the specified interval.
 *
 * Features:
 *   - Auto-starts on mount, cleans up on unmount
 *   - Skips polling while a previous request is still in-flight
 *   - Supports immediate first call via `immediate: true`
 *   - Manual `refresh()` returns a promise that resolves when done
 *
 * Usage:
 *   const { refresh, isPolling } = usePolling(loadData, 3000)
 */
export function usePolling(
  fn: () => Promise<void>,
  intervalMs: number,
  options?: { immediate?: boolean }
) {
  const savedFn = useRef(fn)
  const pollingRef = useRef(false)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Always use the latest callback without restarting the interval
  savedFn.current = fn

  const refresh = useCallback(async () => {
    if (pollingRef.current) return // Skip if already in-flight
    pollingRef.current = true
    try {
      await savedFn.current()
    } finally {
      pollingRef.current = false
    }
  }, [])

  useEffect(() => {
    // Immediate first call
    if (options?.immediate !== false) {
      refresh()
    }

    // Start periodic polling
    timerRef.current = setInterval(() => {
      refresh()
    }, intervalMs)

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [intervalMs]) // eslint-disable-line react-hooks/exhaustive-deps

  return { refresh }
}
