import { useCallback, useEffect, useRef, useState } from 'react'

export interface AsyncState<T> {
  data: T | undefined
  loading: boolean
  error: Error | undefined
  reload: () => void
}

/**
 * Run an async loader whenever `deps` change. Stale results are discarded so a
 * fast project switch never renders the wrong data.
 */
export function useAsync<T>(loader: () => Promise<T>, deps: unknown[]): AsyncState<T> {
  const [data, setData] = useState<T | undefined>(undefined)
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<Error | undefined>(undefined)
  const [tick, setTick] = useState(0)
  const reqId = useRef(0)

  const reload = useCallback(() => setTick((t) => t + 1), [])

  useEffect(() => {
    const id = ++reqId.current
    setLoading(true)
    setError(undefined)
    loader()
      .then((res) => {
        if (id === reqId.current) {
          setData(res)
          setLoading(false)
        }
      })
      .catch((err: unknown) => {
        if (id === reqId.current) {
          setError(err instanceof Error ? err : new Error(String(err)))
          setLoading(false)
        }
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick])

  return { data, loading, error, reload }
}
