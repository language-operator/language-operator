import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "./providers";
import { checkAndPerformInitialSetup } from "@/lib/initial-setup";

// Use system fonts to avoid network dependency during build
const geistSans = {
  variable: "--font-geist-sans",
  className: "",
};

const geistMono = {
  variable: "--font-geist-mono", 
  className: "",
};

export const metadata: Metadata = {
  title: "Language Operator Dashboard",
  description: "Manage your Language Operator resources",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  // Perform initial setup if needed (server-side only)
  // This will create the first admin user if the database is empty and CLI provided credentials
  await checkAndPerformInitialSetup();

  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
