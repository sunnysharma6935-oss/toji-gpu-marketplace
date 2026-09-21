interface LogoProps {
  size?: number
  showWordmark?: boolean
}

export function Logo({ size = 28, showWordmark = true }: LogoProps) {
  return (
    <div className="flex items-center gap-2.5">
      <svg width={size} height={size} viewBox="0 0 180 180" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M30 30 H98 V50 H50 V98 H30 Z" fill="#F6F3EE" />
        <path d="M150 150 L82 150 L82 130 L130 130 L130 82 L150 82 Z" fill="#FF5A1F" />
      </svg>
      {showWordmark && (
        <span className="font-sans text-lg font-bold tracking-tight">
          TOJI
        </span>
      )}
    </div>
  )
}
