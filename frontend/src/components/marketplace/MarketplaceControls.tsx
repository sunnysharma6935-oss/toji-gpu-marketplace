export type SortOption = 'price-asc' | 'price-desc' | 'vram-desc'

interface MarketplaceControlsProps {
  search: string
  onSearchChange: (v: string) => void
  sort: SortOption
  onSortChange: (v: SortOption) => void
  modelFilter: string
  onModelFilterChange: (v: string) => void
  availableModels: string[]
}

export function MarketplaceControls({
  search,
  onSearchChange,
  sort,
  onSortChange,
  modelFilter,
  onModelFilterChange,
  availableModels,
}: MarketplaceControlsProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <input
        type="text"
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        placeholder="Search GPUs"
        className="w-full rounded border border-border bg-surface px-4 py-2.5 text-sm text-paper placeholder:text-muted-dim focus:border-border-strong focus:outline-none sm:max-w-xs"
      />

      <div className="flex flex-wrap gap-3">
        <select
          value={modelFilter}
          onChange={(e) => onModelFilterChange(e.target.value)}
          className="rounded border border-border bg-surface px-3 py-2.5 text-sm text-paper focus:outline-none"
        >
          <option value="">All models</option>
          {availableModels.map((m) => (
            <option key={m} value={m}>{m}</option>
          ))}
        </select>

        <select
          value={sort}
          onChange={(e) => onSortChange(e.target.value as SortOption)}
          className="rounded border border-border bg-surface px-3 py-2.5 text-sm text-paper focus:outline-none"
        >
          <option value="price-asc">Price: Low to High</option>
          <option value="price-desc">Price: High to Low</option>
          <option value="vram-desc">VRAM: High to Low</option>
        </select>
      </div>
    </div>
  )
}
