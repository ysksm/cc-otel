import { Navigate, Route, Routes } from 'react-router-dom'
import { ProjectsProvider } from './hooks/useProjects'
import { Layout } from './components/Layout'
import { HomePage } from './pages/HomePage'
import { DashboardPage } from './pages/DashboardPage'
import { TracesPage } from './pages/TracesPage'
import { SessionsPage } from './pages/SessionsPage'
import { UsersPage } from './pages/UsersPage'
import { ToolsPage } from './pages/ToolsPage'
import { ToolDetailPage } from './pages/ToolDetailPage'
import { SettingsPage } from './pages/SettingsPage'

export function App() {
  return (
    <ProjectsProvider>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/projects/:slug/dashboard" element={<DashboardPage />} />
          <Route path="/projects/:slug/traces" element={<TracesPage />} />
          <Route path="/projects/:slug/sessions" element={<SessionsPage />} />
          <Route path="/projects/:slug/users" element={<UsersPage />} />
          <Route path="/projects/:slug/tools" element={<ToolsPage />} />
          <Route path="/projects/:slug/tools/:toolName" element={<ToolDetailPage />} />
          <Route path="/projects/:slug/settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </ProjectsProvider>
  )
}
