import { createFileRoute } from '@tanstack/react-router'
import BoardScreen from '@features/board/screen/board-screen'

export const Route = createFileRoute('/_authed/')({ component: BoardScreen })
