import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import MenuScreen from '@features/menu/screen/menu-screen'

export const Route = createFileRoute('/_authed/menu')({
  validateSearch: z.object({ edit: z.string().optional() }),
  component: () => {
    const { edit } = Route.useSearch()
    return <MenuScreen edit={edit} />
  },
})
