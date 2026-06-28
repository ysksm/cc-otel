import { createContext, useContext, type ReactNode } from 'react'
import { api, type Project } from '../lib/api'
import { useAsync } from './useAsync'

interface ProjectsContextValue {
  projects: Project[]
  loading: boolean
  error: Error | undefined
  reload: () => void
}

const ProjectsContext = createContext<ProjectsContextValue | undefined>(undefined)

export function ProjectsProvider({ children }: { children: ReactNode }) {
  const { data, loading, error, reload } = useAsync(() => api.listProjects(), [])
  const value: ProjectsContextValue = {
    projects: data?.items ?? [],
    loading,
    error,
    reload,
  }
  return <ProjectsContext.Provider value={value}>{children}</ProjectsContext.Provider>
}

export function useProjects(): ProjectsContextValue {
  const ctx = useContext(ProjectsContext)
  if (!ctx) throw new Error('useProjects must be used within ProjectsProvider')
  return ctx
}
