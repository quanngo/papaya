import { render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import App from '../App'
import { backend } from './server'

function LocationProbe() {
  const location = useLocation()
  return <div data-testid="location">{location.pathname + location.search}</div>
}

/**
 * Renders the whole app (router + auth) at `route`.
 * @param {string} route
 * @param {{ as?: string }} [options]  sign in as this seeded email first
 */
export function renderApp(route, { as } = {}) {
  if (as) backend.user = as
  const user = userEvent.setup()
  const utils = render(
    <MemoryRouter initialEntries={[route]}>
      <App />
      <LocationProbe />
    </MemoryRouter>,
  )
  return { user, ...utils }
}
