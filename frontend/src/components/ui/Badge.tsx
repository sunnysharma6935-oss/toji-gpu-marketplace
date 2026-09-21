interface BadgeProps {
  tone?: 'success' | 'muted' | 'accent' | 'warn'
  children: React.ReactNode
}

const toneClasses: Record<NonNullable<BadgeProps['tone']>, string> = {
  success: 'text-success',
  muted: 'text-muted-dim',
  accent: 'text-accent',
  warn: 'text-warn',
}

export function Badge({ tone = 'muted', children }: BadgeProps) {
  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${toneClasses[tone]}`}>
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {children}
    </span>
  )
}
