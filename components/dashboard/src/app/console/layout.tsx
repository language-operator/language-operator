import { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'Console | Language Operator',
  description: 'Interactive agent console for Language Operator',
}

export default function ConsoleLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return children
}
