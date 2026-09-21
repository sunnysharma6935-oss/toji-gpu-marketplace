// One place paise-to-rupee formatting happens. Never format currency inline in a
// component — always through here, so "no $ anywhere" is enforced structurally.

export function formatPaise(paise: number): string {
  const rupees = paise / 100
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    maximumFractionDigits: paise % 100 === 0 ? 0 : 2,
  }).format(rupees)
}

export function formatPaisePerHour(paise: number): string {
  return `${formatPaise(paise)}/hr`
}
