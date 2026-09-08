import React from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

function App() {
  return (
    <main className="shell">
      <p className="eyebrow">MISTRAL / PRE-ALPHA</p>
      <h1>Idle MMORPG, server-authoritative by design.</h1>
      <p>
        The first vertical slice is establishing deterministic dungeon time resolution and versioned Data-Driven content before gameplay formulas are expanded.
      </p>
    </main>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
