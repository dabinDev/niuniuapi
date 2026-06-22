import type { CustomMenuItem } from '@/types'

export type CustomMenuVisibility = CustomMenuItem['visibility']

export function isCustomMenuUserVisible(visibility: unknown): boolean {
  return visibility === 'user'
}

export function customMenuVisibilityFromUserVisible(checked: boolean): CustomMenuVisibility {
  return checked ? 'user' : 'admin'
}

export function normalizeCustomMenuVisibility(visibility: unknown): CustomMenuVisibility {
  return isCustomMenuUserVisible(visibility) ? 'user' : 'admin'
}

export function normalizeCustomMenuItemsForAdminForm(items: unknown): CustomMenuItem[] {
  if (!Array.isArray(items)) return []
  return items.map((item, index) => {
    const raw = (item ?? {}) as Partial<CustomMenuItem>
    return {
      id: raw.id ?? '',
      label: raw.label ?? '',
      icon_svg: raw.icon_svg ?? '',
      url: raw.url ?? '',
      page_slug: raw.page_slug,
      visibility: normalizeCustomMenuVisibility(raw.visibility),
      sort_order: typeof raw.sort_order === 'number' ? raw.sort_order : index,
    }
  })
}
