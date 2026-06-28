import { Link, Navigate } from 'react-router-dom'
import { useProjects } from '../hooks/useProjects'
import { EmptyState, ErrorBox, Spinner } from '../components/common'

/**
 * Root route. Redirects to the first project's dashboard, or shows an onboarding
 * empty state when there are no projects.
 */
export function HomePage() {
  const { projects, loading, error } = useProjects()

  if (loading) {
    return (
      <div className="page">
        <Spinner label="Loading projects…" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="page">
        <ErrorBox error={error} />
      </div>
    )
  }

  if (projects.length > 0) {
    return <Navigate to={`/projects/${projects[0].slug}/dashboard`} replace />
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Welcome to cc-otel</h1>
      </div>
      <EmptyState title="No projects yet">
        <p>
          cc-otel is a local LLM / agent observability tool. To see data here, send OpenTelemetry
          traces to the ingest endpoint:
        </p>
        <pre className="code-block">{`POST http://localhost:8080/v1/traces
Authorization: Bearer <your-api-key>`}</pre>
        <p>
          A project is created automatically from your first trace. Once you have one, create an API
          key under <strong>Settings</strong>.
        </p>
        <p className="dim">
          Tip: projects appear in the selector in the sidebar as soon as traces arrive.
        </p>
        <p>
          <Link className="link" to="/">
            Refresh
          </Link>{' '}
          after sending traces.
        </p>
      </EmptyState>
    </div>
  )
}
