import type { StudioFeatureVisibility } from '@/types'
import { firstVisibleStudioPath, isStudioFeatureVisible, studioFeatureForPath } from './studioFeatures'

export function resolveStudioVisibilityRedirect(
  path: string,
  visibility?: Partial<StudioFeatureVisibility> | null,
  isAdmin = false,
): string | null {
  if (isAdmin || !(path === '/studio' || path.startsWith('/studio/'))) {
    return null
  }

  const fallbackPath = firstVisibleStudioPath(visibility, false)
  const featureId = studioFeatureForPath(path)

  if (path === '/studio') {
    return fallbackPath
  }

  if (featureId && !isStudioFeatureVisible(featureId, visibility, false)) {
    return fallbackPath === path ? '/dashboard' : fallbackPath
  }

  return null
}
