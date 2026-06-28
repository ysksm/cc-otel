import { NavLink, useNavigate, useParams } from 'react-router-dom'
import { useProjects } from '../hooks/useProjects'

export function Sidebar() {
  const { projects } = useProjects()
  const params = useParams()
  const navigate = useNavigate()
  const currentSlug = params.slug ?? ''

  const onSelectProject = (slug: string) => {
    if (slug) navigate(`/projects/${slug}/traces`)
  }

  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">◆</span>
        <span className="brand-name">cc-otel</span>
      </div>

      <div className="project-selector">
        <label className="field-label">Project</label>
        <select
          value={currentSlug}
          onChange={(e) => onSelectProject(e.target.value)}
          disabled={projects.length === 0}
        >
          {projects.length === 0 && <option value="">No projects</option>}
          {projects.map((p) => (
            <option key={p.id} value={p.slug}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      <nav className="nav">
        <NavItem
          to={`/projects/${currentSlug}/dashboard`}
          label="Dashboard"
          disabled={!currentSlug}
        />
        <NavItem to={`/projects/${currentSlug}/traces`} label="Traces" disabled={!currentSlug} />
        <NavItem
          to={`/projects/${currentSlug}/sessions`}
          label="Sessions"
          disabled={!currentSlug}
        />
        <NavItem to={`/projects/${currentSlug}/users`} label="Users" disabled={!currentSlug} />
        <NavItem to={`/projects/${currentSlug}/tools`} label="Tools" disabled={!currentSlug} />
        <NavItem
          to={`/projects/${currentSlug}/signals`}
          label="Signals"
          disabled={!currentSlug}
        />
        <NavItem
          to={`/projects/${currentSlug}/monitors`}
          label="Monitors"
          disabled={!currentSlug}
        />
        <NavItem
          to={`/projects/${currentSlug}/settings`}
          label="Settings"
          disabled={!currentSlug}
        />
      </nav>

      <div className="sidebar-footer">
        <span className="dim">local observability</span>
      </div>
    </aside>
  )
}

function NavItem({ to, label, disabled }: { to: string; label: string; disabled?: boolean }) {
  if (disabled) {
    return <span className="nav-link nav-disabled">{label}</span>
  }
  return (
    <NavLink to={to} className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}>
      {label}
    </NavLink>
  )
}
