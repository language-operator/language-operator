/**
 * Console-specific layout that removes padding for full-screen chat interface
 */

interface ConsoleLayoutProps {
  children: React.ReactNode
}

export default function ConsoleLayout({
  children
}: ConsoleLayoutProps) {
  return (
    <div className="absolute inset-0">
      {children}
    </div>
  )
}