import { createFileRoute } from '@tanstack/react-router'
import SettingsScreen from '@features/settings/screen/settings-screen'

export const Route = createFileRoute('/_authed/settings')({ component: SettingsScreen })
