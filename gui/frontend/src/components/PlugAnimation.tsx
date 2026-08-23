import React, { useEffect } from 'react'
import './PlugAnimation.css'

export type PlugAnimationMode = 'connect' | 'connected' | 'disconnect'

interface PlugAnimationProps {
  mode?: PlugAnimationMode
  onComplete?: () => void
}

export const PlugAnimation: React.FC<PlugAnimationProps> = ({
  mode = 'connect',
  onComplete,
}) => {
  useEffect(() => {
    if (mode === 'connected') return

    const duration = mode === 'connect' ? 1700 : 1400
    const timer = setTimeout(() => {
      if (onComplete) {
        onComplete()
      }
    }, duration)

    return () => clearTimeout(timer)
  }, [mode, onComplete])

  const branchLeftClass =
    mode === 'connect'
      ? 'm3-branch-left-connect'
      : mode === 'disconnect'
      ? 'm3-branch-left-disconnect'
      : 'm3-branch-left-connected'

  const branchRightClass =
    mode === 'connect'
      ? 'm3-branch-right-connect'
      : mode === 'disconnect'
      ? 'm3-branch-right-disconnect'
      : 'm3-branch-right-connected'

  // Top-left cable curves with wide clearance around settings icon and enters directly into strain relief at (199, 247)
  const topLeftPath =
    'M -40 -40 C -10 40, 20 110, 55 145 C 95 185, 155 203, 199 247'

  // Bottom-right cable sweeps wide (X >= 440) around the status text and enters directly into strain relief at (281, 329)
  const bottomRightPath =
    'M 540 660 C 470 540, 450 440, 435 390 C 410 335, 325 373, 281 329'

  return (
    <div className="plug-animation-overlay">
      <svg
        viewBox="0 0 480 600"
        width="480"
        height="600"
        style={{
          width: '100%',
          height: '100%',
          overflow: 'visible',
        }}
      >
        <defs>
          {/* Material 3 Gradients & Glows */}
          <radialGradient id="m3ContactGlow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#ffffff" stopOpacity="1" />
            <stop offset="40%" stopColor="#d8b4fe" stopOpacity="0.8" />
            <stop offset="70%" stopColor="#a855f7" stopOpacity="0.4" />
            <stop offset="100%" stopColor="#7e22ce" stopOpacity="0" />
          </radialGradient>

          <radialGradient id="m3BreakGlow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#ffffff" stopOpacity="1" />
            <stop offset="35%" stopColor="#c084fc" stopOpacity="0.7" />
            <stop offset="70%" stopColor="#64748b" stopOpacity="0.3" />
            <stop offset="100%" stopColor="#334155" stopOpacity="0" />
          </radialGradient>

          <linearGradient id="m3PlugSurface" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor="#2e273d" />
            <stop offset="100%" stopColor="#1a1526" />
          </linearGradient>

          <filter id="m3Elevation" x="-20%" y="-20%" width="140%" height="140%">
            <feDropShadow dx="0" dy="4" stdDeviation="6" floodColor="#000000" floodOpacity="0.5" />
            <feDropShadow dx="0" dy="0" stdDeviation="8" floodColor="#d8b4fe" floodOpacity="0.25" />
          </filter>
        </defs>

        {/* Top-Left Unified Branch (Cable seamlessly connected into Male Connector) */}
        <g className={branchLeftClass}>
          {/* Cable Track */}
          <path d={topLeftPath} fill="none" className="m3-cable-track" />
          {/* Energy Pulse Flow */}
          {mode !== 'disconnect' && (
            <path d={topLeftPath} fill="none" className="m3-cable-pulse" />
          )}

          {/* Male Connector Head (docked at 240, 288 at 45 deg) */}
          <g transform="translate(240, 288) rotate(45)">
            {/* Strain Relief Boot */}
            <rect
              x="-58"
              y="-7"
              width="14"
              height="14"
              rx="4"
              fill="#363045"
              stroke="#49454f"
              strokeWidth="1"
            />

            {/* Connector Body (Material 3 Capsule) */}
            <rect
              x="-44"
              y="-16"
              width="36"
              height="32"
              rx="10"
              fill="url(#m3PlugSurface)"
              stroke="#d8b4fe"
              strokeWidth="1.5"
              filter="url(#m3Elevation)"
            />

            {/* M3 Tonal Surface Pill */}
            <rect x="-35" y="-7" width="14" height="14" rx="7" fill="#4f378b" />
            <circle cx="-28" cy="0" r="3" fill="#d8b4fe" />

            {/* Front Collar */}
            <rect x="-8" y="-12" width="6" height="24" rx="2" fill="#475569" stroke="#94a3b8" strokeWidth="1" />

            {/* Male Connector Tip */}
            <rect
              x="-2"
              y="-8"
              width="10"
              height="16"
              rx="3"
              fill="#382e4d"
              stroke="#d8b4fe"
              strokeWidth="1.2"
            />
            {/* Gold Contact Pins */}
            <line x1="1" y1="-3.5" x2="6" y2="-3.5" stroke="#fde047" strokeWidth="1.5" strokeLinecap="round" />
            <line x1="1" y1="3.5" x2="6" y2="3.5" stroke="#fde047" strokeWidth="1.5" strokeLinecap="round" />
          </g>
        </g>

        {/* Bottom-Right Unified Branch (Cable seamlessly connected into Female Socket) */}
        <g className={branchRightClass}>
          {/* Cable Track */}
          <path d={bottomRightPath} fill="none" className="m3-cable-track" />
          {/* Energy Pulse Flow */}
          {mode !== 'disconnect' && (
            <path d={bottomRightPath} fill="none" className="m3-cable-pulse" />
          )}

          {/* Female Socket Head (docked at 240, 288 at 225 deg) */}
          <g transform="translate(240, 288) rotate(225)">
            {/* Strain Relief Boot */}
            <rect
              x="-58"
              y="-7"
              width="14"
              height="14"
              rx="4"
              fill="#363045"
              stroke="#49454f"
              strokeWidth="1"
            />

            {/* Socket Body (Material 3 Capsule) */}
            <rect
              x="-44"
              y="-16"
              width="36"
              height="32"
              rx="10"
              fill="url(#m3PlugSurface)"
              stroke="#d8b4fe"
              strokeWidth="1.5"
              filter="url(#m3Elevation)"
            />

            {/* M3 Tonal Surface Pill */}
            <rect x="-35" y="-7" width="14" height="14" rx="7" fill="#4f378b" />
            <circle cx="-28" cy="0" r="3" fill="#d8b4fe" />

            {/* Socket Mouth */}
            <rect
              x="-8"
              y="-13"
              width="8"
              height="26"
              rx="3"
              fill="#334155"
              stroke="#d8b4fe"
              strokeWidth="1.2"
            />
            {/* Receptacle Slot */}
            <rect x="-4" y="-9" width="10" height="18" rx="3" fill="#0f0b18" stroke="#49454f" strokeWidth="0.8" />
          </g>
        </g>

        {/* Dynamic Contact / Break Effects */}
        {mode === 'connect' && (
          <g>
            {/* M3 Ripples */}
            <circle cx="240" cy="288" r="40" fill="none" stroke="#d8b4fe" strokeWidth="2.5" className="m3-ripple-1" />
            <circle cx="240" cy="288" r="40" fill="none" stroke="#c084fc" strokeWidth="1.5" className="m3-ripple-2" />

            {/* Contact Glow */}
            <circle cx="240" cy="288" r="30" fill="url(#m3ContactGlow)" className="m3-contact-glow" />
            <circle cx="240" cy="288" r="7" fill="#ffffff" className="m3-contact-glow" />
          </g>
        )}

        {mode === 'disconnect' && (
          <g>
            {/* Disconnect Break Flash */}
            <circle cx="240" cy="288" r="28" fill="url(#m3BreakGlow)" className="m3-disconnect-break" />
            <circle cx="240" cy="288" r="36" fill="none" stroke="#c084fc" strokeWidth="1.5" className="m3-disconnect-break" />
          </g>
        )}
      </svg>
    </div>
  )
}
